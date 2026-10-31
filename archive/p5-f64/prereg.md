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

## Exploration pass: result (e3bf0b2)

The logs are `exploration-{amd64,arm64}-e3bf0b2.log`, each with its gate-p4 and gate-p3 sub-logs.
- **amd64:** 55 PASS / 1 FAIL / 4 UNMEASURED / 3 BASELINE / 3 REPORTED, 2h30m (about $4.30).
- **arm64:** 50 PASS / 0 FAIL / 4 UNMEASURED / 3 BASELINE / 0 REPORTED, 3h04m (about $6.05).

Each fleet was verified down by a readable `spawn list`. **Seven predictions held and three
missed. All three misses are mine: I anchored on judged-size float32 values when the same
instance sizes' float32 explorations were in the record.**

1. ✓ Era **f64init** on the preflight and the era line, on both architectures.
2. ✓ Dispatch: `2x16/avx512` on skx, `4x16/avx512` on Zen 4 and Zen 5, `5x8-pb/neon` on V1, V2 and V3.
3. ✓ Every Level3F64, WideRoof64 and Level1F64 row on every host.
4. ✗ **skx:** 91.82 against 104.5 (−12%). Every other host was within 4%. c5n.4xlarge read −12% at
   float32 too (183.7 against 208.6, `archive/p5-amd64-famaxx86/exploration-d14cee0.log`), so this
   is the instance size, not float64.
5. ✗ **V1:** the 8-thread ceiling reads 67.8% of 8×1T. Every other host is in band (Zen 5 99.1%,
   Zen 4, skx, V3 99.8%, V2 88.4%). c7g.4xlarge read 68.6% at float32 (`archive/p5-arm64-famax/`).
   Also wrong: skx sits at 99.8%, not at the low end; float32's 0.88 was the 18xlarge's.
6. ✗ **V1 and V2 shares:** Dgemm reads 48.9% and 50.3%, against the judged float32 64.7% and 64.9%.
   At the same 4xlarge size float32 read 48.2% and 50.4%. **Same instance size, float64 against
   float32, the 8-thread GEMM share agrees within 2 points on all six CPUs:** Zen 5 49.3→48.9,
   Zen 4 77.8→77.3, skx 29.2→29.5, V3 61.7→59.8, V2 50.4→50.3, V1 48.2→48.9.
7. ✓ arm64 OpenBLAS is 0.3.34, with corename `neoversev1` on V1 and `neoversev2` on V2 and V3. V2's
   reference is pinned to `armv8` at +0.2%. amd64 is 0.3.26: `Cooperlake` on Zen 4 and Zen 5,
   `SkylakeX` on skx. keel/OpenBLAS at float64 is Zen 5 0.80, Zen 4 0.97 and skx 75.6% amended
   (issue-capped; plain 38.2%). All three clear 60%, though none is judged, by admission. arm64:
   V1 0.78, V2 0.87, V3 0.93, REPORTED.
8. ✓ `-race` is clean on all six hosts.
9. ✓ No FAIL that is not carried, and no unexpected UNMEASURED. The enumeration was imperfect in
   both directions, though:
   - amd64's gate-p3 kernel row is a fixed P2 floor (PASS: Zen 5 65.7%, Zen 4 96.9%, skx 96.8% of
     roofline), not a BASELINE;
   - arm64 has no admission UNMEASURED, because REPORTED replaces the aggregate there;
   - amd64's 3 REPORTED lines are the sub-tier L1 rows that float32 also reports.
   **Defect found:** arm64 gate-p3's fleet line says "covers 0 of 3 (the UNMEASURED lines above say
   which)", but the per-host withholds are `info` lines and no UNMEASURED line exists. Recorded on
   #193, not fixed in this campaign (no shell spend).
10. ✓ Both teardowns verified.

## Judged r1, the founding run: predictions (written from the exploration, before r1)

r1 founds f64init on the judged fleets:
- amd64: c5n.18xlarge, c7a.48xlarge and c8a.48xlarge;
- arm64: c7g.16xlarge, c8g.48xlarge and c9g.48xlarge.

The driver was re-frozen with a 5h judged TTL.

Nothing is judged against a float64 bar. Predicted:
- **Era f64init.** No `CEIL_FRACTION` is in force. The L1 rows render BASELINE, and so does arm64
  gate-p3's kernel row.
- **gate-p4 and gate-p3 underneath are GREEN on amd64**: the kernel sentinels and the OpenBLAS
  ratios above clear their floors, and the hosts are admitted. **gate-p4 is GREEN on arm64**, with
  gate-p3's only non-PASS lines being BASELINE and REPORTED. `-race` is clean everywhere.
- **1-thread FMARoof64 within 2%** of #193's table: Zen 5 143.3, skx 104.5, Zen 4 58.55, V3 51.58,
  V2 42.16, V1 39.06.
- **The 8-thread ceiling's ratio to 8×1T within 3 points** of float32's judged ratio on the same
  host: Zen 5 100.0%, Zen 4 99.7%, skx 93.5%, V3 99.8%, V2 99.9%, V1 89.2%.
- **8-thread Dgemm share of 8×1T within ±3 points** of float32's judged Sgemm: Zen 5 49.1, Zen 4
  74.9, skx 27.3, V3 69.8, V2 64.9, V1 64.6.
- **1-thread Dgemm within 6% of half** the judged 1-thread Sgemm: Zen 5 88.8, Zen 4 54.2, skx 34.4,
  V3 40.0, V2 32.7, V1 29.3. Exploration put Zen 5 and Zen 4 at 86.7 and 53.5.
- **Duration:** amd64 about 2h35m and arm64 about 3h10m, both inside 5h.
