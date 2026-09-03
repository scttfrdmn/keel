#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# `#150`'s instrument: sample cpu0's frequency WHILE an arm runs, not only between arms.
#
# `#150` recorded a 13-16% between-arm level term on janus that no treatment explains, and its
# leading hypothesis -- a clock change -- is untestable with the data in hand, because `#81`'s gap
# means every frequency reading in every driver log is an IDLE reading taken between arms. This
# driver closes that gap and runs test 3's ten arms again underneath it.
#
# The boundaries, the phase order, the classifier, the prediction, the falsifier and the
# out-of-domain list are in archive/freq150/predictions-freq150.py, committed BEFORE this file runs
# and imported by the analyzer rather than restated in it. Read that first.
#
# TWO PHASES, AND THE HUNT GOES FIRST. Deliberately inverted: the elevation appeared at run
# positions 1 and 4 of ten, so a cold host is part of the only condition under which the phenomenon
# has ever been seen, and an hour of control arms first would spend it. The control's contrast is
# internal and its order is a palindrome, so it does not care about host warmth. It pays.
#
#   phase hunt      10 arms, test 3's exact sequence and reversal, SAMPLER ON throughout
#   phase control    4 arms of the `pre` config, sampler off/on/on/off
#
# THE SAMPLER IS A CO-TENANT BY CONSTRUCTION, which is Scott's condition on this run: its own
# perturbation is MEASURED, not assumed. An instrument measures a noun; it must not become the
# noun. Hence the control phase, and hence the palindrome -- under monotone drift in host state,
# arms 1 and 4 (off) straddle arms 2 and 3 (on), so drift enters both terms of the on-vs-off
# contrast with the same sign and cancels to first order. An unreversed off/on pair cannot tell the
# sampler's cost from the drift between two arms; this campaign has already been bitten by exactly
# that (docs/issue148-mech-205a7a8.md section 6).
#
# WHAT IS NEW HERE BESIDES THE SAMPLER: the quietness guard is PEDESTAL-SUBTRACTED, per `#149`'s
# ruling, and the pedestal is DERIVED -- archive/freq150/derive-pedestal.py reads both tracked
# campaign logs and measures it. See the guard section below for the numbers and the one place they
# are hardcoded.
#
# No `set -e`: every step reports its own rc rather than vanishing, and a killed run must not be
# able to look like a verdict.
set -uo pipefail

cd /Users/scttfrdmn/src/keel || exit 3

say() { printf '\n=== %s ===\n' "$*"; }

REV="$(git rev-parse --short HEAD)"
FULLREV="$(git rev-parse HEAD)"
HOST="${HOST:-janus.local}"
# Overridable so a smoke run can exercise THIS script rather than a second, differently worded copy
# of it. The judged values are the defaults and the log records which were used. `-` and not `:-`
# on the arm lists: with `:-`, HUNT_A="" falls back to the full default, so asking for one arm
# silently runs five (found by driver-width148.sh's smoke run).
FILTER="${FILTER:-BenchmarkKernel}"
COUNT="${COUNT:-30}"
BTIME="${BTIME:-1s}"
HUNT_A="${HUNT_A-ref c0 gc pre both}"
HUNT_B="${HUNT_B-both pre gc c0 ref}"
CONTROL="${CONTROL-k1:off k2:on k3:on k4:off}"
CONTROL_ARM="${CONTROL_ARM:-pre}"
# The binary test 3 observed the elevation on. The REASON this is a refusal differs from test 3's:
# there, the registered bounds were absolute GFLOP/s that transfer to no other artifact. Here every
# criterion is a ratio computed inside this run, so a rebuild could be absorbed -- but the SUBJECT
# could not. The phenomenon under investigation was observed on this binary, and reproducing a
# phenomenon is this run's precondition, so the artifact is pinned to the one it appeared on.
WANT_SHA="${WANT_SHA:-d0d46d26c15cc8b2}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.27.0}"   # yields that sha; go1.27.1 does not (T-arm: exit 8)
TAG="${TAG:-}"
OUT="build"
SAMPLE_PERIOD="${SAMPLE_PERIOD:-0.2}"
SAMPLED_CPU="${SAMPLED_CPU:-0}"

