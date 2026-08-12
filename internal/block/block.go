// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package block implements the Goto/BLIS loop nest for SGEMM (DESIGN.md §4/P3):
// NC → KC → MC → NR → MR → microkernel, with the blocking parameters as vars so
// P5 can tune them without a recompile of anything else.
//
// The loop order is the standard one and the reason is standard too: the packed
// B panel (KC×NC) is built once per (jc, pc) and reused by every A panel, so it
// wants to live in L3 for the whole ic loop; the packed A panel (MC×KC) is built
// once per ic and streamed through L2; the microkernel's MR×NR C tile stays in
// registers. Goto & van de Geijn, "Anatomy of High-Performance Matrix
// Multiplication" (TOMS 2008) is the source of the shape, and BLIS is the source
// of the two details that are easy to get wrong: alpha folded into the packed A
// (internal/pack) and beta applied to the C block once, on the first depth
// iteration, rather than inside the microkernel.
//
// # Beta is applied outside the kernel, in three variants
//
// The microkernels compute C += A·B and nothing else — that is the loop P2
// audited, and a variant that multiplied C by beta would have been a different
// loop with a different instruction count. So beta is a separate pass over the
// mc×nc C block, run when pc == 0 (the first depth block, before anything has
// accumulated into it), and selected once per call from three implementations:
// zero writes zeros without reading C, one does nothing at all, general
// multiplies. That is DESIGN.md §4/P3's "beta handling as kernel variants, not
// branches in the loop" — the branch happens once per C block, not once per
// element and never per k.
//
// beta == 0 writes zeros rather than skipping the pass, because C is documented
// as not read in that case: a caller who hands keel an uninitialized (or
// NaN-poisoned) C and beta = 0 must get alpha·A·B, not NaN. Reference SGEMM's
// `IF (BETA.EQ.ZERO)` matches -0 as well, and so does the switch below.
//
// # The edge strategy: zero-padded panels plus a temporary C tile
//
// DESIGN.md §4/P3 leaves this open — "masked loads/stores … or a scalar fringe;
// read the API, then choose" — so here is the reading and the choice.
//
// go1.26.5's archsimd *does* support masking well: there are Mask8x16-style mask
// types, Masked/Merge forms of the arithmetic ops, LoadMaskedFloat32x16 and
// StoreMasked, and — closest of all to this problem —
// LoadFloat32x16SlicePart/StoreSlicePart in slice_gen_amd64.go, which build the
// mask from a short slice's length for you. The ability is not the constraint.
//
// The constraint is what a masked edge would cost *elsewhere*. Masking the C
// update means a second family of microkernels — one per shape, taking a valid
// row and column count — and P2's whole result is a per-shape instruction count
// audited on the object code the hosts run (scripts/gate-p3.sh criterion 4
// re-checks it here for exactly this reason). Doubling the kernel family doubles
// what has to stay zero-spill, and the gate's throughput sentinel exists because
// a fatter K-loop is the risk P3 carries.
//
// Zero-padding pays instead of a fringe kernel: pack fills the ragged last panel
// with zeros (a zero column contributes zero to every accumulator), the
// microkernel runs its full MR×NR shape on every tile, and a fringe tile
// accumulates into an MR×NR scratch buffer whose row stride is NR, after which
// the valid sub-rectangle is added into C by a scalar loop. The audited K-loop is
// byte-identical for interior and edge tiles, and there is exactly one of it.
//
// What that costs is arithmetic on padding — up to (MR-1) rows and (NR-1)
// columns of a tile — plus one copy-back per fringe tile. It is bounded by the
// perimeter, so it is O(1/min(m,n)) of the work asymptotically and only visible
// at the small sizes where DESIGN.md already says a scalar fringe is acceptable.
// The masked variant remains available if a measurement ever asks for it; issue
// #22 carries the numbers on the sizes where padding is proportionally worst.
package block

import (
	"fmt"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/pack"
)

