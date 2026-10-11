// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package l1

import (
	"math"
	"testing"
)

// ScalDiv has no public BLAS routine behind it — reference BLAS has SSCAL and no
// divide-by-scalar — so the oracle-driven sweep in the root l1_test.go cannot
// reach it, and neither can any gate criterion. This file is where it is held.

// bits compares as a bit pattern so -0 is distinguishable from 0 and a NaN
// payload is not silently accepted as equal to a different one. Two NaNs count
// as equal here *only* because the vec-level TestDiffBinary already pins the
// payloads bit-for-bit over every edge-value pair; at this level the claim being
// tested is the loop and the tail, not the lowering.
func sameF32(a, b float32) bool {
	if math.IsNaN(float64(a)) && math.IsNaN(float64(b)) {
		return true
	}
	return math.Float32bits(a) == math.Float32bits(b)
}

// TestScalDivIsExactAgainstPlainDivision holds every backend's ScalDiv to x/alpha
// computed in plain Go, with NO tolerance. That is a stronger claim than l1's
// package doc makes for its other ops, and it is available here for a reason:
// VDIVPS and VFDIV are per-lane IEEE divides, so unlike a reduction there is no
// reassociation and unlike an FMA there is no fusion. Anything other than
// equality would mean the lowering is not the divide it claims to be.
//
// The lengths run past every lane count and deliberately include the awkward
// ones — 0, 1, and n where n mod lanes is 1 — because each backend's tail is a
// separate code path from its body (LoadPart/StorePart), and a tail that scaled
// by a reciprocal while the body divided would pass any tolerance-based check.
func TestScalDivIsExactAgainstPlainDivision(t *testing.T) {
	alphas := []float32{
		1, -1, 2, -2, 0.5, 3.25, -1e6, 1e-6,
		0,                             // x/0 -> +/-Inf or NaN, and must not be special-cased
		float32(math.Copysign(0, -1)), // -0, so the sign of the Inf is checked too
		float32(math.Inf(1)),          // x/Inf -> 0
		float32(math.NaN()),           // NaN propagates
		math.SmallestNonzeroFloat32,   // overflows the quotient
		math.MaxFloat32,               // underflows it
	}
	// 0 and 1 are the degenerate cases; 17, 33 and 65 are one past the three
	// lane widths (4 NEON, 8 AVX2, 16 AVX-512) so the tail is non-empty on every
	// backend; 64 lands exactly on all three so the tail is empty on all three.
	lengths := []int{0, 1, 3, 4, 7, 8, 15, 16, 17, 31, 33, 63, 64, 65, 129}

	for _, b := range Backends() {
		t.Run(b.Name, func(t *testing.T) {
			for _, alpha := range alphas {
				for _, n := range lengths {
					src := make([]float32, n)
					for i := range src {
						// A mix of magnitudes and signs, plus an Inf and a NaN in
						// the input so 0/0 and Inf/Inf are both reached.
						switch i % 5 {
						case 0:
							src[i] = float32(i+1) * 1.5
						case 1:
							src[i] = -float32(i+1) * 0.25
						case 2:
							src[i] = 0
						case 3:
							src[i] = float32(math.Inf(1))
						default:
							src[i] = float32(math.NaN())
						}
					}
					want := make([]float32, n)
					for i, v := range src {
						want[i] = v / alpha
					}
					got := make([]float32, n)
					copy(got, src)
					b.ScalDiv(alpha, got)
					for i := range want {
						if !sameF32(got[i], want[i]) {
							t.Fatalf("ScalDiv(alpha=%v) n=%d lane %d: got %v (bits %#08x), want %v (bits %#08x)",
								alpha, n, i, got[i], math.Float32bits(got[i]),
								want[i], math.Float32bits(want[i]))
						}
					}
				}
			}
		})
	}
}

