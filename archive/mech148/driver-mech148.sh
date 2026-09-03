#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #148's test 3: five NAMED cpu masks and environments against the scalar rows, one binary,
# to bound how much of test 2's one-core collapse is removable by turning off Go runtime
# work that competes for the single confined cpu. Test 2 attributed the collapse to mask
# width 1 and killed branch A (cpu0 is not special, by identity); it could not separate
# WHICH runtime activity on that one cpu costs the rate.
#
# The boundaries, the arms, the preconditions, the twelve scored cells and the out-of-domain
# list are in archive/mech148/predictions-mech148.py, committed at c14f870 BEFORE this file
# and imported by the analyzer rather than restated in it. Read that file first; this one
# only collects the samples it will be read against. Structure, wording and refusal style
# are driver-core148.sh's, deliberately: this is test 2's driver with the arm table replaced
# and one mechanism added (the quietness precondition below).
#
# THE ARMS. Masks are test 2's, so `ref` and `c0` are further draws of arms already
# characterised; only the environment changes, and only on the one-cpu arms. Note what `ref`
# actually is: TWO physical cores at GOMAXPROCS=1 -- recovery does not need eight cpus, it
# needs a second one.
#
#   ref   0,1   GOMAXPROCS=1                                    recovered control
#   c0    0     GOMAXPROCS=1                                    collapse positive control
#   gc    0     GOMAXPROCS=1 GOGC=off                           removes GC assist + workers
#   pre   0     GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1          removes preemption signals
#   both  0     GOMAXPROCS=1 GOGC=off GODEBUG=asyncpreemptoff=1 the joint arm
#
# GOMAXPROCS COMES FIRST IN EVERY ENV STRING, AND THAT ORDERING IS LOAD-BEARING. remote.sh
# interpolates $KEEL_REMOTE_ENV unquoted ahead of `env` (scripts/remote.sh:1687), so
# "GOMAXPROCS=1 GOGC=off" must word-split into two assignments. If it ever stopped doing so,
# `env` would set GOMAXPROCS to the literal "1 GOGC=off", Go would reject it, and the value
# would fall back to ncpu -- which the EXISTING `keel-bench-gomaxprocs:` readback sees. So
# the first assignment is the canary for all of them, and this driver asserts it per arm
# instead of printing it. No new mechanism was needed for that hole; it needed reading.
#
# WHY THE READBACKS MATTER MORE HERE THAN IN TEST 2. My registered prediction is the NULL --
# all three treatment arms stay confined. That inverts the usual risk: every way a treatment
# can silently fail to apply produces exactly the result I predicted. A confirmation reached
# by a broken treatment is the one outcome this run must not be able to report, so the
# treatment is witnessed before it is trusted (the gctrace probe below), and the binary is a
# refusal rather than a note.
#
# TWO PASSES, THE SECOND IN REVERSED ARM ORDER, carried over from test 2 and test 1, where
# it turned "no time drift" into a measurement that read 1.000-1.003.
#
# ALL 20 ROWS, NOT THE REGISTERED 4, for the same reason as test 2: the other 16 read
# 0.947-1.056 across every arm of test 1, so if they move here the finding is a property of
# the host or the window rather than of the environment.
#
# No `set -e`: every step reports its own rc rather than vanishing, and a killed run must not
# be able to look like a verdict.
set -uo pipefail

cd /Users/scttfrdmn/src/keel || exit 3

say() { printf '\n=== %s ===\n' "$*"; }

