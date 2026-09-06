// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
)

// Shape is one candidate microkernel: MR rows by NR columns, unrolled U k-steps,
// in one of the two accumulate forms.
//
// NR is stored as a count of 16-lane vectors rather than columns, because that is
// the quantity every derived number is in terms of: accumulators are MR*V, B loads
// per k-step are V, and the register budget is counted in vectors. Columns are
// V*16 and appear only in names and slice bounds.
type Shape struct {
	MR   int  // rows of C the tile accumulates
	V    int  // 16-lane vectors per row, so NR == V*16
	U    int  // k-steps per steady-state body
	Form Form // how a row's A scalar reaches all 16 lanes
}

// Form is the accumulate form. The two are not variants of a style; they trade a
// memory operand for an ALU operand, which is why the sweep audits both.
type Form int

const (
	// Broadcast reaches all 16 lanes with one VBROADCASTSS per row per k-step,
	// reading the A panel through archsimd.BroadcastFloat32x16. Every shipped
	// kernel is this form.
	Broadcast Form = iota
	// Permute loads a 16-lane window of the A panel once per body and reaches
	// each row's lane with archsimd.Float32x16.Permute against a hoisted index
	// vector. It trades MR*U memory broadcasts for one load plus MR*U permutes.
	Permute
)

func (f Form) String() string {
	if f == Permute {
		return "permute"
	}
	return "broadcast"
}

// NR is the tile's column count: V vectors of the active ISA's lane width (16 for
// AVX-512's Float32x16, 4 for NEON's Float32x4). isa defaults to amd64, so on the
// unset-arch default this is the original V*16.
func (s Shape) NR() int { return s.V * isa.Lanes }

// Accs is the number of accumulator registers the shape needs live across the
// whole call, and therefore the number of independent dependency chains the FMAs
// form. Both readings are the same number, which is what makes the chain term in
// objective.go computable without any new instrument.
func (s Shape) Accs() int { return s.MR * s.V }

// FMAs is the FMA count in one steady-state body.
func (s Shape) FMAs() int { return s.MR * s.V * s.U }

// Name is the emitted function's name. It matches the shipped kernels' names, so
// an emission of a shipped shape is comparable to the tree by text and not only
// by audit reading.
func (s Shape) Name() string { return fmt.Sprintf("Kernel%dx%d", s.MR, s.NR()) }

// Label identifies a shape in sweep output.
func (s Shape) Label() string {
	return fmt.Sprintf("%dx%d u=%d %s", s.MR, s.NR(), s.U, s.Form)
}

// PermuteWindowExact reports whether the Permute form's single 16-lane A-panel
// load both stays in bounds and covers every index the body needs.
//
// Two inequalities, and they close on one value. The A panel is k-major with MR
// floats per k-step, so a body's loop condition guarantees exactly MR*U floats:
//
//	MR*U >= 16   or the 16-lane load reads past what the condition guarantees,
//	             and past the panel itself on the last iteration
//	MR*U <= 16   or the body needs an index beyond lane 15, which one window
//	             cannot supply
//
// So MR*U == 16 exactly, which is (MR,U) in {(2,8),(4,4),(8,2)}. Everything else
// is either an out-of-bounds read or a reference to a lane the window does not
// hold — the first draft of this predicate tested only the lower bound and duly
// emitted code referring to an idx16 that was never hoisted.
//
// docs/spill-report.md:202-206 attributes the best zero-spill reading in either
// form, 4.438 insns/FMA, to `2x64 u=2` in the Permute form. MR*U there is 4, so
// that shape satisfies neither inequality, and whatever kernel produced 4.438 did
// not read its A panel the way the shipped kernels read theirs. What it did do is
// not recorded and is not inferable from the surviving prose, so this generator
// does not reproduce that reading. #107 predicted exactly this; the disagreement
// is reported by `-sweep` rather than closed by guessing.
func (s Shape) PermuteWindowExact() bool { return s.MR*s.U == 16 }

// accNames returns the accumulator variable names for one row.
//
// The two-vector case is named l/h — columns 0-15 and 16-31 — because that is what
// the shipped kernels call them and NR=32 is every shipped shape. Naming is not a
// property the audit can see, so this exists only so that an emission of a shipped
// shape is diffable against the tree.
func (s Shape) accNames(row int) []string {
	if s.V == 2 {
		return []string{fmt.Sprintf("c%dl", row), fmt.Sprintf("c%dh", row)}
	}
	out := make([]string, s.V)
	for j := range out {
		out[j] = fmt.Sprintf("c%dv%d", row, j)
	}
	return out
}

