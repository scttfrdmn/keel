// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

// NEON SGEMM microkernels — the five #136 candidates. Each computes C += A·B for
// one MR×NR tile from k-major packed panels, in the reflected-tile broadcast-A
// shape the codegen read fixed (docs/neon-sweep.md): vectors run along N as
// Float32x4 columns, the A operand is a scalar broadcast (VDUP), MulAdd is a fused
// in-place VFMLA. Straight-line, no calls in the K-loop, pointer-free, BCE-clean
// under the len() guard — the P2 kernel rules, ported.
//
// Three shapes the register model fits and two it predicts spill, all written to
// the same standard so the -S audit grades a real prediction rather than a strawman
// (Kernel6x32's ghost): the model proposes, the assembly disposes. Unroll is 1 for
// every shape, which is the form the model live = MR·(NR/4) + NR/4 + 1 describes;
// unrolled variants are a follow-up the sweep may motivate, not a candidate here.
//
// These are built on the vec shim's own hot wrappers (Load128/Broadcast128/FMA128/
// Add128/Store128), which vec_arm64_test.go holds bit-exact against the scalar spec,
// so a differential failure on a kernel is a kernel bug, not a shim question.
package vec

// Kernel8x8 computes C += A·B for one 8x8 tile from k-major packed
// panels, one k-step per pass. 8×2 = 16 accumulators, 2 B-vectors, 1
// broadcast: live = 16+2+1 = 19 of 32 V-registers (fits).
func Kernel8x8(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1 F32x4
	var c1_0, c1_1 F32x4
	var c2_0, c2_1 F32x4
	var c3_0, c3_1 F32x4
	var c4_0, c4_1 F32x4
	var c5_0, c5_1 F32x4
	var c6_0, c6_1 F32x4
	var c7_0, c7_1 F32x4
	var b0, b1, av F32x4
	ap := a[:kc*8]
	bp := b[:kc*8]
	for len(ap) >= 8 && len(bp) >= 8 {
		b0, b1 = Load128(bp[0:4]), Load128(bp[4:8])
		av = Broadcast128(ap[0])
		c0_0, c0_1 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1)
		av = Broadcast128(ap[1])
		c1_0, c1_1 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1)
		av = Broadcast128(ap[2])
		c2_0, c2_1 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1)
		av = Broadcast128(ap[3])
		c3_0, c3_1 = FMA128(av, b0, c3_0), FMA128(av, b1, c3_1)
		av = Broadcast128(ap[4])
		c4_0, c4_1 = FMA128(av, b0, c4_0), FMA128(av, b1, c4_1)
		av = Broadcast128(ap[5])
		c5_0, c5_1 = FMA128(av, b0, c5_0), FMA128(av, b1, c5_1)
		av = Broadcast128(ap[6])
		c6_0, c6_1 = FMA128(av, b0, c6_0), FMA128(av, b1, c6_1)
		av = Broadcast128(ap[7])
		c7_0, c7_1 = FMA128(av, b0, c7_0), FMA128(av, b1, c7_1)
		ap, bp = ap[8:], bp[8:]
	}
	r0 := c[0*ldc : 0*ldc+8]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	r1 := c[1*ldc : 1*ldc+8]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	r2 := c[2*ldc : 2*ldc+8]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	r3 := c[3*ldc : 3*ldc+8]
	Store128(r3[0:4], Add128(Load128(r3[0:4]), c3_0))
	Store128(r3[4:8], Add128(Load128(r3[4:8]), c3_1))
	r4 := c[4*ldc : 4*ldc+8]
	Store128(r4[0:4], Add128(Load128(r4[0:4]), c4_0))
	Store128(r4[4:8], Add128(Load128(r4[4:8]), c4_1))
	r5 := c[5*ldc : 5*ldc+8]
	Store128(r5[0:4], Add128(Load128(r5[0:4]), c5_0))
	Store128(r5[4:8], Add128(Load128(r5[4:8]), c5_1))
	r6 := c[6*ldc : 6*ldc+8]
	Store128(r6[0:4], Add128(Load128(r6[0:4]), c6_0))
	Store128(r6[4:8], Add128(Load128(r6[4:8]), c6_1))
	r7 := c[7*ldc : 7*ldc+8]
	Store128(r7[0:4], Add128(Load128(r7[0:4]), c7_0))
	Store128(r7[4:8], Add128(Load128(r7[4:8]), c7_1))
}

