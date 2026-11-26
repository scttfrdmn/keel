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

## Result (`ad46f68`, `run-ad46f68.log`; `benchci-<host>-{O,N}.csv` over each arm's 12 samples): ships

Six judged-size hosts, with masks as in the driver: spread on Zen, packed elsewhere. The run
produced 192 rows per host, the full count. 18 of 18 correctness suites exited rc=0, Trsm and Trmm
included. The fleet was verified down by a readable inventory.

**Strsm, N/O** (% median [range net of both CIs]):

| host | 8T | predicted | 1T | predicted |
|---|---|---|---|---|
| Skylake-SP | **+22.1** [+20.9, +23.2] | +20.6 ✓ | +24.0 [+19.9, +26.9] | +23.8 ✓ |
| Zen 5 | **+9.3** [+0.3, +12.9] | +6.8 ✓ | +12.1 [+10.6, +13.6] | +10.7 ✓ |
| Zen 4 | **+7.9** [+2.1, +18.5] | +5.8 ✓ | +11.6 [+10.5, +20.2] | +20.3 ✗ (−4.7 below the band) |
| V1 | **+4.9** [+3.6, +5.9] | +4.3 ✓ | +5.0 [+2.4, +8.8] | +4.1 ✓ |
| V2 | **+1.6** [+0.8, +2.4] | +1.2 ✓ | +1.9 [+0.9, +4.2] | +2.5 ✓ |
| V3 | **+3.2** [+2.2, +4.1] | +3.2 ✓ | +2.3 [+0.9, +4.3] | +2.9 ✓ |

1. ✓ **Correctness.**
2. **11 of 12 in band.** Zen 4 1T gains +11.6%, below the predicted +16.3–24.3%.
3. ✓ **Controls: 36 of 36** (Sgemm/Ssyrk/Ssymm × 2 thread counts × 6 hosts) have a net range
   containing zero.

**The ship rule holds.** Strsm 8T gains net of CI on **6 of 6** hosts, and no row regresses beyond
its CI.

**Scope, settled after the run.** MB was also Trmm's block, and this run times Strsm only, so Trmm
gets its own `trmmMB`, held at 64. Two reasons:
- Trmm computes its diagonal blocks as dense GEMMs, so its response to a larger block is not
  inferable from Trsm's.
- `trmm_test.go`'s sizes straddle 64 to run the multi-block path, and at 128 they would not.

Trmm therefore behaves exactly as on main. Trsm's multi-block coverage moves with MB on its own:
`TestStrsmMultiBlock` sizes itself as `block.MB + 1`. float64 Dtrsm/Dtrmm stay at 64 through the
f64gen override.
