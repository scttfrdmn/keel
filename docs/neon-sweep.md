<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# NEON MR×NR sweep (`#136`): codegen read, candidate set, and what the sweep will decide

This is characterization by design — **no judged verdicts, no bars, no registry entries**. The
arm64 judged tier is gated on the `#73`/`#119` ratio-criterion extension, which is not landed. Every
artifact here is a tile-selection measurement on a characterization host, stated as such. The winner
is "the shape the sweep picked on GB10", not "keel's NEON kernel".

## Step 1 — the codegen read, done first (before any timing)

The charter's front-loaded question: read the emitted FMLA, and let the by-element question decide
which shapes are even worth sweeping. Read on the dev host (darwin/arm64), `go1.27.0`,
`GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd`, `go build -gcflags=-S`, over a keel-style BCE-clean 4×8
kernel (`ap:=a[:kc*4]; bp:=b[:kc*8]`, fixed-offset loads, broadcast-A). Three findings, all from the
object code, not from the datasheet:

1. **`Float32x4.MulAdd` emits a real fused `VFMLA`** (`VFMLA V13.S4, V9.S4, V8.S4`). The sweep is
   viable — the arithmetic is one fused instruction per multiply-add.
2. **`VFMLA` accumulates in place** — the destination *is* the addend. Unlike amd64's 213 form,
   which writes its first multiplicand and forces one scratch register per accumulator (the T10
   constraint, `kern.go:42`), arm64 needs **no per-FMA scratch**. This is why the amd64 8-accumulator
   spill frontier does not carry over. Put in one line for the record: **NEON's `VFMLA` is natively
   the 231-form accumulate that CL 1 (`#127`) teaches amd64's compiler to emit** — keel's first
   upstream CL exists because the ssa layer lowers only the 213 form and accumulation pays a copy per
   FMA; arm64 built those semantics into the instruction, so the port re-derives its tile family from
   a fresh register budget instead of translating amd64's.
3. **`BroadcastFloat32x4` emits a separate `VDUP`** (`VDUP V15.S[0], V9.S4`), not folded into the
   FMLA. archsimd exposes no by-element `MulAdd`, so the A operand costs one `VDUP` per element where
   NEON hardware's `FMLA Vd.4S, Vn.4S, Vm.S[i]` would index the lane in the FMLA itself. See `#130`.

The steady-state K-loop of the 4×8 probe is `2 FMOVQ (B loads) + 4 VDUP (A broadcasts) + 8 VFMLA`,
**call-free and spill-free**; the only `CALL`/BCE noise was in the probe's slice-`Store` epilogue,
which the real kernel writes back outside the K-loop (as the avx512 kernel does).

## Step 2 — the register-budget model, derived from that read

NEON has 32 128-bit vector registers (V0–V31); a `Float32x4` holds 4 columns. With in-place FMLA and
the broadcast-A pattern, the live set of an MR×NR tile is:

    live = MR·(NR/4)   accumulators
         +    (NR/4)   B vectors (loaded per k-step, reused across the MR rows)
         +        1    A broadcast temp   (no FMA scratch — finding 2)

So a shape fits iff `MR·(NR/4) + NR/4 + 1 ≤ 32`. This is the NEON analog of the amd64 spill audit,
and it is a *prediction to be checked against the emitted code per shape*, not a licence to skip the
audit: the sweep still reads `-S` for every shape and refuses any that spills, exactly as P2 does.

| shape | acc | +B | +bcast | live | fits ≤32? | note |
|---|---|---|---|---|---|---|
| 4×8   | 8  | 2 | 1 | 11 | ✓ | low arithmetic intensity; the read probe |
| **8×8**   | 16 | 2 | 1 | 19 | ✓ | design-doc candidate |
| 12×8  | 24 | 2 | 1 | 27 | ✓ | neighbor |
| **8×12**  | 24 | 3 | 1 | 28 | ✓ | design-doc candidate |
| **4×16**  | 16 | 4 | 1 | 21 | ✓ | design-doc candidate |
| 6×16  | 24 | 4 | 1 | 29 | ✓ | neighbor |
| 8×16  | 32 | 4 | 1 | 37 | ✗ | **predicted spill** — excluded unless the read says otherwise |
| 4×32  | 32 | 8 | 1 | 41 | ✗ | **predicted spill** |

The design-doc family (8×8, 8×12, 4×16) all fit with room; the sweep set is those three plus the
neighbors 12×8 and 6×16 to bracket the MR/NR trade, and 4×8 as the low-intensity floor. 8×16 and 4×32
are predicted to spill and are carried only as read-confirmed exclusions (Kernel6x32's ghost: the
prediction is falsifiable, the `-S` read judges it, not this table).

## Step 4 — what the sweep will measure (pre-registration, to be completed with the harness)

- **Host:** GB10 (`pollux.local` / `castor.local`, Linux aarch64, 20 cores, pueue `measured` group),
  the arm64 home judged host *of record* — but run here as characterization, since the judged tier
  awaits `#73`/`#119`. M-series is excluded until the macOS placement law (`#138`) exists; it may run
  labeled characterization cross-checks later.
- **Metric:** GFLOP/s per shape at the P3 blocking, benchstat across the shapes, plus per-shape
  `InsnsPerFMA` and `MemOpsPerFMA` from the `-S` audit — the same instruments `kern` already carries.
