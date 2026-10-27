// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

import "simd/archsimd"

// NEON float64 backend (128-bit Float64x2, two lanes), #191: vec_neon.go's
// float32 shim restated for float64, with the element in every name (Load128D,
// Block64) as vec_avx512_f64.go does. Every archsimd identifier below was copied
// from `GOEXPERIMENT=simd GOARCH=arm64 go doc simd/archsimd Float64x2` on
// go1.27.3 and $(go env GOROOT)/src/simd/archsimd/*_arm64.go, not recalled.
// vec_neon.go's three arm64 facts carry over unchanged: MulAdd is VFMLA
// accumulating in place, Abs is a native VFABS, Max/Min are compare+select.

// F64x2 is the NEON float64 vector, 128 bits / two lanes.
type F64x2 = archsimd.Float64x2

// ---------------------------------------------------------------- hot layer

// Load128D loads two float64 from s; requires len(s) >= 2 (panics otherwise).
func Load128D(s []float64) F64x2 { return archsimd.LoadFloat64x2(s) }

// LoadPart128D loads min(len(s), 2) elements and zero-fills the rest.
func LoadPart128D(s []float64) F64x2 { v, _ := archsimd.LoadFloat64x2Part(s); return v }

// Store128D stores both lanes into s; requires len(s) >= 2.
func Store128D(s []float64, x F64x2) { x.Store(s) }

// StorePart128D stores as many lanes as fit in s.
func StorePart128D(s []float64, x F64x2) { _ = x.StorePart(s) }

// Broadcast128D returns v in both lanes (VDUP).
func Broadcast128D(v float64) F64x2 { return archsimd.BroadcastFloat64x2(v) }

// Zero128D returns the all-zero vector (+0 in both lanes).
func Zero128D() F64x2 { var z F64x2; return z }

// Add128D returns x+y lanewise.
func Add128D(x, y F64x2) F64x2 { return x.Add(y) }

// Sub128D returns x-y lanewise.
func Sub128D(x, y F64x2) F64x2 { return x.Sub(y) }

// Mul128D returns x*y lanewise.
func Mul128D(x, y F64x2) F64x2 { return x.Mul(y) }

// Div128D returns x/y lanewise. Asm: VFDIV, IEEE on a zero divisor.
func Div128D(x, y F64x2) F64x2 { return x.Div(y) }

// FMA128D returns x*y+z lanewise with a single rounding (VFMLA), the argument
// order of FMA512D and FMA128.
func FMA128D(x, y, z F64x2) F64x2 { return x.MulAdd(y, z) }

// Max128D / Min128D are `x>y ? x : y` and `x<y ? x : y` — ScalarDMax/ScalarDMin
// — from a compare and a select, not native FMAX/FMIN, for the NaN and ±0
// reasons Max128 gives. Less is y.Greater(x) in archsimd's arm64 source, so
// both are VFCMGT, false on unordered and on +0 vs -0.
func Max128D(x, y F64x2) F64x2 { return x.IfElse(x.Greater(y), y) }
func Min128D(x, y F64x2) F64x2 { return x.IfElse(x.Less(y), y) }

// Abs128D clears the sign bit of both lanes (VFABS).
func Abs128D(x F64x2) F64x2 { return x.Abs() }

// HSum128D sums the two lanes: the final-pair step of ScalarDHSum's tree.
func HSum128D(x F64x2) float64 {
	var a [2]float64
	x.StoreArray(&a)
	return a[0] + a[1]
}

// --------------------------------------------------------------- test layer
// Block64-in, Block64-out, the scalar spec's signatures: an 8-lane Block64 is
// four Float64x2 quarters. These round-trip through memory; not for hot loops.

func neonQuartersD(b Block64) (q0, q1, q2, q3 F64x2) {
	return Load128D(b[0:2]), Load128D(b[2:4]), Load128D(b[4:6]), Load128D(b[6:8])
}

