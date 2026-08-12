// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Package bench holds the benchmark harness (grown from P1). Every reported
// number must carry its denominator: CPU model, theoretical peak, and the
// OpenBLAS reference when the dev-only cgo harness (build tag `openblas`)
// is available on the machine. See DESIGN.md §5 and §7 rule 7.
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
)

// # What this harness measures, and what it deliberately does not
//
// These benchmarks call keel through its public API with whatever backend
// dispatch selected, which is what a caller actually gets. They do not iterate
// over backends: the backend table is unexported, and a benchmark that reached
// into it would be measuring something no caller can invoke. Pin a backend with
// KEEL_FORCE=scalar|avx2|avx512 and run again to compare — the same mechanism
// gate-p1 uses, so the comparison is one users can reproduce.
//
// Provenance is printed once per run: CPU model, core count, governor, and the
// clock as reported at bench time. DESIGN.md §7 rule 7 requires every number to
// carry its denominator.
//
// The percent-of-peak denominator is measured on this host by BenchmarkPeak, not
// derived from a formula (decision on issue #11 — DESIGN.md's formula assumes two
// full-width 512-bit FMA units, which Zen 4 does not have). The formula is printed
// as a cross-check only; see peak_test.go, and internal/vec/peak.go for what makes
// the measured number a ceiling.
//
// The percentage itself is computed by the gate, from benchstat medians of this
// benchmark and BenchmarkPeak on the same host in the same run. Nothing in this
// process divides one rate by another: a second, statistics-free denominator is
// precisely what DESIGN.md §7 rule 7 forbids.
//
// GFLOP/s below is therefore a measured rate: comparable across runs on one
// machine, and between machines only as a rate.

var provenanceOnce sync.Once

func provenance() {
	provenanceOnce.Do(func() {
		fmt.Println("keel-bench-cpu:", cpuModel())
		fmt.Println("keel-bench-cores:", runtime.NumCPU(), "logical")
		fmt.Println("keel-bench-governor:", governor())
		fmt.Println("keel-bench-clock-mhz:", clockMHz())
		fmt.Println("keel-bench-platform:", runtime.GOOS+"/"+runtime.GOARCH)
		fmt.Println("keel-bench-backend:", keel.ActiveL1Backend(),
			"(available: "+strings.Join(keel.AvailableL1Backends(), " ")+")")
		fmt.Println("keel-bench-peak-method: measured on this host by " +
			"BenchmarkPeak (register-only FMA saturation); the formula below is a " +
			"cross-check, not the denominator (issue #11)")
		for _, line := range peakFormulaLines() {
			fmt.Println("keel-bench-peak-formula:", line)
		}
		fmt.Println("keel-bench-gomaxprocs:", runtime.GOMAXPROCS(0))
		fmt.Println("keel-bench-kern:", keel.ActiveKernTile()+"/"+keel.ActiveKernBackend(),
			"(available: "+strings.Join(keel.AvailableKernels(), " ")+")")
		// The shape above is chosen per host, so the marker has to carry the
		// grounds as well as the answer: the gate measures the host's class
		// itself and fails if its verdict and this classification disagree
		// (issue #24). Without the class line, "4x32" alone cannot be told from
		// "4x32 on a host that should be running 2x32".
		fmt.Println("keel-bench-kern-class:", keel.ActiveKernClass(),
			"insns-per-fma="+strconv.FormatFloat(keel.ActiveKernInsnsPerFMA(), 'f', 3, 64),
			"("+keel.ActiveKernClassEvidence()+")")
		fmt.Println("keel-bench-kern-audit:", strings.Join(keel.KernelAudits(), " "))
		fmt.Println("keel-bench-openblas:", openblasProvenance)
	})
}

