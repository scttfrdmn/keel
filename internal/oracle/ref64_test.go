// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// ddSizes are the reduction lengths the float64 references are checked against
// the double-double arm at: every small size, the MR/NR-ish neighbourhoods, both
// sides of MC=144 and KC=384, and up to 768 — the size DESIGN.md §5 rule 1 derives
// as the smallest crossing a KC boundary.
var ddSizes = []int{1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65,
	127, 128, 129, 143, 144, 145, 255, 256, 257, 383, 384, 385, 511, 512, 513, 767, 768}

func vec64(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = r.NormFloat64()
	}
	return v
}

// vec32 returns a float32 vector and its exact float64 image, so a float32
// reference and a float64 one can be run on the same values.
func vec32(r *rand.Rand, n int) ([]float32, []float64) {
	v, w := make([]float32, n), make([]float64, n)
	for i := range v {
		v[i] = float32(r.NormFloat64())
		w[i] = float64(v[i])
	}
	return v, w
}

func rn(b *big.Float) float64 { f, _ := b.Float64(); return f }

// within fails t for every element where |got − want| exceeds tol(n, scale).
func within(t *testing.T, name string, got, want, scale []float64, n int, tol func(int, float64) float64) {
	t.Helper()
	bad := 0
	for i := range want {
		if d := math.Abs(got[i] - want[i]); !(d <= tol(n, scale[i])) {
			if bad++; bad <= 3 {
				t.Errorf("%s[%d]: got %v want %v (|diff| %.3g > tol %.3g)", name, i, got[i], want[i], d, tol(n, scale[i]))
			}
		}
	}
	if bad > 3 {
		t.Errorf("%s: %d elements out of tolerance in all", name, bad)
	}
}

// TestDRefsAreUnfused pins the property the D-references were split from the
// float32 ones to have (ref64.go): every product and every sum is rounded on its
// own. The expected values are computed by simulating exactly that through
// big.Float, one correctly rounded operation at a time, and compared by bits — so
// a fused reference, on a host that fuses, fails here rather than being a
// different reference on a different GOARCH.
func TestDRefsAreUnfused(t *testing.T) {
	r := rand.New(rand.NewSource(10))
	const n = 64
	for trial := 0; trial < 200; trial++ {
		x, y := vec64(r, n), vec64(r, n)
		alpha := r.NormFloat64()

		var want float64
		for j := 0; j < n; j++ {
			want = rn(bnew().Add(bf(want), bf(rn(bnew().Mul(bf(x[j]), bf(y[j]))))))
		}
		if got, _ := Ddot(n, x, 1, y, 1); got != want {
			t.Fatalf("Ddot = %v, per-operation rounding gives %v", got, want)
		}

		// DgemmEntry over a 1×n A and an n×1 B: (alpha·a)·b, then the sum.
		want = 0
		for p := 0; p < n; p++ {
			ab := rn(bnew().Mul(bf(rn(bnew().Mul(bf(alpha), bf(x[p])))), bf(y[p])))
			want = rn(bnew().Add(bf(want), bf(ab)))
		}
		if got, _ := DgemmEntry(false, false, 0, 0, n, alpha, x, n, y, 1, 0, nil, 1); got != want {
			t.Fatalf("DgemmEntry = %v, per-operation rounding gives %v", got, want)
		}

		out, _ := Daxpy(n, alpha, x, 1, y, 1)
		sc, _ := Dscal(n, alpha, x, 1)
		for j := 0; j < n; j++ {
			ax := rn(bnew().Mul(bf(alpha), bf(x[j])))
			if w := rn(bnew().Add(bf(ax), bf(y[j]))); out[j] != w {
				t.Fatalf("Daxpy[%d] = %v, per-operation rounding gives %v", j, out[j], w)
			}
			if sc[j] != ax {
				t.Fatalf("Dscal[%d] = %v, want %v", j, sc[j], ax)
			}
		}

		// Dnrm2's power-of-two rescaling is exact in range, so it must equal the
		// unscaled per-operation sum of squares bit for bit.
		var ss float64
		for j := 0; j < n; j++ {
			ss = rn(bnew().Add(bf(ss), bf(rn(bnew().Mul(bf(x[j]), bf(x[j]))))))
		}
		if got, _ := Dnrm2(n, x, 1); got != math.Sqrt(ss) {
			t.Fatalf("Dnrm2 = %v, unscaled definition gives %v", got, math.Sqrt(ss))
		}
	}
}

