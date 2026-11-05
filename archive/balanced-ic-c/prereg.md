<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #206 stage C: the balanced partition scoped to rectangular calls. Predictions

**Why.** Stage B (`archive/balanced-ic/`) lifted 8T Sgemm and Ssymm net of CI on all seven hosts:
+6–11% on six of them, and +1.6% on V1. It failed its ship rule on V2 Ssyrk 8T, −0.4% [−0.5, −0.3].
Ssyrk's triangular units were pre-registered as outside the round model, and the change helped them
nowhere. Scott, 2026-11-04: "Stage C: scope it".

**The change** (branch `exp-balanced-ic`, this commit). `callMC` re-blocks only a split call with no
triangular mask. Syrk and Syr2k (masked) keep the shipped MC, and so do Trsm's rank updates
(`noSplit`). `TestCallMCScope` pins all three cases, and a planted unscoped version turns the
masked case red. Every other part of the run is stage B's: the same driver, hosts, arms, samples and
`O = a716d34`.

**Predictions** (N/O, net of both CIs):
1. **Correctness:** 21 of 21 rc=0.
2. **The 1T identity control:** every 1T row's net range contains zero.
3. **Ssyrk is now an identity row:** every Ssyrk row, 1T and 8T, on every host, has a net range
   containing zero. This is the scope's own witness.
4. **Sgemm and Ssymm 8T reproduce stage B** within ±2 points per host:
   - V1: +1.6 / +1.7;
   - V2: +6.7 / +6.7;
   - V3: +7.3 / +7.3;
   - Zen 5: +9.1 / +8.5;
   - Zen 4: +9.7 / +10.7;
   - Skylake-SP: +11.4 / +10.3;
   - SPR: +5.9 / +6.5.
5. **Strsm** has a net range containing zero on every host.

**Ship rule (stage B's, unchanged).** It ships if no row on any host regresses beyond its CI, and
Sgemm 8T gains net of CI on at least five of the seven hosts. Judged pairs then re-derive the six
README hosts' rows before any published number moves.

## Result (`548d68b`, `run-548d68b.log`; `benchci-<host>-{O,N}.csv`): the ship rule holds

The seven hosts and masks were stage B's. The run took about 19 minutes and produced 2240 rows. The
fleet was verified down by a readable inventory. 21 of 21 correctness suites exited rc=0.

**8T, N/O** (% median [range net of both CIs]):

| host | Sgemm | Ssymm | Ssyrk | Strsm |
|---|---|---|---|---|
| V1 | +0.8 [+0.5, +1.1] | +1.0 [+0.5, +1.2] | −0.2 [−0.6, +0.2] | −0.2 |
| V2 | **+6.8** [+6.6, +7.0] | **+6.8** [+6.7, +7.0] | +0.0 [−0.2, +0.1] | −0.3 |
| V3 | **+7.2** [+7.0, +7.3] | **+7.3** [+7.1, +7.5] | −0.1 [−0.6, +0.1] | −0.0 |
| Zen 5 | **+7.6** [+5.7, +12.0] | **+7.4** [+5.3, +11.7] | +0.2 [−3.5, +2.9] | +1.8 [−3.4, +5.3] |
| Zen 4 | **+9.4** [+6.6, +12.6] | **+10.3** [+8.4, +11.1] | +1.9 [−0.5, +3.2] | +0.6 |
| Skylake-SP | **+10.1** [+9.1, +11.1] | **+9.8** [+9.3, +11.8] | +0.1 [−0.9, +1.0] | −0.2 |
| SPR | **+6.3** [+5.5, +6.9] | **+7.0** [+6.5, +7.4] | +0.5 [+0.0, +0.8] | +1.1 [+0.6, +1.8] |

1. ✓ **Correctness:** 21 of 21.
2. **The 1T identity control: 27 of 28.** Zen 4 Ssymm 1T reads +0.3% [+0.1, +0.5] on a path the
   change cannot reach. With 28 intervals at about 95% each, about 1.4 are expected to exclude zero
   by chance, so this one is within that.
3. ✓ **Ssyrk is now an identity row on all 14.** SPR Ssyrk 8T touches zero at its lower bound
   (+0.0). V2's stage-B −0.4% is gone, at +0.0% [−0.2, +0.1].
4. ✓ **Sgemm and Ssymm 8T reproduce stage B within ±2 points on all seven hosts.** The largest
   drift is Zen 5 Sgemm, 9.1 → 7.6.
5. **Strsm, 6 of 7.** SPR Strsm 8T reads +1.1% [+0.6, +1.8]. Trsm runs `noSplit` (tri.go), so the
   change cannot reach it. Like note 2, it is an unchanged path outside its CI, and it is a gain.

**Ship rule: holds.** No row on any host regresses beyond its CI (the smallest upper bound is
+0.1), and Sgemm 8T gains net of CI on 7 of 7 hosts. V1's gain is small (+0.8%) and remains
unattributed. **The scoped balanced partition ships.** The six README hosts' rows move, so judged
pairs re-derive them before any published number changes.
