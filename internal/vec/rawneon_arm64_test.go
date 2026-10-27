// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// The NEON shim-removal experiment (#191): tools/shapegen's `-form raw` kernels,
// which call archsimd directly instead of through the vec shim, beside the shipped
// shim kernels, so one benchmark binary times both on one host. Test-only on purpose:
// nothing here ships, and no dispatch can see it, until a ruling adopts the form.
// Bodies are the emitter's output with the package clause rewritten; regenerate with
//   go run ./tools/shapegen -arch arm64 -elem {f32|f64} -form raw -emit MRxNR/U

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"simd/archsimd"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// KernelR3x24U1 is a generated candidate: 3x24 u=1 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelR3x24U1(kc int, a, b, c []float32, ldc int) {
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

// KernelR2x24U2 is a generated candidate: 2x24 u=2 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelR2x24U2(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*2]
	bp := b[:kc*24]
	for len(ap) >= 4 && len(bp) >= 48 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.BroadcastFloat32x4(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.BroadcastFloat32x4(ap[2])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[3])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		ap, bp = ap[4:], bp[48:]
	}
	for len(ap) >= 2 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.BroadcastFloat32x4(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		ap, bp = ap[2:], bp[24:]
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
}

// KernelR3x24U2 is a generated candidate: 3x24 u=2 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelR3x24U2(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*3]
	bp := b[:kc*24]
	for len(ap) >= 6 && len(bp) >= 48 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.BroadcastFloat32x4(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat32x4(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.BroadcastFloat32x4(ap[3])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat32x4(ap[4])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat32x4(ap[5])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[6:], bp[48:]
	}
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

// KernelDR3x12U1 is a generated candidate: 3x12 u=1 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDR3x12U1(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*3]
	bp := b[:kc*12]
	for len(ap) >= 3 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.BroadcastFloat64x2(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat64x2(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[3:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[10:12])).Add(c0_5).StoreArray((*[2]float64)(r0[10:12]))
	r1 := c[1*ldc : 1*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[10:12])).Add(c1_5).StoreArray((*[2]float64)(r1[10:12]))
	r2 := c[2*ldc : 2*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[8:10])).Add(c2_4).StoreArray((*[2]float64)(r2[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[10:12])).Add(c2_5).StoreArray((*[2]float64)(r2[10:12]))
}

// KernelDR2x12U2 is a generated candidate: 2x12 u=2 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDR2x12U2(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*2]
	bp := b[:kc*12]
	for len(ap) >= 4 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.BroadcastFloat64x2(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.BroadcastFloat64x2(ap[2])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[3])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		ap, bp = ap[4:], bp[24:]
	}
	for len(ap) >= 2 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.BroadcastFloat64x2(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		ap, bp = ap[2:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[10:12])).Add(c0_5).StoreArray((*[2]float64)(r0[10:12]))
	r1 := c[1*ldc : 1*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[10:12])).Add(c1_5).StoreArray((*[2]float64)(r1[10:12]))
}

// KernelDR3x12U2 is a generated candidate: 3x12 u=2 raw.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDR3x12U2(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*3]
	bp := b[:kc*12]
	for len(ap) >= 6 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.BroadcastFloat64x2(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat64x2(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.BroadcastFloat64x2(ap[3])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[4])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat64x2(ap[5])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[6:], bp[24:]
	}
	for len(ap) >= 3 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.BroadcastFloat64x2(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.BroadcastFloat64x2(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.BroadcastFloat64x2(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[3:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[10:12])).Add(c0_5).StoreArray((*[2]float64)(r0[10:12]))
	r1 := c[1*ldc : 1*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[10:12])).Add(c1_5).StoreArray((*[2]float64)(r1[10:12]))
	r2 := c[2*ldc : 2*ldc+12]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[8:10])).Add(c2_4).StoreArray((*[2]float64)(r2[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[10:12])).Add(c2_5).StoreArray((*[2]float64)(r2[10:12]))
}

type rawPair struct {
	name   string
	mr, nr int
	f32    func(kc int, a, b, c []float32, ldc int)
	f64    func(kc int, a, b, c []float64, ldc int)
}

var rawPairs = []rawPair{
	{"shim/3x24u2", 3, 24, Kernel3x24, nil},
	{"raw/3x24u1", 3, 24, KernelR3x24U1, nil},
	{"raw/2x24u2", 2, 24, KernelR2x24U2, nil},
	{"shim/D3x12u2", 3, 12, nil, KernelD3x12},
	{"raw/D3x12u1", 3, 12, nil, KernelDR3x12U1},
	{"raw/D2x12u2", 2, 12, nil, KernelDR2x12U2},
}

