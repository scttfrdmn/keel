// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// exactPrec is the big.Float precision every "exact" value here is computed at.
//
// It is 4500 bits and not the 300 that would be ample for an ordinary dd value,
// because a dd's two parts need not be close in exponent: twoSum(1e300, 1e-300)
// is exact, and representing its sum takes about 2000 bits. Below the span of
// the operands, big.Float rounds and the comparison stops being one. 4500 covers
// any sum of float64s (exponents 2¹⁰²³ down to 2⁻¹⁰⁷⁴) and any product of two
// such sums, so every value a test below calls exact is.
const exactPrec = 4500

const u = 0x1p-53 // unit roundoff of float64

func bf(x float64) *big.Float { return new(big.Float).SetPrec(exactPrec).SetFloat64(x) }

func bnew() *big.Float { return new(big.Float).SetPrec(exactPrec) }

// bdd returns the exact value hi + lo of a dd.
func bdd(x dd) *big.Float { return bnew().Add(bf(x.hi), bf(x.lo)) }

// relErr returns |got − want| / |want| as a float64, or |got − want| when want is
// zero.
func relErr(got, want *big.Float) float64 {
	d := bnew().Sub(got, want)
	d.Abs(d)
	if want.Sign() != 0 {
		w := bnew().Abs(want)
		d.Quo(d, w)
	}
	f, _ := d.Float64()
	return f
}

// randF returns a float64 with a uniformly random 53-bit significand, a random
// sign, and a binary exponent uniform in [emin, emax].
func randF(r *rand.Rand, emin, emax int) float64 {
	m := float64(r.Uint64()>>11 | 1<<52) // 53 bits, top bit set
	v := math.Ldexp(m, emin+r.Intn(emax-emin+1)-52)
	if r.Intn(2) == 0 {
		v = -v
	}
	return v
}

// randDD returns a normalized dd whose hi is randF(emin, emax) and whose lo sits
// a random 53 to 110 bits below it, so the parts overlap anywhere from not at all
// to barely — both ends of what a normalized dd can hold.
func randDD(r *rand.Rand, emin, emax int) dd {
	hi := randF(r, emin, emax)
	lo := math.Ldexp(randF(r, 0, 0), math.Ilogb(hi)-53-r.Intn(58))
	return fastTwoSum(hi, lo)
}

