// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// The float64 Level-2 suite (#192), held to DESIGN.md §5 rule 1's three checks.
// This file has two of them; Gonum's native float64 is gonumblas'
// TestLevel2AgreesWithGonum64.
//
// Every routine's result is compared on two arms: (1) a definition-order float64
// reference under PeerTolerance64 (Dgemv, Dtrsm, Dsyrk and Dsyr2k at k = 1, Dger),
// and (2) a double-double reference under Tolerance64 (GemvDD, and GemmEntryDD for
// the rank updates). The solves' second arm is a backward error, not a forward
// one: substitution guarantees |b − op(A)·x̂| ≤ γₙ·|op(A)|·|x̂| componentwise, so
// op(A)·x̂ formed in dd must match b within Tolerance64 at that scale. No dd
// triangular solver is needed for that bound.
//
// Symmetric, triangular, band and packed operands are compared through their
// LOGICAL dense matrix: zeros outside the stored triangle or band, ones on a unit
// diagonal, the mirror for symmetric. The arrays the routines are handed carry NaN
// in everything they must not read, so a stray read propagates.

var (
	l2Dims64 = []int{0, 1, 2, 7, 17, 64}
	l2Incs64 = []int{1, -2}
	nan64    = math.NaN()
)

// mat64 returns a rows×cols matrix at a padded stride, NaN in the padding.
func mat64(r *rand.Rand, rows, cols, pad int) ([]float64, int) {
	ld := max(cols+pad, 1)
	m := make([]float64, max(rows*ld, 1))
	for i := range m {
		m[i] = nan64
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m[i*ld+j] = r.NormFloat64()
		}
	}
	return m, ld
}

// triMat64 is triMat at float64: a well-conditioned triangle, its unread part NaN.
func triMat64(r *rand.Rand, n int, lower, unit bool) ([]float64, int) {
	a, lda := mat64(r, n, n, 2)
	for i := 0; i < n; i++ {
		a[i*lda+i] = 4 + r.Float64()
		for j := 0; j < n; j++ {
			if (lower && j > i) || (!lower && j < i) || (unit && i == j) {
				a[i*lda+j] = nan64
			}
		}
	}
	return a, lda
}

// logicalTri64 is the n×n dense (ld n) triangle a holds: zero outside it, one on a
// unit diagonal.
func logicalTri64(a []float64, lda, n int, lower, unit bool) []float64 {
	d := make([]float64, max(n*n, 1))
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			switch {
			case i == j && unit:
				d[i*n+j] = 1
			case (lower && j <= i) || (!lower && j >= i):
				d[i*n+j] = a[i*lda+j]
			}
		}
	}
	return d
}

// logicalSym64 is the full n×n symmetric matrix whose ul triangle a stores.
func logicalSym64(a []float64, lda, n int, lower bool) []float64 {
	d := make([]float64, max(n*n, 1))
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (lower && j <= i) || (!lower && j >= i) {
				d[i*n+j], d[j*n+i] = a[i*lda+j], a[i*lda+j]
			}
		}
	}
	return d
}

// checkMV64 holds y = alpha·op(D)·x + beta·y, for the logical dense D (rows×cols at
// ld), to both arms. got is the routine's y in element order, and xv and yv are x
// and the input y in element order.
func checkMV64(t *testing.T, what string, got []float64, trans bool, m, n int, alpha float64,
	d []float64, ld int, xv []float64, beta float64, yv []float64) {
	t.Helper()
	inner := m
	if !trans {
		inner = n
	}
	w1, s1 := oracle.Dgemv(trans, m, n, alpha, d, ld, xv, 1, beta, yv, 1)
	w2, s2 := oracle.GemvDD(trans, m, n, alpha, d, ld, xv, 1, beta, yv, 1)
	for i := range got {
		check64(t, fmt.Sprintf("%s [%d] ref", what, i), inner+1, got[i], w1[i], s1[i], oracle.PeerTolerance64)
		check64(t, fmt.Sprintf("%s [%d] dd", what, i), inner+1, got[i], w2[i], s2[i], oracle.Tolerance64)
	}
}

