<!-- Copyright 2026 The keel Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The `free-placement` arm, preserved

The 35 files beside this one are every `gate-p5` sample archive keel produced under the
`free-placement` measurement era — the era in which the OS chose which cores a benchmark's
goroutines ran on. They are here, tracked, because `scripts/measurement-eras.tsv` makes a
**both-arms transition archive** a condition of an era existing at all, and an arm cited by a
path under `build/` is cited on one operator's laptop and nowhere else (`.gitignore:13`; the
same defect issue #114 names for the BASELINE witness). Tracking them costs 464 KB of plain
text and buys a citation that resolves in any clone.

They are the *free* arm of the `pinned8` transition. The pinned arm is that transition run's
own archive, and `transition_archive` cannot be filled in until it exists.

## This set is closed, so it has no generator

`INDEX.tsv` was generated once, from the files beside it, by a command recorded in the commit
that landed it — deliberately not by a tracked script. No new member of this set can ever be
produced: `remote_exec` masks every invocation carrying `-test.bench` and **refuses** with
status 121 rather than running free (DESIGN.md §5 rule 5), so a thirty-sixth free-placement
archive would take an instrument change to make. A generator for a set that cannot grow would
be apparatus with nothing left to measure.

## What the arm covers, measured rather than assumed

Two kinds of file, told apart by which benchmarks they contain and not by their names:

| kind | files | contents |
|---|---|---|
| `ladder` | 15 | the judged benchmarks (`Scale/{Sgemm,Ssyrk,Ssymm,Strsm}` at 1 and 8 threads) plus `Peak` and, at four of five revisions, the `Ceiling` arm |
| `clock-window` | 20 | `Peak/avx512` alone, 10 samples — one invocation of §5 rule 5's substitute clock instrument |

The ladder archives are one per `(revision, host)` across five revisions and four CPU models.
**Twelve of the fifteen carry the `Ceiling` arm**; the three at `ce43bca` predate it, so a
share whose denominator is the measured attainable ceiling is recomputable at four revisions,
not five. Per CPU model, the ceiling-bearing coverage is:

| cpu_model | host | ceiling-bearing ladder archives |
|---|---|---|
| `AMD EPYC 9R14` | keel-zen4 | 4 (`0bbf964`, `33de3b2`, `5ec5fea`, `651d1bd`) |
| `AMD EPYC 9R45` | keel-zen5 | 4 (same revisions) |
| `Intel(R) Xeon(R) 6975P-C` | keel-gnr | 2 (`0bbf964`, `651d1bd`) |
| `Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz` | keel-skx | 2 (`33de3b2`, `5ec5fea`) |

## The disclosure this arm ships with

**These are single draws per configuration** (ruled 2026-08-21 on #6). Each archive holds ten
`-count` samples of each benchmark, so within-run spread is here and `benchstat` can report it.
What is *not* here is a second draw at the same code: the several archives a CPU model has sit
at different revisions, so they are one draw each of several configurations rather than several
draws of one. DESIGN.md §5 rule 16 — a published reference is an estimator, never a draw —
therefore applies to this arm exactly as it applies to the README rows measured alongside it,
and it is why `scripts/host-baselines.tsv` refuses to import keel-skx's baseline from
`5ec5fea` or `33de3b2`.

The consequence is stated rather than absorbed: **the era mapping's precision is bounded by the
free arm's own spread**, and this arm cannot bound its own run-to-run component. A pinned row
that differs from its free counterpart by less than that unmeasured component has not been shown
to differ at all. Nothing here is upgraded to make the transition read better.

## What the names do not record

The 20 clock-window archives cannot be mapped back to the head/middle/tail series the gate log
reports, and two independent causes are available for that. `peak_window` calls
`bench_csv "$log"` with no tag (`scripts/bench.sh:509`), so the window's own name — literally
`head` or `tail` — reaches the scratch log and not the archive; and these archives predate
`RUN_STAMP`, so a second run at one revision overwrote the first's files rather than sitting
beside them. Either would explain why each revision preserves **four** clock windows where a
three-host fleet running a three-invocation series implies six. Which one accounts for it is not
settled here, and neither is asserted: the tag is a one-word fix that has to wait, because
changing archive naming immediately before the era-founding run is the worst possible moment for
it, and `scripts/readme-numbers.sh` parses these names.

This limitation costs nothing that is published. Every keel number that cites this era cites a
`ladder` archive, and those are unambiguous — the host is in the name, one per `(revision,
host)`, with no collisions to resolve.

## Provenance of the era itself

- Named, retroactively and deliberately undatedly, by DESIGN.md §5 rule 17 —
  `scripts/measurement-eras.tsv`, row 1.
- Ended by `804fb75`, which put the mask in `remote_exec`.
- Every keel benchmark number published before `804fb75` was measured under free placement.
