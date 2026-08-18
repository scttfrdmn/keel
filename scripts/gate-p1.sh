#!/usr/bin/env bash
# Gate P1 — see DESIGN.md §4/P1. Exits 0 only when every criterion holds.
# A red gate blocks P2; there is no override flag on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P1):
#   "all L1 tests green on avx512 + scalar; benchmarks show >=4x scalar for
#    Sdot at n=4096."
#
# How those are mechanized, and the judgement calls involved — all printed in
# the gate output rather than hidden:
#
#  1. EXECUTION TARGETS. Same structure as gate-p0: the dev host is
#     darwin/arm64 and cannot run a vector op, so "green on avx512" is a claim
#     about another machine. Each host in .keel-hosts is a target, reached by
#     shipping a cross-compiled static test binary (scripts/remote.sh). The
#     local host still runs, because it is what proves the scalar path holds on
#     a stock GOARCH.
#
#  2. BOTH BACKENDS, TWO WAYS. DESIGN.md wants avx512 *and* scalar green. The
#     differential tests exercise every compiled-in backend in one run and
#     report which (keel-l1-backends-exercised), so the gate can insist that
#     nothing available was skipped. Separately each remote target re-runs the
#     whole suite under KEEL_FORCE=scalar, which proves the dispatch override
#     works on a machine that *has* AVX-512 — a scalar pass on arm64, where
#     there was never another option, would not prove that.
#
#  3. THE 4x BASELINE IS NOT A STRAW MAN. The scalar Sdot it is measured
#     against uses four independent accumulators, i.e. what a competent Go
#     programmer would write, not a naive single-accumulator loop that would
#     inflate the ratio for free. This makes the gate harder to pass and the
#     resulting claim worth making.
#
#  4. BENCH NOISE. `-count=10 -benchtime=1s`, aggregated by benchstat, and the
#     4x bar counts as cleared only if the median clears it *net of both
#     confidence intervals* (decision on issue #14; mechanics and rationale in
#     scripts/bench.sh). The first P1 numbers came from `-benchtime=3x -count=5`
#     reduced by min-of-samples, whose raw samples spread by 3.08x within one
#     backend; those are superseded here rather than kept for comparison,
#     because comparing across measurement regimes is how a perf project loses
#     track of what it knows. Every host that exercised avx512 must clear the
#     bar, and EVERY MEASURING HOST must have its clock established stable,
#     asserted per host in the preamble and again at the moment of measurement
#     (#77) — by the performance governor where `cpufreq` is readable, which is
#     what this gate asserts and what every host it has run against has. The older form of this sentence said "at least one of them" — a
#     criterion any single host satisfied on the others' behalf, under which two
#     archived green runs published a rate taken on a `powersave` host (#79).
#     The ratio is within-machine, so a throttled host has no excuse; it also
#     does not get to contribute a rate under a governor nobody asserted.
#
#  5. THE PEAK DENOMINATOR IS MEASURED HERE TOO, and printed. P1 has no
#     percent-of-peak criterion, so this is provenance rather than a gate
#     criterion — but P2's 55% floor divides by this number, and recording it
#     alongside P1's ratios means the two phases' numbers share a regime from
#     the start (issue #11). BenchmarkPeak's witness check runs on real silicon
#     as a side effect, which is the only place a collapsed accumulator chain
#     could still hide.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

echo "== gate-p1: Level 1 + test harness =="
echo

# ------------------------------------------------------------- tree state (#63)
assert_no_strays

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi

# ----------------------------------------------------------------- L1 tests
echo
echo "-- L1 tests (oracle, cross-backend differential, properties) --"
gate_tmpdir

AVX512_GREEN=""
AVX512_SEEN=0

