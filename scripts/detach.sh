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
#   scripts/detach.sh wait NAME                    block until it ends, then stat
#   scripts/detach.sh log  NAME                    print the log path
#   scripts/detach.sh kill NAME                    stop it
#
# `wait` blocks on a tmux wait-for channel rather than polling, so waiting costs
# nothing and adds no latency to noticing. It is the sleep-free half of the same
# rule: a `sleep 60` loop both burns time and reports late.
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

  # THE CALLER'S ENVIRONMENT IS CARRIED, and the runner file is the record of it.
  # `tmux new-session` seeds a session from the SERVER's environment, and with
  # `exit-empty off` below pinning that server forever, the server's env stays whatever
  # the first detached run of the machine's life happened to hold. Both directions bite
  # and they are SEPARATE defects: an override the caller sets is dropped unless this
  # very call starts the server (cost a gate-p5 pre-flight, 2026-08-28), and a var the
  # caller does NOT set is inherited from the server and outranks the run's own
  # configuration (cost the first release run, 2026-08-29: a stale
  # `KEEL_REMOTE_HOSTS=antares` beat the `.keel-hosts` `aws-fleet.sh up` had just
  # written, so a $24/hr three-host fleet idled while the gate measured a lab box).
  # So the runner CLEARS the whole KEEL_/BENCH_ namespace before re-exporting the
  # carried set: the run is then the program that was launched — the one thing this
  # script exists to guarantee — and `build/<name>.cmd` a COMPLETE statement of it, not
  # just of the deltas. PATH is carried because it picks the `go` building the arms.
  local carried=() v
  for v in PATH GOEXPERIMENT GOMAXPROCS $(compgen -v | grep -E '^(KEEL_|BENCH_)' || true); do
    [[ -n "${!v+set}" ]] && carried+=("$v")
  done

  # The command is written to a runner script rather than interpolated into
  # tmux's argument, which a shell on the far side would re-expand. Same
  # hazard, and the same fix, as remote_exec's printf %q (scripts/remote.sh).
  #
  # The runner signals a wait-for channel on the way out, so `wait` can block
  # instead of polling. Two properties of the signalling matter:
  #
  #   - it fires on death by signal as well as on normal exit, or a `wait` on a
  #     killed run would block forever;
  #   - it does NOT write the status file when it dies by signal. A killed run
  #     has no exit code to report, and manufacturing one is the single failure
  #     mode this script exists to prevent (DESIGN.md §5.6). `wait` therefore
  #     hands the verdict to `stat`, which says `died` (it ran and was killed) or
  #     `never-started` (nothing under this NAME ever ran) -- two words, because
  #     one word for both left the reader unable to tell which (#122).
  {
    echo '#!/usr/bin/env bash'
    printf 'cd %q || exit 3\n' "$ROOT"
    echo 'for v in $(compgen -v | grep -E "^(KEEL_|BENCH_)" || true); do unset "$v"; done'
    for v in "${carried[@]}"; do printf 'export %s=%q\n' "$v" "${!v}"; done
    # THE FILE CHANNEL, ENUMERATED (2026-08-31, #146(c), ruled). The clear above makes
    # this file a complete statement of one channel, and a total restatement is only as
    # total as the set of channels it enumerates: `.keel-sentinel` — gitignored,
    # machine-local, three weeks old — outranked the fleet for a JUDGED criterion on the
    # run that signed v0.1.0-a2, and an `unset` loop cannot reach a file. So every
    # `.keel-*` decision file is copied in verbatim, with its mtime, as comments the
    # runner never reads. By GLOB and not by list, because a hardcoded list is how the
    # next decision file goes unenumerated; this is rule 21 and rule 22 meeting from
    # opposite ends — the certificate's input closure is what the declaration must
    # STATE, not merely what the clear must reset.
    # Verbatim and UNFILTERED, including files git could recover on its own: a filter is
    # where the silence comes back, and the one it would have excused here is `.keel-hosts`.
    printf '# -- input closure, FILE channel: %s/.keel-* verbatim (comments only) --\n' "$ROOT"
    local cf
    for cf in "$ROOT"/.keel-*; do
      [[ -f "$cf" ]] || continue
      printf '#   %s  (mtime %s)\n' "${cf##*/}" "$(date -u -r "$cf" +%Y-%m-%dT%H:%M:%SZ)"
      awk '{print "#     | " $0}' "$cf"
    done
    # The DEFAULTS channel, which is the third one and cannot be enumerated here: what a gate
    # decides when neither an env var nor a file says otherwise is a property of the TREE, and
    # this script has no business knowing any gate's defaults. Naming the revision states that
    # channel completely and by reference — and it is also the only place the .cmd said which
    # tree it launched, which the frozen-tree rule makes the identity of the run.
    printf '# -- input closure, DEFAULTS channel: this tree, by reference --\n#   revision: %s %s\n' \
      "$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo not-a-git-tree)" \
      "$(git -C "$ROOT" diff --quiet HEAD 2>/dev/null && echo '(clean)' || echo '(DIRTY: the defaults in effect are not any committed revision)')"
    printf 'signalled=\n'
    # shellcheck disable=SC2016  # deliberately unexpanded: this is the runner's source
    printf 'signal() { [[ -n "$signalled" ]] && return 0; signalled=1; tmux wait-for -S %q 2>/dev/null || true; }\n' "$sess"
    printf 'trap signal EXIT\n'
    printf 'trap "signal; exit 143" HUP TERM INT\n'
    local a
    for a in "$@"; do printf '%q ' "$a"; done
    echo
    printf 'echo $? > %q\n' "$status"
  } >"$runner"
  chmod +x "$runner"

  tmux new-session -d -s "$sess" "$(printf '%q > %q 2>&1' "$runner" "$log")"

  # exit-empty defaults on, which would take the server down when this session
  # is the last one to end -- and a recorded wait-for signal lives in the
  # server, so it would go with it. A later `wait` would then start a fresh
  # server and block on a channel nobody will ever signal. Keeping the server
  # up is what makes `wait` safe to call after the run has already finished.
  tmux set-option -s exit-empty off 2>/dev/null || true

  printf 'detached: session=%s log=%s\n' "$sess" "$log"
  # Names only, values in the runner file: the failure this replaces was silent, so a
  # caller who names a fleet should see it echoed back.
  printf 'carried:  %s\n' "${carried[*]}"
  printf 'status:   %s (absent until it finishes)\n' "$status"
}

