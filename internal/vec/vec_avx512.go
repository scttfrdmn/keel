// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// AVX-512 backend (512-bit, 16 float32 lanes), written against the *read*
// archsimd API of go1.26.5. Every identifier below was copied from
// `go doc simd/archsimd` output and the sources under
// $(go env GOROOT)/src/simd/archsimd/, per the standing order in DESIGN.md
// §4/P0 — none is recalled.
//
// Two layers here, described in the package doc: the native-typed hot layer
// the microkernels call (Load512, FMA512, ...), and the Block-typed test
// layer the differential tests drive (AVX512MulAdd, ...).

// F32x16 is the 512-bit float32 vector, aliased so kernels in internal/kern
// can name the type without importing simd/archsimd themselves. The alias is
// deliberate: it keeps the "all simd imports live in internal/vec" rule
// enforceable by grep while still letting values stay in registers across
// wrapper calls.
type F32x16 = archsimd.Float32x16

// ---------------------------------------------------------------- hot layer
//
// One archsimd operation each and nothing else, so they inline to a single
// instruction. gate-p0.sh disassembles FMA512 and requires exactly that.

// Load512 loads 16 float32 from s, which must have at least 16 elements; it
// panics otherwise (the underlying slice-to-array conversion does). That is
// the intended contract: kernels pre-slice packed panels to exact length
// outside the K-loop so the bounds check is eliminated (DESIGN.md §4/P2).
func Load512(s []float32) F32x16 { return archsimd.LoadFloat32x16Slice(s) }

// LoadPart512 loads min(len(s), 16) elements and zero-fills the rest — the
// edge-kernel path for M%MR and N%NR remainders.
func LoadPart512(s []float32) F32x16 { return archsimd.LoadFloat32x16SlicePart(s) }

// Store512 stores all 16 lanes into s, which must have at least 16 elements.
func Store512(s []float32, x F32x16) { x.StoreSlice(s) }

// StorePart512 stores as many lanes as fit in s.
func StorePart512(s []float32, x F32x16) { x.StoreSlicePart(s) }

// Broadcast512 returns v in all 16 lanes.
func Broadcast512(v float32) F32x16 { return archsimd.BroadcastFloat32x16(v) }

// Zero512 returns the all-zero vector.
func Zero512() F32x16 { var z F32x16; return z }

// Add512 returns x+y lanewise.
func Add512(x, y F32x16) F32x16 { return x.Add(y) }

// Sub512 returns x-y lanewise.
func Sub512(x, y F32x16) F32x16 { return x.Sub(y) }

// Mul512 returns x*y lanewise.
func Mul512(x, y F32x16) F32x16 { return x.Mul(y) }

// FMA512 returns x*y+z lanewise as one fused operation.
//
// This is the most load-bearing wrapper in keel: the whole performance thesis
// is that the K-loop is a chain of these and nothing else. gate-p0.sh
// disassembles it and requires exactly one instruction from the
// VFMADD{132,213,231}PS family with zero separate VMULPS/VADDPS. On go1.26.5
// it lowers to VFMADD213PS — DESIGN.md predicted the 231 form, and the two
// are the same instruction under a different operand-order encoding. See
// docs/toolchain-notes.md.
func FMA512(x, y, z F32x16) F32x16 { return x.MulAdd(y, z) }

// Max512 returns the lanewise maximum (VMAXPS semantics; see ScalarMax).
func Max512(x, y F32x16) F32x16 { return x.Max(y) }

// Min512 returns the lanewise minimum (VMINPS semantics; see ScalarMin).
func Min512(x, y F32x16) F32x16 { return x.Min(y) }

// signMaskI32 is the float32 sign bit as an int32, for building the abs mask.
// Written as -1<<31 because the constant int32(1<<31) overflows.
const signMaskI32 int32 = -1 << 31

// Abs512 clears the sign bit of every lane.
//
// The archsimd API of go1.26.5 has no float32 Abs and no float32 bitwise ops
// at all, so this bitcasts to the integer vector type, clears the sign bit
// there, and bitcasts back. AndNot is documented as Go's `x &^ y` (lowering
// to VPANDND), which mirrors ScalarAbs's `bits &^ signMask32` exactly rather
// than merely approximating it. The As* conversions are reinterpretations
// rather than data movement, so this is at most two instructions of real
// work. Recorded in docs/toolchain-notes.md.
func Abs512(x F32x16) F32x16 { return AbsWith512(x, AbsMask512()) }

// I32x16 is the 512-bit int32 vector, aliased for the same reason F32x16 is:
// a caller that hoists the abs mask out of its own loop has to be able to name
// the mask's type, and only this package may import simd/archsimd.
type I32x16 = archsimd.Int32x16

