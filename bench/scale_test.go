// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel"
)

// The rows scripts/gate-p5.sh's headline criterion divides (DESIGN.md §4/P5):
// each Level-3 routine at 4096³, once on one thread and once on eight, so the
// ratio is between two measurements of one operation on one machine in one
// process. Anything else — two hosts, two binaries, two sizes — would make the
// number a comparison of something other than parallel speedup.
//
// # The thread count is a property of the row, not of the environment
//
// GOMAXPROCS is set inside each sub-benchmark and restored after, rather than
// exported by the harness that launches the run. That is deliberate and the gate
// depends on it: gate-p5.sh runs ONE invocation per host with both thread counts
// inside it, because a run per thread count would put the two arms of the ratio
// in two processes with two page-cache states and two frequency histories. A
// single KEEL_REMOTE_ENV="GOMAXPROCS=8" would also cap the threads=1 row's twin
// at 8 and the threads=8 row at whatever that line happened to say.
//
// So each row declares what it ran on and the gate checks the declaration against
// the row's own name — see declareThreads. The declaration is the library's
// answer (keel.GOMAXPROCS, keel.WorkersLastCall), not the harness's belief about
// what it set, because those are the two things that can disagree.
//
// # Why the harness reports a wrong worker count instead of failing on it
//
// If the eight-thread row somehow ran on five workers, this file prints workers=5
// and the gate fails the criterion with that number in the message. Failing the
// benchmark here instead would surface as "the scaling benchmark run failed",
// which sends whoever reads it looking for a crash. A measurement that came out
// wrong and a harness that broke are two different states, and only one of them
// is about the nest.

// scaleN is gate-p5.sh's P5_SIZE.
//
// 4096 and not the 2048 of P3's criterion, because a parallel speedup measured
// where the serial nest still fits comfortably in cache is a measurement of the
// cache and not of the pool: at 2048 the three matrices are 48 MB, eight workers
// contend for one L3, and the ratio that falls out says more about the sharing
// than about the partition. At 4096 they are 201 MB, which is where a caller who
// cares about eight cores actually is.
const scaleN = 4096

// scaleThreads is the pair the ratio is taken over: 1 is the denominator every
// published keel number so far was measured at, and 8 is gate-p5.sh's P5_THREADS.
var scaleThreads = []int{1, 8}

// symmWork is Ssymm's numerator: GEMM's. A is symmetric and k×k, but every entry
// of C still receives a full k-deep dot product — symm's saving is in the memory
// it touches, never in the arithmetic it does — so counting anything less than
// 2·m·n·k here would report a rate above the one the hardware delivered.
func symmWork(n int) work {
	return work{
		m: n, n: n, k: n,
		formula: "2*m*n*k",
		flops:   2 * float64(n) * float64(n) * float64(n),
	}
}

// trsmWork is Strsm's, for the left-side solve: one multiply-add per (row, column)
// pair of one m×m triangle including its diagonal, per right-hand side, i.e.
// 2·n·m(m+1)/2 = n·m·(m+1).
//
// k is declared equal to m because a triangular solve has no independent third
// dimension — the depth of its rank updates is the already-solved part of the
// triangle, which sums to m over the block loop. The gate's count for Strsm uses
// m and n only; k is printed for the shape, not for the arithmetic.
func trsmWork(m, n int) work {
	return work{
		m: m, n: n, k: m,
		formula: "n*m*(m+1)",
		flops:   float64(n) * float64(m) * float64(m+1),
	}
}

// scaleCase is one routine's row pair: a numerator, and a prep that allocates its
// operands and returns the call to time.
//
// prep runs per trial and before the timer starts, so every trial of a row starts
// from the same operands — which matters for Strsm, whose body overwrites them.
type scaleCase struct {
	name string
	work work
	prep func(b *testing.B) (body func())
}

