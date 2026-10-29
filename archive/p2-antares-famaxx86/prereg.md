<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# gate-p2 on antares.local under the family-max AVX-512 ceiling (#199): predictions

antares.local is an AMD Ryzen AI MAX+ 395 (Zen 5, Strix Halo). It is dev-tier, unrentable consumer
silicon: a **labelled lab row, never part of the citable set** (`docs/hosts.md`). Scott suggested it as a
newer x86-64 AMD part. It runs through its `measured` pueue slot, at both elements:
`KEEL_REMOTE_HOSTS=antares.local scripts/gate-p2.sh`, then the same with `KEEL_ELEM=f64`.

**What the record already says.** Three earlier gate-p2 runs on this host, under the old `avx512Peak`
ceiling (uncommitted `build/gate-p2-*.log`), read 4x32/avx512 at 210.5–211.6 GFLOP/s = **64.2–64.6% of
peak**, FMA-bound. That puts the old 1-thread f32 peak at about 328 GFLOP/s. This is the judged c8a
Zen 5 profile (64.8%), so the predictions follow that host.

Predicted:
- **x44 ties `shared` (the old `avx512Peak`) within 2%**, as on c8a's Zen 5 (+0.15%), and the family
  maximality check passes. So the canonical f32 peak reads 320–336 GFLOP/s;
- FMA-bound at both elements. 4x32 f32 at **60–68% of peak**, and 4x16 f64 within 1 point of the f32
  fraction (the judged f64/f32 pairs agreed to 0.6 points). The f64 peak is half the f32 peak, within 2%;
- GREEN at both elements: 0 FAIL, 0 UNMEASURED.

**REFUTED if** x44 beats `shared` by more than 2% (Strix Halo would then be Skylake-shaped and not
c8a-shaped), if either element is RED, or if the classifier calls this host issue-bound.

## Run 1 (1a6ca68): unmeasured, governor `powersave`

Both elements are RED (f32 21/2/1, f64 20/2/1) and **measured nothing**: the only FAILs are §5 rule 5's
governor check, and the floor is UNMEASURED for it. antares rebooted at 2026-10-29 03:44, and nothing on
the host persists `performance` (there is no cpupower unit), so the reboot left the distribution default
(`powersave`, amd_pstate active, EPP balance_performance). Everything below the timing held: both
x44 audits at 1.1875 insns/FMA, register-only. Setting the governor is a host-wide change on a shared
lab machine, so it waits on Scott. The predictions above are unchanged for run 2.

## Run 2 (be72f31): GREEN at both elements, every prediction held

Governor set to `performance` on all 32 CPUs for the run (Scott's go, 2026-10-29) and **restored to
`powersave`** in an EXIT trap afterwards, read back on all 32 (`p2-antares-r2-be72f31.log`).
- **f32 24/0/0 GREEN**: 4x32/avx512 at **212.9 GFLOP/s = 64.8% of peak** (64.6% net), FMA-bound ✓ (band 60–68).
  The family against `Peak/avx512` (x44): x44 99.9%, **shared 99.9%**, fo 90.1%. So x44 ties `shared` ✓,
  and this is c8a-shaped, not Skylake-shaped;
- **f64 23/0/0 GREEN**: 4x16 at **105.2 GFLOP/s = 64.1%** (61.7% net), FMA-bound, 0.7 points from f32 ✓;
- the peaks, derived from rate ÷ fraction because the gate prints no GFLOP/s line for them: f32 ≈ 328.5, f64 ≈ 164.1.
  f64 is half of f32 within 0.1% ✓. The f32 peak equals the old-ceiling ~328, as a tie predicts.
A labelled lab row (consumer Zen 5, Strix Halo), not part of the citable set. No README row depends on it.
