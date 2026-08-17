#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# remote-exec-test.sh -- drives remote_exec's supervisor (#62) against localhost.
#
# WHY LOCALHOST IS A SUFFICIENT FAR SIDE. Everything this file checks is a property
# of the transport: an exit status recovered from a file, a session killed out from
# under the driver, a severed link, arguments surviving two shells, a missing tmux.
# None of it is a property of the CPU, which is the only thing a real host has that
# this does not. "There is no fleet" was the standing reason remote_exec went
# unsupervised, and for these four branches it was never the right reason.
#
# What localhost cannot supply is a Linux host's PATH. sshd's command PATH is not the
# login shell's, so `ssh localhost 'command -v tmux'` finds nothing though tmux is at
# /opt/homebrew/bin, and the same is true of coreutils' nproc -- both of which live in
# /usr/bin on every host in the fleet. So this script symlinks what it needs into a
# directory that IS on that PATH and removes it afterwards. That is a change to the
# ENVIRONMENT around the shipped function, never to the function: the code under test
# is sourced from scripts/remote.sh and has no test-only branch to be wrong in. Shim
# rather than lower the assertion -- a case that cannot check what it names is a case
# that should say so, and case 6 exists to check the fields #62 appended to.
#
#   ./scripts/remote-exec-test.sh            # all cases
#
# Requires: ssh to localhost working (BatchMode), tmux, and a writable directory on
# sshd's PATH. UNRUN, not failed, if any is missing -- reporting "the supervisor is
# broken" when nothing could test it is the error class DESIGN.md §5.6 forbids.
set -uo pipefail

# `|| exit` because this file deliberately omits `set -e` (a test harness that dies
# on its first failed assertion reports one case instead of six).
cd "$(dirname "$0")/.." || exit 2
# shellcheck source=scripts/remote.sh
. scripts/remote.sh

FAILS=0
pass_() { printf '  ok    %s\n' "$*"; }
fail_() { printf '  FAIL  %s\n' "$*"; FAILS=$((FAILS + 1)); }
info_() { printf '        %s\n' "$*"; }
head_() { printf '\n%s\n' "$*"; }

H=localhost
KEEL_REMOTE_DIR="/tmp/keel-remote-test-$$"
SHIM=""        # the tmux shim specifically: case 5 hides and restores this one
SHIMS=()       # everything placed on sshd's PATH, removed on exit

cleanup() {
  local s
  for s in ${SHIMS+"${SHIMS[@]}"}; do rm -f "$s"; done
  ssh "${KEEL_SSH_OPTS[@]}" "$H" "rm -rf '$KEEL_REMOTE_DIR'" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}

# ------------------------------------------------------------------- preflight
WORK="$(mktemp -d)"
trap cleanup EXIT

if ! ssh "${KEEL_SSH_OPTS[@]}" "$H" true 2>/dev/null; then
  echo "remote-exec-test: cannot ssh to $H (BatchMode). Enable Remote Login." >&2
  echo "remote-exec-test: UNRUN (no verdict -- nothing was exercised)" >&2
  exit 2
fi
for t in tmux nproc; do
  command -v "$t" >/dev/null 2>&1 && continue
  echo "remote-exec-test: no $t on this machine." >&2
  echo "remote-exec-test: UNRUN (no verdict -- nothing was exercised)" >&2
  exit 2
done
LOCAL_TMUX="$(command -v tmux)"
# A directory on sshd's PATH that this user can write. Chosen by asking sshd for its
# PATH rather than by guessing: the guess is what makes a test like this pass on one
# machine and skip on another for reasons nobody can see.
SSHD_PATH="$(ssh "${KEEL_SSH_OPTS[@]}" "$H" 'echo "$PATH"' 2>/dev/null || true)"
SHIMDIR=""
while IFS= read -r d; do
  [[ -d "$d" && -w "$d" ]] || continue
  SHIMDIR="$d"; break
done < <(tr ':' '\n' <<<"$SSHD_PATH")
if [[ -z "$SHIMDIR" ]]; then
  echo "remote-exec-test: no writable directory on sshd's PATH ($SSHD_PATH)." >&2
  echo "remote-exec-test: UNRUN (no verdict -- nothing was exercised)" >&2
  exit 2
fi
for t in tmux nproc; do
  ssh "${KEEL_SSH_OPTS[@]}" "$H" "command -v $t >/dev/null 2>&1" && continue
  ln -sf "$(command -v "$t")" "$SHIMDIR/$t"
  SHIMS+=("$SHIMDIR/$t")
  [[ "$t" == tmux ]] && SHIM="$SHIMDIR/tmux"
  ssh "${KEEL_SSH_OPTS[@]}" "$H" "command -v $t >/dev/null 2>&1" ||
    { echo "remote-exec-test: shim at $SHIMDIR/$t still not visible to sshd" >&2; exit 2; }