# --- the quietness guard: pedestal-subtracted, and the pedestal is derived --------------------
#
# `#149` ruled the flat bound stood for test 3 and the DEFECT filed rather than fixed: at 1.25 on
# the raw 5-minute load, the first arm of a pass had ~1.08 of headroom and every later arm ~0.24,
# because the driver's own preceding arm contributes ~1.0 that the bound never accounted for. Same
# guard, two sensitivities, decided by run position -- which is no property of any co-tenant.
#
# The fix, with Scott's condition attached ("the pedestal is derived, not assumed. Don't write 1.00
# because one pinned thread 'should' read 1.00"): subtract the driver's OWN contribution, computed
# from its own recorded busy timeline by the same exponentially weighted average the kernel uses to
# produce the number being read -- 300 s time constant, which is the DEFINITION of the 5-minute
# field and not a model of it. Then bound what is left.
#
# Both constants below are outputs of archive/freq150/derive-pedestal.py over the 20 distinct
# instants in the two tracked campaign logs (36 reads; 16 were duplicate reads of an instant and
# are collapsed BY INSTANT, not by read, so the n is not wrong in the direction of confidence):
#
#   SELF_N = 0.9800   min of l5/P over 17 saturated clean instants, which span 0.980..1.054.
#                     NOT 1.00. Taking the minimum underestimates the pedestal, which overestimates
#                     the residue, which errs toward FALSE REFUSAL -- the direction #149 ruled
#                     recoverable, as against a quiet null which is not.
#   QUIET_FOREIGN_MAX = 0.27 = test 3's own 1.25 MINUS SELF_N. Derived from its predecessor rather
#                     than chosen against these residues: at saturation the pedestal absorbs
#                     SELF_N, so this leaves the guard's sensitivity exactly where a bound that
#                     predates this run put it, and changes only its position dependence. A rounder
#                     0.30 would have made the saturated threshold 1.28 -- LOOSER than the guard it
#                     replaces, and looser by an amount picked after seeing the numbers.
#
# Scored on the record: 0 of 18 clean instants refused; the 1 tracked co-tenant excursion (2.17 at
# `after bref`, core148) refused with 0.92 to spare; effective L5 bound 0.27 cold, 1.25 saturated,
# so never looser than 1.25 at any position and 4.6x tighter at the first arm.
#
# What it still cannot see, stated rather than carried as a debt: a co-tenant sustaining under 0.27
# of one cpu over five minutes. The action that would shrink that is a longer per-arm sampling
# window, which costs run time this question does not need.
SELF_N="${SELF_N:-0.98}"
QUIET_FOREIGN_MAX="${QUIET_FOREIGN_MAX:-0.27}"

# OWN_WINDOWS: "start end" epoch pairs, one line each, appended by EVERY step that runs work on the
# host -- the build probe as well as the arms. A newline-joined string and not an array, because the
# driver host is darwin and its bash is 3.2, where "${arr[@]}" on an empty array trips `set -u`.
OWN_WINDOWS=""
T0="$(date +%s)"
own_add() { OWN_WINDOWS="${OWN_WINDOWS}$1 $2"$'\n'; }

# pedestal_at EPOCH -- echoes SELF_N * P, the driver's own expected contribution to a 5-minute load
# read at EPOCH. n=1 inside a recorded window (every arm is GOMAXPROCS=1 pinned, and that is
# witnessed per arm by the gomaxprocs readback below), 0 in the gaps.
pedestal_at() {
  printf '%s' "$OWN_WINDOWS" | awk -v t="$1" -v t0="$T0" -v sn="$SELF_N" '
    NF==2 { s[++n]=$1; e[n]=$2 }
    END { decay=exp(-1/300); p=0
          for (sec=t0; sec<t; sec++) { b=0
            for (i=1;i<=n;i++) if (sec>=s[i] && sec<e[i]) { b=1; break }
            p = p*decay + b*(1-decay) }
          printf "%.4f", sn*p }'
}

