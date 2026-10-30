// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import "github.com/scttfrdmn/keel/internal/block64"

// Level 3 at float64 (#192). Each routine is its S twin at float64: the same checks
// and panic messages, then one call into internal/block64, the blocked nest that
// tools/f64gen generates from internal/block (#190), around the float64 microkernel
// activeKern64 (#191). Dtrsm's diagonal solves go through activeL164's Axpy and
// ScalDiv, as Strsm's go through activeL1's. Thin, so hand-written (the Level-1 rule,
// docs/f64-design.md). TestDParity3 pins each pair to the same behaviour on bad
// arguments, and the S routine's doc comment is the contract.

// Dgemm is Sgemm at float64; see Sgemm.
func Dgemm(tA, tB Transpose, m, n, k int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	ta := checkTranspose("Dgemm", "tA", tA)
	tb := checkTranspose("Dgemm", "tB", tB)
	if m < 0 || n < 0 || k < 0 {
		panic("keel: Dgemm: negative dimension")
	}
	// op(A) is m×k and op(B) is k×n; the stored arrays are those shapes
	// transposed when the flag says so, and it is the stored shape that lda and
	// ldb have to fit.
	ra, ca := m, k
	if ta {
		ra, ca = k, m
	}
	rb, cb := k, n
	if tb {
		rb, cb = n, k
	}
	checkMatrix64("Dgemm", "a", ra, ca, a, lda)
	checkMatrix64("Dgemm", "b", rb, cb, b, ldb)
	checkMatrix64("Dgemm", "c", m, n, c, ldc)
	block64.Gemm(activeKern64, ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}

// Dsyrk is Ssyrk at float64; see Ssyrk.
func Dsyrk(ul Uplo, t Transpose, n, k int, alpha float64, a []float64, lda int, beta float64, c []float64, ldc int) {
	lower := checkUplo("Dsyrk", "ul", ul)
	trans := checkTranspose("Dsyrk", "t", t)
	if n < 0 || k < 0 {
		panic("keel: Dsyrk: negative dimension")
	}
	ra, ca := n, k
	if trans {
		ra, ca = k, n
	}
	checkMatrix64("Dsyrk", "a", ra, ca, a, lda)
	checkMatrix64("Dsyrk", "c", n, n, c, ldc)
	block64.Syrk(activeKern64, lower, trans, n, k, alpha, a, lda, beta, c, ldc)
}

// Dsymm is Ssymm at float64; see Ssymm.
func Dsymm(s Side, ul Uplo, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	left := checkSide("Dsymm", "s", s)
	lower := checkUplo("Dsymm", "ul", ul)
	if m < 0 || n < 0 {
		panic("keel: Dsymm: negative dimension")
	}
	d := n
	if left {
		d = m
	}
	checkMatrix64("Dsymm", "a", d, d, a, lda)
	checkMatrix64("Dsymm", "b", m, n, b, ldb)
	checkMatrix64("Dsymm", "c", m, n, c, ldc)
	block64.Symm(activeKern64, left, lower, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
}

// Dtrsm is Strsm at float64; see Strsm.
func Dtrsm(s Side, ul Uplo, tA Transpose, d Diag, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
	left := checkSide("Dtrsm", "s", s)
	lower := checkUplo("Dtrsm", "ul", ul)
	trans := checkTranspose("Dtrsm", "tA", tA)
	unit := checkDiag("Dtrsm", "d", d)
	if m < 0 || n < 0 {
		panic("keel: Dtrsm: negative dimension")
	}
	da := n
	if left {
		da = m
	}
	checkMatrix64("Dtrsm", "a", da, da, a, lda)
	checkMatrix64("Dtrsm", "b", m, n, b, ldb)
	block64.Trsm(activeKern64, activeL164.Axpy, activeL164.ScalDiv, left, lower, trans, unit, m, n, alpha, a, lda, b, ldb)
}

// Dtrmm is Strmm at float64; see Strmm.
func Dtrmm(s Side, ul Uplo, tA Transpose, d Diag, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
	left := checkSide("Dtrmm", "s", s)
	lower := checkUplo("Dtrmm", "ul", ul)
	trans := checkTranspose("Dtrmm", "tA", tA)
	unit := checkDiag("Dtrmm", "d", d)
	if m < 0 || n < 0 {
		panic("keel: Dtrmm: negative dimension")
	}
	da := n
	if left {
		da = m
	}
	checkMatrix64("Dtrmm", "a", da, da, a, lda)
	checkMatrix64("Dtrmm", "b", m, n, b, ldb)
	block64.Trmm(activeKern64, left, lower, trans, unit, m, n, alpha, a, lda, b, ldb)
}

// Dsyr2k is Ssyr2k at float64; see Ssyr2k.
func Dsyr2k(ul Uplo, t Transpose, n, k int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
	lower := checkUplo("Dsyr2k", "ul", ul)
	trans := checkTranspose("Dsyr2k", "t", t)
	if n < 0 || k < 0 {
		panic("keel: Dsyr2k: negative dimension")
	}
	ra, ca := n, k
	if trans {
		ra, ca = k, n
	}
	checkMatrix64("Dsyr2k", "a", ra, ca, a, lda)
	checkMatrix64("Dsyr2k", "b", ra, ca, b, ldb)
	checkMatrix64("Dsyr2k", "c", n, n, c, ldc)
	block64.Syr2k(activeKern64, lower, trans, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}
