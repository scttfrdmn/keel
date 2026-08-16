// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"testing"

	"github.com/scttfrdmn/keel"
)

// Edge-handling throughput: the fixture for issue #22's A-vs-C ranking.
//
// # Why these shapes and not the ones already benchmarked
//
// At 2048³ the two candidates are byte-identical almost everywhere they execute.
// 2048 is a multiple of both MR (2 or 4) and NR (32), so Sgemm takes the fringe
// branch at `internal/block/block.go:456` zero times, and an A/B measured there
// reports the layout floor with the edge code never entered — the same class of
// error as #48's tautology trap: a comparison whose arms cannot differ is not a
// comparison. The ruling on #22 names the falsifier that follows from that, and it
// is the reason for the last group below: **the interior controls must come out a
// wash.** If A and C differ at n=4096, this harness is measuring something other
// than edge handling and the run is void.
//
// # The two groups measure two different claims
//
// The gemm-fringe shapes (`edgeGemm`) are where padding is proportionally worst:
// one column past a multiple of NR pads 31 of 32 columns, and k is large so the
// K-loop dominates and the fringe cost is not lost in setup.
//
// The masked shapes (`edgeSyrk`, `edgeSymm`) are the half of C's claim that B
// structurally cannot reach. A diagonal tile's live region is a per-row [lo, hi)
// rather than a rectangle, so Ssyrk, Ssymm and Strsm take the scratch-tile path on
// their diagonal blocks *at every size* — including edge-free ones. Ssyrk at
// n=2048 is therefore an edge-heavy fixture that looks like an interior one, and it
// is also criterion 7's own numerator: if C is worth anything on mask-crossing
// tiles, it moves the 87-92% Ssyrk/Sgemm ratio that gate-p4 grades.
//
// Not a gate. scripts/edge-bench.sh A/Bs two builds over these names and prints a
// table; nothing here is a threshold.
var (
	// m, n, k. MR is 2 or 4 and NR is 32, so 31/33 straddle NR and 63/65 straddle
	// 2*NR; m=3 and m=5 straddle both MR values.
	edgeGemm = [][3]int{
		{31, 31, 2048},
		{33, 33, 2048},
		{63, 63, 2048},
		{65, 65, 2048},
		{3, 2048, 2048},
		{5, 2048, 2048},
		{2048, 33, 2048},
	}
	// Interior controls: no fringe tile exists at either size, so any delta here
	// voids the run rather than informing it.
	edgeControl = [][3]int{
		{2048, 2048, 2048},
		{4096, 4096, 4096},
	}
	// n, k for Ssyrk. 2048 is the edge-free-but-fully-masked case; 33 and 65 are
	// ragged and masked at once.
	edgeSyrk = [][2]int{
		{33, 2048},
		{65, 2048},
		{2048, 2048},
	}
	// m, n for Ssymm, the second mask shape.
	edgeSymm = [][2]int{
		{33, 2048},
		{2048, 2048},
	}
)

// gemmWorkMNK is gemmWork for a non-square shape. Kept separate rather than
// generalizing gemmWork, because gemmWork is the *shared* declaration that makes
// BenchmarkSgemm and BenchmarkOpenBLAS provably one numerator (see bench_test.go),
// and widening its signature would put a second caller between those two.
func gemmWorkMNK(m, n, k int) work {
	return work{
		m: m, n: n, k: k,
		formula: "2*m*n*k",
		flops:   2 * float64(m) * float64(n) * float64(k),
	}
}

// syrkWorkNK is syrkWork with k free of n: one multiply and one add per (i, j, p)
// over one triangle's n(n+1)/2 entries, i.e. k*n*(n+1).
func syrkWorkNK(n, k int) work {
	return work{
		m: n, n: n, k: k,
		formula: "k*n*(n+1)",
		flops:   float64(k) * float64(n) * float64(n+1),
	}
}

// symmWorkMN is Ssymm's count for a non-square shape: 2*m*n*m for Left side, where
// A is m×m. The discarded half of a diagonal tile is not counted, as everywhere
// else in this package — counting executed flops instead of useful ones would hide
// the very cost this fixture exists to measure.
func symmWorkMN(m, n int) work {
	return work{
		m: m, n: n, k: m,
		formula: "2*m*n*m",
		flops:   2 * float64(m) * float64(n) * float64(m),
	}
}

func BenchmarkEdgeSgemm(b *testing.B) {
	provenance()
	for _, s := range append(append([][3]int{}, edgeGemm...), edgeControl...) {
		m, n, k := s[0], s[1], s[2]
		b.Run(fmt.Sprintf("m=%d/n=%d/k=%d", m, n, k), func(b *testing.B) {
			a, bm, c := makeMat(m, k), makeMat(k, n), makeMat(m, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Sgemm(keel.NoTrans, keel.NoTrans, m, n, k, 1, a, k, bm, n, 0, c, n)
			}
			rateWork(b, gemmWorkMNK(m, n, k))
		})
	}
}

func BenchmarkEdgeSsyrk(b *testing.B) {
	provenance()
	for _, s := range edgeSyrk {
		n, k := s[0], s[1]
		b.Run(fmt.Sprintf("n=%d/k=%d", n, k), func(b *testing.B) {
			a, c := makeMat(n, k), makeMat(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Ssyrk(keel.Lower, keel.NoTrans, n, k, 1, a, k, 0, c, n)
			}
			rateWork(b, syrkWorkNK(n, k))
		})
	}
}

func BenchmarkEdgeSsymm(b *testing.B) {
	provenance()
	for _, s := range edgeSymm {
		m, n := s[0], s[1]
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
