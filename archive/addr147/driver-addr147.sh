#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
# #147's two decisive tests, one run: the allocation-placement trace (hypothesis 1)
# and a width-1 mask (hypothesis 2). Lives in /tmp rather than scripts/ because the
# apparatus cap has no paid session to spend on a driver; the commands are recorded
# verbatim in the tracked log this produces, which is the same trade D1 made.
#
# No `set -e`: every step reports its own rc rather than vanishing, and a killed run
# must not be able to look like a verdict.
set -uo pipefail

cd /Users/scttfrdmn/src/keel || exit 3

say() { printf '\n=== %s ===\n' "$*"; }

REV="$(git rev-parse --short HEAD)"
FULLREV="$(git rev-parse HEAD)"
HOST="${HOST:-janus.local}"
# Overridable so the smoke run can exercise this exact script against the real host
# cheaply instead of a second, differently-worded copy of it. The judged values are
# the defaults, and the log records which ones were used.
FILTER="${FILTER:-BenchmarkKernel}"
COUNT="${COUNT:-30}"
BTIME="${BTIME:-1s}"
TAG="${TAG:-}"

say "provenance and the frozen-tree guard"
date -u +%FT%TZ
echo "settings: HOST=$HOST FILTER=$FILTER COUNT=$COUNT BTIME=$BTIME TAG=${TAG:-<none>}"
echo "driver host: $(hostname)"
echo "rev: $FULLREV"
dirty="$(git status --porcelain)"
if [[ -n "$dirty" ]]; then
  echo "REFUSED: the tree is dirty, so a log from this run could not be attributed to a revision."
  sed 's/^/  /' <<<"$dirty"
  exit 4
fi
echo "tree clean: yes"
if [[ "$REV" != 537661a ]]; then
  echo "NOTE: HEAD is $REV, not the 537661a the pre-registration names. Recorded, not overridden."
fi

# shellcheck source=/dev/null
source scripts/remote.sh
echo "sourced scripts/remote.sh; KEEL_REMOTE_DIR=$KEEL_REMOTE_DIR; default KEEL_PIN_WIDTH=$KEEL_PIN_WIDTH"
echo "go: $(go version)"

say "build: one binary, used by BOTH arms"
# One binary for both arms is the point: the arms differ ONLY in the affinity mask,
# so a layout difference cannot be mistaken for a pinning effect (#141's lesson).
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN"
rc=$?
echo "remote_build_test rc=$rc"
[[ "$rc" -eq 0 ]] || { echo "no binary, no arms"; exit 5; }
# Digest inline: ab_arm_provenance lives in scripts/ab.sh, which this driver does not
# source (it wants bench.sh and a main() too). build_settings is remote.sh's own.
sha="$({ shasum -a 256 "$BIN" 2>/dev/null || sha256sum "$BIN"; } | cut -c1-16)"
echo "traced binary: sha256=${sha} bytes=$(wc -c <"$BIN" | tr -d ' ') flags=[$(build_settings "$BIN")]"
echo "toolchain read off the artifact: $(builder_toolchain "$BIN")"

arm() {
  local label="$1" width="$2" trace="$KEEL_REMOTE_DIR/addrtrace-$1.txt" rc
  say "arm $label: KEEL_PIN_WIDTH=$width, trace -> $trace"

  # Append mode means a stale file from a previous arm would silently merge into
  # this one's analysis. Remove it and read the removal back.
  ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "mkdir -p '$KEEL_REMOTE_DIR'; rm -f '$trace'; ls -l '$trace' 2>&1 | tail -1"
  echo "pre-arm trace file removed (the line above should say No such file)"

  KEEL_PIN_WIDTH="$width" \
  KEEL_REMOTE_ENV="GOMAXPROCS=1 KEEL_ADDR_TRACE=$trace" \
    remote_exec "$HOST" "$BIN" \
      -test.run=NONE -test.bench="$FILTER" -test.count="$COUNT" -test.benchtime="$BTIME" \
      > "build/addr147-$REV$TAG-$label.log" 2>&1
  rc=$?
  echo "arm $label: REMOTE_STATE=$REMOTE_STATE REMOTE_SUPERVISED=$REMOTE_SUPERVISED rc=$rc"
  if remote_vanished; then
    echo "arm $label: UNMEASURED -- the far side never reported a status, so rc=$rc is not an exit code"
    return 0
  fi
  echo "arm $label: log rows: $(/usr/bin/grep -c '^BenchmarkKernel' "build/addr147-$REV$TAG-$label.log")"
  echo "arm $label: keel-pin line: $(/usr/bin/grep -m1 '^keel-pin:' "build/addr147-$REV$TAG-$label.log")"
  # A log with trace lines in it would mean the stderr splice is back.
  echo "arm $label: trace lines INSIDE the bench log (must be 0): $(/usr/bin/grep -c 'keel-addrtrace' "build/addr147-$REV$TAG-$label.log")"

  scp -q "${KEEL_SCP_OPTS[@]}" "$HOST:$trace" "build/addr147-$REV$TAG-$label-trace.txt"
  local srr=$?
  echo "arm $label: scp trace rc=$srr lines=$(/usr/bin/grep -c . "build/addr147-$REV$TAG-$label-trace.txt" 2>/dev/null || echo 0)"
}

# Arm 1 first: it is the arm whose conditions match this issue's three draws, so if
# only one arm survives, the one that survives is the comparable one.
arm w8 8
arm w1 1

say "done"
date -u +%FT%TZ
