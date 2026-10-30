// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/block64"
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
	"github.com/scttfrdmn/keel/internal/pack64"
)

// Dgemm over DESIGN.md §4/P3's sweep (#193): gemm_test.go at float64, so gate-p3's
// KEEL_ELEM=f64 arm audits the same extent from the keel-dgemm-* markers that it
// audits from keel-sgemm-* at float32.
//
// The extent is SHARED, not copied: sweepSizes, sweepExactMax, sweepSampleN,
// sweepSeed and sweepTrans are gemm_test.go's, so float64 cannot quietly sweep less.
// Only the element-typed lattice is its own (sweepAlphas64, sweepBetas64).
//
// Each comparison runs both float64 oracle arms of DESIGN.md §5 rule 1: the
// definition-order reference (Dgemm, DgemmEntry) under PeerTolerance64, and the
// double-double one (GemmDD, GemmEntryDD) under Tolerance64.
//
// Produced from gemm_test.go by a one-off rename and kept as ordinary test code;
// a float32 sweep change must be mirrored here by hand.

var (
	sweepAlphas64 = []float64{0, 1, -0.75}
	sweepBetas64  = []float64{0, 1, 0.5}
)

type gemmCombo64 struct {
	gemmTrans
	alpha, beta float64
}

func (c gemmCombo64) String() string {
	return fmt.Sprintf("%s/alpha=%g/beta=%g", c.name, c.alpha, c.beta)
}

// gemmRunner64 is one way to compute an SGEMM: the public entry point, or a
// specific microkernel driven through the blocking nest directly.
//
// Both are needed and neither substitutes for the other. The public path is what
// callers get, including argument validation and dispatch; the per-kernel paths
// are what make this a differential test — on an AVX-512 host the same sweep runs
// on both shipped vector shapes and on both scalar references, so a shim bug shows
// up as a disagreement even where the oracle's tolerance would have absorbed it.
type gemmRunner64 struct {
	name    string // subtest name
	backend string // for the gate's backends-exercised marker
	run     func(ta, tb bool, m, n, k int, alpha float64, a []float64, lda int,
		b []float64, ldb int, beta float64, c []float64, ldc int)
}

func gemmRunners64() []gemmRunner64 {
	rs := []gemmRunner64{{
		name:    "public",
		backend: ActiveKern64Backend(),
		run: func(ta, tb bool, m, n, k int, alpha float64, a []float64, lda int,
			b []float64, ldb int, beta float64, c []float64, ldc int) {
			Dgemm(transFlag(ta), transFlag(tb), m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		},
	}}
	for _, kn := range kern.Kernels64() {
		kn := kn
		rs = append(rs, gemmRunner64{
			name:    kn.ID(),
			backend: kn.Name,
			run: func(ta, tb bool, m, n, k int, alpha float64, a []float64, lda int,
				b []float64, ldb int, beta float64, c []float64, ldc int) {
				block64.Gemm(kn, ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
			},
		})
	}
	return rs
}

// Coverage recorded for the markers TestMain prints. Written from the sequential
// sweep only, so no synchronization is needed; a map that a parallel subtest
// wrote would need it, and the gate would rather have a slow marker than a racy
// one.
var (
	gemmSizesRun64    = map[int]bool{}
	gemmBackendsRun64 = map[string]bool{}
	gemmVerifyMode64  = map[int]string{}
	gemmExtras64      []string
	packCombos64      int
)

func gemmExtra64(name string) { gemmExtras64 = append(gemmExtras64, name) }

// ------------------------------------------------------------------- the sweep

func TestDgemmSweep(t *testing.T) {
	for _, rn := range gemmRunners64() {
		gemmBackendsRun64[rn.backend] = true
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range sweepSizes {
				gemmSizesRun64[sz] = true
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range sweepCombos64(sz) {
						t.Run(cb.String(), func(t *testing.T) {
							sweepCase64(t, rn, sz, cb)
						})
					}
				})
			}
		})
	}
}

