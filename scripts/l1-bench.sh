#!/usr/bin/env bash
# A/B the Level-1 routines across the cache hierarchy: this ref against another.
#
# The measurement for issue #47 — internal/l1's loops were reshaped from
# index-driven to slice-advancing, which removed every surviving bounds check and
# roughly halved the six unrolled reductions' instruction counts, while making the
# four non-unrolled loops (Axpy, Scal, both widths) 1-5 instructions *longer* for
# the reason docs/toolchain-notes.md T19 gives. Static counts cannot say what
# either of those is worth, because these routines are not all issue-bound: at
# n = 1<<20 they are reading 4 MB per call and the loop body is not the limit.
#
# So this is NOT a gate: it certifies nothing, it changes no criterion, and it
# exits 0 whatever it measures. Its product is a table, per host, of the same
# benchmark run against two builds.
#
# The four sizes in bench/bench_test.go's `sizes` are the whole point of running
# this at all: 256, 4096, 65536 and 1<<20 float32s are 1 KB, 16 KB, 256 KB and
# 4 MB, i.e. L1-, L2-, L3- and memory-resident on all three hosts in
# docs/hosts.md. A loop-body change should show up at 1 KB and wash out at 4 MB,
# and if it does not, the explanation is not "the loop got shorter".
#
# The base build comes from a detached git worktree at BASE_REF, not from
# stashing: a stash would mutate the tree another long-running measurement may be
# reading, and it would make the two arms differ by whatever else was dirty.
# The new build comes from the working tree, so an uncommitted change can be
# measured before it is committed.
#
# The comparison goes through bench_compare rather than calling benchstat
# directly. This script's first run is why that function exists: benchstat groups
# by configuration, keel's provenance preamble is in that namespace, and one
# volatile key in it (keel-bench-clock-mhz, a live snapshot) silently turned every
# A/B into two independent one-column tables with no delta in them — issue #50,
# docs/toolchain-notes.md T20. The output looked like a comparison. A comparison
# that did not happen is now a failed measurement and says so.
#
# GOMAXPROCS=1, as every single-thread measurement in this repo does.
#
# usage: scripts/l1-bench.sh [BASE_REF]     (default: HEAD~1)
set -euo pipefail

# Everything below is a function definition and the last line of the file is
# `main "$@"`. Bash reads a script incrementally as it executes it, so a script
# that does its work at the top level can be corrupted by an edit that lands
# mid-run: the parser resumes at a byte offset that now holds different text.
# Defining everything before anything runs forces one whole-file parse before the
# first host is touched, which makes the instrument immune instead of leaving
# "never edit a running instrument" as a rule someone has to remember (#51).

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

# ab_host HOST — both arms on one host, then the comparison.
#
# Both arms run back to back on the same machine before anything is compared, so
# the two builds see the same thermal and residency state as nearly as this setup
# can arrange. A host whose first arm fails reports nothing at all rather than half
# a table.
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
  # benchstat labels the columns from the file names, so the temp files are named
  # for the two commits rather than passed as base.txt/new.txt.
  cp "$BINDIR/base.log" "$BINDIR/$BASE_SHA.txt"
  cp "$BINDIR/new.log" "$BINDIR/$NEW_SHA.txt"
  # The `set -o pipefail` at the top of the file is what makes this test
  # bench_compare's verdict rather than sed's: without it the indent pipe would
  # swallow the one status that says whether a comparison happened.
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
  FILTER="${KEEL_L1_FILTER:-BenchmarkL1S}"

  BINDIR="$(mktemp -d)"
  local WORKTREE="$BINDIR/base"
  # The worktree is removed as well as deleted: leaving it registered would make
  # `git worktree list` grow one stale entry per run of this script.
  trap 'git worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true; rm -rf "$BINDIR"' EXIT

  BASE_BIN="$BINDIR/base.test"
  NEW_BIN="$BINDIR/new.test"

  echo "== l1-bench — issue #47. Not a gate: this certifies nothing. =="
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
  info "sizes are bench/bench_test.go's: 1 KB / 16 KB / 256 KB / 4 MB of float32."
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

  echo "done. Numbers above are this host set at these two builds; nothing here is a criterion."
}

main "$@"
