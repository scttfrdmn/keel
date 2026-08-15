// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"runtime"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/par"
)

// The Level-3 routines distribute their work over a bounded pool of goroutines
// sized by runtime.GOMAXPROCS(0), started per call and joined before the call
// returns (DESIGN.md §4/P5). Nothing here is configurable: GOMAXPROCS is the
// knob, because it is the knob a caller already has and already expects a Go
// library to respect.
//
// What that means for a caller:
//
//   - GOMAXPROCS=1 runs the serial nest in the calling goroutine, with no
//     goroutine, no atomic and no scheduling in the path. Every published keel
//     number was measured that way.
//   - No goroutine outlives a call, so keel is safe to call from inside something
//     that counts them, and repeated calls leak nothing.
//   - The result does not depend on the thread count. It is BIT-IDENTICAL at every
//     GOMAXPROCS, because the parallel axis partitions the output rather than any
//     single output element's sum. A float32 BLAS whose answer moved with the core
//     count would be a different library on every machine.
//
// Level 1 is not parallelized. Sdot and its neighbours are memory-bound at every
// size where a thread would pay for itself, and BLAS callers reach Level-1
// parallelism by calling from parallel code rather than by having each call
// fan out.

// Workers reports how many goroutines a Level-3 call would distribute n units of
// independent work over at the current GOMAXPROCS. It is exported as documentation
// of the sizing rule rather than as a tuning knob: the answer is
// min(GOMAXPROCS(0), n), and the only way to change it is to change GOMAXPROCS.
func Workers(n int) int { return par.Workers(n) }

// WorkersLastCall reports how many workers the most recently completed Level-3
// call in this process actually distributed work to, counting the calling
// goroutine as a worker. It is 0 before the first such call.
//
// # Why a library exposes this at all
//
// Because scripts/gate-p5.sh criterion 3 requires a benchmark row named
// threads=8 to declare the number of workers it ran on. A row that silently ran
// on one worker produces a 1.0× scaling ratio, which reads as a performance
// problem when it is really a measurement failure — and the two want opposite
// responses. Only the library can answer the question, so the library answers it,
// and the gate refuses any Scale row whose declared worker count disagrees with
// the thread count in its own name.
//
// It is instrumentation, not configuration and not state that anything computes
// from: see block.WorkersLastCall for the precise argument that it does not
// violate P5's "no state between calls", and for its one limitation — under
// concurrent Level-3 calls the value belongs to whichever finished last.
func WorkersLastCall() int { return block.WorkersLastCall() }

// GOMAXPROCS reports runtime.GOMAXPROCS(0): the pool's size bound, read the same
// way the nest reads it.
//
// It exists so a benchmark harness can declare the thread count it believes it set
// without importing runtime alongside keel, and so that the declaration and the
// nest cannot read two different numbers.
func GOMAXPROCS() int { return runtime.GOMAXPROCS(0) }
