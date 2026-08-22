// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/scttfrdmn/keel"
	"github.com/scttfrdmn/keel/internal/vec"
)

// The two halves of P5's per-host attainable ceiling (ruling on #6, 2026-08-20).
//
// # What replaced what
//
// gate-p5's scaling criterion was a fixed 6.0x floor on the 8-thread/1-thread
// ratio, and that floor was rank-ordered AGAINST per-core efficiency: at ce43bca
// it refused zen4 keeping 65.9% of 8x its own core peak and passed gnr keeping
// 34.3%, monotonically across all three hosts, and no AMD host has ever cleared it
// in this project's history. A fixed T8/T1 ratio does not reward good parallel
// code; it rewards a BAD single-thread baseline, because a host whose one thread
// already saturates its memory system has no ratio headroom left while a
// front-end-crippled one scales beautifully into its own slack.
//
// So the denominator becomes measured: a per-host 8-thread attainable ceiling,
// min(compute ceiling at 8 threads, bandwidth bound at 8 threads), with the
// criterion judging achieved-against-own-ceiling. That is the same move as the
// issue-roofline class in scripts/roofline.sh and the SPR feed-bound reading,
// applied to the parallel nest, and its legitimacy predates the run by the whole
// of DESIGN.md §5: measured denominators, derived ceilings, never typed constants.
//
// # Why the compute half is measured AT 8 threads and not 8x the 1-thread peak
//
// gate-p5 already printed "% of 8x the single-thread avx512 peak" as an info line,
// carrying its own disclaimer: "that denominator ignores the clock's drop with core
// count". It does, and that makes 8x the 1-thread peak a ceiling no host can reach
// by construction — every part here drops several hundred MHz moving from one
// active core under a 512-bit license to eight. A ceiling that is unattainable for
// a reason having nothing to do with the parallel nest cannot be the denominator of
// a criterion about the parallel nest. BenchmarkCeiling/compute measures the same
// register-only FMA saturation kernels BenchmarkPeak uses, run concurrently on P
// threads, so the clock droop is inside the reading rather than disclaimed beside
// it. The 1-thread arm is measured too, so the droop itself is a published number
// (compute/threads=8 against 8x compute/threads=1) rather than an assertion.
//
// # Why the bandwidth half exists even though it is not expected to bind
//
// Counted from the nest's own blocking at 4096³ — KC=384, MC=144, NC=4096, so
// eleven pc iterations each sweeping all of C — the traffic is about 64 MB for A,
// 64 MB for B and 11x128 MB for C's read-modify-write, near 1.5 GB against 137
// GFLOP, i.e. of order 90 FLOP per byte. Against any plausible 8-thread bandwidth
// that puts the memory bound several times above the compute bound, and the min()
// should collapse to compute.
//
// That is a PREDICTION, and DESIGN.md §5 rule 11 is explicit that the instrument
// adjudicates the reasoning that motivated it before that reasoning is published.
// It is measured here for exactly that reason. It is also the half that can change
// under a host this project has not met: a guest with fewer memory channels per
// core, or a future criterion at a size where the nest is no longer this
// compute-dense, flips which term binds — and a min() whose second term was never
// measured is a min() over one number.
//
// # The probes are keel's own L1 routines, not a new kernel
//
// Sdot is two streaming reads per element and four independent accumulator chains,
// so it is bandwidth-bound and not FMA-latency-bound; Saxpy is two reads and one
// write with no chain between iterations at all. Both are already
// differential-tested against the scalar shim, which a purpose-built stream kernel
// in this file would not be, and both have a byte count that is countable rather
// than modelled. No SIMD is written here.
//
// Neither is parallel — keel's L1 routines are single-threaded by design — so the
// P-thread arm runs P goroutines over DISJOINT slices, which is also what the
// nest's ic partition does to C.

// streamThreads is the pair every arm here is measured over, and it is
// scaleThreads itself rather than a copy of its value. The ceiling and the ratio
// it is the denominator of must be measured at the same thread counts or the
// division is between two different machines; sharing the var makes that
// impossible to break by editing one of two lists.
var streamThreads = scaleThreads

