// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// The Level-3 references below take the BLAS flags as bools rather than as
// keel.Uplo/Side/Diag values: package oracle is imported *by* the keel tests, so
// depending on keel's types would be a cycle. Each bool is named for the case it
// selects — lower, left, trans, unit — and every caller converts once.

// symAt returns the (i, j) entry of a symmetric matrix stored in one triangle of
// a, reflecting the index when it falls in the half that is not stored.
//
// This is the whole content of "the unreferenced triangle": it is not that the
// other half is ignored, it is that reading it is a bug. A caller may legally hand
// SSYMM a matrix whose unreferenced half is uninitialized, NaN, or somebody else's
// data, so this function reflects rather than reads.
func symAt(lower bool, i, j int, a []float32, lda int) float64 {
	if (lower && i >= j) || (!lower && i <= j) {
		return float64(a[i*lda+j])
	}
	return float64(a[j*lda+i])
}

// InTriangle reports whether element (i, j) is in the triangle a Level-3 routine
// with this uplo references. Exported because the tests need exactly the same
// predicate to poison the other half and to skip it when comparing.
func InTriangle(lower bool, i, j int) bool {
	if lower {
		return i >= j
	}
	return i <= j
}

// Syrk returns the float64 reference for C = alpha·A·Aᵀ + beta·C (trans false, A
// n×k) or C = alpha·Aᵀ·A + beta·C (trans true, A k×n), and the per-element error
// scale. Both are indexed by row-major element (i*n+j).
//
// Entries outside the referenced triangle are left at zero in both slices, and
// they are not computed: SSYRK does not define them, and a caller comparing them
// would be asserting something about memory the routine promises not to touch.
// The test checks that half by a different means — it poisons C's other triangle
// and requires it back bit-identical (scripts/gate-p4.sh criterion 6).
func Syrk(lower, trans bool, n, k int, alpha float32, a []float32, lda int,
	beta float32, c []float32, ldc int) (out, scale []float64) {

	out = make([]float64, n*n)
	scale = make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !InTriangle(lower, i, j) {
				continue
			}
			out[i*n+j], scale[i*n+j] = SyrkEntry(trans, i, j, k, alpha, a, lda, beta, c, ldc)
		}
	}
	return out, scale
}