// Kernel8x12 computes C += A·B for one 8x12 tile from k-major packed
// panels, one k-step per pass. 8×3 = 24 accumulators, 3 B-vectors, 1
// broadcast: live = 24+3+1 = 28 of 32 V-registers (fits).
func Kernel8x12(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2 F32x4
	var c1_0, c1_1, c1_2 F32x4
	var c2_0, c2_1, c2_2 F32x4
	var c3_0, c3_1, c3_2 F32x4
	var c4_0, c4_1, c4_2 F32x4
	var c5_0, c5_1, c5_2 F32x4
	var c6_0, c6_1, c6_2 F32x4
	var c7_0, c7_1, c7_2 F32x4
	var b0, b1, b2, av F32x4
	ap := a[:kc*8]
	bp := b[:kc*12]
	for len(ap) >= 8 && len(bp) >= 12 {
		b0, b1, b2 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2)
		av = Broadcast128(ap[3])
		c3_0, c3_1, c3_2 = FMA128(av, b0, c3_0), FMA128(av, b1, c3_1), FMA128(av, b2, c3_2)
		av = Broadcast128(ap[4])
		c4_0, c4_1, c4_2 = FMA128(av, b0, c4_0), FMA128(av, b1, c4_1), FMA128(av, b2, c4_2)
		av = Broadcast128(ap[5])
		c5_0, c5_1, c5_2 = FMA128(av, b0, c5_0), FMA128(av, b1, c5_1), FMA128(av, b2, c5_2)
		av = Broadcast128(ap[6])
		c6_0, c6_1, c6_2 = FMA128(av, b0, c6_0), FMA128(av, b1, c6_1), FMA128(av, b2, c6_2)
		av = Broadcast128(ap[7])
		c7_0, c7_1, c7_2 = FMA128(av, b0, c7_0), FMA128(av, b1, c7_1), FMA128(av, b2, c7_2)
		ap, bp = ap[8:], bp[12:]
	}
	r0 := c[0*ldc : 0*ldc+12]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	Store128(r0[8:12], Add128(Load128(r0[8:12]), c0_2))
	r1 := c[1*ldc : 1*ldc+12]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	Store128(r1[8:12], Add128(Load128(r1[8:12]), c1_2))
	r2 := c[2*ldc : 2*ldc+12]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	Store128(r2[8:12], Add128(Load128(r2[8:12]), c2_2))
	r3 := c[3*ldc : 3*ldc+12]
	Store128(r3[0:4], Add128(Load128(r3[0:4]), c3_0))
	Store128(r3[4:8], Add128(Load128(r3[4:8]), c3_1))
	Store128(r3[8:12], Add128(Load128(r3[8:12]), c3_2))
	r4 := c[4*ldc : 4*ldc+12]
	Store128(r4[0:4], Add128(Load128(r4[0:4]), c4_0))
	Store128(r4[4:8], Add128(Load128(r4[4:8]), c4_1))
	Store128(r4[8:12], Add128(Load128(r4[8:12]), c4_2))
	r5 := c[5*ldc : 5*ldc+12]
	Store128(r5[0:4], Add128(Load128(r5[0:4]), c5_0))
	Store128(r5[4:8], Add128(Load128(r5[4:8]), c5_1))
	Store128(r5[8:12], Add128(Load128(r5[8:12]), c5_2))
	r6 := c[6*ldc : 6*ldc+12]
	Store128(r6[0:4], Add128(Load128(r6[0:4]), c6_0))
	Store128(r6[4:8], Add128(Load128(r6[4:8]), c6_1))
	Store128(r6[8:12], Add128(Load128(r6[8:12]), c6_2))
	r7 := c[7*ldc : 7*ldc+12]
	Store128(r7[0:4], Add128(Load128(r7[0:4]), c7_0))
	Store128(r7[4:8], Add128(Load128(r7[4:8]), c7_1))
	Store128(r7[8:12], Add128(Load128(r7[8:12]), c7_2))
}

