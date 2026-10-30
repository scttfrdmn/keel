// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// The float64 Level-3 suite (#192), DESIGN.md §5 rule 1's three checks; the third,
// Gonum's native float64, is gonumblas' TestLevel3AgreesWithGonum64.
//
// (1) Definition-order float64 references (Dgemm, Dsyrk, Dsymm, Dtrmm, Dtrsm,
// Dsyr2k) under PeerTolerance64. (2) Double-double under Tolerance64: GemmDD and
// GemmEntryDD on LOGICAL dense operands (the symmetric matrix filled out, the
// triangle made dense with a unit diagonal, Syr2k's two products as one length-2k
// product), so the expected values do not come from the index logic under test. The
// solves' arm (2) is the backward error op(A)·X̂ = alpha·B, formed in dd. The DD arm
// reaches k = 768, past block64's KC = 512, which is the size DESIGN.md asks of it.
//
// Every float64 microkernel this build registers is run by swapping activeKern64,
// so the public routine and its checks are in the tested path for each.

func forEachKern64(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	saved := activeKern64
	t.Cleanup(func() { activeKern64 = saved })
	for _, kn := range kern.Kernels64() {
		activeKern64 = kn
		t.Run(kn.ID(), f)
	}
}

func tf(b bool) Transpose {
	if b {
		return Trans
	}
	return NoTrans
}

// check2D64 holds an m×n result at ldc (got) against two m×n row-major references.
func check2D64(t *testing.T, what string, got []float64, ldc, m, n, red int, w1, s1, w2, s2 []float64) {
	t.Helper()
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			e := i*n + j
			check64(t, fmt.Sprintf("%s (%d,%d) ref", what, i, j), red, got[i*ldc+j], w1[e], s1[e], oracle.PeerTolerance64)
			check64(t, fmt.Sprintf("%s (%d,%d) dd", what, i, j), red, got[i*ldc+j], w2[e], s2[e], oracle.Tolerance64)
		}
	}
}

// dense64 copies an m×n matrix at ld into an m×n matrix at ld n.
func dense64(a []float64, ld, m, n int) []float64 {
	d := make([]float64, max(m*n, 1))
	for i := 0; i < m; i++ {
		copy(d[i*n:i*n+n], a[i*ld:i*ld+n])
	}
	return d
}

var l3Dims64 = []int{0, 1, 7, 17, 33}

func TestDgemm(t *testing.T) {
	forEachKern64(t, func(t *testing.T) {
		r := rand.New(rand.NewSource(101))
		for _, m := range l3Dims64 {
			for _, n := range l3Dims64 {
				for _, k := range []int{0, 1, 17, 33} {
					for _, f := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
						ra, ca, rb, cb := m, k, k, n
						if f[0] {
							ra, ca = k, m
						}
						if f[1] {
							rb, cb = n, k
						}
						a, lda := mat64(r, ra, ca, 2)
						b, ldb := mat64(r, rb, cb, 1)
						c, ldc := mat64(r, m, n, 3)
						for _, ab := range [][2]float64{{1, 0}, {-0.75, 0.5}} {
							got := append([]float64(nil), c...)
							Dgemm(tf(f[0]), tf(f[1]), m, n, k, ab[0], a, lda, b, ldb, ab[1], got, ldc)
							w1, s1 := oracle.Dgemm(f[0], f[1], m, n, k, ab[0], a, lda, b, ldb, ab[1], c, ldc)
							w2, s2 := oracle.GemmDD(f[0], f[1], m, n, k, ab[0], a, lda, b, ldb, ab[1], c, ldc)
							check2D64(t, fmt.Sprintf("Dgemm %v m=%d n=%d k=%d a,b=%v", f, m, n, k, ab), got, ldc, m, n, k+1, w1, s1, w2, s2)
						}
					}
				}
			}
		}
		// Past KC: k = 768 crosses block64's KC = 512, every entry on both arms.
		const m, n, k = 19, 21, 768
		a, lda := mat64(r, m, k, 1)
		b, ldb := mat64(r, k, n, 1)
		c, ldc := mat64(r, m, n, 1)
		got := append([]float64(nil), c...)
		Dgemm(NoTrans, NoTrans, m, n, k, 0.75, a, lda, b, ldb, -1.25, got, ldc)
		w1, s1 := oracle.Dgemm(false, false, m, n, k, 0.75, a, lda, b, ldb, -1.25, c, ldc)
		w2, s2 := oracle.GemmDD(false, false, m, n, k, 0.75, a, lda, b, ldb, -1.25, c, ldc)
		check2D64(t, "Dgemm k=768", got, ldc, m, n, k+1, w1, s1, w2, s2)
	})
}

