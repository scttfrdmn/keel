![keel — pure-Go float32 BLAS, SIMD-accelerated linear algebra for Go](doc-site/assets/keel-hero.webp)

# keel

A pure-Go float32 BLAS subset built on Go's experimental SIMD support
(`GOEXPERIMENT=simd`, Go 1.27+; scalar path Go 1.26+). Kernels that inline, respect safepoints,
and parallelize on the caller's `GOMAXPROCS` — no cgo, no background
threads, deploys as a normal Go module.

**Status: v0.1.0.** Levels 1–3 are implemented and gated, and all six phase
gates P0–P5 are green — certified by a `gate-p5` run on a three-host AWS fleet,
with the certified rev and the seam between it and the tag named in the
[release notes](../../releases/tag/v0.1.0). Major is still 0, so a minor bump may
break the API. What keel does not do — and which parts of that are scheduled,
open or parked — is stated once, in
[Capabilities & limits](https://scttfrdmn.github.io/keel/limits/).

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
| Neoverse-V2 | Sgemm | 1 | 37.33 | 61.1% of 61.12 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V2 | Sgemm | 8 | 265 | 54.2% of 489.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V2 | Ssyrk | 1 | 37.19 | 60.9% of 61.12 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V2 | Ssyrk | 8 | 285.5 | 58.4% of 489.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V2 | Ssymm | 1 | 37.1 | 60.7% of 61.12 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V2 | Ssymm | 8 | 263.8 | 53.9% of 489.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V2 | Strsm | 1 | 30.41 | 49.7% of 61.12 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V2 | Strsm | 8 | 230.1 | 47.1% of 489.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V2 | Ceiling/compute | 8 | 493.4 | 100.9% of 489.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V1 | Sgemm | 1 | 34.45 | 83.2% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V1 | Sgemm | 8 | 243.5 | 73.5% of 331.4 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V1 | Ssyrk | 1 | 34.14 | 82.4% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V1 | Ssyrk | 8 | 262.6 | 79.3% of 331.4 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V1 | Ssymm | 1 | 34.25 | 82.7% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V1 | Ssymm | 8 | 242.5 | 73.2% of 331.4 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V1 | Strsm | 1 | 27.67 | 66.8% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; median of N=2 archives |
| Neoverse-V1 | Strsm | 8 | 206.6 | 62.3% of 331.4 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Neoverse-V1 | Ceiling/compute | 8 | 331 | 99.9% of 331.4 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Sgemm | 1 | 173.2 | 60.4% of 286.9 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Sgemm | 8 | 1093 | 47.6% of 2295.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Ssyrk | 1 | 145.9 | 50.9% of 286.9 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Ssyrk | 8 | 1080 | 47.0% of 2295.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Ssymm | 1 | 166.8 | 58.2% of 286.9 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Ssymm | 8 | 1063 | 46.3% of 2295.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Strsm | 1 | 58.01 | 20.2% of 286.9 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R45 | Strsm | 8 | 388.8 | 16.9% of 2295.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R45 | Ceiling/compute | 8 | 2287 | 99.6% of 2295.2 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Sgemm | 1 | 107.2 | 91.6% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Sgemm | 8 | 693.4 | 74.1% of 935.6 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Ssyrk | 1 | 96.61 | 82.6% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Ssyrk | 8 | 704.6 | 75.3% of 935.6 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Ssymm | 1 | 104.6 | 89.4% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Ssymm | 8 | 681.9 | 72.9% of 935.6 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Strsm | 1 | 38.08 | 32.6% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| AMD EPYC 9R14 | Strsm | 8 | 262.7 | 28.1% of 935.6 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| AMD EPYC 9R14 | Ceiling/compute | 8 | 933.8 | 99.8% of 935.6 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 1 | 66.4 | 35.8% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 8 | 445.7 | 30.0% of 1484.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 1 | 62.37 | 33.6% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 8 | 446.4 | 30.1% of 1484.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 1 | 65.83 | 35.5% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 8 | 435.9 | 29.4% of 1484.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 1 | 23.69 | 12.8% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 8 | 164.5 | 11.1% of 1484.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ceiling/compute | 8 | 1391 | 93.7% of 1484.0 GFLOP/s, that same peak x 8 cores; median of N=2 archives |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 45 rows are per-row medians over the 4 archived runs of one era — `scripts/gate-p5.sh` at rev `5e769f7` (the judged run, which dates this page) and `119b673`, `9a5e2f5`, `9a5e2f5`, logs in `archive/arm64-ceil8-confirm/gate-p5-arm64-ceil8-confirm.log`, `archive/arm64-ceil8-transition/gate-p5-arm64-ceil8-transition.log`, `archive/ceil8-amd64-transition/gate-p5-amd64-ceil8-c10.log`, `archive/ceil8-amd64-c30/gate-p5-amd64-ceil8-c30.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. Each cell names its own N. The 1-thread and 8-thread rows for a routine pool the same archives, so their ratio is a ratio of like estimators; rows from different CPUs are not comparable, because the peaks differ. The verdicts below are the judged run's alone — a verdict belongs to the gate that rendered it and two cannot be averaged.

The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at 8 threads on the host itself, where the droop is inside the reading.

Measured in the judged run, as a share of each host's own 8-thread ceiling: Neoverse-V2 44.5-57.7%; Neoverse-V1 61.4-79.3%; AMD EPYC 9R45 16.2-47.6%; AMD EPYC 9R14 27.2-75.2%; Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz 11.8-32.0%. Those ceilings are 100.9%, 99.9%, 99.6%, 99.8% and 93.7% of 8x each host's own 1-thread peak -- a different factor per host, which is why the retired 6.0x cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings' own rates are the `Ceiling/compute` rows of the table above, one per host, so criterion 9 re-measures each on the host whose CPU the row names -- until #113 they were caption-only, outside the region that gate re-measures, which made the one rate every judged share divides by the one rate nothing re-checked (§7 rule 7, and criterion 9 is what noticed).

8 of the 20 routine-host pairs those 40 routine rows form clear the bars scripts/gate-p5.sh enforces, net of confidence intervals: the judged routines must reach 42.8% of each host's own measured 8-thread ceiling (#6), and Strsm's own >= x T8/T1 floor is RETIRED (#177, 2026-10-05): it was the same quantity the 6.0x cross-host floor was retired for, and it reddened a microkernel that improved both of its arms. A further 12 of those pairs are RECORDED as a candidate baseline in era ceil8 and judged by nothing, so those rows are published as measurements and not as passes (#6). The 6.0x cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.
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
amd64 fast paths (AVX-512/AVX2) and arm64 fast paths (NEON, Level 1 and
Level 3), with a scalar fallback that builds on a stock toolchain. Target:
≥70% of single-threaded OpenBLAS SGEMM on AVX-512 hardware. What is out of
scope, split into commitments, scheduled work, upstream-gated items, one open
question and parked ideas:
[Capabilities & limits](https://scttfrdmn.github.io/keel/limits/) — the canonical
statement. `DESIGN.md` §1–2 carries the design rationale behind it.

## Building
```
GOEXPERIMENT=simd go build ./...   # fast paths (Go 1.27+, see doc-site/troubleshooting.md)
go build ./...                     # scalar-only, stock toolchain (Go 1.26+)
make test                          # or: gate-p0 … gate-p5
```

## Project management
GitHub milestones = phases P0–P5 (DESIGN.md §4); each has an umbrella issue
with the task checklist and gate criteria. `scripts/bootstrap-github.sh`
creates the repo, labels, milestones, issues, and project board.

## License
Apache-2.0 — Copyright 2026 Scott Friedman. See `LICENSE` and `NOTICE`.
