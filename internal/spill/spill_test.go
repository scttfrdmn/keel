// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package spill

import (
	"strings"
	"testing"
)

// The audit's job is to answer three questions about a loop body — does it spill,
// does it call, does a bounds check survive — and each answer is a judgement about
// operand text that a wrong regexp would get silently wrong. So the test drives a
// hand-written listing whose every line is there for a reason, rather than
// compiling something and hoping the compiler produces the instruction under test.
//
// The listing below is realistic in the ways that matter: the panic block sits
// after the RET (out of line, which is why the audit has to follow the branch
// rather than grep the body), the anchor NOP is spelled XCHGL AX, AX rather than
// NOP (T9, and the reason isNop exists), and the broadcast reads memory (so it
// must not be mistaken for either a spill or a register copy).
const listing = `main.spillsAndPanics STEXT size=96 args=0x18 locals=0x40 funcid=0x0 align=0x0
	0x0000 00000 (k.go:5)	TEXT	main.spillsAndPanics(SB), ABIInternal, $64-24
	0x0004 00004 (k.go:6)	XORL	AX, AX
	0x0006 00006 (k.go:7)	JMP	41
	0x0008 00008 (k.go:8)	VMOVDQU64	(CX)(AX*4), Z0
	0x000e 00014 (k.go:8)	XCHGL	AX, AX
	0x000f 00015 (k.go:9)	VBROADCASTSS	(DX), Z1
	0x0015 00021 (k.go:9)	VFMADD213PS	Z1, Z0, Z2
	0x001b 00027 (k.go:9)	VMOVDQU64	Z2, Z3
	0x0021 00033 (k.go:10)	VMOVDQU64	Z3, main.acc(SP)
	0x0027 00039 (k.go:11)	JLE	60
	0x0029 00041 (k.go:7)	INCQ	AX
	0x002c 00044 (k.go:7)	CMPQ	AX, BX
	0x002f 00047 (k.go:7)	JLT	8
	0x0031 00049 (k.go:12)	RET
	0x0032 00060 (k.go:11)	MOVQ	AX, CX
	0x0035 00063 (k.go:11)	CALL	runtime.panicIndex(SB)
main.panicsBehindPadding STEXT size=48 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (k.go:25)	TEXT	main.panicsBehindPadding(SB), ABIInternal, $0-16
	0x0004 00004 (k.go:26)	XORL	AX, AX
	0x0006 00006 (k.go:27)	VMOVDQU64	(CX)(AX*4), Z0
	0x000c 00012 (k.go:27)	VFMADD213PS	Z1, Z0, Z2
	0x0012 00018 (k.go:27)	JCS	40
	0x0014 00020 (k.go:27)	INCQ	AX
	0x0017 00023 (k.go:27)	CMPQ	AX, BX
	0x001a 00026 (k.go:27)	JLT	6
	0x001c 00028 (k.go:28)	RET
	0x0028 00040 (k.go:27)	NOP
	0x0030 00048 (k.go:27)	CALL	runtime.panicBounds(SB)
main.clean STEXT size=32 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (k.go:16)	TEXT	main.clean(SB), ABIInternal, $0-16
	0x0004 00004 (k.go:17)	XORL	AX, AX
	0x0006 00006 (k.go:18)	VMOVDQU64	(CX)(AX*4), Z0
	0x000c 00012 (k.go:18)	VFMADD213PS	Z1, Z0, Z2
	0x0012 00018 (k.go:18)	JLE	28
	0x0014 00020 (k.go:18)	INCQ	AX
	0x0017 00023 (k.go:18)	CMPQ	AX, BX
	0x001a 00026 (k.go:18)	JLT	6
	0x001c 00028 (k.go:19)	RET
`

func report(t *testing.T, name string) Report {
	t.Helper()
	fns, err := Parse(strings.NewReader(listing))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	f, err := Find(fns, name)
	if err != nil {
		t.Fatalf("Find(%q): %v", name, err)
	}
	loop, err := f.SteadyLoop()
	if err != nil {
		t.Fatalf("SteadyLoop: %v", err)
	}
	return Audit(f, loop)
}

