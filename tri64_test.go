// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Produced from tri_test.go by a one-off rename for #193 and kept as ordinary test code: the
// float64 P4 sweep, registering keel-p4-* coverage for the D routines so gate-p4's
// KEEL_ELEM=f64 arm audits the same lattice it audits at float32. Its references are
// the definition-order float64 oracle under PeerTolerance64 (checkScalarD); the
// double-double arm for these routines is l2_64_test.go's and l3_64_test.go's.
// A float32 change to tri_test.go must be mirrored here by hand.

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"github.com/scttfrdmn/keel/internal/block64"
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/l164"
	"github.com/scttfrdmn/keel/internal/oracle"
)

// Dsyrk, Dsymm and Dtrsm against the float64 oracle (DESIGN.md §4/P4).
//
// # What these tests are for, given that Dgemm is already tested
//
// The three routines are derivations on Dgemm's loop nest (internal/block/tri.go),
// so the arithmetic underneath them is P3's and is already held to the oracle over
// a bigger sweep than this one. What is new in P4 is not arithmetic, it is
// *addressing under a promise*: which triangle is read, which is written, which
// diagonal is referenced, and which half of a tile that straddles the diagonal
// reaches C. Every one of those is a promise about memory the routine must not
// touch, and a wrong answer is the least of the ways to break it — the worse one
// is a correct answer computed by reading a caller's other matrix.
//
// So the sweeps below poisonF64 what must not be read (NaN in A's unreferenced
// triangle, NaN on a unit diagonal) and copy what must not be written (C's
// unreferenced triangle, compared bit-for-bit afterwards). A NaN that is read
// propagates into the result and fails the value comparison; a write that must
// not have happened fails the bit comparison. Neither can be absorbed by a
// tolerance, which is the point.

// l3runner64 is one way to reach a derived routine: the public entry point, or a
// specific microkernel driven through the blocking nest directly. Same rationale
// as gemmRunner's, and the same differential value — on an AVX-512 host the sweep
// runs on both shipped vector shapes and both scalar references, so a masking bug
// that happens to be invisible at one tile shape still shows up.
type l3runner64 struct {
	name    string
	backend string
	syrk    func(lower, trans bool, n, k int, alpha float64, a []float64, lda int, beta float64, c []float64, ldc int)
	symm    func(left, lower bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int)
	trsm    func(left, lower, trans, unit bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int)
}

