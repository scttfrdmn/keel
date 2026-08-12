// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// Ssyrk, Ssymm and Strsm against the float64 oracle (DESIGN.md §4/P4).
//
// # What these tests are for, given that Sgemm is already tested
//
// The three routines are derivations on Sgemm's loop nest (internal/block/tri.go),
// so the arithmetic underneath them is P3's and is already held to the oracle over
// a bigger sweep than this one. What is new in P4 is not arithmetic, it is
// *addressing under a promise*: which triangle is read, which is written, which
// diagonal is referenced, and which half of a tile that straddles the diagonal
// reaches C. Every one of those is a promise about memory the routine must not
// touch, and a wrong answer is the least of the ways to break it — the worse one
// is a correct answer computed by reading a caller's other matrix.
//
// So the sweeps below poison what must not be read (NaN in A's unreferenced
// triangle, NaN on a unit diagonal) and copy what must not be written (C's
// unreferenced triangle, compared bit-for-bit afterwards). A NaN that is read
// propagates into the result and fails the value comparison; a write that must
// not have happened fails the bit comparison. Neither can be absorbed by a
// tolerance, which is the point.

// l3runner is one way to reach a derived routine: the public entry point, or a
// specific microkernel driven through the blocking nest directly. Same rationale
// as gemmRunner's, and the same differential value — on an AVX-512 host the sweep
// runs on both shipped vector shapes and both scalar references, so a masking bug
// that happens to be invisible at one tile shape still shows up.
type l3runner struct {
	name    string
	backend string
	syrk    func(lower, trans bool, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int)
	symm    func(left, lower bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int)
	trsm    func(left, lower, trans, unit bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int)
}

func l3Runners() []l3runner {
	rs := []l3runner{{
		name:    "public",
		backend: ActiveKernBackend(),
		syrk: func(lower, trans bool, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int) {
			Ssyrk(uploFlag(lower), transFlag(trans), n, k, alpha, a, lda, beta, c, ldc)
		},
		symm: func(left, lower bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
			Ssymm(sideFlag(left), uploFlag(lower), m, n, alpha, a, lda, b, ldb, beta, c, ldc)
		},
		trsm: func(left, lower, trans, unit bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
			Strsm(sideFlag(left), uploFlag(lower), transFlag(trans), diagFlag(unit), m, n, alpha, a, lda, b, ldb)
		},
	}}
	for _, kn := range kern.Kernels() {
		kn := kn
		rs = append(rs, l3runner{
			name:    kn.ID(),
			backend: kn.Name,
			syrk: func(lower, trans bool, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int) {
				block.Syrk(kn, lower, trans, n, k, alpha, a, lda, beta, c, ldc)
			},
			symm: func(left, lower bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
				block.Symm(kn, left, lower, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
			},
			trsm: func(left, lower, trans, unit bool, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
				block.Trsm(kn, left, lower, trans, unit, m, n, alpha, a, lda, b, ldb)
			},
		})
	}
	return rs
}

// ---------------------------------------------------------------------- Ssyrk

type syrkCombo struct {
	lower, trans bool
	alpha, beta  float32
}

func (c syrkCombo) String() string {
	return fmt.Sprintf("uplo=%s/%s/alpha=%s/beta=%s",
		uploStr(c.lower), transStr(c.trans), f32str(c.alpha), f32str(c.beta))
}

func (c syrkCombo) parts() []string {
	return []string{uploStr(c.lower), transStr(c.trans), f32str(c.alpha), f32str(c.beta)}
}

func syrkCombos() []syrkCombo {
	var out []syrkCombo
	for _, lo := range []bool{false, true} {
		for _, tr := range []bool{false, true} {
			for _, al := range p4Alphas {
				for _, be := range p4Betas {
					out = append(out, syrkCombo{lo, tr, al, be})
				}
			}
		}
	}
	return out
}

// syrkCorners is the flag lattice with the shortcuts removed: the four
// (uplo, trans) combinations at a general alpha and beta. Sizes above p4ExactMax
// run one of these, chosen by the runner's index, for the reason P3's sweep gives
// for its own rotation — the large sizes are here to exercise the loop nest, and
// running 36 combinations of it costs a gate run and buys nothing. Rotating over
// the runner index rather than fixing one combination means all four corners are
// covered at the large size, just not all four per kernel.
func syrkCorners() []syrkCombo {
	var out []syrkCombo
	for _, lo := range []bool{false, true} {
		for _, tr := range []bool{false, true} {
			out = append(out, syrkCombo{lo, tr, -0.75, 0.5})
		}
	}
	return out
}

func TestSsyrkSweep(t *testing.T) {
	c := p4("Ssyrk")
	c.dim("uplo", "U", "L")
	c.dim("trans", "N", "T")
	c.dim("alpha", f32set(p4Alphas)...)
	c.dim("beta", f32set(p4Betas)...)
	c.l3config("keel.Ssyrk->internal/block.Syrk->block.gemm(triMask)")

	for ri, rn := range l3Runners() {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := syrkCombos()
				if sz > p4ExactMax {
					cs := syrkCorners()
					combos = []syrkCombo{cs[ri%len(cs)]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							syrkCase(t, c, rn, sz, sz, cb, sz)
						})
						if t.Failed() {
							return
						}
					}
				})
			}
		})
	}
	c.extra("untouched-triangle")
}

