// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package pack turns a caller's row-major matrix into the k-major packed panels
// internal/kern's microkernels consume (DESIGN.md §4/P3). It absorbs the three
// things the K-loop must not have to know about: the transpose flags, alpha, and
// the ragged edge.
//
// # The layout
//
// Both operands pack to the same shape — a sequence of panels, each one blocked
// index by kc floats, k-major within the panel:
//
//	dst[ib*blk*kc + p*blk + x] = src[block ib, element x, depth p]
//
// with blk = MR for A and NR for B. That is exactly kern's protocol
// (`a[p*MR+i]`, `b[p*NR+j]`), so a microkernel walks its panel straight forward
// with no index arithmetic. The panels of one block are contiguous, so the
// macrokernel reaches panel ib at a single multiply.
//
// # Alpha is folded into A here
//
// C = alpha·A·B + beta·C could scale A, B, the product, or C. This packs
// alpha·A, which is what BLIS does, for a reason that survives being written
// down: packing A costs O(mc·kc) multiplies against the tile's O(mc·nc·kc) FMAs,
// so the scaling is amortized by a factor of nc and — more to the point — it
// stays out of the audited K-loop entirely. Scaling the product instead would put
// a multiply next to every FMA; scaling C would need a second pass over C and
// would break down for beta ≠ 0.
//
// It is not numerically identical to scaling the product: (alpha·aᵢₚ)·bₚⱼ rounds
// once more per term than alpha·(aᵢₚ·bₚⱼ) would. BLAS does not specify the
// association, LAPACK does not rely on it, and internal/oracle.Gemm folds alpha
// the same way so the reference is comparing like with like. The tolerance model
// (oracle.Tolerance) covers the extra rounding as part of its f(n) term; no
// per-test epsilon is involved.
//
// alpha == 0 never reaches this package — Sgemm answers that case with beta·C, so
// a NaN in A cannot leak into the result through a multiply by zero.
//
// # The ragged edge is zero-padded, and that is the whole edge strategy
//
// A block whose row or column count is not a multiple of the tile gets its last
// panel filled out with zeros rather than truncated. A zero column of A or B
// contributes exactly zero to the accumulators, so the microkernel runs its full
// MR×NR shape on every tile, including the last one — one loop, one audited
// instruction count, no shape-dependent variant. internal/block then keeps the
// garbage out of C by accumulating fringe tiles into a temporary tile and adding
// back only the valid rectangle; see its package doc for why that choice was made
// over masked stores.
//
// The zeros are written, not assumed: these buffers are reused across every
// (jc, pc, ic) block of a call, so a stale value from a previous block would be
// indistinguishable from data.
//
// # Why this is not vectorized through the shim
//
// DESIGN.md §4/P3 lists "packing routines SIMD-accelerated through the shim".
// One direction of each pack already is, and the other cannot be on go1.26.5:
//
//   - Whichever axis is contiguous in the source is copied with `copy`, i.e. the
//     runtime's memmove — vectorized, and by hand-written assembly rather than by
//     the experiment under test.
//   - The other direction is a transpose: consecutive source elements land blk
//     floats apart in the panel. That needs either a strided store or an
//     in-register transpose. archsimd in go1.26.5 has no gather or scatter of any
//     width — the identifiers do not exist anywhere in
//     $GOROOT/src/simd/archsimd — so a strided store is not expressible. An
//     in-register 16×16 transpose *is* constructible from Permute (VPERMD) and
//     ConcatPermute, which do exist, but that is a permutation network to write,
//     test and audit for a routine whose cost is O(mc·kc) against the K-loop's
//     O(mc·nc·kc).
//
// So the transposing direction is a scalar loop until a measurement says the
// packing fraction is worth a permutation network. Issue #21 carries that
// measurement; this comment is the record of the decision, not an omission.
package pack

import "fmt"

// ALen returns the number of floats APanels writes for a rows×kc block of A
// blocked by mr — the ragged last panel included, since it is padded rather
// than truncated.
func ALen(mr, rows, kc int) int { return panelsLen(mr, rows, kc) }

// BLen returns the number of floats BPanels writes for a kc×cols block of B
// blocked by nr.
func BLen(nr, cols, kc int) int { return panelsLen(nr, cols, kc) }

func panelsLen(blk, count, kc int) int {
	if blk < 1 || count < 0 || kc < 0 {
		panic(fmt.Sprintf("pack: bad panel geometry blk=%d count=%d kc=%d", blk, count, kc))
	}
	return ((count + blk - 1) / blk) * blk * kc
}

