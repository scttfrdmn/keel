<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# CL 824624 PS2 (golang/go#80829) on Zen 5: predictions (#197)

**Why.** Zen 5's shipped 4x32 kernel reaches 65.3% of its own measured peak in the judged P2
(`archive/p2-amd64-anchor/`), against Zen 4's 96.5%. That is the largest kernel-term deficit on a
current CPU in the fleet. The 12-accumulator 6x32 does not rescue it, because it spills (56.1%). PS2's
reported gain, 64.3 → 78.6% of peak on Zen 5, lives only on the Gerrit thread, with no archived log.
This run gives it archived, interval-bearing evidence, and it bounds the layout term (#61: up to 45%
on Zen 5 from function placement alone).

**Arms**, built from `~/src/go` worktrees by `build-toolchains.sh`. The keel tree is the same for all
three; only the toolchain differs:

| arm | toolchain | layout |
|---|---|---|
| stock | go1.27.3, the shipped toolchain | default |
| base | `go1.28-devel_0d37007fda`, PS2's own parent | `-randlayout` seeds 1–8 |
| cl | `go1.28-devel_7745cfc920`, base + PS2 | `-randlayout` seeds 1–8 |

**Run:** one c8a.2xlarge (Turin, exploration tier), pinned to core 2. There are 5 rounds of every
binary, with base/cl order alternating by round. `Kernel/{4x32,2x32}/avx512/kc=128` and
`Peak/avx512`, `-benchtime 1s`.

**Measured before the run** (`spill-audit -goarch amd64 -mode spill`, `internal/vec` steady-state
K-loop):
- Kernel4x32 goes from 50 insns with 12 reg copies (stock) and 48 / 12 (base) to **36 insns with 0
  copies (cl)**.
- Kernel2x32 goes from 74 / 8 (stock) and 66 / 8 (base) to **58 / 0 (cl)**.
- Base also has **0 anchor NOPs**, where go1.27.3 has 2 and 8, so master already drops the
  inlined-wrapper anchors (#125's subject).

**Predictions:**
1. **4x32 gains at least 15% under cl** over base, median of medians across seeds. PS2's unarchived
   claim is about +22%.
2. **Layout floor:** within each fork arm, the 4x32 seed spread (max/min − 1 of the seed medians) is
   at most 5%, and **cl's slowest seed beats base's fastest**.
3. **The stock anchor:** stock 4x32 reads 63–67% of stock `Peak/avx512` (P2: 65.3%).
4. **2x32 also gains under cl**, by at least 5%.
5. **`Peak/avx512` moves at most 2% under cl** against base. This one is low-confidence: the peak loop
   is independent FMA chains, which T28 showed can carry copies too. A miss changes the
   percent-of-peak denominator, and the result is reported in GFLOP/s either way.

**REFUTED if** cl's 4x32 median is within base's seed spread. The copies would then be gone without a
measurable rate effect on Zen 5, and the 65% would not be this defect's.

These are patched-toolchain experiment numbers and never a published row (#197 guardrail).

## Result (`0510f94`, `exploration-zen5-0510f94.log`)

One c8a.2xlarge (read back as `AMD EPYC 9R45`, 8 cores, load 0.22 at start). It took 7m28s, and the
fleet was verified down by a readable inventory. There are 255 benchmark rows, 5 rounds × 17 binaries
× 3 benchmarks. **Not refuted; three predictions held, one split, one missed.**

| GFLOP/s, median of seed medians | stock | base | cl | cl/base |
|---|---|---|---|---|
| Kernel 4x32/kc=128 | 188.1 | 185.3 (seeds 180.4–188.5) | **239.6** (213.4–241.9) | **+29.3%** |
| Kernel 2x32/kc=128 | 153.2 | 153.4 | 152.6 | −0.5% |
| Peak/avx512 | 288.1 | 288.2 | 288.2 | +0.0% |
| 4x32 as % of its arm's peak | 65.3% | 64.3% | **83.1%** | |

1. ✓ **4x32 gains +29.3%** (predicted ≥15%). PS2's Gerrit figure, 64.3 → 78.6%, is exceeded:
   **83.1%** here.
2. **Split.** Base's seed spread is 4.49% ✓. **cl's is 13.4% ✗**, because seeds 2 and 8 read 213.4 and
   226.6 against 238–242 for the rest. Separation holds ✓: cl's slowest seed, 213.4, beats base's
   fastest, 188.5. Rates sit on discrete levels: base at 180 / 182 / 188, and cl at 213 / 227 / 234 /
   239 / 242. That is placement modes, not noise, and seed 2 switched mode between rounds. **With the
   copies gone, the kernel is three times as sensitive to layout**, which is consistent with a loop
   now limited by the front end rather than by copies.
3. ✓ **Stock reads 65.3%**, matching P2's judged figure exactly.
4. ✗ **2x32 does not move:** −0.5%, although its 8 copies are gone too (74 → 58 insns). So copies are
   not 2x32's limit on Zen 5. Its 16 FMAs per iteration carry 8 broadcasts, against 4 in 4x32.
5. ✓ **Peak/avx512 is unchanged**, so every percent-of-peak above shares one denominator.

**What it attributes:** about half of Zen 5's kernel gap to Zen 4 (65.3 → 83.1, against Zen 4's 96.5)
is golang/go#80829's accumulator copies, removed by CL 824624 PS2. The remaining ~17 points are not
copies.

**Not settled here:** 4x32 is dispatched on Zen 5, and 2x32 only on the issue class. This run is one
host on the exploration tier, single-threaded, and makes no claim about the nest.

**VOID (2026-11-02), by `archive/kernel4x32-strict/`:** the sentence above that the copy-free loop's
layout sensitivity is "consistent with a loop now limited by the front end rather than by copies".
Removing 10 more instructions per iteration did not move the kernel under either toolchain, so Zen 5's
4x32 is not front-end bound. The measured figures above stand (+29.3%, 83.1% of peak, the 13.4%
seed spread). Only that reading of them is withdrawn.
