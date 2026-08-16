// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

// The AVX-512 fringe add-back: issue #22's candidate C.
//
// # Archived: measured, correct, and slower. Nothing in keel calls this.
//
// #22 is closed and **A ships** — internal/block inlines a scalar add-back, as it
// did before this file existed. This file stays because it is the measured arm and
// because C′ starts from it, and its differential tests keep running so it cannot
// rot into a wrong starting point.
//
// The measurement (build/edge-fba229f.log; three hosts, -count=10, GOMAXPROCS=1,
// A=757acb8 vs C=fba229f): geomean sec/op −0.87% on Zen 4, +0.34% on Skylake-X,
// +1.52% on Zen 5 — one host slightly better, two slightly worse, over per-shape
// swings of −7.92% to +8.19% that agree across hosts on almost nothing. The
// controls held: every interior excursion sits inside its host's between-binary
// layout floor (1.71 / 0.99 / 1.32%, build/layout-ensemble-e829a61.log).
//
// The loss is *located*, which is the useful part. It concentrates on the thin
// shapes — m=5/n=2048 at +5.69% and +8.19%, m=2048/n=33 at +6.72% — i.e. the
// shapes with the most fringe tiles per unit of arithmetic, hence the most add-back
// **calls**. That is the risk the "What this replaces" section below named *before*
// the run, and it is a registered risk confirmed rather than a surprise. What the
// run did **not** establish is that A leaves resolvable edge cost on the table: no
// arm isolated A's fringe cost against a fringe-free baseline, so A may be
// near-optimal on thin shapes for all this shows. Candidate B's trigger as worded
// is therefore *not* met, and B sits one rung behind C′.
//
// C′ — the named follow-up — is this arithmetic with the call removed:
// monomorphize per backend, or hoist the dispatch out of the row loop so it
// amortizes per tile. It must keep two properties this arm had: the per-row live
// window (which is the diagonal coverage B structurally cannot reach) and
// KEEL_FORCE=scalar forcing the add-back too. If C′ still loses to a scalar loop,
// the vector-add-back thesis is dead on its own terms.
//
// # What this replaces and what it does not
//
// A fringe or mask-crossing tile in internal/block is computed at full MR×NR
// into a scratch tile and only its live sub-rectangle is added back into C.
// Candidate A — the incumbent — does that add-back one float at a time.
// Candidate C keeps the kernel and the scratch tile exactly as they are and
// vectorizes only the add-back, which is why it needs no new kernel family and
// costs P2's zero-spill audit nothing. Candidate B, a masked C update inside the
// microkernel, is the one that would have cost both; it is unbuilt pending a
// measured gap here.
//
// Nothing in this file is in a K-loop. DESIGN.md's "no calls in the K-loop"
// does not bind, but the *call count* is still the cost that decides A vs C:
// for a 4×32 tile the whole add-back is at most 8 full-width ops, so an indirect
// call per row could plausibly cost more than the scalar loop it replaced. That
// is the measurement, not an objection — scripts/edge-bench.sh ranks them, and
// this is why the rectangular case takes the row loop *inside* the callee
// (AddTile512, one call per tile) while only the masked case pays per row
// (AddRow512, because its live window differs per row).
//
// # The masked tail is deliberate
//
// The ≤15-element remainder could be a scalar loop, which would need no partial
// op at all and would stay checkptr-clean on go1.26.x. It uses LoadPart512 and
// StorePart512 instead, because #22's ruling says admissibility is satisfied by
// the toolchain keel will *require* and the candidate is to be measured as
// written rather than in a copy-based costume. One consequence to state rather
// than discover: on the hosts' go1.26.5 this puts a partial op on a Level-3 path
// for the first time, so the -race/-d=checkptr fatal of T17 (#42, fixed upstream
// by CL 761120 in go1.27) now reaches Level 3 as well. No gate moves — those
// four criteria are already unmeasured on all three hosts for exactly this
// reason — but the surface is wider, and that is a fact about the tree.
//
// # Numerics
//
// Exact. Every output element is the sum of exactly two inputs, so there is no
// association to choose and no tolerance to argue about: the differential test
// holds these bit-equal to ScalarAddTile and ScalarAddRow.

// AddTile512 adds the im×jn top-left sub-rectangle of tile into c, with rows
// ldc and nr apart respectively.
//
// One call per tile: the row loop is here rather than at the call site so a
// fringe tile costs one indirect call instead of MR of them.
func AddTile512(c []float32, ldc int, tile []float32, nr, im, jn int) {
	for i := 0; i < im; i++ {
		AddRow512(c[i*ldc:i*ldc+jn], tile[i*nr:i*nr+jn])
	}
}

// AddRow512 adds src into dst elementwise, in 16-lane chunks with a masked
// tail. len(dst) must be at least len(src).
//
// AddTile512 calls this per row. internal/block did too, for a mask-crossing tile
// whose live window is its own [lo, hi) per row with no rectangular call
// available — that is the coverage C′ has to keep — but as of #22's close nothing
// outside this package's tests calls either function.
func AddRow512(dst, src []float32) {
	n := len(src)
	j := 0
	// This loop's shape is dictated by four measured properties of go1.26.5 and
	// go1.27rc3, all in docs/toolchain-notes.md T25 (issue #74). None of them is
	// a style preference: the body is one add and three memory ops, and written
	// the obvious way it compiles to 36 instructions per 16 elements instead of
	// 13. Do not simplify any of the four without re-running T25's counts.
	//
	//   - Every slice expression is `[j:j+Lanes]`, never the open-ended `[j:]`.
	//     The open form keeps archsimd's own `CMPQ $16` length check *and* a
	//     five-instruction conditional pointer advance, per operand: 36
	//     instructions per iteration against this loop's 13, for the same load.
	//     This is the largest of the four and the least visible in the source.
	//   - Strict `<` against `len-Lanes+1`, not the natural `j+Lanes <= len`.
	//     T19 recorded the natural form's checks surviving and prescribed the
	//     slice-advancing rewrite; the strict form is a second remedy T19 missed,
	//     and it keeps the indexed loop.
	//   - Both lengths named. `dst = dst[:n]` does *not* substitute for naming
	//     len(dst): a reslice establishes the length equality and then loses it
	//     for the resliced operand, so the surviving check simply moves to
	//     whichever slice was resliced. internal/l1 names both for the same
	//     reason (T19).
	//   - src's limit hoisted, dst's not. `len(src)-Lanes+1` is loop-invariant
	//     and hoisting it saves an LEAQ per iteration with every check still
	//     eliminated; hoisting *both* into one `min` local puts both checks back.
	//     dst's LEAQ is therefore the price of the elimination, and is cheaper
	//     than the check it buys.
	lim := n - Lanes + 1
	for ; j < lim && j < len(dst)-Lanes+1; j += Lanes {
		Store512(dst[j:j+Lanes], Add512(Load512(dst[j:j+Lanes]), Load512(src[j:j+Lanes])))
	}
	if j < n {
		// Both operands are the same short length, so the mask is the same on
		// the load and the store: lanes past n-j are read as zero and not
		// written back.
		StorePart512(dst[j:n], Add512(LoadPart512(dst[j:n]), LoadPart512(src[j:n])))
	}
}