say "provenance and the frozen-tree guard"
date -u +%FT%TZ
echo "settings: HOST=$HOST FILTER=$FILTER COUNT=$COUNT BTIME=$BTIME"
echo "settings: HUNT_A=[$HUNT_A] HUNT_B=[$HUNT_B] CONTROL=[$CONTROL] CONTROL_ARM=$CONTROL_ARM"
echo "settings: SAMPLE_PERIOD=$SAMPLE_PERIOD SAMPLED_CPU=$SAMPLED_CPU TAG=${TAG:-<none>}"
echo "settings: SELF_N=$SELF_N QUIET_FOREIGN_MAX=$QUIET_FOREIGN_MAX WANT_SHA=$WANT_SHA GOTOOLCHAIN=$GOTOOLCHAIN"
echo "driver host: $(hostname)"
echo "rev: $FULLREV"
# The tmux server this runs under inherits an environment, and a stale host list in one once turned
# a healthy run RED with zero FAILs. KEEL_PIN_CPUS especially: an inherited one would silently
# override every arm's mask with a single value.
echo "inherited KEEL_*/HOST/GO* env: $(env | /usr/bin/grep -E '^(KEEL|HOST|GOGC|GODEBUG|GOMAXPROCS)=' | tr '\n' ' ' || true)"
for v in KEEL_PIN_CPUS GOGC GODEBUG GOMAXPROCS; do
  if [[ -n "${!v:-}" ]]; then
    echo "REFUSED: $v=${!v} is set in the inherited environment. Every arm of this run sets its own"
    echo "  environment; an inherited value would apply a treatment to the controls."
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

say "topology preflight: ref's two-ness and the SAMPLER's cpu, both derived"
# Two claims here, not one. `ref`=0,1 must be two physical cores or every arm sits at the confined
# level for a topology reason. And the sampler's cpu must share no SMT thread with any measured cpu
# -- if it did, it would be a co-tenant of the very core whose frequency it reports, and the number
# it produced would describe a cpu it had changed. Both are READ off sysfs; neither is asserted.
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
online="$(sed -n 's/^online=//p' <<<"$topo")"
[[ -n "$s0" && -n "$s1" && -n "$online" ]] || { echo "REFUSED: sysfs did not report siblings/online"; exit 7; }
echo "derived: core(0)=$(first_sib "$s0") core(1)=$(first_sib "$s1")"
[[ "$(first_sib "$s0")" != "$(first_sib "$s1")" ]] || {
  echo "REFUSED: cpu0 and cpu1 are siblings of ONE core ($s0), so ref would not be two cores and"
  echo "  every arm in this run would sit at the confined level for a topology reason."; exit 7; }
# The forbidden set is the union of the measured cpus' sibling lists, DERIVED, then cross-checked
# against the registration's FORBIDDEN_SAMPLER_CPUS. A mismatch means the host's topology is not
# the one the registration was written against, which is a refusal and not a note.
FORBIDDEN="$(printf '%s,%s' "$s0" "$s1" | tr ',' '\n' | sort -n -u | tr '\n' ' ')"
echo "derived forbidden sampler cpus: [$(echo "$FORBIDDEN" | tr -s ' ')]"
want_forbidden="$(python3 -c 'import sys;sys.path.insert(0,"archive/freq150");import importlib.util as u
s=u.spec_from_file_location("p","archive/freq150/predictions-freq150.py");m=u.module_from_spec(s);s.loader.exec_module(m)
print(" ".join(str(c) for c in sorted(m.FORBIDDEN_SAMPLER_CPUS)))')"
if [[ "$(echo "$FORBIDDEN" | tr -s ' ' | sed 's/ $//')" != "$want_forbidden" ]]; then
  echo "REFUSED: the derived forbidden set differs from the registered one."
  echo "  derived: [$FORBIDDEN]  registered: [$want_forbidden]"
  echo "  The registration was written against a topology this host no longer has, so a sampler cpu"
  echo "  chosen under it could be an SMT sibling of a measured cpu."
  exit 7
