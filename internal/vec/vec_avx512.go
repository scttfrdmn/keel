// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

// AVX-512 backend, written in P0 against the *read* archsimd API.
// Intentionally empty in the skeleton: do not write intrinsic calls
// until go doc output for the active toolchain is in front of you.
