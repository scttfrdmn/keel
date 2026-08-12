// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/scttfrdmn/keel/internal/oracle"
)

// Sgemv and Sger against the float64 oracle (DESIGN.md §4/P4).
//
// The lattice is the product of every flag each routine has, including both
// strides, and it runs in full at every size. That is a bigger sweep than P3's
// relative to the work, and the reason is that Level 2 has no arithmetic to speak
// of: at these sizes the routine is dominated by index arithmetic, and index
// arithmetic is what a negative stride, a transposed access and a beta shortcut
// each perturb differently. The lattice is cheap here precisely because the
// routine is O(m·n) rather than O(n³).
//
// Both routines are verified entry by entry at every size, including 500. There
// is no sampling because there is nothing to sample away from: oracle.GemvEntry
// costs one reduction per output element, so an exhaustive comparison of a 500×500
// Sgemv is 500 reductions — the same order as the routine itself, not the
// n³-against-n² blowup that forces P3's sweep to sample.

type gemvCombo struct {
	trans       bool
	alpha, beta float32
	incX, incY  int
}

func (c gemvCombo) String() string {
	return fmt.Sprintf("%s/alpha=%s/beta=%s/incx=%d/incy=%d",
		transStr(c.trans), f32str(c.alpha), f32str(c.beta), c.incX, c.incY)
}

func (c gemvCombo) parts() []string {
	return []string{transStr(c.trans), f32str(c.alpha), f32str(c.beta),
		strconv.Itoa(c.incX), strconv.Itoa(c.incY)}
}

func gemvCombos() []gemvCombo {
	var out []gemvCombo
	for _, tr := range []bool{false, true} {
		for _, al := range p4Alphas {
			for _, be := range p4Betas {
				for _, ix := range p4Incs {
					for _, iy := range p4Incs {
						out = append(out, gemvCombo{tr, al, be, ix, iy})
					}
				}
			}
		}
	}
	return out
}

func TestSgemvSweep(t *testing.T) {
	c := p4("Sgemv")
	c.dim("trans", "N", "T")
	c.dim("alpha", f32set(p4Alphas)...)
	c.dim("beta", f32set(p4Betas)...)
	c.dim("incx", intSet(p4Incs)...)
	c.dim("incy", intSet(p4Incs)...)
	// Recorded before forEachBackend swaps the dispatch state, so the marker
	// describes what a caller gets rather than whichever backend the loop ended on.
	c.l2config("keel.Sgemv->internal/l1.Dot(row) | l1.Axpy(row) when transposed")

	forEachBackend(t, func(t *testing.T) {
		c.backend(ActiveL1Backend())
		for _, sz := range p4Sizes {
			c.size(sz)
			t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
				for _, cb := range gemvCombos() {
					c.combo(cb.parts()...)
					t.Run(cb.String(), func(t *testing.T) {
						gemvCase(t, c, sz, sz, cb)
					})
					if t.Failed() {
						return
					}
				}
			})
		}
	})
}

// gemvCase runs one (size, combination) and verifies every output element.
//
// x and y are laid out with their strides by spread(), which poisons the gaps: a
// routine that walked a stride of 2 as though it were 1 reads poison and fails on
// the value, and one that writes a gap fails on the gap check below. Both halves
// are needed — reading and writing are separate bugs.
func gemvCase(t *testing.T, c *p4cov, m, n int, cb gemvCombo) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	rows, inner := m, n
	if cb.trans {
		rows, inner = n, m
	}
	lda := n
	a := randMatrix(r, m*lda)
	a0 := append([]float32(nil), a...)
	xs := spread(randMatrix(r, inner), cb.incX)
	xs0 := append([]float32(nil), xs...)
	ys := spread(randMatrix(r, rows), cb.incY)
	ys0 := append([]float32(nil), ys...)

	Sgemv(transFlag(cb.trans), m, n, cb.alpha, a, lda, xs, cb.incX, cb.beta, ys, cb.incY)

	want, scale := oracle.Gemv(cb.trans, m, n, cb.alpha, a0, lda, xs0, cb.incX, cb.beta, ys0, cb.incY)
	what := fmt.Sprintf("Sgemv[%s] %s %dx%d", ActiveL1Backend(), cb, m, n)
	base := offset(rows, cb.incY)
	for i := 0; i < rows; i++ {
		// inner+1 roundings: one per term of the reduction plus the beta term.
		checkScalar(t, fmt.Sprintf("%s [%d]", what, i), inner+1,
			ys[base+i*cb.incY], want[i], scale[i])
		if t.Failed() {
			return
		}
	}
	c.verified(rows, exactMode(rows))
	if !equalF32(a, a0) {
		t.Errorf("%s: A was modified", what)
	}
	if !equalF32(xs, xs0) {
		t.Errorf("%s: x was modified", what)
	}
	checkStrideGaps(t, what+" y", ys, ys0, rows, cb.incY)
}

