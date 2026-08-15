// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/keel/internal/block"
)

// Parallelism as a correctness property (DESIGN.md §4/P5, scripts/gate-p5.sh
// criteria 5, 6 and 7), and the markers that gate parses out of this log.
//
// # Why the markers are printed after the assertions and not beside them
//
// A marker is the test telling the gate what it proved. Printed unconditionally it
// would be the test telling the gate what it attempted, which is the shape of
// every claim this project has spent five phases refusing. So each marker below is
// emitted at the END of a check that has already failed the test if it did not
// hold, and the thread counts and property names in it are the ones the loop
// actually ran rather than a list declared next to one.
//
// # Why the thread counts are 1, 3 and 8
//
// gate-p5.sh criterion 5 requires that set and says why: a row-partition
// off-by-one hides perfectly at 1, 2, 4 and 8, because every one of those divides
// both the block count and the core count of the machines here. 3 divides neither.
// 1 is there because it is the denominator of the headline ratio and the path
// every published keel number was measured on, so "the parallel nest agrees with
// the serial one" is only meaningful if the serial one is one of the arms.
//
// # Why bitwise and not a tolerance
//
// Partitioning the ic loop splits C by row panels. It does not reassociate any
// single output element's sum: the depth loop stays serial, and the microkernel is
// untouched. So the parallel result is not merely close to the serial one, it is
// the same bits, and comparing it any other way would leave room for a real
// reassociation bug to hide inside a tolerance that was sized for rounding. If
// some future change parallelizes the K loop, this test breaks loudly and the
// correct response is a ruling on non-determinism, not a wider epsilon.

// p5Threads is the set gate-p5.sh's P5_DET_THREADS names. Kept in this order
// because the marker prints it and the gate reads it as a set.
var p5Threads = []int{1, 3, 8}

// p5 shapes. m is deliberately larger than 8·MC so that the ic loop has at least
// as many blocks as the widest thread count has workers — at m below that, the
// pool is capped by the work and a "runs on 8 threads" test would be measuring
// three of them. n is small to keep the local scalar run of this file to about a
// second: the property under test is the partition, and the partition is over m.
const (
	p5M = 1200 // 9 blocks at MC = 144
	p5N = 192  // ≥ 8 columns, so Strsm's right-hand-side split reaches 8 workers too
	p5K = 144
)

// p5case is one routine's parallel behaviour under test: one or more shapes, each
// a run that writes its result into dst from inputs fixed once, so that two runs
// of a shape differ only in the thread count.
//
// A routine has more than one shape when its shapes reach the pool by different
// routes. Strsm has two, because its two sides partition different axes.
type p5case struct {
	routine string
	shapes  []p5shape
}

type p5shape struct {
	label  string // printed in the marker, so the covered shapes are stated not implied
	outLen int
	run    func(dst []float32)
}

