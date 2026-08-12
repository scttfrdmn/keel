// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package kern

import "github.com/scttfrdmn/keel/internal/vec"

// HostClass classifies this machine's front end, and HostClassEvidence says on
// what grounds, because the grounds are a proxy and hiding that would be the
// defect.
//
// # Why this is a fingerprint and not an identity
//
// The classification wanted here is per-microarchitecture, and every production
// BLAS reads it from the CPU's identity — vendor plus family/model. On
// GOEXPERIMENT=simd in go1.26.5 that identity is unreachable from pure Go:
// archsimd exposes twenty *feature* predicates and no vendor, family, model or
// brand string, and the data is not merely unexported one layer down, since
// internal/cpu calls cpuid(1, 0) and keeps only ecx, discarding the signature
// word (docs/toolchain-notes.md T14, issue #25).
//
// So the generation is fingerprinted from a feature bundle. AVX512_VBMI2 and
// AVX512_VPOPCNTDQ arrived together with Ice Lake on Intel and Zen 4 on AMD; an
// AVX-512 machine *without* them is the Skylake-X / Cascade Lake / Cooper Lake
// generation, which is the generation P2 measured as issue-bound (KERNEL.md §7:
// janus, an i9-9960X, retires ~4.2 instructions per cycle against two full-width
// FMA units, so its instruction budget per FMA is half a Zen 4's).
//
// # Both ways this can be wrong, and why neither can flatter a gate
//
// The proxy is not exact — it will call an Ice Lake or a Sapphire Rapids part
// FMA-bound on the strength of a feature bit, and P2 has measured neither. What
// makes that acceptable is the direction of each error:
//
//   - Fingerprint says FMA-bound, the host is really issue-bound: dispatch runs
//     4×32, whose 6.25 insns/FMA is outside P2's anti-vacuity shape guard, so the
//     P3 denominator refuses it a roofline and the host faces the unmodified
//     60%-of-OpenBLAS bar. The gate gets *stricter*, and the mistake shows up as a
//     red gate naming the shape.
//   - Fingerprint says issue-bound, the host is really FMA-bound: dispatch runs
//     2×32 and gives up throughput. The gate notices, because it measures both
//     shapes and requires the dispatched one to be no slower than the alternative
//     on that host (scripts/gate-p3.sh).
//
// Neither direction can make a gate lenient, because the gate's own
// classification is measured — convergence of ceiling mixes, not a feature bit —
// and it compares its verdict against the marker this function produces. A
// disagreement is a gate failure, so the proxy is checked on every host on every
// run rather than trusted.
func HostClass() Class {
	if !vec.HasAVX512() {
		// No AVX-512 means no vector shape to choose between: the scalar
		// fallback has one shape per size and the class is unused. FMA-bound is
		// the answer that changes nothing.
		return ClassFMA
	}
	if vec.HasAVX512VBMI2() && vec.HasAVX512VPOPCNTDQ() {
		return ClassFMA
	}
	return ClassIssue
}

// HostClassEvidence describes the bits HostClass read, in the form the gate
// prints beside the class. It names the fingerprint rather than asserting a
// microarchitecture, so a wrong classification can be recognized from the log
// without rerunning anything.
func HostClassEvidence() string {
	switch {
	case !vec.HasAVX512():
		return "no avx512 (scalar fallback; class unused)"
	case vec.HasAVX512VBMI2() && vec.HasAVX512VPOPCNTDQ():
		return "avx512 with vbmi2+vpopcntdq (Ice Lake / Zen 4 or later)"
	default:
		return "avx512 without vbmi2+vpopcntdq (Skylake-X / Cascade Lake / Cooper Lake generation)"
	}
}
