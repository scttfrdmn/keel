# Judged arm64 run, GREEN — us-east-1, rev 25007d8, 2026-10-04

`gate-p5 tally: 53 PASS / 0 FAIL / 0 UNMEASURED / 2 BASELINE / 0 REPORTED` → **GREEN**,
with `gate-p4: GREEN` (64/0/0) and gate-p3 green beneath it. Driver chained
`provision-openblas.sh --yes && gate-p5.sh` in one detached run, so the tree stayed
frozen for the whole campaign.

## What it settles

**The `-race` criterion is MEASURED on the vector path for the first time on arm64**, which
is #70's last remaining fleet row and #42's outstanding confirmation:

```
PASS [keel-gvt4, vector path live, go version go1.27.0 linux/arm64] race detector clean
PASS [keel-gvt3, vector path live, go version go1.27.0 linux/arm64] race detector clean
```

It needed a host-local toolchain, because `-race` cannot be cross-compiled — so it was
`UNMEASURED` on every prior arm64 run and became measurable only once provisioning
installed Go.

**Both hosts have the same-host OpenBLAS reference** §4/P3 requires, with the thread pin
read back from the library rather than assumed: `corename=neoversev1 threads=1 procs=192`
and `...procs=64`.

## What it does NOT settle

**The 48 L1 baseline candidates are draw 1 of N and are NOT landable.** Their estimator
column says "net-of-CI lower bound of **this run**", which is rule 16's disclosure, and
rule 17(b) makes a single-draw candidate unlandable as it stands. They are archived here so
a later median has a first member. `baseline-candidates-25007d8-*.tsv`.

**`3x24` is NOT promoted and this run does not promote it.** Dispatch was `4x16/neon`
throughout, so this GREEN certifies the *current* shape. #136's evidence (3x24 ahead by
+10.2% / +8.9%, CI-disjoint) came from the earlier us-west-2 run and is confirmed by the
kernel rows here; the flip itself still needs a campaign of its own, because README's rows
can only be regenerated from a run of the flipped code.
