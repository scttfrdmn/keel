<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The upstream plan

keel's second deliverable is a field report on `GOEXPERIMENT=simd`. This file is
the root of the workstream that turns the report into contributions: three CLs, one
experience report, one conditional arm64 filing, and the prerequisites without
which none of them can be mailed.

Everything here is on an **external clock**. Review latency, not writing time, is
the budget, and the Go 1.28 tree freeze is the deadline nothing in this repo can
move. That is why the milestone
[`upstream-go1.28`](https://github.com/scttfrdmn/keel/milestone/8) has a due date
and the others do not.

## The ledger

| # | what | upstream | keel evidence |
|---|---|---|---|
| [#124](https://github.com/scttfrdmn/keel/issues/124) | prerequisites: CLA, Gerrit, Go from source | — | — |
| [#127](https://github.com/scttfrdmn/keel/issues/127) | **CL 1** — MulAdd accumulate-in-place + `.BCST` folding | `golang/go#80829` | [#20](https://github.com/scttfrdmn/keel/issues/20), [#18](https://github.com/scttfrdmn/keel/issues/18), [#104](https://github.com/scttfrdmn/keel/issues/104) |
| [#126](https://github.com/scttfrdmn/keel/issues/126) | CL 2 — adopt CL 803220, SIMD ops in LICM | `golang/go#79984` | [#54](https://github.com/scttfrdmn/keel/issues/54) |
| [#125](https://github.com/scttfrdmn/keel/issues/125) | **CL 3** — emulated broadcast + inlined-wrapper NOP anchor | `golang/go#80830` | [#17](https://github.com/scttfrdmn/keel/issues/17) |
| [#128](https://github.com/scttfrdmn/keel/issues/128) | experience report on the graduation thread | `golang/go#73787` | the whole tree |
| [#129](https://github.com/scttfrdmn/keel/issues/129) | NEON feasibility probe (no upstream artifact) | — | — |
| [#130](https://github.com/scttfrdmn/keel/issues/130) | arm64 filing, **conditional on #129** | TBD or none | TBD |

Order: **#129 first** (it needs nothing and nobody), Scott's half of #124 in
parallel, then CL 1 within two to three weeks, then the rest as review allows.
#130 may close as not-needed; that is a result, not a failure.

## The CL order is keyed to the subject, not to the number

**Ruled 2026-08-30.** The first version of this file, and the plan it descends
from, listed "CL 1: `golang/go#80830` embedded-broadcast lowering". Embedded
broadcast — `.BCST`, `VFMADD231PS` — is not in `golang/go#80830`; it is in
`golang/go#80829`. Both of the original descriptors named halves of one issue.
Caught by searching the tracker before filing, which is `CLAUDE.md`'s standing
order and the same order that turned T18 from a new register-allocation finding
into a duplicate of an already-open `golang/go#79984` with a fix CL in flight.

The ruling: **the number serves the subject.** The intent was always
embedded-broadcast-first, so `golang/go#80829` is CL 1 (#127) and
`golang/go#80830` is CL 3 (#125). Each issue kept its own subject, evidence and
verified upstream number; only the ordinal moved. Swapping the *upstream
references* onto unchanged bodies would have re-created the mis-key in a
better-hidden form. A cross-reference is a claim, and it gets the same
transcription discipline as any other — which is why the correction landed in all
three places that carried it rather than only in the two issues.

**Two of the ordering's three premises did not verify; it stands on the third.**

- *"smallest CL first"* — `golang/go#80829` is the **largest** of the three. What
  lets it lead is not size but that its two halves are independent, so a freeze
  squeeze degrades it to a partial landing instead of nothing.
- *"the 110× spill price is its evidence"* — the magnitude is **contradicted**, not
  merely unlocated (see below). The subject linkage survives: the 6×32 tile spills
  because only 15 of 32 vector registers are allocatable, which is #18, which is
  CL 1's other half.
- *"highest keel payoff"* — **verified**, and this is what the order rests on.
  #104, the P2 STOP, states that 55% of measured peak is unreachable on Sapphire
  Rapids until `golang/go#80829` lands. No other CL here has a blocked phase gate
  behind it.

## Three figures did not survive verification and are not in any CL description

Numbers reach a CL description only after being re-read from the tree, because a
figure carried from a plan is a figure nobody checked:

- *"the measured 110× µarch spill price"* — **contradicted, and by the report that
  is supposed to contain it.** `docs/spill-report.md` measures exactly this
  quantity — the spilling 6×32 tile against the best shipped shape, numerator and
  denominator in the same run — and its value is **2.54×** on janus
  (`99.48/39.24`), **3.17×** on vesta (`159.90/50.44`) and **4.37×** on antares
  (`210.20/48.13`). The largest is 4.37×, so 110× is **25× larger than anything
  the report contains**. This is a stronger finding than the "searched for and not
  found" recorded on 2026-08-29, and it was reached by asking what the report
  measures instead of grepping for a string. **Cite 4.37× with its host and
  denominator, or cite the structural figure instead:** 90 vector stack refs in
  `Kernel6x32`'s steady-state loop against **0** in both shipped shapes
  (`docs/spill-report.md:37-47`).
- *"T17 nosplit −15.5%"* — **wrong sign, and the caveat was missing.** It is
  **+15.5%** static instructions in `internal/l1`, and it *was never paid*:
  `vec.LoadPart512` still calls `archsimd.LoadFloat32x16SlicePart` directly, so
  the figure is a measured **quotation** for a change that became obsolete before
  it was written (`docs/toolchain-notes.md:1409-1417`). It may be cited only with
  that clause attached.
- *"nest-on-SKX 18.4-pt gap"* — **does not reproduce.** The tree's figure is
  **+33.4%**, the improvement the nest term needs, from the decomposition
  `share = 0.4633 × 0.5822 × 1.6067` in `CHANGELOG.md`'s 2026-08-21 entry citing
  `build/onethread-decomp-3fceaa9.log`. A share gap in points and a required
  improvement percentage are different quantities.

## The discipline every item inherits

1. **Search the tracker first, always** — including before asking a question, not
   only before filing. It has already paid once.
2. **Read the thread, not the title.** An upstream title is a symptom report; the
   comments and any linked CL may have moved the cause.
3. **A duplicate carrying a wrong causal story is worse than no filing.** When the
   bug is already open, the deliverable is a `standing-task` issue keyed to it and
   its CL — not a second report.
4. **Comment upstream only with a fact the issue lacks.** A measured delta on a
   real kernel qualifies. A second repro of a known miss does not.
5. **Failing `test/codegen` case before the fix**, committed first, so the CL
   demonstrates the miss rather than asserting it.
6. **keel is the verification harness.** Pinned before/after `benchstat` plus an
   assembly diff, archived, with the patched toolchain **read back from the
   artifact** rather than from `PATH` — a stray toolchain is exactly how those two
   diverge silently.
7. **Never write against this API from memory.** `go doc simd/archsimd`, `go doc
   simd`, and the sources under `$(go env GOROOT)/src/simd/` first; identifiers
   recalled from training are presumptively wrong, and this API has had breaking
   renames between releases.
8. **Nothing mails without #124.** The CLA is Scott's; the build, the hooks and
   the toolchain read-back are not.

## Where this leads

`golang/go#80829` is the one with a measured consequence for keel that is already
recorded as a blocked gate: [#104](https://github.com/scttfrdmn/keel/issues/104)
states outright that 55% of measured peak is unreachable on Sapphire Rapids until
it lands. If it lands, that gate is re-adjudicated against a re-measured
percent-of-peak with its own denominator — not against the old one.

It is also the item that yields if the freeze approaches, because its two halves
are independent and either can be mailed alone. A partial landing beats a dropped
one — **and that is why it leads rather than why it waits.** Putting the largest
CL first is only defensible because failing it costs a half rather than the whole;
the same property that makes it a safe thing to run out of time on makes it the
right thing to start.

## What this plan has not verified (§5 rule 12)

- **Every upstream number, title, label set and CL status here was read from the
  tracker on 2026-08-29** and is stated as of that date. `golang/go#79984`'s fix,
  **CL 803220**, was `NEW` — open, not merged — and that is the one status most
  likely to have moved by the time anyone acts on this file. Re-read before
  starting; do not trust this table's freshness.
- **No CL has been written, and no keel delta has been measured against a patched
  toolchain.** Every claim about what a fix would be worth to keel is an
  expectation, including #104's 55%.
- **The arm64 half is entirely unmeasured.** #129 has not run, so nothing here
  says whether arm64 shares any of these misses. The one statement made in the
  other direction — that `FMLA` writes its own accumulator, so the
  accumulate-in-place miss may have no arm64 counterpart — is an architectural
  reading, not a measurement.
- **The `110×` spill figure is now refuted, not merely unlocated** — updated
  2026-08-30. The 2026-08-29 version of this bullet said the figure "may exist
  somewhere this file did not look", which was the honest limit of a `grep`. The
  stronger claim came from a different question: instead of searching for the
  string, ask what `docs/spill-report.md` *measures*. It measures the named
  quantity and reads 2.54×–4.37×. A search can only ever report absence; the
  instrument reports disagreement (§5 rule 11 — the instrument adjudicates, so run
  it against the reasoning that motivated it). What remains unknown is where 110×
  came from, and that is a provenance question, not a numerical one.
