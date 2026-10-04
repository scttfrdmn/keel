// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build darwin && cgo && accelerate

package bench

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// TestAccelerateComputesTheSameProduct runs BEFORE any rate is believed.
//
// A reference that computes something else is not a reference, however fast it
// is. The OpenBLAS arm inherits this from the gate's own oracle sweep; this arm
// has no gate, so it carries the check here. Tolerance comes from
// internal/oracle and nowhere else (§7 rule 6).
//
// # Two input sets, because the obvious one cannot fail on rounding
//
// `makeMat` returns `((i%13)-6) * 0.125` — every value an exact multiple of 1/8,
// so every product is a multiple of 1/64 and every sum of them at these sizes
// fits float32's mantissa exactly. The first version of this test used only that
// and reported `worst |delta| = 0`, which looked like a strong result and was
// arithmetic: the tolerance was inert and could not have been exceeded by any
// correct implementation. §5 rule 7, in the precision dimension.
//
// So the dyadic set is kept and its expectation TIGHTENED to exact equality,
// where it is a real claim, and a second non-dyadic set is added in which float32
// accumulation genuinely rounds and the oracle tolerance is what decides.
//
// # And a positive control, because neither set proves the check discriminates
//
// Both sets pass trivially if the comparison is broken. The control calls the
// reference with B transposed — the realistic failure mode for a CBLAS binding,
// where row-major/column-major confusion silently computes A·Bᵀ — and requires
// the check to REJECT it.
func TestAccelerateComputesTheSameProduct(t *testing.T) {
	const n = 192 // large enough to be blocked, small enough to reference-multiply

	dyadic := func(i int) float32 { return float32((i%13)-6) * 0.125 }
	// Non-dyadic: 1/7 and 1/3 scalings are not representable in binary, so the
	// products do not land on exact boundaries and the sum accumulates error.
	rounding := func(i int) float32 { return float32(float64(i%17-8) / 7.0 * 0.3333333333333333) }

	for _, tc := range []struct {
		name  string
		gen   func(int) float32
		exact bool
	}{
		{"dyadic (exact by construction)", dyadic, true},
		{"non-dyadic (tolerance is load-bearing)", rounding, false},
	} {
		a, b := genMat(n, tc.gen), genMat(n, tc.gen)
		worst, tol := accelerateWorstDelta(n, a, b, false)
		switch {
		case tc.exact && worst != 0:
			t.Errorf("%s: worst |delta| = %g, want exactly 0 — these inputs are all "+
				"multiples of 1/8 and every intermediate fits float32 exactly, so any "+
				"nonzero delta is a real disagreement and not rounding", tc.name, worst)
		case !tc.exact && worst > tol:
			t.Errorf("%s: worst |delta| = %g exceeds oracle tolerance %g", tc.name, worst, tol)
		default:
			t.Logf("keel-accelerate-correct[%s]: worst |delta| %g, tolerance %g", tc.name, worst, tol)
		}
	}

	// The positive control. Without it, every line above is compatible with a
	// comparison that cannot see anything.
	a, b := genMat(n, rounding), genMat(n, rounding)
	worst, tol := accelerateWorstDelta(n, a, b, true)
	if worst <= tol {
		t.Errorf("positive control FAILED: computing A.B^T instead of A.B gave worst |delta| "+
			"= %g, within tolerance %g — the comparison cannot detect a transposed operand, "+
			"so it cannot certify the untransposed one either", worst, tol)
	} else {
		t.Logf("keel-accelerate-control: A.B^T rejected, worst |delta| %g against tolerance %g "+
			"— the check discriminates", worst, tol)
	}
}

// genMat builds an n x n matrix from an index function.
func genMat(n int, f func(int) float32) []float32 {
	v := make([]float32, n*n)
	for i := range v {
		v[i] = f(i)
	}
	return v
}

// accelerateWorstDelta multiplies through Accelerate and through the float64
// oracle and returns the worst absolute disagreement with the tolerance for the
// shape. transposeB computes A.B^T from the reference only, for the control.
func accelerateWorstDelta(n int, a, b []float32, transposeB bool) (worst float64, tol float64) {
	got := make([]float32, n*n)
	accelerateSgemm(n, n, n, 1, a, n, b, n, 0, got, n)

	want := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for p := 0; p < n; p++ {
			av := float64(a[i*n+p])
			for j := 0; j < n; j++ {
				bv := b[p*n+j]
				if transposeB {
					bv = b[j*n+p]
				}
				want[i*n+j] += av * float64(bv)
			}
		}
	}
	for i := range want {
		if d := math.Abs(float64(got[i]) - want[i]); d > worst {
			worst = d
		}
	}
	return worst, float64(oracle.Tolerance(n, 1))
}

// BenchmarkAccelerate measures Apple's Accelerate sgemm beside keel's own rate.
//
// # The caption, which is the point of this harness
//
// This rate is NOT on keel's NEON roofline. Apple's path may dispatch to the
// undocumented AMX/AME coprocessor, which keel cannot target from pure Go. So a
// keel/Accelerate ratio and a keel percent-of-NEON-peak have **different
// denominators**, and the first is not a measure of how good keel's NEON code is
// — it is a measure of what a reader's actual alternative achieves on hardware
// keel structurally cannot reach. Both are worth publishing and they answer
// different questions (#175, §5 rule 16).
//
// # Threading, which is measured because it cannot be read back
//
// Accelerate exposes no `openblas_get_num_threads()` equivalent, so the capped
// and uncapped arms are both measured and the delta between them is the witness
// that `VECLIB_MAXIMUM_THREADS` arrived (§5 rule 26). The capped arm is the one
// comparable to keel's single-thread rate; the uncapped arm is reported because
// it is what an unconfigured caller gets, which is the honest thing to tell a
// reader choosing between the two libraries.
func BenchmarkAccelerate(b *testing.B) {
	provenance()
	b.Logf("keel-accelerate-caption: NOT on keel's NEON roofline — Apple's path may " +
		"dispatch to AMX/AME, which keel cannot target from Go, so a keel/Accelerate " +
		"ratio and a keel %%-of-NEON-peak are different quantities (#175)")
	b.Logf("keel-accelerate-threads: VECLIB_MAXIMUM_THREADS=%q at process start; "+
		"Accelerate has no read-back, so the capped/uncapped delta below IS the witness",
		os.Getenv("VECLIB_MAXIMUM_THREADS"))

	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			a, bm, c := makeMat(n, n), makeMat(n, n), makeMat(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				accelerateSgemm(n, n, n, 1, a, n, bm, n, 0, c, n)
			}
			// gemmWork, the same work function BenchmarkSgemm and BenchmarkOpenBLAS
			// use, so the three rates are the same quantity and a ratio between them
			// is a ratio rather than two independent claims.
			rateWork(b, gemmWork(n))
		})
	}
}