func l3Runners64() []l3runner64 {
	rs := []l3runner64{{
		name:    "public",
		backend: ActiveKern64Backend(),
		syrk: func(lower, trans bool, n, k int, alpha float64, a []float64, lda int, beta float64, c []float64, ldc int) {
			Dsyrk(uploFlag(lower), transFlag(trans), n, k, alpha, a, lda, beta, c, ldc)
		},
		symm: func(left, lower bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
			Dsymm(sideFlag(left), uploFlag(lower), m, n, alpha, a, lda, b, ldb, beta, c, ldc)
		},
		trsm: func(left, lower, trans, unit bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
			Dtrsm(sideFlag(left), uploFlag(lower), transFlag(trans), diagFlag(unit), m, n, alpha, a, lda, b, ldb)
		},
	}}
	for _, kn := range kern.Kernels64() {
		kn := kn
		rs = append(rs, l3runner64{
			name:    kn.ID(),
			backend: kn.Name,
			syrk: func(lower, trans bool, n, k int, alpha float64, a []float64, lda int, beta float64, c []float64, ldc int) {
				block64.Syrk(kn, lower, trans, n, k, alpha, a, lda, beta, c, ldc)
			},
			symm: func(left, lower bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int, beta float64, c []float64, ldc int) {
				block64.Symm(kn, left, lower, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
			},
			trsm: func(left, lower, trans, unit bool, m, n int, alpha float64, a []float64, lda int, b []float64, ldb int) {
				block64.Trsm(kn, l164.Backends()[0].Axpy, l164.Backends()[0].ScalDiv, left, lower, trans, unit, m, n, alpha, a, lda, b, ldb)
			},
		})
	}
	return rs
}

// ---------------------------------------------------------------------- Dsyrk

type syrkCombo64 struct {
	lower, trans bool
	alpha, beta  float64
}

func (c syrkCombo64) String() string {
	return fmt.Sprintf("uplo=%s/%s/alpha=%s/beta=%s",
		uploStr(c.lower), transStr(c.trans), f64str(c.alpha), f64str(c.beta))
}

func (c syrkCombo64) parts() []string {
	return []string{uploStr(c.lower), transStr(c.trans), f64str(c.alpha), f64str(c.beta)}
}

func syrkCombos64() []syrkCombo64 {
	var out []syrkCombo64
	for _, lo := range []bool{false, true} {
		for _, tr := range []bool{false, true} {
			for _, al := range p4Alphas64 {
				for _, be := range p4Betas64 {
					out = append(out, syrkCombo64{lo, tr, al, be})
				}
			}
		}
	}
	return out
}

// syrkCorners64 is the flag lattice with the shortcuts removed: the four
// (uplo, trans) combinations at a general alpha and beta. Sizes above p4ExactMax
// run one of these, chosen by the runner's index, for the reason P3's sweep gives
// for its own rotation — the large sizes are here to exercise the loop nest, and
// running 36 combinations of it costs a gate run and buys nothing. Rotating over
// the runner index rather than fixing one combination spreads the corners across
// the kernels; how many are reached depends on how many runners the host has, and
// TestDweepCornerCoverage names the ones that are not.
func syrkCorners64() []syrkCombo64 {
	var out []syrkCombo64
	for _, lo := range []bool{false, true} {
		for _, tr := range []bool{false, true} {
			out = append(out, syrkCombo64{lo, tr, -0.75, 0.5})
		}
	}
	return out
}

func TestDsyrkSweep(t *testing.T) {
	c := p4("Dsyrk")
	c.dim("uplo", "U", "L")
	c.dim("trans", "N", "T")
	c.dim("alpha", f64set(p4Alphas64)...)
	c.dim("beta", f64set(p4Betas64)...)
	c.l3config64("keel.Dsyrk->internal/block64.Syrk->block64.gemm(triMask)")

	rns := l3Runners64()
	for ri, rn := range rns {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := syrkCombos64()
				if sz > p4ExactMax {
					cs := syrkCorners64()
					combos = []syrkCombo64{cs[cornerFor64(ri, len(rns), len(cs))]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							syrkCase64(t, c, rn, sz, sz, cb, sz)
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

// syrkCase64 runs one (runner, size, combination) and checks both halves of SSYRK's
// contract: the referenced triangle against the oracle, and the other one against
// the bytes that were there before the call.
func syrkCase64(t *testing.T, c *p4cov, rn l3runner64, n, k int, cb syrkCombo64, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(n*1000+k)))
	ra, ca := n, k
	if cb.trans {
		ra, ca = k, n
	}
	lda, ldc := ca, n
	a := randMatrix64(r, ra*lda)
	a0 := append([]float64(nil), a...)
	cm := randMatrix64(r, n*ldc)
	c0 := append([]float64(nil), cm...)

	rn.syrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, cm, ldc)

	what := fmt.Sprintf("Dsyrk[%s] %s n=%d k=%d", rn.name, cb, n, k)
	if !equalF64(a, a0) {
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
		want, scale := oracle.Dsyrk(cb.lower, cb.trans, n, k, cb.alpha, a0, lda, cb.beta, c0, ldc)
		entries := 0
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if !oracle.InTriangle(cb.lower, i, j) {
					continue
				}
				entries++
				checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), k+1,
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
		want, scale := oracle.DsyrkEntry(cb.trans, p[0], p[1], k, cb.alpha, a0, lda, cb.beta, c0, ldc)
		checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, p[0], p[1]), k+1,
			cm[p[0]*ldc+p[1]], want, scale)
		if t.Failed() {
			return
		}
	}
	c.verified(sz, sampledMode(len(pts), p4Seed))
}

// TestDsyrkShapes moves k independently of n, around both tile bounds and the
// kernels' unroll. The square sweep cannot distinguish a depth-loop bug from a
// mask bug because it never has one dimension without the other.
func TestDsyrkShapes(t *testing.T) {
	c := p4("Dsyrk")
	ns := []int{1, 2, 3, 5, 8, 31, 32, 33, 65}
	ks := []int{1, 2, 3, 4, 7, 8, 33}
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			for _, n := range ns {
				for _, k := range ks {
					for _, cb := range syrkCorners64() {
						syrkCase64(t, c, rn, n, k, cb, n)
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

// ---------------------------------------------------------------------- Dsymm

type symmCombo64 struct {
	left, lower bool
	alpha, beta float64
}

func (c symmCombo64) String() string {
	return fmt.Sprintf("side=%s/uplo=%s/alpha=%s/beta=%s",
		sideStr(c.left), uploStr(c.lower), f64str(c.alpha), f64str(c.beta))
}

func (c symmCombo64) parts() []string {
	return []string{sideStr(c.left), uploStr(c.lower), f64str(c.alpha), f64str(c.beta)}
}

func symmCombos64() []symmCombo64 {
	var out []symmCombo64
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, al := range p4Alphas64 {
				for _, be := range p4Betas64 {
					out = append(out, symmCombo64{le, lo, al, be})
				}
			}
		}
	}
	return out
}

func symmCorners64() []symmCombo64 {
	var out []symmCombo64
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			out = append(out, symmCombo64{le, lo, -0.75, 0.5})
		}
	}
	return out
}

