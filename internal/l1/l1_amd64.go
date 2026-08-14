// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package l1

import "github.com/scttfrdmn/keel/internal/vec"

// Vector Level-1 kernels, over the internal/vec hot layer. No archsimd import
// here: the shim owns every simd import (CLAUDE.md), and vec's exported type
// aliases let this package name a zmm register without reaching past it.
//
// # Why four accumulators
//
// Every reduction below carries four independent accumulator chains. A single
// chain would stall on FMA latency — roughly 4 cycles on the targets in
// docs/hosts.md — while the two FMA ports sit idle, capping throughput near
// 1/8 of peak no matter how wide the vectors are. Four chains is the smallest
// count that keeps both ports fed and still leaves the register allocator
// nothing to spill at this loop's register pressure (4 accumulators + 2 loaded
// operands). It is the same reasoning DESIGN.md §4/P2 applies to the SGEMM
// microkernel's accumulator count, one level simpler.
//
// # Why the tail is a partial vector op rather than a scalar loop
//
// vec.LoadPart512 zero-fills lanes past the end of the slice, and +0 is the
// identity for every reduction here, so the remainder needs no separate code
// path and no branch per element. For Axpy and Scal the store side uses
// StorePart512, which writes only the lanes that exist — so a remainder cannot
// scribble past the end of y even though the arithmetic ran on all 16 lanes.
//
// # Why these loops advance slices instead of an index
//
// Every loop below is driven by len(x) and re-slices at the bottom, with
// *constant* offsets in the body — `x[16:32]`, never `x[i+16:i+32]`. That is not
// a style preference: it is what makes the bounds checks go away, and issue #47
// is the measurement that says the obvious form does not.
//
// The obvious form is an induction variable with the guard `i+16 <= len(x)`. It
// leaves two checks per sub-slice standing. `i+16 <= len(x)` and the slice
// invariant `len(x) <= cap(x)` together imply `i+16 <= cap(x)`, but `prove` does
// not take that step, and `i <= i+16` needs no-overflow reasoning it also does
// not do. So an unrolled body paid one surviving check per offset sub-slice:
// avx512Dot ran 69 instructions to issue four FMAs, and even avx512Scal — one
// slice, no unrolling, a guard matching its slice exactly — paid one.
//
// Against `len(x) >= 64` with a constant offset, the same reasoning is
// constant-versus-constant and the prover does make the step. It is the idiom
// internal/vec's microkernels already use for their panels (CLAUDE.md's
// "pre-sliced panels"), and P2 wrote that rule for kernels only; these loops
// were written in P1 and were never held to it.
//
// The two-slice routines re-slice y to len(x) once, above the loop. That is
// where the "len(x) == len(y) is the caller's job" precondition gets enforced —
// once, before anything is written, instead of as a bounds check per iteration —
// and it is what lets one guard on len(x) discharge y's bounds as well.
//
// # Why the guards are `>` and not `>=` (T19)
//
// Removing the bounds checks moved the cost rather than removing it. Under
// `len(x) >= 16`, the advance `x = x[16:]` can produce an *empty* slice, and an
// empty slice may not carry a past-the-end pointer — so the compiler emits a
// branchless conditional bump rather than an add:
//
//	ADDQ $-16, CX;  MOVQ CX, DX;  NEGQ DX;  SARQ $63, DX;  ANDL $64, DX
//	...
//	ADDQ DX, AX                       // += 64 if len-16 > 0, else += 0
//
// Five instructions to compute an offset that is 64 on every iteration but the
// last. Axpy and Dot advance two slices, so they paid it twice: avx512Axpy ran
// 26 instructions in its steady state to issue four vector ops, seventeen of
// them bookkeeping.
//
// Under `len(x) > 16` the emptiness case is gone, the prover sees it, and the
// whole sequence collapses to `ADDQ $64, AX`. Measured on go1.26.6, steady-state
// loop instructions (linux/amd64, the object code the hosts run):
//
//	avx512Scal   16 -> 9     avx512Dot    52 -> 44
//	avx512Axpy   26 -> 15    avx512Asum   29 -> 25
//
// The reductions gain less because their advance amortizes over 64 elements
// where Axpy's and Scal's amortizes over 16. That asymmetry is the shape of
// #47's regression: Saxpy was the routine that lost, and Saxpy's loop is the one
// where the advance outweighed the work.
//
// # What `>` leaves behind, and why it is not the partial tail's problem
//
// A `>` guard exits with up to a full step still unconsumed, and the partial
// tail cannot absorb that: LoadPart512 is documented as *equivalent to a full
// load* at 16 or more elements, which means it silently ignores a 17th. So the
// full block is handled explicitly, and differently for the two loop shapes:
//
//   - The reductions already have a 16-wide mop-up loop under the unrolled one.
//     It keeps its `>=` guard and drains the ≤64 elements `>` leaves, in at most
//     four iterations. There is no epilogue to write and nothing about the tail
//     changes. Only the hot loop's guard moves.
//   - Axpy and Scal have no second loop, so they get an explicit exact-fit
//     epilogue that runs the body once at *full* width. It could have been left
//     to the partial tail instead, and that was rejected on two counts: a masked
//     store where a full one would do, and — the deciding one — it would make
//     every exact-multiple call execute a partial op, where today only ragged
//     lengths do. Partial ops are what #42 makes fatal under -race, so widening
//     their reach from ragged-n to every call would have traded instructions for
//     a larger `checkptr` blast radius. The epilogue keeps that frequency
//     exactly where it was.
//
// Two shapes that were tried and rejected, both measured:
//
//   - `>=` with an early `return` on exact fit, to hand the prover a `len != 16`
//     fact before the advance. It does not use it: Scal 16 -> 15, and Axpy got
//     *worse* at 27.
//   - Dropping the redundant `&& len(y) > 16` conjunct, since y was re-sliced to
//     len(x) above. Worse: Axpy 15 -> 24. The conjunct is load-bearing.

