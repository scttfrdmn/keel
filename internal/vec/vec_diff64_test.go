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

// The float64 differential harness (#191): vec_diff_test.go's discipline for
// the float64 surface. Every registered backend is held BIT-EXACT to the scalar
// float64 spec on identical inputs, with the same single carve-out (NaN payload,
// see sameF64). Separate types rather than a generic harness: the float32 suite
// is load-bearing for gate-p0.sh and stays byte-for-byte unchanged.

// backend64 is one implementation of the float64 Block64-level op set. Every
// field is mandatory; a nil op is a registration bug, not a skip.
type backend64 struct {
	name      string
	Load      func([]float64) Block64
	LoadPart  func([]float64) Block64
	Store     func([]float64, Block64)
	StorePart func([]float64, Block64)
	Broadcast func(float64) Block64
	Zero      func() Block64
	Add       func(x, y Block64) Block64
	Sub       func(x, y Block64) Block64
	Mul       func(x, y Block64) Block64
	Div       func(x, y Block64) Block64
	MulAdd    func(x, y, z Block64) Block64
	Max       func(x, y Block64) Block64
	Min       func(x, y Block64) Block64
	Abs       func(Block64) Block64
	HSum      func(Block64) float64
}

// backends64 is the float64 registry. The scalar spec is always index 0; the
// amd64 build appends AVX-512 in vec_amd64_f64_test.go.
var backends64 = []backend64{{
	name:      BackendScalar,
	Load:      ScalarDLoad,
	LoadPart:  ScalarDLoadPart,
	Store:     ScalarDStore,
	StorePart: ScalarDStorePart,
	Broadcast: ScalarDBroadcast,
	Zero:      ScalarDZero,
	Add:       ScalarDAdd,
	Sub:       ScalarDSub,
	Mul:       ScalarDMul,
	Div:       ScalarDDiv,
	MulAdd:    ScalarDMulAdd,
	Max:       ScalarDMax,
	Min:       ScalarDMin,
	Abs:       ScalarDAbs,
	HSum:      ScalarDHSum,
}}

func spec64() backend64 { return backends64[0] }

// TestBackendCoverage64 reports which float64 backends ran, as
// TestBackendCoverage does for float32. The marker is deliberately
// `keel-backends64-exercised`, not a suffix of the float32 one: gate-p0.sh
// greps `keel-backends-exercised:` and takes the last match, so a float64 line
// that also matched would silently replace the float32 coverage it judges.
func TestBackendCoverage64(t *testing.T) {
	names := make([]string, 0, len(backends64))
	for _, b := range backends64 {
		names = append(names, b.name)
	}
	t.Logf("keel-backends64-exercised: %s", strings.Join(names, " "))
	if len(backends64) < 2 {
		t.Logf("note: only the scalar float64 spec exercised on this host; the AVX-512 " +
			"float64 backend needs GOEXPERIMENT=simd, GOARCH=amd64 and AVX-512")
	}
}

// ---------------------------------------------------------------- edge pool

// edgeValues64 is the adversarial float64 pool: the classes edgeValues covers,
// at float64's own boundaries.
var edgeValues64 = []float64{
	0,
	math.Copysign(0, -1), // -0
	1, -1, 0.5, -0.5, 2, -2,
	math.Inf(1),
	math.Inf(-1),
	math.NaN(),
	math.Float64frombits(1),                  // smallest positive denormal
	math.Float64frombits(0x000fffffffffffff), // largest denormal
	math.Float64frombits(0x0010000000000000), // smallest positive normal
	math.Float64frombits(0x800fffffffffffff), // largest negative denormal
	math.MaxFloat64,
	-math.MaxFloat64,
	math.SmallestNonzeroFloat64,
	1 + math.Float64frombits(0x3ca0000000000000), // 1 + 2^-53: rounds away in float64
	9007199254740992,                             // 2^53, first integer with a gap
	9007199254740993,                             // not representable
	1e-300, 1e300, -3.7, 12345.678,
	float64(math.MaxFloat32), // float32 boundaries are ordinary values here
	float64(math.SmallestNonzeroFloat32),
}

// blockPool64 mirrors blockPool: sliding windows over the edge pool, uniform
// blocks, and a fixed-seed pseudorandom set.
func blockPool64() []Block64 {
	var out []Block64
	for off := range edgeValues64 {
		var b Block64
		for i := 0; i < Lanes64; i++ {
			b[i] = edgeValues64[(off+i)%len(edgeValues64)]
		}
		out = append(out, b)
	}
	for _, v := range edgeValues64 {
		out = append(out, ScalarDBroadcast(v))
	}
	rng := rand.New(rand.NewSource(20261026))
	for n := 0; n < 64; n++ {
		var b Block64
		for i := range b {
			switch n % 4 {
			case 0:
				b[i] = rng.NormFloat64()
			case 1:
				b[i] = rng.NormFloat64() * 1e150 // overflow-prone
			case 2:
				b[i] = rng.NormFloat64() * 1e-150 // denormal-prone
			default:
				b[i] = edgeValues64[rng.Intn(len(edgeValues64))]
			}
		}
		out = append(out, b)
	}
	return out
}