// TestRawNEONKernels holds every raw kernel to a float64 definition-order reference
// over C += A·B, with a guard column past NR that must come back untouched.
func TestRawNEONKernels(t *testing.T) {
	ks := append(rawPairs[:0:0], rawPairs...)
	ks = append(ks, rawPair{"raw/3x24u2", 3, 24, KernelR3x24U2, nil}, rawPair{"raw/D3x12u2", 3, 12, nil, KernelDR3x12U2})
	for _, p := range ks {
		for _, kc := range []int{1, 2, 3, 8, 17} {
			rng := rand.New(rand.NewSource(int64(kc)))
			ldc := p.nr + 3
			a, b := make([]float64, kc*p.mr), make([]float64, kc*p.nr)
			c := make([]float64, (p.mr-1)*ldc+p.nr+3)
			for _, s := range [][]float64{a, b, c} {
				for i := range s {
					s[i] = 2*rng.Float64() - 1
				}
			}
			got := append([]float64(nil), c...)
			if p.f32 != nil {
				a32, b32, c32 := to32(a), to32(b), to32(c)
				for i := range a {
					a[i] = float64(a32[i]) // the reference sees the rounded inputs
				}
				for i := range b {
					b[i] = float64(b32[i])
				}
				for i := range c {
					c[i] = float64(c32[i])
				}
				p.f32(kc, a32, b32, c32, ldc)
				for i := range got {
					got[i] = float64(c32[i])
				}
			} else {
				p.f64(kc, a, b, got, ldc)
			}
			for i := range c {
				row, col := i/ldc, i%ldc
				if row >= p.mr || col >= p.nr {
					if got[i] != c[i] {
						t.Fatalf("%s kc=%d: wrote c[%d], outside the tile", p.name, kc, i)
					}
					continue
				}
				want, scale := c[i], math.Abs(c[i])
				for q := 0; q < kc; q++ {
					want += a[q*p.mr+row] * b[q*p.nr+col]
					scale += math.Abs(a[q*p.mr+row] * b[q*p.nr+col])
				}
				tol := oracle.Tolerance(kc+1, scale)
				if p.f64 != nil {
					tol = oracle.Tolerance64(kc+1, scale)
				}
				if d := math.Abs(got[i] - want); !(d <= tol) {
					t.Fatalf("%s kc=%d: C[%d][%d] = %v, want %v (|diff| %g > %g)", p.name, kc, row, col, got[i], want, d, tol)
				}
			}
		}
	}
}

func to32(s []float64) []float32 {
	o := make([]float32, len(s))
	for i, v := range s {
		o[i] = float32(v)
	}
	return o
}

var rawSink float64

// BenchmarkRawNEON times the raw kernels against the shipped shim kernels of each
// element, L1-resident, ldc padded by one cache line as bench.BenchmarkKernel pads.
// Run with -count and compare within an element: the shipped kernel is the control.
func BenchmarkRawNEON(b *testing.B) {
	for _, p := range rawPairs {
		for _, kc := range []int{128, 512} {
			b.Run(fmt.Sprintf("%s/kc=%d", p.name, kc), func(b *testing.B) {
				rng := rand.New(rand.NewSource(1))
				flops := 2.0 * float64(p.mr*p.nr*kc)
				if p.f32 != nil {
					ldc := p.nr + 16
					a, bp, c := make([]float32, kc*p.mr), make([]float32, kc*p.nr), make([]float32, p.mr*ldc)
					for i := range a {
						a[i] = rng.Float32()*2 - 1
					}
					for i := range bp {
						bp[i] = rng.Float32()*2 - 1
					}
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						p.f32(kc, a, bp, c, ldc)
					}
					b.StopTimer()
					rawSink = float64(c[0])
				} else {
					ldc := p.nr + 8
					a, bp, c := make([]float64, kc*p.mr), make([]float64, kc*p.nr), make([]float64, p.mr*ldc)
					for i := range a {
						a[i] = rng.Float64()*2 - 1
					}
					for i := range bp {
						bp[i] = rng.Float64()*2 - 1
					}
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						p.f64(kc, a, bp, c, ldc)
					}
					b.StopTimer()
					rawSink = c[0]
				}
				b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
			})
		}
	}
}
