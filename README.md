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
| AMD Ryzen 9 7950X3D 16-Core Processor | Sgemm | 1 | 152.1 | 92.0% of 165.4 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD Ryzen 9 7950X3D 16-Core Processor | Sgemm | 8 | 880.1 | 66.5% of 1323.2 GFLOP/s, that same peak x 8 cores |
| AMD Ryzen 9 7950X3D 16-Core Processor | Ssyrk | 1 | 129.3 | 78.2% of 165.4 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD Ryzen 9 7950X3D 16-Core Processor | Ssyrk | 8 | 888 | 67.1% of 1323.2 GFLOP/s, that same peak x 8 cores |
| AMD Ryzen 9 7950X3D 16-Core Processor | Ssymm | 1 | 141.6 | 85.6% of 165.4 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD Ryzen 9 7950X3D 16-Core Processor | Ssymm | 8 | 869.8 | 65.7% of 1323.2 GFLOP/s, that same peak x 8 cores |
| AMD Ryzen 9 7950X3D 16-Core Processor | Strsm | 1 | 51.45 | 31.1% of 165.4 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD Ryzen 9 7950X3D 16-Core Processor | Strsm | 8 | 372 | 28.1% of 1323.2 GFLOP/s, that same peak x 8 cores |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Sgemm | 1 | 74.12 | 34.2% of 216.7 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Sgemm | 8 | 498.5 | 28.8% of 1733.6 GFLOP/s, that same peak x 8 cores |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Ssyrk | 1 | 69.82 | 32.2% of 216.7 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Ssyrk | 8 | 501.8 | 28.9% of 1733.6 GFLOP/s, that same peak x 8 cores |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Ssymm | 1 | 73.33 | 33.8% of 216.7 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Ssymm | 8 | 487.2 | 28.1% of 1733.6 GFLOP/s, that same peak x 8 cores |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Strsm | 1 | 26.69 | 12.3% of 216.7 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz | Strsm | 8 | 187.1 | 10.8% of 1733.6 GFLOP/s, that same peak x 8 cores |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Sgemm | 1 | 194.6 | 59.4% of 327.55 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Sgemm | 8 | 1108 | 42.3% of 2620.4 GFLOP/s, that same peak x 8 cores |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Ssyrk | 1 | 168.5 | 51.4% of 327.55 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Ssyrk | 8 | 1134 | 43.3% of 2620.4 GFLOP/s, that same peak x 8 cores |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Ssymm | 1 | 187.6 | 57.3% of 327.55 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Ssymm | 8 | 1080 | 41.2% of 2620.4 GFLOP/s, that same peak x 8 cores |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Strsm | 1 | 59.95 | 18.3% of 327.55 GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |
| AMD RYZEN AI MAX+ 395 w/ Radeon 8060S | Strsm | 8 | 423.2 | 16.2% of 2620.4 GFLOP/s, that same peak x 8 cores |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 24 rows come from one run — `scripts/gate-p5.sh` at rev `335ea9d`, log in `build/gate-p5-335ea9d.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `performance` governor on every host. The 1-thread and 8-thread rows for a routine are the two arms of that run's scaling ratio, so they are directly comparable to each other; rows from different CPUs are not, because the peaks differ.

4 of the 12 scaling ratios those 24 rows form do not clear the floor scripts/gate-p5.sh enforces (>= 6.0x at 8 threads, >= 7.0x for Strsm, judged net of confidence intervals). 3 sit below it outright: AMD Ryzen 9 7950X3D 16-Core Processor Sgemm at 5.788x (5.730x net of CI); AMD RYZEN AI MAX+ 395 w/ Radeon 8060S Sgemm at 5.695x (5.582x net of CI); AMD RYZEN AI MAX+ 395 w/ Radeon 8060S Ssymm at 5.760x (5.703x net of CI). 1 clears it on the point estimate and misses only net of CI, which is a verdict decided by the measurement precision rather than by the parallel nest: AMD RYZEN AI MAX+ 395 w/ Radeon 8060S Strsm (7.059x, 6.919x net of CI). These are published shortfalls against a floor checked on every gate run, not regressions against an earlier reading.
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
