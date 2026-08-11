// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package oracle holds float64 reference implementations of every public
// routine (P1 onward) and the single tolerance model from DESIGN.md §5.
package oracle

// Tolerance returns the allowed absolute error for a length-n reduction
// with the given magnitude scale: C·f(n)·eps32·scale. It is the ONLY
// place epsilons live; tests must not carry their own.
func Tolerance(n int, scale float64) float64 {
	const eps32 = 1.1920929e-07 // 2^-23
	const c = 4.0               // slack for FMA/reassociation
	fn := float64(n)
	if fn < 1 {
		fn = 1
	}
	return c * fn * eps32 * scale
}
