// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

import "simd/archsimd"

// The NEON FMA ceiling as the maximum over a family of FMA-only loops (#198, Scott's
// ruling (b), 2026-10-27). archive/neon-raw/ found that loops of the same FMA count read
// very different rates on Neoverse: the old shared-multiplicand neonPeak read 1.26-1.89x
// below an FMA-only kernel loop, for a cause established on V2/V3 (every FMA read the
// same multiplicand registers) and NOT established on V1, where every counter loop tried
// read exactly 2.0 FMA/cycle against the kernel loop's 3.76. A ceiling must be at least
// every FMA rate a loop has achieved, so the denominator is the family's maximum, not
// any one member's, and that needs no causal story. BenchmarkPeak's "neon" row runs the
// member that measured highest on all three Neoverse generations, neonPeakFO; the
// family benchmark times every member each run, and gate-p2 refuses a run where some
// member beats it by more than 1%, so the maximality is witnessed, not assumed.
//
// Every member keeps peak.go's contract: no memory in the steady-state loop (audited in
// gate-p2), independent chains, and an exact Witness. Multiplicands are exactly 1.0,
// read from peakOnes so that CSE cannot merge registers that must stay distinct (the
// raw shared-multiplicand loop merged x and y into ONE register when both were the
// constant 1.0).

// ChainsFO is the FO members' chain count: the 3x6 accumulator tile of a 3x24 (float32)
// or 3x12 (float64) kernel.
const ChainsFO = 18

// PeakFamilyNEON returns every member of the NEON float32 ceiling family, canonical
// member first. Names are the member's: "fo" (canonical), "x44", and "shared", the
// neonPeak that was the denominator before #198.
func PeakFamilyNEON() []PeakKernel {
	return []PeakKernel{
		{Name: "fo", Chains: ChainsFO, Lanes: 4, FlopsPerIter: ChainsFO * 4 * 2, Fused: true, Run: neonPeakFO},
		{Name: "x44", Chains: ChainsNEON, Lanes: 4, FlopsPerIter: ChainsNEON * 4 * 2, Fused: true, Run: neonPeakX44},
		{Name: "shared", Chains: ChainsNEON, Lanes: 4, FlopsPerIter: ChainsNEON * 4 * 2, Fused: true, Run: neonPeak},
	}
}

// PeakFamilyNEON64 is PeakFamilyNEON for the float64 ceiling.
func PeakFamilyNEON64() []PeakKernel64 {
	return []PeakKernel64{
		{Name: "fo", Chains: ChainsFO, Lanes: 2, FlopsPerIter: ChainsFO * 2 * 2, Fused: true, Run: neonPeakFOD},
		{Name: "x44", Chains: ChainsNEON, Lanes: 2, FlopsPerIter: ChainsNEON * 2 * 2, Fused: true, Run: neonPeakX44D},
		{Name: "shared", Chains: ChainsNEON, Lanes: 2, FlopsPerIter: ChainsNEON * 2 * 2, Fused: true, Run: neonPeakD},
	}
}

// neonPeakX44 is a 4x4 outer product: 16 chains whose FMAs read 8 distinct multiplicand
// registers. On V2/V3 it reached the FMA-only kernel loop's rate (archive/neon-raw/).
func neonPeakX44(iters int) float32 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat32x4(peakOnes[0]), archsimd.BroadcastFloat32x4(peakOnes[1]), archsimd.BroadcastFloat32x4(peakOnes[2]), archsimd.BroadcastFloat32x4(peakOnes[3])
	y0, y1, y2, y3 := archsimd.BroadcastFloat32x4(peakOnes[4]), archsimd.BroadcastFloat32x4(peakOnes[5]), archsimd.BroadcastFloat32x4(peakOnes[6]), archsimd.BroadcastFloat32x4(peakOnes[7])
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

// neonPeakX44D is neonPeakX44 on Float64x2.
func neonPeakX44D(iters int) float64 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat64x2(peakOnesD[0]), archsimd.BroadcastFloat64x2(peakOnesD[1]), archsimd.BroadcastFloat64x2(peakOnesD[2]), archsimd.BroadcastFloat64x2(peakOnesD[3])
	y0, y1, y2, y3 := archsimd.BroadcastFloat64x2(peakOnesD[4]), archsimd.BroadcastFloat64x2(peakOnesD[5]), archsimd.BroadcastFloat64x2(peakOnesD[6]), archsimd.BroadcastFloat64x2(peakOnesD[7])
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

