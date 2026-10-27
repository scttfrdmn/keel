# CL 827904 (golang/go#81352): fold arm64 broadcasts of inserted scalars

The first #197 A/B, at a common master base and never against a release.

| arm | toolchain | identity |
|---|---|---|
| base | master@46f508b3aa (the CL's own parent) | `go1.28-devel_46f508b3aa` |
| cl | base + CL 827904 PS2 | `go1.28-devel_213d6fb3d2` (Nikita Kruglov, CR+2, not submitted as of 2026-10-26) |

Both were built from `~/src/go` worktrees by `build-toolchains.sh`. The keel tree is `afac2e0` for
both, and only the toolchain differs. Exploration tier: c7g, c8g and c9g 4xlarge, us-east-1
on-demand, read back as Neoverse-V1, V2 and V3. Each round is pinned to core 2, with 10 interleaved
rounds and base/cl order alternating by round. It took 8m33s, and the fleet was verified down.
These are experiment numbers on a patched toolchain and never a published row (#197 guardrail).

**Audit, `spill-audit -goarch arm64`, steady-state K-loop insns:**

| kernel | base | cl |
|---|---|---|
| raw 3x24/1 (`KernelR3x24U1`) | 49 (6 reg copies) | 43 (0) |
| shipped shim `Kernel3x24` (3x24/2) | 142 (12) | 130 (0) |
| shipped shim `Kernel4x16` | 78 (8) | 70 (0) |

Each broadcast goes from zero-copy + insert + `VDUP` to scalar load + one `VDUP`.

**Throughput, median GFLOP/s at kc=512** (`summary.txt`). There is no OpenBLAS reference;
this is a toolchain-to-toolchain A/B on one kernel:

| kernel | V1 base → cl | V2 base → cl | V3 base → cl |
|---|---|---|---|
| raw 3x24/1 (`ctl`) | 36.53 → 46.08 (+26.1%) | 41.55 → 56.69 (+36.4%) | 49.95 → 66.15 (+32.4%) |
| shipped shim 3x24/2 | 36.14 → 46.46 (+28.6%) | 40.48 → 51.45 (+27.1%) | 47.91 → 54.86 (+14.5%) |
| raw D3x12/1 (float64) | 18.28 → 23.00 (+25.8%) | 20.75 → 28.29 (+36.3%) | 24.50 → 32.39 (+32.2%) |
| PB control (no broadcast in the loop) | 58.77 → 58.70 | 65.94 → 65.93 | 72.98 → 73.07 |

In every row, every cl round beats every base round. The PB control does not move, which
is what a patch touching only broadcasts should do.

**Pre-registered predictions** (`../../neon-raw/probe-prereg.md`): raw 3x24/1 at least 15% faster
under cl — **held** on V2 and V3 (and on V1). Raw 3x24/1 under cl at least 85% of PB under cl —
**held** on V2 (86.0%) and V3 (90.5%), **missed** on V1 (78.5%). The refutation clause (a gain
under 5%) was not triggered.

**Against #163's expectation.** #163 framed the fold as about 10% of the K-loop's instructions:
"a free codegen win". The instruction share is 12/142 = 8.5% on the shipped 3x24. The
throughput gain is 14.5–28.6%, two to three times the instruction share. The removed ops
are vector-pipe ops, the pipes the FMAs need, so counting instructions undersold the fold.
