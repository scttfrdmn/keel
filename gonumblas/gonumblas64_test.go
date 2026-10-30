// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel"
	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/blas/blas64"
	"gonum.org/v1/gonum/blas/gonum"
)

// The blas.Float64 adapter's own tests (#192): the drop-in claim through blas64.Use,
// the panic set, the five translations, and routing. Numerics are the D routines'
// own suites (package keel, and level{1,2,3}_64_test.go here).

// TestBlas64Use is TestBlas32Use at float64: register keel through gonum's own entry
// point and call gonum's own high-level wrapper.
func TestBlas64Use(t *testing.T) {
	blas64.Use(Implementation{})
	defer blas64.Use(gonum.Implementation{})
	a := blas64.General{Rows: 2, Cols: 2, Stride: 2, Data: []float64{1, 2, 3, 4}}
	b := blas64.General{Rows: 2, Cols: 2, Stride: 2, Data: []float64{5, 6, 7, 8}}
	c := blas64.General{Rows: 2, Cols: 2, Stride: 2, Data: make([]float64, 4)}
	blas64.Gemm(blas.NoTrans, blas.NoTrans, 1, a, b, 0, c)
	if want := []float64{19, 22, 43, 50}; fmt.Sprint(c.Data) != fmt.Sprint(want) {
		t.Fatalf("blas64.Gemm through keel = %v, want %v", c.Data, want)
	}
	if got := blas64.Implementation(); got != (Implementation{}) {
		t.Fatalf("blas64.Implementation() = %T, want gonumblas.Implementation", got)
	}
}

// TestPanicsWhereGonumPanics64 is TestPanicsWhereGonumPanics at float64: the adapter
// panics on exactly the inputs gonum panics on.
func TestPanicsWhereGonumPanics64(t *testing.T) {
	x, a, c := make([]float64, 8), make([]float64, 16), make([]float64, 16)
	cases := []struct {
		name string
		call func(b blas.Float64)
	}{
		{"Ddot n<0", func(b blas.Float64) { b.Ddot(-1, x, 1, x, 1) }},
		{"Ddot incX=0", func(b blas.Float64) { b.Ddot(2, x, 0, x, 1) }},
		{"Ddot short x", func(b blas.Float64) { b.Ddot(9, x, 1, x, 1) }},
		{"Daxpy incY=0", func(b blas.Float64) { b.Daxpy(2, 1, x, 1, x, 0) }},
		{"Dscal incX=0", func(b blas.Float64) { b.Dscal(2, 1, x, 0) }},
		{"Dscal incX<0", func(b blas.Float64) { b.Dscal(2, 1, x, -1) }},
		{"Dnrm2 incX=0", func(b blas.Float64) { b.Dnrm2(2, x, 0) }},
		{"Dasum short x", func(b blas.Float64) { b.Dasum(9, x, 1) }},
		{"Idamax incX=0", func(b blas.Float64) { b.Idamax(2, x, 0) }},
		{"Idamax n=0", func(b blas.Float64) { b.Idamax(0, x, 1) }},
		{"Dgemv bad trans", func(b blas.Float64) { b.Dgemv('X', 2, 2, 1, a, 2, x, 1, 0, x, 1) }},
		{"Dgemv lda<n", func(b blas.Float64) { b.Dgemv(blas.NoTrans, 2, 3, 1, a, 2, x, 1, 0, x, 1) }},
		{"Dger m<0", func(b blas.Float64) { b.Dger(-1, 2, 1, x, 1, x, 1, a, 2) }},
		{"Dtrsv bad diag", func(b blas.Float64) { b.Dtrsv(blas.Upper, blas.NoTrans, 'X', 2, a, 2, x, 1) }},
		{"Dtbmv k<0", func(b blas.Float64) { b.Dtbmv(blas.Upper, blas.NoTrans, blas.NonUnit, 2, -1, a, 2, x, 1) }},
		{"Dspr short ap", func(b blas.Float64) { b.Dspr(blas.Upper, 20, 1, x, 1, a) }},
		{"Dgemm lda<k", func(b blas.Float64) { b.Dgemm(blas.NoTrans, blas.NoTrans, 2, 2, 3, 1, a, 2, a, 2, 0, c, 2) }},
		{"Dgemm short c", func(b blas.Float64) { b.Dgemm(blas.NoTrans, blas.NoTrans, 4, 4, 1, 1, a, 1, a, 4, 0, c[:8], 4) }},
		{"Dgemm k=0 is legal", func(b blas.Float64) { b.Dgemm(blas.NoTrans, blas.NoTrans, 2, 2, 0, 1, a, 1, a, 2, 0, c, 2) }},
		{"Dsyrk uplo All", func(b blas.Float64) { b.Dsyrk(blas.All, blas.NoTrans, 2, 2, 1, a, 2, 0, c, 2) }},
		{"Dsymm bad side", func(b blas.Float64) { b.Dsymm('X', blas.Upper, 2, 2, 1, a, 2, a, 2, 0, c, 2) }},
		{"Dtrsm bad diag", func(b blas.Float64) { b.Dtrsm(blas.Left, blas.Upper, blas.NoTrans, 'X', 2, 2, 1, a, 2, c, 2) }},
		{"Dtrsm ldb<n", func(b blas.Float64) { b.Dtrsm(blas.Left, blas.Upper, blas.NoTrans, blas.NonUnit, 2, 3, 1, a, 2, c, 2) }},
	}
	for _, tc := range cases {
		if pk, pg := panics(func() { tc.call(keelImpl) }), panics(func() { tc.call(gonumImpl) }); pk != pg {
			t.Errorf("%s: adapter panics=%v, gonum panics=%v", tc.name, pk, pg)
		}
	}
}

