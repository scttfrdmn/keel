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
# # Mechanism, and why it is l1-bench.sh's
#
# The base build comes from a detached git worktree at BASE_REF, not from stashing:
# a stash would mutate the tree another long-running measurement may be reading, and
# it would make the two arms differ by whatever else was dirty. The new build comes
# from the working tree, so candidate C can be measured before it is committed.
# BASE_REF must therefore contain bench/edge_test.go — commit the fixture, then
# implement C.
#
# The comparison goes through bench_compare rather than calling benchstat directly:
# benchstat groups by configuration, keel's provenance preamble is in that
# namespace, and one volatile key in it silently turns an A/B into two independent
# one-column tables with no delta in them (#50, T20). A comparison that did not
# happen is a failed measurement and says so.
#
# GOMAXPROCS=1, as every single-thread measurement in this repo does: the candidates
# differ in a per-tile copy, and letting the thread count vary would put the answer
# inside the scheduler's noise.
#
# usage: scripts/edge-bench.sh [BASE_REF]     (default: HEAD~1)
set -euo pipefail

# Everything below is a function definition and the last line of the file is
# `main "$@"`. Bash reads a script incrementally as it executes it, so a script that
# does its work at the top level can be corrupted by an edit that lands mid-run:
# the parser resumes at a byte offset that now holds different text. Defining
# everything before anything runs forces one whole-file parse before the first host
# is touched (#51).

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

# ab_host HOST — both arms on one host, then the comparison.
#
# Both arms run back to back on the same machine before anything is compared, so the
# two builds see the same thermal and residency state as nearly as this setup can
# arrange. A host whose first arm fails reports nothing at all rather than half a
# table.
ab_host() {
  local host="$1" arm bin
  for arm in base new; do
    bin="$BASE_BIN"
    [[ "$arm" == new ]] && bin="$NEW_BIN"
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$bin" "${BFLAGS[@]}" \
         -test.bench="$FILTER" >"$BINDIR/$arm.log" 2>&1; then
      warn "[$host] the $arm run failed; nothing is reported for this host"
      sed 's/^/        /' "$BINDIR/$arm.log" | tail -20
      return 0
    fi
  done
  cp "$BINDIR/base.log" "$BINDIR/$BASE_SHA.txt"
  cp "$BINDIR/new.log" "$BINDIR/$NEW_SHA.txt"
  # Archived under build/ so the ranking can be recomputed from the logs later
  # rather than from this script's stdout (DESIGN.md §5.8).
  cp "$BINDIR/base.log" "build/edge-$host-$BASE_SHA.log"
  cp "$BINDIR/new.log" "build/edge-$host-$NEW_SHA.log"
  if ! bench_compare "$BINDIR/$BASE_SHA.txt" "$BINDIR/$NEW_SHA.txt" | sed 's/^/        /'; then
    warn "[$host] the two arms were not compared, so this host contributes no delta"
  fi
}

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/remote.sh
  source scripts/remote.sh
  # shellcheck source=scripts/bench.sh
  source scripts/bench.sh

  local BASE_REF="${1:-HEAD~1}"
  FILTER="${KEEL_EDGE_FILTER:-BenchmarkEdge}"

  BINDIR="$(mktemp -d)"
  # Deliberately not `local`: the EXIT trap below runs in global scope, after this
  # function has returned, so a local would be unbound there — and under `set -u`
  # that aborts the trap on its first command, leaking both the worktree and BINDIR
  # while exiting 1 on a successful run (#55).
  WORKTREE="$BINDIR/base"
  trap "git worktree remove --force '$WORKTREE' >/dev/null 2>&1 || true; rm -rf '$BINDIR'" EXIT

  BASE_BIN="$BINDIR/base.test"
  NEW_BIN="$BINDIR/new.test"
  mkdir -p build

  echo "== edge-bench — issue #22, A vs C. Not a gate: this certifies nothing. =="
  echo

  BASE_SHA="$(git rev-parse --short "$BASE_REF")"
  NEW_SHA="$(git rev-parse --short HEAD)"
  local DIRTY=""
  git diff --quiet HEAD -- || DIRTY=" + uncommitted changes"
  echo "base: $BASE_SHA ($BASE_REF)"
  echo "new:  $NEW_SHA (working tree$DIRTY)"

  local HOSTS
  HOSTS="$(remote_hosts)"
  if [[ -z "$HOSTS" ]]; then
    echo "no execution hosts configured (.keel-hosts or \$KEEL_REMOTE_HOSTS)." >&2
    echo "simd/archsimd is amd64-only (T1), so there is nothing to measure here." >&2
    exit 2
  fi

  git worktree add --detach "$WORKTREE" "$BASE_REF" >/dev/null
  if ! ( cd "$WORKTREE" && remote_build_test ./bench "$BASE_BIN" ) >"$BINDIR/build" 2>&1; then
    echo "cross-compile of the base bench binary failed:" >&2
    sed 's/^/  /' "$BINDIR/build" >&2
    echo "BASE_REF must contain bench/edge_test.go — commit the fixture before" >&2
    echo "measuring a candidate against it." >&2
    exit 2
  fi
  if ! remote_build_test ./bench "$NEW_BIN" >"$BINDIR/build" 2>&1; then
    echo "cross-compile of the new bench binary failed:" >&2
    sed 's/^/  /' "$BINDIR/build" >&2
    exit 2
  fi
  echo "built two linux/amd64 bench binaries (static)"

  mapfile -t BFLAGS < <(bench_flags)
  echo "methodology: -count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, GOMAXPROCS=1, filter=$FILTER"
  info "benchstat compares the two logs, ignoring $KEEL_BENCH_IGNORE — configuration"
  info "keys that describe the run rather than the build, and which forked the table"
  info "into two delta-free halves when they were not ignored (#50, T20)."
  info "shapes are bench/edge_test.go's: ragged m/n around MR=2|4 and NR=32 at large k,"
  info "the triangular routines' mask-crossing tiles, and interior controls at"
  info "n=2048 and n=4096 that MUST come out a wash or the run is void (#22 ruling),"
  info "where wash means inside this host's between-binary layout floor -- 1.71/0.99/1.32%"
  info "on Zen 4/Skylake-X/Zen 5 from build/layout-ensemble-e829a61.log, not p > 0.05."
  echo

  local host prov
  while read -r host; do
    [[ -n "$host" ]] || continue
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      warn "[$host] unreachable, or /proc/cpuinfo unreadable — skipped, and its row is missing rather than estimated"
      continue
    fi
    echo "-- $host --"
    info "$prov"
    ab_host "$host"
    echo
  done <<<"$HOSTS"

  echo "done. Read the interior controls first: a non-wash there voids every ragged"
  echo "delta above it. Numbers are this host set at these two builds on the hosts'"
  echo "own go1.26.5; nothing here is a criterion, and a 1.26.x ranking can invert"
  echo "on the go1.27 floor (T23, CL 803220) — the winner is re-measured there."
}

main "$@"
