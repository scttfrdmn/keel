// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package pack

import (
	"fmt"
	"math"
	"testing"
)

// The packed layout, the padding, and the equivalence of the two branches (issue
// #45).
//
// This package had no tests of its own for three phases. That was not the same as
// untested: a wrong packed layout makes Sgemm produce wrong numbers, and the root
// package's differential tests cover all four transpose flags against
// internal/oracle. What those tests cannot see is the difference between "the
// layout is self-consistent with the microkernels" and "the layout is the one the
// package doc specifies", and they see a stale buffer slot only when some shape
// happens to reuse it in a way that reaches C.
//
// Both distinctions matter now rather than in the abstract. #21's measurement says
// the branch-selection rule is wrong on the A side (the copy branch is 2.8x slower
// than the transposing branch at MR=2, because its runs are two floats long), so
// stage 1 intends to change which branch runs for a given blk. These tests are what
// make that change checkable at this boundary instead of at Sgemm's output.
//
// The layout formula is written out here as index arithmetic on the source rather
// than by calling any helper from this package, so a change to the layout has to be
// made in two places by someone who means it.

// srcAt returns op(M)[blocked, depth] for a source in either orientation:
// depthContig says the depth index is the contiguous one, which is exactly the
// flag panels() branches on.
func srcAt(src []float32, ld int, depthContig bool, blocked, depth int) float32 {
	if depthContig {
		return src[blocked*ld+depth]
	}
	return src[depth*ld+blocked]
}

// checkLayout asserts dst[ib*blk*kc + p*blk + x] == alpha * op(src)[b0+ib*blk+x, p0+p]
// for every valid element, and exactly +0 for every padding slot. It is the package
// doc's formula and nothing else.
//
// Padding is checked bitwise rather than with ==, because a -0 slot compares equal
// to +0 and is not what the doc promises; and because the buffers these tests pass
// in are poisoned first, a slot that merely happens to compare equal to zero would
// pass a == check while holding a stale sign bit.
func checkLayout(t *testing.T, what string, dst []float32, blk int, alpha float32,
	src []float32, ld int, depthContig bool, b0, count, p0, kc int) {
	t.Helper()
	if kc == 0 || count == 0 {
		return
	}
	nb := (count + blk - 1) / blk
	for ib := 0; ib < nb; ib++ {
		for p := 0; p < kc; p++ {
			for x := 0; x < blk; x++ {
				got := dst[ib*blk*kc+p*blk+x]
				if b := ib*blk + x; b < count {
					want := alpha * srcAt(src, ld, depthContig, b0+b, p0+p)
					if math.Float32bits(got) != math.Float32bits(want) {
						t.Fatalf("%s: dst[ib=%d p=%d x=%d] = %v, want %v (src element [%d,%d])",
							what, ib, p, x, got, want, b0+b, p0+p)
					}
					continue
				}
				if math.Float32bits(got) != 0 {
					t.Fatalf("%s: padding slot [ib=%d p=%d x=%d] = %v (bits %#08x), "+
						"want exactly +0 — the doc says the zeros are written, not assumed",
						what, ib, p, x, got, math.Float32bits(got))
				}
			}
		}
	}
}

// poison fills a buffer with a value no test source can produce, so that an
// unwritten slot is a failure rather than a coincidence. Negative and non-integral
// so it cannot be confused with any seq() element, and its bit pattern is not zero.
func poison(n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = -7.5e-3
	}
	return v
}

// seq is a source matrix whose every element is distinct, so a layout error moves a
// value the assertion can name rather than substituting an equal one. Exactly
// representable in float32, and small enough that alpha scaling stays exact.
func seq(rows, cols, ld int) []float32 {
	v := poison(rows*ld + cols) // pad past the last row so a stride bug reads poison
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v[i*ld+j] = float32(i*cols+j+1) * 0.25
		}
	}
	return v
}

// packShapes covers the cases the layout can be wrong in: exact multiples of the
// tile in both axes, ragged in each axis separately and in both at once, a count
// smaller than one block, a single element, and nonzero offsets so that i0/p0/j0
// are not silently ignored (every shape here is packed at an offset as well).
var packShapes = []struct {
	blk, count, kc int
}{
	{4, 8, 4},   // exact in both axes, two whole blocks
	{4, 8, 5},   // ragged in depth only — depth is not blocked, so nothing pads
	{4, 9, 4},   // ragged in the blocked axis: last panel is 1 valid + 3 padding
	{4, 10, 6},  // ragged in the blocked axis, several depths
	{4, 3, 3},   // count < blk: one panel, mostly padding
	{4, 1, 1},   // single element
	{2, 5, 7},   // blk=2, the shipped thin tile's MR
	{32, 33, 3}, // blk=32, the shipped NR, one element into the second panel
	{32, 32, 1}, // exactly one panel at the shipped NR
	{1, 5, 3},   // blk=1 degenerates to a plain transpose; no padding is possible
}

