// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/kern"
)

// TestAVX512KC pins the backend-keyed depth block: block.KCDeep is avx512KC whenever the
// dispatched float32 kernel is AVX-512, in either class, and off (0) everywhere else.
// block.KC is never touched. Run it under KEEL_KERN_CLASS=issue and =fma: both must read
// avx512KC on an AVX-512 host.
func TestAVX512KC(t *testing.T) {
	want := 0
	if activeKern.Name == kern.AVX512 {
		want = avx512KC
	}
	if block.KCDeep != want || block.KC != 384 {
		t.Fatalf("class %s, kernel %s: block.KCDeep = %d (want %d), block.KC = %d (want 384)", activeKernClass, activeKern.Name, block.KCDeep, want, block.KC)
	}
}
