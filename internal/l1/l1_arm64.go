// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package l1

import "github.com/scttfrdmn/keel/internal/vec"

// NEON Level-1 kernels (#154), the L1 half of the arm64 port. This mirrors
// l1_amd64.go over the 128-bit Float32x4 shim in internal/vec — the same shim
// ops the #136 microkernels use, differentially tested against the scalar spec
// since that unit. No archsimd import here: the shim owns every simd import
// (CLAUDE.md), and vec's type aliases let this name a V-register without
// reaching past it.
//
// # What carries over from amd64, and what does not
//
// The loop *shape* is identical and for the same reasons, so l1_amd64.go's
// header is the authority on all of it: four independent accumulator chains to
// keep both FMA ports fed past the ~4-cycle latency; constant-offset sub-slices
// (`x[0:4]`, never `x[i:i+4]`) so the bounds checks fall to `prove`; the `>`
// rather than `>=` loop guard so the advance is an unconditional add rather than
// the empty-slice conditional bump (T19); and the exact-fit epilogue that keeps
// all four accumulators live when n is an exact multiple of the step, so a NEON
// reduction on such an n does not collapse to one dependent chain in the mop-up.
//
// What does NOT carry over is the *measurement*: T19's instruction counts, the
// T22 placement drift, and the #47 A/B were all taken on linux/amd64. This file
// inherits the structure because it is correct and bounds-check-clean by
// construction (verified by the spill audit, `spill-audit -goarch arm64 -mode
// spill`, and by the ragged-length differential in l1_test.go), not because the
// amd64 deltas are assumed to reproduce. A re-measurement on Graviton/GB10 is
// #154's own follow-up, not a claim made here.
//
// Two NEON-specific simplifications versus amd64:
//
//   - The absolute value is a single VFABS (vec.Abs128), so Asum needs no
//     loop-invariant sign mask to hand-hoist the way avx512Asum hoists
//     AbsMask512. There is nothing loop-invariant in the reduction body to lift,
//     so the #54/#130 LICM caveat does not bite Asum here; it still bites the
//     alpha broadcast in Axpy/Scal, which is hand-hoisted above the loop exactly
//     as amd64 does it.
//   - The vector is four lanes, not sixteen, so the step is 16 elements
//     (4 lanes x unroll=4) rather than 64.

const (
	lanes128 = 4
	// Elements consumed per unrolled iteration: four V-registers of four lanes.
	step128 = lanes128 * unroll
)

func neonDot(x, y []float32) float32 {
	a0, a1, a2, a3 := vec.Zero128(), vec.Zero128(), vec.Zero128(), vec.Zero128()
	y = y[:len(x)]
	// The len(y) conjunct feeds the prover; it is redundant to a reader after the
	// re-slice above but load-bearing for bounds-check elimination on amd64 (T19).
	// Kept on arm64 for the same reason and confirmed clean by the spill audit —
	// do not drop it without re-running it.
	for len(x) > step128 && len(y) > step128 {
		a0 = vec.FMA128(vec.Load128(x[0:4]), vec.Load128(y[0:4]), a0)
		a1 = vec.FMA128(vec.Load128(x[4:8]), vec.Load128(y[4:8]), a1)
		a2 = vec.FMA128(vec.Load128(x[8:12]), vec.Load128(y[8:12]), a2)
		a3 = vec.FMA128(vec.Load128(x[12:16]), vec.Load128(y[12:16]), a3)
		x, y = x[step128:], y[step128:]
	}
	if len(x) == step128 && len(y) == step128 {
		a0 = vec.FMA128(vec.Load128(x[0:4]), vec.Load128(y[0:4]), a0)
		a1 = vec.FMA128(vec.Load128(x[4:8]), vec.Load128(y[4:8]), a1)
		a2 = vec.FMA128(vec.Load128(x[8:12]), vec.Load128(y[8:12]), a2)
		a3 = vec.FMA128(vec.Load128(x[12:16]), vec.Load128(y[12:16]), a3)
		x, y = x[:0], y[:0]
	}
	for len(x) >= lanes128 && len(y) >= lanes128 {
		a0 = vec.FMA128(vec.Load128(x[0:4]), vec.Load128(y[0:4]), a0)
		x, y = x[lanes128:], y[lanes128:]
	}
	if len(x) > 0 {
		a0 = vec.FMA128(vec.LoadPart128(x), vec.LoadPart128(y), a0)
	}
	acc := vec.Add128(vec.Add128(a0, a1), vec.Add128(a2, a3))
	return vec.HSum128(acc)
}

