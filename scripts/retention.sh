#!/usr/bin/env bash
# Retention: where the blocked Sgemm's time goes that its microkernel's does not.
#
# The measurement for issue #26 — the blocked nest keeps ~90% of its own dispatched
# microkernel's throughput on both Zen hosts and ~77% on janus (Skylake-X) — and
# nothing else. This is NOT a gate: it certifies nothing, it changes no criterion,
# and it exits 0 whatever it measures. Its product is a table.
#
# Three modes:
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
#                        Any axis can be replaced with KEEL_BLOCKING_KC / _MC / _NC
#                        (comma-separated), which is how a fine scan around a
#                        suspected point defect runs the same benchmark on a
#                        different grid — janus at 2x32 has one at kc=384, and a
#                        5-point coarse grid can alias a sharp associativity effect
#                        into fiction in either direction. A fine scan is small
#                        enough to afford KEEL_BENCH_COUNT=10, and the header then
#                        says so instead of telling the reader to re-measure.
#   feed                 BenchmarkFeed: the per-call decomposition resolved against
#                        KC, which is what the sweep left open (issue #48). The
#                        sweep found that janus's per-call penalty is not a constant
#                        — it RISES with KC — so something in the nest scales with a
#                        call's own duration rather than with the number of calls.
#                        Two candidates, and both have KC-independent totals, so
#                        size cannot tell them apart: real C traffic is flat in KC
#                        per call, panel feed is not. Six arms, each one variable
#                        from the last, all making the identical call multiset, so
#                        the columns are differences of measured quantities instead
#                        of derivatives of a fit.
#
# The three modes have deliberately different methodologies and say so in their
# output. decompose and feed run the standard gate methodology (scripts/bench.sh:
# -count=10 -benchtime=1s, benchstat medians, ratios net of CI) because its numbers
# are meant to be quoted. sweep runs -count=5 by default over 72 points per shape
# and is EXPLORATORY: it exists to find which way the surface tilts, and any point
# it nominates has to be re-measured under the full methodology before it becomes a
# default. A 72-point grid at count=10 is half an hour per host, and the first
# question is only whether the parameters matter at all.
#
# GOMAXPROCS=1 in all three, as every single-thread measurement in this repo does: P5's
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

# pct FRACTION — a fraction as a percentage, or a dash when it was not measured.
pct() { [[ -n "$1" ]] && awk -v r="$1" 'BEGIN{printf "%.1f%%", r*100}' || printf -- '-'; }

# rows_per_bench moved to scripts/bench.sh, beside the KEEL_BENCH_COUNT it reads back.

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

# ns_call NAME CSV — one arm's per-call cost in nanoseconds, or empty.
#
# The unit asked for is sec/call, not the ns/call the benchmark prints: benchfmt
# rewrites any ns/<x> metric to sec/<x> and divides by 1e9 before benchstat ever
# sees it. Asking for "ns/call" here would find nothing, silently, because
# bench_stat prints nothing for a unit that is absent — the same trap its own doc
# describes for B/s. So this reads the unit benchstat has and converts back.
ns_call() { bench_stat "$1" "$2" sec/call | awk '{ if ($1 != "") printf "%.2f", $1 * 1e9 }'; }

# ci_call NAME CSV — the same arm's interval as a percentage, or "inf".
#
# benchstat rounds this to a whole percent before printing it, so the value read
# back is not the interval but a bucket containing it: "0" means "under 0.5%",
# which is why nothing here treats a 0 as an exact median (T21). ci_floor_ns turns
# the bucket into the nanosecond bound it actually supports.
ci_call() {
  bench_stat "$1" "$2" sec/call |
    awk '{ if ($2 == "inf") printf "inf"; else printf "%.1f", $2 * 100 }'
}

# ci_floor_ns NS CI_PERCENT — the widest interval, in ns, consistent with what
# benchstat printed. A reported p% stands for a true interval below (p+0.5)%,
# since p is rounded, so this is an upper bound and errs toward calling a step
# unresolved. That is the safe direction for a decomposition: it can lose a real
# term to caution, but it cannot manufacture one out of rounding.
ci_floor_ns() {
  [[ "$2" == inf ]] && { printf 'inf'; return; }
  awk -v v="$1" -v c="$2" 'BEGIN{ printf "%.2f", (c + 0.5) / 100 * v }'
}

