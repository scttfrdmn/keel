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
// 0.625 mem-ops/FMA — so Preferred ranks it first under ClassFMA and ClassIssue
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
	}
}

// referenceTiles carries the two shapes the model predicts SPILL (8x16 → 37 live,
// 4x32 → 41 live, both over 32). They are Measured() but not in Kernels(), exactly
// as amd64's ReferenceTile is: benchmarked and -S-audited so the spill prediction
// is graded on the record rather than assumed, and excluded from what ships so
// nothing ever runs a spilling tile. If the audit refutes a prediction — a shape
// the model called spilled that does not — it is promoted to vectorKernels then,
// on the assembly's word, not this file's guess.
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
