// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && amd64

package vec

import "simd/archsimd"

// This file holds the SGEMM microkernel bodies. The tile protocol, the shape
// registry and the scalar reference live in internal/kern; only the K-loops are
// here, and they are here for a measured reason.
//
// # Why the loops are not in internal/kern
//
// CLAUDE.md puts every simd import in this package. A kernel in internal/kern
// therefore has to reach archsimd through the one-line shims above — and each
// level of inlined wrapper *with a Go body* costs a 1-byte XCHGL anchor NOP per
// call site in the loop body, because the generated instruction takes the
// wrapper's source position and the caller's statement position needs an anchor
// of its own (docs/toolchain-notes.md T9).
//
// Measured on the 2x32 tile's steady-state body, 16 FMAs per pass:
//
//	kern calling vec.Load512/vec.Broadcast512   90 insns, 24 NOPs   5.62 ins/FMA
//	vec calling archsimd directly               74 insns,  8 NOPs   4.62 ins/FMA
//
// Two levels of wrapper, two NOPs per call site, sixteen call sites: 27% of the
// loop body was anchor NOPs. Putting the loops in the package that is allowed to
// name archsimd removes one level and 16 instructions per pass, with no change to
// the rule and no archsimd import outside this directory. The remaining 8 NOPs
// are archsimd's own Slice-load wrappers, which have Go bodies of their own; a
// variant using the bodyless array-pointer form eliminates them but spills 12
// times, so it is not used (issue #17 records both measurements).
//
// # Why the loop conditions are slice lengths and the panels are re-sliced
//
// Both loop conditions are stated in terms of slice *lengths* rather than a
// counter, and both panels are re-sliced at the bottom of the body rather than
// indexed from a base. This is what lets the prover eliminate the bounds checks:
// `len(bp) >= 128` is exactly the fact needed to know that bp[112:128] is in
// range, and it is the loop condition, so it holds by construction on entry to
// the body. Indexing bp[p*32+112] from a counter would need the prover to reason
// about p's range and a multiplication, which it does not do reliably.
//
// It is not free: `ap, bp = ap[k:], bp[j:]` compiles to five integer
// instructions per panel — an ADDQ plus a branchless MOVQ/NEGQ/SARQ $63/ANDL
// clamp that advances the pointer only if the remainder is non-empty — and there
// are two panels. That, plus the counters and the compare-and-branch, is 18 of
// the 74 instructions in the 2x32 body: the largest non-FMA block in the loop
// (docs/toolchain-notes.md T10). It buys zero bounds checks and zero calls,
// which is what P2 asks for.
//
// # Why no shape has 12 accumulators
//
// go1.26.5 offers SIMD values only X0-X14, and its only FMA form writes to its
// first multiplicand, so an accumulate needs a live scratch register beyond the
// working set (docs/toolchain-notes.md T10, issue #18). 12 accumulators + 2 B
// vectors + 1 scratch is already 15, leaving nothing to hold the A broadcast.
// The zero-spill frontier is 8 accumulators. Kernel6x32 is DESIGN.md's tile,
// kept and benchmarked so the cost of that constraint is a number.