const (
	lanes512 = 16
	lanes256 = 8
	// Elements consumed per unrolled iteration.
	step512 = lanes512 * unroll
	step256 = lanes256 * unroll
)

// ------------------------------------------------------------------ AVX-512

func avx512Dot(x, y []float32) float32 {
	a0, a1, a2, a3 := vec.Zero512(), vec.Zero512(), vec.Zero512(), vec.Zero512()
	y = y[:len(x)]
	// The len(y) conjunct looks redundant after the re-slice above and is not:
	// it feeds the prover. Measured on avx512Axpy, dropping it costs 9
	// instructions per iteration (T19). Not separately measured here, but the
	// shape is identical — do not simplify without re-running spill-audit.
	for len(x) > step512 && len(y) > step512 {
		a0 = vec.FMA512(vec.Load512(x[0:16]), vec.Load512(y[0:16]), a0)
		a1 = vec.FMA512(vec.Load512(x[16:32]), vec.Load512(y[16:32]), a1)
		a2 = vec.FMA512(vec.Load512(x[32:48]), vec.Load512(y[32:48]), a2)
		a3 = vec.FMA512(vec.Load512(x[48:64]), vec.Load512(y[48:64]), a3)
		x, y = x[step512:], y[step512:]
	}
	for len(x) >= lanes512 && len(y) >= lanes512 {
		a0 = vec.FMA512(vec.Load512(x[0:16]), vec.Load512(y[0:16]), a0)
		x, y = x[lanes512:], y[lanes512:]
	}
	if len(x) > 0 {
		a0 = vec.FMA512(vec.LoadPart512(x), vec.LoadPart512(y), a0)
	}
	acc := vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3))
	return vec.HSum512(acc)
}

func avx512Axpy(alpha float32, x, y []float32) {
	va := vec.Broadcast512(alpha)
	y = y[:len(x)]
	// The len(y) conjunct is redundant to a reader — y is exactly len(x) long by
	// the line above — and load-bearing to the prover: dropping it took this
	// loop from 15 instructions to 24 (T19). It is not a defensive check and
	// deleting it is not a cleanup.
	for len(x) > lanes512 && len(y) > lanes512 {
		xs, ys := x[0:16], y[0:16]
		vec.Store512(ys, vec.FMA512(va, vec.Load512(xs), vec.Load512(ys)))
		x, y = x[lanes512:], y[lanes512:]
	}
	if len(x) == lanes512 && len(y) == lanes512 {
		xs, ys := x[0:16], y[0:16]
		vec.Store512(ys, vec.FMA512(va, vec.Load512(xs), vec.Load512(ys)))
		return
	}
	if len(x) > 0 {
		vec.StorePart512(y, vec.FMA512(va, vec.LoadPart512(x), vec.LoadPart512(y)))
	}
}

