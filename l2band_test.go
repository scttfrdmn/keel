// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// Each test builds a DENSE matrix, packs it into band or packed storage from the
// format's definition, and holds the routine to the dense float64 reference. Every
// storage slot the format leaves unused is NaN, and so is the dense matrix outside
// the stored triangle and, for Unit, on the diagonal, so a read of anything the
// routine must not touch poisons its result.
//
// The packing here and the indexing in l2band.go could share one misreading of the
// layout. They are checked independently in keel/gonumblas, against gonum, whose
// documentation defines the row-major layouts.

var (
	bandDims = []int{0, 1, 2, 5, 17}
	bandKs   = []int{0, 1, 3, 20}
	bandIncs = []int{1, -2}
)

// triBand returns an n×n triangular band matrix with k off-diagonals: the dense
// form (ld n) and the band array (stride lda = k+1+pad).
func triBand(r *rand.Rand, n, k int, lower, unit bool) (dense, band []float32, lda int) {
	dense = make([]float32, max(n*n, 1))
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			switch {
			case (lower && j > i) || (!lower && j < i) || (unit && i == j):
				dense[i*n+j] = nanPoison
			case (lower && i-j > k) || (!lower && j-i > k):
				dense[i*n+j] = 0 // inside the triangle, outside the band
			case i == j:
				dense[i*n+j] = 4 + float32(r.Float64())
			default:
				dense[i*n+j] = float32(r.NormFloat64())
			}
		}
	}
	lda = k + 1 + 2
	band = make([]float32, max(n*lda, 1))
	for i := range band {
		band[i] = nanPoison
	}
	for i := 0; i < n; i++ {
		for p := 0; p <= k; p++ {
			c := i + p // upper: position p is column i+p
			if lower {
				c = i - k + p // lower: position p is column i-k+p
			}
			if c >= 0 && c < n {
				band[i*lda+p] = dense[i*n+c]
			}
		}
	}
	return dense, band, lda
}

// packed returns an n×n triangle in dense form and packed row-major form.
func packed(r *rand.Rand, n int, lower, unit bool) (dense, ap []float32) {
	dense = make([]float32, max(n*n, 1))
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			switch {
			case (lower && j > i) || (!lower && j < i) || (unit && i == j):
				dense[i*n+j] = nanPoison
			case i == j:
				dense[i*n+j] = 4 + float32(r.Float64())
			default:
				dense[i*n+j] = float32(r.NormFloat64())
			}
		}
	}
	for i := 0; i < n; i++ {
		lo, hi := i, n
		if lower {
			lo, hi = 0, i+1
		}
		for j := lo; j < hi; j++ {
			ap = append(ap, dense[i*n+j])
		}
	}
	if ap == nil {
		ap = []float32{}
	}
	return dense, ap
}

func TestTriBandPackedMVSV(t *testing.T) {
	r := rand.New(rand.NewSource(71))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, ta := range []Transpose{NoTrans, Trans} {
			for _, d := range []Diag{NonUnit, Unit} {
				unit := d == Unit
				for _, n := range bandDims {
					for _, inc := range bandIncs {
						xv := randVals(r, n)
						x := spread(xv, inc)
						check := func(name string, got []float32, want, scale []float64) {
							t.Helper()
							ge := elems(got, n, inc)
							for i := 0; i < n; i++ {
								checkScalar(t, fmt.Sprintf("%s %c%c%c n=%d inc=%d [%d]", name, ul, ta, d, n, inc, i), max(n, 1), ge[i], want[i], scale[i])
							}
							checkGaps(t, name, got, x, n, inc)
						}
						for _, k := range bandKs {
							dense, band, lda := triBand(r, n, k, lower, unit)
							g := append([]float32(nil), x...)
							Stbmv(ul, ta, d, n, k, band, lda, g, inc)
							w, s := oracle.Trmm(true, lower, ta == Trans, unit, n, 1, 1, dense, max(n, 1), xv, 1)
							check(fmt.Sprintf("Stbmv k=%d", k), g, w, s)
							g = append([]float32(nil), x...)
							Stbsv(ul, ta, d, n, k, band, lda, g, inc)
							w, s = oracle.Trsm(true, lower, ta == Trans, unit, n, 1, 1, dense, max(n, 1), xv, 1)
							check(fmt.Sprintf("Stbsv k=%d", k), g, w, s)
						}
						dense, ap := packed(r, n, lower, unit)
						g := append([]float32(nil), x...)
						Stpmv(ul, ta, d, n, ap, g, inc)
						w, s := oracle.Trmm(true, lower, ta == Trans, unit, n, 1, 1, dense, max(n, 1), xv, 1)
						check("Stpmv", g, w, s)
						g = append([]float32(nil), x...)
						Stpsv(ul, ta, d, n, ap, g, inc)
						w, s = oracle.Trsm(true, lower, ta == Trans, unit, n, 1, 1, dense, max(n, 1), xv, 1)
						check("Stpsv", g, w, s)
					}
				}
			}
		}
	}
}

