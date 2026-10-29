// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/l164"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// The float64 Level-1 suite (#192), l1_test.go's discipline at float64 and held to
// DESIGN.md §5 rule 1's three checks per routine. This file has two of them:
// (1) the definition-order float64 references (oracle.Ddot, …) under
// PeerTolerance64, and (2) the double-double references (oracle.DotDD, AsumDD,
// Nrm2DD, AxpyDD) under Tolerance64. Dscal and Idamax need no dd twin, because
// their float64 references are already the correctly rounded and the exact
// answers, so the reference itself is the isolating arm under Tolerance64. (3), the
// independent implementation, is gonumblas_test.go's TestLevel1AgreesWithGonum64.
//
// As in l1_test.go the per-backend loop swaps the package-level activeL164, so the
// public routine, its checks, its stride split and Dnrm2's rescue are in the
// tested path. No t.Parallel.

// patterns64 are l1_test.go's patterns at float64 scale. "tiny" squares to zero
// and "subnormal-squares" lands its squares in float64's subnormal band, which is
// what Dnrm2's threshold exists for (the #97 lesson at float64).
var patterns64 = []struct {
	name string
	gen  func(r *rand.Rand, n int) []float64
}{
	{"uniform", func(r *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			v[i] = r.Float64()*2 - 1
		}
		return v
	}},
	{"ramp", func(_ *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			v[i] = float64(i+1) / 16
		}
		return v
	}},
	{"cancelling", func(_ *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			v[i] = 1e12 + float64(i)
			if i%2 == 1 {
				v[i] = -v[i]
			}
		}
		return v
	}},
	{"mixed-magnitude", func(r *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			if i%2 == 0 {
				v[i] = 1e12 * (r.Float64() + 0.5)
			} else {
				v[i] = 1e-12 * (r.Float64() + 0.5)
			}
		}
		return v
	}},
	{"sparse", func(r *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			if r.Intn(8) == 0 {
				v[i] = r.Float64() * 4
			}
		}
		return v
	}},
	{"tiny", func(r *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			v[i] = (r.Float64() + 0.5) * 1e-170
		}
		return v
	}},
	// Squares in [~1e-323, ~2e-308]: nonzero, finite and carrying lost significance.
	{"subnormal-squares", func(r *rand.Rand, n int) []float64 {
		v := make([]float64, n)
		for i := range v {
			// 3e-162 · 10^u, u uniform on [0, 7]: |v| in [3e-162, 3e-155].
			v[i] = 3e-162 * math.Pow(10, 7*r.Float64())
			if r.Intn(2) == 0 {
				v[i] = -v[i]
			}
		}
		return v
	}},
}

var exercised64 = map[string]bool{}

func forEachBackend64(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	saved := activeL164
	t.Cleanup(func() { activeL164 = saved })
	for _, b := range l164.Backends() {
		activeL164 = b
		exercised64[b.Name] = true
		t.Run(b.Name, func(t *testing.T) { f(t) })
	}
}

// tolFn is one arm's bound: oracle.PeerTolerance64 or oracle.Tolerance64.
type tolFn func(n int, scale float64) float64

// check64 is checkScalar at float64, with the arm's bound passed in: non-finite
// results must match exactly, finite ones within tol(n, scale).
func check64(t *testing.T, what string, n int, got, want, scale float64, tol tolFn) {
	t.Helper()
	switch {
	case math.IsNaN(want) || math.IsNaN(got):
		if !(math.IsNaN(want) && math.IsNaN(got)) {
			t.Errorf("%s: got %v, reference %v (NaN mismatch)", what, got, want)
		}
	case math.IsInf(want, 0) || math.IsInf(got, 0):
		if got != want {
			t.Errorf("%s: got %v, reference %v (infinity mismatch)", what, got, want)
		}
	default:
		if b := tol(n, scale); math.Abs(got-want) > b {
			t.Errorf("%s: got %v, reference %v, |err| %.3g > tol %.3g (scale %.3g)",
				what, got, want, math.Abs(got-want), b, scale)
		}
	}
}

// checkVec64 compares an elementwise result at f(n) = 1, for checkVecResult's reason.
func checkVec64(t *testing.T, what string, got, want, scale []float64, tol tolFn) {
	t.Helper()
	for j := range want {
		check64(t, fmt.Sprintf("%s[%d]", what, j), 1, got[j], want[j], scale[j], tol)
	}
}

