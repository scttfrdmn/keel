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

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/oracle"
	"github.com/scttfrdmn/keel/internal/pack"
)

// Sgemm against the float64 oracle, over DESIGN.md §4/P3's sweep.
//
// The sizes, the exact/sampled boundary and the sample floor are duplicated in
// scripts/gate-p3.sh, which parses the markers printed at the end of the run and
// fails if the coverage it finds is smaller than the coverage it demands. The
// duplication is deliberate: a test that shrank its own sweep would otherwise
// still report "ok", and the gate's first criterion is that the extent of this
// sweep is enforced rather than trusted.
var sweepSizes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 63, 64, 65, 500, 1000, 2048}

const (
	// sweepExactMax is the largest size compared entry by entry. Above it the
	// oracle costs more than the routine under test (see oracle.GemmEntry), so
	// the comparison moves to a seeded sample.
	sweepExactMax = 65
	// sweepSampleN entries per case above sweepExactMax. The gate demands at
	// least 256; more is allowed, fewer is a failure.
	sweepSampleN = 256
	// sweepSeed seeds every sampler and every data fill, so a failure at any size
	// replays exactly. Printed in the keel-sgemm-verify markers.
	sweepSeed = 0x6b33
)

// The alpha and beta lattices each cover the two special-cased values and one
// general value. Only the general value exercises the multiply at all: a lattice
// of {0, 1} would test every shortcut and no arithmetic.
var (
	sweepAlphas = []float32{0, 1, -0.75}
	sweepBetas  = []float32{0, 1, 0.5}
	sweepTrans  = []gemmTrans{
		{"NN", false, false},
		{"NT", false, true},
		{"TN", true, false},
		{"TT", true, true},
	}
)

type gemmTrans struct {
	name   string
	ta, tb bool
}

type gemmCombo struct {
	gemmTrans
	alpha, beta float32
}

func (c gemmCombo) String() string {
	return fmt.Sprintf("%s/alpha=%g/beta=%g", c.name, c.alpha, c.beta)
}

// gemmRunner is one way to compute an SGEMM: the public entry point, or a
// specific microkernel driven through the blocking nest directly.
//
// Both are needed and neither substitutes for the other. The public path is what
// callers get, including argument validation and dispatch; the per-kernel paths
// are what make this a differential test — on an AVX-512 host the same sweep runs
// on both shipped vector shapes and on both scalar references, so a shim bug shows
// up as a disagreement even where the oracle's tolerance would have absorbed it.
type gemmRunner struct {
	name    string // subtest name
	backend string // for the gate's backends-exercised marker
	run     func(ta, tb bool, m, n, k int, alpha float32, a []float32, lda int,
		b []float32, ldb int, beta float32, c []float32, ldc int)
}

