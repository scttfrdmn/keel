// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

/*
Package keel is a float32 BLAS subset in pure Go: Levels 1, 2 and a
GEMM-centered Level 3, with amd64 AVX-512 kernels written against the
experimental simd/archsimd packages and a scalar path that builds on an ordinary
toolchain. There is no cgo, no assembly, no code generation and no background
goroutine — it deploys as a normal Go module and respects GOMAXPROCS.

# Two build modes

	GOEXPERIMENT=simd go build ./...   # vector kernels on amd64, Go 1.27+
	go build ./...                     # scalar path, any Go 1.26+ toolchain

Both modes compile the same source and produce the same answers. The vector
kernels are compiled only in the first, so a build without GOEXPERIMENT=simd is
correct and slow. Nothing warns you, so ask:

	fmt.Println(keel.ActiveL1Backend(), keel.ActiveKernBackend())

That prints avx512 twice on an amd64 machine with AVX-512 built under
GOEXPERIMENT=simd, and scalar twice wherever no vector backend is compiled in or
detected — a stock-toolchain build, and also any non-amd64 host, which prints
scalar twice even under GOEXPERIMENT=simd.

The two do not always agree, because the chains have a different number of rungs:
Level 1 dispatches avx512 → avx2 → scalar, while the Level-3 microkernel
dispatches avx512 → scalar. On AVX2-only silicon it therefore prints "avx2
scalar" — a vector Level 1 over a scalar SGEMM. See dispatch.go for why the
middle rung is deliberately absent at Level 3 (issue #40).

# Matrices are row-major

Every matrix argument is one []float32 in row-major order plus a leading
dimension: the element count from the start of one row to the start of the next.
For an m×n matrix A with leading dimension lda,

	A[i][j] == a[i*lda+j]

lda may exceed n, which is how a submatrix is passed without copying it. Here a
2×3 matrix sits inside a 5-wide array:

	      col 0   1   2   3   4        m = 2, n = 3, lda = 5
	row 0   [ 1   2   3   ·   · ]      A[1][2] is a[1*5+2], which is 6
	row 1   [ 4   5   6   ·   · ]      the · elements are never read

Reference BLAS is column-major and keel is not a translation of it, so there is
no order argument to get wrong. A leading dimension always describes the array as
stored, before any [Transpose] flag is applied.

Vectors take a stride instead: incX of 1 is contiguous, 2 reads every other
element, and a negative stride walks backwards. [Sdot], [Saxpy], [Sgemv] and
[Sger] accept a negative stride; [Sscal], [Sasum], [Snrm2] and [Isamax] require
incX > 0, as reference BLAS defines them.

# A minimal call

	// C = A·B, A 2×3, B 3×2, C 2×2, each tightly packed so ld is the row length.
	a := []float32{1, 2, 3, 4, 5, 6}
	b := []float32{7, 8, 9, 10, 11, 12}
	c := make([]float32, 4)
	keel.Sgemm(keel.NoTrans, keel.NoTrans, 2, 2, 3, 1, a, 3, b, 2, 0, c, 2)
	// c is now {58, 64, 139, 154}

Each routine carries a runnable example; run `go test` to check them.

# What is here

Level 1: [Sdot], [Saxpy], [Sscal], [Sasum], [Snrm2], [Isamax].

Level 2: [Sgemv], [Sger].

Level 3: [Sgemm], [Ssyrk], [Ssymm], [Strsm].

Flags are [Transpose], [Uplo], [Side] and [Diag], each holding reference BLAS's
own letter.

# Invalid arguments panic

A negative dimension, a zero stride, an unrecognized flag, a leading dimension
narrower than the row it describes, or a slice too short for the shape it was
given panics at the call. Reference BLAS returns silently from several of these
and leaves a message with XERBLA; in Go there is nothing to consult afterwards,
so keel reports it where it happened.

n == 0 is not an error. It is the empty vector or the empty matrix, the call does
nothing, and the slices may be nil. A matrix whose declared shape is not empty
must be long enough for that shape even on a call that will not read it.

Where reference BLAS skips work for a zero multiplier, keel skips exactly the
same work, because those rules are observable: alpha == 0 must not pull a NaN out
of an operand nobody wanted read, and beta == 0 must not read an uninitialized
destination. Each routine's own comment states its case.

# Numerics you can rely on

Results are float32, computed in the active backend's summation order. So:

  - A result is bit-stable for a given backend, and bit-identical at every
    GOMAXPROCS — exactly, not within a tolerance. The parallel axis splits the
    output, never a single output element's sum, so the answer does not move with
    the core count.
  - A result is not bit-identical across backends, or to a textbook triple loop.
    Blocked accumulation and vector reduction add in a different order.
  - Special values follow reference BLAS rather than convenience. 0·Inf
    propagates NaN, the unreferenced triangle of a symmetric or triangular
    argument is never read, and no routine takes a fast path on a zero it was not
    told to expect.

# Choosing a backend

The backend is chosen once, at init, from CPU feature detection: Level 1 tries
avx512, then avx2, then scalar; Level 3 tries avx512, then scalar.

Set KEEL_FORCE to scalar, avx2 or avx512 to pin it. This is a testing knob — it
is how a machine with AVX-512 runs the suite scalar-only — and naming a backend
the machine cannot run panics at init rather than quietly downgrading. There is
no AVX2 microkernel, so KEEL_FORCE=avx2 gives an AVX2 Level 1 and a scalar Level
3, with [ActiveKernBackend] reporting scalar so that no measurement can believe
otherwise.

[L1Chain] and [KernChain] report those two chains. [AvailableL1Backends] and
[AvailableKernels] report what is runnable on this machine, which is properly
shorter without AVX-512, and [ActiveKernTile] reports which microkernel shape was
selected for this host.

# Parallelism

The Level-3 routines spread their work over goroutines sized by
runtime.GOMAXPROCS(0), started per call and joined before the call returns.
GOMAXPROCS is the only knob.

At GOMAXPROCS=1 the nest runs in the calling goroutine, with no goroutine and no
atomic in the path. No goroutine outlives a call, so repeated calls leak nothing
and keel is safe to call from code that counts goroutines. [Workers] reports the
sizing rule.

Level 1 is not parallelized: those routines are memory-bound at every size where
a thread would pay for itself, so reach Level-1 parallelism by calling from
parallel code.

# Performance numbers, and the rest of the documentation

Measured rates, each with the CPU model it was measured on and the denominator it
was divided by, are published on the site rather than here. A doc comment is a
contract, and contracts do not go stale the way numbers do:

	https://scttfrdmn.github.io/keel/numbers/

The site's user pages cover installing, calling and troubleshooting keel; its
Project records section carries the design document, the testing methodology, the
toolchain field notes and the changelog, for anyone who wants why rather than how.
*/
package keel