REV="$(git rev-parse --short HEAD)"
FULLREV="$(git rev-parse HEAD)"
HOST="${HOST:-janus.local}"
# Overridable so a smoke run can exercise THIS script rather than a second, differently
# worded copy of it. The judged values are the defaults and the log records which were used.
FILTER="${FILTER:-BenchmarkKernel}"
COUNT="${COUNT:-30}"
BTIME="${BTIME:-1s}"
# `-` and not `:-`: with `:-`, ARMS_B="" falls back to the full default, so asking for one
# arm silently runs five. Found by driver-width148.sh's smoke run.
ARMS_A="${ARMS_A-ref c0 gc pre both}"
ARMS_B="${ARMS_B-both pre gc c0 ref}"
# The binary the registered boundaries were derived on. Absolute GFLOP/s carried across runs
# transfer to no other artifact, so this is a REFUSAL and not a note -- test 2 recorded the
# same prefix at 97a21f4 (archive/core148/core148-97a21f4.log:23) and only reported it,
# because its criterion was a ratio against its own reference and could absorb a rebuild.
WANT_SHA="${WANT_SHA:-d0d46d26c15cc8b2}"
# ...and the toolchain that yields that binary is PINNED here rather than left to whatever the
# driver host happens to have, because the dev host cross-compiles every fleet run
# (scripts/remote.sh:662) so its Go version IS part of the artifact. Measured, not assumed: the
# first launch of this driver REFUSED at exit 8 with sha d10b953b924316d8 / 5066561 bytes, and
# the cause was terror upgrading go1.27.0 -> go1.27.1 with NO keel input changed at all
# (`git diff 97a21f4..HEAD -- '*.go' go.mod go.sum` is empty; test 2's log:23 records the
# builder as go1.27.0-X:simd and this run's as go1.27.1-X:simd). Rebuilding at go1.27.0
# reproduces d0d46d26c15cc8b2 / 5066553 bytes EXACTLY, so the registration needs no amendment
# and none is taken. Pinned and not documented-for-the-operator for the reason ruled on
# 2026-09-02: a precondition only the operator remembers to check isn't one. Nothing is trusted
# here either -- WANT_SHA still adjudicates, so a pin that stops working refuses like anything
# else.
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.27.0}"
TAG="${TAG:-}"
OUT="build"

# --- the quietness precondition, and where every number in it comes from ----------------
#
# Ruled in 2026-09-02: a violation renders THAT ARM unmeasured, naming the load, sampled
# between arms and never during one. Per-arm because a co-tenant that arrives mid-run must
# cost the arms it overlaps and not the ones it does not; partial evidence survives.
#
# WHICH FIELD. Not the 1-minute average, and that is derived rather than preferred. The one
# contaminated sample in test 2's tracked record (`after bref`,
# archive/core148/core148-97a21f4.log:257) reads `0.99 2.17 1.83`: its 1-MINUTE value 0.99
# sits INSIDE the clean 1-minute range [0.92, 1.05] over the other 15 samples, so a
# 1-minute bound scores 0 of 1 on the only positive example the record contains. The
# 5-minute field separates it cleanly, and the reason is mechanical -- the co-tenant shape
# observed on this host is a benchmark respawning about once a second, which a short average
# and an instantaneous count both miss. That is also why /proc/loadavg's runnable field is
# not the gate here despite being the right instrument for #81: it read `1` in 16 of 16
# tracked samples, INCLUDING the contaminated one.
#
# THE BOUND. 1.25 on the 5-minute average, built from three tracked quantities:
#   - the driver's own floor: every arm is GOMAXPROCS=1, one runnable thread, so a between-
#     arms sample carries ~1.0 of the driver's own load. Confirmed, not assumed: the 15
#     clean samples span [0.98, 1.02] once an arm has run.
#   - headroom: 1.25 clears the clean maximum 1.02 by 0.23, which is 11.5x the 0.02 clean
#     half-spread. Measured consequence on the tracked data: 0 of 15 clean samples refuse.
#   - margin: the tracked excursion is 2.17, refused with 0.92 to spare. 1 of 1.
# The one free parameter is that 0.23 of headroom, and its meaning is stated rather than
# hidden: a co-tenant sustaining less than about a quarter of one cpu over five minutes is
# invisible to this precondition. Named, not carried as a debt -- the action that would
# remove it is a longer per-arm sampling window, which costs run time the campaign does not
# need to spend to answer its question.
QUIET_L5_MAX="${QUIET_L5_MAX:-1.25}"

