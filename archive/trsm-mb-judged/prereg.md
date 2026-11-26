<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Strsm's diagonal-solve share and MB on the judged hosts: predictions (#37)

**Why.** #37's status comment (2026-10-21) leaves two done-when criteria unmet. Both were taken on
the dev host only, one M4 Pro NEON core:
- the diagonal solves' **time share on the gate hosts**;
- **MB chosen from that measurement**.

Strsm is the weakest published row on every host. At 8T it reaches 50–80% of Sgemm's rate, from
Skylake-SP's 268 against 549 GFLOP/s up to Zen 4's 489 against 813. Scott, 2026-11-26: "go".

**What is known.**
- **The work share.** At n=4096, left side, MB=64, the solves are 1.59% of the work.
- **The dev host** (tri.go's comment, n=4096): left prefers MB=256 (−5.6% time) and right prefers
  MB=48 (−6.5%).
- **The 2026-08 lab sweep** (n=2048, before the axpy/divide fixes) put the solve at 53% of the time
  at MB=64. That predates `090eb32` and `06f35a4`, so it is not a prior here.

**Run** (`driver.sh`). Six README hosts at judged sizes, stock go1.27.4:
- **Hosts and kernels:** Skylake-SP 2x32; Zen 5 and Zen 4 4x32; V1, V2 and V3 5x16-pb.
- **Benchmark:** BenchmarkTrsmMB's `full` and `solve-only` arms at n=4096, both sides, MB ∈ {32,
  48, 64, 96, 128, 192, 256}.
- **Threads:** 1T on core 2, and 8T with gate-p5's placement.
- **Samples:** 4 passes.

The solve share is solve-only/full on the same side, MB and thread count.

**Predictions:**
1. **Left 1T share at MB=64:** 3–20% on every host.
2. **Right 1T share at MB=64:** 15–60%.
3. **Thread count:** the 8T share is within ±5 points of the 1T share on the same host and side,
   because `strips()` splits rank updates and solves over the same column axis.
4. **Left optimum:** the best full time at 8T is at MB ≥ 96 on at least 4 of 6 hosts.
5. **Right optimum:** the best full time at 8T is at MB ≤ 64 on at least 4 of 6.
6. **Cost of the shipped MB:** MB=64 costs at most 6% against that host's best MB on the left side
   at 8T, on every host.

**What this decides.** If the left side's best MB beats 64 by ≥ 3% at 8T on at least 4 hosts, a
pre-registered stage B on the published rows follows. Otherwise MB stays 64, chosen by this
measurement, and #37's last two criteria close. A per-side MB (#181's finding) is a separate design
question, and this run gives it its gate-host input.
