// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
	"github.com/scttfrdmn/keel/internal/pack"
)

// Retention: where the blocked Sgemm's time goes that the microkernel's does not
// (issue #26).
//
// # The question
//
// The blocked Sgemm reaches ~90% of its own dispatched microkernel's throughput on
// both Zen hosts and ~77% on janus (Skylake-X). The roofline amendment (#19/#23)
// excuses instruction count *inside* the K-loop on an issue-bound host; it says
// nothing about packing, memory traffic or loop-nest overhead, so the ~13-point
// gap is a plain performance question. #26 lists three candidates: packing cost as
// a fraction of the whole, the B-panel pack in particular (the transposing
// direction, scalar on this toolchain), and whether the gap tracks the host's
// *class* rather than the host.
//
// # The instrument
//
// Four sub-benchmarks per (shape, size), which decompose one blocked Sgemm:
//
//	full          Gemm — the whole thing, packing included
//	nest-no-pack  every loop of gemm except the two pack calls
//	pack-a        only the APanels calls the nest makes, over the same blocks
//	pack-b        only the BPanels calls the nest makes, over the same blocks
//
// so that
//
//	residual = full - nest-no-pack - pack-a - pack-b
//
// is what the decomposition does not explain, and it is reported rather than
// absorbed into whichever part is being argued about. All four walk the identical
// block structure, because they all get it from plan() — see its doc.
//
// nest-no-pack packs one set of panels *outside* the timer and then reuses those
// panel buffers for every block, which is a timing equivalent rather than a
// correct GEMM: the values in ap/bp are wrong for every block but the first, and
// nothing reads the result. What it preserves is exactly what a cost measurement
// needs — the same number of microkernel calls at the same kk, the same panel
// buffers at the same addresses (the real nest reuses these two allocations too),
// the same C traffic over the caller's matrix, the same beta pass, and the same
// fringe handling.
//
// What it drops — and therefore what the residual contains — is three things: the
// pack calls' own cost, which is the point of dropping them; any cache interference
// *between* packing and the kernel that runs after it, which belongs to no single
// part; and gemm's three per-call allocations, since ap/bp/tile are made here once
// outside the timer. The third is small but real — bp is nc·kc·4 bytes, 3.1 MB at
// n=2048, and Go zeroes it — so it is named here rather than left for somebody to
// rediscover inside the residual.
//
// # The kernel denominator, in the same invocation
//
// The fifth sub-benchmark, kernel/kc=<kc>, is the microkernel alone on L1-resident
// panels at the depth this nest actually calls it with. #26's table divided a
// kernel rate from one invocation by a blocked rate from another and said so as a
// caveat; here both are in one binary on one host in one run, so the ratio is a
// ratio. It is still a ratio of medians and no interval is claimed for it.
//
// GFLOP/s is reported for full and nest-no-pack (useful gemm flops, 2mnk) and for
// kernel (its own 2·MR·NR·kc per call), which is what makes those three
// comparable. The two pack sub-benchmarks report no flop rate at all: they do no
// arithmetic, and inventing a numerator for them is the thing rule 7 forbids.
// Their contribution is read off ns/op, in the same units as full.
//
// # Shape is a dimension, not a dispatch
//
// The shipped shapes are iterated rather than asking dispatch for this host's
// choice, for two reasons. internal/block cannot import the root package (that is
// the cycle), so "the dispatched shape" here would be a second implementation of
// selectKern, free to drift from the one that ships. And #26's third candidate —
// whether retention tracks the class — is answered directly by measuring both
// shapes' retention on every host, with no KEEL_KERN_CLASS pinning needed: if the
// thin shape retains less everywhere, the nest is spending front-end slots and the
// two findings are one finding.
//
// Scalar reference tiles are skipped. At 2048³ a scalar microkernel would take
// minutes per iteration and it is not a shape any host dispatches.
func BenchmarkNest(b *testing.B) {
	for _, kn := range benchKernels(b) {
		for _, n := range []int{1024, 2048} {
			m, k := n, n
			flops := 2 * float64(m) * float64(n) * float64(k)
			kc, mc, nc := plan(kn, m, n, k)
			b.Run(fmt.Sprintf("%s/n=%d", kn.ID(), n), func(b *testing.B) {
				// One marker per (shape, size) so a reader of the raw benchmark
				// output can see the block structure the parts were measured over
				// without recomputing plan() by hand. The counts are counted off
				// the generator the parts run, not derived a second time by
				// arithmetic that could disagree with it.
				blocks := nestBlocks(kn, m, n, k)
				bpacks := 0
				for _, blk := range blocks {
					if blk.newPanel {
						bpacks++
					}
				}
				fmt.Printf("keel-nest-plan: name=%s kc=%d mc=%d nc=%d mr=%d nr=%d "+
					"blocks=%d apacks=%d bpacks=%d\n",
					kn.ID()+"/n="+fmt.Sprint(n), kc, mc, nc, kn.MR, kn.NR,
					len(blocks), len(blocks), bpacks)

				am, bm, c := benchMat(m, k), benchMat(k, n), benchMat(m, n)
				ap := make([]float32, pack.ALen(kn.MR, mc, kc))
				bp := make([]float32, pack.BLen(kn.NR, nc, kc))
				tile := make([]float32, kn.MR*kn.NR)

				b.Run("full", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						Gemm(kn, false, false, m, n, k, 1, am, k, bm, n, 0, c, n)
					}
					b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
				})
				b.Run("nest-no-pack", func(b *testing.B) {
					// The panels the real nest would have built for the first
					// block, built once. See the package comment above for why
					// reusing them everywhere is a timing equivalent.
					pack.APanels(ap, kn.MR, 1, am, k, false, 0, min(mc, m), 0, kc)
					pack.BPanels(bp, kn.NR, bm, n, false, 0, kc, 0, min(nc, n))
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						nestNoPack(kn, m, n, k, 0, ap, bp, c, n, tile)
					}
					b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
				})
				b.Run("pack-a", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						packAOnly(kn, m, n, k, am, k, ap, false)
					}
				})
				b.Run("pack-b", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						packBOnly(kn, m, n, k, bm, n, bp, false)
					}
				})
				b.Run(fmt.Sprintf("kernel/kc=%d", kc), func(b *testing.B) {
					ka, kb, kc2, ldc := benchPanels(kn, kc)
					perCall := 2.0 * float64(kn.MR) * float64(kn.NR) * float64(kc)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						kn.Fn(kc, ka, kb, kc2, ldc)
					}
					b.ReportMetric(perCall*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
					benchSink = kc2[0]
				})
			})
		}
	}
}

