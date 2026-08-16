// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package kern

import "github.com/scttfrdmn/keel/internal/vec"

// vectorKernels lists the shipped shapes widest-tile-first. Both are zero-spill
// on go1.26.5; which is faster is a per-host measurement (see the package doc and
// KERNEL.md), so both are built and both are benchmarked.
//
// THIS ORDER NO LONGER DECIDES WHAT SHIPS. It did, and that was issue #24: the
// first matching entry won on every host, so every host ran 4×32 including the one
// where 2×32 measures 11 percentage points faster. Dispatch now asks Preferred with
// the host's class (class.go), and this list is the candidate set. Order still
// decides between shapes the class rule cannot separate — two shapes with equal
// insns/FMA and equal loads/FMA, or shapes with no audited instruction count at
// all — so it stays widest-first rather than becoming arbitrary.
//
// The InsnsPerFMA numbers are the spill audit's own counts for the loop bodies in
// internal/vec: 74 instructions per pass for 16 FMAs, and 50 for 8. They are
// written as that division rather than as 4.625 and 6.25 so the provenance is
// visible at the assignment, and scripts/gate-p3.sh recomputes both from the audit
// on every run and fails on any disagreement.
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
		{Name: AVX512, MR: 4, NR: 32, Unroll: 1, Fn: vec.Kernel4x32, InsnsPerFMA: 50.0 / 8},
		{Name: AVX512, MR: 2, NR: 32, Unroll: 4, Fn: vec.Kernel2x32, InsnsPerFMA: 74.0 / 16},
	}
}

// ReferenceTile is DESIGN.md §4/P2's 12-accumulator tile, which spills. It is
// benchmarked and differentially tested but deliberately absent from Kernels(),
// so that nothing ships it and the gate's zero-spill criterion stays binding on
// the kernels that do. It exists to make the T10 register ceiling a measured
// number rather than a spill count.
//
// It carries no InsnsPerFMA on purpose. The gate audits and cross-checks that
// number only for the shapes that ship, and a recorded measurement nothing checks
// is a measurement that drifts; zero also keeps it unrankable by Preferred, so
// there is no arrangement of classes under which a spilling tile could be chosen.
var ReferenceTile = Kernel{Name: AVX512, MR: 6, NR: 32, Unroll: 4, Fn: vec.Kernel6x32}

// referenceTiles is what the benchmark adds to Kernels(): shapes that are
// interesting to measure but not to ship.
//
// It carries the same HasAVX512 guard as vectorKernels above, and for a harder
// reason than symmetry: Kernel6x32 is AVX-512 unconditionally, so returning it on
// a CPU without AVX-512 does not produce a wrong number, it produces SIGILL. This
// guard was missing for three days (#87). Dispatch was never at risk — Kernels()
// was guarded, and nothing ships this tile — but Measured() is Kernels() plus this,
// and six test and benchmark sites iterate Measured(), so `go test ./...` executed
// EVEX-encoded instructions on any amd64 CPU lacking AVX-512 and died. Every gate
// host has AVX-512, which is exactly why only CI could see it.
//
// The guard belongs here rather than as a t.Skip at those six sites: Measured()'s
// contract is "the kernels this host can run", and a registry that hands out an
// unrunnable kernel is the defect, not each caller's failure to re-check.
func referenceTiles() []Kernel {
	if !vec.HasAVX512() {
		return nil
	}
	return []Kernel{ReferenceTile}
}
