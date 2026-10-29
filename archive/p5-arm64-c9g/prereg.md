<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# gate-p5 on c9g (Neoverse-V3, Graviton5) in era famax: predictions (#196)

c9g gets its own campaign, which is the follow-up named on #196 once OpenBLAS v0.3.34 was shown to name
the core (`archive/openblas-c9g/`, corename `neoversev2`). V3 is not in `CEIL_DERIVED_FROM` (V1|V2), so it
is **registry-governed**. Its share rows render BASELINE on the founding and confirmation runs, and its
rows land over N=2 afterwards by `host-baselines.tsv`'s rules, as famaxx86's did. Nothing in either judged
run is a verdict against a bar. The product is the archive and the README rows it later supports. The
driver is `driver-frozen.sh`, which exports `KEEL_GOARCH=arm64` and `KEEL_OPENBLAS_VERSION=v0.3.34`.

**From the record.** PB P2 (`c9e191f`): `5x16-pb/neon` at 83.98 GFLOP/s, 81.5% of peak (81.3% net), FMA-bound,
the best shipped f32 shape. #196's probe (c9g.4xlarge, n=2048, 1 thread): keel Sgemm 77.19, OpenBLAS 0.3.34
`neoversev2` 86.41 GFLOP/s.

## Exploration pass (TIER=explore, c9g.4xlarge, $0.70/hr), predictions

- provision builds v0.3.34, and gate-p3's read-back names **corename `neoversev2`**. `armv8sve` refutes it;
- era **famax**. The dispatch marker names **`5x16-pb/neon`**;
- every routine row is produced at 1 and 8 threads. The only UNMEASURED is the correctness-class admission;
- **fo is the 8-thread family max within 1%**. On V2, x44 came within 97%;
- gate-p3's keel/OpenBLAS ratio is REPORTED, not judged (NEON against an SVE reference), at **0.85–0.95** at 2048³.