func TestDsymmSweep(t *testing.T) {
	c := p4("Dsymm")
	c.dim("side", "L", "R")
	c.dim("uplo", "U", "L")
	c.dim("alpha", f64set(p4Alphas64)...)
	c.dim("beta", f64set(p4Betas64)...)
	c.l3config64("keel.Dsymm->internal/block64.Symm->block64.gemm(sym-pack)")

	rns := l3Runners64()
	for ri, rn := range rns {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := symmCombos64()
				if sz > p4ExactMax {
					cs := symmCorners64()
					combos = []symmCombo64{cs[cornerFor64(ri, len(rns), len(cs))]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							symmCase64(t, c, rn, sz, sz, cb, sz)
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

// symMatrix64 builds a d×d symmetric matrix stored in one triangle, with a NaN in
// every element of the other one.
//
// The NaN is the test: SSYMM's contract is that the unreferenced half is never
// read, and a routine that read it would produce NaN everywhere rather than a
// slightly wrong number. That makes this the rare case where a wrong answer is
// unmistakable instead of merely out of tolerance.
func symMatrix64(r *rand.Rand, d, lda int, lower bool) []float64 {
	a := make([]float64, (d-1)*lda+d)
	nan := math.NaN()
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if oracle.InTriangle(lower, i, j) {
				a[i*lda+j] = r.Float64()*2 - 1
			} else {
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func symmCase64(t *testing.T, c *p4cov, rn l3runner64, m, n int, cb symmCombo64, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	d := n
	if cb.left {
		d = m
	}
	lda, ldb, ldc := d, n, n
	a := symMatrix64(r, d, lda, cb.lower)
	a0 := append([]float64(nil), a...)
	b := randMatrix64(r, m*ldb)
	b0 := append([]float64(nil), b...)
	cm := randMatrix64(r, m*ldc)
	c0 := append([]float64(nil), cm...)

	rn.symm(cb.left, cb.lower, m, n, cb.alpha, a, lda, b, ldb, cb.beta, cm, ldc)

	what := fmt.Sprintf("Dsymm[%s] %s %dx%d", rn.name, cb, m, n)
	if !equalF32bits64(a, a0) {
		t.Errorf("%s: A was modified", what)
		return
	}
	if !equalF64(b, b0) {
		t.Errorf("%s: B was modified", what)
		return
	}
	want, scale := oracle.Dsymm(cb.left, cb.lower, m, n, cb.alpha, a0, lda, b0, ldb, cb.beta, c0, ldc)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), d+1,
				cm[i*ldc+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return
			}
		}
	}
	// Every entry of C is checked at every size: unlike Dsyrk's, Dsymm's oracle is
	// one reduction per output element (oracle.SymmEntry), so an exhaustive
	// comparison at 500 costs the same order as the routine and there is nothing to
	// gain by sampling.
	c.verified(sz, exactMode(m*n))
}

// equalF32bits64 is equalF64 for data containing NaN: it compares bit patterns, so a
// NaN equals itself. Used where the check is "this was not written" rather than
// "this has the right value".
func equalF32bits64(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return false
		}
	}
	return true
}

// TestDsymmShapes moves m and n independently: for Left they are A's dimension and
// B's width, for Right the other way round, so a shape bug that cancels in the
// square case cannot cancel here.
func TestDsymmShapes(t *testing.T) {
	c := p4("Dsymm")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			for _, d := range dims {
				for _, cb := range symmCorners64() {
					symmCase64(t, c, rn, d[0], d[1], cb, d[0])
					if t.Failed() {
						return
					}
				}
			}
		})
	}
	c.extra("rectangular")
}

// ---------------------------------------------------------------------- Dtrsm

type trsmCombo64 struct {
	left, lower, trans, unit bool
	alpha                    float64
}

func (c trsmCombo64) String() string {
	return fmt.Sprintf("side=%s/uplo=%s/%s/diag=%s/alpha=%s",
		sideStr(c.left), uploStr(c.lower), transStr(c.trans), diagStr(c.unit), f64str(c.alpha))
}

func (c trsmCombo64) parts() []string {
	return []string{sideStr(c.left), uploStr(c.lower), transStr(c.trans),
		diagStr(c.unit), f64str(c.alpha)}
}

func trsmCombos64() []trsmCombo64 {
	var out []trsmCombo64
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, tr := range []bool{false, true} {
				for _, un := range []bool{false, true} {
					for _, al := range p4Alphas64 {
						out = append(out, trsmCombo64{le, lo, tr, un, al})
					}
				}
			}
		}
	}
	return out
}