func gemmRunners() []gemmRunner {
	rs := []gemmRunner{{
		name:    "public",
		backend: ActiveKernBackend(),
		run: func(ta, tb bool, m, n, k int, alpha float32, a []float32, lda int,
			b []float32, ldb int, beta float32, c []float32, ldc int) {
			Sgemm(transFlag(ta), transFlag(tb), m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
		},
	}}
	for _, kn := range kern.Kernels() {
		kn := kn
		rs = append(rs, gemmRunner{
			name:    kn.ID(),
			backend: kn.Name,
			run: func(ta, tb bool, m, n, k int, alpha float32, a []float32, lda int,
				b []float32, ldb int, beta float32, c []float32, ldc int) {
				block.Gemm(kn, ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
			},
		})
	}
	return rs
}

func transFlag(t bool) Transpose {
	if t {
		return Trans
	}
	return NoTrans
}

// Coverage recorded for the markers TestMain prints. Written from the sequential
// sweep only, so no synchronization is needed; a map that a parallel subtest
// wrote would need it, and the gate would rather have a slow marker than a racy
// one.
var (
	gemmSizesRun    = map[int]bool{}
	gemmBackendsRun = map[string]bool{}
	gemmVerifyMode  = map[int]string{}
	gemmExtras      []string
	packCombos      int
)

func gemmExtra(name string) { gemmExtras = append(gemmExtras, name) }

// ------------------------------------------------------------------- the sweep

func TestSgemmSweep(t *testing.T) {
	for _, rn := range gemmRunners() {
		gemmBackendsRun[rn.backend] = true
		t.Run(rn.name, func(t *testing.T) {
			for _, sz := range sweepSizes {
				gemmSizesRun[sz] = true
				t.Run("n="+strconv.Itoa(sz), func(t *testing.T) {
					for _, cb := range sweepCombos(sz) {
						t.Run(cb.String(), func(t *testing.T) {
							sweepCase(t, rn, sz, cb)
						})
					}
				})
			}
		})
	}
}

// sweepCombos is the full lattice up to sweepExactMax and one rotating
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
func sweepCombos(sz int) []gemmCombo {
	if sz <= sweepExactMax {
		out := make([]gemmCombo, 0, len(sweepTrans)*len(sweepAlphas)*len(sweepBetas))
		for _, tr := range sweepTrans {
			for _, al := range sweepAlphas {
				for _, be := range sweepBetas {
					out = append(out, gemmCombo{tr, al, be})
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
	return []gemmCombo{{sweepTrans[rank%len(sweepTrans)], -0.75, 0.5}}
}

// sweepCase runs one (runner, size, combination) and verifies it.
//
// The matrices are square at the sweep sizes because that is what DESIGN.md's
// sweep asks for; TestSgemmShapes covers m, n and k moving independently around
// the tile boundaries, which is where the fringe logic actually lives.
func sweepCase(t *testing.T, rn gemmRunner, sz int, cb gemmCombo) {
	m, n, k := sz, sz, sz
	r := rand.New(rand.NewSource(sweepSeed + int64(sz)))
	a := randMatrix(r, m*k)
	b := randMatrix(r, k*n)
	c := randMatrix(r, m*n)
	c0 := append([]float32(nil), c...)

	lda, ldb := k, n
	if cb.ta {
		lda = m
	}
	if cb.tb {
		ldb = k
	}
	rn.run(cb.ta, cb.tb, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c, n)

	what := fmt.Sprintf("Sgemm[%s] %s n=%d", rn.name, cb, sz)
	if sz <= sweepExactMax {
		gemmVerifyMode[sz] = fmt.Sprintf("mode=exact n=%d", m*n)
		verifyExact(t, what, cb, m, n, k, a, lda, b, ldb, c0, c, n)
		return
	}
	gemmVerifyMode[sz] = fmt.Sprintf("mode=sampled n=%d seed=%#x", sweepSampleN, sweepSeed)
	verifySampled(t, what, cb, m, n, k, a, lda, b, ldb, c0, c, n, sz)
}

func verifyExact(t *testing.T, what string, cb gemmCombo, m, n, k int,
	a []float32, lda int, b []float32, ldb int, c0, got []float32, ldc int) {
	t.Helper()
	want, scale := oracle.Gemm(cb.ta, cb.tb, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, i, j), k+1,
				got[i*ldc+j], want[i*n+j], scale[i*n+j])
			if t.Failed() {
				return // one wrong entry is the whole story; 4M more is noise
			}
		}
	}
}

// verifySampled checks sweepSampleN entries drawn from a seeded generator, plus
// the four corners unconditionally.
//
// The corners are not superstition: index arithmetic errors concentrate at the
// last row and column, which is exactly where a uniform sample of 256 out of
// 4.2M entries is unlikely to look.
func verifySampled(t *testing.T, what string, cb gemmCombo, m, n, k int,
	a []float32, lda int, b []float32, ldb int, c0, got []float32, ldc, sz int) {
	t.Helper()
	type ij struct{ i, j int }
	pts := []ij{{0, 0}, {0, n - 1}, {m - 1, 0}, {m - 1, n - 1}}
	rs := rand.New(rand.NewSource(sweepSeed + int64(sz)))
	for len(pts) < sweepSampleN {
		pts = append(pts, ij{rs.Intn(m), rs.Intn(n)})
	}
	for _, p := range pts {
		want, scale := oracle.GemmEntry(cb.ta, cb.tb, p.i, p.j, k,
			cb.alpha, a, lda, b, ldb, cb.beta, c0, ldc)
		checkScalar(t, fmt.Sprintf("%s [%d,%d]", what, p.i, p.j), k+1,
			got[p.i*ldc+p.j], want, scale)
		if t.Failed() {
			return
		}
	}
}

func randMatrix(r *rand.Rand, n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = r.Float32()*2 - 1
	}
	return v
}

// ------------------------------------------------- shapes, strides, edge cases

// TestSgemmShapes moves m, n and k independently through the neighbourhood of
// every shipped tile bound.
//
// The square sweep above cannot distinguish an m-edge bug from an n-edge bug,
// because it never has one without the other. This does: the set includes each
// shipped MR and NR, one below and one above, so every combination of "full tile"
// and "fringe tile" in both directions occurs, with a k that is and is not a
// multiple of the kernels' unroll.
func TestSgemmShapes(t *testing.T) {
	dims := []int{1, 2, 3, 4, 5, 7, 8, 31, 32, 33, 64, 65}
	kdims := []int{1, 2, 3, 4, 5, 7, 8, 33}
	cb := gemmCombo{sweepTrans[0], -0.75, 0.5}
	for _, rn := range gemmRunners() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(sweepSeed))
			for _, m := range dims {
				for _, n := range dims {
					for _, k := range kdims {
						a := randMatrix(r, m*k)
						b := randMatrix(r, k*n)
						c := randMatrix(r, m*n)
						c0 := append([]float32(nil), c...)
						rn.run(false, false, m, n, k, cb.alpha, a, k, b, n, cb.beta, c, n)
						verifyExact(t, fmt.Sprintf("Sgemm[%s] %dx%dx%d", rn.name, m, n, k),
							cb, m, n, k, a, k, b, n, c0, c, n)
						if t.Failed() {
							return
						}
					}
				}
			}
		})
	}
	gemmExtra("rectangular")
}

