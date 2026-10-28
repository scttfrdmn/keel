// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

import "simd/archsimd"

// This file holds the NEON float64 DGEMM microkernel bodies (#191), the float64
// counterparts of gemm_neon.go's: the float32 NEON shapes at half the columns, since
// a V register holds 2 float64 lanes where it holds 4 float32. Every body is
// tools/shapegen's `-arch arm64 -elem f64` emission with the shim's package
// qualifier removed (the emitter compiles its candidates outside this package);
// TestNEONF64ShapesAuditIdentically holds each body's audit to the emitter's, field
// for field, which is how the float32 NEON kernels are held too.
//
// shapegen's NEON float64 sweep: 3x12/2 is the frontier at 4.111 insns/FMA, the
// float32 NEON frontier's own figure (3x24/2), over the same 107 zero-spill shapes.
// 8x4, 4x8 and 3x12/2 ship, the twins of float32's 8x8, 4x16 and 3x24/2; 8x6, 8x8
// and 4x16 are the spilling references, the twins of 8x12, 8x16 and 4x32.
// No ClearAVXUpperBits counterpart: NEON has no AVX-SSE transition.

// KernelD8x4 computes C += A·B for one 8x4 float64 tile, the twin of Kernel8x8.
func KernelD8x4(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1 F64x2
	var c1_0, c1_1 F64x2
	var c2_0, c2_1 F64x2
	var c3_0, c3_1 F64x2
	var c4_0, c4_1 F64x2
	var c5_0, c5_1 F64x2
	var c6_0, c6_1 F64x2
	var c7_0, c7_1 F64x2
	var b0, b1, av F64x2
	ap := a[:kc*8]
	bp := b[:kc*4]
	for len(ap) >= 8 && len(bp) >= 4 {
		b0, b1 = Load128D(bp[0:2]), Load128D(bp[2:4])
		av = Broadcast128D(ap[0])
		c0_0, c0_1 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1)
		av = Broadcast128D(ap[1])
		c1_0, c1_1 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1)
		av = Broadcast128D(ap[2])
		c2_0, c2_1 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1)
		av = Broadcast128D(ap[3])
		c3_0, c3_1 = FMA128D(av, b0, c3_0), FMA128D(av, b1, c3_1)
		av = Broadcast128D(ap[4])
		c4_0, c4_1 = FMA128D(av, b0, c4_0), FMA128D(av, b1, c4_1)
		av = Broadcast128D(ap[5])
		c5_0, c5_1 = FMA128D(av, b0, c5_0), FMA128D(av, b1, c5_1)
		av = Broadcast128D(ap[6])
		c6_0, c6_1 = FMA128D(av, b0, c6_0), FMA128D(av, b1, c6_1)
		av = Broadcast128D(ap[7])
		c7_0, c7_1 = FMA128D(av, b0, c7_0), FMA128D(av, b1, c7_1)
		ap, bp = ap[8:], bp[4:]
	}
	r0 := c[0*ldc : 0*ldc+4]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	r1 := c[1*ldc : 1*ldc+4]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	r2 := c[2*ldc : 2*ldc+4]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	r3 := c[3*ldc : 3*ldc+4]
	Store128D(r3[0:2], Add128D(Load128D(r3[0:2]), c3_0))
	Store128D(r3[2:4], Add128D(Load128D(r3[2:4]), c3_1))
	r4 := c[4*ldc : 4*ldc+4]
	Store128D(r4[0:2], Add128D(Load128D(r4[0:2]), c4_0))
	Store128D(r4[2:4], Add128D(Load128D(r4[2:4]), c4_1))
	r5 := c[5*ldc : 5*ldc+4]
	Store128D(r5[0:2], Add128D(Load128D(r5[0:2]), c5_0))
	Store128D(r5[2:4], Add128D(Load128D(r5[2:4]), c5_1))
	r6 := c[6*ldc : 6*ldc+4]
	Store128D(r6[0:2], Add128D(Load128D(r6[0:2]), c6_0))
	Store128D(r6[2:4], Add128D(Load128D(r6[2:4]), c6_1))
	r7 := c[7*ldc : 7*ldc+4]
	Store128D(r7[0:2], Add128D(Load128D(r7[0:2]), c7_0))
	Store128D(r7[2:4], Add128D(Load128D(r7[2:4]), c7_1))
}

