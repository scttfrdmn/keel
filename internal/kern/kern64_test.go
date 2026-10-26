// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern

import (
	"math/rand"
	"testing"
)

// TestScalarTile64MatchesScalarTile holds the float64 reference tile to the
// float32 one. Small integers make every product and partial sum exact in both
// types (|value| < 2^24), so the two must agree bit for bit; any divergence is a
// change made to one copy and not the other, not rounding.
func TestScalarTile64MatchesScalarTile(t *testing.T) {
	r := rand.New(rand.NewSource(64))
	for _, s := range []struct{ mr, nr, kc int }{{1, 1, 1}, {4, 16, 24}, {3, 5, 7}, {MaxMR, MaxNR, 9}, {2, 32, 0}} {
		ldc := s.nr + 3
		a32, b32 := make([]float32, s.kc*s.mr), make([]float32, s.kc*s.nr)
		c32 := make([]float32, s.mr*ldc)
		for i := range a32 {
			a32[i] = float32(r.Intn(17) - 8)
		}
		for i := range b32 {
			b32[i] = float32(r.Intn(17) - 8)
		}
		for i := range c32 {
			c32[i] = float32(r.Intn(201) - 100)
		}
		a64, b64, c64 := widen(a32), widen(b32), widen(c32)
		ScalarTile(s.mr, s.nr, s.kc, a32, b32, c32, ldc)
		ScalarTile64(s.mr, s.nr, s.kc, a64, b64, c64, ldc)
		for i := range c32 {
			if float64(c32[i]) != c64[i] {
				t.Fatalf("%dx%d kc=%d: c[%d] float32 tile %v, float64 tile %v", s.mr, s.nr, s.kc, i, c32[i], c64[i])
			}
		}
	}
	if got := ScalarKernel64(4, 16).ID(); got != ScalarKernel(4, 16).ID() {
		t.Errorf("ScalarKernel64 ID %q, ScalarKernel ID %q", got, ScalarKernel(4, 16).ID())
	}
}

func widen(v []float32) []float64 {
	w := make([]float64, len(v))
	for i, x := range v {
		w[i] = float64(x)
	}
	return w
}
