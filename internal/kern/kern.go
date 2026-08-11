// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package kern holds the microkernels (P2). Straight-line code, one
// shim op per statement, no calls inside the K-loop, pre-sliced panels.
// Tile protocol and spill frontier are recorded in KERNEL.md at repo root
// once P2 begins.
package kern
