# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Test 3's pre-registration for #148, in SAMPLE-SHAPE space.

Committed BEFORE the run, imported by `analyze-mech148.py`, and never edited after the
first arm log exists. This is test 2's convention (`archive/core148/predictions-core148.py`)
with one change forced by test 2's own result: the registration is no longer a band on a
ratio of medians.

WHY SHAPE AND NOT A RATIO. Test 2's published `smt/c0` sequence -- 3.321x, 1.325x, 1.030x,
1.005x -- read as a graded recovery monotone in flopsPerCall, and it is not one. The
per-sample distributions differ in KIND: complete recovery, intermittent recovery, none.
1.325x is the midpoint of a switch and was never a level the arm occupied. Scott's ruling,
2026-09-02: "Ratio-of-medians is what hid section 3 -- twice now -- and rule 24 says
predictions live in the instrument's output space, which for a mode-switching quantity IS
THE SHAPE: exact per-mode counts, n/60."

THE TWO LEGITIMACY CONDITIONS the ruling attached, and where each is met here:

  (1) "the mode boundaries derive in advance from test 2's tracked distributions -- cited,
      not chosen."  BOUNDS below are 1.15x and 1.50x of each row's own pooled `c0` median,
      which are not new constants: both cuts are PUBLISHED in test 2's report at
      `docs/issue148-core-97a21f4.md:121`, and the c0 medians are recomputed here from the
      tracked per-sample logs in `archive/core148/`. Nothing in this file was picked to
      make a prediction come out; every number in BOUNDS has a line number.

  (2) "the predictions are counts against those boundaries, stated per row before the run."
      PREDICT below is exactly that, and the count RULE is arithmetic rather than a
      threshold: test 2 classified 16 arm-rows under these boundaries and 15 of them came
      out perfectly pure (60/0/0 or 0/0/60). So "unimodal" in this apparatus MEANS 60/60,
      and anything else is by definition another shape. There is no band to relitigate.

