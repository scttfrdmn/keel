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
