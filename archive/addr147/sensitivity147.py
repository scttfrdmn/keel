#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""POST-HOC sensitivity, not a registered criterion, and it is separated from
analyze147.py so the two cannot be confused: #147's classifier takes the LARGEST
adjacent gap in the sorted 30 and only then checks that both clusters have >= 3
members, so one gross outlier owns the gap and a real two-cluster row reads unimodal.
This variant maximizes the gap OVER SPLITS THAT ALREADY SATISFY the >= 3 rule and
reports which cells that adds, with the same R2 address verdict for each.

Usage: sensitivity147.py <bench.log> <trace.txt> <label>
"""
import collections, os, sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from analyze147 import load, join, classify


def robust(vals):
    s = sorted(vals)
    n = len(s)
    med = s[n // 2] if n % 2 else (s[n // 2 - 1] + s[n // 2]) / 2
    best, at = 0.0, None
    for k in range(2, n - 3):          # both clusters >= 3 by construction
        if s[k + 1] - s[k] > best:
            best, at = s[k + 1] - s[k], k
    if at is None:
        return False, 0.0, med, None
    pct = 100.0 * best / med
    return pct > 1.5, pct, med, s[at + 1]


def main():
    benchlog, tracefile, label = sys.argv[1], sys.argv[2], sys.argv[3]
    S, E = load(benchlog, tracefile)
    print(f'== {label}: post-hoc outlier-robust classifier ==')
    added = 0
    for row in sorted(S):
        sm, ex = S[row], E[row]
        tidx, ok, _ = join(sm, ex, 'render')
        if not ok:
            continue                    # R5 still gates: unmeasured stays unmeasured
        if classify([s[1] for s in sm])[0]:
            continue                    # already bimodal under the registered classifier
        rb, pct, _, cut = robust([s[1] for s in sm])
        if not rb:
            continue
        added += 1
        timed = [ex[j] for j in tidx]
        seq = ''.join('H' if s[1] >= cut else 'L' for s in sm)
        tup = [t[2:] for t in timed]
        by = collections.defaultdict(set)
        for t, ch in zip(tup, seq):
            by[t].add(ch)
        pure = all(len(v) == 1 for v in by.values())
        nc = len(set(tup))
        verdict = ('one tuple for all samples -> REFUTED' if nc == 1 else
                   '2 tuples, partition equals the modes -> SUPPORTED' if nc == 2 and pure else
                   f'{nc} classes, each mode-pure -> inconclusive (finer than the modes)' if pure else
                   'a tuple appears in both modes -> REFUTED')
        print(f'  + {row.replace("BenchmarkKernel/", ""):24s} gap={pct:5.2f}% '
              f'H={seq.count("H")} L={seq.count("L")} tuples={nc} mode-pure={pure}')
        print(f'      seq {seq}')
        print(f'      {verdict}')
        for nm, u, w in (('a-b', 0, 1), ('a-c', 0, 2), ('b-c', 1, 2)):
            al = [(int(t[u], 16) - int(t[w], 16)) % 4096 for t in tup]
            if len(set(al)) > 1:
                ba = collections.defaultdict(set)
                for x, ch in zip(al, seq):
                    ba[x].add(ch)
                print(f'      alias {nm} mod 4096: {sorted(set(al))} '
                      f'each value mode-pure={all(len(z) == 1 for z in ba.values())}')
    print(f'  cells added by the variant: {added}')


if __name__ == '__main__':
    main()
