// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package kern

// Shape selection: which of the shipped tiles a given host should run.
//
// # Why there is a choice to make at all
//
// KERNEL.md §7 measures both shipped shapes on all three gate hosts, and the
// winner flips: the load-lean 4×32 wins on Zen 4 (96.6% vs 92.4% of measured
// peak) and Zen 5 (64.2% vs 53.1%), and the instruction-lean 2×32 wins on
// Skylake-X (46.0% vs 35.2%). One shape everywhere is therefore wrong on at
// least one machine in this fleet by 11 percentage points, and it was wrong on
// exactly that one: the registry's first entry is 4×32, and `Sgemm` dispatched to
// it unconditionally (issue #24).
//
// This is not an exotic requirement. Per-target microkernel shapes and
// blocksizes are what BLIS and OpenBLAS both carry; "one shape everywhere" was
// never a design principle here either, only the skeleton's simplicity.
//
// # The rule, and why it is the classifier P2 already defined
//
// P2's throughput model (DESIGN.md §4/P2 as amended on issue #19,
// scripts/roofline.sh) divides hosts by what binds their K-loop:
//
//	FMA-bound    arithmetic throughput is the ceiling; instructions are cheap
//	issue-bound  the front end retires a fixed number of instructions per
//	             cycle and the FMA units are not the constraint
//
// Those two constraints rank the shapes in opposite orders, and each shape is
// the extreme of one axis (KERNEL.md §1, §3):
//
//	issue-bound  → fewest instructions per FMA        → 2×32 ×4 at 4.625
//	FMA-bound    → fewest memory operations per FMA   → 4×32 ×1 at 0.75
//
// so the rule below is a re-use of the gate's own classification rather than a
// new model, and P2's data is what says the physics is real: on janus the two
// shapes' throughputs stand in the inverse ratio of their instruction counts
// (1.308 measured against 1.351 predicted, KERNEL.md §7), which is what
// "issue-bound" means quantitatively. On an issue-bound machine the thin shape
// *is* the fast shape.
//
// # What ranks the shapes
//
// InsnsPerFMA is an audited compile-time measurement recorded on each Kernel
// (see the field's doc). MemOpsPerFMA is exact arithmetic on the tile shape.
// Neither is a runtime measurement: dispatch does not benchmark at init, which
// is P5's business, and a shape chosen by a startup micro-measurement would make
// every benchmark in the repo depend on the weather.
func Preferred(class Class, ks []Kernel) (Kernel, bool) {
	if len(ks) == 0 {
		return Kernel{}, false
	}
	best := ks[0]
	for _, k := range ks[1:] {
		if betterFor(class, k, best) {
			best = k
		}
	}
	return best, true
}

// betterFor reports whether a is the shape to prefer over b on a host of this
// class.
//
// A shape with no audited instruction count cannot be ranked on the axis that
// matters for an issue-bound host, and ranking it on the other axis alone would
// be picking a shape for a reason nobody measured. So an unaudited shape never
// displaces an incumbent, which leaves registry order in charge — the scalar
// fallback's two shapes reach dispatch that way, and they are a correctness path
// with no measurement behind either one.
//
// The comparisons are exact rather than epsilon-based on purpose. Both
// quantities are small binary rationals — 4.625 and 6.25 from integer
// instruction counts, 0.75 and 1.0 from 1/MR + Lanes/NR — so an epsilon here
// would only blur a tie the ranking already resolves in the next clause.
func betterFor(class Class, a, b Kernel) bool {
	if a.InsnsPerFMA <= 0 || b.InsnsPerFMA <= 0 {
		return false
	}
	am, bm := a.MemOpsPerFMA(), b.MemOpsPerFMA()
	if class == ClassIssue {
		if a.InsnsPerFMA != b.InsnsPerFMA {
			return a.InsnsPerFMA < b.InsnsPerFMA
		}
		return am < bm
	}
	if am != bm {
		return am < bm
	}
	return a.InsnsPerFMA < b.InsnsPerFMA
}

// Class is what binds a host's K-loop: the same two-way classification P2's
// throughput verdict makes from measurements (scripts/roofline.sh), reused here
// as the input to shape selection.
//
// The string values are the ones the gate prints and compares, so a marker and a
// verdict can be diffed without a translation table.
type Class string

// The two classes. There is no "unknown": a host that cannot be identified is
// treated as FMA-bound, which is both the common case and the strict direction —
// see HostClass.
const (
	// ClassFMA means arithmetic throughput binds, so the shape that reads the
	// least memory per FMA wins.
	ClassFMA Class = "fma"
	// ClassIssue means instruction issue binds, so the shape that issues the
	// fewest instructions per FMA wins.
	ClassIssue Class = "issue"
)

// Classes lists the classes a caller may force, for the tests and for the
// KEEL_KERN_CLASS override's error message.
func Classes() []Class { return []Class{ClassFMA, ClassIssue} }

// ParseClass turns a KEEL_KERN_CLASS value into a Class. An unrecognized value
// is rejected rather than defaulted: a run that believed it had pinned the
// issue-bound shape and silently measured the other one would be worse than a
// crash, which is the same rule KEEL_FORCE follows.
func ParseClass(s string) (Class, bool) {
	for _, c := range Classes() {
		if string(c) == s {
			return c, true
		}
	}
	return "", false
}
