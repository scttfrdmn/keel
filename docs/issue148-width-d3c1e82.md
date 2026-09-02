<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #148: the scalar collapse is a step between width 1 and width 2

Eight arms on `janus.local` at `d3c1e82`, launched detached, 2026-09-02 17:04:25–19:12:49Z.
`KEEL_PIN_WIDTH` ∈ {8, 4, 2, 1} run twice, pass A in that order and pass B reversed. Every number
below is re-derivable:

    python3 archive/width148/analyze-width148.py d3c1e82 archive/width148

reproduces `archive/width148/analysis-width148.out` byte for byte, from either `archive/width148/`
or `build/` (checked). The driver is `archive/width148/driver-width148.sh`, **committed at `defd87d`
before the run**, so the two design choices that are not in the pre-registration are at a hash that
predates all of this data.

## Provenance

One binary for all eight arms — `sha256=d0d46d26c15cc8b2`, 5066553 bytes,
`-buildmode=exe -compiler=gc -trimpath=true`, `go1.27.0-X:simd` read off the artifact. It is
**byte-identical to `#147`'s traced binary**, so the `w8` and `w1` arms here are further draws of the
same artifact and the between-binary layout term (1.71/0.99/1.32%, `#54`/`#61`) is not in the
cross-run comparison at all. Intel Core i9-9960X, `governor=performance`, single NUMA node,
`GOMAXPROCS=1` in every arm, `-count=30 -benchtime=1s`, 20 rows × 30 samples × 8 arms = 4800 samples.
Masks read back from each log: `0,1,2,3,4,5,6,7` / `0,1,2,3` / `0,1` / `0`. `REMOTE_STATE=ok` and
600 rows on all eight. No OpenBLAS reference is available on janus, so every rate below is absolute,
against the `8.8 GFLOP/s` scalar and `281.6 GFLOP/s` avx512 *formula* cross-checks and not against a
measured peak.

The mask readback matters more here than in a gate run: `GOMAXPROCS` is 1 in every arm, so the width
is **not** recoverable from the row names, and that line is the only witness that `a4` was four cores
rather than eight.

## 1. The control, read first because it was declared first

If the twelve `avx512` rows moved across this sweep, the eight scalar rows would be a reading about
the host. Ratios of median `ns/op` to the same pass's `w8` arm, over all twelve rows:

| | w4/w8 | w2/w8 | w1/w8 |
|---|---|---|---|
| pass A | 0.995–1.005 | 0.999–1.050 | 0.990–1.054 |
| pass B | 0.972–1.002 | 0.998–1.001 | 0.984–1.030 |

Worst deviation from 1.000 anywhere: **0.054**. `#147` measured every `avx512` row within 0.97–1.10
of its width-8 median, so this sweep's control is *tighter* than the band it was declared against.
The scalar reading is about the scalar kernel.

## 2. The drift test the mirrored pass order buys

Within-pass ratios to that pass's own `w8` arm, then pass A's ratio divided by pass B's, over all 20
rows: median **1.000** (w4), **1.002** (w2), **1.003** (w1). A drift monotone in time would show up
here as a systematic departure from 1, because the two passes visit each width at mirrored positions
in the clock. There is none.

The ranges are wide — 0.865–1.150, 0.867–1.398, 0.868–1.154 — and the rows holding the ends are
named in `analysis-width148.out` §3c. All six extremes are scalar small-`kc` rows, and they are the
level structure of §6, not drift.

## 3. The registered criterion

> the small-`kc` rows read within 1.15× of their width-8 rate at width 4

Rate is the log's own `GFLOP/s` column, the column `#148`'s table published. **6 of the 8 registered
cells are within 1.15×**, with a seeded nonparametric CI on each ratio of medians (10000 resamples;
n=30 per arm and `#147` found within-window bimodality on these very rows, so a t-interval would be
a claim about a shape nobody measured):

| pass | row | w8 | w4 | w4/w8 | 95% CI | verdict |
|---|---|---|---|---|---|---|
| A | `2x32/kc=8` | 3.186 | 2.770 | 0.869 | [0.868, 0.871] | **outside** |
| A | `4x32/kc=8` | 2.764 | 2.771 | 1.003 | [1.001, 1.004] | within |
| A | `2x32/kc=32` | 3.654 | 3.651 | 0.999 | [0.996, 1.001] | within |
| A | `4x32/kc=32` | 3.067 | 3.548 | 1.157 | [1.154, 1.159] | **outside** |
| B | `2x32/kc=8` | 2.768 | 2.768 | 1.000 | [0.998, 1.001] | within |
| B | `4x32/kc=8` | 3.184 | 3.188 | 1.001 | [0.999, 1.003] | within |
| B | `2x32/kc=32` | 3.168 | 3.171 | 1.001 | [0.999, 1.002] | within |
| B | `4x32/kc=32` | 3.538 | 3.538 | 1.000 | [0.998, 1.002] | within |

