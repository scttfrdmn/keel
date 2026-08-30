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
| — | **answered 2026-08-30, duplicate of `golang/go#78753`** — X16–X31 withheld from the SIMD allocator; fixed by CL 768262 | `golang/go#80828` | [#18](https://github.com/scttfrdmn/keel/issues/18), `docs/toolchain-notes.md` T28 |
| [#144](https://github.com/scttfrdmn/keel/issues/144) | **already filed by another reporter** — legacy-SSE `MOVUPS` in `archsimd`, AVX-SSE transition penalties | `golang/go#80835` | `docs/spill-report.md` §11.1 |

The last two rows are not CLs and have no keel issue: they are **debts on filings
that already exist**, and both were ahead of every CL in urgency because they cost
one comment each. `golang/go#80828` is keel's own and was `WaitingForInfo` since
2026-08-13 — `cherrymui`: *"Could you try Go 1.27RC3 or tip? I believe it should
already work now."* T28 is the answer, it is **yes**, and the reply was **sent
2026-08-30** under Scott's authorization with the standing rule that every number
in an upstream comment passes the same verification a CL description does. That
rule earned its keep immediately: verification killed two of the drafted reply's
claims before it was sent — the CL 767380 refutation (abandoned CL) and a
*"9 refs against 25, roughly 3×"* comparison whose `25` had no provenance in the
tree or in the issue, whose own table says **44** at N=20. What went instead is a
seven-row sweep with two controls, and the correction that `golang/go#80828` duplicates
`golang/go#78753`, closed before keel filed.

Order: **#129 first** — done 2026-08-30. Then the two outstanding replies. Scott's
half of #124 in parallel, then CL 1 within two to three weeks, then the rest as
review allows. #130 does not
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

**#18's stated cause is refuted on go1.27.0, and CL 1's register-allocation half
is therefore dropped from the description rather than deferred.** T-78's read
landed 2026-08-30; `docs/toolchain-notes.md` T28 and `docs/spill-report.md` §11
carry it. Three findings, each with its own witness:

1. **31 of 32 vector registers are allocatable.** `ssa/regalloc.go:785` unions
   four masks and `specialRegMaskAMD64 = 71776114766249984` supplies X16–X31 plus
   K1–K7; only X15, the zero register, is in no mask. Identical under `GOAMD64`
   v1/v3/v4 — which **confirms the CL that landed rather than refuting anything.**
   CL 767380, whose title gates on *"v4 or higher"*, was **abandoned 2026-04-17**;
   the merged fix is CL 768262, unconditional, on `dev.simd` 2026-04-23 against
   `golang/go#78753`. A 2026-08-30 claim in T28 and in this file that the
   invariance *refutes* that framing is **retracted**: the framing binds nothing,
   and gabyhelp's "Related Code Changes" line names a CL without its status, which
   is where the mistake came from — read the thread, not the title, applied to a CL.
2. **The spill count halved: 90 → 44** vector stack refs, 5.62 → 4.56
   instructions per arithmetic op — but see the re-keying section below: on the
   isolated repro that reduction is **almost entirely a conversion into register
   copies**, not a saving.
3. **The residual is not accumulator pressure.** 36 of the 44 are
   broadcast-scalar round-trips through legacy-SSE `MOVUPS` in the inlined
   wrapper at `archsimd/other_gen_amd64.go:265`; only 3 of the 12 accumulators
   spill; and the allocator leaves X24–X31 idle while paying them. That shape is
   already open upstream as `golang/go#80835` (AVX-SSE transition penalties, 65×
   measured on Emerald Rapids), so it earns a `standing-task` keyed to that
   issue, not a filing.

**#18 was right when it was written** — its N=20 repro sat above the spill
frontier and still named nothing above `Z14`. The toolchain moved under a true
finding. It is *not* the reason `docs/neon-probe.md` §2d first gave, either: that
reasoning was the near-miss #18's own folklore note predicted, and the mask
arithmetic settles nothing on its own. The empirical finding on go1.27.0, from the
shipped listing:

| kernel | max vector register | above X14 |
|---|---|---|
| `Kernel2x32` | 14 | — |
| `Kernel4x32` | 14 | — |
| `Kernel6x32` | **23** | `Z16`–`Z23` |

`VFMADD213PS Z11, Z16, Z12` at `gemm_amd64.go:228` is an ordinary allocation, not
spill scratch, and the two clean rows are the positive control that the probe
discriminates. So `Kernel6x32` **holds 23 vector registers, needs about 15 values,
leaves X24–X31 untouched, and still carries 44 vector stack refs** — which is not
register starvation and is a stranger question than the one #18 asked.

**What CL 1 may say about registers: nothing.** The half that survives is
accumulate-in-place, cited by the 12 moves. The 90-vs-0 stack-ref contrast is not
merely uncited now, it is **stale**: 90 was a go1.26.5 reading and the shipped
shapes' 0 is the only half of it still true.

## CL 1 re-keyed against `golang/go#80835`: adjacent, and it does not move

**Ruled 2026-08-30, before a line of CL 1's codegen test was written.** T-78 put 36
of `Kernel6x32`'s 44 stack references on legacy-SSE `MOVUPS` in the inlined
broadcast wrapper, which is `golang/go#80835`'s subject, already open, by another
reporter (`achille-roussel`) and **assigned to `JunyangShao`**, who wrote *"I will
take a look!"* on 2026-08-13. The question that had to be settled first: is
`golang/go#80835` the same defect as CL 1's target, a superset, or adjacent?

**Adjacent.** One symptom, three separable causes, and no fix subsumes another:

| upstream | what it is | where it is | keel's term |
|---|---|---|---|
| `golang/go#80830` | `BroadcastFloat32x16` is **emulated**, so there is a wrapper to inline at all | intrinsic coverage | why the round-trip exists — **CL 3** |
| `golang/go#80835` | the `MOVUPS` chosen is **legacy-SSE**, not VEX | ssa→prog encoding | the AVX-SSE **price**, not the traffic |
| `golang/go#80829` | no packed `231` FMA **op**, and no `.BCST` memory-operand folding — the assembler encodes both | SSA op table, then lowering rules | 45 reg copies; and `.BCST` would delete the materialization outright — **CL 1** |

So the premise the re-keying order rested on — *"CL 1's surviving half is the
broadcast lowering"* — does not hold, and correcting it is what settles the
question. CL 1's surviving half is **accumulate-in-place** (`231`), which touches
none of the 44 stack refs; and the `.BCST` half, which does touch them, would
remove the materialization at the source where `golang/go#80835` would only make its
encoding VEX. `golang/go#80835`'s two documented manifestations are a shift count and
stack zeroing; keel's broadcast-scalar round-trip is a **third**, on a real GEMM
kernel, and that is new information the issue lacks.

**CL 1 therefore stays keyed to `golang/go#80829` and does not attach to
`golang/go#80835`.** Attaching would race an assigned maintainer on a defect CL 1 does not
fix, which is the failure the order was written to prevent — the same reflex that
made `#144` a `standing-task` rather than a filing, applied one level up. keel's
contribution to `golang/go#80835` is the 36/44 decomposition as `#144`'s single comment,
not a CL.

**And CL 1 gains its best evidence yet, which is not a µarch figure.** The
go1.27.0 register fix bought almost nothing on FMA chains because **every spill it
avoided came back as a register-to-register copy** — exactly one for one at N=14
(−12 refs, +12 copies, **0 net instructions**) and N=15 (−16, +16, **0**), one
instruction at N=16, and 11 of 79 at N=20 (T28's table, two controls on it). The
cause is precisely `golang/go#80829`: with `213`-only forms, `a = v.MulAdd(v, a)` cannot
land in `a`, so at 15 registers the pressure surfaced as spills and at 31 it
surfaces as copies. That is a **compiler-only** datum, measured, about the miss
itself — it says the register fix is close to neutral for this shape until CL 1
lands, and unlike the 109.7× it points at the mechanism it names.

### The population check the one-for-one figure was admitted on

**Ruled 2026-08-30 as the condition on admitting it: *the copies it counts must be
the copies CL 1's lowering removes*** — a true, controlled, well-stated fact off the
CL's subject being the failure mode this week was spent curing. Checked before the
figure went into the description, and **it passes**: the repro's copies and
`Kernel4x32`'s cited 12 are one population in the same two subcategories.

Every copy in each `spill-audit -v` loop body was classified by two predicates —
**rescue**, a pre-FMA copy where a `VFMADD213PS` within the next two instructions
overwrites the register the copy read or wrote (so the copy exists only because that
FMA's destination is not its addend); and **rotation**, a post-FMA copy whose source
holds a value this iteration produced, restoring the loop-header register assignment:

| loop | FMAs writing their own addend | copies | rescue | rotation | unattributed |
|---|---|---|---|---|---|
| `Kernel4x32` — shipped, CL 1's cited 12 | **0 of 8** | 12 | 4 | 8 | **0** |
| `Chains13` — repro, the N=13 control | **0 of 13** | 27 | 13 | 14 | **0** |
| `Chains14` — repro, the one-for-one row | **0 of 14** | 26 | 12 | 14 | **0** |
| `Kernel6x32` — never shipped, cited nowhere | **0 of 48** | 45 | 24 | 6 | **15** |

The 4/8 split on `Kernel4x32` is the figure's own description — *four preserving
broadcasts that `VFMADD213PS` clobbers, eight rotating accumulators at the loop
bottom* — now reached mechanically rather than by reading, which makes it a second
derivation and not a second witness (§5 rule 10). **`0 of N` in every row is the
machine-checkable form of `golang/go#80829`**: across 83 FMAs in four loops, not one
writes its own addend.

**`Kernel6x32`'s 15 is the positive control**, and it is why the other three zeros
mean something: shown a loop whose copies are a *different* population the classifier
declines to attribute them, and the 15 it flags are the `X`-register scalar moves of
the broadcast round-trip — `golang/go#80835`'s subject, in the kernel CL 1 does not
cite. An instrument that attributed everything would have proved nothing.

**The one step that is structural rather than measured, and it binds both figures
equally.** That `231` removes these copies follows from the operand form — both
sources read-only, so no rescue; destination tied to the addend, so the loop-carried
accumulator never changes register, so no rotation — but **no toolchain emits `231`
here yet, so nothing about the post-fix code is measured.** This is stated in the CL
description rather than implied, and it is not a reason to prefer the 12 over the
one-for-one figure: the 12 counts 8 rotation copies itself, so the unmeasured step is
already inside CL 1's headline number. Classification was done with a throwaway
script; the predicates are stated above because the loops are 46 and 50 instructions
and the check is meant to be redoable by eye.

### The `231` gap was already localized; today only re-dated it

I set out on 2026-08-30 to sharpen `golang/go#80829` past the shorthand "no `231`
form", and found the tree had done it on **2026-08-11**, in three places and better:
T12's row, `docs/toolchain-notes.md:852-859` (*"`simdAMD64ops.go` defines
`VFMADD213PS{128,256,512}` … and no 231-shaped op for any width … `AMD64Ops.go` does
define scalar `VFMADD231SS` and `VFMADD231SD`, and `AMD64.rules` uses them for scalar
`math.FMA` — so the 231 *shape* is already understood by the compiler, just not for
vectors"*), and `docs/spill-report.md:295`. That record names the *generator*;
mine counted its output, which is the weaker citation.

So this is **corroboration, not a second witness** (§5 rule 10), and the only thing
today's read adds is a date: the prior record is go1.27.0's predecessor, and the gap
is unchanged on go1.27.0 — `AVFMADD231PS` still an opcode with an optab entry, still
no packed `231` SSA op. The shorthand that sent me looking is at
`docs/toolchain-notes.md:42` and `:643`; the precise statement was two lines below the
first one the whole time. The live table cell above, which is what feeds CL 1, now
carries the precise form. Nothing here goes into the CL as new evidence.

### `golang/go#80835`: the third manifestation is reported, and `GOAMD64` does not move it

Sent 2026-08-30 under authorization
([comment](https://github.com/golang/go/issues/80835#issuecomment-5471826854)), framed
as a third manifestation with no priority claim and keel's fleet offered as the
assignee's verification set. Two things were measured for it that the plan did not
have:

- **`Kernel6x32`'s loop is identical instruction for instruction at `GOAMD64` v1, v2,
  v3 and v4** — four 219-line bodies that diff clean against each other, 36 legacy
  `MOVUPS` and 44 stack refs at every level. This bears on the *proposed* fix of
  selecting VEX when AVX is known available: at v3 and v4 the compiler already may,
  and does not. **Control, because an ignored variable and an invariant one read
  alike:** the same sweep moves the whole package's listing at v3 (7643 → 7625 lines),
  and the scalar probe above moves with it. This is a **different** invariance from
  T28's, which was measured on the Chains register-allocation sweep; that one does not
  transfer to the encoding question and was not cited for it.
- **The round-trip is the compiler's 128-bit spill idiom, not the wrapper.** All 18
  stores leave `X2` at 18 distinct call-site lines and all 18 reloads are attributed to
  `archsimd/other_gen_amd64.go:265`; the shipped `Kernel4x32` inlines that same wrapper
  at that same position with **zero** stack refs and **zero** legacy encodings.
- **What the comment says it does not have:** any timing. `Kernel6x32` is not shipped
  and no counter-based attribution was run, so unlike the issue's own two
  manifestations keel's is an instruction-count observation, stated as one.

## Three figures did not survive verification and are not in any CL description

Numbers reach a CL description only after being re-read from the tree, because a
figure carried from a plan is a figure nobody checked:

- *"the measured 110× µarch spill price"* — **located, real, and off-subject. The
  2026-08-30 entry that called it contradicted is retracted in full.** Provenance:
  #104's own table, `Kernel6x32` at **30.5%** of keel-zen4's peak against
  **0.278%** of keel-spr's, so 30.5/0.278 = **109.7×** — the fifth pairing #18's
  *"no log yields 110x under any pairing I can construct"* did not enumerate,
  because it used raw rates and this one is normalized per host (53.6 × 2.05 =
  109.9, the 2.05 being the SPR/Zen 4 peak ratio, also #104's). Primary source for
  every term, traced 2026-08-30: **one sweep**, `build/gate-p2-f19a977.log` —
  `[keel-zen4] 6x32 35.79 = 30.5% of peak` (line 78), `[keel-spr] 6x32 0.6677` and
  `peak 240.2` (lines 88–89). The two derivations are algebraically the same
  quotient, so they are **one witness, not two** (§5 rule 10). The same quantity
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
    was never shipped. **A second figure is admitted 2026-08-30**, about
    consequence rather than count: on the `golang/go#80828` chain repro the
    go1.27.0 register fix converted spills into register copies **one for one**,
    for **zero** net instructions at N=14 and N=15. Compiler-only, measured, with
    two controls on the table it comes from — it says what this miss costs.
    **Admitted on the population check above**, not on the controls alone: both
    figures count rescue and rotation copies in the same two subcategories, so the
    second is not a second subject. The `0 of N` column is the same statement as the
    12 — one instruction form that cannot write its addend — which is why the two
    figures belong in one description.
  - *register allocation* (the #18 half): **withdrawn 2026-08-30, not deferred.**
    The figure was 90 vector stack refs in `Kernel6x32`'s steady-state loop
    against 0 in both shipped shapes (`docs/spill-report.md:37-47`); on go1.27.0
    it is 44, its stated cause is refuted, and 82% of what remains belongs to
    `golang/go#80835`. A CL description may not carry it in any form.
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
   comments and any linked CL may have moved the cause. **This binds citation
   checks too, and `gh issue view --json body` does not satisfy it.** On
   2026-08-30 the 109.7× provenance above was checked that way, the table was not
   in the body, and the citation was declared fabricated and "corrected" in two
   files and a commit message (`3a7ad60`) — while the table sat in a #104
   *comment*, exactly as originally written. A citation check reads `--json
   body,comments`, or it is testing a different artifact than the one cited.
   **It binds CLs the same way, and a CL's title without its status is a title**:
   the 2026-08-30 claim that keel's `GOAMD64` invariance refuted CL 767380's *"v4
   or higher"* framing was read off gabyhelp's "Related Code Changes" line, which
   prints subjects and no statuses. CL 767380 was **abandoned** four months
   earlier and the merged fix is unconditional, so the refutation had no subject.
   One `curl` against Gerrit's `/changes/<n>/detail` would have said so, and now
   does before any CL is cited.
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
