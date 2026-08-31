#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# A/B the edge-handling candidates of issue #22: this ref against another.
#
# The measurement the #22 ruling ordered — rank **A** (the incumbent: zero-padded
# panels, a scratch MR×NR tile, and a *scalar* add-back of the live sub-rectangle)
# against **C** (the same kernel and the same scratch tile, with that add-back
# vectorized). **B**, a masked C update inside the microkernel, is deliberately not
# built: its costs are certain (a branch in the hot loop, and double P2's zero-spill
# audit surface) and its coverage is a subset of C's, so building it before knowing
# whether the A/C winner leaves edge cost on the table is paying an audit cost on
# speculation. Its trigger is a measured gap here.
#
# NOT a gate: it certifies nothing, it changes no criterion, and it exits 0 whatever
# it measures. Its product is a table, per host, of the same benchmarks against two
# builds.
#
# # The fixture is the part that could produce a wrong verdict
#
# At 2048³ the candidates are byte-identical almost everywhere they execute — 2048
# is a multiple of both MR and NR, so Sgemm's fringe branch is entered zero times.
# So this script's filter deliberately does NOT include BenchmarkSgemm: it runs
# bench/edge_test.go's ragged shapes, plus that file's two interior controls, which
# are the falsifier. **A control delta above its host's between-binary layout floor
# voids the run** — the controls are read first, before any ragged delta is believed.
#
# Amended 2026-08-15 on #22: the floor, not a p-value. The floors are 1.71% (Zen 4),
# 0.99% (Skylake-X), 1.32% (Zen 5) — the largest resolved |sec/op| excursion of the
# layout ensemble's *control* routine, whose code is identical in both binaries
# (build/layout-ensemble-e829a61.log, #54/#61). At n=2048 and n=4096 the controls are
# in exactly that position, so that log is their denominator; a resolved sub-floor
# delta with signs disagreeing across hosts is placement, which is what the ensemble
# exists to have measured in advance of runs like this one.
#
# The masked shapes (EdgeSsyrk, EdgeSsymm) are in the filter because the fringe
# branch is shared with triangular masking: a diagonal tile's live region is a
# per-row [lo, hi), so the triangular routines take the scratch path at *every* size.
# That is the half of C's coverage B cannot reach, and it is invisible in any
# gemm-only fixture.
#
# The harness is scripts/ab.sh, shared with l1-bench.sh (#131): two builds, the base
# one from a detached worktree at BASE_REF, then every configured host. BASE_REF must
# therefore contain bench/edge_test.go — commit the fixture, then implement C.
#
# usage: scripts/edge-bench.sh [BASE_REF]     (default: HEAD~1)
set -euo pipefail

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/ab.sh
  source scripts/ab.sh

  AB_TITLE="== edge-bench — issue #22, A vs C. Not a gate: this certifies nothing. =="
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
  ab_run "${1:-HEAD~1}"

  echo "done. Read the interior controls first: a non-wash there voids every ragged"
  echo "delta above it. Numbers are this host set at these two builds on the hosts'"
  echo "own go1.26.5; nothing here is a criterion, and a 1.26.x ranking can invert"
  echo "on the go1.27 floor (T23, CL 803220) — the winner is re-measured there."
}

main "$@"