// sweepCombos64 is the full lattice up to sweepExactMax and one rotating
// combination above it.
//
// The reason is cost, and it is worth stating rather than hiding: 36 combinations
// at 2048³ is 620 GFLOP per runner, most of it through a scalar reference kernel,
// which would put a single gate run into the tens of minutes for no new
// information. What the large sizes are here to exercise is the loop nest — more
// than one KC block (2048 > 384), more than one MC block (2048 > 144) — and one
// combination per size does that. The lattice's job is edge and shortcut
// coverage, and it runs in full at every size where a fringe tile can occur.
//
// The rotation means the three large sizes do not all pick the same transpose, so
// the multi-block path is exercised with both a copying and a transposing pack.
func sweepCombos64(sz int) []gemmCombo64 {
	if sz <= sweepExactMax {
		out := make([]gemmCombo64, 0, len(sweepTrans)*len(sweepAlphas64)*len(sweepBetas64))
		for _, tr := range sweepTrans {
			for _, al := range sweepAlphas64 {
				for _, be := range sweepBetas64 {
					out = append(out, gemmCombo64{tr, al, be})
				}
			}
		}
		return out
	}
	rank := 0
	for _, s := range sweepSizes {
		if s > sweepExactMax && s < sz {
			rank++
		}
	}
	return []gemmCombo64{{sweepTrans[rank%len(sweepTrans)], -0.75, 0.5}}
}

// sweepCase64 runs one (runner, size, combination) and verifies it.
//
// The matrices are square at the sweep sizes because that is what DESIGN.md's
// sweep asks for; TestDgemmShapes covers m, n and k moving independently around
// the tile boundaries, which is where the fringe logic actually lives.
func sweepCase64(t *testing.T, rn gemmRunner64, sz int, cb gemmCombo64) {
	m, n, k := sz, sz, sz
	r := rand.New(rand.NewSource(sweepSeed + int64(sz)))
	a := randMatrix64(r, m*k)
	b := randMatrix64(r, k*n)
	c := randMatrix64(r, m*n)
	c0 := append([]float64(nil), c...)

	lda, ldb := k, n
	if cb.ta {
		lda = m
	}
	if cb.tb {
		ldb = k
	}
	rn.run(cb.ta, cb.tb, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c, n)

	what := fmt.Sprintf("Dgemm[%s] %s n=%d", rn.name, cb, sz)
	if sz <= sweepExactMax {
		gemmVerifyMode64[sz] = fmt.Sprintf("mode=exact n=%d", m*n)
		verifyExact64(t, what, cb, m, n, k, a, lda, b, ldb, c0, c, n)
		return
	}
	gemmVerifyMode64[sz] = fmt.Sprintf("mode=sampled n=%d seed=%#x", sweepSampleN, sweepSeed)
	verifySampled64(t, what, cb, m, n, k, a, lda, b, ldb, c0, c, n, sz)
}

func verifyExact64(t *testing.T, what string, cb gemmCombo64, m, n, k int,
	a []float64, lda int, b []float64, ldb int, c0, got []float64, ldc int) {
	t.Helper()
	want, scale := oracle.Dgemm(cb.ta, cb.tb, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
	wdd, sdd := oracle.GemmDD(cb.ta, cb.tb, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			check64(t, fmt.Sprintf("%s [%d,%d] ref", what, i, j), k+1,
				got[i*ldc+j], want[i*n+j], scale[i*n+j], oracle.PeerTolerance64)
			check64(t, fmt.Sprintf("%s [%d,%d] dd", what, i, j), k+1,
				got[i*ldc+j], wdd[i*n+j], sdd[i*n+j], oracle.Tolerance64)
			if t.Failed() {
				return // one wrong entry is the whole story; 4M more is noise
			}
		}
	}
}

