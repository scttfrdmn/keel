// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// Register the NEON backend in the differential harness (vec_diff_test.go), the
// arm64 mirror of vec_amd64_test.go's AVX2/AVX512 registration. On arm64+simd this
// init runs and every Block op above is held bit-exact against the scalar spec;
// on the dev host (Apple Silicon) it exercises real NEON instructions natively, so
// "differential tests first" (the #136 charter, DESIGN.md §5.2) is verified here
// before any NEON microkernel is written.
func init() {
	if HasNEON() {
		backends = append(backends, backend{
			name:      BackendNEON,
			Load:      NEONLoad,
			LoadPart:  NEONLoadPart,
			Store:     NEONStore,
			StorePart: NEONStorePart,
			Broadcast: NEONBroadcast,
			Zero:      NEONZero,
			Add:       NEONAdd,
			Sub:       NEONSub,
			Mul:       NEONMul,
			MulAdd:    NEONMulAdd,
			Max:       NEONMax,
			Min:       NEONMin,
			Abs:       NEONAbs,
			HSum:      NEONHSum,
		})
	}
}
