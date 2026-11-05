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
