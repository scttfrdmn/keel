// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// GemvRows reports how many elements y has for a given transpose, so a caller
// sizing the result does not restate the convention: reference SGEMV takes A as
// m×n *as stored* and lets trans decide which dimension y spans.
func GemvRows(trans bool, m, n int) int {
	if trans {
		return n
	}
	return m
}

// GemvInner reports the length of the reduction, i.e. the length of x.
func GemvInner(trans bool, m, n int) int {
	if trans {
		return m
	}
	return n
}

// Gemv returns the float64 reference for y = alpha·op(A)·x + beta·y and the
// per-element error scale, both indexed by element rather than by storage offset,
// so a caller comparing against incY != 1 does not have to mirror the stride.
//
// Same three conventions as Gemm, for the same reasons stated there: alpha is
// folded into A one element at a time, alpha == 0 does not read A or x, and
// beta == 0 does not read y.
func Gemv(trans bool, m, n int, alpha float32, a []float32, lda int,
	x []float32, incX int, beta float32, y []float32, incY int) (out, scale []float64) {

	rows := GemvRows(trans, m, n)
	out = make([]float64, rows)
	scale = make([]float64, rows)
	for i := 0; i < rows; i++ {
		out[i], scale[i] = GemvEntry(trans, i, m, n, alpha, a, lda, x, incX, beta, y, incY)
	}
	return out, scale
}

// GemvEntry is Gemv for the single output element i, value and scale. It exists
// for the same reason GemmEntry does: above the size where an exhaustive
// comparison is affordable the test samples entries, and a sampled entry must be
// computed by the same code as an exhaustive one.
func GemvEntry(trans bool, i, m, n int, alpha float32, a []float32, lda int,
	x []float32, incX int, beta float32, y []float32, incY int) (val, scale float64) {

	k := GemvInner(trans, m, n)
	al, be := float64(alpha), float64(beta)
	if alpha != 0 {
		ix := Index(k, incX)
		for p := 0; p < k; p++ {
			var av float64
			if trans {
				av = float64(a[p*lda+i])
			} else {
				av = float64(a[i*lda+p])
			}
			t := (al * av) * float64(x[ix+p*incX])
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		iy := Index(GemvRows(trans, m, n), incY)
		yv := be * float64(y[iy+i*incY])
		val += yv
		scale += math.Abs(yv)
	}
	return val, scale
}

// Ger returns the float64 reference for A += alpha·x·yᵀ and the per-element error
// scale, indexed by row-major element (i*n+j).
//
// alpha == 0 leaves A alone without reading x or y, matching reference SGER's
// early return: this is the rule that makes `alpha = 0` safe over a vector holding
// an infinity, and an oracle that computed 0·Inf would disagree with a correct
// implementation exactly there.
func Ger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int,
	a []float32, lda int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = GerEntry(i, j, m, n, alpha, x, incX, y, incY, a, lda)
		}
	}
	return out, scale
}

// GerEntry is Ger for the single element (i, j), value and scale.
//
// The rank-1 update is one product and one add, so the scale is |alpha·xᵢ·yⱼ| +
// |aᵢⱼ| and the caller pairs it with Tolerance(2, scale): unlike a dot product
// there is no length to grow the bound with, and using |result| instead would
// call a wrong answer correct whenever the update cancels against A.
func GerEntry(i, j, m, n int, alpha float32, x []float32, incX int, y []float32, incY int,
	a []float32, lda int) (val, scale float64) {

	av := float64(a[i*lda+j])
	val, scale = av, math.Abs(av)
	if alpha == 0 {
		return val, scale
	}
	ix, iy := Index(m, incX), Index(n, incY)
	t := (float64(alpha) * float64(x[ix+i*incX])) * float64(y[iy+j*incY])
	val += t
	scale += math.Abs(t)
	return val, scale
}
