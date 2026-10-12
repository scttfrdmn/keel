# Strsm refresh, judged amd64 run 2 of 2 (#37)

Rev `852ad55`, era `ceil8`, us-east-1 on-demand, 2026-10-11 22:25Z–00:06Z (1h41m,
$24.09/hr, ~$40.5). Pairs with `archive/strsm-refresh-amd64-r1/` to give N=2.

**69 PASS / 3 FAIL / 0 UNMEASURED / 3 BASELINE → RED — exactly the pre-registration.**
Run 1 had a fourth failure (an `errcheck` finding of mine); fixing it leaves precisely
the three criterion-9 Strsm failures and three BASELINE that were predicted.

## Why the pair exists, measured

N=2 is the standard the published CIs rest on, and it means two *runs* because
pooling two runs is what sees between-run drift — a quantity one run's internal CI
provably cannot reach (#141). This pair measures it:

| host | thr | run 1 | run 2 | drift |
|---|---|---|---|---|
| keel-skx | 1 | 37.64 | 37.98 | +0.90% |
| keel-skx | 8 | 256.20 | 253.45 | −1.07% |
| keel-zen4 | 1 | 73.98 | 73.72 | −0.35% |
| keel-zen4 | 8 | 464.00 | 471.05 | +1.52% |
| keel-zen5 | 1 | 125.30 | 120.80 | −3.59% |
| keel-zen5 | 8 | 778.90 | 839.85 | **+7.83%** |

Max |drift| **7.83%**, mean 2.54%. The zen5 pair moves in *opposite* directions on
its two thread arms, which no single-run interval would have disclosed. This is the
empirical case for the N=2 rule rather than the argument for it.

## Rev difference inside the pair
Run 1 is `628c38a`, run 2 is `852ad55`, which adds only an `errcheck` fix in
`tools/benchci` — a package `./bench` does not import. The measured binary's
dependency-closure hash is identical at both revs (`0da19ff8…`), so the pair pools.
Era ceil8's own arm64 pair (`119b673` + `5e769f7`) is the precedent for two revs in
one pair.

`driver-frozen.sh` is the exact driver this run executed, archived because it lives
under gitignored `build/` and is therefore not in the commit record.
