<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Which tile shapes hold more than 8 accumulators without spilling, under CL 824624 PS2

**Why.** On Zen 5, keel's 4x32 kernel reaches 65% of its peak on stock go1.27.3 and 83% with CL 824624
PS2 (`archive/go-patches/cl824624/`). Removing 10 more instructions moved nothing
(`archive/kernel4x32-strict/`), so the limit is not instruction count. The candidate left is
latency: 8 accumulators on two FMA pipes leave no slack, so more accumulators may help. On stock Go
every shape above 8 accumulators spills, and the 12-accumulator 6x32 still spills under PS2. This
archive finds which shapes do not.

**Method.** `tools/shapegen -sweep`, unchanged, run twice over its 140-shape space: once with stock
go1.27.3 on `PATH`, once with `go1.28-devel_7745cfc920` (master@0d37007fda + PS2). Both shapegen
and spill-audit compile with the `go` on `PATH`. Static audit only, a linux/amd64 cross-compile on
the arm64 dev host, with no execution. The four candidates were then emitted (`k_*.go.txt`) and
re-audited by the standalone `internal/spill/cmd/spill-audit` as a second instrument.

**Result.** Four shapes above 8 accumulators are spill-free, and only under PS2:

| shape | accumulators | stock: stack refs / copies / insns per FMA | PS2: stack refs / copies / insns per FMA |
|---|---|---|---|
| 3x64 u=1 | 12 | 2 / 20 / 5.50 | **0 / 0 / 3.33** |
| 5x32 u=1 | 10 | 2 / 15 / 5.90 | **0 / 0 / 4.00** |
| 3x48 u=2 | 9 | 2 / 19 / 4.50 | **0 / 0 / 3.00** |
| 3x48 u=1 | 9 | 2 / 14 / 6.11 | **0 / 0 / 4.00** |

Both instruments agree on every row.
- **Every other shape above 8 accumulators still spills under PS2**, including 6x32 at every unroll
  (6–90 refs) and 4x48 u=1 (2).
- **The spill-free frontier therefore moves from 8 accumulators on stock to 12 with PS2**, reached by
  3-row tiles wide in N, not by DESIGN.md's 6x32.
- **This is a property of the patch** (none of the four is clean on stock), and it is upstream-relevant
  evidence for golang/go#80829.

**Not shown here:** any rate. These shapes are not in `internal/vec`. 5x32 keeps the shipped NR=32;
3x64 and 3x48 need NR=64 and NR=48 panels. Measuring them is a separate, pre-registered run.