Both failures are in pass A, both are the level structure of §6, and one of them misses the band by
**0.007**. The criterion is reported as measured, 6 of 8, and not restated.

What the criterion cannot do is more interesting than its score: the between-invocation level
structure is 5–15% wide, and the registered band is 15% wide. **At width 4 the criterion has
approximately no resolution** — it is comparing two draws of a quantity whose draw-to-draw spread is
the size of the band. That is a fact about the registered instrument, found by running it, and it is
not a reason to move the band.

## 4. The shape: neither registered branch

`#148` registered a dichotomy — "if the cliff is monotone in width it is contention or frequency; if
it is a step between 8 and everything else it is neither". The measured shape is a **third** one it
did not enumerate. Ratios of median `GFLOP/s` to the same pass's `w8` arm:

| row | a4/a8 | a2/a8 | a1/a8 | b4/b8 | b2/b8 | b1/b8 |
|---|---|---|---|---|---|---|
| `scalar 2x32/kc=8` | 0.869 | 0.868 | **0.258** | 1.000 | 1.152 | **0.298** |
| `scalar 4x32/kc=8` | 1.003 | 1.154 | **0.271** | 1.001 | 1.001 | **0.252** |
| `scalar 2x32/kc=32` | 0.999 | 0.825 | **0.232** | 1.001 | 1.153 | **0.268** |
| `scalar 4x32/kc=32` | 1.157 | 1.155 | **0.278** | 1.000 | 1.001 | **0.241** |
| `scalar 2x32/kc=128` | 1.035 | 0.948 | 0.977 | 1.001 | 0.984 | 0.947 |
| `scalar 4x32/kc=128` | 1.018 | 1.006 | 0.998 | 0.993 | 0.998 | 0.992 |
| `scalar 2x32/kc=512` | 1.005 | 0.954 | 0.994 | 1.006 | 1.001 | 0.994 |
| `scalar 4x32/kc=512` | 1.002 | 1.002 | 0.996 | 0.996 | 0.999 | 0.995 |

All twelve `avx512` rows read 0.949–1.029 in the same table.

**The collapse is entirely between width 1 and width 2.** All four registered rows fall to
0.232–0.298 of their width-8 rate — a 3.36–4.31× collapse — in **both** passes, 8 of 8 arm-pairs, and
show no partial degradation at width 2 or 4, where they sit at 0.825–1.157. Nothing else in the sweep
moves at any width.

## 5. What that rules out

- **A contention gradient in the mask width is refuted.** `GOMAXPROCS` is 1 in every arm, so widths
  2, 4 and 8 differ only in how many cores one thread may occupy — and they are indistinguishable.
  `#148`'s candidate 1 in its "eight masked threads facing eight free SMT siblings" form predicts a
  gradient across those three widths. There is none.
- **Frequency residency is refuted by sign.** Under `performance`, one busy core boosts *higher* than
  eight, not lower; the measurement has one busy core running 3.4–4.3× slower. The per-arm clock
  snapshots agree: `1200-3835` on `a1` is the *highest* ceiling of the eight arms, and every arm reads
  `1200-36xx`–`1200-38xx`. Those are snapshots and not sustained clocks, so they are corroboration of
  a sign, not a measurement of one.
- **A sustained userspace co-tenant is refuted by §7's load record** — and only by it.

