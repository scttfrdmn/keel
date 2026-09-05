// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package spill

import (
	"bytes"
	"os"
	"os/exec"
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

// The symbol names a real listing carries, in the three forms #99 names. A
// generic's shape instantiation is the one that matters here: the struct types in
// its argument list contain spaces, so `^(\S+) STEXT` did not match it, and the
// header was skipped — which is not a lost function but a *moved* body, since
// every instruction after it was appended to `plainKernel`. `type:.eq.[2]interface {}`
// is the same shape and is live in internal/kern's listing today, which is how the
// defect was found still open after being reported dormant.
const genericListing = `main.plainKernel STEXT size=16 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (k.go:5)	TEXT	main.plainKernel(SB), ABIInternal, $0-16
	0x0004 00004 (k.go:6)	VFMADD213PS	Z1, Z0, Z2
	0x000a 00010 (k.go:7)	RET
main.Broadcast[float64,simd/archsimd.Float64x8] STEXT dupok size=16 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (k.go:12)	TEXT	main.Broadcast[float64,simd/archsimd.Float64x8](SB), ABIInternal, $0-16
	0x0004 00004 (k.go:13)	VFMADD213PS	Z1, Z0, Z2
	0x000a 00010 (k.go:14)	RET
main.Shaped[go.shape.float64,go.shape.struct { simd/archsimd.float64x8 simd/archsimd.v512; simd/archsimd.vals [8]float64 },go.shape.struct {}] STEXT dupok size=24 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (k.go:20)	TEXT	main.Shaped(SB), ABIInternal, $0-16
	0x0004 00004 (k.go:21)	VFMADD213PS	Z1, Z0, Z2
	0x000a 00010 (k.go:22)	VFMADD213PS	Z4, Z3, Z5
	0x0010 00016 (k.go:23)	RET
type:.eq.[2]interface {} STEXT dupok size=8 args=0x10 locals=0x0 funcid=0x0 align=0x0
	0x0000 00000 (<autogenerated>:1)	TEXT	type:.eq.[2]interface {}(SB), DUPOK|ABIInternal, $0-16
	0x0004 00004 (<autogenerated>:1)	RET
`

func TestParseNamesSymbolsHoldingSpaces(t *testing.T) {
	fns, err := Parse(strings.NewReader(genericListing))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(fns) != 4 {
		got := make([]string, len(fns))
		for i, f := range fns {
			got[i] = f.Name
		}
		t.Fatalf("Parse found %d function(s), want 4: %v", len(fns), got)
	}
	// Names and body sizes together, because the count is the half that states the
	// consequence: a header this parser cannot name is credited to the last one it
	// could, so the absorber's body grows by exactly the skipped function's — and
	// the audit then reports instructions that function does not contain. Against
	// `^(\S+) STEXT` this fixture parses two headers and `Broadcast` carries six
	// instructions rather than two.
	for i, want := range []struct {
		name  string
		insns int
	}{
		{"main.plainKernel", 2},
		{"main.Broadcast[float64,simd/archsimd.Float64x8]", 2},
		{"main.Shaped[go.shape.float64,go.shape.struct { simd/archsimd.float64x8 simd/archsimd.v512; simd/archsimd.vals [8]float64 },go.shape.struct {}]", 3},
		{"type:.eq.[2]interface {}", 1},
	} {
		if fns[i].Name != want.name {
			t.Errorf("function %d is named %q, want %q", i, fns[i].Name, want.name)
		}
		if n := len(fns[i].Insns); n != want.insns {
			t.Errorf("%s holds %d instruction(s), want %d", fns[i].Name, n, want.insns)
		}
	}
}

// TestFindResolvesAGenericByItsShortName covers the second half of #99: the gate
// and the CLI are given short names, and every instantiated symbol ends in `]`, so
// `-func Shaped` resolved nothing at all.
func TestFindResolvesAGenericByItsShortName(t *testing.T) {
	fns, err := Parse(strings.NewReader(genericListing))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, short := range []string{"Broadcast", "Shaped", "plainKernel"} {
		if _, err := Find(fns, short); err != nil {
			t.Errorf("Find(%q): %v", short, err)
		}
	}
	// The exact symbol still matches exactly, which is the escape hatch when one
	// generic has several instantiations and the short name is genuinely ambiguous.
	long := fns[1].Name
	if f, err := Find(fns, long); err != nil || f.Name != long {
		t.Errorf("Find(%q) = %q, %v; want the symbol itself", long, f.Name, err)
	}
}

// TestParseRejectsAnUnparsedHeader drives the fail-closed branch on purpose. A
// tab-separated header is hypothetical — no toolchain emits one — and that is the
// point: the parser cannot know which form the *next* release emits, so the
// requirement is that an unrecognised header stops the audit rather than silently
// moving a body (#99).
func TestParseRejectsAnUnparsedHeader(t *testing.T) {
	for _, c := range []struct{ what, header string }{
		{"tab-separated", "main.tabbed\tSTEXT size=8 args=0x0 locals=0x0"},
		{"empty name", " STEXT size=8 args=0x0 locals=0x0"},
	} {
		src := c.header + "\n\t0x0000 00000 (k.go:5)\tVFMADD213PS\tZ1, Z0, Z2\n"
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("Parse accepted a %s header: an unparsed header is indistinguishable from no header", c.what)
		} else if !strings.Contains(err.Error(), "unparsed symbol header") {
			t.Errorf("Parse rejected the %s header with %v, want the unparsed-header error", c.what, err)
		}
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

// -------------------------------------------------------------- arm64 (NEON, #13)

// A hand-written arm64 listing whose every line exercises a rule that DIFFERS from
// amd64 (#13): the K-loop is closed by BGE (a B-branch, not J); the anchor is a real
// HINT $0 that must be counted (not XCHGL); bare NOP carries <unknown line number>
// and is dropped; the arith is VFMLA; the broadcast is VDUP; and the spill is a
// vector F-register to the pseudo (SP). MOVD.W to (RSP) at the top is the hardware
// prologue and must NOT count as a spill.
const arm64Listing = `github.com/x.neonKernel STEXT size=200 args=0x18 locals=0x40 funcid=0x0 align=0x0
	0x0000 00000 (k.go:5)	MOVD.W	R30, -16(RSP)
	0x0080 00128 (k.go:10)	VFMLA	V21.S4, V17.S4, V16.S4
	0x0084 00132 (k.go:10)	VDUP	V25.S[0], V17.S4
	0x0088 00136 (k.go:10)	HINT	$0
	0x008c 00140 (k.go:10)	FMOVQ	F0, github.com/x.acc-16(SP)
	0x0090 00144 (<unknown line number>)	NOP
	0x0094 00148 (k.go:10)	ADD	$1, R0, R0
	0x0098 00152 (k.go:10)	CMP	R0, R1
	0x009c 00156 (k.go:10)	BGE	128
	0x00a0 00160 (k.go:11)	RET
`

func TestARM64AuditClassifiesLoopBody(t *testing.T) {
	if err := SetArch("arm64"); err != nil {
		t.Fatalf("SetArch: %v", err)
	}
	t.Cleanup(func() { SetArch("amd64") })

	fns, err := Parse(strings.NewReader(arm64Listing))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	f, err := Find(fns, "neonKernel")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	loop, err := f.SteadyLoop()
	if err != nil {
		t.Fatalf("SteadyLoop: %v", err)
	}
	// The BGE closes the loop at [128,156]; getting this wrong makes every count wrong.
	if loop.Start != 128 || loop.End != 156 {
		t.Errorf("steady loop = [%d,%d], want [128,156] (closed by BGE, a B-branch)", loop.Start, loop.End)
	}
	r := Audit(f, loop)
	for _, c := range []struct {
		what      string
		got, want int
	}{
		{"arith (VFMLA)", r.Arith, 1},
		{"vector stack refs (FMOVQ F0->(SP))", r.Spills(), 1},
		{"broadcasts (VDUP)", r.Broadcasts, 1},
		{"anchor nops (HINT $0)", r.Nops, 1},
		{"calls", len(r.Calls), 0},
		{"bounds-check exits", len(r.PanicExits), 0},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.what, c.got, c.want)
		}
	}
	// The spill is the FMOVQ to the pseudo (SP), not the MOVD.W to (RSP) prologue.
	if len(r.VecStack) == 1 && r.VecStack[0].Offset != 140 {
		t.Errorf("spill at offset %d, want 140", r.VecStack[0].Offset)
	}
}

