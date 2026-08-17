#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# exercise-dead-host.sh [HOST] -- drive gate-p2's fleet-incomplete aggregate by
# making one host genuinely unreachable, and nothing else.
#
# WHAT IS BEING EXERCISED -- AND THE TARGET HAS MOVED ONCE ALREADY. This script was
# written to drive criterion 5b's PASS in its fleet-INCOMPLETE rendering:
#
#     "every host that produced a judgeable throughput reading cleared its floor
#      (2/2) ..."      <-- a green line crediting two thirds of the fleet
#
# It fired, printed exactly that, and the reading of it produced #90: a fraction over
# survivors is arithmetically true and reads as fleet-wide. Scott's ruling of
# 2026-08-16 then DELETED that branch -- a fleet with an absent member has not
# measured a claim about the fleet, so a partial fleet resolves to UNMEASURED, as
# criterion 6's aggregate already did. The branch this exercise was built to test no
# longer exists, and the thing that replaced it is now the target:
#
#     "the fleet's floor is unmeasured: 2 of 3 configured hosts cleared it and none
#      measured below it, but 1 produced no floor verdict (0 indeterminate, 1 with no
#      judgeable reading at all) ..."
#
# So the read-back below looks for the UNMEASURED rendering, not the PASS. That is not
# a weakening of what the exercise proves: UNMEASURED sets FAIL, so the induced state
# still blocks green, and the branch under test is still one no complete fleet can
# reach. An unexecuted branch in the instrument that issues the certificates is the
# thing this whole apparatus exists to distrust, and "a host will be down someday"
# means the branch will fire live eventually. Its first firing should not double as
# its first test -- which is exactly why the target moved: the first firing found a
# defect, so a second run is owed to the code that replaced it (ruled 2026-08-16).
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

