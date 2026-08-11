// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/l1"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// Level-1 test suite (DESIGN.md §5).
//
// # What is being tested against what
//
// Every routine is checked against internal/oracle's float64 reference, once
// per backend the machine can actually execute, with the allowed error coming
// only from oracle.Tolerance. There are no epsilons in this file — if a
// comparison needs more slack, the tolerance *model* is what gets argued about
// (DESIGN.md §5.2), not one test's constant.
//
// The per-backend loop reassigns the package-level activeL1 rather than calling
// internal/l1 kernels directly. That is deliberate: it puts the public routine,
// its argument validation, its stride split and its Snrm2 rescue logic inside
// the tested path, so a backend that is correct in isolation but wired up wrong
// still fails. It also means these tests cannot run in parallel, hence no
// t.Parallel anywhere below.
//
// # Shapes
//
// The shape list is built around the boundaries the kernels actually branch on:
// the 16- and 8-element vector widths, the 64- and 32-element unrolled steps,
// and every remainder in between. n=0 and n=1 are in the list because the empty
// and single-element cases are where BLAS conventions and slice arithmetic
// disagree most often.
var shapes = []int{
	0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 23, 24, 31, 32, 33,
	47, 48, 63, 64, 65, 66, 79, 96, 127, 128, 129, 255, 257, 1000, 4096,
}

// Data patterns. Magnitudes are bounded so that no *reference* result
// overflows float32 for any n in shapes — overflow is a real behaviour but it
// belongs in the dedicated tests below, not mixed into the oracle comparison
// where it would be indistinguishable from a kernel bug.
var patterns = []struct {
	name string
	gen  func(r *rand.Rand, n int) []float32
}{
	{"uniform", func(r *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = r.Float32()*2 - 1
		}
		return v
	}},
	{"ramp", func(_ *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = float32(i+1) / 16
		}
		return v
	}},
	// Alternating large values: the sum and the dot product very nearly cancel,
	// so |result| is tiny while the error scale Σ|xᵢyᵢ| is enormous. This is the
	// case that catches a test which substituted |result| for the scale and
	// thereby demanded impossible accuracy — or, worse, passed anything.
	{"cancelling", func(_ *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = 1e6 + float32(i)
			if i%2 == 1 {
				v[i] = -v[i]
			}
		}
		return v
	}},
	// Six orders of magnitude between neighbours: summation order matters most
	// here, so this is where backends disagree with each other by the largest
	// amount that is still legal.
	{"mixed-magnitude", func(r *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			if i%2 == 0 {
				v[i] = 1e6 * (r.Float32() + 0.5)
			} else {
				v[i] = 1e-6 * (r.Float32() + 0.5)
			}
		}
		return v
	}},
	{"sparse", func(r *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			if r.Intn(8) == 0 {
				v[i] = r.Float32() * 4
			}
		}
		return v
	}},
	// Subnormals and values whose squares underflow. Snrm2's rescue path exists
	// for these; Sdot and Sasum must simply not blow up on them.
	{"tiny", func(r *rand.Rand, n int) []float32 {
		v := make([]float32, n)
		for i := range v {
			v[i] = float32(float64(r.Float32()+0.5) * 1e-25)
		}
		return v
	}},
}

// exercised records which backends actually ran, for the gate's coverage
// marker. Written only from forEachBackend, which runs on one goroutine.
var exercised = map[string]bool{}

func forEachBackend(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	saved := activeL1
	t.Cleanup(func() { activeL1 = saved })
	for _, b := range l1.Backends() {
		activeL1 = b
		exercised[b.Name] = true
		t.Run(b.Name, func(t *testing.T) { f(t) })
	}
}

// TestMain prints the markers scripts/gate-p1.sh parses. They are printed after
// the run and unconditionally, so a failing run still reports what it covered —
// the gate needs both facts (pass/fail and coverage) and treats a missing
// marker as a failure in its own right.
func TestMain(m *testing.M) {
	code := m.Run()
	var ran []string
	for _, name := range AvailableL1Backends() {
		if exercised[name] {
			ran = append(ran, name)
		}
	}
	fmt.Println("keel-l1-available:", strings.Join(AvailableL1Backends(), " "))
	fmt.Println("keel-l1-active:", ActiveL1Backend())
	fmt.Println("keel-l1-backends-exercised:", strings.Join(ran, " "))
	os.Exit(code)
}

