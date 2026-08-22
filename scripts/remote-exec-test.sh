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
#
# `-test.run=` AND NOT `-test.bench=`, changed 2026-08-21 when remote_exec began pinning.
# What this case checks is transport -- an argument containing a shell pipeline surviving
# two shells -- and `-test.run` carries that hazard identically. `-test.bench` now also
# selects the affinity branch, whose far side must have taskset and a NUMA sysfs; this
# machine has neither, so keeping it here would have turned four transport assertions into
# one refusal. The bench form is driven in case 9, with its refusal asserted rather than
# tolerated. A case narrowed to what it can actually witness is not a weakened case; a case
# that reports "argv was altered" because the mask was declined would be a wrong one.
remote_exec "$H" "$BIN" -test.run='Sgemm|Ssyrk' -test.count=2 > "$WORK/1.log" 2>&1; rc=$?
out="$(cat "$WORK/1.log")"
[[ "$REMOTE_SUPERVISED" == yes ]] && pass_ "ran supervised (REMOTE_SUPERVISED=yes)" ||
  fail_ "expected a supervised run, got REMOTE_SUPERVISED=$REMOTE_SUPERVISED"
[[ "$rc" -eq 0 && "$REMOTE_STATE" == ok ]] && pass_ "exit 0, REMOTE_STATE=ok" ||
  fail_ "expected exit 0 / ok, got exit $rc / $REMOTE_STATE"
# The hazard #62 named: a tmux layer is a second shell on the far side, so an
# argument that survived printf %q once must survive it twice. 'Sgemm|Ssyrk' as
# shell input is a pipeline -- the failure that made printf %q necessary at all.
if grep -q 'argv: \[-test\.run=Sgemm|Ssyrk\] \[-test\.count=2\]' <<<"$out"; then
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
# #106 appended virt= ahead of the cpus field, so the same re-check applies to it: the
# field has to be THERE (an absent field reads as an unread one, which is fail-closed and
# also indistinguishable from a probe that broke) and the parser after it has to survive.
[[ "$(sed -n 's/.*virt=\([^ |]*\).*/\1/p' <<<"$prov")" =~ ^(metal|guest|\?)$ ]] &&
  pass_ "provenance carries a virt= token host_admission can read" ||
  fail_ "provenance has no readable virt= field, so bare metal would be unclassifiable"
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

# ------------------- 8. the admission classifier: two arms widen, ten must not
#
# #106: `instance=none` was answering two questions with one word, so bare metal fell
# through the case default to `correctness` and would have demoted the three lab hosts
# whose evidence the v0.1.0 record rests on. The fix ADDS AN ARM; it does not widen the
# default. Which means the fix cannot be verified by the arm that widens: a change that
# only ever granted admission would prove nothing about the three arms that must still
# refuse, and #79 -- a published ratio measured under powersave -- is the precedent for
# why the governor conjunct has to be shown to bite.
#
# WHICH ARMS ARE FIXTURED, stated rather than left to be assumed (§5 rule 12). All arms
# are driven here from synthetic provenance lines, which is the whole space including the
# two combinations no host in this fleet can produce. In addition:
#   - the WIDENING arm is driven LIVE on vesta/janus/antares in the re-admission run
#     (bare metal, three vendors, governor=performance), and that is the arm whose live
#     result matters, because it is the one this fix changes;
#   - bare-metal-under-powersave stays fixture-only: driving it live means setting a
#     governor on Scott's hardware, which is both a sudo action and a perturbation of the
#     fleet a measurement is about to run on;
#   - virt=guest stays fixture-only: the AWS fleet's last configured guest was retired,
#     so there is no live host with a hypervisor flag to read. It is the one arm whose
#     PROBE half is unexercised anywhere -- that a real guest emits virt=guest is an
#     inference from CPUID.1:ECX.31, tested here only against a line this file wrote.
#
# THE SPAWN CONJUNCTS (2026-08-19), same disclosure. The CLASSIFIER half is fully driven
# below from synthetic lines. The PROBE half -- spawn_probe, which turns a launcher query
# into the token these lines carry -- is exercised where it can be: `none` and `?` against
# the real `spawn list` and a real absent binary, and the single-record arm against real
# launcher output for two live instances belonging to other work. `ambiguous`, `spot` and
# the malformed-record `?` were driven against fixtures in spawn's real schema, because
# producing them live means launching two instances under one name, launching a spot
# instance, or corrupting the launcher's output. The arm NOTHING has exercised end to end
# is the CONTRADICTION: no run has yet had a guest and a launcher disagree, and the
# fixture asserts what host_admission does with a disagreement, not that a real
# disagreement would ever be phrased this way.
head_ "8. host_admission: two arms widen, ten must not (#106, and the 2026-08-19 ruling)"
# The gate primitives remote.sh's verdict lines call, defined at the head of the only case
# that reads them: "the refusing arms must print their refusals" is a claim about the
# text, and the text is discarded if these stay undefined.
pass() { printf '        gate> ok     %s\n' "$1"; }
fail() { printf '        gate> FAIL   %s\n' "$1"; }
info() { printf '        gate> %s\n' "$1"; }
unmeasured() { printf '        gate> UNMEAS %s\n' "$1"; }

