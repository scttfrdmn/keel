// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package l2

import (
	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/par"
)

// The parallel Level-2 routines — Sgemv both ways, Sger, Ssyr, Ssyr2 and Strmv's
// NoTrans case — split their OUTPUT into contiguous ranges, one per worker, and
// every output element is computed by exactly one worker running exactly the code
// the serial call runs on it. So the result is bit-identical at every GOMAXPROCS
// for the reason Level 3's is (doc.go): the split is of the output, never of one
// element's sum.
//
// The serial call is the same function body over the whole range, not a second
// loop, so there is no serial twin for the parallel one to drift from.

// Grain is how many element updates of A one worker must have before a Level-2
// call starts it (Ssyr2 updates each element twice): a call with fewer than
// 2·Grain runs serially in the calling goroutine, with no goroutine and no
// atomic.
//
// A minimum per WORKER rather than a size cutoff, so the width grows with the work
// instead of jumping from one to GOMAXPROCS at a single n: 3·Grain gets three
// workers at GOMAXPROCS=8.
//
// MEASURED 2026-10-24 on ONE host — orion, M4 Pro 10P+4E, NEON, its measured pueue
// slot, dev tier, unjudged — by BenchmarkL2Par with the grain forced to 1 so every
// width ran at every size. Speedup over one thread, median of 5, for the dense
// three (SgemvN, SgemvT, Sger), by elements per worker:
//
//	32K  (256², w=2):            0.75–0.77
//	64K  (512², w=4):            0.96–1.51
//	128K (512² w=2, 1024² w=8):  1.06–2.46
//
// so 1<<17 is the smallest power of two at which no swept row lost; the
// triangular three agree (1.36–2.47 at 128K per worker). Above it the gain is the
// memory system's to give: at 4096² and 8 workers SgemvN is 3.6x and SgemvT 2.3x.
// A host with a different goroutine-start cost or cache would place it elsewhere,
// as MB's comment says of MB.
var Grain = 1 << 17

// Cols is the alignment unit of a column split: gemvTrans cuts y at multiples
// of 16 floats. Not for the bits — every Axpy backend does the same operation on
// every element whatever its position (an FMA in each vector lane, the masked
// tail included; one compiled scalar statement on the scalar path), so an offset
// subrange returns the same bits as the whole — but so that every worker but the
// last runs whole 512-bit vectors, and pays no masked tail on each of A's m rows.
const Cols = 16

// workers is the worker count for a Level-2 call over work elements of A whose
// output splits into at most units independent pieces.
func workers(work, units int) int {
	w := par.Workers(units)
	if c := work / Grain; c < w {
		w = c
	}
	return max(w, 1)
}

// split runs fn over a partition of [0, axis) into contiguous ranges, in
// parallel when the work is large enough, and records the width it used.
//
// align > 1 cuts at multiples of align (the last range takes the remainder).
// cost, when non-nil, gives row i's share of the work, and the cuts then balance
// cost rather than rows — the triangular routines' rows run from 1 to n elements,
// and an equal row split would hand one worker three times another's share.
func split(axis, work, align int, cost func(i int) int, fn func(lo, hi int)) {
	units := (axis + align - 1) / align
	w := workers(work, units)
	if w == 1 {
		fn(0, axis)
		block.NoteWorkers(1)
		return
	}
	cut := make([]int, w+1)
	cut[w] = axis
	switch {
	case cost != nil:
		total := 0
		for i := 0; i < axis; i++ {
			total += cost(i)
		}
		u, sum := 1, 0
		for i := 0; i < axis && u < w; i++ {
			sum += cost(i)
			for u < w && sum*w >= u*total {
				cut[u] = i + 1
				u++
			}
		}
	default:
		for u := 1; u < w; u++ {
			cut[u] = min(axis, u*units/w*align)
		}
	}
	block.NoteWorkers(par.Run(w, func(claim func() int) {
		for u := claim(); u >= 0; u = claim() {
			fn(cut[u], cut[u+1])
		}
	}))
}

// triCost is a triangular routine's per-row work: the stored part of row i.
func triCost(lower bool, n int) func(i int) int {
	if lower {
		return func(i int) int { return i + 1 }
	}
	return func(i int) int { return n - i }
}
