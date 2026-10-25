// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// The float64 definition-order references: one per float32 reference, same
// conventions, same (out, scale) return, named for the BLAS routine they are a
// reference for (Ddot, Dgemm, …). The D prefix rather than a suffix is because
// the double-double arm already owns the DD suffix — DotDD beside a DotD would be
// one letter from a different precision.
//
// These are the PEER-precision arm of DESIGN.md §5 rule 1: a float64 reference
// for a float64 routine, so a comparison against one bounds the sum of two errors
// and must use PeerTolerance64, not Tolerance64. They catch a deviation from the
// specified order and a gross defect; keel's own accumulation error, isolated, is
// the double-double arm's job (dd_ref.go).
//
// # Why parallel functions and not one generic core
//
// A core generic over [T float32 | float64] was the first choice and is refused
// on measured evidence, not on taste. The arm64 backend fuses any float add whose
// operand is a multiply into one FMA, across statements, and amd64 does not; only
// an explicit float64(x*y) conversion forbids it. Measured on the dev host
// (go1.27.3, darwin/arm64, 1000 seeded length-64 trials):
//
//   - GemmEntry, as shipped, differs from the same loop with per-operation
//     rounding in 696 of 1000 trials: the float32 references are already fused on
//     arm64 and unfused on amd64. At float32 that is harmless — the difference is
//     2⁻²⁹ of the float32 bound.
//   - The float64 instantiation of a generic dot differs from per-operation
//     rounding in 675 of 1000. At float64 it is not harmless: one rounding per
//     term is the same order as the error the comparison measures, so a fused
//     reference is two different references depending on GOARCH.
//
// So a generic core either ships a float64 reference that means different things
// on the two architectures keel benchmarks on, or carries the barriers and moves
// 696 of 1000 float32 GemmEntry results on arm64 — breaking the requirement that
// every float32 reference stay byte-identical. These functions therefore duplicate
// the float32 bodies with a float64(…) barrier on every product that feeds a sum,
// and are the same per-operation-rounded definition on every GOARCH. The float32
// files are not touched.

// Ddot returns the float64 inner product and the scale Σ|xᵢyᵢ| for its bound.
func Ddot(n int, x []float64, incX int, y []float64, incY int) (val, scale float64) {
	ix, iy := Index(n, incX), Index(n, incY)
	for j := 0; j < n; j++ {
		p := float64(x[ix+j*incX] * y[iy+j*incY])
		val += p
		scale += math.Abs(p)
	}
	return val, scale
}

// Dasum returns Σ|xᵢ| and, as with Asum, that same value as its scale.
func Dasum(n int, x []float64, incX int) (val, scale float64) {
	ix := Index(n, incX)
	for j := 0; j < n; j++ {
		val += math.Abs(x[ix+j*incX])
	}
	return val, val
}

// Dnrm2 returns the Euclidean norm, scaled by a power of two so that it does not
// overflow or underflow where the norm itself does not.
//
// Nrm2 needs no scaling because a squared float32 fits in float64. Nothing wider
// than float64 has a wider exponent range — double-double included — so here the
// reference has to rescue itself, the same way keel's Snrm2 does. It rescales by
// 2^-e with 2^e just above max|xᵢ|, which is exact (a power-of-two scale only
// moves the exponent) except for elements so far below the maximum that they
// become subnormal, whose squares are then below eta64 relative to the sum. So the
// sum of squares is still the definition-order one, on rescaled values.
//
// A NaN anywhere gives NaN and otherwise an infinity gives +Inf, as the unscaled
// sum of squares would.
func Dnrm2(n int, x []float64, incX int) (val, scale float64) {
	ix := Index(n, incX)
	var m float64
	nan := false
	for j := 0; j < n; j++ {
		v := math.Abs(x[ix+j*incX])
		if v != v {
			nan = true
		} else if v > m {
			m = v
		}
	}
	switch {
	case nan:
		return math.NaN(), math.NaN()
	case math.IsInf(m, 1):
		return m, m
	case m == 0:
		return 0, 0
	}
	_, e := math.Frexp(m)
	var sumsq float64
	for j := 0; j < n; j++ {
		v := math.Ldexp(x[ix+j*incX], -e)
		sumsq += float64(v * v)
	}
	val = math.Ldexp(math.Sqrt(sumsq), e)
	return val, val
}

