#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Negative and positive controls for P2's throughput verdict (scripts/roofline.sh).
#
# The amendment on issue #19 replaces a flat floor with a roofline-relative one on
# issue-bound hosts. That is a re-expression of the same intent rather than a hole
# only if three things hold: a host cannot classify itself issue-bound by being
# slow; a kernel cannot raise its own roofline by emitting more instructions; and
# the roofline test is not implied by the classifier that admits you to it. Those
# are the three claims DESIGN.md §4/P2 makes, so they are tested here rather than
# asserted. The third one caught a real defect — see roofline.sh, INDEPENDENCE.
#
# Fixtures feed measured (fraction, insns/FMA) pairs, never pre-reduced spreads, so
# a fixture cannot describe a host that could not exist. An earlier draft of this
# file did exactly that, and its "issue-bound but underperforming" control was
# consequently fake.
#
# Run: bash scripts/roofline-test.sh   (part of gate-p2, and standalone)
set -euo pipefail

# Everything below is a function definition and the last line of the file is
# `main "$@"`. Bash reads a script incrementally as it executes it, so a script
# that does its work at the top level can be corrupted by an edit that lands
# mid-run: the parser resumes at a byte offset that now holds different text.
# Defining everything before anything runs forces one whole-file parse before the
# first check runs, which makes the instrument immune instead of leaving "never
# edit a running instrument" as a rule someone has to remember (#51).
#
# The fixture constants and FAILED are assigned in main without `local`, so the
# three check functions see them however they are reached. They are constants of
# the file, not state threaded between calls.

# check NAME EXPECT_CLASS EXPECT_RESULT best_lo[:best_hi] best_ipf ceiling_mix...
#
# The shape under test's reading is `lo` for a point fixture and `lo:hi` for one
# that carries an interval; a ceiling mix is `f:I` or `f:I:f_lo:f_hi`. Absent
# bounds mean a zero-width interval, which is what every fixture written before
# the #86 amendment asserted, so all of those keep their exact verdicts — and that
# they still pass unchanged is the regression check on the third state.
check() {
  local name="$1" xclass="$2" xresult="$3"; shift 3
  local best="$1" best_ipf="$2"; shift 2
  local best_lo="${best%%:*}" best_hi="${best#*:}"
  [[ "$best_hi" == "$best" ]] && best_hi="$best_lo"
  local out; out="$(throughput_verdict "$best_lo" "$best_hi" "$best_ipf" \
                      "$PF" "$RF" "$CM" "$MM" "$SB" "$SL" "$@")"
  local class cspread mspread roof attain result why cslo cshi attainhi
  read -r class cspread mspread roof attain result why cslo cshi attainhi <<<"$out"
  if [[ "$class" == "$xclass" && "$result" == "$xresult" ]]; then
    printf '  ok    %-53s %-13s/%-10s cspread=%.3fx [%.3f,%.3f] mspread=%.3fx roof=%5.1f%% attain=%6.1f%%..%.1f%% why=%s\n' \
      "$name" "$class" "$result" "$cspread" "$cslo" "$cshi" "$mspread" \
      "$(awk -v r="$roof" 'BEGIN{print r*100}')" \
      "$(awk -v a="$attain" 'BEGIN{print a*100}')" \
      "$(awk -v a="$attainhi" 'BEGIN{print a*100}')" "$why"
  else
    printf '  FAIL  %-53s got %s/%s (why=%s), want %s/%s\n' \
      "$name" "$class" "$result" "$why" "$xclass" "$xresult"
    FAILED=1
  fi
}

