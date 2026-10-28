// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block64

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// The float64 block-size sweep (#191). block64's KC, MC, NC and MB are float32's
// values copied by tools/f64gen, and a float64 element is twice the bytes, so the
// cache arithmetic those values encode does not transfer: a KC×NR B micro-panel and
// an MC×KC A panel hold half as many float64s in the same cache. These benchmarks
// measure where the float64 surface actually peaks, on the shipped float64 kernels.
//
// # The selection rule, pre-registered (committed with this file, before it ran)
//
// The incumbent is float32's (KC, MC, NC) = (384, 144, 4096); at n = 2048 an NC at or
// above n collapses to one point, so the grid tops out at 2048, which IS the
// incumbent. A candidate replaces it only if, for BOTH shipped kernels (KernelD2x16
// and KernelD4x16), its median over 3 interleaved passes beats the incumbent's AND
// its worst pass beats the incumbent's best pass (disjoint ranges). Among the
// candidates that clear that, the one with the highest geometric mean of the two
// kernels' medians wins. If none clears it, nothing changes: an override that is not
// a measured win on both kernels is a preference, not a result.
//
// MB follows the same rule over BenchmarkTrsmMB64: incumbent 64, a candidate must
// beat it on both kernels and BOTH sides (left and right) with disjoint ranges.
// The rule a float32 MB re-sweep was held to (internal/block/tri.go's MB comment).
//
// Measured on one host (janus, Skylake-X), so a chosen value is janus's optimum and
// carries that caveat in tools/f64gen's override comment, as MB's does.

func benchKernels64(b *testing.B) []kern.Kernel64 {
	var out []kern.Kernel64
	for _, k := range kern.Kernels64() {
		if k.Name != kern.Scalar {
			out = append(out, k)
		}
	}
	for _, k := range kern.Measured64() { // block's benchKernels: the #198 candidates
		if k.ALanes > 1 && k.Name != kern.Scalar {
			out = append(out, k)
		}
	}
	if len(out) == 0 {
		b.Skip("no float64 vector microkernel on this build; run on an AVX-512 host")
	}
	return out
}

// grid64 is internal/block's benchGrid: an env override for a sweep axis, refusing
// a bad value rather than defaulting it.
func grid64(b *testing.B, env string, def []int) []int {
	b.Helper()
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return def
	}
	var out []int
	for _, f := range strings.Split(raw, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || v <= 0 {
			b.Fatalf("%s=%q: %q is not a positive integer", env, raw, f)
		}
		out = append(out, v)
	}
	return out
}

func mat64(rows, cols int) []float64 {
	v := make([]float64, rows*cols)
	for i := range v {
		v[i] = float64((i%13)-6) * 0.125
	}
	return v
}

// BenchmarkBlocking64 sweeps KC/MC/NC for a 2048³ float64 Gemm, one thread.
func BenchmarkBlocking64(b *testing.B) {
	const n = 2048
	flops := 2 * float64(n) * float64(n) * float64(n)
	defer func(kc, mc, nc, kp, mp int) { KC, MC, NC, KCPB, MCPB = kc, mc, nc, kp, mp }(KC, MC, NC, KCPB, MCPB)
	kcs := grid64(b, "KEEL_BLOCKING64_KC", []int{96, 128, 192, 256, 384, 512})
	mcs := grid64(b, "KEEL_BLOCKING64_MC", []int{48, 72, 96, 144, 288})
	ncs := grid64(b, "KEEL_BLOCKING64_NC", []int{512, 1024, 2048})
	for _, kn := range benchKernels64(b) {
		am, bm, c := mat64(n, n), mat64(n, n), mat64(n, n)
		for _, kc := range kcs {
			for _, mc := range mcs {
				for _, nc := range ncs {
					b.Run(fmt.Sprintf("%s/kc=%d/mc=%d/nc=%d", kn.ID(), kc, mc, nc), func(b *testing.B) {
						KC, MC, NC, KCPB, MCPB = kc, mc, nc, kc, mc // the grid binds pre-broadcast kernels too
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							Gemm(kn, false, false, n, n, n, 1, am, n, bm, n, 0, c, n)
						}
						b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
					})
				}
			}
		}
	}
}

// BenchmarkTrsmMB64 sweeps MB for a 2048×2048 float64 Trsm, both sides, one thread.
// The diagonal solves use plain-Go axpy/divide closures: float64 Level 1 is #192's,
// so this measures the blocking, which is what MB is.
func BenchmarkTrsmMB64(b *testing.B) {
	const n = 2048
	defer func(mb int) { MB = mb }(MB)
	mbs := grid64(b, "KEEL_TRSM64_MB", []int{32, 48, 64, 96, 128})
	ax := func(alpha float64, x, y []float64) {
		for i, v := range x {
			y[i] += alpha * v
		}
	}
	sd := func(alpha float64, x []float64) {
		for i := range x {
			x[i] /= alpha
		}
	}
	a := mat64(n, n)
	for i := 0; i < n; i++ {
		a[i*n+i] = 4 // well conditioned, so the timed work is the blocking, not denormals
	}
	b0 := mat64(n, n)
	bm := make([]float64, len(b0))
	flops := float64(n) * float64(n) * float64(n) // m·n·(n+1) for one triangle, ≈ n³
	for _, kn := range benchKernels64(b) {
		for _, left := range []bool{true, false} {
			side := "L"
			if !left {
				side = "R"
			}
			for _, mb := range mbs {
				b.Run(fmt.Sprintf("%s/side=%s/mb=%d", kn.ID(), side, mb), func(b *testing.B) {
					MB = mb
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						copy(bm, b0)
						b.StartTimer()
						Trsm(kn, ax, sd, left, true, false, false, n, n, 1, a, n, bm, n)
					}
					b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
				})
			}
		}
	}
}
