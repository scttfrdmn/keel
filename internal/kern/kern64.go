// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern

import "fmt"

// Kernel64 is Kernel over float64: the seam the generated float64 middle
// (internal/block64, internal/pack64; tools/f64gen) is parameterized by. It is a
// second struct rather than a type parameter on Kernel because the element-type
// axis is cut by code generation, not generics (#190, docs/generic-middle.md
// §Refusal register): Go's dictionaries changed the float32 object code.
//
// It carries only the fields the middle reads (MR, NR, Fn) and the two that name
// it in a test (Name, Unroll). Dispatch, ranking and the audit numbers stay on
// Kernel until a float64 vector kernel exists to need them (#191).
type Kernel64 struct {
	Name   string // backend: Scalar until #191 adds a vector one
	MR, NR int    // tile shape
	Unroll int    // k-steps per pass of the steady-state loop
	Fn     func(kc int, a, b, c []float64, ldc int)
}

// Tile is "MRxNR", as Kernel.Tile.
func (k Kernel64) Tile() string { return fmt.Sprintf("%dx%d", k.MR, k.NR) }

// ID is "MRxNR/backend", as Kernel.ID.
func (k Kernel64) ID() string { return k.Tile() + "/" + k.Name }

// ScalarKernel64 is ScalarKernel over float64: the executable specification of an
// mr×nr tile, and until #191 the only float64 kernel there is.
func ScalarKernel64(mr, nr int) Kernel64 {
	return Kernel64{
		Name:   Scalar,
		MR:     mr,
		NR:     nr,
		Unroll: 1,
		Fn: func(kc int, a, b, c []float64, ldc int) {
			ScalarTile64(mr, nr, kc, a, b, c, ldc)
		},
	}
}

// ScalarTile64 is ScalarTile over float64, the same arithmetic in the same order.
// TestScalarTile64MatchesScalarTile holds the two to each other on inputs that
// are exact in both types, so a change to one that is not made to the other
// fails rather than drifting.
func ScalarTile64(mr, nr, kc int, a, b, c []float64, ldc int) {
	if mr < 1 || mr > MaxMR || nr < 1 || nr > MaxNR {
		panic(fmt.Sprintf("kern: tile %dx%d outside %dx%d", mr, nr, MaxMR, MaxNR))
	}
	var buf [MaxMR * MaxNR]float64
	acc := buf[: mr*nr : mr*nr]
	for p := 0; p < kc; p++ {
		ap := a[p*mr : p*mr+mr]
		bp := b[p*nr : p*nr+nr]
		for i, av := range ap {
			row := acc[i*nr : i*nr+nr]
			for j, bv := range bp {
				row[j] += av * bv
			}
		}
	}
	for i := 0; i < mr; i++ {
		row := c[i*ldc : i*ldc+nr]
		src := acc[i*nr : i*nr+nr]
		for j, v := range src {
			row[j] += v
		}
	}
}
