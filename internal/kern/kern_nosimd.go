// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !(goexperiment.simd && amd64)

package kern

// vectorKernels is empty here: the exact complement of kern_amd64.go's build
// tag, so the scalar tile is what remains and `make stock` builds this package
// on any toolchain and any GOARCH.
func vectorKernels() []Kernel { return nil }

// referenceTiles is likewise empty: the shapes it names are AVX-512 loop bodies.
func referenceTiles() []Kernel { return nil }
