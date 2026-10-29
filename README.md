![keel — pure-Go float32 BLAS, SIMD-accelerated linear algebra for Go](doc-site/assets/keel-hero.webp)

# keel

A pure-Go float32 BLAS subset built on Go's experimental SIMD support
(`GOEXPERIMENT=simd`, Go 1.27+; scalar path Go 1.26+). Kernels that inline, respect safepoints,
and parallelize on the caller's `GOMAXPROCS` — no cgo, no background
threads, deploys as a normal Go module.

**Status: v0.2.4.** keel implements every routine in Gonum's `blas.Float32` interface,
with parallel Level 2 ([release notes](../../releases/tag/v0.2.4)). The gated core (Levels 1 and 3, all six
phase gates P0–P5 green) was certified on amd64 and arm64 by judged `gate-p5` runs on
AWS at v0.2.0, with the certified rev named in its
[release notes](../../releases/tag/v0.2.0). v0.2.3 and v0.2.4 are not re-certifications. Major is
still 0, so a minor bump may break the API. What keel does not do — and which parts of that are scheduled,
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
| AMD EPYC 9R45 | Sgemm | 1 | 177.8 | 61.9% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [177.6, 177.9] |
| AMD EPYC 9R45 | Sgemm | 8 | 1127 | 49.0% of 2298.8 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1127, 1128] |
| AMD EPYC 9R45 | Ssyrk | 1 | 158.5 | 55.2% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [158, 158.7] |
| AMD EPYC 9R45 | Ssyrk | 8 | 1163 | 50.6% of 2298.8 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1163, 1164] |
| AMD EPYC 9R45 | Ssymm | 1 | 168.1 | 58.5% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [167.9, 168.4] |
| AMD EPYC 9R45 | Ssymm | 8 | 1076 | 46.8% of 2298.8 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1075, 1078] |
| AMD EPYC 9R45 | Strsm | 1 | 126.3 | 44.0% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [126.1, 126.4] |
| AMD EPYC 9R45 | Strsm | 8 | 847.2 | 36.9% of 2298.8 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [810.9, 881.8] |
| AMD EPYC 9R45 | Ceiling/compute | 8 | 2294 | 99.8% of 2298.8 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [2293, 2295] |
| AMD EPYC 9R14 | Sgemm | 1 | 108.5 | 92.8% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [108.3, 108.7] |
| AMD EPYC 9R14 | Sgemm | 8 | 701 | 74.9% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [699.6, 701.7] |
| AMD EPYC 9R14 | Ssyrk | 1 | 102.3 | 87.4% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [102.2, 102.4] |
| AMD EPYC 9R14 | Ssyrk | 8 | 731 | 78.1% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [728.3, 733.1] |
| AMD EPYC 9R14 | Ssymm | 1 | 104.2 | 89.1% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [102.7, 105.3] |
| AMD EPYC 9R14 | Ssymm | 8 | 684.8 | 73.2% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [683.8, 687.3] |
| AMD EPYC 9R14 | Strsm | 1 | 74.57 | 63.7% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [72.49, 75.11] |
| AMD EPYC 9R14 | Strsm | 8 | 490.7 | 52.4% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [471.6, 503.6] |
| AMD EPYC 9R14 | Ceiling/compute | 8 | 934.2 | 99.8% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [933.1, 934.5] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 1 | 68.72 | 32.9% of 208.6 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [68.64, 68.84] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 8 | 456 | 27.3% of 1669.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [455.5, 456.3] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 1 | 66.84 | 32.0% of 208.6 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [66.7, 67.01] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 8 | 476.9 | 28.6% of 1669.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [476.7, 477.2] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 1 | 65.44 | 31.4% of 208.6 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [65.19, 67.34] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 8 | 446.8 | 26.8% of 1669.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [446, 447.7] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 1 | 38.73 | 18.6% of 208.6 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [37.91, 38.76] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 8 | 269.4 | 16.1% of 1669.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [267.9, 269.8] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ceiling/compute | 8 | 1565 | 93.8% of 1669.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1565, 1566] |
| Neoverse-V2 | Sgemm | 1 | 65.39 | 77.5% of 84.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [65.33, 65.44] |
| Neoverse-V2 | Sgemm | 8 | 438.4 | 64.9% of 675.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [438.3, 438.5] |
| Neoverse-V2 | Ssyrk | 1 | 63.09 | 74.7% of 84.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [62.82, 63.16] |
| Neoverse-V2 | Ssyrk | 8 | 457.6 | 67.8% of 675.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [456.8, 457.9] |
| Neoverse-V2 | Ssymm | 1 | 64.98 | 77.0% of 84.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [64.9, 65.02] |
| Neoverse-V2 | Ssymm | 8 | 435.9 | 64.6% of 675.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [435.8, 436.3] |
| Neoverse-V2 | Strsm | 1 | 49.48 | 58.6% of 84.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [49.32, 49.55] |
| Neoverse-V2 | Strsm | 8 | 337.3 | 49.9% of 675.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [336.2, 337.8] |
| Neoverse-V2 | Ceiling/compute | 8 | 674.8 | 99.9% of 675.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [674.7, 674.8] |
| Neoverse-V1 | Sgemm | 1 | 58.78 | 75.2% of 78.16 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [58.58, 59.09] |
| Neoverse-V1 | Sgemm | 8 | 404.5 | 64.7% of 625.3 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [404, 405.1] |
| Neoverse-V1 | Ssyrk | 1 | 56.75 | 72.6% of 78.16 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [56.72, 56.82] |
| Neoverse-V1 | Ssyrk | 8 | 415.1 | 66.4% of 625.3 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [414.9, 415.2] |
| Neoverse-V1 | Ssymm | 1 | 58.11 | 74.3% of 78.16 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [58.06, 58.16] |
| Neoverse-V1 | Ssymm | 8 | 402.6 | 64.4% of 625.3 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [401.8, 403.4] |
| Neoverse-V1 | Strsm | 1 | 42.95 | 55.0% of 78.16 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [42.87, 43.15] |
| Neoverse-V1 | Strsm | 8 | 289.6 | 46.3% of 625.3 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [289.4, 290.1] |
| Neoverse-V1 | Ceiling/compute | 8 | 556.9 | 89.1% of 625.3 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [555, 558] |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 45 rows are one pooled `benchstat` estimator per cell, not a median of per-run medians: every sample a host contributed across the 4 archived runs of one era is pooled into a single population, and the median and 95% CI printed beside it are both read off that one population — `scripts/gate-p5.sh` at rev `37d7c13` (the judged run, which dates this page) and `f18f110`, `c0c373d`, `085f23e`, logs in `archive/p5-arm64-famax/judged-r2-37d7c13.log`, `archive/p5-amd64-famaxx86/judged-r1-f18f110.log`, `archive/p5-amd64-famaxx86/judged-r2-c0c373d.log`, `archive/p5-arm64-famax/judged-r1-085f23e.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. Each cell names its own N, counting archives. Because the point and the interval come from the same estimator, every published value lies inside its own CI; the percent-of-peak denominator is pooled the same way, so the 1-thread and 8-thread rows for a routine are a ratio of like estimators. Rows from different CPUs are not comparable, because the peaks differ. The verdicts below are the judged run's alone — a verdict belongs to the gate that rendered it and two cannot be averaged.

The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at 8 threads on the host itself, where the droop is inside the reading.

Derived from the table above, so pooled over the era's archives rather than read off the judged run alone, as a share of each host's own 8-thread ceiling: AMD EPYC 9R45 33.7-50.5%; AMD EPYC 9R14 50.3-77.0%; Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz 17.1-30.4%; Neoverse-V2 49.8-67.5%; Neoverse-V1 51.6-74.3%. Those ceilings are 99.8%, 99.8%, 93.8%, 99.9% and 89.0% of 8x each host's own 1-thread peak -- a different factor per host, which is why the retired 6.0x cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings' own rates are the `Ceiling/compute` rows of the table above, one per host, so criterion 9 re-measures each on the host whose CPU the row names -- until #113 they were caption-only, outside the region that gate re-measures, which made the one rate every judged share divides by the one rate nothing re-checked (§7 rule 7, and criterion 9 is what noticed).

8 of the 20 routine-host pairs those 40 routine rows form clear the bars scripts/gate-p5.sh enforces, net of confidence intervals: the judged routines must reach 47.0% of each host's own measured 8-thread ceiling (#6), and Strsm's own >= x T8/T1 floor is RETIRED (#177, 2026-10-05): it was the same quantity the 6.0x cross-host floor was retired for, and it reddened a microkernel that improved both of its arms. A further 11 of those pairs are RECORDED as a candidate baseline in era famaxx86 and judged by nothing, so those rows are published as measurements and not as passes (#6). A further 1 pair was NOISE-LIMITED: its own intervals cost more than the margin every bar is set by, so no bar could decide it and it carries no verdict (§5 rule 19). The 6.0x cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.
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
Level 1 (`Sdot Sdsdot Saxpy Sscal Snrm2 Sasum Isamax Scopy Sswap Srot Srotg Srotm Srotmg`),
Level 2 (`Sgemv Sger Strmv Strsv Ssymv Ssyr Ssyr2`, banded `Sgbmv Stbmv Stbsv Ssbmv`,
packed `Stpmv Stpsv Sspmv Sspr Sspr2`),
Level 3 (`Sgemm` plus derived `Ssyrk Ssyr2k Ssymm Strsm Strmm`). Row-major, float32,
amd64 fast paths (AVX-512/AVX2) and arm64 fast paths (NEON, Level 1 and
Level 3), with a scalar fallback that builds on a stock toolchain. Target:
≥70% of single-threaded OpenBLAS SGEMM on AVX-512 hardware. What is out of
scope, split into commitments, scheduled work, upstream-gated items, one open
question and parked ideas:
[Capabilities & limits](https://scttfrdmn.github.io/keel/limits/) — the canonical
statement. `DESIGN.md` §1–2 carries the design rationale behind it.

## Using keel from Gonum

keel is a drop-in float32 BLAS for [Gonum](https://www.gonum.org/). One line at
start-up routes every `blas32` call through it:

```go
import (
	"gonum.org/v1/gonum/blas/blas32"
	"github.com/scttfrdmn/keel/gonumblas"
)

func init() { blas32.Use(gonumblas.Implementation{}) }
```

All 35 `blas.Float32` routines run in keel. Where keel and Gonum differ on an edge
case of the interface's contract (a negative stride on `Sscal`, an empty `Sgemv`), the
adapter hands that call to Gonum, so code written against Gonum keeps its behaviour.
Where Gonum is wrong, the adapter keeps keel's answer: `Ssyrk` at `beta = 0`
([gonum#2129](https://github.com/gonum/gonum/issues/2129)), and `Sdsdot`, which Gonum
rounds twice and which returns 0 rather than `alpha` when `n = 0`
([gonum#2130](https://github.com/gonum/gonum/issues/2130)). Results agree with Gonum within keel's
tolerance model rather than bit-for-bit, because keel's kernels sum in a different
order and fuse multiply-adds. `keel/gonumblas` is its own module
(`github.com/scttfrdmn/keel/gonumblas`), so the core `keel` module does not depend on
Gonum at all; only the adapter does.

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
