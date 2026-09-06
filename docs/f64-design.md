<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# float64: the design, if it is ever built

Design doc for the float64 (`f64`) port, written **before any kernel, shim, or benchmark** — the
same discipline the arm64 seam followed (`docs/generic-middle.md`, `#135`). It is the map `#135`
promised the type axis: *"do not cut it yet, and here is the map for when `#103` does."*

**This doc does not decide that f64 gets built.** `#103` records float64 as a v0 non-goal
(`DESIGN.md:13`, "Gonum owns it") and the finding that its real blocker is the *oracle*, not the
kernels. keel was built for the float32 gap Gonum leaves open; a float64 keel competes with Gonum's
mature float64 BLAS on Gonum's own ground. So this is the design *conditional on* a decision to
pursue f64 (ruled post-freeze for attention, not a hard gate) — the engineering, costed honestly, so
the decision is made against a real plan rather than a guess. What it is **not** is a commitment to
cut it. Every number here about tile shape or tolerance is a **prediction**, labelled as such, that
a sweep or a measurement decides later.

Grounded in the tree at write time: `docs/generic-middle.md` (the seam), `internal/kern/kern.go`,
`internal/block/block.go`, `internal/pack`, `internal/l1/l1.go`, `internal/oracle/oracle.go`,
`internal/vec/vec_avx512.go` + `vec_neon.go`, `docs/neon-probe.md` / `docs/cross-isa-positioning.md`
(lane-width reasoning), and issues `#103` / `#135` / `#136`. archsimd names verified against
`GOEXPERIMENT=simd go doc simd/archsimd` on the current toolchain, per the prime directive.

## 1. The genericize-the-middle boundary: what f64 inherits, what is new

`docs/generic-middle.md`'s headline: the middle is **already generic across the ISA axis** —
`internal/block`, `internal/pack`, `internal/par` carry no build tags and no lane count, and the
nest is parameterized over a `kern.Kernel` *value* (`block.go:271`, `func Gemm(kn kern.Kernel, …)`).
The **element-type axis is not cut**: `float32` is baked into the signatures. The seam value itself
is f32-typed —

```go
// internal/kern/kern.go:117
Fn func(kc int, a, b, c []float32, ldc int)
```

