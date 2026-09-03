#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Derive the quietness guard's PEDESTAL and bound from the tracked record, for `#150`'s run.

`#149`'s ruling: pedestal-subtracted guarding is right for the next registration, on one
condition -- "the pedestal is DERIVED, not assumed. Don't write 1.00 because one pinned thread
'should' read 1.00." This is that derivation. It reads the two tracked campaign logs, reconstructs
each driver's OWN load timeline from its own recorded arm boundaries, and computes what a
between-arm 5-minute load reading should have been if the only load were the driver's.

Why a pedestal at all: `docs/issue148-mech-205a7a8.md` section 8 measured the defect. A flat bound of
1.25 leaves the FIRST arm of a pass ~1.08 of headroom and every later arm ~0.24, because the
driver's own preceding arm contributes ~1.0 that the bound never accounted for. Same guard, two
sensitivities, decided by run position -- which is not a property of any co-tenant.

Two things this must not do:
  * assume the self-term is 1.00. It is measured here, and it is not 1.00.
  * pool the two reads of one instant. The driver samples "after arm N" and "before arm N+1"
    within the same second; those are two genuinely distinct reads (the pid counter moves) of ONE
    instant, so keeping both would give an n wrong in the direction of false confidence. Deduped
    by INSTANT below, and the count of collapsed pairs is printed.

