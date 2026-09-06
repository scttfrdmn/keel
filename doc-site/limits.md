# Capabilities & limits

## What keel does

- **float32 only.** `Sdot`, `Saxpy`, `Sscal`, `Sasum`, `Snrm2`, `Isamax`,
  `Sgemv`, `Sger`, `Sgemm`, `Ssyrk`, `Ssymm`, `Strsm`.
- **Row-major storage only**, with a leading dimension per matrix. `lda > n` is
  supported, so submatrices need no copy.
- **AVX-512 kernels on amd64 and NEON kernels on arm64**, written in Go, chosen
  at process start from CPU feature detection. Both ISAs ship Level 1 *and*
  Level 3; the microkernel shape is ranked per host by its audited
  instructions-per-FMA and memory-ops-per-FMA, not by registry order (on arm64
  that ranking picks the 4×16 NEON tile, measured +33% over 8×8 at full Sgemm —
  see the [cross-ISA positioning](https://github.com/scttfrdmn/keel/blob/main/docs/cross-isa-positioning.md)
  and the [simd experience report](https://github.com/scttfrdmn/keel/blob/main/docs/simd-experience-report.md)).
- **A scalar path that builds anywhere**, with the same results.
- **Parallel Level 3**, sized by `GOMAXPROCS`, bit-identical at every thread
  count.
- **Pure Go**: no cgo, no assembly, no code generation, no `init` that spawns
  anything, no goroutine that outlives a call.

## What keel will not do

These are commitments, not gaps. Each one follows from what keel is for, and a
version that reversed it would be a different library.

- **No column-major option.** There is no `order` argument. Transpose flags cover
  the cases that matter; a genuinely column-major matrix has to be transposed by
  the caller. keel is not a translation of reference BLAS, so there is no order
  argument to get wrong.
- **No `Isamax` vector kernel.** Reference BLAS defines it by a sequential
  comparison chain, which gives NaN a specific behaviour — a NaN in the first
  position wins, a NaN anywhere else is skipped — that a lane-parallel maximum
  cannot reproduce, because `VMAXPS` resolves NaN by operand position rather than
  by sequence. Matching the convention is worth more than the speed of a routine
  with no arithmetic in it.
- **Not the whole of BLAS.** Level 2 is `Sgemv` and `Sger`; there is no `Strmv`,
  `Ssymv`, `Ssyr` or `Ssyr2`. Level 3 is `Sgemm` and three routines derived from
  it; there is no `Strmm` or `Ssyr2k`. "Subset" is in the first sentence of the
  design document, and `Sgemm` is where the effort concentrates on purpose.
- **No error returns.** Invalid arguments panic. There is nothing to check a
  status code against, because in Go there is no XERBLA to consult afterwards and
  a silently-returning no-op looks like success.

## What keel does not do yet — or may never do

Split from the list above because the two are different claims, and reading a
schedule as a promise or a commitment as an oversight both mislead. Nothing here
is a delivery date: the roadmapped items have owners and issue numbers, which is
all that distinguishes them from the parked ones.

*(The arm64 NEON path was here as "scheduled" through v0.1.0. It shipped — Level 3
[#136](https://github.com/scttfrdmn/keel/issues/136), Level 1
[#154](https://github.com/scttfrdmn/keel/issues/154) — so it moved up to "What keel does" above.
The tiers are structured so the next shipped feature is a one-line move between them, not a
rewrite.)*

**Roadmapped — scheduled, not written.**

- **No AVX2 Level-3 microkernel.** Level 1 has an AVX2 backend; Level 3 does not, so a
  machine with AVX2 but no AVX-512 gets a vector Level 1 and a scalar Level 3.
  Tracked as [#40](https://github.com/scttfrdmn/keel/issues/40).

**Upstream-gated — keel cannot build these until the `simd` toolchain exposes them, and may never.**

Distinct from roadmapped: these are not unwritten, they are *unbuildable in pure Go today*. The
gate is the experimental `archsimd` surface, not keel's schedule.

- **No SVE path on arm64.** keel's arm64 kernels are NEON — 128-bit, four float32 lanes.
  `archsimd` exposes no SVE intrinsics under any name
  ([golang/go#79781](https://github.com/golang/go/issues/79781), open), so a pure-Go SVE kernel
  cannot be written. It matters: on Graviton3, OpenBLAS's V1 SVE kernel runs ~24% faster than its
  NEON kernel ([the SVE≈NEON measurement](https://github.com/scttfrdmn/keel/blob/main/docs/graviton-sve-neon.md)),
  and that margin is unreachable from Go — not because keel's NEON kernel is weak but because the
  wider ISA is not on the API surface. Tracked as
  [#162](https://github.com/scttfrdmn/keel/issues/162).
- **No float16 or bfloat16.** No such element type exists in the `simd` packages. If one appears,
  the shape worth building is a *storage* variant — load reduced-precision, widen, accumulate in
  float32, narrow on store — cheaper than float64 and needing no new oracle. Conditional, no
  schedule: [#139](https://github.com/scttfrdmn/keel/issues/139).

Two NEON *lowering* improvements are also upstream rather than keel's to write: the emulated
`BroadcastFloat32x4` costs two dead instructions per broadcast, and a fix is in flight
([golang/go#81352](https://github.com/golang/go/issues/81352) / CL 827904, keel tracks it at
[#163](https://github.com/scttfrdmn/keel/issues/163)) — when it lands the NEON kernels' dead
broadcast copies disappear with no keel change.

**Open question — not a plan either way.**

- **No float64.** Use [Gonum](https://www.gonum.org/): it is mature, and float64
  was never this project's gap to fill. Whether that stays permanent is
  explicitly open ([#103](https://github.com/scttfrdmn/keel/issues/103)), and the
  finding recorded there is that the cost is mispriced by everyone who guesses at
  it. The blocker is not the kernels — the float64 simd surface is already
  symmetric with the float32 one, and the ISA-agnostic middle seam a second element
  type would reuse now exists, cut for the arm64 port
  ([#135](https://github.com/scttfrdmn/keel/issues/135)), so if float64 is ever
  pursued it is post-freeze work on an existing seam, not a rewrite. The gate is
  the **oracle**: keel's whole correctness argument is testing float32 against a
  wider type, and a float64 reference for a float64 routine is the same precision
  as the thing under test, so it validates nothing about accumulation order. That
  is unchanged by the seam, and it is the actual open question.

**Parked — no owner, no schedule, and may never happen.**

- **No complex types.** Neither `complex64` nor `complex128`; no `C`- or
  `Z`-prefixed routines.
- **No packed, banded or triangular-packed storage.** Every matrix is a dense
  rectangle plus a leading dimension.
- **No int8 / VNNI path.** The AVX-512 VNNI int8 dot product exists on some of
  keel's own development hardware and is unused. Quantized inference is a
  different library's problem.
- **No auto-tuning.** Blocking parameters are fixed; the microkernel *shape* is
  chosen per host from a feature-bundle classification, not by measuring at
  startup. Parked rather than committed: a measured selection is defensible, it
  is just not what ships.

## Requirements

| | requirement |
| --- | --- |
| Go | 1.26 or newer for the scalar path; **1.27 or newer** for the vector path |
| scalar path | any Go toolchain, any platform Go supports |
| vector path, build | `GOEXPERIMENT=simd`, **amd64 or arm64**, Go 1.27+ |
| vector path, run | amd64: AVX-512 for Level 3, AVX2 or AVX-512 for Level 1. arm64: NEON for both levels |

The vector path depends on `simd`/`archsimd`, which are **experimental** and not
covered by Go's compatibility promise. They have already had breaking renames
between releases, and a toolchain upgrade can therefore break the vector build
without breaking anything else.

!!! warning "As of 2026-08-28: the vector path needs Go 1.27, and no longer builds on 1.26"

    `archsimd` swapped its load and store names between 1.26 and 1.27 — the slice
    forms took over the bare names and the array forms gained an `Array` suffix —
    and keel is now written against the 1.27 spelling
    ([issue #69](https://github.com/scttfrdmn/keel/issues/69), landed). The swap
    admits no source that satisfies both, so this is a floor and not a preference:
    on 1.26, `GOEXPERIMENT=simd go build ./...` fails to compile. The scalar path is
    unaffected and still builds on 1.26. This warning has now pointed *both*
    directions in twelve days, which is the experimental-package risk itself rather
    than a fact about either release.

## Maturity

**v0.1.0, tagged 2026-08-29.** All six phase gates P0–P5 are green — the release
is certified by a `gate-p5` run on a three-host AWS fleet at `72 PASS / 0 FAIL /
0 UNMEASURED`, with the certified rev, its log, and the two-commit seam between
that rev and the tag all named in the
[release notes](https://github.com/scttfrdmn/keel/releases/tag/v0.1.0).

Major is still 0, so a minor bump may break the API and the module carries no
compatibility guarantee yet. Two shortfalls are known and open rather than
hidden: Sapphire Rapids reads 34.2% of measured peak with 55% unreachable until
an upstream lowering lands
([#104](https://github.com/scttfrdmn/keel/issues/104)), and Granite Rapids falls
short of the `gate-p3` OpenBLAS criterion
([#111](https://github.com/scttfrdmn/keel/issues/111)) — both machines are
therefore characterization hosts, not certified ones.

**Since v0.1.0 (toward v0.2.0):** the arm64 NEON path landed — Level 3
([#136](https://github.com/scttfrdmn/keel/issues/136)) and Level 1
([#154](https://github.com/scttfrdmn/keel/issues/154)), both differentially tested and
spill-audited, with the microkernel ranked by measured instructions-per-FMA (the 4×16 tile, +33%
over 8×8 at full Sgemm). keel took its first judged arm64 verdicts on rented Graviton3/Graviton4
([#137](https://github.com/scttfrdmn/keel/issues/137)); those judged numbers used the 8×8 tile and
so understate current keel by ~a third, a clean post-fix re-measure being future work. The
[cross-ISA positioning](https://github.com/scttfrdmn/keel/blob/main/docs/cross-isa-positioning.md)
records what the two ISAs' numbers say together — and why the raw percent-of-OpenBLAS figure tracks
the *reference's* per-µarch coverage rather than keel's kernel quality.

What *is* settled is the testing floor every routine already meets: a float64
oracle per routine, differential agreement across backends on identical inputs,
adversarial shapes and strides, and a scalar path proven by running the whole
suite with the vector backends forced off. See the
[testing methodology](records/methodology.md) for the rules those gates enforce.
