// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

// Banded and packed storage, in gonum's row-major conventions (not reference
// BLAS's column-major ones). In every one of these formats the stored part of each
// row of A is contiguous, so the algorithms are written once, over a row-segment
// accessor, and each format supplies the accessor:
//
//	triangular/symmetric band, k off-diagonals, leading dimension lda:
//	  upper: row i is a[i*lda : …], holding A[i, i … i+k], diagonal first
//	  lower: row i is a[i*lda : …], holding A[i, i−k … i], diagonal at position k
//	packed, n(n+1)/2 elements:
//	  upper: row i holds A[i, i … n−1], starting at i·n − i(i−1)/2
//	  lower: row i holds A[i, 0 … i],   starting at i(i+1)/2
//
// The work per row is one Dot or Axpy on that segment, through the active Level-1
// backend, as for the dense triangular and symmetric routines (l2sym.go), whose
// conventions these share: negative strides allowed, alpha == 0 does not read the
// vectors, no zero-scale row is skipped, and with Unit the diagonal is not read.

// rowSeg returns the stored part of row i: s[j] is A[i, lo+j]. It always includes
// the diagonal, which is s[0] for an upper and s[len(s)-1] for a lower triangle.
type rowSeg func(i int) (lo int, s []float32)

func bandSeg(lower bool, n, k int, a []float32, lda int) rowSeg {
	if lower {
		return func(i int) (int, []float32) {
			l := max(0, k-i) // positions before l lie left of column 0
			return i - k + l, a[i*lda+l : i*lda+k+1]
		}
	}
	return func(i int) (int, []float32) {
		return i, a[i*lda : i*lda+min(k+1, n-i)]
	}
}

func packedSeg(lower bool, n int, ap []float32) rowSeg {
	if lower {
		return func(i int) (int, []float32) {
			o := i * (i + 1) / 2
			return 0, ap[o : o+i+1]
		}
	}
	return func(i int) (int, []float32) {
		o := i*n - i*(i-1)/2
		return i, ap[o : o+n-i]
	}
}

// strict splits a row segment into the diagonal element and the off-diagonal part,
// returning that part with the column it starts at.
func strict(lower bool, i, lo int, s []float32) (diag float32, slo int, off []float32) {
	if lower {
		return s[len(s)-1], lo, s[:len(s)-1]
	}
	return s[0], i + 1, s[1:]
}

// triMV computes x = op(A)·x for triangular A given by seg.
func triMV(seg rowSeg, lower, trans, unit bool, n int, x []float32, incX int) {
	xs := gather(n, x, incX) // read-only; results go to out
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		lo, s := seg(i)
		d, slo, off := strict(lower, i, lo, s)
		dv := xs[i]
		if !unit {
			dv *= d
		}
		if !trans {
			out[i] += dv + activeL1.Dot(off, xs[slo:slo+len(off)])
		} else {
			out[i] += dv
			activeL1.Axpy(xs[i], off, out[slo:slo+len(off)])
		}
	}
	if incX == 1 {
		copy(x[:n], out)
		return
	}
	scatter(n, out, x, incX)
}

// triSV solves op(A)·x = b for triangular A given by seg, overwriting b. As Strsv:
// it divides rather than multiplying by a reciprocal, and skips no zero multiplier.
func triSV(seg rowSeg, lower, trans, unit bool, n int, x []float32, incX int) {
	xs := gather(n, x, incX)
	forward := lower != trans
	for step := 0; step < n; step++ {
		i := step
		if !forward {
			i = n - 1 - step
		}
		lo, s := seg(i)
		d, slo, off := strict(lower, i, lo, s)
		if !trans {
			xs[i] -= activeL1.Dot(off, xs[slo:slo+len(off)])
			if !unit {
				xs[i] /= d
			}
		} else {
			if !unit {
				xs[i] /= d
			}
			activeL1.Axpy(-xs[i], off, xs[slo:slo+len(off)])
		}
	}
	scatter(n, xs, x, incX)
}

// symMV computes y = alpha·A·x + beta·y for symmetric A given by seg. beta == 0
// does not read y.
func symMV(seg rowSeg, lower bool, n int, alpha float32, x []float32, incX int, beta float32, y []float32, incY int) {
	if alpha == 0 {
		scaleStrided(beta, n, y, incY)
		return
	}
	xs := gather(n, x, incX)
	t := make([]float32, n)
	for i := 0; i < n; i++ {
		lo, s := seg(i)
		_, slo, off := strict(lower, i, lo, s)
		t[i] += activeL1.Dot(s, xs[lo:lo+len(s)])
		activeL1.Axpy(xs[i], off, t[slo:slo+len(off)])
	}
	iy := offset(n, incY)
	for i := 0; i < n; i++ {
		v := alpha * t[i]
		if beta != 0 {
			v += beta * y[iy+i*incY]
		}
		y[iy+i*incY] = v
	}
}

