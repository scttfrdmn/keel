// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/l164"
)

// P5's determinism, no-state and dispatch tests at float64 (#193), for gate-p5's
// KEEL_ELEM=f64 arm. Produced from p5_test.go by a one-off rename and kept as
// ordinary test code; a float32 change there must be mirrored here by hand.
//
// The determinism and no-state markers use float32's keys, keel-p5-determinism and
// keel-p5-nostate, with routine=Dgemm and the other D names: the gate reads each
// routine's line by its routine= field, so the two elements cannot collide. The
// dispatch marker has its own key, keel-p5-dispatch64, because it is not per routine.

type p5caseD struct {
	routine string
	shapes  []p5shapeD
}

type p5shapeD struct {
	label  string // printed in the marker, so the covered shapes are stated not implied
	outLen int
	run    func(dst []float64)
}

// p5casesD builds the four routines gate-p5.sh judges, at shapes whose ic loop or
// right-hand-side axis has room for every thread count in p5Threads.
//
// Dsymm is exercised right-side. Left-side would make A's dimension m, so k = m =
// 1200 and this one case would cost more than the other three together on the
// scalar path this file also has to run on; right-side keeps the m-partition under
// test (C is still m×n) while A stays n×n. That also fixes which half of the
// symmetric pack this file reaches: right-side makes A the second operand, so only
// pack.BSymPanelsPart runs here. TestSsymmSweep drives both halves at every size.
//
// Dtrsm is exercised on BOTH sides, and that is the one case here where two shapes
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
func p5casesD() []p5caseD {
	r := rand.New(rand.NewSource(20260815))
	a := randMatrix64(r, p5M*p5K)
	b := randMatrix64(r, p5K*p5N)
	c0 := randMatrix64(r, p5M*p5N)
	sy := randMatrix64(r, p5M*p5K) // n×k for Dsyrk with n = p5M
	syc := randMatrix64(r, p5M*p5M)
	sym := symMatrix64(r, p5N, p5N, true) // right-side Dsymm: A is n×n
	symb := randMatrix64(r, p5M*p5N)
	symc := randMatrix64(r, p5M*p5N)
	triL := triMatrix64(r, p5M, p5M, true, false) // left-side Dtrsm: A is m×m
	triR := triMatrix64(r, p5N, p5N, true, false) // right-side Dtrsm: A is n×n
	trb := randMatrix64(r, p5M*p5N)

	return []p5caseD{{
		routine: "Dgemm",
		shapes: []p5shapeD{{
			label:  fmt.Sprintf("NN,m=%d,n=%d,k=%d", p5M, p5N, p5K),
			outLen: p5M * p5N,
			run: func(dst []float64) {
				copy(dst, c0)
				Dgemm(NoTrans, NoTrans, p5M, p5N, p5K, 0.75, a, p5K, b, p5N, -0.25, dst, p5N)
			},
		}},
	}, {
		routine: "Dsyrk",
		shapes: []p5shapeD{{
			label:  fmt.Sprintf("lower,N,n=%d,k=%d", p5M, p5K),
			outLen: p5M * p5M,
			run: func(dst []float64) {
				copy(dst, syc)
				Dsyrk(Lower, NoTrans, p5M, p5K, 0.75, sy, p5K, -0.25, dst, p5M)
			},
		}},
	}, {
		routine: "Dsymm",
		shapes: []p5shapeD{{
			label:  fmt.Sprintf("right,lower,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float64) {
				copy(dst, symc)
				Dsymm(Right, Lower, p5M, p5N, 0.75, sym, p5N, symb, p5N, -0.25, dst, p5N)
			},
		}},
	}, {
		routine: "Dtrsm",
		shapes: []p5shapeD{{
			label:  fmt.Sprintf("left,lower,N,nonunit,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float64) {
				// Dtrsm solves in place, so the input is restored per run; that copy is
				// also what makes a repeat call comparable at all.
				copy(dst, trb)
				Dtrsm(Left, Lower, NoTrans, NonUnit, p5M, p5N, 0.75, triL, p5M, dst, p5N)
			},
		}, {
			label:  fmt.Sprintf("right,lower,N,nonunit,m=%d,n=%d", p5M, p5N),
			outLen: p5M * p5N,
			run: func(dst []float64) {
				copy(dst, trb)
				Dtrsm(Right, Lower, NoTrans, NonUnit, p5M, p5N, 0.75, triR, p5N, dst, p5N)
			},
		}},
	}}
}

// shapeLabelsD lists a case's shape labels, for the markers. The gate reads the
// covering= field as documentation rather than as a set to match, but a marker that
// named a routine without naming what was run under it would be the "declared next
// to the loop rather than run by it" shape this file's header rejects.
func shapeLabelsD(c p5caseD) []string {
	out := make([]string, len(c.shapes))
	for i, s := range c.shapes {
		out[i] = s.label
	}
	return out
}

