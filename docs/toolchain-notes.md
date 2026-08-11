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
| 2026-08-10 | go1.26.5 | `Max`/`Min` operand order for NaN/±0 — **resolved on hardware**, spec was right | [T6](#t6) | closed (#9) |
| 2026-08-10 | go1.26.5 | `GOAMD64` level does not gate archsimd intrinsics; a v1 binary runs AVX-512 | [T7](#t7) | none — load-bearing for keel |
| 2026-08-10 | go1.26.5 | CSE merges identical FMA accumulator chains; two more ways a peak kernel lies | [T8](#t8) | none — not a defect |
| 2026-08-11 | go1.26.5 | Every inlined non-intrinsic call costs a 1-byte NOP in the loop body | [T9](#t9) | filed: [golang/go#80830](https://github.com/golang/go/issues/80830) (#17) |
| 2026-08-11 | go1.26.5 | Only 15 of 32 vector registers are allocatable; no `231` FMA form exists | [T10](#t10) | filed: [golang/go#80828](https://github.com/golang/go/issues/80828), [#80829](https://github.com/golang/go/issues/80829) (#18) |
| 2026-08-11 | go1.26.5 | `GOSSAFUNC` is not in the build cache key: on a cache hit it writes no `ssa.html` but still prints `dumped SSA … to ./ssa.html` | [T11](#t11) | candidate |
| 2026-08-11 | go1.26.5 | The assembler encodes `vfmadd231ps mem{1to16}` and the intrinsic layer cannot reach it: no 231 SSA op, the load-merge rule folds memory into the addend, and nothing emits `.BCST` | [T12](#t12) | filed: [golang/go#80829](https://github.com/golang/go/issues/80829) (#20) |
| 2026-08-11 | go1.26.5 | `import "C"` in a `_test.go` file is rejected outright — a benchmark cannot call C directly, so a reference harness needs a package file | [T13](#t13) | none — long-standing, not simd |

All repros below were run on `go1.26.5 darwin/arm64` with Homebrew's Go.
Where a repro needs amd64 it cross-compiles, which is enough for anything the
compiler decides but not for anything the CPU decides — see T1. Where a repro
needed *execution*, it ran on the amd64 hosts in docs/hosts.md, and says which.

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
- *Not verifiable here:* anything the CPU decides. Nothing on this host can
  execute an AVX2 or AVX-512 op, so the differential tests, P1's ≥4×
  benchmark, P2's 55%-of-peak go/no-go and P3's OpenBLAS comparison are all
  unrunnable locally.

Rosetta 2 does not close the gap: it provides no AVX support, so an
`amd64` build would feature-detect false and take the scalar path.

**Resolution (2026-08-10).** Cross-compile the test binary and execute it
elsewhere. `go test -c` with `CGO_ENABLED=0` produces a static, pure-Go ELF
binary, so an amd64 Linux box with nothing but `sshd` — no Go toolchain, no
libc match — can run the very artifact this toolchain produced.
`scripts/remote.sh` does the ship-and-run; `scripts/gate-p0.sh` treats the
local host and each remote host as *execution targets* and requires that one
of them exercised all three backends, naming it in the gate output. Because
`go test -c` also compiles `Benchmark*`, the same path carries P1 and P2's
measurements. Two hosts are configured (docs/hosts.md) and gate P0 is green on
both. Issue #7 closed.

The one thing that does not cross the wire is compile-time introspection —
`ssa.html` for P2's spill audit — which is generated by the local compiler
anyway and never needed a CPU.

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

## T6 — `Max`/`Min` operand order for NaN and ±0 — resolved, spec was right

**Observation.** x86's `VMAXPS` is not IEEE-754 `maxNum`: when either operand
is NaN it returns the *second source operand*, and `max(+0,-0)` likewise
returns the second. Which of `x` and `y` in archsimd's `x.Max(y)` becomes that
second source is not stated in the doc comment, and disassembly does not
settle it — the register order in `VMAXPS Z1, Z0, Z0` is a naming convention
question that only execution resolves.

**Resolution (2026-08-10).** `y` is the second source. `x.Max(y)` returns `y`
when either operand is NaN, and `max(±0, ∓0)` returns `y`, which is exactly
what `vec.ScalarMax`'s `if x > y then x else y` already specified. Confirmed by
`TestDiffBinary/{avx512,avx2}/{Max,Min}` over every ordered pair from the
NaN/±0 pool, at bit-exact equality, on both hosts in docs/hosts.md:

```
$ ./scripts/gate-p0.sh
        [vesta.local] AMD Ryzen 9 7950X3D ... keel-backends-exercised: scalar avx512 avx2
  PASS  [vesta.local] internal/vec tests pass
        [janus.local] Intel(R) Core(TM) i9-9960X ... keel-backends-exercised: scalar avx512 avx2
  PASS  [janus.local] internal/vec tests pass
```

Both microarchitectures were checked on purpose. This is a property of the ISA
rather than of one chip, so a single agreeing machine would have been weaker
evidence than it looked — Zen 4 and Skylake-X are independent implementations
of `VMAXPS`, and they agree.

**Assessment.** The guess in the spec was correct, which is luck rather than
method: the honest position before execution was that it was a coin flip
documented as one. What made it cheap to settle was writing the differential
test to cover the case and marking the claim UNVERIFIED, so the answer arrived
the moment hardware appeared instead of becoming a wrong-answer bug in
`Isamax` three phases later. The `UNVERIFIED ON HARDWARE` marker is now
replaced by a `VERIFIED ON HARDWARE` note naming both machines. Issue #9
closed.

## T7 — `GOAMD64` does not gate archsimd intrinsics

**Observation.** `GOAMD64` selects the baseline instruction set the compiler
may use for *ordinary* Go code, so the natural assumption is that AVX-512
intrinsics need `GOAMD64=v4` (or at least `v3` for AVX2). They do not:
archsimd's intrinsics are emitted identically at every level, because the API
is an explicit request rather than an autovectorization opportunity.

```
$ for lvl in v1 v3; do
    n=$(GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 GOAMD64=$lvl \
        go build -gcflags=-S ./internal/vec/ 2>&1 \
        | grep -cE 'VFMADD213PS|VPANDND|VEXTRACTF64X4'); echo "GOAMD64=$lvl -> $n"; done
GOAMD64=v1 -> 11
GOAMD64=v3 -> 11
```

Not just emitted — executed. The binary that ran the full differential suite on
both AVX-512 hosts was built at the default `GOAMD64` (v1):

```
$ go env GOAMD64          # empty, i.e. v1
$ GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/vec/
$ scp ... && ./vec.test -test.v
    keel-backends-exercised: scalar avx512 avx2
PASS
```

**Assessment.** This is the property keel's distribution story depends on, so
it is worth having pinned rather than assumed. DESIGN.md §3 plans one binary
that feature-detects at runtime and dispatches to the widest available backend.
That plan only works if a `GOAMD64=v1` build — what `go install` produces for a
user who sets nothing — can still contain and reach the AVX-512 path. It can.
The runtime guard (`archsimd.X86.AVX512()`) is therefore the only thing
standing between a v1 binary and an illegal instruction on an old CPU, which
raises the stakes on that check appreciably: there is no build-tag backstop
underneath it. Gate P0 already refuses to accept a backend that was compiled in
but never executed, which is the same concern from the other side.

## T8 — CSE merges identical FMA accumulator chains (and two neighbours)

**Observation.** A register-only FMA-saturation kernel — N independent
accumulator chains, no memory, used as the measured percent-of-peak denominator
(DESIGN.md §4/P2, issue #11) — does not measure what it looks like it measures
if the chains are written the obvious way. Three separate ways to get it wrong
showed up in one file, and all three are invisible in the Go source.

**(a) Identical initial values collapse every chain into one.** Twelve
accumulators all starting at `Zero512()`, all fed `FMA512(x, y, aᵢ)` with the
same `x` and `y`, are twelve copies of the same SSA expression, and CSE keeps
one:

```
$ cat > /tmp/cse/peak_amd64.go <<'GO'
//go:build goexperiment.simd && amd64
package p
import "simd/archsimd"
func Peak(iters int, out []float32) {
    x, y := archsimd.BroadcastFloat32x16(1), archsimd.BroadcastFloat32x16(1)
    var a0, a1, a2, a3 archsimd.Float32x16   // four chains, all starting at zero
    for i := 0; i < iters; i++ {
        a0 = x.MulAdd(y, a0); a1 = x.MulAdd(y, a1)
        a2 = x.MulAdd(y, a2); a3 = x.MulAdd(y, a3)
    }
    a0.Add(a1).Add(a2.Add(a3)).StoreSlice(out)
}
GO
$ GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go build -gcflags=-S ./... 2>&1 \
    | sed -n '/Peak STEXT/,/RET/p' | grep -E 'VFMADD|VADDPS|INCQ|JGT'
	VFMADD213PS	Z0, Z1, Z2        # one FMA in the loop, not four
	INCQ	DX
	JGT	34
	VADDPS	Z0, Z0, Z0        # ... and the reduction is x+x, twice
	VADDPS	Z0, Z0, Z0
```

One chain measures FMA *latency* (~4 cycles), not throughput. With twelve chains
intended, the measured "peak" comes in ~8× low — and as a denominator, that
inflates every percent-of-peak figure by ~8×. A P2 gate would have passed a bad
kernel while displaying what looked like a hardware measurement.

*Fix:* start each accumulator at a distinct value (1…N). Nothing is left for CSE
to merge. The result is then an exact function of how many chains survived
(`vec.PeakKernel.Witness`), so `TestPeakChainsAreIndependent` detects a
recurrence by arithmetic, on any host, with no assembly grep.

**(b) `VFMADD213PS` clobbers a multiplicand, so the natural accumulator form
costs 26 register copies per iteration.** `x.MulAdd(y, z)` is `(x*y)+z` and
lowers to `VFMADD213PS` (T2), whose destination is the *first multiplicand*.
Written as `a = FMA(x, y, a)`, the destination cannot be `a`, so the allocator
preserves `x` around every FMA:

```
	VMOVDQU64	Z0, Z13
	VFMADD213PS	Z1, Z0, Z0
	VMOVDQU64	Z13, Z14
	VFMADD213PS	Z2, Z13, Z13
	...                        # 12 FMAs, 12 copies, then 14 more for the phis
```

38 uops per iteration against a 4-wide allocator is ~9.5 cycles of dispatch for
6 cycles of FMA work: on a dual-512-FMA Skylake-X the loop would have measured
issue width and reported roughly two thirds of the real ceiling. (Register moves
are latency-free and may be eliminated at rename, but they still take issue
slots.)

*Fix:* put the accumulator in the destination position — `a = FMA(a, y, x)`,
i.e. `a*1 + 1`. The dependency then runs through a multiplicand rather than the
addend, which costs nothing on Zen or Skylake (FMA latency is operand-uniform),
and the loop body becomes exactly twelve `VFMADD213PS`, one `INCQ`, one `CMPQ`,
one `JGT`. Zero copies.

**(c) Constant multiplicands delete the multiply and add a load.** In the scalar
reference kernel, `const x, y = float32(1), float32(1)` means `x*y` folds at
compile time. The loop body came out as ten `ADDSS` and no multiply — half the
flops the harness was counting — plus a `MOVSS $f32.3f800000(SB), X8` *inside*
the loop, violating the no-memory property the kernel's whole claim rests on:

```
	MOVSS	$f32.3f800000(SB), X8      # a load, per iteration
	ADDSS	X8, X7
	...                                # 10 adds, 0 multiplies
```

*Fix:* obtain the multiplicands from a `//go:noinline` function returning 1. The
call sits outside the loop; the loop body becomes ten `MULSS` and ten `ADDSS`,
all register operands.

**Assessment.** None of these is a compiler bug — (a) and (c) are CSE and
constant folding doing exactly their jobs, and (b) is a reasonable lowering of an
API whose destination operand is fixed by the ISA. What they have in common is
that the Go source reads correctly in all three cases while the measurement is
wrong by 2×–8×, and always in the direction that flatters the code being
measured. Two conclusions, both now standing practice in this repo: a
microbenchmark that claims a hardware ceiling is not credible until its
steady-state loop has been read in disassembly, and where the property can be
restated as arithmetic it should be, so a test rather than a human catches the
regression. `internal/vec/peak.go` states the four properties; the arithmetic
witness guards two of them, and `scripts/gate-p2.sh` greps the loop for the
third.

---

## T9 — every inlined non-intrinsic call costs a 1-byte NOP in the loop body

**Observation.** The P2 spill audit's first clean run on the peak kernel
reported a steady-state loop of **27 instructions carrying 12 FMAs**. The
expected body is 15: twelve `VFMADD213PS`, `INCQ`, `CMPQ`, `JGT`. The other
twelve are 1-byte NOPs, encoded `XCHGL AX, AX` (`0x90`), one per FMA:

```
	0x0067 00266 (peak_amd64.go:33)	XCHGL	AX, AX
	0x0068 00267 (peak_amd64.go:34)	XCHGL	AX, AX
	...                                            # twelve, one per source statement
	0x006b 00278 (vec_avx512.go:72)	VFMADD213PS	Z1, Z0, Z2
	...                                            # twelve FMAs, all at vec_avx512.go:72
	0x0083 00350 (peak_amd64.go:47)	INCQ	CX
```

Note the source positions: the NOPs are at the *caller's* lines and the FMAs are
all at one line inside `internal/vec`, which is what points at the cause.

**Cause: a statement-position anchor.** Go's line table has to be able to say
"execution is at `peak_amd64.go:33`" for a debugger, a profile sample and an
inlined traceback. When a statement's only instruction is an FMA that has been
attributed to the *callee's* line — because `vec.FMA512` was inlined and the
instruction inherited its body's position — nothing in the emitted code carries
the statement's own position, so the compiler emits a 1-byte NOP to carry it.
One statement whose work all moved into an inlinee, one anchor.

**It is inlining, not statements.** Four separate statements and one
tuple-assignment of four calls emit the same four NOPs, because the anchor
belongs to each *inlined call site*, not to each `;`. And calling the intrinsic
directly emits none at all:

| Shape | NOPs in the loop body |
|---|---|
| `a0 = a0.MulAdd(x, x)` ×4, separate statements | **0** |
| `a0, a1, a2, a3 = a0.MulAdd(...), ...`, one statement | **0** |
| `a0 = fma(a0, x, x)` ×4 through a one-line wrapper | **4** |
| `a0, ... = fma(a0,...), ...` one statement, same wrapper | **4** |

**The line that divides the two halves of that table is bodyless vs emulated.**
`MulAdd` is declared with no body in `archsimd/ops_amd64.go`:

```go
func (x Float32x16) MulAdd(y Float32x16, z Float32x16) Float32x16
```

It is a compiler intrinsic, so the instruction is emitted *at the call site* and
carries the call site's position — there is no callee position to lose, and no
anchor is needed. A one-line Go wrapper around it is an ordinary function that
gets inlined, so its body's position wins and the caller's statement buys a NOP.

The same split exists *inside* `archsimd`. Ops in `ops_amd64.go` are bodyless
intrinsics; ops in `other_gen_amd64.go` are marked `// Emulated` and have real
Go bodies:

```go
// Emulated, CPU Feature: AVX512F
func BroadcastFloat32x16(x float32) Float32x16 {
	var z Float32x4
	return z.SetElem(0, x).Broadcast1To16()
}
```

So a broadcast costs an anchor NOP *and* lowers to two instructions
(`VMOVSS` then `VBROADCASTSS`) rather than the one a hand-written
`VBROADCASTSS mem, Z` would use. That matters much more in a microkernel, where
six broadcasts happen per k-step, than in the peak kernel, where they are
hoisted out.

**Cost, stated carefully.** A 1-byte NOP occupies no execution port and has zero
latency, but it is decoded, allocated and retired like any other instruction, so
it consumes front-end and retire bandwidth. Twelve FMAs on a machine with two
512-bit FMA units are 6 cycles of *execution*; 27 instructions through a 4-wide
allocator are 6.75 cycles of *issue*. On Skylake-X, therefore, the NOPs are
plausibly the binding constraint in `avx512Peak` — which would mean the measured
peak on janus (215.9 GFLOP/s, docs/hosts.md) is understated by up to ~12%, and
that P2's percent-of-peak bar is correspondingly *easier* than it should be.
Confirming that needs a cycle counter this repo does not yet have, so it is
recorded as a caveat on the denominator rather than as a correction to it. On
Zen 4 and Zen 5 the front end is 6-wide and double-pumped or full-width FMAs
take ≥6 cycles per twelve, so there is headroom and no effect is expected.

**Not suppressible by the obvious flags.** `-gcflags=-dwarf=false` (no debug info
wanted) and `-gcflags=-l=4` (maximal inlining) both leave the count unchanged: the
anchors are for the pprof/traceback line table, not for DWARF, and they are emitted
after inlining rather than by it. `-gcflags=-N` does remove them, and is not a
workaround — with optimizations off, each statement's values go to the stack, so
real instructions carry the caller's own positions and no anchor is needed:

```
$ go build -gcflags='all=-S -N' .   # 0 XCHGL; the positions ride on stores instead
	0x0041 00065 (main.go:24)	MOVSS	main.s+1912(SP), X1
	0x009c 00156 (main.go:24)	VMOVDQU64	Z1, main.a0+1368(SP)
```

(An earlier version of this note cited `-gcflags=-N=0`, which is the *default*
— `-N` is a boolean — and therefore said nothing. Corrected 2026-08-11 while
preparing the upstream report.)

**Consequence for keel, and the shape of the fix.** Every op in a keel hot loop
goes through a one-line `internal/vec` shim today, so every op in every hot loop
costs one extra instruction. The microkernel makes ~20 shim calls per k-step,
which is ~20 NOPs against 12 FMAs.

The fix does not require abandoning the rule that all `simd` imports live in
`internal/vec`, because `vec.F32x16` is a type *alias* for
`archsimd.Float32x16`. A caller in `internal/kern` can therefore write
`acc.MulAdd(a, b)` — the aliased type's own bodyless intrinsic method, at the
call site, with no anchor NOP and no `simd` import to grep for. The shim
functions remain the documented spelling for everything that is not a hot loop.
That is a P2 kernel-shaping step (issue #17), taken after this note rather than
before it, and it is measured in `KERNEL.md`.

**Assessment.** Not a bug: the line table is doing its job, and the alternative
— a hot loop whose profile samples all land on one line inside a helper package
— would be worse for everyone who is not counting instructions. It is a real
abstraction cost of the shim pattern, though, and one that no amount of reading
Go source will reveal. Worth an upstream conversation about whether a
`//go:linkname`-style "transparent wrapper" marker is wanted, since keel's shim
layer is the shape every simd-using library will grow.

---

## T10 — the register allocator can use 15 of the 32 vector registers, and only the `213` FMA form exists

**Date:** 2026-08-11 · **Toolchain:** go1.26.5, `GOEXPERIMENT=simd`, linux/amd64
· **Issue:** #18 · **Upstream:** filed as [golang/go#80828](https://github.com/golang/go/issues/80828) (property 1) and [golang/go#80829](https://github.com/golang/go/issues/80829) (property 2)

This is the note P2 turns on. Two independent properties of go1.26.5 combine to
cap what a Go SGEMM microkernel can hold in registers, and the cap is below what
`DESIGN.md` §4/P2 budgeted for.

### Property 1: SIMD values are allocated out of 15 registers, not 32

`cmd/compile/internal/ssa/regalloc.go:942` picks the candidate register set for
a value by type:

```go
if t.IsSIMD() {
    if t.Size() > 8 {
        return s.f.Config.fpRegMask & s.allocatable
    } else {
        // K mask
        return s.f.Config.gpRegMask & s.allocatable
    }
}
```

and `s.allocatable` (`regalloc.go:721`) is

```go
s.allocatable = s.f.Config.gpRegMask | s.f.Config.fpRegMask | s.f.Config.specialRegMask
```

so for a 512-bit value the candidate set is exactly `fpRegMask`. On amd64
(`opGen.go:96120`):

| mask | value | bits | registers |
| --- | --- | --- | --- |
| `fpRegMaskAMD64` | `2147418112` = `0x7FFF0000` | 16–30 | **X0–X14** |
| `specialRegMaskAMD64` | `71494644084506624` | 49–55 | K1–K7 |
| `gpRegMaskAMD64` | `49135` = `0xBFEF` | 0–13, 15 | integer, minus BP/g |

`registersAMD64` numbers `X15` as 31 and `X16`–`X31` as 32–47. Bit 31 is
excluded deliberately — X15 is the ABI's zero register. Bits 32–47 are in *no*
mask at all, so **X16–X31 are not allocatable to any value**, and half the
architectural register file is unreachable.

This is not visible from the operation definitions, which is what makes it
surprising. `_gen/simdAMD64ops.go` gives `VFMADD213PS512` the register shape
`w31`, and `w31`'s masks list X16–X31 among the legal operands. The op says the
instruction may use Z16; the allocator will never offer it one.

**Repro** (`/tmp/hireg`, 20 independent register-only FMA chains with distinct
starts so CSE cannot merge them — see [T8](#t8) for why that matters):

```
$ GOEXPERIMENT=simd spill-audit -pkg . -func chains -mode nomemory
chains: steady-state loop [575,905] 49 insns: 20 arith, 25 vector stack refs, 1 reg copies, 0 calls, 0 other mem refs
chains: 25 memory reference(s) in a loop that must be register-only:
    00575 (/tmp/hireg/main.go:11)	VMOVDQU64	Z0, main.a19(SP)
    …
$ … -v | grep -oE '\bZ[0-9]+' | sort -u | tail -1
Z14
```

Every vector operand in the body names Z0–Z14. There is not one Z15 or higher in
the listing; the allocator makes 25 stack references per iteration rather than
touch a register it does not believe exists. A 20-chain kernel needs 20
registers, has 32 on the machine, is offered 15, and pays L1 round trips for the
rest.

### Property 2: there is no `231` or `132` FMA, so an accumulate needs a scratch register

[T2](#t2) recorded that `MulAdd` lowers to `VFMADD213PS`. The stronger statement
is that nothing else is available:

```
$ grep -n 'name: "VF.*PS512"' _gen/simdAMD64ops.go
VFMADD213PS512      VFMADDSUB213PS512      VFMSUBADD213PS512
```

plus their `load` and `Masked` variants. No `VFMADD231PS512`, no `132`.

`213` writes its result to its first multiplicand (`resultInArg0: true`). A GEMM
accumulate is `acc += a·b`, so `acc` is the *addend* — argument 2 — and the
result therefore cannot land in `acc`'s register. The compiler handles this
well: it lets the second FMA of a row consume the broadcast destructively, so
the accumulator migrates into the broadcast's old register and the measured cost
is one `VMOVDQU64` register copy per *broadcast*, not per FMA. But it needs one
live scratch register beyond the working set, always.

The load form does not help. The rule is

```
(VFMADD213PS512 x y l:(VMOVDQUload512 …)) => (VFMADD213PS512load … x y ptr mem)
```

which folds argument 2 — the addend, i.e. the accumulator. Folding the
accumulator from memory is the one thing a GEMM kernel must never do. Had a
`231` form existed, its memory operand would have been a multiplicand: the B
panel could have been read straight out of the FMA, and with AVX-512 embedded
broadcast the A operand too, at zero registers and zero extra instructions.

### The consequence: 8 accumulators, not 12

Put the two together. For an MR×(NB·16) tile with unroll u, the values live
across the unrolled body are

```
MR·NB  accumulators
NB     B-panel vectors
MR·u   A-panel scalars   ← the SSA scheduler hoists every step's loads to the top
1      FMA scratch (Property 2)
------
       ≤ 15 (Property 1)
```

`DESIGN.md` budgeted 12 accumulators + 2 B + 1 broadcast = 15 live and called it
exactly full. That budget is right on Property 1's number and wrong on Property
2's: 12 + 2 leaves one register, and one register has to be both the A operand
and the scratch. `MR·u` makes it worse, because Go's scheduler hoists all of an
unrolled body's independent loads to the top of the body, so unrolling *raises*
peak pressure instead of amortizing the copies — which is the opposite of why
`DESIGN.md` asked for the unroll.

Sweeping the shape space confirms it (115 generated kernels, audited with
`spill-audit`; `b` = scalar broadcast, `p` = one A vector load plus `VPERMPS`
lane broadcasts with the selector folded from memory):

| shape | u | insns | FMAs | ins/FMA | vec stack refs |
| --- | --- | --- | --- | --- | --- |
| 2×64 `p` | 2 | 71 | 16 | **4.44** | 0 |
| 2×32 `p` | 4 | 73 | 16 | 4.56 | 0 |
| 2×32 `b` | 4 | 74 | 16 | **4.62** | 0 |
| 3×32 `p` | 2 | 56 | 12 | 4.67 | 0 |
| 3×32 `b` | 2 | 60 | 12 | 5.00 | 0 |
| 4×32 `b` | 1 | 50 | 8 | 6.25 | 0 |
| 4×32 `b` | 2 | 83 | 16 | 5.19 | 8 |
| **6×32 `b`** | **1** | 76 | 12 | 6.33 | **12** ← DESIGN.md's tile, no unroll |
| **6×32 `b`** | **4** | 270 | 48 | 5.62 | **90** ← DESIGN.md's tile as specified |

Every configuration with 12 accumulators spills. The frontier is at 8, and it is
reached by shrinking M (2×64, 2×32) rather than N, because N is where the
vectors are and M is what costs a broadcast.

Where the 74 instructions of the best scalar-broadcast shape go, per iteration
of 16 FMAs:

| class | count | per FMA |
| --- | --- | --- |
| `VFMADD213PS` | 16 | 1.00 |
| integer loop overhead (two slice re-slices, counters, compare, branch) | 18 | 1.13 |
| `VMOVDQU64` B loads | 8 | 0.50 |
| `VMOVSS` A scalar loads | 8 | 0.50 |
| `VBROADCASTSS` | 8 | 0.50 |
| `VMOVDQU64` register copies (Property 2) | 8 | 0.50 |
| `XCHGL AX, AX` anchor NOPs ([T9](#t9)) | 8 | 0.50 |

Nothing here is a spill; the kernel is clean by P2's stated criterion. It is
still 4.6 instructions per unit of arithmetic, and on a machine whose front end
retires 4 per cycle and whose FMA units accept 2 per cycle, that is the binding
constraint rather than the arithmetic. The 18 integer instructions are the
largest single non-FMA block: `a, b = a[k:], b[j:]` compiles to
`ADDQ/MOVQ/NEGQ/SARQ $63/ANDL` per slice — a branchless "advance the pointer
only if the remainder is non-empty" clamp — and there are two panels.

### Assessment

Property 1 looks like a plain oversight and is the more valuable of the two
upstream: nothing in the design of `archsimd` wants X16–X31 withheld, the op
definitions already permit them, and the cost is that any kernel needing more
than 15 live vectors silently spills instead of using idle hardware. Property 2
is a missing-op report rather than a bug, but it is the one with the larger
effect on generated code: a `231` form would remove one instruction per
broadcast *and* let both GEMM operands come from memory.

Neither is worked around here. The tile is shrunk to what fits, the shrink is
recorded in `KERNEL.md`, and P2's percent-of-peak measurement is taken on the
kernel the compiler can actually allocate. What that measurement says about the
go/no-go is a separate question from this note, and is answered with numbers in
`KERNEL.md` rather than with the model above.

---

## T11 — `GOSSAFUNC` is not in the build cache key, and says otherwise on a cache hit

`GOSSAFUNC=Name go build` dumps the compiler's SSA for `Name` to `./ssa.html`.
The environment variable is read by the compiler, but it is not part of the
action ID the `go` command caches on — so the second identical build is a cache
hit, the compiler never runs, and no file is written. The dangerous half is that
the *cached compiler stderr is replayed*, so the build still prints the success
message.

```
$ mkdir /tmp/gossacache && cd /tmp/gossacache
$ printf 'module gossaprobe\n\ngo 1.26\n' > go.mod
$ cat > f.go <<'GO'
package p

func Add(a, b []float32) {
	for i := range a {
		a[i] += b[i]
	}
}
GO
$ export GOCACHE=/tmp/gossacache/cache      # a cache of our own, so run 1 is cold

$ GOSSAFUNC=Add go build -o /dev/null .
# gossaprobe
dumped SSA for Add,1 to ./ssa.html
$ ls -s ssa.html
344 ssa.html

$ rm ssa.html
$ GOSSAFUNC=Add go build -o /dev/null .     # identical build: cache hit
# gossaprobe
dumped SSA for Add,1 to ./ssa.html          # <-- replayed from the cache
$ ls ssa.html
ls: ssa.html: No such file or directory     # <-- and nothing was written
```

Run 3, with a *different* `GOSSAFUNC` value, printed nothing at all and also
wrote nothing: the action ID is the same either way, so the value cannot select
a different dump once the package is cached.

This is a `cmd/go` caching question rather than a simd one, and it would bite any
use of `GOSSAFUNC` in a script. It is filed here because it bit this project's
gate: `scripts/gate-p2.sh` requires an archived `ssa.html` per audited kernel so
that a red spill audit always ships with the allocator evidence behind it, and the
requirement passed on the first run of the session and failed on the second — the
audit's own earlier `-gcflags=-S` compile having warmed the cache in between.
Trusting the printed message would have produced a gate that reported evidence it
had not collected.

### Workaround

`internal/spill/cmd/spill-audit` gives each dump a private `GOCACHE` under its
scratch directory, which guarantees a cold compile because `GOCACHE` *is* part of
the lookup. Measured cost for `./internal/vec` cross-compiled to linux/amd64:
**0.63 s and 18 MB** per function, discarded afterwards — cheap enough that no
cleverer invalidation is worth the fragility. The tool then verifies the file
exists rather than believing the exit status, which is what turned this up.

`-a` would also work and rebuilds the standard library; a content change to the
package would not, since Go hashes source content rather than mtimes.

## T12 — the assembler encodes the FMA a GEMM wants; the intrinsic layer cannot reach it

This is the note behind P2's roofline amendment (DESIGN.md §4/P2) and the upstream
report in #20. It is three findings that look like one wall from inside a kernel.

A hand-written SGEMM K-loop accumulates with a single instruction per FMA:

```
vfmadd231ps zmm_acc, zmm_b, [a_ptr]{1to16}     // acc += b * broadcast(*a_ptr)
```

Everything about that is available in Go's assembler today. Verified by
assembling it, then decoding the emitted bytes with `llvm-mc` because Go's own
`objdump` does not decode EVEX:

```
$ cat a_amd64.s
#include "textflag.h"
TEXT ·try(SB), NOSPLIT, $0
        VFMADD231PS.BCST 12(SI), Z1, Z0
        RET
$ GOOS=linux GOARCH=amd64 go tool asm -I $(go env GOROOT)/pkg/include -p x -o s.o a_amd64.s
$ llvm-mc --disassemble --arch=x86-64 <<< '0x62 0xf2 0x75 0x58 0xb8 0x46 0x03'
        vfmadd231ps 12(%rsi){1to16}, %zmm1, %zmm0  ## zmm0 = (zmm1 * mem) + zmm0
```

EVEX.512, embedded broadcast, disp8 compression, accumulate-in-place, memory as a
*multiplicand*. The assembler is not the constraint.

The intrinsic path emits four instructions for that same work — `VMOVSS`,
`VBROADCASTSS`, a `VMOVDQU64` register copy, and `VFMADD213PS` — for three
independent reasons, each individually small:

**1. Only the 213 form has an SSA op.** `simdAMD64ops.go` defines
`VFMADD213PS{128,256,512}` and their masked variants, and no 231-shaped op for
any width. All are `resultInArg0: true`, so the destination is the first
multiplicand and `acc += a·b` can never land in `acc` (this is T10 property 2;
the register copy is its cost). `AMD64Ops.go` does define scalar `VFMADD231SS`
and `VFMADD231SD`, and `AMD64.rules` uses them for scalar `math.FMA` — so the 231
*shape* is already understood by the compiler, just not for vectors.

**2. The load-merge rule that exists folds into the wrong operand.** This is the
sharp one, because the machinery is not missing — it fires, into the operand a
GEMM cannot spare:

```
simdAMD64.rules:2774
(VFMADD213PS512 x y l:(VMOVDQUload512 {sym} [off] ptr mem)) && canMergeLoad(v, l) && clobber(l)
  => (VFMADD213PS512load {sym} [off] x y ptr mem)
```

`MulAddFloat32x16(x,y,z)` is `x*y+z`, so the merged third operand is the
**addend**. In the 213 encoding the addend is the only operand that may be
memory, and in a GEMM K-loop the addend is the accumulator — the one value that
must stay in a register for the whole loop. So the fold is unreachable in the
only loop that would benefit. Getting a *multiplicand* from memory requires the
231 form, where the destination is the addend and `src3/m512` is a multiplicand.
The 16 `VFMADD*load` rules present all have this property.

**3. Nothing emits `.BCST`.** `obj/x86/evex.go` supports the suffix
(`BroadcastEnabled`, and `"BCST"`/`"BCST.Z"` in the suffix table), and there is
no occurrence of `BCST` anywhere under `cmd/compile/internal/ssa/_gen/`. So a
broadcast from a slice element costs `VMOVSS` + `VBROADCASTSS` (T9 adds an anchor
NOP at the load call site) where the ISA allows zero instructions.

### Measured cost, on the shipped 2×32 tile

74 instructions per pass, 16 FMAs. Removing what the 231 form and the embedded
broadcast would remove — 8 register copies, 8 `VMOVSS`, 8 `VBROADCASTSS`, and the
anchor NOPs at the A-side load sites — leaves ~46, so `insns/FMA` goes
4.625 → ~2.875. On a host whose front end is the binding constraint that is the
difference between 46% and ~74% of measured peak (DESIGN.md §4/P2, the janus
roofline). The three causes are individually modest and jointly a factor of ~1.6
on 2-FMA/cycle hardware; none of them alone clears keel's own P2 floor.

### Why this is not "archsimd is missing an operation"

Every intrinsic keel needs exists and lowers to the right instruction. The gap is
entirely in *operand shape*: which operand may be memory, which may be broadcast,
and which is overwritten. A vector intrinsic API that lowers each call in
isolation has no way to express those choices, and the peephole layer that would
normally recover them (the `canMergeLoad` rules) is present but pointed at the
addend. That framing is the upstream report; #20 carries it.

---

## T13 — `import "C"` in a `_test.go` file is rejected, so a C reference benchmark needs a package file

**Observation.** P3's gate compares keel against OpenBLAS (DESIGN.md §4/P3,
">= 60% of OpenBLAS at 2048³"). The obvious shape for that is one build-tagged
test file holding both the cgo binding and the benchmark, so nothing keel ships
can link a BLAS. `go test` refuses to build it:

```
$ cat p_test.go
package p

/*
int seven(void) { return 7; }
*/
import "C"

import "testing"

func TestSeven(t *testing.T) {
	if got := int(C.seven()); got != 7 {
		t.Fatal(got)
	}
}

$ CGO_ENABLED=1 go test ./...
# cgoprobe
use of cgo in test /tmp/cgotest/p_test.go not supported
FAIL	cgoprobe [setup failed]
```

The rule is explicit in `cmd/go/internal/modindex/read.go:589`: any file whose
imports include `"C"` and whose name ends in `_test.go` is marked a bad Go file.
Note the failure mode — `[setup failed]`, not a compile error inside the file, so
in a script it can read as a missing library or a bad tag rather than a rejected
file layout.

**Not a simd note, and not new.** This restriction long predates
`GOEXPERIMENT=simd`; it is recorded here because it shaped a file layout in this
repo, and because the phrasing "not supported" invites the assumption that cgo in
tests works with some flag. It does not.

**Workaround (the layout `bench/` now uses).** The binding goes in a *package*
file under the same build tag, exporting plain-Go wrappers; the benchmark stays in
a `_test.go` file and calls them.

```
bench/openblas.go       //go:build openblas   — import "C", cblas_sgemm wrappers
bench/openblas_test.go  //go:build openblas   — BenchmarkOpenBLAS, no import "C"
```

One consequence worth stating, because it is the reverse of the usual direction:
a package file cannot see identifiers declared in `_test.go` files. So the tagged
package file cannot touch `openblasProvenance` (declared in `bench_test.go`); the
tagged *test* file's `init` sets it from the wrappers instead. Verified building
and running against Homebrew OpenBLAS 0.3.34 on the dev host — see docs/hosts.md
for why a number from that host is not the P3 criterion.
