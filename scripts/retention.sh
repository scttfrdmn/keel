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
#
#                        Three denominators are printed, not one. The historical
#                        one (a single call at a single depth into a single C tile,
#                        which is what bench/BenchmarkKernel measures and what #26's
#                        "~77%" divided by) turned out not to be a ceiling: the
#                        repeated calls share one C tile, a dependency the real nest
#                        does not have. So the table shows that measurement, the same
#                        measurement with the dependency broken, and the nest's own
#                        call multiset at its own depths — which is the one retention
#                        is quoted against. The superseded number stays visible
#                        because three gates reported it.
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

# Everything below is a function definition and the last line of the file is
# `main "$@"`. Bash reads a script incrementally as it executes it, so a script
# that does its work at the top level can be corrupted by an edit that lands
# mid-run: the parser resumes at a byte offset that now holds different text.
# Defining everything before anything runs forces one whole-file parse before the
# first host is touched, which makes the instrument immune instead of leaving
# "never edit a running instrument" as a rule someone has to remember.

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

# pct FRACTION — a fraction as a percentage, or a dash when it was not measured.
pct() { [[ -n "$1" ]] && awk -v r="$1" 'BEGIN{printf "%.1f%%", r*100}' || printf -- '-'; }

# rows_per_bench LOG — how many sample rows this run actually produced per
# benchmark, counted out of the log.
#
# WHY THIS EXISTS. The methodology line used to print $KEEL_BENCH_COUNT, i.e. the
# variable the run intended to use — and issue #49 was that variable being
# silently overwritten: scripts/bench.sh is sourced first and defaults it to 10,
# so this script's own `${KEEL_BENCH_COUNT:-5}` could never see either the
# caller's setting or its own documented default, and the header printed a
# methodology that was not the one that ran. A parameter read back out of the
# measurement cannot be shadowed by whatever set it, so the header now prints
# both: what was asked for, and what the log says arrived.
rows_per_bench() {
  awk '
    /^Benchmark/ { n = $1; sub(/-[0-9]+$/, "", n); c[n]++ }
    END {
      for (k in c) { if (min == "" || c[k] < min) min = c[k]; if (c[k] > max) max = c[k]; nb++ }
      if (nb == 0) { printf "no benchmark rows at all"; exit }
      if (min == max) printf "%d rows x %d benchmarks", min, nb
      else printf "%d-%d rows x %d benchmarks — uneven, so some did not finish", min, max, nb
    }' "$1"
}

