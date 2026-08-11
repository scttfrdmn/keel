// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package keel is a pure-Go float32 BLAS subset whose hot kernels target the
// experimental simd/archsimd packages (GOEXPERIMENT=simd). See DESIGN.md.
//
// Status: pre-alpha skeleton. Every routine below is a stub until its phase
// gate is green; see DESIGN.md §4 and the GitHub milestones.
package keel

import (
	"math"

	"github.com/scttfrdmn/keel/internal/block"
	"github.com/scttfrdmn/keel/internal/l1"
)

// Level 1 — Phase P1.
//
// Each routine validates its arguments, then splits two ways: unit stride goes
// to the active backend's kernel (internal/l1, one indirect call for the whole
// vector), and any other stride goes to a scalar strided loop right here. See
// internal/l1's package doc for why strided access is not vectorized.
//
// Argument errors panic rather than returning silently as reference BLAS does.
// Reference SSCAL's `IF (n.LE.0 .OR. incx.LE.0) RETURN` turns a caller's
// off-by-one into a no-op that looks like success; in Go the caller has no
// XERBLA to consult, so a bad n, a zero stride, or a slice too short for the
// stride is a program bug and says so. n == 0 is not an error: it is the empty
// vector, and the slices may be nil.

// Sdot returns the inner product of x and y.
//
// The summation order is the active backend's, so the result is not bit-stable
// across backends — it is stable within one, and every backend is held to
// DESIGN.md §5's tolerance model against a float64 oracle.
func Sdot(n int, x []float32, incX int, y []float32, incY int) float32 {
	checkVector("Sdot", "x", n, x, incX)
	checkVector("Sdot", "y", n, y, incY)
	if n == 0 {
		return 0
	}
	if incX == 1 && incY == 1 {
		return activeL1.Dot(x[:n], y[:n])
	}
	return stridedDot(n, x, incX, y, incY)
}

// Saxpy computes y += alpha*x.
//
// alpha == 0 returns without touching y, matching reference SAXPY. That is a
// numerics decision, not just a shortcut: y[i] += 0*x[i] would poison y with
// NaN wherever x holds a NaN or an infinity, and callers that scale by a
// computed-to-zero alpha rely on the vector being left alone.
//
// x and y must not overlap unless they are the same vector with the same
// stride; BLAS leaves partial overlap undefined and keel does not detect it.
func Saxpy(n int, alpha float32, x []float32, incX int, y []float32, incY int) {
	checkVector("Saxpy", "x", n, x, incX)
	checkVector("Saxpy", "y", n, y, incY)
	if n == 0 || alpha == 0 {
		return
	}
	if incX == 1 && incY == 1 {
		activeL1.Axpy(alpha, x[:n], y[:n])
		return
	}
	ix, iy := offset(n, incX), offset(n, incY)
	for j := 0; j < n; j++ {
		y[iy+j*incY] += alpha * x[ix+j*incX]
	}
}

// Sscal computes x *= alpha.
//
// No alpha == 0 shortcut, unlike Saxpy: reference SSCAL has none either, and
// the difference is real. Scaling by zero must write zeros (that is the whole
// point of the call), and it must leave NaN as NaN because 0*NaN is NaN.
func Sscal(n int, alpha float32, x []float32, incX int) {
	checkVectorPos("Sscal", "x", n, x, incX)
	if n == 0 {
		return
	}
	if incX == 1 {
		activeL1.Scal(alpha, x[:n])
		return
	}
	for j := 0; j < n; j++ {
		x[j*incX] *= alpha
	}
}

// Sasum returns the sum of magnitudes of x.
func Sasum(n int, x []float32, incX int) float32 {
	checkVectorPos("Sasum", "x", n, x, incX)
	if n == 0 {
		return 0
	}
	if incX == 1 {
		return activeL1.Asum(x[:n])
	}
	var a0, a1 float32
	j := 0
	for ; j+2 <= n; j += 2 {
		a0 += absf32(x[j*incX])
		a1 += absf32(x[(j+1)*incX])
	}
	for ; j < n; j++ {
		a0 += absf32(x[j*incX])
	}
	return a0 + a1
}