// Kernel2x32 computes C += A·B for one 2x32 tile from k-major packed panels,
// unrolled 4 k-steps.
//
// Four accumulators — two rows by two 16-lane vectors — live in zmm for the whole
// call. Per k-step: two vector loads from the B panel, two scalar broadcasts from
// the A panel, four FMAs. Nothing else. C is read and written once, outside the
// K-loop, so the loop touches only the two panels.
//
// This is the shape with the fewest instructions per FMA that go1.26.5 will
// allocate without spilling. Peak live pressure is exactly the 15 registers T10
// allows: 4 accumulators, 2 B vectors, 8 hoisted A scalars (the SSA scheduler
// moves all four k-steps' scalar loads to the top of the body), 1 FMA scratch.
// Unrolling 8 would need 16 A scalars and spill — which is why the unroll factor
// is a property of the shape rather than a free parameter.
//
// Requires len(a) >= kc*2, len(b) >= kc*32, ldc >= 32, and len(c) >= ldc+32. The
// caller guarantees all of it; see kern.Kernel.
func Kernel2x32(kc int, a, b, c []float32, ldc int) {
	// The zero Float32x16 is a zeroed vector register, so the accumulators need
	// no broadcast to start. Named <row><half>: l is columns 0-15, h is 16-31.
	var (
		c0l, c0h archsimd.Float32x16
		c1l, c1h archsimd.Float32x16
	)
	var bl, bh, av archsimd.Float32x16

	ap := a[:kc*2]
	bp := b[:kc*32]

	for len(ap) >= 8 && len(bp) >= 128 {
		// k + 0
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 1
		bl, bh = archsimd.LoadFloat32x16(bp[32:48]), archsimd.LoadFloat32x16(bp[48:64])
		av = archsimd.BroadcastFloat32x16(ap[2])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[3])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 2
		bl, bh = archsimd.LoadFloat32x16(bp[64:80]), archsimd.LoadFloat32x16(bp[80:96])
		av = archsimd.BroadcastFloat32x16(ap[4])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[5])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		// k + 3
		bl, bh = archsimd.LoadFloat32x16(bp[96:112]), archsimd.LoadFloat32x16(bp[112:128])
		av = archsimd.BroadcastFloat32x16(ap[6])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[7])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)

		ap, bp = ap[8:], bp[128:]
	}

	// Remainder: kc mod 4 k-steps. Correctness for user-supplied k, not a hot
	// path — P3 chooses KC as a multiple of the unroll.
	for len(ap) >= 2 && len(bp) >= 32 {
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		ap, bp = ap[2:], bp[32:]
	}

	// C += the accumulated tile, one row at a time. Outside the K-loop, so the
	// bounds checks here cost nothing per k and are left alone.
	r := c[0*ldc : 0*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c0l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c0h).Store(r[16:32])
	r = c[1*ldc : 1*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c1l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c1h).Store(r[16:32])
}

// Kernel4x32 computes C += A·B for one 4x32 tile, one k-step per pass.
//
// Eight accumulators — the most go1.26.5 will hold without spilling — plus 2 B
// vectors, 4 A scalars and 1 FMA scratch: 15 live, exactly the budget. That is
// also why there is no unroll: a second k-step would add four more hoisted
// scalar loads and spill.
//
// It issues more instructions per FMA than Kernel2x32 (6.25 against 4.62,
// because 18 instructions of loop overhead amortize over 8 FMAs instead of 16)
// and fewer *loads* per FMA (0.75 against 1.0, because one B panel load feeds
// four rows instead of two). Which of those two costs binds is a property of the
// host's front-end width and load ports, so both kernels ship and the benchmark
// decides per host.
//
// Requires len(a) >= kc*4, len(b) >= kc*32, ldc >= 32, and len(c) >= 3*ldc+32.
func Kernel4x32(kc int, a, b, c []float32, ldc int) {
	var (
		c0l, c0h archsimd.Float32x16
		c1l, c1h archsimd.Float32x16
		c2l, c2h archsimd.Float32x16
		c3l, c3h archsimd.Float32x16
	)
	var bl, bh, av archsimd.Float32x16

	ap := a[:kc*4]
	bp := b[:kc*32]

	for len(ap) >= 4 && len(bp) >= 32 {
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		ap, bp = ap[4:], bp[32:]
	}

	r := c[0*ldc : 0*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c0l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c0h).Store(r[16:32])
	r = c[1*ldc : 1*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c1l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c1h).Store(r[16:32])
	r = c[2*ldc : 2*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c2l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c2h).Store(r[16:32])
	r = c[3*ldc : 3*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c3l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c3h).Store(r[16:32])
}