// verifySampled64 checks sweepSampleN entries drawn from a seeded generator, plus
// the four corners unconditionally.
//
// The corners are not superstition: index arithmetic errors concentrate at the
// last row and column, which is exactly where a uniform sample of 256 out of
// 4.2M entries is unlikely to look.
func verifySampled64(t *testing.T, what string, cb gemmCombo64, m, n, k int,
	a []float64, lda int, b []float64, ldb int, c0, got []float64, ldc, sz int) {
	t.Helper()
	type ij struct{ i, j int }
	pts := []ij{{0, 0}, {0, n - 1}, {m - 1, 0}, {m - 1, n - 1}}
	rs := rand.New(rand.NewSource(sweepSeed + int64(sz)))
	for len(pts) < sweepSampleN {
		pts = append(pts, ij{rs.Intn(m), rs.Intn(n)})
	}
	for _, p := range pts {
		want, scale := oracle.DgemmEntry(cb.ta, cb.tb, p.i, p.j, k,
			cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
		check64(t, fmt.Sprintf("%s [%d,%d] ref", what, p.i, p.j), k+1,
			got[p.i*ldc+p.j], want, scale, oracle.PeerTolerance64)
		wdd, sdd := oracle.GemmEntryDD(cb.ta, cb.tb, p.i, p.j, k,
			cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
		check64(t, fmt.Sprintf("%s [%d,%d] dd", what, p.i, p.j), k+1,
			got[p.i*ldc+p.j], wdd, sdd, oracle.Tolerance64)
		if t.Failed() {
			return
		}
	}
}

func randMatrix64(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = r.Float64()*2 - 1
	}
	return v
}

// ------------------------------------------------- shapes, strides, edge cases

// TestDgemmShapes moves m, n and k independently through the neighbourhood of
// every shipped tile bound.
//
// The square sweep above cannot distinguish an m-edge bug from an n-edge bug,
// because it never has one without the other. This does: the set includes each
// shipped MR and NR, one below and one above, so every combination of "full tile"
// and "fringe tile" in both directions occurs, with a k that is and is not a
// multiple of the kernels' unroll.
func TestDgemmShapes(t *testing.T) {
	dims := []int{1, 2, 3, 4, 5, 7, 8, 31, 32, 33, 64, 65}
	kdims := []int{1, 2, 3, 4, 5, 7, 8, 33}
	cb := gemmCombo64{sweepTrans[0], -0.75, 0.5}
	for _, rn := range gemmRunners64() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(sweepSeed))
			for _, m := range dims {
				for _, n := range dims {
					for _, k := range kdims {
						a := randMatrix64(r, m*k)
						b := randMatrix64(r, k*n)
						c := randMatrix64(r, m*n)
						c0 := append([]float64(nil), c...)
						rn.run(false, false, m, n, k, cb.alpha, a, k, b, n, cb.beta, c, n)
						verifyExact64(t, fmt.Sprintf("Dgemm[%s] %dx%dx%d", rn.name, m, n, k),
							cb, m, n, k, a, k, b, n, c0, c, n)
						if t.Failed() {
							return
						}
					}
				}
			}
		})
	}
	gemmExtra64("rectangular")
}

// TestDgemmLdPad passes submatrix views: every ld wider than the matrix it
// describes, with poisonF64 in the padding.
//
// This is how a caller hands keel a block of a larger matrix, and it is the case
// where an off-by-one in the packing or in the C update writes into somebody
// else's data rather than off the end of a slice — which no bounds check would
// catch and no square, tightly-packed test would notice.
func TestDgemmLdPad(t *testing.T) {
	const pad = 3
	cb := gemmCombo64{sweepTrans[0], -0.75, 0.5}
	for _, rn := range gemmRunners64() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(sweepSeed))
			for _, sz := range []int{1, 5, 17, 33, 65} {
				m, n, k := sz, sz, sz
				lda, ldb, ldc := k+pad, n+pad, n+pad
				a := padded64(r, m, k, lda)
				b := padded64(r, k, n, ldb)
				c := padded64(r, m, n, ldc)
				c0 := append([]float64(nil), c...)
				rn.run(false, false, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c, ldc)
				verifyExact64(t, fmt.Sprintf("Dgemm[%s] ldpad n=%d", rn.name, sz),
					cb, m, n, k, a, lda, b, ldb, c0, c, ldc)
				// The padding of C must be exactly as it was, and A's and B's
				// too: nothing here writes to them at all.
				checkPadding64(t, fmt.Sprintf("%s a n=%d", rn.name, sz), a, m, k, lda, c0[:0])
				checkPadding64(t, fmt.Sprintf("%s b n=%d", rn.name, sz), b, k, n, ldb, c0[:0])
				checkPadding64(t, fmt.Sprintf("%s c n=%d", rn.name, sz), c, m, n, ldc, c0)
				if t.Failed() {
					return
				}
			}
		})
	}
	gemmExtra64("ldpad")
}

