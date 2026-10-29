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