M="Zen 4 | instance=none | virt=metal"
adm_case() {
  local want="$1" line="$2" note="$3"
  host_admission "$line"
  [[ "$ADM_CLASS" == "$want" ]] && pass_ "$note -> $ADM_CLASS" ||
    fail_ "$note -> $ADM_CLASS, expected $want"
  info_ "why: $ADM_WHY"
}
adm_case evidentiary "$M | governor=performance | tmux=yes" "WIDENS: bare metal at performance"
adm_case correctness "$M | governor=powersave | tmux=yes"   "REFUSES: bare metal under powersave (#79)"
adm_case correctness "$M | governor=absent | tmux=yes"       "REFUSES: bare metal with no cpufreq to assert"
adm_case correctness "x | instance=none | virt=guest | governor=performance" \
  "REFUSES: hypervisor flag present, no EC2 identity"
adm_case correctness "x | instance=none | virt=? | governor=performance" \
  "REFUSES (default arm): virt unread"
adm_case correctness "x | instance=none | governor=performance" \
  "REFUSES (default arm): no virt field at all, i.e. a pre-#106 provenance line"
# CHANGED 2026-08-19, and the old line is worth naming because THIS PIN CAUGHT THE
# CHANGE: it read `instance=c7i.48xlarge` with no spawn field and expected evidentiary,
# which is what an approved type alone used to buy. Scott's judged-tier ruling made the
# launcher's record a second conjunct, so the same line now refuses -- correctly, and the
# fixture's old label ("unchanged") became the false part. A fixture updated because a
# requirement moved is not a weakened assertion; the arm it used to check is below, with
# the conjunct it was missing supplied.
adm_case evidentiary "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=i-0ab:c7a.48xlarge:ondemand" \
  "WIDENS: approved type the launcher independently confirms, on-demand"
adm_case correctness "x | instance=c7i.4xlarge | virt=guest | governor=absent | spawn=i-0ab:c7i.4xlarge:ondemand" \
  "unchanged: partial-size instance type, launcher agreeing or not"
# The four spawn conjuncts. Three refuse for three distinct absences and one refuses for
# a market, because "the launcher denies launching this", "no launcher was consulted",
# "two records answer to this name" and "it was spot" are four facts, only three of which
# are about the host at all.
adm_case correctness "x | instance=c7a.48xlarge | virt=guest | governor=absent" \
  "REFUSES: approved type, no spawn field at all (a pre-ruling provenance line)"
adm_case correctness "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=?" \
  "REFUSES: approved type, launcher could not be consulted"
adm_case correctness "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=none" \
  "REFUSES: approved type, launcher has no running record under this name"
adm_case correctness "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=ambiguous" \
  "REFUSES: approved type, two launcher records answer to this name"
adm_case correctness "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=i-0ab:c7a.48xlarge:spot" \
  "REFUSES: approved type the launcher confirms, but launched spot"
# The one arm that reaches `unknown` through a READING rather than an absence, and the
# whole reason a second witness was worth having. It must not land on `correctness`:
# that would grade the host "not admitted" when what is broken is the instrument.
adm_case unknown "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=i-0ab:c7a.2xlarge:ondemand" \
  "REFUSES as UNKNOWN: the guest and the launcher name different instance types"
adm_case unknown "x | instance=? | virt=metal | governor=performance" \
  "unchanged: no way to ask -> unknown, and unknown is unmeasured"
adm_case unknown "" "unchanged: no reading at all -> unknown"

# N refusals, N causes. §5 rule 6 forbids one verdict standing for two causes, and a
# shared WHY string is that defect at the message layer -- which is what the code this
# replaced had, telling a bare-metal host it was "not a full-size instance".
#
# THE EXPECTED COUNT IS THE LIST'S LENGTH, not a literal. It was a hardcoded 4 beside a
# four-element list, and the spawn conjuncts took the list to nine while the 4 stayed --
# a check that would have gone green on nine arms sharing six causes. The property is
# "as many distinct causes as refusing arms", so that is what it computes.
refusals=(
  "$M | governor=powersave"
  "x | instance=none | virt=guest"
  "x | instance=none | virt=?"
  "x | instance=c7i.4xlarge | virt=guest"
  "x | instance=c7a.48xlarge | virt=guest"
  "x | instance=c7a.48xlarge | virt=guest | spawn=?"
  "x | instance=c7a.48xlarge | virt=guest | spawn=none"
  "x | instance=c7a.48xlarge | virt=guest | spawn=ambiguous"
  "x | instance=c7a.48xlarge | virt=guest | spawn=i-0ab:c7a.48xlarge:spot"
  "x | instance=c7a.48xlarge | virt=guest | spawn=i-0ab:c7a.2xlarge:ondemand"
)
whys=()
for l in "${refusals[@]}"; do host_admission "$l"; whys+=("$ADM_WHY"); done
nw="$(printf '%s\n' "${whys[@]}" | sort -u | wc -l | tr -d ' ')"
[[ "$nw" -eq "${#refusals[@]}" ]] &&
  pass_ "the ${#refusals[@]} refusals name ${#refusals[@]} distinct causes" ||
  fail_ "$nw distinct causes across ${#refusals[@]} refusing arms: at least two share a verdict"