// SyrkEntry is Syrk for the single element (i, j), value and scale. The caller
// pairs the scale with Tolerance(k+1, scale): k roundings in the sum plus one for
// the beta term, as in GemmEntry.
func SyrkEntry(trans bool, i, j, k int, alpha float32, a []float32, lda int,
	beta float32, c []float32, ldc int) (val, scale float64) {

	al, be := float64(alpha), float64(beta)
	if alpha != 0 {
		for p := 0; p < k; p++ {
			var av, bv float64
			if trans {
				// A is k×n: the (i, p) entry of Aᵀ is a[p][i].
				av, bv = float64(a[p*lda+i]), float64(a[p*lda+j])
			} else {
				// A is n×k: the (p, j) entry of Aᵀ is a[j][p].
				av, bv = float64(a[i*lda+p]), float64(a[j*lda+p])
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

// Symm returns the float64 reference for C = alpha·A·B + beta·C (left) or
// C = alpha·B·A + beta·C (right), where A is symmetric and stored in one triangle,
// plus the per-element error scale. Indexed by row-major element (i*n+j).
//
// A is m×m for the left case and n×n for the right one, and only the triangle
// named by lower is ever read — see symAt.
func Symm(left, lower bool, m, n int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			out[i*n+j], scale[i*n+j] = SymmEntry(left, lower, i, j, m, n,
				alpha, a, lda, b, ldb, beta, c, ldc)
		}
	}
	return out, scale
}

// SymmEntry is Symm for the single element (i, j), value and scale. The reduction
// is over m terms in the left case and n in the right one, so the caller pairs the
// scale with Tolerance(k+1, scale) for that k.
func SymmEntry(left, lower bool, i, j, m, n int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) (val, scale float64) {

	al, be := float64(alpha), float64(beta)
	if alpha != 0 {
		k := n
		if left {
			k = m
		}
		for p := 0; p < k; p++ {
			var av, bv float64
			if left {
				av, bv = symAt(lower, i, p, a, lda), float64(b[p*ldb+j])
			} else {
				av, bv = symAt(lower, p, j, a, lda), float64(b[i*ldb+p])
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

// triAt returns the (i, j) entry of op(A) for a triangular matrix stored in a,
// and reports whether it is on the diagonal.
//
// unit is handled here rather than by the caller because "unit diagonal" in BLAS
// means the stored diagonal is NOT REFERENCED — not that it happens to contain
// ones. A factored matrix legitimately stores L's and U's diagonals in one array,
// so a reference that read a[i][i] and multiplied by it would disagree with a
// correct implementation on exactly the input the convention exists for.
func triAt(trans, unit bool, i, j int, a []float32, lda int) float64 {
	if i == j && unit {
		return 1
	}
	if trans {
		return float64(a[j*lda+i])
	}
	return float64(a[i*lda+j])
}

// Trsm returns the float64 reference solution X of op(A)·X = alpha·B (left) or
// X·op(A) = alpha·B (right), and the per-element error scale. Indexed by
// row-major element (i*n+j).
//
// Unlike every other reference here there is no per-entry form, and that is a
// property of the problem rather than an omission: a triangular solve is a
// dependency chain, so computing one entry of X costs the whole substitution that
// leads to it. Above the size where an exhaustive comparison is affordable the
// test therefore economizes on the number of flag combinations it runs rather than
// on the number of entries it checks — see scripts/gate-p4.sh criterion 2 and the
// sweep's own comment.
//
// # The error scale, and why it is a recursion
//
// For the other routines the scale is Σ|terms|, because each output is an
// independent reduction. Here output (i, j) is computed *from* earlier outputs, so
// its error has two sources: the roundings in its own subtraction, bounded by
// Σₚ|op(A)ᵢₚ·xₚⱼ| + |alpha·bᵢⱼ| as usual, and the error already present in the xₚⱼ
// it subtracts, amplified by |op(A)ᵢₚ| and then by the division. So:
//
//	s(i,j) = ( Σₚ |op(A)ᵢₚ·xₚⱼ| + Σₚ |op(A)ᵢₚ|·s(p,j) + |alpha·bᵢⱼ| ) / |op(A)ᵢᵢ|
//
// with s = 0 for entries not yet solved. That is the standard forward-substitution
// growth factor written as a magnitude rather than as a norm, and it is why a
// badly conditioned triangular matrix legitimately admits a large absolute error:
// the caller pairs it with Tolerance(k+1, scale) for k the dimension of A, so the
// count of roundings stays an upper bound over every row.
//
// The tests keep A diagonally dominant, which keeps this recursion from growing
// the bound to uselessness; a solve that needed a loose bound to pass would be
// telling us about the test matrix rather than about keel.
//
// alpha == 0 returns a zero X without reading A, matching reference STRSM's early
// `B := 0`. Substituting through instead would compute 0/aᵢᵢ, which agrees for a
// well-conditioned A and disagrees exactly where the convention exists — a
// singular or infinite diagonal, on a call the reference answers with zeros.
func Trsm(left, lower, trans, unit bool, m, n int, alpha float32, a []float32, lda int,
	b []float32, ldb int) (out, scale []float64) {

	out = make([]float64, m*n)
	scale = make([]float64, m*n)
	if alpha == 0 {
		return out, scale
	}
	al := float64(alpha)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			v := al * float64(b[i*ldb+j])
			out[i*n+j] = v
			scale[i*n+j] = math.Abs(v)
		}
	}
	// op(A) is lower triangular when exactly one of lower and trans holds: the
	// transpose of a lower triangle is an upper one. Everything below keys off
	// that single derived fact rather than off the two flags separately.
	lowerEff := lower != trans
	if left {
		trsmLeft(lowerEff, trans, unit, m, n, a, lda, out, scale)
	} else {
		trsmRight(lowerEff, trans, unit, m, n, a, lda, out, scale)
	}
	return out, scale
}

// trsmLeft solves op(A)·X = out in place, forward when op(A) is lower triangular
// and backward when it is upper. A is m×m.
func trsmLeft(lowerEff, trans, unit bool, m, n int, a []float32, lda int, out, scale []float64) {
	for step := 0; step < m; step++ {
		i := step
		if !lowerEff {
			i = m - 1 - step
		}
		d := triAt(trans, unit, i, i, a, lda)
		// The already-solved rows: above i when op(A) is lower, below it when it
		// is upper. Stated as bounds rather than as a test inside the loop because
		// at n = 500 the skipped half of the iteration space is the difference
		// between a fast test and a slow one.
		plo, phi := 0, i
		if !lowerEff {
			plo, phi = i+1, m
		}
		for j := 0; j < n; j++ {
			v, s := out[i*n+j], scale[i*n+j]
			for p := plo; p < phi; p++ {
				aip := triAt(trans, unit, i, p, a, lda)
				t := aip * out[p*n+j]
				v -= t
				s += math.Abs(t) + math.Abs(aip)*scale[p*n+j]
			}
			out[i*n+j] = v / d
			scale[i*n+j] = s / math.Abs(d)
		}
	}
}

// trsmRight solves X·op(A) = out in place. A is n×n, and the direction is the
// mirror image of the left case: X's column j depends on the columns before it
// when op(A) is upper, and on the columns after it when op(A) is lower.
func trsmRight(lowerEff, trans, unit bool, m, n int, a []float32, lda int, out, scale []float64) {
	for step := 0; step < n; step++ {
		j := step
		if lowerEff {
			j = n - 1 - step
		}
		d := triAt(trans, unit, j, j, a, lda)
		plo, phi := 0, j
		if lowerEff {
			plo, phi = j+1, n
		}
		for i := 0; i < m; i++ {
			v, s := out[i*n+j], scale[i*n+j]
			for p := plo; p < phi; p++ {
				apj := triAt(trans, unit, p, j, a, lda)
				t := out[i*n+p] * apj
				v -= t
				s += math.Abs(t) + math.Abs(apj)*scale[i*n+p]
			}
			out[i*n+j] = v / d
			scale[i*n+j] = s / math.Abs(d)
		}
	}
}