func neonAxpy(alpha float32, x, y []float32) {
	va := vec.Broadcast128(alpha) // hand-hoisted; SIMD ops are not lifted by LICM (#54, T18)
	y = y[:len(x)]
	// Load-bearing len(y) conjunct, as in neonDot / avx512Axpy.
	for len(x) > lanes128 && len(y) > lanes128 {
		xs, ys := x[0:4], y[0:4]
		vec.Store128(ys, vec.FMA128(va, vec.Load128(xs), vec.Load128(ys)))
		x, y = x[lanes128:], y[lanes128:]
	}
	if len(x) == lanes128 && len(y) == lanes128 {
		xs, ys := x[0:4], y[0:4]
		vec.Store128(ys, vec.FMA128(va, vec.Load128(xs), vec.Load128(ys)))
		return
	}
	if len(x) > 0 {
		vec.StorePart128(y, vec.FMA128(va, vec.LoadPart128(x), vec.LoadPart128(y)))
	}
}

func neonScal(alpha float32, x []float32) {
	va := vec.Broadcast128(alpha) // hand-hoisted; see neonAxpy
	for len(x) > lanes128 {
		xs := x[0:4]
		vec.Store128(xs, vec.Mul128(va, vec.Load128(xs)))
		x = x[lanes128:]
	}
	if len(x) == lanes128 {
		xs := x[0:4]
		vec.Store128(xs, vec.Mul128(va, vec.Load128(xs)))
		return
	}
	if len(x) > 0 {
		vec.StorePart128(x, vec.Mul128(va, vec.LoadPart128(x)))
	}
}

func neonAsum(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero128(), vec.Zero128(), vec.Zero128(), vec.Zero128()
	// No hoisted mask: vec.Abs128 is a single VFABS, so unlike avx512Asum there is
	// nothing loop-invariant to lift.
	for len(x) > step128 {
		a0 = vec.Add128(a0, vec.Abs128(vec.Load128(x[0:4])))
		a1 = vec.Add128(a1, vec.Abs128(vec.Load128(x[4:8])))
		a2 = vec.Add128(a2, vec.Abs128(vec.Load128(x[8:12])))
		a3 = vec.Add128(a3, vec.Abs128(vec.Load128(x[12:16])))
		x = x[step128:]
	}
	if len(x) == step128 {
		a0 = vec.Add128(a0, vec.Abs128(vec.Load128(x[0:4])))
		a1 = vec.Add128(a1, vec.Abs128(vec.Load128(x[4:8])))
		a2 = vec.Add128(a2, vec.Abs128(vec.Load128(x[8:12])))
		a3 = vec.Add128(a3, vec.Abs128(vec.Load128(x[12:16])))
		x = x[:0]
	}
	for len(x) >= lanes128 {
		a0 = vec.Add128(a0, vec.Abs128(vec.Load128(x[0:4])))
		x = x[lanes128:]
	}
	if len(x) > 0 {
		a0 = vec.Add128(a0, vec.Abs128(vec.LoadPart128(x)))
	}
	return vec.HSum128(vec.Add128(vec.Add128(a0, a1), vec.Add128(a2, a3)))
}

func neonSumSq(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero128(), vec.Zero128(), vec.Zero128(), vec.Zero128()
	for len(x) > step128 {
		v0 := vec.Load128(x[0:4])
		v1 := vec.Load128(x[4:8])
		v2 := vec.Load128(x[8:12])
		v3 := vec.Load128(x[12:16])
		a0 = vec.FMA128(v0, v0, a0)
		a1 = vec.FMA128(v1, v1, a1)
		a2 = vec.FMA128(v2, v2, a2)
		a3 = vec.FMA128(v3, v3, a3)
		x = x[step128:]
	}
	if len(x) == step128 {
		v0 := vec.Load128(x[0:4])
		v1 := vec.Load128(x[4:8])
		v2 := vec.Load128(x[8:12])
		v3 := vec.Load128(x[12:16])
		a0 = vec.FMA128(v0, v0, a0)
		a1 = vec.FMA128(v1, v1, a1)
		a2 = vec.FMA128(v2, v2, a2)
		a3 = vec.FMA128(v3, v3, a3)
		x = x[:0]
	}
	for len(x) >= lanes128 {
		v := vec.Load128(x[0:4])
		a0 = vec.FMA128(v, v, a0)
		x = x[lanes128:]
	}
	if len(x) > 0 {
		v := vec.LoadPart128(x)
		a0 = vec.FMA128(v, v, a0)
	}
	return vec.HSum128(vec.Add128(vec.Add128(a0, a1), vec.Add128(a2, a3)))
}

// vectorBackends returns the NEON backend when this arm64 CPU has it. Gated on
// runtime feature detection rather than the build tag alone, symmetric with
// l1_amd64.go — a binary carries these kernels and must not run them on a core
// without NEON.
func vectorBackends() []Kernels {
	if !vec.HasNEON() {
		return nil
	}
	return []Kernels{{
		Name: NEON, Dot: neonDot, Axpy: neonAxpy,
		Scal: neonScal, Asum: neonAsum, SumSq: neonSumSq,
	}}
}