say "provenance and the frozen-tree guard"
date -u +%FT%TZ
echo "settings: HOST=$HOST FILTER=$FILTER COUNT=$COUNT BTIME=$BTIME"
echo "settings: ARMS_A=[$ARMS_A] ARMS_B=[$ARMS_B] TAG=${TAG:-<none>}"
echo "settings: QUIET_L5_MAX=$QUIET_L5_MAX WANT_SHA=$WANT_SHA GOTOOLCHAIN=$GOTOOLCHAIN"
echo "driver host: $(hostname)"
echo "rev: $FULLREV"
# The tmux server this may run under inherits an environment, and a stale host list in it
# once turned a healthy run RED with zero FAILs. KEEL_PIN_CPUS is in this grep deliberately:
# an inherited one would silently override every arm's mask with a single value. GOGC and
# GODEBUG are in it for this test specifically -- an inherited GOGC=off would give the `c0`
# control the `gc` arm's treatment, and the collapse control is what every other arm is read
# against.
echo "inherited KEEL_*/HOST/GO* env: $(env | /usr/bin/grep -E '^(KEEL|HOST|GOGC|GODEBUG|GOMAXPROCS)=' | tr '\n' ' ' || true)"
for v in KEEL_PIN_CPUS GOGC GODEBUG GOMAXPROCS; do
  if [[ -n "${!v:-}" ]]; then
    echo "REFUSED: $v=${!v} is set in the inherited environment. Every arm of this run sets its"
    echo "  own environment; an inherited value would apply a treatment to the controls and the"
    echo "  null result this run predicts would look real. Unset it and relaunch."
    exit 4
  fi
done
dirty="$(git status --porcelain)"
if [[ -n "$dirty" ]]; then
  echo "REFUSED: the tree is dirty, so a log from this run could not be attributed to a revision."
  sed 's/^/  /' <<<"$dirty"
  exit 4
fi
echo "tree clean: yes"

# shellcheck source=/dev/null
source scripts/remote.sh
echo "sourced scripts/remote.sh; KEEL_REMOTE_DIR=$KEEL_REMOTE_DIR"
echo "go: $(go version)"

say "topology preflight: ref is DERIVED as two cores, not asserted to be"
# Only `ref` makes a topology claim in this test -- there is no smt arm -- but it is the
# recovered control, so if 0,1 were one core's two threads every arm would sit at the
# confined level and the run would read as a null result for the most boring reason there is.
topo="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" '
  T=/sys/devices/system/cpu
  for c in 0 1; do echo "siblings$c=$(cat $T/cpu$c/topology/thread_siblings_list 2>/dev/null)"; done
  echo "online=$(cat $T/online 2>/dev/null)"' 2>&1)"
rc=$?
sed 's/^/   /' <<<"$topo"
[[ "$rc" -eq 0 ]] || { echo "REFUSED: could not read topology from $HOST (rc=$rc)"; exit 6; }
first_sib() { local v="${1%%,*}"; printf '%s' "${v%%-*}"; }
s0="$(sed -n 's/^siblings0=//p' <<<"$topo")"
s1="$(sed -n 's/^siblings1=//p' <<<"$topo")"
echo "derived: core(0)=$(first_sib "$s0") core(1)=$(first_sib "$s1")"
[[ -n "$s0" && -n "$s1" ]] || { echo "REFUSED: sysfs did not report thread siblings for cpu0 or cpu1"; exit 7; }
[[ "$(first_sib "$s0")" != "$(first_sib "$s1")" ]] || {
  echo "REFUSED: cpu0 and cpu1 are siblings of ONE core ($s0), so ref would not be two cores and"
  echo "  every arm in this run would sit at the confined level for a topology reason."; exit 7; }
echo "preflight: OK -- ref=0,1 is two distinct physical cores"