// Daxpy is Axpy for float64: y += alpha*x, indexed by element, with per-element
// scale |alpha·xⱼ| + |yⱼ|.
func Daxpy(n int, alpha float64, x []float64, incX int, y []float64, incY int) (out, scale []float64) {
	ix, iy := Index(n, incX), Index(n, incY)
	out = make([]float64, n)
	scale = make([]float64, n)
	for j := 0; j < n; j++ {
		ax := float64(alpha * x[ix+j*incX])
		yj := y[iy+j*incY]
		out[j] = ax + yj
		scale[j] = math.Abs(ax) + math.Abs(yj)
	}
	return out, scale
}

// Dscal is Scal for float64: x *= alpha, indexed by element.
func Dscal(n int, alpha float64, x []float64, incX int) (out, scale []float64) {
	ix := Index(n, incX)
	out = make([]float64, n)
	scale = make([]float64, n)
	for j := 0; j < n; j++ {
		out[j] = alpha * x[ix+j*incX]
		scale[j] = math.Abs(out[j])
	}
	return out, scale
}

// Idamax is Iamax for float64, with the same sequential `>` chain and therefore
// the same NaN convention (see Iamax).
func Idamax(n int, x []float64, incX int) int {
	if n < 1 {
		return -1
	}
	ix := Index(n, incX)
	best, bi := math.Abs(x[ix]), 0
	for j := 1; j < n; j++ {
		if v := math.Abs(x[ix+j*incX]); v > best {
			best, bi = v, j
		}
	}
	return bi
}

// Dgemv is Gemv for float64, with Gemv's conventions.
func Dgemv(trans bool, m, n int, alpha float64, a []float64, lda int,
	x []float64, incX int, beta float64, y []float64, incY int) (out, scale []float64) {

	rows := GemvRows(trans, m, n)
	out = make([]float64, rows)
	scale = make([]float64, rows)
	for i := 0; i < rows; i++ {
		out[i], scale[i] = DgemvEntry(trans, i, m, n, alpha, a, lda, x, incX, beta, y, incY)
	}
	return out, scale
}

// DgemvEntry is Dgemv for the single output element i.
func DgemvEntry(trans bool, i, m, n int, alpha float64, a []float64, lda int,
	x []float64, incX int, beta float64, y []float64, incY int) (val, scale float64) {

	k := GemvInner(trans, m, n)
	if alpha != 0 {
		ix := Index(k, incX)
		for p := 0; p < k; p++ {
			var av float64
			if trans {
				av = a[p*lda+i]
			} else {
				av = a[i*lda+p]
			}
			t := float64(float64(alpha*av) * x[ix+p*incX])
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		iy := Index(GemvRows(trans, m, n), incY)
		yv := float64(beta * y[iy+i*incY])
		val += yv
		scale += math.Abs(yv)
	}
	return val, scale
}

// Dger is Ger for float64, with Ger's alpha == 0 convention.
func Dger(m, n int, alpha float64, x []float64, incX int, y []float64, incY int,
	a []float64, lda int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = DgerEntry(i, j, m, n, alpha, x, incX, y, incY, a, lda)
		}
	}
	return out, scale
}

// DgerEntry is Dger for the single element (i, j); the caller pairs its scale
// with a reduction length of 2, as for GerEntry.
func DgerEntry(i, j, m, n int, alpha float64, x []float64, incX int, y []float64, incY int,
	a []float64, lda int) (val, scale float64) {

	av := a[i*lda+j]
	val, scale = av, math.Abs(av)
	if alpha == 0 {
		return val, scale
	}
	ix, iy := Index(m, incX), Index(n, incY)
	t := float64(float64(alpha*x[ix+i*incX]) * y[iy+j*incY])
	val += t
	scale += math.Abs(t)
	return val, scale
}

// Dgemm is Gemm for float64, with Gemm's conventions: alpha folded into A one
// element at a time, and the alpha == 0 and beta == 0 shortcuts.
func Dgemm(transA, transB bool, m, n, k int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = DgemmEntry(transA, transB, i, j, k,
				alpha, a, lda, b, ldb, beta, c, ldc)
		}
	}
	return out, scale
}

