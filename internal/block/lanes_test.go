// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// TestPreBroadcastIsBitIdentical drives every Level-3 routine through a
// pre-broadcast kernel (kern.Kernel.ALanes, #198) and through the conventional
// kernel of the same shape, and requires the same bits. The two scalar twins do the
// same arithmetic in the same order, so any difference is the layout path's: the
// expansion in macro, the panel stride, or a fringe or masked tile reading a copy it
// should not. Small blocks put several blocks and both fringes in every call; the
// odd tile puts a fringe on both axes.
func TestPreBroadcastIsBitIdentical(t *testing.T) {
	defer func(kc, mc, nc, mb int) { KC, MC, NC, MB = kc, mc, nc, mb }(KC, MC, NC, MB)
	KC, MC, NC, MB = 24, 16, 64, 16
	ax := func(alpha float32, x, y []float32) {
		for i, v := range x {
			y[i] += alpha * v
		}
	}
	sd := func(alpha float32, x []float32) {
		for i := range x {
			x[i] /= alpha
		}
	}
	rng := rand.New(rand.NewSource(198))
	mat := func(n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = float32(2*rng.Float64() - 1)
		}
		return v
	}
	tri := func(n int) []float32 { // well conditioned for the solves
		a := mat(n * n)
		for i := 0; i < n; i++ {
			a[i*n+i] = 4
		}
		return a
	}
	const m, n, k = 37, 29, 41
	for _, sh := range [][2]int{{5, 16}, {3, 5}} {
		conv, pb := kern.ScalarKernel(sh[0], sh[1]), kern.ScalarKernelLanes(sh[0], sh[1], 4)
		run := func(name string, f func(kn kern.Kernel) []float32) {
			t.Run(fmt.Sprintf("%dx%d/%s", sh[0], sh[1], name), func(t *testing.T) {
				want, got := f(conv), f(pb)
				for i := range want {
					if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
						t.Fatalf("element %d: pre-broadcast %v, conventional %v", i, got[i], want[i])
					}
				}
			})
		}
		a, b, c := mat(m*k), mat(k*n), mat(m*n)
		for _, tr := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			run(fmt.Sprintf("Gemm/tA=%v,tB=%v", tr[0], tr[1]), func(kn kern.Kernel) []float32 {
				cc := append([]float32(nil), c...)
				lda, ldb := k, n
				if tr[0] {
					lda = m
				}
				if tr[1] {
					ldb = k
				}
				Gemm(kn, tr[0], tr[1], m, n, k, 1.5, a, lda, b, ldb, -0.5, cc, n)
				return cc
			})
		}
		sq := mat(m * m)
		for _, lower := range []bool{true, false} {
			for _, trans := range []bool{false, true} {
				run(fmt.Sprintf("Syrk/lower=%v,trans=%v", lower, trans), func(kn kern.Kernel) []float32 {
					cc := append([]float32(nil), sq...)
					lda := k
					if trans {
						lda = m
					}
					Syrk(kn, lower, trans, m, k, 1.25, a, lda, 0.5, cc, m)
					return cc
				})
				run(fmt.Sprintf("Syr2k/lower=%v,trans=%v", lower, trans), func(kn kern.Kernel) []float32 {
					cc := append([]float32(nil), sq...)
					lda := k
					if trans {
						lda = m
					}
					Syr2k(kn, lower, trans, m, k, 1.25, a, lda, a, lda, 0.5, cc, m)
					return cc
				})
			}
			for _, left := range []bool{true, false} {
				d := m
				if !left {
					d = n
				}
				sa := mat(d * d)
				bb := mat(m * n)
				run(fmt.Sprintf("Symm/left=%v,lower=%v", left, lower), func(kn kern.Kernel) []float32 {
					cc := append([]float32(nil), c...)
					Symm(kn, left, lower, m, n, 0.75, sa, d, bb, n, -1, cc, n)
					return cc
				})
				ta := tri(d)
				for _, trans := range []bool{false, true} {
					for _, unit := range []bool{false, true} {
						tag := fmt.Sprintf("left=%v,lower=%v,trans=%v,unit=%v", left, lower, trans, unit)
						run("Trmm/"+tag, func(kn kern.Kernel) []float32 {
							x := append([]float32(nil), bb...)
							Trmm(kn, left, lower, trans, unit, m, n, 1.5, ta, d, x, n)
							return x
						})
						run("Trsm/"+tag, func(kn kern.Kernel) []float32 {
							x := append([]float32(nil), bb...)
							Trsm(kn, ax, sd, left, lower, trans, unit, m, n, 1.5, ta, d, x, n)
							return x
						})
					}
				}
			}
		}
	}
}
