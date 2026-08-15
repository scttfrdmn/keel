// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/par"
)

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
	//
	// The ic loop is the parallel axis here exactly as it is for Sgemm, which is
	// the whole reason gate-p5.sh judges the two at the same floor: they are one
	// parallelism class, and this routine is the same nest with a mask. The mask
	// is what makes the blocks unequal, and icOrder is where that is dealt with.
	beginCall()
	gemm(kn, trans, !trans, n, n, k, alpha, a, lda, a, lda, beta, c, ldc,
		triMask{on: true, lower: lower}, splitIC)
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
	beginCall()
	if alpha == 0 {
		scaleTri(beta, 0, 0, m, n, c, ldc, triMask{})
		return
	}
	d := n
	if left {
		d = m
	}
	dense := make([]float32, d*d)
	expandSym(dense, d, lower, a, lda)
	if left {
		gemm(kn, false, false, m, n, m, alpha, dense, d, b, ldb, beta, c, ldc, triMask{}, splitIC)
	} else {
		gemm(kn, false, false, m, n, n, alpha, b, ldb, dense, d, beta, c, ldc, triMask{}, splitIC)
	}
}

// expandSym writes the full d×d symmetric matrix whose stored triangle is a.
//
// # Why this is parallel and row-wise, when it is O(d²) against the nest's O(d²·n)
//
// Because Amdahl's law does not care that a term is asymptotically free. At the
// shape gate-p5.sh measures — d = n = 4096 — this pass touches 16.8M elements
// while the nest that follows does 137 GFLOP, and the nest is about to be spread
// over eight workers while this pass is not. A serial pass costing 5% of the
// serial nest costs 29% of the parallel one, and caps the ratio the headline
// criterion reads at well under its floor. So the pass is split over the same
// pool: its rows are independent, which is all the partition needs.
//
// Row-wise rather than element-wise for the same measurement-shaped reason. One
// row of the result is a contiguous run of the stored triangle plus a strided
// walk down its reflection, so the stored half is a copy and only the reflected
// half pays for the stride — where the element-wise form paid a branch and a
// strided address on every element of both halves. The values written are
// identical either way: these are plain assignments, so nothing here can quiet a
// signalling NaN that the old loop preserved.
//
// Issue #36 removes this pass altogether by teaching internal/pack to read a
// stored triangle directly. Until it lands, this is the pass that has to not
// dominate.
func expandSym(dense []float32, d int, lower bool, a []float32, lda int) {
	recordWorkers(par.Run(d, func(claim func() int) {
		for i := claim(); i >= 0; i = claim() {
			row := dense[i*d : (i+1)*d]
			if lower {
				// Stored: columns 0..i of row i. Reflected: a[j][i] for j > i.
				copy(row[:i+1], a[i*lda:i*lda+i+1])
				for j := i + 1; j < d; j++ {
					row[j] = a[j*lda+i]
				}
				continue
			}
			// Stored: columns i..d-1 of row i. Reflected: a[j][i] for j < i.
			copy(row[i:], a[i*lda+i:i*lda+d])
			for j := 0; j < i; j++ {
				row[j] = a[j*lda+i]
			}
		}
	}))
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
	beginCall()
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
		strips(n, func(j0, jn int) {
			trsmLeft(kn, lowerEff, trans, unit, m, jn, alpha, a, lda, b[j0:], ldb)
		})
		return
	}
	strips(m, func(i0, im int) {
		trsmRight(kn, lowerEff, trans, unit, im, n, alpha, a, lda, b[i0*ldb:], ldb)
	})
}

