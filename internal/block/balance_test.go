// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"runtime"
	"testing"
)

// TestBalanceMC pins #206's rule against hand-computed cases. The round efficiency
// (m/b)/(w*rounds) of each expected block is in its comment.
func TestBalanceMC(t *testing.T) {
	for _, c := range []struct{ m, mc, mr, w, want int }{
		{4096, 70, 5, 8, 65},   // 59 units, 0.914 -> 64 units, 0.985 (40 rows ties it; the larger block wins)
		{4096, 144, 4, 8, 128}, // amd64 4x32: 29 units, 0.889 -> 32 units, 1.000
		{4096, 144, 2, 8, 128}, // amd64 2x32: the same; 130 rows reads 0.985, under the 1-point band
		{4100, 144, 4, 8, 104}, // 0.889 -> 0.985; 132 rows reads 0.971, outside the band
		{4096, 70, 5, 1, 70},   // one worker: the serial path never changes
		{4096, 128, 4, 8, 128}, // already 1.000
		{300, 70, 5, 8, 70},    // 5 units < 8 workers: parallelism, not a tail, so left alone
		{4096, 70, 5, 64, 70},  // 59 units < 64 workers: left alone
	} {
		got := balanceMC(c.m, c.mc, c.mr, c.w)
		if got != c.want {
			t.Errorf("balanceMC(m=%d, mc=%d, mr=%d, w=%d) = %d, want %d", c.m, c.mc, c.mr, c.w, got, c.want)
		}
		if got%c.mr != 0 || got > c.mc || got < c.mc/2 {
			t.Errorf("balanceMC(m=%d, mc=%d, mr=%d, w=%d) = %d: not whole tiles in [mc/2, mc]", c.m, c.mc, c.mr, c.w, got)
		}
	}
}

// TestCallMCScope pins the scope (archive/balanced-ic/): only a split call with no triangular
// mask is re-blocked. Syrk/Syr2k (masked) and Trsm's rank updates (noSplit) keep the shipped MC.
func TestCallMCScope(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(8))
	for _, c := range []struct {
		name string
		tri  triMask
		sp   split
		want int
	}{
		{"rectangular, split", triMask{}, splitIC, 65},
		{"masked, split", triMask{on: true, lower: true}, splitIC, 70},
		{"rectangular, noSplit", triMask{}, noSplit, 70},
	} {
		if got := callMC(4096, 70, 5, c.tri, c.sp); got != c.want {
			t.Errorf("%s: callMC(4096, 70, 5) = %d, want %d", c.name, got, c.want)
		}
	}
}