// TestErrorFreeTransforms checks that twoSum, fastTwoSum and twoProd are exact:
// hi is the correctly rounded result and hi + lo is the exact one, against
// big.Float at exactPrec — on random operands spanning the exponent range and on
// the adversarial ones listed.
func TestErrorFreeTransforms(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	checkSum := func(name string, a, b float64, got dd) {
		t.Helper()
		want := bnew().Add(bf(a), bf(b))
		rn, _ := want.Float64()
		if got.hi != rn || bdd(got).Cmp(want) != 0 {
			t.Errorf("%s(%g, %g) = {%g, %g}: not exact (want hi %g)", name, a, b, got.hi, got.lo, rn)
		}
	}
	checkProd := func(a, b float64) {
		t.Helper()
		got := twoProd(a, b)
		want := bnew().Mul(bf(a), bf(b))
		rn, _ := want.Float64()
		if got.hi != rn || bdd(got).Cmp(want) != 0 {
			t.Errorf("twoProd(%g, %g) = {%g, %g}: not exact (want hi %g)", a, b, got.hi, got.lo, rn)
		}
	}

	// Random: twoSum across the whole finite range (exponents kept below the one
	// where a + b could overflow, and down into the subnormals); twoProd where the
	// product stays above 2^-969, the floor of its exactness.
	for i := 0; i < 20000; i++ {
		a, b := randF(r, -1074, 1022), randF(r, -1074, 1022)
		checkSum("twoSum", a, b, twoSum(a, b))
		if math.Abs(a) < math.Abs(b) {
			a, b = b, a
		}
		checkSum("fastTwoSum", a, b, fastTwoSum(a, b))
		checkProd(randF(r, -480, 480), randF(r, -480, 480))
	}

	// Adversarial sums: cancellation to one ulp and to nothing, the largest finite
	// values cancelling, subnormals (where twoSum stays exact), and the two parts
	// at opposite ends of the exponent range.
	sub := math.SmallestNonzeroFloat64
	for _, p := range [][2]float64{
		{1, -1 + 0x1p-53}, {1 + 0x1p-52, -1}, {0x1p60, -(0x1p60 - 1)}, {3, -3},
		{math.MaxFloat64, -math.MaxFloat64}, {math.MaxFloat64, -math.MaxFloat64 / 3},
		{sub, sub}, {sub, -2 * sub}, {0x1p-1022, -sub}, {0x1p-1022 - sub, sub},
		{1e300, 1e-300}, {-1e308, 5e-324}, {1, 0x1p-1074},
	} {
		checkSum("twoSum", p[0], p[1], twoSum(p[0], p[1]))
		checkSum("twoSum", p[1], p[0], twoSum(p[1], p[0]))
		a, b := p[0], p[1]
		if math.Abs(a) < math.Abs(b) {
			a, b = b, a
		}
		checkSum("fastTwoSum", a, b, fastTwoSum(a, b))
	}
	for _, p := range [][2]float64{
		{1 + 0x1p-52, 1 - 0x1p-53}, {math.Sqrt2, math.Sqrt2}, {0x1p511 + 1, 0x1p511 - 1},
		{-(1 + 0x1p-52), 0x1p-480 * (1 + 0x1p-51)},
	} {
		checkProd(p[0], p[1])
	}

	// Signed zeros: hi carries IEEE's sign.
	for _, z := range []struct {
		a, b float64
		neg  bool
	}{{0, 0, false}, {math.Copysign(0, -1), math.Copysign(0, -1), true}, {0, math.Copysign(0, -1), false}} {
		if s := twoSum(z.a, z.b); s.hi != 0 || s.lo != 0 || math.Signbit(s.hi) != z.neg {
			t.Errorf("twoSum(%g, %g) = {%g, %g}, want hi a zero with signbit %v", z.a, z.b, s.hi, s.lo, z.neg)
		}
	}
	if p := twoProd(math.Copysign(0, -1), 3); p.hi != 0 || !math.Signbit(p.hi) {
		t.Errorf("twoProd(-0, 3).hi = %g, want -0", p.hi)
	}
}

// TestTwoProdUnderflowFloor pins the one place twoProd is not exact, so the claim
// in dd.go is witnessed rather than stated, and so the exactness check above is
// known to be able to fail: a product near 2^-1000 whose error term is near
// 2^-1104, which no float64 holds. hi is still the correctly rounded product, and
// hi + lo misses the exact product by no more than eta64/2.
func TestTwoProdUnderflowFloor(t *testing.T) {
	a, b := 1+0x1p-52, 0x1p-1000*(1+0x1p-52)
	got := twoProd(a, b)
	want := bnew().Mul(bf(a), bf(b))
	rn, _ := want.Float64()
	if got.hi != rn {
		t.Fatalf("twoProd hi = %g, want correctly rounded %g", got.hi, rn)
	}
	if bdd(got).Cmp(want) == 0 {
		t.Fatalf("twoProd(%g, %g) reads exact; its error term is below the subnormal range "+
			"and cannot be, so the exactness comparison is not seeing what it claims to", a, b)
	}
	miss := bnew().Sub(bdd(got), want)
	if f, _ := miss.Abs(miss).Float64(); f > 0x1p-1075 {
		t.Errorf("twoProd underflow miss %g exceeds eta64/2", f)
	}
}

