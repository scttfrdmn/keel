// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import "math"

// The copy, swap and rotation routines of Level 1 at float64 (#192): l1rot.go's
// routines at float64, including its rounding barrier on every product, which keeps
// the rotations unfused and so bit-identical on every architecture (Go may fuse an
// unbarriered c*vx+s*vy at float64 as at float32). Drotg's safe scaling takes
// float64's safmin. See l1rot.go for everything else.

// Dcopy copies x into y.
func Dcopy(n int, x []float64, incX int, y []float64, incY int) {
	checkVector64("Dcopy", "x", n, x, incX)
	checkVector64("Dcopy", "y", n, y, incY)
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

// Dswap exchanges x and y.
func Dswap(n int, x []float64, incX int, y []float64, incY int) {
	checkVector64("Dswap", "x", n, x, incX)
	checkVector64("Dswap", "y", n, y, incY)
	if n == 0 {
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		x[px], y[py] = y[py], x[px]
	}
}

// Drot applies the plane rotation (c, s) to the pairs (x_i, y_i):
// x_i = c·x_i + s·y_i and y_i = c·y_i − s·x_i.
func Drot(n int, x []float64, incX int, y []float64, incY int, c, s float64) {
	checkVector64("Drot", "x", n, x, incX)
	checkVector64("Drot", "y", n, y, incY)
	if n == 0 {
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		vx, vy := x[px], y[py]
		x[px], y[py] = float64(c*vx)+float64(s*vy), float64(c*vy)-float64(s*vx)
	}
}

// Drotg constructs the plane rotation that zeroes b: it returns c, s and r with
//
//	[ c  s ] [ a ]   [ r ]
//	[-s  c ] [ b ] = [ 0 ]
//
// and z, the reference BLAS value from which c and s can be recovered. It uses
// Anderson's safe scaling (ACM TOMS Algorithm 978), as current reference BLAS and
// gonum do, so no intermediate overflows or underflows. r itself can: it is
// sqrt(a²+b²), which overflows to ±Inf when that exceeds the float64 range (a =
// b = 1.5e308), and when it is subnormal it carries only a few bits, which c = a/r and
// s = b/r then inherit.
func Drotg(a, b float64) (c, s, r, z float64) {
	const (
		safmin = 0x1p-1022
		safmax = 1 / safmin
	)
	anorm, bnorm := math.Abs(a), math.Abs(b)
	switch {
	case bnorm == 0:
		return 1, 0, a, 0
	case anorm == 0:
		return 0, 1, b, 1
	}
	maxab := max(anorm, bnorm)
	scl := min(max(safmin, maxab), safmax)
	sigma := math.Copysign(1, b)
	if anorm > bnorm {
		sigma = math.Copysign(1, a)
	}
	ascl, bscl := a/scl, b/scl
	r = sigma * (scl * math.Sqrt(ascl*ascl+bscl*bscl))
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

// The Rotm* flags as float64 values, for DPARAM(1). The exported constants stay
// typed float64 (l1rot.go), so no existing caller's inferred type changes.
const (
	rotmIdentity64    = float64(RotmIdentity)
	rotmRescaling64   = float64(RotmRescaling)
	rotmOffDiagonal64 = float64(RotmOffDiagonal)
	rotmDiagonal64    = float64(RotmDiagonal)
)

// Drotm applies the modified Givens transformation H to the pairs (x_i, y_i):
// [x_i; y_i] = H·[x_i; y_i]. param is reference BLAS's DPARAM: param[0] is the
// flag (RotmIdentity, RotmRescaling, RotmOffDiagonal or RotmDiagonal) and
// param[1:5] are h11, h21, h12, h22, of which only those the flag names are read.
// Any other flag panics.
func Drotm(n int, x []float64, incX int, y []float64, incY int, param [5]float64) {
	checkVector64("Drotm", "x", n, x, incX)
	checkVector64("Drotm", "y", n, y, incY)
	flag := param[0]
	switch flag {
	case rotmIdentity64, rotmRescaling64, rotmOffDiagonal64, rotmDiagonal64:
	default:
		panic("keel: Drotm: param[0] is not a valid flag")
	}
	if n == 0 || flag == rotmIdentity64 {
		return
	}
	h11, h21, h12, h22 := param[1], param[2], param[3], param[4]
	ix, iy := offset(n, incX), offset(n, incY)
	for i := 0; i < n; i++ {
		px, py := ix+i*incX, iy+i*incY
		vx, vy := x[px], y[py]
		switch flag {
		case rotmRescaling64:
			x[px], y[py] = float64(vx*h11)+float64(vy*h12), float64(vx*h21)+float64(vy*h22)
		case rotmOffDiagonal64:
			x[px], y[py] = vx+float64(vy*h12), float64(vx*h21)+vy
		case rotmDiagonal64:
			x[px], y[py] = float64(vx*h11)+vy, -vx+float64(vy*h22)
		}
	}
}

// Drotmg constructs the modified Givens transformation that zeroes the second
// component of (sqrt(d1)·x1, sqrt(d2)·y1). It returns the DPARAM array for Drotm
// and the updated d1, d2 and x1. A negative d1, or a configuration with no valid
// transformation, returns the RotmRescaling flag with H and the three values all
// zero, which is reference BLAS's error state. It follows Hopkins (1997), Appendix
// A, with gamma = 4096 for the rescaling, as gonum does; H elements a flag does not
// use are returned as zero.
func Drotmg(d1, d2, x1, y1 float64) (param [5]float64, rd1, rd2, rx1 float64) {
	const (
		gam    = 4096.0
		gamsq  = gam * gam
		rgamsq = 1.0 / gamsq
	)
	if d1 < 0 {
		param[0] = rotmRescaling64
		return param, 0, 0, 0
	}
	if d2 == 0 || y1 == 0 {
		param[0] = rotmIdentity64
		return param, d1, d2, x1
	}
	var flag, h11, h12, h21, h22 float64
	if (d1 == 0 || x1 == 0) && d2 > 0 {
		flag = rotmDiagonal64
		h12, h21 = 1, -1
		x1 = y1
		d1, d2 = d2, d1
	} else {
		p2 := d2 * y1
		p1 := d1 * x1
		q2 := p2 * y1
		q1 := p1 * x1
		if math.Abs(q1) > math.Abs(q2) {
			flag = rotmOffDiagonal64
			h11, h22 = 1, 1
			h21 = -y1 / x1
			h12 = p2 / p1
			u := 1 - float64(h12*h21)
			if u <= 0 {
				param[0] = rotmRescaling64
				return param, 0, 0, 0
			}
			d1 /= u
			d2 /= u
			x1 *= u
		} else {
			if q2 < 0 {
				param[0] = rotmRescaling64
				return param, 0, 0, 0
			}
			flag = rotmDiagonal64
			h21, h12 = -1, 1
			h11 = p1 / p2
			h22 = x1 / y1
			u := 1 + float64(h11*h22)
			d1, d2 = d2/u, d1/u
			x1 = y1 * u
		}
	}
	for d1 <= rgamsq && d1 != 0 {
		flag = rotmRescaling64
		d1 = (d1 * gam) * gam
		x1 /= gam
		h11 /= gam
		h12 /= gam
	}
	for d1 > gamsq {
		flag = rotmRescaling64
		d1 = (d1 / gam) / gam
		x1 *= gam
		h11 *= gam
		h12 *= gam
	}
	for math.Abs(d2) <= rgamsq && d2 != 0 {
		flag = rotmRescaling64
		d2 = (d2 * gam) * gam
		h21 /= gam
		h22 /= gam
	}
	for math.Abs(d2) > gamsq {
		flag = rotmRescaling64
		d2 = (d2 / gam) / gam
		h21 *= gam
		h22 *= gam
	}
	param[0] = flag
	switch flag {
	case rotmDiagonal64:
		param[1], param[4] = h11, h22
	case rotmOffDiagonal64:
		param[2], param[3] = h21, h12
	case rotmRescaling64:
		param[1], param[2], param[3], param[4] = h11, h21, h12, h22
	}
	return param, d1, d2, x1
}
