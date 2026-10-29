// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// The float64 copy, swap and rotation suite (#192): l1rot_test.go at float64.
// Dcopy and Dswap are exact, so they are compared bit for bit. Drot is held to
// oracle.RotDD, the double-double arm, at Tolerance64. Drotm is held to H's
// products formed in dd the same way, through RotDD's construction on the pair
// (h11, h12) and (h21, h22). Drotg and Drotmg are judged by their defining
// properties, at Tolerance64 bounds, as their float32 twins are.

func randVals64(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = r.NormFloat64()
	}
	return v
}

func TestDcopyDswap(t *testing.T) {
	r := rand.New(rand.NewSource(59))
	for _, n := range rotSizes {
		for _, ix := range l2Incs {
			for _, iy := range l2Incs {
				xv, yv := randVals64(r, n), randVals64(r, n)
				x, y := spread64(xv, ix), spread64(yv, iy)
				what := fmt.Sprintf("n=%d incX=%d incY=%d", n, ix, iy)

				yc := append([]float64(nil), y...)
				Dcopy(n, x, ix, yc, iy)
				if got := gather64(yc, n, iy); fmt.Sprint(got) != fmt.Sprint(xv) {
					t.Errorf("Dcopy %s: y = %v, want %v", what, got, xv)
				}
				checkGaps64(t, "Dcopy "+what, yc, y, n, iy)

				xs, ys := append([]float64(nil), x...), append([]float64(nil), y...)
				Dswap(n, xs, ix, ys, iy)
				if gx, gy := gather64(xs, n, ix), gather64(ys, n, iy); fmt.Sprint(gx) != fmt.Sprint(yv) || fmt.Sprint(gy) != fmt.Sprint(xv) {
					t.Errorf("Dswap %s: did not exchange", what)
				}
				checkGaps64(t, "Dswap x "+what, xs, x, n, ix)
				checkGaps64(t, "Dswap y "+what, ys, y, n, iy)
			}
		}
	}
}

// rotmRef64 applies H = [h11 h12; h21 h22] to one pair through the dd arm:
// RotDD(x, y, c, s) computes c·x + s·y and c·y − s·x, so the first row is
// RotDD's x' at (c, s) = (h11, h12), and the second row is its x' with the pair
// (h21, h22).
func rotmRef64(x, y, h11, h12, h21, h22 float64) (rx, ry, sx, sy float64) {
	rx, _, sx, _ = oracle.RotDD(x, y, h11, h12)
	ry, _, sy, _ = oracle.RotDD(x, y, h21, h22)
	return
}

func TestDrotDrotm(t *testing.T) {
	r := rand.New(rand.NewSource(61))
	c, s := math.Cos(0.7), math.Sin(0.7)
	rr := float64(RotmRescaling)
	params := []struct {
		name string
		p    [5]float64
		h    [4]float64 // h11, h12, h21, h22
	}{
		{"identity", [5]float64{float64(RotmIdentity), 9, 9, 9, 9}, [4]float64{1, 0, 0, 1}},
		{"rescaling", [5]float64{rr, 1.5, -0.25, 0.75, 2}, [4]float64{1.5, 0.75, -0.25, 2}},
		{"off-diagonal", [5]float64{float64(RotmOffDiagonal), 9, -0.25, 0.75, 9}, [4]float64{1, 0.75, -0.25, 1}},
		{"diagonal", [5]float64{float64(RotmDiagonal), 1.5, 9, 9, 2}, [4]float64{1.5, 1, -1, 2}},
	}
	for _, n := range rotSizes {
		for _, ix := range l2Incs {
			for _, iy := range []int{1, -2} {
				xv, yv := randVals64(r, n), randVals64(r, n)
				x, y := spread64(xv, ix), spread64(yv, iy)
				check := func(name string, gx, gy []float64, h [4]float64) {
					t.Helper()
					ex, ey := gather64(gx, n, ix), gather64(gy, n, iy)
					for i := 0; i < n; i++ {
						rx, ry, sx, sy := rotmRef64(xv[i], yv[i], h[0], h[1], h[2], h[3])
						check64(t, fmt.Sprintf("%s n=%d incX=%d incY=%d x[%d]", name, n, ix, iy, i), 2, ex[i], rx, sx, oracle.Tolerance64)
						check64(t, fmt.Sprintf("%s n=%d incX=%d incY=%d y[%d]", name, n, ix, iy, i), 2, ey[i], ry, sy, oracle.Tolerance64)
					}
					checkGaps64(t, name+" x", gx, x, n, ix)
					checkGaps64(t, name+" y", gy, y, n, iy)
				}
				gx, gy := append([]float64(nil), x...), append([]float64(nil), y...)
				Drot(n, gx, ix, gy, iy, c, s)
				check("Drot", gx, gy, [4]float64{c, s, -s, c})
				for _, p := range params {
					gx, gy := append([]float64(nil), x...), append([]float64(nil), y...)
					Drotm(n, gx, ix, gy, iy, p.p)
					check("Drotm "+p.name, gx, gy, p.h)
				}
			}
		}
	}
}

