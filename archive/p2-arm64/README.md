<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The judged arm64 P2 run, both elements (#191) — GREEN

Rev `7e36d00`: the pre-registration `763e1a7` plus a mode-only fix that made the driver
executable. The first launch exited 126 before any step ran and billed nothing. Fleet read
off the archives: c7g.16xlarge (keel-gvt3, Neoverse-V1) and c8g.48xlarge (keel-gvt4,
Neoverse-V2), on-demand, us-east-1, 2026-10-27 01:58:14Z–02:23:00Z (24m46s, about $4.3 at
$10.35/hr). Teardown was verified against `spawn list`.

**`KEEL_ARCH=arm64 scripts/gate-p2.sh`: float64 25/0/0 GREEN, float32 25/0/0 GREEN**
(`driver-p2-arm64-judged-7e36d00.log`). The exploration pre-check is
`exploration-gvt-4xlarge-a404442.log`.

| host | class (predicted / measured) | float64 3x12 | float32 3x24 | band | floor | held |
|---|---|---|---|---|---|---|
| keel-gvt3 | FMA / FMA | 86.1% of peak (86.0% net of CI) | 85.8% (85.8%) | 78–94% | ≥55% | yes |
| keel-gvt4 | FMA / FMA | 62.6% (60.9%) | 61.7% (61.4%) | 55–68% | ≥55% | yes |

Every prediction held. **The float32 NEON kernels' first P2-style verdict is GREEN.** They
had shipped through #136's sweep and gate-p3/p5 without one. As on amd64, float64 tracks
float32 within a point on each host, because the kernels audit identically and the peak is
half.
