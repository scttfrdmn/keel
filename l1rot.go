// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import "math"

// The copy, swap and rotation routines of Level 1. They are memory-bound, so they
// are plain loops rather than vector kernels: one load and one store per element,
// which the hardware bounds before arithmetic does. Strides follow every other
// Level-1 routine here: either may be negative, and the vector then runs backwards
// from its far end, as in reference BLAS; a zero stride panics.
//
// Every product in the rotations is rounded with float32(...) before it is added, so
// no multiply-add is fused and the results are bit-identical on every architecture.
// Srotm, Srotg and Srotmg are then bit-identical to gonum too, which writes them the
// same way. Srot is not: gonum writes c*vx+s*vy unrounded, and which product the Go
// compiler fuses on arm64 depends on the surrounding loop, so gonum's own unit-stride
// and strided paths can disagree in the last bit. keel's Srot agrees with gonum
// within the tolerance model instead.

// Scopy copies x into y.
func Scopy(n int, x []float32, incX int, y []float32, incY int) {
	checkVector("Scopy", "x", n, x, incX)
	checkVector("Scopy", "y", n, y, incY)
	if n == 0 {
		return
	}
	if incX == 1 && incY == 1 {
		copy(y[:n], x[:n])
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		y[iy+i*incY] = x[ix+i*incX]
	}
}

// Sswap exchanges x and y.
func Sswap(n int, x []float32, incX int, y []float32, incY int) {
	checkVector("Sswap", "x", n, x, incX)
	checkVector("Sswap", "y", n, y, incY)
	if n == 0 {
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		x[px], y[py] = y[py], x[px]
	}
}

// Srot applies the plane rotation (c, s) to the pairs (x_i, y_i):
// x_i = c·x_i + s·y_i and y_i = c·y_i − s·x_i.
func Srot(n int, x []float32, incX int, y []float32, incY int, c, s float32) {
	checkVector("Srot", "x", n, x, incX)
	checkVector("Srot", "y", n, y, incY)
	if n == 0 {
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		vx, vy := x[px], y[py]
		x[px], y[py] = float32(c*vx)+float32(s*vy), float32(c*vy)-float32(s*vx)
	}
}

// Srotg constructs the plane rotation that zeroes b: it returns c, s and r with
//
//	[ c  s ] [ a ]   [ r ]
//	[-s  c ] [ b ] = [ 0 ]
//
// and z, the reference BLAS value from which c and s can be recovered. It uses
// Anderson's safe scaling (ACM TOMS Algorithm 978), as current reference BLAS and
// gonum do, so no intermediate overflows or underflows. r itself can: it is
// sqrt(a²+b²), which overflows to ±Inf when that exceeds the float32 range (a =
// b = 3e38), and when it is subnormal it carries only a few bits, which c = a/r and
// s = b/r then inherit.
func Srotg(a, b float32) (c, s, r, z float32) {
	const (
		safmin = 0x1p-126
		safmax = 1 / safmin
	)
	anorm, bnorm := abs32(a), abs32(b)
	switch {
	case bnorm == 0:
		return 1, 0, a, 0
	case anorm == 0:
		return 0, 1, b, 1
	}
	maxab := max(anorm, bnorm)
	scl := min(max(safmin, maxab), safmax)
	sigma := float32(math.Copysign(1, float64(b)))
	if anorm > bnorm {
		sigma = float32(math.Copysign(1, float64(a)))
	}
	ascl, bscl := a/scl, b/scl
	r = sigma * (scl * sqrt32(ascl*ascl+bscl*bscl))
	c, s = a/r, b/r
	switch {
	case anorm > bnorm:
		z = s
	case c != 0:
		z = 1 / c
	default:
		z = 1
	}
	return c, s, r, z
}

// Modified-rotation flags, as reference BLAS's SPARAM(1) encodes them: which
// elements of H are meaningful.
const (
	RotmIdentity    float32 = -2 // H is the identity; nothing is applied
	RotmRescaling   float32 = -1 // H is full: h11, h21, h12, h22 all used
	RotmOffDiagonal float32 = 0  // h11 = h22 = 1 implied; h21 and h12 used
	RotmDiagonal    float32 = 1  // h21 = -1 and h12 = 1 implied; h11 and h22 used
)

