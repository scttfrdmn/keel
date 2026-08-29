// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

import "math"

// Scalar backend: the executable spec. Every op added to the AVX backends
// must appear here first, and the differential tests hold them equal.
//
// This file has no build tag on purpose. It compiles on a stock toolchain
// with no GOEXPERIMENT, and on every GOARCH — that is what makes it the
// reference the other backends are measured against, and what lets keel
// exist on normal `go get` terms (DESIGN.md §4/P5).
//
// # Numerics of the spec
//
// The spec is not "whatever Go happens to do"; it is a deliberate choice per
// op, matching what the AVX backends produce, so the differential tests can
// demand *bit-exact* agreement rather than hiding disagreement under a
// tolerance. Where a choice was made, it is stated on the op.

// ScalarZero returns the all-zero block (+0 in every lane, not -0).
func ScalarZero() Block { return Block{} }

// ScalarBroadcast returns a block with v in every lane.
func ScalarBroadcast(v float32) Block {
	var r Block
	for i := range r {
		r[i] = v
	}
	return r
}

// ScalarLoad loads Lanes elements from s.
//
// Like the archsimd LoadFloat32x16 it mirrors, it requires
// len(s) >= Lanes and panics otherwise: the kernels pre-slice their packed
// panels to exact length outside the loop, so a short slice here is a bug in
// the caller rather than a case to silently zero-fill. Use ScalarLoadPart
// for the deliberately-partial case.
func ScalarLoad(s []float32) Block {
	return Block(*(*[Lanes]float32)(s[:Lanes]))
}

// ScalarLoadPart loads min(len(s), Lanes) elements from s and fills the
// remaining lanes with +0, matching LoadFloat32x16Part. This is the
// edge-kernel path for M%MR and N%NR remainders (DESIGN.md §4/P3).
func ScalarLoadPart(s []float32) Block {
	var r Block
	n := len(s)
	if n > Lanes {
		n = Lanes
	}
	copy(r[:n], s[:n])
	return r
}

// ScalarStore stores all Lanes elements of x into s, requiring
// len(s) >= Lanes (see ScalarLoad on why this panics rather than truncates).
func ScalarStore(s []float32, x Block) {
	copy(s[:Lanes], x[:])
}

// ScalarStorePart stores as many of x's lanes as fit in s, matching
// StorePart.
func ScalarStorePart(s []float32, x Block) {
	n := len(s)
	if n > Lanes {
		n = Lanes
	}
	copy(s[:n], x[:n])
}

// ScalarAdd returns x+y lanewise.
func ScalarAdd(x, y Block) Block {
	var r Block
	for i := range r {
		r[i] = x[i] + y[i]
	}
	return r
}

// ScalarSub returns x-y lanewise.
func ScalarSub(x, y Block) Block {
	var r Block
	for i := range r {
		r[i] = x[i] - y[i]
	}
	return r
}

// ScalarMul returns x*y lanewise.
func ScalarMul(x, y Block) Block {
	var r Block
	for i := range r {
		r[i] = x[i] * y[i]
	}
	return r
}

// ScalarMulAdd returns x*y+z lanewise with a *single* rounding — true fused
// multiply-add semantics, matching VFMADD*PS.
//
// Numerics: it is computed as float32(math.FMA(float64...)) rather than as
// x[i]*y[i] + z[i], for two reasons.
//
// First, correctness of the spec. Go's compiler is *permitted* to fuse a
// float32 multiply-add on its own, and does on arm64 where FMADD exists, so
// plain `x*y + z` would make the spec's rounding depend on the host
// architecture — the one thing a reference implementation must not do.
// math.FMA pins it to one rounding everywhere.
//
// Second, the double rounding here is provably harmless. float64 carries 53
// mantissa bits; the exact product of two float32s needs at most 48, so
// math.FMA returns the correctly-rounded float64 of the exact sum, and
// rounding that to float32 agrees with rounding the exact value straight to
// float32 whenever the intermediate format has at least 2p+2 = 50 bits. It
// has 53. So this is bit-exact with hardware FMA, not merely close.
//
// Being the spec, this is slower than a plain mul+add. That is the right
// trade for internal/vec's scalar twin; the scalar *kernel* (internal/kern)
// is where scalar-path throughput is bought back.
func ScalarMulAdd(x, y, z Block) Block {
	var r Block
	for i := range r {
		r[i] = float32(math.FMA(float64(x[i]), float64(y[i]), float64(z[i])))
	}
	return r
}

