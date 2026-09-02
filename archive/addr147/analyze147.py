#!/usr/bin/env python3
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
"""#147 arm analysis. Implements the pre-registered R1-R5 and nothing else.

Usage: analyze147.py <bench.log> <trace.txt> <label>

R5 as registered reads: each of the 30 reported samples must match EXACTLY ONE trace
execution by `n`, with `elapsed/n` agreeing to the printed 4 significant figures. That
sentence has two possible implementations and they do not agree, so both are computed
and both are reported per row:

  sig4    round elapsed/n and the printed value each to 4 significant figures and
          compare. This double-rounds -- testing has ALREADY rounded once to produce the
          log -- and the second rounding is both too loose and too tight. Too loose:
          testing prints anything >= 999.95 as an integer (%10.0f, read from
          $GOROOT/src/testing/benchmark.go), so on a row where all 30 samples share one
          b.N and sit inside one 4-sig-fig bucket, several executions match one sample
          and the join is ambiguous. Too tight: a value like 19845.4 renders as "19845",
          whose own 4-sig-fig rounding (ties-to-even, 19840) differs from that of the
          true value (19850), so a correct pair misses.
  render  reproduce testing.prettyPrint's rendering of elapsed/n and compare it to the
          token the log actually contains. This is the number in the log, it is strictly
          MORE discriminating per sample than sig4, and it has no second rounding.

`render` is used for R1/R2/R3. That choice was made after seeing sig4 fail, and it is a
property of testing's formatter rather than of the data -- so the sig4 column stays in
the output for every row, and any row where the two disagree is named in the report.
"""
import re, sys, collections

BENCH = re.compile(r'^(BenchmarkKernel\S*?)(?:-\d+)?\s+(\d+)\s+([\d.]+) ns/op')
TRACE = re.compile(r'^keel-addrtrace name=(\S+) n=(\d+) elapsed_ns=(\d+) a=(\S+) b=(\S+) c=(\S+)')


def sig4(x):
    if x == 0:
        return 0.0
    from math import floor, log10
    d = 4 - 1 - int(floor(log10(abs(x))))
    return round(x, d)


def pretty(x):
    """testing.prettyPrint, transcribed from $GOROOT/src/testing/benchmark.go (read, not
    recalled). The first case is the one that matters here: at or above 999.95 the value
    prints as an integer, so a large ns/op carries more than four significant figures."""
    y = abs(x)
    if y == 0 or y >= 999.95:
        return f'{x:.0f}'
    if y >= 99.995:
        return f'{x:.1f}'
    if y >= 9.9995:
        return f'{x:.2f}'
    if y >= 0.99995:
        return f'{x:.3f}'
    if y >= 0.099995:
        return f'{x:.4f}'
    if y >= 0.0099995:
        return f'{x:.5f}'
    if y >= 0.00099995:
        return f'{x:.6f}'
    return f'{x:.7f}'


def load(benchlog, tracefile):
    samples = collections.defaultdict(list)   # row -> [(n, nsop, printed token)] in order
    for line in open(benchlog, errors='replace'):
        m = BENCH.match(line.strip())
        if m:
            samples[m.group(1)].append((int(m.group(2)), float(m.group(3)), m.group(3)))
    execs = collections.defaultdict(list)     # row -> [(n, elapsed, a, b, c)] in order
    for line in open(tracefile, errors='replace'):
        m = TRACE.match(line.strip())
        if m:
            execs[m.group(1)].append((int(m.group(2)), int(m.group(3)),
                                      m.group(4), m.group(5), m.group(6)))
    return samples, execs


def join(samples, execs, mode):
    """Returns (indices into execs, ok, ambiguities). Indices, not tuples: two executions
    can carry identical (n, elapsed, a, b, c) and identity-by-value would confuse them.
    A sample with two candidates is AMBIGUOUS and the row fails -- breaking the tie by
    position would be inventing a rule the pre-registration does not contain."""
    idx, amb = [], []
    for i, (n, nsop, token) in enumerate(samples):
        if mode == 'render':
            cands = [j for j, e in enumerate(execs)
                     if e[0] == n and pretty(e[1] / e[0]) == token]
        else:
            cands = [j for j, e in enumerate(execs)
                     if e[0] == n and abs(sig4(e[1] / e[0]) - sig4(nsop)) < 1e-6 * max(1.0, abs(nsop))]
        if len(cands) == 1:
            idx.append(cands[0])
        else:
            amb.append((i, cands))
    return idx, len(idx) == len(samples), amb


