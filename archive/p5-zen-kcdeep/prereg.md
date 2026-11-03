<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# gate-p5 on Zen 5 and Zen 4 with AVX-512 KCDeep, era famaxx86: predictions (#26, #142)

**Why.** Main now blocks full-height float32 GEMMs at KC=1024 on every AVX-512 host, not only the
issue class. See CHANGELOG [Unreleased] and `archive/blocking-avx512-b/`, which measured the
published n=4096 rows on c8a.2xlarge and c7a.2xlarge:
- Zen 5: +0.2–5.0%;
- Zen 4: +0.4–6.0%;
- Ssyrk 1T was the largest mover on both.

So the README's Zen rows move. This judged pair re-derives them on the judged sizes, c8a.48xlarge and
c7a.48xlarge masked to 8 cores. Skylake-SP's dispatch did not change, and its rows were re-derived under
KCDeep in `archive/p5-skx-kcdeep/`, so it is not re-run. The instrument is unchanged, so the era stays
famaxx86. The driver is `driver-frozen.sh`. r1 and r2 run back to back on the same frozen tree.

**A disclosed cross-size step.** The deltas below come from 2xlarge hosts and are applied to 48xlarge
README rows. Each 48xlarge is masked to 8 cores, but it is not the same host as a 2xlarge: the 2xlarge
Zen 5 read Ssyrk 1T at 164.3 old, against the README's 158.5. So each band is ±3 points around the
stage B delta, not the delta's own CI.

**Predictions for r1** (r2 is predicted to reproduce r1). Each rate is new against the README:

| row | Zen 5 | Zen 4 |
|---|---|---|
| Sgemm 1T / 8T | −1.3% to +4.7% / −1.4% to +4.6% | −1.8% to +4.2% / −1.1% to +4.9% |
| Ssyrk 1T / 8T | +2.0% to +8.0% / −0.2% to +5.8% | +3.0% to +9.0% / +0.3% to +6.3% |
| Ssymm 1T / 8T | −2.8% to +3.2% / −1.3% to +4.7% | −2.6% to +3.4% / −1.3% to +4.7% |
| Strsm 1T / 8T | −3% to +3% | −3% to +3% |

1. **Era and gates:** era famaxx86, with gate-p4 and gate-p3 GREEN underneath and `-race` clean.
2. **Rates:** every rate is in its band above.
3. **Shares:** every 8T share is at or above the host's registered famaxx86 baseline, so each PASSes.
4. **Criterion 9:** any FAIL is on a Ssyrk 1T row, the only rows predicted to move more than
   README_TOL (5%). Every other row passes criterion 9.
5. **Reproduction:** r2 reproduces every r1 share within 5%, the same-claim band.

**REFUTED if** any of these happens:
- gate-p3 or gate-p4 is RED;
- a FAIL appears anywhere other than a criterion-9 Ssyrk 1T row;
- a Zen Sgemm, Ssyrk or Ssymm rate falls more than 3% below the README.
