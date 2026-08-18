// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/spill"
)

// TestShippedShapesAreFixedPoints is the guard that keeps the generator honest
// about provenance.
//
// KERNEL.md cites this generator as the reason each shipped shape is the shape it
// is. That citation is only true while the generator still emits those shapes, so
// the claim is checked rather than asserted: if a shipped kernel is edited by hand
// and the generator is not, this fails and the two have to be reconciled in the
// same commit.
//
// It needs no simd toolchain — it compares source text — so it runs everywhere,
// including on the stock-toolchain CI job. The compile-and-audit half of the same
// verification lives in TestShippedShapesAuditIdentically, which does need the
// experiment.
func TestShippedShapesAreFixedPoints(t *testing.T) {
	src := readShipped(t)
	for _, s := range shipped {
		t.Run(s.Label(), func(t *testing.T) {
			want := normalize(funcBody(src, s.Name()))
			if want == "" {
				t.Fatalf("%s is not in internal/vec/gemm_amd64.go; the shipped list is stale", s.Name())
			}
			if got := normalize(funcBody(s.Emit(), s.Name())); got != want {
				t.Errorf("emitted body differs from the shipped kernel.\n--- tree ---\n%s\n--- emitted ---\n%s", want, got)
			}
		})
	}
}

// TestFixedPointComparisonCanFail is the control for the test above.
//
// A comparison that always passes proves nothing about what it compares, and a
// fixed-point check is exactly the shape that can pass vacuously — funcBody
// returning "" for both sides, or normalize flattening away the difference, would
// each read as agreement. So the same comparison is driven with a shape that must
// not match: 2x32 at unroll 2 emits a function with the shipped kernel's name and
// a different body.
func TestFixedPointComparisonCanFail(t *testing.T) {
	src := readShipped(t)
	perturbed := Shape{MR: 2, V: 2, U: 2, Form: Broadcast}
	if perturbed.Name() != "Kernel2x32" {
		t.Fatalf("the control needs the shipped kernel's name, got %s", perturbed.Name())
	}
	want := normalize(funcBody(src, "Kernel2x32"))
	got := normalize(funcBody(perturbed.Emit(), "Kernel2x32"))
	if want == "" || got == "" {
		t.Fatalf("one side is empty, so the comparison would pass vacuously: tree %d bytes, emitted %d bytes", len(want), len(got))
	}
	if got == want {
		t.Error("unroll 2 emitted the same body as the shipped unroll 4, so the fixed-point check cannot detect a change")
	}
}

// TestNormalizeStripsOnlyProse pins what the text comparison is allowed to ignore.
func TestNormalizeStripsOnlyProse(t *testing.T) {
	in := "func f() {\n\t// a comment\n\n\tx := 1 \n}\n"
	want := "func f() {\n\tx := 1\n}"
	if got := normalize(in); got != want {
		t.Errorf("normalize(%q) = %q, want %q", in, got, want)
	}
	if strings.Contains(normalize(in), "comment") {
		t.Error("a comment survived normalization, so a prose-only edit would read as a code change")
	}
}

// TestBoundFlipsAtPortsTimesLatency drives the three roofline terms on purpose.
//
// The dependency term is the one the vanished objective lacked, so "it is present"
// is the claim under test and an unexercised branch would not support it. The
// algebra being checked is that (F/A)*L > F/P reduces to A < P*L, so the flip
// depends on the accumulator count alone and not on the unroll.
func TestBoundFlipsAtPortsTimesLatency(t *testing.T) {
	u := UArch{Name: "test", Width: 4, Ports: 2, Lat: 4}
	threshold := u.Ports * u.Lat // 8 chains

	// 4 chains (2x32): below the threshold, so the chain term must bind. Insns are
	// set low enough that the issue term cannot be the winner.
	below := Shape{MR: 2, V: 2, U: 4, Form: Broadcast}
	if got := u.Score(below, 8).Bound; got != ChainBound {
		t.Errorf("%s has %d chains, under %d, so it must be dependency-bound; got %s",
			below.Label(), below.Accs(), threshold, got)
	}
	// 8 chains (4x32): at the threshold, so the chain and port terms tie and the
	// chain term must no longer win outright.
	at := Shape{MR: 4, V: 2, U: 1, Form: Broadcast}
	if got := u.Score(at, 8).Bound; got == ChainBound {
		t.Errorf("%s has %d chains, at the threshold %d, so the chain term must not bind alone; got %s",
			at.Label(), at.Accs(), threshold, got)
	}
	// A large instruction count must reach the issue term, or the first branch is
	// the only one this test ever proves.
	if got := u.Score(at, 400).Bound; got != IssueBound {
		t.Errorf("400 insns over width 4 is 100 cycles and must dominate; got %s", got)
	}
	// The unroll must not move the classification, since A does not depend on U.
	for _, unroll := range []int{1, 2, 4, 8} {
		s := Shape{MR: 2, V: 2, U: unroll, Form: Broadcast}
		if got := u.Score(s, s.FMAs()).Bound; got != ChainBound {
			t.Errorf("unroll %d changed the bound to %s; A=%d is unroll-independent", unroll, got, s.Accs())
		}
	}
}

