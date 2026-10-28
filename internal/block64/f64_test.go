// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block64

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// Tests of the generated float64 middle (tools/f64gen, #190). Every Level-3 routine
// runs through internal/block64 with the scalar reference kernel and is held to a
// float64 definition computed here, so what this proves is that the generated copy
// of the nest is correct at float64 -- not that any float64 kernel is fast (#191).
// It is hand-written, not generated: it is the test OF the copy. Ported from the
// generics spike's f64_test.go (696942a), which held the same nest to the same
// bounds when it was instantiated by type parameters instead.
//
// # The bound, and where it comes from
//
// The bound is derived here, from the standard worst-case model rather than
// fitted to anything observed. A length-K dot product evaluated in any order, with
// or without fused multiply-adds, satisfies |fl(Σ xᵢyᵢ) − Σ xᵢyᵢ| ≤ γ_K·Σ|xᵢyᵢ|,
// γ_K = K·u/(1−K·u), u = 2⁻⁵³ (Higham, Accuracy and Stability, §3.1). The nest
// computes alpha·(A·B) + beta·C with alpha folded into the packed A and beta
// applied in a separate pass, which adds at most three roundings per element; the
// test's own definition-order reference has the same error model. Both lie within
// γ_{K+3}·mag of the exact value, mag = |alpha|·Σ|aᵢ||bᵢ| + |beta|·|c|, so they lie
// within 2·γ_{K+3}·mag of each other, and that is the bound — computed per element.
//
// For Trsm the forward error depends on A's conditioning, so it is checked by its
// residual instead: a substitution is componentwise backward stable, (T+ΔT)·x = b
// with |ΔT| ≤ γ_d·|T| (Higham Thm 8.5), and the blocked form adds a rank update
// over the same |T|·|X| term (Higham §13.1). The residual op(A)·X − alpha·B is then
// bounded by γ_{2d+2}·(|op(A)|·|X| + |alpha·B|) including its own evaluation; the
// factor 2 on top is margin, stated as such.
//
// At K ≈ 100 the bound is ~2.4e-14 relative, eight orders of magnitude below
// float32's unit roundoff, so a float32 rounding anywhere in the path — a stray
// float32 buffer, a constant typed wrong — fails it by a wide margin. The positive
// control in TestF64NestGemm checks that claim instead of leaving it to the
// arithmetic: a float32-rounded result must be refused.

const u64 = 0x1p-53

func gamma64(k int) float64 { return float64(k) * u64 / (1 - float64(k)*u64) }

// smallBlocks sets block sizes small enough that a test-sized problem runs several
// jc, pc and ic blocks and a ragged last one in each, with mb diagonal blocks for
// the triangular routines, and restores them when the test ends.
func smallBlocks(t *testing.T, mb int) {
	t.Helper()
	kc, mc, nc, b := KC, MC, NC, MB
	t.Cleanup(func() { KC, MC, NC, MB = kc, mc, nc, b })
	KC, MC, NC, MB = 24, 16, 64, mb
}

// f64Kernels are the scalar reference at the tile docs/f64-design.md §2 predicts
// for AVX-512 (4x16), plus an odd shape so the fringe path runs on both axes.
func f64Kernels() []kern.Kernel64 {
	ks := []kern.Kernel64{kern.ScalarKernel64(4, 16), kern.ScalarKernel64(3, 5),
		// The pre-broadcast A contract (#198) through the generated nest: macro's
		// expansion and panel stride, at the shipped NEON shape and at an odd one.
		kern.ScalarKernelLanes64(5, 8, 2), kern.ScalarKernelLanes64(3, 5, 2)}
	// Every float64 vector kernel this host can run goes through the generated nest
	// too (#191), so the shipped tiles are tested inside the blocking they serve,
	// not only tile by tile against their scalar twins.
	for _, k := range kern.Kernels64() {
		if k.Name != kern.Scalar {
			ks = append(ks, k)
		}
	}
	return ks
}

func randF64(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = 2*r.Float64() - 1
	}
	return v
}

// at reads op(X)[i][j] for X stored with row stride ld.
func at(x []float64, ld int, trans bool, i, j int) float64 {
	if trans {
		return x[j*ld+i]
	}
	return x[i*ld+j]
}

// checkProduct holds got to want within 2·γ_{K+3}·mag, element by element; see the
// file comment. It returns the first violation rather than failing, so that the
// positive control can require one.
func checkProduct(got, want, mag []float64, k int) error {
	g := 2 * gamma64(k+3)
	for i := range got {
		if d := math.Abs(got[i] - want[i]); !(d <= g*mag[i]) {
			return fmt.Errorf("element %d: got %v want %v, |diff| %.3g > bound %.3g", i, got[i], want[i], d, g*mag[i])
		}
	}
	return nil
}

