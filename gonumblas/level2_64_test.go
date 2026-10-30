// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel"
	"gonum.org/v1/gonum/blas"
)

// The third arm of DESIGN.md §5 rule 1 for float64 Level 2 (#192): every keel D
// Level-2 routine, called directly (the blas.Float64 adapter is a later stage),
// against Gonum's native float64 on the same operands. As TestBandPackedAgreeWithGonum
// does at float32, each call is judged at one global scale per call: n·max|A|·max|x|,
// plus |beta|·max|y|, and for the solves additionally times max|x̂|. That is looser
// per element than the oracle arms in package keel, which judge every element at its
// own scale; this arm's job is independence, not resolution. A NaN in storage a
// routine must not read propagates, so a stray read fails either way.

func maxAbs64(v []float64) float64 {
	m := 0.0
	for _, e := range v {
		if f := math.Abs(e); !math.IsNaN(f) && f > m {
			m = f
		}
	}
	return m
}

func cmp64(t *testing.T, what string, k, g []float64, n int, scale float64) {
	t.Helper()
	for i := range k {
		if !sameElem64(k[i], g[i], n, scale) {
			t.Errorf("%s: storage[%d]: keel %v, gonum %v (scale %.3g)", what, i, k[i], g[i], scale)
			return
		}
	}
}

// triDense64 returns an n×n row-major matrix at stride n+1, its diagonal well
// conditioned, the triangle a routine must not read (and a unit diagonal) NaN.
func triDense64(r *rand.Rand, n int, lower, unit bool) ([]float64, int) {
	lda := n + 1
	a := make([]float64, max(n*lda, 1))
	for i := range a {
		a[i] = math.NaN()
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			switch {
			case (lower && j > i) || (!lower && j < i) || (unit && i == j):
			case i == j:
				a[i*lda+j] = 4 + r.Float64()
			default:
				a[i*lda+j] = r.NormFloat64()
			}
		}
	}
	return a, lda
}

// emptyReduction asserts keel's documented deviation (Sgemv, Sgbmv): with an empty
// reduction keel returns y = beta·y, where gonum and reference BLAS leave y alone.
// keel's side of it is checked in package keel (TestDgemvDger); this pins gonum's.
func emptyReduction(t *testing.T, what string, yg, y []float64) {
	t.Helper()
	for i := range y {
		if yg[i] != y[i] && !(math.IsNaN(yg[i]) && math.IsNaN(y[i])) {
			t.Errorf("%s: gonum changed y[%d] on an empty reduction; the deviation this pins has moved", what, i)
		}
	}
}

