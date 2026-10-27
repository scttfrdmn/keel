<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Pre-registration: the judged arm64 P2 run on Graviton5 (c9g.48xlarge), both elements (#191, 2026-10-27)

Written and committed before launch, with the commit that admits c9g.48xlarge to the
evidentiary class (`scripts/remote.sh`, Scott's go). Fleet: c9g.48xlarge (keel-gvt5),
on-demand, us-east-1, $8.35/hr. It runs `KEEL_ARCH=arm64 scripts/gate-p2.sh` at f64, then f32.

## The exploration pre-check this follows
c9g.4xlarge at `c917b05` (`archive/p2-arm64/exploration-c9g-4xlarge-c917b05.log`).
- **Read-back:** MIDR part 0xd84, `lscpu` Model name Neoverse-V3, 16 cores, 1 thread/core,
  SVE2.
- **Gate:** 20 PASS / 0 FAIL / 1 UNMEASURED at each element. The UNMEASURED is the
  correctness-class floor, as designed.
- **Readings, reported not judged:** float64 3x12 at 63.7% of peak (62.4% net of CI), and
  float32 3x24 at 63.2% (62.7% net). Both FMA-bound (ceiling spreads 1.19x and 1.24x).

## Predictions
| element | class | band (best shipped shape) | floor |
|---|---|---|---|
| float64 | FMA-bound | 56–71% of peak | ≥55% |
| float32 | FMA-bound | 56–71% of peak | ≥55% |

**Predicted verdict: GREEN at both elements**, with about a 7-point margin, the thinnest on
the arm64 fleet so far. The same per-cycle reading as Neoverse-V2 predicts this: the kernels
retire about 7–8 instructions per cycle, while V3's FMA peak per cycle is about 2.85 (V1: 2.0).
**REFUTED if** the class differs, or a fraction falls outside its band. A RED invokes the
STOP rule verbatim, for the float32 NEON kernels as well.