# score_run NAME LOG OK — check one test run passed, and that it left no
# compiled-in backend unexercised.
score_run() {
  local name="$1" log="$2" ok="$3" cover avail missing="" want
  test_verdict "$name" "$log" "$ok" "tests pass"

  cover="$(grep -o 'keel-l1-backends-exercised:.*' "$log" | tail -1 || true)"
  avail="$(grep -o 'keel-l1-available:.*' "$log" | tail -1 || true)"
  # Coverage state for the avx512 aggregate below (#73's tier C): count the runs
  # that reported the backend *available*, which is the discriminator between a
  # fleet that could look and came back short and a fleet with nothing to ask.
  # Counted before the coverage-marker return, because a host whose availability
  # marker reads avx512 and whose coverage marker is missing still establishes
  # that the capability was present.
  if grep -qE "keel-l1-available:.*(^| )avx512( |$)" <<<"$avail"; then
    AVX512_SEEN=$((AVX512_SEEN + 1))
  fi
  if [[ -z "$cover" ]]; then
    unmeasured "[$name] no L1 backend-coverage marker in test output, so what this run exercised cannot be read"
    return
  fi
  info "[$name] $cover"
  [[ -n "$avail" ]] && info "[$name] $avail"
  for want in $(sed 's/.*keel-l1-available: *//' <<<"$avail"); do
    grep -qE "keel-l1-backends-exercised:.*(^| )$want( |$)" <<<"$cover" || missing="$missing $want"
  done
  if [[ -n "$missing" ]]; then
    fail "[$name] available backends were not exercised:$missing"
  else
    pass "[$name] every available L1 backend was exercised"
  fi
  if [[ "$ok" -eq 0 ]] && grep -qE "keel-l1-backends-exercised:.*(^| )avx512( |$)" <<<"$cover"; then
    AVX512_GREEN="$name"
  fi
}

LOCAL_OK=0
GOEXPERIMENT=simd go test -v -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
score_run "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$LOG" "$LOCAL_OK"

HOSTS="$(remote_hosts)"
# The ledger of what this gate trusts rather than checks (#73 tier C, ruled
# 2026-08-15). Declared here, where the fleet is named; printed beside the
# verdict by assumed_ledger below.
assume_fleet "$HOSTS"
require_disk

# ---- the measurement precondition, asserted rather than assumed (#77)
#
# DESIGN.md §5 rule 5 as amended: EVERY measuring host's clock established stable,
# per host, never satisfied by one host on behalf of another. The rule names the
# instrument by what the host has; where `cpufreq` is readable — which is every host
# this gate has ever run against — that instrument is the `performance` governor, so
# what the code below asserts is the rule as it applies here, not the whole rule. A
# guest, where there is no `cpufreq` directory at all, takes rule 5's other branch
# (`BenchmarkPeak` sampled head/middle/tail) and is not yet implemented in this
# harness; it blocks at remote.sh's assert_governor as `unmeasured` rather than
# greening on an unasserted precondition.
# This gate used to read the governor, print it as `info`, and assert only that
# *some* host had cleared the 4x bar under `performance` — a criterion satisfied
# by any single host, which therefore said nothing about the others. It is not a
# hypothetical hole: both archived green gate-p1 logs on #2 have the Zen 5 host
# on `powersave`, and one of their rates is published (#79). The check itself now
# lives in remote.sh's assert_governor — it used to be a copy of gate-p4's, and
# four such copies shared a mislabel none of them could reveal (#83).
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    assert_governor "$host" preamble
  done <<<"$HOSTS"
fi

N_CONF=0
N_SCORED=0
if [[ -z "$HOSTS" ]]; then
  info "no remote targets configured (.keel-hosts or \$KEEL_REMOTE_HOSTS)"
else
  remote_build_test_or_fail ./ "$BIN" "$LOG" \
    "cross-compiled linux/amd64 test binary" \
    "cross-compile of linux/amd64 test binary"
  while read -r host; do
    [[ -n "$host" ]] || continue
    N_CONF=$((N_CONF + 1))
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      unmeasured "[$host] unreachable, so this target produced no reading"
      continue
    fi
    info "[$host] $prov"

    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    score_run "$host" "$LOG" "$OK"

    # Same suite with dispatch forced to scalar, on a machine that has
    # AVX-512 to fall back *from*.
    OK=0
    KEEL_REMOTE_ENV="KEEL_FORCE=scalar" remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] KEEL_FORCE=scalar: tests pass with dispatch overridden"
      info "[$host] $(grep -o 'keel-l1-active:.*' "$LOG" | tail -1 || echo 'no active marker')"
    elif remote_vanished; then
      unmeasured "[$host] KEEL_FORCE=scalar: the forced run did not finish (#62), so the overridden dispatch is unmeasured here rather than broken"
    else
      fail "[$host] KEEL_FORCE=scalar: tests pass with dispatch overridden"
      sed 's/^/        /' "$LOG" | tail -30
    fi
    N_SCORED=$((N_SCORED + 1))
  done <<<"$HOSTS"
  if [[ "$N_SCORED" -eq "$N_CONF" ]]; then
    pass "every configured remote target ran ($N_SCORED/$N_CONF)"
  else
    fail "only $N_SCORED of $N_CONF configured remote targets ran"
  fi
