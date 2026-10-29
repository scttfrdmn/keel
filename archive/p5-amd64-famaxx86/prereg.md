# gate-p5 on amd64 in era famaxx86: predictions (#199)

These runs produce the archive that re-derives the README's amd64 rows. They are needed because
#199 made the AVX-512 ceiling the family maximum (`avx512PeakX44`) and moved the roofline's pin to
the anchor. The amd64 era `famaxx86` was opened for this, scoped to amd64 under Scott's ruling
(a) on #199 (DESIGN.md §5 rule 17). arm64 stays in `famax`. The driver is `driver-frozen.sh`,
committed with this file. The amd64 kernels have not changed since ceil8. Only the denominator
has: `Peak/avx512`, `FMARoof64/avx512` and `Ceiling/compute` now run x44.

## Exploration pass (TIER=explore: c5n.4xlarge, c7a.2xlarge, c8a.2xlarge), predictions

This pass witnesses the whole path before the judged trio (about $24/hr) is spent. These sizes
are correctness-class, so nothing here is a verdict. Predicted:
- the gate resolves era **famaxx86** for amd64. A `ceil8` in an amd64 era line refutes the wiring;
- every routine row is produced at 1 and 8 threads. No criterion is UNMEASURED except for the
  correctness-class admission;
- the `Ceiling/family/*/threads=8` rows exist for x44, shared and fo, and **x44 is the maximum
  within 1%** on all three hosts;
- at 8 threads, x44 reads **10–15% above `shared`** (the old `avx512Peak`) on Skylake-SP, against
  #199's single-thread +12.5%, and within 2% of it on Zen 4 and Zen 5.

## Exploration attempt 1 (f2804ee): unmeasured, killed by a dev-host reboot

The run reached gate-p5's `-race` step on era **famaxx86** (that prediction held) and was killed
at 02:26Z when the dev host rebooted. It is `died`, with no verdict. Its teardown could not read
`spawn list` while the host was shutting down, and the verification then printed "fleet is down"
because grep found no `keel-` line. All three instances kept billing for about 2h40m (about
$4.60) until they were torn down by hand. The driver now treats an unreadable inventory as
UNKNOWN, and it was re-frozen for attempt 2. The predictions above are unchanged.

## Exploration attempt 2 (d14cee0): result

55 PASS / 4 FAIL / 1 UNMEASURED / 3 BASELINE / 3 REPORTED. It took 2h09m (about $3.70) and the
fleet was verified down by the new fail-closed check. **Every prediction held:**
- era **famaxx86** on the preflight, the banner and the era line ✓;
- all 8 routine rows (4 routines × 1 and 8 threads) produced on every host, 10 draws each ✓. The
  only UNMEASUREDs are the correctness-class admission: gate-p5's headline, and gate-p3's under it.
  Gate-p3 has 0 FAIL, and its RED is that admission, carried into gate-p4 as its one FAIL ✓;
- x44 is the 8-thread family max on all three hosts ✓. `Ceiling/compute/avx512` reads x44's
  value on each, so the canonical wiring holds;
- skx x44 / shared at 8 threads = 1466 / 1303 = **1.125** (band 1.10–1.15) ✓. Zen 4: 935.3 against
  shared 934.8 (+0.05%). Zen 5: 2282 against 2278.5 (+0.15%) ✓. fo trails on all three (773, 932,
  2058).
The other 3 FAILs are criterion 9: these are small instances checked against the README's
judged-size rows.

## Judged r1: predictions (written from the exploration, before r1)

**r1 is famaxx86's founding run** on c5n.18xlarge (skx), c7a.48xlarge (zen4) and c8a.48xlarge
(zen5). The kernels are unchanged since ceil8. Only the denominator moved. Predicted:
- era famaxx86. The share rows render as PASS lines reading "NO FRACTION IN FORCE" (§5 rule 17(d));
- gate-p4 and gate-p3 underneath it **GREEN**, `-race` clean on all three hosts;
- x44 is the 8-thread family max within 1% everywhere. skx `Ceiling/compute` =
  1391 × 1.125 ≈ **1565 ± 3%**, Zen 4 and Zen 5 within 1% of the README's 934.2 and 2293;
- skx shares fall by that factor: from ceil8's 31.2–32.0% to **27.0–29.5%** on Sgemm, Ssyrk and
  Ssymm. Zen shares move by less than 1 point from ceil8;
- 1-thread Sgemm within 3% of the README: **skx 64.3–68.3, zen4 104.0–110.4, zen5 168.7–179.1 GFLOP/s**;
- criterion 9 **FAILs on skx's `Ceiling/compute` row**, which is the changed instrument. Any other
  criterion-9 FAIL, on any host, is a finding about rate drift since ceil8, not about this era.
  Apart from criterion 9 and its aggregates, nothing FAILs.
**REFUTED if** gate-p3/p4 is RED, x44 loses its family max, skx's ceiling is outside 1518–1612,
or any FAIL appears outside criterion 9.

