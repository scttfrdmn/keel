<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# What does Graviton3E's "35% higher vector performance" mean? Predictions

**Why.** Scott, 2026-11-05: "lets probe and figure out what that claim means."

AWS markets Graviton3E (c7gn, hpc7g) as having about 35% higher vector-instruction performance
than Graviton3. Both are Neoverse V1. The EC2 API states a sustained clock of 2.6 GHz for both
c7g.16xlarge and c7gn.16xlarge. The visible difference is network: 200 Gbit against 30. keel has
never measured a 3E.

**Instrument added.** `archive/graviton3e/clock/` is a dependent ADD/EOR chain whose op rate
reads as the clock under 1-cycle ALU latency.
- **Validated** on pollux's Cortex-X925 core 9 (governor `performance`, cpufreq 3.900 GHz),
  through pueue's measured group: **3.884 Gops/s, 99.6%**, in 5 of 5 samples.
- **Assumption, stated:** Neoverse V1's ADD/EOR latency is also 1 cycle. That is recalled, not
  measured here.

**Run** (`driver.sh`). One c7g.16xlarge against one c7gn.16xlarge, stock go1.27.3, this tree:
- **idle clock;**
- **clock and FMA-only peak on core 2** while **63 loaders** run the FMA-only peak loop on every
  other core;
- **idle peak,** plus BenchmarkCeiling (compute and stream) on 8 cores;
- **the shipped 5x16-pb kernel;**
- **BenchmarkScale** at n=4096, 1T and 8T;
- **idle clock again.**

**Predictions:**
1. **Idle clock:** both read 2.6 GHz ±2%. They are the same core at the same stated clock.
2. **This is the claim's candidate mechanism.** Under 63-core vector load, c7g's clock on the
   measured core drops **≥5%** below its idle reading, and c7gn's drops **at least 3 points less**.
   That would be a larger power envelope holding the vector clock. If both drop equally, or
   neither drops, this explanation is refuted.
3. **Single-core vector work is the same:** idle FMA-only peak and the 5x16-pb kernel agree
   within ±3% between hosts.
4. **8-thread work is the same:** the 8T compute ceiling and the 8T BenchmarkScale rows agree
   within ±3%. Eight of 64 cores are far from any power limit.
5. **Memory:** stream bandwidth agrees within ±5%.

