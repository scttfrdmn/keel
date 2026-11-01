<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Blocking sweep on Skylake-SP (c5n.18xlarge): predictions (#26, #142)

The lead from `archive/retention-skx-c5n/`: in the feed replay, the 2x32 `full` arm ran **233.5 ms at
kc=512 against 242.9 at the shipped 384** (−3.9%). The C traffic saved at the larger KC outweighed the
extra panel feed. That arm replays costs and is not a real GEMM. This run measures real 2048³ GEMMs
with `BenchmarkBlocking`, through `scripts/retention.sh sweep` unchanged.
- **Grid:** KC ∈ {256, 320, 384, 448, 512, 640, 768} × MC ∈ {72, 144, 288} × NC = 2048, for 2x32 and
  4x32 (42 points).
- **Methodology:** `-count=10 -benchtime=1s`, `GOMAXPROCS=1`. That is the quotable count rather than
  the sweep's exploratory 5.
- **Host:** one on-demand c5n.18xlarge. The driver is `driver-frozen.sh`.

**Disclosed before the run:**
- The thread is unpinned (as in the decomposition run).
- Single-thread only. The published share is 8-thread, so a single-thread win is a lead for an
  8-thread measurement, not a change to any published row.
- Any KC adopted for this host class would need its own pre-registered nest measurement, and a check
  that the FMA-bound class is unaffected.

**Predictions:**
1. **At 2x32, mc=144:** kc=512 is faster than kc=384 by **2–5%**, net of both intervals. The best KC
   at mc=144 is 512 or 640.
2. **Below the shipped KC, 2x32 is slower,** kc=256 by at least 3% against 384. The C traffic per
   call grows as KC shrinks (34.7 ms at kc=128 against 13.1 at 384, from the decomposition run).
3. **MC is second-order:** at each shape's best KC, the three MC values lie within 2% of each other.
4. **4x32 moves in the same direction** as 2x32 between kc=384 and 512, but 4x32 is not dispatched on
   this host.

**REFUTED if** kc=384's interval overlaps the 2x32 best point's at mc=144. In that case the replay's
3.9% does not survive a real GEMM, and the lead is dropped.
