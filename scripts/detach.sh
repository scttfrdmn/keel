#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# detach.sh — run a long command in a detached tmux session so that nothing
# about the *caller's* lifetime can truncate it.
#
# # Why this exists
#
# Two gate-p5 runs on 2026-08-12 were killed 25-28 minutes in, partway through
# the nested gate-p3 of the carried p5->p4->p3->p2 chain, producing
# byte-identical truncated logs. The diagnosis at the time was a lifetime limit
# on the agent's background shell, and the workaround was to hand the run to a
# human. That was the wrong conclusion to draw: a measurement whose completion
# depends on who typed the command is a measurement with a defect in its
# harness, not a scheduling problem.
#
# tmux fixes it at the root. `tmux new-session -d` starts (or reuses) a server
# that daemonises — it leaves the caller's process group and session entirely —
# so reaping the caller, by signal, by group, or by hangup, does not reach the
# work. The log keeps growing and the exit status still lands.
#
# This matters most for the *remote* half. scripts/remote.sh runs each benchmark
# synchronously over ssh, so a dropped connection SIGHUPs a measurement
# mid-flight on the far side; the gate then reports missing rows, which its own
# vocabulary correctly calls unmeasured, but the host-minutes are gone. Keeping
# the local driver alive keeps every ssh under it alive.
#
# # Usage
#
#   scripts/detach.sh run  NAME -- CMD [ARGS...]   start CMD detached
#   scripts/detach.sh stat NAME                    running, or the exit code
#   scripts/detach.sh log  NAME                    print the log path
#   scripts/detach.sh kill NAME                    stop it
#
# NAME picks the log: build/NAME.log, plus build/NAME.status once it finishes.
# Include the revision in NAME, as the gates' own logs do:
#
#   scripts/detach.sh run gate-p5-3f70ae4 -- ./scripts/gate-p5.sh
#   scripts/detach.sh stat gate-p5-3f70ae4
#
# # What this does not do
#
# It does not make a run reproducible or a log a verdict. A detached run is
# still one invocation at one commit, and the tree must stay frozen for its
# whole life for the same reasons a foreground run must.

set -euo pipefail

die() { printf 'detach.sh: %s\n' "$1" >&2; exit 2; }

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIR="$ROOT/build"

command -v tmux >/dev/null || die "tmux is not installed; that is the whole mechanism"

sess_of() { printf 'keel-%s' "$1"; }

cmd_run() {
  local name="$1"; shift
  [[ "${1:-}" == "--" ]] || die "expected -- before the command"
  shift
  [[ $# -gt 0 ]] || die "no command given"

  local sess; sess="$(sess_of "$name")"
  if tmux has-session -t "=$sess" 2>/dev/null; then
    die "session $sess is already running; 'stat' it or 'kill' it first"
  fi

  mkdir -p "$DIR"
  local log="$DIR/$name.log" status="$DIR/$name.status" runner="$DIR/$name.cmd"
  rm -f "$status"

  # The command is written to a runner script rather than interpolated into
  # tmux's argument, which a shell on the far side would re-expand. Same
  # hazard, and the same fix, as remote_exec's printf %q (scripts/remote.sh).
  {
    echo '#!/usr/bin/env bash'
    printf 'cd %q || exit 3\n' "$ROOT"
    local a
    for a in "$@"; do printf '%q ' "$a"; done
    echo
    printf 'echo $? > %q\n' "$status"
  } >"$runner"
  chmod +x "$runner"

  tmux new-session -d -s "$sess" "$(printf '%q > %q 2>&1' "$runner" "$log")"

  printf 'detached: session=%s log=%s\n' "$sess" "$log"
  printf 'status:   %s (absent until it finishes)\n' "$status"
}

cmd_stat() {
  local name="$1" sess; sess="$(sess_of "$1")"
  local log="$DIR/$name.log" status="$DIR/$name.status"
  if tmux has-session -t "=$sess" 2>/dev/null; then
    printf 'running  %s  (%s lines so far)\n' "$sess" "$(wc -l <"$log" 2>/dev/null || echo 0)"
    return 0
  fi
  if [[ -r "$status" ]]; then
    printf 'exited   %s  status=%s  log=%s\n' "$sess" "$(cat "$status")" "$log"
    return 0
  fi
  # No session and no status file: the runner never wrote one, so the work was
  # killed rather than finished. Reporting that as an exit code would be a lie.
  printf 'vanished %s  no status file: killed, not finished  log=%s\n' "$sess" "$log"
  return 1
}

main() {
  local sub="${1:-}"; shift || true
  case "$sub" in
    run)  [[ $# -ge 1 ]] || die "usage: detach.sh run NAME -- CMD..."; cmd_run "$@" ;;
    stat) [[ $# -eq 1 ]] || die "usage: detach.sh stat NAME"; cmd_stat "$1" ;;
    log)  [[ $# -eq 1 ]] || die "usage: detach.sh log NAME"; printf '%s\n' "$DIR/$1.log" ;;
    kill) [[ $# -eq 1 ]] || die "usage: detach.sh kill NAME"; tmux kill-session -t "=$(sess_of "$1")" ;;
    *)    die "usage: detach.sh {run|stat|log|kill} NAME [-- CMD...]" ;;
  esac
}

main "$@"
