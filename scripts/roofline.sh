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

# throughput_verdict — the whole decision, as a pure function.
#
# Arguments, in order:
#   1 best_lo         fraction of peak of the shape under test, net of CI
#   2 best_ipf        that shape audited instructions per FMA
#   3 peak_floor      flat floor for an FMA-bound host (0.55)
#   4 roof_floor      fraction of roofline required of an issue-bound host (0.90)
#   5 converge_max    max rate spread across ceiling mixes to call a host issue-bound
#   6 mix_spread_min  min insns/FMA spread required for that call to mean anything
#   7 sweep_best_ipf  best zero-spill insns/FMA in the recorded sweep
#   8 shape_slack     how far above it a shape may be and still get a roofline
#   9.. ceiling mixes as f:I pairs, EXCLUDING the shape under test. The
#       register-only peak kernel is one of these, entered as 1.0:I_peak.
#
# Echoes one line: CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY
#   CLASS  in {fma, issue}
#   ROOF   fraction of peak the front end permits (0 if not computed)
#   ATTAIN best_lo / ROOF (0 if not computed)
#   RESULT in {pass, fail, refuse}
#            refuse = classified issue-bound but denied a roofline, because the
#            shape is too far off the sweep best for its own instruction count
#            to be a fair denominator. A refusal is a gate failure, not a pass.
#   WHY    in {-, nomixes, diverge, samemix, falsified, shape}
throughput_verdict() {
  local best_lo="$1" best_ipf="$2" peak_floor="$3" roof_floor="$4"
  local converge_max="$5" mix_spread_min="$6" sweep_best="$7" slack="$8"
  shift 8
  awk -v best_lo="$best_lo" -v best_ipf="$best_ipf" \
      -v peak_floor="$peak_floor" -v roof_floor="$roof_floor" \
      -v converge_max="$converge_max" -v mix_spread_min="$mix_spread_min" \
      -v sweep_best="$sweep_best" -v slack="$slack" \
      -v mixes="$*" '
  BEGIN {
    # Reduce the ceiling set. Any pair with a non-positive I is dropped rather
    # than silently contributing a zero rate.
    n = split(mixes, m, /[ \t]+/)
    cnt = 0
    for (k = 1; k <= n; k++) {
      if (m[k] == "") continue
      if (split(m[k], fi, ":") != 2) continue
      f = fi[1] + 0; ii = fi[2] + 0
      if (ii <= 0) continue
      p = f * ii
      if (cnt == 0) { pmin = p; pmax = p; imin = ii; imax = ii }
      else {
        if (p  < pmin) pmin = p
        if (p  > pmax) pmax = p
        if (ii < imin) imin = ii
        if (ii > imax) imax = ii
      }
      cnt++
    }

    class = "fma"; roof = 0; attain = 0; why = "-"
    cspread = 0; mspread = 0

    if (cnt < 2) {
      # One mix cannot establish a ceiling: nothing to converge with.
      why = "nomixes"
    } else {
      cspread = (pmin > 0) ? pmax / pmin : 0
      mspread = (imin > 0) ? imax / imin : 0

      # Issue-bound requires BOTH: the ceiling mixes agree on a retirement rate,
      # AND they were different enough for that agreement to be evidence rather
      # than a coincidence between two similar loops.
      if (cspread <= 0 || cspread > converge_max)     why = "diverge"
      else if (mspread < mix_spread_min)              why = "samemix"
      else {
        roof   = (best_ipf > 0) ? pmax / best_ipf : 0
        attain = (roof > 0) ? best_lo / roof : 0

        if (attain > 1.0) {
          # A ceiling the machine exceeds is not a ceiling. The issue-bound
          # hypothesis is falsified by the data offered in its support.
          why = "falsified"
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

    if (class == "issue" && why == "shape") result = "refuse"
    else if (class == "issue")             result = (attain  >= roof_floor) ? "pass" : "fail"
    else                                   result = (best_lo >= peak_floor) ? "pass" : "fail"

    printf "%s %.4f %.4f %.6f %.6f %s %s\n", class, cspread, mspread, roof, attain, result, why
  }'
}
