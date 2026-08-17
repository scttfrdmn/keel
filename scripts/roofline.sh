#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Shared helper: P2's throughput verdict (DESIGN.md §4/P2 as amended by the
# ruling on issue #19).
#
# WHY THIS IS ITS OWN FILE. The amendment says an issue-bound host is held to 90%
# of a roofline computed from the instruction count of the kernel under test,
# instead of to a flat 55% of measured peak. That is a weaker gate unless three
# things are true, and all three are easy to assert and hard to believe:
#
#   1. A host cannot classify itself issue-bound merely by being slow.
#   2. A kernel cannot raise its own roofline by emitting more instructions.
#   3. The roofline test is not implied by the classifier that admits you to it.
#
# So the decision lives in a pure function with no I/O, and scripts/roofline-test.sh
# drives it with adversarial fixtures — a padded kernel, a slow kernel, a host that
# is issue-bound but genuinely underperforming, a sandbagged alternate shape — and
# asserts each is refused. The logic the test exercises is the logic the gate runs;
# the gate sources this file.
#
# Property 3 is not decoration: the first draft of this file failed it. See the
# INDEPENDENCE note below.
#
# THE ROOFLINE IS CLOCK-FREE. With f_i the measured fraction of peak for mix i, R
# the host FMA/cycle and I_i its audited instructions per FMA, the observed
# retirement rate is r_i = f_i * R * I_i. R is common to every mix on a host, so
# it cancels from both the classifier (a ratio of rates) and the roofline
# (r_max / (R * I)):
#
#       roofline(I) = max_i(f_i * I_i) / I
#
# which is why P2 needs no clock measurement, no taskset and no perf counters.
# Rates below are therefore quoted in units of R, i.e. as the products f_i * I_i.
#
# INDEPENDENCE (property 3). The obvious formulation computes the convergence
# spread and the roofline over the same set of mixes, the shape under test
# included. That is circular in a way the derivation hides. Write p_i = f_i * I_i
# and let b be the shape under test. Then
#
#       attain = f_b / (max_i p_i / I_b) = p_b / max_i p_i
#
# and if b is one of the i, then p_b >= min_i p_i, so
#
#       attain >= min_i p_i / max_i p_i = 1 / cspread >= 1 / converge_max
#
# With converge_max = 1.10 that is attain >= 0.909, which already exceeds the 0.90
# floor. Every host admitted by the classifier would clear the floor by
# construction, and the "90% of roofline" clause would decide nothing. Worse, the
# binding case is exactly the one where it is vacuous: on janus the shape under
# test *is* the argmin of p_i, so 1/cspread and attain are the same number to four
# places (0.9476).
#
# So the ceiling is established by the OTHER mixes — the alternate shapes from the
# sweep plus the register-only peak kernel — and the shape under test is then
# judged against it. Now p_b is absent from max_i p_i and the two tests are
# independent measurements.
#
# FALSIFICATION. Excluding b invites the mirror abuse: understate the ceiling by
# sandbagging the alternates, and any b clears it. Two things stop that. First,
# the register-only peak kernel is always in the ceiling set, is audited, and has
# f = 1 by definition, so max_i p_i >= I_peak (2.25 here) no matter what the
# alternates do — sandbagging an alternate breaks convergence rather than lowering
# the roofline. Second, a ceiling the machine demonstrably exceeds is not a
# ceiling: if attain > 1 the issue-bound hypothesis is falsified by its own data
# and the host reverts to the flat floor. This is what correctly returns antares
# (Zen 5) to the FMA-bound branch.
#
# HOW MUCH SLACK THE AMENDMENT ACTUALLY GRANTS. Combining the two guards, the
# effective floor for an issue-bound host is
#
#       roof_floor * max_i p_i / I_b  >=  0.90 * I_peak / (sweep_best * slack)
#                                     =   0.90 * 2.25 / 4.659  =  0.435
#
# so the amendment can lower the bar from 55% of peak to no less than 43.5% of
# peak, and only for a host that has independently demonstrated a front-end
# ceiling. That is the number the amendment costs; it is not unbounded.
#
# WHEN THE CLASSIFICATION CANNOT BE DECIDED (the ruling on issue #86). Two of the
# comparisons below select the CLASS, and both consume measured fractions of peak:
# the convergence test (cspread vs converge_max) and the falsification test
# (attain vs 1). The class then chooses P2's floor for this host *and* P3's
# denominator for it, so one noisy measurement three steps upstream can move four
# verdicts at once. On 2026-08-15 it did: janus's ceiling mixes read ~7% low in one
# classify invocation, the spread crossed 1.10, the host reclassified fma-bound,
# and P3's criterion 7 divided an unchanged 76.8 GFLOP/s by OpenBLAS 194.4 instead
# of by the 105.4 roofline. 72.9% PASS became 39.5% FAIL with the numerator steady
# to 0.1%, and the log dispatched the operator to repair a fingerprint that was
# right.
#
# The ruling: *a verdict cannot be more certain than the least certain link in its
# derivation chain.* So each class-selecting comparison is made against the
# INTERVAL rather than the point estimate, and has three outcomes instead of two —
# clear of the bar, clear on the wrong side, or straddling it. Straddling is
# CLASS=indeterminate, RESULT=unmeasured, and every criterion that consumes the
# class reports UNMEASURED naming that one cause. This is #67's third state
# (bench_ratio_grade in scripts/bench.sh) applied one link further upstream, at the
# derivation instead of at the criterion.
#
# WHY THE §4 RE-RUN ALLOWANCE IS NOT THE REMEDY HERE, AND STAYS WHERE IT IS.
# DESIGN.md §4 grants a throughput sentinel reading red exactly one archived
# re-run. That allowance is priced for a self-contained red: one verdict, one
# deliberate re-run, friction as the feature, both outputs archived. A class
# straddle spends four verdicts per exercise, which is not the purchase §4
# authorised. So the RESULT boundaries — attain vs roof_floor, best_lo vs
# peak_floor — are deliberately NOT interval-tested here and keep the allowance;
# the CLASS boundaries get the straddle test instead.
#
# The spread's interval is the worst case over both ends:
#
#       cspread_hi = max_i p_i_hi / min_j p_j_lo
#       cspread_lo = max(1, max_i p_i_lo / min_j p_j_hi)
#
# When one very wide mix supplies both ends the interval widens and the verdict is
# indeterminate, which is the right answer rather than a defect: a mix that wide
# has not established a ceiling. Bounds arrive from the caller as measurements
# (bench_ratio_lo / bench_ratio_hi), not from a symmetry assumption about the
# point estimate; a mix given no bounds is treated as zero-width, so a fixture
# written before this amendment means exactly what it meant then.

