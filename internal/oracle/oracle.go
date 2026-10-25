// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package oracle holds float64 reference implementations of every public
// routine (P1 onward) and the single tolerance model from DESIGN.md §5.
//
// The references are written for obviousness, not speed: straight sequential
// loops accumulating in float64, doing exactly what the BLAS definition says.
// They are never called from keel's hot paths — only from tests — so the right
// trade is always "easier to audit" over "faster".
//
// Each reduction returns both its value and the *scale* its error bound should
// be measured against, because those two belong together. The bound for a
// float32 dot product is proportional to Σ|xᵢyᵢ| (see Tolerance for the full
// expression), and that Σ|xᵢyᵢ| is not derivable
// from the returned value — a dot product with heavy cancellation has a tiny
// result and a large error bound. Returning the scale from the same loop that
// computed the value is what stops a test from quietly substituting |result|
// and calling a wrong answer correct.
//
// For float64 routines there is no wider hardware type, so the package carries
// two of DESIGN.md §5 rule 1's three float64 arms: definition-order float64
// references (ref64.go, peer precision, compared under PeerTolerance64) and
// double-double references (dd.go, dd_ref.go, isolating, compared under
// Tolerance64). The third arm, Gonum, is an independent implementation rather
// than a reference and does not live here.
package oracle

import "math"

// Tolerance returns the allowed absolute error for a length-n reduction with
// the given magnitude scale:
//
//	C·f(n)·(eps32·scale + eta32/2)
//
// It is the ONLY place epsilons live; tests must not carry their own.
//
// # The two terms
//
// The first is the familiar relative one: each of the f(n) roundings can move
// the result by eps32 times the magnitude being accumulated, so the total drift
// is bounded by f(n)·eps32·scale. C is slack for FMA and reassociation.
//
// The second is the underflow floor, and it is not optional. Rounding error in
// binary floating point is only relatively bounded above the smallest
// representable magnitude; below it, a value rounds to zero and the error is
// absolute. A float32 dot product of elements near 1e-25 has products near
// 1e-50 — under eta32 = 1.4e-45, the smallest subnormal — so every product
// rounds to zero and the routine returns exactly 0. That 0 is the correctly
// rounded float32 answer, and the float64 oracle's 6e-49 is not representable
// in float32 at all, so a bound with only a relative term calls a
// correctly-rounded result a failure. Each rounding contributes at most
// eta32/2, and there are f(n) of them.
//
// The floor term is around 1e-44 for realistic n, i.e. some thirty orders of
// magnitude below anything the relative term admits for a scale near 1. It
// changes nothing for ordinary data and cannot be used to hide a real error;
// it only stops the model from asserting an accuracy float32 cannot express.
//
// Added 2026-08-10 after the `tiny` data pattern in l1_test.go failed against
// the relative-only model with results that were, on inspection, exact.
func Tolerance(n int, scale float64) float64 {
	const eps32 = 1.1920929e-07   // 2^-23, float32 unit roundoff
	const eta32 = 1.401298464e-45 // 2^-149, smallest positive float32 subnormal
	const c = 4.0                 // slack for FMA/reassociation
	fn := float64(n)
	if fn < 1 {
		fn = 1
	}
	return c * fn * (eps32*scale + eta32/2)
}

// Tolerance64 is Tolerance for a float64 routine:
//
//	C·f(n)·(eps64·scale + eta64/2)
//
// with eps64 = 2⁻⁵² and eta64 = 2⁻¹⁰⁷⁴, and the same C = 4 and f(n) = n.
//
// # The derivation, at float64
//
// It is Tolerance's derivation with the type's two constants substituted, and
// neither term changes meaning. A length-n reduction rounds f(n) = n times; each
// rounding moves the result by at most eps64 times the magnitude being
// accumulated, so the relative drift is bounded by n·eps64·scale (the standard
// γₙ·Σ|terms| bound, written with eps64 rather than the unit roundoff 2⁻⁵³, which
// is a factor of two of headroom both models already carry). C is the same slack
// for FMA and reassociation: an FMA removes a rounding and a reassociated sum
// rounds the same number of times, so neither raises the count, and neither
// depends on the type. Below 2⁻¹⁰²² the error is absolute, eta64/2 per rounding,
// for exactly the reason the float32 floor exists — a correctly rounded
// subnormal or zero result must not read as a failure.
//
// # What it is the bound OF, and the factor of two
//
// Tolerance64 bounds ONE implementation's distance from the exact answer. That is
// directly the comparison bound only against a reference whose own error is
// negligible beside it — the double-double references (DotDD, GemvDD, GemmDD),
// whose error is about 2⁻⁵³ of this (dd_ref.go). Against the float64
// definition-order references (Ddot, Dgemm, …) it is not: those are
// peer-precision, each side is within Tolerance64 of exact, and the triangle
// inequality gives |keel − reference| ≤ 2·Tolerance64. That doubled bound is
// PeerTolerance64, stated as its own function so the factor lives here and never
// in a test (DESIGN.md §5 rule 1).
func Tolerance64(n int, scale float64) float64 {
	const eps64 = 2.220446049250313e-16   // 2^-52, float64 machine epsilon
	const eta64 = 4.9406564584124654e-324 // 2^-1074, smallest positive float64 subnormal
	const c = 4.0                         // slack for FMA/reassociation
	fn := float64(n)
	if fn < 1 {
		fn = 1
	}
	// Not c*fn*(eps64*scale + eta64/2), though it is the same quantity: eta64/2 is
	// 2^-1075, which float64 cannot hold, and it rounds (ties to even) to 0. The
	// literal port of Tolerance's expression therefore had no floor at all, and
	// TestTolerance64 is what caught it. Halving after the multiply keeps it.
	return c*fn*eps64*scale + c*fn/2*eta64
}