// TestSgemmLdPad passes submatrix views: every ld wider than the matrix it
// describes, with poison in the padding.
//
// This is how a caller hands keel a block of a larger matrix, and it is the case
// where an off-by-one in the packing or in the C update writes into somebody
// else's data rather than off the end of a slice — which no bounds check would
// catch and no square, tightly-packed test would notice.
func TestSgemmLdPad(t *testing.T) {
	const pad = 3
	cb := gemmCombo{sweepTrans[0], -0.75, 0.5}
	for _, rn := range gemmRunners() {
		t.Run(rn.name, func(t *testing.T) {
			r := rand.New(rand.NewSource(sweepSeed))
			for _, sz := range []int{1, 5, 17, 33, 65} {
				m, n, k := sz, sz, sz
				lda, ldb, ldc := k+pad, n+pad, n+pad
				a := padded(r, m, k, lda)
				b := padded(r, k, n, ldb)
				c := padded(r, m, n, ldc)
				c0 := append([]float32(nil), c...)
				rn.run(false, false, m, n, k, cb.alpha, a, lda, b, ldb, cb.beta, c, ldc)
				verifyExact(t, fmt.Sprintf("Sgemm[%s] ldpad n=%d", rn.name, sz),
					cb, m, n, k, a, lda, b, ldb, c0, c, ldc)
				// The padding of C must be exactly as it was, and A's and B's
				// too: nothing here writes to them at all.
				checkPadding(t, fmt.Sprintf("%s a n=%d", rn.name, sz), a, m, k, lda, c0[:0])
				checkPadding(t, fmt.Sprintf("%s b n=%d", rn.name, sz), b, k, n, ldb, c0[:0])
				checkPadding(t, fmt.Sprintf("%s c n=%d", rn.name, sz), c, m, n, ldc, c0)
				if t.Failed() {
					return
				}
			}
		})
	}
	gemmExtra("ldpad")
}

// padded builds a rows×cols matrix with row stride ld, poison in the gaps.
func padded(r *rand.Rand, rows, cols, ld int) []float32 {
	v := make([]float32, (rows-1)*ld+cols)
	for i := range v {
		v[i] = poison
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v[i*ld+j] = r.Float32()*2 - 1
		}
	}
	return v
}

