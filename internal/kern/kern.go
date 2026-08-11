// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package kern holds the SGEMM microkernels (DESIGN.md §4/P2). Straight-line
// code, no calls inside the K-loop, pre-sliced panels, pointer-free data.
// The tile protocol and the measured spill frontier are recorded in KERNEL.md.
//
// # The tile is reflected: MR rows by NR columns, not the other way round
//
// DESIGN.md §4/P2 specifies MR=32, NR=6: two 16-lane vectors along M and six
// scalar columns. This package implements the same tile with M and N exchanged —
// rows counted in MR, columns in NR, vectors along N — for one reason:
// DESIGN.md §3 makes keel's public API row-major.
//
// In a row-major C, sixteen consecutive elements of a *column* are ldc floats
// apart. A tile vectorized along M would have to write each accumulator lane to
// a different cache line: 192 scalar stores per tile, each needing a lane
// extracted from a vector register. archsimd has no cheap general lane extract
// for Float32x16; the natural way to get lane i out is to store the vector and
// reload the element, which is literally a spill. So the M-vectorized tile would
// fail P2's own spill audit by construction, for a reason that has nothing to do
// with the compiler being audited.
//
// Vectorizing along N instead makes every vector load and store — on the B panel
// and on C — contiguous, and turns the A operand into a scalar broadcast. This is
// a discrepancy in the design document rather than a departure from it (issue
// #16); the arithmetic and the register pressure the phase was designed to test
// are unchanged.
//
// # Why there is more than one shape, and why none of them has 12 accumulators
//
// DESIGN.md budgeted 12 accumulators + 2 B vectors + 1 broadcast = 15 live
// vector registers and called that exactly full. Two properties of go1.26.5
// (docs/toolchain-notes.md T10, issue #18) say it is one register short:
//
//   - The register allocator offers SIMD values only X0–X14. Fifteen, not
//     thirty-two: X15 is the ABI zero register and X16–X31 are in no allocatable
//     mask at all.
//   - Only the 213 FMA form exists, and it writes to its first multiplicand. An
//     accumulate `acc += a·b` therefore cannot land in acc's own register and
//     needs one live scratch register beyond the working set, always.
//
// 12 + 2 + 1 scratch = 15 leaves nothing to hold the A broadcast. Measured
// across 115 generated shapes, every 12-accumulator configuration spills; the
// zero-spill frontier is at 8, and it is reached by shrinking M rather than N,
// because N is where the vectors are and M is what costs a broadcast.
//
// So this package ships two clean shapes rather than picking one on theory:
//
//	2×32 unroll 4   4 accumulators, 4.62 instructions per FMA — the fewest
//	4×32 unroll 1   8 accumulators, 0.75 loads per FMA — the fewest, and a floor:
//	                a lower ratio needs 9 accumulators (KERNEL.md §3)
//
// The first minimizes instructions issued per unit of arithmetic; the second
// minimizes loads per unit of arithmetic. Which one wins is a property of the
// host's front-end width and load ports, not of the source, so the benchmark
// decides and KERNEL.md records the answer per host.
//
// ReferenceTile — DESIGN.md's 12-accumulator tile, which spills — is kept and
// benchmarked deliberately. It is excluded from Kernels() so that nothing ships
// it and the spill gate stays binding, and it exists so the P2 report can state
// the cost of the T10 constraint as a measured number instead of a spill count.
// Measured() is Kernels() plus that tile: what the benchmark runs.
//
// # The packed panel protocol
//
// Both panels are k-major, which is what makes the K-loop a straight walk
// forward through memory with no index arithmetic:
//
//	a[p*MR+i] = A[i0+i][p0+p]   MR floats per k, one per row of the tile
//	b[p*NR+j] = B[p0+p][j0+j]   NR floats per k, contiguous columns
//	c[i*ldc+j]                  the tile in row-major C, row stride ldc
//
// The kernel computes C += A·B for the tile. Alpha and beta are P3's business,
// as kernel variants rather than as branches in this loop; a P2 microkernel that
// multiplied by alpha would be measuring a different loop than the one P3 ships.
//
// P3 will produce these panels in internal/pack. For P2 the packing is done by
// the tests and benchmarks, because the phase's question is what the compiler
// does with the K-loop, and packed input is the premise of that question.
package kern

import "fmt"

// Tile bounds. They exist to size the scalar reference's accumulator buffer and
// to bound what a caller may ask for; the shapes actually implemented are in
// Kernels().
const (
	// MaxMR is the largest number of rows any kernel in this package computes.
	MaxMR = 8
	// MaxNR is the largest number of columns any kernel in this package
	// computes. Four 16-lane vectors.
	MaxNR = 64
)

// Kernel is one backend's microkernel for one tile shape.
//
// Fn computes C += A·B, where a holds at least kc*MR packed floats, b at least
// kc*NR, and c addresses the tile in row-major memory with row stride ldc, so it
// must have at least (MR-1)*ldc+NR elements. Fn assumes those lengths rather
// than checking them: the caller — P3's blocking loops, or a test — is what
// guarantees them, because a bounds check in the K-loop is the thing this phase
// exists to avoid.
//
// MR, NR and Unroll are carried on the value rather than being package
// constants because the shape is a measurement result, not a fixed property of
// the package: the tests have to pack panels for whatever shape they are handed,
// and the benchmark has to compare shapes against each other.
type Kernel struct {
	Name   string // backend: Scalar or AVX512
	MR, NR int    // tile shape
	Unroll int    // k-steps per pass of the steady-state loop
	Fn     func(kc int, a, b, c []float32, ldc int)
}