// TestDnrm2Range checks the rescue the float32 oracle never needed: no type wider
// than float64 has a wider exponent range, so an unscaled sum of squares
// overflows or underflows where the norm does not.
func TestDnrm2Range(t *testing.T) {
	for _, c := range []struct {
		x     []float64
		naive float64 // what an unscaled float64 sum of squares returns
	}{
		{[]float64{1e200, -1e200, 3e200}, math.Inf(1)},
		{[]float64{1e-200, 2e-200, -2e-200}, 0},
		{[]float64{math.MaxFloat64, math.MaxFloat64 / 2}, math.Inf(1)},
		{[]float64{math.SmallestNonzeroFloat64, 0, math.SmallestNonzeroFloat64}, 0},
	} {
		var naive float64
		for _, v := range c.x {
			naive += v * v
		}
		if math.Sqrt(naive) != c.naive {
			t.Fatalf("%v: the unscaled sum gives %v, not the %v this case is about", c.x, math.Sqrt(naive), c.naive)
		}
		ss := bnew()
		for _, v := range c.x {
			ss.Add(ss, bnew().Mul(bf(v), bf(v)))
		}
		want := rn(bnew().Sqrt(ss))
		got, _ := Dnrm2(len(c.x), c.x, 1)
		if math.Abs(got-want) > Tolerance64(len(c.x), want) {
			t.Errorf("Dnrm2(%v) = %v, want %v", c.x, got, want)
		}
	}
	inf, nan := math.Inf(1), math.NaN()
	if v, _ := Dnrm2(3, []float64{1, -inf, 2}, 1); !math.IsInf(v, 1) {
		t.Errorf("Dnrm2 with -Inf = %v, want +Inf", v)
	}
	if v, _ := Dnrm2(3, []float64{inf, nan, 2}, 1); v == v {
		t.Errorf("Dnrm2 with NaN = %v, want NaN", v)
	}
	if v, _ := Dnrm2(0, nil, 1); v != 0 {
		t.Errorf("Dnrm2 of nothing = %v", v)
	}
}

