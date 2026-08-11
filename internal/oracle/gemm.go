// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// Gemm returns the float64 reference for C = alpha·op(A)·op(B) + beta·C and the
// per-element error scale, both indexed by row-major element (i*n+j) rather than
// by storage offset, so a caller comparing against a padded ldc does not have to
// mirror the padding.
//
// It is the definition, written the obvious way: one dot product per output
// element, in increasing p, accumulated in float64. Nothing is blocked and
// nothing is reassociated, which is the point — keel's summation order is a
// performance artifact and the oracle's is the specification.
//
// # Two conventions it deliberately copies from keel rather than from a textbook
//
// alpha is folded into A, one element at a time: the terms summed are
// (alpha·aᵢₚ)·bₚⱼ, not alpha·(aᵢₚ·bₚⱼ). internal/pack does the same and explains
// why. In float64 the difference is far below what a float32 comparison can see,
// but stating it here keeps the reference honest about what it is a reference for.
//
// The alpha == 0 and beta == 0 shortcuts are semantic, not optimizations.
// alpha == 0 must not read A or B (0·NaN is NaN, and reference SGEMM does not
// multiply), and beta == 0 must not read C (an uninitialized destination is legal
// for beta = 0). An oracle that took the general path would disagree with a
// correct implementation exactly on the inputs those rules exist for.
//
// # The error scale
//
// The bound for a length-k float32 dot product is proportional to Σₚ|xₚyₚ|, not
// to |result|: an output element with heavy cancellation has a tiny value and a
// large bound. So the scale accumulated here is Σₚ|alpha·aᵢₚ·bₚⱼ| + |beta·cᵢⱼ|,
// and the caller pairs it with Tolerance(k+1, scale) — k roundings in the sum
// plus one for the beta term. Returning it from the same loop that computed the
// value is what stops a test from substituting |result| and calling a wrong
// answer correct (see the package doc).
func Gemm(transA, transB bool, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = GemmEntry(transA, transB, i, j, k,
				alpha, a, lda, b, ldb, beta, c, ldc)
		}
	}
	return out, scale
}

// GemmEntry is Gemm for the single output element (i, j), value and scale.
//
// It exists because Gemm's cost is m·n·k and the sweep goes to 2048³: computing
// the whole reference matrix there would cost more than the routine under test by
// a wide margin, in a float64 loop written for obviousness rather than speed. So
// above the size where an exhaustive comparison is affordable, the test verifies a
// seeded random sample of entries and prints the seed — and the entry it verifies
// is computed by exactly the same code as the exhaustive one, so "sampled" changes
// the coverage and not the reference.
func GemmEntry(transA, transB bool, i, j, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) (val, scale float64) {

	al, be := float64(alpha), float64(beta)
	if alpha != 0 {
		for p := 0; p < k; p++ {
			var av, bv float64
			if transA {
				av = float64(a[p*lda+i])
			} else {
				av = float64(a[i*lda+p])
			}
			if transB {
				bv = float64(b[j*ldb+p])
			} else {
				bv = float64(b[p*ldb+j])
			}
			t := (al * av) * bv
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		cv := be * float64(c[i*ldc+j])
		val += cv
		scale += math.Abs(cv)
	}
	return val, scale
}
