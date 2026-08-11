// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// AVX2 backend (256-bit, 8 float32 lanes), written against the *read*
// archsimd API of go1.26.5 — identifiers copied from `go doc`, never
// recalled (DESIGN.md §4/P0).
//
// The Block test layer here is 16 lanes wide like every other backend, done
// as two 8-lane halves. That is what makes a Block op mean the same thing on
// every backend, so the differential test needs no per-backend width
// bookkeeping and the scalar spec stays the single reference.

// F32x8 is the 256-bit float32 vector; see F32x16 on why this alias exists.
type F32x8 = archsimd.Float32x8

// ---------------------------------------------------------------- hot layer

// Load256 loads 8 float32 from s, which must have at least 8 elements.
func Load256(s []float32) F32x8 { return archsimd.LoadFloat32x8Slice(s) }

// LoadPart256 loads min(len(s), 8) elements and zero-fills the rest.
func LoadPart256(s []float32) F32x8 { return archsimd.LoadFloat32x8SlicePart(s) }

// Store256 stores all 8 lanes into s, which must have at least 8 elements.
func Store256(s []float32, x F32x8) { x.StoreSlice(s) }

// StorePart256 stores as many lanes as fit in s.
func StorePart256(s []float32, x F32x8) { x.StoreSlicePart(s) }

// Broadcast256 returns v in all 8 lanes.
func Broadcast256(v float32) F32x8 { return archsimd.BroadcastFloat32x8(v) }

// Zero256 returns the all-zero vector.
func Zero256() F32x8 { var z F32x8; return z }

// Add256 returns x+y lanewise.
func Add256(x, y F32x8) F32x8 { return x.Add(y) }

// Sub256 returns x-y lanewise.
func Sub256(x, y F32x8) F32x8 { return x.Sub(y) }

// Mul256 returns x*y lanewise.
func Mul256(x, y F32x8) F32x8 { return x.Mul(y) }

// FMA256 returns x*y+z lanewise as one fused operation. Requires the FMA3
// CPU feature, which is why HasAVX2 checks FMA() as well as AVX2():
// archsimd documents Float32x8.MulAdd as "CPU Feature: FMA", separate from
// the AVX2 bundle. gate-p0.sh disassembles this alongside FMA512.
func FMA256(x, y, z F32x8) F32x8 { return x.MulAdd(y, z) }

// Max256 returns the lanewise maximum (VMAXPS semantics; see ScalarMax).
func Max256(x, y F32x8) F32x8 { return x.Max(y) }

// Min256 returns the lanewise minimum (VMINPS semantics; see ScalarMin).
func Min256(x, y F32x8) F32x8 { return x.Min(y) }

// Abs256 clears the sign bit of every lane; see Abs512 on why this detours
// through the integer vector type.
func Abs256(x F32x8) F32x8 {
	return x.AsInt32x8().AndNot(archsimd.BroadcastInt32x8(signMaskI32)).AsFloat32x8()
}

// HSum256 sums all 8 lanes, in the fold order ScalarHSum specifies.
func HSum256(x F32x8) float32 {
	h4 := x.GetLo().Add(x.GetHi()) // lanes i + i+4
	var a [4]float32
	h4.Store(&a)
	a[0] += a[2] // lanes i + i+2
	a[1] += a[3]
	return a[0] + a[1] // final pair
}

// --------------------------------------------------------------- test layer
//
// Sixteen lanes as two 8-lane halves, matching the scalar spec's signatures.

func AVX2Load(s []float32) Block {
	return blockOf256(Load256(s[:8]), Load256(s[8:16]))
}

// AVX2LoadPart mirrors ScalarLoadPart: min(len(s), 16) elements, rest zero.
// The halves are taken with the same partial semantics, so a length of, say,
// 5 zero-fills the rest of the low half and all of the high half.
func AVX2LoadPart(s []float32) Block {
	if len(s) > Lanes {
		s = s[:Lanes]
	}
	var lo, hi []float32
	if len(s) > 8 {
		lo, hi = s[:8], s[8:]
	} else {
		lo, hi = s, nil
	}
	return blockOf256(LoadPart256(lo), LoadPart256(hi))
}

func AVX2Store(s []float32, x Block) {
	l, h := halves256(x)
	Store256(s[:8], l)
	Store256(s[8:16], h)
}

func AVX2StorePart(s []float32, x Block) {
	if len(s) > Lanes {
		s = s[:Lanes]
	}
	l, h := halves256(x)
	if len(s) > 8 {
		StorePart256(s[:8], l)
		StorePart256(s[8:], h)
		return
	}
	StorePart256(s, l)
}

func AVX2Broadcast(v float32) Block { return blockOf256(Broadcast256(v), Broadcast256(v)) }
func AVX2Zero() Block               { return blockOf256(Zero256(), Zero256()) }

func AVX2Add(x, y Block) Block { return zip256(x, y, Add256) }
func AVX2Sub(x, y Block) Block { return zip256(x, y, Sub256) }
func AVX2Mul(x, y Block) Block { return zip256(x, y, Mul256) }
func AVX2Max(x, y Block) Block { return zip256(x, y, Max256) }
func AVX2Min(x, y Block) Block { return zip256(x, y, Min256) }

func AVX2Abs(x Block) Block {
	l, h := halves256(x)
	return blockOf256(Abs256(l), Abs256(h))
}

func AVX2MulAdd(x, y, z Block) Block {
	xl, xh := halves256(x)
	yl, yh := halves256(y)
	zl, zh := halves256(z)
	return blockOf256(FMA256(xl, yl, zl), FMA256(xh, yh, zh))
}

// AVX2HSum folds the two halves together first, which is exactly the
// half=8 level of ScalarHSum's tree, then continues in the same order — so
// this is bit-exact with the scalar spec and with HSum512.
func AVX2HSum(x Block) float32 {
	l, h := halves256(x)
	return HSum256(Add256(l, h))
}

func halves256(b Block) (lo, hi F32x8) {
	return archsimd.LoadFloat32x8((*[8]float32)(b[0:8])),
		archsimd.LoadFloat32x8((*[8]float32)(b[8:16]))
}

func blockOf256(lo, hi F32x8) Block {
	var b Block
	lo.Store((*[8]float32)(b[0:8]))
	hi.Store((*[8]float32)(b[8:16]))
	return b
}

func zip256(x, y Block, op func(a, b F32x8) F32x8) Block {
	xl, xh := halves256(x)
	yl, yh := halves256(y)
	return blockOf256(op(xl, yl), op(xh, yh))
}

// HasAVX2 reports whether the AVX2 backend may run here. It requires FMA3 in
// addition to the AVX2 bundle, because the backend's whole point is FMA256.
func HasAVX2() bool { return archsimd.X86.AVX2() && archsimd.X86.FMA() }