// The panels the FO members read. Only their first MR and NR elements are loaded (before
// the loop); the lengths drive its two-counter exit, which is the loop as measured.
var (
	peakPanelA  = peakFill32(3 * peakChunk)
	peakPanelB  = peakFill32(24 * peakChunk)
	peakPanelAD = peakFill64(3 * peakChunk)
	peakPanelBD = peakFill64(12 * peakChunk)
)

// neonPeakFO runs neonPeakTile over iters K-steps in peakChunk pieces. C starts at each
// chain's 1-based index in all four lanes, and every lane gains exactly a·b = 1 per
// K-step, so the sum is PeakKernel.Witness's l·(c(c+1)/2 + c·iters) with c = 18, l = 4.
func neonPeakFO(iters int) float32 {
	var c [3 * 24]float32
	for i := range c {
		c[i] = float32(i/4 + 1) // lanes 4k..4k+3 are chain k: row k/6, register k%6
	}
	for n := iters; n > 0; n -= peakChunk {
		neonPeakTile(min(n, peakChunk), peakPanelA, peakPanelB, c[:], 24)
	}
	var s float32
	for _, v := range c {
		s += v
	}
	return s
}

// neonPeakFOD is neonPeakFO on Float64x2: 18 chains of 2 lanes.
func neonPeakFOD(iters int) float64 {
	var c [3 * 12]float64
	for i := range c {
		c[i] = float64(i/2 + 1)
	}
	for n := iters; n > 0; n -= peakChunk {
		neonPeakTileD(min(n, peakChunk), peakPanelAD, peakPanelBD, c[:], 12)
	}
	var s float64
	for _, v := range c {
		s += v
	}
	return s
}

// neonPeakTile is the FMA-only kernel loop (archive/neon-raw/, KernelP3x24FO): shapegen's
// raw 3x24/1 emission with the B loads and A broadcasts hoisted out of the K-loop, so the
// loop is 18 FMAs and two-counter bookkeeping, 24 instructions. Derived mechanically by
// archive/neon-raw/probe-gen.py's FO transform; do not hand-edit.
func neonPeakTile(kc int, a, b, c []float32, ldc int) {
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

// neonPeakTileD is neonPeakTile on Float64x2: raw 3x12/1 with the same hoisting.
func neonPeakTileD(kc int, a, b, c []float64, ldc int) {
	var c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 archsimd.Float64x2
	var c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 archsimd.Float64x2
	var c2_0, c2_1, c2_2, c2_3, c2_4, c2_5 archsimd.Float64x2
	var b0, b1, b2, b3, b4, b5, av archsimd.Float64x2
	ap := a[:kc*3]
	bp := b[:kc*12]
	b0, b1, b2, b3, b4, b5 = archsimd.LoadFloat64x2Array((*[2]float64)(bp[0:2])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[2:4])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[4:6])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[6:8])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[8:10])), archsimd.LoadFloat64x2Array((*[2]float64)(bp[10:12]))
	a0, a1, a2 := archsimd.BroadcastFloat64x2(ap[0]), archsimd.BroadcastFloat64x2(ap[1]), archsimd.BroadcastFloat64x2(ap[2])
	for len(ap) >= 3 && len(bp) >= 12 {
		av = a0
		c0_0, c0_1, c0_2, c0_3, c0_4, c0_5 = av.MulAdd(b0, c0_0), av.MulAdd(b1, c0_1), av.MulAdd(b2, c0_2), av.MulAdd(b3, c0_3), av.MulAdd(b4, c0_4), av.MulAdd(b5, c0_5)
		av = a1
		c1_0, c1_1, c1_2, c1_3, c1_4, c1_5 = av.MulAdd(b0, c1_0), av.MulAdd(b1, c1_1), av.MulAdd(b2, c1_2), av.MulAdd(b3, c1_3), av.MulAdd(b4, c1_4), av.MulAdd(b5, c1_5)
		av = a2
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
