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
