// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

import "testing"

// The peak kernels are a *denominator*, so a bug in them does not look like a
// failure — it looks like a percentage. These tests defend the two properties
// that make the number a ceiling and can be checked without a CPU counter.
//
// The third property (no memory operand in the steady-state loop) is checked by
// disassembly in scripts/gate-p2.sh, because no amount of arithmetic can see it.
//
// A fourth property is checked NOWHERE, and it is not hypothetical. Nothing pins the
// instruction count of these loop bodies, so a toolchain change that inflates them
// lowers the measured ceiling and therefore silently *raises* every percent-of-peak
// figure computed against it — the failure mode T8 caused in the other direction, with
// no test to object. Measured while developing the upstream 231-operand rewrite: an
// unconditional form of that rule took avx512Peak's loop from 27 instructions and 0
// copies to 53 and 26, because these kernels carry the accumulator as a *multiplicand*
// and so want 213, the opposite of what a GEMM tile wants. Both tests below stayed
// green, correctly — the arithmetic and the chain count were untouched.
//
// The action that would close it is an insns/FMA criterion for the peak kernels in
// gate-p2.sh, the shape gate-p3.sh already applies to the shipped tiles through
// InsnsPerFMA; it is deferred, not undoable (GitHub #145).

// TestPeakChainsAreIndependent is the CSE guard.
//
// Each accumulator starts at its 1-based chain index and gains exactly 1 per
// iteration, so the exact result is a function of how many chains *survived
// compilation*. When every accumulator started at the same value, the compiler
// merged all twelve AVX-512 chains into one and the loop measured FMA latency
// rather than throughput — a peak roughly 8x too low, which would have inflated
// every percent-of-peak figure computed against it by the same factor
// (docs/toolchain-notes.md T8).
//
// Checking the arithmetic rather than grepping the assembly means this test
// fails the same way on every host and every future compiler, including ones
// that find a new way to collapse the chains.
func TestPeakChainsAreIndependent(t *testing.T) {
	// Every value stays a small exact integer in float32: the largest total is
	// 16*(78+12*1000) = 193248, far below 2^24, so equality is exact and the
	// comparison needs no tolerance. Anything else here would be a numerics
	// question, and this is a compiler question.
	for _, k := range PeakKernels() {
		for _, iters := range []int{0, 1, 2, 3, 17, 1000} {
			got, want := k.Run(iters), k.Witness(iters)
			if got != want {
				t.Errorf("%s: Run(%d) = %v, want %v — %d chains of %d lanes did not "+
					"all survive compilation; the measured peak is not a ceiling",
					k.Name, iters, got, want, k.Chains, k.Lanes)
			}
		}
	}
}

// TestPeakKernelsSelfConsistent pins the flop accounting to the kernel shape.
//
// FlopsPerIter is the numerator of every peak measurement. If a chain count or a
// lane count is edited without it, the peak silently changes by that ratio.
func TestPeakKernelsSelfConsistent(t *testing.T) {
	ks := PeakKernels()
	if len(ks) == 0 {
		t.Fatal("PeakKernels returned nothing; the scalar ceiling is always measurable")
	}
	if last := ks[len(ks)-1]; last.Name != BackendScalar {
		t.Errorf("last kernel is %q, want %q — the scalar reference must always be "+
			"present and last, as in l1.Backends", last.Name, BackendScalar)
	}
	for _, k := range ks {
		if k.Run == nil {
			t.Errorf("%s: nil Run", k.Name)
			continue
		}
		if want := k.Chains * k.Lanes * 2; k.FlopsPerIter != want {
			t.Errorf("%s: FlopsPerIter = %d, want %d (%d chains x %d lanes x 2 flops)",
				k.Name, k.FlopsPerIter, want, k.Chains, k.Lanes)
		}
		// Latency x ports is about 8 on every microarchitecture keel targets; a
		// kernel with fewer chains than that measures latency, not throughput.
		if k.Chains < 8 {
			t.Errorf("%s: %d chains is too few to cover FMA latency x ports (~8); "+
				"this would measure latency and understate the ceiling", k.Name, k.Chains)
		}
	}
	if got := peakOne(); got != 1 {
		t.Errorf("peakOne() = %v, want 1", got)
	}
}