// openblasProvenance describes the reference this binary can measure against.
// The openblas-tagged file replaces it with the library's own report of itself
// and its thread count; without that tag there is no reference, and the marker
// says so rather than leaving the gate to infer it from a missing line.
//
// GOMAXPROCS is printed beside it because the P3 criterion is single-thread on
// both sides: an OpenBLAS pinned to one thread against a keel free to use
// sixteen would be a comparison of two different questions.
var openblasProvenance = "not available (no cgo reference harness built; " +
	"rebuild with -tags openblas on a host that has OpenBLAS)"

// The three readers below are best-effort and say so when they fail. A missing
// governor is reported as unknown rather than guessed at: "performance" assumed
// wrongly would silently flatter every number underneath it.

func cpuModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown (" + runtime.GOARCH + ", no /proc/cpuinfo)"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return "unknown (no model name in /proc/cpuinfo)"
}

func governor() string {
	b, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor")
	if err != nil {
		return "unknown (no cpufreq)"
	}
	return strings.TrimSpace(string(b))
}

// clockMHz reports the range of per-core clocks at the moment of the call. It is
// a snapshot, not the frequency sustained during the measurement — enough to
// catch a throttled or idling host, not enough to compute a peak from, which is
// one more reason the peak line above is withheld.
func clockMHz() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	lo, hi := 0.0, 0.0
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) != "cpu MHz" {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%g", &f); err != nil {
			continue
		}
		if lo == 0 || f < lo {
			lo = f
		}
		if f > hi {
			hi = f
		}
	}
	if hi == 0 {
		return "unknown (no cpu MHz field)"
	}
	return fmt.Sprintf("%.0f-%.0f (snapshot, not sustained)", lo, hi)
}

// Sizes span L1-resident to memory-resident so the shape of the rate curve is
// visible: the small end measures the kernel, the large end measures the memory
// system, and reporting only one of them is how a BLAS gets to claim a number it
// cannot sustain.
var sizes = []int{256, 4096, 65536, 1 << 20}

func makeVec(n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32(i%17) - 8
	}
	return v
}

// rate reports GFLOP/s for a routine performing flopsPerElem operations per
// element. b.Elapsed() excludes setup, so the denominator is the timed region
// only.
func rate(b *testing.B, n int, flopsPerElem float64) {
	b.ReportMetric(flopsPerElem*float64(n)*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
}

func BenchmarkL1Sdot(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x, y := makeVec(n), makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(2 * 4 * n))
			for i := 0; i < b.N; i++ {
				sink = keel.Sdot(n, x, 1, y, 1)
			}
			rate(b, n, 2) // one multiply + one add per element
		})
	}
}

func BenchmarkL1Saxpy(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x, y := makeVec(n), makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(3 * 4 * n)) // read x, read y, write y
			for i := 0; i < b.N; i++ {
				keel.Saxpy(n, 1.0000001, x, 1, y, 1)
			}
			rate(b, n, 2)
		})
	}
}

func BenchmarkL1Sscal(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x := makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(2 * 4 * n))
			for i := 0; i < b.N; i++ {
				keel.Sscal(n, 1.0000001, x, 1)
			}
			rate(b, n, 1)
		})
	}
}

func BenchmarkL1Sasum(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x := makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(4 * n))
			for i := 0; i < b.N; i++ {
				sink = keel.Sasum(n, x, 1)
			}
			rate(b, n, 1)
		})
	}
}

func BenchmarkL1Snrm2(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x := makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(4 * n))
			for i := 0; i < b.N; i++ {
				sink = keel.Snrm2(n, x, 1)
			}
			rate(b, n, 2)
		})
	}
}

func BenchmarkL1Isamax(b *testing.B) {
	provenance()
	for _, n := range sizes {
		x := makeVec(n)
		b.Run(fmt.Sprint("n=", n), func(b *testing.B) {
			b.SetBytes(int64(4 * n))
			for i := 0; i < b.N; i++ {
				sinkI = keel.Isamax(n, x, 1)
			}
			// No GFLOP/s here: Isamax does comparisons, not arithmetic.
			// Reporting a flop rate for it would be inventing a numerator.
		})
	}
}

var (
	sink  float32
	sinkI int
)
