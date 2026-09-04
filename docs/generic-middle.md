<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Genericizing the middle: one seam for arm64 and f64

Design doc for `#135`, written before any refactor commit. It states where the seam is, what
crosses it, and the nest's determinism invariants in enumerated form with the check that guards
each — so a later refactor can *re-verify* them rather than hope, and so the two future consumers
(arm64, `#136`; float64, `#103`) slot into a seam that already-shipping consumers exercise.

## Thesis: the ISA seam is already cut; this documents it and refuses to re-cut it

The headline finding, and it decides the shape of everything below: **the middle is already generic
across the ISA axis, and has been since P3.** `internal/block`, `internal/pack`, and `internal/par`
carry **no build tags** and contain **no lane count, no `Float32x16`, no `amd64`, no cache-size
constant keyed to x86** — audited, zero matches. The nest is parameterized over a `kern.Kernel`
*value* (`internal/block/block.go:271`, `func Gemm(kn kern.Kernel, …)`), and every ISA-specific
byte lives behind that value, in the three packages that already have `_amd64.go` / `_nosimd.go`
twins: `internal/vec` (the shim), `internal/kern` (the microkernels), `internal/l1`.

So `#135` is mostly **not a refactor**. Cutting a seam that is already cut, twice, is how the second
cutter makes a better copy instead of lifting — the exact failure `#135` exists to prevent. The
deliverable is this document: the seam and its contract named, the determinism invariants that were
expensive to establish written down with their checks, and the one axis that is *not* yet
generic — the element type — designed but **deferred to its real consumer**, because lifting it now
would be speculative generality with no second caller to keep it honest (`#135` acceptance
criterion 4).

Two axes, two very different states:

| axis | consumer | state | this doc |
|---|---|---|---|
| **ISA** (avx512 → +NEON) | `#136` arm64 | seam **cut**, 2 existing consumers (avx512, scalar) | document + audit it holds a third |
| **element type** (f32 → +f64) | `#103` float64 | seam **not cut**, 1 existing consumer (f32) | design, then **defer the lift** |

## The seam: `kern.Kernel`, and what crosses it

The typed-kernel interface is a plain value, `internal/kern/kern.go`:

```go
type Kernel struct {
	Name        string  // backend: "Scalar" or "AVX512"
	MR, NR      int     // tile shape (rows × columns; the tile is reflected — vectors run along N)
	Unroll      int     // k-steps per steady-state pass
	Fn          func(kc int, a, b, c []float32, ldc int)
	InsnsPerFMA float64 // this shape's audited K-loop instructions per FMA; 0 = unaudited (scalar)
}
```