// ------------------------------------------------------------- comparators

// sameF64 is sameF32 for float64: bitwise, except that two NaNs agree
// regardless of payload (see sameF32 for why the payload is not spec).
func sameF64(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Float64bits(a) == math.Float64bits(b)
}

func sameBlock64(a, b Block64) bool {
	for i := range a {
		if !sameF64(a[i], b[i]) {
			return false
		}
	}
	return true
}

func sameSlice64(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameF64(a[i], b[i]) {
			return false
		}
	}
	return true
}

func fmtBlock64(b Block64) string {
	var sb strings.Builder
	for i, v := range b {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmtF64(v))
	}
	return sb.String()
}

func fmtF64(v float64) string {
	return fmt.Sprintf("%g[%#016x]", v, math.Float64bits(v))
}

// ------------------------------------------------------- differential tests

func TestDiff64Unary(t *testing.T) {
	pool := blockPool64()
	forEachVectorBackend64(t, func(t *testing.T, b backend64) {
		s := spec64()
		for _, x := range pool {
			if got, want := b.Abs(x), s.Abs(x); !sameBlock64(got, want) {
				t.Errorf("Abs mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock64(x), fmtBlock64(got), fmtBlock64(want))
			}
			if got, want := b.HSum(x), s.HSum(x); !sameF64(got, want) {
				t.Errorf("HSum mismatch\n in:   %s\n got:  %s\n spec: %s",
					fmtBlock64(x), fmtF64(got), fmtF64(want))
			}
		}
	})
}

func TestDiff64Binary(t *testing.T) {
	pool := blockPool64()
	ops := []struct {
		name string
		get  func(backend64) func(x, y Block64) Block64
	}{
		{"Add", func(b backend64) func(x, y Block64) Block64 { return b.Add }},
		{"Sub", func(b backend64) func(x, y Block64) Block64 { return b.Sub }},
		{"Mul", func(b backend64) func(x, y Block64) Block64 { return b.Mul }},
		{"Div", func(b backend64) func(x, y Block64) Block64 { return b.Div }},
		{"Max", func(b backend64) func(x, y Block64) Block64 { return b.Max }},
		{"Min", func(b backend64) func(x, y Block64) Block64 { return b.Min }},
	}
	forEachVectorBackend64(t, func(t *testing.T, b backend64) {
		for _, op := range ops {
			t.Run(op.name, func(t *testing.T) {
				gotOp, wantOp := op.get(b), op.get(spec64())
				// Every ordered pair: Max/Min's NaN and signed-zero rule is
				// operand-order dependent.
				for i, x := range pool {
					for _, y := range pool[i:] {
						for _, p := range [2][2]Block64{{x, y}, {y, x}} {
							got, want := gotOp(p[0], p[1]), wantOp(p[0], p[1])
							if !sameBlock64(got, want) {
								t.Errorf("%s mismatch\n x:    %s\n y:    %s\n got:  %s\n spec: %s",
									op.name, fmtBlock64(p[0]), fmtBlock64(p[1]),
									fmtBlock64(got), fmtBlock64(want))
								return
							}
						}
					}
				}
			})
		}
	})
}

// TestDiff64MulAdd reaches the memory-operand encoding, not the register form
// a kernel uses — the TestDiffMulAdd caveat applies unchanged.
func TestDiff64MulAdd(t *testing.T) {
	pool := blockPool64()
	forEachVectorBackend64(t, func(t *testing.T, b backend64) {
		s := spec64()
		for i, x := range pool {
			for j, y := range pool {
				if (i+j)%7 != 0 {
					continue
				}
				for k, z := range pool {
					if (i+j+k)%11 != 0 {
						continue
					}
					got, want := b.MulAdd(x, y, z), s.MulAdd(x, y, z)
					if !sameBlock64(got, want) {
						t.Errorf("MulAdd mismatch\n x:    %s\n y:    %s\n z:    %s\n got:  %s\n spec: %s",
							fmtBlock64(x), fmtBlock64(y), fmtBlock64(z),
							fmtBlock64(got), fmtBlock64(want))
						return
					}
				}
			}
		}
	})
}

func TestDiff64BroadcastZero(t *testing.T) {
	forEachVectorBackend64(t, func(t *testing.T, b backend64) {
		s := spec64()
		if got, want := b.Zero(), s.Zero(); !sameBlock64(got, want) {
			t.Errorf("Zero mismatch\n got:  %s\n spec: %s", fmtBlock64(got), fmtBlock64(want))
		}
		for _, v := range edgeValues64 {
			if got, want := b.Broadcast(v), s.Broadcast(v); !sameBlock64(got, want) {
				t.Errorf("Broadcast(%s) mismatch\n got:  %s\n spec: %s",
					fmtF64(v), fmtBlock64(got), fmtBlock64(want))
			}
		}
	})
}

// TestDiff64LoadStore covers full-width ops at every offset and partial ops at
// every length 0..Lanes64+1.
func TestDiff64LoadStore(t *testing.T) {
	pool := blockPool64()
	forEachVectorBackend64(t, func(t *testing.T, b backend64) {
		s := spec64()
		for _, off := range []int{0, 1, 3, 4, 7, 8, 9} {
			buf := make([]float64, off+Lanes64+4)
			for i := range buf {
				buf[i] = edgeValues64[i%len(edgeValues64)]
			}
			src := buf[off : off+Lanes64]
			if got, want := b.Load(src), s.Load(src); !sameBlock64(got, want) {
				t.Errorf("Load at offset %d mismatch\n got:  %s\n spec: %s",
					off, fmtBlock64(got), fmtBlock64(want))
			}
			for _, x := range pool[:8] {
				gotBuf := append([]float64(nil), buf...)
				wantBuf := append([]float64(nil), buf...)
				b.Store(gotBuf[off:off+Lanes64], x)
				s.Store(wantBuf[off:off+Lanes64], x)
				if !sameSlice64(gotBuf, wantBuf) {
					t.Errorf("Store at offset %d mismatch\n got:  %v\n spec: %v",
						off, gotBuf, wantBuf)
				}
			}
		}
		for n := 0; n <= Lanes64+1; n++ {
			src := make([]float64, n)
			for i := range src {
				src[i] = edgeValues64[i%len(edgeValues64)]
			}
			if got, want := b.LoadPart(src), s.LoadPart(src); !sameBlock64(got, want) {
				t.Errorf("LoadPart(len=%d) mismatch\n got:  %s\n spec: %s",
					n, fmtBlock64(got), fmtBlock64(want))
			}
			for _, x := range pool[:8] {
				gotBuf := make([]float64, n)
				wantBuf := make([]float64, n)
				b.StorePart(gotBuf, x)
				s.StorePart(wantBuf, x)
				if !sameSlice64(gotBuf, wantBuf) {
					t.Errorf("StorePart(len=%d) mismatch\n got:  %v\n spec: %v",
						n, gotBuf, wantBuf)
				}
			}
		}
	})
}

// forEachVectorBackend64 runs fn for every registered float64 backend except
// the spec, skipping (not passing) when none is registered.
func forEachVectorBackend64(t *testing.T, fn func(*testing.T, backend64)) {
	t.Helper()
	if len(backends64) < 2 {
		t.Skipf("no float64 vector backend registered on this host (available: %v)", Available())
	}
	for _, b := range backends64[1:] {
		t.Run(b.name, func(t *testing.T) { fn(t, b) })
	}
}

// TestSpec64MulAddIsFused is TestSpecMulAddIsFused at float64's precision, and
// it runs everywhere, so the float64 spec's one-rounding claim is checked on
// hosts with no float64 vector backend. a = 1 + 2^-27 gives a*a = 1 + 2^-26 +
// 2^-54 exactly: a tie at float64's 53 bits, which an unfused multiply rounds to
// 1 + 2^-26 (ties-to-even), so adding c = -(1 + 2^-26) yields 0 unfused and
// 2^-54 fused.
func TestSpec64MulAddIsFused(t *testing.T) {
	a := 1 + math.Ldexp(1, -27)
	c := -(1 + math.Ldexp(1, -26))
	wantFused := math.Ldexp(1, -54)
	// float64(a*a) forbids fusing the product into the add (T16); the guard keeps
	// the witness discriminating.
	if unfused := float64(a*a) + c; unfused == wantFused {
		t.Fatalf("witness is no longer discriminating: unfused = %v", unfused)
	}
	got := ScalarDMulAdd(ScalarDBroadcast(a), ScalarDBroadcast(a), ScalarDBroadcast(c))
	for i, v := range got {
		if v != wantFused {
			t.Errorf("lane %d: ScalarDMulAdd = %s, want fused %s", i, fmtF64(v), fmtF64(wantFused))
		}
	}
}
