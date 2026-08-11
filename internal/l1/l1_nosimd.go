// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build !goexperiment.simd || !amd64

package l1

// The exact complement of l1_amd64.go's build tag, so there is always exactly
// one definition of vectorBackends and the scalar path builds on a stock
// toolchain on every GOARCH (DESIGN.md §4/P5).
func vectorBackends() []Kernels { return nil }
