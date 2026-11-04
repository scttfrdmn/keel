// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// The V3 load probe (archive/v3-load/). Each KernelQ5x16* is the shipped KernelPB5x16 with
// one class of load hoisted out of the K-loop, emitted by archive/v3-load/probe-gen.py, so
// timing them side by side attributes the gap between the shipped kernel and the FMA-only
// rate to A loads, B loads, or neither. Test-only: they reuse operands and are not GEMMs.

import (
	"fmt"
	"math/rand"
	"testing"

	"simd/archsimd"
)

// KernelQ5x16Ctl is a load probe (archive/v3-load/): the control: the shipped KernelPB5x16, verbatim.
func KernelQ5x16Ctl(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float32x4
	var c4_0, c4_1, c4_2, c4_3 archsimd.Float32x4
	var b0, b1, b2, b3, av archsimd.Float32x4
	ap := a[:kc*20]
	bp := b[:kc*16]
	for len(ap) >= 160 && len(bp) >= 128 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[128:132]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[132:136]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[136:140]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[140:144]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[144:148]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[148:152]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[152:156]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[156:160]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[160:], bp[128:]
	}
	for len(ap) >= 20 && len(bp) >= 16 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[20:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	r1 := c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	r2 := c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	r3 := c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	r4 := c[4*ldc : 4*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[12:16])).Add(c4_3).StoreArray((*[4]float32)(r4[12:16]))
}

// KernelQ5x16NA is a load probe (archive/v3-load/): the 5 pre-broadcast A vectors are loaded once before the loop, so the loop has no A loads.
func KernelQ5x16NA(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float32x4
	var c4_0, c4_1, c4_2, c4_3 archsimd.Float32x4
	var b0, b1, b2, b3, av archsimd.Float32x4
	ap := a[:kc*20]
	bp := b[:kc*16]
	a0, a1, a2, a3, a4 := archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
	for len(ap) >= 160 && len(bp) >= 128 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[160:], bp[128:]
	}
	for len(ap) >= 20 && len(bp) >= 16 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[20:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	r1 := c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	r2 := c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	r3 := c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	r4 := c[4*ldc : 4*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[12:16])).Add(c4_3).StoreArray((*[4]float32)(r4[12:16]))
}

// KernelQ5x16NB is a load probe (archive/v3-load/): the 4 B vectors are loaded once before the loop, so the loop has no B loads.
func KernelQ5x16NB(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float32x4
	var c4_0, c4_1, c4_2, c4_3 archsimd.Float32x4
	var b0, b1, b2, b3, av archsimd.Float32x4
	ap := a[:kc*20]
	bp := b[:kc*16]
	b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
	for len(ap) >= 160 && len(bp) >= 128 {
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[128:132]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[132:136]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[136:140]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[140:144]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[144:148]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[148:152]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[152:156]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[156:160]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[160:], bp[128:]
	}
	for len(ap) >= 20 && len(bp) >= 16 {
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[20:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	r1 := c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	r2 := c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	r3 := c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	r4 := c[4*ldc : 4*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[12:16])).Add(c4_3).StoreArray((*[4]float32)(r4[12:16]))
}

// KernelQ5x16FO is a load probe (archive/v3-load/): both hoisted: the loop is the 160 FMAs and its own bookkeeping.
func KernelQ5x16FO(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float32x4
	var c4_0, c4_1, c4_2, c4_3 archsimd.Float32x4
	var b0, b1, b2, b3, av archsimd.Float32x4
	ap := a[:kc*20]
	bp := b[:kc*16]
	b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
	a0, a1, a2, a3, a4 := archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
	for len(ap) >= 160 && len(bp) >= 128 {
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[160:], bp[128:]
	}
	for len(ap) >= 20 && len(bp) >= 16 {
		av = a0
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = a1
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = a2
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = a3
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = a4
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[20:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	r1 := c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	r2 := c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	r3 := c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	r4 := c[4*ldc : 4*ldc+16]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[12:16])).Add(c4_3).StoreArray((*[4]float32)(r4[12:16]))
}

var loadProbeSink float32

// BenchmarkLoadProbe times the four variants at three depths, L1- to L2-resident. GFLOP/s
// counts 2·5·16·kc per call for every variant, so each reads as its loop's FMA rate.
func BenchmarkLoadProbe(b *testing.B) {
	const mr, nr = 5, 16
	for _, kc := range []int{128, 384, 512} {
		for _, p := range []struct {
			name string
			f    func(kc int, a, b, c []float32, ldc int)
		}{
			{"Ctl", KernelQ5x16Ctl}, {"NA", KernelQ5x16NA}, {"NB", KernelQ5x16NB}, {"FO", KernelQ5x16FO},
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
				loadProbeSink = c[0]
				b.ReportMetric(2*mr*nr*float64(kc)*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
			})
		}
	}
}
