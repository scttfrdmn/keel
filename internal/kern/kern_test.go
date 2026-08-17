// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern_test

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// tile is a float64 reference for C += A·B over one packed tile, plus the error
// scale each output element's bound should be measured against.
//
// It is here rather than in internal/oracle because the microkernel tile is not
// a public routine; what comes from oracle is the tolerance model, which is the
// part that must not be reinvented per test. The scale is Σ|a·b| over the k-loop
// plus |c_in|, i.e. the sum of the magnitudes actually accumulated — not
// |result| — because a tile whose products cancel has a small result and a large
// bound, and substituting the result would call a wrong answer correct.
func tile(mr, nr, kc int, a, b, cIn []float32, ldc int) (want, scale []float64) {
	want = make([]float64, mr*nr)
	scale = make([]float64, mr*nr)
	for i := 0; i < mr; i++ {
		for j := 0; j < nr; j++ {
			v := float64(cIn[i*ldc+j])
			s := math.Abs(v)
			for p := 0; p < kc; p++ {
				t := float64(a[p*mr+i]) * float64(b[p*nr+j])
				v += t
				s += math.Abs(t)
			}
			want[i*nr+j] = v
			scale[i*nr+j] = s
		}
	}
	return want, scale
}

// panels builds packed A and B panels and a C block with the tile embedded in a
// larger row-major buffer, so that a kernel writing outside its tile is caught.
//
// C is deliberately given a stride wider than the tile in some cases: ldc == nr
// makes the tile contiguous, which would let an off-by-one row write land inside
// the next row and go unnoticed.
func panels(rng *rand.Rand, mr, nr, kc, ldc, rows int) (a, b, c []float32) {
	a = make([]float32, kc*mr)
	b = make([]float32, kc*nr)
	c = make([]float32, rows*ldc)
	for i := range a {
		a[i] = rng.Float32()*2 - 1
	}
	for i := range b {
		b[i] = rng.Float32()*2 - 1
	}
	for i := range c {
		c[i] = rng.Float32()*2 - 1
	}
	return a, b, c
}

// finite gates the tolerance comparisons below, and it has to run before them
// rather than beside them.
//
// `math.Abs(got-want) > tol` is false when either side is NaN, because every
// ordering comparison against NaN is false. So a kernel emitting NaN passed both
// the oracle test and the differential test — silently, and in exactly the two
// tests whose whole purpose is to catch a kernel producing the wrong bits (#98).
// panels() generates only ordinary values in [-1, 1) and kc never exceeds 130, so
// neither the inputs nor the float64 oracle can reach a non-finite value: one here
// can only have come from the kernel under test.
//
// This is the reason the check is a separate statement instead of a wider
// tolerance. "Is this a NaN" is not an approximation, and a bound loose enough to
// notice one would be loose enough to hide a real numerical fault.
func finite(t *testing.T, what string, vals ...float64) bool {
	t.Helper()
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("%s: non-finite value %v from finite inputs — the kernel produced it, "+
				"since panels() generates only values in [-1, 1)", what, v)
			return false
		}
	}
	return true
}

// kcs covers the three things kc can be: shorter than one unrolled pass, an
// exact multiple of the unroll, and a multiple plus a remainder. Every kernel's
// remainder loop is reached by at least one of them, and kc=0 has to leave C
// alone rather than fault.
func kcs(unroll int) []int {
	set := map[int]bool{0: true, 1: true, 2: true, 3: true, 128: true}
	for m := 1; m <= 3; m++ {
		set[m*unroll] = true
		set[m*unroll+1] = true
	}
	set[128+unroll-1] = true
	var out []int
	for k := 0; k <= 130; k++ {
		if set[k] {
			out = append(out, k)
		}
	}
	return out
}

