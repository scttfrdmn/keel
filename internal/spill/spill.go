// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package spill is the P2 spill audit: it reads the compiler's own assembly
// listing and reports what the steady-state loop of a named function actually
// contains (DESIGN.md §4/P2).
//
// # Why a tool rather than a grep
//
// The P2 gate criterion is "0 accumulator spills in the steady-state K-loop".
// Every word of that needs a decision that a grep cannot make:
//
//   - *Steady-state*: a Go function's listing contains the loop, its prologue,
//     its remainder handling, and the panic paths. A spill in the prologue costs
//     nothing amortized over K; a spill in the loop body costs a round trip to
//     L1 per iteration. Only the loop body is the criterion, so the loop body
//     has to be identified, which means reading branch targets.
//   - *Spill*: a stack reference that moves a vector register. A stack
//     reference that spills the loop counter is worth reporting and is not what
//     the gate is about. A register-to-register vector move is not a spill at
//     all — it costs an issue slot, not memory traffic — and conflating the two
//     would have made the peak kernel's 26 copies per iteration (T8) look like
//     26 spills, which would have sent P2 down the wrong road entirely.
//   - *Accumulator*: distinguishing an accumulator's stack slot from any other
//     vector's requires knowing which value is which, which the listing does not
//     say. This tool therefore reports *all* vector stack traffic in the loop
//     and treats any of it as a failure. That is stricter than DESIGN.md's
//     wording, and deliberately so: in a kernel whose loop should touch nothing
//     but its panels and its registers, there is no benign vector stack
//     reference to excuse.
//
// The same argument applies to P2's other two compile-time criteria, which is why
// they are checked here too rather than by grepping the compiler's diagnostics.
// "No calls in the K-loop" and "pre-sliced panels" are statements about the loop
// body; `-d=ssa/check_bce` reports bounds checks anywhere in the file, including
// the ones in a kernel's prologue and write-out that cost nothing per k. See
// Func.panicExits.
//
// # What it reads
//
// `go build -gcflags=-S` emits, on stderr, one line per instruction:
//
//	0x0116 00278 (kern.go:42)	VFMADD213PS	Z1, Z0, Z2
//
// The second column is the byte offset of the instruction within the function,
// and a jump's operand is that same offset — so a backward jump identifies a
// loop with no need for a control-flow graph. Pseudo-instructions (PCDATA,
// FUNCDATA) and NOPs share offsets with real ones and are dropped.
//
// This reads the *final* object code, after every optimization pass. That is the
// point: ssa.html explains why the compiler did something, but only the listing
// says what it emitted, and P2's criterion is about what executes. The gate
// archives both.
//
// # Portability
//
// The parser is toolchain-format-specific (Go's own listing) and the operand
// classification is amd64-specific. It has to be: the whole question is what
// instructions came out. A future arm64 port needs its own register-name and
// stack-reference rules, and issue #13 records that. The tool itself builds and
// runs on any host with a stock toolchain — it inspects a cross-compile rather
// than needing to run the code.
package spill

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Insn is one instruction from the assembly listing.
type Insn struct {
	Offset int    // byte offset within the function; a jump operand refers to this
	Loc    string // source position the compiler attributed it to
	Op     string // mnemonic, e.g. VFMADD213PS
	Args   string // operand text, verbatim and unparsed
}

// String renders an instruction the way the listing did, for gate output and
// for pasting into a report.
func (i Insn) String() string {
	if i.Args == "" {
		return fmt.Sprintf("%05d (%s)\t%s", i.Offset, i.Loc, i.Op)
	}
	return fmt.Sprintf("%05d (%s)\t%s\t%s", i.Offset, i.Loc, i.Op, i.Args)
}

// Func is one function's listing.
type Func struct {
	Name  string
	Insns []Insn
}

// Loop is a contiguous instruction range closed by a backward branch.
type Loop struct {
	Start, End int // byte offsets: the branch target and the branch itself
	Insns      []Insn
}

