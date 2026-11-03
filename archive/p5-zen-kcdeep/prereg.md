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

## Result (`35c2894`, `judged-r{1,2,3}-35c2894.log`): refuted on the 8-thread bands, in the good direction

Three runs on one c8a.48xlarge (EPYC 9R45) and one c7a.48xlarge (EPYC 9R14). Both were masked to 8
cores over 8 of the node's 12 cache domains (mask 0,8,…,56). Each fleet was verified down by a
readable inventory.

**r1 is set aside, not counted.** Its gate-p3 died at the dev host with "No space left on device".
The internal disk filled transiently: keel's `build/` was 0.9 GB, and the space went to other
projects' scratch. So gate-p4 rendered **1 UNMEASURED and is RED**, with 0 FAIL. Its remote samples
are intact: they match r2 within 0.6% on every row except Strsm 8T, which is noise-limited. They are
archived. But a run without green sub-gates is not a README source, so r3 was run on the same frozen
tree.

**r2 and r3:**
- gate-p4 65/0/0 GREEN on both, with gate-p3 GREEN underneath; `-race` clean.
- The only FAILs are the two criterion-9 lines, one per host.
- Tallies: r2 55 PASS / 2 FAIL / 2 BASELINE / 2 REPORTED; r3 54 / 2 / 2 / 3.

**r2 rates against the README** (r3 within 0.6% except Strsm 8T):

| row | Zen 5 | Zen 4 |
|---|---|---|
| Sgemm 1T / 8T | +2.9% ✓ / **+7.6%** ✗ | +2.4% ✓ / **+6.2%** ✗ |
| Ssyrk 1T / 8T | **+9.2%** ✗ / **+8.2%** ✗ | +3.1% ✓ / +5.6% ✓ |
| Ssymm 1T / 8T | +2.8% ✓ / **+7.2%** ✗ | +2.7% ✓ / **+5.8%** ✗ |
| Strsm 1T / 8T | −0.8% ✓ / +1.3% ✓ | +0.6% ✓ / −0.7% ✓ |

1. ✗ **r1 only** (the disk); r2 and r3 ✓.
2. ✗ **Six rate bands are exceeded, every one upward.** Five of them are 8T rows. The 8T gain on the
   judged size is 2–4 points above the 2xlarge stage-B delta.
   - **Unattributed.** A plausible mechanism is the mask: each of the 8 threads owns a separate L3
     domain here, while the 2xlarge's 8 cores shared theirs. A deeper KC then pays more per thread.
   - Second candidate: the README's denominator is `f18f110`/`c0c373d`, so the delta also carries
     every library change since.
   - Separating the two would take a 48xlarge A/B of `9bf3bdc` against `35c2894`.
3. ✓ **Shares.** Every judged share passes its registered baseline: Zen 5 Sgemm 52.8% against 48.70%,
   Zen 4 Sgemm 78.9% against 74.65%.
4. ✗ **Criterion 9 fails on more rows than predicted:**
   - Zen 5: Sgemm 8T, Ssyrk 1T and 8T, Ssymm 8T;
   - Zen 4: Sgemm 8T, Ssymm 8T.
5. ✓ **Reproduction.** r3 reproduces r2 within 5% on every share.

**REFUTED** under the pre-registered clause ("a FAIL anywhere other than a criterion-9 Ssyrk 1T row"),
in the good direction: no row regressed.

**The README is not regenerated yet.** `readme-numbers.sh` refuses the pair on one row: Zen 4 Strsm
reads PASS (50.6%) in r2 and NOISE-LIMITED REPORTED (47.3%, the intervals costing 4.30 points) in r3.
Every other verdict agrees. No earlier pair produced PASS-against-NOISY on one row. The generator's
comment says a REPORTED reading "is not a rival verdict", but its code treats this one as rival. The
publication rule is put to Scott on #142, not resolved here.