// BenchmarkBlocking sweeps KC/MC/NC at the gate's shape, which #26 asks for: the
// three parameters were chosen once from DESIGN.md §4/P3's Zen4/Ice-Lake starting
// point and have never been swept per host, and janus has the smallest cache per
// core of the three gate machines.
//
// The grid is coarse on purpose. This is a measurement of whether the parameters
// matter and in which direction, not a tuner: DESIGN.md §4 puts auto-tuning in P5
// but #26's deliverable is "a measurement and a note", and a per-host default may
// not be worth carrying at all if the surface turns out to be flat.
//
// NC tops out at 2048 rather than the shipped 4096 because at n=2048 they are the
// same measurement — plan() clamps nc to whole tiles covering n, so every value at
// or above n collapses to one point. A sweep that listed 4096 would be reporting
// the same number twice under two names.
//
// The parameters are package vars, so each point sets them, runs, and restores.
// Sub-benchmarks run sequentially, so this is safe; it would not be under -cpu
// parallelism, and nothing here uses b.RunParallel.
func BenchmarkBlocking(b *testing.B) {
	const n = 2048
	flops := 2 * float64(n) * float64(n) * float64(n)
	defer func(kc, mc, nc int) { KC, MC, NC = kc, mc, nc }(KC, MC, NC)
	for _, kn := range benchKernels(b) {
		am, bm, c := benchMat(n, n), benchMat(n, n), benchMat(n, n)
		for _, kc := range []int{128, 256, 384, 512} {
			for _, mc := range []int{72, 144, 288} {
				for _, nc := range []int{512, 1024, 2048} {
					b.Run(fmt.Sprintf("%s/kc=%d/mc=%d/nc=%d", kn.ID(), kc, mc, nc), func(b *testing.B) {
						KC, MC, NC = kc, mc, nc
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							Gemm(kn, false, false, n, n, n, 1, am, n, bm, n, 0, c, n)
						}
						b.ReportMetric(flops*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
					})
				}
			}
		}
	}
}