// checkPadding verifies the inter-row gaps still hold poison. When orig is
// non-empty it is the pre-call copy, and the gaps are compared against it
// instead — same check, but it also catches a write that happened to store the
// poison value.
func checkPadding(t *testing.T, what string, v []float32, rows, cols, ld int, orig []float32) {
	t.Helper()
	for i := 0; i < rows-1; i++ {
		for j := cols; j < ld; j++ {
			want := poison
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

// TestSgemmZeroDim covers every dimension being zero.
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
func TestSgemmZeroDim(t *testing.T) {
	c := []float32{1, 2, 3, 4}
	// a is 2×0 and b is 0×2, so both may be nil; ldb must still hold a row of B.
	Sgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 1, c, 2)
	if want := []float32{1, 2, 3, 4}; !equalF32(c, want) {
		t.Errorf("k=0 beta=1: c = %v, want %v (beta=1 must not change C)", c, want)
	}
	Sgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 2, c, 2)
	if want := []float32{2, 4, 6, 8}; !equalF32(c, want) {
		t.Errorf("k=0 beta=2: c = %v, want %v (the empty product still scales C)", c, want)
	}
	Sgemm(NoTrans, NoTrans, 2, 2, 0, -0.75, nil, 1, nil, 2, 0, c, 2)
	if want := []float32{0, 0, 0, 0}; !equalF32(c, want) {
		t.Errorf("k=0 beta=0: c = %v, want %v", c, want)
	}

	// m == 0 and n == 0: nothing is read or written. The operands that are not
	// themselves empty are passed at full length, per the rule above.
	full := make([]float32, 16)
	Sgemm(NoTrans, NoTrans, 0, 0, 0, 1, nil, 1, nil, 1, 3, nil, 1)
	Sgemm(NoTrans, NoTrans, 0, 4, 4, 1, nil, 4, full, 4, 3, nil, 4)
	Sgemm(NoTrans, NoTrans, 4, 0, 4, 1, full, 4, nil, 1, 3, nil, 1)
	Sgemm(Trans, Trans, 0, 4, 4, 1, nil, 1, full, 4, 3, nil, 4)
	gemmExtra("zerodim")
}

// TestSgemmArgPanic: every argument error panics rather than computing something
// plausible. Reference BLAS reports these through XERBLA and returns; keel has no
// XERBLA, and a caller who passed the wrong ld has a bug that a silent partial
// result would hide.
func TestSgemmArgPanic(t *testing.T) {
	a := make([]float32, 16)
	c := make([]float32, 16)
	cases := []struct {
		what string
		f    func()
	}{
		{"tA not N or T", func() { Sgemm('C', NoTrans, 2, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"tB not N or T", func() { Sgemm(NoTrans, 'C', 2, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"m < 0", func() { Sgemm(NoTrans, NoTrans, -1, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"n < 0", func() { Sgemm(NoTrans, NoTrans, 2, -1, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"k < 0", func() { Sgemm(NoTrans, NoTrans, 2, 2, -1, 1, a, 2, a, 2, 1, c, 2) }},
		{"lda < k", func() { Sgemm(NoTrans, NoTrans, 2, 2, 4, 1, a, 2, a, 2, 1, c, 2) }},
		{"lda < m when transA", func() { Sgemm(Trans, NoTrans, 4, 2, 2, 1, a, 2, a, 2, 1, c, 2) }},
		{"ldb < n", func() { Sgemm(NoTrans, NoTrans, 2, 4, 2, 1, a, 2, a, 2, 1, c, 4) }},
		{"ldb < k when transB", func() { Sgemm(NoTrans, Trans, 2, 2, 4, 1, a, 4, a, 2, 1, c, 2) }},
		{"ldc < n", func() { Sgemm(NoTrans, NoTrans, 2, 4, 2, 1, a, 2, a, 4, 1, c, 2) }},
		{"ldc == 0 with empty n", func() { Sgemm(NoTrans, NoTrans, 2, 0, 2, 1, a, 2, a, 1, 1, c, 0) }},
		{"a too short", func() { Sgemm(NoTrans, NoTrans, 4, 4, 4, 1, a[:12], 4, a, 4, 1, c, 4) }},
		{"b too short", func() { Sgemm(NoTrans, NoTrans, 4, 4, 4, 1, a, 4, a[:12], 4, 1, c, 4) }},
		{"c too short", func() { Sgemm(NoTrans, NoTrans, 4, 4, 4, 1, a, 4, a, 4, 1, c[:12], 4) }},
	}
	for _, tc := range cases {
		mustPanic(t, "Sgemm "+tc.what, tc.f)
	}
	gemmExtra("argpanic")
}

// TestSgemmNonFinite pins the two shortcut rules and the one hazard the
// zero-padded edge strategy introduces.
//
// alpha == 0 must not multiply, so a NaN in A cannot reach C; beta == 0 must not
// read C, so a NaN already in C cannot survive. And on a fringe tile the packed
// panels carry zeros in the padding, so an infinity in real data meets a zero and
// produces a NaN in the scratch tile — internal/block's temp-tile copy-back is
// what keeps that NaN out of C, and this is the test that says so.
func TestSgemmNonFinite(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))

	// alpha = 0 with a poisoned A.
	a := []float32{nan, nan, nan, nan}
	c := []float32{1, 2, 3, 4}
	Sgemm(NoTrans, NoTrans, 2, 2, 2, 0, a, 2, a, 2, 2, c, 2)
	if want := []float32{2, 4, 6, 8}; !equalF32(c, want) {
		t.Errorf("alpha=0 with NaN in A: c = %v, want %v (A must not be read)", c, want)
	}

	// beta = 0 with a poisoned C.
	a = []float32{1, 0, 0, 1}
	c = []float32{nan, nan, nan, nan}
	Sgemm(NoTrans, NoTrans, 2, 2, 2, 1, a, 2, a, 2, 0, c, 2)
	if want := []float32{1, 0, 0, 1}; !equalF32(c, want) {
		t.Errorf("beta=0 with NaN in C: c = %v, want %v (C must not be read)", c, want)
	}

	// A fringe tile in both directions, with an infinity in the data. Every
	// runner, because the padding lives in the packed panels every kernel reads.
	cb := gemmCombo{sweepTrans[0], 1, 1}
	for _, rn := range gemmRunners() {
		m, n, k := 5, 5, 3
		r := rand.New(rand.NewSource(sweepSeed))
		av := randMatrix(r, m*k)
		bv := randMatrix(r, k*n)
		cv := randMatrix(r, m*n)
		av[0] = inf
		bv[k*n-1] = inf
		c0 := append([]float32(nil), cv...)
		rn.run(false, false, m, n, k, cb.alpha, av, k, bv, n, cb.beta, cv, n)
		verifyExact(t, "Sgemm["+rn.name+"] fringe inf", cb, m, n, k, av, k, bv, n, c0, cv, n)
	}
	gemmExtra("nonfinite")
}

func equalF32(a, b []float32) bool {
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

// TestPackPanels holds internal/pack to an independent statement of the layout.
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
func TestPackPanels(t *testing.T) {
	r := rand.New(rand.NewSource(sweepSeed))
	blks := []int{2, 4, 32}
	counts := []int{0, 1, 3, 4, 5, 8, 33}
	kcs := []int{0, 1, 3, 4, 8}
	alphas := []float32{1, -0.75}
	combos := 0

	for _, blk := range blks {
		for _, count := range counts {
			for _, kc := range kcs {
				for _, trans := range []bool{false, true} {
					for _, alpha := range alphas {
						// Offsets deliberately nonzero: the block being packed is
						// almost never at the origin of the caller's matrix, and an
						// implementation that ignored i0/p0 would pass at (0,0).
						// Every source is ld-padded with poison for the same reason
						// TestSgemmLdPad exists.
						const off = 2
						// Two source shapes, because which one a call reads depends
						// on the operand and the transpose: "bc" is
						// blocked-axis-by-depth, "cb" is its transpose.
						bcLd := kc + off + 1
						cbLd := count + off + 1
						bc := padded(r, count+off, kc+off, bcLd)
						cb := padded(r, kc+off, count+off, cbLd)

						aSrc, aLd := bc, bcLd
						if trans {
							aSrc, aLd = cb, cbLd
						}
						gotA := filled(pack.ALen(blk, count, kc))
						pack.APanels(gotA, blk, alpha, aSrc, aLd, trans, off, count, off, kc)
						wantA := refPanels(blk, alpha, aSrc, aLd, !trans, off, count, off, kc)
						if !equalF32(gotA, wantA) {
							t.Fatalf("APanels blk=%d count=%d kc=%d trans=%v alpha=%g:\n got %v\nwant %v",
								blk, count, kc, trans, alpha, gotA, wantA)
						}

						bSrc, bLd := cb, cbLd
						if trans {
							bSrc, bLd = bc, bcLd
						}
						gotB := filled(pack.BLen(blk, count, kc))
						pack.BPanels(gotB, blk, bSrc, bLd, trans, off, kc, off, count)
						wantB := refPanels(blk, 1, bSrc, bLd, trans, off, count, off, kc)
						if !equalF32(gotB, wantB) {
							t.Fatalf("BPanels blk=%d count=%d kc=%d trans=%v:\n got %v\nwant %v",
								blk, count, kc, trans, gotB, wantB)
						}
						combos++
					}
				}
			}
		}
	}
	packCombos = combos
}

// refPanels is the packed layout stated directly: element x of block ib at depth
// p goes to dst[ib*blk*kc + p*blk + x], scaled by alpha, zero where x is past the
// end. depthContig says the source's contiguous index is the depth index.
func refPanels(blk int, alpha float32, src []float32, ld int, depthContig bool, b0, count, p0, kc int) []float32 {
	nb := (count + blk - 1) / blk
	dst := make([]float32, nb*blk*kc)
	for ib := 0; ib < nb; ib++ {
		for p := 0; p < kc; p++ {
			for x := 0; x < blk; x++ {
				var v float32
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

// filled returns a buffer of poison, so that a padding element pack forgets to
// write is a failure rather than a lucky zero.
func filled(n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = poison
	}
	return v
}

// ------------------------------------------------------------------- markers

// printGemmMarkers emits what scripts/gate-p3.sh parses. Called from TestMain in
// l1_test.go — one TestMain per package — and unconditionally, so a failing run
// still reports what it covered.
func printGemmMarkers() {
	kc, mc, nc := block.Params(activeKern)
	fmt.Printf("keel-sgemm-config: mr=%d nr=%d kc=%d mc=%d nc=%d edge=%s beta-variants=%d pack=%s\n",
		activeKern.MR, activeKern.NR, kc, mc, nc, block.EdgeStrategy, block.BetaVariants,
		"go-copy+scalar-transpose")
	fmt.Println("keel-sgemm-active:", ActiveKernTile()+"/"+ActiveKernBackend())
	fmt.Println("keel-sgemm-available:", strings.Join(AvailableKernels(), " "))

	var sizes []string
	for _, sz := range sweepSizes {
		if gemmSizesRun[sz] {
			sizes = append(sizes, strconv.Itoa(sz))
		}
	}
	fmt.Println("keel-sgemm-sizes-exercised:", strings.Join(sizes, " "))

	fmt.Printf("keel-sgemm-combos-exercised: trans=%s alpha=%s beta=%s combos=%d\n",
		joinTrans(sweepTrans), joinF32(sweepAlphas), joinF32(sweepBetas),
		len(sweepTrans)*len(sweepAlphas)*len(sweepBetas))

	for _, sz := range sweepSizes {
		if mode, ok := gemmVerifyMode[sz]; ok {
			fmt.Printf("keel-sgemm-verify: size=%d %s\n", sz, mode)
		}
	}

	var backends []string
	for name := range gemmBackendsRun {
		backends = append(backends, name)
	}
	sort.Strings(backends)
	fmt.Println("keel-sgemm-backends-exercised:", strings.Join(backends, " "))
	fmt.Println("keel-sgemm-extra-exercised:", strings.Join(gemmExtras, " "))
	fmt.Printf("keel-pack-combos-exercised: operands=A,B trans=false,true alpha=1,general "+
		"ragged=full,partial combos=%d\n", packCombos)
}

func joinTrans(ts []gemmTrans) string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.name
	}
	return strings.Join(out, ",")
}

func joinF32(vs []float32) string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = strconv.FormatFloat(float64(v), 'g', -1, 32)
	}
	return strings.Join(out, ",")
}
