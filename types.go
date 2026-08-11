// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

// Row-major storage throughout; ld* are element strides between rows.

type Transpose byte

const (
	NoTrans Transpose = 'N'
	Trans   Transpose = 'T'
)

type Uplo byte

const (
	Upper Uplo = 'U'
	Lower Uplo = 'L'
)

type Side byte

const (
	Left  Side = 'L'
	Right Side = 'R'
)

type Diag byte

const (
	NonUnit Diag = 'N'
	Unit    Diag = 'U'
)