// syrkCase runs one (runner, size, combination) and checks both halves of SSYRK's
// contract: the referenced triangle against the oracle, and the other one against
// the bytes that were there before the call.
func syrkCase(t *testing.T, c *p4cov, rn l3runner, n, k int, cb syrkCombo, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(n*1000+k)))
	ra, ca := n, k
	if cb.trans {
		ra, ca = k, n
	}
	lda, ldc := ca, n
	a := randMatrix(r, ra*lda)
	a0 := append([]float32(nil), a...)
	cm := randMatrix(r, n*ldc)
	c0 := append([]float32(nil), cm...)

	rn.syrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, cm, ldc)

	what := fmt.Sprintf("Ssyrk[%s] %s n=%d k=%d", rn.name, cb, n, k)
	if !equalF32(a, a0) {
		t.Errorf("%s: A was modified", what)
		return
	}
	// The triangle SSYRK does not name must come back bit-identical. Not "close":
	// the routine promised not to write it at all.
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if oracle.InTriangle(cb.lower, i, j) {
				continue
			}
			if cm[i*ldc+j] != c0[i*ldc+j] {
				t.Errorf("%s: wrote the unreferenced triangle at [%d,%d]: %v, was %v",
					what, i, j, cm[i*ldc+j], c0[i*ldc+j])
				return
			}
		}
	}
	if sz <= p4ExactMax {
		want, scale := oracle.Syrk(cb.lower, cb.trans, n, k, cb.alpha, a0, lda, cb.beta, c0, ldc)
		entries := 0
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !oracle.InTriangle(cb.lower, i, j) {
					continue
				}
				entries++
				checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), k+1,
					cm[i*ldc+j], want[i*n+j], scale[i*n+j])
				if t.Failed() {
					return
				}
			}
		}
		c.verified(sz, exactMode(entries))
		return
	}
	// Above the exact bound the oracle costs more than the routine, so a seeded
	// sample of entries in the triangle, plus its three corners: the diagonal ends
	// and the far corner are where a mask off-by-one lands and where a uniform
	// sample of 256 out of n²/2 is unlikely to look.
	pts := [][2]int{{0, 0}, {n - 1, n - 1}}
	if cb.lower {
		pts = append(pts, [2]int{n - 1, 0})
	} else {
		pts = append(pts, [2]int{0, n - 1})
	}
	rs := rand.New(rand.NewSource(p4Seed + int64(sz)))
	for len(pts) < p4SampleN {
		i, j := rs.Intn(n), rs.Intn(n)
		if !oracle.InTriangle(cb.lower, i, j) {
			i, j = j, i
		}
		pts = append(pts, [2]int{i, j})
	}
	for _, p := range pts {
		want, scale := oracle.SyrkEntry(cb.trans, p[0], p[1], k, cb.alpha, a0, lda, cb.beta, c0, ldc)
		checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, p[0], p[1]), k+1,
			cm[p[0]*ldc+p[1]], want, scale)
		if t.Failed() {
			return
		}
	}
	c.verified(sz, sampledMode(len(pts), p4Seed))
}

