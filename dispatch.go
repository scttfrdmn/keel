// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

// Backend selection, finalized in P5. Order: avx512 → avx2 → scalar.
// Override with KEEL_FORCE=scalar|avx2|avx512 for testing.

type backend uint8

const (
	backendScalar backend = iota
	backendAVX2
	backendAVX512
)

// active is chosen at init by the build-tagged detect files in internal/vec
// once P0 lands; the skeleton pins scalar so the package is honest about
// what it can do.
var active = backendScalar
