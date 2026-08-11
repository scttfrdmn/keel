// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package vec

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
)

// The differential harness (DESIGN.md §5.2): every backend's ops are held
// equal to the scalar spec's on identical inputs, in one binary, so a shim
// bug is caught independently of any routine-level oracle.
//
// Agreement is required to be BIT-EXACT, not within a tolerance. That is
// affordable because the scalar spec was written to match what the vector
// units do rather than what is idiomatic in Go: ScalarMulAdd rounds once
// (math.FMA), ScalarHSum folds in the same halving order as a vector reduce,
// ScalarAbs masks the sign bit. There is exactly one carve-out, in sameF32.

// backend is one implementation of the shim's Block-level op set. Every field
// is mandatory; a nil op is a registration bug, not a skip.
type backend struct {
	name      string
	Load      func([]float32) Block
	LoadPart  func([]float32) Block
	Store     func([]float32, Block)
	StorePart func([]float32, Block)
	Broadcast func(float32) Block
	Zero      func() Block
	Add       func(x, y Block) Block
	Sub       func(x, y Block) Block
	Mul       func(x, y Block) Block
	MulAdd    func(x, y, z Block) Block
	Max       func(x, y Block) Block
	Min       func(x, y Block) Block
	Abs       func(Block) Block
	HSum      func(Block) float32
}

// backends is the registry the differential tests iterate. The scalar spec is
// always index 0 and is the reference; the amd64 build appends the vector
// backends in an init in vec_amd64_test.go.
var backends = []backend{{
	name:      BackendScalar,
	Load:      ScalarLoad,
	LoadPart:  ScalarLoadPart,
	Store:     ScalarStore,
	StorePart: ScalarStorePart,
	Broadcast: ScalarBroadcast,
	Zero:      ScalarZero,
	Add:       ScalarAdd,
	Sub:       ScalarSub,
	Mul:       ScalarMul,
	MulAdd:    ScalarMulAdd,
	Max:       ScalarMax,
	Min:       ScalarMin,
	Abs:       ScalarAbs,
	HSum:      ScalarHSum,
}}

// spec is the reference backend: the scalar twin every other backend is
// measured against.
func spec() backend { return backends[0] }

// TestBackendCoverage prints which backends the differential suite actually
// ran ops against. gate-p0.sh parses this line and fails unless all three
// appear — the point being that a backend which was never compiled in, or
// which this CPU cannot execute, must not be able to masquerade as green.
//
// It is deliberately a *report*, not a skip: the test itself always passes,
// and the gate decides whether the coverage is acceptable.
func TestBackendCoverage(t *testing.T) {
	names := make([]string, 0, len(backends))
	for _, b := range backends {
		names = append(names, b.name)
	}
	t.Logf("keel-backends-exercised: %s", strings.Join(names, " "))
	t.Logf("keel-backends-available: %s", strings.Join(Available(), " "))
	if len(backends) < 3 {
		t.Logf("note: %d of 3 backends exercised on this host; "+
			"vector backends need GOEXPERIMENT=simd, GOARCH=amd64, and the CPU features",
			len(backends))
	}
}

// ---------------------------------------------------------------- edge pool

// edgeValues is the adversarial float32 pool: every class the shim must not
// mangle. DESIGN.md §4/P0 calls for NaN, ±Inf, denormals and -0 explicitly;
// the rest are here because they sit on rounding boundaries where a fused and
// an unfused multiply-add disagree.
var edgeValues = []float32{
	0,
	float32(math.Copysign(0, -1)), // -0
	1, -1, 0.5, -0.5, 2, -2,
	float32(math.Inf(1)),
	float32(math.Inf(-1)),
	float32(math.NaN()),
	math.Float32frombits(1),          // smallest positive denormal
	math.Float32frombits(0x007fffff), // largest denormal
	math.Float32frombits(0x00800000), // smallest positive normal
	math.Float32frombits(0x807fffff), // largest negative denormal
	math.MaxFloat32,
	-math.MaxFloat32,
	math.SmallestNonzeroFloat32,
	1 + math.Float32frombits(0x33800000), // 1 + 2^-24: rounds away in float32
	16777216,                             // 2^24, first integer with a gap
	16777217,                             // not representable
	1e-30, 1e30, -3.7, 12345.678,
}

