// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"fmt"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/pack"
)

// The retention decomposition (#26, nest_bench_test.go) measures parts of the loop
// nest separately and reports what fraction of the whole each one is. That
// arithmetic is only a decomposition if the parts walk the same blocks the shipped
// nest walks, and the parts get their blocks from nestBlocks — a copy of gemm's
// three outer loops, which is exactly the kind of copy that drifts.
//
// So the copy is not trusted, it is executed: this drives a complete
// pack-and-multiply GEMM from nestBlocks alone and requires the result to equal
// Gemm's, element for element. A wrong bound, a missed remainder block, a B panel
// packed at the wrong (jc, pc) or one beta pass too many all show up as a numeric
// difference. Element-for-element rather than within a tolerance is right here
// because both sides do the same arithmetic in the same order — the summation order
// is the block structure, and that is the thing under test.
//
// Shapes: multiples of the block sizes and not, sizes above and below one block in
// each of the three dimensions, and non-square shapes where m, n and k differ, so
// the ic/jc/pc remainder blocks are all exercised. The block parameters are shrunk
// for the duration so a several-block nest fits in a test-sized problem — the
// structure under test is the loop bounds, and 4096-wide blocks would give every
// case exactly one of everything.
func TestNestBlocksDriveTheSameGemm(t *testing.T) {
	defer func(kc, mc, nc int) { KC, MC, NC = kc, mc, nc }(KC, MC, NC)
	KC, MC, NC = 24, 16, 64

	shapes := []struct{ m, n, k int }{
		{16, 64, 24},   // exactly one block in each dimension
		{48, 192, 72},  // several whole blocks
		{50, 200, 73},  // remainders in all three
		{3, 5, 7},      // smaller than one block, and smaller than one tile
		{129, 33, 1},   // k = 1: every depth block is the remainder
		{17, 256, 100}, // tall-thin C against a wide B panel
		{256, 17, 100}, // and the transpose of that
	}
	for _, kn := range kern.Measured() {
		for _, s := range shapes {
			name := fmt.Sprintf("%s/m=%d,n=%d,k=%d", kn.ID(), s.m, s.n, s.k)
			t.Run(name, func(t *testing.T) {
				const alpha, beta = 1.5, -0.75
				a, b := seq(s.m*s.k, 3), seq(s.k*s.n, 5)
				want, got := seq(s.m*s.n, 7), seq(s.m*s.n, 7)

				Gemm(kn, false, false, s.m, s.n, s.k, alpha, a, s.k, b, s.n, beta, want, s.n)
				nestFromBlocks(kn, s.m, s.n, s.k, alpha, a, s.k, b, s.n, beta, got, s.n)

				for i := range want {
					if want[i] != got[i] {
						t.Fatalf("element %d (row %d, col %d): the nest driven from nestBlocks gives %v, "+
							"Gemm gives %v — nestBlocks no longer walks the blocks block.go walks, so the "+
							"#26 decomposition's parts are not parts of the whole",
							i, i/s.n, i%s.n, got[i], want[i])
					}
				}
			})
		}
	}
}

// nestFromBlocks is a complete GEMM whose only loop structure is nestBlocks: pack B
// where the sequence says a new panel starts, pack A for every block, apply beta on
// the first depth block, multiply. It is the decomposition's parts put back
// together, which is what makes the comparison above meaningful.
func nestFromBlocks(kn kern.Kernel, m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	kc, mc, nc := plan(kn, m, n, k)
	ap := make([]float32, pack.ALen(kn.MR, mc, kc))
	bp := make([]float32, pack.BLen(kn.NR, nc, kc))
	tile := make([]float32, kn.MR*kn.NR)
	for _, blk := range nestBlocks(kn, m, n, k) {
		if blk.newPanel {
			pack.BPanels(bp, kn.NR, b, ldb, false, blk.pc, blk.kk, blk.jc, blk.jn)
		}
		pack.APanels(ap, kn.MR, alpha, a, lda, false, blk.ic, blk.im, blk.pc, blk.kk)
		cb := c[blk.ic*ldc+blk.jc:]
		if blk.pc == 0 {
			scaleTri(beta, blk.ic, blk.jc, blk.im, blk.jn, cb, ldc, triMask{})
		}
		macro(kn, ap, bp, blk.ic, blk.jc, blk.im, blk.jn, blk.kk, cb, ldc, tile, triMask{})
	}
}

// seq is a bounded, non-degenerate pattern with a per-array offset, so that A, B and
// C are distinguishable and a mixed-up index shows as a wrong value rather than as a
// coincidence.
func seq(n, off int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32((i+off)%11-5) * 0.25
	}
	return v
}
