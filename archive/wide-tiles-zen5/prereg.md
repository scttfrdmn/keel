<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Wide tiles on Zen 5 (5x32: 10 accumulators, 3x64: 12): predictions

**The reading under test.** Under CL 824624 PS2, Zen 5's 4x32 kernel reaches 83% of peak
(`archive/go-patches/cl824624/`), and 10 fewer instructions per iteration moved nothing
(`archive/kernel4x32-strict/`). So the limit is not issue width. The remaining explanation is latency:
two FMA pipes at about 4 cycles need about 8 FMAs in flight, and 4x32 has exactly 8 accumulators, so it
has no slack. If that is right, more accumulators should raise the rate, and only under PS2, because only
there are 10- and 12-accumulator tiles spill-free (`archive/shapegen-ps2/`).

**Kernels** (branch `exp-wide-tiles`; 5x32 and 3x64 are emitted unchanged by `tools/shapegen`,
measured and never shipped):

| kernel | accumulators | stock: stack refs / insns per FMA | PS2: stack refs / insns per FMA |
|---|---|---|---|
| 4x32 (shipped) | 8 | 0 / 6.25 | 0 / 4.50 |
| 5x32 | 10 | **2** / 5.90 | 0 / 4.00 |
| 3x64 | 12 | **2** / 5.50 | 0 / 3.33 |

**Run:** one c8a.2xlarge, pinned to core 2. Two toolchain arms, `stk` (go1.27.3) and `ps2`
(go1.28-devel_7745cfc920), each built under `-randlayout` seeds 1–8. 5 rounds with alternating order.
`Kernel/{4x32,5x32,3x64}/avx512/kc=128` and `Peak/avx512`.

**Predictions** (median of seed medians, as % of the same arm's `Peak/avx512`):
1. **Correctness first:** `TestKernelsAgainstOracle` and `TestVectorAgainstScalarTile` exit 0 under
   both toolchains, with 5x32 and 3x64 subtests present (counted). A failure voids their rates.
2. **The anchor:** ps2 4x32 reads 82.7–83.1% ± 2.
3. **Under PS2, both wide tiles beat 4x32 by more than the seed spread:** 5x32 at least 88% and 3x64 at
   least 90%. The more accumulators, the higher: 3x64 ≥ 5x32.
4. **Under stock, the spilling wide tiles do not beat stock 4x32** (65%), because their stack refs cost
   more than the slack buys.

**REFUTED if** neither wide tile beats ps2 4x32 beyond the seed spread. In that case accumulator count
is not the limit either, and the latency reading is withdrawn as the issue-width one was.

**Scope:** an experiment. These shapes ship only through `tools/shapegen` and `kern.Preferred`, and only
on a toolchain that compiles them spill-free. So nothing changes for go1.27 users either way.

## Result (`5a6518b`, `exploration-zen5-5a6518b.log`): not refuted, and 3x64 reaches the FMA peak

One c8a.2xlarge (AMD EPYC 9R45). It took 9m14s, and the fleet was verified down by a readable
inventory. There are 320 rows, 5 rounds × 8 seeds × 2 toolchains × 4 benchmarks.

| GFLOP/s, median of seed medians [seed range] | stock go1.27.3 (peak 286.6) | PS2 (peak 286.4) |
|---|---|---|
| 4x32 (8 acc) | 179.5 [179.1–186.8], 62.6% | 225.1 [224.9–240.0], 78.6% |
| 5x32 (10 acc) | 173.6 [170.2–176.5], 60.6% | 232.8 [170.7–239.3], 81.3% |
| **3x64 (12 acc)** | 183.4 [182.3–188.9], 64.0% | **283.5 [277.0–284.3], 99.0%** |

1. ✓ **Correctness:** 36 subtests for 5x32 and 3x64 passed under each toolchain, and both
   `kern.test` runs exited 0.
2. ✗ **Anchor:** PS2 4x32 reads 78.6%, outside the 80.7–85.1% band. This rev adds two kernels to the
   package, which re-places every function, and 4x32 under PS2 is the layout-sensitive kernel (6.7%
   spread here, 13.4% at `0510f94`). It is a miss of the anchor, not of the hypothesis.
3. **Split.**
   - ✓ **3x64 reads 99.0%.** Its slowest seed, 277.0, is above 4x32's fastest, 240.0, and 3x64 is not
     below 5x32.
   - ✗ **5x32 reads 81.3%**, below the predicted 88%, and does not separate from 4x32. Two seeds
     collapse to 171 and 178, a 40% layout spread, so 10 accumulators appear to give too little slack to
     be robust to placement.
4. ✓ **Stock:** neither spilling wide tile beats stock 4x32 beyond the spread. 3x64 is +2.2% with
   overlapping ranges, and 5x32 is below.

**What it establishes:**
- **The latency reading stands.** With PS2 removing the copies, 8 accumulators sit near 80%, 10 are
  marginal and fragile, and 12 reach the FMA limit on Zen 5.
- **Under PS2, keel's best AVX-512 kernel on Zen 5 is 1.58× the kernel shipped on stock today**
  (283.5 against 179.5 GFLOP/s).
- **That shape exists only because PS2 makes a 12-accumulator tile spill-free**
  (`archive/shapegen-ps2/`). Without the patch it spills and gains nothing.

**Not established:** the nest (3x64 needs NR=64 B panels and its own blocking), any other host
(Zen 4, Skylake-SP, SPR/GNR), and anything about go1.27 users, for whom nothing changes. Exploration
tier, one host, single-threaded, on a patched toolchain (#197 guardrail: never a published row).
