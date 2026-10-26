// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// This file holds the float64 DGEMM microkernel bodies (#191), the float64
// counterparts of gemm_amd64.go's. Every body is tools/shapegen's -elem f64
// emission verbatim: TestShippedShapesAreFixedPoints holds them to it, as it holds
// the float32 kernels, so a hand edit fails rather than drifting from the shapes
// the sweep ranked.
//
// The shapes are the float32 ones with half the columns: one ZMM holds 8 float64
// lanes rather than 16 float32, so an MR×16 float64 tile has the accumulator count,
// register budget and loads-per-FMA of an MR×32 float32 tile (docs/f64-design.md
// §2). shapegen's float64 sweep confirms it from the audit: 124 of 128 float64
// shapes audit identically to their float32 twins, 2x16/4 is the frontier at 4.625
// insns/FMA, and 4x16 is the one zero-spill shape with 8 accumulators. 6x16 is the
// 12-accumulator reference that spills, measured as evidence and never shipped,
// exactly as Kernel6x32 is.
//
// Each body ends with archsimd.ClearAVXUpperBits for the reason gemm_amd64.go
// gives (#188); the generator emits that line.

// KernelD2x16 computes C += A·B for one 2x16 float64 tile, four k-steps per pass:
// the float64 twin of Kernel2x32 and the sweep's frontier shape.
//
// Requires len(a) >= kc*2, len(b) >= kc*16, ldc >= 16, and len(c) >= ldc+16.
func KernelD2x16(kc int, a, b, c []float64, ldc int) {
	var (
		c0l, c0h archsimd.Float64x8
		c1l, c1h archsimd.Float64x8
	)
	var bl, bh, av archsimd.Float64x8

	ap := a[:kc*2]
	bp := b[:kc*16]

	for len(ap) >= 8 && len(bp) >= 64 {
		// k + 0
		bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
		av = archsimd.BroadcastFloat64x8(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 1
		bl, bh = archsimd.LoadFloat64x8(bp[16:24]), archsimd.LoadFloat64x8(bp[24:32])
		av = archsimd.BroadcastFloat64x8(ap[2])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[3])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 2
		bl, bh = archsimd.LoadFloat64x8(bp[32:40]), archsimd.LoadFloat64x8(bp[40:48])
		av = archsimd.BroadcastFloat64x8(ap[4])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[5])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 3
		bl, bh = archsimd.LoadFloat64x8(bp[48:56]), archsimd.LoadFloat64x8(bp[56:64])
		av = archsimd.BroadcastFloat64x8(ap[6])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[7])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		ap, bp = ap[8:], bp[64:]
	}

	// Remainder: kc mod 4 k-steps. Correctness for user-supplied k, not a
	// hot path — P3 chooses KC as a multiple of the unroll.
	for len(ap) >= 2 && len(bp) >= 16 {
		bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
		av = archsimd.BroadcastFloat64x8(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		ap, bp = ap[2:], bp[16:]
	}

	r := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c0l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c0h).Store(r[8:16])
	r = c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c1l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c1h).Store(r[8:16])
	archsimd.ClearAVXUpperBits()
}

// KernelD4x16 computes C += A·B for one 4x16 float64 tile, one k-step per pass:
// the float64 twin of Kernel4x32, eight accumulators, fewer loads per FMA.
//
// Requires len(a) >= kc*4, len(b) >= kc*16, ldc >= 16, and len(c) >= 3*ldc+16.
func KernelD4x16(kc int, a, b, c []float64, ldc int) {
	var (
		c0l, c0h archsimd.Float64x8
		c1l, c1h archsimd.Float64x8
		c2l, c2h archsimd.Float64x8
		c3l, c3h archsimd.Float64x8
	)
	var bl, bh, av archsimd.Float64x8

	ap := a[:kc*4]
	bp := b[:kc*16]

	for len(ap) >= 4 && len(bp) >= 16 {
		bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
		av = archsimd.BroadcastFloat64x8(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		ap, bp = ap[4:], bp[16:]
	}

	r := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c0l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c0h).Store(r[8:16])
	r = c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c1l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c1h).Store(r[8:16])
	r = c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c2l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c2h).Store(r[8:16])
	r = c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c3l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c3h).Store(r[8:16])
	archsimd.ClearAVXUpperBits()
}

// KernelD6x16 is the 12-accumulator reference tile, the float64 twin of
// Kernel6x32: measured as evidence that the frontier is real, never dispatched.
//
// Requires len(a) >= kc*6, len(b) >= kc*16, ldc >= 16, and len(c) >= 5*ldc+16.
func KernelD6x16(kc int, a, b, c []float64, ldc int) {
	var (
		c0l, c0h archsimd.Float64x8
		c1l, c1h archsimd.Float64x8
		c2l, c2h archsimd.Float64x8
		c3l, c3h archsimd.Float64x8
		c4l, c4h archsimd.Float64x8
		c5l, c5h archsimd.Float64x8
	)
	var bl, bh, av archsimd.Float64x8

	ap := a[:kc*6]
	bp := b[:kc*16]

	for len(ap) >= 24 && len(bp) >= 64 {
		// k + 0
		bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
		av = archsimd.BroadcastFloat64x8(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat64x8(ap[4])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat64x8(ap[5])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 1
		bl, bh = archsimd.LoadFloat64x8(bp[16:24]), archsimd.LoadFloat64x8(bp[24:32])
		av = archsimd.BroadcastFloat64x8(ap[6])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[7])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[8])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[9])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat64x8(ap[10])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat64x8(ap[11])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 2
		bl, bh = archsimd.LoadFloat64x8(bp[32:40]), archsimd.LoadFloat64x8(bp[40:48])
		av = archsimd.BroadcastFloat64x8(ap[12])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[13])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[14])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[15])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat64x8(ap[16])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat64x8(ap[17])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 3
		bl, bh = archsimd.LoadFloat64x8(bp[48:56]), archsimd.LoadFloat64x8(bp[56:64])
		av = archsimd.BroadcastFloat64x8(ap[18])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[19])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[20])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[21])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat64x8(ap[22])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat64x8(ap[23])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)
		ap, bp = ap[24:], bp[64:]
	}

	// Remainder: kc mod 4 k-steps. Correctness for user-supplied k, not a
	// hot path — P3 chooses KC as a multiple of the unroll.
	for len(ap) >= 6 && len(bp) >= 16 {
		bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
		av = archsimd.BroadcastFloat64x8(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat64x8(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat64x8(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat64x8(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat64x8(ap[4])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat64x8(ap[5])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)
		ap, bp = ap[6:], bp[16:]
	}

	r := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c0l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c0h).Store(r[8:16])
	r = c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c1l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c1h).Store(r[8:16])
	r = c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c2l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c2h).Store(r[8:16])
	r = c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c3l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c3h).Store(r[8:16])
	r = c[4*ldc : 4*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c4l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c4h).Store(r[8:16])
	r = c[5*ldc : 5*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c5l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c5h).Store(r[8:16])
	archsimd.ClearAVXUpperBits()
}
