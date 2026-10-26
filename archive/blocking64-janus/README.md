<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# float64 block-size sweep, janus (#191)

The test binary was built at `2a41213`, the commit that added the sweep and its selection rule
(`internal/block64/blocking64_bench_test.go`). It ran on janus.local (i9-9960X, Skylake-X)
in the `measured` pueue slot, idle (load 0.00), one thread, `-benchtime 3x`, 3 interleaved
passes, 600 readings. Dev tier, unjudged. `select.py` applies the pre-registered rule to
`sweep-2a41213-janus.log`.

| | KernelD2x16 | KernelD4x16 |
|---|---|---|
| incumbent kc=384/mc=144/nc=2048 | 36.38 [36.35, 36.39] | 33.68 [33.64, 33.72] |
| **kc=512/mc=144/nc=2048** (selected) | **38.69** | **34.82** |

13 grid points cleared the rule (a win on both kernels with disjoint ranges); kc=512/mc=144
has the best geometric mean. **512 is the top of the grid**, so a larger KC is untested.
NC 2048 is the same point as the incumbent's 4096 at n=2048, so only KC moves.

**MB stays 64.** No value beat it on both kernels and both sides; 48 won only on 4x16's
right side (20.63 against 20.13).
