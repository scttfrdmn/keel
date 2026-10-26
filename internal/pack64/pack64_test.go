// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package pack64

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/pack"
)

// TestPack64IsPack holds the generated float64 packing to the float32 original it
// was generated from. With alpha = 1 and small-integer data, every value is exact
// in both types, so the two layouts must agree element for element: the same
// panel order, the same zero padding, the same symmetric reflection. Panel widths
// span both sides of each package's memmoveFloor (16 elements at float32, 8 at
// float64, both 64 bytes), so the copy() branch and the assignment loop are each
// exercised under both types. Hand-written: it is the test of the copy.
func TestPack64IsPack(t *testing.T) {
	r := rand.New(rand.NewSource(64))
	const rows, cols = 37, 41
	m32 := make([]float32, rows*cols)
	for i := range m32 {
		m32[i] = float32(r.Intn(33) - 16)
	}
	m64 := make([]float64, len(m32))
	for i, v := range m32 {
		m64[i] = float64(v)
	}
	for _, blk := range []int{2, 4, 6, 7, 8, 9, 15, 16, 17, 24, 32} {
		for _, trans := range []bool{false, true} {
			for _, sh := range []struct{ off, count, p0, kc int }{{0, 20, 0, 13}, {3, 17, 5, 9}, {0, 1, 0, 1}} {
				what := fmt.Sprintf("blk=%d trans=%v %+v", blk, trans, sh)
				ld := cols
				a32, a64 := make([]float32, pack.ALen(blk, sh.count, sh.kc)), make([]float64, ALen(blk, sh.count, sh.kc))
				pack.APanels(a32, blk, 1, m32, ld, trans, sh.off, sh.count, sh.p0, sh.kc)
				APanels(a64, blk, 1, m64, ld, trans, sh.off, sh.count, sh.p0, sh.kc)
				same(t, "APanels "+what, a32, a64)

				b32, b64 := make([]float32, pack.BLen(blk, sh.count, sh.kc)), make([]float64, BLen(blk, sh.count, sh.kc))
				pack.BPanels(b32, blk, m32, ld, trans, sh.p0, sh.kc, sh.off, sh.count)
				BPanels(b64, blk, m64, ld, trans, sh.p0, sh.kc, sh.off, sh.count)
				same(t, "BPanels "+what, b32, b64)

				lower := trans // reuse the flag for the triangle
				s32, s64 := make([]float32, pack.ALen(blk, sh.count, sh.kc)), make([]float64, ALen(blk, sh.count, sh.kc))
				pack.ASymPanels(s32, blk, 1, m32, ld, lower, sh.off, sh.count, sh.p0, sh.kc)
				ASymPanels(s64, blk, 1, m64, ld, lower, sh.off, sh.count, sh.p0, sh.kc)
				same(t, "ASymPanels "+what, s32, s64)
			}
		}
	}
	if memmoveFloor != 8 {
		t.Errorf("float64 memmoveFloor = %d elements, want 8 (64 bytes)", memmoveFloor)
	}
}

func same(t *testing.T, what string, a []float32, b []float64) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("%s: lengths %d and %d", what, len(a), len(b))
	}
	for i := range a {
		if float64(a[i]) != b[i] {
			t.Fatalf("%s: element %d is %v at float32, %v at float64", what, i, a[i], b[i])
		}
	}
}
