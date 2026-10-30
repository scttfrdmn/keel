// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// The AVX-512 FMA-ceiling family (#199): peak_family_arm64.go's question asked of
// avx512Peak, whose 12 FMAs all read one multiplicand and one addend register, the
// shared-operand structure that made neonPeak under-read by 1.26-1.89x on Neoverse
// (#198). ADOPTED 2026-10-28 (#199, Scott's ruling (a)): the exploration run
// archive/avx512-family/ found x44 at 112.5% of avx512Peak on Skylake-SP and tied with it,
// within 0.1%, on Zen 4 and Zen 5. So x44 is the canonical member: the "avx512" rows of
// BenchmarkPeak and BenchmarkFMARoof64 run it, and DESIGN.md P2's family-max rule extends to
// amd64.
//
// Every member keeps peak.go's contract (no memory in the steady-state loop, exact
// Witness). x44 keeps avx512Peak's accumulator-as-multiplicand form, a = a*y + x,
// which updates in place with no register copies (property 4); fo keeps the kernel's
// real form, c = av*b + c, copies included, because the family measures what loops
// achieve rather than an idealised version of them.

// ChainsFO512 is the amd64 fo members' chain count: the 4x2 accumulator tile of a
// 4x32 (float32) or 4x16 (float64) kernel.
const ChainsFO512 = 8

// PeakFamilyAVX512 returns every member of the AVX-512 float32 ceiling family, canonical
// member first: x44 since #199. "shared" is the old avx512Peak.
func PeakFamilyAVX512() []PeakKernel {
	if !HasAVX512() {
		return nil
	}
	return []PeakKernel{
		{Name: "x44", Chains: 16, Lanes: 16, FlopsPerIter: 16 * 16 * 2, Fused: true, Run: avx512PeakX44},
		{Name: "shared", Chains: Chains512, Lanes: 16, FlopsPerIter: Chains512 * 16 * 2, Fused: true, Run: avx512Peak},
		{Name: "fo", Chains: ChainsFO512, Lanes: 16, FlopsPerIter: ChainsFO512 * 16 * 2, Fused: true, Run: avx512PeakFO},
	}
}

// PeakFamilyAVX51264 is PeakFamilyAVX512 for the float64 ceiling.
func PeakFamilyAVX51264() []PeakKernel64 {
	if !HasAVX512() {
		return nil
	}
	return []PeakKernel64{
		{Name: "x44", Chains: 16, Lanes: 8, FlopsPerIter: 16 * 8 * 2, Fused: true, Run: avx512PeakX44D},
		{Name: "shared", Chains: Chains512, Lanes: 8, FlopsPerIter: Chains512 * 8 * 2, Fused: true, Run: avx512PeakD},
		{Name: "fo", Chains: ChainsFO512, Lanes: 8, FlopsPerIter: ChainsFO512 * 8 * 2, Fused: true, Run: avx512PeakFOD},
	}
}

// avx512PeakX44 is a 4x4 outer product: 16 chains whose FMAs read 8 distinct operand
// registers, in avx512Peak's in-place form.
func avx512PeakX44(iters int) float32 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat32x16(peakOnes[0]), archsimd.BroadcastFloat32x16(peakOnes[1]), archsimd.BroadcastFloat32x16(peakOnes[2]), archsimd.BroadcastFloat32x16(peakOnes[3])
	y0, y1, y2, y3 := archsimd.BroadcastFloat32x16(peakOnes[4]), archsimd.BroadcastFloat32x16(peakOnes[5]), archsimd.BroadcastFloat32x16(peakOnes[6]), archsimd.BroadcastFloat32x16(peakOnes[7])
	a0_0 := archsimd.BroadcastFloat32x16(1)
	a0_1 := archsimd.BroadcastFloat32x16(2)
	a0_2 := archsimd.BroadcastFloat32x16(3)
	a0_3 := archsimd.BroadcastFloat32x16(4)
	a1_0 := archsimd.BroadcastFloat32x16(5)
	a1_1 := archsimd.BroadcastFloat32x16(6)
	a1_2 := archsimd.BroadcastFloat32x16(7)
	a1_3 := archsimd.BroadcastFloat32x16(8)
	a2_0 := archsimd.BroadcastFloat32x16(9)
	a2_1 := archsimd.BroadcastFloat32x16(10)
	a2_2 := archsimd.BroadcastFloat32x16(11)
	a2_3 := archsimd.BroadcastFloat32x16(12)
	a3_0 := archsimd.BroadcastFloat32x16(13)
	a3_1 := archsimd.BroadcastFloat32x16(14)
	a3_2 := archsimd.BroadcastFloat32x16(15)
	a3_3 := archsimd.BroadcastFloat32x16(16)
	for i := 0; i < iters; i++ {
		a0_0 = a0_0.MulAdd(y0, x0)
		a0_1 = a0_1.MulAdd(y1, x0)
		a0_2 = a0_2.MulAdd(y2, x0)
		a0_3 = a0_3.MulAdd(y3, x0)
		a1_0 = a1_0.MulAdd(y0, x1)
		a1_1 = a1_1.MulAdd(y1, x1)
		a1_2 = a1_2.MulAdd(y2, x1)
		a1_3 = a1_3.MulAdd(y3, x1)
		a2_0 = a2_0.MulAdd(y0, x2)
		a2_1 = a2_1.MulAdd(y1, x2)
		a2_2 = a2_2.MulAdd(y2, x2)
		a2_3 = a2_3.MulAdd(y3, x2)
		a3_0 = a3_0.MulAdd(y0, x3)
		a3_1 = a3_1.MulAdd(y1, x3)
		a3_2 = a3_2.MulAdd(y2, x3)
		a3_3 = a3_3.MulAdd(y3, x3)
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
	return HSum512(s)
}