func TestKernelsAgainstOracle(t *testing.T) {
	for _, k := range kern.Measured() {
		for _, ldcExtra := range []int{0, 7} {
			for _, kc := range kcs(k.Unroll) {
				name := fmt.Sprintf("%s/kc=%d/ldc=%d", k.ID(), kc, k.NR+ldcExtra)
				t.Run(name, func(t *testing.T) {
					rng := rand.New(rand.NewSource(int64(kc*131 + ldcExtra)))
					ldc := k.NR + ldcExtra
					rows := k.MR + 2 // two guard rows past the tile
					a, b, c := panels(rng, k.MR, k.NR, kc, ldc, rows)

					want, scale := tile(k.MR, k.NR, kc, a, b, c, ldc)
					guard := make([]float32, len(c))
					copy(guard, c)

					k.Fn(kc, a, b, c, ldc)

					for i := 0; i < k.MR; i++ {
						for j := 0; j < k.NR; j++ {
							got := float64(c[i*ldc+j])
							w := want[i*k.NR+j]
							tol := oracle.Tolerance(kc+1, scale[i*k.NR+j])
							finite(t, fmt.Sprintf("C[%d][%d]", i, j), got, w)
							if math.Abs(got-w) > tol {
								t.Fatalf("C[%d][%d] = %v, want %v (|diff| %g > tol %g)",
									i, j, got, w, math.Abs(got-w), tol)
							}
						}
					}
					// Everything outside the tile must be untouched: the guard
					// rows, and the columns past NR in every row of the tile.
					for i := 0; i < rows; i++ {
						for j := 0; j < ldc; j++ {
							if i < k.MR && j < k.NR {
								continue
							}
							if c[i*ldc+j] != guard[i*ldc+j] {
								t.Fatalf("wrote outside the tile at [%d][%d]: %v became %v",
									i, j, guard[i*ldc+j], c[i*ldc+j])
							}
						}
					}
				})
			}
		}
	}
}

// TestVectorAgainstScalarTile is the differential test DESIGN.md §5 asks for:
// every vector kernel against the scalar twin of its own shape, on the same
// inputs, held to the same tolerance model. Both are checked against the float64
// oracle in TestKernelsAgainstOracle; this one catches a disagreement that
// happens to fall inside the oracle's bound on both sides, which the per-kernel
// test cannot see.
func TestVectorAgainstScalarTile(t *testing.T) {
	for _, k := range kern.Measured() {
		if k.Name == kern.Scalar {
			continue
		}
		ref := k.Ref()
		for _, kc := range kcs(k.Unroll) {
			t.Run(fmt.Sprintf("%s/kc=%d", k.ID(), kc), func(t *testing.T) {
				rng := rand.New(rand.NewSource(int64(kc)*7919 + 3))
				ldc := k.NR + 3
				a, b, c := panels(rng, k.MR, k.NR, kc, ldc, k.MR)
				cRef := make([]float32, len(c))
				copy(cRef, c)

				_, scale := tile(k.MR, k.NR, kc, a, b, c, ldc)
				k.Fn(kc, a, b, c, ldc)
				ref.Fn(kc, a, b, cRef, ldc)

				for i := 0; i < k.MR; i++ {
					for j := 0; j < k.NR; j++ {
						got, want := float64(c[i*ldc+j]), float64(cRef[i*ldc+j])
						// Both paths round; the bound has to admit both, so it
						// is the same model applied twice.
						tol := 2 * oracle.Tolerance(kc+1, scale[i*k.NR+j])
						finite(t, fmt.Sprintf("%s vs %s: C[%d][%d]", k.ID(), ref.ID(), i, j), got, want)
						if math.Abs(got-want) > tol {
							t.Fatalf("%s vs %s: C[%d][%d] = %v, scalar %v (|diff| %g > tol %g)",
								k.ID(), ref.ID(), i, j, got, want, math.Abs(got-want), tol)
						}
					}
				}
			})
		}
	}
}

// TestBackendCoverage prints the marker the gate parses. A gate that only knows
// "the tests passed" cannot tell a green run on an AVX-512 host from a green run
// on a machine where every vector kernel was compiled out.
func TestBackendCoverage(t *testing.T) {
	b := kern.Backends()
	if len(b) == 0 {
		t.Fatal("no kernel backends at all")
	}
	var ids []string
	for _, k := range kern.Measured() {
		ids = append(ids, k.ID())
	}
	t.Logf("keel-kern-backends-exercised: %s", strings.Join(b, " "))
	t.Logf("keel-kern-shapes-exercised: %s", strings.Join(ids, " "))
}

// TestScalarTileRejectsUnsupportedShape checks that the reference refuses a
// shape it cannot hold rather than truncating. A reference that silently
// computed a smaller tile would make every differential test pass by agreeing
// with nothing.
func TestScalarTileRejectsUnsupportedShape(t *testing.T) {
	for _, s := range [][2]int{{0, 32}, {kern.MaxMR + 1, 32}, {2, 0}, {2, kern.MaxNR + 1}} {
		mr, nr := s[0], s[1]
		t.Run(fmt.Sprintf("%dx%d", mr, nr), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("ScalarTile(%d, %d, …) did not panic", mr, nr)
				}
			}()
			kern.ScalarTile(mr, nr, 1, make([]float32, 64), make([]float32, 64), make([]float32, 4096), 64)
		})
	}
}