**Reading rule.** If no quantity here differs by ≥ 10% between the hosts, the 35% is not visible
in keel's workloads at ≤ 8 threads or in a single core's peak. The result then names what is left
for it: all-core vector throughput (prediction 2's reading), or something this run does not
measure, such as network-coupled HPC codes.

## Result (`214e077`, `run-214e077.log`; `benchci-<host>-<phase>.csv`): the 3E does not throttle vector throughput under multi-core load, and Graviton3 does

One c7g.16xlarge and one c7gn.16xlarge. Both are part 0xd40 (Neoverse V1), with 64 cores, one 32 MiB
L3 and no cpufreq. The run took 8m36s and produced 420 rows. The fleet was verified down by a
readable inventory.

| quantity (core 2 unless stated) | c7g (Graviton3) | c7gn (Graviton3E) | 3E / 3 |
|---|---|---|---|
| clock, idle | 2.589 (2.588–2.589) | 2.588 (2.588–2.589) | 1.000 |
| clock, 63 other cores running FMA | 2.585 (2.584–2.585) | 2.585 (2.584–2.585) | 1.000 |
| **FMA-only peak, 63 other cores running FMA** | **51.08** (50.96–51.08) | **80.54** (80.45–80.55) | **1.577** |
| FMA-only peak, idle | 80.60 | 80.61 | 1.000 |
| 8T compute ceiling, cores 0–7 | 566.3 (560.9–568.5), **87.8% of 8×1T** | 643.7 (643.5–644.0), **99.8%** | 1.137 |
| 8T x44 / shared family loops | 330.9 / 331.0 | 331.3 / 346.4 | 1.00 / 1.05 |
| stream dot / axpy, 8T (GB/s) | 115.3 / 200.1 | 115.5 / 199.3 | 1.00 |
| shipped 5x16-pb kernel, kc 128 / 384 / 512 | 67.46 / 66.60 / 61.73 | 67.49 / 66.98 / 61.70 | 1.00 |
| Sgemm / Ssyrk / Ssymm / Strsm, 1T | 58.50 / 53.97 / 58.09 / 42.80 | 58.55 / 56.19 / 56.42 / 42.73 | 1.00 / 1.04 / 0.97 / 1.00 |
| **Sgemm / Ssyrk / Ssymm / Strsm, 8T** | 407.4 / 408.0 / 405.0 / 289.2 | 449.7 / 439.7 / 447.4 / 293.1 | **1.104 / 1.078 / 1.105** / 1.013 |

1. ✓ **Idle clocks match:** both 2.588–2.589, and the AWS-stated 2.6 GHz. They read the same again
   at the end (F).
2. ✗ **The clock stays the same under load.** My candidate mechanism, a dropping clock, is
   refuted. Under 63-core FMA load the integer clock is 2.585 on both hosts, a 0.15% sag on both,
   and **Graviton3's vector throughput falls 36.6% at that same clock** (80.60 → 51.08). The
   3E's falls 0.1%. Graviton3 throttles its **vector issue**, not its clock, and the 3E does not.
   That mechanism is a reading of the two instruments, not measured: no PMU was read.
3. ✓ **Single-core vector work matches:** idle peak, the kernel at three depths and every 1T row
   agree within ±0.6%, except Ssyrk 1T (+4.1%) and Ssymm 1T (−2.9%). Their intervals are
   [52.9, 55.0] / [55.7, 56.7] and [57.9, 58.5] / [55.2, 57.5]. Both are outside ±3% and
   unexplained. They point in opposite directions, so they are not a host-level effect.
4. ✗ **8-thread work differs.** Graviton3's 8T FMA ceiling is only 87.8% of 8 × its own 1T. The
   3E's is 99.8%. The published-size 8T rows follow: Sgemm +10.4%, Ssyrk +7.8% and Ssymm +10.5%.
   Strsm, the least FMA-dense row, moves only +1.3%.
5. ✓ **Memory matches:** stream agrees within ±0.6%.

**What the "35%" means, as far as this run can see.** The 3E sustains the V1 core's full vector
FMA rate when many cores run vector code at once, and Graviton3 cannot. The gap grows with the
number of active vector cores: 12% at 8 cores of 64 and 37% at 64. Applications with all cores
in dense SIMD land in AWS's quoted range. Single-thread and memory-bound work see nothing.

The throttle is selective. The x44 and shared family loops, at about 2.0 FMA per cycle, are
unthrottled on both hosts at 8T (330.9 against 8 × 41.4 = 331.4). Only the about-3.8 FMA per cycle
FO loop is cut. So the limit binds on vector issue *rate*, which is consistent with a power or
current cap on the SIMD units. The cause is not attributed.

**Consequences for keel, as readings and hypotheses, not attributed:**
- **The README's long-standing V1 ceiling anomaly is this throttle.** `Ceiling/compute` on V1 is
  556.9, 89.1% of 8 × peak, against 99.8–99.9% on V2 and V3. Here it reads 87.8% on c7g and 99.8%
  on c7gn.
- **V1's small gain from the balanced partition** (+0.8–1.6%, `archive/balanced-ic-c/`) is
  plausibly the same cause. A c7g at 8T is held below its compute by this cap, so removing an idle
  tail buys less. This is a hypothesis.
- **On c7g, multi-core rows depend on how many other cores are busy.** A co-tenant running vector
  code on the same instance would lower a measurement. keel's 8-core masks on a 64-core c7g are
  exposed only through the instance's own other cores, which keel leaves idle.
