// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// The ~1.8 FMA/cycle limiter probe (#191, archive/neon-raw/). Each KernelP3x24* is the
// raw 3x24/1 body (tools/shapegen -arch arm64 -form raw -emit 3x24/1) with one thing
// taken out of the K-loop, so timing them side by side attributes the gap between the
// kernel and the measured NEON peak. Test-only: these compute different things (PB reads
// a 4x-expanded A, NB/FO reuse one B row), so none is a GEMM and none is checked as one.

import (
	"fmt"
	"math/rand"
	"testing"

	"simd/archsimd"
)

// KernelP3x24Ctl is a limiter probe (#191): the control: raw 3x24/1 as emitted.
func KernelP3x24Ctl(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*3]
	bp := b[:kc*24]
	for len(ap) >= 3 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.BroadcastFloat32x4(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat32x4(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[3:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	r1 := c[1*ldc : 1*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	r2 := c[2*ldc : 2*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
}

// KernelP3x24PB is a limiter probe (#191): A arrives pre-broadcast, one 128-bit load per row instead of an element load, insert and VDUP.
func KernelP3x24PB(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*12] // A pre-broadcast: 4 copies of each element
	bp := b[:kc*24]
	for len(ap) >= 12 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[12:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	r1 := c[1*ldc : 1*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	r2 := c[2*ldc : 2*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
}

// KernelP3x24NB is a limiter probe (#191): B is loaded once before the loop, so the loop has no B loads.
func KernelP3x24NB(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*3]
	bp := b[:kc*24]
	b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
	for len(ap) >= 3 && len(bp) >= 24 {
		av = archsimd.BroadcastFloat32x4(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat32x4(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[3:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	r1 := c[1*ldc : 1*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	r2 := c[2*ldc : 2*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
}

// KernelP3x24PBNB is a limiter probe (#191): both: pre-broadcast A loads and no B loads.
func KernelP3x24PBNB(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*12] // A pre-broadcast: 4 copies of each element
	bp := b[:kc*24]
	b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
	for len(ap) >= 12 && len(bp) >= 24 {
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[12:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	r1 := c[1*ldc : 1*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	r2 := c[2*ldc : 2*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
}

// KernelP3x24FO is a limiter probe (#191): FMA only: A broadcasts and B loads both hoisted; the loop is the 18 FMAs and its own bookkeeping.
func KernelP3x24FO(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*3]
	bp := b[:kc*24]
	b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
	a0, a1, a2 := archsimd.BroadcastFloat32x4(ap[0]), archsimd.BroadcastFloat32x4(ap[1]), archsimd.BroadcastFloat32x4(ap[2])
	for len(ap) >= 3 && len(bp) >= 24 {
		av = a0
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = a1
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = a2
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[3:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	r1 := c[1*ldc : 1*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	r2 := c[2*ldc : 2*ldc+24]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
}

var probeSink float32

// BenchmarkLimiterProbe times the probes at kc=512, L1-resident, beside the shipped shim
// 3x24 as the link to archive/neon-raw/'s run. GFLOP/s counts 2·3·24·kc per call for
// every variant, so each number reads as the FMA rate of its loop.
func BenchmarkLimiterProbe(b *testing.B) {
	const kc, mr, nr = 512, 3, 24
	for _, p := range []struct {
		name string
		f    func(kc int, a, b, c []float32, ldc int)
	}{
		{"shim", Kernel3x24}, {"ctl", KernelP3x24Ctl}, {"PB", KernelP3x24PB},
		{"NB", KernelP3x24NB}, {"PBNB", KernelP3x24PBNB}, {"FO", KernelP3x24FO},
	} {
		b.Run(fmt.Sprintf("%s/kc=%d", p.name, kc), func(b *testing.B) {
			rng := rand.New(rand.NewSource(1))
			ldc := nr + 16
			a, bp, c := make([]float32, 4*kc*mr), make([]float32, kc*nr), make([]float32, mr*ldc)
			for i := range a {
				a[i] = rng.Float32()*2 - 1
			}
			for i := range bp {
				bp[i] = rng.Float32()*2 - 1
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.f(kc, a, bp, c, ldc)
			}
			b.StopTimer()
			probeSink = c[0]
			b.ReportMetric(2*mr*nr*kc*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
}

// rawPeak16 is neonPeak with archsimd called directly: 16 chains, each `a = x.MulAdd(y, a)`,
// so no inlined shim body leaves an anchor NOP between the FMAs.
func rawPeak16(iters int) float32 {
	x, y := archsimd.BroadcastFloat32x4(peakOne()), archsimd.BroadcastFloat32x4(peakOne())
	a0 := archsimd.BroadcastFloat32x4(1)
	a1 := archsimd.BroadcastFloat32x4(2)
	a2 := archsimd.BroadcastFloat32x4(3)
	a3 := archsimd.BroadcastFloat32x4(4)
	a4 := archsimd.BroadcastFloat32x4(5)
	a5 := archsimd.BroadcastFloat32x4(6)
	a6 := archsimd.BroadcastFloat32x4(7)
	a7 := archsimd.BroadcastFloat32x4(8)
	a8 := archsimd.BroadcastFloat32x4(9)
	a9 := archsimd.BroadcastFloat32x4(10)
	a10 := archsimd.BroadcastFloat32x4(11)
	a11 := archsimd.BroadcastFloat32x4(12)
	a12 := archsimd.BroadcastFloat32x4(13)
	a13 := archsimd.BroadcastFloat32x4(14)
	a14 := archsimd.BroadcastFloat32x4(15)
	a15 := archsimd.BroadcastFloat32x4(16)
	for i := 0; i < iters; i++ {
		a0 = x.MulAdd(y, a0)
		a1 = x.MulAdd(y, a1)
		a2 = x.MulAdd(y, a2)
		a3 = x.MulAdd(y, a3)
		a4 = x.MulAdd(y, a4)
		a5 = x.MulAdd(y, a5)
		a6 = x.MulAdd(y, a6)
		a7 = x.MulAdd(y, a7)
		a8 = x.MulAdd(y, a8)
		a9 = x.MulAdd(y, a9)
		a10 = x.MulAdd(y, a10)
		a11 = x.MulAdd(y, a11)
		a12 = x.MulAdd(y, a12)
		a13 = x.MulAdd(y, a13)
		a14 = x.MulAdd(y, a14)
		a15 = x.MulAdd(y, a15)
	}
	s := a0
	s = s.Add(a1)
	s = s.Add(a2)
	s = s.Add(a3)
	s = s.Add(a4)
	s = s.Add(a5)
	s = s.Add(a6)
	s = s.Add(a7)
	s = s.Add(a8)
	s = s.Add(a9)
	s = s.Add(a10)
	s = s.Add(a11)
	s = s.Add(a12)
	s = s.Add(a13)
	s = s.Add(a14)
	s = s.Add(a15)
	return HSum128(s)
}

// rawPeak24 is neonPeak with archsimd called directly: 24 chains, each `a = x.MulAdd(y, a)`,
// so no inlined shim body leaves an anchor NOP between the FMAs.
func rawPeak24(iters int) float32 {
	x, y := archsimd.BroadcastFloat32x4(peakOne()), archsimd.BroadcastFloat32x4(peakOne())
	a0 := archsimd.BroadcastFloat32x4(1)
	a1 := archsimd.BroadcastFloat32x4(2)
	a2 := archsimd.BroadcastFloat32x4(3)
	a3 := archsimd.BroadcastFloat32x4(4)
	a4 := archsimd.BroadcastFloat32x4(5)
	a5 := archsimd.BroadcastFloat32x4(6)
	a6 := archsimd.BroadcastFloat32x4(7)
	a7 := archsimd.BroadcastFloat32x4(8)
	a8 := archsimd.BroadcastFloat32x4(9)
	a9 := archsimd.BroadcastFloat32x4(10)
	a10 := archsimd.BroadcastFloat32x4(11)
	a11 := archsimd.BroadcastFloat32x4(12)
	a12 := archsimd.BroadcastFloat32x4(13)
	a13 := archsimd.BroadcastFloat32x4(14)
	a14 := archsimd.BroadcastFloat32x4(15)
	a15 := archsimd.BroadcastFloat32x4(16)
	a16 := archsimd.BroadcastFloat32x4(17)
	a17 := archsimd.BroadcastFloat32x4(18)
	a18 := archsimd.BroadcastFloat32x4(19)
	a19 := archsimd.BroadcastFloat32x4(20)
	a20 := archsimd.BroadcastFloat32x4(21)
	a21 := archsimd.BroadcastFloat32x4(22)
	a22 := archsimd.BroadcastFloat32x4(23)
	a23 := archsimd.BroadcastFloat32x4(24)
	for i := 0; i < iters; i++ {
		a0 = x.MulAdd(y, a0)
		a1 = x.MulAdd(y, a1)
		a2 = x.MulAdd(y, a2)
		a3 = x.MulAdd(y, a3)
		a4 = x.MulAdd(y, a4)
		a5 = x.MulAdd(y, a5)
		a6 = x.MulAdd(y, a6)
		a7 = x.MulAdd(y, a7)
		a8 = x.MulAdd(y, a8)
		a9 = x.MulAdd(y, a9)
		a10 = x.MulAdd(y, a10)
		a11 = x.MulAdd(y, a11)
		a12 = x.MulAdd(y, a12)
		a13 = x.MulAdd(y, a13)
		a14 = x.MulAdd(y, a14)
		a15 = x.MulAdd(y, a15)
		a16 = x.MulAdd(y, a16)
		a17 = x.MulAdd(y, a17)
		a18 = x.MulAdd(y, a18)
		a19 = x.MulAdd(y, a19)
		a20 = x.MulAdd(y, a20)
		a21 = x.MulAdd(y, a21)
		a22 = x.MulAdd(y, a22)
		a23 = x.MulAdd(y, a23)
	}
	s := a0
	s = s.Add(a1)
	s = s.Add(a2)
	s = s.Add(a3)
	s = s.Add(a4)
	s = s.Add(a5)
	s = s.Add(a6)
	s = s.Add(a7)
	s = s.Add(a8)
	s = s.Add(a9)
	s = s.Add(a10)
	s = s.Add(a11)
	s = s.Add(a12)
	s = s.Add(a13)
	s = s.Add(a14)
	s = s.Add(a15)
	s = s.Add(a16)
	s = s.Add(a17)
	s = s.Add(a18)
	s = s.Add(a19)
	s = s.Add(a20)
	s = s.Add(a21)
	s = s.Add(a22)
	s = s.Add(a23)
	return HSum128(s)
}

// rawPeakD16 is neonPeakD with archsimd called directly: 16 chains, each `a = x.MulAdd(y, a)`,
// so no inlined shim body leaves an anchor NOP between the FMAs.
func rawPeakD16(iters int) float64 {
	x, y := archsimd.BroadcastFloat64x2(peakOneD()), archsimd.BroadcastFloat64x2(peakOneD())
	a0 := archsimd.BroadcastFloat64x2(1)
	a1 := archsimd.BroadcastFloat64x2(2)
	a2 := archsimd.BroadcastFloat64x2(3)
	a3 := archsimd.BroadcastFloat64x2(4)
	a4 := archsimd.BroadcastFloat64x2(5)
	a5 := archsimd.BroadcastFloat64x2(6)
	a6 := archsimd.BroadcastFloat64x2(7)
	a7 := archsimd.BroadcastFloat64x2(8)
	a8 := archsimd.BroadcastFloat64x2(9)
	a9 := archsimd.BroadcastFloat64x2(10)
	a10 := archsimd.BroadcastFloat64x2(11)
	a11 := archsimd.BroadcastFloat64x2(12)
	a12 := archsimd.BroadcastFloat64x2(13)
	a13 := archsimd.BroadcastFloat64x2(14)
	a14 := archsimd.BroadcastFloat64x2(15)
	a15 := archsimd.BroadcastFloat64x2(16)
	for i := 0; i < iters; i++ {
		a0 = x.MulAdd(y, a0)
		a1 = x.MulAdd(y, a1)
		a2 = x.MulAdd(y, a2)
		a3 = x.MulAdd(y, a3)
		a4 = x.MulAdd(y, a4)
		a5 = x.MulAdd(y, a5)
		a6 = x.MulAdd(y, a6)
		a7 = x.MulAdd(y, a7)
		a8 = x.MulAdd(y, a8)
		a9 = x.MulAdd(y, a9)
		a10 = x.MulAdd(y, a10)
		a11 = x.MulAdd(y, a11)
		a12 = x.MulAdd(y, a12)
		a13 = x.MulAdd(y, a13)
		a14 = x.MulAdd(y, a14)
		a15 = x.MulAdd(y, a15)
	}
	s := a0
	s = s.Add(a1)
	s = s.Add(a2)
	s = s.Add(a3)
	s = s.Add(a4)
	s = s.Add(a5)
	s = s.Add(a6)
	s = s.Add(a7)
	s = s.Add(a8)
	s = s.Add(a9)
	s = s.Add(a10)
	s = s.Add(a11)
	s = s.Add(a12)
	s = s.Add(a13)
	s = s.Add(a14)
	s = s.Add(a15)
	return HSum128D(s)
}

// BenchmarkPeakProbe is the denominator check (#191): the shipped neonPeak/neonPeakD
// beside their raw-archsimd twins, in GFLOP/s, chains x lanes x 2 flops per iteration.
// If the raw twins read above the shim ones, the shipped ceiling is NOP-bound.
func BenchmarkPeakProbe(b *testing.B) {
	const iters = 1 << 16
	for _, p := range []struct {
		name         string
		flopsPerIter int
		f32          func(int) float32
		f64          func(int) float64
	}{
		{"shim16", 16 * 4 * 2, neonPeak, nil},
		{"raw16", 16 * 4 * 2, rawPeak16, nil},
		{"raw24", 24 * 4 * 2, rawPeak24, nil},
		{"shimD16", 16 * 2 * 2, nil, neonPeakD},
		{"rawD16", 16 * 2 * 2, nil, rawPeakD16},
	} {
		b.Run(p.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if p.f32 != nil {
					probeSink += p.f32(iters)
				} else {
					probeSink += float32(p.f64(iters))
				}
			}
			b.ReportMetric(float64(p.flopsPerIter)*iters*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
}

// pkVals: 16 chains reading one x and one y register as neonPeak does, but operands 0.75 and 1.25 instead of 1.0.
func pkVals(iters int) float32 {
	x, y := archsimd.BroadcastFloat32x4(0.75*peakOne()), archsimd.BroadcastFloat32x4(1.25*peakOne())
	a0 := archsimd.BroadcastFloat32x4(1)
	a1 := archsimd.BroadcastFloat32x4(2)
	a2 := archsimd.BroadcastFloat32x4(3)
	a3 := archsimd.BroadcastFloat32x4(4)
	a4 := archsimd.BroadcastFloat32x4(5)
	a5 := archsimd.BroadcastFloat32x4(6)
	a6 := archsimd.BroadcastFloat32x4(7)
	a7 := archsimd.BroadcastFloat32x4(8)
	a8 := archsimd.BroadcastFloat32x4(9)
	a9 := archsimd.BroadcastFloat32x4(10)
	a10 := archsimd.BroadcastFloat32x4(11)
	a11 := archsimd.BroadcastFloat32x4(12)
	a12 := archsimd.BroadcastFloat32x4(13)
	a13 := archsimd.BroadcastFloat32x4(14)
	a14 := archsimd.BroadcastFloat32x4(15)
	a15 := archsimd.BroadcastFloat32x4(16)
	for i := 0; i < iters; i++ {
		a0 = x.MulAdd(y, a0)
		a1 = x.MulAdd(y, a1)
		a2 = x.MulAdd(y, a2)
		a3 = x.MulAdd(y, a3)
		a4 = x.MulAdd(y, a4)
		a5 = x.MulAdd(y, a5)
		a6 = x.MulAdd(y, a6)
		a7 = x.MulAdd(y, a7)
		a8 = x.MulAdd(y, a8)
		a9 = x.MulAdd(y, a9)
		a10 = x.MulAdd(y, a10)
		a11 = x.MulAdd(y, a11)
		a12 = x.MulAdd(y, a12)
		a13 = x.MulAdd(y, a13)
		a14 = x.MulAdd(y, a14)
		a15 = x.MulAdd(y, a15)
	}
	s := a0
	s = s.Add(a1)
	s = s.Add(a2)
	s = s.Add(a3)
	s = s.Add(a4)
	s = s.Add(a5)
	s = s.Add(a6)
	s = s.Add(a7)
	s = s.Add(a8)
	s = s.Add(a9)
	s = s.Add(a10)
	s = s.Add(a11)
	s = s.Add(a12)
	s = s.Add(a13)
	s = s.Add(a14)
	s = s.Add(a15)
	return HSum128(s)
}

// pkX44: 16 chains in a 4x4 outer product, so the FMAs read 8 distinct multiplicand registers; operands 1.0.
func pkX44(iters int) float32 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat32x4(peakOne()), archsimd.BroadcastFloat32x4(peakOne()+1.0/1024), archsimd.BroadcastFloat32x4(peakOne()+2.0/1024), archsimd.BroadcastFloat32x4(peakOne()+3.0/1024)
	y0, y1, y2, y3 := archsimd.BroadcastFloat32x4(peakOne()+0.5/1024), archsimd.BroadcastFloat32x4(peakOne()+1.5/1024), archsimd.BroadcastFloat32x4(peakOne()+2.5/1024), archsimd.BroadcastFloat32x4(peakOne()+3.5/1024)
	a0_0 := archsimd.BroadcastFloat32x4(1)
	a0_1 := archsimd.BroadcastFloat32x4(2)
	a0_2 := archsimd.BroadcastFloat32x4(3)
	a0_3 := archsimd.BroadcastFloat32x4(4)
	a1_0 := archsimd.BroadcastFloat32x4(5)
	a1_1 := archsimd.BroadcastFloat32x4(6)
	a1_2 := archsimd.BroadcastFloat32x4(7)
	a1_3 := archsimd.BroadcastFloat32x4(8)
	a2_0 := archsimd.BroadcastFloat32x4(9)
	a2_1 := archsimd.BroadcastFloat32x4(10)
	a2_2 := archsimd.BroadcastFloat32x4(11)
	a2_3 := archsimd.BroadcastFloat32x4(12)
	a3_0 := archsimd.BroadcastFloat32x4(13)
	a3_1 := archsimd.BroadcastFloat32x4(14)
	a3_2 := archsimd.BroadcastFloat32x4(15)
	a3_3 := archsimd.BroadcastFloat32x4(16)
	for i := 0; i < iters; i++ {
		a0_0 = x0.MulAdd(y0, a0_0)
		a0_1 = x0.MulAdd(y1, a0_1)
		a0_2 = x0.MulAdd(y2, a0_2)
		a0_3 = x0.MulAdd(y3, a0_3)
		a1_0 = x1.MulAdd(y0, a1_0)
		a1_1 = x1.MulAdd(y1, a1_1)
		a1_2 = x1.MulAdd(y2, a1_2)
		a1_3 = x1.MulAdd(y3, a1_3)
		a2_0 = x2.MulAdd(y0, a2_0)
		a2_1 = x2.MulAdd(y1, a2_1)
		a2_2 = x2.MulAdd(y2, a2_2)
		a2_3 = x2.MulAdd(y3, a2_3)
		a3_0 = x3.MulAdd(y0, a3_0)
		a3_1 = x3.MulAdd(y1, a3_1)
		a3_2 = x3.MulAdd(y2, a3_2)
		a3_3 = x3.MulAdd(y3, a3_3)
	}
	s := a0_0
	s = s.Add(a0_1)
	s = s.Add(a0_2)
	s = s.Add(a0_3)
	s = s.Add(a1_0)
	s = s.Add(a1_1)
	s = s.Add(a1_2)
	s = s.Add(a1_3)
	s = s.Add(a2_0)
	s = s.Add(a2_1)
	s = s.Add(a2_2)
	s = s.Add(a2_3)
	s = s.Add(a3_0)
	s = s.Add(a3_1)
	s = s.Add(a3_2)
	s = s.Add(a3_3)
	return HSum128(s)
}

// pkX44v: pkX44 with operands 0.75 and 1.25.
func pkX44v(iters int) float32 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat32x4(0.75*peakOne()), archsimd.BroadcastFloat32x4(0.75*peakOne()+1.0/1024), archsimd.BroadcastFloat32x4(0.75*peakOne()+2.0/1024), archsimd.BroadcastFloat32x4(0.75*peakOne()+3.0/1024)
	y0, y1, y2, y3 := archsimd.BroadcastFloat32x4(1.25*peakOne()), archsimd.BroadcastFloat32x4(1.25*peakOne()+1.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+2.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+3.0/1024)
	a0_0 := archsimd.BroadcastFloat32x4(1)
	a0_1 := archsimd.BroadcastFloat32x4(2)
	a0_2 := archsimd.BroadcastFloat32x4(3)
	a0_3 := archsimd.BroadcastFloat32x4(4)
	a1_0 := archsimd.BroadcastFloat32x4(5)
	a1_1 := archsimd.BroadcastFloat32x4(6)
	a1_2 := archsimd.BroadcastFloat32x4(7)
	a1_3 := archsimd.BroadcastFloat32x4(8)
	a2_0 := archsimd.BroadcastFloat32x4(9)
	a2_1 := archsimd.BroadcastFloat32x4(10)
	a2_2 := archsimd.BroadcastFloat32x4(11)
	a2_3 := archsimd.BroadcastFloat32x4(12)
	a3_0 := archsimd.BroadcastFloat32x4(13)
	a3_1 := archsimd.BroadcastFloat32x4(14)
	a3_2 := archsimd.BroadcastFloat32x4(15)
	a3_3 := archsimd.BroadcastFloat32x4(16)
	for i := 0; i < iters; i++ {
		a0_0 = x0.MulAdd(y0, a0_0)
		a0_1 = x0.MulAdd(y1, a0_1)
		a0_2 = x0.MulAdd(y2, a0_2)
		a0_3 = x0.MulAdd(y3, a0_3)
		a1_0 = x1.MulAdd(y0, a1_0)
		a1_1 = x1.MulAdd(y1, a1_1)
		a1_2 = x1.MulAdd(y2, a1_2)
		a1_3 = x1.MulAdd(y3, a1_3)
		a2_0 = x2.MulAdd(y0, a2_0)
		a2_1 = x2.MulAdd(y1, a2_1)
		a2_2 = x2.MulAdd(y2, a2_2)
		a2_3 = x2.MulAdd(y3, a2_3)
		a3_0 = x3.MulAdd(y0, a3_0)
		a3_1 = x3.MulAdd(y1, a3_1)
		a3_2 = x3.MulAdd(y2, a3_2)
		a3_3 = x3.MulAdd(y3, a3_3)
	}
	s := a0_0
	s = s.Add(a0_1)
	s = s.Add(a0_2)
	s = s.Add(a0_3)
	s = s.Add(a1_0)
	s = s.Add(a1_1)
	s = s.Add(a1_2)
	s = s.Add(a1_3)
	s = s.Add(a2_0)
	s = s.Add(a2_1)
	s = s.Add(a2_2)
	s = s.Add(a2_3)
	s = s.Add(a3_0)
	s = s.Add(a3_1)
	s = s.Add(a3_2)
	s = s.Add(a3_3)
	return HSum128(s)
}

// pkX36: FO's 18-chain 3x6 shape on a counter loop: no panel slicing, non-unit operands.
func pkX36(iters int) float32 {
	x0, x1, x2 := archsimd.BroadcastFloat32x4(0.75*peakOne()), archsimd.BroadcastFloat32x4(0.75*peakOne()+1.0/1024), archsimd.BroadcastFloat32x4(0.75*peakOne()+2.0/1024)
	y0, y1, y2, y3, y4, y5 := archsimd.BroadcastFloat32x4(1.25*peakOne()), archsimd.BroadcastFloat32x4(1.25*peakOne()+1.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+2.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+3.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+4.0/1024), archsimd.BroadcastFloat32x4(1.25*peakOne()+5.0/1024)
	a0_0 := archsimd.BroadcastFloat32x4(1)
	a0_1 := archsimd.BroadcastFloat32x4(2)
	a0_2 := archsimd.BroadcastFloat32x4(3)
	a0_3 := archsimd.BroadcastFloat32x4(4)
	a0_4 := archsimd.BroadcastFloat32x4(5)
	a0_5 := archsimd.BroadcastFloat32x4(6)
	a1_0 := archsimd.BroadcastFloat32x4(7)
	a1_1 := archsimd.BroadcastFloat32x4(8)
	a1_2 := archsimd.BroadcastFloat32x4(9)
	a1_3 := archsimd.BroadcastFloat32x4(10)
	a1_4 := archsimd.BroadcastFloat32x4(11)
	a1_5 := archsimd.BroadcastFloat32x4(12)
	a2_0 := archsimd.BroadcastFloat32x4(13)
	a2_1 := archsimd.BroadcastFloat32x4(14)
	a2_2 := archsimd.BroadcastFloat32x4(15)
	a2_3 := archsimd.BroadcastFloat32x4(16)
	a2_4 := archsimd.BroadcastFloat32x4(17)
	a2_5 := archsimd.BroadcastFloat32x4(18)
	for i := 0; i < iters; i++ {
		a0_0 = x0.MulAdd(y0, a0_0)
		a0_1 = x0.MulAdd(y1, a0_1)
		a0_2 = x0.MulAdd(y2, a0_2)
		a0_3 = x0.MulAdd(y3, a0_3)
		a0_4 = x0.MulAdd(y4, a0_4)
		a0_5 = x0.MulAdd(y5, a0_5)
		a1_0 = x1.MulAdd(y0, a1_0)
		a1_1 = x1.MulAdd(y1, a1_1)
		a1_2 = x1.MulAdd(y2, a1_2)
		a1_3 = x1.MulAdd(y3, a1_3)
		a1_4 = x1.MulAdd(y4, a1_4)
		a1_5 = x1.MulAdd(y5, a1_5)
		a2_0 = x2.MulAdd(y0, a2_0)
		a2_1 = x2.MulAdd(y1, a2_1)
		a2_2 = x2.MulAdd(y2, a2_2)
		a2_3 = x2.MulAdd(y3, a2_3)
		a2_4 = x2.MulAdd(y4, a2_4)
		a2_5 = x2.MulAdd(y5, a2_5)
	}
	s := a0_0
	s = s.Add(a0_1)
	s = s.Add(a0_2)
	s = s.Add(a0_3)
	s = s.Add(a0_4)
	s = s.Add(a0_5)
	s = s.Add(a1_0)
	s = s.Add(a1_1)
	s = s.Add(a1_2)
	s = s.Add(a1_3)
	s = s.Add(a1_4)
	s = s.Add(a1_5)
	s = s.Add(a2_0)
	s = s.Add(a2_1)
	s = s.Add(a2_2)
	s = s.Add(a2_3)
	s = s.Add(a2_4)
	s = s.Add(a2_5)
	return HSum128(s)
}

// pkDX44: pkX44v on Float64x2.
func pkDX44(iters int) float64 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat64x2(0.75*peakOneD()), archsimd.BroadcastFloat64x2(0.75*peakOneD()+1.0/1024), archsimd.BroadcastFloat64x2(0.75*peakOneD()+2.0/1024), archsimd.BroadcastFloat64x2(0.75*peakOneD()+3.0/1024)
	y0, y1, y2, y3 := archsimd.BroadcastFloat64x2(1.25*peakOneD()), archsimd.BroadcastFloat64x2(1.25*peakOneD()+1.0/1024), archsimd.BroadcastFloat64x2(1.25*peakOneD()+2.0/1024), archsimd.BroadcastFloat64x2(1.25*peakOneD()+3.0/1024)
	a0_0 := archsimd.BroadcastFloat64x2(1)
	a0_1 := archsimd.BroadcastFloat64x2(2)
	a0_2 := archsimd.BroadcastFloat64x2(3)
	a0_3 := archsimd.BroadcastFloat64x2(4)
	a1_0 := archsimd.BroadcastFloat64x2(5)
	a1_1 := archsimd.BroadcastFloat64x2(6)
	a1_2 := archsimd.BroadcastFloat64x2(7)
	a1_3 := archsimd.BroadcastFloat64x2(8)
	a2_0 := archsimd.BroadcastFloat64x2(9)
	a2_1 := archsimd.BroadcastFloat64x2(10)
	a2_2 := archsimd.BroadcastFloat64x2(11)
	a2_3 := archsimd.BroadcastFloat64x2(12)
	a3_0 := archsimd.BroadcastFloat64x2(13)
	a3_1 := archsimd.BroadcastFloat64x2(14)
	a3_2 := archsimd.BroadcastFloat64x2(15)
	a3_3 := archsimd.BroadcastFloat64x2(16)
	for i := 0; i < iters; i++ {
		a0_0 = x0.MulAdd(y0, a0_0)
		a0_1 = x0.MulAdd(y1, a0_1)
		a0_2 = x0.MulAdd(y2, a0_2)
		a0_3 = x0.MulAdd(y3, a0_3)
		a1_0 = x1.MulAdd(y0, a1_0)
		a1_1 = x1.MulAdd(y1, a1_1)
		a1_2 = x1.MulAdd(y2, a1_2)
		a1_3 = x1.MulAdd(y3, a1_3)
		a2_0 = x2.MulAdd(y0, a2_0)
		a2_1 = x2.MulAdd(y1, a2_1)
		a2_2 = x2.MulAdd(y2, a2_2)
		a2_3 = x2.MulAdd(y3, a2_3)
		a3_0 = x3.MulAdd(y0, a3_0)
		a3_1 = x3.MulAdd(y1, a3_1)
		a3_2 = x3.MulAdd(y2, a3_2)
		a3_3 = x3.MulAdd(y3, a3_3)
	}
	s := a0_0
	s = s.Add(a0_1)
	s = s.Add(a0_2)
	s = s.Add(a0_3)
	s = s.Add(a1_0)
	s = s.Add(a1_1)
	s = s.Add(a1_2)
	s = s.Add(a1_3)
	s = s.Add(a2_0)
	s = s.Add(a2_1)
	s = s.Add(a2_2)
	s = s.Add(a2_3)
	s = s.Add(a3_0)
	s = s.Add(a3_1)
	s = s.Add(a3_2)
	s = s.Add(a3_3)
	return HSum128D(s)
}

// BenchmarkPeakVariants is the denominator probe (#191, archive/neon-raw/): neonPeak's raw
// twin beside variants that each change one property, in GFLOP/s at chains x lanes x 2
// flops per iteration.
func BenchmarkPeakVariants(b *testing.B) {
	const iters = 1 << 16
	for _, p := range []struct {
		name         string
		flopsPerIter int
		f32          func(int) float32
		f64          func(int) float64
	}{
		{"raw16", 16 * 4 * 2, rawPeak16, nil},
		{"vals", 16 * 4 * 2, pkVals, nil},
		{"x44", 16 * 4 * 2, pkX44, nil},
		{"x44v", 16 * 4 * 2, pkX44v, nil},
		{"x36", 18 * 4 * 2, pkX36, nil},
		{"rawD16", 16 * 2 * 2, nil, rawPeakD16},
		{"Dx44", 16 * 2 * 2, nil, pkDX44},
	} {
		b.Run(p.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if p.f32 != nil {
					probeSink += p.f32(iters)
				} else {
					probeSink += float32(p.f64(iters))
				}
			}
			b.ReportMetric(float64(p.flopsPerIter)*iters*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
}
