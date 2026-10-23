// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/blas/blas32"
	"gonum.org/v1/gonum/blas/gonum"
)

// The arbiter is the float64 oracle, not gonum. Each element's tolerance comes
// from oracle.Tolerance at the oracle's own error scale, and keel and gonum must
// each agree with each other within twice it. An element the oracle does not
// define, such as padding or the other triangle, has scale 0, which makes its
// tolerance the underflow floor: keel and gonum must leave it bit-identical.

var (
	keelImpl  = Implementation{}
	gonumImpl = gonum.Implementation{}
)

var nan32 = float32(math.NaN())

func randVec(r *rand.Rand, n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32(r.NormFloat64())
	}
	return v
}

// storage is the slice length a BLAS vector of n elements at stride inc needs.
func storage(n, inc int) int {
	if n == 0 {
		return 1
	}
	if inc < 0 {
		inc = -inc
	}
	return 1 + (n-1)*inc
}

// sameElem reports whether a keel result and a gonum result agree within twice
// the oracle's tolerance for an n-term reduction at that scale.
func sameElem(k, g float32, n int, scale float64) bool {
	kf, gf := float64(k), float64(g)
	if math.IsNaN(kf) || math.IsNaN(gf) {
		return math.IsNaN(kf) && math.IsNaN(gf)
	}
	if math.IsInf(kf, 0) || math.IsInf(gf, 0) {
		return kf == gf
	}
	return math.Abs(kf-gf) <= 2*oracle.Tolerance(n, scale)
}

// compareStorage compares two whole output buffers. scaleAt maps a storage index
// to the oracle's error scale there, or reports false for storage the routine
// must not touch.
func compareStorage(t *testing.T, what string, k, g []float32, n int, scaleAt func(int) (float64, bool)) {
	t.Helper()
	for i := range k {
		s, ok := scaleAt(i)
		if !ok {
			s = 0
		}
		if !sameElem(k[i], g[i], n, s) {
			t.Errorf("%s: storage[%d]: keel %v, gonum %v (scale %.3g, referenced=%v)", what, i, k[i], g[i], s, ok)
			return
		}
	}
}

// panics reports whether f panicked.
func panics(f func()) (p bool) {
	defer func() { p = recover() != nil }()
	f()
	return false
}

var (
	l1Sizes = []int{0, 1, 3, 17, 64, 257}
	l1Incs  = []int{1, 2, -1, -3}
)

