#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The null-change witness for the #155 gate arm64 port (docs/gate-arm64-port.md, unit 1).
#
# WHAT IT IS. The #155 ruling is that the gate's rendering on amd64 evidence must be
# byte-identical before and after the arm64 port. Two live gate runs cannot prove that --
# benchmark numbers vary run to run, so their renderings differ from noise, not code. The
# proof needs FIXED input: capture a run's host-side outputs once, then re-drive the gate's
# analysis and rendering against them. Two replays of one corpus render identically, so
# diffing them ACROSS a gate edit isolates exactly what the edit changed. That diff, proven
# non-vacuous (a planted change must show), is the witness.
#
# HOW. Sourced by the gates AFTER remote.sh + gate-lib.sh. Inert unless KEEL_REPLAY is set,
# so a normal run is byte-unchanged: with KEEL_REPLAY empty this file defines nothing and
# returns before touching a single name.
#
#   KEEL_REPLAY=record : run the real host calls AND save each output under KEEL_REPLAY_DIR,
#                        keyed by CALL IDENTITY (host + binary + -test.bench/-test.run +
#                        KEEL_FORCE), not by sequence -- so a corpus entry maps to a call by
#                        what it is, and a reorder or an arg change shows up as a miss rather
#                        than a silently-shifted replay.
#   KEEL_REPLAY=replay : return the saved output instead of touching a host, so the gate runs
#                        entirely locally against fixed input. A call with no corpus entry
#                        returns a fixed miss (rc 99, no output), which the gate renders as
#                        `unmeasured` deterministically -- identical in every replay, so it
#                        cancels in the before/after diff and leaves only what the port moved.
#
# SCOPE (DESIGN.md rule 12). It replays the HOST-TOUCHING calls the port's rendering flows
# from: remote_exec (the bench sweep; and, when recorded, the parallel-correctness binary and
# the forced-backend runs), remote_probe (the CPU-model key and the admission provenance).
# The governor/clock preconditions and remote_build_test are stubbed benign and SILENT: the
# port does not change them, so their absence from both replay renderings cancels. What the
# witness therefore does NOT see: whether a real host's governor/clock gate correctly, the
# -race leg (a native compile), and any accuracy of the numbers (a replayed archive is fixed
# input, not a measurement). Those are covered by the live amd64 baseline pair, or are out of
# the port's scope. A partial corpus is legal and expected: the throughput witness supplies
# only the sweep archive, so the dispatch/forced calls render `unmeasured` in both replays.

[[ -n "${KEEL_REPLAY:-}" ]] || return 0
: "${KEEL_REPLAY_DIR:?KEEL_REPLAY is set but KEEL_REPLAY_DIR is not — nowhere to record to or replay from}"
case "$KEEL_REPLAY" in
  record | replay) ;;
  *) printf 'gate-replay: KEEL_REPLAY=%s is neither record nor replay\n' "$KEEL_REPLAY" >&2; exit 2 ;;
esac
mkdir -p "$KEEL_REPLAY_DIR"

# _copyfn SRC DST — clone shell function SRC to name DST, so the real implementation survives
# being shadowed. `declare -f` prints "name ()\n{ … }"; dropping line 1 leaves the body.
_copyfn() { eval "${2}() $(declare -f "$1" | sed '1d')"; }

# _replay_key HOST BINARY ARGS... — a call's identity. KEEL_REMOTE_ENV carries KEEL_FORCE=…,
# which is how the forced-backend runs differ from the default one at the same binary; the
# bench/run filters distinguish the sweep from the correctness binary. Everything else about a
# call (ssh options, absolute temp paths) is host- and run-specific noise and deliberately out.
_replay_key() {
  local host="$1" bin sig a
  bin="$(basename "${2:-nobin}")"
  sig="${KEEL_REMOTE_ENV:-}|$host|$bin"
  shift 2 || true
  for a in "$@"; do
    case "$a" in -test.bench=* | -test.run=* | -test.v | -test.count=* | -test.benchtime=*) sig="$sig|$a" ;; esac
  done
  printf '%s' "$sig" | shasum -a 256 2>/dev/null | cut -c1-16 || printf '%s' "$sig" | sha256sum | cut -c1-16
}

_copyfn remote_exec _real_remote_exec
_copyfn remote_probe _real_remote_probe

if [[ "$KEEL_REPLAY" = record ]]; then
  # Capture real outputs; the preconditions and the local build are bypassed so the record
  # run completes on any reachable host regardless of its governor (the corpus needs fixed
  # DATA, not a passing precondition — accuracy is not what the witness tests).
  remote_exec() {
    local key out rc
    key="$(_replay_key "$@")"
    out="$(_real_remote_exec "$@")"; rc=$?
    printf '%s\n' "$out" >"$KEEL_REPLAY_DIR/exec-$key.out"
    printf '%s\n' "$rc" >"$KEEL_REPLAY_DIR/exec-$key.rc"
    printf '%s\n' "$*" >"$KEEL_REPLAY_DIR/exec-$key.cmd"
    printf '%s\n' "$out"
    return "$rc"
  }
  remote_probe() {
    local out; out="$(_real_remote_probe "$@")"
    printf '%s\n' "$out" >"$KEEL_REPLAY_DIR/probe-$1.out"
    printf '%s\n' "$out"
  }
else
  # replay: fixed input, no host contact. A missing corpus entry is a fixed miss.
  remote_exec() {
    local key f rcf
    key="$(_replay_key "$@")"
    f="$KEEL_REPLAY_DIR/exec-$key.out"; rcf="$KEEL_REPLAY_DIR/exec-$key.rc"
    if [[ ! -f "$f" ]]; then
      # A miss is a fixed, silent rc 99 (the gate renders `unmeasured`, identically in both
      # replays). The attempted key+command is logged for bootstrapping a corpus by hand.
      printf '%s\t%s\n' "$key" "$*" >>"$KEEL_REPLAY_DIR/_misses.log"
      return 99
    fi
    cat "$f"
    return "$(cat "$rcf" 2>/dev/null || printf 0)"
  }
  remote_probe() {
    local f="$KEEL_REPLAY_DIR/probe-$1.out"
    [[ -f "$f" ]] && cat "$f"
    return 0
  }
  # Only in REPLAY: skip the local cross-compile. There is no host to ship to, so the binary is
  # never used and building it only costs time. In RECORD the real build MUST run — the real
  # remote_exec it feeds needs a real binary to ship to the host.
  remote_build_test() { : >"${2:-/dev/null}" 2>/dev/null || true; return 0; }
  remote_build_test_or_fail() { return 0; }
fi

# Preconditions: benign and silent in BOTH modes, so a record run reaches every remote_exec
# call regardless of the host's clock (the corpus is host-governor-independent), and the two
# replays carry no precondition text to differ on. remote_vanished never fires in replay (a
# miss is a fixed rc 99, handled above).
# shellcheck disable=SC2034  # GOV_STATE/GOV_SHOWN are read by the sourcing gate, not here
assert_governor() { GOV_STATE=performance; GOV_SHOWN=performance; return 0; }
clock_gate() { return 0; }
clock_head() { return 0; }
clock_post() { return 0; }
remote_vanished() { return 1; }

printf 'gate-replay: %s mode, corpus %s\n' "$KEEL_REPLAY" "$KEEL_REPLAY_DIR" >&2
