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
expensive to establish written down with their checks, and the one axis that was *not* yet
generic — the element type — designed but **deferred to its real consumer**, because lifting it then
would have been speculative generality with no second caller to keep it honest (`#135` acceptance
criterion 4). That consumer has since arrived: the type axis is **done, by code generation**
(`77f0459`), after Go generics were built and refused (`709840d`); see §The element-type axis.

Two axes, two very different states:

| axis | consumer | state | this doc |
|---|---|---|---|
| **ISA** (avx512 → +NEON) | `#136` arm64 | seam **cut**, 2 existing consumers (avx512, scalar) | document + audit it holds a third |
| **element type** (f32 → +f64) | `#103` float64 | **done by generation** (`77f0459`): f32 source canonical, f64 copy generated | record the mechanism, check each invariant at f64 |

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

## The element-type axis (f64): done by code generation

Deferred at `b71ba0b` for want of a second consumer, and decided by Scott's ruling in `#190`. The
generator landed with the hand-written `ScalarKernel64` as its float64 consumer; the vector float64
kernels followed a day later (`aa6c78a`, `be5238f`, `#191`). Of the three mechanisms the doc laid out, **Go type parameters** were built in full and refused on
object-code identity (`709840d`; the evidence is in §Refusal register), **two hand-maintained copies**
were rejected on their face, and **code generation** shipped: `tools/f64gen` writes `internal/pack64`
and `internal/block64` from `internal/pack` and `internal/block` (`77f0459`), and later
`internal/l164` and `internal/l264` from `internal/l1` and `internal/l2` (`9ef368a`, `4781a3b`,
`#192`). The float32 source stays the only source. The transform renames the identifier `float32`, the package and the `pack`
import, and maps three `kern` selectors to hand-written twins in `kern64.go`; any other `kern`
selector is an error, so a new dependency of the middle must be decided rather than copied. The
generator is a Go program, not shell, so it costs the apparatus budget nothing.

**Held by:** `TestGeneratedIsCurrent` (`tools/f64gen/gen_test.go`) fails `go test ./...` on a stale
or orphaned copy, and `TestDriftCheckCanFail` shows a one-token source change is seen. **Float32 did
not move:** object code is instruction-identical before and after for `panels`, `depthPanel`,
`solveRight`, `trsmRight` and `gemm` on amd64 and arm64 (`77f0459`), and normalized `objdump` of
`internal/l1` is identical on linux/amd64, linux/arm64 and darwin/arm64, with a planted-difference
control (`57e2202`). The one lift that did change float32 object code, Level 2's bodies moving into
`internal/l2` behind an `l1.Kernels` parameter (`eefa858`), was measured instead: 27 of 28
`BenchmarkL2Par` rows within 0.986–1.020, the 28th bimodal in both arms, over 7 of the 16 moved
routines (`22daa94`). README rows that moved in `#198`/`#199`/`#200` moved because the peak
denominator was redefined, not because of this axis.

**f64gen copies no tests.** Each float64 check below is hand-written (or a hand port of the float32
file), so a float32 invariant check does not reach float64 by generation and is listed separately.

## The determinism invariants, enumerated, each with its check

These were expensive to establish and are load-bearing for every published number. A refactor that
reorders any of them changes results while the tests still pass — *unless* the invariant has a check
that pins the order itself. Each row names the check, or names the gap; the last column is the
float64 check over the generated copy (see §The element-type axis for why it is separate).

