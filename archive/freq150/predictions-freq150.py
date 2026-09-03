#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Registration for `#150`'s in-arm frequency sampler. Written and committed BEFORE the run.

`#150` recorded a 13-16% between-arm level term on janus that no treatment explains, and named the
leading hypothesis: a clock change. That hypothesis is currently untestable because `freq_khz` is
sampled only BETWEEN arms (`#81`), so every frequency reading in the record is an idle reading.

This registration binds two things before the instrument exists in a runnable state:

  1. WHAT A FREQUENCY STEP LOOKS LIKE IN THE SAMPLES, stated in the instrument's own output space
     (counts of classified samples), not as a ratio of medians -- DESIGN.md section 5 rule 24.
  2. THE PHENOMENON'S OWN APPLIED-WITNESS. The elevation is the subject, and it is not under our
     control: it appeared in 2 of 10 arms and we do not know why. If it does not recur, a null
     frequency difference is output-indistinguishable from "the clock is not the cause" -- which is
     rule 26's exact subject, one level up from a treatment. So the elevation's presence is a
     PRECONDITION, witnessed in the GFLOP/s column, and its absence makes the frequency comparison
     UNMEASURED rather than negative.

Plus one criterion on the instrument itself (Scott's condition, 2026-09-03): the sampler's own
perturbation is MEASURED, not assumed. An in-arm sampler is a co-tenant by construction, and an
instrument must not become the noun it measures.
"""

# ----------------------------------------------------------------- the host, and why 1.127 is live
#
# janus.local, read off the host 2026-09-03 before this file was written:
#
#   model                Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz   (Skylake-X, 16c/32t)
#   scaling_driver       intel_pstate        governor  performance
#   cpuinfo_min_freq     1200000 kHz         base_frequency  2200000 kHz
#   cpuinfo_max_freq     4400000 kHz         no_turbo  0     hwp_dynamic_boost  0
#
# The observed elevation ratio is 1.126-1.128. The ladder spans 1.2-4.4 GHz, so that ratio is
# comfortably available (3.500 -> 3.944 GHz is 1.127; 3.100 -> 3.494 is 1.127). The clock
# hypothesis is therefore NOT arithmetically dead before it is tested, which is the first thing
# worth checking and the cheapest.
#
# NAMED SUB-MECHANISM, because the silicon supplies one: Skylake-X applies LICENSE-BASED
# frequency limits keyed to the instruction mix -- L0 (scalar/SSE), L1 (AVX2, light AVX-512),
# L2 (heavy AVX-512) -- with hysteresis of order milliseconds after the last wide instruction.
# A license step is the right ORDER OF MAGNITUDE for 1.127. This matters beyond `#150`: keel's
# subject is AVX-512 kernels, and a license transition is a confound for every arm that mixes
# wide and scalar work in one process.
#
# This file does NOT register a license prediction. Licenses are a hypothesis about WHY a step
# occurs; the sampler tests only WHETHER one occurs. Registering both here would let a null on
# the second be read as a verdict on the first.
HOST_FREQ_KHZ = {"min": 1200000, "base": 2200000, "max": 4400000}
HOST_MODEL = "Intel(R) Core(TM) i9-9960X CPU @ 3.10GHz"
SCALING_DRIVER = "intel_pstate"

# ----------------------------------------------------------------- rows and arms
# The same four rows and the same five arms as test 3, in the SAME order, because the elevation
# appeared at run positions 1 and 4 and we cannot reproduce a phenomenon we have re-ordered.
REGISTERED_ROWS = [("2x32", 8), ("4x32", 8), ("2x32", 32), ("4x32", 32)]

ARMS = {
    "ref":  {"mask": "0,1", "cores": 2, "env": "GOMAXPROCS=1"},
    "c0":   {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1"},
    "gc":   {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GOGC=off"},
    "pre":  {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1"},
    "both": {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GOGC=off GODEBUG=asyncpreemptoff=1"},
}

# ----------------------------------------------------------------- phase order, and why
#
# PHASE 2 (the hunt) RUNS FIRST. This inverts the obvious order deliberately.
#
# The elevated arms were run positions 1 and 4 of ten; positions 5-10 were all at the floor. A
# cold-host explanation is therefore live, and a perturbation-control phase run first would spend
# ~1 hour warming the host before the hunt began -- destroying the only condition under which the
# phenomenon has ever been observed. The control phase does not care about host warmth (its
# contrast is internal and its order is a palindrome), so it pays the cost instead.
PHASES = ["hunt", "control"]

# The hunt: test 3's exact sequence, sampler ON throughout.
HUNT_A = ["ref", "c0", "gc", "pre", "both"]
HUNT_B = ["both", "pre", "gc", "c0", "ref"]

# The control: four arms of ONE config, sampler off/on/on/off. The palindrome is the point -- under
# any monotone drift in host state, arms 1 and 4 (off) straddle arms 2 and 3 (on), so the drift
# enters both terms of the on-vs-off contrast with the same sign and cancels to first order. A
# plain off/on pair cannot distinguish the sampler's cost from the drift between two arms, which is
# the same defect an unreversed A/B has.
CONTROL_ARM = "pre"
CONTROL_SEQUENCE = [("k1", False), ("k2", True), ("k3", True), ("k4", False)]

# ----------------------------------------------------------------- the sampler
#
# Reads cpu0's `scaling_cur_freq` every SAMPLE_PERIOD_S from a cpu that shares no SMT thread with
# any measured cpu. Period chosen deliberately conservative: under `intel_pstate` a
# `scaling_cur_freq` read may or may not require an MSR read on the TARGET cpu, and this
# registration does not assume which -- that is what the control phase measures. At 0.2s over a
# ~950s arm the trace carries ~4750 samples per arm, which is far more than any criterion here
# needs, so there is no reason to sample faster and every reason not to.
SAMPLE_PERIOD_S = 0.2
SAMPLED_CPU = 0
# The sampler's own cpu is DERIVED by the driver (disjoint from every measured cpu and its
# siblings) and asserted, never assumed. cpu0's siblings are 0,16 and cpu1's are 1,17.
FORBIDDEN_SAMPLER_CPUS = {0, 1, 16, 17}

# ----------------------------------------------------------------- 1. the instrument's own criterion
#
# THE SAMPLER MUST BE SMALLER THAN THE TERM IT HUNTS. The term is 12.6%-15.4%. A sampler costing a
# third of that would be a co-tenant masquerading as an observer, and every frequency reading it
# produced would describe a host it had changed.
#
# Criterion: |median(on) - median(off)| / median(off), per row, from the control phase.
#   PASS   < 4.2%   (one third of the smallest elevation this hunts, 12.6%)
#   REFUSE >= 4.2%  -- the sampler is rejected as an instrument and the hunt's frequency data is
#                      reported as SUSPECT, whatever it shows. Not renegotiated afterwards.
SAMPLER_PERTURBATION_MAX = 0.042
SAMPLER_PERTURBATION_BASIS = (
    "one third of 12.6%, the smallest elevation in #150's table; fixed before the run")

# ----------------------------------------------------------------- 2. preconditions
#
# The first is the phenomenon's applied-witness and the reason this file exists in this shape.
PRECONDITIONS = {
    "elevation_present": (
        "At least one arm of the hunt phase is ELEVATED: its median GFLOP/s exceeds the pooled "
        "floor of the same row by >= 1.08x on at least one registered row. Without this the "
        "frequency comparison has no subject and every cell below is UNMEASURED -- NOT 'the clock "
        "is refuted'. Those two readings are indistinguishable in a null and opposite in meaning "
        "(rule 26)."),
    "trace_present": (
        "Every arm admitted by the driver log carries >= 1000 in-arm frequency samples. A short "
        "trace means the sampler died mid-arm, which is an absent measurement, not a quiet one."),
    "sampler_ok": (
        "The control phase's perturbation is < 4.2% on every registered row (SAMPLER_PERTURBATION_MAX)."),
}
ELEVATION_FACTOR = 1.08   # the GFLOP/s witness for "this arm is elevated"

# ----------------------------------------------------------------- 2b. admissibility: the quietness guard
#
# NOT a criterion -- it decides which arms exist to be scored, and `#149`'s ruling makes it part of the
# registration rather than a driver detail, because its position dependence was a defect that reached
# a published report. Test 3 bounded the RAW 5-minute load at 1.25, which left the first arm of a
# pass ~1.08 of headroom and every later arm ~0.24; same guard, two sensitivities, decided by run
# position. Here the driver's OWN contribution is subtracted first and the residue is bounded.
#
# Both values are OUTPUTS of archive/freq150/derive-pedestal.py, which reads the 20 distinct instants
# in the two tracked campaign logs (36 reads, deduplicated BY INSTANT and not by read). They are
# restated here so the admissibility rule is legible in the registration, and the driver and this
# file are cross-checked against each other by test-driver-freq150.sh.
#
#   QUIET_SELF_N = 0.9800       min of l5/P over 17 saturated clean instants (which span
#                               0.980..1.054). Not 1.00, and not assumed: Scott's condition.
#   QUIET_FOREIGN_MAX = 0.27    = test 3's own 1.25 minus QUIET_SELF_N, so the guard's sensitivity
#                               stays where a bound that PREDATES this run put it and only its
#                               position dependence changes. Scored on the record: 0 of 18 clean
#                               instants refused, 1 of 1 tracked co-tenant excursions refused.
#
# Blind spot, stated rather than carried: a co-tenant sustaining under 0.27 of one cpu over five
# minutes. Removing it needs a longer per-arm sampling window, which this question does not need.
QUIET_SELF_N = 0.98
QUIET_FOREIGN_MAX = 0.27

# ----------------------------------------------------------------- 3. the classifier, in output space
#
# Each in-arm frequency sample is classified against the pooled median frequency of the FLOOR arms
# -- the arms whose GFLOP/s is NOT elevated. The boundary is the geometric midpoint between "no
# step" (1.000) and the observed elevation (1.127), i.e. sqrt(1.127) = 1.0616, rounded to 1.06.
# Stating it geometrically rather than arithmetically matters because the quantity is a ratio: the
# arithmetic midpoint 1.0635 would sit closer to the step than to the null on a log scale.
FREQ_STEP_BOUNDARY = 1.06
FREQ_STEP_BASIS = "geometric midpoint of 1.000 and 1.127, sqrt(1.127) = 1.0616, rounded to 1.06"


def fmode(khz, floor_khz):
    """LOW = at the floor clock, HIGH = at a stepped clock. Two classes, no middle: the
    boundary is a single derived threshold and inventing a middle band here would create a
    third outcome no prediction covers."""
    return "HIGH" if khz > floor_khz * FREQ_STEP_BOUNDARY else "LOW"


# ----------------------------------------------------------------- 4. the prediction
#
# H_clock is registered as a POSITIVE prediction, not as a null. The leading hypothesis makes a
# falsifiable claim about the samples and gets to state it in advance.
#
#   H_clock:    an ELEVATED arm's in-arm samples are >= 95% HIGH, and a FLOOR arm's are >= 95% LOW.
#   H_notclock: every arm, elevated or not, reads >= 95% LOW -- the clocks do not separate and the
#               level term is something else (IPC, cache state, an unmodelled co-tenant).
#
# 95% and not 100%: an arm's trace spans its whole wall-clock life including build teardown and the
# gaps between benchmark rows, when the cpu is briefly idle and the governor drops it. Those
# samples are real and are not evidence against a step during the timed region. 5% of ~4750 is
# ~237 samples of slack, which is generous for that and tight for anything else.
PREDICT_FRACTION = 0.95
PREDICT = {
    "elevated": "HIGH >= 95% of samples",
    "floor":    "LOW  >= 95% of samples",
}

# ----------------------------------------------------------------- 5. the falsifier
#
# Registered so that it can fire against the author's own leading hypothesis.
FALSIFIER = (
    "An ELEVATED arm (>= 1.08x its row's floor in GFLOP/s) whose in-arm frequency samples are "
    "< 50% HIGH refutes H_clock for that arm: the work sped up by >= 8% while the clock did not "
    "move, so the term is not a clock term. Between 50% and 95% is MIXED and is reported as "
    "mixed, with the sample fractions printed -- not rounded to a verdict.")

# A secondary quantity, REPORTED AND NOT SCORED: the ratio of an elevated arm's median in-arm
# frequency to the floor's, against that arm's GFLOP/s ratio. If a clock step explains the term
# these agree; if the frequency step is real but too small, the residue is a second mechanism.
# It is not a criterion because turning a two-number agreement into a threshold here would be
# choosing the band after seeing the data on every future run.
SECONDARY = "freq_ratio vs gflops_ratio per elevated arm, printed with both terms, never scored"

# ----------------------------------------------------------------- 6. out of domain
OUT_OF_DOMAIN = [
    "1. ROW-LEVEL ATTRIBUTION. The trace is timestamped but the benchmark log carries no per-row "
    "wall-clock, so a frequency sample cannot be attributed to the row that was running. Every "
    "criterion here is therefore ARM-level. This is why `#150` item 3 -- aref's elevation being "
    "row-selective, present on 4x32 and absent on 2x32 -- is explicitly NOT addressed by this "
    "run, and a null on it must not be read as evidence.",
    "2. WHY a step occurs, if one does. AVX-512 license transitions are the leading candidate on "
    "this silicon and are not tested here; the sampler sees frequency, not instruction mix.",
    "3. THE CONTROL PHASE'S GENERALITY. Perturbation is measured on the `pre` config only. A "
    "sampler cost that depended on the arm's own env would be invisible to it, and nothing here "
    "claims otherwise.",
    "4. WHETHER THE ELEVATION RECURS AT ALL. It appeared in 2 of 10 arms once. This run may "
    "simply not reproduce it, in which case the deliverable is `UNMEASURED` plus a second "
    "independent observation of its absence -- which is itself worth having, since a phenomenon "
    "that does not recur under an identical sequence is a different finding from one that does.",
]
