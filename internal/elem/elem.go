// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package elem holds the element-type constraint the middle of the nest
// (internal/pack, internal/block) and its seam (internal/kern) are generic over.
//
// It is a package of its own, and of one declaration, because none of the three
// packages that need it can host it without a dependency it does not otherwise
// have: pack imports nothing of keel's, and a constraint living in kern would make
// the packing routines depend on the microkernels for the sake of one type set.
//
// The set is closed and exact — float32 and float64, no ~ — because the middle's
// per-type facts (internal/pack's byte thresholds, internal/block's blocking
// parameters) are stated for those two types and no other.
package elem

// Float is the element type of one instantiation of the nest.
type Float interface{ float32 | float64 }
