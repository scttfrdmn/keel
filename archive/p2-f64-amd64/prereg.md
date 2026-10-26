<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Pre-registration: the judged float64 P2 run, amd64 (#191, 2026-10-26)

Written and committed before launch. The fleet is read off the archives: c5n.18xlarge
(keel-skx), c7a.48xlarge (keel-zen4) and c8a.48xlarge (keel-zen5), on-demand, us-east-1,
$24.09/hr for the three. It runs `KEEL_ELEM=f64 scripts/gate-p2.sh`, the gate
pre-registered in `958ce2d` (DESIGN.md "P2 at float64").

## The lab pre-check this follows
janus (i9-9960X, Skylake-X, measured slot) at `aa6c78a`: **GREEN, 22 PASS / 0 FAIL /
0 UNMEASURED**. janus is issue-bound, with 2x16/avx512/kc=128 at 46.1% of the float64 peak,
**93.1% of its 49.3% issue roofline**. The float32 2x32 on keel-skx read 94.1% at
`5ec5fea`.

## Predictions
The float64 kernels audit identically to their float32 twins (same insns, spills and
insns/FMA), and the float64 peak is exactly half the float32 one at the same loop. So each
host's **fraction of peak** should match its judged float32 P2 reading:

| host | class (predicted) | float32 P2 reading | float64 prediction | floor |
|---|---|---|---|---|
| keel-skx (Xeon 8124M) | issue-bound | 94.1% of roofline (2x32) | 88–100% of roofline | ≥90% of roofline |
| keel-zen4 (EPYC 9R14) | FMA-bound | 96.8% of peak (4x32) | 85–100% of peak | ≥55% |
| keel-zen5 (EPYC 9R45) | FMA-bound | 65.3% of peak (4x32) | 58–72% of peak | ≥55% |

**Predicted verdict: GREEN.** The margin risk is keel-skx, whose band crosses its 90% floor,
and then keel-zen5. **REFUTED if** any host classifies differently from the table, or any
fraction falls outside its band. A refutation is a finding about whether float32
transfers to float64, whichever way the verdict goes.

**If RED:** the STOP rule applies verbatim (CLAUDE.md, DESIGN.md P2). That means kernel
shaping plus one tile shrink, then `docs/spill-report.md`, a `blocked` issue, and a
decision from Scott. Nothing is retried into green.