# The governor conjunct is now derived TWICE from one field -- here, and in
# assert_governor's cascade -- so the two are pinned to agree rather than assumed to
# (§5 rule 10: two statements of one fact are a pair that can disagree). `evidentiary`
# on a bare-metal line must mean exactly `GOV_STATE == performance`.
for g in performance powersave absent unreadable unknown; do
  line="x | instance=none | virt=metal | governor=$g | tmux=yes"
  assert_governor fixture preamble "$line" >/dev/null 2>&1
  host_admission "$line"
  if [[ "$GOV_STATE" == performance ]]; then want=evidentiary; else want=correctness; fi
  [[ "$ADM_CLASS" == "$want" ]] &&
    pass_ "governor=$g: GOV_STATE=$GOV_STATE and class=$ADM_CLASS -- the two derivations agree" ||
    fail_ "governor=$g: GOV_STATE=$GOV_STATE but class=$ADM_CLASS, expected $want -- diverged"
done

# adm_judgeable is what a gate calls. It prints the refusal itself and returns 1, so both
# halves are checked: a refusal that returned 0 would let the number through with the
# words still on the page, and an unread identity must land on UNMEASURED rather than on
# the same "reported, not judged" as a class that was read.
head_ "8b. adm_judgeable prints the refusal a gate would tally"
jcase() {
  local want_rc="$1" want_re="$2" line="$3" note="$4" rc=0 out
  out="$(adm_judgeable fixture "$line" "reads 42.7% of measured peak")" || rc=$?
  [[ "$rc" -eq "$want_rc" ]] && pass_ "$note -> rc=$rc" ||
    fail_ "$note -> rc=$rc, expected $want_rc"
  if [[ "$want_re" == SILENT ]]; then
    [[ -z "$out" ]] && pass_ "  and says nothing: an admitted host gets no verdict here" ||
      fail_ "  printed a verdict for an admitted host: $out"
    return
  fi
  printf '%s\n' "${out:-        gate> <no verdict line at all>}"
  grep -Eq "$want_re" <<<"$out" && pass_ "  and its cause reads '$want_re'" ||
    fail_ "  its verdict line does not name '$want_re'"
}
jcase 0 SILENT                        "$M | governor=performance" "bare metal at performance is judgeable"
jcase 1 'governor=powersave, not performance' "$M | governor=powersave" "powersave refused"
jcase 1 'hypervisor flag present'     "x | instance=none | virt=guest | governor=performance" "guest refused"
jcase 1 'neither route to whole-socket ownership' "x | instance=none | virt=? | governor=performance" "default arm refused"
jcase 1 'UNMEAS.*class is unreadable' "x | instance=? | virt=metal | governor=performance" "unread identity unmeasured"
# The second route to UNMEASURED, and the reason adm_judgeable's parenthetical had to
# become $ADM_WHY: it was the literal string "instance=... absent from the provenance
# line", which is the ONE cause `unknown` had before the launcher became a witness. On a
# contradiction that wording reported an absent identity for a line carrying two of them.
# Keyed to the contradiction's own phrase, not to a shared prefix, so this case cannot go
# green on the absence arm's sentence (a verifier keyed to text both outcomes emit
# certifies neither).
jcase 1 'UNMEAS.*two witnesses of this host.s identity disagree' \
  "x | instance=c7a.48xlarge | virt=guest | governor=absent | spawn=i-0ab:c7a.2xlarge:ondemand" \
  "contradicting witnesses unmeasured, naming the contradiction"

