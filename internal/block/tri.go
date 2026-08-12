// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import "github.com/scttfrdmn/keel/internal/kern"

// The derived Level-3 routines (DESIGN.md §4/P4). Each one is a statement about
// how to reach the GEMM in block.go, not a second implementation of it:
//
//   - Syrk is one gemm call with A on both sides and the C update masked to a
//     triangle.
//   - Symm expands the stored triangle of A into a dense square and is then one
//     unmasked gemm call.
//   - Trsm is a sequence of gemm rank updates against the already-solved part of
//     B, each followed by an unblocked solve against one diagonal block.
//
// This is where the phase's cost/benefit sits. A routine written directly would
// beat the derivation on some shape or other; a routine derived this way inherits
// the packing, the blocking parameters, the beta variants, the edge strategy and —
// the part that took P2 and P3 to establish — the audited K-loop. scripts/gate-p4.sh
// criterion 5 checks the inheritance rather than trusting this comment: every
// derived routine must report the same microkernel and the same kc/mc/nc that
// Sgemm dispatched in the same process.

// triMask confines a C update to one triangle of a square matrix. on false is no
// mask at all, which is how plain Gemm passes through this code unchanged.
//
// The mask is over C's *global* indices, so every predicate takes the position of
// the block or tile in the whole matrix and not its position within its parent
// block. That is the only reason the loop nest in block.go threads ic/jc down to
// macro: a tile cannot tell whether it straddles the diagonal without knowing
// where it is.
type triMask struct {
	on    bool
	lower bool
}

// keeps reports whether element (i, j) of C is written. Unused by the loop nest,
// which works in ranges rather than element-wise; kept because it is the
// definition the three range predicates below are derived from, and the test
// checks them against it.
func (t triMask) keeps(i, j int) bool {
	if !t.on {
		return true
	}
	if t.lower {
		return i >= j
	}
	return i <= j
}

// whole reports whether every element of the m×n rectangle at (i0, j0) is
// written, i.e. whether the microkernel may write straight into C.
//
// For a lower mask that needs the rectangle's largest column index to be at most
// its smallest row index; for an upper mask, the mirror image. An empty rectangle
// is trivially whole, which keeps the caller from having to special-case it.
func (t triMask) whole(i0, j0, m, n int) bool {
	if !t.on || m <= 0 || n <= 0 {
		return true
	}
	if t.lower {
		return i0 >= j0+n-1
	}
	return i0+m-1 <= j0
}

// none reports whether no element of the m×n rectangle at (i0, j0) is written, so
// the caller can skip packing and the kernel entirely. This is where the roughly
// half of a Syrk that is not computed goes away.
func (t triMask) none(i0, j0, m, n int) bool {
	if m <= 0 || n <= 0 {
		return true
	}
	if !t.on {
		return false
	}
	if t.lower {
		return j0 > i0+m-1
	}
	return i0 > j0+n-1
}

// rowRange returns the half-open range of local column indices written in global
// row gi of a block whose columns start at global j0 and run for n.
//
// The range is contiguous, which is the property that makes the mask cheap: a
// masked row is a shorter row, not a row with a per-element test. lo <= hi always
// holds, so a row entirely outside the triangle comes back empty rather than
// inverted.
func (t triMask) rowRange(gi, j0, n int) (lo, hi int) {
	if !t.on {
		return 0, n
	}
	if t.lower {
		// Columns 0..gi are kept, so the block's window ends at gi+1-j0.
		hi = gi + 1 - j0
		if hi > n {
			hi = n
		}
		if hi < 0 {
			hi = 0
		}
		return 0, hi
	}
	// Columns gi..∞ are kept, so the block's window starts at gi-j0.
	lo = gi - j0
	if lo < 0 {
		lo = 0
	}
	if lo > n {
		lo = n
	}
	return lo, n
}

