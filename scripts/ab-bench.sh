#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# ab-bench.sh PRESET [BASE_REF] — the A/B measurements that are not gates.
#
# One caller for every scripts/ab.sh use: l1 (#47), edge (#22), drift (#141). The
# three differ only in the strings ab.sh already takes as parameters, so the table
# below is the entire difference and nothing else here asks which preset it is
# serving — #131's rule, applied to the callers this time. One file per preset was
# the wrong shape: each paid a shebang, a licence, a `set`, a main() and a source
# to set four strings.
#
# NONE of these is a gate. They certify nothing, change no criterion, and exit 0
# whatever they measure; their product is a table, per host, of the same
# benchmarks against two builds. Read scripts/ab.sh for the mechanism, and for why
# a driver does no work at its top level (#51).
#
# usage: scripts/ab-bench.sh {l1|edge|drift} [BASE_REF]
set -euo pipefail

# ab_preset NAME — the four strings ab.sh takes, plus this caller's default base
# ref and closing lines.
#
# Deliberately called BEFORE ab.sh is sourced, which is what lets a preset raise a
# KEEL_BENCH_* default: bench.sh defaults those at source time, so a value set
# after that point cannot be told apart from the operator's own, and the operator
# has to keep winning.
ab_preset() {
  AB_BASE="HEAD~1"
  AB_DONE=("done. Numbers above are this host set at these two builds; nothing here is a criterion.")
  case "$1" in
  l1)
    # #47: internal/l1's loops went index-driven to slice-advancing, removing every
    # surviving bounds check and roughly halving the six unrolled reductions'
    # instruction counts, while making the four non-unrolled loops (Axpy, Scal, both
    # widths) 1-5 instructions *longer*, for the reason T19 gives. Static counts
    # cannot say what either is worth: these routines are not all issue-bound, and
    # at n = 1<<20 they read 4 MB per call, where the loop body is not the limit.
    #
    # bench/bench_test.go's four sizes are the whole point of running this: 256,
    # 4096, 65536 and 1<<20 float32s are 1 KB, 16 KB, 256 KB and 4 MB, i.e. L1-,
    # L2-, L3- and memory-resident on all three hosts in docs/hosts.md. A loop-body
    # change should show up at 1 KB and wash out at 4 MB; if it does not, the
    # explanation is not "the loop got shorter".
    AB_TITLE="== l1 — issue #47. Not a gate: this certifies nothing. =="
    AB_FILTER="${KEEL_L1_FILTER:-BenchmarkL1S}"
    AB_TAG="l1"
    AB_NOTES=("sizes are bench/bench_test.go's: 1 KB / 16 KB / 256 KB / 4 MB of float32.")
    ;;
  edge)
    # The measurement the #22 ruling ordered — rank **A** (the incumbent:
    # zero-padded panels, a scratch MR×NR tile, and a *scalar* add-back of the live
    # sub-rectangle) against **C** (same kernel, same scratch tile, that add-back
    # vectorized). **B**, a masked C update inside the microkernel, is deliberately
    # not built: its costs are certain (a branch in the hot loop, and double P2's
    # zero-spill audit surface) and its coverage is a subset of C's, so building it
    # before knowing whether the A/C winner leaves edge cost on the table is paying
    # an audit cost on speculation. Its trigger is a measured gap here.
    #
    # THE FIXTURE IS THE PART THAT COULD PRODUCE A WRONG VERDICT. At 2048³ the
    # candidates are byte-identical almost everywhere they execute — 2048 is a
    # multiple of both MR and NR, so Sgemm's fringe branch is entered zero times,
    # and the filter therefore does NOT include BenchmarkSgemm. It runs
    # bench/edge_test.go's ragged shapes plus that file's two interior controls,
    # which are the falsifier: **a control delta above its host's between-binary
    # layout floor voids the run**, and the controls are read before any ragged
    # delta is believed. Amended 2026-08-15 on #22: the floor, not a p-value. The
    # floors are 1.71% (Zen 4), 0.99% (Skylake-X), 1.32% (Zen 5) — the largest
    # resolved |sec/op| excursion of the layout ensemble's *control* routine, whose
    # code is identical in both binaries (build/layout-ensemble-e829a61.log,
    # #54/#61). At n=2048 and n=4096 the controls sit in exactly that position, so
    # that log is their denominator; a resolved sub-floor delta with signs
    # disagreeing across hosts is placement, which is what the ensemble exists to
    # have measured in advance of runs like this one.
    #
    # The masked shapes (EdgeSsyrk, EdgeSsymm) are in the filter because the fringe
    # branch is shared with triangular masking: a diagonal tile's live region is a
    # per-row [lo, hi), so the triangular routines take the scratch path at *every*
    # size. That is the half of C's coverage B cannot reach, invisible in any
    # gemm-only fixture.
    AB_TITLE="== edge — issue #22, A vs C. Not a gate: this certifies nothing. =="
    AB_FILTER="${KEEL_EDGE_FILTER:-BenchmarkEdge}"
    AB_TAG="edge"
    AB_BASE_HINT="BASE_REF must contain bench/edge_test.go — commit the fixture before