# ------------------- 9. the affinity mask: selected from a topology, or refused
#
# §5 rule 5's mask (adopted fleet-wide 2026-08-21). The selector is sh, reads a topology
# out of sysfs, and runs on the far side -- so it is driven HERE against topologies this
# script builds, on a machine with no sysfs at all. That is the same move case 7 makes for
# the governor: hand the function its input rather than find a host that has it. What it
# buys is the shapes no host in the fleet has, and those are the ones a mask gets wrong:
# a node too small to satisfy the width, a sysfs with no sibling lists, a cpulist written
# as a range where the sibling list is written as a comma pair.
#
# KEEL_SYSFS is a root path, not a mode flag: the function has no test-only branch, and
# the fixtures below are ordinary callers supplying an ordinary argument.
#
# WHAT THESE CANNOT SEE (§5 rule 12). That a real Linux host's node/cpulist,
# thread_siblings_list and cache/indexN/{level,shared_cpu_list} are formatted the way
# these fixtures write them is read off two hosts' sysfs, off T-45's reading of EPYC
# 9R45's index3, and off Documentation/ABI/testing/sysfs-devices-node -- it is not
# established here. Nor is the fixtures' NODE partition: whether keel-zen5's node0 is a
# socket or half of one is unread, and it does not reach the selector, which answers out
# of the first node that satisfies the width. Nor is `taskset -c <mask>` itself exercised anywhere in this file;
# what proves the mask TOOK is gate-p5's second reading of it (GOMAXPROCS off Go's own
# affinity), and that needs a Linux host with eight cores, which is a fleet run.
head_ "9. keel_pin_mask spreads one core per cache domain, or refuses"
eval "$KEEL_PIN_SH"     # the shipped text, defined in this shell exactly as the far side sees it

# llc_dom ROOT SHARED_CPU_LIST CPU... -- a cache topology for the named cpus: a level-1
# index shared with nobody, and a level-3 index shared across SHARED_CPU_LIST. Both, not
# just the L3, so the selector's "highest level" is actually put to work: a version that
# took the FIRST readable index would make every core its own domain and hand back
# consecutive cores, which is the old form wearing the new label.
llc_dom() {
  local root="$1" list="$2"; shift 2
  local c
  for c in "$@"; do
    mkdir -p "$root/cpu/cpu$c/cache/index0" "$root/cpu/cpu$c/cache/index3"
    printf '1\n'    > "$root/cpu/cpu$c/cache/index0/level"
    printf '%d\n' "$c" > "$root/cpu/cpu$c/cache/index0/shared_cpu_list"
    printf '3\n'    > "$root/cpu/cpu$c/cache/index3/level"
    printf '%s\n' "$list" > "$root/cpu/cpu$c/cache/index3/shared_cpu_list"
  done
}

# A two-socket SMT host in the shape keel-skx really has: 18 physical cores per node, the
# second thread of core c numbered c+36, and a cpulist written as two ranges. Its L3 is
# per-socket, so each node is ONE domain and the spread form degenerates to consecutive
# cores -- which is why every expectation below this is byte-identical to the packed
# form's, and why that is evidence of a refinement rather than of an unexercised change.
topo_skx() {
  local root="$1" c
  mkdir -p "$root/node/node0" "$root/node/node1"
  printf '0-17,36-53\n' > "$root/node/node0/cpulist"
  printf '18-35,54-71\n' > "$root/node/node1/cpulist"
  for c in $(seq 0 35); do
    mkdir -p "$root/cpu/cpu$c/topology"
    printf '%d,%d\n' "$c" "$((c + 36))" > "$root/cpu/cpu$c/topology/thread_siblings_list"
    mkdir -p "$root/cpu/cpu$((c + 36))/topology"
    printf '%d,%d\n' "$c" "$((c + 36))" > "$root/cpu/cpu$((c + 36))/topology/thread_siblings_list"
  done
  llc_dom "$root" '0-17,36-53'  $(seq 0 17)  $(seq 36 53)
  llc_dom "$root" '18-35,54-71' $(seq 18 35) $(seq 54 71)
}
pin_case() {
  local note="$1" want="$2" expect="$3" root="$4" got=""
  got="$(KEEL_SYSFS="$root" keel_pin_mask "$want" || true)"
  [[ "$got" == "$expect" ]] && pass_ "$note -> ${got:-REFUSED}" ||
    fail_ "$note -> '${got:-REFUSED}', expected '${expect:-REFUSED}'"
}

SKX="$WORK/topo-skx"; topo_skx "$SKX"
pin_case "18-core SMT node, want 8" 8 "0,1,2,3,4,5,6,7" "$SKX"
# The cross-socket guard, driven on purpose: 72 cpus and 36 cores are present and the
# answer must still be no, because completing the count out of node 1 is the migration the
# mask exists to prevent. An unrefused 20 here is a mask spanning two sockets.
pin_case "18-core SMT node, want 20 (more cores than any ONE node has)" 20 "" "$SKX"

# node0 unreadable, node1 fine: the loop moves on, and the count restarts at the node it
# lands on rather than carrying node0's partial run across.
NOD0="$WORK/topo-nonode0"; topo_skx "$NOD0"; rm -f "$NOD0/node/node0/cpulist"
pin_case "node0 has no cpulist, node1 does" 4 "18,19,20,21" "$NOD0"