func TestLevel1AgreesWithGonum(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, n := range l1Sizes {
		for _, ix := range l1Incs {
			x := randVec(r, storage(n, ix))
			for _, iy := range l1Incs {
				y := randVec(r, storage(n, iy))
				what := fmt.Sprintf("n=%d incX=%d incY=%d", n, ix, iy)

				_, sc := oracle.Dot(n, x, ix, y, iy)
				if k, g := keelImpl.Sdot(n, x, ix, y, iy), gonumImpl.Sdot(n, x, ix, y, iy); !sameElem(k, g, n, sc) {
					t.Errorf("Sdot %s: keel %v, gonum %v", what, k, g)
				}

				for _, alpha := range []float32{0, 1, -0.75} {
					yk, yg := append([]float32(nil), y...), append([]float32(nil), y...)
					keelImpl.Saxpy(n, alpha, x, ix, yk, iy)
					gonumImpl.Saxpy(n, alpha, x, ix, yg, iy)
					_, s := oracle.Axpy(n, alpha, x, ix, y, iy)
					start := oracle.Index(n, iy)
					compareStorage(t, "Saxpy "+what, yk, yg, 1, func(i int) (float64, bool) {
						if n == 0 || (i-start)%iy != 0 {
							return 0, false
						}
						j := (i - start) / iy
						if j < 0 || j >= n {
							return 0, false
						}
						return s[j], true
					})
				}
			}

			for _, alpha := range []float32{0, 2.5} {
				xk, xg := append([]float32(nil), x...), append([]float32(nil), x...)
				keelImpl.Sscal(n, alpha, xk, ix)
				gonumImpl.Sscal(n, alpha, xg, ix)
				compareStorage(t, fmt.Sprintf("Sscal n=%d inc=%d", n, ix), xk, xg, 1, func(i int) (float64, bool) {
					return math.Abs(float64(alpha) * float64(x[i])), true
				})
			}

			what := fmt.Sprintf("n=%d inc=%d", n, ix)
			if ix > 0 {
				_, sa := oracle.Asum(n, x, ix)
				if k, g := keelImpl.Sasum(n, x, ix), gonumImpl.Sasum(n, x, ix); !sameElem(k, g, n, sa) {
					t.Errorf("Sasum %s: keel %v, gonum %v", what, k, g)
				}
				_, sn := oracle.Nrm2(n, x, ix)
				if k, g := keelImpl.Snrm2(n, x, ix), gonumImpl.Snrm2(n, x, ix); !sameElem(k, g, n, sn) {
					t.Errorf("Snrm2 %s: keel %v, gonum %v", what, k, g)
				}
			} else {
				// Reference BLAS defines the result for a non-positive stride, so a
				// drop-in must return exactly what gonum returns there.
				for name, f := range map[string]func(blas.Float32Level1) float32{
					"Sasum": func(b blas.Float32Level1) float32 { return b.Sasum(n, x, ix) },
					"Snrm2": func(b blas.Float32Level1) float32 { return b.Snrm2(n, x, ix) },
				} {
					if pk, pg := panics(func() { f(keelImpl) }), panics(func() { f(gonumImpl) }); pk != pg {
						t.Errorf("%s %s: keel panics=%v, gonum panics=%v", name, what, pk, pg)
					} else if !pk {
						if k, g := f(keelImpl), f(gonumImpl); k != g {
							t.Errorf("%s %s: keel %v, gonum %v", name, what, k, g)
						}
					}
				}
			}
			pk, pg := panics(func() { keelImpl.Isamax(n, x, ix) }), panics(func() { gonumImpl.Isamax(n, x, ix) })
			if pk != pg {
				t.Errorf("Isamax %s: keel panics=%v, gonum panics=%v", what, pk, pg)
			} else if !pk {
				if k, g := keelImpl.Isamax(n, x, ix), gonumImpl.Isamax(n, x, ix); k != g {
					t.Errorf("Isamax %s: keel %d, gonum %d", what, k, g)
				}
			}
		}
	}
}

var (
	dims   = []int{0, 1, 4, 17}
	alphas = []float32{0, 1, -0.75}
	betas  = []float32{0, 1, 0.5}
	transT = []blas.Transpose{blas.NoTrans, blas.Trans, blas.ConjTrans}
)

// padded returns a rows x cols matrix with leading dimension cols+pad. The padding
// is NaN, so a routine that reads it disagrees with one that does not.
func padded(r *rand.Rand, rows, cols, pad int) ([]float32, int) {
	ld := cols + pad
	if ld < 1 {
		ld = 1
	}
	m := make([]float32, max(rows*ld, 1))
	for i := range m {
		m[i] = nan32
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			m[i*ld+j] = float32(r.NormFloat64())
		}
	}
	return m, ld
}

// betaZeroPoison makes C entirely NaN when beta is 0. BLAS does not read C then,
// so any implementation that does returns NaN where the other does not.
func betaZeroPoison(c []float32, beta float32) {
	if beta == 0 {
		for i := range c {
			c[i] = nan32
		}
	}
}

