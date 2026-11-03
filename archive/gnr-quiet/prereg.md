<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# A quieter Granite Rapids re-measure: predictions (#26, #142)

**Why.** GNR read inconclusive in `archive/blocking-fma-sweep/`:
- its 1T intervals on c8i.4xlarge were ±8–10%;
- its best 8T point was kc=256/mc=144 at +6.1%, on a non-monotone surface where kc=256 lost
  everywhere else.

AVX-512 KCDeep then shipped to GNR on stage B (`archive/blocking-avx512-b/`). There GNR read −0.7%
to +4.0%, with no row regressing beyond its CI, though Strsm 8T reached only [−0.8%, +0.04%]. This
run re-measures both on a full-size c8i.48xlarge (96 cores, SMT2), so no small-instance co-tenancy
is involved. Where socket 0 has 8 or more L3 domains, it also measures both placements, because
`archive/kcdeep-placement-ab/` showed a 1-domain mask under-predicts the gate's placement.

The driver is `driver.sh`:
- **sweep:** 4x32 at 2048³, the same 15 KC × MC points, 1T and 8T, 4 seeds × 5;
- **A/B:** `9bf3bdc` against this tree on BenchmarkScale at n=4096, 4 seeds × 5.

**Predictions:**
1. **Quieter:** every sweep point's 1T CI is ≤ 1% of its median, against ±8–10% before.
2. **The kc=256 lead was noise:** at 8T packed, kc=256/mc=144 does not beat kc=384/mc=144 net of
   both CIs.
3. **The deeper KC is non-negative at 8T packed:** kc=1024/mc=144 is not below kc=384/mc=144 net
   of CI.
4. **A/B, packed:** N/O lies within [−2%, +5%] on Sgemm, Ssyrk and Ssymm at 1T and 8T, and Strsm
   within ±2%. No row regresses beyond its CI.
5. **A/B, spread (if run):** N/O at 8T is at least packed N/O on Sgemm and Ssymm.

**REFUTED for keeping KCDeep on GNR if** any A/B row regresses beyond its CI under packed. KCDeep
is shipped, so that outcome reopens #142 for a GNR-specific scope, which would need a feature
fingerprint and a decision.

## Result (`af673a3`, `run-af673a3.log`; `benchci-{s1,s8,ab-O,ab-N}.csv` over each point's 20 samples): KCDeep holds on GNR

One c8i.48xlarge, a Xeon 6975P-C. Its socket 0 has a **single 480 MiB L3 domain** and 3 NUMA nodes
(SNC3), so spread is not a distinct placement on this part and only packed ran. The driver said so,
as designed. The 8 cores were 0–7, all on NUMA node 0. The run took 23m22s and produced 920 rows. The
fleet was verified down by a readable inventory.

**The sweep, 4x32 at 2048³, against the shipped kc=384/mc=144**, change with its range net of both CIs:

| point | 1T | 8T |
|---|---|---|
| kc=256/mc=144 | +0.5 [+0.2, +0.9] | −0.8 [−1.0, −0.1] |
| kc=512/mc=144 | +1.2 [+1.0, +1.3] | +2.1 [+1.9, +2.4] |
| kc=1024/mc=144 | +2.3 [+2.1, +2.5] | **+3.1** [+2.8, +3.4] |
| kc=1024/mc=288 (grid best) | +2.9 [+2.7, +3.1] | +3.8 [+3.6, +4.2] |

**The A/B on the published rows at n=4096**, N against O:

| row | 1T | 8T |
|---|---|---|
| Sgemm | +2.5% [+0.7, +5.3] | +3.5% [+3.0, +3.9] |
| Ssyrk | +2.1% [+0.9, +3.6] | +2.1% [+1.6, +2.6] |
| Ssymm | +1.9% [−0.2, +2.6] | +1.8% [+1.4, +2.5] |
| Strsm | −0.4% [−4.4, +3.5] | −0.1% [−0.7, +0.5] |

1. ✓ **Quieter.** The widest 1T half-interval is 0.43%, at kc=256/mc=72, against ±8–10% on the
   c8i.4xlarge.
2. ✓ **The kc=256 lead was noise.** At 8T, kc=256/mc=144 reads −0.8% against shipped, not +6.1%.
3. ✓ **The deeper KC is positive.** kc=1024/mc=144 reads +3.1% at 8T. The surface is monotone in KC
   at mc=144 and still rising at the grid's edge, as on Skylake-SP and SPR.
4. ✓ **No row regresses beyond its CI.** Sgemm, Ssyrk and Ssymm gain +1.8% to +3.5%, and Strsm is
   flat.
5. **Not run:** there is one L3 domain.

**Not refuted. KCDeep stays on GNR.**

A side reading: the same code runs about 16% faster here than on the c8i.4xlarge. 4x32 1T at
kc=384/mc=144 read 105.9 here against 89.0 there, and 8T 690.2 against 595.7. This is one instance
each, so it is a reading, not an attribution. It does say the 4xlarge sweep's absolute rates were
not this part's.
