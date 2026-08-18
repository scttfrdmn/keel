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
// # A symmetric operand is reflected here, not expanded by the caller
//
// ASymPanels and BSymPanelsPart read a matrix of which only one triangle is
// stored, taking an element on the other side from its reflection. The panel at
// (ic, pc) needs entries on both sides of the diagonal, so the alternative was for
// the caller to reflect A once into a dense square and hand this package an
// ordinary matrix — which is what internal/block.Symm did until issue #36, at a cost
// of O(d²) scratch (67 MB at d=4096) and a whole d² pass over A before this package
// made its own.
//
// Doing it here costs no scratch and no extra pass: each run is split at the
// diagonal instead of tested per element, so the stored part stays one contiguous
// source run and at most one side of the run is strided.
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
	panels(dst, mr, alpha, a, lda, !trans, symSrc{}, i0, rows, p0, kc, 0, NPanels(mr, rows))
}

// ASymPanels packs the block APanels packs, from a symmetric A of which only the
// triangle named by lower is stored. An element on the other side of the diagonal
// is read from its reflection; the unstored triangle is never touched.
//
// There is no trans, and that is not an omission: a symmetric matrix is its own
// transpose, so the flag would have one meaning and two spellings. The orientation
// passed below is the untransposed one because it is the loop order whose stored
// run is contiguous — one row of A per packed slot — not because the source is
// assumed to be anything.
func ASymPanels(dst []float32, mr int, alpha float32, a []float32, lda int, lower bool, i0, rows, p0, kc int) {
	panels(dst, mr, alpha, a, lda, true, symSrc{on: true, lower: lower}, i0, rows, p0, kc, 0, NPanels(mr, rows))
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
	panels(dst, nr, 1, b, ldb, trans, symSrc{}, j0, cols, p0, kc, 0, NPanels(nr, cols))
}

// BPanelsPart packs only panels [from, to) of the same block BPanels packs, and
// writes nothing outside dst[from*nr*kc : to*nr*kc].
//
// # Why this exists
//
// So the caller can pack one block of B with several goroutines. The panels are a
// partition of dst — panel ib is exactly dst[ib*nr*kc:(ib+1)*nr*kc], including the
// zero padding of a ragged last one — so disjoint ranges need no lock, no ordering
// and no reduction, and the packed result is bit-identical to the serial pack
// whatever order the ranges run in, because packing copies and scales rather than
// accumulating.
//
// It takes the WHOLE dst rather than a subslice, and validates it against the whole
// block's BLen, so that a range call has the same bounds check a serial call has.
// Handing each worker its own subslice would move that check into the caller, where
// an off-by-one in the partition would become a silent short pack instead of a panic
// — and a short pack leaves whatever the pooled buffer held last in the panel the
// microkernel then reads.
//
// A from >= to is legal and packs nothing, so a partition may hand a worker an empty
// range without the caller special-casing it.
func BPanelsPart(dst []float32, nr int, b []float32, ldb int, trans bool, p0, kc, j0, cols, from, to int) {
	panels(dst, nr, 1, b, ldb, trans, symSrc{}, j0, cols, p0, kc, from, to)
}

// BSymPanelsPart is BPanelsPart from a symmetric B of which only the triangle
// named by lower is stored. See ASymPanels for why there is no trans.
//
// There is no serial BSymPanels beside it because internal/block packs every B
// block through the range form, so a second entry point would be reached only from
// tests. A full pack is from = 0, to = NPanels(nr, cols).
func BSymPanelsPart(dst []float32, nr int, b []float32, ldb int, lower bool, p0, kc, j0, cols, from, to int) {
	panels(dst, nr, 1, b, ldb, false, symSrc{on: true, lower: lower}, j0, cols, p0, kc, from, to)
}

