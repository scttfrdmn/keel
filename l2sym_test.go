// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// The references are the Level-3 ones with a single column: Strmv is Strmm with
// B = x as an n×1 matrix, Strsv is Strsm likewise, Ssymv is Ssymm with B = x and
// C = y, and Ssyr/Ssyr2 are Ssyrk/Ssyr2k with k = 1 and beta = 1. Those references
// are float64 definitions written independently of keel, so reusing them adds no
// shared code path with the routines under test.

var (
	l2Dims = []int{0, 1, 5, 33, 70}
	l2Incs = []int{1, 2, -1, -3}
)

func randVals(r *rand.Rand, n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32(r.NormFloat64())
	}
	return v
}

// elems reads a strided BLAS vector back into element order.
func elems(v []float32, n, inc int) []float32 {
	out := make([]float32, n)
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[j] = v[base+j*inc]
	}
	return out
}

func triMat(r *rand.Rand, n int, lower, unit bool) ([]float32, int) {
	a, lda := paddedMat(r, n, n, 2)
	for i := 0; i < n; i++ {
		a[i*lda+i] = 4 + float32(r.Float64()) // keeps the solves well conditioned
	}
	poisonTriangle(a, n, lda, lower, unit)
	return a, lda
}

func TestStrmvStrsvLattice(t *testing.T) {
	r := rand.New(rand.NewSource(43))
	for _, ul := range []Uplo{Upper, Lower} {
		for _, ta := range []Transpose{NoTrans, Trans} {
			for _, d := range []Diag{NonUnit, Unit} {
				for _, n := range l2Dims {
					a, lda := triMat(r, n, ul == Lower, d == Unit)
					for _, inc := range l2Incs {
						xv := randVals(r, n)
						x := spread(xv, inc)
						what := fmt.Sprintf("%c%c%c n=%d inc=%d", ul, ta, d, n, inc)

						got := append([]float32(nil), x...)
						Strmv(ul, ta, d, n, a, lda, got, inc)
						want, scale := oracle.Trmm(true, ul == Lower, ta == Trans, d == Unit, n, 1, 1, a, lda, xv, 1)
						ge := elems(got, n, inc)
						for i := 0; i < n; i++ {
							checkScalar(t, fmt.Sprintf("Strmv %s [%d]", what, i), n, ge[i], want[i], scale[i])
						}
						checkGaps(t, "Strmv "+what, got, x, n, inc)

						got = append([]float32(nil), x...)
						Strsv(ul, ta, d, n, a, lda, got, inc)
						want, scale = oracle.Trsm(true, ul == Lower, ta == Trans, d == Unit, n, 1, 1, a, lda, xv, 1)
						ge = elems(got, n, inc)
						for i := 0; i < n; i++ {
							checkScalar(t, fmt.Sprintf("Strsv %s [%d]", what, i), n, ge[i], want[i], scale[i])
						}
						checkGaps(t, "Strsv "+what, got, x, n, inc)
					}
				}
			}
		}
	}
}

func TestSsymvLattice(t *testing.T) {
	r := rand.New(rand.NewSource(47))
	for _, ul := range []Uplo{Upper, Lower} {
		for _, n := range l2Dims {
			a, lda := paddedMat(r, n, n, 2)
			poisonTriangle(a, n, lda, ul == Lower, false)
			for _, inc := range l2Incs {
				xv := randVals(r, n)
				x := spread(xv, inc)
				for _, alpha := range []float32{0, 1, -0.75} {
					for _, beta := range []float32{0, 1, 0.5} {
						yv := randVals(r, n)
						if beta == 0 {
							for i := range yv {
								yv[i] = nanPoison // beta 0 must not read y
							}
						}
						y := spread(yv, inc)
						got := append([]float32(nil), y...)
						Ssymv(ul, n, alpha, a, lda, x, inc, beta, got, inc)
						ref := yv
						if beta == 0 {
							ref = make([]float32, n)
						}
						want, scale := oracle.Symm(true, ul == Lower, n, 1, alpha, a, lda, xv, 1, beta, ref, 1)
						what := fmt.Sprintf("Ssymv %c n=%d inc=%d alpha=%v beta=%v", ul, n, inc, alpha, beta)
						ge := elems(got, n, inc)
						for i := 0; i < n; i++ {
							checkScalar(t, fmt.Sprintf("%s [%d]", what, i), n+1, ge[i], want[i], scale[i])
						}
						checkGaps(t, what, got, y, n, inc)
					}
				}
			}
		}
	}
}

