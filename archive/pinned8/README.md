<!-- Copyright 2026 The keel Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The `pinned8` arm, preserved — and superseded 2026-08-22

The 24 files beside this one are every `gate-p5` sample archive keel produced under the **first
form** of §5 rule 5's affinity mask: eight distinct physical cores taken in ascending order
inside one NUMA node (implemented in `804fb75`). They are here, tracked, for the reason the free
arm is: an arm cited by a path under `build/` is cited on one operator's laptop and nowhere else,
and `build/` is ignored (`.gitignore:13`).

**That form was amended on 2026-08-22 and these 24 readings are the *provisional* arm, not the
era's founding one.** On EPYC 9R45 sysfs reports `index3`'s `shared_cpu_list` as exactly eight
cores wide, so "the first eight in order" was definitionally one CCD, and the 8-thread stream
ceiling these files carry sits **5.96×** (dot) and **4.65×** (axpy) below the same host measured
one core per CCD — and 1.69× below *free placement* (T-45, three controls; DESIGN.md §5 rule 5).
Rule 5 now takes one core per cache domain, and the era finalizes on that form by the 2026-08-22
ruling on #6. Nothing here is edited: these are what the instrument read, and they are the
evidence that the amendment was needed.

**They identify themselves.** Every archive taken after the amendment carries `doms=` and
`nodedoms=` on its `keel-pin:` line — the domain of each selected core and the count its node
offered. None of these 24 do, so the two arms are separable by their own witnesses with no edit
to either, which is the membership discipline this file already uses one field over.

The free half of the transition is `archive/free-placement/` (35 archives). These 24 confined files
do *not* by themselves close the era; the spread arm that does now sits beside them — see
"`transition_archive` is filled and the era is CLOSED".

## Membership is a measured predicate, not a revision range

A file is in this set because **it contains a `keel-pin:` line**: the instrument that produced it
recorded the mask it actually ran under. That is the same discipline `INDEX.tsv` uses for every
other column — read the fact out of the archive, not out of the name.

It matters here because a revision range would have got this wrong. The mask landed in `804fb75`,
but the set is not "everything after `804fb75`": what makes an archive pinned is that the mask
applied to *it*, and `keel-pin:` is the only witness of that. Applying the predicate to `build/`
also returned exactly **35 unpinned** archives whose names are identical to the 35 already in
`archive/free-placement/`, which is an independent check that the free arm was preserved whole.

