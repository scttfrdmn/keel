// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package gonumblas makes keel a drop-in float32 and float64 BLAS for Gonum.
//
//	import (
//		"gonum.org/v1/gonum/blas/blas32"
//		"gonum.org/v1/gonum/blas/blas64"
//		"github.com/scttfrdmn/keel/gonumblas"
//	)
//
//	func init() {
//		blas32.Use(gonumblas.Implementation{})
//		blas64.Use(gonumblas.Implementation{})
//	}
//
// Implementation satisfies gonum's blas.Float32, and every one of its 35 routines
// runs in keel. It satisfies blas.Float64 too, and 34 of those 35 run in keel; see
// gonumblas64.go for the float64 half, which follows everything below at float64. gonum's own implementation is still embedded, for the five
// translations below and for any routine a future gonum adds. Both libraries use
// row-major storage with explicit leading dimensions, so arguments pass through
// unchanged; that includes gonum's row-major band and packed layouts, which keel
// adopts as-is.
//
// Five deliberate translations, all matching gonum's semantics:
//
//   - blas.ConjTrans is accepted and treated as blas.Trans, since the data is real.
//     keel itself rejects ConjTrans.
//   - A negative stride on Sscal, Sasum, Snrm2 or Isamax is passed to gonum, which
//     follows reference BLAS: no-op, 0, 0 and -1. keel itself panics there.
//   - Sgemv and Sgbmv with m == 0 or n == 0 leave y untouched, as reference BLAS does.
//     keel itself returns y = beta*y there, deliberately (see keel.Sgemv).
//   - Srotm with a flag outside the four defined ones gets gonum's behaviour, where
//     keel itself panics.
//   - An enum value neither library accepts (blas.All as an Uplo, an unknown
//     Transpose) is passed to gonum, so the caller gets gonum's panic, exactly
//     as with blas32.Use(gonum.Implementation{}).
//
// Other invalid arguments (a short slice, a bad leading dimension) panic in keel
// with keel's messages. The panicking inputs are the same set gonum panics on;
// TestPanicsWhereGonumPanics holds that.
//
// Where gonum is wrong rather than different, the adapter keeps keel's answer:
// Ssyrk with Lower, Trans and beta == 0 (gonum/gonum#2129), Sger with negative
// strides on amd64 (gonum/gonum#2105), and Sdsdot, which gonum rounds twice and
// which returns 0 rather than alpha for n == 0 (gonum/gonum#2130).
//
// Results agree with gonum within keel's numerical tolerance model
// (internal/oracle), not bit-for-bit: keel's vector kernels sum in a different
// order and use fused multiply-adds.
package gonumblas

import (
	"github.com/scttfrdmn/keel"
	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/blas/gonum"
)

// Implementation is gonum's blas.Float32 with keel underneath. The zero value is
// ready to use.
type Implementation struct {
	gonum.Implementation
}

var _ blas.Float32 = Implementation{}

// trans maps a gonum Transpose to keel's, reporting whether keel can take it.
func trans(t blas.Transpose) (keel.Transpose, bool) {
	switch t {
	case blas.NoTrans:
		return keel.NoTrans, true
	case blas.Trans, blas.ConjTrans:
		return keel.Trans, true
	}
	return 0, false
}

func uplo(u blas.Uplo) (keel.Uplo, bool) {
	switch u {
	case blas.Upper:
		return keel.Upper, true
	case blas.Lower:
		return keel.Lower, true
	}
	return 0, false
}

func side(s blas.Side) (keel.Side, bool) {
	switch s {
	case blas.Left:
		return keel.Left, true
	case blas.Right:
		return keel.Right, true
	}
	return 0, false
}

func diag(d blas.Diag) (keel.Diag, bool) {
	switch d {
	case blas.NonUnit:
		return keel.NonUnit, true
	case blas.Unit:
		return keel.Unit, true
	}
	return 0, false
}

// Level 1.

func (Implementation) Sdot(n int, x []float32, incX int, y []float32, incY int) float32 {
	return keel.Sdot(n, x, incX, y, incY)
}

func (Implementation) Saxpy(n int, alpha float32, x []float32, incX int, y []float32, incY int) {
	keel.Saxpy(n, alpha, x, incX, y, incY)
}

func (g Implementation) Sscal(n int, alpha float32, x []float32, incX int) {
	if incX < 0 {
		g.Implementation.Sscal(n, alpha, x, incX) // a no-op, by reference BLAS
		return
	}
	keel.Sscal(n, alpha, x, incX)
}