// TestARM64InversionIsLoadBearing proves the arm64 rules are not cosmetic: applying
// the amd64 classification to an arm64 listing gets it WRONG, which is exactly the
// "silently triples every anchor count" / "audits nothing" failure #13 (and #46's
// class) warns about. Two distinct inversions, each pinned:
//
//   - the anchor: HINT $0 is an anchor on arm64 and nothing on amd64; XCHGL AX,AX is
//     the reverse. Counting NOP instead of HINT is the tripling.
//   - the branch: BGE closes the loop on arm64; under amd64's J-only rule it is not a
//     branch at all, so no loop is found and the audit sees nothing.
func TestARM64InversionIsLoadBearing(t *testing.T) {
	t.Cleanup(func() { SetArch("amd64") })

	SetArch("arm64")
	if !isNop("HINT", "$0") {
		t.Error("arm64: HINT $0 must be counted as an anchor")
	}
	if isNop("XCHGL", "AX, AX") {
		t.Error("arm64: XCHGL is not an arm64 anchor")
	}
	if !active.isBranch("BGE") || !active.isBranch("CBNZ") {
		t.Error("arm64: BGE/CBNZ must be recognized as branches")
	}
	if active.isBranch("BL") {
		t.Error("arm64: BL is a call, not an offset-branch")
	}

	SetArch("amd64")
	if isNop("HINT", "$0") {
		t.Error("amd64: HINT is not an amd64 anchor (counting it would triple the count on an arm64 listing)")
	}
	if !isNop("XCHGL", "AX, AX") {
		t.Error("amd64: XCHGL AX,AX is the amd64 anchor")
	}
	if active.isBranch("BGE") {
		t.Error("amd64: BGE is not a J-branch; treating it as one would misread arm64 listings")
	}
	// The end-to-end consequence: amd64 rules find NO loop in the arm64 listing (BGE
	// missed), so SteadyLoop fails — the audit would see nothing rather than the spill.
	fns, err := Parse(strings.NewReader(arm64Listing))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	f, _ := Find(fns, "neonKernel")
	if _, err := f.SteadyLoop(); err == nil {
		t.Error("amd64 rules found a loop in an arm64 listing: the branch inversion is not load-bearing?")
	}
}