fi
echo "forbidden set MATCHES predictions-freq150.py's FORBIDDEN_SAMPLER_CPUS: [$want_forbidden]"
# The sampler's own cpu: the lowest online cpu not in the forbidden set. Lowest rather than
# arbitrary so the choice is reproducible from the log. Two functions rather than one inline loop
# so that test-driver-freq150.sh can drive them against topologies this host does not have --
# including the one where there is nowhere left to put the sampler.
expand_online() {   # "0,1,4-6" -> one cpu per line. sysfs writes both forms and mixes them.
  tr ',' '\n' <<<"$1" | awk -F- 'NF==2{for(i=$1;i<=$2;i++)print i} NF==1&&$1!=""{print $1}'
}
pick_scpu() {       # $1 = online spec, $2 = space-separated forbidden set. Empty output = nowhere.
  local c
  for c in $(expand_online "$1"); do
    case " $2 " in *" $c "*) continue ;; esac
    printf '%s' "$c"; return 0
  done
  return 1
}
SCPU="$(pick_scpu "$online" "$FORBIDDEN")"
[[ -n "$SCPU" ]] || { echo "REFUSED: no online cpu outside the forbidden set; the sampler has nowhere to run"; exit 7; }
echo "sampler cpu DERIVED: $SCPU (lowest online cpu not in the forbidden set)"

say "preflight quietness: refuse the RUN, not its first arm"
# Why this exists and is separate from the per-arm gate: the hunt's phenomenon appeared at run
# position 1, so a cold quiet host is part of the condition being reproduced. If the per-arm gate
# were the only check, a contaminated host would refuse arm 1 -- spending the cold start on an
# UNMEASURED arm and warming the host for the rest. Refusing here costs nothing and loses nothing.
HS=""
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
# quiet_violation -- echoes a reason and returns 0 if the LAST host sample is not quiet, 1 if it is.
# Returns quiet only on a reading that exists and is under the bound, so an unreadable sample
# refuses. Kept in the caller's shell, never a process substitution: `set -u` stops at that
# boundary and a guard that dies past it prints nothing and fails open.
quiet_violation() {
  local l5 ped foreign
  l5="$(sed -n 's/^loadavg: [0-9.]* \([0-9.]*\) .*/\1/p' <<<"$HS" | head -1)"
  if [[ -z "$l5" ]]; then
    printf 'the host sample carried no parsable /proc/loadavg line, so quietness is unknown'
    return 0
  fi
  ped="$(pedestal_at "$(date +%s)")"
  if [[ -z "$ped" ]]; then
    printf 'the pedestal did not compute, so the residue this guard bounds is unknown'
    return 0
  fi
  foreign="$(awk -v a="$l5" -v b="$ped" 'BEGIN{printf "%.3f", a-b}')"
  if awk -v v="$foreign" -v b="$QUIET_FOREIGN_MAX" 'BEGIN{exit !(v>b)}'; then
    printf 'foreign load %s exceeds %s (5-min %s minus this driver'\''s own derived pedestal %s)' \
      "$foreign" "$QUIET_FOREIGN_MAX" "$l5" "$ped"
    return 0
  fi
  printf '5-min %s - pedestal %s = foreign %s' "$l5" "$ped" "$foreign"
  return 1
}
hostsample "before anything ran"
if why="$(quiet_violation)"; then
  echo "REFUSED at preflight: $why"
  echo "  Nothing has been built or run, so this costs one ssh. Relaunch when the host is quiet:"
  echo "  refusing here keeps the cold-host start, which is part of the condition #150's phenomenon"
  echo "  was observed under, instead of spending it on an arm that would be UNMEASURED anyway."
  exit 10
fi
echo "preflight: host quiet ($why)"

