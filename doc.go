// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

/*
Package keel is a pure-Go float32 BLAS subset whose hot kernels are written
against the experimental simd/archsimd packages (GOEXPERIMENT=simd).

There is no cgo, no assembly and no build-time code generation: the vector code
is Go, and on a toolchain or a machine without the vector backend the same Go
compiles and runs through a scalar path. DESIGN.md is the full contract; this
comment is what a caller needs.

# Build requirements

The vector path needs a Go toolchain built with GOEXPERIMENT=simd on amd64. The
scalar path needs neither, and is not a stub — it is the reference the vector
kernels are differentially tested against, and it builds on a stock toolchain.
That is a gated property, not an aspiration: gate-p1 runs the whole suite twice,
once with the vector backend forced off.

# What is here

Level 1: [Sdot], [Saxpy], [Sscal], [Sasum], [Snrm2], [Isamax].

Level 2: [Sgemv], [Sger].

Level 3: [Sgemm], [Ssyrk], [Ssymm], [Strsm].

The Level-3 routines are one blocked GEMM nest and three reductions to it — a
triangular C mask for Ssyrk, a symmetric expansion for Ssymm, a block-solve
recurrence for Strsm — rather than four independent implementations. What that
buys is that all four inherit one packing strategy, one set of blocking
parameters, one edge strategy and, the part that took two phases to establish,
one audited K-loop.

Row-major, and the leading dimension is the row stride. Reference BLAS is
column-major and Fortran-ordered; keel is not a translation of it and does not
pretend to be, so there is no `order` argument to get wrong.

# Errors are panics

Argument errors panic instead of returning quietly. Reference SSCAL's
`IF (n.LE.0 .OR. incx.LE.0) RETURN` turns a caller's off-by-one into a no-op
that looks like success, and in Go there is no XERBLA to consult afterwards. A
bad n, a zero stride, or a slice too short for the stride it was given is a
program bug and says so, at the call.

n == 0 is not an error. It is the empty vector or the empty matrix, and the
slices may be nil.

# Numerics

Results are float32 and the accumulation order is the active backend's, so a
result is not bit-stable across backends. It is bit-stable within one, and every
backend is checked against a float64 oracle under one shared error model
(internal/oracle) rather than against per-test epsilons. The tolerance is a
function of the shape and the operation, derived once; a routine that needs a
wider one has a numerics question, not a test question.

Special values follow reference BLAS rather than convenience. 0·Inf propagates
NaN, the unreferenced triangle of a symmetric or triangular argument is never
read, and no routine takes a fast path for a zero multiplier — a "fast path" for
a zero in a factored matrix would silently return a different matrix on an input
that is both legal and common.

# Backend dispatch

Chosen once at init from CPU feature detection. Level 1 dispatches avx512 →
avx2 → scalar; Level 3 dispatches avx512 → scalar, because there is no AVX2
microkernel and no host this project measures on is AVX2-only silicon — a middle
rung with no evidence behind it would be a claim rather than a fallback.
[L1Chain] and [KernChain] report the advertised chains; [AvailableL1Backends]
and [AvailableKernels] report what is runnable here, which on a machine without
AVX-512 is properly shorter.

Set KEEL_FORCE=scalar|avx2|avx512 to pin the choice. It exists for testing — it
is how the suite is run scalar-only on a machine that has AVX-512, which is the
only way to prove the fallback works rather than assume it. An unavailable value
panics at init rather than downgrading silently: a run that believed it was
measuring AVX-512 and quietly measured scalar is worse than a crash.

The microkernel shape is also chosen per host, from a measured classification of
what binds the K-loop on this part. [ActiveKernTile], [ActiveKernClass] and
[ActiveKernClassEvidence] report the choice and its grounds.

# Parallelism

The Level-3 routines distribute work over a bounded pool of goroutines sized by
runtime.GOMAXPROCS(0), started per call and joined before the call returns.
GOMAXPROCS is the only knob, because it is the knob a caller already has and
already expects a Go library to respect.

Three consequences a caller can rely on:

  - At GOMAXPROCS=1 the nest runs in the calling goroutine, with no goroutine,
    no atomic and no scheduling in the path.
  - No goroutine outlives a call, so keel is safe to call from inside something
    that counts them, and repeated calls leak nothing.
  - The result is bit-identical at every GOMAXPROCS. The parallel axis
    partitions the output rather than any single output element's sum, so this
    is exact equality and not a tolerance. A float32 BLAS whose answer moved
    with the core count would be a different library on every machine.

Level 1 is deliberately not parallelized: those routines are memory-bound at
every size where a thread would pay for itself, and a caller reaches Level-1
parallelism by calling from parallel code rather than by having each call fan
out. [Workers], [WorkersLastCall] and [GOMAXPROCS] report the sizing rule, what
the last Level-3 call actually used, and GOMAXPROCS as the nest reads it.

# Numbers

Every performance figure keel publishes carries its denominator: the CPU model,
the peak measured on that same host in that same run, and the OpenBLAS reference
where one was available — and says so explicitly where one was not. The README's
published rates live in a block that the phase gate re-measures on the hosts it
runs on and fails on a 5% disagreement, so a stale number cannot survive a gate
run. Nothing here reports a rate divided by a peak taken from a formula.
*/
package keel
