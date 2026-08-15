// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "testing"

// Registers the vector backends with the differential harness.
//
// Registration is gated on runtime CPU support, not just on the build tags:
// an amd64 binary built with GOEXPERIMENT=simd still must not execute AVX-512
// ops on a CPU that lacks them. A backend that is compiled in but unsupported
// here simply does not appear in the registry, and TestBackendCoverage then
// reports it as unexercised — which gate-p0.sh treats as a red gate rather
// than as a pass. The gate, not the test, is what refuses to accept partial
// coverage.
func init() {
	if HasAVX512() {
		backends = append(backends, backend{
			name:      BackendAVX512,
			Load:      AVX512Load,
			LoadPart:  AVX512LoadPart,
			Store:     AVX512Store,
			StorePart: AVX512StorePart,
			Broadcast: AVX512Broadcast,
			Zero:      AVX512Zero,
			Add:       AVX512Add,
			Sub:       AVX512Sub,
			Mul:       AVX512Mul,
			MulAdd:    AVX512MulAdd,
			Max:       AVX512Max,
			Min:       AVX512Min,
			Abs:       AVX512Abs,
			HSum:      AVX512HSum,
		})
	}
	if HasAVX2() {
		backends = append(backends, backend{
			name:      BackendAVX2,
			Load:      AVX2Load,
			LoadPart:  AVX2LoadPart,
			Store:     AVX2Store,
			StorePart: AVX2StorePart,
			Broadcast: AVX2Broadcast,
			Zero:      AVX2Zero,
			Add:       AVX2Add,
			Sub:       AVX2Sub,
			Mul:       AVX2Mul,
			MulAdd:    AVX2MulAdd,
			Max:       AVX2Max,
			Min:       AVX2Min,
			Abs:       AVX2Abs,
			HSum:      AVX2HSum,
		})
	}
}

// ------------------------------------------- the hoisted abs mask (#8, #54)

// TestAbsWithHoistedMask holds AbsWith512/AbsWith256 equal to the scalar spec
// when the mask is built *once* and reused across every input in the pool, which
// is how internal/l1's Asum kernels use them and the whole reason they exist
// (#8; the compiler will not lift the mask itself — #54, T18, golang/go#79984).
//
// # What this is worth, stated small
//
// Not much, and the reason to write the limit down is that the first draft of
// this comment claimed more. Abs512 delegates to AbsWith512, so TestDiffUnary
// already covers the sign-bit trick on every pool input; what is left here is
// narrow: that AbsMask512's output is a valid mask for AbsWith512 *as a
// caller-supplied argument*, independently of Abs512's composition. It is
// regression insurance for the day Abs512 stops delegating — then TestDiffUnary
// exercises whatever replaced it and this pair would otherwise go uncovered.
//
// It does not show the mask survives a loop in a register. Go may rematerialize
// a pure call like AbsMask512 at each use, so "hoisted" is a source-level
// property and nothing observable from inside the program distinguishes the two.
// #8's evidence for the hoist is the instruction count in
// docs/toolchain-notes.md, not this test.
//
// An earlier draft also compared the hoisted result against Abs512, the
// per-call wrapper, on the theory that a mask "clobbered or lane-shifted by the
// ops between uses" would pass TestDiffUnary and fail here. Both halves of that
// were wrong. Such a mask fails the spec comparison below, which is what makes
// that comparison the falsifier; and since Abs512 *is*
// AbsWith512(x, AbsMask512()), the dropped check compared f(x, g()) against
// f(x, g()) differing only in g's call site — it could not have come out
// otherwise (DESIGN.md §5.7).
func TestAbsWithHoistedMask(t *testing.T) {
	pool := blockPool()
	s := spec()

	if HasAVX512() {
		m := AbsMask512() // once, outside the loop, as the kernels do
		for _, x := range pool {
			got := blockOf512(AbsWith512(load512(x), m))
			if want := s.Abs(x); !sameBlock(got, want) {
				t.Errorf("AbsWith512 (hoisted mask) mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock(x), fmtBlock(got), fmtBlock(want))
			}
		}
	}
	if HasAVX2() {
		m := AbsMask256()
		for _, x := range pool {
			lo, hi := halves256(x)
			got := blockOf256(AbsWith256(lo, m), AbsWith256(hi, m))
			if want := s.Abs(x); !sameBlock(got, want) {
				t.Errorf("AbsWith256 (hoisted mask) mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock(x), fmtBlock(got), fmtBlock(want))
			}
		}
	}
}
