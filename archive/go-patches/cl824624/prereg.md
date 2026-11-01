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
