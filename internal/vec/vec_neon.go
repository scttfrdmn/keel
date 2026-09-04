// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

// NEON backend: the arm64 shim, 128-bit Float32x4 (four lanes). Structured to
// mirror vec_avx512.go exactly — a hot layer of native-typed one-instruction
// wrappers the microkernel calls, and a Block-typed test layer that the one
// differential table in vec_diff_test.go binds to the scalar spec.
//
// Every name here was copied from $(go env GOROOT)/src/simd/archsimd/*_arm64.go
// under GOEXPERIMENT=simd, per the P0 standing order — not recalled. Three arm64
// facts that differ from the amd64 shim and are load-bearing:
//
//   - MulAdd emits a fused VFMLA that accumulates IN PLACE (destination is the
//     addend), so there is no per-FMA scratch register — unlike amd64's 213 form
//     and its T10 constraint. This is why the port re-derives its tile family
//     from a fresh register budget rather than scaling amd64's (docs/neon-sweep.md,
//     #136). Put in one line: VFMLA is natively the 231 accumulate CL 1 (#127)
//     teaches amd64's compiler to emit.
//   - Abs is a native Float32x4.Abs() (FABS), not the AsInt32x16().And(mask) trick
//     the amd64 shim uses: arm64 exposes no As* bitcast on these types, and FABS
//     clears the sign bit with the same result the mask gives, which the
//     differential test verifies bit-exactly rather than assumes.
//   - BroadcastFloat32x4 lowers to a separate VDUP (no by-element FMLA); recorded
//     as a candidate on #130, not a defect this file works around.
package vec

import "simd/archsimd"

// F32x4 is the NEON float32 vector, 128 bits / four lanes.
type F32x4 = archsimd.Float32x4

// --------------------------------------------------------------- hot layer
// Native-typed wrappers; each inlines to one instruction and keeps values in
// registers across a chain. This is the layer a NEON microkernel is built from.

// Load128 loads four float32 from s; requires len(s) >= 4 (see ScalarLoad on why
// this panics rather than truncates). Use LoadPart128 for the partial tail.
func Load128(s []float32) F32x4 { return archsimd.LoadFloat32x4(s) }

// LoadPart128 loads min(len(s), 4) elements and zero-fills the rest.
func LoadPart128(s []float32) F32x4 { v, _ := archsimd.LoadFloat32x4Part(s); return v }

// Store128 stores all four lanes into s; requires len(s) >= 4.
func Store128(s []float32, x F32x4) { x.Store(s) }

// StorePart128 stores as many lanes as fit in s.
func StorePart128(s []float32, x F32x4) { _ = x.StorePart(s) }

// Broadcast128 returns a vector with v in every lane (VDUP).
func Broadcast128(v float32) F32x4 { return archsimd.BroadcastFloat32x4(v) }

// Zero128 returns the all-zero vector (+0 in every lane).
func Zero128() F32x4 { var z F32x4; return z }

// Add128 returns x+y lanewise.
func Add128(x, y F32x4) F32x4 { return x.Add(y) }

// Sub128 returns x-y lanewise.
func Sub128(x, y F32x4) F32x4 { return x.Sub(y) }

// Mul128 returns x*y lanewise.
func Mul128(x, y F32x4) F32x4 { return x.Mul(y) }

// FMA128 returns x*y+z lanewise with a single rounding — VFMLA, true fused
// multiply-add, matching ScalarMulAdd's math.FMA semantics.
func FMA128(x, y, z F32x4) F32x4 { return x.MulAdd(y, z) }

// Max128 / Min128 return the lanewise max / min with the SPEC's semantics —
// `x>y ? x : y` and `x<y ? x : y` — built from a compare and a select, NOT from
// NEON's native FMAX/FMIN.
//
// This is a measured shim decision, not a preference. NEON FMAX/FMIN disagree with
// the scalar spec (which matches x86 VMAXPS/VMINPS, keel's first ISA) on exactly the
// two edge classes the differential test drives: FMAX propagates NaN where the spec
// returns the second operand, and FMAX(+0,-0)=+0 where the spec returns the second
// operand's -0. `Greater` is FCMGT — unordered (NaN) and +0-vs-0 both compare false,
// so the mask picks y in exactly the cases the spec does. IfElse(mask, y) keeps x
// where the mask is true and takes y where it is false, so this is `x>y ? x : y`
// to the bit. The cost is a compare+select instead of one FMAX; Max/Min are not
// GEMM-kernel ops (#136 uses FMA/Load/Broadcast/Store/Add), so this is off every
// hot path the sweep measures, and an L1 arm64 routine that can tolerate IEEE
// max/min semantics may revisit it against a spec discussion, not silently.
func Max128(x, y F32x4) F32x4 { return x.IfElse(x.Greater(y), y) }
func Min128(x, y F32x4) F32x4 { return x.IfElse(x.Less(y), y) }

// Abs128 clears the sign bit of every lane (FABS).
func Abs128(x F32x4) F32x4 { return x.Abs() }

// HSum128 sums the four lanes, folding in the same order as HSum512's tail
// (a[0]+a[2], a[1]+a[3], then the pair) so it agrees bit-for-bit with the scalar
// spec's tree. Off the hot path (one per dot product / one per peak reduction).
func HSum128(x F32x4) float32 {
	var a [4]float32
	x.StoreArray(&a)
	a[0] += a[2]
	a[1] += a[3]
	return a[0] + a[1]
}

