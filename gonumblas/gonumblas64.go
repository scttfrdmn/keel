// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"github.com/scttfrdmn/keel"
	"gonum.org/v1/gonum/blas"
)

// The blas.Float64 half of Implementation (#192): gonumblas.go's float32 adapter at
// float64, with the same five translations and the same rule where gonum is wrong
// rather than different (Dger keeps keel's answer at every stride; gonum/gonum#2105
// is in f64.Ger as well as f32.Ger). 34 of blas.Float64's 35 routines run in keel.
// The 35th, Dsdot, takes float32 vectors and keel has no twin for it, so it stays
// gonum's, through the embedding.

var _ blas.Float64 = Implementation{}

// Level 1.

func (Implementation) Ddot(n int, x []float64, incX int, y []float64, incY int) float64 {
	return keel.Ddot(n, x, incX, y, incY)
}

func (Implementation) Daxpy(n int, alpha float64, x []float64, incX int, y []float64, incY int) {
	keel.Daxpy(n, alpha, x, incX, y, incY)
}

func (g Implementation) Dscal(n int, alpha float64, x []float64, incX int) {
	if incX < 0 {
		g.Implementation.Dscal(n, alpha, x, incX) // a no-op, by reference BLAS
		return
	}
	keel.Dscal(n, alpha, x, incX)
}

func (g Implementation) Dnrm2(n int, x []float64, incX int) float64 {
	if incX < 0 {
		return g.Implementation.Dnrm2(n, x, incX) // 0, by reference BLAS
	}
	return keel.Dnrm2(n, x, incX)
}

func (g Implementation) Dasum(n int, x []float64, incX int) float64 {
	if incX < 0 {
		return g.Implementation.Dasum(n, x, incX) // 0, by reference BLAS
	}
	return keel.Dasum(n, x, incX)
}

func (g Implementation) Idamax(n int, x []float64, incX int) int {
	if incX < 0 {
		return g.Implementation.Idamax(n, x, incX) // -1, by reference BLAS
	}
	return keel.Idamax(n, x, incX)
}

// Level 2.

