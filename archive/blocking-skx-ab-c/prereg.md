<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Skylake-SP stock blocking, stage C (deep KC scoped to full-height calls): predictions (#26, #142)

**Stage B** (`archive/blocking-skx-ab-b/`): KC=1024 everywhere on the issue class lifted Sgemm, Ssymm and
Ssyrk by 8.9% to 13.0% on the published n=4096 rows, but cost Strsm 6.2% at 1T and 10.0% at 8T. The
cause is that Strsm's rank updates are thin GEMMs (MB = 64 rows), which cannot amortize a deeper
packed B panel.

**The change** (branch `exp-skx-kc`): a new `block.KCDeep` replaces KC only where `plan()` sees a call
full-height in both m and n (at least one MC block each). The issue class sets `KCDeep = 1024` and
leaves `block.KC` at 384. Thin left and right Trsm rank updates, and pre-broadcast kernels, keep KC.
`TestPlanKCDeep` pins the rule; a planted shape-blind version turns it red. `TestIssueClassKC` pins the
class keying. All suites pass on the dev host under both forced classes.

**Run:** identical to stage B, with old = main `b5ed22d` against the new branch rev. One c5n.18xlarge,
`BenchmarkScale` at n=4096, 1 and 8 threads, gate-p5's mask, 4 seeds × count 5, alternating.

**Predictions** (new/old, net of both intervals):
1. **Correctness:** the branch's suite exits 0 on the host.
2. **Strsm** does not regress: the new/old interval includes 1.00 or lies above it, at both thread
   counts.
3. **Sgemm, Ssymm and Ssyrk keep stage B's gain** to within 2 points: at least +7% at both thread
   counts.

**REFUTED for shipping if** Strsm regresses beyond its interval at either thread count, or Sgemm at
8T does not beat old net of both intervals.

## Result (`94e4dcf`, `run-94e4dcf.log`, `benchci-{old,new}.csv`): all three predictions held

One c5n.18xlarge (Xeon Platinum 8124M), masked to cores 0–7 of socket 0. It took 15m15s, and the fleet
was verified down by a readable inventory. There are 320 rows. The branch's Level-3 suite exited 0 on
the host.

| row (n=4096) | old GFLOP/s [CI] | new GFLOP/s [CI] | new/old, range net of both CIs |
|---|---|---|---|
| Sgemm 1T | 67.06 [66.80, 68.05] | 74.15 [73.66, 74.43] | **+10.6%** [+8.2%, +11.4%] |
| Sgemm 8T | 453.40 [445.00, 454.70] | 493.65 [492.20, 495.30] | **+8.9%** [+8.3%, +11.3%] |
| Ssymm 1T | 65.94 | 74.83 | +13.5% [+8.2%, +14.3%] |
| Ssymm 8T | 444.10 | 491.75 | +10.7% [+9.4%, +11.9%] |
| Ssyrk 1T | 64.94 | 72.33 | +11.4% [+7.9%, +14.8%] |
| Ssyrk 8T | 476.65 | 527.40 | +10.7% [+9.7%, +11.5%] |
| Strsm 1T | 38.45 | 38.42 | −0.1% [−1.4%, +2.1%] |
| Strsm 8T | 286.20 | 285.05 | −0.4% [−1.2%, +1.1%] |

1. ✓ **Correctness.**
2. ✓ **Strsm does not regress:** both intervals include 1.00.
3. ✓ **Sgemm, Ssymm and Ssyrk keep stage B's gain:** +8.9% to +13.5%, each at least +7.9% net of
   both intervals.

**Not refuted.** Scoping the deep KC to full-height calls keeps the whole gain and removes the Strsm
regression. **On Skylake-SP this is shippable on stock go1.27.** It moves the README's Skylake-SP rows,
so shipping needs a judged amd64 gate-p5 pair to re-derive them, and that is a cost decision for Scott.
