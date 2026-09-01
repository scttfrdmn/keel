#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# detach-test.sh — detach.sh's incident log, made executable.
#
# Scope, ruled 2026-08-31 and deliberately narrow. This file encodes only failures that
# ACTUALLY HAPPENED, and every one is shown to FAIL FIRST: each case runs twice, once
# against a MUTANT copy with the fixing line reverted — where the arm must reproduce the
# incident — and once against the shipped script, where it must not. A harness test that has
# never failed proves only that the harness cannot be tested, and detach.sh is the harness
# whose failures are denominated in dollars rather than lines: case 2 below billed a
# three-host fleet for eight idle hours.
#
# The mutation reverts a line rather than checking out an old revision, so each arm names the
# line that fixes it. Same behaviour either way; only this form survives a rebase.
#
# NOT COVERED, stated rather than implied. The fourth incident of that week — a 45-second-old
# log read as a seven-hour stall — was a defect in an ad-hoc PROBE of a detached run, not in
# this script: detach.sh has no mtime, age or timezone logic at all (grep finds such logic in
# bench.sh, exercise-baseline.sh and gate-p5.sh, so the zero here is a reading). There is no
# behaviour in this file to drive, so it gets no arm. Giving `stat` a self-describing age of
# its own would remove the class, and that is a change to propose, not to smuggle in here.
#
# ALSO NOT COVERED, and this one is #122's own list rather than the incident log: item 3, that the
# stated exceptions PATH/GOEXPERIMENT/GOMAXPROCS survive the namespace clear. It has never failed,
# and the only-what-happened scope excludes it; a regression there breaks every run loudly rather
# than quietly, which is why it is the cheapest of the four to leave out. Named, not implied.

# shellcheck disable=SC2015  # `cond && ok || bad` is if-then-else HERE: ok and bad both end in
# printf, which cannot fail, so the C arm never runs on a true A. Keeps each arm to one line.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
P=0; F=0
ok()  { P=$((P + 1)); printf 'ok    %s\n' "$1"; }
bad() { F=$((F + 1)); printf 'FAIL  %s\n' "$1"; }
# Exits 0 without tmux so a machine lacking the mechanism does not fail `make lint` — but says
# so in the count, because a skip that reads like a pass is the thing this file exists against.
command -v tmux >/dev/null || { printf 'SKIP  tmux absent, so NONE of the arms below ran; detach.sh is untested on this machine\n'; exit 0; }

