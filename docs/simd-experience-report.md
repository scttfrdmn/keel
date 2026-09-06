<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Experience report: building a float32 BLAS subset on `GOEXPERIMENT=simd`

**Status: DRAFT for posting to golang/go#73787 (#128).** Not yet posted. The graduation
proposal is already `Proposal-Accepted` ("likely accept," aclements 2025-10-15), so this is
framed as a **post-acceptance field report to inform API stabilization** — what a real library
hit, with numbers and denominators — not as an argument for acceptance. Posting is gated on the
in-flight CLs being mailed (the "contributor, not bystander" posture, #125/#126/#127) and on a
human decision about venue and timing, since the thread has been quiet since 2025-10.

Every figure below is re-read from the keel tree at drafting time (2026-09-06) and cited to a
file, an issue, or a run log. Each item carries its own confidence class. Two items are caveats,
and they keep their caveats here.

---

keel is a pure-Go float32 BLAS subset (Level 1/2/3) written entirely against `simd`/`archsimd`,
with every SIMD import confined to one package (`internal/vec`) for auditability and a scalar
twin behind every vector op. It ships AVX-512 microkernels on amd64 and NEON microkernels on
arm64 (both Level-1 and Level-3), and it is judged on rented Graviton and a lab x86 fleet against
source-built OpenBLAS. That gives this report two ISAs of measured experience with the API rather
than one ISA of opinion.

The single most useful thing we can tell an API designer: **several of these findings refuted our
own first guess about their cause.** A wishlist cannot do that; a library under a differential
test and a compile-time audit can.

## 1. The one-package rule pays a per-wrapper anchor NOP — a structural cost of the recommended pattern

**Confidence: high (static, measured on both ISAs, independently filed and being fixed upstream).**

Confining `simd` imports to one package — the auditability pattern — means the hot kernels reach
`archsimd` through thin inlined wrappers. Every inlined non-intrinsic wrapper leaves an anchor
carrying the statement's position in the loop body:

- **amd64** (golang/go#80830, keel #17): a 1-byte `XCHGL AX, AX` per anchor. In the peak kernel
  that is 12 anchors against 12 FMAs.
- **arm64** (keel `docs/neon-probe.md`): a **4-byte `HINT $0`** — fixed-width, so 4× the bytes per
  anchor. It is emitted for the bodied wrapper `LoadFloat32x4` but **not** for the bodiless
  intrinsic `LoadFloat32x4Array`; swapping the four wrapper loads for the intrinsic removes exactly
  four anchors and four instructions. On the ported `4×16` NEON kernel this is 6 anchors (24 bytes)
  per K-iteration against 2 on the shipped amd64 `4×32`.

Why it belongs on a stabilization thread: this cost is a consequence of the **usage pattern the
API invites** (a hot layer of wrappers), not of any one program's mistake. `LoadFloat32x4Array`'s
bodiless lowering shows the fix already exists for one op; generalizing it would make the
one-package rule free.

## 2. `MulAdd` cannot accumulate in place or fold a broadcast operand (amd64), and the ISA contrast is instructive

**Confidence: high. This blocks a keel phase gate.**

On amd64 (golang/go#80829, keel #18/#20) the SSA `MulAdd` lowers to `VFMADD213PS`, whose `arg0` is
a multiplicand, and the compiler emits no `231` form — though the assembler encodes it. The shipped
`Kernel4x32` therefore pays **12 register-to-register moves per 8 FMAs — 24% of a 50-instruction
loop with zero spills** (re-verified this session: `spill-audit -goarch amd64` on `Kernel4x32`
reports 50 loop insns, 8 arith, 12 reg copies). Four preserve broadcasts the `213` form clobbers;
eight rotate accumulators at the loop bottom. keel #104 states 55% of measured peak **unreachable**
on Sapphire Rapids until this lands — an API absence that blocks a gate is a stronger datum than
one that costs percent. CL 824624 addresses it and is in review.

The **ISA contrast** is the part a designer cannot get from the amd64 issue alone: on arm64 the
same SSA op lowers to `VFMLA` with `resultInArg0: true` where `arg0` is the *accumulator*
(`simdARM64.rules`), so the NEON kernel carries **zero** accumulator copies for the same shape.
The instruction set does not settle this on either side (`VFMADD231PS` also writes its own
accumulator and x86 still misses it) — **the lowering rule does.** The lesson for the `simd` API:
the same portable op has very different codegen quality per ISA today, and that gap is a compiler
matter the portable surface hides.

## 3. `checkptr` false positive on partial load/store — RESOLVED, and included because it was

**Confidence: high, resolved (CL merged).**

golang/go#80856 (keel #42): partial-slice load/store helpers tripped `-race`/`checkptr`. Fixed by
**CL 761120** ("mark pa* unsafe helpers as nocheckptr"), **merged**. Listed deliberately: an
experience report that carries only open grievances misrepresents the maintainers' responsiveness.
This one turned around, and partial-vector ops are now usable under `-race`.

## 4. A quotation, labelled as such: the `StackSmall`/`nosplit` figure that was never paid

**Confidence: quotation — a measured cost for a change that became obsolete before it shipped.**

keel T17 (`docs/toolchain-notes.md`): a measured **+15.5% static instructions** in `internal/l1`
if all ten Level-1 kernels crossed `StackSmall` and lost `nosplit`. It **was never paid** —
`vec.LoadPart512` still calls `archsimd.LoadFloat32x16SlicePart` directly, so the crossing never
happened. It appears here only as a labelled quotation of what the wrapper-heavy shape *would*
have cost, not as a live number. (The planning charter had the sign wrong, "−15.5%", and omitted
the never-paid clause; both corrected against the tree.)

## 5. Percent-of-peak, with the denominator stated — and one charter figure excluded pending proof

**Confidence: high (re-read from the tree); one figure excluded.**

Kernel efficiency, as a fraction of each host's own measured FMA ceiling (L1-resident microkernel,
`BenchmarkKernel`, KERNEL.md §7):

| host | best-tile percent-of-peak |
|---|---|
| Zen 4 (vesta) | 96.6% (4×32) |
| Zen 5 (antares) | 64.2% (4×32) |
| Skylake-X (janus) | 46.0% (2×32) |
| GB10 Grace (arm64, NEON 4×16, post-tile-fix) | ~40% (extrapolated; 32.8% at 8×8) |

The full blocked 2048³ GEMM on a full-size `c7i.48xlarge` reaches **51.0%** of same-host OpenBLAS
(101.7 / 199.3 GFLOP/s, `docs/spill-report.md` §10.5), where OpenBLAS itself reaches 85.8% of the
same measured peak — so the gap is the kernel, not the blocking.

**Excluded pending verification:** the charter's "measured 110× µarch spill price" is real but
**off-subject** and must not be posted as a compiler-miss datum — it is a *microarchitecture*
price (`#104`: Zen4 30.5% vs SPR 0.278% of peak on the spilling `Kernel6x32`, = 109.7×), and the
compiler emits the *same* spilled code on every host in that table, so what varies by 110× is the
silicon, not a miss. It stays out of a compiler-directed report.

## 6. SVE ≈ NEON on Graviton — now MEASURED on a judged fleet (upgraded from the charter's caveat)

**Confidence: measured on a judged fleet (the charter listed this "pending reproduction"; it has
since been reproduced).**

On rented Graviton (`docs/graviton-sve-neon.md`, #137, N=2 + confirmation pass), forcing each
OpenBLAS 0.3.29 family at load time, single-thread 2048³:

- **Neoverse-V1 (Graviton3): SVE ≈ NEON is REFUTED.** The V1-tuned SVE families run ~78.8 GFLOP/s
  against ~63.3 for the NEON (`ARMV8`/`NEOVERSEN1`) families — **~24% faster**, a wide margin.
- **Neoverse-V2 (Graviton4): the reference is anomalous.** OpenBLAS 0.3.29 ships **no V2 kernel**,
  so `NEOVERSEV2` falls back to the V1-SVE codepath and every V2 family runs at ~half V1's rate.

Consequence for the `simd` API, and the reason it matters here: **`archsimd` exposes no SVE
intrinsics on arm64** (golang/go#79781, open) and **no lane-indexed FMLA** (`FMLA Vd.4S, Vn.4S,
Vm.S[i]`) under any name. A pure-Go BLAS therefore cannot answer OpenBLAS's V1 SVE kernel — the
~24% is currently unreachable from Go, not because the kernel is bad but because the API surface
stops at fixed-width NEON. That is a concrete stabilization input: **the arm64 surface's ceiling is
NEON until SVE intrinsics land.**

## 7. New: the cross-ISA positioning, and why the quoted ratio misleads

**Confidence: high (synthesis over archived measurements; `docs/cross-isa-positioning.md`).**

The number most likely to be quoted about a Go SIMD library — percent-of-OpenBLAS — moves with the
*reference's* per-µarch kernel coverage, not the Go library's quality. keel's arm64 Sgemm is 31% of
OpenBLAS on Graviton3 and 54% on Graviton4 — but that inverts keel's actual standing: 31% is low
because OpenBLAS ships a *strong* V1 SVE kernel, and 54% is high because OpenBLAS ships *no* V2
kernel. On percent-of-peak (which divides by the host's own ceiling, not a reference), keel's NEON
kernel is a normal member of the cross-ISA spread. The stabilization takeaway: **a Go SIMD library's
headline number is dominated by the reference's coverage; judge the API by percent-of-peak.**

## 8. New: register allocation is clean on arm64; the amd64 question is open

**Confidence: static; one half explicitly retracted and re-opened.**

arm64 puts all 32 vector registers in one allocatable set and the NEON kernels demonstrably use the
whole file (`docs/neon-probe.md` §2d). On amd64, keel #18 (T10) reported "only 15 of 32," and this
report explicitly **retracts** the mask-arithmetic explanation once offered for it — the shipped
`Kernel6x32` allocates `Z16`–`Z23` yet still carries 90 vector stack refs, which is a stranger
question than register starvation and is left open pending an `ssa.html` read. Included as an
honest open item, not a finding.

## What this report cannot see (§5 rule 12)

- **The arm64 numbers are pre- or post-tile-fix as labelled**, and the Graviton perf numbers are
  from `#137`'s fleet, which used the 8×8 tile; keel now ships the faster 4×16 (a +33% dispatch
  fix, keel `972ee47`), so the judged arm64 ratios understate current keel by ~a third. A clean
  post-fix judged re-measure is future work.
- **GB10 (DGX Spark Grace) is characterization-only** — no cpufreq governor, boosts under load, so
  its percent-of-peak denominator is less stable than governed Graviton's. Absolute GB10 rates are
  reported with that caveat; the judged tier is Graviton.
- **Several fixes are in flight, not merged** — CL 824624 (item 2), the LICM CL 803220 (golang/go#79984),
  the broadcast fold CL 827904 (golang/go#81352, item 1). This report describes the state at
  2026-09-06 and links CLs; it does not claim they landed.
- **One toolchain family** (go1.27.x with `GOEXPERIMENT=simd`); the arm64 static column is a
  re-implementation of keel's amd64 loop-auditor's rules, positive-controlled against the tool on
  amd64 but not itself the gate's instrument.