// ------------------------------------------------------------------ helpers

func newRand() *rand.Rand { return rand.New(rand.NewSource(0x6b65656c)) } // "keel"

// checkScalar compares one float32 result against the float64 reference.
//
// Non-finite results are required to match exactly rather than within a
// tolerance, because "is this a NaN" is not an approximation: a kernel that
// turns a NaN into a large finite number has a masking bug, and a tolerance
// wide enough to hide it would be wide enough to hide anything.
func checkScalar(t *testing.T, what string, n int, got float32, want, scale float64) {
	t.Helper()
	g := float64(got)
	switch {
	case math.IsNaN(want) || math.IsNaN(g):
		if !(math.IsNaN(want) && math.IsNaN(g)) {
			t.Errorf("%s: got %v, oracle %v (NaN mismatch)", what, got, want)
		}
	case math.IsInf(want, 0) || math.IsInf(g, 0):
		if g != want {
			t.Errorf("%s: got %v, oracle %v (infinity mismatch)", what, got, want)
		}
	default:
		if tol := oracle.Tolerance(n, scale); math.Abs(g-want) > tol {
			t.Errorf("%s: got %v, oracle %v, |err| %.3g > tol %.3g (scale %.3g)",
				what, got, want, math.Abs(g-want), tol, scale)
		}
	}
}

// checkVec compares an elementwise result. f(n) is 1 here, not n: axpy and scal
// commit one rounding per element regardless of vector length, so passing n
// would grant a length-4096 vector 4096x the slack its arithmetic can consume.
func checkVecResult(t *testing.T, what string, got []float32, want, scale []float64) {
	t.Helper()
	for j := range want {
		checkScalar(t, fmt.Sprintf("%s[%d]", what, j), 1, got[j], want[j], scale[j])
	}
}

func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic, returned normally", what)
		}
	}()
	f()
}

// spread lays out a vector with the given stride, filling the gaps with a
// poison value. Any kernel that reads or writes a gap gets a wrong answer or
// corrupts the poison, and both are checked.
const poison = float32(-7777)

func spread(v []float32, inc int) []float32 {
	n := len(v)
	if n == 0 {
		return nil
	}
	span := inc
	if span < 0 {
		span = -span
	}
	out := make([]float32, (n-1)*span+1)
	for i := range out {
		out[i] = poison
	}
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[base+j*inc] = v[j]
	}
	return out
}

func gatherStrided(s []float32, n, inc int) []float32 {
	out := make([]float32, n)
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[j] = s[base+j*inc]
	}
	return out
}

// ------------------------------------------------------- oracle comparisons

func TestSdot(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				x, y := p.gen(r, n), p.gen(r, n)
				want, scale := oracle.Dot(n, x, 1, y, 1)
				checkScalar(t, fmt.Sprintf("Sdot(%s,n=%d)", p.name, n),
					n, Sdot(n, x, 1, y, 1), want, scale)

				// x aliased to itself: legal for a dot product, and the shape
				// Snrm2's sum of squares takes.
				want, scale = oracle.Dot(n, x, 1, x, 1)
				checkScalar(t, fmt.Sprintf("Sdot(%s,n=%d,aliased)", p.name, n),
					n, Sdot(n, x, 1, x, 1), want, scale)
			}
		}
	})
}

func TestSaxpy(t *testing.T) {
	alphas := []float32{1, -1, 0.5, 3.25, -1e6, 1e-6}
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				for _, alpha := range alphas {
					x, y := p.gen(r, n), p.gen(r, n)
					want, scale := oracle.Axpy(n, alpha, x, 1, y, 1)
					xc := append([]float32(nil), x...)
					Saxpy(n, alpha, x, 1, y, 1)
					checkVecResult(t, fmt.Sprintf("Saxpy(%s,n=%d,a=%v)", p.name, n, alpha), y, want, scale)
					for j := range xc {
						if x[j] != xc[j] {
							t.Fatalf("Saxpy(%s,n=%d,a=%v): x was modified at %d", p.name, n, alpha, j)
						}
					}
				}
			}
		}
	})
}

