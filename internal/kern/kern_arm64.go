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
		// PROMOTED 2026-10-27 (#198): the pre-broadcast-A kernels, A packed with each element
		// replicated across the 4 lanes (ALanes) so a row's multiplicand is one q-load, not
		// the 3-op BroadcastFloat32x4 (golang/go#81352). Counts are spill-audit's (`shapegen
		// -arch arm64 -verify`): 249/160 and 264/168. Preferred picks 5x16 under both
		// classes (1.556 < 4.111 insns/FMA, 0.45 < 0.5 mem-ops/FMA).
		//
		// On measured rate, by two pre-registered rules (archive/neon-pb/): kernel level,
		// 5x16/8 reached 86.2/81.5/81.4% of the family-max peak on Neoverse V1/V2/V3
		// against 3x24's 45.5-46.4%; and in the nest, single-thread Sgemm n=2048, 5x16 at
		// its own blocking (block.KCPB/MCPB) beat 3x24 at its shipped blocking by
		// +67.8/+70.0/+70.5%, worst round over best on every host (def1b79). 7x12 is the
		// kernel-rate alternate.
		{Name: NEON, MR: 5, NR: 16, Unroll: 8, Fn: vec.KernelPB5x16, InsnsPerFMA: 249.0 / 160, ALanes: 4},
		{Name: NEON, MR: 7, NR: 12, Unroll: 8, Fn: vec.KernelPB7x12, InsnsPerFMA: 264.0 / 168, ALanes: 4},
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
		// Not a spilling tile: the pre-broadcast 4x20/8 (#198), measured and never
		// dispatched, because it ties the dispatched 5x16/8 exactly on both of Preferred's
		// axes (249/160 insns/FMA, 0.45 mem-ops/FMA): as a candidate it would make the
		// choice depend on list order (TestPreferredIsOrderIndependent).
		{Name: NEON, MR: 4, NR: 20, Unroll: 8, Fn: vec.KernelPB4x20, InsnsPerFMA: 249.0 / 160, ALanes: 4},
	}
}

// vectorKernels64 is vectorKernels at float64 (#191): the float32 NEON shapes at
// half the columns (two float64 lanes per V register). The audited counts are the
// shipped bodies' own, which shapegen -verify -arch arm64 -elem f64 reproduces.
func vectorKernels64() []Kernel64 {
	if !vec.HasNEON() {
		return nil
	}
	return []Kernel64{
		{Name: NEON, MR: 8, NR: 4, Unroll: 1, Fn: vec.KernelD8x4, InsnsPerFMA: 94.0 / 16},
		{Name: NEON, MR: 4, NR: 8, Unroll: 1, Fn: vec.KernelD4x8, InsnsPerFMA: 80.0 / 16},
		{Name: NEON, MR: 3, NR: 12, Unroll: 2, Fn: vec.KernelD3x12, InsnsPerFMA: 148.0 / 36},
		// PROMOTED 2026-10-27 (#198): pre-broadcast A at float64 (ALanes 2). Kernel level
		// 86.2/81.6/81.4% of peak (V1/V2/V3); nest, 5x8 at KCPB=256/MCPB=72 beat 3x12 at its
		// shipped blocking by +75.6/+61.6/+69.8% (archive/neon-pb/, def1b79).
		{Name: NEON, MR: 5, NR: 8, Unroll: 8, Fn: vec.KernelDPB5x8, InsnsPerFMA: 249.0 / 160, ALanes: 2},
		{Name: NEON, MR: 7, NR: 6, Unroll: 8, Fn: vec.KernelDPB7x6, InsnsPerFMA: 264.0 / 168, ALanes: 2},
	}
}

// referenceTiles64 is referenceTiles at float64: the spilling twins of float32's
// NEON references (8x12, 8x16, 4x32), measured as evidence, never shipped.
func referenceTiles64() []Kernel64 {
	if !vec.HasNEON() {
		return nil
	}
	return []Kernel64{
		{Name: NEON, MR: 8, NR: 6, Unroll: 1, Fn: vec.KernelD8x6},
		{Name: NEON, MR: 8, NR: 8, Unroll: 1, Fn: vec.KernelD8x8},
		{Name: NEON, MR: 4, NR: 16, Unroll: 1, Fn: vec.KernelD4x16},
		// The pre-broadcast 4x10/8: referenceTiles' 4x20 at float64, for the same reason.
		{Name: NEON, MR: 4, NR: 10, Unroll: 8, Fn: vec.KernelDPB4x10, InsnsPerFMA: 249.0 / 160, ALanes: 2},
	}
}