// p5cases builds the four routines gate-p5.sh judges, at shapes whose ic loop or
// right-hand-side axis has room for every thread count in p5Threads.
//
// Ssymm is exercised right-side. Left-side would make A's dimension m, so k = m =
// 1200 and this one case would cost more than the other three together on the
// scalar path this file also has to run on; right-side keeps the m-partition under
// test (C is still m×n) while A stays n×n. The left-side expansion is covered by
// TestSsymmSweep, and expandSym's own parallel pass is exercised either way.
//
// Strsm is exercised on BOTH sides, and that is the one case here where two shapes
// of a routine are not the same test run twice. The two sides split different axes
// — left splits B's columns, right splits B's rows (block.strips) — and each side's
// rank-update GEMM therefore receives a different m, so the two sides reach the
// blocked nest with different ic-block counts at the same GOMAXPROCS. The left side
// is the side gate-p5.sh's flop formula describes (n·m·(m+1) counts one m×m
// triangle per column of B); the right side is the side that catches a rank update
// which runs only its first ic block, because at m = 1200 a serial run gives that
// GEMM nine ic blocks and an 8-worker run gives it two. That defect existed, was
// wrong by 4e-2 against the float64 oracle at GOMAXPROCS ≤ 2 and right at 8, and
// the only reason it did not ship is that it was found by hand; see par.Serial.
// A bitwise serial-vs-parallel comparison on this shape is what finds the next one.
//
// beta is non-zero in the two cases that have one, so that the beta pass inside
// the parallel region is part of what is compared bitwise. It runs per ic block,
// which makes it exactly the kind of thing a partition bug would corrupt.
func p5cases() []p5case {
	r := rand.New(rand.NewSource(20260815))
	a := randMatrix(r, p5M*p5K)
	b := randMatrix(r, p5K*p5N)
	c0 := randMatrix(r, p5M*p5N)
	sy := randMatrix(r, p5M*p5K) // n×k for Ssyrk with n = p5M
	syc := randMatrix(r, p5M*p5M)
	sym := symMatrix(r, p5N, p5N, true) // right-side Ssymm: A is n×n
	symb := randMatrix(r, p5M*p5N)
	symc := randMatrix(r, p5M*p5N)
	triL := triMatrix(r, p5M, p5M, true, false) // left-side Strsm: A is m×m
	triR := triMatrix(r, p5N, p5N, true, false) // right-side Strsm: A is n×n
	trb := randMatrix(r, p5M*p5N)

	return []p5case{{
		routine: "Sgemm",
		shapes: []p5shape{{
			label:  fmt.Sprintf("NN,m=%d,n=%d,k=%d", p5M, p5N, p5K),
			outLen: p5M * p5N,
			run: func(dst []float32) {
				copy(dst, c0)
				Sgemm(NoTrans, NoTrans, p5M, p5N, p5K, 0.75, a, p5K, b, p5N, -0.25, dst, p5N)
			},
		}},
	}, {
		routine: "Ssyrk",
		shapes: []p5shape{{
			label:  fmt.Sprintf("lower,N,n=%d,k=%d", p5M, p5K),
			outLen: p5M * p5M,
			run: func(dst []float32) {
				copy(dst, syc)
				Ssyrk(Lower, NoTrans, p5M, p5K, 0.75, sy, p5K, -0.25, dst, p5M)
			},
		}},
	}, {
		routine: "Ssymm",
		shapes: []p5shape{{
			label:  fmt.Sprintf("right,lower,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float32) {
				copy(dst, symc)
				Ssymm(Right, Lower, p5M, p5N, 0.75, sym, p5N, symb, p5N, -0.25, dst, p5N)
			},
		}},
	}, {
		routine: "Strsm",
		shapes: []p5shape{{
			label:  fmt.Sprintf("left,lower,N,nonunit,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float32) {
				// Strsm solves in place, so the input is restored per run; that copy is
				// also what makes a repeat call comparable at all.
				copy(dst, trb)
				Strsm(Left, Lower, NoTrans, NonUnit, p5M, p5N, 0.75, triL, p5M, dst, p5N)
			},
		}, {
			label:  fmt.Sprintf("right,lower,N,nonunit,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float32) {
				copy(dst, trb)
				Strsm(Right, Lower, NoTrans, NonUnit, p5M, p5N, 0.75, triR, p5N, dst, p5N)
			},
		}},
	}}
}

// shapeLabels lists a case's shape labels, for the markers. The gate reads the
// covering= field as documentation rather than as a set to match, but a marker that
// named a routine without naming what was run under it would be the "declared next
// to the loop rather than run by it" shape this file's header rejects.
func shapeLabels(c p5case) []string {
	out := make([]string, len(c.shapes))
	for i, s := range c.shapes {
		out[i] = s.label
	}
	return out
}

// withProcs runs fn at GOMAXPROCS=p and restores the previous value. No test in
// this file calls t.Parallel(): GOMAXPROCS is process-wide, so two of them running
// at once would each be measuring the other's setting.
func withProcs(p int, fn func()) {
	prev := runtime.GOMAXPROCS(p)
	defer runtime.GOMAXPROCS(prev)
	fn()
}

// TestP5Determinism is criterion 5: the parallel nest returns the same bits as the
// serial one, at every thread count, for every routine gate-p5.sh judges.
//
// One marker per routine, at the end of all of that routine's shapes, because
// gate-p5.sh's p5_line reads the FIRST line matching a key: a second
// keel-p5-determinism line for the same routine would be invisible to it, and a
// per-shape marker would therefore silently report only one shape's result. So the
// line is emitted once, with the shape count and labels it covers, and only if
// every shape passed at every thread count.
func TestP5Determinism(t *testing.T) {
	for _, c := range p5cases() {
		ok := true
		for _, s := range c.shapes {
			serial := make([]float32, s.outLen)
			withProcs(1, func() { s.run(serial) })

			for _, procs := range p5Threads {
				got := make([]float32, s.outLen)
				var workers int
				withProcs(procs, func() {
					s.run(got)
					workers = WorkersLastCall()
				})
				// The worker count is asserted alongside the bits, because a run that
				// used one worker would agree bitwise for the least interesting reason
				// there is: it never partitioned anything. That is the same failure the
				// gate guards the benchmark rows against, and it would be a silent pass
				// here without this line.
				if want := Workers(procs); workers != want {
					t.Errorf("%s %s at GOMAXPROCS=%d ran on %d workers, want %d: this arm did not "+
						"partition, so its agreement with the serial nest proves nothing",
						c.routine, s.label, procs, workers, want)
					ok = false
				}
				if i, bad := firstBitDiff(serial, got); bad {
					t.Errorf("%s %s at GOMAXPROCS=%d differs from the serial nest at index %d: "+
						"serial %v (%#08x), parallel %v (%#08x) — splitting the ic loop must not "+
						"reassociate any element's sum",
						c.routine, s.label, procs, i, serial[i], math.Float32bits(serial[i]),
						got[i], math.Float32bits(got[i]))
					ok = false
				}
			}
		}
		if !ok {
			continue
		}
		// The shape labels contain commas, so they are joined with semicolons: the
		// gate splits a marker on whitespace and then on the first '=', which leaves
		// the value opaque, but a human reading the log should not have to guess where
		// one shape ends.
		fmt.Printf("keel-p5-determinism: routine=%s mode=bitwise-vs-serial threads=%s shapes=%d covering=%s\n",
			c.routine, joinInts(p5Threads), len(c.shapes), strings.Join(shapeLabels(c), ";"))
	}
}

