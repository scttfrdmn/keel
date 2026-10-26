<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The judged float64 P2 run, amd64 (#191) — GREEN

Rev `fe3992f` (the float64 kernels landed at `aa6c78a`; the commit after it adds only this
directory's pre-registration and driver). Fleet read off the archives: c5n.18xlarge
(keel-skx), c7a.48xlarge (keel-zen4) and c8a.48xlarge (keel-zen5), on-demand, us-east-1,
2026-10-26 21:26:57Z–21:43:48Z (16m51s, about $6.8 at $24.09/hr). Teardown was verified
against `spawn list` and `aws-fleet.sh status`, and nothing is billing.

**`KEEL_ELEM=f64 scripts/gate-p2.sh`: 28 PASS / 0 FAIL / 0 UNMEASURED → GREEN**
(`driver-p2-f64-judged-fe3992f.log`). The lab pre-check on janus at `aa6c78a` was 22/0/0
GREEN (`gate-p2-f64-lab-janus-aa6c78a.log`).

## Against the pre-registration (`prereg.md`, committed before launch)

| host | class predicted / measured | float64 reading | predicted band | floor | held |
|---|---|---|---|---|---|
| keel-skx | issue / issue | 2x16 at 46.1% of peak = **93.5% of its 49.2% roofline** | 88–100% | ≥90% | yes |
| keel-zen4 | FMA / FMA | 4x16 at **96.5%** of peak (96.4% net of CI) | 85–100% | ≥55% | yes |
| keel-zen5 | FMA / FMA | 4x16 at **65.2%** of peak (64.9% net of CI) | 58–72% | ≥55% | yes |

Every prediction held. The float32 judged P2 readings were 94.1% of roofline (skx),
96.8% (zen4) and 65.3% (zen5); float64 lands within 0.6 points of each. That is what the
structural argument predicts: identical audits and a peak exactly half at the same loop
mean the same fraction. One run is a verdict, not a published figure. No README row
depends on it.
