// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

// The NEON FMA ceiling — the arm64 compute arm of the measured denominator (#11).
// peak.go carries the reasoning for why the denominator is MEASURED, not derived,
// and the four properties that make this a ceiling rather than a loop.
//
// This is the arm64 ceiling instrument's inception: before #137 the arm64 build had
// no vector peak, so peak_nosimd.go answered with the SCALAR ceiling, and a NEON
// kernel divided by it would have read far above 100% of a peak that was not its
// arch's. Born here, register-only NEON FMA saturation; the arm64 baselines #137
// registers are scoped to the era in which THIS instrument is the reference (§5 rule
// 17(d)) — stated in the launch record, not left implicit.
//
// Property 4 (one instruction per chain, no spill/copy) holds more cleanly than on
// amd64: arm64's VFMLA accumulates IN PLACE (#136), so the natural accumulator form
// `a = FMA128(x, y, a)` — accumulator as the ADDEND — lowers to one VFMLA per chain
// with no register-preserving copy, where amd64's 213-form forced 26 VMOVDQU64 copies.
// No new archsimd identifier is introduced: Broadcast128, FMA128, Add128, HSum128 are
// the differentially-tested hot wrappers from vec_neon.go.
package vec

func neonPeak(iters int) float32 {
	x := Broadcast128(peakOne())
	y := Broadcast128(peakOne())
	a0, a1, a2, a3 := Broadcast128(1), Broadcast128(2), Broadcast128(3), Broadcast128(4)
	a4, a5, a6, a7 := Broadcast128(5), Broadcast128(6), Broadcast128(7), Broadcast128(8)
	a8, a9, a10, a11 := Broadcast128(9), Broadcast128(10), Broadcast128(11), Broadcast128(12)
	a12, a13, a14, a15 := Broadcast128(13), Broadcast128(14), Broadcast128(15), Broadcast128(16)
	for i := 0; i < iters; i++ {
		a0 = FMA128(x, y, a0)
		a1 = FMA128(x, y, a1)
		a2 = FMA128(x, y, a2)
		a3 = FMA128(x, y, a3)
		a4 = FMA128(x, y, a4)
		a5 = FMA128(x, y, a5)
		a6 = FMA128(x, y, a6)
		a7 = FMA128(x, y, a7)
		a8 = FMA128(x, y, a8)
		a9 = FMA128(x, y, a9)
		a10 = FMA128(x, y, a10)
		a11 = FMA128(x, y, a11)
		a12 = FMA128(x, y, a12)
		a13 = FMA128(x, y, a13)
		a14 = FMA128(x, y, a14)
		a15 = FMA128(x, y, a15)
	}
	s := Add128(Add128(Add128(Add128(a0, a1), Add128(a2, a3)), Add128(Add128(a4, a5), Add128(a6, a7))), Add128(Add128(Add128(a8, a9), Add128(a10, a11)), Add128(Add128(a12, a13), Add128(a14, a15))))
	return HSum128(s)
}

// vectorPeakKernels is the arm64 half: the NEON ceiling, gated on HasNEON (always
// true on arm64, kept for symmetry with the amd64 feature gate). FlopsPerIter is
// chains x 4 lanes x 2 flops per FMA.
func vectorPeakKernels() []PeakKernel {
	if !HasNEON() {
		return nil
	}
	return []PeakKernel{{
		Name:         BackendNEON,
		Chains:       ChainsNEON,
		Lanes:        4,
		FlopsPerIter: ChainsNEON * 4 * 2,
		Fused:        true,
		Run:          neonPeak,
	}}
}
