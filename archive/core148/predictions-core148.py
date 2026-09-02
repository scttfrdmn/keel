#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""#148 test 2: the REGISTERED bands, committed before the run and imported by the analyzer.

This file exists so the thresholds cannot be restated after the deltas are visible.
DESIGN.md §5 rule 24 requires a pre-registered prediction to be stated in the instrument's
own output space AND at that instrument's own row granularity; the second clause was added
2026-09-01 because supplying thresholds after seeing the deltas would be fresh judgment
wearing pre-registration's clothes. An analyzer carrying its own copies of these numbers
would satisfy the letter of that and invert it, so the analyzer imports THIS module and
prints the provenance of every number it applies.

WHAT TEST 2 CAN AND CANNOT SEPARATE
#148 measured a 3.36-4.31x collapse confined to mask width 1 and killed two of its three
registered candidates: the contention gradient (widths 2/4/8 should differ under it and are
indistinguishable) and frequency residency (refuted by sign). What survived was one family --
something specific to cpu0, the only core a width-1 derived mask can name -- plus a FOURTH
candidate #148 never listed: a width-1 mask confines the WHOLE PROCESS to one core, so the
runtime's own threads (sysmon, the GC workers, the timer goroutine) share the single permitted
cpu with the benchmark regardless of WHICH core it is. That predicts collapse on any single
core exactly as generic one-core confinement does, so cpu5 alone cannot separate them. Hence
the fourth arm: two THREAD SIBLINGS are two logical cpus on ONE physical core, which gives the
runtime somewhere else to run without giving the benchmark a second core of throughput.

    c5  separates A from {B, C}
    smt separates C from B

`ref` is the same mask keel_pin_mask derives for width 2 on a single-LLC host, so it is #148's
own w2 arm reached by the explicit path: the denominator AND a harness control. If `ref` does
not land on a level #148 measured, the explicit path changed the experiment and no other arm
in the run is interpretable.

THE LEVEL LOTTERY, and why it does not move the bands
Re-derived while verifying this file's first draft, whose absolute-GFLOP/s harness control was
typed from recall and wrong on two of four rows. The disagreement was the instrument, not the
recall: #148's healthy arms are internally tight (30-sample spread typically 1.01) but each
INVOCATION lands on one of 2-4 discrete levels, and which level is unrelated to width --
`4x32/kc=8` reads 3.19 in four arms and 2.77 in two, `2x32/kc=32` has three levels spanning
1.21x, and two arms switched level MID-INVOCATION.

NONE OF THAT IS NEW HERE, and the attribution matters because this file is the record. The
between-invocation levels are #148 §6's own finding, published with the width sweep, together
with the conclusion that the registered band has no resolution where it was registered -- the
structure is 5-15% wide and the band was 15%. The two mid-invocation switches are #147, filed,
whose title says the hazard outright: a benchstat within-run band can be the mode gap, 18 of
138 (row,arm) cells. What this check adds is neither of those findings but a number about THIS
test: the measured envelope each candidate estimator produces under them, below, which is what
decides whether test 2's dichotomy is resolvable at all before it is run.

Measured on #148's own arms (ratio of width w to width 2, six healthy arms, four registered
rows), so this predates test 2 entirely:

    estimator                 registered envelope   16 control rows    width-1 collapse
    median of one pass        [0.867, 1.000]        [0.995, 1.028]     [0.232, 0.258]
    max of two passes         [0.868, 1.001]        [0.998, 1.028]     [0.232, 0.258]
    pooled median, 60 samples [0.928, 1.001]        [0.997, 1.056]     [0.241, 0.296]

