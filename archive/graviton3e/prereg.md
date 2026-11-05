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
