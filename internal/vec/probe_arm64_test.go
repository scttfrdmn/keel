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
