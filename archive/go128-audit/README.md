<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# keel's kernels under go1.27.4 against Go master (go1.28-devel_f9d574dc87, 2026-11-25)

This is a static audit. `spill-audit` ran on every non-test kernel in `internal/vec`, for both
architectures, under go1.27.4 (now the dev host's toolchain) and under master at `f9d574dc87`. No
host was measured, so it is free to re-take. The files are `audit-<arch>-{go,tip}.txt`. keel
builds and tests cleanly on master on darwin/arm64, linux/amd64 and linux/arm64, with and without
`GOEXPERIMENT=simd`, except for one test (adoption cost 1 below).

Each cell is steady-state K-loop instructions / vector stack refs / reg copies / anchor NOPs, as
go1.27.4 → master:

| kernel | go1.27.4 | master |
|---|---|---|
| amd64 `Kernel2x32` (shipped, issue class) | 74 / 0 / 8 / 8 | **66** / 0 / 8 / **0** |
| amd64 `Kernel4x32` (shipped, FMA class) | 50 / 0 / 12 / 2 | **48** / 0 / 12 / **0** |
| amd64 `Kernel6x32` (12-accumulator reference) | 219 / 44 / 45 / 8 | 211 / 44 / 45 / 0 |
| arm64 `KernelPB5x16` (shipped) | 249 / 0 / 0 / 0 | **213** / 0 / 0 / 0 |
| arm64 `KernelPB7x12` | 264 / 0 / 0 / 0 | 224 / 0 / 0 / 0 |
| arm64 `Kernel3x24` | 148 / 0 / 12 / 63 | **67** / 0 / 0 / 0 |
| arm64 `Kernel4x16` | 80 / 0 / 8 / 30 | 40 / 0 / 0 / 0 |
| arm64 `Kernel8x8` | 92 / 0 / 16 / 32 | 43 / 0 / 0 / 0 |
| arm64 `Kernel4x32` / `8x16` (spilling) | 161 / 36 / 17 / 46, and 169 / 38 / 19 / 48 | 98 / 28 / 12 / 0, and 103 / 28 / 13 / 0 |

The float64 twins move identically.

**What it settles:**
- **Anchor NOPs** (T9, #17, #125, golang/go#80830) are 0 in every keel loop on master, on both
  architectures.
- **arm64 reg copies** are 0. That is the broadcast fold, CL 866091; see `../go-patches/cl866091/`.
- **amd64 accumulator copies are unchanged** at 8 and 12. That is golang/go#80829, and CL 824624
  (ours) is still the only fix.
- **Kernel6x32's 72 moves per iteration** are unchanged in number but now VEX-encoded: `VMOVUPS`, 72,
  against legacy `MOVUPS`, 72, under go1.27.4. That is golang/go#80835 (CLs 825185/825186): the
  AVX-SSE transition penalty goes and the spills stay.
- **The shipped arm64 kernel's 249 → 213 instructions is not NOPs or copies**, since it had none.
  The cause is unattributed.

**Adoption costs for a go1.28 toolchain** (none applies to go1.27.x):
1. `internal/spill`'s `TestARM64SeesRealSpiller` pins 5 spills for `Kernel8x12`, and master gives 3.
   This is attributed to CL 866091.
2. gate-p3 recomputes each shipped shape's `InsnsPerFMA` from the audit and fails on any
   disagreement. The constants move:
   - amd64: 2x32 74/16 → 66/16, and 4x32 50/8 → 48/8;
   - arm64: the pre-broadcast shapes 249/160 → 213/160, and 7x12 264/168 → 224/168;
   - float64: the same.

   The class rankings these constants feed are preserved. Issue class: 2x32 is still below 4x32
   (4.125 against 6.0). arm64: 5x16-pb is still below 7x12-pb, but now only by 1.33125 against
   1.33333, against 1.556 / 1.571 before. That margin is narrow enough to state on adoption.