func TestSscal(t *testing.T) {
	alphas := []float32{1, -1, 0, 0.5, 3.25, -1e6, 1e-6}
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				for _, alpha := range alphas {
					x := p.gen(r, n)
					want, scale := oracle.Scal(n, alpha, x, 1)
					Sscal(n, alpha, x, 1)
					checkVecResult(t, fmt.Sprintf("Sscal(%s,n=%d,a=%v)", p.name, n, alpha), x, want, scale)
				}
			}
		}
	})
}

func TestSasum(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				x := p.gen(r, n)
				want, scale := oracle.Asum(n, x, 1)
				checkScalar(t, fmt.Sprintf("Sasum(%s,n=%d)", p.name, n),
					n, Sasum(n, x, 1), want, scale)
			}
		}
	})
}

func TestSnrm2(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				x := p.gen(r, n)
				want, scale := oracle.Nrm2(n, x, 1)
				// f(n) = n on the sum of squares, which is what accumulated the
				// error; sqrt is a single correctly-rounded operation and halves
				// the relative error rather than adding to it, so bounding the
				// norm by the sum's bound is conservative.
				checkScalar(t, fmt.Sprintf("Snrm2(%s,n=%d)", p.name, n),
					n, Snrm2(n, x, 1), want, scale)
			}
		}
	})
}

func TestIsamax(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, p := range patterns {
			for _, n := range shapes {
				x := p.gen(r, n)
				want := oracle.Iamax(n, x, 1)
				if got := Isamax(n, x, 1); got != want {
					t.Errorf("Isamax(%s,n=%d): got %d, oracle %d", p.name, n, got, want)
				}
			}
		}
		// Ties, sign symmetry and the sentinel. Isamax is exact, so these are
		// equality checks with no tolerance in sight.
		for _, tc := range []struct {
			name string
			x    []float32
			want int
		}{
			{"empty", nil, -1},
			{"single", []float32{-3}, 0},
			{"tie-first-wins", []float32{2, -2, 2}, 0},
			{"negative-largest", []float32{1, -5, 3}, 1},
			{"minus-zero-vs-zero", []float32{0, -0.0}, 0},
			{"all-zero", []float32{0, 0, 0}, 0},
			{"last", []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}, 16},
		} {
			if got := Isamax(len(tc.x), tc.x, 1); got != tc.want {
				t.Errorf("Isamax(%s): got %d, want %d", tc.name, got, tc.want)
			}
		}
	})
}

// ---------------------------------------------------------------- strides

