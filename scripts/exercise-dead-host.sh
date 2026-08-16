#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# exercise-dead-host.sh [HOST] -- drive gate-p2's fleet-incomplete aggregate by
# making one host genuinely unreachable, and nothing else.
#
# WHAT IS BEING EXERCISED. Criterion 5b's aggregate has three branches. Two fire
# routinely: nobody produced a reading (UNMEASURED) and somebody missed the floor
# (FAIL). The third is the PASS
#
#     "every host that produced a judgeable throughput reading cleared its floor
#      (N_CLEARED/N_JUDGED) ..."
#
# and on this fleet it has only ever printed 3/3. Its fleet-INCOMPLETE rendering --
# 2/2 with a third host absent -- has never executed, and that is the rendering
# where a green line credits two thirds of the fleet. An unexecuted branch in the
# instrument that issues the certificates is the thing this whole apparatus exists
# to distrust, and "a host will be down someday" means the branch will fire live
# eventually. Its first firing should not double as its first test (ruled
# 2026-08-16).
#
# WHY A PATH SHIM AND NOT A FLAG. The ruling's first condition: the induction is
# environmental, not a knob in the judging code. A flag that told the gate to skip a
# host would prove the flag works. A shim that makes `ssh` fail proves the GATE
# works -- it discovers the host is unreachable through the same machinery it would
# use on a real outage, from its natural cause, while the other two hosts genuinely
# build, run and measure. The gate is not modified and does not know.
#
# WHAT IT COSTS. Two hosts really measure, so this is a full P2 benchmark run's
# worth of host time (~25 minutes). It is also the fleet's first degraded-mode
# rehearsal, which is worth having independently of the branch.
#
# HOW IT STAYS HONEST. KEEL_INSTRUMENT_EXERCISE is exported, so the gate stamps
# every verdict line [synthetic], prints a banner, withholds GREEN/RED, and exits
# 2. The log goes to build/instrument-exercise-*, never a gate-pN-<rev> path where
# something hungry for a reference could pick it up (#78). This script cannot
# produce a gate result; it has no path that prints one.
#
# Usage:
#   scripts/exercise-dead-host.sh              # kills the third .keel-hosts entry
#   scripts/exercise-dead-host.sh some.host    # kills a named host
#
# Launch it detached, like any long run (CLAUDE.md):
#   scripts/detach.sh run dead-host-<rev> -- ./scripts/exercise-dead-host.sh
set -euo pipefail

cd "$(dirname "$0")/.."

# The host list is machine-local and gitignored; read it, do not hardcode it. The
# third entry by default because that is the ruling's wording, and because the third
# is the one whose absence leaves the other two in the aggregate.
if [[ -n "${1:-}" ]]; then
  DEAD="$1"
else
  DEAD="$(grep -v '^[[:space:]]*#' .keel-hosts 2>/dev/null | grep . | sed -n '3p' || true)"
fi
if [[ -z "$DEAD" ]]; then
  echo "exercise-dead-host: no host to kill (pass one, or configure a third .keel-hosts entry)" >&2
  exit 2
fi

# A shim that refuses one host must still be able to reach the others, so the real
# ssh is resolved BEFORE its directory goes on PATH. Resolving it inside the shim
# would find the shim.
REAL_SSH="$(command -v ssh)"
REAL_SCP="$(command -v scp)"
if [[ -z "$REAL_SSH" || -z "$REAL_SCP" ]]; then
  echo "exercise-dead-host: need both ssh and scp on PATH to shim (ssh='$REAL_SSH' scp='$REAL_SCP')" >&2
  exit 2
fi

# The shim's OWN controls run first, before any host time is spent, for the reason the
# gates run roofline-test.sh before benchmarking: the shim decides what the next 25
# minutes measure. One that refuses too much makes a healthy host look down and the
# branch fires for the wrong reason; one that refuses too little lets the dead host
# answer and the whole exercise is a fiction with a green-looking log.
echo "-- fakessh controls (scripts/fakessh-test.sh) --"
if ! bash scripts/fakessh-test.sh; then
  echo "exercise-dead-host: the shim's own fixtures fail, so nothing it induces would" >&2
  echo "  mean anything. Not spending host time on it." >&2
  exit 2
fi
echo

# The shim itself is scripts/fakessh -- committed, reviewed and fixtured, not a
# heredoc written at launch. Installed under BOTH transport names by symlink so PATH
# resolution finds it either way, with its inputs in the environment: it has no host
# knowledge of its own.
SHIMDIR="$(mktemp -d "${TMPDIR:-/tmp}/keel-deadhost.XXXXXX")"
trap 'rm -rf "$SHIMDIR"' EXIT
ln -s "$PWD/scripts/fakessh" "$SHIMDIR/ssh"
ln -s "$PWD/scripts/fakessh" "$SHIMDIR/scp"

# EVERY TRANSPORT remote.sh USES MUST BE SHIMMED, and this asserts it rather than
# trusting a reading of the file. The first version of this exercise trusted: it
# shimmed ssh only, on the belief that everything crossed the wire through ssh's
# stdin. remote.sh:440 copies the bench binary with `scp`, so the dead host would have
# answered that call and the exercise would have been a fiction with a green-looking
# log. This guard is what caught it, before any host time was spent, which is the
# argument for asserting a premise you are confident about.
#
# It now enumerates what is covered instead of asserting an absence, so a transport
# added later fails here loudly rather than quietly reaching a host declared dead.
UNCOVERED=""
while read -r transport; do
  case "$transport" in
    ssh|scp) : ;;             # shimmed above
    *) UNCOVERED="$UNCOVERED $transport" ;;
  esac
