// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import "github.com/scttfrdmn/keel/internal/l264"

// Level 2 at float64 (#192). Each routine is its S twin at float64: the same
// checks and panic messages, and the same body, internal/l264, which tools/f64gen
// generates from internal/l2 (ruled 2026-10-29: Level 2 is generated, because it is
// algorithm and not a thin wrapper). The S routine's doc comment is the contract,
// including both deviations from reference BLAS that Sgemv and Sgbmv document.
// TestDParity2 pins each pair to the same behaviour on bad arguments.
//
// float64 Level 2 parallelizes by l264.Grain, which is float32's element count
// (1<<17) and so twice the bytes per worker. It is unmeasured at float64.

// Dgemv is Sgemv at float64; see Sgemv.
func Dgemv(tA Transpose, m, n int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	trans := checkTranspose("Dgemv", "tA", tA)
	if m < 0 || n < 0 {
		panic("keel: Dgemv: negative dimension")
	}
	checkMatrix64("Dgemv", "a", m, n, a, lda)
	rows, inner := m, n
	if trans {
		rows, inner = n, m
	}
	checkVector64("Dgemv", "x", inner, x, incX)
	checkVector64("Dgemv", "y", rows, y, incY)
	l264.Gemv(activeL164, trans, m, n, alpha, a, lda, x, incX, beta, y, incY)
}

// Dger is Sger at float64; see Sger.
func Dger(m, n int, alpha float64, x []float64, incX int, y []float64, incY int, a []float64, lda int) {
	if m < 0 || n < 0 {
		panic("keel: Dger: negative dimension")
	}
	checkVector64("Dger", "x", m, x, incX)
	checkVector64("Dger", "y", n, y, incY)
	checkMatrix64("Dger", "a", m, n, a, lda)
	l264.Ger(activeL164, m, n, alpha, x, incX, y, incY, a, lda)
}

// Dtrmv is Strmv at float64; see Strmv.
func Dtrmv(ul Uplo, tA Transpose, d Diag, n int, a []float64, lda int, x []float64, incX int) {
	lower := checkUplo("Dtrmv", "ul", ul)
	trans := checkTranspose("Dtrmv", "tA", tA)
	unit := checkDiag("Dtrmv", "d", d)
	if n < 0 {
		panic("keel: Dtrmv: negative dimension")
	}
	checkMatrix64("Dtrmv", "a", n, n, a, lda)
	checkVector64("Dtrmv", "x", n, x, incX)
	l264.Trmv(activeL164, lower, trans, unit, n, a, lda, x, incX)
}

// Dtrsv is Strsv at float64; see Strsv.
func Dtrsv(ul Uplo, tA Transpose, d Diag, n int, a []float64, lda int, x []float64, incX int) {
	lower := checkUplo("Dtrsv", "ul", ul)
	trans := checkTranspose("Dtrsv", "tA", tA)
	unit := checkDiag("Dtrsv", "d", d)
	if n < 0 {
		panic("keel: Dtrsv: negative dimension")
	}
	checkMatrix64("Dtrsv", "a", n, n, a, lda)
	checkVector64("Dtrsv", "x", n, x, incX)
	l264.Trsv(activeL164, lower, trans, unit, n, a, lda, x, incX)
}

// Dsymv is Ssymv at float64; see Ssymv.
func Dsymv(ul Uplo, n int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	lower := checkUplo("Dsymv", "ul", ul)
	if n < 0 {
		panic("keel: Dsymv: negative dimension")
	}
	checkMatrix64("Dsymv", "a", n, n, a, lda)
	checkVector64("Dsymv", "x", n, x, incX)
	checkVector64("Dsymv", "y", n, y, incY)
	l264.Symv(activeL164, lower, n, alpha, a, lda, x, incX, beta, y, incY)
}

// Dsyr is Ssyr at float64; see Ssyr.
func Dsyr(ul Uplo, n int, alpha float64, x []float64, incX int, a []float64, lda int) {
	lower := checkUplo("Dsyr", "ul", ul)
	if n < 0 {
		panic("keel: Dsyr: negative dimension")
	}
	checkVector64("Dsyr", "x", n, x, incX)
	checkMatrix64("Dsyr", "a", n, n, a, lda)
	l264.Syr(activeL164, lower, n, alpha, x, incX, a, lda)
}

