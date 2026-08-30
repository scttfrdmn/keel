# Toolchain field notes — GOEXPERIMENT=simd

A first-class deliverable (DESIGN.md §7 rule 8): every surprise from the
experimental simd packages — missing intrinsics, unexpected lowerings,
allocator behavior, API changes between releases — gets an entry here with
a minimal repro *before* any workaround lands. Entries feed upstream issues.

**Shape of a new entry, capped 2026-08-16.** The repro is the deliverable and is
never abridged; the prose around it is. A new entry is one table row, an
**Observation** of at most three lines, the repro verbatim, and at most three
lines of **Consequence for keel** — what changed in the tree, not why. Extended
causal analysis, rejected hypotheses and the forensics that got there belong in
the issue this entry cites, which is where they can be argued with. §7 rule 8
asks for a repro before a workaround and sets no length, so this narrows nothing
it requires. Existing entries are left as written: they are dated records, and
rewriting them would cost real reasoning to buy back lines already spent.

**Every entry here is dated by its toolchain, and that is a law about claims and not
a filing convention** (added 2026-08-30, ruling on T28). *The toolchain moved under a
true finding.* T10/#18 measured that 15 of 32 vector registers were allocatable and
was right; go1.27.0 made it false with nobody touching keel, and three sites in the
tree had meanwhile promoted it from a dated reading to a settled cause. So a codegen
claim carries the toolchain it was measured on **inside** the claim, a superseded
entry is marked at its head rather than rewritten — the observation stood, the era
ended — and the era-boundary law applies to compiler findings exactly as it applies
to a measured rate: what dates a rate is the fleet and the denominator, what dates a
lowering is `go version`. The corollary is a re-measurement obligation: a toolchain
bump re-opens every entry it could touch, and citing one across a bump without
re-running it is citing a quotation as a measurement.