# throughput_verdict — the whole decision, as a pure function.
#
# Arguments, in order:
#   1 best_lo         fraction of peak of the shape under test, net of CI
#   2 best_hi         the upper end of that same interval (bench_ratio_hi)
#   3 best_ipf        that shape audited instructions per FMA
#   4 peak_floor      flat floor for an FMA-bound host (0.55)
#   5 roof_floor      fraction of roofline required of an issue-bound host (0.90)
#   6 converge_max    max rate spread across ceiling mixes to call a host issue-bound
#   7 mix_spread_min  min insns/FMA spread required for that call to mean anything
#   8 sweep_best_ipf  best zero-spill insns/FMA in the recorded sweep
#   9 shape_slack     how far above it a shape may be and still get a roofline
#  10.. ceiling mixes as f:I[:f_lo[:f_hi]], EXCLUDING the shape under test. The
#       register-only peak kernel is one of these, entered as 1.0:I_peak — its
#       f is 1 exactly by definition, and its own interval is already folded into
#       every other mix's bounds, which are ratios against it. An absent f_lo or
#       f_hi defaults to f, i.e. to a zero-width interval.
#
# Echoes one line:
#   CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY CSPREAD_LO CSPREAD_HI ATTAIN_HI
#   CLASS  in {fma, issue, indeterminate}
#   ROOF   fraction of peak the front end permits (0 if not computed)
#   ATTAIN best_lo / ROOF (0 if not computed), and ATTAIN_HI is best_hi / ROOF
#   RESULT in {pass, fail, refuse, unmeasured}
#            refuse = classified issue-bound but denied a roofline, because the
#            shape is too far off the sweep best for its own instruction count
#            to be a fair denominator. A refusal is a gate failure, not a pass.
#            unmeasured = the class straddled a bar, so no floor applies to this
#            host this run. Not a pass either: the caller reports it UNMEASURED,
#            which fails the gate for a failure to measure rather than for a
#            missed floor.
#   WHY    in {-, nomixes, diverge, samemix, falsified, shape,
#              nearconverge, nearceiling, falsifiedanyway, samemixanyway}
#            nearconverge = the spread's interval straddles converge_max
#            nearceiling  = attainment's interval straddles 1, so whether the
#                           machine exceeds its own claimed ceiling is undecided
#            *anyway      = the spread's interval straddles converge_max AND both
#                           branches of that straddle give the same class, so the
#                           class is decided despite the undecided comparison.
#                           CSPREAD_LO/HI still show the straddle, so the record
#                           says "could not decide this, did not need to".
throughput_verdict() {
  local best_lo="$1" best_hi="$2" best_ipf="$3" peak_floor="$4" roof_floor="$5"
  local converge_max="$6" mix_spread_min="$7" sweep_best="$8" slack="$9"
  shift 9
  awk -v best_lo="$best_lo" -v best_hi="$best_hi" -v best_ipf="$best_ipf" \
      -v peak_floor="$peak_floor" -v roof_floor="$roof_floor" \
      -v converge_max="$converge_max" -v mix_spread_min="$mix_spread_min" \
      -v sweep_best="$sweep_best" -v slack="$slack" \
      -v mixes="$*" '
  BEGIN {
    # Reduce the ceiling set. Any mix with a non-positive I is dropped rather
    # than silently contributing a zero rate. Each mix carries three products:
    # the point estimate, and the two ends of its interval. A mix given no
    # bounds is zero-width, so pre-#86 fixtures reduce to exactly what they did.
    n = split(mixes, m, /[ \t]+/)
    cnt = 0
    for (k = 1; k <= n; k++) {
      if (m[k] == "") continue
      nf = split(m[k], fi, ":")
      if (nf < 2) continue
      f = fi[1] + 0; ii = fi[2] + 0
      if (ii <= 0) continue
      flo = (nf >= 3 && fi[3] != "") ? fi[3] + 0 : f
      fhi = (nf >= 4 && fi[4] != "") ? fi[4] + 0 : f
      p = f * ii; plo = flo * ii; phi = fhi * ii
      if (cnt == 0) {
        pmin = p; pmax = p; imin = ii; imax = ii
        pminlo = plo; pminhi = phi; pmaxlo = plo; pmaxhi = phi
      } else {
        if (p  < pmin) pmin = p
        if (p  > pmax) pmax = p
        if (ii < imin) imin = ii
        if (ii > imax) imax = ii
        # The interval ends are taken over the whole set, independently of which
        # mix is the point argmax or argmin: the worst case for the ratio is the
        # highest any mix could be over the lowest any mix could be. When one
        # wide mix supplies both, the interval widens and the verdict goes
        # indeterminate, which is the honest reading of a mix that wide.
        if (plo < pminlo) pminlo = plo
        if (phi < pminhi) pminhi = phi
        if (plo > pmaxlo) pmaxlo = plo
        if (phi > pmaxhi) pmaxhi = phi
      }
      cnt++
    }

    class = "fma"; roof = 0; attain = 0; attain_hi = 0; why = "-"
    cspread = 0; mspread = 0; cspread_lo = 0; cspread_hi = 0

    if (cnt < 2) {
      # One mix cannot establish a ceiling: nothing to converge with.
      why = "nomixes"
    } else {
      cspread = (pmin > 0) ? pmax / pmin : 0
      mspread = (imin > 0) ? imax / imin : 0
      # A spread cannot be below 1 -- fully overlapping intervals mean "as
      # converged as a measurement can show", not "converged by -3%".
      cspread_lo = (pminhi > 0) ? pmaxlo / pminhi : 0
      if (cspread_lo < 1.0) cspread_lo = 1.0
      cspread_hi = (pminlo > 0) ? pmaxhi / pminlo : 0

      # Issue-bound requires BOTH: the ceiling mixes agree on a retirement rate,
      # AND they were different enough for that agreement to be evidence rather
      # than a coincidence between two similar loops.
      #
      # The convergence test is the first of the two class-selecting comparisons
      # and it is graded three ways (#86). Note that with zero-width intervals
      # `cspread_hi <= converge_max` and `cspread_lo > converge_max` partition
      # the outcomes exactly as the single `cspread > converge_max` test did, so
      # the third state is carved out of neither of the old two: it is carved out
      # of the region where they disagreed with each other.
      if (cspread <= 0 || cspread_hi <= 0)            why = "diverge"
      else if (cspread_lo > converge_max)             why = "diverge"
      else if (cspread_hi > converge_max) {
        # THE COLLAPSE CASE, and it is the converse of the law on #86. "No verdict
        # more certain than the least certain link" has a second edge: no verdict
        # LESS certain than the derivation requires. An undecided comparison whose
        # branches lead to the SAME class decides the class anyway, and withholding
        # it would be the same defect pointed the other way — an UNMEASURED that
        # the data settles.
        #
        # Found by a real reading grazing the bar: on 2026-08-16 antares measured
        # a ceiling spread interval of [1.077, 1.100] against the 1.10 bar while
        # retiring at 162.2%..165.2% of the roofline that interval implies. Had
        # the upper end landed one rounding step higher, a host that falsifies its
        # claimed ceiling by 62% would have been reported "classification
        # indeterminate" — on the strength of a comparison whose two branches both
        # say fma-bound: converged means falsified means fma, and not converged
        # means no ceiling means fma. Neither branch reads `roof`, and the flat
        # floor that then applies is the same expression in both.
        #
        # The test is over the WHOLE interval, not the point estimate: roof_hi =
        # pmax_hi / I is the highest ceiling the reading admits, so best_lo/roof_hi
        # > 1 means the shape retires above the ceiling at every reading in it.
        # A collapse justified by the midpoint would be exactly the noise-driven
        # verdict this amendment exists to prevent.
        roof_hi = (best_ipf > 0) ? pmaxhi / best_ipf : 0
        if (mspread < mix_spread_min) {
          # No ceiling either way: a spread this narrow in insns/FMA is not
          # evidence whether or not the rates agree. mspread is audited integers,
          # so it carries no interval and "at every reading" is unconditional.
          why = "samemixanyway"
        } else if (roof_hi > 0 && best_lo / roof_hi > 1.0) {
          why = "falsifiedanyway"
          roof      = (best_ipf > 0) ? pmax / best_ipf : 0
          attain    = (roof > 0) ? best_lo / roof : 0
          attain_hi = (roof > 0) ? best_hi / roof : 0
        } else {
          class = "indeterminate"; why = "nearconverge"
        }
      }
      else if (mspread < mix_spread_min)              why = "samemix"
      else {
        roof      = (best_ipf > 0) ? pmax / best_ipf : 0
        attain    = (roof > 0) ? best_lo / roof : 0
        attain_hi = (roof > 0) ? best_hi / roof : 0

        if (attain > 1.0) {
          # A ceiling the machine exceeds is not a ceiling. The issue-bound
          # hypothesis is falsified by the data offered in its support.
          why = "falsified"
        } else if (attain_hi > 1.0) {
          # ... but whether it exceeds the ceiling is itself a measurement, and
          # this reading cannot say. Second class-selecting comparison, same
          # three-way grading. It precedes the shape guard for the same reason
          # falsification does: a hypothesis whose status is unknown cannot be
          # "issue-bound but refused a roofline".
          class = "indeterminate"; why = "nearceiling"
        } else if (best_ipf > sweep_best * slack) {
          # A roofline built from the instruction count of the kernel under test
          # rises as that kernel gets worse. Without this guard, "90% of
          # roofline" would mean "90% of whatever we happened to emit".
          class = "issue"; why = "shape"
        } else {
          class = "issue"
        }
      }
    }

    # RESULT boundaries stay two-state on purpose; see the §4 pricing note above.
    if (class == "indeterminate")           result = "unmeasured"
    else if (class == "issue" && why == "shape") result = "refuse"
    else if (class == "issue")              result = (attain  >= roof_floor) ? "pass" : "fail"
    else                                    result = (best_lo >= peak_floor) ? "pass" : "fail"

    printf "%s %.4f %.4f %.6f %.6f %s %s %.4f %.4f %.6f\n", \
      class, cspread, mspread, roof, attain, result, why, \
      cspread_lo, cspread_hi, attain_hi
  }'
}

