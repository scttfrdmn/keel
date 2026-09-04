// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !(goexperiment.simd && (amd64 || arm64))

package kern

// vectorKernels is empty here: the exact complement of the two backend build
// tags (kern_amd64.go and kern_arm64.go), so the scalar tile is what remains and
// `make stock` builds this package on any toolchain and any GOARCH. The tag gained
// `|| arm64` when the NEON backend landed (#136): without it, this file and
// kern_arm64.go would both define vectorKernels on arm64+simd.
func vectorKernels() []Kernel { return nil }

// referenceTiles is likewise empty: the shapes it names are AVX-512 loop bodies.
func referenceTiles() []Kernel { return nil }
