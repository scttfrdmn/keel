// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/l2"
)

// l2parCase is one call of a parallel Level-2 routine. run builds its inputs from
// a fixed seed and returns the whole output buffer, stride gaps included, so a
// worker that wrote outside its range shows up as a bit difference too.
type l2parCase struct {
	name string
	work int // elements of A updated, the quantity l2.Grain divides
	run  func() []float32
}

func l2parCases(m, n int, inc int) []l2parCase {
	vec := func(r *rand.Rand, k, inc int) []float32 {
		return randVals(r, 1+(k-1)*abs(inc))
	}
	nan := func(v []float32) []float32 {
		for i := range v {
			v[i] = float32(math.NaN())
		}
		return v
	}
	tri := n * (n + 1) / 2
	sfx := fmt.Sprintf("/m=%d/n=%d/inc=%d", m, n, inc)
	var cs []l2parCase
	for _, beta := range []float32{0, 1, -0.75} {
		cs = append(cs,
			l2parCase{fmt.Sprintf("SgemvN/beta=%g", beta) + sfx, m * n, func() []float32 {
				r := rand.New(rand.NewSource(1))
				a, x, y := randVals(r, m*(n+3)), vec(r, n, inc), vec(r, m, -inc)
				if beta == 0 {
					nan(y) // beta == 0 must not read y, at any width
				}
				Sgemv(NoTrans, m, n, 1.25, a, n+3, x, inc, beta, y, -inc)
				return y
			}},
			l2parCase{fmt.Sprintf("SgemvT/beta=%g", beta) + sfx, m * n, func() []float32 {
				r := rand.New(rand.NewSource(2))
				a, x, y := randVals(r, m*(n+3)), vec(r, m, inc), vec(r, n, -inc)
				if beta == 0 {
					nan(y)
				}
				Sgemv(Trans, m, n, 1.25, a, n+3, x, inc, beta, y, -inc)
				return y
			}})
	}
	cs = append(cs,
		l2parCase{"Sger" + sfx, m * n, func() []float32 {
			r := rand.New(rand.NewSource(3))
			a, x, y := randVals(r, m*(n+3)), vec(r, m, inc), vec(r, n, -inc)
			Sger(m, n, 0.5, x, inc, y, -inc, a, n+3)
			return a
		}})
	for _, ul := range []Uplo{Upper, Lower} {
		cs = append(cs,
			l2parCase{fmt.Sprintf("Ssyr/%v", ul) + sfx, tri, func() []float32 {
				r := rand.New(rand.NewSource(4))
				a, x := randVals(r, n*(n+1)), vec(r, n, inc)
				Ssyr(ul, n, 0.5, x, inc, a, n+1)
				return a
			}},
			l2parCase{fmt.Sprintf("Ssyr2/%v", ul) + sfx, 2 * tri, func() []float32 {
				r := rand.New(rand.NewSource(5))
				a, x, y := randVals(r, n*(n+1)), vec(r, n, inc), vec(r, n, -inc)
				Ssyr2(ul, n, 0.5, x, inc, y, -inc, a, n+1)
				return a
			}})
		for _, d := range []Diag{NonUnit, Unit} {
			cs = append(cs, l2parCase{fmt.Sprintf("StrmvN/%v/%v", ul, d) + sfx, tri, func() []float32 {
				r := rand.New(rand.NewSource(6))
				a, x := randVals(r, n*(n+1)), vec(r, n, inc)
				Strmv(ul, NoTrans, d, n, a, n+1, x, inc)
				return x
			}}, l2parCase{fmt.Sprintf("StrmvT/%v/%v", ul, d) + sfx, tri, func() []float32 {
				r := rand.New(rand.NewSource(7))
				a, x := randVals(r, n*(n+1)), vec(r, n, inc)
				Strmv(ul, Trans, d, n, a, n+1, x, inc)
				return x
			}})
		}
	}
	return cs
}

func abs(i int) int { return max(i, -i) }

// l2parProcs are the widths compared against GOMAXPROCS=1: powers of two, and odd
// widths that leave a remainder in every split here. GOMAXPROCS may exceed the
// core count, so 16 partitions 16 ways on any host.
var l2parProcs = []int{2, 3, 5, 8, 16}

