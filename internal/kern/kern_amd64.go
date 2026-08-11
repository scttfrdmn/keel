// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package kern

import "github.com/scttfrdmn/keel/internal/vec"

// vectorKernels lists the shipped shapes widest-tile-first. Both are zero-spill
// on go1.26.5; which is faster is a per-host measurement (see the package doc and
// KERNEL.md), so both are built and both are benchmarked.
//
// The loop bodies are in internal/vec, not here: reaching archsimd through this
// package's shims costs one anchor NOP per inlined wrapper per call site, which
// was 27% of the 2x32 loop body (docs/toolchain-notes.md T9, and the header
// comment of internal/vec/gemm_amd64.go for the measurement). This file is the
// shape registry; the arithmetic is one directory over, where the simd import
// already lives.
func vectorKernels() []Kernel {
	if !vec.HasAVX512() {
		return nil
	}
	return []Kernel{
		{Name: AVX512, MR: 4, NR: 32, Unroll: 1, Fn: vec.Kernel4x32},
		{Name: AVX512, MR: 2, NR: 32, Unroll: 4, Fn: vec.Kernel2x32},
	}
}

// ReferenceTile is DESIGN.md §4/P2's 12-accumulator tile, which spills. It is
// benchmarked and differentially tested but deliberately absent from Kernels(),
// so that nothing ships it and the gate's zero-spill criterion stays binding on
// the kernels that do. It exists to make the T10 register ceiling a measured
// number rather than a spill count.
var ReferenceTile = Kernel{Name: AVX512, MR: 6, NR: 32, Unroll: 4, Fn: vec.Kernel6x32}

// referenceTiles is what the benchmark adds to Kernels(): shapes that are
// interesting to measure but not to ship.
func referenceTiles() []Kernel { return []Kernel{ReferenceTile} }
