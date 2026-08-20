// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"math"
	"math/rand"
	"testing"
)

// The three range predicates against the element-wise definition they are derived
// from (DESIGN.md §4/P4).
//
// whole, none and rowRange are the reason a masked GEMM costs nothing extra: they
// decide per tile and per row, without a test per element. That also makes them the
// kind of code where an off-by-one is invisible in the routine's output at most
// sizes and wrong at one — a tile that straddles the diagonal by a single element
// is rare enough that a size sweep can miss it and a mask that is one column wide
// at the wrong end still produces plausible numbers. So they are checked here
// directly, over every rectangle in a small square, against keeps().
//
// keeps() is the definition and has no other caller; this is what keeps it honest.

func TestTriMaskWhole(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for i0 := 0; i0 < d; i0++ {
			for j0 := 0; j0 < d; j0++ {
				for h := 1; h <= d-i0; h++ {
					for w := 1; w <= d-j0; w++ {
						all := true
						for i := i0; i < i0+h; i++ {
							for j := j0; j < j0+w; j++ {
								if !m.keeps(i, j) {
									all = false
								}
							}
						}
						if got := m.whole(i0, j0, h, w); got != all {
							t.Fatalf("lower=%v whole(%d,%d,%d,%d) = %v, want %v",
								lower, i0, j0, h, w, got, all)
						}
					}
				}
			}
		}
	}
}

func TestTriMaskNone(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for i0 := 0; i0 < d; i0++ {
			for j0 := 0; j0 < d; j0++ {
				for h := 1; h <= d-i0; h++ {
					for w := 1; w <= d-j0; w++ {
						any := false
						for i := i0; i < i0+h; i++ {
							for j := j0; j < j0+w; j++ {
								if m.keeps(i, j) {
									any = true
								}
							}
						}
						if got := m.none(i0, j0, h, w); got != !any {
							t.Fatalf("lower=%v none(%d,%d,%d,%d) = %v, want %v",
								lower, i0, j0, h, w, got, !any)
						}
					}
				}
			}
		}
	}
}

// TestTriMaskRowRange checks both halves of rowRange's contract: that the range it
// returns is exactly the kept elements of that row, and that it is a *range* at all
// — no kept element outside it, which is what makes a masked row a shorter row
// rather than a row with holes.
func TestTriMaskRowRange(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for gi := 0; gi < d; gi++ {
			for j0 := 0; j0 < d; j0++ {
				for w := 1; w <= d-j0; w++ {
					lo, hi := m.rowRange(gi, j0, w)
					if lo < 0 || hi > w || lo > hi {
						t.Fatalf("lower=%v rowRange(%d,%d,%d) = %d,%d: not a valid range in [0,%d]",
							lower, gi, j0, w, lo, hi, w)
					}
					for j := 0; j < w; j++ {
						in := j >= lo && j < hi
						if want := m.keeps(gi, j0+j); in != want {
							t.Fatalf("lower=%v rowRange(%d,%d,%d) = %d,%d: local column %d in=%v, keeps=%v",
								lower, gi, j0, w, lo, hi, j, in, want)
						}
					}
				}
			}
		}
	}
}

// TestTriMaskOff pins the pass-through case. Every call plain Gemm makes goes
// through these predicates with on false, so "no mask" has to be exactly no mask:
// every rectangle whole, none empty-only, every row full width.
func TestTriMaskOff(t *testing.T) {
	var m triMask
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if !m.keeps(i, j) {
				t.Fatalf("mask off: keeps(%d,%d) = false", i, j)
			}
		}
	}
	for i0 := 0; i0 < 5; i0++ {
		for j0 := 0; j0 < 5; j0++ {
			for h := 1; h <= 3; h++ {
				for w := 1; w <= 3; w++ {
					if !m.whole(i0, j0, h, w) {
						t.Fatalf("mask off: whole(%d,%d,%d,%d) = false", i0, j0, h, w)
					}
					if m.none(i0, j0, h, w) {
						t.Fatalf("mask off: none(%d,%d,%d,%d) = true", i0, j0, h, w)
					}
					if lo, hi := m.rowRange(i0, j0, w); lo != 0 || hi != w {
						t.Fatalf("mask off: rowRange(%d,%d,%d) = %d,%d, want 0,%d",
							i0, j0, w, lo, hi, w)
					}
				}
			}
		}
	}
	// An empty rectangle is whole and none at once, on either setting. Both callers
	// rely on it: the loop nest asks about blocks it has already clamped to the
	// matrix, and clamping to zero must not turn into a kernel call.
	for _, mm := range []triMask{{}, {on: true}, {on: true, lower: true}} {
		if !mm.whole(0, 0, 0, 4) || !mm.none(0, 0, 0, 4) {
			t.Fatalf("%+v: empty rectangle should be both whole and none", mm)
		}
		if !mm.whole(0, 0, 4, 0) || !mm.none(0, 0, 4, 0) {
			t.Fatalf("%+v: empty rectangle should be both whole and none", mm)
		}
	}
}

// solveRightStrided is solveRight exactly as it stood before issue #37's loop
// interchange: the innermost loop over m with stride ldb, the trans test per
// element, and the in-place update of B rather than a float32 accumulator.
//
// It is kept verbatim, and only here, because it is the reference the interchange
// is claimed *bit-identical* to. A paraphrase would weaken the claim to "two
// implementations of my current understanding agree", which is the failure the
// differential idiom exists to avoid: the point of this reference is that it is the
// code that produced the published 0.21 GFLOP/s, not a fresh reading of it.
func solveRightStrided(lowerEff, trans, unit bool, m, d int, a []float32, lda int, b []float32, ldb int) {
	for step := 0; step < d; step++ {
		j := step
		if lowerEff {
			j = d - 1 - step
		}
		plo, phi := 0, j
		if lowerEff {
			plo, phi = j+1, d
		}
		for p := plo; p < phi; p++ {
			apj := a[p*lda+j]
			if trans {
				apj = a[j*lda+p]
			}
			for i := 0; i < m; i++ {
				b[i*ldb+j] -= b[i*ldb+p] * apj
			}
		}
		if !unit {
			dv := a[j*lda+j]
			for i := 0; i < m; i++ {
				b[i*ldb+j] /= dv
			}
		}
	}
}

