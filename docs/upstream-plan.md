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
| [#129](https://github.com/scttfrdmn/keel/issues/129) | NEON feasibility probe — **done**, `docs/neon-probe.md` | — | — |
| [#130](https://github.com/scttfrdmn/keel/issues/130) | arm64 filing — **subject now fixed**: dead zero-init in emulated broadcast | new | `docs/neon-probe.md` §2b |

Order: **#129 first** — done 2026-08-30. Scott's half of #124 in parallel, then
CL 1 within two to three weeks, then the rest as review allows. #130 does not
close as not-needed: the probe found one clean arm64-specific miss and its subject
is settled, though it is the lowest-priority of the five since no keel gate is
behind it and there is no arm64 backend yet to measure a delta on.

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

**One of the ordering's three premises did not verify, one verified in a form that
disqualifies it anyway, and it stands on the third.**

- *"smallest CL first"* — `golang/go#80829` is the **largest** of the three. What
  lets it lead is not size but that its two halves are independent, so a freeze
  squeeze degrades it to a partial landing instead of nothing.
- *"the 110× spill price is its evidence"* — the figure is **real, measured, and
  keel's own**; a 2026-08-30 claim in this file that it was contradicted is
  **retracted** (see below). It is nonetheless **not evidence for this CL**, and
  for a reason no arithmetic fixes: the compiler emits *the same spilled code on
  every host in #18's table*, and what varies by two orders of magnitude is how the
  silicon prices it. A µarch response to spilled code is not a compiler-miss datum,
  so putting it in a CL description would be a well-formed claim pointing at the
  wrong mechanism — the failure `golang/go#80829` vs `golang/go#79984` already caught this
  project out on twice. **CL 1 cites the 12 moves, full stop.** The 109.7×/121.1×
  belongs in the experience report (#128) if anywhere, framed as silicon response
  and never as a miss.
- *"highest keel payoff"* — **verified**, and this is what the order rests on.
  #104, the P2 STOP, states that 55% of measured peak is unreachable on Sapphire
  Rapids until `golang/go#80829` lands. No other CL here has a blocked phase gate
  behind it.

**#18's stated cause is in question, and CL 1's register-allocation half is
excluded from the description until it is settled.** Not for the reason
`docs/neon-probe.md` §2d first gave — that reasoning was the near-miss #18's own
folklore note predicted, and the mask arithmetic settles nothing either way. The
empirical finding on go1.27.0, from the shipped listing:

| kernel | max vector register | above X14 |
|---|---|---|
| `Kernel2x32` | 14 | — |
| `Kernel4x32` | 14 | — |
| `Kernel6x32` | **23** | `Z16`–`Z23` |

`VFMADD213PS Z11, Z16, Z12` at `gemm_amd64.go:228` is an ordinary allocation, not
spill scratch, and the two clean rows are the positive control that the probe
discriminates. So `Kernel6x32` **holds 23 vector registers, needs about 15 values,
and still carries 90 vector stack refs** — which is not register starvation and is
a stranger question than the one #18 asked. #18's `Z0…Z14`-at-any-N sweep was
measured 2026-08-18, before the go1.27 port: the toolchain moved under the finding.
`build/ssa/Kernel6x32.html` adjudicates what is live at the spill points; no
mechanism goes in writing ahead of that read.

## Three figures did not survive verification and are not in any CL description

Numbers reach a CL description only after being re-read from the tree, because a
figure carried from a plan is a figure nobody checked:

- *"the measured 110× µarch spill price"* — **located, real, and off-subject. The
  2026-08-30 entry that called it contradicted is retracted in full.** Provenance:
  #104's own table, `Kernel6x32` at **30.5%** of keel-zen4's peak against
  **0.278%** of keel-spr's, so 30.5/0.278 = **109.7×** — the fifth pairing #18's
  *"no log yields 110x under any pairing I can construct"* did not enumerate,
  because it used raw rates and this one is normalized per host (53.6 × 2.05 =
  109.9, the 2.05 being the SPR/Zen 4 peak ratio). The same quantity
  `docs/spill-report.md` measures reads **121.1× on Sapphire Rapids** in #18's
  six-host table, reproduced at 120.0× on a second guest size, 2.3% spread.
  **How the refutation went wrong:** it adjudicated a *µarch* claim against
  `docs/spill-report.md`, whose host set is janus, vesta and antares — **no
  Sapphire Rapids**. "25× larger than anything the report contains" is true of the
  report and false of the tree. The adjective named the scope and the comparand
  held that scope fixed; both halves of the sentence were individually true, which
  is exactly why it read as a finding. **Do not cite it in a CL description
  regardless** — see the premise above — and prefer the structural figures, which
  serve CL 1's two *different* halves:
  - *accumulate-in-place* (the `VFMADD231PS` half): **12 register-to-register
    moves per 8 FMAs, 24.0% of `Kernel4x32`'s 50-instruction loop, in a kernel
    with zero spills** — four preserving broadcasts that `VFMADD213PS` clobbers,
    eight rotating accumulators at the loop bottom. Two independent derivations:
    `spill-audit` prints `12 reg copies`, and `docs/neon-probe.md` reaches 12 from
    the listing. This is the sharpest figure available, because it is measured on
    a **shipped** kernel and is about the miss itself rather than about a tile that
    was never shipped.
  - *register allocation* (the #18 half): 90 vector stack refs in `Kernel6x32`'s
    steady-state loop against **0** in both shipped shapes
    (`docs/spill-report.md:37-47`). **This half is excluded from the description
    until T-78's `ssa.html` read lands** — the count is solid, the *cause* is not,
    and a CL cannot assert a mechanism its author has not seen.
- *"T17 nosplit −15.5%"* — **wrong sign, and the caveat was missing.** It is
  **+15.5%** static instructions in `internal/l1`, and it *was never paid*:
  `vec.LoadPart512` still calls `archsimd.LoadFloat32x16SlicePart` directly, so
  the figure is a measured **quotation** for a change that became obsolete before
  it was written (`docs/toolchain-notes.md:1409-1417`). It may be cited only with
  that clause attached. Re-read 2026-08-30: the note still says exactly this, and
  `vec.LoadPart512` still calls `archsimd.LoadFloat32x16SlicePart` — so the
  quotation is still unpaid, not merely recorded as having been.
- *"nest-on-SKX 18.4-pt gap"* — **does not reproduce, in either form.** The tree's
  figure is **+33.4%**, the improvement the nest term needs, from the decomposition
  `share = 0.4633 × 0.5822 × 1.6067` in `CHANGELOG.md`'s 2026-08-21 entry citing
  `build/onethread-decomp-3fceaa9.log`. Re-derived 2026-08-30: the product is
  **43.34%**, matching that entry's stated share, and the middle term must reach
  0.7765 — **+33.4%** — to put the share on `CEIL_FRACTION`'s typed 57.8. So the
  *points* gap is **14.46 pt**, and `18.4` has no provenance anywhere in the tree
  under either reading (grepped tracked files and `build/`; the pattern's live
  matches are unrelated GFLOP/s figures). A share gap in points and a required
  improvement percentage are different quantities, and this charter figure is
  neither.

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
- ~~**The arm64 half is entirely unmeasured.**~~ **VOID 2026-08-30** — #129 ran;
  the result is `docs/neon-probe.md`. Its verdict: accumulate-in-place and the
  15-register limit are **absent** on arm64, emulated broadcast is **present and
  worse** (3 instructions against amd64's 1, from identical Go source), and the
  wrapper anchor is **present with the same cause at 4 bytes instead of 1**. The
  architectural reading recorded in this bullet — that `FMLA` writes its own
  accumulator — reached the right conclusion by incomplete reasoning: the miss is
  absent because `simdARM64.rules:242` rotates the accumulator into arg0, not
  because the instruction set forces it. `VFMADD231PS` also writes its own
  accumulator and amd64 still misses it, so architecture proposes and the lowering
  rule decides. Still unmeasured on arm64: anything timed, and any tile but
  `{MR:4, NR:16}`.
- ~~**The `110×` spill figure is now refuted, not merely unlocated.**~~ **VOID
  2026-08-30, same day, by the condition attached to the ruling that accepted it.**
  Ruled: grep the published surfaces before closing the figure out. The grep found
  it had never escaped — it *originated* in keel's own measurements (#104, #18), and
  the refutation was the error. Retracted above with the arithmetic. Three lessons,
  all of them about this bullet rather than about the figure:
  - **A verified-instrument reading is scoped by its sample.** §5 rule 11 says the
    instrument adjudicates; it does not say every instrument may adjudicate every
    claim. `docs/spill-report.md` cannot see Sapphire Rapids, so on a *µarch* claim
    the honest verdict was "my instrument cannot see this", not "contradicted".
  - **The escalation from "unlocated" to "refuted" was the mistake, not the fix.**
    The 2026-08-29 bullet's `grep`-limited honesty was correct. Replacing a weak
    true claim with a strong false one felt like progress because the new claim came
    from an instrument rather than a search.
  - **A three-state resolution was available and was not used:** real, measured, and
    *off-subject*. Reaching for refuted-or-confirmed on a two-state axis is what
    made a scope error look like a numerical one.
