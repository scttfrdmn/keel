<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#148` test 3: the single-core collapse is async preemption, single-factor

Test 2 established that the collapse is not cpu0 and is not graded (`docs/issue148-core-97a21f4.md`).
Test 3 asks which runtime mechanism it *is*, by removing candidates one at a time from a
one-core arm and asking whether the recovery comes back.

**Result: it is async preemption, and turning it off alone recovers 98.8–99.7% of the
two-core control.** The prediction registered before the run said the opposite on eight of
twelve cells, and those eight are refuted.

## 1. Provenance

| | |
|---|---|
| rev | `205a7a8dddfb412ae04521ed1dde34e115031eb7`, tree clean at launch and throughout |
| host | `janus.local`, `governor=performance`, up 42 days, no interactive users |
| driver host | `terror` (darwin/arm64), detached under tmux as `keel-mech148-205a7a8` |
| toolchain | `go1.27.0`, read **off the artifact** (`builder toolchain go1.27.0-X:simd`), not off the driver's shell |
| binary | ONE binary for all ten arms, `sha256=d0d46d26c15cc8b2`, `-trimpath`, byte-identical to the binary the classifier boundaries were derived on |
| samples | 5 arms × 2 passes × 30 = 300 benchmark invocations, 60 per (row, arm) cell |
| window | 2026-09-03T17:44:58Z → 20:20:49Z (2h36m), `=== done ===`, status=0 |
| logs | `archive/mech148/bench-mech148-205a7a8-{a,b}{ref,c0,gc,pre,both}.txt`, driver log `archive/mech148/mech148-205a7a8.log` |
| scorer | `archive/mech148/analyze-mech148.py`, output recorded verbatim in `archive/mech148/analysis-mech148.out` |

No OpenBLAS reference: these are scalar-path rows measured to compare *arms of the same
binary against each other*, so every number below is a ratio between two arms and its
denominator is named at the point of use. Absolute GFLOP/s appear only as levels.

## 2. The arms, and what makes them mean something

| arm | mask | cores | env | role |
|---|---|---|---|---|
| `ref` | `0,1` | 2 | `GOMAXPROCS=1` | recovered control |
| `c0` | `0` | 1 | `GOMAXPROCS=1` | collapse positive control |
| `gc` | `0` | 1 | `GOMAXPROCS=1 GOGC=off` | removes GC assist and worker activity |
| `pre` | `0` | 1 | `GOMAXPROCS=1 GODEBUG=asyncpreemptoff=1` | removes async preemption signals |
| `both` | `0` | 1 | `GOMAXPROCS=1 GOGC=off GODEBUG=asyncpreemptoff=1` | the joint arm |

Pass a runs them in that order, pass b in reverse; the reversal is what makes a time-ordered
drift measurable rather than invisible, and §6 is that reversal earning its keep.

Three readbacks per arm, all ten green:

- `gomaxprocs readback OK: 1` — the canary for the unquoted word-split at `scripts/remote.sh:1687`.
  It leads every env string, because a canary in second place is not one.
- `cores readback OK: N distinct physical core(s)` — `ref`'s two-ness is *derived* from
  `siblings0=0,16`, `siblings1=1,17` at preflight, not asserted.
- `keel-pin:` line agreeing with the registered mask.

And one treatment witness, because rule 26 requires that a treatment prove it arrived when the
registered prediction is the null: `GOGC=off` moved the heap goal from 4 MB to 8532210231539 MB,
which is the runtime saying in its own output that it honoured the variable.

## 3. The twelve cells against their `(60,0,0)`s

`mode()` classifies each sample against its own row's pooled `c0` median: **C**onfined at
≤1.15×, **R**ecovered at >1.50×, **M**iddle between. The registered prediction for all twelve
cells was `(60,0,0)` — `unimodal-at-confined` — i.e. *no environment variable recovers anything*.

| row | arm | predicted | observed | shape | verdict |
|---|---|---|---|---|---|
| 2x32/kc=8 | `gc` | (60,0,0) | (60,0,0) | unimodal-at-confined | **CONFIRMED**, OOD for GC |
| 2x32/kc=8 | `pre` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED** |
| 2x32/kc=8 | `both` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED**, OOD for GC |
| 4x32/kc=8 | `gc` | (60,0,0) | (60,0,0) | unimodal-at-confined | **CONFIRMED**, OOD for GC |
| 4x32/kc=8 | `pre` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED** |
| 4x32/kc=8 | `both` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED**, OOD for GC |
| 2x32/kc=32 | `gc` | (60,0,0) | (60,0,0) | unimodal-at-confined | **CONFIRMED**, OOD for GC |
| 2x32/kc=32 | `pre` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED** |
| 2x32/kc=32 | `both` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED**, OOD for GC |
| 4x32/kc=32 | `gc` | (60,0,0) | (60,0,0) | unimodal-at-confined | **CONFIRMED**, OOD for GC |
| 4x32/kc=32 | `pre` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED** |
| 4x32/kc=32 | `both` | (60,0,0) | (0,0,60) | unimodal-at-recovered | **REFUTED**, OOD for GC |

