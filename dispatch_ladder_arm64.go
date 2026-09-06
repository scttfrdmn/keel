// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd

package keel

import (
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/l1"
)

// isaLadder is arm64's SIMD capability ordering: NEON then scalar. Every dispatch
// chain on this arch is a subsequence of it (TestP5Dispatch, #153).
func isaLadder() []string { return []string{kern.NEON, l1.Scalar} }

// L1Chain reports the advertised Level-1 dispatch chain: the L1 backends compiled
// into this build. With #154 landed the NEON Level-1 backend joins scalar, so the
// chain is NEON then scalar — unconditionally, like amd64's [avx512, avx2, scalar]
// (advertised is compile-time; runtime feature detection in l1.Backends() picks
// among them). The keel-p5-dispatch marker now reads `l1=neon,scalar
// kern=neon,scalar`: the arm64 port is complete at both levels. Still a subsequence
// of the ladder [neon, scalar], and the partial-port state #136/#153 documented —
// L3 NEON over scalar L1 — no longer holds, so the marker no longer advertises it.
func L1Chain() []string { return []string{l1.NEON, l1.Scalar} }