// TestStrides checks every routine on non-unit strides, including the negative
// strides BLAS defines for Sdot and Saxpy, against the oracle — and checks that
// the elements *between* strided entries are left alone. The gaps are filled
// with a poison value: a kernel that treated a strided vector as contiguous
// would read poison and fail the value check, and one that wrote through the
// gaps would fail the poison check.
func TestStrides(t *testing.T) {
	incs := []int{2, 3, 7, -1, -2, -3}
	forEachBackend(t, func(t *testing.T) {
		r := newRand()
		for _, n := range []int{1, 2, 3, 8, 17, 64, 129} {
			for _, inc := range incs {
				xd, yd := patterns[0].gen(r, n), patterns[0].gen(r, n)
				x, y := spread(xd, inc), spread(yd, inc)

				want, scale := oracle.Dot(n, x, inc, y, inc)
				checkScalar(t, fmt.Sprintf("Sdot(n=%d,inc=%d)", n, inc),
					n, Sdot(n, x, inc, y, inc), want, scale)

				// Mixed strides, including opposite signs.
				x2 := spread(xd, 1)
				want, scale = oracle.Dot(n, x2, 1, y, inc)
				checkScalar(t, fmt.Sprintf("Sdot(n=%d,incX=1,incY=%d)", n, inc),
					n, Sdot(n, x2, 1, y, inc), want, scale)

				wv, sv := oracle.Axpy(n, 2.5, x, inc, y, inc)
				yc := append([]float32(nil), y...)
				Saxpy(n, 2.5, x, inc, y, inc)
				checkVecResult(t, fmt.Sprintf("Saxpy(n=%d,inc=%d)", n, inc), gatherStrided(y, n, inc), wv, sv)
				checkGaps(t, fmt.Sprintf("Saxpy(n=%d,inc=%d)", n, inc), y, yc, n, inc)

				if inc > 0 { // Sscal/Sasum/Snrm2/Isamax are inc>0 only
					xs := spread(xd, inc)
					xc := append([]float32(nil), xs...)
					wv, sv = oracle.Scal(n, -0.75, xs, inc)
					Sscal(n, -0.75, xs, inc)
					checkVecResult(t, fmt.Sprintf("Sscal(n=%d,inc=%d)", n, inc), gatherStrided(xs, n, inc), wv, sv)
					checkGaps(t, fmt.Sprintf("Sscal(n=%d,inc=%d)", n, inc), xs, xc, n, inc)

					want, scale = oracle.Asum(n, x, inc)
					checkScalar(t, fmt.Sprintf("Sasum(n=%d,inc=%d)", n, inc), n, Sasum(n, x, inc), want, scale)

					want, scale = oracle.Nrm2(n, x, inc)
					checkScalar(t, fmt.Sprintf("Snrm2(n=%d,inc=%d)", n, inc), n, Snrm2(n, x, inc), want, scale)

					if got, wantI := Isamax(n, x, inc), oracle.Iamax(n, x, inc); got != wantI {
						t.Errorf("Isamax(n=%d,inc=%d): got %d, oracle %d", n, inc, got, wantI)
					}
				}
			}
		}
	})
}

// checkGaps verifies that only the strided elements changed.
func checkGaps(t *testing.T, what string, got, before []float32, n, inc int) {
	t.Helper()
	touched := make(map[int]bool, n)
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		touched[base+j*inc] = true
	}
	for i := range got {
		if !touched[i] && got[i] != before[i] {
			t.Errorf("%s: wrote through a stride gap at %d (%v -> %v)", what, i, before[i], got[i])
		}
	}
}

// -------------------------------------------------- cross-backend agreement

// TestBackendsAgree is §5.2's differential test at the routine level: the same
// call, every backend, results compared to each other rather than to the
// oracle.
//
// The bound is 2x oracle.Tolerance, and the 2 is a derivation rather than a
// fudge factor: each backend is separately within T of the float64 reference,
// so by the triangle inequality any two of them are within 2T of each other.
// Nothing here is bit-exact and nothing here should be — the whole point of the
// vector kernels is a different summation order. internal/vec is where
// bit-exactness is demanded, because there it is achievable.
func TestBackendsAgree(t *testing.T) {
	backends := l1.Backends()
	if len(backends) < 2 {
		t.Skipf("only one backend available (%s); nothing to differentiate",
			strings.Join(l1.Names(), " "))
	}
	saved := activeL1
	t.Cleanup(func() { activeL1 = saved })

	r := newRand()
	for _, p := range patterns {
		for _, n := range shapes {
			x, y := p.gen(r, n), p.gen(r, n)
			_, dotScale := oracle.Dot(n, x, 1, y, 1)
			_, asumScale := oracle.Asum(n, x, 1)
			_, nrmScale := oracle.Nrm2(n, x, 1)

			type result struct {
				name           string
				dot, asum, nrm float32
				axpy, scal     []float32
				axpyScale      []float64
			}
			var results []result
			for _, b := range backends {
				activeL1 = b
				exercised[b.Name] = true
				ax := append([]float32(nil), y...)
				sc := append([]float32(nil), x...)
				Saxpy(n, 2.5, x, 1, ax, 1)
				Sscal(n, 2.5, sc, 1)
				_, axScale := oracle.Axpy(n, 2.5, x, 1, y, 1)
				results = append(results, result{
					name: b.Name,
					dot:  Sdot(n, x, 1, y, 1),
					asum: Sasum(n, x, 1),
					nrm:  Snrm2(n, x, 1),
					axpy: ax, scal: sc, axpyScale: axScale,
				})
			}
			ref := results[0]
			for _, got := range results[1:] {
				what := fmt.Sprintf("%s vs %s (%s,n=%d)", got.name, ref.name, p.name, n)
				agree(t, what+" Sdot", n, got.dot, ref.dot, dotScale)
				agree(t, what+" Sasum", n, got.asum, ref.asum, asumScale)
				agree(t, what+" Snrm2", n, got.nrm, ref.nrm, nrmScale)
				for j := range ref.axpy {
					agree(t, fmt.Sprintf("%s Saxpy[%d]", what, j), 1, got.axpy[j], ref.axpy[j], ref.axpyScale[j])
				}
				// Scal is one multiply per element in every backend, so unlike
				// the reductions it must agree exactly. Demanding that here is
				// free and would catch a mis-masked tail lane that a tolerance
				// on a small value would swallow.
				for j := range ref.scal {
					if got.scal[j] != ref.scal[j] {
						t.Errorf("%s Sscal[%d]: %v vs %v (expected bit-exact)", what, j, got.scal[j], ref.scal[j])
					}
				}
			}
		}
	}
}

