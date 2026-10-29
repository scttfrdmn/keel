// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"math"

	"github.com/scttfrdmn/keel/internal/l164"
)

// Level 1 at float64 (#192). Each routine is its S twin in keel.go at float64:
// the same argument checks and the same panics, the same split between the active
// backend's unit-stride kernel (internal/l164, generated from internal/l1) and a
// scalar strided loop here, and the same documented deviations from reference BLAS.
// The doc comments state only what differs; the S routine's comment is the
// rationale for the rest. TestDParity pins each pair to the same behaviour on bad
// arguments.
//
// Hand-written, not generated (ruled 2026-10-29, docs/f64-design.md): these bodies
// are thin, and Dnrm2 differs from Snrm2 in its algorithm, not only its type.

// Ddot returns the inner product of x and y; see Sdot.
func Ddot(n int, x []float64, incX int, y []float64, incY int) float64 {
	checkVector64("Ddot", "x", n, x, incX)
	checkVector64("Ddot", "y", n, y, incY)
	if n == 0 {
		return 0
	}
	if incX == 1 && incY == 1 {
		return activeL164.Dot(x[:n], y[:n])
	}
	ix, iy := offset(n, incX), offset(n, incY)
	var a0, a1 float64
	j := 0
	for ; j+2 <= n; j += 2 {
		a0 += x[ix+j*incX] * y[iy+j*incY]
		a1 += x[ix+(j+1)*incX] * y[iy+(j+1)*incY]
	}
	for ; j < n; j++ {
		a0 += x[ix+j*incX] * y[iy+j*incY]
	}
	return a0 + a1
}

