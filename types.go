// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

// The four BLAS flag types. Each holds the same letter reference BLAS uses, so a
// caller porting a Fortran or CBLAS call site can read the flags across
// unchanged. Every routine rejects a value that is not one of the constants below
// with a panic; there is no "treat an unknown flag as the default" path.

// A Transpose selects whether a routine reads a matrix argument as stored or
// transposed.
//
// There is no ConjTrans: keel is real-valued, and reference SGEMM rejects it too.
type Transpose byte

// The Transpose values. NoTrans reads the matrix as stored; Trans reads it
// transposed.
//
// The flag describes how the argument is *read*, not how it is stored: the
// leading dimension always describes the stored array, so lda bounds the rows of
// a as passed, before any transpose.
const (
	NoTrans Transpose = 'N'
	Trans   Transpose = 'T'
)

// An Uplo names which triangle of a symmetric or triangular matrix argument holds
// the data.
//
// The other triangle is never read and never written. It may hold anything,
// including a second matrix packed into the same array.
type Uplo byte

// The Uplo values. Upper names the triangle on and above the diagonal, Lower the
// triangle on and below it; the diagonal belongs to both.
const (
	Upper Uplo = 'U'
	Lower Uplo = 'L'
)

// A Side selects which operand of a product the special matrix is.
type Side byte

// The Side values. Left puts the symmetric or triangular matrix on the left of
// the product, Right on the right.
const (
	Left  Side = 'L'
	Right Side = 'R'
)

// A Diag states whether a triangular matrix's stored diagonal is to be read.
type Diag byte

// The Diag values. NonUnit reads the stored diagonal; Unit does not read it at
// all and takes every diagonal entry to be 1.
//
// Unit is the stronger statement of the two: it is not "the diagonal contains
// ones" but "the diagonal is not referenced", so a caller holding an LU
// factorization in one array — where L's unit diagonal is overwritten by U's —
// may pass it as-is.
const (
	NonUnit Diag = 'N'
	Unit    Diag = 'U'
)
