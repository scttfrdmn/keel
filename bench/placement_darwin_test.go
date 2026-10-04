// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build darwin && cgo && darwinplacement

package bench

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/keel/internal/vec"
)

// PRE-REGISTERED before the probe was first run (#175, 2026-10-03).
//
// P1. `QOS_CLASS_BACKGROUND` lands the thread on efficiency cores, so the
// register-only peak rate falls MATERIALLY below the default-class rate.
// "Materially" is not a free parameter: the arms are separated by core type or they
// are not. The comparable separation this project has measured is 2.77x (GB10 X925
// against A725, #171), so the registered boundary is a fall of at least **25%** —
// far below any plausible cluster ratio and far above this instrument's spread,
// which is under 1% on repeated arms.
//
// P2. `QOS_CLASS_USER_INTERACTIVE` is at or above the default rate. Registered as
// an expected NULL, so that a large rise would be a finding about what the default
// class actually gets rather than a surprise.
//
// P3. The treatment arrives, or the probe FAILS rather than reports. Every QoS arm
// reads its class back off the thread. §5 rule 26: under a null-ish prediction an
// unapplied treatment produces exactly the predicted result, so corroboration and
// apparatus failure are the same reading unless the treatment is witnessed.
//
// WHAT WOULD FALSIFY THE PROBE ITSELF: P3 failing, or default and BACKGROUND
// agreeing within the instrument's spread. Either means darwin offers no core-type
// instrument and #138's law stands exactly as written.
//
// # THREAD_AFFINITY_POLICY is reported, not tested, and that is a scope decision
//
// #138 named that lever and forbade restating "it is inert on Apple silicon" as a
// finding. This probe does not restate it — and it also does not test it, for two
// stated reasons rather than by omission.
//
// First, **a positive result could not answer rule 5 anyway.** Apple documents the
// policy as an L2-sharing hint: it asks that threads sharing a tag be scheduled onto
// cores sharing an L2. Rule 5 needs core-TYPE stability — this arm ran on the same
// kind of core as that arm — and cache affinity promises nothing about that. So the
// hint is the wrong instrument for the question even when it works.
//
// Second, **testing it needs a design this probe does not have.** The policy
// co-locates MULTIPLE threads; a single-threaded arm cannot exhibit it at all, so
// running one and reporting a null would be a check that could not come out
// otherwise (§5 rule 7). A real test is ≥2 threads over a shared working set sized
// against the two clusters' different L2s, comparing same-tag against distinct-tag
// aggregate bandwidth.
//
// What IS reported is the one cheap fact available without that design: whether the
// kernel accepts the call at all. A refusal would be informative; acceptance says
// only that the call returned, which is stated at the line rather than inflated.
func TestDarwinPlacementLevers(t *testing.T) {
	t.Logf("keel-darwin-topology: %s", darwinTopology())

	peak, ok := peakArm()
	if !ok {
		t.Skip("no vector peak kernel in this build; the core-type instrument is unavailable")
	}
	t.Logf("keel-darwin-instrument: %s, register-only FMA saturation — FLOPs/cycle is "+
		"ISA-fixed, so the rate is monotone in clock x pipe count and a cluster change is visible",
		peak.Name)

	arms := []struct {
		name string
		qos  int
	}{
		{"default", qosUnspecified},
		{"qos=USER_INTERACTIVE", qosUserInteractive},
		{"qos=UTILITY", qosUtility},
		{"qos=BACKGROUND", qosBackground},
		{"default (repeat)", qosUnspecified},
	}

	rates := map[string]float64{}
	for _, a := range arms {
		rates[a.name] = runQoSArm(t, a.name, a.qos, func() float64 { return peakRate(peak) })
		t.Logf("  %-24s %8.2f GFLOP/s", a.name, rates[a.name])
	}

	// The repeat arm is the control that makes every delta below attributable: if
	// the two default readings disagree, the machine moved and nothing else here
	// separates treatment from drift.
	base, repeat := rates["default"], rates["default (repeat)"]
	drift := 100 * (repeat/base - 1)
	t.Logf("keel-darwin-control: default repeated %+0.3f%% — the spread every delta "+
		"below must clear to be a treatment rather than drift", drift)

	for _, a := range arms {
		if strings.HasPrefix(a.name, "default") {
			continue
		}
		t.Logf("  %-24s %+7.2f%% against default", a.name, 100*(rates[a.name]/base-1))
	}

	bg := 100 * (rates["qos=BACKGROUND"]/base - 1)
	if bg <= -25 {
		t.Logf("keel-darwin-verdict: P1 CONFIRMED (BACKGROUND %+.2f%%, boundary -25%%). QoS is a "+
			"core-type instrument on this silicon: darwin satisfies rule 5's placement INTENT "+
			"without an affinity mask, and #138's law needs amending", bg)
	} else {
		t.Logf("keel-darwin-verdict: P1 REFUTED (BACKGROUND %+.2f%%, boundary -25%%). No cluster "+
			"change observed, so #138's law stands as written", bg)
	}

	// The one affinity fact available without a multi-thread design. See the
	// function comment for why efficacy is out of scope rather than untested.
	for _, tag := range []int{1, 2} {
		rc := runAffinityAccept(tag)
		t.Logf("keel-darwin-affinity: thread_policy_set(THREAD_AFFINITY_POLICY, tag=%d) = %d (%s). "+
			"Acceptance only; efficacy needs >=2 threads over a shared working set, and a "+
			"positive result would still not answer rule 5 (cache affinity, not core type)",
			tag, rc, map[bool]string{true: "accepted", false: "REFUSED"}[rc == 0])
	}
}