Run: python3 archive/freq150/derive-pedestal.py
"""

import math
import re
import sys
from datetime import datetime, timezone

LOGS = [
    ("core148", "archive/core148/core148-97a21f4.log"),
    ("mech148", "archive/mech148/mech148-205a7a8.log"),
]

# The kernel's 5-minute load average is an exponentially weighted moving average with a 300 s time
# constant. That is not a model of the load; it is the DEFINITION of the quantity being read, so
# applying it to our own measured busy timeline is arithmetic rather than hypothesis.
TAU_S = 300.0

# A sample is SATURATED when the driver's own EWMA term has essentially converged. Only the very
# first gate of a run is below this, by construction -- arms are ~950 s and run back to back.
SATURATED_P = 0.95


def parse(path):
    """-> (samples, arms). samples are (label, epoch, l1, l5, l15, pidfield)."""
    samples, arms = [], []
    label = None
    pend = None
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            m = re.match(r"^-- host sample: (.*) --$", line)
            if m:
                pend = {"label": m.group(1)}
                continue
            if pend is not None:
                m = re.match(r"^\s*loadavg: ([\d.]+) ([\d.]+) ([\d.]+) (\S+)", line)
                if m:
                    pend["l"] = (float(m.group(1)), float(m.group(2)), float(m.group(3)))
                    pend["pid"] = m.group(4)
                    continue
                m = re.match(r"^\s*utc: (\S+)Z", line)
                if m:
                    pend["t"] = datetime.strptime(m.group(1), "%Y-%m-%dT%H:%M:%S").replace(
                        tzinfo=timezone.utc).timestamp()
                    samples.append(pend)
                    pend = None
                    continue
    # Own-busy windows: from a "before X" instant to the matching "after X" instant. The convention
    # includes each arm's own build teardown and host sampling -- a few seconds out of ~950 -- and
    # the LIVE guard in driver-freq150.sh uses the identical convention, which is what makes the
    # self-term derived here transferable to it.
    byl = {s["label"]: s for s in samples}
    for lab, s in byl.items():
        if lab.startswith("before "):
            arm = lab[len("before "):]
            aft = byl.get("after " + arm)
            if aft:
                arms.append((arm, s["t"], aft["t"]))
    arms.sort(key=lambda a: a[1])
    return samples, arms


def dedup(samples):
    """One entry per INSTANT: identical (utc, l1, l5, l15) is one reading of the host, however
    many times it was read. Returns (kept, collapsed_pairs)."""
    seen, kept, collapsed = {}, [], 0
    for s in samples:
        key = (s["t"], s["l"])
        if key in seen:
            seen[key]["also"] = s["label"]
            collapsed += 1
            continue
        seen[key] = s
        s["also"] = None
        kept.append(s)
    return kept, collapsed


def ewma(t, arms, t0):
    """The driver's own contribution to a 5-minute load read at epoch t, stepped over its own
    measured busy timeline at 1 s resolution. n=1 inside an arm (every arm is GOMAXPROCS=1 pinned,
    witnessed per arm by the gomaxprocs readback), 0 in the gaps between them."""
    p, decay = 0.0, math.exp(-1.0 / TAU_S)
    busy = [(a, b) for _, a, b in arms]
    for sec in range(int(t0), int(t)):
        n = 1.0 if any(a <= sec < b for a, b in busy) else 0.0
        p = p * decay + n * (1.0 - decay)
    return p


def main():
    rows = []
    for camp, path in LOGS:
        try:
            samples, arms = parse(path)
        except OSError as exc:
            print("REFUSED: %s" % exc)
            return 2
        kept, collapsed = dedup(samples)
        t0 = min(s["t"] for s in samples)
        print("\n%s: %s" % (camp, path))
        print("  %d reads -> %d distinct instants (%d duplicate reads of an instant collapsed), "
              "%d arms" % (len(samples), len(kept), collapsed, len(arms)))
        for s in kept:
            rows.append({
                "camp": camp, "label": s["label"], "also": s["also"], "t": s["t"],
                "l5": s["l"][1], "p": ewma(s["t"], arms, t0),
                "first": s["t"] == t0,
            })

    # The self-term. min(l5/p) over saturated samples is the SMALLEST per-unit own contribution the
    # record supports; using the smallest underestimates the pedestal, which overestimates the
    # foreign residue, which errs toward FALSE REFUSAL -- the direction #149 ruled recoverable.
    sat = [r for r in rows if r["p"] >= SATURATED_P and r["l5"] < 1.5]
    ratios = sorted(r["l5"] / r["p"] for r in sat)
    self_n = ratios[0]
    print("\nself-term, derived from %d saturated clean instants:" % len(sat))
    print("  l5/P spans %.4f .. %.4f ; SELF_N = min = %.4f  (NOT 1.00, and not assumed)"
          % (ratios[0], ratios[-1], self_n))

    print("\n%-8s %-22s %6s %6s %8s %9s" % ("camp", "instant", "l5", "P", "pedestal", "foreign"))
    clean, positives = [], []
    for r in rows:
        r["ped"] = self_n * r["p"]
        r["foreign"] = r["l5"] - r["ped"]
        tag = ""
        if r["l5"] >= 1.5:
            positives.append(r)
            tag = "  <== known co-tenant excursion"
        elif r["first"] and r["camp"] == "mech148":
            tag = "  <== upper bound only, see note"
        else:
            clean.append(r)
        print("%-8s %-22s %6.2f %6.3f %8.3f %9.3f%s"
              % (r["camp"], r["label"], r["l5"], r["p"], r["ped"], r["foreign"], tag))

    cmax = max(r["foreign"] for r in clean)
    cmin = min(r["foreign"] for r in clean)
    print("\nclean instants (n=%d): foreign spans %.3f .. %.3f" % (len(clean), cmin, cmax))
    print("known positive: foreign = %.3f (%s)"
          % (positives[0]["foreign"], positives[0]["label"]) if positives else "no positive")

    # The bound is DERIVED FROM ITS PREDECESSOR, not chosen freshly against these numbers. Test 3's
    # flat bound was QUIET_L5_MAX=1.25, built before that run from test 2's samples; at saturation
    # (P=1) the pedestal absorbs SELF_N of it, so the residue it allowed was 1.25 - SELF_N. Taking
    # exactly that keeps the guard's sensitivity where a pre-existing derivation put it and changes
    # only its POSITION DEPENDENCE, which is the defect #149 named. Picking a rounder 0.30 here
    # would have made the saturated threshold 1.28 -- looser than the guard it replaces, chosen
    # after seeing the residues it had to clear.
    prev_flat = 1.25
    bound = round(prev_flat - self_n, 2)
    print("\nQUIET_FOREIGN_MAX = %.2f   (= test 3's QUIET_L5_MAX %.2f - SELF_N %.4f)"
          % (bound, prev_flat, self_n))
    print("  headroom over the clean maximum:  %.3f  (%.1fx the clean spread %.3f)"
          % (bound - cmax, (bound - cmax) / (cmax - cmin), cmax - cmin))
    ref = [r for r in clean if r["foreign"] > bound]
    print("  clean instants refused:           %d of %d" % (len(ref), len(clean)))
    print("  known positives refused:          %d of %d, with %.3f to spare"
          % (sum(1 for r in positives if r["foreign"] > bound), len(positives),
             positives[0]["foreign"] - bound if positives else float("nan")))
    print("  invisible to it: a co-tenant sustaining < %.2f of one cpu over five minutes." % bound)
    # ...and the point of the whole exercise, stated as the comparison that motivated it: the
    # equivalent flat bound this guard applies at each run position.
    print("\n  position         old flat bound   this guard's effective L5 bound   headroom")
    for p, what in ((0.0, "first arm (P=0)"), (0.959, "second arm (P=.96)"), (1.0, "saturated (P=1)")):
        eff = bound + self_n * p
        print("  %-16s %14.2f %33.2f %10.2f" % (what, prev_flat, eff, eff - self_n * p))
    print("  Never looser than 1.25 at any position, and 4.6x tighter at the first arm, where"
          "\n  test 3 had ~1.08 of unaccounted headroom.")
    print("\nNOTE on mech148's first instant: its 0.17 is the decay tail of THIS driver's own"
          "\n  earlier launch, which refused at exit 8 (toolchain pin) minutes before. That work is"
          "\n  outside the arm timeline, so P=0 there and the 0.17 is charged entirely to 'foreign'."
          "\n  It is therefore an UPPER BOUND on the foreign load at that instant, excluded from the"
          "\n  clean set rather than allowed to set the bound. The live driver has no such hole: it"
          "\n  records its own pre-arm probe window into the same timeline.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
