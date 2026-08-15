// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package par

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// withProcs runs fn at GOMAXPROCS=p and restores the previous value.
//
// Every test in this file mutates process-wide state, so none of them calls
// t.Parallel(): two tests changing GOMAXPROCS at once would measure each other.
func withProcs(p int, fn func()) {
	prev := runtime.GOMAXPROCS(p)
	defer runtime.GOMAXPROCS(prev)
	fn()
}

func TestWorkersCaps(t *testing.T) {
	withProcs(8, func() {
		for _, c := range []struct{ n, want int }{
			{-1, 0}, {0, 0}, {1, 1}, {3, 3}, {8, 8}, {9, 8}, {1000, 8},
		} {
			if got := Workers(c.n); got != c.want {
				t.Errorf("Workers(%d) at GOMAXPROCS=8 = %d, want %d", c.n, got, c.want)
			}
		}
	})
	withProcs(1, func() {
		if got := Workers(1000); got != 1 {
			t.Errorf("Workers(1000) at GOMAXPROCS=1 = %d, want 1", got)
		}
	})
}

// TestRunCoversEveryUnitOnce is the property the nest depends on for
// correctness: every ic block is packed and multiplied exactly once. A unit run
// twice would double an accumulation into C, and a unit skipped would leave a
// row panel of C holding beta·C — neither of which any tolerance would forgive.
func TestRunCoversEveryUnitOnce(t *testing.T) {
	for _, procs := range []int{1, 2, 3, 8, 13} {
		for _, n := range []int{0, 1, 2, 7, 8, 9, 64, 257} {
			withProcs(procs, func() {
				var mu sync.Mutex
				seen := make([]int, n)
				used := Run(n, func(claim func() int) {
					for u := claim(); u >= 0; u = claim() {
						mu.Lock()
						seen[u]++
						mu.Unlock()
					}
				})
				for u, c := range seen {
					if c != 1 {
						t.Fatalf("GOMAXPROCS=%d n=%d: unit %d ran %d times, want 1", procs, n, u, c)
					}
				}
				if want := Workers(n); used != want {
					t.Fatalf("GOMAXPROCS=%d n=%d: Run reports %d workers, Workers says %d",
						procs, n, used, want)
				}
			})
		}
	}
}

// TestRunGivesEveryWorkerAUnit is what makes Run's return value a count rather
// than an estimate — scripts/gate-p5.sh criterion 3 reads it as the number of
// workers a benchmark row actually ran on. A worker that claimed nothing would
// make that declaration false, so the static first assignment is asserted here
// rather than trusted to a comment.
//
// The check is by goroutine identity: each worker records the units it claimed
// under its own index, and every index must be non-empty.
func TestRunGivesEveryWorkerAUnit(t *testing.T) {
	for _, procs := range []int{2, 3, 8} {
		for _, n := range []int{2, 3, 8, 9, 100} {
			if n < procs {
				continue
			}
			withProcs(procs, func() {
				var mu sync.Mutex
				counts := map[int]int{}
				var next int
				used := Run(n, func(claim func() int) {
					mu.Lock()
					me := next
					next++
					mu.Unlock()
					for u := claim(); u >= 0; u = claim() {
						mu.Lock()
						counts[me]++
						mu.Unlock()
					}
				})
				if used != procs {
					t.Fatalf("GOMAXPROCS=%d n=%d: %d workers, want %d", procs, n, used, procs)
				}
				for w := 0; w < used; w++ {
					if counts[w] == 0 {
						t.Fatalf("GOMAXPROCS=%d n=%d: worker %d claimed nothing, so a %d-worker "+
							"declaration would be false", procs, n, w, used)
					}
				}
			})
		}
	}
}

// TestRunSerialUsesNoGoroutine pins the property every measurement this project
// has published depends on: at GOMAXPROCS=1 the body runs in the caller, so the
// serial nest is still the serial nest and not the parallel one with a pool of
// one. gate-p3, gate-p4, l1-bench.sh, retention.sh and layout-ensemble.sh all
// measure with GOMAXPROCS=1.
func TestRunSerialUsesNoGoroutine(t *testing.T) {
	withProcs(1, func() {
		base := runtime.NumGoroutine()
		var inside int
		if used := Run(16, func(claim func() int) {
			inside = runtime.NumGoroutine()
			for u := claim(); u >= 0; u = claim() {
			}
		}); used != 1 {
			t.Fatalf("Run reports %d workers at GOMAXPROCS=1, want 1", used)
		}
		if inside != base {
			t.Errorf("goroutines during the body: %d, baseline %d — the serial path started one", inside, base)
		}
	})
}

// TestRunLeavesNoGoroutines is DESIGN.md §4/P5's "no background threads" at the
// level where it is implemented. See settled() for why this converges rather
// than asserting immediately.
func TestRunLeavesNoGoroutines(t *testing.T) {
	withProcs(8, func() {
		base := runtime.NumGoroutine()
		for i := 0; i < 20; i++ {
			Run(64, func(claim func() int) {
				for u := claim(); u >= 0; u = claim() {
				}
			})
		}
		if n, ok := settled(base); !ok {
			t.Errorf("goroutines after 20 Runs: %d, baseline %d — a worker outlived its Run", n, base)
		}
	})
}

// settled waits for the goroutine count to come back to base and reports what it
// last saw.
//
// It converges rather than asserting immediately because Run joins its workers on
// a WaitGroup, and a goroutine's last act — Done — necessarily precedes its
// return: the scheduler has not always reaped it by the time Wait's caller runs
// again. That is a goroutine on its way out, which is not what "no background
// threads" is about; a parked pool never converges, and this still fails on one.
// The same distinction, and the same helper, appear in the root package's
// p5_test.go, which is where the gate reads it.
func settled(base int) (int, bool) {
	deadline := time.Now().Add(2 * time.Second)
	n := runtime.NumGoroutine()
	for n > base && time.Now().Before(deadline) {
		runtime.Gosched()
		n = runtime.NumGoroutine()
	}
	return n, n <= base
}
