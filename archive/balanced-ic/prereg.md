<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# A balanced ic partition (#206): predictions

**Why.** `archive/v3-load/` found the 8-thread nest losing about 8% to the ic loop's last round.
The model, computed after the fact, predicted the MCPB=36 gain on V2 to within 0.1 points. Scott,
2026-11-04: "a pre-registered A/B of a balanced split for #206".

**The change** (branch `exp-balanced-ic`, `0bd47a5`). `balanceMC` in `internal/block`, regenerated
into `block64`, picks MC per call when the ic loop has more units than workers.
- **The objective** is round efficiency, (m/b) / (w · rounds).
- **Candidates** are whole tiles from MC down to MC/2.
- **The choice** is the largest candidate within 1 point of the best.
- **What it does not touch:** the serial path (w ≤ 1) and calls with fewer units than workers.
- **Tests:** `TestBalanceMC` checks hand-computed cases, re-derived independently in Python. A
  planted disabled rule turns 4 of them red. `TestP5Determinism` (m=1200, 2–8 threads) and
  `TestF64NestParallelIsSerial` both reach the new branch, and both stay bit-identical.

At n=4096 and 8 workers the rule predicts these blocks:

| hosts | kernel | MC: shipped → balanced | units | round efficiency |
|---|---|---|---|---|
| V1, V2, V3 | 5x16-pb | 70 → 65 | 59 → 64 | 0.914 → 0.985 |
| Zen 5, Zen 4, SPR | 4x32 | 144 → 128 | 29 → 32 | 0.889 → 1.000 |
| Skylake-SP | 2x32 | 144 → 128 | 29 → 32 | 0.889 → 1.000 |

**Run** (`driver.sh`). Main `a716d34` (O) against the branch (N) on seven judged-size hosts:
c7g.16xlarge, c8g.48xlarge, c9g.48xlarge, c8a.48xlarge, c7a.48xlarge, c5n.18xlarge and
c7i.48xlarge.
- **Benchmark:** BenchmarkScale at n=4096, 1T and 8T.
- **Placement:** gate-p5's, which is spread on Zen and packed elsewhere.
- **Samples:** 4 seeds × 5 per arm, with the arms alternating.
- **Correctness first:** the branch's tests run on each host.

**Predictions** (N/O, net of both CIs):
1. **Correctness:** all three rc=0 on all seven hosts.
2. **1T is an identity control.** Every 1T row is within its CI on every host, because the change
   is unreachable at one worker. A 1T row moving beyond its CI would mean the arms differ in
   something other than the change.
3. **arm64, 8T:** Sgemm and Ssymm gain **+4% to +7%** on V1, V2 and V3. The model's ceiling is
   0.985/0.914 = +7.8%, less the small-block cost, which was 0–3% at MC=35.
4. **amd64, 8T:** Sgemm and Ssymm gain **+5% to +11%** on Zen 5, Zen 4, Skylake-SP and SPR. The
   model's ceiling is +12.5%. This is the first measurement of #206's amd64 prediction.
5. **Ssyrk 8T** lies within [−2%, +8%] on every host. Its units are triangular, so the round model
   does not describe it.
6. **Strsm** is within ±2% everywhere. Trsm's rank updates run `noSplit`, so the change cannot
   reach them.

**Ship rule.** It ships if no row on any host regresses beyond its CI, and Sgemm 8T gains net of
its CI on at least five of the seven hosts. Six of these hosts carry README rows, so shipping is
followed by judged pairs before any README number moves. **REFUTED** if prediction 2 fails, which
voids the run, or if no host's Sgemm 8T gains net of CI.
