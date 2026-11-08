<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The idle shared-loop gap between Graviton3 and Graviton3E: predictions

**Why.** Scott, 2026-11-08: "measure the unexplained idle shared-loop gap before publishing".

On an idle core, Graviton3E (c7gn ×2, hpc7g) runs the `shared` family loop at 43.3–43.6 GFLOP/s.
Graviton3 (c7g ×3) runs it at 41.33–41.43, about 4.8% lower. `x44` and `fo` match across the types
(`archive/graviton3e/`, `-pass/`). This is the one finding that contradicts the draft post's "same
core" framing (#207).

**What separates the loops** (spill-audit, `-goarch arm64`):

| loop | steady-state instructions | FMAs | anchor NOPs |
|---|---|---|---|
| `shared` (`neonPeak`, shim-built) | 35 | 16 | **16** |
| `x44` (raw archsimd) | 19 | 16 | 0 |
| `fo` | 6 | 1 | 0 |

`shared` is the only loop with NOPs: the 4-byte `HINT` anchors of golang/go#80830. On Graviton3
both 16-chain loops sit at exactly 2.0 FMA/cycle, the V1 "counter-loop ceiling" that
`archive/neon-raw/` could not explain. On the 3E, `shared` reads about 2.1.

**Run** (`driver.sh`). One c7g.16xlarge and one c7gn.16xlarge, idle, core 2, 10 samples each.
- **core identity:** MIDR_EL1, REVIDR_EL1 and cpuinfo's variant and revision;
- **`BenchmarkPeakProbe`:** shim16 is `neonPeak`; raw16 is the same 16 shared-multiplicand chains
  without NOPs; raw24 and the float64 twins come with them;
- **`BenchmarkPeakVariants`** and **`BenchmarkV1Probe`:** the V1 loop variants;
- **the family loops**, as a reproduction.

**Predictions:**
1. **The gap reproduces:** family `shared` on c7gn is 3–6% above c7g, while `x44` and `fo` stay
   within ±0.5%.
2. **It belongs to the NOPs (H-nop):** shim16 on c7gn is 3–6% above c7g, while **raw16, the
   NOP-free twin, is within ±0.5%** between the types.
3. **Every NOP-free variant** in PeakVariants and V1Probe is within ±0.5% between the types.
4. **Core identity** is reported, not predicted. If MIDR or REVIDR differ, that names a candidate
   cause (stepping or erratum handling). If they match, the difference is below what the ID
   registers show.

**Reading.** If prediction 2 holds, the "same core" framing survives with one qualification: the
3E handles a NOP-dense instruction stream better. NOP-free vector code, which is what a production
kernel is, runs identically. If raw16 also differs, the gap is in the shared-multiplicand FMA
pattern itself, and the post must say the two parts differ even at idle.
