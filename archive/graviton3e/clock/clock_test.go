// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package clock is a core-clock probe for hosts that expose no cpufreq, such as AWS Graviton
// guests (archive/graviton3e/). One iteration is 64 dependent integer ops, alternating ADD and
// EOR so that no algebraic fold can shorten the chain. On a core whose ADD and EOR have 1-cycle
// latency, ops per nanosecond is the clock in GHz. That latency is an assumption, and the probe
// is validated against a cpufreq-readable core before its readings are used.
package clock

import "testing"

var (
	ka, kb uint64 = 0x9E3779B97F4A7C15, 0xC2B2AE3D27D4EB4F
	sink   uint64
)

func chain(n int, x uint64) uint64 {
	a, b := ka, kb
	for i := 0; i < n; i++ {
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
		x += a
		x ^= b
	}
	return x
}

// BenchmarkClock reports dependent ops per nanosecond, which reads as GHz under the 1-cycle
// latency assumption above.
func BenchmarkClock(b *testing.B) {
	const n = 1 << 16
	x := uint64(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = chain(n, x)
	}
	b.StopTimer()
	sink = x
	b.ReportMetric(64*n*float64(b.N)/b.Elapsed().Seconds()/1e9, "Gops/s")
}