# ---------------------------------------------------- P3's denominator (#23)
# p3_denominator — which number keel's Sgemm is divided by at 2048^3, as a pure
# function of measurements already taken.
#
# The ruling (DESIGN.md §4/P3, as amended): the reference is same-host OpenBLAS,
# and on an issue-bound host the denominator is
#
#       min(same-host OpenBLAS, roofline * measured peak)
#
# because OpenBLAS's K-loop on such a host is hand assembly that folds
# accumulation and an embedded broadcast into single FMAs — instructions the
# intrinsic layer provably cannot emit (T12, #17/#18) — so it sits *above* the
# front-end ceiling keel's kernels are capped by. Asking for 60% of that is asking
# the decode stage rather than the kernel.
#
# Four properties, each of which is a line of code below rather than a promise:
#
# ONE-SIDED. The result is never larger than OB_RATE, so this can only ever lower
# a denominator and never raise one. An FMA-bound host takes the OpenBLAS branch
# unconditionally: no classification, no roofline, no leniency.
#
# THE SHAPE UNDER TEST IS THE SHAPE THAT RAN. I_ACTIVE is the audited insns/FMA of
# the kernel `Sgemm` actually dispatched to (read from the keel-bench-kern marker of
# the run that produced the ratio), not of the best shape available. It carries P2's anti-vacuity guard
# unchanged: a shape more than `slack` above the sweep's best zero-spill insns/FMA
# is refused a roofline and the host reverts to plain OpenBLAS. A fatter kernel
# cannot buy itself a lower bar.
#
# SELF-RETIRING. roof = PMAX / I_ACTIVE rises as the lowering improves, and PMAX is
# pinned from below by the register-only peak kernel. Once roof * peak exceeds
# OpenBLAS the min() picks OpenBLAS and the amendment is gone with no expiry clause
# to remember — reported as why=reference, which is the outcome to hope for.
#
# NO HIDDEN DENOMINATOR. The caller is handed SOURCE and ROOF so it can print both
# ratios; §7 rule 7 applies to the gate's own arithmetic.
#
# NO DENOMINATOR CHOSEN BY NOISE (#86). `class` is a derived quantity, and when its
# derivation straddled a bar it arrives here as "indeterminate". That is neither
# branch: it takes the fourth exit below, which yields no denominator at all, and
# the caller must report UNMEASURED rather than divide. The failure this closes is
# specific — an indeterminate class used to reach the `class != "issue"` test and
# come out of it as a confident `fma-bound`, which is the strict direction and so
# could never flatter keel, but did produce a FAIL naming a floor on the strength
# of a classification the run could not make.
#
# Arguments, in order:
#   1 class        "issue", "indeterminate", or anything else, from
#                  throughput_verdict on this host
#   2 pmax         max_i p_i for this host (ROOF * I_b from that same verdict)
#   3 i_active     audited insns/FMA of the shape Sgemm ran
#   4 sweep_best   best zero-spill insns/FMA in the recorded sweep
#   5 slack        how far above it a shape may be and still get a roofline
#   6 ob_rate      same-host OpenBLAS GFLOP/s at 2048^3
#   7 peak_rate    same-host measured peak GFLOP/s
#
# Echoes one line: DENOM SOURCE ROOF WHY
#   SOURCE in {openblas, roofline, indeterminate}
#   ROOF   the roofline fraction used (0 when the denominator is OpenBLAS)
#   DENOM  0 when SOURCE is indeterminate -- there is no denominator, and the
#          caller must branch on SOURCE before dividing by it
#   WHY    in {fma-bound, shape, reference, issue-capped, nopeak,
#              classindeterminate}
p3_denominator() {
  awk -v class="$1" -v pmax="$2" -v i_active="$3" -v sweep_best="$4" \
      -v slack="$5" -v ob="$6" -v peak="$7" '
  BEGIN {
    if (class == "indeterminate") { print "0.000000 indeterminate 0.000000 classindeterminate"; exit }
    if (class != "issue")                { print_ob("fma-bound"); exit }
    if (i_active <= 0 || pmax <= 0)      { print_ob("nopeak");    exit }
    if (peak <= 0)                       { print_ob("nopeak");    exit }
    if (i_active > sweep_best * slack)   { print_ob("shape");     exit }

    roof = pmax / i_active
    cap  = roof * peak
    # min(), stated as a comparison so the branch taken is reportable.
    if (cap < ob) printf "%.6f roofline %.6f issue-capped\n", cap, roof
    else          printf "%.6f openblas %.6f reference\n",    ob,  roof
  }
  function print_ob(why) { printf "%.6f openblas 0.000000 %s\n", ob, why }'
}

