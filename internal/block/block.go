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
//
// # The ic loop is the parallel axis (DESIGN.md §4/P5)
//
// "Parallelize the MC (ic) loop over a bounded worker pool sized by
// runtime.GOMAXPROCS(0); shared packed-B panel per NC iteration, per-worker
// packed-A buffers from a sync.Pool." That is what the ic loop below does, and
// the reason the instruction picks that loop is that it is the one whose
// iterations write disjoint memory: block ic writes rows [ic, ic+mc) of C and
// nothing else, so the partition needs no lock, no reduction and no ordering.
//
// Three consequences, each of which the gate checks rather than takes on trust:
//
//   - The result is BIT-IDENTICAL to the serial nest at every thread count.
//     Splitting ic does not reassociate any single output element's sum: the
//     depth (pc) loop stays serial, the microkernel's k-loop is untouched, and
//     each element still accumulates over the same pc blocks in the same order.
//     So the parallel path is checked against equality, not against a tolerance,
//     and a future implementation that parallelized the K loop would break that
//     equality loudly instead of quietly widening an epsilon.
//   - Nothing outlives the call. internal/par starts its goroutines per parallel
//     region and joins them before returning; the only thing that persists is the
//     sync.Pool of packed-A buffers, which holds memory rather than goroutines.
//   - At GOMAXPROCS=1 there is no goroutine, no atomic and no scheduling in the
//     path — par.Run calls the body in the caller. Every measurement this project
//     has published was taken pinned to one thread, so the serial nest those
//     numbers describe is still the code that runs when it is pinned.
//
// The packed B panel is shared by every worker in the region, which is what the
// instruction asks for and also what makes the parallelization cheap: the
// KC×NC panel is built once per (jc, pc) and then read concurrently, so eight
// workers reuse one L3-resident copy rather than streaming eight.
package block

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/pack"
	"github.com/scttfrdmn/keel/internal/par"
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

