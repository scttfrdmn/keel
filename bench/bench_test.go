// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package bench holds the benchmark harness (grown from P1). Every reported
// number must carry its denominator: CPU model, theoretical peak, and the
// OpenBLAS reference when the dev-only cgo harness (build tag `openblas`)
// is available on the machine. See DESIGN.md §5 and §7 rule 7.
package bench
