// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// The package comment lives in doc.go.
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
// Instead the fast kernel runs unguarded and its *result* is inspected. Either
// outcome (plus NaN, which the same check catches) reruns the whole vector in
// float64, where no rescaling is needed at all: the largest float32 squared is
// ~1.2e77 and float64 reaches ~1.8e308, so float32 inputs cannot overflow a
// float64 accumulator until n exceeds 1e230 elements. The rescue is therefore
// exact-by-construction rather than clever, which is the property worth having
// in a path that runs only on inputs nobody tested.
//
// # Why the underflow test is a threshold and not s == 0
//
// This comment used to claim that "total underflow can only end at exactly 0, so
// one check after the loop sees everything a per-element check would". That was
// wrong, and it was wrong in a way that made the fast path silently inaccurate
// rather than merely slow (#97). Underflow is gradual: a term v*v below ~1.2e-38
// rounds to a *subnormal*, which is neither 0 nor +Inf, so a partially
// underflowed sum sails through a `s > 0 && !IsInf(s)` guard carrying whatever
// relative error the rounding committed. Measured before the fix,
// Snrm2(1, []float32{3e-23}, 1) was off by 24.78%, and the damage decays
// smoothly rather than at an edge — 0.96% at 1e-22, 0.026% at 1e-21.
//
// The replacement guard is a bound, not a probe. Every term v*v is rounded to a
// multiple of the smallest subnormal eta = 2^-149, so it carries an absolute
// error of at most eta/2, and n terms carry at most n·eta/2 between them. The
// relative error in s is therefore at most n·eta/(2s), which is below float32's
// eps = 2^-23 exactly when s >= n·eta/(2·eps) = n·2^-127. Taking the guard one
// binade stricter at n·2^-126 — the smallest *normal* float32 times n — bounds
// the error at eps/2 and keeps the constant recognisable.
//
// Two properties worth stating because they are what make this cheap. It is an
// absolute-error argument, so it holds for any distribution of magnitudes: one
// ordinary element among subnormal ones dominates s and the tiny terms' rounding
// is correctly judged irrelevant, with no per-element test. And the threshold is
// minuscule — a vector of a million elements takes the fast path whenever its
// norm exceeds ~1e-16 — so the rescue stays rare, which was the whole reason for
// inspecting the result instead of guarding each element.
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
	// An s below n·2^-126 may have lost significance to gradual underflow (see
	// the derivation above); a non-finite s means overflow or a NaN in the data.
	// Both go to the float64 path, as does s == 0 — whether from a genuinely zero
	// vector, which costs one pass to confirm, or from total underflow. NaN fails
	// the comparison rather than passing it, which is the direction that matters:
	// every ordering against NaN is false, so NaN lands in the rescue.
	if s >= float32(n)*0x1p-126 && !math.IsInf(float64(s), 1) {
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

// checkUplo maps a Uplo to the bool the internals use — true for Lower — and
// panics on anything else, for the same reason checkTranspose does: a routine that
// read the wrong triangle would return a plausible answer computed from memory the
// caller never meant it to see.
func checkUplo(fn, arg string, ul Uplo) bool {
	switch ul {
	case Upper:
		return false
	case Lower:
		return true
	}
	panic("keel: " + fn + ": " + arg + " is neither Upper nor Lower")
}

// checkSide maps a Side to true for Left.
func checkSide(fn, arg string, s Side) bool {
	switch s {
	case Left:
		return true
	case Right:
		return false
	}
	panic("keel: " + fn + ": " + arg + " is neither Left nor Right")
}

// checkDiag maps a Diag to true for Unit.
func checkDiag(fn, arg string, d Diag) bool {
	switch d {
	case NonUnit:
		return false
	case Unit:
		return true
	}
	panic("keel: " + fn + ": " + arg + " is neither NonUnit nor Unit")
}

// Level 2 — Phase P4.
//
// Both routines are a loop over A's rows whose body is one Level-1 kernel call on
// a contiguous row: Sgemv's NoTrans case is a dot product per row, its Trans case
// and Sger are an axpy per row. That is the whole design, and the reason is that
// row-major A makes a row the only unit-stride vector in sight — a column-wise
// formulation would gather with stride lda and lose the kernel entirely (see
// internal/l1's package doc on why keel does not vectorize strided access).
//
// A non-unit stride on x or y is handled by gathering it into a contiguous scratch
// buffer once per call rather than by a strided inner loop: the gather is O(len)
// against the O(m·n) of the multiply, so it buys the kernel back for the price of
// one pass. That is the opposite of the Level-1 decision, and the difference is
// the ratio: in Sdot a gather would be the whole cost of the routine.

// Sgemv computes y = alpha·op(A)·x + beta·y, where a holds A as m×n row-major
// regardless of tA, so lda >= n always. op(A)·x is m long for NoTrans and n long
// for Trans, and x is the other one.
//
// Argument errors panic, as everywhere in keel. Either stride may be negative,
// following reference SGEMV, in which case that vector runs backwards from its far
// end.
//
// # Two deliberate deviations from reference SGEMV
//
// Reference returns without touching y when m == 0, n == 0, or alpha == 0 and
// beta == 1. keel returns y = beta·y when the reduction is empty, because that is
// the value of the expression: the empty product is zero, exactly as documented for
// Sgemm's k == 0, and a caller who scales by beta through a loop whose n reaches
// zero on the last iteration should not get a different rule at the boundary. When
// there are no output elements at all — m == 0 for NoTrans — there is nothing to
// write and the routines agree.
//
// alpha is applied to the dot product rather than folded into A one element at a
// time, which is not what Sgemm does (internal/pack folds it) and not what the
// oracle does. The difference is one rounding on a value the tolerance model
// already carries n+1 of, so it is covered rather than ignored — but it is a real
// difference, and it is here because folding alpha per element would cost a pass
// over A that a Level-2 routine, which touches each element of A exactly once,
// cannot amortize.
func Sgemv(tA Transpose, m, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	trans := checkTranspose("Sgemv", "tA", tA)
	if m < 0 || n < 0 {
		panic("keel: Sgemv: negative dimension")
	}
	checkMatrix("Sgemv", "a", m, n, a, lda)
	rows, inner := m, n
	if trans {
		rows, inner = n, m
	}
	checkVector("Sgemv", "x", inner, x, incX)
	checkVector("Sgemv", "y", rows, y, incY)
	if rows == 0 {
		return
	}
	if inner == 0 || alpha == 0 {
		scaleStrided(beta, rows, y, incY)
		return
	}
	if trans {
		gemvTrans(m, n, alpha, a, lda, x, incX, beta, y, incY)
		return
	}
	xs := gather(inner, x, incX)
	iy := offset(rows, incY)
	for i := 0; i < m; i++ {
		d := alpha * activeL1.Dot(a[i*lda:i*lda+n], xs)
		switch beta {
		case 0:
			y[iy+i*incY] = d
		case 1:
			y[iy+i*incY] += d
		default:
			y[iy+i*incY] = d + beta*y[iy+i*incY]
		}
	}
}

// gemvTrans is Sgemv's y = alpha·Aᵀ·x + beta·y, split out because its shape is the
// other one: the output is as long as a row of A, so beta is applied to the whole
// of y up front and each row of A then contributes an axpy.
//
// That ordering is why beta is not folded in afterwards: y = beta·y first means
// the accumulation runs on the final destination and beta == 0 never reads y,
// which is the case a caller relies on when y is uninitialized.
func gemvTrans(m, n int, alpha float32, a []float32, lda int, x []float32, incX int,
	beta float32, y []float32, incY int) {

	ys, scattered := y, false
	if incY != 1 {
		// n accumulations per element would each pay the stride; one gather and one
		// scatter pay it twice in total.
		ys, scattered = make([]float32, n), true
		if beta != 0 {
			iy := offset(n, incY)
			for j := 0; j < n; j++ {
				ys[j] = y[iy+j*incY]
			}
		}
	}
	scaleStrided(beta, n, ys, 1)
	ix := offset(m, incX)
	for i := 0; i < m; i++ {
		// No `if s != 0` guard, deliberately. Skipping a zero row scale would be
		// free and would change the answer: 0·Inf is NaN, so a guard makes y
		// depend on which of two mathematically equal formulations ran. Reference
		// SGER has such a guard and reference SGEMV does not; keel takes the
		// unguarded rule for both, because it is the one a float64 oracle can
		// check element by element (see internal/oracle.GemvEntry).
		activeL1.Axpy(alpha*x[ix+i*incX], a[i*lda:i*lda+n], ys[:n])
	}
	if scattered {
		iy := offset(n, incY)
		for j := 0; j < n; j++ {
			y[iy+j*incY] = ys[j]
		}
	}
}

// Sger computes A += alpha·x·yᵀ for A m×n row-major, x m long and y n long.
// Either stride may be negative. alpha == 0 returns without reading x or y,
// matching reference SGER — the same rule as Saxpy's, and load-bearing for the
// same reason.
//
// Reference SGER additionally skips a row whose scale is zero; keel does not. See
// gemvTrans for why not.
func Sger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	if m < 0 || n < 0 {
		panic("keel: Sger: negative dimension")
	}
	checkVector("Sger", "x", m, x, incX)
	checkVector("Sger", "y", n, y, incY)
	checkMatrix("Sger", "a", m, n, a, lda)
	if m == 0 || n == 0 || alpha == 0 {
		return
	}
	ys := gather(n, y, incY)
	ix := offset(m, incX)
	for i := 0; i < m; i++ {
		activeL1.Axpy(alpha*x[ix+i*incX], ys, a[i*lda:i*lda+n])
	}
}

