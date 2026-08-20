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

head_ "verdict"
if [[ "$FAILS" -eq 0 ]]; then
  echo "  GREEN -- a finished run reports its own exit code, a killed one reports"
  echo "  vanished, a severed link costs nothing, a missing supervisor is loud,"
  echo "  a host with no cpufreq is told apart from one whose knob will not read,"
  echo "  bare metal is admitted by a named arm while ten arms still refuse, and a"
  echo "  guest whose launcher contradicts it is unmeasured rather than demoted."
  exit 0
fi
echo "  RED -- $FAILS check(s) failed."
exit 1
