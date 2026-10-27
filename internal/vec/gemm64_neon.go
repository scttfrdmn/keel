// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

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
