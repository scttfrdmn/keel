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

## Result (`93ceb55`, `run-93ceb55.log`)

About 18 minutes (about $1.25). The fleet was verified down by a readable inventory. Load at launch was 0.18.
**Not refuted, but three of the four finer predictions missed.**

1. **kc=512 against 384 at 2x32/mc=144: +4.1% on medians** (71.2 against 68.4 GFLOP/s), inside the
   predicted 2–5%. But the best KC at mc=144 is **768** (72.3), not 512 or 640 ✗. The **"net of both
   intervals" clause is UNMEASURED**: `retention.sh sweep` prints ranked medians only and deletes its
   raw samples (`mktemp`), so no interval is in the log. That is a property of the instrument,
   disclosed and not assumed away.
2. ✗ **kc=256 is not slower:** 68.9 against 68.4 at mc=144. The C-traffic argument behind the
   prediction did not show up in real GEMMs at this size.
3. ✗ **MC is not second-order:** at kc=768, mc=288/144/72 read 73.2 / 72.3 / 69.6, a 5% spread,
   with mc=288 best at every KC from 448 up.
4. ✓ **4x32 moves the same way:** 61.0 → 62.4 from kc=384 to 512, and its best is also kc=768/mc=288
   (63.0).

**What it shows:**
- **The best point is the grid's corner,** kc=768/mc=288 at **73.2 GFLOP/s against the shipped
  kc=384/mc=144 at 68.4 (+7.0%)**, so the optimum may lie beyond it.
- **There is a nearby limit:** the A block at mc=288, kc=768 is 288 × 768 × 4 = 884,736 B, just
  under this part's 1 MiB L2.
- **August's janus sweep** (grid capped at kc=512) found the shipped point near-optimal. Neither its
  host nor its grid is this one's, so nothing here contradicts it.
- **This is single-thread on medians.** Before it can change anything it needs an interval-bearing
  A/B (shipped against the candidate, plus one point past the corner), and an 8-thread measurement,
  since the published share is 8-thread. Any default would change through `kern.Class`, as the sweep
  header says, never by being the top row here.