// TestPermuteWindowIsExactlySixteen pins the constraint that makes the two forms
// cover different shape sets, and drives both inequalities.
//
// Testing only the lower bound is exactly the defect this replaced: the predicate
// admitted MR*U > 16 and the emitter then referenced an index vector it had never
// hoisted, which the compiler caught as an undefined idx16 partway through a sweep.
// So both directions are asserted, not just the interesting one.
func TestPermuteWindowIsExactlySixteen(t *testing.T) {
	// 2x64 u=2 is docs/spill-report.md:206's Permute optimum and the mint of
	// gate-p2's SWEEP_BEST_IPF. MR*U is 4, so one 16-lane A load is out of bounds.
	lost := Shape{MR: 2, V: 4, U: 2, Form: Permute}
	if lost.PermuteWindowExact() {
		t.Errorf("%s guarantees only %d A floats; a 16-lane load must not be considered in bounds",
			lost.Label(), lost.MR*lost.U)
	}
	// The upper bound: 24 floats is in bounds but needs lanes the window lacks.
	if over := (Shape{MR: 3, V: 1, U: 8, Form: Permute}); over.PermuteWindowExact() {
		t.Errorf("%s needs %d A values from a 16-lane window; it must be excluded", over.Label(), over.MR*over.U)
	}
	if in := (Shape{MR: 4, V: 2, U: 4, Form: Permute}); !in.PermuteWindowExact() {
		t.Errorf("%s needs exactly %d A values, which one window holds", in.Label(), in.MR*in.U)
	}
	// Every enumerated Permute shape must be emittable, and there must be some.
	n := 0
	for _, s := range space() {
		if s.Form != Permute {
			continue
		}
		n++
		if !s.PermuteWindowExact() {
			t.Errorf("the sweep enumerated %s, whose A-window load is not exactly covered", s.Label())
		}
	}
	if n == 0 {
		t.Error("the sweep enumerated no Permute shapes at all, so the loop above proved nothing")
	}
}

// TestParseUArchRejectsRatherThanDefaults exists because the failure mode of a
// -uarch flag is silent: a parser that returned the zero value or the Skylake-X
// default on a malformed spec would score an SPR re-sweep as Skylake-X and print the
// wrong name beside it, which is a whole run's worth of wrong verdicts and no error.
// So every rejection path is driven, and the default is checked to still be the
// microarchitecture every published reading was scored against.
func TestParseUArchRejectsRatherThanDefaults(t *testing.T) {
	for _, bad := range []string{"", "skylake-x", "skylake-x:4:2", "skylake-x:4:2:4:1", ":4:2:4", "spr:6:x:4", "spr:0:2:4", "spr:6:2:-4"} {
		if u, err := parseUArch(bad); err == nil {
			t.Errorf("parseUArch(%q) accepted a malformed spec as %+v", bad, u)
		}
	}
	got, err := parseUArch("sapphire-rapids:6:2:4")
	if err != nil {
		t.Fatalf("parseUArch of a well-formed spec: %v", err)
	}
	if want := (UArch{Name: "sapphire-rapids", Width: 6, Ports: 2, Lat: 4}); got != want {
		t.Errorf("parseUArch gave %+v, want %+v", got, want)
	}
	if want := (UArch{Name: "skylake-x", Width: 4, Ports: 2, Lat: 4}); shippedUArch != want {
		t.Errorf("the default uarch is %+v; every published insns/FMA reading was scored against %+v", shippedUArch, want)
	}
}

// TestFrontierRefusesRatherThanReportingZero drives the fail-closed claim in best's
// comment, because the failure it guards is silent and maximally permissive: gate-p2
// divides by SWEEP_BEST_IPF, so a frontier of 0.000 out of an empty contender set
// would widen the shape guard to admit any kernel at all. The state is reachable
// rather than hypothetical — the Permute form has no zero-spill shape today, which is
// why summarize has a branch for it.
func TestFrontierRefusesRatherThanReportingZero(t *testing.T) {
	if got, ok := best(nil, fewestInsnsPerFMA); ok {
		t.Errorf("best over no rows reported %v as a frontier", got.Score.InsnsPerFMA())
	}
	s := Shape{MR: 2, V: 2, U: 4, Form: Broadcast}
	spilling := row{Shape: s, Report: spill.Report{Insns: 74, Arith: 16, VecStack: []spill.Insn{{}}}}
	if contender(spilling) {
		t.Errorf("a row with %d spills was admitted as a frontier candidate", spilling.Report.Spills())
	}
	if contender(row{Shape: s, Err: errors.New("did not compile")}) {
		t.Error("a shape that failed to audit was admitted as a frontier candidate")
	}
	clean := row{Shape: s, Report: spill.Report{Insns: 74, Arith: 16}, Score: shippedUArch.Score(s, 74)}
	if !contender(clean) {
		t.Fatal("a zero-spill audited row was rejected, so the two checks above prove nothing")
	}
	if got, ok := best([]row{clean}, fewestInsnsPerFMA); !ok || got.Score.InsnsPerFMA() != 4.625 {
		t.Errorf("best over one clean row gave %.3f (ok=%v), want 4.625", got.Score.InsnsPerFMA(), ok)
	}
}

func readShipped(t *testing.T) string {
	t.Helper()
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("locating the repo root: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, "internal", "vec", "gemm_amd64.go"))
	if err != nil {
		t.Fatalf("reading the shipped kernels: %v", err)
	}
	return string(b)
}
