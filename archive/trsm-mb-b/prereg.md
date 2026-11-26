<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #37 stage B: Trsm MB=128 on the published rows. Predictions

**Why.** `archive/trsm-mb-judged/` found MB=128 faster than the shipped 64 in all 24 cells on six
judged hosts. Its 4 samples per cell gave no CI, so this stage decides. Scott, 2026-11-26: "go".

**The change** (branch `exp-mb128`):
- `internal/block`'s `MB` goes from 64 to 128. It drives Trsm's outer block and Trmm's.
- float64 is held at 64 by an f64gen override, the value its own 2a41213 rule left in place, so
  `block64` does not inherit an unmeasured 128 (cf. #209).
- 128 is a whole number of tiles for both shipped kernel families (NR 32 and 16).
- Tests pass on the simd and stock toolchains.

**Run** (`driver.sh`). main `cdf98f7` against the branch, on the six README hosts at judged sizes:
- **Benchmark:** BenchmarkScale at n=4096, 1T and 8T, with gate-p5's placement.
- **Samples:** 4 seeds × 3 = 12 per arm, with the arms alternating.
- **Correctness first:** the full suites, Trsm and Trmm included, run on each host.

**Predictions** (Strsm GFLOP/s, N/O). Each figure is stage A's median time change converted to a
rate change, with a ±4-point band:

| host | 8T | 1T |
|---|---|---|
| Skylake-SP | +20.6% | +23.8% |
| Zen 5 | +6.8% | +10.7% |
| Zen 4 | +5.8% | +20.3% |
| V1 | +4.3% | +4.1% |
| V2 | +1.2% | +2.5% |
| V3 | +3.2% | +2.9% |

1. **Correctness:** rc=0 on all six hosts.
2. **Strsm** lands in its band on every host and thread count.
3. **Controls:** Sgemm, Ssyrk and Ssymm, which MB cannot reach, have net ranges containing zero.

**Ship rule.** It ships if Strsm 8T gains net of CI on ≥ 4 of 6 hosts and no row regresses beyond its
CI. The README's Strsm rows then move, so judged pairs re-derive them before any published number
changes. **REFUTED for shipping** if no host's Strsm 8T gains net of CI.
