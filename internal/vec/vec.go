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
package vec