fi

if [[ -n "$AVX512_GREEN" ]]; then
  pass "L1 tests green with the avx512 backend exercised (target: $AVX512_GREEN)"
elif [[ "$AVX512_SEEN" -gt 0 ]]; then
  fail "no target ran the L1 tests green with the avx512 backend, though $AVX512_SEEN target(s) reported it available"
else
  unmeasured "no target reported an avx512 backend available, so whether the L1 tests pass on it is unmeasured rather than short: there was no host to ask"
fi

# -------------------------------------------------------- Sdot >= 4x scalar
echo
echo "-- Sdot n=4096: avx512 vs scalar (issue #14 methodology) --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, benchstat median, bar cleared net of CI"
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

# Coverage state, not a host name (#77 and #73's tier C). N_RATIO counts hosts
# that produced a bounded ratio — the reading this criterion is about — and
# N_CLEARED how many of those cleared the bar. Two counters because "nobody
# cleared 4x" and "nobody produced a ratio" are different facts and the old
# single-name variable could not tell them apart.
N_RATIO=0
N_CLEARED=0
if [[ -z "$HOSTS" ]]; then
  unmeasured "the 4x criterion needs an amd64 host and none is configured, so it is unmeasured rather than missed"
else
  remote_build_test_or_fail ./bench "$BENCHBIN" "$LOG" \
    "cross-compiled linux/amd64 bench binary" \
    "cross-compile of linux/amd64 bench binary"

  while read -r host; do
    [[ -n "$host" ]] || continue
    # Re-read at the moment of measurement, not merely in the preamble above: a
    # governor that changed in between belongs to a machine somebody started
    # using, and the reading it produces is not one §5 rule 5 covers. Silent on
    # success — the preamble printed the PASS — and the provenance info line is
    # therefore the only record of what was read on a passing host.
    assert_governor "$host" measured
    clock_gate "$host" || continue
    # The one gate §5 rule 5's substitute instrument does not reach, said out loud
    # rather than worked around. It needs a BenchmarkPeak window inside the
    # sweep to be the middle of its series; this gate's sweep runs the ROOT package's
    # test binary, and BenchmarkPeak lives in ./bench. Sampling it from a second
    # binary would put two compilers in a three-point trend, and the 4x criterion is
    # a ratio of two ADJACENT windows — `-count` is the inner loop, so scalar's
    # samples all precede avx512's — which a clock that moved between them shifts
    # rather than widens. So: unmeasured on a host with no governor, not assumed steady.
    if [[ "$GOV_STATE" == nocpufreq ]]; then
      unmeasured "[$host] no governor to assert and no peak window inside this gate's sweep to substitute for one, so the Sdot speedup is unmeasured on this host (§5 rule 5 as amended 2026-08-16)"
      continue
    fi

    if ! remote_exec "$host" "$BIN" "${BFLAGS[@]}" -test.bench='GateSdot' \
         >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the GateSdot benchmark run failed, so this host's speedup is unmeasured rather than short of 4x"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    info "[$host] scalar $(bench_describe GateSdot/scalar "$BENCHCSV")"
    info "[$host] avx512 $(bench_describe GateSdot/avx512 "$BENCHCSV")"

    lo="$(bench_ratio_lo GateSdot/scalar GateSdot/avx512 "$BENCHCSV")"
    pt="$(bench_ratio GateSdot/scalar GateSdot/avx512 "$BENCHCSV")"
    if [[ -z "$lo" ]]; then
      if [[ -n "$(bench_stat GateSdot/avx512 "$BENCHCSV")" ]]; then
        # A median with no interval is not a measurement this gate can use.
        unmeasured "[$host] no bounded ratio: benchstat established no confidence interval, which is a failure to measure rather than a pass"
      else
        info "[$host] no avx512 sub-benchmark (CPU lacks it); not counted"
      fi
      continue
    fi
    N_RATIO=$((N_RATIO + 1))
    if awk -v r="$lo" 'BEGIN{exit !(r >= 4.0)}'; then
      pass "[$host] Sdot n=4096 speedup ${pt}x, ${lo}x net of CI (>= 4x)"
      N_CLEARED=$((N_CLEARED + 1))
    else
      fail "[$host] Sdot n=4096 speedup ${pt}x, only ${lo}x net of CI (< 4x required)"
    fi
  done <<<"$HOSTS"

  # The aggregate, three-way over coverage state (#73's tier C). Every host that
  # got this far is under `performance`, asserted twice above, so this line no
  # longer names one host on the fleet's behalf. The third branch is the one the
  # old form could not express: a fleet whose CPUs all lack AVX-512 reaches the
  # `no avx512 sub-benchmark` info above, produces no ratio at all, and used to
  # leave this criterion with no verdict line whatsoever.
  if [[ "$N_RATIO" -eq 0 ]]; then
    unmeasured "no host produced a bounded Sdot ratio, so the 4x criterion is unmeasured rather than short: #14 requires one host to clear it and none looked"
  elif [[ "$N_CLEARED" -eq "$N_RATIO" ]]; then
    pass "every host that produced a bounded ratio cleared 4x ($N_CLEARED/$N_RATIO), each under the performance governor asserted per host (#14 requires one)"
  else
    fail "$((N_RATIO - N_CLEARED)) of $N_RATIO hosts that produced a bounded ratio did not clear 4x (the per-host lines above say which)"
  fi
