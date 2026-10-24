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

var rotSizes = []int{0, 1, 2, 7, 64, 257}

func TestScopySswap(t *testing.T) {
	r := rand.New(rand.NewSource(59))
	for _, n := range rotSizes {
		for _, ix := range l2Incs {
			for _, iy := range l2Incs {
				xv, yv := randVals(r, n), randVals(r, n)
				x, y := spread(xv, ix), spread(yv, iy)
				what := fmt.Sprintf("n=%d incX=%d incY=%d", n, ix, iy)

				yc := append([]float32(nil), y...)
				Scopy(n, x, ix, yc, iy)
				if got := elems(yc, n, iy); fmt.Sprint(got) != fmt.Sprint(xv) {
					t.Errorf("Scopy %s: y = %v, want %v", what, got, xv)
				}
				checkGaps(t, "Scopy "+what, yc, y, n, iy)

				xs, ys := append([]float32(nil), x...), append([]float32(nil), y...)
				Sswap(n, xs, ix, ys, iy)
				if gx, gy := elems(xs, n, ix), elems(ys, n, iy); fmt.Sprint(gx) != fmt.Sprint(yv) || fmt.Sprint(gy) != fmt.Sprint(xv) {
					t.Errorf("Sswap %s: did not exchange", what)
				}
				checkGaps(t, "Sswap x "+what, xs, x, n, ix)
				checkGaps(t, "Sswap y "+what, ys, y, n, iy)
			}
		}
	}
}

// rotRef is the float64 reference for one rotated pair, with its error scale.
func rotRef(x, y, h11, h12, h21, h22 float32) (rx, ry, sx, sy float64) {
	a, b := float64(x), float64(y)
	rx = float64(h11)*a + float64(h12)*b
	ry = float64(h21)*a + float64(h22)*b
	sx = math.Abs(float64(h11)*a) + math.Abs(float64(h12)*b)
	sy = math.Abs(float64(h21)*a) + math.Abs(float64(h22)*b)
	return
}

func TestSrotSrotm(t *testing.T) {
	r := rand.New(rand.NewSource(61))
	c, s := float32(math.Cos(0.7)), float32(math.Sin(0.7))
	params := []struct {
		name string
		p    [5]float32
		h    [4]float32 // the matrix the flag denotes: h11, h12, h21, h22
	}{
		{"identity", [5]float32{RotmIdentity, 9, 9, 9, 9}, [4]float32{1, 0, 0, 1}},
		{"rescaling", [5]float32{RotmRescaling, 1.5, -0.25, 0.75, 2}, [4]float32{1.5, 0.75, -0.25, 2}},
		{"off-diagonal", [5]float32{RotmOffDiagonal, 9, -0.25, 0.75, 9}, [4]float32{1, 0.75, -0.25, 1}},
		{"diagonal", [5]float32{RotmDiagonal, 1.5, 9, 9, 2}, [4]float32{1.5, 1, -1, 2}},
	}
	for _, n := range rotSizes {
		for _, ix := range l2Incs {
			for _, iy := range []int{1, -2} {
				xv, yv := randVals(r, n), randVals(r, n)
				x, y := spread(xv, ix), spread(yv, iy)
				check := func(name string, gx, gy []float32, h [4]float32) {
					t.Helper()
					ex, ey := elems(gx, n, ix), elems(gy, n, iy)
					for i := 0; i < n; i++ {
						rx, ry, sx, sy := rotRef(xv[i], yv[i], h[0], h[1], h[2], h[3])
						checkScalar(t, fmt.Sprintf("%s n=%d incX=%d incY=%d x[%d]", name, n, ix, iy, i), 2, ex[i], rx, sx)
						checkScalar(t, fmt.Sprintf("%s n=%d incX=%d incY=%d y[%d]", name, n, ix, iy, i), 2, ey[i], ry, sy)
					}
					checkGaps(t, name+" x", gx, x, n, ix)
					checkGaps(t, name+" y", gy, y, n, iy)
				}
				gx, gy := append([]float32(nil), x...), append([]float32(nil), y...)
				Srot(n, gx, ix, gy, iy, c, s)
				check("Srot", gx, gy, [4]float32{c, s, -s, c})
				for _, p := range params {
					gx, gy := append([]float32(nil), x...), append([]float32(nil), y...)
					Srotm(n, gx, ix, gy, iy, p.p)
					check("Srotm "+p.name, gx, gy, p.h)
				}
			}
		}
	}
	mustPanic(t, "Srotm bad flag", func() { Srotm(1, []float32{1}, 1, []float32{1}, 1, [5]float32{3}) })
	mustPanic(t, "Srot zero incY", func() { Srot(2, make([]float32, 2), 1, make([]float32, 2), 0, c, s) })
	mustPanic(t, "Scopy short y", func() { Scopy(3, make([]float32, 3), 1, make([]float32, 2), 1) })
	mustPanic(t, "Sswap negative n", func() { Sswap(-1, nil, 1, nil, 1) })
}

