// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

import "fmt"

// The objective, and why the previous one was wrong
//
// The sweep this generator replaces ranked shapes by instructions per FMA and
// nothing else. docs/spill-report.md:565 states the consequence: "the generator
// sweep never optimised for chain length at all". Ranking by insns/FMA is not a
// neutral choice of metric — it is the assertion that the front end binds, and on
// a shape with few accumulators that assertion need not hold. The old objective
// could not tell the two cases apart, and that is the defect; it is keel's own
// rather than upstream's.
//
// The corrected objective is a three-term roofline on cycles per steady-state
// body. Every term is computed from a quantity already in hand — I and F come
// from the spill audit's own report, A from the shape — so this adds no
// instrument:
//
//	issue:      cycles >= I / W       front end retires W instructions per cycle
//	port:       cycles >= F / P       P pipes accept an FMA each per cycle
//	dependency: cycles >= (F/A) * L   each of A chains is F/A serial FMAs, L deep
//
// The body's cycle count is the max of the three, and the shape's figure of merit
// is flops per cycle: each FMA is 16 lanes of multiply-add, so 32 flops.
//
// # What the corrected objective found, which is not what this comment first claimed
//
// Measured 2026-08-18, first full sweep. The draft above this one asserted that the
// shipped 2x32 is latency-bound. It is not. Its four chains do fail to keep two FMA
// ports fed — chain 16.00 cycles against port 8.00 — but its 74 instructions retire
// in 18.50 cycles at width 4, so the front end binds first and the chain term never
// surfaces. The corrected objective therefore agrees with the old one on the
// broadcast form's optimum, and 2x32 u=4 is vindicated rather than displaced.
//
// The algebra of when that stops holding is the amendment's whole practical content:
// issue > chain iff I/F > W*L/A. At W=4, L=4, A=4 the crossover is 4.00 insns/FMA,
// and docs/spill-report.md:565 records that no shape in the vanished sweep cleared
// 4.09. So on the four-accumulator family the front end binds across the entire
// achievable range, and ranking by insns/FMA ranked correctly by accident rather
// than by construction. Where the amendment does change the ranking is the
// low-accumulator, high-unroll corner: 1x48 u=8, 1x32 u=8 and 2x16 u=4 are
// dependency-bound, and there the two orders genuinely invert. The old objective
// ranks 1x48 u=8 (5.083 insns/FMA) above 2x48 u=2 (5.167); the corrected one ranks
// 2x48 u=2 above it, 24.77 flops/cycle against 24.00, because three chains at
// 4-cycle latency hold 1x48 u=8 to 32.00 cycles when its 122 instructions would
// have retired in 30.50. Neither is the frontier, so the amendment changes the
// ordering of the field without changing the shape at its head.
//
// The dependency term is the one the old objective lacked, and it is also the
// third roofline class ruled on #104: fma-bound, issue-bound, and now
// dependency-bound. A shape is dependency-bound exactly when (F/A)*L > F/P, which
// reduces to A < P*L with F cancelling — the accumulator count alone decides it,
// independently of unroll. At L=4 and P=2 the threshold is A < 8, which is why
// eight accumulators is both the zero-spill frontier T10 describes and the point
// where the chain stops binding. The two constraints meeting at the same number is
// a coincidence of this microarchitecture, not a law, and the arithmetic above is
// printed with every verdict so a host where they part company is visible rather
// than surprising.

// UArch is the three microarchitectural quantities the roofline needs.
//
// These are vendor-documented pipeline properties, not keel measurements, and they
// are named inputs rather than defaults buried in a formula so that a reading can
// be reproduced or disputed. Provenance belongs beside the number: whoever changes
// one states where it came from. They are also NOT a per-host floor — nothing here
// sets a threshold a gate compares against. They parameterise a ceiling, and the
// ruling on #104 is that a derived ceiling is legitimate exactly in this form,
// with the derivation printed.
type UArch struct {
	Name  string
	Width int // instructions retired per cycle
	Ports int // pipes that accept an FMA
	Lat   int // FMA result latency in cycles
}

// Bound names which roofline term binds a shape.
type Bound string