# feed_calls LOG NAME — the exact kernel-call count at one point, or "?".
#
# Read out of the keel-feed-panels marker, which the benchmark prints from the same
# callsPerDepth() sum it divides its own ns/call by. Not recomputed here: two
# implementations of one count is how a totals table comes to disagree with the
# per-call column it was derived from. A log written before the marker existed gives
# "?", and the caller must then print no totals rather than a plausible count — the
# #49 rule, that an instrument may not measure a grid other than the one it reports.
feed_calls() {
  awk -v want="name=$2 " '
    index($0, "keel-feed-panels:") == 1 && index($0, want) {
      for (i = 1; i <= NF; i++) if (index($i, "calls=") == 1) { sub(/calls=/, "", $i); print $i; exit }
    }' "$1" | head -1
}

# feed_host HOST CSV LOG — which term curves with KC (issue #48).
#
# Six arms per (shape, KC), each one variable from the one before it, all making the
# identical call multiset at identical depths — so per-call cost is a measurement
# divided by an exact count, and the step between two arms is a difference of two
# measured quantities rather than the derivative of a line fitted through four grid
# points. See BenchmarkFeed's doc for what each arm holds and what it does not.
#
# The steps are point estimates with no interval, being differences of medians. Each
# arm's own median has one, and the widest on each row is printed next to it: a step
# narrower than that is not resolved by this run and must not be read as a term.
feed_host() {
  local host="$1" csv="$2" log="$3" case_ kc arm rows worst ci floor f calls
  local -a kcs vals
  rows="$BINDIR/feedrows"
  while read -r case_; do
    mapfile -t kcs < <(awk -v c="$case_" -v q="/kc=" '
      index($1, "Benchmark" c q) == 1 {
        n = $1; sub(/-[0-9]+$/, "", n); sub(/.*\/kc=/, "", n); sub(/\/.*/, "", n); print n
      }' "$log" | sort -n -u)
    if [[ ${#kcs[@]} -eq 0 ]]; then
      warn "[$host] $case_: no kc points ran, so there is no KC axis to read"
      continue
    fi
    : >"$rows"
    for kc in "${kcs[@]}"; do
      if ! bench_expect "$log" "$csv" sec/call \
           "$case_/kc=$kc/kernel-calls" "$case_/kc=$kc/loops" "$case_/kc=$kc/cold-c" \
           "$case_/kc=$kc/cold-panels" "$case_/kc=$kc/nest-no-pack" "$case_/kc=$kc/full" \
           >"$BINDIR/miss"; then
        warn "[$host] $case_/kc=$kc: incomplete —$(tr '\n' ' ' <"$BINDIR/miss")"
        continue
      fi
      vals=()
      worst=0
      floor=0
      for arm in kernel-calls loops cold-c cold-panels nest-no-pack full; do
        vals+=("$(ns_call "$case_/kc=$kc/$arm" "$csv")")
        ci="$(ci_call "$case_/kc=$kc/$arm" "$csv")"
        # The floor is per arm, not the widest percent applied to the largest arm:
        # a percentage and the quantity it is a percentage of belong together.
        f="$(ci_floor_ns "${vals[-1]}" "$ci")"
        if [[ "$ci" == inf ]]; then worst=inf; floor=inf; fi
        [[ "$worst" != inf ]] && worst="$(awk -v a="$worst" -v b="$ci" 'BEGIN{print (b>a)?b:a}')"
        [[ "$floor" != inf ]] && floor="$(awk -v a="$floor" -v b="$f" 'BEGIN{print (b>a)?b:a}')"
      done
      calls="$(feed_calls "$log" "${case_#Feed/}/kc=$kc")"
      printf '%s %s %s %s %s\n' "$kc" "${vals[*]}" "$worst" "$floor" "${calls:-?}" >>"$rows"
    done
    [[ -s "$rows" ]] || { warn "[$host] $case_: no complete kc point"; continue; }
    printf '  %s\n' "$case_"
    info "per-call cost of each arm, ns/call. Every arm makes the same calls at the same"
    info "depths. worst-ci is the widest of the six arms' intervals; floor is that same"
    info "bound in ns, and it is what a step has to clear to be a term rather than noise."
    info "benchstat rounds the percent to an integer, so a printed 0% only says 'under"
    info "0.5%' and the floor takes it as (p+0.5)% of the arm it belongs to (T21)."
    printf '        %-5s %9s %9s %9s %9s %9s %9s %9s %8s\n' \
      kc calls loops cold-c cold-pan no-pack full worst-ci floor
    awk '{ printf "        %-5s %9.2f %9.2f %9.2f %9.2f %9.2f %9.2f %8s%% %8s\n",
             $1, $2, $3, $4, $5, $6, $7, $8, $9 }' "$rows" | sed 's/inf%/     inf/'
    info ""
    info "the single-variable steps between them, each in ns per call. A * marks a step"
    info "smaller than its own row's floor: measured, but not resolved by this run."
    printf '        %-5s %12s %12s %12s %10s %10s %12s\n' \
      kc macro-loops C-traffic panel-feed rest pack "total"
    # rest = no-pack - loops - (cold-c - loops) - (cold-panels - loops), i.e. what
    # the nest costs that none of the three isolated steps accounts for: the beta
    # pass, the fringe branch, the mask checks, and any interaction between C
    # traffic and panel feed that measuring them one at a time cannot see. The six
    # columns sum to full - kernel-calls by construction, which is the point: a
    # decomposition whose parts do not add up to the whole is hiding a term.
    awk 'function m(x, f) {
           if (f == "inf") return sprintf("%+.2f*", x)
           return sprintf("%+.2f%s", x, ((x < 0 ? -x : x) < f ? "*" : " "))
         }
         { printf "        %-5s %12s %12s %12s %10s %10s %12s\n",
             $1, m($3-$2, $9), m($4-$3, $9), m($5-$3, $9),
             m($6-$4-$5+$3, $9), m($7-$6, $9), m($7-$2, $9) }' "$rows"
    info ""
    info "which term curves (the question #48 exists to ask):"
    feed_curve "$rows" 'C traffic ' 4 3 \
      "predicted FLAT. Total C traffic is m*n*4*ceil(k/KC), which scales exactly like" \
      "the call count, so per call it is a constant and adds nothing to the KC slope."
    feed_curve "$rows" 'panel feed' 5 3 \
      "predicted to CURVE, and this is the surviving form of the feed hypothesis." \
      "Total panel bytes are (m/MR)(n/NR)*k*(MR+NR)*4 — KC-INDEPENDENT — so a" \
      "constant-rate per-byte tax adds nothing either. Only the level those bytes" \
      "come from can, which is a threshold, hence a curve rather than a slope." \
      "READ THIS AGAINST reused-panel-bytes IN THE keel-feed-panels LINES ABOVE, and" \
      "against this host's L1d in the provenance line: where the reused pair does not" \
      "fit L1d, both arms feed from L2 and this step is a difference of locality" \
      "within one level, not of level. It is a measurement of feed cost only at the" \
      "KC where the premise holds."
    feed_curve "$rows" 'pack      ' 7 6 \
      "KC-proportional by construction: total pack work is KC-independent and the" \
      "call count falls as 1/KC, so this column must rise roughly linearly and its" \
      "slope means nothing. Its INTERCEPT is the finding — pack's per-row loop setup" \
      "count is ceil(k/KC)*m, which scales like the call count, and that constant is" \
      "the term separating the sweep's slope from the decomposition's per-call cost."
    feed_fit "$rows" 'pack      ' 7 6
    feed_rest "$rows" 'rest      '
    feed_totals "$rows" "$host" "$case_"
  done < <(awk '/^BenchmarkFeed\// {
      n = $1; sub(/-[0-9]+$/, "", n); sub(/^Benchmark/, "", n); sub(/\/kc=.*/, "", n); print n
    }' "$log" | sort -u)
}

# feed_totals ROWS HOST CASE — the same steps as whole-GEMM totals.
#
# A per-call cost says how expensive a term is; it does not say whether the term
# matters, because the number of calls changes with KC by a factor of four across
# this grid. Multiplying by the point's own exact call count answers the second
# question, and it also makes the analytic predictions checkable in a way the
# per-call columns cannot:
#
#	C traffic   total is m*n*4*ceil(k/KC), so it must FALL like the call count
#	pack        total pack work is KC-independent, so it must be FLAT
#	panel feed  total panel bytes are KC-independent too, so a flat total means a
#	            constant-rate tax and a rising one means the bytes are coming from
#	            a colder level — the #48 question again, now in milliseconds
#
# The arithmetic is per-call ns x calls, nothing fitted. Where the call count is
# unknown (a log from before the keel-feed-panels marker) this prints nothing at all
# rather than assuming (m/MR)(n/NR)ceil(k/KC) from parameters it cannot see.
feed_totals() {
  local rows="$1" host="$2" case_="$3"
  if grep -q ' ?$' "$rows"; then
    warn "[$host] $case_: no keel-feed-panels marker, so the exact call count is not in" \
         "this log and no whole-GEMM totals are printed. Re-measure to get them; a" \
         "count reconstructed here could disagree with the one the ns/call column" \
         "was divided by, and then the two tables would describe different runs."
    return
  fi
  info ""
  info "the same steps as whole-GEMM totals: this point's per-call cost times its own"
  info "exact call count, in ms of one GEMM, with each term's share of full. Nothing is"
  info "fitted; this is the per-call table times an integer. A * still marks a step"
  info "below its row's noise floor, and a share computed from one means little."
  printf '        %-5s %10s %14s %14s %14s %11s\n' \
    kc calls C-traffic panel-feed pack full
  awk 'function ms(x, c) { return x * c / 1e6 }
       function cell(x, c, tot, f) {
         return sprintf("%.2f%s (%.1f%%)", ms(x, c),
                        (f == "inf" || (x < 0 ? -x : x) < f) ? "*" : "", 100 * x / tot)
       }
       { printf "        %-5s %10d %14s %14s %14s %11.1f\n",
           $1, $10, cell($4-$3, $10, $7, $9), cell($5-$3, $10, $7, $9),
           cell($7-$6, $10, $7, $9), ms($7, $10) }' "$rows"
  # The predicted C ratio is not the constant 4: it is ceil(k/KC1)/ceil(k/KCn), and
  # since calls = (m/MR)(n/NR)*ceil(k/KC) with a KC-independent prefactor, the call
  # count ratio IS that prediction — measured off this run, not assumed from a k this
  # script never sees.
  awk 'function ms(x, cc) { return x * cc / 1e6 }
       { n++
         cn[n] = $10; kcs[n] = $1
         c[n] = ms($4-$3, $10); p[n] = ms($5-$3, $10); q[n] = ms($7-$6, $10)
         # A ratio between endpoints is only a number if both endpoints are one: a
         # negative or unresolved end makes c[1]/c[n] an artefact of noise, and the
         # honest output there is the two values and the reason, not a quotient.
         cbad[n] = (c[n] <= 0) || ($9 != "inf" && (($4-$3 < 0 ? $3-$4 : $4-$3) < $9))
         if (n == 1 || p[n] > pworst) { pworst = p[n]; pat = $1; pshare = 100*($5-$3)/$7 }
         if (n == 1 || q[n] < qlo) qlo = q[n]
         if (n == 1 || q[n] > qhi) qhi = q[n] }
       END {
         if (cbad[1] || cbad[n]) {
           printf "          C total     %.2f -> %.2f ms; NO RATIO: an endpoint is negative or below\n", c[1], c[n]
           printf "                      its noise floor, so the quotient would be an artefact. The\n"
           printf "                      predicted fall is x%.2f, the call-count ratio this run reported.\n",
             (cn[n] != 0) ? cn[1]/cn[n] : 0
         } else {
           printf "          C total     %.2f -> %.2f ms, x%.2f over the grid; predicted x%.2f,\n",
             c[1], c[n], c[1]/c[n], (cn[n] != 0) ? cn[1]/cn[n] : 0
           printf "                      which is the call-count ratio this run reported, not a constant.\n"
         }
         printf "          pack total  %.2f -> %.2f ms, spread %.2f ms over the grid;\n", q[1], q[n], qhi - qlo
         printf "                      predicted FLAT, and a spread here is the residual model error.\n"
         printf "          feed total  %.2f -> %.2f ms, worst %.2f ms at kc=%s (%.1f%% of full);\n",
           p[1], p[n], pworst, pat, pshare
         printf "                      the KC to distrust is that one, not necessarily the last.\n"
       }' "$rows"
}

# feed_rest ROWS LABEL — the residual column, and what its SIGN means.
#
# rest = no-pack − loops − (cold-c − loops) − (cold-panels − loops) is the only column
# whose sign carries information beyond its size. It holds the nest's remaining real
# work — beta pass, fringe branch, mask checks — which is positive, plus the
# INTERACTION between C traffic and panel feed, which has either sign:
#
#	rest ≈ 0 or small positive   the two steps are additive, so each isolated step is
#	                             a fair estimate of its own term
#	rest large NEGATIVE          sub-additive: the two traffic streams overlap in
#	                             time, so measuring them one at a time overstates
#	                             both, and the isolated steps are upper bounds
#	rest large POSITIVE          super-additive: together they cost more than apart,
#	                             e.g. contending for the same fill buffers, and the
#	                             isolated steps are lower bounds
#
# It cannot go through feed_curve because it is not one column minus another. It is
# printed last because it says how far the three above it can be trusted.
feed_rest() {
  local rows="$1" label="$2"
  awk -v label="$label" '
    { n++
      v = $6 - $4 - $5 + $3
      s = s sprintf("%s%+.2f%s", (n > 1 ? " -> " : ""), v,
                    ($9 == "inf" || (v < 0 ? -v : v) < $9) ? "*" : "")
      if (v < -worstneg) worstneg = -v
      if (v > worstpos) worstpos = v }
    END {
      printf "          %s  %s\n", label, s
      if (worstneg > worstpos)
        printf "          %*s  SUB-ADDITIVE by up to %.2f ns: C traffic and panel feed overlap in\n          %*s  time, so each isolated step above is an UPPER BOUND on its term.\n", \
               length(label), "", worstneg, length(label), ""
      else if (worstpos > 0)
        printf "          %*s  up to %+.2f ns of the nest that no isolated step accounts for: the\n          %*s  beta pass, the fringe branch, the mask checks, and any super-additive\n          %*s  interaction, which would make the steps above LOWER bounds.\n", \
               length(label), "", worstpos, length(label), "", length(label), ""
    }' "$rows"
}

# feed_fit ROWS LABEL MINUEND SUBTRAHEND — a least-squares line through one step
# against KC, printed as slope, intercept and worst residual.
#
# Only pack gets this, and only for its intercept. Pack per call is expected to be
# α·KC + β: α is per-byte work and says nothing (total pack work is KC-independent,
# so α is an artifact of dividing a constant total by a call count that falls as
# 1/KC), while β is the per-call constant — pack's per-row loop setup, whose count
# ceil(k/KC)·m scales like the call count. β is the term that separates the sweep's
# per-call slope from this decomposition's per-call costs, which is why #26 promised
# it as a number.
#
# It is a fit through four medians, not a measurement, and it is printed as one: the
# worst residual is shown so the reader can see whether a line describes these points
# at all, and an intercept smaller than the rows' own noise floors is called
# indistinguishable from zero rather than quoted as a small number.
feed_fit() {
  local rows="$1" label="$2" a="$3" b="$4"
  awk -v label="$label" -v a="$a" -v b="$b" '
    { n++
      x = $1 + 0; y = $a - $b
      xs[n] = x; ys[n] = y
      sx += x; sy += y; sxx += x * x; sxy += x * y
      if ($9 != "inf") { fs += $9; fn++ } }
    END {
      d = n * sxx - sx * sx
      if (n < 3 || d == 0) {
        printf "          %*s  too few KC points for a fit\n", length(label), ""
        exit
      }
      m = (n * sxy - sx * sy) / d
      c = (sy - m * sx) / n
      for (i = 1; i <= n; i++) {
        r = ys[i] - (m * xs[i] + c)
        if (r < 0) r = -r
        if (r > worst) worst = r
      }
      floor = (fn > 0) ? fs / fn : 0
      printf "          %*s  least squares: %+.4f ns per KC unit, intercept %+.2f ns", \
             length(label), "", m, c
      printf ", worst residual %.2f ns\n", worst
      printf "          %*s  the intercept is the bridge term, and it is ", length(label), ""
      if (fn == 0)
        printf "unbounded here (no row had a\n          %*s  usable interval).\n", length(label), ""
      else if ((c < 0 ? -c : c) < floor)
        printf "INDISTINGUISHABLE FROM ZERO: |%.2f| is\n          %*s  under the mean row floor of %.2f ns, so pack carries no measurable\n          %*s  per-call constant on this host at this shape — the whole column is\n          %*s  per-byte work divided by a shrinking call count.\n", \
               c, length(label), "", floor, length(label), "", length(label), ""
      else
        printf "%+.2f ns per call against a mean row floor\n          %*s  of %.2f ns, so it is resolved and is the number #26 asked for.\n", \
               c, length(label), "", floor
    }' "$rows"
}

# feed_curve ROWS LABEL MINUEND SUBTRAHEND NOTE... — one step's values across the KC
# grid, with its spread and what was predicted of it. The step is column MINUEND
# minus column SUBTRAHEND of the row file, both 1-based awk field numbers.
#
# The spread is what answers #48. A term that is flat in KC contributes a constant to
# per-call cost and therefore nothing to the sweep's KC slope; a term that grows with
# KC is the one that slope was measuring. Since both candidates have KC-independent
# *totals*, size cannot separate them and shape has to.
#
# No verdict is printed. The prediction goes next to the numbers and whether they
# match is the reader's call — this script certifies nothing.
feed_curve() {
  local rows="$1" label="$2" a="$3" b="$4" note pad
  shift 4
  awk -v label="$label" -v a="$a" -v b="$b" '
    { v[++n] = $a - $b; k[n] = $1; f[n] = $9
      if (n == 1 || v[n] < lo) lo = v[n]
      if (n == 1 || v[n] > hi) hi = v[n]
      if (f[n] == "inf" || (v[n] < 0 ? -v[n] : v[n]) < f[n]) under++
      if (v[n] < 0) neg++
      sum += v[n] }
    END {
      s = ""
      for (i = 1; i <= n; i++)
        s = s sprintf("%s%+.2f%s", (i > 1 ? " -> " : ""), v[i],
                      (f[i] == "inf" || (v[i] < 0 ? -v[i] : v[i]) < f[i]) ? "*" : "")
      m = sum / n
      printf "          %s  %s\n", label, s
      printf "          %*s  spread %.2f ns over kc=%s..%s", length(label), "", hi - lo, k[1], k[n]
      if (m != 0) printf ", %.0f%% of its own mean", 100 * (hi - lo) / (m < 0 ? -m : m)
      printf "\n"
    }' "$rows"
  pad="$(printf '%*s' "${#label}" '')"
  for note in "$@"; do
    printf '          %s  %s\n' "$pad" "$note"
  done
  # The prediction is printed before these because they qualify how far the numbers
  # can be read against it. A spread computed over unresolved points is a spread of
  # noise, and a negative cost is not a cost: both say the reading stops here, and
  # both are stated rather than left for a reader to notice.
  awk -v pad="$pad" -v a="$a" -v b="$b" '
    { n++
      v = $a - $b
      if ($9 == "inf" || (v < 0 ? -v : v) < $9) under++
      if (v < 0) neg++ }
    END {
      if (under > 0) {
        printf "          %s  %d of %d points are below their own row%cs floor, so the spread\n", pad, under, n, 39
        printf "          %s  above is partly this run%cs noise and not only this term%cs shape.\n", pad, 39, 39
      }
      if (neg > 0) {
        printf "          %s  %d point(s) NEGATIVE. A cost cannot be negative, so at those KC the\n", pad, neg
        printf "          %s  two arms differ by something other than the variable named, and\n", pad
        printf "          %s  nothing about this term is being measured there.\n", pad
      }
    }' "$rows"
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
    decompose|sweep|feed) ;;
    *) echo "usage: $0 [decompose|sweep|feed]" >&2; exit 2 ;;
  esac

  BINDIR="$(mktemp -d)"
  trap 'rm -rf "$BINDIR"' EXIT
  local BIN="$BINDIR/block.test" LOG="$BINDIR/log" CSV="$BINDIR/csv"

  echo "== retention ($MODE) — issue #26. Not a gate: this certifies nothing. =="
  echo

  remote_require_hosts

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
  [[ "$MODE" == feed ]] && FILTER='BenchmarkFeed'

  # sshd strips arbitrary env, so anything the benchmark reads has to be handed to
  # the remote command explicitly. KEEL_BLOCKING_{KC,MC,NC} replace one axis of
  # BenchmarkBlocking's grid, which is how a fine scan around a suspected point
  # defect runs the same benchmark rather than a new one. Unset axes are not
  # forwarded, so the binary's own defaults stay the defaults.
  local RENV="GOMAXPROCS=1" axis
  for axis in KEEL_BLOCKING_KC KEEL_BLOCKING_MC KEEL_BLOCKING_NC; do
    [[ -n "${!axis:-}" ]] && RENV+=" $axis=${!axis}"
  done

  echo "methodology: -count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, GOMAXPROCS=1, benchstat medians"
  if [[ "$MODE" == sweep ]]; then
    # The distinction the old unconditional line lost: a count of 10 is the standard
    # methodology, so telling a reader of a 10-count log to "re-measure at 10" reads
    # as though this run were not it. What stays true at any count is that a grid
    # point is not a default — that decision goes through #24's kern.Class, not
    # through a good row in a sweep.
    if [[ "$KEEL_BENCH_COUNT" -ge 10 ]]; then
      echo "             full methodology, and still not a tuner: a winning point becomes a default"
      echo "             only through kern.Class (#24), never by being the top row here"
    else
      echo "             EXPLORATORY: re-measure any nominated point at -count=10 before it becomes a default"
    fi
    # What the grid was *asked* to be. What it actually was is readable off the
    # point names in the ranking below, which is the half that cannot be shadowed.
    [[ "$RENV" != "GOMAXPROCS=1" ]] && echo "             grid override requested: ${RENV#GOMAXPROCS=1 }"
  fi
  echo

  local host
  while read -r host; do
    [[ -n "$host" ]] || continue
    remote_host_header "$host" || continue
    if ! KEEL_REMOTE_ENV="$RENV" remote_exec "$host" "$BIN" "${BFLAGS[@]}" \
         -test.bench="$FILTER" >"$LOG" 2>&1; then
      warn "[$host] the benchmark run failed; nothing is reported for it"
      sed 's/^/        /' "$LOG" | tail -20
      continue
    fi
    # What arrived, not what was asked for: -count is a request, and the header
    # above prints the request. This line is the log counting itself (#49).
    info "samples this host produced: $(rows_per_bench "$LOG")"
    # The plan markers next: they say which blocks the parts were measured over.
    # keel-feed-panels is the same idea for the arms' premise: the reused panel pair
    # is L1-resident only while (MR+NR)*kk*4 fits, so the size is printed and the
    # panel-feed column below is read against it rather than against a claim.
    grep -E '^keel-(nest-plan|feed-panels):' "$LOG" | sed 's/^/        /' || true
    bench_csv "$LOG" >"$CSV" 2>"$BINDIR/bserr" || true
    [[ -s "$BINDIR/bserr" ]] && sed 's/^/        benchci: /' "$BINDIR/bserr"
    case "$MODE" in
      decompose) decompose_host "$host" "$CSV" "$LOG" ;;
      sweep)     sweep_host "$host" "$CSV" "$LOG" ;;
      feed)      feed_host "$host" "$CSV" "$LOG" ;;
    esac
    echo
  done <<<"$HOSTS"

  echo "done. Numbers above are this host set at this commit; nothing here is a criterion."
}

main "$@"
