// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !goexperiment.simd || (!amd64 && !arm64)

package keel

import "github.com/scttfrdmn/keel/internal/l1"

// isaLadder and L1Chain for every build with no vector backend: the scalar path on
// a stock toolchain (no experiment) and any GOARCH that is neither amd64 nor arm64.
// The ladder is scalar alone, so it is the exact complement of the two vector
// ladders' build tags — one definition per build, always (TestP5Dispatch, #153).
func isaLadder() []string { return []string{l1.Scalar} }

func L1Chain() []string { return []string{l1.Scalar} }
