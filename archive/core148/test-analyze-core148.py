#!/usr/bin/env python3
"""Fixtures for analyze-core148.py, run before the real data exists.

Every case drives the analyzer to a NAMED outcome, including each refusal. A checker whose
fail paths were never executed is an unread witness: the three branch cases prove it can
name a mechanism, and the six refusals prove it can decline to.
"""
import os
import random
import shutil
import subprocess
import sys

TMP = "/tmp/core148-fixtures"
# Resolved against this file's own directory, never an absolute path under /tmp: these fixtures
# were written beside the analyzer in /tmp and archived afterwards, and a hardcoded /tmp path
# would pass here and fail for every other reader.
ANALYZER = os.path.join(os.path.dirname(os.path.abspath(__file__)), "analyze-core148.py")
REV = "fixture"
ARMS = ["ref", "c0", "c5", "smt"]
CORES = {"ref": "0,1", "c0": "0", "c5": "5", "smt": "0,0"}
MASK = {"ref": "0,1", "c0": "0", "c5": "5", "smt": "0,16"}
NCPU = {"ref": 2, "c0": 1, "c5": 1, "smt": 2}

SCALAR_BASE = {("2x32", 8): 2.90, ("4x32", 8): 3.00, ("2x32", 32): 3.30, ("4x32", 32): 3.30,
               ("2x32", 128): 0.870, ("4x32", 128): 0.845,
               ("2x32", 512): 0.839, ("4x32", 512): 0.813}
AVX_BASE = {("2x32", 8): 55.0, ("4x32", 8): 61.5, ("6x32", 8): 58.0,
            ("2x32", 32): 56.0, ("4x32", 32): 62.0, ("6x32", 32): 59.0,
            ("2x32", 128): 57.0, ("4x32", 128): 63.0, ("6x32", 128): 60.0,
            ("2x32", 512): 57.5, ("4x32", 512): 63.5, ("6x32", 512): 60.5}
REGISTERED = {("2x32", 8), ("4x32", 8), ("2x32", 32), ("4x32", 32)}


def write_arm(d, p, a, factors, cores=None, ncpu=None, pinline=True, avx_factor=1.0):
    rng = random.Random(hash((p, a)) & 0xFFFF)
    path = f"{d}/bench-core148-{REV}-{p}{a}.txt"
    with open(path, "w") as f:
        if pinline:
            c = cores if cores is not None else CORES[a]
            n = ncpu if ncpu is not None else NCPU[a]
            f.write(f"keel-pin: explicit=1 mask={MASK[a]} width={n} cores={c} "
                    f"doms=0,0 nodedoms=1\n")
        f.write("keel-bench-gomaxprocs: 1\n")
        for (shape, kc), base in sorted(AVX_BASE.items()):
            g = base * avx_factor
            for _ in range(30):
                v = g * (1 + rng.uniform(-0.003, 0.003))
                f.write(f"BenchmarkKernel/{shape}/avx512/kc={kc}\t 1000\t"
                        f" {1000/v:.2f} ns/op\t {v:.2f} GFLOP/s\t 2048 flops/call\n")
        for (shape, kc), base in sorted(SCALAR_BASE.items()):
            fac = factors.get((shape, kc), 1.0) if (shape, kc) in REGISTERED else 1.0
            g = base * fac
            for _ in range(30):
                v = g * (1 + rng.uniform(-0.003, 0.003))
                f.write(f"BenchmarkKernel/{shape}/scalar/kc={kc}\t 1000\t"
                        f" {1000/v:.2f} ns/op\t {v:.2f} GFLOP/s\t 2048 flops/call\n")


ALL = {k: 1.0 for k in REGISTERED}
COL = {k: 0.25 for k in REGISTERED}


def build(name, spec, **kw):
    d = f"{TMP}/{name}"
    shutil.rmtree(d, ignore_errors=True)
    os.makedirs(d)
    for p in ("a", "b"):
        for a in ARMS:
            write_arm(d, p, a, spec.get(a, ALL), **kw.get(a, {}))
    return d


def run(d):
    r = subprocess.run([sys.executable, ANALYZER, REV, d], capture_output=True, text=True)
    return r.returncode, r.stdout + r.stderr


CASES = []


def case(name, spec, expect, absent=None, **kw):
    CASES.append((name, spec, expect, absent, kw))


# The three branches, each of which must be named UNIQUELY.
case("branch-C: c0 and c5 collapse, smt intact",
     {"c0": COL, "c5": COL, "smt": ALL},
     ["Branch C is the only one", "runtime"])
case("branch-B: everything on one core collapses",
     {"c0": COL, "c5": COL, "smt": COL},
     ["Branch B is the only one", "whichever core"])
case("branch-A: only cpu0 collapses",
     {"c0": COL, "c5": ALL, "smt": ALL},
     ["Branch A is the only one", "smt=out-of-domain"])