What survives is a single family: **something specific to `cpu0` that the thread escapes at width ≥ 2
and cannot escape at width 1.** `#148`'s candidate 2 (interrupt affinity on `cpu0`) is in it; so is a
kernel-side form of candidate 1 (work on `cpu0`'s unmasked sibling `cpu16`). This harness still cannot
separate "width 1" from "`cpu0`" — the mask is a function of the width alone — so `#148`'s **decisive
test 2 is now the single blocking discriminator**, and it needs the `scripts/` change that test named.

A **fourth candidate** this run cannot exclude, and which was not in `#148`'s three: the Go runtime's
own background threads. At width 1, `sysmon`, the GC workers and the timer thread share the one
permitted core with the benchmark goroutine; at width ≥ 2 they do not. The one piece of evidence
bearing on it is an asymmetry that is hard to reconcile with it: `BenchmarkKernel` allocates its
panels inside the `b.Run` closure, so every row allocates once per execution and therefore at
approximately the same rate per second — yet the `avx512` rows lose at most 5.1% at width 1 while the
scalar small-`kc` rows lose 70–77%. Recorded as a candidate, not as a conclusion.

## 6. Unregistered: the small-`kc` scalar rows visit discrete levels per process invocation

Per-arm medians of the four registered rows, sorted (`analysis-width148.out` §3d):

| row | levels above 1.5 GFLOP/s |
|---|---|
| `2x32/kc=8` | 2.766, 2.768, 2.768, 2.770 \| 3.186, 3.190 |
| `4x32/kc=8` | 2.764, 2.771 \| 3.184, 3.187, 3.188, 3.189 |
| `2x32/kc=32` | 3.014 \| 3.168, 3.171 \| 3.651, 3.654, 3.654 |
| `4x32/kc=32` | 3.067 \| 3.538, 3.538, 3.541, 3.544, 3.548 |

The eight arm medians of a row do not spread over a continuum; they cluster into **two or three
groups a few thousandths wide, 5–15% apart**, and group membership tracks neither width nor pass
order. Both of §3's failures are one row landing in a different group in the `w4` arm than in the
`w8` arm. Two rows are enough to see that the groups are not shared structure across rows:
`2x32/kc=8` is in its low group on `a4` while `4x32/kc=8` is in its low group on `a8`, and on `b2`
both are high — so this is per-row and per-invocation, not one machine state that all rows read.

The obvious candidate is `#147`'s placement hypothesis at **process** granularity — same binary, so
no layout difference, but a fresh heap and fresh panel addresses per invocation. `#147` refuted
placement *within* a run at sample granularity; between invocations it is untested, and this sweep
cannot test it because `KEEL_ADDR_TRACE` was not set. That is a cheap fix for the next run of this
kind: the flag already exists.

## 7. The load record, and what it changed

Sixteen samples, before and after every arm, never during one. `/proc/loadavg` in order:

| boundary | loadavg | runnable/threads | UTC |
|---|---|---|---|
| before `a8` | 0.01 0.00 0.00 | 1/698 | 17:04:26 |
| after `a8` | 1.00 0.98 0.69 | 1/699 | 17:20:24 |
| after `a4` | 1.00 1.00 0.92 | 1/696 | 17:36:08 |
| after `a2` | **1.93 1.72 1.33** | **2/710** | 17:51:54 |
| after `a1` | **1.50 1.20 1.23** | 1/696 | 18:08:47 |
| after `b1` | 1.00 1.00 1.06 | 1/698 | 18:25:42 |
| after `b2` | 1.00 1.00 1.00 | 1/697 | 18:41:25 |
| after `b4` | 1.00 1.01 1.00 | 1/696 | 18:57:12 |
| after `b8` | 1.02 1.02 1.00 | 1/695 | 19:12:49 |

A load average of 1.00 is *this run's own thread*. **A co-tenant was present for roughly `a2` and
`a1`**: two runnable at 17:51:54, thread count up 696→710, five- and fifteen-minute averages lifted,
and gone by 18:25:42.

This is the difference between a publishable finding and `#147`'s undefendable one, and it cuts in
the direction the finding needed:

- `a1`'s window is 17:51:54–18:08:47 and the 15-minute average at its end is **1.23**, so the mean
  extra runnable load during `a1` was ≈0.23.
- `b1`'s window is 18:08:47–18:25:42 and the 15-minute average at its end is **1.06**, covering
  18:10:42–18:25:42 — nearly all of `b1`. The mean extra load during `b1` was ≈0.06.
- `b1` collapses by 0.241–0.298 and `a1` by 0.232–0.278. **The clean width-1 arm collapses as much as
  the contaminated one**, so the co-tenant is not the cause.
- `a2` ran inside the contaminated window at width 2 and shows **no** collapse, which is the same
  conclusion from the other side.

One defect in the instrument this run added, found by its own data: the samples also print the top
three processes by `ps -o pcpu`, and **that column is a lifetime average, not an instantaneous one**,
so it showed nothing above 1.1% in all sixteen samples while `/proc/loadavg` was reporting two
runnable threads. The `ps` line cannot see a co-tenant; the `loadavg` line can. The recommendation
this hands `#81` is therefore specific: the provenance block should carry `/proc/loadavg` **including
its runnable/threads field**, which is where the 2/710 was, and not a `%CPU` ranking.

## 8. What this sweep cannot see

- **One host, one revision, one 30-sample window per arm.** Two passes make time-ordering testable;
  they do not make eight arms into a population.
- **"Width 1" and "`cpu0`" are still one thing.** The mask is derived from the width, so every
  width-1 arm in this harness is `cpu0`. Nothing here distinguishes a property of one-core
  confinement from a property of that particular core.
- **No sustained clock was measured**, only per-arm snapshots and idle samples between arms. The
  frequency candidate is refuted by *sign*, not by a sustained reading.
- **No address trace.** The level structure of §6 is described and unattributed; the placement
  candidate for it is untested here.
- **Neither width-1 arm ran in a provably idle window** — `a1` at ≈0.23 extra runnable load and `b1`
  at ≈0.06. The argument in §7 is that the cleaner arm collapses as much, which is an argument from
  two arms, not a clean measurement of one.
- The 3.36–4.31× is a ratio of medians of a quantity that also has the §6 level structure in it;
  the collapse is 20× larger than that structure, which is why the two do not need separating here,
  and would if the effect were smaller.
