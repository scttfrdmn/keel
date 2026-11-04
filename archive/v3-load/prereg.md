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

## Result (`30c5711`, `run-30c5711.log`; `benchci-<host>-<arm>.csv` over each arm's 12 samples, `benchci-<host>-B.csv` over 10)

One c8g.48xlarge (V2, part 0xd4f) and one c9g.48xlarge (V3, part 0xd84). Each has 64 KiB L1d and
2 MiB L2 per core. L3 is 36 MiB per socket on V2 and 96 MiB on V3. Both were masked to cores 0–7.
The run took 38m11s and produced 2108 rows. The fleet was verified down by a readable inventory.
The Level-3 suite exited rc=0 on both hosts.

### B: the load probe (the shipped 5x16-pb, 1T)

| host, kc | Ctl (% of peak) | NA | NB | FO (% of peak) | load share 1 − Ctl/FO |
|---|---|---|---|---|---|
| V2, 128 | 68.81 (81.4%) | +18.2% | +15.8% | 81.88 (96.9%) | 16.0% |
| V2, 384 | 70.63 (83.6%) | +19.5% | +17.1% | 84.58 (100.1%) | 16.5% |
| V2, 512 | 71.28 (84.4%) | +18.9% | +16.6% | 84.94 (100.5%) | 16.1% |
| V3, 128 | 83.94 (81.3%) | +18.9% | +16.2% | 99.87 (96.8%) | 15.9% |
| V3, 384 | 85.91 (83.2%) | +20.1% | +17.3% | 103.30 (100.1%) | 16.8% |
| V3, 512 | 83.87 (81.3%) | +23.5% | +20.7% | 103.70 (100.5%) | 19.1% |

The family-max peak was 84.48 on V2 and 103.20 on V3.

1. ✓ **Ctl at kc=128** reproduces the P2 kernel rates: 68.81 against 68.70, and 83.94 against 83.98.
2. ✓ **FO** reaches 96.8–100.5% of the peak.
3. ✗ **V3 is only slightly more load-bound.** At kc=512 its load share is 19.1% against V2's 16.1%,
   a gap of 3.0 points, below the pre-registered 5. **At the shipped depth, kc=384, the two are
   equal: 16.8% against 16.5%.**
4. **At kc=512 only:** both load classes cost V3 more there. At 128 and 384 the gains match within
   about 1 point.
5. ✓ **The share grows with depth on V3** (15.9 → 16.8 → 19.1%). V2's is flat (16.0 → 16.5 →
   16.1%). V3's kernel-level deep-kc slip is real, and the shipped KCPB=384 sits before it.

Not pre-registered: **hoisting either load class alone recovers almost all of the gap**, on both
hosts. NA reaches 99.6–99.9% of FO, and NB 97–98%. So the loop is not short of A bandwidth or B
bandwidth separately. It is limited by load-op throughput in total. Each load class is enough to
bind on its own, and removing either one unbinds it. **The tax is about 16% on both V2 and V3 at
the shipped depth.**

### A: blocking at n=4096, 8 threads (vs shipped KCPB/MCPB = 384/72; % median [net of CI])

| arm | V2 Sgemm | V2 Ssyrk | V2 Ssymm | V3 Sgemm | V3 Ssyrk | V3 Ssymm |
|---|---|---|---|---|---|---|
| 384/36 | **+6.5** [+3.4, +9.4] | −0.5 [−0.6, −0.4] | **+6.3** [+6.1, +6.5] | +3.2 [+0.5, +3.5] | −2.0 [−2.2, −1.4] | +3.2 [+3.1, +3.6] |
| 512/36 | +6.4 [+5.8, +9.3] | −0.3 | +6.3 | +4.1 [+1.3, +4.4] | −1.1 | +4.1 |
| 512/72 | +0.8 | +0.6 | +0.8 | −0.1 | +0.2 | +0.2 |
| 384/144 | +0.5 | −1.1 | +0.7 | −2.9 | −5.1 | −4.2 |
| 512/144 | +0.5 | −2.0 | −0.7 | **−7.1** | **−9.0** | **−8.0** |

At 1T, MCPB=36 costs V3 1.3–3.1% while V2's change is within ±0.5% (−0.5% on Ssyrk). Every
other 1T cell is within ±3.6%. Strsm stays within ±3.0% on every arm.

1. ✓ **Correctness.**
2. ✓ **The prediction holds as worded, but its point is wrong.** No arm beats shipped on all three
   rows at 8T, only because Ssyrk 8T loses at MCPB=36. On Sgemm and Ssymm there is a real 8T lever:
   **+6.3–6.5% on V2 and +3.2–4.1% on V3.**
3. ✓ **There is no V3-only blocking win.** V2 gains more from the lever than V3 does.
4. ✓ **Strsm** is within ±3.0%.

**The 8T lever is partition granularity, not cache, by a model computed after the fact.** The ic
loop's units are MC-row blocks, claimed dynamically per panel (`internal/par`).
- MCPB=72 rounds to 70 rows (5-row tiles), so n=4096 makes 59 units for 8 workers. The last round
  is 3/8 busy, and the efficiency is 59/64 = 92.2%.
- MCPB=36 gives 35 rows and 118 units, an efficiency of 118/120 = 98.3%.
- The predicted gain is +6.6%, against the measured +6.5% (Sgemm) and +6.3% (Ssymm) on V2.
- Ssyrk's units are triangular, which is why it does not follow.
- V3 nets less, because its 1T cost for the smaller block (1.3–3.1%) is larger than V2's (±0.5%).

**V3's one specific sensitivity is to large blocks.** 512/144 costs V3 7.1–9.5% (1T and 8T), and V2
only 0.5–2.6%. Together with B's deep-kc slip, this says V3 degrades faster than V2 *away from* the
shipped point. The mechanism is **unmeasured**: no counter was read.

**Answer to the question.** Graviton 5 is not under-tuned relative to Graviton 4, either by the
blocking or by loads. At the shipped point both chips pay the same ~16% load tax and the same
~8% 8T partition tax, and V3's extra sensitivity appears only off that point.

Two levers are common to both chips and to the shipped code:
1. **The 8T partition tail.** A partition balanced to the worker count would take the +6% without
   the small-block cost. This is filed separately.
2. **The load tax.** A shape with fewer loads per FMA, or removing the need for pre-broadcast
   (golang/go#81352), would address it.

Nothing ships from this run.