func (g Implementation) Snrm2(n int, x []float32, incX int) float32 {
	if incX < 0 {
		return g.Implementation.Snrm2(n, x, incX) // 0, by reference BLAS
	}
	return keel.Snrm2(n, x, incX)
}

func (g Implementation) Sasum(n int, x []float32, incX int) float32 {
	if incX < 0 {
		return g.Implementation.Sasum(n, x, incX) // 0, by reference BLAS
	}
	return keel.Sasum(n, x, incX)
}

func (g Implementation) Isamax(n int, x []float32, incX int) int {
	if incX < 0 {
		return g.Implementation.Isamax(n, x, incX) // -1, by reference BLAS
	}
	return keel.Isamax(n, x, incX)
}

// Level 2.

func (g Implementation) Sgemv(tA blas.Transpose, m, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	t, ok := trans(tA)
	// keel deliberately returns y = beta*y for an empty reduction (see keel.Sgemv);
	// reference BLAS, and so gonum, returns without touching y. No arithmetic is
	// lost by handing these shapes to gonum.
	if !ok || m == 0 || n == 0 {
		g.Implementation.Sgemv(tA, m, n, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Sgemv(t, m, n, alpha, a, lda, x, incX, beta, y, incY)
}

func (Implementation) Sger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	keel.Sger(m, n, alpha, x, incX, y, incY, a, lda)
}

// Level 3.

func (g Implementation) Sgemm(tA, tB blas.Transpose, m, n, k int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	ta, okA := trans(tA)
	tb, okB := trans(tB)
	if !okA || !okB {
		g.Implementation.Sgemm(tA, tB, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Sgemm(ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Ssyrk(ul blas.Uplo, tA blas.Transpose, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	if !okU || !okT {
		g.Implementation.Ssyrk(ul, tA, n, k, alpha, a, lda, beta, c, ldc)
		return
	}
	keel.Ssyrk(u, t, n, k, alpha, a, lda, beta, c, ldc)
}

func (g Implementation) Ssymm(s blas.Side, ul blas.Uplo, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	if !okS || !okU {
		g.Implementation.Ssymm(s, ul, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Ssymm(sd, u, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Strsm(s blas.Side, ul blas.Uplo, tA blas.Transpose, d blas.Diag, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okS || !okU || !okT || !okD {
		g.Implementation.Strsm(s, ul, tA, d, m, n, alpha, a, lda, b, ldb)
		return
	}
	keel.Strsm(sd, u, t, dg, m, n, alpha, a, lda, b, ldb)
}

func (g Implementation) Strmm(s blas.Side, ul blas.Uplo, tA blas.Transpose, d blas.Diag, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
	sd, okS := side(s)
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okS || !okU || !okT || !okD {
		g.Implementation.Strmm(s, ul, tA, d, m, n, alpha, a, lda, b, ldb)
		return
	}
	keel.Strmm(sd, u, t, dg, m, n, alpha, a, lda, b, ldb)
}

func (g Implementation) Ssyr2k(ul blas.Uplo, tA blas.Transpose, n, k int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	if !okU || !okT {
		g.Implementation.Ssyr2k(ul, tA, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		return
	}
	keel.Ssyr2k(u, t, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}

func (g Implementation) Strmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, a []float32, lda int, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Strmv(ul, tA, d, n, a, lda, x, incX)
		return
	}
	keel.Strmv(u, t, dg, n, a, lda, x, incX)
}

func (g Implementation) Strsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, a []float32, lda int, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Strsv(ul, tA, d, n, a, lda, x, incX)
		return
	}
	keel.Strsv(u, t, dg, n, a, lda, x, incX)
}

func (g Implementation) Ssymv(ul blas.Uplo, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Ssymv(ul, n, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Ssymv(u, n, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Ssyr(ul blas.Uplo, n int, alpha float32, x []float32, incX int, a []float32, lda int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Ssyr(ul, n, alpha, x, incX, a, lda)
		return
	}
	keel.Ssyr(u, n, alpha, x, incX, a, lda)
}

func (g Implementation) Ssyr2(ul blas.Uplo, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Ssyr2(ul, n, alpha, x, incX, y, incY, a, lda)
		return
	}
	keel.Ssyr2(u, n, alpha, x, incX, y, incY, a, lda)
}

func (Implementation) Scopy(n int, x []float32, incX int, y []float32, incY int) {
	keel.Scopy(n, x, incX, y, incY)
}

func (Implementation) Sswap(n int, x []float32, incX int, y []float32, incY int) {
	keel.Sswap(n, x, incX, y, incY)
}

func (Implementation) Srot(n int, x []float32, incX int, y []float32, incY int, c, s float32) {
	keel.Srot(n, x, incX, y, incY, c, s)
}

func (Implementation) Srotg(a, b float32) (c, s, r, z float32) {
	return keel.Srotg(a, b)
}

// toParam and fromParam convert between gonum's SrotmParams and reference BLAS's
// SPARAM array, which keel uses. The H order is the same (h11, h21, h12, h22), and
// so are the flag values (-2, -1, 0, 1).
func toParam(p blas.SrotmParams) [5]float32 {
	return [5]float32{float32(p.Flag), p.H[0], p.H[1], p.H[2], p.H[3]}
}

func fromParam(q [5]float32) blas.SrotmParams {
	return blas.SrotmParams{Flag: blas.Flag(q[0]), H: [4]float32{q[1], q[2], q[3], q[4]}}
}

func (g Implementation) Srotm(n int, x []float32, incX int, y []float32, incY int, p blas.SrotmParams) {
	switch p.Flag {
	case blas.Identity, blas.Rescaling, blas.OffDiagonal, blas.Diagonal:
		keel.Srotm(n, x, incX, y, incY, toParam(p))
	default:
		// keel panics on an unknown flag; gonum's own behaviour is what a drop-in owes.
		g.Implementation.Srotm(n, x, incX, y, incY, p)
	}
}

func (Implementation) Srotmg(d1, d2, x1, y1 float32) (p blas.SrotmParams, rd1, rd2, rx1 float32) {
	q, rd1, rd2, rx1 := keel.Srotmg(d1, d2, x1, y1)
	return fromParam(q), rd1, rd2, rx1
}

func (Implementation) Sdsdot(n int, alpha float32, x []float32, incX int, y []float32, incY int) float32 {
	return keel.Sdsdot(n, alpha, x, incX, y, incY)
}

func (g Implementation) Sgbmv(tA blas.Transpose, m, n, kL, kU int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	t, ok := trans(tA)
	// As Sgemv: keel returns y = beta*y for an empty reduction, reference BLAS and
	// gonum leave y untouched, and these shapes carry no arithmetic to lose.
	if !ok || m == 0 || n == 0 {
		g.Implementation.Sgbmv(tA, m, n, kL, kU, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Sgbmv(t, m, n, kL, kU, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Stbmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n, k int, a []float32, lda int, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Stbmv(ul, tA, d, n, k, a, lda, x, incX)
		return
	}
	keel.Stbmv(u, t, dg, n, k, a, lda, x, incX)
}

func (g Implementation) Stbsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n, k int, a []float32, lda int, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Stbsv(ul, tA, d, n, k, a, lda, x, incX)
		return
	}
	keel.Stbsv(u, t, dg, n, k, a, lda, x, incX)
}

func (g Implementation) Ssbmv(ul blas.Uplo, n, k int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Ssbmv(ul, n, k, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	keel.Ssbmv(u, n, k, alpha, a, lda, x, incX, beta, y, incY)
}

func (g Implementation) Stpmv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, ap []float32, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Stpmv(ul, tA, d, n, ap, x, incX)
		return
	}
	keel.Stpmv(u, t, dg, n, ap, x, incX)
}

func (g Implementation) Stpsv(ul blas.Uplo, tA blas.Transpose, d blas.Diag, n int, ap []float32, x []float32, incX int) {
	u, okU := uplo(ul)
	t, okT := trans(tA)
	dg, okD := diag(d)
	if !okU || !okT || !okD {
		g.Implementation.Stpsv(ul, tA, d, n, ap, x, incX)
		return
	}
	keel.Stpsv(u, t, dg, n, ap, x, incX)
}

func (g Implementation) Sspmv(ul blas.Uplo, n int, alpha float32, ap []float32, x []float32, incX int, beta float32, y []float32, incY int) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Sspmv(ul, n, alpha, ap, x, incX, beta, y, incY)
		return
	}
	keel.Sspmv(u, n, alpha, ap, x, incX, beta, y, incY)
}

func (g Implementation) Sspr(ul blas.Uplo, n int, alpha float32, x []float32, incX int, ap []float32) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Sspr(ul, n, alpha, x, incX, ap)
		return
	}
	keel.Sspr(u, n, alpha, x, incX, ap)
}

func (g Implementation) Sspr2(ul blas.Uplo, n int, alpha float32, x []float32, incX int, y []float32, incY int, ap []float32) {
	u, ok := uplo(ul)
	if !ok {
		g.Implementation.Sspr2(ul, n, alpha, x, incX, y, incY, ap)
		return
	}
	keel.Sspr2(u, n, alpha, x, incX, y, incY, ap)
}
