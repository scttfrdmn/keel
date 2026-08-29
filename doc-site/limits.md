# Capabilities & limits

## What keel does

- **float32 only.** `Sdot`, `Saxpy`, `Sscal`, `Sasum`, `Snrm2`, `Isamax`,
  `Sgemv`, `Sger`, `Sgemm`, `Ssyrk`, `Ssymm`, `Strsm`.
- **Row-major storage only**, with a leading dimension per matrix. `lda > n` is
  supported, so submatrices need no copy.
- **AVX-512 kernels on amd64**, written in Go, chosen at process start from CPU
  feature detection.
- **A scalar path that builds anywhere**, with the same results.
- **Parallel Level 3**, sized by `GOMAXPROCS`, bit-identical at every thread
  count.
- **Pure Go**: no cgo, no assembly, no code generation, no `init` that spawns
  anything, no goroutine that outlives a call.

## What keel does not do

- **No float64.** Not planned. Use [Gonum](https://www.gonum.org/) — it is
  mature, and float64 was never this project's gap to fill.
- **No complex types.** Neither `complex64` nor `complex128`; no `C`- or
  `Z`-prefixed routines.
- **No column-major option.** There is no `order` argument. Transpose flags cover
  the cases that matter; a genuinely column-major matrix has to be transposed by
  the caller.
- **No packed, banded or triangular-packed storage.** Every matrix is a dense
  rectangle plus a leading dimension.
- **No ARM64 vector path.** NEON and SVE are a real intention rather than a
  promise, and they are not written. On arm64 keel builds and runs the scalar
  path.
- **No AVX2 microkernel.** Level 1 has an AVX2 backend; Level 3 does not, so a
  machine with AVX2 but no AVX-512 gets a vector Level 1 and a scalar Level 3.
- **No `Isamax` vector kernel.** Reference BLAS defines it by a sequential
  comparison chain, which gives NaN a specific behaviour — a NaN in the first
  position wins, a NaN anywhere else is skipped — that a lane-parallel maximum
  cannot reproduce, because `VMAXPS` resolves NaN by operand position rather than
  by sequence. Matching the convention is worth more than the speed of a routine
  with no arithmetic in it.
- **Not the whole of BLAS.** Level 2 is `Sgemv` and `Sger`; there is no `Strmv`,
  `Ssymv`, `Ssyr` or `Ssyr2`. Level 3 is `Sgemm` and three routines derived from
  it; there is no `Strmm` or `Ssyr2k`.
- **No error returns.** Invalid arguments panic. There is nothing to check a
  status code against.
- **No auto-tuning.** Blocking parameters are fixed; the microkernel *shape* is
  chosen per host from a feature-bundle classification, not by measuring at
  startup.

## Requirements

| | requirement |
| --- | --- |
| Go | 1.26 or newer for the scalar path; **1.27 or newer** for the vector path |
| scalar path | any Go toolchain, any platform Go supports |
| vector path, build | `GOEXPERIMENT=simd`, amd64, Go 1.27+ |
| vector path, run | AVX-512 for Level 3; AVX2 or AVX-512 for Level 1 |

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

There is no tagged release. As of 2026-08-16, phases P0–P4 of the build plan are
green and P5 (parallelism, dispatch, polish) is in progress, so the API can still
change and the module has no compatibility guarantee yet.

What *is* settled is the testing floor every routine already meets: a float64
oracle per routine, differential agreement across backends on identical inputs,
adversarial shapes and strides, and a scalar path proven by running the whole
suite with the vector backends forced off. See the
[testing methodology](records/methodology.md) for the rules those gates enforce.
