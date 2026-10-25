// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package pack

import (
	"fmt"
	"math"
	"testing"
)

// depthPanelByRow is the depth-contiguous arm of panels exactly as it was before
// #186 interchanged it: one source row at a time, so the reads are contiguous and
// the writes stride by blk. Kept as the reference the shipped depthPanel must
// match bit for bit — each slot is still alpha*v computed once, so any difference
// at all is a wrong index, not rounding.
func depthPanelByRow(panel []float32, blk int, alpha float32, src []float32, ld, base, valid, p0, kc int) {
	for x := 0; x < valid; x++ {
		row := src[(base+x)*ld+p0 : (base+x)*ld+p0+kc]
		for p, v := range row {
			panel[p*blk+x] = alpha * v
		}
	}
	for x := valid; x < blk; x++ {
		for p := 0; p < kc; p++ {
			panel[p*blk+x] = 0
		}
	}
}

// TestDepthPanelIsTheRowLoop drives depthPanel and depthPanelByRow over every
// valid/blk pair up to 9 rows and the shipped NR values beyond, into poisoned
// buffers, from a source carrying -0, ±Inf, NaN and denormals, at a nonzero
// base and p0 — and requires the bits to agree in every slot, padding included.
// Bits, not ==: sameBits's reason, and a -0 that became +0 through a reordered
// multiply would compare equal.
func TestDepthPanelIsTheRowLoop(t *testing.T) {
	const ld, base, p0 = 300, 3, 5
	special := []float32{
		float32(math.Copysign(0, -1)), 0,
		float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN()),
		math.SmallestNonzeroFloat32, -math.MaxFloat32,
	}
	src := seq(base+64, ld, ld)
	for i, v := range special {
		for r := 0; r+i < base+64; r += len(special) {
			src[(r+i)*ld+p0+(i*11+r)%(ld-p0)] = v
		}
	}
	blks := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 16, 24, 32, 64}
	for _, blk := range blks {
		for valid := 1; valid <= blk; valid++ {
			if blk > 9 && valid > 9 && valid < blk-4 {
				continue // the 4/3/2/1 row groups are covered by the ends
			}
			for _, kc := range []int{1, 2, 7, 64, 257} {
				for _, alpha := range []float32{1, -1, -0.5, 3} {
					name := fmt.Sprintf("blk=%d/valid=%d/kc=%d/alpha=%v", blk, valid, kc, alpha)
					got, want := poison(blk*kc), poison(blk*kc)
					depthPanel(got, blk, alpha, src, ld, base, valid, p0, kc)
					depthPanelByRow(want, blk, alpha, src, ld, base, valid, p0, kc)
					for i := range want {
						if g, w := math.Float32bits(got[i]), math.Float32bits(want[i]); g != w {
							t.Fatalf("%s: slot %d is %v (%#08x), the row loop wrote %v (%#08x)",
								name, i, got[i], g, want[i], w)
						}
					}
				}
			}
		}
	}
}

func BenchmarkDepthPanel(b *testing.B) {
	const ld, rows, kc = 2048, 144, 256
	src := make([]float32, (rows+64)*ld)
	for i := range src {
		src[i] = float32(i%13) * 0.125
	}
	for _, blk := range []int{2, 3, 4, 8, 16, 32} {
		panel := make([]float32, blk*kc)
		npan := (rows + blk - 1) / blk
		for _, arm := range []struct {
			name string
			fn   func([]float32, int, float32, []float32, int, int, int, int, int)
		}{{"by-row", depthPanelByRow}, {"by-depth", depthPanel}} {
			b.Run(fmt.Sprintf("blk=%d/%s", blk, arm.name), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for ib := 0; ib < npan; ib++ {
						valid := min(blk, rows-ib*blk)
						arm.fn(panel, blk, -1, src, ld, ib*blk, valid, 0, kc)
					}
				}
				b.ReportMetric(float64(rows*kc)*float64(b.N)/b.Elapsed().Seconds()/1e9, "Gelem/s")
			})
		}
	}
}
