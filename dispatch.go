// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"os"
	"strconv"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/l1"
)

// Backend selection, chosen once at init from runtime CPU feature detection.
//
// # The chain is per level, and Level 3 has two rungs, not three
//
// Level 1 dispatches avx512 → avx2 → scalar. Level 3 — the SGEMM microkernel and
// everything blocked around it — dispatches avx512 → scalar. That asymmetry is a
// ruling (2026-08-12, issue #40), not an accident of what got written first, and
// the reason is evidentiary: internal/kern has no AVX2 microkernel, and no host
// this project measures on is AVX2-only silicon. KEEL_FORCE=avx2 on an AVX-512
// machine establishes correctness under forced narrowing but says nothing about
// performance on a part that lacks AVX-512, so a three-rung Level-3 chain would
// advertise a middle link no gate could back. Level 1 keeps its AVX2 path because
// that one is measured, and has been gated since P1.
//
// The AVX2 microkernel is deferred rather than dropped, and its unblocking
// condition is named on #40: an AVX2-native evidentiary host. Debt with a trigger.
//
// KEEL_FORCE=scalar|avx2|avx512 pins the choice. It exists for testing — gate
// P1 uses it to run the whole suite scalar-only on a machine that *has*
// AVX-512, which is the only way to prove the fallback works rather than
// assuming it. An unavailable value panics at init rather than silently
// downgrading: a test run that believed it was measuring AVX-512 and quietly
// measured scalar is worse than a crash.
const envForce = "KEEL_FORCE"

// L1Chain and KernChain report the *advertised* dispatch chains: the claim the
// documentation makes about what this library will try, in order, on a machine
// that has everything. AvailableL1Backends and AvailableKernels answer a
// different question — what is runnable *here* — and on a host without AVX-512
// they are properly shorter. The gate checks the advertised chains against
// DESIGN.md §4/P5 and checks that neither one advertises a rung with no
// implementation behind it, which is how #40 was found: keeping the claim in a
// function means a gate can read it, where a claim in prose can only be believed.
func L1Chain() []string { return []string{l1.AVX512, l1.AVX2, l1.Scalar} }

// KernChain reports the advertised Level-3 chain. Two rungs by ruling; see the
// envForce comment above for why the middle one is absent.
func KernChain() []string { return []string{kern.AVX512, kern.Scalar} }

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
// (internal/kern's package doc explains why the shapes are what they are), and as
// of the #40 ruling the Level-3 chain does not claim one — see KernChain. So
// KEEL_FORCE=avx2 asks for a Level-3 backend that does not exist. Panicking
// would make a legitimate Level-1 test configuration unable to call Sgemm at
// all, so instead KEEL_FORCE acts as a *ceiling* here and Level 3 runs scalar —
// and ActiveKernBackend reports "scalar", so no benchmark or gate marker can
// believe it measured an AVX2 SGEMM. That is the property the no-silent-downgrade
// rule is protecting; the panic is only the usual way of getting it.
//
// A name that is not a backend at all still panics, in selectL1 above: it is
// declared first, so it initializes first.
//
// # The shape, as opposed to the backend
//
// Which *shape* of that backend runs is a second decision, and it is per-host.
// Both shipped tiles are zero-spill and neither dominates: the load-lean 4×32 wins
// on Zen 4 and Zen 5, the instruction-lean 2×32 wins on Skylake-X by 11 percentage
// points (KERNEL.md §7). This function used to take the first entry of the registry
// and therefore shipped 4×32 everywhere, which was wrong on one of three hosts —
// issue #24, and the ruling there is that shape selection uses the same
// issue-bound/FMA-bound classification P2's gate model already defines.
//
// So: internal/kern classifies the host (kern.HostClass, a documented feature-bundle
// fingerprint because archsimd exposes no µarch identity — T14/#25) and ranks the
// candidate shapes for that class (kern.Preferred). Nothing here measures anything
// at init; auto-tuning is P5's subject.
//
// KEEL_KERN_CLASS=fma|issue overrides the classification, which is how the gate
// measures the shape it did *not* choose on each host and requires the chosen one to
// be no slower. An unrecognized value panics rather than falling back, for the same
// reason KEEL_FORCE does: a run that believed it had pinned a shape and quietly
// measured the other one is worse than a crash.
func selectKern() kern.Kernel {
	avail := kern.Kernels() // widest tile first, scalar references last
	want, forced := os.LookupEnv(envForce)
	backend := avail[0].Name
	if forced && want != "" {
		backend = ""
		for _, k := range avail {
			if k.Name == want {
				backend = want
				break
			}
		}
		if backend == "" {
			// KEEL_FORCE names a backend with no microkernel — avx2 today. It
			// acts as a ceiling at Level 3 rather than a panic, so that a
			// legitimate Level-1 configuration can still call Sgemm, and
			// ActiveKernBackend reports what actually ran.
			backend = kern.Scalar
		}
	}
	var cand []kern.Kernel
	for _, k := range avail {
		if k.Name == backend {
			cand = append(cand, k)
		}
	}
	k, ok := kern.Preferred(activeKernClass, cand)
	if !ok {
		panic("keel: no microkernel available for backend " + backend +
			" (internal/kern.Kernels has none, not even scalar)")
	}
	return k
}