// --------------------------------------------------------------- test layer
// Block-in, Block-out, same signatures as the scalar spec, so the one
// differential table drives every backend. A 16-lane Block is four Float32x4
// quarters. These round-trip through memory; not for hot loops.

func neonQuarters(b Block) (q0, q1, q2, q3 F32x4) {
	return Load128(b[0:4]), Load128(b[4:8]), Load128(b[8:12]), Load128(b[12:16])
}

func neonBlock(q0, q1, q2, q3 F32x4) Block {
	var b Block
	Store128(b[0:4], q0)
	Store128(b[4:8], q1)
	Store128(b[8:12], q2)
	Store128(b[12:16], q3)
	return b
}

func NEONLoad(s []float32) Block {
	return neonBlock(Load128(s[0:4]), Load128(s[4:8]), Load128(s[8:12]), Load128(s[12:16]))
}

// NEONLoadPart loads min(len(s), Lanes) elements, zero-filling the rest, matching
// ScalarLoadPart. Each quarter is a native partial load, or a zero vector when the
// quarter lies entirely past the valid length.
func NEONLoadPart(s []float32) Block {
	n := len(s)
	if n > Lanes {
		n = Lanes
	}
	var q [4]F32x4
	for i := range q {
		lo := i * 4
		if lo >= n {
			q[i] = Zero128()
			continue
		}
		hi := lo + 4
		if hi > n {
			hi = n
		}
		q[i] = LoadPart128(s[lo:hi])
	}
	return neonBlock(q[0], q[1], q[2], q[3])
}

func NEONStore(s []float32, x Block) {
	q0, q1, q2, q3 := neonQuarters(x)
	Store128(s[0:4], q0)
	Store128(s[4:8], q1)
	Store128(s[8:12], q2)
	Store128(s[12:16], q3)
}

// NEONStorePart stores min(len(s), Lanes) lanes, matching ScalarStorePart.
func NEONStorePart(s []float32, x Block) {
	n := len(s)
	if n > Lanes {
		n = Lanes
	}
	q := [4]F32x4{}
	q[0], q[1], q[2], q[3] = neonQuarters(x)
	for i := range q {
		lo := i * 4
		if lo >= n {
			break
		}
		hi := lo + 4
		if hi > n {
			hi = n
		}
		StorePart128(s[lo:hi], q[i])
	}
}

func NEONBroadcast(v float32) Block {
	q := Broadcast128(v)
	return neonBlock(q, q, q, q)
}

func NEONZero() Block { return neonBlock(Zero128(), Zero128(), Zero128(), Zero128()) }

func neonBinop(x, y Block, op func(a, b F32x4) F32x4) Block {
	x0, x1, x2, x3 := neonQuarters(x)
	y0, y1, y2, y3 := neonQuarters(y)
	return neonBlock(op(x0, y0), op(x1, y1), op(x2, y2), op(x3, y3))
}

func NEONAdd(x, y Block) Block { return neonBinop(x, y, Add128) }
func NEONSub(x, y Block) Block { return neonBinop(x, y, Sub128) }
func NEONMul(x, y Block) Block { return neonBinop(x, y, Mul128) }
func NEONMax(x, y Block) Block { return neonBinop(x, y, Max128) }
func NEONMin(x, y Block) Block { return neonBinop(x, y, Min128) }

func NEONMulAdd(x, y, z Block) Block {
	x0, x1, x2, x3 := neonQuarters(x)
	y0, y1, y2, y3 := neonQuarters(y)
	z0, z1, z2, z3 := neonQuarters(z)
	return neonBlock(FMA128(x0, y0, z0), FMA128(x1, y1, z1), FMA128(x2, y2, z2), FMA128(x3, y3, z3))
}

func NEONAbs(x Block) Block {
	q0, q1, q2, q3 := neonQuarters(x)
	return neonBlock(Abs128(q0), Abs128(q1), Abs128(q2), Abs128(q3))
}

// NEONHSum sums all 16 lanes in the exact binary-tree order ScalarHSum specifies
// (i += i+half for half = 8,4,2,1), so the two agree bit-for-bit and the
// differential test demands equality, not a tolerance. The half=8 and half=4
// levels are vector adds over disjoint lane ranges — q0+=q2, q1+=q3, then the two
// results add — which is exactly HSum512's fold; the last two levels drop to
// scalar registers, as HSum512 does, because there is nothing off the hot path to
// gain by keeping them vector (one HSum per dot product, never in a K-loop).
func NEONHSum(x Block) float32 {
	q0, q1, q2, q3 := neonQuarters(x)
	s0 := Add128(q0, q2) // lanes i + i+8, i in 0..3
	s1 := Add128(q1, q3) // lanes i + i+8, i in 4..7
	h := Add128(s0, s1)  // + lanes i+4
	var a [4]float32
	h.StoreArray(&a)
	a[0] += a[2] // half=2
	a[1] += a[3]
	return a[0] + a[1] // half=1
}

// HasNEON reports whether this binary can run NEON. On arm64 it is always true:
// NEON (ASIMD) is mandatory in the ARMv8-A baseline. Referenced only from arm64
// code, so no stub is needed on other arches.
func HasNEON() bool { return true }