var (
	// 0x0116 00278 (file.go:42)	VFMADD213PS	Z1, Z0, Z2
	insnRE = regexp.MustCompile(`^\t0x[0-9a-f]+ (\d+) \((.*)\)\t(\S+)\t?(.*)$`)
	// github.com/x/y.Fn STEXT size=538 args=0x8 locals=0x28 ...
	//
	// `.*`, not `\S+`: a symbol name contains spaces whenever a type in it does,
	// which is true of every generic instantiated over a struct shape and true
	// today of `type:.eq.[2]interface {}` in internal/kern's listing. `\S+` did not
	// match those, and Parse's skip-what-you-don't-know rule then credited the
	// unmatched function's whole body to the last one it did recognize — so the
	// failure mode was misattribution, not omission (#99). stextRE is the
	// fail-closed half: a line naming STEXT that funcRE cannot parse is an error,
	// because an unparsed header is otherwise indistinguishable from no header.
	funcRE  = regexp.MustCompile(`^(.*) STEXT(?: |$)`)
	// Anchored off column 0 because only a header starts there: an instruction or a
	// data dump begins with a tab, and a `go:string` dump can carry any bytes at all
	// in its ASCII column, including these seven.
	stextRE = regexp.MustCompile(`^[^\t]*[\t ]STEXT(?:[\t ]|$)`)
	// A branch whose operand is a bare instruction offset.
	targetRE = regexp.MustCompile(`^(\d+)$`)
	// Memory operands: (SP), 8(SP), sym+8(SP), (AX), (AX)(CX*4), $f32.0(SB).
	memRE = regexp.MustCompile(`\((SP|SB|[A-Z][A-Z0-9]*)\)`)
	// Vector registers, in Go's amd64 assembler names.
	vecRegRE = regexp.MustCompile(`\b[XYZ][0-9]{1,2}\b`)
	// Stack-relative references specifically.
	stackRE = regexp.MustCompile(`\(SP\)`)
)

// pseudo instructions carry no code and share an offset with the real
// instruction that follows them.
var pseudo = map[string]bool{
	"PCDATA":   true,
	"FUNCDATA": true,
	"NOP":      true,
	"TEXT":     true,
}

