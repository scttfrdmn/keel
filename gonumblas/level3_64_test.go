// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel"
	"gonum.org/v1/gonum/blas"
)

// The third arm of DESIGN.md §5 rule 1 for float64 Level 3 (#192): keel's D
// routines called directly, against Gonum's native float64, on the same operands,
// at one global scale per call (as level2_64_test.go). Dimensions straddle keel's
// microkernel tiles, and k = 600 crosses block64's KC = 512.

// mat64g is a rows×cols matrix at stride cols+pad, NaN in the padding.
func mat64g(r *rand.Rand, rows, cols, pad int) ([]float64, int) {
	ld := max(cols+pad, 1)
	m := make([]float64, max(rows*ld, 1))
	for i := range m {
		m[i] = math.NaN()
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m[i*ld+j] = r.NormFloat64()
		}
	}
	return m, ld
}

func TestLevel3AgreesWithGonum64(t *testing.T) {
	r := rand.New(rand.NewSource(31))
	ts := []blas.Transpose{blas.NoTrans, blas.Trans}
	shapes := [][3]int{{0, 3, 2}, {1, 1, 1}, {7, 9, 5}, {17, 13, 33}, {37, 29, 600}}
	for _, sh := range shapes {
		m, n, k := sh[0], sh[1], sh[2]
		for _, ta := range ts {
			for _, tb := range ts {
				ra, ca, rb, cb := m, k, k, n
				if ta == blas.Trans {
					ra, ca = k, m
				}
				if tb == blas.Trans {
					rb, cb = n, k
				}
				a, lda := mat64g(r, ra, ca, 1)
				b, ldb := mat64g(r, rb, cb, 2)
				c, ldc := mat64g(r, m, n, 1)
				ck, cg := append([]float64(nil), c...), append([]float64(nil), c...)
				keel.Dgemm(keel.Transpose(ta), keel.Transpose(tb), m, n, k, -0.75, a, lda, b, ldb, 0.5, ck, ldc)
				gonumImpl.Dgemm(ta, tb, m, n, k, -0.75, a, lda, b, ldb, 0.5, cg, ldc)
				cmp64(t, fmt.Sprintf("Dgemm %c%c m=%d n=%d k=%d", ta, tb, m, n, k), ck, cg, k+1,
					float64(max(k, 1))*maxAbs64(a)*maxAbs64(b)+maxAbs64(c))
			}
		}
		for _, ul := range []blas.Uplo{blas.Upper, blas.Lower} {
			lower := ul == blas.Lower
			kU := keel.Uplo(ul)
			for _, ta := range ts {
				ra, ca := n, k
				if ta == blas.Trans {
					ra, ca = k, n
				}
				a, lda := mat64g(r, ra, ca, 1)
				b, ldb := mat64g(r, ra, ca, 1)
				for _, beta := range []float64{0, 0.5} {
					c, ldc := mat64g(r, n, n, 2)
					what := fmt.Sprintf("%c%c n=%d k=%d beta=%v", ul, ta, n, k, beta)
					sc := 2*float64(max(k, 1))*maxAbs64(a)*maxAbs64(b) + maxAbs64(c)
					ck, cg := append([]float64(nil), c...), append([]float64(nil), c...)
					keel.Dsyrk(kU, keel.Transpose(ta), n, k, -0.75, a, lda, beta, ck, ldc)
					gonumImpl.Dsyrk(ul, ta, n, k, -0.75, a, lda, beta, cg, ldc)
					cmp64(t, "Dsyrk "+what, ck, cg, k+1, sc)
					ck, cg = append([]float64(nil), c...), append([]float64(nil), c...)
					keel.Dsyr2k(kU, keel.Transpose(ta), n, k, -0.75, a, lda, b, ldb, beta, ck, ldc)
					gonumImpl.Dsyr2k(ul, ta, n, k, -0.75, a, lda, b, ldb, beta, cg, ldc)
					cmp64(t, "Dsyr2k "+what, ck, cg, 2*k+1, sc)
				}
			}
			for _, s := range []blas.Side{blas.Left, blas.Right} {
				da := n
				if s == blas.Left {
					da = m
				}
				kS := keel.Side(s)
				bm, ldb := mat64g(r, m, n, 1)
				as, lds := triDense64(r, da, lower, false)
				c, ldc := mat64g(r, m, n, 2)
				ck, cg := append([]float64(nil), c...), append([]float64(nil), c...)
				keel.Dsymm(kS, kU, m, n, -0.75, as, lds, bm, ldb, 0.5, ck, ldc)
				gonumImpl.Dsymm(s, ul, m, n, -0.75, as, lds, bm, ldb, 0.5, cg, ldc)
				cmp64(t, fmt.Sprintf("Dsymm %c%c m=%d n=%d", s, ul, m, n), ck, cg, da+1,
					float64(max(da, 1))*maxAbs64(as)*maxAbs64(bm)+maxAbs64(c))
				for _, ta := range ts {
					for _, dg := range []blas.Diag{blas.NonUnit, blas.Unit} {
						at, ldt := triDense64(r, da, lower, dg == blas.Unit)
						sc := float64(max(da, 1)) * maxAbs64(at) * maxAbs64(bm)
						what := fmt.Sprintf("%c%c%c%c m=%d n=%d", s, ul, ta, dg, m, n)
						bk, bg := append([]float64(nil), bm...), append([]float64(nil), bm...)
						keel.Dtrmm(kS, kU, keel.Transpose(ta), keel.Diag(dg), m, n, 1.5, at, ldt, bk, ldb)
						gonumImpl.Dtrmm(s, ul, ta, dg, m, n, 1.5, at, ldt, bg, ldb)
						cmp64(t, "Dtrmm "+what, bk, bg, da+1, 1.5*sc)
						bk, bg = append([]float64(nil), bm...), append([]float64(nil), bm...)
						keel.Dtrsm(kS, kU, keel.Transpose(ta), keel.Diag(dg), m, n, 1.5, at, ldt, bk, ldb)
						gonumImpl.Dtrsm(s, ul, ta, dg, m, n, 1.5, at, ldt, bg, ldb)
						cmp64(t, "Dtrsm "+what, bk, bg, da+1, maxAbs64(bg)*sc+1.5*maxAbs64(bm))
					}
				}
			}
		}
	}
}
