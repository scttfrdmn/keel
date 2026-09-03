#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Scores test 3's twelve registered cells against `predictions-mech148.py`.

Reads three kinds of input and never mixes their authority:

  1. the REGISTRATION, `archive/mech148/predictions-mech148.py`, imported and not reimplemented;
  2. the DRIVER LOG, which is the only authority on whether an arm was measured at all;
  3. the ARM LOGS, which supply samples and are consulted only for arms the driver log admits.

Order matters. An arm log exists on disk whether or not its arm ran to completion, so asking the
samples whether they are trustworthy asks the wrong witness -- the driver already recorded a
verdict per arm and this program keys to the driver's own phrases, one branch per phrase, and
fails closed when a label carries no recognised phrase at all.
"""
import importlib.util
import pathlib
import re
import sys
from collections import defaultdict

def _root():
    """Walk up to the repo root rather than hardcoding it, so this file works from
    /tmp while a run is frozen and from archive/mech148/ after it lands."""
    for d in [pathlib.Path.cwd(), *pathlib.Path(__file__).resolve().parents]:
        for cand in (d, *d.parents):
            if (cand / "archive" / "mech148" / "predictions-mech148.py").exists():
                return cand
    sys.exit("could not locate the keel repo root from cwd or this file's path")


ROOT = _root()
REV = sys.argv[1] if len(sys.argv) > 1 else "205a7a8"

spec = importlib.util.spec_from_file_location(
    "pred", ROOT / "archive" / "mech148" / "predictions-mech148.py")
P = importlib.util.module_from_spec(spec)
spec.loader.exec_module(P)

# ---------------------------------------------------------------- parsing
# One scalar row, e.g.
#   BenchmarkKernel/4x32/scalar/kc=8   1874432   641.7 ns/op   3.192 GFLOP/s   2048 flops/call
ROW_RE = re.compile(
    r"^BenchmarkKernel/(?P<shape>\d+x\d+)/scalar/kc=(?P<kc>\d+)\S*\s+"
    r"\d+\s+(?P<ns>[\d.]+) ns/op\s+(?P<gflops>[\d.]+) GFLOP/s")


def find(name):
    """Prefer the ARCHIVED copy over the live one in build/.

    build/ is gitignored, so an analysis that can only read build/ stops being reproducible
    the moment the output directory is cleaned -- and every number this program prints is
    cited in a report. archive/mech148/ is the tracked evidence; build/ is where a fresh run
    lands before it is archived. Checking archive first also means re-running this after a
    later run cannot silently rescore the published one against new samples.
    """
    for d in (ROOT / "archive" / "mech148", ROOT / "build"):
        p = d / name
        if p.exists():
            return p
    return ROOT / "build" / name


def samples(path):
    """(row -> [(gflops, printed_string)]). The printed string is kept because the
    classifier's boundaries were derived on the printed column, so the print quantum is
    part of the instrument and is disclosed rather than assumed away."""
    out = defaultdict(list)
    if not path.exists():
        return out
    for line in path.read_text().splitlines():
        m = ROW_RE.match(line)
        if m:
            row = (m.group("shape"), int(m.group("kc")))
            out[row].append((float(m.group("gflops")), m.group("gflops")))
    return out


def quantum(printed):
    """The width of one printed step, read off the rendering itself rather than assumed."""
    return 10.0 ** -len(printed.split(".")[1]) if "." in printed else 1.0


# ---------------------------------------------------------------- 1. positive control
# Before these boundaries are trusted on new data they must reproduce the published table they
# were derived from. This is the instrument's positive control: it asks whether the classifier
# can see a signal that is known to be there, which is a different question from whether this
# run's treatments arrived.
print("=" * 78)
print("1. POSITIVE CONTROL: do the boundaries reproduce test 2's published shape table?")
print("=" * 78)
ctl_ok, ctl_seen = True, 0
for row in P.REGISTERED_ROWS:
    for arm, want in P.TEST2_COUNTS[row].items():
        pooled = []
        for p in ("a", "b"):
            f = ROOT / "archive" / "core148" / f"bench-core148-97a21f4-{p}{arm}.txt"
            pooled += samples(f)[row]
        if not pooled:
            print(f"   {row[0]}/kc={row[1]:<3} {arm:<4} NO TRACKED SAMPLES -- control cannot run")
            ctl_ok = False
            continue
        got = {"C": 0, "M": 0, "R": 0}
        for g, _ in pooled:
            got[P.mode(g, row)] += 1
        trip = (got["C"], got["M"], got["R"])
        ctl_seen += 1
        if trip != want:
            print(f"   {row[0]}/kc={row[1]:<3} {arm:<4} got {trip} want {want}  MISMATCH")
            ctl_ok = False
if ctl_ok:
    print(f"   all {ctl_seen} of test 2's arm-rows reproduce exactly, so the classifier is")
    print("   known to separate a signal that is known to be present.")
else:
    print("   REFUSING to score: the classifier does not reproduce its own derivation.")
    sys.exit(4)

# ---------------------------------------------------------------- 2. admissibility
print()
print("=" * 78)
print("2. ADMISSIBILITY, from the driver log and nowhere else")
print("=" * 78)
dlog = find(f"mech148-{REV}.log")
if not dlog.exists():
    print(f"   no driver log at {dlog} -- nothing is measured, and that is not a result")
    sys.exit(5)
text = dlog.read_text()
# The driver's say() prints `\n=== %s ===\n`, so its terminal marker is the LINE
# `=== done ===`, and the file's last line is a timestamp printed after it. Keying this to
# the bare word `done`, or to what the file ends with, matched neither -- and reported a
# finished run as INCOMPLETE. Fails closed: no marker means not complete.
complete = re.search(r"^=== done ===$", text, re.M) is not None

verdict = {}   # label -> (state, detail)
for label in [p + a for p in ("a", "b") for a in P.ARMS]:
    esc = re.escape(label)
    # One branch per driver phrase. A label matching none is INCOMPLETE, never "fine".
    if re.search(rf"^arm {esc}: UNMEASURED -- host not quiet: (.*)$", text, re.M):
        why = re.search(rf"^arm {esc}: UNMEASURED -- host not quiet: (.*)$", text, re.M).group(1)
        verdict[label] = ("UNMEASURED", f"host not quiet: {why}")
    elif re.search(rf"^arm {esc}: UNMEASURED -- the far side never reported", text, re.M):
        verdict[label] = ("UNMEASURED", "far side never reported a status")
    elif re.search(rf"^arm {esc}: WARNING, gomaxprocs readback", text, re.M):
        verdict[label] = ("UNMEASURED", "gomaxprocs canary failed: treatment did not arrive")
    elif re.search(rf"^arm {esc}: WARNING, cores readback", text, re.M):
        verdict[label] = ("UNMEASURED", "cores readback wrong: arm did not measure its design")
    elif (re.search(rf"^arm {esc}: gomaxprocs readback OK: 1", text, re.M)
          and re.search(rf"^arm {esc}: cores readback OK", text, re.M)):
        verdict[label] = ("measured", "both readbacks OK")
    elif re.search(rf"^=== arm {esc}:", text, re.M):
        verdict[label] = ("INCOMPLETE", "started, no terminal verdict in the log")
    else:
        verdict[label] = ("INCOMPLETE", "never started")

for label in sorted(verdict):
    st, d = verdict[label]
    print(f"   {label:<6} {st:<11} {d}")
if not complete:
    print("   NOTE: the driver log carries no `=== done ===` marker, so this is a PARTIAL")
    print("   reading of a run that did not finish -- it may be in flight, or it may have been")
    print("   killed. Cells below are scored against the arms admitted so far, and an absent")
    print("   arm is an absent measurement rather than a slow one.")

# ---------------------------------------------------------------- 3. pooled shapes
def cell(arm):
    """Pool the admissible passes of one arm. Returns (per-row counts, n, passes used)."""
    used = [p for p in ("a", "b") if verdict[p + arm][0] == "measured"]
    per = {}
    for row in P.REGISTERED_ROWS:
        pooled = []
        for p in used:
            pooled += samples(find(f"bench-mech148-{REV}-{p}{arm}.txt"))[row]
        counts = {"C": 0, "M": 0, "R": 0}
        amb = 0
        lo, hi = P.BOUNDS[row]
        for g, s in pooled:
            counts[P.mode(g, row)] += 1
            q = quantum(s) / 2.0
            if abs(g - lo) <= q or abs(g - hi) <= q:
                amb += 1
        vals = [g for g, _ in pooled]
        per[row] = (counts, len(pooled), amb, (min(vals), max(vals)) if vals else None)
    return per, used


cells = {arm: cell(arm) for arm in P.ARMS}

print()
print("=" * 78)
print("3. PRECONDITIONS -- not scored; if these fail the twelve cells are uninterpretable")
print("=" * 78)
pre_ok, pre_pending = True, False
for arm, need in (("ref", "R >= 59/60"), ("c0", "C == 60/60")):
    per, used = cells[arm]
    print(f"   {arm} ({P.ARMS[arm]['role']}), registered: {P.PRECONDITIONS[arm]}")
    if not used:
        print(f"      NO ADMISSIBLE PASS -- the {arm} precondition is unmeasured, not failed")
        pre_ok = False
        continue
    for row in P.REGISTERED_ROWS:
        c, n, amb, rng = per[row]
        trip = (c["C"], c["M"], c["R"])
        if arm == "ref":
            good = c["R"] >= 59 and n == 60
        else:
            good = c["C"] == 60 and n == 60
        # A short n because a pass has not run yet is PENDING, not a failure. Printing `!!`
        # for both would make an in-flight run and a broken one look the same.
        pending = (not good and n
                   and all(verdict[p + arm][0] == "INCOMPLETE" or verdict[p + arm][0] == "measured"
                           for p in ("a", "b"))
                   and any(verdict[p + arm][0] == "INCOMPLETE" for p in ("a", "b")))
        mark = "OK " if good else (".. " if pending else "!! ")
        pre_ok = pre_ok and (good or pending)
        pre_pending = pre_pending or bool(pending)
        rs = f"{rng[0]:.3f}-{rng[1]:.3f}" if rng else "no samples"
        print(f"      {mark}{row[0]}/kc={row[1]:<3} (C,M,R)={trip} n={n} range {rs}"
              f"{'  passes ' + ''.join(used) if n != 60 else ''}")
if not pre_ok:
    print("   preconditions: NOT ALL MET (see !! above)")
elif pre_pending or not complete:
    print("   preconditions: nothing failed, but passes are still outstanding (..) -- INCOMPLETE")
else:
    print("   preconditions: ALL MET")

# ---------------------------------------------------------------- 4. the twelve cells
print()
print("=" * 78)
print("4. THE TWELVE REGISTERED CELLS, each against its own (60,0,0)")
print("=" * 78)
print("   OOD = out of domain for the GC hypothesis by OUT_OF_DOMAIN item 5 (the GOGC")
print("   treatment arrives and has nothing to remove: 0 heap-triggered cycles either way).")
print()
print(f"   {'row':<12} {'arm':<5} {'predicted':<12} {'observed':<14} {'n':>3} "
      f"{'shape':<22} verdict")
tally = defaultdict(int)
for row in P.REGISTERED_ROWS:
    for arm in ("gc", "pre", "both"):
        per, used = cells[arm]
        c, n, amb, rng = per[row]
        trip = (c["C"], c["M"], c["R"])
        want_shape = P.PREDICT[row][arm]
        want_counts = P.PREDICT_COUNTS[row][arm]
        if not used:
            st, d = verdict["a" + arm]
            v = f"UNMEASURED ({d})"
        elif trip == want_counts:
            v = "CONFIRMED"
        elif P.shape(c) == want_shape:
            v = f"shape confirmed at n={n}, counts differ"
        else:
            v = f"REFUTED (registered {want_shape})"
        # OOD is a property of the ARM's hypothesis reach, not of how the cell came out, so it
        # is attached on arm membership and to every non-empty verdict. Keying it to the string
        # "CONFIRMED" dropped the label the moment a gc cell landed on any other verdict, and a
        # gc row printed without it reads as evidence about GC -- the exact misreading item 5
        # exists to prevent.
        if used and arm in ("gc", "both"):
            v += ", OOD for GC"
        tally[v.split(" (")[0].split(",")[0]] += 1
        rs = f"{rng[0]:.3f}-{rng[1]:.3f}" if rng else "-"
        print(f"   {row[0] + '/kc=' + str(row[1]):<12} {arm:<5} {str(want_counts):<12} "
              f"{str(trip):<14} {n:>3} {P.shape(c) if n else '-':<22} {v}")
        if amb:
            print(f"   {'':<12} {'':<5} {amb} of {n} samples sit within half a print quantum "
                  f"of a boundary (range {rs})")

print()
print("   tally: " + ", ".join(f"{k}={v}" for k, v in sorted(tally.items())))
print(f"   cells: {sum(tally.values())} (registration declares 12)")

# ---------------------------------------------------------------- 5. the falsifier
print()
print("=" * 78)
print("5. THE REGISTERED FALSIFIER: >= 1/60 RECOVERED in any treatment arm on any row")
print("=" * 78)
hits = []
scored = 0
for row in P.REGISTERED_ROWS:
    for arm in ("gc", "pre", "both"):
        c, n, _, _ = cells[arm][0][row]
        if not n:
            continue
        scored += 1
        if c["R"]:
            hits.append((row, arm, c["R"], n))
if hits:
    for row, arm, r, n in hits:
        print(f"   {row[0]}/kc={row[1]} {arm}: {r}/{n} RECOVERED -- attribution: "
              f"{'single-factor' if arm in ('gc', 'pre') else 'joint'}")
elif scored == 0:
    # Rule 26's own failure mode, turned on this program: a falsifier that examined nothing
    # prints the same silence as a falsifier that examined everything and found nothing. The
    # registered prediction here IS the null, so "did not fire" is the author's own hoped-for
    # answer and must never be reachable from an empty sample.
    print("   UNMEASURED: no treatment cell has a single sample, so the falsifier was not")
    print("   evaluated. This is NOT 'the falsifier did not fire' -- under a null prediction")
    print("   those two readings are indistinguishable in the output and opposite in meaning")
    print("   (DESIGN.md section 5 rule 26).")
else:
    print(f"   no RECOVERED sample in any admissible treatment cell ({scored} of 12 cells")
    print("   carried samples) on any registered row.")
    print("   Per the registration this does NOT refute branch C; it bounds C's removable")
    print("   part, and OUT_OF_DOMAIN item 1 names what no environment variable can reach.")
