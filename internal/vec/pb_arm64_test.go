// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// Pre-broadcast-A candidates (#198, the shaping step): tools/shapegen -form pb emissions,
// held to a definition-order reference and timed side by side, so the shapes that ship
// are chosen by measured rate (the selection rule is archive/neon-pb/prereg.md). Test-only:
// nothing dispatches a PB kernel until the packing path that feeds it lands.
// Regenerate with: go run ./tools/shapegen -arch arm64 -elem {f32|f64} -form pb -emit MRxNR/U

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"simd/archsimd"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// KernelPB4x20U8 is a generated candidate: 4x20 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB4x20U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3, c3_4 archsimd.Float32x4
	var b0, b1, b2, b3, b4, av archsimd.Float32x4
	ap := a[:kc*16]
	bp := b[:kc*20]
	for len(ap) >= 128 && len(bp) >= 160 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[128:132])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[132:136])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[136:140]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[140:144])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[144:148])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[148:152])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[152:156])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[156:160]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[128:], bp[160:]
	}
	for len(ap) >= 16 && len(bp) >= 20 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[16:], bp[20:]
	}
	r0 := c[0*ldc : 0*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	r1 := c[1*ldc : 1*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	r2 := c[2*ldc : 2*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	r3 := c[3*ldc : 3*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[16:20])).Add(c3_4).StoreArray((*[4]float32)(r3[16:20]))
}

// KernelPB5x16U8 is a generated candidate: 5x16 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB5x16U8(kc int, a, b, c []float32, ldc int) {
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

// KernelPB3x28U8 is a generated candidate: 3x28 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB3x28U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, b6, av archsimd.Float32x4
	ap := a[:kc*12]
	bp := b[:kc*28]
	for len(ap) >= 96 && len(bp) >= 224 {
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[128:132])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[132:136])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[136:140]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[140:144])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[144:148])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[148:152])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[152:156])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[156:160])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[160:164])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[164:168]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[168:172])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[172:176])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[176:180])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[180:184])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[184:188])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[188:192])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[192:196]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[196:200])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[200:204])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[204:208])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[208:212])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[212:216])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[216:220])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[220:224]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		ap, bp = ap[96:], bp[224:]
	}
	for len(ap) >= 12 && len(bp) >= 28 {
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		ap, bp = ap[12:], bp[28:]
	}
	r0 := c[0*ldc : 0*ldc+28]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[20:24])).Add(c0_5).StoreArray((*[4]float32)(r0[20:24]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[24:28])).Add(c0_6).StoreArray((*[4]float32)(r0[24:28]))
	r1 := c[1*ldc : 1*ldc+28]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[20:24])).Add(c1_5).StoreArray((*[4]float32)(r1[20:24]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[24:28])).Add(c1_6).StoreArray((*[4]float32)(r1[24:28]))
	r2 := c[2*ldc : 2*ldc+28]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[20:24])).Add(c2_5).StoreArray((*[4]float32)(r2[20:24]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[24:28])).Add(c2_6).StoreArray((*[4]float32)(r2[24:28]))
}