// symR computes A += alpha·x·xᵀ, and symR2 A += alpha·(x·yᵀ + y·xᵀ), on the stored
// triangle given by seg.
func symR(seg rowSeg, n int, alpha float32, x []float32, incX int) {
	xs := gather(n, x, incX)
	for i := 0; i < n; i++ {
		lo, s := seg(i)
		activeL1.Axpy(alpha*xs[i], xs[lo:lo+len(s)], s)
	}
}

func symR2(seg rowSeg, n int, alpha float32, x []float32, incX int, y []float32, incY int) {
	xs, ys := gather(n, x, incX), gather(n, y, incY)
	for i := 0; i < n; i++ {
		lo, s := seg(i)
		activeL1.Axpy(alpha*xs[i], ys[lo:lo+len(s)], s)
		activeL1.Axpy(alpha*ys[i], xs[lo:lo+len(s)], s)
	}
}

// checkBand panics unless a holds an n-row band with k off-diagonals at stride lda.
func checkBand(fn string, n, k int, a []float32, lda int) {
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

// checkPacked panics unless ap holds the n(n+1)/2 elements of a packed triangle.
func checkPacked(fn string, n int, ap []float32) {
	if len(ap) < n*(n+1)/2 {
		panic("keel: " + fn + ": ap shorter than n(n+1)/2")
	}
}

func checkN(fn string, n int) {
	if n < 0 {
		panic("keel: " + fn + ": negative dimension")
	}
}

// Stbmv computes x = op(A)·x for A an n×n triangular band matrix with k
// off-diagonals, stored as described at the top of this file.
func Stbmv(ul Uplo, tA Transpose, d Diag, n, k int, a []float32, lda int, x []float32, incX int) {
	lower, trans, unit := checkUplo("Stbmv", "ul", ul), checkTranspose("Stbmv", "tA", tA), checkDiag("Stbmv", "d", d)
	checkN("Stbmv", n)
	checkBand("Stbmv", n, k, a, lda)
	checkVector("Stbmv", "x", n, x, incX)
	if n > 0 {
		triMV(bandSeg(lower, n, k, a, lda), lower, trans, unit, n, x, incX)
	}
}

// Stbsv solves op(A)·x = b for A an n×n triangular band matrix, overwriting b.
func Stbsv(ul Uplo, tA Transpose, d Diag, n, k int, a []float32, lda int, x []float32, incX int) {
	lower, trans, unit := checkUplo("Stbsv", "ul", ul), checkTranspose("Stbsv", "tA", tA), checkDiag("Stbsv", "d", d)
	checkN("Stbsv", n)
	checkBand("Stbsv", n, k, a, lda)
	checkVector("Stbsv", "x", n, x, incX)
	if n > 0 {
		triSV(bandSeg(lower, n, k, a, lda), lower, trans, unit, n, x, incX)
	}
}

// Ssbmv computes y = alpha·A·x + beta·y for A an n×n symmetric band matrix with k
// off-diagonals, stored in the triangle ul names.
func Ssbmv(ul Uplo, n, k int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	lower := checkUplo("Ssbmv", "ul", ul)
	checkN("Ssbmv", n)
	checkBand("Ssbmv", n, k, a, lda)
	checkVector("Ssbmv", "x", n, x, incX)
	checkVector("Ssbmv", "y", n, y, incY)
	if n > 0 {
		symMV(bandSeg(lower, n, k, a, lda), lower, n, alpha, x, incX, beta, y, incY)
	}
}

// Stpmv computes x = op(A)·x for A an n×n triangular matrix in packed storage.
func Stpmv(ul Uplo, tA Transpose, d Diag, n int, ap []float32, x []float32, incX int) {
	lower, trans, unit := checkUplo("Stpmv", "ul", ul), checkTranspose("Stpmv", "tA", tA), checkDiag("Stpmv", "d", d)
	checkN("Stpmv", n)
	checkPacked("Stpmv", n, ap)
	checkVector("Stpmv", "x", n, x, incX)
	if n > 0 {
		triMV(packedSeg(lower, n, ap), lower, trans, unit, n, x, incX)
	}
}

// Stpsv solves op(A)·x = b for A an n×n triangular matrix in packed storage,
// overwriting b.
func Stpsv(ul Uplo, tA Transpose, d Diag, n int, ap []float32, x []float32, incX int) {
	lower, trans, unit := checkUplo("Stpsv", "ul", ul), checkTranspose("Stpsv", "tA", tA), checkDiag("Stpsv", "d", d)
	checkN("Stpsv", n)
	checkPacked("Stpsv", n, ap)
	checkVector("Stpsv", "x", n, x, incX)
	if n > 0 {
		triSV(packedSeg(lower, n, ap), lower, trans, unit, n, x, incX)
	}
}

// Sspmv computes y = alpha·A·x + beta·y for A an n×n symmetric matrix in packed
// storage of the triangle ul names.
func Sspmv(ul Uplo, n int, alpha float32, ap []float32, x []float32, incX int, beta float32, y []float32, incY int) {
	lower := checkUplo("Sspmv", "ul", ul)
	checkN("Sspmv", n)
	checkPacked("Sspmv", n, ap)
	checkVector("Sspmv", "x", n, x, incX)
	checkVector("Sspmv", "y", n, y, incY)
	if n > 0 {
		symMV(packedSeg(lower, n, ap), lower, n, alpha, x, incX, beta, y, incY)
	}
}

// Sspr computes A += alpha·x·xᵀ for A an n×n symmetric matrix in packed storage.
func Sspr(ul Uplo, n int, alpha float32, x []float32, incX int, ap []float32) {
	lower := checkUplo("Sspr", "ul", ul)
	checkN("Sspr", n)
	checkVector("Sspr", "x", n, x, incX)
	checkPacked("Sspr", n, ap)
	if n > 0 && alpha != 0 {
		symR(packedSeg(lower, n, ap), n, alpha, x, incX)
	}
}

// Sspr2 computes A += alpha·(x·yᵀ + y·xᵀ) for A an n×n symmetric matrix in packed
// storage.
func Sspr2(ul Uplo, n int, alpha float32, x []float32, incX int, y []float32, incY int, ap []float32) {
	lower := checkUplo("Sspr2", "ul", ul)
	checkN("Sspr2", n)
	checkVector("Sspr2", "x", n, x, incX)
	checkVector("Sspr2", "y", n, y, incY)
	checkPacked("Sspr2", n, ap)
	if n > 0 && alpha != 0 {
		symR2(packedSeg(lower, n, ap), n, alpha, x, incX, y, incY)
	}
}

// Sgbmv computes y = alpha·op(A)·x + beta·y for A an m×n band matrix with kL
// sub-diagonals and kU super-diagonals. Row i of the band array, a[i*lda : …],
// holds A[i, i−kL … i+kU], so band position c is column i−kL+c. Only the first
// min(m, n+kL) rows can hold an element; rows past that are zero.
//
// Like Sgemv, an empty reduction (n == 0 for NoTrans, m == 0 for Trans) returns
// y = beta·y rather than leaving y untouched; see Sgemv for why.
func Sgbmv(tA Transpose, m, n, kL, kU int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	trans := checkTranspose("Sgbmv", "tA", tA)
	if m < 0 || n < 0 {
		panic("keel: Sgbmv: negative dimension")
	}
	if kL < 0 || kU < 0 {
		panic("keel: Sgbmv: negative bandwidth")
	}
	if lda < kL+kU+1 {
		panic("keel: Sgbmv: lda < kL+kU+1")
	}
	rowsA := min(m, n+kL)
	if rowsA > 0 && len(a) < lda*(rowsA-1)+kL+kU+1 {
		panic("keel: Sgbmv: a shorter than m, n, kL, kU and lda require")
	}
	rows, inner := m, n
	if trans {
		rows, inner = n, m
	}
	checkVector("Sgbmv", "x", inner, x, incX)
	checkVector("Sgbmv", "y", rows, y, incY)
	if rows == 0 {
		return
	}
	if inner == 0 || alpha == 0 {
		scaleStrided(beta, rows, y, incY)
		return
	}
	xs := gather(inner, x, incX)
	t := make([]float32, rows)
	nCol := kL + kU + 1
	for i := 0; i < rowsA; i++ {
		l := max(0, kL-i)      // first band position inside column 0
		u := min(nCol, n+kL-i) // one past the last inside column n-1
		off := i - kL + l      // the dense column band position l maps to
		seg := a[i*lda+l : i*lda+u]
		if !trans {
			t[i] = activeL1.Dot(seg, xs[off:off+u-l])
		} else {
			activeL1.Axpy(xs[i], seg, t[off:off+u-l])
		}
	}
	iy := offset(rows, incY)
	for i := 0; i < rows; i++ {
		v := alpha * t[i]
		if beta != 0 {
			v += beta * y[iy+i*incY]
		}
		y[iy+i*incY] = v
	}
}

// Sdsdot returns alpha + x·y, with the dot product and the add accumulated in
// float64 and rounded to float32 once at the end, as reference SDSDOT specifies.
// Elements are summed in BLAS element order, so the result is the same on every
// architecture. gonum's differs twice: it rounds the dot product to float32 before
// adding alpha, a second rounding, and for n == 0 it returns 0 rather than alpha.
func Sdsdot(n int, alpha float32, x []float32, incX int, y []float32, incY int) float32 {
	checkVector("Sdsdot", "x", n, x, incX)
	checkVector("Sdsdot", "y", n, y, incY)
	sum := float64(alpha)
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		sum += float64(x[ix+i*incX]) * float64(y[iy+i*incY])
	}
	return float32(sum)
}