// DgemmEntry is Dgemm for the single output element (i, j).
func DgemmEntry(transA, transB bool, i, j, k int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (val, scale float64) {

	if alpha != 0 {
		for p := 0; p < k; p++ {
			av, bv := gemmOperands(transA, transB, i, j, p, a, lda, b, ldb)
			t := float64(float64(alpha*av) * bv)
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		cv := float64(beta * c[i*ldc+j])
		val += cv
		scale += math.Abs(cv)
	}
	return val, scale
}

// gemmOperands returns op(A)[i,p] and op(B)[p,j]. Shared by the float64 and the
// double-double Gemm, so the two arms cannot disagree about a transpose.
func gemmOperands(transA, transB bool, i, j, p int, a []float64, lda int,
	b []float64, ldb int) (av, bv float64) {

	if transA {
		av = a[p*lda+i]
	} else {
		av = a[i*lda+p]
	}
	if transB {
		bv = b[j*ldb+p]
	} else {
		bv = b[p*ldb+j]
	}
	return av, bv
}

// dsymAt is symAt for float64: reflect rather than read the unreferenced half.
func dsymAt(lower bool, i, j int, a []float64, lda int) float64 {
	if (lower && i >= j) || (!lower && i <= j) {
		return a[i*lda+j]
	}
	return a[j*lda+i]
}

// Dsyrk is Syrk for float64; entries outside the referenced triangle are left
// zero and not computed.
func Dsyrk(lower, trans bool, n, k int, alpha float64, a []float64, lda int,
	beta float64, c []float64, ldc int) (out, scale []float64) {

	out = make([]float64, n*n)
	scale = make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !InTriangle(lower, i, j) {
				continue
			}
			out[i*n+j], scale[i*n+j] = DsyrkEntry(trans, i, j, k, alpha, a, lda, beta, c, ldc)
		}
	}
	return out, scale
}

// DsyrkEntry is Dsyrk for the single element (i, j).
func DsyrkEntry(trans bool, i, j, k int, alpha float64, a []float64, lda int,
	beta float64, c []float64, ldc int) (val, scale float64) {

	if alpha != 0 {
		for p := 0; p < k; p++ {
			var av, bv float64
			if trans {
				av, bv = a[p*lda+i], a[p*lda+j]
			} else {
				av, bv = a[i*lda+p], a[j*lda+p]
			}
			t := float64(float64(alpha*av) * bv)
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		cv := float64(beta * c[i*ldc+j])
		val += cv
		scale += math.Abs(cv)
	}
	return val, scale
}

// Dsymm is Symm for float64; only the triangle named by lower is read.
func Dsymm(left, lower bool, m, n int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = DsymmEntry(left, lower, i, j, m, n,
				alpha, a, lda, b, ldb, beta, c, ldc)
		}
	}
	return out, scale
}

// DsymmEntry is Dsymm for the single element (i, j).
func DsymmEntry(left, lower bool, i, j, m, n int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (val, scale float64) {

	if alpha != 0 {
		k := n
		if left {
			k = m
		}
		for p := 0; p < k; p++ {
			var av, bv float64
			if left {
				av, bv = dsymAt(lower, i, p, a, lda), b[p*ldb+j]
			} else {
				av, bv = dsymAt(lower, p, j, a, lda), b[i*ldb+p]
			}
			t := float64(float64(alpha*av) * bv)
			val += t
			scale += math.Abs(t)
		}
	}
	if beta != 0 {
		cv := float64(beta * c[i*ldc+j])
		val += cv
		scale += math.Abs(cv)
	}
	return val, scale
}

// dtriAt is triAt for float64: a unit diagonal is not referenced.
func dtriAt(trans, unit bool, i, j int, a []float64, lda int) float64 {
	if i == j && unit {
		return 1
	}
	if trans {
		return a[j*lda+i]
	}
	return a[i*lda+j]
}

// Dtrsm is Trsm for float64, with Trsm's recursive error scale and its
// alpha == 0 convention.
func Dtrsm(left, lower, trans, unit bool, m, n int, alpha float64, a []float64, lda int,
	b []float64, ldb int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	if alpha == 0 {
		return out, scale
	}
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			v := alpha * b[i*ldb+j]
			out[i*n+j] = v
			scale[i*n+j] = math.Abs(v)
		}
	}
	lowerEff := lower != trans
	if left {
		dtrsmLeft(lowerEff, trans, unit, m, n, a, lda, out, scale)
	} else {
		dtrsmRight(lowerEff, trans, unit, m, n, a, lda, out, scale)
	}
	return out, scale
}

