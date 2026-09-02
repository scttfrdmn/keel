#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #148's decisive test 1: KEEL_PIN_WIDTH in {1,2,4,8} against the scalar rows, one
# binary, to separate "monotone in width" (contention or frequency) from "a step
# between 8 and everything else" (neither). The registered prediction is that the
# small-kc rows read within 1.15x of their width-8 rate at WIDTH 4.
#
# This driver is committed BEFORE the run rather than written into /tmp mid-flight,
# which is what archive/addr147/driver-addr147.sh had to do. The design is therefore
# in the tree, at a hash, before any of its data exists — the cheapest possible proof
# that the two decisions below predate the numbers they will be read against. It is
# under archive/ and not scripts/: the apparatus cap is on scripts/, and an
# evidence-directory driver for one run is not apparatus anyone else calls.
#
# TWO DECISIONS, DECLARED HERE BECAUSE THEY ARE NOT IN THE PRE-REGISTRATION:
#
# 1. TWO PASSES, THE SECOND IN REVERSED WIDTH ORDER (8 4 2 1, then 1 2 4 8).
#    #148 registered a sweep, and the verdict it asks for is the SHAPE of the
#    width-to-rate curve. A single time-ordered sweep cannot deliver that: thermal
#    drift or a co-tenant arriving mid-run is also monotone in time, and in a
#    1,2,4,8 pass that is indistinguishable from monotone in width. Reversing the
#    second pass makes the two confounds separable — a real width effect agrees
#    between passes, and drift disagrees in the direction that tracks the clock
#    rather than the mask. Cost is one extra hour on an idle lab host.
#
# 2. ALL 20 ROWS, NOT THE REGISTERED 8. The criterion is still evaluated on the
#    8 scalar rows and nothing else. The 12 avx512 rows are a CONTROL, declared as
#    one here, before the run: #147 measured every avx512 row within 0.97-1.10 of
#    its width-8 median, so if they move in this sweep the finding is a property of
#    the host or the window and not of the scalar kernel. A superset of the
#    registered rows cannot weaken the registered criterion; it can only supply the
#    reading that says whether the criterion was measured on a quiet machine.
#
# Recording load and frequency around every arm is #148's decisive test 3 as far as
# it can be taken without a scripts/ change: the harness's provenance block still
# records no co-tenant load (#81), but this driver's own transcript can, and the
# reason the 3.7-4.2x it is chasing is weaker than it should be is precisely that
# #147's window cannot be shown to have been quiet afterwards. Samples are taken
# BEFORE and AFTER each arm and never during one — reading the host mid-measurement
# is the contamination this whole file exists to rule out.
#
# No `set -e`: every step reports its own rc rather than vanishing, and a killed run
# must not be able to look like a verdict.
set -uo pipefail

cd /Users/scttfrdmn/src/keel || exit 3

say() { printf '\n=== %s ===\n' "$*"; }

REV="$(git rev-parse --short HEAD)"
FULLREV="$(git rev-parse HEAD)"
HOST="${HOST:-janus.local}"
# Overridable so a smoke run can exercise THIS script rather than a second,
# differently-worded copy of it. The judged values are the defaults and the log
# records which ones were used.
FILTER="${FILTER:-BenchmarkKernel}"
COUNT="${COUNT:-30}"
BTIME="${BTIME:-1s}"
WIDTHS_A="${WIDTHS_A:-8 4 2 1}"
WIDTHS_B="${WIDTHS_B:-1 2 4 8}"
TAG="${TAG:-}"
OUT="build"

say "provenance and the frozen-tree guard"
date -u +%FT%TZ
echo "settings: HOST=$HOST FILTER=$FILTER COUNT=$COUNT BTIME=$BTIME"
echo "settings: WIDTHS_A=[$WIDTHS_A] WIDTHS_B=[$WIDTHS_B] TAG=${TAG:-<none>}"
echo "driver host: $(hostname)"
echo "rev: $FULLREV"
# The tmux server this may run under inherits an environment, and a stale host list
# in it once turned a healthy run RED with zero FAILs. Print what is set, so a later
# reader can see what the run was actually told rather than what it was meant to be.
echo "inherited KEEL_*/HOST env: $(env | /usr/bin/grep -E '^(KEEL|HOST)=' | tr '\n' ' ' || true)"
dirty="$(git status --porcelain)"
if [[ -n "$dirty" ]]; then
  echo "REFUSED: the tree is dirty, so a log from this run could not be attributed to a revision."
  sed 's/^/  /' <<<"$dirty"
  exit 4