// checkSV64 holds a solve's x̂ to both arms: the definition-order Dtrsm, and the
// backward error op(D)·x̂ = b in dd.
func checkSV64(t *testing.T, what string, got []float64, lower, trans, unit bool, n int, a []float64, lda int, d []float64, bv []float64) {
	t.Helper()
	w1, s1 := oracle.Dtrsm(true, lower, trans, unit, n, 1, 1, a, lda, bv, 1)
	r, rs := oracle.GemvDD(trans, n, n, 1, d, n, got, 1, 0, nil, 1)
	for i := range got {
		check64(t, fmt.Sprintf("%s [%d] ref", what, i), n, got[i], w1[i], s1[i], oracle.PeerTolerance64)
		check64(t, fmt.Sprintf("%s [%d] backward", what, i), n+1, r[i], bv[i], rs[i], oracle.Tolerance64)
	}
}

// checkRank64 holds a rank-1 or rank-2 update of the stored triangle to both arms,
// and requires the other triangle to be untouched bit for bit. y == nil is Syr.
func checkRank64(t *testing.T, what string, got, a []float64, lda, n int, lower bool, alpha float64, xv, yv []float64) {
	t.Helper()
	var w, s []float64
	k := 1
	ax, by := make([]float64, n), make([]float64, n) // [x] and [x], or [x y] and [y x]
	if yv == nil {
		w, s = oracle.Dsyrk(lower, false, n, 1, alpha, xv, 1, 1, a, lda)
		copy(ax, xv)
		copy(by, xv)
	} else {
		w, s = oracle.Dsyr2k(lower, false, n, 1, alpha, xv, 1, yv, 1, 1, a, lda)
		k = 2
		ax, by = make([]float64, 2*n), make([]float64, 2*n)
		for i := 0; i < n; i++ {
			ax[2*i], ax[2*i+1] = xv[i], yv[i]
			by[2*i], by[2*i+1] = yv[i], xv[i]
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			e := fmt.Sprintf("%s (%d,%d)", what, i, j)
			if (lower && j > i) || (!lower && j < i) {
				if math.Float64bits(got[i*lda+j]) != math.Float64bits(a[i*lda+j]) {
					t.Errorf("%s: wrote the other triangle", e)
				}
				continue
			}
			check64(t, e+" ref", k+1, got[i*lda+j], w[i*n+j], s[i*n+j], oracle.PeerTolerance64)
			wd, sd := oracle.GemmEntryDD(false, true, i, j, k, alpha, ax, k, by, k, 1, a, lda)
			check64(t, e+" dd", k+1, got[i*lda+j], wd, sd, oracle.Tolerance64)
		}
	}
}

func TestDgemvDger(t *testing.T) {
	r := rand.New(rand.NewSource(81))
	for _, m := range l2Dims64 {
		for _, n := range l2Dims64 {
			a, lda := mat64(r, m, n, 3)
			for _, trans := range []bool{false, true} {
				rows, inner := m, n
				tA := NoTrans
				if trans {
					rows, inner, tA = n, m, Trans
				}
				for _, inc := range l2Incs64 {
					for _, ab := range [][2]float64{{1, 0}, {-0.75, 1}, {2.5, 0.5}} {
						xv, yv := randVals64(r, inner), randVals64(r, rows)
						if ab[1] == 0 {
							for i := range yv {
								yv[i] = nan64 // beta 0 must not read y
							}
						}
						x, y := spread64(xv, inc), spread64(yv, inc)
						got := append([]float64(nil), y...)
						Dgemv(tA, m, n, ab[0], a, lda, x, inc, ab[1], got, inc)
						what := fmt.Sprintf("Dgemv %c m=%d n=%d inc=%d a,b=%v", tA, m, n, inc, ab)
						checkMV64(t, what, gather64(got, rows, inc), trans, m, n, ab[0], a, lda, xv, ab[1], yv)
						checkGaps64(t, what, got, y, rows, inc)
					}
				}
			}
			for _, inc := range l2Incs64 {
				xv, yv := randVals64(r, m), randVals64(r, n)
				got := append([]float64(nil), a...)
				Dger(m, n, -0.75, spread64(xv, inc), inc, spread64(yv, inc), inc, got, lda)
				w1, s1 := oracle.Dger(m, n, -0.75, xv, 1, yv, 1, a, lda)
				for i := 0; i < m; i++ {
					for j := 0; j < n; j++ {
						e := fmt.Sprintf("Dger m=%d n=%d inc=%d (%d,%d)", m, n, inc, i, j)
						check64(t, e+" ref", 2, got[i*lda+j], w1[i*n+j], s1[i*n+j], oracle.PeerTolerance64)
						wd, sd := oracle.GemmEntryDD(false, false, i, j, 1, -0.75, xv, 1, yv, max(n, 1), 1, a, lda)
						check64(t, e+" dd", 2, got[i*lda+j], wd, sd, oracle.Tolerance64)
					}
				}
			}
		}
	}
}

