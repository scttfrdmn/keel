![keel — pure-Go float32 BLAS, SIMD-accelerated linear algebra for Go](doc-site/assets/keel-hero.webp)

# keel

A pure-Go float32 BLAS subset built on Go's experimental SIMD support
(`GOEXPERIMENT=simd`, Go 1.27+; scalar path Go 1.26+). Kernels that inline, respect safepoints,
and parallelize on the caller's `GOMAXPROCS` — no cgo, no background
threads, deploys as a normal Go module.

**Status: v0.2.0.** Levels 1–3 are implemented and gated on amd64 and arm64, and
all six phase gates P0–P5 are green, certified by judged `gate-p5` runs on AWS, with
the certified rev named in the [release notes](../../releases/tag/v0.2.0). Major is still 0, so a minor bump may
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
| AMD EPYC 9R45 | Sgemm | 1 | 173.9 | 60.5% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [173.1, 174.4] |
| AMD EPYC 9R45 | Sgemm | 8 | 1095 | 47.6% of 2299.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1094, 1095] |
| AMD EPYC 9R45 | Ssyrk | 1 | 147.2 | 51.2% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [146.5, 148.1] |
| AMD EPYC 9R45 | Ssyrk | 8 | 1082 | 47.0% of 2299.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1080, 1083] |
| AMD EPYC 9R45 | Ssymm | 1 | 166.7 | 58.0% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [165, 167.3] |
| AMD EPYC 9R45 | Ssymm | 8 | 1062 | 46.2% of 2299.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1055, 1068] |
| AMD EPYC 9R45 | Strsm | 1 | 121.2 | 42.2% of 287.4 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [120.7, 125.2] |
| AMD EPYC 9R45 | Strsm | 8 | 824.5 | 35.9% of 2299.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [773.3, 859] |
| AMD EPYC 9R45 | Ceiling/compute | 8 | 2293 | 99.7% of 2299.2 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [2293, 2293] |
| AMD EPYC 9R14 | Sgemm | 1 | 107.2 | 91.6% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [107, 107.2] |
| AMD EPYC 9R14 | Sgemm | 8 | 693.4 | 74.1% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [692.7, 693.8] |
| AMD EPYC 9R14 | Ssyrk | 1 | 96.92 | 82.8% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [96.74, 97.03] |
| AMD EPYC 9R14 | Ssyrk | 8 | 707.6 | 75.6% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [707, 708.3] |
| AMD EPYC 9R14 | Ssymm | 1 | 104.3 | 89.1% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [104.2, 104.4] |
| AMD EPYC 9R14 | Ssymm | 8 | 682.8 | 72.9% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [682.3, 683.7] |
| AMD EPYC 9R14 | Strsm | 1 | 73.84 | 63.1% of 117 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [73.61, 73.98] |
| AMD EPYC 9R14 | Strsm | 8 | 466.7 | 49.9% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [461.8, 491.9] |
| AMD EPYC 9R14 | Ceiling/compute | 8 | 934.2 | 99.8% of 936.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [933.4, 935.1] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 1 | 66.34 | 35.8% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [66.06, 68.44] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Sgemm | 8 | 448.5 | 30.2% of 1484.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [447.4, 450.5] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 1 | 62.42 | 33.6% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [62.22, 63.85] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssyrk | 8 | 446.8 | 30.1% of 1484.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [446.4, 451.2] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 1 | 66 | 35.6% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [65.79, 67.81] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ssymm | 8 | 439.8 | 29.6% of 1484.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [435.2, 444.2] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 1 | 37.78 | 20.4% of 185.5 GFLOP/s, the 1-thread avx512 microkernel peak; pooled benchstat over N=2 archives, CI [37.54, 38.17] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Strsm | 8 | 254.8 | 17.2% of 1484.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [253.6, 256.2] |
| Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | Ceiling/compute | 8 | 1391 | 93.7% of 1484.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [1391, 1392] |
| Neoverse-V2 | Sgemm | 1 | 37.33 | 60.3% of 61.87 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [37.3, 37.36] |
| Neoverse-V2 | Sgemm | 8 | 265 | 53.5% of 495.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [264.9, 265.1] |
| Neoverse-V2 | Ssyrk | 1 | 37.2 | 60.1% of 61.87 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [37.14, 37.23] |
| Neoverse-V2 | Ssyrk | 8 | 285.4 | 57.7% of 495.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [285.3, 285.5] |
| Neoverse-V2 | Ssymm | 1 | 37.06 | 59.9% of 61.87 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [37.02, 37.09] |
| Neoverse-V2 | Ssymm | 8 | 263.4 | 53.2% of 495.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [263.3, 263.5] |
| Neoverse-V2 | Strsm | 1 | 31.79 | 51.4% of 61.87 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [31.78, 31.8] |
| Neoverse-V2 | Strsm | 8 | 243 | 49.1% of 495.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [242.7, 243.4] |
| Neoverse-V2 | Ceiling/compute | 8 | 487 | 98.4% of 495.0 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [486.4, 487.8] |
| Neoverse-V1 | Sgemm | 1 | 34.39 | 83.0% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [34.35, 34.44] |
| Neoverse-V1 | Sgemm | 8 | 243.5 | 73.5% of 331.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [243.4, 243.6] |
| Neoverse-V1 | Ssyrk | 1 | 34.14 | 82.4% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [34.13, 34.17] |
| Neoverse-V1 | Ssyrk | 8 | 262.5 | 79.2% of 331.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [262.4, 262.5] |
| Neoverse-V1 | Ssymm | 1 | 34.22 | 82.6% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [34.2, 34.25] |
| Neoverse-V1 | Ssymm | 8 | 242.4 | 73.2% of 331.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [242.2, 242.5] |
| Neoverse-V1 | Strsm | 1 | 28.95 | 69.9% of 41.42 GFLOP/s, the 1-thread neon microkernel peak; pooled benchstat over N=2 archives, CI [28.92, 28.95] |
| Neoverse-V1 | Strsm | 8 | 217.8 | 65.7% of 331.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [217.8, 217.9] |
| Neoverse-V1 | Ceiling/compute | 8 | 331 | 99.9% of 331.4 GFLOP/s, that same peak x 8 cores; pooled benchstat over N=2 archives, CI [331, 331.1] |
<!-- keel-numbers: end -->