func spread64(v []float64, inc int) []float64 {
	n := len(v)
	if n == 0 {
		return nil
	}
	span := max(inc, -inc)
	out := make([]float64, (n-1)*span+1)
	for i := range out {
		out[i] = poison64
	}
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[base+j*inc] = v[j]
	}
	return out
}

// poison64 fills stride gaps: a kernel that reads one or writes one shows.
var poison64 = math.Float64frombits(0x7ff8_dead_beef_0064) // a NaN with a recognisable payload

func gather64(s []float64, n, inc int) []float64 {
	out := make([]float64, n)
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[j] = s[base+j*inc]
	}
	return out
}

func checkGaps64(t *testing.T, what string, got, before []float64, n, inc int) {
	t.Helper()
	touched := make(map[int]bool, n)
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		touched[base+j*inc] = true
	}
	for i := range got {
		if !touched[i] && math.Float64bits(got[i]) != math.Float64bits(before[i]) {
			t.Errorf("%s: wrote through a stride gap at %d (%v -> %v)", what, i, before[i], got[i])
		}
	}
}

// TestL1Oracle64 is checks (1) and (2) for every unit-stride routine, every
// backend, every pattern and shape.
func TestL1Oracle64(t *testing.T) {
	forEachBackend64(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns64 {
			for _, n := range shapes {
				x, y := p.gen(r, n), p.gen(r, n)
				what := func(fn string) string { return fmt.Sprintf("%s %s n=%d", fn, p.name, n) }

				got := Ddot(n, x, 1, y, 1)
				w, s := oracle.Ddot(n, x, 1, y, 1)
				check64(t, what("Ddot/ref"), n, got, w, s, oracle.PeerTolerance64)
				w, s = oracle.DotDD(n, x, 1, y, 1)
				check64(t, what("Ddot/dd"), n, got, w, s, oracle.Tolerance64)

				for _, alpha := range []float64{1, -0.75, 3.5} {
					yk := append([]float64(nil), y...)
					Daxpy(n, alpha, x, 1, yk, 1)
					wv, sv := oracle.Daxpy(n, alpha, x, 1, y, 1)
					checkVec64(t, what(fmt.Sprintf("Daxpy/ref a=%v", alpha)), yk, wv, sv, oracle.PeerTolerance64)
					wv, sv = oracle.AxpyDD(n, alpha, x, 1, y, 1)
					checkVec64(t, what(fmt.Sprintf("Daxpy/dd a=%v", alpha)), yk, wv, sv, oracle.Tolerance64)

					xk := append([]float64(nil), x...)
					Dscal(n, alpha, xk, 1)
					wv, sv = oracle.Dscal(n, alpha, x, 1)
					checkVec64(t, what(fmt.Sprintf("Dscal a=%v", alpha)), xk, wv, sv, oracle.Tolerance64)
				}

				got = Dasum(n, x, 1)
				w, s = oracle.Dasum(n, x, 1)
				check64(t, what("Dasum/ref"), n, got, w, s, oracle.PeerTolerance64)
				w, s = oracle.AsumDD(n, x, 1)
				check64(t, what("Dasum/dd"), n, got, w, s, oracle.Tolerance64)

				got = Dnrm2(n, x, 1)
				w, s = oracle.Dnrm2(n, x, 1)
				check64(t, what("Dnrm2/ref"), n, got, w, s, oracle.PeerTolerance64)
				w, s = oracle.Nrm2DD(n, x, 1)
				check64(t, what("Dnrm2/dd"), n, got, w, s, oracle.Tolerance64)

				if g, wi := Idamax(n, x, 1), oracle.Idamax(n, x, 1); g != wi {
					t.Errorf("%s: got %d, reference %d", what("Idamax"), g, wi)
				}
			}
		}
	})
}