## r1: result (f18f110, `judged-r1-f18f110.log`)

56 PASS / 3 FAIL / 1 UNMEASURED / 14 BASELINE / 4 REPORTED. It took 2h08m (about $51), and the
fleet was verified down with a readable inventory. **No REFUTED condition fired.** Gate-p4
GREEN (68/0/0), gate-p3 GREEN (56/0/0), `-race` clean on all three hosts, x44 the 8-thread family
max everywhere (Zen 4 ties `shared` at 934.35 against 934.4), and **skx `Ceiling/compute` 1565.5**
(band 1518–1612), with x44/shared = 1565/1391 = 1.125. Against the softer predictions:
- **missed, rendering:** amd64 hosts are registry-governed, so the share rows render BASELINE
  (candidate rows), not arm64's "NO FRACTION IN FORCE" PASS lines. The headline is UNMEASURED
  because all three hosts are BASELINE. I predicted the arm64 rendering without reading this branch;
- skx shares: Sgemm 29.0 ✓ and Ssymm 28.5 ✓ (band 27.0–29.5); **Ssyrk 30.4 ✗**, see below;
- Zen shares within 1 point of ceil8: held on zen4 except Ssyrk (77.0 against 75.4). **Not held on
  zen5:** Sgemm 49.0 (+1.4), Ssyrk 50.6 (+3.6), Ssymm 44.3 (−2.2, an 8-thread rate 4.2% below the
  README, inside criterion 9's 5%);
- 1-thread Sgemm: zen4 108.4 ✓, zen5 177.6 ✓, **skx 68.72 ✗** (band 64.3–68.3, so +3.6%);
- criterion 9 FAILs on skx `Ceiling/compute` (1391 → 1565.5) ✓, and on the published peak
  denominator (185.5 → 208.55, the same instrument change). **Unpredicted: Ssyrk is 5–7% above
  the README on all three hosts.** The candidate cause is `4e8724c` (2026-10-24, about 2× on the
  transposing pack). Ssyrk packs A untransposed as gemm's A operand, which is the branch that commit
  sped up. It is UNCONFIRMED, with no A/B. Either way, the README regeneration over r1+r2 re-derives
  these rows.

## r2: predictions (written from r1, before r2)

r2 is the second draw. The registry rows land over r1 + r2 by `host-baselines.tsv`'s rules, so
r2 is **not yet judged against a bar** either. Predicted:
- the same rendering: 14 BASELINE, the UNMEASURED headline, gate-p3/p4 GREEN, `-race` clean;
- every share row reproduces r1 within README_TOL (5%), so **all 12 are admissible** at N=2. The
  widest spread is under 2.6 points, as ceil8's amd64 pair was (2.56%);
- skx `Ceiling/compute` within 0.5% of 1565.5, and x44 the family max on every host;
- criterion 9 FAILs on the same set as r1, and nothing outside criterion 9 FAILs.
**REFUTED if** a share row's cross-draw spread exceeds 5% (that row is inadmissible and cannot land),
gate-p3/p4 is RED, or any FAIL appears outside criterion 9.

## r2: result (c0c373d, `judged-r2-c0c373d.log`)

56 PASS / 3 FAIL / 1 UNMEASURED / 14 BASELINE / 4 REPORTED, the same tally as r1. It took 2h06m
(about $51), and the fleet was verified down with a readable inventory. **REFUTED, on its first
clause:** keel-zen5 `share/4x32/avx512/Ssymm` read 44.3 then 46.9, a **5.54%** cross-draw spread, which
is past the 5% same-claim band, so that row is inadmissible. Beyond the prediction, keel-zen5
`rate/Saxpy/n=1048576` spread **8.81%**. I also predicted 12 admissible share rows, and there are
only 11 candidates: keel-zen5 Strsm was NOISE-LIMITED in both runs (8-thread interval about 9%), so the
gate refused it before any bar. Everything else held: the rendering, gate-p3/p4 GREEN, `-race`
clean, x44 the family max, skx `Ceiling/compute` 1565 (+0.0% on r1), and no FAIL outside
criterion 9. Criterion 9 added one row that r1 did not have: skx Strsm/threads=8, at 269.5 against the
README's 254.8.

**Landed over N=2**, from the gate's own candidates and by ceil8's precedents: **27 of 29** rows.
All of skx's and zen4's rows landed, plus 7 of zen5's 9. The two refused rows are named in zen5's
grounds column, and they render `new` until a third draw settles them (#178). **Witness rows: skx and
zen4 landed; zen5's is HELD.** These runs judged nothing (all BASELINE), so landing zen5's witness
would turn its refused rows into an `owing` red that is not a regression. That is
ceil8-amd64's precedent, quoted in `host-baselines.tsv`. All 27 drive `registered` through
`baseline_state` with the gate's production arguments. zen5's three missing rows drive `new`, and a
positive control (a missing row on skx, whose witness did land) drives `owing`. Era famaxx86 is closed.