done
head_ "0. preflight"
pass_ "ssh $H, tmux $("$LOCAL_TMUX" -V | awk '{print $2}'), far side $(ssh "${KEEL_SSH_OPTS[@]}" "$H" 'echo $0')"
[[ "${#SHIMS[@]}" -eq 0 ]] || info_ "shimmed onto sshd's PATH for this run: ${SHIMS[*]}"

# A stand-in for a test binary: prints to both streams, echoes its argv so the
# quoting can be inspected, sleeps if told to, and exits with a status it is given.
# It is a shell script rather than a Go binary because remote_exec ships and runs
# whatever it is handed; making it a compiled thing would test the compiler.
BIN="$WORK/fake-bench"
cat > "$BIN" <<'EOF'
#!/bin/sh
printf 'argv:'; for a in "$@"; do printf ' [%s]' "$a"; done; printf '\n'
printf 'to stderr\n' >&2
[ -z "${FAKE_SLEEP:-}" ] || sleep "$FAKE_SLEEP"
printf 'trailing line with no newline after it'
exit "${FAKE_RC:-0}"
EOF
chmod +x "$BIN"

# --------------------------------------------------- 1. the ordinary case
head_ "1. a run that finishes"
# NOT `out="$(remote_exec ...)"`: a command substitution is a subshell, so every
# REMOTE_* variable it sets dies with it and the parent reads a STALE value from an
# earlier case. That is how this file first reported "expected a supervised run, got
# REMOTE_SUPERVISED=" against a function that had set it correctly — and worse, case 5
# read `yes` left over from case 2 and would have passed had the assertion been weaker.
# Every case here runs remote_exec in this shell and redirects to a file.
remote_exec "$H" "$BIN" -test.bench='Sgemm|Ssyrk' -test.count=2 > "$WORK/1.log" 2>&1; rc=$?
out="$(cat "$WORK/1.log")"
[[ "$REMOTE_SUPERVISED" == yes ]] && pass_ "ran supervised (REMOTE_SUPERVISED=yes)" ||
  fail_ "expected a supervised run, got REMOTE_SUPERVISED=$REMOTE_SUPERVISED"
[[ "$rc" -eq 0 && "$REMOTE_STATE" == ok ]] && pass_ "exit 0, REMOTE_STATE=ok" ||
  fail_ "expected exit 0 / ok, got exit $rc / $REMOTE_STATE"
# The hazard #62 named: a tmux layer is a second shell on the far side, so an
# argument that survived printf %q once must survive it twice. 'Sgemm|Ssyrk' as
# shell input is a pipeline -- the failure that made printf %q necessary at all.
if grep -q 'argv: \[-test\.bench=Sgemm|Ssyrk\] \[-test\.count=2\]' <<<"$out"; then
  pass_ "argv crossed two shells intact: $(grep -o 'argv:.*' <<<"$out")"
else
  fail_ "argv was altered on the far side:"; sed 's/^/        /' <<<"$out"
fi
grep -q 'to stderr' <<<"$out" && pass_ "stderr is in the log" || fail_ "stderr was lost"

# Byte-exactness, checked because the log is benchstat's and the marker greps'
# input. A `$(...)` capture anywhere in the path would silently drop the final
# newline of every log in the project.
remote_exec "$H" "$BIN" > "$WORK/raw.log" 2>&1
if cmp -s <(printf 'argv:\nto stderr\ntrailing line with no newline after it') "$WORK/raw.log"; then
  pass_ "log came back byte-exact, including a final line with no newline"
else
  fail_ "log was edited in transit:"; od -c "$WORK/raw.log" | sed 's/^/        /' | tail -3
fi

head_ "2. a run that fails"
FAKE_RC=7 KEEL_REMOTE_ENV="FAKE_RC=7" remote_exec "$H" "$BIN" >/dev/null 2>&1; rc=$?
[[ "$rc" -eq 7 && "$REMOTE_STATE" == ok ]] && pass_ "exit 7 recovered from the status file, REMOTE_STATE=ok" ||
  fail_ "expected exit 7 / ok, got exit $rc / $REMOTE_STATE"
remote_vanished && fail_ "remote_vanished is true for a run that finished" ||
  pass_ "remote_vanished is false: a nonzero status is a claim about the program"

