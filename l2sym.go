// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import "github.com/scttfrdmn/keel/internal/l2"

// The triangular and symmetric Level-2 routines. Every one is a loop over A's
// rows, each row being a contiguous run of row-major storage, so the work goes
// through the active Level-1 backend's Dot and Axpy, which is where the vector
// code lives. Strided x and y are gathered into a contiguous buffer once per call,
// as Sgemv and Sger do.
//
// Shared conventions, matching Sgemv and Sger:
//   - Either stride may be negative, and the vector then runs backwards from its
//     far end, as in reference BLAS. A zero stride panics.
//   - alpha == 0 returns without reading x or y (Ssymv still applies beta to y).
//   - No row is skipped because its scale is zero. Reference SSYR and SSYR2 skip
//     such rows; keel computes them, for the reason internal/l2's gemvTrans gives.
//   - Triangular A: the other triangle is never read, and with Unit neither is the
//     diagonal, which is taken to be 1.

// Strmv computes x = op(A)·x for A n×n triangular, stored in the triangle ul names.
func Strmv(ul Uplo, tA Transpose, d Diag, n int, a []float32, lda int, x []float32, incX int) {
	lower := checkUplo("Strmv", "ul", ul)
	trans := checkTranspose("Strmv", "tA", tA)
	unit := checkDiag("Strmv", "d", d)
	if n < 0 {
		panic("keel: Strmv: negative dimension")
	}
	checkMatrix("Strmv", "a", n, n, a, lda)
	checkVector("Strmv", "x", n, x, incX)
	l2.Trmv(activeL1, lower, trans, unit, n, a, lda, x, incX)
}

// Strsv solves op(A)·x = b for A n×n triangular, stored in the triangle ul names,
// overwriting b (passed as x) with the solution. It divides by the diagonal rather
// than multiplying by a reciprocal, and has no fast path for a zero multiplier, for
// the reasons Strsm gives. A must be nonsingular in its referenced triangle; keel
// does not check that, and neither does reference BLAS.
func Strsv(ul Uplo, tA Transpose, d Diag, n int, a []float32, lda int, x []float32, incX int) {
	lower := checkUplo("Strsv", "ul", ul)
	trans := checkTranspose("Strsv", "tA", tA)
	unit := checkDiag("Strsv", "d", d)
	if n < 0 {
		panic("keel: Strsv: negative dimension")
	}
	checkMatrix("Strsv", "a", n, n, a, lda)
	checkVector("Strsv", "x", n, x, incX)
	l2.Trsv(activeL1, lower, trans, unit, n, a, lda, x, incX)
}

// Ssymv computes y = alpha·A·x + beta·y for A n×n symmetric, stored in the triangle
// ul names; the other triangle is never read. beta == 0 does not read y.
func Ssymv(ul Uplo, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	lower := checkUplo("Ssymv", "ul", ul)
	if n < 0 {
		panic("keel: Ssymv: negative dimension")
	}
	checkMatrix("Ssymv", "a", n, n, a, lda)
	checkVector("Ssymv", "x", n, x, incX)
	checkVector("Ssymv", "y", n, y, incY)
	l2.Symv(activeL1, lower, n, alpha, a, lda, x, incX, beta, y, incY)
}

// Ssyr computes A += alpha·x·xᵀ on the triangle of the n×n symmetric A that ul
// names; the other triangle is neither read nor written.
func Ssyr(ul Uplo, n int, alpha float32, x []float32, incX int, a []float32, lda int) {
	lower := checkUplo("Ssyr", "ul", ul)
	if n < 0 {
		panic("keel: Ssyr: negative dimension")
	}
	checkVector("Ssyr", "x", n, x, incX)
	checkMatrix("Ssyr", "a", n, n, a, lda)
	l2.Syr(activeL1, lower, n, alpha, x, incX, a, lda)
}

// Ssyr2 computes A += alpha·(x·yᵀ + y·xᵀ) on the triangle of the n×n symmetric A
// that ul names; the other triangle is neither read nor written.
func Ssyr2(ul Uplo, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	lower := checkUplo("Ssyr2", "ul", ul)
	if n < 0 {
		panic("keel: Ssyr2: negative dimension")
	}
	checkVector("Ssyr2", "x", n, x, incX)
	checkVector("Ssyr2", "y", n, y, incY)
	checkMatrix("Ssyr2", "a", n, n, a, lda)
	l2.Syr2(activeL1, lower, n, alpha, x, incX, y, incY, a, lda)
}