func TestAuditClassifiesLoopBody(t *testing.T) {
	r := report(t, "spillsAndPanics")

	// The loop is the backward branch's range: offsets 8 through 47, ten
	// instructions. Getting this wrong would make every count below wrong in the
	// same direction, so it is asserted rather than assumed.
	if r.Loop.Start != 8 || r.Loop.End != 47 {
		t.Errorf("steady loop = [%d,%d], want [8,47]", r.Loop.Start, r.Loop.End)
	}
	for _, c := range []struct {
		what string
		got  int
		want int
	}{
		{"insns", r.Insns, 10},
		{"arith", r.Arith, 1},
		{"vector stack refs", r.Spills(), 1},
		{"reg copies", r.VecCopies, 1},
		{"broadcasts", r.Broadcasts, 1},
		{"anchor nops", r.Nops, 1},
		{"calls", len(r.Calls), 0},
		{"bounds-check exits", len(r.PanicExits), 1},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.what, c.got, c.want)
		}
	}
	// The distinctions this tool exists to draw, stated as such: a memory
	// broadcast is neither a spill nor a copy, and the register-to-register move
	// is a copy and not a spill. Conflating them is the mistake that would send
	// a P2 decision the wrong way (see the package doc).
	if got := r.VecStack[0].Offset; got != 33 {
		t.Errorf("the spill is at offset %d, want 33 (the (SP) store, not the broadcast)", got)
	}
	if got := r.PanicExits[0].Offset; got != 39 {
		t.Errorf("the bounds-check exit is at offset %d, want 39", got)
	}
	// The call is out of line, past the RET. A body-only search would miss it,
	// which is the whole reason Audit takes the function.
	if n := len(r.Calls); n != 0 {
		t.Errorf("%d call(s) inside the body; the panic call is out of line", n)
	}
}

// TestAuditCleanLoop is the negative control. Without it, a detector that never
// fires would pass every real audit and report zero of everything.
func TestAuditCleanLoop(t *testing.T) {
	r := report(t, "clean")
	if r.Spills() != 0 || r.VecCopies != 0 || r.Nops != 0 || r.Broadcasts != 0 {
		t.Errorf("clean loop misclassified: %s", r.Summary())
	}
	// The loop's own exit branch (JLE 28, leaving the body) must not be counted:
	// every loop has one, and it does not lead to a panic.
	if n := len(r.PanicExits); n != 0 {
		t.Errorf("%d bounds-check exit(s) in a loop that has none: %s", n, r.Summary())
	}
	if got, want := r.Memory(), 1; len(got) != want {
		t.Errorf("%d memory reference(s), want %d (the panel load)", len(got), want)
	}
}

// TestAuditSeesAPanicBehindAlignmentPadding is the regression test for issue #46:
// the audit reported 0 bounds-check exits for loops that had one, because the
// branch targeted an *aligned* out-of-line panic block and the alignment NOP —
// which Parse drops, and which owns its own offset rather than sharing the next
// instruction's — was the exact offset the resolver looked for.
//
// This is the failure mode a detector must never have, because gate-p2 converts
// this count into a passing criterion ("0 surviving bounds checks in the
// steady-state K-loop"). A false clean does not merely lose information; it
// manufactures a green. So the listing above spells the case out — JCS 40, a NOP
// at 40, the CALL at 48 — rather than trusting that some real function happens to
// exercise it.
func TestAuditSeesAPanicBehindAlignmentPadding(t *testing.T) {
	r := report(t, "panicsBehindPadding")
	if n := len(r.PanicExits); n != 1 {
		t.Fatalf("%d bounds-check exit(s), want 1 — the branch at 18 targets an "+
			"alignment NOP at 40 and the panic call is at 48, so an exact-offset "+
			"resolver misses it entirely: %s", n, r.Summary())
	}
	if got := r.PanicExits[0].Offset; got != 18 {
		t.Errorf("the exit is reported at offset %d, want 18 (the branch, not its target)", got)
	}
}

// TestSteadyLoopIgnoresMorestack guards the correction recorded in Loops(): the
// morestack re-entry jump spans the whole function, and treating it as a loop made
// the first version of this tool report every function's entire body as its own
// steady state.
func TestSteadyLoopIgnoresMorestack(t *testing.T) {
	const withMorestack = `main.leaf STEXT size=32 args=0x8 locals=0x8 funcid=0x0 align=0x0
	0x0000 00000 (k.go:5)	TEXT	main.leaf(SB), ABIInternal, $8-8
	0x0008 00008 (k.go:6)	VFMADD213PS	Z1, Z0, Z2
	0x000e 00014 (k.go:6)	JLT	8
	0x0010 00016 (k.go:7)	RET
	0x0011 00017 (k.go:5)	CALL	runtime.morestack_noctxt(SB)
	0x0016 00022 (k.go:5)	JMP	0
`
	fns, err := Parse(strings.NewReader(withMorestack))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	loop, err := fns[0].SteadyLoop()
	if err != nil {
		t.Fatalf("SteadyLoop: %v", err)
	}
	if loop.Start != 8 || loop.End != 14 {
		t.Errorf("steady loop = [%d,%d], want [8,14]: the morestack jump is not a loop",
			loop.Start, loop.End)
	}
}
