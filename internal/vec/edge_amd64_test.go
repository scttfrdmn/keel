// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import (
	"math"
	"math/rand"
	"testing"
)

// The differential test for issue #22's candidate C: AddTile512 and AddRow512
// against the scalar spec, bit for bit.
//
// Bit-equality rather than internal/oracle.Tolerance, and that is not this test
// being stricter for its own sake. Every output element here is the sum of
// exactly two inputs, so there is no summation order to choose: a backend that
// disagreed with the spec would be disagreeing about one IEEE add, which is a
// bug and not a rounding question. The tolerance model exists for reductions
// whose error grows with n (DESIGN.md §5), and n is 1 in this operation.
//
// The shapes swept are the ones internal/block actually produces: jn from 1 to
// 2*Lanes+1 to cross the vector width twice and land on both sides of it, im
// from 1 to MaxMR, and ldc strictly greater than jn — a fringe tile is by
// definition writing into a row that continues past the part it owns, and a
// helper that ignored ldc would pass every test with ldc == jn.

func TestDiffAddRow(t *testing.T) {
	if !HasAVX512() {
		t.Skip("no AVX-512 on this CPU")
	}
	rng := rand.New(rand.NewSource(0x22c))
	for n := 0; n <= 2*Lanes+1; n++ {
		src := make([]float32, n)
		want := make([]float32, n)
		got := make([]float32, n)
		for i := range src {
			src[i] = float32(rng.NormFloat64())
			v := float32(rng.NormFloat64())
			want[i], got[i] = v, v
		}
		ScalarAddRow(want, src)
		AddRow512(got, src)
		for i := range want {
			if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
				t.Fatalf("n=%d lane %d: AddRow512 = %v (%#x), spec = %v (%#x)",
					n, i, got[i], math.Float32bits(got[i]), want[i], math.Float32bits(want[i]))
			}
		}
	}
}

// TestAddRowLeavesTheTailAlone is the property a masked store could break
// silently: internal/block hands AddRow512 a destination that runs past the
// live window (the row of C continues, and beyond it the caller's own memory),
// so writing a full vector where a partial one was due would clobber somebody
// else's data rather than fail a comparison. The sentinel is what notices.
func TestAddRowLeavesTheTailAlone(t *testing.T) {
	if !HasAVX512() {
		t.Skip("no AVX-512 on this CPU")
	}
	const guard = 2 * Lanes
	for n := 0; n <= 2*Lanes+1; n++ {
		dst := make([]float32, n+guard)
		for i := range dst {
			dst[i] = float32(i) + 0.5
		}
		src := make([]float32, n)
		for i := range src {
			src[i] = 1
		}
		AddRow512(dst[:n+guard], src) // len(dst) > len(src): only n elements are live
		for i := n; i < n+guard; i++ {
			if want := float32(i) + 0.5; dst[i] != want {
				t.Fatalf("n=%d: AddRow512 wrote past its source at index %d: got %v, want %v",
					n, i, dst[i], want)
			}
		}
	}
}

func TestDiffAddTile(t *testing.T) {
	if !HasAVX512() {
		t.Skip("no AVX-512 on this CPU")
	}
	rng := rand.New(rand.NewSource(0x22d))
	for _, nr := range []int{32, 64} {
		for im := 1; im <= 8; im++ {
			for jn := 1; jn <= nr; jn++ {
				ldc := jn + 7 // the row continues past the live part, always
				tile := make([]float32, im*nr)
				for i := range tile {
					tile[i] = float32(rng.NormFloat64())
				}
				want := make([]float32, im*ldc)
				got := make([]float32, im*ldc)
				for i := range want {
					v := float32(rng.NormFloat64())
					want[i], got[i] = v, v
				}
				ScalarAddTile(want, ldc, tile, nr, im, jn)
				AddTile512(got, ldc, tile, nr, im, jn)
				for i := range want {
					if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
						t.Fatalf("nr=%d im=%d jn=%d ldc=%d index %d: AddTile512 = %v, spec = %v",
							nr, im, jn, ldc, i, got[i], want[i])
					}
				}
			}
		}
	}
}

// TestAddNonFinite is the one numerics case the fringe path exists for. A
// zero-padded panel meeting an infinity produces 0·Inf = NaN in a padding lane
// of the scratch tile, and the whole point of adding back only the live
// sub-rectangle is that such a lane never reaches C. Here the NaN is placed
// *inside* the live region, where it must propagate exactly as the spec does —
// the add-back may not launder a NaN either.
func TestAddNonFinite(t *testing.T) {
	if !HasAVX512() {
		t.Skip("no AVX-512 on this CPU")
	}
	inf := float32(math.Inf(1))
	nan := float32(math.NaN())
	for _, v := range []float32{inf, -inf, nan, 0, -0} {
		for n := 1; n <= Lanes+1; n++ {
			src := make([]float32, n)
			want := make([]float32, n)
			got := make([]float32, n)
			for i := range src {
				src[i] = v
				want[i], got[i] = 1, 1
			}
			ScalarAddRow(want, src)
			AddRow512(got, src)
			for i := range want {
				if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
					t.Fatalf("v=%v n=%d lane %d: AddRow512 = %#x, spec = %#x",
						v, n, i, math.Float32bits(got[i]), math.Float32bits(want[i]))
				}
			}
		}
	}
}