| Date | Toolchain | Observation | Repro | Upstream issue |
|---|---|---|---|---|
| 2026-08-10 | go1.26.5 | `simd/archsimd` is amd64-only; no vector type exists on other GOARCHes | [T1](#t1) | none — documented upstream |
| 2026-08-10 | go1.26.5 | `MulAdd` lowers to `VFMADD213PS`, not the `231` form DESIGN.md predicted | [T2](#t2) | none — not a defect |
| 2026-08-10 | go1.26.5 | No `Abs` and no bitwise ops on any float32 vector type | [T3](#t3) | candidate |
| 2026-08-10 | go1.26.5 | No horizontal-reduce op for `Float32x16` | [T4](#t4) | candidate |
| 2026-08-10 | go1.26.5 | No portable `simd` package; only `simd/archsimd` exists | [T5](#t5) | none — **shipped in 1.27**, see [T24](#t24) |
| 2026-08-10 | go1.26.5 | `Max`/`Min` operand order for NaN/±0 — **resolved on hardware**, spec was right | [T6](#t6) | closed (#9) |
| 2026-08-10 | go1.26.5 | `GOAMD64` level does not gate archsimd intrinsics; a v1 binary runs AVX-512 | [T7](#t7) | none — load-bearing for keel |
| 2026-08-10 | go1.26.5 | CSE merges identical FMA accumulator chains; two more ways a peak kernel lies | [T8](#t8) | none — not a defect |
| 2026-08-11 | go1.26.5 | Every inlined non-intrinsic call costs a 1-byte NOP in the loop body | [T9](#t9) | filed: [golang/go#80830](https://github.com/golang/go/issues/80830) (#17) |
| 2026-08-11 | go1.26.5 | Only 15 of 32 vector registers are allocatable; no `231` FMA form exists | [T10](#t10) | filed: [golang/go#80828](https://github.com/golang/go/issues/80828), [golang/go#80829](https://github.com/golang/go/issues/80829) (#18) |
| 2026-08-11 | go1.26.5 | `GOSSAFUNC` is not in the build cache key: on a cache hit it writes no `ssa.html` but still prints `dumped SSA … to ./ssa.html` | [T11](#t11) | candidate |
| 2026-08-11 | go1.26.5 | The assembler encodes `vfmadd231ps mem{1to16}` and the intrinsic layer cannot reach it: no 231 SSA op, the load-merge rule folds memory into the addend, and nothing emits `.BCST` | [T12](#t12) | filed: [golang/go#80829](https://github.com/golang/go/issues/80829) (#20) |
| 2026-08-11 | go1.26.5 | `import "C"` in a `_test.go` file is rejected outright — a benchmark cannot call C directly, so a reference harness needs a package file | [T13](#t13) | none — long-standing, not simd |
| 2026-08-11 | go1.26.5 | `archsimd` exposes CPU *features* and nothing else: no vendor, family, model or name, and `internal/cpu` discards the signature word. Per-µarch kernel selection has to fingerprint a feature bundle | [T14](#t14) | candidate (#25) |
| 2026-08-11 | go1.26.5 | `-bench` splits on top-level `\|` **before** `/`, so `A\|B/c` is `{A}` or `{B,c}` — not `{A,B}` then `{c}`. A gate filter silently ran neither the benchmark it named nor only the ones it named | [T15](#t15) | none — documented behavior |
| 2026-08-12 | go1.26.5 | On arm64, whether `a*a + c` is FMA-fused depends on whether the compiler **constant-folds** it, and `-race` defeats the folding: the same source line yields `0` in a plain build and `2^-24` under `-race` | [T16](#t16) | none — not a defect |
| 2026-08-12 | go1.26.5 | `archsimd`'s partial slice load/store convert `&s[0]` to a full-width `*[N]T` through an `unsafe` helper, so **`checkptr` fatals** — any partial op on a short slice near the end of its heap object aborts under `-race` or `-d=checkptr`. Not a warning: `fatal error` | [T17](#t17) | **fixed upstream in go1.27** ([CL 761120](https://go-review.googlesource.com/c/go/+/761120) marks all 30 `pa*` helpers `nocheckptr`; 0 in go1.26.6, 30 in `go1.27rc1`), so every 1.26.x reproduces it. Filed: [golang/go#80856](https://github.com/golang/go/issues/80856), closed as a duplicate of [#78413](https://github.com/golang/go/issues/78413). keel's copy-based workaround is a **1.26.x bridge** (#42, #22) |
| 2026-08-12 | go1.26.5 | A **loop-invariant vector constant is re-materialized every iteration**: `BroadcastInt32x16(const)` inside a loop stays inside it (3 insns/iteration), while the same constant written above the loop stays above it. LICM does not lift SIMD ops at all; no helper, inlining or CSE is needed to trigger it. CSE shares it across unrolled uses within one iteration | [T18](#t18) | **pre-existing upstream**: [golang/go#79984](https://github.com/golang/go/issues/79984), fix WIP in [CL 803220](https://go-review.googlesource.com/c/go/+/803220) (#8) |
| 2026-08-12 | go1.26.5 | `s = s[n:]` in a loop guarded by `len(s) >= n` costs **7 instructions**, not 2: the pointer bump is made conditional (`NEGQ`/`SARQ $63`/`ANDL`) because the loop may leave the slice exactly empty and the data pointer must not pass the end of the allocation. Guarding with `len(s) > n` collapses it to one `ADDQ` | [T19](#t19) | none — GC-correctness, not a defect; the `>` form is **taken** as of 2026-08-14, all ten `internal/l1` loops |
| 2026-08-12 | benchstat v0.0.0-20260709024250 | `benchstat A B` **does not compare A and B** when their logs differ in any one `key: value` configuration line — it prints two independent one-column tables, no deltas, no p-values, and **exits 0**. This repo's own provenance markers include a live clock snapshot, so two runs on one host never compare | [T20](#t20) | candidate |
| 2026-08-12 | benchstat v0.0.0-20260709024250 | benchstat's CI is an **integer percent** in both the text table and the CSV, so `± 0%` means "narrower than 0.5%", not "zero" — a consumer that does arithmetic with the interval inherits the formatting's precision, not the quantity's | [T21](#t21) | none — by design |
| 2026-08-14 | go1.26.6 | A commit that changes **no instruction byte** of a function can still make it **45% slower**; function-entry alignment mod 64 is the discriminator, and the magnitude is a property of the µarch (~0% Skylake-X, ~7% Zen 4, ~45% Zen 5) | [T22](#t22) | none — long-standing ([golang/go#8717](https://github.com/golang/go/issues/8717), [#18977](https://github.com/golang/go/issues/18977), [#6752](https://github.com/golang/go/issues/6752)) |
| 2026-08-15 | go1.27rc3 | `archsimd`'s load/store are **renamed with a swap**: the slice forms take over the bare names (`LoadFloat32x16Slice`→`LoadFloat32x16`, `StoreSlice`→`Store`) while the array forms gain an `Array` suffix, and the `…SlicePart` forms become `…Part` *and grow a return value*. keel does not compile under 1.27: 51 errors in 3 files, all of them type errors | [T23](#t23) | none — pre-GA API churn, expected (T5) |
| 2026-08-15 | go1.27rc3 | The portable `simd` package **ships** (T5's guess was right), with arm64, wasm and a pure-Go emulated fallback — but its vector length is a **runtime** quantity (`VectorBitSize()`, `Len()`), so it cannot express a register-blocked microkernel's compile-time tile, and has no `GetLo`/`GetHi` for `HSum`'s fold tree | [T24](#t24) | none — as designed |
| 2026-08-15 | go1.26.6, go1.26.5, go1.27rc3 | Four ways the *spelling* of an equivalent SIMD loop changes its object code, worth **36 vs 13 instructions** per iteration in keel's fringe add-back: `Load512(x[j:])` keeps `archsimd`'s own `CMPQ $16` **and** T19's conditional pointer advance where `Load512(x[j:j+Lanes])` folds both; the natural guard `j+16 <= len(x)` keeps a bounds check where the identical `j < len(x)-15` does not (a **second remedy [T19](#t19) missed**, and one that keeps the loop indexed); `dst = dst[:len(src)]` *moves* the surviving check onto the resliced operand rather than removing it; and hoisting one invariant limit is free while hoisting both into a `min` puts both checks back | [T25](#t25) | none — known class ([#17370](https://github.com/golang/go/issues/17370), [#25197](https://github.com/golang/go/issues/25197), [#28941](https://github.com/golang/go/issues/28941), fix in flight [#80146](https://github.com/golang/go/issues/80146)); keel #74 |
| 2026-08-20 | go1.27.0 | Benchmark output precision is chosen by a value's **magnitude**, not by the measurement's: `testing.prettyPrint` gives four significant figures under `999.95` and every integer digit at or above it, for `ns/op` and `ReportMetric` columns alike. So one run's 245.1 GFLOP/s (0.1 quantum, 0.041%) and its own reciprocal 102762 ns/op (1 ns, 0.00097%) sit **42× apart in resolution**, and a check reading the coarse column decides below its own quantum | [T26](#t26) | none — accepted design, raised from three figures to four by [#34626](https://github.com/golang/go/issues/34626) / [CL 267102](https://go-review.googlesource.com/c/go/+/267102), whose thread anticipates this exact 0.05 step |

All repros below were run on `go1.26.5 darwin/arm64` with Homebrew's Go.
Where a repro needs amd64 it cross-compiles, which is enough for anything the
compiler decides but not for anything the CPU decides — see T1. Where a repro
needed *execution*, it ran on the amd64 hosts in docs/hosts.md, and says which.

---

<a name="t1" id="t1"></a>
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

<a name="t2" id="t2"></a>
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

<a name="t3" id="t3"></a>
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

<a name="t4" id="t4"></a>
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

<a name="t5" id="t5"></a>
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

**Resolved (2026-08-15).** It ships in `go1.27rc3`, and DESIGN.md §8's parking of
the ARM64 kernel behind it needs revisiting rather than resuming: the package is
vector-length-agnostic at runtime, which is the one property a register-blocked
microkernel cannot use. See [T24](#t24).

<a name="t6" id="t6"></a>
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

<a name="t7" id="t7"></a>
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

<a name="t8" id="t8"></a>
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

<a name="t9" id="t9"></a>
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

<a name="t10" id="t10"></a>
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

<a name="t11" id="t11"></a>
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

<a name="t12" id="t12"></a>
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

<a name="t13" id="t13"></a>
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

---

<a name="t14" id="t14"></a>
## T14 — archsimd reports features, never a microarchitecture, so per-µarch dispatch must fingerprint

**Observation.** P3 needs to pick a microkernel *shape* per host: KERNEL.md §7
measures the instruction-lean 2×32 winning on Skylake-X and the load-lean 4×32
winning on Zen 4 and Zen 5, so a single shape is wrong on at least one machine in
this fleet (issue #24). Every production BLAS makes that choice from the CPU's
identity — BLIS and OpenBLAS both dispatch on vendor plus family/model. That
identity is not reachable from a pure-Go `GOEXPERIMENT=simd` program.

`archsimd`'s entire CPU-introspection surface is twenty feature predicates:

```
$ GOEXPERIMENT=simd go doc simd/archsimd.X86Features
type X86Features struct{}
    var X86 X86Features
func (X86Features) AVX() bool
func (X86Features) AVX2() bool
func (X86Features) AVX512() bool
func (X86Features) AVX512BITALG() bool
func (X86Features) AVX512GFNI() bool
func (X86Features) AVX512VAES() bool
func (X86Features) AVX512VBMI() bool
func (X86Features) AVX512VBMI2() bool
func (X86Features) AVX512VNNI() bool
func (X86Features) AVX512VPCLMULQDQ() bool
func (X86Features) AVX512VPOPCNTDQ() bool
func (X86Features) AVXAES() bool
func (X86Features) AVXVNNI() bool
func (X86Features) FMA() bool
func (X86Features) SHA() bool
func (X86Features) VAES() bool
```

No vendor, no family, no model, no brand string. Nor is the data available one
layer down and merely unexported by `archsimd`: `internal/cpu` calls
`cpuid(1, 0)` and keeps only `ecx`, dropping `eax` — the processor signature word
holding stepping, model and family — on the floor.

```
$ R=$(go env GOROOT)
$ grep -n 'cpuid(1, 0)' $R/src/internal/cpu/cpu_x86.go
124:	_, _, ecx1, _ := cpuid(1, 0)
$ grep -c 'Family\|Model\|Stepping' $R/src/internal/cpu/cpu_x86.go
0
```

`internal/cpu.Name()` does read the brand string (`cpu_x86.go:236`, CPUID leaves
`0x80000002`–`0x80000004`), but `internal/cpu` is not importable and `archsimd`
re-exports none of it. So a pure-Go program on this toolchain can ask *what
instructions* a CPU has and never *which CPU* it is.

**Consequence for keel, stated as a proxy rather than hidden as one.** Dispatch
classifies the front end by the feature bundle that arrived with Ice Lake and Zen 4
— `AVX512VBMI2` and `AVX512VPOPCNTDQ` — and treats an AVX-512 machine *without*
that bundle as the Skylake-X/Cascade Lake/Cooper Lake generation, which is the
generation P2 measured as issue-bound. See `internal/kern/class_amd64.go` for the
rule and the direction each way it can be wrong; the short version is that a wrong
answer costs throughput or a red gate, never a lenient one, because the gate's
own classification is measured rather than fingerprinted and the two are compared.

**Alternatives considered.** `/proc/cpuinfo` carries `vendor_id`, `cpu family` and
`model` and would give the true identity — on Linux only, so the same silicon
would classify differently under a different OS, and dispatch would do file I/O at
init. A CPUID stub in assembly would be exact and portable across OSes, but keel
is a pure-Go experiment and P2's ruling was explicit that assembly is not a
unilateral move. Both are recorded in #25 as the shapes an upstream fix would make
unnecessary: one exported accessor for the signature word, or a documented
`archsimd` µarch enum, removes the fingerprint entirely.

---

<a name="t15" id="t15"></a>
## T15 — `-bench` splits on top-level `|` before `/`, so an alternation of paths is not what it reads as

**Observation.** `go test -bench` patterns are not one regexp matched against the
full benchmark name. `testing.splitRegexp` splits the pattern on **top-level `|`
first**, producing an *alternation of whole patterns*, and only then splits each
alternative on `/` into per-depth elements. So

```
Peak|Sgemm|OpenBLAS/avx512|n=2048
```

is four independent alternatives — `{Peak}`, `{Sgemm}`, `{OpenBLAS, avx512}`,
`{n=2048}` — and not the two-level filter `{Peak,Sgemm,OpenBLAS}` then
`{avx512,n=2048}` that it reads as. Two things follow, in opposite directions:

- an alternative with **fewer** elements than the name is depth-unconstrained, so
  `Peak` runs every `Peak/*` sub-benchmark;
- an alternative with **more** elements than any real name matches *nothing*, but
  matches the parent **partially** — `simpleMatch.matches` returns
  `ok=true, partial=true` when `len(name) < len(m)` — so the parent benchmark is
  entered and every child is then rejected.

The second is the dangerous one: the parent runs its `init`-time output and no
sub-benchmark result appears, which looks exactly like a benchmark that was
present but produced nothing.

**Repro.** Any package with two-level sub-benchmarks; here keel's `bench/`, whose
top-level names are `Peak` (children `avx512`, `avx2`, `scalar`), `Sgemm` and
`OpenBLAS` (children `n=256…2048`). Run on janus (linux/amd64); nothing here is
architecture-dependent.

```
$ B="./bench-ob.test -test.run=NONE -test.count=1 -test.benchtime=1x"

$ $B -test.bench='Peak|Sgemm|OpenBLAS/avx512|n=2048' | grep -c '^BenchmarkOpenBLAS'
0                                    # the OpenBLAS alternative can never match
$ $B -test.bench='Peak|Sgemm|OpenBLAS/avx512|n=2048' | grep -c '^BenchmarkSgemm'
4                                    # ...and the Sgemm alternative is unconstrained

$ $B -test.bench='Peak|Sgemm|OpenBLAS/n=2048' | grep -c '^BenchmarkOpenBLAS'
1                                    # moving the '/' into the right alternative fixes that one
$ $B -test.bench='Peak|Sgemm|OpenBLAS/n=2048' | grep -c '^BenchmarkSgemm'
4                                    # but not the other

$ $B -test.bench='(Peak|Sgemm|OpenBLAS)/(avx512|n=2048)' | grep '^Benchmark' | cut -d' ' -f1
BenchmarkSgemm/n=2048-32
BenchmarkOpenBLAS/n=2048-32
BenchmarkPeak/avx512-32              # exactly three, which is what was meant
```

The mechanism is `$GOROOT/src/testing/match.go`: `splitRegexp` tracks `[`/`]` and
`(`/`)` nesting and splits `|` and `/` only at depth zero, so parentheses suppress
both splits and turn the `|`s back into ordinary regexp alternation inside a
two-element pattern.

**Not a defect** — it is what the code says and what `go help testflag`'s "the
regular expression is split by unbracketed slash characters" implies if read
closely. Recorded because of the failure mode rather than the behavior: a shell
variable holding a pattern like the one above reads as a two-level filter to
everyone who reviews it, runs without error, and produces a benchmark log that is
missing a row nobody asked twice about. In keel it meant the P3 gate's headline
criterion — keel vs same-host OpenBLAS — never ran its denominator's benchmark,
which stayed invisible for as long as the hosts had no OpenBLAS to fail on first
(issue #32). It also cost every gate run three unread `Sgemm` sizes and two unread
`Peak` variants at `-count=10 -benchtime=1s` per host.

**Consequence for keel.** Gate filters are parenthesized, and `scripts/gate-p3.sh`
now fails a host with a message that distinguishes "no reference on this host" from
"the run produced no such row", because those two states had the same wording and
only the second is a bug in the gate.

<a name="t16" id="t16"></a>
## T16 — on arm64, whether `a*a + c` fuses depends on constant folding, and `-race` changes the answer

**Observation.** The Go spec permits an implementation to fuse a floating-point
multiply and add, and gc does so on arm64. What is not obvious is that gc's decision
is not a property of the *expression* — it is a property of whether the expression
survives to code generation at all. When the operands are compile-time constants the
product is folded at compile time, with the spec's per-operation rounding, and the
result is the **unfused** value. Add anything that defeats constant propagation —
`-race` instrumentation, or a `//go:noinline` call — and the same source line is
evaluated at run time by a single `FMADD`, giving the **fused** value.

So one line yields two different float32 results in the same toolchain on the same
machine, depending on a flag that is not about arithmetic.

**Repro.** `go1.26.5 darwin/arm64`, no build tags, no experiment needed. The witness is
keel's own: `a = 1+2^-12`, so `a*a = 1 + 2^-11 + 2^-24` exactly, which needs a 24th
fractional bit float32 does not have and sits exactly halfway between neighbours.
Unfused it rounds to `1+2^-11` (ties-to-even) and `+c` gives exactly `0`; fused it
returns `2^-24 = 5.9604645e-08`.

```go
package fmarepro

import (
	"math"
	"testing"
)

//go:noinline
func opaque(f float32) float32 { return f }

func TestFuse(t *testing.T) {
	a := float32(1) + math.Float32frombits(0x39800000)  // 1 + 2^-12
	c := -(float32(1) + math.Float32frombits(0x3A000000)) // -(1 + 2^-11)

	t.Logf("constants:     a*a + c          = %v", a*a+c)
	t.Logf("constants:     float32(a*a) + c = %v", float32(a*a)+c)

	b, d := opaque(a), opaque(c)
	t.Logf("through call:  b*b + d          = %v", b*b+d)
	t.Logf("through call:  float32(b*b) + d = %v", float32(b*b)+d)
}
```

```
$ go test -count=1 -v -run TestFuse .
    constants:     a*a + c          = 0                 # folded at compile time: unfused
    constants:     float32(a*a) + c = 0
    through call:  b*b + d          = 5.9604645e-08     # run-time FMADD: fused
    through call:  float32(b*b) + d = 0

$ go test -race -count=1 -v -run TestFuse .
    constants:     a*a + c          = 5.9604645e-08     # -race defeats the folding: fused
    constants:     float32(a*a) + c = 0
    through call:  b*b + d          = 5.9604645e-08
    through call:  float32(b*b) + d = 0
```

Three readings, one flag apart. `float32(a*a) + c` is `0` in every configuration: an
explicit conversion of an operand to its own type is the spec's documented way to
forbid fusion, and it is the only form here that means the same thing twice.

**Not a defect.** Both results are spec-compliant — the language explicitly allows
fusing and explicitly allows constant expressions to be evaluated with the rounding of
each operation. It is recorded because of what it does to *tests*: an assertion whose
premise is "this expression is not fused" is testing the optimizer's mood, and it will
hold for a year and then fail on a flag that has nothing to do with floating point.
amd64 hides it completely — gc does not contract `x*y+z` there, so an amd64-only CI
would never see either reading change.

**Consequence for keel.** `internal/vec.TestSpecMulAddIsFused` proves `ScalarMulAdd`
rounds once, and it does that by comparing against an unfused witness computed as
`a*a + c`. It carries a guard — "if the compiler or the platform changed such that the
unfused route no longer differs, this test would pass vacuously and stop protecting
anything" — and that guard is what found this: the test **failed loudly** under
`gate-p5.sh`'s `-race` criterion on darwin/arm64 rather than quietly comparing the
fused answer against itself. It had been passing everywhere else for a reason that was
not the one anybody wrote down: on amd64 because gc does not fuse, and on arm64 without
`-race` because the witness was folded before codegen. The witness now writes the
rounding it wants (`float32(a*a) + c`) instead of hoping for it, which is the same
correction T2 made to a different assumption: say what you require, do not infer it
from what the compiler happened to emit. See issue #41.

---

<a name="t17" id="t17"></a>
## T17 — `archsimd`'s partial slice load/store are not `checkptr`-safe, so `-race` is a fatal error

**Observation.** `archsimd`'s `LoadFloat32x16SlicePart` / `StoreSlicePart` (and the same
pair at every other width) handle a short slice by building a full-width masked
operation over `&s[0]`. To get an address of the right static type they go through an
`unsafe` helper that reinterprets the slice's first element as a pointer to a
**full-width array**, whatever the slice's actual length:

```go
// $(go env GOROOT)/src/simd/archsimd/unsafe_helpers.go:205
// paFloat32x16 returns a type-unsafe pointer to array that can
// only be used with partial load/store operations that only
// access the known-safe portions of the array.
func paFloat32x16(s []float32) *[16]float32 {
	return (*[16]float32)(unsafe.Pointer(&s[0]))
}
```

The mask does keep the *hardware* from touching anything past `len(s)` — the comment's
claim is true of the instruction. But the **conversion itself** is what `checkptr`
instruments, and `checkptr` has no way to know a mask will narrow the access later. It
sees a 64-byte type placed at an address with fewer than 64 bytes left in its heap
object and throws:

```
fatal error: checkptr: converted pointer straddles multiple allocations
```

Three properties make this worse than a normal instrumentation gripe:

1. **It is fatal, not reported.** A data race prints `WARNING: DATA RACE` and the test
   keeps going with a non-zero exit. This aborts the process at the first occurrence, so
   it also destroys any race-detector run it happens to precede.
2. **`-race` is not the trigger; `checkptr` is.** `-race` merely implies
   `-d=checkptr`. `-gcflags=all=-d=checkptr` alone reproduces it identically, so this
   is not the race detector's fault and cannot be dodged by tweaking race options.
3. **It is data-dependent, not shape-dependent.** Whether it fires depends on how much
   room the *allocation* has past `&s[0]`, not on the slice's length or capacity. The
   same call site can be quiet for a whole test suite and abort when an allocator layout
   changes. My first attempt at this repro passed for exactly that reason.

**Repro.** `go1.26.5 linux/amd64`, `GOEXPERIMENT=simd`, no keel code involved. The
buffer is exactly one vector wide and forced to the heap, and the slices are taken from
its tail so that the 64-byte window provably runs past the object's end.

```go
package ckptr

import (
	"simd/archsimd"
	"testing"
)

var sink []float32

func TestLoadSlicePartNearObjectEnd(t *testing.T) {
	buf := make([]float32, 16) // exactly one vector wide: 64 bytes
	sink = buf                 // keep it on the heap
	for _, n := range []int{1, 2, 4, 8, 15} {
		s := buf[16-n:] // len n, and &s[0]+64 runs past the end of buf
		v := archsimd.LoadFloat32x16SlicePart(s)
		var out [16]float32
		v.StoreSlice(out[:])
		t.Logf("len=%d lane0=%v", n, out[0])
	}
}
```

Uninstrumented, every length is fine — the masking works exactly as documented:

```
$ GOEXPERIMENT=simd go test -count=1 -v ./...
=== RUN   TestLoadSlicePartNearObjectEnd
    ckptr_test.go:18: len=1 lane0=0
    ckptr_test.go:18: len=2 lane0=0
    ckptr_test.go:18: len=4 lane0=0
    ckptr_test.go:18: len=8 lane0=0
    ckptr_test.go:18: len=15 lane0=0
--- PASS: TestLoadSlicePartNearObjectEnd (0.00s)
ok  	ckptr	0.001s
```

Under `-race`, the first iteration kills the process:

```
$ GOEXPERIMENT=simd CGO_ENABLED=1 go test -count=1 -race -v ./...
=== RUN   TestLoadSlicePartNearObjectEnd
fatal error: checkptr: converted pointer straddles multiple allocations

goroutine 20 gp=0xc000104b40 m=0 mp=0x7d03c0 [running]:
runtime.throw({0x652aca?, 0x566a01?})
	/usr/local/go/src/runtime/panic.go:1229 +0x48 fp=0xc0000cbd80 sp=0xc0000cbd50 pc=0x4beba8
runtime.checkptrAlignment(0x5668e0?, 0x75f460?, 0x7a54c0?)
	/usr/local/go/src/runtime/checkptr.go:26 +0x5b fp=0xc0000cbda0 sp=0xc0000cbd80 pc=0x44fafb
simd/archsimd.paFloat32x16(...)
	/usr/local/go/src/simd/archsimd/unsafe_helpers.go:209
simd/archsimd.LoadFloat32x16SlicePart(...)
	/usr/local/go/src/simd/archsimd/slice_gen_amd64.go:578
ckptr.TestLoadSlicePartNearObjectEnd(0xc000160248)
	/tmp/ckptr3/ckptr_test.go:15 +0x312 fp=0xc0000cbee0 sp=0xc0000cbda0 pc=0x5e36f2
```

And with `checkptr` alone, no race instrumentation, no cgo — the same fatal error at the
same two frames, which is what isolates the cause:

```
$ GOEXPERIMENT=simd go test -count=1 -gcflags=all=-d=checkptr -v ./...
=== RUN   TestLoadSlicePartNearObjectEnd
fatal error: checkptr: converted pointer straddles multiple allocations
...
runtime.checkptrAlignment(0x24?, 0x755?, 0x754?)
	/usr/local/go/src/runtime/checkptr.go:26 +0x5b
simd/archsimd.paFloat32x16(...)
	/usr/local/go/src/simd/archsimd/unsafe_helpers.go:209
simd/archsimd.LoadFloat32x16SlicePart(...)
	/usr/local/go/src/simd/archsimd/slice_gen_amd64.go:578
```

**Assessment.** This is an upstream defect, not a keel bug and not a false positive to
be waved away. `checkptr`'s complaint is literally correct about the conversion it is
shown; the `unsafe` helper's own comment concedes the pointer is "type-unsafe" and
constrains its use to operations that "only access the known-safe portions". What is
missing is any way to *say* that to the compiler. The consequence is that a supported,
exported, non-`unsafe` API — the documented way to handle a remainder — is unusable
under the standard Go debugging flag. Upstream has options keel does not (an intrinsic
that takes a `*T` and a length, a `checkptr` exemption, or generating the masked op
without a full-width array type); the right fix is theirs.

**Both widths, both directions.** `paFloat32x8` behaves identically
(`unsafe_helpers.go:139`, reached from `slice_gen_amd64.go:962`), and the store side goes
through the same helper (`StoreSlicePart` → `paFloat32x16` at `slice_gen_amd64.go:594`).
So this is the whole partial-memory family, not one function.

**Consequence for keel.** Four call sites, all of them the remainder handling that every
Level-1 routine reaches on any length that is not a multiple of the vector width:

```
internal/vec/vec_avx2.go:28    LoadPart256  → archsimd.LoadFloat32x8SlicePart
internal/vec/vec_avx2.go:34    StorePart256 → x.StoreSlicePart
internal/vec/vec_avx512.go:40  LoadPart512  → archsimd.LoadFloat32x16SlicePart
internal/vec/vec_avx512.go:46  StorePart512 → x.StoreSlicePart
```

Two things follow, and the second is the one that matters for v0.1.0:

- `gate-p5.sh`'s "race detector clean" criterion is **unmeetable on amd64** as long as
  keel calls these. It is worth noting *how* this surfaced: the gate's native `-race`
  run failed with no `WARNING: DATA RACE` in the log, and the script's three-way verdict
  called that **unmeasured** rather than clean — "a test that fails under instrumentation
  says nothing either way about whether keel has a race". A two-way pass/fail check would
  have recorded a red for a race that does not exist, or worse, been written to accept a
  non-zero exit with no warning as a pass.
- **Any keel user who runs `go test -race` on their own code gets a fatal error**, from
  inside a library they did not write, on any vector whose length is not a multiple of 16.
  That is not an internal testing inconvenience; it is a shipping defect.

A local workaround exists and is confirmed `checkptr`-clean: copy the remainder into a
full-width stack array and use the *full-width* `LoadFloat32x16Slice` / `StoreSlice`,
which take a `[]T` and never convert a pointer. In the same instrumented run that fataled
on `LoadFloat32x8SlicePart`, the copy-based form on the line above it completed. The cost
is a 64-byte zero-and-copy on the tail iteration only, never in the steady-state loop or
the K-loop. **Disposition (ruled 2026-08-12).** keel takes the workaround, and the P5 `-race`
criterion **stands unamended**: race-clean is table stakes for a Go library, and
`checkptr`-clean is what race-clean means for code holding `unsafe`, so excluding
`checkptr` from the criterion would certify keel safe minus the instrument that
checks. The fix lands inside issue #22's edge-handling campaign rather than as a
point patch, because #22 and #42 are the same question from two directions — how
keel touches memory at a vector or tile edge — and **`checkptr`-cleanliness is an
admissibility condition on #22's candidates, not a competitor in the measurement**:
a faster variant that fatals under the pointer checker is disqualified, not ranked.
Filed upstream as [golang/go#80856](https://github.com/golang/go/issues/80856) with
this repro; #42 carries the standing task, so when upstream's helpers go
`checkptr`-clean, keel's copy-based form retires rather than calcifying.

**Resolution (2026-08-15): fixed upstream in go1.27, and the retirement condition is met.**
CL 761120 — *`simd/archsimd: mark pa* unsafe helpers as nocheckptr`* — marks all 30 `pa*`
helpers, `paFloat32x16` among them. Read at the tag rather than from a changelog:
`go:nocheckptr` appears 30 times in `src/simd/archsimd/unsafe_helpers.go` on `master`,
`release-branch.go1.27` and **`go1.27rc1`**, and 0 times in go1.26.6. The CL merged
2026-03-31, after the 1.26 branch point, so **every 1.26.x reproduces this and 1.27 does
not**. Upstream closed #80856 as a duplicate of #78413; the closure was right about the bug
and silent about which releases carry the fix, which is the part that decided keel's
sequencing.

The fix's sufficiency was not obvious, and the doubt is worth recording because it was
wrong. [golang/go#42880](https://github.com/golang/go/issues/42880) —
*`cmd/compile: -race does not obey go:nocheckptr`*, open since 2020, `NeedsInvestigation` —
would have meant CL 761120 silences `-d=checkptr` and not `-race`, i.e. no fix at all for
the four `-race` criteria this note blocks. **Reading the thread rather than the title
settles it the other way.** mdempsky, same day it was filed: the reporter's failing
conversion was inside a *function literal*, and "`//go:` directives only apply to declared
functions". `-race` is incidental to that issue, and its prescribed workaround is CL 761120's
exact shape. The title has misdescribed its own maintainer diagnosis for six years.

Measured rather than argued, on go1.26.6 darwin/arm64 — a declared cross-package helper
carrying this note's exact conversion, against a no-pragma control, on a heap-allocated
`make([]float32, 4)`:

| helper | no flag | `-race` | `-gcflags=all=-d=checkptr` |
|---|---|---|---|
| no pragma (control) | survives | **fatal error: … straddles multiple allocations** | **fatal error: … straddles** |
| `//go:nocheckptr` | survives | **survives** | **survives** |

The control fires with this note's message verbatim, so the reproducer can fail; the pragma
suppresses it under `-race`. Property 3 above bit again while building it: the first version
had all six cells survive because `s := make([]float32, 4)` did not escape, and
`checkptrStraddles` compares `checkptrBase(ptr)` against `checkptrBase(end)`
(`runtime/checkptr.go:47`), which for a stack pointer is one base at both ends. **A control
arm that passes is a broken reproducer, not a result.**

The mechanism, in the compiler and then in the object code. Two checks, both reading
`base.Debug.Checkptr` and neither reading `Flag.Race` separately, which is why there is no
`-race`-shaped hole:

- `base/flag.go:351-356` — "`-race`, `-msan` and `-asan` imply `-d=checkptr` for now".
- `ir/expr.go:1113` — `ShouldCheckPtr` is `base.Debug.Checkptr >= level && fn.Pragma&NoCheckPtr == 0`, so the marked callee is not instrumented.
- `inline/inl.go:349-351` — under any `Checkptr != 0` build a `go:nocheckptr` function **is
  not inlined**. This is the load-bearing half: instrumentation is decided on the *enclosing*
  function (`ssagen/ssa.go:338`), so an inlined helper would be governed by `avx512Dot`'s
  absence of a pragma. Blocking the inline is what keeps the conversion somewhere clean.

Confirmed in the object code — `CALL`s to the helper in the caller: 0 uninstrumented (inlined),
1 under `-race`, 1 under `-d=checkptr`. So the pragma costs one real call per partial op **in
instrumented builds only**; release builds still inline it.

**Consequences for keel.** The copy-into-a-full-width-array workaround above is a **1.26.x
bridge, not a permanent spelling**, and its price — all ten Level-1 kernels crossing
`StackSmall` and losing `nosplit`, `internal/l1` +15.5% static instructions — was never
actually paid: the 2026-08-12 disposition parked the implementation inside #22's campaign
rather than landing it as a point patch, so `vec.LoadPart512` still calls
`archsimd.LoadFloat32x16SlicePart` directly and the +15.5% remains a measured *quotation*
for a change that is now obsolete before it was written. The admissibility condition on
#22's candidates is therefore satisfied by the toolchain rather than by a workaround, so
the masked-partial candidate is measured **as written** instead of in a costume.

**Correction (2026-08-15), same day.** The sentence above originally read "on 1.27 the four
`-race` criteria clear with no keel change at all". That was a prediction, and running it is
what refuted it: **keel does not compile under `go1.27rc3`** — `archsimd`'s load/store were
renamed, with two of the old names surviving under new parameter types, and the port is 51
type errors across three files ([T23](#t23)). So there *is* a keel change between here and
those four criteria, the criteria remain **unmeasured** on 1.27 rather than clear, and the
chain is floor → port → `-race` → criteria. Recorded rather than quietly rewritten because
the failure mode it illustrates is the one this file exists for: a fix confirmed in the
*compiler* was assumed to be a fix in *keel's build*, which is a different claim needing its
own run.

---

<a name="t18" id="t18"></a>
## T18 — a loop-invariant vector constant is re-materialized every iteration

**Toolchain.** go1.26.5, `GOEXPERIMENT=simd`, cross-compiled to `linux/amd64`.

`internal/vec.Abs512` has to build its mask from a constant, because T3 says there is
no float32 `Abs` and no float32 bitwise op to use instead. The mask is
loop-invariant. It is not hoisted: it is rebuilt on every iteration of any loop that
inlines the function.

**Repro.** Two loops that differ only in where the source puts the constant.

```go
//go:build goexperiment.simd && amd64

package licm

import "simd/archsimd"

const signMaskI32 int32 = -1 << 31

func inLoop(x, out []float32) {
	acc := archsimd.BroadcastFloat32x16(0)
	for i := 0; i+16 <= len(x); i += 16 {
		v := archsimd.LoadFloat32x16Slice(x[i : i+16])
		abs := v.AsInt32x16().AndNot(archsimd.BroadcastInt32x16(signMaskI32)).AsFloat32x16()
		acc = acc.Add(abs)
	}
	acc.StoreSlice(out)
}

func hoisted(x, out []float32) {
	acc := archsimd.BroadcastFloat32x16(0)
	mask := archsimd.BroadcastInt32x16(signMaskI32)
	for i := 0; i+16 <= len(x); i += 16 {
		v := archsimd.LoadFloat32x16Slice(x[i : i+16])
		abs := v.AsInt32x16().AndNot(mask).AsFloat32x16()
		acc = acc.Add(abs)
	}
	acc.StoreSlice(out)
}
```

```
GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go build -gcflags=-S ./...
```

`inLoop` — the loop is entered at `JMP 64`, its latch is `[64,81]`, and its back-edge
is `JLS 28`, so the body is `[28,61]`. The three mask instructions are at 32/38/43,
**inside**:

```
0x001a 00026 (licm.go:13)	JMP	64
0x001c 00028 (licm.go:14)	LEAQ	(AX)(DX*4), R9
0x0020 00032 (other_gen_amd64.go:211)	MOVL	$-2147483648, R10     <- in the loop
0x0026 00038 (other_gen_amd64.go:211)	VMOVD	R10, X1               <- in the loop
0x002b 00043 (other_gen_amd64.go:211)	VPBROADCASTD	X1, Z1        <- in the loop
0x0031 00049 (licm.go:15)	VPANDND	(R9), Z1, Z1
0x0037 00055 (<unknown line number>)	NOP                           <- T9
0x0037 00055 (licm.go:16)	VADDPS	Z1, Z0, Z0
0x003d 00061 (licm.go:13)	MOVQ	R8, DX
0x0040 00064 (licm.go:13)	LEAQ	16(DX), R8
...
0x0051 00081 (licm.go:14)	JLS	28
```

`hoisted` — the same three instructions at 14/19/34, ahead of the `JMP 63` that
enters the loop:

```
0x000e 00014 (other_gen_amd64.go:211)	MOVL	$-2147483648, DX
0x0013 00019 (other_gen_amd64.go:211)	VMOVD	DX, X0
0x0022 00034 (other_gen_amd64.go:211)	VPBROADCASTD	X0, Z0
0x002a 00042 (licm.go:25)	JMP	63
0x0030 00048 (licm.go:27)	VPANDND	(R9), Z0, Z2
0x0036 00054 (licm.go:28)	VADDPS	Z2, Z1, Z1
```

Total function size barely moves (116 vs 115 bytes) — the instructions are not
removed, they are relocated out of the loop. That is the whole effect, and it is the
effect that matters.

**Not a register-pressure decision, as far as this repro can tell.** The hoisted form
*stays* hoisted, so nothing forces the materialization back down at this pressure.
But `hoisted` uses three vector registers, and the real `avx512Asum` runs four
accumulators against T10's 15-register ceiling — so whether the allocator would
rematerialize the constant into the body under real pressure is a question this repro
does not answer. It has to be re-audited in place, not assumed.

**Re-audited in place, 2026-08-12: it rematerializes, and it is still not pressure.**
`go build -gcflags=-S ./internal/l1/` on the real `avx512Asum`. The unrolled loop is
entered at `JMP 141`, its condition is at `[141,145]`, and its back-edge is `JGE 40`,
so the body is `[40,137]` — and the three mask instructions are the *first* three of
it:

```
0x0028 00040 (other_gen_amd64.go:211)	MOVL	$-2147483648, DX   <- back-edge target
0x002d 00045 (other_gen_amd64.go:211)	VMOVD	DX, X4
0x0031 00049 (other_gen_amd64.go:211)	VPBROADCASTD	X4, Z4
0x0037 00055 (vec_avx512.go:94)	VPANDND	(AX), Z4, Z5
0x003d 00061 (vec_avx512.go:94)	VPANDND	64(AX), Z4, Z6
0x0044 00068 (vec_avx512.go:94)	VPANDND	128(AX), Z4, Z7
0x004b 00075 (vec_avx512.go:94)	VPANDND	192(AX), Z4, Z4    <- clobbers the mask
...
0x008d 00141 (l1_amd64.go:117)	CMPQ	BX, $64
0x0091 00145 (l1_amd64.go:117)	JGE	40
```

The last line looked like the interesting one at the time. **The fourth `VPANDND` writes
its result over `Z4`, the register holding the mask**, and the highest vector register
this function touches is `Z7`, so `Z8`–`Z15` are free and unused, well inside T10's
15-register ceiling.

**The causal reading that stood here was wrong; the reduction below is what corrected
it.** It said the allocator "chose to clobber a live loop-invariant value with a free
register available", that this was *why* nothing hoists, and that it was the same
character as T8's `VFMADD213PS` finding — a cost in operand assignment. The direction is
the other way round. Because the build sits inside the body, the mask's last use *is*
that fourth `AndNot`; the value is dead at that point, so reusing its register is both
free and correct, and the eleven idle registers are irrelevant. The clobber is a
consequence of the missing hoist, not its cause, and there is no operand-assignment
defect in this dump at all. The observation stands; the inference does not.

Two further details from the same dump:

- **Three loops rematerialize it, not one.** The 64-lane unrolled loop (above), the
  16-lane cleanup loop (`JGE 149`, mask rebuilt at 166–175), and the masked tail
  (rebuilt at 238–247). Six `VPBROADCASTD` mask builds in the function. The *worst*
  ratio is the cleanup loop — 3 mask instructions against 2 useful vector ops — though
  it runs at most three times.
- **`avx2Asum` does the same thing for the same reason.** `VPANDN Y8, Y5, Y5`
  clobbers the mask in `Y5`, with `Y9`–`Y15` free.

The unrolled body is 29 instructions: 3 mask, 4 `VPANDND`, 4 `VADDPS`, 8 one-byte NOPs
(T9/#17), 1 alignment NOP, 7 pointer/counter (T19's BCE payment), 2 for the latch. So
8 of 29 do the arithmetic. (This count does not match the "41 instructions, of which
#47's bounds checks are 7" written above; that number needs rechecking against a
current build before either is relied on, and the discrepancy is recorded rather than
silently overwritten.)

**One more observation, not yet reduced to a repro.** The AVX-512 form folds the load
into the operation — `VPANDND (AX), Z4, Z5` — and the AVX2 form does not, emitting
`VMOVDQU (AX), Y4` then `VPANDN Y4, Y5, Y4`. Both encodings admit a memory source, so
this looks like a lowering-rule gap on the AVX2 side rather than an encoding
limitation, and it is adjacent to T12's memory-operand story (#20). It is written down
as an observation from real code; it needs its own minimal repro before it becomes a
note in its own right.

**Interaction with T8.** CSE and this are orthogonal, and the combination is why the
observed cost is smaller than the naive count. `avx512Asum` inlines `Abs512` four
times per iteration; all four `VPANDND`s share `Z5`, so the mask is built **once**
per iteration, not four times. CSE works within an iteration; nothing lifts across
iterations. The cost is therefore 3 instructions per 64 elements, not 12 — which is
what #8 asked and is the reason the answer to #8 is "partially".

**Reduced to a minimal repro, 2026-08-12, and the trigger is one thing.** The re-audit
above is a reading of real code; this is the 15-line version, and the reduction was worth
doing because the first attempt at it **did not reproduce** and the difference between the
two attempts is the whole finding.

The repro (`GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go build -gcflags=-S`):

```go
package andnotrepro

import "simd/archsimd"

// The mask is built HERE, inside the callee, exactly as internal/vec.Abs512 does.
func abs(x archsimd.Float32x16) archsimd.Float32x16 {
	return x.AsInt32x16().AndNot(archsimd.BroadcastInt32x16(-2147483648)).AsFloat32x16()
}

func Sum(x []float32) float32 {
	acc := archsimd.BroadcastFloat32x16(0)
	for len(x) >= 64 {
		acc = acc.Add(abs(archsimd.LoadFloat32x16Slice(x[0:16])))
		acc = acc.Add(abs(archsimd.LoadFloat32x16Slice(x[16:32])))
		acc = acc.Add(abs(archsimd.LoadFloat32x16Slice(x[32:48])))
		acc = acc.Add(abs(archsimd.LoadFloat32x16Slice(x[48:64])))
		x = x[64:]
	}
	/* horizontal-sum tail, omitted; it does not affect the loop */
}
```

emits, with `JMP 136`, the condition at `[136,140]` and the back-edge `JGE 43`, so the
body is `[43,140]` and the first two instructions of it rebuild the mask:

```
0x003f 00063 (…/archsimd/other_gen_amd64.go:211)  MOVL  $-2147483648, SI   <- inside the body
0x0048 00072 (…/archsimd/other_gen_amd64.go:211)  VPBROADCASTD  X2, Z2
0x004e 00078 (repro.go:6)  VPANDND  (AX), Z2, Z3
0x005a 00090 (repro.go:6)  VPANDND  64(AX), Z2, Z4
0x0067 00103 (repro.go:6)  VPANDND  128(AX), Z2, Z4
0x0074 00116 (repro.go:6)  VPANDND  192(AX), Z2, Z2   <- clobbers the mask
```

Highest vector register used: `Z4`. `Z5`–`Z15` are free and unused — eleven of them.

**The negative control, which is also the fix.** Hoist the mask by hand — take it as a
parameter, `func abs(x archsimd.Float32x16, mask archsimd.Int32x16)`, and build it once
in the caller before the loop — and the rematerialization **disappears**: the build moves
to the preheader (`VPBROADCASTD X1, Z1` before `JMP`, body `[59,141]`), the mask lives in
`Z1` for the whole loop, and the four `VPANDND` write `Z3/Z4/Z4/Z4` without touching it.
Same instruction count in the body minus the two mask instructions.

Three variables were then held against the reproducing form and **none of them matters**:
one accumulator vs. four, one loop vs. three, and the presence of the 16-lane cleanup and
masked tail. All four combinations rematerialize; the hand-hoisted form never does.

**The trigger, after one more control: the constant is simply never hoisted.** What stood
here was that the trigger is "whether the invariant reaches the loop as a CSE'd value from
inside the callee or as a live local", and that the compiler lifts the constant to the
preheader in both cases. Both halves are wrong. The discriminating control is a third arm
with **no helper function at all** — the constant written directly in the loop body, so
nothing is inlined and nothing is CSE'd across a call boundary:

```go
func SumNoHelper(x []float32) archsimd.Float32x16 {
	acc := archsimd.BroadcastFloat32x16(0)
	for len(x) >= 64 {
		m := archsimd.BroadcastInt32x16(-2147483648)
		acc = acc.Add(archsimd.LoadFloat32x16Slice(x[0:16]).AsInt32x16().AndNot(m).AsFloat32x16())
		acc = acc.Add(archsimd.LoadFloat32x16Slice(x[16:32]).AsInt32x16().AndNot(m).AsFloat32x16())
		acc = acc.Add(archsimd.LoadFloat32x16Slice(x[32:48]).AsInt32x16().AndNot(m).AsFloat32x16())
		acc = acc.Add(archsimd.LoadFloat32x16Slice(x[48:64]).AsInt32x16().AndNot(m).AsFloat32x16())
		x = x[64:]
	}
	return acc
}
```

It emits the identical defect — body `[17,115]` (`JMP 111`, back-edge `JGE 17`), mask
rebuilt at 37/42/46 inside it, and the same `VPANDND 192(AX), Z2, Z2` — at 124 bytes
against the callee form's 123, one `XCHGL` nop apart.

So the callee, the inlining and the CSE are all incidental. The single fact is that **a
SIMD op with loop-invariant operands is not lifted out of the loop**, and the hand-hoisted
arm "works" only because its definition was already outside the loop in the source and
never needed lifting. `hoisted`/`SumLocal` is not successful LICM; it is LICM not being
required.

**Method note, because this is the second wrong isolation in one note** — and the second
structurally-uninformative check in one campaign, so it is named in DESIGN.md §5.7 (*"a
check that could not have come out otherwise is not evidence"*) alongside #48's tautology
trap. The two arms of
the reduction differed in *two* ways at once — where the definition sits relative to the
loop, and whether it arrives through a callee — and the write-up attributed the effect to
the salient difference without varying it alone. The three variables that were dutifully
held were all ones that did not matter; the one that did was never held. The first minimal
repro not reproducing was a finding; the second reproducing was not, by itself, an
isolation.

**Already filed upstream: golang/go#79984, "cmd/compile: simd operations not marked as
hoistable."** Searched before filing anything, which is why nothing was filed. The issue
predates this note. `randall77` gives the cause — LICM was extended to more ops in
CL 697235 / CL 772964 and SIMD ops were mostly not allowed, the hard part being that they
must not be lifted past their CPU feature check — `dr2chase` points at
`ssa/cpufeatures.go`, and **`balasanjay`'s addendum is keel's exact shape**: a vector
constant built inside a helper, an expectation of inline-then-hoist, the same workaround of
passing the constant in, and ~4× throughput in a vectorised base64 decoder. A fix is in
flight, **CL 803220 "cmd/compile: handle SIMD ops in LICM"** — as of 2026-08-12 `NEW` and
still **work-in-progress**, last updated 2026-07-25, so not in go1.26.5 and the workaround
is still needed.

Nothing was posted upstream. The repro adds no fact the issue does not already carry, and
the one thing keel could contribute that is not there yet is a *measured* delta on a real
kernel — which #8's A/B will produce. A number earns a comment; a fourth restatement of a
known miss does not.

**Consequence for keel.** Issue #8, and the repro above names the fix. One `Abs512` call
site pattern, in the two `Asum` loops (`avx512Asum`, `avx2Asum`; the two `SumSq`
neighbours do not use the mask). The hoist means changing `internal/vec`'s API, because
`Abs512(x)` builds the mask internally and there is nowhere else to put it — the negative
control says a mask parameter is exactly what removes the rematerialization — which needs
a scalar twin and a differential test like every other vector op. It is 3 of the
instructions in `avx512Asum`'s loop; #47's bounds checks are 7 of them. Both edit the same
bodies, so they should be measured together rather than as two claims about the same
benchmark.

---

<a name="t19" id="t19"></a>
## T19 — the slice-advancing loop pays for its own bounds-check elimination, and `>=` is why

**Toolchain.** go1.26.5, cross-compiled to `linux/amd64`. Not simd-specific: this is
plain slice arithmetic and reproduces with no `archsimd` import at all. It is here
because it is the cost side of the idiom CLAUDE.md requires of kernels ("pre-sliced
panels"), and issue #47 is where keel paid it.

**Observation.** Two things, and the second is the useful one.

1. An index-driven loop guarded by `i+4 <= len(x)` does **not** discharge the
   sub-slice bounds: `i+4 <= len(x)` plus the slice invariant `len(x) <= cap(x)`
   implies `i+4 <= cap(x)`, and `prove` does not take that step. Rewriting the loop
   to advance the slice (`x = x[4:]`) with *constant* offsets in the body removes
   every check, because the reasoning becomes constant-versus-constant.
2. The rewrite is not free. `x = x[4:]` compiles to **seven** instructions, five of
   them a branchless conditional: the pointer is advanced only when the remaining
   length is still positive. Guard the same loop with `len(x) > 4` instead of `>= 4`
   and it collapses to a single `ADDQ`.

**Repro.** Three loops with identical bodies.

```go
package adv

func indexed(x []float32, a float32) {
	for i := 0; i+4 <= len(x); i += 4 {
		x[i] *= a; x[i+1] *= a; x[i+2] *= a; x[i+3] *= a
	}
}

func advancing(x []float32, a float32) {
	for len(x) >= 4 {
		x[0] *= a; x[1] *= a; x[2] *= a; x[3] *= a
		x = x[4:]
	}
}

func strict(x []float32, a float32) {   // the only change is >= to >
	for len(x) > 4 {
		x[0] *= a; x[1] *= a; x[2] *= a; x[3] *= a
		x = x[4:]
	}
}
```

`GOOS=linux GOARCH=amd64 go build -gcflags=-S`:

`indexed` — four surviving checks, one per offset, each with its own panic block:

```
	00041 (adv.go:6)	CMPQ	BX, CX
	00044 (adv.go:6)	JLS	143        ---> CALL runtime.panicBounds
	00064 (adv.go:7)	CMPQ	BX, SI
	00067 (adv.go:7)	JLS	138        ---> CALL runtime.panicBounds
	00089 (adv.go:8)	JLS	133        ---> CALL runtime.panicBounds
	00114 (adv.go:9)	CMPQ	BX, SI
	00117 (adv.go:9)	JHI	13
```

`advancing` — no checks, no panic block, and this advance:

```
	00061 (adv.go:20)	ADDQ	$-4, CX     <- cap -= 4
	00065 (adv.go:20)	MOVQ	CX, DX
	00068 (adv.go:20)	NEGQ	DX
	00071 (adv.go:20)	SARQ	$63, DX     <- DX = -1 if the new cap is > 0, else 0
	00075 (adv.go:20)	ANDL	$16, DX     <- ... so 16 bytes, or 0
	00078 (adv.go:20)	ADDQ	DX, AX      <- advance the data pointer, conditionally
	00081 (adv.go:20)	ADDQ	$-4, BX     <- len -= 4
	00085 (adv.go:15)	CMPQ	BX, $4
	00089 (adv.go:15)	JGE	7
```

`strict` — no checks, and the advance is what one would have written by hand:

```
	00061 (adv.go:31)	ADDQ	$16, AX
	00065 (adv.go:31)	ADDQ	$-4, BX
	00069 (adv.go:26)	CMPQ	BX, $4
	00073 (adv.go:26)	JGT	7
```

Function sizes: 91 bytes for `advancing`, 75 for `strict`.

**Why.** Not a missed optimization. With `>=`, the last iteration can take a slice of
length exactly 4 down to length 0, and Go's runtime requires a slice's data pointer to
stay inside its allocation — a pointer one past the end can be attributed to the *next*
heap object and keep it alive, or worse. So the compiler emits the conditional. With
`>` the loop provably exits while at least one element remains, the pointer can never
reach the end, and the guard is unnecessary. The five instructions are GC correctness,
not waste.

**Consequence for keel.** Issue #47. The shaping rewrite took all ten `internal/l1`
vector loops from 1–11 surviving bounds-check exits to **0**, and the effect on
instruction count is split by whether the loop is unrolled:

| function | insns (real, excl. T9 nops) | per arith | bounds-check exits |
|---|---|---|---|
| `avx512Asum` | 41 → **20** | 10.25 → 7.00 | 7 → **0** |
| `avx512Dot` | 61 → **32** | 17.25 → 13.00 | 11 → **0** |
| `avx512SumSq` | 42 → **21** | 12.50 → 8.25 | 7 → **0** |
| `avx512Axpy` | 16 → **21** | 21.00 → 26.00 | 2 → **0** |
| `avx512Scal` | 11 → **12** | 14.00 → 15.00 | 1 → **0** |

The three 4×-unrolled reductions roughly halve, because one advance is amortized over
four vector ops. `Axpy` gets 5 instructions *worse* and `Scal` 1, because they are not
unrolled and `Axpy` advances two slices: 2 × 7 instructions of advance against one FMA.
The `>` form would recover exactly those, at the price of handing the last full vector
to the masked-store tail path. That trade is not taken here — it is a runtime question
and #47's deliverable is a measurement, not a second guess.

**The `>` form, taken (2026-08-14, go1.26.6).** #47's A/B answered the runtime
question: Saxpy regressed at 256, 4096 *and* 65536 on all three hosts, worst −40.65%,
and Saxpy's loop is the one where the advance outweighs the work. So all ten loops
moved to `>`. Steady-state loop instructions, this time counting everything in the
body (so these totals are not comparable with the table above, which excludes T9's
NOPs — the *deltas* are what carry over):

| function | insns | of which bookkeeping | function | insns | bookkeeping |
|---|---|---|---|---|---|
| `avx512Dot`   | 52 → **44** | 16 → **8** | `avx2Dot`   | 52 → **44** | 16 → **8** |
| `avx512Axpy`  | 26 → **15** | 17 → **6** | `avx2Axpy`  | 29 → **18** | 17 → **6** |
| `avx512Scal`  | 16 → **9**  | 10 → **4** | `avx2Scal`  | 19 → **12** | 10 → **4** |
| `avx512Asum`  | 29 → **25** | 10 → **6** | `avx2Asum`  | 41 → **37** | 10 → **6** |
| `avx512SumSq` | 34 → **29** | 9 → **5**  | `avx2SumSq` | 33 → **29** | 9 → **5** |

Vector-op counts are unchanged in all ten, and all ten still audit at 0 bounds-check
exits, 0 calls and 0 spills. The advance is now `ADDQ $64, AX` (plus `ADDQ $64, DI`
where a second slice is advanced), exactly as `strict` above predicts.

**The anticipated price was avoidable.** The paragraph above assumed `>` means handing
the last full vector to the masked tail. It does not have to. The reductions already
carry a 16-wide mop-up loop beneath the unrolled one; it keeps `>=`, drains whatever
`>` leaves in at most four iterations, and the tail never sees a full vector. *That last
inference was wrong about cost — see "The mop-up loop is not free" below.* Axpy and
Scal have no such loop, so they got an explicit exact-fit epilogue that runs the body
once at full width. That matters beyond one masked store: without it, *every*
exact-multiple call would execute a partial op where today only ragged lengths do, and
partial ops are what #42 makes fatal under `-race`. The epilogue keeps that frequency
where it was.

Two variants were measured and rejected. An early `return` on exact fit, to hand the
prover a `len != 16` fact before the advance, is not used by `prove`: Scal 16 → 15, and
Axpy got *worse* at 27. Dropping the redundant `&& len(y) > 16` conjunct — y having been
re-sliced to `len(x)` above the loop — costs 9 instructions (Axpy 15 → 24), so the
conjunct is load-bearing rather than defensive.

**The mop-up loop is not free** (2026-08-14, after the A/B). "The tail never sees a full
vector" was true and taken to mean the reductions needed no epilogue. The A/B
(`build/l1ab-a2b76eb.log`) refutes the cost claim on all three hosts at once — the only
unanimous signal in the run:

| n=256 | Zen 4 | Skylake-X | Zen 5 |
|---|---|---|---|
| Sdot  | +6.2% | +13.7% | +9.4% |
| Sasum | +9.6% | +12.8% | +17.4% |
| Snrm2 | +4.5% | +6.6%  | +9.8% |

(time; positive is slower.) 256 is an exact multiple of `step512 = 64`, so `>=` consumed it
in four unrolled iterations while `>` stops with 64 left and the mop-up loop takes them as
four 16-wide iterations — all accumulating into `a0`. Four independent FMAs become a
four-deep dependent chain at ~4-cycle latency. Instruction count barely moves; the
*dependency graph* is the cost, which is why a static audit could not have caught this and
only the benchmark did. It is the same latency stall the four-accumulator design was
adopted to avoid, reintroduced one level down, and the general lesson is that a drain loop
inherits none of the ILP of the loop it drains.

The fix is the epilogue the reductions were argued out of: `if len(x) == step512 { <body
once, four accumulators> ; x = x[:0] }`. `x[:0]` rather than `x[step512:]` because
truncating a length cannot produce a past-the-end pointer — the whole subject of this note —
so it needs no conditional bump, and the mop-up loop and partial tail fall through untaken.

**Also observed, not filed.** The two-conjunct guard emits a redundant branch: `CMPQ
BX, $16; JLE 86; JGT 37`, where `JLE` already decided it. Same in the epilogue's `JNE
115; JNE 115`. One wasted branch per iteration, correctly predicted; below the noise
floor of anything keel can measure, and named here only so the next reader of this
assembly does not take it for a bug.

---

<a name="t20" id="t20"></a>
## T20 — `benchstat A B` silently declines to compare A and B when a config line differs

**Observation.** benchstat groups results into one table per distinct
*configuration*, where a configuration is every `key: value` line in the log
(Go's benchmark format allows them anywhere, applying to the rows that follow).
Two files that differ in one such key are therefore never compared. The output
still contains both arms' numbers, in table form, one table per arm — so it
*looks* like a comparison — but it contains no delta, no percentage and no
p-value. The exit status is 0.

Minimal repro. Two logs of the same benchmark, one 9.7% faster, differing only
in a `keel-bench-clock-mhz` line:

```
$ cat r-base.txt
goos: linux
goarch: amd64
pkg: x/bench
keel-bench-clock-mhz: 2986-5164 (snapshot, not sustained)
BenchmarkFoo 100 10.1 ns/op          (x6, 10.1 … 10.6)
$ cat r-new.txt
goos: linux
goarch: amd64
pkg: x/bench
keel-bench-clock-mhz: 3001-5150 (snapshot, not sustained)
BenchmarkFoo 100 9.1 ns/op           (x6, 9.1 … 9.6)

$ go tool benchstat r-base.txt r-new.txt
goos: linux
goarch: amd64
pkg: x/bench
keel-bench-clock-mhz: 2986-5164 (snapshot, not sustained)
    │ /tmp/bstest/r-base.txt │
    │         sec/op         │
Foo              10.35n ± 2%

keel-bench-clock-mhz: 3001-5150 (snapshot, not sustained)
    │ /tmp/bstest/r-new.txt │
    │        sec/op         │
Foo             9.350n ± 3%
exit=0
```

The same two files with the volatile key ignored:

```
$ go tool benchstat -ignore=keel-bench-clock-mhz r-base.txt r-new.txt
goos: linux
goarch: amd64
pkg: x/bench
    │ /tmp/bstest/r-base.txt │       /tmp/bstest/r-new.txt       │
    │         sec/op         │   sec/op     vs base              │
Foo             10.350n ± 2%   9.350n ± 3%  -9.66% (p=0.002 n=6)
exit=0
```

**Why.** Deliberate, and right for benchstat's usual job: if two runs were made
under different conditions, averaging them into one comparison would be worse
than declining to. `-ignore` exists precisely to say which keys are not
conditions. The surprise is only that declining is *silent* — same exit status,
similar-looking output, and the difference between "no significant change" and
"never compared" is one absent column.

**Consequence for keel.** This one is self-inflicted, which is the interesting
part. keel prints a provenance preamble (`keel-bench-cpu`, `keel-bench-kern`,
`keel-bench-peak-method`, …) so that no number ships without its denominator —
DESIGN.md §7 rule 7. Those markers are `key: value` lines, so they are in
benchstat's configuration namespace whether we intended that or not, and one of
them, `keel-bench-clock-mhz`, is a snapshot of the CPU's current frequency
range: it differs between any two runs on the same host by construction.

The first run of `scripts/l1-bench.sh` therefore produced a per-host A/B of
#47's loop reshaping in which no arm was ever compared to the other — 20
benchmarks × 3 hosts × 2 builds of medians with not one delta among them. The
numbers were all correct; the measurement was not the one the script claimed to
make. Fixed in `scripts/bench.sh:bench_compare`, which ignores the volatile
keys *and then checks that a `vs base` column actually appeared*, reporting
which configuration keys forked the table when it did not. A comparison that did
not happen now reads as a failed measurement rather than as a table.

The general lesson for anything that prints machine-readable provenance into a
benchmark log: those lines are not comments, and a tool downstream may treat
them as the identity of the experiment.

<a name="t21" id="t21"></a>
## T21 — benchstat's CI is an integer percent, so `± 0%` means "under 0.5%", not "zero"

**Observation.** benchstat formats its confidence interval as a whole number of
percent, in both the text table and the CSV the scripts here parse. A
measurement tight enough to round down prints `0%`, which reads as an exact
number and is not one: it is an interval whose only stated property is that it
is narrower than half a percent.

Minimal repro. Ten samples spread ±0.2% about 100 ns, then the same at ±0.5%:

```
$ head -5 tight.txt
goos: linux
goarch: amd64
pkg: x/bench
BenchmarkFoo 1 99.800 ns/op
BenchmarkFoo 1 100.200 ns/op

$ go tool benchstat tight.txt
    │ /tmp/cist/tight.txt │
    │       sec/op        │
Foo           100.0n ± 0%

$ go tool benchstat -format=csv tight.txt
,sec/op,CI
Foo,1.0000000000000001e-07,0%

$ go tool benchstat -format=csv loose.txt   # samples ±0.5% instead of ±0.2%
,sec/op,CI
Foo,1.0000000000000001e-07,1%
```

**Why.** A display choice, and a defensible one: benchstat's usual question is
whether two configurations differ, and for that a percent is the useful unit and
a tenth of a percent is noise about noise. The rounding is only a problem for a
consumer that does arithmetic with the interval instead of reading it.

**Consequence for keel.** `scripts/bench.sh:bench_stat` divides that percent by
100 and every gate here compares a *median net of its CI* against a threshold,
so on any measurement tight enough to print `0%` the gates compare a bare median
and their stated safety margin is silently zero. That direction is conservative
for a floor — a threshold cleared by the median alone is cleared — so no shipped
criterion is wrong because of this. It is not conservative for a *difference*
between two arms, which is what `scripts/retention.sh feed` is built out of:
that instrument prints a `worst-ci` column so a reader can tell a resolved step
from noise, and on vesta it printed `0%` on seven of eight rows. Read as
written, that says every step is resolved, including the ±0.60 ns ones. Read
correctly, it bounds the noise floor at 0.5% of the arm — up to 4.4 ns on the
4×32 kc=512 row, which is larger than three of the four panel-feed steps that
row's decomposition reports.

So the floor is now printed in nanoseconds as the upper bound the rounded
percent actually supports (`0.005 × the largest arm on the row`), and a step
below it is labelled unresolved rather than left for the reader to divide. The
percent alone was a number without its denominator; the denominator is the arm
it is a percentage of.

The general lesson, and it is the same one as T20: a formatted number is a
statement to a human, and a script that parses it inherits the formatting's
precision, not the underlying quantity's.

**Correction (2026-08-19), seven days later.** "That direction is conservative for
a floor — a threshold cleared by the median alone is cleared — so no shipped
criterion is wrong because of this" has the sign backwards. With the CI read as 0
the check becomes *median ≥ floor*, which is **easier** to pass than *median net of
CI ≥ floor*: lenient, not conservative. The gate has since produced the verdict
that paragraph said could not exist — janus Strsm flipped FAIL → PASS on a
**0.014%** move in the point estimate (7.0098 → 7.0101), because one arm's reported
CI crossed 0.5% from `1%` to `0%` and with both arms at `0%` the net-of-CI bound
*is* the raw ratio. One rounding step is worth 0.1386 on a 7.0 ratio; the margins
being adjudicated are 0.011 and 0.081. Nothing in the tree changed here yet: the
fix to `bench_ratio_lo`, and the two `zero-width` justifications resting on the
same misreading, are [#110](https://github.com/scttfrdmn/keel/issues/110), open on
a decision. A correction rather than a rewrite because the observation and repro
above are still exactly right — what failed was the consequence reasoned from them,
published without being run against the instrument it described (§5 rule 11).

<a name="t22" id="t22"></a>
## T22 — a commit that changes no instruction bytes in a function can make it 45% slower, and function-entry alignment mod 64 is the discriminator

**Toolchain.** go1.26.6, cross-compiled to `linux/amd64`. Not simd-specific in
mechanism, but found on SIMD kernels and worth this much space because it sets an
upper bound on what any A/B in this repo can attribute.

**Observation.** Commit `53417e8` added an exact-fit epilogue to the six reductions
(#59). It touches no line of `avx512Axpy` or `avx512Scal` — `git diff` has zero Axpy
and zero Scal lines. On antares (RYZEN AI MAX+ 395, Zen 5) those two untouched
routines nevertheless moved, a long way:

| antares, time, `a1c9aa6` → `53417e8` | delta |
|---|---|
| Saxpy n=256   | +26.80% |
| Saxpy n=4096  | **+45.06%** |
| Saxpy n=65536 | +20.89% |
| Sscal n=4096  | +19.40% |

Both arms ±0–1%, p=0.000, n=10. The same commit moved these cells by ~0% on
Skylake-X and by at most +7.04% on Zen 4, so the *magnitude* is a property of the
microarchitecture, not of the change.

**Minimal repro.** Build the bench binary at both revisions and compare the
disassembly of a routine the diff does not touch:

```
go tool objdump -s 'l1\.avx512Scal$' bench-a1c9aa6.bin | awk 'NR>1{print $3,$4}' > a.txt
go tool objdump -s 'l1\.avx512Scal$' bench-53417e8.bin | awk 'NR>1{print $3,$4}' > b.txt
diff a.txt b.txt        # empty: 46 instructions, byte-identical
```

`avx512Axpy` likewise differs in exactly one byte-field across the whole function —
the relative displacement of an off-hot-path `CALL`. The code is the same code. What
changed is where it sits:

| entry address | `a1c9aa6` | `53417e8` |
|---|---|---|
| avx512Dot   | 0x53c320 (mod64=32) | 0x53c320 (mod64=32) |
| avx512Axpy  | 0x53c540 (**mod64=0**) | 0x53c5e0 (mod64=32) |
| avx512Scal  | 0x53c680 (**mod64=0**) | 0x53c720 (mod64=32) |
| avx512Asum  | 0x53c700 (**mod64=0**) | 0x53c7a0 (mod64=32) |
| avx512SumSq | 0x53c880 (**mod64=0**) | 0x53c960 (mod64=32) |

`avx512Dot`'s epilogue grew it by 0xa0 = 160 bytes. 160 is a multiple of Go's
32-byte function alignment on amd64 but not of 64, so every function after it in the
object flipped from a 64-byte-aligned entry to 64+32 — and the two whose code did not
change are precisely the two that regressed.

**What it is not.** Two hypotheses were tested and both failed:

  - *Loop alignment within the function.* It points the wrong way. The base `avx512Axpy`
    hot loop (`0x53c565`–`0x53c596`) straddles a 64-byte boundary; the new one
    (`0x53c605`–`0x53c636`) fits entirely inside one line. The better-aligned loop is
    the slower one, and both span two 32-byte fetch windows either way.
  - *Arm order in the harness.* `scripts/l1-bench.sh` always runs base first and new
    second, which would penalise the second arm on a thermally-limited part. Ruled out
    by re-running the identical pair with the arms swapped: Saxpy n=4096 went from
    +45.06% to −30.98%, i.e. the sign follows the *code*, not the position. Absolute
    numbers reproduce regardless of arm position (`53417e8` 157.9n as second arm,
    158.5n as first; `a1c9aa6` 108.9n as first, 109.4n as second). The harness is sound.

**Upstream.** This is `golang/go#8717` ("random performance fluctuations after
unrelated changes", open since 2014), and the mechanism is worked out in the comments
of `golang/go#18977`: recent amd64 branch predictors hash the jump IP, so moving code
creates and destroys prediction aliasing. dvyukov reported the same shape in 2016 —
a function at 0x10 fast, at 0x20 slow, fixed by forcing alignment — and rsc's note on
#8717 is that days of investigation produced no actionable rule. `golang/go#6752`
(explicit alignment annotations) is the knob that does not exist. Nothing here is a
new bug and nothing was filed; see the keel issue keyed to those three.

**Consequence for keel, which is the reason this entry is long.** An A/B in this repo
cannot attribute a delta smaller than the drift its own untouched routines show in the
same run. That floor is not a constant: it was ~0% on Skylake-X, ~7% on Zen 4 and
~45% on Zen 5 for one commit. So a run has to *carry its own control* — the delta on
routines the diff provably does not touch is the noise floor for that run on that
host, and any reported number has to clear it. This is the same discipline as "never a
number without its denominator", applied to the attribution rather than the units.

It also means a percent-of-peak floor and a two-arm A/B have different exposure: the
floor compares a median against a fixed denominator and is unaffected, while the A/B
compares two placements. Gates that assert a floor are safe; gates that assert an
improvement are not, unless they carry the control.

---

<a name="t23" id="t23"></a>
## T23 — `archsimd`'s load/store are renamed with a *swap*: the slice forms take the bare names, and the array forms gain an `Array` suffix

**Toolchain.** `go1.27rc3 linux/amd64`, installed to its own prefix
`/usr/local/go1.27rc3` on all three amd64 hosts (docs/hosts.md) and read back
there — `/usr/local/go` stays on `go1.26.5`, because clobbering it would silently
re-point every host-invoked gate step and every published number's compiler.
Compared against `go1.26.5 linux/amd64` on the same host, with the same command.
keel at `c983e3b`.

**Observation.** `simd/archsimd`'s load and store surface was reshaped between
1.26 and 1.27, and it is not a set of independent renames. Two of the old names
*survive with different parameter types*: the slice-taking forms were promoted
onto the bare names, and the array-taking forms they displaced were pushed out to
an `Array` suffix. A third pair changed shape as well as name, growing a return
value.

Both columns are `go doc` output, not recollection (CLAUDE.md's prime directive):

| go1.26.5 | go1.27rc3 | kind |
|---|---|---|
| `LoadFloat32x16(y *[16]float32) Float32x16` | `LoadFloat32x16Array(y *[16]float32) Float32x16` | displaced by the swap |
| `LoadFloat32x16Slice(s []float32) Float32x16` | **`LoadFloat32x16(s []float32) Float32x16`** | takes over the bare name |
| `LoadFloat32x16SlicePart(s []float32) Float32x16` | **`LoadFloat32x16Part(s []float32) (Float32x16, int)`** | renamed **and** returns a count |
| `(x Float32x16) Store(y *[16]float32)` | `(x Float32x16) StoreArray(y *[16]float32)` | displaced by the swap |
| `(x Float32x16) StoreSlice(s []float32)` | **`(x Float32x16) Store(s []float32)`** | takes over the bare name |
| `(x Float32x16) StoreSlicePart(s []float32)` | **`(x Float32x16) StorePart(s []float32) int`** | renamed **and** returns a count |
| `(x Float32x16) StoreMasked(y *[16]float32, m Mask32x16)` | `(x Float32x16) StoreArrayMasked(y *[16]float32, m Mask32x16)` | displaced by the swap |
| `BroadcastFloat32x16`, `BroadcastFloat32x8`, `BroadcastInt32x16`, `BroadcastInt32x8` | unchanged | — |

The same pattern holds for `Float32x8`, `Int32x16` and `Int32x8`.

**The mechanism, which makes the direction predictable.** 1.27 also ships the
portable `simd` package (T24), whose documented convention is slice-first:
`Load<Types>(s []T)`, `Load<Types>Part(s []T) (<Types>, int)`, `Store([]T)`,
`StorePart([]T) int`. `archsimd`'s new names are exactly that convention. So this
is not arbitrary churn — it is `archsimd` converging on the portable package's
spelling, and the useful prediction is that *the slice form is now the canonical
one and any array form is the suffixed special case*. Anything still named for a
`*[N]T` parameter is the shape most likely to move again.

**Minimal repro.** Both toolchains, one host, one command:

```
$ GOEXPERIMENT=simd /usr/local/go/bin/go doc simd/archsimd | grep 'LoadFloat32x16'
func LoadFloat32x16(y *[16]float32) Float32x16
func LoadFloat32x16Slice(s []float32) Float32x16
func LoadFloat32x16SlicePart(s []float32) Float32x16

$ GOEXPERIMENT=simd /usr/local/go1.27rc3/bin/go doc simd/archsimd | grep 'LoadFloat32x16'
func LoadFloat32x16(s []float32) Float32x16
func LoadFloat32x16Array(y *[16]float32) Float32x16
func LoadFloat32x16Part(s []float32) (Float32x16, int)
```

**keel does not compile under 1.27.** `go build -gcflags=-e ./...` (the `-e`
matters: without it the error list is truncated and the size of the port is
understated) reports 51 errors, every one a type error, in three files:

| file | errors | what |
|---|---|---|
| `internal/vec/gemm_amd64.go` | 35 | all `undefined: archsimd.LoadFloat32x16Slice` |
| `internal/vec/vec_avx2.go` | 9 | 2 undefined, 2 missing methods, 5 array-pointer-into-slice-parameter |
| `internal/vec/vec_avx512.go` | 7 | 2 undefined, 2 missing methods, 3 array-pointer-into-slice-parameter |

Sorted by the edit each needs, which is how the port should be planned:

- **Pure rename, signature unchanged — 39 sites.** 36 ×
  `archsimd.LoadFloat32x16Slice` → `LoadFloat32x16`, 1 ×
  `archsimd.LoadFloat32x8Slice` → `LoadFloat32x8`, 2 × `.StoreSlice` → `.Store`.
- **Rename plus a new return value — 4 sites.** `LoadFloat32x16SlicePart` and
  `LoadFloat32x8SlicePart` become `…Part` returning `(vector, int)`;
  `.StoreSlicePart` becomes `.StorePart` returning `int`. keel's shim wrappers
  have to decide whether to expose the count or discard it — it is the same
  quantity the wrappers currently recompute from `len(s)`.
- **Array-form sites that must gain the `Array` suffix — 8 sites.** Two
  `h4.Store(&a)` in the `HSum` fold trees (`vec_avx2.go:83`,
  `vec_avx512.go:129`), and the six `Block` round-trips in the test layer
  (`vec_avx512.go:159,163`, `vec_avx2.go:162,163,168,169`). These are real
  array stores of stack temporaries, so `StoreArray`/`LoadFloat32x16Array` is
  the right destination; they are not the T17 workaround and are not deleted.

**Every one of the 51 is caught by the type checker, and that is not luck.** A
name swap is the API change that *can* compile with new semantics, so it is worth
saying why this one cannot: `*[N]float32` and `[]float32` are mutually
unassignable in Go, so every displaced call fails to type-check rather than
silently rebinding, and the two `…Part` forms grow a second result, which Go
rejects in a single-assignment context. The general rule this instance
illustrates — a swap is safe only when the two overloads' parameter types are
mutually unassignable, and this one is — is what makes "it still compiles"
sufficient evidence *here* and insufficient in general.

**Consequence for keel.** The 2026-08-15 ruling makes go1.27 keel's floor, and
the first act under the floor was to be `-race` on the vector path. That is a
two-step chain that turns out to be four: **floor → port `internal/vec` → `-race`
→ the four `-race` gate criteria.** The `-race` question is still unanswered on
1.27, because the probe never got far enough to ask it: arm B failed at `go
build`, before any test ran. Instrumenting that probe for four outcomes
(`compile-fail` / `checkptr` / `data-race` / `clean`) rather than two is what kept
this from being reported as "checkptr is still broken".

The floor cannot move yet for a second, independent reason: `go1.27.0` **final
does not exist** — go.dev/dl lists `go1.27rc3` as the tip and `go1.26.6` as
stable — and the ruling's own condition is 1.27.0 final installed and read back on
all three hosts. Landing the port before then would break all three benchmark
hosts (`go1.26.5`) and the dev host (`go1.26.6`) simultaneously, taking the
project's whole measurement apparatus offline to chase a toolchain that has not
shipped. So the port is written against the rename table above and held.

**The one piece of good news, and it is the design bet paying off.** DESIGN.md §3
confines every `simd` import to `internal/vec` behind a hand-written shim, on the
argument that an experimental API will move. It moved, in the most invasive way
short of a semantic change, and the blast radius is 3 files, 51 lines, and **zero
changes to keel's own API**: `vec.Load512`, `vec.LoadPart512`, `vec.Store512`,
`vec.StorePart512` and the rest keep their names *and* their signatures — the
`…Part` wrappers absorb the new return value rather than passing it on — so
nothing outside `internal/vec` is touched. Five
files reference `simd/archsimd`; two of those references are comments.

**Amendment, 2026-08-28 — the hold ended by accident, and the port landed.** This
entry closed by *holding* the port: `go1.27.0` final did not exist, and landing it
would have broken the dev host and all three benchmark hosts at once. Instead
`go1.27.0` final arrived on the **dev host alone**, via Homebrew, between sessions
— and the dev host is the one that cross-compiles every judged binary, so the hold
was overtaken rather than lifted. The failure surfaced as a $3.888/hr fleet run
that measured nothing (`build/confirm-skx-7142b6f.log`, 6 FAIL / 12 UNMEASURED).

```
$ go version
go version go1.27.0 darwin/arm64
$ GOEXPERIMENT=simd go build ./...                        # darwin/arm64: silent
$ GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go build -gcflags=-e ./... 2>&1 | grep -c .
51
```

**What changed in the tree.** The port above landed verbatim — 51 errors, the same
count as against `rc3`, so the rename table and the 8-site array inventory are
**identical between rc3 and final**. `tools/shapegen/emit.go` had to be ported too,
which this entry's file inventory missed: `gemm_amd64.go` is generated, and
`internal/kern`'s fixed-point test is what caught the generator drifting from its
output. Two further 1.27 API facts, both recorded and neither acted on: `paFloat32x16`
now carries `//go:nocheckptr`, from [CL 761120](https://go-review.googlesource.com/c/go/+/761120)
(merged), which closed [golang/go#78413](https://github.com/golang/go/issues/78413) — the
still-open [golang/go#80856](https://github.com/golang/go/issues/80856) is a duplicate of it. That
settles the `-d=checkptr` half **only**: [golang/go#42880](https://github.com/golang/go/issues/42880),
open, records that `-race` does *not* obey `go:nocheckptr`, so the `-race` half is now
predicted **still broken** and an AVX-512 host's `-race` run stays the decisive branch. One
annotation, two consumers, opposite answers. And `(Float32x16) Abs()` now exists, retiring
this project's bitcast workaround at #54's convenience, not during a freeze.

**And a second machine, found by CI.** Moving the dev host to 1.27 made the tree
unbuildable on every machine still at 1.26 — CI included, which pins `1.26.x` and so
failed `fed1e70` with the same 51 errors *mirrored*: `cannot use bp[0:16] (value of type
[]float32) as *[16]float32 value in argument to archsimd.LoadFloat32x16`. The swap admits
no spelling that satisfies both toolchains, and `go.mod` cannot express which one is
required, because `archsimd` ships *with* the toolchain and is therefore not a module
requirement. So the vector path's floor is now go1.27 and is stated only in prose and in
CI's pin; the scalar path's floor stays go1.26.

**The line that does not generalise.** The first paragraph says clobbering
`/usr/local/go` "would silently re-point every host-invoked gate step and every
published number's compiler". That is exactly what happened, and *nothing in the
apparatus noticed*: no archive header records a compiler (17 `keel-bench-*`
provenance fields, none of them the toolchain), so a build-breaking upgrade was
loud only because it broke the build. A silent one would not have been.

---

<a name="t24" id="t24"></a>
## T24 — 1.27 ships the portable `simd` package, and it is vector-length-agnostic *at runtime*, which a register-blocked microkernel cannot be

**Toolchain.** `go1.27rc3 linux/amd64` vs `go1.26.5 linux/amd64`, same host,
`GOEXPERIMENT=simd`. Found while surveying the T23 rename.

**Observation.** T5 recorded that 1.26 has no portable `simd` package, only
`simd/archsimd`, and guessed "expected in 1.27". It is there:

```
$ ls $(/usr/local/go/bin/go env GOROOT)/src/simd/
archsimd

$ ls $(/usr/local/go1.27rc3/bin/go env GOROOT)/src/simd/
archsimd  doc.go  midway_amd64.go  midway_arm64.go  midway_wasm.go
simd_emulated.go  simd_stubs.go  simd_types.go  tofrom_amd64.go
tofrom_arm64.go  tofrom_wasm.go  ...  (tests elided)
```

It has amd64, arm64 and wasm backends plus a pure-Go emulated fallback, one
vector type per primitive numeric type (`Float32s`, `Int16s`, …), and a documented
bridge in both directions: `ToArch() any` and `Float32sFromArch[T](x T)`.

**The property that decides its usefulness here.** Its own doc: "the vector length
is at least 128 bits, and within a given program execution, all vectors have the
same length." The lane count is therefore a *runtime* quantity —
`simd.VectorBitSize() int`, `simd.Emulated() bool`, `(x Float32s) Len() int` —
with no compile-time constant anywhere in the API.

**Consequence for keel, which cuts two ways.**

- **Not for the kernel.** keel's microkernel is register-blocked: `MR`, `NR` and
  `Lanes = 16` are compile-time constants because the accumulator tile has to be
  a fixed number of named vector registers, which is the entire content of
  DESIGN.md §3's tile shaping and of the P2 spill audit. A type whose width is
  only known at runtime cannot express that tile. There is also no `GetLo`/`GetHi`
  on `Float32s`, so `HSum`'s specified fold tree — the one the differential test
  demands bit-for-bit — is not expressible portably either. The kernel stays on
  `archsimd`.
- **Possibly for the dev host, which is the interesting half.** T1 is that the dev
  host (darwin/arm64) can execute *no* vector op, so every vector correctness
  check has to travel to an amd64 host over ssh, and the scalar twin is all that
  runs locally. An emulated-plus-arm64 portable package is the first thing that
  could run a vector path *on the dev host* — not for numbers, which stay
  hardware-keyed, but for the differential tests, where the whole point is
  agreement rather than throughput.

That second possibility is a design question, not a finding, and it is not decided
here: it would mean a third backend behind the shim, with its own differential
test obligations and its own bit-exactness argument (an emulated FMA that fuses
where hardware does not would be a *worse* oracle than no oracle). Filed as a
question rather than acted on.

---

<a name="t25" id="t25"></a>
## T25 — the bounds check on a strided slice-expression loop depends on how the *same* bound is spelled, and a reslice moves the check rather than removing it

**Toolchain.** `go1.26.6 darwin/arm64` (host compiler), and confirmed
byte-for-byte identical on `go1.26.5 linux/amd64` and `go1.27rc3 linux/amd64`.
Not simd-specific: the repro imports nothing. Found while writing #22's candidate
C, whose inner loop is one vector add and three memory ops, so a surviving
per-iteration check is a double-digit percentage of the body.

**Relation to T19, which owns half of this.** T19 recorded that a loop guarded by
`i+4 <= len(x)` does not discharge the sub-slice bounds, and prescribed the
slice-advancing rewrite (`x = x[4:]`), whose own cost T19 then measured — seven
instructions, or one with `>` instead of `>=`. That prescription was adopted in
all ten `internal/l1` loops. **T19 missed a second remedy**: the indexed loop is
clean too if the identical bound is spelled `i < len(x)-3` instead of
`i+4 <= len(x)`. Nothing else changes — same arithmetic, same iteration set, same
body. So a keel loop that wants to stay indexed never had to advance its slices.

**Observation, as a table of measured variants.** Each row is a loop with stride
16 over 16-element sub-slices; the last column is `-gcflags=-d=ssa/check_bce`
output for the slice expressions *in the loop body*.

| # | guard | slices touched | checks left in body |
|---|---|---|---|
| B | `j+16 <= len(dst)` | dst | **1** |
| G | `j < len(dst)-15` | dst | 0 |
| O | `n := len(dst)`, `j < n-15` | dst | 0 |
| E | `j+16 <= len(dst)`, element access `dst[j+15]` | dst | **1** |
| H | `j+16 <= len(dst)`, three-index `dst[j:j+16:j+16]` | dst | **1** |
| I | `c < len(dst)/16`, `dst[c*16:c*16+16]` | dst | **1** |
| J | unsigned `j`, `j+16 <= uint(len(dst))` | dst | **1** |
| F | `for len(dst) >= 16 { dst[:16]; dst = dst[16:] }` | dst | 0 — T19's remedy |
| M | `j < len(dst)-15 && j < len(src)-15` | dst, src | 0 |
| K | `dst = dst[:len(src)]`, `j < len(dst)-15` | dst, src | **1 — on dst** |
| L | `src = src[:len(dst)]`, `j < len(dst)-15` | dst, src | **1 — on src** |
| P | `if len(dst) != len(src) { return }`, `j < len(src)-15` | dst, src | **1 — on dst** |
| R | as P with `j+16 <= len(src)` | dst, src | **1 — on dst** |
| T | `lim := min(len(dst), len(src)) - 15`, `j < lim` | dst, src | **2 — both** |
| U | `lim := len(src)-15`, `j < lim && j < len(dst)-15` | dst, src | 0 |

Three things in that table are worth stating separately.

1. **A reslice moves the check to the operand it resliced** (K vs L). `dst =
   dst[:len(src)]` is the idiom for establishing that two slices have the same
   length, and after it the *other* slice verifies clean while the resliced one
   does not — swap which one is resliced and the surviving check swaps with it.
   The reslice is not merely insufficient; it is where the check ends up.
2. **An explicit length comparison does not help either** (P, R). `if len(dst) !=
   len(src) { return }` is the exact shape [CL 699155](https://go-review.googlesource.com/c/go/+/699155)
   (`519ae51`, closing [golang/go#75144](https://github.com/golang/go/issues/75144))
   taught `prove` to propagate, and that commit is an ancestor of both
   `release-branch.go1.26` and `release-branch.go1.27`. It does not reach this
   loop shape: the CL's own case is element indexing under `for i := range
   len(a)`, and a strided slice-expression loop still keeps one check.
3. **Hoisting the loop-invariant limit is only free once** (U vs T). `len(src)-15`
   is loop-invariant and is recomputed every iteration — LICM does not lift it,
   a scalar-integer sibling of T18. Hoisting it into a local keeps every check
   eliminated and saves an `LEAQ` per iteration. Hoisting *both* limits into one
   `min` local puts **both** checks back. The second `LEAQ` is therefore the
   price of the elimination, and keel pays it deliberately.

**Repro.** Fifteen functions, one file, no imports; the variants above are
verbatim. `sink` is a package-level `[]float32` so the slice expressions are not
dead.

```go
package bce

var sink []float32

func B(dst []float32) {                       // 1 check
	for j := 0; j+16 <= len(dst); j += 16 {
		sink = dst[j : j+16]
	}
}

func G(dst []float32) {                       // 0 checks — same iteration set
	for j := 0; j < len(dst)-15; j += 16 {
		sink = dst[j : j+16]
	}
}

func K(dst, src []float32) {                  // 1 check, and it is on dst
	dst = dst[:len(src)]
	for j := 0; j < len(dst)-15; j += 16 {
		sink = dst[j : j+16]
		sink = src[j : j+16]
	}
}

func U(dst, src []float32) {                  // 0 checks — the shape keel ships
	lim := len(src) - 15
	for j := 0; j < lim && j < len(dst)-15; j += 16 {
		sink = dst[j : j+16]
		sink = src[j : j+16]
	}
}
```

```
$ go build -gcflags=-d=ssa/check_bce ./...
./a.go:6:13:  Found IsSliceInBounds        <- B
./a.go:19:13: Found IsSliceInBounds        <- K, on dst
                                           <- G and U print nothing
```

**A fourth finding, and the largest of them: the slice-form intrinsic carries its
own length check, and only a statically-sized slice expression folds it away.**
`Load512(x[j:])` and `Load512(x[j:j+Lanes])` are the same load. The first keeps
two things per iteration that the second does not: `archsimd`'s internal
`CMPQ …, $16` / `JCS →panic` (`slice_gen_amd64.go:291`, the wrapper's own "is this
slice at least 16 long" test), and T19's five-instruction conditional pointer
advance — twice, once per operand, because an open-ended slice expression produces
a slice *value* whose data pointer must not be allowed to pass the end of its
allocation. Measured on `AddRow512`, holding the body and the iteration set fixed
and varying only how the four slice expressions and the guard are spelled:

| spelling | checks in file | loop body |
|---|---|---|
| `j+Lanes <= n`, `dst[j:]` — the natural way to write it | 7 | **36 instructions** |
| `j+Lanes <= n`, `dst[j:j+Lanes]` | 4 | 16 |
| `j < len(dst)-Lanes+1 && j < len(src)-Lanes+1` | 3 | 15 |
| the same with `len(src)-Lanes+1` hoisted — variant U, shipped | 3 | **13** |

Three of the file's checks are irreducible and are the same in every row
(`AddTile512`'s two row slices, and the masked tail's store); the differences are
all in the loop. So the add-back's steady-state body is **2.8× larger written the
obvious way**, for 16 elements of `C += tile` either way, with no difference in
semantics — and `internal/l1` was already immune to the largest part of it, having
used `x[0:16]` from the start.

**What keel does.** `internal/vec.AddRow512` takes the last row: 13 instructions
per 16 elements, no bounds check and no panic block in the loop. Two of the 13 are
`XCHGL AX, AX` — T9's NOP per inlined non-intrinsic call, one each for `Load512`
and `Store512` — and the load of `src` is folded into the add
(`VADDPS (R11), Z0, Z0`), which the source does not show.

**Upstream.** Nothing is filed and nothing should be. The class is known and
open: [#17370](https://github.com/golang/go/issues/17370) is the umbrella,
[#25197](https://github.com/golang/go/issues/25197) is arithmetic in the guard
(with `martisch`'s overflow argument for why the natural form is not a trivial
miss), [#28941](https://github.com/golang/go/issues/28941) is `prove` and slice
expressions, and [#80146](https://github.com/golang/go/issues/80146) (eSSA for
`prove`) is the general fix in flight. The two asymmetries above are not on any
of them, but the standing rule is that a second repro of a known miss is not
worth an upstream comment — a *measured delta on a real kernel* would be, and
keel has a static instruction count and not a time. 36→13 is large enough that
it is hard to see it costing nothing, but a bounds check and a guard comparison
are both a well-predicted compare-and-branch, and this loop is memory-bound: the
36-instruction body may be entirely hidden behind the stores. Tracked as keel #74,
which is where the timing would land, and only that would be worth an upstream
comment.

<a name="t26" id="t26"></a>
## T26 — benchmark output precision is chosen by a value's magnitude, so a rate and its reciprocal from one run differ 42× in resolution

**Observation.** `testing.prettyPrint` selects decimals from `math.Abs(x)`, not from
the precision of the measurement: under `999.95` a column gets four significant
figures, at or over it every integer digit and none after the point. Both the
built-in `ns/op` and every `ReportMetric` column of one row go through it, so which
side of that boundary a quantity happens to land on sets its resolution.

Minimal repro. keel's peak series at 245 GFLOP/s and the same measurement's ns/op,
plus the boundary either side:

```
$ cat m_test.go
package metricprobe

import "testing"

func BenchmarkPrecision(b *testing.B) {
	for b.Loop() {
	}
	b.ReportMetric(245.14999, "GFLOP/s")   // keel's peak series, run 1, keel-gnr
	b.ReportMetric(102762.4, "nsish/op")   // the same measurement's ns/op
	b.ReportMetric(999.94999, "under1000")
	b.ReportMetric(1000.4999, "over1000")
}

$ go version
go version go1.27.0 darwin/arm64
$ go test -run x -bench Precision -benchtime 10x
goos: darwin
goarch: arm64
pkg: metricprobe
cpu: Apple M4 Pro
BenchmarkPrecision-12    	      10	        20.90 ns/op	       245.1 GFLOP/s	    102762 nsish/op	      1000 over1000	       999.9 under1000
PASS
ok  	metricprobe	0.256s
```

`245.1` is a 0.1 quantum, 0.041% relative; `102762` is a 1-unit quantum, 0.00097%.
One run, a rate and its reciprocal, **42× apart** — and `1000.4999 → 1000` against
`999.94999 → 999.9` shows the discriminator is the magnitude and nothing else.

**What changed in the tree.** §5 rule 5's clock series judged the `GFLOP/s` column,
where the 0.05 median quantum is larger than any decline it ever reported; it now
reads `sec/op` at `tools/benchci`'s full float64 and gates on a measured
resolution floor (`clock_series`, `scripts/bench.sh`; six controls in
`scripts/roofline-test.sh`; ruling on #6, 2026-08-20). Nothing else here judged a
sub-1000 column: `bench_describe`'s own `%.4g` is display-only.

**Upstream.** Nothing to file — this is *accepted design*, and recently made so.
[#34626](https://github.com/golang/go/issues/34626) proposed moving statistics
into `go test` precisely because "the `prettyPrint` output causes loss of precision
in any tool that computes the statistics based on the output"; it was redirected to
printing more digits and completed as
[CL 267102](https://go-review.googlesource.com/c/go/+/267102), *"testing: increase
benchmark output to four significant figures"* — so four is the raised value, up
from three. That thread also anticipates this exact failure: rsc, arguing against a
further digit, wrote "I'm skeptical that the 0.05 ns/op has any accuracy behind
it". keel's phantom throttle verdicts were 0.05 GFLOP/s steps, which is his point
one column over and in agreement with it. A fifth digit is not the remedy anyway —
the precision was already in the same row.

## T27 (#117) — the ternlog rewrite transposes `AndNot`'s operands, so any fused expression containing one gets the truth table for `y &^ x`

**Observation.** `ssa.rewriteTern` contracts a tree of vector logical ops into one
`VPTERNLOGD` and builds the imm8 in `computeTT`, whose `sloAndNot` case reads
`Args[0]` as the non-negated operand. AMD64's `VPANDND` carries the negated one
there, so the immediate comes out for `Args[1] &^ Args[0]`. `AndNot` is the only
non-commutative op in that switch and so the only one affected; a lone `AndNot`
is left alone and is correct, and two ops in one tree is the whole trigger — no
loop and no partial load are needed. **Present in go1.26.5 and go1.27.0 alike**,
identical immediates, so it is not a regression of either.

How it reached a shipped keel routine, which is a second fact: go1.27.0 reimplemented
`LoadFloat32x16Part` as `LoadFloat32x16Array(...).Masked(mask)`, and `Masked` is an
explicit vector `And`. That second logical op is what made `Sasum`'s
`LoadPart512` + abs tail fusable, where go1.26.x had left one unfused `AndNot`. The
compiler bug is old; the library change walked keel into it, and the tail returned
`-0` in all sixteen lanes.

Minimal repro. The inputs are the ternlog truth-table basis, so each result byte is
the imm8 the hardware actually got and no arm can pass by expecting zero:

```
$ cat main.go
// rewriteTern computes the wrong truth table for AndNot: it treats Args[0] as the
// NON-negated operand, but AMD64's VPANDND carries the negated one there. No loop
// and no partial load are needed — only a two-op logical tree for the pass to fuse.
package main

import (
	"fmt"
	"simd/archsimd"
)

//go:noinline
func andThenAndNot(a, b, c archsimd.Int32x16) archsimd.Int32x16 { return a.And(b).AndNot(c) }

//go:noinline
func plainAndNot(a, c archsimd.Int32x16) archsimd.Int32x16 { return a.AndNot(c) }

//go:noinline
func orThenAndNot(a, b, c archsimd.Int32x16) archsimd.Int32x16 { return a.Or(b).AndNot(c) }

//go:noinline
func andNotThenAnd(a, b, c archsimd.Int32x16) archsimd.Int32x16 { return a.AndNot(b).And(c) }

func lane0(v archsimd.Int32x16) int32 {
	out := make([]int32, 16)
	v.Store(out)
	return out[0]
}

// The inputs are the ternlog truth-table basis, so every one of the eight
// (a,b,c) bit-triples occurs in each byte and the result byte IS the imm8 the
// hardware was given. got != want therefore names the wrong immediate directly,
// and no arm can pass by having a zero expected value.
const (
	ta = uint32(0xf0f0f0f0)
	tb = uint32(0xcccccccc)
	tc = uint32(0xaaaaaaaa)
)

func i32(u uint32) int32 { return int32(u) }

func row(name string, got, want uint32) {
	flag := "ok"
	if got != want {
		flag = "WRONG"
	}
	fmt.Printf("  %-10s imm8 used = 0x%02x   want 0x%02x   %s\n", name, got&0xff, want&0xff, flag)
}

func main() {
	a := archsimd.BroadcastInt32x16(i32(ta))
	b := archsimd.BroadcastInt32x16(i32(tb))
	c := archsimd.BroadcastInt32x16(i32(tc))
	row("(a&b)&^c", uint32(lane0(andThenAndNot(a, b, c))), (ta&tb)&^tc)
	row("a&^c", uint32(lane0(plainAndNot(a, c))), ta&^tc)
	row("(a|b)&^c", uint32(lane0(orThenAndNot(a, b, c))), (ta|tb)&^tc)
	row("(a&^b)&c", uint32(lane0(andNotThenAnd(a, b, c))), (ta&^tb)&tc)
}

$ go version
go version go1.27.0 darwin/arm64
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOEXPERIMENT=simd go build -o tern-min .
$ ./tern-min                          # Intel i9-9960X (Skylake-X)
  (a&b)&^c   imm8 used = 0x2a   want 0x40   WRONG
  a&^c       imm8 used = 0x50   want 0x50   ok
  (a|b)&^c   imm8 used = 0x02   want 0x54   WRONG
  (a&^b)&c   imm8 used = 0x08   want 0x20   WRONG

$ ./tern-min                          # AMD RYZEN AI MAX+ 395 (Zen 5): identical
  (a&b)&^c   imm8 used = 0x2a   want 0x40   WRONG
  a&^c       imm8 used = 0x50   want 0x50   ok
  (a|b)&^c   imm8 used = 0x02   want 0x54   WRONG
  (a&^b)&c   imm8 used = 0x08   want 0x20   WRONG

$ GOTOOLCHAIN=local go version        # on the host, with `go 1.26` in go.mod and
go version go1.26.5 linux/amd64       # lane0 using 1.26's Store(&arr) spelling (T23)
$ GOEXPERIMENT=simd go build -o tm126 . && ./tm126
  (a&b)&^c   imm8 used = 0x2a   want 0x40   WRONG
  a&^c       imm8 used = 0x50   want 0x50   ok
  (a|b)&^c   imm8 used = 0x02   want 0x54   WRONG
  (a&^b)&c   imm8 used = 0x08   want 0x20   WRONG
```

Every wrong immediate above is `Args[1] &^ Args[0]` evaluated on the same inputs:
`0xAA &^ 0xC0 = 0x2a`, `0xAA &^ 0xFC = 0x02`, `(0xCC &^ 0xF0) & 0xAA = 0x08`. keel's
own `0x70` is the fourth: `absmask &^ (data & lanemask)` = `0xF0 & 0x77`.

**What changed in the tree.** `AbsWith512`/`AbsWith256` spell abs as `And` against
the complement mask, which `AbsMask512`/`AbsMask256` now build
(`internal/vec/vec_avx2.go`, `vec_avx512.go`). The property relied on is the
transposition's own harmlessness under commutativity, not that AND happens to work:
the fused immediate for three ANDs is `0x80`, bit 7 alone, invariant under every
permutation of the inputs. Verified in the shipped kernel — 9 `VPANDND` became 9
`VPANDD` at identical displacements and the one ternlog kept its slot with `$112`
becoming `$128`, so no rate is re-measured. The 256-bit twin was never wrong (it
compiles to a plain `VPANDN`; the pass does not fuse it here) and moves anyway, so
that correctness stops resting on a fusion continuing not to fire.

**Upstream.** No report of a wrong immediate; searched before filing. Two open
issues cover the *same pass* under-triggering:
[#79666](https://github.com/golang/go/issues/79666) "TernLog rewrite does not trigger
for unsigned vectors" and
[#79767](https://github.com/golang/go/issues/79767) "`Not()` is not recognized in
`rewriteTern()`". #79666 is not just precedent, it is the explanation of a control
arm: `archsimd`'s own `Float32x16.Abs()` is `ToBits().AndNot(...)` on `Uint32x16`
and measured correct, because the rewrite skips unsigned vectors entirely. The
closed [#80140](https://github.com/golang/go/issues/80140) / [CL
794680](https://go-review.googlesource.com/c/go/+/794680) "repair mask
optimizations" is the nearest fixed neighbour. Filing, the refuted first story ("go1.27.0 swaps
`VPANDND`'s operands") and two more retracted claims are on
[#117](https://github.com/scttfrdmn/keel/issues/117).

## T28 (#18) — go1.27.0 offers X16–X31 to the SIMD allocator, and `Kernel6x32`'s stack traffic halved

**Observation.** T10/#18 recorded that `fpRegMaskAMD64` withholds X16–X31 from every SIMD
value, so 15 of 32 vector registers are allocatable. On go1.27.0 `ssa/regalloc.go:785`
unions **four** masks and `specialRegMaskAMD64 = 71776114766249984` supplies X16–X31 plus
K1–K7, so **31 of 32 are allocatable** — only X15, the zero register, is in no mask.
`Kernel6x32` uses 23 and its steady-state stack refs fell 90 → 44.

Repro — independent register-only FMA chains, N a parameter, distinct starts so CSE
cannot merge them:

```sh
mkdir -p build/n80828 && cd build/n80828
gen() { n=$1; { echo 'package repro'; echo; echo 'import "simd/archsimd"'; echo;
  echo '//go:noinline'; echo "func Chains$n(x []float32, k int) archsimd.Float32x16 {";
  echo '	v := archsimd.LoadFloat32x16(x)';
  i=0; while [ $i -lt $n ]; do echo "	a$i := archsimd.LoadFloat32x16(x[$((16*i)):])"; i=$((i+1)); done
  echo '	for j := 0; j < k; j++ {';
  i=0; while [ $i -lt $n ]; do echo "		a$i = v.MulAdd(v, a$i)"; i=$((i+1)); done
  echo '	}'; echo '	s := a0';
  i=1; while [ $i -lt $n ]; do echo "	s = s.Add(a$i)"; i=$((i+1)); done
  echo '	return s'; echo '}'; } > chains$n.go; }
gen 13; gen 14; cd ../..
go run ./internal/spill/cmd/spill-audit -pkg ./build/n80828 -func Chains13
go run ./internal/spill/cmd/spill-audit -pkg ./build/n80828 -func Chains14
rm -rf build/n80828   # required: see below
```

**`rm -rf build/n80828` is not tidying.** `build/` is gitignored but `go build ./...`
still walks it, and this package cannot compile natively on darwin/arm64 (`undefined:
archsimd.Float32x16` — the build tags exclude `simd/archsimd`). Leaving it in place turns
`make build` *and* `make stock` red on a clean tree, which is a false red of exactly the
kind the session-start smoke build exists to rule out.

```
github.com/scttfrdmn/keel/build/n80828.Chains13: steady-state loop [744,995] 43 insns for 13 arith (3.31 per arith): 0 vector stack refs, 27 reg copies, 0 broadcasts, 0 anchor nops, 0 calls, 0 bounds-check exits, 0 other mem refs
github.com/scttfrdmn/keel/build/n80828.Chains14: steady-state loop [830,1091] 45 insns for 14 arith (3.21 per arith): 2 vector stack refs, 26 reg copies, 0 broadcasts, 0 anchor nops, 0 calls, 0 bounds-check exits, 0 other mem refs
```

Swept 2026-08-30, `spill-audit` for the counts and `-gcflags=-S` for the highest `Z`
named in the function; the go1.26.5 columns are `golang/go#80828`'s own table, quoted, since no
1.26.5 toolchain is installed:

| N | insns 1.26.5→1.27.0 | stack refs | reg copies | highest Z |
|---|---|---|---|---|
| 13 | 43 → 43 | 0 → 0 | 27 → 27 | `Z14` |
| 14 | 45 → 45 | 14 → **2** | 14 → **26** | `Z16` |
| 15 | 51 → 51 | 19 → **3** | 14 → **30** | `Z17` |
| 16 | 55 → 54 | 24 → **3** | 12 → **32** | `Z18` |
| 20 | 79 → 68 | 44 → **9** | 12 → **36** | `Z23` |
| 24 | — → 82 | — → 15 | — → 40 | `Z29` |
| 31 | — → 107 | — → 43 | — → 30 | `Z31` |

The frontier is **still 13**, as #18 measured, and **the spills came back as
register-to-register copies almost one for one** — exactly so at N=14 (−12 refs, +12
copies, net zero instructions) and N=15 (−16, +16, zero), 1 instruction at N=16, 11 of
79 at N=20. Two controls: N=13 reproduces all three go1.26.5 columns exactly, and every
row on both toolchains closes as `insns = N + refs + copies + 3`. Every copy in the
N=13 and N=14 loops is a rescue or a rotation forced by `VFMADD213PS`, none is a
broadcast, and no FMA in either loop writes its own addend — the classification, and
`Kernel6x32`'s 15 unattributed copies as its positive control, are in
`docs/upstream-plan.md`. Identical under
`GOAMD64` v1, v3 and v4 — **CL 767380, whose title gates on v4, was abandoned
2026-04-17; the fix that landed is CL 768262, unconditional, merged to `dev.simd`
2026-04-23 against `golang/go#78753`.** So the invariance confirms the merged CL rather
than refuting live framing, and `golang/go#80828` is a duplicate of `golang/go#78753`, which was
already closed when keel filed it.

**What changed in the tree.** Nothing executable. Three sites carried #18's cause as
settled and are corrected: `DESIGN.md` §4/P2, `Kernel6x32`'s doc comment in
`internal/vec/gemm_amd64.go`, and `docs/spill-report.md`, which gains §11 with the
decomposition of the 44 — 36 are broadcast-scalar round-trips through legacy-SSE `MOVUPS`
in the inlined wrapper at `archsimd/other_gen_amd64.go:265`, and only 3 of the 12
accumulators spill, so the residual is not register pressure.

**Not established (§5 rule 12).** No go1.26.5 toolchain is installed on the dev host, so
every 1.26.5 figure here is #18's quotation, not a re-run: this is measurement against
quotation, not two measurements. The repro is not byte-identical to #18's either — its
`LoadFloat32x16Slice` no longer exists (T23) — so what licenses the comparison is the
N=13 row and the closing identity, not the repro's provenance. Nothing here is timed.

**Upstream.** Answered on `golang/go#80828` 2026-08-30 with the table above, the
duplicate disclosure, and one open question: `simdRegMaskAMD64` is **still** `2147418112`
(X0–X14) and `compatRegs` intersects with it for SIMD values wider than 8 bytes, yet the
listing allocates `Z16`–`Z23` including plain copies and one reload — so the path that
reaches those registers is not the one #18 read, and keel has not identified it. Why #18
was right when written, and why the residual is `golang/go#80835`'s subject rather than a
new filing, are on [#18](https://github.com/scttfrdmn/keel/issues/18) and in
`docs/upstream-plan.md`.
