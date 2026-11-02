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