**What crosses the seam, from the middle into the kernel** (`Fn`'s contract): two pre-packed,
k-major, pointer-free panels — `a` is `MR×kc`, `b` is `kc×NR` — and a C tile with row stride `ldc`.
The kernel computes `C += A·B` for one `MR×NR` tile, straight-line, no calls in the K-loop
(DESIGN.md §4/P2). The shape (`MR`, `NR`, `Unroll`) rides on the value, not as package constants, so
the nest asks the kernel its shape and blocks to whole tiles of it (`block.go:134` `Params`,
`block.go:146` `plan`).

**What stays behind the seam, and is the whole of what an ISA port touches:** `Fn`'s body and the
shim ops it calls. The NEON port adds `vec_neon.go` / `kern_arm64.go` / `l1_arm64.go` beside the
existing `_amd64` and `_nosimd` files, and registers NEON `Kernel` values from
`vectorKernels()` — and touches nothing in `block`/`pack`/`par`.

**What the kernel deliberately does *not* own**, because these are the middle's and are shared
already: packing (`internal/pack`, k-major with `alpha` folded into A), the blocking parameters and
loop nest (`internal/block`), the worker pool (`internal/par`), and the fringe add-back — which
`internal/block` inlines as a scalar loop rather than dispatching through the kernel value. That
last one is a *recorded refusal*, not an omission; see §Refusal register.

## The ISA axis (arm64): already generic, and what a port actually adds

`kern.Kernel` abstracts the ISA by construction: `Fn` is a closure, `MR/NR/Unroll` are data. The
existing consumers are the avx512 kernels (`4x32`, `2x32×4`) and the scalar reference
(`ScalarKernel`), and the nest treats them identically — two consumers already prove the seam.
arm64 is a **third**:

- **NEON `Kernel` values.** MR×NR is *re-derived*, not scaled: NEON's register file is 32×128-bit,
  so `NR=32` (two `Float32x16`) is not the NEON divisor, and the reflected-tile / spill-frontier
  argument (`kern.go:8`, `#16`) must be re-run against 128-bit lanes. That derivation is `#136`'s
  sweep, not this doc's — the seam does not care what MR×NR the sweep returns, only that a `Kernel`
  value carries it.
- **Per-host blocking, not per-ISA.** `KC=384, MC=144, NC=4096` (`block.go:121`) are already `var`,
  tuned to a host's cache hierarchy, not branched on ISA. arm64 retunes the values; it does not add
  a branch. A `switch isa { … }` over blocking params would be the mode-flag failure — refuse it and
  make them host-tuned vars, which they already are.

**Audit result (the "informed by `#129`" clause):** the probe says what the NEON *kernel* needs;
this doc's audit says the *middle* needs nothing. Grep of `internal/{block,pack,par}` for
`amd64|avx|Float32x16|\b16\b|lane|x86` returns zero. The one place a lane count could have leaked —
`MaxMR=8`, `MaxNR=64` (`kern.go:94`), which size the scalar reference's stack tile — is in `kern`,
behind the seam, not in the middle. So the ISA port adds files in three packages and edits none in
the other four.

## The element-type axis (f64): designed, deferred

This axis is **not** cut. `float32` appears in the middle's signatures 12× in `block.go`, 8× in
`pack.go`, and throughout `par` and the `Fn` signature. float64 needs the same nest over
`[]float64`, a different microkernel family, and a different shim element type.

Three ways to cut it, recorded so the decision is made once with the tradeoffs visible — but **not
taken here**:

1. **Go type parameters** — `Gemm[T float32 | float64](kn Kernel[T], …)`, packing and blocking
   generic over `T`. Cleanest source, one nest. Risks: `simd/archsimd` is not generic over element
   type, so `Fn` cannot be; the type parameter stops at the seam and the kernel table must be
   per-`T` anyway. And generic code has its own codegen questions on `GOEXPERIMENT=simd` that are
   themselves unmeasured.
2. **Code generation** — one templated nest emitted per type. No generics risk; two compiled copies
   that cannot silently share a bug. Cost: a generator in `scripts/`, which the apparatus budget
   counts.
3. **Two hand-maintained copies** — rejected on its face by the same rule that motivates `#135`: the
   second copy diverges.

**Why defer rather than decide:** acceptance criterion 4 — every lift justified by two *existing*
consumers. There is one f32 consumer of the type axis. Lifting the nest to `[T]` now, with no f64
kernel to hand it a `float64`, is generality justified by an anticipated consumer, which is the
thing the criterion forbids. The mechanism is chosen in `#103`, against a real `float64` microkernel
that exercises it — the same discipline that had the ISA axis proven by scalar+avx512 before arm64
arrives. **This doc's decision on the type axis is: do not cut it yet, and here is the map for when
`#103` does.**

## The determinism invariants, enumerated, each with its check

These were expensive to establish and are load-bearing for every published number. A refactor that
reorders any of them changes results while the tests still pass — *unless* the invariant has a check
that pins the order itself. Each row names the check, or names the gap.

| # | invariant | where | check |
|---|---|---|---|
| I1 | **Parallel nest is bit-identical to the serial nest at every thread count.** The `pc` (K-depth) loop stays serial; parallelism is a *spatial* partition of the `ic` loop over disjoint C rows — no lock, no reduction, no reassociation. | `block.go:75-96` | **`TestP5Determinism`** (`p5_test.go:205`, gate-p5 criterion 5): runs serial then every `p5Threads`, asserts bit-identical via `firstBitDiff` on `Float32bits`, **and** asserts `workers==procs` so agreement cannot come from an arm that never partitioned. |
| I2 | **The parallel B-pack equals the serial pack, byte for byte, whatever order the ranges run.** Packing copies and scales rather than reduces, so disjoint ranges need no ordering. | `pack.go:159-161` | **`TestBPanelsPartIsTheSerialPack`** (`pack_test.go:422`). |
| I3 | **`GOMAXPROCS=1` uses no goroutine, atomic, or scheduling** — the exact serial nest every published number was taken on. | `block.go:94`, `par.go` | **`TestRunSerialUsesNoGoroutine`** (`par_test.go:119`). |
| I4 | **k-major packed layout with `alpha` folded into A** (`a[p*MR+i]`, `b[p*NR+j]`; `alpha·A` not `alpha·C`, so it survives `beta≠0`). | `pack.go:4-29` | GEMM oracle differential (`gemm_test.go`) over `alpha,beta` combinations; the kernel↔scalar-tile diff (`kern_test.go`). |
| I5 | **Trsm's block interchange is bit-identical** to the un-interchanged solve (the derived-L3 reduction order). | `tri.go` | **`TestSolveRightInterchangeIsBitIdentical`** (`tri_test.go:204`). |
| I6 | **Every shim op equals its scalar twin** — the shim is the executable spec. | `internal/vec` | differential tests (`vec_diff_test.go`). |
| I7 | **The microkernel spill frontier** — ≤8 accumulators, no calls in the K-loop, `InsnsPerFMA` matching the audit. | `kern`, `internal/spill` | the spill audit (gate criterion 4) + the `InsnsPerFMA` recompute-and-compare. |

**Stated gaps, not papered over (criterion 2):**

- **G1 — the fringe add-back's backend is not pinned by any check.** `internal/block` inlines a
  scalar add-back today, so `KEEL_FORCE=scalar` and the default agree and nothing can diverge. The
  moment a future `C′` monomorphizes the add-back per backend (the named follow-up in
  `kern.go`'s `Kernel` doc), `KEEL_FORCE=scalar` must force the add-back too or a forced run stops
  describing what ran — and **no test asserts that today** because there is nothing to assert yet.
  The type/ISA seam must carry this requirement forward; it is booked here so the person who adds
  `C′` reads it.
- **G2 — nothing pins the blocking params as host-tuned rather than ISA-branched.** They are `var`,
  so there is no branch for a check to catch; the guard is the code review that refuses a
  `switch isa`, not an assertion. Stated so the absence is deliberate.

**I1 is the preservation requirement `#135` names in its own words:** *bit-identical output where the
invariants promise bit-identical output*. The refactor's exit criterion is I1–I7 re-run with
before/after results shown, I1 bit-identical at every thread count. Because I1 is already a gate
criterion, "re-verify after the refactor" is "the gate stays green with the bits unchanged", not new
apparatus.

## Refusal register: where the middle must *not* be genericized

Recorded here and at each site, naming why the cases are lookalikes rather than the same thing:

- **The fringe add-back stays a scalar inline in `internal/block`; it is not a `Kernel` field.**
  `#22`'s candidate C dispatched it through two `Kernel` function fields and lost: the indirect call
  per fringe row cost more than the scalar loop saved, worst on thin shapes where fringe tiles are
  densest. Recorded in `kern.go`'s `Kernel` doc; the struct is back to its `757acb8` shape. A shared
  add-back field would make the incumbent pay C's defect to keep a seam nobody needs.
- **`InsnsPerFMA` is per-kernel audited data (0 for scalar), not a middle concept.** It ranks shapes
  against each other and is recomputed from the audit every gate run. It does not generalize; a NEON
  kernel carries its own audited value or `0`.
- **Any `switch isa` or `switch type` inside `block`/`pack`/`par` is the refusal trigger.** If the
  middle would need to be told which ISA or element type it serves to behave correctly, the two
  cases were lookalikes at that point: keep them apart, and record it at the site.

## Refactor plan and its guardrails (for when code follows this doc)

1. This doc lands first (criterion 1). ✔ on commit.
2. Lifts are justified by the **two existing** consumers only. On the ISA axis that is
   avx512+scalar, and the audit above shows the lift is already done — so the ISA-axis code change
   is expected to be **near-empty**: at most, moving a stray constant behind the seam if the port
   surfaces one, and adding the NEON files `#136` owns. On the type axis there is no second consumer,
   so no lift (criterion 4).
3. **Fixture each caller's shape, not just the shared function** (criterion 6): when a lift does
   happen, build the arm at each existing caller (avx512, scalar) handing the *same* situation as
   different tuples, not just a test over the lifted function — a latent divergence hides in the
   caller shapes, not the callee.
4. Re-verify I1–I7 after, before/after shown. No published number moves; if one moves, stop and
   explain first (criterion 8).
5. `make stock` still builds the scalar path on a stock toolchain; `make build` still cross-builds
   `GOOS=linux GOARCH=amd64` as its second line (criterion 7).

## What this doc does not decide (§5 rule 12)

- **NEON's MR×NR** — `#136`'s sweep, re-derived against 128-bit lanes, not scaled from avx512.
- **The f64 type mechanism** (generics vs codegen) — `#103`, decided against a real f64 kernel.
- **macOS placement law** — `#138`; the M-series has no `cpufreq` and no affinity mask, so it is a
  characterization host, not a judged one, and that is orthogonal to the seam.
- **The serial B-pack Amdahl term** — `#65`. It is invariant I-adjacent (its determinism is
  preserved) and separately a thing to fix, but not here.
- **f16/bf16** — `#139`, and it is a storage variant conditional on an upstream element type, not a
  compute axis this seam covers.