func TestDtrmvDtrsv(t *testing.T) {
	r := rand.New(rand.NewSource(83))
	for _, ul := range []Uplo{Upper, Lower} {
		for _, ta := range []Transpose{NoTrans, Trans} {
			for _, dg := range []Diag{NonUnit, Unit} {
				lower, trans, unit := ul == Lower, ta == Trans, dg == Unit
				for _, n := range l2Dims64 {
					a, lda := triMat64(r, n, lower, unit)
					d := logicalTri64(a, lda, n, lower, unit)
					for _, inc := range l2Incs64 {
						xv := randVals64(r, n)
						x := spread64(xv, inc)
						what := fmt.Sprintf("%c%c%c n=%d inc=%d", ul, ta, dg, n, inc)
						got := append([]float64(nil), x...)
						Dtrmv(ul, ta, dg, n, a, lda, got, inc)
						checkMV64(t, "Dtrmv "+what, gather64(got, n, inc), trans, n, n, 1, d, n, xv, 0, nil)
						checkGaps64(t, "Dtrmv "+what, got, x, n, inc)
						got = append([]float64(nil), x...)
						Dtrsv(ul, ta, dg, n, a, lda, got, inc)
						checkSV64(t, "Dtrsv "+what, gather64(got, n, inc), lower, trans, unit, n, d, n, d, xv)
						checkGaps64(t, "Dtrsv "+what, got, x, n, inc)
					}
				}
			}
		}
	}
}

func TestDsymvDsyrDsyr2(t *testing.T) {
	r := rand.New(rand.NewSource(87))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, n := range l2Dims64 {
			a, lda := triMat64(r, n, lower, false)
			d := logicalSym64(a, lda, n, lower)
			for _, inc := range l2Incs64 {
				xv, yv := randVals64(r, n), randVals64(r, n)
				x, y := spread64(xv, inc), spread64(yv, inc)
				what := fmt.Sprintf("%c n=%d inc=%d", ul, n, inc)
				got := append([]float64(nil), y...)
				Dsymv(ul, n, -0.75, a, lda, x, inc, 0.5, got, inc)
				checkMV64(t, "Dsymv "+what, gather64(got, n, inc), false, n, n, -0.75, d, n, xv, 0.5, yv)
				checkGaps64(t, "Dsymv "+what, got, y, n, inc)

				ga := append([]float64(nil), a...)
				Dsyr(ul, n, 1.5, x, inc, ga, lda)
				checkRank64(t, "Dsyr "+what, ga, a, lda, n, lower, 1.5, xv, nil)
				ga = append([]float64(nil), a...)
				Dsyr2(ul, n, 1.5, x, inc, y, inc, ga, lda)
				checkRank64(t, "Dsyr2 "+what, ga, a, lda, n, lower, 1.5, xv, yv)
			}
		}
	}
}

// bandOf64 returns the band array (stride k+1+2, NaN-padded) that holds the logical
// n×n triangle or symmetric half d with k off-diagonals, in internal/l2's row
// conventions (upper: diagonal first; lower: diagonal at position k).
func bandOf64(d []float64, n, k int, lower bool) ([]float64, int) {
	lda := k + 1 + 2
	b := make([]float64, max(n*lda, 1))
	for i := range b {
		b[i] = nan64
	}
	for i := 0; i < n; i++ {
		for p := 0; p <= k; p++ {
			c := i + p
			if lower {
				c = i - k + p
			}
			if c >= 0 && c < n {
				b[i*lda+p] = d[i*n+c]
			}
		}
	}
	return b, lda
}

// packOf64 returns the packed row-major triangle of the n×n d.
func packOf64(d []float64, n int, lower bool) []float64 {
	ap := []float64{}
	for i := 0; i < n; i++ {
		lo, hi := i, n
		if lower {
			lo, hi = 0, i+1
		}
		ap = append(ap, d[i*n+lo:i*n+hi]...)
	}
	return ap
}

