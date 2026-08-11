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
	n := len(x)
	i := 0
	for ; i+step512 <= n; i += step512 {
		a0 = vec.FMA512(vec.Load512(x[i:i+lanes512]), vec.Load512(y[i:i+lanes512]), a0)
		a1 = vec.FMA512(vec.Load512(x[i+16:i+32]), vec.Load512(y[i+16:i+32]), a1)
		a2 = vec.FMA512(vec.Load512(x[i+32:i+48]), vec.Load512(y[i+32:i+48]), a2)
		a3 = vec.FMA512(vec.Load512(x[i+48:i+64]), vec.Load512(y[i+48:i+64]), a3)
	}
	for ; i+lanes512 <= n; i += lanes512 {
		a0 = vec.FMA512(vec.Load512(x[i:i+lanes512]), vec.Load512(y[i:i+lanes512]), a0)
	}
	if i < n {
		a0 = vec.FMA512(vec.LoadPart512(x[i:]), vec.LoadPart512(y[i:]), a0)
	}
	acc := vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3))
	return vec.HSum512(acc)
}

func avx512Axpy(alpha float32, x, y []float32) {
	va := vec.Broadcast512(alpha)
	n := len(x)
	i := 0
	for ; i+lanes512 <= n; i += lanes512 {
		xs, ys := x[i:i+lanes512], y[i:i+lanes512]
		vec.Store512(ys, vec.FMA512(va, vec.Load512(xs), vec.Load512(ys)))
	}
	if i < n {
		xs, ys := x[i:], y[i:]
		vec.StorePart512(ys, vec.FMA512(va, vec.LoadPart512(xs), vec.LoadPart512(ys)))
	}
}

func avx512Scal(alpha float32, x []float32) {
	va := vec.Broadcast512(alpha)
	n := len(x)
	i := 0
	for ; i+lanes512 <= n; i += lanes512 {
		xs := x[i : i+lanes512]
		vec.Store512(xs, vec.Mul512(va, vec.Load512(xs)))
	}
	if i < n {
		xs := x[i:]
		vec.StorePart512(xs, vec.Mul512(va, vec.LoadPart512(xs)))
	}
}

func avx512Asum(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero512(), vec.Zero512(), vec.Zero512(), vec.Zero512()
	n := len(x)
	i := 0
	for ; i+step512 <= n; i += step512 {
		a0 = vec.Add512(a0, vec.Abs512(vec.Load512(x[i:i+lanes512])))
		a1 = vec.Add512(a1, vec.Abs512(vec.Load512(x[i+16:i+32])))
		a2 = vec.Add512(a2, vec.Abs512(vec.Load512(x[i+32:i+48])))
		a3 = vec.Add512(a3, vec.Abs512(vec.Load512(x[i+48:i+64])))
	}
	for ; i+lanes512 <= n; i += lanes512 {
		a0 = vec.Add512(a0, vec.Abs512(vec.Load512(x[i:i+lanes512])))
	}
	if i < n {
		a0 = vec.Add512(a0, vec.Abs512(vec.LoadPart512(x[i:])))
	}
	return vec.HSum512(vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3)))
}

func avx512SumSq(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero512(), vec.Zero512(), vec.Zero512(), vec.Zero512()
	n := len(x)
	i := 0
	for ; i+step512 <= n; i += step512 {
		v0 := vec.Load512(x[i : i+lanes512])
		v1 := vec.Load512(x[i+16 : i+32])
		v2 := vec.Load512(x[i+32 : i+48])
		v3 := vec.Load512(x[i+48 : i+64])
		a0 = vec.FMA512(v0, v0, a0)
		a1 = vec.FMA512(v1, v1, a1)
		a2 = vec.FMA512(v2, v2, a2)
		a3 = vec.FMA512(v3, v3, a3)
	}
	for ; i+lanes512 <= n; i += lanes512 {
		v := vec.Load512(x[i : i+lanes512])
		a0 = vec.FMA512(v, v, a0)
	}
	if i < n {
		v := vec.LoadPart512(x[i:])
		a0 = vec.FMA512(v, v, a0)
	}
	return vec.HSum512(vec.Add512(vec.Add512(a0, a1), vec.Add512(a2, a3)))
}

