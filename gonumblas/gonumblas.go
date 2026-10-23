// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package gonumblas makes keel a drop-in float32 BLAS for Gonum.
//
//	import (
//		"gonum.org/v1/gonum/blas/blas32"
//		"github.com/scttfrdmn/keel/gonumblas"
//	)
//
//	func init() { blas32.Use(gonumblas.Implementation{}) }
//
// Implementation satisfies gonum's blas.Float32. The routines keel implements
// (Sdot, Saxpy, Sscal, Snrm2, Sasum, Isamax, Sgemv, Sger, Sgemm, Ssyrk, Ssymm,
// Strsm) run in keel; every other routine falls through to gonum's own pure-Go
// implementation, which is embedded. Both libraries use row-major storage with
// explicit leading dimensions, so arguments pass through unchanged.
//
// Four deliberate translations, all matching gonum's semantics:
//
//   - blas.ConjTrans is accepted and treated as blas.Trans, since the data is real.
//     keel itself rejects ConjTrans.
//   - A negative stride on Sscal, Sasum, Snrm2 or Isamax is passed to gonum, which
//     follows reference BLAS: no-op, 0, 0 and -1. keel itself panics there.
//   - Sgemv with m == 0 or n == 0 leaves y untouched, as reference BLAS does.
//     keel itself returns y = beta*y there, deliberately (see keel.Sgemv).
//   - An enum value neither library accepts (blas.All as an Uplo, an unknown
//     Transpose) is passed to gonum, so the caller gets gonum's panic, exactly
//     as with blas32.Use(gonum.Implementation{}).
//
// Other invalid arguments (a short slice, a bad leading dimension) panic in keel
// with keel's messages. The panicking inputs are the same set gonum panics on;
// TestPanicsWhereGonumPanics holds that.
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
