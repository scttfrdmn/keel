# Judged arm64 P2 against the family-max NEON denominator (#198, ba29105)

The shipped NEON kernels, unchanged since `c917b05`/`5d3f023`, re-judged against the
corrected denominator (DESIGN.md P2, 2026-10-27). Hosts: c7g.16xlarge (Neoverse-V1),
c8g.48xlarge (V2) and c9g.48xlarge (V3), on-demand us-east-1, all admitted evidentiary.
It took 35m37s, about $11.1 at $18.70/hr, and the fleet was verified down. The log is
`driver-p2-arm64-family-judged-ba29105.log`, and the predictions are in `prereg.md`,
committed with the driver before launch.

| host | element | `neon` peak, GFLOP/s | family x44 / shared | best shipped shape | % of peak, net of CI | class | verdict |
|---|---|---|---|---|---|---|---|
| V1 | f64 | 39.07 | 53.0% / 53.0% | 3x12 | 45.6% | FMA | FAIL |
| V2 | f64 | 42.14 | 93.9% / 72.3% | 3x12 | 45.5% | FMA | FAIL |
| V3 | f64 | 51.6 | 95.9% / 73.6% | 3x12 | 46.0% | FMA | FAIL |
| V1 | f32 | 78.16 | 53.0% / 53.0% | 3x24 | 45.5% | FMA | FAIL |
| V2 | f32 | 84.43 | 94.3% / 72.1% | 3x24 | 45.5% | FMA | FAIL |
| V3 | f32 | 103.2 | 94.0% / 72.3% | 3x24 | 46.2% | FMA | FAIL |

Tally: 24 PASS / 4 FAIL / 0 UNMEASURED per element. The four FAILs are the three per-host
floors and their aggregate. Spill audit, BCE, the peak's memory-free and insns/FMA checks,
and the family-maximality check all pass.

**Every pre-registered prediction held:** RED at both elements on all three hosts,
FMA-bound everywhere, best fractions 45.5–46.4% (band 40–52%), and the family check
passing.

**This supersedes the arm64 P2 greens** of `archive/p2-arm64/` (V1 86.1/85.8%, V2
62.6/61.7%) and `archive/p2-arm64-c9g/` (V3 62.7/64.0%). Their kernel rates reproduce
here to within 0.3% (old → new GFLOP/s: V1 3x24 35.55 → 35.59, V2 3x24 38.47 → 38.47,
V1 3x12 17.83 → 17.83, V3 3x12 23.81 → 23.84, V3 3x24 47.95 → 47.84). The verdicts flip because their denominator, `neonPeak`,
read 1.26–1.89× below a loop that beats it.

**Next** (Scott's ruling on #198, decision 2): the kernel-shaping step is pre-broadcast A
packing, then this gate again. The STOP rule binds if it is still RED after that step and
one tile shrink.