// APanels packs rows [i0, i0+rows) and depth [p0, p0+kc) of A into mr-row
// panels, scaled by alpha.
//
// trans says a holds Aᵀ — a k×m matrix with row stride lda — so that the
// caller's op(A) is m×k either way. Which of the two the source is decides only
// which axis is contiguous, and therefore whether this is a copy or a transpose;
// the packed result is identical.
//
// dst must hold at least ALen(mr, rows, kc) floats.
func APanels(dst []float32, mr int, alpha float32, a []float32, lda int, trans bool, i0, rows, p0, kc int) {
	// A's rows are the blocked axis. Untransposed, A is m×k: the depth index is
	// the contiguous one, so packing transposes. Transposed, A is k×m and the
	// blocked axis is contiguous, so it copies.
	panels(dst, mr, alpha, a, lda, !trans, i0, rows, p0, kc)
}

// BPanels packs depth [p0, p0+kc) and columns [j0, j0+cols) of B into nr-column
// panels. No alpha: it is folded into A (see the package doc).
//
// trans says b holds Bᵀ — an n×k matrix with row stride ldb.
//
// dst must hold at least BLen(nr, cols, kc) floats.
func BPanels(dst []float32, nr int, b []float32, ldb int, trans bool, p0, kc, j0, cols int) {
	// The mirror image of APanels: B's columns are the blocked axis, and
	// untransposed B is k×n, so the blocked axis is the contiguous one.
	panels(dst, nr, 1, b, ldb, trans, j0, cols, p0, kc)
}

// panels is both packs. depthContig says the source's contiguous index is the
// depth index p — src[(b0+x)*ld + p0+p] — which is the transposing case;
// otherwise the blocked index is contiguous — src[(p0+p)*ld + b0+x] — and each
// k-step is one run of blk floats.
//
// The two branches are separate loop nests rather than one nest with an index
// expression: the whole point of packing is that the K-loop reads memory in
// order, and a shared nest would have made the contiguous case load-strided for
// the sake of sharing four lines.
// memmoveFloor is the run length, in float32 elements, at or above which the
// contiguous branch uses copy() instead of an assignment loop (issue #21).
//
// copy() on a slice of statically-unknown length is a runtime.memmove call —
// confirmed in the object code, `pack.go:169 CALL runtime.memmove`. That is the
// right instrument for B, whose blocked axis is NR = 32 (128 bytes per run), and
// the wrong one for A, whose blocked axis is MR ∈ {2,4}: 8 or 16 bytes per call,
// about 27,600 calls per pass at the shapes the nest uses. BenchmarkPackDirections
// measured the contiguous branch at 2.8× slower than the transposing branch on the
// A side for exactly that reason — the cost scales as 1/blk and flattens once the
// run reaches 64 bytes, and removing the source stride entirely changes nothing
// (1.0–1.1×), which is what ruled out locality as the explanation.
//
// 16 elements is that 64-byte flattening point. It is a measured threshold and not
// a natural constant: it has no relation to a vector width, and a host whose
// memmove has a cheaper entry path would want it lower.
const memmoveFloor = 16

func panels(dst []float32, blk int, alpha float32, src []float32, ld int, depthContig bool, b0, count, p0, kc int) {
	need := panelsLen(blk, count, kc)
	if len(dst) < need {
		panic(fmt.Sprintf("pack: dst has %d floats, need %d", len(dst), need))
	}
	if kc == 0 || count == 0 {
		return // nothing to pack; also what keeps nb's divisor nonzero
	}
	nb := need / (blk * kc)
	for ib := 0; ib < nb; ib++ {
		panel := dst[ib*blk*kc : (ib+1)*blk*kc]
		// Valid elements in this panel: blk, except in the last one.
		valid := blk
		if r := count - ib*blk; r < blk {
			valid = r
		}
		base := b0 + ib*blk
		if depthContig {
			for x := 0; x < valid; x++ {
				row := src[(base+x)*ld+p0 : (base+x)*ld+p0+kc]
				for p, v := range row {
					panel[p*blk+x] = alpha * v
				}
			}
			for x := valid; x < blk; x++ {
				for p := 0; p < kc; p++ {
					panel[p*blk+x] = 0
				}
			}
			continue
		}
		for p := 0; p < kc; p++ {
			row := src[(p0+p)*ld+base : (p0+p)*ld+base+valid]
			out := panel[p*blk : (p+1)*blk]
			switch {
			case alpha != 1:
				for x, v := range row {
					out[x] = alpha * v
				}
			case blk >= memmoveFloor:
				copy(out, row)
			default:
				// Deliberately not copy(): see memmoveFloor. A plain assignment
				// loop, not alpha*v with alpha known to be 1, because a multiply
				// would quiet a signalling NaN where copy does not — the
				// asymmetry TestBranchesAgree documents stays exactly as
				// documented rather than moving to a new place.
				for x, v := range row {
					out[x] = v
				}
			}
			for x := valid; x < blk; x++ {
				out[x] = 0
			}
		}
	}
}
