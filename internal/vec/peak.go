// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

// Measured FMA ceiling.
//
// # Why this exists
//
// DESIGN.md §4/P2 gates the microkernel at ">=55% of theoretical peak", with
// peak derived as cores·freq·2 FMA ports·16 lanes·2 flops. That formula assumes
// two full-width 512-bit FMA units. Zen 4 does not have them: it double-pumps
// AVX-512 over 256-bit datapaths and retires one 512-bit FMA per cycle, so the
// formula overstates its ceiling by exactly 2x and would turn a 55% floor into a
// demand for ~110% of what the silicon can do. Skylake-X does have two, and the
// Zen 5 host's width was an open question rather than something to look up.
//
// So the denominator is measured, not derived (decision on issue #11): saturate
// the FMA units from registers and see what the machine actually retires. The
// formula survives only as a printed cross-check, and a divergence of >=1.5x is
// the double-pump signature rather than a bug.
//
// # What makes this a ceiling rather than just a loop
//
// Four properties, all load-bearing, all verified by disassembly (gate-p2.sh)
// rather than by trusting this comment. Three of the four were violated by the
// first draft of this file, in ways that would have moved the number by 2x-8x
// while still looking like a hardware measurement — see docs/toolchain-notes.md
// T8, and note that a denominator that is too *small* inflates every
// percent-of-peak figure computed against it.
//
// 1. No memory. Accumulators, multiplicands and the loop counter live in
// registers for the whole loop, so nothing here can be limited by load
// throughput, cache latency or store-buffer capacity. The only resource under
// contention is the FMA units, which is the point.
//
// 2. Enough independent chains to cover the latency. One chain measures FMA
// *latency* (~4 cycles), not throughput, and would report a quarter of the
// ceiling. Twelve is comfortably more than latency x ports (about 4x2 = 8) and,
// with two multiplicands live, comfortably inside AVX-512's 32 architectural
// registers. AVX2 and scalar float32 have only 16 registers, so Chains256 and
// ChainsScalar are 10: ten accumulators plus two multiplicands is twelve.
//
// 3. Chains that CSE cannot merge. Identical initial values make every chain the
// same SSA expression and the compiler folds twelve into one. Distinct starting
// values (1..Chains) prevent that, and TestPeakChainsAreIndependent asserts it
// by arithmetic rather than by reading assembly.
//
// 4. One instruction's worth of work per chain per iteration, and nothing else.
// This is why the accumulator is the *multiplicand* rather than the addend:
// `a = a*y + x`, not `a = x*y + a`. archsimd's MulAdd lowers to VFMADD213PS
// (docs/toolchain-notes.md T2), whose destination register is the first
// multiplicand and is clobbered. In the natural accumulator form the destination
// therefore cannot be the accumulator, so the register allocator has to preserve
// the multiplicand around every FMA: the AVX-512 loop body came out as twelve
// VFMADD213PS *and twenty-six* VMOVDQU64 register copies. Those copies are free
// in latency terms and can be eliminated at rename, but they still occupy issue
// slots — 38 uops against a 4-wide allocator is ~9.5 cycles of dispatch for 6
// cycles of FMA work, so on a dual-512-FMA Skylake-X the loop would have
// measured dispatch width and reported roughly two thirds of the real ceiling.
// Putting the accumulator in the destination position removes every copy. The
// dependency runs through a multiplicand instead of the addend, which costs
// nothing: FMA latency is uniform across operands on both Zen and Skylake, and
// where it is not, the multiplicand path is the conservative one.
//
// # Numerics
//
// The multiplicands are 1, so each accumulator counts iterations up from its
// distinct starting value: a*1 + 1. Once an accumulator passes 2^24 the
// additions stop changing it, which is fine and deliberate — the value is only
// ever used as an independence witness, and the alternatives are worse. Values
// that grow without bound reach +Inf, and values that shrink reach denormals,
// whose arithmetic is microcoded and drastically slower on some
// microarchitectures; either would silently turn a peak measurement into a
// special-value-handling measurement.
const (
	// Independent accumulator chains per iteration, per width.
	Chains512 = 12
	// ChainsNEON is 16: enough independent accumulator chains to saturate a Neoverse
	// V2's four 128-bit FMA pipes at ~4-cycle FMLA latency (4x4=16), which over-covers
	// c7g's V1 and any N-series (two pipes, saturating at 8). NEON has 32 vector
	// registers, so 16 accumulators plus two multiplicands (18) leaves ample headroom and
	// nothing spills. Unlike amd64's 213-form FMA, arm64's VFMLA accumulates in place
	// (#136), so each chain is one instruction with no register copy — property 4 of the
	// ceiling holds more cleanly here than on amd64.
	ChainsNEON = 16
	Chains256  = 10
	// Chains for the scalar reference ceiling.
	ChainsScalar = 10
)

