// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/scttfrdmn/keel"
	"github.com/scttfrdmn/keel/internal/vec"
)

// The float64 measurement instruments for #193: what the float64 performance gates
// will divide by and judge, measured before any float64 target exists (DESIGN.md
// §1, 2026-10-25).
//
// THE NAMES ARE CHOSEN SO NO FLOAT32 GATE SELECTS THEM. Every gate filters with
// unanchored -bench regexes ('Scale|Peak|Ceiling|L1', 'Peak|Kernel/…', 'FMARoof64|…',
// '(Peak|Sgemm|…)/(avx512|…)'), so a float64 row named Scale64 or CeilingF64 would run
// inside every float32 gate and lengthen it. "WideRoof64" and "Level3F64" match none.
// The float64 gates (#193, KEEL_ELEM=f64) select them by name.

// BenchmarkWideRoof64 is BenchmarkCeiling's compute and family arms at float64: every
// float64 FMA ceiling and family member at 1 and 8 threads, so a float64 share can be
// taken of the host's own measured 8-thread float64 ceiling, as float32's is.
//
//	WideRoof64/<backend>/threads=N          the float64 ceilings (PeakKernels64)
//	WideRoof64/family/<member>/threads=N    the float64 family (PeakFamily64)
func BenchmarkWideRoof64(b *testing.B) {
	provenance()
	for _, k := range vec.PeakKernels64() {
		b.Run(k.Name, func(b *testing.B) {
			for _, procs := range streamThreads {
				b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) { computeArm64(b, procs, k) })
			}
		})
	}
	if fam := vec.PeakFamily64(); len(fam) > 0 {
		b.Run("family", func(b *testing.B) {
			for _, k := range fam {
				b.Run(k.Name, func(b *testing.B) {
					for _, procs := range streamThreads {
						b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) { computeArm64(b, procs, k) })
					}
				})
			}
		})
	}
}

// computeArm64 is computeArm for a PeakKernel64, line for line. It is a copy, not a
// generic shared with computeArm, on purpose: a generic would change the float32
// ceiling's compiled instrument, and a changed instrument is a new measurement era
// (DESIGN.md §5 rule 17), not a refactor.
func computeArm64(b *testing.B, procs int, k vec.PeakKernel64) {
	prev := runtime.GOMAXPROCS(procs)
	defer runtime.GOMAXPROCS(prev)
	if got, want := k.Run(1000), k.Witness(1000); got != want {
		b.Fatalf("%s: witness Run(1000) = %v, want %v — accumulator chains did not "+
			"survive compilation, so this is not a ceiling", k.Name, got, want)
	}
	sinks := make([]float64, procs*8) // a cache line apart, as computeArm's
	n := b.N
	var wg sync.WaitGroup
	warm := make(chan struct{}, procs)
	start := make(chan struct{})
	for t := 0; t < procs; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			sinks[t*8] = k.Run(peakItersPerOp)
			warm <- struct{}{}
			<-start
			for i := 0; i < n; i++ {
				sinks[t*8] = k.Run(peakItersPerOp)
			}
		}(t)
	}
	for t := 0; t < procs; t++ {
		<-warm
	}
	b.ResetTimer()
	close(start)
	wg.Wait()
	b.StopTimer()
	flops := float64(k.FlopsPerIter) * float64(peakItersPerOp) * float64(procs) * float64(n)
	b.ReportMetric(flops/b.Elapsed().Seconds()/1e9, "GFLOP/s")
	declareCeiling(b, procs, fmt.Sprintf("kind=compute64 instrument=v2 width=%s flops=%s iters=%d",
		k.Name, strconv.FormatFloat(flops/float64(n), 'f', 0, 64), peakItersPerOp))
}

func makeMat64(rows, cols int) []float64 {
	v := make([]float64, rows*cols)
	for i := range v {
		v[i] = float64((i%13)-6) * 0.125
	}
	return v
}

// makeTri64 is makeTri at float64, for the same reasons.
func makeTri64(n int) []float64 {
	v := make([]float64, n*n)
	scale := 1 / float64(n)
	for i := 0; i < n; i++ {
		row := v[i*n : (i+1)*n]
		for j := range row {
			if i == j {
				row[j] = 1
				continue
			}
			row[j] = float64((i+j)%13-6) * scale
		}
	}
	return v
}