// blockPool returns the blocks every op is tested on: blocks tiled from the
// edge pool so that adjacent lanes hold different value classes, blocks of a
// single repeated edge value, and pseudorandom blocks with a fixed seed.
func blockPool() []Block {
	var out []Block

	// Sliding windows over the edge pool: guarantees every edge value appears
	// in every lane position across the set, and that neighbouring lanes hold
	// unrelated classes (catches lane-crossing bugs in the AVX2 halves).
	for off := range edgeValues {
		var b Block
		for i := 0; i < Lanes; i++ {
			b[i] = edgeValues[(off+i)%len(edgeValues)]
		}
		out = append(out, b)
	}

	// Uniform blocks: isolates "this value is mishandled" from "this lane is".
	for _, v := range edgeValues {
		out = append(out, ScalarBroadcast(v))
	}

	// Fixed seed: these tests must be reproducible, so a failure Scott sees in
	// a gate log is the same failure on re-run.
	rng := rand.New(rand.NewSource(20260810))
	for n := 0; n < 64; n++ {
		var b Block
		for i := range b {
			switch n % 4 {
			case 0:
				b[i] = float32(rng.NormFloat64())
			case 1:
				b[i] = float32(rng.NormFloat64()) * 1e18 // overflow-prone
			case 2:
				b[i] = float32(rng.NormFloat64()) * 1e-18 // denormal-prone
			default:
				b[i] = edgeValues[rng.Intn(len(edgeValues))]
			}
		}
		out = append(out, b)
	}
	return out
}

// ------------------------------------------------------------- comparators

// sameF32 reports whether two float32 results agree.
//
// Bitwise equality, with one carve-out: two NaNs compare equal regardless of
// payload. The payload a fused multiply-add propagates when an input is NaN
// is hardware-defined (x86 quiets and forwards a source operand; Go's
// math.FMA makes its own choice), and BLAS semantics are about NaN
// *propagation*, not about which NaN. So NaN-ness is part of keel's spec and
// the payload explicitly is not.
//
// Everything else is bit-exact on purpose: -0 does not equal +0 here, because
// a shim that loses the sign of zero has a real bug (it changes the sign of
// results downstream through division and atan2-style code paths, and it
// would mean the sign-bit masking in Abs is wrong).
func sameF32(a, b float32) bool {
	if math.IsNaN(float64(a)) && math.IsNaN(float64(b)) {
		return true
	}
	return math.Float32bits(a) == math.Float32bits(b)
}

func sameBlock(a, b Block) bool {
	for i := range a {
		if !sameF32(a[i], b[i]) {
			return false
		}
	}
	return true
}

func fmtBlock(b Block) string {
	var sb strings.Builder
	for i, v := range b {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmtF32(v))
	}
	return sb.String()
}

// fmtF32 prints the value and its bit pattern. The bits matter: +0 versus -0,
// and one NaN payload versus another, are invisible in decimal, and those are
// exactly the cases these tests exist to pin down.
func fmtF32(v float32) string {
	return fmt.Sprintf("%g[%#08x]", v, math.Float32bits(v))
}

// ------------------------------------------------------- differential tests

