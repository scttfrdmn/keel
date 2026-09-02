#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""#148's width sweep, read in the order declared before the run existed.

Section 1 is the CONTROL and it runs first on purpose: if the twelve avx512 rows
moved across the sweep, the eight scalar rows are a reading about the host and the
registered criterion is not interpretable. Section 2 is the DRIFT test the mirrored
pass order buys. Only section 3 reads the registered criterion, and only if 1 and 2
allow it to mean anything.

Usage: python3 archive/width148/analyze-width148.py <rev> [logdir]
"""
import re
import statistics
import sys
import random

ROW = re.compile(
    r"^BenchmarkKernel/(?P<shape>[^/]+)/(?P<backend>[^/]+)/kc=(?P<kc>\d+)\S*\s+"
    r"(?P<n>\d+)\s+(?P<ns>[\d.e+]+) ns/op\s+(?P<gf>[\d.e+]+) GFLOP/s"
)
WIDTHS = [8, 4, 2, 1]
PASSES = ["a", "b"]
# #148's registered prediction is about "the small-kc rows", which in the table it
# publishes are the four rows carrying the 2.9-3.2 GFLOP/s advantage: kc=8 and kc=32
# on both shapes. kc=128/512 never had the advantage and the prediction is not about
# them; they are reported anyway, marked as outside the registered set.
SMALL_KC = {8, 32}
BAND = 1.15
SEED = 20260902


def parse(path):
    rows, skipped = {}, 0
    with open(path) as f:
        for line in f:
            m = ROW.match(line)
            if not m:
                if line.startswith("BenchmarkKernel"):
                    skipped += 1
                continue
            key = (m["shape"], m["backend"], int(m["kc"]))
            rows.setdefault(key, []).append((float(m["ns"]), float(m["gf"])))
    return rows, skipped


def med(samples, idx):
    return statistics.median(s[idx] for s in samples)


def ratio_ci(num, den, idx, reps=10000):
    """Nonparametric CI for the ratio of medians, resampling both arms.

    n=30 per arm and #147 showed within-window bimodality on some of these very
    rows, so the spread is not Gaussian and a t-interval would be a claim about a
    shape nobody measured. Seeded, so the interval is a function of the data.
    """
    rng = random.Random(SEED)
    a = [s[idx] for s in num]
    b = [s[idx] for s in den]
    out = []
    for _ in range(reps):
        ra = statistics.median(rng.choices(a, k=len(a)))
        rb = statistics.median(rng.choices(b, k=len(b)))
        out.append(ra / rb)
    out.sort()
    return out[int(0.025 * reps)], out[int(0.975 * reps)]


def main():
    rev = sys.argv[1] if len(sys.argv) > 1 else "d3c1e82"
    logdir = sys.argv[2] if len(sys.argv) > 2 else "build"
    arms, skipped_total = {}, 0
    for p in PASSES:
        for w in WIDTHS:
            path = f"{logdir}/bench-width148-{rev}-{p}{w}.txt"
            arms[(p, w)], sk = parse(path)
            skipped_total += sk
    keys = sorted(set().union(*[set(v) for v in arms.values()]))
    counts = {len(v) for a in arms.values() for v in a.values()}
    print(f"arms: {len(arms)}  rows per arm: {len(keys)}  samples per row: {sorted(counts)}")
    print(f"unparsed BenchmarkKernel lines: {skipped_total} (must be 0)")
    if skipped_total or counts != {30} or len(keys) != 20:
        print("REFUSED: the logs are not the shape this analysis assumes.")
        return 1

    scal = [k for k in keys if k[1] == "scalar"]
    vec = [k for k in keys if k[1] != "scalar"]

    print("\n=== 1. CONTROL: the twelve avx512 rows, declared before the run ===")
    print("If these move, the scalar reading is about the host. #147 measured every")
    print("avx512 row within 0.97-1.10 of its width-8 median at width 1.")
    worst = (0.0, None)
    for p in PASSES:
        for w in [4, 2, 1]:
            rs = [med(arms[(p, w)][k], 0) / med(arms[(p, 8)][k], 0) for k in vec]
            lo, hi = min(rs), max(rs)
            dev = max(abs(1 - lo), abs(1 - hi))
            if dev > worst[0]:
                worst = (dev, f"pass {p} w{w}")
            print(f"  pass {p}  w{w}/w8 ns ratio over 12 avx512 rows: {lo:.3f}-{hi:.3f}")
    print(f"  worst deviation from 1.000: {worst[0]:.3f} at {worst[1]}")

    print("\n=== 2. DRIFT: pass A (8,4,2,1) against pass B (1,2,4,8) ===")
    print("Within-pass ratios to that pass's own w8 arm. A real width effect reads the")
    print("same in both passes; a drift monotone in time does not, because the two")
    print("passes visit each width at mirrored positions in the clock.")
    for w in [4, 2, 1]:
        ds = []
        for k in keys:
            ra = med(arms[("a", w)][k], 0) / med(arms[("a", 8)][k], 0)
            rb = med(arms[("b", w)][k], 0) / med(arms[("b", 8)][k], 0)
            ds.append(ra / rb)
        print(f"  w{w}: ratio_A/ratio_B over 20 rows: {min(ds):.3f}-{max(ds):.3f}  median {statistics.median(ds):.3f}")
    print("  raw cross-pass agreement at each width (median_A/median_B, same width):")
    for w in WIDTHS:
        rs = [med(arms[("a", w)][k], 0) / med(arms[("b", w)][k], 0) for k in keys]
        print(f"    w{w}: {min(rs):.3f}-{max(rs):.3f}  median {statistics.median(rs):.3f}")

    print("\n=== 3. REGISTERED CRITERION: small-kc scalar rows at width 4 ===")
    print(f'"the small-kc rows read within {BAND}x of their width-8 rate at width 4"')
    print("Rate = the log's own GFLOP/s column, which is the column #148's table published.")
    verdicts = []
    for p in PASSES:
        print(f"  -- pass {p} --")
        for k in sorted(scal, key=lambda k: (k[2], k[0])):
            shape, _, kc = k
            g8 = med(arms[(p, 8)][k], 1)
            g4 = med(arms[(p, 4)][k], 1)
            r = g4 / g8
            lo, hi = ratio_ci(arms[(p, 4)][k], arms[(p, 8)][k], 1)
            inb = (1 / BAND) <= r <= BAND
            tag = "registered" if kc in SMALL_KC else "outside the registered set"
            if kc in SMALL_KC:
                verdicts.append((p, k, r, inb))
            print(f"    {shape}/kc={kc:<3} w8={g8:6.3f} w4={g4:6.3f} GFLOP/s  "
                  f"w4/w8={r:.3f} [{lo:.3f},{hi:.3f}]  "
                  f"{'within' if inb else 'OUTSIDE'} {BAND}x  ({tag})")
    npass = sum(1 for v in verdicts if v[3])
    print(f"  registered rows within {BAND}x at width 4: {npass} of {len(verdicts)}")

    print("\n=== 3b. The SHAPE the criterion was asked about: every width, not just 4 ===")
    print("#148 registered the dichotomy 'monotone in width' vs 'a step between 8 and")
    print("everything else'. Ratios of median GFLOP/s to the same pass's w8 arm:")
    print(f"    {'row':<20} {'a4/a8':>7} {'a2/a8':>7} {'a1/a8':>7}   {'b4/b8':>7} {'b2/b8':>7} {'b1/b8':>7}")
    for backend in ("scalar", "avx512"):
        for k in sorted([x for x in keys if x[1] == backend], key=lambda k: (k[2], k[0])):
            cells = []
            for p in PASSES:
                for w in [4, 2, 1]:
                    cells.append(f"{med(arms[(p, w)][k], 1) / med(arms[(p, 8)][k], 1):7.3f}")
            mark = " *" if (backend == "scalar" and k[2] in SMALL_KC) else ""
            print(f"    {backend+' '+k[0]+'/kc='+str(k[2]):<20} {'  '.join(cells)}{mark}")
    print("    (* = a row in the registered set)")

    print("\n=== 3c. The rows at the extremes of section 2's ranges ===")
    print("A range is not a finding until the row holding its end is named.")
    for w in [4, 2, 1]:
        ds = []
        for k in keys:
            ra = med(arms[("a", w)][k], 0) / med(arms[("a", 8)][k], 0)
            rb = med(arms[("b", w)][k], 0) / med(arms[("b", 8)][k], 0)
            ds.append((ra / rb, k, ra, rb))
        ds.sort()
        for tag, (v, k, ra, rb) in (("min", ds[0]), ("max", ds[-1])):
            print(f"  w{w} {tag}: {v:.3f}  {k[1]} {k[0]}/kc={k[2]}  ratio_A={ra:.3f} ratio_B={rb:.3f}")

    print("\n=== 3d. The level structure behind those extremes (unregistered) ===")
    print("Per-arm medians of the four registered rows, sorted, to show whether the")
    print("spread is a continuum or a small set of discrete levels visited per process.")
    for k in sorted([x for x in scal if x[2] in SMALL_KC], key=lambda k: (k[2], k[0])):
        vals = sorted((med(arms[(p, w)][k], 1), f"{p}{w}") for p in PASSES for w in WIDTHS)
        big = [f"{v:.3f}({a})" for v, a in vals if v > 1.5]
        small = [f"{v:.3f}({a})" for v, a in vals if v <= 1.5]
        print(f"  {k[0]}/kc={k[2]:<3} above 1.5: {' '.join(big)}")
        print(f"  {'':<{len(k[0]) + 6}} at/below 1.5: {' '.join(small)}")

    print("\n=== 4. The full sweep, median GFLOP/s, both passes ===")
    for backend in ("scalar", "avx512"):
        print(f"  -- {backend} --")
        hdr = "  ".join(f"{p}{w:<6}" for p in PASSES for w in WIDTHS)
        print(f"    {'row':<16} {hdr}")
        for k in sorted([x for x in keys if x[1] == backend], key=lambda k: (k[0], k[2])):
            cells = "  ".join(f"{med(arms[(p, w)][k], 1):<7.3f}" for p in PASSES for w in WIDTHS)
            print(f"    {k[0]+'/kc='+str(k[2]):<16} {cells}")

    print("\n=== 5. Against #147's own two arms (same binary, sha256=d0d46d26c15cc8b2) ===")
    print("#147 published these medians; this sweep re-draws the same widths from the")
    print("same artifact, so a difference here is a between-draw difference and nothing else.")
    prior = {("2x32", 8): (2.926, 0.772), ("2x32", 32): (2.976, 0.801),
             ("2x32", 128): (0.870, 0.827), ("2x32", 512): (0.839, 0.838),
             ("4x32", 8): (3.184, 0.759), ("4x32", 32): (3.206, 0.804),
             ("4x32", 128): (0.845, 0.818), ("4x32", 512): (0.813, 0.807)}
    print(f"    {'row':<16} {'#147 w8':>8} {'a8':>8} {'b8':>8}   {'#147 w1':>8} {'a1':>8} {'b1':>8}")
    for (shape, kc), (p8, p1) in sorted(prior.items(), key=lambda t: (t[0][0], t[0][1])):
        k = (shape, "scalar", kc)
        print(f"    {shape+'/kc='+str(kc):<16} {p8:8.3f} {med(arms[('a',8)][k],1):8.3f} "
              f"{med(arms[('b',8)][k],1):8.3f}   {p1:8.3f} {med(arms[('a',1)][k],1):8.3f} "
              f"{med(arms[('b',1)][k],1):8.3f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
