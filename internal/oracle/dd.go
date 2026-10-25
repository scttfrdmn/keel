// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package oracle

import "math"

// Double-double arithmetic: the wider-type arm DESIGN.md §5 rule 1 requires
// beside every float64 reference.
//
// A dd is an unevaluated sum hi + lo of two float64s with |lo| ≤ ulp(hi)/2, so
// it carries about 106 significant bits. That is what restores, at float64, the
// property float64 gives the float32 oracle: the reference's own error is
// ε_dd/ε₆₄ ≈ 2⁻⁵⁴ of the error under test, so a bound measured against it
// isolates keel's error instead of summing two.
//
// # Exactness, and where it ends
//
// twoSum, fastTwoSum and twoProd are error-free transformations: the pair they
// return sums to the exact result, not an approximation of it. Their conditions:
//
//   - twoSum is exact for all finite inputs, subnormals included, unless a+b
//     overflows. fastTwoSum is the same under |a| ≥ |b| (or a == 0).
//   - twoProd is exact unless a·b overflows or its error term falls below the
//     subnormal range — roughly |a·b| < 2⁻⁹⁶⁹. Below that the lo part is the
//     error rounded to a subnormal, so hi+lo misses a·b by at most eta64/2: the
//     same absolute underflow floor Tolerance64 already carries.
//
// The arithmetic ops (add, addF, mulF, mul, divF) are not exact; they are the
// algorithms of Joldes, Muller and Popescu, "Tight and rigorous error bounds for
// basic building blocks of double-word arithmetic" (ACM TOMS 2017), with their
// proved relative bounds, u = 2⁻⁵³: addF (DWPlusFP) 2u², add (AccurateDWPlusDW)
// 3u²+13u³ — the accurate variant, because the sloppy one has no relative bound
// under cancellation, which is the input this arm exists for — mulF (DWTimesFP3)
// 2u², mul (DWTimesDW3) 5u², divF (DWDivFP3) 3u². dd_test.go checks every one
// of them against math/big.Float.
//
// Range is not widened. A dd has float64's exponent range, so dd defends against
// lost digits and not against overflow: Dnrm2's reference has to scale for the
// same reason keel's must, where the float32 oracle got that for free.
//
// # Inf and NaN
//
// hi follows IEEE 754 for the operation applied to the hi parts — so Inf and
// NaN propagate as a float64 op would propagate them — and lo is then 0. The
// guard is needed, not decorative: an error-free transformation of an infinity
// computes Inf − Inf for its error term, and that NaN would otherwise be folded
// back into hi by the renormalization and turn 1e308·10 into NaN instead of Inf.
// The sign of a zero hi is IEEE's; the sign of a zero lo is unspecified.
//
// # The fusion barriers
//
// Every product that feeds an add or subtract is written float64(x*y). The Go
// spec lets an implementation fuse x*y + z into one FMA "possibly across
// statements", and an explicit conversion is the only thing that forbids it. The
// arm64 backend does fuse — it rewrites any FADDD whose operand is an FMULD,
// regardless of other uses of the product — and amd64 does not. A fused
// twoProd(a, b).hi feeding twoSum is no longer an error-free transformation, so
// without the barriers this file would be exact on amd64 and silently wrong on
// the dev host. The explicit math.FMA calls are the fusions that are wanted, and
// math.FMA is correctly rounded on every GOARCH.
//
// # Cost
//
// The dd mul-add a reference loop does per term (twoProd or mulF, then add) is
// about 20 float64 operations with a dependent chain through two renormalizations,
// so a dd reference costs roughly 10–20× its float64 twin. BenchmarkDotDD against
// BenchmarkDdot at n = 768 read 5.8 µs against 0.54 µs, about 11×, on the dev host
// (Apple M4 Pro, go1.27.3, 2026-10-24; an unqueued dev-tier reading, not a
// result), and a full 768³ GemmDD with its Dgemm twin took about 10 s there. That
// is a long-test cost, not a sweep-wide one, which is the trade DESIGN.md §5 rule
// 1 states.
type dd struct{ hi, lo float64 }

// finite reports whether s is neither infinite nor NaN.
func finite(s float64) bool { return s-s == 0 }

// twoSum returns s = fl(a+b) and e with s + e = a + b exactly (Knuth).
func twoSum(a, b float64) dd {
	s := a + b
	bb := s - a
	e := (a - (s - bb)) + (b - bb)
	return dd{s, e}
}

// fastTwoSum is twoSum for |a| ≥ |b| (Dekker): three operations instead of six.
// Every renormalization below calls it with that precondition established by the
// algorithm, never by the data.
func fastTwoSum(a, b float64) dd {
	s := a + b
	return dd{s, b - (s - a)}
}

// twoProd returns p = fl(a·b) and e with p + e = a·b exactly, under the
// conditions in the type comment. The barrier on p is what keeps p the rounded
// product when a caller adds it to something; see "The fusion barriers".
func twoProd(a, b float64) dd {
	p := float64(a * b)
	return dd{p, math.FMA(a, b, -p)}
}

// addF returns x + y for a dd x and a float64 y (DWPlusFP, 2u²).
func addF(x dd, y float64) dd {
	s := twoSum(x.hi, y)
	if !finite(s.hi) {
		return dd{s.hi, 0}
	}
	return fastTwoSum(s.hi, x.lo+s.lo)
}

// add returns x + y (AccurateDWPlusDW, 3u² + 13u³).
func add(x, y dd) dd {
	s := twoSum(x.hi, y.hi)
	if !finite(s.hi) {
		return dd{s.hi, 0}
	}
	t := twoSum(x.lo, y.lo)
	v := fastTwoSum(s.hi, s.lo+t.hi)
	return fastTwoSum(v.hi, t.lo+v.lo)
}

// mulF returns x · y for a dd x and a float64 y (DWTimesFP3, 2u²).
func mulF(x dd, y float64) dd {
	c := twoProd(x.hi, y)
	if !finite(c.hi) {
		return dd{c.hi, 0}
	}
	return fastTwoSum(c.hi, math.FMA(x.lo, y, c.lo))
}

// mul returns x · y (DWTimesDW3, 5u²).
func mul(x, y dd) dd {
	c := twoProd(x.hi, y.hi)
	if !finite(c.hi) {
		return dd{c.hi, 0}
	}
	tl := math.FMA(x.hi, y.lo, float64(x.lo*y.lo))
	return fastTwoSum(c.hi, c.lo+math.FMA(x.lo, y.hi, tl))
}

// divF returns x / y for a dd x and a float64 y (DWDivFP3, 3u²).
func divF(x dd, y float64) dd {
	th := x.hi / y
	if !finite(th) || !finite(y) { // a finite x over ±Inf is ±0, not twoProd's 0·Inf
		return dd{th, 0}
	}
	p := twoProd(th, y)
	d := (x.hi - p.hi) - p.lo // x.hi − p.hi is exact (Sterbenz)
	return fastTwoSum(th, (d+x.lo)/y)
}
