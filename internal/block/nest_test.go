// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"fmt"
	"math"
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

// The MB sweep (#37, BenchmarkTrsmMB) publishes three counted claims, and the
// benchmark that computes them cannot run here: benchKernels skips every host with no
// vector microkernel, which is this one and CI both. So the counts are checked by a
// test, on the tree's own arithmetic, where they can actually be executed before
// being quoted anywhere (DESIGN.md §5 rule 11).
//
// The three claims, each checked against an independent derivation rather than
// against a restatement of trsmPackCount (§5 rule 10):
//
//	sum         TrsmWork's two terms add to the gate's n·m·(m+1) exactly
//	level       the re-pack at the published point is rhs·MB·nb(nb−1)/2 elements,
//	            which is the closed form obtained by summing `solved` over the block
//	            loop by hand, and it comes out at 5.29e8 for m = n = 4096, MB = 64
//	direction   over the swept grid, solve flops rise strictly with MB while re-pack
//	            elements fall strictly — the two opposed slopes that make MB the
//	            parameter under test rather than the solve's inner loop
//
// The level claim holds only where the partition is even and the rank update has one
// block on the axis that would otherwise multiply the count, so both preconditions
// are asserted rather than assumed: a shape that violated either would make the
// closed form disagree with the count for a reason that is not a defect.
//
// Writing the closed form per side is the point rather than an inconvenience. The
// first version of trsmPackCount counted B panels only, which is right on the left
// and 64× low on the right, because X is gemm's B operand there and its A operand
// here (tri.go:407 against :442). This test is what would have caught it, and did.
func TestTrsmMBCounts(t *testing.T) {
	defer func(mb int) { MB = mb }(MB)
	// The shipped AVX-512 tile shape, constructed rather than dispatched: the counts
	// are a property of MR, NR and the blocking vars, and the published figures are
	// for the shape the gate hosts run. Fn is nil and never called — nothing here
	// multiplies anything.
	kn := kern.Kernel{Name: kern.AVX512, MR: 2, NR: 32}

	for _, side := range []struct {
		tag  string
		left bool
	}{{"L", true}, {"R", false}} {
		t.Run("level/side="+side.tag, func(t *testing.T) {
			const d, mb = 4096, 64
			MB = mb
			rank, solve := TrsmWork(side.left, d, d)
			total := float64(d) * float64(d) * float64(d+1)
			if rank+solve != total {
				t.Errorf("TrsmWork sums to %.10g, want %.10g (rank %.10g, solve %.10g)",
					rank+solve, total, rank, solve)
			}
			aelem, belem, _, blocks := trsmPackCount(kn, side.left, d, d)
			if want := d / mb; blocks != want {
				t.Fatalf("blocks = %d, want %d", blocks, want)
			}
			// Preconditions for the closed form, checked in the shape's own terms.
			gm, gn := mb, d
			if !side.left {
				gm, gn = d, mb
			}
			if _, _, nc := plan(kn, gm, gn, d-mb); nc < gn {
				t.Fatalf("nc = %d < gn = %d: the rank update has more than one column "+
					"block at this shape, so the closed form below does not apply", nc, gn)
			}
			// Summing `solved` over the block loop by hand: block i (1-based) has
			// i·MB rows solved, and every one of them is packed once, over rhs
			// right-hand sides on the big operand and over MB on the small one.
			nb := float64(blocks)
			solvedSum := mb * nb * (nb - 1) / 2
			big, small := float64(d)*solvedSum, float64(mb)*solvedSum
			wantA, wantB := small, big
			if !side.left {
				wantA, wantB = big, small
			}
			if aelem != wantA || belem != wantB {
				t.Errorf("packed elements: A %.10g B %.10g, want A %.10g B %.10g",
					aelem, belem, wantA, wantB)
			}
			t.Logf("side=%s n=%d mb=%d: total %.6g rank %.6g solve %.6g (solve %.4f%% of work); "+
				"repack A %.6g + B %.6g = %.6g elements, %.6g bytes",
				side.tag, d, mb, total, rank, solve, 100*solve/total,
				aelem, belem, aelem+belem, 4*(aelem+belem))
		})

		t.Run("direction/side="+side.tag, func(t *testing.T) {
			const d = 2048
			prevSolve, prevPack := 0.0, math.Inf(1)
			for _, mb := range []int{32, 64, 128, 256, 512} {
				MB = mb
				_, solve := TrsmWork(side.left, d, d)
				aelem, belem, _, _ := trsmPackCount(kn, side.left, d, d)
				pack := aelem + belem
				if solve <= prevSolve {
					t.Errorf("mb=%d: solve flops %.6g did not rise above %.6g", mb, solve, prevSolve)
				}
				if pack >= prevPack {
					t.Errorf("mb=%d: repack elements %.6g did not fall below %.6g", mb, pack, prevPack)
				}
				prevSolve, prevPack = solve, pack
			}
		})
	}
}