// TestDRefsMatchFloat32Refs runs every float64 reference beside its float32 twin
// on float32-representable inputs, over every flag and a negative stride. Both
// compute in float64 and differ only by the fusion the float32 ones allow, so
// each pair is two float64 computations of one quantity: PeerTolerance64. This is
// the check that the duplicated bodies kept every convention — transpose, uplo,
// side, unit, the alpha/beta shortcuts, the index mapping — and Idamax's must be
// exactly Iamax's.
func TestDRefsMatchFloat32Refs(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	const m, n, k = 13, 11, 17
	const tal32, tal = float32(0.75), 0.75 // alpha for the triangular routines
	for _, ab := range [][2]float32{{0.75, -1.25}, {0, -1.25}, {0.75, 0}} {
		a32, a := vec32(r, 32*32)
		b32, b := vec32(r, 32*32)
		c32, c := vec32(r, 32*32)
		x32, x := vec32(r, 64)
		y32, y := vec32(r, 64)
		al32, be32 := ab[0], ab[1]
		al, be := float64(al32), float64(be32)
		tag := fmt.Sprintf("alpha=%v beta=%v", al, be)
		for _, inc := range []int{1, -2} {
			o, s := Daxpy(n, al, x, inc, y, 3)
			w, _ := Axpy(n, al32, x32, inc, y32, 3)
			within(t, "Daxpy "+tag, o, w, s, 2, PeerTolerance64)
			o, s = Dscal(n, al, x, inc)
			w, _ = Scal(n, al32, x32, inc)
			within(t, "Dscal "+tag, o, w, s, 1, PeerTolerance64)
			for _, tr := range []bool{false, true} {
				o, s = Dgemv(tr, m, n, al, a, 20, x, inc, be, y, -1)
				w, _ = Gemv(tr, m, n, al32, a32, 20, x32, inc, be32, y32, -1)
				within(t, "Dgemv "+tag, o, w, s, m+1, PeerTolerance64)
			}
			o, s = Dger(m, n, al, x, inc, y, 2, a, 20)
			w, _ = Ger(m, n, al32, x32, inc, y32, 2, a32, 20)
			within(t, "Dger "+tag, o, w, s, 2, PeerTolerance64)
		}
		for _, f := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
			o, s := Dgemm(f[0], f[1], m, n, k, al, a, 20, b, 21, be, c, 22)
			w, _ := Gemm(f[0], f[1], m, n, k, al32, a32, 20, b32, 21, be32, c32, 22)
			within(t, "Dgemm "+tag, o, w, s, k+1, PeerTolerance64)
			o, s = Dsyrk(f[0], f[1], n, k, al, a, 20, be, c, 22)
			w, _ = Syrk(f[0], f[1], n, k, al32, a32, 20, be32, c32, 22)
			within(t, "Dsyrk "+tag, o, w, s, k+1, PeerTolerance64)
			o, s = Dsymm(f[0], f[1], m, n, al, a, 20, b, 21, be, c, 22)
			w, _ = Symm(f[0], f[1], m, n, al32, a32, 20, b32, 21, be32, c32, 22)
			within(t, "Dsymm "+tag, o, w, s, m+1, PeerTolerance64)
			o, s = Dsyr2k(f[0], f[1], n, k, al, a, 20, b, 21, be, c, 22)
			w, _ = Syr2k(f[0], f[1], n, k, al32, a32, 20, b32, 21, be32, c32, 22)
			within(t, "Dsyr2k "+tag, o, w, s, 2*k+1, PeerTolerance64)
		}
	}
	// The triangular routines, on a diagonally dominant A so Trsm's recursive scale
	// stays meaningful (see Trsm).
	a32, a := vec32(r, 20*20)
	for i := 0; i < 20; i++ {
		a32[i*20+i] = float32(20 + i)
		a[i*20+i] = float64(a32[i*20+i])
	}
	b32, b := vec32(r, 20*20)
	for bits := 0; bits < 16; bits++ {
		left, lower, trans, unit := bits&1 != 0, bits&2 != 0, bits&4 != 0, bits&8 != 0
		tag := fmt.Sprintf("left=%v lower=%v trans=%v unit=%v", left, lower, trans, unit)
		o, s := Dtrsm(left, lower, trans, unit, m, n, tal, a, 20, b, 20)
		w, _ := Trsm(left, lower, trans, unit, m, n, tal32, a32, 20, b32, 20)
		within(t, "Dtrsm "+tag, o, w, s, m+1, PeerTolerance64)
		o, s = Dtrmm(left, lower, trans, unit, m, n, tal, a, 20, b, 20)
		w, _ = Trmm(left, lower, trans, unit, m, n, tal32, a32, 20, b32, 20)
		within(t, "Dtrmm "+tag, o, w, s, m+1, PeerTolerance64)
	}

	for _, xs := range [][]float32{
		{1, -3, 3, 2}, {float32(math.NaN()), 5, 1}, {1, float32(math.NaN()), 5}, {0, 0}, {-0.5},
	} {
		w := make([]float64, len(xs))
		for i, v := range xs {
			w[i] = float64(v)
		}
		for _, inc := range []int{1, -1} {
			if g, want := Idamax(len(xs), w, inc), Iamax(len(xs), xs, inc); g != want {
				t.Errorf("Idamax(%v, inc %d) = %d, Iamax gives %d", xs, inc, g, want)
			}
		}
	}
	if Idamax(0, nil, 1) != -1 {
		t.Error("Idamax(0) != -1")
	}

	x32, x := vec32(r, 40)
	y32, y := vec32(r, 40)
	for _, inc := range []int{1, -3} {
		g, s := Ddot(13, x, inc, y, 2)
		w, _ := Dot(13, x32, inc, y32, 2)
		within(t, "Ddot", []float64{g}, []float64{w}, []float64{s}, 13, PeerTolerance64)
		g, s = Dasum(13, x, inc)
		w, _ = Asum(13, x32, inc)
		within(t, "Dasum", []float64{g}, []float64{w}, []float64{s}, 13, PeerTolerance64)
		g, s = Dnrm2(13, x, inc)
		w, _ = Nrm2(13, x32, inc)
		within(t, "Dnrm2", []float64{g}, []float64{w}, []float64{s}, 13, PeerTolerance64)
	}
}