// bNames returns the B-vector variable names.
func (s Shape) bNames() []string {
	if s.V == 2 {
		return []string{"bl", "bh"}
	}
	out := make([]string, s.V)
	for j := range out {
		out[j] = fmt.Sprintf("b%d", j)
	}
	return out
}

// Emit writes the candidate as a compilable file in package vec.
//
// The body is the shipped kernels' idiom, and deliberately so: every property P2
// audits is a property of the emitted instruction stream, so a generator whose
// emission differs in shape from what ships would be measuring a kernel the
// project would not write. Three details are load-bearing rather than stylistic,
// and internal/vec/gemm_amd64.go:36-52 is their authority:
//
//   - Both loop conditions are slice lengths, not a counter, and both panels are
//     re-sliced at the bottom of the body. That is what eliminates the bounds
//     checks: `len(bp) >= NR*U` is exactly the fact the prover needs, and it holds
//     by construction on entry.
//   - C is read and written outside the K-loop, so the body touches only panels
//     and registers.
//   - The remainder loop exists only when U > 1, for correctness on user-supplied
//     k. At U == 1 the main loop already handles every k-step, and the shipped
//     Kernel4x32 has no remainder loop for that reason.
//
// Emit writes the candidate as a compilable, self-contained file: raw archsimd,
// no dependency on package vec's shim, so it compiles in isolation in a dot-dir
// under internal/vec (see audit()). It is parameterized by the active ISA's lane
// width — Float32x16 on amd64, Float32x4 on arm64 — through isa.Lanes, so one body
// serves both. On the default (amd64) it is byte-identical to what shipped before
// the parameterization; -verify proves that against gemm_amd64.go.
//
// The emitted amd64 body matches gemm_amd64.go by text. The arm64 body does NOT
// match gemm_neon.go by text, and cannot: the tree's NEON kernels call the vec
// shim (Load128/FMA128/…), which an isolated candidate in its own package cannot
// name, so this emits the equivalent raw archsimd. The two produce the same object
// code — the shim wrappers inline away — which is why arm64 -verify binds on the
// audit report (identical insns/FMAs/spills/…) rather than on the text.
func (s Shape) Emit() string {
	if isa.Arch == "arm64" {
		return s.emitNEON()
	}
	return s.emitRawArchsimd()
}

