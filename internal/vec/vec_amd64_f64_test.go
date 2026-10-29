// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "testing"

// Registers the AVX-512 float64 backend with the float64 differential harness,
// gated on runtime CPU support as in vec_amd64_test.go.
func init() {
	if HasAVX512() {
		backends64 = append(backends64, backend64{
			name:      BackendAVX512,
			Load:      AVX512DLoad,
			LoadPart:  AVX512DLoadPart,
			Store:     AVX512DStore,
			StorePart: AVX512DStorePart,
			Broadcast: AVX512DBroadcast,
			Zero:      AVX512DZero,
			Add:       AVX512DAdd,
			Sub:       AVX512DSub,
			Mul:       AVX512DMul,
			Div:       AVX512DDiv,
			MulAdd:    AVX512DMulAdd,
			Max:       AVX512DMax,
			Min:       AVX512DMin,
			Abs:       AVX512DAbs,
			HSum:      AVX512DHSum,
		})
	}
	// The AVX2 rung (#192), registered as vec_amd64_test.go registers float32's.
	if HasAVX2() {
		backends64 = append(backends64, backend64{
			name:      BackendAVX2,
			Load:      AVX2DLoad,
			LoadPart:  AVX2DLoadPart,
			Store:     AVX2DStore,
			StorePart: AVX2DStorePart,
			Broadcast: AVX2DBroadcast,
			Zero:      AVX2DZero,
			Add:       AVX2DAdd,
			Sub:       AVX2DSub,
			Mul:       AVX2DMul,
			Div:       AVX2DDiv,
			MulAdd:    AVX2DMulAdd,
			Max:       AVX2DMax,
			Min:       AVX2DMin,
			Abs:       AVX2DAbs,
			HSum:      AVX2DHSum,
		})
	}
}

// TestAbsWithHoistedMask64 is TestAbsWithHoistedMask for AbsWith512D: the mask
// built once and reused, held to the spec. Same narrow worth, stated there.
func TestAbsWithHoistedMask64(t *testing.T) {
	if !HasAVX512() {
		t.Skip("no AVX-512 on this CPU")
	}
	s := spec64()
	m := AbsMask512D()
	for _, x := range blockPool64() {
		got := block64Of512(AbsWith512D(load512D(x), m))
		if want := s.Abs(x); !sameBlock64(got, want) {
			t.Errorf("AbsWith512D (hoisted mask) mismatch\n in:   %s\n got:  %s\n spec: %s",
				fmtBlock64(x), fmtBlock64(got), fmtBlock64(want))
		}
	}
}

// TestAbsWithHoistedMask256D is TestAbsWithHoistedMask64 for AbsWith256D.
func TestAbsWithHoistedMask256D(t *testing.T) {
	if !HasAVX2() {
		t.Skip("no AVX2+FMA on this CPU")
	}
	s := spec64()
	m := AbsMask256D()
	for _, x := range blockPool64() {
		l, h := halves256D(x)
		got := block64Of256(AbsWith256D(l, m), AbsWith256D(h, m))
		if want := s.Abs(x); !sameBlock64(got, want) {
			t.Errorf("AbsWith256D (hoisted mask) mismatch\n in:   %s\n got:  %s\n spec: %s",
				fmtBlock64(x), fmtBlock64(got), fmtBlock64(want))
		}
	}
}
