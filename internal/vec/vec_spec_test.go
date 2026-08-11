// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

import (
	"math"
	"testing"
)

// Characterization tests for the scalar spec itself.
//
// The differential tests hold the vector backends *equal to* the spec, which
// is worthless if the spec's own semantics are not what the comments claim.
// These tests pin those claims. They have no build tag and no CPU
// requirement, so they run on a stock toolchain and on every GOARCH — which
// is what makes the spec reviewable on hardware that cannot execute a single
// AVX-512 instruction.
//
// Constants below are written as bit patterns because that is the only way to
// say "exactly 2^-24" without trusting a decimal literal's rounding.
const (
	bitsPow2Neg12 = 0x39800000 // 2^-12
	bitsPow2Neg11 = 0x3A000000 // 2^-11
	bitsPow2Neg24 = 0x33800000 // 2^-24
)

// TestSpecMulAddIsFused proves ScalarMulAdd rounds once rather than twice —
// the property that makes it a legitimate twin for VFMADD*PS, and the one
// that would silently rot if someone "simplified" it to x*y + z.
//
// The witness: let a = 1 + 2^-12, so a*a = 1 + 2^-11 + 2^-24 exactly. That
// needs a 24th fractional bit, which float32 does not have, and it sits
// exactly halfway between its neighbours, so an unfused multiply rounds it
// to 1 + 2^-11 (ties-to-even). Adding c = -(1 + 2^-11) then yields exactly 0.
// A fused op instead adds c to the *unrounded* product and returns 2^-24.
// Two different answers from the same three inputs; the spec must give the
// fused one.
func TestSpecMulAddIsFused(t *testing.T) {
	a := float32(1) + math.Float32frombits(bitsPow2Neg12)
	c := -(float32(1) + math.Float32frombits(bitsPow2Neg11))
	wantFused := math.Float32frombits(bitsPow2Neg24)

	// Guard the witness itself: if the compiler or the platform changed such
	// that the unfused route no longer differs, this test would pass
	// vacuously and stop protecting anything.
	unfused := a*a + c
	if unfused == wantFused {
		t.Fatalf("witness is no longer discriminating: unfused a*a+c = %v, "+
			"which already equals the fused answer %v", unfused, wantFused)
	}

	got := ScalarMulAdd(
		ScalarBroadcast(a),
		ScalarBroadcast(a),
		ScalarBroadcast(c),
	)
	for i, v := range got {
		if v != wantFused {
			t.Errorf("lane %d: ScalarMulAdd(1+2^-12, 1+2^-12, -(1+2^-11)) = %s, want %s (2^-24); "+
				"spec is not rounding once",
				i, fmtF32(v), fmtF32(wantFused))
			break
		}
	}
	t.Logf("fused = %s, unfused a*a+c = %s (they must differ)",
		fmtF32(wantFused), fmtF32(unfused))
}

// TestSpecHSumIsPairwiseTree proves ScalarHSum folds in the documented
// halving order rather than summing left to right. The order is load-bearing:
// it is what lets the differential tests demand bit-exact agreement with a
// vector reduce instead of falling back to a tolerance.
//
// The witness: 2^24 followed by fifteen 1s. Summed left to right, every
// increment of 1 is exactly half an ULP at 2^24 and ties back down to 2^24,
// so the total is 2^24. Folded pairwise, the 1s combine with each other first
// and the result is 2^24 + 14.
func TestSpecHSumIsPairwiseTree(t *testing.T) {
	var b Block
	b[0] = 1 << 24
	for i := 1; i < Lanes; i++ {
		b[i] = 1
	}

	var leftToRight float32
	for _, v := range b {
		leftToRight += v
	}
	tree := referenceTree(b)

	if leftToRight == tree {
		t.Fatalf("witness is no longer discriminating: left-to-right and "+
			"pairwise both give %v", tree)
	}
	if got := ScalarHSum(b); got != tree {
		t.Errorf("ScalarHSum = %s, want %s (pairwise tree); left-to-right would give %s",
			fmtF32(got), fmtF32(tree), fmtF32(leftToRight))
	}
	t.Logf("pairwise = %s, left-to-right = %s", fmtF32(tree), fmtF32(leftToRight))
}