// dtrsmLeft is trsmLeft for float64.
func dtrsmLeft(lowerEff, trans, unit bool, m, n int, a []float64, lda int, out, scale []float64) {
	for step := 0; step < m; step++ {
		i := step
		if !lowerEff {
			i = m - 1 - step
		}
		d := dtriAt(trans, unit, i, i, a, lda)
		plo, phi := 0, i
		if !lowerEff {
			plo, phi = i+1, m
		}
		for j := 0; j < n; j++ {
			v, s := out[i*n+j], scale[i*n+j]
			for p := plo; p < phi; p++ {
				aip := dtriAt(trans, unit, i, p, a, lda)
				t := float64(aip * out[p*n+j])
				v -= t
				s += math.Abs(t) + float64(math.Abs(aip)*scale[p*n+j])
			}
			out[i*n+j] = v / d
			scale[i*n+j] = s / math.Abs(d)
		}
	}
}

// dtrsmRight is trsmRight for float64.
func dtrsmRight(lowerEff, trans, unit bool, m, n int, a []float64, lda int, out, scale []float64) {
	for step := 0; step < n; step++ {
		j := step
		if lowerEff {
			j = n - 1 - step
		}
		d := dtriAt(trans, unit, j, j, a, lda)
		plo, phi := 0, j
		if lowerEff {
			plo, phi = j+1, n
		}
		for i := 0; i < m; i++ {
			v, s := out[i*n+j], scale[i*n+j]
			for p := plo; p < phi; p++ {
				apj := dtriAt(trans, unit, p, j, a, lda)
				t := float64(out[i*n+p] * apj)
				v -= t
				s += math.Abs(t) + float64(math.Abs(apj)*scale[i*n+p])
			}
			out[i*n+j] = v / d
			scale[i*n+j] = s / math.Abs(d)
		}
	}
}

// Dsyr2k is Syr2k for float64; the reduction length is 2k.
func Dsyr2k(lower, trans bool, n, k int, alpha float64, a []float64, lda int,
	b []float64, ldb int, beta float64, c []float64, ldc int) (out, scale []float64) {

	out = make([]float64, n*n)
	scale = make([]float64, n*n)
	at := func(m []float64, ld, r, p int) float64 { // op(M)[r, p]
		if trans {
			return m[p*ld+r]
		}
		return m[r*ld+p]
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !InTriangle(lower, i, j) {
				continue
			}
			var val, sc float64
			if alpha != 0 {
				for p := 0; p < k; p++ {
					t1 := float64(float64(alpha*at(a, lda, i, p)) * at(b, ldb, j, p))
					t2 := float64(float64(alpha*at(b, ldb, i, p)) * at(a, lda, j, p))
					val += t1 + t2
					sc += math.Abs(t1) + math.Abs(t2)
				}
			}
			if beta != 0 {
				cv := float64(beta * c[i*ldc+j])
				val += cv
				sc += math.Abs(cv)
			}
			out[i*n+j], scale[i*n+j] = val, sc
		}
	}
	return out, scale
}

// Dtrmm is Trmm for float64: the unreferenced triangle is never read, and with
// unit neither is the diagonal.
func Dtrmm(left, lower, trans, unit bool, m, n int, alpha float64, a []float64, lda int,
	b []float64, ldb int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	if alpha == 0 {
		return out, scale
	}
	opA := func(r, c int) float64 {
		if trans {
			r, c = c, r
		}
		if r == c {
			if unit {
				return 1
			}
			return a[r*lda+c]
		}
		if (lower && c > r) || (!lower && c < r) {
			return 0
		}
		return a[r*lda+c]
	}
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			var val, sc float64
			if left {
				for p := 0; p < m; p++ {
					t := float64(float64(alpha*opA(i, p)) * b[p*ldb+j])
					val += t
					sc += math.Abs(t)
				}
			} else {
				for p := 0; p < n; p++ {
					t := float64(float64(alpha*b[i*ldb+p]) * opA(p, j))
					val += t
					sc += math.Abs(t)
				}
			}
			out[i*n+j], scale[i*n+j] = val, sc
		}
	}
	return out, scale
}