// bandTri64 is a random logical triangular band: diagonal well conditioned, zero
// outside the band.
func bandTri64(r *rand.Rand, n, k int, lower bool) []float64 {
	d := make([]float64, max(n*n, 1))
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			in := (lower && j <= i && i-j <= k) || (!lower && j >= i && j-i <= k)
			switch {
			case !in:
			case i == j:
				d[i*n+j] = 4 + r.Float64()
			default:
				d[i*n+j] = r.NormFloat64()
			}
		}
	}
	return d
}

func TestDBandPacked(t *testing.T) {
	r := rand.New(rand.NewSource(89))
	for _, ul := range []Uplo{Upper, Lower} {
		lower := ul == Lower
		for _, n := range []int{0, 1, 2, 5, 17} {
			for _, k := range []int{0, 1, 3, 20} {
				tri := bandTri64(r, n, k, lower)
				band, lda := bandOf64(tri, n, k, lower)
				full := bandTri64(r, n, n, lower) // a dense triangle, for packed storage
				ap := packOf64(full, n, lower)
				for _, ta := range []Transpose{NoTrans, Trans} {
					for _, dg := range []Diag{NonUnit, Unit} {
						trans, unit := ta == Trans, dg == Unit
						dt, df := tri, full
						if unit { // the routine must not read the diagonal: poison it, and the logical one is 1
							band, ap = append([]float64(nil), band...), append([]float64(nil), ap...)
							dt, df = append([]float64(nil), tri...), append([]float64(nil), full...)
							for i := 0; i < n; i++ {
								dt[i*n+i], df[i*n+i] = 1, 1
								p := 0
								if lower {
									p = k
								}
								band[i*lda+p] = nan64
								o := i*n - i*(i-1)/2
								if lower {
									o = i*(i+1)/2 + i
								}
								ap[o] = nan64
							}
						}
						for _, inc := range []int{1, -2} {
							xv := randVals64(r, n)
							x := spread64(xv, inc)
							what := fmt.Sprintf("%c%c%c n=%d k=%d inc=%d", ul, ta, dg, n, k, inc)
							got := append([]float64(nil), x...)
							Dtbmv(ul, ta, dg, n, k, band, lda, got, inc)
							checkMV64(t, "Dtbmv "+what, gather64(got, n, inc), trans, n, n, 1, dt, max(n, 1), xv, 0, nil)
							got = append([]float64(nil), x...)
							Dtbsv(ul, ta, dg, n, k, band, lda, got, inc)
							checkSV64(t, "Dtbsv "+what, gather64(got, n, inc), lower, trans, unit, n, dt, max(n, 1), dt, xv)
							got = append([]float64(nil), x...)
							Dtpmv(ul, ta, dg, n, ap, got, inc)
							checkMV64(t, "Dtpmv "+what, gather64(got, n, inc), trans, n, n, 1, df, max(n, 1), xv, 0, nil)
							got = append([]float64(nil), x...)
							Dtpsv(ul, ta, dg, n, ap, got, inc)
							checkSV64(t, "Dtpsv "+what, gather64(got, n, inc), lower, trans, unit, n, df, max(n, 1), df, xv)
						}
						if unit {
							band, _ = bandOf64(tri, n, k, lower)
							ap = packOf64(full, n, lower)
						}
					}
				}
				// Symmetric band and packed: the stored half is tri / full, mirrored.
				sb, sbl := bandOf64(tri, n, k, lower)
				symB := logicalSym64(tri, max(n, 1), n, lower)
				symP := logicalSym64(full, max(n, 1), n, lower)
				for _, inc := range []int{1, -2} {
					xv, yv := randVals64(r, n), randVals64(r, n)
					x, y := spread64(xv, inc), spread64(yv, inc)
					what := fmt.Sprintf("%c n=%d k=%d inc=%d", ul, n, k, inc)
					got := append([]float64(nil), y...)
					Dsbmv(ul, n, k, -0.75, sb, sbl, x, inc, 0.5, got, inc)
					checkMV64(t, "Dsbmv "+what, gather64(got, n, inc), false, n, n, -0.75, symB, max(n, 1), xv, 0.5, yv)
					got = append([]float64(nil), y...)
					Dspmv(ul, n, -0.75, ap, x, inc, 0.5, got, inc)
					checkMV64(t, "Dspmv "+what, gather64(got, n, inc), false, n, n, -0.75, symP, max(n, 1), xv, 0.5, yv)
					// Dspr / Dspr2 against the dense rank updates of the same stored triangle.
					for _, two := range []bool{false, true} {
						gp := append([]float64(nil), ap...)
						var yy []float64
						if two {
							Dspr2(ul, n, 1.5, x, inc, y, inc, gp)
							yy = yv
						} else {
							Dspr(ul, n, 1.5, x, inc, gp)
						}
						dense := make([]float64, max(n*n, 1))
						gd := make([]float64, max(n*n, 1))
						o := 0
						for i := 0; i < n; i++ {
							lo, hi := i, n
							if lower {
								lo, hi = 0, i+1
							}
							for j := lo; j < hi; j++ {
								dense[i*n+j], gd[i*n+j] = full[i*n+j], gp[o]
								o++
							}
						}
						name := "Dspr "
						if two {
							name = "Dspr2 "
						}
						checkRank64(t, name+what, gd, dense, max(n, 1), n, lower, 1.5, xv, yy)
					}
				}
			}
		}
	}
}

