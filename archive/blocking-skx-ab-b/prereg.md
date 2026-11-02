<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Skylake-SP stock blocking, stage B (the published rows): predictions (#26, #142)

**The change** (branch `exp-skx-kc`): on a host `kern.HostClass` reports as issue-bound, the root
package sets `block.KC = 1024` at init. FMA-class hosts are untouched. MC stays 144. The value comes
from stage A (`archive/blocking-skx-ab/`), where kc=1024/mc=144 at 2048³ was +7.2% at 1T and +5.4% at
8T, net of both intervals. `TestIssueClassKC` pins `block.KC` under both classes; a planted wrong value
turns it red.

**Run.** One c5n.18xlarge, stock go1.27.3. Old = main `b5ed22d`, new = branch. Both run
`BenchmarkScale`, the very rows the README publishes and gate-p5 judges: Sgemm, Ssyrk, Ssymm and Strsm
at n=4096, threads ∈ {1, 8}. Pinned to gate-p5's mask (cores 0–7 of socket 0), with 4 `-randlayout`
seeds × `-test.count=5` per arm, arms alternating, pooled by `tools/benchci`. The branch's Level-3
suite runs on the instance first.

**Predictions** (new/old, net of both intervals):
1. **Correctness:** the branch's suite exits 0 on the issue-bound host itself.
2. **Sgemm, Ssyrk, Ssymm:** +4% to +9% at 1T and +3% to +7% at 8T. That is stage A's gain, diluted by
   n=4096's larger problem.
3. **Strsm:** +0% to +5% at both thread counts. Its diagonal solves do not use the nest's KC.
4. **No row regresses** beyond its interval.

**REFUTED for shipping if** Sgemm at 8 threads does not beat old net of both intervals. That is the row
the published share is computed from.

**If it holds, shipping still needs** the CHANGELOG, and **a judged amd64 gate-p5 pair**, because the
README's Skylake-SP rows move. That is a cost decision for Scott, not part of this run.

## Result (`9784cd8`, `run-9784cd8.log`, `benchci-{old,new}.csv`): not shippable as built

One c5n.18xlarge (Xeon Platinum 8124M), masked to cores 0–7 of socket 0. It took 15m08s, and the fleet
was verified down by a readable inventory. There are 320 rows. The branch's Level-3 suite exited 0 on
the host.

| row (n=4096) | old GFLOP/s [CI] | new GFLOP/s [CI] | new/old, range net of both CIs |
|---|---|---|---|
| Sgemm 1T | 66.91 [65.73, 67.63] | 73.62 [72.52, 74.36] | **+10.0%** [+7.2%, +13.1%] |
| Sgemm 8T | 453.90 [444.30, 454.40] | 494.40 [492.30, 496.80] | **+8.9%** [+8.3%, +11.8%] |
| Ssymm 1T | 65.91 | 72.91 | +10.6% [+7.5%, +13.6%] |
| Ssymm 8T | 446.35 | 490.30 | +9.9% [+9.2%, +12.4%] |
| Ssyrk 1T | 66.00 | 74.57 | +13.0% [+8.1%, +15.7%] |
| Ssyrk 8T | 475.85 | 527.35 | +10.8% [+10.6%, +12.0%] |
| **Strsm 1T** | 38.31 | 35.94 | **−6.2%** [−7.4%, −4.0%] |
| **Strsm 8T** | 285.15 | 256.60 | **−10.0%** [−10.9%, −9.2%] |

1. ✓ **Correctness.**
2. ✗ (in the good direction) **Sgemm, Ssymm and Ssyrk** gain +8.9% to +13.0%, above the predicted
   +4–9% (1T) and +3–7% (8T).
3. ✗ **Strsm regresses:** −6.2% and −10.0%, against a predicted 0 to +5%.
4. ✗ **A row regresses**, beyond its interval.

The refutation clause did not fire: Sgemm 8T beats old by at least 8.3% net of CIs. But a 10%
regression on a published row means **this change does not ship**.

**Why Strsm regresses, and why my prediction was wrong about it.** The prediction assumed Strsm does
not use the nest's KC. Its diagonal blocks (`MB = 64`) indeed do not. But its rank updates go through
the same nest as thin GEMMs, with M = MB = 64 rows (`internal/block/tri.go`). Each packed B panel of
KC × NC is then reused by only 64 rows, so raising KC from 384 to 1024 makes the B pack and its trip
through the cache dominate. The full-height Sgemm/Ssymm/Ssyrk (m = 4096) amortize it; Strsm's rank
updates cannot. The next step is to keep the larger KC for full-height GEMMs only, so the Trsm path
keeps the shipped KC, and to re-measure all eight rows.
