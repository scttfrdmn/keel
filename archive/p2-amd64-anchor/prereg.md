# Judged amd64 P2 on the family-max AVX-512 ceiling and the roofline anchor: predictions (#199)

Committed with the frozen driver, before launch. The fleet is the judged amd64 one:
c5n.18xlarge (Skylake-SP 8124M), c7a.48xlarge (Zen 4 9R14) and c8a.48xlarge (Zen 5 9R45),
on-demand us-east-1, at both elements. What changed since the last judged amd64 P2
(`a4a2ca1` at f64): the canonical `avx512` peak is the family max, x44 (DESIGN.md P2,
2026-10-28), and the roofline's anchor is the old peak loop at its measured fraction, not
the canonical peak (Scott's ruling, the same day).

**Basis.** The same gate on the exploration tier, twice: `c98cd9f` (the canonical swap
alone, where skx reclassified FMA-bound at 42.5%), and `403aceb` (with the anchor,
`archive/p2-amd64-family/` and `build/p2amd2-403aceb.log`), where skx read issue-bound at
spread 1.012× and 2x32 attained 97.1% of a 43.8% roofline, computed through the gate's own
functions.

**Predictions, per host and element:**
- **skx: issue-bound.** Ceiling spread at most 1.05×. Best shape (2x32, or 2x16 at f64) at
  **38–47% of the new peak**, and **90–100% of its roofline, so PASS**;
- **Zen 4: FMA-bound**, best shape **93–99% of peak, PASS**;
- **Zen 5: FMA-bound**, reached through the falsification clause (its mixes converge but 4x32
  retires above their ceiling), best shape **60–70% of peak, PASS**;
- on every host: the family check passes (x44 within 1% of the `avx512` row), the anchor is
  measured, and the anchor and canonical-peak IPF ceilings (2.25, 1.1875) pass;
- **verdict: GREEN at both elements on all three hosts.**

**REFUTED if** any host/element is RED or UNMEASURED, skx is not issue-bound, a class
differs from the above, or a fraction falls outside its band. A RED invokes the STOP rule as
written.