// TestDotDDCorrectlyRounded checks the double-double dot against the exact dot in
// big.Float. On random data the dd error is ~2⁻⁵³ of an ulp, so the result must
// be the correctly rounded exact value — every time, not within a tolerance. On
// cancellation-heavy data it must be within half an ulp plus the dd error bound
// stated in dd_ref.go.
func TestDotDDCorrectlyRounded(t *testing.T) {
	r := rand.New(rand.NewSource(12))
	exact := func(n int, x []float64, incX int, y []float64, incY int) *big.Float {
		s := bnew()
		ix, iy := Index(n, incX), Index(n, incY)
		for j := 0; j < n; j++ {
			s.Add(s, bnew().Mul(bf(x[ix+j*incX]), bf(y[iy+j*incY])))
		}
		return s
	}
	for _, n := range ddSizes {
		x, y := vec64(r, 2*n), vec64(r, 3*n)
		for _, inc := range [][2]int{{1, 1}, {2, -3}} {
			got, _ := DotDD(n, x, inc[0], y, inc[1])
			if want := rn(exact(n, x, inc[0], y, inc[1])); got != want {
				t.Errorf("DotDD n=%d inc=%v = %v, correctly rounded exact is %v", n, inc, got, want)
			}
		}
		// Cancellation: y's second half undoes x's first half up to a rounding.
		xc := append(append([]float64{}, x[:n]...), x[:n]...)
		yc := append(append([]float64{}, y[:n]...), y[:n]...)
		for j := n; j < 2*n; j++ {
			yc[j] = -yc[j] * (1 + 0x1p-40*r.NormFloat64())
		}
		got, scale := DotDD(2*n, xc, 1, yc, 1)
		ex := exact(2*n, xc, 1, yc, 1)
		w := rn(ex)
		bound := u*math.Abs(w) + float64(2*n+1)*5*u*u*scale
		if d := rn(bnew().Abs(bnew().Sub(bf(got), ex))); d > bound {
			t.Errorf("DotDD cancellation n=%d: error %.3g > bound %.3g", 2*n, d, bound)
		}
	}
}

// TestDRefsVsDD is the self-test DESIGN.md §5 rule 1 asks of the float64 arm: the
// definition-order references must meet their own ONE-error bound, Tolerance64,
// against the double-double ones, at every size in ddSizes up to 768. If a
// float64 reference failed this it would be the reference that is wrong, and the
// doubled peer bound would have been hiding it.
func TestDRefsVsDD(t *testing.T) {
	r := rand.New(rand.NewSource(13))
	const al, be = 0.75, -1.25
	for _, n := range ddSizes {
		x, y := vec64(r, 2*n), vec64(r, 3*n)
		g, _ := Ddot(n, x, 2, y, -3)
		w, s := DotDD(n, x, 2, y, -3)
		within(t, fmt.Sprintf("Ddot n=%d", n), []float64{g}, []float64{w}, []float64{s}, n, Tolerance64)

		// Gemv over an n×n A with a padded lda, both transposes.
		a := vec64(r, n*(n+3))
		for _, tr := range []bool{false, true} {
			o, _ := Dgemv(tr, n, n, al, a, n+3, x, 2, be, y, -3)
			want, sc := GemvDD(tr, n, n, al, a, n+3, x, 2, be, y, -3)
			within(t, fmt.Sprintf("Dgemv trans=%v n=%d", tr, n), o, want, sc, n+1, Tolerance64)
		}

		// Gemm at m = n = k: a seeded sample of entries per transpose pair, which
		// at k = 768 is where the reduction crosses KC.
		b, c := vec64(r, n*(n+1)), vec64(r, n*(n+2))
		for _, f := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
			for e := 0; e < 24; e++ {
				i, j := r.Intn(n), r.Intn(n)
				o, _ := DgemmEntry(f[0], f[1], i, j, n, al, a, n+3, b, n+1, be, c, n+2)
				want, sc := GemmEntryDD(f[0], f[1], i, j, n, al, a, n+3, b, n+1, be, c, n+2)
				within(t, fmt.Sprintf("DgemmEntry %v n=%d (%d,%d)", f, n, i, j),
					[]float64{o}, []float64{want}, []float64{sc}, n+1, Tolerance64)
			}
		}
	}
}

