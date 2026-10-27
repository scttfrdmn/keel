<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Pre-registration: the judged arm64 P2 run, both elements (#191, 2026-10-27)

Written and committed before launch. The fleet is read off the archives: c7g.16xlarge
(keel-gvt3, Neoverse-V1) and c8g.48xlarge (keel-gvt4, Neoverse-V2), on-demand, us-east-1,
$10.35/hr. It runs `KEEL_ARCH=arm64 scripts/gate-p2.sh` at `KEEL_ELEM=f64`, then `f32`, the
arm pre-registered in `a404442`.

## The exploration pre-check this follows
c7g.4xlarge and c8g.4xlarge at `a404442` (`exploration-gvt-4xlarge-a404442.log`): 22 PASS
/ 0 FAIL at each element. The floor is UNMEASURED by design: 4xlarge is correctness-class,
not judged. Readings, reported not judged:

| host | class | float64 best shipped | float32 best shipped |
|---|---|---|---|
| Neoverse-V1 | FMA-bound | 3x12 at 86.1% of peak net of CI | 3x24 at 85.7% |
| Neoverse-V2 | FMA-bound | 3x12 at 60.6% net | 3x24 at 60.8% |

## Predictions (per-core fractions should not depend on the instance size)
| host | class | float64 band | float32 band | floor |
|---|---|---|---|---|
| keel-gvt3 | FMA-bound | 78–94% | 78–94% | ≥55% |
| keel-gvt4 | FMA-bound | 55–68% | 55–68% | ≥55% |

**Predicted verdict: GREEN at both elements.** The margin risk is keel-gvt4, about 6 points
over its floor in the pre-check, with its band starting at the floor itself. **REFUTED if**
a host classifies differently or a fraction falls outside its band. **The float32 arm is a
new verdict on kernels that already shipped** (DESIGN.md "P2 on arm64"): if it is RED, the
STOP rule applies to them too, and it is a finding about kernels already in the tree, not
a reason to adjust the gate. Nothing is retried into green.
