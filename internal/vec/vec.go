// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package vec is THE SHIM: the only package in keel allowed to import
// simd/archsimd. Every vector op exposed here has a scalar twin in
// vec_scalar.go, and differential tests bind all backends to it.
//
// P0 standing order (DESIGN.md §4/§7): before writing or editing any
// wrapper, run `go doc simd/archsimd` and `go doc simd` and read the
// sources under $(go env GOROOT)/src/simd/. The API is experimental and
// has had breaking renames between releases; identifiers recalled from
// training are presumptively wrong. Copy names from go doc output.
//
// # Two layers, on purpose
//
// The shim has a hot layer and a test layer, and they exist for different
// reasons.
//
// The hot layer is the native-typed wrappers the microkernels call:
// Load512/FMA512/Store512 and the 256-bit equivalents. Their arguments and
// results are the architecture's own vector types, so values stay in
// registers across a chain of them and each wrapper inlines to a single
// instruction. This is the layer P2's kernel is built from, and the layer
// gate-p0.sh greps for FMA fusion.
//
// The test layer is the Block-typed ops (ScalarMulAdd, AVX512MulAdd, ...).
// Block is a plain [Lanes]float32, so every backend's op has the *same Go
// signature* regardless of the vector type underneath, which is what lets
// one differential test table drive all three backends in one binary. These
// round-trip through memory and are not for hot loops.
//
// # Widths
//
// Lanes is 16 — the AVX-512 float32 width, and the width of the semantic
// spec. The AVX2 backend implements the same 16-lane Block ops as two
// 8-lane halves, so a Block op means the same thing on every backend and the
// differential test needs no per-backend width bookkeeping.
package vec

// Lanes is the width of a Block: the number of float32 elements the shim's
// semantic spec operates on at once. It is the AVX-512 float32 vector width.
const Lanes = 16

// Block is the shim's differential-testing currency: one Lanes-wide vector
// of float32 as a plain Go array. Every backend exposes the same set of
// Block->Block ops (see the package doc), which is what binds the vector
// backends to the scalar spec in one test binary.
type Block [Lanes]float32

// Backend names, as they appear in test output and in KEEL_FORCE.
const (
	BackendScalar = "scalar"
	BackendAVX2   = "avx2"
	BackendAVX512 = "avx512"
	BackendNEON   = "neon"
)

// Available returns the backends that are both compiled into this binary and
// runnable on this CPU, widest first. The scalar backend is always present
// and always last, so the result is never empty.
//
// Dispatch (P5) takes the head of this list; gate-p0.sh uses it, via the
// differential suite's coverage marker, to tell "all three backends passed"
// apart from "two of them were never built".
func Available() []string {
	var b []string
	if HasAVX512() {
		b = append(b, BackendAVX512)
	}
	if HasAVX2() {
		b = append(b, BackendAVX2)
	}
	return append(b, BackendScalar)
}