`CONFIRMED=4, REFUTED=8`. "Unimodal" here means 60/60 — there is no band to relitigate, which is
why the prediction was registered as a count triple in the instrument's own output space and not
as a ratio of medians.

Preconditions, which are not scored but without which none of the above is interpretable:
`ref` ≥59/60 RECOVERED on every row (**60/60** on all four) and `c0` 60/60 CONFINED on every row
(**60/60** on all four). `ALL MET`.

## 4. Attribution: single-factor

The registration's falsifier reads "≥1/60 RECOVERED in any treatment arm on any registered row
refutes the position above, and WHICH arm attributes it — `gc` alone or `pre` alone is
single-factor, `both` alone is joint." It fired 60/60 in `pre` on every row, and `pre` is a
single-factor arm. It fired independently in each pass: 120/120 above the RECOVERED cut in
`apre`, 120/120 in `bpre`, 120/120 in `aboth`, 120/120 in `bboth`.

How complete is the recovery? Computed between arms that share the same host-state level (§6 is
why that qualifier is load-bearing — all four arms below are pass b):

| row | `bc0` | `bpre` | `bref` | `bpre/bc0` | `bpre/bref` |
|---|---|---|---|---|---|
| 2x32/kc=8 | 0.821 | 2.757 | 2.769 | 3.358 | **0.996** |
| 4x32/kc=8 | 0.805 | 2.744 | 2.765 | 3.408 | **0.992** |
| 2x32/kc=32 | 0.849 | 3.160 | 3.170 | 3.722 | **0.997** |
| 4x32/kc=32 | 0.854 | 3.045 | 3.080 | 3.566 | **0.988** |

So `asyncpreemptoff=1` on **one** core reaches 98.8–99.7% of what a **second physical core**
buys. There is no material residue for a second mechanism to explain.

`both` adds nothing over `pre`: `bboth/bpre` = 1.0002, 0.9993, 0.9979, 0.9992. Since `pre` alone
already recovers ~99% of the gap, a joint arm has nothing left to contribute, and it does not
contribute it.

### It is a duty cycle, not a per-call cost: ~71% of the cpu

"3.4× slower" understates what the shape of the data says. A fixed cost paid once per call would
show as a constant **delta** in ns/op; a mechanism consuming a constant *fraction of the cpu* shows
as a constant **ratio**. Across rows spanning **7.2×** in call duration:

| row | `bpre` ns/op | `bc0` ns/op | delta | ratio | cpu lost |
|---|---|---|---|---|---|
| 2x32/kc=8 | 371.4 | 1247.5 | 876.1 | 3.359 | 70.2% |
| 4x32/kc=8 | 746.4 | 2544.0 | 1797.6 | 3.408 | 70.7% |
| 2x32/kc=32 | 1296.0 | 4823.5 | 3527.5 | 3.722 | 73.1% |
| 4x32/kc=32 | 2691.0 | 9593.5 | 6902.5 | 3.565 | 71.9% |

The delta spans **7.88×** and the ratio spans **1.108×**. So it is the ratio that is the invariant,
and the statement that survives is rate-independent:

> **Async preemption consumes 70.2%–73.1% of a Go process confined to one cpu**, on loops with no
> call sites in the hot path.

That form needs no assumption about how often anything fires, which is why it is the headline rather
than the 3.4×.

### The rate × cost decomposition is *not* measured, and the arithmetic constrains it

§4's attribution names *what* is removed. It does not establish the rate at which it fires, and
this report should not be read as having measured that. The duty cycle above puts a hard constraint
on the pair:

| assumed rate | required cpu occupancy per event |
|---|---|
| ~100/s — sysmon's nominal retake (`forcePreemptNS = 10ms`) | **7130 µs** — impossible |
| ~50 000/s — sysmon's *fastest* poll (`delay = 20µs`) | **14.3 µs** — plausible |

A bare Linux signal delivery, handler and `sigreturn` is O(1–5 µs). So only the fast-poll branch is
arithmetically open, and even it requires each preemption to occupy ~3–14× a bare signal's cost —
consistent with each SIGURG on a *single* cpu costing a full deschedule/reschedule round trip rather
than a handler, but **not measured here**. Two unmeasured multipliers stand between the flag and the
number, and naming them is the honest form. Filed as `#151`.