// TestTranslations64 pins the five translations at float64: each input keel itself
// rejects or answers differently gets gonum's result through the adapter.
func TestTranslations64(t *testing.T) {
	x := []float64{3, -4, 1}
	a := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	// Negative stride on the inc>0-only routines: reference BLAS's answers.
	xs := append([]float64(nil), x...)
	keelImpl.Dscal(3, 2, xs, -1)
	if fmt.Sprint(xs) != fmt.Sprint(x) {
		t.Errorf("Dscal incX<0 changed x: %v", xs)
	}
	if g, k := gonumImpl.Dasum(3, x, -1), keelImpl.Dasum(3, x, -1); g != k {
		t.Errorf("Dasum incX<0: adapter %v, gonum %v", k, g)
	}
	if g, k := gonumImpl.Dnrm2(3, x, -1), keelImpl.Dnrm2(3, x, -1); g != k {
		t.Errorf("Dnrm2 incX<0: adapter %v, gonum %v", k, g)
	}
	if g, k := gonumImpl.Idamax(3, x, -1), keelImpl.Idamax(3, x, -1); g != k {
		t.Errorf("Idamax incX<0: adapter %v, gonum %v", k, g)
	}
	// Empty reduction: y untouched, as reference BLAS (keel itself returns beta·y).
	for _, f := range []func(y []float64){
		func(y []float64) { keelImpl.Dgemv(blas.NoTrans, 3, 0, 1, a, 1, x, 1, 0.5, y, 1) },
		func(y []float64) { keelImpl.Dgbmv(blas.NoTrans, 3, 0, 0, 0, 1, a, 1, x, 1, 0.5, y, 1) },
	} {
		y := []float64{7, 8, 9}
		f(y)
		if fmt.Sprint(y) != "[7 8 9]" {
			t.Errorf("empty reduction changed y: %v", y)
		}
	}
	// ConjTrans is Trans for real data.
	yc, yt := make([]float64, 3), make([]float64, 3)
	keelImpl.Dgemv(blas.ConjTrans, 3, 3, 1, a, 3, x, 1, 0, yc, 1)
	keelImpl.Dgemv(blas.Trans, 3, 3, 1, a, 3, x, 1, 0, yt, 1)
	if fmt.Sprint(yc) != fmt.Sprint(yt) {
		t.Errorf("Dgemv ConjTrans %v != Trans %v", yc, yt)
	}
	// An unknown Drotm flag gets gonum's behaviour; keel itself panics on it.
	p := blas.DrotmParams{Flag: 7, H: [4]float64{1, 2, 3, 4}}
	xk, yk := []float64{1, 2}, []float64{3, 4}
	xg, yg := []float64{1, 2}, []float64{3, 4}
	pk, pg := panics(func() { keelImpl.Drotm(2, xk, 1, yk, 1, p) }), panics(func() { gonumImpl.Drotm(2, xg, 1, yg, 1, p) })
	if pk != pg || fmt.Sprint(xk, yk) != fmt.Sprint(xg, yg) {
		t.Errorf("Drotm unknown flag: adapter panics=%v %v %v, gonum panics=%v %v %v", pk, xk, yk, pg, xg, yg)
	}
	// Dsdot is gonum's: keel has no twin, so the embedding answers.
	xf, yf := []float32{1, 2}, []float32{3, 4}
	if k, g := keelImpl.Dsdot(2, xf, 1, yf, 1), gonumImpl.Dsdot(2, xf, 1, yf, 1); k != g {
		t.Errorf("Dsdot: adapter %v, gonum %v", k, g)
	}
}