// TestStrides64 is TestStrides at float64, including Ddot's and Daxpy's negative
// strides and the gap poison.
func TestStrides64(t *testing.T) {
	forEachBackend64(t, func(t *testing.T) {
		r := newRand()
		for _, n := range []int{1, 2, 3, 8, 17, 64, 129} {
			for _, inc := range []int{2, 3, 7, -1, -2, -3} {
				xd, yd := patterns64[0].gen(r, n), patterns64[0].gen(r, n)
				x, y := spread64(xd, inc), spread64(yd, inc)
				what := func(fn string) string { return fmt.Sprintf("%s(n=%d,inc=%d)", fn, n, inc) }

				w, s := oracle.DotDD(n, x, inc, y, inc)
				check64(t, what("Ddot"), n, Ddot(n, x, inc, y, inc), w, s, oracle.Tolerance64)
				x1 := spread64(xd, 1)
				w, s = oracle.DotDD(n, x1, 1, y, inc)
				check64(t, what("Ddot incX=1"), n, Ddot(n, x1, 1, y, inc), w, s, oracle.Tolerance64)

				wv, sv := oracle.AxpyDD(n, 2.5, x, inc, y, inc)
				yc := append([]float64(nil), y...)
				Daxpy(n, 2.5, x, inc, y, inc)
				checkVec64(t, what("Daxpy"), gather64(y, n, inc), wv, sv, oracle.Tolerance64)
				checkGaps64(t, what("Daxpy"), y, yc, n, inc)

				if inc > 0 {
					xs := spread64(xd, inc)
					xc := append([]float64(nil), xs...)
					wv, sv = oracle.Dscal(n, -0.75, xs, inc)
					Dscal(n, -0.75, xs, inc)
					checkVec64(t, what("Dscal"), gather64(xs, n, inc), wv, sv, oracle.Tolerance64)
					checkGaps64(t, what("Dscal"), xs, xc, n, inc)

					w, s = oracle.AsumDD(n, x, inc)
					check64(t, what("Dasum"), n, Dasum(n, x, inc), w, s, oracle.Tolerance64)
					w, s = oracle.Nrm2DD(n, x, inc)
					check64(t, what("Dnrm2"), n, Dnrm2(n, x, inc), w, s, oracle.Tolerance64)
					if g, wi := Idamax(n, x, inc), oracle.Idamax(n, x, inc); g != wi {
						t.Errorf("%s: got %d, reference %d", what("Idamax"), g, wi)
					}
				}
			}
		}
	})
}

// TestBackendsAgree64 holds every backend to every other at PeerTolerance64: each
// is within Tolerance64 of exact, so any two are within twice it.
func TestBackendsAgree64(t *testing.T) {
	bs := l164.Backends()
	saved := activeL164
	t.Cleanup(func() { activeL164 = saved })
	r := newRand()
	for _, p := range patterns64 {
		for _, n := range shapes {
			x, y := p.gen(r, n), p.gen(r, n)
			_, sd := oracle.Ddot(n, x, 1, y, 1)
			_, sa := oracle.Dasum(n, x, 1)
			_, sn := oracle.Dnrm2(n, x, 1)
			type res struct{ dot, asum, nrm float64 }
			out := make([]res, len(bs))
			for i, b := range bs {
				activeL164 = b
				out[i] = res{Ddot(n, x, 1, y, 1), Dasum(n, x, 1), Dnrm2(n, x, 1)}
			}
			for i := 1; i < len(bs); i++ {
				w := fmt.Sprintf("%s vs %s, %s n=%d", bs[i].Name, bs[0].Name, p.name, n)
				check64(t, "Ddot "+w, n, out[i].dot, out[0].dot, sd, oracle.PeerTolerance64)
				check64(t, "Dasum "+w, n, out[i].asum, out[0].asum, sa, oracle.PeerTolerance64)
				check64(t, "Dnrm2 "+w, n, out[i].nrm, out[0].nrm, sn, oracle.PeerTolerance64)
			}
		}
	}
}

