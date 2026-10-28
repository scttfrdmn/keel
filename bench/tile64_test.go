// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// BenchmarkTile64 is BenchmarkKernel for the float64 microkernels (#191): every
// shape kern.Measured64 returns, at the same kc values, in GFLOP/s and flops/call.
// It is the numerator the float64 P2 gate (KEEL_ELEM=f64 scripts/gate-p2.sh)
// divides by BenchmarkFMARoof64.
//
// Named Tile64 rather than Kernel64 for the reason BenchmarkFMARoof64 avoids
// "Peak": gate-p2's float32 filter is the unanchored 'Peak|Kernel/...', and a name
// containing "Kernel" would run the float64 tiles inside every float32 gate.
func BenchmarkTile64(b *testing.B) {
	provenance()
	for _, k := range kern.Measured64() {
		for _, kc := range []int{8, 32, 128, 512} {
			b.Run(fmt.Sprintf("%s/kc=%d", k.ID(), kc), func(b *testing.B) {
				a, bp, c, ldc := tilePanels64(k, kc)
				flopsPerCall := 2.0 * float64(k.MR) * float64(k.NR) * float64(kc)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k.Fn(kc, a, bp, c, ldc)
				}
				b.StopTimer()

				b.ReportMetric(flopsPerCall*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
				b.ReportMetric(flopsPerCall, "flops/call")
				tileSink64 = c[0]
			})
		}
	}
}

var tileSink64 float64

// tilePanels64 is kernelPanels at float64. ldc pads the tile by one 64-byte cache
// line, 8 float64s, as kernelPanels pads by 16 float32s: C's rows are ldc apart in
// a real GEMM, and the pad keeps that true while the measurement stays L1-resident.
func tilePanels64(k kern.Kernel64, kc int) (a, b, c []float64, ldc int) {
	rng := rand.New(rand.NewSource(1))
	ldc = k.NR + 8
	a = make([]float64, kc*k.MR)
	b = make([]float64, kc*k.NR)
	c = make([]float64, k.MR*ldc)
	for i := range a {
		a[i] = rng.Float64()*2 - 1
	}
	for i := range b {
		b[i] = rng.Float64()*2 - 1
	}
	return k.ExpandA(a), b, c, ldc // a pre-broadcast kernel times the panel it reads
}