// otherTriangleUntouched64 requires C's unreferenced triangle to keep its bits.
func otherTriangleUntouched64(t *testing.T, what string, got, c []float64, ldc, n int, lower bool) {
	t.Helper()
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (lower && j > i) || (!lower && j < i) {
				if math.Float64bits(got[i*ldc+j]) != math.Float64bits(c[i*ldc+j]) {
					t.Errorf("%s: wrote C's other triangle at (%d,%d)", what, i, j)
					return
				}
			}
		}
	}
}

func TestDsyrkDsyr2k(t *testing.T) {
	forEachKern64(t, func(t *testing.T) {
		r := rand.New(rand.NewSource(103))
		for _, lower := range []bool{false, true} {
			ul := Upper
			if lower {
				ul = Lower
			}
			for _, trans := range []bool{false, true} {
				for _, n := range l3Dims64 {
					for _, k := range []int{0, 1, 17, 40} {
						ra, ca := n, k
						if trans {
							ra, ca = k, n
						}
						a, lda := mat64(r, ra, ca, 2)
						b, ldb := mat64(r, ra, ca, 1)
						c, ldc := mat64(r, n, n, 2)
						what := fmt.Sprintf("%c%c n=%d k=%d", ul, tf(trans), n, k)
						// The dd arm's operands: op(A) as n×k (and op(B)), dense.
						opA, opB := make([]float64, max(n*k, 1)), make([]float64, max(n*k, 1))
						for i := 0; i < n; i++ {
							for p := 0; p < k; p++ {
								if trans {
									opA[i*k+p], opB[i*k+p] = a[p*lda+i], b[p*ldb+i]
								} else {
									opA[i*k+p], opB[i*k+p] = a[i*lda+p], b[i*ldb+p]
								}
							}
						}
						got := append([]float64(nil), c...)
						Dsyrk(ul, tf(trans), n, k, -0.75, a, lda, 0.5, got, ldc)
						w1, s1 := oracle.Dsyrk(lower, trans, n, k, -0.75, a, lda, 0.5, c, ldc)
						for i := 0; i < n; i++ {
							for j := 0; j < n; j++ {
								if !oracle.InTriangle(lower, i, j) {
									continue
								}
								e := fmt.Sprintf("Dsyrk %s (%d,%d)", what, i, j)
								check64(t, e+" ref", k+1, got[i*ldc+j], w1[i*n+j], s1[i*n+j], oracle.PeerTolerance64)
								wd, sd := oracle.GemmEntryDD(false, true, i, j, k, -0.75, opA, max(k, 1), opA, max(k, 1), 0.5, c, ldc)
								check64(t, e+" dd", k+1, got[i*ldc+j], wd, sd, oracle.Tolerance64)
							}
						}
						otherTriangleUntouched64(t, "Dsyrk "+what, got, c, ldc, n, lower)

						// Syr2k: alpha·(op(A)·op(B)ᵀ + op(B)·op(A)ᵀ) is one product of
						// length 2k between [op(A) op(B)] and [op(B) op(A)].
						cat1, cat2 := make([]float64, max(n*2*k, 1)), make([]float64, max(n*2*k, 1))
						for i := 0; i < n; i++ {
							copy(cat1[i*2*k:], opA[i*k:i*k+k])
							copy(cat1[i*2*k+k:], opB[i*k:i*k+k])
							copy(cat2[i*2*k:], opB[i*k:i*k+k])
							copy(cat2[i*2*k+k:], opA[i*k:i*k+k])
						}
						got = append([]float64(nil), c...)
						Dsyr2k(ul, tf(trans), n, k, -0.75, a, lda, b, ldb, 0.5, got, ldc)
						w1, s1 = oracle.Dsyr2k(lower, trans, n, k, -0.75, a, lda, b, ldb, 0.5, c, ldc)
						for i := 0; i < n; i++ {
							for j := 0; j < n; j++ {
								if !oracle.InTriangle(lower, i, j) {
									continue
								}
								e := fmt.Sprintf("Dsyr2k %s (%d,%d)", what, i, j)
								check64(t, e+" ref", 2*k+1, got[i*ldc+j], w1[i*n+j], s1[i*n+j], oracle.PeerTolerance64)
								wd, sd := oracle.GemmEntryDD(false, true, i, j, 2*k, -0.75, cat1, max(2*k, 1), cat2, max(2*k, 1), 0.5, c, ldc)
								check64(t, e+" dd", 2*k+1, got[i*ldc+j], wd, sd, oracle.Tolerance64)
							}
						}
						otherTriangleUntouched64(t, "Dsyr2k "+what, got, c, ldc, n, lower)
					}
				}
			}
		}
	})
}

