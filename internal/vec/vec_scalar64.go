// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

import "math"

// Scalar float64 backend: the executable spec for the float64 surface (#191),
// op for op the float64 twin of vec_scalar.go and held bit-exact against the
// vector backends by vec_diff64_test.go. No build tag, for the same reason as
// vec_scalar.go: it is the reference on a stock toolchain and every GOARCH.
//
// Every semantic choice below is the float32 spec's, restated for float64, and
// differs only where float64 itself changes the argument (ScalarDMulAdd).

// ScalarDZero returns the all-zero block (+0 in every lane, not -0).
func ScalarDZero() Block64 { return Block64{} }

// ScalarDBroadcast returns a block with v in every lane.
func ScalarDBroadcast(v float64) Block64 {
	var r Block64
	for i := range r {
		r[i] = v
	}
	return r
}

// ScalarDLoad loads Lanes64 elements from s, panicking if len(s) < Lanes64
// (the ScalarLoad contract, matching LoadFloat64x8).
func ScalarDLoad(s []float64) Block64 {
	return Block64(*(*[Lanes64]float64)(s[:Lanes64]))
}

// ScalarDLoadPart loads min(len(s), Lanes64) elements and fills the rest with
// +0, matching LoadFloat64x8Part.
func ScalarDLoadPart(s []float64) Block64 {
	var r Block64
	n := min(len(s), Lanes64)
	copy(r[:n], s[:n])
	return r
}

// ScalarDStore stores all Lanes64 elements of x into s, requiring
// len(s) >= Lanes64.
func ScalarDStore(s []float64, x Block64) {
	copy(s[:Lanes64], x[:])
}

// ScalarDStorePart stores as many of x's lanes as fit in s.
func ScalarDStorePart(s []float64, x Block64) {
	n := min(len(s), Lanes64)
	copy(s[:n], x[:n])
}

// ScalarDAdd returns x+y lanewise.
func ScalarDAdd(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = x[i] + y[i]
	}
	return r
}

// ScalarDSub returns x-y lanewise.
func ScalarDSub(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = x[i] - y[i]
	}
	return r
}

// ScalarDMul returns x*y lanewise.
func ScalarDMul(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = x[i] * y[i]
	}
	return r
}

// ScalarDDiv returns x/y lanewise; IEEE 754 on a zero divisor, as VDIVPD.
func ScalarDDiv(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = x[i] / y[i]
	}
	return r
}

// ScalarDMulAdd returns x*y+z lanewise with a single rounding, matching
// VFMADD*PD.
//
// This is where float64 differs from the float32 spec, and it is simpler:
// math.FMA is float64-native, so it IS the correctly-rounded fused op and there
// is no widening or double-rounding argument to make. Plain `x*y + z` is still
// forbidden for the T16 reason — Go may fuse it on arm64 and not on amd64,
// making the spec's rounding depend on the host.
func ScalarDMulAdd(x, y, z Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = math.FMA(x[i], y[i], z[i])
	}
	return r
}

// ScalarDMax returns the lanewise maximum with x86 VMAXPD semantics: if either
// operand is NaN the result is y, and max(+0,-0) is y. VMAXPD and VMAXPS share
// that rule (the Intel SDM defines both with the same pseudocode), so this is
// ScalarMax's "if x > y then x else y" unchanged; the hardware check is
// vec_diff64_test.go over every ordered pair of the NaN/±0 pool.
func ScalarDMax(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		if x[i] > y[i] {
			r[i] = x[i]
		} else {
			r[i] = y[i]
		}
	}
	return r
}

// ScalarDMin returns the lanewise minimum with x86 VMINPD semantics; see
// ScalarDMax.
func ScalarDMin(x, y Block64) Block64 {
	var r Block64
	for i := range r {
		if x[i] < y[i] {
			r[i] = x[i]
		} else {
			r[i] = y[i]
		}
	}
	return r
}

// signMask64 is the float64 sign bit.
const signMask64 uint64 = 1 << 63

// ScalarDAbs clears the sign bit of every lane: -0 becomes +0 and NaN payloads
// survive with the sign cleared, which is what the vector mask produces.
func ScalarDAbs(x Block64) Block64 {
	var r Block64
	for i := range r {
		r[i] = math.Float64frombits(math.Float64bits(x[i]) &^ signMask64)
	}
	return r
}

// ScalarDHSum returns the sum of all lanes by the fixed pairwise-halving tree
// (i+4, then i+2, then the final pair) that a vector reduce produces, so every
// backend agrees with it bit-for-bit. See ScalarHSum.
func ScalarDHSum(x Block64) float64 {
	acc := x
	for half := Lanes64 / 2; half >= 1; half /= 2 {
		for i := 0; i < half; i++ {
			acc[i] += acc[i+half]
		}
	}
	return acc[0]
}
