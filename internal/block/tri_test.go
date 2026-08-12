// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import "testing"

// The three range predicates against the element-wise definition they are derived
// from (DESIGN.md §4/P4).
//
// whole, none and rowRange are the reason a masked GEMM costs nothing extra: they
// decide per tile and per row, without a test per element. That also makes them the
// kind of code where an off-by-one is invisible in the routine's output at most
// sizes and wrong at one — a tile that straddles the diagonal by a single element
// is rare enough that a size sweep can miss it and a mask that is one column wide
// at the wrong end still produces plausible numbers. So they are checked here
// directly, over every rectangle in a small square, against keeps().
//
// keeps() is the definition and has no other caller; this is what keeps it honest.

func TestTriMaskWhole(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for i0 := 0; i0 < d; i0++ {
			for j0 := 0; j0 < d; j0++ {
				for h := 1; h <= d-i0; h++ {
					for w := 1; w <= d-j0; w++ {
						all := true
						for i := i0; i < i0+h; i++ {
							for j := j0; j < j0+w; j++ {
								if !m.keeps(i, j) {
									all = false
								}
							}
						}
						if got := m.whole(i0, j0, h, w); got != all {
							t.Fatalf("lower=%v whole(%d,%d,%d,%d) = %v, want %v",
								lower, i0, j0, h, w, got, all)
						}
					}
				}
			}
		}
	}
}

func TestTriMaskNone(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for i0 := 0; i0 < d; i0++ {
			for j0 := 0; j0 < d; j0++ {
				for h := 1; h <= d-i0; h++ {
					for w := 1; w <= d-j0; w++ {
						any := false
						for i := i0; i < i0+h; i++ {
							for j := j0; j < j0+w; j++ {
								if m.keeps(i, j) {
									any = true
								}
							}
						}
						if got := m.none(i0, j0, h, w); got != !any {
							t.Fatalf("lower=%v none(%d,%d,%d,%d) = %v, want %v",
								lower, i0, j0, h, w, got, !any)
						}
					}
				}
			}
		}
	}
}

// TestTriMaskRowRange checks both halves of rowRange's contract: that the range it
// returns is exactly the kept elements of that row, and that it is a *range* at all
// — no kept element outside it, which is what makes a masked row a shorter row
// rather than a row with holes.
func TestTriMaskRowRange(t *testing.T) {
	const d = 9
	for _, lower := range []bool{false, true} {
		m := triMask{on: true, lower: lower}
		for gi := 0; gi < d; gi++ {
			for j0 := 0; j0 < d; j0++ {
				for w := 1; w <= d-j0; w++ {
					lo, hi := m.rowRange(gi, j0, w)
					if lo < 0 || hi > w || lo > hi {
						t.Fatalf("lower=%v rowRange(%d,%d,%d) = %d,%d: not a valid range in [0,%d]",
							lower, gi, j0, w, lo, hi, w)
					}
					for j := 0; j < w; j++ {
						in := j >= lo && j < hi
						if want := m.keeps(gi, j0+j); in != want {
							t.Fatalf("lower=%v rowRange(%d,%d,%d) = %d,%d: local column %d in=%v, keeps=%v",
								lower, gi, j0, w, lo, hi, j, in, want)
						}
					}
				}
			}
		}
	}
}

// TestTriMaskOff pins the pass-through case. Every call plain Gemm makes goes
// through these predicates with on false, so "no mask" has to be exactly no mask:
// every rectangle whole, none empty-only, every row full width.
func TestTriMaskOff(t *testing.T) {
	var m triMask
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if !m.keeps(i, j) {
				t.Fatalf("mask off: keeps(%d,%d) = false", i, j)
			}
		}
	}
	for i0 := 0; i0 < 5; i0++ {
		for j0 := 0; j0 < 5; j0++ {
			for h := 1; h <= 3; h++ {
				for w := 1; w <= 3; w++ {
					if !m.whole(i0, j0, h, w) {
						t.Fatalf("mask off: whole(%d,%d,%d,%d) = false", i0, j0, h, w)
					}
					if m.none(i0, j0, h, w) {
						t.Fatalf("mask off: none(%d,%d,%d,%d) = true", i0, j0, h, w)
					}
					if lo, hi := m.rowRange(i0, j0, w); lo != 0 || hi != w {
						t.Fatalf("mask off: rowRange(%d,%d,%d) = %d,%d, want 0,%d",
							i0, j0, w, lo, hi, w)
					}
				}
			}
		}
	}
	// An empty rectangle is whole and none at once, on either setting. Both callers
	// rely on it: the loop nest asks about blocks it has already clamped to the
	// matrix, and clamping to zero must not turn into a kernel call.
	for _, mm := range []triMask{{}, {on: true}, {on: true, lower: true}} {
		if !mm.whole(0, 0, 0, 4) || !mm.none(0, 0, 0, 4) {
			t.Fatalf("%+v: empty rectangle should be both whole and none", mm)
		}
		if !mm.whole(0, 0, 4, 0) || !mm.none(0, 0, 4, 0) {
			t.Fatalf("%+v: empty rectangle should be both whole and none", mm)
		}
	}
}
