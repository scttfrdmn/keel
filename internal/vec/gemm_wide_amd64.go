// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// Experimental wide tiles (branch exp-wide-tiles; not shipped, not in kern.Kernels()). Emitted unchanged
// by tools/shapegen (archive/shapegen-ps2/). Under CL 824624 PS2 both are spill-free with more than 8
// accumulators (5x32: 10, 3x64: 12); on stock go1.27.3 both spill (2 stack refs each). They are
// registered in kern.referenceTiles() so the oracle and scalar-tile tests and BenchmarkKernel run them.

// Kernel5x32 is a generated candidate: 5x32 u=1 broadcast.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func Kernel5x32(kc int, a, b, c []float32, ldc int) {
	var (
		c0l, c0h archsimd.Float32x16
		c1l, c1h archsimd.Float32x16
		c2l, c2h archsimd.Float32x16
		c3l, c3h archsimd.Float32x16
		c4l, c4h archsimd.Float32x16
	)
	var bl, bh, av archsimd.Float32x16

	ap := a[:kc*5]
	bp := b[:kc*32]

	for len(ap) >= 5 && len(bp) >= 32 {
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[4])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		ap, bp = ap[5:], bp[32:]
	}

	r := c[0*ldc : 0*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c0l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c0h).Store(r[16:32])
	r = c[1*ldc : 1*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c1l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c1h).Store(r[16:32])
	r = c[2*ldc : 2*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c2l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c2h).Store(r[16:32])
	r = c[3*ldc : 3*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c3l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c3h).Store(r[16:32])
	r = c[4*ldc : 4*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c4l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c4h).Store(r[16:32])
	archsimd.ClearAVXUpperBits()
}

// Kernel3x64 is a generated candidate: 3x64 u=1 broadcast.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func Kernel3x64(kc int, a, b, c []float32, ldc int) {
	var (
		c0v0, c0v1, c0v2, c0v3 archsimd.Float32x16
		c1v0, c1v1, c1v2, c1v3 archsimd.Float32x16
		c2v0, c2v1, c2v2, c2v3 archsimd.Float32x16
	)
	var b0, b1, b2, b3, av archsimd.Float32x16

	ap := a[:kc*3]
	bp := b[:kc*64]

	for len(ap) >= 3 && len(bp) >= 64 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32]), archsimd.LoadFloat32x16(bp[32:48]), archsimd.LoadFloat32x16(bp[48:64])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0v0, c0v1, c0v2, c0v3 = av.MulAdd(b0, c0v0), av.MulAdd(b1, c0v1), av.MulAdd(b2, c0v2), av.MulAdd(b3, c0v3)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1v0, c1v1, c1v2, c1v3 = av.MulAdd(b0, c1v0), av.MulAdd(b1, c1v1), av.MulAdd(b2, c1v2), av.MulAdd(b3, c1v3)
		av = archsimd.BroadcastFloat32x16(ap[2])
		c2v0, c2v1, c2v2, c2v3 = av.MulAdd(b0, c2v0), av.MulAdd(b1, c2v1), av.MulAdd(b2, c2v2), av.MulAdd(b3, c2v3)
		ap, bp = ap[3:], bp[64:]
	}

	r := c[0*ldc : 0*ldc+64]
	archsimd.LoadFloat32x16(r[0:16]).Add(c0v0).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c0v1).Store(r[16:32])
	archsimd.LoadFloat32x16(r[32:48]).Add(c0v2).Store(r[32:48])
	archsimd.LoadFloat32x16(r[48:64]).Add(c0v3).Store(r[48:64])
	r = c[1*ldc : 1*ldc+64]
	archsimd.LoadFloat32x16(r[0:16]).Add(c1v0).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c1v1).Store(r[16:32])
	archsimd.LoadFloat32x16(r[32:48]).Add(c1v2).Store(r[32:48])
	archsimd.LoadFloat32x16(r[48:64]).Add(c1v3).Store(r[48:64])
	r = c[2*ldc : 2*ldc+64]
	archsimd.LoadFloat32x16(r[0:16]).Add(c2v0).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c2v1).Store(r[16:32])
	archsimd.LoadFloat32x16(r[32:48]).Add(c2v2).Store(r[32:48])
	archsimd.LoadFloat32x16(r[48:64]).Add(c2v3).Store(r[48:64])
	archsimd.ClearAVXUpperBits()
}
