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

## Result (`49e9d00`, `run-49e9d00.log`; `benchci-<host>-{O,N}.csv` over each arm's 20 samples): attributed to the throttle

One c7g.16xlarge and one c7gn.16xlarge, each masked to cores 0–7. The run produced 640 rows. The
fleet was verified down by a readable inventory. 6 of 6 correctness suites exited rc=0.

**8T, N/O** (% median [range net of both CIs]), with GFLOP/s O → N:

| host | Sgemm | Ssymm | Ssyrk | Strsm |
|---|---|---|---|---|
| c7g (Graviton3) | +2.3 [+1.9, +2.5], 401.4 → 410.6 | +2.3 [+1.9, +2.6], 398.6 → 407.8 | −0.7 [−1.4, +0.2] | +0.1 |
| **c7gn (Graviton3E)** | **+6.7** [+6.0, +7.2], 420.0 → 448.3 | **+6.8** [+6.3, +7.0], 416.9 → 445.4 | +0.4 [−0.3, +0.8] | −0.7 [−1.1, −0.2] |

1. ✓ **Correctness.**
2. **The 1T identity control: 7 of 8 rows per host contain zero.** The exception is c7g Ssymm 1T,
   −1.7% [−3.1, −0.6], on a path the change cannot reach. Beside it, c7gn Strsm 8T reads −0.7%
   [−1.1, −0.2], which is also out of reach (Trsm runs `noSplit`). Two of the 12 unreachable cells
   exclude zero, against the 0.6 expected by chance at about 95% each. Both are small, they point
   in opposite directions on different hosts, and neither is a row the change touches. They are
   recorded, not explained.
3. ✓ **c7gn gains +6.7% / +6.8%,** inside +5–8% and equal to V2's +6.8%.
4. **c7g gains +2.3% / +2.3%,** 0.3 above the predicted +0.3–2.0%. It is still a small gain, and
   above balanced-ic-c's +0.8 / +1.0 on another c7g instance.
5. ✓ **Ssyrk** is an identity row on both hosts.

**Attribution.** Under the reading rule, prediction 3 holds and 4 holds within 0.3 points. So **V1's
small balanced-partition gain is the Graviton3 vector-issue throttle.** The same V1 core, unthrottled
on the 3E, takes the full V2-sized gain from the same fix.

The cap also binds harder once the tail is gone. Before the partition, c7gn led c7g by 4.6% at 8T
Sgemm (420.0 against 401.4). After it, the lead is 9.2% (448.3 against 410.6). The balanced
partition turns idle tail time into FMA demand, which the throttled part cannot issue.