// Blocking parameters, from DESIGN.md §4/P3's Zen4/Ice-Lake-class starting
// point. Vars, not consts, because they are a cache-hierarchy measurement rather
// than a property of the algorithm, and P5 auto-tunes them. Gemm clamps each one
// to the problem and rounds MC/NC to whole tiles, so a value that is not a
// multiple of the shipped MR/NR is wasteful rather than wrong.
var (
	// KC is the depth of one packed block: A panel MC×KC and B panel KC×NC must
	// both stay resident, so this is the parameter L2 pays for.
	KC = 384
	// MC is the rows of A packed at once. A multiple of every shipped MR.
	MC = 144
	// NC is the columns of B packed at once, sized for L3.
	NC = 4096
)

// Params reports the blocking parameters actually in force for kn: the vars
// above, clamped to whole tiles. It exists so the test suite can print them in
// the gate's provenance marker instead of restating the constants and hoping
// they match.
func Params(kn kern.Kernel) (kc, mc, nc int) {
	return KC, wholeTiles(MC, kn.MR), wholeTiles(NC, kn.NR)
}

// wholeTiles rounds v down to a multiple of blk, floored at one tile. A partial
// tile at the end of an interior block would pack a padded panel in the middle
// of the matrix, which is correct but pays for arithmetic on zeros where nothing
// forces it.
func wholeTiles(v, blk int) int {
	if v < blk {
		return blk
	}
	return (v / blk) * blk
}

// Gemm computes C = alpha·op(A)·op(B) + beta·C for row-major matrices, using kn
// as the microkernel. transA/transB say whether a/b hold the transpose.
//
// It assumes its arguments have already been validated — keel.Sgemm does that,
// and it is the only caller outside tests. The one thing checked here is the
// microkernel's own shape bound, because a kernel too wide for the scratch tile
// would corrupt memory rather than return a wrong answer.
func Gemm(kn kern.Kernel, transA, transB bool, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	gemm(kn, transA, transB, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc, triMask{})
}

// gemm is Gemm with a triangular mask on the C update: every write to C, beta
// included, is confined to tri's triangle, and a tile lying entirely outside it
// is not computed at all. tri.on false is plain GEMM and costs nothing — the mask
// predicates fold to constants, the whole-tile test is the one that was already
// there, and no extra work appears in the loop nest. See tri.go for who uses it.
func gemm(kn kern.Kernel, transA, transB bool, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int, tri triMask) {

	if kn.MR < 1 || kn.MR > kern.MaxMR || kn.NR < 1 || kn.NR > kern.MaxNR {
		panic(fmt.Sprintf("block: kernel tile %dx%d outside %dx%d", kn.MR, kn.NR, kern.MaxMR, kern.MaxNR))
	}
	if m == 0 || n == 0 {
		return
	}
	// The empty product. alpha == 0 lands here too, and deliberately does not
	// touch A or B: reference SGEMM does not multiply in that case either, so a
	// NaN or infinity in A cannot reach C through 0·x.
	if k == 0 || alpha == 0 {
		scaleTri(beta, 0, 0, m, n, c, ldc, tri)
		return
	}

	mr, nr := kn.MR, kn.NR
	kc, mc, nc := Params(kn)
	if kc > k {
		kc = k
	}
	if mc > m {
		mc = wholeTiles(m+mr-1, mr) // whole tiles covering m, not m rounded down
	}
	if nc > n {
		nc = wholeTiles(n+nr-1, nr)
	}

	ap := make([]float32, pack.ALen(mr, mc, kc))
	bp := make([]float32, pack.BLen(nr, nc, kc))
	// One scratch tile per call, reused by every fringe tile. Its row stride is
	// nr, so the microkernel's ldc >= NR requirement holds even when the
	// caller's n is smaller than one tile.
	tile := make([]float32, mr*nr)

	for jc := 0; jc < n; jc += nc {
		jn := min(nc, n-jc)
		// No row of this column block is in the triangle, so neither B's panels
		// nor anything downstream of them is worth building.
		if tri.none(0, jc, m, jn) {
			continue
		}
		for pc := 0; pc < k; pc += kc {
			kk := min(kc, k-pc)
			pack.BPanels(bp, nr, b, ldb, transB, pc, kk, jc, jn)
			for ic := 0; ic < m; ic += mc {
				im := min(mc, m-ic)
				if tri.none(ic, jc, im, jn) {
					continue
				}
				pack.APanels(ap, mr, alpha, a, lda, transA, ic, im, pc, kk)
				cb := c[ic*ldc+jc:]
				if pc == 0 {
					// First depth block for this C block: apply beta before
					// anything accumulates into it.
					scaleTri(beta, ic, jc, im, jn, cb, ldc, tri)
				}
				macro(kn, ap, bp, ic, jc, im, jn, kk, cb, ldc, tile, tri)
			}
		}
	}
}

