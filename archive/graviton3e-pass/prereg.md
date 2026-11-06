<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Graviton3 vs 3E, the strengthening pass: predictions

**Why.** `archive/graviton3e/` found that under 63-core FMA load a c7g's core-2 FMA rate drops
36.6% at an unchanged clock, while a c7gn's does not. That rests on one instance of each type and
two load levels, which is not enough to publish (Scott, 2026-11-05: "its own blog post"). This pass
adds:
- instance-to-instance spread;
- the throttle curve over the number of loaded cores;
- the throttle's selectivity by FMA density;
- the other Graviton3E type, hpc7g.

**Run** (`driver.sh`). It runs as two fleets. **main** is two fresh c7g.16xlarge and two fresh
c7gn.16xlarge. **hpc** is one hpc7g.16xlarge: it is offered in us-east-1a only, `aws-fleet.sh`
cannot pin a zone, so it runs alone and may fail to launch. Per host:
- **idle clock** on core 2;
- **the throttle curve:** for k = 0, 1, 3, 7, 15, 31 and 63 loader cores running the FMA-only
  loop, core 2's FMA-only peak and clock;
- **selectivity:** with 63 loaders, core 2 runs the three family loops: fo at about 3.8 FMA/cycle,
  and x44 and shared at about 2.0;
- **idle clock** again.

**Predictions** (single-core FMA-only peak; idle is about 80.6 GFLOP/s on both types):
1. **Reproducible by type.** At k=63, each new c7g reads within ±3% of the first run's 51.08, and
   each new c7gn within ±3% of its 80.54.
2. **The c7g curve** is non-increasing in k:
   - at k=7 (8 cores active), core 2 reads 85–91% of its k=0 rate, against the first run's 8T
     ceiling at 87.8% of 8×1T;
   - at k=63 it reads 60–66%.
3. **The c7gn curve** stays ≥ 98% of its k=0 rate at every k.
4. **Selectivity.** With 63 loaders on a c7g, fo is throttled (≤ 70% of idle), while x44 and shared
   stay within 2% of idle, since about 2.0 FMA/cycle fits under the reduced cap. On a c7gn all
   three stay within 2%.
5. **The integer clock** stays within 0.5% of idle at every k, on every host.
6. **hpc7g, if it launches,** behaves as the c7gn does (predictions 3 and 4's c7gn half).

**Reading.** If 1–4 hold, the throttle is a property of the c7g type: rate-selective and graded by
the active vector core count. The 3E types do not have it. If 1 fails, the result is a property
of the instances drawn, and the post waits.

## Result (`9d9e9de`, `g3pass-{main,hpc}-9d9e9de.log`; `tabulate.py` reproduces every figure below)

Five fresh instances: two c7g.16xlarge, two c7gn.16xlarge and one hpc7g.16xlarge. All five are
part 0xd40, and each instance type was read back from the metadata service. Each fleet was verified
down by a readable inventory. Both fleets launched, the hpc7g included. Each host produced 47 rows.

**Core 2's FMA-only rate, % of its own k=0** (median of 3):

| loaded cores k | c7g a | c7g b | c7gn a | c7gn b | hpc7g |
|---|---|---|---|---|---|
| 0 | 78.11 (100%) | 78.14 | 78.14 | 78.14 | 78.13 |
| 1, 3 | 100.0–100.1% | 100.0% | 100.0% | 100.0% | 100.0% |
| 7 | **90.7%** | **90.6%** | 100.0% | 100.0% | 100.0% |
| 15 | **75.5%** | **75.5%** | 100.0% | 100.0% | 100.0% |
| 31 | **67.9%** | **67.9%** | 99.9% | 99.9% | 100.0% |
| 63 | **64.6%** (50.48–50.50) | **64.6%** (50.48–50.49) | 99.9% (78.00–78.04) | 99.8% | 99.9% |

With 63 loaders, core 2's fo / x44 / shared loops read:
- c7g: 50.47 / 41.34 / 41.34, and the same on both instances;
- c7gn: 78.03 / 41.38 / 42.61 and 77.98 / 41.36 / 42.69;
- hpc7g: 78.02 / 41.38 / 42.61.

The integer clock was 2.588–2.589 idle on every host. Its lowest reading at any k was 2.575, on
c7gn b, which is −0.50%. Every other host stayed at or above 2.583.

1. ✓ **Reproducible by type, as a ratio.** The two new c7g agree with each other to 0.02 GFLOP/s at
   every k, and the two c7gn to 0.06. **At k=63 the c7g reads 64.6% of idle, against the first
   run's 63.4%**, and the c7gn 99.9% in both runs.
   - In absolute terms, every host in this pass, both types, reads 3.1% below the first run
     (k=0: 78.1 against 80.6). The first run's binary was built with `-randlayout=1` and this
     pass's with no layout seed. A uniform shift across five instances of two types is a property
     of the binary, not the silicon. That reading is not separately tested.
   - So c7gn k=63 (78.0) misses the ±3% absolute band against 80.54 by 0.1, while its ratio
     reproduces.
2. ✓ **The c7g curve:** flat through 4 active cores, then 90.7% at 8 (band 85–91), 75.5% at 16,
   67.9% at 32 and 64.6% at 64 (band 60–66). Cross-check: 8 × 70.86 = 567 GFLOP/s, which equals the
   first run's measured 8T ceiling (566.3) on another c7g instance.
3. ✓ **The 3E curve:** ≥ 99.8% at every k, on both c7gn and the hpc7g.
4. ✓ **Selectivity.** On c7g, fo is cut to 64.6% while x44 and shared (about 2.0 FMA/cycle) stay at
   41.34, within 0.3% of their idle 41.43. On the 3E, all three are unthrottled.
5. ✓ **The clock** stays within 0.50%, borderline on one host.
6. ✓ **hpc7g** matches c7gn at every point.

**Reading: the throttle is a property of the Graviton3 instance type.** It holds on three c7g
instances across two runs, with ±0.02 between same-run instances. No Graviton3E type throttles
(two c7gn and one hpc7g). It is graded by the number of active vector cores, starting between 4 and
8, and it cuts only the high FMA issue rate. Chip-wide at 64 cores: c7g 64 × 50.5 = 3232 GFLOP/s
and the 3E 64 × 78.0 = 4994, **+54.5%**.

AWS's "35%" sits between this run's 8-core (+10.4%) and 64-core (+54.5%) readings. Which workload
they measured is not known here.
