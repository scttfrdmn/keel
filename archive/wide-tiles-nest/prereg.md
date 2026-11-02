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
