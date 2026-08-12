// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !(goexperiment.simd && amd64)

package kern

// The exact complement of class_amd64.go's build tag. With no vector kernels
// compiled in there is nothing for a class to choose between — the scalar
// fallback's shapes are a correctness path with no measurement behind either one,
// and Preferred leaves unaudited shapes in registry order — so the answer here is
// the one that changes no behaviour, and the evidence string says why rather than
// implying a machine was examined.

// HostClass reports ClassFMA: no vector shape exists in this build to choose
// between.
func HostClass() Class { return ClassFMA }

// HostClassEvidence explains that nothing was fingerprinted.
func HostClassEvidence() string {
	return "no vector backend in this build (class unused)"
}