// checkStrideGaps verifies that every element of a strided vector's storage that
// is not one of its n elements still holds what it did before the call.
func checkStrideGaps(t *testing.T, what string, got, orig []float32, n, inc int) {
	t.Helper()
	base := offset(n, inc)
	on := map[int]bool{}
	for j := 0; j < n; j++ {
		on[base+j*inc] = true
	}
	for i := range got {
		if !on[i] && got[i] != orig[i] {
			t.Errorf("%s: gap at %d overwritten: %v, want %v", what, i, got[i], orig[i])
			return
		}
	}
}

// TestSgemvRectangular moves m and n independently. The square sweep cannot tell
// an m-edge bug from an n-edge bug because it never has one without the other;
// this can, and it does it with both transposes, where the two dimensions swap
// roles between the output and the reduction.
func TestSgemvRectangular(t *testing.T) {
	c := p4("Sgemv")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	forEachBackend(t, func(t *testing.T) {
		for _, d := range dims {
			for _, tr := range []bool{false, true} {
				cb := gemvCombo{tr, -0.75, 0.5, 2, -1}
				gemvCase(t, c, d[0], d[1], cb)
				if t.Failed() {
					return
				}
			}
		}
	})
	c.extra("rectangular")
}

// TestSgemvLdPad passes a submatrix view: lda wider than n, with poison in the
// padding. This is how a caller hands keel a block of a larger matrix, and it is
// the case where an off-by-one reads somebody else's data instead of running off
// the end of a slice.
func TestSgemvLdPad(t *testing.T) {
	c := p4("Sgemv")
	const pad = 3
	forEachBackend(t, func(t *testing.T) {
		r := rand.New(rand.NewSource(p4Seed))
		for _, sz := range []int{1, 5, 17, 33, 65} {
			for _, tr := range []bool{false, true} {
				m, n := sz, sz
				lda := n + pad
				a := padded(r, m, n, lda)
				a0 := append([]float32(nil), a...)
				rows, inner := m, n
				if tr {
					rows, inner = n, m
				}
				x := randMatrix(r, inner)
				y := randMatrix(r, rows)
				y0 := append([]float32(nil), y...)
				Sgemv(transFlag(tr), m, n, -0.75, a, lda, x, 1, 0.5, y, 1)
				want, scale := oracle.Gemv(tr, m, n, -0.75, a0, lda, x, 1, 0.5, y0, 1)
				what := fmt.Sprintf("Sgemv[%s] ldpad %s n=%d", ActiveL1Backend(), transStr(tr), sz)
				for i := 0; i < rows; i++ {
					checkScalar(t, fmt.Sprintf("%s [%d]", what, i), inner+1, y[i], want[i], scale[i])
				}
				checkPadding(t, what, a, m, n, lda, nil)
				if t.Failed() {
					return
				}
			}
		}
	})
	c.extra("ldpad")

	c2 := p4("Sger")
	const gpad = 2
	forEachBackend(t, func(t *testing.T) {
		r := rand.New(rand.NewSource(p4Seed))
		for _, sz := range []int{1, 5, 17, 33, 65} {
			m, n := sz, sz
			lda := n + gpad
			a := padded(r, m, n, lda)
			a0 := append([]float32(nil), a...)
			x := randMatrix(r, m)
			y := randMatrix(r, n)
			Sger(m, n, -0.75, x, 1, y, 1, a, lda)
			want, scale := oracle.Ger(m, n, -0.75, x, 1, y, 1, a0, lda)
			what := fmt.Sprintf("Sger[%s] ldpad n=%d", ActiveL1Backend(), sz)
			gerCompare(t, what, m, n, a, lda, want, scale)
			// The padding is inside A here, so the pre-call copy is the reference:
			// a write that happened to store the poison value is still a write.
			checkPadding(t, what, a, m, n, lda, a0)
			if t.Failed() {
				return
			}
		}
	})
	c2.extra("ldpad")
}