The dichotomy survives all three with a factor of 2.2 to spare: the worst healthy floor is
0.868 and the worst collapse is 0.296, with the registered boundary at 0.40 between them. So
the bands below are the ones written before this check and they STAY there -- "found by running
it is not a reason to move it". Had the check gone the other way, widening INTACT using #148's
published envelope would have been legitimate (the standard predates the result) and widening
it using test 2's own deltas would not.
"""

# ---------------------------------------------------------------- arms

# `smt` resolves cpu0's sibling ON THE HOST: the driver reads thread_siblings_list from sysfs
# and refuses the run if it cannot prove the pair is one physical core. A hard-coded sibling
# id is a recalled fact about a machine and this file may not carry one.
ARMS = ["ref", "c0", "c5", "smt"]
REF = "ref"
POSITIVE_CONTROL = "c0"   # every branch predicts this collapses; if it does not, the HARNESS
                          # failed to reproduce #148's width-1 arm and nothing else can be read

# ---------------------------------------------------------------- bands

# Output space: ratio of an arm's estimator to the SAME RUN's `ref` arm, per row.
COLLAPSE = (0.00, 0.40)
INTACT = (0.85, 1.15)
# Deliberately not adjacent. A row in (0.40, 0.85) or above 1.15 is INDETERMINATE and decides
# nothing -- it refutes the clean dichotomy rather than being rounded into a branch. The level
# lottery above is the expected cause if one appears.

# Primary estimator: pooled median of both passes' samples for that arm and row. Tightest
# floor of the three measured (0.928 vs 0.868), and unlike max-of-two-passes it does not bias
# the DENOMINATOR upward -- max gave 0.868 on 2x32/kc=8 by pairing a twice-unlucky numerator
# with a once-lucky reference, which is the one asymmetry a ratio cannot afford.
ESTIMATOR = "pooled-median"
# Secondary, printed beside it: the branch verdict must agree under both estimators. A
# disagreement is reported as INDETERMINATE, not resolved -- two dirty estimators bracket the
# truth and the width between them is the systematic, not a tiebreak to be spent.
ESTIMATOR_SECONDARY = "max-of-two-pass-medians"

# ---------------------------------------------------------------- rows

# The four small-kc scalar rows: the ones carrying the advantage that vanished at width 1.
REGISTERED = [("2x32", "scalar", 8), ("4x32", "scalar", 8),
              ("2x32", "scalar", 32), ("4x32", "scalar", 32)]

# Every other row is a control, and that is a criterion here rather than a remark: all 16 must
# read INTACT in all 8 arm-passes. #148 measured the 12 avx512 rows at 0.949-1.029 and the 4
# large-kc scalar rows at 0.947-1.035 across every width, and 0.997-1.056 under the pooled
# estimator, so INTACT is a band they cleared with room. If they move, the registered reading
# is about the host and not about affinity.
CONTROL_INTACT_ALL_ARMS = True

# ---------------------------------------------------------------- truth table

# "ood" = OUT OF DOMAIN, declared before the run. Branch A makes no prediction for `smt`:
# if the interference is cpu0 servicing interrupts, whether the benchmark escapes it by moving
# to cpu0's own sibling depends on what fraction of that physical core the servicing costs,
# which nothing in this run measures. Rule 24's alternative to a threshold is declaring the
# cell out of domain in advance, not explaining it afterwards.
BRANCHES = {
    "A": {"name": "cpu0-specific: interrupt affinity or kernel work on cpu0 and its sibling",
          "ref": "intact", "c0": "collapse", "c5": "intact", "smt": "ood"},
    "B": {"name": "one physical core is not enough throughput, whichever core it is",
          "ref": "intact", "c0": "collapse", "c5": "collapse", "smt": "collapse"},
    "C": {"name": "the Go runtime's own threads contend for the single permitted cpu",
          "ref": "intact", "c0": "collapse", "c5": "collapse", "smt": "intact"},
}

# ---------------------------------------------------------------- harness control

# Stated as admissible LEVELS, not a single median, because the level lottery makes any single
# figure a draw. Derived from the six healthy arms of archive/width148 (a2,a4,a8,b2,b4,b8) by
# taking each row's per-arm medians; `ref` must land inside this range with 5% of slack on each
# side. The bracket is the lottery's own width, so this control can only fail if `ref` is off
# the lottery entirely -- which is what "the explicit path changed the experiment" looks like.
REF_ADMISSIBLE_GFLOPS = {
    ("2x32", 8):  (2.766, 3.190),
    ("4x32", 8):  (2.764, 3.189),
    ("2x32", 32): (3.014, 3.654),
    ("4x32", 32): (3.067, 3.548),
}
REF_ADMISSIBLE_SLACK = 0.05