// padded64 builds a rows×cols matrix with row stride ld, poisonF64 in the gaps.
func padded64(r *rand.Rand, rows, cols, ld int) []float64 {
	v := make([]float64, (rows-1)*ld+cols)
	for i := range v {
		v[i] = poisonF64
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v[i*ld+j] = r.Float64()*2 - 1
		}
	}
	return v
}

// checkPadding64 verifies the inter-row gaps still hold poisonF64. When orig is
// non-empty it is the pre-call copy, and the gaps are compared against it
// instead — same check, but it also catches a write that happened to store the
// poisonF64 value.
func checkPadding64(t *testing.T, what string, v []float64, rows, cols, ld int, orig []float64) {
	t.Helper()
	for i := 0; i < rows-1; i++ {
		for j := cols; j < ld; j++ {
			want := poisonF64
			if len(orig) > i*ld+j {
				want = orig[i*ld+j]
			}
			if v[i*ld+j] != want {
				t.Errorf("%s: padding at [%d,%d] overwritten: %v, want %v", what, i, j, v[i*ld+j], want)
				return
			}
		}
	}
}

// TestDgemmZeroDim covers every dimension being zero.
//
// k == 0 is the empty product and must still apply beta — the one case where
// "there is nothing to do" is wrong. m or n == 0 must touch nothing at all,
// because a caller looping over a partitioned matrix will hand keel an empty
// block and expects it to be free rather than fatal.
//
// A matrix whose declared shape is empty may be nil; one whose declared shape is
// not empty must still be long enough, even on a call that will not read it. That
// asymmetry is deliberate: `nil` for a 4×4 B is a caller bug that happens not to
// matter today, and keel says so rather than waiting for the day it does.
func TestDgemmZeroDim(t *testing.T) {
	c := []float64{1, 2, 3, 4}
	// a is 2×0 and b is 0×2, so both may be nil; ldb must still hold a row of B.
	Dgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 1, c, 2)
	if want := []float64{1, 2, 3, 4}; !equalF64(c, want) {
		t.Errorf("k=0 beta=1: c = %v, want %v (beta=1 must not change C)", c, want)
	}
	Dgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 2, c, 2)
	if want := []float64{2, 4, 6, 8}; !equalF64(c, want) {
		t.Errorf("k=0 beta=2: c = %v, want %v (the empty product still scales C)", c, want)
	}
	Dgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 0, c, 2)
	if want := []float64{0, 0, 0, 0}; !equalF64(c, want) {
		t.Errorf("k=0 beta=0: c = %v, want %v", c, want)
	}

	// m == 0 and n == 0: nothing is read or written. The operands that are not
	// themselves empty are passed at full length, per the rule above.
	full := make([]float64, 16)
	Dgemm(NoTrans, NoTrans, 0, 0, 0, 1, nil, 1, nil, 1, 3, nil, 1)
	Dgemm(NoTrans, NoTrans, 0, 4, 4, 1, nil, 4, full, 4, 3, nil, 4)
	Dgemm(NoTrans, NoTrans, 4, 0, 4, 1, full, 4, nil, 1, 3, nil, 1)
	Dgemm(Trans, Trans, 0, 4, 4, 1, nil, 1, full, 4, 3, nil, 4)
	gemmExtra64("zerodim")
}

