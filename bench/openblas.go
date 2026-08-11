// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build openblas

// The cgo half of DESIGN.md §4/P3's ">= 60% of OpenBLAS at 2048³" reference.
//
// # Why this file is behind a build tag and why that matters
//
// keel must never depend on OpenBLAS, so the binding is guarded by `openblas`,
// which means it is invisible to `go build`, to `go test` without the tag, and to
// the module graph. Nothing keel ships links a BLAS.
//
// It also means this file is compiled only where somebody asked for it, on a
// machine that has the library. scripts/gate-p3.sh builds it natively on the
// configured reference host — see docs/hosts.md, "the OpenBLAS reference host",
// for why this one criterion cannot be evaluated by cross-compiling like
// everything else here, and why that host needs a Go toolchain when no other
// execution host does.
//
// # Why it is a package file rather than part of the benchmark
//
// Go does not allow cgo in _test.go files ("use of cgo in test ... not
// supported"), so the binding lives here and openblas_test.go holds the benchmark
// that calls it. That split is forced by the toolchain, not chosen.
//
// # Why the prototypes are declared here instead of including <cblas.h>
//
// The header's location is not portable: Debian/Ubuntu's libopenblas-dev puts
// cblas.h in the default include path, RHEL's openblas-devel puts it under
// /usr/include/openblas, and Homebrew puts it in its own prefix. Two of those
// three need a -I that is wrong on the other two. The ABI, by contrast, is stable
// and tiny — three functions, all with C-integer arguments in an LP64 build, which
// is what every distribution package is. Declaring them here makes the harness
// build wherever the *library* is, which is the thing that actually has to be
// true.
//
// If a host keeps OpenBLAS somewhere the linker does not look, CGO_LDFLAGS and
// CGO_CFLAGS override the directives below without editing this file.
package bench

/*
#cgo linux LDFLAGS: -lopenblas -lm
#cgo darwin CFLAGS: -I/opt/homebrew/opt/openblas/include
#cgo darwin LDFLAGS: -L/opt/homebrew/opt/openblas/lib -lopenblas

// The Homebrew paths are here to compile-verify this harness on the darwin/arm64
// dev host, which does have OpenBLAS. A number measured there would be keel's
// scalar path against OpenBLAS's NEON kernels: not the P3 criterion, and not
// reported as one. The criterion runs on amd64 (DESIGN.md §7 rule 7).

enum { keelCblasRowMajor = 101, keelCblasNoTrans = 111 };

void cblas_sgemm(int order, int transa, int transb, int m, int n, int k,
                 float alpha, const float *a, int lda, const float *b, int ldb,
                 float beta, float *c, int ldc);
int openblas_get_num_threads(void);
int openblas_get_num_procs(void);
char *openblas_get_config(void);
char *openblas_get_corename(void);
*/
import "C"

import "unsafe"

// openblasConfig is the library's own report of how it was built — target
// microarchitecture, threading model, and whether it is an ILP64 build. Printed
// as provenance: "OpenBLAS" alone does not identify a denominator.
func openblasConfig() string { return C.GoString(C.openblas_get_config()) }

// openblasThreads is the thread count OpenBLAS will actually use, after
// OPENBLAS_NUM_THREADS and OMP_NUM_THREADS have had their say. The gate requires
// it to be 1: a reference measured with sixteen threads against keel's one would
// clear or miss the 60% bar for a reason that has nothing to do with either
// implementation.
func openblasThreads() int { return int(C.openblas_get_num_threads()) }

// openblasProcs is what the library thinks the machine has, printed beside the
// thread count so a pin can be read as a pin: "threads=1 procs=1" is a container
// with one CPU, "threads=1 procs=32" is a genuinely restrained library on a big
// host, and the second is the measurement DESIGN.md §4/P3 asks for.
func openblasProcs() int { return int(C.openblas_get_num_procs()) }

// openblasCorename is the kernel family DYNAMIC_ARCH selected at load time — the
// one piece of the reference's configuration that can silently make it slow.
//
// A build that picks a pre-AVX2 kernel on an AVX-512 host produces a reference
// that reads low, which *inflates* keel's ratio: the direction a gate must never
// fail in, and invisible in every other marker, since the version, the thread
// count and the config string can all look correct while a Nehalem kernel does
// the arithmetic. The gate holds this to an AVX2-or-better target and refuses a
// name it does not recognize.
func openblasCorename() string { return C.GoString(C.openblas_get_corename()) }

// openblasSgemm is cblas_sgemm with keel.Sgemm's argument order and row-major
// convention, so the benchmark's two sides read as the same call. No transposes:
// the comparison DESIGN.md asks for is the untransposed square product.
func openblasSgemm(m, n, k int, alpha float32, a []float32, lda int,
	b []float32, ldb int, beta float32, c []float32, ldc int) {

	C.cblas_sgemm(C.keelCblasRowMajor, C.keelCblasNoTrans, C.keelCblasNoTrans,
		C.int(m), C.int(n), C.int(k),
		C.float(alpha), (*C.float)(unsafe.Pointer(&a[0])), C.int(lda),
		(*C.float)(unsafe.Pointer(&b[0])), C.int(ldb),
		C.float(beta), (*C.float)(unsafe.Pointer(&c[0])), C.int(ldc))
}
