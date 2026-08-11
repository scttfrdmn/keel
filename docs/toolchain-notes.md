# Toolchain field notes — GOEXPERIMENT=simd

A first-class deliverable (DESIGN.md §7 rule 8): every surprise from the
experimental simd packages — missing intrinsics, unexpected lowerings,
allocator behavior, API changes between releases — gets an entry here with
a minimal repro *before* any workaround lands. Entries feed upstream issues.

| Date | Toolchain | Observation | Repro | Upstream issue |
|---|---|---|---|---|
| 2026-08-10 | go1.26.5 | `simd/archsimd` is amd64-only; no vector type exists on other GOARCHes | [T1](#t1) | none — documented upstream |
| 2026-08-10 | go1.26.5 | `MulAdd` lowers to `VFMADD213PS`, not the `231` form DESIGN.md predicted | [T2](#t2) | none — not a defect |
| 2026-08-10 | go1.26.5 | No `Abs` and no bitwise ops on any float32 vector type | [T3](#t3) | candidate |
| 2026-08-10 | go1.26.5 | No horizontal-reduce op for `Float32x16` | [T4](#t4) | candidate |
| 2026-08-10 | go1.26.5 | No portable `simd` package; only `simd/archsimd` exists | [T5](#t5) | none — expected in 1.27 |
| 2026-08-10 | go1.26.5 | `Max`/`Min` operand order for NaN/±0 not yet verified on hardware | [T6](#t6) | open question |

All repros below were run on `go1.26.5 darwin/arm64` with Homebrew's Go.
Where a repro needs amd64 it cross-compiles, which is enough for anything the
compiler decides but not for anything the CPU decides — see T1.

---

## T1 — `simd/archsimd` is amd64-only; this dev host cannot execute any vector op

**Observation.** The package doc says "It currently supports AMD64", and that
is load-bearing rather than aspirational: every vector type lives in an
`*_amd64.go` file. On any other GOARCH the package still *imports* and still
exports the `X86` feature struct, whose methods are defined for all
architectures and return false. So code that only feature-detects compiles and
runs anywhere and silently reports "no SIMD", while code that names a vector
type does not compile at all.

```
$ ls $(go env GOROOT)/src/simd/archsimd/ | grep -c amd64
9
$ GOEXPERIMENT=simd GOARCH=arm64 go doc simd/archsimd.Float32x16
doc: cannot find package "simd/archsimd.Float32x16" in any of: ...
```

Feature detection on the arm64 host, which compiles and runs cleanly:

```go
//go:build goexperiment.simd

package main

import ("fmt"; "runtime"; "simd/archsimd")

func main() {
	fmt.Printf("arch=%s AVX512=%v AVX2=%v FMA=%v\n", runtime.GOARCH,
		archsimd.X86.AVX512(), archsimd.X86.AVX2(), archsimd.X86.FMA())
}
```

```
$ GOEXPERIMENT=simd go run .
arch=arm64 AVX512=false AVX2=false FMA=false
```

**Consequence for keel.** This is not a defect — it matches the documented
scope — but it is a hard constraint on the build plan, because keel's dev host
is `darwin/arm64`:

- *Verifiable here:* anything the compiler decides. The scalar spec, and the
  vector backends' compilation and lowering, via `GOARCH=amd64` cross-compile
  plus `-gcflags=-S`. Gate P0's FMA criterion is fully checked this way.
- *Not verifiable here:* anything the CPU decides. The differential tests
  cannot execute the AVX2 or AVX-512 backends, so gate P0 is red on this host
  by design (it refuses to call scalar-only coverage a pass). Nor can P1's
  ≥4× benchmark, P2's 55%-of-peak go/no-go, or P3's OpenBLAS comparison run
  here at all.

Rosetta 2 does not close the gap: it provides no AVX support, so an
`amd64` build would feature-detect false and take the scalar path.

Tracked as issue #7; needs an amd64 AVX-512 host to proceed past P0.

## T2 — `MulAdd` lowers to `VFMADD213PS`, not `VFMADD231PS`

**Observation.** DESIGN.md §4/P0 and §6 name `VFMADD231PS` as the instruction
the FMA wrapper must disassemble to. The compiler emits the `213` form
instead. This is not a surprise in the compiler — archsimd's own doc comment
states it:

```
$ GOEXPERIMENT=simd GOARCH=amd64 go doc simd/archsimd.Float32x16.MulAdd
// MulAdd performs a fused (x * y) + z.
//
// Asm: VFMADD213PS, CPU Feature: AVX512
```

Confirmed in the emitted assembly for keel's own wrapper:

```
$ GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 GOAMD64=v3 go build -gcflags=-S ./internal/vec/
...vec.FMA512 STEXT nosplit size=7
	VFMADD213PS	Z2, Z1, Z0
	RET
```

**Assessment.** `VFMADD132PS`, `VFMADD213PS` and `VFMADD231PS` are the same
FMA instruction under three operand-order encodings; they differ only in which
operand supplies the addend and which is overwritten. Latency, throughput and
port assignment are identical, so nothing about DESIGN.md's performance
reasoning changes. DESIGN.md simply predicted the wrong encoding.

**Consequence for keel.** `scripts/gate-p0.sh` checks the property DESIGN.md
was actually reaching for — *one* instruction from the
`VFMADD{132,213,231}PS` family, and *zero* separate `VMULPS`/`VADDPS`, in the
wrapper body — and prints the mnemonic it found on every run so the
substitution stays visible rather than becoming folklore. Flagged to Scott as
a wording question on DESIGN.md; the gate was not weakened, and if the
wrapper ever degrades to mul+add the gate still fails.

## T3 — no `Abs`, and no bitwise ops at all, on float32 vector types

**Observation.** Across all three float32 widths there is no `Abs`, `And`,
`AndNot`, `Or`, `Xor` or `Neg`:

```
$ GOEXPERIMENT=simd GOARCH=amd64 go doc -all simd/archsimd \
    | grep -cE 'func \(x Float32x(4|8|16)\) (Abs|And|AndNot|Or|Xor)\('
0
```

The integer vector types have the full bitwise set, plus reinterpreting
conversions in both directions:

```
$ GOEXPERIMENT=simd GOARCH=amd64 go doc simd/archsimd.Int32x16 | grep -E 'And|AsFloat'
func (x Int32x16) And(y Int32x16) Int32x16
func (x Int32x16) AndNot(y Int32x16) Int32x16
func (x Int32x16) AsFloat32x16() Float32x16
```

**Workaround (landed).** `vec.Abs512` bitcasts to `Int32x16`, clears the sign
bit, and bitcasts back. `AndNot` is documented as Go's `x &^ y` — not x86's
reversed `VPANDN` operand order — so it mirrors the scalar spec's
`bits &^ signMask32` exactly rather than approximately. The `As*` conversions
are reinterpretations, so the cost is the mask broadcast plus one `VPANDND`:

```
...vec.Abs512 STEXT
	MOVL	$-2147483648, AX
	VMOVD	AX, X1
	VPBROADCASTD	X1, Z1
	VPANDND	Z0, Z1, Z0
	RET
```

**Open follow-up.** The mask broadcast sits inside the function body here.
Whether it hoists out of a loop when `Abs512` is inlined into one is not yet
verified; it matters for `Sasum`/`Isamax` in P1, not for SGEMM. Tracked as
issue #8.

## T4 — no horizontal reduce for `Float32x16`

**Observation.** There is no `ReduceSum`-style op on any float32 width, and
the pairwise-add ops are unevenly distributed: `Float32x4` has `AddPairs`,
`Float32x8` has `AddPairsGrouped`, and `Float32x16` has neither.

```
$ GOEXPERIMENT=simd GOARCH=amd64 go doc -all simd/archsimd \
    | grep -E 'func \(x Float32x(4|8|16)\) (AddPairs|.*Reduce)'
func (x Float32x4) AddPairs(y Float32x4) Float32x4
func (x Float32x8) AddPairsGrouped(y Float32x8) Float32x8
```

`Float32x4` also has no `GetLo`/`GetHi`, being the narrowest type, so a fold
cannot be carried all the way down in vector registers.

**Workaround (landed).** `vec.HSum512` folds 512→256→128 with
`GetLo`/`GetHi`+`Add`, then stores four floats and finishes the last two
levels in scalar registers. Lowers to two `VEXTRACTF64X4`, two
`VEXTRACTF128`, two `VADDPS`, a 128-bit store and three scalar adds.

**Assessment.** Costs nothing that matters. A horizontal sum happens once per
dot product and never inside a K-loop, which is why DESIGN.md's op list treats
it as incidental. It did shape the *spec*: `ScalarHSum` was written to fold in
this same halving order, so the scalar and vector reductions agree bit-for-bit
and the differential tests can demand exact equality instead of a tolerance.

## T5 — no portable `simd` package in 1.26

**Observation.** DESIGN.md §7 rule 2 instructs running both
`go doc simd/archsimd` and `go doc simd`. The second package does not exist:

```
$ GOEXPERIMENT=simd go doc simd
doc: no symbol simd in package .
$ ls $(go env GOROOT)/src/simd/
archsimd
```

**Assessment.** Consistent with DESIGN.md §8, which lists the portable `simd`
package as a 1.27 thing and parks the ARM64/NEON kernel behind it. No action
beyond noting that the P0 instruction to read `go doc simd` is unsatisfiable
on 1.26 and was not skipped through oversight.

## T6 — `Max`/`Min` operand order for NaN and ±0 is unverified

**Observation.** x86's `VMAXPS` is not IEEE-754 `maxNum`: when either operand
is NaN it returns the *second source operand*, and `max(+0,-0)` likewise
returns the second. Which of `x` and `y` in archsimd's `x.Max(y)` becomes that
second source is not stated in the doc comment, and disassembly does not
settle it — the register order in `VMAXPS Z1, Z0, Z0` is a naming convention
question that only execution resolves.

**Status.** `vec.ScalarMax`/`ScalarMin` currently specify "NaN or ±0 yields
`y`", written as `if x > y then x else y`, and say so in a doc comment marked
UNVERIFIED. `TestSpecMaxMinNaNAndSignedZero` pins that claim and
`TestDiffBinary` feeds Max/Min every ordered pair from the NaN/±0 pool, so the
first differential run on an amd64 host either confirms the spec or fails on a
named case. No workaround has been applied, because nothing is known to be
wrong yet. Tracked as issue #9.