**The predicate over-collects, and the nine `drift-*` files are why this says so (2026-09-01,
#141).** They are three null A/B runs on `janus.local` — a lab host, neither fleet nor judged — and
each run's two arm files carry `keel-pin: mask=0,1,2,3,4,5,6,7 width=8` and `doms=`, which is *both*
stated predicates, so a re-derivation by predicate alone would file all six arm files in the spread
arm. They are **excluded by name**, which is in doctrine rather than around it: `INDEX.tsv` and
`INDEX-spread.tsv` are the membership, and the predicate is the check on them. Neither index cites
any of the nine and no verdict rests on them.

Extension does not separate them and I am not going to pretend it does. Measured before adding
them: all 8 `.log` files here carried **zero** `keel-pin:` lines, so `.log` meant driver transcript
and `.txt` meant sample archive, exactly. The six arm files are `.log` files that are sample
archives — 1 `keel-pin:` line and 690 `Benchmark` rows each, against 0 and 0 for the three driver
transcripts — so the convention now has six exceptions and a named exclusion list is what carries
the fact instead. They are tracked at all for #114's reason only: a published attribution whose
evidence is a `build/` path is evidence on one laptop.

**Two naming forms, and the reason is worth more than the tidiness would have been.** The three
`df999da` files were *renamed* when archived (`drift-janus-df999da-base.log` here against
`drift-janus.local-base-df999da.log` under `build/` — contents byte-identical, verified). The six
added for W0/W1 keep **the names the driver wrote**. Normalizing them would have been one `mv`, and
it is refused for this directory's own stated reason: a citation should resolve to what the
instrument produced, and a renamed file is a small edit to the record of a run. So the mixed
convention is deliberate and is the tell for exactly one fact — which files were touched on the way
in. Read the host, revision and arm out of each file's header, not out of its name.

| file | sha256 | role |
|---|---|---|
| `drift-janus-df999da.log` | `5889db9df6fc1fa6…` | driver, draw 1 |
| `drift-janus-df999da-base.log` | `b9d9f5e6000f41b8…` | arm |
| `drift-janus-df999da-new.log` | `4a250b06686edaf1…` | arm |
| `drift-w0-9862637.log` | `526fb48e447f637d…` | driver, draw 2 (#141's W0) |
| `drift-janus.local-base-9862637.log` | `1de1e449346851fd…` | arm |
| `drift-janus.local-new-9862637.log` | `34496914ad8bb167…` | arm |
| `drift-w1-065608d.log` | `9ed6c9d34db99591…` | driver, draw 3 (#141's W1, `-trimpath`) |
| `drift-janus.local-base-065608d.log` | `f0ed1a1ccb2fc6b0…` | arm |
| `drift-janus.local-new-065608d.log` | `934e7bc12f8d695c…` | arm |

W1's two arm files are the evidence for a claim no earlier archive here could make: **both arms of
that run are the same binary**, `sha256=34a87563622bb6c3… bytes=5066630 flags=[-buildmode=exe
-compiler=gc -trimpath=true]`, so the deltas between them are the host between two windows with
build layout excluded by construction. All six arm files together are the input to #147 — 18 of 138
(row, arm) cells bimodal *within* a single `-count=30` window — which is why the arm files and not
only the driver transcripts had to be preserved: the finding is in the per-sample rows, and a
benchstat table cannot show it.

The three revisions present are `e64b34e` (9), `be5bb91` (8) and `d2fe477` (7). All 24 carry the
same mask, `mask=0,1,2,3,4,5,6,7 width=8`, and all 24 report `gomaxprocs=8` — uniform, as rule 5's
"fleet-wide and never selectively" requires. Rows carry `-8` where the free arm's carried `-192`
or `-72`.

## Two kinds, told apart by contents

| kind | files | contents |
|---|---|---|
| `ladder` | 9 | 210 sample rows: `Scale` (the judged benchmarks) + `Ceiling` + `Peak` |
| `clock-window` | 15 | 10 sample rows: `Peak` only — rule 5's clock check |

## The mapping to the free arm, by `cpu_model`

The era table asks for the same fleet measured under both instruments. Mapped at the level of
**CPU model**, which is the key the registries use, and not at instance identity — AWS instances
are ephemeral, so `keel-zen5` in the free arm and `keel-zen5` in this one are the same silicon and
different machines. That is the same `(host, era)` → `(cpu_model, era)` witness-key deviation
already disclosed on #6.

| `cpu_model` | free `ladder` | pinned `ladder` | free `clock-window` | pinned `clock-window` |
|---|---|---|---|---|
| AMD EPYC 9R14 | 5 | 2 | 6 | 2 |
| AMD EPYC 9R45 | 5 | 2 | 5 | 3 |
| Intel Xeon Platinum 8124M | 2 | 5 | 4 | 10 |
| **Intel Xeon 6975P-C** | **3** | **0** | **5** | **0** |

`8124M` is over-represented here because three of the runs (`e64b34e`) were the
BASELINE-REGISTERED exercise, which drove that host repeatedly. Those archives are members
regardless: the era's arm is every reading the instrument took, not only the readings a criterion
judged, exactly as the free arm includes its `clock-window` files.

## `transition_archive` is filled and the era is CLOSED (2026-08-28)

**It was empty because the campaign that fills it had not run.** The column wants the fleet
measured under the old instrument and the new one, and from the 2026-08-22 amendment the new
instrument is the spread mask — so the pinned half of the transition is the spread-form campaign,
not the 24 confined files, which are the arm of a form that no longer defines the era.

That campaign has now run and **its 21 archives are preserved here**, indexed in
`INDEX-spread.tsv`: 9 `ladder` sweeps at `instrument=v2` (three revs × three judged hosts) and 12
`clock-window` peak logs from the same runs. Both arms preserve both `kind`s, which is what lets
them be enumerated on the same terms. Membership is the **measured predicate** in both directions —
spread self-identifies by carrying `doms=`/`nodedoms=`, confined by lacking it — so the two indexes
partition this directory and each is the other's complement rather than a list someone maintained.

The three judged CPU models are present in **both** arms, which is the condition itself:

| cpu_model | free | confined | spread |
|---|---|---|---|
| `AMD EPYC 9R14` | 11 | 4 | 3 |
| `AMD EPYC 9R45` | 10 | 5 | 3 |
| `Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz` | 6 | 15 | 3 |
| `Intel(R) Xeon(R) 6975P-C` | 8 | 0 | 0 |

**What actually forced the preservation was not this column.** The published README rows cited
their own evidence as `build/` paths, and `build/` is gitignored — so 24 published numbers rested
on logs present on one operator's machine and nowhere in the repo. That is #114's defect arriving
at the *publication* layer rather than the witness layer: right for the reader who ran the gate,
unverifiable for every other one. Preserving the arm fixes the era condition and the provenance
citation with one act, and the README was regenerated against the `archive/` paths so its sentence
resolves for anyone with a clone. The three driver logs the generator reads are preserved beside
the archives for the same reason: a citation to a file nobody else has is not provenance.

The **gnr question is settled and is no longer why this column is empty.** `Intel(R) Xeon(R)
6975P-C` appears 8 times in the free arm and 0 times here because `keel-gnr` was ruled a
*characterization* host and is absent from the pinned fleet (#6 Q3), and the 2026-08-22 ruling
scopes the both-arms condition to **judged** hosts: gnr is a **stated exclusion** in the era row's
record rather than an unmet condition, since the era certifies nothing about a host it does not
govern. A pinned-era gnr re-measure stays available as characterization work (#104/#112) and is
not owed to this column.

Note what provisional never cost, now that it is behind us: the era table says PROVISIONAL "is a
disclosure, not a permission — it does not widen what BASELINE covers", and what bounds BASELINE is
`scripts/judged-runs.tsv`, one row per `(host, era)`, provisional or not. Closing the era therefore
loosened nothing; it landed the row that **spends** keel-skx's one BASELINE, which is the opposite
direction. That row and the three baselines beside it went in together on purpose: a witness with no
baseline is the `owing` state, which gate-p5 renders **FAIL**.

## This set is not closed, unlike the free arm — but its confined subset is

`free-placement/` has no generator because no thirty-sixth member can ever exist — the mask
refuses with status 121 rather than running free. **This directory can still grow, and what it
grows with is spread-form archives**: every future `gate-p5` run adds to it while rule 5 stands,
and every one of those carries `doms=`/`nodedoms=`. **Amended 2026-09-01: not only `gate-p5`.**
The 13 `bench-gate-p3-afb108e-*` files are the first `gate-p3` samples preserved here, and they
are here because that gate publishes a percent-of-peak whose numerator and denominator appear in
**no** log — only the ratio does — so without them the first on-fleet P2 judgment would have been
a number citable on one laptop, which is the failure this whole directory exists to prevent. Nine
of the 13 are spread-form and say so; the other four say the opposite by carrying **no**
`keel-pin:` line at all — the three OpenBLAS-tagged single-thread runs and the 72-thread peak run
ran unpinned, and each of those three holds `BenchmarkSgemm` and `BenchmarkOpenBLAS` in one file,
so the mission criterion's two arms share a placement even where that placement is free. None of
the 13 are in `INDEX.tsv` (frozen at the 24, by its own header), and which host each one ran on is
read from its `keel-bench-cpu:`/`keel-pin:` header rather than from its name, exactly as the
membership predicate above prefers.

**Extended the same day to the two `janus` runs, for a sharper version of the same reason.** The 2
`gate-p3-under-p4-{68a9bec,ba6f286}` logs and their 28 samples are here because `janus.local` was
the sole judged sentinel on both — that is #146's defect — and `janus` appears **nowhere** in
either run's tracked driver log (`release-a2-68a9bec.log`, `validate113-ba6f286.log`: zero
mentions). So v0.1.0-a2's certificate carried a P2-floor judgment, and #113's `gate-p5: RED`
carried the FAIL that contradicted it, with the entire witness for both sitting under a gitignored
`build/`. Preserving them also fixes a second thing the ratios could not show: the two runs'
`BenchmarkPeak` medians agree to within 0.16% on all three backends while four of five
`BenchmarkKernel` medians move by 5-12%, which is a statement about that host that no percentage
in either log could have made. What is closed is the **confined subset**, at
24 as of 2026-08-22: the selector now spreads or refuses, so no twenty-fifth pre-amendment reading
can ever be taken. Every count and table above is a statement about those 24 and does not move.

`INDEX.tsv` was still generated once by a command
recorded in the commit that landed it, rather than by a tracked script, because a generator is
`scripts/` budget spent on a file that regenerating by hand costs one command
(CLAUDE.md, "the apparatus pays its own way"). If this set starts growing per-run, that trade
changes and the generator becomes worth its lines.