// KernelPB7x12U8 is a generated candidate: 7x12 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB7x12U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2 archsimd.Float32x4
	var c1_0, c1_1, c1_2 archsimd.Float32x4
	var c2_0, c2_1, c2_2 archsimd.Float32x4
	var c3_0, c3_1, c3_2 archsimd.Float32x4
	var c4_0, c4_1, c4_2 archsimd.Float32x4
	var c5_0, c5_1, c5_2 archsimd.Float32x4
	var c6_0, c6_1, c6_2 archsimd.Float32x4
	var b0, b1, b2, av archsimd.Float32x4
	ap := a[:kc*28]
	bp := b[:kc*12]
	for len(ap) >= 224 && len(bp) >= 96 {
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[128:132]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[132:136]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[136:140]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[140:144]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[144:148]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[148:152]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[152:156]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[156:160]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[160:164]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[164:168]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[168:172]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[172:176]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[176:180]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[180:184]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[184:188]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[188:192]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[192:196]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[196:200]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[200:204]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[204:208]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[208:212]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[212:216]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[216:220]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[220:224]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		ap, bp = ap[224:], bp[96:]
	}
	for len(ap) >= 28 && len(bp) >= 12 {
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		ap, bp = ap[28:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	r1 := c[1*ldc : 1*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	r2 := c[2*ldc : 2*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	r3 := c[3*ldc : 3*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	r4 := c[4*ldc : 4*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	r5 := c[5*ldc : 5*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[0:4])).Add(c5_0).StoreArray((*[4]float32)(r5[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[4:8])).Add(c5_1).StoreArray((*[4]float32)(r5[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[8:12])).Add(c5_2).StoreArray((*[4]float32)(r5[8:12]))
	r6 := c[6*ldc : 6*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r6[0:4])).Add(c6_0).StoreArray((*[4]float32)(r6[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r6[4:8])).Add(c6_1).StoreArray((*[4]float32)(r6[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r6[8:12])).Add(c6_2).StoreArray((*[4]float32)(r6[8:12]))
}

// KernelPB4x16U8 is a generated candidate: 4x16 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB4x16U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float32x4
	var b0, b1, b2, b3, av archsimd.Float32x4
	ap := a[:kc*16]
	bp := b[:kc*16]
	for len(ap) >= 128 && len(bp) >= 128 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		ap, bp = ap[128:], bp[128:]
	}
	for len(ap) >= 16 && len(bp) >= 16 {
		b0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		ap, bp = ap[16:], bp[16:]
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
}

// KernelPB3x24U8 is a generated candidate: 3x24 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB3x24U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*12]
	bp := b[:kc*24]
	for len(ap) >= 96 && len(bp) >= 192 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[96:100])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[100:104])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[104:108])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[108:112])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[112:116])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[116:120]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[120:124])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[124:128])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[128:132])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[132:136])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[136:140])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[140:144]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[144:148])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[148:152])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[152:156])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[156:160])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[160:164])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[164:168]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[168:172])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[172:176])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[176:180])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[180:184])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[184:188])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[188:192]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[96:], bp[192:]
	}
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