// TestL3DRefsVsDD checks the Level-3 float64 references that have no dd twin of
// their own against GemmDD on operands the test materializes independently — the
// symmetric matrix filled out, op(A) of a triangle made dense, Syr2k's two
// products concatenated into one of length 2k — so the expected values do not
// come from the index logic under test.
func TestL3DRefsVsDD(t *testing.T) {
	r := rand.New(rand.NewSource(14))
	const al, be = 0.75, -1.25
	for _, d := range [][3]int{{1, 1, 1}, {5, 7, 3}, {17, 9, 33}, {40, 33, 65}} {
		m, n, k := d[0], d[1], d[2]
		for bits := 0; bits < 4; bits++ {
			f0, f1 := bits&1 != 0, bits&2 != 0
			tag := fmt.Sprintf("%v %v m=%d n=%d k=%d", f0, f1, m, n, k)

			// Syrk: C = αA·Aᵀ (or αAᵀ·A) is Gemm with B = A, transposed the other way.
			lower, trans := f0, f1
			a, c := vec64(r, 70*70), vec64(r, 70*70)
			o, _ := Dsyrk(lower, trans, n, k, al, a, 70, be, c, 70)
			want, sc := GemmDD(trans, !trans, n, n, k, al, a, 70, a, 70, be, c, 70)
			mask(lower, n, want, sc)
			within(t, "Dsyrk "+tag, o, want, sc, k+1, Tolerance64)

			// Syr2k: concatenate [A B] against [B A] (side by side, or stacked when
			// trans), so A·Bᵀ + B·Aᵀ is one product of length 2k.
			b := vec64(r, 70*70)
			l, rr := make([]float64, 2*k*n), make([]float64, 2*k*n)
			for i := 0; i < n; i++ {
				for p := 0; p < k; p++ {
					if trans { // A, B are k×n; L, R are 2k×n
						l[p*n+i], l[(k+p)*n+i] = a[p*70+i], b[p*70+i]
						rr[p*n+i], rr[(k+p)*n+i] = b[p*70+i], a[p*70+i]
					} else { // A, B are n×k; L, R are n×2k
						l[i*2*k+p], l[i*2*k+k+p] = a[i*70+p], b[i*70+p]
						rr[i*2*k+p], rr[i*2*k+k+p] = b[i*70+p], a[i*70+p]
					}
				}
			}
			ld := 2 * k
			if trans {
				ld = n
			}
			o, _ = Dsyr2k(lower, trans, n, k, al, a, 70, b, 70, be, c, 70)
			want, sc = GemmDD(trans, !trans, n, n, 2*k, al, l, ld, rr, ld, be, c, 70)
			mask(lower, n, want, sc)
			within(t, "Dsyr2k "+tag, o, want, sc, 2*k+1, Tolerance64)

			// Symm: fill the symmetric matrix out from its stored triangle.
			left := f0
			lower = f1
			dim := n
			if left {
				dim = m
			}
			s := make([]float64, dim*dim)
			for i := 0; i < dim; i++ {
				for j := 0; j <= i; j++ {
					v := a[i*70+j] // the lower triangle, as stored
					if !lower {
						v = a[j*70+i] // the upper one
					}
					s[i*dim+j], s[j*dim+i] = v, v
				}
			}
			o, _ = Dsymm(left, lower, m, n, al, a, 70, b, 70, be, c, 70)
			if left {
				want, sc = GemmDD(false, false, m, n, m, al, s, dim, b, 70, be, c, 70)
			} else {
				want, sc = GemmDD(false, false, m, n, n, al, b, 70, s, dim, be, c, 70)
			}
			within(t, "Dsymm "+tag, o, want, sc, dim+1, Tolerance64)

			// Trmm: make the stored triangle dense — zeros off it, ones on a unit
			// diagonal — then transpose it if asked.
			for _, unit := range []bool{false, true} {
				for _, tr := range []bool{false, true} {
					tri := make([]float64, dim*dim)
					for i := 0; i < dim; i++ {
						for j := 0; j < dim; j++ {
							if (lower && j <= i) || (!lower && j >= i) {
								tri[i*dim+j] = a[i*70+j]
							}
						}
						if unit {
							tri[i*dim+i] = 1
						}
					}
					o, _ = Dtrmm(left, lower, tr, unit, m, n, al, a, 70, b, 70)
					if left {
						want, sc = GemmDD(tr, false, m, n, m, al, tri, dim, b, 70, 0, nil, 0)
					} else {
						want, sc = GemmDD(false, tr, m, n, n, al, b, 70, tri, dim, 0, nil, 0)
					}
					within(t, fmt.Sprintf("Dtrmm unit=%v trans=%v %s", unit, tr, tag), o, want, sc, dim, Tolerance64)
				}
			}
		}
	}
}

