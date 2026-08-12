#!/usr/bin/env bash
# Retention: where the blocked Sgemm's time goes that its microkernel's does not.
#
# The measurement for issue #26 — the blocked nest keeps ~90% of its own dispatched
# microkernel's throughput on both Zen hosts and ~77% on janus (Skylake-X) — and
# nothing else. This is NOT a gate: it certifies nothing, it changes no criterion,
# and it exits 0 whatever it measures. Its product is a table.
#
# Two modes:
#
#   decompose (default)  BenchmarkNest on every host: the blocked Sgemm split into
#                        nest-no-pack + pack-a + pack-b, with the residual, and the
#                        microkernel measured in the same invocation so that
#                        retention is a bounded ratio rather than a quotient of two
#                        point estimates from two runs (which is what gate-p3
#                        prints, with that caveat attached).
#   sweep                BenchmarkBlocking: KC/MC/NC over a coarse grid at 2048^3.
#
# The two modes have deliberately different methodologies and say so in their
# output. decompose runs the standard gate methodology (scripts/bench.sh:
# -count=10 -benchtime=1s, benchstat medians, ratios net of CI) because its numbers
# are meant to be quoted. sweep runs -count=5 by default over 72 points per shape
# and is EXPLORATORY: it exists to find which way the surface tilts, and any point
# it nominates has to be re-measured under the full methodology before it becomes a
# default. A 72-point grid at count=10 is half an hour per host, and the first
# question is only whether the parameters matter at all.
#
# GOMAXPROCS=1 on both, as every single-thread measurement in this repo does: P5's
# parallel nest does not exist yet and a benchmark that used all cores would be
# measuring the runtime's scheduler.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

MODE="${1:-decompose}"
case "$MODE" in
  decompose|sweep) ;;
  *) echo "usage: $0 [decompose|sweep]" >&2; exit 2 ;;
esac

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

BINDIR="$(mktemp -d)"
trap 'rm -rf "$BINDIR"' EXIT
BIN="$BINDIR/block.test"
LOG="$BINDIR/log"
CSV="$BINDIR/csv"

echo "== retention ($MODE) — issue #26. Not a gate: this certifies nothing. =="
echo

HOSTS="$(remote_hosts)"
if [[ -z "$HOSTS" ]]; then
  echo "no execution hosts configured (.keel-hosts or \$KEEL_REMOTE_HOSTS)." >&2
  echo "simd/archsimd is amd64-only (T1), so there is nothing to measure here." >&2
  exit 2
fi

if ! remote_build_test ./internal/block/ "$BIN" >"$LOG" 2>&1; then
  echo "cross-compile of the internal/block test binary failed:" >&2
  sed 's/^/  /' "$LOG" >&2
  exit 2
fi
echo "built linux/amd64 internal/block test binary (static)"

if [[ "$MODE" == sweep ]]; then
  KEEL_BENCH_COUNT="${KEEL_BENCH_COUNT:-5}"
  # shellcheck disable=SC2034  # read by bench_expect in the sourced scripts/bench.sh
  KEEL_BENCH_MIN_ROWS="$KEEL_BENCH_COUNT"
fi
mapfile -t BFLAGS < <(bench_flags)
FILTER='BenchmarkNest'
[[ "$MODE" == sweep ]] && FILTER='BenchmarkBlocking'

echo "methodology: -count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, GOMAXPROCS=1, benchstat medians"
[[ "$MODE" == sweep ]] && echo "             EXPLORATORY: re-measure any nominated point at -count=10 before it becomes a default"
echo

# pct FRACTION — a fraction as a percentage, or a dash when it was not measured.
pct() { [[ -n "$1" ]] && awk -v r="$1" 'BEGIN{printf "%.1f%%", r*100}' || printf -- '-'; }