say "build: ONE binary, used by ALL arms, and it must be the boundaries' own binary"
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN"
rc=$?
echo "remote_build_test rc=$rc"
[[ "$rc" -eq 0 ]] || { echo "no binary, no arms"; exit 5; }
sha="$({ shasum -a 256 "$BIN" 2>/dev/null || sha256sum "$BIN"; } | cut -c1-16)"
echo "binary: sha256=${sha} bytes=$(wc -c <"$BIN" | tr -d ' ') flags=[$(build_settings "$BIN")]"
echo "toolchain read off the artifact: $(builder_toolchain "$BIN")"
if [[ "$sha" != "$WANT_SHA" ]]; then
  echo "REFUSED: this binary is $sha and the registered boundaries were derived on $WANT_SHA."
  echo "  predictions-mech148.py's BOUNDS are ABSOLUTE GFLOP/s, not ratios against this run's"
  echo "  own reference, which is what makes them immune to a contaminated reference -- and the"
  echo "  price of that immunity is that they transfer to no other artifact. Rescaling them to a"
  echo "  new binary would make them chosen rather than derived, so this run is not taken."
  exit 8
fi
echo "binary IDENTICAL to the boundaries' binary: BOUNDS apply as registered"

say "treatment witness: does GOGC=off actually arrive, and does it have anything to remove?"
# NON-JUDGED probe, and the reason it exists is in the header: predicting the null means a
# treatment that never applied corroborates me. It runs the SAME binary twice with gctrace on,
# once with GOGC=off, and reads TWO different things off the runtime's own `gc #` lines.
#
# WHICH FIELD, and this is the SECOND time this campaign that a plausible field was blind to the
# event it was chosen to see. The first version counted `gc #` LINES and refused this run at
# exit 9 on `2 vs 2` -- a false refusal, because every cycle at these parameters is `(forced)`,
# i.e. `runtime.GC()` called by `testing`'s own runN before it resets the timer, and GOGC has no
# authority over a forced cycle. The witness was in the same line all along: the HEAP GOAL, which
# the runtime prints as it decided it. Measured on this host, first line of each probe:
#     GC on:     0->0->0 MB,             4 MB goal, ... (forced)
#     GOGC=off:  0->0->0 MB, 8532210231539 MB goal, ... (forced)
# That is the runtime reporting, in its own voice, that it honoured GOGC=off. A `>` on the goal
# has a known positive (4 against 8.5e12) where the line count had none, which is the check the
# first version skipped: replay the candidate field against a case where the treatment MUST show.
#
# The second field is the count of NON-forced cycles, and it is a disclosure rather than a gate.
gcprobe() {
  # SC2029 is the intent, not a slip: $KEEL_REMOTE_DIR and $FILTER must expand HERE, because
  # the far side has no keel checkout and no such variables.
  # shellcheck disable=SC2029
  ssh "${KEEL_SSH_OPTS[@]}" "$HOST" \
    "cd '$KEEL_REMOTE_DIR' && env $1 GODEBUG=gctrace=1 taskset -c 0 ./bench.test \
       -test.run=NONE -test.bench='$FILTER/scalar' -test.benchtime=200x -test.count=1 2>&1 \
     | /usr/bin/grep '^gc [0-9]'" 2>/dev/null
}
# The heap goal in MB off the FIRST gc line, empty if there is no parsable one.
gcgoal() { sed -n 's/^gc [0-9].*, \([0-9][0-9]*\) MB goal,.*/\1/p' <<<"$1" | head -1; }
# Cycles the runtime started on its own. `grep -c` cannot be used: it exits 1 on zero matches,
# and zero is the answer this campaign turned out to have.
gcnonforced() { local n; n="$(/usr/bin/grep '^gc [0-9]' <<<"$1" | /usr/bin/grep -v '(forced)' | /usr/bin/grep -c . )"; printf '%s' "${n:-0}"; }
gc_on_raw="$(gcprobe '')"; gc_off_raw="$(gcprobe 'GOGC=off')"
goal_on="$(gcgoal "$gc_on_raw")"; goal_off="$(gcgoal "$gc_off_raw")"
nf_on="$(gcnonforced "$gc_on_raw")"; nf_off="$(gcnonforced "$gc_off_raw")"
echo "heap goal with GC on: ${goal_on:-<unparsable>} MB   with GOGC=off: ${goal_off:-<unparsable>} MB"
echo "heap-triggered (non-forced) cycles: GC on $nf_on, GOGC=off $nf_off"
if [[ -z "$goal_on" || -z "$goal_off" ]]; then
  echo "WARNING: the gctrace probe produced no parsable heap goal, so the gc and both arms'"
  echo "  treatment is UNWITNESSED. The arms still run and are still scored, but the analyzer"
  echo "  must read a confined result on them as consistent with 'no effect' AND 'no treatment'."