var f64Shapes = []struct{ m, n, k int }{{16, 64, 24}, {50, 200, 73}, {3, 5, 7}, {129, 33, 1}, {37, 71, 100}}

func TestF64NestGemm(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	smallBlocks(t, MB)
	for _, kn := range f64Kernels() {
		for _, s := range f64Shapes {
			for _, tr := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
				name := fmt.Sprintf("%s/m=%d,n=%d,k=%d/tA=%v,tB=%v", kn.ID(), s.m, s.n, s.k, tr[0], tr[1])
				t.Run(name, func(t *testing.T) {
					const alpha, beta = 1.5, -0.75
					lda, ldb := s.k, s.n
					if tr[0] {
						lda = s.m
					}
					if tr[1] {
						ldb = s.k
					}
					a, b, c0 := randF64(r, s.m*s.k), randF64(r, s.k*s.n), randF64(r, s.m*s.n)
					want, mag := make([]float64, s.m*s.n), make([]float64, s.m*s.n)
					for i := 0; i < s.m; i++ {
						for j := 0; j < s.n; j++ {
							var sum, abs float64
							for p := 0; p < s.k; p++ {
								x, y := at(a, lda, tr[0], i, p), at(b, ldb, tr[1], p, j)
								sum += x * y
								abs += math.Abs(x * y)
							}
							want[i*s.n+j] = alpha*sum + beta*c0[i*s.n+j]
							mag[i*s.n+j] = math.Abs(alpha)*abs + math.Abs(beta*c0[i*s.n+j])
						}
					}
					got := append([]float64(nil), c0...)
					gemm(kn, tr[0], tr[1], s.m, s.n, s.k, alpha, a, lda, b, ldb, beta, got, s.n,
						triMask{}, symOperand{}, splitIC)
					if err := checkProduct(got, want, mag, s.k); err != nil {
						t.Fatal(err)
					}
					// Positive control: the same result rounded through float32 must be
					// refused, or the bound could not see a float32 leak in the nest.
					if s.k > 1 {
						r32 := make([]float64, len(got))
						for i, v := range got {
							r32[i] = float64(float32(v))
						}
						if checkProduct(r32, want, mag, s.k) == nil {
							t.Fatal("positive control: a float32-rounded result passed the float64 bound")
						}
					}
				})
			}
		}
	}
}

// TestF64NestParallelIsSerial is invariant I1 for the generated copy: the
// float64 nest at GOMAXPROCS > 1 is bit-identical to itself at 1, and it actually
// partitioned (workers > 1), so agreement cannot come from an arm that never split.
func TestF64NestParallelIsSerial(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	smallBlocks(t, MB)
	kn := f64Kernels()[0]
	m, n, k := 129, 200, 73
	a, b, c0 := randF64(r, m*k), randF64(r, k*n), randF64(r, m*n)
	run := func(procs int) ([]float64, int) {
		defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(procs))
		c := append([]float64(nil), c0...)
		beginCall()
		gemm(kn, false, false, m, n, k, 1.25, a, k, b, n, 0.5, c, n, triMask{}, symOperand{}, splitIC)
		return c, WorkersLastCall()
	}
	serial, _ := run(1)
	par, w := run(4)
	if w < 2 {
		t.Fatalf("GOMAXPROCS=4 ran on %d worker(s): the comparison would prove nothing", w)
	}
	for i := range serial {
		if math.Float64bits(serial[i]) != math.Float64bits(par[i]) {
			t.Fatalf("element %d: serial %v, parallel %v", i, serial[i], par[i])
		}
	}
}

// symDense is the full d×d matrix whose lower (or upper) triangle a stores.
func symDense(a []float64, lda, d int, lower bool) []float64 {
	s := make([]float64, d*d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if (i >= j) == lower || i == j {
				s[i*d+j], s[j*d+i] = a[i*lda+j], a[i*lda+j]
			}
		}
	}
	return s
}

// triDense is op(A) as a dense d×d matrix, from the triangle named by lower, with
// a unit diagonal under unit. Only referenced entries are read.
func triDense(a []float64, lda, d int, lower, trans, unit bool) []float64 {
	t := make([]float64, d*d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			var v float64
			switch {
			case i == j && unit:
				v = 1
			case i == j || (i > j) == lower:
				v = a[i*lda+j]
			}
			if trans {
				t[j*d+i] = v
			} else {
				t[i*d+j] = v
			}
		}
	}
	return t
}

