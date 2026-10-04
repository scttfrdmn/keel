// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build darwin && cgo && accelerate

// The cgo half of #175's M-series dual reference: Apple's Accelerate beside
// keel's own NEON roofline.
//
// # Why this is a SECOND reference and not a second number
//
// The whole point of #175 is that an Accelerate ratio and a percent-of-NEON-peak
// are **different quantities**. Apple's matmul path may dispatch to the
// undocumented AMX/AME coprocessor, which keel cannot target from Go and which is
// **not on the NEON roofline at all**. So a table printing the two side by side
// without saying so invites the reading that keel is losing to NEON code when it
// is losing to a different execution unit — §5 rule 16, and the
// two-terms-comparable-only-at-their-rates rule. accelerate_test.go carries that
// disclosure in the caption rather than a footnote, and the ratio is named
// `vs-accelerate` rather than anything suggesting a roofline.
//
// # Why it mirrors openblas.go rather than inventing a shape
//
// Same tag discipline: keel must never link a BLAS, so `accelerate` keeps this
// invisible to `go build`, to untagged `go test`, and to the module graph. Same
// reason for a package file rather than a _test.go — Go rejects cgo in test
// files. Same reason for declaring the prototype here rather than including
// <Accelerate/Accelerate.h>: recent macOS SDKs gate parts of that header behind
// ACCELERATE_NEW_LAPACK/ACCELERATE_LAPACK_ILP64, so including it makes the build
// depend on which SDK generation is installed, while the CBLAS ABI this needs is
// four decades stable and one function wide. Declaring it makes the harness build
// wherever the *framework* is, which on darwin is always.
//
// # The threading problem, which has no read-back and so is measured
//
// DESIGN.md §4/P3 requires a reference's thread count to be *verified from the
// library's own report*, not merely requested — `openblas_get_num_threads()` is
// why that is possible for OpenBLAS. **Accelerate exposes no such call.**
// `VECLIB_MAXIMUM_THREADS` is the documented request and there is no documented
// way to read back what was honoured.
//
// So the precondition is established empirically instead, per §5 rule 26: the
// harness measures Accelerate at the capped and uncapped settings in the same
// run, and the *difference between them* is the witness that the cap arrived. A
// null there means either the cap did nothing or the library was single-threaded
// anyway — two causes the harness cannot separate, which it says rather than
// picking one. An unverifiable precondition is disclosed, never assumed.
package bench

/*
#cgo LDFLAGS: -framework Accelerate

enum { keelCblasRowMajor = 101, keelCblasNoTrans = 111 };

// Declared rather than included; see the file comment. Accelerate's CBLAS uses
// int for the integer arguments in the LP64 build every macOS ships.
void cblas_sgemm(int order, int transa, int transb, int m, int n, int k,
                 float alpha, const float *a, int lda, const float *b, int ldb,
                 float beta, float *c, int ldc);
*/
import "C"

import "unsafe"

// accelerateSgemm is cblas_sgemm with keel.Sgemm's argument order and row-major
// convention, so the benchmark's two sides read as the same call. No transposes:
// the comparison is the untransposed square product, as with the OpenBLAS arm.
func accelerateSgemm(m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	C.cblas_sgemm(C.keelCblasRowMajor, C.keelCblasNoTrans, C.keelCblasNoTrans,
		C.int(m), C.int(n), C.int(k),
		C.float(alpha), (*C.float)(unsafe.Pointer(&a[0])), C.int(lda),
		(*C.float)(unsafe.Pointer(&b[0])), C.int(ldb),
		C.float(beta), (*C.float)(unsafe.Pointer(&c[0])), C.int(ldc))
}
