// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// The double-double references: the isolating arm of DESIGN.md §5 rule 1 for
// float64 routines. Each accumulates in dd and rounds once at the end, so its
// result is the float64 nearest the exact answer up to an error 2⁻⁵⁴ of what
// Tolerance64 admits; a comparison against one therefore uses Tolerance64 itself,
// not PeerTolerance64. The returned scale is the same Σ|terms| quantity, in the
// same convention, as the float64 reference's.
//
// # Their own error
//
// Each term is formed exactly or nearly so — twoProd for x·y, and mulF (2u²)
// for the (alpha·a)·b of Gemm and Gemv, whose alpha·a is itself an exact twoProd —
// and each add contributes at most 3u²+13u³ of the partial sum, u = 2⁻⁵³. So the
// dd sum is within about (n+1)·5u²·scale of exact, and rounding it to float64 adds
// at most half an ulp of the result. Against Tolerance64(n, scale) =
// 4n·(2u·scale + eta64/2) the first term is about 2⁻⁵³ of the budget and the
// second at most 1/(8n) of it, and the underflow floor is shared: twoProd's lo is rounded
// below 2⁻⁹⁶⁹, by at most eta64/2 per term (dd.go).
//
// # Their size
//
// DESIGN.md requires this arm to reach n ≈ 768: the smallest size crossing a
// KC=384 boundary, so the smallest at which an accumulation-order defect in the
// blocked nest can exist to be caught. The cost is the dd mul-add per term (see
// dd.go), so a full GemmDD at 768³ is a long test and per-entry GemmEntryDD is
// what a sampled sweep uses — the same split, for the same reason, as GemmEntry.
//
// alpha == 0 and beta == 0 are the same semantic shortcuts as in Gemm: neither
// operand is read.

// DotDD returns the double-double inner product, rounded once to float64, and the
// scale Σ|xᵢyᵢ| for its bound.
func DotDD(n int, x []float64, incX int, y []float64, incY int) (val, scale float64) {
	ix, iy := Index(n, incX), Index(n, incY)
	var s dd
	for j := 0; j < n; j++ {
		p := twoProd(x[ix+j*incX], y[iy+j*incY])
		s = add(s, p)
		scale += math.Abs(p.hi)
	}
	return s.hi, scale
}

// GemvDD is the double-double Dgemv: y = alpha·op(A)·x + beta·y, indexed by
// element, every output rounded once.
func GemvDD(trans bool, m, n int, alpha float64, a []float64, lda int,
	x []float64, incX int, beta float64, y []float64, incY int) (out, scale []float64) {

	rows := GemvRows(trans, m, n)
	out = make([]float64, rows)
	scale = make([]float64, rows)
	for i := 0; i < rows; i++ {
		out[i], scale[i] = GemvEntryDD(trans, i, m, n, alpha, a, lda, x, incX, beta, y, incY)
	}
	return out, scale
}

// GemvEntryDD is GemvDD for the single output element i.
func GemvEntryDD(trans bool, i, m, n int, alpha float64, a []float64, lda int,
	x []float64, incX int, beta float64, y []float64, incY int) (val, scale float64) {

	k := GemvInner(trans, m, n)
	var s dd
	if alpha != 0 {
		ix := Index(k, incX)
		for p := 0; p < k; p++ {
			var av float64
			if trans {
				av = a[p*lda+i]
			} else {
				av = a[i*lda+p]
			}
			t := mulF(twoProd(alpha, av), x[ix+p*incX])
			s = add(s, t)
			scale += math.Abs(t.hi)
		}
	}
	if beta != 0 {
		iy := Index(GemvRows(trans, m, n), incY)
		t := twoProd(beta, y[iy+i*incY])
		s = add(s, t)
		scale += math.Abs(t.hi)
	}
	return s.hi, scale
}

// GemmDD is the double-double Dgemm, indexed by row-major element (i*n+j).
func GemmDD(transA, transB bool, m, n, k int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = GemmEntryDD(transA, transB, i, j, k,
				alpha, a, lda, b, ldb, beta, c, ldc)
		}
	}
	return out, scale
}

// GemmEntryDD is GemmDD for the single output element (i, j).
func GemmEntryDD(transA, transB bool, i, j, k int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (val, scale float64) {

	var s dd
	if alpha != 0 {
		for p := 0; p < k; p++ {
			av, bv := gemmOperands(transA, transB, i, j, p, a, lda, b, ldb)
			t := mulF(twoProd(alpha, av), bv)
			s = add(s, t)
			scale += math.Abs(t.hi)
		}
	}
	if beta != 0 {
		t := twoProd(beta, c[i*ldc+j])
		s = add(s, t)
		scale += math.Abs(t.hi)
	}
	return s.hi, scale
}
