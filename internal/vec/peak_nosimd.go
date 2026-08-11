// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !goexperiment.simd || !amd64

package vec

// The exact complement of peak_amd64.go's build tag, so there is always exactly
// one definition of vectorPeakKernels and the scalar ceiling remains measurable
// on a stock toolchain on every GOARCH.
func vectorPeakKernels() []PeakKernel { return nil }