// poisonOff puts NaN in the triangle a routine must not read, and on the diagonal
// under unit, so that reading it shows up in the result.
func poisonOff(a []float64, d int, lower, unit bool) {
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if (i != j && (i > j) != lower) || (i == j && unit) {
				a[i*d+j] = math.NaN()
			}
		}
	}
}

// mulRef is want = alpha·x·y + beta·c (x m×k, y k×n, dense row-major) and its mag.
func mulRef(x, y, c []float64, m, n, k int, alpha, beta float64) (want, mag []float64) {
	want, mag = make([]float64, m*n), make([]float64, m*n)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			var sum, abs float64
			for p := 0; p < k; p++ {
				sum += x[i*k+p] * y[p*n+j]
				abs += math.Abs(x[i*k+p] * y[p*n+j])
			}
			want[i*n+j], mag[i*n+j] = alpha*sum, math.Abs(alpha)*abs
			if c != nil {
				want[i*n+j] += beta * c[i*n+j]
				mag[i*n+j] += math.Abs(beta * c[i*n+j])
			}
		}
	}
	return want, mag
}

func transpose(x []float64, r, c int) []float64 {
	t := make([]float64, r*c)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			t[j*r+i] = x[i*c+j]
		}
	}
	return t
}

// keepTri replaces want/mag outside the written triangle by c0 with a zero bound,
// so the untouched triangle is checked for exact equality.
func keepTri(want, mag, c0 []float64, n int, lower bool) {
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j && (i > j) != lower {
				want[i*n+j], mag[i*n+j] = c0[i*n+j], 0
			}
		}
	}
}

func TestF64NestSyrkSyr2k(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	smallBlocks(t, MB)
	const alpha, beta = 0.75, 1.25
	for _, kn := range f64Kernels() {
		for _, nk := range [][2]int{{37, 53}, {64, 24}, {5, 1}} {
			n, k := nk[0], nk[1]
			for _, lower := range []bool{true, false} {
				for _, trans := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/n=%d,k=%d/lower=%v,trans=%v", kn.ID(), n, k, lower, trans), func(t *testing.T) {
						a, b, c0 := randF64(r, n*k), randF64(r, n*k), randF64(r, n*n)
						lda := k
						opA, opB := a, b // op(A) n×k, dense
						if trans {
							lda = n
							opA, opB = transpose(a, k, n), transpose(b, k, n)
						}
						// Syrk: alpha·op(A)·op(A)ᵀ + beta·C.
						want, mag := mulRef(opA, transpose(opA, n, k), c0, n, n, k, alpha, beta)
						keepTri(want, mag, c0, n, lower)
						got := append([]float64(nil), c0...)
						Syrk(kn, lower, trans, n, k, alpha, a, lda, beta, got, n)
						if err := checkProduct(got, want, mag, k); err != nil {
							t.Fatal("syrk:", err)
						}
						// Syr2k: alpha·(op(A)·op(B)ᵀ + op(B)·op(A)ᵀ) + beta·C, one 2k-term
						// reduction per element, which is how the bound treats it.
						w1, m1 := mulRef(opA, transpose(opB, n, k), c0, n, n, k, alpha, beta)
						w2, m2 := mulRef(opB, transpose(opA, n, k), nil, n, n, k, alpha, 0)
						for i := range w1 {
							w1[i] += w2[i]
							m1[i] += m2[i]
						}
						keepTri(w1, m1, c0, n, lower)
						got = append(got[:0], c0...)
						Syr2k(kn, lower, trans, n, k, alpha, a, lda, b, lda, beta, got, n)
						if err := checkProduct(got, w1, m1, 2*k); err != nil {
							t.Fatal("syr2k:", err)
						}
					})
				}
			}
		}
	}
}

func TestF64NestSymm(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	smallBlocks(t, MB)
	const alpha, beta = -1.5, 0.5
	for _, kn := range f64Kernels() {
		for _, mn := range [][2]int{{37, 71}, {64, 16}, {3, 5}} {
			m, n := mn[0], mn[1]
			for _, left := range []bool{true, false} {
				for _, lower := range []bool{true, false} {
					t.Run(fmt.Sprintf("%s/m=%d,n=%d/left=%v,lower=%v", kn.ID(), m, n, left, lower), func(t *testing.T) {
						d := n
						if left {
							d = m
						}
						a, b, c0 := randF64(r, d*d), randF64(r, m*n), randF64(r, m*n)
						s := symDense(a, d, d, lower)
						poisonOff(a, d, lower, false)
						var want, mag []float64
						if left {
							want, mag = mulRef(s, b, c0, m, n, m, alpha, beta)
						} else {
							want, mag = mulRef(b, s, c0, m, n, n, alpha, beta)
						}
						got := append([]float64(nil), c0...)
						Symm(kn, left, lower, m, n, alpha, a, d, b, n, beta, got, n)
						if err := checkProduct(got, want, mag, d); err != nil {
							t.Fatal(err)
						}
					})
				}
			}
		}
	}
}

