# gate-p2's amd64 arms on the family-max AVX-512 denominator: exploration predictions (#199)

This is the canonical swap (`avx512Peak` → x44, DESIGN.md P2, 2026-10-28) run through gate-p2
at both elements, on the exploration tier: c5n.4xlarge (Skylake-SP 8124M), c7a.2xlarge
(Zen 4 9R14) and c8a.2xlarge (Zen 5 9R45), before any judged spend. 4xlarge/2xlarge sizes
are correctness-class, so nothing here is a verdict. It exists to see what the classifier
does.

Basis: `archive/avx512-family/` (7b55f74). x44 read 112.5% of `avx512Peak` on skx, and
100.0% on Zen 4 and Zen 5.

**Predictions, per host and element:**
- the `avx512` row is the family max: the family check passes, x44 within 1% of the row,
  and shared below it on skx (about 89%) and tied on the Zens;
- **Zen 4 and Zen 5 read as before**: the same class, and best-shape fractions within 1
  point of `a4a2ca1`'s, because their denominator does not move;
- **skx reclassifies issue-bound → FMA-bound** at both elements. Its ceiling mixes lose the
  high-IPF peak anchor (p = 2.25 → 1.1875) and the kernel mixes scale by 1/1.125, so they
  stop converging within 10%. The best shape then reads about 1/1.125 of its old fraction of
  peak, which is **under the flat 55% floor**.

**REFUTED if** skx stays issue-bound, a Zen fraction moves more than 1 point, or the family
check fails anywhere. If skx does reclassify and sits under 55%, that is the honest
consequence of a corrected denominator, not a defect to be tuned away. What follows is the
same sequence as #198: a judged re-run that can confirm it, then the STOP rule's
documented-shaping question for amd64. Whether every family member should count as a ceiling
mix (DESIGN.md P2, 2026-10-28) goes to Scott with these numbers.
