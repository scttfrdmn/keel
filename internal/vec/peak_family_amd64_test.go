// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "testing"

// TestPeakFamilyWitness holds every member of both AVX-512 ceiling families (#199) to the exact
// witness, as TestPeakChainsAreIndependent holds the canonical rows. The counts straddle
// peakChunk, where the FO members split the K-steps across tile calls, so an off-by-one
// in the chunking changes the sum and fails here.
func TestPeakFamilyWitness(t *testing.T) {
	counts := []int{0, 1, 2, 3, 17, peakChunk - 1, peakChunk, peakChunk + 1, 1000, 3*peakChunk + 5}
	for _, k := range PeakFamilyAVX512() {
		if want := k.Chains * k.Lanes * 2; k.FlopsPerIter != want {
			t.Errorf("%s: FlopsPerIter %d, want %d", k.Name, k.FlopsPerIter, want)
		}
		for _, n := range counts {
			if got, want := k.Run(n), k.Witness(n); got != want {
				t.Errorf("float32 %s: Run(%d) = %v, want %v", k.Name, n, got, want)
			}
		}
	}
	for _, k := range PeakFamilyAVX51264() {
		if want := k.Chains * k.Lanes * 2; k.FlopsPerIter != want {
			t.Errorf("%s: FlopsPerIter %d, want %d", k.Name, k.FlopsPerIter, want)
		}
		for _, n := range append(counts, 1<<20) {
			if got, want := k.Run(n), k.Witness(n); got != want {
				t.Errorf("float64 %s: Run(%d) = %v, want %v", k.Name, n, got, want)
			}
		}
	}
}

// TestPeakCanonicalIsFamilyHead pins the "avx512" rows to the family's first member, the
// one gate-p2 checks against the rest.
func TestPeakCanonicalIsFamilyHead(t *testing.T) {
	if a, b := PeakKernels()[0], PeakFamilyAVX512()[0]; a.Name != BackendAVX512 || a.Chains != b.Chains || a.FlopsPerIter != b.FlopsPerIter {
		t.Errorf("float32 avx512 row %+v is not family head %s", a, b.Name)
	}
	if a, b := PeakKernels64()[0], PeakFamilyAVX51264()[0]; a.Name != BackendAVX512 || a.Chains != b.Chains || a.FlopsPerIter != b.FlopsPerIter {
		t.Errorf("float64 avx512 row %+v is not family head %s", a, b.Name)
	}
}