// BenchmarkPackDirections is #21's first item: what the two pack directions cost,
// per element and per call, at the block shapes the nest actually uses, for each of
// the four transpose flag combinations.
//
// # Which side transposes, since both issues have it backwards
//
// #21 says "the transposing direction is on the A side for TN/TT and the B side for
// NN/TN", and #26 says the B pack is "the transposing direction" for the benchmark
// shape. Both are inverted. internal/pack is the arbiter and it is unambiguous:
// APanels passes !trans as depthContig and BPanels passes trans, so
//
//	NN   A transposes, B copies      <- the shape every benchmark in this repo runs
//	NT   A transposes, B transposes  <- both scalar; #21's real worst case
//	TN   A copies,     B copies      <- neither scalar
//	TT   A copies,     B transposes
//
// The package doc of internal/pack states the rule correctly ("whichever axis is
// contiguous in the source is copied"); it is the issue text that misapplied it. It
// matters for what gets measured: a campaign that went looking for a scalar B pack
// at NN would find memmove and conclude packing was cheap, and it would never reach
// NT, where both directions are scalar at once.
//
// So the flags are a dimension here rather than a claim. Elements are counted off
// the block generator — valid elements only, padding excluded, since padding is
// zero-fill rather than data movement — so Gelem/s compares the two directions on
// the axis they differ on, and ns/op is what the decomposition's fractions use.
//
// This benchmark has no vector code under it: internal/pack is pure Go with no
// build tags. The dev host is therefore a legitimate platform for the *shape* of
// the asymmetry, though not for any host's absolute numbers.
func BenchmarkPackDirections(b *testing.B) {
	flags := []struct {
		name           string
		transA, transB bool
	}{{"NN", false, false}, {"NT", false, true}, {"TN", true, false}, {"TT", true, true}}

	for _, kn := range tileShapes() {
		for _, n := range []int{2048} {
			m, k := n, n
			blocks := nestBlocks(kn, m, n, k)
			var aelem, belem float64
			for _, blk := range blocks {
				aelem += float64(blk.im) * float64(blk.kk)
				if blk.newPanel {
					belem += float64(blk.jn) * float64(blk.kk)
				}
			}
			kc, mc, nc := plan(kn, m, n, k)
			ap := make([]float32, pack.ALen(kn.MR, mc, kc))
			bp := make([]float32, pack.BLen(kn.NR, nc, kc))
			src := benchMat(n, n) // square, so one source serves every flag combination
			for _, f := range flags {
				b.Run(fmt.Sprintf("%s/n=%d/%s/pack-a", kn.Tile(), n, f.name), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						packAOnly(kn, m, n, k, src, k, ap, f.transA)
					}
					b.ReportMetric(aelem*float64(b.N)/b.Elapsed().Seconds()/1e9, "Gelem/s")
				})
				b.Run(fmt.Sprintf("%s/n=%d/%s/pack-b", kn.Tile(), n, f.name), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						packBOnly(kn, m, n, k, src, n, bp, f.transB)
					}
					b.ReportMetric(belem*float64(b.N)/b.Elapsed().Seconds()/1e9, "Gelem/s")
				})
			}
		}
	}
}

// nestBlock is one (jc, pc, ic) block of the nest: the column block, the depth
// block and the row block, each with the length actually covered.
//
// newPanel marks the blocks where gemm packs B — once per (jc, pc), before the ic
// loop, so it is the ic == 0 block of each column/depth pair.
type nestBlock struct {
	jc, jn   int
	pc, kk   int
	ic, im   int
	newPanel bool
}

// nestBlocks is the block sequence of gemm's three outer loops, as data.
//
// Every part of the decomposition below is driven by this one generator, so the
// parts cannot walk different block structures from each other. That it walks the
// same structure as the *shipped* nest is not asserted by inspection: nest_test.go
// drives a full pack-and-multiply GEMM from this sequence and requires it to
// reproduce Gemm's output element for element, at shapes that are and are not
// multiples of the block sizes. If a bound here drifted from block.go, that test
// fails.
//
// The triangular mask is deliberately absent: the mask's skips are P4's subject and
// #26 is about the plain GEMM path every routine's bulk work goes through.
func nestBlocks(kn kern.Kernel, m, n, k int) []nestBlock {
	kc, mc, nc := plan(kn, m, n, k)
	var out []nestBlock
	for jc := 0; jc < n; jc += nc {
		jn := min(nc, n-jc)
		for pc := 0; pc < k; pc += kc {
			kk := min(kc, k-pc)
			for ic := 0; ic < m; ic += mc {
				out = append(out, nestBlock{
					jc: jc, jn: jn, pc: pc, kk: kk,
					ic: ic, im: min(mc, m-ic), newPanel: ic == 0,
				})
			}
		}
	}
	return out
}