fi
echo "tree clean: yes"

# shellcheck source=/dev/null
source scripts/remote.sh
echo "sourced scripts/remote.sh; KEEL_REMOTE_DIR=$KEEL_REMOTE_DIR; default KEEL_PIN_WIDTH=$KEEL_PIN_WIDTH"
echo "go: $(go version)"

say "build: ONE binary, used by ALL arms"
# One binary across every arm is the point: the arms differ only in the affinity
# mask, so a layout difference cannot be mistaken for a width effect (#141).
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN"
rc=$?
echo "remote_build_test rc=$rc"
[[ "$rc" -eq 0 ]] || { echo "no binary, no arms"; exit 5; }
sha="$({ shasum -a 256 "$BIN" 2>/dev/null || sha256sum "$BIN"; } | cut -c1-16)"
echo "binary: sha256=${sha} bytes=$(wc -c <"$BIN" | tr -d ' ') flags=[$(build_settings "$BIN")]"
echo "toolchain read off the artifact: $(builder_toolchain "$BIN")"
# #147's arms used sha256=d0d46d26c15cc8b2 at 537661a. Whether this rebuild matches
# is a fact to record, not a requirement: the three commits since 537661a touched
# docs and archive/ only, so a match makes this sweep's w8 and w1 arms further draws
# of #147's own arms, and a mismatch means they are merely comparable.
if [[ "$sha" == d0d46d26c15cc8b2 ]]; then
  echo "binary IDENTICAL to #147's traced binary: this sweep's w8/w1 arms are further draws of the same artifact"
else
  echo "binary DIFFERS from #147's d0d46d26c15cc8b2: cross-run comparisons carry a between-binary layout term (#54/#61: 1.71/0.99/1.32%)"
fi

# hostsample LABEL -- load, top consumers and per-core frequency, from the far side.
# Called only between arms. cpu0-7 covers every core any of these masks can select.
hostsample() {
  echo "-- host sample: $1 --"
  ssh "${KEEL_SSH_OPTS[@]}" "$HOST" '
    printf "uptime: "; uptime
    printf "loadavg: "; cat /proc/loadavg
    printf "freq_khz cpu0-7:"; for c in 0 1 2 3 4 5 6 7; do printf " %s" "$(cat /sys/devices/system/cpu/cpu$c/cpufreq/scaling_cur_freq 2>/dev/null || echo NA)"; done; echo
    printf "governor: "; cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo NA
    echo "top by cpu:"; ps -eo pid,pcpu,etimes,comm --sort=-pcpu | head -4 | sed "s/^/  /"
    printf "utc: "; date -u +%FT%TZ' 2>&1 | sed 's/^/   /'
}

arm() {
  local pass="$1" width="$2" label="$1$2" log rc
  log="$OUT/bench-width148-$REV$TAG-$label.txt"
  say "arm $label: pass $pass, KEEL_PIN_WIDTH=$width"
  hostsample "before $label"

  KEEL_PIN_WIDTH="$width" \
  KEEL_REMOTE_ENV="GOMAXPROCS=1" \
    remote_exec "$HOST" "$BIN" \
      -test.run=NONE -test.bench="$FILTER" -test.count="$COUNT" -test.benchtime="$BTIME" \
      > "$log" 2>&1
  rc=$?
  echo "arm $label: REMOTE_STATE=$REMOTE_STATE REMOTE_SUPERVISED=$REMOTE_SUPERVISED rc=$rc"
  if remote_vanished; then
    echo "arm $label: UNMEASURED -- the far side never reported a status, so rc=$rc is not an exit code"
    hostsample "after $label (unmeasured)"
    return 0
  fi
  echo "arm $label: log rows: $(/usr/bin/grep -c '^BenchmarkKernel' "$log")"
  echo "arm $label: scalar rows: $(/usr/bin/grep -c '^BenchmarkKernel.*/scalar/' "$log")"
  # The mask readback, not the mask requested: GOMAXPROCS is 1 in every arm, so the
  # width is NOT recoverable from the row names here the way it is in a gate run.
  # This line is the only witness that arm w4 was four cores and not eight.
  echo "arm $label: keel-pin line: $(/usr/bin/grep -m1 '^keel-pin:' "$log")"
  echo "arm $label: gomaxprocs line: $(/usr/bin/grep -m1 '^keel-bench-gomaxprocs:' "$log")"
  hostsample "after $label"
}

for w in $WIDTHS_A; do arm a "$w"; done
for w in $WIDTHS_B; do arm b "$w"; done

say "done"
date -u +%FT%TZ