# checkd NAME EXPECT_SOURCE EXPECT_WHY EXPECT_DENOM class pmax i_active ob peak
checkd() {
  local name="$1" xsrc="$2" xwhy="$3" xdenom="$4"; shift 4
  local class="$1" pmax="$2" iact="$3" ob="$4" peak="$5"
  local out; out="$(p3_denominator "$class" "$pmax" "$iact" "$SB" "$SL" "$ob" "$peak")"
  local denom src roof why
  read -r denom src roof why <<<"$out"
  # Denominators are GFLOP/s, so 0.05 is far tighter than any measurement.
  if [[ "$src" == "$xsrc" && "$why" == "$xwhy" ]] &&
     awk -v a="$denom" -v b="$xdenom" 'BEGIN{exit !(a-b < 0.05 && b-a < 0.05)}'; then
    printf '  ok    %-53s %-8s denom=%7.2f roof=%5.1f%% why=%s\n' \
      "$name" "$src" "$denom" "$(awk -v r="$roof" 'BEGIN{print r*100}')" "$why"
  else
    printf '  FAIL  %-53s got %s/%s denom=%s, want %s/%s denom=%s\n' \
      "$name" "$src" "$why" "$denom" "$xsrc" "$xwhy" "$xdenom"
    FAILED=1
  fi
}

