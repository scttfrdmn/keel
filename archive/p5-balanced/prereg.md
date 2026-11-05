<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Judged pairs after the balanced ic partition (#206): predictions

**Why.** `64073d7` shipped the balanced partition (`archive/balanced-ic-c/`). It moves the README's
8T Sgemm and Ssymm rows on the six published hosts, so they are re-derived by judged gate-p5 pairs.
Scott, 2026-11-05: "Run both now".

**Runs.** There are three fleets, each run r1 then r2 on the same frozen tree:
- **amd64:** Skylake-SP, Zen 5 and Zen 4, era famaxx86, OpenBLAS v0.3.29;
- **arm64:** V1 and V2, era famax, v0.3.29;
- **V3:** era famax, v0.3.34, by the #196 ruling.

V3 is a fleet of its own because the OpenBLAS pin is fleet-wide. Mixing it in would change V1/V2's
reference within the era. Each driver copies its judged precedent byte for byte apart from its
header and SELF: `p5-amd64-famaxx86`, `p5-arm64-famax` and `p5-arm64-c9g`.

**Predictions for r1** (r2 is predicted to reproduce r1). Each 8T band is the README × stage C's
delta, ±3%:

| host | row | README | stage C | band | criterion 9 |
|---|---|---|---|---|---|
| Zen 5 | Sgemm 8T | 1212 | +7.6% | 1265–1343 | FAIL |
| Zen 5 | Ssymm 8T | 1153 | +7.4% | 1201–1275 | FAIL |
| Zen 4 | Sgemm 8T | 743.7 | +9.4% | 789–838 | FAIL |
| Zen 4 | Ssymm 8T | 724.8 | +10.3% | 776–823 | FAIL |
| Skylake-SP | Sgemm 8T | 497.7 | +10.1% | 532–564 | FAIL |
| Skylake-SP | Ssymm 8T | 488.5 | +9.8% | 520–553 | FAIL |
| V2 | Sgemm 8T | 438.4 | +6.8% | 454–482 | FAIL |
| V2 | Ssymm 8T | 435.9 | +6.8% | 452–480 | FAIL |
| V3 | Sgemm 8T | 576.3 | +7.2% | 599–636 | FAIL |
| V3 | Ssymm 8T | 573.8 | +7.3% | 597–634 | FAIL |
| V1 | Sgemm 8T | 404.5 | +0.8% | 396–420 | pass |
| V1 | Ssymm 8T | 402.6 | +1.0% | 394–419 | pass |

1. **Gates:** each fleet's era is the one its driver checks. gate-p4 and gate-p3 are GREEN
   underneath, and `-race` is clean.
2. **Rates:** every Sgemm and Ssymm 8T rate is in its band. Every 1T row and every Ssyrk and Strsm
   row passes criterion 9, since stage C's identity rows are unchanged.
3. **Criterion 9** fails on exactly the ten 8T rows marked FAIL: past README_TOL, and with CIs
   disjoint.
4. **Shares:** every judged 8T share is at or above its registered baseline.
5. **Reproduction:** r2 reproduces every r1 share within 5%.

**REFUTED if** any of these happens:
- a sub-gate is RED;
- a FAIL appears outside those ten criterion-9 rows;
- a predicted-unchanged row fails criterion 9.

**Landing:**
- superseded and canonical rows in `readme-archives.tsv`;
- `readme-numbers.sh` over the new pairs;
- the moved share baselines re-landed.

Each architecture lands after its own pair.