// Kernel4x16 computes C += A·B for one 4x16 tile from k-major packed
// panels, one k-step per pass. 4×4 = 16 accumulators, 4 B-vectors, 1
// broadcast: live = 16+4+1 = 21 of 32 V-registers (fits).
func Kernel4x16(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 F32x4
	var c1_0, c1_1, c1_2, c1_3 F32x4
	var c2_0, c2_1, c2_2, c2_3 F32x4
	var c3_0, c3_1, c3_2, c3_3 F32x4
	var b0, b1, b2, b3, av F32x4
	ap := a[:kc*4]
	bp := b[:kc*16]
	for len(ap) >= 4 && len(bp) >= 16 {
		b0, b1, b2, b3 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12]), Load128(bp[12:16])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2, c0_3 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2, c1_3 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2, c2_3 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3)
		av = Broadcast128(ap[3])
		c3_0, c3_1, c3_2, c3_3 = FMA128(av, b0, c3_0), FMA128(av, b1, c3_1), FMA128(av, b2, c3_2), FMA128(av, b3, c3_3)
		ap, bp = ap[4:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	Store128(r0[8:12], Add128(Load128(r0[8:12]), c0_2))
	Store128(r0[12:16], Add128(Load128(r0[12:16]), c0_3))
	r1 := c[1*ldc : 1*ldc+16]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	Store128(r1[8:12], Add128(Load128(r1[8:12]), c1_2))
	Store128(r1[12:16], Add128(Load128(r1[12:16]), c1_3))
	r2 := c[2*ldc : 2*ldc+16]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	Store128(r2[8:12], Add128(Load128(r2[8:12]), c2_2))
	Store128(r2[12:16], Add128(Load128(r2[12:16]), c2_3))
	r3 := c[3*ldc : 3*ldc+16]
	Store128(r3[0:4], Add128(Load128(r3[0:4]), c3_0))
	Store128(r3[4:8], Add128(Load128(r3[4:8]), c3_1))
	Store128(r3[8:12], Add128(Load128(r3[8:12]), c3_2))
	Store128(r3[12:16], Add128(Load128(r3[12:16]), c3_3))
}

// Kernel3x24 computes C += A·B for one 3x24 tile from k-major packed panels,
// TWO k-steps per pass. 3×6 = 18 accumulators, 6 B-vectors, 1 broadcast:
// live = 18+6+1 = 25 of 32 V-registers (fits).
//
// This is `shapegen -arch arm64 -frontier`'s own winner: the leanest of the 107
// emittable zero-spill NEON shapes at 4.111 insns/FMA, against the shipped
// Kernel4x16's 5.000, which gate-p3 already states as SWEEP_BEST_IPF_ARM64 and
// reconciles live on every run. The unroll is 2 because u=1 reads 4.722 and u=4
// spills 11 — u=2 is the frontier, not a tuning choice.
//
// The body is the emitter's VERBATIM output with the package qualifier stripped,
// so the shape measured here is the shape the frontier is stated over rather
// than a hand transcription of it. Hence the two loops: the unrolled pass, then
// a single-k remainder pass for an odd kc.
//
// SHIPPED since 2026-10-05 (#136): this is the dispatched arm64 microkernel. It was
// staged as a referenceTile first so the sweep could rank it without the registry
// handing it dispatch on a static instruction count, and it won on measured rate --
// +10.2%/+8.9% at kc=128 on the two judged Neoverse parts and +6.62% at full
// Sgemm/n=2048, every interval disjoint. The staging rationale below is why it was
// measurable before it was shippable, and
// that staging is deliberate rather than cautious. It would win dispatch the
// moment an audited InsnsPerFMA were recorded on it — under either divisor, which
// #170 is why that needs saying: at the native width 3x24 ties 4x16 at exactly
// 0.5 mem-ops/FMA and takes the InsnsPerFMA tie-break, and under the Block width
// it won the primary axis outright at 1.0 against 1.25. So registering the count
// now would ship a shape on a static instruction count, which is the move
// 972ee47 had to undo in the other direction, and which #136's own caution 2
// forbids: rank on the sweep's measured rate, and treat the audit as a filter.
func Kernel3x24(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 F32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 F32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 F32x4
	var b0, b1, b2, b3, b4, b5, av F32x4
	ap := a[:kc*3]
	bp := b[:kc*24]
	for len(ap) >= 6 && len(bp) >= 48 {
		b0, b1, b2, b3, b4, b5 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12]), Load128(bp[12:16]), Load128(bp[16:20]), Load128(bp[20:24])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3), FMA128(av, b4, c0_4), FMA128(av, b5, c0_5)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3), FMA128(av, b4, c1_4), FMA128(av, b5, c1_5)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3), FMA128(av, b4, c2_4), FMA128(av, b5, c2_5)
		b0, b1, b2, b3, b4, b5 = Load128(bp[24:28]), Load128(bp[28:32]), Load128(bp[32:36]), Load128(bp[36:40]), Load128(bp[40:44]), Load128(bp[44:48])
		av = Broadcast128(ap[3])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3), FMA128(av, b4, c0_4), FMA128(av, b5, c0_5)
		av = Broadcast128(ap[4])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3), FMA128(av, b4, c1_4), FMA128(av, b5, c1_5)
		av = Broadcast128(ap[5])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3), FMA128(av, b4, c2_4), FMA128(av, b5, c2_5)
		ap, bp = ap[6:], bp[48:]
	}
	for len(ap) >= 3 && len(bp) >= 24 {
		b0, b1, b2, b3, b4, b5 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12]), Load128(bp[12:16]), Load128(bp[16:20]), Load128(bp[20:24])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3), FMA128(av, b4, c0_4), FMA128(av, b5, c0_5)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3), FMA128(av, b4, c1_4), FMA128(av, b5, c1_5)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3), FMA128(av, b4, c2_4), FMA128(av, b5, c2_5)
		ap, bp = ap[3:], bp[24:]
	}
	r0 := c[0*ldc : 0*ldc+24]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	Store128(r0[8:12], Add128(Load128(r0[8:12]), c0_2))
	Store128(r0[12:16], Add128(Load128(r0[12:16]), c0_3))
	Store128(r0[16:20], Add128(Load128(r0[16:20]), c0_4))
	Store128(r0[20:24], Add128(Load128(r0[20:24]), c0_5))
	r1 := c[1*ldc : 1*ldc+24]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	Store128(r1[8:12], Add128(Load128(r1[8:12]), c1_2))
	Store128(r1[12:16], Add128(Load128(r1[12:16]), c1_3))
	Store128(r1[16:20], Add128(Load128(r1[16:20]), c1_4))
	Store128(r1[20:24], Add128(Load128(r1[20:24]), c1_5))
	r2 := c[2*ldc : 2*ldc+24]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	Store128(r2[8:12], Add128(Load128(r2[8:12]), c2_2))
	Store128(r2[12:16], Add128(Load128(r2[12:16]), c2_3))
	Store128(r2[16:20], Add128(Load128(r2[16:20]), c2_4))
	Store128(r2[20:24], Add128(Load128(r2[20:24]), c2_5))
}