func TestLevel2AgreesWithGonum64(t *testing.T) {
	r := rand.New(rand.NewSource(21))
	ts := []blas.Transpose{blas.NoTrans, blas.Trans}
	for _, n := range []int{0, 1, 5, 33} {
		for _, inc := range []int{1, -2} {
			x := randVec64(r, storage(n, inc))
			// Dense general: Dgemv both ways and Dger, at m = n+3.
			m := n + 3
			a := randVec64(r, max(m*(n+2), 1))
			for _, ta := range ts {
				rows, inner := m, n
				if ta == blas.Trans {
					rows, inner = n, m
				}
				xx, y := randVec64(r, storage(inner, inc)), randVec64(r, storage(rows, inc))
				yk, yg := append([]float64(nil), y...), append([]float64(nil), y...)
				keel.Dgemv(keel.Transpose(ta), m, n, -0.75, a, n+2, xx, inc, 0.5, yk, inc)
				gonumImpl.Dgemv(ta, m, n, -0.75, a, n+2, xx, inc, 0.5, yg, inc)
				if inner == 0 {
					emptyReduction(t, fmt.Sprintf("Dgemv %c n=%d inc=%d", ta, n, inc), yg, y)
					continue
				}
				cmp64(t, fmt.Sprintf("Dgemv %c n=%d inc=%d", ta, n, inc), yk, yg, inner+1,
					float64(max(inner, 1))*maxAbs64(a)*maxAbs64(xx)+maxAbs64(y))
			}
			// Dger only where gonum is right, as for Sger: gonum's amd64 f64.Ger, like
			// f32.Ger, mishandles a negative stride (gonum/gonum#2105, open; its fix PR
			// #2107 closed unmerged). Measured on janus 2026-10-29 with a 2×2 case: incY < 0
			// leaves A unchanged, and incX < 0 reads past x. Its pure-Go path is correct.
			// keel's Dger is held to the oracle on every stride in package keel
			// (TestDgemvDger, inc = -2).
			if inc > 0 {
				y := randVec64(r, storage(n, inc))
				xm := randVec64(r, storage(m, inc))
				ak, ag := append([]float64(nil), a...), append([]float64(nil), a...)
				keel.Dger(m, n, 1.5, xm, inc, y, inc, ak, n+2)
				gonumImpl.Dger(m, n, 1.5, xm, inc, y, inc, ag, n+2)
				cmp64(t, fmt.Sprintf("Dger n=%d inc=%d", n, inc), ak, ag, 2, maxAbs64(a)+1.5*maxAbs64(xm)*maxAbs64(y))
			}

			for _, ul := range []blas.Uplo{blas.Upper, blas.Lower} {
				lower := ul == blas.Lower
				for _, ta := range ts {
					for _, dg := range []blas.Diag{blas.NonUnit, blas.Unit} {
						ad, lda := triDense64(r, n, lower, dg == blas.Unit)
						sc := float64(max(n, 1)) * maxAbs64(ad) * maxAbs64(x)
						what := fmt.Sprintf("%c%c%c n=%d inc=%d", ul, ta, dg, n, inc)
						kU, kT, kD := keel.Uplo(ul), keel.Transpose(ta), keel.Diag(dg)
						xk, xg := append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtrmv(kU, kT, kD, n, ad, lda, xk, inc)
						gonumImpl.Dtrmv(ul, ta, dg, n, ad, lda, xg, inc)
						cmp64(t, "Dtrmv "+what, xk, xg, max(n, 1), sc)
						xk, xg = append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtrsv(kU, kT, kD, n, ad, lda, xk, inc)
						gonumImpl.Dtrsv(ul, ta, dg, n, ad, lda, xg, inc)
						cmp64(t, "Dtrsv "+what, xk, xg, max(n, 1), maxAbs64(xg)*sc+1)

						// The same triangle as a band (k = 2) and in packed form.
						const kb = 2
						bl := kb + 2
						band := make([]float64, max(n*bl, 1))
						for i := range band {
							band[i] = math.NaN()
						}
						ap := []float64{}
						for i := 0; i < n; i++ {
							for p := 0; p <= kb; p++ {
								c := i + p
								if lower {
									c = i - kb + p
								}
								if c >= 0 && c < n {
									band[i*bl+p] = ad[i*lda+c]
								}
							}
							lo, hi := i, n
							if lower {
								lo, hi = 0, i+1
							}
							ap = append(ap, ad[i*lda+lo:i*lda+hi]...)
						}
						bsc := float64(max(n, 1)) * maxAbs64(band) * maxAbs64(x)
						xk, xg = append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtbmv(kU, kT, kD, n, kb, band, bl, xk, inc)
						gonumImpl.Dtbmv(ul, ta, dg, n, kb, band, bl, xg, inc)
						cmp64(t, "Dtbmv "+what, xk, xg, max(n, 1), bsc)
						xk, xg = append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtbsv(kU, kT, kD, n, kb, band, bl, xk, inc)
						gonumImpl.Dtbsv(ul, ta, dg, n, kb, band, bl, xg, inc)
						cmp64(t, "Dtbsv "+what, xk, xg, max(n, 1), maxAbs64(xg)*bsc+1)
						xk, xg = append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtpmv(kU, kT, kD, n, ap, xk, inc)
						gonumImpl.Dtpmv(ul, ta, dg, n, ap, xg, inc)
						cmp64(t, "Dtpmv "+what, xk, xg, max(n, 1), sc)
						xk, xg = append([]float64(nil), x...), append([]float64(nil), x...)
						keel.Dtpsv(kU, kT, kD, n, ap, xk, inc)
						gonumImpl.Dtpsv(ul, ta, dg, n, ap, xg, inc)
						cmp64(t, "Dtpsv "+what, xk, xg, max(n, 1), maxAbs64(xg)*sc+1)
					}
				}
				// Symmetric: dense, band and packed, plus the rank updates.
				ad, lda := triDense64(r, n, lower, false)
				band := make([]float64, max(n*4, 1))
				for i := range band {
					band[i] = math.NaN()
				}
				ap := []float64{}
				for i := 0; i < n; i++ {
					for p := 0; p <= 2; p++ {
						c := i + p
						if lower {
							c = i - 2 + p
						}
						if c >= 0 && c < n {
							band[i*4+p] = ad[i*lda+c]
						}
					}
					lo, hi := i, n
					if lower {
						lo, hi = 0, i+1
					}
					ap = append(ap, ad[i*lda+lo:i*lda+hi]...)
				}
				kU := keel.Uplo(ul)
				y := randVec64(r, storage(n, inc))
				sc := 2*float64(max(n, 1))*maxAbs64(ad)*maxAbs64(x) + maxAbs64(y)
				what := fmt.Sprintf("%c n=%d inc=%d", ul, n, inc)
				for _, c := range []struct {
					name string
					k, g func(y []float64)
				}{
					{"Dsymv", func(y []float64) { keel.Dsymv(kU, n, -0.75, ad, lda, x, inc, 0.5, y, inc) },
						func(y []float64) { gonumImpl.Dsymv(ul, n, -0.75, ad, lda, x, inc, 0.5, y, inc) }},
					{"Dsbmv", func(y []float64) { keel.Dsbmv(kU, n, 2, -0.75, band, 4, x, inc, 0.5, y, inc) },
						func(y []float64) { gonumImpl.Dsbmv(ul, n, 2, -0.75, band, 4, x, inc, 0.5, y, inc) }},
					{"Dspmv", func(y []float64) { keel.Dspmv(kU, n, -0.75, ap, x, inc, 0.5, y, inc) },
						func(y []float64) { gonumImpl.Dspmv(ul, n, -0.75, ap, x, inc, 0.5, y, inc) }},
				} {
					yk, yg := append([]float64(nil), y...), append([]float64(nil), y...)
					c.k(yk)
					c.g(yg)
					cmp64(t, c.name+" "+what, yk, yg, n+1, sc)
				}
				rsc := maxAbs64(ad) + 3*maxAbs64(x)*maxAbs64(y) + 1
				for _, c := range []struct {
					name string
					arr  []float64
					k, g func(a []float64)
				}{
					{"Dsyr", ad, func(a []float64) { keel.Dsyr(kU, n, 1.5, x, inc, a, lda) },
						func(a []float64) { gonumImpl.Dsyr(ul, n, 1.5, x, inc, a, lda) }},
					{"Dsyr2", ad, func(a []float64) { keel.Dsyr2(kU, n, 1.5, x, inc, y, inc, a, lda) },
						func(a []float64) { gonumImpl.Dsyr2(ul, n, 1.5, x, inc, y, inc, a, lda) }},
					{"Dspr", ap, func(a []float64) { keel.Dspr(kU, n, 1.5, x, inc, a) },
						func(a []float64) { gonumImpl.Dspr(ul, n, 1.5, x, inc, a) }},
					{"Dspr2", ap, func(a []float64) { keel.Dspr2(kU, n, 1.5, x, inc, y, inc, a) },
						func(a []float64) { gonumImpl.Dspr2(ul, n, 1.5, x, inc, y, inc, a) }},
				} {
					ak, ag := append([]float64(nil), c.arr...), append([]float64(nil), c.arr...)
					c.k(ak)
					c.g(ag)
					cmp64(t, c.name+" "+what, ak, ag, 3, rsc)
				}
			}
			// General band.
			for _, kl := range []int{0, 2} {
				for _, ku := range []int{1, 4} {
					bl := kl + ku + 1 + 1
					band := randVec64(r, max(m*bl, 1))
					for _, ta := range ts {
						rows, inner := m, n
						if ta == blas.Trans {
							rows, inner = n, m
						}
						xx, y := randVec64(r, storage(inner, inc)), randVec64(r, storage(rows, inc))
						yk, yg := append([]float64(nil), y...), append([]float64(nil), y...)
						keel.Dgbmv(keel.Transpose(ta), m, n, kl, ku, -0.75, band, bl, xx, inc, 0.5, yk, inc)
						gonumImpl.Dgbmv(ta, m, n, kl, ku, -0.75, band, bl, xx, inc, 0.5, yg, inc)
						if inner == 0 {
							emptyReduction(t, fmt.Sprintf("Dgbmv %c n=%d", ta, n), yg, y)
							continue
						}
						cmp64(t, fmt.Sprintf("Dgbmv %c n=%d kl=%d ku=%d inc=%d", ta, n, kl, ku, inc), yk, yg, inner+1,
							float64(max(inner, 1))*maxAbs64(band)*maxAbs64(xx)+maxAbs64(y))
					}
				}
			}
		}
	}
}