// streamBytesPerThread is how much memory each thread streams per array.
//
// Sized from the measured last-level cache rather than typed, because the quantity
// that matters is "past the largest cache that could be absorbing this", and that
// is a host property differing by an order of magnitude across the fleet — a
// Granite Rapids socket carries hundreds of MB of L3 where the desktop parts this
// project started on carried tens. A stream that fits in L3 measures L3 and
// reports it as memory bandwidth, which is the flattering direction.
//
// Four times the LLC, floored at 256 MB for a host whose cache sizes are
// unreadable. The multiplier and the floor are both stated in the declaration line
// with the LLC that produced them, so a reader can see which one applied.
// KEEL_STREAM_BYTES overrides it, for two uses that would otherwise be impossible:
// a race or checkptr build, whose shadow memory multiplies a 4 GB aggregate past
// what a machine has, and a host too small for the derived size. The override is
// echoed into the marker line so any run carrying one is self-identifying — a
// measurement whose working set came from the environment must say so, or the
// number is not comparable to one that derived its own.
func streamBytesPerThread() (n int, why string) {
	if v := os.Getenv("KEEL_STREAM_BYTES"); v != "" {
		if b, err := strconv.Atoi(v); err == nil && b >= 1<<20 {
			return b, "KEEL_STREAM_BYTES=" + v + " (overridden, NOT derived from this host)"
		}
	}
	if llc, src := llcBytes(); llc > 0 {
		if b := 4 * llc; b > 256<<20 {
			return b, fmt.Sprintf("4x %d MB LLC from %s", llc>>20, src)
		}
		return 256 << 20, fmt.Sprintf("256 MB floor (4x %d MB LLC from %s is smaller)", llc>>20, src)
	}
	return 256 << 20, "256 MB floor (no readable cache size)"
}

// llcBytes returns the largest cache size sysfs reports for cpu0, and where it
// read it. Zero when nothing is readable, which the caller reports rather than
// papering over: a fabricated cache size would set the working-set size of every
// bandwidth number here.
func llcBytes() (int, string) {
	var best int
	var src string
	for i := 0; i < 5; i++ {
		p := fmt.Sprintf("/sys/devices/system/cpu/cpu0/cache/index%d/size", i)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(b))
		mult := 1
		switch {
		case strings.HasSuffix(s, "K"):
			mult, s = 1<<10, strings.TrimSuffix(s, "K")
		case strings.HasSuffix(s, "M"):
			mult, s = 1<<20, strings.TrimSuffix(s, "M")
		case strings.HasSuffix(s, "G"):
			mult, s = 1<<30, strings.TrimSuffix(s, "G")
		}
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			continue
		}
		if v*mult > best {
			best, src = v*mult, fmt.Sprintf("cache/index%d/size", i)
		}
	}
	return best, src
}

// maxStreamThreads is the widest arm, taken from the thread list rather than typed
// so the buffer pool below cannot be sized for a thread count the arms no longer
// use.
var maxStreamThreads = func() int {
	m := 1
	for _, p := range streamThreads {
		if p > m {
			m = p
		}
	}
	return m
}()

// streamBufs hands out the per-thread slice pairs, allocating them ONCE for the
// whole process at the widest arm's size.
//
// Allocating inside an arm made the arm's timing a function of which arm ran
// before it: each one wanted about 4 GB at the 8-thread size, so the first sample
// paid page faults and every later one ran against the previous arm's garbage.
// That is not a small effect and it is not random — the first smoke run had
// stream/dot at 8 threads reading 15.4 GB/s against its own 1-thread arm's 24.1,
// a decline with thread count that no memory system produces and the allocator
// explains. Allocated once, touched once, reused by every arm: the only thing
// differing between two rows here is the thread count and the probe.
var (
	streamBufOnce sync.Once
	streamBufX    [][]float32
	streamBufY    [][]float32
)

func streamBufs(elems int) (xs, ys [][]float32) {
	streamBufOnce.Do(func() {
		streamBufX = make([][]float32, maxStreamThreads)
		streamBufY = make([][]float32, maxStreamThreads)
		for t := 0; t < maxStreamThreads; t++ {
			streamBufX[t], streamBufY[t] = makeVec(elems), makeVec(elems)
		}
	})
	return streamBufX, streamBufY
}