// Srotm applies the modified Givens transformation H to the pairs (x_i, y_i):
// [x_i; y_i] = H·[x_i; y_i]. param is reference BLAS's SPARAM: param[0] is the
// flag (RotmIdentity, RotmRescaling, RotmOffDiagonal or RotmDiagonal) and
// param[1:5] are h11, h21, h12, h22, of which only those the flag names are read.
// Any other flag panics.
func Srotm(n int, x []float32, incX int, y []float32, incY int, param [5]float32) {
	checkVector("Srotm", "x", n, x, incX)
	checkVector("Srotm", "y", n, y, incY)
	flag := param[0]
	switch flag {
	case RotmIdentity, RotmRescaling, RotmOffDiagonal, RotmDiagonal:
	default:
		panic("keel: Srotm: param[0] is not a valid flag")
	}
	if n == 0 || flag == RotmIdentity {
		return
	}
	h11, h21, h12, h22 := param[1], param[2], param[3], param[4]
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		vx, vy := x[px], y[py]
		switch flag {
		case RotmRescaling:
			x[px], y[py] = float32(vx*h11)+float32(vy*h12), float32(vx*h21)+float32(vy*h22)
		case RotmOffDiagonal:
			x[px], y[py] = vx+float32(vy*h12), float32(vx*h21)+vy
		case RotmDiagonal:
			x[px], y[py] = float32(vx*h11)+vy, -vx+float32(vy*h22)
		}
	}
}

// Srotmg constructs the modified Givens transformation that zeroes the second
// component of (sqrt(d1)·x1, sqrt(d2)·y1). It returns the SPARAM array for Srotm
// and the updated d1, d2 and x1. A negative d1, or a configuration with no valid
// transformation, returns the RotmRescaling flag with H and the three values all
// zero, which is reference BLAS's error state. It follows Hopkins (1997), Appendix
// A, with gamma = 4096 for the rescaling, as gonum does; H elements a flag does not
// use are returned as zero.
func Srotmg(d1, d2, x1, y1 float32) (param [5]float32, rd1, rd2, rx1 float32) {
	const (
		gam    = 4096.0
		gamsq  = gam * gam
		rgamsq = 1.0 / gamsq
	)
	if d1 < 0 {
		param[0] = RotmRescaling
		return param, 0, 0, 0
	}
	if d2 == 0 || y1 == 0 {
		param[0] = RotmIdentity
		return param, d1, d2, x1
	}
	var flag, h11, h12, h21, h22 float32
	if (d1 == 0 || x1 == 0) && d2 > 0 {
		flag = RotmDiagonal
		h12, h21 = 1, -1
		x1 = y1
		d1, d2 = d2, d1
	} else {
		p2 := d2 * y1
		p1 := d1 * x1
		q2 := p2 * y1
		q1 := p1 * x1
		if abs32(q1) > abs32(q2) {
			flag = RotmOffDiagonal
			h11, h22 = 1, 1
			h21 = -y1 / x1
			h12 = p2 / p1
			u := 1 - float32(h12*h21)
			if u <= 0 {
				param[0] = RotmRescaling
				return param, 0, 0, 0
			}
			d1 /= u
			d2 /= u
			x1 *= u
		} else {
			if q2 < 0 {
				param[0] = RotmRescaling
				return param, 0, 0, 0
			}
			flag = RotmDiagonal
			h21, h12 = -1, 1
			h11 = p1 / p2
			h22 = x1 / y1
			u := 1 + float32(h11*h22)
			d1, d2 = d2/u, d1/u
			x1 = y1 * u
		}
	}
	for d1 <= rgamsq && d1 != 0 {
		flag = RotmRescaling
		d1 = (d1 * gam) * gam
		x1 /= gam
		h11 /= gam
		h12 /= gam
	}
	for d1 > gamsq {
		flag = RotmRescaling
		d1 = (d1 / gam) / gam
		x1 *= gam
		h11 *= gam
		h12 *= gam
	}
	for abs32(d2) <= rgamsq && d2 != 0 {
		flag = RotmRescaling
		d2 = (d2 * gam) * gam
		h21 /= gam
		h22 /= gam
	}
	for abs32(d2) > gamsq {
		flag = RotmRescaling
		d2 = (d2 / gam) / gam
		h21 *= gam
		h22 *= gam
	}
	param[0] = flag
	switch flag {
	case RotmDiagonal:
		param[1], param[4] = h11, h22
	case RotmOffDiagonal:
		param[2], param[3] = h21, h12
	case RotmRescaling:
		param[1], param[2], param[3], param[4] = h11, h21, h12, h22
	}
	return param, d1, d2, x1
}

func abs32(v float32) float32 { return math.Float32frombits(math.Float32bits(v) &^ (1 << 31)) }

// sqrt32 is the correctly rounded float32 square root: a float64 sqrt rounded to
// float32 is exact here, since 53 >= 2·24 + 2.
func sqrt32(v float32) float32 { return float32(math.Sqrt(float64(v))) }
