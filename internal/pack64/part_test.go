// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package pack64

import (
	"fmt"
	"math"
	"testing"
)

// This file is the float64 twin of internal/pack's I2 checks
// (docs/generic-middle.md): TestBPanelsPartIsTheSerialPack and
// TestBPanelsPartRejectsARangeOutsideTheBlock. tools/f64gen carries no tests, and
// TestPack64IsPack compares only the serial packs across types, so without this
// nothing held the generated BPanelsPart — the range form internal/block64's
// parallel B-pack calls — to the generated serial pack. Hand-written, mirroring the
// float32 tests shape for shape.

// poison fills a buffer with a value no seq() element can equal, so an unwritten
// slot is a failure rather than a coincidence. Same value as internal/pack's.
func poison(n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = -7.5e-3
	}
	return v
}

// seq is internal/pack's seq at float64: every element distinct and exact, padded
// past the last row with poison so a stride bug reads it.
func seq(rows, cols, ld int) []float64 {
	v := poison(rows*ld + cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v[i*ld+j] = float64(i*cols+j+1) * 0.25
		}
	}
	return v
}

// TestBPanelsPartIsTheSerialPack: a B pack split into panel ranges must be bit-
// identical to the serial pack, for every contiguous partition a claim loop could
// produce, including the ragged last panel where a range boundary could drop the
// zero fill. The output buffer is poisoned, not zeroed, because zero is a value the
// padding legitimately holds. internal/block64's TestF64NestParallelIsSerial is the
// check that runs it concurrently.
func TestBPanelsPartIsTheSerialPack(t *testing.T) {
	const ldb = 40
	b := seq(ldb, ldb, ldb)
	shapes := []struct{ cols, kc, blk int }{
		{16, 8, 4},  // exact
		{9, 8, 4},   // ragged last panel: 3 panels, one of them 1/4 valid
		{1, 1, 4},   // one element, all padding
		{17, 3, 4},  // ragged, shallow
		{12, 5, 32}, // a single panel narrower than blk (internal/pack's shape, kept exact)
	}
	for _, s := range shapes {
		for _, trans := range []bool{false, true} {
			name := fmt.Sprintf("cols=%d/kc=%d/blk=%d/trans=%v", s.cols, s.kc, s.blk, trans)
			t.Run(name, func(t *testing.T) {
				npan := NPanels(s.blk, s.cols)
				want := poison(BLen(s.blk, s.cols, s.kc))
				BPanels(want, s.blk, b, ldb, trans, 0, s.kc, 0, s.cols)

				cuts := [][]int{{0, npan}}
				for i := 0; i <= npan; i++ {
					cuts = append(cuts, []int{0, i, npan})
				}
				single := []int{}
				for i := 0; i <= npan; i++ {
					single = append(single, i)
				}
				cuts = append(cuts, single)

				for _, cut := range cuts {
					got := poison(len(want))
					for i := 0; i+1 < len(cut); i++ {
						BPanelsPart(got, s.blk, b, ldb, trans, 0, s.kc, 0, s.cols, cut[i], cut[i+1])
					}
					for i := range want {
						if math.Float64bits(got[i]) != math.Float64bits(want[i]) {
							t.Fatalf("partition %v: slot %d is %v, serial pack wrote %v",
								cut, i, got[i], want[i])
						}
					}
				}
			})
		}
	}
}

// TestBPanelsPartRejectsARangeOutsideTheBlock: an off-by-one in a partition must
// panic rather than leave a short pack holding the pooled buffer's last contents.
func TestBPanelsPartRejectsARangeOutsideTheBlock(t *testing.T) {
	const ldb = 16
	b := seq(ldb, ldb, ldb)
	dst := make([]float64, BLen(4, 8, 4))
	for _, r := range []struct{ from, to int }{{-1, 2}, {0, 3}, {2, 4}} {
		t.Run(fmt.Sprintf("[%d,%d)", r.from, r.to), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("range [%d,%d) outside [0,2) did not panic", r.from, r.to)
				}
			}()
			BPanelsPart(dst, 4, b, ldb, false, 0, 4, 0, 8, r.from, r.to)
		})
	}
}