<!-- keel-caption: begin -->
All 45 rows are one pooled `benchstat` estimator per cell, not a median of per-run medians: every sample a host contributed across the 4 archived runs of one era is pooled into a single population, and the median and 95% CI printed beside it are both read off that one population — `scripts/gate-p5.sh` at rev `628c38a` (the judged run, which dates this page) and `852ad55`, `3745b72`, `5948dda`, logs in `archive/strsm-refresh-amd64-r1/gate-p5-amd64-strsm-r1-628c38a.log`, `archive/strsm-refresh-amd64-r2/gate-p5-amd64-strsm-r2-852ad55.log`, `archive/strsm-refresh-arm64-r1/gate-p5-arm64-strsm-r1-3745b72.log`, `archive/strsm-refresh-arm64-r2/gate-p5-arm64-strsm-r2-5948dda.log` — at n=4096 square, `GOMAXPROCS` pinned to the threads column, `absent` governor on every host. Each cell names its own N, counting archives. Because the point and the interval come from the same estimator, every published value lies inside its own CI; the percent-of-peak denominator is pooled the same way, so the 1-thread and 8-thread rows for a routine are a ratio of like estimators. Rows from different CPUs are not comparable, because the peaks differ. A gate-p5 run measures one GOARCH, so no single run can judge this table: the rates pool every archive named above, while each host takes its verdicts from the first of the 4 judged runs named for that host. Two judged runs that grade one row differently are refused rather than averaged, because a verdict belongs to the gate that rendered it.

The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at 8 threads on the host itself, where the droop is inside the reading.

Derived from the table above, so pooled over the era's archives rather than read off the judged run alone, as a share of each host's own 8-thread ceiling: AMD EPYC 9R45 32.4-47.5%; AMD EPYC 9R14 47.7-75.6%; Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz 18.3-32.3%; Neoverse-V2 49.5-58.4%; Neoverse-V1 65.6-79.0%. Those ceilings are 99.7%, 99.8%, 93.8%, 98.2% and 99.9% of 8x each host's own 1-thread peak -- a different factor per host, which is why the retired 6.0x cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings' own rates are the `Ceiling/compute` rows of the table above, one per host, so criterion 9 re-measures each on the host whose CPU the row names -- until #113 they were caption-only, outside the region that gate re-measures, which made the one rate every judged share divides by the one rate nothing re-checked (§7 rule 7, and criterion 9 is what noticed).

Every one of the 20 routine-host pairs those 40 routine rows form clears the bars scripts/gate-p5.sh enforces, net of confidence intervals: the judged routines must reach 42.8% of each host's own measured 8-thread ceiling (#6), and Strsm's own >= x T8/T1 floor is RETIRED (#177, 2026-10-05): it was the same quantity the 6.0x cross-host floor was retired for, and it reddened a microkernel that improved both of its arms. The 6.0x cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.
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
order and fuse multiply-adds. The core `keel` package does not depend on Gonum; only
`keel/gonumblas` does.

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
