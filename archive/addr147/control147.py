#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""Negative control for analyze147.py: drive every registered R2 branch, plus the R5
refusal, with synthetic input whose verdict is known by construction. A verdict table
that has never been driven is an unread witness."""
import os, subprocess, sys
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from analyze147 import pretty     # the same renderer the analyzer joins with

N = 30
A = '0x1000'


def case(name, vals, tuples, drop_join=False, ramp_shift=False, ramp_dup=False):
    """vals: 30 ns/op values. tuples: 30 (a,b,c). One row."""
    row = f'BenchmarkKernel/{name}'
    bench, trace = [], []
    for i, (v, t) in enumerate(zip(vals, tuples)):
        n = 1000 + i               # distinct n per sample: no ambiguity to hide behind
        elapsed = int(round(v * n))
        # Rendered the way testing renders it, so the controls drive the real join path.
        bench.append(f'{row}-12\t{n:12d}\t{pretty(elapsed / n):>12s} ns/op\t 1.000 GFLOP/s\t 1024 flops/call')
        # a ramp step before each timed run, so ramp-vs-timed has something to read
        rt = (t[0], t[1], hex(int(t[2], 16) + 0x8000)) if ramp_shift else t
        rn, re_ = (n, elapsed) if ramp_dup else (n // 2, elapsed // 2)
        trace.append(f'keel-addrtrace name={row} n={rn} elapsed_ns={re_} '
                     f'a={rt[0]} b={rt[1]} c={rt[2]}')
        if not drop_join or i < N - 1:
            trace.append(f'keel-addrtrace name={row} n={n} elapsed_ns={elapsed} '
                         f'a={t[0]} b={t[1]} c={t[2]}')
    return bench, trace


def run(label, bench, trace):
    open('/tmp/c.log', 'w').write('\n'.join(bench) + '\nPASS\n')
    open('/tmp/c.trace', 'w').write('\n'.join(trace) + '\n')
    out = subprocess.run([sys.executable, os.path.join(HERE, 'analyze147.py'), '/tmp/c.log',
                          '/tmp/c.trace', label], capture_output=True, text=True)
    print(out.stdout.strip())
    if out.returncode != 0:
        print('ANALYZER CRASHED:\n' + out.stderr)
    print('-' * 72)


slow = [103.0 + 0.01 * i for i in range(15)]
fast = [100.0 + 0.01 * i for i in range(15)]
alt = [fast[i // 2] if i % 2 == 0 else slow[i // 2] for i in range(N)]   # period-2 L/H
blk = fast + slow                                                        # persistent mode

# 1. bimodal, addresses constant -> REFUTED
run('ctl-constant', *case('ctl/constant', blk, [(A, '0x2000', '0x3000')] * N))
# 2. bimodal, two tuples, partition == modes -> SUPPORTED
tw = [(A, '0x2000', '0x3000')] * 15 + [(A, '0x2000', '0x4000')] * 15
run('ctl-partition-equal', *case('ctl/equal', blk, tw))
# 3. bimodal, a tuple appears in both modes -> REFUTED
mixed = [(A, '0x2000', '0x3000' if i % 3 else '0x4000') for i in range(N)]
run('ctl-partition-differs', *case('ctl/differs', blk, mixed))
# 4. bimodal, tuple classes strictly finer than the modes -> INCONCLUSIVE
finer = [(A, '0x2000', hex(0x3000 + 0x1000 * (i // 5))) for i in range(N)]
run('ctl-finer', *case('ctl/finer', blk, finer))
# 5. period-2, addresses alternate with it -> SUPPORTED + period2=True + ramp diff
p2 = [(A, '0x2000', '0x3000' if i % 2 == 0 else '0x4000') for i in range(N)]
run('ctl-period2', *case('ctl/period2', alt, p2))
# 6. unimodal (gap under 1.5%) -> no R2 at all
run('ctl-unimodal', *case('ctl/unimodal', [100.0 + 0.01 * i for i in range(N)],
                          [(A, '0x2000', '0x3000')] * N))
# 7. a missing timed execution -> R5 refusal
run('ctl-unmeasured', *case('ctl/unmeasured', blk, [(A, '0x2000', '0x3000')] * N,
                            drop_join=True))
# 8. period-2 whose ramp step sits at a DIFFERENT address -> ramp-vs-timed must be 30/30,
#    not 0/30: a counter that has only ever printed 0 has not been read.
run('ctl-ramp-differs', *case('ctl/rampdiff', alt, p2, ramp_shift=True))
# 9. two executions with the same n AND the same printed ns/op -> ambiguous, R5 refuses
#    rather than picking one. This is the branch that would silently mis-attribute an
#    address if the join were positional.
run('ctl-ambiguous', *case('ctl/ambig', blk, [(A, '0x2000', '0x3000')] * N, ramp_dup=True))
