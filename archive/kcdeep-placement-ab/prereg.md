<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Why the judged Zen hosts gained more from KCDeep at 8T: predictions (#26, #142)

**Why.** The Zen judged pair (`archive/p5-zen-kcdeep/`) moved the README rows more than the 2xlarge
stage B predicted:

| row | stage B, 2xlarge (N/O) | judged pair vs README (N/R) |
|---|---|---|
| Zen 5 Sgemm 1T / 8T | +1.7 / +1.6 | +2.9 / +7.6 |
| Zen 5 Ssyrk 1T / 8T | +5.0 / +2.8 | +9.2 / +8.2 |
| Zen 5 Ssymm 1T / 8T | +0.2 / +1.7 | +2.8 / +7.2 |
| Zen 4 Sgemm 1T / 8T | +1.2 / +1.9 | +2.4 / +6.2 |
| Zen 4 Ssyrk 1T / 8T | +6.0 / +3.3 | +3.1 / +5.6 |
| Zen 4 Ssymm 1T / 8T | +0.4 / +1.7 | +2.7 / +5.8 |

Two explanations are open, and they are not exclusive.
- **Placement.** gate-p5 pins 8 threads one per L3 domain (mask 0,8,…,56). The 2xlarge's 8 cores
  share one domain. A deeper KC would pay more when each thread owns its L3.
- **Drift.** The README's denominator is the old revs `f18f110`/`c0c373d`. N/R also carries every
  library change since, including `c3ddce5`, "one worker counter for both nests".

**Run.** One c8a.48xlarge (Zen 5) and one c7a.48xlarge (Zen 4), stock go1.27.3. There are three arms:
- **R** = `c0c373d`, the README's rev;
- **O** = `9bf3bdc`, main before KCDeep;
- **N** = this tree, whose Go code is byte-identical to the judged `35c2894`.

The benchmark is BenchmarkScale Sgemm/Ssyrk/Ssymm at n=4096, at 1T and 8T, under two 8-core
socket-0 masks:
- **spread:** the first core of each of 8 distinct L3 domains, which is the gate's placement;
- **packed:** 8 cores sharing cpu 0's domain, which is the 2xlarge's situation.

Each arm and mask gets 4 `-randlayout` seeds × `-test.count=5`, with the arm order rotated per seed.

A second witness comes for free: the 1T rows run under both masks, so their two readings should agree.

**Predictions** (net of both CIs, in percentage points of ratio):
1. **Packed N/O reproduces stage B** within ±1.5 points on every 8T row.
2. **Placement:** on Sgemm and Ssymm, on both hosts, spread N/O at 8T exceeds packed N/O at 8T by at
   least 2 points.
3. **Drift exists at 1T.** The 1T excess cannot be placement, since there is one thread. So O/R is
   at least 1% in magnitude, net of CI, on at least 2 of the 6 1T rows.
4. **Consistency:** spread N/R reproduces the judged-pair-vs-README column within 2 points per row.

**Attribution rule.**
- Placement is attributed if 2 holds.
- Drift is attributed on each row where O/R excludes zero by at least 1%.
- If neither holds, the excess stays unattributed and is said so.