fi

# ------------------------------------- measured FMA peak (provenance for P2)
echo
echo "-- measured FMA peak per host (issue #11; provenance, not a P1 criterion) --"
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench='Peak' \
         >"$BENCHLOG" 2>&1; then
      # Not a P1 criterion, but a failure here is either a collapsed accumulator
      # chain or an unmeasurable host, and both must stop P2 from starting.
      unmeasured "[$host] BenchmarkPeak failed, so P2's denominator is unmeasured here"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    pass "[$host] BenchmarkPeak ran; accumulator-chain witness held on real silicon"
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>/dev/null || true
    g512=""; g256=""
    for be in avx512 avx2 scalar; do
      g="$(bench_gflops "Peak/$be" "$BENCHCSV")"
      [[ -n "$g" ]] || continue
      [[ "$be" == "avx512" ]] && g512="$g"
      [[ "$be" == "avx2" ]] && g256="$g"
      f="$(sed -n "s/.*keel-bench-peak-formula: $be: \([0-9.]*\) GFLOP\/s.*/\1/p" "$BENCHLOG" | head -1)"
      if [[ -n "$f" ]]; then
        # The formula assumes two full-width FMA ports running at max clock.
        # Divergence means one of those assumptions does not hold here — a
        # double-pumped datapath, an AVX-512 frequency license, or a latency
        # coupling the formula does not model. It is reported, never corrected
        # for: the measured number is the denominator either way.
        info "[$host] peak $be: measured $(awk -v g="$g" 'BEGIN{printf "%.1f", g}') GFLOP/s, formula $f — $(
          awk -v g="$g" -v f="$f" 'BEGIN{
            d = f / g
            if (d >= 1) printf "formula %.2fx high", d
            else printf "formula %.2fx low", 1 / d
            if (d >= 1.5) printf " (>=1.5x: a formula assumption overstates this host)"
            else if (d <= 0.667) printf " (>=1.5x: the formula understates this host, so its port count is wrong here too)"
          }')"
      else
        info "[$host] peak $be: measured $(awk -v g="$g" 'BEGIN{printf "%.1f", g}') GFLOP/s, no formula cross-check available"
      fi
    done
    # The width ratio is a direct measurement of the 512-bit datapath, and a far
    # better diagnostic than measured-vs-formula: it needs no clock, no port
    # count and no assumption about the frequency license, because both halves
    # were measured on this host under identical conditions.
    if [[ -n "$g512" && -n "$g256" ]]; then
      info "[$host] peak width ratio avx512/avx2 = $(awk -v a="$g512" -v b="$g256" 'BEGIN{
        r = a / b
        printf "%.2fx", r
        if (r >= 1.75) printf " (~2x: a true full-width 512-bit datapath)"
        else if (r <= 1.25) printf " (~1x: AVX-512 double-pumped over a 256-bit datapath)"
        else printf " (between the two shapes; treat with suspicion)"
      }')"
    fi
  done <<<"$HOSTS"
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
gate_verdict gate-p1