// TestAPanelsLayout checks APanels in both orientations against the doc's formula,
// with alpha folded in, at an offset, into a poisoned buffer.
//
// The trans flag decides only which axis of the source is contiguous — the packed
// result is specified to be identical — so the expectation is the same formula with
// depthContig flipped, and the two orientations hold the same logical matrix.
func TestAPanelsLayout(t *testing.T) {
	for _, s := range packShapes {
		for _, trans := range []bool{false, true} {
			for _, alpha := range []float32{1, -0.5} {
				name := fmt.Sprintf("mr=%d/rows=%d/kc=%d/trans=%v/alpha=%v",
					s.blk, s.count, s.kc, trans, alpha)
				t.Run(name, func(t *testing.T) {
					const i0, p0 = 3, 2
					// Untransposed A is (i0+rows) x (p0+kc); transposed it is the
					// other way round. lda is padded past the used columns so a
					// stride error lands in poison rather than in the next row.
					rows, cols := i0+s.count, p0+s.kc
					if trans {
						rows, cols = cols, rows
					}
					lda := cols + 5
					a := seq(rows, cols, lda)
					dst := poison(ALen(s.blk, s.count, s.kc))
					APanels(dst, s.blk, alpha, a, lda, trans, i0, s.count, p0, s.kc)
					// APanels passes !trans as depthContig: untransposed A is m x k,
					// so the depth index is the contiguous one.
					checkLayout(t, "APanels/"+name, dst, s.blk, alpha,
						a, lda, !trans, i0, s.count, p0, s.kc)
				})
			}
		}
	}
}

// TestBPanelsLayout is the mirror image, and there is no alpha: BPanels hard-codes 1
// because alpha is folded into A. Passing an alpha here would be testing a parameter
// the signature does not have, so the expectation is pinned at 1 — if BPanels ever
// grows an alpha, this test stops compiling rather than silently accepting it.
func TestBPanelsLayout(t *testing.T) {
	for _, s := range packShapes {
		for _, trans := range []bool{false, true} {
			name := fmt.Sprintf("nr=%d/cols=%d/kc=%d/trans=%v", s.blk, s.count, s.kc, trans)
			t.Run(name, func(t *testing.T) {
				const j0, p0 = 2, 3
				// Untransposed B is (p0+kc) x (j0+cols).
				rows, cols := p0+s.kc, j0+s.count
				if trans {
					rows, cols = cols, rows
				}
				ldb := cols + 5
				b := seq(rows, cols, ldb)
				dst := poison(BLen(s.blk, s.count, s.kc))
				BPanels(dst, s.blk, b, ldb, trans, p0, s.kc, j0, s.count)
				// BPanels passes trans as depthContig: untransposed B is k x n, so
				// the blocked axis is the contiguous one.
				checkLayout(t, "BPanels/"+name, dst, s.blk, 1,
					b, ldb, trans, j0, s.count, p0, s.kc)
			})
		}
	}
}