# checkr NAME EXPECT src rlo pklo roof   (EXPECT "" means "no bound at all")
checkr() {
  local name="$1" want="$2"; shift 2
  local got; got="$(p3_ratio_lo "$1" "$2" "$3" "$4")"
  if [[ -z "$want" && -z "$got" ]] ||
     { [[ -n "$want" && -n "$got" ]] &&
       awk -v a="$got" -v b="$want" 'BEGIN{exit !(a-b < 0.0005 && b-a < 0.0005)}'; }; then
    printf '  ok    %-53s %s\n' "$name" "${got:-<no bound, treated as a failure to measure>}"
  else
    printf '  FAIL  %-53s got "%s", want "%s"\n' "$name" "$got" "$want"
    FAILED=1
  fi
}

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/roofline.sh
  source scripts/roofline.sh

  # The gate constants, so the fixtures are judged by the shipped thresholds.
  PF=0.55; RF=0.90; CM=1.10; MM=1.25; SB=4.438; SL=1.05
  PEAK="1.0:2.25"   # register-only peak kernel: f = 1 by definition, audited I

  FAILED=0

  echo "== roofline verdict: controls =="
  echo
  echo "-- the three real hosts, 2026-08-11 measurements --"
  echo "   (shape under test is each host's best shape; ceiling set is the other"
  echo "    shipped shape plus the peak kernel)"
  # vesta 2x32 f=.924 I=4.625 -> 4.274 vs peak 2.250: nowhere near a front-end wall.
  check "vesta (Zen 4): wide front end, 4x32 wins"          fma   pass \
        0.966 6.250   0.924:4.625 "$PEAK"
  # janus 4x32 .352*6.25=2.200 vs peak 2.250: two mixes 2.8x apart in I agree on
  # rate to 2.3%. That is a ceiling, and the 2x32 shape sits at 94.8% of it.
  check "janus (Skylake-X): ceiling mixes converge"         issue pass \
        0.461 4.625   0.352:6.250 "$PEAK"
  # antares 2x32 .531*4.625=2.456 vs peak 2.250 converge to 9.2% -- but the 4x32
  # shape retires .628*6.25=3.925, far above that ceiling, so it is not one.
  check "antares (Zen 5): beats its claimed ceiling"        fma   pass \
        0.628 6.250   0.531:4.625 "$PEAK"

  echo
  echo "-- negative controls: the ways this could have been a hole --"

  # 1. A kernel padded with dead instructions. Throughput falls in proportion, so
  #    the ceiling still looks like a ceiling -- but the shape is now far off the
  #    sweep best, so it is denied a roofline. 74 -> 114 insns, I=7.125, f 46.1% ->
  #    29.9%. Note it would otherwise PASS at 94.7% of its self-inflated roofline.
  check "padded kernel (+40 dead insns) cannot buy a roofline" issue refuse \
        0.299 7.125   0.352:6.250 "$PEAK"

  # 2. A merely slow kernel on a wide machine: the ceiling mixes diverge (5.6x), so
  #    it classifies FMA-bound and faces the full 55%, which it misses.
  check "slow kernel on a wide host is not excused"          fma   fail \
        0.300 4.625   0.900:6.250 "$PEAK"

  # 3. Issue-bound and genuinely leaving throughput on the table. Same ceiling as
  #    janus, but the shape under test retires .365*4.625=1.688 against a 2.250
  #    ceiling. This is the control the previous draft got wrong.
  check "issue-bound but only 75% of roofline still fails"   issue fail \
        0.365 4.625   0.352:6.250 "$PEAK"

  # 4. Convergence between two nearly identical mixes is not evidence. Rates agree
  #    to 5.9%, but insns/FMA spread is only 1.016x, so no ceiling is established.
  check "convergence across similar mixes proves nothing"    fma   fail \
        0.400 4.625   0.480:4.625 0.500:4.700

  # 5. Sandbagging an alternate shape to understate the ceiling does not work: the
  #    peak kernel is always in the set with f=1, so a slow alternate breaks
  #    convergence instead of lowering the roofline.
  check "sandbagged alternate breaks convergence, not roofline" fma fail \
        0.400 4.625   0.100:6.250 "$PEAK"

  # 6. Fewer than two ceiling mixes: nothing to converge with, so the flat floor.
  check "a single ceiling mix establishes nothing"           fma   fail \
        0.461 4.625   "$PEAK"

  # 7. The ratchet. Once the lowering folds broadcast+load into the FMA (T12/#20),
  #    I falls 4.625 -> 2.875 and the roofline RISES to 2.250/2.875 = 78.3%, so the
  #    required floor becomes 0.90*78.3% = 70.4% -- stricter than the flat 55% the
  #    amendment replaced. The amendment does not retire on the fix; it tightens.
  check "after the T12 fix, janus needs 70.4% and makes 74%" issue pass \
        0.740 2.875   0.560:3.875 "$PEAK"
  # ... and the ratchet has teeth: 60% would clear the flat 55% and still fails.
  check "after the T12 fix, 60% of peak now FAILS"          issue fail \
        0.600 2.875   0.560:3.875 "$PEAK"

  # 8. Boundary on the roofline floor. roof = 2.250/4.625 = 0.486486;
  #    0.90*roof = 0.437838.
  check "exactly 90.0% of roofline passes"                   issue pass \
        0.437838 4.625   0.352:6.250 "$PEAK"
  check "just under 90% of roofline fails"                   issue fail \
        0.437000 4.625   0.352:6.250 "$PEAK"

  # 9. Boundary on the shape guard: sweep best 4.438 * 1.05 = 4.6599. Tested at
  #    +-1e-3 rather than at the exact tie, because insns/FMA is a ratio of small
  #    integers -- 16 or 32 FMAs per body -- so realizable values are quantized to
  #    1/32 = 0.03125 and no measurement can land inside 1e-3 of the threshold.
  check "shape 4.6590 insns/FMA (+5.0%) gets a roofline"     issue pass \
        0.437 4.6590   0.352:6.250 "$PEAK"
  check "shape 4.7000 insns/FMA (+5.9%) is refused"          issue refuse \
        0.437 4.7000   0.352:6.250 "$PEAK"

  echo
  echo "-- the third state (#86): a class-selecting comparison the reading cannot decide --"
  echo "   (CLASS and RESULT are graded differently on purpose: a class straddle moves"
  echo "    four verdicts, a result miss moves one and keeps DESIGN.md §4's re-run)"

  # 20. THE INCIDENT, as close as the log supports. On 2026-08-15 janus's classify
  #     invocation read both microkernels ~7% low against the peak kernel in the same
  #     run; the spread crossed 1.10, the host reclassified fma-bound, and criterion 7
  #     switched denominators under an unchanged numerator. The exact f for 4x32 is NOT
  #     recoverable from that log -- gate-p3 discarded the spread it decided on, which
  #     is the reporting defect this commit also fixes -- but `why=diverge` bounds it:
  #     f*6.25 < 2.25/1.10, so f < 0.3273 against 0.352 on the two adjacent runs. At
  #     f=0.327 the point spread is 1.101, just over the bar, and the interval reaches
  #     1.068 on the other side of it. Two states used to share that reading.
  check "janus 2026-08-15: the spread cannot decide the class"  indeterminate unmeasured \
        0.411:0.443 4.625   0.327:6.250:0.317:0.337 "$PEAK"

  # 21. Real divergence is still divergence. Tight intervals on a 2.5x spread: the
  #     third state must not become a way for a wide front end to escape the flat
  #     floor, which is negative control 2 with the intervals switched on.
  check "a 2.5x spread with tight CIs still diverges"        fma   fail \
        0.300:0.305 4.625   0.900:6.250:0.890:0.910 "$PEAK"

  # 22. Real convergence is still convergence: janus as it healthily reads, with its
  #     measured intervals attached. The third state is carved out of the region where
  #     the two old outcomes disagreed with each other, not out of either of them.
  check "janus healthy, intervals attached: still issue/pass" issue pass \
        0.461:0.465 4.625   0.352:6.250:0.348:0.356 "$PEAK"

  # 23. It takes PASSes away too, which is what makes it a measurement rule and not
  #     leniency. Attainment 98.7% would sail past the 90% floor, but its interval
  #     crosses 1.0, so whether this host exceeds its own claimed ceiling -- the
  #     falsification test -- is undecided, and a host cannot be judged against a
  #     roofline it may have already beaten.
  check "attainment straddling the ceiling withholds a PASS"  indeterminate unmeasured \
        0.480:0.500 4.625   0.352:6.250 "$PEAK"

  # 24. Falsified beats undecided: antares clears its claimed ceiling by 53%, and an
  #     interval on the reading does not make that a close call.
  check "clearly falsified stays falsified, not indeterminate" fma  pass \
        0.600:0.620 6.250   0.531:4.625 "$PEAK"

  # 25. Where the bar is, checked from both sides. Tested at +-1e-3 of a 1.10 spread
  #     rather than at the exact tie, for the same reason as control 9: a tie between
  #     two doubles is unreachable in practice and pinning it would test the FPU. The
  #     lower end is held at 1.083 in both, so it is the upper end doing the work.
  check "spread interval reaching 1.099x: converged"          issue pass \
        0.500:0.510 4.625   0.393:6.250:0.390:0.395640 "$PEAK"
  check "spread interval reaching 1.101x: cannot decide"      indeterminate unmeasured \
        0.500:0.510 4.625   0.393:6.250:0.390:0.396360 "$PEAK"

  # 26. One mix wide enough to supply both ends of the spread. Its own interval spans
  #     1.35x, so the set cannot show a ceiling however well the point estimates line
  #     up; reporting indeterminate here is the honest reading of a mix that wide, not
  #     an artifact of taking the ends over the whole set.
  check "a single very wide mix establishes no ceiling"       indeterminate unmeasured \
        0.461:0.465 4.625   0.352:6.250:0.300:0.404 "$PEAK"

  # 27. The clamp at 1. Two heavily overlapping mixes give pmax_lo < pmin_hi, so the
  #     raw lower end of the spread is 0.967 -- a spread below 1 is not a thing, and
  #     without the clamp it would read as "converged by -3%". Clamped, this is what it
  #     should be: as converged as a measurement can show, and judged.
  check "overlapping intervals are converged, not undecided"  issue pass \
        0.461:0.465 4.625   0.352:6.250:0.340:0.364 0.98:2.250:0.9667:1.0

  echo
  echo "-- P3's denominator (#23): min(same-host OpenBLAS, roofline x measured peak) --"
  echo "   (the amendment may only ever LOWER a denominator, only on an issue-bound"
  echo "    host, and only for a shape inside the guard)"

  # 10. An FMA-bound host is untouched, whatever its shape or its peak. This is the
  #     property that keeps vesta and antares measuring the comparison unassisted.
  checkd "vesta (FMA-bound): plain OpenBLAS, no roofline"    openblas fma-bound 165.00 \
         fma 0 6.250 165.00 165.10
  checkd "FMA-bound host with a guard-clean shape: still plain" openblas fma-bound 175.00 \
         fma 0 4.625 175.00 216.90

  # 11. janus as it ships today: issue-bound, but Sgemm dispatches to 4x32 at 6.250
  #     insns/FMA, 41% above the sweep best, so the guard refuses it a roofline and
  #     the host faces plain OpenBLAS. This is issue #24's arithmetic.
  checkd "janus, Sgemm on 4x32 (6.250): guard refuses"       openblas shape 175.00 \
         issue 2.25 6.250 175.00 216.90

  # 12. The same host if Sgemm ran the guard-clean 2x32: roofline 2.250/4.625 =
  #     48.65% of a 216.9 peak = 105.52, well under OpenBLAS, so the cap binds.
  checkd "janus, Sgemm on 2x32 (4.625): roofline caps"       roofline issue-capped 105.52 \
         issue 2.25 4.625 175.00 216.90

  # 13. Self-retirement, both halves. Post-T12-fix I = 2.875 lifts the roofline to
  #     78.26%; against a 175 GFLOP/s reference the cap is still (just) the binding
  #     one, and against a 150 GFLOP/s reference the min() picks OpenBLAS and the
  #     amendment is simply gone. No expiry clause is involved in either.
  checkd "post-fix I=2.875 vs strong reference: cap still binds" roofline issue-capped 169.75 \
         issue 2.25 2.875 175.00 216.90
  checkd "post-fix I=2.875 vs weaker reference: retires itself" openblas reference 150.00 \
         issue 2.25 2.875 150.00 216.90

  # 14. The amendment is one-sided: a roofline above the reference cannot raise the
  #     bar. Same host, an absurdly high peak, and the denominator stays OpenBLAS.
  checkd "roofline above the reference never raises the bar"  openblas reference 100.00 \
         issue 2.25 4.625 100.00 900.00

  # 15. Missing inputs fail closed onto the unmodified criterion rather than onto a
  #     zero denominator, which would divide keel's rate into infinity.
  checkd "no peak measured: no cap, plain OpenBLAS"          openblas nopeak 175.00 \
         issue 2.25 4.625 175.00 0
  checkd "no audited insns/FMA: no cap, plain OpenBLAS"      openblas nopeak 175.00 \
         issue 2.25 0 175.00 216.90

  # 15b. An indeterminate class is neither branch (#86). It used to fall through the
  #      `class != "issue"` test and come out as a confident fma-bound, which is the
  #      strict direction -- but it produced a FAIL against a floor on the strength of
  #      a classification the run could not make. There is no denominator here.
  checkd "an indeterminate class yields no denominator"      indeterminate classindeterminate 0.00 \
         indeterminate 2.25 4.625 175.00 216.90

  echo
  echo "-- P3's ratio, net of CI, against whichever denominator was chosen --"

  # 16. The plain branch passes benchstat's own bound through untouched.
  checkr "openblas denominator: the bound is benchstat's" 0.641 openblas 0.641 0.311 0
  # 17. The roofline branch divides keel/peak by the roofline: janus's 31.1% net of CI
  #     against a 48.65% roofline is 63.9% of the cap. This is the arithmetic a wrong
  #     division direction would turn into 15.1%, or into 64.6%-looking nonsense.
  checkr "roofline denominator: pklo/roof"               0.6393 roofline 0.390 0.311 0.486486
  # 18. When the reference's interval is the wider one, keel/OpenBLAS is the tighter
  #     bound on the same ratio and is the one reported.
  checkr "the tighter of the two valid bounds wins"      0.700 roofline 0.700 0.311 0.486486
  # 19. Fail closed: no bounded keel/peak ratio yields no bound, not a zero and not an
  #     unbounded pass.
  checkr "no bounded keel/peak: no amended bound at all" ""    roofline 0.390 ""    0.486486
  checkr "no roofline: no amended bound at all"          ""    roofline 0.390 0.311 0
  # 20. And no bound against a denominator that was never chosen: passing the plain
  #     OpenBLAS bound through here would make the substitution #86 forbids, silently,
  #     inside a helper the caller trusts to fail closed.
  checkr "indeterminate source: no bound at all"         ""    indeterminate 0.641 0.311 0.486486

  echo
  if [[ "$FAILED" -eq 0 ]]; then
    echo "roofline controls: all pass"
  else
    echo "roofline controls: FAILED" >&2
    exit 1
  fi
}

main "$@"