// KernelPB6x12U8 is a generated candidate: 6x12 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB6x12U8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2 archsimd.Float32x4
	var c1_0, c1_1, c1_2 archsimd.Float32x4
	var c2_0, c2_1, c2_2 archsimd.Float32x4
	var c3_0, c3_1, c3_2 archsimd.Float32x4
	var c4_0, c4_1, c4_2 archsimd.Float32x4
	var c5_0, c5_1, c5_2 archsimd.Float32x4
	var b0, b1, b2, av archsimd.Float32x4
	ap := a[:kc*24]
	bp := b[:kc*12]
	for len(ap) >= 192 && len(bp) >= 96 {
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[64:68]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[68:72]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[72:76]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[76:80]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[80:84]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[84:88]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[88:92]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[92:96]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[96:100]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[100:104]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[104:108]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[108:112]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[112:116]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[116:120]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[120:124]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[124:128]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[128:132]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[132:136]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[136:140]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[140:144]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[144:148]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[148:152]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[152:156]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[156:160]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[160:164]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[164:168]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[168:172]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[172:176]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[176:180]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[180:184]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[184:188]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[188:192]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		ap, bp = ap[192:], bp[96:]
	}
	for len(ap) >= 24 && len(bp) >= 12 {
		b0, b1, b2 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		ap, bp = ap[24:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	r1 := c[1*ldc : 1*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	r2 := c[2*ldc : 2*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	r3 := c[3*ldc : 3*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	r4 := c[4*ldc : 4*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[0:4])).Add(c4_0).StoreArray((*[4]float32)(r4[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[4:8])).Add(c4_1).StoreArray((*[4]float32)(r4[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r4[8:12])).Add(c4_2).StoreArray((*[4]float32)(r4[8:12]))
	r5 := c[5*ldc : 5*ldc+12]
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[0:4])).Add(c5_0).StoreArray((*[4]float32)(r5[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[4:8])).Add(c5_1).StoreArray((*[4]float32)(r5[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r5[8:12])).Add(c5_2).StoreArray((*[4]float32)(r5[8:12]))
}

// KernelPB3x24U4 is a generated candidate: 3x24 u=4 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB3x24U4(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float32x4
	var b0, b1, b2, b3, b4, b5, av archsimd.Float32x4
	ap := a[:kc*12]
	bp := b[:kc*24]
	for len(ap) >= 48 && len(bp) >= 96 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[80:84])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[84:88])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[88:92])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[92:96]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[48:], bp[96:]
	}
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

// KernelPB4x20U4 is a generated candidate: 4x20 u=4 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelPB4x20U4(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4 archsimd.Float32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4 archsimd.Float32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4 archsimd.Float32x4
	var c3_0, c3_1, c3_2, c3_3, c3_4 archsimd.Float32x4
	var b0, b1, b2, b3, b4, av archsimd.Float32x4
	ap := a[:kc*16]
	bp := b[:kc*20]
	for len(ap) >= 64 && len(bp) >= 80 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[20:24])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[24:28])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[28:32])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[32:36])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[36:40]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[16:20]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[20:24]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[24:28]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[28:32]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[40:44])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[44:48])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[48:52])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[52:56])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[56:60]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[32:36]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[36:40]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[40:44]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[44:48]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[60:64])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[64:68])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[68:72])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[72:76])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[76:80]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[48:52]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[52:56]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[56:60]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[60:64]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[64:], bp[80:]
	}
	for len(ap) >= 16 && len(bp) >= 20 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[16:20]))
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[0:4]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[4:8]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[8:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat32x4Array((*[4]float32)(ap[12:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[16:], bp[20:]
	}
	r0 := c[0*ldc : 0*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[0:4])).Add(c0_0).StoreArray((*[4]float32)(r0[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[4:8])).Add(c0_1).StoreArray((*[4]float32)(r0[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[8:12])).Add(c0_2).StoreArray((*[4]float32)(r0[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[12:16])).Add(c0_3).StoreArray((*[4]float32)(r0[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r0[16:20])).Add(c0_4).StoreArray((*[4]float32)(r0[16:20]))
	r1 := c[1*ldc : 1*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[0:4])).Add(c1_0).StoreArray((*[4]float32)(r1[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[4:8])).Add(c1_1).StoreArray((*[4]float32)(r1[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[8:12])).Add(c1_2).StoreArray((*[4]float32)(r1[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[12:16])).Add(c1_3).StoreArray((*[4]float32)(r1[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r1[16:20])).Add(c1_4).StoreArray((*[4]float32)(r1[16:20]))
	r2 := c[2*ldc : 2*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[0:4])).Add(c2_0).StoreArray((*[4]float32)(r2[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[4:8])).Add(c2_1).StoreArray((*[4]float32)(r2[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[8:12])).Add(c2_2).StoreArray((*[4]float32)(r2[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[12:16])).Add(c2_3).StoreArray((*[4]float32)(r2[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r2[16:20])).Add(c2_4).StoreArray((*[4]float32)(r2[16:20]))
	r3 := c[3*ldc : 3*ldc+20]
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[0:4])).Add(c3_0).StoreArray((*[4]float32)(r3[0:4]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[4:8])).Add(c3_1).StoreArray((*[4]float32)(r3[4:8]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[8:12])).Add(c3_2).StoreArray((*[4]float32)(r3[8:12]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[12:16])).Add(c3_3).StoreArray((*[4]float32)(r3[12:16]))
	archsimd.LoadFloat32x4Array((*[4]float32)(r3[16:20])).Add(c3_4).StoreArray((*[4]float32)(r3[16:20]))
}

// KernelDPB4x10U8 is a generated candidate: 4x10 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB4x10U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4 archsimd.Float64x2
	var c3_0, c3_1, c3_2, c3_3, c3_4 archsimd.Float64x2
	var b0, b1, b2, b3, b4, av archsimd.Float64x2
	ap := a[:kc*8]
	bp := b[:kc*10]
	for len(ap) >= 64 && len(bp) >= 80 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[48:50]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[50:52])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[52:54])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[54:56])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[56:58])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[58:60]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[60:62])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[62:64])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[64:66])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[66:68])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[68:70]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[48:50]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[50:52]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[52:54]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[54:56]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[70:72])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[72:74])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[74:76])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[76:78])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[78:80]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[56:58]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[58:60]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[60:62]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[62:64]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[64:], bp[80:]
	}
	for len(ap) >= 8 && len(bp) >= 10 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[8:], bp[10:]
	}
	r0 := c[0*ldc : 0*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	r1 := c[1*ldc : 1*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	r2 := c[2*ldc : 2*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[8:10])).Add(c2_4).StoreArray((*[2]float64)(r2[8:10]))
	r3 := c[3*ldc : 3*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[6:8])).Add(c3_3).StoreArray((*[2]float64)(r3[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[8:10])).Add(c3_4).StoreArray((*[2]float64)(r3[8:10]))
}