# decompose_host HOST CSV LOG — one host's table.
#
# Every quantity is read out of one CSV from one invocation, which is the whole
# reason BenchmarkNest measures the microkernel itself: a ratio between two numbers
# in one CSV can be bounded by both their intervals (bench_ratio_lo), and a ratio
# across two invocations cannot be bounded at all (§7 rule 7).
decompose_host() {
  local host="$1" csv="$2" log="$3" case_ kc full ret retlo pa pb np resid
  local d1 d1r d8 d8r calls callsr nprate npret npretlo
  # The sub-benchmark set is discovered from the log rather than listed here: the
  # shapes are whatever internal/kern registers for this host's backend, and a
  # hard-coded list would silently measure fewer shapes than ran.
  while read -r case_; do
    kc="$(awk -v c="$case_" '$0 ~ "^Benchmark"c"/kernel/kc=" { n=$1; sub(/-[0-9]+$/,"",n); sub(/\/ctiles=.*/,"",n); sub(/.*kc=/,"",n); print n; exit }' "$log")"
    if [[ -z "$kc" ]]; then
      warn "[$host] $case_: no kernel sub-benchmark ran, so retention has no denominator this run"
      continue
    fi
    d1="$case_/kernel/kc=$kc/ctiles=1"
    d8="$case_/kernel/kc=$kc/ctiles=8"
    calls="$case_/kernel-calls"
    if ! bench_expect "$log" "$csv" sec/op \
         "$case_/full" "$case_/nest-no-pack" "$case_/pack-a" "$case_/pack-b" \
         "$d1" "$d8" "$calls" >"$BINDIR/miss"; then
      warn "[$host] $case_: incomplete —$(tr '\n' ' ' <"$BINDIR/miss")"
      continue
    fi
    full="$(bench_gflops "$case_/full" "$csv")"
    # The three candidate denominators. d1 is what bench/BenchmarkKernel measures and
    # what #26's original number divided by; d8 is d1 with consecutive calls made
    # independent, which is the manipulation; calls is the nest's own call multiset at
    # its own depths. Only the last is a denominator for retention — see
    # internal/block/nest_bench_test.go — but all three are printed, because the
    # first is the number three earlier gates reported and a revision that hides
    # what it revised is not a revision.
    d1r="$(bench_gflops "$d1" "$csv")"
    d8r="$(bench_gflops "$d8" "$csv")"
    callsr="$(bench_gflops "$calls" "$csv")"
    # The rate the nest would reach if packing were free: measured, not derived from
    # the time fractions, so bench_ratio_lo can bound it.
    nprate="$(bench_gflops "$case_/nest-no-pack" "$csv")"
    # Retention with both intervals honoured: the blocked rate pushed down by its
    # own CI, the denominator pushed up by its own. A retention that clears a number
    # here would clear it if both measurements were as wrong as benchstat allows.
    ret="$(bench_ratio "$case_/full" "$calls" "$csv" GFLOP/s)"
    retlo="$(bench_ratio_lo "$case_/full" "$calls" "$csv" GFLOP/s)"
    npret="$(bench_ratio "$case_/nest-no-pack" "$calls" "$csv" GFLOP/s)"
    npretlo="$(bench_ratio_lo "$case_/nest-no-pack" "$calls" "$csv" GFLOP/s)"
    # The parts, as fractions of the whole, in seconds — the unit they share.
    np="$(bench_ratio "$case_/nest-no-pack" "$case_/full" "$csv")"
    pa="$(bench_ratio "$case_/pack-a" "$case_/full" "$csv")"
    pb="$(bench_ratio "$case_/pack-b" "$case_/full" "$csv")"
    resid="$(awk -v a="${np:-0}" -v b="${pa:-0}" -v c="${pb:-0}" 'BEGIN{printf "%.4f", 1-a-b-c}')"
    printf '  %s\n' "$case_"
    info "blocked        $(printf '%8.1f' "${full:-0}") GFLOP/s   $(bench_describe "$case_/full" "$csv" GFLOP/s)"
    info "no-pack        $(printf '%8.1f' "${nprate:-0}") GFLOP/s   $(bench_describe "$case_/nest-no-pack" "$csv" GFLOP/s)"
    info "denominators, per call at kc=$kc and over the nest's own call multiset:"
    info "  ctiles=1     $(printf '%8.1f' "${d1r:-0}") GFLOP/s   $(bench_describe "$d1" "$csv" GFLOP/s)"
    info "  ctiles=8     $(printf '%8.1f' "${d8r:-0}") GFLOP/s   $(bench_describe "$d8" "$csv" GFLOP/s)"
    info "  kernel-calls $(printf '%8.1f' "${callsr:-0}") GFLOP/s   $(bench_describe "$calls" "$csv" GFLOP/s)"
    info "  ctiles=8/ctiles=1 $(pct "$(bench_ratio "$d8" "$d1" "$csv" GFLOP/s)") ($(pct "$(bench_ratio_lo "$d8" "$d1" "$csv" GFLOP/s)") net of both CIs)"
    info "  — above 100% means the historical one-C-tile measurement is not a ceiling."
    info "  ctiles=* and kernel-calls are NOT comparable to each other: the first two"
    info "  are per-call rates over 2*MR*NR*kc, kernel-calls is a whole-GEMM rate over"
    info "  2mnk. Only the ctiles ratio answers the dependency question, and only"
    info "  kernel-calls is a denominator for retention."
    info "retention   $(pct "$ret") of the kernel calls it makes ($(pct "$retlo") net of both CIs)"
    info "  with packing removed: $(pct "$npret") ($(pct "$npretlo") net of both CIs)"
    # Positive is a cost: points of the denominator's rate that this part gives up.
    # The two terms sum to the total by construction (they are 1-npret and npret-ret).
    info "  the $(awk -v a="${ret:-0}" 'BEGIN{printf "%.1f", (1-a)*100}') points it gives up split into"
    info "    packing + residual  $(awk -v a="${npret:-0}" -v b="${ret:-0}" 'BEGIN{printf "%+.1f", (a-b)*100}')  (both, since nest-no-pack drops both)"
    info "    the loop nest       $(awk -v a="${npret:-0}" 'BEGIN{printf "%+.1f", (1-a)*100}')  (macro loops, beta pass, real C, fringe tile)"
    if awk -v a="${npret:-0}" 'BEGIN{exit !(a > 1)}'; then
      info "    the loop-nest term is NEGATIVE: nest-no-pack is faster than the flat"
      info "    call sequence, so kernel-calls is not an upper bound either and this"
      info "    split is not a cost breakdown. Reported, not explained away."
    fi
    info "  parts of the blocked time: nest-no-pack $(pct "$np")  pack-a $(pct "$pa")  pack-b $(pct "$pb")  residual $(pct "$resid")"
    info "  (residual is a point estimate with no interval — a difference of four"
    info "   medians. It holds what nest-no-pack drops by construction: the"
    info "   pack/kernel cache interference, and gemm's three per-call allocations."
    info "   See internal/block/nest_bench_test.go.)"
  done < <(awk '/^Benchmark.*\/full/ { n=$1; sub(/-[0-9]+$/,"",n); sub(/^Benchmark/,"",n); sub(/\/full$/,"",n); print n }' "$log" | sort -u)
}

