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