// TestSsyrkShapes moves k independently of n, around both tile bounds and the
// kernels' unroll. The square sweep cannot distinguish a depth-loop bug from a
// mask bug because it never has one dimension without the other.
func TestSsyrkShapes(t *testing.T) {
	c := p4("Ssyrk")
	ns := []int{1, 2, 3, 5, 8, 31, 32, 33, 65}
	ks := []int{1, 2, 3, 4, 7, 8, 33}
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			for _, n := range ns {
				for _, k := range ks {
					for _, cb := range syrkCorners() {
						syrkCase(t, c, rn, n, k, cb, n)
						if t.Failed() {
							return
						}
					}
				}
			}
		})
	}
	c.extra("rectangular")
}

// ---------------------------------------------------------------------- Ssymm

type symmCombo struct {
	left, lower bool
	alpha, beta float32
}

func (c symmCombo) String() string {
	return fmt.Sprintf("side=%s/uplo=%s/alpha=%s/beta=%s",
		sideStr(c.left), uploStr(c.lower), f32str(c.alpha), f32str(c.beta))
}

func (c symmCombo) parts() []string {
	return []string{sideStr(c.left), uploStr(c.lower), f32str(c.alpha), f32str(c.beta)}
}

func symmCombos() []symmCombo {
	var out []symmCombo
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, al := range p4Alphas {
				for _, be := range p4Betas {
					out = append(out, symmCombo{le, lo, al, be})
				}
			}
		}
	}
	return out
}

func symmCorners() []symmCombo {
	var out []symmCombo
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			out = append(out, symmCombo{le, lo, -0.75, 0.5})
		}
	}
	return out
}

func TestSsymmSweep(t *testing.T) {
	c := p4("Ssymm")
	c.dim("side", "L", "R")
	c.dim("uplo", "U", "L")
	c.dim("alpha", f32set(p4Alphas)...)
	c.dim("beta", f32set(p4Betas)...)
	c.l3config("keel.Ssymm->internal/block.Symm->expand+block.gemm")

	for ri, rn := range l3Runners() {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := symmCombos()
				if sz > p4ExactMax {
					cs := symmCorners()
					combos = []symmCombo{cs[ri%len(cs)]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							symmCase(t, c, rn, sz, sz, cb, sz)
						})
						if t.Failed() {
							return
						}
					}
				})
			}
		})
	}
	c.extra("unreferenced-triangle")
}

// symMatrix builds a d×d symmetric matrix stored in one triangle, with a NaN in
// every element of the other one.
//
// The NaN is the test: SSYMM's contract is that the unreferenced half is never
// read, and a routine that read it would produce NaN everywhere rather than a
// slightly wrong number. That makes this the rare case where a wrong answer is
// unmistakable instead of merely out of tolerance.
func symMatrix(r *rand.Rand, d, lda int, lower bool) []float32 {
	a := make([]float32, (d-1)*lda+d)
	nan := float32(math.NaN())
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if oracle.InTriangle(lower, i, j) {
				a[i*lda+j] = r.Float32()*2 - 1
			} else {
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func symmCase(t *testing.T, c *p4cov, rn l3runner, m, n int, cb symmCombo, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	d := n
	if cb.left {
		d = m
	}
	lda, ldb, ldc := d, n, n
	a := symMatrix(r, d, lda, cb.lower)
	a0 := append([]float32(nil), a...)
	b := randMatrix(r, m*ldb)
	b0 := append([]float32(nil), b...)
	cm := randMatrix(r, m*ldc)
	c0 := append([]float32(nil), cm...)

	rn.symm(cb.left, cb.lower, m, n, cb.alpha, a, lda, b, ldb, cb.beta, cm, ldc)

	what := fmt.Sprintf("Ssymm[%s] %s %dx%d", rn.name, cb, m, n)
	if !equalF32bits(a, a0) {
		t.Errorf("%s: A was modified", what)
		return
	}
	if !equalF32(b, b0) {
		t.Errorf("%s: B was modified", what)
		return
	}
	want, scale := oracle.Symm(cb.left, cb.lower, m, n, cb.alpha, a0, lda, b0, ldb, cb.beta, c0, ldc)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), d+1,
				cm[i*ldc+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return
			}
		}
	}
	// Every entry of C is checked at every size: unlike Ssyrk's, Ssymm's oracle is
	// one reduction per output element (oracle.SymmEntry), so an exhaustive
	// comparison at 500 costs the same order as the routine and there is nothing to
	// gain by sampling.
	c.verified(sz, exactMode(m*n))
}

// equalF32bits is equalF32 for data containing NaN: it compares bit patterns, so a
// NaN equals itself. Used where the check is "this was not written" rather than
// "this has the right value".
func equalF32bits(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float32bits(a[i]) != math.Float32bits(b[i]) {
			return false
		}
	}
	return true
}

