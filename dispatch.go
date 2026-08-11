// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"os"

	"github.com/scttfrdmn/keel/internal/l1"
)

// Backend selection. Order: avx512 → avx2 → scalar, chosen once at init from
// runtime CPU feature detection. Finalized in P5; the Level-1 dispatch below
// is what P1 needs, and P2/P3 will add the kernel tables beside it.
//
// KEEL_FORCE=scalar|avx2|avx512 pins the choice. It exists for testing — gate
// P1 uses it to run the whole suite scalar-only on a machine that *has*
// AVX-512, which is the only way to prove the fallback works rather than
// assuming it. An unavailable value panics at init rather than silently
// downgrading: a test run that believed it was measuring AVX-512 and quietly
// measured scalar is worse than a crash.
const envForce = "KEEL_FORCE"

// activeL1 is the Level-1 kernel set every public L1 routine calls through.
// One indirect call per routine invocation, outside every loop.
var activeL1 = selectL1()

func selectL1() l1.Kernels {
	avail := l1.Backends() // widest first, scalar last
	want, forced := os.LookupEnv(envForce)
	if !forced || want == "" {
		return avail[0]
	}
	for _, b := range avail {
		if b.Name == want {
			return b
		}
	}
	panic("keel: " + envForce + "=" + want + " is not available on this machine; have " +
		joinNames(avail) + " (unset " + envForce + " to auto-select)")
}

func joinNames(bs []l1.Kernels) string {
	s := ""
	for i, b := range bs {
		if i > 0 {
			s += " "
		}
		s += b.Name
	}
	return s
}

// ActiveL1Backend reports which Level-1 backend is dispatched to. Exported for
// the benchmark harness and the gate's coverage markers, not for callers to
// branch on.
func ActiveL1Backend() string { return activeL1.Name }

// AvailableL1Backends reports every Level-1 backend runnable here, widest
// first. Same audience as ActiveL1Backend.
func AvailableL1Backends() []string { return l1.Names() }