// streamCase is one memory probe: a name, the bytes of architectural traffic it
// moves per element, and a body over one thread's own slices.
//
// bytesPerElem counts what the ISA asks for. It is NOT a claim about DRAM traffic:
// a write on a write-allocate cache pulls the line in first, so axpy's real bus
// traffic is up to 16 bytes per element where the architecture asks for 12. That
// systematic is disclosed in the marker line and left in rather than corrected,
// because the correction depends on the host's write policy and this file has no
// instrument for it — two estimators 1.33x apart bracket the truth, and the
// bracket is the honest object (#118's estimator-spread reasoning). The read-only
// dot probe has no such ambiguity, which is why both patterns are measured.
type streamCase struct {
	name         string
	arrays       int
	bytesPerElem int
	rfo          string
	body         func(t int, x, y []float32)
}

func streamCases() []streamCase {
	return []streamCase{{
		// Two streaming reads, no writes, so its byte count carries no write-allocate
		// ambiguity — but "unambiguous" describes the COUNT and not the reading, and
		// the first run of this file made the difference matter. On the dev host's
		// scalar path dot read 32.7 GB/s at one thread where axpy read 44.6: the
		// read-only probe slower than the read-modify-write one, which no memory
		// system does. Sdot's own throughput was the limit there, not memory, and a
		// probe that is not at the memory bound is not measuring the memory bound. At
		// eight threads the two converged (190.6 vs 193.0), which is what both being
		// memory-limited looks like. So this arm is a bandwidth reading only where its
		// 8-thread figure sits near axpy's; where it sits well below, it is a floor on
		// bandwidth and a measurement of Sdot (§5 rule 11 — the instrument adjudicated
		// the sentence that used to be on this line).
		name: "dot", arrays: 2, bytesPerElem: 8, rfo: "none: read-only",
		body: func(t int, x, y []float32) { streamSink[t*16] = keel.Sdot(len(x), x, 1, y, 1) },
	}, {
		// Two reads and one write — the pattern the nest applies to C, which is
		// where its traffic is dominated (eleven sweeps of a 64 MB C at 4096³).
		name: "axpy", arrays: 2, bytesPerElem: 12, rfo: "up to 16 B/elem with write-allocate, i.e. this figure x1.33",
		body: func(t int, x, y []float32) { keel.Saxpy(len(x), 1.0000001, x, 1, y, 1) },
	}}
}

// One sink per thread, spaced a cache line apart.
//
// A shared sink is a data race whether or not anything reports it, and here the
// detector largely cannot: T17 makes `-race` a fatal checkptr error on amd64
// wherever archsimd's partial load/store are reached, which Sdot's tail reaches, so
// on the fleet this discipline holds without an instrument behind it (§5 rule 12 —
// the blind spot goes inside the claim). It was checked under `-race` on
// darwin/arm64, where the scalar path is taken and the build survives, which is a
// weaker witness than it looks. Independently of the race: one line written by
// eight cores is a line eight cores ping-pong, and that traffic would be reported
// as memory bandwidth.
var streamSink = make([]float32, maxStreamThreads*16)

// BenchmarkCeiling measures both halves of the attainable ceiling, each at every
// thread count the scaling ratio is taken over.
//
// Row names are four-element like BenchmarkScale's, so a gate can address one arm
// exactly — "Ceiling/compute/avx512/threads=8", "Ceiling/stream/axpy/threads=1".
// The half sits before the variant so adding a third probe renumbers nothing.
func BenchmarkCeiling(b *testing.B) {
	provenance()
	b.Run("compute", func(b *testing.B) {
		for _, k := range vec.PeakKernels() {
			b.Run(k.Name, func(b *testing.B) {
				for _, procs := range streamThreads {
					b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) {
						computeArm(b, procs, k)
					})
				}
			})
		}
	})
	b.Run("stream", func(b *testing.B) {
		for _, c := range streamCases() {
			b.Run(c.name, func(b *testing.B) {
				for _, procs := range streamThreads {
					b.Run(fmt.Sprint("threads=", procs), func(b *testing.B) {
						streamArm(b, procs, c)
					})
				}
			})
		}
	})
}

