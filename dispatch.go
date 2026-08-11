// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"os"

	"github.com/scttfrdmn/keel/internal/kern"
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

// activeKern is the SGEMM microkernel every Level-3 routine blocks around. One
// indirect call per MR×NR tile, which is thousands of FMAs at the sizes that
// matter and is why the shape can be a runtime value at all.
var activeKern = selectKern()

// selectKern picks the microkernel, honouring KEEL_FORCE with one documented
// asymmetry against selectL1: the kernel table is sparser than the Level-1
// table.
//
// P2 shipped AVX-512 tiles and a scalar reference; there is no AVX2 microkernel
// (internal/kern's package doc explains why the shapes are what they are). So
// KEEL_FORCE=avx2 asks for a Level-3 backend that does not exist. Panicking
// would make a legitimate Level-1 test configuration unable to call Sgemm at
// all, so instead KEEL_FORCE acts as a *ceiling* here and Level 3 runs scalar —
// and ActiveKernBackend reports "scalar", so no benchmark or gate marker can
// believe it measured an AVX2 SGEMM. That is the property the no-silent-downgrade
// rule is protecting; the panic is only the usual way of getting it.
//
// A name that is not a backend at all still panics, in selectL1 above: it is
// declared first, so it initializes first.
func selectKern() kern.Kernel {
	avail := kern.Kernels() // widest tile first, scalar references last
	want, forced := os.LookupEnv(envForce)
	if !forced || want == "" {
		return avail[0]
	}
	for _, k := range avail {
		if k.Name == want {
			return k
		}
	}
	for _, k := range avail {
		if k.Name == kern.Scalar {
			return k
		}
	}
	panic("keel: no microkernel available, not even scalar (internal/kern.Kernels is empty)")
}

// ActiveKernBackend reports which SGEMM microkernel backend is dispatched to,
// and ActiveKernTile its shape. Same audience as ActiveL1Backend: the benchmark
// harness and the gate's coverage markers.
func ActiveKernBackend() string { return activeKern.Name }

// ActiveKernTile reports the active microkernel's tile as it appears in gate
// markers and benchmark names, e.g. "4x32".
func ActiveKernTile() string { return activeKern.Tile() }

// AvailableKernels reports every microkernel runnable here, widest tile first,
// as shape/backend identifiers.
func AvailableKernels() []string {
	ks := kern.Kernels()
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.ID()
	}
	return out
}
