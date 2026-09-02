#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""The cross-arm LEVEL comparison, which the pre-registration did not ask for and which
is reported as an observation: arm 2 was registered as hypothesis 2's test, not as a
second point in a width sweep. Prints each row's median under both masks and the ratio.

Usage: crossarm147.py <w8.log> <w1.log>
"""
import os, statistics, sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from analyze147 import load


def main():
    S8, _ = load(sys.argv[1], '/dev/null')
    S1, _ = load(sys.argv[2], '/dev/null')
    print(f'{"row":26s} {"w8 med":>10s} {"w1 med":>10s} {"w1/w8":>6s}   '
          f'{"w8 min..max":>21s}   w1 min..max')
    for row in sorted(S8):
        v8 = sorted(s[1] for s in S8[row])
        v1 = sorted(s[1] for s in S1[row])
        m8, m1 = statistics.median(v8), statistics.median(v1)
        print(f'{row.replace("BenchmarkKernel/", ""):26s} {m8:10.4g} {m1:10.4g} '
              f'{m1 / m8:6.2f}   {v8[0]:10.4g}..{v8[-1]:<9.4g} {v1[0]:10.4g}..{v1[-1]:.4g}')


if __name__ == '__main__':
    main()
