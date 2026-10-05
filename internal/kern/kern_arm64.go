// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package kern

import "github.com/scttfrdmn/keel/internal/vec"

// vectorKernels returns the NEON candidate shapes the register model fits
// (docs/neon-sweep.md, #136): live = MR·(NR/4) + NR/4 + 1 ≤ 32 V-registers.
//
// InsnsPerFMA was left 0 (unaudited) through #136 while arm64 was characterization
// only, so Preferred tie-broke to the first-listed 8x8 by registry order rather
// than by any measurement. #137's judged ratio (31%/54% of OpenBLAS) surfaced the
// cost: 8x8 is not the faster tile. The negative-control witness on castor (GB10,
// one measured slot, both markers verified) measured the shipped 8x8 at 45.67
// GFLOP/s against a 4x16-only build at 60.44 on the full BenchmarkSgemm/n=2048 —
// 4x16 is +32% and it survives packing and blocking, not just the isolated kernel.
//
// So the counts are now recorded, and they are the spill-audit tool's own — not
// hand-typed (the hazard the old comment named): `spill-audit -goarch arm64` reads
// the steady-state K-loop as 92 insns / 16 FMAs for 8x8 and 80 / 16 for 4x16, the
// same Insns/Arith division the amd64 registry writes. 4x16 is leaner on both axes
// — 5.00 vs 5.75 insns/FMA (fewer broadcasts and reg copies per pass) and 0.5 vs
// 0.625 mem-ops/FMA (which is what MemOpsPerFMA returns only as of #170: it
// divided by the 16-lane Block width until then, so these two figures were right
// here and 4x high in the function that ranks on them — the ordering was not
// affected) — so Preferred ranks it first under ClassFMA and ClassIssue
// alike, and on an FMA-bound host (arm64's default class) the exact MemOpsPerFMA
// decides, so the ranking cannot drift with a recompile even though no arm64 gate
// recomputes these the way the amd64 spill audit does.
func vectorKernels() []Kernel {
	if !vec.HasNEON() {
		return nil
	}
	return []Kernel{
		{Name: NEON, MR: 8, NR: 8, Unroll: 1, Fn: vec.Kernel8x8, InsnsPerFMA: 92.0 / 16},
		{Name: NEON, MR: 4, NR: 16, Unroll: 1, Fn: vec.Kernel4x16, InsnsPerFMA: 80.0 / 16},
		// PROMOTED 2026-10-05 (#136), and the count is the spill-audit tool's own:
		// `spill-audit -goarch arm64` reads Kernel3x24's steady-state loop as 148
		// insns / 36 arith. It is the leanest of the 107 emittable zero-spill NEON
		// shapes (`shapegen -arch arm64 -frontier`), and Preferred therefore selects
		// it over 4x16 under BOTH classes -- 4.111 against 5.000 insns/FMA, and
		// 1/3+4/24 == 1/4+4/16 == 0.5 mem-ops/FMA exactly, so ClassFMA ties on its
		// primary axis and falls through to the same answer (#170).
		//
		// It ships on MEASURED rate and not on that arithmetic, which is #136's own
		// caution 2: rank on the sweep, treat the audit as a filter. The judged
		// evidence, BenchmarkKernel at kc=128 on the two Neoverse parts in us-east-1
		// (archive/arm64-us-east-1-d757ea9): 38.48 against 4x16's 34.91 (+10.2%) and
		// 35.56 against 32.65 (+8.9%), intervals essentially zero-width so both are
		// CI-disjoint by orders of magnitude. Characterization agrees and brackets it:
		// +7.40% on a Cortex-X925 and +20.78% on an A725 (archive/neon-3x24), with the
		// judged parts at the low end as 4-pipe-class FP predicts. It also survives
		// packing and blocking rather than winning only in isolation: +6.62% at full
		// Sgemm/n=2048 under the unmodified registry, at the size most hostile to
		// NR=24, since 4x16 divides 2048 exactly while 3x24 pays both an M- and an
		// N-fringe.
		{Name: NEON, MR: 3, NR: 24, Unroll: 2, Fn: vec.Kernel3x24, InsnsPerFMA: 148.0 / 36},
	}
}

// referenceTiles carries the two shapes the model predicts SPILL (8x16 → 37 live,
// 4x32 → 41 live, both over 32). They are Measured() but not in Kernels(), exactly
// as amd64's ReferenceTile is: benchmarked and -S-audited so the spill prediction
// is graded on the record rather than assumed, and excluded from what ships so
// nothing ever runs a spilling tile. If the audit refutes a prediction — a shape
// the model called spilled that does not — it is promoted to vectorKernels then,
// on the assembly's word, not this file's guess.
// It also carries a candidate whose prediction is the OPPOSITE — 3x24, which the
// model says fits — and that is a second use of this list rather than a stretch
// of its charter. Measured() is what the differential test and the kernel
// benchmark walk, Kernels() is what dispatch ranks, so a shape placed here is
// evidence before it is a product. Both of this list's jobs are the same
// mechanism read in the two directions: a shape is held out of dispatch until
// the assembly and then the sweep have had their say.
func referenceTiles() []Kernel {
	if !vec.HasNEON() {
		return nil
	}
	return []Kernel{
		// 8x12 was PREDICTED to fit (28 live <= 32) but the -S audit found it spills 5
		// accumulators with all 32 V-registers in use (docs/neon-sweep.md step 3): the naive
		// live-set model undercounts the scheduler's transients. Moved here on the assembly's
		// word, so nothing ships a spilling tile.
		{Name: NEON, MR: 8, NR: 12, Unroll: 1, Fn: vec.Kernel8x12},
		{Name: NEON, MR: 8, NR: 16, Unroll: 1, Fn: vec.Kernel8x16},
		{Name: NEON, MR: 4, NR: 32, Unroll: 1, Fn: vec.Kernel4x32},
	}
}
