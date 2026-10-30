<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# gate-p5 at float64 in era f64init: predictions (#193)

These runs found the float64 performance gates. Nothing in this campaign is a verdict against a
float64 bar, because none exists yet: era `f64init` carries no `CEIL_FRACTION`, so every share
renders REPORTED/BASELINE. The founding run measures each host's 8-thread float64 ceiling with
gate-p5's own placement mask. That is the construction that founded float32's bars (#193's phase-
start comment). The bars are then typed from it, and a confirmation run lands the baselines over
N=2. The driver is `driver-frozen.sh ARCH TIER TAG`. It exports `KEEL_ELEM=f64` and `KEEL_GOARCH`,
and runs the two architectures one after the other.

**From the record.**
- 1-thread float64 peak, judged P2 (#193): Zen 5 143.3, Skylake-SP 104.5, Zen 4 58.55, V3 51.58,
  V2 42.16 and V1 39.06 GFLOP/s, each half of float32 to within 0.2%.
- Float64 P2: skx `2x16/avx512` at 93.5% of its issue roofline, Zen 4 `4x16` at 96.5% of peak and
  Zen 5 `4x16` at 65.2% (`archive/p2-f64-amd64/`). V1/V2/V3 `5x8-pb/neon` at 86.2%, 81.4% and 81.3%
  (`archive/p2-arm64-pb/`).
- The lab smoke at `3c71db5`: janus green underneath. pollux (GB10, correctness class) had no FAIL
  of its own: an 8-thread WideRoof64 of 177.3 at 99.9% of 8× its 1-thread reading, and Dgemm,
  Dsyrk, Dsymm and Dtrsm 8-thread rows at 82.5%, 87.7%, 82.2% and 65.1% of 8× the 1-thread peak.

## Exploration pass (TIER=explore), predictions

amd64 runs on c5n.4xlarge, c7a.2xlarge and c8a.2xlarge ($1.71/hr). arm64 runs on c7g.4xlarge,
c8g.4xlarge and c9g.4xlarge ($1.97/hr). These sizes are correctness class. The pass witnesses the
whole float64 path before about $43/hr of judged fleets is spent.

1. **Era and element.** The preflight, the banner and gate-p5's era line all read **f64init**,
   provisional, from `measurement-eras-f64.tsv`. A float32 era (`famaxx86`, `famax`) anywhere
   refutes the wiring.
2. **Dispatch.** `keel-bench-kern64` reads **`2x16/avx512`** on skx, **`4x16/avx512`** on Zen 4 and
   Zen 5, and **`5x8-pb/neon`** on V1, V2 and V3.
3. **Rows.** Every `Level3F64` row (Dgemm, Dsyrk, Dsymm, Dtrsm × 1 and 8 threads, 10 draws each),
   every `WideRoof64` ceiling and family row at 1 and 8 threads, and all 24 `Level1F64` rows are
   produced on every host.
4. **1-thread FMARoof64** is within 10% of the record above on each CPU. These are the same parts on
   smaller instances.
5. **8-thread ceiling.** `WideRoof64` at 8 threads reads **0.85–1.00 × 8 ×** its own 1-thread
   reading on every host. skx sits at the low end: float32's x44 read 0.88 there.
6. **Shares track float32.** Each 8-thread Dgemm row's share of 8× the 1-thread FMARoof64 is within
   **±8 points** of the README's float32 Sgemm share for the same CPU: Zen 5 49.0%, Zen 4 74.9%,
   skx 27.3%, V2 64.9% and V1 64.7%. V3 has no published float32 row yet, so it gets no prediction.
   This is the weakest prediction, because the README rows come from judged instance sizes. A
   miss is a finding about float64, not a verdict.
7. **OpenBLAS DGEMM.** `provision-openblas.sh` builds v0.3.34 on arm64, and gate-p3's corename
   read-back reads **`neoversev1`** on V1 and **`neoversev2`** on V2 and V3. On amd64 it reads the
   corename the float32 gate-p3 reads at the default version. arm64's keel/OpenBLAS ratio is
   REPORTED (NEON against an SVE reference). On amd64 the float64 ratio clears the shared 60%
   floor on all three hosts.
8. **`-race`** is clean on every host.
9. **Expected non-PASS lines, and only these:**
   - UNMEASURED: the correctness-class admission (gate-p5's headline, and gate-p3's under it);
   - UNMEASURED: README criterion 9, guarded at float64;
   - BASELINE: first sight in f64init, i.e. the 24 L1 rows per host and gate-p3's kernel row;
   - FAIL: only the carried REDs that these produce (gate-p3 → gate-p4 → gate-p5).
   Any other FAIL or UNMEASURED is a finding.
10. **Teardown.** Each fleet is verified down by a *readable* `spawn list`.