# p3_ratio_lo SOURCE RLO PKLO ROOF — the conservative lower bound on keel's ratio
# against whichever denominator p3_denominator chose.
#
# Against plain OpenBLAS that is just $RLO, benchstat's bound on Sgemm/OpenBLAS.
# Against the roofline cap it is keel_lo / (roof · peak_hi), and since
# bench_ratio_lo already returns keel_lo/peak_hi as $PKLO, that is $PKLO/roof.
#
# Why a max() and not simply the second expression: the cap is *below* OpenBLAS by
# construction, so $RLO is also a valid lower bound on the same ratio, and which of
# the two is tighter depends on whose confidence interval is wider — the reference's
# or the peak kernel's. Taking the larger is not cherry-picking; both bound the same
# quantity from below, and discarding the tighter one would report a ratio lower than
# the measurements support. The direction that would be cherry-picking — inventing a
# bound the measurements do not support — is impossible here, because both inputs are
# benchstat's own net-of-CI ratios.
#
# It is a separate function with fixtures because it is the arithmetic where a
# division in the wrong direction would silently flatter keel, and reading it inline
# in a gate script is not evidence that it is right.
#
# Echoes the bound, or nothing if the inputs cannot support one (empty $PKLO on the
# roofline branch is a failure to measure, and the caller must treat it as one).
# SOURCE=indeterminate is one of those: there is no denominator to bound a ratio
# against, and passing the plain OpenBLAS bound through would be the very
# substitution #86 forbids, made silently by a helper.
p3_ratio_lo() {
  awk -v src="$1" -v rlo="$2" -v pklo="$3" -v roof="$4" '
  BEGIN {
    if (src == "indeterminate") exit
    if (src != "roofline") { if (rlo != "") printf "%.6f\n", rlo; exit }
    if (pklo == "" || roof + 0 <= 0) exit
    v = pklo / roof
    printf "%.6f\n", (v > rlo + 0) ? v : rlo + 0
  }'
}