// Syrk computes C = alpha·A·Aᵀ + beta·C (trans false, A n×k) or
// C = alpha·Aᵀ·A + beta·C (trans true, A k×n), writing only the triangle named by
// lower. It assumes validated arguments; keel.Ssyrk does that.
//
// The whole routine is one GEMM with a itself on both sides and the C update
// masked. Passing the same slice twice is safe because both operands are copied
// into packed panels before the kernel runs and neither is written: the aliasing
// is read-only, which is exactly the case pack.APanels/BPanels do not care about.
//
// The masked half is not computed — see triMask.none — but a tile that straddles
// the diagonal is computed in full and half-discarded, and that waste is what
// gate-p4.sh criterion 7's 85%-of-Sgemm bar measures. It is bounded by the
// diagonal, so it is O(MR·NR/n) of the work.
func Syrk(kn kern.Kernel, lower, trans bool, n, k int, alpha float32, a []float32, lda int,
	beta float32, c []float32, ldc int) {

	// op(A) is n×k either way, so the second operand is its transpose: A·Aᵀ takes
	// transB true, Aᵀ·A takes transA true and transB false.
	gemm(kn, trans, !trans, n, n, k, alpha, a, lda, a, lda, beta, c, ldc,
		triMask{on: true, lower: lower})
}

// Symm computes C = alpha·A·B + beta·C (left) or C = alpha·B·A + beta·C (right)
// for A symmetric and stored in the triangle named by lower. A is m×m in the left
// case and n×n in the right one. It assumes validated arguments; keel.Ssymm does
// that.
//
// # Why this expands A instead of teaching pack about symmetry
//
// The stored half of A is not a matrix the packing routines can read: the panel at
// (ic, pc) needs A's entries on both sides of the diagonal, and the ones past it
// live at their reflections. The two ways to handle that are to reflect at pack
// time — one branch per panel, inside internal/pack — or to reflect once into a
// dense square here and hand GEMM an ordinary matrix.
//
// This does the second, and it is a deliberate cost: O(d²) of scratch and one
// extra pass over A, which is asymptotically free against the O(d²·n) of the
// multiply but is not free at the small sizes where d² is comparable to the work.
// The first is strictly better and belongs in pack, where it also covers a future
// Ssymv; it is issue #36, and it is not on P4's critical path because P4's job is
// to establish that the derivation is correct and within 85% of Sgemm, not to
// remove its last copy. The alternative — a symmetric microkernel — is not on the
// table at all: it would double the kernel family P2 audited.
//
// alpha == 0 returns beta·C without allocating or reading A, matching reference
// SSYMM. That is not just a shortcut: A's unreferenced triangle may hold anything,
// and a routine that expanded it to multiply it by zero would fault or produce
// NaN on a legal call.
func Symm(kn kern.Kernel, left, lower bool, m, n int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	if m == 0 || n == 0 {
		return
	}
	if alpha == 0 {
		scaleTri(beta, 0, 0, m, n, c, ldc, triMask{})
		return
	}
	d := n
	if left {
		d = m
	}
	dense := make([]float32, d*d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			// Read the stored half and reflect the other one. Reading a[i*lda+j]
			// unconditionally would be a bug the caller cannot defend against.
			if (lower && i >= j) || (!lower && i <= j) {
				dense[i*d+j] = a[i*lda+j]
			} else {
				dense[i*d+j] = a[j*lda+i]
			}
		}
	}
	if left {
		gemm(kn, false, false, m, n, m, alpha, dense, d, b, ldb, beta, c, ldc, triMask{})
	} else {
		gemm(kn, false, false, m, n, n, alpha, b, ldb, dense, d, beta, c, ldc, triMask{})
	}
}

// MB is the row (left) or column (right) block size of Trsm's outer loop: how
// much of B is solved against one diagonal block of A before the next rank update.
// A var for the same reason KC/MC/NC are — it trades the unblocked solve's scalar
// work against the rank update's blocked work, and that balance is a measurement.
var MB = 64