// TestSsymmShapes moves m and n independently: for Left they are A's dimension and
// B's width, for Right the other way round, so a shape bug that cancels in the
// square case cannot cancel here.
func TestSsymmShapes(t *testing.T) {
	c := p4("Ssymm")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			for _, d := range dims {
				for _, cb := range symmCorners() {
					symmCase(t, c, rn, d[0], d[1], cb, d[0])
					if t.Failed() {
						return
					}
				}
			}
		})
	}
	c.extra("rectangular")
}

// ---------------------------------------------------------------------- Strsm

type trsmCombo struct {
	left, lower, trans, unit bool
	alpha                    float32
}

func (c trsmCombo) String() string {
	return fmt.Sprintf("side=%s/uplo=%s/%s/diag=%s/alpha=%s",
		sideStr(c.left), uploStr(c.lower), transStr(c.trans), diagStr(c.unit), f32str(c.alpha))
}

func (c trsmCombo) parts() []string {
	return []string{sideStr(c.left), uploStr(c.lower), transStr(c.trans),
		diagStr(c.unit), f32str(c.alpha)}
}

func trsmCombos() []trsmCombo {
	var out []trsmCombo
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, tr := range []bool{false, true} {
				for _, un := range []bool{false, true} {
					for _, al := range p4Alphas {
						out = append(out, trsmCombo{le, lo, tr, un, al})
					}
				}
			}
		}
	}
	return out
}

// trsmCorners is the sixteen flag combinations at a general alpha. Sizes above
// p4ExactMax run one of them per runner, chosen by index — five of the sixteen on
// an AVX-512 host. That is a real reduction and it is recorded here rather than
// implied: a 500×500 solve costs O(n³/2) in the routine and the same in the float64
// oracle, so sixteen of them per kernel is minutes for coverage the small sizes
// already have. What the large size adds is the blocked path — m > MB and m > MC
// both hold at 500 — and one combination per runner exercises it in each of the
// four block directions.
func trsmCorners() []trsmCombo {
	var out []trsmCombo
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, tr := range []bool{false, true} {
				for _, un := range []bool{false, true} {
					out = append(out, trsmCombo{le, lo, tr, un, -0.75})
				}
			}
		}
	}
	return out
}

func TestStrsmSweep(t *testing.T) {
	c := p4("Strsm")
	c.dim("side", "L", "R")
	c.dim("uplo", "U", "L")
	c.dim("trans", "N", "T")
	c.dim("diag", "N", "U")
	c.dim("alpha", f32set(p4Alphas)...)
	c.l3config("keel.Strsm->internal/block.Trsm->block.gemm+unblocked diagonal solve")

	for ri, rn := range l3Runners() {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := trsmCombos()
				if sz > p4ExactMax {
					cs := trsmCorners()
					combos = []trsmCombo{cs[ri%len(cs)]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							trsmCase(t, c, rn, sz, sz, cb, sz)
						})
						if t.Failed() {
							return
						}
					}
				})
			}
		})
	}
	c.extra("unreferenced-triangle")
	c.extra("unit-diagonal-ignored")
}