// Kernel6x32 is DESIGN.md §4/P2's tile — 12 accumulators, unrolled 4 — and it
// spills. It is kept, exported, differentially tested and benchmarked, and
// deliberately left out of kern.Kernels() so that nothing ships it and the
// gate's zero-spill criterion stays binding on the kernels that do.
//
// It exists to make the T10 constraint a measured number, and the number moved
// when the toolchain did. On go1.26.5 the audit said 270 instructions with 90
// vector stack references for 48 FMAs; on go1.27.0, re-measured 2026-08-30, it
// says 219 for 44. The residual is not accumulator pressure — 36 of the 44 are
// broadcast-scalar round-trips through legacy-SSE MOVUPS inside the inlined
// archsimd wrapper, only 3 of the 12 accumulators spill, and the allocator
// leaves X24-X31 unused while doing it. docs/spill-report.md carries the
// decomposition; KERNEL.md the comparison against the two shapes that fit.
//
// Requires len(a) >= kc*6, len(b) >= kc*32, ldc >= 32, and len(c) >= 5*ldc+32.
func Kernel6x32(kc int, a, b, c []float32, ldc int) {
	var (
		c0l, c0h archsimd.Float32x16
		c1l, c1h archsimd.Float32x16
		c2l, c2h archsimd.Float32x16
		c3l, c3h archsimd.Float32x16
		c4l, c4h archsimd.Float32x16
		c5l, c5h archsimd.Float32x16
	)
	var bl, bh, av archsimd.Float32x16

	ap := a[:kc*6]
	bp := b[:kc*32]

	for len(ap) >= 24 && len(bp) >= 128 {
		// k + 0
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[4])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat32x16(ap[5])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 1
		bl, bh = archsimd.LoadFloat32x16(bp[32:48]), archsimd.LoadFloat32x16(bp[48:64])
		av = archsimd.BroadcastFloat32x16(ap[6])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[7])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[8])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[9])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[10])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat32x16(ap[11])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 2
		bl, bh = archsimd.LoadFloat32x16(bp[64:80]), archsimd.LoadFloat32x16(bp[80:96])
		av = archsimd.BroadcastFloat32x16(ap[12])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[13])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[14])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[15])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[16])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat32x16(ap[17])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		// k + 3
		bl, bh = archsimd.LoadFloat32x16(bp[96:112]), archsimd.LoadFloat32x16(bp[112:128])
		av = archsimd.BroadcastFloat32x16(ap[18])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[19])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[20])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[21])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[22])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat32x16(ap[23])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)

		ap, bp = ap[24:], bp[128:]
	}

	for len(ap) >= 6 && len(bp) >= 32 {
		bl, bh = archsimd.LoadFloat32x16(bp[0:16]), archsimd.LoadFloat32x16(bp[16:32])
		av = archsimd.BroadcastFloat32x16(ap[0])
		c0l, c0h = av.MulAdd(bl, c0l), av.MulAdd(bh, c0h)
		av = archsimd.BroadcastFloat32x16(ap[1])
		c1l, c1h = av.MulAdd(bl, c1l), av.MulAdd(bh, c1h)
		av = archsimd.BroadcastFloat32x16(ap[2])
		c2l, c2h = av.MulAdd(bl, c2l), av.MulAdd(bh, c2h)
		av = archsimd.BroadcastFloat32x16(ap[3])
		c3l, c3h = av.MulAdd(bl, c3l), av.MulAdd(bh, c3h)
		av = archsimd.BroadcastFloat32x16(ap[4])
		c4l, c4h = av.MulAdd(bl, c4l), av.MulAdd(bh, c4h)
		av = archsimd.BroadcastFloat32x16(ap[5])
		c5l, c5h = av.MulAdd(bl, c5l), av.MulAdd(bh, c5h)
		ap, bp = ap[6:], bp[32:]
	}

	r := c[0*ldc : 0*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c0l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c0h).Store(r[16:32])
	r = c[1*ldc : 1*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c1l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c1h).Store(r[16:32])
	r = c[2*ldc : 2*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c2l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c2h).Store(r[16:32])
	r = c[3*ldc : 3*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c3l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c3h).Store(r[16:32])
	r = c[4*ldc : 4*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c4l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c4h).Store(r[16:32])
	r = c[5*ldc : 5*ldc+32]
	archsimd.LoadFloat32x16(r[0:16]).Add(c5l).Store(r[0:16])
	archsimd.LoadFloat32x16(r[16:32]).Add(c5h).Store(r[16:32])
}
