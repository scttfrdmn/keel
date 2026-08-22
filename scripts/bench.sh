#!/usr/bin/env bash
# Shared helper: the one measurement methodology every gate uses.
#
# WHY THIS EXISTS. P1's first perf numbers came from `-benchtime=3x -count=5`
# reduced by min-of-samples, and the raw samples spread by up to 3.08x within a
# single backend. A 4.28x result standing on that is not a measurement, it is a
# draw. Decision on issue #14: gate benchmarks run -count=10 -benchtime=1s on a
# host whose clock has been established stable (DESIGN.md §5 rule 5 as amended
# 2026-08-16 — the `performance` governor where `cpufreq` is readable, else
# `BenchmarkPeak` sampled head/middle/tail), are aggregated by benchstat, and a
# threshold counts
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
# How many result rows a declared benchmark must have produced before a criterion
# is allowed to read it (bench_expect). It is -count, not benchstat's minimum of
# 6: a run that produced 7 of 10 rows is a run that died partway, and it should
# say so rather than be silently aggregated over whatever survived.
KEEL_BENCH_MIN_ROWS="${KEEL_BENCH_MIN_ROWS:-$KEEL_BENCH_COUNT}"

# rows_per_bench LOG — sample rows this run actually produced per benchmark,
# counted out of the log. A header prints both this and the requested -count,
# because #49 was a caller's `${KEEL_BENCH_COUNT:-5}` silently losing to the
# default declared above it, and a parameter read back out of the measurement
# cannot be shadowed by whatever set it. It lives beside that declaration, not in
# retention.sh where it was written: a caller that sources this file for the
# count and then cannot count its log is #49 again, which is what build/trsm-mb.sh
# did on all six arms of the Trsm MB sweep.
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

# bench_flags — the -test.* flags for a gate benchmark run.
bench_flags() {
  printf '%s\n' -test.run=NONE "-test.count=$KEEL_BENCH_COUNT" "-test.benchtime=$KEEL_BENCH_TIME"
}

# bench_csv LOG [TAG] — aggregate a benchmark log, and keep the samples.
#
# stderr is kept: the aggregator's warnings ("need >= 6 samples...") are exactly
# the kind of thing a gate must not swallow.
#
# WHY NOT `benchstat -format=csv` ANY MORE (#110, ruled 2026-08-19). benchstat's
# CSV `CI` column is a *display* string: `benchmath.Summary` holds float64 bounds
# and `benchtab.ToCSV` writes the center at full precision beside
# `PctRangeString()`, which is `%.0f%%`. Every criterion here grades a threshold
# *net of that interval*, so a CI that rounds to `0%` makes the bound equal the
# raw ratio — the degeneracy DESIGN.md's P4 clause exists to prevent — and it
# decided a shipped verdict (janus Strsm, FAIL -> PASS on a 0.014% move in the
# point estimate). The quantum, 0.5%, exceeded every margin it adjudicated:
# 0.1386 on a 7.0 ratio against margins of 0.011 and 0.081.
#
# The ruling: no arithmetic in the rounded domain can make those verdicts
# measurement-decided, so the instrument gains resolution instead.
# tools/benchci is benchstat's own statistics — same projections, same
# confidence, same thresholds, same per-unit assumption read from the file — with
# the formatting step replaced, and `-verify` requires that rounding its CI back
# to `%.0f%%` reproduces benchstat's column cell for cell.
#
# EVERY CONSUMER IS BEHIND THIS ONE FUNCTION, which is why the fix is four lines
# and not eleven call sites; bench_stat's parse is unchanged, so bench_ratio_lo's
# formula is bit-identical and only its inputs got sharper.
#
# ARCHIVING IS HERE FOR THE SAME REASON. BENCHLOG lives in gate_tmpdir's
# `mktemp -d`, which the EXIT trap removes, so until now the raw samples of every
# judged run in this project's history were destroyed at the end of it — leaving
# the rounded gate log as the only record, which is precisely why #110's
# historical verdicts can only be re-read by band-top instead of recomputed. A
# summarizer that can reach the samples is worth nothing if nothing kept them.
# Archived from inside the aggregator so no call site can forget: whatever gets
# summarized gets saved.
#
# The path is SET, not printed. This function's stdout is the CSV and its stderr
# is the aggregator's warning stream, which every gate relays under a label; a
# line announcing an archive on that stream would arrive labelled as a warning
# about the measurement, and would make the "any warnings?" branch fire on every
# run. The caller prints $BENCH_ARCHIVE where it already prints provenance.
#
# THE COUNTER IS PER PROCESS, so it discriminated archives within a run and nothing
# between runs: two runs at one rev on one host wrote one path and the second overwrote
# the first (measured, 2026-08-21). Hence the run stamp, set on first use. It goes at the
# END because readme-numbers.sh reads the rev by offset from `bench-gate-p5-`.
bench_csv() {
  BENCH_ARCHIVE_RUN="${BENCH_ARCHIVE_RUN:-$(date -u +%Y%m%dT%H%M%SZ)}"
  BENCH_ARCHIVE_N=$((${BENCH_ARCHIVE_N:-0} + 1))
  BENCH_ARCHIVE="build/bench-$(basename "${0%.sh}")-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)${2:+-$2}-$BENCH_ARCHIVE_RUN-$BENCH_ARCHIVE_N.txt"
  mkdir -p build && cp "$1" "$BENCH_ARCHIVE" 2>/dev/null || BENCH_ARCHIVE="(not archived)"
  go run ./tools/benchci "$1"
}