# detach.sh derives ROOT from its own path, so a copy under a temporary `scripts/` puts every
# build/ artifact in the temp tree: this test cannot litter the repo it is testing. One parent
# temp dir, because mkroot is called inside `$(...)` and an array appended there is appended in
# a subshell and lost -- which would leak a root per arm.
# -P because on macOS mktemp yields /var/folders/... while tmux reports the resolved
# /private/var/folders/..., so the isolation arm below compared two spellings of one path and
# read as a leak. Found by that arm going red on a correctly isolated run.
BASE="$(cd "$(mktemp -d)" && pwd -P)" || exit 1
# A PRIVATE tmux server, per #122: "the defect *was* the server, so a fake server is a fake
# test" -- and equally, the operator's real server must be neither read nor perturbed, because
# case 2 below is BORN of a global variable set on a shared server. Setting one on the server
# that is hosting a live gate run would reach every session it starts afterwards. TMUX_TMPDIR
# moves the socket for this shell AND for the detach.sh children it spawns, which is what makes
# one variable enough; kill-server then takes the whole thing with it.
export TMUX_TMPDIR="$BASE/tmux"; mkdir -p "$TMUX_TMPDIR"
mkroot() { # mkroot [SED-EXPR] -> path to a root whose detach.sh is optionally mutated
  local t; t="$(mktemp -d "$BASE/rXXXXXX")"; mkdir -p "$t/scripts"
  if [[ $# -gt 0 ]]; then sed -E "$1" "$ROOT/scripts/detach.sh" >"$t/scripts/detach.sh"
  else cp "$ROOT/scripts/detach.sh" "$t/scripts/detach.sh"; fi
  chmod +x "$t/scripts/detach.sh"; printf '%s\n' "$t"
}
cleanup() { tmux kill-server 2>/dev/null; rm -rf "$BASE"; }
trap cleanup EXIT

# Every assertion below matches a CAPTURED string, never a pipeline's exit status. `stat`
# returns 1 for both unmeasured verdicts by design, and under `pipefail` that 1 becomes the
# status of `stat | grep -q` even when grep matched -- so a piped assertion reads false on a
# correct script. Found here, by this file's own fail-first arm.
saw() { # saw ROOT NAME -> the value the RUN itself saw, read from its own log
  local r="$1" n="$2"
  # shellcheck disable=SC2016  # deliberately unexpanded: the RUN must expand it, not this shell
  "$r/scripts/detach.sh" run "$n" -- sh -c 'echo "SAW=[${KEEL_TEST_FLEET-}]"' >/dev/null 2>&1
  "$r/scripts/detach.sh" wait "$n" >/dev/null 2>&1
  sed -n 's/^SAW=\[\(.*\)\]$/\1/p' "$r/build/$n.log" 2>/dev/null
}

# ---- incident 1: THE DROPPED OVERRIDE (2026-08-28, cost a gate-p5 pre-flight).
# `tmux new-session` seeds from the SERVER's environment, so a var the caller exports is
# invisible to the run unless the runner file re-exports it. The fix is detach.sh:117.
export KEEL_TEST_FLEET=alpha
m="$(mkroot 's/^ *for v in "\$\{carried\[@\]\}".*$//')"
[[ "$(saw "$m" "t1m-$$")" == "" ]] && ok "1 fails first: with :117 reverted the caller's KEEL_TEST_FLEET=alpha never reaches the run" \
                                    || bad "1 mutant did not reproduce the incident, so the arm below proves nothing"
[[ "$(saw "$(mkroot)" "t1r-$$")" == alpha ]] && ok "1 fixed: the shipped runner carries KEEL_TEST_FLEET=alpha into the run" \
                                             || bad "1 the caller's override is being dropped"
unset KEEL_TEST_FLEET

# ---- incident 2: THE INJECTED OVERRIDE (2026-08-29, idled a $24/hr three-host fleet for
# eight hours). `exit-empty off` pins the server forever, so a stale var in the SERVER env is
# inherited by every later run and outranks that run's own configuration. A stale
# KEEL_REMOTE_HOSTS beat the .keel-hosts `aws-fleet.sh up` had just written. The runner
# therefore CLEARS the whole KEEL_/BENCH_ namespace before re-exporting the carried set
# (detach.sh:116) — so the run is a complete statement of itself, not of the deltas.
tmux start-server 2>/dev/null; tmux set-environment -g KEEL_TEST_FLEET beta
m="$(mkroot 's/^ *echo .*compgen.*unset.*$//')"
[[ "$(saw "$m" "t2m-$$")" == beta ]] && ok "2 fails first: with :116 reverted the SERVER's stale KEEL_TEST_FLEET=beta is inherited by a run that never asked for it" \
                                     || bad "2 mutant did not reproduce the incident, so the arm below proves nothing"
[[ "$(saw "$(mkroot)" "t2r-$$")" == "" ]] && ok "2 fixed: the shipped runner clears the namespace, so the server cannot inject a fleet" \
                                          || bad "2 a stale server variable is still reaching runs"
tmux set-environment -gu KEEL_TEST_FLEET

# ---- incident 3: ONE WORD FOR TWO FACTS (#122). `vanished ... killed or never started`
# named both causes at once and they call for opposite next actions; querying
# `validate113-ba6f286` as `keel-validate113-ba6f286` printed it for a run that was healthy
# and 25 minutes in. The property is DISTINGUISHABILITY, so that is what is asserted.
words() { # words ROOT -> "<first word of the died case> <first word of the never-started case>"
  local r="$1" d="t3d-$$"
  "$r/scripts/detach.sh" run "$d" -- sh -c 'echo up; sleep 120' >/dev/null 2>&1
  local _i; for _i in $(seq 1 40); do [[ -s "$r/build/$d.log" ]] && break; sleep 0.25; done
  tmux kill-session -t "=keel-$d" 2>/dev/null
  printf '%s %s\n' "$("$r/scripts/detach.sh" stat "$d" | awk '{print $1}')" \
                   "$("$r/scripts/detach.sh" stat "t3n-$$" | awk '{print $1}')"
}
w="$(words "$(mkroot "s/'died     %s/'vanished %s/; s/'never-started %s/'vanished %s/")")"
[[ "$w" == "vanished vanished" ]] && ok "3 fails first: with the two words reverted to one, killed-25-minutes-in and never-launched are indistinguishable ($w)" \
                                  || bad "3 mutant did not reproduce the incident ($w), so the arm below proves nothing"
w="$(words "$(mkroot)")"
[[ "$w" == "died never-started" ]] && ok "3 fixed: the two causes render as two words ($w)" || bad "3 got '$w', want 'died never-started'"

# The prefix is HINTED AT, never stripped: `keel-foo` is a legal NAME, so rejecting it would
# break a run merely named that way. The hint is the half a mutation can drive.
m="$(mkroot 's/ -- if that is what happened, drop the prefix//')"
o="$("$m/scripts/detach.sh" stat "t4-$$" 2>&1)"
[[ "$o" != *"drop the prefix"* ]] && ok "3b fails first: with the hint reverted, never-started does not name the prefix that caused the false alarm" \
                                 || bad "3b mutant kept the hint, so the arm below proves nothing"
r="$(mkroot)"
o="$("$r/scripts/detach.sh" stat "t4-$$" 2>&1)"
[[ "$o" == *"drop the prefix"* ]] && ok "3b fixed: never-started names the keel- prefix as the usual cause" \
                                 || bad "3b the prefix hint is gone"
"$r/scripts/detach.sh" run "keel-t5-$$" -- sh -c 'echo up; sleep 120' >/dev/null 2>&1
o="$("$r/scripts/detach.sh" stat "keel-t5-$$")"
[[ "$o" == running*keel-keel-t5-$$* ]] && ok "3b a legitimately keel-prefixed NAME still runs, and the doubled session name is visible" \
                                       || bad "3b prefixed NAME broke or hid its session: $o"

# ---- control, not an incident: a run that finishes must still report its exit code. Without
# it every arm above could be green because `stat` is broken for everything.
r="$(mkroot)"
"$r/scripts/detach.sh" run "t6-$$" -- sh -c 'exit 7' >/dev/null 2>&1
"$r/scripts/detach.sh" wait "t6-$$" >/dev/null 2>&1
[[ "$("$r/scripts/detach.sh" stat "t6-$$")" == exited*status=7* ]] \
  && ok "control: a finished run still reports exited status=7" || bad "control: stat is broken for finished runs"

# ---- isolation, asserted rather than assumed: every session above lived on the private server,
# so nothing here could reach a live detached run. Checked AFTER the arms, when a leak would exist.
[[ "$(tmux display-message -p '#{socket_path}' 2>/dev/null)" == "$BASE"/* ]] \
  && ok "isolation: the arms above ran on this test's own tmux server, not the operator's" || bad "isolation: TMUX_TMPDIR did not move the socket; the arms above touched a shared server"
# The second arm is VACUOUS when no default server is running, and it is not made non-vacuous by
# starting one -- that is the perturbation it is checking for. The socket-path arm above is the
# load-bearing one; this is the direct observation, kept for the case where a server does exist.
env -u TMUX_TMPDIR tmux list-sessions -F '#{session_name}' 2>/dev/null | /usr/bin/grep -qE "t[0-9]b?-$$|keel-t5-$$" \
  && bad "isolation: a test session leaked onto the default server" || ok "isolation: the operator's default server carries none of this test's sessions"

# ---- scope guard: remote.sh's REMOTE_STATE=vanished is a DIFFERENT mechanism with its own
# consumers and its own tests. #122 was not about it, and a future pass at "consistent
# vocabulary" should trip this rather than quietly rename it.
/usr/bin/grep -q 'REMOTE_STATE=vanished' "$ROOT/scripts/remote.sh" \
  && ok "scope: remote.sh's own 'vanished' is untouched" || bad "scope: remote.sh's vocabulary was changed; that was not #122's subject"

printf '\n%d ok, %d fail -- detach-test %s\n' "$P" "$F" "$([[ $F -eq 0 ]] && echo GREEN || echo RED)"
[[ $F -eq 0 ]]
