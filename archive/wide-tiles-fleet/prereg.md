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

## Result (`435e7f4`, `exploration-fleet-435e7f4.log`): not refuted

Five hosts, read back as AMD EPYC 9R45, AMD EPYC 9R14, Xeon Platinum 8124M, Xeon Platinum 8488C and
Xeon 6975P-C. It took 18m40s, and the fleet was verified down by a readable inventory.
- Each host produced 320 kernel rows (5 rounds × 8 seeds × 2 toolchains × 4 benchmarks).
- Zen 5 also produced 300 nest rows.

**1. ✓ Correctness.** Under both toolchains on all five hosts, `kern.test` and `blk.test` exited 0.
Each reported 18 3x64 subtests in the kernel tests and 5 in the blocked-Gemm oracle check.

**Kernels** (median of seed medians, % of the same arm's `Peak/avx512`):

| host | stock: 2x32 / 4x32 / 3x64 | PS2: 2x32 / 4x32 / 3x64 |
|---|---|---|
| Zen 5 (9R45) | 151.8 / 182.2 / 187.9 (65.7%) | 151.3 / 224.1 / **281.8 (98.5%)** |
| Zen 4 (9R14) | 112.4 / 113.0 / 38.0 (32.5%) | 108.8 / **116.2 (99.3%)** / 111.5 (95.3%) |
| Skylake-SP (8124M) | 78.0 / 59.5 / 66.5 | 87.4 / 82.9 / **105.3 (57.4%)** |
| Sapphire Rapids (8488C) | 105.6 / 105.5 / 110.6 | 105.8 / 122.0 / **149.2 (69.5%)** |
| Granite Rapids (6975P-C) | 119.7 / 119.2 / 130.6 | 120.2 / 141.2 / **164.8 (67.1%)** |

2. ✓ **Zen 5:** 98.5%.
3. ✗ **Zen 4:** 3x64 is 4.0% below PS2 4x32 (95.3% against 99.3%), outside ±3%. On stock, the spilling
   3x64 collapses to 32.5%.
4. ✓ **Skylake-SP:** +20.5% over PS2 2x32.
5. ✓ **Sapphire Rapids +22.3% and Granite Rapids +16.7%** over PS2 4x32.

Against each host's best shipped shape on stock, PS2 3x64 is:
- Zen 5 +55%, Skylake-SP +35%, Sapphire Rapids +41%, Granite Rapids +38%;
- Zen 4 −1%.

**Nest, Zen 5, 2048³, single-thread** (median of seed medians, mc = 72 / 144 / 288):

| KC | PS2 4x32 | PS2 3x64 |
|---|---|---|
| 128 | 200.6 / 203.4 / 203.9 | **240.8 / 244.9 / 246.1** |
| 192 | 203.2 / 206.4 / 207.2 | 236.7 / 237.9 / 238.4 |
| 256 | 204.0 / 205.7 / 205.4 | 229.9 / 232.9 / 233.4 |
| 384 | 205.8 / 205.1 / 205.4 | 238.2 / 239.8 / 239.2 |

6. ✓ **The best 3x64 point is 1.19× the best 4x32**: 246.1 [244.1–247.7] at kc=128/mc=288, against
   207.2 [194.4–207.9] at kc=192/mc=288, with full seed separation. Its best KC is 128, within the
   predicted ≤ 256.
7. ✓ **Stock shipped 4x32** (kc=384/mc=144) reads 165.9 [165.9–166.4], inside 165–185 at the edge.

**What it establishes:**
- **The 12-accumulator win is an x86 fact, not a Zen 5 one.** Under PS2, 3x64 is the best shape on
  four of five hosts, by 17–55% over what ships today.
- **On Zen 4 it is not:** that host is already at the FMA limit with 8 accumulators.
- **The win survives the blocked GEMM on Zen 5:** PS2 + 3x64 is **1.48×** today's shipped stock
  configuration (246.1 against 165.9 GFLOP/s at 2048³), 86% of the kernel peak.
- **Not monotone in KC:** 3x64's best is the smallest KC tried and its worst is 256, unlike 4x32's flat
  curve, so blocking for it needs its own sweep.

**Not established:**
- the nest on the other four hosts;
- 8 threads;
- any published row (patched toolchain, exploration tier, #197 guardrail).

Nothing changes for go1.27 users.
