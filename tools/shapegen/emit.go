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

// NR is the tile's column count.
func (s Shape) NR() int { return s.V * 16 }

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
func (s Shape) Emit() string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }

	p("// Copyright 2026 Scott Friedman")
	p("// SPDX-License-Identifier: Apache-2.0")
	p("")
	p("//go:build goexperiment.simd && amd64")
	p("")
	p("package vec")
	p("")
	p(`import "simd/archsimd"`)
	p("")
	p("// %s is a generated candidate: %s.", s.Name(), s.Label())
	p("//")
	p("// Emitted by tools/shapegen. Not a shipped kernel; see KERNEL.md.")
	p("func %s(kc int, a, b, c []float32, ldc int) {", s.Name())

	// Accumulators. The zero Float32x16 is a zeroed register, so no broadcast is
	// needed to start them (gemm_amd64.go:81).
	p("\tvar (")
	for r := 0; r < s.MR; r++ {
		p("\t\t%s archsimd.Float32x16", strings.Join(s.accNames(r), ", "))
	}
	p("\t)")

	bn := s.bNames()
	if s.Form == Broadcast {
		p("\tvar %s, av archsimd.Float32x16", strings.Join(bn, ", "))
	} else {
		p("\tvar %s, aw, av archsimd.Float32x16", strings.Join(bn, ", "))
		// The index vectors are loop-invariant, so they are hoisted here. Each
		// costs a live register for the whole call, which is the Permute form's
		// price and the reason its register pressure differs from Broadcast's.
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
			lo, hi := j*16, j*16+16
			p("\tarchsimd.LoadFloat32x16Slice(r[%d:%d]).Add(%s).StoreSlice(r[%d:%d])", lo, hi, acc, lo, hi)
		}
	}
	p("}")
	return b.String()
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
			off := k*s.NR() + j*16
			loads[j] = fmt.Sprintf("archsimd.LoadFloat32x16Slice(bp[%d:%d])", off, off+16)
		}
		p("\t\t%s = %s", strings.Join(bn, ", "), strings.Join(loads, ", "))

		if s.Form == Permute && k == 0 {
			// One 16-lane load per body serves every row of every k-step. Guarded
			// by PermuteWindowExact, so the loop condition guarantees the read.
			p("\t\taw = archsimd.LoadFloat32x16Slice(ap[0:16])")
		}
		for r := 0; r < s.MR; r++ {
			i := k*s.MR + r
			if s.Form == Permute {
				p("\t\tav = aw.Permute(idx%d)", i)
			} else {
				p("\t\tav = archsimd.BroadcastFloat32x16(ap[%d])", i)
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
