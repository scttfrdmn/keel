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

## Exploration pass: result (e3ddc1b, archived here)

39 PASS / 4 FAIL / 2 UNMEASURED / 2 BASELINE. Against the predictions above:
- **era famax: held.** The preflight, the gate banner and its era line all name famax,
  PROVISIONAL, and first-sight rows are RECORDED as famax candidates.
- **dispatch: held, and it exposed a marker bug.** The kernel ran (Sgemm 1T 54.3 / 61.3
  GFLOP/s on V1 / V2), but the marker printed `5x16/neon` against the row `5x16-pb/neon`, so
  gate-p3 could not find its benchmark. Fixed in `30af10d`.
- **every row produced: not held.** Four unpredicted findings, each fixed in `30af10d`: the
  famax-incompatible 42.8 bar and its misattributed derivation line; gate-p3/p4 arm64 arms
  missed at the promotion; `-race` at the 10-minute default (+27% under PB); and my README
  note stating rates outside the generated block. One FAIL was self-inflicted: a second git
  worktree, registered for #196 mid-run, which the stray-worktree check correctly failed.
- **fo is the family max at 8 threads: held** on both hosts. V1: fo 429.45, x44 331.00,
  shared 330.95 GFLOP/s. V2: fo 620.40, x44 601.70 (97.0%), shared 511.25.

## Judged pair: predictions (written from the exploration numbers, before r1)

**r1 is famax's founding run** on c7g.16xlarge (V1) + c8g.48xlarge (V2). Predicted:
- era famax; dispatch marker `5x16-pb/neon` on both hosts;
- the share criterion is **REPORTED, with no fraction in force** (CEIL_FRACTION suspended
  outside ceil8, §5 rule 17(d)), never PASS or FAIL;
- **criterion 9 FAILs** on every Neoverse README row (they are ceil8 rows on the old kernel
  and denominator), and it is the only FAIL apart from its aggregate lines;
- gate-p4 and gate-p3 underneath it are **GREEN**;
- `-race` is clean, and `fo` is the family max at 8 threads within 1% on both hosts;
- single-thread Sgemm within 8% of the exploration rates: **V1 50–59, V2 56–66 GFLOP/s**.

**Between r1 and r2:** a reviewed commit types famax's bar from r1's rows, by the precedent
construction (the lowest admissible judged row less the 2.6-point margin), with its
derivation printed, and the arm64 baseline candidates are reduced and landed by the rules
in `host-baselines.tsv`.

**r2 is the first judged run** in famax, and it can fail. Its predictions are written after
the bar is typed, and committed before r2 launches.