func TestDgbmv(t *testing.T) {
	r := rand.New(rand.NewSource(91))
	for _, m := range []int{0, 1, 5, 17} {
		for _, n := range []int{0, 1, 5, 17} {
			for _, kl := range []int{0, 1, 3} {
				for _, ku := range []int{0, 2, 20} {
					d := make([]float64, max(m*n, 1))
					for i := 0; i < m; i++ {
						for j := 0; j < n; j++ {
							if j-i <= ku && i-j <= kl {
								d[i*n+j] = r.NormFloat64()
							}
						}
					}
					lda := kl + ku + 1 + 2
					band := make([]float64, max(m*lda, 1))
					for i := range band {
						band[i] = nan64
					}
					for i := 0; i < min(m, n+kl); i++ {
						for c := 0; c < kl+ku+1; c++ {
							if j := i - kl + c; j >= 0 && j < n {
								band[i*lda+c] = d[i*n+j]
							}
						}
					}
					for _, ta := range []Transpose{NoTrans, Trans} {
						trans := ta == Trans
						rows, inner := m, n
						if trans {
							rows, inner = n, m
						}
						for _, inc := range []int{1, -2} {
							xv, yv := randVals64(r, inner), randVals64(r, rows)
							x, y := spread64(xv, inc), spread64(yv, inc)
							got := append([]float64(nil), y...)
							Dgbmv(ta, m, n, kl, ku, -0.75, band, lda, x, inc, 0.5, got, inc)
							what := fmt.Sprintf("Dgbmv %c m=%d n=%d kl=%d ku=%d inc=%d", ta, m, n, kl, ku, inc)
							checkMV64(t, what, gather64(got, rows, inc), trans, m, n, -0.75, d, max(n, 1), xv, 0.5, yv)
							checkGaps64(t, what, got, y, rows, inc)
						}
					}
				}
			}
		}
	}
}