// TestARM64SeesRealSpiller is the positive control the port's whole value rests on:
// the audit must see the accumulator spills in a REAL NEON kernel, or it audits
// nothing. Kernel8x12 is #136's known spiller — five accumulators the allocator moved
// to the stack (docs/neon-sweep.md). Reconciled: the audit reports vector-stack
// references (stores + reloads) over those five distinct slots. Skipped where the
// simd toolchain is unavailable (the stock-floor CI job), since the kernel needs it.
func TestARM64SeesRealSpiller(t *testing.T) {
	if err := SetArch("arm64"); err != nil {
		t.Fatalf("SetArch: %v", err)
	}
	t.Cleanup(func() { SetArch("amd64") })

	cmd := exec.Command("go", "build", "-gcflags=-S", "-o", os.DevNull,
		"github.com/scttfrdmn/keel/internal/vec")
	cmd.Env = append(os.Environ(), "GOEXPERIMENT=simd", "GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot cross-compile internal/vec for arm64 (needs the simd toolchain): %v", err)
	}
	fns, err := Parse(&errb)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	f, err := Find(fns, "Kernel8x12")
	if err != nil {
		t.Fatalf("Find Kernel8x12: %v", err)
	}
	loop, err := f.SteadyLoop()
	if err != nil {
		t.Fatalf("SteadyLoop: %v", err)
	}
	r := Audit(f, loop)
	if r.Spills() == 0 {
		t.Fatal("Kernel8x12 audited 0 spills, but #136 records it as a spiller — the audit sees nothing")
	}
	// Distinct accumulator slots, which must reconcile with #136's five.
	slots := map[string]bool{}
	for _, in := range r.VecStack {
		if i := strings.Index(in.Args, ".c"); i >= 0 {
			slot := in.Args[i:]
			if j := strings.Index(slot, "-"); j >= 0 {
				slot = slot[:j]
			}
			slots[slot] = true
		}
	}
	if len(slots) != 5 {
		t.Errorf("Kernel8x12 spilled %d distinct accumulators, want 5 (docs/neon-sweep.md); slots=%v", len(slots), slots)
	}
}
