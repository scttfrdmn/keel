# gate-p2 amd64 on the family-max AVX-512 ceiling: exploration (#199, c98cd9f)

The predictions are in `prereg.md`, committed before the run. Hosts: c5n.4xlarge (8124M),
c7a.2xlarge (9R14) and c8a.2xlarge (9R45), exploration tier (correctness-class, so nothing
here is a verdict). The fleet was verified down. 24 PASS / 0 FAIL at each element.

| host | class, was → now | best shape f32 / f64, net of CI |
|---|---|---|
| skx | issue → **FMA** (ceiling-mix spread 1.705×) | 2x32 **42.5%** / 2x16 **42.4%** |
| Zen 4 | FMA → FMA | 4x32 96.4% / 4x16 96.8% |
| Zen 5 | FMA → FMA | 4x32 65.1% / 4x16 65.5% |

**Every prediction held.** The family check passed on every host (x44 the maximum). The
Zen fractions moved by less than 1 point from `a4a2ca1`. skx reclassified FMA-bound and fell
under the flat 55%.

**What the skx mixes show.** As ceiling mixes, p = fraction of peak × insns/FMA. f32:
4x32 0.323 × 6.25 = 2.02; shared 0.889 × 2.25 = 2.00; fo 0.528 × 3.75 = 1.98; 2x32 0.425 ×
4.625 = 1.97; and **x44 1.0 × 1.1875 = 1.19**. Every loop but x44 converges within 3% of the
same retirement rate, the front end's ceiling, which is what classified skx issue-bound
before #199. x44 sits alone below it because it is the one loop that is not issue-limited:
more FMAs per cycle from fewer instructions. The classifier counts the canonical peak as a
ceiling mix, a rule written when the peak sat at the issue ceiling. With an FMA-bound
canonical peak, it reads divergence. Reading it as issue-bound with x44 out of the
convergence set, 2x32 attains 1.97/2.02 ≈ 97% of its roofline.

**This is a classifier question, put to Scott on #199.** No gate was changed here.
