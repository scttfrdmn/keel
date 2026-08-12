// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build openblas

// The OpenBLAS reference benchmark for DESIGN.md §4/P3's ">= 60% of OpenBLAS at
// 2048³". The cgo binding it calls lives in openblas.go, under the same build tag,
// because Go rejects cgo in a _test.go file ("use of cgo in test ... not
// supported") — that file's doc explains the split and the prototype declarations.

package bench

import (
	"fmt"
	"testing"
)

// init replaces the provenance line the untagged build prints, with everything
// about the reference that can change the ratio: version and build flags, the
// DYNAMIC_ARCH-selected kernel, the thread count, and the CPU count that thread
// count is a restriction of.
//
// The gate requires threads=1 and an AVX2-or-better corename (DESIGN.md §4/P3, as
// amended). Both failure directions are real and only one is intuitive: a
// sixteen-thread reference would fail keel for a reason that is not keel's, and a
// generic-kernel reference would *pass* keel for a reason that is not keel's
// either. The second is why corename is here.
func init() {
	openblasProvenance = fmt.Sprintf("%s corename=%s threads=%d procs=%d",
		openblasConfig(), openblasCorename(), openblasThreads(), openblasProcs())
}

// BenchmarkOpenBLAS is BenchmarkSgemm's operation, argument for argument: same
// sizes, same row-major order, same alpha and beta, same data. The gate divides
// one median by the other, so any difference here that is not the implementation
// itself would land in that ratio.
func BenchmarkOpenBLAS(b *testing.B) {
	provenance()
	for _, n := range gemmSizes {
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			a, bm, c := makeMat(n, n), makeMat(n, n), makeMat(n, n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				openblasSgemm(n, n, n, 1, a, n, bm, n, 0, c, n)
			}
			// gemmWork, not a second count of the same thing: the P3 ratio divides
			// this rate by BenchmarkSgemm's, and two independently written numerators
			// would agree at these sizes while still being two claims.
			rateWork(b, gemmWork(n))
		})
	}
}
