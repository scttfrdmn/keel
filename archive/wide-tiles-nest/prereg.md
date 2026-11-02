<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 3x64 through the blocked GEMM on four more hosts, at 8 threads on all five, and the KC curve: predictions

**From `archive/wide-tiles-fleet/`.** Under CL 824624 PS2, the 3x64 kernel beats every host's shipped
shape by 17–55%, except on Zen 4 (−1%). On Zen 5, single-thread, the blocked GEMM is 1.19× PS2 4x32
and 1.48× stock's shipped configuration. 3x64's KC curve there is not monotone: best at 128, worst at
256.

**Run.** The same five hosts, 2048³. Blocked GEMM via `BenchmarkBlocking` (the real nest, `splitIC`),
`-randlayout` seeds 1–4, 3 rounds:
- **1T:** on Zen 4, Skylake-SP, SPR and GNR, the PS2 grid KC ∈ {128, 192, 256, 384} × MC ∈ {72, 144,
  288} for 2x32 / 4x32 / 3x64, pinned to core 2;
- **8T:** on all five hosts, PS2 at KC ∈ {128, 384} × MC ∈ {144, 288}, pinned to 8 physical cores (one
  thread per core, read from `lscpu`);
- **stock anchor at both thread counts:** the shipped point, kc=384/mc=144, for 2x32 and 4x32;
- **KC test, Zen 5 only:** `Kernel/{4x32,3x64}/avx512/kc={128,512}` under PS2, seeds 1–8, 3 rounds.

Correctness first on every host: the blocked-Gemm oracle check of every measured kernel under both
toolchains, with exit status kept.

**Predictions** (median of seed medians; "best" is the best grid point):
1. **Correctness:** exit 0 on every host and toolchain.
2. **1T nest under PS2:**
   - on Skylake-SP, SPR and GNR, best 3x64 is ≥ 1.10× the best of that host's other shapes;
   - on Zen 4, best 3x64 is within −8% to +3% of best 4x32 (the kernel was −4%).
3. **1T against stock's shipped point** (the dispatched shape at kc=384/mc=144): PS2 best 3x64 is
   ≥ 1.25× on Skylake-SP, SPR and GNR.
4. **8T:**
   - on Zen 5, Skylake-SP, SPR and GNR, PS2 best 3x64 is ≥ 1.10× PS2's best other shape, and ≥ 1.25×
     stock's shipped point at 8T;
   - on Zen 4, within −8% to +3% of PS2 4x32.
5. **The KC mechanism, Zen 5:**
   - 3x64's kernel at kc=512 is at least 5% slower than at kc=128. Its panel pair is 137 KB against
     34 KB, either side of the L1d;
   - its kc=512/kc=128 ratio is lower than 4x32's (pair 73 KB against 18 KB).

   This brackets the curve. It cannot say why 256 is worse than 384, because BenchmarkKernel has no
   kc=192 or 256 row, and `BenchmarkFeed` refuses MR=3 at 2048 (its fringe-block skip).

**REFUTED if**, at 8 threads on Zen 5, PS2 best 3x64 does not beat PS2 best 4x32 beyond the seed
spread. The single-thread win would then not survive the parallel nest.

**Scope:** an experiment on a patched toolchain (#197 guardrail: never a published row).

## Result (`d95248e`, `exploration-d95248e.log`): not refuted, 6 of 7 predictions held

The same five CPUs as `archive/wide-tiles-fleet/`, read back by model. It took 24m00s, and the fleet
was verified down by a readable inventory.
- The 8-core mask was 0–7 on every host. `lscpu` places those on distinct physical cores, including
  the SMT c5n.
- Row counts match the design: 456 1T and 168 8T rows per host, and 96 kernel-KC rows on Zen 5.

**2048³ blocked GEMM, best grid point** (median of seed medians, GFLOP/s):

| host | 1T: PS2 3x64 vs PS2 best other vs stock shipped | 8T: PS2 3x64 vs PS2 best other vs stock shipped |
|---|---|---|
| Zen 5 | (wide-tiles-fleet) 246.1 · 207.2 · 165.9 | **1626.5** [1622–1628] · 1427.5 [1381–1435] · 1168.0 → 1.14× / 1.39× |
| Skylake-SP | **91.7** · 71.0 · 59.3 → 1.29× / 1.55× | **623.1** · 492.3 · 414.8 → 1.27× / 1.50× |
| Sapphire Rapids | **131.8** · 109.8 · 94.6 → 1.20× / 1.39× | **702.0** · 608.0 · 551.4 → 1.16× / 1.27× |
| Granite Rapids | **131.8** · 112.5 · 94.5 → 1.17× / 1.39× | **757.0** · 676.1 · 602.2 → 1.12× / 1.26× |
| Zen 4 | 101.7 · 107.0 · 105.6 → 0.95× / 0.96× | 708.5 · 739.0 · 727.8 → 0.96× / 0.97× |

The stock shipped point is the dispatched shape (2x32 on Skylake-SP, 4x32 elsewhere) at kc=384/mc=144.
The best other PS2 shape was 4x32 everywhere.

1. ✓ **Correctness:** exit 0 on every host under both toolchains.
2. ✓ **1T nest:** Skylake-SP, SPR and GNR ≥ 1.10×. Zen 4 is −5.0%, inside −8% to +3%.
3. ✓ **1T against stock shipped:** ≥ 1.25× on all three (1.39–1.55×).
4. ✓ **8T:** every margin held, GNR's 1.26× only just. **Not refuted:** on Zen 5 at 8 threads, 3x64's
   slowest seed (1622) is above 4x32's fastest (1435).
5. **Split.**
   - ✗ The 3x64 kernel at kc=512 is only 1.7% slower than at kc=128 (281.9 against 276.9), where at
     least 5% was predicted.
   - ✓ That is a larger loss than 4x32's 0.2%.

   So L1 overflow of the panel pair costs little at the kernel level, and **it does not explain the
   1T nest's KC curve** (best 128, worst 256 on Zen 5). That curve stays unexplained. At 8 threads,
   kc=384 is the best KC on four of five hosts.

**What it establishes.** Under PS2, 3x64 is the best blocked-GEMM shape on four of five x86 hosts:

| | against PS2's best other shape | against stock's shipped configuration |
|---|---|---|
| 1T | +17–29% | +39–55% |
| 8T | +12–27% | +26–50% |

On Zen 4 it is 3–5% behind, because 8 accumulators already saturate that host. A shipped 3x64 would
need dispatch to keep 4x32 on Zen 4.

**Not established:**
- the KC curve's cause;
- larger thread counts;
- any published row (patched toolchain, #197 guardrail; nothing changes for go1.27 users).
