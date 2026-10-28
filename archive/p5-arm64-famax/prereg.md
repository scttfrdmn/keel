# gate-p5 on arm64 in era famax: predictions (#200)

The product of these runs is the archive that re-derives the README's 18 Neoverse rows,
after #198 changed both the arm64 ceiling instrument (family-max `neonPeakTile`) and the
dispatched NEON kernel (`5x16-pb/neon`). The arm64 era `famax` was opened for exactly that,
scoped to arm64 by Scott's #200 ruling (DESIGN.md §5 rule 17, 2026-10-28), and amd64 stays
in `ceil8`. The driver is `driver-frozen.sh`, committed with this file.

## Exploration pass (TIER=explore, c7g.4xlarge + c8g.4xlarge), predictions

This pass witnesses that the whole path works before the judged pair (about $27) is
spent. 4xlarge sizes are correctness-class, so nothing here is a verdict. Predicted:
- the gate's own log resolves era **famax** for arm64. Any `ceil8` in an arm64 era line
  refutes the per-architecture era wiring;
- the dispatch provenance names **5x16-pb/neon** as the active Sgemm kernel;
- every routine row is produced at 1 and 8 threads, and the gate ends with no criterion
  that failed to run (UNMEASURED) for a reason other than the correctness-class admission;
- the `Ceiling/family/*/threads=8` rows exist for fo, x44 and shared, and **fo is the
  maximum within 1% at 8 threads on both hosts**. That is #200's question 1. If x44 or
  shared beats fo by more than 1% at 8 threads, the 8-thread ceiling is not the family max,
  and the judged pair waits on that.

## Judged pair (TIER=judged, two runs)

Its predictions are written after the exploration pass, from its numbers, and committed
before the first judged launch.