# bench_pin LOG — "mask width" as the measurement itself recorded them, or nothing.
#
# Read off the artifact that carries the numbers, never off the driver that asked for the
# mask: a constant this shell can print certifies only that this shell can print it, and
# what a criterion needs to know is what the far side actually ran under. remote_exec
# writes the line immediately before the binary, so it is inside the same log as the rows
# it shaped and travels with them into the archive.
bench_pin() { sed -n 's/^keel-pin: mask=\([^ ]*\) width=\([0-9]*\)$/\1 \2/p' "$1" | tail -1; }

# bench_gomaxprocs LOG — the distinct -GOMAXPROCS suffixes Go appended to the row names
# in LOG, space-separated and sorted. The SECOND reading of the mask width, and an
# independent one: bench_pin above reports what the harness asked the kernel for, this
# reports what the Go runtime saw when it called sched_getaffinity. A mask that was
# requested and did not take shows up here and nowhere else.
bench_gomaxprocs() {
  grep -oE '^Benchmark[^[:space:]]*-[0-9]+' "$1" | sed 's/.*-//' | sort -un | tr '\n' ' ' |
    sed 's/ $//'
}

# Configuration keys that describe the run rather than the build under test, and
# so must not split a comparison. See bench_compare.
KEEL_BENCH_IGNORE="${KEEL_BENCH_IGNORE:-keel-bench-clock-mhz,keel-bench-flops}"

# bench_config FILE — the "key: value" configuration lines benchstat reads out of
# a benchmark log, sorted. Go's benchmark format treats any such line as
# configuration applying to the rows that follow it, wherever it appears.
bench_config() { grep -E '^[a-zA-Z][a-zA-Z0-9_-]*: ' "$1" | sort -u; }

