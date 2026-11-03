<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Sapphire Rapids: is there room for the deep KC beyond 1024? Predictions (#26, #142)

**Why.** Every AVX-512 host blocks full-height float32 GEMMs at `avx512KC = 1024`. On three parts
the KC surface was still rising at 1024, the edge of every grid so far:
- Sapphire Rapids (`archive/blocking-fma-sweep/`): kc=512 +6.1%, kc=768 +8.8% and kc=1024 +11.9%
  at 8T against kc=384, at n=2048;
- Granite Rapids (`archive/gnr-quiet/`);
- Skylake-SP.

SPR has the largest lever, so the question is asked there. Scott, 2026-11-03: "move to Sapphire
Rapids". The sweep's n=2048 caps KC at k, so this run measures the published n=4096 rows directly.

**Run.** One full-size c7i.48xlarge, masked to one thread per physical core of socket 0, stock
go1.27.3. There are four arms of this tree, differing only in `avx512KC`: 1024 (shipped), 1536, 2048
and 3072. The benchmark is BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at n=4096, at 1T and 8T, with
4 seeds × `-test.count=5` and the arm order rotated. The Level-3 suite runs first at 3072, with its
exit status.

The working sets at the largest KC are disclosed. The A block, MC × KC × 4 B, is 1.18 MB at 2048 and
1.77 MB at 3072, inside SPR's 2 MB L2 at both. The B panel is KC × NC × 4 B with NC = 4096: 32 MB at
2048 and 48 MB at 3072, inside the 105 MB L3 at both.

**Predictions** (each candidate against 1024, net of both CIs):
1. **Correctness:** rc=0 at 3072.
2. **Diminishing returns:** 1536 beats 1024 by +1% to +4% on Sgemm, Ssyrk and Ssymm at 8T.
3. **A peak:** the best of 1536, 2048 and 3072 at 8T sits at 1536 or 2048, and 3072 is no better
   than the best of the others.
4. **Strsm** is within ±2% for every candidate, because its thin rank updates keep KC=384.

**What this run can and cannot decide.** `avx512KC` applies to every AVX-512 host, so an SPR win does
not ship from SPR alone. If some candidate beats 1024 at 8T net of CI on all three of Sgemm, Ssyrk
and Ssymm, with no row regressing beyond its CI, the next step is the same candidate on Zen 5, Zen 4,
GNR and Skylake-SP. That would use both placements where a part has several L3 domains. If no
candidate wins, 1024 is the measured optimum on SPR's published rows, and the open "still rising"
caveat in `dispatch.go` closes for SPR.
