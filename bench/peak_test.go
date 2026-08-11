// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/scttfrdmn/keel/internal/vec"
)

// The percent-of-peak denominator.
//
// Decision on issue #11: peak is measured, not derived. BenchmarkPeak runs
// internal/vec's register-only FMA saturation kernels — no memory in the loop,
// enough independent accumulator chains to cover FMA latency — and its GFLOP/s
// is the denominator used everywhere a percent-of-peak appears, including P2's
// 55% floor. See internal/vec/peak.go for why each property of those kernels is
// load-bearing and what happens to the number when one is violated.
//
// There is exactly one denominator, and it is produced the same way as every
// other gate number: as a benchmark, aggregated by benchstat over -count=10
// (issue #14). The gate divides one median by the other and requires the ratio to
// clear the bar net of both confidence intervals. Nothing here computes a
// percentage in-process, because an in-process number would be a second
// denominator without statistics attached, and two denominators is the failure
// DESIGN.md §7 rule 7 exists to prevent.
//
// The DESIGN.md formula (freq · 2 FMA ports · lanes · 2 flops) survives only as
// the printed cross-check on the provenance lines: it assumes two full-width FMA
// units, which Zen 4 does not have. A measured/formula divergence of >=1.5x is
// the double-pump signature and is expected on Zen 4, not a bug — the gate prints
// it either way rather than letting it pass unremarked.

// peakItersPerOp is how many kernel iterations one benchmark op runs.
//
// Large enough that the indirect call through PeakKernel.Run and the loop
// counter's setup are lost in the noise (65536 iterations is tens of
// microseconds, the call is nanoseconds), small enough that b.N stays a useful
// unit and -benchtime=1s yields thousands of samples to average over.
const peakItersPerOp = 1 << 16

// BenchmarkPeak measures the FMA ceiling per available width.
//
// It reports GFLOP/s for whichever widths this machine can execute, always
// including the scalar reference. Note that the scalar kernel's Fused is false on
// amd64 (Go does not fuse a*b+c there), so its ceiling is the multiply and add
// ports rather than the FMA units — a different quantity, labelled as such in the
// benchmark name rather than silently averaged in.
func BenchmarkPeak(b *testing.B) {
	provenance()
	for _, k := range vec.PeakKernels() {
		b.Run(k.Name, func(b *testing.B) {
			// The witness check is here, not just in internal/vec's tests,
			// because this is the process that produces the denominator. A
			// collapsed kernel must fail loudly at the point of measurement
			// rather than report a plausible number that inflates every
			// percentage derived from it.
			if got, want := k.Run(1000), k.Witness(1000); got != want {
				b.Fatalf("%s: witness Run(1000) = %v, want %v — accumulator chains "+
					"did not survive compilation, so this is not a ceiling", k.Name, got, want)
			}
			for i := 0; i < b.N; i++ {
				peakSink = k.Run(peakItersPerOp)
			}
			flops := float64(k.FlopsPerIter) * float64(peakItersPerOp) * float64(b.N)
			b.ReportMetric(flops/b.Elapsed().Seconds()/1e9, "GFLOP/s")
		})
	}
}

var peakSink float32

// peakFormulaLines returns one cross-check line per measurable width.
//
// Printed as provenance, never used as a denominator. The clock is the maximum
// the kernel reports, not the clock sustained under an AVX-512 load — Skylake-X
// drops several hundred MHz under a 512-bit license, which is one of the reasons
// the formula is a cross-check and the measurement is the number.
func peakFormulaLines() []string {
	ghz, src := maxClockGHz()
	var out []string
	for _, k := range vec.PeakKernels() {
		if ghz == 0 {
			out = append(out, fmt.Sprintf("%s: unavailable (%s)", k.Name, src))
			continue
		}
		// DESIGN.md §4: freq x 2 FMA ports x lanes x flops-per-op, single core.
		//
		// An unfused kernel gets 1 flop per op, not 2. The formula as written in
		// DESIGN.md describes an FMA machine; applying it unchanged to a path that
		// issues a separate multiply and add claims twice the ceiling that path can
		// have, and the resulting "divergence" would be an artifact of the formula
		// rather than anything about the silicon. PeakKernel.Fused is exactly this
		// distinction, so it is honoured here.
		flopsPerOp, note := 2.0, "2 FMA ports"
		if !k.Fused {
			flopsPerOp, note = 1.0, "2 FP ports, unfused: 1 flop/op"
		}
		g := ghz * 2 * float64(k.Lanes) * flopsPerOp
		out = append(out, fmt.Sprintf("%s: %.1f GFLOP/s (%.2f GHz %s x %s x %d lanes)",
			k.Name, g, ghz, src, note, k.Lanes))
	}
	return out
}

// maxClockGHz reads the kernel's maximum core frequency, returning 0 and a reason
// when it cannot. Reporting an assumed clock would put a fabricated number in the
// denominator position of a printed formula, which is exactly the failure mode
// this file is built to avoid.
func maxClockGHz() (float64, string) {
	b, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	if err == nil {
		if khz, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil && khz > 0 {
			return khz / 1e6, "cpuinfo_max_freq"
		}
	}
	return 0, "no cpuinfo_max_freq"
}