say "build: ONE binary, used by ALL arms, and it must be the binary the elevation appeared on"
bstart="$(date +%s)"
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN"
rc=$?
own_add "$bstart" "$(date +%s)"   # the build's own remote work enters the pedestal timeline
echo "remote_build_test rc=$rc"
[[ "$rc" -eq 0 ]] || { echo "no binary, no arms"; exit 5; }
sha="$({ shasum -a 256 "$BIN" 2>/dev/null || sha256sum "$BIN"; } | cut -c1-16)"
echo "binary: sha256=${sha} bytes=$(wc -c <"$BIN" | tr -d ' ') flags=[$(build_settings "$BIN")]"
echo "toolchain read off the artifact: $(builder_toolchain "$BIN")"
if [[ "$sha" != "$WANT_SHA" ]]; then
  echo "REFUSED: this binary is $sha and the elevation under investigation was observed on"
  echo "  $WANT_SHA. Every criterion here is a ratio inside this run, so a rebuild would not"
  echo "  invalidate the arithmetic -- it would change the SUBJECT, and reproducing the subject is"
  echo "  this run's precondition."
  exit 8
fi
echo "binary IDENTICAL to the binary #150's elevation was measured on"
# No gctrace treatment witness here, unlike test 3: no criterion in predictions-freq150.py depends
# on GOGC=off arriving. The gc and both arms are carried only because re-ordering the sequence would
# forfeit the phenomenon's observed positions, and test 3 already witnessed that treatment on this
# same binary (docs/issue148-mech-205a7a8.md section 2).

say "sampler install, and its own preconditions"
# Installed as a FILE and hash-checked, rather than interpolated into an ssh command line, so that
# what ran is recoverable from the log and so the quoting has one place to be wrong.
SAMPLER_REMOTE="$KEEL_REMOTE_DIR/freq150-sampler.sh"
SAMPLER_LOCAL="$(mktemp -d)/freq150-sampler.sh"
cat >"$SAMPLER_LOCAL" <<'SAMPLER'
#!/usr/bin/env bash
# freq150-sampler.sh CPU PERIOD OUT -- appends "<epoch.frac> <khz>" per sample.
# Zero forks per sample: $EPOCHREALTIME is a bash variable and `read` is a builtin. Appending with
# >> per line rather than redirecting the whole loop is deliberate -- it settles stdio buffering by
# construction instead of by argument, so the trace is complete even if the sampler is killed
# between samples, and a reader can see it grow.
set -u
p="/sys/devices/system/cpu/cpu$1/cpufreq/scaling_cur_freq"
while :; do
  read -r khz < "$p" || khz=NA
  printf '%s %s\n' "$EPOCHREALTIME" "$khz" >> "$3"
  sleep "$2"
done
SAMPLER
shash="$({ shasum -a 256 "$SAMPLER_LOCAL" 2>/dev/null || sha256sum "$SAMPLER_LOCAL"; } | cut -c1-16)"
echo "sampler: $(wc -l <"$SAMPLER_LOCAL" | tr -d ' ') lines, sha256=$shash"
sed 's/^/   | /' "$SAMPLER_LOCAL"
# SHIPPED BY scp, NOT BY ssh's STDIN, and this driver's first launch is why. `KEEL_SSH_OPTS` carries
# `-n` (scripts/remote.sh:29), which redirects ssh's stdin from /dev/null -- so `cat > file` on the
# far side received nothing and wrote an EMPTY sampler. The hash check below caught it (remote
# e3b0c442..., the sha256 of the empty string, against b9d9c233...) and the run refused at exit 11
# before any arm ran, which is that check earning its place: an empty sampler is a sampler that
# produces a complete-looking trace of nothing. remote.sh already keeps `KEEL_SCP_OPTS` for exactly
# this -- the same options minus `-n` (scripts/remote.sh:30) -- and already ships its own runner
# script by scp, so the fix is the repo's existing instrument rather than a new one.
ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "mkdir -p '$KEEL_REMOTE_DIR'" >/dev/null 2>&1
scp -q "${KEEL_SCP_OPTS[@]}" "$SAMPLER_LOCAL" "$HOST:$SAMPLER_REMOTE"
echo "scp of the sampler: rc=$?"
# shellcheck disable=SC2029  # every $VAR here must expand on THIS side
pre="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "
  chmod +x '$SAMPLER_REMOTE'
  echo \"remote sha: \$(sha256sum '$SAMPLER_REMOTE' | cut -c1-16)\"
  echo \"bash: \$(bash --version | head -1)\"
  echo \"epochrealtime: \$(bash -c 'echo \$EPOCHREALTIME')\"
  t0=\$(date +%s.%N); sleep $SAMPLE_PERIOD; t1=\$(date +%s.%N)
  echo \"fractional sleep $SAMPLE_PERIOD took: \$(awk -v a=\$t0 -v b=\$t1 'BEGIN{printf \"%.3f\", b-a}')s\"
  echo \"freq readable: \$(cat /sys/devices/system/cpu/cpu$SAMPLED_CPU/cpufreq/scaling_cur_freq 2>&1)\"
  echo \"taskset: \$(command -v taskset || echo MISSING)\"