// Daxpy computes y += alpha*x; see Saxpy, including why alpha == 0 leaves y alone.
func Daxpy(n int, alpha float64, x []float64, incX int, y []float64, incY int) {
	checkVector64("Daxpy", "x", n, x, incX)
	checkVector64("Daxpy", "y", n, y, incY)
	if n == 0 || alpha == 0 {
		return
	}
	if incX == 1 && incY == 1 {
		activeL164.Axpy(alpha, x[:n], y[:n])
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for j := 0; j < n; j++ {
		y[iy+j*incY] += alpha * x[ix+j*incX]
	}
}

// Dscal computes x *= alpha; see Sscal on why there is no alpha == 0 shortcut.
func Dscal(n int, alpha float64, x []float64, incX int) {
	checkVectorPos64("Dscal", "x", n, x, incX)
	if n == 0 {
		return
	}
	if incX == 1 {
		activeL164.Scal(alpha, x[:n])
		return
	}
	for j := 0; j < n; j++ {
		x[j*incX] *= alpha
	}
}

// Dasum returns the sum of magnitudes of x.
func Dasum(n int, x []float64, incX int) float64 {
	checkVectorPos64("Dasum", "x", n, x, incX)
	if n == 0 {
		return 0
	}
	if incX == 1 {
		return activeL164.Asum(x[:n])
	}
	var a0, a1 float64
	j := 0
	for ; j+2 <= n; j += 2 {
		a0 += math.Abs(x[j*incX])
		a1 += math.Abs(x[(j+1)*incX])
	}
	for ; j < n; j++ {
		a0 += math.Abs(x[j*incX])
	}
	return a0 + a1
}

// Dnrm2 returns the Euclidean norm of x.
//
// # The same design as Snrm2, with a different rescue
//
// The fast path is Snrm2's: the active kernel's unguarded sum of squares, whose
// result is inspected rather than each element guarded. The acceptance bound is
// Snrm2's derivation at float64. Each term v·v is rounded to a multiple of the
// smallest subnormal eta = 2^-1074, so n terms carry at most n·eta/2 of absolute
// error, and the relative error in s is below eps = 2^-52 when s >= n·2^-1021.
// Taking it one binade stricter at n·2^-1022, n times the smallest normal float64,
// bounds it at eps/2. A non-finite s (overflow, or NaN in the data) and s == 0 are
// rescued too.
//
// Snrm2 rescues by rerunning in float64, which has room to spare for squared
// float32s. Nothing wider than float64 is available, so Dnrm2 rescues itself the way
// internal/oracle's Dnrm2 does. It takes max|xᵢ| first, then scales every element by
// 2^-e, with 2^e just above that maximum, before squaring. A power-of-two scale only
// moves the exponent, so it is exact, except for elements so far below the maximum
// that they become subnormal. Their squares are then below eta relative to the sum.
// The result is unscaled by the same power. A NaN anywhere gives NaN, and otherwise
// an infinity gives +Inf, as the unscaled sum of squares would.
func Dnrm2(n int, x []float64, incX int) float64 {
	checkVectorPos64("Dnrm2", "x", n, x, incX)
	if n == 0 {
		return 0
	}
	var s float64
	if incX == 1 {
		s = activeL164.SumSq(x[:n])
	} else {
		var a0, a1 float64
		j := 0
		for ; j+2 <= n; j += 2 {
			v0, v1 := x[j*incX], x[(j+1)*incX]
			a0 += v0 * v0
			a1 += v1 * v1
		}
		for ; j < n; j++ {
			v := x[j*incX]
			a0 += v * v
		}
		s = a0 + a1
	}
	// NaN fails the comparison, so it lands in the rescue, as in Snrm2.
	if s >= float64(n)*0x1p-1022 && !math.IsInf(s, 1) {
		return math.Sqrt(s)
	}
	return nrm2Scaled(n, x, incX)
}

// nrm2Scaled is Dnrm2's rescue: the power-of-two-scaled two-pass norm derived in
// Dnrm2's comment.
func nrm2Scaled(n int, x []float64, incX int) float64 {
	var m float64
	nan := false
	for j := 0; j < n; j++ {
		v := math.Abs(x[j*incX])
		if v != v {
			nan = true
		} else if v > m {
			m = v
		}
	}
	switch {
	case nan:
		return math.NaN()
	case math.IsInf(m, 1):
		return m
	case m == 0:
		return 0
	}
	_, e := math.Frexp(m)
	var sum float64
	for j := 0; j < n; j++ {
		v := math.Ldexp(x[j*incX], -e)
		sum += v * v
	}
	return math.Ldexp(math.Sqrt(sum), e)
}

// Idamax returns the 0-based index of the first element of greatest magnitude, or
// -1 when n == 0; see Isamax for both deviations from Fortran and the NaN
// convention.
func Idamax(n int, x []float64, incX int) int {
	checkVectorPos64("Idamax", "x", n, x, incX)
	if n == 0 {
		return -1
	}
	if incX == 1 {
		return l164.Iamax(x[:n])
	}
	best, bi := math.Abs(x[0]), 0
	for j := 1; j < n; j++ {
		if v := math.Abs(x[j*incX]); v > best {
			best, bi = v, j
		}
	}
	return bi
}

// checkVector64 is checkVector for a float64 vector, message for message.
func checkVector64(fn, arg string, n int, v []float64, inc int) {
	if n < 0 {
		panic("keel: " + fn + ": n < 0")
	}
	if inc == 0 {
		panic("keel: " + fn + ": inc" + arg + " == 0")
	}
	if n == 0 {
		return // empty vector; v may be nil
	}
	span := inc
	if span < 0 {
		span = -span
	}
	if need := (n-1)*span + 1; len(v) < need {
		panic("keel: " + fn + ": " + arg + " shorter than n and inc" + arg + " require")
	}
}

// checkVectorPos64 is checkVectorPos for a float64 vector, message for message.
func checkVectorPos64(fn, arg string, n int, v []float64, inc int) {
	if inc <= 0 {
		if n < 0 {
			panic("keel: " + fn + ": n < 0")
		}
		panic("keel: " + fn + ": inc" + arg + " <= 0 (not defined for this routine)")
	}
	checkVector64(fn, arg, n, v, inc)
}