// ScalarMax returns the lanewise maximum with x86 VMAXPS semantics rather
// than IEEE-754 maxNum: if either operand is NaN the result is y, and
// max(+0,-0) is y. Written as "if x > y then x else y", which reproduces
// both of those cases exactly.
//
// VERIFIED ON HARDWARE (2026-08-10): archsimd's x.Max(y) does put y in the
// second-source position, so the NaN and signed-zero results above are what
// the vector backends produce. Confirmed by the differential tests over every
// ordered pair from the NaN/±0 pool on both Zen 4 (Ryzen 9 7950X3D) and
// Skylake-X (Core i9-9960X) — two microarchitectures, since this is a
// property of the ISA and not of one chip. Disassembly could not settle it;
// only execution could. See docs/hosts.md and docs/toolchain-notes.md T6.
func ScalarMax(x, y Block) Block {
	var r Block
	for i := range r {
		if x[i] > y[i] {
			r[i] = x[i]
		} else {
			r[i] = y[i]
		}
	}
	return r
}

// ScalarMin returns the lanewise minimum with x86 VMINPS semantics; see
// ScalarMax for the NaN/signed-zero rule and for the hardware verification of
// the operand order, which covered Min on the same inputs.
func ScalarMin(x, y Block) Block {
	var r Block
	for i := range r {
		if x[i] < y[i] {
			r[i] = x[i]
		} else {
			r[i] = y[i]
		}
	}
	return r
}

// signMask32 is the float32 sign bit.
const signMask32 uint32 = 1 << 31

// ScalarAbs clears the sign bit of every lane.
//
// It is defined bitwise rather than as "negate if negative", so -0 becomes
// +0 and NaN payloads survive with the sign cleared — which is what the AVX
// backends do, since they implement abs by masking the sign bit off. (The
// archsimd API has no float32 Abs and in fact no float32 bitwise ops at all,
// so the vector path detours through the integer vector types; see
// docs/toolchain-notes.md.)
func ScalarAbs(x Block) Block {
	var r Block
	for i := range r {
		r[i] = math.Float32frombits(math.Float32bits(x[i]) &^ signMask32)
	}
	return r
}

// ScalarHSum returns the sum of all lanes using a fixed pairwise-halving
// tree: lanes i and i+8 first, then i and i+4, then i and i+2, then the
// final pair.
//
// The order is part of the spec. It is precisely the order a vector reduce
// produces — fold the upper half onto the lower half, repeatedly — so every
// backend's HSum agrees with this one bit-for-bit and the differential test
// can require exact equality. A left-to-right sum would have been the
// obvious scalar choice, and would have forced a tolerance onto the
// comparison for no gain.
//
// Reduction error growth is what the f(n)=n term in DESIGN.md §5's tolerance
// model is sized against; that applies at the routine level in P1, not here,
// where agreement is exact.
func ScalarHSum(x Block) float32 {
	acc := x
	for half := Lanes / 2; half >= 1; half /= 2 {
		for i := 0; i < half; i++ {
			acc[i] += acc[i+half]
		}
	}
	return acc[0]
}

// ---------------------------------------------- the fringe add-back (#22 C)
//
// C = C + tile for the live sub-rectangle of a fringe or mask-crossing tile,
// the operation internal/block performed inline with a `for j, v := range src`
// loop until #22's candidate C moved it here. Two entry points, because the
// two shapes of live region have different call counts (see AddTile512's
// comment in edge_amd64.go for the measurement that decides between them):
// a rectangle, and a single row.
//
// Numerics: exact, not merely within tolerance. One add per element in the
// same order as any backend's, no reassociation available to a vector form
// because each output element is the sum of exactly two inputs. So the
// differential test demands bit-equality, and a backend that reassociated
// anything here would be a bug rather than a tolerance question.

// ScalarAddTile adds the im×jn top-left sub-rectangle of tile into c.
//
// tile's rows are nr apart and c's are ldc apart; both are the strides the
// caller already has, so no copy and no per-row slicing happens above this.
func ScalarAddTile(c []float32, ldc int, tile []float32, nr, im, jn int) {
	for i := 0; i < im; i++ {
		dst := c[i*ldc : i*ldc+jn]
		src := tile[i*nr : i*nr+jn]
		for j, v := range src {
			dst[j] += v
		}
	}
}

// ScalarAddRow adds src into dst elementwise. len(dst) must be at least
// len(src); the extra is untouched, which is what lets a masked row pass the
// live [lo, hi) window of a row whose destination continues past it.
func ScalarAddRow(dst, src []float32) {
	for j, v := range src {
		dst[j] += v
	}
}
