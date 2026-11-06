<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Does the Graviton3 throttle explain V1's small balanced-partition gain? Predictions

**Why.** The balanced ic partition (#206) gave Neoverse V2/V3 +6.8–7.3% at 8T on Sgemm/Ssymm, but V1
(c7g) only +0.8–1.0% (`archive/balanced-ic-c/`). Both use the same kernel (5x16-pb) and the same
unit arithmetic: MC 70 → 65, 59 → 64 units. `archive/graviton3e/` then found that a c7g's vector
issue throttles under multi-core load: its 8T FMA ceiling is 87.8% of 8×1T, against 99.8% on the
c7gn (Graviton3E, the same V1 core). The hypothesis is that c7g's 8T rows are held by the throttle,
not by the partition tail, so the balanced partition buys little there. Scott, 2026-11-05: "rerun
the balanced-split A/B on a c7gn."

**Run** (`driver.sh`). This is `archive/balanced-ic-c/driver.sh` with the fleet changed to one
c7g.16xlarge and one c7gn.16xlarge.
- **Arms:** O = `a716d34` (before the partition) against N = this tree (ships it).
- **Benchmark:** BenchmarkScale at n=4096, 1T and 8T, on cores 0–7.
- **Samples:** 4 seeds × 5 per arm.
- **Correctness first:** each host runs its suites before measuring.

**Predictions** (N/O, net of both CIs):
1. **Correctness:** 6 of 6 rc=0.
2. **The 1T identity control:** every 1T row's net range contains zero, on both hosts.
3. **c7gn (no throttle):** 8T Sgemm and Ssymm gain **+5% to +8%**, in V2's range (+6.8%).
4. **c7g (control):** 8T Sgemm and Ssymm gain **+0.3% to +2.0%**, reproducing balanced-ic-c's
   +0.8/+1.0.
5. **Ssyrk** is an identity row on both hosts, since the scope excludes it.

**Reading rule.** If 3 and 4 both hold, V1's small gain is attributed to the Graviton3 throttle.
The same partition fix pays on the unthrottled V1 as on V2. If the c7gn also gains ≤ 2%, the
throttle is not the explanation, and V1's small gain stays unattributed.