# ------------------------------------- 3. the session killed under the driver
#
# The acceptance bar's second clause: a killed session must be reported as
# vanished, never as an exit code. Driven on purpose, because an unchanged green
# run proves nothing about a branch nothing entered.
head_ "3. the session is killed mid-run"
KEEL_REMOTE_ENV="FAKE_SLEEP=30" remote_exec "$H" "$BIN" >/dev/null 2>&1 &
ep=$!
killed=no
for _ in $(seq 1 40); do
  s="$(ssh "${KEEL_SSH_OPTS[@]}" "$H" "tmux ls 2>/dev/null | sed -n 's/^\(keel-[0-9]*-[0-9]*\):.*/\1/p' | head -1" 2>/dev/null || true)"
  if [[ -n "$s" ]]; then
    ssh "${KEEL_SSH_OPTS[@]}" "$H" "tmux kill-session -t '$s'" >/dev/null 2>&1 && killed=yes
    break
  fi
  sleep 1
done
wait "$ep"; rc=$?
if [[ "$killed" != yes ]]; then
  fail_ "could not find the remote session to kill, so this case proved nothing"
elif [[ "$rc" -eq "$REMOTE_EXIT_VANISHED" ]]; then
  # The subshell means REMOTE_STATE is not visible here; the exit code is the
  # part of the contract that crosses the process boundary, and it is the part
  # a caller would otherwise misread as the program's.
  pass_ "killed session returned \$REMOTE_EXIT_VANISHED ($rc), not a program exit code"
else
  fail_ "killed session returned $rc, which a caller would read as the program's status"
fi

# ---------------------------------- 3b. the predicate the gates actually branch on
#
# Case 3 proves the exit code crosses a process boundary; it cannot prove
# remote_vanished, because the subshell that ran remote_exec took REMOTE_STATE with
# it. The five gate sites do not read the exit code -- they call remote_vanished --
# and until here that function was only ever asserted FALSE (case 2). An assertion
# that only sees a predicate's false branch is the shape that let the KEEL_FORCE
# fail-open stand for as long as it did. So: kill from the background, measure in
# THIS shell.
head_ "3b. remote_vanished is true in the caller's own shell"
( for _ in $(seq 1 40); do
    s="$(ssh "${KEEL_SSH_OPTS[@]}" "$H" "tmux ls 2>/dev/null | sed -n 's/^\(keel-[0-9]*-[0-9]*\):.*/\1/p' | head -1" 2>/dev/null || true)"
    [[ -z "$s" ]] || { ssh "${KEEL_SSH_OPTS[@]}" "$H" "tmux kill-session -t '$s'" >/dev/null 2>&1; break; }
    sleep 1
  done ) &
kp=$!
KEEL_REMOTE_ENV="FAKE_SLEEP=30" remote_exec "$H" "$BIN" >/dev/null 2>&1; rc=$?
wait "$kp" 2>/dev/null
if [[ "$rc" -ne "$REMOTE_EXIT_VANISHED" ]]; then
  fail_ "the session was not killed in time (exit $rc), so this case proved nothing"
elif remote_vanished; then
  pass_ "remote_vanished is true, so the gates' vanished branch is reachable (REMOTE_STATE=$REMOTE_STATE)"
else
  fail_ "remote_vanished is FALSE after a vanished run: every gate would read it as the program's failure"
fi

# ------------------------------------------ 4. the link severed under the driver
#
# The bar's first clause: the measurement survives a deliberately severed ssh and
# its status is still recoverable afterward. The wait connection is killed from
# outside -- SIGKILL to the ssh client, which is what a dropped link looks like to
# this driver -- while the far side keeps computing.
head_ "4. the ssh connection is severed mid-run"
KEEL_REMOTE_ENV="FAKE_SLEEP=8" remote_exec "$H" "$BIN" > "$WORK/severed.log" 2>&1 &
ep=$!
severed=no
for _ in $(seq 1 40); do
  # the wait connection, identified by the has-session loop in its command line
  p="$(pgrep -f "tmux has-session -t 'keel-" 2>/dev/null | head -1 || true)"
  if [[ -n "$p" ]]; then kill -9 "$p" 2>/dev/null && severed=yes; break; fi
  sleep 1
done
wait "$ep"; rc=$?
if [[ "$severed" != yes ]]; then
  fail_ "never saw a wait connection to sever, so this case proved nothing"
elif [[ "$rc" -eq 0 ]] && grep -q 'argv:' "$WORK/severed.log"; then
  pass_ "measurement survived the severed link, reconnected, and its exit status came back 0"