// TestSrotg checks the definition rather than one implementation's arithmetic:
// the rotation must zero b, have c² + s² = 1, and be safe at the ends of the range.
func TestSrotg(t *testing.T) {
	vals := []float32{0, 1, -1, 3, -4, 0.5, 1e-30, -1e-30, 1e30, -1e30, 3e38, -3e38,
		math.SmallestNonzeroFloat32, 1e-40, 2.5, -7}
	for _, a := range vals {
		for _, b := range vals {
			c, s, rr, z := Srotg(a, b)
			what := fmt.Sprintf("Srotg(%v, %v) = c %v, s %v, r %v, z %v", a, b, c, s, rr, z)
			trueR := math.Hypot(float64(a), float64(b))
			if trueR > math.MaxFloat32 {
				// r is not representable, so ±Inf is the right answer, and c, s are then 0.
				if !math.IsInf(float64(rr), 0) {
					t.Errorf("%s: |r| should overflow to Inf (true value %.4g)", what, trueR)
				}
				continue
			}
			if math.IsInf(float64(rr), 0) || math.IsNaN(float64(rr)) {
				t.Errorf("%s: r is not finite though %.4g is representable", what, trueR)
				continue
			}
			if trueR < 0x1p-126 {
				// A subnormal r has few bits, and c = a/r, s = b/r inherit that, so the
				// identities below hold only to r's own precision. Check r alone.
				if d := math.Abs(math.Abs(float64(rr)) - trueR); d > 0x1p-149 {
					t.Errorf("%s: |r| %.4g is more than one subnormal step from %.4g", what, rr, trueR)
				}
				continue
			}
			if d := math.Abs(float64(c)*float64(c) + float64(s)*float64(s) - 1); d > float64(oracle.Tolerance(2, 1)) {
				t.Errorf("%s: c²+s² is off 1 by %.3g", what, d)
			}
			// [c s; -s c]·[a; b] = [r; 0], judged in float64 at the inputs' scale.
			fa, fb := float64(a), float64(b)
			scale := math.Abs(fa) + math.Abs(fb)
			if d := math.Abs(float64(c)*fa + float64(s)*fb - float64(rr)); d > oracle.Tolerance(4, scale) {
				t.Errorf("%s: c·a + s·b misses r by %.3g", what, d)
			}
			if d := math.Abs(-float64(s)*fa + float64(c)*fb); d > oracle.Tolerance(4, scale) {
				t.Errorf("%s: −s·a + c·b is %.3g, not 0", what, d)
			}
			// z recovers (c, s), the reference BLAS convention.
			var zc, zs float64
			switch zf := float64(z); {
			case zf == 1:
				zc, zs = 0, 1
			case math.Abs(zf) < 1:
				zc, zs = math.Sqrt(1-zf*zf), zf
			default:
				zc, zs = 1/zf, math.Sqrt(1-1/(zf*zf))
			}
			if b != 0 && a != 0 && (math.Abs(math.Abs(zc)-math.Abs(float64(c))) > 1e-6 || math.Abs(math.Abs(zs)-math.Abs(float64(s))) > 1e-6) {
				// Only magnitudes are recoverable from z; signs come from r.
				t.Errorf("%s: z does not recover |c|, |s| (got %v, %v)", what, zc, zs)
			}
		}
	}
	if c, s, rr, z := Srotg(2, 0); c != 1 || s != 0 || rr != 2 || z != 0 {
		t.Errorf("Srotg(2, 0) = %v %v %v %v, want 1 0 2 0", c, s, rr, z)
	}
	if c, s, rr, z := Srotg(0, -3); c != 0 || s != 1 || rr != -3 || z != 1 {
		t.Errorf("Srotg(0, -3) = %v %v %v %v, want 0 1 -3 1", c, s, rr, z)
	}
}

// hOf expands Srotmg's SPARAM into the full H the flag denotes.
func hOf(p [5]float32) (h11, h12, h21, h22 float64) {
	switch p[0] {
	case RotmIdentity:
		return 1, 0, 0, 1
	case RotmRescaling:
		return float64(p[1]), float64(p[3]), float64(p[2]), float64(p[4])
	case RotmOffDiagonal:
		return 1, float64(p[3]), float64(p[2]), 1
	default: // RotmDiagonal
		return float64(p[1]), 1, -1, float64(p[4])
	}
}