// TestDgemmArgPanic: every argument error panics rather than computing something
// plausible. Reference BLAS reports these through XERBLA and returns; keel has no
// XERBLA, and a caller who passed the wrong ld has a bug that a silent partial
// result would hide.
func TestDgemmArgPanic(t *testing.T) {
	a := make([]float64, 16)
	c := make([]float64, 16)
	cases := []struct {
		what string
		f    func()
	}{
		{"tA not N or T", func() { Dgemm('C', NoTrans, 2, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"tB not N or T", func() { Dgemm(NoTrans, 'C', 2, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"m < 0", func() { Dgemm(NoTrans, NoTrans, -1, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"n < 0", func() { Dgemm(NoTrans, NoTrans, 2, -1, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"k < 0", func() { Dgemm(NoTrans, NoTrans, 2, 2, -1, 1, a, 2, a, 2, 1, c, 2) }},
		{"lda < k", func() { Dgemm(NoTrans, NoTrans, 2, 2, 4, 1, a, 2, a, 2, 1, c, 2) }},
		{"lda < m when transA", func() { Dgemm(Trans, NoTrans, 4, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"ldb < n", func() { Dgemm(NoTrans, NoTrans, 2, 4, 2, 1, a, 2, a, 2, 1, c, 4) }},
		{"ldb < k when transB", func() { Dgemm(NoTrans, Trans, 2, 2, 4, 1, a, 4, a, 2, 1, c, 2) }},
		{"ldc < n", func() { Dgemm(NoTrans, NoTrans, 2, 4, 2, 1, a, 2, a, 4, 1, c, 2) }},
		{"ldc == 0 with empty n", func() { Dgemm(NoTrans, NoTrans, 2, 0, 2, 1, a, 2, a, 1, 1, c, 0) }},
		{"a too short", func() { Dgemm(NoTrans, NoTrans, 4, 4, 4, 1, a[:12], 4, a, 4, 1, c, 4) }},
		{"b too short", func() { Dgemm(NoTrans, NoTrans, 4, 4, 4, 1, a, 4, a[:12], 4, 1, c, 4) }},
		{"c too short", func() { Dgemm(NoTrans, NoTrans, 4, 4, 4, 1, a, 4, a, 4, 1, c[:12], 4) }},
	}
	for _, tc := range cases {
		mustPanic(t, "Dgemm "+tc.what, tc.f)
	}
	gemmExtra64("argpanic")
}

// TestDgemmNonFinite pins the two shortcut rules and the one hazard the
// zero-padded64 edge strategy introduces.
//
// alpha == 0 must not multiply, so a NaN in A cannot reach C; beta == 0 must not
// read C, so a NaN already in C cannot survive. And on a fringe tile the packed
// panels carry zeros in the padding, so an infinity in real data meets a zero and
// produces a NaN in the scratch tile — internal/block's temp-tile copy-back is
// what keeps that NaN out of C, and this is the test that says so.
func TestDgemmNonFinite(t *testing.T) {
	nan := math.NaN()
	inf := math.Inf(1)

	// alpha = 0 with a poisoned A.
	a := []float64{nan, nan, nan, nan}
	c := []float64{1, 2, 3, 4}
	Dgemm(NoTrans, NoTrans, 2, 2, 2, 0, a, 2, a, 2, 2, c, 2)
	if want := []float64{2, 4, 6, 8}; !equalF64(c, want) {
		t.Errorf("alpha=0 with NaN in A: c = %v, want %v (A must not be read)", c, want)
	}

	// beta = 0 with a poisoned C.
	a = []float64{1, 0, 0, 1}
	c = []float64{nan, nan, nan, nan}
	Dgemm(NoTrans, NoTrans, 2, 2, 2, 1, a, 2, a, 2, 0, c, 2)
	if want := []float64{1, 0, 0, 1}; !equalF64(c, want) {
		t.Errorf("beta=0 with NaN in C: c = %v, want %v (C must not be read)", c, want)
	}

	// A fringe tile in both directions, with an infinity in the data. Every
	// runner, because the padding lives in the packed panels every kernel reads.
	cb := gemmCombo64{sweepTrans[0], 1, 1}
	for _, rn := range gemmRunners64() {
		m, n, k := 5, 5, 3
		r := rand.New(rand.NewSource(sweepSeed))
		av := randMatrix64(r, m*k)
		bv := randMatrix64(r, k*n)
		cv := randMatrix64(r, m*n)
		av[0] = inf
		bv[k*n-1] = inf
		c0 := append([]float64(nil), cv...)
		rn.run(false, false, m, n, k, cb.alpha, av, k, bv, n, cb.beta, cv, n)
		verifyExact64(t, "Dgemm["+rn.name+"] fringe inf", cb, m, n, k, av, k, bv, n, c0, cv, n)
	}
	gemmExtra64("nonfinite")
}

func equalF64(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------- the packing

// TestPack64Panels holds internal/pack to an independent statement of the layout.
//
// The reference below is the definition transcribed one element at a time —
// slower than the thing it checks by any measure, and written from the protocol
// in internal/kern's package doc rather than from pack's implementation. The two
// disagree if either the copying or the transposing direction is wrong, if a
// padding element is left stale, or if alpha is folded into the wrong operand.
//
// It lives in this package, not in internal/pack, for a mechanical reason worth
// recording: the gate audits coverage markers from the sweep log of the one test
// binary it ships to each host, so a marker printed by a second binary would not
// be seen. The packing is exercised here and unit-tested here.
func TestPack64Panels(t *testing.T) {
	r := rand.New(rand.NewSource(sweepSeed))
	blks := []int{2, 4, 32}
	counts := []int{0, 1, 3, 4, 5, 8, 33}
	kcs := []int{0, 1, 3, 4, 8}
	alphas := []float64{1, -0.75}
	combos := 0

	for _, blk := range blks {
		for _, count := range counts {
			for _, kc := range kcs {
				for _, trans := range []bool{false, true} {
					for _, alpha := range alphas {
						// Offsets deliberately nonzero: the block being packed is
						// almost never at the origin of the caller's matrix, and an
						// implementation that ignored i0/p0 would pass at (0,0).
						// Every source is ld-padded64 with poisonF64 for the same reason
						// TestDgemmLdPad exists.
						const off = 2
						// Two source shapes, because which one a call reads depends
						// on the operand and the transpose: "bc" is
						// blocked-axis-by-depth, "cb" is its transpose.
						bcLd := kc + off + 1
						cbLd := count + off + 1
						bc := padded64(r, count+off, kc+off, bcLd)
						cb := padded64(r, kc+off, count+off, cbLd)

						aSrc, aLd := bc, bcLd
						if trans {
							aSrc, aLd = cb, cbLd
						}
						gotA := filled64(pack64.ALen(blk, count, kc))
						pack64.APanels(gotA, blk, alpha, aSrc, aLd, trans, off, count, off, kc)
						wantA := refPanels64(blk, alpha, aSrc, aLd, !trans, off, count, off, kc)
						if !equalF64(gotA, wantA) {
							t.Fatalf("APanels blk=%d count=%d kc=%d trans=%v alpha=%g:\n got %v\nwant %v",
								blk, count, kc, trans, alpha, gotA, wantA)
						}

						bSrc, bLd := cb, cbLd
						if trans {
							bSrc, bLd = bc, bcLd
						}
						gotB := filled64(pack64.BLen(blk, count, kc))
						pack64.BPanels(gotB, blk, bSrc, bLd, trans, off, kc, off, count)
						wantB := refPanels64(blk, 1, bSrc, bLd, trans, off, count, off, kc)
						if !equalF64(gotB, wantB) {
							t.Fatalf("BPanels blk=%d count=%d kc=%d trans=%v:\n got %v\nwant %v",
								blk, count, kc, trans, gotB, wantB)
						}
						combos++
					}
				}
			}
		}
	}
	packCombos64 = combos
}

// refPanels64 is the packed layout stated directly: element x of block ib at depth
// p goes to dst[ib*blk*kc + p*blk + x], scaled by alpha, zero where x is past the
// end. depthContig says the source's contiguous index is the depth index.
func refPanels64(blk int, alpha float64, src []float64, ld int, depthContig bool, b0, count, p0, kc int) []float64 {
	nb := (count + blk - 1) / blk
	dst := make([]float64, nb*blk*kc)
	for ib := 0; ib < nb; ib++ {
		for p := 0; p < kc; p++ {
			for x := 0; x < blk; x++ {
				var v float64
				if idx := ib*blk + x; idx < count {
					if depthContig {
						v = alpha * src[(b0+idx)*ld+p0+p]
					} else {
						v = alpha * src[(p0+p)*ld+b0+idx]
					}
				}
				dst[ib*blk*kc+p*blk+x] = v
			}
		}
	}
	return dst
}

// filled64 returns a buffer of poisonF64, so that a padding element pack forgets to
// write is a failure rather than a lucky zero.
func filled64(n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = poisonF64
	}
	return v
}

// ------------------------------------------------------------------- markers

// printDgemmMarkers emits what scripts/gate-p3.sh parses. Called from TestMain in
// l1_test.go — one TestMain per package — and unconditionally, so a failing run
// still reports what it covered.
func printDgemmMarkers() {
	kc, mc, nc := block64.Params(activeKern64)
	fmt.Printf("keel-dgemm-config: mr=%d nr=%d kc=%d mc=%d nc=%d edge=%s beta-variants=%d pack=%s\n",
		activeKern64.MR, activeKern64.NR, kc, mc, nc, block64.EdgeStrategy, block64.BetaVariants,
		"go-copy+scalar-transpose")
	fmt.Println("keel-dgemm-active:", ActiveKern64Tile()+"/"+ActiveKern64Backend())
	// The shape is per-host now (issue #24), so the correctness run reports the
	// classification it was chosen under too. It is the same marker the benchmark
	// prints, from a different binary: if the two disagree the gate is comparing
	// a correctness result on one shape against a rate on another.
	fmt.Println("keel-dgemm-class:", ActiveKernClass(),
		"insns-per-fma="+strconv.FormatFloat(ActiveKern64InsnsPerFMA(), 'f', 3, 64),
		"("+ActiveKernClassEvidence()+")")
	fmt.Println("keel-dgemm-available:", strings.Join(AvailableKernels64(), " "))

	var sizes []string
	for _, sz := range sweepSizes {
		if gemmSizesRun64[sz] {
			sizes = append(sizes, strconv.Itoa(sz))
		}
	}
	fmt.Println("keel-dgemm-sizes-exercised:", strings.Join(sizes, " "))

	fmt.Printf("keel-dgemm-combos-exercised: trans=%s alpha=%s beta=%s combos=%d\n",
		joinTrans(sweepTrans), joinF64(sweepAlphas64), joinF64(sweepBetas64),
		len(sweepTrans)*len(sweepAlphas64)*len(sweepBetas64))

	for _, sz := range sweepSizes {
		if mode, ok := gemmVerifyMode64[sz]; ok {
			fmt.Printf("keel-dgemm-verify: size=%d %s\n", sz, mode)
		}
	}

	var backends []string
	for name := range gemmBackendsRun64 {
		backends = append(backends, name)
	}
	sort.Strings(backends)
	fmt.Println("keel-dgemm-backends-exercised:", strings.Join(backends, " "))
	fmt.Println("keel-dgemm-extra-exercised:", strings.Join(gemmExtras64, " "))
	fmt.Printf("keel-pack64-combos-exercised: operands=A,B trans=false,true alpha=1,general "+
		"ragged=full,partial combos=%d\n", packCombos64)
}

// poisonF64 is gemm_test.go's finite poison: a NaN would fail every equality check.
const poisonF64 = float64(-7777)

func joinF64(vs []float64) string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = strconv.FormatFloat(float64(v), 'g', -1, 64)
	}
	return strings.Join(out, ",")
}

// checkScalarD is checkScalar at float64, against a definition-order float64
// reference: PeerTolerance64, the bound between two float64 computations. The P4
// sweep ports (gemv64_test.go, tri64_test.go) call it.
func checkScalarD(t *testing.T, what string, n int, got, want, scale float64) {
	t.Helper()
	check64(t, what, n, got, want, scale, oracle.PeerTolerance64)
}

// spreadFinite64 is spread at float64 with the finite poison, because the P4 ports
// compare stride gaps with != and a NaN gap would never equal itself.
func spreadFinite64(v []float64, inc int) []float64 {
	n := len(v)
	if n == 0 {
		return nil
	}
	span := max(inc, -inc)
	out := make([]float64, (n-1)*span+1)
	for i := range out {
		out[i] = poisonF64
	}
	base := offset(n, inc)
	for j := 0; j < n; j++ {
		out[base+j*inc] = v[j]
	}
	return out
}