else
  fail_ "severed link cost the measurement: exit $rc, log $(wc -l < "$WORK/severed.log" | tr -d ' ') line(s)"
fi

# --------------------------------------------------- 5. no supervisor present
#
# Not a failure and not an exemption: the run still happens, and the fact that it
# was unsupervised is recorded rather than assumed away. Driven by hiding tmux from
# the far side's PATH the same way the environment hid it to begin with.
head_ "5. no usable tmux on the far side"
if [[ -n "$SHIM" ]]; then
  mv "$SHIM" "$SHIM.hidden"
  remote_exec "$H" "$BIN" -test.count=1 > "$WORK/5.log" 2>&1; rc=$?
  out="$(cat "$WORK/5.log")"
  mv "$SHIM.hidden" "$SHIM"
  [[ "$REMOTE_SUPERVISED" == no ]] && pass_ "REMOTE_SUPERVISED=no — the absence is recorded, not silent" ||
    fail_ "expected REMOTE_SUPERVISED=no, got '$REMOTE_SUPERVISED'"
  [[ "$rc" -eq 0 && "$REMOTE_STATE" == ok ]] && pass_ "unsupervised run still measured: exit 0, REMOTE_STATE=ok" ||
    fail_ "unsupervised run lost its result: exit $rc / $REMOTE_STATE"
  grep -q 'argv: \[-test\.count=1\]' <<<"$out" && pass_ "unsupervised log is the program's output" ||
    fail_ "unsupervised log is wrong:"
else
  info_ "skipped: tmux is on sshd's PATH natively here, so it cannot be hidden without"
  info_ "touching a system directory. The branch is exercised wherever the shim was needed."
fi

# --------------------------------------------------- 6. the provenance field
head_ "6. tmux= reaches the provenance line"
prov="$(remote_probe "$H" || true)"
info_ "${prov:-no reading}"
grep -q 'tmux=yes' <<<"$prov" && pass_ "provenance carries tmux=yes" ||
  fail_ "provenance has no tmux=yes field, so an absent supervisor would be silent"
# The three field parsers that read this line, re-checked because #62 appended to it.
[[ "$(sed -n 's/.*| \([0-9]*\) cpus |.*/\1/p' <<<"$prov")" =~ ^[0-9]+$ ]] &&
  pass_ "gate-p5's cpus parser still resolves" || fail_ "gate-p5's cpus parser broke"
assert_governor "$H" preamble "$prov"
[[ -n "$GOV_STATE" ]] && pass_ "assert_governor still reads the line (GOV_STATE=$GOV_STATE)" ||
  fail_ "assert_governor read nothing from the line"

# ------------------------------------- 7. every governor state, driven on purpose
#
# §5 rule 5 forbids "no cpufreq interface" and "the file is present and unreadable"
# sharing a verdict, so the two must be shown to LAND DIFFERENTLY -- and this machine
# can only produce one of them. PROV is a documented parameter of assert_governor
# precisely so a caller that has already probed need not probe twice; handing it a
# synthetic reading drives every branch with no host and no test-only code path. What
# it does not prove is that a real guest emits `governor=absent`, which is a claim
# about the probe and is checked above only for THIS machine's shape (also absent).
head_ "7. the five governor states are distinguishable"
gov_case() {
  local want="$1" line="$2"
  assert_governor fixture preamble "$line" >/dev/null 2>&1
  [[ "$GOV_STATE" == "$want" ]] && pass_ "'$line' -> $GOV_STATE" ||
    fail_ "'$line' -> $GOV_STATE, expected $want"
}
gov_case performance "x | governor=performance | y"
gov_case wrong       "x | governor=powersave | y"
gov_case nocpufreq   "x | governor=absent | y"
gov_case unreadable  "x | governor=unreadable | y"
gov_case unreadable  "x | governor=unknown | y"
gov_case unreachable ""
# The one that would silently undo the split: `absent` must not survive into a log
# line as though it were a value somebody read off the host.
assert_governor fixture preamble "x | governor=absent | y" >/dev/null 2>&1
[[ -z "$GOV_VALUE" ]] && pass_ "a guest reports no governor VALUE, only a state" ||
  fail_ "GOV_VALUE='$GOV_VALUE' would print as a reading nobody took"

head_ "verdict"
if [[ "$FAILS" -eq 0 ]]; then
  echo "  GREEN -- a finished run reports its own exit code, a killed one reports"
  echo "  vanished, a severed link costs nothing, a missing supervisor is loud, and"
  echo "  a host with no cpufreq is told apart from one whose knob will not read."
  exit 0
fi
echo "  RED -- $FAILS check(s) failed."
exit 1