// emitRawArchsimd is the amd64 emitter: self-contained raw archsimd (Float32x16),
// which matches gemm_amd64.go by text and compiles in isolation. Parameterized by
// isa.Lanes, but only ever reached with isa.Arch=="amd64" (lanes 16) now that arm64
// has its own shim-form emitter; the parameterization is retained so the default
// path is provably the pre-parameterization output (-verify).
func (s Shape) emitRawArchsimd() string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	typ := fmt.Sprintf("archsimd.Float32x%d", isa.Lanes)

	p("// Copyright 2026 Scott Friedman")
	p("// SPDX-License-Identifier: Apache-2.0")
	p("")
	p("//go:build goexperiment.simd && %s", isa.Arch)
	p("")
	p("package vec")
	p("")
	p(`import "simd/archsimd"`)
	p("")
	p("// %s is a generated candidate: %s.", s.Name(), s.Label())
	p("//")
	p("// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.")
	p("func %s(kc int, a, b, c []float32, ldc int) {", s.Name())

	// Accumulators. The zero vector is a zeroed register, so no broadcast is
	// needed to start them (gemm_amd64.go:81).
	p("\tvar (")
	for r := 0; r < s.MR; r++ {
		p("\t\t%s %s", strings.Join(s.accNames(r), ", "), typ)
	}
	p("\t)")

	bn := s.bNames()
	if s.Form == Broadcast {
		p("\tvar %s, av %s", strings.Join(bn, ", "), typ)
	} else {
		p("\tvar %s, aw, av %s", strings.Join(bn, ", "), typ)
		// The index vectors are loop-invariant, so they are hoisted here. Each
		// costs a live register for the whole call, which is the Permute form's
		// price and the reason its register pressure differs from Broadcast's.
		// Permute is amd64-only (16-lane); space() never enumerates it on arm64.
		for i := 0; i < s.MR*s.U && i < 16; i++ {
			p("\tidx%d := archsimd.BroadcastUint32x16(%d)", i, i)
		}
	}
	p("")
	p("\tap := a[:kc*%d]", s.MR)
	p("\tbp := b[:kc*%d]", s.NR())
	p("")

	p("\tfor len(ap) >= %d && len(bp) >= %d {", s.MR*s.U, s.NR()*s.U)
	s.emitBody(p, s.U, true)
	p("\t\tap, bp = ap[%d:], bp[%d:]", s.MR*s.U, s.NR()*s.U)
	p("\t}")

	if s.U > 1 {
		p("")
		p("\t// Remainder: kc mod %d k-steps. Correctness for user-supplied k, not a", s.U)
		p("\t// hot path — P3 chooses KC as a multiple of the unroll.")
		p("\tfor len(ap) >= %d && len(bp) >= %d {", s.MR, s.NR())
		s.emitBody(p, 1, false)
		p("\t\tap, bp = ap[%d:], bp[%d:]", s.MR, s.NR())
		p("\t}")
	}

	// Write-out, one row at a time, outside the K-loop.
	p("")
	for r := 0; r < s.MR; r++ {
		assign := "r ="
		if r == 0 {
			assign = "r :="
		}
		p("\t%s c[%d*ldc : %d*ldc+%d]", assign, r, r, s.NR())
		for j, acc := range s.accNames(r) {
			lo, hi := j*isa.Lanes, j*isa.Lanes+isa.Lanes
			p("\tarchsimd.LoadFloat32x%d(r[%d:%d]).Add(%s).Store(r[%d:%d])", isa.Lanes, lo, hi, acc, lo, hi)
		}
	}
	p("}")
	return b.String()
}

// emitNEON is the arm64 emitter. It reproduces gemm_neon.go's idiom — Float32x4
// accumulators named c{row}_{col}, the vec shim wrappers Load128/Broadcast128/
// FMA128/Add128/Store128, and the nested-call write-out — because the shim is what
// the shipped NEON kernels use, and each bodied wrapper costs an anchor NOP in the
// loop (golang/go#80830, keel neon-probe). A raw-archsimd emission would audit ~26
// NOPs lighter than the tree and mint a frontier the shipped kernels do not sit on
// — the rank inversion #156 exists to avoid. So it emits the shim.
//
// It is vec-QUALIFIED (vec.Load128, package shapegencand, importing internal/vec)
// rather than package-local like gemm_neon.go: audit() compiles one candidate in
// isolation in its own dir, where a package-local Load128 is undefined and a
// same-named package-vec Kernel would collide with the real one. The qualified shim
// inlines to the identical object code, so the audit report matches the tree — which
// is why arm64 -verify binds on the audit, not the text. NEON has no window-permute,
// so Broadcast only; space() never enumerates Permute on arm64.
func (s Shape) emitNEON() string {
	lanes := isa.Lanes // 4
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }

	p("// Copyright 2026 Scott Friedman")
	p("// SPDX-License-Identifier: Apache-2.0")
	p("")
	p("//go:build goexperiment.simd && arm64")
	p("")
	p("package shapegencand")
	p("")
	p(`import "github.com/scttfrdmn/keel/internal/vec"`)
	p("")
	p("// %s is a generated candidate: %s.", s.Name(), s.Label())
	p("//")
	p("// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.")
	p("func %s(kc int, a, b, c []float32, ldc int) {", s.Name())

	for r := 0; r < s.MR; r++ {
		names := make([]string, s.V)
		for j := range names {
			names[j] = fmt.Sprintf("c%d_%d", r, j)
		}
		p("\tvar %s vec.F32x4", strings.Join(names, ", "))
	}
	bn := make([]string, s.V)
	for j := range bn {
		bn[j] = fmt.Sprintf("b%d", j)
	}
	p("\tvar %s, av vec.F32x4", strings.Join(bn, ", "))

	p("\tap := a[:kc*%d]", s.MR)
	p("\tbp := b[:kc*%d]", s.NR())

	p("\tfor len(ap) >= %d && len(bp) >= %d {", s.MR*s.U, s.NR()*s.U)
	s.emitBodyNEON(p, lanes, s.U)
	p("\t\tap, bp = ap[%d:], bp[%d:]", s.MR*s.U, s.NR()*s.U)
	p("\t}")

	if s.U > 1 {
		// Remainder: kc mod U k-steps, for user-supplied k. All shipped NEON kernels
		// are U=1 and have no remainder loop, matching gemm_amd64.go's rationale.
		p("\tfor len(ap) >= %d && len(bp) >= %d {", s.MR, s.NR())
		s.emitBodyNEON(p, lanes, 1)
		p("\t\tap, bp = ap[%d:], bp[%d:]", s.MR, s.NR())
		p("\t}")
	}

	for r := 0; r < s.MR; r++ {
		p("\tr%d := c[%d*ldc : %d*ldc+%d]", r, r, r, s.NR())
		for j := 0; j < s.V; j++ {
			lo, hi := j*lanes, j*lanes+lanes
			p("\tvec.Store128(r%d[%d:%d], vec.Add128(vec.Load128(r%d[%d:%d]), c%d_%d))", r, lo, hi, r, lo, hi, r, j)
		}
	}
	p("}")
	return b.String()
}

