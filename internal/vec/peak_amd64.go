// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

// Vector FMA ceilings. peak.go carries the reasoning: why the denominator is
// measured rather than derived, what the four load-bearing properties are, why
// the chain counts are what they are, and why the accumulator sits in the
// multiplicand position.
//
// Every operation below is an existing hot-layer wrapper (Broadcast512, FMA512,
// Add512, HSum512). No new archsimd identifier is introduced here, which is
// deliberate: the P0 standing order makes any recalled-from-memory identifier
// presumptively wrong, and a measurement kernel is a bad place to find out.
//
// The multiplicands are hoisted out of the loop by hand rather than trusted to
// the compiler, because a Broadcast inside the loop would add a uop per
// iteration and depress the very ceiling being measured.
//
// Shape of the steady-state loop these compile to, which is the whole point and
// is asserted by gate-p2.sh: exactly Chains512 (resp. Chains256) VFMADD213PS,
// one INCQ, one CMPQ and one JGT. No loads, no stores, no register copies.

func avx512Peak(iters int) float32 {
	x := Broadcast512(peakOne())
	y := Broadcast512(peakOne())
	a0, a1, a2, a3 := Broadcast512(1), Broadcast512(2), Broadcast512(3), Broadcast512(4)
	a4, a5, a6, a7 := Broadcast512(5), Broadcast512(6), Broadcast512(7), Broadcast512(8)
	a8, a9, a10, a11 := Broadcast512(9), Broadcast512(10), Broadcast512(11), Broadcast512(12)
	for i := 0; i < iters; i++ {
		a0 = FMA512(a0, y, x)
		a1 = FMA512(a1, y, x)
		a2 = FMA512(a2, y, x)
		a3 = FMA512(a3, y, x)
		a4 = FMA512(a4, y, x)
		a5 = FMA512(a5, y, x)
		a6 = FMA512(a6, y, x)
		a7 = FMA512(a7, y, x)
		a8 = FMA512(a8, y, x)
		a9 = FMA512(a9, y, x)
		a10 = FMA512(a10, y, x)
		a11 = FMA512(a11, y, x)
	}
	s := Add512(Add512(Add512(a0, a1), Add512(a2, a3)), Add512(Add512(a4, a5), Add512(a6, a7)))
	s = Add512(s, Add512(Add512(a8, a9), Add512(a10, a11)))
	return HSum512(s)
}

func avx2Peak(iters int) float32 {
	x := Broadcast256(peakOne())
	y := Broadcast256(peakOne())
	a0, a1, a2, a3 := Broadcast256(1), Broadcast256(2), Broadcast256(3), Broadcast256(4)
	a4, a5, a6, a7 := Broadcast256(5), Broadcast256(6), Broadcast256(7), Broadcast256(8)
	a8, a9 := Broadcast256(9), Broadcast256(10)
	for i := 0; i < iters; i++ {
		a0 = FMA256(a0, y, x)
		a1 = FMA256(a1, y, x)
		a2 = FMA256(a2, y, x)
		a3 = FMA256(a3, y, x)
		a4 = FMA256(a4, y, x)
		a5 = FMA256(a5, y, x)
		a6 = FMA256(a6, y, x)
		a7 = FMA256(a7, y, x)
		a8 = FMA256(a8, y, x)
		a9 = FMA256(a9, y, x)
	}
	s := Add256(Add256(Add256(a0, a1), Add256(a2, a3)), Add256(Add256(a4, a5), Add256(a6, a7)))
	s = Add256(s, Add256(a8, a9))
	return HSum256(s)
}

// vectorPeakKernels gates on runtime feature detection for the same reason
// l1.vectorBackends does: GOAMD64 does not gate archsimd intrinsics
// (docs/toolchain-notes.md T7), so this check is the only backstop.
func vectorPeakKernels() []PeakKernel {
	var out []PeakKernel
	if HasAVX512() {
		out = append(out, PeakKernel{
			Name:         BackendAVX512,
			Chains:       Chains512,
			Lanes:        16,
			FlopsPerIter: Chains512 * 16 * 2,
			Fused:        true,
			Run:          avx512Peak,
		})
	}
	if HasAVX2() {
		out = append(out, PeakKernel{
			Name:         BackendAVX2,
			Chains:       Chains256,
			Lanes:        8,
			FlopsPerIter: Chains256 * 8 * 2,
			Fused:        true,
			Run:          avx2Peak,
		})
	}
	return out
}