# bench_compare BASE NEW — benchstat's two-file comparison, with the comparison
# guaranteed to be present or the failure said out loud.
#
# WHY THIS EXISTS. benchstat groups results into one table per distinct
# *configuration*, and a configuration is every "key: value" line in the log. Two
# files that differ in one such key are therefore not compared at all: benchstat
# prints two independent single-column tables, no deltas, no p-values — and exits
# 0. scripts/l1-bench.sh's first run hit exactly this. Its arms differed only in
# `keel-bench-clock-mhz`, which is a live snapshot of the CPU's clock range and so
# differs between any two runs on the same host by construction, and the output
# looked like a comparison (two tables of the same benchmarks, one per build)
# while containing not one delta. This repo's own provenance preamble is what
# broke it: those markers exist so no number ships without its denominator, and
# they are in benchstat's configuration namespace whether we meant them to be or
# not.
#
# So: the volatile keys are ignored explicitly, and then the result is checked for
# the thing that was wanted. A comparison that did not happen is a failed
# measurement, and it now reads as one instead of as a table.
bench_compare() {
  local base="$1" new="$2" out differ
  out="$(go tool benchstat -ignore="$KEEL_BENCH_IGNORE" "$base" "$new" 2>&1)" || true
  printf '%s\n' "$out"
  grep -q 'vs base' <<<"$out" && return 0
  # The ignored keys are dropped from this list: they differ in most runs and did
  # not cause the fork, so naming them would bury whatever did.
  differ="$(diff <(bench_config "$base") <(bench_config "$new") |
    sed -n 's/^[<>] //p' | sed 's/:.*/: <differs>/' | sort -u |
    grep -Ev "^(${KEEL_BENCH_IGNORE//,/|}):" || true)"
  printf 'NOT COMPARED: benchstat produced no "vs base" column, so the two arms above\n'
  printf 'are independent tables and nothing here is a delta. '
  if [[ -n "$differ" ]]; then
    printf 'Configuration keys that\ndiffer between the two logs (each one forks the table):\n'
    printf '%s\n' "$differ" | sed 's/^/  /'
  else
    printf 'The two logs configure\nbenchstat identically, so the cause is not a forked table: look for an arm with\nno benchmark rows in it.\n'
  fi
  return 1
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