// withProcs runs fn at GOMAXPROCS=p and restores the previous value. No test in
// this file calls t.Parallel(): GOMAXPROCS is process-wide, so two of them running
// at once would each be measuring the other's setting.
func TestP5DeterminismD(t *testing.T) {
	for _, c := range p5casesD() {
		ok := true
		for _, s := range c.shapes {
			serial := make([]float64, s.outLen)
			withProcs(1, func() { s.run(serial) })

			for _, procs := range p5Threads {
				got := make([]float64, s.outLen)
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
				//
				// The expectation is procs itself, not Workers(procs) (#87). Workers takes
				// a UNIT count, not a thread count, and returns min(GOMAXPROCS(0), units);
				// called here it was passed procs as the unit count AND evaluated outside
				// withProcs, so it read the ambient GOMAXPROCS and the expectation became a
				// function of the host's core count. On a machine with >= 8 cores it
				// coincidentally equalled procs; on a 2-core CI runner it was 2 against an
				// actual 3 and 8, and the library was right both times. A test whose
				// expected value moves with the hardware is not testing the library.
				//
				// procs is the right number because par.Workers reads GOMAXPROCS(0) rather
				// than NumCPU — so an explicitly-set GOMAXPROCS=8 really does start 8
				// workers on a 2-core box — and because every shape here has more units
				// than max(p5Threads): m=1200 over MC=144 is 9 ic blocks against 8 threads.
				// Shrink p5M and this goes red, which is correct: the arm would have
				// stopped partitioning max-way and its bitwise agreement would prove less.
				if workers != procs {
					t.Errorf("%s %s at GOMAXPROCS=%d ran on %d workers, want %d: this arm did not "+
						"partition, so its agreement with the serial nest proves nothing",
						c.routine, s.label, procs, workers, procs)
					ok = false
				}
				if i, bad := firstBitDiffD(serial, got); bad {
					t.Errorf("%s %s at GOMAXPROCS=%d differs from the serial nest at index %d: "+
						"serial %v (%#016x), parallel %v (%#016x) — splitting the ic loop must not "+
						"reassociate any element's sum",
						c.routine, s.label, procs, i, serial[i], math.Float64bits(serial[i]),
						got[i], math.Float64bits(got[i]))
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
			c.routine, joinInts(p5Threads), len(c.shapes), strings.Join(shapeLabelsD(c), ";"))
	}
}

// TestP5NoStateD is criterion 6: "no background threads, no state between calls".
//
// Both halves are checkable and both are checked, because both cost nothing until
// a caller runs keel inside something that counts goroutines or calls it twice.
// The second half is what a sync.Pool can break: a pooled packed-A buffer that
// some path read before writing would make the second call's result depend on the
// first's, and no amount of oracle comparison on a single call would see it.
func TestP5NoStateD(t *testing.T) {
	for _, c := range p5casesD() {
		ok := true
		for _, s := range c.shapes {
			withProcs(8, func() {
				first := make([]float64, s.outLen)
				second := make([]float64, s.outLen)

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
				if i, bad := firstBitDiffD(first, second); bad {
					t.Errorf("%s %s: a repeated identical call differs at index %d (%#016x vs %#016x): "+
						"something survived the first call and reached the second's result",
						c.routine, s.label, i, math.Float64bits(first[i]), math.Float64bits(second[i]))
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
			c.routine, len(c.shapes), strings.Join(shapeLabelsD(c), ";"))
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
func firstBitDiffD(a, b []float64) (int, bool) {
	if len(a) != len(b) {
		return 0, true
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return i, true
		}
	}
	return 0, false
}

// TestP5DispatchD is TestP5Dispatch for float64: both float64 chains end at scalar,
// repeat no rung, are subsequences of the ISA ladder, and advertise every float64
// backend runnable here.
func TestP5DispatchD(t *testing.T) {
	l1c, knc := L164Chain(), KernChain64()
	ladder := isaLadder()
	for _, c := range []struct {
		level string
		chain []string
	}{{"l164", l1c}, {"kern64", knc}} {
		if len(c.chain) == 0 || c.chain[len(c.chain)-1] != kern.Scalar {
			t.Errorf("%s chain %v does not end at scalar", c.level, c.chain)
		}
		seen := map[string]bool{}
		for _, name := range c.chain {
			if seen[name] {
				t.Errorf("%s chain %v repeats %q", c.level, c.chain, name)
			}
			seen[name] = true
		}
		if !subsequence(c.chain, ladder) {
			t.Errorf("%s chain %v is not a subsequence of the arch ISA ladder %v", c.level, c.chain, ladder)
		}
	}
	for _, name := range l164.Names() {
		if !containsStr(l1c, name) {
			t.Errorf("float64 Level-1 backend %q is runnable here but absent from L164Chain %v", name, l1c)
		}
	}
	for _, id := range AvailableKernels64() {
		if _, backend, ok := strings.Cut(id, "/"); ok && !containsStr(knc, backend) {
			t.Errorf("float64 microkernel %s is runnable here but its backend %q is absent from KernChain64 %v", id, backend, knc)
		}
	}
	fmt.Printf("keel-p5-dispatch64: l1=%s kern=%s\n", strings.Join(l1c, ","), strings.Join(knc, ","))
}
