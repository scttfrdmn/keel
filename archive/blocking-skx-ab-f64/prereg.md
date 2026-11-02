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