// KernelD4x8 computes C += A·B for one 4x8 float64 tile, the twin of Kernel4x16.
func KernelD4x8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 F64x2
	var c1_0, c1_1, c1_2, c1_3 F64x2
	var c2_0, c2_1, c2_2, c2_3 F64x2
	var c3_0, c3_1, c3_2, c3_3 F64x2
	var b0, b1, b2, b3, av F64x2
	ap := a[:kc*4]
	bp := b[:kc*8]
	for len(ap) >= 4 && len(bp) >= 8 {
		b0, b1, b2, b3 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6]), Load128D(bp[6:8])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2, c0_3 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2, c1_3 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2, c2_3 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3)
		av = Broadcast128D(ap[3])
		c3_0, c3_1, c3_2, c3_3 = FMA128D(av, b0, c3_0), FMA128D(av, b1, c3_1), FMA128D(av, b2, c3_2), FMA128D(av, b3, c3_3)
		ap, bp = ap[4:], bp[8:]
	}
	r0 := c[0*ldc : 0*ldc+8]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	Store128D(r0[4:6], Add128D(Load128D(r0[4:6]), c0_2))
	Store128D(r0[6:8], Add128D(Load128D(r0[6:8]), c0_3))
	r1 := c[1*ldc : 1*ldc+8]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	Store128D(r1[4:6], Add128D(Load128D(r1[4:6]), c1_2))
	Store128D(r1[6:8], Add128D(Load128D(r1[6:8]), c1_3))
	r2 := c[2*ldc : 2*ldc+8]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	Store128D(r2[4:6], Add128D(Load128D(r2[4:6]), c2_2))
	Store128D(r2[6:8], Add128D(Load128D(r2[6:8]), c2_3))
	r3 := c[3*ldc : 3*ldc+8]
	Store128D(r3[0:2], Add128D(Load128D(r3[0:2]), c3_0))
	Store128D(r3[2:4], Add128D(Load128D(r3[2:4]), c3_1))
	Store128D(r3[4:6], Add128D(Load128D(r3[4:6]), c3_2))
	Store128D(r3[6:8], Add128D(Load128D(r3[6:8]), c3_3))
}

// KernelD3x12 computes C += A·B for one 3x12 float64 tile, two k-steps per pass:
// the twin of Kernel3x24 and the sweep's frontier shape.
func KernelD3x12(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 F64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 F64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 F64x2
	var b0, b1, b2, b3, b4, b5, av F64x2
	ap := a[:kc*3]
	bp := b[:kc*12]
	for len(ap) >= 6 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6]), Load128D(bp[6:8]), Load128D(bp[8:10]), Load128D(bp[10:12])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3), FMA128D(av, b4, c0_4), FMA128D(av, b5, c0_5)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3), FMA128D(av, b4, c1_4), FMA128D(av, b5, c1_5)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3), FMA128D(av, b4, c2_4), FMA128D(av, b5, c2_5)
		b0, b1, b2, b3, b4, b5 = Load128D(bp[12:14]), Load128D(bp[14:16]), Load128D(bp[16:18]), Load128D(bp[18:20]), Load128D(bp[20:22]), Load128D(bp[22:24])
		av = Broadcast128D(ap[3])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3), FMA128D(av, b4, c0_4), FMA128D(av, b5, c0_5)
		av = Broadcast128D(ap[4])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3), FMA128D(av, b4, c1_4), FMA128D(av, b5, c1_5)
		av = Broadcast128D(ap[5])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3), FMA128D(av, b4, c2_4), FMA128D(av, b5, c2_5)
		ap, bp = ap[6:], bp[24:]
	}
	for len(ap) >= 3 && len(bp) >= 12 {
		b0, b1, b2, b3, b4, b5 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6]), Load128D(bp[6:8]), Load128D(bp[8:10]), Load128D(bp[10:12])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3), FMA128D(av, b4, c0_4), FMA128D(av, b5, c0_5)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3), FMA128D(av, b4, c1_4), FMA128D(av, b5, c1_5)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3), FMA128D(av, b4, c2_4), FMA128D(av, b5, c2_5)
		ap, bp = ap[3:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	Store128D(r0[4:6], Add128D(Load128D(r0[4:6]), c0_2))
	Store128D(r0[6:8], Add128D(Load128D(r0[6:8]), c0_3))
	Store128D(r0[8:10], Add128D(Load128D(r0[8:10]), c0_4))
	Store128D(r0[10:12], Add128D(Load128D(r0[10:12]), c0_5))
	r1 := c[1*ldc : 1*ldc+12]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	Store128D(r1[4:6], Add128D(Load128D(r1[4:6]), c1_2))
	Store128D(r1[6:8], Add128D(Load128D(r1[6:8]), c1_3))
	Store128D(r1[8:10], Add128D(Load128D(r1[8:10]), c1_4))
	Store128D(r1[10:12], Add128D(Load128D(r1[10:12]), c1_5))
	r2 := c[2*ldc : 2*ldc+12]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	Store128D(r2[4:6], Add128D(Load128D(r2[4:6]), c2_2))
	Store128D(r2[6:8], Add128D(Load128D(r2[6:8]), c2_3))
	Store128D(r2[8:10], Add128D(Load128D(r2[8:10]), c2_4))
	Store128D(r2[10:12], Add128D(Load128D(r2[10:12]), c2_5))
}