done < <(grep -oE '(^|[^[:alnum:]_./-])(ssh|scp|sftp|rsync|rcp)([^[:alnum:]_-]|$)' scripts/remote.sh \
         | grep -oE '(ssh|scp|sftp|rsync|rcp)' | sort -u)
if [[ -n "$UNCOVERED" ]]; then
  echo "exercise-dead-host: remote.sh uses transports this shim does not cover:$UNCOVERED" >&2
  echo "  The dead host would be reachable by that path and the exercise would be a" >&2
  echo "  fiction. Teach scripts/fakessh the new transport (and give it a fixture)" >&2
  echo "  before spending host time here." >&2
  exit 2
fi

# The collateral scope, computed rather than assumed. P2 re-checks its throughput
# floor on the sentinel host (.keel-sentinel), so if the dead host were the sentinel
# this exercise would knock out a second criterion and the log should say so instead
# of leaving a reader to infer which UNMEASURED lines belong to the induction.
SENTINEL="$(grep -v '^[[:space:]]*#' .keel-sentinel 2>/dev/null | grep . | head -1 || true)"
if [[ -z "$SENTINEL" ]]; then
  SCOPE="the sentinel is unset, so every host is a sentinel and the dead one's absence reaches that criterion too"
elif [[ "$SENTINEL" == "$DEAD" ]]; then
  SCOPE="the dead host IS the sentinel, so the sentinel criterion goes unmeasured as well -- collateral, expected, and not the branch under test"
else
  SCOPE="the sentinel is a different, live host, so the induction is isolated to the fleet aggregate"
fi

REV="$(git rev-parse --short HEAD)"
LOG="build/instrument-exercise-dead-host-$REV.log"
mkdir -p build

{
  echo "== instrument exercise: a dead host, gate-p2 criterion 5b =="
  echo "   rev:            $REV"
  echo "   unreachable:    $DEAD  (via an ssh shim on PATH, exit 255 'No route to host')"
  echo "   ssh underneath: $REAL_SSH"
  echo "   scp underneath: $REAL_SCP  (shimmed too: remote.sh:440 copies the bench binary)"
  echo "   shim dir:       $SHIMDIR"
  echo "   scope:          $SCOPE"
  echo "   target:         the fleet-incomplete rendering of criterion 5b's PASS,"
  echo "                   i.e. N_CLEARED/N_JUDGED reading 2/2 rather than 3/3."
  echo "   NOT a gate run: the verdict is withheld, exit code 2, every verdict"
  echo "                   line stamped [synthetic]."
  echo
} | tee "$LOG"

set +e
PATH="$SHIMDIR:$PATH" \
KEEL_FAKESSH_REAL="$REAL_SSH" KEEL_FAKESCP_REAL="$REAL_SCP" KEEL_FAKESSH_DEAD="$DEAD" \
KEEL_INSTRUMENT_EXERCISE="dead-host:$DEAD" \
  bash scripts/gate-p2.sh 2>&1 | tee -a "$LOG"
RC="${PIPESTATUS[0]}"
set -e

# The read-back on the exercise itself: did the branch actually fire? An exercise
# that reports only "it ran" is the unfired-branch problem one level up, so the
# aggregate line is extracted and its two counts compared. Colour codes are stripped
# because the log carries them.
echo | tee -a "$LOG"
AGG="$(sed 's/\x1b\[[0-9;]*m//g' "$LOG" \
       | grep -E 'hosts? that produced a judgeable throughput reading' | tail -1 || true)"
{
  echo "-- did the target branch fire? --"
  if [[ -z "$AGG" ]]; then
    echo "   NO: no criterion 5b aggregate line in the log at all. The run did not"
    echo "   reach the aggregate, so this exercise established nothing about it."
  else
    echo "   aggregate line: $AGG"
    if [[ "$AGG" == *"did not clear the floor"* ]]; then
      echo "   NO: the aggregate took its FAIL branch -- a surviving host missed its"
      echo "   floor. That branch already fires routinely, so the target rendering is"
      echo "   still unexercised, AND a real floor miss on a real host is a finding"
      echo "   that outranks this exercise. Read the per-host lines before re-running."
    elif [[ "$AGG" =~ \(([0-9]+)/([0-9]+)\) ]]; then
      c="${BASH_REMATCH[1]}"; j="${BASH_REMATCH[2]}"
      nhosts="$(grep -v '^[[:space:]]*#' .keel-hosts 2>/dev/null | grep -c . || echo 0)"
      if [[ "$c" == "$j" && "$j" -lt "$nhosts" ]]; then
        echo "   YES: judged $j of $nhosts configured hosts and credited $c of them."
        echo "   This is the rendering that had never executed: a PASS crediting a"
        echo "   proper subset of the fleet, with the absent host reported separately."
      else
        echo "   NOT the target rendering: cleared=$c judged=$j configured=$nhosts."
        echo "   The exercise ran but did not put the aggregate in the state it was"
        echo "   meant to test; treat this as unmeasured, not as a green branch."
      fi
    else
      echo "   INDETERMINATE: could not read the two counts out of that line."
    fi
  fi
  echo
  echo "instrument exercise: COMPLETE, verdict withheld (gate-p2 exited $RC; 2 is the"
  echo "expected synthetic exit, and it is a fact about this exercise, not about P2)."
  echo "log: $LOG"
} | tee -a "$LOG"

exit 2