// TestSpecHSumMatchesTreeOnPool checks ScalarHSum against an independently
// written tree over the whole adversarial pool, so the implementation cannot
// drift from the documented order in some corner the single witness misses.
func TestSpecHSumMatchesTreeOnPool(t *testing.T) {
	for _, b := range blockPool() {
		if got, want := ScalarHSum(b), referenceTree(b); !sameF32(got, want) {
			t.Fatalf("ScalarHSum disagrees with reference tree\n in:   %s\n got:  %s\n want: %s",
				fmtBlock(b), fmtF32(got), fmtF32(want))
		}
	}
}

// referenceTree is a deliberately naive, separately-written statement of the
// fold order: pair lane i with lane i+half, halving until one value remains.
// Kept independent of ScalarHSum so the two can disagree.
func referenceTree(b Block) float32 {
	vals := append([]float32(nil), b[:]...)
	for len(vals) > 1 {
		half := len(vals) / 2
		next := make([]float32, half)
		for i := 0; i < half; i++ {
			next[i] = vals[i] + vals[i+half]
		}
		vals = next
	}
	return vals[0]
}

// TestSpecAbsIsBitwise pins ScalarAbs to sign-bit masking: -0 must become +0,
// and a NaN must stay a NaN with its sign cleared. "Negate if negative" would
// get -0 wrong (comparisons say -0 is not negative) and is the mistake this
// test exists to catch.
func TestSpecAbsIsBitwise(t *testing.T) {
	negZero := float32(math.Copysign(0, -1))
	negNaN := math.Float32frombits(math.Float32bits(float32(math.NaN())) | signMask32)

	cases := []struct {
		name string
		in   float32
		want uint32
	}{
		{"-0 becomes +0", negZero, 0},
		{"+0 stays +0", 0, 0},
		{"-1 becomes 1", -1, math.Float32bits(1)},
		{"-Inf becomes +Inf", float32(math.Inf(-1)), math.Float32bits(float32(math.Inf(1)))},
		{"sign-bit NaN keeps payload, loses sign", negNaN,
			math.Float32bits(negNaN) &^ signMask32},
		{"smallest denormal is preserved", -math.Float32frombits(1), 1},
	}
	for _, c := range cases {
		got := ScalarAbs(ScalarBroadcast(c.in))[0]
		if math.Float32bits(got) != c.want {
			t.Errorf("%s: ScalarAbs(%s) = %s, want bits %#08x",
				c.name, fmtF32(c.in), fmtF32(got), c.want)
		}
	}
}

// TestSpecMaxMinNaNAndSignedZero pins the VMAXPS/VMINPS-shaped semantics the
// spec claims: NaN in either operand yields y, and max(+0,-0) yields y. These
// are the cases where x86 differs from IEEE-754 maxNum and from Go's
// math.Max, so getting them wrong would be invisible in ordinary data and
// wrong at the edges.
//
// The operand-order half of this claim is still unverified against hardware —
// see ScalarMax's doc comment and docs/toolchain-notes.md. This test locks in
// what the spec currently says so that the first differential run on an amd64
// host either confirms it or produces a specific, legible failure.
func TestSpecMaxMinNaNAndSignedZero(t *testing.T) {
	nan := float32(math.NaN())
	posZero, negZero := float32(0), float32(math.Copysign(0, -1))

	cases := []struct {
		name    string
		x, y    float32
		max     float32
		min     float32
		compare func(got, want float32) bool
	}{
		{name: "NaN as x yields y", x: nan, y: 3, max: 3, min: 3, compare: bitsEqual},
		{name: "NaN as y yields y", x: 3, y: nan, max: nan, min: nan, compare: bothNaN},
		{name: "max(+0,-0) yields y", x: posZero, y: negZero, max: negZero, min: negZero, compare: bitsEqual},
		{name: "max(-0,+0) yields y", x: negZero, y: posZero, max: posZero, min: posZero, compare: bitsEqual},
		{name: "ordinary ordering", x: -2, y: 5, max: 5, min: -2, compare: bitsEqual},
	}
	for _, c := range cases {
		gotMax := ScalarMax(ScalarBroadcast(c.x), ScalarBroadcast(c.y))[0]
		if !c.compare(gotMax, c.max) {
			t.Errorf("%s: ScalarMax(%s, %s) = %s, want %s",
				c.name, fmtF32(c.x), fmtF32(c.y), fmtF32(gotMax), fmtF32(c.max))
		}
		gotMin := ScalarMin(ScalarBroadcast(c.x), ScalarBroadcast(c.y))[0]
		if !c.compare(gotMin, c.min) {
			t.Errorf("%s: ScalarMin(%s, %s) = %s, want %s",
				c.name, fmtF32(c.x), fmtF32(c.y), fmtF32(gotMin), fmtF32(c.min))
		}
	}
}

