// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package l1 holds the unit-stride Level-1 kernels behind the public keel
// routines (DESIGN.md §4/P1).
//
// # Shape of this package
//
// A Kernels value is one backend's full set of unit-stride loops. The public
// routines pick one at init (see dispatch.go) and call through it once per
// call, so the indirect call is amortized over the whole vector — there are no
// calls inside any loop.
//
// Only the *unit-stride* case lives here. Strided access (incX != 1, including
// the negative strides BLAS defines) stays scalar in the keel package: gather
// and scatter for Level-1 is a bandwidth-bound operation whose vector form
// wins little, and keeping it out of here keeps every kernel below free of
// index arithmetic. Revisit if profiling ever says otherwise.
//
// # Numerics
//
// Each backend sums in its own order, so results differ from each other and
// from the float64 oracle by reassociation. That is expected and bounded:
// DESIGN.md §5's tolerance model (internal/oracle.Tolerance) is what all three
// are held to, and the cross-backend differential tests use the same model
// rather than demanding bit-equality. This is the one place keel's testing is
// deliberately looser than internal/vec's, where bit-exactness *was*
// achievable — a 16-lane reduction tree and a 4-accumulator scalar loop cannot
// agree bit-for-bit, and pretending otherwise would mean either crippling the
// vector path or writing a scalar path nobody would ship.
package l1

import "math"

// Kernels is one backend's unit-stride Level-1 loop set. Every field is
// mandatory; a nil op is a registration bug, not a fallback.
type Kernels struct {
	Name string
	// Dot returns the inner product. len(x) == len(y) is the caller's job.
	Dot func(x, y []float32) float32
	// Axpy computes y += alpha*x elementwise.
	Axpy func(alpha float32, x, y []float32)
	// Scal computes x *= alpha elementwise.
	Scal func(alpha float32, x []float32)
	// Asum returns the sum of magnitudes.
	Asum func(x []float32) float32
	// SumSq returns the sum of squares, used by Snrm2. It may overflow to +Inf
	// or underflow to 0; Snrm2 detects both and reruns a scaled reference.
	SumSq func(x []float32) float32
}

// Backend names, shared with the vec shim and KEEL_FORCE.
const (
	Scalar = "scalar"
	AVX2   = "avx2"
	AVX512 = "avx512"
)

// Backends returns every backend runnable on this machine, widest first, with
// the scalar backend always last and always present.
//
// It is a function rather than a package-level slice built by init()s on
// purpose: init order across files in a package is defined but easy to get
// subtly wrong, and a dispatch table that depends on it is a bug waiting for
// someone to rename a file.
func Backends() []Kernels {
	return append(vectorBackends(), ScalarKernels)
}

// Names returns Backends()' names, for the coverage markers the gate parses.
func Names() []string {
	bs := Backends()
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Name
	}
	return out
}

// ScalarKernels is the always-available backend and the reference the others
// are held against.
var ScalarKernels = Kernels{
	Name:  Scalar,
	Dot:   scalarDot,
	Axpy:  scalarAxpy,
	Scal:  scalarScal,
	Asum:  scalarAsum,
	SumSq: scalarSumSq,
}

// The scalar reductions below use four independent accumulators.
//
// This is a deliberate choice with two consequences worth stating. It is what
// a competent Go programmer would write, so the >=4x speedup gate P1 measures
// against it is a real claim rather than a win over a straw man — a single
// accumulator would serialize on float-add latency and inflate the ratio for
// free. And it means the scalar path is a fallback keel is not embarrassed to
// ship on a machine without AVX, which DESIGN.md §4/P5 requires it to be.
//
// unroll is 4 for the same reason the vector kernels use 4 accumulators: FMA
// and float-add latency is several cycles, and one chain cannot fill the pipe.
const unroll = 4

func scalarDot(x, y []float32) float32 {
	var a0, a1, a2, a3 float32
	n := len(x)
	i := 0
	for ; i+unroll <= n; i += unroll {
		a0 += x[i] * y[i]
		a1 += x[i+1] * y[i+1]
		a2 += x[i+2] * y[i+2]
		a3 += x[i+3] * y[i+3]
	}
	for ; i < n; i++ {
		a0 += x[i] * y[i]
	}
	return (a0 + a1) + (a2 + a3)
}

func scalarAxpy(alpha float32, x, y []float32) {
	for i, v := range x {
		y[i] += alpha * v
	}
}

func scalarScal(alpha float32, x []float32) {
	for i := range x {
		x[i] *= alpha
	}
}

func scalarAsum(x []float32) float32 {
	var a0, a1, a2, a3 float32
	n := len(x)
	i := 0
	for ; i+unroll <= n; i += unroll {
		a0 += absf32(x[i])
		a1 += absf32(x[i+1])
		a2 += absf32(x[i+2])
		a3 += absf32(x[i+3])
	}
	for ; i < n; i++ {
		a0 += absf32(x[i])
	}
	return (a0 + a1) + (a2 + a3)
}

func scalarSumSq(x []float32) float32 {
	var a0, a1, a2, a3 float32
	n := len(x)
	i := 0
	for ; i+unroll <= n; i += unroll {
		a0 += x[i] * x[i]
		a1 += x[i+1] * x[i+1]
		a2 += x[i+2] * x[i+2]
		a3 += x[i+3] * x[i+3]
	}
	for ; i < n; i++ {
		a0 += x[i] * x[i]
	}
	return (a0 + a1) + (a2 + a3)
}

// absf32 clears the sign bit rather than branching on sign, matching
// vec.ScalarAbs: -0 becomes +0 and NaN keeps its payload. Written out here so
// this package does not depend on the shim for one scalar operation.
func absf32(v float32) float32 {
	return math.Float32frombits(math.Float32bits(v) &^ (1 << 31))
}

// Iamax returns the index of the first element of greatest magnitude, or -1
// for an empty slice.
//
// This one has NO vector backend, on purpose. Reference BLAS ISAMAX is defined
// by a sequential comparison chain — `if abs(x[i]) > max` — and that gives NaN
// a very specific and load-bearing behaviour: a NaN in the first position wins
// (it seeds max, and no later comparison against NaN succeeds), while a NaN
// anywhere else is skipped entirely. A lane-parallel max cannot reproduce that,
// because x86's VMAXPS resolves NaN by operand position rather than by
// sequence, so the answer would depend on which lane the NaN landed in. Since
// Isamax is O(n) with no arithmetic to speed up and is not part of gate P1's
// throughput criterion, matching the convention exactly is worth more than
// vectorizing it. A masked vector version is possible; it is a P5 perf item,
// not a correctness gap.
func Iamax(x []float32) int {
	if len(x) == 0 {
		return -1
	}
	best, bi := absf32(x[0]), 0
	for i := 1; i < len(x); i++ {
		if v := absf32(x[i]); v > best {
			best, bi = v, i
		}
	}
	return bi
}