// TestP5NoState is criterion 6: "no background threads, no state between calls".
//
// Both halves are checkable and both are checked, because both cost nothing until
// a caller runs keel inside something that counts goroutines or calls it twice.
// The second half is what a sync.Pool can break: a pooled packed-A buffer that
// some path read before writing would make the second call's result depend on the
// first's, and no amount of oracle comparison on a single call would see it.
func TestP5NoState(t *testing.T) {
	for _, c := range p5cases() {
		ok := true
		for _, s := range c.shapes {
			withProcs(8, func() {
				first := make([]float32, s.outLen)
				second := make([]float32, s.outLen)

				base := runtime.NumGoroutine()
				s.run(first)
				if n, done := settled(base); !done {
					t.Errorf("%s %s: %d goroutines after the call, baseline %d — the pool outlived it",
						c.routine, s.label, n, base)
					ok = false
				}
				// A second call, from the same inputs, into different memory. Same
				// bits or the library is carrying something between calls.
				s.run(second)
				if i, bad := firstBitDiff(first, second); bad {
					t.Errorf("%s %s: a repeated identical call differs at index %d (%#08x vs %#08x): "+
						"something survived the first call and reached the second's result",
						c.routine, s.label, i, math.Float32bits(first[i]), math.Float32bits(second[i]))
					ok = false
				}
				if n, done := settled(base); !done {
					t.Errorf("%s %s: %d goroutines after the second call, baseline %d",
						c.routine, s.label, n, base)
					ok = false
				}
			})
		}
		if !ok {
			continue
		}
		// One line per routine, for the same reason as the determinism marker: the
		// gate's p5_line takes the first match.
		fmt.Printf("keel-p5-nostate: routine=%s checks=goroutines-return-to-baseline,repeat-call-bit-identical shapes=%d covering=%s\n",
			c.routine, len(c.shapes), strings.Join(shapeLabels(c), ";"))
	}
}

// TestP5Dispatch is criterion 7's declaration half: the chains the library
// advertises, per level. The gate checks these strings against DESIGN.md §4/P5 and
// then pulls on each rung itself with KEEL_FORCE, which is the half a self-report
// cannot do.
//
// Two chains, not one, by the #40 ruling: Level 1 dispatches avx512 → avx2 →
// scalar and Level 3 dispatches avx512 → scalar, because internal/kern has no AVX2
// microkernel and no host here is AVX2-only silicon. The marker states them
// separately because a single chain= field could not express the narrowing at all,
// and a ruling that cannot be stated is one the next session re-litigates.
func TestP5Dispatch(t *testing.T) {
	l1c, knc := L1Chain(), KernChain()
	if len(l1c) == 0 || len(knc) == 0 {
		t.Fatalf("empty dispatch chain: l1=%v kern=%v", l1c, knc)
	}

	// A chain must terminate somewhere that always exists. scalar last, once, and
	// nowhere else: a chain that fell off the end would panic at init on the one
	// class of machine nobody here tests on.
	for _, c := range []struct {
		level string
		chain []string
	}{{"l1", l1c}, {"kern", knc}} {
		if last := c.chain[len(c.chain)-1]; last != "scalar" {
			t.Errorf("%s chain %v ends at %q, not scalar: the fallback has to terminate", c.level, c.chain, last)
		}
		seen := map[string]bool{}
		for _, name := range c.chain {
			if seen[name] {
				t.Errorf("%s chain %v repeats %q", c.level, c.chain, name)
			}
			seen[name] = true
		}
	}

	// The #40 ruling as a checkable relation rather than as prose: Level 3
	// dispatches the Level-1 ladder with rungs REMOVED, in the same order. That is
	// what makes the asymmetry a documented narrowing instead of a second,
	// independently-drifting ladder — and it fails the moment someone adds a
	// Level-3 backend that Level 1 does not have, which would mean the two levels
	// disagree about what this machine is.
	if !subsequence(knc, l1c) {
		t.Errorf("kern chain %v is not a subsequence of the l1 chain %v: Level 3 is supposed to be "+
			"the same ladder with the unbacked rungs removed", knc, l1c)
	}

	// The other direction, and the one a host can actually check: nothing runnable
	// here may be missing from the chain. A backend that ships, works, and is not
	// advertised is a rung dispatch will never climb — silent lost performance,
	// which is the failure mode no correctness test would ever report.
	for _, name := range AvailableL1Backends() {
		if !containsStr(l1c, name) {
			t.Errorf("Level-1 backend %q is runnable here but absent from the advertised chain %v",
				name, l1c)
		}
	}
	for _, id := range AvailableKernels() {
		if _, backend, ok := strings.Cut(id, "/"); ok && !containsStr(knc, backend) {
			t.Errorf("microkernel %s is runnable here but its backend %q is absent from the advertised "+
				"chain %v", id, backend, knc)
		}
	}

	fmt.Printf("keel-p5-dispatch: l1=%s kern=%s\n", strings.Join(l1c, ","), strings.Join(knc, ","))
}