// PeakKernel is one measurable arithmetic ceiling.
type PeakKernel struct {
	// Name is the backend name (BackendAVX512, BackendAVX2, BackendScalar).
	Name string
	// Chains is how many independent accumulator chains the kernel runs.
	Chains int
	// Lanes is how many float32 lanes each chain operates on.
	Lanes int
	// FlopsPerIter is how many floating-point operations one Run iteration
	// performs: chains x lanes x 2 (a multiply and an add per lane).
	FlopsPerIter int
	// Fused reports whether an iteration issues true fused multiply-adds. The
	// scalar kernel's answer is false on amd64 — Go does not fuse a separate
	// multiply and add there — so its ceiling is a different quantity and is
	// labelled as such rather than quietly compared against the vector ones.
	Fused bool
	// Run executes iters iterations and returns the horizontal sum of every
	// accumulator, so none of them can be optimized away and the result is an
	// exact witness of how many chains survived compilation (see Witness).
	Run func(iters int) float32
}

// Witness returns the exact value Run(iters) must produce.
//
// Every accumulator starts at its 1-based chain index and gains exactly 1 per
// iteration, so the sum over c chains of l identical lanes is
// l·(c(c+1)/2 + c·iters). The value is exact in float32 as long as no
// accumulator passes 2^24 and the total stays below 2^24 — the caller's job, and
// what makes this a witness rather than an approximation.
//
// This is how TestPeakChainsAreIndependent detects a collapsed kernel without
// reading assembly: the result depends on the number of chains that survived, so
// CSE merging twelve chains into one changes the answer on every host.
func (k PeakKernel) Witness(iters int) float32 {
	c, l := k.Chains, k.Lanes
	return float32(l) * float32(c*(c+1)/2+c*iters)
}

// PeakKernels returns the ceilings measurable on this machine, widest first,
// with the scalar reference always last and always present. Same shape and same
// reasoning as l1.Backends: a function rather than an init-built slice.
func PeakKernels() []PeakKernel {
	return append(vectorPeakKernels(), PeakKernel{
		Name:         BackendScalar,
		Chains:       ChainsScalar,
		Lanes:        1,
		FlopsPerIter: ChainsScalar * 2,
		Fused:        false,
		Run:          scalarPeak,
	})
}

// peakOne returns 1 without letting the compiler know that.
//
// It is not paranoia. With the multiplicands written as constants, the scalar
// kernel's `x*y` folded at compile time and its loop body came out as ten ADDSS
// and no multiply at all — half the flops FlopsPerIter claims — plus a MOVSS
// from the constant pool *inside the loop*, which is precisely the memory
// operand property 1 above forbids. The vector kernels are safe from the folding
// (an intrinsic is opaque) but take the same opaque input for uniformity and to
// stay safe from a future compiler that learns to fold through a broadcast.
//
// The call is outside the loop, so it costs one call per measurement.
//
//go:noinline
func peakOne() float32 { return 1 }

// scalarPeak is the non-vector reference ceiling: ChainsScalar independent
// multiply-accumulate chains on float32 scalars.
//
// This is written as a separate multiply and add, not as math.FMA. math.FMA
// takes float64 arguments, so a float32 chain through it would measure two
// conversions and a double rounding per operation rather than an arithmetic
// ceiling. Go's compiler does not fuse `a*b+c` into an FMA on amd64 (it does on
// arm64 and ppc64), which is exactly why PeakKernel.Fused is false here: on
// amd64 this measures the multiply and add ports, on arm64 it may measure the
// FMA unit, and pretending those are the same number would be the kind of
// undenominated comparison DESIGN.md §7 rule 7 forbids.
func scalarPeak(iters int) float32 {
	x, y := peakOne(), peakOne()
	// Distinct starting values, and the accumulator in the multiplicand
	// position: properties 3 and 4 above.
	a0, a1, a2, a3, a4 := float32(1), float32(2), float32(3), float32(4), float32(5)
	a5, a6, a7, a8, a9 := float32(6), float32(7), float32(8), float32(9), float32(10)
	for i := 0; i < iters; i++ {
		a0 = a0*y + x
		a1 = a1*y + x
		a2 = a2*y + x
		a3 = a3*y + x
		a4 = a4*y + x
		a5 = a5*y + x
		a6 = a6*y + x
		a7 = a7*y + x
		a8 = a8*y + x
		a9 = a9*y + x
	}
	return ((a0 + a1) + (a2 + a3)) + ((a4 + a5) + (a6 + a7)) + (a8 + a9)
}