// macro is the two innermost loops: the jr walk over NR-column panels of B and
// the ir walk over MR-row panels of A, calling the microkernel once per tile.
//
// It takes the panels already packed, which is the whole point — the kernel's
// two operands are consecutive runs of memory and reaching tile ib is one
// multiply. kk is this block's depth, and it is also the panel stride, because
// the last depth block is shorter than KC and the panels were packed for kk, not
// for the buffer's capacity.
//
// i0 and j0 are the block's position in the whole C matrix, needed only by the
// mask: c is already offset to (i0, j0), and every index below is local.
func macro(kn kern.Kernel, ap, bp []float32, i0, j0, mc, nc, kk int, c []float32, ldc int, tile []float32, tri triMask) {
	mr, nr := kn.MR, kn.NR
	for jr := 0; jr < nc; jr += nr {
		bpanel := bp[(jr/nr)*nr*kk:][:nr*kk]
		jn := min(nr, nc-jr)
		for ir := 0; ir < mc; ir += mr {
			im := min(mr, mc-ir)
			if tri.none(i0+ir, j0+jr, im, jn) {
				continue
			}
			apanel := ap[(ir/mr)*mr*kk:][:mr*kk]
			ct := c[ir*ldc+jr:]
			if im == mr && jn == nr && tri.whole(i0+ir, j0+jr, im, jn) {
				kn.Fn(kk, apanel, bpanel, ct, ldc)
				continue
			}
			// Fringe or mask-crossing tile: the kernel computes the full MR×NR
			// shape into the scratch buffer (the padding rows and columns of the
			// panels are zero, so those results are zero), and only the part that
			// belongs to C is added back. Writing the padded columns straight into
			// C would clobber the caller's memory past n, or past the end of the
			// slice; writing the masked half would clobber a triangle the routine
			// promises not to touch. One path serves both, which is why the
			// triangular routines need no edge handling of their own.
			clear(tile)
			kn.Fn(kk, apanel, bpanel, tile, nr)
			for i := 0; i < im; i++ {
				lo, hi := tri.rowRange(i0+ir+i, j0+jr, jn)
				dst := ct[i*ldc+lo : i*ldc+hi]
				src := tile[i*nr+lo : i*nr+hi]
				for j, v := range src {
					dst[j] += v
				}
			}
		}
	}
}

// scaleTri applies C = beta·C to the part of an m×n block that lies in tri's
// triangle, as one of three variants chosen once. See the package doc for why
// beta lives here and not in the kernel, and why beta == 0 writes rather than
// skips.
//
// i0 and j0 place the block in the whole matrix, as in macro. With tri.on false
// every row range is the full row and this is the unmasked pass verbatim.
func scaleTri(beta float32, i0, j0, m, n int, c []float32, ldc int, tri triMask) {
	switch beta {
	case 1:
		return
	case 0:
		for i := 0; i < m; i++ {
			lo, hi := tri.rowRange(i0+i, j0, n)
			clear(c[i*ldc+lo : i*ldc+hi])
		}
	default:
		for i := 0; i < m; i++ {
			lo, hi := tri.rowRange(i0+i, j0, n)
			row := c[i*ldc+lo : i*ldc+hi]
			for j := range row {
				row[j] *= beta
			}
		}
	}
}

// BetaVariants is the number of distinct beta implementations, for the gate's
// config marker. Stated as a constant next to the switch it describes so the two
// cannot drift.
const BetaVariants = 3

// EdgeStrategy names the edge handling in the gate's config marker. See the
// package doc for the API reading behind it.
const EdgeStrategy = "zero-padded-panels+temp-tile"
