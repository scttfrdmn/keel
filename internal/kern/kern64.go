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
// It carries the fields the middle reads (MR, NR, Fn), the two that name it in a
// test (Name, Unroll), and InsnsPerFMA, which ranks it exactly as Kernel's field
// ranks a float32 shape (see that field's doc: an audited count, recomputed by the
// gate, zero meaning unaudited).
type Kernel64 struct {
	Name        string // backend: Scalar or AVX512
	MR, NR      int    // tile shape
	Unroll      int    // k-steps per pass of the steady-state loop
	Fn          func(kc int, a, b, c []float64, ldc int)
	InsnsPerFMA float64
	ALanes      int // Kernel.ALanes at float64
}

// APanelLen is Kernel.APanelLen at float64.
func (k Kernel64) APanelLen(kc int) int { return k.MR * kc * max(1, k.ALanes) }

// ExpandA is Kernel.ExpandA at float64.
func (k Kernel64) ExpandA(a []float64) []float64 {
	if k.ALanes <= 1 {
		return a
	}
	o := make([]float64, len(a)*k.ALanes)
	for i, v := range a {
		for l := 0; l < k.ALanes; l++ {
			o[i*k.ALanes+l] = v
		}
	}
	return o
}

// nativeLanes is Kernel.nativeLanes at float64: the backend's register width in
// bytes over eight. Stated as bytes so the ISA fact is written once per backend
// and the lane count derives from the element size, rather than a second table
// of lane counts that can disagree with the first.
func (k Kernel64) nativeLanes() int {
	switch k.Name {
	case AVX512:
		return 64 / 8 // a ZMM register
	case NEON:
		return 16 / 8 // a V register
	default:
		return 0
	}
}

// MemOpsPerFMA is Kernel.MemOpsPerFMA at float64: 1/MR + nativeLanes/NR.
func (k Kernel64) MemOpsPerFMA() float64 {
	lanes := k.nativeLanes()
	if k.MR <= 0 || k.NR <= 0 || lanes <= 0 {
		return 0
	}
	return 1/float64(k.MR) + float64(lanes)/float64(k.NR)
}

// Ref is the scalar reference for k's shape, as Kernel.Ref.
func (k Kernel64) Ref() Kernel64 {
	if k.ALanes > 1 {
		return ScalarKernelLanes64(k.MR, k.NR, k.ALanes)
	}
	return ScalarKernel64(k.MR, k.NR)
}

// ScalarKernelLanes64 is ScalarKernelLanes at float64.
func ScalarKernelLanes64(mr, nr, lanes int) Kernel64 {
	return Kernel64{
		Name:   Scalar,
		MR:     mr,
		NR:     nr,
		Unroll: 1,
		ALanes: lanes,
		Fn: func(kc int, a, b, c []float64, ldc int) {
			ScalarTileLanes64(mr, nr, lanes, kc, a, b, c, ldc)
		},
	}
}

// ScalarTileLanes64 is ScalarTileLanes at float64.
func ScalarTileLanes64(mr, nr, lanes, kc int, a, b, c []float64, ldc int) {
	if mr < 1 || mr > MaxMR || nr < 1 || nr > MaxNR || lanes < 1 {
		panic(fmt.Sprintf("kern: tile %dx%d lanes %d outside %dx%d", mr, nr, lanes, MaxMR, MaxNR))
	}
	var buf [MaxMR * MaxNR]float64
	acc := buf[: mr*nr : mr*nr]
	for p := 0; p < kc; p++ {
		bp := b[p*nr : p*nr+nr]
		for i := 0; i < mr; i++ {
			av := a[(p*mr+i)*lanes]
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

// Kernels64 is Kernels at float64: the vector shapes, then the scalar reference
// for each, or two scalar shapes when there is no float64 vector kernel.
func Kernels64() []Kernel64 {
	vs := vectorKernels64()
	out := make([]Kernel64, 0, len(vs)+2)
	out = append(out, vs...)
	seen := map[string]bool{}
	for _, v := range vs {
		if r := v.Ref(); !seen[r.ID()] {
			seen[r.ID()] = true
			out = append(out, r)
		}
	}
	if len(vs) == 0 {
		out = append(out, ScalarKernel64(2, 16), ScalarKernel64(4, 16))
	}
	return out
}

// Measured64 is Measured at float64.
func Measured64() []Kernel64 { return append(Kernels64(), referenceTiles64()...) }

// Preferred64 is Preferred at float64, through the same comparator.
func Preferred64(class Class, ks []Kernel64) (Kernel64, bool) {
	if len(ks) == 0 {
		return Kernel64{}, false
	}
	best := ks[0]
	for _, k := range ks[1:] {
		if better(class, k.InsnsPerFMA, k.MemOpsPerFMA(), best.InsnsPerFMA, best.MemOpsPerFMA()) {
			best = k
		}
	}
	return best, true
}

// Tile is "MRxNR", as Kernel.Tile.
func (k Kernel64) Tile() string { return fmt.Sprintf("%dx%d", k.MR, k.NR) }

// ID is "MRxNR/backend", as Kernel.ID.
func (k Kernel64) ID() string { return k.Tile() + pbSuffix(k.ALanes) + "/" + k.Name }

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