// ------------------------------------------------------ float64 ceilings (#191)
//
// The float64 denominator, by the same four properties and the same chain
// counts: register file, FMA latency and operand form are per-instruction, not
// per-element, so twelve 512-bit chains cover VFMADD213PD exactly as they cover
// VFMADD213PS. Only the lane count halves. A separate type rather than a
// float64 Run on PeakKernel, so PeakKernels() and every consumer of its
// float32 Run stay untouched.

// PeakKernel64 is PeakKernel for float64: same fields, a float64 Run.
type PeakKernel64 struct {
	Name         string
	Chains       int
	Lanes        int // float64 lanes per chain
	FlopsPerIter int // Chains x Lanes x 2
	Fused        bool
	Run          func(iters int) float64
}

// Witness returns the exact value Run(iters) must produce, by PeakKernel.Witness's
// formula. In float64 it is exact while the total stays below 2^53, so it holds
// for any iteration count a benchmark runs, not only the small ones float32 allows.
func (k PeakKernel64) Witness(iters int) float64 {
	c, l := k.Chains, k.Lanes
	return float64(l) * float64(c*(c+1)/2+c*iters)
}

// PeakKernels64 returns the float64 ceilings measurable on this machine, widest
// first, with the scalar reference always last and always present.
func PeakKernels64() []PeakKernel64 {
	return append(vectorPeakKernels64(), PeakKernel64{
		Name:         BackendScalar,
		Chains:       ChainsScalar,
		Lanes:        1,
		FlopsPerIter: ChainsScalar * 2,
		Fused:        false,
		Run:          scalarPeakD,
	})
}

// peakOneD is peakOne for float64: 1, opaque to constant folding.
//
//go:noinline
func peakOneD() float64 { return 1 }

// scalarPeakD is scalarPeak on float64 scalars: a separate multiply and add,
// unfused on amd64 for the reason scalarPeak gives, hence Fused false.
func scalarPeakD(iters int) float64 {
	x, y := peakOneD(), peakOneD()
	a0, a1, a2, a3, a4 := float64(1), float64(2), float64(3), float64(4), float64(5)
	a5, a6, a7, a8, a9 := float64(6), float64(7), float64(8), float64(9), float64(10)
	for i := 0; i < iters; i++ {
		a0 = a0*y + x
		a1 = a1*y + x
		a2 = a2*y + x
		a3 = a3*y + x
		a4 = a4*y + x
		a5 = a5*y + x
		a6 = a6*y + x
		a7 = a7*y + x
		a8 = a8*y + x
		a9 = a9*y + x
	}
	return ((a0 + a1) + (a2 + a3)) + ((a4 + a5) + (a6 + a7)) + (a8 + a9)
}

// The pieces the FMA-ceiling families share (#198 arm64, #199 amd64).

// peakOnes are the family members' multiplicands: a package variable rather than
// constants, so the compiler cannot prove two of them equal and merge registers
// that must stay distinct.
var peakOnes = [8]float32{1, 1, 1, 1, 1, 1, 1, 1}
var peakOnesD = [8]float64{1, 1, 1, 1, 1, 1, 1, 1}

// peakChunk is the K-loop trip count per call of an fo member's tile: 512, the kc at
// which the FMA-only kernel loop was first measured (archive/neon-raw/).
const peakChunk = 512

func peakFill32(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

func peakFill64(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

// PeakFamily returns this build's float32 FMA-ceiling family (#198 arm64, #199 amd64), canonical
// member first, or nothing where no family exists. For instruments that must check, on the host
// being measured, that the canonical ceiling is the family's maximum (BenchmarkCeiling's
// family arm, #200).
func PeakFamily() []PeakKernel { return peakFamily() }
