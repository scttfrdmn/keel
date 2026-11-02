<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# gate-p5 on Skylake-SP with issue-class KCDeep, era famaxx86: predictions (#26, #142)

**Why.** Main now blocks full-height GEMMs at KC=1024 on issue-bound hosts (`block.KCDeep`, CHANGELOG
[Unreleased]; stage C, `archive/blocking-skx-ab-c/`, measured Sgemm/Ssymm/Ssyrk +8.9% to +13.5% on the
published n=4096 rows, and Strsm flat). The README's Skylake-SP rows therefore move. This judged pair
re-derives them on keel-skx (c5n.18xlarge) alone, the precedent being `archive/p5-arm64-c9g/`. FMA-class
hosts never set KCDeep, so the Zen rows cannot move. The instrument is unchanged, so the era stays
famaxx86. The driver is `driver-frozen.sh`; r1 and r2 run back to back on the same frozen tree.

**Predictions for r1** (r2 is predicted to reproduce r1):
1. **Era and gates:** era famaxx86, gate-p4 and gate-p3 GREEN underneath, and `-race` clean.
2. **8T shares of the host's own 8-thread ceiling**, against famaxx86's registered 29.0 / 30.4 / 28.4 /
   17.0:
   - Sgemm **30.8–32.6**, Ssyrk **32.5–34.5**, Ssymm **30.5–32.3** (each famaxx86 share × stage C's
     gain, ±3%);
   - Strsm **16.5–17.5**, unchanged.

   Each is judged against keel-skx's registered famaxx86 baseline and PASSes, being above it.
3. **Rates against the README:** Sgemm 1T 73–77 GFLOP/s (README 68.72) and 8T 485–505 (README 456).
4. **Criterion 9 FAILs on exactly the six moved rows**, Sgemm, Ssyrk and Ssymm at 1T and 8T, each more
   than README_TOL (5%) off the published figure. The Strsm rows and the `Ceiling/compute` row pass.
   That is the expected rendering of a library change before the README is regenerated, not a defect.
5. **r2 reproduces every r1 share within 5%** (the same-claim band), so the pair can land and
   regenerate the rows.

**REFUTED if:**
- gate-p3 or gate-p4 is RED;
- any FAIL appears outside those six criterion-9 rows;
- the Strsm share leaves 16.0–18.0;
- or the Sgemm 8T share is not above 30.0.