# decompose_host HOST CSV LOG — one host's table.
#
# Every quantity is read out of one CSV from one invocation, which is the whole
# reason BenchmarkNest measures the microkernel itself: a ratio between two numbers
# in one CSV can be bounded by both their intervals (bench_ratio_lo), and a ratio
# across two invocations cannot be bounded at all (§7 rule 7).
decompose_host() {
  local host="$1" csv="$2" log="$3" case_ kc full kern ret retlo pa pb np resid
  # The sub-benchmark set is discovered from the log rather than listed here: the
  # shapes are whatever internal/kern registers for this host's backend, and a
  # hard-coded list would silently measure fewer shapes than ran.
  while read -r case_; do
    kc="$(awk -v c="$case_" '$0 ~ "^Benchmark"c"/kernel/kc=" { n=$1; sub(/-[0-9]+$/,"",n); sub(/.*kc=/,"",n); print n; exit }' "$log")"
    if [[ -z "$kc" ]]; then
      warn "[$host] $case_: no kernel sub-benchmark ran, so retention has no denominator this run"
      continue
    fi
    if ! bench_expect "$log" "$csv" sec/op \
         "$case_/full" "$case_/nest-no-pack" "$case_/pack-a" "$case_/pack-b" "$case_/kernel/kc=$kc" >"$BINDIR/miss"; then
      warn "[$host] $case_: incomplete —$(tr '\n' ' ' <"$BINDIR/miss")"
      continue
    fi
    full="$(bench_gflops "$case_/full" "$csv")"
    kern="$(bench_gflops "$case_/kernel/kc=$kc" "$csv")"
    # Retention with both intervals honoured: the blocked rate pushed down by its
    # own CI, the kernel rate pushed up by its own. A retention that clears a number
    # here would clear it if both measurements were as wrong as benchstat allows.
    ret="$(bench_ratio "$case_/full" "$case_/kernel/kc=$kc" "$csv" GFLOP/s)"
    retlo="$(bench_ratio_lo "$case_/full" "$case_/kernel/kc=$kc" "$csv" GFLOP/s)"
    # The parts, as fractions of the whole, in seconds — the unit they share.
    np="$(bench_ratio "$case_/nest-no-pack" "$case_/full" "$csv")"
    pa="$(bench_ratio "$case_/pack-a" "$case_/full" "$csv")"
    pb="$(bench_ratio "$case_/pack-b" "$case_/full" "$csv")"
    resid="$(awk -v a="${np:-0}" -v b="${pa:-0}" -v c="${pb:-0}" 'BEGIN{printf "%.4f", 1-a-b-c}')"
    printf '  %s\n' "$case_"
    info "kernel      $(printf '%8.1f' "${kern:-0}") GFLOP/s at kc=$kc   $(bench_describe "$case_/kernel/kc=$kc" "$csv" GFLOP/s)"
    info "blocked     $(printf '%8.1f' "${full:-0}") GFLOP/s            $(bench_describe "$case_/full" "$csv" GFLOP/s)"
    info "retention   $(pct "$ret") of its own microkernel ($(pct "$retlo") net of both CIs)"
    info "  parts of the blocked time: nest-no-pack $(pct "$np")  pack-a $(pct "$pa")  pack-b $(pct "$pb")  residual $(pct "$resid")"
    info "  (residual is a point estimate with no interval — a difference of four"
    info "   medians. It holds what nest-no-pack drops by construction: the"
    info "   pack/kernel cache interference, and gemm's three per-call allocations."
    info "   See internal/block/nest_bench_test.go.)"
  done < <(awk '/^Benchmark.*\/full/ { n=$1; sub(/-[0-9]+$/,"",n); sub(/^Benchmark/,"",n); sub(/\/full$/,"",n); print n }' "$log" | sort -u)
}

# sweep_host HOST CSV LOG — the KC/MC/NC grid, ranked, with the shipped point marked.
sweep_host() {
  local host="$1" csv="$2" log="$3" shipped
  shipped="$(awk '/^[[:space:]]*KC = / {kc=$3} /^[[:space:]]*MC = / {mc=$3} /^[[:space:]]*NC = / {nc=$3} END{print "kc="kc"/mc="mc"/nc="nc}' internal/block/block.go)"
  info "[$host] shipped point: $shipped (internal/block/block.go). NC>=n collapses to one point at n=2048."
  awk -F, '
    /^,/ { unit = $2; next }
    unit == "GFLOP/s" {
      name = $1; sub(/-[0-9]+$/, "", name); sub(/^BenchmarkBlocking\//, "", name)
      print $2, name
    }' "$csv" | sort -rn | awk -v s="$shipped" '
      { mark = (index($2, s) > 0) ? "  <- shipped" : ""
        printf "        %2d. %8.1f GFLOP/s  %s%s\n", ++i, $1, $2, mark }'
}

while read -r host; do
  [[ -n "$host" ]] || continue
  prov="$(remote_probe "$host" || true)"
  if [[ -z "$prov" ]]; then
    warn "[$host] unreachable, or /proc/cpuinfo unreadable — skipped, and its row is missing rather than estimated"
    continue
  fi
  echo "-- $host --"
  info "$prov"
  if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BIN" "${BFLAGS[@]}" \
       -test.bench="$FILTER" >"$LOG" 2>&1; then
    warn "[$host] the benchmark run failed; nothing is reported for it"
    sed 's/^/        /' "$LOG" | tail -20
    continue
  fi
  # The plan markers first: they say which blocks the parts were measured over.
  grep '^keel-nest-plan:' "$LOG" | sed 's/^/        /' || true
  bench_csv "$LOG" >"$CSV" 2>"$BINDIR/bserr" || true
  [[ -s "$BINDIR/bserr" ]] && sed 's/^/        benchstat: /' "$BINDIR/bserr"
  if [[ "$MODE" == decompose ]]; then
    decompose_host "$host" "$CSV" "$LOG"
  else
    sweep_host "$host" "$CSV" "$LOG"
  fi
  echo
done <<<"$HOSTS"

echo "done. Numbers above are this host set at this commit; nothing here is a criterion."
