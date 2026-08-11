#!/usr/bin/env bash
# Shared helper: the one measurement methodology every gate uses.
#
# WHY THIS EXISTS. P1's first perf numbers came from `-benchtime=3x -count=5`
# reduced by min-of-samples, and the raw samples spread by up to 3.08x within a
# single backend. A 4.28x result standing on that is not a measurement, it is a
# draw. Decision on issue #14: gate benchmarks run -count=10 -benchtime=1s under
# the performance governor, are aggregated by benchstat, and a threshold counts
# as cleared only if the *median net of benchstat's reported confidence interval*
# clears it. A lucky run can no longer turn a gate green.
#
# Consequences worth stating out loud:
#
#  - The gate never reads a raw ns/op. Every number it compares comes out of
#    benchstat, so the aggregation is one auditable tool rather than an awk
#    one-liner per gate. benchstat is pinned as a module tool in go.mod, so the
#    version is recorded in the repo instead of being whatever `@latest` was on
#    the day the gate ran.
#
#  - Ratios are computed conservatively at both ends: the numerator is pushed
#    down by its CI and the denominator up by its own. A ratio that clears the
#    bar here would still clear it if both measurements were as wrong as
#    benchstat thinks they might be.
#
#  - No confidence interval, no verdict. Fewer than 6 samples, or a distribution
#    too wide for benchstat to bound, yields "∞" — which is treated as a failure
#    to measure, not as a pass.
#
#  - -benchtime=3x remains fine for smoke runs that inform nobody's gate.
#
# Sourced by scripts/gate-p*.sh; not meant to be run directly.

# The methodology, in one place so no gate can quietly deviate from it.
KEEL_BENCH_COUNT="${KEEL_BENCH_COUNT:-10}"
KEEL_BENCH_TIME="${KEEL_BENCH_TIME:-1s}"

# bench_flags — the -test.* flags for a gate benchmark run.
bench_flags() {
  printf '%s\n' -test.run=NONE "-test.count=$KEEL_BENCH_COUNT" "-test.benchtime=$KEEL_BENCH_TIME"
}

# bench_csv OUT — aggregate a benchmark log with the pinned benchstat.
#
# stderr is kept: benchstat's warnings ("need >= 6 samples...") are exactly the
# kind of thing a gate must not swallow.
bench_csv() {
  go tool benchstat -format=csv "$1"
}

# bench_stat NAME CSV [UNIT] — print "median ci_fraction" for one benchmark.
#
# UNIT defaults to sec/op. Prints nothing if the benchmark is absent under that
# unit, and a ci of "inf" when benchstat could not establish an interval. The
# unit is tracked from the section headers so a B/s or GFLOP/s row can never be
# read as a time: benchstat emits one section per unit and every section carries
# the same benchmark names, which is a trap worth closing once here rather than
# per gate.
bench_stat() {
  awk -F, -v want="$1" -v wantunit="${3:-sec/op}" '
    /^,/ { unit = $2; next }
    {
      name = $1
      sub(/-[0-9]+$/, "", name)
      if (name == want && unit == wantunit) {
        ci = $3
        sub(/%$/, "", ci)
        if (ci == "" || ci ~ /∞/) ci = "inf"; else ci = ci / 100
        print $2, ci
        exit
      }
    }' "$2"
}

# bench_ratio_lo NUM DEN CSV [UNIT] — conservative lower bound on NUM/DEN.
#
# (median_num · (1 − ci_num)) / (median_den · (1 + ci_den)): the smallest ratio
# consistent with both confidence intervals. Prints nothing if either benchmark
# is missing or unbounded, so the caller must treat empty as "not measured"
# rather than as zero.
#
# The formula is unit-agnostic on purpose. For sec/op the caller puts the slow
# side on top (scalar/vector) and reads a speedup; for GFLOP/s it puts the
# achieved rate on top and reads a fraction of peak. Both want the same
# conservative treatment — numerator down by its interval, denominator up by
# its own — so both get it from one function.
bench_ratio_lo() {
  local n d
  n="$(bench_stat "$1" "$3" "${4:-sec/op}")"
  d="$(bench_stat "$2" "$3" "${4:-sec/op}")"
  [[ -n "$n" && -n "$d" ]] || return 0
  awk -v n="$n" -v d="$d" '
    BEGIN {
      split(n, a, " "); split(d, b, " ")
      if (a[2] == "inf" || b[2] == "inf") exit
      printf "%.3f", (a[1] * (1 - a[2])) / (b[1] * (1 + b[2]))
    }'
}

# bench_ratio NUM DEN CSV [UNIT] — the point estimate, for reporting alongside
# the bound. Printing both is the point: the gap between them is how much of the
# result is measurement noise, and hiding it would make a 43%-CI run look like a
# clean one (see docs/hosts.md on antares).
bench_ratio() {
  local n d
  n="$(bench_stat "$1" "$3" "${4:-sec/op}")"
  d="$(bench_stat "$2" "$3" "${4:-sec/op}")"
  [[ -n "$n" && -n "$d" ]] || return 0
  awk -v n="$n" -v d="$d" 'BEGIN { split(n, a, " "); split(d, b, " "); printf "%.3f", a[1] / b[1] }'
}

# bench_describe NAME CSV [UNIT] — a human-readable "1.23e-06 s +/- 2%" for a
# gate line. Rate units are printed with their own name and 4 significant
# figures; sec/op keeps the exponent form, which is what benchstat gives.
bench_describe() {
  local s unit
  unit="${3:-sec/op}"
  s="$(bench_stat "$1" "$2" "$unit")"
  [[ -n "$s" ]] || { printf 'not measured'; return 0; }
  awk -v s="$s" -v unit="$unit" '
    BEGIN {
      split(s, a, " ")
      label = (unit == "sec/op") ? "s" : unit
      if (a[2] == "inf") printf "%.4g %s (no CI: too few or too noisy samples)", a[1], label
      else printf "%.4g %s +/- %.1f%%", a[1], label, a[2] * 100
    }'
}

# bench_gflops NAME CSV — the GFLOP/s a benchmark reported, median over samples,
# without its interval. For printing a provenance line; anything that compares
# against a threshold must go through bench_ratio_lo instead, which is why this
# drops the CI rather than returning it and inviting a raw comparison.
bench_gflops() {
  bench_stat "$1" "$2" GFLOP/s | awk '{ print $1 }'
}