// TestDDOpBounds checks each dd arithmetic op against its published relative
// error bound (dd.go), on random operands and on cancellation, against big.Float.
// The worst observed error, in units of u², is logged: it is the margin under the
// bound, and the literature's bounds are known to be nearly attained.
func TestDDOpBounds(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	type op struct {
		name  string
		bound float64 // in u²
		run   func(x, y dd) (got dd, exact *big.Float)
	}
	ops := []op{
		{"addF", 2, func(x, y dd) (dd, *big.Float) { return addF(x, y.hi), bnew().Add(bdd(x), bf(y.hi)) }},
		{"add", 3 + 13*u, func(x, y dd) (dd, *big.Float) { return add(x, y), bnew().Add(bdd(x), bdd(y)) }},
		{"mulF", 2, func(x, y dd) (dd, *big.Float) { return mulF(x, y.hi), bnew().Mul(bdd(x), bf(y.hi)) }},
		{"mul", 5, func(x, y dd) (dd, *big.Float) { return mul(x, y), bnew().Mul(bdd(x), bdd(y)) }},
		{"divF", 3, func(x, y dd) (dd, *big.Float) { return divF(x, y.hi), bnew().Quo(bdd(x), bf(y.hi)) }},
	}
	for _, o := range ops {
		worst := 0.0
		check := func(x, y dd) {
			got, exact := o.run(x, y)
			if e := relErr(bdd(got), exact) / (u * u); e > worst {
				worst = e
			}
			if math.Abs(got.lo) > math.Abs(got.hi)*u {
				t.Errorf("%s(%v, %v) = %v: not normalized", o.name, x, y, got)
			}
		}
		for i := 0; i < 20000; i++ {
			// Exponents kept to ±300 so no intermediate of mul or div underflows:
			// the bounds are relative and hold only above the subnormal range.
			x, y := randDD(r, -300, 300), randDD(r, -300, 300)
			check(x, y)
			// Cancellation: y's hi is −x's hi, or one ulp from it, so the result
			// is all low-order parts — the input the accurate add exists for.
			c := dd{-x.hi, y.lo * math.Ldexp(1, math.Ilogb(x.hi)-math.Ilogb(y.hi))}
			check(x, fastTwoSum(c.hi, c.lo))
			check(x, fastTwoSum(math.Nextafter(-x.hi, 0), c.lo))
		}
		if worst > o.bound {
			t.Errorf("%s: worst relative error %.3g u² exceeds its bound %.3g u²", o.name, worst, o.bound)
		}
		t.Logf("%s: worst %.3g u² (bound %.3g u²)", o.name, worst, o.bound)
	}
}

// TestDDSubnormal checks add and addF where every operand is subnormal or near
// it. The relative bound does not apply below 2^-1022, but twoSum is exact there,
// so the sum of two subnormal dd values is exact and that is what is asserted.
func TestDDSubnormal(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < 5000; i++ {
		x := dd{randF(r, -1074, -1020), 0}
		y := dd{randF(r, -1074, -1020), 0}
		want := bnew().Add(bdd(x), bdd(y))
		if got := add(x, y); bdd(got).Cmp(want) != 0 {
			t.Fatalf("add(%v, %v) = %v, want exact %v", x, y, got, want)
		}
		if got := addF(x, y.hi); bdd(got).Cmp(want) != 0 {
			t.Fatalf("addF(%v, %g) = %v, want exact %v", x, y.hi, got, want)
		}
	}
}

// TestDDNonFinite pins the propagation rule in dd.go: hi is what the float64 op
// on the hi parts would give, and lo is 0.
func TestDDNonFinite(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	one := dd{1, 0}
	cases := []struct {
		name string
		got  dd
		want float64
	}{
		{"add(Inf, 1)", add(dd{inf, 0}, one), inf},
		{"add(1, -Inf)", add(one, dd{-inf, 0}), -inf},
		{"add(Inf, -Inf)", add(dd{inf, 0}, dd{-inf, 0}), nan},
		{"add(NaN, 1)", add(dd{nan, 0}, one), nan},
		{"add overflow", add(dd{math.MaxFloat64, 0}, dd{math.MaxFloat64, 0}), inf},
		{"addF(Inf, 1)", addF(dd{inf, 0}, 1), inf},
		{"mul overflow", mul(dd{1e308, 0}, dd{10, 0}), inf},
		{"mul(Inf, 0)", mul(dd{inf, 0}, dd{0, 0}), nan},
		{"mulF(-Inf, 2)", mulF(dd{-inf, 0}, 2), -inf},
		{"mulF(NaN, 2)", mulF(dd{nan, 0}, 2), nan},
		{"divF(1, 0)", divF(one, 0), inf},
		{"divF(-1, 0)", divF(dd{-1, 0}, 0), -inf},
		{"divF(0, 0)", divF(dd{0, 0}, 0), nan},
		{"divF(1, Inf)", divF(one, inf), 0},
	}
	for _, c := range cases {
		ok := c.got.hi == c.want || (c.want != c.want && c.got.hi != c.got.hi)
		if !ok || c.got.lo != 0 {
			t.Errorf("%s = {%g, %g}, want {%g, 0}", c.name, c.got.hi, c.got.lo, c.want)
		}
	}
}
