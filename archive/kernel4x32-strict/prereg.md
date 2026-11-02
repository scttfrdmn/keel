<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Kernel4x32 with strict guards on Zen 5: predictions

**The model under test.** On Zen 5 the kernel's rate falls with instructions per FMA (IPF). Stock
loops at 1.19 and 2.25 IPF read ~100% of peak, `fo` at 3.75 reads 91%, Kernel4x32 under PS2 at 4.50
reads 83.1%, and stock Kernel4x32 at 6.25 reads 65.3%. Those points fit an issue limit of
W ≈ 7.5 instructions per cycle at 2 FMAs per cycle: ceiling = min(1, W / (2·IPF)). If the model is
right, cutting instructions raises the rate by the predicted amount. If it is wrong, the cut does
nothing.

**The change** (branch `exp-kernel4x32-indexed`, `d77d9f8`): the reslice guards become strict
(`len > width`, T19), and the last k-step is peeled. spill-audit, steady-state K-loop:

| | shipped | re-spelled |
|---|---|---|
| stock go1.27.3 | 50 insns (6.25 IPF), 12 copies | **40 (5.00)**, 12 copies |
| + CL 824624 PS2 | 36 (4.50), 0 copies | **26 (3.25)**, 0 copies |

**Run:** one c8a.2xlarge, pinned to core 2. Arms: `sold` / `snew` (stock, shipped / re-spelled) and
`cold` / `cnew` (PS2 toolchain, shipped / re-spelled). Each is built under `-randlayout` seeds 1–8.
5 rounds, with the arm order rotating each round. `Kernel/4x32/avx512/kc=128` and `Peak/avx512`.

**Predictions** (each arm's median of seed medians, as % of its own `Peak/avx512`):
1. **Correctness first:** `TestKernelsAgainstOracle`, `TestVectorAgainstScalarTile` and the root
   Level-3 sweeps PASS for the new kernel under both toolchains. A FAIL voids every rate below.
2. **sold reproduces 65.3% ± 2**, and **cold reproduces 83.1% ± 2** (this campaign's anchors).
3. **snew reads 70–80%.** The model gives 7.5 / 10 = 75%. That is a gain of at least 8% over sold,
   with **snew's slowest seed above sold's fastest**.
4. **cnew reads at least 88%.** The model caps it at 100% (7.5 / 6.5 > 1), so another limit, such as
   loads at 0.75 per FMA or the broadcasts, may bind first. That is why the band is open above and
   not a point.

**REFUTED if** snew's median is within sold's seed spread. In that case instruction count is not
what limits Zen 5's kernel, and the issue-width model is withdrawn.

**Scope:** an experiment on a branch. Shipping it would mean changing `tools/shapegen`'s emitter
(the shipped kernels must stay its fixed points) and re-measuring every number that depends on the
kernel.
