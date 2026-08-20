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
| AMD EPYC 9R14 | Sgemm | 1 | 106.2 | 90.7% of 117.1 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Sgemm | 8 | 642.2 | 68.6% of 936.8 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Ssyrk | 1 | 89.47 | 76.4% of 117.1 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Ssyrk | 8 | 651.9 | 69.6% of 936.8 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Ssymm | 1 | 106.8 | 91.2% of 117.1 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Ssymm | 8 | 654.3 | 69.8% of 936.8 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R14 | Strsm | 1 | 37.61 | 32.1% of 117.1 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R14 | Strsm | 8 | 264.9 | 28.3% of 936.8 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Sgemm | 1 | 174.2 | 60.8% of 286.65 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Sgemm | 8 | 1052 | 45.9% of 2293.2 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Ssyrk | 1 | 148.1 | 51.7% of 286.65 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Ssyrk | 8 | 968.4 | 42.2% of 2293.2 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Ssymm | 1 | 166.9 | 58.2% of 286.65 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Ssymm | 8 | 991.9 | 43.3% of 2293.2 GFLOP/s, that same peak x 8 cores |
| AMD EPYC 9R45 | Strsm | 1 | 53.64 | 18.7% of 286.65 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD EPYC 9R45 | Strsm | 8 | 416 | 18.1% of 2293.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Sgemm | 1 | 101.2 | 41.5% of 243.9 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Sgemm | 8 | 663 | 34.0% of 1951.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Ssyrk | 1 | 95.12 | 39.0% of 243.9 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Ssyrk | 8 | 672.7 | 34.5% of 1951.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Ssymm | 1 | 97.41 | 39.9% of 243.9 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Ssymm | 8 | 657 | 33.7% of 1951.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Xeon(R) 6975P-C | Strsm | 1 | 54.09 | 22.2% of 243.9 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Xeon(R) 6975P-C | Strsm | 8 | 400.9 | 20.5% of 1951.2 GFLOP/s, that same peak x 8 cores |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 24 rows come from one run — `scripts/gate-p5.sh` at rev `651d1bd`, log in `build/gate-p5-651d1bd.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. The 1-thread and 8-thread rows for a routine are the two arms of that run's scaling ratio, so they are directly comparable to each other; rows from different CPUs are not, because the peaks differ.

The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at 8 threads on the host itself, where the droop is inside the reading.

Measured this run, as a share of each host's own 8-thread ceiling: AMD EPYC 9R14 82.5-90.0%; AMD EPYC 9R45 61.1-65.3%; Intel(R) Xeon(R) 6975P-C 86.3-87.4%. Those ceilings are 76%, 68% and 38% of 8x each host's own 1-thread peak -- a different factor per host, which is why the retired 6.0x cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings' own rates are deliberately not republished here: nothing in the table above re-measures them, and a rate no instrument re-checks is a claim rather than a measurement (§7 rule 7, and criterion 9 is what noticed). They are in the gate log this caption names, and publishing them here means first making them re-measured rows.

1 of the 12 routine-host pairs those 24 rows form does not clear the bars scripts/gate-p5.sh enforces (the judged routines are reported against each host's own measured 8-thread ceiling with no fraction yet ratified (#6); Strsm must scale >= 7.0x (#37); both judged net of confidence intervals). 1 clears it on the point estimate and misses only net of CI, which is a verdict decided by the measurement precision rather than by the parallel nest: AMD EPYC 9R14 Strsm (7.046x, 6.491x net of CI). These are published shortfalls against bars checked on every gate run, not regressions against an earlier reading.
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