// KernelDPB5x8U8 is a generated candidate: 5x8 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB5x8U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float64x2
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float64x2
	var c4_0, c4_1, c4_2, c4_3 archsimd.Float64x2
	var b0, b1, b2, b3, av archsimd.Float64x2
	ap := a[:kc*10]
	bp := b[:kc*8]
	for len(ap) >= 80 && len(bp) >= 64 {
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[48:50]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[50:52]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[52:54]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[54:56]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[56:58]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[58:60]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[48:50])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[50:52])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[52:54])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[54:56]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[60:62]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[62:64]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[64:66]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[66:68]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[68:70]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[56:58])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[58:60])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[60:62])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[62:64]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[70:72]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[72:74]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[74:76]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[76:78]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[78:80]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[80:], bp[64:]
	}
	for len(ap) >= 10 && len(bp) >= 8 {
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2, c4_3 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2), av.MulAdd(b3, c4_3)
		ap, bp = ap[10:], bp[8:]
	}
	r0 := c[0*ldc : 0*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	r1 := c[1*ldc : 1*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	r2 := c[2*ldc : 2*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	r3 := c[3*ldc : 3*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[6:8])).Add(c3_3).StoreArray((*[2]float64)(r3[6:8]))
	r4 := c[4*ldc : 4*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[0:2])).Add(c4_0).StoreArray((*[2]float64)(r4[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[2:4])).Add(c4_1).StoreArray((*[2]float64)(r4[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[4:6])).Add(c4_2).StoreArray((*[2]float64)(r4[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[6:8])).Add(c4_3).StoreArray((*[2]float64)(r4[6:8]))
}

// KernelDPB3x14U8 is a generated candidate: 3x14 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB3x14U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, b6, av archsimd.Float64x2
	ap := a[:kc*6]
	bp := b[:kc*14]
	for len(ap) >= 48 && len(bp) >= 112 {
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[48:50])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[50:52])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[52:54])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[54:56]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[56:58])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[58:60])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[60:62])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[62:64])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[64:66])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[66:68])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[68:70]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[70:72])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[72:74])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[74:76])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[76:78])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[78:80])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[80:82])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[82:84]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[84:86])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[86:88])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[88:90])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[90:92])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[92:94])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[94:96])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[96:98]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[98:100])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[100:102])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[102:104])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[104:106])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[106:108])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[108:110])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[110:112]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		ap, bp = ap[48:], bp[112:]
	}
	for len(ap) >= 6 && len(bp) >= 14 {
		b0, b1, b2, b3, b4, b5, b6 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5), av.MulAdd(b6, c0_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5), av.MulAdd(b6, c1_6)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5), av.MulAdd(b6, c2_6)
		ap, bp = ap[6:], bp[14:]
	}
	r0 := c[0*ldc : 0*ldc+14]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[10:12])).Add(c0_5).StoreArray((*[2]float64)(r0[10:12]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[12:14])).Add(c0_6).StoreArray((*[2]float64)(r0[12:14]))
	r1 := c[1*ldc : 1*ldc+14]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[10:12])).Add(c1_5).StoreArray((*[2]float64)(r1[10:12]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[12:14])).Add(c1_6).StoreArray((*[2]float64)(r1[12:14]))
	r2 := c[2*ldc : 2*ldc+14]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[8:10])).Add(c2_4).StoreArray((*[2]float64)(r2[8:10]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[10:12])).Add(c2_5).StoreArray((*[2]float64)(r2[10:12]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[12:14])).Add(c2_6).StoreArray((*[2]float64)(r2[12:14]))
}