// plan is the blocking parameters clamped to one problem's shape: the vars above
// rounded to whole tiles by Params, then cut down to m/n/k where the problem is
// smaller than a block.
//
// A function rather than four lines inside gemm because the retention
// decomposition (nest_bench_test.go, issue #26) has to walk the *same* blocks the
// real nest walks — a decomposition whose parts were measured over a different
// block structure than the whole would not be a decomposition of anything.
func plan(kn kern.Kernel, m, n, k int) (kc, mc, nc int) {
	kc, mc, nc = Params(kn)
	if kc > k {
		kc = k
	}
	if mc > m {
		mc = wholeTiles(m+kn.MR-1, kn.MR) // whole tiles covering m, not m rounded down
	}
	if nc > n {
		nc = wholeTiles(n+kn.NR-1, kn.NR)
	}
	return kc, mc, nc
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

// split says whether one nest call may hand its ic loop to the worker pool.
//
// It is a parameter rather than a global because the one caller that must decline
// is Trsm: it has already split its problem across the pool one level up (over
// the right-hand sides, which are independent where its row blocks are not — see
// tri.go), so a rank update that split again would oversubscribe the machine by a
// factor of GOMAXPROCS and produce a bounded pool in name only.
type split bool

const (
	splitIC split = true  // partition the ic loop over the pool
	noSplit split = false // run in the calling goroutine, whatever GOMAXPROCS says
)

// apBuf is one worker's packed-A panel. A struct rather than a bare []float32
// because sync.Pool stores an interface value, and putting a slice in one
// allocates the header every time — the thing the pool exists to avoid.
type apBuf struct{ v []float32 }

// apPool holds packed-A panels between parallel regions. This is the sync.Pool
// DESIGN.md §4/P5 names, and it is the only thing in this package that survives a
// call.
//
// It holds no result and is not read before it is written: pack.APanels fills
// every panel the macrokernel will then read (including the zero padding of a
// ragged last panel — see internal/pack), so a buffer arrives carrying nothing
// that can reach C. scripts/gate-p5.sh checks that claim from the outside by
// requiring a second identical call to be bit-identical to the first, which is
// exactly the assertion a pooled buffer could falsify.
var apPool sync.Pool

// apGet returns a packed-A buffer of exactly n floats, reusing a pooled one when
// it is large enough. A buffer too small for this call is dropped rather than
// grown: the sizes come from the blocking parameters, so the pool converges on
// one size after the first call and a mixed workload pays an allocation instead
// of every caller paying an indirection.
func apGet(n int) *apBuf {
	if b, ok := apPool.Get().(*apBuf); ok && cap(b.v) >= n {
		b.v = b.v[:n]
		return b
	}
	return &apBuf{v: make([]float32, n)}
}

func apPut(b *apBuf) { apPool.Put(b) }

// workersLast records how many workers the most recently completed Level-3 call
// distributed work to. See WorkersLastCall for what it is for and what it is not.
var workersLast atomic.Int64

// beginCall resets the worker accounting for one public Level-3 call.
func beginCall() { workersLast.Store(0) }

// recordWorkers keeps the widest parallel region of the current call. Widest
// rather than last, because a call can have regions of different widths — Symm's
// expansion pass and the nest it feeds — and the honest answer to "how many
// workers did this call use" is the most it ever had running at once.
func recordWorkers(w int) {
	for {
		cur := workersLast.Load()
		if int64(w) <= cur {
			return
		}
		if workersLast.CompareAndSwap(cur, int64(w)) {
			return
		}
	}
}

// WorkersLastCall reports how many worker goroutines the most recently completed
// Level-3 call distributed work to, counting the calling goroutine as a worker.
//
// # Why this exists, and why it is not "state between calls"
//
// scripts/gate-p5.sh criterion 3 requires a benchmark row named threads=8 to
// declare the workers it actually ran on, because a row that silently ran on one
// worker yields a 1.0× ratio and reads as a performance problem rather than as
// the measurement failure it is. Only the library can answer that, so the library
// answers it.
//
// It is instrumentation, and the distinction from state matters enough to state
// precisely. No computation reads it; no result depends on it; every public
// Level-3 entry point overwrites it before doing any work, so nothing it holds
// can reach a later call. The "no state between calls" criterion is about parked
// goroutines and about buffers whose contents survive into a later result, and a
// counter that only the harness reads is neither.
//
// Its one real limitation, stated rather than papered over: with two Level-3
// calls in flight in the same process the value belongs to whichever wrote last,
// so it is meaningless under concurrent callers. The benchmark harness calls one
// at a time, which is the only context that reads it.
func WorkersLastCall() int { return int(workersLast.Load()) }

// Gemm computes C = alpha·op(A)·op(B) + beta·C for row-major matrices, using kn
// as the microkernel. transA/transB say whether a/b hold the transpose.
//
// It assumes its arguments have already been validated — keel.Sgemm does that,
// and it is the only caller outside tests. The one thing checked here is the
// microkernel's own shape bound, because a kernel too wide for the scratch tile
// would corrupt memory rather than return a wrong answer.
func Gemm(kn kern.Kernel, transA, transB bool, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	beginCall()
	gemm(kn, transA, transB, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc, triMask{}, symOperand{}, splitIC)
}

// gemm is Gemm with a triangular mask on the C update: every write to C, beta
// included, is confined to tri's triangle, and a tile lying entirely outside it
// is not computed at all. tri.on false is plain GEMM and costs nothing — the mask
// predicates fold to constants, the whole-tile test is the one that was already
// there, and no extra work appears in the loop nest. See tri.go for who uses it.
//
// sym says one of the two operands stores only a triangle of a symmetric matrix,
// which the pack reflects; see symOperand. sp says whether the ic loop may be
// distributed over the worker pool; see split.
func gemm(kn kern.Kernel, transA, transB bool, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int, tri triMask, sym symOperand, sp split) {

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
	kc, mc, nc := plan(kn, m, n, k)

	alen := pack.ALen(mr, mc, kc)
	bp := make([]float32, pack.BLen(nr, nc, kc))
	nic := (m + mc - 1) / mc

	for jc := 0; jc < n; jc += nc {
		jn := min(nc, n-jc)
		// No row of this column block is in the triangle, so neither B's panels
		// nor anything downstream of them is worth building.
		if tri.none(0, jc, m, jn) {
			continue
		}
		for pc := 0; pc < k; pc += kc {
			kk := min(kc, k-pc)
			// Built once per (jc, pc) and then read by every worker below — and
			// built BY every worker, because this pack is an Amdahl term rather
			// than the rounding error its size suggests. See packB.
			packB(bp, nr, b, ldb, transB, sym.b, sym.lower, pc, kk, jc, jn, sp)

			body := func(claim func() int) {
				// Per worker, not per call: this is the "per-worker packed-A
				// buffers from a sync.Pool" half of the design instruction. The
				// scratch tile is per worker for the same reason — two workers
				// accumulating fringe tiles into one buffer would corrupt each
				// other's C — and is allocated rather than pooled because it is
				// mr*nr floats against a region of O(mc·nc·kc) flops.
				ab := apGet(alen)
				tile := make([]float32, mr*nr)
				for u := claim(); u >= 0; u = claim() {
					ic := icOrder(u, nic, tri) * mc
					im := min(mc, m-ic)
					if tri.none(ic, jc, im, jn) {
						continue
					}
					if sym.a {
						pack.ASymPanels(ab.v, mr, alpha, a, lda, sym.lower, ic, im, pc, kk)
					} else {
						pack.APanels(ab.v, mr, alpha, a, lda, transA, ic, im, pc, kk)
					}
					cb := c[ic*ldc+jc:]
					if pc == 0 {
						// First depth block for this C block: apply beta before
						// anything accumulates into it.
						scaleTri(beta, ic, jc, im, jn, cb, ldc, tri)
					}
					macro(kn, ab.v, bp, ic, jc, im, jn, kk, cb, ldc, tile, tri)
				}
				apPut(ab)
			}
			if sp == noSplit {
				// Every ic block, in this goroutine, and no accounting: Trsm owns
				// the worker count for its own calls.
				//
				// par.Serial and not par.Run(1, ...): Run's argument is the unit
				// count, so asking for one unit runs one ic block and silently
				// drops the rest. See par.Serial for the measured consequence.
				par.Serial(nic, body)
				continue
			}
			recordWorkers(par.Run(nic, body))
		}
	}
}

// packB fills the shared packed-B panel for one (jc, pc) block, over the same pool
// the ic loop below it uses.
//
// # Why this is parallel when its cost looks negligible
//
// It is O(kc·nc) against the region's O(m·nc·kc), which is the argument for leaving
// it serial and it is wrong, because this pack sits BETWEEN two parallel regions with
// every worker idle. Amdahl does not care what fraction of the *work* it is, only
// what fraction of the *time*. At n=k=4096 with NC=4096 and KC=384 the serial version
// was eleven single-threaded copies of 1.57M floats — about 69 MB — inside a call
// whose parallel part takes ~90 ms at eight threads.
//
// The first gate-p5 run on the parallel nest measured what that costs (#65,
// build/gate-p5-175098d.log): Sgemm, Ssyrk and Ssymm missed the >=6.0x floor on all
// three hosts while Strsm cleared it on all three, and the difference between them is
// exactly this — Trsm splits its right-hand sides at the top, so its parallel region
// *encloses* its packing instead of being enclosed by it. That 6.0x floor is retired
// (#6, 2026-08-20) and the two classes were never judged by the same bar anyway, so
// the CONTRAST above is evidence only as far as its own log; the mechanism is not,
// because "enclosing versus enclosed" is a property of this source that a reader can
// check here rather than a verdict borrowed from a criterion. The routine ordering within
// the miss follows too: Ssyrk pays this same pack for half the flops, because the mask
// discards the tiles above the diagonal after the full kc x nc panel has been packed,
// and Ssyrk was last on every host.
//
// # Why it is safe, and why it is still bit-identical
//
// The panels are a partition of bp, so the workers write disjoint ranges: no lock, no
// ordering, no reduction. And packing copies and scales rather than accumulating, so
// the packed panel is the same bits whatever order the ranges ran in — the same
// property that makes the ic split exact, one level down. pack.BPanelsPart takes the
// whole buffer and checks it against the whole block's BLen so that a partition
// off-by-one panics instead of leaving a panel holding whatever the pooled buffer
// held last.
//
// The worker count is recorded, because this region is as real as the ic one and
// WorkersLastCall promises the widest region of the call.
func packB(bp []float32, nr int, b []float32, ldb int, transB, sym, lower bool, pc, kk, jc, jn int, sp split) {
	npan := pack.NPanels(nr, jn)
	body := func(claim func() int) {
		for u := claim(); u >= 0; u = claim() {
			if sym {
				pack.BSymPanelsPart(bp, nr, b, ldb, lower, pc, kk, jc, jn, u, u+1)
				continue
			}
			pack.BPanelsPart(bp, nr, b, ldb, transB, pc, kk, jc, jn, u, u+1)
		}
	}
	if sp == noSplit {
		// Trsm's rank updates: serial for the same reason their ic loop is, and
		// par.Serial rather than par.Run(1, ...) for the same reason too.
		par.Serial(npan, body)
		return
	}
	recordWorkers(par.Run(npan, body))
}

// icOrder maps a claim index to an ic block index, heaviest block first.
//
// The order is irrelevant to the result — the blocks write disjoint rows of C —
// and it decides the makespan. Dynamic claiming finishes at (ideal + the last
// unit claimed), so the tail wants to be the cheap end of the work. With a lower
// triangular mask, block ic keeps only the columns up to its own rows, so work
// grows with ic and the last block is the largest one there is: claiming in index
// order would leave a worker starting the biggest block when every other worker
// is nearly done. Reversed, the same partition ends on the smallest.
//
// An upper mask is the mirror image and already has its heavy end first, and
// unmasked blocks are equal, so both take the index unchanged. This is
// longest-processing-time-first with the sort replaced by the one fact the mask
// already tells us.
func icOrder(u, nic int, tri triMask) int {
	if tri.on && tri.lower {
		return nic - 1 - u
	}
	return u
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
			// The add-back stays a scalar loop *inlined here*, which is issue #22's
			// answer and not an omission. Candidate C vectorized it
			// (internal/vec/edge_amd64.go, still in the tree and still tested) and
			// was measured to lose: geomean sec/op −0.87% / +0.34% / +1.52% on
			// Zen 4 / Skylake-X / Zen 5, its losses concentrated on the thin
			// shapes — most fringe tiles per unit of arithmetic, hence most
			// add-back *calls* (build/edge-fba229f.log).
			//
			// So the loop body is not what this site must protect; the call is.
			// Dispatching through a kern.Kernel function field would hand this
			// scalar loop exactly the per-row indirect call that sank C, which is
			// why the fields were removed rather than repointed at the scalar
			// twins: keeping A means keeping A as measured. A future C′ that
			// monomorphizes per backend has to re-establish the property those
			// fields existed for — KEEL_FORCE=scalar must force the add-back too,
			// or a forced run stops describing what ran.
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
