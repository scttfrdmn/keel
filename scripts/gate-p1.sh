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
#     bar, and at least one of them must have done so under the performance
#     governor: the ratio is within-machine, so a throttled host has no excuse,
#     but nor does it get to be the only evidence.
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
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

FAIL=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=1; }
info() { printf '        %s\n' "$1"; }

echo "== gate-p1: Level 1 + test harness =="
echo

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi

# ----------------------------------------------------------------- L1 tests
echo
echo "-- L1 tests (oracle, cross-backend differential, properties) --"
LOG="$(mktemp)"
BINDIR="$(mktemp -d)"
BIN="$BINDIR/keel.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
trap 'rm -rf "$LOG" "$BINDIR"' EXIT

AVX512_GREEN=""

# score_run NAME LOG OK — check one test run passed, and that it left no
# compiled-in backend unexercised.
score_run() {
  local name="$1" log="$2" ok="$3" cover avail missing="" want
  if [[ "$ok" -eq 0 ]]; then
    pass "[$name] tests pass"
  else
    fail "[$name] tests pass"
    sed 's/^/        /' "$log" | tail -40
  fi

  cover="$(grep -o 'keel-l1-backends-exercised:.*' "$log" | tail -1 || true)"
  avail="$(grep -o 'keel-l1-available:.*' "$log" | tail -1 || true)"
  if [[ -z "$cover" ]]; then
    fail "[$name] no L1 backend-coverage marker in test output"
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
N_CONF=0
N_SCORED=0
if [[ -z "$HOSTS" ]]; then
  info "no remote targets configured (.keel-hosts or \$KEEL_REMOTE_HOSTS)"
else
  if remote_build_test ./ "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary"
  else
    fail "cross-compile of linux/amd64 test binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    N_CONF=$((N_CONF + 1))
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      fail "[$host] unreachable"
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
else
  fail "no target ran the L1 tests green with the avx512 backend"
fi

# -------------------------------------------------------- Sdot >= 4x scalar
echo
echo "-- Sdot n=4096: avx512 vs scalar (issue #14 methodology) --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, benchstat median, bar cleared net of CI"
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

PERF_GOV_HOST=""
if [[ -z "$HOSTS" ]]; then
  fail "the 4x criterion needs an amd64 host; none configured"
else
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary"
  else
    fail "cross-compile of linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi

  while read -r host; do
    [[ -n "$host" ]] || continue
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    info "[$host] governor=${gov:-unknown}"

    if ! remote_exec "$host" "$BIN" "${BFLAGS[@]}" -test.bench='GateSdot' \
         >"$BENCHLOG" 2>&1; then
      fail "[$host] benchmark run failed"
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
        fail "[$host] no bounded ratio: benchstat established no confidence interval"
      else
        info "[$host] no avx512 sub-benchmark (CPU lacks it); not counted"
      fi
      continue
    fi
    if awk -v r="$lo" 'BEGIN{exit !(r >= 4.0)}'; then
      pass "[$host] Sdot n=4096 speedup ${pt}x, ${lo}x net of CI (>= 4x)"
      [[ "$gov" == "performance" ]] && PERF_GOV_HOST="$host"
    else
      fail "[$host] Sdot n=4096 speedup ${pt}x, only ${lo}x net of CI (< 4x required)"
    fi
  done <<<"$HOSTS"

  if [[ -n "$PERF_GOV_HOST" ]]; then
    pass "the 4x bar was cleared under the performance governor ($PERF_GOV_HOST)"
  else
    fail "no host cleared 4x under the performance governor (issue #14 requires one)"
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
      fail "[$host] BenchmarkPeak failed; P2's denominator cannot be measured here"
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

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p1: GREEN"
  exit 0
fi
echo "gate-p1: RED" >&2
exit 1
