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

## Result (`eb76812`, `run-eb76812.log`; `tabulate.py` → `tables.txt`): the solves are no longer the cost, and MB=128 beats 64 in all 24 cells

Six judged-size hosts, stock go1.27.4, each dispatching its own kernel. The run took 15m53s and
produced 224 rows per host (28 × 2 thread counts × 4 passes), the full count. The fleet was verified
down by a readable inventory. Strsm/Trsm/Strmm correctness exited rc=0 on all six.

**A design gap, recorded first.** At 4 samples per cell, `benchci` returns infinite intervals,
because a 95% interval needs at least 6 samples. So this run supports medians and a worst-case
range bound (slowest sample of one arm against the fastest of the other), not net-of-CI verdicts.
Stage B must take at least 6.

**The solve share at the shipped MB=64** (solve-only/full, median):

| | Skylake-SP | Zen 5 | Zen 4 | V1 | V2 | V3 |
|---|---|---|---|---|---|---|
| left, 1T | 2.5% | 4.5% | 3.9% | 6.8% | 7.6% | 7.6% |
| left, 8T | 3.5% | 8.2% | 6.0% | 6.4% | 6.9% | 6.9% |
| right, 1T | 6.5% | 12.1% | 8.5% | 4.7% | 4.3% | 4.1% |
| right, 8T | 5.9% | 12.5% | 9.0% | 4.0% | 4.2% | 4.1% |

**Full time at MB=128 against MB=64** (median; negative is faster):

| | Skylake-SP | Zen 5 | Zen 4 | V1 | V2 | V3 |
|---|---|---|---|---|---|---|
| left, 8T (the published row's side) | −17.1% | −6.4% | −5.5% | −4.1% | −1.2% | −3.1% |
| left, 1T | −19.2% | −9.7% | −16.9% | −3.9% | −2.4% | −2.8% |
| right, 8T | −17.3% | −8.3% | −11.3% | −27.7% | −29.3% | −29.5% |
| right, 1T | −16.9% | −11.3% | −14.3% | −28.2% | −25.2% | −30.7% |

1. **5 of 6.** Left 1T share at MB=64 is 3.9–7.6%, except Skylake-SP at 2.5%, below the 3% floor.
2. ✗ **0 of 6.** Right 1T share is 4.1–12.1%, against the predicted 15–60%. The right-side solve is
   no longer slow. #181's interchange and the axpy/divide fixes did their job.
3. ✓ **12 of 12.** The 8T share is within ±5 points of 1T on every host and side.
4. ✓ **6 of 6.** The left 8T optimum is at MB ≥ 96: MB=192 on all three amd64 hosts and MB=128 on
   all three Graviton.
5. ✗ **0 of 6.** The right 8T optimum is at MB=192–256 everywhere, the *opposite* of the predicted
   ≤ 64. It is the opposite of the dev host's MB=48 too (tri.go's comment). At MB=256 the Graviton
   right side is 37–40% faster.
6. ✗ **3 of 6.** MB=64's left-8T cost against the best is ≤ 6% on the Graviton hosts, but 10.0%,
   11.3% and 18.3% on Zen 5, Zen 4 and Skylake-SP.

**What it shows.**
- **#37's premise is closed: the diagonal solves are not where Strsm's time goes.** At MB=64 they
  take 2.5–12.5% of the time, for 1.6% of the work.
- **The cost is the blocking.** Larger MB makes the rank updates fewer and fatter: the repack term
  falls as 1/MB, and each update GEMM gets more rows. That reading is consistent with the shares
  above, and it is not separately attributed.
- **One MB serves both sides on every host.** MB=128 is faster than 64 in **all 24 cells** (6 hosts
  × 2 sides × 2 thread counts). 192 and 256 gain more on amd64 and on the right side, but lose to
  128 on the Graviton left side, which is the published row's side.

**The decision rule** ("left's best beats 64 by ≥ 3% at 8T on ≥ 4 hosts"):
- On medians it fires on 5 of 6: 18.3, 11.3, 10.0, 4.1 and 3.1%, with V2 at 1.2%.
- On the range bound, which is all 4 samples allow, it fires on 2: Skylake-SP +16.8% and Zen 5
  +3.3%.

Stage B is the instrument that decides, so it follows. It tests MB=128 on the published rows, with
≥ 6 samples per arm.

**Scope for stage B.** MB is also Trmm's block, and `block64` inherits it through f64gen with no
override. So float64 Dtrsm/Dtrmm would move unmeasured, which is #209's shape. The float32 change
should either carry a float64 measurement or hold float64 at 64 by an f64gen override.
