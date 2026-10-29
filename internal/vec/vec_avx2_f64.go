// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// AVX2 float64 backend (256-bit, 4 float64 lanes), #192: the rung that gives
// float64 Level 1 the same ladder float32's has (docs/f64-design.md, ruled
// 2026-10-29). vec_avx2.go is the template and vec_avx512_f64.go the naming
// (the element in every name). Every archsimd identifier below was copied from
// `GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go doc simd/archsimd` on go1.27.3.
// AsInt64x4/AsFloat64x4 are marked deprecated there, in favour of ToBits and
// BitsTo; they are kept to match the As* forms every other shim here uses, so
// the AbsWith twins stay textually parallel.

// F64x4 is the 256-bit float64 vector; see F32x16 on why this alias exists.
type F64x4 = archsimd.Float64x4

// ---------------------------------------------------------------- hot layer

// Load256D loads 4 float64 from s, which must have at least 4 elements.
func Load256D(s []float64) F64x4 { return archsimd.LoadFloat64x4(s) }

// LoadPart256D loads min(len(s), 4) elements and zero-fills the rest; the lane
// count is discarded as in LoadPart512.
func LoadPart256D(s []float64) F64x4 { v, _ := archsimd.LoadFloat64x4Part(s); return v }

// Store256D stores all 4 lanes into s, which must have at least 4 elements.
func Store256D(s []float64, x F64x4) { x.Store(s) }

// StorePart256D stores as many lanes as fit in s.
func StorePart256D(s []float64, x F64x4) { _ = x.StorePart(s) }

// Broadcast256D returns v in all 4 lanes.
func Broadcast256D(v float64) F64x4 { return archsimd.BroadcastFloat64x4(v) }

// Zero256D returns the all-zero vector.
func Zero256D() F64x4 { var z F64x4; return z }

// Add256D returns x+y lanewise.
func Add256D(x, y F64x4) F64x4 { return x.Add(y) }

// Sub256D returns x-y lanewise.
func Sub256D(x, y F64x4) F64x4 { return x.Sub(y) }

// Mul256D returns x*y lanewise.
func Mul256D(x, y F64x4) F64x4 { return x.Mul(y) }

// Div256D returns x/y lanewise. Asm: VDIVPD (AVX).
func Div256D(x, y F64x4) F64x4 { return x.Div(y) }

// FMA256D returns x*y+z lanewise as one fused operation: VFMADD213PD, CPU
// feature FMA, which HasAVX2 already requires.
func FMA256D(x, y, z F64x4) F64x4 { return x.MulAdd(y, z) }

// Max256D returns the lanewise maximum (VMAXPD semantics; see ScalarDMax and
// Max512D on the operand order).
func Max256D(x, y F64x4) F64x4 { return x.Max(y) }

// Min256D returns the lanewise minimum (VMINPD semantics; see ScalarDMin).
func Min256D(x, y F64x4) F64x4 { return x.Min(y) }

// I64x4 is the 256-bit int64 vector, aliased so a caller hoisting the abs mask
// can name its type.
type I64x4 = archsimd.Int64x4

// Abs256D clears the sign bit of every lane; see Abs512D on why it is not
// Float64x4.Abs().
func Abs256D(x F64x4) F64x4 { return AbsWith256D(x, AbsMask256D()) }

// AbsMask256D returns every float64 bit except the sign, in every lane.
// Loop-invariant; hoisting it is the caller's job (#54).
func AbsMask256D() I64x4 { return archsimd.BroadcastInt64x4(keepMaskI64) }

// AbsWith256D is Abs256D with a caller-supplied mask, in the And form of
// AbsWith256 and AbsWith512D (T27, #117).
func AbsWith256D(x F64x4, mask I64x4) F64x4 {
	return x.AsInt64x4().And(mask).AsFloat64x4()
}