func TestSgemmAgreesWithGonum(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for _, ta := range transT {
		for _, tb := range transT {
			for _, m := range dims {
				for _, n := range dims {
					for _, k := range dims {
						ar, ac := m, k
						if ta != blas.NoTrans {
							ar, ac = k, m
						}
						br, bc := k, n
						if tb != blas.NoTrans {
							br, bc = n, k
						}
						a, lda := padded(r, ar, ac, 2)
						b, ldb := padded(r, br, bc, 1)
						for _, alpha := range alphas {
							for _, beta := range betas {
								c, ldc := padded(r, m, n, 3)
								betaZeroPoison(c, beta)
								ck, cg := append([]float32(nil), c...), append([]float32(nil), c...)
								keelImpl.Sgemm(ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, ck, ldc)
								gonumImpl.Sgemm(ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, cg, ldc)
								ref := c
								if beta == 0 {
									ref = make([]float32, len(c))
								}
								_, s := oracle.Gemm(ta != blas.NoTrans, tb != blas.NoTrans, m, n, k, alpha, a, lda, b, ldb, beta, ref, ldc)
								compareStorage(t, fmt.Sprintf("Sgemm %c%c m=%d n=%d k=%d a=%v b=%v", ta, tb, m, n, k, alpha, beta),
									ck, cg, k, func(i int) (float64, bool) {
										if row, col := i/ldc, i%ldc; row < m && col < n {
											return s[row*n+col], true
										}
										return 0, false
									})
							}
						}
					}
				}
			}
		}
	}
}

func TestSgemvSgerAgreeWithGonum(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for _, ta := range transT {
		for _, m := range dims {
			for _, n := range dims {
				a, lda := padded(r, m, n, 2)
				for _, inc := range []int{1, -2} {
					rows, inner := oracle.GemvRows(ta != blas.NoTrans, m, n), oracle.GemvInner(ta != blas.NoTrans, m, n)
					x := randVec(r, storage(inner, inc))
					for _, alpha := range alphas {
						for _, beta := range betas {
							y := randVec(r, storage(rows, inc))
							betaZeroPoison(y, beta)
							yk, yg := append([]float32(nil), y...), append([]float32(nil), y...)
							keelImpl.Sgemv(ta, m, n, alpha, a, lda, x, inc, beta, yk, inc)
							gonumImpl.Sgemv(ta, m, n, alpha, a, lda, x, inc, beta, yg, inc)
							ref := y
							if beta == 0 {
								ref = make([]float32, len(y))
							}
							_, s := oracle.Gemv(ta != blas.NoTrans, m, n, alpha, a, lda, x, inc, beta, ref, inc)
							start := oracle.Index(rows, inc)
							compareStorage(t, fmt.Sprintf("Sgemv %c m=%d n=%d inc=%d a=%v b=%v", ta, m, n, inc, alpha, beta),
								yk, yg, inner, func(i int) (float64, bool) {
									if rows == 0 || (i-start)%inc != 0 {
										return 0, false
									}
									j := (i - start) / inc
									if j < 0 || j >= rows {
										return 0, false
									}
									return s[j], true
								})
						}
					}
				}
			}
		}
	}
	for _, m := range dims {
		for _, n := range dims {
			for _, inc := range []int{1, -2} {
				x, y := randVec(r, storage(m, inc)), randVec(r, storage(n, inc))
				a, lda := padded(r, m, n, 2)
				ak, ag := append([]float32(nil), a...), append([]float32(nil), a...)
				keelImpl.Sger(m, n, -0.75, x, inc, y, inc, ak, lda)
				gonumImpl.Sger(m, n, -0.75, x, inc, y, inc, ag, lda)
				_, s := oracle.Ger(m, n, -0.75, x, inc, y, inc, a, lda)
				compareStorage(t, fmt.Sprintf("Sger m=%d n=%d inc=%d", m, n, inc), ak, ag, 1, func(i int) (float64, bool) {
					if row, col := i/lda, i%lda; row < m && col < n {
						return s[row*n+col], true
					}
					return 0, false
				})
			}
		}
	}
}

// triangle poisons the triangle a routine must not read, and the diagonal too
// when unit is true, since a unit-diagonal routine must not read that either.
func triangle(a []float32, n, lda int, lower, unit bool) {
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (lower && j > i) || (!lower && j < i) || (unit && i == j) {
				a[i*lda+j] = nan32
			}
		}
	}
}

