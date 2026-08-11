// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// BenchmarkKernel measures each microkernel shape on packed panels, with no
// blocking around it: the numerator of P2's percent-of-peak.
//
// # What is and is not being measured
//
// The panels are packed once, outside the timer, and stay in L1 for the whole
// run. That is deliberate and it is what P2's criterion means: DESIGN.md §4/P2
// asks what fraction of the FMA ceiling the *K-loop* reaches, with packing and
// blocking excluded because those are P3's subject. A number measured on cold
// panels would be a memory-bandwidth measurement with a kernel attached.
//
// So this is an upper bound on what P3 can deliver, and it is labelled as one.
// The same panels are reused across iterations, which also means C accumulates
// without bound across b.N; that is harmless for timing (float32 tops out around
// 3e38 and a run reaches order b.N) and it avoids clearing C inside the timer,
// which would add a store per element that the real kernel does not do.
//
// # Why kc varies
//
// kc is the loop trip count, so it sets what fraction of the call is steady-state
// body versus prologue and write-out. At kc=128 the 2x32 kernel runs 32 unrolled
// passes of 74 instructions against a prologue and write-out of a few dozen —
// under 2% overhead. kc=8 is included precisely because it is *not* negligible
// there: P3 will choose KC, and a shape that only wins at large kc is a different
// recommendation than one that wins everywhere.
//
// # Why shapes are compared in GFLOP/s and not ns/op
//
// The shapes do different amounts of work per call (a 4x32 tile is twice a 2x32
// tile), so ns/op cannot compare them. GFLOP/s normalizes by the arithmetic,
// which is the quantity the 55% floor is stated in.
func BenchmarkKernel(b *testing.B) {
	provenance()
	for _, k := range kern.Measured() {
		for _, kc := range []int{8, 32, 128, 512} {
			b.Run(fmt.Sprintf("%s/kc=%d", k.ID(), kc), func(b *testing.B) {
				a, bp, c, ldc := kernelPanels(k, kc)
				flopsPerCall := 2.0 * float64(k.MR) * float64(k.NR) * float64(kc)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k.Fn(kc, a, bp, c, ldc)
				}
				b.StopTimer()

				b.ReportMetric(flopsPerCall*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
				// Instructions issued per FMA is a compile-time property
				// (docs/toolchain-notes.md T10) and the reason the shapes differ;
				// reporting flops per call here lets a reader reconstruct the
				// arithmetic from the benchmark output alone rather than trusting
				// the name to encode the shape.
				b.ReportMetric(flopsPerCall, "flops/call")
				kernelSink = c[0]
			})
		}
	}
}

var kernelSink float32

// kernelPanels builds the packed panels and C tile one kernel call needs.
//
// ldc is the tile width plus a pad rather than the tile width itself. A
// contiguous C would put the whole tile on as few cache lines as possible, which
// is the best case P3 will never see: in a real GEMM, C's rows are ldc apart
// because C is the user's matrix. The pad is small (one cache line) so this stays
// an L1-resident measurement.
func kernelPanels(k kern.Kernel, kc int) (a, b, c []float32, ldc int) {
	rng := rand.New(rand.NewSource(1))
	ldc = k.NR + 16
	a = make([]float32, kc*k.MR)
	b = make([]float32, kc*k.NR)
	c = make([]float32, k.MR*ldc)
	for i := range a {
		a[i] = rng.Float32()*2 - 1
	}
	for i := range b {
		b[i] = rng.Float32()*2 - 1
	}
	return a, b, c, ldc
}
