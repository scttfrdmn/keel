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
disagreement, so a stale row cannot survive a gate run. Rows are keyed by CPU
model, never by hostname.

<!-- keel-numbers: begin -->
| CPU | benchmark | threads | GFLOP/s | denominator |
| --- | --- | --- | --- | --- |
<!-- keel-numbers: end -->

The table is empty because no gate-p5 run has produced these rows yet, and an
unmeasured row is not something to publish and correct later. gate-p5 reports
this as "publishes no row for &lt;CPU&gt;", which is the state it is in.

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
