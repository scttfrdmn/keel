// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// The examples are in the external test package so they read the way a caller
// writes them, qualified with the package name.
//
// Every one of them uses small integer-valued matrices. That is not decoration:
// integers up to 2^24 are exact in float32 and so is any sum and product of them
// that stays in range, which makes the `// Output:` lines exact rather than
// rounded. An example whose output depended on the summation order would be an
// example that fails on one backend, and the point of putting the documentation
// in Example functions is that `go test` compiles and runs them — so a call that
// stops matching this package cannot sit in the docs looking correct
// (DESIGN.md §5 rule 8).
package keel_test

import (
	"fmt"

	"github.com/scttfrdmn/keel"
)

func ExampleSgemm() {
	// C = A·B for A 2×3 and B 3×2, so C is 2×2. Each matrix is tightly packed,
	// which makes every leading dimension its own row length.
	a := []float32{
		1, 2, 3,
		4, 5, 6,
	}
	b := []float32{
		7, 8,
		9, 10,
		11, 12,
	}
	c := make([]float32, 2*2)

	keel.Sgemm(keel.NoTrans, keel.NoTrans,
		2, 2, 3, // m, n, k: op(A) is m×k, op(B) is k×n, C is m×n
		1, a, 3, // alpha, A, lda
		b, 2, // B, ldb
		0, c, 2) // beta, C, ldc

	fmt.Println(c)
	// Output: [58 64 139 154]
}

func ExampleSgemm_submatrix() {
	// A leading dimension larger than the row length names a submatrix, with no
	// copy. Here the 2×2 top-left block of a 3-wide array is multiplied by the
	// identity, so C comes out holding just that block: lda is 3, not 2.
	a := []float32{
		1, 2, 99,
		3, 4, 99,
		99, 99, 99,
	}
	id := []float32{
		1, 0,
		0, 1,
	}
	c := make([]float32, 2*2)

	keel.Sgemm(keel.NoTrans, keel.NoTrans, 2, 2, 2, 1, a, 3, id, 2, 0, c, 2)

	fmt.Println(c)
	// Output: [1 2 3 4]
}

func ExampleSaxpy() {
	// y += alpha*x, elementwise, both vectors contiguous (incX and incY of 1).
	x := []float32{1, 2, 3, 4}
	y := []float32{10, 20, 30, 40}

	keel.Saxpy(len(x), 2, x, 1, y, 1)

	fmt.Println(y)
	// Output: [12 24 36 48]
}

func ExampleSaxpy_stride() {
	// A stride reaches every incX'th element. This adds x into every other
	// element of y, leaving the rest untouched: n is the number of elements
	// touched, never the length of the slice.
	x := []float32{1, 2, 3}
	y := []float32{0, -1, 0, -1, 0}

	keel.Saxpy(3, 1, x, 1, y, 2)

	fmt.Println(y)
	// Output: [1 -1 2 -1 3]
}

func ExampleTranspose() {
	// The flag says how a matrix argument is read, not how it is stored. a holds
	// the same 2×3 row-major matrix in both calls below, and m and n are always
	// its stored shape — so lda is 3 either way.
	a := []float32{
		1, 2, 3,
		4, 5, 6,
	}

	// NoTrans: op(A) is the stored 2×3, so x has 3 elements and y has 2.
	x := []float32{1, 2, 3}
	y := make([]float32, 2)
	keel.Sgemv(keel.NoTrans, 2, 3, 1, a, 3, x, 1, 0, y, 1)
	fmt.Println("A·x  =", y)

	// Trans: op(A) is the 3×2 transpose, so the lengths swap.
	xt := []float32{1, 1}
	yt := make([]float32, 3)
	keel.Sgemv(keel.Trans, 2, 3, 1, a, 3, xt, 1, 0, yt, 1)
	fmt.Println("Aᵀ·xt =", yt)

	// Output:
	// A·x  = [14 32]
	// Aᵀ·xt = [5 7 9]
}