// trsmCorners64 is the sixteen flag combinations at a general alpha. Sizes above
// p4ExactMax run one of them per runner, chosen by index — five of the sixteen on
// an AVX-512 host. That is a real reduction and it is recorded here rather than
// implied: a 500×500 solve costs O(n³/2) in the routine and the same in the float64
// oracle, so sixteen of them per kernel is minutes for coverage the small sizes
// already have. What the large size adds is the blocked path — m > MB and m > MC
// both hold at 500 — and one combination per runner exercises it in each of the
// four block directions.
func trsmCorners64() []trsmCombo64 {
	var out []trsmCombo64
	for _, le := range []bool{true, false} {
		for _, lo := range []bool{false, true} {
			for _, tr := range []bool{false, true} {
				for _, un := range []bool{false, true} {
					out = append(out, trsmCombo64{le, lo, tr, un, -0.75})
				}
			}
		}
	}
	return out
}

// cornerFor64 picks runner ri's corner out of n, spreadFinite64 across the whole list
// rather than walking its front. `ri % n` was the bug (#64): the corner lists vary
// `side` slowest, and max(ri) is 1 on the dev host and 4 on an AVX-512 one, so no
// host this project runs on ever reached a right-side corner at a size where the
// blocked path runs. The miss was already cashed — `par.Run(1, body)` passed a
// unit count where a block count belonged, and right-side Dtrsm at 500 was wrong
// by 4.2e-2 against the float64 oracle at GOMAXPROCS<=2 and right at 8.
func cornerFor64(ri, nr, n int) int { return ri * n / nr % n }

// TestDweepCornerCoverage asserts on the sweeps' index space rather than on a
// kernel, because no assertion inside a case could have caught #64: the sweep
// passed, and passed correctly, on every case it ran. A parameter sweep whose
// index space is smaller than its parameter space has a hole, and the hole is
// invisible from inside the test.
//
// What is asserted is the property that was lost, not one the host cannot meet.
// With nr runners and n corners only nr of them can run, so "all corners" is
// unassertable at nr < n and asserting it would be theatre. The leading flag —
// `side` for Dsymm and Dtrsm, `uplo` for Dsyrk — is what selects a distinct code
// path through the nest, and every corner list varies it slowest, so it is exactly
// the flag `ri % n` aliased away.
//
// Asserted for every plausible runner count and not only this host's, because the
// count varies with the silicon: 3 on a scalar-only build, 5 on an AVX-512 host,
// and CI is the one machine here without AVX-512. A host-local check would leave
// the shapes that matter most unasserted. The live host's own reduction is logged
// beside it, since a reduction nobody can see reads as coverage.
func TestDweepCornerCoverage(t *testing.T) {
	nr := len(l3Runners64())
	if nr < 2 {
		t.Fatalf("l3Runners64() returned %d runner(s): the large-size arm reaches one corner per runner, so a single runner cannot span any flag", nr)
	}
	for _, s := range []struct {
		name  string
		lead  string
		heads []string
	}{
		{"Dsyrk", "uplo", cornerHeads64(syrkCorners64(), func(c syrkCombo64) string { return uploStr(c.lower) })},
		{"Dsymm", "side", cornerHeads64(symmCorners64(), func(c symmCombo64) string { return sideStr(c.left) })},
		{"Dtrsm", "side", cornerHeads64(trsmCorners64(), func(c trsmCombo64) string { return sideStr(c.left) })},
	} {
		n := len(s.heads)
		for k := 2; k <= 8; k++ {
			hit, reached := map[int]bool{}, map[string]bool{}
			for ri := 0; ri < k; ri++ {
				i := cornerFor64(ri, k, n)
				hit[i] = true
				reached[s.heads[i]] = true
			}
			for _, v := range s.heads {
				if !reached[v] {
					t.Errorf("%s's large-size sweep never runs %s=%s at %d runners: it reaches indices %v, and the corner list varies %s slowest, so an unreached half is a code path no host of that shape exercises above p4ExactMax (#64)",
						s.name, s.lead, v, k, sortedKeys64(hit), s.lead)
					break
				}
			}
			if k == nr && len(hit) < n {
				t.Logf("%s on this host (%d runners): %d of %d corners run at the large size, at indices %v — a stated reduction, not a silent one",
					s.name, nr, len(hit), n, sortedKeys64(hit))
			}
		}
	}
}

func cornerHeads64[T any](cs []T, lead func(T) string) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = lead(c)
	}
	return out
}