// TestPanelsLeaveNothingOfThePreviousPack is the invariant the package doc states
// and no other test can see: "the zeros are written, not assumed", because these
// buffers are reused across every (jc, pc, ic) block of a call, so a stale value
// from a previous block would be indistinguishable from data.
//
// A poisoned buffer catches an unwritten slot. It does not catch the case that
// actually happens in gemm, which is a slot written by a *previous pack* with a
// plausible value — the last block of a call packs a ragged remainder into the
// buffer the full blocks just filled. So: pack a large block, then pack a smaller
// one into the same buffer, and require every slot the second pack claims to hold
// what the second pack should have put there. The first pack's values are the
// poison for the second.
//
// Both the blocked axis and the depth shrink, and separately, because they fail
// differently: a smaller count leaves stale values in the padding of the last
// panel, while a smaller kc changes the panel stride, so slot i of the second
// layout is a *different element* of the first, not the same one.
func TestPanelsLeaveNothingOfThePreviousPack(t *testing.T) {
	const blk, lda = 4, 40
	first := struct{ count, kc int }{count: 16, kc: 8}
	seconds := []struct {
		what      string
		count, kc int
	}{
		{"narrower: ragged remainder into a buffer full of whole blocks", 9, 8},
		{"shallower: the panel stride changes, so every slot moves", 16, 3},
		{"both", 5, 2},
		{"one element", 1, 1},
	}
	for _, second := range seconds {
		for _, trans := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/trans=%v", second.what, trans), func(t *testing.T) {
				a := seq(lda, lda, lda)
				// One buffer, sized for the first (larger) pack, as gemm's is.
				dst := poison(ALen(blk, first.count, first.kc))
				APanels(dst, blk, 1, a, lda, trans, 0, first.count, 0, first.kc)
				// A different source region for the second pack, so a stale value
				// cannot coincide with the value that belongs there.
				const i0, p0 = 5, 7
				APanels(dst, blk, 1, a, lda, trans, i0, second.count, p0, second.kc)
				checkLayout(t, "second pack", dst, blk, 1,
					a, lda, !trans, i0, second.count, p0, second.kc)
			})
		}
	}
}

// TestBranchesAgree is the guard that makes #21's change safe to attempt.
//
// The two branches of panels() — the transposing one and the copy one — are
// specified to produce identical output; which one runs is decided only by which
// axis of the source is contiguous. So packing the same logical matrix from a
// row-major source and from its transpose must give bit-identical panels, and a
// change to the branch-selection rule (which is what #21's measurement argues for,
// since the copy branch is the slower one at MR in {2,4}) cannot change any number
// keel produces.
//
// Bit-identical, not close: these are copies and multiplications by the same alpha,
// in a different order. The data includes both zeros, both infinities and a quiet
// NaN, because the interesting failures are in the values where a reordering is not
// harmless.
//
// A *signalling* NaN is deliberately absent, and it is the one input where the
// branches provably differ: the copy branch takes copy() when alpha == 1 and moves
// the payload untouched, while the transposing branch always computes alpha*v, and
// 1*sNaN quiets it. Nothing in keel produces a signalling NaN — no IEEE operation
// does — and BLAS specifies nothing about NaN payloads, so this is recorded here as
// a known asymmetry rather than tested as a requirement. If a future change makes
// the branches agree on it, this comment is what says the agreement was not free.
func TestBranchesAgree(t *testing.T) {
	const dim = 24
	special := []float32{
		0, float32(math.Copysign(0, -1)),
		float32(math.Inf(1)), float32(math.Inf(-1)),
		float32(math.NaN()),
		math.SmallestNonzeroFloat32, math.MaxFloat32,
	}
	// A dim x dim matrix and its exact transpose. Distinct values everywhere, with
	// the specials sprinkled along a diagonal so they land in different positions
	// within a panel for every blk.
	m := make([]float32, dim*dim)
	for i := 0; i < dim; i++ {
		for j := 0; j < dim; j++ {
			m[i*dim+j] = float32(i*dim+j+1) * 0.125
		}
	}
	for i, v := range special {
		m[i*dim+(i*7+3)%dim] = v
	}
	mT := make([]float32, dim*dim)
	for i := 0; i < dim; i++ {
		for j := 0; j < dim; j++ {
			mT[j*dim+i] = m[i*dim+j]
		}
	}

	for _, blk := range []int{1, 2, 3, 4, 8, 16, 32} {
		for _, alpha := range []float32{1, -0.5} {
			for _, count := range []int{dim, dim - 1, 5} {
				name := fmt.Sprintf("blk=%d/alpha=%v/count=%d", blk, alpha, count)
				t.Run(name, func(t *testing.T) {
					const kc = dim
					// The same logical op(A): from m (untransposed, so the pack
					// transposes) and from mT with trans set (so the pack copies).
					viaTranspose := poison(ALen(blk, count, kc))
					viaCopy := poison(ALen(blk, count, kc))
					APanels(viaTranspose, blk, alpha, m, dim, false, 0, count, 0, kc)
					APanels(viaCopy, blk, alpha, mT, dim, true, 0, count, 0, kc)
					for i := range viaTranspose {
						tb, cb := math.Float32bits(viaTranspose[i]), math.Float32bits(viaCopy[i])
						if tb != cb {
							t.Fatalf("%s: slot %d differs between branches: "+
								"transposing %v (%#08x), copy %v (%#08x)",
								name, i, viaTranspose[i], tb, viaCopy[i], cb)
						}
					}
				})
			}
		}
	}
}

