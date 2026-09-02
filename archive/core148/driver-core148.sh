#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #148's decisive test 2: four NAMED cpu masks against the scalar rows, one binary, to
# attribute the 3.36-4.31x collapse test 1 confined to mask width 1. Test 1 could not do
# this: keel_pin_mask derives its mask from a width alone, so its only width-1 answer is
# cpu0, and "one core is not enough" and "cpu0 specifically" were the same arm.
#
# The registered bands, the estimator, the truth table and the out-of-domain cell are in
# archive/core148/predictions-core148.py, committed in the same commit as this file and
# imported by the analyzer rather than restated in it. Read that file first; this one only
# collects the samples it will be read against.
#
# Committed BEFORE the run, like driver-width148.sh and for the same reason: the design is
# in the tree at a hash before any of its data exists. Under archive/ and not scripts/ --
# the apparatus cap is on scripts/, and a one-run evidence driver is not apparatus anyone
# else calls. The +113 harness lines this driver depends on are EXEMPT under the
# decisive-discriminator precedent, with their ledger row stating so.
#
# THE ARMS, and what each one is for:
#
#   ref  0,1        two DISTINCT physical cores. The same mask keel_pin_mask derives for
#                   width 2 on a single-LLC host, so this is test 1's w2 arm reached by the
#                   explicit path: the denominator for every ratio AND a harness control.
#   c0   0          the positive control. Every branch predicts this collapses; if it does
#                   not, the explicit path failed to reproduce test 1's width-1 arm and no
#                   other arm in the run is interpretable.
#   c5   5          one core, and NOT cpu0. Separates "cpu0 specifically" from the two
#                   candidates that do not care which core it is.
#   smt  0,<SIB>    two logical cpus on ONE physical core. Gives the Go runtime's own
#                   threads somewhere else to run WITHOUT giving the benchmark a second
#                   core of throughput, which is the only way to separate those two.
#
# SIB is read off the host below and never written here. A hard-coded sibling id would be a
# recalled fact about a machine, and this whole test exists because a mask derived from a
# width could not be interrogated -- replacing that with a mask typed from memory would
# reproduce the defect in a new place.
#
# TWO PASSES, THE SECOND IN REVERSED ARM ORDER, carried over from driver-width148.sh where
# it turned "no time drift" into a measurement that read 1.000-1.003. The verdict here is a
# pattern across four arms; a single time-ordered pass cannot distinguish that pattern from
# anything else monotone in the clock.
#
# ALL 20 ROWS, NOT THE REGISTERED 4. The criterion is evaluated on the four small-kc scalar
# rows and nothing else. The other 16 are a CONTROL, declared as one in the predictions
# module: they read 0.947-1.056 across every arm of test 1, so if they move here the finding
# is a property of the host or the window rather than of affinity.
#
# Host samples are taken BEFORE and AFTER each arm and never during one. Test 1's samples
# are what let its finding survive a co-tenant on the record's own evidence instead of by
# re-running, and the same samples are what indicted `ps -o pcpu` (a lifetime average
# watching a transient) in favour of /proc/loadavg's runnable field for #81.
#
# No `set -e`: every step reports its own rc rather than vanishing, and a killed run must
# not be able to look like a verdict.
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
# arm silently runs four. Found by driver-width148.sh's smoke run.
ARMS_A="${ARMS_A-ref c0 c5 smt}"
ARMS_B="${ARMS_B-smt c5 c0 ref}"
# The non-zero core c5 names. Overridable for a host where cpu5 is cpu0's sibling or absent;
# the preflight refuses in that case rather than substituting, so this never changes silently.
OTHER="${OTHER:-5}"
TAG="${TAG:-}"
OUT="build"

