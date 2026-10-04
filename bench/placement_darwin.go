// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build darwin && cgo && darwinplacement

// The cgo half of #175's precondition: does darwin have an instrument that
// satisfies §5 rule 5's placement intent?
//
// # Why this probe exists
//
// Rule 5 establishes a stable clock and a pinned placement from whichever
// instrument the host has — the `performance` governor where cpufreq is readable,
// `BenchmarkPeak`'s head/middle/tail series in a virtualized guest, and a
// deterministic one-core-per-cache-domain affinity mask for placement. #138
// recorded that darwin supplies none of the three and that this costs nothing
// while no darwin reading is published. Scott has now approved publishing
// M-series rows, so the cost is no longer zero and the claim has to be tested
// rather than carried.
//
// It matters on this silicon specifically: the dev host is an M4 Pro with 8
// performance cores (16 MB L2) and 4 efficiency cores (4 MB L2). That is the same
// big.LITTLE confound that cost `archive/neon136` its attribution on the GB10,
// where the *vector* core-type ratio turned out to be 2.77x (#171) — on a machine
// whose `cpu_capacity` claimed 1.43x. An M-series row measured without core-type
// control is a draw over which cluster the scheduler happened to pick.
//
// # The two candidate levers, and why only one of them can answer rule 5
//
// `THREAD_AFFINITY_POLICY` is the lever #138 named, and the honest scoping is
// that **even a positive result would not satisfy rule 5's placement intent**.
// Apple documents it as an *L2-sharing hint*: it asks that threads sharing a tag
// be scheduled onto cores sharing an L2. Rule 5 needs something different — that
// this arm ran on the same *kind* of core as that arm — and a cache-affinity hint
// promises nothing about core type. So it is probed for what it claims and
// reported, not relied on. This is stated here rather than discovered later
// because a probe aimed at the wrong property would have produced a number and
// settled nothing.
//
// `pthread_set_qos_class_self_np` is the lever that can answer it. Apple
// documents QoS classes as controlling *which cluster* work runs on, with
// `QOS_CLASS_BACKGROUND` confined to the efficiency cores. If that holds, darwin
// has a core-type instrument — not an affinity mask, but something that satisfies
// the intent — and #138's law needs amending. If it does not hold, the law stands
// and is now tested rather than assumed.
//
// # Why a register-only kernel cannot test the affinity hint
//
// The workload driven under these levers is `BenchmarkPeak`'s register-only FMA
// saturation, chosen because its FLOPs/cycle is ISA-fixed, so its rate is a
// monotone function of effective clock x pipe count with every other term held —
// the same property rule 5 relies on in a guest. That makes it the right
// instrument for a *core-type* question and the WRONG one for an L2-sharing
// question: a loop with no memory operands cannot respond to a cache-placement
// hint, so running the affinity lever against it could not come out other than
// null. §5 rule 7. The affinity arm therefore gets a second, memory-resident
// workload, and its null result (if that is what it is) is reported only against
// the workload that could have shown something.
//
// # Why this file is behind a build tag
//
// Same reason as openblas.go: keel must never require cgo, so the binding is
// invisible to `go build`, to `go test` without the tag, and to the module graph.
// And Go does not allow cgo in _test.go files, so the binding lives here and
// placement_darwin_test.go holds the probe that calls it.
package bench

/*
#include <pthread.h>
#include <pthread/qos.h>
#include <mach/mach.h>
#include <mach/thread_policy.h>
#include <mach/mach_init.h>

// qos_apply sets this pthread's QoS class. Returns the errno-style result.
static int qos_apply(int cls) {
	return pthread_set_qos_class_self_np((qos_class_t)cls, 0);
}

// qos_read reports the QoS class actually in force on this thread, which is the
// read-back that makes the treatment witnessed rather than assumed (§5 rule 26).
static int qos_read(void) {
	qos_class_t c = QOS_CLASS_UNSPECIFIED;
	int rel = 0;
	if (pthread_get_qos_class_np(pthread_self(), &c, &rel) != 0) return -1;
	return (int)c;
}

// affinity_apply sets THREAD_AFFINITY_POLICY with the given tag on this mach
// thread. Returns the kern_return_t; KERN_SUCCESS is 0.
static int affinity_apply(int tag) {
	thread_affinity_policy_data_t p;
	p.affinity_tag = tag;
	return (int)thread_policy_set(mach_thread_self(), THREAD_AFFINITY_POLICY,
		(thread_policy_t)&p, THREAD_AFFINITY_POLICY_COUNT);
}
*/
import "C"

// QoS class values, copied from <pthread/qos.h> rather than recalled.
const (
	qosUserInteractive = C.QOS_CLASS_USER_INTERACTIVE
	qosUserInitiated   = C.QOS_CLASS_USER_INITIATED
	qosDefault         = C.QOS_CLASS_DEFAULT
	qosUtility         = C.QOS_CLASS_UTILITY
	qosBackground      = C.QOS_CLASS_BACKGROUND
	qosUnspecified     = C.QOS_CLASS_UNSPECIFIED
)

// setQoS applies a QoS class to the calling thread. The caller must hold
// runtime.LockOSThread, since the class is a property of the OS thread and a
// goroutine that migrates would carry the measurement away from the treatment.
func setQoS(cls int) int { return int(C.qos_apply(C.int(cls))) }

// readQoS reports the class in force, so a treatment can be witnessed.
func readQoS() int { return int(C.qos_read()) }

// setAffinityTag sets THREAD_AFFINITY_POLICY. 0 is the documented "null" tag,
// meaning no affinity.
func setAffinityTag(tag int) int { return int(C.affinity_apply(C.int(tag))) }

// qosName renders a class for the log, so a reading names its own treatment.
func qosName(c int) string {
	switch c {
	case qosUserInteractive:
		return "USER_INTERACTIVE"
	case qosUserInitiated:
		return "USER_INITIATED"
	case qosDefault:
		return "DEFAULT"
	case qosUtility:
		return "UTILITY"
	case qosBackground:
		return "BACKGROUND"
	case qosUnspecified:
		return "UNSPECIFIED"
	}
	return "unknown"
}