// TestNonFinite64: a NaN or an infinity must survive every reduction, and Idamax
// must keep the reference's NaN convention.
func TestNonFinite64(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	forEachBackend64(t, func(t *testing.T) {
		for _, n := range []int{1, 7, 64, 129} {
			for _, at := range []int{0, n / 2, n - 1} {
				for _, v := range []float64{nan, inf, -inf} {
					x := make([]float64, n)
					for i := range x {
						x[i] = 1
					}
					x[at] = v
					what := fmt.Sprintf("n=%d x[%d]=%v", n, at, v)
					w, s := oracle.DotDD(n, x, 1, x, 1)
					check64(t, "Ddot "+what, n, Ddot(n, x, 1, x, 1), w, s, oracle.Tolerance64)
					w, s = oracle.AsumDD(n, x, 1)
					check64(t, "Dasum "+what, n, Dasum(n, x, 1), w, s, oracle.Tolerance64)
					w, s = oracle.Nrm2DD(n, x, 1)
					check64(t, "Dnrm2 "+what, n, Dnrm2(n, x, 1), w, s, oracle.Tolerance64)
					if g, wi := Idamax(n, x, 1), oracle.Idamax(n, x, 1); g != wi {
						t.Errorf("Idamax %s: got %d, reference %d", what, g, wi)
					}
				}
			}
		}
	})
}

// TestDnrm2Rescue drives each way into the rescue and the edges of the fast
// path's threshold n·2^-1022, against the dd reference.
func TestDnrm2Rescue(t *testing.T) {
	fill := func(n int, v float64) []float64 {
		x := make([]float64, n)
		for i := range x {
			x[i] = v
		}
		return x
	}
	cases := map[string][]float64{
		"one tiny":          {1e-300},
		"one subnormal":     {5e-324},
		"one huge":          {1e300},
		"huge overflow":     fill(16, 1e300),
		"max":               fill(4, math.MaxFloat64),
		"tiny vector":       fill(1000, 1e-170),
		"just below thresh": fill(64, math.Sqrt(0x1p-1022)*0.99),
		"just above thresh": fill(64, math.Sqrt(0x1p-1022)*1.01),
		"zero":              fill(33, 0),
		"one big many tiny": append([]float64{1}, fill(99, 1e-300)...),
	}
	forEachBackend64(t, func(t *testing.T) {
		for name, x := range cases {
			n := len(x)
			w, s := oracle.Nrm2DD(n, x, 1)
			check64(t, "Dnrm2 "+name, n, Dnrm2(n, x, 1), w, s, oracle.Tolerance64)
			// The strided path takes the same rescue.
			xs := spread64(x, 2)
			check64(t, "Dnrm2 strided "+name, n, Dnrm2(n, xs, 2), w, s, oracle.Tolerance64)
		}
		// n = 1 is exact: the norm of one element is its magnitude.
		for _, v := range []float64{1e-300, -5e-324, 1e300, -math.MaxFloat64, 3} {
			if g := Dnrm2(1, []float64{v}, 1); g != math.Abs(v) {
				t.Errorf("Dnrm2(%v) = %v, want %v exactly", v, g, math.Abs(v))
			}
		}
	})
}

// panicMsg returns the panic value of f as a string, or "" if it returned.
func panicMsg(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}