# p3_collapse NCAND NCLEAR NMISS — whether an undecidable classification still
# decides criterion 6, from the tally of how its candidate denominators graded.
#
# This is the collapse rule of the class decision above (why=falsifiedanyway,
# why=samemixanyway) applied at its second site, by ruling 2026-08-16. Same law:
# UNMEASURED is for verdicts that VARY over the uncertainty, so a doubt whose two
# resolutions lead to the same verdict is immaterial, and reporting UNMEASURED on
# it spends the scarcity three-state grading exists to protect. The caller grades
# each candidate on its own net-of-CI bound — never the point estimate, for the
# reason the class collapse states at length — and hands the tally here.
#
# SYMMETRIC BY DESIGN. Two agreeing misses collapse to `fail`, not to `split`. A
# host that is below the bar under every reading of every candidate is a host
# below the bar, and letting an undecidable class shelter it would be the same
# defect as letting one condemn it: in both directions the verdict would be
# reported as being about the classification when it is not.
#
# AGREEMENT REQUIRES A VERDICT FROM EVERY CANDIDATE. NCLEAR + NMISS < NCAND means
# some candidate had no bounded ratio (p3_ratio_lo declining, which is a failure to
# measure and not a value), and a candidate with no verdict cannot agree with one.
# That is `split`, so the missing bound surfaces as unmeasured rather than being
# read as assent by omission.
#
# Echoes one word: pass | fail | split.
p3_collapse() {
  awk -v ncand="$1" -v nclear="$2" -v nmiss="$3" '
  BEGIN {
    if (ncand + 0 <= 0)      { print "split"; exit }   # nothing to collapse
    if (nclear + 0 == ncand + 0) { print "pass";  exit }
    if (nmiss  + 0 == ncand + 0) { print "fail";  exit }
    print "split"
  }'
}