# A pattern no branch predicts: c5 intact (A) but smt collapses (not A, which is ood there,
# and not B or C, which both need c5 to collapse). A scores 4/4 on c0 and 4/4 on c5 and is
# not scored on smt, so A still wins -- which is exactly what "out of domain" MEANS, and this
# case exists to prove the ood cell is doing that rather than being quietly ignored.
case("ood: c5 intact and smt collapsed -- A survives BECAUSE smt is out of domain",
     {"c0": COL, "c5": ALL, "smt": COL},
     ["Branch A is the only one", "smt=out-of-domain"])
# The refusals.
case("refuse: c0 does NOT collapse (the harness failed)",
     {"c0": ALL, "c5": COL, "smt": ALL},
     ["did not collapse", "refutes the HARNESS", "NOT interpretable"])
case("refuse: smt reports TWO cores, so it is not the arm it was built to be",
     {"c0": COL, "c5": COL, "smt": ALL},
     ["distinct=2 want=1", "REFUSED", "NOT interpretable"],
     smt={"cores": "0,16"})
case("refuse: no keel-pin line at all",
     {"c0": COL, "c5": COL, "smt": ALL},
     ["no `keel-pin: explicit=1` line", "UNMEASURED", "NOT interpretable"],
     smt={"pinline": False})
case("refuse: a control row moved (avx512 down 30% in one arm)",
     {"c0": COL, "c5": COL, "smt": ALL},
     ["OUTSIDE", "about the host", "NOT interpretable"],
     smt={"avx_factor": 0.70})
case("refuse: an arm log is absent", {"c0": COL, "c5": COL, "smt": ALL},
     ["arm log(s) absent", "UNMEASURED"], absent="bsmt")


def main():
    shutil.rmtree(TMP, ignore_errors=True)
    os.makedirs(TMP)
    bad = 0
    for i, (name, spec, expect, absent, kw) in enumerate(CASES):
        d = build(f"c{i}", spec, **kw)
        if absent:
            os.remove(f"{d}/bench-core148-{REV}-{absent}.txt")
        rc, out = run(d)
        miss = [e for e in expect if e not in out]
        if miss:
            bad += 1
            print(f"FAIL  {name}\n      missing: {miss}\n      rc={rc}")
            print("\n".join("      | " + l for l in out.splitlines()[-28:]))
        else:
            print(f"ok    {name}  (rc={rc})")
    # The ref-admissible check needs a ref OFF the lottery entirely, which the factor
    # machinery above cannot express (it scales non-ref arms). Driven directly.
    d = build("cref", {"c0": COL, "c5": COL, "smt": ALL})
    for p in ("a", "b"):
        s = open(f"{d}/bench-core148-{REV}-{p}ref.txt").read()
        open(f"{d}/bench-core148-{REV}-{p}ref.txt", "w").write(
            s.replace("2.90 GFLOP/s", "9.99 GFLOP/s").replace("2.89 GFLOP/s", "9.99 GFLOP/s"))
    rc, out = run(d)
    if "admissible" in out and "REFUSED" in out and "NOT interpretable" in out:
        print(f"ok    refuse: ref is off the levels test 1 measured  (rc={rc})")
    else:
        bad += 1
        print(f"FAIL  refuse: ref off the admissible levels\n      rc={rc}")
        print("\n".join("      | " + l for l in out.splitlines()[-24:]))
    # No branch fits: c5 collapses (kills A) and smt lands in the declared indeterminate
    # zone (kills B, which needs collapse, and C, which needs intact). This is the outcome
    # the pre-registration says is a result rather than a failure, so it must say so.
    d = build("cnone", {"c0": COL, "c5": COL, "smt": {k: 0.60 for k in REGISTERED}})
    rc, out = run(d)
    if "NO registered branch" in out and "fourth candidate" in out and "indeterminate row-arms: 4" in out:
        print(f"ok    no branch fits: smt in the indeterminate zone  (rc={rc})")
    else:
        bad += 1
        print(f"FAIL  no branch fits\n      rc={rc}")
        print("\n".join("      | " + l for l in out.splitlines()[-24:]))
    # The two estimators disagree: smt collapses in pass A and is intact in pass B, so
    # max-of-two-passes reads intact while the pooled median lands between the levels. The
    # pre-registration says that is INDETERMINATE and not a tiebreak, so it must not be scored.
    d = build("casym", {"c0": COL, "c5": COL, "smt": ALL})
    write_arm(d, "a", "smt", COL)
    rc, out = run(d)
    if "DISAGREE" in out and "not a tiebreak" in out:
        print(f"ok    estimators disagree on smt: INDETERMINATE, not resolved  (rc={rc})")
    else:
        bad += 1
        print(f"FAIL  estimator disagreement\n      rc={rc}")
        print("\n".join("      | " + l for l in out.splitlines()[-24:]))
    print(f"\n{'GREEN' if not bad else f'RED: {bad} case(s) failed'}"
          f" -- {len(CASES) + 3} cases")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