// TestPanelsLenIsExactlyEnough pins the contract between the length functions and
// the routines: dst must hold ALen/BLen floats, and that many is enough at every
// ragged shape. An exact-length slice makes an overrun a panic rather than a silent
// write into whatever gemm allocated next.
//
// The other half — one float short must panic, with the geometry in the message —
// pins the guard itself. Without it, a caller who under-allocated would get a
// truncated pack and plausible-looking wrong numbers, which is the failure mode the
// guard exists to convert into a crash.
//
// The message is asserted, not just the panic, and that is the whole point of this
// half: deleting the explicit guard entirely still panics, because the panel
// re-slice runs off the end of dst — so a test that accepted any panic would pass
// over code with no guard at all (verified by mutation). What the guard buys is a
// message that names both lengths instead of "slice bounds out of range", so that is
// what gets checked.
func TestPanelsLenIsExactlyEnough(t *testing.T) {
	for _, s := range packShapes {
		name := fmt.Sprintf("blk=%d/count=%d/kc=%d", s.blk, s.count, s.kc)
		t.Run(name, func(t *testing.T) {
			a := seq(s.count+2, s.kc+2, s.kc+2)
			need := ALen(s.blk, s.count, s.kc)
			// make's capacity is its length, so these have no slack: a re-slice
			// past the end faults rather than reaching into spare capacity, which
			// is the property this test needs and the reason it does not take a
			// subslice of something larger.
			exact := make([]float32, need)
			APanels(exact, s.blk, 1, a, s.kc+2, false, 0, s.count, 0, s.kc)

			if need == 0 {
				return // nothing to be one float short of
			}
			short := make([]float32, need-1)
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Fatalf("%s: APanels accepted %d floats when it needs %d",
							name, need-1, need)
					}
					msg := fmt.Sprint(r)
					want := fmt.Sprintf("pack: dst has %d floats, need %d", need-1, need)
					if msg != want {
						t.Fatalf("%s: panicked with %q, want %q — an under-allocated dst "+
							"must name both lengths, not fall through to a slice-bounds panic",
							name, msg, want)
					}
				}()
				APanels(short, s.blk, 1, a, s.kc+2, false, 0, s.count, 0, s.kc)
			}()
		})
	}
}

// TestPanelsLenRejectsBadGeometry: panelsLen is the only arithmetic both ALen and
// BLen are, and a zero or negative blk would make its division by blk*kc a divide
// by zero deep inside panels() instead of a named panic at the boundary.
func TestPanelsLenRejectsBadGeometry(t *testing.T) {
	bad := []struct{ blk, count, kc int }{
		{0, 4, 4}, {-1, 4, 4}, {4, -1, 4}, {4, 4, -1},
	}
	for _, b := range bad {
		t.Run(fmt.Sprintf("blk=%d/count=%d/kc=%d", b.blk, b.count, b.kc), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("ALen(%d,%d,%d) returned instead of panicking",
						b.blk, b.count, b.kc)
				}
			}()
			_ = ALen(b.blk, b.count, b.kc)
		})
	}
}

// TestEmptyPacksWriteNothing: kc == 0 and count == 0 are reachable from gemm's edge
// shapes, and panels() returns early on them.
//
// Only the kc == 0 half of that early return is load-bearing: it is what keeps nb's
// divisor nonzero. The count == 0 half is redundant — need is 0, so nb is 0 and the
// panel loop does not run — and deleting it survives this test, correctly, because
// the two versions are equivalent code rather than one being a bug. Recorded so that
// a reader does not mistake this test for evidence that the clause is needed.
func TestEmptyPacksWriteNothing(t *testing.T) {
	a := seq(8, 8, 8)
	for _, s := range []struct{ count, kc int }{{0, 4}, {4, 0}, {0, 0}} {
		t.Run(fmt.Sprintf("count=%d/kc=%d", s.count, s.kc), func(t *testing.T) {
			dst := poison(16)
			before := append([]float32(nil), dst...)
			APanels(dst, 4, 2, a, 8, false, 0, s.count, 0, s.kc)
			BPanels(dst, 4, a, 8, false, 0, s.kc, 0, s.count)
			for i := range dst {
				if math.Float32bits(dst[i]) != math.Float32bits(before[i]) {
					t.Fatalf("count=%d kc=%d: slot %d was written (%v -> %v)",
						s.count, s.kc, i, before[i], dst[i])
				}
			}
		})
	}
}
