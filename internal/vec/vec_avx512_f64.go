// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// AVX-512 float64 backend (512-bit, 8 float64 lanes), #191. The float32
// surface in vec_avx512.go is the template: same two layers, same contracts,
// the element in every name (Load512D, Block64) because the width suffix alone
// would collide with the float32 ops (docs/f64-design.md §1). Every archsimd
// identifier below was copied from `GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd
// go doc simd/archsimd` on go1.27.3 and the sources under
// $(go env GOROOT)/src/simd/archsimd/ — the cross-arch flags are not optional on
// the arm64 dev host, where a bare go doc reports these types as nonexistent
// (docs/toolchain-notes.md T29).

// F64x8 is the 512-bit float64 vector, aliased for the same reason F32x16 is:
// kernels outside this package can name it without importing simd/archsimd.
type F64x8 = archsimd.Float64x8

// ---------------------------------------------------------------- hot layer

// Load512D loads 8 float64 from s, which must have at least 8 elements; it
// panics otherwise. Same pre-sliced-panel contract as Load512.
func Load512D(s []float64) F64x8 { return archsimd.LoadFloat64x8(s) }

// LoadPart512D loads min(len(s), 8) elements and zero-fills the rest; the lane
// count archsimd returns is discarded, as in LoadPart512 (T23).
func LoadPart512D(s []float64) F64x8 { v, _ := archsimd.LoadFloat64x8Part(s); return v }

// Store512D stores all 8 lanes into s, which must have at least 8 elements.
func Store512D(s []float64, x F64x8) { x.Store(s) }

// StorePart512D stores as many lanes as fit in s.
func StorePart512D(s []float64, x F64x8) { _ = x.StorePart(s) }

// Broadcast512D returns v in all 8 lanes.
func Broadcast512D(v float64) F64x8 { return archsimd.BroadcastFloat64x8(v) }

// Zero512D returns the all-zero vector.
func Zero512D() F64x8 { var z F64x8; return z }

// Add512D returns x+y lanewise.
func Add512D(x, y F64x8) F64x8 { return x.Add(y) }

// Sub512D returns x-y lanewise.
func Sub512D(x, y F64x8) F64x8 { return x.Sub(y) }

// Mul512D returns x*y lanewise.
func Mul512D(x, y F64x8) F64x8 { return x.Mul(y) }

// Div512D returns x/y lanewise. Asm: VDIVPD (AVX512), IEEE on a zero divisor.
func Div512D(x, y F64x8) F64x8 { return x.Div(y) }

// FMA512D returns x*y+z lanewise as one fused operation. go doc names the
// lowering VFMADD213PD — the float64 twin of FMA512's VFMADD213PS, so the
// operand-position reasoning in peak.go (accumulator as the multiplicand)
// carries over unchanged.
func FMA512D(x, y, z F64x8) F64x8 { return x.MulAdd(y, z) }

// Max512D returns the lanewise maximum (VMAXPD semantics; see ScalarDMax).
//
// The NaN/±0 operand order is NOT pinned by the source order of x.Max(y).
// The SSA ops are marked commutative (golang/go#81468, open), and measured on
// janus 2026-10-26: swapping this body to y.Max(x) changed this function's
// register form but not the inlined AVX512DMax, where load folding put the
// memory operand y second either way — so TestDiff64Binary stayed green. The
// float32 Max512/Min512 (T6) share the mechanism. No kernel calls Max/Min yet;
// the first that does owes a register-form check.
func Max512D(x, y F64x8) F64x8 { return x.Max(y) }

// Min512D returns the lanewise minimum (VMINPD semantics; see ScalarDMin).
func Min512D(x, y F64x8) F64x8 { return x.Min(y) }

// keepMaskI64 is every float64 bit except the sign, as an int64.
const keepMaskI64 int64 = ^(-1 << 63)

// I64x8 is the 512-bit int64 vector, aliased so a caller hoisting the abs mask
// can name its type (the I32x16 reasoning).
type I64x8 = archsimd.Int64x8

