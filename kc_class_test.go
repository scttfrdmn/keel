// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/kern"
)

// TestIssueClassKC pins the class-keyed depth block: issueClassKC on an issue-bound host,
// the shipped default everywhere else. Run it under KEEL_KERN_CLASS=issue and =fma to cover
// both branches on one machine.
func TestIssueClassKC(t *testing.T) {
	want := 384
	if activeKernClass == kern.ClassIssue {
		want = issueClassKC
	}
	if block.KC != want {
		t.Fatalf("class %s: block.KC = %d, want %d", activeKernClass, block.KC, want)
	}
}