// Parse reads a `go build -gcflags=-S` listing and returns the functions in it.
//
// Lines it does not recognize are ignored rather than rejected: a listing also
// carries symbol dumps, rel entries and compiler notes, and a parser that
// insisted on understanding all of it would break on the next toolchain release
// for no gain. What matters is that an *instruction* line is never
// misinterpreted, which is why insnRE is anchored on the offset column.
//
// A *header* is the exception, and is rejected (#99). Skipping one is not the
// benign miss the rule above describes: every instruction that follows it is
// appended to the previous function, so an unparsed header does not lose a body,
// it moves it — and the audit then reports another function's spills as this
// one's. There is no reading of that as "no header", so it is an error.
func Parse(r io.Reader) ([]Func, error) {
	var (
		out  []Func
		cur  *Func
		scan = bufio.NewScanner(r)
	)
	scan.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scan.Scan() {
		line := scan.Text()
		if m := funcRE.FindStringSubmatch(line); m != nil && m[1] != "" {
			out = append(out, Func{Name: m[1]})
			cur = &out[len(out)-1]
			continue
		}
		if stextRE.MatchString(line) {
			return nil, fmt.Errorf("unparsed symbol header %q: its body would be credited to the function before it", line)
		}
		m := insnRE.FindStringSubmatch(line)
		if m == nil || cur == nil {
			continue
		}
		op := m[3]
		if pseudo[op] {
			continue
		}
		off, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		cur.Insns = append(cur.Insns, Insn{Offset: off, Loc: m[2], Op: op, Args: m[4]})
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Find returns the function whose name ends in "."+short, e.g. "avx512Peak"
// matching "github.com/scttfrdmn/keel/internal/vec.avx512Peak".
//
// Matching on the short name keeps callers (and the gate) from having to spell
// out a fully qualified symbol, at the cost of ambiguity if two packages in one
// build have same-named functions — so ambiguity is an error rather than a
// silent first-match.
//
// A generic's symbol ends in its type-argument list, so `G` is matched against
// the name with that list cut off (#99); an instantiated kernel was otherwise
// unfindable by any name a caller would type. Two instantiations of one generic
// then collide, and that is reported as the ambiguity it is — the exact symbol
// still matches exactly.
func Find(fns []Func, short string) (Func, error) {
	var hits []Func
	for _, f := range fns {
		b := baseName(f.Name)
		if f.Name == short || b == short || strings.HasSuffix(b, "."+short) {
			hits = append(hits, f)
		}
	}
	switch len(hits) {
	case 0:
		return Func{}, fmt.Errorf("no function named %q in the listing", short)
	case 1:
		return hits[0], nil
	default:
		names := make([]string, len(hits))
		for i, h := range hits {
			names[i] = h.Name
		}
		return Func{}, fmt.Errorf("%q is ambiguous: %s", short, strings.Join(names, ", "))
	}
}

// baseName drops a generic instantiation's type-argument list. The first `[` is
// enough: a Go identifier holds none, and a package path holds none either, so
// everything from it onwards is type arguments — however deeply the shape types
// inside nest their own brackets.
func baseName(name string) string {
	if i := strings.IndexByte(name, '['); i >= 0 {
		return name[:i]
	}
	return name
}

// isArith reports whether an instruction does the floating-point work this
// audit is about. Used to pick the steady-state loop: the loop that carries the
// arithmetic is the one whose spills matter.
func isArith(op string) bool {
	switch {
	case strings.HasPrefix(op, "VFMADD"), strings.HasPrefix(op, "VFMSUB"),
		strings.HasPrefix(op, "VFNMADD"), strings.HasPrefix(op, "VFNMSUB"):
		return true
	}
	for _, p := range []string{"MUL", "ADD", "SUB", "DIV"} {
		if !strings.HasPrefix(strings.TrimPrefix(op, "V"), p) {
			continue
		}
		for _, s := range []string{"PS", "PD", "SS", "SD"} {
			if strings.HasSuffix(op, s) {
				return true
			}
		}
	}
	return false
}

// isVecMove reports whether an instruction is a vector data movement — the
// class a spill or a register copy belongs to.
func isVecMove(op string) bool {
	switch op {
	case "MOVUPS", "MOVAPS", "MOVSS", "MOVSD", "MOVLPS", "MOVHPS":
		return true
	}
	if !strings.HasPrefix(op, "V") {
		return false
	}
	for _, p := range []string{"VMOVUPS", "VMOVAPS", "VMOVDQU", "VMOVDQA", "VMOVSS",
		"VMOVSD", "VBROADCAST", "VPBROADCAST", "VEXTRACT", "VINSERT"} {
		if strings.HasPrefix(op, p) {
			return true
		}
	}
	return false
}

// isBroadcast reports whether an instruction splats a scalar across a vector.
//
// Counted apart from register copies because it is not one: a GEMM microkernel's
// broadcasts are arithmetic setup it cannot avoid — one per row per k-step — while
// a copy is the register allocator working around VFMADD213PS's destination
// (docs/toolchain-notes.md T2). Lumping them together, which the first version of
// this tool did, made the 2x32 tile look like it had 16 allocator copies per pass
// when it has 8.
func isBroadcast(op string) bool {
	return strings.HasPrefix(op, "VBROADCAST") || strings.HasPrefix(op, "VPBROADCAST")
}

// isNop reports whether an instruction is a statement-position anchor.
//
// The compiler emits these as 1-byte `XCHGL AX, AX` rather than as the NOP
// pseudo-op, so the listing parser cannot drop them and the audit has to name
// them: one appears per inlined call site whose instructions were re-attributed
// to a callee's source position (docs/toolchain-notes.md T9). They cost an issue
// slot each and were 27% of one draft of the microkernel body, which is why they
// are counted rather than ignored.
func isNop(op, args string) bool {
	switch op {
	case "NOP":
		return true
	case "XCHGL", "XCHGQ":
		return strings.ReplaceAll(args, " ", "") == "AX,AX"
	}
	return false
}

// Loops returns every loop closed by a backward branch, in listing order and
// deduplicated by extent.
//
// One backward branch is not a loop and has to be excluded: every non-leaf Go
// function ends with `CALL runtime.morestack_noctxt` followed by a jump back to
// the function's first instruction, which is a *re-entry* after the stack grows,
// not an iteration. It spans the whole function, so leaving it in would make
// every audit report the entire body as its own steady-state loop — which is
// exactly what the first version of this tool did.
func (f Func) Loops() []Loop {
	seen := map[[2]int]bool{}
	var out []Loop
	for i, in := range f.Insns {
		if !strings.HasPrefix(in.Op, "J") {
			continue
		}
		m := targetRE.FindStringSubmatch(strings.TrimSpace(in.Args))
		if m == nil {
			continue
		}
		target, err := strconv.Atoi(m[1])
		if err != nil || target > in.Offset {
			continue
		}
		if i > 0 && isMorestack(f.Insns[i-1]) {
			continue
		}
		key := [2]int{target, in.Offset}
		if seen[key] {
			continue
		}
		seen[key] = true
		l := Loop{Start: target, End: in.Offset}
		for _, b := range f.Insns {
			if b.Offset >= target && b.Offset <= in.Offset {
				l.Insns = append(l.Insns, b)
			}
		}
		out = append(out, l)
	}
	return out
}

func isMorestack(in Insn) bool {
	return strings.HasPrefix(in.Op, "CALL") && strings.Contains(in.Args, "runtime.morestack")
}

// Arith counts the floating-point work in a loop body.
func (l Loop) Arith() int {
	n := 0
	for _, in := range l.Insns {
		if isArith(in.Op) {
			n++
		}
	}
	return n
}

// contains reports whether l encloses inner as a strictly smaller range.
func (l Loop) contains(inner Loop) bool {
	if l.Start == inner.Start && l.End == inner.End {
		return false
	}
	return inner.Start >= l.Start && inner.End <= l.End
}

// SteadyLoop returns the loop this audit is about: among the innermost loops,
// the one carrying the most floating-point arithmetic, and on a tie the
// tightest.
//
// Innermost first, then arithmetic — the order matters, and both halves are
// doing work:
//
//   - Innermost, because an outer loop's range subsumes its inner loops'
//     instructions. Picking by arithmetic alone would report the N-loop of a
//     blocked kernel (P3) as the steady state, and its "spills" would include
//     everything the K-loop does.
//   - Most arithmetic, because a ×4-unrolled K-loop leaves a remainder loop
//     behind that is also innermost and also has FMAs in it. The unrolled body
//     is the steady state — it is where iterations are spent — and it is the one
//     whose register pressure the tile shape was chosen for. The remainder runs
//     at most three times per call.
//
// A kernel whose remainder loop spills while its main loop does not would pass
// this audit. That is a real limitation, recorded rather than hidden: at KC=128
// the remainder is at most 3 of 128 iterations, and P3's blocking chooses KC as
// a multiple of the unroll, so the remainder exists for correctness on
// user-supplied k rather than for the hot path.
func (f Func) SteadyLoop() (Loop, error) {
	loops := f.Loops()
	if len(loops) == 0 {
		return Loop{}, fmt.Errorf("%s: no loop found (no backward branch in the listing)", f.Name)
	}
	var innermost []Loop
	for _, l := range loops {
		nested := false
		for _, other := range loops {
			if l.contains(other) {
				nested = true
				break
			}
		}
		if !nested {
			innermost = append(innermost, l)
		}
	}
	best, bestArith := Loop{}, -1
	for _, l := range innermost {
		n := l.Arith()
		if n > bestArith || (n == bestArith && l.End-l.Start < best.End-best.Start) {
			best, bestArith = l, n
		}
	}
	return best, nil
}

// Report is what the audit found in one loop body.
type Report struct {
	Func  string
	Loop  Loop
	Insns int // real instructions in the body, excluding pseudo-ops

	Arith      int // FMA and other float arithmetic
	VecStack   []Insn
	OtherMem   []Insn // memory references that move no vector register
	VecCopies  int    // register-to-register vector moves: issue slots, not spills
	Broadcasts int    // scalar-to-vector splats: arithmetic setup, not copies
	Nops       int    // statement-position anchors (T9)
	Calls      []Insn
	// PanicExits are conditional branches leaving the loop body for a block
	// that calls a runtime panic — a bounds check the prover failed to remove.
	// See Func.panicExits for why the audit looks for the branch rather than
	// for the call.
	PanicExits []Insn
}

// Audit classifies a loop body.
//
// It takes the whole function rather than just the loop because two of the
// properties P2 cares about are not visible inside the body: a surviving bounds
// check shows up as a branch *out* of it, and telling that branch apart from the
// loop's own exit needs the instructions at the target.
func Audit(f Func, l Loop) Report {
	r := Report{Func: f.Name, Loop: l, Insns: len(l.Insns)}
	for _, in := range l.Insns {
		if strings.HasPrefix(in.Op, "CALL") {
			r.Calls = append(r.Calls, in)
		}
		if isArith(in.Op) {
			r.Arith++
		}
		if isNop(in.Op, in.Args) {
			r.Nops++
			continue
		}
		mem := memRE.MatchString(in.Args)
		vec := vecRegRE.MatchString(in.Args)
		switch {
		case mem && vec && stackRE.MatchString(in.Args):
			r.VecStack = append(r.VecStack, in)
		case mem && !vec:
			r.OtherMem = append(r.OtherMem, in)
		case isBroadcast(in.Op):
			r.Broadcasts++
		case !mem && vec && isVecMove(in.Op):
			r.VecCopies++
		}
	}
	r.PanicExits = f.panicExits(l)
	return r
}

// panicLookahead is how many instructions past a branch target to search for the
// panic call.
//
// A failed bounds check compiles to a conditional branch to a small block that
// loads the index and length into argument registers and calls
// runtime.panicIndex or a sibling. Eight instructions is comfortably more than
// any of those blocks needs and short enough not to wander into unrelated code;
// if the compiler ever grows a longer panic prologue, this audit under-reports
// rather than mis-reports, and the spill and call criteria still hold.
const panicLookahead = 8

// panicExits returns the conditional branches out of l whose target reaches a
// runtime panic call.
//
// This is how the audit checks DESIGN.md §4/P2's "pre-sliced panels", i.e.
// bounds-check elimination in the K-loop. The obvious check — grep the loop body
// for a CALL — does not work, because the panic block is laid out *after* the
// function's hot path and only the branch to it is in the body. And the obvious
// alternative — `-d=ssa/check_bce` over the package — is not the criterion
// either: it reports every bounds check anywhere in the file, including the
// prologue's `a[:kc*MR]` and the write-out's `c[i*ldc:...]`, both of which are
// outside the loop and cost nothing amortized over k. Dozens of those are
// expected and fine. What must be zero is the ones inside the body, and that is
// what this finds. The gate prints check_bce's count as provenance and enforces
// this.
func (f Func) panicExits(l Loop) []Insn {
	var out []Insn
	for _, in := range l.Insns {
		if !strings.HasPrefix(in.Op, "J") {
			continue
		}
		m := targetRE.FindStringSubmatch(strings.TrimSpace(in.Args))
		if m == nil {
			continue
		}
		target, err := strconv.Atoi(m[1])
		if err != nil || (target >= l.Start && target <= l.End) {
			continue
		}
		if f.reachesPanic(target) {
			out = append(out, in)
		}
	}
	return out
}

// reachesPanic reports whether a runtime panic call sits within
// panicLookahead instructions of the given offset.
//
// The target is resolved to the first instruction at *or after* it rather than to
// an exact offset match, and that is load-bearing rather than defensive. Parse
// drops NOP lines, and a NOP is two different things in one mnemonic: the
// zero-length inlining marker, which shares the following instruction's offset
// (T9), and real alignment padding, which owns its offset and is several bytes
// wide. When the compiler aligns an out-of-line panic block, the branch targets
// the padding — so an exact match found nothing, this returned false, and the
// bounds check was invisible.
//
// That was not hypothetical: it is why avx512Scal and avx512Axpy were reported
// with 0 bounds-check exits while carrying one each (issue #46). A miss here is
// the worst kind this tool can have, because the P2 gate turns it into a passing
// criterion — "0 surviving bounds checks in the steady-state K-loop" — and a
// detector that cannot fire certifies nothing. Resolving forward is also strictly
// conservative in the direction that matters: where an exact match existed it
// finds the same instruction, so no previously-reported exit disappears.
func (f Func) reachesPanic(offset int) bool {
	for i, in := range f.Insns {
		if in.Offset < offset {
			continue
		}
		for j := i; j < len(f.Insns) && j < i+panicLookahead; j++ {
			c := f.Insns[j]
			if strings.HasPrefix(c.Op, "CALL") && strings.Contains(c.Args, "runtime.panic") {
				return true
			}
		}
		return false
	}
	return false
}

// Spills reports the count the P2 gate criterion is stated in terms of.
func (r Report) Spills() int { return len(r.VecStack) }

// Memory returns every instruction in the body with a memory operand, for the
// peak kernels' no-memory property (issue #11, property 1).
func (r Report) Memory() []Insn {
	var out []Insn
	for _, in := range r.Loop.Insns {
		if memRE.MatchString(in.Args) {
			out = append(out, in)
		}
	}
	return out
}

// Summary is the one-line form the gate prints.
func (r Report) Summary() string {
	perFMA := "n/a"
	if r.Arith > 0 {
		perFMA = fmt.Sprintf("%.2f", float64(r.Insns)/float64(r.Arith))
	}
	return fmt.Sprintf("%s: steady-state loop [%d,%d] %d insns for %d arith (%s per arith): "+
		"%d vector stack refs, %d reg copies, %d broadcasts, %d anchor nops, %d calls, "+
		"%d bounds-check exits, %d other mem refs",
		r.Func, r.Loop.Start, r.Loop.End, r.Insns, r.Arith, perFMA,
		len(r.VecStack), r.VecCopies, r.Broadcasts, r.Nops, len(r.Calls),
		len(r.PanicExits), len(r.OtherMem))
}
