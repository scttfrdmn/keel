#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""#148's test 2, read in the order declared before the run existed.

Every threshold this script applies is IMPORTED from predictions-core148.py and none is
written here. Section 0 prints them with the git hash of the module they came from, so a
reader can check that the numbers being applied are the numbers that were committed before
the data existed rather than numbers this file agrees with.

Sections 1 and 2 can REFUSE. The registered criterion in section 3 is not read at all unless
the run is interpretable: the positive control must collapse, `ref` must land on a level test 1
measured, every arm's `cores=` readback must match what its arm was built to be, and the 16
control rows must be intact. Section 4 scores the three branches only against what 3 produced.

Usage: python3 archive/core148/analyze-core148.py <rev> [logdir]
"""
import importlib.util
import os
import re
import statistics
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))

ROW = re.compile(
    r"^BenchmarkKernel/(?P<shape>[^/]+)/(?P<backend>[^/]+)/kc=(?P<kc>\d+)\S*\s+"
    r"(?P<n>\d+)\s+(?P<ns>[\d.e+]+) ns/op\s+(?P<gf>[\d.e+]+) GFLOP/s"
)
# The readback the whole fourth arm rests on. Anchored on `explicit=1` deliberately: a line
# without it came from keel_pin_mask's width path, which means this arm is not the arm the
# driver asked for and the analyzer must not read it as one.
PIN = re.compile(r"^keel-pin: explicit=1 mask=(?P<mask>[\d,]+) width=(?P<w>\d+) "
                 r"cores=(?P<cores>[\d,]+) doms=(?P<doms>[\d,]+) nodedoms=(?P<nd>\d+)")
PASSES = ["a", "b"]


def load_predictions():
    path = os.path.join(HERE, "predictions-core148.py")
    spec = importlib.util.spec_from_file_location("predictions_core148", path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    try:
        h = subprocess.run(["git", "log", "-1", "--format=%h %cI", "--", path],
                           capture_output=True, text=True, cwd=HERE).stdout.strip()
    except OSError:
        h = "<git unavailable>"
    return mod, path, h


def parse(path):
    rows, skipped, pin = {}, 0, None
    with open(path) as f:
        for line in f:
            m = PIN.match(line)
            if m:
                pin = m
                continue
            m = ROW.match(line)
            if not m:
                if line.startswith("BenchmarkKernel"):
                    skipped += 1
                continue
            rows.setdefault((m["shape"], m["backend"], int(m["kc"])), []).append(float(m["gf"]))
    return rows, skipped, pin


def band(v, lo, hi):
    return lo <= v <= hi


def classify(v, P):
    if band(v, *P.COLLAPSE):
        return "collapse"
    if band(v, *P.INTACT):
        return "intact"
    return "indeterminate"


def main():
    P, ppath, phash = load_predictions()
    rev = sys.argv[1] if len(sys.argv) > 1 else "97a21f4"
    logdir = sys.argv[2] if len(sys.argv) > 2 else "build"

    print("=== 0. THE IMPORTED THRESHOLDS, and where they came from ===")
    print(f"  module: {os.path.relpath(ppath, os.path.join(HERE, '..', '..'))}")
    print(f"  committed at: {phash}")
    print(f"  collapse band {P.COLLAPSE}   intact band {P.INTACT}")
    print(f"  estimator: {P.ESTIMATOR} (primary), {P.ESTIMATOR_SECONDARY} (secondary, must agree)")
    print(f"  registered rows: {len(P.REGISTERED)}   arms: {P.ARMS}   reference: {P.REF}")
    print(f"  positive control: {P.POSITIVE_CONTROL}   ref admissible slack: {P.REF_ADMISSIBLE_SLACK}")
    for b, d in sorted(P.BRANCHES.items()):
        cells = "  ".join(f"{a}={d[a]}" for a in P.ARMS)
        print(f"  branch {b}: {cells}")
        print(f"            {d['name']}")

    print("\n=== 1. SHAPE: is this the run the analysis assumes? ===")
    arms, pins, skipped_total, missing = {}, {}, 0, []
    for p in PASSES:
        for a in P.ARMS:
            path = f"{logdir}/bench-core148-{rev}-{p}{a}.txt"
            if not os.path.exists(path):
                missing.append(os.path.basename(path))
                continue
            arms[(p, a)], sk, pin = parse(path)
            pins[(p, a)] = pin
            skipped_total += sk
    if missing:
        print(f"  REFUSED: {len(missing)} arm log(s) absent: {' '.join(missing)}")
        print("  An absent arm is UNMEASURED. No verdict is computed from the arms that ran.")
        return 1
    keys = sorted(set().union(*[set(v) for v in arms.values()]))
    counts = {len(v) for a in arms.values() for v in a.values()}
    print(f"  arms: {len(arms)} (expect 8)   rows per arm: {len(keys)} (expect 20)   "
          f"samples per row: {sorted(counts)} (expect [30])")
    print(f"  unparsed BenchmarkKernel lines: {skipped_total} (must be 0)")
    if skipped_total or counts != {30} or len(keys) != 20 or len(arms) != 8:
        print("  REFUSED: the logs are not the shape this analysis assumes.")
        return 1

    print("\n=== 2. IS THE RUN INTERPRETABLE? Four checks, any of which refuses ===")
    ok = True

    print("  2a. the cores= readback, per arm, against what the arm was BUILT to be")
    print("      smt must report ONE physical core and ref must report TWO. This is the")
    print("      discriminator itself, taken from the log rather than from the request.")
    want = {"ref": 2, "c0": 1, "c5": 1, "smt": 1}
    for p in PASSES:
        for a in P.ARMS:
            pin = pins[(p, a)]
            if pin is None:
                print(f"      {p}{a}: REFUSED, no `keel-pin: explicit=1` line in the log. This arm's "
                      f"mask is unknown, so it is UNMEASURED rather than assumed.")
                ok = False
                continue
            n = len(set(pin["cores"].split(",")))
            good = n == want[a]
            print(f"      {p}{a}: mask={pin['mask']:<6} cores={pin['cores']:<6} "
                  f"distinct={n} want={want[a]} nodedoms={pin['nd']}  {'ok' if good else 'REFUSED'}")
            ok = ok and good

    def pooled(a, k):
        return statistics.median(arms[("a", a)][k] + arms[("b", a)][k])

    def maxpass(a, k):
        return max(statistics.median(arms[("a", a)][k]), statistics.median(arms[("b", a)][k]))

    EST = {"pooled-median": pooled, "max-of-two-pass-medians": maxpass}
    est = EST[P.ESTIMATOR]
    est2 = EST[P.ESTIMATOR_SECONDARY]
    reg = [(s, b, kc) for (s, b, kc) in P.REGISTERED]
    ctl = [k for k in keys if k not in reg]

    print("\n  2b. the POSITIVE CONTROL: every branch predicts c0 collapses")
    for k in reg:
        r = est(P.POSITIVE_CONTROL, k) / est(P.REF, k)
        c = classify(r, P)
        print(f"      {k[0]}/kc={k[2]:<3} c0/ref = {r:.3f}  {c}")
        if c != "collapse":
            ok = False
    if not ok:
        print("      REFUSED: c0 did not collapse. That refutes the HARNESS -- the explicit path")
        print("      did not reproduce test 1's width-1 arm -- and no other arm can be read.")

    print("\n  2c. does ref land on a level test 1 measured? (admissible LEVELS, not one median)")
    slack = P.REF_ADMISSIBLE_SLACK
    for (shape, kc), (lo, hi) in sorted(P.REF_ADMISSIBLE_GFLOPS.items()):
        k = (shape, "scalar", kc)
        v = est(P.REF, k)
        good = (lo * (1 - slack)) <= v <= (hi * (1 + slack))
        print(f"      {shape}/kc={kc:<3} ref={v:6.3f}  admissible [{lo:.3f},{hi:.3f}] "
              f"+-{slack:.0%} -> [{lo*(1-slack):.3f},{hi*(1+slack):.3f}]  {'ok' if good else 'REFUSED'}")
        ok = ok and good

    print(f"\n  2d. the {len(ctl)} CONTROL rows must be intact in all 8 arm-passes")
    worst = (1.0, None)
    nbad = 0
    for k in ctl:
        for a in P.ARMS:
            if a == P.REF:
                continue
            r = est(a, k) / est(P.REF, k)
            if not band(r, *P.INTACT):
                nbad += 1
                print(f"      {k[1]} {k[0]}/kc={k[2]} {a}/ref = {r:.3f}  OUTSIDE {P.INTACT}")
            if abs(1 - r) > abs(1 - worst[0]):
                worst = (r, f"{k[1]} {k[0]}/kc={k[2]} {a}")
    print(f"      control rows outside the intact band: {nbad} (must be 0)")
    print(f"      worst control deviation from 1.000: {worst[0]:.3f} at {worst[1]}")
    if nbad:
        print("      REFUSED: a control row moved, so the registered reading is about the host.")
        ok = False

    if not ok:
        print("\n=== VERDICT: the run is NOT interpretable. No branch is scored. ===")
        print("Section 2 refused, and refusing is the point: a mechanism named from an")
        print("uninterpretable run would be a story, not a measurement.")
        return 1
    print("\n  all four checks pass: the registered criterion can be read.")

    print("\n=== 3. THE REGISTERED CRITERION, per row, per arm, both estimators ===")
    print(f"  ratio of an arm's estimator to the same run's {P.REF} arm.")
    print(f"  collapse = {P.COLLAPSE}, intact = {P.INTACT}, anything else INDETERMINATE.")
    hdr = "".join(f"{a:>22}" for a in P.ARMS if a != P.REF)
    print(f"    {'row':<16}{hdr}")
    obs = {}
    for k in sorted(reg, key=lambda x: (x[2], x[0])):
        cells = []
        for a in P.ARMS:
            if a == P.REF:
                continue
            r1 = est(a, k) / est(P.REF, k)
            r2 = est2(a, k) / est2(P.REF, k)
            c1, c2 = classify(r1, P), classify(r2, P)
            agree = c1 == c2
            obs[(k, a)] = c1 if agree else "indeterminate"
            cells.append(f"{r1:6.3f}/{r2:6.3f} {c1[:4] if agree else 'DISAGREE':>8}")
        print(f"    {k[0]+'/kc='+str(k[2]):<16}{''.join(cells)}")
    print("    (primary/secondary; a disagreement between estimators is INDETERMINATE, not a tiebreak)")

    print("\n  3b. per-pass, so the mirrored order can be read as the drift check it is")
    for a in P.ARMS:
        if a == P.REF:
            continue
        rs = []
        for k in sorted(reg, key=lambda x: (x[2], x[0])):
            ra = statistics.median(arms[("a", a)][k]) / statistics.median(arms[("a", P.REF)][k])
            rb = statistics.median(arms[("b", a)][k]) / statistics.median(arms[("b", P.REF)][k])
            rs.append((k, ra, rb))
        cells = "  ".join(f"{k[0]}/kc={k[2]}: {ra:.3f}|{rb:.3f}" for k, ra, rb in rs)
        print(f"      {a:<4} {cells}")
    print("      (pass A | pass B, each to its own pass's ref. A real effect agrees; drift does not.)")

    print("\n=== 4. WHICH BRANCH, scored against section 3 and nothing else ===")
    scored = {}
    for b, d in sorted(P.BRANCHES.items()):
        hits = tot = 0
        detail = []
        for a in P.ARMS:
            if a == P.REF:
                continue
            pred = d[a]
            if pred == "ood":
                detail.append(f"{a}=out-of-domain")
                continue
            got = [obs[(k, a)] for k in reg]
            agree = sum(1 for g in got if g == pred)
            hits += agree
            tot += len(got)
            detail.append(f"{a}: predicted {pred}, {agree}/{len(got)} rows agree")
        scored[b] = (hits, tot)
        print(f"  branch {b} -- {d['name']}")
        print(f"    {hits}/{tot} registered row-arms agree")
        for x in detail:
            print(f"      {x}")

    best = [b for b, (h, t) in scored.items() if t and h == t]
    print("\n=== VERDICT ===")
    if len(best) == 1:
        b = best[0]
        print(f"  Branch {b} is the only one every registered row-arm agrees with:")
        print(f"    {P.BRANCHES[b]['name']}")
        others = [f"{o} ({scored[o][0]}/{scored[o][1]})" for o in sorted(scored) if o != b]
        print(f"  The others are refuted at: {', '.join(others)}")
    elif len(best) > 1:
        print(f"  {len(best)} branches survive ({', '.join(sorted(best))}), so this run did not")
        print("  separate them. The arms that would have are named in the truth table.")
    else:
        print("  NO registered branch is fully consistent with the data.")
        print("  That is a result, not a failure: the mechanism is none of the three as stated,")
        print("  and the pattern below is what a fourth candidate has to explain.")
        for b, (h, t) in sorted(scored.items()):
            print(f"    branch {b}: {h}/{t}")
    ind = sum(1 for v in obs.values() if v == "indeterminate")
    print(f"  indeterminate row-arms: {ind} of {len(obs)}"
          + ("  (each one is a cell no branch was scored on)" if ind else ""))

    print("\n=== 5. THE FULL SWEEP, median GFLOP/s per arm-pass ===")
    for backend in ("scalar", "avx512"):
        print(f"  -- {backend} --")
        h = "  ".join(f"{p}{a:<5}" for p in PASSES for a in P.ARMS)
        print(f"    {'row':<16} {h}")
        for k in sorted([x for x in keys if x[1] == backend], key=lambda x: (x[0], x[2])):
            cells = "  ".join(f"{statistics.median(arms[(p, a)][k]):<6.3f}"
                              for p in PASSES for a in P.ARMS)
            mark = " *" if k in reg else ""
            print(f"    {k[0]+'/kc='+str(k[2]):<16} {cells}{mark}")
    print("    (* = a row in the registered set)")

    print("\n=== 6. THE LEVEL STRUCTURE in this run's own arms (unregistered) ===")
    print("  Per-arm medians of the registered rows, sorted, so the discrete levels that")
    print("  #148 section 6 and #147 documented can be seen in THIS run rather than assumed")
    print("  from the last one. The estimator was chosen against them; this is the check that")
    print("  the structure is still the shape that choice was made for.")
    for k in sorted(reg, key=lambda x: (x[2], x[0])):
        vals = sorted((statistics.median(arms[(p, a)][k]), f"{p}{a}") for p in PASSES for a in P.ARMS)
        print(f"    {k[0]}/kc={k[2]:<3} " + "  ".join(f"{v:.3f}({t})" for v, t in vals))
    return 0


if __name__ == "__main__":
    sys.exit(main())
