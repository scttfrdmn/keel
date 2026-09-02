<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #147: what the panel addresses say about the within-run bimodality

Two arms on `janus.local` at `537661a`, launched detached, 2026-09-02 04:45–05:18Z. Pre-registered
before the run as [#147 comment](https://github.com/scttfrdmn/keel/issues/147) — criteria R1–R5,
their predictions, and the branches that would teach nothing. Every number below is re-derivable:

    python3 archive/addr147/analyze147.py archive/addr147/bench-addr147-537661a-w8.txt \
        archive/addr147/addr147-537661a-w8.trace w8

reproduces `archive/addr147/analysis-w8.out` byte for byte (checked; likewise `w1`).

The analyzer was controlled before any real data reached it: `archive/addr147/control147.py` drives
all four registered R2 verdicts, the R5 refusal, the R5 *ambiguity* refusal, the unimodal no-op and
the period-2 branch with synthetic input whose verdict is known by construction (`controls.out`).
Control 8 exists because the `ramp-vs-timed` counter had only ever printed 0 — correctly, but a
counter that has never printed anything else is an unread witness; it prints 30/30 there.

## Provenance

One binary for both arms — `sha256=d0d46d26c15cc8b2`, 5066553 bytes,
`-buildmode=exe -compiler=gc -trimpath=true`, `go1.27.0-X:simd` read off the artifact rather than off
the driver's shell — so a layout difference cannot masquerade as a pinning effect. Intel Core i9-9960X,
`governor=performance`, `GOMAXPROCS=1`, `-count=30 -benchtime=1s`, 20 rows (5 shapes × 4 `kc`).
Arm 1 `mask=0,1,2,3,4,5,6,7 width=8`; arm 2 `mask=0 width=1`. Both arms `REMOTE_STATE=ok`, 600 rows
each, 0 trace lines inside either bench log. Driver, logs, traces and analysis: `archive/addr147/`.

## R5, the join gate — and a correction to how it was implemented

R5 reads: each of the 30 reported samples must match **exactly one** trace execution by `n`, with
`elapsed/n` agreeing to the printed 4 significant figures. That sentence admits two implementations
and they disagree, so both are computed and both are in the output:

| implementation | measurable row-arms of 40 |
|---|---|
| `render` — reproduce `testing.prettyPrint`'s rendering of `elapsed/n` and compare it to the token in the log | **37** |
| `sig4` — round `elapsed/n` and the printed value each to 4 significant figures | 30 |

`sig4` double-rounds: `testing` has already rounded once to produce the log. Read from
`$GOROOT/src/testing/benchmark.go`, `prettyPrint` prints anything `>= 999.95` with `%10.0f` — an
integer, so a large `ns/op` carries *more* than four significant figures, and the second rounding is
both too loose (samples collide inside one 4-sig-fig bucket) and too tight (a true 19845.4 renders
"19845", whose own 4-sig-fig rounding is 19840 while the true value's is 19850, so a correct pair
misses). All 7 row-arms where the two implementations disagree have medians above 10000 ns/op, which
is the mechanism's own prediction.

`render` is used for R1–R4. **The choice was made after seeing `sig4` fail**, so: it is strictly more
discriminating per sample than `sig4`, it is a property of `testing`'s formatter and not of the data,
and the `sig4` column stays in the published table for every row. The transcription of `prettyPrint`
is checked against the toolchain rather than against a second reading of the same source —
`testing.BenchmarkResult.String()` calls it, so Go itself renders the same values
(`archive/addr147/prettycheck/main.go`, compared by `archive/addr147/prettycheck147.py`, output in
`prettycheck.out`):

    pretty() vs the toolchain: 18 match, 0 mismatch (1 not observable through String())
    positive control (4 sig figs everywhere): 13 of 18 mismatch -- a nonzero count is what
    makes the 18/18 above meaningful

19 cases straddle every boundary in the switch; the 19th is `T=0`, where `String()` omits the `ns/op`
field entirely, so `prettyPrint`'s `y == 0` branch is **not** observable through this witness and is
reported as skipped rather than counted as a match. The control is the reading R5's wording literally
suggests, and it fails on exactly the large values where `sig4` failed on real data.

**Three row-arms stay unmeasured, and the cause is a limit of the join itself, not of the run**:
`4x32/scalar/kc=512` (w8), `2x32/scalar/kc=8` and `4x32/scalar/kc=8` (w1). In each, all 30 samples
share one `b.N` and two or more samples *render identically* — w8 samples 25 and 26 both print
`162216` at `n=10000`. Two samples with the same `(n, printed)` pair are indistinguishable **in
principle** by an arithmetic join. The fix for the next run is an instrument change, not a criterion
change: emit a monotone execution counter alongside the addresses, and let arithmetic check the
positional pairing instead of carrying it alone.

## R1 — address variation: the registered prediction failed

Registered: **≥ 8 of 20 rows with all three addresses constant across the 30 timed samples**,
extrapolated from the dev host. Measured: **2 of 19** analyzable rows (w8) and **3 of 18** (w1).

The prediction failed in an informative direction. All five constant rows are `kc=512`; on the dev
host it was the `kc=8` rows that were constant and the `kc=512` rows that moved. On janus the small
panels churn: on four of the five w8 `kc=8` rows, `c` takes 10–15 distinct addresses across the 30
timed samples and 19–27 across all executions, while the large ones get a span of their own and stay
in it. (The fifth, `6x32/avx512/kc=8`, holds `c` at 2 addresses — it is also the one cell whose tuple
classes came out finer than its modes.) The registration
said this was a prediction that could simply fail, and it is recorded as one, not repaired.

## R2/R3 — the decisive test: 4 bimodal cells, hypothesis 1 refuted on 3

#147's classifier, unchanged (largest adjacent gap > 1.5% of the median, both clusters ≥ 3), on the
37 measurable row-arms:

| arm | cell | gap | H\|L | address tuples | registered verdict |
|---|---|---|---|---|---|
| w8 | `2x32/avx512/kc=32` | 5.87% | 13\|17 | 9, not mode-pure | **hypothesis 1 refuted** — a tuple appears in both modes |
| w8 | `2x32/scalar/kc=128` | 5.91% | 24\|6 | 12, not mode-pure | **hypothesis 1 refuted** |
| w1 | `4x32/avx512/kc=128` | 2.39% | 3\|27 | 4, not mode-pure | **hypothesis 1 refuted** |
| w1 | `6x32/avx512/kc=8` | 2.95% | 15\|15 | 7, each mode-pure | **inconclusive** — strictly finer than the modes |

2 cells per arm, inside the registered expectation of 2–3, so R2 is `unmeasured` on neither arm (R3's
null branch did not fire). Placement is refuted on three of the four cells by the strongest available
evidence: the same panel addresses appear on both sides of the mode boundary, so the addresses cannot
be what separates them.

The inconclusive cell is the interesting one and it is *not* a registered verdict. `6x32/avx512/kc=8`
at width 1 is a signature-A cell — strict period-2 `HLHLHL…`, 15\|15 — with 7 address tuples, each
one confined to a single mode, and the 4 KB-alias columns the registration asked for regardless of
verdict come out as: `(a−c) mod 4096 ∈ {0, 640}` with **each alias value mode-pure**, and
`(b−c) mod 4096 ∈ {1024, 1664}`, likewise. The tuple identity is finer than the modes; the *alias* is
exactly as fine as the modes on this cell. Its `ramp-vs-timed` count is 30/30 — every timed run sat at
a different address from the execution just before it. That is the pre-registration for the next run,
not a conclusion from this one: one cell, one draw, and the alias test was chosen after seeing it.

## R4 — hypothesis 2: core migration is refuted as the mechanism

The registered asymmetry was stated in advance: a clean width-1 arm would be *consistent with*
migration without establishing it, and the informative branch is persistence. **Arm 2 yields 2
bimodal cells**, so the modes survive with the thread confined to one core, where there is no
migration among the eight masked cores to have caused them. Migration is refuted as the mechanism for
those cells. Note the mask semantics that makes this clean: `keel_pin_mask` selects distinct physical
cores one per cache domain, so arm 1's eight CPUs are eight cores, not four cores' worth of SMT pairs.

## Post-hoc: #147's classifier is defeated by a single outlier

Reported as a sensitivity, in `archive/addr147/sensitivity147.py` so it cannot be mistaken for the
registered criterion. The classifier takes the largest adjacent gap **and only then** checks cluster
sizes, so one gross outlier owns the gap and the row reads unimodal. Two w8 scalar rows show this
plainly: `2x32/scalar/kc=32` runs 1213…1384 with one sample at 4373, and `4x32/scalar/kc=32` runs
2297…2681 with one at 8505. The registered classifier calls both unimodal; a variant that maximizes
the gap *over splits that already satisfy the ≥ 3 rule* finds a 9.23% split at 18\|12 and an 8.80%
split at 25\|5.

That variant adds **4 cells on w8 and 0 on w1**, and every one of the four is **refuted** — a tuple
appears in both modes. So the sensitivity strengthens the headline rather than qualifying it: 7 of 8
cells refute placement, 1 is inconclusive. The proposal for the next registration is the variant
above; it is not applied retroactively to anything.

## An unregistered cross-arm finding, recorded and not explained

`archive/addr147/crossarm.out`. Every `avx512` row lands within 0.97–1.10 of its width-8 median at
width 1, and the scalar `kc=128`/`kc=512` rows within 1.00–1.05. Four rows do not:

| row | w8 median | w1 median | w1/w8 |
|---|---|---|---|
| `2x32/scalar/kc=8` | 350 ns | 1328 ns | **3.79** |
| `2x32/scalar/kc=32` | 1376 ns | 5114 ns | **3.72** |
| `4x32/scalar/kc=8` | 643.1 ns | 2700 ns | **4.20** |
| `4x32/scalar/kc=32` | 2556 ns | 1.018e4 ns | **3.99** |

The affected rows are exactly the small-`kc` scalar ones, and the sharper statement is not "four rows
got slower" but **"the small-`kc` scalar advantage disappeared"**. Median `GFLOP/s` per scalar row,
read from the logs' own column (no OpenBLAS reference was available on janus, so these are absolute
rates against the `8.8 GFLOP/s` scalar *formula* cross-check, not against a measured scalar peak):

