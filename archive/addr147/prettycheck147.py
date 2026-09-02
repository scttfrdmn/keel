#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Checks analyze147.py's pretty() against the toolchain instead of against a second
reading of $GOROOT/src/testing/benchmark.go. testing.prettyPrint is unexported, but
BenchmarkResult.String() calls it, so prettycheck/main.go makes Go render values that
straddle every boundary in the switch and this compares them. A positive control -- the
4-significant-figures-everywhere renderer the R5 wording literally suggests -- must
MISMATCH, or the comparison is incapable of failing and proves nothing.

Usage: prettycheck147.py     (runs `go run ./prettycheck` relative to this file)
"""
import os, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from analyze147 import pretty


def control(x):
    """The literal reading of R5: four significant figures, always."""
    return f'{x:.4g}'


def main():
    out = subprocess.run(['go', 'run', '.'], cwd=os.path.join(HERE, 'prettycheck'),
                         capture_output=True, text=True)
    if out.returncode != 0:
        print('go run failed:\n' + out.stderr)
        return 1
    ok = bad = cbad = skipped = 0
    for line in out.stdout.splitlines():
        head, _, rest = line.partition('\t')
        n, elapsed = int(head.split()[0]), int(head.split()[1])
        if ' ns/op' not in rest:
            # T=0: String() omits the ns/op field entirely, so prettyPrint's y==0 branch
            # is not observable through this witness. Stated, not silently dropped.
            print(f'  SKIP n={n} elapsed={elapsed}: String() emitted no ns/op field')
            skipped += 1
            continue
        want = rest.split(' ns/op')[0].strip()
        got, ctl = pretty(elapsed / n), control(elapsed / n)
        mark = 'ok  ' if got == want else 'MISMATCH'
        if got == want:
            ok += 1
        else:
            bad += 1
        if ctl != want:
            cbad += 1
        print(f'  {mark} n={n:>8d} elapsed={elapsed:>10d} go={want:>10s} '
              f'pretty={got:>10s} control(%.4g)={ctl:>10s}')
    print(f'pretty() vs the toolchain: {ok} match, {bad} mismatch '
          f'({skipped} not observable through String())')
    print(f'positive control (4 sig figs everywhere): {cbad} of {ok + bad} mismatch '
          f'-- a nonzero count is what makes the {ok}/{ok + bad} above meaningful')
    return 1 if bad or cbad == 0 else 0


if __name__ == '__main__':
    sys.exit(main())