func TestSsyrkSsymmAgreeWithGonum(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for _, ul := range []blas.Uplo{blas.Upper, blas.Lower} {
		lower := ul == blas.Lower
		for _, ta := range transT {
			for _, n := range dims {
				for _, k := range dims {
					ar, ac := n, k
					if ta != blas.NoTrans {
						ar, ac = k, n
					}
					a, lda := padded(r, ar, ac, 1)
					for _, alpha := range alphas {
						for _, beta := range betas {
							c, ldc := padded(r, n, n, 2)
							betaZeroPoison(c, beta)
							triangle(c, n, ldc, lower, false) // the other triangle must survive untouched
							ck, cg := append([]float32(nil), c...), append([]float32(nil), c...)
							keelImpl.Ssyrk(ul, ta, n, k, alpha, a, lda, beta, ck, ldc)
							gonumImpl.Ssyrk(ul, ta, n, k, alpha, a, lda, beta, cg, ldc)
							ref := append([]float32(nil), c...)
							if beta == 0 {
								ref = make([]float32, len(c))
							}
							_, s := oracle.Syrk(lower, ta != blas.NoTrans, n, k, alpha, a, lda, beta, ref, ldc)
							// gonum's Lower+Trans path scales C by beta without a beta == 0
							// case, so NaN or Inf already in C survives where BLAS says C is
							// not read. Same class as gonum/gonum#1293 (fixed for [DS]symm in
							// 2020); [DS]syrk was not covered. Here keel is held to the
							// oracle instead of to gonum's answer.
							if lower && ta != blas.NoTrans && beta == 0 {
								want, sc := oracle.Syrk(lower, true, n, k, alpha, a, lda, beta, ref, ldc)
								for i := 0; i < n; i++ {
									for j := 0; j <= i; j++ {
										e := i*n + j
										if !sameElem(ck[i*ldc+j], float32(want[e]), k, sc[e]/2) {
											t.Errorf("Ssyrk %c%c n=%d k=%d a=%v b=0: [%d,%d] keel %v, oracle %v", ul, ta, n, k, alpha, i, j, ck[i*ldc+j], want[e])
										}
									}
								}
								continue
							}
							compareStorage(t, fmt.Sprintf("Ssyrk %c%c n=%d k=%d a=%v b=%v", ul, ta, n, k, alpha, beta),
								ck, cg, k, func(i int) (float64, bool) {
									row, col := i/ldc, i%ldc
									if row < n && col < n && oracle.InTriangle(lower, row, col) {
										return s[row*n+col], true
									}
									return 0, false
								})
						}
					}
				}
			}
		}
		for _, sd := range []blas.Side{blas.Left, blas.Right} {
			left := sd == blas.Left
			for _, m := range dims {
				for _, n := range dims {
					na := n
					if left {
						na = m
					}
					a, lda := padded(r, na, na, 1)
					triangle(a, na, lda, lower, false)
					b, ldb := padded(r, m, n, 2)
					for _, alpha := range alphas {
						for _, beta := range betas {
							c, ldc := padded(r, m, n, 1)
							betaZeroPoison(c, beta)
							ck, cg := append([]float32(nil), c...), append([]float32(nil), c...)
							keelImpl.Ssymm(sd, ul, m, n, alpha, a, lda, b, ldb, beta, ck, ldc)
							gonumImpl.Ssymm(sd, ul, m, n, alpha, a, lda, b, ldb, beta, cg, ldc)
							ref := c
							if beta == 0 {
								ref = make([]float32, len(c))
							}
							_, s := oracle.Symm(left, lower, m, n, alpha, a, lda, b, ldb, beta, ref, ldc)
							compareStorage(t, fmt.Sprintf("Ssymm %c%c m=%d n=%d a=%v b=%v", sd, ul, m, n, alpha, beta),
								ck, cg, na, func(i int) (float64, bool) {
									if row, col := i/ldc, i%ldc; row < m && col < n {
										return s[row*n+col], true
									}
									return 0, false
								})
						}
					}
				}
			}
		}
	}
}

