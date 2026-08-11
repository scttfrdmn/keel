// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/l1"
)

// Gate benchmarks. Only what a gate criterion actually measures lives here; the
// full benchmark suite with percent-of-peak arrives with P5.
//
// BenchmarkGateSdot is DESIGN.md §4/P1's ">=4x scalar for Sdot at n=4096". It
// runs one sub-benchmark per backend the machine can execute, named after the
// backend, so scripts/gate-p1.sh can read both ns/op values from a single run
// on a single machine and divide. Within-machine is the only comparison that
// means anything: an absolute ns/op has no denominator without a CPU model and
// a theoretical peak beside it (CLAUDE.md), and a ratio measured across two
// hosts would be measuring the hosts.
//
// n=4096 is 16 KiB per vector, 32 KiB for the pair — resident in L1d on every
// host in docs/hosts.md, which is what makes this a kernel measurement rather
// than a memory-bandwidth measurement.
func BenchmarkGateSdot(b *testing.B) {
	const n = 4096
	r := rand.New(rand.NewSource(0x6b65656c))
	x := make([]float32, n)
	y := make([]float32, n)
	for i := range x {
		x[i] = r.Float32()*2 - 1
		y[i] = r.Float32()*2 - 1
	}

	saved := activeL1
	b.Cleanup(func() { activeL1 = saved })
	for _, backend := range l1.Backends() {
		activeL1 = backend
		b.Run(backend.Name, func(b *testing.B) {
			b.SetBytes(2 * 4 * n)
			var acc float32
			for i := 0; i < b.N; i++ {
				acc = Sdot(n, x, 1, y, 1)
			}
			// Keeping the result alive is not a formality: without it a future
			// version of the compiler is free to notice Sdot is pure and hoist
			// the whole call out of the loop, and the gate would then certify a
			// speedup over nothing at all.
			benchSink = acc
		})
	}
}

var benchSink float32