// mask zeroes the entries outside the referenced triangle, which Syrk and Syr2k
// leave zero and GemmDD computes.
func mask(lower bool, n int, v ...[]float64) {
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !InTriangle(lower, i, j) {
				for _, s := range v {
					s[i*n+j] = 0
				}
			}
		}
	}
}

// TestDgemmVsGemmDDFull768 is the whole-matrix form at the size DESIGN.md
// derives, 768³: every entry, not a sample. It is the long test the dd arm's cost
// makes it (about 10 s on the dev host; see dd.go) — so -short skips it.
func TestDgemmVsGemmDDFull768(t *testing.T) {
	if testing.Short() {
		t.Skip("768³ double-double Gemm is a long test")
	}
	r := rand.New(rand.NewSource(15))
	const n = 768
	a, b, c := vec64(r, n*n), vec64(r, n*n), vec64(r, n*n)
	o, _ := Dgemm(false, true, n, n, n, 0.75, a, n, b, n, -1.25, c, n)
	want, sc := GemmDD(false, true, n, n, n, 0.75, a, n, b, n, -1.25, c, n)
	within(t, "Dgemm 768³", o, want, sc, n+1, Tolerance64)
}

// TestDDSeesWhatPeerCannot is the positive control DESIGN.md §5 rule 1's dd arm
// owes: a case the double-double arm sees and the peer float64 reference cannot.
//
// Part 1 is a defect both sides share. x·y = 2⁵³ + 1 + 1 + … + 1 − 2⁵³ in that
// order: every +1 is lost to rounding (2⁵³ + 1 ties to 2⁵³), so a subject summing
// in the definition order returns 0 against an exact m. Ddot shares the order and
// returns the same 0, so the peer comparison measures an error of exactly zero for
// a result with no correct digit; DotDD returns m and measures the true error.
//
// It is also within Tolerance64, and the test says so rather than hiding it:
// against Σ|terms| ≈ 2⁵⁴ the bound admits ~16(m+2), more than m. That is not a
// flaw of this case but a property of the model — any summation order is within
// γₙ·Σ|terms|, an eighth of Tolerance64 — so an accumulation-ORDER defect alone
// cannot redden against either arm; what the dd arm adds at the verdict level is
// Part 2.
//
// Part 2 is the band the doubled peer bound opens: a subject whose error is
// 1.5·Tolerance64 — over its own budget — is caught against DotDD and passes
// against Ddot under PeerTolerance64. The error is planted, and says so; what is
// under test is the harness's ability to tell the two arms apart.
func TestDDSeesWhatPeerCannot(t *testing.T) {
	const m = 40
	x := make([]float64, m+2)
	y := make([]float64, m+2)
	for i := range x {
		x[i], y[i] = 1, 1
	}
	x[0], x[m+1] = 0x1p53, -0x1p53
	subject := 0.0 // the shared definition-order loop, written again
	for i := range x {
		subject += float64(x[i] * y[i])
	}
	peer, _ := Ddot(m+2, x, 1, y, 1)
	exact, scale := DotDD(m+2, x, 1, y, 1)
	if subject != 0 || peer != 0 || exact != m {
		t.Fatalf("construction: subject %v, Ddot %v, DotDD %v; want 0, 0, %d", subject, peer, exact, m)
	}
	if e := math.Abs(subject - peer); e != 0 {
		t.Errorf("peer arm measured %v, want 0: the shared defect is supposed to be invisible to it", e)
	}
	if e := math.Abs(subject - exact); e != m {
		t.Errorf("dd arm measured %v, want the true error %d", e, m)
	}
	if tol := Tolerance64(m+2, scale); !(m < tol) {
		t.Errorf("Tolerance64 = %v no longer admits the shared defect's error %d; update the comment", tol, m)
	}

	r := rand.New(rand.NewSource(16))
	const n = 768
	xs, ys := vec64(r, n), vec64(r, n)
	pv, _ := Ddot(n, xs, 1, ys, 1)
	dv, ds := DotDD(n, xs, 1, ys, 1)
	planted := pv + 1.5*Tolerance64(n, ds)
	if d := math.Abs(planted - dv); !(d > Tolerance64(n, ds)) {
		t.Errorf("dd arm: planted error %.3g not caught at Tolerance64 %.3g", d, Tolerance64(n, ds))
	}
	if d := math.Abs(planted - pv); !(d <= PeerTolerance64(n, ds)) {
		t.Errorf("peer arm: planted error %.3g caught at PeerTolerance64 %.3g; the blind band this control shows is gone", d, PeerTolerance64(n, ds))
	}
}