# bench_expect LOG CSV UNIT NAME... — the declared-then-checked half of every
# measurement. Prints one word per declared benchmark that did not fully arrive,
# each with the state it was found in, and returns 1 if any did not.
#
# WHY THIS EXISTS. bench_stat prints nothing both when a benchmark never ran and
# when it ran and reported nothing under the unit asked for, so every call site
# had to invent its own reading of empty — and each read it as whatever that
# criterion happened to be about. Issue #32 was a -bench filter that silently
# never ran BenchmarkOpenBLAS for the whole of P3: the gate reported "no
# OpenBLAS/n=2048 benchmark result to divide by", which reads as a missing
# library, and the real cause — the gate measuring less than it claimed — went
# unnamed. A gate's product is verdicts, so a red for the wrong reason is as
# untrustworthy as a green for the wrong reason (DESIGN.md §5 rule 6).
#
# So a criterion states up front which benchmarks it intends to read and calls
# this before reading any of them. An absent measurement then has exactly one
# verdict available to it — unmeasured — instead of a shape each caller misreads
# its own way.
#
# The three states are reported distinctly rather than collapsed into "missing",
# because they have different causes: no rows at all is a filter that did not
# select the benchmark, too few rows is a run that died partway, and rows without
# a unit section is a benchmark reporting a metric this gate does not read.
#
# Row counting is exact-name, matching bench_stat and the sweep parser: strip the
# -GOMAXPROCS suffix, strip the Benchmark prefix, compare as a string. A prefix
# match would let Sgemm/n=2048 be satisfied by Sgemm/n=20480.
bench_expect() {
  local log csv unit name got bad=""
  log="$1"; csv="$2"; unit="$3"; shift 3
  for name in "$@"; do
    got="$(awk -v want="$name" '
      /^Benchmark/ {
        n = $1
        sub(/-[0-9]+$/, "", n)
        sub(/^Benchmark/, "", n)
        if (n == want) c++
      }
      END { print c + 0 }' "$log")"
    if [[ "$got" -eq 0 ]]; then
      bad="$bad ${name}(no result row at all: this run never produced it)"
    elif [[ "$got" -lt "$KEEL_BENCH_MIN_ROWS" ]]; then
      bad="$bad ${name}($got of $KEEL_BENCH_MIN_ROWS sample rows: the run did not finish it)"
    elif [[ -z "$(bench_stat "$name" "$csv" "$unit")" ]]; then
      bad="$bad ${name}($got sample rows, but benchstat reports no $unit for it)"
    fi
  done
  [[ -z "$bad" ]] && return 0
  printf '%s\n' "${bad# }"
  return 1
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

# bench_ratio_hi NUM DEN CSV [UNIT] — the mirror of bench_ratio_lo: the LARGEST
# ratio consistent with both confidence intervals,
# (median_num · (1 + ci_num)) / (median_den · (1 − ci_den)).
#
# WHY A GATE NEEDS THE UPPER END TOO. bench_ratio_lo alone supports exactly one
# question — "is the whole interval above the bar?" — and a two-state gate reads
# its negative answer as "below the bar". Those are different claims, and issue
# #67 is the case where the difference mattered: janus's Ssyrk/Sgemm ratio read
# 87.6% raw with ±4.0%/±3.0% intervals, so the bound landed at 81.6% and the
# criterion failed, while the same tree on the same commit read 87.0% raw at
# ±0.0% and passed. The raw quantity agreed to within 0.6 points both times. What
# the FAIL actually recorded was that the day was noisy, and DESIGN.md §5.6
# forbids one verdict standing for two causes.
#
# So a threshold comparison gets three outcomes, and this is the second bound it
# needs. See bench_ratio_grade.
bench_ratio_hi() {
  local n d
  n="$(bench_stat "$1" "$3" "${4:-sec/op}")"
  d="$(bench_stat "$2" "$3" "${4:-sec/op}")"
  [[ -n "$n" && -n "$d" ]] || return 0
  awk -v n="$n" -v d="$d" '
    BEGIN {
      split(n, a, " "); split(d, b, " ")
      if (a[2] == "inf" || b[2] == "inf") exit
      if (b[2] >= 1) exit   # the denominator interval reaches zero: the ratio is unbounded above
      printf "%.3f", (a[1] * (1 + a[2])) / (b[1] * (1 - b[2]))
    }'
}

# bench_ratio_grade NUM DEN CSV UNIT BAR — three-state verdict for "is NUM/DEN at
# or above BAR?". Prints exactly one of:
#
#   pass           the whole interval sits at or above BAR
#   fail           the whole interval sits below BAR
#   indeterminate  the interval straddles BAR — this measurement cannot decide
#   unbounded      benchstat established no interval for one of the two arms
#
# and prints nothing if either benchmark is missing, so empty stays "not
# measured" as everywhere else in this file.
#
# THE PASS CONDITION IS UNCHANGED, BIT FOR BIT. `pass` is `bench_ratio_lo >= BAR`,
# which is the same comparison every gate here has made since issue #14. Nothing
# that passed before passes differently now, and nothing that was below the bar
# gets in: the third state is carved out of the old FAIL, never out of the old
# PASS. Replayed against the six archived criterion-7 readings at 746cc98 and
# 2eda333 before it landed: five stayed PASS and the one noisy janus reading
# (interval [81.6%, 93.9%] about a raw 87.5%, bar 85%) moved FAIL ->
# indeterminate, which is the reading that started #67. The replay drives these
# functions from the medians as the logs *printed* them, which bench_describe
# renders to 4 significant figures, so a replayed raw ratio can sit 0.1 point off
# the gate's own — antares replays 88.3% where its log says 88.2%. Verdicts are
# unaffected (every margin here is more than a point), but the numbers are a
# reproduction to display precision, not to the gate's.
#
# The remedy for indeterminate is precision, never a wider judgment: one archived
# re-run under DESIGN.md §4's allowance, and if a host is *chronically*
# indeterminate on a criterion, raise KEEL_BENCH_COUNT for that criterion on that
# host. Moving the bar, or grading the raw ratio, would let a genuinely-below
# ratio through on a lucky draw. That is the one thing three-state grading exists
# to prevent.
bench_ratio_grade() {
  local lo hi bar
  bar="$5"
  lo="$(bench_ratio_lo "$1" "$2" "$3" "$4")"
  hi="$(bench_ratio_hi "$1" "$2" "$3" "$4")"
  # Missing is not the same as unbounded: bench_ratio_lo prints nothing for both,
  # so the arms are re-read here to tell them apart.
  [[ -n "$(bench_stat "$1" "$3" "$4")" && -n "$(bench_stat "$2" "$3" "$4")" ]] || return 0
  if [[ -z "$lo" || -z "$hi" ]]; then
    printf 'unbounded\n'
    return 0
  fi
  awk -v lo="$lo" -v hi="$hi" -v bar="$bar" 'BEGIN {
    if (lo >= bar)      print "pass"
    else if (hi < bar)  print "fail"
    else                print "indeterminate"
  }'
}

