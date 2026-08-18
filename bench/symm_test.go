// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"testing"

	"github.com/scttfrdmn/keel"
)

// The shape where Ssymm's handling of its symmetric operand is a large fraction of
// the call, which is what issue #36 changed and the only place a measurement of that
// change can be read.
//
// A is m×m and B is m×n, so the multiply is 2·m·n·m flops while reflecting A is
// O(m²) — a ratio of 1/(2n), independent of m. At n = 1 that is half the arithmetic
// of the whole call and at n = 1024 it is 0.05%, so one sweep over n at fixed m spans
// the crossover, and at GOMAXPROCS=1 the wide row is its own control: a change that
// moved it there is a change to the nest rather than to the pack.
//
// Two corrections from the first A/B run of this fixture (2026-08-18, dev host,
// scalar kernel, indicative only — see the #36 thread):
//
// The flop ratio is not a time ratio. Every shipped kernel has NR = 32 and the nest
// zero-pads n out to it, so a call with n < 32 buys one tile of column work whatever
// n is: n = 1 measured 35.9× the per-column time of the n = 1024 row, close to the 32
// padding predicts. The reflection was ~5% of the time at n = 1, not the ~50% its
// flop share suggests.
//
// And the control does not hold above one thread. At GOMAXPROCS=8 every row moved,
// n = 1024 included (+5.4%), so the wide row cannot be read there as "the nest is
// unchanged". Why is not established by this fixture and should not be asserted from
// it. The deleted expansion was *not* a serial region — it ran under par.Run over
// rows — and consistently with that its measured cost fell with thread count (the
// n = 1 delta was 2.54 ms at one thread and 1.15 ms at eight), which is far too small
// to be the 11.9 ms the n = 1024 row moved by. The 16.9 MB per-call allocation is the
// unexcluded candidate, and the base arm's variance (±4-6% against the new arm's
// ±0-2%) is itself an effect of it. What does the control's job unconditionally is
// the bit-for-bit equivalence test, internal/pack.TestSymPackMatchesExpansion.
//
// Nothing here is new work for the routine. Until #36, Ssymm reflected A into a dense
// m×m square before the nest ran; now internal/pack reads the stored triangle in
// place. Both do one pass over A's entries, so this sweep is an A/B between two
// revisions (scripts/bench.sh's compare), not a criterion with a floor.
//
// n=1 is also the shape a future Ssymv occupies (issue #36's "it also covers a future
// Ssymv"), so this row is the standing measurement of how much of Ssymv's case the
// Level-3 path already serves.
var symmNarrow = []struct{ m, n int }{
	{2048, 1},
	{2048, 8},
	{2048, 32},
	{2048, 128},
	{2048, 1024}, // control at GOMAXPROCS=1 only: the reflection is 0.05% of the call
}

// BenchmarkSymmNarrow times Ssymm with far fewer columns of B than rows of A.
//
// Left and Lower, matching BenchmarkScale's Ssymm row and BenchmarkEdgeSsymm, so
// the three fixtures differ in shape alone.
func BenchmarkSymmNarrow(b *testing.B) {
	provenance()
	for _, s := range symmNarrow {
		m, n := s.m, s.n
		b.Run(fmt.Sprintf("m=%d/n=%d", m, n), func(b *testing.B) {
			a, bm, c := makeMat(m, m), makeMat(m, n), makeMat(m, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Ssymm(keel.Left, keel.Lower, m, n, 1, a, m, bm, n, 0, c, n)
			}
			rateWork(b, symmWorkMN(m, n))
		})
	}
}