elif awk -v a="$goal_off" -v b="$goal_on" 'BEGIN{exit !(a>b)}'; then
  echo "treatment ARRIVED: GOGC=off raises the heap goal $goal_on -> $goal_off MB, which is the"
  echo "  runtime saying in its own output that it honoured the variable."
else
  echo "REFUSED: GOGC=off did not raise the heap goal ($goal_off vs $goal_on), so the runtime did"
  echo "  not honour it and the gc and both arms would carry no treatment at all."
  exit 9
fi
# ...and now the part that is a finding rather than a check, measured 2026-09-03 BEFORE any arm
# ran and therefore before any outcome sample exists. At the ARMS' own parameters (benchtime=1s,
# the full BenchmarkKernel filter) this binary produces 71 gc cycles and *0* heap-triggered ones,
# with and without the treatment. Every cycle is forced by testing's runN, which calls
# runtime.GC() and only then resets the timer, so none of them is inside a timed window either.
# So GOGC=off ARRIVES and REMOVES NOTHING: there is no GC activity in the timed region for it to
# take away. That is the asyncpreemptoff shape reached from the other side, and it is disclosed
# in predictions-mech148.py's OUT_OF_DOMAIN rather than fixed, because nothing here is broken.
# The gc and both arms still RUN -- their samples are further draws of the confined level, which
# rule 25 wants more of -- but a confirmed `unimodal-at-confined` on them is not evidence about
# the GC hypothesis. It cannot be: the mechanism was already absent.
if [[ "$nf_on" == "0" ]]; then
  echo "NOTE: 0 heap-triggered cycles with GC ON, so the gc treatment has nothing to remove here."
  echo "  The gc/both cells are OUT OF DOMAIN for the GC hypothesis by predictions-mech148.py's"
  echo "  item 5; they still run, and still characterize the confined level."
fi
# asyncpreemptoff has no equally cheap witness: unknown GODEBUG keys are ignored silently and
# preemption signals are not counted in any output this binary produces. So arrival is proven
# (the GOMAXPROCS canary) and HONOURING is not. Declared out of domain rather than carried as
# a debt; the action that would close it is a runtime built with preemption counters, which is
# the same patched-runtime action predictions-mech148.py's OUT_OF_DOMAIN already names.

# --- host sampling, and the per-arm quietness gate --------------------------------------
HS=""  # the last host sample's raw text; set by hostsample, read by quiet_violation
hostsample() {
  echo "-- host sample: $1 --"
  HS="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" '
    printf "uptime: "; uptime
    printf "loadavg: "; cat /proc/loadavg
    printf "freq_khz:"; for c in 0 1 2 3 6 7; do printf " cpu%s=%s" "$c" "$(cat /sys/devices/system/cpu/cpu$c/cpufreq/scaling_cur_freq 2>/dev/null || echo NA)"; done; echo
    printf "governor: "; cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo NA
    echo "top by cpu:"; ps -eo pid,pcpu,etimes,comm --sort=-pcpu | head -4 | sed "s/^/  /"
    printf "utc: "; date -u +%FT%TZ' 2>&1)"
  sed 's/^/   /' <<<"$HS"
}
# quiet_violation -- echoes a reason and returns 0 if the LAST host sample is not quiet.
# Returns 1 (quiet) only on a reading that exists and is under the bound, so an unreadable
# sample refuses. Kept in the caller's shell rather than a process substitution, because a
# guard that dies past that boundary prints nothing and fails open.
quiet_violation() {
  local l5
  l5="$(sed -n 's/^loadavg: [0-9.]* \([0-9.]*\) .*/\1/p' <<<"$HS" | head -1)"
  if [[ -z "$l5" ]]; then
    printf 'the host sample carried no parsable /proc/loadavg line, so quietness is unknown'
    return 0
  fi
  if awk -v v="$l5" -v b="$QUIET_L5_MAX" 'BEGIN{exit !(v>b)}'; then
    printf '5-minute load %s exceeds the bound %s derived from test 2 (clean max 1.02 over 15 samples; tracked excursion 2.17)' "$l5" "$QUIET_L5_MAX"
    return 0
  fi
  printf '5-minute load %s' "$l5"
  return 1
}