// Snrm2 returns the Euclidean norm of x.
//
// # Why this is not just sqrt(sum of squares)
//
// Squaring costs half of float32's exponent range. Any |xᵢ| above ~1.8e19
// squares to +Inf and any below ~1.1e-19 squares to zero, so the naive sum
// overflows for large inputs and silently loses small ones — for x = {1e-30}
// it returns 0 where the answer is 1e-30 exactly, representable in float32.
// LAPACK's SNRM2 solves this with a running-scale loop; keel does not, because
// that loop has a branch per element and would drag the vector kernel down to
// scalar speed for the 99% of inputs that never come near either limit.
//
// Instead the fast kernel runs unguarded and its *result* is inspected. A sum
// of squares is monotonically non-decreasing, so overflow can only end at +Inf
// and total underflow can only end at exactly 0 — one check after the loop sees
// everything a per-element check would. Either outcome (plus NaN, which the
// same check catches) reruns the whole vector in float64, where no rescaling is
// needed at all: the largest float32 squared is ~1.2e77 and float64 reaches
// ~1.8e308, so float32 inputs cannot overflow a float64 accumulator until
// n exceeds 1e230 elements. The rescue is therefore exact-by-construction
// rather than clever, which is the property worth having in a path that runs
// only on inputs nobody tested.
func Snrm2(n int, x []float32, incX int) float32 {
	checkVectorPos("Snrm2", "x", n, x, incX)
	if n == 0 {
		return 0
	}
	var s float32
	if incX == 1 {
		s = activeL1.SumSq(x[:n])
	} else {
		var a0, a1 float32
		j := 0
		for ; j+2 <= n; j += 2 {
			v0, v1 := x[j*incX], x[(j+1)*incX]
			a0 += v0 * v0
			a1 += v1 * v1
		}
		for ; j < n; j++ {
			v := x[j*incX]
			a0 += v * v
		}
		s = a0 + a1
	}
	// s == 0 with a nonzero input means underflow; a non-finite s means overflow
	// or a NaN in the data. Both go to the float64 path. s == 0 for a genuinely
	// zero vector also lands there and costs one pass to confirm 0.
	if s > 0 && !math.IsInf(float64(s), 1) {
		return float32(math.Sqrt(float64(s)))
	}
	var sum float64
	for j := 0; j < n; j++ {
		v := float64(x[j*incX])
		sum += v * v
	}
	return float32(math.Sqrt(sum))
}

// Isamax returns the 0-based index of the first element of greatest magnitude,
// or -1 when n == 0.
//
// Two deviations from Fortran ISAMAX, both deliberate. The index is 0-based
// because every other index in this package is. And an empty vector returns -1
// rather than 0, because 0 is a valid answer for n == 1 and a sentinel that
// collides with a real result is how off-by-one bugs survive review.
//
// The NaN convention is inherited exactly from the reference: a NaN in the
// first position wins, a NaN anywhere else is ignored. See internal/l1.Iamax
// for why that convention is worth preserving and why this one routine has no
// vector backend.
func Isamax(n int, x []float32, incX int) int {
	checkVectorPos("Isamax", "x", n, x, incX)
	if n == 0 {
		return -1
	}
	if incX == 1 {
		return l1.Iamax(x[:n])
	}
	best, bi := absf32(x[0]), 0
	for j := 1; j < n; j++ {
		if v := absf32(x[j*incX]); v > best {
			best, bi = v, j
		}
	}
	return bi
}

// stridedDot is Sdot's non-unit-stride path, split out only to keep Sdot's own
// body short. Two accumulators rather than the kernels' four: this loop is
// bound by the gather, not by FMA latency.
func stridedDot(n int, x []float32, incX int, y []float32, incY int) float32 {
	ix, iy := offset(n, incX), offset(n, incY)
	var a0, a1 float32
	j := 0
	for ; j+2 <= n; j += 2 {
		a0 += x[ix+j*incX] * y[iy+j*incY]
		a1 += x[ix+(j+1)*incX] * y[iy+(j+1)*incY]
	}
	for ; j < n; j++ {
		a0 += x[ix+j*incX] * y[iy+j*incY]
	}
	return a0 + a1
}

// offset returns the storage index of element 0 of a strided BLAS vector. For
// inc < 0 the vector runs backwards from the far end, so element j is at
// offset(n, inc) + j*inc. Same rule as oracle.Index, stated separately because
// the oracle is a test-only cross-check and must not become this code's
// definition of the convention.
//
// Only Sdot and Saxpy call it: reference BLAS defines negative strides for
// those two and requires inc > 0 for the rest (see checkVectorPos), so the
// remaining routines index from zero.
func offset(n, inc int) int {
	if inc > 0 {
		return 0
	}
	return -(n - 1) * inc
}

// checkVector validates n and one vector argument for the routines that accept
// any nonzero stride.
func checkVector(fn, arg string, n int, v []float32, inc int) {
	if n < 0 {
		panic("keel: " + fn + ": n < 0")
	}
	if inc == 0 {
		panic("keel: " + fn + ": inc" + arg + " == 0")
	}
	if n == 0 {
		return // empty vector; v may be nil
	}
	span := inc
	if span < 0 {
		span = -span
	}
	if need := (n-1)*span + 1; len(v) < need {
		panic("keel: " + fn + ": " + arg + " shorter than n and inc" + arg + " require")
	}
}

// checkVectorPos is checkVector for the routines reference BLAS defines only
// for inc > 0: SSCAL, SASUM, SNRM2, ISAMAX. Reference returns silently on
// inc <= 0; keel panics, because a caller passing a negative stride to Sscal
// believes it is scaling a vector and would instead scale nothing.
func checkVectorPos(fn, arg string, n int, v []float32, inc int) {
	if inc <= 0 {
		if n < 0 {
			panic("keel: " + fn + ": n < 0")
		}
		panic("keel: " + fn + ": inc" + arg + " <= 0 (not defined for this routine)")
	}
	checkVector(fn, arg, n, v, inc)
}