// Abs512D clears the sign bit of every lane.
//
// Float64x8 DOES have an Abs() in go1.27.3, but go doc marks it "Emulated", and
// its body in ops_emulated_amd64.go is `x.ToBits().AndNot(mask)` — the exact
// shape Abs512 refuses for float32 (T27, #117: rewriteTern transposes AndNot's
// operands inside a fused logical tree, and Abs()'s correctness today rests on
// the unsigned-vector skip of golang/go#79666). So the float64 path takes the
// float32 path's AND-with-complement form for the same reason: its fused
// immediate is symmetric in its inputs and has no wrong answer to transpose to.
func Abs512D(x F64x8) F64x8 { return AbsWith512D(x, AbsMask512D()) }

// AbsMask512D returns the mask AbsWith512D wants: every float64 bit except the
// sign, in every lane. Loop-invariant; hoisting it is the caller's job (#54).
func AbsMask512D() I64x8 { return archsimd.BroadcastInt64x8(keepMaskI64) }

// AbsWith512D is Abs512D with a caller-supplied mask; the one place the float64
// sign-bit trick is written. See AbsWith512 for why it is And and not AndNot.
func AbsWith512D(x F64x8, mask I64x8) F64x8 {
	return x.AsInt64x8().And(mask).AsFloat64x8()
}

// HSum512D sums all 8 lanes in the halving order ScalarDHSum specifies (lanes
// i+4, then i+2, then the final pair), so the two agree bit-for-bit. Unlike
// HSum512, every fold but the last stays in vector registers: Float64x4 has the
// GetLo/GetHi that Float32x4 lacks.
func HSum512D(x F64x8) float64 {
	h4 := x.GetLo().Add(x.GetHi())   // lanes i + i+4
	h2 := h4.GetLo().Add(h4.GetHi()) // lanes i + i+2
	var a [2]float64
	h2.StoreArray(&a)
	return a[0] + a[1] // final pair
}

// --------------------------------------------------------------- test layer

func AVX512DLoad(s []float64) Block64         { return block64Of512(Load512D(s)) }
func AVX512DLoadPart(s []float64) Block64     { return block64Of512(LoadPart512D(s)) }
func AVX512DStore(s []float64, x Block64)     { Store512D(s, load512D(x)) }
func AVX512DStorePart(s []float64, x Block64) { StorePart512D(s, load512D(x)) }
func AVX512DBroadcast(v float64) Block64      { return block64Of512(Broadcast512D(v)) }
func AVX512DZero() Block64                    { return block64Of512(Zero512D()) }
func AVX512DAdd(x, y Block64) Block64         { return block64Of512(Add512D(load512D(x), load512D(y))) }
func AVX512DSub(x, y Block64) Block64         { return block64Of512(Sub512D(load512D(x), load512D(y))) }
func AVX512DMul(x, y Block64) Block64         { return block64Of512(Mul512D(load512D(x), load512D(y))) }
func AVX512DDiv(x, y Block64) Block64         { return block64Of512(Div512D(load512D(x), load512D(y))) }
func AVX512DMax(x, y Block64) Block64         { return block64Of512(Max512D(load512D(x), load512D(y))) }
func AVX512DMin(x, y Block64) Block64         { return block64Of512(Min512D(load512D(x), load512D(y))) }
func AVX512DAbs(x Block64) Block64            { return block64Of512(Abs512D(load512D(x))) }
func AVX512DHSum(x Block64) float64           { return HSum512D(load512D(x)) }

func AVX512DMulAdd(x, y, z Block64) Block64 {
	return block64Of512(FMA512D(load512D(x), load512D(y), load512D(z)))
}

func load512D(b Block64) F64x8 { return archsimd.LoadFloat64x8Array((*[Lanes64]float64)(&b)) }

func block64Of512(v F64x8) Block64 {
	var b Block64
	v.StoreArray((*[Lanes64]float64)(&b))
	return b
}