// strips splits axis into par.Workers(axis) contiguous non-empty ranges and runs
// fn on one range per worker.
//
// # Why Trsm parallelizes here and not in the nest it calls
//
// This is the second parallelism class scripts/gate-p5.sh names (criterion 1),
// and the reason it is a second class is visible in the shape of its rank update.
// trsmLeft solves MB rows of B at a time, so its gemm is MB×n×k with MB = 64 —
// fewer rows than one MC block. The ic loop the other three routines parallelize
// has exactly one iteration there, so splitting it would yield one worker no
// matter what GOMAXPROCS says.
//
// What Trsm has instead is the axis its block loop cannot use: the right-hand
// sides. Column j of X depends on column j of B and on nothing else, so for the
// left-side solve the whole routine — rank updates, diagonal solves and all —
// partitions over columns of B with no interaction whatsoever. The right-side
// solve is the mirror image over rows. That is a better partition than the ic loop
// would have been even if the shape allowed it: each worker touches a disjoint
// slice of B rather than sharing one.
//
// So the split happens once, at the top, and the rank updates inside run
// noSplit — see the split type. Nesting the two would multiply GOMAXPROCS by
// itself and make "bounded pool" a description of nothing.
//
// The ranges are cut as u·axis/w rather than by a fixed width, which is what
// guarantees every one of them is non-empty when w ≤ axis: a ceil-width partition
// of 10 units into 8 leaves three workers with nothing, and a worker with nothing
// is a workers=5 declaration on a threads=8 row.
func strips(axis int, fn func(off, count int)) {
	w := par.Workers(axis)
	if w < 1 {
		return
	}
	recordWorkers(par.Run(w, func(claim func() int) {
		for u := claim(); u >= 0; u = claim() {
			lo, hi := u*axis/w, (u+1)*axis/w
			fn(lo, hi-lo)
		}
	}))
}

// TrsmWork reports how the useful flops of a Trsm at this shape divide between
// the blocked rank updates, which go through the audited microkernel, and the
// unblocked diagonal solves, which do not.
//
// The two are absolute counts rather than fractions, so that a caller can check
// their sum against the total it believes the shape has before dividing by it. A
// pair of fractions summing to 1 is self-consistent no matter how wrong both are.
//
// This is the parallelism model gate-p5.sh criterion 1 requires beside Strsm's
// measured scaling, and requiring it is the point: Strsm's floor is deferred to a
// measurement PLUS a model, because a scaling number with no account of what
// fraction of the work can scale sets no threshold anybody can defend. The
// diagonal solves are the part with the serial dependency; the rank updates are
// the part that behaves like the class above.
//
// It walks the same MB partition Trsm walks rather than evaluating a closed form,
// so a change to MB or to the blocking moves this with it. The count is of USEFUL
// flops on the same convention as scripts/gate-p5.sh's flops_expect — one
// multiply-add per (row, column) pair of one triangle including its diagonal, per
// right-hand side — so the two must sum to n·m·(m+1) for a left-side solve, and
// p5_test.go asserts exactly that rather than leaving the two counts to agree by
// eye.
//
// Direction does not enter it. The rank-update term is 2·rhs·Σ_{i<j} mb_i·mb_j
// over the block partition, which is symmetric in the block order, so a forward
// solve over a lower triangle and a backward solve over an upper one have the
// same decomposition.
func TrsmWork(left bool, m, n int) (rankUpdate, diagSolve float64) {
	// The blocked axis and the number of right-hand sides swap with the side: a
	// left solve blocks over m and carries n columns, a right solve blocks over n
	// and carries m rows.
	axis, rhs := m, n
	if !left {
		axis, rhs = n, m
	}
	var solved int
	for off := 0; off < axis; off += MB {
		bl := min(MB, axis-off)
		rankUpdate += 2 * float64(bl) * float64(rhs) * float64(solved)
		diagSolve += float64(rhs) * float64(bl) * float64(bl+1)
		solved += bl
	}
	return rankUpdate, diagSolve
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
			// noSplit: this whole solve is already one worker's strip of B (see
			// strips), and the rank update's mb rows are fewer than one MC block
			// anyway, so the ic loop has nothing to hand out.
			gemm(kn, trans, false, mb, n, kk, -1, asl, lda, b[p0*ldb:], ldb,
				alpha, b[i0*ldb:], ldb, triMask{}, noSplit)
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
			// noSplit, for the reason trsmLeft's rank update gives.
			gemm(kn, false, trans, m, nb, kk, -1, b[p0:], ldb, bsl, lda,
				alpha, b[j0:], ldb, triMask{}, noSplit)
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