// activeKernClass is the host classification shape selection is made from, and it
// is a variable rather than a call so that both it and the kernel chosen under it
// can be reported together: a marker saying "2x32 on an issue-bound host" is
// checkable against the gate's own measured classification, where "2x32" alone is
// not.
var activeKernClass = selectKernClass()

const envKernClass = "KEEL_KERN_CLASS"

func selectKernClass() kern.Class {
	want, forced := os.LookupEnv(envKernClass)
	if !forced || want == "" {
		return kern.HostClass()
	}
	c, ok := kern.ParseClass(want)
	if !ok {
		names := ""
		for i, v := range kern.Classes() {
			if i > 0 {
				names += "|"
			}
			names += string(v)
		}
		panic("keel: " + envKernClass + "=" + want + " is not a kernel class; want " +
			names + " (unset " + envKernClass + " to classify this host)")
	}
	return c
}

// ActiveKernBackend reports which SGEMM microkernel backend is dispatched to,
// and ActiveKernTile its shape. Same audience as ActiveL1Backend: the benchmark
// harness and the gate's coverage markers.
func ActiveKernBackend() string { return activeKern.Name }

// ActiveKernTile reports the active microkernel's tile as it appears in gate
// markers and benchmark names, e.g. "4x32".
func ActiveKernTile() string { return activeKern.Tile() }

// ActiveKernClass reports the host classification the tile was chosen under —
// "fma" or "issue" — and ActiveKernClassEvidence the grounds for it. Both are
// printed as provenance so that the gate can compare this classification against
// the one it measures on the same host, rather than taking the fingerprint's word
// for it (docs/toolchain-notes.md T14).
func ActiveKernClass() string { return string(activeKernClass) }

// ActiveKernClassEvidence describes how the class was arrived at: the feature
// bundle read, or the override that replaced it.
func ActiveKernClassEvidence() string {
	if want, forced := os.LookupEnv(envKernClass); forced && want != "" {
		return envKernClass + "=" + want + " (override; fingerprint says " +
			string(kern.HostClass()) + ")"
	}
	return kern.HostClassEvidence()
}

// ActiveKernInsnsPerFMA reports the audited instructions per FMA recorded for the
// dispatched shape, 0 for an unaudited one. The gate cross-checks it against the
// spill audit, so a stale number in the registry cannot survive a gate run.
func ActiveKernInsnsPerFMA() float64 { return activeKern.InsnsPerFMA }

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

// KernelAudits reports the audited instructions per FMA recorded for every
// shipped shape that has one, as "shape/backend=value" pairs.
//
// It exists so the gate can check the registry against the object code: those
// numbers are the input to shape selection, and a measurement recorded in source
// drifts unless something recomputes it. scripts/gate-p3.sh disassembles each
// shape's loop body, recounts, and fails on disagreement — so the ranking cannot
// come to rest on a stale count. Unaudited shapes are absent rather than listed
// as zero: there is nothing to check them against.
func KernelAudits() []string {
	var out []string
	for _, k := range kern.Kernels() {
		if k.InsnsPerFMA > 0 {
			out = append(out, k.ID()+"="+strconv.FormatFloat(k.InsnsPerFMA, 'f', 3, 64))
		}
	}
	return out
}
