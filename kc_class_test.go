// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/kern"
)

// TestIssueClassKC pins the class-keyed depth block: block.KCDeep is issueClassKC on an
// issue-bound host and off (0) everywhere else, and block.KC is never touched. Run it under KEEL_KERN_CLASS=issue and =fma to cover
// both branches on one machine.
func TestIssueClassKC(t *testing.T) {
	want := 0
	if activeKern.Name == kern.AVX512 {
		want = issueClassKC
	}
	if block.KCDeep != want || block.KC != 384 {
		t.Fatalf("class %s: block.KCDeep = %d (want %d), block.KC = %d (want 384)", activeKernClass, block.KCDeep, want, block.KC)
	}
}