func TestSsyrSsyr2Lattice(t *testing.T) {
	r := rand.New(rand.NewSource(53))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, n := range l2Dims {
			for _, inc := range l2Incs {
				xv, yv := randVals(r, n), randVals(r, n)
				x, y := spread(xv, inc), spread(yv, inc)
				for _, alpha := range []float32{0, 1, -0.75} {
					a0, lda := paddedMat(r, n, n, 2)
					poisonTriangle(a0, n, lda, lower, false) // the other triangle must survive
					for _, two := range []bool{false, true} {
						got := append([]float32(nil), a0...)
						var want, scale []float64
						name := "Ssyr"
						if two {
							name = "Ssyr2"
							Ssyr2(ul, n, alpha, x, inc, y, inc, got, lda)
							want, scale = oracle.Syr2k(lower, false, n, 1, alpha, xv, 1, yv, 1, 1, a0, lda)
						} else {
							Ssyr(ul, n, alpha, x, inc, got, lda)
							want, scale = oracle.Syrk(lower, false, n, 1, alpha, xv, 1, 1, a0, lda)
						}
						what := fmt.Sprintf("%s %c n=%d inc=%d alpha=%v", name, ul, n, inc, alpha)
						for i := 0; i < n; i++ {
							for j := 0; j < n; j++ {
								if !oracle.InTriangle(lower, i, j) {
									if !math.IsNaN(float64(got[i*lda+j])) {
										t.Fatalf("%s: wrote the other triangle at [%d,%d]", what, i, j)
									}
									continue
								}
								checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), 3, got[i*lda+j], want[i*n+j], scale[i*n+j])
							}
						}
					}
				}
			}
		}
	}
}

func TestLevel2SymRejectsBadArguments(t *testing.T) {
	a, x := make([]float32, 16), make([]float32, 8)
	mustPanic(t, "Strmv bad uplo", func() { Strmv('X', NoTrans, NonUnit, 2, a, 2, x, 1) })
	mustPanic(t, "Strmv zero inc", func() { Strmv(Upper, NoTrans, NonUnit, 2, a, 2, x, 0) })
	mustPanic(t, "Strsv bad diag", func() { Strsv(Upper, NoTrans, 'X', 2, a, 2, x, 1) })
	mustPanic(t, "Strsv lda < n", func() { Strsv(Upper, NoTrans, NonUnit, 3, a, 2, x, 1) })
	mustPanic(t, "Ssymv short y", func() { Ssymv(Upper, 4, 1, a, 4, x, 1, 0, x[:3], 1) })
	mustPanic(t, "Ssymv negative n", func() { Ssymv(Upper, -1, 1, a, 1, x, 1, 0, x, 1) })
	mustPanic(t, "Ssyr short x", func() { Ssyr(Upper, 4, 1, x[:3], 1, a, 4) })
	mustPanic(t, "Ssyr2 zero incY", func() { Ssyr2(Lower, 2, 1, x, 1, x, 0, a, 2) })
}

// TestStrsvDividesRatherThanMultiplyingByAReciprocal pins what the tolerance-based
// lattice cannot see: a reciprocal changes only the last bit, which the tolerance
// admits. 5/49 and 5·(1/49) differ in float32, so the exact answer is the witness.
// The 1×1 case reaches the diagonal divide with no dot product in front of it.
func TestStrsvDividesRatherThanMultiplyingByAReciprocal(t *testing.T) {
	const b, d float32 = 5, 49
	if b/d == b*(1/d) {
		t.Fatal("witness lost: 5/49 equals 5*(1/49) in float32, so this cannot tell a divide from a reciprocal")
	}
	for _, ta := range []Transpose{NoTrans, Trans} {
		x := []float32{b}
		Strsv(Upper, ta, NonUnit, 1, []float32{d}, 1, x, 1)
		if x[0] != b/d {
			t.Errorf("Strsv %c: got %v, want the divide %v (the reciprocal would give %v)", ta, x[0], b/d, b*(1/d))
		}
	}
}