" 2>&1)"
sed 's/^/   /' <<<"$pre"
# Each precondition is asserted against its own phrase, and a phrase that matches nothing fails
# closed. Three ways the sampler could silently become the noun it measures:
#   * a shell without $EPOCHREALTIME -> the fallback would be a fork per sample;
#   * a `sleep` that rejects a fraction -> a spin loop at 100% of a cpu;
#   * an unreadable sysfs file -> a trace of the string "NA", which no criterion can use.
got_sha="$(sed -n 's/^remote sha: //p' <<<"$pre")"
[[ "$got_sha" == "$shash" ]] || { echo "REFUSED: sampler on the host hashes $got_sha, not $shash"; exit 11; }
ert="$(sed -n 's/^epochrealtime: //p' <<<"$pre")"
[[ -n "$ert" ]] || { echo "REFUSED: the host's bash has no \$EPOCHREALTIME, so each sample would fork date(1)"; exit 11; }
slept="$(sed -n 's/^fractional sleep .* took: \([0-9.]*\)s/\1/p' <<<"$pre")"
if [[ -z "$slept" ]] || awk -v v="$slept" -v p="$SAMPLE_PERIOD" 'BEGIN{exit !(v+0 < p/2 || v+0 > p*5)}'; then
  echo "REFUSED: 'sleep $SAMPLE_PERIOD' measured ${slept:-<unparsable>}s on the host. A sleep that"
  echo "  rounds a fraction to zero turns this sampler into a spin loop on cpu $SCPU, which is the"
  echo "  one thing an instrument for a FREQUENCY question must not be."
  exit 11
fi
freqline="$(sed -n 's/^freq readable: //p' <<<"$pre")"
[[ "$freqline" =~ ^[0-9]+$ ]] || { echo "REFUSED: scaling_cur_freq for cpu$SAMPLED_CPU read '$freqline'"; exit 11; }
grep -q '^taskset: /' <<<"$pre" || { echo "REFUSED: no taskset on the host; the sampler cannot be pinned"; exit 11; }
echo "sampler preconditions: OK (hash matches, EPOCHREALTIME=$ert, sleep honoured, freq $freqline kHz, taskset present)"