// computeArm runs one width's FMA saturation kernel on procs threads at once.
//
// # Instrument v2: persistent workers, and the choreography is off the clock
//
// v1 forked procs goroutines and joined them inside every one of b.N iterations,
// and that made this instrument measure the wrong noun. A ceiling named "what
// eight threads can compute" was reading **78% compute and 22% fork/join**: the
// term costs ~60 µs on keel-zen4 under rule 5's spread mask against a 268 µs op,
// and the arm hoisting it out of the loop read 1.010 of eight times this host's
// own 1-thread rate where v1 read 1.290 (keel-zen5: 1.014 against 1.599, duty
// cycle 0.97-0.99 against 0.67-0.88 — the lost time was workers *parked*). T-52's
// full attribution is on #115; the v1 arm and its paired diagnostic are at
// 2b1d60b if the comparison ever needs re-running.
//
// **This is a fidelity repair to a defect that predates the mask.** At 25.5 µs per
// fork/join the confined-placement ceilings carried the same contamination at
// smaller amplitude, so the spread mask did not degrade compute — it amplified a
// tax this arm was always paying, and one the judged routines never could feel:
// the same fork/join in internal/par is 0.03% of a 204 ms Sgemm op against 22%
// here, a 700:1 exposure ratio to one term (§5 rule 14, severity as a function of
// deployment context). It is also why a ceiling could read BELOW rates it
// denominated without any paradox — this arm was timing compute plus its own
// harness while the routines timed compute alone (the refusal that catches it is
// gate-p5's, landed 8e6c6ac).
//
// So: workers are created ONCE, before the timer, and each has already executed a
// full op on its own cpu before the clock starts, which pays the cold-M wake-up
// off the clock too. The timed region is one channel close, the steady-state
// loops, and one Wait — b.N joins reduced to one, whose ~60 µs is then 0.005% of
// a 1 s benchmark instead of 22% of an op. Keep that shape. Reintroducing a
// per-iteration barrier does not merely add noise; it silently lowers every
// judged share's denominator, and no test here will go red for it.
//
// Readings from before this arm landed are instrument-v1 and are era-scoped, not
// corrected: v1's bias is host- and mask-dependent AND varies run to run (that
// variance was the scatter), so no formula recovers them honestly. Nothing rests
// on them — the pre-commitment discipline meant no bar was ever typed from a v1
// ceiling, so an instrument defect cost a version label instead of a retraction.
//
// # What v2 trades for it, which v1 did not have
//
// v1's reading was b.N-independent by construction, because every iteration paid
// its own fork. v2's is not: the one remaining fork/join is amortized over b.N, so
// a SHORT run under-reads. Exercised on the author's laptop rather than measured
// on a judged host (darwin/arm64, scalar path, no mask — a mechanism check, not a
// number that goes anywhere), the 8-thread arm reads 74.15 GFLOP/s at
// -benchtime=1x, 94.71 at 5x, 99.00 at 50x and 127.2 / 127.5 at 500x / 5000x:
// converged two decades before the 1 s the gates use, and 127.5 against 8x the
// same host's 17.98 single-thread reading is 0.886 — the residual there is an M4
// Pro putting 8 threads across heterogeneous cores, which is why that host is not
// a judged one. The bias is one fork/join over the whole timed region, so at the
// 1 s every gate uses against 268 µs ops it is ~0.005%. A too-short run biases the
// ceiling DOWN and every share UP, which is the direction gate-p5's
// impossible-denominator refusal already fails closed on.
//
// b.N is therefore load-bearing for this reading in a way it was not for v1, and
// it is auditable where testing already prints it: the iteration column of the row
// itself, which is what benchstat parses and what the archive keeps. It is NOT put
// in the declaration line below, and that is a measured refusal rather than a
// preference — declareCeiling keeps one line per row and testing calls this
// function once per b.N trial, ramping from 1, so the first call wins and any
// b.N-dependent field there is frozen at b.N=1 forever. Adding `ops=%d` printed
// `ops=1` beside a row that had just measured 3853 iterations. Every other field
// in that line is b.N-invariant by construction (flops= is per-op, iters= is a
// constant), which is why the dedup was safe before and is a trap now.
func computeArm(b *testing.B, procs int, k vec.PeakKernel) {
	prev := runtime.GOMAXPROCS(procs)
	defer runtime.GOMAXPROCS(prev)
	// The same witness BenchmarkPeak checks, for the same reason: a collapsed
	// accumulator chain reports a plausible number, and this one is a denominator.
	if got, want := k.Run(1000), k.Witness(1000); got != want {
		b.Fatalf("%s: witness Run(1000) = %v, want %v — accumulator chains did not "+
			"survive compilation, so this is not a ceiling", k.Name, got, want)
	}
	// One sink per thread, spaced a cache line apart, for the reasons on streamSink.
	// The write lands once per peakItersPerOp iterations so the sharing would be
	// immaterial to the timing either way; the race would not be.
	sinks := make([]float32, procs*16)
	n := b.N
	var wg sync.WaitGroup
	// Buffered by procs so reporting warm cannot itself park a worker.
	warm := make(chan struct{}, procs)
	start := make(chan struct{})
	for t := 0; t < procs; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			sinks[t*16] = k.Run(peakItersPerOp)
			warm <- struct{}{}
			<-start
			for i := 0; i < n; i++ {
				sinks[t*16] = k.Run(peakItersPerOp)
			}
		}(t)
	}
	// Every worker has now run a full op on the cpu the mask gave it, so what the
	// clock below sees is a steady state rather than eight threads starting up.
	for t := 0; t < procs; t++ {
		<-warm
	}
	b.ResetTimer()
	close(start)
	wg.Wait()
	b.StopTimer()
	flops := float64(k.FlopsPerIter) * float64(peakItersPerOp) * float64(procs) * float64(n)
	b.ReportMetric(flops/b.Elapsed().Seconds()/1e9, "GFLOP/s")
	declareCeiling(b, procs, fmt.Sprintf("kind=compute instrument=v2 width=%s flops=%s iters=%d",
		k.Name, strconv.FormatFloat(flops/float64(n), 'f', 0, 64), peakItersPerOp))
}

