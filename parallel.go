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
// The caller-facing consequences — serial in the calling goroutine at
// GOMAXPROCS=1, no goroutine outliving a call, a bit-identical result at every
// GOMAXPROCS, and Level 1 unparallelized — are stated once in doc.go and not
// restated here. Every published keel number carries the GOMAXPROCS it was
// measured at in its own row: README's table publishes a 1-thread and an
// 8-thread arm for each Level-3 routine, each against its own denominator.

// Workers reports how many goroutines a Level-3 call would distribute n units of
// independent work over at the current GOMAXPROCS. It is exported as documentation
// of the sizing rule rather than as a tuning knob: the answer is
// min(GOMAXPROCS(0), n), and the only way to change it is to change GOMAXPROCS.
func Workers(n int) int { return par.Workers(n) }

// Why a library exposes WorkersLastCall at all: a benchmark row that believes it
// ran on eight workers and silently ran on one reports a 1.0x scaling ratio, which
// reads as a performance problem when it is a measurement failure — and the two
// want opposite responses. Only the library can settle it, so gate-p5 criterion 3
// requires every Scale row to declare this value and refuses any row that
// disagrees with the thread count in its own name. See block.WorkersLastCall for
// why exposing it does not violate P5's "no state between calls".

// WorkersLastCall reports how many workers the most recently completed Level-3
// call in this process distributed work to, counting the calling goroutine as a
// worker. It is 0 before the first such call.
//
// It is instrumentation for a benchmark harness, not configuration: under
// concurrent Level-3 calls the value belongs to whichever call finished last.
func WorkersLastCall() int { return block.WorkersLastCall() }

// GOMAXPROCS reports runtime.GOMAXPROCS(0): the pool's size bound, read the same
// way the nest reads it.
//
// It exists so a benchmark harness can declare the thread count it believes it set
// without importing runtime alongside keel, and so that the declaration and the
// nest cannot read two different numbers.
func GOMAXPROCS() int { return runtime.GOMAXPROCS(0) }