func flags4(f func(left, lower, trans, unit bool)) {
	for _, left := range []bool{true, false} {
		for _, lower := range []bool{true, false} {
			for _, trans := range []bool{false, true} {
				for _, unit := range []bool{false, true} {
					f(left, lower, trans, unit)
				}
			}
		}
	}
}

func TestF64NestTrmm(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	smallBlocks(t, 8) // several MB blocks at test size
	const alpha = 1.75
	for _, kn := range f64Kernels() {
		for _, mn := range [][2]int{{37, 29}, {16, 64}, {3, 5}} {
			m, n := mn[0], mn[1]
			flags4(func(left, lower, trans, unit bool) {
				t.Run(fmt.Sprintf("%s/m=%d,n=%d/L=%v,lo=%v,t=%v,u=%v", kn.ID(), m, n, left, lower, trans, unit), func(t *testing.T) {
					d := n
					if left {
						d = m
					}
					a, b0 := randF64(r, d*d), randF64(r, m*n)
					op := triDense(a, d, d, lower, trans, unit)
					poisonOff(a, d, lower, unit)
					var want, mag []float64
					if left {
						want, mag = mulRef(op, b0, nil, m, n, m, alpha, 0)
					} else {
						want, mag = mulRef(b0, op, nil, m, n, n, alpha, 0)
					}
					got := append([]float64(nil), b0...)
					Trmm(kn, left, lower, trans, unit, m, n, alpha, a, d, got, n)
					// Trmm adds the diagonal block's product and the off-diagonal one in
					// two passes: one d-term reduction per element, plus a rounding.
					if err := checkProduct(got, want, mag, d+1); err != nil {
						t.Fatal(err)
					}
				})
			})
		}
	}
}

func TestF64NestTrsm(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12))
	smallBlocks(t, 8)                           // several MB blocks at test size
	ax := func(alpha float64, x, y []float64) { // no alpha == 0 short-circuit, as l1's
		for i, v := range x {
			y[i] += alpha * v
		}
	}
	sd := func(alpha float64, x []float64) { // divide, never a reciprocal multiply
		for i := range x {
			x[i] /= alpha
		}
	}
	const alpha = -0.5
	for _, kn := range f64Kernels() {
		for _, mn := range [][2]int{{37, 29}, {16, 64}, {3, 5}} {
			m, n := mn[0], mn[1]
			flags4(func(left, lower, trans, unit bool) {
				t.Run(fmt.Sprintf("%s/m=%d,n=%d/L=%v,lo=%v,t=%v,u=%v", kn.ID(), m, n, left, lower, trans, unit), func(t *testing.T) {
					d := n
					if left {
						d = m
					}
					a, b0 := randF64(r, d*d), randF64(r, m*n)
					for i := 0; i < d; i++ {
						a[i*d+i] = 2 + r.Float64() // nonsingular; conditioning is irrelevant to a residual check
					}
					op := triDense(a, d, d, lower, trans, unit)
					poisonOff(a, d, lower, unit)
					x := append([]float64(nil), b0...)
					Trsm(kn, ax, sd, left, lower, trans, unit, m, n, alpha, a, d, x, n)
					// Residual op(A)·X − alpha·B (left) or X·op(A) − alpha·B (right).
					var prod, mag []float64
					absOp, absX := make([]float64, len(op)), make([]float64, len(x))
					for i, v := range op {
						absOp[i] = math.Abs(v)
					}
					for i, v := range x {
						absX[i] = math.Abs(v)
					}
					if left {
						prod, _ = mulRef(op, x, nil, m, n, m, 1, 0)
						mag, _ = mulRef(absOp, absX, nil, m, n, m, 1, 0)
					} else {
						prod, _ = mulRef(x, op, nil, m, n, n, 1, 0)
						mag, _ = mulRef(absX, absOp, nil, m, n, n, 1, 0)
					}
					g := 2 * gamma64(2*d+2)
					for i := range prod {
						res := prod[i] - alpha*b0[i]
						if bound := g * (mag[i] + math.Abs(alpha*b0[i])); !(math.Abs(res) <= bound) {
							t.Fatalf("element %d: residual %.3g > bound %.3g", i, res, bound)
						}
					}
				})
			})
		}
	}
}
