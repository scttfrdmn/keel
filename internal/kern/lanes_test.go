// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern

import (
	"math"
	"math/rand"
	"testing"
)

// TestScalarTileLanesMatchesScalarTile holds the pre-broadcast scalar twins to the
// conventional ones bit for bit: the same A, packed both ways, through the same loop
// order, must give the same C. The copies are filled with a different value from the
// lane read, so a twin reading the wrong copy disagrees.
func TestScalarTileLanesMatchesScalarTile(t *testing.T) {
	for _, sh := range [][2]int{{5, 16}, {4, 20}, {7, 12}, {5, 8}, {1, 1}} {
		mr, nr := sh[0], sh[1]
		for _, lanes := range []int{2, 4} {
			for _, kc := range []int{0, 1, 3, 17} {
				rng := rand.New(rand.NewSource(int64(kc*31 + lanes)))
				a, b := make([]float64, kc*mr), make([]float64, kc*nr)
				c := make([]float64, mr*(nr+2))
				for _, s := range [][]float64{a, b, c} {
					for i := range s {
						s[i] = 2*rng.Float64() - 1
					}
				}
				pb := make([]float64, len(a)*lanes)
				for i, v := range a {
					pb[i*lanes] = v
					for l := 1; l < lanes; l++ {
						pb[i*lanes+l] = 99 // never read
					}
				}
				want, got := append([]float64(nil), c...), append([]float64(nil), c...)
				ScalarTile64(mr, nr, kc, a, b, want, nr+2)
				ScalarTileLanes64(mr, nr, lanes, kc, pb, b, got, nr+2)
				for i := range want {
					if math.Float64bits(got[i]) != math.Float64bits(want[i]) {
						t.Fatalf("f64 %dx%d lanes %d kc %d: c[%d] = %v, want %v", mr, nr, lanes, kc, i, got[i], want[i])
					}
				}
				a32, b32, pb32 := to32s(a), to32s(b), to32s(pb)
				w32, g32 := to32s(c), to32s(c)
				ScalarTile(mr, nr, kc, a32, b32, w32, nr+2)
				ScalarTileLanes(mr, nr, lanes, kc, pb32, b32, g32, nr+2)
				for i := range w32 {
					if math.Float32bits(g32[i]) != math.Float32bits(w32[i]) {
						t.Fatalf("f32 %dx%d lanes %d kc %d: c[%d] = %v, want %v", mr, nr, lanes, kc, i, g32[i], w32[i])
					}
				}
			}
		}
	}
	if k := (Kernel{MR: 5, NR: 16, ALanes: 4}).Ref(); k.ALanes != 4 || (Kernel{MR: 5, NR: 16}).Ref().ALanes != 0 {
		t.Error("Ref does not carry the A-panel contract")
	}
	if k := (Kernel64{MR: 5, NR: 8, ALanes: 2}).Ref(); k.ALanes != 2 {
		t.Error("Ref64 does not carry the A-panel contract")
	}
	if a, b := ScalarKernel(5, 16).ID(), ScalarKernelLanes(5, 16, 4).ID(); a == b {
		t.Errorf("conventional and pre-broadcast 5x16 twins share the ID %q", a)
	}
	if a, b := ScalarKernel64(5, 8).ID(), ScalarKernelLanes64(5, 8, 2).ID(); a == b {
		t.Errorf("conventional and pre-broadcast 5x8 float64 twins share the ID %q", a)
	}
}

func to32s(s []float64) []float32 {
	o := make([]float32, len(s))
	for i, v := range s {
		o[i] = float32(v)
	}
	return o
}