| # | invariant | where | check | float64 check |
|---|---|---|---|---|
| I1 | **Parallel nest is bit-identical to the serial nest at every thread count.** The `pc` (K-depth) loop stays serial; parallelism is a *spatial* partition of the `ic` loop over disjoint C rows — no lock, no reduction, no reassociation. | `block.go:75-96` | **`TestP5Determinism`** (`p5_test.go:205`, gate-p5 criterion 5): runs serial then every `p5Threads`, asserts bit-identical via `firstBitDiff` on `Float32bits`, **and** asserts `workers==procs` so agreement cannot come from an arm that never partitioned. | **`TestP5Determinism64`** (`l3_64_test.go:366`) and **`TestP5DeterminismD`** (`p5_64_test.go:145`), both bit-identical with the `workers==procs` control; **`TestF64NestParallelIsSerial`** (`internal/block64/f64_test.go:168`) at the nest. |
| I2 | **The parallel B-pack equals the serial pack, byte for byte, whatever order the ranges run.** Packing copies and scales rather than reduces, so disjoint ranges need no ordering. | `pack.go:159-161` | **`TestBPanelsPartIsTheSerialPack`** (`pack_test.go:422`). | **`TestBPanelsPartIsTheSerialPack`** (`internal/pack64/part_test.go`, `#135`) on `Float64bits`, with its range-rejection sibling. Positive control: a one-element bump on partitioned calls turns it red. |
| I3 | **`GOMAXPROCS=1` uses no goroutine, atomic, or scheduling** — the exact serial nest every published number was taken on. | `block.go:94`, `par.go` | **`TestRunSerialUsesNoGoroutine`** (`par_test.go:119`). | The same check: `internal/par` is not generated, and `internal/block64` calls it (`par.Serial`, `par.Run`) from the same generated lines. |
| I4 | **k-major packed layout with `alpha` folded into A** (`a[p*MR+i]`, `b[p*NR+j]`; `alpha·A` not `alpha·C`, so it survives `beta≠0`). | `pack.go:4-29` | GEMM oracle differential (`gemm_test.go`) over `alpha,beta` combinations; the kernel↔scalar-tile diff (`kern_test.go`). | The `dgemm_test.go`, `l3_64_test.go` and `tri64_test.go` oracle sweeps; `TestScalarTile64MatchesScalarTile` and `TestVectorAgainstScalarTile64` (`internal/kern`). |
| I5 | **VOID since `4d38e95` (#181).** Was: Trsm's block interchange is bit-identical to the un-interchanged solve. `solveRight` is now `solveLeft` on a transpose, held to `oracle.Tolerance`. | `tri.go` | `TestSolveRightMatchesOracle` (tolerance); the bit-identity test is retired. | VOID at f64 too. `TestF64NestTrsm` (`internal/block64/f64_test.go:411`) and `TestDtrsmSweep`, both tolerance. There is no nest-level twin of `TestSolveRightMatchesOracle`; the routine sweep covers the solve. |
| I6 | **Every shim op equals its scalar twin** — the shim is the executable spec. | `internal/vec` | differential tests (`vec_diff_test.go`). | `vec_diff64_test.go` (`TestDiff64*`, `TestSpec64MulAddIsFused`). |
| I7 | **The microkernel spill frontier** — ≤8 accumulators, no calls in the K-loop, `InsnsPerFMA` matching the audit. | `kern`, `internal/spill` | the spill audit (gate criterion 4) + the `InsnsPerFMA` recompute-and-compare. | gate-p2 at `KEEL_ELEM=f64` audits the `KernelD*` set per arch (`scripts/gate-p2.sh:157-185`); `TestF64RegisterCopiesAreCounted` (`internal/spill`). |

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
- **G1 and G2 hold at float64 unchanged.** The generated nest inlines the same scalar add-back and
  reads the same `var` blocking params, so the same absence applies to both copies.

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
- **The element type is not a Go type parameter (#135 F2, refused 2026-10-25).** Built in full as
  `79249cb` and reverted: `pack`/`block` generic over `float32 | float64`, `kern.KernelOf[T]` behind a
  `Kernel` alias, per-type blocking on a `*nest[T]` receiver, and a float64 test of all six Level-3
  routines on the scalar kernel, which passed. Of the three acceptance criteria, two passed:
  float32 output bit-identical (a sha256 over every L3 routine's output, old against new, on the
  M4 Pro and janus's AVX-512), and orion's `measured` A/B was flat (M4 Pro, 6 interleaved passes, one
  thread, dev tier: 0 of 57 rows with disjoint ranges; geomean of medians `BenchmarkNest` −0.09%,
  `BenchmarkTrsmMB` +0.07%, `BenchmarkPackDirections` −0.79%). The third failed:
  **object-code identity**. go1.27's gcshape stenciling passes every non-inlined generic function a
  dictionary in the first integer argument register, so every argument moves one register over, the
  frame grows 8–16 bytes, and the allocator spills differently. Of 18 hot-path functions, 15 differ
  in body on arm64 and 18 of 18 on amd64; on arm64 `depthPanel`, `symPanel` and `scaleTri` differ
  only in the morestack trampoline. Dictionary loads appear at generic call sites (sub-dictionaries)
  and for `make([]T)` / `*apBuf[T]` type descriptors; there are no new indirect calls. The hunks
  read by hand (arm64 `panels`/`macro`, amd64 `depthPanel`/`macro`) sit outside the per-element
  loops; the other functions' hunks were counted, not read. It is a property of the compiler's
  generics implementation, not of this source, so no reshaping removes it: the published numbers
  describe object code a type parameter would replace. The mechanism falls back to **code
  generation** (`#103`): the float32 source stays canonical and a float64 copy is generated from it.

## Refactor plan and its guardrails (for when code follows this doc)

1. This doc lands first (criterion 1). ✔ on commit.
2. Lifts are justified by the **two existing** consumers only. On the ISA axis that is
   avx512+scalar, and the audit above shows the lift is already done — so the ISA-axis code change
   is expected to be **near-empty**: at most, moving a stray constant behind the seam if the port
   surfaces one, and adding the NEON files `#136` owns. On the type axis the second consumer is the
   float64 kernel family, and the lift is code generation (`77f0459`), not a shared function.
3. **Fixture each caller's shape, not just the shared function** (criterion 6): when a lift does
   happen, build the arm at each existing caller (avx512, scalar) handing the *same* situation as
   different tuples, not just a test over the lifted function — a latent divergence hides in the
   caller shapes, not the callee.
   **Across element types this does not apply:** generation leaves no function shared by f32 and
   f64, so each type has its own compiled copy and its own callers. Within one type, the lifts that
   did happen take a backend value, and each is tested through its public caller rather than on
   its own. `l1.Kernels` (`57e2202`) is tested by the per-backend loops that swap the package-level
   `activeL1` (`forEachBackend`, `l1_test.go:138`; `TestBackendsAgree`, `:484`) and `activeL164`
   (`forEachBackend64`, `l1_64_test.go:104`; `TestBackendsAgree64`, `:286`), which keep the public
   routine's checks, stride split and rescue in the tested path. `l2.<Name>(kn, …)` (`eefa858`;
   16 call sites in `keel.go`/`l2band.go`/`l2sym.go`, and 16 of `l264.<Name>` in `l2_64.go`) is
   tested by the public-routine tests. **Coverage stated (§5 rule 12):** only
   `Sgemv`/`Sger`/`Dgemv`/`Dger` iterate every backend in-process (`gemv_test.go`,
   `gemv64_test.go`). The other 14 entry points per type run on the dispatched backend only: CI's
   `ubuntu-latest` default leg (whatever it dispatches, reported, not judged), its
   `ubuntu-24.04-arm` NEON leg, and its `KEEL_FORCE=scalar` leg (`.github/workflows/ci.yml`).
4. Re-verify I1–I7 after, before/after shown. No published number moves; if one moves, stop and
   explain first (criterion 8).
5. `make stock` still builds the scalar path on a stock toolchain; `make build` still cross-builds
   `GOOS=linux GOARCH=amd64` as its second line (criterion 7).

## What this doc does not decide (§5 rule 12)

- **NEON's MR×NR** — `#136`'s sweep, re-derived against 128-bit lanes, not scaled from avx512.
- ~~**The f64 type mechanism**~~ — decided since: code generation (`77f0459`), generics refused
  (`709840d`).
- **macOS placement law** — `#138`; the M-series has no `cpufreq` and no affinity mask, so it is a
  characterization host, not a judged one, and that is orthogonal to the seam.
- **The serial B-pack Amdahl term** — `#65`. It is invariant I-adjacent (its determinism is
  preserved) and separately a thing to fix, but not here.
- **f16/bf16** — `#139`, and it is a storage variant conditional on an upstream element type, not a
  compute axis this seam covers.
