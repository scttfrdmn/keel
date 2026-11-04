<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Graviton 4 vs 5: arm64 blocking at the published size, and is V3 load-bound? Predictions

**Why.** Scott, 2026-11-03: "measure arm64 blocking at the published size and test whether
Graviton 5 is load-bound."

V3 tracks V2 at the same kernel efficiency: 81.3% of the family-max peak on both
(`archive/p2-arm64-pb/`), and +22% absolute. Two signals still point at V3 specifically.
1. **Loads cost V3 more.** In the 3x24 limiter probe at kc=512 (`archive/neon-raw/`), removing B
   loads from the pre-broadcast form lifted V3 from 69% to 100% of the FMA-only rate, against 75%
   to 94% on V2.
2. **Deep kc hurts V3 at kernel level.** At kc=512 the shipped 5x16 reads 84.3% on V2 and 80.7% on
   V3 (`archive/neon-pb/summary.txt`).

Separately, arm64's `KCPB`/`MCPB` = 384/72 were chosen single-thread at n=2048. The README rows are
n=4096 at 1T and 8T, and the x86 work showed n=2048 sweeps can mislead there
(`archive/spr-kc-probe/`, `archive/kcdeep-placement-ab/`).

**Run** (`driver.sh`). One c8g.48xlarge (V2) and one c9g.48xlarge (V3), stock go1.27.3, masked to
one core each of socket 0's first 8 physical cores.
- **A, blocking.** Nine builds of this tree, `KCPB` ∈ {256, 384, 512} × `MCPB` ∈ {36, 72, 144}.
  - Rows: BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at n=4096, 1T and 8T.
  - Samples: 4 seeds × 3, with the arm order rotated.
- **B, loads.** `BenchmarkLoadProbe` runs the shipped `KernelPB5x16` four ways:
  - Ctl, verbatim;
  - NA, its 5 A vectors hoisted out of the K-loop;
  - NB, its 4 B vectors hoisted;
  - FO, both hoisted.

  Each runs at kc 128, 384 and 512, beside `BenchmarkPeak/neon` (the family-max ceiling) and the
  archived 3x24 `BenchmarkLimiterProbe`. All are 1T on core 2, with 10 samples each.
  - All four probe variants are spill-audited zero-spill (`probe-audit.txt`).
  - Ctl's steady loop is byte-for-byte the shipped kernel's: 249 instructions over [148, 1140].
  - NA has 203 instructions, NB 212 and FO 166.
  - Per 8-step pass the shipped loop issues 40 A loads and 32 B loads for 160 FMAs.

**Predictions, A (net of both CIs):**
1. **Correctness:** rc=0 on both hosts.
2. **Null on both hosts:** no candidate beats the shipped (384, 72) at 8T on all three of Sgemm,
   Ssyrk and Ssymm. The n=2048 witness put the top four points within 1% of each other.
3. **No V3-only blocking win:** there is no candidate that beats shipped on V3 by ≥ 2% at 8T on
   Sgemm while not beating it on V2. This is the "blocking not tuned for V3" hypothesis, and I
   predict it **fails**.
4. **Strsm** varies ≤ 3% across arms. It does use `KCPB`, since KCDeep never reaches pre-broadcast
   kernels.

**Predictions, B** (load share = 1 − Ctl/FO):
1. **Ctl at kc=128** is within ±3% of the P2 kernel rates: 68.70 on V2 and 83.98 on V3.
2. **FO** reaches ≥ 95% of `BenchmarkPeak/neon` on both hosts at every kc.
3. **V3 is more load-bound than V2.** At kc=512, V3's load share exceeds V2's by ≥ 5 points: V3 at
   22–30% and V2 at 14–20%. The hypothesis **holds** if this does.
4. **Both load classes cost V3 more:** NA/Ctl and NB/Ctl are each larger on V3 than on V2.
5. **The share grows with depth:** V3's load share is larger at kc=512 than at kc=128, which would
   be the deep-kc slip seen at kernel level.

**What can ship from this.** `KCPB`/`MCPB` apply to every arm64 host, V1 included, and V1 is not in
this run. So a blocking win here opens a stage B that includes V1 and does not ship directly. A
load-bound V3 is a finding about the shape. Its lever, a shape with fewer loads per FMA, is a
separate pre-registered step.