func TestDsymmDtrmmDtrsm(t *testing.T) {
	forEachKern64(t, func(t *testing.T) {
		r := rand.New(rand.NewSource(107))
		for _, left := range []bool{true, false} {
			side := Left
			if !left {
				side = Right
			}
			for _, lower := range []bool{false, true} {
				ul := Upper
				if lower {
					ul = Lower
				}
				for _, m := range l3Dims64 {
					for _, n := range l3Dims64 {
						da := n
						if left {
							da = m
						}
						b, ldb := mat64(r, m, n, 1)
						what := fmt.Sprintf("%c%c m=%d n=%d", side, ul, m, n)

						// Symm through the full symmetric matrix.
						as, lds := triMat64(r, da, lower, false)
						full := logicalSym64(as, lds, da, lower)
						c, ldc := mat64(r, m, n, 2)
						got := append([]float64(nil), c...)
						Dsymm(side, ul, m, n, -0.75, as, lds, b, ldb, 0.5, got, ldc)
						w1, s1 := oracle.Dsymm(left, lower, m, n, -0.75, as, lds, b, ldb, 0.5, c, ldc)
						var w2, s2 []float64
						if left {
							w2, s2 = oracle.GemmDD(false, false, m, n, m, -0.75, full, max(da, 1), b, ldb, 0.5, c, ldc)
						} else {
							w2, s2 = oracle.GemmDD(false, false, m, n, n, -0.75, b, ldb, full, max(da, 1), 0.5, c, ldc)
						}
						check2D64(t, "Dsymm "+what, got, ldc, m, n, da+1, w1, s1, w2, s2)

						for _, trans := range []bool{false, true} {
							for _, unit := range []bool{false, true} {
								dg := NonUnit
								if unit {
									dg = Unit
								}
								wt := fmt.Sprintf("%s %c%c", what, tf(trans), dg)
								at, ldt := triMat64(r, da, lower, unit)
								tri := logicalTri64(at, ldt, da, lower, unit)
								// op(A) dense, for the dd arm.
								op := tri
								if trans {
									op = make([]float64, max(da*da, 1))
									for i := 0; i < da; i++ {
										for j := 0; j < da; j++ {
											op[i*da+j] = tri[j*da+i]
										}
									}
								}
								got := append([]float64(nil), b...)
								Dtrmm(side, ul, tf(trans), dg, m, n, 1.5, at, ldt, got, ldb)
								w1, s1 := oracle.Dtrmm(left, lower, trans, unit, m, n, 1.5, at, ldt, b, ldb)
								if left {
									w2, s2 = oracle.GemmDD(false, false, m, n, m, 1.5, op, max(da, 1), b, ldb, 0, nil, 1)
								} else {
									w2, s2 = oracle.GemmDD(false, false, m, n, n, 1.5, b, ldb, op, max(da, 1), 0, nil, 1)
								}
								check2D64(t, "Dtrmm "+wt, got, ldb, m, n, da+1, w1, s1, w2, s2)

								got = append([]float64(nil), b...)
								Dtrsm(side, ul, tf(trans), dg, m, n, 1.5, at, ldt, got, ldb)
								w1, s1 = oracle.Dtrsm(left, lower, trans, unit, m, n, 1.5, at, ldt, b, ldb)
								// Backward error: op(A)·X̂ (left) or X̂·op(A) (right) = 1.5·B, in dd.
								xh := dense64(got, ldb, m, n)
								var rr, rs []float64
								if left {
									rr, rs = oracle.GemmDD(false, false, m, n, m, 1, op, max(da, 1), xh, max(n, 1), 0, nil, 1)
								} else {
									rr, rs = oracle.GemmDD(false, false, m, n, n, 1, xh, max(n, 1), op, max(da, 1), 0, nil, 1)
								}
								for i := 0; i < m; i++ {
									for j := 0; j < n; j++ {
										e := fmt.Sprintf("Dtrsm %s (%d,%d)", wt, i, j)
										check64(t, e+" ref", da, got[i*ldb+j], w1[i*n+j], s1[i*n+j], oracle.PeerTolerance64)
										check64(t, e+" backward", da+1, rr[i*n+j], 1.5*b[i*ldb+j], rs[i*n+j]+math.Abs(1.5*b[i*ldb+j]), oracle.Tolerance64)
									}
								}
							}
						}
					}
				}
			}
		}
	})
}