func TestSbmvSpmvSprSpr2(t *testing.T) {
	r := rand.New(rand.NewSource(73))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, n := range bandDims {
			for _, inc := range bandIncs {
				xv, yv0 := randVals(r, n), randVals(r, n)
				x := spread(xv, inc)
				for _, alpha := range []float32{0, 1, -0.75} {
					for _, beta := range []float32{0, 1, 0.5} {
						yv := append([]float32(nil), yv0...)
						if beta == 0 {
							for i := range yv {
								yv[i] = nanPoison
							}
						}
						y := spread(yv, inc)
						ref := yv
						if beta == 0 {
							ref = make([]float32, n)
						}
						mv := func(name string, f func(got []float32), dense []float32) {
							t.Helper()
							got := append([]float32(nil), y...)
							f(got)
							w, s := oracle.Symm(true, lower, n, 1, alpha, dense, max(n, 1), xv, 1, beta, ref, 1)
							ge := elems(got, n, inc)
							for i := 0; i < n; i++ {
								checkScalar(t, fmt.Sprintf("%s %c n=%d inc=%d a=%v b=%v [%d]", name, ul, n, inc, alpha, beta, i), n+1, ge[i], w[i], s[i])
							}
							checkGaps(t, name, got, y, n, inc)
						}
						for _, k := range bandKs {
							dense, band, lda := triBand(r, n, k, lower, false)
							mv(fmt.Sprintf("Ssbmv k=%d", k), func(g []float32) { Ssbmv(ul, n, k, alpha, band, lda, x, inc, beta, g, inc) }, dense)
						}
						dense, ap := packed(r, n, lower, false)
						mv("Sspmv", func(g []float32) { Sspmv(ul, n, alpha, ap, x, inc, beta, g, inc) }, dense)
					}
				}
				// Sspr and Sspr2: the reference is Ssyrk/Ssyr2k with k = 1 on the dense
				// form, read back through the same packed positions.
				y := spread(yv0, inc)
				for _, alpha := range []float32{0, 1, -0.75} {
					dense, ap := packed(r, n, lower, false)
					for _, two := range []bool{false, true} {
						got := append([]float32(nil), ap...)
						var w, s []float64
						name := "Sspr"
						if two {
							name = "Sspr2"
							Sspr2(ul, n, alpha, x, inc, y, inc, got)
							w, s = oracle.Syr2k(lower, false, n, 1, alpha, xv, 1, yv0, 1, 1, dense, max(n, 1))
						} else {
							Sspr(ul, n, alpha, x, inc, got)
							w, s = oracle.Syrk(lower, false, n, 1, alpha, xv, 1, 1, dense, max(n, 1))
						}
						p := 0
						for i := 0; i < n; i++ {
							lo, hi := i, n
							if lower {
								lo, hi = 0, i+1
							}
							for j := lo; j < hi; j++ {
								checkScalar(t, fmt.Sprintf("%s %c n=%d inc=%d a=%v [%d,%d]", name, ul, n, inc, alpha, i, j), 3, got[p], w[i*n+j], s[i*n+j])
								p++
							}
						}
					}
				}
			}
		}
	}
}