| `kc` | 2x32 w8 | 2x32 w1 | 4x32 w8 | 4x32 w1 |
|---|---|---|---|---|
| 8 | **2.926** | 0.772 | **3.184** | 0.759 |
| 32 | **2.976** | 0.801 | **3.206** | 0.804 |
| 128 | 0.870 | 0.827 | 0.845 | 0.818 |
| 512 | 0.839 | 0.838 | 0.813 | 0.807 |

At width 1 *every* scalar row sits in 0.76–0.84 GFLOP/s — a flat band, indistinguishable from the
large-`kc` rows' rate in **either** arm. The L1-resident rows' 3.5–3.9× advantage exists at width 8
and is gone at width 1; the memory-bound rows never had it and do not change. The mechanism is
unmeasured and I am not
going to name one: candidates are SMT-sibling contention (janus is 16 cores / 32 threads, and
`cpu0`'s siblings are `0,16` — the width-8 mask covers CPUs 0–7 and leaves 16–23 free for any other
process, so pinning does not isolate a core from a co-tenant), CPU-0 interrupt affinity, and
frequency residency across eight cores versus one. Two things are measured: the effect, and that
**no instrument recorded whether the host was quiet**. Nine minutes after the run ended, janus's
15-minute load average read 1.45 with my own single-threaded arm long finished; the one visible
co-tenant (`rmw-new.bin`, PID 2519612) started at 22:27:31 local, *after* the run, so it is excluded
as the cause and the residual load is unattributable. A repeat is not takeable while that co-tenant
runs. Filed separately; the harness change it argues for is a co-tenant/load line in the provenance
block, so a contaminated window is detectable after the fact instead of never.

## What this pair of arms cannot see

- **One host, one revision, one 30-sample window per arm.** Bimodality is a property of the draw:
  #147 already showed no row was bimodal in all six of its arms.
- The **20 outlier cells** #147 could not classify are untouched, and the 1.5%/≥3 threshold was
  chosen after seeing #147's data — what is pre-registered is everything keyed to the classifier's
  output, never the threshold.
- An instrumented binary is a different binary; between-binary layout on keel is 1.71/0.99/1.32%
  (#54/#61), the same order as the mode gaps. Arm 1 is a **fourth draw**, not a continuation of the
  three in `archive/pinned8/`.
- The three unmeasured row-arms are unmeasured, not clean: `4x32/scalar/kc=512` (w8),
  `2x32/scalar/kc=8` and `4x32/scalar/kc=8` (w1) were never analyzed for R1 or R2.
- Nothing here says what *does* cause the modes. Placement is out on 7 cells and migration is out on
  the 2 that survived width 1; the remaining candidates — frequency/thermal residency, co-tenant
  interference, alias structure finer than tuple identity — are not separated by this run.