func sortedKeys64(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func TestDtrsmSweep(t *testing.T) {
	c := p4("Dtrsm")
	c.dim("side", "L", "R")
	c.dim("uplo", "U", "L")
	c.dim("trans", "N", "T")
	c.dim("diag", "N", "U")
	c.dim("alpha", f64set(p4Alphas64)...)
	c.l3config64("keel.Dtrsm->internal/block64.Trsm->block64.gemm+unblocked diagonal solve")

	rns := l3Runners64()
	for ri, rn := range rns {
		c.backend(rn.backend)
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range p4Sizes {
				c.size(sz)
				combos := trsmCombos64()
				if sz > p4ExactMax {
					cs := trsmCorners64()
					combos = []trsmCombo64{cs[cornerFor64(ri, len(rns), len(cs))]}
				}
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range combos {
						c.combo(cb.parts()...)
						t.Run(cb.String(), func(t *testing.T) {
							trsmCase64(t, c, rn, sz, sz, cb, sz)
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

// triMatrix64 builds a d×d triangular matrix for Dtrsm: NaN everywhere the routine
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
func triMatrix64(r *rand.Rand, d, lda int, lower, unit bool) []float64 {
	a := make([]float64, (d-1)*lda+d)
	nan := math.NaN()
	scale := float64(1) / float64(d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			switch {
			case i == j && unit:
				a[i*lda+j] = nan
			case i == j:
				a[i*lda+j] = 1 + r.Float64()
			case oracle.InTriangle(lower, i, j):
				a[i*lda+j] = (r.Float64()*2 - 1) * scale
			default:
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func trsmCase64(t *testing.T, c *p4cov, rn l3runner64, m, n int, cb trsmCombo64, sz int) {
	t.Helper()
	r := rand.New(rand.NewSource(p4Seed + int64(m*1000+n)))
	d := n
	if cb.left {
		d = m
	}
	lda, ldb := d, n
	a := triMatrix64(r, d, lda, cb.lower, cb.unit)
	a0 := append([]float64(nil), a...)
	b := randMatrix64(r, m*ldb)
	b0 := append([]float64(nil), b...)

	rn.trsm(cb.left, cb.lower, cb.trans, cb.unit, m, n, cb.alpha, a, lda, b, ldb)

	what := fmt.Sprintf("Dtrsm[%s] %s %dx%d", rn.name, cb, m, n)
	if !equalF32bits64(a, a0) {
		t.Errorf("%s: A was modified", what)
		return
	}
	// No entry form and no sampling: a substitution computes the whole solution to
	// reach any one entry of it, so the oracle has already produced every entry by
	// the time the first comparison happens. The economy for the large sizes is in
	// the number of flag combinations (see trsmCorners64), not in the entries.
	want, scale := oracle.Dtrsm(cb.left, cb.lower, cb.trans, cb.unit, m, n, cb.alpha, a0, lda, b0, ldb)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), d+1,
				b[i*ldb+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return
			}
		}
	}
	c.verified(sz, exactMode(m*n))
}

// TestDtrsmShapes moves m and n independently, so that A's dimension and B's other
// dimension are never the same number. For Left that varies the number of MB
// blocks against the width of the rank update; for Right it is the other way round.
func TestDtrsmShapes(t *testing.T) {
	c := p4("Dtrsm")
	dims := [][2]int{{1, 7}, {7, 1}, {3, 16}, {16, 3}, {17, 33}, {33, 17}, {64, 65}, {65, 64}}
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			for _, d := range dims {
				for _, cb := range trsmCorners64() {
					trsmCase64(t, c, rn, d[0], d[1], cb, d[0])
					if t.Failed() {
						return
					}
				}
			}
		})
	}
	c.extra("rectangular")
}

// TestDtrsmMultiBlock runs the one size where the outer block loop has more than
// one iteration but the sweep's rotation gives each runner only one combination:
// all sixteen corners at m = n = MB + 1, on every runner.
//
// MB is a var, so this is where a P5 retune that changed it would have to keep
// working; and it is the cheapest size at which the rank update against
// already-solved blocks runs at all, which is the part of the recipe the small
// sizes never reach.
func TestDtrsmMultiBlock(t *testing.T) {
	c := p4("Dtrsm")
	sz := block64.MB + 1
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			for _, cb := range trsmCorners64() {
				trsmCase64(t, c, rn, sz, sz, cb, sz)
				if t.Failed() {
					return
				}
			}
		})
	}
	c.extra("multi-block")
}

// ------------------------------------------------------------------- ld padding

// TestL3LdPad64 passes submatrix views to all three routines: every ld wider than
// the matrix it describes, with poisonF64 in the padding, which must come back
// untouched. This is the case where an off-by-one in the masked C update or in the
// unblocked solve writes into a neighbouring matrix rather than off the end of a
// slice.
func TestL3LdPad64(t *testing.T) {
	const pad = 3
	sizes := []int{1, 5, 17, 33, 65}
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(p4Seed))
			for _, sz := range sizes {
				n, k := sz, sz

				// Dsyrk.
				cb := syrkCombo64{true, false, -0.75, 0.5}
				lda, ldc := k+pad, n+pad
				a := padded64(r, n, k, lda)
				cm := padded64(r, n, n, ldc)
				c0 := append([]float64(nil), cm...)
				rn.syrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, cm, ldc)
				want, scale := oracle.Dsyrk(cb.lower, cb.trans, n, k, cb.alpha, a, lda, cb.beta, c0, ldc)
				what := fmt.Sprintf("Dsyrk[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j <= i; j++ {
						checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), k+1,
							cm[i*ldc+j], want[i*n+j], scale[i*n+j])
					}
				}
				checkPadding64(t, what+" a", a, n, k, lda, nil)
				checkPadding64(t, what+" c", cm, n, n, ldc, c0)

				// Dsymm, left, lower.
				lda, ldb, ldc2 := n+pad, n+pad, n+pad
				as := symPadded64(r, n, lda, true)
				bm := padded64(r, n, n, ldb)
				cm2 := padded64(r, n, n, ldc2)
				c02 := append([]float64(nil), cm2...)
				rn.symm(true, true, n, n, -0.75, as, lda, bm, ldb, 0.5, cm2, ldc2)
				want2, scale2 := oracle.Dsymm(true, true, n, n, -0.75, as, lda, bm, ldb, 0.5, c02, ldc2)
				what = fmt.Sprintf("Dsymm[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), n+1,
							cm2[i*ldc2+j], want2[i*n+j], scale2[i*n+j])
					}
				}
				checkPadding64(t, what+" c", cm2, n, n, ldc2, c02)

				// Dtrsm, right, upper, transposed, unit.
				at := triPadded64(r, n, lda, false, true)
				bt := padded64(r, n, n, ldb)
				b0 := append([]float64(nil), bt...)
				rn.trsm(false, false, true, true, n, n, -0.75, at, lda, bt, ldb)
				want3, scale3 := oracle.Dtrsm(false, false, true, true, n, n, -0.75, at, lda, b0, ldb)
				what = fmt.Sprintf("Dtrsm[%s] ldpad n=%d", rn.name, sz)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						checkScalarD(t, fmt.Sprintf("%s [%d,%d]", what, i, j), n+1,
							bt[i*ldb+j], want3[i*n+j], scale3[i*n+j])
					}
				}
				checkPadding64(t, what+" b", bt, n, n, ldb, b0)
				if t.Failed() {
					return
				}
			}
		})
	}
	for _, r := range []string{"Dsyrk", "Dsymm", "Dtrsm"} {
		p4(r).extra("ldpad")
	}
}

