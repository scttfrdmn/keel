# Strsm refresh, judged amd64 run 1 of 2 (#37)

Rev `628c38a`, era `ceil8`, us-east-1 on-demand, 2026-10-11 20:40Z–22:21Z (1h41m,
$24.09/hr, ~$40.5). Fleet read off era ceil8's own archives: c5n.18xlarge
(keel-skx), c7a.48xlarge (keel-zen4), c8a.48xlarge (keel-zen5).

**68 PASS / 4 FAIL / 0 UNMEASURED / 3 BASELINE → RED.** `prereg.md`, written before
launch, predicted RED with three criterion-9 failures and three BASELINE. Both the
3 BASELINE and the 3 criterion-9 failures came out exactly as stated. The **fourth
failure was not predicted** and is mine: an `errcheck` finding on a `defer f.Close()`
I added to `tools/benchci/archive_test.go` the same day. I had run `go vet` and
`gofmt` locally but not `golangci-lint`, which the gate does.

## The measurement

Criterion 9 named ONLY the Strsm rows on each host. Of the 9 published rows per CPU
it checks, the other 7 still matched — which is the control that makes the Strsm
delta attributable rather than merely coincident with a fresh run.

| host | thr | published (stale) | this run | delta | pre-registered band |
|---|---|---|---|---|---|
| keel-zen5 | 1 | 58.16 | **125.30** | +115.4% | 81–98 — **OVERSHOT** |
| keel-zen5 | 8 | 388.20 | 778.90 | +100.6% | not predicted |
| keel-zen4 | 1 | 38.09 | **73.98** | +94.2% | 53–64 — **OVERSHOT** |
| keel-zen4 | 8 | 260.60 | 464.00 | +78.1% | not predicted |
| keel-skx | 1 | 23.78 | **37.64** | +58.3% | 33–40 — inside |
| keel-skx | 8 | 164.60 | 256.20 | +55.7% | not predicted |

## The pre-registration MISSED on two of three hosts

Stated as a miss, not as a success with a large number attached. The band came from
the janus dev-tier A/B (+64.6% GFLOP/s on an i9-9960X Skylake-X), widened to
+40%..+70%. It transferred well to the host janus matches microarchitecturally —
**keel-skx landed inside at +58.3%** — and under-predicted both Zen hosts by a
factor of ~1.6–2. So the diagonal solve was a larger share of Strsm's time on
Genoa and Turin than on Skylake, which the model had no term for.

**What this run cannot settle:** it measures only the new code, so each delta above
is a fresh reading against a stale published figure and mixes the code change with
any drift since era ceil8. The attribution rests on two other things — janus's true
A/B (both arms, one host, one session) and the 7-of-9 control rows here. A per-host
A/B on Zen would be needed to explain the overshoot itself.

**Not publishable alone.** N=2 is the standard the published CIs rest on, and it
means two runs, because pooling two runs is what sees between-run drift and one
run's internal CI cannot (#141). Run 2 pairs with this one.
