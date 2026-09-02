<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#148` test 2: it is not cpu0, and the recovery is not graded

Evidence: `archive/core148/`. Driver `driver-core148.sh`, bands `predictions-core148.py`, analyzer
`analyze-core148.py`, its fixtures `test-analyze-core148.py`, eight arm logs, the driver transcript,
and `analysis-core148.out` regenerated from the archived logs.

Pre-registered on `#148` before any data existed
([comment 5515657242](https://github.com/scttfrdmn/keel/issues/148#issuecomment-5515657242)), design
committed at `4dd6334`, run at `97a21f4`. The analyzer **imports** the bands from
`predictions-core148.py` rather than restating them, and prints that module's commit hash in its own
section 0, so what was applied is checkable against what was committed.

Two results are clean. A third — the one that looked like the interesting finding — dissolved when the
per-sample distributions were read instead of the medians, and §5 is that correction.

## 1. Provenance

- Host janus.local, 32 logical cpus (`online=0-31`), `performance` governor at all 16 samples.
- One binary for all eight arms, `sha256=d0d46d26c15cc8b2` — **byte-identical to `#147`'s and test
  1's** — `bytes=5066553`, `flags=[-buildmode=exe -compiler=gc -trimpath=true]`, toolchain
  `go1.27.0-X:simd` read off `bench.test` rather than off the driver's shell. So `ref` is a further
  draw of test 1's own w2 arm and no between-binary layout term applies.
- `GOMAXPROCS=1` in every arm, confirmed from each log's own `keel-bench-gomaxprocs: 1`.
- 20 rows × 30 samples × 8 arms, 0 unparsed lines, all eight `cores=` readbacks matching their arm.
- **Load**, sampled before and after every arm and never during one — 16 samples. The **runnable
  numerator is 1 in all 16** (`1/693` through `1/703`), which is the co-tenant check and it passes.
  The 1-minute figure reads `0.00` before the first arm and `0.92`–`1.05` at every sample thereafter,
  i.e. it tracks keel's own single thread. **The final sample reads `0.99 2.17 1.83`** — 5- and
  15-minute averages that one runnable thread cannot produce, in a window that overlaps the last arm.
  That is unexplained, it is not smoothed away here, and §6 measures whether it reached the numbers.
- No OpenBLAS reference was taken. Every figure below is a within-run ratio between keel arms, so no
  percent-of-peak is claimed and no denominator is implied beyond `ref`.

## 2. The arms, and the readback that makes them mean something

| arm | `KEEL_PIN_CPUS` | `cores=` readback | distinct physical cores |
|---|---|---|---|
| `ref` | `0,1` | `0,1` | 2 |
| `c0` | `0` | `0` | 1 |
| `c5` | `5` | `5` | 1 |
| `smt` | `0,16` | `0,0` | **1** |

`siblings0=0,16` was read from `thread_siblings_list` on the host, never written in the driver. The
preflight would have refused four ways — no sibling for cpu0, cpu0 and cpu1 siblings of one core,
cpu5 sharing cpu0's core, unreadable topology — and cleared all four. `smt` reporting `cores=0,0` is
the whole point: two logical cpus, **one** physical core.

## 3. Branch A is refuted, and not by a band

cpu0-specific interference — interrupt affinity, kernel work on cpu0 — predicted `c5` intact. `c5` is
instead **indistinguishable from `c0`**, pooled medians over both passes, GFLOP/s:

| row | `c0` | `c5` | `c5/c0` |
|---|---:|---:|---:|
| `2x32/kc=8` | 0.8243 | 0.8198 | **0.9946** |
| `4x32/kc=8` | 0.8054 | 0.8029 | **0.9968** |
| `2x32/kc=32` | 0.8494 | 0.8484 | **0.9988** |
| `4x32/kc=32` | 0.8521 | 0.8505 | **0.9980** |

Within 0.6% on every row, and in each pass independently (`c5`/`c0` per-pass ratios differ by at most
0.001). The registered prediction for `c5` was intact, `[0.85, 1.15]`; the observed ratios to `ref`
are 0.295 / 0.252 / 0.267 / 0.240, scoring **0 of 4**. A mechanism that does not care *which* core it
is cannot be about cpu0. This is the kill the test was built for and it is the unambiguous result.

## 4. The registered criterion

Ratio of an arm's estimator to the same run's `ref`, primary (pooled median of both passes) /
secondary (max of the two pass medians). Collapse ≤ 0.40, intact 0.85–1.15, both registered before
the run; the gap between them is INDETERMINATE by design.

| row | `c0` | `c5` | `smt` | `smt` verdict |
|---|---:|---:|---:|---|
| `2x32/kc=8` | 0.297 / 0.258 | 0.295 / 0.258 | **0.986 / 0.857** | **intact** |
| `4x32/kc=8` | 0.252 / 0.252 | 0.252 / 0.252 | 0.334 / 0.343 | collapse |
| `2x32/kc=32` | 0.267 / 0.232 | 0.267 / 0.232 | 0.275 / 0.240 | collapse |
| `4x32/kc=32` | 0.241 / 0.240 | 0.240 / 0.240 | 0.242 / 0.241 | collapse |

Scores: **A 4/8, B 11/12, C 9/12**, 0 indeterminate cells of 12.

- **B** (one physical core is not enough throughput, whichever core it is) fails only on
  `2x32/kc=8`, and cannot explain that row at all: two hyperthreads on one core add no execution
  resources to a single-threaded compute kernel, so a throughput story predicts no recovery there.
- **C** (the runtime's own threads contend for the single permitted cpu) explains `2x32/kc=8` and
  nothing else.
- **A** is scored on 8 cells because `smt` was declared out of domain for it in advance.

**No registered branch is fully consistent with the data.** The analyzer says so rather than awarding
the win to 11/12: "mostly consistent" is how a branch gets adopted on the strength of a criterion it
failed.

## 5. The correction: the recovery is not graded, it is per-row and one row is intermittent

The medians invite a tidy story. `smt` divided by the one-cpu floor `c0`, against the benchmark's own
`flopsPerCall = 2·MR·NR·kc`:

| row | flops/call | `c0` | `smt` | `smt/c0` |
|---|---:|---:|---:|---:|
| `2x32/kc=8` | 1024 | 0.8243 | 2.7375 | **3.321×** |
| `4x32/kc=8` | 2048 | 0.8054 | 1.0670 | **1.325×** |
| `2x32/kc=32` | 4096 | 0.8494 | 0.8750 | 1.030× |
| `4x32/kc=32` | 8192 | 0.8521 | 0.8561 | 1.005× |
| `2x32/kc=128` | 16384 | 0.8767 | 0.8801 | 1.004× |
| `4x32/kc=128` | 32768 | 0.8713 | 0.8715 | 1.000× |
| `2x32/kc=512` | 65536 | 0.8858 | 0.8887 | 1.003× |
| `4x32/kc=512` | 131072 | 0.8614 | 0.8650 | 1.004× |

That reads as a smooth decline in `flopsPerCall`, and I wrote it up that way before looking at the
samples underneath. **It is not one.** The four registered rows' `smt` distributions are different in
*kind*, not in degree:

| row | `smt` samples, both passes | shape | at the `c0` level | recovered |
|---|---|---|---:|---:|
| `2x32/kc=8` | 2.719 – 2.757 | **tight, one level** | 0/60 | **60/60** |
| `4x32/kc=8` | 0.835 – 2.721 | **spread across the whole range** | 12/60 | 20/60 |
| `2x32/kc=32` | 0.853 – 0.972 | tight, one level | 60/60 | 0/60 |
| `4x32/kc=32` | 0.843 – 0.932 | tight, one level | 60/60 | 0/60 |

("at the `c0` level" = within 1.15× of that row's own pooled `c0` median; "recovered" = above 1.5× it.)

So the second logical cpu delivers **complete and stable** recovery on one row, **intermittent**
recovery on one, and **none whatsoever** on two. The 1.325× on `4x32/kc=8` is not a partial
recovery — it is the median of a distribution whose samples are *either* at the confined level or
several times above it, so the arm intermittently attains the recovered state rather than steadily
attaining a fraction of it. Reporting 1.325× as a magnitude would have been reporting the midpoint of
a switch as a level. Rule 25's "an arm is an estimator, never a draw" has a corollary here: an
estimator over a distribution that is not unimodal is not a level either, and the monotone sequence
above is an artifact of taking medians across four qualitatively different shapes.

One incidental fact worth keeping: on **both** kc=8 rows the recovered state sits at ≈2.72–2.74
GFLOP/s regardless of shape, while `ref` on those rows sits at 2.77/3.19 and 3.19. A recovered level
that is the same absolute figure for two different `flopsPerCall` looks like another of the discrete
levels §6 documents rather than a fraction of `ref` — unregistered, n=2, recorded only as a shape.

### One hypothesis of mine, measured and refuted

Recovery ordered by work per call suggested a fixed cost *per call*. The ns/op column settles it
against me: `c0 − ref` is **873, 1900, 3534, 7300 ns** on the four registered rows, doubling with
`flopsPerCall` rather than staying constant. That is a proportional rate loss — the collapse
restated — not a per-call overhead.

## 6. The reference arm is contaminated, and the verdict survives it anyway

§1's unexplained `2.17` 5-minute average has a second, independent witness. Censusing every arm-row
pair for samples below 0.90× that row's own median finds **5 of 320**, and all five are scalar
registered rows:

| arm-pass | row | median | min | samples < 0.90× |
|---|---|---:|---:|---:|
| `bref` | `2x32/kc=8` | 3.194 | 2.760 | 5/30 |
| `bref` | `2x32/kc=32` | 3.654 | **1.058** | 7/30 |
| `bref` | `4x32/kc=32` | 3.538 | **1.283** | 9/30 |
| `asmt` | `4x32/kc=8` | 1.046 | 0.860 | 5/30 |
| `bsmt` | `4x32/kc=8` | 1.095 | 0.835 | 12/30 |

**Three of the five are in `bref` — the pass-B reference, the last arm to run, whose window is the one
the anomalous load average overlaps.** Two of those excursions reach 1.058 and 1.283 GFLOP/s, which is
*collapse magnitude inside the reference arm*. Separately, `aref`'s `avx512 6x32/kc=512` steps from a
first-half median of 87.09 to a second-half median of 82.92 mid-arm, which is why that row's control
ratios read ≈1.05: the step is `aref`'s, not the other arms' speed. This is `#147`'s within-run
bimodality, now measured in the reference rather than in a treatment arm, and the pre-registration's
interpretability checks did not test for it — check 2c validated `ref`'s pooled *median* against test
1's admissible levels (2.776, 3.179, 3.190, 3.542, all inside), which a partly-contaminated arm
passes. That is a gap in the registration, stated here rather than patched after the fact.

So the verdict was re-scored under four unregistered denominator choices. All are labelled
sensitivity analyses; none replaces the registered criterion:

| denominator | A | B | C | fully consistent |
|---|---:|---:|---:|---|
| registered: pooled median, both passes | 4/8 | 11/12 | 9/12 | NONE |
| pass A only (`aref` is clean on all four registered rows) | 4/8 | 11/12 | 9/12 | NONE |
| pass B only (`bref` carries three of the five excursions) | 4/8 | 11/12 | 9/12 | NONE |
| highest `ref` sample seen, adversarial to *intact* | 4/8 | 11/12 | 9/12 | NONE |
| lowest `ref` sample seen — i.e. dividing by an excursion | 2/8 | 5/12 | 5/12 | NONE |

**The scores are identical under the first four**, including under pass A alone, whose reference has
no excursion on any registered row. The fifth divides by a single contaminated sample rather than by a
level, and it does not hand the verdict to any branch either — it turns two rows INDETERMINATE and
lowers every score. The contamination is real, it is disclosed, and no conclusion in this report rests
on the arm that carries it.

## 7. Controls

- **16 control rows intact in all 8 arm-passes**, 0 outside the band, worst deviation 0.945 at
  `scalar 2x32/kc=128 c5`.
- **Positive control**: `c0` collapses on all four rows (0.297 / 0.252 / 0.267 / 0.241), so the
  explicit path reproduced test 1's width-1 arm and the run is interpretable.
- **`ref` on test 1's admissible levels**: `2x32/kc=8` 2.776 in [2.628, 3.349]; `2x32/kc=32` 3.179 in
  [2.863, 3.837]; `4x32/kc=8` 3.190 in [2.626, 3.348]; `4x32/kc=32` 3.542 in [2.914, 3.725].
- **Mirrored passes**: `c0` and `c5` agree pass-to-pass on every row. Where the per-pass *ratios*
  differ — `2x32/kc=8` reads 0.296 in pass A and 0.258 in pass B — the arm medians are 0.820 and
  0.825 and the difference is entirely `ref`'s level in the denominator. Drift would have moved the
  arms; it moved the reference.

## 8. The load-bearing cell, and how much of it is a band choice

`2x32/kc=8`'s `smt` cell is the only thing between this run and a clean 12/12 for branch B, so its
margin belongs in the open:

| denominator | `smt/ref` | verdict |
|---|---:|---|
| `ref` pass A (2.772) | 0.988 | intact |
| `ref` pooled (2.776) | 0.986 | intact |
| `ref` pass B (3.194) | 0.857 | intact, **+0.007** over the floor |
| highest single `ref` sample (3.205) | 0.854 | intact, **+0.004** over the floor |
| lowest `smt` sample / highest `ref` sample | 0.848 | **outside**, by 0.002 |

The margin is thin, but §5 says *where* the thinness lives, and it is not in `smt`. `smt`'s 60 samples
span 2.719–2.757 — the tightest distribution of any registered arm-row in the run. `ref`'s span
2.758–3.205, straddling two levels. The two ranges do not overlap, though the gap is one printed digit
wide, so read them as adjacent rather than separated. **The uncertainty in this cell is entirely
`ref`'s bimodality in the denominator, not noise in the arm** — and it reads intact against either of
`ref`'s levels, which is why the verdict holds under all four legitimate denominators in §6.

The band is **not** moved: *found by running it is not a reason to move it*, and shifting a floor a
result sits 0.007 above is tuning to the data. The conclusion is stated at the strength the margin
supports: branch B is refuted *if* 2.737 counts as intact beside a reference that runs at 2.772 and
3.194, and it does under every estimator declared in advance. The neighbouring row is not close to
anything — `4x32/kc=8` reads 0.328–0.343 under every choice.

## 9. What would settle it, and it needs no new apparatus

Branch C names a mechanism, so it can be attacked directly at width 1 without changing any mask:
remove the runtime's work and see whether the collapse follows. `GOGC=off` removes the GC workers;
`GODEBUG=asyncpreemptoff=1` removes preemption signals. Both are environment variables and
`KEEL_REMOTE_ENV` already carries any number of them into an arm — `remote.sh` interpolates it
unquoted ahead of `env`, so `GOMAXPROCS=1 GOGC=off` is two assignments. **Test 3 costs zero apparatus
lines**, only a driver under `archive/`, which `be6cca9` puts outside both ratio terms.

Two things test 3 should carry that this one lacked:

1. **An interpretability check for within-arm bimodality in `ref`**, not only for its pooled median.
   §6 is the argument for it and §6 is also the reason it is written into the *next* registration
   rather than retrofitted into this one.
2. **`4x32/kc=8` as a first-class row.** It is the only intermittent arm in the run, so it is where a
   mechanism that switches will be visible, and a median is the wrong instrument for it.

If the collapse is unchanged with both variables off, C is dead too and what remains is a property of
one logical cpu that neither throughput nor housekeeping explains. If `2x32/kc=8` recovers on a single
cpu with them off, C is confirmed and the per-row pattern becomes the question.

## 10. What `#148` now knows

| candidate | status |
|---|---|
| contention gradient | refuted, test 1 (widths 2/4/8 indistinguishable) |
| frequency residency | refuted **by sign**, test 1 |
| cpu0-specific | **refuted, test 2** (`c5/c0` = 0.995–0.999, §3) |
| one core is not enough throughput | **refuted at one row** — `2x32/kc=8` recovers 3.32× with no execution resources added, 60/60 samples; subject to §8's band margin |
| the runtime's own threads contend | **explains one row of four**; directly testable, §9 |

Four candidates down or wounded and the mechanism is still unnamed. What test 2 adds is that the
collapse is a property of having one *logical* cpu, does not care which physical core, and is relieved
by a second hyperthread **completely on one row, intermittently on one, and not at all on two** —
which is a sharper description than test 1 could give, and a different shape than the medians alone
suggested.
