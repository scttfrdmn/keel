<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Judged pairs after Strsm's MB=128 (#37): predictions

**Why.** `8de1523` shipped MB=128 for Trsm (`archive/trsm-mb-b/`), which moves the README's Strsm
rows on all six hosts. Scott, 2026-11-26: "float32 now, f64 later". The float64 rows are #209's,
in a separate pass.

**Runs.** Three fleets, each run r1 then r2 on one frozen tree: amd64, V1+V2 and V3. Each driver is
`archive/p5-balanced/`'s, byte-identical in code apart from its header and SELF.

**Predictions for r1** (r2 is predicted to reproduce r1). Each band is the README value × stage B's
Strsm delta, ±3%:

| host | row | README | stage B | band |
|---|---|---|---|---|
| Skylake-SP | Strsm 1T / 8T | 38.16 / 267.9 | +24.0 / +22.1% | 45.9–48.7 / 317–337 |
| Zen 5 | Strsm 1T / 8T | 125.2 / 829.0 | +12.1 / +9.3% | 136–145 / 879–933 |
| Zen 4 | Strsm 1T / 8T | 75.08 / 488.6 | +11.6 / +7.9% | 81.3–86.3 / 511–543 |
| V1 | Strsm 1T / 8T | 43.11 / 290.0 | +5.0 / +4.9% | 43.9–46.6 / 295–313 |
| V2 | Strsm 1T / 8T | 49.55 / 335.8 | +1.9 / +1.6% | 49.0–52.0 / 331–351 |
| V3 | Strsm 1T / 8T | 60.63 / 393.5 | +2.3 / +3.2% | 60.2–63.9 / 394–418 |

1. **Gates:** each fleet's era is the one its driver checks. gate-p4 and gate-p3 are GREEN, and
   `-race` is clean.
2. **Rates:** every Strsm rate is in its band. Every Sgemm, Ssyrk, Ssymm and Ceiling row passes
   criterion 9, since MB cannot reach them.
3. **Criterion 9 FAILs** fall only on Strsm rows, and must include Skylake-SP 1T and 8T and Zen 5
   and Zen 4 1T. Those are past README_TOL with tight intervals.
   - Zen 5 and Zen 4 8T may pass despite moving 8–9%, because Zen Strsm 8T is noise-limited and
     criterion 9 also requires disjoint intervals.
   - V1, V2 and V3 pass, since they move ≤ 5%.
4. **Shares:** every judged share passes its registered baseline.
5. **Reproduction:** r2 reproduces every r1 share within 5%.

**REFUTED if** any of these happens:
- a sub-gate is RED;
- a FAIL appears outside the Strsm rows;
- a Strsm rate falls below its README value.

**Landing:**
- superseded and canonical rows in `readme-archives.tsv`;
- `readme-numbers.sh` over the new logs;
- the moved Strsm share baselines re-landed.