// TestDParity2 is TestDParity for Level 2: each D routine panics exactly as its S
// twin does on bad arguments, with the routine's name the only difference.
func TestDParity2(t *testing.T) {
	a32, x32, y32 := make([]float32, 64), make([]float32, 16), make([]float32, 16)
	a64, x64, y64 := make([]float64, 64), make([]float64, 16), make([]float64, 16)
	type pair struct {
		name string
		s, d func()
	}
	ps := []pair{
		{"gemv bad trans", func() { Sgemv('X', 2, 2, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dgemv('X', 2, 2, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"gemv neg m", func() { Sgemv(NoTrans, -1, 2, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dgemv(NoTrans, -1, 2, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"gemv lda", func() { Sgemv(NoTrans, 2, 3, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dgemv(NoTrans, 2, 3, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"gemv short a", func() { Sgemv(NoTrans, 9, 8, 1, a32, 8, x32, 1, 0, y32, 1) }, func() { Dgemv(NoTrans, 9, 8, 1, a64, 8, x64, 1, 0, y64, 1) }},
		{"ger zero inc", func() { Sger(2, 2, 1, x32, 0, y32, 1, a32, 2) }, func() { Dger(2, 2, 1, x64, 0, y64, 1, a64, 2) }},
		{"trmv bad uplo", func() { Strmv('X', NoTrans, NonUnit, 2, a32, 2, x32, 1) }, func() { Dtrmv('X', NoTrans, NonUnit, 2, a64, 2, x64, 1) }},
		{"trsv bad diag", func() { Strsv(Upper, NoTrans, 'X', 2, a32, 2, x32, 1) }, func() { Dtrsv(Upper, NoTrans, 'X', 2, a64, 2, x64, 1) }},
		{"trsv lda", func() { Strsv(Upper, NoTrans, NonUnit, 3, a32, 2, x32, 1) }, func() { Dtrsv(Upper, NoTrans, NonUnit, 3, a64, 2, x64, 1) }},
		{"symv short y", func() { Ssymv(Upper, 4, 1, a32, 4, x32, 1, 0, y32[:3], 1) }, func() { Dsymv(Upper, 4, 1, a64, 4, x64, 1, 0, y64[:3], 1) }},
		{"syr short x", func() { Ssyr(Upper, 4, 1, x32[:3], 1, a32, 4) }, func() { Dsyr(Upper, 4, 1, x64[:3], 1, a64, 4) }},
		{"syr2 zero incY", func() { Ssyr2(Lower, 2, 1, x32, 1, y32, 0, a32, 2) }, func() { Dsyr2(Lower, 2, 1, x64, 1, y64, 0, a64, 2) }},
		{"gbmv neg band", func() { Sgbmv(NoTrans, 2, 2, -1, 0, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dgbmv(NoTrans, 2, 2, -1, 0, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"gbmv lda", func() { Sgbmv(NoTrans, 2, 2, 1, 1, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dgbmv(NoTrans, 2, 2, 1, 1, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"tbmv k<0", func() { Stbmv(Upper, NoTrans, NonUnit, 2, -1, a32, 2, x32, 1) }, func() { Dtbmv(Upper, NoTrans, NonUnit, 2, -1, a64, 2, x64, 1) }},
		{"tbsv short a", func() { Stbsv(Upper, NoTrans, NonUnit, 40, 1, a32, 2, x32, 1) }, func() { Dtbsv(Upper, NoTrans, NonUnit, 40, 1, a64, 2, x64, 1) }},
		{"sbmv neg n", func() { Ssbmv(Upper, -1, 1, 1, a32, 2, x32, 1, 0, y32, 1) }, func() { Dsbmv(Upper, -1, 1, 1, a64, 2, x64, 1, 0, y64, 1) }},
		{"tpmv short ap", func() { Stpmv(Upper, NoTrans, NonUnit, 20, a32, x32, 1) }, func() { Dtpmv(Upper, NoTrans, NonUnit, 20, a64, x64, 1) }},
		{"tpsv bad trans", func() { Stpsv(Upper, 'X', NonUnit, 2, a32, x32, 1) }, func() { Dtpsv(Upper, 'X', NonUnit, 2, a64, x64, 1) }},
		{"spmv short x", func() { Sspmv(Upper, 4, 1, a32, x32[:2], 1, 0, y32, 1) }, func() { Dspmv(Upper, 4, 1, a64, x64[:2], 1, 0, y64, 1) }},
		{"spr bad uplo", func() { Sspr('X', 2, 1, x32, 1, a32) }, func() { Dspr('X', 2, 1, x64, 1, a64) }},
		{"spr2 short ap", func() { Sspr2(Upper, 20, 1, x32, 1, y32, 1, a32) }, func() { Dspr2(Upper, 20, 1, x64, 1, y64, 1, a64) }},
	}
	for _, p := range ps {
		ms, md := panicMsg(p.s), panicMsg(p.d)
		if ms == "" {
			t.Errorf("%s: the float32 case does not panic, so it tests nothing", p.name)
			continue
		}
		name := ms[len("keel: ") : strings.Index(ms[len("keel: "):], ":")+len("keel: ")]
		if want := strings.Replace(ms, "keel: "+name+":", "keel: D"+name[1:]+":", 1); md != want {
			t.Errorf("%s: float32 panics %q, float64 panics %q", p.name, ms, md)
		}
	}
}