// Dsyr2 is Ssyr2 at float64; see Ssyr2.
func Dsyr2(ul Uplo, n int, alpha float64, x []float64, incX int, y []float64, incY int, a []float64, lda int) {
	lower := checkUplo("Dsyr2", "ul", ul)
	if n < 0 {
		panic("keel: Dsyr2: negative dimension")
	}
	checkVector64("Dsyr2", "x", n, x, incX)
	checkVector64("Dsyr2", "y", n, y, incY)
	checkMatrix64("Dsyr2", "a", n, n, a, lda)
	l264.Syr2(activeL164, lower, n, alpha, x, incX, y, incY, a, lda)
}

// Dgbmv is Sgbmv at float64; see Sgbmv.
func Dgbmv(tA Transpose, m, n, kL, kU int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	trans := checkTranspose("Dgbmv", "tA", tA)
	if m < 0 || n < 0 {
		panic("keel: Dgbmv: negative dimension")
	}
	if kL < 0 || kU < 0 {
		panic("keel: Dgbmv: negative bandwidth")
	}
	if lda < kL+kU+1 {
		panic("keel: Dgbmv: lda < kL+kU+1")
	}
	rowsA := min(m, n+kL)
	if rowsA > 0 && len(a) < lda*(rowsA-1)+kL+kU+1 {
		panic("keel: Dgbmv: a shorter than m, n, kL, kU and lda require")
	}
	rows, inner := m, n
	if trans {
		rows, inner = n, m
	}
	checkVector64("Dgbmv", "x", inner, x, incX)
	checkVector64("Dgbmv", "y", rows, y, incY)
	l264.Gbmv(activeL164, trans, m, n, kL, kU, alpha, a, lda, x, incX, beta, y, incY)
}

// Dtbmv is Stbmv at float64; see Stbmv.
func Dtbmv(ul Uplo, tA Transpose, d Diag, n, k int, a []float64, lda int, x []float64, incX int) {
	lower, trans, unit := checkUplo("Dtbmv", "ul", ul), checkTranspose("Dtbmv", "tA", tA), checkDiag("Dtbmv", "d", d)
	checkN("Dtbmv", n)
	checkBand64("Dtbmv", n, k, a, lda)
	checkVector64("Dtbmv", "x", n, x, incX)
	if n > 0 {
		l264.Tbmv(activeL164, lower, trans, unit, n, k, a, lda, x, incX)
	}
}

// Dtbsv is Stbsv at float64; see Stbsv.
func Dtbsv(ul Uplo, tA Transpose, d Diag, n, k int, a []float64, lda int, x []float64, incX int) {
	lower, trans, unit := checkUplo("Dtbsv", "ul", ul), checkTranspose("Dtbsv", "tA", tA), checkDiag("Dtbsv", "d", d)
	checkN("Dtbsv", n)
	checkBand64("Dtbsv", n, k, a, lda)
	checkVector64("Dtbsv", "x", n, x, incX)
	if n > 0 {
		l264.Tbsv(activeL164, lower, trans, unit, n, k, a, lda, x, incX)
	}
}

// Dsbmv is Ssbmv at float64; see Ssbmv.
func Dsbmv(ul Uplo, n, k int, alpha float64, a []float64, lda int, x []float64, incX int, beta float64, y []float64, incY int) {
	lower := checkUplo("Dsbmv", "ul", ul)
	checkN("Dsbmv", n)
	checkBand64("Dsbmv", n, k, a, lda)
	checkVector64("Dsbmv", "x", n, x, incX)
	checkVector64("Dsbmv", "y", n, y, incY)
	if n > 0 {
		l264.Sbmv(activeL164, lower, n, k, alpha, a, lda, x, incX, beta, y, incY)
	}
}