// symPadded64 and triPadded64 are symMatrix64 and triMatrix64 with a row stride wider than
// the matrix, poisonF64 in the gaps. The two concerns are orthogonal — what must not
// be read, and what must not be written — so the padding is poisonF64 (a finite value
// that a bit comparison catches) while the unreferenced triangle stays NaN.
func symPadded64(r *rand.Rand, d, lda int, lower bool) []float64 {
	a := padded64(r, d, d, lda)
	nan := math.NaN()
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			if !oracle.InTriangle(lower, i, j) {
				a[i*lda+j] = nan
			}
		}
	}
	return a
}

func triPadded64(r *rand.Rand, d, lda int, lower, unit bool) []float64 {
	a := padded64(r, d, d, lda)
	nan := math.NaN()
	scale := float64(1) / float64(d)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			switch {
			case i == j && unit:
				a[i*lda+j] = nan
			case i == j:
				a[i*lda+j] = 1 + r.Float64()
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

// TestL3ZeroDim64 covers every dimension being zero, and the alpha == 0 shortcut
// that behaves differently in each of the three routines: Dsyrk and Dsymm scale C,
// Dtrsm zeroes B.
func TestL3ZeroDim64(t *testing.T) {
	// Dsyrk, k == 0: the empty product still applies beta, to the referenced
	// triangle only. C's other half must survive, which is the interesting part —
	// "there is nothing to do" is wrong twice over here.
	cm := []float64{1, 2, 3, 4}
	Dsyrk(Lower, NoTrans, 2, 0, -0.75, nil, 1, 2, cm, 2)
	if want := []float64{2, 2, 6, 8}; !equalF64(cm, want) {
		t.Errorf("Dsyrk k=0 beta=2 lower: c = %v, want %v", cm, want)
	}
	cm = []float64{1, 2, 3, 4}
	Dsyrk(Upper, NoTrans, 2, 0, -0.75, nil, 1, 0, cm, 2)
	if want := []float64{0, 0, 3, 0}; !equalF64(cm, want) {
		t.Errorf("Dsyrk k=0 beta=0 upper: c = %v, want %v", cm, want)
	}
	// alpha == 0 with a NaN in A: A must not be read.
	nan := math.NaN()
	cm = []float64{1, 2, 3, 4}
	Dsyrk(Lower, NoTrans, 2, 2, 0, []float64{nan, nan, nan, nan}, 2, 2, cm, 2)
	if want := []float64{2, 2, 6, 8}; !equalF64(cm, want) {
		t.Errorf("Dsyrk alpha=0 with NaN in A: c = %v, want %v", cm, want)
	}
	// n == 0: nothing at all, and the operands may be nil.
	Dsyrk(Lower, NoTrans, 0, 4, -0.75, nil, 4, 2, nil, 1)
	Dsyrk(Upper, Trans, 0, 4, -0.75, make([]float64, 16), 1, 2, nil, 1)

	// Dsymm: alpha == 0 scales C without reading A or B.
	cm = []float64{1, 2, 3, 4}
	bad := []float64{nan, nan, nan, nan}
	Dsymm(Left, Upper, 2, 2, 0, bad, 2, bad, 2, 2, cm, 2)
	if want := []float64{2, 4, 6, 8}; !equalF64(cm, want) {
		t.Errorf("Dsymm alpha=0: c = %v, want %v", cm, want)
	}
	Dsymm(Left, Upper, 0, 2, 1, nil, 1, nil, 2, 1, nil, 2)
	Dsymm(Right, Lower, 2, 0, 1, nil, 1, nil, 1, 1, nil, 1)

	// Dtrsm: alpha == 0 zeroes B without reading A, which is where a "solve" that
	// substituted through would divide by a diagonal it promised not to touch.
	b := []float64{1, 2, 3, 4}
	Dtrsm(Left, Lower, NoTrans, Unit, 2, 2, 0, bad, 2, b, 2)
	if want := []float64{0, 0, 0, 0}; !equalF64(b, want) {
		t.Errorf("Dtrsm alpha=0: b = %v, want %v", b, want)
	}
	Dtrsm(Left, Lower, NoTrans, NonUnit, 0, 2, 1, nil, 1, nil, 2)
	Dtrsm(Right, Upper, Trans, NonUnit, 2, 0, 1, nil, 1, nil, 1)

	for _, r := range []string{"Dsyrk", "Dsymm", "Dtrsm"} {
		p4(r).extra("zerodim")
	}
}

// ------------------------------------------------------------- argument errors

func TestL3ArgPanic64(t *testing.T) {
	a := make([]float64, 16)
	cm := make([]float64, 16)
	cases := []struct {
		routine, what string
		f             func()
	}{
		{"Dsyrk", "uplo not U or L", func() { Dsyrk('X', NoTrans, 2, 2, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "trans not N or T", func() { Dsyrk(Lower, 'C', 2, 2, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "n < 0", func() { Dsyrk(Lower, NoTrans, -1, 2, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "k < 0", func() { Dsyrk(Lower, NoTrans, 2, -1, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "lda < k", func() { Dsyrk(Lower, NoTrans, 2, 4, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "lda < n when trans", func() { Dsyrk(Lower, Trans, 4, 2, 1, a, 2, 1, cm, 4) }},
		{"Dsyrk", "ldc < n", func() { Dsyrk(Lower, NoTrans, 4, 2, 1, a, 2, 1, cm, 2) }},
		{"Dsyrk", "c too short", func() { Dsyrk(Lower, NoTrans, 4, 4, 1, a, 4, 1, cm[:12], 4) }},
		{"Dsyrk", "a too short", func() { Dsyrk(Lower, NoTrans, 4, 4, 1, a[:12], 4, 1, cm, 4) }},

		{"Dsymm", "side not L or R", func() { Dsymm('X', Lower, 2, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Dsymm", "uplo not U or L", func() { Dsymm(Left, 'X', 2, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Dsymm", "m < 0", func() { Dsymm(Left, Lower, -1, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Dsymm", "n < 0", func() { Dsymm(Left, Lower, 2, -1, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Dsymm", "lda < m when left", func() { Dsymm(Left, Lower, 4, 2, 1, a, 2, a, 2, 1, cm, 2) }},
		{"Dsymm", "lda < n when right", func() { Dsymm(Right, Lower, 2, 4, 1, a, 2, a, 4, 1, cm, 4) }},
		{"Dsymm", "ldb < n", func() { Dsymm(Left, Lower, 2, 4, 1, a, 2, a, 2, 1, cm, 4) }},
		{"Dsymm", "ldc < n", func() { Dsymm(Left, Lower, 2, 4, 1, a, 2, a, 4, 1, cm, 2) }},
		{"Dsymm", "b too short", func() { Dsymm(Left, Lower, 4, 4, 1, a, 4, a[:12], 4, 1, cm, 4) }},

		{"Dtrsm", "side not L or R", func() { Dtrsm('X', Lower, NoTrans, NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "uplo not U or L", func() { Dtrsm(Left, 'X', NoTrans, NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "tA not N or T", func() { Dtrsm(Left, Lower, 'C', NonUnit, 2, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "diag not N or U", func() { Dtrsm(Left, Lower, NoTrans, 'X', 2, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "m < 0", func() { Dtrsm(Left, Lower, NoTrans, NonUnit, -1, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "n < 0", func() { Dtrsm(Left, Lower, NoTrans, NonUnit, 2, -1, 1, a, 2, cm, 2) }},
		{"Dtrsm", "lda < m when left", func() { Dtrsm(Left, Lower, NoTrans, NonUnit, 4, 2, 1, a, 2, cm, 2) }},
		{"Dtrsm", "lda < n when right", func() { Dtrsm(Right, Lower, NoTrans, NonUnit, 2, 4, 1, a, 2, cm, 4) }},
		{"Dtrsm", "ldb < n", func() { Dtrsm(Left, Lower, NoTrans, NonUnit, 2, 4, 1, a, 2, cm, 2) }},
		{"Dtrsm", "b too short", func() { Dtrsm(Left, Lower, NoTrans, NonUnit, 4, 4, 1, a, 4, cm[:12], 4) }},
	}
	for _, tc := range cases {
		mustPanic(t, tc.routine+" "+tc.what, tc.f)
	}
	for _, r := range []string{"Dsyrk", "Dsymm", "Dtrsm"} {
		p4(r).extra("argpanic")
	}
}

// ------------------------------------------------------------------ non-finite

// TestL3NonFinite64 pins the shortcut rules and the hazard the zero-padded64 edge
// strategy carries into the masked routines.
//
// A tile that straddles the diagonal is computed in full and half-discarded, and a
// fringe tile's panels are zero-padded64, so an infinity in real data meets a zero
// and produces a NaN in the scratch tile. internal/block's copy-back is what keeps
// that NaN out of both C's other triangle and its padding; this is the test that
// says so, at a size where the two effects coincide.
func TestL3NonFinite64(t *testing.T) {
	inf := math.Inf(1)
	for _, rn := range l3Runners64() {
		t.Run(rn.name, func(t *testing.T) {
			const n, k = 5, 3
			r := rand.New(rand.NewSource(p4Seed))

			for _, lower := range []bool{false, true} {
				a := randMatrix64(r, n*k)
				a[0] = inf
				cm := randMatrix64(r, n*n)
				c0 := append([]float64(nil), cm...)
				rn.syrk(lower, false, n, k, 1, a, k, 1, cm, n)
				want, scale := oracle.Dsyrk(lower, false, n, k, 1, a, k, 1, c0, n)
				for i := 0; i < n; i++ {
					for j := 0; j < n; j++ {
						if !oracle.InTriangle(lower, i, j) {
							if cm[i*n+j] != c0[i*n+j] {
								t.Fatalf("Dsyrk[%s] inf uplo=%s: the unreferenced triangle at [%d,%d] changed to %v",
									rn.name, uploStr(lower), i, j, cm[i*n+j])
							}
							continue
						}
						checkScalarD(t, fmt.Sprintf("Dsyrk[%s] inf uplo=%s [%d,%d]", rn.name, uploStr(lower), i, j),
							k+1, cm[i*n+j], want[i*n+j], scale[i*n+j])
					}
				}
			}

			// Dsymm with an infinity in B, and Dtrsm with one in B: both compared
			// against the oracle rather than against an expectation, so that
			// whatever the arithmetic does with it has to be the same on both sides.
			as := symMatrix64(r, n, n, true)
			bm := randMatrix64(r, n*n)
			bm[n-1] = inf
			cm := randMatrix64(r, n*n)
			c0 := append([]float64(nil), cm...)
			rn.symm(true, true, n, n, 1, as, n, bm, n, 1, cm, n)
			want, scale := oracle.Dsymm(true, true, n, n, 1, as, n, bm, n, 1, c0, n)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					checkScalarD(t, fmt.Sprintf("Dsymm[%s] inf [%d,%d]", rn.name, i, j),
						n+1, cm[i*n+j], want[i*n+j], scale[i*n+j])
				}
			}

			at := triMatrix64(r, n, n, true, false)
			bt := randMatrix64(r, n*n)
			bt[0] = inf
			b0 := append([]float64(nil), bt...)
			rn.trsm(true, true, false, false, n, n, 1, at, n, bt, n)
			want3, scale3 := oracle.Dtrsm(true, true, false, false, n, n, 1, at, n, b0, n)
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					checkScalarD(t, fmt.Sprintf("Dtrsm[%s] inf [%d,%d]", rn.name, i, j),
						n+1, bt[i*n+j], want3[i*n+j], scale3[i*n+j])
				}
			}
		})
	}
	for _, r := range []string{"Dsyrk", "Dsymm", "Dtrsm"} {
		p4(r).extra("nonfinite")
	}
}

// The mask's three range predicates are tested where they live, against the
// element-wise definition they are derived from: internal/block/tri_test.go.