// TestDrotg is TestSrotg at float64: the rotation zeroes b, c² + s² = 1, and it is
// safe at the ends of float64's range.
func TestDrotg(t *testing.T) {
	vals := []float64{0, 1, -1, 3, -4, 0.5, 1e-300, -1e-300, 1e300, -1e300, 1.5e308, -1.5e308,
		math.SmallestNonzeroFloat64, 1e-310, 2.5, -7}
	for _, a := range vals {
		for _, b := range vals {
			c, s, rr, z := Drotg(a, b)
			what := fmt.Sprintf("Drotg(%v, %v) = c %v, s %v, r %v, z %v", a, b, c, s, rr, z)
			// The true r, without overflow: hypot of the halves, doubled.
			trueR := 2 * math.Hypot(a/2, b/2)
			if math.IsInf(trueR, 0) {
				if !math.IsInf(rr, 0) {
					t.Errorf("%s: |r| should overflow to Inf", what)
				}
				continue
			}
			if math.IsInf(rr, 0) || math.IsNaN(rr) {
				t.Errorf("%s: r is not finite though %.4g is representable", what, trueR)
				continue
			}
			if trueR < 0x1p-1022 {
				// A subnormal r carries few bits; check r alone, to one subnormal step.
				if d := math.Abs(math.Abs(rr) - math.Hypot(a, b)); d > 0x1p-1074 {
					t.Errorf("%s: |r| %.4g is more than one subnormal step from %.4g", what, rr, math.Hypot(a, b))
				}
				continue
			}
			if d := math.Abs(c*c + s*s - 1); d > oracle.Tolerance64(2, 1) {
				t.Errorf("%s: c²+s² is off 1 by %.3g", what, d)
			}
			// [c s; -s c]·[a; b] = [r; 0], judged relative to max(|a|, |b|) so that
			// the products cannot overflow the check itself.
			m := math.Max(math.Abs(a), math.Abs(b))
			fa, fb, fr := a/m, b/m, rr/m
			scale := math.Abs(fa) + math.Abs(fb)
			if d := math.Abs(c*fa + s*fb - fr); d > oracle.Tolerance64(4, scale) {
				t.Errorf("%s: c·a + s·b misses r by %.3g (scaled)", what, d)
			}
			if d := math.Abs(-s*fa + c*fb); d > oracle.Tolerance64(4, scale) {
				t.Errorf("%s: −s·a + c·b is %.3g (scaled), not 0", what, d)
			}
			var zc, zs float64
			switch {
			case z == 1:
				zc, zs = 0, 1
			case math.Abs(z) < 1:
				zc, zs = math.Sqrt(1-z*z), z
			default:
				zc, zs = 1/z, math.Sqrt(1-1/(z*z))
			}
			// Recovering c or s from z goes through sqrt(1 − z²), which near |z| = 1
			// keeps only half the digits, so the bound is the square root of the
			// rounding bound (float32's twin uses 1e-6 for the same reason).
			if b != 0 && a != 0 {
				if d := math.Abs(math.Abs(zc) - math.Abs(c)); d > math.Sqrt(oracle.Tolerance64(4, 1)) {
					t.Errorf("%s: z does not recover |c| (got %v)", what, zc)
				}
				if d := math.Abs(math.Abs(zs) - math.Abs(s)); d > math.Sqrt(oracle.Tolerance64(4, 1)) {
					t.Errorf("%s: z does not recover |s| (got %v)", what, zs)
				}
			}
		}
	}
	if c, s, rr, z := Drotg(2, 0); c != 1 || s != 0 || rr != 2 || z != 0 {
		t.Errorf("Drotg(2, 0) = %v %v %v %v, want 1 0 2 0", c, s, rr, z)
	}
	if c, s, rr, z := Drotg(0, -3); c != 0 || s != 1 || rr != -3 || z != 1 {
		t.Errorf("Drotg(0, -3) = %v %v %v %v, want 0 1 -3 1", c, s, rr, z)
	}
}

// hOf64 expands Drotmg's DPARAM into the full H its flag denotes.
func hOf64(p [5]float64) (h11, h12, h21, h22 float64) {
	switch p[0] {
	case float64(RotmIdentity):
		return 1, 0, 0, 1
	case float64(RotmRescaling):
		return p[1], p[3], p[2], p[4]
	case float64(RotmOffDiagonal):
		return 1, p[3], p[2], 1
	default: // RotmDiagonal
		return p[1], 1, -1, p[4]
	}
}

