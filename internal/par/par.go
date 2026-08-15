// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package par is the bounded worker pool the Level-3 nest distributes work over
// (DESIGN.md §4/P5: "a bounded worker pool sized by runtime.GOMAXPROCS(0) …  no
// background threads, no state between calls").
//
// It is deliberately smaller than a work-stealing scheduler. The nest hands it a
// count of independent units and a body; it starts min(GOMAXPROCS, units)
// goroutines, each pulling unit indices until they run out, and joins them before
// returning. There is no pool that outlives a call, no channel, and no state
// carried between Runs — which is what makes the design instruction's second
// sentence checkable rather than aspirational (scripts/gate-p5.sh criterion 6).
//
// # Why the goroutines are per call rather than a resident pool
//
// A resident pool would save the goroutine starts. At the sizes P5 measures that
// saving is unmeasurable — a Run over one (jc, pc) block of a 4096³ SGEMM does
// ~226 MFLOP per unit against a goroutine start of a few hundred nanoseconds —
// and it would cost the property the gate checks: a library that parks eight
// goroutines forever is a library that cannot be called from inside something
// that counts them. Per-call goroutines are what "no background threads" means
// operationally, so that is what this does.
//
// # Why claim() rather than a range per worker
//
// A static range per worker is the obvious partition and it is the wrong one for
// this nest. Ssyrk's ic blocks differ in work by a factor of the block count —
// the triangular mask keeps columns up to the block's own rows — so an equal
// split of the *index* space is a wildly unequal split of the *work*. Dynamic
// claiming makes the makespan (ideal + the cost of the last unit claimed) rather
// than (ideal × the worst worker's share).
//
// # Why the worker count Run returns is exact and not an estimate
//
// scripts/gate-p5.sh criterion 3 requires the library to declare how many
// workers a benchmark row actually used, because a threads=8 row that silently
// ran on one worker reads as a performance problem rather than as the
// measurement failure it is. A count that came from "GOMAXPROCS, probably" would
// not catch that. So the assignment guarantees it: worker j is handed unit j
// before any dynamic claiming begins, and Run does not return until every
// goroutine it started has been joined. With w = min(GOMAXPROCS, n) ≤ n, every
// worker therefore has at least one unit, and w is the observed count rather
// than a prediction of it.
package par

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Workers reports how many goroutines Run will use for n independent units of
// work: GOMAXPROCS(0), capped by the work there is to do.
//
// The cap is not just tidiness. It is what makes Run's return value the real
// worker count (see the package doc), and it is what keeps a two-block problem
// from starting thirty-two goroutines to leave thirty of them idle.
func Workers(n int) int {
	if n < 1 {
		return 0
	}
	w := runtime.GOMAXPROCS(0)
	if w < 1 {
		w = 1
	}
	if w > n {
		w = n
	}
	return w
}

// Run distributes the units 0..n-1 over Workers(n) goroutines and returns how
// many of them ran. Each goroutine is given its own claim function, which returns
// the next unit index to work on, or -1 when there are none left.
//
// body may be called concurrently and must confine its writes to memory the unit
// index selects. The units may be executed in any order and any interleaving; Run
// makes no promise beyond having joined every goroutine by the time it returns.
//
// At Workers(n) == 1 there is no goroutine at all: body runs in the calling
// goroutine and the count is 1, because the caller is the worker. That is the
// path every measurement this project has taken so far runs on — gate-p3,
// gate-p4, l1-bench.sh, retention.sh and layout-ensemble.sh all pin
// GOMAXPROCS=1 — so pinning one thread continues to measure exactly the serial
// nest it measured before this package existed, with no goroutine, no atomic and
// no scheduling in the path.
func Run(n int, body func(claim func() int)) int {
	w := Workers(n)
	switch w {
	case 0:
		return 0
	case 1:
		Serial(n, body)
		return 1
	}
	// The dynamic phase starts after the units handed out statically below, so a
	// worker's first claim is its own index and cannot be stolen.
	var next atomic.Int64
	next.Store(int64(w))
	var wg sync.WaitGroup
	wg.Add(w)
	for j := 0; j < w; j++ {
		go func(first int) {
			defer wg.Done()
			body(func() int {
				if first >= 0 {
					u := first
					first = -1
					return u
				}
				u := next.Add(1) - 1
				if u >= int64(n) {
					return -1
				}
				return int(u)
			})
		}(j)
	}
	wg.Wait()
	return w
}

// Serial runs all n units in the calling goroutine, whatever GOMAXPROCS says.
//
// # Why this is a separate function and not Run(1, body)
//
// Because Run's first argument is the number of UNITS, not the number of workers,
// and conflating the two is a wrong answer rather than a slow one. `Run(1, body)`
// asks for one unit: the claim function yields index 0 and then -1, so every unit
// above the first is silently never executed.
//
// That is not hypothetical. It is the defect this function exists to make
// unwriteable: block.gemm's non-parallel path was written as `par.Run(1, body)`,
// and since Trsm's rank update has ceil(m/MC) ic blocks — four of them for a
// right-side solve at m=500 on one thread — three quarters of the update was
// skipped. Right-side Strsm came out wrong by 4e-2 against the float64 oracle at
// GOMAXPROCS ≤ 2 and right by 6e-7 at GOMAXPROCS=8, because a wide pool cuts each
// strip small enough that ceil(im/MC) really is 1. A bug that a *narrower*
// machine exposes and a wider one hides is the worst shape available, and the
// caller reads far better as "run these serially" than as "run these on one
// worker" anyway.
func Serial(n int, body func(claim func() int)) {
	next := 0
	body(func() int {
		if next >= n {
			return -1
		}
		next++
		return next - 1
	})
}