// Kernel8x16 computes C += A·B for one 8x16 tile from k-major packed
// panels, one k-step per pass. 8×4 = 32 accumulators, 4 B-vectors, 1
// broadcast: live = 32+4+1 = 37 of 32 V-registers (PREDICTED SPILL).
func Kernel8x16(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3 F32x4
	var c1_0, c1_1, c1_2, c1_3 F32x4
	var c2_0, c2_1, c2_2, c2_3 F32x4
	var c3_0, c3_1, c3_2, c3_3 F32x4
	var c4_0, c4_1, c4_2, c4_3 F32x4
	var c5_0, c5_1, c5_2, c5_3 F32x4
	var c6_0, c6_1, c6_2, c6_3 F32x4
	var c7_0, c7_1, c7_2, c7_3 F32x4
	var b0, b1, b2, b3, av F32x4
	ap := a[:kc*8]
	bp := b[:kc*16]
	for len(ap) >= 8 && len(bp) >= 16 {
		b0, b1, b2, b3 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12]), Load128(bp[12:16])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2, c0_3 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2, c1_3 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2, c2_3 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3)
		av = Broadcast128(ap[3])
		c3_0, c3_1, c3_2, c3_3 = FMA128(av, b0, c3_0), FMA128(av, b1, c3_1), FMA128(av, b2, c3_2), FMA128(av, b3, c3_3)
		av = Broadcast128(ap[4])
		c4_0, c4_1, c4_2, c4_3 = FMA128(av, b0, c4_0), FMA128(av, b1, c4_1), FMA128(av, b2, c4_2), FMA128(av, b3, c4_3)
		av = Broadcast128(ap[5])
		c5_0, c5_1, c5_2, c5_3 = FMA128(av, b0, c5_0), FMA128(av, b1, c5_1), FMA128(av, b2, c5_2), FMA128(av, b3, c5_3)
		av = Broadcast128(ap[6])
		c6_0, c6_1, c6_2, c6_3 = FMA128(av, b0, c6_0), FMA128(av, b1, c6_1), FMA128(av, b2, c6_2), FMA128(av, b3, c6_3)
		av = Broadcast128(ap[7])
		c7_0, c7_1, c7_2, c7_3 = FMA128(av, b0, c7_0), FMA128(av, b1, c7_1), FMA128(av, b2, c7_2), FMA128(av, b3, c7_3)
		ap, bp = ap[8:], bp[16:]
	}
	r0 := c[0*ldc : 0*ldc+16]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	Store128(r0[8:12], Add128(Load128(r0[8:12]), c0_2))
	Store128(r0[12:16], Add128(Load128(r0[12:16]), c0_3))
	r1 := c[1*ldc : 1*ldc+16]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	Store128(r1[8:12], Add128(Load128(r1[8:12]), c1_2))
	Store128(r1[12:16], Add128(Load128(r1[12:16]), c1_3))
	r2 := c[2*ldc : 2*ldc+16]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	Store128(r2[8:12], Add128(Load128(r2[8:12]), c2_2))
	Store128(r2[12:16], Add128(Load128(r2[12:16]), c2_3))
	r3 := c[3*ldc : 3*ldc+16]
	Store128(r3[0:4], Add128(Load128(r3[0:4]), c3_0))
	Store128(r3[4:8], Add128(Load128(r3[4:8]), c3_1))
	Store128(r3[8:12], Add128(Load128(r3[8:12]), c3_2))
	Store128(r3[12:16], Add128(Load128(r3[12:16]), c3_3))
	r4 := c[4*ldc : 4*ldc+16]
	Store128(r4[0:4], Add128(Load128(r4[0:4]), c4_0))
	Store128(r4[4:8], Add128(Load128(r4[4:8]), c4_1))
	Store128(r4[8:12], Add128(Load128(r4[8:12]), c4_2))
	Store128(r4[12:16], Add128(Load128(r4[12:16]), c4_3))
	r5 := c[5*ldc : 5*ldc+16]
	Store128(r5[0:4], Add128(Load128(r5[0:4]), c5_0))
	Store128(r5[4:8], Add128(Load128(r5[4:8]), c5_1))
	Store128(r5[8:12], Add128(Load128(r5[8:12]), c5_2))
	Store128(r5[12:16], Add128(Load128(r5[12:16]), c5_3))
	r6 := c[6*ldc : 6*ldc+16]
	Store128(r6[0:4], Add128(Load128(r6[0:4]), c6_0))
	Store128(r6[4:8], Add128(Load128(r6[4:8]), c6_1))
	Store128(r6[8:12], Add128(Load128(r6[8:12]), c6_2))
	Store128(r6[12:16], Add128(Load128(r6[12:16]), c6_3))
	r7 := c[7*ldc : 7*ldc+16]
	Store128(r7[0:4], Add128(Load128(r7[0:4]), c7_0))
	Store128(r7[4:8], Add128(Load128(r7[4:8]), c7_1))
	Store128(r7[8:12], Add128(Load128(r7[8:12]), c7_2))
	Store128(r7[12:16], Add128(Load128(r7[12:16]), c7_3))
}

