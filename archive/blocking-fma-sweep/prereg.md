<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Blocking on the FMA-class hosts, stock Go: predictions (#26, #142)

**Why.** On Skylake-SP a KC × MC sweep at the published size found +8.9–13.5% sitting unused
(`archive/blocking-skx-ab*/`, shipped as issue-class KCDeep). Zen 5, Zen 4, Sapphire Rapids and Granite
Rapids never got that sweep. They all dispatch 4x32 and run the shipped KC=384 / MC=144. This run asks
whether any of them has a similar win.

**Run.** Four on-demand hosts, one per CPU: c8a.2xlarge (Zen 5), c7a.2xlarge (Zen 4), c7i.4xlarge
(SPR) and c8i.4xlarge (GNR). Stock go1.27.3 on main, with no code change. `BenchmarkBlocking` at 2048³,
for 4x32 (dispatched) and 2x32 (reference):
- **grid:** KC ∈ {256, 384, 512, 768, 1024} × MC ∈ {72, 144, 288} × NC = 2048;
- **samples:** 4 `-randlayout` seeds × `-test.count=3`, so 12 samples per point, pooled by
  `tools/benchci`;
- **placement:** 1T on core 2; 8T on one thread per physical core, from `lscpu`.

Two placement facts are disclosed:
- The two AMD sizes have exactly 8 cores, so their 8T run shares cores with sshd.
- MC=288 at n=2048 leaves 8 row blocks for 8 workers, so at 8T it is partition-limited. This is the
  `blocking-skx-ab` lesson, so MC is judged at 1T only.

**Predictions** (4x32, net of both intervals):
1. **Null at 8T on every host.** No point beats the shipped kc=384/mc=144 by more than 2%.
2. **Null at 1T on every host.** The best point is within 3% of shipped. Skylake-SP's lever was an
   issue-bound 2x32 amortising its C traffic over a deeper K. 4x32 on two FMA ports is not
   issue-bound in the same way.
3. **KC=256 loses** by at least 2% at 1T on every host, because its per-tile C load/store overhead is
   1.5× shipped's.

**REFUTED for shipping on a host if** no point beats shipped at 8T net of both intervals. A host where
some point does beat shipped would go to stage B (the published rows at n=4096, as in
`blocking-skx-ab-b/`) before anything ships. Prediction 1 failing is that case.

## Result (`3e8c29f`, `run-3e8c29f.log`; `benchci-<host>-{1,8}t.csv` are `tools/benchci` over each point's 12 samples)

There were four hosts:
- c8a.2xlarge, AMD EPYC 9R45 (Zen 5);
- c7a.2xlarge, AMD EPYC 9R14 (Zen 4);
- c7i.4xlarge, Xeon Platinum 8488C (SPR);
- c8i.4xlarge, Xeon 6975P-C (GNR).

The run took 24m46s and produced 2880 rows. The fleet was verified down by a readable inventory.

**4x32 (dispatched): deeper KC at the shipped MC=144, against shipped kc=384/mc=144.** Each cell is
the change in the median, with its range net of both CIs.

| host | 1T kc=512 | 1T kc=1024 | 8T kc=512 | 8T kc=1024 |
|---|---|---|---|---|
| Zen 5 | +0.7% [+0.5, +0.8] | +1.3% [+1.1, +1.6] | +1.7% [+1.1, +2.4] | +0.1% [−0.3, +0.9] |
| Zen 4 | +1.1% [+0.8, +1.2] | +0.9% [+0.7, +1.5] | +1.7% [+1.0, +2.7] | +1.1% [+0.7, +1.8] |
| **SPR** | +6.0% [+5.3, +6.7] | **+11.2%** [+10.2, +11.6] | +6.1% [+5.1, +6.8] | **+11.9%** [+11.1, +12.7] |
| GNR | +1.9% [−7.3, +11.3] | +3.3% [−7.1, +7.4] | +0.8% [−0.9, +2.3] | +2.3% [−1.2, +3.5] |

1. **Split.**
   - ✓ Zen 5 and Zen 4: no point beats shipped by more than 2% at 8T.
   - ✗ SPR: the best 8T point is kc=1024/mc=72, at **+12.7% (+12.0% net)**. The surface still rises
     at the grid's edge, KC = 1024.
   - ✗ GNR: the best 8T point is kc=256/mc=144, at +6.1% (+4.3% net). This is not credible as a
     lever. The GNR surface is non-monotone, kc=256 loses elsewhere, and GNR's 1T intervals are
     ±8–10%. That host was noisy, so its reading is **inconclusive**, not a finding.
2. **Split.**
   - ✓ Zen 5 and Zen 4: the best 1T point is within 1.3%.
   - ✗ SPR: +11.7% at 1T (kc=1024/mc=288).

   My mechanism was wrong. SPR runs 4x32 at only about 35% of its vector peak, which is
   Skylake-SP's regime, not Zen's. The "FMA class" label groups SPR with Zen on kernel *shape*. That
   says nothing about how SPR responds to blocking.
3. **Split.**
   - ✓ SPR: −4.1%.
   - ✗ Zen 5: −0.8%.
   - ✗ Zen 4: −0.7%.
   - ✗ GNR: −1.1%, within noise.

**Not refuted for shipping on SPR.** SPR goes to stage B: the published rows at n=4096 with
KCDeep = 1024, as in `blocking-skx-ab-b/`. SPR is not a README host, so no published row moves.
Zen 5 and Zen 4 are refuted for shipping. GNR needs a quieter re-measure before it reads either way.