// -------------------------------------------------------------------- Sger

type gerCombo struct {
	alpha      float32
	incX, incY int
}

func (c gerCombo) String() string {
	return fmt.Sprintf("alpha=%s/incx=%d/incy=%d", f32str(c.alpha), c.incX, c.incY)
}

func (c gerCombo) parts() []string {
	return []string{f32str(c.alpha), strconv.Itoa(c.incX), strconv.Itoa(c.incY)}
}

func gerCombos() []gerCombo {
	var out []gerCombo
	for _, al := range p4Alphas {
		for _, ix := range p4Incs {
			for _, iy := range p4Incs {
				out = append(out, gerCombo{al, ix, iy})
			}
		}
	}
	return out
}

func TestSgerSweep(t *testing.T) {
	c := p4("Sger")
	c.dim("alpha", f32set(p4Alphas)...)
	c.dim("incx", intSet(p4Incs)...)
	c.dim("incy", intSet(p4Incs)...)
	c.l2config("keel.Sger->internal/l1.Axpy(row)")

	forEachBackend(t, func(t *testing.T) {
		c.backend(ActiveL1Backend())
		for _, sz := range p4Sizes {
			c.size(sz)
			t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
				for _, cb := range gerCombos() {
					c.combo(cb.parts()...)
					t.Run(cb.String(), func(t *testing.T) {
						gerCase(t, c, sz, sz, cb)
					})
					if t.Failed() {
						return
					}
				}
			})
		}
	})
}

func gerCase(t *testing.T, c *p4cov, m, n int, cb gerCombo) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	lda := n
	a := randMatrix(r, m*lda)
	a0 := append([]float32(nil), a...)
	xs := spread(randMatrix(r, m), cb.incX)
	xs0 := append([]float32(nil), xs...)
	ys := spread(randMatrix(r, n), cb.incY)
	ys0 := append([]float32(nil), ys...)

	Sger(m, n, cb.alpha, xs, cb.incX, ys, cb.incY, a, lda)

	want, scale := oracle.Ger(m, n, cb.alpha, xs0, cb.incX, ys0, cb.incY, a0, lda)
	what := fmt.Sprintf("Sger[%s] %s %dx%d", ActiveL1Backend(), cb, m, n)
	gerCompare(t, what, m, n, a, lda, want, scale)
	c.verified(m, exactMode(m*n))
	if !equalF32(xs, xs0) {
		t.Errorf("%s: x was modified", what)
	}
	if !equalF32(ys, ys0) {
		t.Errorf("%s: y was modified", what)
	}
}

// gerCompare checks every entry of the updated A. Two roundings, not m or n: a
// rank-1 update is one product and one add per element regardless of the
// dimensions, so passing a length here would grant slack the arithmetic cannot
// consume (the same argument as checkVecResult's f(n) = 1).
func gerCompare(t *testing.T, what string, m, n int, got []float32, lda int, want, scale []float64) {
	t.Helper()
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), 2,
				got[i*lda+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return
			}
		}
	}
}

// TestSgerRectangular is TestSgemvRectangular's twin; see there.
func TestSgerRectangular(t *testing.T) {
	c := p4("Sger")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	forEachBackend(t, func(t *testing.T) {
		for _, d := range dims {
			gerCase(t, c, d[0], d[1], gerCombo{-0.75, 2, -1})
			if t.Failed() {
				return
			}
		}
	})
	c.extra("rectangular")
}

// ------------------------------------------------------- zero dimensions

// TestSgemvZeroDim pins the two boundary rules, one of which is a deliberate
// deviation from reference SGEMV.
//
// An empty reduction still applies beta, because y = beta·y is the value of the
// expression when the sum is empty — the same rule as Sgemm's k == 0, and see
// Sgemv's doc comment for why reference's early return is not followed. No output
// elements at all (m == 0 untransposed) touches nothing, where the two agree.
func TestSgemvZeroDim(t *testing.T) {
	y := []float32{1, 2}
	// m=2, n=0: the reduction is empty, so y = beta*y.
	Sgemv(NoTrans, 2, 0, -0.75, nil, 1, nil, 1, 2, y, 1)
	if want := []float32{2, 4}; !equalF32(y, want) {
		t.Errorf("n=0 beta=2: y = %v, want %v (the empty sum still scales y)", y, want)
	}
	Sgemv(NoTrans, 2, 0, -0.75, nil, 1, nil, 1, 0, y, 1)
	if want := []float32{0, 0}; !equalF32(y, want) {
		t.Errorf("n=0 beta=0: y = %v, want %v", y, want)
	}
	// Transposed, the empty dimension is m and y is n long.
	y = []float32{1, 2}
	Sgemv(Trans, 0, 2, -0.75, nil, 2, nil, 1, 3, y, 1)
	if want := []float32{3, 6}; !equalF32(y, want) {
		t.Errorf("m=0 trans beta=3: y = %v, want %v", y, want)
	}
	// No output elements: nothing is read or written, and y may be nil.
	full := make([]float32, 16)
	Sgemv(NoTrans, 0, 4, -0.75, nil, 4, full, 1, 3, nil, 1)
	Sgemv(Trans, 4, 0, -0.75, nil, 1, full, 1, 3, nil, 1)
	p4("Sgemv").extra("zerodim")
}