def classify(vals):
    """#147's classifier, unchanged: largest adjacent gap in the sorted 30 > 1.5% of the
    median AND both clusters >= 3 members."""
    s = sorted(vals)
    n = len(s)
    med = s[n // 2] if n % 2 else (s[n // 2 - 1] + s[n // 2]) / 2
    best, at = 0.0, None
    for k in range(n - 1):
        if s[k + 1] - s[k] > best:
            best, at = s[k + 1] - s[k], k
    if at is None:
        return False, 0.0, med, None
    pct = 100.0 * best / med
    return (pct > 1.5 and at + 1 >= 3 and n - at - 1 >= 3), pct, med, s[at + 1]


def alias(xs, ys):
    return sorted({(int(x, 16) - int(y, 16)) % 4096 for x, y in zip(xs, ys)})


def main():
    benchlog, tracefile, label = sys.argv[1], sys.argv[2], sys.argv[3]
    samples, execs = load(benchlog, tracefile)
    print(f'== {label} ==  rows={len(samples)}  trace rows={len(execs)}')
    bimodal_rows, unmeasured, disagree = [], [], []
    for row in sorted(samples):
        sm, ex = samples[row], execs.get(row, [])
        short = row.replace('BenchmarkKernel/', '')
        tidx, ok, amb = join(sm, ex, 'render')
        sidx, sok, samb = join(sm, ex, 'sig4')
        if ok != sok:
            disagree.append((short, len(tidx), len(sidx)))
        if not ok:
            zero = sum(1 for _, c in amb if not c)
            multi = [(i, c) for i, c in amb if len(c) > 1]
            unmeasured.append(short)
            print(f'{short:26s} UNMEASURED (R5): render join {len(tidx)}/{len(sm)} of '
                  f'{len(ex)} execs; {zero} with no candidate, {len(multi)} ambiguous '
                  f'| sig4 join {len(sidx)}/{len(sm)}')
            for i, c in multi[:3]:
                print(f'    sample {i} n={sm[i][0]} printed={sm[i][2]}: {len(c)} candidates, '
                      f'{len({ex[j][2:] for j in c})} distinct address tuple(s)')
            continue
        timed = [ex[j] for j in tidx]
        a = [t[2] for t in timed]; b = [t[3] for t in timed]; c = [t[4] for t in timed]
        da, db, dc = len(set(a)), len(set(b)), len(set(c))
        alla, allb, allc = (len({t[2] for t in ex}), len({t[3] for t in ex}),
                            len({t[4] for t in ex}))
        bim, pct, med, cut = classify([s[1] for s in sm])
        print(f'{short:26s} join=30/30 sig4={len(sidx)}/{len(sm)} med={med:9.4g} '
              f'gap={pct:6.2f}% {"BIMODAL" if bim else ".":8s} '
              f'distinct timed a/b/c={da}/{db}/{dc} all-exec={alla}/{allb}/{allc} '
              f'alias4k ab={alias(a, b)} ac={alias(a, c)} bc={alias(b, c)}')
        if bim:
            bimodal_rows.append((row, short, sm, timed, tidx, cut, pct))

    print(f'\n-- R1:{len(samples) - len(unmeasured)} of {len(samples)} rows analyzable; '
          f'{len(unmeasured)} unmeasured --')
    const = 0
    for row in sorted(samples):
        short = row.replace('BenchmarkKernel/', '')
        if short in unmeasured:
            continue
        tidx, ok, _ = join(samples[row], execs[row], 'render')
        timed = [execs[row][j] for j in tidx]
        if len({t[2:] for t in timed}) == 1:
            const += 1
    print(f'R1: rows with all three addresses constant across the 30 timed samples: '
          f'{const} of {len(samples) - len(unmeasured)} analyzable '
          f'(registered prediction: >= 8 of 20)')

    print(f'\n-- R3: {len(bimodal_rows)} bimodal cells of '
          f'{len(samples) - len(unmeasured)} analyzable rows --')
    for row, short, sm, timed, tidx, cut, pct in bimodal_rows:
        seq = ''.join('H' if s[1] >= cut else 'L' for s in sm)
        tup = [t[2:] for t in timed]
        by_tuple = collections.defaultdict(set)
        for t, ch in zip(tup, seq):
            by_tuple[t].add(ch)
        pure = all(len(v) == 1 for v in by_tuple.values())
        nc = len(set(tup))
        if nc == 1:
            verdict = 'HYPOTHESIS 1 REFUTED on this cell: one address tuple for all samples'
        elif nc == 2 and pure:
            verdict = 'HYPOTHESIS 1 SUPPORTED on this cell: 2 tuples, partition equals the modes'
        elif pure:
            verdict = (f'INCONCLUSIVE: {nc} tuple classes, each pure in mode '
                       f'(strictly finer than the modes)')
        else:
            verdict = 'HYPOTHESIS 1 REFUTED on this cell: a tuple appears in both modes'
        per2 = all(seq[i] != seq[i + 1] for i in range(len(seq) - 1))
        print(f'\n{short}  gap={pct:.2f}%  H={seq.count("H")} L={seq.count("L")}  period2={per2}')
        print(f'  seq  {seq}')
        print(f'  address tuples: {nc} distinct over 30 timed samples; each mode-pure: {pure}')
        print(f'  -> {verdict}')
        # Registered as reported data, not as a verdict: the 4 KB-alias form.
        for nm, u, v in (('a-b', 0, 1), ('a-c', 0, 2), ('b-c', 1, 2)):
            al = [(int(t[u], 16) - int(t[v], 16)) % 4096 for t in tup]
            byal = collections.defaultdict(set)
            for x, ch in zip(al, seq):
                byal[x].add(ch)
            if len(set(al)) > 1:
                print(f'  alias {nm} mod 4096: {sorted(set(al))}, '
                      f'each value mode-pure: {all(len(v2) == 1 for v2 in byal.values())} '
                      f'(reported, not a registered verdict)')
        if per2:
            ex = execs[row]
            diffs = sum(1 for j in tidx if j > 0 and ex[j - 1][2:] != ex[j][2:])
            print(f'  ramp-vs-timed: {diffs}/{len(timed)} timed runs differ in address from '
                  f'the execution immediately before them')

    if disagree:
        print('\n-- rows where the two R5 implementations disagree on measurability --')
        for short, r, s in disagree:
            print(f'   {short}: render {r}/30, sig4 {s}/30')


if __name__ == '__main__':
    main()