// checkTranspose maps a Transpose to the bool the internals use, panicking on
// anything else. BLAS's ConjTrans is meaningless for a real type and reference
// SGEMM rejects it; a byte that is neither 'N' nor 'T' is a caller typo, and
// treating an unknown flag as NoTrans would silently compute the wrong product.
func checkTranspose(fn, arg string, t Transpose) bool {
	switch t {
	case NoTrans:
		return false
	case Trans:
		return true
	}
	panic("keel: " + fn + ": " + arg + " is neither NoTrans nor Trans")
}

// checkMatrix validates one row-major matrix argument: the row stride must hold a
// row, and the slice must reach the last element of the last row.
//
// The length needed is (rows-1)*ld + cols, not rows*ld: a caller may legitimately
// pass a slice that stops at the end of the last row rather than at the end of
// its stride, and demanding the padding would reject a valid submatrix view.
//
// ld >= max(1, cols) is checked even for an empty matrix, matching reference
// BLAS's LDA >= MAX(1,...) tests, because an ld of zero is a caller bug whether or
// not this particular call would dereference it.
func checkMatrix(fn, arg string, rows, cols int, v []float32, ld int) {
	if ld < 1 || ld < cols {
		panic("keel: " + fn + ": ld" + arg + " < max(1, columns of " + arg + ")")
	}
	if rows == 0 || cols == 0 {
		return // empty matrix; v may be nil
	}
	if need := (rows-1)*ld + cols; len(v) < need {
		panic("keel: " + fn + ": " + arg + " shorter than its dimensions and ld" + arg + " require")
	}
}

// absf32 clears the sign bit rather than branching, so -0 becomes +0 and a NaN
// keeps its payload — identical to internal/l1.absf32 and vec.ScalarAbs. The
// strided paths above need it and must not import l1 for one scalar op.
func absf32(v float32) float32 {
	return math.Float32frombits(math.Float32bits(v) &^ (1 << 31))
}

// Level 2 — Phase P4.

func Sgemv(tA Transpose, m, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	panic(nyi("Sgemv", "P4"))
}
func Sger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	panic(nyi("Sger", "P4"))
}

// Level 3 — Phases P3 (Sgemm) and P4 (derived).

// Sgemm computes C = alpha*op(A)*op(B) + beta*C for row-major matrices, where
// op(X) is X or Xᵀ. op(A) is m×k, op(B) is k×n and C is m×n; lda, ldb and ldc
// are row strides in elements and must each be at least the number of columns of
// the array they describe (before any transpose — lda bounds a's own rows).
//
// Argument errors panic, as everywhere else in keel. A zero m, n or k is not an
// error: k == 0 is the empty product, so C = beta*C, and m or n == 0 does
// nothing at all. A matrix whose declared shape is empty may be nil; one whose
// declared shape is not empty must be long enough for it even on a call that
// will not read it.
//
// alpha == 0 gives C = beta*C without reading A or B, matching reference SGEMM:
// a NaN or an infinity in A must not reach C through a multiply by zero. beta == 0
// writes zeros without reading C, also matching reference, so an uninitialized C
// is a legitimate destination for beta = 0.
//
// The summation order is the blocked one — packed panels, KC-deep accumulation
// per tile, alpha folded into the packed A — so results are not bit-identical to
// a textbook triple loop, or to another backend's. Every backend is held to
// DESIGN.md §5's tolerance model against a float64 oracle that folds alpha the
// same way. See internal/block for the loop nest and internal/pack for the panel
// layout.
func Sgemm(tA, tB Transpose, m, n, k int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	ta := checkTranspose("Sgemm", "tA", tA)
	tb := checkTranspose("Sgemm", "tB", tB)
	if m < 0 || n < 0 || k < 0 {
		panic("keel: Sgemm: negative dimension")
	}
	// op(A) is m×k and op(B) is k×n; the stored arrays are those shapes
	// transposed when the flag says so, and it is the stored shape that lda and
	// ldb have to fit.
	ra, ca := m, k
	if ta {
		ra, ca = k, m
	}
	rb, cb := k, n
	if tb {
		rb, cb = n, k
	}
	checkMatrix("Sgemm", "a", ra, ca, a, lda)
	checkMatrix("Sgemm", "b", rb, cb, b, ldb)
	checkMatrix("Sgemm", "c", m, n, c, ldc)
	block.Gemm(activeKern, ta, tb, m, n, k, alpha, a, lda, b, ldb, beta, c, ldc)
}
func Ssyrk(ul Uplo, t Transpose, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int) {
	panic(nyi("Ssyrk", "P4"))
}
func Ssymm(s Side, ul Uplo, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	panic(nyi("Ssymm", "P4"))
}
func Strsm(s Side, ul Uplo, tA Transpose, d Diag, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
	panic(nyi("Strsm", "P4"))
}

func nyi(fn, phase string) string {
	return "keel: " + fn + " not implemented until phase " + phase + " gate is green (see DESIGN.md)"
}
