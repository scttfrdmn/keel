// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package keel is a pure-Go float32 BLAS subset whose hot kernels target the
// experimental simd/archsimd packages (GOEXPERIMENT=simd). See DESIGN.md.
//
// Status: pre-alpha skeleton. Every routine below is a stub until its phase
// gate is green; see DESIGN.md §4 and the GitHub milestones.
package keel

// Level 1 — Phase P1.

func Sdot(n int, x []float32, incX int, y []float32, incY int) float32 { panic(nyi("Sdot", "P1")) }
func Saxpy(n int, alpha float32, x []float32, incX int, y []float32, incY int) { panic(nyi("Saxpy", "P1")) }
func Sscal(n int, alpha float32, x []float32, incX int) { panic(nyi("Sscal", "P1")) }
func Snrm2(n int, x []float32, incX int) float32 { panic(nyi("Snrm2", "P1")) }
func Sasum(n int, x []float32, incX int) float32 { panic(nyi("Sasum", "P1")) }
func Isamax(n int, x []float32, incX int) int { panic(nyi("Isamax", "P1")) }

// Level 2 — Phase P4.

func Sgemv(tA Transpose, m, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int) {
	panic(nyi("Sgemv", "P4"))
}
func Sger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int) {
	panic(nyi("Sger", "P4"))
}

// Level 3 — Phases P3 (Sgemm) and P4 (derived).

// Sgemm computes C = alpha*op(A)*op(B) + beta*C for row-major matrices.
func Sgemm(tA, tB Transpose, m, n, k int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int) {
	panic(nyi("Sgemm", "P3"))
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
