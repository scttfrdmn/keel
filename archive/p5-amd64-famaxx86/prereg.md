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