// Backend names, shared with the vec shim's vocabulary.
const (
	Scalar = "scalar"
	AVX512 = "avx512"
)

// Tile is the shape as it appears in benchmark and gate names, e.g. "4x32".
func (k Kernel) Tile() string { return fmt.Sprintf("%dx%d", k.MR, k.NR) }

// ID names one kernel uniquely: shape then backend, e.g. "4x32/avx512". This is
// the string the benchmark sub-names and the gate's thresholds are keyed on.
func (k Kernel) ID() string { return k.Tile() + "/" + k.Name }

// Ref is the scalar reference for k's shape — the kernel the differential test
// holds k to. Every vector kernel has one, and it is derived from the shape
// rather than written per shape, so a new tile cannot arrive without its
// reference.
func (k Kernel) Ref() Kernel { return ScalarKernel(k.MR, k.NR) }

// Kernels returns every microkernel runnable on this machine: the vector shapes
// widest-tile-first, then the scalar reference for each shape they use, so that
// the gate can tell "all backends passed" from "only one was built".
//
// ReferenceTile is deliberately absent; see the package doc and Measured().
func Kernels() []Kernel {
	vs := vectorKernels()
	out := make([]Kernel, 0, len(vs)+2)
	out = append(out, vs...)
	seen := map[string]bool{}
	for _, v := range vs {
		if r := v.Ref(); !seen[r.ID()] {
			seen[r.ID()] = true
			out = append(out, r)
		}
	}
	if len(vs) == 0 {
		out = append(out, ScalarKernel(2, 32), ScalarKernel(4, 32))
	}
	return out
}

// Measured returns everything the benchmark runs: Kernels() plus the reference
// tiles that are measured but not shipped. The two lists are separate so that a
// shape can be evidence without being a product, and so that the gate's
// zero-spill criterion can be enforced on Kernels() while the audit of a
// deliberately-spilling shape is printed as provenance.
func Measured() []Kernel { return append(Kernels(), referenceTiles()...) }

// Backends returns the distinct backend names in Kernels(), for the coverage
// marker the gate parses. Names, not shapes: the gate's question is whether the
// AVX-512 path ran at all.
func Backends() []string {
	seen := map[string]bool{}
	var out []string
	for _, k := range Kernels() {
		if !seen[k.Name] {
			seen[k.Name] = true
			out = append(out, k.Name)
		}
	}
	return out
}

// ScalarKernel returns the executable specification of an mr×nr tile: the same
// arithmetic in the same order as the vector kernels, with no vector type in
// sight. It builds on a stock toolchain on every GOARCH, and the differential
// test holds every vector kernel of that shape to it.
func ScalarKernel(mr, nr int) Kernel {
	return Kernel{
		Name:   Scalar,
		MR:     mr,
		NR:     nr,
		Unroll: 1,
		Fn: func(kc int, a, b, c []float32, ldc int) {
			ScalarTile(mr, nr, kc, a, b, c, ldc)
		},
	}
}

// ScalarTile computes C += A·B for one mr×nr tile.
//
// The loop order is k, then i, then j — the same order the vector kernels
// accumulate in, so the two agree up to reassociation of nothing at all. The
// more natural i,j,k (a dot product per output element) would have given a
// *different* summation order, which would then have had to be excused in the
// tolerance rather than explained.
//
// It accumulates into a local tile rather than into C directly so that C is read
// once and written once, matching the vector kernels' register behaviour;
// otherwise the reference would do kc times more memory traffic than the thing
// it is a reference for. The buffer is a fixed-size array bounded by MaxMR and
// MaxNR — pointer-free, stack-allocated, and sized at compile time even though
// the shape is not.
//
// It panics on a shape it cannot hold. A reference that silently truncated would
// make the differential test pass by agreeing with nothing.
func ScalarTile(mr, nr, kc int, a, b, c []float32, ldc int) {
	if mr < 1 || mr > MaxMR || nr < 1 || nr > MaxNR {
		panic(fmt.Sprintf("kern: tile %dx%d outside %dx%d", mr, nr, MaxMR, MaxNR))
	}
	var buf [MaxMR * MaxNR]float32
	acc := buf[: mr*nr : mr*nr]
	for p := 0; p < kc; p++ {
		ap := a[p*mr : p*mr+mr]
		bp := b[p*nr : p*nr+nr]
		for i, av := range ap {
			row := acc[i*nr : i*nr+nr]
			for j, bv := range bp {
				row[j] += av * bv
			}
		}
	}
	for i := 0; i < mr; i++ {
		row := c[i*ldc : i*ldc+nr]
		src := acc[i*nr : i*nr+nr]
		for j, v := range src {
			row[j] += v
		}
	}
}