// TestSgerZeroDim: an empty rank-1 update is free and touches nothing. There is no
// beta to apply, so unlike Sgemv there is no boundary rule to argue about.
func TestSgerZeroDim(t *testing.T) {
	full := make([]float32, 16)
	Sger(0, 4, -0.75, nil, 1, full, 1, nil, 4)
	Sger(4, 0, -0.75, full, 1, nil, 1, nil, 1)
	a := []float32{1, 2, 3, 4}
	// alpha == 0 leaves A alone, but x and y are still validated (see
	// TestSgerArgPanic), so they are present and poisoned rather than nil.
	px := []float32{poison, poison}
	Sger(2, 2, 0, px, 1, px, 1, a, 2)
	if want := []float32{1, 2, 3, 4}; !equalF32(a, want) {
		t.Errorf("alpha=0: a = %v, want %v (x and y must not be read)", a, want)
	}
	p4("Sger").extra("zerodim")
}

// ------------------------------------------------------------ argument errors

func TestSgemvArgPanic(t *testing.T) {
	a := make([]float32, 16)
	v := make([]float32, 4)
	cases := []struct {
		what string
		f    func()
	}{
		{"tA not N or T", func() { Sgemv('C', 2, 2, 1, a, 2, v, 1, 1, v, 1) }},
		{"m < 0", func() { Sgemv(NoTrans, -1, 2, 1, a, 2, v, 1, 1, v, 1) }},
		{"n < 0", func() { Sgemv(NoTrans, 2, -1, 1, a, 2, v, 1, 1, v, 1) }},
		{"lda < n", func() { Sgemv(NoTrans, 2, 4, 1, a, 2, v, 1, 1, v, 1) }},
		{"lda == 0", func() { Sgemv(NoTrans, 0, 0, 1, nil, 0, nil, 1, 1, nil, 1) }},
		{"incx == 0", func() { Sgemv(NoTrans, 2, 2, 1, a, 2, v, 0, 1, v, 1) }},
		{"incy == 0", func() { Sgemv(NoTrans, 2, 2, 1, a, 2, v, 1, 1, v, 0) }},
		{"a too short", func() { Sgemv(NoTrans, 4, 4, 1, a[:12], 4, v, 1, 1, v, 1) }},
		{"x too short", func() { Sgemv(NoTrans, 4, 4, 1, a, 4, v[:3], 1, 1, v, 1) }},
		{"y too short", func() { Sgemv(NoTrans, 4, 4, 1, a, 4, v, 1, 1, v[:3], 1) }},
		{"x too short for stride", func() { Sgemv(NoTrans, 4, 4, 1, a, 4, v, 2, 1, v, 1) }},
		{"y too short for negative stride", func() { Sgemv(NoTrans, 4, 4, 1, a, 4, v, 1, 1, v, -2) }},
	}
	for _, tc := range cases {
		mustPanic(t, "Sgemv "+tc.what, tc.f)
	}
	p4("Sgemv").extra("argpanic")
}