### What `asyncpreemptoff=1` actually removes

Read from the runtime source, not from memory. The flag gates exactly one call: `preemptM(mp)`
is skipped (`runtime/proc.go:6940`, `runtime/preempt.go:224`). sysmon still runs, still retakes
Ps, still sets `gp.preempt` and `stackguard0 = stackPreempt`. So the removed cost is **SIGURG
delivery and handling**, not sysmon's existence and not the marking. That is a much narrower
claim than "preemption is expensive," and it is the one the measurement supports.

`runtime/extern.go:227` notes the flag also disables conservative stack scanning. That is a
rival explanation for the recovery — and it is closed off by the treatment witness in §2: there
were **0 heap-triggered (non-forced) GC cycles** with GC on, so there was no stack scanning to
make conservative or otherwise. The rival requires a GC that never ran.

## 5. The four `gc` cells are out of domain, and say less than they appear to

`gc` CONFIRMED on all four rows. That is not evidence about GC, and the scorer labels every
`gc` and `both` cell `OOD for GC` on arm membership — attached to the arm's *reach*, which is
where a property of reach belongs, and not to the cell's outcome.

The driver's own witness says why: **0 heap-triggered cycles with GC on**. `GOGC=off` arrived
(the heap goal moved by twelve orders of magnitude) and had nothing to remove. A `(60,0,0)` from
an arm whose treatment had no work to do is a statement about the confined level, not about the
garbage collector. `gc` sits at 0.999×–1.002× of `c0` on all four rows, exactly as the
registration's `OUT_OF_DOMAIN` item 5 predicted before the run.

The margin caveat belongs here rather than in a footnote. The eight REFUTED cells clear their
boundary by 120–147% — three orders of the largest confound in §6. The four CONFIRMED cells
clear theirs by only **12.1–12.6%**, which is the *same size* as that confound. They are
confirmed as observed, and neither `agc` nor `bgc` shows any trace of the elevation; but had the
§6 term landed on a `gc` arm, its worst sample would have crossed the CONFINED cut and the cell
would have read `M`. Stated as a conditional, not a retraction.

## 6. A between-arm level term of 13–16%, unattributed, and why it does not touch the verdict

Filed as `#150`; it is the complement of `#147`, which found bimodality *within* a window.

Pass b's reversal earned its keep: the `pre` and `ref` arms **do not reproduce across passes.**

Every arm's own 30 samples are tight — 0.79% to 1.86% spread about its median (one exception,
`bref` on 2x32/kc=32 at 3.82%, driven by a single high sample). So this is not a bimodal arm;
it is a level that differs *between* arms and is stable *within* each. Two distinct patterns,
against the floor shared by arms 6, 7, 9 and 10:

| row | floor | `aref`/floor | `apre`/floor |
|---|---|---|---|
| 2x32/kc=8 | 2.757 | 1.006 | 1.126 |
| 4x32/kc=8 | 2.744 | **1.163** | 1.128 |
| 2x32/kc=32 | 3.158 | 1.005 | 1.126 |
| 4x32/kc=32 | 3.044 | **1.164** | 1.127 |

`apre` is elevated by 1.126–1.128 on **all four** rows, a spread of 0.002 — a uniform rate
factor. `aref` is elevated by 1.163/1.164 on the two **4x32** rows and 1.005/1.006 on the two
2x32 rows — row-selective. Two shapes, so at least two things, and this report attributes
neither.

**What it is not:** the treatment. `apre` and `bpre` have byte-identical env and mask and sit
12.6% apart; `aref` and `bref` likewise, 15.4% apart on 4x32. A term that varies between two
arms configured identically is host state, not a property of the variable under test. That is
the whole reason the §4 table is computed inside pass b only: pooling `apre` with `bpre` would
produce a median that is a mixing fraction of two tight modes, and a ratio built on it would be
a number with no referent.

**Why the verdicts stand anyway:** the recovery this test measures is 3.36×–3.72×. The confound
is 1.13×–1.16×. They differ by a factor of ~3, and every REFUTED cell's worst single sample
clears its boundary by 120–147%. The §5 conditional is the one place the margins are comparable.

**What would settle it,** and it is already an open gap: `#81` notes that `freq_khz` is sampled
only *between* arms, never during one. Every frequency reading in the driver log is therefore an
idle reading. A uniform 12.7% speedup across four unrelated rows is what a clock change looks
like, and the apparatus currently cannot see the clock while the work runs. Nothing else in the
provenance block discriminates.

## 7. Controls

- **The classifier reproduces its own derivation.** Before scoring anything, the scorer replays
  test 2's published shape table out of `archive/core148/`'s tracked logs: **16 of 16 arm-rows
  reproduce exactly**. This asks whether the *instrument* can see a signal known to be present
  (rule 7) — a different question from whether this run's treatments arrived (rule 26), which is
  what §2's witnesses answer.