# No SMT at all -- every cpu is its own sibling list, which is what a host with
# hyperthreading disabled in firmware reports.
NOSMT="$WORK/topo-nosmt"
mkdir -p "$NOSMT/node/node0"; printf '0-3\n' > "$NOSMT/node/node0/cpulist"
for c in 0 1 2 3; do
  mkdir -p "$NOSMT/cpu/cpu$c/topology"
  printf '%d\n' "$c" > "$NOSMT/cpu/cpu$c/topology/thread_siblings_list"
done
llc_dom "$NOSMT" '0-3' 0 1 2 3
pin_case "4 cores, no SMT, want 4" 4 "0,1,2,3" "$NOSMT"
pin_case "4 cores, no SMT, want 8" 8 "" "$NOSMT"

# THE SHAPE THE AMENDMENT WAS MADE FOR (2026-08-22). EPYC 9R45 as T-45 read it out of
# sysfs: eight-core L3 domains, the second thread of core c numbered c+96. The expected
# answer is not a preference, it is the arm the 5.96x/4.65x ratio in DESIGN §5 rule 5 was
# measured on -- so this fixture asserts that what the law now says and what the harness
# now selects are the same eight cores, which is the one claim T-45 could not make about
# the packed form.
EPYC="$WORK/topo-epyc"
mkdir -p "$EPYC/node/node0"; printf '0-95,96-191\n' > "$EPYC/node/node0/cpulist"
for c in $(seq 0 95); do
  mkdir -p "$EPYC/cpu/cpu$c/topology" "$EPYC/cpu/cpu$((c + 96))/topology"
  printf '%d,%d\n' "$c" "$((c + 96))" > "$EPYC/cpu/cpu$c/topology/thread_siblings_list"
  printf '%d,%d\n' "$c" "$((c + 96))" > "$EPYC/cpu/cpu$((c + 96))/topology/thread_siblings_list"
done
for d in $(seq 0 11); do
  a=$((d * 8)); b=$((a + 7))
  llc_dom "$EPYC" "$a-$b,$((a + 96))-$((b + 96))" $(seq "$a" "$b")
done
pin_case "EPYC 9R45 node: 12 eight-core L3 domains, want 8" 8 "0,8,16,24,32,40,48,56" "$EPYC"
# Twelve domains, want 4: the first FOUR domains and no domain twice, which is what
# distinguishes one-per-domain from a stride that happens to be eight.
pin_case "same node, want 4 (uses 4 of 12 domains)" 4 "0,8,16,24" "$EPYC"

# THE TWO SHAPE GLOBALS, which stdout does not carry and pin_case therefore cannot see. The
# pin line is built out of these, so an unasserted global is an unrecorded shape — and the
# skx pair below is the reason the second one exists at all: the same eight consecutive cores
# are a confined mask on a 12-domain node and the only correct mask on a 1-domain one, and
# nothing but `nodedoms` tells those apart.
# `wantv`, not `want`: the shipped selector is POSIX sh and assigns `want=$1` with no local,
# and bash locals are dynamically scoped — so calling it clobbers a caller local of that name.
# `pin_case` above is safe only because it never reads `want` after the call. Cost of getting
# this wrong is a comparison against 8, which reads as three broken assertions.
shape_case() {
  local note="$1" wantv="$2" root="$3"
  KEEL_SYSFS="$root" keel_pin_mask 8 >/dev/null || true
  local got="$KEEL_PIN_DOMLIST|$KEEL_PIN_NODEDOMS"
  [[ "$got" == "$wantv" ]] && pass_ "$note -> $got" || fail_ "$note -> $got, expected $wantv"
}
shape_case "EPYC node records doms and nodedoms" "0,8,16,24,32,40,48,56|12" "$EPYC"
shape_case "skx node: consecutive cores, and one domain to spread over" "0,0,0,0,0,0,0,0|1" "$SKX"
# A refusal must not leave the previous run's shape standing. Driven immediately after a
# success, which is the only ordering in which the bug is visible.
mkdir -p "$WORK/topo-none"
shape_case "a refusal clears them rather than keeping the last success" "|" "$WORK/topo-none"
# More width than domains, so pass two runs four times: every domain gives its first core
# before any gives its second. Nothing in the fleet has this shape; the round-robin has a
# second pass whether or not a host exercises it, and an unexercised loop is not evidence.
WRAP="$WORK/topo-wrap"
mkdir -p "$WRAP/node/node0"; printf '0-7\n' > "$WRAP/node/node0/cpulist"
for c in $(seq 0 7); do
  mkdir -p "$WRAP/cpu/cpu$c/topology"
  printf '%d\n' "$c" > "$WRAP/cpu/cpu$c/topology/thread_siblings_list"
done
llc_dom "$WRAP" '0-3' 0 1 2 3; llc_dom "$WRAP" '4-7' 4 5 6 7
pin_case "2 domains of 4, want 8 (four passes)" 8 "0,4,1,5,2,6,3,7" "$WRAP"

