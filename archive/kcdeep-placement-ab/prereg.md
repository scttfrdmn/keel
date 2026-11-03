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

## Result (`8667b2a`, `run-8667b2a.log`; `benchci-<host>-<mask>-<arm>.csv` over each cell's 20 samples): placement explains the 8T excess

Hosts: one c8a.48xlarge (EPYC 9R45) and one c7a.48xlarge (EPYC 9R14), each with 24 L3 domains.
- The derived masks were spread `0,8,…,56` (gate-p5's own) and packed `0–7`.
- The run took 27m51s and produced 1440 rows.
- The fleet was verified down by a readable inventory.

**8T, KCDeep's own effect (N/O) under each placement**, change in the median, with its range net of
both CIs:

| row | packed N/O | stage B N/O (2xlarge) | spread N/O | O spread vs O packed |
|---|---|---|---|---|
| Zen 5 Sgemm 8T | +1.6 [+1.5, +2.0] | +1.6 | **+8.2** [+5.5, +10.5] | −4.6% |
| Zen 5 Ssyrk 8T | +2.8 [+2.5, +3.1] | +2.8 | **+6.5** [+4.0, +12.0] | −5.6% |
| Zen 5 Ssymm 8T | +2.2 [+1.6, +3.1] | +1.7 | **+6.7** [+3.7, +8.4] | −3.6% |
| Zen 4 Sgemm 8T | +1.4 [+0.5, +2.1] | +1.9 | **+8.0** [+4.3, +11.9] | −8.6% |
| Zen 4 Ssyrk 8T | +4.0 [+0.7, +7.4] | +3.3 | **+7.6** [+6.1, +9.8] | −7.9% |
| Zen 4 Ssymm 8T | +1.0 [+0.8, +1.7] | +1.7 | **+6.4** [+4.0, +8.1] | −7.1% |

1. ✓ **Packed N/O reproduces stage B within ±1.5 points on all six 8T rows.** The largest gap is
   0.7, on Zen 4 Ssyrk.
2. ✓ **Placement.** On Sgemm and Ssymm, on both hosts, spread N/O exceeds packed N/O by 4.5–6.6
   points. The intervals are disjoint on all four:
   - Zen 5 Sgemm: spread lower bound +5.5 against packed upper bound +2.0;
   - Zen 5 Ssymm: +3.7 against +3.1;
   - Zen 4 Sgemm: +4.3 against +2.1;
   - Zen 4 Ssymm: +4.0 against +1.7.
3. ✗ **No drift.** No 1T row has O/R excluding zero by ≥ 1%. The only O/R that excludes zero at all
   is Zen 5 Ssyrk 1T packed, at −0.3% [−0.7, −0.2]. Across all 24 cells, `c0c373d` → `9bf3bdc` moved
   nothing beyond its CI. That includes `c3ddce5`'s worker counter.
4. **8 of 12.**
   - ✓ All six 8T rows reproduce the judged-pair-vs-README column within 2 points.
   - ✗ Four 1T rows miss: Zen 5 Sgemm, Ssyrk and Ssymm, and Zen 4 Ssyrk.

   The R arm explains them. On this instance, the README's own rev `c0c373d` reads Zen 5 Ssyrk 1T
   at 164.5 and Ssymm 1T at 174.5, against the README's 158.5 and 168.1. Zen 4 Ssyrk 1T reads 98.08
   against 102.3, in the other direction. So the 1T excess is between-instance variation in the
   README's reference draw, which is up to ±4% on Ssyrk 1T even though each draw's own CI is tight.
   It is not drift. That is one witness: one instance per CPU.

**Attribution.** The 8T excess is **placement**, and drift is excluded on every row.

Not pre-registered: under the gate's spread placement, the pre-KCDeep library was 3.6–8.6% slower
at 8T than under packed. Under KCDeep, spread runs level with packed or faster: Zen 5 Sgemm 8T reads
1189.5 spread against 1171.5 packed. So KCDeep did not so much gain from owning an L3 as recover a
penalty that the shipped KC=384 paid when the 8 workers sat on 8 domains. A plausible mechanism is
that the shared packed B panel is re-read across domains once per KC step, and a deeper KC means
fewer steps. That mechanism is **unmeasured**: no counter was read.

Consequences:
- A 1-domain stage B, which is what any 2xlarge is, under-predicts a gate-placement result for any
  change to the nest's KC or NC traffic. Every future blocking stage B on a multi-domain CPU should
  carry both masks.
- How many points a given user sees depends on where the OS places their threads.
