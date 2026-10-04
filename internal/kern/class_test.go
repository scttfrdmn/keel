// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern_test

import (
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// The selection rule's whole job is to rank the shapes the way KERNEL.md §7
// measured them, so the tests below assert the measured outcome — 2×32 on an
// issue-bound host, 4×32 on an FMA-bound one — against the registry as shipped,
// not against a hand-built table that could agree with a wrong registry.

func TestPreferredPicksTheMeasuredWinnerPerClass(t *testing.T) {
	var vec []kern.Kernel
	for _, k := range kern.Kernels() {
		// Only AUDITED kernels are rankable: Preferred/betterFor rank by InsnsPerFMA and
		// treat 0 (unaudited) as unrankable, so an unaudited kernel is not a candidate for a
		// "measured winner". This is the test's premise, made explicit. Both shipping ISAs
		// now audit their tiles (amd64 from KERNEL.md §7, arm64 from the GB10 witness in
		// kern_arm64.go), so this runs on both; a build with no vector kernels still skips.
		if k.Name != kern.Scalar && k.InsnsPerFMA > 0 {
			vec = append(vec, k)
		}
	}
	if len(vec) == 0 {
		t.Skip("no audited vector kernels in this build; nothing to rank")
	}
	// The expected winner per class comes from the measurement, not from re-reading the
	// registry: amd64's split from KERNEL.md §7 (2×32 issue / 4×32 fma), arm64's from the
	// negative-control witness on castor (GB10), where 4×16 beat 8×8 by +32% at full
	// Sgemm/n=2048 and audits leaner on both axes (5.00<5.75 insns/FMA, 0.5<0.625 mem-ops),
	// so it is the winner under both classes. Keyed off the audited backend, since Kernels()
	// is build-tagged per ISA.
	want := map[kern.Class]string{
		kern.ClassIssue: "2x32", // fewest instructions per FMA
		kern.ClassFMA:   "4x32", // fewest memory ops per FMA
	}
	if vec[0].Name == kern.NEON {
		want = map[kern.Class]string{kern.ClassIssue: "4x16", kern.ClassFMA: "4x16"}
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

// TestMemOpsPerFMA checks the arithmetic against values typed by hand, which is
// the whole point of the rewrite: the previous version computed every `want` as
// an expression over `vec.Lanes` — the same constant the function divides by —
// so it asserted the formula against itself and could not fail for any value of
// the divisor. Its three cases were all AVX-512 shapes, where the Block width
// and the native vector width coincide, so nothing in it was wrong; it simply
// could not see the one backend where they differ. Expectations here are
// decimal literals with the division written in a comment beside them.
//
// The divisor is the backend's NATIVE float32 vector width, not `vec.Lanes`.
// `vec.Lanes` is 16 because a Block is the shim's 16-lane semantic currency on
// every backend (vec.go); one *vector FMA instruction* covers 16 columns on
// AVX-512 and 4 on NEON, and this ratio counts instructions.
func TestMemOpsPerFMA(t *testing.T) {
	for _, tc := range []struct {
		backend string
		mr, nr  int
		want    float64
	}{
		// AVX-512: one Float32x16 per 16 columns, so the native width is 16 and
		// these are the figures KERNEL.md §3's 0.75 floor is derived from. They
		// are byte-identical to what the pre-fix function returned.
		{kern.AVX512, 2, 32, 1.0},   // 1/2 + 16/32
		{kern.AVX512, 4, 32, 0.75},  // 1/4 + 16/32
		{kern.AVX512, 8, 64, 0.375}, // 1/8 + 16/64
		// NEON: one Float32x4 per 4 columns. These are the figures
		// kern_arm64.go's registry comment publishes for the two shipped tiles —
		// the comment had the physics right while the function returned 2.125 and
		// 1.25, four times the B-load term.
		{kern.NEON, 8, 8, 0.625}, // 1/8 + 4/8
		{kern.NEON, 4, 16, 0.5},  // 1/4 + 4/16
		// 3x24 ties 4x16 here EXACTLY — and exactly in float64 too, the two
		// roundings cancelling, so == is the right comparison and not a lucky
		// epsilon. It is the arm64 zero-spill frontier shape (#136).
		{kern.NEON, 3, 24, 0.5}, // 1/3 + 4/24
	} {
		k := kern.Kernel{Name: tc.backend, MR: tc.mr, NR: tc.nr}
		if got := k.MemOpsPerFMA(); got != tc.want {
			t.Errorf("%s %dx%d MemOpsPerFMA = %v, want %v", tc.backend, tc.mr, tc.nr, got, tc.want)
		}
	}
	// A degenerate shape reports 0 rather than dividing by zero, which is also
	// what makes it unrankable. A backend with no native width stated reports 0
	// for the same reason: unknown must not fall through to a default, because
	// the default would be some real ISA's width and would rank a shape on it.
	for _, k := range []kern.Kernel{
		{Name: kern.AVX512, MR: 0, NR: 32},
		{Name: kern.AVX512, MR: 4, NR: 0},
		{Name: kern.Scalar, MR: 4, NR: 32},
		{Name: "sve2", MR: 4, NR: 32},
	} {
		if got := k.MemOpsPerFMA(); got != 0 {
			t.Errorf("%s %dx%d MemOpsPerFMA = %v, want 0", k.Name, k.MR, k.NR, got)
		}
	}
}

// TestMemOpsPerFMAReordersNEONShapes is the witness that the divisor is
// load-bearing rather than a cosmetic 4x. The term it scales is the B-load one
// and not the A-broadcast one, so a wrong divisor REWEIGHTS the two terms
// instead of scaling the ratio — which can reverse the order of two shapes that
// differ in both MR and NR, on the axis betterFor ranks FIRST for ClassFMA,
// which is arm64's class.
//
// 8x12 against 4x16 is such a pair: 0.458 vs 0.500 correctly, 1.458 vs 1.250
// under the Block width. No shipped verdict moved when this was fixed, because
// the two shipped NEON tiles are 8x8 and 4x16 and both divisors agree that 4x16
// wins; 8x12 is excluded from dispatch anyway, for spilling 5 accumulators
// (docs/neon-sweep.md step 3). So this test pins the FUNCTION, not a shipping
// decision — the reorder is what the defect could have done, demonstrated on
// the shapes where it does it.
func TestMemOpsPerFMAReordersNEONShapes(t *testing.T) {
	// Audited counts are required for betterFor to read the memory axis at all,
	// so both arms carry one; the values are the audited figures for these two
	// shapes and the ranking below does not depend on them, since ClassFMA reads
	// MemOps first and these two differ on it.
	lean := kern.Kernel{Name: kern.NEON, MR: 8, NR: 12, Unroll: 1, InsnsPerFMA: 5.0}
	ship := kern.Kernel{Name: kern.NEON, MR: 4, NR: 16, Unroll: 1, InsnsPerFMA: 5.0}
	if lean.MemOpsPerFMA() >= ship.MemOpsPerFMA() {
		t.Fatalf("premise broken: 8x12 reads %v mem ops/FMA and 4x16 reads %v; "+
			"8x12 must be the leaner one on this axis or the reorder is not demonstrated",
			lean.MemOpsPerFMA(), ship.MemOpsPerFMA())
	}
	got, ok := kern.Preferred(kern.ClassFMA, []kern.Kernel{ship, lean})
	if !ok || got.Tile() != lean.Tile() {
		t.Errorf("Preferred(fma) = %s, want %s: the memory axis is not being read at the native width",
			got.Tile(), lean.Tile())
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