SPID=""
# THE BRACKET IN THE PATTERN IS LOAD-BEARING, and it was measured rather than remembered. ssh runs
# its command through a shell, so that shell's own cmdline contains the pattern, and `pgrep -f
# freq150-sampler.sh` matches IT. On janus, with no sampler running at all:
#     pgrep -c -f freq150-sampler.sh      -> 1
#     pgrep -c -f "freq150-sampl[e]r"     -> 0
# The first form is a counter that can never read zero, so every sampler-OFF control arm would have
# refused as UNMEASURED with a stray sampler it invented -- and the perturbation criterion, which is
# Scott's condition on this whole run, would have been unevaluable. `[e]` matches "e" in the target
# and does not match the literal "[e]" sitting in the invoking shell's cmdline.
SPAT='freq150-sampl[e]r'
sampler_count() { ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "pgrep -c -f '$SPAT' || true" 2>/dev/null | tr -d ' \n'; }
sampler_kill()  { ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "pkill -f '$SPAT' || true" >/dev/null 2>&1; }
sampler_start() {                       # $1 = label; sets SPID, echoes its witnesses
  local label="$1" rtrace n
  rtrace="$KEEL_REMOTE_DIR/freq150-trace-$label.txt"
  # setsid + nohup so the sampler outlives the ssh channel that started it -- the same reason every
  # long run here is launched detached. A sampler that dies with its ssh would leave a short trace,
  # which trace_present would then report as an absent measurement rather than a quiet one.
  # shellcheck disable=SC2029  # $rtrace and $SCPU must expand HERE; the host has no such vars
  SPID="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "
    rm -f '$rtrace'
    nohup setsid taskset -c $SCPU '$SAMPLER_REMOTE' $SAMPLED_CPU $SAMPLE_PERIOD '$rtrace' \
      >/dev/null 2>&1 < /dev/null &
    echo \$!" 2>/dev/null | tr -d ' \n')"
  sleep 1
  # shellcheck disable=SC2029
  n="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "wc -l < '$rtrace' 2>/dev/null || echo 0" 2>/dev/null | tr -d ' \n')"
  echo "arm $label: sampler pid=$SPID on cpu $SCPU, trace grew to ${n:-0} lines in 1s, live=$(sampler_count)"
  # The APPLIED-WITNESS for the instrument itself. A sampler that never started produces exactly the
  # output a sampler that found no frequency step produces, and this run must not be able to report
  # the second when the first happened.
  if [[ -z "${n:-}" ]] || [[ "$n" -lt 2 ]]; then
    echo "arm $label: REFUSED -- the sampler produced fewer than 2 samples in its first second, so"
    echo "  it is not running and every frequency cell of this arm would be empty for a reason no"
    echo "  criterion can distinguish from a null."
    return 1
  fi
  return 0
}
sampler_stop() {                        # $1 = label; fetches the trace, asserts the sampler is gone
  local label="$1" rtrace n live
  rtrace="$KEEL_REMOTE_DIR/freq150-trace-$label.txt"
  sampler_kill
  sleep 1
  live="$(sampler_count)"
  # shellcheck disable=SC2029
  ssh "${KEEL_SSH_OPTS[@]}" "$HOST" "cat '$rtrace'" > "$OUT/freq150-trace-$REV$TAG-$label.txt" 2>/dev/null
  n="$(/usr/bin/grep -c . "$OUT/freq150-trace-$REV$TAG-$label.txt")"
  echo "arm $label: sampler stopped, live=$live, trace fetched: $n samples -> $OUT/freq150-trace-$REV$TAG-$label.txt"
  if [[ "$live" != "0" ]]; then
    echo "REFUSED: $live sampler process(es) survived the stop. A leftover sampler is an unrecorded"
    echo "  co-tenant of every arm after this one, including the sampler-OFF control arms whose whole"
    echo "  purpose is to have none. This run is not taken."
    exit 12
  fi
}
# Kill any sampler on the way out however this driver ends, so a crashed run cannot leave a
# co-tenant on the host for whatever runs next.
# A function and not an inline ssh: inlined, the single quotes around the pattern would close the
# trap string itself, so $SPAT would expand at trap-set time and land UNQUOTED on the far side,
# where a remote shell gets to glob `[e]` for itself.
trap sampler_kill EXIT

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

