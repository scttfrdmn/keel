// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd

package keel

import (
	"github.com/scttfrdmn/keel/internal/l1"
)

// isaLadder is amd64's SIMD capability ordering, widest first, scalar last: the
// canonical order every dispatch chain on this arch is a subsequence of
// (TestP5Dispatch, #153). L1 has a backend for every rung, so L1Chain is the whole
// ladder; Level 3 drops avx2 (no AVX2 microkernel, #40).
func isaLadder() []string { return []string{l1.AVX512, l1.AVX2, l1.Scalar} }

// L1Chain reports the advertised Level-1 dispatch chain: the L1 backends compiled
// into this build, in ladder order, on a machine that has all of them. On amd64 all
// three exist.
func L1Chain() []string { return []string{l1.AVX512, l1.AVX2, l1.Scalar} }
