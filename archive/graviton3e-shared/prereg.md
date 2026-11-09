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

## Result (`2151ed3`, `run-2151ed3.log`): H-nop REFUTED. The 3E breaks Graviton3's 2.0 FMA/cycle counter-loop cap on some loops, at idle

One c7g.16xlarge and one c7gn.16xlarge, core 2, idle, 10 samples per row. The fleet was verified
down by a readable inventory. **The core identity is identical:** MIDR_EL1 `0x411fd401` (r1p1, part
0xd40) and REVIDR_EL1 `0x1` on both, with the same kernel `7.0.0-1014-aws`.

| loop (GFLOP/s, median and range) | c7g | c7gn | c7gn/c7g |
|---|---|---|---|
| family `fo` | 78.14 | 78.16 | +0.0% |
| family `x44` | 41.42 | 41.44 | +0.0% |
| family `shared` (shim, 16 NOPs) | 41.41 | 42.72 (42.61–42.80) | +3.2% |
| PeakProbe shim16 (the same loop) | 41.41 | 43.41 (43.28–43.60) | +4.8% |
| **PeakProbe raw16 (NOP-free twin)** | **41.42** | **53.00** (52.82–53.37) | **+27.9%** |
| PeakVariants raw16 / x44 / x44v | 41.43 / 41.43 / 41.43 | 52.29 / 42.36 / 42.53 | +26.2 / +2.2 / +2.7% |
| **PeakVariants vals** | **41.42** | **66.11** (64.69–67.45) | **+59.6%** |
| raw24, x36, every V1Probe variant | 41.42–41.45 | 41.41–41.42 | ±0.1% |
| float64 rawD16 / shimD16 / Dx44 | 20.71 / 20.71 / 20.71 | 26.71–26.87 / 21.93 / 21.51 | +29 / +5.9 / +3.8% |

1. ✓ **The gap reproduces:** family `shared` +3.2%, with `x44` and `fo` within 0.1%.
2. ✗ **H-nop is refuted.** The NOP-free twin raw16 is **+27.9%** on the 3E, against the predicted
   ±0.5%. The NOPs *shrink* the 3E's lead. They do not cause it.
3. ✗ **NOP-free variants also differ:** raw16, vals, both x44 test variants and the float64 twins all
   read higher on the 3E.
4. **Core identity:** MIDR and REVIDR match, so the difference is below what the ID registers show.
   That points at firmware or power-management configuration, though nothing here measures it.

**What it shows.** On the c7g, **every loop except fo runs at exactly 2.0 FMA/cycle** (41.42 at 2.588
GHz) at idle on one core: 16, 18 or 24 chains, shared or distinct multiplicands, raw or shim. That is
`archive/neon-raw/`'s open "V1 counter-loop ceiling". On the 3E the same cap binds for some loops
(raw24, x36, the V1Probe set) and **not for others**, which run at 2.1–3.2 FMA/cycle with a visibly
wider spread (vals 64.69–67.45). The production-shaped loops, fo and the 5x16-pb kernel
(`archive/graviton3e/`), match exactly between the types. Which loop property escapes the 3E's cap
is **unattributed**. It is not chain count alone, since raw16 escapes while raw24 does not.

**Consequence for #207.** "Alone on a core they run the same rate" holds for production-shaped code
(fo, the shipped kernel and every keel 1T row bar two), not for every FMA loop. Graviton3's limiter
also acts at idle on one core, as a hard 2.0 FMA/cycle cap on these loops, and the 3E's is
selectively looser even at idle. The multi-core throttle and this cap may be one mechanism or two.
Nothing here separates them.