# THE NEW FAIL-CLOSED BRANCH, driven on purpose. Sibling lists prove distinctness and
# cache levels prove the spread, so a host that cannot show the second refuses exactly as
# one that cannot show the first: falling back to consecutive cores here would put a
# single-CCD reading under the spread label, which is the forgery the era ledger exists to
# prevent, one layer in from the free-placement one.
NOLLC="$WORK/topo-nollc"; topo_skx "$NOLLC"; rm -rf "$NOLLC"/cpu/cpu*/cache
pin_case "36 cores, sibling lists, no cache topology at all" 8 "" "$NOLLC"
# And the check is per-core, not per-host: one core of node0 cannot prove its domain, so
# node0 goes whole and node1 answers instead. A version reading only cpu0 would pass this
# and hand back a mask over a topology it never looked at.
ONELLC="$WORK/topo-onecoreblind"; topo_skx "$ONELLC"; rm -rf "$ONELLC/cpu/cpu5/cache"
pin_case "one core of node0 is cache-blind: node0 abandoned, node1 answers" \
  8 "18,19,20,21,22,23,24,25" "$ONELLC"

# THE BRANCH THIS FIXTURE CREATED. The first version of the selector fell back to
# `sib=$c` when no sibling list could be read, which on a hyperthreaded host hands back
# eight cpus that may be four cores -- and gate-p5's readback cannot catch it, because
# GOMAXPROCS is 8 either way. Distinctness is the whole claim, so an unreadable topology
# now abandons the node. Sixteen cpus present, answer still no.
NOSIB="$WORK/topo-nosib"
mkdir -p "$NOSIB/node/node0" "$NOSIB/cpu"; printf '0-15\n' > "$NOSIB/node/node0/cpulist"
pin_case "16 cpus but no sibling lists: distinctness unprovable" 8 "" "$NOSIB"

# A sibling list written as a RANGE, which some kernels do for a pair. `first` has to
# survive both spellings or the second thread of every core counts as a core.
RANGE="$WORK/topo-range"
mkdir -p "$RANGE/node/node0"; printf '0-3\n' > "$RANGE/node/node0/cpulist"
for c in 0 1 2 3; do
  mkdir -p "$RANGE/cpu/cpu$c/topology"
  printf '%d-%d\n' "$(( (c / 2) * 2 ))" "$(( (c / 2) * 2 + 1 ))" \
    > "$RANGE/cpu/cpu$c/topology/thread_siblings_list"
done
llc_dom "$RANGE" '0-3' 0 1 2 3
pin_case "sibling lists spelled as ranges, want 2" 2 "0,2" "$RANGE"
pin_case "sibling lists spelled as ranges, want 4 (only 2 cores exist)" 4 "" "$RANGE"

# No node directory at all: a container or a kernel without NUMA in sysfs.
mkdir -p "$WORK/topo-empty"
pin_case "no node directories in sysfs" 8 "" "$WORK/topo-empty"

# ------------------- 9b. and remote_exec refuses the measurement it cannot pin
#
# The live half, on whatever far side this machine has. Both arms are assertions: a far
# side with no taskset must take NO MEASUREMENT and say why (macOS localhost, which is
# what runs here), and one that can pin must write the mask into the log the caller reads.
# Free placement is the arm that must be unreachable, so the check is that the program did
# not run -- not merely that the status was nonzero.
head_ "9b. a benchmark invocation is pinned, or not taken"
grep -q 'keel-pin:' "$WORK/1.log" &&
  fail_ "a NON-benchmark invocation was pinned: case 1's log carries a keel-pin line" ||
  pass_ "a non-benchmark invocation is untouched: no keel-pin line in case 1's log"
source scripts/bench.sh   # bench_pin / bench_gomaxprocs, so the readings below are the
                          # ones gate-p5's criterion takes and not a second sed that agrees
remote_exec "$H" "$BIN" -test.bench=Sgemm -test.count=1 > "$WORK/9.log" 2>&1; rc=$?
info_ "$(head -1 "$WORK/9.log" 2>/dev/null || echo '<empty log>')"
if grep -q 'keel-pin: REFUSED' "$WORK/9.log"; then
  [[ "$rc" -eq 121 && "$REMOTE_STATE" == ok ]] &&
    pass_ "unpinnable far side: exit 121 recovered, REMOTE_STATE=ok (a refusal, not a vanishing)" ||
    fail_ "refusal reported exit $rc / REMOTE_STATE=$REMOTE_STATE, expected 121 / ok"
  grep -q 'argv:' "$WORK/9.log" &&
    fail_ "the binary RAN anyway: a refusal that still measures is free placement with a log line" ||
    pass_ "the binary did not run, so no unpinned sample can reach an archive"
  info_ "the pinned arm is unexercised here: this far side has no taskset. It is driven on"
  info_ "every fleet host by gate-p5's placement criterion, which reads the mask back twice."
