![keel — pure-Go float32 BLAS, SIMD-accelerated linear algebra for Go](doc-site/assets/keel-hero.webp)

# keel

A pure-Go float32 BLAS subset built on Go's experimental SIMD support
(`GOEXPERIMENT=simd`, Go 1.26+). Kernels that inline, respect safepoints,
and parallelize on the caller's `GOMAXPROCS` — no cgo, no background
threads, deploys as a normal Go module.

**Status: pre-release.** Levels 1–3 are implemented and gated; P0–P4 are
green and P5 (parallelism, dispatch, polish) is in progress with its gate
written and red. Watch the [milestones](../../milestones) for progress.

## Measured rates

Every number keel publishes carries its denominator (DESIGN.md §7 rule 7):
the CPU model, the peak measured on that same host in that same run, and the
OpenBLAS reference where one was available. The table below is re-measured by
`scripts/gate-p5.sh` on each benchmark host and the gate fails on a 5%
disagreement, so a stale row turns the gate red rather than passing unnoticed —
it cannot hide, which is not the same as it cannot exist. The table and the
caption under it are both written by `scripts/readme-numbers.sh` from a gate log,
so neither is typed and neither can drift from the other. Rows are keyed by CPU
model, never by hostname.

<!-- keel-numbers: begin -->
| CPU | benchmark | threads | GFLOP/s | denominator |
| --- | --- | --- | --- | --- |
| Intel(R) Xeon(R) 6975P-C | Sgemm | 1 | 101.7 | 41.5% of 245.14999999999998 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Sgemm | 8 | 672.1 | 34.3% of 1961.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Ssyrk | 1 | 94.84 | 38.7% of 245.14999999999998 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Ssyrk | 8 | 670.1 | 34.2% of 1961.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Ssymm | 1 | 102.4 | 41.8% of 245.14999999999998 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Ssymm | 8 | 673.6 | 34.3% of 1961.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Strsm | 1 | 55.71 | 22.7% of 245.14999999999998 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Strsm | 8 | 422.6 | 21.5% of 1961.2 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Sgemm | 1 | 173.4 | 60.4% of 286.95 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Sgemm | 8 | 1034 | 45.0% of 2295.6 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Ssyrk | 1 | 146.1 | 50.9% of 286.95 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Ssyrk | 8 | 991.4 | 43.2% of 2295.6 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Ssymm | 1 | 170.4 | 59.4% of 286.95 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Ssymm | 8 | 1024 | 44.6% of 2295.6 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Strsm | 1 | 53.79 | 18.7% of 286.95 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Strsm | 8 | 423.2 | 18.4% of 2295.6 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Sgemm | 1 | 106.5 | 91.0% of 117 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Sgemm | 8 | 616.8 | 65.9% of 936.0 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Ssyrk | 1 | 92.16 | 78.8% of 117 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Ssyrk | 8 | 629.5 | 67.3% of 936.0 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Ssymm | 1 | 106.2 | 90.8% of 117 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Ssymm | 8 | 618.4 | 66.1% of 936.0 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Strsm | 1 | 36.11 | 30.9% of 117 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Strsm | 8 | 268.9 | 28.7% of 936.0 GFLOP/s, that same peak x 8 cores |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 24 rows come from one run — `scripts/gate-p5.sh` at rev `ce43bca`, log in `build/gate-p5-ce43bca.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. The 1-thread and 8-thread rows for a routine are the two arms of that run's scaling ratio, so they are directly comparable to each other; rows from different CPUs are not, because the peaks differ.

4 of the 12 scaling ratios those 24 rows form do not clear the floor scripts/gate-p5.sh enforces (>= 6.0x at 8 threads, >= 7.0x for Strsm, judged net of confidence intervals). 3 sit below it outright: AMD EPYC 9R45 Sgemm at 5.960x (5.754x net of CI); AMD EPYC 9R14 Sgemm at 5.792x (5.678x net of CI); AMD EPYC 9R14 Ssymm at 5.826x (5.707x net of CI). 1 clears it on the point estimate and misses only net of CI, which is a verdict decided by the measurement precision rather than by the parallel nest: AMD EPYC 9R45 Ssymm (6.011x, 5.801x net of CI). These are published shortfalls against a floor checked on every gate run, not regressions against an earlier reading.
<!-- keel-caption: end -->

**The denominator here is keel's own microkernel, not OpenBLAS.** No OpenBLAS
reference was taken at these thread counts, so the comparison DESIGN.md §7
rule 7 asks for is absent rather than unflattering, and the only bar these
rows are measured against is what keel's own AVX-512 microkernel achieved on
the same host in the same run. Read the percentages as "how much of its own
kernel does the blocked nest keep", not as a competitive result.

**Both denominators are measured with one core loaded, which cuts against the
8-thread column.** The 1-thread peak is taken with one core loaded, so it
includes a single-core boost clock the 8-thread arm does not get to keep;
multiplying it by 8 therefore asks the parallel nest to beat a clock it never
runs at. That is why the 8-thread percentages sit below the 1-thread ones on
every host, and it is one of the candidate reasons some of these ratios miss
the gate's own scaling floor — which ones is a fact about this run, so it is
stated in the generated caption above rather than described here, where the
earlier hand-written version of this clause named two hosts against a table
holding five shortfalls on three. Ruled 2026-08-17 (#66): the clock is
now measured rather than set, because the AWS fleet that replaced these three
desktops exposes no boost knob at all — so this handicap is disclosed rather than
removed, and these percentages stay a floor on the parallel nest's efficiency and
not an estimate of it.

## Why
Go has excellent float64 numerics (Gonum) and no maintained fast float32
story. The experimental `simd`/`archsimd` packages make preemption-safe
vector kernels possible for the first time; keel is a Goto/BLIS-style
SGEMM-centered BLAS subset built on them — and a field report on the
experiment (`docs/toolchain-notes.md`).

## Scope (v0)
Level 1 (`Sdot Saxpy Sscal Snrm2 Sasum Isamax`), Level 2 (`Sgemv Sger`),
Level 3 (`Sgemm` plus derived `Ssyrk Ssymm Strsm`). Row-major, float32,
amd64 fast paths (AVX-512/AVX2) with a scalar fallback that builds on a
stock toolchain. Target: ≥70% of single-threaded OpenBLAS SGEMM on AVX-512
hardware. Non-goals and rationale: `DESIGN.md` §1–2.

## Building
```
GOEXPERIMENT=simd go build ./...   # fast paths (Go 1.26+)
go build ./...                     # scalar-only, stock toolchain
make test                          # or: gate-p0 … gate-p5
```

## Project management
GitHub milestones = phases P0–P5 (DESIGN.md §4); each has an umbrella issue
with the task checklist and gate criteria. `scripts/bootstrap-github.sh`
creates the repo, labels, milestones, issues, and project board.

## License
Apache-2.0 — Copyright 2026 Scott Friedman. See `LICENSE` and `NOTICE`.