func TestSgerArgPanic(t *testing.T) {
	a := make([]float32, 16)
	v := make([]float32, 4)
	cases := []struct {
		what string
		f    func()
	}{
		{"m < 0", func() { Sger(-1, 2, 1, v, 1, v, 1, a, 2) }},
		{"n < 0", func() { Sger(2, -1, 1, v, 1, v, 1, a, 2) }},
		{"lda < n", func() { Sger(2, 4, 1, v, 1, v, 1, a, 2) }},
		{"incx == 0", func() { Sger(2, 2, 1, v, 0, v, 1, a, 2) }},
		{"incy == 0", func() { Sger(2, 2, 1, v, 1, v, 0, a, 2) }},
		{"a too short", func() { Sger(4, 4, 1, v, 1, v, 1, a[:12], 4) }},
		{"x too short", func() { Sger(4, 4, 1, v[:3], 1, v, 1, a, 4) }},
		{"y too short", func() { Sger(4, 4, 1, v, 1, v[:3], 1, a, 4) }},
		// alpha == 0 does not read x or y, but the arguments are still checked:
		// a caller who passed a short vector has a bug today, not next release.
		{"x too short with alpha 0", func() { Sger(4, 4, 0, v[:3], 1, v, 1, a, 4) }},
	}
	for _, tc := range cases {
		mustPanic(t, "Sger "+tc.what, tc.f)
	}
	p4("Sger").extra("argpanic")
}

// ---------------------------------------------------------------- non-finite

// TestSgemvNonFinite pins the two shortcut rules and the one thing keel does that
// reference SGER does not.
//
// alpha == 0 must not read A or x, so a NaN there cannot reach y; beta == 0 must
// not read y, so a NaN already in y cannot survive. And a zero row scale is NOT
// skipped: 0·Inf is NaN and must propagate, which is checked against the oracle
// rather than against a hand-written expectation.
func TestSgemvNonFinite(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))

	forEachBackend(t, func(t *testing.T) {
		a := []float32{nan, nan, nan, nan}
		y := []float32{1, 2}
		Sgemv(NoTrans, 2, 2, 0, a, 2, a, 1, 2, y, 1)
		if want := []float32{2, 4}; !equalF32(y, want) {
			t.Errorf("alpha=0 with NaN in A and x: y = %v, want %v (neither may be read)", y, want)
		}

		a = []float32{1, 0, 0, 1}
		x := []float32{1, 2}
		y = []float32{nan, nan}
		Sgemv(NoTrans, 2, 2, 1, a, 2, x, 1, 0, y, 1)
		if want := []float32{1, 2}; !equalF32(y, want) {
			t.Errorf("beta=0 with NaN in y: y = %v, want %v (y must not be read)", y, want)
		}

		// An infinity in the data, both transposes, checked against the oracle so
		// that 0·Inf = NaN is required rather than assumed.
		for _, tr := range []bool{false, true} {
			r := rand.New(rand.NewSource(p4Seed))
			const m, n = 5, 5
			av := randMatrix(r, m*n)
			av[0] = inf
			xv := randMatrix(r, n)
			xv[1] = 0
			yv := randMatrix(r, m)
			y0 := append([]float32(nil), yv...)
			a0 := append([]float32(nil), av...)
			Sgemv(transFlag(tr), m, n, 1, av, n, xv, 1, 1, yv, 1)
			want, scale := oracle.Gemv(tr, m, n, 1, a0, n, xv, 1, 1, y0, 1)
			for i := 0; i < m; i++ {
				checkScalar(t, fmt.Sprintf("Sgemv inf %s [%d]", transStr(tr), i), n+1,
					yv[i], want[i], scale[i])
			}
		}
	})
	p4("Sgemv").extra("nonfinite")
}

func TestSgerNonFinite(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))
	forEachBackend(t, func(t *testing.T) {
		// alpha == 0 leaves A alone without reading x or y.
		a := []float32{1, 2, 3, 4}
		bad := []float32{nan, inf}
		Sger(2, 2, 0, bad, 1, bad, 1, a, 2)
		if want := []float32{1, 2, 3, 4}; !equalF32(a, want) {
			t.Errorf("alpha=0 with NaN in x: a = %v, want %v", a, want)
		}
		// A zero in x meeting an infinity in y must give NaN, not a skipped row.
		r := rand.New(rand.NewSource(p4Seed))
		const m, n = 5, 5
		av := randMatrix(r, m*n)
		a0 := append([]float32(nil), av...)
		xv := randMatrix(r, m)
		xv[2] = 0
		yv := randMatrix(r, n)
		yv[3] = inf
		Sger(m, n, 1, xv, 1, yv, 1, av, n)
		want, scale := oracle.Ger(m, n, 1, xv, 1, yv, 1, a0, n)
		gerCompare(t, "Sger inf", m, n, av, n, want, scale)
	})
	p4("Sger").extra("nonfinite")
}