func agree(t *testing.T, what string, n int, a, b float32, scale float64) {
	t.Helper()
	af, bf := float64(a), float64(b)
	if math.IsNaN(af) || math.IsNaN(bf) {
		if !(math.IsNaN(af) && math.IsNaN(bf)) {
			t.Errorf("%s: %v vs %v (NaN mismatch)", what, a, b)
		}
		return
	}
	if tol := 2 * oracle.Tolerance(n, scale); math.Abs(af-bf) > tol {
		t.Errorf("%s: %v vs %v, |diff| %.3g > 2*tol %.3g", what, a, b, math.Abs(af-bf), tol)
	}
}

// --------------------------------------------------------- edge behaviours

// TestEmptyVectors checks that n == 0 is the empty vector and not an error,
// with nil slices — the case a caller hits at a loop boundary and never tests.
func TestEmptyVectors(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		if got := Sdot(0, nil, 1, nil, 1); got != 0 {
			t.Errorf("Sdot(0): got %v, want 0", got)
		}
		if got := Sasum(0, nil, 1); got != 0 {
			t.Errorf("Sasum(0): got %v, want 0", got)
		}
		if got := Snrm2(0, nil, 1); got != 0 {
			t.Errorf("Snrm2(0): got %v, want 0", got)
		}
		if got := Isamax(0, nil, 1); got != -1 {
			t.Errorf("Isamax(0): got %d, want -1", got)
		}
		Saxpy(0, 2, nil, 1, nil, 1)
		Sscal(0, 2, nil, 1)
	})
}

// TestSaxpyAlphaZero pins the reference-BLAS behaviour that alpha == 0 leaves y
// untouched. The NaN in x is the whole test: a kernel that computed
// y += 0*NaN would return a vector of NaNs and look plausible doing it.
func TestSaxpyAlphaZero(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		for _, n := range []int{1, 8, 17, 64, 129} {
			x := make([]float32, n)
			y := make([]float32, n)
			for i := range x {
				x[i] = float32(math.NaN())
				y[i] = float32(i + 1)
			}
			Saxpy(n, 0, x, 1, y, 1)
			for i := range y {
				if y[i] != float32(i+1) {
					t.Errorf("Saxpy(n=%d,alpha=0): y[%d] = %v, want %v", n, i, y[i], float32(i+1))
				}
			}
		}
	})
}

