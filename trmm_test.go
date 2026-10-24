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

// Sizes straddle internal/block.MB (64) so the multi-block path runs, with both
// a partial last block and an exact multiple. 0 and 1 are the degenerate edges.
var trmmDims = []int{0, 1, 5, 64, 65, 130}

var nanPoison = float32(math.NaN()) // NaN, unlike l1_test.go's finite poison: a read anywhere propagates

// paddedMat returns a rows×cols matrix with leading dimension cols+pad, its
// padding poisoned so a read of it shows up as NaN in the result.
func paddedMat(r *rand.Rand, rows, cols, pad int) ([]float32, int) {
	ld := max(cols+pad, 1)
	m := make([]float32, max(rows*ld, 1))
	for i := range m {
		m[i] = nanPoison
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m[i*ld+j] = float32(r.NormFloat64())
		}
	}
	return m, ld
}

// poisonTriangle overwrites the triangle a routine must not read, and the
// diagonal too when unit, so reading either poisons the result.
func poisonTriangle(a []float32, d, lda int, lower, unit bool) {
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if (lower && j > i) || (!lower && j < i) || (unit && i == j) {
				a[i*lda+j] = nanPoison
			}
		}
	}
}

func TestStrmmLattice(t *testing.T) {
	r := rand.New(rand.NewSource(37))
	for _, s := range []Side{Left, Right} {
		for _, ul := range []Uplo{Upper, Lower} {
			for _, ta := range []Transpose{NoTrans, Trans} {
				for _, d := range []Diag{NonUnit, Unit} {
					for _, m := range trmmDims {
						for _, n := range trmmDims {
							da := n
							if s == Left {
								da = m
							}
							a, lda := paddedMat(r, da, da, 3)
							poisonTriangle(a, da, lda, ul == Lower, d == Unit)
							for _, alpha := range []float32{0, 1, -0.75} {
								b, ldb := paddedMat(r, m, n, 2)
								got := append([]float32(nil), b...)
								Strmm(s, ul, ta, d, m, n, alpha, a, lda, got, ldb)
								want, scale := oracle.Trmm(s == Left, ul == Lower, ta == Trans, d == Unit, m, n, alpha, a, lda, b, ldb)
								what := fmt.Sprintf("Strmm %c%c%c%c m=%d n=%d alpha=%v", s, ul, ta, d, m, n, alpha)
								for i := 0; i < m; i++ {
									for j := 0; j < n; j++ {
										checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), max(da, 1), got[i*ldb+j], want[i*n+j], scale[i*n+j])
									}
								}
								// The padding is outside B and must come back untouched.
								for i := 0; i < m; i++ {
									for j := n; j < ldb; j++ {
										if !math.IsNaN(float64(got[i*ldb+j])) {
											t.Fatalf("%s: wrote B's padding at [%d,%d]", what, i, j)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestSsyr2kLattice(t *testing.T) {
	r := rand.New(rand.NewSource(41))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, ta := range []Transpose{NoTrans, Trans} {
			for _, n := range trmmDims {
				for _, k := range []int{0, 1, 7, 70} {
					ra, ca := n, k
					if ta == Trans {
						ra, ca = k, n
					}
					a, lda := paddedMat(r, ra, ca, 1)
					b, ldb := paddedMat(r, ra, ca, 2)
					for _, alpha := range []float32{0, 1, -0.75} {
						for _, beta := range []float32{0, 1, 0.5} {
							c, ldc := paddedMat(r, n, n, 3)
							// C's other triangle must survive untouched, and at beta = 0
							// C must not be read at all, so nanPoison both.
							for i := 0; i < n; i++ {
								for j := 0; j < n; j++ {
									if !oracle.InTriangle(lower, i, j) || beta == 0 {
										c[i*ldc+j] = nanPoison
									}
								}
							}
							got := append([]float32(nil), c...)
							Ssyr2k(ul, ta, n, k, alpha, a, lda, b, ldb, beta, got, ldc)
							ref := c
							if beta == 0 {
								ref = make([]float32, len(c))
							}
							want, scale := oracle.Syr2k(lower, ta == Trans, n, k, alpha, a, lda, b, ldb, beta, ref, ldc)
							what := fmt.Sprintf("Ssyr2k %c%c n=%d k=%d alpha=%v beta=%v", ul, ta, n, k, alpha, beta)
							for i := 0; i < n; i++ {
								for j := 0; j < n; j++ {
									if !oracle.InTriangle(lower, i, j) {
										if !math.IsNaN(float64(got[i*ldc+j])) {
											t.Fatalf("%s: wrote the other triangle at [%d,%d]", what, i, j)
										}
										continue
									}
									checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), max(2*k, 1), got[i*ldc+j], want[i*n+j], scale[i*n+j])
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestStrmmSsyr2kRejectBadArguments(t *testing.T) {
	a := make([]float32, 16)
	mustPanic(t, "Strmm bad side", func() { Strmm('X', Upper, NoTrans, NonUnit, 2, 2, 1, a, 2, a, 2) })
	mustPanic(t, "Strmm bad diag", func() { Strmm(Left, Upper, NoTrans, 'X', 2, 2, 1, a, 2, a, 2) })
	mustPanic(t, "Strmm negative m", func() { Strmm(Left, Upper, NoTrans, NonUnit, -1, 2, 1, a, 2, a, 2) })
	mustPanic(t, "Strmm lda < m", func() { Strmm(Left, Upper, NoTrans, NonUnit, 3, 2, 1, a, 2, a, 2) })
	mustPanic(t, "Strmm short b", func() { Strmm(Right, Upper, NoTrans, NonUnit, 4, 4, 1, a, 4, a[:8], 4) })
	mustPanic(t, "Ssyr2k bad uplo", func() { Ssyr2k('X', NoTrans, 2, 2, 1, a, 2, a, 2, 0, a, 2) })
	mustPanic(t, "Ssyr2k short b", func() { Ssyr2k(Upper, NoTrans, 4, 4, 1, a, 4, a[:8], 4, 0, a, 4) })
	mustPanic(t, "Ssyr2k ldc < n", func() { Ssyr2k(Upper, NoTrans, 3, 1, 1, a, 1, a, 1, 0, a, 2) })
}
