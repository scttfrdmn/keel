<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Moving Level 2 into internal/l2: does float32 Level 2 move? (#192, before the run)

Scott's 2026-10-29 ruling on #192 moves the float32 Level-2 bodies out of package keel into
`internal/l2`, so `internal/l264` can be generated from them. The bodies are moved, not rewritten.
The one code change is that each body calls the Level-1 kernels through an `l1.Kernels` parameter
instead of the package-level `activeL1`, which costs a few loads per call, against O(m·n) work. So the
float32 object code changes, and unlike step 2 of stage 1 an identity witness is not available.
This A/B is the witness.

**Arms.** `before` is `109b274` and `after` is the refactor commit. Both are `go test -c -trimpath`
from the same checkout path. **Host:** orion.local (M4 Pro, NEON), its `measured` pueue slot,
where `l2Grain` was measured. **Rows:** `BenchmarkL2Par` for SgemvN, SgemvT, Sger, Ssyr, Ssyr2,
StrmvN and StrmvT at n = 1024 and 4096, `-cpu 1,8`. **Design:** 6 rounds, alternating which arm
runs first, `-count 1` per arm per round.

**Prediction:** no row's after/before ratio of medians lies outside **[0.98, 1.02]**.
**Refuted if** any row is outside that band AND its two arms' 6-draw ranges are disjoint.

**What this cannot see (§5 rule 12):** Ssymv, Strsv and the 8 banded and packed routines have no
benchmark, so 9 of the 16 moved routines are unmeasured here. Their change is the same one-parameter
mechanism. The run measures 7 of 16.

## Result (orion.local `measured`, task 49, 2026-10-29 19:59–20:08 PDT, 0 co-tenants at submit)

`before.test` sha256 `62ef326f…`, `after.test` `aace2ffb…`. 336 lines = 28 rows × 2 arms × 6 rounds,
all present (`ab-orion-109b274-vs-eefa858.log`). **The prediction held as written:** no row is both
outside [0.98, 1.02] and range-disjoint. 27 of 28 ratios of medians lie in **0.9856–1.0197**.

**The one row outside the band, stated rather than ruled away by the rule:** `Ssyr/m=4096/n=4096`
at 8 threads, ratio 1.110 with overlapping ranges. Its draws are **bimodal in both arms**. One mode is
near 260K ns and one is at 318–380K ns:

| round | 1 | 2 | 3 | 4 | 5 | 6 |
|---|---|---|---|---|---|---|
| before | 260629 | 259615 | 259872 | 261645 | 318712 | 371239 |
| after | 260931 | 262284 | **317508** | 261716 | 337306 | 379496 |

The arms agree within rounds 1, 2 and 4, and both are slow in 5 and 6. A median over a non-unimodal
arm is not a level, and this one's 1.110 is which mode the six draws happened to hit. The paired
within-round ratios are 1.001, 1.010, 1.222, 1.000, 1.058 and 1.022; round 3's `after` is the only
lone outlier. That is consistent with host state at 8 threads on a 10P+4E part, not with a
treatment that costs a few loads per call. No other 8-thread row shows it.

**Scope, restated:** 7 of the 16 moved routines are measured, at 2 sizes × 2 widths. Ssymv, Strsv
and the banded/packed routines are unmeasured; their change is the same one-parameter mechanism.
