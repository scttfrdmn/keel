// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// Registers the NEON float64 backend with the float64 differential harness
// (vec_diff64_test.go), the arm64 mirror of vec_amd64_f64_test.go.
func init() {
	if HasNEON() {
		backends64 = append(backends64, backend64{
			name:      BackendNEON,
			Load:      NEONDLoad,
			LoadPart:  NEONDLoadPart,
			Store:     NEONDStore,
			StorePart: NEONDStorePart,
			Broadcast: NEONDBroadcast,
			Zero:      NEONDZero,
			Add:       NEONDAdd,
			Sub:       NEONDSub,
			Mul:       NEONDMul,
			Div:       NEONDDiv,
			MulAdd:    NEONDMulAdd,
			Max:       NEONDMax,
			Min:       NEONDMin,
			Abs:       NEONDAbs,
			HSum:      NEONDHSum,
		})
	}
}