// Dtpmv is Stpmv at float64; see Stpmv.
func Dtpmv(ul Uplo, tA Transpose, d Diag, n int, ap []float64, x []float64, incX int) {
	lower, trans, unit := checkUplo("Dtpmv", "ul", ul), checkTranspose("Dtpmv", "tA", tA), checkDiag("Dtpmv", "d", d)
	checkN("Dtpmv", n)
	checkPacked64("Dtpmv", n, ap)
	checkVector64("Dtpmv", "x", n, x, incX)
	if n > 0 {
		l264.Tpmv(activeL164, lower, trans, unit, n, ap, x, incX)
	}
}

// Dtpsv is Stpsv at float64; see Stpsv.
func Dtpsv(ul Uplo, tA Transpose, d Diag, n int, ap []float64, x []float64, incX int) {
	lower, trans, unit := checkUplo("Dtpsv", "ul", ul), checkTranspose("Dtpsv", "tA", tA), checkDiag("Dtpsv", "d", d)
	checkN("Dtpsv", n)
	checkPacked64("Dtpsv", n, ap)
	checkVector64("Dtpsv", "x", n, x, incX)
	if n > 0 {
		l264.Tpsv(activeL164, lower, trans, unit, n, ap, x, incX)
	}
}

// Dspmv is Sspmv at float64; see Sspmv.
func Dspmv(ul Uplo, n int, alpha float64, ap []float64, x []float64, incX int, beta float64, y []float64, incY int) {
	lower := checkUplo("Dspmv", "ul", ul)
	checkN("Dspmv", n)
	checkPacked64("Dspmv", n, ap)
	checkVector64("Dspmv", "x", n, x, incX)
	checkVector64("Dspmv", "y", n, y, incY)
	if n > 0 {
		l264.Spmv(activeL164, lower, n, alpha, ap, x, incX, beta, y, incY)
	}
}

// Dspr is Sspr at float64; see Sspr.
func Dspr(ul Uplo, n int, alpha float64, x []float64, incX int, ap []float64) {
	lower := checkUplo("Dspr", "ul", ul)
	checkN("Dspr", n)
	checkVector64("Dspr", "x", n, x, incX)
	checkPacked64("Dspr", n, ap)
	if n > 0 && alpha != 0 {
		l264.Spr(activeL164, lower, n, alpha, x, incX, ap)
	}
}

// Dspr2 is Sspr2 at float64; see Sspr2.
func Dspr2(ul Uplo, n int, alpha float64, x []float64, incX int, y []float64, incY int, ap []float64) {
	lower := checkUplo("Dspr2", "ul", ul)
	checkN("Dspr2", n)
	checkVector64("Dspr2", "x", n, x, incX)
	checkVector64("Dspr2", "y", n, y, incY)
	checkPacked64("Dspr2", n, ap)
	if n > 0 && alpha != 0 {
		l264.Spr2(activeL164, lower, n, alpha, x, incX, y, incY, ap)
	}
}

// checkMatrix64 is checkMatrix for a float64 matrix, message for message.
func checkMatrix64(fn, arg string, rows, cols int, v []float64, ld int) {
	if ld < 1 || ld < cols {
		panic("keel: " + fn + ": ld" + arg + " < max(1, columns of " + arg + ")")
	}
	if rows == 0 || cols == 0 {
		return // empty matrix; v may be nil
	}
	if need := (rows-1)*ld + cols; len(v) < need {
		panic("keel: " + fn + ": " + arg + " shorter than its dimensions and ld" + arg + " require")
	}
}

// checkBand64 is checkBand for a float64 band, message for message.
func checkBand64(fn string, n, k int, a []float64, lda int) {
	if k < 0 {
		panic("keel: " + fn + ": k < 0")
	}
	if lda < k+1 {
		panic("keel: " + fn + ": lda < k+1")
	}
	if n > 0 && len(a) < lda*(n-1)+k+1 {
		panic("keel: " + fn + ": a shorter than n, k and lda require")
	}
}

// checkPacked64 is checkPacked for a float64 packed triangle, message for message.
func checkPacked64(fn string, n int, ap []float64) {
	if len(ap) < n*(n+1)/2 {
		panic("keel: " + fn + ": ap shorter than n(n+1)/2")
	}
}
