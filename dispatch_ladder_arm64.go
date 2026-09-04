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
// into this build. On arm64 that is scalar ONLY — there is no NEON L1 backend yet
// (#136 shipped the NEON Level-3 microkernels; NEON Level 1 is a separate v0.2.0
// unit, #154). So keel's arm64 dispatch is a PARTIAL PORT — Level 3 NEON over a
// scalar Level 1 — and this chain states it: the keel-p5-dispatch marker reads
// `l1=scalar kern=neon,scalar`, so the mixed state can never read as complete
// (#153). It is a subsequence of the ladder (scalar is the ladder's lower rung),
// and L3 being ahead of L1 is a legitimate partial-port state, ruled 2026-09-04.
func L1Chain() []string { return []string{l1.Scalar} }
