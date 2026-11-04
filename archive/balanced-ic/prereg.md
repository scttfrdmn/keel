<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# A balanced ic partition (#206): predictions

**Why.** `archive/v3-load/` found the 8-thread nest losing about 8% to the ic loop's last round.
The model, computed after the fact, predicted the MCPB=36 gain on V2 to within 0.1 points. Scott,
2026-11-04: "a pre-registered A/B of a balanced split for #206".

**The change** (branch `exp-balanced-ic`, `0bd47a5`). `balanceMC` in `internal/block`, regenerated
into `block64`, picks MC per call when the ic loop has more units than workers.
- **The objective** is round efficiency, (m/b) / (w · rounds).
- **Candidates** are whole tiles from MC down to MC/2.
- **The choice** is the largest candidate within 1 point of the best.
- **What it does not touch:** the serial path (w ≤ 1) and calls with fewer units than workers.
- **Tests:** `TestBalanceMC` checks hand-computed cases, re-derived independently in Python. A
  planted disabled rule turns 4 of them red. `TestP5Determinism` (m=1200, 2–8 threads) and
  `TestF64NestParallelIsSerial` both reach the new branch, and both stay bit-identical.

At n=4096 and 8 workers the rule predicts these blocks:

| hosts | kernel | MC: shipped → balanced | units | round efficiency |
|---|---|---|---|---|
| V1, V2, V3 | 5x16-pb | 70 → 65 | 59 → 64 | 0.914 → 0.985 |
| Zen 5, Zen 4, SPR | 4x32 | 144 → 128 | 29 → 32 | 0.889 → 1.000 |
| Skylake-SP | 2x32 | 144 → 128 | 29 → 32 | 0.889 → 1.000 |

**Run** (`driver.sh`). Main `a716d34` (O) against the branch (N) on seven judged-size hosts:
c7g.16xlarge, c8g.48xlarge, c9g.48xlarge, c8a.48xlarge, c7a.48xlarge, c5n.18xlarge and
c7i.48xlarge.
- **Benchmark:** BenchmarkScale at n=4096, 1T and 8T.
- **Placement:** gate-p5's, which is spread on Zen and packed elsewhere.
- **Samples:** 4 seeds × 5 per arm, with the arms alternating.
- **Correctness first:** the branch's tests run on each host.

**Predictions** (N/O, net of both CIs):
1. **Correctness:** all three rc=0 on all seven hosts.
2. **1T is an identity control.** Every 1T row is within its CI on every host, because the change
   is unreachable at one worker. A 1T row moving beyond its CI would mean the arms differ in
   something other than the change.
3. **arm64, 8T:** Sgemm and Ssymm gain **+4% to +7%** on V1, V2 and V3. The model's ceiling is
   0.985/0.914 = +7.8%, less the small-block cost, which was 0–3% at MC=35.
4. **amd64, 8T:** Sgemm and Ssymm gain **+5% to +11%** on Zen 5, Zen 4, Skylake-SP and SPR. The
   model's ceiling is +12.5%. This is the first measurement of #206's amd64 prediction.
5. **Ssyrk 8T** lies within [−2%, +8%] on every host. Its units are triangular, so the round model
   does not describe it.
6. **Strsm** is within ±2% everywhere. Trsm's rank updates run `noSplit`, so the change cannot
   reach them.

**Ship rule.** It ships if no row on any host regresses beyond its CI, and Sgemm 8T gains net of
its CI on at least five of the seven hosts. Six of these hosts carry README rows, so shipping is
followed by judged pairs before any README number moves. **REFUTED** if prediction 2 fails, which
voids the run, or if no host's Sgemm 8T gains net of CI.

## Result (`7e3f860`, `run-7e3f860.log`; `benchci-<host>-{O,N}.csv` over each arm's 20 samples): large 8T gains, but the ship rule fails on Ssyrk

There were seven hosts, with masks as in the driver: spread on Zen, packed elsewhere. The run took
19m09s and produced 2240 rows. The fleet was verified down by a readable inventory. Every host's
three correctness suites exited rc=0.

**8T, N/O** (% median [range net of both CIs]):

| host | Sgemm | Ssymm | Ssyrk | Strsm |
|---|---|---|---|---|
| V1 | +1.6 [+1.1, +1.9] | +1.7 [+1.3, +2.2] | +0.3 [−0.1, +1.0] | +0.6 |
| V2 | **+6.7** [+6.5, +6.9] | **+6.7** [+6.6, +6.8] | **−0.4 [−0.5, −0.3]** | −0.1 |
| V3 | **+7.3** [+7.2, +7.4] | **+7.3** [+7.1, +7.5] | −0.3 [−0.4, −0.0] | +0.3 |
| Zen 5 | **+9.1** [+7.1, +10.5] | **+8.5** [+6.3, +10.6] | −0.6 [−5.1, +3.4] | +3.3 [−2.6, +11.8] |
| Zen 4 | **+9.7** [+5.2, +12.7] | **+10.7** [+8.5, +11.6] | +0.1 | +0.3 |
| Skylake-SP | **+11.4** [+9.7, +12.6] | **+10.3** [+9.0, +11.2] | −0.1 | −0.0 |
| SPR | **+5.9** [+4.3, +6.8] | **+6.5** [+5.6, +7.6] | −0.4 [−1.0, +0.0] | +0.1 |

1. ✓ **Correctness:** 21 of 21 rc=0.
2. ✓ **The 1T identity control holds.** Every one of the 28 1T rows has a net range containing zero.
   The arms differ only in the change.
3. **arm64, 1 of 3 in band.**
   - V2 is +6.7% ✓.
   - V3 is +7.3%, 0.3 above the band, in the good direction.
   - **V1 is only +1.6–1.7% ✗.** V1's 8T rows gain far less than the round model's +7.8% ceiling.
     This is **unattributed**. A plausible reading is that V1's 8T is bound by something other than
     the partition tail, but nothing here measures that.
4. ✓ **amd64, all within one point of band.** The Sgemm/Ssymm gains are:
   - Zen 5: +9.1 / +8.5;
   - Zen 4: +9.7 / +10.7;
   - Skylake-SP: +11.4 / +10.3, with Sgemm 0.4 above the band;
   - SPR: +5.9 / +6.5.

   **#206's amd64 prediction holds.** The 8T tail was costing amd64 6–11%.
5. ✓ **Ssyrk** is within [−0.6%, +0.3%] on every host.
6. **Strsm, 6 of 7.** Zen 5 Strsm 8T reads +3.3%, but its net range [−2.6, +11.8] is noise-limited
   and straddles zero.

**The ship rule fails as written.** Sgemm 8T gains net of CI on 7 of 7 hosts. But **V2 Ssyrk 8T
regresses beyond its CI: −0.4% [−0.5, −0.3]**. V3 Ssyrk 8T sits at the edge, at [−0.4, −0.0].

The round model was pre-registered as not describing Ssyrk, whose units are triangular, and the
change does not help it on any host. So the honest form of the change scopes it to the calls the
model describes. That is a stage C to measure, as `blocking-skx-ab-c/` was for KCDeep, not a relaxed
rule. Nothing ships from this run.
