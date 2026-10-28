// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package pack

import "testing"

// TestBroadcast pins the layout ScalarTileLanes reads: element i at [i*lanes,
// (i+1)*lanes), every copy equal, nothing written past len(src)*lanes, and a short
// dst refused.
func TestBroadcast(t *testing.T) {
	src := []float32{1, -2, 3.5, 0}
	for _, lanes := range []int{1, 2, 4} {
		dst := make([]float32, len(src)*lanes+3)
		for i := range dst {
			dst[i] = 7
		}
		Broadcast(dst, src, lanes)
		for i, v := range src {
			for l := 0; l < lanes; l++ {
				if got := dst[i*lanes+l]; got != v {
					t.Fatalf("lanes %d: dst[%d] = %v, want %v", lanes, i*lanes+l, got, v)
				}
			}
		}
		for i := len(src) * lanes; i < len(dst); i++ {
			if dst[i] != 7 {
				t.Fatalf("lanes %d: wrote dst[%d] past the expansion", lanes, i)
			}
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("Broadcast accepted a dst too short for the expansion")
		}
	}()
	Broadcast(make([]float32, 7), src, 2)
}