WHAT THIS BUYS THAT TEST 2's REGISTRATION COULD NOT. The boundaries are ABSOLUTE GFLOP/s
carried over from test 2, not ratios against this run's own reference arm. Test 2's blind
spot was precisely that its `ref` check validated a pooled MEDIAN against admissible
levels, which a partly contaminated arm passes -- and `bref` was partly contaminated. A
boundary that does not divide by this run's reference cannot be moved by that reference's
excursions. The price is a precondition: the boundaries are only meaningful if the binary
and the host are the same, so PRECONDITIONS refuses rather than rescaling.
"""

# --- provenance of every number below -------------------------------------------------
#
# The four `c0` pooled medians, recomputed 2026-09-02 from the 240 tracked samples in
# archive/core148/bench-core148-97a21f4-{a,b}c0.txt (scalar backend, both passes pooled,
# 30 samples per pass per row). They agree to four decimals with the medians published in
# docs/issue148-core-97a21f4.md section 3 and printed by the archived analyzer at
# archive/core148/analysis-core148.out:95,99,96,100.
C0_MEDIAN_GFLOPS = {
    ("2x32", 8):  0.8243,
    ("4x32", 8):  0.8054,
    ("2x32", 32): 0.8494,
    ("4x32", 32): 0.8521,
}

# The two cuts, verbatim from docs/issue148-core-97a21f4.md:121: '("at the `c0` level" =
# within 1.15x of that row's own pooled `c0` median; "recovered" = above 1.5x it.)'
CONFINED_FACTOR  = 1.15
RECOVERED_FACTOR = 1.50

# Derived, so a reader can check the arithmetic without running anything:
# Printed to four decimals as the code yields them, not re-rounded from the medians above:
#   row          CONFINED <=   RECOVERED >
#   2x32/kc=8       0.9479        1.2365
#   4x32/kc=8       0.9262        1.2081
#   2x32/kc=32      0.9768        1.2741
#   4x32/kc=32      0.9799        1.2781
BOUNDS = {k: (CONFINED_FACTOR * v, RECOVERED_FACTOR * v) for k, v in C0_MEDIAN_GFLOPS.items()}

REGISTERED_ROWS = tuple(C0_MEDIAN_GFLOPS)


def mode(gflops, row):
    """The three-way classifier. C = confined, R = recovered, M = neither.

    M exists so the instrument can decline. A shape that is mostly M is a CONTINUUM, which
    test 2 never produced on a one-cpu arm and which no prediction below covers -- it is
    declared out of domain in OUT_OF_DOMAIN rather than absorbed into a nearby verdict.
    """
    lo, hi = BOUNDS[row]
    if gflops <= lo:
        return "C"
    if gflops > hi:
        return "R"
    return "M"


def shape(counts):
    """Name the shape from a (C, M, R) count triple. No thresholds: the names partition."""
    c, m, r = counts["C"], counts["M"], counts["R"]
    if c and not r and not m:
        return "unimodal-at-confined"
    if r and not c and not m:
        return "unimodal-at-recovered"
    if c and r:
        return "bimodal"
    if m > c + r:
        return "continuum"
    return "unimodal-with-tail"


# --- test 2's counts under these exact boundaries, as the citation -------------------
#
# Recomputed 2026-09-02 from archive/core148/'s tracked logs and reproducing the shape
# table published at docs/issue148-core-97a21f4.md:114-121. Printed by the analyzer so a
# reader sees the boundaries reproduce the prior result before trusting them on new data.
# Triples are (C, M, R) out of 60 pooled samples.
TEST2_COUNTS = {
    ("2x32", 8):  {"ref": (0, 0, 60), "c0": (60, 0, 0), "c5": (60, 0, 0), "smt": (0, 0, 60)},
    ("4x32", 8):  {"ref": (0, 0, 60), "c0": (60, 0, 0), "c5": (60, 0, 0), "smt": (12, 28, 20)},
    ("2x32", 32): {"ref": (0, 1, 59), "c0": (60, 0, 0), "c5": (60, 0, 0), "smt": (60, 0, 0)},
    ("4x32", 32): {"ref": (0, 0, 60), "c0": (60, 0, 0), "c5": (60, 0, 0), "smt": (60, 0, 0)},
}

# --- the arms -------------------------------------------------------------------------
#
# Same masks as test 2, so `ref` and `c0` are further draws of arms already characterised.
# Only the environment changes, and only on the one-cpu arms. Note what test 2's `ref`
# actually is: TWO physical cores at GOMAXPROCS=1 (mask 0,1) -- recovery does not need
# eight cpus, it needs a second one.
ARMS = {
    "ref":  {"mask": "0,1", "cores": 2, "env": "GOMAXPROCS=1",
             "role": "recovered control"},
    "c0":   {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1",
             "role": "collapse positive control"},
    "gc":   {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GOGC=off",
             "role": "removes GC assist and worker activity"},
    "pre":  {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1",
             "role": "removes async preemption signals"},
    "both": {"mask": "0",   "cores": 1, "env": "GOMAXPROCS=1 GOGC=off GODEBUG=asyncpreemptoff=1",
             "role": "removes both; the joint arm"},
}

# --- preconditions: the run is uninterpretable if any fails -------------------------
#
# Not scored. A precondition that fails means REFUSE and report `unmeasured`, never a
# verdict scored against a broken denominator.
PRECONDITIONS = {
    # Absolute boundaries carried across runs are only meaningful on the same artifact.
    "binary": "sha256 prefix d0d46d26c15cc8b2, byte-identical to #147's, test 1's and test 2's",
    "host":   "the same 32-logical-cpu host, `performance` governor read back at every sample",
    # The floor is the WORST test 2 gave, not a chosen band: its `ref` scored 60/60/60/59
    # recovered across the four rows, the single non-recovered sample being the documented
    # 1.058 GFLOP/s excursion in `bref` on 2x32/kc=32.
    "ref":    "RECOVERED >= 59/60 on every registered row",
    # Test 2's `c0` and `c5` gave 60/0/0 on all four rows, in both passes: 8 arm-rows, 8
    # perfect. So 60/60 is the observed value and not a rounded-down bar.
    "c0":     "CONFINED 60/60 on every registered row",
}

# --- the scored predictions, per row, per arm, before the run ----------------------
#
# MY PRE-RUN POSITION, stated so it can be wrong: branch C's REMOVABLE part is not the
# mechanism, and all three treatment arms stay confined. The reasoning, which is what is
# actually being tested:
#
#   - GC has almost nothing to do here. The kernel allocates nothing in the K-loop, its
#     panels are pre-sliced and its data is pointer-free -- three standing project rules --
#     so `GOGC=off` should remove work that was not being done.
#   - Async preemption fires on a goroutine running past ~10ms. One signal per 10ms cannot
#     cost 70% of a rate by direct expense, and test 2 already refuted the fixed-cost shape
#     of explanation by its own instrument (`c0 - ref` is 873/1900/3534/7300 ns, doubling
#     with flopsPerCall -- a proportional rate loss, not an overhead).
#
# If that is right, the finding is NOT "C is refuted". It is that C's removable part is
# bounded, which points at what neither arm can touch: sysmon, the scavenger, the
# netpoller, and every kernel-side thing that must run on the one cpu the task is confined
# to -- timer ticks, RCU callbacks, kworkers. Call that candidate D. It is a DIFFERENT
# candidate from branch A, which asked only whether cpu0 is special and died by identity;
# kernel-side work follows the task's cpu whichever cpu that is.
#
# THE FALSIFIER, stated as a count so it cannot be argued afterwards: >= 1/60 RECOVERED in
# any treatment arm on any registered row refutes the position above, and WHICH arm
# attributes it -- `gc` alone or `pre` alone is single-factor, `both` alone is joint.
PREDICT = {
    ("2x32", 8):  {"gc": "unimodal-at-confined", "pre": "unimodal-at-confined",
                   "both": "unimodal-at-confined"},
    ("4x32", 8):  {"gc": "unimodal-at-confined", "pre": "unimodal-at-confined",
                   "both": "unimodal-at-confined"},
    ("2x32", 32): {"gc": "unimodal-at-confined", "pre": "unimodal-at-confined",
                   "both": "unimodal-at-confined"},
    ("4x32", 32): {"gc": "unimodal-at-confined", "pre": "unimodal-at-confined",
                   "both": "unimodal-at-confined"},
}
# Every prediction above is 60/60 CONFINED and 0/60 RECOVERED, by the count rule derived at
# the top of this file. 12 cells, 4 rows x 3 treatment arms.
PREDICT_COUNTS = {row: {arm: (60, 0, 0) for arm in ("gc", "pre", "both")} for row in PREDICT}

# --- what this registration cannot see (DESIGN.md section 5 rule 12) ----------------
OUT_OF_DOMAIN = (
    "No environment variable removes sysmon, the scavenger, the netpoller, or any "
    "kernel-side work on the confined cpu. So a null result BOUNDS branch C's removable "
    "part and does not refute C. The action that would remove the limitation is a patched "
    "runtime, which is out of scope for this campaign and is named here rather than "
    "carried as a debt.",

    "The 4x32/kc=8 row's recovered mode is NOT a tight level -- test 2's `smt` arm spanned "
    "0.835-2.721 GFLOP/s there, 12/28/20. If a treatment arm recovers on that row, the "
    "COUNT is the finding and the level is not, and no prediction here is made about where "
    "in that range its samples land.",

    "A CONTINUUM shape (mostly M) is out of domain: test 2 produced none on any one-cpu "
    "arm, so there is no tracked distribution to derive a prediction from. It would be "
    "reported as unpredicted rather than scored against the nearest named shape.",

    "These boundaries are absolute GFLOP/s from one host and one binary. They transfer to "
    "no other host and no rebuild; PRECONDITIONS refuses instead of rescaling, because a "
    "rescaled boundary is a chosen one.",
)