cmd_wait() {
  local name="$1" sess; sess="$(sess_of "$1")"
  # Both fast paths matter. If it already finished, waiting would block on a
  # channel whose signal may have been consumed by an earlier `wait`; if it
  # already died, there is nothing left to signal at all.
  if [[ -r "$DIR/$name.status" ]] || ! tmux has-session -t "=$sess" 2>/dev/null; then
    cmd_stat "$name"
    return
  fi
  tmux wait-for "$sess" 2>/dev/null || true
  cmd_stat "$name"
}

cmd_stat() {
  local name="$1" sess; sess="$(sess_of "$1")"
  local log="$DIR/$name.log" status="$DIR/$name.status"
  if tmux has-session -t "=$sess" 2>/dev/null; then
    printf 'running  %s  (%s lines so far)\n' "$sess" "$(wc -l <"$log" 2>/dev/null | tr -d ' ' || echo 0)"
    return 0
  fi
  if [[ -r "$status" ]]; then
    printf 'exited   %s  status=%s  log=%s\n' "$sess" "$(cat "$status")" "$log"
    return 0
  fi
  # ONE WORD FOR TWO FACTS (#122). "no session and no status file" had two causes and
  # `vanished ... killed or never started` named both at once, so the reader could not tell
  # which had happened -- and they call for opposite next actions. Work that started and was
  # killed has a log to read and host-minutes already spent; work that never started has
  # nothing, and the usual cause is a NAME that does not match what was launched. That is not
  # hypothetical: querying `validate113-ba6f286` as `keel-validate113-ba6f286` reported this
  # line for a run that was healthy and 25 minutes in. Both still return 1 -- a run without a
  # status file is unmeasured either way, and an exit code is what neither of them has.
  local runner="$DIR/$name.cmd" hint=""
  if [[ -e "$log" ]]; then
    printf 'died     %s  no status file after %s log line(s): it started and was killed, so it has no exit code and inventing one is the failure mode this script exists to prevent  log=%s\n' \
      "$sess" "$(wc -l <"$log" 2>/dev/null | tr -d ' ' || echo 0)" "$log"
    return 1
  fi
  [[ -e "$runner" ]] && hint=" ($runner does exist, so a launch under this exact name was attempted and tmux never started it)"
  # The prefix is HINTED AT, never stripped. `keel-foo` is a legal NAME, so a stat that
  # rewrote or rejected one would break a run that is merely named that way -- which is why
  # the fix for the false alarm above is a sentence and not a transformation.
  printf 'never-started %s  no session, no status file, no log%s. NAME is always looked up as keel-<NAME>, so a NAME that already begins with keel- is queried as %s -- if that is what happened, drop the prefix\n' \
    "$sess" "$hint" "$sess"
  return 1
}

main() {
  local sub="${1:-}"; shift || true
  case "$sub" in
    run)  [[ $# -ge 1 ]] || die "usage: detach.sh run NAME -- CMD..."; cmd_run "$@" ;;
    stat) [[ $# -eq 1 ]] || die "usage: detach.sh stat NAME"; cmd_stat "$1" ;;
    wait) [[ $# -eq 1 ]] || die "usage: detach.sh wait NAME"; cmd_wait "$1" ;;
    log)  [[ $# -eq 1 ]] || die "usage: detach.sh log NAME"; printf '%s\n' "$DIR/$1.log" ;;
    # Signal the channel here as well as from the runner's trap. The trap is the
    # normal path, but it depends on the killed shell getting to run it; this
    # does not, and a `wait` that outlives its run is the one bug that cannot be
    # noticed by looking at the log.
    kill) [[ $# -eq 1 ]] || die "usage: detach.sh kill NAME"
          tmux kill-session -t "=$(sess_of "$1")"
          tmux wait-for -S "$(sess_of "$1")" 2>/dev/null || true ;;
    *)    die "usage: detach.sh {run|stat|wait|log|kill} NAME [-- CMD...]" ;;
  esac
}

main "$@"