// TestSolveRightInterchangeIsBitIdentical is the whole warrant for #37's first arm
// being an optimisation rather than a numerical change.
//
// Rows of X are independent, so moving the row loop from innermost to outermost
// cannot reassociate anything: for a fixed row, the j order, the p order and the
// division are unchanged. That predicts *bit* equality, not equality within a
// tolerance, so this test compares raw bits and no oracle.Tolerance appears in it.
// Comparing with == would silently pass on two NaNs of different payloads and fail
// on nothing this test can produce; math.Float32bits says what is meant.
//
// The shapes exercise both triangles, both transpose flags and both diagonal
// conventions — the four (lowerEff, trans) pairs are the ones that pick between the
// contiguous-run and stride-lda operands, and the interchange touched that split.
// ldb > d and lda > d on purpose: with ldb == d the strided nest's inner loop is
// accidentally contiguous, so a padded window is the only shape that distinguishes
// the two implementations at all.
func TestSolveRightInterchangeIsBitIdentical(t *testing.T) {
	shapes := []struct{ m, d int }{
		{1, 1},    // one element: the degenerate case both loops have to survive
		{1, 7},    // one row, so the interchange has no outer loop to speak of
		{9, 1},    // one column: every p range is empty and only the divide runs
		{5, 4},    // fewer rows than columns
		{64, 64},  // the shipped MB
		{130, 33}, // neither dimension a multiple of anything, m > any tile
	}
	for _, sh := range shapes {
		for _, lowerEff := range []bool{false, true} {
			for _, trans := range []bool{false, true} {
				for _, unit := range []bool{false, true} {
					lda, ldb := sh.d+5, sh.d+11
					rng := rand.New(rand.NewSource(int64(sh.m*1000 + sh.d)))
					a := make([]float32, sh.d*lda+lda)
					for i := range a {
						a[i] = rng.Float32()*2 - 1
					}
					// A diagonal bounded away from zero: this test is about which
					// loop order ran, and a near-singular block would let ordinary
					// ill-conditioning masquerade as a difference between them.
					for j := 0; j < sh.d; j++ {
						a[j*lda+j] = 1 + rng.Float32()
					}
					b0 := make([]float32, sh.m*ldb+ldb)
					for i := range b0 {
						b0[i] = rng.Float32()*2 - 1
					}
					got := append([]float32(nil), b0...)
					want := append([]float32(nil), b0...)
					solveRight(lowerEff, trans, unit, sh.m, sh.d, a, lda, got, ldb)
					solveRightStrided(lowerEff, trans, unit, sh.m, sh.d, a, lda, want, ldb)
					for i := range want {
						if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
							t.Fatalf("m=%d d=%d lowerEff=%v trans=%v unit=%v: b[%d] = %v (bits %#x), strided nest gives %v (bits %#x)",
								sh.m, sh.d, lowerEff, trans, unit, i,
								got[i], math.Float32bits(got[i]), want[i], math.Float32bits(want[i]))
						}
					}
				}
			}
		}
	}
}

// BenchmarkSolveRightInterchange is #37's first arm measured against the code it
// replaced, both in one binary and one process.
//
// Two arms in one run, not two runs of one arm: the quantity of interest is a
// ratio, and a ratio taken across two builds picks up between-binary layout noise
// that has been measured on these hosts at 1.0–1.7% (#22's layout ensemble). Here
// the only difference between the arms is which function the loop calls.
//
// The shape is the one Trsm actually generates at n = 2048 and MB = 64 under one
// worker: strips(m) hands the whole 2048 rows to one strip, so a diagonal block's
// solve is m = 2048, d = MB, with lda = ldb = n. That matters — a benchmark at
// ldb == d would make the strided arm accidentally contiguous and measure nothing.
//
// B is restored between iterations with the timer stopped, for the reason
// bench/scale_test.go's Strsm row gives: the solve is in place, so iteration two
// would otherwise solve against iteration one's output, and repeated application of
// A⁻¹ walks the magnitudes toward the dominant eigendirection until what is timed
// is denormal arithmetic rather than the routine.
//
// solve-GFLOP/s is reported on both arms so the numbers are comparable to the MB
// sweep's solve-only column: one multiply-add per (row, column) pair of the
// triangle including the diagonal, i.e. 2·m·d(d+1)/2 = m·d·(d+1).
func BenchmarkSolveRightInterchange(b *testing.B) {
	const n, d = 2048, 64
	m, lda, ldb := n, n, n
	a := benchTri(n)
	b0 := benchMat(m, n)
	bm := make([]float32, len(b0))
	flops := float64(m) * float64(d) * float64(d+1)
	arms := []struct {
		name string
		fn   func(lowerEff, trans, unit bool, m, d int, a []float32, lda int, b []float32, ldb int)
	}{
		{"interchanged", solveRight},
		{"strided", solveRightStrided},
	}
	for _, arm := range arms {
		b.Run(arm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				copy(bm, b0)
				b.StartTimer()
				arm.fn(true, false, false, m, d, a, lda, bm, ldb)
			}
			b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "solve-GFLOP/s")
		})
	}
}