# bench_ratio_headroom RAW BAR — the symmetric confidence interval at which a raw
# ratio's conservative lower bound reaches BAR. Prints a fraction; negative means
# the raw ratio is itself below BAR, so no amount of quiet would clear it.
#
# net = raw·(1−a)/(1+a) = bar  =>  a = (raw − bar)/(raw + bar)
#
# This is a diagnostic, not a verdict, and it is printed every run because it is
# the number that says how close a green ran to undecidable. At the 85% bar it
# came out 4.17% on the 7950X3D, 1.16% on the i9-9960X and 1.85% on the AI MAX+
# 395 — against intervals those hosts produce up to 1.0%, 3.0% and 2.0% in the
# same gate. Two of three decide that criterion on how quiet the machine was, and
# no one could see it from the log until it was printed (#67).
bench_ratio_headroom() {
  awk -v raw="$1" -v bar="$2" 'BEGIN {
    if (raw + bar <= 0) exit
    printf "%.4f", (raw - bar) / (raw + bar)
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

# bench_gflops_lo NAME CSV — median · (1 − ci): the smallest rate consistent with
# the interval. Prints nothing when benchstat established no interval, so an
# unbounded measurement reads as "not measured" rather than as its median.
#
# This is a bare bound, not a ratio, and it exists for exactly one comparison that
# bench_ratio_lo cannot express: two rates from two different benchmark
# invocations, where there is no common denominator to divide by. Do not use it to
# build a ratio — dividing this by another median is precisely the second
# statistics-free denominator DESIGN.md §7 rule 7 forbids, and bench_ratio_lo
# handles both intervals correctly when the two numbers share a CSV.
bench_gflops_lo() {
  bench_stat "$1" "$2" GFLOP/s |
    awk '{ if ($2 == "inf") exit; printf "%.4f", $1 * (1 - $2) }'
}

# ---------------------------------------------------------------------------
# §5 rule 5's substitute clock instrument, for a host with no governor to assert.
#
# WHAT THE RULE ASKS FOR, AND NOTHING MORE. §5 rule 5 as amended 2026-08-16 names the
# instrument and both of its verdicts itself: BenchmarkPeak sampled at the head, middle
# and tail of a sweep, where "a declining series is throttling while a wide one is
# contention", and it says in terms that this "adds no benchmark and no new bar" —
# peak is already the sweep's own denominator, and "too wide for benchstat to bound is a
# failure to measure" above it already refuses the wide case. So there is no threshold
# in this section to tune, and there must not be one: a bar chosen after meeting the
# fleet it judges is not a criterion, it is a result. The two tests are therefore
# `benchstat established an interval for each window` and `the three medians are not
# monotonically decreasing`, both of which are threshold-free.
#
# THREE INVOCATIONS, NOT ONE, which is the part that is easy to get wrong and was
# nearly got wrong here. `-count` is Go's INNER loop — measured rather than read:
# `-bench='Aaa|Zzz' -count=3` prints Aaa three times and then Zzz three times — so
# BenchmarkPeak's ten rows inside a sweep occupy ONE contiguous window of wall clock.
# Their spread is the spread of ten adjacent seconds and says nothing about the twenty
# minutes around them. Reusing them as "head, middle and tail" would be an instrument
# that reads a real number and answers a different question, which is exactly the
# `cpu MHz` failure the rule warns about one layer up. So the sweep's own rows are the
# MIDDLE, and the head and tail are separate invocations either side of it.
#
# WHY THE VERDICT IS unmeasured AND NEVER fail. A throttling or contended host says
# nothing about keel. §5.6 forbids one verdict standing for two causes, and "this
# kernel is slow" and "this machine would not hold still" are two causes.
# THREE CALLS, BECAUSE THE THREE POINTS ARE NOT ALWAYS ADJACENT. clock_gate says
# whether the host may be measured, clock_head opens the series, clock_post closes and
# judges it. gate-p3 is why that is three functions and not one: its middle window comes
# from the natively built OpenBLAS harness, so `clock_head` has to open at a deliberately
# tight bracket around the judged run and exclude the minutes-long coretype sweep that
# precedes it. Every one of the three returns non-zero to refuse, and every
# call site is `... || continue`; called bare, the refusal would take `set -e` with it.
CLOCK_STATE=""       # ok | stable | declining | unbounded | incomplete | <GOV_STATE>
CLOCK_HEAD=""        # median GFLOP/s of the head window, empty if there was none
CLOCK_TAIL=""

# peak_window HOST BIN TAG — one BenchmarkPeak-only invocation. Prints its median
# GFLOP/s, or nothing if the window did not fully arrive; TAG names the files so the
# head and tail windows cannot overwrite each other. Guarded like every other reader
# here: an unreachable host must not become the gate's own exit status (#76).
#
# BIN is normally a local binary, shipped and supervised by remote_exec like every other
# measurement here. Written `@DIR/NAME` it is one that already exists on the far side and
# is run in place, which gate-p3's OpenBLAS section needs: its middle window comes from a
# harness built natively on the host, and three windows from two different compilers is a
# trend test with a build in it. That form forgoes #62's supervision, which is the right
# way round — a lost peak window resolves to `incomplete`, the verdict a lost measurement
# is supposed to get, and it is a seconds-long run rather than the twenty-minute sweep
# supervision exists for.
#
# GOMAXPROCS is not set: BenchmarkPeak is a single-goroutine register loop, so its rate
# cannot depend on it, and the callers disagree (p4 pins it to 1, p5 deliberately does
# not). Setting it would make the window differ from the middle on one of them.
peak_window() {
  local host="$1" bin="$2" tag="$3" log csv flags=() args="" f
  log="$BINDIR/peak-$tag.log"; csv="$BINDIR/peak-$tag.csv"
  while read -r f; do flags+=("$f"); done < <(bench_flags)
  if [[ "$bin" == @* ]]; then
    local rbin="${bin#@}"
    for f in "${flags[@]}" "-test.bench=$GATE_PEAK"; do args+=" $(printf '%q' "$f")"; done
    # shellcheck disable=SC2029  # client-side expansion of this script's own values
    ssh "${KEEL_SSH_OPTS[@]}" "$host" "cd '${rbin%/*}' && ./'${rbin##*/}'$args" >"$log" 2>&1 || return 0
  elif ! remote_exec "$host" "$bin" "${flags[@]}" -test.bench="$GATE_PEAK" >"$log" 2>&1; then
    return 0
  fi
  bench_csv "$log" >"$csv" 2>/dev/null || return 0
  bench_expect "$log" "$csv" GFLOP/s "$GATE_PEAK" >/dev/null || return 0
  bench_gflops "$GATE_PEAK" "$csv"
}

# clock_gate HOST — may this host be measured at all. Call immediately after
# `assert_governor HOST measured`.
#
# It absorbs the `GOV_STATE != performance` branch that stood at this point in all five
# gates rather than sitting beside it, because two consecutive guards on one question is
# how a host comes to be refused by one and admitted by the other.
clock_gate() {
  CLOCK_STATE=""; CLOCK_HEAD=""; CLOCK_TAIL=""
  case "$GOV_STATE" in
    performance|nocpufreq) CLOCK_STATE=ok; return 0 ;;
  esac
  # assert_governor has already printed the verdict and the cause for every other
  # state; restating it here would be the second place a reason lives.
  CLOCK_STATE="$GOV_STATE"
  return 1
}

