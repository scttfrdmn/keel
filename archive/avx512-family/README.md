# The AVX-512 ceiling family (#199, 7b55f74): avx512Peak under-reads on Skylake-SP

The reading is in `prereg.md`, committed before the run; the numbers are in `summary.txt`.
Hosts: c5n.4xlarge (Xeon 8124M), c7a.2xlarge (EPYC 9R14) and c8a.2xlarge (EPYC 9R45), on
the exploration tier, pinned to core 2, in 10 interleaved rounds. All three hosts passed
the witness tests before timing. The run took 5m28s, and the fleet was verified down.

| host | `avx512` row, f32 / f64 GFLOP/s | x44 | fo | reading |
|---|---|---|---|---|
| Skylake-SP 8124M | 166.1 / 83.05 | **112.5% / 112.5%** (worst round / best row round 1.125) | 59.4 / 58.8% | **under-reads by 12.5%** |
| Zen 4 9R14 | 117.0 / 58.51 | 100.0 / 100.0% | 99.9 / 99.9% | stands |
| Zen 5 9R45 | 285.8 / 142.9 | 100.0 / 100.0% | 91.3 / 90.1% | stands |

By the pre-registered reading, `avx512Peak` under-reads on Skylake-SP and stands as a
ceiling on Zen 4 and Zen 5. The cause is untested. x44 differs from `avx512Peak` in using
distinct operand registers, 16 chains instead of 12, and no anchor NOPs (19 against 27
insns per FMA group), and skx is the issue-bound host of the fleet.

**VOID, 2026-10-28, by name: the "bound" in `prereg.md` and on #191/#199** ("skx and zen4
kernels reached 93.5% and 96.5% of `avx512Peak`, so a member can beat it by at most about
7% and 4% there"). That inference runs backwards. A kernel *below* the peak bounds nothing
about how far the peak sits below the true ceiling; only a loop *above* the peak shows
under-reading. And skx's 93.5% was of its issue roofline, not of peak. This run refutes the
claimed bound directly: x44 beats the peak by 12.5% on skx.
