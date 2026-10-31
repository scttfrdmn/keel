<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Experience report for golang/go#73787 (scttfrdmn/keel#128)

**Status: FINAL DRAFT, not posted.** Re-verified 2026-10-31 against the tree and the tracker,
replacing the 2026-09-06 draft (`f1c203c`). Posting is Scott's call. The text between the two
markers below is what would be posted, verbatim.

Each number below was re-derived from an archived log, from `spill-audit` re-run on go1.27.3, or
from a go1.27.3 build of a repro in `docs/toolchain-notes.md`. There are two exceptions. The VZEROUPPER
before/after figures are recorded only in `CHANGELOG.md` [0.2.4] and T30, and their raw logs are
not in the tree. CL 824624 PS2's measurements are not restated: the post points at the CL, where
they were posted, because their logs are not in the tree either. Excluded as stale or unsourced:
the 2026-08 lab percent-of-peak table, the SPR 51.0%-of-OpenBLAS (it predates the VZEROUPPER fix),
the T17 quotation, and the Graviton4 "half of V1" reference, which the judged runs do not reproduce.

<!-- post: begin -->

## Experience report: a pure-Go BLAS on `archsimd`, amd64 and arm64, float32 and float64

[keel](https://github.com/scttfrdmn/keel) implements every routine in Gonum's `blas.Float32` and 34 of 35 in `blas.Float64`.
It's written against `simd/archsimd`, with AVX-512/AVX2 kernels on amd64 and NEON kernels on
arm64. All of its SIMD code is in one package, and each vector op has a scalar twin that it's tested
against. Since amd64 GA moved to Go 1.28 here, this report is about what to settle
before then. Issues that are already tracked are linked, not argued again.

Three tiers of evidence, and each number says which it is. **Judged**: keel's release gate on
full-size on-demand AWS instances, two draws pooled, on go1.27.0. **Exploration**: AWS 4xlarge,
unjudged. **Lab**: a bare-metal i9-9960X in a queued measurement slot, unjudged. Static
instruction counts are from go1.27.3.

### Where it stands (judged)

Single-thread rates. Each peak column divides by keel's own microkernel peak, measured on the same
host in the same run, at n=4096. The OpenBLAS column is 2048³ against OpenBLAS's best swept core
family, for both draws.

| CPU (instance) | Sgemm, of own peak | Dgemm, of own peak | Sgemm / OpenBLAS |
|---|---|---|---|
| Zen 4, EPYC 9R14 (c7a.48xl) | 108.5 GF/s, 92.8% of 117 | 53.72, 91.8% of 58.53 | 95.6% (0.3.26) |
| Zen 5, EPYC 9R45 (c8a.48xl) | 177.8, 61.9% of 287.4 | 86.65, 60.3% of 143.8 | 65.7–66.1% |
| Skylake-SP, Xeon 8124M (c5n.18xl) | 68.72, 32.9% of 208.6 | 33.83, 32.4% of 104.4 | 38.6–39.2% |
| Neoverse-V1, Graviton3 (c7g.16xl) | 58.78, 75.2% of 78.16 | 30.23, 77.4% of 39.06 | 70.5–73.3% (0.3.29) |
| Neoverse-V2, Graviton4 (c8g.48xl) | 65.39, 77.5% of 84.42 | 31.84, 75.4% of 42.22 | 90.2–90.3% (0.3.29) |
| Neoverse-V3, Graviton5 (c9g.48xl) | 80.06, 77.6% of 103.2 | 39.92, 77.4% of 51.6 | 89.0–89.1% (0.3.34) |

float64 tracks float32. At 8 threads, Dgemm's share of 8× peak is within 1.5 points of Sgemm's on
all six CPUs.

### 1. Keep `ClearAVXUpperBits` until the compiler places `VZEROUPPER`, and place it at transitions, not at every return

go1.27.3 emits no `VZEROUPPER` in any Go function ([repro](https://github.com/scttfrdmn/keel/blob/main/docs/toolchain-notes.md)).
Our AVX-512 kernels left dirty upper halves for the scalar legacy-SSE packing code that runs next. On
Skylake-X (lab), that pack ran 2.2× (2×32 kernel) and 3.3× (4×32) slower than when run alone. With one
`ClearAVXUpperBits()` after each kernel's stores, the pack runs at its standalone speed. Right-side
`Strsm` at n=2048 went from 245 to 195 ms and from 254 to 185 ms.

Clearing at every AVX Level-1 exit cost more than it saved. `Sgemv` got 5–7% slower at 64²–256²,
`Sdot` 3% slower at n=256, and large n gained nothing. A `VZEROUPPER` before every return would
therefore slow down short vector calls. What we needed was one at the transition into SSE code.
This issue floated removing `ClearAVXUpperBits` (2026-05-01) or keeping it only under the experiment. For us
it's load-bearing today. The data is on golang/go#80835 (the boundary sub-case).

### 2. `MulAdd` should accumulate in place on amd64, as planned on this thread

In this thread's 2025-08-28 discussion, the plan was for the compiler to choose among the 213/231 forms
"depending on which operand is not live." In go1.27.3, amd64 lowers `MulAdd` only to `VFMADD213PS`,
and `simdAMD64ops.go` has no packed 231 op. So 12 of the 50 steady-state loop instructions in our
shipped 4×32 AVX-512 kernel are register-to-register copies, for 8 FMAs and zero spills. On arm64 the
same Go source lowers to `FMLA` with the accumulator as arg0 (`simdARM64.rules:242`) and has no
accumulator copies. The lowering rule makes the difference, not the ISA.

This also blunted go1.27's register fix. On independent FMA chains, the spills it removed came back
one for one as copies: at 14 chains, 12 fewer stack refs, 12 more copies, and no net change in
instructions. This is golang/go#80829. Our [CL 824624](https://go.dev/cl/824624) PS2 lets regalloc choose the form from
liveness, and the CL has before/after measurements on four x86 microarchitectures. The `.BCST`
embedded-broadcast half comes next.

### 3. On arm64, broadcast is the biggest lever we measured. NEON needs a one-instruction broadcast or a by-element FMA

`BroadcastFloat32x4` is emulated (`SetElem(0, x).broadcast1To4()`) and lowers to 3 vector ops. It
limits our 3×24 NEON GEMM kernel. Pre-broadcasting A during packing makes the A panel 4× larger, and
the same tile ran 43–62% faster on Neoverse V1/V2/V3 (exploration). keel now ships that: a library
paying 4× packed-A memory to route around a lowering. [CL 827904](https://go.dev/cl/827904) (golang/go#81352) folds the broadcast
to a load and a `DUP`. It gave +14.5–36.4% on broadcast-form kernels and brings the 3×24 to 78.5–90.5% of
the pre-broadcast rate (exploration; the A/B is posted on that issue).

C GEMM kernels use a by-element FMA here: `FMLA Vd.4S, Vn.4S, Vm.S[i]`, with one q-load of A feeding
four FMAs. Neither go1.27.3's NEON surface nor the SVE surface on master (2026-10-24) has an indexed
FMA. On SVE, `BroadcastFloat32s` is already a bodiless intrinsic (`DUP`). `BroadcastFloat32x4`, and
`BroadcastFloat32x16` on amd64 (golang/go#80830), are emulated.

### 4. Wrapper anchors are real but cheap, and not an API blocker

When `archsimd` is confined to one package, every inlined non-intrinsic wrapper leaves a
statement-position anchor: a 1-byte `XCHGL AX, AX` on amd64 and a 4-byte `HINT $0` on arm64
(golang/go#80830). In our wrapper-based 3×24 NEON kernel, 63 of the 148 loop instructions are
anchors. We predicted removing them would matter, and it barely did. Calling `archsimd` directly cut
instructions per FMA by 30% (4.111 → 2.889) and gained 0.9–2.6% (exploration). The broadcast (item
3) was the limiter.

### 5. SVE: for FP32 GEMM on cloud arm64, the gain is V1's vector width

In the judged runs, OpenBLAS's core-tuned family beats its generic ARMV8 NEON family by about 23–26%
on Neoverse-V1, which has 256-bit SVE (78.8 vs 62.5–64.1 GF/s). On V3 the gap is 0.1% (86.69 vs 86.62,
OpenBLAS 0.3.34). keel's NEON-only Sgemm reads 70.5–73.3% of OpenBLAS on V1 and 89–90% on V2/V3.
That matches the vector-length table on golang/go#79781: for this kernel the 128-bit SVE parts gain
little over NEON. A NEON surface that is complete for GEMM (item 3) pays off on most of today's
fleet.

### Resolved in go1.27, with thanks

- Z16–Z31 are allocatable (CL 768262). Our 6×32 reference tile's loop stack refs halved, from 90 to
  44. Our golang/go#80828 duplicated golang/go#78753.
- Partial loads and stores pass `checkptr` (CL 761120). Our golang/go#80856 duplicated golang/go#78413.

### What this doesn't show

- We took no hardware-counter measurements. Causes are attributed by A/B and by instruction count.
- Exploration and lab numbers are unjudged single sessions. Only the table above is release-gated.
- Skylake-SP is our weakest host (38.6–39.2% of OpenBLAS). This report doesn't decompose that gap.

Details, repros and logs: [`docs/toolchain-notes.md`](https://github.com/scttfrdmn/keel/blob/main/docs/toolchain-notes.md) and [`archive/`](https://github.com/scttfrdmn/keel/tree/main/archive) in the keel repo.

<!-- post: end -->