func scaleCases() []scaleCase {
	return []scaleCase{{
		name: "Sgemm",
		work: gemmWork(scaleN),
		prep: func(*testing.B) func() {
			a, bm, c := makeMat(scaleN, scaleN), makeMat(scaleN, scaleN), makeMat(scaleN, scaleN)
			return func() {
				keel.Sgemm(keel.NoTrans, keel.NoTrans, scaleN, scaleN, scaleN,
					1, a, scaleN, bm, scaleN, 0, c, scaleN)
			}
		},
	}, {
		name: "Ssyrk",
		work: syrkWork(scaleN),
		prep: func(*testing.B) func() {
			a, c := makeMat(scaleN, scaleN), makeMat(scaleN, scaleN)
			return func() {
				keel.Ssyrk(keel.Lower, keel.NoTrans, scaleN, scaleN, 1, a, scaleN, 0, c, scaleN)
			}
		},
	}, {
		// Left side, so A is m×m and the shape is the square m = n = k = 4096 the
		// gate recomputes 2·m·n·k from. Until issue #36 this row also paid for a
		// reflection of A into a dense square on every call — 67 MB of scratch at this
		// size, and a whole d² pass over A before the pack made its own. It was not a
		// serial region: expandSym ran under par.Run over rows. internal/pack now reads
		// the stored triangle in place; BenchmarkSymmNarrow is the fixture at the shape
		// where that shows.
		name: "Ssymm",
		work: symmWork(scaleN),
		prep: func(*testing.B) func() {
			a, bm, c := makeMat(scaleN, scaleN), makeMat(scaleN, scaleN), makeMat(scaleN, scaleN)
			return func() {
				keel.Ssymm(keel.Left, keel.Lower, scaleN, scaleN,
					1, a, scaleN, bm, scaleN, 0, c, scaleN)
			}
		},
	}, {
		// Left side, lower, non-unit: the side and the count the gate's n·m·(m+1)
		// describes.
		//
		// B is restored between iterations with the timer stopped, because Strsm
		// solves in place: iteration two would otherwise be solving against
		// iteration one's output. That is not merely untidy — repeated application
		// of A⁻¹ walks the magnitudes toward the dominant eigendirection, and a
		// benchmark that drifted into denormals would report the slowdown as a
		// property of the routine.
		name: "Strsm",
		work: trsmWork(scaleN, scaleN),
		prep: func(b *testing.B) func() {
			a, b0 := makeTri(scaleN), makeMat(scaleN, scaleN)
			bm := make([]float32, len(b0))
			return func() {
				b.StopTimer()
				copy(bm, b0)
				b.StartTimer()
				keel.Strsm(keel.Left, keel.Lower, keel.NoTrans, keel.NonUnit,
					scaleN, scaleN, 1, a, scaleN, bm, scaleN)
			}
		},
	}}
}

// BenchmarkScale is the headline criterion's input: four routines × two thread
// counts, at one size, in one process.
//
// The name has four elements so the gate's scale_name() can address a row exactly
// — "Scale/Sgemm/n=4096/threads=8". n= sits between the routine and the thread
// count rather than being folded into either, so that adding a second size later
// does not renumber anything the gate reads.
func BenchmarkScale(b *testing.B) {
	provenance()
	for _, c := range scaleCases() {
		b.Run(c.name, func(b *testing.B) {
			b.Run(fmt.Sprint("n=", scaleN), func(b *testing.B) {
				for _, procs := range scaleThreads {
					b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) {
						scaleRow(b, procs, c)
					})
				}
			})
		})
	}
}

// scaleRow runs one (routine, thread count) row: it sets GOMAXPROCS for the row,
// allocates outside the timed region, times the call, then declares both the
// numerator and the thread count it used.
func scaleRow(b *testing.B, procs int, c scaleCase) {
	prev := runtime.GOMAXPROCS(procs)
	defer runtime.GOMAXPROCS(prev)
	body := c.prep(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body()
	}
	declareThreads(b, benchRowName(b))
	rateWork(b, c.work)
}

// threadsDeclared keeps the marker to one line per row, for the same reason
// flopsDeclared does: the body runs once per b.N trial and once more per -count.
var threadsDeclared = map[string]bool{}

// declareThreads prints what the row actually ran on. Called after the timed loop,
// so the print cannot land inside the measurement, and after the routine has run
// at least once, so WorkersLastCall has something to report.
func declareThreads(b *testing.B, name string) {
	if threadsDeclared[name] {
		return
	}
	threadsDeclared[name] = true
	fmt.Printf("keel-bench-threads: name=%s gomaxprocs=%d workers=%d\n",
		name, keel.GOMAXPROCS(), keel.WorkersLastCall())
}

// benchRowName is the sub-benchmark path as `go test` reports it, minus the
// Benchmark prefix — "Scale/Sgemm/n=4096/threads=8". b.Name() carries no
// -GOMAXPROCS suffix; that is appended to the reported line, and scripts/bench.sh
// strips it there.
func benchRowName(b *testing.B) string {
	return strings.TrimPrefix(b.Name(), "Benchmark")
}

// makeTri fills an n×n row-major lower-triangular matrix with a unit diagonal and
// small off-diagonals.
//
// The diagonal is exactly 1 and the off-diagonals are scaled by 1/n for the reason
// tri_test.go's triMatrix scales its own: an unscaled patterned triangle at
// n = 4096 has a growth factor large enough to send the solve's intermediates to
// infinity part way down a column, and an Inf or a NaN in the data is a timing
// artifact on hardware that handles them slowly. The upper half is filled too
// rather than left at zero — Strsm does not read it, and a matrix that is only
// half-touched pages differently from one that is not.
func makeTri(n int) []float32 {
	v := make([]float32, n*n)
	scale := float32(1) / float32(n)
	for i := 0; i < n; i++ {
		row := v[i*n : (i+1)*n]
		for j := range row {
			if i == j {
				row[j] = 1
				continue
			}
			row[j] = float32((i+j)%13-6) * scale
		}
	}
	return v
}