func TestStrsmAgreesWithGonum(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for _, sd := range []blas.Side{blas.Left, blas.Right} {
		for _, ul := range []blas.Uplo{blas.Upper, blas.Lower} {
			for _, ta := range transT {
				for _, dg := range []blas.Diag{blas.NonUnit, blas.Unit} {
					for _, m := range dims {
						for _, n := range dims {
							na := n
							if sd == blas.Left {
								na = m
							}
							a, lda := padded(r, na, na, 1)
							for i := 0; i < na; i++ {
								a[i*lda+i] = 4 + float32(r.Float64()) // well conditioned
							}
							triangle(a, na, lda, ul == blas.Lower, dg == blas.Unit)
							for _, alpha := range alphas {
								b, ldb := padded(r, m, n, 2)
								bk, bg := append([]float32(nil), b...), append([]float32(nil), b...)
								keelImpl.Strsm(sd, ul, ta, dg, m, n, alpha, a, lda, bk, ldb)
								gonumImpl.Strsm(sd, ul, ta, dg, m, n, alpha, a, lda, bg, ldb)
								_, s := oracle.Trsm(sd == blas.Left, ul == blas.Lower, ta != blas.NoTrans, dg == blas.Unit,
									m, n, alpha, a, lda, b, ldb)
								compareStorage(t, fmt.Sprintf("Strsm %c%c%c%c m=%d n=%d a=%v", sd, ul, ta, dg, m, n, alpha),
									bk, bg, max(na, 1), func(i int) (float64, bool) {
										if row, col := i/ldb, i%ldb; row < m && col < n {
											return s[row*n+col], true
										}
										return 0, false
									})
							}
						}
					}
				}
			}
		}
	}
}

// TestPanicsWhereGonumPanics: a drop-in that panics on input gonum accepts
// breaks callers, and one that accepts input gonum rejects hides their bugs.
// Messages may differ; the set of panicking inputs may not.
func TestPanicsWhereGonumPanics(t *testing.T) {
	x := make([]float32, 8)
	a := make([]float32, 16)
	c := make([]float32, 16)
	cases := []struct {
		name string
		call func(b blas.Float32)
	}{
		{"Sdot n<0", func(b blas.Float32) { b.Sdot(-1, x, 1, x, 1) }},
		{"Sdot incX=0", func(b blas.Float32) { b.Sdot(2, x, 0, x, 1) }},
		{"Sdot short x", func(b blas.Float32) { b.Sdot(9, x, 1, x, 1) }},
		{"Saxpy incY=0", func(b blas.Float32) { b.Saxpy(2, 1, x, 1, x, 0) }},
		{"Sscal incX=0", func(b blas.Float32) { b.Sscal(2, 1, x, 0) }},
		{"Sscal incX<0", func(b blas.Float32) { b.Sscal(2, 1, x, -1) }},
		{"Snrm2 incX=0", func(b blas.Float32) { b.Snrm2(2, x, 0) }},
		{"Sasum short x", func(b blas.Float32) { b.Sasum(9, x, 1) }},
		{"Isamax incX=0", func(b blas.Float32) { b.Isamax(2, x, 0) }},
		{"Isamax n=0", func(b blas.Float32) { b.Isamax(0, x, 1) }},
		{"Sgemv bad trans", func(b blas.Float32) { b.Sgemv('X', 2, 2, 1, a, 2, x, 1, 0, x, 1) }},
		{"Sgemv lda<n", func(b blas.Float32) { b.Sgemv(blas.NoTrans, 2, 3, 1, a, 2, x, 1, 0, x, 1) }},
		{"Sger m<0", func(b blas.Float32) { b.Sger(-1, 2, 1, x, 1, x, 1, a, 2) }},
		{"Sgemm lda<k", func(b blas.Float32) { b.Sgemm(blas.NoTrans, blas.NoTrans, 2, 2, 3, 1, a, 2, a, 2, 0, c, 2) }},
		{"Sgemm short c", func(b blas.Float32) { b.Sgemm(blas.NoTrans, blas.NoTrans, 4, 4, 1, 1, a, 1, a, 4, 0, c[:8], 4) }},
		{"Sgemm k=0 is legal", func(b blas.Float32) { b.Sgemm(blas.NoTrans, blas.NoTrans, 2, 2, 0, 1, a, 1, a, 2, 0, c, 2) }},
		{"Ssyrk uplo All", func(b blas.Float32) { b.Ssyrk(blas.All, blas.NoTrans, 2, 2, 1, a, 2, 0, c, 2) }},
		{"Ssymm bad side", func(b blas.Float32) { b.Ssymm('X', blas.Upper, 2, 2, 1, a, 2, a, 2, 0, c, 2) }},
		{"Strsm bad diag", func(b blas.Float32) { b.Strsm(blas.Left, blas.Upper, blas.NoTrans, 'X', 2, 2, 1, a, 2, c, 2) }},
		{"Strsm ldb<n", func(b blas.Float32) { b.Strsm(blas.Left, blas.Upper, blas.NoTrans, blas.NonUnit, 2, 3, 1, a, 2, c, 2) }},
	}
	for _, tc := range cases {
		pk := panics(func() { tc.call(keelImpl) })
		pg := panics(func() { tc.call(gonumImpl) })
		if pk != pg {
			t.Errorf("%s: keel panics=%v, gonum panics=%v", tc.name, pk, pg)
		}
	}
}

