# Pre-registration: the judged amd64 run that refreshes Strsm (#37, 2026-10-11)

Written before launch. Rev d62848a..628c38a (HEAD), era ceil8, fleet read off the
archives: c5n.18xlarge (keel-skx), c7a.48xlarge (keel-zen4), c8a.48xlarge
(keel-zen5), on-demand, us-east-1, $24.09/hr for the three.

## THIS RUN IS EXPECTED RED, AND THAT IS NOT A FAILURE
Its product is the archive, not a verdict. Criterion 9 compares the published
README against this run, and the README was regenerated at era ceil8 from
archives that PREDATE 090eb32. Strsm is now ~39% faster on this silicon, which is
8x README_TOL and disjoint from any run interval, so:

- **criterion 9 FAILS on the Strsm rows of all three amd64 hosts.** PREDICTED, and
  the reason the run exists. Era ceil8's own confirm run was RED for exactly this
  reason and its DIGESTS header says so.
- Every other criterion should PASS or BASELINE. Specifically:
  - **share/* : PASS.** The 12 amd64 share baselines landed at d59f8b2 for era
    ceil8, so these hosts are `registered` and judged against the registry. The
    registered value less margin is a FLOOR, and Strsm's share RISES, so a 39%
    improvement passes rather than trips it.
  - **the L1 criterion: BASELINE x3.** Those rows were deliberately held back and
    no witness row was landed, so the hosts resolve `new`, which is
    green-compatible. If any L1 row resolves `owing` instead, the witness rows
    leaked in and that is a finding about d59f8b2, not about Strsm.
  - **gate-p3/p4 chain: GREEN**, as at 9a5e2f5.

## Row-granular predictions for the quantity under test
Strsm GFLOP/s, threads=1, against the currently published figure:

| host | published (stale) | predicted |
|---|---|---|
| Intel Xeon 8124M | 23.78 | 33-40 |
| AMD EPYC 9R14 | 38.09 | 53-64 |
| AMD EPYC 9R45 | 58.16 | 81-98 |

The band is the janus result (-39.26% sec/op, +64.6% GFLOP/s) applied to each
published figure, widened to +40%..+70% because janus is bare metal and these are
virtualized. **REFUTED if any host comes in under +15%**, which is the same floor
the dev-tier run was held to. The three controls (Sgemm, Ssyrk, Ssymm) must land
inside their published CIs; one outside says something other than Strsm moved.

## What this run does NOT finish
N=2 is the published standard and it means TWO runs per architecture, because
pooling two runs is what sees between-run drift and a single run's internal CI
cannot (#141). So this run alone leaves amd64 at N=1 and the arm64 rows untouched
and still stale. Regenerating the README whole requires the complement; that is a
separate decision and a separate spend, recorded here so the first run is not
mistaken for the whole job.

## Safety, all four required and all four present
`--yes` on every step, a `timeout` on every step, an EXIT/INT/TERM teardown trap,
and KEEL_FLEET_TTL as the backstop for a hang the trap cannot catch. Teardown is
verified against the LAUNCHER's own list (`spawn list --state running | grep keel-`),
never against the script's own bookkeeping. Launched via scripts/detach.sh so the
driver outlives this session.
