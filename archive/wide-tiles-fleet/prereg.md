<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 3x64 across five x86 µarchs, and through the blocked GEMM on Zen 5: predictions

**From the record.** Under CL 824624 PS2, the 12-accumulator 3x64 tile is spill-free
(`archive/shapegen-ps2/`). On Zen 5 it reaches 99.0% of peak against 78.6% for the shipped 4x32
(`archive/wide-tiles-zen5/`). The reading is latency: two FMA pipes need more than 8 chains in flight.
This run asks two things. Is that a Zen 5 fact or an x86 one? Does it survive the blocked GEMM?

**Hosts:** five 2xlarge/4xlarge instances, single-thread, pinned to core 2, exploration tier:

| role | instance | µarch |
|---|---|---|
| zen5 | c8a.2xlarge | Zen 5 |
| zen4 | c7a.2xlarge | Zen 4 |
| skx | c5n.4xlarge | Skylake-SP |
| spr | c7i.4xlarge | Sapphire Rapids |
| gnr | c8i.4xlarge | Granite Rapids |

**Kernel arms:** `stk` (go1.27.3) and `ps2`, each under `-randlayout` seeds 1–8, 5 rounds with
alternating order. Rows are `Kernel/{2x32,4x32,3x64}/avx512/kc=128` and `Peak/avx512`.

**Nest (zen5 only):** `BenchmarkBlocking` at 2048³ for 4x32 and 3x64 under PS2. The grid is
KC ∈ {128, 192, 256, 384} × MC ∈ {72, 144, 288} × NC = 2048, with seeds 1–4 and 3 rounds. The anchor
is the shipped stock 4x32 at its shipped kc=384/mc=144.

**Correctness** runs on every host, under both toolchains, with the exit status kept. It covers
internal/kern's oracle and scalar-tile tests, and internal/block's new oracle check of every measured
kernel through the blocked Gemm. That check was planted-failure tested: a wrong beta turns it red.

**Predictions** (median of seed medians, % of the same arm's `Peak/avx512`):
1. **Correctness:** every host and toolchain exits 0, with 3x64 subtests present. A failure voids
   that host's 3x64 rows.
2. **Zen 5 anchor:** ps2 3x64 ≥ 95%.
3. **Zen 4:** ps2 3x64 is within ±3% of ps2 4x32. Zen 4's 4x32 already reaches 96.5% on stock, so 8
   accumulators leave it no latency deficit.
4. **Skylake-SP**, the issue-bound class, which dispatches 2x32: ps2 3x64 beats ps2 2x32 by ≥ 10%.
5. **Sapphire Rapids and Granite Rapids:** ps2 3x64 beats ps2 4x32 by ≥ 10% on each.
6. **Nest, Zen 5 under PS2:** the best 3x64 grid point is ≥ 1.15× the best 4x32 grid point. 3x64's
   best KC is ≤ 256, because its B panel per call is 64·KC·4 B, which is 98 KB at kc=384.
7. **Stock nest anchor:** stock 4x32 at its shipped point reads 165–185 GFLOP/s.

**REFUTED if** the best 3x64 nest point is not above the best 4x32 nest point beyond the seed spread.
The kernel win would then not survive the blocked GEMM.

**Scope:** an experiment on a patched toolchain (#197 guardrail: never a published row). It does not
change what keel ships on go1.27.