// TestDParity pins each float64 Level-1 routine to its float32 twin on bad
// arguments: both panic, with the same message bar the routine's name. That is the
// guard the hand-written D API is ruled to carry (docs/f64-design.md, 2026-10-29).
func TestDParity(t *testing.T) {
	type bad struct{ n, incX, incY, lenX, lenY int }
	cases := []bad{
		{-1, 1, 1, 0, 0}, // n < 0
		{2, 0, 1, 2, 2},  // zero incX
		{2, 1, 0, 2, 2},  // zero incY
		{3, 1, 1, 2, 3},  // x too short
		{3, 1, 1, 3, 2},  // y too short
		{3, 2, 1, 4, 3},  // x too short for the stride
		{2, -1, 1, 2, 2}, // negative incX: legal for some, refused by the Pos routines
	}
	pairs := []struct {
		name string
		s, d func(c bad, x32 []float32, y32 []float32, x64, y64 []float64)
	}{
		{"dot", func(c bad, x, y []float32, _, _ []float64) { Sdot(c.n, x, c.incX, y, c.incY) },
			func(c bad, _, _ []float32, x, y []float64) { Ddot(c.n, x, c.incX, y, c.incY) }},
		{"axpy", func(c bad, x, y []float32, _, _ []float64) { Saxpy(c.n, 2, x, c.incX, y, c.incY) },
			func(c bad, _, _ []float32, x, y []float64) { Daxpy(c.n, 2, x, c.incX, y, c.incY) }},
		{"scal", func(c bad, x, _ []float32, _, _ []float64) { Sscal(c.n, 2, x, c.incX) },
			func(c bad, _, _ []float32, x, _ []float64) { Dscal(c.n, 2, x, c.incX) }},
		{"asum", func(c bad, x, _ []float32, _, _ []float64) { Sasum(c.n, x, c.incX) },
			func(c bad, _, _ []float32, x, _ []float64) { Dasum(c.n, x, c.incX) }},
		{"nrm2", func(c bad, x, _ []float32, _, _ []float64) { Snrm2(c.n, x, c.incX) },
			func(c bad, _, _ []float32, x, _ []float64) { Dnrm2(c.n, x, c.incX) }},
		{"iamax", func(c bad, x, _ []float32, _, _ []float64) { Isamax(c.n, x, c.incX) },
			func(c bad, _, _ []float32, x, _ []float64) { Idamax(c.n, x, c.incX) }},
		{"copy", func(c bad, x, y []float32, _, _ []float64) { Scopy(c.n, x, c.incX, y, c.incY) },
			func(c bad, _, _ []float32, x, y []float64) { Dcopy(c.n, x, c.incX, y, c.incY) }},
		{"swap", func(c bad, x, y []float32, _, _ []float64) { Sswap(c.n, x, c.incX, y, c.incY) },
			func(c bad, _, _ []float32, x, y []float64) { Dswap(c.n, x, c.incX, y, c.incY) }},
		{"rot", func(c bad, x, y []float32, _, _ []float64) { Srot(c.n, x, c.incX, y, c.incY, 0.6, 0.8) },
			func(c bad, _, _ []float32, x, y []float64) { Drot(c.n, x, c.incX, y, c.incY, 0.6, 0.8) }},
		{"rotm", func(c bad, x, y []float32, _, _ []float64) {
			Srotm(c.n, x, c.incX, y, c.incY, [5]float32{RotmRescaling, 1, 2, 3, 4})
		}, func(c bad, _, _ []float32, x, y []float64) {
			Drotm(c.n, x, c.incX, y, c.incY, [5]float64{float64(RotmRescaling), 1, 2, 3, 4})
		}},
		{"rotm bad flag", func(c bad, x, y []float32, _, _ []float64) {
			Srotm(c.n, x, c.incX, y, c.incY, [5]float32{3})
		}, func(c bad, _, _ []float32, x, y []float64) {
			Drotm(c.n, x, c.incX, y, c.incY, [5]float64{3})
		}},
	}
	// The routine's name appears in each message; normalize S… and I…s… to D… and I…d….
	norm := func(m string) string {
		for _, r := range []string{"dot", "axpy", "scal", "asum", "nrm2", "copy", "swap", "rotm", "rot"} {
			m = strings.ReplaceAll(m, ": S"+r+":", ": D"+r+":")
		}
		return strings.ReplaceAll(m, ": Isamax:", ": Idamax:")
	}
	valid := bad{3, 1, 1, 3, 3}
	for _, p := range pairs {
		for _, c := range append(cases, valid) {
			x32, y32 := make([]float32, max(c.lenX, 0)), make([]float32, max(c.lenY, 0))
			x64, y64 := make([]float64, max(c.lenX, 0)), make([]float64, max(c.lenY, 0))
			ms := panicMsg(func() { p.s(c, x32, y32, nil, nil) })
			md := panicMsg(func() { p.d(c, nil, nil, x64, y64) })
			if norm(ms) != md {
				t.Errorf("%s %+v: float32 panics %q, float64 panics %q", p.name, c, ms, md)
			}
		}
	}
}

// TestL164BackendsExercised is the float64 coverage claim stated as a test: every
// backend l164 registers on this machine ran the suites above.
func TestL164BackendsExercised(t *testing.T) {
	if len(exercised64) == 0 {
		t.Skip("no float64 Level-1 suite ran in this invocation")
	}
	for _, b := range l164.Backends() {
		if !exercised64[b.Name] {
			t.Errorf("float64 backend %s is registered but no suite ran it", b.Name)
		}
	}
	t.Logf("keel-l164-backends-exercised: %s (active %s)", strings.Join(l164.Names(), " "), ActiveL164Backend())
}