// -------------------------------------------------------------------- AVX2

func avx2Dot(x, y []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	n := len(x)
	i := 0
	for ; i+step256 <= n; i += step256 {
		a0 = vec.FMA256(vec.Load256(x[i:i+lanes256]), vec.Load256(y[i:i+lanes256]), a0)
		a1 = vec.FMA256(vec.Load256(x[i+8:i+16]), vec.Load256(y[i+8:i+16]), a1)
		a2 = vec.FMA256(vec.Load256(x[i+16:i+24]), vec.Load256(y[i+16:i+24]), a2)
		a3 = vec.FMA256(vec.Load256(x[i+24:i+32]), vec.Load256(y[i+24:i+32]), a3)
	}
	for ; i+lanes256 <= n; i += lanes256 {
		a0 = vec.FMA256(vec.Load256(x[i:i+lanes256]), vec.Load256(y[i:i+lanes256]), a0)
	}
	if i < n {
		a0 = vec.FMA256(vec.LoadPart256(x[i:]), vec.LoadPart256(y[i:]), a0)
	}
	return vec.HSum256(vec.Add256(vec.Add256(a0, a1), vec.Add256(a2, a3)))
}

func avx2Axpy(alpha float32, x, y []float32) {
	va := vec.Broadcast256(alpha)
	n := len(x)
	i := 0
	for ; i+lanes256 <= n; i += lanes256 {
		xs, ys := x[i:i+lanes256], y[i:i+lanes256]
		vec.Store256(ys, vec.FMA256(va, vec.Load256(xs), vec.Load256(ys)))
	}
	if i < n {
		xs, ys := x[i:], y[i:]
		vec.StorePart256(ys, vec.FMA256(va, vec.LoadPart256(xs), vec.LoadPart256(ys)))
	}
}

func avx2Scal(alpha float32, x []float32) {
	va := vec.Broadcast256(alpha)
	n := len(x)
	i := 0
	for ; i+lanes256 <= n; i += lanes256 {
		xs := x[i : i+lanes256]
		vec.Store256(xs, vec.Mul256(va, vec.Load256(xs)))
	}
	if i < n {
		xs := x[i:]
		vec.StorePart256(xs, vec.Mul256(va, vec.LoadPart256(xs)))
	}
}

func avx2Asum(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	n := len(x)
	i := 0
	for ; i+step256 <= n; i += step256 {
		a0 = vec.Add256(a0, vec.Abs256(vec.Load256(x[i:i+lanes256])))
		a1 = vec.Add256(a1, vec.Abs256(vec.Load256(x[i+8:i+16])))
		a2 = vec.Add256(a2, vec.Abs256(vec.Load256(x[i+16:i+24])))
		a3 = vec.Add256(a3, vec.Abs256(vec.Load256(x[i+24:i+32])))
	}
	for ; i+lanes256 <= n; i += lanes256 {
		a0 = vec.Add256(a0, vec.Abs256(vec.Load256(x[i:i+lanes256])))
	}
	if i < n {
		a0 = vec.Add256(a0, vec.Abs256(vec.LoadPart256(x[i:])))
	}
	return vec.HSum256(vec.Add256(vec.Add256(a0, a1), vec.Add256(a2, a3)))
}

func avx2SumSq(x []float32) float32 {
	a0, a1, a2, a3 := vec.Zero256(), vec.Zero256(), vec.Zero256(), vec.Zero256()
	n := len(x)
	i := 0
	for ; i+step256 <= n; i += step256 {
		v0 := vec.Load256(x[i : i+lanes256])
		v1 := vec.Load256(x[i+8 : i+16])
		v2 := vec.Load256(x[i+16 : i+24])
		v3 := vec.Load256(x[i+24 : i+32])
		a0 = vec.FMA256(v0, v0, a0)
		a1 = vec.FMA256(v1, v1, a1)
		a2 = vec.FMA256(v2, v2, a2)
		a3 = vec.FMA256(v3, v3, a3)
	}
	for ; i+lanes256 <= n; i += lanes256 {
		v := vec.Load256(x[i : i+lanes256])
		a0 = vec.FMA256(v, v, a0)
	}
	if i < n {
		v := vec.LoadPart256(x[i:])
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