// KernelD8x6 is a spilling reference tile, the twin of Kernel8x12: evidence, never shipped.
func KernelD8x6(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2 F64x2
	var c1_0, c1_1, c1_2 F64x2
	var c2_0, c2_1, c2_2 F64x2
	var c3_0, c3_1, c3_2 F64x2
	var c4_0, c4_1, c4_2 F64x2
	var c5_0, c5_1, c5_2 F64x2
	var c6_0, c6_1, c6_2 F64x2
	var c7_0, c7_1, c7_2 F64x2
	var b0, b1, b2, av F64x2
	ap := a[:kc*8]
	bp := b[:kc*6]
	for len(ap) >= 8 && len(bp) >= 6 {
		b0, b1, b2 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2)
		av = Broadcast128D(ap[3])
		c3_0, c3_1, c3_2 = FMA128D(av, b0, c3_0), FMA128D(av, b1, c3_1), FMA128D(av, b2, c3_2)
		av = Broadcast128D(ap[4])
		c4_0, c4_1, c4_2 = FMA128D(av, b0, c4_0), FMA128D(av, b1, c4_1), FMA128D(av, b2, c4_2)
		av = Broadcast128D(ap[5])
		c5_0, c5_1, c5_2 = FMA128D(av, b0, c5_0), FMA128D(av, b1, c5_1), FMA128D(av, b2, c5_2)
		av = Broadcast128D(ap[6])
		c6_0, c6_1, c6_2 = FMA128D(av, b0, c6_0), FMA128D(av, b1, c6_1), FMA128D(av, b2, c6_2)
		av = Broadcast128D(ap[7])
		c7_0, c7_1, c7_2 = FMA128D(av, b0, c7_0), FMA128D(av, b1, c7_1), FMA128D(av, b2, c7_2)
		ap, bp = ap[8:], bp[6:]
	}
	r0 := c[0*ldc : 0*ldc+6]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	Store128D(r0[4:6], Add128D(Load128D(r0[4:6]), c0_2))
	r1 := c[1*ldc : 1*ldc+6]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	Store128D(r1[4:6], Add128D(Load128D(r1[4:6]), c1_2))
	r2 := c[2*ldc : 2*ldc+6]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	Store128D(r2[4:6], Add128D(Load128D(r2[4:6]), c2_2))
	r3 := c[3*ldc : 3*ldc+6]
	Store128D(r3[0:2], Add128D(Load128D(r3[0:2]), c3_0))
	Store128D(r3[2:4], Add128D(Load128D(r3[2:4]), c3_1))
	Store128D(r3[4:6], Add128D(Load128D(r3[4:6]), c3_2))
	r4 := c[4*ldc : 4*ldc+6]
	Store128D(r4[0:2], Add128D(Load128D(r4[0:2]), c4_0))
	Store128D(r4[2:4], Add128D(Load128D(r4[2:4]), c4_1))
	Store128D(r4[4:6], Add128D(Load128D(r4[4:6]), c4_2))
	r5 := c[5*ldc : 5*ldc+6]
	Store128D(r5[0:2], Add128D(Load128D(r5[0:2]), c5_0))
	Store128D(r5[2:4], Add128D(Load128D(r5[2:4]), c5_1))
	Store128D(r5[4:6], Add128D(Load128D(r5[4:6]), c5_2))
	r6 := c[6*ldc : 6*ldc+6]
	Store128D(r6[0:2], Add128D(Load128D(r6[0:2]), c6_0))
	Store128D(r6[2:4], Add128D(Load128D(r6[2:4]), c6_1))
	Store128D(r6[4:6], Add128D(Load128D(r6[4:6]), c6_2))
	r7 := c[7*ldc : 7*ldc+6]
	Store128D(r7[0:2], Add128D(Load128D(r7[0:2]), c7_0))
	Store128D(r7[2:4], Add128D(Load128D(r7[2:4]), c7_1))
	Store128D(r7[4:6], Add128D(Load128D(r7[4:6]), c7_2))
}

