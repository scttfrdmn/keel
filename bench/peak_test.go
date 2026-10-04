// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"os"
	"runtime"
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
	ghz, src, clockCaveat := maxClockGHz()
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
		// The "2 ports" term is DESIGN.md §4's, and it is an amd64 observation.
		// On arm64 it is simply unverified: this project has measured 4 FMA pipes
		// on a Cortex-X925 (124.20 GFLOP/s at 3.900 GHz) and 2 on a Cortex-A725
		// (44.79 at 2.808), so the term is right for one core type and 2x low for
		// the other. Said on every arm64 line and not only on a heterogeneous
		// host, because a uniform arm64 host does not make the constant any more
		// verified -- it just removes the second core that would have exposed it
		// (#171). Not fed back from the measurement: the formula's whole value is
		// being independent of what it cross-checks, so deriving the port count
		// would force the divergence to 1.0 and destroy the double-pump signal
		// §4/P2 keeps it for.

		g := ghz * 2 * float64(k.Lanes) * flopsPerOp
		out = append(out, fmt.Sprintf("%s: %.1f GFLOP/s (%.2f GHz %s x %s x %d lanes)%s",
			k.Name, g, ghz, src, note, k.Lanes, formulaCaveats(clockCaveat)))
	}
	return out
}

// maxClockGHz reads the maximum core frequency of a CPU this process is actually
// allowed to run on, returning 0 and a reason when it cannot. Reporting an assumed
// clock would put a fabricated number in the denominator position of a printed
// formula, which is exactly the failure mode this file is built to avoid.
//
// It read cpu0 unconditionally until #171, and on a heterogeneous host that is
// wrong in the most misleading way available. The GB10 is 10x Cortex-X925 +
// 10x Cortex-A725; cpu0 is a little core at 2.808 GHz while a benchmark pinned to
// cpu19 runs at 3.900. So the formula described a core the measurement never
// touched — and because it was accurate to +0.25% for the *other* core type
// (44.9 against that core's measured 44.79) it read as validation, while the core
// under test measured 124.20. Naming the CPU in the output is half the fix; the
// other half is that a reader must be able to see the host is not uniform.
//
// The affinity mask comes from /proc/self/status's Cpus_allowed_list — a plain
// file read, so no new dependency. Falling back to cpu0 when that is unreadable is
// deliberate and is labelled as the fallback it is: on a uniform host it is the
// right answer, and on a heterogeneous one the label is what stops it being
// mistaken for a measurement of the core in use.
func maxClockGHz() (ghzOut float64, src, caveat string) {
	cpus := allowedCPUs()
	fellBack := len(cpus) == 0
	if fellBack {
		cpus = []int{0}
	}
	ghz, which := 0.0, ""
	uniform := true
	for _, c := range cpus {
		g := cpuMaxGHz(c)
		if g == 0 {
			continue
		}
		if which == "" {
			ghz, which = g, fmt.Sprintf("cpu%d", c)
			continue
		}
		if g != ghz {
			uniform = false
		}
	}
	if which == "" {
		return 0, "no cpuinfo_max_freq", ""
	}
	src = which + " cpuinfo_max_freq"
	if fellBack {
		src += ", affinity unreadable so this is a fallback and not the core in use"
	}
	if !uniform {
		// A single formula line cannot describe a host whose cores differ, so it
		// says which core it describes rather than implying it describes the
		// machine. Note also that the 2-FMA-port term below is an *assumption*
		// from DESIGN.md's formula, and it is 2 on the A725 and 4 on the X925 —
		// so on a host like this a measured/formula divergence has a second
		// cause besides the double-pumping §4/P2 reads it as (#171).
		caveat = "NON-UNIFORM host: this clock describes " + which + " only, and the " +
			"process may run on a core with a different one"
	}
	return ghz, src, caveat
}

// formulaCaveats is the trailing "-- ..." the formula line carries, assembled in
// one place so the formula body stays readable and no caveat is stated twice.
//
// They are independent and both can apply. The port-count one fires on every
// arm64 line rather than only on a heterogeneous host, because a uniform arm64
// host does not make the constant verified — it only removes the second core type
// that would have exposed it (#171).
func formulaCaveats(clock string) string {
	var cs []string
	if runtime.GOARCH == "arm64" {
		cs = append(cs, "the 2-port term is ASSUMED: DESIGN.md §4's formula is an amd64 "+
			"observation and arm64 pipe counts vary by core (measured 4 on Cortex-X925, "+
			"2 on Cortex-A725), so a measured/formula divergence here has a cause "+
			"besides double-pumping (#171)")
	}
	if clock != "" {
		cs = append(cs, clock)
	}
	if len(cs) == 0 {
		return ""
	}
	return " -- " + strings.Join(cs, "; ")
}

// cpuMaxGHz is one CPU's cpuinfo_max_freq in GHz, or 0.
func cpuMaxGHz(c int) float64 {
	b, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_max_freq", c))
	if err != nil {
		return 0
	}
	khz, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	if err != nil || khz <= 0 {
		return 0
	}
	return khz / 1e6
}

// allowedCPUs parses /proc/self/status's Cpus_allowed_list ("19", "0-7",
// "0,8,16-18"). An empty result means the mask could not be read, which the
// caller labels rather than papers over.
func allowedCPUs() []int {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return nil
	}
	var list string
	for _, line := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(line, "Cpus_allowed_list:"); ok {
			list = strings.TrimSpace(rest)
			break
		}
	}
	return parseCPUList(list)
}

// parseCPUList parses the body of a Cpus_allowed_list line. Separate from the file
// read so it is testable on a host that has no /proc at all.
func parseCPUList(list string) []int {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(list, ",") {
		lo, hi, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			continue
		}
		b := a
		if isRange {
			if v, err := strconv.Atoi(strings.TrimSpace(hi)); err == nil {
				b = v
			}
		}
		for c := a; c <= b && c-a < 4096; c++ {
			out = append(out, c)
		}
	}
	return out
}

// TestAllowedCPUsParsesAMask covers the three shapes Linux writes into
// Cpus_allowed_list, plus the two ways it can be absent. It is a pure parser test
// so it runs on every platform, including the darwin dev host where the /sys and
// /proc reads around it return nothing at all — which is exactly why the parser is
// a separate function from the file read (#171).
func TestAllowedCPUsParsesAMask(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []int
	}{
		{"19", []int{19}},           // one pinned core, the judged shape
		{"0-3", []int{0, 1, 2, 3}},  // a range, as an unpinned 4-cpu guest reports
		{"0,8,16", []int{0, 8, 16}}, // the spread mask §5 rule 5 specifies
		{"0-1,19", []int{0, 1, 19}}, // ranges and singletons mixed
		{" 2 ", []int{2}},           // surrounding space
		{"", nil},                   // present but empty
		{"garbage", nil},            // unparseable, and NOT silently cpu0
	} {
		got := parseCPUList(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("parseCPUList(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseCPUList(%q) = %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}