- **Admissibility comes from the driver log and nowhere else.** An arm log exists on disk
  whether or not its arm finished, so asking the samples whether they are trustworthy asks the
  wrong witness. One regex branch per driver phrase, failing closed to `INCOMPLETE` on a label
  matching none. All ten arms `measured`.
- **The scorer imports the registration rather than reimplementing it**, so the boundaries have
  one source of truth and cannot drift from what was registered.
- **One binary, hash-checked against the boundaries' binary.** A recompile between arms would
  make every cross-arm ratio here a comparison of two programs.
- **Reciprocal columns agree.** `ns/op` and `GFLOP/s` are printed independently to 4 significant
  figures and quantize independently; worst disagreement **0.052%** over all 2400 scalar rows.
- **The quietness gate was driven red on purpose** before the run, including the branches no
  healthy run reaches (`archive/mech148/test-driver-mech148.sh`).

## 8. Disclosure: the quietness guard's headroom is position-dependent

Required by `#149`'s ruling, so that a *refused* arm in a future run reads correctly.

The guard refuses an arm whose 5-minute load exceeds `QUIET_L5_MAX=1.25`. A single thread pinned
to a single cpu for ~16 minutes keeps one task runnable, which contributes ~1.0 to the 5-minute
average and decays over the following five minutes. The gate samples immediately after the
previous arm ends. So:

| arm | 5-minute load at its gate | headroom under 1.25 |
|---|---|---|
| `aref` (first arm of a pass) | 0.17 | **≈1.08** |
| every arm after it | 1.00–1.01 | **≈0.24** |

That ~1.0 is this driver's own preceding arm, not a co-tenant. The bound is not miscalibrated —
it was derived on samples taken on the same pedestal, and the one tracked contaminated sample it
caught (2.17) is pedestal plus ~1.2 of foreign work. The defect is that the pedestal is
*implicit*, which makes the guard hair-triggered for arms 2–10.

All ten arms passed the gate in this run, so nothing here was refused and no measurement is
affected. The error direction is toward *false refusal*, which is recoverable; a quiet null is
not. `#149` carries the fix for the next registration — a derived pedestal, deduplicated **by
instant rather than by read**, since the driver samples the same between-arm instant twice.

## 9. What would settle what remains

1. **The §6 level term.** Sample `freq_khz` *during* an arm, not only between arms (`#81`). If a
   uniform 12.7% tracks the clock, the term is explained and the `gc` cells' 12% margin stops
   being a caveat. No new apparatus, one added sampler.
2. **The rate × cost decomposition** (`#151`). Not corroboration of a settled claim: the duty
   cycle is measured, the decomposition is not, and the arithmetic in §4 rules out sysmon's
   *nominal* rate outright. A `perf` count of SIGURG deliveries on `c0` versus `pre`, divided into
   the 71%, yields per-event occupancy directly and decides between "sysmon polls far faster than
   its retake period suggests" and "each preemption costs a full deschedule/reschedule."
3. **Why `aref`'s elevation is row-selective.** It appears on 4x32 and not 2x32, which is a
   shape dependence a pure clock story does not predict. Unexplained; may resolve with (1).

## 10. What `#148` now knows

- The single-core collapse is **async preemption**, single-factor, and removing it on one core
  recovers 98.8–99.7% of a second physical core.
- Its magnitude is best stated as a **duty cycle: 70.2%–73.1% of the cpu**, because the *ratio* is
  invariant (spans 1.108×) across rows whose call duration spans 7.2× while the per-call delta
  spans 7.88×. That form assumes no rate.
- The mechanism is narrower than "preemption": sysmon still runs and still marks, so what is
  removed is **SIGURG delivery and handling**. But the **rate × cost decomposition is unmeasured**
  — sysmon's nominal 10ms retake is arithmetically impossible (it would need 7130 µs per event) and
  only the fast-poll branch is open, at ~14.3 µs per event against O(1–5 µs) for a bare signal.
  `#151`.
- **GC is not involved and this campaign cannot test it here** — 0 heap-triggered cycles means
  `GOGC=off` has nothing to remove. Four cells characterize the confined level and say nothing
  about the collector.
- The pre-registered prediction was **wrong on eight of twelve cells**, and it was registered
  before the run in the instrument's own output space, which is why that is a result rather than
  an argument.
- An unattributed **13–16% between-arm level term** exists on this host, is not caused by any
  treatment, and is invisible to the current provenance sampling. It does not reach any verdict
  in §3, and it is the reason no level in this report is pooled across passes.
