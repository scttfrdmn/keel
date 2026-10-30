// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"math/rand"
	"testing"
)

// BenchmarkL2Par is the sweep l2.Grain was chosen by. Vary the thread count with
// -cpu, never by setting GOMAXPROCS around a cached run: GOMAXPROCS is not in the
// test cache key, and -cpu is what a benchmark's name records.
//
//	go test -run '^$' -bench L2Par -cpu 1,2,4,8 -count 5
//
// SetBytes counts A once, read (and for Sger/Ssyr/Ssyr2 written back), so MB/s is
// A's streaming rate. These routines touch each element of A once: above the
// last-level cache the rate is the memory system's, and adding workers then buys
// what bandwidth the cores could not already draw, not a multiple of it.
// workers/op is what the call actually used, so a row whose width the threshold
// capped says so in its own line.
func BenchmarkL2Par(b *testing.B) {
	type shape struct{ m, n int }
	var shapes []shape
	for n := 64; n <= 4096; n *= 2 {
		shapes = append(shapes, shape{n, n})
	}
	shapes = append(shapes, shape{16384, 256}, shape{256, 16384})

	r := rand.New(rand.NewSource(0x6b65656c))
	for _, s := range shapes {
		m, n := s.m, s.n
		a := randVals(r, m*n)
		x, y := randVals(r, max(m, n)), randVals(r, max(m, n))
		run := func(name string, elems int, fn func()) {
			b.Run(fmt.Sprintf("%s/m=%d/n=%d", name, m, n), func(b *testing.B) {
				b.SetBytes(int64(4 * elems))
				for i := 0; i < b.N; i++ {
					fn()
				}
				b.ReportMetric(float64(WorkersLastCall()), "workers/op")
			})
		}
		run("SgemvN", m*n, func() { Sgemv(NoTrans, m, n, 1, a, n, x, 1, 0, y, 1) })
		run("SgemvT", m*n, func() { Sgemv(Trans, m, n, 1, a, n, x, 1, 0, y, 1) })
		run("Sger", m*n, func() { Sger(m, n, 1e-9, x, 1, y, 1, a, n) })
		if m != n {
			continue
		}
		tri := n * (n + 1) / 2
		run("Ssyr", tri, func() { Ssyr(Upper, n, 1e-9, x, 1, a, n) })
		run("Ssyr2", tri, func() { Ssyr2(Upper, n, 1e-9, x, 1, y, 1, a, n) })
		run("StrmvN", tri, func() {
			copy(y, x)
			Strmv(Upper, NoTrans, NonUnit, n, a, n, y, 1)
		})
		run("StrmvT", tri, func() {
			copy(y, x)
			Strmv(Upper, Trans, NonUnit, n, a, n, y, 1)
		})
	}
}