// Trsm solves op(A)·X = alpha·B (left) or X·op(A) = alpha·B (right) for X,
// overwriting B, where A is triangular and stored in the triangle named by lower.
// A is m×m in the left case and n×n in the right one. unit means A's stored
// diagonal is not referenced and is taken to be 1. It assumes validated
// arguments; keel.Strsm does that.
//
// # The recipe
//
// This is the blocked algorithm from BLIS (Van Zee & van de Geijn, "BLIS: A
// Framework for Rapidly Instantiating BLAS Functionality", TOMS 2015, §4.3; the
// same partitioning appears in Goto & van de Geijn's TOMS 2008 §4 as the trsm
// variant of the GEMM nest). Partition B into MB-row blocks in solve order. Each
// block's own equations involve only one diagonal block of A plus the blocks
// already solved, so:
//
//	B_i := alpha·B_i - op(A)[i, solved]·X[solved]     one gemm, k = solved rows
//	solve op(A)_ii · X_i = B_i                        unblocked, MB×MB triangle
//
// The first block has nothing solved yet, so its update degenerates to B_i·=alpha
// — which is what gemm's k == 0 path already does, so it is not special-cased
// here. All of the flops except the diagonal blocks' O(m·MB·n) go through the
// audited kernel.
//
// The unblocked solves are scalar, and at MB = 64 they are a few percent of the
// work; issue #37 carries the measurement and the option of an L1-backed inner
// loop. What they must not do is get clever: they divide rather than multiply by a
// reciprocal (a reciprocal changes the answer in the last bit and disagrees with
// the oracle for no gain), and they do not skip a zero multiplier (BLAS requires
// 0·Inf to propagate NaN, and a "fast path" for a zero in A would silently return
// a different matrix on an input that is legal and, in a factored matrix, common).
//
// alpha == 0 sets B to zero and returns without reading A at all, matching
// reference STRSM. A is otherwise required to be nonsingular in its referenced
// triangle; keel does not check that, and neither does reference BLAS.
func Trsm(kn kern.Kernel, left, lower, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	if m == 0 || n == 0 {
		return
	}
	if alpha == 0 {
		for i := 0; i < m; i++ {
			clear(b[i*ldb : i*ldb+n])
		}
		return
	}
	// op(A) is lower triangular when exactly one of lower and trans holds, since
	// transposing a lower triangle gives an upper one. Everything below keys off
	// that one derived fact rather than off the two flags separately — the same
	// reduction the oracle makes, arrived at independently there.
	lowerEff := lower != trans
	if left {
		trsmLeft(kn, lowerEff, trans, unit, m, n, alpha, a, lda, b, ldb)
	} else {
		trsmRight(kn, lowerEff, trans, unit, m, n, alpha, a, lda, b, ldb)
	}
}

// trsmLeft solves op(A)·X = alpha·B, forward over row blocks when op(A) is lower
// triangular and backward when it is upper.
func trsmLeft(kn kern.Kernel, lowerEff, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	nb := (m + MB - 1) / MB
	for step := 0; step < nb; step++ {
		bi := step
		if !lowerEff {
			bi = nb - 1 - step
		}
		i0 := bi * MB
		mb := min(MB, m-i0)
		// The rows of X already solved: everything above this block when op(A) is
		// lower, everything below it when op(A) is upper.
		p0, kk := 0, i0
		if !lowerEff {
			p0, kk = i0+mb, m-(i0+mb)
		}
		if kk == 0 {
			// Nothing solved yet, so the update is just the scaling. Not routed
			// through gemm's k == 0 path because p0 can be m here, and indexing a
			// zero-length operand at the end of the matrix is out of range.
			scaleTri(alpha, 0, 0, mb, n, b[i0*ldb:], ldb, triMask{})
		} else {
			// op(A)[i0+r, p0+p] is a[p0+p][i0+r] transposed and a[i0+r][p0+p] not,
			// so the operand's origin moves but its shape is always mb×kk with the
			// transpose flag doing the work.
			asl := a[i0*lda+p0:]
			if trans {
				asl = a[p0*lda+i0:]
			}
			gemm(kn, trans, false, mb, n, kk, -1, asl, lda, b[p0*ldb:], ldb,
				alpha, b[i0*ldb:], ldb, triMask{})
		}
		solveLeft(lowerEff, trans, unit, mb, n, a[i0*lda+i0:], lda, b[i0*ldb:], ldb)
	}
}

