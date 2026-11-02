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
