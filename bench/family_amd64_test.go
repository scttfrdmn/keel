// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package bench

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/vec"
)

// BenchmarkFMAFamily times every member of the AVX-512 ceiling families (#199), witness
// first, in GFLOP/s, so a run can check whether the "avx512" row of BenchmarkPeak and
// BenchmarkFMARoof64, which runs the family's first member, is the family's maximum on
// the host being judged. Named without "Peak", "Kernel" or "FMARoof64" for the reason
// BenchmarkFMARoof64 gives: every gate selects with unanchored -bench regexes.
func BenchmarkFMAFamily(b *testing.B) {
	provenance()
	for _, k := range vec.PeakFamilyAVX512() {
		b.Run("f32/"+k.Name, func(b *testing.B) {
			if got, want := k.Run(1000), k.Witness(1000); got != want {
				b.Fatalf("f32/%s: witness Run(1000) = %v, want %v", k.Name, got, want)
			}
			for i := 0; i < b.N; i++ {
				peakSink = k.Run(peakItersPerOp)
			}
			b.ReportMetric(float64(k.FlopsPerIter)*peakItersPerOp*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
	for _, k := range vec.PeakFamilyAVX51264() {
		b.Run("f64/"+k.Name, func(b *testing.B) {
			if got, want := k.Run(1000), k.Witness(1000); got != want {
				b.Fatalf("f64/%s: witness Run(1000) = %v, want %v", k.Name, got, want)
			}
			for i := 0; i < b.N; i++ {
				peakSink64 = k.Run(peakItersPerOp)
			}
			b.ReportMetric(float64(k.FlopsPerIter)*peakItersPerOp*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
}
