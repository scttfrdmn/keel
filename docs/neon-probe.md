<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The NEON feasibility probe

Answers [#129](https://github.com/scttfrdmn/keel/issues/129): do the amd64
lowering misses have arm64 analogs? Static only — object code and compiler
sources, no benchmark, no timing claim. Run 2026-08-30 on the dev host,
`go1.27.0 darwin/arm64`, `GOEXPERIMENT=simd`.

**Verdict: file one arm64 issue.** Of the four amd64 misses, two are *absent* on
arm64, one is *present and worse*, one is *present with the same cause and four
times the byte cost*. None is "the same finding on another ISA".

## The instrument

`internal/spill/cmd/spill-audit` is the gate-audited loop auditor, and its own
`# Portability` note says the classification "is amd64-specific… A future arm64
port needs its own register-name and stack-reference rules". So the arm64 column
here comes from a re-implementation of its two load-bearing rules — drop pseudo
lines, and exclude the function-level backward branch that is not a loop — rather
than from the tool.

That re-implementation was **positive-controlled against the tool on amd64**,
where both exist. On `Kernel4x32` it reproduces every figure the tool prints:
50 loop instructions, 8 arith, 0 vector stack refs, 4 broadcasts, 2 anchor nops,
0 calls, 0 bounds-check exits — and 12 register copies, once the 14 `VMOVDQU64`
are split into 2 memory loads and 12 register-to-register moves. Both rules
mattered: without the pseudo-drop the anchor counts triple, and without the
backward-branch exclusion the "loop" becomes the whole function.

Porting `internal/spill` properly is filed, not done here.

## Question 1 — what keel needs, and how arm64 lowers it

arm64 `archsimd` has exactly one float32 vector type: **`Float32x4`**. There is
no `Float32x8` or `Float32x16`, and no lane-indexed multiply-add under any name.
Every identifier below was copied from `go doc simd/archsimd` output, per
`CLAUDE.md`'s prime directive.

| op keel uses | arm64 lowering | status |
|---|---|---|
| `Float32x4.MulAdd` | `VFMLA` | **real**, and accumulates in place |
| `Float32x4.Add` | `VFADD` | real, 1 instruction |
| `Float32x4.Mul` | `VFMUL` | real, 1 instruction |
| `LoadFloat32x4` | `FMOVQ` | real, but a bodied wrapper — costs an anchor |
| `Float32x4.Store` | `FMOVQ` | real, bodied wrapper |
| `LoadFloat32x4Array` | `FMOVQ` | real, **bodiless intrinsic — no anchor** |
| `LoadFloat32x4Part` | `FMOVQ VMOVI FMOVD VMOV FMOVS` | multi-instruction edge path |
| `Float32x4.StorePart` | `FMOVQ VDUP FMOVD FMOVS` | multi-instruction edge path |
| **`BroadcastFloat32x4`** | **`VMOV` + `VMOV` + `VDUP`** | **emulated — see 2b** |

## Question 2 — do the amd64 misses have analogs?

### 2a. Accumulate-in-place — **ABSENT on arm64**

Half of `golang/go#80829`. Absent, and for a compiler reason rather than an
architectural gift. `simdARM64.rules:242` rotates the addend into arg0 of a
`resultInArg0` op, so the accumulator *is* the destination:

    (MulAddFloat32x4 x y z) => (VFMLA4S z x y)   // specialLower
    {name: "VFMLA4S", argLength: 3, reg: v31, asm: "VFMLA", resultInArg0: true}

Confirmed in the object code — all 16 `VFMLA`s write their own accumulator, and
the loop carries **zero** register copies:

    VFMLA V21.S4, V17.S4, V16.S4      <- V16 is accumulator and destination
    VFMLA V22.S4, V17.S4, V15.S4

amd64 lowers the same SSA op to `VFMADD213PS`, also `resultInArg0` — but there
arg0 is a *multiplicand*. The compiler emits no 231 form at all
(`grep VFMADD231` over `simdAMD64ops.go` and `simdAMD64.rules` → nothing) while
`cmd/internal/obj/x86/avx_optabs.go:1845` shows the assembler encodes
`VFMADD231PS` perfectly well. That is what `golang/go#80829`'s title asserts,
and the shipped kernel pays 12 moves for it — see "A replacement figure".

Recorded caution: `docs/upstream-plan.md` said `FMLA` writes its own accumulator
"so the miss may have no arm64 counterpart — an architectural reading, not a
measurement." The conclusion holds; the reasoning was incomplete.
`VFMADD231PS` *also* writes its own accumulator and x86 still misses it, so the
instruction set never settles this. The lowering rule does.

### 2b. Broadcast — **PRESENT on arm64, and worse than amd64**

Both backends implement broadcast as emulation in Go, from the same source
shape. amd64 (`other_gen_amd64.go:262-266`) is explicit about it:

    // Emulated, CPU Feature: AVX512F
    func BroadcastFloat32x16(x float32) Float32x16 {
        var z Float32x4
        return z.SetElem(0, x).broadcast1To16()
    }

`other_gen_arm64.go:65` is the same construct on `Float32x4`. The difference is
what the optimizer does with it. Once the scalar is in a vector register, amd64
needs **one** instruction and arm64 needs **three**:

    amd64:  VBROADCASTSS X8, Z8

    arm64:  VMOV V1.B16, V25.B16      <- copies the ZERO vector into a scratch
            VMOV V17.S[0], V25.S[0]   <- inserts the scalar into lane 0
            VDUP V25.S[0], V17.S4     <- splats, overwriting every lane

The first `VMOV` copies a zero vector whose every lane the `VDUP` then
overwrites. The zero-initialization of `var z Float32x4` is dead and survives
anyway, and it burns a scratch register (`V25`) of pressure on top. NEON does
this in one instruction — `DUP V17.S4, V17.S[0]` — from a scalar already sitting
in lane 0. **Two of the three instructions are waste, 8 of the 12 per loop
iteration.**

**This is the single clearest arm64 miss and the one worth filing.**

Two things it is *not*:

- Not the same as `.BCST`. The other half of `golang/go#80829` is about folding a
  broadcast *memory operand* into the FMA. This is about the broadcast itself
  being emulated. Related consequence, different mechanism.
- Not the lane-indexed form. `FMLA Vd.4S, Vn.4S, Vm.S[i]` would remove the need
  for any broadcast in a GEMM K-loop, and it is **not exposed by `archsimd` under
  any name**. That is a missing-API observation, not a bad-lowering one, and it
  must never be reported as "x86 `.BCST` on arm64".

### 2c. The wrapper anchor — **PRESENT, same cause, 4× the bytes per anchor**

keel [#17](https://github.com/scttfrdmn/keel/issues/17) (T9) on amd64: every
inlined non-intrinsic call costs an anchor carrying the statement's position.
Present on arm64, established by control:

| kernel | loads used | loop insns | anchors | attributed to |
|---|---|---|---|---|
| `NeonKernel4x16` | `LoadFloat32x4` (bodied wrapper) | 56 | **6** | 4× load, 2× broadcast |
| `NeonKernel4x16Array` | `LoadFloat32x4Array` (bodiless) | 52 | **2** | 2× broadcast |

Swapping the four wrapper loads for the bodiless intrinsic removes **exactly
four anchors and exactly four instructions**. The loads remain loop-varying in
both arms, so LICM is not confounding it (which is precisely what went wrong in
the first attempt — see "Corrections"). The two residual anchors land on the
`BroadcastFloat32x4` call sites, the only bodied `archsimd` function left in the
loop, which corroborates the mechanism from the other direction: **fix 2b and
these two go too.**

The difference from amd64 is price. keel #17 records a **1-byte** `XCHGL AX, AX`.
arm64 is fixed-width: `cmd/internal/obj/arm64/asm7.go:7671` gives
`case ANOOP: return SYSHINT(0)`, size 4 at `asm7.go:909`, so each anchor is a
real **4-byte** `HINT $0`. Per loop iteration that is **24 bytes on the ported
arm64 kernel against 2 bytes on shipped `Kernel4x32`** — a 12× gap driven as
much by count as by width, and the count is a property of how many bodied
wrappers the kernel calls.

Anchors are not one-per-call-site in general: four `BroadcastFloat32x4` sites
(kernel.go:29, 31, 33, 35) produce two anchors, so some statement positions are
carried by real instructions instead. Counting call sites would overstate it.

### 2d. Register allocation — **ABSENT on arm64**

keel [#18](https://github.com/scttfrdmn/keel/issues/18) (T10) is "only 15 of 32
vector registers allocatable". The amd64 mechanism is narrower than that phrasing
(`AMD64Ops.go:126-128`):

    v = buildReg("X0 ... X14")                     // 15 registers
    w = buildReg("X0 ... X14 X16 ... X31")         // 31 registers

"15 of 32" is a property of the **`v` (VEX) ops**. AVX-512 ops use `w` and get
31, and `VFMADD213PS512` is a `w31` op — so keel's shipped `Float32x16` kernels
are **not** subject to the 15-register limit. **#18's causal story needs a
re-read on the amd64 side**; recorded here, not acted on, because this probe's
scope is arm64.

arm64 has no such split. `ARM64Ops.go:145` puts all 32 in one set
(`fp = buildReg("F0 ... F31")`) and the simd ops take `fp11/fp21/fp31`
(`ARM64Ops.go:847`). The allocator demonstrably uses the whole file: the 16
accumulators land in `V0`, `V2`–`V16`, with `V17`–`V25` carrying broadcasts, B
panels and scratch.

## Question 3 — register pressure, the tile, and what the loop costs

Neither shipped tile carries over. `NR:32` at 128 bits is **8 accumulator
registers per row**, so `{MR:4, NR:32}` needs 32 accumulators + 8 B vectors + 1
broadcast = **41 > 32**. A faithful port of `Kernel4x32`'s *shape* at
`{MR:4, NR:16}` — slice-shrinking loop, pre-sliced panels, no calls in the
K-loop, one broadcast per row — fits in 16 accumulators and audits clean:

| | arm64 `NeonKernel4x16` `[132,352]` | amd64 `Kernel4x32` `[101,341]` |
|---|---|---|
| loop instructions | 56 | 50 |
| FMA instructions | 16 | 8 |
| lanes per FMA | 4 | 16 |
| **vector stack refs** | **0** | **0** |
| calls | 0 | 0 |
| bounds-check exits | 0 | 0 |
| **accumulator copies** | **0** | **12** |
| broadcast splats | 12 (3 each) | 4 (1 each) |
| anchors | 6 (`HINT $0`, 4 B) | 2 (`XCHGL AX, AX`, 1 B) |
| instructions per FMA | 3.50 | 6.25 |
| instructions per **lane**-FMA | 0.875 | 0.391 |

Both meet keel's zero-spill criterion, and the slice-shrinking idiom achieves
**full bounds-check elimination inside the arm64 K-loop** — `-d=ssa/check_bce`
reports checks only at the prologue slicing and the epilogue stores.

Two readings of the same table, and they point opposite ways:

- **Per FMA instruction, arm64 looks better** (3.50 vs 6.25) — mostly because it
  pays no accumulator copies.
- **Per lane of actual work, amd64 is 2.24× more instruction-efficient**
  (0.391 vs 0.875), because a `Float32x16` FMA does four times the work of a
  `Float32x4` one. This is the dominant fact about arm64 as a target and it has
  nothing to do with any compiler miss.

Of arm64's 56 instructions, **14 are overhead the two fixes above would remove**
— 8 of the 12 broadcast instructions and all 6 anchors — taking the loop to 42
and 2.63 instructions per FMA. On amd64 the comparable figure is the 12
accumulator copies of 50, i.e. **25.0% removable on arm64 against 24.0% on
amd64**, from entirely different misses.

## A replacement figure for CL 1, measured on the shipped kernel

`docs/upstream-plan.md` records that CL 1's "110× spill price" is refuted — the
spill report measures that quantity at 2.54×–4.37×. This probe supplies a figure
that is *directly* about the miss `golang/go#80829` names, from keel's own
shipped `Kernel4x32`, and it has two independent derivations: the gate's
`spill-audit` prints `12 reg copies`, and the classifier here reaches 12 from the
listing separately.

    VMOVDQU64   Z8, Z14           <- preserve broadcast; 213 clobbers arg0
    VFMADD213PS Z7, Z12, Z8       <- destroys Z8
    VFMADD213PS Z6, Z13, Z14      <- uses the copy
    ... 2 more such pairs ...
    VMOVDQU64   Z2, Z0            <- 8 more at the loop bottom, rotating
    VMOVDQU64   Z11, Z1              accumulators back to canonical registers

**12 register-to-register moves per 8 FMAs — 24.0% of a 50-instruction loop — in
a kernel with zero spills.** Four preserve broadcasts that `VFMADD213PS`
clobbers; eight rotate accumulators at the loop bottom. Unlike the 110×, this is
re-derivable by anyone: `GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go build
-gcflags=-S ./internal/vec/`, then read `Kernel4x32` between offsets 101 and 341.
It is a static instruction count, not a timing claim, and must be cited as such.

## Question 4 — would `internal/vec` build on arm64?

Not today, and nothing is broken. Every vector file is tagged
`goexperiment.simd && amd64`; `vec_nosimd.go` and `peak_nosimd.go` claim
`!goexperiment.simd || !amd64`. So arm64 takes the scalar path by construction,
which is why `make stock` has never had an opinion about NEON.

What a backend needs: a `vec_neon.go` at `goexperiment.simd && arm64`, the two
`_nosimd` guards narrowed to `!goexperiment.simd || (!amd64 && !arm64)`, and an
arm64 arm for `internal/spill`. `kern.MaxNR` needs **no** change — it is 64
(`internal/kern/kern.go:97`), a ceiling on any kernel's columns rather than a
tile, and a NEON tile at `NR:16` sits under it.

## Recommendation

**File one arm64 issue: the dead zero-initialization in emulated broadcast**
(2b). One defect, one generated function, a one-instruction ideal lowering, and a
four-line repro. It is the only finding here that is both arm64-specific and
clearly a compiler miss rather than a missing API — and by 2c it is worth more
than its own 8 instructions, since the two residual anchors are attributed to its
call sites.

Do **not** file the lane-indexed `FMLA` as a bug: it is an API-surface request,
and `archsimd` is explicit that not all hardware instructions are exposed.

Do **not** file the anchor separately. It is the same cause as
`golang/go#80830`'s second half; per `CLAUDE.md` the deliverable when the bug is
already open is a comment carrying a fact the issue lacks. The fact this probe
adds: the anchor is **4 bytes on arm64, not 1**, it is emitted for
`LoadFloat32x4` but not for `LoadFloat32x4Array`, and the swap removes exactly
one instruction per call site.

## Reproducing the arm64 column

The probe is not in the tree — it would need a scalar twin and a differential
test to earn a place in `internal/vec`, and it has neither. It is 60 lines in a
scratch module (`module neonprobe`, `go 1.27`), one file tagged
`//go:build goexperiment.simd && arm64`. The K-loop every arm64 figure above is
taken from:

    var (
        c00, c01, c02, c03 archsimd.Float32x4
        c10, c11, c12, c13 archsimd.Float32x4
        c20, c21, c22, c23 archsimd.Float32x4
        c30, c31, c32, c33 archsimd.Float32x4
    )
    var b0, b1, b2, b3, av archsimd.Float32x4

    ap := a[:kc*4]
    bp := b[:kc*16]

    for len(ap) >= 4 && len(bp) >= 16 {
        b0 = archsimd.LoadFloat32x4(bp[0:4])
        b1 = archsimd.LoadFloat32x4(bp[4:8])
        b2 = archsimd.LoadFloat32x4(bp[8:12])
        b3 = archsimd.LoadFloat32x4(bp[12:16])
        av = archsimd.BroadcastFloat32x4(ap[0])
        c00, c01, c02, c03 = av.MulAdd(b0, c00), av.MulAdd(b1, c01), av.MulAdd(b2, c02), av.MulAdd(b3, c03)
        av = archsimd.BroadcastFloat32x4(ap[1])
        c10, c11, c12, c13 = av.MulAdd(b0, c10), av.MulAdd(b1, c11), av.MulAdd(b2, c12), av.MulAdd(b3, c13)
        av = archsimd.BroadcastFloat32x4(ap[2])
        c20, c21, c22, c23 = av.MulAdd(b0, c20), av.MulAdd(b1, c21), av.MulAdd(b2, c22), av.MulAdd(b3, c23)
        av = archsimd.BroadcastFloat32x4(ap[3])
        c30, c31, c32, c33 = av.MulAdd(b0, c30), av.MulAdd(b1, c31), av.MulAdd(b2, c32), av.MulAdd(b3, c33)
        ap, bp = ap[4:], bp[16:]
    }

The epilogue is 16 mechanical `LoadFloat32x4(r).Add(cNN).Store(r)` lines over
`r := c[i*ldc : i*ldc+16]`, and the `Array` control is the same file with the
four B loads written `LoadFloat32x4Array((*[4]float32)(bp[0:4]))`. Build with
`GOEXPERIMENT=simd go build -gcflags=-S`, then take the loop as the extent of the
backward branch that is not the function-level stack check.

## Corrections this probe made to itself

Three of its own intermediate results were wrong and were caught before
publication. Recorded because each was wrong in a way that read as a finding:

1. **A register-pressure "knee" at 16 accumulators.** Read as an arm64 echo of
   amd64's "15 of 32". It was LICM hoisting loop-invariant B loads, so the probe
   had measured *accumulators plus hoisted invariants*, not a K-loop. Refuted by
   checking which physical registers were live (all 32, including `F16`–`F31`).
2. **Anchor counts inflated ~3×**, by counting the pseudo `NOP` lines that
   `spill.go`'s `pseudo` map deliberately drops. The tell was in the listing:
   every bare `NOP` carries `<unknown line number>`, and several appear **twice
   at one offset** — a zero-length marker, not code. Real anchors own their
   offset and carry a source line.
3. **The first anchor control was invalid and had the right answer by luck.**
   `P4Unrolled` (wrapper loads) vs `P5ArrayLoads` (intrinsic loads) reported
   5 → 0. Both actually contain **0** in-loop anchors: P4's loads are
   loop-invariant, so LICM hoisted them *and their anchors* out of the loop. The
   arms differed by hoisting, not by wrapper kind. Redone inside the kernel where
   loads vary per iteration, which is the 6 → 2 result above.

## What this probe cannot see (§5 rule 12)

- **No timing, on any host.** Every number here is a static instruction or byte
  count. Nothing says what the 14 removable instructions are *worth*; an
  instruction count is not a rate, and the 0.875-vs-0.391 per-lane figure is not
  a speed ratio either.
- **One toolchain, one host**: `go1.27.0 darwin/arm64`. No Graviton, no other
  arm64 microarchitecture; Apple silicon's issue width was never consulted, and
  NEON's two-FMA-per-cycle pipelines make instruction count an especially poor
  proxy here.
- **The arm64 column is not the gate's instrument**, only a re-implementation of
  its rules that reproduces its amd64 output exactly. A defect in a rule neither
  version has would be invisible to that control.
- **One tile shape.** `{MR:4, NR:16}` was chosen to fit 32 registers, not tuned.
  The best NEON tile is unknown and is `v0.2.0-arm64` work.
- **`DUP`/`LD1R` ideals are architectural readings.** That one instruction
  suffices, and that load-and-replicate would additionally fold the scalar load,
  come from the ISA — not from anything the compiler was observed to emit.
- **No differential test exists for the port.** `NeonKernel4x16` compiles and
  audits clean; it has never been checked against a scalar twin, so nothing here
  claims it computes the right answer. It stays out of `internal/vec` for exactly
  that reason.
- **`archsimd/doc.go` says "It currently supports AMD64"** while the arm64 files
  are generated and present. The doc is stale; whether upstream considers arm64
  supported was not investigated, and "the API exists" is not "the API is
  supported".
- **The #18 re-read (2d) is stated, not done** — an amd64 question this probe
  opened and deliberately left open.