elif grep -q 'keel-pin: mask=' "$WORK/9.log"; then
  m="$(bench_pin "$WORK/9.log")"
  nf="$(tr ',' '\n' <<<"${m%% *}" | grep -c .)"
  [[ "$rc" -eq 0 && "${m##* }" == "$KEEL_PIN_WIDTH" && "$nf" -eq "$KEEL_PIN_WIDTH" ]] &&
    pass_ "pinned far side: mask ${m%% *} of $nf cores, width ${m##* }, exit $rc" ||
    fail_ "pinned run disagrees with itself: '$m' names width ${m##* } over $nf cores, exit $rc"
  grep -q 'argv: \[-test\.bench=Sgemm\]' "$WORK/9.log" &&
    pass_ "and the measurement ran under it" || fail_ "pinned but the program did not run"
  info_ "the REFUSAL arm is unexercised here: this far side can pin. It is driven wherever"
  info_ "taskset or an eight-core node is missing, and by the selector fixtures above."
else
  fail_ "a benchmark invocation produced neither a mask nor a refusal (exit $rc), so placement is unrecorded:"
  sed 's/^/        /' "$WORK/9.log"
fi

# ------------------- 9c. the two readings gate-p5 compares, and their disagreement
#
# gate-p5's placement criterion is `bench_pin` against `bench_gomaxprocs` over one log.
# Its PASS arm runs on every fleet host; its FAIL arm — a mask requested that did not take
# — is by construction something no correctly working host produces, so without fixtures it
# would be an unexecuted branch in the criterion that scopes every reading to an era. These
# are logs, so both arms are drivable here: what the criterion reads is a file.
head_ "9c. bench_pin vs bench_gomaxprocs, agreeing and disagreeing"
mklog() { printf '%s\n' "$@" > "$WORK/pin.log"; }
readback() {   # prints "mask width gomaxprocs" the way gate-p5 derives them
  local pm pw pg; pm="$(bench_pin "$WORK/pin.log")"; pw="${pm##* }"; pm="${pm%% *}"
  pg="$(bench_gomaxprocs "$WORK/pin.log")"
  printf '%s|%s|%s\n' "$pm" "$pw" "$pg"
}
shape() {      # prints "domlist|nodedoms|distinct expected imbalance|verdict", gate-p5's second half
  local pd pn ps rc
  pd="$(bench_pin_doms "$WORK/pin.log")"; pn="${pd##* }"; pd="${pd%% *}"
  if [[ -z "$pd" || -z "$pn" ]]; then printf '%s|%s|-|unmeasured\n' "$pd" "$pn"; return; fi
  ps="$(bench_pin_spread "$pd" 8 "$pn")" && rc=pass || rc=FAIL
  printf '%s|%s|%s|%s\n' "$pd" "$pn" "$ps" "$rc"
}
sh_case() {
  local note="$1" want="$2" got; got="$(shape)"
  [[ "$got" == "$want" ]] && pass_ "$note -> $got" || fail_ "$note -> $got, expected $want"
}
rb_case() {
  local note="$1" want="$2" got; got="$(readback)"
  [[ "$got" == "$want" ]] && pass_ "$note -> $got" || fail_ "$note -> $got, expected $want"
}
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8' \
      'BenchmarkScale/n=1024/threads=1-8   	      10	 118000000 ns/op' \
      'BenchmarkScale/n=1024/threads=8-8   	      10	  16000000 ns/op'
rb_case "a pinned sweep: mask, width and GOMAXPROCS all read" "0,1,2,3,4,5,6,7|8|8"
# THE FAIL ARM. Same mask line, rows named -192: the harness asked and the kernel did not
# deliver, which is free placement wearing a pinned label. This is the log shape every
# reading published before 2026-08-21 has (minus the mask line), and the criterion has to
# see 8 against 192 rather than read one of them twice.
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8' \
      'BenchmarkScale/n=1024/threads=8-192   	      10	  16000000 ns/op'
rb_case "a mask that did not take: 8 requested, 192 seen" "0,1,2,3,4,5,6,7|8|192"
# Two widths in one log, which is what a concatenation or a re-run into the same file gives:
# the readings must DISAGREE rather than quietly agree on the last one, because a log with
# rows from two placements is not a measurement of either.
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8' \
      'BenchmarkScale/threads=8-8   	      10	  16000000 ns/op' \
      'BenchmarkScale/threads=8-72   	      10	  16000000 ns/op'
rb_case "rows from two placements in one log" "0,1,2,3,4,5,6,7|8|8 72"
# An unpinned log: no mask line at all, which is the UNMEASURED arm. It must not be
# mistaken for a mask of width zero, and the rows must still be readable — the criterion
# distinguishes "the instrument did not report" from "it reported free".
mklog 'BenchmarkScale/threads=8-192   	      10	  16000000 ns/op'
rb_case "a pre-2026-08-21 log: no mask line" "||192"
# A refusal log, which has a keel-pin line that is NOT a mask. `bench_pin` matches on the
# full `mask=... width=N` shape for exactly this reason: a refusal must read as no mask
# rather than as a parse of the word REFUSED.
mklog 'keel-pin: REFUSED, no taskset on this host. DESIGN section 5 rule 5 pins fleet-wide and never selectively, so this measurement is not taken rather than taken unpinned.'
rb_case "a refusal log carries a keel-pin line but no mask" "||"

