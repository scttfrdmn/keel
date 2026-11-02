// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"fmt"
	"math"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// TestMeasuredKernelsGemmAgainstOracle drives every measured kernel through the whole
// blocked Gemm and checks it against the float64 definition-order oracle. It exists
// for the experimental wide tiles (branch exp-wide-tiles): TestNestBlocksDriveTheSameGemm
// compares the nest with its own block replay, kernel against kernel, so a wrong kernel
// agrees with itself there. The blocks are shrunk so every remainder path runs.
func TestMeasuredKernelsGemmAgainstOracle(t *testing.T) {
	defer func(kc, mc, nc int) { KC, MC, NC = kc, mc, nc }(KC, MC, NC)
	KC, MC, NC = 40, 24, 128

	shapes := []struct{ m, n, k int }{
		{48, 192, 72},  // several whole blocks
		{50, 200, 73},  // remainders in all three
		{3, 5, 7},      // smaller than one tile
		{17, 256, 100}, // tall-thin C against a wide B panel
		{129, 65, 90},  // one column past a 64-wide tile
	}
	for _, kn := range kern.Measured() {
		for _, s := range shapes {
			t.Run(fmt.Sprintf("%s/m=%d,n=%d,k=%d", kn.ID(), s.m, s.n, s.k), func(t *testing.T) {
				const alpha, beta = 1.5, -0.75
				a, b, c0 := seq(s.m*s.k, 3), seq(s.k*s.n, 5), seq(s.m*s.n, 7)
				got := append([]float32(nil), c0...)
				Gemm(kn, false, false, s.m, s.n, s.k, alpha, a, s.k, b, s.n, beta, got, s.n)
				want, scale := oracle.Gemm(false, false, s.m, s.n, s.k, alpha, a, s.k, b, s.n, beta, c0, s.n)
				for i := range want {
					if d := math.Abs(float64(got[i]) - want[i]); d > oracle.Tolerance(s.k+1, scale[i]) {
						t.Fatalf("C[%d] = %v, oracle %v (|diff| %g > tolerance %g)", i, got[i], want[i], d, oracle.Tolerance(s.k+1, scale[i]))
					}
				}
			})
		}
	}
}