- **Discipline, ported from amd64:** pre-registration at full row (shape × size) granularity in the
  harness's output space before the run; `-trimpath` builds; toolchain read back from the artifact;
  evidence tracked, not gitignored; the `-S` audit read before the timing for every shape.
- **The by-element cost is instruction-count only, not throughput.** On 4×8 the 4 VDUPs are 0.5
  insns/FMA over a hypothetical by-element FMLA. Whether they cost *throughput* is unmeasurable here:
  there is no by-element API to build the other A/B arm against, so the sweep reports the shapes as
  they compile and does not attribute a throughput number to the missing instruction. That restraint
  is the `#130` discipline, not a gap in the sweep.

## What is not decided here (§5 rule 12)

- The winning shape — that is the sweep's output, not this doc's.
- Any judged verdict, bar, or registry row — gated on `#73`/`#119`, which lands in a hygiene slot
  before the first Graviton fleet campaign (`#137`).
- Whether the by-element FMLA absence is a lowering-fusion miss or an archsimd API gap — recorded on
  `#130`, resolved against the graduation thread's context (`#128`), not extrapolated from x86.

---

## Step 3 — the -S audit, graded (before any timing)

Built `GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd go build -gcflags=-S`, go1.27.0, each kernel's
steady-state K-loop read from the object code. FMLA = `MR·(NR/4)` per k-step, VDUP = `MR` (the
broadcast-A cost, whole-function count), spill = distinct accumulators the allocator moved to a
`cN_M(SP)` stack slot, CALL(loop) = calls in the steady state.

| shape | model live | pred | V-regs used | FMLA | VDUP | acc spilled | CALL(loop) | grade |
|---|---|---|---|---|---|---|---|---|
| 8×8  | 19 | fit   | 28 (→V27) | 16 | 8 | 0  | 0 | **✓ fit** |
| 4×16 | 21 | fit   | 26 (→V25) | 16 | 4 | 0  | 0 | **✓ fit** |
| 8×12 | 28 | fit   | **32 (→V31)** | 24 | 8 | **5** | 0 | **✗ SPILL — model refuted** |
| 8×16 | 37 | spill | 32 (→V31) | 32 | 8 | 13 | 0 | ✓ spill |
| 4×32 | 41 | spill | 32 (→V31) | 32 | 4 | 13 | 0 | ✓ spill |

**4 of 5 predictions correct; 8×12 refuted — the model proposes, the assembly disposes.** 8×12 was
predicted to fit (28 live ≤ 32), but the allocator uses **all 32** V-registers and *still* spills 5
of its 24 accumulators. So the naive live-set `MR·(NR/4) + NR/4 + 1` **undercounts the true peak**:
24 accumulators plus the 3 B-vectors, the broadcast, and the scheduler's transients exceed 32 at
once. The empirical fit frontier is between **16 accumulators (8×8, 4×16 fit, 26–28 regs) and 24
(8×12 spills)** — not the 28-live line the model drew. This is not an archsimd or allocator gap: the
allocator correctly uses every register (reaches V31); the model was optimistic, and the ~4–5 regs
of scheduler headroom it ignored are the correction. No upstream filing — a characterization result,
recorded here.

Every K-loop is **call-free** (CALL=0 in the steady state); the 20–44 whole-function CALLs are
write-back bounds-check stubs from the `c[i*ldc:…]` slicing, off the hot path, once per tile. VDUP =
MR confirms the by-element cost the #130 read predicted: one DUP per A element, no by-element FMLA.

Consequence: 8×12 joins the spillers. The non-spilling candidates are **8×8 and 4×16**; 8×12, 8×16,
4×32 are `referenceTiles` — measured in the sweep, not shippable, so nothing ships a spilling tile.

---

## Step 4 pre-registration — the sweep grid (before the run)

Fixed here before any GB10 timing, so the output space cannot be chosen after the fact:

- **Rows:** `BenchmarkKernel/<shape>/neon/kc=<K>` for every shape in `kern.Measured()` — **8x8,
  4x16** (non-spilling) and **8x12, 8x16, 4x32** (referenceTiles: spilling, measured not shipped) —
  × the harness's `kcs`. The quantity is **GFLOP/s** (`bench/bench_test.go`'s `rate`), a measured
  rate, comparable across shapes on one host, never a judged verdict.
- **Host:** `pollux.local` (GB10, Linux aarch64), pueue `measured` group (serialized against
  co-tenants by the daemon). `castor.local` is the identical-hardware alternate.
- **Predicted ranking, stated so it can be wrong:** among the non-spillers, the winner is whichever
  of 8x8 / 4x16 the clock picks — both are 16-accumulator, ~1.5 mem-ops/FMA shapes, so the µarch's
  load-port width and front-end decide, not the source. The spillers (8x12, 8x16, 4x32) are predicted
  **below** the non-spillers at the in-cache sizes, by the spill traffic the -S audit counted; where
  a spiller is NOT below (e.g. its extra accumulators hide memory latency at a bandwidth-bound size),
  **that disagreement between the model and the clock is the finding**, per the charter.
- **Discipline:** `-trimpath` (via `remote_build_test`), toolchain read back from the artifact,
  GOMAXPROCS=1, governor=performance asserted, a quietness refusal before the run, evidence tracked
  under `archive/neon136/`. Characterization-labeled in every line; no bars, no registry.