func (g Implementation) Dgemv(tA blas.Transpose, m, n int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	t, ok := trans(tA)
	// keel deliberately returns y = beta*y for an empty reduction (see keel.Sgemv);
	// reference BLAS, and so gonum, returns without touching y. No arithmetic is
	// lost by handing these shapes to gonum.
	if !ok || m == 0 || n == 0 {
		g.Implementation.Dgemv(tA, m, n, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Dgemv(t, m, n, alpha, a, lda, x, incX, beta, y, incY)
}

func (Implementation) Dger(m, n int, alpha float64, x []float64, incX int, y []float64, incY int, a []float64, lda int) {
	keel.Dger(m, n, alpha, x, incX, y, incY, a, lda)
}

// Level 3.

func (g Implementation) Dgemm(tA, tB blas.Transpose, m, n, k int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	ta, okA := trans(tA)
	tb, okB := trans(tB)
	if !okA || !okB {
		g.Implementation.Dgemm(tA, tB, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Dgemm(ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Dsyrk(ul blas.Uplo, tA blas.Transpose, n, k int, alpha float64, a []float64, lda int, beta float64, c []float64, ldc int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	if !okU || !okT {
		g.Implementation.Dsyrk(ul, tA, n, k, alpha, a, lda, beta, c, ldc)
		return
	}
	keel.Dsyrk(u, t, n, k, alpha, a, lda, beta, c, ldc)
}

func (g Implementation) Dsymm(s blas.Side, ul blas.Uplo, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	if !okS || !okU {
		g.Implementation.Dsymm(s, ul, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Dsymm(sd, u, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Dtrsm(s blas.Side, ul blas.Uplo, tA blas.Transpose, d blas.Diag, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okS || !okU || !okT || !okD {
		g.Implementation.Dtrsm(s, ul, tA, d, m, n, alpha, a, lda, b, ldb)
		return
	}
	keel.Dtrsm(sd, u, t, dg, m, n, alpha, a, lda, b, ldb)
}

func (g Implementation) Dtrmm(s blas.Side, ul blas.Uplo, tA blas.Transpose, d blas.Diag, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okS || !okU || !okT || !okD {
		g.Implementation.Dtrmm(s, ul, tA, d, m, n, alpha, a, lda, b, ldb)
		return
	}
	keel.Dtrmm(sd, u, t, dg, m, n, alpha, a, lda, b, ldb)
}

func (g Implementation) Dsyr2k(ul blas.Uplo, tA blas.Transpose, n, k int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	if !okU || !okT {
		g.Implementation.Dsyr2k(ul, tA, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Dsyr2k(u, t, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Dtrmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, a []float64, lda int, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtrmv(ul, tA, d, n, a, lda, x, incX)
		return
	}
	keel.Dtrmv(u, t, dg, n, a, lda, x, incX)
}

func (g Implementation) Dtrsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, a []float64, lda int, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtrsv(ul, tA, d, n, a, lda, x, incX)
		return
	}
	keel.Dtrsv(u, t, dg, n, a, lda, x, incX)
}

func (g Implementation) Dsymv(ul blas.Uplo, n int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dsymv(ul, n, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Dsymv(u, n, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Dsyr(ul blas.Uplo, n int, alpha float64, x []float64, incX int, a []float64, lda int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dsyr(ul, n, alpha, x, incX, a, lda)
		return
	}
	keel.Dsyr(u, n, alpha, x, incX, a, lda)
}

func (g Implementation) Dsyr2(ul blas.Uplo, n int, alpha float64, x []float64, incX int, y []float64, incY int, a []float64, lda int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dsyr2(ul, n, alpha, x, incX, y, incY, a, lda)
		return
	}
	keel.Dsyr2(u, n, alpha, x, incX, y, incY, a, lda)
}

func (Implementation) Dcopy(n int, x []float64, incX int, y []float64, incY int) {
	keel.Dcopy(n, x, incX, y, incY)
}

func (Implementation) Dswap(n int, x []float64, incX int, y []float64, incY int) {
	keel.Dswap(n, x, incX, y, incY)
}

func (Implementation) Drot(n int, x []float64, incX int, y []float64, incY int, c, s float64) {
	keel.Drot(n, x, incX, y, incY, c, s)
}

func (Implementation) Drotg(a, b float64) (c, s, r, z float64) {
	return keel.Drotg(a, b)
}

// toParam64 and fromParam64 are toParam and fromParam for gonum's DrotmParams and
// reference BLAS's DPARAM. The H order is the same (h11, h21, h12, h22), and
// so are the flag values (-2, -1, 0, 1).
func toParam64(p blas.DrotmParams) [5]float64 {
	return [5]float64{float64(p.Flag), p.H[0], p.H[1], p.H[2], p.H[3]}
}

func fromParam64(q [5]float64) blas.DrotmParams {
	return blas.DrotmParams{Flag: blas.Flag(q[0]), H: [4]float64{q[1], q[2], q[3], q[4]}}
}

func (g Implementation) Drotm(n int, x []float64, incX int, y []float64, incY int, p blas.DrotmParams) {
	switch p.Flag {
	case blas.Identity, blas.Rescaling, blas.OffDiagonal, blas.Diagonal:
		keel.Drotm(n, x, incX, y, incY, toParam64(p))
	default:
		// keel panics on an unknown flag; gonum's own behaviour is what a drop-in owes.
		g.Implementation.Drotm(n, x, incX, y, incY, p)
	}
}

func (Implementation) Drotmg(d1, d2, x1, y1 float64) (p blas.DrotmParams, rd1, rd2, rx1 float64) {
	q, rd1, rd2, rx1 := keel.Drotmg(d1, d2, x1, y1)
	return fromParam64(q), rd1, rd2, rx1
}

func (g Implementation) Dgbmv(tA blas.Transpose, m, n, kL, kU int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	t, ok := trans(tA)
	// As Sgemv: keel returns y = beta*y for an empty reduction, reference BLAS and
	// gonum leave y untouched, and these shapes carry no arithmetic to lose.
	if !ok || m == 0 || n == 0 {
		g.Implementation.Dgbmv(tA, m, n, kL, kU, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Dgbmv(t, m, n, kL, kU, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Dtbmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n, k int, a []float64, lda int, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtbmv(ul, tA, d, n, k, a, lda, x, incX)
		return
	}
	keel.Dtbmv(u, t, dg, n, k, a, lda, x, incX)
}

func (g Implementation) Dtbsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n, k int, a []float64, lda int, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtbsv(ul, tA, d, n, k, a, lda, x, incX)
		return
	}
	keel.Dtbsv(u, t, dg, n, k, a, lda, x, incX)
}

func (g Implementation) Dsbmv(ul blas.Uplo, n, k int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dsbmv(ul, n, k, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Dsbmv(u, n, k, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Dtpmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, ap []float64, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtpmv(ul, tA, d, n, ap, x, incX)
		return
	}
	keel.Dtpmv(u, t, dg, n, ap, x, incX)
}

func (g Implementation) Dtpsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, ap []float64, x []float64, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Dtpsv(ul, tA, d, n, ap, x, incX)
		return
	}
	keel.Dtpsv(u, t, dg, n, ap, x, incX)
}

func (g Implementation) Dspmv(ul blas.Uplo, n int, alpha float64, ap []float64, x []float64, incX int, beta float64, y []float64, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dspmv(ul, n, alpha, ap, x, incX, beta, y, incY)
		return
	}
	keel.Dspmv(u, n, alpha, ap, x, incX, beta, y, incY)
}

func (g Implementation) Dspr(ul blas.Uplo, n int, alpha float64, x []float64, incX int, ap []float64) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dspr(ul, n, alpha, x, incX, ap)
		return
	}
	keel.Dspr(u, n, alpha, x, incX, ap)
}

func (g Implementation) Dspr2(ul blas.Uplo, n int, alpha float64, x []float64, incX int, y []float64, incY int, ap []float64) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Dspr2(ul, n, alpha, x, incX, y, incY, ap)
		return
	}
	keel.Dspr2(u, n, alpha, x, incX, y, incY, ap)
}