— and so is everything above it: `block.Gemm(kn, transA, transB, m, n, k, alpha float32, a []float32,
…)` (`block.go:271`), `float32` appears 12× in `block.go` and 8× in `pack.go` (per
generic-middle.md's audit), and `l1.Kernels` fields are `Dot func(x, y []float32) float32`,
`Axpy func(alpha float32, x, y []float32)`, … (`l1.go:37-49`).

**What f64 inherits UNCHANGED — the algorithm, not the types.** The following are type-independent in
*structure* and are not re-derived, only re-typed:

- **The blocking loop nest and its order** (`block.go`): the `jc → pc → ic` structure, whole-tile
  blocking to the kernel's `MR/NR`, the fringe handling. The logic does not know the element type.
- **k-major packing with `alpha` folded into A** (`internal/pack`, invariant I4): copy-and-scale,
  not reduce; the layout `a[p*MR+i]`, `b[p*NR+j]` is element-agnostic.
- **The spatial `ic` partition and worker pool** (`internal/par`, `block.go:75-96`): parallelism
  splits disjoint C rows, no reduction, no reassociation. Type-independent by construction.
- **The dispatch skeleton** (`dispatch.go`, `kern.Preferred`): ranking shapes by `MemOpsPerFMA` /
  `InsnsPerFMA` is arithmetic on the shape, not the type.

**What is NEW f64 code — everything at or below the seam, plus the oracle:**

- **The seam's type.** `Fn`'s `[]float32` must become `[]float64` for an f64 kernel. This forces the
  one API decision the type axis cannot avoid (§below): a parameterized `Kernel[T]`, a parallel
  `Kernel64` type, or generated copies. Until that is chosen the seam cannot carry an f64 kernel at
  all — this is the type axis's equivalent of "the seam does not care what MR×NR the sweep returns."
- **The f64 shim** (`internal/vec`, new `_f64` surface). New code, and it collides at the naming
  layer: the shim suffix is **register width in bits, not lane count** (`Add512`, `FMA512`,
  `LoadPart512` — `vec_avx512.go:37-76`). A float64 add at 512 bits *is* `Add512` by that scheme,
  colliding with the float32 one (`#103` §3). So the shim must put the element type in the name
  (`Add512f64` / `AddF64x8`) or go generic over the element — an API decision forced before the
  first kernel.
- **The f64 microkernel family** (`internal/kern`, new `Kernel` values + `internal/vec` bodies).
- **The f64 oracle and a second tolerance model** (`internal/oracle`) — §3, the actual blocker.

**The refusal register carries forward (generic-middle.md).** Any `switch type { … }` inside
`block`/`pack`/`par` is the mode-flag failure: if the middle needs to be told the element type to
behave correctly, f32 and f64 were lookalikes there, and the answer is two implementations, not a
branch. The lift to `[T]` (option 1 below) is the *opposite* of a type-switch — it makes the type a
parameter the compiler monomorphizes, with no runtime branch — and that is the distinction the build
unit must hold: parameterize, never branch.

### The type-mechanism choice — presented, deferred (per #135 criterion 4)

generic-middle.md records three ways to cut the type axis; this doc does not pick one, because
`#135` criterion 4 forbids a lift justified by an anticipated consumer, and there is exactly **one**
existing consumer of the type axis (f32). The pick is the build unit's, made against a real f64
kernel:

1. **Go type parameters** — `Gemm[T float32 | float64](kn Kernel[T], …)`. Cleanest source. Risk:
   `archsimd` is not generic over element type, so `Fn`'s body cannot be — the type parameter stops
   at the seam and the kernel table is per-`T` regardless; and generic codegen under
   `GOEXPERIMENT=simd` is itself unmeasured.
2. **Code generation** — one templated nest per type. No generics risk, two copies that cannot
   silently share a bug. Cost: a generator in `scripts/`, which the apparatus budget counts.
3. **Two hand-maintained copies** — rejected on its face (the divergence `#135` exists to prevent).

The seam being *already ISA-generic* means f64 does not re-touch `block`/`pack`/`par` logic; it
re-types their signatures once, by whichever mechanism wins. That is the leverage `#135` bought.

## 2. Kernel shapes — a falsifiable prediction, the sweep decides

archsimd exposes the f64 vector types (verified, current toolchain): `Float64x2`, `Float64x4`,
`Float64x8`, with `LoadFloat64x2` / `LoadFloat64x2Array` / `LoadFloat64x2Part` / `BroadcastFloat64x2`
present and the surface symmetric with f32 up through `Float64x8.MulAdd` and the masked/partial
loads (`#103` §1, verified on go1.26.6 and the types re-confirmed here; load/store follow the T23
go1.27 spelling — bare name = slice form, `Array` suffix = array form, as the f32 shim already uses).

**f64 has half the lanes of f32** — `Float64x8` holds 8, where `Float32x16` holds 16; `Float64x2`
holds 2, where `Float32x4` holds 4. The reflected-tile model (`kern.go` `MemOpsPerFMA` =
`1/MR + Lanes/NR`, with `Lanes` the lane count and live accumulators = `MR · NR/Lanes`) says the
right move is **halve the columns, keep the rows and the accumulator count**, because that preserves
both the arithmetic intensity and the register budget:

**PREDICTION (falsifiable; the sweep is the authority, this is not):**

| ISA | f32 shipped | Lanes | f64 predicted | f64 accum = MR·(NR/Lanes) | MemOpsPerFMA (=f32's) |
|---|---|---|---|---|---|
| AVX-512 | `4×32` | 16→8 | **`4×16`** | 4·(16/8) = 8 | 1/4 + 8/16 = **0.75** |
| AVX-512 | `2×32` ×4 | 16→8 | **`2×16` ×4** | 2·(16/8) = 4 | 1/2 + 8/16 = **1.0** |
| NEON | `4×16` | 4→2 | **`4×8`** | 4·(8/2) = 16 | 1/4 + 2/8 = **0.5** |
| NEON | `8×8` | 4→2 | **`8×4`** | 8·(4/2) = 16 | 1/8 + 2/4 = **0.625** |

The predicted f64 tiles carry the **same accumulator count** as their f32 analogs (8 on AVX-512's
`4×16`, the fit frontier 16 on NEON's `4×8`) and the **same `MemOpsPerFMA`**, so the register-budget
and arithmetic-intensity arguments that selected the f32 shapes carry over — *shape*-wise. What does
**not** carry over, and is the whole reason `#103` says "re-derive, not rescale": each lane is 8
bytes not 4, so **bytes-per-flop doubles and peak FLOP/s halves**, which moves `KC`/`MC`/`NC`,
`PEAK_FLOOR`, and `SWEEP_BEST_IPF` — all measured against the f32 shape (`#103` §3).

**Kernel6x32's ghost — why this is only a prediction.** On amd64, `Kernel6x32` was predicted to fit
and spills 13 (`docs/spill-report.md`); on arm64, `Kernel8x12` was predicted to fit (28 live ≤ 32)
and spills 5, because the naive live-set undercounts scheduler transients (`docs/neon-probe.md`,
`#136` step 3). The f64 spill audit must be **re-run from scratch** (`#103` §3): half as many values
per register at the same 32-register (arm64) / mask-limited (amd64, `neon-probe.md` §2d: the shipped
`Kernel6x32` allocates `Z16`–`Z23`, so the usable set is an empirical question not a mask reading)
budget is a *different* allocation problem. So the table above is the sweep's **starting hypothesis**,
falsifiable exactly as the f32 and NEON sweeps falsified theirs. P2 is a go/no-go for f64 as it was
for f32; a predicted-fit tile that spills gets reclassified to `referenceTiles`, not shipped.

## 3. Gonum as the numerical oracle — what the differential proves, and what it cannot

**The blocker, restated (`#103` §2).** keel's entire correctness argument is *testing float32
against a wider type*: `internal/oracle` accumulates in float64, and `Tolerance` (oracle.go:55) bounds
the drift by `C·f(n)·(eps32·scale + eta32/2)` with float32 unit roundoff `eps32 = 2^-23`. **At
float64 this construction is unavailable** — a float64 reference for a float64 routine is the same
precision as the thing under test, so it validates nothing about accumulation order, the exact defect
class the oracle exists to catch. The tolerance constants are hardcoded to the narrow type and cannot
be retyped in place.

**The plan, in two layers, because one oracle cannot do both jobs at f64:**

**Layer A — Gonum, for cross-implementation agreement (broad, cheap).** Gonum's
`gonum.org/v1/gonum/blas/blas64` (Gonum-backed) is a mature, independent float64 BLAS. It is **not
currently a keel dependency** (`go.mod` has only `x/perf`), so adopting it is itself a design
decision to record. The differential runs every routine keel would add — `Ddot Daxpy Dscal Dnrm2
Dasum Idamax`, `Dgemv Dger`, `Dgemm Dsyrk Dsymm Dtrsm` — against Gonum on the same adversarial shapes
and strides the f32 suite uses.

- **Tolerance set by the problem, not by how close they land** (DESIGN.md §5 rule 1). A second
  tolerance model, `Tolerance64`, mirrors the existing derivation with the f64 constants:
  `C·f(n)·(eps64·scale + eta64/2)`, `eps64 = 2^-52 ≈ 2.22e-16`, `eta64 = 2^-1074`. It lives in
  `internal/oracle` beside `Tolerance`, changed via a numerics comment, never a per-test epsilon —
  the same rule the f32 model follows. The bound is the f64 reduction-error model, derived, not
  tuned to observed agreement.
- **Bit-exactness expectation: none across implementations.** Where keel's kernel fuses via
  `MulAdd`/`VFMLA` and Gonum does not (or fuses differently), and wherever the two reduce in a
  different order, the results differ within `Tolerance64`. keel keeps its *own* bit-stability
  guarantee internally (I1: bit-identical across thread counts, a keel-vs-keel property), but
  keel-vs-Gonum is tolerance-based, not bit-exact.

**What Layer A proves, and what it does not (the `#103` subtlety, head-on).** Gonum-vs-keel-f64 is
*same-precision vs same-precision*. It **proves cross-implementation agreement**: keel's f64 result
matches an independent mature f64 BLAS within the f64 error bound — which catches gross defects
(wrong indexing, mis-folded `alpha`/`beta`, transpose/uplo/side errors, a broken fringe) and any
accumulation defect large enough to exceed `Tolerance64`. It **does not prove** what the f32 oracle's
wider type proves: that keel's accumulation order is sound against a *more precise* witness. A subtle
order defect that stays within f64 roundoff is invisible to a same-precision comparison — and `#98`
showed that class can hide. So Layer A alone is a **weaker** correctness argument than keel holds at
f32, and saying so is the point.

**Layer B — a wider-than-f64 witness, for a small adversarial set (recovers the lost property).**
The wider type f64 lacks in hardware is recovered in software: a reference in `math/big.Float` at a
declared working precision (e.g. 200 bits), or a compensated/exact-accumulation reference, run on a
**small set of accumulation-adversarial shapes** (long reductions, mixed magnitudes, cancellation).
`big.Float` is slow, so it is not the broad oracle — it is the wider-type witness for the shapes
where accumulation order is the risk, exactly where Layer A is blind. This is the "different kind of
oracle" `#103` §2 says f64 needs; Layer A + Layer B together restore, for f64, the two things the f32
oracle gives in one: broad agreement (A) and a wider-precision accumulation witness (B).

## 4. The determinism invariants (I1–I7) carried forward

generic-middle.md enumerates seven, each with its check. For f64 they split three ways — **transfers
unchanged**, **transfers in mechanism but re-prove on f64 bits**, or **new (establish fresh)**:

| # | invariant | f64 disposition | why |
|---|---|---|---|
| I1 | parallel nest bit-identical to serial at every thread count | **re-prove** | The partition (spatial, no reduction, `block.go:75-96`) is type-independent, so the *mechanism* transfers — but bit-identity is per-type: `TestP5Determinism` compares `Float32bits`; an f64 version must compare `Float64bits`, re-run and shown before/after. |
| I2 | parallel B-pack byte-identical to serial pack | **re-prove** | Copy-not-reduce is type-independent; the byte comparison is over f64 bytes. New fixture (`TestBPanelsPartIsTheSerialPack` twin). |
| I3 | `GOMAXPROCS=1` uses no goroutine/atomic | **transfers UNCHANGED** | This is a `par`/`block` property and shared code; the test (`TestRunSerialUsesNoGoroutine`) asserts *no goroutine*, not bits — type-agnostic. If the middle is genericized (not re-cut), the same `par` serves both and this needs no f64 twin. |
| I4 | k-major layout, `alpha` folded into A | **re-prove** | Layout structure is type-independent; correctness re-verified via the f64 GEMM differential over `alpha,beta` combinations. |
| I5 | Trsm block interchange bit-identical | **re-prove** | The interchange is an algorithm property that transfers; the bit-identity check (`TestSolveRightInterchangeIsBitIdentical`) is per-type and needs a `Dtrsm` twin on `Float64bits`. |
| I6 | every shim op equals its scalar twin | **NEW** | The f64 shim is new code — this is *established fresh*, not carried. New `vec_diff` tests over the f64 ops. It is the executable-spec property for a spec that does not exist yet. |
| I7 | microkernel spill frontier (≤ accum budget, no calls, `InsnsPerFMA` matches audit) | **NEW / re-run** | The f64 kernels are new and the allocation problem differs (half the values per register, `#103` §3). The spill audit re-runs from scratch; `InsnsPerFMA` is re-audited per f64 shape. The audit tool reads assembly and is type-agnostic at the listing level, so it *works* — but the frontier it measures re-derives. |

**Gaps carried forward (generic-middle.md G1–G2):**

- **G1 — the fringe add-back's backend is not pinned by any check.** `internal/block` inlines a
  scalar add-back, so nothing diverges today. f64 must keep this property: the add-back stays a
  scalar inline (now over `float64`), *not* a `Kernel` field — the same refusal recorded at
  `kern.go`'s doc (dispatching it lost in `#22`). If f64's add-back is ever monomorphized per
  backend, `KEEL_FORCE` must force it too, and that check still does not exist. Booked forward.
- **G2 — blocking params are host-tuned `var`s, not ISA/type-branched.** f64 retunes `KC/MC/NC`
  (they must change — bytes-per-flop doubled), and it does so by giving f64 its own tuned values,
  never a `switch type`. The guard is review, not an assertion (there is no branch to catch).

**The preservation requirement (`#135`, criterion 8):** no published number moves. For f64 this is
vacuous *for f32* — the f32 numbers must stay bit-identical through any genericization, which I1
(f32) already gates — and for f64 it means the port's own numbers are established fresh under I1–I7
re-run, not inherited.

## 5. What this doc does NOT decide (§5 rule 12)

- **Whether f64 is built at all.** `#103` records it a non-goal past v0.1.0; this is the design
  *if* it is pursued (post-freeze attention item), not a decision to pursue. The strong argument
  against — a float64 keel competes with Gonum on Gonum's ground while the f32 gap it exists for
  stays open (`README`, `#103` §4) — is unrebutted here.
- **The type mechanism** (generics vs codegen). Deferred to the build unit, decided against a real
  f64 kernel per `#135` criterion 4 — there is one existing type-axis consumer, and a lift needs two.
- **The sweep's actual tile winner.** §2 is a falsifiable prediction; a predicted-fit tile that
  spills is reclassified, not shipped. P2 is a go/no-go for f64.
- **The real tolerances.** `Tolerance64`'s constant `C` (the FMA/reassociation slack) and the
  Layer-B working precision and shape set are set when measured, not now. §3 states the *model* and
  its derivation, not the fitted numbers.
- **Adopting Gonum as a dependency**, and the Layer-A/Layer-B boundary (which shapes need the
  `big.Float` witness) — both design decisions the build unit makes with measurements in hand.
- **The shim naming resolution** (element-in-name vs generic-over-element) — forced before the first
  kernel, decided with the type mechanism, not here.