// HSum256D sums all 4 lanes in ScalarDHSum's halving order from its 4-lane
// level on: lanes i + i+2, then the final pair.
func HSum256D(x F64x4) float64 {
	h2 := x.GetLo().Add(x.GetHi()) // lanes i + i+2
	var a [2]float64
	h2.StoreArray(&a)
	return a[0] + a[1] // final pair
}

// --------------------------------------------------------------- test layer
//
// Eight lanes as two 4-lane halves, matching the float64 scalar spec.

const half256D = Lanes64 / 2

func AVX2DLoad(s []float64) Block64 {
	return block64Of256(Load256D(s[:half256D]), Load256D(s[half256D:Lanes64]))
}

// AVX2DLoadPart mirrors ScalarDLoadPart: min(len(s), 8) elements, rest zero.
func AVX2DLoadPart(s []float64) Block64 {
	if len(s) > Lanes64 {
		s = s[:Lanes64]
	}
	var lo, hi []float64
	if len(s) > half256D {
		lo, hi = s[:half256D], s[half256D:]
	} else {
		lo, hi = s, nil
	}
	return block64Of256(LoadPart256D(lo), LoadPart256D(hi))
}

func AVX2DStore(s []float64, x Block64) {
	l, h := halves256D(x)
	Store256D(s[:half256D], l)
	Store256D(s[half256D:Lanes64], h)
}

func AVX2DStorePart(s []float64, x Block64) {
	if len(s) > Lanes64 {
		s = s[:Lanes64]
	}
	l, h := halves256D(x)
	if len(s) > half256D {
		StorePart256D(s[:half256D], l)
		StorePart256D(s[half256D:], h)
		return
	}
	StorePart256D(s, l)
}

func AVX2DBroadcast(v float64) Block64 { return block64Of256(Broadcast256D(v), Broadcast256D(v)) }
func AVX2DZero() Block64               { return block64Of256(Zero256D(), Zero256D()) }

func AVX2DAdd(x, y Block64) Block64 { return zip256D(x, y, Add256D) }
func AVX2DSub(x, y Block64) Block64 { return zip256D(x, y, Sub256D) }
func AVX2DMul(x, y Block64) Block64 { return zip256D(x, y, Mul256D) }
func AVX2DDiv(x, y Block64) Block64 { return zip256D(x, y, Div256D) }
func AVX2DMax(x, y Block64) Block64 { return zip256D(x, y, Max256D) }
func AVX2DMin(x, y Block64) Block64 { return zip256D(x, y, Min256D) }

func AVX2DAbs(x Block64) Block64 {
	l, h := halves256D(x)
	return block64Of256(Abs256D(l), Abs256D(h))
}

func AVX2DMulAdd(x, y, z Block64) Block64 {
	xl, xh := halves256D(x)
	yl, yh := halves256D(y)
	zl, zh := halves256D(z)
	return block64Of256(FMA256D(xl, yl, zl), FMA256D(xh, yh, zh))
}

// AVX2DHSum folds the halves first, which is the half=4 level of ScalarDHSum's
// tree, then continues in the same order, so it is bit-exact with the spec.
func AVX2DHSum(x Block64) float64 {
	l, h := halves256D(x)
	return HSum256D(Add256D(l, h))
}

func halves256D(b Block64) (lo, hi F64x4) {
	return archsimd.LoadFloat64x4Array((*[half256D]float64)(b[0:half256D])),
		archsimd.LoadFloat64x4Array((*[half256D]float64)(b[half256D:Lanes64]))
}

func block64Of256(lo, hi F64x4) Block64 {
	var b Block64
	lo.StoreArray((*[half256D]float64)(b[0:half256D]))
	hi.StoreArray((*[half256D]float64)(b[half256D:Lanes64]))
	return b
}

func zip256D(x, y Block64, op func(a, b F64x4) F64x4) Block64 {
	xl, xh := halves256D(x)
	yl, yh := halves256D(y)
	return block64Of256(op(xl, yl), op(xh, yh))
}