// The MB sweep's second arm, solve-only, is a replay: it makes the diagonal-solve
// calls Trsm makes and none of the rank updates, and its time divided by the full
// arm's is the share #37 asks for. It is therefore a copy of the shipped block loop,
// and TestNestBlocksDriveTheSameGemm's premise applies unchanged — a copy of a loop
// nest is exactly the kind of thing that drifts, and a drifted replay reports a
// perfectly plausible number.
//
// So the copy is executed rather than trusted, on the two things it can get wrong:
//
//	partition   Against Trsm itself, at a block-diagonal A. Every off-diagonal block
//	            of the triangle is zero, so each rank update subtracts exactly zero
//	            and Trsm reduces to its diagonal solves — the replay must then equal
//	            it element for element. A wrong block offset, a missed remainder
//	            block, a wrong bl, a transposed slice or the wrong side's indexing
//	            all move a result. Element-for-element and not to a tolerance: both
//	            sides run the same solveLeft/solveRight on the same addresses, so any
//	            difference at all is structural.
//	order       Against a written-out expectation, per side and per triangle.
//
// What the partition check CANNOT see is the order (§5 rule 12), and it is worth
// stating why rather than testing harder: with the rank updates gone the solves
// commute. Each one reads and writes only its own rows (or columns) of B, so a replay
// running the blocks backwards computes the identical array. The order matters to the
// *timing* — a backwards replay touches B's memory in the opposite direction — and to
// nothing this check can observe, which is what the second sub-test is for.
//
// The alpha is 1 in the partition check, deliberately and not incidentally: Trsm
// scales B by alpha (tri.go:395 on the first block, gemm's beta after it) and the
// replay does not scale at all, so the two agree only at alpha = 1. At any other
// alpha this test would be measuring that difference instead of the partition.
func TestTrsmSolveReplay(t *testing.T) {
	defer func(mb int) { MB = mb }(MB)

	t.Run("partition", func(t *testing.T) {
		const mb = 8
		MB = mb
		// Shapes with and without a remainder block on the solved axis, and with the
		// right-hand-side count above and below the axis, so the strips() split is not
		// always the same shape as the block partition.
		shapes := []struct{ m, n int }{
			{16, 24}, // exact multiple of MB on both
			{20, 24}, // remainder on m
			{16, 21}, // remainder on n
			{21, 5},  // remainder on both, few right-hand sides
			{5, 40},  // smaller than one block on the axis
		}
		for _, kn := range kern.Measured() {
			for _, s := range shapes {
				for _, side := range []struct {
					tag  string
					left bool
				}{{"L", true}, {"R", false}} {
					t.Run(fmt.Sprintf("%s/m=%d,n=%d,side=%s", kn.ID(), s.m, s.n, side.tag), func(t *testing.T) {
						d := s.m
						if !side.left {
							d = s.n
						}
						a := blockDiagTri(d, mb)
						want, got := seq(s.m*s.n, 7), seq(s.m*s.n, 7)
						Trsm(kn, side.left, true, false, false, s.m, s.n, 1, a, d, want, s.n)
						trsmSolvesOnly(side.left, s.m, s.n, a, d, got, s.n)
						for i := range want {
							if want[i] != got[i] {
								t.Fatalf("element %d (row %d, col %d): the replay gives %v, Trsm at a "+
									"block-diagonal A gives %v — trsmSolvesOnly no longer walks the "+
									"partition tri.go walks, so solve-only ÷ full is not the diagonal "+
									"solves' share of anything",
									i, i/s.n, i%s.n, got[i], want[i])
							}
						}
					})
				}
			}
		}
	})

	// The rule, from tri.go:385 and :416: a left solve's row block depends on the
	// blocks above it when op(A) is lower, so it walks forward; a right solve's column
	// block depends on the blocks after it when op(A) is lower, so it walks backward.
	// Transposing swaps both. The offsets are written out rather than computed so this
	// is an independent statement of the rule and not a second evaluation of
	// trsmSolveOrder's own expression (§5 rule 10).
	t.Run("order", func(t *testing.T) {
		const mb, axis = 4, 10 // three blocks, the last one a remainder
		MB = mb
		fwd, back := []int{0, 4, 8}, []int{8, 4, 0}
		for _, c := range []struct {
			left, lowerEff bool
			want           []int
		}{
			{true, true, fwd},
			{true, false, back},
			{false, true, back},
			{false, false, fwd},
		} {
			got := trsmSolveOrder(c.left, c.lowerEff, axis)
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("left=%v lowerEff=%v: solve order %v, want %v",
					c.left, c.lowerEff, got, c.want)
			}
		}
	})
}

// blockDiagTri is a d×d lower-triangular matrix whose only nonzero entries are inside
// the mb×mb blocks on the diagonal, which is what makes Trsm's rank updates subtract
// exactly zero and reduces it to its diagonal solves.
//
// The diagonal is 2 rather than 1 so that the solve actually divides — a unit
// diagonal would leave the substitution's divide untested, and the replay indexes the
// diagonal block itself. Off-diagonals within a block are small and nonzero so that
// each solve's inner axpy runs over real data.
func blockDiagTri(d, mb int) []float32 {
	v := make([]float32, d*d)
	for i := 0; i < d; i++ {
		for j := 0; j <= i; j++ {
			if i/mb != j/mb {
				continue
			}
			if i == j {
				v[i*d+j] = 2
				continue
			}
			v[i*d+j] = float32((i+2*j)%7-3) * 0.125
		}
	}
	return v
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
