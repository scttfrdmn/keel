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