func avx512Scal(alpha float32, x []float32) {
	va := vec.Broadcast512(alpha)
	for len(x) > lanes512 {
		xs := x[0:16]
		vec.Store512(xs, vec.Mul512(va, vec.Load512(xs)))
		x = x[lanes512:]
	}
	if len(x) == lanes512 {
		xs := x[0:16]
		vec.Store512(xs, vec.Mul512(va, vec.Load512(xs)))
		return
	}
	if len(x) > 0 {
		vec.StorePart512(x, vec.Mul512(va, vec.LoadPart512(x)))
	}
}

func avx512Asum(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero512(), vec.Zero512(), vec.Zero512(), vec.Zero512()
	for len(x) > step512 {
		a0 = vec.Add512(a0, vec.Abs512(vec.Load512(x[0:16])))
		a1 = vec.Add512(a1, vec.Abs512(vec.Load512(x[16:32])))
		a2 = vec.Add512(a2, vec.Abs512(vec.Load512(x[32:48])))
		a3 = vec.Add512(a3, vec.Abs512(vec.Load512(x[48:64])))
		x = x[step512:]
	}
	for len(x) >= lanes512 {
		a0 = vec.Add512(a0, vec.Abs512(vec.Load512(x[0:16])))
		x = x[lanes512:]
	}
	if len(x) > 0 {
		a0 = vec.Add512(a0, vec.Abs512(vec.LoadPart512(x)))
	}
	return vec.HSum512(vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3)))
}

func avx512SumSq(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero512(), vec.Zero512(), vec.Zero512(), vec.Zero512()
	for len(x) > step512 {
		v0 := vec.Load512(x[0:16])
		v1 := vec.Load512(x[16:32])
		v2 := vec.Load512(x[32:48])
		v3 := vec.Load512(x[48:64])
		a0 = vec.FMA512(v0, v0, a0)
		a1 = vec.FMA512(v1, v1, a1)
		a2 = vec.FMA512(v2, v2, a2)
		a3 = vec.FMA512(v3, v3, a3)
		x = x[step512:]
	}
	for len(x) >= lanes512 {
		v := vec.Load512(x[0:16])
		a0 = vec.FMA512(v, v, a0)
		x = x[lanes512:]
	}
	if len(x) > 0 {
		v := vec.LoadPart512(x)
		a0 = vec.FMA512(v, v, a0)
	}
	return vec.HSum512(vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3)))
}

// -------------------------------------------------------------------- AVX2

func avx2Dot(x, y []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	y = y[:len(x)]
	// Load-bearing len(y) conjunct, as in avx512Dot — see avx512Axpy for the
	// measurement it rests on.
	for len(x) > step256 && len(y) > step256 {
		a0 = vec.FMA256(vec.Load256(x[0:8]), vec.Load256(y[0:8]), a0)
		a1 = vec.FMA256(vec.Load256(x[8:16]), vec.Load256(y[8:16]), a1)
		a2 = vec.FMA256(vec.Load256(x[16:24]), vec.Load256(y[16:24]), a2)
		a3 = vec.FMA256(vec.Load256(x[24:32]), vec.Load256(y[24:32]), a3)
		x, y = x[step256:], y[step256:]
	}
	for len(x) >= lanes256 && len(y) >= lanes256 {
		a0 = vec.FMA256(vec.Load256(x[0:8]), vec.Load256(y[0:8]), a0)
		x, y = x[lanes256:], y[lanes256:]
	}
	if len(x) > 0 {
		a0 = vec.FMA256(vec.LoadPart256(x), vec.LoadPart256(y), a0)
	}
	return vec.HSum256(vec.Add256(vec.Add256(a0, a1), vec.Add256(a2, a3)))
}

