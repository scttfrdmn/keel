// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

// AVX2 backend (256-bit). Written in P0 alongside the AVX-512 path.
