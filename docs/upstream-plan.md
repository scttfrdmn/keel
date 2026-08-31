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
| [#124](https://github.com/scttfrdmn/keel/issues/124) | prerequisites: CLA, Gerrit, Go from source — **discharged** | — | — |
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
declines to attribute them. An instrument that attributed everything would have proved
nothing. **12 of those 15 are the `X`-register scalar moves of the broadcast
round-trip** — `golang/go#80835`'s subject, in the kernel CL 1 does not cite — and the
remaining 3 are `Z` copies whose FMA is further than the predicate's two-instruction
window, not a third population; a coarser `Z`-vs-`X` cut over the same 45 puts them
with rescue (27/6/12). Both cuts sum to 45 and the 45 stands; only the sentence that
called all 15 `X` moves was wrong.

**No longer unmeasured: a patched toolchain now emits `231` here.** The step this
section used to flag as structural — that `231` removes these copies, because both
sources are read-only so nothing needs rescuing, and the destination is tied to the
addend so the loop-carried accumulator keeps its register — was reasoning, and the
instrument has now been run against it (§5 rule 11). `spill-audit`'s own library over
four listings of `Kernel6x32`'s steady-state loop, one per candidate rule, same
quantity in each column:

| `Kernel6x32` loop body | before | `z.Uses == 1` | **shipped** | unconditional |
|---|---|---|---|---|
| `Z` rescue | 27 | 9 | **0** | 0 |
| `Z` rotation | 6 | 13 | **1** | 1 |
| `X` broadcast rescue (`golang/go#80835`) | 12 | 18 | **18** | 18 |
| register copies | 45 | 40 | **19** | 19 |
| instructions ÷ arith | 219/48 = 4.56 | 214/48 = 4.46 | 193/48 = **4.02** | 193/48 = 4.02 |
| FMA forms | 48× 213 | 12× 213, 36× 231 | **48× 231** | 48× 231 |
| vector stack refs | 44 | 44 | **44** | 44 |

The last two columns are not merely equal per row: **the two rules emit identical code
for this function — an address-stripped diff of the two listings is empty over all 619
instructions** (651 listing lines, counting the FUNCDATA/PCDATA pseudo-ops) — which is
why the shipped column can carry a copy decomposition measured on the unconditional arm
without being reclassified. The empty diff is not a vacuous pass: the same comparison
against the stock arm differs in 826 lines. An earlier version of this paragraph and of
CL 1's description said "the same 674 instructions", which is not reproducible under any
counting rule the listing admits — 674 is neither the all-lines count (651), the count
less pseudo-ops (619), nor the count of lines carrying a source position (506). The
identity was real and the figure beside it was not, which is the failure mode of
quoting a count taken at a different scope than the claim it is attached to. Package-wide the condition changes exactly 22 FMAs —
`internal/vec` goes 3× 213 / 112× 231 unconditional to 25× 213 / 90× 231 shipped, and
the 22 are the 12 in `avx512Peak` and the 10 in `avx2Peak`. Nothing else in the
package sees the condition at all.

`Kernel4x32`, the loop CL 1 cites, goes 50 instructions and 12 copies to **38 and 0**
— both populations the 4/8 split named, gone, 6.25 → 4.75 instructions per FMA.
`Kernel2x32` goes 74/8 to 66/0. Under the shipped rule `avx512Peak` and `avx2Peak`
stay at 27/0 and 23/0, unchanged from stock, which is the null the measurement needs;
under the unconditional rule they do **not** — that is the next paragraph. Spills sit
at 44 in every column, so nothing here bought copies with memory traffic.

**The two middle columns are the finding, and they cost two rewrites of the rule.**

A `z.Uses == 1` guard looks prudent and is the one thing that breaks the rewrite: a
loop-carried accumulator's addend is the loop `Phi`, read by both the FMA and whatever
consumes the value after the loop, so the guard declines on exactly the shape it was
written for — and the rotation copies it left behind *grew*, 6 → 13. On `Kernel4x32`,
the loop CL 1 actually cites, that guard converts **nothing**: 50 instructions and 12
copies, indistinguishable from stock.

So the rule went unconditional, and **the unconditional form is wrong in the other
direction.** 231 ties the destination to the addend, which only helps when the addend
is what the surrounding chain continues in. `avx512Peak` is the mirror image —
`a_i = FMA512(a_i, y, x)`, where the loop-carried value is a *multiplicand* and the
addend `x` is loop-invariant — and converting it forced a copy of `x` for each of 12
chains: **27 instructions and 0 copies became 53 and 26**, 2.25 → 4.42 per FMA.
`avx2Peak` regressed the same way and now with a number rather than the word: **23 and 0
became 44 and 21**, 2.30 → 4.40 per FMA, measured off the archived unconditional listing
with the same instrument. It was asserted as "likewise" for one draft on the strength of
the FMA count alone, which establishes that the rule fired there and says nothing about
what it cost. These two kernels are keel's percent-of-peak
*denominator*, so that regression would not have surfaced as a failure; it would have
lowered measured peak GFLOP/s and silently *raised* every published percentage.

The shipped rule is therefore conditional on a helper, `z.Uses == 1 || z.Op == OpPhi`
— the addend either dies here or arrives as the phi that a loop-carried accumulator
looks like on the way in. Both clauses are load-bearing and a third, narrower reading
of the second was measured and discarded: testing whether *this* FMA supplies the
phi's own backedge argument fires only at unroll 1, because an unrolled body updates
each accumulator several times per iteration and the value closing the phi is the
chain's last FMA. Measured across three kernels of unroll 1, 4 and 4, that narrow form
left exactly one FMA per accumulator behind — 0, 4 and 12 of 8, 16 and 48 — and cost
21 of the 26 copies the rewrite removes from `Kernel6x32`.

The phi clause is permissive in a way a reviewer will ask about, and the answer is
measured rather than argued (`build/phi-arm2.log`, three probe loops audited under both
arms with `internal/spill` — the instrument that produced the table above, not a fresh
counter). A multi-use phi whose extra reader is *inside* the loop and wants the pre-FMA
accumulator has to be rescued under 231, and when the multiplicands are freshly loaded
— dead at the FMA — 213 needs no rescue at all: **1 copy under 213 against 2 under
231**. So "231 is never worse for a phi" is false, and CL 1 now says so rather than
asserting the reverse. The control identifies the cause: the same loop with
*loop-invariant* multiplicands is level, 2 against 2, because then 213 must rescue a
multiplicand too — what decides it is multiplicand liveness, not the phi's use count.
Tightening the clause to exclude that shape would mean locating the phi's other
readers, and `ssa.Value` carries `Uses int32` with no use list, so that is a function
scan rather than a field test. `rewrite.go:857` supplies the standard instead: a
load-clobber detector consulting `Loopnest`, whose comment says outright that such a
detector "does not need to be perfect" because regalloc issues the reg-reg move when it
is wrong — which is exactly the one copy measured here.

Both defects that delayed this measurement were in the instrument, not the subject, and
both were the same failure: a quantity that could not move. The first probe redirected
its `-gcflags=-S` listing into the package directory, where Go reads a `.s` as assembly
source for the package; that build had already enumerated its inputs so it succeeded,
and every build after it failed in *both* arms, whereupon a check reading "stock arm
emits no 231" passed on an empty file. The replacement's copies counter then excluded
any line containing `(` to skip memory operands — which is every line, since each
carries a source position — and it reported 0 copies for all six cells while printing
`VMOVDQU64 Z0, Z4` two paragraphs below in its own log.

**No part of this is the scalar path's precedent.** `AMD64Ops.go:834-835` defines
`VFMADD231SS` and `VFMADD231SD` and *no* scalar 213 op at all, so
`(FMA x y z) => (VFMADD231SD z x y)` is not a rule choosing between two forms — it is
the only form there is. An earlier draft of this section cited it as authority for
rewriting unconditionally; it never was, and that citation is withdrawn.

The `18` in the last two columns is the honest cost term and it is not CL 1's: those
are `golang/go#80835`'s `X` moves, up from 12 because the freed registers pushed more
broadcast sources into the high half.

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

## #124's environment half is discharged, and the harness has a green-on-nothing mode

Built from source at `603439a1c6` (`go1.28-devel_603439a1c6 ... X:simd`), `git codereview
hooks` installed, and the compiler recorded off the artifact rather than off the shell
(#58). A throwaway `test/codegen` case in CL 1's exact shape — `//go:build
goexperiment.simd && amd64`, `archsimd.Float32x16.MulAdd` — was run, driven to fail on
purpose, and deleted; the clone is clean. Four facts came out of it, and two of them
change how CL 1's case must be written:

- **Without `-all_codegen` the case is SKIPped and the run says `ok`, exit 0.** Measured
  with a *false* assertion, which is the only version of this control that decides
  anything: `--- SKIP` under an overall `PASS`, and non-verbose output is one `ok` line.
  `defaultAllCodeGen()` is `strings.HasPrefix(testenv.Builder(), "gotip-linux-amd64")`
  (`testdir_test.go:54`), so on this darwin/arm64 host every amd64 assertion is inert by
  default. **A CL 1 verification run that omits the flag proves nothing and looks
  identical to success.**
- **A bare `amd64:` assertion is checked at all four `GOAMD64` levels, separately.** The
  false control failed four times, once each for `linux/amd64/v1` … `/v4`
  (`archVariants`, `testdir_test.go:1538`); `amd64/v1:` scopes to one. So CL 1's
  `VFMADD231PS` assertion must hold at every level or say which level it means.
- **`VFMADD213PS` is emitted at `GOAMD64=v1`** — an AVX512 instruction at the baseline
  level, proven by a scoped `amd64/v1:` assertion passing. This is an independent
  derivation of the mechanism behind the invariance reported on `golang/go#80835`
  (archsimd lowering does not consult `GOAMD64`), from a different instrument than the
  hand-rolled `spill-audit` sweep — §5 rule 10 corroboration, and *only* of the
  mechanism: it says nothing about the spill idiom, which a three-argument leaf has none
  of.
- **`$(go env GOPATH)/bin` must be on `PATH` or every commit in the clone fails.**
  `git codereview hooks` writes a `pre-commit` that `exec`s `git-codereview`, and the
  binary lives in `GOPATH/bin`, which is not on this host's `PATH` — so `git commit`
  aborts with `exec: git-codereview: not found`. **This box was first ticked on the
  hook file's existence, which is the cold path; the warm path is a commit, and it
  failed.** Now proven the other way: a real commit on branch `keel-cl1-fma` produced
  `Change-Id: Ifb1d4f47…`, so the hook chain works end to end. Note `GOPATH` here
  contains a space (`/Volumes/External HD/go`), so the export needs quoting.
- **asmcheck failure output carries encoding bytes** (`62 f2 75 48 a8 c2 c3`, EVEX)
  where `-gcflags=-S` carries none. Recorded because this session had to strike
  fabricated encodings from an upstream draft for exactly that lack; the bytes have a
  source, and it is this harness.

## CL 1 exists as a failing test, committed and unmailed

**This heading is superseded and kept for the record: "committed and unmailed" was true of
`03b7769900` and of every rev through `7e50c40693`, and stopped being true on 2026-08-30
when CL 1 mailed as `golang/go` CL 824624. See "CL 1 is mailed" below.** Nothing else in
this section is retracted; it describes the state it describes.

Branch `keel-cl1-fma` in the clone, commit `03b7769900`,
`Change-Id: Ifb1d4f4766f4ac43e018ecea75ed6a3dc91cd979`, marked `[WIP, unmailed]` in its own
subject. Two cases appended to `test/codegen/simd.go`, one per half of `golang/go#80829`,
both failing at all four `GOAMD64` levels — three assertions × four levels, every failure
`opcode not found`. **Mailing is not authorized and has not happened.** The fix amends this
commit; Gerrit is one commit per CL.

The accumulate case's current output, which is the CL description's evidence:

```
VMOVDQU64	(AX), Z1
VMOVDQU64	64(AX), Z0
VMOVDQU64	Z0, Z2		<-- the copy
VFMADD213PS	Z1, Z0, Z2
VFMADD213PS	Z2, Z0, Z0
```

**The mechanism, stated correctly.** `213` computes `dst = src2*dst + src3`, so the
destination holds a **multiplicand** — not the addend, which is how my first draft of the
test comment had it. The copy therefore appears exactly where that multiplicand is still
live afterwards: at the first FMA and not the second, which the listing above shows and
which is why "one copy per FMA" is the wrong rate to quote. `231` computes
`dst = src2*src3 + dst`, reading both multiplicands, which is what an accumulation wants.
This refines rather than contradicts the existing framing that an accumulator cannot be
written in place.

`BroadcastFloat32x16` is documented **"Emulated, CPU Feature: AVX512F"**, so the `.BCST`
half touches `golang/go#80830` (CL 3). If 80830 moves first the second case must be
re-read, and its comment says so.

### The amd64 execution witness, and the scope defect in its own first attempt

asmcheck greps opcodes, not operand order, and the dev host is darwin/arm64, so nothing
in the CL's own verification ever *runs* the emitted amd64. `build/fma-witness4.log` does.
Phase A perturbs the operand order only — `=> (VFMADD231PS(128|256|512) x y z)`, the
condition held constant, so the destination is a multiplicand again and the instruction
computes `x = y*z + x` — and must fail on a host; Phase B is the witness proper. Clean: 0
gated failures, every prediction stated before the run and exact, whole-module census
exactly **213=29 / 231=132** with no `PD` forms and per-function tallies matching, and
**15 of 15** package-host runs passing on `AMD Ryzen 9 7950X3D`, `Intel i9-9960X` and
`AMD Ryzen AI MAX+ 395`. An earlier run does not substitute for it: that one restored an
*unconditional* rule, so its Phase B witnessed code that no longer ships.

Two coverage facts belong inside that number (§5 rule 12), and the second is a defect of
mine, not of the fix:

- **Phase A caught 3 of 5 packages.** `vec` and `pack` **passed** a knowingly-wrong
  permutation, because `internal/vec`'s own tests never call `FMA512`/`FMA256` in register
  form — its differential suite cannot see a wrong FMA operand order at all. The witness
  rests on the root package's tests and on `kern`/`block`, never on `vec`'s.
- **The first attempt executed and counted only `vec`/`kern`/`block`/`pack`, omitting 42
  of the 132 emitted 231s** — 32%, all of them in `internal/l1`, which has no test files
  of its own and is reached only through the root package. Same failure as the "674
  instructions" figure this CL already corrected: a count taken at one scope attached to a
  claim made at
  another. What made it look deliberate was a comment justifying the scope whose second
  clause did not follow from its first — *every FMA intrinsic lives in `internal/vec`,
  **which is why** a census of it covers all four binaries*. Rescoping the census to
  `./...` and adding the root package turned the omission into evidence: under Phase A the
  root package **FAILs**, so the third of the subject that went unexecuted is the third
  carrying the strongest witness in it.

Confirmed incidentally, by the census rather than by argument: `avx2Axpy`/`avx512Axpy`
emit **both** forms, and 4 of the 29 residual 213s are theirs, because Axpy's addend is a
freshly loaded `y` that the load form folds — the deliberate non-match the rule comment
describes, turning up on its own in a function nobody wrote the rule for.

## CL 1 is mailed: `golang/go` CL 824624

https://go-review.googlesource.com/c/go/+/824624, mailed 2026-08-30, status `NEW`, branch
`master`, +343/−0, `Change-Id: Ifb1d4f4766f4ac43e018ecea75ed6a3dc91cd979`. Confirmed from
Gerrit's own `/detail` rather than from the push output: the change number, the preserved
`Change-Id`, the owner, and the insertion count all read back as expected.

**Reviewers assigned 2026-08-31:** Keith Randall, Martin Möhrmann, Jorropo, with Gopher
Robot CC'd. No votes, no inline comments, and the only message is Gopher Robot's
first-change boilerplate — assignment is not a response, so no patchset is owed. That
boilerplate does name the freeze windows, which the plan had been treating as a budget
without stating: **May–July and Nov–Jan**. At the mailing date we are inside an open
development cycle, so the runway is real rather than nominal.

### The keel rev CL 2 is verified against is pinned at `ac0f650`

CL 2's instrument is keel's `internal/spill` audit and bench harness, and keel's own
hygiene work is editing `scripts/` underneath it. So every CL 2 verification run is made
against keel `ac0f6508e2a4ba6bcbf123e6f397c38f92650574`, and the CL 2 description's
footnote cites that rev rather than "keel at time of writing". Archived logs are never
rewritten, so a pinned rev plus its archived logs stay mutually consistent no matter what
lands on `main` afterwards. The commit recording this pin necessarily comes *after*
`ac0f650`, and changes only this file — so the instrument is identical at both, and the pin
names the earlier rev deliberately rather than chasing its own SHA.

What the pin does *not* certify: it freezes whatever state `ac0f650` is in, correct or not
(a pin certifies stability, not birth-correctness — `#143`'s rendered-share comparison is
open at this rev). That is admissible because `#143` is a defect in `gate-p5.sh`'s share
criterion, which is not CL 2's instrument; CL 2 measures instruction counts and codegen
through `internal/spill`, which the fix does not touch. **The condition on that reasoning:
if any CL 2 number turns out to come from a `gate-p5.sh` share, the pin advances past
`#143`'s fix and the description cites the newer rev.** Advancing the pin is a decision to
be recorded here, not a silent bump.

The mailed rev is `fcc582225c`, not the `7e50c40693` that was read and approved. The only
difference is the author and committer email: `~/.gitconfig` carries the GitHub noreply
address, Gerrit rejected the first push with *"email address … is not registered in your
account, and you lack 'forge author' permission"*, and the fix was a repo-local
`user.email` in the clone plus an `--amend --author`. **The tree is byte-identical to the
witnessed `f70aa0a5b4` (0 diff lines) and the message is byte-identical to the reviewed
`7e50c40693` (0 diff lines)**, so every certificate earned by those revs transfers across a
disclosed delta that touches neither the code nor the description. The global config was
left alone; the override is confined to the Go clone.

Two process notes for the next CL. First, that rejection is a *pre*-mail gate nothing in
the verification apparatus covered — the CL was witnessed for correctness on three hosts
and verified line by line, and it still bounced on an identity field, because every check
we built asks about the content and none asked whether Gerrit would accept the author. It
costs one command to ask beforehand: the author email must appear in the mailing account's
registered addresses. Second, the first `mail` invocation was piped to `tail`, so the
`$?` printed beside it was `tail`'s status and read `0` on a push that had been **rejected**
— the rejection was legible only in the text. `&&`, never a pipe, when the exit status is
the thing being reported.

Post-mail discipline now in force: reviewer comments get same-session treatment, since
review turnaround is the freeze budget; a reviewer request produces a patchset on this CL,
never a scope expansion, and nothing in keel rides along with a review response; and if a
reviewer asks for the memory-operand forms, the answer is the "Not included" paragraph
already in the description, with the follow-up CL keyed to `golang/go#80830` when its turn
comes.

## CL 1's first review: the TryBots passed, and the mechanism the objection points at cannot express the change

Read off Gerrit 2026-08-31, one `/detail` and one `/comments` query, no polling. CL 824624
is at patchset 1, status `NEW`, 8 messages, 1 inline comment.

**Two events, opposite signs.** Junyang Shao ran the trybots (`Commit-Queue+1`, 18:16) and
they **passed** — `LUCI-TryBot-Result+1` at 18:38, on revision `fcc582225c`. So the
mechanical half of review is clear. Then Jorropo, 19:03, one **unresolved** patchset-level
comment:

> This doesn't make sense, it is regalloc's job to do register allocation not rewrite.
>
> You're having to do a bad job at guessing which of the two FMA is better. What you should
> do instead is let regalloc pick, you could this trivially using my commuted regalloc CL
> series: CL 778460

**The first sentence is right and is conceded.** `FMAPrefers231` exists *because* late lower
runs before regalloc: its own doc comment says "liveness is not available here — late lower
runs before regalloc — so this asks a structural question instead". A predicate that
approximates liveness is doing with a proxy what regalloc does with the fact. If the
information were available at the right phase, the rule should not exist.

**The second sentence names a mechanism that, on its own documented terms, cannot express
this change.** The series is a three-CL stack — `778820` (`add the concept of commuted forms
to ssa/_gen`) → `778460` (`teach regalloc how to commute amd64 CMOV`) → `810740` (`teach CSE
and rewrite to canonicalize commuted instructions`). Its representation is one field, added
in `778820` to `opInfo` in `ssa/op.go`, and the field's own comment is the answer:

```go
// if not [OpInvalid], this operation can be commuted by swapping the
// first two arguments and changing to the commuted op (e.g. CMOVQCS ↔ CMOVQCC or ADD ↔ ADD).
commuted Op
```

Swapping the **first two** arguments. CL 1's rewrite is
`(VFMADD213PS x y z) => (VFMADD231PS z x y)` — a three-cycle over three arguments, because
the two forms differ in *which of three operands is the destination*: 213's destination holds
a multiplicand, 231's holds the addend. `x↔y` is the trivially-commutative part that
multiplication already gives for free; the part that matters is `arg0 ↔ arg2`, and a single
pairwise `commuted Op` has no way to say it. Extending the field to a three-argument
permutation is a change **to** `778820`, not a use **of** it, so "trivially" does not hold
for this instruction.

Series state, as facts rather than as an argument: all three are `NEW`; `778820` is
+3948/−3933 at patchset 10, last updated 2026-08-05, TryBot+1, **no `Code-Review` vote of
any sign**; and its own description says commuted instructions "are completely useless for
now" and that both consumers "currently keep the previous behavior and only optimize on
trivially commutative ops." Read via Gerrit `/detail`, not off a subject line — a title
carries no status, and this series' `ex-wait-release` hashtag is exactly the kind of state a
subject hides.

**"Guessing" is answerable, and the answer is already in the CL.** The condition was not
picked; each of its two clauses has a measured falsifier, both already in `FMAPrefers231`'s
doc comment. Those were measured when the CL was written, so under the watch protocol's rule
5 — *any* number in a reply passes the same verification a CL description does — they could
not be quoted in a reply as they stood. They are now re-measured; see below.

**The mechanism finding has a second, independent derivation, and it is the stronger one.**
The argument above reads `778820`'s field comment and observes that "the first two arguments"
is not a three-cycle. That is an argument from wording. The structural form does not depend on
the comment at all: `VFMADD213PS512` is declared `resultInArg0` with `argLength: 3`
(`ssa/_gen/simdAMD64ops.go:183`), so it computes `arg0 = arg1*arg0 + arg2`, whose value
`arg0*arg1 + arg2` is **symmetric in the two arguments the field swaps**. Swapping them is
therefore a no-op on this op, no chain of such swaps can move `arg2`, and the destination is
the only thing that distinguishes the forms. `cmd/internal/obj/x86` assembles all three —
`VFMADD132/213/231`, for both PS and PD (`aenum.go`) — while `ssa` models 213 alone today and
231 with this CL; 132 is modeled nowhere. So the family is a three-element orbit and a
pairwise field is exactly its degenerate case. Two derivations that share no premise, which is
what §5 rule 10 asks for before either is called confirmed.

**Both falsifiers re-measured 2026-08-31 at patch set 1**, `spill-audit` on the same
`internal/vec` the P2 gate audits — the repo's instrument, not a regex over `-gcflags=-S`.
Four arms, one `FMAPrefers231` body each. The run wrote `build/fma-remeasure-fcc5822.log`,
which is gitignored, so it is **tracked** as `docs/cl1-falsifiers-fcc5822.log`: these figures
are bound for a public reply, and evidence for a published claim living on one machine is the
`#114` defect this same session found one layer up in the v0.1.0 certificate.

| arm | predicate | avx512Peak | avx2Peak | K4x32 | K2x32 | K6x32 |
|---|---|---|---|---|---|---|
| **A** as mailed | `z.Uses == 1 \|\| z.Op == OpPhi` | 27 / 0 | 23 / 0 | 38 / 0 | 66 / 0 | 193 / 19 |
| **B** unconditional | `true` | **53 / 26** | **44 / 21** | 38 / 0 | 66 / 0 | 193 / 19 |
| **C** no rewrite | `false` | 27 / 0 | 23 / 0 | 50 / 12 | 74 / 8 | 219 / **45** |
| **D** narrow | `Uses==1 \|\| v` closes a 2-arg phi | 27 / 0 | 23 / 0 | 38 / 0 | 72 / 6 | 214 / **40** |

*insns / register copies in the steady-state loop.* Falsifier 1 is B against A: the Horner
shape regresses 27→53 insns and 0→26 copies, and its AVX2 twin 23→44 and 0→21 — a figure
never published before, because only the direction was predicted for it. Falsifier 2 is D
against C and A: on `Kernel6x32` the copies run 45 → 40 → 19, so the narrow test recovers 5
of the 26 the rule removes and leaves **21**, exactly the doc comment's claim, by leaving one
213 per accumulator (12 of 48 FMAs; 4 of 16 in `Kernel2x32`). C also re-derives `T-85`'s
`InsnsPerFMA` before-figures, 50/8 and 74/16.

Arm D's predicate is a **reconstruction** — the original no longer exists in the tree — so its
four pre-stated predictions were a fidelity test of the reconstruction as much as of the
claim, and all four landed. Across the whole run, 14 predictions were stated before any arm
ran — 13 exact figures and one directional, `avx2Peak` under B, for which no figure had ever
been published — and all 14 were confirmed, none missed. The run's own control: arm A measured first and last, `diff`-identical across all five
functions, which is what rules out a reading taken against a stale or half-built compiler. It
earned its keep — A1 ran against a binary stamped `f70aa0a5b4`, a *prior amend of this same
CL*, and the two commits' trees are byte-identical (`git diff --stat` empty), so the reading
was sound for a reason no mtime could have shown.

**The reply is sent; patch set 1 was not amended.** Its two numbers survived re-measurement, so
there was nothing to correct. Of the three courses the draft laid out — defend the rule as a
pre-regalloc approximation and offer to delete it when the series lands; split CL 1 down to the
six packed ops alone; or rebase onto the stack and extend `778820` first — the protocol forbade
the third on its own terms ("never a scope expansion"), and Scott ruled the first on 2026-08-31,
rejecting the split on the draft's own caveat: a half that defines six ops no rule emits is
unreachable code wearing a CL's clothes, nobody asked for it, and offering it against a
*mechanism* objection reads as motion rather than an answer. Posted 21:46:11Z as a threaded
patchset-level reply under Jorropo's comment (`in_reply_to 41a77fbb_d00fa00a`, own id
`7eea0614_d296a1eb`), left **unresolved** deliberately — the architectural objection is not
settled and it is the objector's to resolve. Confirmed from Gerrit's own `/detail` and
`/comments`, not from the POST's response body: messages 8→9, comments 1→2, unresolved still 1.

Its shape, in order: concede the principle; contest the mechanism with the structural
derivation; supply the two re-measured falsifiers against the "guessing" half; close by offering
to delete the predicate if the information belongs in regalloc. One sentence was written and
then killed before sending — an offer to hold CL 1 behind the series. Unprompted, that hands a
code owner a schedule concession he never asked for and converts a technical exchange into a
negotiation. It is one line to add back if it is ever wanted, which is the right shape for it.

**The send was gated on reconciling two CL numbers, and the gate caught a real miscue.** Jorropo
wrote "my commuted regalloc CL series: CL 778460"; the draft cited `778820` twice, including in
its load-bearing sentence. Both numbers were verified by `/detail`, the stack order by
`/changes/778460/revisions/current/related`, and the per-CL file lists by their diffs: `778820`
is the parent `778460` sits on, and it is the only member of the three that touches `ssa/op.go`
where the field is defined. So it was the bridge case rather than an error, and both sites now
give his number first and the member second — "your series at CL 778460 — specifically the
commuted-`Op` mechanism in the CL 778820 it sits on". Citing a number other than the one an
author gave for *his own series*, to that author, is this project's transcription-error class
aimed at the one reader guaranteed to notice.

**Reviewer calibration, measured rather than assumed.** `golang/build`'s owners table
(`devapp/owners/table.go:170`) lists Jorropo as a **secondary owner of
`cmd/compile/internal/ssa`** — the package this rule lives in — with 164 commits in
`src/cmd/compile` and backports to `release-branch.go1.27`. He is not on `internal/amd64`'s
list. So this is not a drive-by, and the posture the reply takes — deference on architecture,
firmness on mechanism, measurements on the merits — was chosen after that measurement, not
before it. Weight of an objection is a fact about the tracker, not an impression from its tone.

**Now pending on the owner, and the watch is re-baselined so my own message cannot wake it**:
9 messages, 2 comments, 1 unresolved, only nonzero label `LUCI-TryBot-Result+1`, **no
`Code-Review` vote of any sign ever cast**. One measured delta left unexplained: Martin Möhrmann
was a REVIEWER in the morning's baseline and is absent from the evening's. The day-six re-arm
now finds the watch by its prompt rather than by an id, because the id changes on every
re-baseline and a hard-coded one reports a healthy watch as lapsed.

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
   renames between releases. **Pass `GOARCH=amd64`**: `go doc` builds the package for
   the *host*, so on this darwin/arm64 dev host `go doc simd/archsimd Float32x16` answers
   *"symbol Float32x16 is not a type"* — the arm64 build has only the 128-bit types. That
   is the mandated instrument reporting a wrong-arch query in the subject's voice, and
   the correct name reads as a hallucination.
8. **Nothing mails without #124** — now discharged both halves: the CLA on Scott's
   direct confirmation, the build, the hooks and the artifact read-back measured above.
9. **A prerequisite is proven by its failure mode, not by its presence.** Both of
   #124's boxes had a mode that greens on nothing, and both were first ticked on the
   cold path. `git codereview hooks` writes a `pre-commit` file whose *existence*
   attests that the installer ran, a different proposition from committing working —
   every real commit aborted with `exec: git-codereview: not found`. And
   `defaultAllCodeGen()` keys on a `gotip-linux-amd64` builder prefix, so on this
   darwin/arm64 dev host **every `amd64:` assertion is inert by default**: a false
   assertion yields `--- SKIP` beneath an overall `PASS`, one `ok` line unverbose,
   **exit 0**. *A true assertion passing would have satisfied both boxes while proving
   neither.* This is §5 rule 7 arriving at this workstream — a check that could not
   have come out otherwise is not evidence, so for every confirmation name the result
   that would have falsified it. A hook file existing, a test file compiling, a `PASS`
   line printing: all cold-path facts, each satisfiable while the mechanism it stands
   for is inert. Item 7 above is the same law about an instrument's *scope* rather
   than its failure mode (§5 rule 23), and the two are checked separately.

   **Therefore every CL's verification run from here carries `-all_codegen` and is
   preceded by a deliberately false assertion shown to fail.** The negative control is
   not a nicety, it is the only witness that the assertion was ever evaluated: a
   verification run that omits the flag is indistinguishable at the exit code from a
   real one, and the test/codegen case is the dangerous one precisely because its
   failure mode is silent. A checker has three outcomes — pass, fail, and *did not
   run* — and the third renders as the first, so the false assertion's failure text is
   recorded beside the passing run or the passing run has not said which it saw.

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