func avx2Axpy(alpha float32, x, y []float32) {
	va := vec.Broadcast256(alpha)
	y = y[:len(x)]
	// Load-bearing len(y) conjunct, as in avx512Axpy.
	for len(x) > lanes256 && len(y) > lanes256 {
		xs, ys := x[0:8], y[0:8]
		vec.Store256(ys, vec.FMA256(va, vec.Load256(xs), vec.Load256(ys)))
		x, y = x[lanes256:], y[lanes256:]
	}
	if len(x) == lanes256 && len(y) == lanes256 {
		xs, ys := x[0:8], y[0:8]
		vec.Store256(ys, vec.FMA256(va, vec.Load256(xs), vec.Load256(ys)))
		return
	}
	if len(x) > 0 {
		vec.StorePart256(y, vec.FMA256(va, vec.LoadPart256(x), vec.LoadPart256(y)))
	}
}

func avx2Scal(alpha float32, x []float32) {
	va := vec.Broadcast256(alpha)
	for len(x) > lanes256 {
		xs := x[0:8]
		vec.Store256(xs, vec.Mul256(va, vec.Load256(xs)))
		x = x[lanes256:]
	}
	if len(x) == lanes256 {
		xs := x[0:8]
		vec.Store256(xs, vec.Mul256(va, vec.Load256(xs)))
		return
	}
	if len(x) > 0 {
		vec.StorePart256(x, vec.Mul256(va, vec.LoadPart256(x)))
	}
}

func avx2Asum(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	for len(x) > step256 {
		a0 = vec.Add256(a0, vec.Abs256(vec.Load256(x[0:8])))
		a1 = vec.Add256(a1, vec.Abs256(vec.Load256(x[8:16])))
		a2 = vec.Add256(a2, vec.Abs256(vec.Load256(x[16:24])))
		a3 = vec.Add256(a3, vec.Abs256(vec.Load256(x[24:32])))
		x = x[step256:]
	}
	for len(x) >= lanes256 {
		a0 = vec.Add256(a0, vec.Abs256(vec.Load256(x[0:8])))
		x = x[lanes256:]
	}
	if len(x) > 0 {
		a0 = vec.Add256(a0, vec.Abs256(vec.LoadPart256(x)))
	}
	return vec.HSum256(vec.Add256(vec.Add256(a0, a1), vec.Add256(a2, a3)))
}

func avx2SumSq(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	for len(x) > step256 {
		v0 := vec.Load256(x[0:8])
		v1 := vec.Load256(x[8:16])
		v2 := vec.Load256(x[16:24])
		v3 := vec.Load256(x[24:32])
		a0 = vec.FMA256(v0, v0, a0)
		a1 = vec.FMA256(v1, v1, a1)
		a2 = vec.FMA256(v2, v2, a2)
		a3 = vec.FMA256(v3, v3, a3)
		x = x[step256:]
	}
	for len(x) >= lanes256 {
		v := vec.Load256(x[0:8])
		a0 = vec.FMA256(v, v, a0)
		x = x[lanes256:]
	}
	if len(x) > 0 {
		v := vec.LoadPart256(x)
		a0 = vec.FMA256(v, v, a0)
	}
	return vec.HSum256(vec.Add256(vec.Add256(a0, a1), vec.Add256(a2, a3)))
}

// vectorBackends returns the vector backends this CPU can actually execute,
// widest first. Gated on runtime feature detection, not just build tags: a
// GOAMD64=v1 binary contains these kernels and must not execute them on a CPU
// without the features (docs/toolchain-notes.md T7 — there is no build-tag
// backstop under this check).
func vectorBackends() []Kernels {
	var out []Kernels
	if vec.HasAVX512() {
		out = append(out, Kernels{
			Name: AVX512, Dot: avx512Dot, Axpy: avx512Axpy,
			Scal: avx512Scal, Asum: avx512Asum, SumSq: avx512SumSq,
		})
	}
	if vec.HasAVX2() {
		out = append(out, Kernels{
			Name: AVX2, Dot: avx2Dot, Axpy: avx2Axpy,
			Scal: avx2Scal, Asum: avx2Asum, SumSq: avx2SumSq,
		})
	}
	return out
}