main() {
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
    echo "   target:         criterion 5b's partial-fleet UNMEASURED (#90's ruling), i.e."
    echo "                   '2 of 3 configured hosts cleared it ... 1 produced no floor"
    echo "                   verdict'. The PASS this exercise first drove -- '(2/2)' -- is"
    echo "                   the branch that ruling deleted; seeing it again would be a"
    echo "                   finding, not a success."
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
  # aggregate line is extracted and its counts checked. Colour codes are stripped
  # because the log carries them.
  #
  # Each of criterion 5b's four branches is matched by a phrase of its own rather than
  # by one loose pattern over all of them, and an unmatched log reports NO. That is the
  # fail-closed direction: if the gate's wording is changed again without this read-back
  # being updated, the exercise says the branch did not fire -- it cannot mistake a
  # different branch for the target. The previous version keyed on "hosts that produced
  # a judgeable throughput reading" and a `(c/j)` fraction, both of which the ruling
  # removed from the target branch; it would have reported INDETERMINATE on the very run
  # meant to prove the fix.
  #
  # ALL NINE OUTCOMES BELOW WERE DRIVEN before this landed, by rendering criterion 5b's
  # five verdict lines out of gate-p2.sh's own bytes (awk over the case block, eval'd with
  # `pass`/`fail`/`unmeasured` stubbed) and feeding each to this block -- so the read-back
  # was tested against the gate's text rather than against my memory of it, which is the
  # failure this whole file documents. The accounting check is redundant on the shipped
  # gate: on the partial branch nmiss == 0, so N_CLEARED == N_JUDGED and
  # N_CLEARED + N_INDET + N_NOCOVER == NHOSTS identically. That is the point of asserting
  # it -- it is a check on the construction staying true, not on this run's arithmetic,
  # and it had to be driven with hand-made counts because the gate cannot produce it.
  echo | tee -a "$LOG"
  CLEAN="$(sed 's/\x1b\[[0-9;]*m//g' "$LOG")"
  nhosts="$(grep -v '^[[:space:]]*#' .keel-hosts 2>/dev/null | grep -c . || echo 0)"
  AGG="$(grep -F 'has not measured a claim about the fleet' <<<"$CLEAN" | tail -1 || true)"
  OTHER="$(grep -E 'did not clear the floor|configured gate hosts cleared its floor|no host produced a judgeable throughput reading|could not read how many hosts were configured' <<<"$CLEAN" | tail -1 || true)"
  {
    echo "-- did the target branch fire? --"
    if [[ -n "$AGG" ]]; then
      echo "   aggregate line: $AGG"
      # Two independent readings of the same line: the cleared/configured pair, and the
      # breakdown of what went unjudged. Both are asserted because the accounting is the
      # part that was wrong before -- a line naming the right verdict over the wrong
      # counts would be the same class of defect one layer in.
      if [[ "$AGG" =~ ([0-9]+)\ of\ ([0-9]+)\ configured\ hosts\ cleared\ it ]]; then
        c="${BASH_REMATCH[1]}"; n="${BASH_REMATCH[2]}"
      else
        c=""; n=""
      fi
      if [[ "$AGG" =~ \(([0-9]+)\ indeterminate,\ ([0-9]+)\ with\ no\ judgeable ]]; then
        ind="${BASH_REMATCH[1]}"; nocov="${BASH_REMATCH[2]}"
      else
        ind=""; nocov=""
      fi
      if [[ -z "$c" || -z "$ind" ]]; then
        echo "   INDETERMINATE: the target verdict fired but its counts could not be read"
        echo "   back (cleared/configured='$c/$n', unjudged breakdown='$ind/$nocov'), so"
        echo "   this run does not establish that the accounting is right."
      elif [[ "$n" != "$nhosts" ]]; then
        echo "   NOT the target rendering: the line says $n configured hosts, .keel-hosts"
        echo "   says $nhosts. The verdict is right and its denominator is not, which is"
        echo "   the defect this branch exists to prevent. Treat as unmeasured."
      elif [[ "$nocov" -lt 1 ]]; then
        echo "   NOT the target rendering: no host is reported as having produced no"
        echo "   judgeable reading, so whatever made this fleet partial, it was not the"
        echo "   induced outage. Treat as unmeasured."
      elif [[ $((c + ind + nocov)) -ne "$nhosts" ]]; then
        echo "   NOT the target rendering: $c cleared + $ind indeterminate + $nocov with no"
        echo "   reading = $((c + ind + nocov)), not the $nhosts configured. A host is"
        echo "   unaccounted for in the very line that claims to account for them."
      else
        echo "   YES: $c of $nhosts configured hosts cleared their floor, $ind were"
        echo "   indeterminate, $nocov produced no judgeable reading, and the verdict is"
        echo "   UNMEASURED rather than a PASS over the survivors. This is the branch that"
        echo "   replaced the one this exercise first drove (#90), and the counts add up."
      fi
    elif [[ -n "$OTHER" ]]; then
      echo "   aggregate line: $OTHER"
      case "$OTHER" in
        *"did not clear the floor"*)
          echo "   NO: the aggregate took its FAIL branch -- a surviving host missed its"
          echo "   floor. That branch already fires routinely, so the target rendering is"
          echo "   still unexercised, AND a real floor miss on a real host is a finding"
          echo "   that outranks this exercise. Read the per-host lines before re-running." ;;
        *"configured gate hosts cleared its floor"*)
          echo "   NO, AND THIS IS A FINDING: the aggregate PASSED over a fleet with an"
          echo "   induced outage. Post-#90 that is unreachable -- a partial fleet cannot"
          echo "   pass -- so either the outage was not induced (the dead host answered)"
          echo "   or fleet_coverage is wrong. Do not read this as a green branch." ;;
        *"no host produced a judgeable throughput reading"*)
          echo "   NO: no host produced a judgeable reading at all, so the fleet was not"
          echo "   partial, it was empty of measurements. The induction hit more than the"
          echo "   one host it was aimed at; check the shim's dead-host match." ;;
        *)
          echo "   NO: the aggregate could not read the fleet's configured size, so it had"
          echo "   no denominator to judge coverage against. Fix .keel-hosts before"
          echo "   spending another run here." ;;
      esac
    else
      echo "   NO: no criterion 5b aggregate line in the log at all. The run did not"
      echo "   reach the aggregate, so this exercise established nothing about it."
    fi
    echo
    echo "instrument exercise: COMPLETE, verdict withheld (gate-p2 exited $RC; 2 is the"
    echo "expected synthetic exit, and it is a fact about this exercise, not about P2)."
    echo "log: $LOG"
  } | tee -a "$LOG"

  exit 2
}

main "$@"
