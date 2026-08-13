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
# GOMAXPROCS=1, as every single-thread measurement in this repo does.
#
# usage: scripts/l1-bench.sh [BASE_REF]     (default: HEAD~1)
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

BASE_REF="${1:-HEAD~1}"
FILTER="${KEEL_L1_FILTER:-BenchmarkL1S}"

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

BINDIR="$(mktemp -d)"
WORKTREE="$BINDIR/base"
# The worktree is removed as well as deleted: leaving it registered would make
# `git worktree list` grow one stale entry per run of this script.
trap 'git worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true; rm -rf "$BINDIR"' EXIT

BASE_BIN="$BINDIR/base.test"
NEW_BIN="$BINDIR/new.test"

echo "== l1-bench — issue #47. Not a gate: this certifies nothing. =="
echo

BASE_SHA="$(git rev-parse --short "$BASE_REF")"
NEW_SHA="$(git rev-parse --short HEAD)"
DIRTY=""
git diff --quiet HEAD -- || DIRTY=" + uncommitted changes"
echo "base: $BASE_SHA ($BASE_REF)"
echo "new:  $NEW_SHA (working tree$DIRTY)"

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
info "benchstat compares the two logs directly, so the deltas carry p-values."
info "sizes are bench/bench_test.go's: 1 KB / 16 KB / 256 KB / 4 MB of float32."
echo

while read -r host; do
  [[ -n "$host" ]] || continue
  prov="$(remote_probe "$host" || true)"
  if [[ -z "$prov" ]]; then
    warn "[$host] unreachable, or /proc/cpuinfo unreadable — skipped, and its row is missing rather than estimated"
    continue
  fi
  echo "-- $host --"
  info "$prov"
  ok=1
  for arm in base new; do
    bin="$BASE_BIN"; [[ "$arm" == new ]] && bin="$NEW_BIN"
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$bin" "${BFLAGS[@]}" \
         -test.bench="$FILTER" >"$BINDIR/$arm.log" 2>&1; then
      warn "[$host] the $arm run failed; nothing is reported for this host"
      sed 's/^/        /' "$BINDIR/$arm.log" | tail -20
      ok=0
      break
    fi
  done
  if (( ok )); then
    # benchstat labels the columns from the file names, so the temp files are
    # named for the arms rather than passed as base.txt/new.txt.
    cp "$BINDIR/base.log" "$BINDIR/$BASE_SHA.txt"
    cp "$BINDIR/new.log" "$BINDIR/$NEW_SHA.txt"
    go tool benchstat "$BINDIR/$BASE_SHA.txt" "$BINDIR/$NEW_SHA.txt" 2>&1 | sed 's/^/        /'
  fi
  echo
done <<<"$HOSTS"

echo "done. Numbers above are this host set at these two builds; nothing here is a criterion."