// TestSrotmg checks the defining properties: H zeroes y1, maps x1 to rx1, and
// preserves the weighted norm, d1·x1² + d2·y1² = rd1·rx1², across the rescaling
// thresholds and the error states.
func TestSrotmg(t *testing.T) {
	r := rand.New(rand.NewSource(67))
	type in struct{ d1, d2, x1, y1 float32 }
	cases := []in{{1, 1, 3, 4}, {2, 3, -1, 5}, {0, 2, 1, 1}, {1, 2, 0, 3}, {1, 1, 5, 0}, {1, 0, 2, 3}, {1e-10, 1, 1, 1},
		{1e10, 1e-10, 1, 1}, {1, 1e9, 1, 1}, {1e-9, 1e-9, 2, 7}}
	for i := 0; i < 200; i++ {
		cases = append(cases, in{float32(r.ExpFloat64()), float32(r.ExpFloat64()), float32(r.NormFloat64()), float32(r.NormFloat64())})
	}
	for _, c := range cases {
		p, rd1, rd2, rx1 := Srotmg(c.d1, c.d2, c.x1, c.y1)
		what := fmt.Sprintf("Srotmg(%v, %v, %v, %v) = %v, %v, %v, %v", c.d1, c.d2, c.x1, c.y1, p, rd1, rd2, rx1)
		if p[0] == RotmRescaling && rd1 == 0 && rd2 == 0 && rx1 == 0 {
			continue // the error state; covered by the explicit cases below
		}
		h11, h12, h21, h22 := hOf(p)
		x, y := float64(c.x1), float64(c.y1)
		// The zeroing is in the weighted sense: sqrt(rd2)·(h21·x1 + h22·y1) = 0. When
		// rd2 is 0 (d1 or d2 was 0), the unweighted second component need not be.
		second := h21*x + h22*y
		if w := float64(rd2) * second * second; w > 1e-8*(float64(rd1)*float64(rx1)*float64(rx1)+1e-30) {
			t.Errorf("%s: H does not zero the weighted second component (rd2·y'² = %.3g)", what, w)
		}
		if got := h11*x + h12*y; math.Abs(got-float64(rx1)) > 1e-5*(math.Abs(got)+1e-30) {
			t.Errorf("%s: H maps x1 to %.6g, not rx1", what, got)
		}
		want := float64(c.d1)*x*x + float64(c.d2)*y*y
		if got := float64(rd1)*float64(rx1)*float64(rx1) + float64(rd2)*second*second; math.Abs(got-want) > 1e-4*math.Abs(want) {
			t.Errorf("%s: weighted norm %.6g, want %.6g", what, got, want)
		}
	}
	if p, a, b, c := Srotmg(-1, 1, 1, 1); p[0] != RotmRescaling || a != 0 || b != 0 || c != 0 || p[1] != 0 {
		t.Errorf("Srotmg with d1 < 0 = %v %v %v %v, want the zeroed RotmRescaling error state", p, a, b, c)
	}
	if p, a, b, c := Srotmg(2, 3, 4, 0); p[0] != RotmIdentity || a != 2 || b != 3 || c != 4 {
		t.Errorf("Srotmg with y1 == 0 = %v %v %v %v, want RotmIdentity and the inputs back", p, a, b, c)
	}
}

// TestSrotgSignConvention pins what the property test cannot: negating c, s and r
// together is still a valid rotation, so only the convention separates them.
// Reference BLAS gives r the sign of whichever of a and b is larger in magnitude.
func TestSrotgSignConvention(t *testing.T) {
	for _, tc := range []struct {
		a, b  float32
		rSign float64
	}{{3, 1, 1}, {-3, 1, -1}, {3, -1, 1}, {1, 3, 1}, {1, -3, -1}, {-1, 3, 1}, {-2, -2, -1}} {
		_, _, r, _ := Srotg(tc.a, tc.b)
		if math.Copysign(1, float64(r)) != tc.rSign {
			t.Errorf("Srotg(%v, %v): r = %v, want sign %+v", tc.a, tc.b, r, tc.rSign)
		}
	}
}

// TestSrotmgFlagChoice pins the branch: both forms zero y1, so only the choice,
// made for stability on |d1·x1²| against |d2·y1²|, separates them.
func TestSrotmgFlagChoice(t *testing.T) {
	for _, tc := range []struct {
		d1, d2, x1, y1, flag float32
	}{
		{1, 1, 3, 1, RotmOffDiagonal}, // |q1| = 9 > |q2| = 1
		{1, 1, 1, 3, RotmDiagonal},    // |q1| = 1 < |q2| = 9
		{0, 2, 1, 1, RotmDiagonal},    // d1 == 0 swaps
		{1, 0, 2, 3, RotmIdentity},    // d2 == 0: nothing to do
	} {
		if p, _, _, _ := Srotmg(tc.d1, tc.d2, tc.x1, tc.y1); p[0] != tc.flag {
			t.Errorf("Srotmg(%v, %v, %v, %v): flag %v, want %v", tc.d1, tc.d2, tc.x1, tc.y1, p[0], tc.flag)
		}
	}
}