// runQoSArm locks an OS thread, applies the class, witnesses it, and measures.
//
// LockOSThread is not optional: QoS is a property of the OS thread, so a goroutine
// that migrated would carry the measurement away from the treatment and the arm
// would measure the default class while claiming otherwise.
func runQoSArm(t *testing.T, name string, qos int, f func() float64) float64 {
	t.Helper()
	out := make(chan float64, 1)
	go func() {
		// LockOSThread WITHOUT a matching Unlock, deliberately. Go terminates an
		// OS thread whose goroutine exits while still locked, and that is the only
		// way to guarantee this arm's QoS class cannot reach another arm.
		//
		// The first run of this probe proved the need rather than assuming it. With
		// Unlock paired, Go RECYCLED the thread that had been demoted to BACKGROUND
		// and the `default (repeat)` control read 30.22 GFLOP/s against the first
		// default's 114.53 -- a -73.6% "drift" that was the treatment persisting past
		// the arm that set it. A QoS class is a property of the OS thread and nothing
		// resets it on reuse, so an arm that returns its thread to the pool poisons
		// whatever runs there next. That is why the repeat arm is in the table: it is
		// the control that makes every delta attributable, and it caught this.
		runtime.LockOSThread()
		if qos != qosUnspecified {
			if rc := setQoS(qos); rc != 0 {
				t.Errorf("%s: pthread_set_qos_class_self_np = %d; the treatment did not apply", name, rc)
				out <- 0
				return // still locked: this thread dies with the goroutine
			}
			if got := readQoS(); got != qos {
				t.Errorf("%s: QoS read-back = %s, want %s. An unapplied treatment produces "+
					"exactly the predicted null, so this fails rather than reports (§5 rule 26)",
					name, qosName(got), qosName(qos))
				out <- 0
				return
			}
		}
		f() // warm, discarded: the first call pays page faults and clock ramp
		out <- f()
	}()
	return <-out
}

// runAffinityAccept reports whether the kernel accepts the policy call, on a
// locked thread so the call lands where it is read.
func runAffinityAccept(tag int) int {
	out := make(chan int, 1)
	go func() {
		runtime.LockOSThread() // unpaired, as above: the thread dies with the goroutine
		out <- setAffinityTag(tag)
	}()
	return <-out
}

// peakRate measures one peak kernel's GFLOP/s over a fixed iteration budget,
// checking the accumulator-chain witness first for the same reason BenchmarkPeak
// does: a collapsed kernel reports a plausible number that is not a ceiling.
func peakRate(k vec.PeakKernel) float64 {
	if got, want := k.Run(1000), k.Witness(1000); got != want {
		panic(fmt.Sprintf("%s: witness Run(1000) = %v, want %v — chains did not survive "+
			"compilation, so this is not a ceiling", k.Name, got, want))
	}
	const iters = 1 << 16
	const reps = 64
	start := time.Now()
	var sink float32
	for i := 0; i < reps; i++ {
		sink = k.Run(iters)
	}
	el := time.Since(start).Seconds()
	_ = sink
	return float64(k.FlopsPerIter) * float64(iters) * float64(reps) / el / 1e9
}

// peakArm picks the widest vector peak kernel: the core-type instrument.
func peakArm() (vec.PeakKernel, bool) {
	for _, k := range vec.PeakKernels() {
		if k.Name != "scalar" {
			return k, true
		}
	}
	return vec.PeakKernel{}, false
}

// darwinTopology reports the P/E split, so every reading names the machine whose
// clusters it is distinguishing.
func darwinTopology() string {
	get := func(k string) string {
		out, err := exec.Command("sysctl", "-n", k).Output()
		if err != nil {
			return "?"
		}
		return strings.TrimSpace(string(out))
	}
	return fmt.Sprintf("%s nperflevels=%s P=%s(L2 %s) E=%s(L2 %s)",
		get("machdep.cpu.brand_string"), get("hw.nperflevels"),
		get("hw.perflevel0.physicalcpu"), get("hw.perflevel0.l2cachesize"),
		get("hw.perflevel1.physicalcpu"), get("hw.perflevel1.l2cachesize"))
}