// TestNaNAndInfPropagation. A poisoned element must poison exactly what the
// definition says it poisons: the reductions entirely, and axpy/scal only at
// that index. The index sweep matters because a NaN in the vectorized body, in
// the single-width remainder and in the masked tail take three different code
// paths through every kernel.
func TestNaNAndInfPropagation(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))
	forEachBackend(t, func(t *testing.T) {
		const n = 70 // 64-element unrolled step + 6-element masked tail
		for _, bad := range []float32{nan, inf, -inf} {
			for _, at := range []int{0, 1, 15, 16, 63, 64, 65, 69} {
				x := make([]float32, n)
				y := make([]float32, n)
				for i := range x {
					x[i], y[i] = 1, 1
				}
				x[at] = bad

				if got := Sdot(n, x, 1, y, 1); !isPoisoned(got) {
					t.Errorf("Sdot(%v at %d): got finite %v, want NaN or Inf", bad, at, got)
				}
				if got := Sasum(n, x, 1); !isPoisoned(got) {
					t.Errorf("Sasum(%v at %d): got finite %v", bad, at, got)
				}
				if got := Snrm2(n, x, 1); !isPoisoned(got) {
					t.Errorf("Snrm2(%v at %d): got finite %v", bad, at, got)
				}

				yc := append([]float32(nil), y...)
				Saxpy(n, 2, x, 1, yc, 1)
				for i := range yc {
					if i == at {
						if !isPoisoned(yc[i]) {
							t.Errorf("Saxpy(%v at %d): y[%d] = %v, want poisoned", bad, at, i, yc[i])
						}
					} else if yc[i] != 3 {
						t.Errorf("Saxpy(%v at %d): y[%d] = %v, want 3 (poison leaked)", bad, at, i, yc[i])
					}
				}

				xc := append([]float32(nil), x...)
				Sscal(n, 2, xc, 1)
				for i := range xc {
					if i == at {
						if !isPoisoned(xc[i]) {
							t.Errorf("Sscal(%v at %d): x[%d] = %v, want poisoned", bad, at, i, xc[i])
						}
					} else if xc[i] != 2 {
						t.Errorf("Sscal(%v at %d): x[%d] = %v, want 2 (poison leaked)", bad, at, i, xc[i])
					}
				}
			}
		}

		// Isamax's convention, stated as a test: a leading NaN wins because it
		// seeds the running maximum and no later comparison beats it; a NaN
		// anywhere else loses every comparison and is skipped. Also checked
		// against the oracle, which implements the same chain independently.
		lead := []float32{nan, 1, 2, 3}
		if got := Isamax(len(lead), lead, 1); got != 0 {
			t.Errorf("Isamax(leading NaN): got %d, want 0", got)
		}
		mid := []float32{1, nan, 2, 3}
		if got := Isamax(len(mid), mid, 1); got != 3 {
			t.Errorf("Isamax(interior NaN): got %d, want 3 (NaN ignored)", got)
		}
		withInf := []float32{1, 2, inf, 3}
		if got := Isamax(len(withInf), withInf, 1); got != 2 {
			t.Errorf("Isamax(+Inf): got %d, want 2", got)
		}
		for _, v := range [][]float32{lead, mid, withInf} {
			if got, want := Isamax(len(v), v, 1), oracle.Iamax(len(v), v, 1); got != want {
				t.Errorf("Isamax(%v): got %d, oracle %d", v, got, want)
			}
		}
	})
}

