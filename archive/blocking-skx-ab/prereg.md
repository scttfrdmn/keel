<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Skylake-SP blocking on stock Go, stage A (intervals, 8 threads, past the edge): predictions (#26, #142)

**Why.** This is the one improvement from this campaign that go1.27 users could get today.
`archive/blocking-skx-c5n/` found kc=768/mc=288 single-thread +7.0% over the shipped kc=384/mc=144 on
c5n.18xlarge, for the dispatched 2x32 kernel. But that was on medians with no interval
(`retention.sh sweep` deletes its samples), single-thread only, and at the grid's corner. Shipping
needs an interval, the 8-thread effect, and the shape of the surface past the corner.

**Run.** One c5n.18xlarge (keel-skx), stock go1.27.3, main with no code change. `BenchmarkBlocking`
at 2048³ for 2x32 (dispatched) and 4x32 (reference):
- **grid:** KC ∈ {384, 512, 768, 1024} × MC ∈ {144, 288, 432} × NC = 2048;
- **samples:** 4 `-randlayout` seeds × `-test.count=5`, so 20 samples per point, pooled by
  `tools/benchci`;
- **placement:** 1T on core 2; 8T on 8 physical cores of socket 0 (from `lscpu`).

The working sets at the edge are disclosed. The A block, MC × KC × 4 B, is 1,179,648 B at 288 × 1024
and 1,327,104 B at 432 × 768, both past this part's 1 MiB L2. 288 × 768 is 884,736 B, inside it.

**Predictions** (2x32, net of both intervals):
1. **1T:** the best point beats the shipped kc=384/mc=144 by at least 5%, and lies at KC ≥ 768 and
   MC ≥ 288.
2. **8T:** the 1T-best point also beats shipped at 8 threads, by at least 3%.
3. **Past the edge:** neither kc=1024/mc=288 nor kc=768/mc=432 beats kc=768/mc=288 by more than 1%,
   because each one's A block outgrows the L2.
4. **4x32 moves the same way** (reference only; it is not dispatched here).

**REFUTED for shipping if** at 8 threads no point beats shipped net of both intervals. A
single-thread-only gain would not move the published 8-thread share, and nothing ships.
