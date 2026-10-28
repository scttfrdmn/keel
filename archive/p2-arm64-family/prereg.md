# Judged arm64 P2 against the family-max NEON denominator: predictions (#198)

Written and committed with the gate change (DESIGN.md P2, "The NEON denominator is the
maximum over a family of FMA-only loops", 2026-10-27) and before launch. It re-judges the
**shipped** NEON kernels, unchanged since `c917b05`/`5d3f023` greened them against
`neonPeak`, on the judged Graviton fleet: c7g.16xlarge (V1), c8g.48xlarge (V2) and
c9g.48xlarge (V3), on-demand in us-east-1, at both elements.

**Basis.** The exploration-tier check `100a055` (`exploration-graviton-4xlarge-100a055.log`, archived
beside this file) ran this exact instrument on 4xlarge sizes of the same three families. There
the best shipped shape reached, as a percentage of the new `neon` row:

| host | float32 3x24/2 | float64 3x12/2 |
|---|---|---|
| V1 | 45.5% | 45.6% |
| V2 | 45.6% | 45.6% |
| V3 | 46.3% | 46.3% |

The `fo` member was the family maximum on every host and element: x44 at 53% (V1) and
94–96% (V2/V3), shared at 53% (V1) and 72–73% (V2/V3).

**Predictions, per host and element:**
- class: **FMA-bound**. Ceiling mixes {alternate shapes, peak at 1.333 insns/FMA} diverge
  well beyond the classifier's 10% (about 1.5× in the exploration numbers).
- best shipped shape (3x24 at float32, 3x12 at float64) at **40–52% of peak**;
- **the family check passes:** no member beats the `neon` row by more than 1% net of CI;
- spill audit, BCE, and the peak insns/FMA ceiling (1.333334) pass, as before;
- **verdict: RED at both elements**, on the 55% floor alone, on all three hosts.

**REFUTED if** any host/element is GREEN, its class is issue-bound, its best fraction falls
outside 40–52%, or the family check fails. In each case the denominator change has not done
what the exploration tier said it would, and that, not the kernels, is the finding.

**Consequence of the predicted RED.** It is the starting state, not the STOP condition. The
STOP rule binds after the documented kernel-shaping steps and one tile shrink. Under
Scott's 2026-10-27 ruling on #198 (decision 2), the shaping step is pre-broadcast A
packing, which reached 69–75% of the FMA-only loop in the probes.
