# Judged amd64 P2 on the family-max ceiling and the roofline anchor: GREEN (#199, ab83784)

Hosts: c5n.18xlarge (8124M), c7a.48xlarge (9R14) and c8a.48xlarge (9R45), on-demand us-east-1,
at both elements. 31m21s, about $12.6, fleet verified down. Predictions are in `prereg.md`,
committed with the driver before launch.

| host | element | class | best shape | % of peak, net of CI | criterion | verdict |
|---|---|---|---|---|---|---|
| skx | f64 | issue (spread 1.014×) | 2x16 | 42.4% | 96.8% of a 43.8% roofline (≥ 90%) | PASS |
| skx | f32 | issue (spread 1.014×) | 2x32 | 42.4% | 96.8% of a 43.8% roofline | PASS |
| Zen 4 | f64 / f32 | FMA | 4x16 / 4x32 | 96.8% / 96.4% | ≥ 55% | PASS |
| Zen 5 | f64 / f32 | FMA, via the falsification clause | 4x16 / 4x32 | 64.8% / 64.8% | ≥ 55% | PASS |

Tally: f64 29 PASS / 0 FAIL / 0 UNMEASURED; f32 30 / 0 / 0. **gate-p2: GREEN, twice.**

**Every prediction held.** On Zen 5 the ceiling-spread interval straddled 1.10 ([1.073, 1.113]),
but 4x16/4x32 retired 164–167% above the ceiling those mixes set, so the class was decided
either way. On Zen 5, `shared` (100.6%) and x44 (100.5%) tied the canonical row inside the 1%
tolerance.

**Against `a4a2ca1`, the last judged amd64 P2 (f64):** Zen 4 96.5% → 96.8% and Zen 5 65.2% →
64.8%, whose denominators did not move. skx went from 93.5% of its old roofline to 96.8% of the
anchored one, against a peak 12.5% higher. Its kernel rate is unchanged; the denominator and the
anchor moved.