// triMatrix builds a d×d triangular matrix for Strsm: NaN everywhere the routine
// promises not to look, and a diagonally dominant referenced triangle.
//
// Both halves of that sentence are load-bearing.
//
// The NaN covers two separate promises with one value — the unreferenced triangle,
// and (when unit) the stored diagonal, which BLAS says is not referenced rather
// than saying it contains ones. A routine that read either would return NaN, and a
// caller holding an LU factorization in one array is relying on exactly this.
//
// Diagonal dominance is about the tolerance model rather than about coverage. A
// triangular solve's error bound grows multiplicatively down the substitution (see
// oracle.Trsm's derivation), so a random triangular matrix at n = 500 has a
// legitimate error bound wide enough to admit anything — the test would pass on a
// broken routine. Off-diagonals scaled by 1/d keep the growth factor near 1, which
// keeps the bound tight enough to mean something while still making every
// off-diagonal contribute: the correction each entry receives from the rank update
// is of order half its own magnitude, not a rounding.
func triMatrix(r *rand.Rand, d, lda int, lower, unit bool) []float32 {
	a := make([]float32, (d-1)*lda+d)
	nan := float32(math.NaN())
	scale := float32(1) / float32(d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			switch {
			case i == j && unit:
				a[i*lda+j] = nan
			case i == j:
				a[i*lda+j] = 1 + r.Float32()
			case oracle.InTriangle(lower, i, j):
				a[i*lda+j] = (r.Float32()*2 - 1) * scale
			default:
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func trsmCase(t *testing.T, c *p4cov, rn l3runner, m, n int, cb trsmCombo, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	d := n
	if cb.left {
		d = m
	}
	lda, ldb := d, n
	a := triMatrix(r, d, lda, cb.lower, cb.unit)
	a0 := append([]float32(nil), a...)
	b := randMatrix(r, m*ldb)
	b0 := append([]float32(nil), b...)

	rn.trsm(cb.left, cb.lower, cb.trans, cb.unit, m, n, cb.alpha, a, lda, b, ldb)

	what := fmt.Sprintf("Strsm[%s] %s %dx%d", rn.name, cb, m, n)
	if !equalF32bits(a, a0) {
		t.Errorf("%s: A was modified", what)
		return
	}
	// No entry form and no sampling: a substitution computes the whole solution to
	// reach any one entry of it, so the oracle has already produced every entry by
	// the time the first comparison happens. The economy for the large sizes is in
	// the number of flag combinations (see trsmCorners), not in the entries.
	want, scale := oracle.Trsm(cb.left, cb.lower, cb.trans, cb.unit, m, n, cb.alpha, a0, lda, b0, ldb)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), d+1,
				b[i*ldb+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return
			}
		}
	}
	c.verified(sz, exactMode(m*n))
}

// TestStrsmShapes moves m and n independently, so that A's dimension and B's other
// dimension are never the same number. For Left that varies the number of MB
// blocks against the width of the rank update; for Right it is the other way round.
func TestStrsmShapes(t *testing.T) {
	c := p4("Strsm")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			for _, d := range dims {
				for _, cb := range trsmCorners() {
					trsmCase(t, c, rn, d[0], d[1], cb, d[0])
					if t.Failed() {
						return
					}
				}
			}
		})
	}
	c.extra("rectangular")
}

// TestStrsmMultiBlock runs the one size where the outer block loop has more than
// one iteration but the sweep's rotation gives each runner only one combination:
// all sixteen corners at m = n = MB + 1, on every runner.
//
// MB is a var, so this is where a P5 retune that changed it would have to keep
// working; and it is the cheapest size at which the rank update against
// already-solved blocks runs at all, which is the part of the recipe the small
// sizes never reach.
func TestStrsmMultiBlock(t *testing.T) {
	c := p4("Strsm")
	sz := block.MB + 1
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			for _, cb := range trsmCorners() {
				trsmCase(t, c, rn, sz, sz, cb, sz)
				if t.Failed() {
					return
				}
			}
		})
	}
	c.extra("multi-block")
}

// ------------------------------------------------------------------- ld padding

// TestL3LdPad passes submatrix views to all three routines: every ld wider than
// the matrix it describes, with poison in the padding, which must come back
// untouched. This is the case where an off-by-one in the masked C update or in the
// unblocked solve writes into a neighbouring matrix rather than off the end of a
// slice.
func TestL3LdPad(t *testing.T) {
	const pad = 3
	sizes := []int{1, 5, 17, 33, 65}
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(p4Seed))
			for _, sz := range sizes {
				n, k := sz, sz

				// Ssyrk.
				cb := syrkCombo{true, false, -0.75, 0.5}
				lda, ldc := k+pad, n+pad
				a := padded(r, n, k, lda)
				cm := padded(r, n, n, ldc)
				c0 := append([]float32(nil), cm...)
				rn.syrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, cm, ldc)
				want, scale := oracle.Syrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, c0, ldc)
				what := fmt.Sprintf("Ssyrk[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j <= i; j++ {
						checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), k+1,
							cm[i*ldc+j], want[i*n+j], scale[i*n+j])
					}
				}
				checkPadding(t, what+" a", a, n, k, lda, nil)
				checkPadding(t, what+" c", cm, n, n, ldc, c0)

				// Ssymm, left, lower.
				lda, ldb, ldc2 := n+pad, n+pad, n+pad
				as := symPadded(r, n, lda, true)
				bm := padded(r, n, n, ldb)
				cm2 := padded(r, n, n, ldc2)
				c02 := append([]float32(nil), cm2...)
				rn.symm(true, true, n, n, -0.75, as, lda, bm, ldb, 0.5, cm2, ldc2)
				want2, scale2 := oracle.Symm(true, true, n, n, -0.75, as, lda, bm, ldb, 0.5, c02, ldc2)
				what = fmt.Sprintf("Ssymm[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), n+1,
							cm2[i*ldc2+j], want2[i*n+j], scale2[i*n+j])
					}
				}
				checkPadding(t, what+" c", cm2, n, n, ldc2, c02)

				// Strsm, right, upper, transposed, unit.
				at := triPadded(r, n, lda, false, true)
				bt := padded(r, n, n, ldb)
				b0 := append([]float32(nil), bt...)
				rn.trsm(false, false, true, true, n, n, -0.75, at, lda, bt, ldb)
				want3, scale3 := oracle.Trsm(false, false, true, true, n, n, -0.75, at, lda, b0, ldb)
				what = fmt.Sprintf("Strsm[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), n+1,
							bt[i*ldb+j], want3[i*n+j], scale3[i*n+j])
					}
				}
				checkPadding(t, what+" b", bt, n, n, ldb, b0)
				if t.Failed() {
					return
				}
			}
		})
	}
	for _, r := range []string{"Ssyrk", "Ssymm", "Strsm"} {
		p4(r).extra("ldpad")
	}
}