# clock_head HOST BIN — open the peak series, on a host whose clock is established by
# peak rather than by a governor. Call immediately before the sweep, after any
# intervention the sweep will run under.
clock_head() {
  local host="$1" bin="$2"
  [[ "$CLOCK_STATE" == ok ]] || return 1
  [[ "$GOV_STATE" != performance ]] || return 0
  CLOCK_HEAD="$(peak_window "$host" "$bin" head)"
  [[ -n "$CLOCK_HEAD" ]] && return 0
  unmeasured "[$host] the head peak window did not produce a result, so §5 rule 5's substitute clock instrument has nothing to start from and this host's readings are unestablished rather than bad"
  CLOCK_STATE=incomplete
  return 1
}

# clock_post HOST BIN SWEEPCSV — take the tail window and judge the series. Returns 0
# when the host's readings may be used. A `performance` host passes straight through: its
# clock was established by the governor, at both times, and running two more windows on
# it would spend minutes a run to re-establish something already asserted by a cheaper
# instrument the rule still prefers.
clock_post() {
  local host="$1" bin="$2" csv="$3" mid
  [[ "$CLOCK_STATE" == ok ]] || return 1
  if [[ "$GOV_STATE" == performance ]]; then CLOCK_STATE=stable; return 0; fi
  mid="$(bench_gflops "$GATE_PEAK" "$csv")"
  CLOCK_TAIL="$(peak_window "$host" "$bin" tail)"
  if [[ -z "$mid" || -z "$CLOCK_TAIL" ]]; then
    unmeasured "[$host] the peak series is incomplete (head=${CLOCK_HEAD:-none} middle=${mid:-none} tail=${CLOCK_TAIL:-none}), so the clock is unestablished on a host that has no governor to fall back on"
    CLOCK_STATE=incomplete
    return 1
  fi
  info "[$host] peak series GFLOP/s: head $CLOCK_HEAD, middle $mid, tail $CLOCK_TAIL (three invocations either side of the sweep; §5 rule 5's substitute instrument, no governor on this host)"
  # DISPLAYED IN GFLOP/s ABOVE, JUDGED ON sec/op BELOW, and at this magnitude the two are not
  # interchangeable. `testing.prettyPrint` picks its decimals from a value's MAGNITUDE, not
  # from the precision of the measurement (T26): under 999.95 a column gets 4 significant
  # figures, over it every integer digit. Both columns of one row go through it, so 245 GFLOP/s
  # is quantized to 0.1 (0.041%, and a median of ten lands on a 0.05 multiple) while the same
  # measurement's 102762 ns/op is quantized to 1 ns (0.00097%) -- one run, a rate and its
  # reciprocal, 42x apart in resolution. Every "decline" this test ever reported was one
  # quantum of the coarse column, i.e. inside the print rounding. benchci then keeps the
  # interval at full float64. Re-parsing a rounded display string is the defect #110 named one
  # field up (T21); this is the same defect one column over. NOT a custom-metric rule: ns/op is
  # formatted by the same function and is only saved by where it sits.
  local v
  v="$(clock_series "$(bench_stat "$GATE_PEAK" "$BINDIR/peak-head.csv" sec/op)" \
                    "$(bench_stat "$GATE_PEAK" "$csv" sec/op)" \
                    "$(bench_stat "$GATE_PEAK" "$BINDIR/peak-tail.csv" sec/op)")"
  case "${v%% *}" in
    ''|unbounded|declining)
      CLOCK_STATE="${v%% *}"; CLOCK_STATE="${CLOCK_STATE:-unbounded}"
      unmeasured "[$host] ${v#* }"
      return 1 ;;
  esac
  pass "[$host] ${v#* }"
  CLOCK_STATE=stable
}

