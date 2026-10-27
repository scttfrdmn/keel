<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The judged arm64 P2 run on Graviton5 (c9g.48xlarge), both elements (#191) — GREEN

Rev `0f40781`: the pre-registration `ee5dac1` plus a driver fix that dropped a provisioning
step P2 never needed. The first launch stopped at it before any gate ran, because OpenBLAS
0.3.29 picks generic armv8sve on Neoverse-V3 (#196; `VOID-provision-refused-ee5dac1.log`).
Fleet: c9g.48xlarge (keel-gvt5, Neoverse-V3, MIDR 0xd84), on-demand, us-east-1,
2026-10-27 03:24:45Z–03:36:49Z (12m04s, about $1.7 at $8.35/hr). Teardown was verified.

**`KEEL_ARCH=arm64 scripts/gate-p2.sh`: float64 22/0/0 GREEN, float32 22/0/0 GREEN**
(`driver-p2-c9g-judged-0f40781.log`).

| element | class (predicted / measured) | best shipped shape | band | floor | held |
|---|---|---|---|---|---|
| float64 | FMA / FMA | 3x12 at 62.7% of peak (62.3% net of CI) | 56–71% | ≥55% | yes |
| float32 | FMA / FMA | 3x24 at 64.0% (63.1% net) | 56–71% | ≥55% | yes |

Both predictions held. Neoverse-V3 lands beside V2 (62.6%/61.7%), as the per-cycle reading
predicts: the kernels retire about 7–8 instructions per cycle on every Neoverse generation,
while the FMA peak per cycle grew from about 2.0 (V1) to about 2.9 (V2, V3).