// KernelD8x8 is a spilling reference tile, the twin of Kernel8x16: evidence, never shipped.
func KernelD8x8(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 F64x2
	var c1_0, c1_1, c1_2, c1_3 F64x2
	var c2_0, c2_1, c2_2, c2_3 F64x2
	var c3_0, c3_1, c3_2, c3_3 F64x2
	var c4_0, c4_1, c4_2, c4_3 F64x2
	var c5_0, c5_1, c5_2, c5_3 F64x2
	var c6_0, c6_1, c6_2, c6_3 F64x2
	var c7_0, c7_1, c7_2, c7_3 F64x2
	var b0, b1, b2, b3, av F64x2
	ap := a[:kc*8]
	bp := b[:kc*8]
	for len(ap) >= 8 && len(bp) >= 8 {
		b0, b1, b2, b3 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6]), Load128D(bp[6:8])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2, c0_3 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2, c1_3 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2, c2_3 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3)
		av = Broadcast128D(ap[3])
		c3_0, c3_1, c3_2, c3_3 = FMA128D(av, b0, c3_0), FMA128D(av, b1, c3_1), FMA128D(av, b2, c3_2), FMA128D(av, b3, c3_3)
		av = Broadcast128D(ap[4])
		c4_0, c4_1, c4_2, c4_3 = FMA128D(av, b0, c4_0), FMA128D(av, b1, c4_1), FMA128D(av, b2, c4_2), FMA128D(av, b3, c4_3)
		av = Broadcast128D(ap[5])
		c5_0, c5_1, c5_2, c5_3 = FMA128D(av, b0, c5_0), FMA128D(av, b1, c5_1), FMA128D(av, b2, c5_2), FMA128D(av, b3, c5_3)
		av = Broadcast128D(ap[6])
		c6_0, c6_1, c6_2, c6_3 = FMA128D(av, b0, c6_0), FMA128D(av, b1, c6_1), FMA128D(av, b2, c6_2), FMA128D(av, b3, c6_3)
		av = Broadcast128D(ap[7])
		c7_0, c7_1, c7_2, c7_3 = FMA128D(av, b0, c7_0), FMA128D(av, b1, c7_1), FMA128D(av, b2, c7_2), FMA128D(av, b3, c7_3)
		ap, bp = ap[8:], bp[8:]
	}
	r0 := c[0*ldc : 0*ldc+8]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	Store128D(r0[4:6], Add128D(Load128D(r0[4:6]), c0_2))
	Store128D(r0[6:8], Add128D(Load128D(r0[6:8]), c0_3))
	r1 := c[1*ldc : 1*ldc+8]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	Store128D(r1[4:6], Add128D(Load128D(r1[4:6]), c1_2))
	Store128D(r1[6:8], Add128D(Load128D(r1[6:8]), c1_3))
	r2 := c[2*ldc : 2*ldc+8]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	Store128D(r2[4:6], Add128D(Load128D(r2[4:6]), c2_2))
	Store128D(r2[6:8], Add128D(Load128D(r2[6:8]), c2_3))
	r3 := c[3*ldc : 3*ldc+8]
	Store128D(r3[0:2], Add128D(Load128D(r3[0:2]), c3_0))
	Store128D(r3[2:4], Add128D(Load128D(r3[2:4]), c3_1))
	Store128D(r3[4:6], Add128D(Load128D(r3[4:6]), c3_2))
	Store128D(r3[6:8], Add128D(Load128D(r3[6:8]), c3_3))
	r4 := c[4*ldc : 4*ldc+8]
	Store128D(r4[0:2], Add128D(Load128D(r4[0:2]), c4_0))
	Store128D(r4[2:4], Add128D(Load128D(r4[2:4]), c4_1))
	Store128D(r4[4:6], Add128D(Load128D(r4[4:6]), c4_2))
	Store128D(r4[6:8], Add128D(Load128D(r4[6:8]), c4_3))
	r5 := c[5*ldc : 5*ldc+8]
	Store128D(r5[0:2], Add128D(Load128D(r5[0:2]), c5_0))
	Store128D(r5[2:4], Add128D(Load128D(r5[2:4]), c5_1))
	Store128D(r5[4:6], Add128D(Load128D(r5[4:6]), c5_2))
	Store128D(r5[6:8], Add128D(Load128D(r5[6:8]), c5_3))
	r6 := c[6*ldc : 6*ldc+8]
	Store128D(r6[0:2], Add128D(Load128D(r6[0:2]), c6_0))
	Store128D(r6[2:4], Add128D(Load128D(r6[2:4]), c6_1))
	Store128D(r6[4:6], Add128D(Load128D(r6[4:6]), c6_2))
	Store128D(r6[6:8], Add128D(Load128D(r6[6:8]), c6_3))
	r7 := c[7*ldc : 7*ldc+8]
	Store128D(r7[0:2], Add128D(Load128D(r7[0:2]), c7_0))
	Store128D(r7[2:4], Add128D(Load128D(r7[2:4]), c7_1))
	Store128D(r7[4:6], Add128D(Load128D(r7[4:6]), c7_2))
	Store128D(r7[6:8], Add128D(Load128D(r7[6:8]), c7_3))
}