// subsequence reports whether sub appears in seq in order, not necessarily
// contiguously.
func subsequence(sub, seq []string) bool {
	i := 0
	for _, s := range seq {
		if i < len(sub) && sub[i] == s {
			i++
		}
	}
	return i == len(sub)
}

// TestP5TrsmModel is the model half of criterion 1's deferral: Strsm's floor is
// deferred to a measurement PLUS a stated parallelism model, because a scaling
// number with no account of what fraction of the work can scale sets no threshold
// anybody can defend.
//
// The two fractions come from block.TrsmWork, which walks the same MB partition
// Trsm walks. This test's job is to stop them being self-consistent nonsense: a
// pair summing to 1 says nothing unless the total they came from is the total the
// gate's own flop formula computes for the shape. So the sum is checked against
// n·m·(m+1) exactly, at the benchmark's shape and at a ragged one, before either
// fraction is printed.
func TestP5TrsmModel(t *testing.T) {
	for _, s := range []struct{ m, n int }{
		{p5ScaleN, p5ScaleN}, // the shape the Scale benchmark measures
		{p5M, p5N},           // this file's shape
		{100, 7},             // m not a multiple of MB, so the last block is ragged
	} {
		ru, ds := block.TrsmWork(true, s.m, s.n)
		want := float64(s.n) * float64(s.m) * float64(s.m+1)
		if ru+ds != want {
			t.Errorf("TrsmWork(left, %d, %d) accounts for %.0f flops, but the shape has %.0f "+
				"(n*m*(m+1), the count gate-p5.sh recomputes): the model does not describe this work",
				s.m, s.n, ru+ds, want)
		}
	}
	ru, ds := block.TrsmWork(true, p5ScaleN, p5ScaleN)
	total := ru + ds
	if total == 0 {
		t.Fatal("no work at the benchmark shape")
	}
	fmt.Printf("keel-p5-model: routine=Strsm shape=left,m=%d,n=%d mb=%d rank_update=%.5f diag_solve=%.5f\n",
		p5ScaleN, p5ScaleN, block.MB, ru/total, ds/total)
}

// p5ScaleN is the size gate-p5.sh's headline criterion measures at, restated here
// because the Strsm model has to be declared for the shape the ratio is taken at
// rather than for the shape this file's correctness tests use.
const p5ScaleN = 4096

// firstBitDiff reports the first index at which two results differ in their bits,
// and whether they differ at all. Bits, not values: NaN != NaN under ==, and a
// partition bug that produced a NaN where the serial nest produced one too would
// slip through a value comparison in both directions.
func firstBitDiff(a, b []float32) (int, bool) {
	if len(a) != len(b) {
		return 0, true
	}
	for i := range a {
		if math.Float32bits(a[i]) != math.Float32bits(b[i]) {
			return i, true
		}
	}
	return 0, false
}

// settled waits for the goroutine count to return to base and reports what it last
// saw.
//
// It converges rather than asserting immediately, and the reason is worth stating
// so the convergence is not mistaken for a weakened check. internal/par joins its
// workers on a WaitGroup, and a goroutine's last act — Done — necessarily precedes
// its return, so the scheduler has not always reaped it by the time Wait's caller
// runs again. A goroutine on its way out is not a background thread. A parked pool
// is, and it never converges: this still fails on one, after two seconds.
func settled(base int) (int, bool) {
	deadline := time.Now().Add(2 * time.Second)
	n := runtime.NumGoroutine()
	for n > base && time.Now().Before(deadline) {
		runtime.Gosched()
		n = runtime.NumGoroutine()
	}
	return n, n <= base
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, x := range v {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, ",")
}

func containsStr(v []string, want string) bool {
	for _, s := range v {
		if s == want {
			return true
		}
	}
	return false
}
