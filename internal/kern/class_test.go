// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern_test

import (
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/vec"
)

// The selection rule's whole job is to rank the shapes the way KERNEL.md §7
// measured them, so the tests below assert the measured outcome — 2×32 on an
// issue-bound host, 4×32 on an FMA-bound one — against the registry as shipped,
// not against a hand-built table that could agree with a wrong registry.

func TestPreferredPicksTheMeasuredWinnerPerClass(t *testing.T) {
	var vec []kern.Kernel
	for _, k := range kern.Kernels() {
		if k.Name != kern.Scalar {
			vec = append(vec, k)
		}
	}
	if len(vec) == 0 {
		t.Skip("no vector kernels in this build; nothing to rank")
	}
	want := map[kern.Class]string{
		kern.ClassIssue: "2x32", // fewest instructions per FMA
		kern.ClassFMA:   "4x32", // fewest memory ops per FMA
	}
	for _, class := range kern.Classes() {
		k, ok := kern.Preferred(class, vec)
		if !ok {
			t.Fatalf("Preferred(%q, %d kernels) found nothing", class, len(vec))
		}
		if k.Tile() != want[class] {
			t.Errorf("Preferred(%q) = %s, want %s (KERNEL.md §7)", class, k.Tile(), want[class])
		}
	}
}

// TestPreferredIsOrderIndependent is the property that made #24 a bug: the old
// selection returned the first matching entry, so the answer was the registry's
// order. Among audited shapes the rule must give the same answer whatever order
// the candidates arrive in, or a future edit to kern_amd64.go's list silently
// changes what ships.
//
// Audited shapes only. Registry order deciding between *unaudited* shapes is the
// documented behaviour, not a bug — see betterFor and the scalar case at the end
// of TestPreferredNeverPicksAnUnauditedShapeOverAnAuditedOne. So on a build with
// no vector kernels the candidates are synthetic, and the property still gets
// tested rather than skipped.
func TestPreferredIsOrderIndependent(t *testing.T) {
	var ks []kern.Kernel
	for _, k := range kern.Kernels() {
		if k.InsnsPerFMA > 0 {
			ks = append(ks, k)
		}
	}
	if len(ks) < 2 {
		t.Logf("build has %d audited shapes; using synthetic ones", len(ks))
		ks = []kern.Kernel{
			{Name: kern.AVX512, MR: 2, NR: 32, Unroll: 4, InsnsPerFMA: 4.625},
			{Name: kern.AVX512, MR: 4, NR: 32, Unroll: 1, InsnsPerFMA: 6.25},
		}
	}
	rev := make([]kern.Kernel, len(ks))
	for i, k := range ks {
		rev[len(ks)-1-i] = k
	}
	for _, class := range kern.Classes() {
		fwd, ok1 := kern.Preferred(class, ks)
		bwd, ok2 := kern.Preferred(class, rev)
		if !ok1 || !ok2 {
			t.Fatalf("Preferred(%q) found nothing", class)
		}
		if fwd.ID() != bwd.ID() {
			t.Errorf("Preferred(%q) depends on candidate order: %s forward, %s reversed",
				class, fwd.ID(), bwd.ID())
		}
	}
}

// TestPreferredRanksOnTheStatedAxis pins the two orderings to synthetic shapes,
// so a rule that happened to agree with the shipped registry for the wrong
// reason — reading MemOps on both classes, say — still fails.
func TestPreferredRanksOnTheStatedAxis(t *testing.T) {
	// thin issues fewer instructions per FMA; wide reads less memory per FMA.
	// This is the real trade-off, in the direction the shipped shapes have it.
	thin := kern.Kernel{Name: kern.AVX512, MR: 2, NR: 32, Unroll: 4, InsnsPerFMA: 4.625}
	wide := kern.Kernel{Name: kern.AVX512, MR: 4, NR: 32, Unroll: 1, InsnsPerFMA: 6.25}
	if thin.MemOpsPerFMA() <= wide.MemOpsPerFMA() {
		t.Fatalf("premise broken: thin reads %v mem ops/FMA, wide %v; the shapes do not trade off",
			thin.MemOpsPerFMA(), wide.MemOpsPerFMA())
	}
	for _, tc := range []struct {
		class kern.Class
		want  kern.Kernel
	}{
		{kern.ClassIssue, thin},
		{kern.ClassFMA, wide},
	} {
		got, ok := kern.Preferred(tc.class, []kern.Kernel{thin, wide})
		if !ok || got.Tile() != tc.want.Tile() {
			t.Errorf("Preferred(%q) = %s, want %s", tc.class, got.Tile(), tc.want.Tile())
		}
	}
}