func neonBlockD(q0, q1, q2, q3 F64x2) Block64 {
	var b Block64
	Store128D(b[0:2], q0)
	Store128D(b[2:4], q1)
	Store128D(b[4:6], q2)
	Store128D(b[6:8], q3)
	return b
}

func NEONDLoad(s []float64) Block64 {
	return neonBlockD(Load128D(s[0:2]), Load128D(s[2:4]), Load128D(s[4:6]), Load128D(s[6:8]))
}

// NEONDLoadPart loads min(len(s), Lanes64) elements, zero-filling the rest, as
// NEONLoadPart does per quarter.
func NEONDLoadPart(s []float64) Block64 {
	n := min(len(s), Lanes64)
	var q [4]F64x2
	for i := range q {
		lo := i * 2
		if lo >= n {
			q[i] = Zero128D()
			continue
		}
		q[i] = LoadPart128D(s[lo:min(lo+2, n)])
	}
	return neonBlockD(q[0], q[1], q[2], q[3])
}

func NEONDStore(s []float64, x Block64) {
	q0, q1, q2, q3 := neonQuartersD(x)
	Store128D(s[0:2], q0)
	Store128D(s[2:4], q1)
	Store128D(s[4:6], q2)
	Store128D(s[6:8], q3)
}

// NEONDStorePart stores min(len(s), Lanes64) lanes, matching ScalarDStorePart.
func NEONDStorePart(s []float64, x Block64) {
	n := min(len(s), Lanes64)
	var q [4]F64x2
	q[0], q[1], q[2], q[3] = neonQuartersD(x)
	for i := range q {
		lo := i * 2
		if lo >= n {
			break
		}
		StorePart128D(s[lo:min(lo+2, n)], q[i])
	}
}

func NEONDBroadcast(v float64) Block64 {
	q := Broadcast128D(v)
	return neonBlockD(q, q, q, q)
}

func NEONDZero() Block64 { return neonBlockD(Zero128D(), Zero128D(), Zero128D(), Zero128D()) }

func neonBinopD(x, y Block64, op func(a, b F64x2) F64x2) Block64 {
	x0, x1, x2, x3 := neonQuartersD(x)
	y0, y1, y2, y3 := neonQuartersD(y)
	return neonBlockD(op(x0, y0), op(x1, y1), op(x2, y2), op(x3, y3))
}

func NEONDAdd(x, y Block64) Block64 { return neonBinopD(x, y, Add128D) }
func NEONDSub(x, y Block64) Block64 { return neonBinopD(x, y, Sub128D) }
func NEONDMul(x, y Block64) Block64 { return neonBinopD(x, y, Mul128D) }
func NEONDDiv(x, y Block64) Block64 { return neonBinopD(x, y, Div128D) }
func NEONDMax(x, y Block64) Block64 { return neonBinopD(x, y, Max128D) }
func NEONDMin(x, y Block64) Block64 { return neonBinopD(x, y, Min128D) }

func NEONDMulAdd(x, y, z Block64) Block64 {
	x0, x1, x2, x3 := neonQuartersD(x)
	y0, y1, y2, y3 := neonQuartersD(y)
	z0, z1, z2, z3 := neonQuartersD(z)
	return neonBlockD(FMA128D(x0, y0, z0), FMA128D(x1, y1, z1), FMA128D(x2, y2, z2), FMA128D(x3, y3, z3))
}

func NEONDAbs(x Block64) Block64 {
	q0, q1, q2, q3 := neonQuartersD(x)
	return neonBlockD(Abs128D(q0), Abs128D(q1), Abs128D(q2), Abs128D(q3))
}

// NEONDHSum sums all 8 lanes in ScalarDHSum's order: the half=4 level is
// q0+q2 and q1+q3 (lanes i + i+4), half=2 adds those two, and HSum128D is the
// final pair.
func NEONDHSum(x Block64) float64 {
	q0, q1, q2, q3 := neonQuartersD(x)
	return HSum128D(Add128D(Add128D(q0, q2), Add128D(q1, q3)))
}