// KernelDPB7x6U8 is a generated candidate: 7x6 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB7x6U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2 archsimd.Float64x2
	var c1_0, c1_1, c1_2 archsimd.Float64x2
	var c2_0, c2_1, c2_2 archsimd.Float64x2
	var c3_0, c3_1, c3_2 archsimd.Float64x2
	var c4_0, c4_1, c4_2 archsimd.Float64x2
	var c5_0, c5_1, c5_2 archsimd.Float64x2
	var c6_0, c6_1, c6_2 archsimd.Float64x2
	var b0, b1, b2, av archsimd.Float64x2
	ap := a[:kc*14]
	bp := b[:kc*6]
	for len(ap) >= 112 && len(bp) >= 48 {
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[48:50]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[50:52]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[52:54]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[54:56]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[56:58]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[58:60]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[60:62]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[62:64]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[64:66]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[66:68]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[68:70]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[70:72]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[72:74]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[74:76]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[76:78]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[78:80]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[80:82]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[82:84]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[84:86]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[86:88]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[88:90]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[90:92]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[92:94]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[94:96]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[96:98]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[98:100]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[100:102]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[102:104]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[104:106]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[106:108]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[108:110]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[110:112]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		ap, bp = ap[112:], bp[48:]
	}
	for len(ap) >= 14 && len(bp) >= 6 {
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c6_0, c6_1, c6_2 = av.MulAdd(b0, c6_0), av.MulAdd(b1, c6_1), av.MulAdd(b2, c6_2)
		ap, bp = ap[14:], bp[6:]
	}
	r0 := c[0*ldc : 0*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	r1 := c[1*ldc : 1*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	r2 := c[2*ldc : 2*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	r3 := c[3*ldc : 3*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	r4 := c[4*ldc : 4*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[0:2])).Add(c4_0).StoreArray((*[2]float64)(r4[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[2:4])).Add(c4_1).StoreArray((*[2]float64)(r4[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[4:6])).Add(c4_2).StoreArray((*[2]float64)(r4[4:6]))
	r5 := c[5*ldc : 5*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[0:2])).Add(c5_0).StoreArray((*[2]float64)(r5[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[2:4])).Add(c5_1).StoreArray((*[2]float64)(r5[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[4:6])).Add(c5_2).StoreArray((*[2]float64)(r5[4:6]))
	r6 := c[6*ldc : 6*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r6[0:2])).Add(c6_0).StoreArray((*[2]float64)(r6[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r6[2:4])).Add(c6_1).StoreArray((*[2]float64)(r6[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r6[4:6])).Add(c6_2).StoreArray((*[2]float64)(r6[4:6]))
}

// KernelDPB4x8U8 is a generated candidate: 4x8 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB4x8U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3 archsimd.Float64x2
	var c3_0, c3_1, c3_2, c3_3 archsimd.Float64x2
	var b0, b1, b2, b3, av archsimd.Float64x2
	ap := a[:kc*8]
	bp := b[:kc*8]
	for len(ap) >= 64 && len(bp) >= 64 {
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[48:50])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[50:52])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[52:54])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[54:56]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[48:50]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[50:52]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[52:54]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[54:56]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[56:58])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[58:60])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[60:62])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[62:64]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[56:58]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[58:60]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[60:62]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[62:64]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		ap, bp = ap[64:], bp[64:]
	}
	for len(ap) >= 8 && len(bp) >= 8 {
		b0, b1, b2, b3 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3)
		ap, bp = ap[8:], bp[8:]
	}
	r0 := c[0*ldc : 0*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	r1 := c[1*ldc : 1*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	r2 := c[2*ldc : 2*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	r3 := c[3*ldc : 3*ldc+8]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[6:8])).Add(c3_3).StoreArray((*[2]float64)(r3[6:8]))
}

