// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd

package main

import (
	"bytes"
	"testing"

	"github.com/scttfrdmn/keel/internal/spill"
)

// TestShippedShapesAuditIdentically is the fixed point at the level that binds.
//
// The text comparison in emit_test.go can be defeated by a difference that does not
// matter (a renamed variable) or fooled by one that does (a reordering the compiler
// undoes). This one compares what the objective actually reads: the audit report of
// the emitted candidate against the audit report of the shipped kernel, field for
// field, both compiled by the same toolchain in the same invocation.
//
// It asserts EQUIVALENCE, not absolute counts, and the distinction is deliberate. A
// toolchain bump may legitimately move 2x32 from 74 instructions to some other
// number — that is a published-number event, and its home is docs/spill-report.md
// and a CHANGELOG line, not a red CI run on the day go1.27 lands. What must never
// change silently is that the generator emits the kernel the project ships, and
// that is toolchain-independent because both sides are built by whichever toolchain
// is running. The absolute readings are logged so a bump is visible in the run.
//
// Tagged goexperiment.simd because the emitted code imports simd/archsimd. The
// target is always linux/amd64 regardless of host, per the audit's own rule.
func TestShippedShapesAuditIdentically(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("locating the repo root: %v", err)
	}
	listing, err := compile("./internal/vec")
	if err != nil {
		t.Fatalf("compiling the shipped kernels: %v", err)
	}
	fns, err := spill.Parse(bytes.NewReader(listing))
	if err != nil {
		t.Fatalf("parsing the listing: %v", err)
	}

	for _, s := range shipped {
		t.Run(s.Label(), func(t *testing.T) {
			f, err := spill.Find(fns, s.Name())
			if err != nil {
				t.Fatalf("%v", err)
			}
			loop, err := f.SteadyLoop()
			if err != nil {
				t.Fatalf("%v", err)
			}
			want := spill.Audit(f, loop)
			got, err := audit(root, s, "")
			if err != nil {
				t.Fatalf("emitting and auditing the candidate: %v", err)
			}
			if d := reportDiff(want, got); d != "" {
				t.Errorf("the emitted candidate is not the shipped kernel: %s", d)
			}
			t.Logf("%s: %d insns / %d FMAs = %.3f insns/FMA, %d spills, %d copies, %d broadcasts, %d nops",
				s.Label(), got.Insns, got.Arith, float64(got.Insns)/float64(got.Arith),
				got.Spills(), got.VecCopies, got.Broadcasts, got.Nops)
		})
	}
}

// TestReportDiffNamesEveryDisagreement is the control for the comparison above: a
// field-by-field diff that returned "" unconditionally would report agreement for
// any pair of reports.
func TestReportDiffNamesEveryDisagreement(t *testing.T) {
	base := spill.Report{Insns: 74, Arith: 16, VecCopies: 8, Broadcasts: 8, Nops: 8}
	if d := reportDiff(base, base); d != "" {
		t.Errorf("a report must not differ from itself: %s", d)
	}
	moved := base
	moved.Insns = 72
	if d := reportDiff(base, moved); d == "" {
		t.Error("a 74-to-72 instruction change was reported as agreement")
	}
	// Spills is the field the P2 criterion is stated in terms of, and it is
	// computed from a slice rather than stored, so it is worth driving on purpose.
	spilled := base
	spilled.VecStack = []spill.Insn{{Op: "VMOVUPS", Args: "Z1, 8(SP)"}}
	if d := reportDiff(base, spilled); d == "" {
		t.Error("a spill appearing in the loop body was reported as agreement")
	}
}
