// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !goexperiment.simd || !amd64

package vec

// Feature-detection stubs for builds where no vector backend was compiled in
// at all: a stock toolchain without GOEXPERIMENT=simd, or any GOARCH other
// than amd64 (simd/archsimd is amd64-only in go1.26.5 — its vector types live
// in *_amd64.go files, so on other architectures the package exports only the
// X86 feature struct, whose methods all return false).
//
// The build tag is the exact complement of the vector backends', so exactly
// one of the two definitions exists in any build. Keeping the stubs in their
// own file rather than guarding each call site is what lets dispatch.go and
// the tests call HasAVX512/HasAVX2 unconditionally.

// HasAVX512 reports false: no AVX-512 backend was compiled into this binary.
func HasAVX512() bool { return false }

// HasAVX2 reports false: no AVX2 backend was compiled into this binary.
func HasAVX2() bool { return false }