env_for() {
  case "$1" in
    ref|c0) printf 'GOMAXPROCS=1' ;;
    gc)     printf 'GOMAXPROCS=1 GOGC=off' ;;
    pre)    printf 'GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1' ;;
    both)   printf 'GOMAXPROCS=1 GOGC=off GODEBUG=asyncpreemptoff=1' ;;
    *)      printf '' ;;
  esac
}
mask_for()  { case "$1" in ref) printf '0,1' ;; c0|gc|pre|both) printf '0' ;; *) printf '' ;; esac; }
cores_for() { case "$1" in ref) printf '2' ;; c0|gc|pre|both) printf '1' ;; *) printf '' ;; esac; }

arm() {
  local pass="$1" name="$2" label="$1$2" cpus aenv log rc pinline got why
  cpus="$(mask_for "$name")"; aenv="$(env_for "$name")"
  log="$OUT/bench-mech148-$REV$TAG-$label.txt"
  say "arm $label: pass $pass, KEEL_PIN_CPUS=$cpus, env=[$aenv]"
  hostsample "before $label"
  # The gate. Sampled between arms and never during one, so the reading it refuses on is not
  # this arm's own load.
  if why="$(quiet_violation)"; then
    echo "arm $label: UNMEASURED -- host not quiet: $why"
    echo "arm $label: not run, so this is an absent measurement and not a slow one. The other"
    echo "  arms are unaffected; a co-tenant costs the arms it overlaps."
    return 0
  fi
  echo "arm $label: host quiet ($why), proceeding"

  KEEL_PIN_CPUS="$cpus" \
  KEEL_REMOTE_ENV="$aenv" \
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
  pinline="$(/usr/bin/grep -m1 '^keel-pin:' "$log")"
  echo "arm $label: keel-pin line: $pinline"
  # THE CANARY, asserted rather than printed. GOMAXPROCS is the first assignment in every
  # env string, so if remote.sh's unquoted interpolation ever stopped word-splitting, this is
  # the value that breaks -- and every treatment in this run arrives by the same path.
  got="$(sed -n 's/^keel-bench-gomaxprocs: *//p' "$log" | head -1)"
  if [[ "$got" == "1" ]]; then
    echo "arm $label: gomaxprocs readback OK: 1, so the env string word-split as intended"
  else
    echo "arm $label: WARNING, gomaxprocs readback is '${got:-<absent>}' and every arm requires 1."
    echo "  The env string did not word-split, so this arm's treatment did not arrive either and"
    echo "  the analyzer must treat it as unmeasured rather than as a confined result."
  fi
  got="$(sed -n 's/.* cores=\([0-9,]*\).*/\1/p' <<<"$pinline" | tr ',' '\n' | sort -u | /usr/bin/grep -c . )"
  if [[ "$got" == "$(cores_for "$name")" ]]; then
    echo "arm $label: cores readback OK: $got distinct physical core(s), as the design requires"
  else
    echo "arm $label: WARNING, cores readback is $got and the design requires $(cores_for "$name"). This arm does not measure what it was built to measure and the analyzer must treat it as unmeasured."
  fi
  hostsample "after $label"
}

for a in $ARMS_A; do arm a "$a"; done
for a in $ARMS_B; do arm b "$a"; done

say "done"
date -u +%FT%TZ
