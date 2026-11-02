<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The float64 twin of issue-class KCDeep on Skylake-SP: predictions (#26, #142)

**Why.** At float32, issue-class KCDeep=1024 shipped (`archive/blocking-skx-ab-c/`, `archive/p5-skx-kcdeep/`).
Sgemm/Ssymm/Ssyrk gained +8.9–13.5% on the published rows, and Strsm was flat. block64 is generated
from block, so it has `KCDeep`, but nothing sets it. Its shipped KC is 512. The published float64
Skylake-SP rows (1T / 8T GFLOP/s) are Dgemm 33.83 / 226.2, Dsyrk 33.52 / 240.2, Dsymm 33.58 / 223.9 and
Dtrsm 18.12 / 130.8.

**Branch** `exp-skx-kc64`: issue-class hosts also set `block64.KCDeep`. `TestIssueClassKC64` and
`TestPlanKCDeep64` pin it, and a planted shape-blind rule reddens both Trsm cases. All suites pass,
including the float64 Level-3 sweeps under the forced issue class.

**Run** (one c5n.18xlarge, stock go1.27.3, one launch):
- **Stage A.** `BenchmarkBlocking64` at 2048³ for 2x16 (dispatched) and 4x16. The grid is KC ∈ {512,
  768, 1024, 1536} × MC ∈ {144, 288}, run at 1T (core 2) and at 8T (8 physical cores of socket 0), with
  4 seeds × count 5.
- **Stage B.** `BenchmarkLevel3F64`, the published rows (Dgemm/Dsyrk/Dsymm/Dtrsm at n=4096, 1T and
  8T), for main `1ebbcd3` against the branch built at three candidates, KCDeep64 ∈ {768, 1024, 1536}.
  The mask is 8 cores, with 4 seeds × count 3 and rotating arm order. Intervals come from
  `tools/benchci`.
- **Correctness first:** the branch's float64 Level-3 suite on the host, with its exit status.

**Predictions:**
1. **Correctness:** exit 0.
2. **Stage A, 1T:** the best 2x16 point beats the shipped kc=512/mc=144 by at least 4% net of both
   intervals, at KC ≥ 1024.
3. **Stage A, 8T:** MC=144 is at least as good as MC=288 at the best KC (the partition starvation
   seen at float32).
4. **Stage B, the best candidate:** Dgemm, Dsyrk and Dsymm each gain +5% to +13% at 1T and at 8T, net
   of both intervals, and the best candidate is 1024 or 1536.
5. **Stage B, every candidate:** Dtrsm stays flat (interval includes 1.00) at 1T and 8T, because the
   thin rank updates keep KC.

**REFUTED for shipping if** no candidate gives Dgemm 8T a gain net of both intervals, or the chosen
candidate regresses any row beyond its interval.

## Result (`c5bcd48`, `run-c5bcd48.log`, `benchci-*.csv`): REFUTED for shipping

One c5n.18xlarge (Xeon Platinum 8124M). It took 37m42s, and the fleet was verified down by a readable
inventory. There are 640 stage-A and 384 stage-B rows. The branch's float64 Level-3 suite exited 0 on
the host.

**Stage A, 2x16 at 2048³, GFLOP/s** (mc = 144 / 288):

| KC | 1T | 8T |
|---|---|---|
| 512 (shipped) | **33.94** / 33.56 | **219.20** / 215.45 |
| 768 | 34.22 / 34.07 | 218.50 / 216.30 |
| 1024 | 34.32 / **34.67** | 216.80 / 216.40 |
| 1536 | 33.76 / 34.18 | 209.35 / 194.40 |

**Stage B, published rows (n=4096), new/old with the range net of both intervals:**

| row | KC 768 | KC 1024 | KC 1536 |
|---|---|---|---|
| Dgemm 1T | +1.2% [−2.7, +3.7] | +2.3% [−2.1, +7.7] | +3.3% [−1.5, +7.2] |
| Dgemm 8T | +1.3% [−1.8, +2.4] | +0.9% [−1.5, +2.6] | +1.3% [−2.2, +2.2] |
| Dsymm 1T / 8T | −1.4% / +1.6% | −1.2% / +1.4% | +2.4% / +1.1% |
| Dsyrk 1T / 8T | +2.5% / +1.7% | +3.0% / +1.2% | +3.3% / +0.2% |
| Dtrsm 1T / 8T | +1.4% / −0.5% | +0.1% / −0.5% | −0.3% / −0.4% |

1. ✓ **Correctness.**
2. ✗ **The best 1T point is +2.2% (+1.6% net)**, against at least 4% predicted, though it is at KC ≥ 1024.
3. ✓ **At 8T, MC=144 ≥ MC=288 at the best KC.** The best 8T point is the shipped one.
4. ✗ **The medians gain +0.2% to +3.3%, and not one is net of both intervals.**
5. ✓ **Dtrsm is flat** with every candidate.

**REFUTED for shipping, on its own clause:** no candidate gives Dgemm 8T a gain net of both intervals.
Float64's shipped KC=512 is already at or near Skylake-SP's optimum: at 8 threads it is the best grid
point, and kc=1536 loses 4.5%. The float32 KCDeep win does not transfer, and this run does not
establish why. Branch `exp-skx-kc64` is kept as the record and not merged. The README's float64 rows are
untouched.