// TestRoutesToKeel64 is TestRoutesToKeel at float64: for each probed routine, an input
// where keel and gonum differ BITWISE, on which the adapter must match keel exactly.
// A probe on which the two libraries agree is reported, never silently passed.
//
// Not probed, for the float32 test's reasons: Dscal and Idamax are exact or one
// rounding in both; Daxpy, Dger, Dcopy, Dswap and the rotations are elementwise and
// often bit-identical to gonum; their routing rests on the pass-throughs in
// gonumblas64.go.
func TestRoutesToKeel64(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	bits := func(v []float64) string { return fmt.Sprintf("%x", v) }
	const n = 4099
	x, y := randVec64(r, n), randVec64(r, n)
	for i := range x {
		x[i] *= math.Exp(12 * r.NormFloat64()) // magnitudes over many decades
	}
	const d = 33
	a := randVec64(r, d*d)
	for i := 0; i < d; i++ {
		a[i*d+i] = 4
	}
	b := randVec64(r, d*d)
	ta := randVec64(r, 130*130)
	for i := 0; i < 130; i++ {
		ta[i*130+i] = 4
	}
	tb := randVec64(r, 130*7)
	k, g := keelImpl, gonumImpl
	type probe struct {
		name                    string
		adapter, keelD, gonumDo func() []float64
	}
	on := func(src []float64, f func(c []float64)) func() []float64 {
		return func() []float64 { c := append([]float64(nil), src...); f(c); return c }
	}
	ps := []probe{
		{"Ddot", func() []float64 { return []float64{k.Ddot(n, x, 1, y, 1)} },
			func() []float64 { return []float64{keel.Ddot(n, x, 1, y, 1)} },
			func() []float64 { return []float64{g.Ddot(n, x, 1, y, 1)} }},
		{"Dasum", func() []float64 { return []float64{k.Dasum(n, x, 1)} },
			func() []float64 { return []float64{keel.Dasum(n, x, 1)} },
			func() []float64 { return []float64{g.Dasum(n, x, 1)} }},
		{"Dnrm2", func() []float64 { return []float64{k.Dnrm2(n, x, 1)} },
			func() []float64 { return []float64{keel.Dnrm2(n, x, 1)} },
			func() []float64 { return []float64{g.Dnrm2(n, x, 1)} }},
		{"Dgemv", on(b, func(c []float64) { k.Dgemv(blas.NoTrans, d, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) }),
			on(b, func(c []float64) { keel.Dgemv(keel.NoTrans, d, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) }),
			on(b, func(c []float64) { g.Dgemv(blas.NoTrans, d, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) })},
		{"Dsymv", on(b, func(c []float64) { k.Dsymv(blas.Upper, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) }),
			on(b, func(c []float64) { keel.Dsymv(keel.Upper, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) }),
			on(b, func(c []float64) { g.Dsymv(blas.Upper, d, 1.5, a, d, x, 1, 0.5, c[:d], 1) })},
		{"Dtrmv", on(x[:d], func(c []float64) { k.Dtrmv(blas.Upper, blas.NoTrans, blas.NonUnit, d, a, d, c, 1) }),
			on(x[:d], func(c []float64) { keel.Dtrmv(keel.Upper, keel.NoTrans, keel.NonUnit, d, a, d, c, 1) }),
			on(x[:d], func(c []float64) { g.Dtrmv(blas.Upper, blas.NoTrans, blas.NonUnit, d, a, d, c, 1) })},
		{"Dgemm", on(b, func(c []float64) { k.Dgemm(blas.NoTrans, blas.Trans, d, d, d, 1.5, a, d, b, d, 0.5, c, d) }),
			on(b, func(c []float64) { keel.Dgemm(keel.NoTrans, keel.Trans, d, d, d, 1.5, a, d, b, d, 0.5, c, d) }),
			on(b, func(c []float64) { g.Dgemm(blas.NoTrans, blas.Trans, d, d, d, 1.5, a, d, b, d, 0.5, c, d) })},
		{"Dsyrk", on(b, func(c []float64) { k.Dsyrk(blas.Upper, blas.NoTrans, d, d, 1.5, a, d, 0.5, c, d) }),
			on(b, func(c []float64) { keel.Dsyrk(keel.Upper, keel.NoTrans, d, d, 1.5, a, d, 0.5, c, d) }),
			on(b, func(c []float64) { g.Dsyrk(blas.Upper, blas.NoTrans, d, d, 1.5, a, d, 0.5, c, d) })},
		{"Dsymm", on(b, func(c []float64) { k.Dsymm(blas.Left, blas.Upper, d, d, 1.5, a, d, b, d, 0.5, c, d) }),
			on(b, func(c []float64) { keel.Dsymm(keel.Left, keel.Upper, d, d, 1.5, a, d, b, d, 0.5, c, d) }),
			on(b, func(c []float64) { g.Dsymm(blas.Left, blas.Upper, d, d, 1.5, a, d, b, d, 0.5, c, d) })},
		// A triangle past keel's 64-row diagonal block, as TestRoutesToKeel's Strsm.
		{"Dtrsm", on(tb, func(c []float64) {
			k.Dtrsm(blas.Left, blas.Lower, blas.NoTrans, blas.NonUnit, 130, 7, 1.5, ta, 130, c, 7)
		}),
			on(tb, func(c []float64) {
				keel.Dtrsm(keel.Left, keel.Lower, keel.NoTrans, keel.NonUnit, 130, 7, 1.5, ta, 130, c, 7)
			}),
			on(tb, func(c []float64) {
				g.Dtrsm(blas.Left, blas.Lower, blas.NoTrans, blas.NonUnit, 130, 7, 1.5, ta, 130, c, 7)
			})},
	}
	for _, p := range ps {
		ad, kd, gd := p.adapter(), p.keelD(), p.gonumDo()
		if bits(kd) == bits(gd) {
			t.Errorf("%s: keel and gonum are bit-identical on this probe, so it cannot tell which one ran; pick a probe where they differ", p.name)
			continue
		}
		if bits(ad) != bits(kd) {
			t.Errorf("%s: the adapter's result is not keel's bit for bit, so the call did not reach keel", p.name)
		}
	}
}