// TestScalDivIsNotAReciprocalMultiply is the assertion the comment on
// Kernels.ScalDiv makes, driven rather than stated. alpha = 49 is the smallest
// positive integer for which 1/49 is inexact in float32 AND x*(1/49) differs
// from x/49 for some x, so a backend that quietly scaled by a reciprocal would
// be caught here even though it would pass any tolerance-based comparison.
//
// Found by search rather than recalled: the loop below re-derives the witness on
// every run, so if a future float32 subtlety invalidates 49 the test says so
// instead of passing vacuously.
func TestScalDivIsNotAReciprocalMultiply(t *testing.T) {
	const alpha float32 = 49
	recip := 1 / alpha

	var witness float32
	for i := 1; i < 4096; i++ {
		x := float32(i)
		if x/alpha != x*recip {
			witness = x
			break
		}
	}
	if witness == 0 {
		t.Fatal("no x in [1,4096) distinguishes x/49 from x*(1/49) in float32; " +
			"this test can no longer tell a divide from a reciprocal multiply and needs a new alpha")
	}
	t.Logf("witness: %v/%v = %v, but %v*%v = %v", witness, alpha, witness/alpha, witness, recip, witness*recip)

	for _, b := range Backends() {
		t.Run(b.Name, func(t *testing.T) {
			// Long enough that the witness sits in the vector body on every
			// backend, not only in the tail.
			x := make([]float32, 64)
			for i := range x {
				x[i] = witness
			}
			b.ScalDiv(alpha, x)
			for i, v := range x {
				if v != witness/alpha {
					t.Fatalf("lane %d = %v, want %v (the divide); %v would be the reciprocal multiply",
						i, v, witness/alpha, witness*recip)
				}
			}
		})
	}
}

// TestMissingNamesEveryMandatoryOp drives the registration check that Kernels'
// doc comment has claimed since P1 and that nothing enforced until 2026-10-11.
// Adding the ScalDiv field left both amd64 backends nil and the tree BUILT; the
// defect would have surfaced as a nil-deref inside Strsm on an AVX-512 host and
// nowhere else, because no other caller uses the op.
//
// Each arm removes exactly one op from a complete set and requires that op to be
// named — a checker that returned a non-empty string for the wrong reason would
// pass a bare "is it non-empty" assertion.
func TestMissingNamesEveryMandatoryOp(t *testing.T) {
	if miss := ScalarKernels.missing(); miss != "" {
		t.Fatalf("the scalar reference backend is itself incomplete: missing %s", miss)
	}
	for _, tc := range []struct {
		name  string
		clear func(*Kernels)
	}{
		{"Dot", func(k *Kernels) { k.Dot = nil }},
		{"Axpy", func(k *Kernels) { k.Axpy = nil }},
		{"Scal", func(k *Kernels) { k.Scal = nil }},
		{"ScalDiv", func(k *Kernels) { k.ScalDiv = nil }},
		{"Asum", func(k *Kernels) { k.Asum = nil }},
		{"SumSq", func(k *Kernels) { k.SumSq = nil }},
		{"Name", func(k *Kernels) { k.Name = "" }},
	} {
		k := ScalarKernels
		tc.clear(&k)
		if got := k.missing(); got != tc.name {
			t.Errorf("a set missing %s reported %q", tc.name, got)
		}
	}
}

// And the registry itself is complete on this host, which is the property the
// panic in Backends defends. Separate from the unit test above because that one
// exercises the predicate and this one exercises the actual registrations —
// a correct predicate nobody wired reads the same as no predicate at all.
func TestEveryRegisteredBackendIsComplete(t *testing.T) {
	bs := Backends()
	if len(bs) == 0 {
		t.Fatal("no backends registered at all")
	}
	for _, b := range bs {
		if miss := b.missing(); miss != "" {
			t.Errorf("backend %s registered with nil %s", b.Name, miss)
		}
	}
	t.Logf("complete backends on this host: %d", len(bs))
}