// KernelDPB3x12U8 is a generated candidate: 3x12 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB3x12U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*6]
	bp := b[:kc*12]
	for len(ap) >= 48 && len(bp) >= 96 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[48:50])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[50:52])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[52:54])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[54:56])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[56:58])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[58:60]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[60:62])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[62:64])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[64:66])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[66:68])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[68:70])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[70:72]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[72:74])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[74:76])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[76:78])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[78:80])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[80:82])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[82:84]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[84:86])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[86:88])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[88:90])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[90:92])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[92:94])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[94:96]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[48:], bp[96:]
	}
	for len(ap) >= 6 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[6:], bp[12:]
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

// KernelDPB6x6U8 is a generated candidate: 6x6 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB6x6U8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2 archsimd.Float64x2
	var c1_0, c1_1, c1_2 archsimd.Float64x2
	var c2_0, c2_1, c2_2 archsimd.Float64x2
	var c3_0, c3_1, c3_2 archsimd.Float64x2
	var c4_0, c4_1, c4_2 archsimd.Float64x2
	var c5_0, c5_1, c5_2 archsimd.Float64x2
	var b0, b1, b2, av archsimd.Float64x2
	ap := a[:kc*12]
	bp := b[:kc*6]
	for len(ap) >= 96 && len(bp) >= 48 {
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[32:34]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[34:36]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[36:38]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[38:40]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[40:42]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[42:44]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[44:46]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[46:48]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[48:50]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[50:52]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[52:54]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[54:56]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[56:58]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[58:60]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[60:62]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[62:64]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[64:66]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[66:68]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[68:70]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[70:72]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[72:74]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[74:76]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[76:78]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[78:80]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[80:82]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[82:84]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[84:86]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[86:88]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[88:90]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[90:92]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[92:94]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[94:96]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		ap, bp = ap[96:], bp[48:]
	}
	for len(ap) >= 12 && len(bp) >= 6 {
		b0, b1, b2 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c4_0, c4_1, c4_2 = av.MulAdd(b0, c4_0), av.MulAdd(b1, c4_1), av.MulAdd(b2, c4_2)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c5_0, c5_1, c5_2 = av.MulAdd(b0, c5_0), av.MulAdd(b1, c5_1), av.MulAdd(b2, c5_2)
		ap, bp = ap[12:], bp[6:]
	}
	r0 := c[0*ldc : 0*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	r1 := c[1*ldc : 1*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	r2 := c[2*ldc : 2*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	r3 := c[3*ldc : 3*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	r4 := c[4*ldc : 4*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[0:2])).Add(c4_0).StoreArray((*[2]float64)(r4[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[2:4])).Add(c4_1).StoreArray((*[2]float64)(r4[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r4[4:6])).Add(c4_2).StoreArray((*[2]float64)(r4[4:6]))
	r5 := c[5*ldc : 5*ldc+6]
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[0:2])).Add(c5_0).StoreArray((*[2]float64)(r5[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[2:4])).Add(c5_1).StoreArray((*[2]float64)(r5[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r5[4:6])).Add(c5_2).StoreArray((*[2]float64)(r5[4:6]))
}

// KernelDPB3x12U4 is a generated candidate: 3x12 u=4 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB3x12U4(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*6]
	bp := b[:kc*12]
	for len(ap) >= 24 && len(bp) >= 48 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[40:42])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[42:44])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[44:46])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[46:48]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[24:], bp[48:]
	}
	for len(ap) >= 6 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4), av.MulAdd(b5, c2_5)
		ap, bp = ap[6:], bp[12:]
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