say "provenance and the frozen-tree guard"
date -u +%FT%TZ
echo "settings: HOST=$HOST FILTER=$FILTER COUNT=$COUNT BTIME=$BTIME OTHER=$OTHER"
echo "settings: ARMS_A=[$ARMS_A] ARMS_B=[$ARMS_B] TAG=${TAG:-<none>}"
echo "driver host: $(hostname)"
echo "rev: $FULLREV"
# The tmux server this may run under inherits an environment, and a stale host list in it
# once turned a healthy run RED with zero FAILs. KEEL_PIN_CPUS is in this grep deliberately:
# an inherited one would silently override every arm's mask with a single value, which is the
# one contamination that would leave four identical arms looking like a real null result.
echo "inherited KEEL_*/HOST env: $(env | /usr/bin/grep -E '^(KEEL|HOST)=' | tr '\n' ' ' || true)"
if [[ -n "${KEEL_PIN_CPUS:-}" ]]; then
  echo "REFUSED: KEEL_PIN_CPUS=$KEEL_PIN_CPUS is set in the inherited environment. Every arm of"
  echo "  this run sets it per-arm; an inherited value would give four arms one mask and the"
  echo "  null result would look real. Unset it and relaunch."
  exit 4
fi
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

say "topology preflight: the arms are DERIVED from the host, not asserted about it"
# Everything the four arms assume, read from the far side's own sysfs. Printed as key=value
# so a later reader sees the topology this run was built on and not just the ids it chose.
topo="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" '
  T=/sys/devices/system/cpu
  for c in 0 1 '"$OTHER"'; do
    s=$(cat $T/cpu$c/topology/thread_siblings_list 2>/dev/null)
    echo "siblings$c=$s"
  done
  echo "online=$(cat $T/online 2>/dev/null)"' 2>&1)"
rc=$?
sed 's/^/   /' <<<"$topo"
[[ "$rc" -eq 0 ]] || { echo "REFUSED: could not read topology from $HOST (rc=$rc)"; exit 6; }

# first_sib LIST -- the lowest cpu in a siblings list, which is what keel_pin_explicit uses
# as a physical core's identity. Handles both spellings sysfs uses: "0,36" and "0-1".
first_sib() { local v="${1%%,*}"; printf '%s' "${v%%-*}"; }
# other_sib LIST -- the sibling that is NOT the first one, for a two-way SMT pair.
other_sib() {
  local v="$1"
  case "$v" in
    *,*) printf '%s' "${v#*,}" ;;
    *-*) printf '%s' "${v#*-}" ;;
    *)   printf '' ;;
  esac
}
s0="$(sed -n 's/^siblings0=//p' <<<"$topo")"
s1="$(sed -n 's/^siblings1=//p' <<<"$topo")"
sO="$(sed -n "s/^siblings$OTHER=//p" <<<"$topo")"
SIB="$(other_sib "$s0")"
c0core="$(first_sib "$s0")"; c1core="$(first_sib "$s1")"; cOcore="$(first_sib "$sO")"
echo "derived: core(0)=$c0core core(1)=$c1core core($OTHER)=$cOcore sibling(0)=${SIB:-<none>}"

# Four refusals, each naming the arm it would have invalidated. Refusing is right rather
# than substituting: an explicitly named arm taken on a different mask is the failure this
# test was built to end.
fail=0
[[ -n "$s0" && -n "$s1" && -n "$sO" ]] || { echo "REFUSED: sysfs did not report siblings for cpu0, cpu1 or cpu$OTHER"; fail=1; }
[[ -n "$SIB" ]] || { echo "REFUSED: cpu0 reports no thread sibling ($s0), so the smt arm cannot be built. SMT is likely off on $HOST."; fail=1; }
[[ "$c0core" != "$c1core" ]] || { echo "REFUSED: cpu0 and cpu1 are siblings of ONE core ($s0), so the ref arm would not be two cores and every ratio in this run would have the wrong denominator."; fail=1; }
[[ "$cOcore" != "$c0core" ]] || { echo "REFUSED: cpu$OTHER shares a physical core with cpu0 ($sO), so the c5 arm would retest cpu0's core and could not separate branch A. Set OTHER to a cpu on another core."; fail=1; }
[[ "$fail" -eq 0 ]] || exit 7
echo "preflight: OK -- ref=0,1 is two cores; c$OTHER is a third core; smt=0,$SIB is one core, two cpus"

