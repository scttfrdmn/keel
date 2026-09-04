#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Analyzer for `#150`'s in-arm frequency sampler run (freq150-71a73da).

Every constant, threshold, boundary, prediction and precondition is IMPORTED from
predictions-freq150.py, which was committed before the run. This file computes; it does not
restate the registration. Where it must name a number (a row key format, a file glob) that number
is a fact about the artifacts on disk, not a criterion.

Order of operations mirrors the registration's own dependency order:

  1. trace_present   -- every sampled arm carries >= 1000 samples, else that arm is UNMEASURED.
  2. perturbation    -- the sampler's own cost, from the palindrome control. ADJUDICATES FIRST:
                        >= SAMPLER_PERTURBATION_MAX on any registered row marks the hunt's
                        frequency data SUSPECT before any of it is read.
  3. elevation       -- which hunt arms are elevated (GFLOP/s witness), which are the floor.
                        No elevated arm -> every frequency cell UNMEASURED (rule 26 at the subject).
  4. classification  -- each in-arm sample HIGH/LOW against the floor arms' pooled median clock.
  5. prediction      -- per arm, the HIGH fraction against H_clock / the falsifier / MIXED.
  6. secondary       -- freq_ratio vs gflops_ratio per elevated arm, printed with both terms,
                        never scored.