func TestSgbmv(t *testing.T) {
	r := rand.New(rand.NewSource(79))
	for _, ta := range []Transpose{NoTrans, Trans} {
		for _, m := range bandDims {
			for _, n := range bandDims {
				for _, kl := range []int{0, 1, 4} {
					for _, ku := range []int{0, 2, 7} {
						dense := make([]float32, max(m*n, 1))
						for i := 0; i < m; i++ {
							for j := 0; j < n; j++ {
								if j-i <= ku && i-j <= kl {
									dense[i*n+j] = float32(r.NormFloat64())
								}
							}
						}
						lda := kl + ku + 1 + 1
						band := make([]float32, max(m*lda, 1))
						for i := range band {
							band[i] = nanPoison
						}
						for i := 0; i < m; i++ {
							for c := 0; c < kl+ku+1; c++ {
								if j := i - kl + c; j >= 0 && j < n {
									band[i*lda+c] = dense[i*n+j]
								}
							}
						}
						rows, inner := m, n
						if ta == Trans {
							rows, inner = n, m
						}
						for _, inc := range bandIncs {
							xv := randVals(r, inner)
							x := spread(xv, inc)
							for _, alpha := range []float32{0, -0.75} {
								for _, beta := range []float32{0, 0.5} {
									yv := randVals(r, rows)
									if beta == 0 {
										for i := range yv {
											yv[i] = nanPoison
										}
									}
									y := spread(yv, inc)
									got := append([]float32(nil), y...)
									Sgbmv(ta, m, n, kl, ku, alpha, band, lda, x, inc, beta, got, inc)
									ref := yv
									if beta == 0 {
										ref = make([]float32, rows)
									}
									w, s := oracle.Gemv(ta == Trans, m, n, alpha, dense, max(n, 1), xv, 1, beta, ref, 1)
									ge := elems(got, rows, inc)
									for i := 0; i < rows; i++ {
										checkScalar(t, fmt.Sprintf("Sgbmv %c m=%d n=%d kl=%d ku=%d inc=%d a=%v b=%v [%d]", ta, m, n, kl, ku, inc, alpha, beta, i),
											max(inner, 1), ge[i], w[i], s[i])
									}
									checkGaps(t, "Sgbmv", got, y, rows, inc)
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestSdsdot(t *testing.T) {
	r := rand.New(rand.NewSource(83))
	for _, n := range []int{0, 1, 7, 300} {
		for _, inc := range []int{1, 3, -2} {
			xv, yv := randVals(r, n), randVals(r, n)
			for i := range xv {
				xv[i] *= float32(math.Exp(8 * r.NormFloat64())) // cancellation float32 would lose
			}
			x, y := spread(xv, inc), spread(yv, inc)
			for _, alpha := range []float32{0, 1.5, -1e6} {
				want := float64(alpha)
				for i := 0; i < n; i++ {
					want += float64(xv[i]) * float64(yv[i])
				}
				// float64 accumulation rounded once: the exact float64 sum rounds to
				// the same float32, which a float32 accumulator would generally miss.
				if got := Sdsdot(n, alpha, x, inc, y, inc); got != float32(want) {
					t.Errorf("Sdsdot n=%d inc=%d alpha=%v: %v, want %v", n, inc, alpha, got, float32(want))
				}
			}
		}
	}
}

func TestBandPackedRejectBadArguments(t *testing.T) {
	a, x := make([]float32, 32), make([]float32, 8)
	mustPanic(t, "Stbmv k<0", func() { Stbmv(Upper, NoTrans, NonUnit, 2, -1, a, 2, x, 1) })
	mustPanic(t, "Stbsv lda<k+1", func() { Stbsv(Upper, NoTrans, NonUnit, 2, 2, a, 2, x, 1) })
	mustPanic(t, "Ssbmv short a", func() { Ssbmv(Upper, 8, 3, 1, a[:20], 4, x, 1, 0, x, 1) })
	mustPanic(t, "Stpmv short ap", func() { Stpmv(Upper, NoTrans, NonUnit, 8, a[:35], x, 1) })
	mustPanic(t, "Sspr zero inc", func() { Sspr(Lower, 2, 1, x, 0, a) })
	mustPanic(t, "Sgbmv lda<kl+ku+1", func() { Sgbmv(NoTrans, 2, 2, 1, 1, 1, a, 2, x, 1, 0, x, 1) })
	mustPanic(t, "Sgbmv negative kl", func() { Sgbmv(NoTrans, 2, 2, -1, 1, 1, a, 3, x, 1, 0, x, 1) })
	mustPanic(t, "Sdsdot short y", func() { Sdsdot(9, 0, x, 1, a[:8], 1) })
}