# ------------------- 9d. and the SHAPE of the mask, which its width cannot show
#
# The 2026-08-22 spread amendment (§5 rule 5) is the reason this section exists: a mask
# confined to one CCD and a mask spread over eight read back identically through
# `bench_gomaxprocs`, because GOMAXPROCS is 8 either way, and on keel-zen5 the confined one
# measured 5.96x less stream bandwidth. So the pin line carries the domain of each selected
# core and the count the node had, and the invariant over them is what the criterion asserts.
# Every arm below is a log, so every arm is drivable here — including the confined one, which
# no correctly working host can now produce.
head_ "9d. the mask's shape: one core per cache domain, or not this era"
mklog 'keel-pin: mask=0,8,16,24,32,40,48,56 width=8 doms=0,8,16,24,32,40,48,56 nodedoms=12' \
      'BenchmarkScale/threads=8-8   	      10	  16000000 ns/op'
sh_case "keel-zen5's spread mask: 8 cores, 8 of 12 domains" \
  "0,8,16,24,32,40,48,56|12|8 8 0|pass"
# The width still reads, which is what keeps the two halves of the criterion independent: a
# log can satisfy the width check and fail the shape one.
rb_case "the same line still yields mask and width to bench_pin" "0,8,16,24,32,40,48,56|8|8"
# THE CONFINED ARM, the defect the amendment was made for, as its log would look. Same width,
# same GOMAXPROCS, one domain out of twelve — caught here and nowhere else. Note the third
# number: imbalance is **0**, because eight cores in one domain are perfectly evenly spread
# over the one domain they occupy. Balance cannot see confinement at all, which is what makes
# `nodedoms` load-bearing rather than belt-and-braces — the two terms of the invariant catch
# disjoint defects. Predicted 7 here and read 0; the fixture adjudicated the field.
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8 doms=0,0,0,0,0,0,0,0 nodedoms=12' \
      'BenchmarkScale/threads=8-8   	      10	  16000000 ns/op'
sh_case "a single-CCD mask on a 12-domain node" "0,0,0,0,0,0,0,0|12|1 8 0|FAIL"
# And the same eight consecutive cores are CORRECT where the node has one domain: keel-skx's
# L3 is per socket, so min(8,1) is 1 and the mask cannot spread further than the silicon does.
# The two logs above and below differ only in `nodedoms`, which is the whole reason it is
# recorded: the mask string cannot tell these apart.
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8 doms=0,0,0,0,0,0,0,0 nodedoms=1' \
      'BenchmarkScale/threads=8-8   	      10	  16000000 ns/op'
sh_case "keel-skx: the same mask, one domain in the node" "0,0,0,0,0,0,0,0|1|1 1 0|pass"
# Fewer domains than the width, so the round-robin wraps and the invariant is balance rather
# than one-each. An unbalanced mask over the same domain count is refused.
mklog 'keel-pin: mask=0,4,1,5,2,6,3,7 width=8 doms=0,4,0,4,0,4,0,4 nodedoms=2'
sh_case "2 domains, 4 cores each, balanced" "0,4,0,4,0,4,0,4|2|2 2 0|pass"
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8 doms=0,0,0,0,0,0,0,4 nodedoms=2'
sh_case "2 domains, 7 cores against 1" "0,0,0,0,0,0,0,4|2|2 2 6|FAIL"
# A pre-amendment archive: the mask line without the two fields. It must read as "the shape
# was not reported", never as a shape that passed — this is exactly the 24 confined archives
# in archive/pinned8/, which is how the provisional arm identifies itself untouched.
mklog 'keel-pin: mask=0,1,2,3,4,5,6,7 width=8' \
      'BenchmarkScale/threads=8-8   	      10	  16000000 ns/op'
sh_case "a 2026-08-21 confined archive: no shape recorded" "||-|unmeasured"

head_ "verdict"
if [[ "$FAILS" -eq 0 ]]; then
  echo "  GREEN -- a finished run reports its own exit code, a killed one reports"
  echo "  vanished, a severed link costs nothing, a missing supervisor is loud,"
  echo "  a host with no cpufreq is told apart from one whose knob will not read,"
  echo "  bare metal is admitted by a named arm while ten arms still refuse, a"
  echo "  guest whose launcher contradicts it is unmeasured rather than demoted, and"
  echo "  a benchmark is pinned to eight distinct cores, one per cache domain, in"
  echo "  one node -- or not taken, and the shape is recorded rather than trusted."
  exit 0
fi
echo "  RED -- $FAILS check(s) failed."
exit 1