# clock_series HEAD MIDDLE TAIL — judge three `bench_stat` readings ("median ci"). Prints
# `<state> <message>`; state is `declining`, `unbounded` or `stable`.
#
# THE MAGNITUDE GATE (ruled 2026-08-20, #6). The ordering test as it stood asked only
# `h > m > t`, so it decided on the sign of differences smaller than its own resolution: on
# keel-gnr it refused two of four triples across a total spread of 0.14%, and a random
# triple is strictly decreasing one time in six, so ~1 in 4 refusals were arriving by
# chance. That is rank statistics on noise, and ordering fabricated from sub-quantum
# differences is fabricated. So a step counts as a decline only when the two windows'
# intervals are DISJOINT in that direction — A's slowest still faster than B's fastest —
# and a monotone verdict needs both steps to clear it.
#
# THE FLOOR IS MEASURED, NEVER TYPED, which is what makes this a resolution gate and not a
# new bar: `(1+cA)/(1-cB) - 1` comes from the two windows' own intervals, computed by the
# same run, on the same host, from the same samples. Nothing here was chosen after meeting
# the fleet — §5 rule 5's "no threshold in this section to tune" is intact. A genuine droop
# clears any honest floor by construction; only the phantoms die.
clock_series() {
  awk -v h="$1" -v m="$2" -v t="$3" 'BEGIN {
    n = split(h, H, " ") + split(m, M, " ") + split(t, T, " ")
    # The wide case, refused by the rule that already existed: too wide for benchstat to
    # bound is a failure to measure, not a rate. Fails closed on a missing reading too,
    # since a step cannot be judged against an interval that does not exist.
    if (n != 6 || H[2] == "inf" || M[2] == "inf" || T[2] == "inf") {
      print "unbounded benchci established no interval for one peak window, which §5 rule 5 reads as contention rather than as a rate: too wide to bound is a failure to measure"
      exit
    }
    # sec/op RISING is the rate FALLING, so a decline is a positive delta here.
    nd = 0
    for (i = 1; i <= 2; i++) {
      a  = (i == 1) ? H[1] : M[1]; ca = (i == 1) ? H[2] : M[2]
      b  = (i == 1) ? M[1] : T[1]; cb = (i == 1) ? M[2] : T[2]
      d[i] = b / a - 1
      f[i] = (1 + ca) / (1 - cb) - 1
      if (d[i] > f[i]) nd++
      s = s sprintf("%s%s %+.4f%% against a %.4f%% floor", (i == 1 ? "" : ", "), \
                    (i == 1 ? "head->middle" : "middle->tail"), d[i] * 100, f[i] * 100)
    }
    if (nd == 2) {
      printf "declining the peak series declines across both steps and both clear the floor their own intervals set (sec/op %s), which §5 rule 5 names as throttling: the sweep between these windows ran on a clock that was falling, so its rates are not this host'\''s\n", s
      exit
    }
    # TWO NON-DECLINING SHAPES, NOT ONE, and the verdict is the same for both while
    # the ground is not (#6, 2026-08-20). This printf was unconditional and said "the
    # windows are ties" at nd == 1 too, which on keel-gnr produced "1 of 2 adjacent
    # steps resolves a decline ... so the windows are ties" -- a sentence whose own
    # count denies its conclusion. One resolved step is not a tie; it is half an
    # order, and rule 5 passes it because rule 5 names a MONOTONE decline. A healthy
    # host never shows this: zen4 and zen5 both took nd == 0.
    if (nd == 0) {
      printf "stable the peak series is bounded in every window and not ordered: neither of 2 adjacent steps resolves a decline above the resolution-plus-jitter floor its own intervals set (sec/op %s), so the windows are ties and a tie is not an order (#6, 2026-08-20)\n", s
      exit
    }
    printf "stable the peak series is bounded in every window and declines in one step but not across both: 1 of 2 adjacent steps resolves a decline above the resolution-plus-jitter floor its own intervals set (sec/op %s), so the series is a non-monotone excursion and not the monotone decline §5 rule 5 names as throttling -- the resolved step is stated above rather than called a tie (#6, 2026-08-20)\n", s
  }'
}