// KernelDPB4x10U4 is a generated candidate: 4x10 u=4 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB4x10U4(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4 archsimd.Float64x2
	var c3_0, c3_1, c3_2, c3_3, c3_4 archsimd.Float64x2
	var b0, b1, b2, b3, b4, av archsimd.Float64x2
	ap := a[:kc*8]
	bp := b[:kc*10]
	for len(ap) >= 32 && len(bp) >= 40 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[12:14])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[14:16])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[16:18])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[18:20]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[8:10]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[10:12]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[12:14]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[14:16]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[20:22])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[22:24])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[24:26])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[26:28])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[28:30]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[16:18]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[18:20]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[20:22]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[22:24]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[30:32])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[32:34])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[34:36])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[36:38])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[38:40]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[24:26]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[26:28]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[28:30]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[30:32]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[32:], bp[40:]
	}
	for len(ap) >= 8 && len(bp) >= 10 {
		b0, b1, b2, b3, b4 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10]))
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[0:2]))
		c0_0, c0_1, c0_2, c0_3, c0_4 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[2:4]))
		c1_0, c1_1, c1_2, c1_3, c1_4 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[4:6]))
		c2_0, c2_1, c2_2, c2_3, c2_4 = av.MulAdd(b0, c2_0), av.MulAdd(b1, c2_1), av.MulAdd(b2, c2_2), av.MulAdd(b3, c2_3), av.MulAdd(b4, c2_4)
		av = archsimd.LoadFloat64x2Array((*[2]float64)(ap[6:8]))
		c3_0, c3_1, c3_2, c3_3, c3_4 = av.MulAdd(b0, c3_0), av.MulAdd(b1, c3_1), av.MulAdd(b2, c3_2), av.MulAdd(b3, c3_3), av.MulAdd(b4, c3_4)
		ap, bp = ap[8:], bp[10:]
	}
	r0 := c[0*ldc : 0*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[0:2])).Add(c0_0).StoreArray((*[2]float64)(r0[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[2:4])).Add(c0_1).StoreArray((*[2]float64)(r0[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[4:6])).Add(c0_2).StoreArray((*[2]float64)(r0[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[6:8])).Add(c0_3).StoreArray((*[2]float64)(r0[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r0[8:10])).Add(c0_4).StoreArray((*[2]float64)(r0[8:10]))
	r1 := c[1*ldc : 1*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[0:2])).Add(c1_0).StoreArray((*[2]float64)(r1[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[2:4])).Add(c1_1).StoreArray((*[2]float64)(r1[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[4:6])).Add(c1_2).StoreArray((*[2]float64)(r1[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[6:8])).Add(c1_3).StoreArray((*[2]float64)(r1[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r1[8:10])).Add(c1_4).StoreArray((*[2]float64)(r1[8:10]))
	r2 := c[2*ldc : 2*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[0:2])).Add(c2_0).StoreArray((*[2]float64)(r2[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[2:4])).Add(c2_1).StoreArray((*[2]float64)(r2[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[4:6])).Add(c2_2).StoreArray((*[2]float64)(r2[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[6:8])).Add(c2_3).StoreArray((*[2]float64)(r2[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r2[8:10])).Add(c2_4).StoreArray((*[2]float64)(r2[8:10]))
	r3 := c[3*ldc : 3*ldc+10]
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[0:2])).Add(c3_0).StoreArray((*[2]float64)(r3[0:2]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[2:4])).Add(c3_1).StoreArray((*[2]float64)(r3[2:4]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[4:6])).Add(c3_2).StoreArray((*[2]float64)(r3[4:6]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[6:8])).Add(c3_3).StoreArray((*[2]float64)(r3[6:8]))
	archsimd.LoadFloat64x2Array((*[2]float64)(r3[8:10])).Add(c3_4).StoreArray((*[2]float64)(r3[8:10]))
}

type pbCand struct {
	name   string
	mr, nr int
	f32    func(kc int, a, b, c []float32, ldc int)
	f64    func(kc int, a, b, c []float64, ldc int)
}

var pbCands = []pbCand{
	{"f32/4x20u8", 4, 20, KernelPB4x20U8, nil},
	{"f32/5x16u8", 5, 16, KernelPB5x16U8, nil},
	{"f32/3x28u8", 3, 28, KernelPB3x28U8, nil},
	{"f32/7x12u8", 7, 12, KernelPB7x12U8, nil},
	{"f32/4x16u8", 4, 16, KernelPB4x16U8, nil},
	{"f32/3x24u8", 3, 24, KernelPB3x24U8, nil},
	{"f32/6x12u8", 6, 12, KernelPB6x12U8, nil},
	{"f32/3x24u4", 3, 24, KernelPB3x24U4, nil},
	{"f32/4x20u4", 4, 20, KernelPB4x20U4, nil},
	{"f64/4x10u8", 4, 10, nil, KernelDPB4x10U8},
	{"f64/5x8u8", 5, 8, nil, KernelDPB5x8U8},
	{"f64/3x14u8", 3, 14, nil, KernelDPB3x14U8},
	{"f64/7x6u8", 7, 6, nil, KernelDPB7x6U8},
	{"f64/4x8u8", 4, 8, nil, KernelDPB4x8U8},
	{"f64/3x12u8", 3, 12, nil, KernelDPB3x12U8},
	{"f64/6x6u8", 6, 6, nil, KernelDPB6x6U8},
	{"f64/3x12u4", 3, 12, nil, KernelDPB3x12U4},
	{"f64/4x10u4", 4, 10, nil, KernelDPB4x10U4},
}

// pbPack replicates each element of a conventionally packed A panel (k-major, MR per
// k-step) across lanes, the layout a PB kernel reads.
func pbPack(a []float64, lanes int) []float64 {
	o := make([]float64, len(a)*lanes)
	for i, v := range a {
		for l := 0; l < lanes; l++ {
			o[i*lanes+l] = v
		}
	}
	return o
}

// TestPBCandidates holds every PB candidate to a definition-order reference over
// C += A·B computed from the conventional A, with a guard column past NR.
func TestPBCandidates(t *testing.T) {
	for _, p := range pbCands {
		for _, kc := range []int{1, 2, 3, 7, 8, 9, 17, 64} {
			rng := rand.New(rand.NewSource(int64(kc)))
			ldc := p.nr + 3
			a, b := make([]float64, kc*p.mr), make([]float64, kc*p.nr)
			c := make([]float64, (p.mr-1)*ldc+p.nr+3)
			for _, s := range [][]float64{a, b, c} {
				for i := range s {
					s[i] = 2*rng.Float64() - 1
				}
			}
			if p.f32 != nil { // the reference sees float32-rounded inputs
				for _, s := range [][]float64{a, b, c} {
					for i := range s {
						s[i] = float64(float32(s[i]))
					}
				}
			}
			got := append([]float64(nil), c...)
			if p.f32 != nil {
				a32, b32, c32 := to32(pbPack(a, 4)), to32(b), to32(c)
				p.f32(kc, a32, b32, c32, ldc)
				for i := range got {
					got[i] = float64(c32[i])
				}
			} else {
				p.f64(kc, pbPack(a, 2), b, got, ldc)
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

var pbSink float64

// BenchmarkPBTile times every candidate at the kc values gate-p2 and the A/B used,
// L1-resident, ldc padded by one cache line.
func BenchmarkPBTile(b *testing.B) {
	for _, p := range pbCands {
		for _, kc := range []int{128, 512} {
			b.Run(fmt.Sprintf("%s/kc=%d", p.name, kc), func(b *testing.B) {
				rng := rand.New(rand.NewSource(1))
				flops := 2.0 * float64(p.mr*p.nr*kc)
				if p.f32 != nil {
					ldc := p.nr + 16
					a, bp, c := make([]float32, 4*kc*p.mr), make([]float32, kc*p.nr), make([]float32, p.mr*ldc)
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
					pbSink = float64(c[0])
				} else {
					ldc := p.nr + 8
					a, bp, c := make([]float64, 2*kc*p.mr), make([]float64, kc*p.nr), make([]float64, p.mr*ldc)
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
					pbSink = c[0]
				}
				b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
			})
		}
	}
}