# arm LABEL CONFIG SAMPLER   -- LABEL names the log, CONFIG picks mask+env, SAMPLER is on|off.
arm() {
  local label="$1" name="$2" smp="$3" cpus aenv log rc pinline got why astart
  cpus="$(mask_for "$name")"; aenv="$(env_for "$name")"
  log="$OUT/bench-freq150-$REV$TAG-$label.txt"
  say "arm $label: config $name, KEEL_PIN_CPUS=$cpus, env=[$aenv], sampler=$smp"
  hostsample "before $label"
  # The gate, sampled between arms and never during one, so the reading it refuses on is not this
  # arm's own load. Per-arm because a co-tenant arriving mid-run must cost the arms it overlaps and
  # not the ones it does not: partial evidence survives.
  if why="$(quiet_violation)"; then
    echo "arm $label: UNMEASURED -- host not quiet: $why"
    echo "arm $label: not run, so this is an absent measurement and not a slow one."
    return 0
  fi
  echo "arm $label: host quiet ($why), proceeding"
  if [[ "$smp" == "on" ]]; then
    sampler_start "$label" || { echo "arm $label: UNMEASURED -- no sampler, so not run"; return 0; }
  else
    # The ABSENCE witness, and it is not decoration: the control phase's off arms are the baseline
    # the sampler's cost is measured against, so a stray sampler here would shrink the very
    # difference it is being asked to expose -- toward the answer the author would like.
    got="$(sampler_count)"
    echo "arm $label: sampler OFF, live sampler processes on host: $got"
    if [[ "$got" != "0" ]]; then
      echo "arm $label: UNMEASURED -- $got sampler(s) running in a sampler-OFF arm, which would bias"
      echo "  the perturbation contrast toward zero. Not run."
      return 0
    fi
  fi

  astart="$(date +%s)"
  KEEL_PIN_CPUS="$cpus" \
  KEEL_REMOTE_ENV="$aenv" \
    remote_exec "$HOST" "$BIN" \
      -test.run=NONE -test.bench="$FILTER" -test.count="$COUNT" -test.benchtime="$BTIME" \
      > "$log" 2>&1
  rc=$?
  echo "arm $label: window $(date -u -r "$astart" +%FT%TZ 2>/dev/null || echo "$astart") .. $(date -u +%FT%TZ)"
  own_add "$astart" "$(date +%s)"
  [[ "$smp" == "on" ]] && sampler_stop "$label"
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
  # THE CANARY, asserted rather than printed. GOMAXPROCS leads every env string, so if remote.sh's
  # unquoted interpolation (scripts/remote.sh:1687) ever stopped word-splitting, this is the value
  # that breaks -- and every treatment in this run arrives by that same path.
  got="$(sed -n 's/^keel-bench-gomaxprocs: *//p' "$log" | head -1)"
  if [[ "$got" == "1" ]]; then
    echo "arm $label: gomaxprocs readback OK: 1, so the env string word-split as intended"
  else
    echo "arm $label: WARNING, gomaxprocs readback is '${got:-<absent>}' and every arm requires 1."
    echo "  The env string did not word-split, so this arm's treatment did not arrive either and the"
    echo "  analyzer must treat it as unmeasured rather than as a result."
  fi
  got="$(sed -n 's/.* cores=\([0-9,]*\).*/\1/p' <<<"$pinline" | tr ',' '\n' | sort -u | /usr/bin/grep -c . )"
  if [[ "$got" == "$(cores_for "$name")" ]]; then
    echo "arm $label: cores readback OK: $got distinct physical core(s), as the design requires"
  else
    echo "arm $label: WARNING, cores readback is $got and the design requires $(cores_for "$name"). This arm does not measure what it was built to measure and the analyzer must treat it as unmeasured."
  fi
  hostsample "after $label"
}

say "phase hunt: test 3's ten arms, sampler ON -- the phenomenon's own positions preserved"
for a in $HUNT_A; do arm "a$a" "$a" on; done
for a in $HUNT_B; do arm "b$a" "$a" on; done

say "phase control: the sampler's own perturbation, off/on/on/off on identical arms"
# Four arms of ONE config. The palindrome is the point: under monotone drift, the two off arms
# straddle the two on arms, so the drift enters both terms of the contrast with the same sign.
for spec in $CONTROL; do arm "${spec%%:*}" "$CONTROL_ARM" "${spec##*:}"; done

say "done"
date -u +%FT%TZ
