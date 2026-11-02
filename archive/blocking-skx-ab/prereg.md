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

## Result (`716d5c8`, `run-716d5c8.log`; `benchci-{1t,8t}.csv` are `tools/benchci` over each point's 20 samples)

One c5n.18xlarge (Xeon Platinum 8124M). It took 29m37s, and the fleet was verified down by a readable
inventory. There are 960 rows. The 8-thread mask was socket-0 cores 0–7.

**2x32, GFLOP/s, median [95% CI]:**

| KC | 1T: mc=144 / 288 / 432 | 8T: mc=144 / 288 / 432 |
|---|---|---|
| 384 | **68.66** [68.62, 68.72] / 70.04 / 70.38 | **450.20** [449.80, 450.60] / 448.20 / 304.60 |
| 512 | 71.27 / 72.71 / 72.67 | 465.15 / 463.35 / 315.35 |
| 768 | 72.39 / 73.03 / 72.77 | 467.85 / 463.90 / 316.30 |
| 1024 | **73.90** [73.66, 74.01] / 73.27 / 73.59 | **476.25** [474.70, 477.90] / 464.70 / 319.40 |

1. **Split.**
   - ✓ The best 1T point beats the shipped kc=384/mc=144 by **+7.6% (+7.2% net of both intervals)**,
     and it lies at KC ≥ 768.
   - ✗ The best MC is 144, not ≥ 288. The best point is kc=1024/mc=144.
2. ✓ **At 8T, the 1T-best point beats shipped by +5.8% (+5.4% net).** The old lead, kc=768/mc=288,
   would have failed this bar: +3.0%, +2.7% net.
3. **Split.**
   - ✓ Past the A-block edge, kc=1024/mc=288 and kc=768/mc=432 sit within 1% of kc=768/mc=288.
   - **The surface still rises along KC at mc=144:** kc=1024 is the grid's best, so KC has no measured
     optimum yet.
4. ✓ **4x32 moves the same way:** +4.6% at 1T and +3.0% at 8T at its best point.

**Not refuted for shipping:** a point beats shipped at 8 threads net of both intervals.

**The mc=432 collapse at 8T is a partition artefact of n=2048, not a cache effect.** `splitIC` hands
out MC-row blocks, and 2048/432 is only 5 blocks for 8 workers. So MC must be judged at the published
n=4096, where 4096/144 = 28 blocks. **Stage B therefore measures the published rows**, BenchmarkScale
Sgemm/Ssyrk/Ssymm/Strsm at n=4096 and 1 and 8 threads, with KC raised for the issue class and MC kept
at 144.