// AbsMask512 returns the mask AbsWith512 wants: the float32 sign bit in every
// lane. It is loop-invariant, and hoisting it is the *caller's* job because the
// compiler will not do it — SIMD ops are not lifted by LICM (#54, T18,
// golang/go#79984). When CL 803220 lands this and AbsWith512 both retire and
// Abs512 goes back to being the only spelling; #54 tracks that.
func AbsMask512() I32x16 { return archsimd.BroadcastInt32x16(signMaskI32) }

// AbsWith512 is Abs512 with the mask supplied by the caller rather than built
// per call. Same two instructions of real work, minus the broadcast.
//
// This is the only place the sign-bit trick is written; Abs512 delegates here,
// so the two spellings cannot drift and the differential test against ScalarAbs
// covers both.
func AbsWith512(x F32x16, mask I32x16) F32x16 {
	return x.AsInt32x16().AndNot(mask).AsFloat32x16()
}

// HSum512 sums all 16 lanes.
//
// The fold order — upper half onto lower half, repeatedly — is the same tree
// ScalarHSum specifies, so the two agree bit-for-bit and the differential
// test demands exact equality rather than a tolerance. The last two levels
// happen in scalar registers because Float32x4 has no GetLo/GetHi to fold
// with; that is off the hot path (one HSum per dot product, never inside a
// K-loop).
func HSum512(x F32x16) float32 {
	h8 := x.GetLo().Add(x.GetHi())   // lanes i + i+8
	h4 := h8.GetLo().Add(h8.GetHi()) // lanes i + i+4
	var a [4]float32
	h4.Store(&a)
	a[0] += a[2] // lanes i + i+2
	a[1] += a[3]
	return a[0] + a[1] // final pair
}

// --------------------------------------------------------------- test layer
//
// Block-in, Block-out versions with the same signatures as the scalar spec,
// so one differential test table drives every backend. These round-trip
// through memory; they are not for hot loops.

func AVX512Load(s []float32) Block         { return blockOf512(Load512(s)) }
func AVX512LoadPart(s []float32) Block     { return blockOf512(LoadPart512(s)) }
func AVX512Store(s []float32, x Block)     { Store512(s, load512(x)) }
func AVX512StorePart(s []float32, x Block) { StorePart512(s, load512(x)) }
func AVX512Broadcast(v float32) Block      { return blockOf512(Broadcast512(v)) }
func AVX512Zero() Block                    { return blockOf512(Zero512()) }
func AVX512Add(x, y Block) Block           { return blockOf512(Add512(load512(x), load512(y))) }
func AVX512Sub(x, y Block) Block           { return blockOf512(Sub512(load512(x), load512(y))) }
func AVX512Mul(x, y Block) Block           { return blockOf512(Mul512(load512(x), load512(y))) }
func AVX512Max(x, y Block) Block           { return blockOf512(Max512(load512(x), load512(y))) }
func AVX512Min(x, y Block) Block           { return blockOf512(Min512(load512(x), load512(y))) }
func AVX512Abs(x Block) Block              { return blockOf512(Abs512(load512(x))) }
func AVX512HSum(x Block) float32           { return HSum512(load512(x)) }

func AVX512MulAdd(x, y, z Block) Block {
	return blockOf512(FMA512(load512(x), load512(y), load512(z)))
}

func load512(b Block) F32x16 { return archsimd.LoadFloat32x16((*[Lanes]float32)(&b)) }

func blockOf512(v F32x16) Block {
	var b Block
	v.Store((*[Lanes]float32)(&b))
	return b
}

// HasAVX512 reports whether this CPU supports the AVX512F+CD+BW+DQ+VL bundle
// that archsimd requires before any AVX-512 op may be used.
func HasAVX512() bool { return archsimd.X86.AVX512() }

// The two predicates below are the only ones here that no op in this package
// uses. They exist because archsimd reports CPU *features* and never a
// microarchitecture (docs/toolchain-notes.md T14, issue #25), and keel's kernel
// *shape* choice is a per-µarch decision, so the shape registry has to
// fingerprint a generation from the features it can see.
//
// They live here because this is the only package allowed to import
// simd/archsimd (CLAUDE.md), and they are deliberately raw: this file reports
// bits, and internal/kern decides what a bit means. A predicate here called
// something like IsSkylakeServer would be putting a guess about silicon into the
// shim, where nothing can test it.
//
// Both identifiers were copied from `go doc simd/archsimd.X86Features` on
// go1.26.5, per DESIGN.md §4/P0.

// HasAVX512VBMI2 reports whether this CPU has AVX512_VBMI2, one half of the
// feature bundle that arrived with Ice Lake and Zen 4.
func HasAVX512VBMI2() bool { return archsimd.X86.AVX512VBMI2() }

// HasAVX512VPOPCNTDQ reports whether this CPU has AVX512_VPOPCNTDQ, the other
// half.
func HasAVX512VPOPCNTDQ() bool { return archsimd.X86.AVX512VPOPCNTDQ() }
