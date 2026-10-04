// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package kern

import "github.com/scttfrdmn/keel/internal/vec"

// This file exists because class_nosimd.go's claim was FALSE here (#172).
//
// That file is tagged as the complement of class_amd64.go's, so until now it
// compiled on arm64-with-simd and reported "no vector backend in this build
// (class unused)". Both halves of that are wrong on this ISA: vectorKernels()
// returns two NEON shapes, and Preferred does choose between them — it picks
// 4x16 over 8x8 on their audited counts, which is a measured decision and the
// one 972ee47 had to land. A marker that denies the backend it is describing is
// the `cpu MHz` failure mode §5 rule 5 refuses by name: present, plausible, and
// wrong, so a future check reaching for it gets a confident false answer.
//
// WHY ClassFMA, stated rather than defaulted. arm64 has no measured
// issue/fma classifier in this project: the roofline machinery and its ceiling
// mixes are amd64-derived, and gate-p3 deliberately does not reach
// throughput_verdict on arm64 because PEAK_FLOOR and the issue/fma frontier are
// amd64 bars (#155). So nothing here fingerprints a front end, and DESIGN.md §4
// already names the answer for a host that cannot be identified: FMA-bound,
// "both the common case and the strict direction".
//
// It changes no dispatch, which is the property that makes this a reporting fix
// rather than a behavioural one: on both shipped NEON shapes the two classes
// agree. ClassFMA ranks MemOpsPerFMA first and 4x16's 0.5 beats 8x8's 0.625;
// ClassIssue ranks InsnsPerFMA first and 4x16's 5.000 beats 8x8's 5.750. Same
// winner either way, which kern_arm64.go's registry comment already states and
// TestPreferredPicksTheMeasuredWinnerPerClass pins for both classes.

// HostClass reports ClassFMA on arm64. See the file comment: no front-end
// fingerprint exists for this ISA, and FMA-bound is §4's strict default.
func HostClass() Class { return ClassFMA }

// HostClassEvidence says what was and was not examined, in the form the gate
// prints beside the class. It must not claim a machine was fingerprinted,
// because none was — and it must not claim the backend is absent, because that
// is what #172 was.
func HostClassEvidence() string {
	if !vec.HasNEON() {
		return "arm64 without neon (scalar fallback; class unused)"
	}
	return "neon, no arm64 front-end fingerprint (class defaults to fma, §4's strict direction)"
}
