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

// TestMemOpsPerFMA64 pins the float64 tile arithmetic to docs/f64-design.md §2's
// table: halving the columns at half the lanes keeps 1/MR + lanes/NR, so the
// float64 4x16 and 2x16 tiles score what the float32 4x32 and 2x32 do.
func TestMemOpsPerFMA64(t *testing.T) {
	for _, c := range []struct {
		k    kern.Kernel64
		want float64
	}{
		{kern.Kernel64{Name: kern.AVX512, MR: 4, NR: 16}, 0.75},
		{kern.Kernel64{Name: kern.AVX512, MR: 2, NR: 16}, 1.0},
		{kern.Kernel64{Name: kern.NEON, MR: 4, NR: 8}, 0.5},
		{kern.Kernel64{Name: kern.NEON, MR: 8, NR: 4}, 0.625},
		{kern.ScalarKernel64(4, 16), 0}, // no native width: unrankable, as at float32
	} {
		if got := c.k.MemOpsPerFMA(); got != c.want {
			t.Errorf("%s: MemOpsPerFMA %v, want %v", c.k.ID(), got, c.want)
		}
	}
}

// TestPreferred64SharesTheComparator holds Preferred64 to Preferred on the same
// shapes: the float64 ranking is the float32 rule, not a second rule. Each
// float32 shape has a float64 twin with the same MR, half the NR and the same
// audited insns/FMA, so the MemOpsPerFMA axis is identical and both registries
// must pick the twin of the same entry under both classes.
func TestPreferred64SharesTheComparator(t *testing.T) {
	k32 := []kern.Kernel{
		{Name: kern.AVX512, MR: 4, NR: 32, InsnsPerFMA: 6.25},
		{Name: kern.AVX512, MR: 2, NR: 32, InsnsPerFMA: 4.625},
		{Name: kern.Scalar, MR: 8, NR: 32}, // unaudited: never displaces an incumbent
	}
	k64 := make([]kern.Kernel64, len(k32))
	for i, k := range k32 {
		k64[i] = kern.Kernel64{Name: k.Name, MR: k.MR, NR: k.NR / 2, InsnsPerFMA: k.InsnsPerFMA}
	}
	for _, class := range []kern.Class{kern.ClassFMA, kern.ClassIssue} {
		a, _ := kern.Preferred(class, k32)
		b, _ := kern.Preferred64(class, k64)
		if a.MR != b.MR || a.NR != 2*b.NR {
			t.Errorf("%s: float32 picks %s, float64 picks %s — the comparators disagree", class, a.ID(), b.ID())
		}
	}
}

// TestVectorAgainstScalarTile64 is TestVectorAgainstScalarTile at float64: every
// float64 vector kernel against the scalar twin of its own shape, on the same
// inputs, with a guard region past NR that must come back untouched. Both sides
// are float64, so the bound is the peer-precision one (oracle.PeerTolerance64):
// each is within Tolerance64 of exact and their errors add.
//
// Until #191 ships a float64 vector kernel there is nothing to compare, and the
// test says so rather than passing silently.
func TestVectorAgainstScalarTile64(t *testing.T) {
	n := 0
	for _, k := range kern.Measured64() {
		if k.Name == kern.Scalar {
			continue
		}
		n++
		ref := k.Ref()
		for _, kc := range kcs(k.Unroll) {
			t.Run(fmt.Sprintf("%s/kc=%d", k.ID(), kc), func(t *testing.T) {
				rng := rand.New(rand.NewSource(int64(kc)*7919 + 5))
				ldc := k.NR + 3
				a, b := make([]float64, kc*k.MR), make([]float64, kc*k.NR)
				c := make([]float64, (k.MR-1)*ldc+k.NR+3) // the last row's guard too
				for i := range a {
					a[i] = 2*rng.Float64() - 1
				}
				for i := range b {
					b[i] = 2*rng.Float64() - 1
				}
				for i := range c {
					c[i] = 2*rng.Float64() - 1
				}
				before := append([]float64(nil), c...)
				cRef := append([]float64(nil), c...)
				k.Fn(kc, a, b, c, ldc)
				ref.Fn(kc, a, b, cRef, ldc)
				for i := range c {
					row, col := i/ldc, i%ldc
					if col >= k.NR || row >= k.MR {
						if math.Float64bits(c[i]) != math.Float64bits(before[i]) {
							t.Fatalf("%s: wrote c[%d], outside the %dx%d tile", k.ID(), i, k.MR, k.NR)
						}
						continue
					}
					scale := math.Abs(before[i])
					for p := 0; p < kc; p++ {
						scale += math.Abs(a[p*k.MR+row] * b[p*k.NR+col])
					}
					if d := math.Abs(c[i] - cRef[i]); !(d <= oracle.PeerTolerance64(kc+1, scale)) {
						t.Fatalf("%s vs %s: C[%d][%d] = %v, scalar %v (|diff| %g > tol %g)",
							k.ID(), ref.ID(), row, col, c[i], cRef[i], d, oracle.PeerTolerance64(kc+1, scale))
					}
				}
			})
		}
	}
	t.Logf("float64 vector kernels compared against their scalar twins: %d", n)
}

// TestBackendCoverage64 prints the float64 marker gate-p2 parses under
// KEEL_ELEM=f64, as TestBackendCoverage does at float32: which backends the
// float64 kernel tests actually ran, so a green run on a host where every float64
// vector kernel was compiled out cannot read as AVX-512 coverage.
func TestBackendCoverage64(t *testing.T) {
	seen := map[string]bool{}
	var b []string
	for _, k := range kern.Kernels64() {
		if !seen[k.Name] {
			seen[k.Name] = true
			b = append(b, k.Name)
		}
	}
	t.Logf("keel-kern64-backends-exercised: %s", strings.Join(b, " "))
}
