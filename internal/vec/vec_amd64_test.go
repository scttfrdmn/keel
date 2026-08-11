// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

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
