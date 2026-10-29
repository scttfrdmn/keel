// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel"
	"github.com/scttfrdmn/keel/internal/oracle"
	"gonum.org/v1/gonum/blas"
)

// The third arm of DESIGN.md §5 rule 1 for float64 Level 1 (#192): Gonum's own
// float64 BLAS, an independent implementation, against keel's D routines called
// directly. The blas.Float64 adapter is a later #192 stage; until it exists, these
// calls are keel's public API, so a negative stride on the inc>0-only routines is
// keel's panic by design and is not compared here. The arbiter is still the
// oracle's scale: keel and gonum must agree within PeerTolerance64 of it, since
// each is bounded by Tolerance64.

func randVec64(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = r.NormFloat64()
	}
	return v
}

// sameElem64 is sameElem at float64.
func sameElem64(k, g float64, n int, scale float64) bool {
	if math.IsNaN(k) || math.IsNaN(g) {
		return math.IsNaN(k) && math.IsNaN(g)
	}
	if math.IsInf(k, 0) || math.IsInf(g, 0) {
		return k == g
	}
	return math.Abs(k-g) <= oracle.PeerTolerance64(n, scale)
}

func TestLevel1AgreesWithGonum64(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, n := range l1Sizes {
		for _, ix := range l1Incs {
			x := randVec64(r, storage(n, ix))
			for _, iy := range l1Incs {
				y := randVec64(r, storage(n, iy))
				what := fmt.Sprintf("n=%d incX=%d incY=%d", n, ix, iy)

				_, sc := oracle.Ddot(n, x, ix, y, iy)
				if k, g := keel.Ddot(n, x, ix, y, iy), gonumImpl.Ddot(n, x, ix, y, iy); !sameElem64(k, g, n, sc) {
					t.Errorf("Ddot %s: keel %v, gonum %v", what, k, g)
				}
				for _, alpha := range []float64{0, 1, -0.75} {
					yk, yg := append([]float64(nil), y...), append([]float64(nil), y...)
					keel.Daxpy(n, alpha, x, ix, yk, iy)
					gonumImpl.Daxpy(n, alpha, x, ix, yg, iy)
					_, s := oracle.Daxpy(n, alpha, x, ix, y, iy)
					start := oracle.Index(n, iy)
					for i := range yk {
						sc := 0.0
						if n > 0 && (i-start)%iy == 0 {
							if j := (i - start) / iy; j >= 0 && j < n {
								sc = s[j]
							}
						}
						if !sameElem64(yk[i], yg[i], 1, sc) {
							t.Errorf("Daxpy %s alpha=%v: storage[%d] keel %v, gonum %v", what, alpha, i, yk[i], yg[i])
							break
						}
					}
				}
			}
			if ix < 0 {
				continue // inc>0-only below: keel panics on purpose; the adapter will translate
			}
			what := fmt.Sprintf("n=%d inc=%d", n, ix)
			for _, alpha := range []float64{0, 2.5} {
				xk, xg := append([]float64(nil), x...), append([]float64(nil), x...)
				keel.Dscal(n, alpha, xk, ix)
				gonumImpl.Dscal(n, alpha, xg, ix)
				for i := range xk {
					if !sameElem64(xk[i], xg[i], 1, math.Abs(alpha*x[i])) {
						t.Errorf("Dscal %s alpha=%v: storage[%d] keel %v, gonum %v", what, alpha, i, xk[i], xg[i])
						break
					}
				}
			}
			_, sa := oracle.Dasum(n, x, ix)
			if k, g := keel.Dasum(n, x, ix), gonumImpl.Dasum(n, x, ix); !sameElem64(k, g, n, sa) {
				t.Errorf("Dasum %s: keel %v, gonum %v", what, k, g)
			}
			_, sn := oracle.Dnrm2(n, x, ix)
			if k, g := keel.Dnrm2(n, x, ix), gonumImpl.Dnrm2(n, x, ix); !sameElem64(k, g, n, sn) {
				t.Errorf("Dnrm2 %s: keel %v, gonum %v", what, k, g)
			}
			if n > 0 {
				if k, g := keel.Idamax(n, x, ix), gonumImpl.Idamax(n, x, ix); k != g {
					t.Errorf("Idamax %s: keel %d, gonum %d", what, k, g)
				}
			}
		}
	}
	// Dnrm2 across the range gonum's scaled algorithm and keel's rescue both cover.
	for _, v := range []float64{1e-300, 1e-170, 1e300, 5e-324} {
		x := make([]float64, 40)
		for i := range x {
			x[i] = v * float64(i%5+1)
		}
		_, s := oracle.Nrm2DD(len(x), x, 1)
		if k, g := keel.Dnrm2(len(x), x, 1), gonumImpl.Dnrm2(len(x), x, 1); !sameElem64(k, g, len(x), s) {
			t.Errorf("Dnrm2 at %g: keel %v, gonum %v", v, k, g)
		}
	}
}

