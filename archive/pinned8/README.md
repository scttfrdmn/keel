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

The free half of the transition is `archive/free-placement/` (35 archives). What this directory
does *not* do is close the era — see "why `transition_archive` is still `—`".

## Membership is a measured predicate, not a revision range

A file is in this set because **it contains a `keel-pin:` line**: the instrument that produced it
recorded the mask it actually ran under. That is the same discipline `INDEX.tsv` uses for every
other column — read the fact out of the archive, not out of the name.

It matters here because a revision range would have got this wrong. The mask landed in `804fb75`,
but the set is not "everything after `804fb75`": what makes an archive pinned is that the mask
applied to *it*, and `keel-pin:` is the only witness of that. Applying the predicate to `build/`
also returned exactly **35 unpinned** archives whose names are identical to the 35 already in
`archive/free-placement/`, which is an independent check that the free arm was preserved whole.

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

## Why `transition_archive` is still `—`

**Because the campaign that fills it has not run yet.** The column wants the fleet measured under
the old instrument and the new one, and as of the 2026-08-22 amendment the new instrument is the
spread mask — so the pinned half of the transition is the spread-form campaign, not these 24
files. They are the arm of a form that no longer defines the era.

The **gnr question is settled and is no longer why this column is empty.** `Intel(R) Xeon(R)
6975P-C` appears 8 times in the free arm and 0 times here because `keel-gnr` was ruled a
*characterization* host and is absent from the pinned fleet (#6 Q3), and the 2026-08-22 ruling
scopes the both-arms condition to **judged** hosts: gnr is a **stated exclusion** in the era row's
record rather than an unmet condition, since the era certifies nothing about a host it does not
govern. A pinned-era gnr re-measure stays available as characterization work (#104/#112) and is
not owed to this column.

Note what provisional does not cost: the era table says PROVISIONAL "is a disclosure, not a
permission — it does not widen what BASELINE covers", and what bounds BASELINE is
`scripts/judged-runs.tsv`, one row per `(host, era)`, provisional or not.

## This set is not closed, unlike the free arm — but its confined subset is

`free-placement/` has no generator because no thirty-sixth member can ever exist — the mask
refuses with status 121 rather than running free. **This directory can still grow, and what it
grows with is spread-form archives**: every future `gate-p5` run adds to it while rule 5 stands,
and every one of those carries `doms=`/`nodedoms=`. What is closed is the **confined subset**, at
24 as of 2026-08-22: the selector now spreads or refuses, so no twenty-fifth pre-amendment reading
can ever be taken. Every count and table above is a statement about those 24 and does not move.

`INDEX.tsv` was still generated once by a command
recorded in the commit that landed it, rather than by a tracked script, because a generator is
`scripts/` budget spent on a file that regenerating by hand costs one command
(CLAUDE.md, "the apparatus pays its own way"). If this set starts growing per-run, that trade
changes and the generator becomes worth its lines.