// gather returns n elements of a strided BLAS vector as a contiguous slice,
// returning v itself when the stride is already one so the common case allocates
// nothing.
func gather(n int, v []float32, inc int) []float32 {
	if inc == 1 {
		return v[:n]
	}
	out := make([]float32, n)
	iv := offset(n, inc)
	for j := 0; j < n; j++ {
		out[j] = v[iv+j*inc]
	}
	return out
}

// scaleStrided applies y = beta·y to n elements of a strided vector, in the same
// three variants as internal/block's C scaling and for the same reason: beta == 0
// must write zeros without reading y, so that an uninitialized destination is
// legal.
func scaleStrided(beta float32, n int, y []float32, inc int) {
	if beta == 1 {
		return
	}
	iy := offset(n, inc)
	if beta == 0 {
		for j := 0; j < n; j++ {
			y[iy+j*inc] = 0
		}
		return
	}
	for j := 0; j < n; j++ {
		y[iy+j*inc] *= beta
	}
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

// The three routines below are derivations on Sgemm's loop nest rather than
// implementations beside it: same packing, same blocking parameters, same beta
// variants, same microkernel — see internal/block/tri.go for each one's reduction
// and for what that inheritance costs. Their argument checking is here, with
// Sgemm's, because the shapes are this package's contract with its caller.

// Ssyrk computes C = alpha·A·Aᵀ + beta·C (NoTrans, A n×k) or C = alpha·Aᵀ·A +
// beta·C (Trans, A k×n), for C symmetric n×n stored in the triangle ul names.
//
// Only that triangle is read or written. The other one is left exactly as the
// caller passed it — not zeroed, not mirrored — which is reference SSYRK's
// contract and the reason a caller may keep two matrices packed in one array.
//
// k == 0 and alpha == 0 both give C = beta·C over the referenced triangle without
// reading A, as in Sgemm. Note that C's own diagonal is part of the referenced
// triangle for either ul.
func Ssyrk(ul Uplo, t Transpose, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int) {
	lower := checkUplo("Ssyrk", "ul", ul)
	trans := checkTranspose("Ssyrk", "t", t)
	if n < 0 || k < 0 {
		panic("keel: Ssyrk: negative dimension")
	}
	ra, ca := n, k
	if trans {
		ra, ca = k, n
	}
	checkMatrix("Ssyrk", "a", ra, ca, a, lda)
	checkMatrix("Ssyrk", "c", n, n, c, ldc)
	block.Syrk(activeKern, lower, trans, n, k, alpha, a, lda, beta, c, ldc)
}

// Ssymm computes C = alpha·A·B + beta·C (Left) or C = alpha·B·A + beta·C (Right),
// where A is symmetric and stored in the triangle ul names. A is m×m for Left and
// n×n for Right; B and C are both m×n.
//
// A's other triangle is never read, so it may hold anything at all — that is the
// property the caller is buying, and internal/block.Symm reflects rather than
// reads. C is a general matrix here, unlike Ssyrk's: all m·n entries are written.
//
// alpha == 0 gives C = beta·C without reading A or B.
func Ssymm(s Side, ul Uplo, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	left := checkSide("Ssymm", "s", s)
	lower := checkUplo("Ssymm", "ul", ul)
	if m < 0 || n < 0 {
		panic("keel: Ssymm: negative dimension")
	}
	d := n
	if left {
		d = m
	}
	checkMatrix("Ssymm", "a", d, d, a, lda)
	checkMatrix("Ssymm", "b", m, n, b, ldb)
	checkMatrix("Ssymm", "c", m, n, c, ldc)
	block.Symm(activeKern, left, lower, m, n, alpha, a, lda, b, ldb, beta, c, ldc)
}

// Strsm solves op(A)·X = alpha·B (Left) or X·op(A) = alpha·B (Right) for X and
// overwrites B with it. A is triangular, stored in the triangle ul names, m×m for
// Left and n×n for Right; B is m×n.
//
// d == Unit means A's stored diagonal is NOT REFERENCED and is taken to be 1,
// which is a stronger statement than "it contains ones": a caller holding an LU
// factorization in one array has L's unit diagonal overwritten by U's, and a
// routine that read it would silently solve a different system. keel does not read
// it, and the tests poison it to prove that.
//
// alpha == 0 sets B to zero and returns without reading A, as reference STRSM
// does. A must be nonsingular in its referenced triangle; keel does not check
// that, and neither does reference BLAS — a zero on the diagonal produces Inf or
// NaN in B rather than an error.
func Strsm(s Side, ul Uplo, tA Transpose, d Diag, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int) {
	left := checkSide("Strsm", "s", s)
	lower := checkUplo("Strsm", "ul", ul)
	trans := checkTranspose("Strsm", "tA", tA)
	unit := checkDiag("Strsm", "d", d)
	if m < 0 || n < 0 {
		panic("keel: Strsm: negative dimension")
	}
	da := n
	if left {
		da = m
	}
	checkMatrix("Strsm", "a", da, da, a, lda)
	checkMatrix("Strsm", "b", m, n, b, ldb)
	block.Trsm(activeKern, left, lower, trans, unit, m, n, alpha, a, lda, b, ldb)
}