// KernelD4x16 is a spilling reference tile, the twin of Kernel4x32: evidence, never shipped.
func KernelD4x16(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6, c0_7 F64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6, c1_7 F64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6, c2_7 F64x2
	var c3_0, c3_1, c3_2, c3_3, c3_4, c3_5, c3_6, c3_7 F64x2
	var b0, b1, b2, b3, b4, b5, b6, b7, av F64x2
	ap := a[:kc*4]
	bp := b[:kc*16]
	for len(ap) >= 4 && len(bp) >= 16 {
		b0, b1, b2, b3, b4, b5, b6, b7 = Load128D(bp[0:2]), Load128D(bp[2:4]), Load128D(bp[4:6]), Load128D(bp[6:8]), Load128D(bp[8:10]), Load128D(bp[10:12]), Load128D(bp[12:14]), Load128D(bp[14:16])
		av = Broadcast128D(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6, c0_7 = FMA128D(av, b0, c0_0), FMA128D(av, b1, c0_1), FMA128D(av, b2, c0_2), FMA128D(av, b3, c0_3), FMA128D(av, b4, c0_4), FMA128D(av, b5, c0_5), FMA128D(av, b6, c0_6), FMA128D(av, b7, c0_7)
		av = Broadcast128D(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6, c1_7 = FMA128D(av, b0, c1_0), FMA128D(av, b1, c1_1), FMA128D(av, b2, c1_2), FMA128D(av, b3, c1_3), FMA128D(av, b4, c1_4), FMA128D(av, b5, c1_5), FMA128D(av, b6, c1_6), FMA128D(av, b7, c1_7)
		av = Broadcast128D(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6, c2_7 = FMA128D(av, b0, c2_0), FMA128D(av, b1, c2_1), FMA128D(av, b2, c2_2), FMA128D(av, b3, c2_3), FMA128D(av, b4, c2_4), FMA128D(av, b5, c2_5), FMA128D(av, b6, c2_6), FMA128D(av, b7, c2_7)
		av = Broadcast128D(ap[3])
		c3_0, c3_1, c3_2, c3_3, c3_4, c3_5, c3_6, c3_7 = FMA128D(av, b0, c3_0), FMA128D(av, b1, c3_1), FMA128D(av, b2, c3_2), FMA128D(av, b3, c3_3), FMA128D(av, b4, c3_4), FMA128D(av, b5, c3_5), FMA128D(av, b6, c3_6), FMA128D(av, b7, c3_7)
		ap, bp = ap[4:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	Store128D(r0[0:2], Add128D(Load128D(r0[0:2]), c0_0))
	Store128D(r0[2:4], Add128D(Load128D(r0[2:4]), c0_1))
	Store128D(r0[4:6], Add128D(Load128D(r0[4:6]), c0_2))
	Store128D(r0[6:8], Add128D(Load128D(r0[6:8]), c0_3))
	Store128D(r0[8:10], Add128D(Load128D(r0[8:10]), c0_4))
	Store128D(r0[10:12], Add128D(Load128D(r0[10:12]), c0_5))
	Store128D(r0[12:14], Add128D(Load128D(r0[12:14]), c0_6))
	Store128D(r0[14:16], Add128D(Load128D(r0[14:16]), c0_7))
	r1 := c[1*ldc : 1*ldc+16]
	Store128D(r1[0:2], Add128D(Load128D(r1[0:2]), c1_0))
	Store128D(r1[2:4], Add128D(Load128D(r1[2:4]), c1_1))
	Store128D(r1[4:6], Add128D(Load128D(r1[4:6]), c1_2))
	Store128D(r1[6:8], Add128D(Load128D(r1[6:8]), c1_3))
	Store128D(r1[8:10], Add128D(Load128D(r1[8:10]), c1_4))
	Store128D(r1[10:12], Add128D(Load128D(r1[10:12]), c1_5))
	Store128D(r1[12:14], Add128D(Load128D(r1[12:14]), c1_6))
	Store128D(r1[14:16], Add128D(Load128D(r1[14:16]), c1_7))
	r2 := c[2*ldc : 2*ldc+16]
	Store128D(r2[0:2], Add128D(Load128D(r2[0:2]), c2_0))
	Store128D(r2[2:4], Add128D(Load128D(r2[2:4]), c2_1))
	Store128D(r2[4:6], Add128D(Load128D(r2[4:6]), c2_2))
	Store128D(r2[6:8], Add128D(Load128D(r2[6:8]), c2_3))
	Store128D(r2[8:10], Add128D(Load128D(r2[8:10]), c2_4))
	Store128D(r2[10:12], Add128D(Load128D(r2[10:12]), c2_5))
	Store128D(r2[12:14], Add128D(Load128D(r2[12:14]), c2_6))
	Store128D(r2[14:16], Add128D(Load128D(r2[14:16]), c2_7))
	r3 := c[3*ldc : 3*ldc+16]
	Store128D(r3[0:2], Add128D(Load128D(r3[0:2]), c3_0))
	Store128D(r3[2:4], Add128D(Load128D(r3[2:4]), c3_1))
	Store128D(r3[4:6], Add128D(Load128D(r3[4:6]), c3_2))
	Store128D(r3[6:8], Add128D(Load128D(r3[6:8]), c3_3))
	Store128D(r3[8:10], Add128D(Load128D(r3[8:10]), c3_4))
	Store128D(r3[10:12], Add128D(Load128D(r3[10:12]), c3_5))
	Store128D(r3[12:14], Add128D(Load128D(r3[12:14]), c3_6))
	Store128D(r3[14:16], Add128D(Load128D(r3[14:16]), c3_7))
}

// The pre-broadcast-A NEON float64 kernels (#198): gemm_neon.go's pre-broadcast shapes
// at half the columns (ALanes = 2). 5x8/8 ships by archive/neon-pb/prereg.md's rule
// (83.1%); 4x10/8 and 7x6/8 are its alternates. tools/shapegen -arch arm64 -elem f64
// -form pb emissions verbatim.

// KernelDPB5x8 is a generated candidate: 5x8 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB5x8(kc int, a, b, c []float64, ldc int) {
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

// KernelDPB4x10 is a generated candidate: 4x10 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB4x10(kc int, a, b, c []float64, ldc int) {
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

// KernelDPB7x6 is a generated candidate: 7x6 u=8 pb.
//
// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.
func KernelDPB7x6(kc int, a, b, c []float64, ldc int) {
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
