// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

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
//     such rows; keel computes them, for the reason gemvTrans gives.
//   - Triangular A: the other triangle is never read, and with Unit neither is the
//     diagonal, which is taken to be 1.

// scatter writes a contiguous buffer back into a strided vector. With inc == 1,
// gather returned v itself, so there is nothing to do.
func scatter(n int, src, v []float32, inc int) {
	if inc == 1 {
		return
	}
	iv := offset(n, inc)
	for j := 0; j < n; j++ {
		v[iv+j*inc] = src[j]
	}
}

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
	if n == 0 {
		return
	}
	xs := gather(n, x, incX) // read-only here: results go to out, written back at the end
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		row := a[i*lda : i*lda+n]
		lo, hi := i+1, n // the strict part of row i that is stored
		if lower {
			lo, hi = 0, i
		}
		diag := xs[i]
		if !unit {
			diag *= row[i]
		}
		if !trans {
			out[i] += diag + activeL1.Dot(row[lo:hi], xs[lo:hi])
		} else {
			// Row i of A is column i of Aᵀ: it scatters x_i into out[lo:hi].
			out[i] += diag
			activeL1.Axpy(xs[i], row[lo:hi], out[lo:hi])
		}
	}
	if incX == 1 {
		copy(x[:n], out)
		return
	}
	scatter(n, out, x, incX)
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
	if n == 0 {
		return
	}
	xs := gather(n, x, incX)
	// op(A) is lower when A is lower and not transposed, or upper and transposed.
	// A lower op(A) is solved top to bottom, an upper one bottom to top.
	forward := lower != trans
	for step := 0; step < n; step++ {
		i := step
		if !forward {
			i = n - 1 - step
		}
		row := a[i*lda : i*lda+n]
		if !trans {
			// Row i of op(A) is row i of A: subtract the solved part, then divide.
			lo, hi := 0, i
			if !lower {
				lo, hi = i+1, n
			}
			xs[i] -= activeL1.Dot(row[lo:hi], xs[lo:hi])
			if !unit {
				xs[i] /= row[i]
			}
		} else {
			// op(A) = Aᵀ: once x_i is final, row i of A is column i of op(A), and it
			// updates every unsolved element at once.
			if !unit {
				xs[i] /= row[i]
			}
			lo, hi := i+1, n
			if lower {
				lo, hi = 0, i
			}
			activeL1.Axpy(-xs[i], row[lo:hi], xs[lo:hi])
		}
	}
	scatter(n, xs, x, incX)
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
	if n == 0 {
		return
	}
	if alpha == 0 {
		scaleStrided(beta, n, y, incY)
		return
	}
	xs := gather(n, x, incX)
	t := make([]float32, n)
	for i := 0; i < n; i++ {
		row := a[i*lda : i*lda+n]
		// Row i holds A[i, j] for the stored j, and by symmetry A[j, i] too: the
		// stored part with the diagonal is a dot into t_i, and the strict part also
		// scatters x_i into t_j.
		lo, hi := i, n
		slo, shi := i+1, n
		if lower {
			lo, hi = 0, i+1
			slo, shi = 0, i
		}
		t[i] += activeL1.Dot(row[lo:hi], xs[lo:hi])
		activeL1.Axpy(xs[i], row[slo:shi], t[slo:shi])
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

// Ssyr computes A += alpha·x·xᵀ on the triangle of the n×n symmetric A that ul
// names; the other triangle is neither read nor written.
func Ssyr(ul Uplo, n int, alpha float32, x []float32, incX int, a []float32, lda int) {
	lower := checkUplo("Ssyr", "ul", ul)
	if n < 0 {
		panic("keel: Ssyr: negative dimension")
	}
	checkVector("Ssyr", "x", n, x, incX)
	checkMatrix("Ssyr", "a", n, n, a, lda)
	if n == 0 || alpha == 0 {
		return
	}
	xs := gather(n, x, incX)
	for i := 0; i < n; i++ {
		lo, hi := i, n
		if lower {
			lo, hi = 0, i+1
		}
		activeL1.Axpy(alpha*xs[i], xs[lo:hi], a[i*lda+lo:i*lda+hi])
	}
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
	if n == 0 || alpha == 0 {
		return
	}
	xs, ys := gather(n, x, incX), gather(n, y, incY)
	for i := 0; i < n; i++ {
		lo, hi := i, n
		if lower {
			lo, hi = 0, i+1
		}
		r := a[i*lda+lo : i*lda+hi]
		activeL1.Axpy(alpha*xs[i], ys[lo:hi], r)
		activeL1.Axpy(alpha*ys[i], xs[lo:hi], r)
	}
}