func isPoisoned(v float32) bool {
	f := float64(v)
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// TestSnrm2Rescue exercises the two paths keel.Snrm2's comment promises: an
// input whose sum of squares overflows float32, and one whose squares all
// underflow to zero. Both would be silently wrong without the rescue — 1e-30
// is exactly representable in float32 and its square is not — so this test is
// the only evidence the check after the loop is doing anything.
func TestSnrm2Rescue(t *testing.T) {
	forEachBackend(t, func(t *testing.T) {
		for _, tc := range []struct {
			name string
			n    int
			val  float32
			// wantInf marks the cases where the *norm itself* is not
			// representable in float32, only the ones where the intermediate
			// sum of squares overflows. +Inf is then the correct answer and the
			// rescue's job is to return it rather than something arbitrary.
			wantInf bool
		}{
			{name: "overflow-single", n: 1, val: 1e30},
			{name: "overflow-many", n: 100, val: 1e30},
			{name: "overflow-tail", n: 70, val: 1e30},
			{name: "underflow-single", n: 1, val: 1e-30},
			{name: "underflow-many", n: 100, val: 1e-25},
			{name: "underflow-tail", n: 70, val: 1e-30},
			{name: "all-zero", n: 33, val: 0},
			{name: "norm-itself-overflows", n: 70, val: 3e38, wantInf: true},
		} {
			x := make([]float32, tc.n)
			for i := range x {
				x[i] = tc.val
			}
			want, scale := oracle.Nrm2(tc.n, x, 1)
			got := Snrm2(tc.n, x, 1)
			if tc.wantInf {
				if !math.IsInf(float64(got), 1) {
					t.Errorf("Snrm2(%s): got %v, want +Inf (oracle %v exceeds float32)", tc.name, got, want)
				}
				continue
			}
			// For every other case the reference norm is inside float32's range —
			// that is the point: only the *intermediate* sum over/underflows. So
			// a zero or non-finite result here means the rescue did not fire.
			if tc.val != 0 && (got == 0 || isPoisoned(got)) {
				t.Errorf("Snrm2(%s): got %v, oracle %v — rescue did not fire", tc.name, got, want)
				continue
			}
			checkScalar(t, "Snrm2("+tc.name+")", tc.n, got, want, scale)
		}
	})
}

// TestArgumentPanics. Reference BLAS returns silently for most of these; keel
// panics (see the Level 1 comment in keel.go), and that promise needs a test or
// it is just a comment.
func TestArgumentPanics(t *testing.T) {
	x := make([]float32, 8)
	y := make([]float32, 8)

	mustPanic(t, "Sdot n<0", func() { Sdot(-1, x, 1, y, 1) })
	mustPanic(t, "Sdot incX=0", func() { Sdot(4, x, 0, y, 1) })
	mustPanic(t, "Sdot incY=0", func() { Sdot(4, x, 1, y, 0) })
	mustPanic(t, "Sdot x short", func() { Sdot(9, x, 1, y, 1) })
	mustPanic(t, "Sdot x short for stride", func() { Sdot(8, x, 2, y, 1) })
	mustPanic(t, "Sdot y short for negative stride", func() { Sdot(8, x, 1, y, -2) })

	mustPanic(t, "Saxpy n<0", func() { Saxpy(-1, 1, x, 1, y, 1) })
	mustPanic(t, "Saxpy incX=0", func() { Saxpy(4, 1, x, 0, y, 1) })
	mustPanic(t, "Saxpy y short", func() { Saxpy(9, 1, x, 1, y, 1) })

	// These four are inc > 0 only, per reference BLAS. A negative stride is
	// rejected rather than quietly treated as forwards.
	mustPanic(t, "Sscal incX=-1", func() { Sscal(4, 2, x, -1) })
	mustPanic(t, "Sasum incX=-1", func() { Sasum(4, x, -1) })
	mustPanic(t, "Snrm2 incX=-1", func() { Snrm2(4, x, -1) })
	mustPanic(t, "Isamax incX=-1", func() { Isamax(4, x, -1) })
	mustPanic(t, "Sscal incX=0", func() { Sscal(4, 2, x, 0) })
	mustPanic(t, "Sasum n<0", func() { Sasum(-1, x, 1) })
	mustPanic(t, "Sscal x short", func() { Sscal(9, 2, x, 1) })
	mustPanic(t, "Isamax x short for stride", func() { Isamax(5, x, 2) })

	// n == 0 with a nil slice is not an error even where inc > 0 is required.
	Sscal(0, 2, nil, 1)
	if got := Sasum(0, nil, 1); got != 0 {
		t.Errorf("Sasum(0,nil): got %v", got)
	}
}

// TestDispatchReportsHonestly. The gate's coverage markers are only worth
// anything if these two functions describe the real state, so check them
// against internal/l1 rather than trusting them.
func TestDispatchReportsHonestly(t *testing.T) {
	avail := AvailableL1Backends()
	if len(avail) == 0 {
		t.Fatal("no L1 backends at all; the scalar backend must always be present")
	}
	if avail[len(avail)-1] != l1.Scalar {
		t.Errorf("backend order %v: scalar must be last (the fallback)", avail)
	}
	active := ActiveL1Backend()
	found := false
	for _, n := range avail {
		if n == active {
			found = true
		}
	}
	if !found {
		t.Errorf("active backend %q is not in the available list %v", active, avail)
	}
	if want := os.Getenv(envForce); want != "" && active != want {
		t.Errorf("%s=%q but active backend is %q", envForce, want, active)
	}
}