// emitBodyNEON writes u k-steps of the NEON steady state (gemm_neon.go's shape):
// one shim B load per column, then per row a Broadcast128 of the A scalar and V
// fused FMA128s into that row's accumulators.
func (s Shape) emitBodyNEON(p func(string, ...any), lanes, u int) {
	bn := make([]string, s.V)
	for j := range bn {
		bn[j] = fmt.Sprintf("b%d", j)
	}
	for k := 0; k < u; k++ {
		loads := make([]string, s.V)
		for j := range loads {
			off := k*s.NR() + j*lanes
			loads[j] = fmt.Sprintf("vec.Load128(bp[%d:%d])", off, off+lanes)
		}
		p("\t\t%s = %s", strings.Join(bn, ", "), strings.Join(loads, ", "))
		for r := 0; r < s.MR; r++ {
			p("\t\tav = vec.Broadcast128(ap[%d])", k*s.MR+r)
			acc := make([]string, s.V)
			fmas := make([]string, s.V)
			for j := range acc {
				acc[j] = fmt.Sprintf("c%d_%d", r, j)
				fmas[j] = fmt.Sprintf("vec.FMA128(av, %s, c%d_%d)", bn[j], r, j)
			}
			p("\t\t%s = %s", strings.Join(acc, ", "), strings.Join(fmas, ", "))
		}
	}
}

// emitBody writes u k-steps of the steady state. label controls the `// k + N`
// comments, which the shipped kernels carry only in their unrolled bodies.
func (s Shape) emitBody(p func(string, ...any), u int, label bool) {
	bn := s.bNames()
	for k := 0; k < u; k++ {
		if label && u > 1 {
			if k > 0 {
				p("")
			}
			p("\t\t// k + %d", k)
		}
		loads := make([]string, s.V)
		for j := range loads {
			off := k*s.NR() + j*isa.Lanes
			loads[j] = fmt.Sprintf("archsimd.LoadFloat32x%d(bp[%d:%d])", isa.Lanes, off, off+isa.Lanes)
		}
		p("\t\t%s = %s", strings.Join(bn, ", "), strings.Join(loads, ", "))

		if s.Form == Permute && k == 0 {
			// One 16-lane load per body serves every row of every k-step. Guarded
			// by PermuteWindowExact, so the loop condition guarantees the read.
			// Permute is amd64-only, so the 16 here is not lane-parameterized.
			p("\t\taw = archsimd.LoadFloat32x16(ap[0:16])")
		}
		for r := 0; r < s.MR; r++ {
			i := k*s.MR + r
			if s.Form == Permute {
				p("\t\tav = aw.Permute(idx%d)", i)
			} else {
				p("\t\tav = archsimd.BroadcastFloat32x%d(ap[%d])", isa.Lanes, i)
			}
			acc := s.accNames(r)
			fmas := make([]string, s.V)
			for j := range fmas {
				fmas[j] = fmt.Sprintf("av.MulAdd(%s, %s)", bn[j], acc[j])
			}
			p("\t\t%s = %s", strings.Join(acc, ", "), strings.Join(fmas, ", "))
		}
	}
}
