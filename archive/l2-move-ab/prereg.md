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