measuring a candidate against it."
    AB_NOTES=(
      "shapes are bench/edge_test.go's: ragged m/n around MR=2|4 and NR=32 at large k,"
      "the triangular routines' mask-crossing tiles, and interior controls at"
      "n=2048 and n=4096 that MUST come out a wash or the run is void (#22 ruling),"
      "where wash means inside this host's between-binary layout floor -- 1.71/0.99/1.32%"
      "on Zen 4/Skylake-X/Zen 5 from build/layout-ensemble-e829a61.log, not p > 0.05."
    )
    AB_DONE=(
      "done. Read the interior controls first: a non-wash there voids every ragged"
      "delta above it. Both binaries are cross-compiled by the BUILDER's toolchain and"
      "shipped static (scripts/remote.sh:612), so T23's caveat -- a go1.26.5 ranking can"
      "invert on the go1.27 floor (CL 803220) -- is answered by whichever toolchain built"
      "this run, and each log's keel-bench-toolchain: line names it. Not a criterion."
    )
    ;;
  drift)
    # #141's decisive branch: the same build measured twice on one host. BASE_REF
    # defaults to HEAD, so both arms are one source and the only difference between
    # them is elapsed time on the machine — the quantity `-count` cannot reach,
    # since every sample of one benchmark sits in one contiguous window and a
    # within-window interval says nothing about drift between windows. Which is
    # exactly what janus did between two gate runs: w=0.099 against ten other
    # (host, run) cells at 0.000-0.009.
    #
    # Filter and count are the gate's, for comparability with the two archived
    # janus runs. Both ABSOLUTE rates are the product, which neither gate log
    # published: they publish the ratio, and a ratio of two co-moving terms
    # self-cancels a host effect (criterion 5b's rule-12 line in docs/gates.md).
    #
    # Point it at one host: KEEL_REMOTE_HOSTS=janus.local scripts/ab-bench.sh drift
    KEEL_BENCH_COUNT="${KEEL_BENCH_COUNT:-30}"
    AB_BASE="HEAD"
    AB_TITLE="== drift — issue #141. One revision, two windows. Not a gate. =="
    AB_FILTER="${KEEL_DRIFT_FILTER:-BenchmarkKernel|BenchmarkPeak}"
    AB_TAG="drift"
    AB_NOTES=(
      "both arms are the same source, so any delta below is the host between two"
      "windows, not the code: a null A/B whose expected reading is a wash."
      "read the absolute sec/op and GFLOP/s columns, not only the delta -- the ratio"
      "the gate publishes divides two terms a host effect moves together."
    )
    AB_DONE=("done. A wash makes #141's widening a transient excursion; a repeat of it makes it a host property.")
    # A dirty tree makes the two arms differ by whatever is uncommitted, which
    # turns the null A/B into an unlabelled comparison of two builds. Fatal here,
    # merely disclosed in ab_run's banner for the other presets.
    [[ -z "$(git status --porcelain)" ]] ||
      { echo "ab-bench drift: the tree must be clean — both arms are one build." >&2; exit 2; }
    ;;
  *)
    echo "usage: scripts/ab-bench.sh {l1|edge|drift} [BASE_REF]" >&2
    exit 2
    ;;
  esac
}

main() {
  cd "$(dirname "$0")/.."
  ab_preset "${1:-}"
  # shellcheck source=scripts/ab.sh
  source scripts/ab.sh
  ab_run "${2:-$AB_BASE}"
  printf '%s\n' "${AB_DONE[@]}"
}

main "$@"
