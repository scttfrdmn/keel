# Pre-registration: the first green L1 result (written 2026-10-21, before either run)

Rev dc519af, era ceil8, both runs at the SAME rev; nothing is archived or committed between them.
arm64 first (keel-gvt3 c7g.16xlarge, keel-gvt4 c8g.48xlarge), then amd64 (keel-skx c5n.18xlarge,
keel-zen4 c7a.48xlarge, keel-zen5 c8a.48xlarge).

## Predicted, per criterion
The last runs at this code shape were arm64 51 PASS / 4 FAIL and amd64 69 PASS / 3 FAIL / 3 BASELINE.
What changed since: the README was regenerated (criterion 9's Strsm rows), the L1 bar takes criterion
9's whole rule, amd64 L1 judges n=1048576 only and its baselines + witness rows are landed, and the
owing arm64 row is landed.

- **L1: PASS on all five hosts**, every judged row `registered`. Drift lines EXPECTED on arm64 (the 9
  #180 rows, 0.3-2.8% below pre-change baselines), and possibly on amd64 zen5 Sdot/Saxpy.
- **amd64 L1: one REPORTED line per host** (n=256/4096/65536, #179). So REPORTED = 3 on amd64, 0 on arm64.
- **criterion 9: PASS on all five hosts** (README regenerated from the refresh at N=2).
- **BASELINE: 0 on both** (every judged row registered, every host witnessed).
- **VERDICT: GREEN on both** -- arm64 ~55 PASS / 0 FAIL, amd64 ~72 PASS / 0 FAIL / 3 REPORTED.

## Named risks, so a red is read correctly rather than explained afterwards
1. **An L1 row more than 5% below its baseline** -> FAIL, and that is the bar doing its job. Most
   exposed: amd64 zen5 Sdot/Saxpy (between-run spread up to 4.24% at this tier) and arm64 V1
   Sdot/n=65536, whose one pre-change draw (16.27) sat 5.9% below the row now landed (17.29).
2. **criterion 9 on a Strsm row**, if this draw sits outside the N=2 interval and >5% off centre.
3. **Anything else red is unpredicted** and is reported as a finding, not absorbed.

A red on risk 1 or 2 does NOT get a baseline re-registered or a band widened in response.