# The four masks, now that every id in them has been proven against the host.
mask_for() {
  case "$1" in
    ref) printf '0,1' ;;
    c0)  printf '0' ;;
    c5)  printf '%s' "$OTHER" ;;
    smt) printf '0,%s' "$SIB" ;;
    *)   printf '' ;;
  esac
}
# What keel_pin_explicit must report back for each arm: distinct physical cores. This is the
# readback assertion, checked per arm against the log rather than assumed from the request.
cores_for() {
  case "$1" in
    ref) printf '2' ;;
    c0)  printf '1' ;;
    c5)  printf '1' ;;
    smt) printf '1' ;;
    *)   printf '' ;;
  esac
}

say "build: ONE binary, used by ALL arms"
# One binary across every arm is the point: the arms differ only in the affinity mask, so a
# layout difference cannot be mistaken for an affinity effect (#141).
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN"
rc=$?
echo "remote_build_test rc=$rc"
[[ "$rc" -eq 0 ]] || { echo "no binary, no arms"; exit 5; }
sha="$({ shasum -a 256 "$BIN" 2>/dev/null || sha256sum "$BIN"; } | cut -c1-16)"
echo "binary: sha256=${sha} bytes=$(wc -c <"$BIN" | tr -d ' ') flags=[$(build_settings "$BIN")]"
echo "toolchain read off the artifact: $(builder_toolchain "$BIN")"
# #147's arms used sha256=d0d46d26c15cc8b2 at 537661a, and test 1's arms matched it. Whether
# this rebuild still does is a fact to record, not a requirement: a match makes ref directly
# comparable to test 1's w2 medians, and a mismatch means the harness control's admissible
# level range carries a between-binary layout term (#54/#61: 1.71/0.99/1.32%).
if [[ "$sha" == d0d46d26c15cc8b2 ]]; then
  echo "binary IDENTICAL to #147/test-1's binary: ref is a further draw of test 1's own w2 arm"
else
  echo "binary DIFFERS from d0d46d26c15cc8b2: the REF_ADMISSIBLE_GFLOPS control carries a between-binary layout term"
fi

# hostsample LABEL -- load, top consumers and per-core frequency, from the far side. Called
# only between arms. The cpu range covers every core these four masks can select, plus the
# sibling, which a fixed 0-7 would miss if SIB is high (it is 36 on a 72-thread host).
hostsample() {
  echo "-- host sample: $1 --"
  ssh "${KEEL_SSH_OPTS[@]}" "$HOST" '
    printf "uptime: "; uptime
    printf "loadavg: "; cat /proc/loadavg
    printf "freq_khz:"; for c in 0 1 '"$OTHER $SIB"' 6 7; do printf " cpu%s=%s" "$c" "$(cat /sys/devices/system/cpu/cpu$c/cpufreq/scaling_cur_freq 2>/dev/null || echo NA)"; done; echo
    printf "governor: "; cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo NA
    echo "top by cpu:"; ps -eo pid,pcpu,etimes,comm --sort=-pcpu | head -4 | sed "s/^/  /"
    printf "utc: "; date -u +%FT%TZ' 2>&1 | sed 's/^/   /'
}

arm() {
  local pass="$1" name="$2" label="$1$2" cpus log rc pinline got
  cpus="$(mask_for "$name")"
  log="$OUT/bench-core148-$REV$TAG-$label.txt"
  say "arm $label: pass $pass, KEEL_PIN_CPUS=$cpus"
  hostsample "before $label"

  KEEL_PIN_CPUS="$cpus" \
  KEEL_REMOTE_ENV="GOMAXPROCS=1" \
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
  # The mask READBACK, not the mask requested. GOMAXPROCS is 1 in every arm, so the mask is
  # not recoverable from the row names the way it is in a gate run: this line is the only
  # witness to what arm $label actually ran on.
  pinline="$(/usr/bin/grep -m1 '^keel-pin:' "$log")"
  echo "arm $label: keel-pin line: $pinline"
  echo "arm $label: gomaxprocs line: $(/usr/bin/grep -m1 '^keel-bench-gomaxprocs:' "$log")"
  # Distinct physical cores, counted from the readback's own cores= field. This is the
  # assertion the smt arm lives or dies by: 0,$SIB must report ONE core and ref must report
  # two, and neither claim is taken from this driver's request.
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
