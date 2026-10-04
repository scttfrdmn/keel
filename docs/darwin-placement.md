# darwin placement: does the M-series have an instrument for §5 rule 5?

**Probe, 2026-10-03, on the dev host. #175's precondition, #138's part 1 re-opened by
measurement.** Driver: `bench/placement_darwin.go` + `bench/placement_darwin_test.go`,
behind `//go:build darwin && cgo && darwinplacement` so nothing keel ships links it.
Re-run with:

```
GOEXPERIMENT=simd go test -tags darwinplacement ./bench/ -run TestDarwinPlacementLevers -v
```

## Why it was run

Scott approved publishing M-series rows. That removed the ground on which #138's part 1
rested: the law *"darwin has neither of rule 5's instruments, and it costs nothing
because no darwin reading is published"* was true in its second half and the first half
had never been tested. A row about to be published needs the first half tested.

It matters on this silicon in particular. The dev host is an **Apple M4 Pro: 8
performance cores (16 MB L2) + 4 efficiency cores (4 MB L2)**. That is the same
big.LITTLE confound that cost `archive/neon136` its attribution on the GB10 — where the
*vector* core-type ratio was 2.77× on a machine whose `cpu_capacity` claimed 1.43× (#171).
An M-series row measured without core-type control is a draw over which cluster the
scheduler happened to pick.

## Instrument

`BenchmarkPeak`'s register-only NEON FMA saturation. Its FLOPs/cycle is ISA-fixed, so
the rate is a monotone function of effective clock × pipe count with every other term
held — the same property rule 5 relies on in a virtualized guest. That makes it the
right instrument for a **core-type** question.

It is the wrong instrument for a cache question, and that is why the affinity lever is
not tested with it: a loop with no memory operands cannot respond to a cache-placement
hint, so the result could not have come out otherwise (§5 rule 7).

## Results

| arm | rate | vs default |
|---|---|---|
| default (inherited class) | **114.68 GFLOP/s** | — |
| `qos=USER_INTERACTIVE` | 114.59 | −0.09% |
| `qos=UTILITY` | 113.14 | −1.35% |
| `qos=BACKGROUND` | **26.07** | **−77.27%** |
| default (repeat) — **the control** | 114.42 | −0.234% |

**P1 CONFIRMED** against its registered boundary of −25%: `QOS_CLASS_BACKGROUND` moves
the rate by −77.27%, a **4.399× cluster separation**, against a control drift of 0.234%.
So **QoS is a core-type instrument on this silicon.**

**P2 confirmed as the registered null**: `USER_INTERACTIVE` is −0.09%, inside the control
drift, so a CLI process already runs at the top of the class range. `UTILITY` at −1.35%
clears the drift and is a real but small effect.

**`THREAD_AFFINITY_POLICY` is REFUSED**, not inert: `thread_policy_set` returns **46 =
`KERN_NOT_SUPPORTED`** (verified against `mach/kern_return.h` in the SDK). #138 forbade
restating "the hint is believed inert on Apple silicon" as a finding; this does not
restate it. A kernel refusal is a stronger and different fact, and it needs no
multi-thread design to establish — a call the kernel declines cannot be doing anything.

## The probe contaminated itself, and the control is what caught it

The first run read `default (repeat)` at **30.22** against the first default's **114.53** —
a −73.6% "drift". The cause was the probe, not the machine: QoS is a property of the OS
thread, Go recycles OS threads, and nothing resets the class on reuse. The repeat arm
landed on the thread that had been demoted to `BACKGROUND` and inherited it.

The fix is `runtime.LockOSThread` **without** a matching `Unlock`, so Go terminates the
thread when the arm's goroutine exits and the class cannot outlive the arm. After it, the
control reads −0.234%.

This is why the repeat arm is in the table at all. A treatment that persists past the arm
that set it produces a clean-looking number for the *next* arm, and no amount of
precision in the treated arms would have revealed it.

## The amended law for darwin readings

Superseding what landed in `bench_test.go` hours earlier:

- **No governor.** darwin has no `cpufreq`. Rule 5's *guest* substitute — `BenchmarkPeak`
  sampled head/middle/tail — needs no `cpufreq` and does work here.
- **No affinity mask**, measured: `KERN_NOT_SUPPORTED`.
- **A core-type lever does exist**, measured at 4.399×. So rule 5's placement **intent**
  is satisfiable on darwin even though its **mechanism** is not.
- **It is reachable only through cgo.** keel's shipped bench is cgo-free, so a reading
  there runs at whatever class the invoking process carried, and that is *unstated*
  rather than controlled. Default and `USER_INTERACTIVE` agree to 0.09%, so the exposure
  is a demoted parent process rather than ordinary variation.
- **What no lever gives: pinning to a core.** QoS selects a cluster, not a core, so
  within-cluster migration remains — and on this part the P-cluster is 8 cores sharing
  one 16 MB L2, so a migrating single-thread row stays L2-warm. That is a weaker promise
  than the Linux fleet's one-core-per-domain mask and it is stated rather than netted out.

**Consequence for #175:** a published M-series row states its QoS class, or states that
it inherited one. The honest default is to measure judged M-series rows through the
tagged path at a fixed, stated class.

## What this does not establish (§5 rule 12)

- **One machine, one part.** M4 Pro only. Whether the ratio or the lever behaves the same
  on an M1/M2/M3 or an M4 Max is unmeasured.
- **Whether the affinity hint would work if accepted** is unanswerable from here and
  deliberately not pursued: it is cache affinity, not core type, so a positive result
  would not satisfy rule 5's intent even then.
- **Single-threaded throughout.** Nothing here bears on how QoS behaves for the parallel
  nest, where 8 P-cores and 4 E-cores would be contending.
- **The 4.399× is one draw per arm**, not a multi-draw estimator. It is three orders above
  the control drift, so the *direction* is not in question; the figure itself is not a
  published reference and is not used as one.