# fleet_coverage NHOSTS NMEASURED NCLEARED NMISSED NINDET — the per-fleet aggregate of
# BOTH gates' throughput criteria (gate-p3 criterion 6, gate-p2 criterion 5b), from the
# per-host tally.
#
# NAMED `p3_coverage` UNTIL 2026-08-16, when the ruling on #90 made it shared. Criterion
# 5b had its own inline aggregate that resolved a partial fleet to PASS while this one
# resolved it to UNMEASURED, and "two aggregates with different absence semantics is the
# divergent-copies defect relocated to the verdict layer — the exact thing fleet_shortfall
# was just built to end." Absence semantics are now stated once, here, and the rename is
# what stops the p3- prefix from inviting a p2- twin. The grounds, in the ruling's words:
# a PASS reading "(2/2)" over a three-host fleet "is a message-level truth carrying a
# fleet-level assertion — the criterion's claim is about the fleet, and a fleet with an
# absent member hasn't measured that claim." §5 rule 6 gives the absent measurement its
# one available verdict. Per-host PASSes stand as measured; only the aggregate refuses.
#
# NOT A WEAKENING, and worth being explicit since P2 is the go/no-go: UNMEASURED blocks
# green identically to FAIL (`unmeasured()` sets FAIL), so nothing that used to be red can
# become green through this. What changes is only that a partial fleet stops being
# describable as a whole one.
#
# Here for p3_collapse's own stated reason: this if-chain can turn a FAIL into an
# UNMEASURED and it was previously inline, where nothing drove its branches but a
# healthy run driving exactly one of them. Two defects were live in the inline form
# and neither could have been noticed from a green. They sit at different levels, and
# only one of them is a defect this function can hold:
#
#  1. THE CONDITION (fixed here, fixtures 25-35; 31-35 are gate-p2's input shape,
#     which differs -- p2 excludes an indeterminate host from nmeas where criterion 6
#     includes it, so the same fleet arrives as `3 2 2 0 1` from one caller and
#     `3 3 2 0 1` from the other, and only asserting both keeps either from being
#     verified by reading alone). The unmeasured branch required
#     cleared + indet == nhosts exactly, so a fleet with one no-coverage host and zero
#     misses fell through to FAIL and blamed keel's speed for a measurement hole. The
#     gate is asked whether any host measured BELOW the bar; that is what decides now.
#     Confirmed discriminating: run against the old inline logic, fixture 26's
#     (3 hosts, 2 measured, 2 cleared, 0 missed) is FAIL there and `partial` here.
#  2. THE COUNT IN THE SENTENCE (fixed at the call site, and NO fixture reaches it).
#     The miss count was DERIVED as `nhosts - ncleared - nindet`, so a fleet with one
#     slow host and one host that never produced a ratio printed "2 measured below
#     it". The verdict was right and the claim was false — the label defect ruling
#     #37 found in P5's scaling aggregate, pointing the other way. The caller now
#     counts NMISSED as it grades and names the leftover as a leftover. Passing it in
#     here also stops the derivation from being reintroduced silently, but the printed
#     sentence is the caller's. gate-p3's is still verified only by reading it.
#     gate-p2's five renderings are not, as of #90: they are extracted from the case
#     block with awk and eval'd with the verdict functions stubbed, so the counts in
#     the sentence are checked against the sentence's own bytes. The mechanism is
#     described in scripts/exercise-dead-host.sh; gate-p3's aggregate is owed the
#     same treatment and does not have it yet.
#
# Echoes one word: unmeasured | pass | fail | partial. `partial` is the caller's cue
# to report UNMEASURED naming the split and no-coverage counts separately, since
# "the two candidate denominators disagreed" and "there was no ratio at all" are
# different failures to measure and collapsing them would hide which one happened.
fleet_coverage() {
  awk -v nhosts="$1" -v nmeas="$2" -v nclear="$3" -v nmiss="$4" -v nindet="$5" '
  BEGIN {
    if (nmeas  + 0 <= 0)          { print "unmeasured"; exit }
    if (nhosts + 0 <= 0)          { print "unmeasured"; exit }
    if (nclear + 0 == nhosts + 0) { print "pass";       exit }
    if (nmiss  + 0 >  0)          { print "fail";       exit }
    print "partial"
  }'
}

