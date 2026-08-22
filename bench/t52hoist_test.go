// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// The diagnostic arm for #115, and DIAGNOSTIC ONLY -- not the shipped
// instrument, not referenced by any gate, and deliberately emitting no
// `keel-bench-ceiling:` line so nothing that parses a ceiling declaration can
// mistake this for one.
//
// It is computeArm with ONE difference: the goroutines are forked once for the
// whole b.N instead of once per iteration, so there are 1 join and b.N-1 fewer
// fork/join round trips for identical total work and an identical flops
// formula. computeArm's per-iteration `wg.Wait()` is a max over eight arrival
// times; this arm has no per-iteration barrier at all.
//
// It exists to settle T-52's refutation branch by experiment. The residency
// probe showed all eight workers on eight distinct masked cpus (so placement is
// not the cause) with duty cycle tracking flop efficiency sample for sample (so
// the loss is parked threads, not throttled ones). That bounded the residual to
// the fork/join term without proving it, and hoisting proves it: the
// spread-vs-confined gap goes from 0.143 to -0.019 on keel-zen4 and from 0.442
// to 0.007 on keel-zen5, duty rises to 0.97-0.99 on all six arms, and every arm
// lands within 1.1% of eight times its own single-thread rate.
//
// It also carries its own control. At -benchtime=1x this arm and computeArm are
// the SAME program -- one fork, one join, one iteration -- so their per-op series
// must agree, and they do; the variant diverges only where the fork count does.
// Keep that property if this file is ever edited: without it, "the hoisted arm is
// faster" is equally consistent with the variant measuring something else.
//
// It is not the ceiling and must not become one by default. Whether the shipped
// arm hoists its fork changes gate-p5's headline denominator and every archived
// share, which is a criterion change and Scott's to make (#115).

package bench

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/scttfrdmn/keel/internal/vec"
)

func BenchmarkT52Hoist(b *testing.B) {
	b.Run("compute", func(b *testing.B) {
		for _, k := range vec.PeakKernels() {
			b.Run(k.Name, func(b *testing.B) {
				for _, procs := range streamThreads {
					b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) {
						hoistArm(b, procs, k)
					})
				}
			})
		}
	})
}

func hoistArm(b *testing.B, procs int, k vec.PeakKernel) {
	prev := runtime.GOMAXPROCS(procs)
	defer runtime.GOMAXPROCS(prev)
	if got, want := k.Run(1000), k.Witness(1000); got != want {
		b.Fatalf("%s: witness Run(1000) = %v, want %v", k.Name, got, want)
	}
	sinks := make([]float32, procs*16)
	n := b.N
	b.ResetTimer()
	var wg sync.WaitGroup
	for t := 0; t < procs; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				sinks[t*16] = k.Run(peakItersPerOp)
			}
		}(t)
	}
	wg.Wait()
	flops := float64(k.FlopsPerIter) * float64(peakItersPerOp) * float64(procs) * float64(b.N)
	b.ReportMetric(flops/b.Elapsed().Seconds()/1e9, "GFLOP/s")
}
