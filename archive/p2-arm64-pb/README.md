# Judged arm64 P2 on the pre-broadcast-A kernels: GREEN (#198, c9e191f)

This is the shaping step's verdict. Hosts: c7g.16xlarge (Neoverse-V1), c8g.48xlarge (V2)
and c9g.48xlarge (V3), on-demand us-east-1, all admitted evidentiary. It took 42m13s,
about $13.2 at $18.70/hr, and the fleet was verified down. The log is
`driver-p2-arm64-pb-judged-c9e191f.log`, and the predictions are in `prereg.md`,
committed with the driver before launch.

| host | element | `neon` peak, GFLOP/s | best shape | GFLOP/s | % of peak, net of CI | class | verdict |
|---|---|---|---|---|---|---|---|
| V1 | f64 | 39.06 | 5x8-pb | 33.69 | 86.2% | FMA | PASS |
| V2 | f64 | 42.16 | 5x8-pb | 34.35 | 81.4% | FMA | PASS |
| V3 | f64 | 51.58 | 5x8-pb | 41.99 | 81.3% | FMA | PASS |
| V1 | f32 | 78.16 | 5x16-pb | 67.44 | 86.1% | FMA | PASS |
| V2 | f32 | 84.44 | 5x16-pb | 68.70 | 81.3% | FMA | PASS |
| V3 | f32 | 103.1 | 5x16-pb | 83.98 | 81.3% | FMA | PASS |

Tally: **28 PASS / 0 FAIL / 0 UNMEASURED per element; gate-p2: GREEN, twice.** The family
check passed on every host: fo 100.0%; x44 53.0% (V1) and 93.9–95.9% (V2/V3); shared
53.0% and 72.1–73.6%.

**Every pre-registered prediction held:** GREEN everywhere, best fractions 81.3–86.2%
(band 76–91%), the pre-broadcast shape best on every host, FMA-bound, and the family check
passing. The rates reproduce the exploration tier (`ddf6e2d`): V1 5x16-pb 67.41 → 67.44
GFLOP/s at 4xlarge → 16xlarge.

**Against the same denominator, before and after the shaping step** (`../p2-arm64-family/`,
`ba29105`): the best shipped shape went 45.5–46.4% → 81.3–86.2% of peak. The kernel did the
moving: 3x24 at 35.59 GFLOP/s → 5x16-pb at 67.44 on V1.
