// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import "github.com/scttfrdmn/keel/internal/kern"

// Syr2k computes C = alpha·(A·Bᵀ + B·Aᵀ) + beta·C (trans false, A and B n×k) or
// C = alpha·(Aᵀ·B + Bᵀ·A) + beta·C (trans true, A and B k×n), writing only the
// triangle of C named by lower. It assumes validated arguments; keel.Ssyr2k does
// that.
//
// It is Syrk's nest run twice with the same mask: the first pass applies beta and
// the second accumulates with beta = 1. Both passes go through the audited
// microkernel, so Syr2k inherits Sgemm's kernel and blocking exactly as Syrk does.
// Two passes round the two products separately, which the tolerance model covers
// as one 2k-term reduction.
//
// beta == 0 does not read C, because the first pass is gemm with beta 0. alpha == 0
// leaves beta·C without reading A or B, because gemm does the same with alpha 0
// and the second pass is then alpha 0, beta 1: a no-op.
func Syr2k(kn kern.Kernel, lower, trans bool, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	beginCall()
	mask := triMask{on: true, lower: lower}
	gemm(kn, trans, !trans, n, n, k, alpha, a, lda, b, ldb, beta, c, ldc, mask, symOperand{}, splitIC)
	gemm(kn, trans, !trans, n, n, k, alpha, b, ldb, a, lda, 1, c, ldc, mask, symOperand{}, splitIC)
}

// Trmm computes B = alpha·op(A)·B (left, A m×m) or B = alpha·B·op(A) (right, A
// n×n) in place, for A triangular in the triangle named by lower. unit means A's
// diagonal is taken to be 1 and never read. It assumes validated arguments;
// keel.Strmm does that.
//
// # The recipe
//
// It is Trsm's partition run forward. B is split into MB-row blocks (left) or
// MB-column blocks (right), and each block of the result is
//
//	B_i := alpha·op(A)_ii·B_i + alpha·op(A)[i, rest]·B[rest]
//
// where rest is the blocks op(A) couples it to. Overwriting in place needs those to
// be the ORIGINAL B[rest], so the blocks run in the order that consumes each one
// before it is overwritten: bottom-up when op(A) is lower (row i reads rows above
// it), top-down when upper; the mirror of that for the right side.
//
// The off-diagonal term is one gemm. The diagonal block is a gemm too, against a
// dense copy of the triangle with the other half zeroed and, for unit, a diagonal
// of 1 — so A's unreferenced triangle and a unit diagonal are never read, which is
// what lets them hold anything, NaN included. The copy is MB×MB, and the source B_i
// is copied once, because gemm cannot read and write the same block.
//
// alpha == 0 sets B to zero without reading A, matching reference STRMM.
func Trmm(kn kern.Kernel, left, lower, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	if m == 0 || n == 0 {
		return
	}
	beginCall()
	if alpha == 0 {
		for i := 0; i < m; i++ {
			row := b[i*ldb : i*ldb+n]
			for j := range row {
				row[j] = 0
			}
		}
		return
	}
	// The triangle op(A) occupies: transposing a lower triangle makes it upper.
	lowerEff := lower != trans
	if left {
		trmmLeft(kn, lower, lowerEff, trans, unit, m, n, alpha, a, lda, b, ldb)
	} else {
		trmmRight(kn, lower, lowerEff, trans, unit, m, n, alpha, a, lda, b, ldb)
	}
}

// diagBlock copies A's d×d diagonal block at (o, o) into a dense d×d square: the
// stored triangle named by lower, zeros elsewhere, and 1 on the diagonal for unit.
// Nothing outside the stored triangle, and no diagonal element under unit, is read.
func diagBlock(lower, unit bool, d int, a []float32, lda, o int) []float32 {
	t := make([]float32, d*d)
	for r := 0; r < d; r++ {
		row := a[(o+r)*lda+o:]
		lo, hi := r+1, d // strictly upper part of row r
		if lower {
			lo, hi = 0, r
		}
		for c := lo; c < hi; c++ {
			t[r*d+c] = row[c]
		}
		if unit {
			t[r*d+r] = 1
		} else {
			t[r*d+r] = row[r]
		}
	}
	return t
}

func trmmLeft(kn kern.Kernel, lower, lowerEff, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	nb := (m + MB - 1) / MB
	tmp := make([]float32, min(MB, m)*n)
	for step := 0; step < nb; step++ {
		// lower op(A): row block i reads rows above it, so go bottom-up.
		bi := nb - 1 - step
		if !lowerEff {
			bi = step
		}
		i0 := bi * MB
		mb := min(MB, m-i0)

		// Diagonal block, from a copy of B_i into B_i.
		for r := 0; r < mb; r++ {
			copy(tmp[r*n:r*n+n], b[(i0+r)*ldb:(i0+r)*ldb+n])
		}
		d := diagBlock(lower, unit, mb, a, lda, i0)
		gemm(kn, trans, false, mb, n, mb, alpha, d, mb, tmp, n, 0, b[i0*ldb:], ldb,
			triMask{}, symOperand{}, splitIC)

		// Off-diagonal: the rows of B op(A) couples this block to, still original.
		p0, kk := 0, i0
		if !lowerEff {
			p0, kk = i0+mb, m-(i0+mb)
		}
		if kk == 0 {
			continue
		}
		// op(A)[i0+r, p0+p] is a[i0+r][p0+p], or a[p0+p][i0+r] when transposed;
		// the same origin shift Trsm makes, with the flag doing the transpose.
		asl := a[i0*lda+p0:]
		if trans {
			asl = a[p0*lda+i0:]
		}
		gemm(kn, trans, false, mb, n, kk, alpha, asl, lda, b[p0*ldb:], ldb, 1, b[i0*ldb:], ldb,
			triMask{}, symOperand{}, splitIC)
	}
}

func trmmRight(kn kern.Kernel, lower, lowerEff, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	nblk := (n + MB - 1) / MB
	w := min(MB, n)
	tmp := make([]float32, m*w)
	for step := 0; step < nblk; step++ {
		// Column j of B·op(A) reads columns p with op(A)[p, j] != 0: p >= j when op(A)
		// is lower, p <= j when upper. Lower goes left to right, upper right to left.
		bj := step
		if !lowerEff {
			bj = nblk - 1 - step
		}
		j0 := bj * MB
		nbk := min(MB, n-j0)

		for r := 0; r < m; r++ {
			copy(tmp[r*w:r*w+nbk], b[r*ldb+j0:r*ldb+j0+nbk])
		}
		d := diagBlock(lower, unit, nbk, a, lda, j0)
		gemm(kn, false, trans, m, nbk, nbk, alpha, tmp, w, d, nbk, 0, b[j0:], ldb,
			triMask{}, symOperand{}, splitIC)

		p0, kk := j0+nbk, n-(j0+nbk)
		if !lowerEff {
			p0, kk = 0, j0
		}
		if kk == 0 {
			continue
		}
		// op(A)[p0+p, j0+c] is a[p0+p][j0+c], or a[j0+c][p0+p] when transposed.
		bsl := a[p0*lda+j0:]
		if trans {
			bsl = a[j0*lda+p0:]
		}
		gemm(kn, false, trans, m, nbk, kk, alpha, b[p0:], ldb, bsl, lda, 1, b[j0:], ldb,
			triMask{}, symOperand{}, splitIC)
	}
}