// TestDParity3 is TestDParity for Level 3.
func TestDParity3(t *testing.T) {
	a32, a64 := make([]float32, 64), make([]float64, 64)
	ps := []struct {
		name string
		s, d func()
	}{
		{"gemm bad tA", func() { Sgemm('X', NoTrans, 2, 2, 2, 1, a32, 2, a32, 2, 0, a32, 2) }, func() { Dgemm('X', NoTrans, 2, 2, 2, 1, a64, 2, a64, 2, 0, a64, 2) }},
		{"gemm neg k", func() { Sgemm(NoTrans, NoTrans, 2, 2, -1, 1, a32, 2, a32, 2, 0, a32, 2) }, func() { Dgemm(NoTrans, NoTrans, 2, 2, -1, 1, a64, 2, a64, 2, 0, a64, 2) }},
		{"gemm ldb", func() { Sgemm(NoTrans, NoTrans, 2, 3, 2, 1, a32, 2, a32, 2, 0, a32, 3) }, func() { Dgemm(NoTrans, NoTrans, 2, 3, 2, 1, a64, 2, a64, 2, 0, a64, 3) }},
		{"gemm short c", func() { Sgemm(NoTrans, NoTrans, 9, 8, 1, 1, a32, 1, a32, 8, 0, a32, 8) }, func() { Dgemm(NoTrans, NoTrans, 9, 8, 1, 1, a64, 1, a64, 8, 0, a64, 8) }},
		{"syrk bad uplo", func() { Ssyrk('X', NoTrans, 2, 2, 1, a32, 2, 0, a32, 2) }, func() { Dsyrk('X', NoTrans, 2, 2, 1, a64, 2, 0, a64, 2) }},
		{"symm bad side", func() { Ssymm('X', Upper, 2, 2, 1, a32, 2, a32, 2, 0, a32, 2) }, func() { Dsymm('X', Upper, 2, 2, 1, a64, 2, a64, 2, 0, a64, 2) }},
		{"trsm bad diag", func() { Strsm(Left, Upper, NoTrans, 'X', 2, 2, 1, a32, 2, a32, 2) }, func() { Dtrsm(Left, Upper, NoTrans, 'X', 2, 2, 1, a64, 2, a64, 2) }},
		{"trsm lda", func() { Strsm(Left, Upper, NoTrans, NonUnit, 3, 2, 1, a32, 2, a32, 2) }, func() { Dtrsm(Left, Upper, NoTrans, NonUnit, 3, 2, 1, a64, 2, a64, 2) }},
		{"trmm neg n", func() { Strmm(Right, Lower, Trans, Unit, 2, -1, 1, a32, 2, a32, 2) }, func() { Dtrmm(Right, Lower, Trans, Unit, 2, -1, 1, a64, 2, a64, 2) }},
		{"syr2k short b", func() { Ssyr2k(Upper, NoTrans, 8, 8, 1, a32, 8, a32[:10], 8, 0, a32, 8) }, func() { Dsyr2k(Upper, NoTrans, 8, 8, 1, a64, 8, a64[:10], 8, 0, a64, 8) }},
	}
	for _, p := range ps {
		ms, md := panicMsg(p.s), panicMsg(p.d)
		if ms == "" {
			t.Errorf("%s: the float32 case does not panic, so it tests nothing", p.name)
			continue
		}
		rest := ms[len("keel: "):]
		name := rest[:strings.Index(rest, ":")]
		if want := strings.Replace(ms, "keel: "+name+":", "keel: D"+name[1:]+":", 1); md != want {
			t.Errorf("%s: float32 panics %q, float64 panics %q", p.name, ms, md)
		}
	}
}

// TestDtrsmDtrsvDivide is TestStrsvDividesRatherThanMultiplyingByAReciprocal at
// float64, for both solves and every backend and kernel. A reciprocal moves only the
// last bit, which every tolerance above admits, so the exact quotient is the witness
// (a planted 1/d multiply in Dtrsm's ScalDiv passed the whole tolerance suite). It is
// also the only check of l164's ScalDiv, since f64gen does not carry internal/l1's
// tests.
func TestDtrsmDtrsvDivide(t *testing.T) {
	const b, d float64 = 5, 49
	if b/d == b*(1/d) {
		t.Fatal("witness lost: 5/49 equals 5*(1/49) in float64, so this cannot tell a divide from a reciprocal")
	}
	forEachBackend64(t, func(t *testing.T) {
		forEachKern64(t, func(t *testing.T) {
			for _, ta := range []Transpose{NoTrans, Trans} {
				x := []float64{b}
				Dtrsv(Upper, ta, NonUnit, 1, []float64{d}, 1, x, 1)
				if x[0] != b/d {
					t.Errorf("Dtrsv %c: got %v, want the divide %v (the reciprocal gives %v)", ta, x[0], b/d, b*(1/d))
				}
				for _, s := range []Side{Left, Right} {
					x = []float64{b}
					Dtrsm(s, Upper, ta, NonUnit, 1, 1, 1, []float64{d}, 1, x, 1)
					if x[0] != b/d {
						t.Errorf("Dtrsm %c%c: got %v, want the divide %v (the reciprocal gives %v)", s, ta, x[0], b/d, b*(1/d))
					}
				}
			}
		})
	})
}
