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
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 1 | 66.01 | 34.3% of 192.6 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 8 | 445.2 | 28.9% of 1541.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 1 | 63.12 | 32.8% of 192.6 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 8 | 453.9 | 29.5% of 1541.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 1 | 66.41 | 34.5% of 192.6 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 8 | 440.2 | 28.6% of 1541.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 1 | 23.44 | 12.2% of 192.6 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 8 | 163.1 | 10.6% of 1541.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Sgemm | 1 | 173.8 | 60.6% of 287 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Sgemm | 8 | 1104 | 48.1% of 2296.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Ssyrk | 1 | 147.8 | 51.5% of 287 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Ssyrk | 8 | 1096 | 47.8% of 2296.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Ssymm | 1 | 166.7 | 58.1% of 287 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Ssymm | 8 | 1074 | 46.8% of 2296.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Strsm | 1 | 57.59 | 20.1% of 287 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Strsm | 8 | 383.8 | 16.7% of 2296.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Sgemm | 1 | 107 | 91.5% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Sgemm | 8 | 673.8 | 72.0% of 936.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Ssyrk | 1 | 93.47 | 79.9% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Ssyrk | 8 | 703.8 | 75.2% of 936.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Ssymm | 1 | 105.2 | 89.9% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Ssymm | 8 | 662.3 | 70.8% of 936.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Strsm | 1 | 37.64 | 32.2% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Strsm | 8 | 257.9 | 27.6% of 936.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 24 rows are per-row medians over the 2 archived runs of one era — `scripts/gate-p5.sh` at rev `969c360` (the judged run, which dates this page) and `6ba6566`, logs in `build/confirm-969c360.log`, `build/campaign-c30-6ba6566.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. Each cell names its own N. The 1-thread and 8-thread rows for a routine pool the same archives, so their ratio is a ratio of like estimators; rows from different CPUs are not comparable, because the peaks differ. The verdicts below are the judged run's alone — a verdict belongs to the gate that rendered it and two cannot be averaged.

The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at 8 threads on the host itself, where the droop is inside the reading.

Measured in the judged run, as a share of each host's own 8-thread ceiling: Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz 30.6-31.5%; AMD EPYC 9R45 46.9-48.2%; AMD EPYC 9R14 70.7-74.9%. Those ceilings are 94%, 100% and 100% of 8x each host's own 1-thread peak -- a different factor per host, which is why the retired 6.0x cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings' own rates are deliberately not republished here: nothing in the table above re-measures them, and a rate no instrument re-checks is a claim rather than a measurement (§7 rule 7, and criterion 9 is what noticed). They are in the gate logs this caption names, and publishing them here means first making them re-measured rows.

9 of the 12 routine-host pairs those 24 rows form clear the bars scripts/gate-p5.sh enforces, net of confidence intervals: the judged routines must reach 44.2% of each host's own measured 8-thread ceiling (#6), and Strsm must scale >= 6.067x (#37). A further 3 of those pairs are RECORDED as a candidate baseline in era pinned8 and judged by nothing, so those rows are published as measurements and not as passes (#6). The 6.0x cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.

In the previously published block and not in this one: Intel(R) Xeon(R) 6975P-C. A host with no archive in an era cannot be a median over it, so it leaves the table rather than being marked inside it; the era's stated exclusions are in `scripts/measurement-eras.tsv`.
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