// symPadded and triPadded are symMatrix and triMatrix with a row stride wider than
// the matrix, poison in the gaps. The two concerns are orthogonal — what must not
// be read, and what must not be written — so the padding is poison (a finite value
// that a bit comparison catches) while the unreferenced triangle stays NaN.
func symPadded(r *rand.Rand, d, lda int, lower bool) []float32 {
	a := padded(r, d, d, lda)
	nan := float32(math.NaN())
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if !oracle.InTriangle(lower, i, j) {
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func triPadded(r *rand.Rand, d, lda int, lower, unit bool) []float32 {
	a := padded(r, d, d, lda)
	nan := float32(math.NaN())
	scale := float32(1) / float32(d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			switch {
			case i == j && unit:
				a[i*lda+j] = nan
			case i == j:
				a[i*lda+j] = 1 + r.Float32()
			case oracle.InTriangle(lower, i, j):
				a[i*lda+j] *= scale
			default:
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

// --------------------------------------------------------------- zero dimensions

// TestL3ZeroDim covers every dimension being zero, and the alpha == 0 shortcut
// that behaves differently in each of the three routines: Ssyrk and Ssymm scale C,
// Strsm zeroes B.
func TestL3ZeroDim(t *testing.T) {
	// Ssyrk, k == 0: the empty product still applies beta, to the referenced
	// triangle only. C's other half must survive, which is the interesting part —
	// "there is nothing to do" is wrong twice over here.
	cm := []float32{1, 2, 3, 4}
	Ssyrk(Lower, NoTrans, 2, 0, -0.75, nil, 1, 2, cm, 2)
	if want := []float32{2, 2, 6, 8}; !equalF32(cm, want) {
		t.Errorf("Ssyrk k=0 beta=2 lower: c = %v, want %v", cm, want)
	}
	cm = []float32{1, 2, 3, 4}
	Ssyrk(Upper, NoTrans, 2, 0, -0.75, nil, 1, 0, cm, 2)
	if want := []float32{0, 0, 3, 0}; !equalF32(cm, want) {
		t.Errorf("Ssyrk k=0 beta=0 upper: c = %v, want %v", cm, want)
	}
	// alpha == 0 with a NaN in A: A must not be read.
	nan := float32(math.NaN())
	cm = []float32{1, 2, 3, 4}
	Ssyrk(Lower, NoTrans, 2, 2, 0, []float32{nan, nan, nan, nan}, 2, 2, cm, 2)
	if want := []float32{2, 2, 6, 8}; !equalF32(cm, want) {
		t.Errorf("Ssyrk alpha=0 with NaN in A: c = %v, want %v", cm, want)
	}
	// n == 0: nothing at all, and the operands may be nil.
	Ssyrk(Lower, NoTrans, 0, 4, -0.75, nil, 4, 2, nil, 1)
	Ssyrk(Upper, Trans, 0, 4, -0.75, make([]float32, 16), 1, 2, nil, 1)

	// Ssymm: alpha == 0 scales C without reading A or B.
	cm = []float32{1, 2, 3, 4}
	bad := []float32{nan, nan, nan, nan}
	Ssymm(Left, Upper, 2, 2, 0, bad, 2, bad, 2, 2, cm, 2)
	if want := []float32{2, 4, 6, 8}; !equalF32(cm, want) {
		t.Errorf("Ssymm alpha=0: c = %v, want %v", cm, want)
	}
	Ssymm(Left, Upper, 0, 2, 1, nil, 1, nil, 2, 1, nil, 2)
	Ssymm(Right, Lower, 2, 0, 1, nil, 1, nil, 1, 1, nil, 1)

	// Strsm: alpha == 0 zeroes B without reading A, which is where a "solve" that
	// substituted through would divide by a diagonal it promised not to touch.
	b := []float32{1, 2, 3, 4}
	Strsm(Left, Lower, NoTrans, Unit, 2, 2, 0, bad, 2, b, 2)
	if want := []float32{0, 0, 0, 0}; !equalF32(b, want) {
		t.Errorf("Strsm alpha=0: b = %v, want %v", b, want)
	}
	Strsm(Left, Lower, NoTrans, NonUnit, 0, 2, 1, nil, 1, nil, 2)
	Strsm(Right, Upper, Trans, NonUnit, 2, 0, 1, nil, 1, nil, 1)

	for _, r := range []string{"Ssyrk", "Ssymm", "Strsm"} {
		p4(r).extra("zerodim")
	}
}

// ------------------------------------------------------------- argument errors

func TestL3ArgPanic(t *testing.T) {
	a := make([]float32, 16)
	cm := make([]float32, 16)
	cases := []struct {
		routine, what string
		f             func()
	}{
		{"Ssyrk", "uplo not U or L", func() { Ssyrk('X', NoTrans, 2, 2, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "trans not N or T", func() { Ssyrk(Lower, 'C', 2, 2, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "n < 0", func() { Ssyrk(Lower, NoTrans, -1, 2, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "k < 0", func() { Ssyrk(Lower, NoTrans, 2, -1, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "lda < k", func() { Ssyrk(Lower, NoTrans, 2, 4, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "lda < n when trans", func() { Ssyrk(Lower, Trans, 4, 2, 1, a, 2, 1, cm, 4) }},
		{"Ssyrk", "ldc < n", func() { Ssyrk(Lower, NoTrans, 4, 2, 1, a, 2, 1, cm, 2) }},
		{"Ssyrk", "c too short", func() { Ssyrk(Lower, NoTrans, 4, 4, 1, a, 4, 1, cm[:12], 4) }},
		{"Ssyrk", "a too short", func() { Ssyrk(Lower, NoTrans, 4, 4, 1, a[:12], 4, 1, cm, 4) }},

		{"Ssymm", "side not L or R", func() { Ssymm('X', Lower, 2, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Ssymm", "uplo not U or L", func() { Ssymm(Left, 'X', 2, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Ssymm", "m < 0", func() { Ssymm(Left, Lower, -1, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Ssymm", "n < 0", func() { Ssymm(Left, Lower, 2, -1, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Ssymm", "lda < m when left", func() { Ssymm(Left, Lower, 4, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Ssymm", "lda < n when right", func() { Ssymm(Right, Lower, 2, 4, 1, a, 2, a, 4, 1, cm, 4) }},
		{"Ssymm", "ldb < n", func() { Ssymm(Left, Lower, 2, 4, 1, a, 2, a, 2, 1, cm, 4) }},
		{"Ssymm", "ldc < n", func() { Ssymm(Left, Lower, 2, 4, 1, a, 2, a, 4, 1, cm, 2) }},
		{"Ssymm", "b too short", func() { Ssymm(Left, Lower, 4, 4, 1, a, 4, a[:12], 4, 1, cm, 4) }},

		{"Strsm", "side not L or R", func() { Strsm('X', Lower, NoTrans, NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "uplo not U or L", func() { Strsm(Left, 'X', NoTrans, NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "tA not N or T", func() { Strsm(Left, Lower, 'C', NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "diag not N or U", func() { Strsm(Left, Lower, NoTrans, 'X', 2, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "m < 0", func() { Strsm(Left, Lower, NoTrans, NonUnit, -1, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "n < 0", func() { Strsm(Left, Lower, NoTrans, NonUnit, 2, -1, 1, a, 2, cm, 2) }},
		{"Strsm", "lda < m when left", func() { Strsm(Left, Lower, NoTrans, NonUnit, 4, 2, 1, a, 2, cm, 2) }},
		{"Strsm", "lda < n when right", func() { Strsm(Right, Lower, NoTrans, NonUnit, 2, 4, 1, a, 2, cm, 4) }},
		{"Strsm", "ldb < n", func() { Strsm(Left, Lower, NoTrans, NonUnit, 2, 4, 1, a, 2, cm, 2) }},
		{"Strsm", "b too short", func() { Strsm(Left, Lower, NoTrans, NonUnit, 4, 4, 1, a, 4, cm[:12], 4) }},
	}
	for _, tc := range cases {
		mustPanic(t, tc.routine+" "+tc.what, tc.f)
	}
	for _, r := range []string{"Ssyrk", "Ssymm", "Strsm"} {
		p4(r).extra("argpanic")
	}
}

// ------------------------------------------------------------------ non-finite

// TestL3NonFinite pins the shortcut rules and the hazard the zero-padded edge
// strategy carries into the masked routines.
//
// A tile that straddles the diagonal is computed in full and half-discarded, and a
// fringe tile's panels are zero-padded, so an infinity in real data meets a zero
// and produces a NaN in the scratch tile. internal/block's copy-back is what keeps
// that NaN out of both C's other triangle and its padding; this is the test that
// says so, at a size where the two effects coincide.
func TestL3NonFinite(t *testing.T) {
	inf := float32(math.Inf(1))
	for _, rn := range l3Runners() {
		t.Run(rn.name, func(t *testing.T) {
			const n, k = 5, 3
			r := rand.New(rand.NewSource(p4Seed))

			for _, lower := range []bool{false, true} {
				a := randMatrix(r, n*k)
				a[0] = inf
				cm := randMatrix(r, n*n)
				c0 := append([]float32(nil), cm...)
				rn.syrk(lower, false, n, k, 1, a, k, 1, cm, n)
				want, scale := oracle.Syrk(lower, false, n, k, 1, a, k, 1, c0, n)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						if !oracle.InTriangle(lower, i, j) {
							if cm[i*n+j] != c0[i*n+j] {
								t.Fatalf("Ssyrk[%s] inf uplo=%s: the unreferenced triangle at [%d,%d] changed to %v",
									rn.name, uploStr(lower), i, j, cm[i*n+j])
							}
							continue
						}
						checkScalar(t, fmt.Sprintf("Ssyrk[%s] inf uplo=%s [%d,%d]", rn.name, uploStr(lower), i, j),
							k+1, cm[i*n+j], want[i*n+j], scale[i*n+j])
					}
				}
			}

			// Ssymm with an infinity in B, and Strsm with one in B: both compared
			// against the oracle rather than against an expectation, so that
			// whatever the arithmetic does with it has to be the same on both sides.
			as := symMatrix(r, n, n, true)
			bm := randMatrix(r, n*n)
			bm[n-1] = inf
			cm := randMatrix(r, n*n)
			c0 := append([]float32(nil), cm...)
			rn.symm(true, true, n, n, 1, as, n, bm, n, 1, cm, n)
			want, scale := oracle.Symm(true, true, n, n, 1, as, n, bm, n, 1, c0, n)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					checkScalar(t, fmt.Sprintf("Ssymm[%s] inf [%d,%d]", rn.name, i, j),
						n+1, cm[i*n+j], want[i*n+j], scale[i*n+j])
				}
			}

			at := triMatrix(r, n, n, true, false)
			bt := randMatrix(r, n*n)
			bt[0] = inf
			b0 := append([]float32(nil), bt...)
			rn.trsm(true, true, false, false, n, n, 1, at, n, bt, n)
			want3, scale3 := oracle.Trsm(true, true, false, false, n, n, 1, at, n, b0, n)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					checkScalar(t, fmt.Sprintf("Strsm[%s] inf [%d,%d]", rn.name, i, j),
						n+1, bt[i*n+j], want3[i*n+j], scale3[i*n+j])
				}
			}
		})
	}
	for _, r := range []string{"Ssyrk", "Ssymm", "Strsm"} {
		p4(r).extra("nonfinite")
	}
}

// The mask's three range predicates are tested where they live, against the
// element-wise definition they are derived from: internal/block/tri_test.go.
