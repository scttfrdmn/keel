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
#  4. BENCH NOISE. DESIGN.md §5.4 specifies `-bench=Gate -benchtime=3x`. Three
#     iterations of a few-microsecond kernel is a very small sample, so the
#     gate takes `-count=5` samples and uses the *minimum* ns/op per backend —
#     the standard way to suppress scheduler noise without changing benchtime.
#     Every raw sample is printed so the spread is visible rather than implied.
#     Every host that exercised avx512 must clear 4x; the ratio is
#     within-machine, so a throttled or busy host has no excuse.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh

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
echo "-- Sdot n=4096: avx512 vs scalar (DESIGN.md §5.4 CI-mode) --"
if [[ -z "$HOSTS" ]]; then
  fail "the 4x criterion needs an amd64 host; none configured"
else
  # min ns/op for one backend's sub-benchmark across all -count samples.
  read_min() {
    awk -v be="$1" '
      $1 ~ ("^BenchmarkGateSdot/" be "(-[0-9]+)?$") {
        for (i = 2; i <= NF; i++)
          if ($i == "ns/op") { v = $(i-1) + 0; if (m == 0 || v < m) m = v }
      }
      END { if (m > 0) printf "%.1f", m }' "$2"
  }
  while read -r host; do
    [[ -n "$host" ]] || continue
    if ! remote_exec "$host" "$BIN" \
         -test.run=NONE -test.bench='GateSdot' -test.benchtime=3x -test.count=5 \
         >"$LOG" 2>&1; then
      fail "[$host] benchmark run failed"
      sed 's/^/        /' "$LOG" | tail -20
      continue
    fi
    s="$(read_min scalar "$LOG")"
    a="$(read_min avx512 "$LOG")"
    raw="$(grep -oE 'BenchmarkGateSdot/[a-z0-9]+(-[0-9]+)?[[:space:]]+[0-9]+[[:space:]]+[0-9.]+ ns/op' "$LOG" | tr -s ' ' | paste -sd'; ' -)"
    if [[ -z "$s" || -z "$a" ]]; then
      info "[$host] samples: ${raw:-none}"
      if grep -q 'avx512' "$LOG"; then
        fail "[$host] could not parse both scalar and avx512 ns/op"
      else
        info "[$host] no avx512 sub-benchmark (CPU lacks it); not counted"
      fi
      continue
    fi
    ratio="$(awk -v s="$s" -v a="$a" 'BEGIN{printf "%.2f", s/a}')"
    info "[$host] scalar $s ns/op, avx512 $a ns/op"
    info "[$host] samples: $raw"
    if awk -v r="$ratio" 'BEGIN{exit !(r >= 4.0)}'; then
      pass "[$host] Sdot n=4096 speedup ${ratio}x (>= 4x)"
    else
      fail "[$host] Sdot n=4096 speedup ${ratio}x (< 4x required)"
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