// TestL2ParDeterminism is TestP5Determinism's check for the parallel Level-2
// routines: the same bits at every GOMAXPROCS, with the width each call actually
// used asserted beside them, because a call that ran on one worker agrees with the
// serial one for the least interesting reason there is.
//
// The shapes are odd on both axes, so no row or column split is a whole number of
// vectors, and every stride is non-unit with x and y running in opposite
// directions. The expected width is procs itself, never a function of the shape:
// each shape is required to be wide enough first, and a fixture that is not fails
// as a fixture rather than silently expecting less.
func TestL2ParDeterminism(t *testing.T) {
	for _, sh := range []struct{ m, n, inc int }{{2053, 2051, 1}, {2051, 2053, -3}, {1031, 2053, 2}} {
		for _, c := range l2parCases(sh.m, sh.n, sh.inc) {
			if c.work < l2parProcs[len(l2parProcs)-1]*l2.Grain || sh.n < 16*l2.Cols {
				t.Fatalf("%s: fixture too small to need %d workers at l2.Grain=%d", c.name, l2parProcs[len(l2parProcs)-1], l2.Grain)
			}
			var serial []float32
			withProcs(1, func() { serial = c.run() })
			for _, procs := range l2parProcs {
				var got []float32
				var workers int
				withProcs(procs, func() {
					got = c.run()
					workers = WorkersLastCall()
				})
				if workers != procs {
					t.Errorf("%s: GOMAXPROCS=%d ran on %d workers", c.name, procs, workers)
				}
				if i, diff := firstBitDiff(serial, got); diff {
					t.Errorf("%s: GOMAXPROCS=%d differs from 1 at [%d]: %v vs %v", c.name, procs, i, got[i], serial[i])
				}
				if math.IsNaN(float64(got[0])) || math.IsNaN(float64(got[len(got)-1])) {
					t.Errorf("%s: GOMAXPROCS=%d: NaN at an end of the output", c.name, procs)
				}
			}
		}
	}
}

// TestL2ParThreshold pins the threshold's form: below 2·l2.Grain elements a call is
// serial at any GOMAXPROCS, and above it the width is the work over l2.Grain, not
// a jump to GOMAXPROCS.
func TestL2ParThreshold(t *testing.T) {
	small := l2parCases(37, 41, -2)
	// 64 rows of just over 3·l2.Grain/64 columns: three grains of work, room for 64.
	mid := l2parCases(64, 3*l2.Grain/64+1, 1)
	for _, tc := range []struct {
		cases []l2parCase
		want  int
	}{{small, 1}, {mid[:6], 3}} { // mid[:6]: the two Sgemv shapes at each beta
		for _, c := range tc.cases {
			var serial, got []float32
			withProcs(1, func() { serial = c.run() })
			var workers int
			withProcs(8, func() {
				got = c.run()
				workers = WorkersLastCall()
			})
			if workers != tc.want {
				t.Errorf("%s: GOMAXPROCS=8 ran on %d workers, want %d", c.name, workers, tc.want)
			}
			if i, diff := firstBitDiff(serial, got); diff {
				t.Errorf("%s: GOMAXPROCS=8 differs from 1 at [%d]", c.name, i)
			}
		}
	}
}

// TestL2ParAlphaZero: alpha == 0 returns before any split, so it is checked for
// its bits and for not reading what it must not, at a width that would otherwise
// be parallel. Sgemv applies beta == 0 to a NaN y and must leave zeros.
func TestL2ParAlphaZero(t *testing.T) {
	const m, n = 1031, 1037
	r := rand.New(rand.NewSource(7))
	a := randVals(r, m*n)
	x := []float32{float32(math.NaN())} // alpha == 0 must not read x, so this NaN must not surface
	x = append(x, randVals(r, max(m, n))...)
	for _, tA := range []Transpose{NoTrans, Trans} {
		rows := m
		if tA == Trans {
			rows = n
		}
		y := make([]float32, rows)
		for i := range y {
			y[i] = float32(math.NaN())
		}
		withProcs(8, func() { Sgemv(tA, m, n, 0, a, n, x, 1, 0, y, 1) })
		for i, v := range y {
			if v != 0 || math.Signbit(float64(v)) {
				t.Fatalf("Sgemv %v alpha=0 beta=0: y[%d] = %v, want +0", tA, i, v)
			}
		}
	}
	before := append([]float32(nil), a...)
	withProcs(8, func() { Sger(m, n, 0, x, 1, x, 1, a, n) })
	if i, diff := firstBitDiff(before, a); diff {
		t.Fatalf("Sger alpha=0 changed A[%d]", i)
	}
}
