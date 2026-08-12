// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"testing"

	"github.com/scttfrdmn/keel"
)

// SGEMM throughput. The gate's criterion is at 2048³ (DESIGN.md §4/P3), and the
// smaller sizes are here because a single point cannot show whether a rate is the
// kernel's or the memory system's: 256³ fits in cache and 2048³ does not, so the
// pair is what makes the 2048 number interpretable.
//
// Sub-benchmark names are "n=<size>", which is what scripts/gate-p3.sh's bench
// filter and threshold keys are written against.
var gemmSizes = []int{256, 512, 1024, 2048}

// BenchmarkSgemm measures C = A·B for square row-major matrices, alpha = 1 and
// beta = 0 — the same call the OpenBLAS reference makes in openblas_test.go, so
// the ratio the gate computes is between two implementations of one operation
// rather than between two operations.
//
// alpha = 1 takes no shortcut worth worrying about: alpha is folded into the
// packed A regardless of its value (internal/pack), so the only thing a general
// alpha would add to this measurement is a multiply per A element, amortized over
// nc columns of arithmetic. beta = 0 skips reading C, which is what a caller
// computing a fresh product does and what the reference does too.
func BenchmarkSgemm(b *testing.B) {
	provenance()
	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			// Allocated inside the sub-benchmark and the timer reset after: at
			// 2048 the three matrices are 48 MB, and a run filtered to one size
			// should not pay for the others' pages or count its own setup.
			a, bm, c := makeMat(n, n), makeMat(n, n), makeMat(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Sgemm(keel.NoTrans, keel.NoTrans, n, n, n, 1, a, n, bm, n, 0, c, n)
			}
			rateWork(b, gemmWork(n)) // 2*m*n*k, declared and checked by gate-p4
		})
	}
}

// BenchmarkSsyrk is P4's numerator in the criterion "Ssyrk >= 85% of Sgemm GFLOPS
// at same size" (DESIGN.md §4/P4). Same sizes as BenchmarkSgemm, and k = n, so the
// two rates are at one shape rather than at two — a ratio between rates measured at
// different sizes is not a ratio (DESIGN.md §7 rule 7), and gate-p4 checks the
// declared dimensions against each other rather than trusting this comment.
//
// uplo=Lower, trans=NoTrans, alpha=1, beta=0: the flags do not change the flop
// count, and the four corners of the triangular mask are a correctness question
// (tri_test.go), not a throughput one — the masked path is the same code with the
// diagonal on the other side. beta=0 skips reading C, as BenchmarkSgemm's does.
//
// What this measures against Sgemm is the whole cost of the derivation: the tiles
// that straddle the diagonal are computed in full and half-discarded, and the flop
// count declared here counts only the half kept. The gap from 100% is that waste
// plus the blocks the mask skipped unevenly, and 85% is where DESIGN.md put the bar.
func BenchmarkSsyrk(b *testing.B) {
	provenance()
	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			a, c := makeMat(n, n), makeMat(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keel.Ssyrk(keel.Lower, keel.NoTrans, n, n, 1, a, n, 0, c, n)
			}
			rateWork(b, syrkWork(n)) // k*n*(n+1) — one triangle, not one square
		})
	}
}

// makeMat fills a rows×cols row-major matrix with a bounded, non-degenerate
// pattern.
//
// Not random, because a benchmark should be reproducible and the data's *values*
// do not affect an FMA's cost. Not constant either: a matrix of ones would let a
// future compiler or a reviewer's eye mistake a folded computation for a fast one,
// and the accumulated result stays representable so nothing turns into an infinity
// half way through a 2048-deep sum.
func makeMat(rows, cols int) []float32 {
	v := make([]float32, rows*cols)
	for i := range v {
		v[i] = float32((i%13)-6) * 0.125
	}
	return v
}