// TestPreferredNeverPicksAnUnauditedShapeOverAnAuditedOne is what keeps a
// spilling tile unrankable. ReferenceTile carries no InsnsPerFMA precisely so
// that no arrangement of classes can select it; that property is worth a test
// rather than a comment, because it is one struct literal away from being lost.
func TestPreferredNeverPicksAnUnauditedShapeOverAnAuditedOne(t *testing.T) {
	audited := kern.Kernel{Name: kern.AVX512, MR: 4, NR: 32, Unroll: 1, InsnsPerFMA: 6.25}
	// Leaner on both axes than anything shipped, and unaudited: if the rule read
	// the shape instead of the audit, this would win every class.
	unaudited := kern.Kernel{Name: kern.AVX512, MR: 8, NR: 64, Unroll: 4}
	if unaudited.MemOpsPerFMA() >= audited.MemOpsPerFMA() {
		t.Fatalf("premise broken: the unaudited shape must look better on the other axis")
	}
	for _, class := range kern.Classes() {
		for _, order := range [][]kern.Kernel{
			{audited, unaudited},
			{unaudited, audited},
		} {
			got, ok := kern.Preferred(class, order)
			if !ok {
				t.Fatalf("Preferred(%q) found nothing", class)
			}
			if got.InsnsPerFMA == 0 && order[0].InsnsPerFMA != 0 {
				t.Errorf("Preferred(%q) displaced an audited shape with an unaudited one", class)
			}
		}
	}
	// With only unaudited candidates the answer is registry order, which is how
	// the scalar fallback's shapes reach dispatch unchanged.
	a := kern.ScalarKernel(2, 32)
	b := kern.ScalarKernel(4, 32)
	for _, class := range kern.Classes() {
		got, _ := kern.Preferred(class, []kern.Kernel{a, b})
		if got.Tile() != a.Tile() {
			t.Errorf("Preferred(%q) over unaudited shapes = %s, want first-listed %s",
				class, got.Tile(), a.Tile())
		}
	}
}

func TestPreferredOnNothing(t *testing.T) {
	if _, ok := kern.Preferred(kern.ClassFMA, nil); ok {
		t.Error("Preferred(class, nil) reported a kernel")
	}
}

// TestMemOpsPerFMA checks the arithmetic against hand-computed values for the
// shipped shapes and for a two-vector-wide tile, since 1/MR + Lanes/NR is the
// identity KERNEL.md §3's 0.75 floor is derived from.
func TestMemOpsPerFMA(t *testing.T) {
	const lanes = float64(vec.Lanes)
	for _, tc := range []struct {
		mr, nr int
		want   float64
	}{
		{2, 32, 0.5 + lanes/32},
		{4, 32, 0.25 + lanes/32},
		{8, 64, 0.125 + lanes/64},
	} {
		k := kern.Kernel{MR: tc.mr, NR: tc.nr}
		if got := k.MemOpsPerFMA(); got != tc.want {
			t.Errorf("%dx%d MemOpsPerFMA = %v, want %v", tc.mr, tc.nr, got, tc.want)
		}
	}
	// A degenerate shape reports 0 rather than dividing by zero, which is also
	// what makes it unrankable.
	for _, k := range []kern.Kernel{{MR: 0, NR: 32}, {MR: 4, NR: 0}} {
		if got := k.MemOpsPerFMA(); got != 0 {
			t.Errorf("%dx%d MemOpsPerFMA = %v, want 0", k.MR, k.NR, got)
		}
	}
}

func TestParseClass(t *testing.T) {
	for _, c := range kern.Classes() {
		got, ok := kern.ParseClass(string(c))
		if !ok || got != c {
			t.Errorf("ParseClass(%q) = %q, %v", c, got, ok)
		}
	}
	// Rejected, not defaulted: the KEEL_KERN_CLASS override panics on these, so
	// a run cannot believe it pinned a shape it did not pin.
	for _, s := range []string{"", "FMA", "fma-bound", "issue ", "scalar", "avx512"} {
		if _, ok := kern.ParseClass(s); ok {
			t.Errorf("ParseClass(%q) accepted", s)
		}
	}
}

// TestHostClassAgreesWithItsEvidence checks the two halves of the fingerprint
// cannot drift apart: the class and the string the gate prints beside it come
// from the same bits, and the gate compares that string against its own measured
// verdict. It asserts consistency, not correctness — whether the fingerprint
// classified *this* host right is a measured question, and scripts/gate-p3.sh is
// where it gets asked.
func TestHostClassAgreesWithItsEvidence(t *testing.T) {
	class, ev := kern.HostClass(), kern.HostClassEvidence()
	if _, ok := kern.ParseClass(string(class)); !ok {
		t.Fatalf("HostClass() = %q, which is not a parseable class", class)
	}
	if ev == "" {
		t.Fatal("HostClassEvidence() is empty; the gate has nothing to check the class against")
	}
	t.Logf("keel-kern-host-class: class=%s evidence=%q", class, ev)

	switch class {
	case kern.ClassIssue:
		// Only one bundle produces issue-bound, and it says so.
		if !strings.Contains(ev, "without vbmi2") {
			t.Errorf("class %q with evidence %q: the issue-bound verdict must name the missing bundle", class, ev)
		}
	case kern.ClassFMA:
		if strings.Contains(ev, "without vbmi2") {
			t.Errorf("class %q with evidence %q: that evidence is the issue-bound case", class, ev)
		}
	}
}