const (
	IssueBound Bound = "issue"
	PortBound  Bound = "port"
	ChainBound Bound = "dependency"
)

// Score is one shape's predicted steady state under a UArch.
type Score struct {
	Insns  int // steady-state body instruction count, from the audit
	FMAs   int // FMA count in that body
	Accs   int // independent dependency chains
	Cycles float64
	Bound  Bound
}

// Score applies the roofline. insns is the audited instruction count of the
// steady-state body; everything else is a property of the shape.
func (u UArch) Score(s Shape, insns int) Score {
	return u.score(insns, s.FMAs(), s.Accs())
}

func (u UArch) score(insns, fmas, accs int) Score {
	f := float64(fmas)
	issue := float64(insns) / float64(u.Width)
	port := f / float64(u.Ports)
	chain := f / float64(accs) * float64(u.Lat)

	sc := Score{Insns: insns, FMAs: fmas, Accs: accs, Cycles: issue, Bound: IssueBound}
	if port > sc.Cycles {
		sc.Cycles, sc.Bound = port, PortBound
	}
	if chain > sc.Cycles {
		sc.Cycles, sc.Bound = chain, ChainBound
	}
	return sc
}

// InsnsPerFMA is the old objective, kept because it is what gate-p2's
// SWEEP_BEST_IPF is stated in terms of and what docs/spill-report.md publishes.
// Reported beside the corrected figure, never instead of it.
func (s Score) InsnsPerFMA() float64 {
	if s.FMAs == 0 {
		return 0
	}
	return float64(s.Insns) / float64(s.FMAs)
}

// FlopsPerCycle is the corrected figure of merit. 16 lanes x 2 flops per FMA.
func (s Score) FlopsPerCycle() float64 {
	if s.Cycles == 0 {
		return 0
	}
	return 32 * float64(s.FMAs) / s.Cycles
}

// Ceiling states a frontier shape's predicted rate against the uarch's FMA peak, and
// states the two things that move that percentage, because both are large and neither
// is this tool's to settle:
//
//   - Anchor NOPs. spill.Report.Insns is len(loop.Insns), which counts the NOPs the
//     compiler emits as loop anchors, so the issue term charges for them. They do
//     occupy issue slots, so charging is defensible — but excluding them is an
//     equally statable number, and on the shipped 2x32 the two differ by 5 points.
//   - The crossover, W*L/A insns/FMA: below it the chain term takes over. It says how
//     far source-level shaping can travel before the binding term changes, which is
//     the question P2's 55%-of-peak floor turns on.
func Ceiling(u UArch, s Score, nops int) string {
	peak := 32 * float64(u.Ports)
	bare := u.score(s.Insns-nops, s.FMAs, s.Accs)
	return fmt.Sprintf(
		"ceiling: %.2f flops/cycle is %.0f%% of this uarch's %.0f (%d ports x 32 flops).\n"+
			"    %d of the %d instructions are anchor NOPs, which Report.Insns counts; without them\n"+
			"    %.2f cycles, %.2f flops/cycle, %.0f%%, %s-bound.\n"+
			"    crossover: the chain term takes over below W*L/A = %.2f insns/FMA; this shape is at %.3f.",
		s.FlopsPerCycle(), 100*s.FlopsPerCycle()/peak, peak, u.Ports,
		nops, s.Insns,
		bare.Cycles, bare.FlopsPerCycle(), 100*bare.FlopsPerCycle()/peak, bare.Bound,
		float64(u.Width*u.Lat)/float64(s.Accs), s.InsnsPerFMA())
}

// Derivation prints the three terms and the winner, so a verdict never arrives
// without the arithmetic that produced it.
func (s Score) Derivation(u UArch) string {
	return fmt.Sprintf("issue %d/%d=%.2f  port %d/%d=%.2f  chain (%d/%d)*%d=%.2f  -> %s at %.2f cycles",
		s.Insns, u.Width, float64(s.Insns)/float64(u.Width),
		s.FMAs, u.Ports, float64(s.FMAs)/float64(u.Ports),
		s.FMAs, s.Accs, u.Lat, float64(s.FMAs)/float64(s.Accs)*float64(u.Lat),
		s.Bound, s.Cycles)
}