// BenchmarkLevel3F64 is BenchmarkScale at float64: the four judged routines at n = 4096,
// 1 and 8 threads, with the same work accounting and thread declaration, so a float64
// share and scaling ratio are computed exactly as float32's are.
//
//	Level3F64/<Dgemm|Dsyrk|Dsymm|Dtrsm>/n=4096/threads=N
func BenchmarkLevel3F64(b *testing.B) {
	provenance()
	cases := []scaleCase{{
		name: "Dgemm", work: gemmWork(scaleN),
		prep: func(*testing.B) func() {
			a, bm, c := makeMat64(scaleN, scaleN), makeMat64(scaleN, scaleN), makeMat64(scaleN, scaleN)
			return func() {
				keel.Dgemm(keel.NoTrans, keel.NoTrans, scaleN, scaleN, scaleN, 1, a, scaleN, bm, scaleN, 0, c, scaleN)
			}
		},
	}, {
		name: "Dsyrk", work: syrkWork(scaleN),
		prep: func(*testing.B) func() {
			a, c := makeMat64(scaleN, scaleN), makeMat64(scaleN, scaleN)
			return func() { keel.Dsyrk(keel.Lower, keel.NoTrans, scaleN, scaleN, 1, a, scaleN, 0, c, scaleN) }
		},
	}, {
		name: "Dsymm", work: symmWork(scaleN),
		prep: func(*testing.B) func() {
			a, bm, c := makeMat64(scaleN, scaleN), makeMat64(scaleN, scaleN), makeMat64(scaleN, scaleN)
			return func() {
				keel.Dsymm(keel.Left, keel.Lower, scaleN, scaleN, 1, a, scaleN, bm, scaleN, 0, c, scaleN)
			}
		},
	}, {
		name: "Dtrsm", work: trsmWork(scaleN, scaleN),
		prep: func(b *testing.B) func() {
			a, b0 := makeTri64(scaleN), makeMat64(scaleN, scaleN)
			bm := make([]float64, len(b0))
			return func() {
				b.StopTimer()
				copy(bm, b0) // in-place solve; restored off the clock, as Strsm's row is
				b.StartTimer()
				keel.Dtrsm(keel.Left, keel.Lower, keel.NoTrans, keel.NonUnit, scaleN, scaleN, 1, a, scaleN, bm, scaleN)
			}
		},
	}}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.Run(fmt.Sprint("n=", scaleN), func(b *testing.B) {
				for _, procs := range scaleThreads {
					b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) { scaleRow(b, procs, c) })
				}
			})
		})
	}
}

// BenchmarkDgemm and BenchmarkDsyrk are BenchmarkSgemm and BenchmarkSsyrk at float64,
// for gate-p3/p4's KEEL_ELEM=f64 arms: the same sizes, operation and data shape, so
// the float64 percent-of-peak and Dsyrk/Dgemm criteria divide like their float32
// twins. Rows "Dgemm/n=N" and "Dsyrk/n=N" match no float32 filter ('Sgemm', 'Ssyrk').
func BenchmarkDgemm(b *testing.B) {
	provenance()
	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			a, bm, c := makeMat64(n, n), makeMat64(n, n), makeMat64(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Dgemm(keel.NoTrans, keel.NoTrans, n, n, n, 1, a, n, bm, n, 0, c, n)
			}
			rateWork(b, gemmWork(n))
		})
	}
}

func BenchmarkDsyrk(b *testing.B) {
	provenance()
	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			a, c := makeMat64(n, n), makeMat64(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Dsyrk(keel.Lower, keel.NoTrans, n, n, 1, a, n, 0, c, n)
			}
			rateWork(b, syrkWork(n))
		})
	}
}

var sink64 float64

func makeVec64(n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = float64(i%17) - 8
	}
	return v
}

// BenchmarkLevel1F64 is the BenchmarkL1S* set at float64: the same sizes, flops per
// element and access pattern, with bytes counted at 8 per element. Rows are
// "Level1F64/<Ddot|Daxpy|Dscal|Dasum|Dnrm2|Idamax>/n=N"; "Level1F64" does not contain
// "L1", so no float32 gate selects them.
func BenchmarkLevel1F64(b *testing.B) {
	provenance()
	type op struct {
		name         string
		bytesPerElem int
		flopsPerElem float64 // 0: no flop rate (Idamax compares, it does no arithmetic)
		run          func(n int, x, y []float64)
	}
	ops := []op{
		{"Ddot", 16, 2, func(n int, x, y []float64) { sink64 = keel.Ddot(n, x, 1, y, 1) }},
		{"Daxpy", 24, 2, func(n int, x, y []float64) { keel.Daxpy(n, 1.0000001, x, 1, y, 1) }},
		{"Dscal", 16, 1, func(n int, x, _ []float64) { keel.Dscal(n, 1.0000001, x, 1) }},
		{"Dasum", 8, 1, func(n int, x, _ []float64) { sink64 = keel.Dasum(n, x, 1) }},
		{"Dnrm2", 8, 2, func(n int, x, _ []float64) { sink64 = keel.Dnrm2(n, x, 1) }},
		{"Idamax", 8, 0, func(n int, x, _ []float64) { sinkI = keel.Idamax(n, x, 1) }},
	}
	for _, o := range ops {
		b.Run(o.name, func(b *testing.B) {
			for _, n := range sizes {
				x, y := makeVec64(n), makeVec64(n)
				b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
					b.SetBytes(int64(o.bytesPerElem * n))
					for i := 0; i < b.N; i++ {
						o.run(n, x, y)
					}
					if o.flopsPerElem > 0 {
						rate(b, n, o.flopsPerElem)
					}
				})
			}
		})
	}
}