// TestBlas32Use is the drop-in claim end to end: register keel through gonum's
// own entry point and call gonum's own high-level wrapper.
func TestBlas32Use(t *testing.T) {
	blas32.Use(Implementation{})
	defer blas32.Use(gonum.Implementation{})
	a := blas32.General{Rows: 2, Cols: 2, Stride: 2, Data: []float32{1, 2, 3, 4}}
	b := blas32.General{Rows: 2, Cols: 2, Stride: 2, Data: []float32{5, 6, 7, 8}}
	c := blas32.General{Rows: 2, Cols: 2, Stride: 2, Data: make([]float32, 4)}
	blas32.Gemm(blas.NoTrans, blas.NoTrans, 1, a, b, 0, c)
	if want := []float32{19, 22, 43, 50}; fmt.Sprint(c.Data) != fmt.Sprint(want) {
		t.Fatalf("blas32.Gemm through keel = %v, want %v", c.Data, want)
	}
	if got := blas32.Implementation(); got != (Implementation{}) {
		t.Fatalf("blas32.Implementation() = %T, want gonumblas.Implementation", got)
	}
}

// TestRoutesToKeel closes the blind spot the agreement tests share: an adapter that
// sent every call to gonum would pass all of them, because gonum agrees with itself.
// For each routine it finds an input where keel and gonum differ BITWISE, then
// requires the adapter to match keel exactly there. keel is deterministic, so
// bit-equality with keel on an input where gonum differs can only mean keel ran.
//
// Not distinguishable here, and stated rather than claimed: Sscal and Isamax are
// exact or single-rounding in both libraries; Saxpy and Sger are one fused
// multiply-add per element in both on arm64, where Go fuses x*y+z itself. No input
// separates these by value, so their routing rests on the four-line pass-throughs
// in gonumblas.go, not on this test.
func TestRoutesToKeel(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	bits := func(v []float32) string { return fmt.Sprintf("%x", v) }
	type probe struct {
		name       string
		viaAdapter func() []float32
		viaKeel    func() []float32
		viaGonum   func() []float32
	}
	const n = 4099
	x, y := randVec(r, n), randVec(r, n)
	for i := range x {
		x[i] *= float32(math.Exp(6 * r.NormFloat64())) // magnitudes over many decades
	}
	a, _ := padded(r, 33, 33, 0)
	for i := 0; i < 33; i++ {
		a[i*33+i] = 4
	}
	b, _ := padded(r, 33, 33, 0)
	run := func(f func(c []float32)) func() []float32 {
		return func() []float32 { c := append([]float32(nil), b...); f(c); return c }
	}
	k := keelImpl
	g := gonumImpl
	probes := []probe{
		{"Sdot", func() []float32 { return []float32{k.Sdot(n, x, 1, y, 1)} }, nil, func() []float32 { return []float32{g.Sdot(n, x, 1, y, 1)} }},
		{"Sasum", func() []float32 { return []float32{k.Sasum(n, x, 1)} }, nil, func() []float32 { return []float32{g.Sasum(n, x, 1)} }},
		{"Snrm2", func() []float32 { return []float32{k.Snrm2(n, x, 1)} }, nil, func() []float32 { return []float32{g.Snrm2(n, x, 1)} }},
		{"Sgemv", run(func(c []float32) { k.Sgemv(blas.NoTrans, 33, 33, 1.5, a, 33, x, 1, 0.5, c[:33], 1) }), nil,
			run(func(c []float32) { g.Sgemv(blas.NoTrans, 33, 33, 1.5, a, 33, x, 1, 0.5, c[:33], 1) })},
		{"Sgemm", run(func(c []float32) { k.Sgemm(blas.NoTrans, blas.Trans, 33, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33) }), nil,
			run(func(c []float32) { g.Sgemm(blas.NoTrans, blas.Trans, 33, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33) })},
		{"Ssyrk", run(func(c []float32) { k.Ssyrk(blas.Upper, blas.NoTrans, 33, 33, 1.5, a, 33, 0.5, c, 33) }), nil,
			run(func(c []float32) { g.Ssyrk(blas.Upper, blas.NoTrans, 33, 33, 1.5, a, 33, 0.5, c, 33) })},
		{"Ssymm", run(func(c []float32) { k.Ssymm(blas.Left, blas.Upper, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33) }), nil,
			run(func(c []float32) { g.Ssymm(blas.Left, blas.Upper, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33) })},
	}
	direct := map[string]func() []float32{
		"Sdot":  func() []float32 { return []float32{keelSdot(n, x, y)} },
		"Sasum": func() []float32 { return []float32{keelSasum(n, x)} },
		"Snrm2": func() []float32 { return []float32{keelSnrm2(n, x)} },
		"Sgemv": run(func(c []float32) { keelSgemv(a, x, c[:33]) }),
		"Sgemm": run(func(c []float32) { keelSgemm(a, b, c) }),
		"Ssyrk": run(func(c []float32) { keelSsyrk(a, c) }),
		"Ssymm": run(func(c []float32) { keelSsymm(a, b, c) }),
	}
	// Strsm needs a triangle past keel's 64-row diagonal block, or the whole solve
	// stays in one block and orders its arithmetic exactly as gonum does.
	ta, _ := padded(r, 130, 130, 0)
	for i := 0; i < 130; i++ {
		ta[i*130+i] = 4
	}
	tb, _ := padded(r, 130, 7, 0)
	trs := func(f func(c []float32)) func() []float32 {
		return func() []float32 { c := append([]float32(nil), tb...); f(c); return c }
	}
	probes = append(probes, probe{"Strsm",
		trs(func(c []float32) {
			k.Strsm(blas.Left, blas.Lower, blas.NoTrans, blas.NonUnit, 130, 7, 1.5, ta, 130, c, 7)
		}), nil,
		trs(func(c []float32) {
			g.Strsm(blas.Left, blas.Lower, blas.NoTrans, blas.NonUnit, 130, 7, 1.5, ta, 130, c, 7)
		})})
	direct["Strsm"] = trs(func(c []float32) { keelStrsm130(ta, c) })
	for _, p := range probes {
		ad, kd, gd := p.viaAdapter(), direct[p.name](), p.viaGonum()
		if bits(kd) == bits(gd) {
			t.Errorf("%s: keel and gonum are bit-identical on this probe, so it cannot tell which one ran; pick a probe where they differ", p.name)
			continue
		}
		if bits(ad) != bits(kd) {
			t.Errorf("%s: the adapter's result is not keel's bit for bit, so the call did not reach keel", p.name)
		}
	}
}