# sweep_host HOST CSV LOG — the KC/MC/NC grid, ranked, with the shipped point marked.
sweep_host() {
  local host="$1" csv="$2" log="$3" kc mc nc gridnc shipped mark
  read -r kc mc nc < <(awk '
    /^[[:space:]]*KC = / {kc=$3} /^[[:space:]]*MC = / {mc=$3} /^[[:space:]]*NC = / {nc=$3}
    END{print kc, mc, nc}' internal/block/block.go)
  shipped="kc=$kc/mc=$mc/nc=$nc"
  # The row to mark is not the shipped triple's own name. The grid's largest NC is
  # smaller than the shipped NC, so matching "nc=$nc" literally marked nothing at
  # all — a marker that could never fire, which reads as "the shipped point is not
  # on the grid" (second defect found while fixing #49). What is true, and all
  # that is claimed here, is that the shipped KC/MC appear on the grid, and that
  # at the grid's largest NC. Whether that row *is* the shipped configuration
  # depends on n, which this script cannot see: NC clamps to n, so it is the same
  # single block whenever the grid's largest NC is >= n, and a different blocking
  # otherwise. Hence the label, which states the condition rather than assuming it.
  gridnc="$(awk -F, '{ if (match($1, /nc=[0-9]+/)) { v = substr($1, RSTART+3, RLENGTH-3) + 0; if (v > m) m = v } } END { print m+0 }' "$csv")"
  mark="kc=$kc/mc=$mc/nc=$gridnc"
  info "[$host] shipped point: $shipped (internal/block/block.go). NC>=n collapses to one point at n=2048,"
  info "        so the marked row is the shipped KC/MC at the grid's largest NC ($gridnc)."
  awk -F, '
    /^,/ { unit = $2; next }
    unit == "GFLOP/s" {
      name = $1; sub(/-[0-9]+$/, "", name); sub(/^BenchmarkBlocking\//, "", name)
      print $2, name
    }' "$csv" | sort -rn | awk -v s="$mark" '
      { m = (index($2, s) > 0) ? "  <- shipped KC/MC" : ""
        printf "        %2d. %8.1f GFLOP/s  %s%s\n", ++i, $1, $2, m }'
}

main() {
  cd "$(dirname "$0")/.."
  # Captured before scripts/bench.sh is sourced: sourcing it assigns
  # KEEL_BENCH_COUNT its own default, after which the sweep's
  # "${KEEL_BENCH_COUNT:-5}" can distinguish neither the caller's setting nor its
  # own default from bench.sh's (issue #49).
  local requested="${KEEL_BENCH_COUNT-}"
  # shellcheck source=scripts/remote.sh
  source scripts/remote.sh
  # shellcheck source=scripts/bench.sh
  source scripts/bench.sh

  local MODE="${1:-decompose}"
  case "$MODE" in
    decompose|sweep) ;;
    *) echo "usage: $0 [decompose|sweep]" >&2; exit 2 ;;
  esac

  BINDIR="$(mktemp -d)"
  trap 'rm -rf "$BINDIR"' EXIT
  local BIN="$BINDIR/block.test" LOG="$BINDIR/log" CSV="$BINDIR/csv"

  echo "== retention ($MODE) — issue #26. Not a gate: this certifies nothing. =="
  echo

  local HOSTS
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
    KEEL_BENCH_COUNT="${requested:-5}"
    # shellcheck disable=SC2034  # read by bench_expect in the sourced scripts/bench.sh
    KEEL_BENCH_MIN_ROWS="$KEEL_BENCH_COUNT"
  fi
  local BFLAGS FILTER='BenchmarkNest'
  mapfile -t BFLAGS < <(bench_flags)
  [[ "$MODE" == sweep ]] && FILTER='BenchmarkBlocking'

  echo "methodology: -count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, GOMAXPROCS=1, benchstat medians"
  [[ "$MODE" == sweep ]] && echo "             EXPLORATORY: re-measure any nominated point at -count=10 before it becomes a default"
  echo

  local host prov
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
    # What arrived, not what was asked for: -count is a request, and the header
    # above prints the request. This line is the log counting itself (#49).
    info "samples this host produced: $(rows_per_bench "$LOG")"
    # The plan markers next: they say which blocks the parts were measured over.
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
}

main "$@"