func bitsEqual(got, want float32) bool {
	return math.Float32bits(got) == math.Float32bits(want)
}

func bothNaN(got, want float32) bool {
	return math.IsNaN(float64(got)) && math.IsNaN(float64(want))
}

// TestSpecDenormalsSurvive checks that the spec's arithmetic does not flush
// denormals to zero. Nothing in Go should, but keel's whole reason to exist is
// numerical results people trust, and FTZ/DAZ behaviour is exactly the kind of
// thing that changes underneath a project when a build flag or a platform
// changes. Cheap to assert, expensive to discover late.
func TestSpecDenormalsSurvive(t *testing.T) {
	tiny := math.Float32frombits(1)         // smallest positive denormal
	big := math.Float32frombits(0x00400000) // a mid-range denormal

	if got := ScalarAdd(ScalarBroadcast(tiny), ScalarBroadcast(tiny))[0]; got != 2*tiny {
		t.Errorf("denormal add flushed: %s + %s = %s", fmtF32(tiny), fmtF32(tiny), fmtF32(got))
	}
	if got := ScalarMul(ScalarBroadcast(big), ScalarBroadcast(0.5))[0]; got != big/2 {
		t.Errorf("denormal mul flushed: %s * 0.5 = %s", fmtF32(big), fmtF32(got))
	}
	if got := ScalarMulAdd(ScalarBroadcast(tiny), ScalarBroadcast(1), ScalarZero())[0]; got != tiny {
		t.Errorf("denormal fused-multiply-add flushed: got %s, want %s", fmtF32(got), fmtF32(tiny))
	}
}

// TestSpecLoadStorePartial covers the partial memory ops at every length
// across the width boundary, including the empty slice.
func TestSpecLoadStorePartial(t *testing.T) {
	for n := 0; n <= Lanes+2; n++ {
		src := make([]float32, n)
		for i := range src {
			src[i] = float32(i + 1)
		}
		got := ScalarLoadPart(src)

		want := min(n, Lanes)
		for i := 0; i < want; i++ {
			if got[i] != float32(i+1) {
				t.Errorf("LoadPart(len=%d) lane %d = %s, want %v", n, i, fmtF32(got[i]), i+1)
			}
		}
		// Everything past the source length must be exactly +0, not stale data.
		for i := want; i < Lanes; i++ {
			if math.Float32bits(got[i]) != 0 {
				t.Errorf("LoadPart(len=%d) lane %d = %s, want +0 fill", n, i, fmtF32(got[i]))
			}
		}

		dst := make([]float32, n)
		full := ScalarBroadcast(7)
		ScalarStorePart(dst, full)
		for i := range dst {
			if i < Lanes && dst[i] != 7 {
				t.Errorf("StorePart(len=%d) index %d = %s, want 7", n, i, fmtF32(dst[i]))
			}
		}
	}
}

// TestSpecLoadRequiresFullWidth pins the panic contract: a short slice into
// the full-width Load is a caller bug (kernels pre-slice panels to exact
// length), not a case to zero-fill silently. Silent zero-filling here would
// turn a packing bug into wrong numbers instead of a crash.
func TestSpecLoadRequiresFullWidth(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("ScalarLoad on a short slice did not panic; the full-width " +
				"load must not silently zero-fill (use ScalarLoadPart for that)")
		}
	}()
	ScalarLoad(make([]float32, Lanes-1))
}

// TestSpecAvailableIncludesScalar checks the dispatch list is never empty and
// always ends in the always-present scalar backend.
func TestSpecAvailableIncludesScalar(t *testing.T) {
	av := Available()
	if len(av) == 0 {
		t.Fatal("Available() is empty; the scalar backend is unconditional")
	}
	if av[len(av)-1] != BackendScalar {
		t.Errorf("Available() = %v; scalar must be last (the fallback)", av)
	}
	t.Logf("available backends here: %v", av)
}