// trsmRight solves X·op(A) = alpha·B. The direction is the mirror of the left
// case: column block j of X depends on the blocks before it when op(A) is upper
// and on the blocks after it when op(A) is lower.
func trsmRight(kn kern.Kernel, lowerEff, trans, unit bool, m, n int, alpha float32,
	a []float32, lda int, b []float32, ldb int) {

	nblk := (n + MB - 1) / MB
	for step := 0; step < nblk; step++ {
		bj := step
		if lowerEff {
			bj = nblk - 1 - step
		}
		j0 := bj * MB
		nb := min(MB, n-j0)
		p0, kk := 0, j0
		if lowerEff {
			p0, kk = j0+nb, n-(j0+nb)
		}
		if kk == 0 {
			scaleTri(alpha, 0, 0, m, nb, b[j0:], ldb, triMask{})
		} else {
			// The X operand is a column window of B, so its row stride is ldb; the
			// A operand is op(A)[p0.., j0..], kk×nb.
			bsl := a[p0*lda+j0:]
			if trans {
				bsl = a[j0*lda+p0:]
			}
			gemm(kn, false, trans, m, nb, kk, -1, b[p0:], ldb, bsl, lda,
				alpha, b[j0:], ldb, triMask{})
		}
		solveRight(lowerEff, trans, unit, m, nb, a[j0*lda+j0:], lda, b[j0:], ldb)
	}
}

// solveLeft solves op(A)·X = B in place for one d×d diagonal block, over all n
// columns of B at once. a points at the diagonal block's first element.
//
// The row-at-a-time shape is deliberate: B's rows are contiguous, so the inner
// loop is a unit-stride axpy over n and the substitution's serial dependency is
// paid once per row of the block rather than once per element.
func solveLeft(lowerEff, trans, unit bool, d, n int, a []float32, lda int, b []float32, ldb int) {
	for step := 0; step < d; step++ {
		i := step
		if !lowerEff {
			i = d - 1 - step
		}
		row := b[i*ldb : i*ldb+n]
		plo, phi := 0, i
		if !lowerEff {
			plo, phi = i+1, d
		}
		for p := plo; p < phi; p++ {
			aip := a[i*lda+p]
			if trans {
				aip = a[p*lda+i]
			}
			src := b[p*ldb : p*ldb+n]
			for j, v := range src {
				row[j] -= aip * v
			}
		}
		if !unit {
			// Divide, do not scale by a reciprocal: see Trsm's doc comment. The
			// diagonal is a[i][i] whether or not op transposes.
			dv := a[i*lda+i]
			for j := range row {
				row[j] /= dv
			}
		}
	}
}

// solveRight solves X·op(A) = B in place for one d×d diagonal block, over all m
// rows of B. a points at the diagonal block's first element, b at its column
// window of B.
//
// Here the loops are the other way up — a column of B, strided by ldb — because
// the dependency runs along columns. Nothing in this shape is unit-stride, which
// is one of the reasons #37 exists.
func solveRight(lowerEff, trans, unit bool, m, d int, a []float32, lda int, b []float32, ldb int) {
	for step := 0; step < d; step++ {
		j := step
		if lowerEff {
			j = d - 1 - step
		}
		plo, phi := 0, j
		if lowerEff {
			plo, phi = j+1, d
		}
		for p := plo; p < phi; p++ {
			apj := a[p*lda+j]
			if trans {
				apj = a[j*lda+p]
			}
			for i := 0; i < m; i++ {
				b[i*ldb+j] -= b[i*ldb+p] * apj
			}
		}
		if !unit {
			dv := a[j*lda+j]
			for i := 0; i < m; i++ {
				b[i*ldb+j] /= dv
			}
		}
	}
}