// avx512PeakX44D is avx512PeakX44 on Float64x8.
func avx512PeakX44D(iters int) float64 {
	x0, x1, x2, x3 := archsimd.BroadcastFloat64x8(peakOnesD[0]), archsimd.BroadcastFloat64x8(peakOnesD[1]), archsimd.BroadcastFloat64x8(peakOnesD[2]), archsimd.BroadcastFloat64x8(peakOnesD[3])
	y0, y1, y2, y3 := archsimd.BroadcastFloat64x8(peakOnesD[4]), archsimd.BroadcastFloat64x8(peakOnesD[5]), archsimd.BroadcastFloat64x8(peakOnesD[6]), archsimd.BroadcastFloat64x8(peakOnesD[7])
	a0_0 := archsimd.BroadcastFloat64x8(1)
	a0_1 := archsimd.BroadcastFloat64x8(2)
	a0_2 := archsimd.BroadcastFloat64x8(3)
	a0_3 := archsimd.BroadcastFloat64x8(4)
	a1_0 := archsimd.BroadcastFloat64x8(5)
	a1_1 := archsimd.BroadcastFloat64x8(6)
	a1_2 := archsimd.BroadcastFloat64x8(7)
	a1_3 := archsimd.BroadcastFloat64x8(8)
	a2_0 := archsimd.BroadcastFloat64x8(9)
	a2_1 := archsimd.BroadcastFloat64x8(10)
	a2_2 := archsimd.BroadcastFloat64x8(11)
	a2_3 := archsimd.BroadcastFloat64x8(12)
	a3_0 := archsimd.BroadcastFloat64x8(13)
	a3_1 := archsimd.BroadcastFloat64x8(14)
	a3_2 := archsimd.BroadcastFloat64x8(15)
	a3_3 := archsimd.BroadcastFloat64x8(16)
	for i := 0; i < iters; i++ {
		a0_0 = a0_0.MulAdd(y0, x0)
		a0_1 = a0_1.MulAdd(y1, x0)
		a0_2 = a0_2.MulAdd(y2, x0)
		a0_3 = a0_3.MulAdd(y3, x0)
		a1_0 = a1_0.MulAdd(y0, x1)
		a1_1 = a1_1.MulAdd(y1, x1)
		a1_2 = a1_2.MulAdd(y2, x1)
		a1_3 = a1_3.MulAdd(y3, x1)
		a2_0 = a2_0.MulAdd(y0, x2)
		a2_1 = a2_1.MulAdd(y1, x2)
		a2_2 = a2_2.MulAdd(y2, x2)
		a2_3 = a2_3.MulAdd(y3, x2)
		a3_0 = a3_0.MulAdd(y0, x3)
		a3_1 = a3_1.MulAdd(y1, x3)
		a3_2 = a3_2.MulAdd(y2, x3)
		a3_3 = a3_3.MulAdd(y3, x3)
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
	return HSum512D(s)
}

var (
	peakPanelA512  = peakFill32(4 * peakChunk)
	peakPanelB512  = peakFill32(32 * peakChunk)
	peakPanelAD512 = peakFill64(4 * peakChunk)
	peakPanelBD512 = peakFill64(16 * peakChunk)
)

// avx512PeakFO runs avx512PeakTile over iters K-steps in peakChunk pieces, C pre-set to
// each chain's 1-based index in all 16 lanes, so the sum is the Witness with c = 8.
func avx512PeakFO(iters int) float32 {
	var c [4 * 32]float32
	for i := range c {
		c[i] = float32(i/16 + 1) // lanes 16k..16k+15 are chain k: row k/2, half k%2
	}
	for n := iters; n > 0; n -= peakChunk {
		avx512PeakTile(min(n, peakChunk), peakPanelA512, peakPanelB512, c[:], 32)
	}
	var s float32
	for _, v := range c {
		s += v
	}
	return s
}

// avx512PeakFOD is avx512PeakFO on Float64x8: 8 chains of 8 lanes.
func avx512PeakFOD(iters int) float64 {
	var c [4 * 16]float64
	for i := range c {
		c[i] = float64(i/8 + 1)
	}
	for n := iters; n > 0; n -= peakChunk {
		avx512PeakTileD(min(n, peakChunk), peakPanelAD512, peakPanelBD512, c[:], 16)
	}
	var s float64
	for _, v := range c {
		s += v
	}
	return s
}

// avx512PeakTile is the shipped-shape FMA-only loop: tools/shapegen's 4x32/1 emission
// with the B loads and A broadcasts hoisted out of the K-loop, the transform
// archive/neon-raw/probe-gen.py applies. Do not hand-edit. 4x32, not the 12-chain 6x32:
// hoisted, 6x32 needs more than the 15 vector registers go1.27.3 allocates (T10, #18)
// and spilled 22 times in its loop, which no ceiling may.
func avx512PeakTile(kc int, a, b, c []float32, ldc int) {
	var (
		c0l, c0h archsimd.Float32x16
		c1l, c1h archsimd.Float32x16
		c2l, c2h archsimd.Float32x16
		c3l, c3h archsimd.Float32x16
	)
	var bl, bh, av archsimd.Float32x16

	ap := a[:kc*4]
	bp := b[:kc*32]

	bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
	a0, a1, a2, a3 := archsimd.BroadcastFloat32x16(ap[0]), archsimd.BroadcastFloat32x16(ap[1]), archsimd.BroadcastFloat32x16(ap[2]), archsimd.BroadcastFloat32x16(ap[3])
	for len(ap) >= 4 && len(bp) >= 32 {
		av = a0
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = a1
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = a2
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = a3
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		ap, bp = ap[4:], bp[32:]
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
	archsimd.ClearAVXUpperBits()
}

// avx512PeakTileD is avx512PeakTile at float64: 4x16/1 with the same hoisting.
func avx512PeakTileD(kc int, a, b, c []float64, ldc int) {
	var (
		c0l, c0h archsimd.Float64x8
		c1l, c1h archsimd.Float64x8
		c2l, c2h archsimd.Float64x8
		c3l, c3h archsimd.Float64x8
	)
	var bl, bh, av archsimd.Float64x8

	ap := a[:kc*4]
	bp := b[:kc*16]

	bl, bh = archsimd.LoadFloat64x8(bp[0:8]), archsimd.LoadFloat64x8(bp[8:16])
	a0, a1, a2, a3 := archsimd.BroadcastFloat64x8(ap[0]), archsimd.BroadcastFloat64x8(ap[1]), archsimd.BroadcastFloat64x8(ap[2]), archsimd.BroadcastFloat64x8(ap[3])
	for len(ap) >= 4 && len(bp) >= 16 {
		av = a0
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = a1
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = a2
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = a3
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		ap, bp = ap[4:], bp[16:]
	}

	r := c[0*ldc : 0*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c0l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c0h).Store(r[8:16])
	r = c[1*ldc : 1*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c1l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c1h).Store(r[8:16])
	r = c[2*ldc : 2*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c2l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c2h).Store(r[8:16])
	r = c[3*ldc : 3*ldc+16]
	archsimd.LoadFloat64x8(r[0:8]).Add(c3l).Store(r[0:8])
	archsimd.LoadFloat64x8(r[8:16]).Add(c3h).Store(r[8:16])
	archsimd.ClearAVXUpperBits()
}

func peakFamily() []PeakKernel { return PeakFamilyAVX512() }

func peakFamily64() []PeakKernel64 { return PeakFamilyAVX51264() }
