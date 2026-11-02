<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# KCDeep on every AVX-512 host, stage B on the published rows: predictions (#26, #142)

**Why.** `archive/blocking-fma-sweep/` found Skylake-SP's KC lever on Sapphire Rapids: kc=1024 beats
the shipped 384 by +11.9% at 8T, or +11.1% net of both CIs. KC=1024 at the shipped MC was
non-negative, net of CI, on Zen 5, Zen 4 and GNR. archsimd exposes features, not CPU identity, and
SPR shares the FMA class with Zen. So the simplest rule is to set KCDeep = 1024 on every AVX-512 host:
no new fingerprint, and the issue class is unchanged. This run measures that rule on the rows it
would move.

**Run.**
- **arms:** main (`9bf3bdc`) against branch `exp-amd64-kcdeep`. The branch's `applyClassBlocking`
  keys KCDeep on `activeKern.Name == kern.AVX512` instead of on the issue class.
- **hosts:** c8a.2xlarge (Zen 5), c7a.2xlarge (Zen 4), c7i.4xlarge (SPR) and c8i.4xlarge (GNR).
- **benchmark:** BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at n=4096, at 1 and 8 threads, masked to
  one thread per physical core, on stock go1.27.3.
- **samples:** 4 `-randlayout` seeds × `-test.count=5` per arm, with the arms alternating.
- **correctness first:** the branch's Level-3 suite runs on each host, with its exit status.

**Predictions** (new/old, net of both CIs):
1. **Correctness:** rc=0 on all four hosts.
2. **SPR:** Sgemm, Ssyrk and Ssymm gain +6% to +14% at both thread counts.
3. **Zen 5, Zen 4:** Sgemm, Ssyrk and Ssymm stay within [−1%, +3%].
4. **GNR:** the same rows stay within [−2%, +5%].
5. **Strsm, every host:** within ±2%. KCDeep needs m and n ≥ MC, so Strsm's MB=64 rank updates
   keep KC=384.

**Ship rule.** If no row on any host regresses beyond its CI, the rule ships for every AVX-512 host.
Zen 5 and Zen 4 are README hosts, so a judged pair then follows before their rows move.
If a Zen row regresses, the every-AVX-512 rule does not ship. Scoping the change to SPR would need a
new feature fingerprint, which is a separate decision.

## Result (`89e6263`, `run-89e6263.log`; `benchci-<host>-{old,new}.csv` are `tools/benchci` over each arm's 20 samples)

The four hosts, each masked to cores 0–7, one thread per physical core:
- Zen 5: EPYC 9R45;
- Zen 4: EPYC 9R14;
- SPR: Xeon Platinum 8488C;
- GNR: Xeon 6975P-C.

The run took 15m11s and produced 1280 rows. The fleet was verified down by a readable inventory. The
branch's Level-3 suite exited rc=0 on all four hosts.

Each cell is new/old for one row at n=4096, followed by its range net of both CIs.

| row | Zen 5 | Zen 4 | SPR | GNR |
|---|---|---|---|---|
| Sgemm 1T | +1.7% [+1.4, +1.9] | +1.2% [+0.1, +6.7] | **+11.4%** [+10.4, +12.0] | +4.0% [+3.0, +4.9] |
| Sgemm 8T | +1.6% [+1.5, +1.9] | +1.9% [+1.7, +2.2] | **+6.4%** [+5.9, +7.0] | +0.1% [−0.5, +1.4] |
| Ssyrk 1T | +5.0% [+4.6, +5.3] | +6.0% [+3.0, +7.5] | +5.4% [+3.4, +7.4] | +3.8% [+3.4, +4.6] |
| Ssyrk 8T | +2.8% [+2.7, +3.3] | +3.3% [+1.3, +5.2] | +6.8% [+4.2, +10.5] | −0.7% [−1.4, +0.1] |
| Ssymm 1T | +0.2% [+0.0, +0.4] | +0.4% [+0.0, +1.1] | +11.1% [+8.3, +12.4] | +2.3% [+1.6, +3.5] |
| Ssymm 8T | +1.7% [+1.4, +1.9] | +1.7% [+1.4, +2.0] | +5.9% [+4.8, +8.7] | −0.6% [−1.2, +0.4] |
| Strsm 1T | +0.2% [−0.3, +0.7] | −0.9% [−2.5, +2.2] | +3.7% [−1.4, +5.8] | +1.5% [−0.5, +3.9] |
| Strsm 8T | −0.3% [−0.9, +0.8] | −0.1% [−0.9, +1.5] | +0.5% [−0.1, +1.6] | −0.4% [−0.8, +0.0] |

1. ✓ **Correctness.**
2. **4 of 6.** SPR Ssyrk 1T (+5.4%) and Ssymm 8T (+5.9%) fell just below the predicted +6%.
3. ✗, in the good direction. The Zen 5 and Zen 4 rows gain more than predicted:
   - Ssyrk 1T: +5.0% on Zen 5 and +6.0% on Zen 4;
   - Ssyrk 8T: +3.3% on Zen 4.

   No Zen row falls below −1%.
4. ✓ **GNR.** Every row is within [−2%, +5%].
5. **7 of 8.** SPR Strsm 1T reads +3.7%, but its net range straddles zero.

**The ship rule holds.** No row on any host regresses beyond its CI: every net range's upper bound is
≥ 0. The worst is GNR Strsm 8T, at [−0.8%, +0.04%]. **KCDeep = 1024 ships for every AVX-512 host.**
The README's Zen 5 and Zen 4 rows will move, so a judged pair on those two hosts follows before they
are re-derived.