// TestCopySwapRotationsAgreeWithGonum64. Dcopy and Dswap must be bit-identical.
// Drot is held to the tolerance model, for Srot's reason (gonum leaves
// c*vx+s*vy unbarriered). Drotm, Drotg and Drotmg are compared bit for bit, which
// is what their float32 twins achieve against gonum.
func TestCopySwapRotationsAgreeWithGonum64(t *testing.T) {
	r := rand.New(rand.NewSource(13))
	bits := func(v []float64) string { return fmt.Sprintf("%x", v) }
	params := []blas.DrotmParams{
		{Flag: blas.Identity, H: [4]float64{9, 9, 9, 9}},
		{Flag: blas.Rescaling, H: [4]float64{1.5, -0.25, 0.75, 2}},
		{Flag: blas.OffDiagonal, H: [4]float64{9, -0.25, 0.75, 9}},
		{Flag: blas.Diagonal, H: [4]float64{1.5, 9, 9, 2}},
	}
	toKeel := func(p blas.DrotmParams) [5]float64 {
		return [5]float64{float64(p.Flag), p.H[0], p.H[1], p.H[2], p.H[3]}
	}
	for _, n := range []int{0, 1, 7, 257} {
		for _, ix := range []int{1, 2, -1, -3} {
			for _, iy := range []int{1, -2} {
				x, y := randVec64(r, storage(n, ix)), randVec64(r, storage(n, iy))
				what := fmt.Sprintf("n=%d incX=%d incY=%d", n, ix, iy)
				pair := func(name string, fk func(x, y []float64), fg func(x, y []float64)) {
					t.Helper()
					xk, yk := append([]float64(nil), x...), append([]float64(nil), y...)
					xg, yg := append([]float64(nil), x...), append([]float64(nil), y...)
					fk(xk, yk)
					fg(xg, yg)
					if bits(xk) != bits(xg) || bits(yk) != bits(yg) {
						t.Errorf("%s %s: not bit-identical to gonum", name, what)
					}
				}
				pair("Dcopy", func(x, y []float64) { keel.Dcopy(n, x, ix, y, iy) }, func(x, y []float64) { gonumImpl.Dcopy(n, x, ix, y, iy) })
				pair("Dswap", func(x, y []float64) { keel.Dswap(n, x, ix, y, iy) }, func(x, y []float64) { gonumImpl.Dswap(n, x, ix, y, iy) })
				for _, p := range params {
					pair(fmt.Sprintf("Drotm flag=%d", p.Flag),
						func(x, y []float64) { keel.Drotm(n, x, ix, y, iy, toKeel(p)) },
						func(x, y []float64) { gonumImpl.Drotm(n, x, ix, y, iy, p) })
				}
				xk, yk := append([]float64(nil), x...), append([]float64(nil), y...)
				xg, yg := append([]float64(nil), x...), append([]float64(nil), y...)
				keel.Drot(n, xk, ix, yk, iy, 0.6, -0.8)
				gonumImpl.Drot(n, xg, ix, yg, iy, 0.6, -0.8)
				for i := range xk {
					if !sameElem64(xk[i], xg[i], 2, math.Abs(0.6*x[i])+math.Abs(0.8*y[min(i, len(y)-1)])+1) {
						t.Errorf("Drot %s: x storage[%d] keel %v, gonum %v", what, i, xk[i], xg[i])
						break
					}
				}
				for i := range yk {
					if !sameElem64(yk[i], yg[i], 2, math.Abs(0.6*y[i])+math.Abs(0.8*x[min(i, len(x)-1)])+1) {
						t.Errorf("Drot %s: y storage[%d] keel %v, gonum %v", what, i, yk[i], yg[i])
						break
					}
				}
			}
		}
	}
	vals := []float64{0, 1, -1, 3, -4, 0.5, 1e-300, -1e300, 1.5e308, -1.5e308, 1e-310, 2.5}
	for _, a := range vals {
		for _, b := range vals {
			c1, s1, r1, z1 := keel.Drotg(a, b)
			c2, s2, r2, z2 := gonumImpl.Drotg(a, b)
			if bits([]float64{c1, s1, r1, z1}) != bits([]float64{c2, s2, r2, z2}) {
				t.Errorf("Drotg(%v, %v): keel %v %v %v %v, gonum %v %v %v %v", a, b, c1, s1, r1, z1, c2, s2, r2, z2)
			}
		}
	}
	type in struct{ d1, d2, x1, y1 float64 }
	cases := []in{{1, 1, 3, 4}, {-1, 1, 1, 1}, {0, 2, 1, 1}, {1, 0, 2, 3}, {1e-10, 1, 1, 1}, {1e10, 1e-10, 1, 1}, {1, 1e9, 1, 1}, {1, 1, 1, 3}}
	for i := 0; i < 300; i++ {
		cases = append(cases, in{r.ExpFloat64(), r.ExpFloat64() * math.Exp(10*r.NormFloat64()), r.NormFloat64(), r.NormFloat64()})
	}
	for _, c := range cases {
		pk, ak, bk, ck := keel.Drotmg(c.d1, c.d2, c.x1, c.y1)
		pg, ag, bg, cg := gonumImpl.Drotmg(c.d1, c.d2, c.x1, c.y1)
		if toKeel(pg) != pk || bits([]float64{ak, bk, ck}) != bits([]float64{ag, bg, cg}) {
			t.Errorf("Drotmg%v: keel %v %v %v %v, gonum %v %v %v %v", c, pk, ak, bk, ck, pg, ag, bg, cg)
		}
	}
}
