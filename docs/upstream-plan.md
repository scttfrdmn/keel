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
| [#125](https://github.com/scttfrdmn/keel/issues/125) | CL 1 — emulated broadcast + inlined-wrapper NOP anchor | `golang/go#80830` | [#17](https://github.com/scttfrdmn/keel/issues/17) |
| [#126](https://github.com/scttfrdmn/keel/issues/126) | CL 2 — adopt CL 803220, SIMD ops in LICM | `golang/go#79984` | [#54](https://github.com/scttfrdmn/keel/issues/54) |
| [#127](https://github.com/scttfrdmn/keel/issues/127) | CL 3 — MulAdd accumulate-in-place + `.BCST` folding | `golang/go#80829` | [#20](https://github.com/scttfrdmn/keel/issues/20), [#18](https://github.com/scttfrdmn/keel/issues/18), [#104](https://github.com/scttfrdmn/keel/issues/104) |
| [#128](https://github.com/scttfrdmn/keel/issues/128) | experience report on the graduation thread | `golang/go#73787` | the whole tree |
| [#129](https://github.com/scttfrdmn/keel/issues/129) | NEON feasibility probe (no upstream artifact) | — | — |
| [#130](https://github.com/scttfrdmn/keel/issues/130) | arm64 filing, **conditional on #129** | TBD or none | TBD |

Order: **#129 first** (it needs nothing and nobody), Scott's half of #124 in
parallel, then CL 1 within two to three weeks, then the rest as review allows.
#130 may close as not-needed; that is a result, not a failure.

## Two things this plan gets right that a first draft got wrong

**The CL numbering was mis-keyed, and the fix was to read the tracker.** The plan
this file descends from listed "CL 1: `golang/go#80830` embedded-broadcast
lowering".
Embedded broadcast — `.BCST`, `VFMADD231PS` — is not in `golang/go#80830`; it is in
`golang/go#80829`, the *third* CL and the largest. Both of the original
descriptors named halves of one issue. Caught by searching the tracker before
filing, which is `CLAUDE.md`'s standing order and the same order that turned T18
from a new register-allocation finding into a duplicate of an already-open
`golang/go#79984` with a fix CL in flight. **If the intent was for embedded
broadcast to go first, #125 and #127 swap, and the "smallest first" rationale
swaps with them.**

**Three figures did not survive verification and are not in any CL description.**
Numbers reach a CL description only after being re-read from the tree, because a
figure carried from a plan is a figure nobody checked:

- *"the measured 110× µarch spill price"* — **excluded.** `grep 110` in
  `docs/spill-report.md` matches only the issue reference `#110`. Probably a third
  instance of this project's number-namespace collision.
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
one.

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
- **The `110×` spill figure was searched for and not found**, rather than shown to
  be wrong. It may exist somewhere this file did not look; it is excluded because
  an unlocated number cannot be cited, which is a weaker claim than refutation.