// Kernel4x32 computes C += A·B for one 4x32 tile from k-major packed
// panels, one k-step per pass. 4×8 = 32 accumulators, 8 B-vectors, 1
// broadcast: live = 32+8+1 = 41 of 32 V-registers (PREDICTED SPILL).
func Kernel4x32(kc int, a, b, c []float32, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6, c0_7 F32x4
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6, c1_7 F32x4
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6, c2_7 F32x4
	var c3_0, c3_1, c3_2, c3_3, c3_4, c3_5, c3_6, c3_7 F32x4
	var b0, b1, b2, b3, b4, b5, b6, b7, av F32x4
	ap := a[:kc*4]
	bp := b[:kc*32]
	for len(ap) >= 4 && len(bp) >= 32 {
		b0, b1, b2, b3, b4, b5, b6, b7 = Load128(bp[0:4]), Load128(bp[4:8]), Load128(bp[8:12]), Load128(bp[12:16]), Load128(bp[16:20]), Load128(bp[20:24]), Load128(bp[24:28]), Load128(bp[28:32])
		av = Broadcast128(ap[0])
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5, c0_6, c0_7 = FMA128(av, b0, c0_0), FMA128(av, b1, c0_1), FMA128(av, b2, c0_2), FMA128(av, b3, c0_3), FMA128(av, b4, c0_4), FMA128(av, b5, c0_5), FMA128(av, b6, c0_6), FMA128(av, b7, c0_7)
		av = Broadcast128(ap[1])
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5, c1_6, c1_7 = FMA128(av, b0, c1_0), FMA128(av, b1, c1_1), FMA128(av, b2, c1_2), FMA128(av, b3, c1_3), FMA128(av, b4, c1_4), FMA128(av, b5, c1_5), FMA128(av, b6, c1_6), FMA128(av, b7, c1_7)
		av = Broadcast128(ap[2])
		c2_0, c2_1, c2_2, c2_3, c2_4, c2_5, c2_6, c2_7 = FMA128(av, b0, c2_0), FMA128(av, b1, c2_1), FMA128(av, b2, c2_2), FMA128(av, b3, c2_3), FMA128(av, b4, c2_4), FMA128(av, b5, c2_5), FMA128(av, b6, c2_6), FMA128(av, b7, c2_7)
		av = Broadcast128(ap[3])
		c3_0, c3_1, c3_2, c3_3, c3_4, c3_5, c3_6, c3_7 = FMA128(av, b0, c3_0), FMA128(av, b1, c3_1), FMA128(av, b2, c3_2), FMA128(av, b3, c3_3), FMA128(av, b4, c3_4), FMA128(av, b5, c3_5), FMA128(av, b6, c3_6), FMA128(av, b7, c3_7)
		ap, bp = ap[4:], bp[32:]
	}
	r0 := c[0*ldc : 0*ldc+32]
	Store128(r0[0:4], Add128(Load128(r0[0:4]), c0_0))
	Store128(r0[4:8], Add128(Load128(r0[4:8]), c0_1))
	Store128(r0[8:12], Add128(Load128(r0[8:12]), c0_2))
	Store128(r0[12:16], Add128(Load128(r0[12:16]), c0_3))
	Store128(r0[16:20], Add128(Load128(r0[16:20]), c0_4))
	Store128(r0[20:24], Add128(Load128(r0[20:24]), c0_5))
	Store128(r0[24:28], Add128(Load128(r0[24:28]), c0_6))
	Store128(r0[28:32], Add128(Load128(r0[28:32]), c0_7))
	r1 := c[1*ldc : 1*ldc+32]
	Store128(r1[0:4], Add128(Load128(r1[0:4]), c1_0))
	Store128(r1[4:8], Add128(Load128(r1[4:8]), c1_1))
	Store128(r1[8:12], Add128(Load128(r1[8:12]), c1_2))
	Store128(r1[12:16], Add128(Load128(r1[12:16]), c1_3))
	Store128(r1[16:20], Add128(Load128(r1[16:20]), c1_4))
	Store128(r1[20:24], Add128(Load128(r1[20:24]), c1_5))
	Store128(r1[24:28], Add128(Load128(r1[24:28]), c1_6))
	Store128(r1[28:32], Add128(Load128(r1[28:32]), c1_7))
	r2 := c[2*ldc : 2*ldc+32]
	Store128(r2[0:4], Add128(Load128(r2[0:4]), c2_0))
	Store128(r2[4:8], Add128(Load128(r2[4:8]), c2_1))
	Store128(r2[8:12], Add128(Load128(r2[8:12]), c2_2))
	Store128(r2[12:16], Add128(Load128(r2[12:16]), c2_3))
	Store128(r2[16:20], Add128(Load128(r2[16:20]), c2_4))
	Store128(r2[20:24], Add128(Load128(r2[20:24]), c2_5))
	Store128(r2[24:28], Add128(Load128(r2[24:28]), c2_6))
	Store128(r2[28:32], Add128(Load128(r2[28:32]), c2_7))
	r3 := c[3*ldc : 3*ldc+32]
	Store128(r3[0:4], Add128(Load128(r3[0:4]), c3_0))
	Store128(r3[4:8], Add128(Load128(r3[4:8]), c3_1))
	Store128(r3[8:12], Add128(Load128(r3[8:12]), c3_2))
	Store128(r3[12:16], Add128(Load128(r3[12:16]), c3_3))
	Store128(r3[16:20], Add128(Load128(r3[16:20]), c3_4))
	Store128(r3[20:24], Add128(Load128(r3[20:24]), c3_5))
	Store128(r3[24:28], Add128(Load128(r3[24:28]), c3_6))
	Store128(r3[28:32], Add128(Load128(r3[28:32]), c3_7))
}