// PeerTolerance64 is the comparison bound between two float64 computations of
// the same quantity, each bounded by Tolerance64: their errors add, so it is
// 2·Tolerance64. It is the bound for keel against a float64 definition-order
// reference, and it is the sum of two errors rather than a measurement of one —
// a defect both sides share cancels inside it, and an error up to twice keel's
// own budget can hide in it. That blind spot is what the double-double arm,
// compared under Tolerance64, exists to close.
func PeerTolerance64(n int, scale float64) float64 {
	return 2 * Tolerance64(n, scale)
}

// Index returns the offset of a strided BLAS vector's first element. For
// inc < 0 a BLAS vector runs backwards from the far end, so element j lives at
// index Index(n, inc) + j*inc for every j in [0,n).
func Index(n, inc int) int {
	if inc > 0 {
		return 0
	}
	return -(n - 1) * inc
}

// Dot returns the float64 inner product and the scale Σ|xᵢyᵢ| for its bound.
func Dot(n int, x []float32, incX int, y []float32, incY int) (val, scale float64) {
	ix, iy := Index(n, incX), Index(n, incY)
	for j := 0; j < n; j++ {
		p := float64(x[ix+j*incX]) * float64(y[iy+j*incY])
		val += p
		scale += math.Abs(p)
	}
	return val, scale
}

// Asum returns Σ|xᵢ| in float64. Its own value is the right error scale: every
// term is non-negative, so there is no cancellation to hide behind.
func Asum(n int, x []float32, incX int) (val, scale float64) {
	ix := Index(n, incX)
	for j := 0; j < n; j++ {
		val += math.Abs(float64(x[ix+j*incX]))
	}
	return val, val
}

// Nrm2 returns the Euclidean norm in float64.
//
// No scaling tricks are needed here even for float32 inputs that would
// overflow a float32 sum of squares: the largest float32 squared is ~1.2e77,
// comfortably inside float64's range, which is precisely why the oracle is
// float64 and keel's own Snrm2 needs a rescue path that this does not.
func Nrm2(n int, x []float32, incX int) (val, scale float64) {
	ix := Index(n, incX)
	var sumsq float64
	for j := 0; j < n; j++ {
		v := float64(x[ix+j*incX])
		sumsq += v * v
	}
	val = math.Sqrt(sumsq)
	return val, val
}

// Axpy returns the reference result of y += alpha*x as a float64 slice indexed
// by element (not by storage offset), along with the per-element error scale
// |alpha·xⱼ| + |yⱼ|.
func Axpy(n int, alpha float32, x []float32, incX int, y []float32, incY int) (out, scale []float64) {
	ix, iy := Index(n, incX), Index(n, incY)
	out = make([]float64, n)
	scale = make([]float64, n)
	a := float64(alpha)
	for j := 0; j < n; j++ {
		ax := a * float64(x[ix+j*incX])
		yj := float64(y[iy+j*incY])
		out[j] = ax + yj
		scale[j] = math.Abs(ax) + math.Abs(yj)
	}
	return out, scale
}

// Scal returns the reference result of x *= alpha, indexed by element.
func Scal(n int, alpha float32, x []float32, incX int) (out, scale []float64) {
	ix := Index(n, incX)
	out = make([]float64, n)
	scale = make([]float64, n)
	a := float64(alpha)
	for j := 0; j < n; j++ {
		out[j] = a * float64(x[ix+j*incX])
		scale[j] = math.Abs(out[j])
	}
	return out, scale
}

// Iamax returns the 0-based index of the first element of greatest magnitude,
// or -1 when n < 1.
//
// This mirrors reference BLAS ISAMAX's sequential comparison chain rather than
// an idiomatic max, because the chain is what defines the NaN behaviour: a NaN
// first seeds the running maximum and then defeats every later `>` comparison,
// so a leading NaN wins and a NaN anywhere else is ignored. Written as `>` on
// magnitudes, in order, so the oracle and internal/l1.Iamax are the same
// algorithm stated twice — deliberately, since agreement between two
// independent statements of a convention is the only evidence that either
// captured it.
func Iamax(n int, x []float32, incX int) int {
	if n < 1 {
		return -1
	}
	ix := Index(n, incX)
	best, bi := math.Abs(float64(x[ix])), 0
	for j := 1; j < n; j++ {
		if v := math.Abs(float64(x[ix+j*incX])); v > best {
			best, bi = v, j
		}
	}
	return bi
}
