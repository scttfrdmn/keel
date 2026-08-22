<!-- Copyright 2026 The keel Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The `pinned8` arm, preserved

The 24 files beside this one are every `gate-p5` sample archive keel has produced under the
`pinned8` measurement era — the era in which every benchmark-carrying invocation runs under one
CPU affinity mask of eight distinct physical cores inside a single NUMA node (DESIGN.md §5 rule 5,
implemented in `804fb75`). They are here, tracked, for the reason the free arm is: an arm cited by
a path under `build/` is cited on one operator's laptop and nowhere else, and `build/` is ignored
(`.gitignore:13`).

This is the **pinned half** of the `pinned8` transition that `scripts/measurement-eras.tsv` makes
a condition of the era existing. The free half is `archive/free-placement/` (35 archives). What
this directory does *not* do is close the era — see "why `transition_archive` is still `—`".

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

**Granite Rapids has no pinned half.** `Intel(R) Xeon(R) 6975P-C` appears 8 times in the free arm
and 0 times here, because `keel-gnr` was ruled a *characterization* host and is absent from the
pinned fleet (#6 Q3). So "the same fleet measured under the old instrument and the new one" holds
for **three of the four** models the free era touched, and does not hold for the fourth.

That gap is arguably consistent with the condition rather than a breach of it — gnr is not part of
the fleet the era governs, so there is nothing for the era to certify about it — but reading the
condition as met is a judgment about what "the same fleet" means, and certifying a condition with
a stated hole in it is not a call to make while filing the evidence for it. **Left as `—`, asked on
#6.** Preserving the arm is unambiguously right and is done; closing the era is the open question.

Note what provisional does not cost: the era table says PROVISIONAL "is a disclosure, not a
permission — it does not widen what BASELINE covers", and what bounds BASELINE is
`scripts/judged-runs.tsv`, one row per `(host, era)`, provisional or not.

## This set is not closed, unlike the free arm

`free-placement/` has no generator because no thirty-sixth member can ever exist — the mask
refuses with status 121 rather than running free. **This set can still grow**: every future
`gate-p5` run adds to it while rule 5 stands. `INDEX.tsv` was still generated once by a command
recorded in the commit that landed it, rather than by a tracked script, because a generator is
`scripts/` budget spent on a file that regenerating by hand costs one command
(CLAUDE.md, "the apparatus pays its own way"). If this set starts growing per-run, that trade
changes and the generator becomes worth its lines.