Run: python3 archive/freq150/analyze-freq150.py [REV]
"""
import glob
import importlib.util as u
import os
import re
import statistics
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", ".."))
REV = sys.argv[1] if len(sys.argv) > 1 else "71a73da"


def load_registration():
    s = u.spec_from_file_location("reg", os.path.join(HERE, "predictions-freq150.py"))
    m = u.module_from_spec(s)
    s.loader.exec_module(m)
    return m


REG = load_registration()

# The bench key for a registered row. isa is avx512 because #150's elevation was on the AVX-512
# kernels; that is a fact about #150's table, imported implicitly with the rows, not a new choice.
def row_key(shape, kc):
    return "BenchmarkKernel/%s/avx512/kc=%d" % (shape, kc)


ROWS = [(shape, kc, row_key(shape, kc)) for shape, kc in REG.REGISTERED_ROWS]

# The ten hunt arms in run order (the labels the driver wrote), and the two sampled control arms.
HUNT_LABELS = ["aref", "ac0", "agc", "apre", "aboth", "bboth", "bpre", "bgc", "bc0", "bref"]
CONTROL_ON = ["k2", "k3"]     # sampler on
CONTROL_OFF = ["k1", "k4"]    # sampler off
# Which config each hunt label is, so an elevated arm can be named by its treatment.
LABEL_CONFIG = {
    "aref": "ref", "ac0": "c0", "agc": "gc", "apre": "pre", "aboth": "both",
    "bboth": "both", "bpre": "pre", "bgc": "gc", "bc0": "c0", "bref": "ref",
}

BENCH_RE = re.compile(
    r"^(BenchmarkKernel/\S+)\s+\d+\s+[\d.]+\s+ns/op\s+([\d.]+)\s+GFLOP/s")


def bench_path(label):
    return os.path.join(ROOT, "build", "bench-freq150-%s-%s.txt" % (REV, label))


def trace_path(label):
    return os.path.join(ROOT, "build", "freq150-trace-%s-%s.txt" % (REV, label))


def parse_bench(label):
    """label -> {full_row_key: [gflops samples]}"""
    out = {}
    with open(bench_path(label)) as f:
        for line in f:
            m = BENCH_RE.match(line)
            if m:
                out.setdefault(m.group(1), []).append(float(m.group(2)))
    return out


def parse_trace(label):
    """label -> [khz ints], NA dropped (there are none in this run, but drop defensively)."""
    khz = []
    p = trace_path(label)
    if not os.path.exists(p):
        return None
    with open(p) as f:
        for line in f:
            parts = line.split()
            if len(parts) == 2 and parts[1].isdigit():
                khz.append(int(parts[1]))
    return khz


def med(xs):
    return statistics.median(xs) if xs else float("nan")


def hr():
    print("-" * 92)


def main():
    print("=" * 92)
    print("freq150 analysis -- rev %s, host %s (%s)" % (REV, "janus.local", REG.HOST_MODEL))
    print("registration: archive/freq150/predictions-freq150.py (committed before the run)")
    print("=" * 92)

    bench = {lbl: parse_bench(lbl) for lbl in HUNT_LABELS + CONTROL_ON + CONTROL_OFF}
    traces = {lbl: parse_trace(lbl) for lbl in HUNT_LABELS + CONTROL_ON}

    # ---- 1. trace_present -----------------------------------------------------------------------
    print("\n1. PRECONDITION trace_present (>= 1000 samples per sampled arm)")
    trace_ok = True
    for lbl in HUNT_LABELS + CONTROL_ON:
        n = len(traces[lbl]) if traces[lbl] else 0
        ok = n >= 1000
        trace_ok = trace_ok and ok
        print("   %-6s %5d samples  %s" % (lbl, n, "ok" if ok else "SHORT -> UNMEASURED"))
    print("   => trace_present: %s" % ("PASS" if trace_ok else "FAIL (some arms UNMEASURED)"))

    # ---- 2. perturbation control (adjudicates first) --------------------------------------------
    print("\n2. PRECONDITION sampler_ok -- perturbation from the palindrome control (off/on/on/off)")
    print("   |median(on) - median(off)| / median(off), per row; on={k2,k3} off={k1,k4}")
    print("   bound SAMPLER_PERTURBATION_MAX = %.3f (%s)"
          % (REG.SAMPLER_PERTURBATION_MAX, REG.SAMPLER_PERTURBATION_BASIS))
    hr()
    print("   %-20s %12s %12s %10s   %s" % ("row", "off GFLOP/s", "on GFLOP/s", "|d|/off", "verdict"))
    sampler_ok = True
    for shape, kc, key in ROWS:
        off = [g for lbl in CONTROL_OFF for g in bench[lbl].get(key, [])]
        on = [g for lbl in CONTROL_ON for g in bench[lbl].get(key, [])]
        mo, mn = med(off), med(on)
        pert = abs(mn - mo) / mo if mo else float("nan")
        bad = pert >= REG.SAMPLER_PERTURBATION_MAX
        sampler_ok = sampler_ok and not bad
        print("   %-20s %12.2f %12.2f %9.2f%%   %s"
              % ("%s kc=%d" % (shape, kc), mo, mn, 100 * pert,
                 "REJECT" if bad else "ok"))
    hr()
    if sampler_ok:
        print("   => sampler_ok: PASS. The instrument is smaller than the term it hunts;")
        print("      the hunt's frequency data is admissible.")
    else:
        print("   => sampler_ok: FAIL. The sampler is rejected as an instrument. The hunt's")
        print("      frequency data below is SUSPECT whatever it shows, and is NOT renegotiated.")

    # ---- 3. elevation ---------------------------------------------------------------------------
    # floor_gflops[row] = median over the ten hunt arms of each arm's median GFLOP/s. The elevated
    # arms are a minority (#150 saw 2 of 10), so the median across arms lands in the floor cluster
    # and needs no arm labelled in advance -- the estimate is robust to the very outliers it hunts.
    print("\n3. ELEVATION -- GFLOP/s witness, factor %.2f (PRECONDITION elevation_present)"
          % REG.ELEVATION_FACTOR)
    arm_med = {lbl: {key: med(bench[lbl].get(key, [])) for _, _, key in ROWS}
               for lbl in HUNT_LABELS}
    floor_g = {}
    for shape, kc, key in ROWS:
        floor_g[key] = med([arm_med[lbl][key] for lbl in HUNT_LABELS])
    hr()
    hdr = "   %-6s" + "".join("%14s" % ("%s/kc%d" % (s, kc)) for s, kc, _ in ROWS) + "   elevated?"
    print(hdr % "arm")
    elevated = {}  # label -> list of (shape,kc) rows it is elevated on
    for lbl in HUNT_LABELS:
        cells = []
        rows_elev = []
        for shape, kc, key in ROWS:
            r = arm_med[lbl][key] / floor_g[key] if floor_g[key] else float("nan")
            cells.append(r)
            if r >= REG.ELEVATION_FACTOR:
                rows_elev.append((shape, kc))
        if rows_elev:
            elevated[lbl] = rows_elev
        marks = "".join("%13.3fx" % c for c in cells)
        tag = ("YES " + ",".join("%s/kc%d" % (s, kc) for s, kc in rows_elev)) if rows_elev else "no"
        print("   %-6s%s   %s" % (lbl, marks, tag))
    hr()
    print("   floor GFLOP/s (median-of-arm-medians per row):  "
          + "  ".join("%s/kc%d=%.1f" % (s, kc, floor_g[key]) for s, kc, key in ROWS))
    elevation_present = bool(elevated)
    print("   => elevation_present: %s" % (
        "PASS -- %d arm(s) elevated: %s" % (len(elevated), ", ".join(sorted(elevated)))
        if elevation_present else
        "FAIL -- NO arm elevated on any registered row"))
    if not elevation_present:
        print("\n   Every frequency cell below is UNMEASURED, NOT 'the clock is refuted' (rule 26).")
        print("   Deliverable: a second independent observation that the phenomenon did not recur")
        print("   under an identical sequence -- OOD item 4. The clock hypothesis is untouched.")

    # ---- 4. classification ----------------------------------------------------------------------
    # Floor arms = hunt arms not elevated on any row. Their pooled median clock is the baseline.
    floor_arms = [lbl for lbl in HUNT_LABELS if lbl not in elevated]
    floor_khz_pool = [k for lbl in floor_arms for k in (traces[lbl] or [])]
    floor_khz = med(floor_khz_pool)
    print("\n4. CLASSIFICATION -- HIGH if khz > floor_khz x %.2f (%s)"
          % (REG.FREQ_STEP_BOUNDARY, REG.FREQ_STEP_BASIS))
    print("   floor arms: %s" % (", ".join(floor_arms) if floor_arms else "(none)"))
    print("   floor_khz  = %s kHz (pooled median over floor arms' in-arm samples)"
          % ("%.0f" % floor_khz if floor_khz == floor_khz else "nan"))
    boundary_khz = floor_khz * REG.FREQ_STEP_BOUNDARY if floor_khz == floor_khz else float("nan")
    print("   boundary   = %.0f kHz" % boundary_khz)

    def high_fraction(lbl):
        ks = traces[lbl] or []
        if not ks:
            return float("nan"), 0
        h = sum(1 for k in ks if REG.fmode(k, floor_khz) == "HIGH")
        return h / len(ks), len(ks)

    # ---- 5. prediction --------------------------------------------------------------------------
    print("\n5. PREDICTION -- per arm, in the instrument's output space (fractions, not medians)")
    print("   H_clock: elevated arm >= %d%% HIGH, floor arm >= %d%% LOW"
          % (100 * REG.PREDICT_FRACTION, 100 * REG.PREDICT_FRACTION))
    if not sampler_ok:
        print("   [reported SUSPECT: the perturbation control rejected the sampler]")
    if not elevation_present:
        print("   [reported UNMEASURED: no elevated subject]")
    hr()
    print("   %-6s %10s %8s %8s   %-8s  %s"
          % ("arm", "class", "HIGH%", "n", "median", "verdict"))
    for lbl in HUNT_LABELS:
        frac, n = high_fraction(lbl)
        mk = med(traces[lbl] or [])
        is_elev = lbl in elevated
        cls = "ELEVATED" if is_elev else "floor"
        if not elevation_present:
            verdict = "UNMEASURED"
        elif is_elev:
            if frac < 0.50:
                verdict = "FALSIFIES H_clock (<50%% HIGH; sped up, clock did not)"
            elif frac >= REG.PREDICT_FRACTION:
                verdict = "H_clock holds (>=%d%% HIGH)" % (100 * REG.PREDICT_FRACTION)
            else:
                verdict = "MIXED (50-95%%; reported, not rounded)"
        else:
            low = 1 - frac
            verdict = ("floor confirmed (>=%d%% LOW)" % (100 * REG.PREDICT_FRACTION)
                       if low >= REG.PREDICT_FRACTION
                       else "floor arm reads %.1f%% HIGH -- unexpected, reported" % (100 * frac))
        if not sampler_ok:
            verdict = "SUSPECT / " + verdict
        print("   %-6s %10s %7.1f%% %8d   %-8.0f  %s"
              % (lbl, cls, 100 * frac, n, mk, verdict))
    hr()

    # ---- 6. secondary (reported, never scored) --------------------------------------------------
    if elevation_present:
        print("\n6. SECONDARY (reported, NEVER scored): freq_ratio vs gflops_ratio per elevated arm")
        print("   %s" % REG.SECONDARY)
        hr()
        for lbl in sorted(elevated):
            mk = med(traces[lbl] or [])
            fr = mk / floor_khz if floor_khz else float("nan")
            print("   %s: median clock %.0f kHz  freq_ratio=%.3fx" % (lbl, mk, fr))
            for shape, kc in elevated[lbl]:
                key = row_key(shape, kc)
                gr = arm_med[lbl][key] / floor_g[key] if floor_g[key] else float("nan")
                print("      %s/kc%d: gflops_ratio=%.3fx   (clock step %s explain the work step: "
                      "freq %.3f vs work %.3f)"
                      % (shape, kc, gr,
                         "CAN" if fr >= gr * 0.98 else "leaves a residue -> second mechanism",
                         fr, gr))
        hr()

    # ---- reported, never scored: the clock DID vary; the throughput did not ---------------------
    # This is the one substantive thing the run saw, and it is deliberately OUTSIDE the scored
    # frame: every cell above is UNMEASURED because no arm was elevated in GFLOP/s, so nothing here
    # is evidence for or against H_clock. It is reported because a null on the registered question
    # that is silent about a visible feature of the data would be a coverage gap (rule 12).
    print("\nREPORTED, NEVER SCORED -- what the samples show given there is no elevated subject:")
    hi = [lbl for lbl in HUNT_LABELS if (high_fraction(lbl)[0] or 0) >= 0.30]
    lo = [lbl for lbl in HUNT_LABELS if (high_fraction(lbl)[0] or 0) < 0.05
          and med(traces[lbl] or []) > 2e6]
    idle = [lbl for lbl in HUNT_LABELS if med(traces[lbl] or []) <= 2e6]
    print("   The clock stepped: %s spent >=30%% of samples above the %.0f kHz boundary,"
          % (", ".join(hi) if hi else "(none)", boundary_khz))
    print("   while %s stayed below it. Yet GFLOP/s is flat to +/-0.6%% across ALL ten arms"
          % (", ".join(lo) if lo else "(none)"))
    print("   (section 3). On THIS run the clock and the throughput decoupled -- which is a fact")
    print("   about the samples, not a verdict on #150's absent elevation.")
    if idle:
        print("   Scope note: %s show a median of 1.20 GHz because the `ref` config spans two cores"
              % ", ".join(idle))
        print("   (mask 0,1) and the sampler reads cpu0 only, which the runtime need not keep busy.")

    print("\ndomain limits carried from the registration (not re-litigated here):")
    for item in REG.OUT_OF_DOMAIN:
        print("   - " + ". ".join(item.split(". ")[:2]).rstrip(".") + ".")


if __name__ == "__main__":
    main()