// NPanels returns how many blk-wide panels a count-wide axis packs into: the
// number of units BPanelsPart's range indexes, and the ragged last one counts.
func NPanels(blk, count int) int {
	if blk < 1 || count < 0 {
		panic(fmt.Sprintf("pack: bad panel geometry blk=%d count=%d", blk, count))
	}
	return (count + blk - 1) / blk
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

// symSrc says the source stores only one triangle of a symmetric matrix. The zero
// value is an ordinary matrix, which is what APanels, BPanels and BPanelsPart pass.
type symSrc struct {
	on    bool
	lower bool
}

// from and to bound the panel indices this call writes; a serial full pack passes
// (0, nb). The bound check is against the whole block either way — see BPanelsPart.
func panels(dst []float32, blk int, alpha float32, src []float32, ld int, depthContig bool, sym symSrc, b0, count, p0, kc, from, to int) {
	need := panelsLen(blk, count, kc)
	if len(dst) < need {
		panic(fmt.Sprintf("pack: dst has %d floats, need %d", len(dst), need))
	}
	if kc == 0 || count == 0 {
		return // nothing to pack; also what keeps nb's divisor nonzero
	}
	nb := need / (blk * kc)
	if from < 0 || to > nb {
		panic(fmt.Sprintf("pack: panel range [%d,%d) outside [0,%d)", from, to, nb))
	}
	for ib := from; ib < to; ib++ {
		panel := dst[ib*blk*kc : (ib+1)*blk*kc]
		// Valid elements in this panel: blk, except in the last one.
		valid := blk
		if r := count - ib*blk; r < blk {
			valid = r
		}
		base := b0 + ib*blk
		if sym.on {
			symPanel(panel, blk, alpha, src, ld, depthContig, sym.lower, base, valid, p0, kc)
			continue
		}
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
				//
				// staticcheck's S1001 says to write copy() here, and copy() is
				// what the branch above this one does. The suppression is the
				// whole point of the change rather than a wart on it: at
				// blk < memmoveFloor, copy() is a runtime.memmove call per 8 or
				// 16 bytes, and replacing it with this loop measured +42.1%
				// (p=0.000) on the A-side pack at MR=2. This is the repo's only
				// lint suppression; if a future toolchain inlines short copies,
				// delete the loop, the constant and this comment together and
				// re-measure.
				//nolint:staticcheck // S1001: copy() is a memmove call here; see memmoveFloor
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

// symPanel fills one panel from a symmetric source, including its zero padding.
//
// In both of panels' orientations the source address is src[row*ld + col] with row
// the first index — the blocked index when the depth one is contiguous, the depth
// index otherwise — so the reflection rule is written once and serves the A side
// and the B side alike. That is also why a symmetric source has no transpose flag:
// the rule does not read one.
//
// The orientation still decides the loop order, because the stored part of a run is
// contiguous in the source only along the column axis. It is a run and not a set of
// scattered elements, which is what keeps this the same cost as an ordinary pack:
// storedRun splits at the diagonal, so at most one end of each run is strided and
// no element is tested individually.
func symPanel(panel []float32, blk int, alpha float32, src []float32, ld int, depthContig, lower bool,
	base, valid, p0, kc int) {

	if depthContig {
		// Row is the blocked index, held across the run over depth. alpha·v
		// throughout, exactly as panels' transposing branch does it, so the two
		// agree on every value including the NaN payloads TestBranchesAgree pins.
		for x := 0; x < valid; x++ {
			r := base + x
			lo, hi := storedRun(lower, r, p0, kc)
			for p := 0; p < lo; p++ {
				panel[p*blk+x] = alpha * src[(p0+p)*ld+r]
			}
			for p, v := range src[r*ld+p0+lo : r*ld+p0+hi] {
				panel[(lo+p)*blk+x] = alpha * v
			}
			for p := hi; p < kc; p++ {
				panel[p*blk+x] = alpha * src[(p0+p)*ld+r]
			}
		}
		for x := valid; x < blk; x++ {
			for p := 0; p < kc; p++ {
				panel[p*blk+x] = 0
			}
		}
		return
	}
	// Row is the depth index, held across the run over the blocked axis.
	for p := 0; p < kc; p++ {
		r := p0 + p
		out := panel[p*blk : (p+1)*blk]
		lo, hi := storedRun(lower, r, base, valid)
		for x := 0; x < lo; x++ {
			out[x] = alpha * src[(base+x)*ld+r]
		}
		run := src[r*ld+base+lo : r*ld+base+hi]
		if alpha == 1 {
			// This nest is the B side, where blk is NR and alpha is folded into A,
			// so the stored run is the same NR-wide copy the ordinary contiguous
			// branch makes — minus the memmoveFloor test, which would fire only on
			// the short remainder next to the diagonal.
			copy(out[lo:hi], run)
		} else {
			for x, v := range run {
				out[lo+x] = alpha * v
			}
		}
		for x := hi; x < valid; x++ {
			out[x] = alpha * src[(base+x)*ld+r]
		}
		for x := valid; x < blk; x++ {
			out[x] = 0
		}
	}
}

// storedRun returns the part [lo, hi) of a run of cnt columns starting at column
// off, in row r of a symmetric matrix, whose elements are the stored ones. lower
// says the stored triangle is the lower one; the diagonal is stored either way.
//
// At most one end of the run is left over, never both, because the stored side of a
// row is a prefix (lower) or a suffix (upper) of it. A run entirely on the reflected
// side comes back empty rather than inverted.
//
// internal/block's triMask.rowRange is the same arithmetic about a different thing —
// which columns of C are *written*, not which entries of A are *stored* — and block
// imports this package, so the resemblance cannot be factored out and should not be:
// one is a property of the output mask and the other of the input layout.
func storedRun(lower bool, r, off, cnt int) (lo, hi int) {
	if lower {
		// Stored where col <= r, i.e. off+x <= r.
		hi = r - off + 1
		if hi < 0 {
			hi = 0
		}
		if hi > cnt {
			hi = cnt
		}
		return 0, hi
	}
	// Stored where col >= r.
	lo = r - off
	if lo < 0 {
		lo = 0
	}
	if lo > cnt {
		lo = cnt
	}
	return lo, cnt
}