// TestDrotmg is TestSrotmg at float64: H zeroes y1 in the weighted sense, maps x1
// to rx1, and preserves d1·x1² + d2·y1² = rd1·rx1², with each bound a Tolerance64
// at the quantity's own scale.
func TestDrotmg(t *testing.T) {
	r := rand.New(rand.NewSource(67))
	type in struct{ d1, d2, x1, y1 float64 }
	cases := []in{{1, 1, 3, 4}, {2, 3, -1, 5}, {0, 2, 1, 1}, {1, 2, 0, 3}, {1, 1, 5, 0}, {1, 0, 2, 3}, {1e-10, 1, 1, 1},
		{1e10, 1e-10, 1, 1}, {1, 1e9, 1, 1}, {1e-9, 1e-9, 2, 7}}
	for i := 0; i < 200; i++ {
		cases = append(cases, in{r.ExpFloat64(), r.ExpFloat64(), r.NormFloat64(), r.NormFloat64()})
	}
	for _, c := range cases {
		p, rd1, rd2, rx1 := Drotmg(c.d1, c.d2, c.x1, c.y1)
		what := fmt.Sprintf("Drotmg(%v, %v, %v, %v) = %v, %v, %v, %v", c.d1, c.d2, c.x1, c.y1, p, rd1, rd2, rx1)
		if p[0] == float64(RotmRescaling) && rd1 == 0 && rd2 == 0 && rx1 == 0 {
			continue // the error state; covered below
		}
		h11, h12, h21, h22 := hOf64(p)
		x, y := c.x1, c.y1
		second := h21*x + h22*y
		ss := math.Abs(h21*x) + math.Abs(h22*y)
		if math.Abs(second) > oracle.Tolerance64(16, ss) && rd2 != 0 {
			t.Errorf("%s: H does not zero the second component (h21·x1 + h22·y1 = %.3g)", what, second)
		}
		first := h11*x + h12*y
		if d := math.Abs(first - rx1); d > oracle.Tolerance64(16, math.Abs(h11*x)+math.Abs(h12*y)) {
			t.Errorf("%s: H maps x1 to %.17g, not rx1 (off by %.3g)", what, first, d)
		}
		want := c.d1*x*x + c.d2*y*y
		got := rd1*rx1*rx1 + rd2*second*second
		if d := math.Abs(got - want); d > oracle.Tolerance64(32, math.Abs(c.d1*x*x)+math.Abs(c.d2*y*y)) {
			t.Errorf("%s: weighted norm %.17g, want %.17g", what, got, want)
		}
	}
	if p, a, b, c := Drotmg(-1, 1, 1, 1); p[0] != float64(RotmRescaling) || a != 0 || b != 0 || c != 0 || p[1] != 0 {
		t.Errorf("Drotmg with d1 < 0 = %v %v %v %v, want the zeroed RotmRescaling error state", p, a, b, c)
	}
	if p, a, b, c := Drotmg(2, 3, 4, 0); p[0] != float64(RotmIdentity) || a != 2 || b != 3 || c != 4 {
		t.Errorf("Drotmg with y1 == 0 = %v %v %v %v, want RotmIdentity and the inputs back", p, a, b, c)
	}
}

func TestDrotgSignConvention(t *testing.T) {
	for _, tc := range []struct{ a, b, rSign float64 }{{3, 1, 1}, {-3, 1, -1}, {3, -1, 1}, {1, 3, 1}, {1, -3, -1}, {-1, 3, 1}, {-2, -2, -1}} {
		if _, _, r, _ := Drotg(tc.a, tc.b); math.Copysign(1, r) != tc.rSign {
			t.Errorf("Drotg(%v, %v): r = %v, want sign %+v", tc.a, tc.b, r, tc.rSign)
		}
	}
}

func TestDrotmgFlagChoice(t *testing.T) {
	for _, tc := range []struct{ d1, d2, x1, y1, flag float64 }{
		{1, 1, 3, 1, float64(RotmOffDiagonal)},
		{1, 1, 1, 3, float64(RotmDiagonal)},
		{0, 2, 1, 1, float64(RotmDiagonal)},
		{1, 0, 2, 3, float64(RotmIdentity)},
	} {
		if p, _, _, _ := Drotmg(tc.d1, tc.d2, tc.x1, tc.y1); p[0] != tc.flag {
			t.Errorf("Drotmg(%v, %v, %v, %v): flag %v, want %v", tc.d1, tc.d2, tc.x1, tc.y1, p[0], tc.flag)
		}
	}
}