// TestDiffUnary holds every backend's single-operand ops equal to the spec.
func TestDiffUnary(t *testing.T) {
	pool := blockPool()
	forEachVectorBackend(t, func(t *testing.T, b backend) {
		s := spec()
		for _, x := range pool {
			if got, want := b.Abs(x), s.Abs(x); !sameBlock(got, want) {
				t.Errorf("Abs mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock(x), fmtBlock(got), fmtBlock(want))
			}
			if got, want := b.HSum(x), s.HSum(x); !sameF32(got, want) {
				t.Errorf("HSum mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock(x), fmtF32(got), fmtF32(want))
			}
		}
	})
}

// TestDiffBinary holds every backend's two-operand ops equal to the spec.
func TestDiffBinary(t *testing.T) {
	pool := blockPool()
	ops := []struct {
		name string
		get  func(backend) func(x, y Block) Block
	}{
		{"Add", func(b backend) func(x, y Block) Block { return b.Add }},
		{"Sub", func(b backend) func(x, y Block) Block { return b.Sub }},
		{"Mul", func(b backend) func(x, y Block) Block { return b.Mul }},
		{"Max", func(b backend) func(x, y Block) Block { return b.Max }},
		{"Min", func(b backend) func(x, y Block) Block { return b.Min }},
	}
	forEachVectorBackend(t, func(t *testing.T, b backend) {
		for _, op := range ops {
			t.Run(op.name, func(t *testing.T) {
				gotOp, wantOp := op.get(b), op.get(spec())
				// Every ordered pair from the pool: Max/Min and Sub are not
				// commutative in their NaN and signed-zero behaviour, so
				// (x,y) and (y,x) are genuinely different tests.
				for i, x := range pool {
					for _, y := range pool[i:] {
						for _, p := range [2][2]Block{{x, y}, {y, x}} {
							got, want := gotOp(p[0], p[1]), wantOp(p[0], p[1])
							if !sameBlock(got, want) {
								t.Errorf("%s mismatch\n x:    %s\n y:    %s\n got:  %s\n spec: %s",
									op.name, fmtBlock(p[0]), fmtBlock(p[1]),
									fmtBlock(got), fmtBlock(want))
								return
							}
						}
					}
				}
			})
		}
	})
}

// TestDiffMulAdd holds every backend's fused multiply-add equal to the spec.
// Separate from the binary ops because it is the op the whole project rests
// on, and because it needs three operands.
func TestDiffMulAdd(t *testing.T) {
	pool := blockPool()
	forEachVectorBackend(t, func(t *testing.T, b backend) {
		s := spec()
		for i, x := range pool {
			for j, y := range pool {
				if (i+j)%7 != 0 { // full cube is O(n^3) blocks; stride it
					continue
				}
				for k, z := range pool {
					if (i+j+k)%11 != 0 {
						continue
					}
					got, want := b.MulAdd(x, y, z), s.MulAdd(x, y, z)
					if !sameBlock(got, want) {
						t.Errorf("MulAdd mismatch\n x:    %s\n y:    %s\n z:    %s\n got:  %s\n spec: %s",
							fmtBlock(x), fmtBlock(y), fmtBlock(z),
							fmtBlock(got), fmtBlock(want))
						return
					}
				}
			}
		}
	})
}

// TestDiffBroadcastZero covers the constant-producing ops.
func TestDiffBroadcastZero(t *testing.T) {
	forEachVectorBackend(t, func(t *testing.T, b backend) {
		s := spec()
		if got, want := b.Zero(), s.Zero(); !sameBlock(got, want) {
			t.Errorf("Zero mismatch\n got:  %s\n spec: %s", fmtBlock(got), fmtBlock(want))
		}
		for _, v := range edgeValues {
			if got, want := b.Broadcast(v), s.Broadcast(v); !sameBlock(got, want) {
				t.Errorf("Broadcast(%s) mismatch\n got:  %s\n spec: %s",
					fmtF32(v), fmtBlock(got), fmtBlock(want))
			}
		}
	})
}

// TestDiffLoadStore covers the memory ops at every partial length and at a
// non-zero offset inside a larger buffer.
//
// The offset sweep is the shim-level analogue of the stride≠1 requirement in
// DESIGN.md §4/P0: at this layer an op sees a slice, so what varies is where
// that slice starts and how long it is. Element strides themselves are a
// routine-level concern and are tested there (P1).
func TestDiffLoadStore(t *testing.T) {
	pool := blockPool()
	forEachVectorBackend(t, func(t *testing.T, b backend) {
		s := spec()

		// Full-width Load/Store, from every offset in a padded buffer.
		for _, off := range []int{0, 1, 3, 7, 8, 15, 16, 17} {
			buf := make([]float32, off+Lanes+8)
			for i := range buf {
				buf[i] = edgeValues[i%len(edgeValues)]
			}
			src := buf[off : off+Lanes]
			if got, want := b.Load(src), s.Load(src); !sameBlock(got, want) {
				t.Errorf("Load at offset %d mismatch\n got:  %s\n spec: %s",
					off, fmtBlock(got), fmtBlock(want))
			}
			for _, x := range pool[:8] {
				gotBuf := append([]float32(nil), buf...)
				wantBuf := append([]float32(nil), buf...)
				b.Store(gotBuf[off:off+Lanes], x)
				s.Store(wantBuf[off:off+Lanes], x)
				if !sameSlice(gotBuf, wantBuf) {
					t.Errorf("Store at offset %d mismatch\n got:  %v\n spec: %v",
						off, gotBuf, wantBuf)
				}
			}
		}

		// Partial Load/Store at every length from 0 through Lanes+1, which
		// covers the empty case and both sides of the width boundary (and,
		// for the AVX2 backend, the seam between its two 8-lane halves).
		for n := 0; n <= Lanes+1; n++ {
			src := make([]float32, n)
			for i := range src {
				src[i] = edgeValues[i%len(edgeValues)]
			}
			if got, want := b.LoadPart(src), s.LoadPart(src); !sameBlock(got, want) {
				t.Errorf("LoadPart(len=%d) mismatch\n got:  %s\n spec: %s",
					n, fmtBlock(got), fmtBlock(want))
			}
			for _, x := range pool[:8] {
				gotBuf := make([]float32, n)
				wantBuf := make([]float32, n)
				b.StorePart(gotBuf, x)
				s.StorePart(wantBuf, x)
				if !sameSlice(gotBuf, wantBuf) {
					t.Errorf("StorePart(len=%d) mismatch\n got:  %v\n spec: %v",
						n, gotBuf, wantBuf)
				}
			}
		}
	})
}

func sameSlice(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameF32(a[i], b[i]) {
			return false
		}
	}
	return true
}

// forEachVectorBackend runs fn for every registered backend except the spec
// itself. If no vector backend is registered it reports that as a skip, since
// the *gate* — not this test — is what must fail when a backend is missing.
func forEachVectorBackend(t *testing.T, fn func(*testing.T, backend)) {
	t.Helper()
	if len(backends) < 2 {
		t.Skipf("no vector backend registered on this host (available: %v); "+
			"gate-p0.sh enforces full backend coverage", Available())
	}
	for _, b := range backends[1:] {
		t.Run(b.name, func(t *testing.T) { fn(t, b) })
	}
}
