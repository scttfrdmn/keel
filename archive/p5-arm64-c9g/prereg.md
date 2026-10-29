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

## Exploration pass: result (be72f31, `exploration-be72f31.log`)

34 PASS / 0 FAIL / 1 UNMEASURED / 2 BASELINE. 57m, about $0.70, and the fleet was verified down with a readable
inventory. **Every prediction held:**
- `corename=neoversev2` on OpenBLAS 0.3.34 ✓;
- era famax ✓, and the dispatch marker reads `5x16-pb/neon` ✓;
- all 8 routine rows were produced, 10 draws each ✓. The one UNMEASURED is the correctness-class admission ✓.
  Gate-p4 is GREEN underneath (61/0/0), and gate-p3 has 0 FAIL;
- at 8 threads, **fo is the family max**: 823.35 GFLOP/s against x44 786.65 (95.5%) and shared 601.3 ✓;
- keel/OpenBLAS = 77.67 / 86.59 = **0.897** at 2048³ ✓ (band 0.85–0.95). It is REPORTED, and the line counts
  0 judged hosts only because a correctness-class host has no judgeable reading.
The two BASELINEs are V3's first sight in famax: 24 L1 rows, and no README row for Neoverse-V3. gate-p3's
`peak/5x16-pb/neon/kc=128` reads 81.3% of peak, matching PB P2's 81.3% net.

## Judged r1: predictions (written from the exploration, before r1)

**r1 is V3's founding run in famax**, on c9g.48xlarge ($8.35/hr). Predicted:
- era famax, corename `neoversev2`, dispatch `5x16-pb/neon`;
- **0 FAIL**: V3 has no README rows, so criterion 9 renders BASELINE and not FAIL. The share, L1 and peak rows
  all render BASELINE. The headline is UNMEASURED because the one host is BASELINE (famaxx86's r1 rendering);
- gate-p4 and gate-p3 underneath **GREEN**, and `-race` clean;
- fo the 8-thread family max within 1%;
- 1-thread Sgemm (n=4096) within 8% of the exploration's 80.1: **73.7–86.5 GFLOP/s**. gate-p3's keel/OpenBLAS
  REPORTED on 1 host at **0.85–0.95**, and its peak row at 80–83% of peak.
**REFUTED if** any FAIL appears, gate-p3/p4 is RED, the corename is not `neoversev2`, or fo loses the family max.

## r1: result (f3bbe01, `judged-r1-f3bbe01.log`)

34 PASS / **0 FAIL** / 1 UNMEASURED / 6 BASELINE. It took 1h01m (about $8.50), and the fleet was verified down
with a readable inventory. **Every prediction held:** corename `neoversev2` (MAX_THREADS=192) ✓; dispatch
`5x16-pb/neon` ✓; gate-p4 GREEN (61/0/0) and gate-p3 24/0/0 ✓; `-race` clean ✓; fo the 8-thread family max at 824.05
against x44 785.15 (95.3%) ✓; Sgemm 1T **80.07** (band 73.7–86.5) ✓; keel/OpenBLAS 77.18 / 86.75 = **0.890** ✓;
gate-p3 peak row 81.2% ✓. The share rows (net of both intervals) are Sgemm 69.8, Ssyrk 73.0, Ssymm 69.5 and
**Strsm 47.4**. The 8-thread ceiling is 824.3 GFLOP/s. The rates reproduce the 4xlarge exploration to about 0.1%.

## r2: predictions (written from r1, before r2)

r2 is the second draw. Nothing is judged against a bar until the rows land over r1 + r2. Predicted:
- the same tally, **34/0/1/6**, with gate-p3/p4 GREEN and `-race` clean;
- Sgemm 1T within 1% of 80.07, and the ceiling within 0.5% of 824.3;
- **all 4 share rows and the peak row admissible**, each cross-draw spread under 2%;
- **at most 2 of the 24 L1 rows refused** at the 5% band. famax's V1/V2 pair refused none (widest 3.52%); ceil8
  refused V1 Sdot at 7.19%.
**REFUTED if** any FAIL appears, a share or peak row spreads past 5%, or more than 2 L1 rows are refused.

## r2: result (e030cc1, `judged-r2-e030cc1.log`)

34/0/1/6, the same tally. It took 59m (about $8.20), and the fleet was verified down with a readable inventory.
**REFUTED, on its L1 clause:** I predicted at most 2 of 24 L1 rows refused, and 4 are:
`rate/Saxpy/n=65536` 11.23 then 13.09 (**14.21%**), `rate/Saxpy/n=256` 7.12%, `rate/Sasum/n=65536` 5.49%, and
`rate/Sasum/n=4096` 5.39%. Everything else held: all 4 share rows within 0.55% (Sgemm 69.8/69.9, Ssyrk 73.0/72.6,
Ssymm 69.5/69.7, Strsm 47.4/47.5), the peak row 81.2/81.3, Sgemm 1T 80.005 (r1 80.07), the ceiling 824.05,
gate-p3/p4 GREEN, and `-race` clean.

**Landed over N=2:** **25 rows**: 4 share, 20 L1 and `peak/5x16-pb/neon/kc=128` at 81.25. All drive
`registered`, the 4 refused rows drive `new`, and a V2 positive control drives `owing`. **V3's witness row is HELD**,
by the famaxx86 precedent: these runs judged nothing, and a landed witness would make the 4 refused rows an
`owing` red that is not a regression. A third draw settles them. **README:** 9 Neoverse-V3 rows, pooled over r1 + r2.
Sgemm 1T is 80.06 at 77.6% of a 103.2 GFLOP/s peak, and the 8-thread ceiling is 824.2.
