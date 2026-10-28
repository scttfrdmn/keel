# Judged arm64 P2 on the pre-broadcast-A kernels: predictions (#198)

Committed with the frozen driver, before launch. This is the shaping step's verdict.
The shipped broadcast kernels were RED against the family-max denominator at
`ba29105` (45.5–46.4%, `../p2-arm64-family/`). The pre-broadcast-A kernels were then
promoted by two pre-registered rules (`../neon-pb/`), and gate-p2's arm64 arms were
amended to include them (DESIGN.md P2, 2026-10-27). Hosts: c7g.16xlarge (V1),
c8g.48xlarge (V2) and c9g.48xlarge (V3), on-demand us-east-1, at both elements.

**Basis.** The same gate at the exploration tier (`ddf6e2d`,
`exploration-graviton-4xlarge-ddf6e2d.log`, 4xlarge sizes) read the best shape, net of CI:

| host | float32 5x16-pb | float64 5x8-pb |
|---|---|---|
| V1 | 86.2% | 86.2% |
| V2 | 81.3% | 81.3% |
| V3 | 81.4% | 81.3% |

That run had 24 PASS / 0 FAIL per element. Every host was FMA-bound and passed the family
check. The previous judged run reproduced the exploration-tier rates to within 0.3%.

**Predictions, per host and element:**
- best shape is the pre-broadcast one (5x16-pb at float32, 5x8-pb at float64), at
  **76–91% of peak, net of CI**;
- class **FMA-bound**;
- the family check passes, and the spill audit, BCE and peak-ceiling checks pass;
- `SWEEP_BEST_IPF` = 1.556 reconciles;
- **verdict: GREEN at both elements on all three hosts.**

**REFUTED if** any host/element is RED or issue-bound, a best fraction falls outside
76–91%, the best shape is not the pre-broadcast one, or the family check fails. A RED
after this step invokes the STOP rule as DESIGN.md states it: one tile shrink, then
`docs/spill-report.md`, a `blocked` issue, and a decision.