// nestNoPack is the nest with the two pack calls removed and nothing else changed:
// the same block sequence, the same beta pass, the same microkernel calls at the
// same kk, the same fringe handling, over the same buffers.
//
// A flag on gemm itself was the alternative and was rejected: an `if !skipPack`
// inside the shipped routine would add a branch to the nest that ships in order to
// measure the nest that ships.
func nestNoPack(kn kern.Kernel, m, n, k int, beta float32, ap, bp, c []float32, ldc int, tile []float32) {
	for _, b := range nestBlocks(kn, m, n, k) {
		cb := c[b.ic*ldc+b.jc:]
		if b.pc == 0 {
			scaleTri(beta, b.ic, b.jc, b.im, b.jn, cb, ldc, triMask{})
		}
		macro(kn, ap, bp, b.ic, b.jc, b.im, b.jn, b.kk, cb, ldc, tile, triMask{})
	}
}

// packAOnly and packBOnly make exactly the pack calls the nest makes, at the same
// shapes and in the same order, and nothing else. Note the asymmetry they expose:
// APanels runs once per (jc, pc, ic) and so is re-done for every column block,
// while BPanels runs once per (jc, pc). At n=2048 there is one column block and the
// two counts differ only by the ic loop, but at a shape with several column blocks
// the A packing is repeated work — a property of this loop order, not a defect, and
// BenchmarkNest's marker prints both counts.
func packAOnly(kn kern.Kernel, m, n, k int, a []float32, lda int, ap []float32, trans bool) {
	for _, b := range nestBlocks(kn, m, n, k) {
		pack.APanels(ap, kn.MR, 1, a, lda, trans, b.ic, b.im, b.pc, b.kk)
	}
}

func packBOnly(kn kern.Kernel, m, n, k int, bm []float32, ldb int, bp []float32, trans bool) {
	for _, b := range nestBlocks(kn, m, n, k) {
		if b.newPanel {
			pack.BPanels(bp, kn.NR, bm, ldb, trans, b.pc, b.kk, b.jc, b.jn)
		}
	}
}

// tileShapes is the distinct MR×NR shapes registered on this build, scalar
// references included: a pack cost depends on the tile geometry and on nothing else
// about the kernel, so restricting these to a vector backend would refuse to
// measure a pure-Go routine on a host that can run it.
func tileShapes() []kern.Kernel {
	var out []kern.Kernel
	seen := map[string]bool{}
	for _, k := range kern.Measured() {
		if seen[k.Tile()] {
			continue
		}
		seen[k.Tile()] = true
		out = append(out, k)
	}
	return out
}

// benchKernels is the shipped shapes worth measuring at 2048³: everything
// internal/kern registers for a real backend, scalar reference tiles excluded.
func benchKernels(b *testing.B) []kern.Kernel {
	var out []kern.Kernel
	for _, k := range kern.Kernels() {
		if k.Name == kern.Scalar {
			continue
		}
		out = append(out, k)
	}
	if len(out) == 0 {
		b.Skip("no vector microkernel on this build (simd/archsimd is amd64-only, T1); " +
			"run this on a host from docs/hosts.md")
	}
	return out
}

func benchMat(rows, cols int) []float32 {
	v := make([]float32, rows*cols)
	for i := range v {
		v[i] = float32((i%13)-6) * 0.125
	}
	return v
}

// benchPanels is bench/kernel_test.go's kernelPanels: one tile's worth of packed
// panels with a padded ldc, so the kernel denominator here is measured the same way
// P2's was.
func benchPanels(kn kern.Kernel, kc int) (a, b, c []float32, ldc int) {
	rng := rand.New(rand.NewSource(1))
	ldc = kn.NR + 16
	a = make([]float32, kc*kn.MR)
	b = make([]float32, kc*kn.NR)
	c = make([]float32, kn.MR*ldc)
	for i := range a {
		a[i] = rng.Float32()*2 - 1
	}
	for i := range b {
		b[i] = rng.Float32()*2 - 1
	}
	return a, b, c, ldc
}

var benchSink float32