# fleet_shortfall NHOSTS NJUDGED -- the clause a fleet aggregate appends when it
# heard from fewer hosts than were configured, and nothing when coverage is complete.
#
# WHY THIS EXISTS (#90). gate-p2's criterion 5b counts hosts that produced a judgeable
# reading and divides by that count, so with one host of three unreachable it printed
#
#     PASS  every host that produced a judgeable throughput reading cleared its floor (2/2)
#
# which is arithmetically true and reads as fleet-wide. gate-p3's OpenBLAS aggregate had
# the same shape and got the fix at its own call site (64a05e1, `OB_NOCOVER`); this is
# the sibling, and putting the clause in one function is what stops there being a third.
# The survivors are the honest numerator of what WAS judged -- the defect is the missing
# statement of how many were asked, not the fraction -- so this appends rather than
# rewriting the fraction, and the verdict is untouched (#90 asks Scott whether a partial
# fleet should resolve to UNMEASURED as criterion 6's now does; that half is not mine).
#
# Fixture-backed because the arithmetic is where an off-by-one would hide and would be
# invisible on a healthy run: complete coverage is the only case a green fleet produces,
# and it is the one case that prints nothing.
fleet_shortfall() {
  awk -v n="$1" -v j="$2" '
  BEGIN {
    n += 0; j += 0
    # A fleet size that could not be read is not a complete fleet. Saying so is the
    # fail-closed direction: silence here would mean "coverage complete".
    if (n <= 0) { print " -- over a fleet whose configured size this run could not read"; exit }
    if (j >= n) { print ""; exit }
    printf " -- but %d of the %d configured host%s produced no judgeable reading at all, so this line covers %d of %d (the UNMEASURED lines above say which)\n", \
      n - j, n, (n == 1 ? "" : "s"), j, n
  }'
}