// streamArm runs one memory probe on procs threads over disjoint slices.
//
// This arm still forks per iteration, deliberately, and the residual is stated
// rather than removed (§5 rule 12). Its op sweeps at least 256 MB per thread —
// order 10 ms against the compute arm's 268 µs — so the same ~60 µs fork/join is
// order 0.5% here instead of 22%, and this half is REPORTED and never in the
// min() that denominates a share. Hoisting it is not the same one-line change
// either: a warm-up round would move first-touch page faults out of the timed
// region, which moves a published bandwidth number for a reason that has nothing
// to do with #115's finding. That is its own decision with its own magnitude to
// measure, so it is not smuggled in beside the ceiling repair.
func streamArm(b *testing.B, procs int, c streamCase) {
	prev := runtime.GOMAXPROCS(procs)
	defer runtime.GOMAXPROCS(prev)
	per, why := streamBytesPerThread()
	elems := per / 4
	// Per-thread footprint held constant across the thread arms, so the aggregate
	// grows with procs. That is what a parallel streaming workload does, and holding
	// the AGGREGATE constant instead would shrink each thread's slice toward cache
	// residency exactly as the arm that needs DRAM the most is measured.
	xs, ys := streamBufs(elems)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for t := 0; t < procs; t++ {
			wg.Add(1)
			go func(t int) {
				defer wg.Done()
				c.body(t, xs[t], ys[t])
			}(t)
		}
		wg.Wait()
	}
	bytes := float64(c.bytesPerElem) * float64(elems) * float64(procs)
	b.ReportMetric(bytes*float64(b.N)/b.Elapsed().Seconds()/1e9, "GB/s")
	declareCeiling(b, procs, fmt.Sprintf("kind=stream pattern=%s bytes=%s elems=%d arrays=%d "+
		"bytes_per_elem=%d per_thread_bytes=%d sizing=%q rfo=%q",
		c.name, strconv.FormatFloat(bytes, 'f', 0, 64), elems, c.arrays,
		c.bytesPerElem, per, why, c.rfo))
}

// ceilingDeclared keeps the marker to one line per row, as flopsDeclared does.
var ceilingDeclared = map[string]bool{}

// declareCeiling prints the denominator this row used, and the worker count it
// actually got from the library's own answer rather than from what the harness
// believes it set.
//
// The numbers are printed for the gate to RECOMPUTE from, which is why the byte and
// flop counts are here in full and not left to the reported metric: that metric is
// formatted by testing.prettyPrint, which picks its decimals from a value's
// magnitude, so a GB/s column near 50 carries a 0.01 quantum where the row's own
// sec/op carries six digits (docs/toolchain-notes.md T26). A gate dividing this
// declared count by sec/op keeps the precision; one re-parsing the display column
// throws it away, which is the defect that made §5 rule 5's clock test decide on
// coin flips.
func declareCeiling(b *testing.B, procs int, detail string) {
	name := benchRowName(b)
	if ceilingDeclared[name] {
		return
	}
	ceilingDeclared[name] = true
	fmt.Printf("keel-bench-ceiling: name=%s gomaxprocs=%d threads=%d %s\n",
		name, keel.GOMAXPROCS(), procs, detail)
}