// TestTolerance64 pins the model's constants and its peer factor, and that the
// float32 model is untouched.
func TestTolerance64(t *testing.T) {
	if got, want := Tolerance64(1, 1), 4*(0x1p-52+0x1p-1075); got != want {
		t.Errorf("Tolerance64(1, 1) = %v, want %v", got, want)
	}
	if got, want := Tolerance64(0, 0), 4*0x1p-1075; got != want {
		t.Errorf("Tolerance64(0, 0) = %v, want %v (the floor, n clamped to 1)", got, want)
	}
	if got, want := PeerTolerance64(768, 3), 2*Tolerance64(768, 3); got != want {
		t.Errorf("PeerTolerance64 = %v, want %v", got, want)
	}
	if got := Tolerance(1, 1); got != 4*(1.1920929e-07+1.401298464e-45/2) {
		t.Errorf("Tolerance(1, 1) = %v moved", got)
	}
}

// BenchmarkDdot and BenchmarkDotDD measure the dd arm's cost ratio at the size
// DESIGN.md derives for it.
func BenchmarkDdot(b *testing.B)  { benchDot(b, Ddot) }
func BenchmarkDotDD(b *testing.B) { benchDot(b, DotDD) }

var sink float64

func benchDot(b *testing.B, f func(int, []float64, int, []float64, int) (float64, float64)) {
	r := rand.New(rand.NewSource(17))
	x, y := vec64(r, 768), vec64(r, 768)
	for b.Loop() {
		v, _ := f(768, x, 1, y, 1)
		sink += v
	}
}
