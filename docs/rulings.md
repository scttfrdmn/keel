<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Rulings behind the testing rules

`DESIGN.md` §5 states what must be done. This file records *why*, one entry per rule,
dated, with the incident that produced it.

The split was made 2026-08-16. §5 had reached 2,275 words across nine rules, of which
rules 6–9 were most of the bulk and almost all of the narrative: each carried a full
incident history — occurrences, authors, the coincidence that supplied a false
confirmation. That reasoning has independent value and is not deleted; it was in the
wrong file, because a reader looking up an operative constraint had to read a case
report to find it. **Every ordinal in §5 is unchanged**, so no citation moved. Rules 1–5
have no entry here: 1–4 are one line each and have held for the whole project, and rule
5's ruling is still inside it because its 2026-08-16 amendment is live work.

Rules **10, 11 and 12** were added 2026-08-18 (issue #36) and are entered here from the start,
appended after rule 9 so no ordinal moved. 11 and 12 were one fused rule for the length of a
single commit; the split and its reason are recorded under rule 11.

What an entry here is *not*: authority. A gate cites `DESIGN.md`, never this file. If an
entry and its rule disagree, the rule governs and the entry is stale — which is the
same precedence `CLAUDE.md` has against §7.

---

## Rule 6 — a gate red for the wrong reason

*Ruled 2026-08-13. Issues #32, #33.*

A gate's product is verdicts, and a verdict whose cause is misattributed corrupts the
record in either colour: it is remembered as what it claimed to measure, and what it
actually measured is never revisited.

Two P3 defects landed on one run and each produced a plausible verdict naming the wrong
cause — a `-bench` filter that never ran the reference benchmark (#32), and a rate parser
that read a theoretical-peak provenance line as a measurement (#33). Neither run looked
broken. The structural consequence, not merely the lesson, is the declare-then-check
contract now in the rule: `bench_stat` printing nothing is otherwise indistinguishable at
every call site from a benchmark that ran and reported nothing under the unit asked for.

## Rule 7 — a check that could not have come out otherwise

*Ruled 2026-08-14. Issues #48, #8 (with toolchain note T18).*

Two confirmations in one campaign turned out to be structural rather than evidential, and
both were caught in-house rather than by a reviewer, which is what makes this a named trap
instead of a recurring surprise.

**(a) The tautology trap (#48).** A "second, independent" check on per-call flatness was
the ratio `4.00 ÷ (per-call rise)`. A total is per-call cost times an exact call count, so
the ratio is arithmetic performed on the very quantity it claimed to corroborate. It
checked to the digit on both hosts and carried no information the per-call column did not
already have. A number that agrees is not a check if disagreement was impossible.

**(b) The unvaried control (#8 / T18).** Two repro arms differed in **two** ways at once —
where a definition sat relative to the loop, and whether it arrived through a callee — and
the effect was attributed to the salient difference. Three variables were dutifully held
(accumulator count, loop count, masked tail) and every one of them was irrelevant; the one
that mattered was never varied alone.

The compiler-forensics corollary comes from (b): the allocator reusing a register holding a
dead value was right, and the missing hoist that left the value dead there was the defect.

## Rule 8 — a summary is a cache with no invalidation protocol

*Ruled 2026-08-15, extended the same day. Issues #21, #65, #66.*

Three occurrences across two authors, the third produced while the rule was being written.
vesta's *unblocked* microkernel percentage was carried into a sentence about its *blocked*
`Sgemm`, which is a different denominator; `KERNEL.md`'s loop ratio needed explicit
re-labelling during the layout ensemble for the same reason; and a session summary's
`gate-p5: 64 PASS / 7 FAIL` was published in prose and repeated once before the log was
consulted, which says `64 PASS / 5 FAIL` — five being the four unmeasured `-race` criteria
plus the carried gate-p4, with no sixth or seventh to name. The mechanism never varies:
prose drops the qualifier that made the number specific, and nothing downstream can tell it
is missing, because a figure with the wrong denominator is still a well-formed figure.

**The same-quantity clause has its own two occurrences, and they cost more than the trap it
prevents.** Rule 8 instructs distrust of the published figure and trust in the fresh
recomputation, so recomputing a *different* quantity from the right log yields a well-formed
number that disagrees with a correct publication — with the rule having pre-committed the
writer to believing theirs.

The #21 comment recomputed 8-thread **rates** against a table captioned "change in the
8-thread **scaling ratio, net of CI**", declared several cells wrong, and published that.
All twelve were correct, and the withdrawn `16.2 / 6.7 / 10.9` was correct too. One pair of
logs answers "how much did Sgemm gain on janus" three ways — +6.22% (8-thread rate, 466.2 →
495.2 GFLOP/s), +4.22% (raw ratio, 6.211 → 6.473), +3.20% (ratio net of CI, 6.090 → 6.285) —
because the change moved the 1-thread denominator too (75.06 → 76.50). A coincidence supplied
the false confirmation: Ssymm/janus's ratio delta is +6.21% and Sgemm/janus's rate delta is
+6.22%, so a matching figure read as a copied cell.

The second occurrence was caught before publication: the #66 boost-off smoke figures differ
by aggregation — 19.6% and 5.91× under the mean against 19.8% and 5.90×→6.32× under the
median — and `scripts/bench.sh` fixes the convention at *"the median net of benchstat's
reported confidence interval"*. Hence **aggregation is part of a quantity's definition**.

The rule's self-binding clause is why the third occurrence above is a `PASS`/`FAIL` tally
rather than the instance the ruling originally named: a #65 correction said to have taken its
Sgemm gains from the wrong row does not survive the check — those figures were right as rate
deltas and merely not the quantity the table published — while a genuine third arrived
independently, from the same author, in the same session.

## Rule 9 — a citation is a claim about where the grounds are

*Ruled 2026-08-16. Issue #85; pinning retired the same day.*

The distinction between the three instruments is not hypothetical. 18 citations of `§5.4` were minted wrong in the same commit that appended the methodology rule as item **5** (`4643b63` — citation-lint:quote(5.4)).
No renumber ever happened, so every pin would have passed forever, while every one of the
18 sent readers to the benchmarks-are-tests rule instead (#85). Two further sites were
wrong inside populations that resolved perfectly. (That first line is deliberately
unwrapped: the marker must share a line with the citation it suppresses, and wrapping it
away is the hazard control T2 exists for — which this file tripped on its first draft.)

That is what "a pin certifies stability, never birth-correctness" is a summary of, and it is
also why **pinning was retired on the day the rule was written**: it was 663 lines across
three files, run on every push, guarding a renumber this document has never had, while the
defect that actually occurred was out of its reach by construction. A renumber is now caught
by review. The instrument that could have caught the real defect — mint-verification — is by
the same argument the one that cannot be automated, and it was run once, over all 112
DESIGN-bound sites, with its census recorded in `docs/citation-externals.txt`.

The worked example behind consequence (a): `DESIGN.md`'s own §7 citation was corrected
against `scripts/gate-p5.sh`'s, because the two sites made the same argument and the rule
bodies — not the majority — picked the winner.

## Rule 10 — agreement across N sites is one witness

*Ruled 2026-08-18. Issue #36; retraction at `bb1caf2`.*

Deleting `internal/block.expandSym` was published in six places — `internal/pack/pack.go`'s
package doc, `internal/block/tri.go`'s *"Why the pack reflects A"* section, `bench/scale_test.go`'s
`Ssymm` row, `bench/symm_test.go`'s fixture comment, the CHANGELOG entry, and two comments on
#36 — as removing "a serial region" and with it an Amdahl term from a routine whose parallel
scaling is a gate-p5 criterion. It was false. `expandSym` ran under `par.Run` over A's rows from
`175098d`, the parallel-nest commit, and `git log -S'par.Run(d, func' -- internal/block/tri.go`
returns exactly two commits, which settles the question in one line.

The six sites read as corroboration and were not: every one was written from the first, so they
agreed by construction. Every *figure* beside the claim was right — 67 MB at d=4096,
16,904,645 → 127,365 B/op at d=2048, the deltas per row — which is the same shape as the
Sapphire Rapids mechanism retraction on #104 two days earlier, and is why rule 8's apparatus
could not touch it: nothing here checks "and this is why."

Two artifacts had already refuted it and neither was consulted. The A/B's own thread-count
behaviour — the n=1 delta *falling* from 2.55 ms at one thread to 1.15 ms at eight, which is what
a parallel pass does — and the P5 umbrella's own task list, which said in words why `expandSym`
was parallelised: *"a serial pass worth 5% of the serial nest is 29% of an 8-way one."* It
surfaced only because ticking an item off that list meant reading it.

The contrast worth keeping is that independence can be arranged on purpose, and where it is, the
agreement means something: `gate-p4.sh` and `gate-p5.sh`'s delegated-gate tallies are documented
as **two independent readers of one vocabulary**, and the §4/P2 threshold constants are
duplicated across three gates rather than factored, on the stated grounds that two independent
statements of a threshold is what makes a divergence visible. Propagation is the failure mode;
duplication chosen for independence is the remedy.

## Rule 11 — an instrument's output overrules its author's claims about it

*Ruled 2026-08-18. Issue #36, commit `9362597`. Split from a fused rule the same day: see rule 12.*

`BenchmarkSymmNarrow` was written to find the crossover in n below which reflecting the
symmetric operand at pack time beats expanding it. Its first run refuted two claims in its own
doc comment. **A flop share is not a time share:** every shipped kernel has `NR = 32` and the
nest zero-pads n out to it, so a call with n < 32 buys one tile of column work whatever n is —
n=1 measured 35.9× the n=1024 row's per-column time, close to the 32 the padding predicts, so
the reflection was ~5% of the time at n=1 and not the ~50% its flop share suggested. **And the
wide-n row is a control at GOMAXPROCS=1 only:** at 8 threads every row moved, n=1024 included.
Both corrections went into the fixture's own comment at `9362597` rather than into a later note,
and the cause of the 8-thread movement is recorded as *not established* rather than filled in.
This is the cheap version of being wrong: the fixture was the first reader of the reasoning that
motivated it.

Why this is worded as *adjudication* and nothing else: the first draft fused it with what is
now rule 12, and the fusion read well and cited badly. A rule invoked from two unrelated
situations can be aimed at the wrong one — "instruments correct the experimenter" quoted to
excuse a coverage omission, or the disclosure clause quoted at a claim that was simply
unrun. Rule 10 needed its contrast clause for the same reason, so the pattern is now twice
observed: in this project a rule earns its keep at citation time, and precision of citation
beats economy of rules.

## Rule 12 — a coverage claim enumerates what it cannot see

*Ruled 2026-08-18. Issue #36, commit `2b54d1d`. Split from rule 11 the same day.*

`TestSymPackMatchesExpansion` was driven by five
mutations of `storedRun`; four are killed — dropping the reflection in either nest, swapping the
two uplos, widening the stored range by one — and the fifth, shrinking that range by one, passes
and **cannot** fail: it moves only the diagonal element, whose reflection is its own address, so
the strided path reads the byte the copy would have written. Reporting "4/5" and stopping would
have been true and useless; reporting "5/5" would have been a lie. The test says which mutation
it is and why it is unkillable, so a later reader does not "fix" the test to catch it. The
boundary is off-by-one-tolerant for values and not for cost, which is a fact about the diagonal
rather than a gap in the check.

**Amended 2026-08-21 — clause (c), an unfixable hole is not a debt.** Issue #6, ratifying the
correction of a note Scott had filed himself. The archive re-adjudication under `CEIL_FRACTION`
resolves a `gnr`-class row only if its true share reached **153.9%**, i.e. never: the recoverable
`8 × 1T` denominator is that much looser than gnr's real ceiling. That was filed as a post-tag
refinement for #113's re-measured ceiling row — a *forward-looking* instrument pointed at twelve
finished runs. No future action can re-measure them. Scott's ruling: "a permanently unfixable
limitation filed as a debt is a lie about the future — it goes inside the number instead," because
a debt entry promises eventual payment and an unpayable one reads as *known, scheduled* where the
truth is *known, permanent*.

Two things this does not do. It does not retire the debt that was correctly scoped: `gnr` and
`spr` carry ceilings dated to `651d1bd` with nothing re-measuring them, and #113's row is the live
denominator waiting on #111's readmission ruling — a real contingency, honestly conditional. And
it does not make the enforced bar the subject: `k` appears only in the retrospective proxy, while
`CEIL_FRACTION` divides by each host's *measured* 8-thread ceiling, where `k` cancels and the bar
(58.5% when this was written, 57.8% after that day's repair to the site computing its input) is
vacuous for no host it judges. The same session's other error was the mirror of this one —
attributing the loose `k` to `zen4` because zen4's name sat beside the 35 resolving rows, when
zen4 owns them *because* its proxy is the tightest of the three (0.765 against gnr's 0.380).
Adjacency is not attribution.

## Rule 13 — two cost terms are comparable only at their rates

*Ruled 2026-08-19. Issue #37, comments 5335770326 (the claim) and 5336454295 (the refutation);
commit `ce66188`.*

`Trsm`'s `MB` was argued down from two exactly-counted terms moving in opposite directions —
diagonal-solve flops `n·m·(MB+1)` rising, rank-update repack `~n·m²/(2·MB)` elements falling —
and the conclusion published on #37 was "all three terms say `MB = 64` is too small." The sweep
found the opposite direction on all three hosts: `MB = 32` is 17–33% faster and full time rises
monotonically with `MB`. **Both predicted directions were right and the ranking was wrong.**
From `MB` = 64 to 512 the repack argument saves 0.0745 s while the solve costs 1.094 s more,
because the solve runs at 1.3% of the host's kernel peak and the rank update near it. A flop
count and an element count are not commensurable, and neither is a time.

The failure mode is specific and worth naming: the countable term gets the weight *because* it
is countable. There was a closed form for the repack and none for the solve's rate, so the
argument organised itself around the half that could be written down — and then published a
direction, the one thing an un-rated comparison cannot supply. One arm of a sweep settled what
three paragraphs of arithmetic got backwards.

Clause (b) is Scott's ruling of the same day, and it is what saved the shipped default from the
correction: the sweep also shows `MB`'s optimum is a *function* of the solve rate — 8× faster
moves the best point to 128, 16× to 256 — so retuning to 32 now would tune the parameter for
the slow-solve world and be re-reversed after #37, putting two tuning flips in the published
history of one routine, each needing its own judged evidence. `MB` stays 64, and the interim
17–33% is priced and recorded on #37 as a decision rather than left to read as an oversight.

## Rule 14 — a defect's severity is a function of its deployment context

*Ruled 2026-08-19. Issue #106, comment 5349633810; directive #109.*

#106 is a classifier defect: `host_admission` resolves a bare-metal host's `instance=none`
through the `case`'s default arm to `correctness`, stripping judged standing from the three
lab hosts the v0.1.0 record rests on. It was filed **"latent, not active"** and that grading
was correct when written — only gate-p2 consulted the class, and the two-tier directive had not
yet been ruled, so the lab fleet was framed as retired and no lab host would run under the code.
The v0.2.0 directive (#109) made the lab a *signing* tier, and the same grading became false
with **not one byte of the defect changed**. Its neighbouring line died the same way: "no live
bare-metal host exists to drive this on" was a fact about the fleet, so the widening arm now
gets driven on three live hosts and only the must-stay-restrictive arms remain fixtured.

What makes this a rule rather than a note is what the stale grading would have caused. Both
dead premises sat in the issue body reading as current, and "latent" is exactly the word that
sequences an issue behind other work. Re-admitting the fleet against the unrepaired classifier
would have **falsely demoted all three hosts by a bug's signature instead of by evidence**,
days before the tag, manufacturing the tag-week failure the sequencing exists to prevent —
rule 11's violation moved up to the gate layer, since the judge would have ruled before its
known defect was repaired.

One condition on the fix, from the same ruling, because the defect's shape is half-correct:
`instance=none` falling through to `correctness` is the right **default** and the wrong
**classification**. Unknown provenance failing closed is what a `case` default is for. So the
fix gives bare metal its own named arm with its own evidence requirements — no `hypervisor`
flag, governor asserted `performance` — and the default arm stays restrictive for genuinely
unrecognised hosts. Widening the default to rescue the lab would trade a false demotion for a
false admission; naming the class keeps the asymmetry pointed the safe way.

## Rule 15 — a conservativeness claim about an instrument is a testable claim

*Ruled 2026-08-19. Issue #110, Scott's Q9 ruling.*

The finding's cleanest statement is Scott's: **the instrument's quantum exceeds every margin
it adjudicated.** benchstat's CSV `CI` column is a display string rounded to a whole percent,
and one rounding step is worth 0.1386 on a 7.0 scaling ratio; the margins that column decided
were 0.011 and 0.081. Every verdict within one quantum of the floor — in either direction —
was decided by the display formatter, not the silicon, and the `7.0098 -> 7.0101` flip is
that fact caught on camera.

The rule is about the *second* defect, in the writing rather than in the arithmetic. T21
reasoned from the meaning of a `± 0%` reading to the conclusion that raising `-count` was
inert on such a row, and that this made the reading **less** likely to be undecidable. That
is a claim about the **sign of the instrument's error**, and it was wrong: the rounding band
a reported `0%` stands for can straddle the bar, so such a reading is *more* likely
undecidable, not less. It was published without being run against the adjudicator it
described. In Scott's words: *"'safe direction' asserted from reasoning, inverted by the
instrument's actual behavior, published without being run against the thing it described.
**Direction-of-error is a measurement.**"*

Why a safe-direction claim is the *worst* place to skip rule 11, rather than the most
forgivable: conservativeness is self-recommending. A bound believed pessimistic cannot be
dangerous, so nobody asks it for evidence — and if the sign is in fact backwards, the
certificate that says "conservative" is precisely what stops the next reader looking. The
error is not merely undetected; it is protected by its own reassurance.

The archive adjudicates the scope, which is the rule applied to itself. Two prose sites
asserted a **measured** interval was zero-width (`DESIGN.md` §4, `scripts/gate-p3.sh`'s
instrument-exercise header), reading width 0 off a reported `± 0%`. The gate logs refute the
categorical form directly: janus's own class-selecting interval reads `[1.014x, 1.034x]`
around 1.026 in one archived run — same host, same comparison, non-zero width, in a run whose
CIs did not happen to round to `0%`. What survives is the *quantitative* claim, and it now
has a denominator: 1.10 − 1.034 = 0.066 of margin against roughly 0.010 of quantization
width, about six quanta, so `KEEL_INSTRUMENT_WIDEN_CI` remains necessary to reach the
three-state renderings — for a measured reason instead of an impossibility one.

One nearby reading is worth recording because it is the closest call the archive holds and it
came out clean for a reason nobody planned. antares read a ceiling spread of `[1.077x, 1.100x]`
against the 1.10 bar — flush with it, well inside one quantum. The class does not move: above
the bar the collapse rule (`why=samemixanyway` / `falsifiedanyway`, added 2026-08-16 for an
unrelated reason) yields `fma-bound`, and below it the falsification test yields `fma-bound`
too. Only the `why=` label could flip, and a label is what a reader uses to understand the run
rather than what a criterion divides by. So a rule written to stop an `UNMEASURED` the data
settles is also what kept the quantization defect off this verdict — worth saying plainly,
because it was luck in the sense that no one checked, and the checking is the rule.

And the scope has a floor as well as a ceiling. Five further `zero-width` sites
(`scripts/roofline.sh` ×4, `scripts/roofline-test.sh`) describe a **fixture given no bounds**,
which the input format *defines* as zero-width. Those are constructions, not readings, and
they are correct; correcting them would assert something false about a definition. One word,
two meanings, and only one of them was ever a measurement — which is why this ruling names
sites rather than a pattern.

## Rule 16 — a published reference is an estimator, never a draw

*Ruled 2026-08-21. Issue #6, Scott's ruling on the README re-measure red.*

`gate-p5.sh`'s criterion 9 compares each README rate to this run's median against a 5% band.
On `33de3b2` it convicted two zen4 rows, and the finding is not that they drifted:

| row | six-run history, same physical instance | README | this run | gate `d` |
|---|---|---|---|---|
| `Ssymm/threads=8` | 610.8 615.4 618.4 622.8 643.4 **654.3** | 654.3 | 615.4 | +6.32% |
| `Strsm/threads=1` | 35.66 36.06 36.11 36.32 37.12 **37.61** | 37.61 | 35.665 | +5.45% |

Both published values are the **maximum** of their own history. That is not cherry-picking
after the fact — it is what single-draw publication does by construction. Any one run lands
somewhere in its spread; the check thereafter measures the reference draw's altitude rather
than the code, with a sign set by luck. Scott's statement of it: *"a high draw manufactures
future reds exactly as a low draw would manufacture future flattery."* The flattery is the
dangerous half, because a red gets investigated and a green does not.

Checked against the intervals the two runs actually measured — the reference's read from
`build/gate-p5-651d1bd.log`, the revision both rows were published from — neither
disagreement resolves:

```
zen4/Ssymm/8   654.3(+/-3.9%)  615.4(+/-3.2%)  |diff| 38.90  half-widths 45.21  TIE
zen4/Strsm/1   37.61(+/-5.7%) 35.665(+/-5.2%)  |diff|  1.95  half-widths  4.00  TIE
```

**The ratified repair is median-over-archive, and it is a repair rather than an amendment.**
The band is untouched: 5% stays 5%, applied to an honest center instead of a peak. The
criterion's standard was always "today within 5% of what this host does", and a
max-of-history reference was never a faithful implementation of that — so restoring the
estimator needs no standard that predates the result, because it *is* the predating standard.
Each regenerated row states its estimator: median of N archived runs, the runs named.

**What was refused, and why it is the reusable half.** The obvious-looking fix was to make the
check interval-aware, on the precedent of the scaling bars and of rule 5's `a tie is not an
order` (2026-08-20) — a standard that genuinely predates this result. It was drafted,
computed, and rejected on direction. All four rows in question tie at their archived
intervals, zen4's two reds included, and nothing is newly convicted; on the data in hand the
change *only acquits*. Scott: *"a correction that on the data in hand only acquits is not a
two-way improvement, it's a loosening wearing rigor's vocabulary."* That is rule 15's
sign-of-error discipline one level up, and the never-weaken order reached before the change
shipped rather than after.

The first computation of that same test was itself wrong, in a way worth recording: it assumed
the reference carried *this* run's interval, which is the tighter one, and so reported zen5's
two `Ssymm` rows as newly resolving. Reading the real archived intervals turned them into ties
as well. Compute, catch the source error, recompute, then refuse your own now-correct fix on
direction grounds — the flattery asymmetry applied twice in one sitting.

**What stays unmeasured, and is not claimed.** The 7.1% peak-to-peak spread in that six-run
history mixes measurement noise with real code change across six revisions. The only
same-revision repeat the archive holds — `ce43bca` against its own rerun, `Ssymm/8T` 618.4 vs
622.8 — spread 0.71%, which does not decompose the 7.1% and is one pair. So the decomposition
is unmeasured, and the ruling rests on the estimator argument alone, which needs no
decomposition: sample size one is the worst estimator available whatever the spread's cause.

## Rule 17 — a criterion may not judge a host its reference artifact predates

*Ruled 2026-08-21. Issue #6, Scott's ruling on the BASELINE-REGISTERED class, ending "Build it."*

`gate-p5` rendered five reds on `keel-skx` at `33de3b2`. Three were a real deficit and one
was a noise-margin miss, but the fifth was structural:

```
FAIL [keel-skx] README.md publishes no row for 'Intel(R) Xeon(R) Platinum 8124M CPU
     @ 3.00GHz', so this host's numbers are either unpublished or published under a
     CPU it does not have
```

A README row is *born* from a judged run. skx's first judged run is the one issuing this
verdict, so the row it is convicted for lacking could not have existed — the criterion is
reading the host's admission date. Scott's structural definition covers the whole family:
**any criterion whose reference artifact — bar derivation set, README row, archived
baseline — predates the host's admission.** Convicting skx at the README criterion for that
absence is *rule 16's error relocated one criterion over*: in both cases the check measures
a property of the reference rather than of the code.

### The three states, and why they come from tracked state rather than a flag

Both facts are read at `(cpu_model, era)`, from two tracked files and nothing else.

| registry row | witness row | verdict | why |
|---|---|---|---|
| absent | absent | `BASELINE`, green-compatible | genuine newness; the run emits the candidate reference formed from its own numbers, and the witness row that will spend it |
| absent | **present** | `FAIL`, naming the unmet registration | absence is no longer newness, it is an obligation someone did not land |
| present | either | judged at `baseline − margin`, derivation printed | the host has a reference of its own, from this era |

The middle row is the whole design. A `BASELINE`-shaped exemption that could be renewed
would be a permanent green for any host nobody got around to registering.

**VOID, and the correction is the interesting part.** The first version of this section said
the closure was structural rather than vigilant — *the run that renders `BASELINE` creates
the prior log that forbids it next time* — with the witness being a glob over
`build/bench-gate-p5-*-<host>-*.txt`. That is elegant and it is wrong outside the machine it
was written on. `build/` is gitignored, so on a fresh clone, on CI, or on a second
operator's machine no host has ever been judged, every host sits in the **top** row forever,
and `BASELINE` renews on every run. **A per-machine witness defeats single-shot exactly as
thoroughly as the permanent exemption this class was built to kill**, and it does it more
quietly, because it fails only for readers who are not the operator. Filed as #114 and fixed
in the same commit as the era amendment, on Scott's direction.

The witness is now `scripts/judged-runs.tsv`: tracked, keyed `(cpu_model, era)`, proposed by
the gate beside the baseline row it spends, landed by the same reviewed commit. What that
costs is stated rather than absorbed — **automatic-and-invisible traded for
reviewed-and-visible**. A session that lands neither row leaves the host unregistered and
the next run renders `BASELINE` again; that repeat is not silent, because the unregistered
state is printed as a debt line on every run until it is discharged. The old scan could not
be skipped but could not be read either.

**Keyed on the CPU model, not the hostname** — the one deviation from the ruling's wording
(`keyed (host, era)`), taken deliberately rather than resolved silently in either direction.
The registry next door keys on the exact probe string because that is the key the criterion
consuming it greps: one key shared with the criterion, so the artifact can never disagree
with the check about who a host is. A hostname key would break that in the direction that
matters — renaming `keel-skx` to `keel-skx2` would produce no witness row and hand the host
a second `BASELINE`, which is exemption-renewal-by-rename. Keying on the model closes it and
costs nothing, because a *second* host of an already-judged model finds a registered row and
is never in `BASELINE`'s path. `scripts/baseline-test.sh` drives the rename on purpose,
by moving the host column and showing the answer does not move.

**Also VOID:** the claim that skx is in the middle row today and renders `FAIL`. Two things
moved it back to the top row, and neither is a weakening. The witness index is tracked and
empty, so no host has a witness at all; and the current era is `pinned8`, in which no
reading has yet been taken. skx renders `BASELINE` on the transition run — as does every
other host — which is the era clause operating, not the exemption being renewed.

### Two boundaries

**The instrument proposes references and never writes them.** An instrument that mints the
reference it will judge against has certified itself, so `gate-p5` emits a fully formed
candidate row to `build/baseline-candidates-<rev>-<run>.tsv` and stops. Landing it is a reviewed
commit act, exactly as `CEIL_FRACTION` and every README row were. `scripts/baseline-test.sh`
asserts that **both** tracked files — the registry and the witness index — have no data rows
and were not written, which makes this the one property whose violation is not a wrong number
but a wrong constitution. Both, because the witness is now half the decision, and a gate that
could write it could spend its own exemption.

**A candidate emitted from one run is not landable as it stands** — rule 16, applied to
this mechanism by its own author's instrument. The emitted row's estimator column says
`SINGLE DRAW … re-reduce as a median over N archived runs before committing`, in the row
itself rather than in a note beside it. So skx's share baseline cannot be imported from
`33de3b2`: that reading is a draw under the free instrument the pinning transition
replaces, and enshrining it as a reference would be precisely the defect rule 16 names.
Its rows are born from the pinned arm of the pinning-transition campaign.

### What is not per-host

The bar is `registered baseline − BASELINE_MARGIN`, and the margin is `CEIL_FRACTION`'s
own: `60.4 − 2.6 = 57.8` is the derivation the gate already prints, so a per-host bar is
the fleet bar's *construction* applied to a different baseline, not a second threshold with
a second justification. The three CPU models that derived the fleet bar stay on it, since
for them the artifact does not predate the admission — they are the artifact. Per-host
convergence for them is a post-tag option, and was not smuggled in here.

### Eras, and why an instrument change is a fleet-wide re-registration

*Amended 2026-08-21 on Scott's ruling, and the grounds are not the ones the literal rule
would have given.* Read literally, "one `BASELINE` per host, ever" said skx must be judged
against a baseline imported from `5ec5fea` and `33de3b2`. It fails on **misattribution**:
those are free-placement readings, §5 rule 5 pinned placement fleet-wide the same day, and
`docs/hosts.md` now says in plain text that every number predating 2026-08-21 was measured
free. Judging pinned readings against baselines minted from unpinned draws would book **the
methodology delta as host drift** — the cross-denominator sin the registry exists to
prevent, arriving through the registry.

So a baseline belongs to the **era** of the instrument that measured it, and the pinning
adoption is an era boundary: a constitution-dated instrument change, which is exactly the
"dated re-registration citing a named change" door the design already had, opened fleet-wide
instead of one host at a time. On the transition run every host renders `BASELINE` for
placement-sensitive quantities; entries land from the pinned arm's medians with enough
repeats per host that rule 16's estimator has an honest N; `CEIL_FRACTION` re-derives by the
same formula over new-era inputs (the CEIL8CI precedent); the README regenerates.

**The loophole guard**, without which "new era" becomes exemption-renewal-by-tweak: an era
exists only via a **dated §5/§7 amendment plus a both-arms transition archive**, recorded per
era in `scripts/measurement-eras.tsv`, and each host gets exactly one `BASELINE` per era.
Operator convenience cannot mint one. The guard is a reader and not a paragraph: a ledger
whose current era cites no amendment resolves to *nothing*, gate-p5 renders `FAIL`, and
resolution does **not** skip the bad row to reach a valid one — a fallback would silently
perform the misattribution the clause forbids. An era is *provisional* until its transition
archive lands, which is a disclosure and not a permission: what bounds `BASELINE` is the
witness index, so a provisional era grants nothing extra.

`free-placement`, the era before eras, is named retroactively and deliberately **undated**:
it was not a concept while it ran, and inventing an opening date would be a provenance claim
no artifact supports.

Note what this ruling did not turn on. Declining to import the free-placement reading was
right procedure, and the ruling lands on grounds that would hold if it had favoured no one —
which is why it is recorded here as misattribution rather than as a convenience.

### What this mechanism does not cover, stated inside the number

**The per-operator-machine limitation is discharged**, and the repair is named so a reader
who arrives at the old wording knows which claim died: the witness was a glob over
gitignored `build/` output, it is now the tracked `scripts/judged-runs.tsv`, and #114 is
closed by that change. What replaced it is a *different* limitation and not a lesser
statement of the same one — the tracked witness is landed by review, so a session that
proposes a witness row and lands nothing re-renders `BASELINE` next run. That has no
available action beyond the debt line gate-p5 already prints, so per §5 rule 12 as amended
it goes inside the number rather than into a new debt.

One branch is written and unreached: an unreadable CPU model resolves the share criterion to
`UNMEASURED` rather than falling back to the fleet bar, since falling back would be *looser*
than the truth for any host whose registered baseline sits above the fleet's. A host that
cannot answer a probe also produced no benchmark rows, so the branch is fail-closed and
unexercised, and that is stated rather than implied.

### The extension to the ratio criteria (#119, ruled 2026-08-29, landed 2026-08-30)

The class was built for the *absolute* criteria and the ratio criteria never got it: a scaling
bar applied to every host that reported, so a host outside the bar's derivation set was judged
against an artifact that predates its admission — rule 17's entire subject, unenforced one
criterion over. Ruled to be legislated **from principle**, with the lab `Strsm` table as
*exhibit A* rather than as the derivation set: a rule fitted to the rows that motivated it has
no independent content, which is the objection §5 rule 10 raises about counting corroboration
as derivation. It is the arm64 prerequisite, because every `c7g`/`c8g` host is by construction
outside every current bar's derivation set.

**The two derivation sets differ, and that is the finding, not an implementation detail.**
`STRSM_FLOOR` (6.067x as ruled here, 6.066x since the re-typing below) was derived from **three**
take-four rows — `keel-zen5` 6.4699x,
`keel-zen4` 6.6357x, `keel-skx` 6.8311x — while `CEIL_FRACTION = 44.2%` was derived from
**two** Zen models with `keel-skx` at `DERIV=0`. So the ratio criterion needs `SCALE_DERIVED_FROM`
of its own; reusing `CEIL_DERIVED_FROM` would have handed `keel-skx` a `BASELINE` for a bar it
helped set, which is rule 17 run *backwards* — an exemption granted by a chronology the host
does not have. Read off the artifact rather than off the constant: recomputing all three rows
from `archive/pinned8/` take four under the current instrument gives `lo` = 6.831 / 6.635 /
6.469x, reproducing the derivation the gate prints.

**The margin is `STRSM_MARGIN`, and refusing to convert 2.6 is the design point.** Clause (c)
requires *"the same margin constant subtracted from a different baseline, never a second
threshold with a second justification"*, so the per-host ratio bar is `baseline − 0.403x`.
`BASELINE_MARGIN`'s 2.6 is **points of share** and this bar is in **x**: carried across, a
registered host's bar would sit 2.6x under its own baseline — looser than the *retired* 6.0x
general floor, i.e. the surface-form error in numerical clothing. `0.403x` is already in the
right units, already derived from readings (7.403x, the lowest of the nine that ratified 7.0x,
less that 7.0x), and predates this era's rows by six days.

That reuse is **checked against the era's archives, not assumed**. Recomputed under the current
instrument, take four's three ratio intervals are **0.040 / 0.221 / 0.230x** wide, so the margin
is **1.75×** the widest — the same shape the gate already prints for the share bar. What the
check cannot see is stated inside it (§5 rule 12): **9 of the era's 15 archived `Strsm` rows are
wider than 0.403x**, and rule 19 refuses those above bar selection, so this class will register
baselines from a *minority* of its own rows. The 15 are also not one instrument — every printed
width in `build/` except `969c360`'s predates #116's CI repair, and the widest of them, 0.929x,
is a pre-repair rendering rather than an in-era measurement. Pooling it with the recomputed
three would be the cross-instrument error the era column exists to prevent, arriving through a
census.

**A byproduct worth recording rather than acting on.** The same recomputation puts `keel-zen5`'s
`lo` at 6.469x where the published derivation says 6.4699x, because #143 moved `bench_ratio_lo`'s
final rounding from nearest to down. Re-running the formula today would therefore type **6.066**,
not 6.067 — so the shipped bar is 0.001x *stricter* than its own formula now yields. Strictness
is the safe direction, and re-typing a bar downward after the fact is loosening a threshold, so
the constant stands and the delta is disclosed here (§5 rule 15). It also positive-controls #143:
the two 0.001 discrepancies are exactly the ones that fix predicts, at the two rows whose bounds
were not already on the lattice.

> **VOID as to its conclusion, 2026-08-31.** *"the constant stands"* is overruled; the
> observation and the #143 positive control above stand unchanged. Ruling below.

### `STRSM_FLOOR` is re-typed to 6.066 (#119, ruled by Scott 2026-08-31)

**The constant becomes the formula's output at full precision: `6.066`.** Scott's grounds:
*"the bar's authority is its derivation, and 6.067 is what the derivation printed through the
display-rounding path the fix just cured. Keeping the old constant 'because it's published'
would preserve the disease's output after curing its mechanism — a criterion constant that
disagrees with its own formula on its own inputs is a fossil the constitution has a name for."*
So the paragraph above got the *facts* right and the *disposition* wrong: it treated the bar as
a published number to be defended and the formula as a description of it, where the formula is
the authority and the number is its output.

**Direction, stated and not left to be discovered: 0.001x MORE LENIENT.** This is the whole
reason the ruling carries conditions. `6.469 - 0.403 = 6.066` exactly on the 3-decimal lattice
`bench_ratio_lo` returns; the shipped 6.067 is `6.4699 - 0.403 = 6.0669` rounded to *nearest*.
**Nothing was re-measured** — no sample, no host, no sweep moved. One quantum of instrument
precision did, and §5 rule 15 is answered by disclosing the direction plus the check below,
rather than by refusing the correction.

**Re-derived under the current instrument, not accepted on the ruling's word.** The three
take-four rows (`archive/pinned8/`, rev `6ba6566` `20260823T004407Z`, one raw log per host) put
through `tools/benchci` and `bench_ratio_lo`: `lo` = 6.831136 / 6.635663 / 6.469880 for
`keel-skx` / `keel-zen4` / `keel-zen5`, reproducing the four-decimal figures the derivation
publishes (6.8311 / 6.6357 / 6.4699). Argmin `keel-zen5`, bound 6.469 as the instrument renders
it, and the gate's own subtraction (`printf "%.3f", b-m`) types **6.066**.

**Three witnesses that it flips nothing, which is what makes "flips nothing" a finding rather
than a hope** (Scott's second condition). A verdict can move only for a bound in the half-open
band `[6.066, 6.067)` — FAIL under the old constant, PASS under the new — and:

1. **33 rows recomputed** from every raw log in `archive/pinned8/` and `archive/free-placement/`
   carrying both `Strsm` arms, under the current instrument: **0 in band**. 47 archived logs
   have no bounded `Strsm` ratio and are counted as skipped rather than as clean.
2. **184 rendered bounds** over the 62 logs in the tree that print one, extracted from every
   `Nx net of CI` rendering on a line naming `Strsm`: **0 in band**, nearest neighbours **6.017**
   below and **6.094** above, so the band is bracketed on *both* sides and not merely from above.
3. **The published pass count, regenerated.** `readme-numbers.sh` re-run over the same two
   archives with the new constant still prints *"9 of the 12 routine-host pairs … clear the
   bars"*, its only diff being the constant it quotes. The figure a reader would cite is
   byte-identical.

Witnesses 1 and 2 were **positive-controlled** — the predicate and the extractor were each run
against a planted in-band 6.066 and both fired — because a zero from an instrument never shown
to be able to return nonzero is unreadable. Witness 2's first form keyed on the comma rendering
alone and found 149 bounds; broadened to every rendering of `$lo` in the ratio criterion
(`:1085`'s parenthesised form, `:1072`'s `at 8 threads,` form, `:1150`'s `>= bar` form) it finds
184. The 149 was a no-finding from an under-inclusive pattern, and it is recorded here because
it agreed with the right answer for the wrong reason.

**One figure corrected in passing, same cause.** `gate-p5.sh`'s derivation line and DESIGN.md
§4 both published the three rule-19 interval widths as **0.229/0.220/0.040x**. Recomputed, the
full-precision widths are 0.229233 / 0.220796 / 0.039598 and the widths *the gate's own rule-19
sentence renders* (from the rounded `pt` and `lo` it also prints) are **0.230 / 0.221 / 0.040**.
So `0.220` matched no path at all — neither rounding nor the instrument — and `0.229` was the
full-precision quantity where the gate prints the lattice one. Both sites now carry the widths
the instrument renders, which is also what this document already carried at rule 19. All three
rows remain admissible against the 0.403x cap by a wide margin, and the two corrections move in
the *stricter* direction (a wider disclosed interval).

**One latent defect fixed in passing.** The share criterion tested `-z "$CEIL_FRACTION"` after
resolving a per-host bar into `$BBAR`, so a suspended *fleet* bar excused a host whose own
registered bar was sitting one branch up — a per-host bar overridden by the absence of a bar
that does not govern it. It now tests the resolved bar. Unreachable while 44.2 is typed, and
stricter when reached, so it is driven by a fixture rather than asserted.

## Rule 18 — an instrument measures a noun

*Ruled 2026-08-22. Issue #115, Scott's ruling on the hoist, opening "Yes, the shipped arm
hoists" and ending "The questions are answered; what remains is execution."*

The 8-thread compute ceiling is the denominator under every judged scaling share. `computeArm`
forked eight goroutines and joined them inside every one of `b.N` iterations, so the clock ran
across the convening as well as the arithmetic. Every figure it produced was correct arithmetic
over the wrong region. Scott's formulation is the rule: **"The ceiling's *definition* is what
eight threads can compute; an instrument whose measured quantity is 78% compute and 22%
choreography is measuring the wrong noun."**

Rule 11 already says the instrument overrules its author about the *number*. This is the
adjacent audit, and the number cannot perform it: a stable, reproducible, cross-host-consistent
reading is exactly what a wrong-noun instrument produces. What adjudicates is the timed region
itself, read as text, against the quantity the name claims.

### The severity is the deployment, not the code

`internal/par/par.go:109` forks per call too, so the judged routines share the shape. What
differs is op duration, by three orders of magnitude — `204126044 ns/op` for
`Scale/Sgemm/n=4096/threads=8` against `267710 ns/op` for the ceiling, from the same run's own
rows, a factor of 762. Converted to a rate so the hosts compare (§5 rule 13): **60.4 µs per
fork/join on keel-zen4 spread, 25.5 µs confined, 51.4 / 13.2 on keel-zen5, 33.4 on keel-skx
either way.** So ~60 µs is 22% of a ceiling op and 0.03% of a routine op. Both sides of a share
pay the term; only the denominator pays it at a rate that matters. That is §5 rule 14 arriving
as a corollary rather than a citation: the same code is honest in one deployment and dishonest
in the other, so the audit is per deployment and never inherited.

Its output was the impossible reading `8e6c6ac` now refuses — `ceiling: compute 461.4 GFLOP/s
+/- 62.68%` beside three 8-thread rates of 675.1, 704.3 and 662.3, i.e. **146.3%, 152.6% and
143.5% of their own ceiling**, printed as `89.8% / 93.7% / 88.1%` passes.

### The repair, and that it predates the mask

Workers are created before the timer, each warmed by a full op on the cpu the mask gave it,
parked on a `start` channel; the timed region is one `close(start)`, the steady-state loops, and
one `wg.Wait()`. Excess over ideal goes 1.290 → 1.010 on keel-zen4 spread and 1.599 → 1.014 on
keel-zen5 spread, with worker duty cycle rising from 0.67–0.88 to 0.97–0.99 — the removed time
was time the workers spent **parked**.

Scott's second clause is the one that matters for attribution: **"note this is a fidelity repair
to a defect that predates the mask entirely: at 25.5 µs per fork/join the confined-era ceilings
carried the same contamination at smaller amplitude, so the repair isn't cleaning up the spread
mask's mess, it's fixing an instrument the mask merely exposed."** The mask is an amplifier of a
harness term. Booking this repair against the mask would have made a placement decision out of
a harness defect, which is what T-51 nearly did.

### Version, do not correct

The archived shares divided by v1 ceilings, whose bias was host-and-mask-dependent and **varied
per run** — that variance *was* the scatter that drew attention. No correction formula recovers
them honestly, and one applied anyway would publish a precision the data never had. Scott:
**"Era-scoped, no re-adjudication."**

Nothing needed recovering, and that is the pre-commitment discipline collecting its payoff. No
bar was typed from those readings, no registry entry landed from them, no verdict rests on them.
The counterfactual is recorded because it is the argument for the discipline rather than for the
repair: **had 74.3% been typed as `CEIL_FRACTION`, this repair would now be forcing a bar
retraction with published consequences.** A defect in a denominator became a version label.

`instrument=v2` is emitted in the ceiling's own declaration line, not merely stated here — a
reader comparing two archives needs the noun's version where they read the number.

### The precondition the hoist trades for, stated inside the number

v1 was `b.N`-independent by construction. v2 amortizes one convening over `b.N`, so a short run
under-reads: locally **74.15 GFLOP/s at `1x`, 127.5 at `5000x`, converged by `500x`**. The gates
run `-benchtime=1s` and sit far inside the flat region (~0.005%), but that is a fact about the
gate configuration and not about the arm, so it is written in the arm's comment where a future
`-benchtime=3x` smoke run will be read against it.

Two limitations, neither with an available action (§5 rule 12 as amended). *Which* fork/join term
is topology-sensitive is unattributed — cold wake-up of parked Ms and cross-cache-domain stealing
are both live and nothing here separates them. And the `ops=` field this repair first tried to
emit is **not** in the declaration: `declareCeiling` keeps one line per row and `testing` calls
the benchmark once per ramping `b.N` trial, so the `b.N=1` call wins and any `b.N`-dependent
field freezes at 1. It printed `ops=1` beside a row that measured 3853 iterations. The
pre-existing fields are `b.N`-invariant, which is why the dedup was safe before and is the reason
the trap is recorded rather than worked around.

## Rule 19 — an interval too wide to adjudicate its criterion is not a vote

*Ruled 2026-08-22. Issue #6, Scott's ruling on the Strsm widening, opening "Ruled, and it's the
third option — your assumption is closer than the exception, but the existing law does better
than 'unjudgeable'" and closing "CEIL_FRACTION types from the GEMM-class rows regardless of how
Strsm scatters, so the founding no longer has a single point of failure."*

Three verdict classes were available for a row whose interval is wider than the bar it is
compared against, and the ruling took none of the two obvious ones. The question arose because
the spread mask, adopted the same day as the standing judged placement, widened `Strsm`'s
intervals 3–15× — ±2.32/2.97% to ±6.52/9.48/11.84% on zen4, ±0.75/0.87% to ±5.13/10.28/13.24%
on zen5 — while keel-skx's identical-mask control did not move (±0.50–1.30% to ±0.82–1.00%).

### What was refused, and why the quiet reading is the dishonest one

The first candidate was a **per-row placement exception**: judge `Strsm` under the confined mask,
which reads tighter, and everything else under spread. Refused, and the grounds are what the mask
*does* to the phenomenon. T-45 and T-52 established that the bimodality exists under **both**
placements. Confined `Strsm` is therefore not measuring a different, truer quantity — it is
sampling the same two-mode behaviour at weights that happen to concentrate. Choosing it would be
choosing the placement that **hides a real behaviour of the routine** because the hidden version
is quieter, which is flattery-shopping's respectable cousin, and "one standing placement" exists
to bar the whole family. In the ruling's words: the 334–455 spread is information — something in
the solve's execution is placement-sensitive and bistable, and the row's honest state says so.

The second candidate was **unjudgeable**, a new verdict meaning "this criterion cannot be applied
to this routine". Refused as over-broad, because the law already covered this shape and had
covered it for months: #105's clause, ruled for the clock series, is that *an interval too wide
to adjudicate its criterion is not a vote, it is an UNMEASURED with its width named*. The ruling
extends that to the scaling and share criteria as a standing clause rather than minting a
routine-specific escape hatch. Rows over cap render `REPORTED`, naming the width; rows within cap
judge normally. Which means **keel-skx judges** — its control is tight and there is no reason to
discard a measurable host because its fleet-mates scatter — while the two EPYCs **report** until
the bimodality is attributed.

### The cap is the criterion's own declared slack, and it must predate the readings

The cap could not be read out of the tree: `git grep admissib` found the law stated and no
numeric threshold anywhere, and #105 and #86 both refuse only the unbounded case. So it is
derived, and the constraint on the derivation is the one that matters — a width cap chosen after
seeing the widths is the threshold-invented-after-the-fact §5 exists against, and this one had to
come from a standard that predates the run (the amend-a-criterion rule).

Each criterion's own **declared slack** satisfies that, and each has exactly one:

| criterion | bar | slack | dated | width measured as |
|---|---|---|---|---|
| share of the 8-thread ceiling | `CEIL_FRACTION` | `BASELINE_MARGIN` = 2.6 points | 2026-08-20 | `100*(share − share net of CI)`, points of share |
| `Strsm` scaling | `STRSM_FLOOR` | `STRSM_MARGIN` = 0.403x | 2026-08-16 | `point − point net of CI`, in x |

Both are the same construction — `bar = reference − slack` — which is why importing them is
reuse and not invention: 53.6 − 2.6 = 51.0 is the derivation the share bar printed while it
stood, and 7.403x (janus, the lowest of the nine) − 0.403x = 7.0x is what `Strsm`'s bar was
ratified as. Two caveats travel with the second: 0.403x is partly a **residual**, since 7.0 also
had to clear the then-live 6.0x general floor, and it settles only the *units* half of the open
construction question at `scripts/gate-p5.sh`'s `STRSM_FLOOR` — what margin the typing commit
subtracts to set the bar is still that commit's to argue.

The units are not interchangeable, and the reason is #110's error one criterion over: the share
bar is in **points**, and a relative rate CI converts to more or fewer points depending on how
high the share sits, so a ±% compared against a points margin is two units in one inequality.
As a fraction of its own bar the two margins land 0.7 points apart — 2.6/51.0 = 5.10% against
0.403/7.0 = 5.76%. That is corroboration and not a derivation, and the independence is weak
enough to say so out loud (§5 rule 10): neither number was chosen with the other in view, but
both were chosen by the same two people six days apart.

### The class is per row, which is what made it new

Every non-pass class before this one was per host: admission is a property of the silicon,
`BASELINE` a property of a (host, criterion) pair. `REPORTED` is the first that is **per row**,
and that produces a defect with no precedent to copy. `HOST_CLEARED` and `HOST_MEASURED` start at
1 and are only ever lowered, so a host every row of which rendered `REPORTED` would leave the row
loop looking like a clean sweep and be **counted as one** — §5 rule 6's vacuous pass, arriving
through a class designed to be green-compatible. So:

- a host with **some** rows over cap keeps the votes its other rows earned, and the fleet
  aggregate names the row count in every branch including `pass`, because a pass over hosts three
  of whose twelve rows were out-resolved is a pass over nine rows;
- a host with **every** row over cap gets its own bucket, subtracted from the no-coverage
  residual whose sentence — "produced no complete set of ratios" — would be false about a
  reading that printed;
- the all-rows test compares against the **row count** rather than tracking a "judged something"
  flag, because that flag would have to be set on all eight paths that reach a verdict and a
  missed one is invisible on a healthy run.

### What a bar may be typed from

`CEIL_FRACTION` and `STRSM_FLOOR` type from **admissible rows only, or stay empty**. Either state
is honest and neither blocks the era. This is the clause's real work in the founding run, and the
reason it had to land before the campaign rather than after: with both bars empty there is no
comparison for it to refuse, so what it decides today is **eligibility**, and a bar typed from an
interval wider than its own slack is noise promoted to law. Landing it first is also the only
order in which it cannot have been tuned to the widths it sorts.

An earlier draft of this reasoning said the clause "changes no verdict in the founding run
because the bars are empty." That is wrong and is recorded as wrong: the width test sits *ahead*
of the empty-bar branch in both criteria, so a wide row renders `REPORTED` where it would
otherwise have rendered a bar-less `PASS`. The pre-registration argument is the one above — the
clause predates the numbers — and not the false claim that it is inert.

### Coverage, and what stays unexercised (§5 rule 12)

`scripts/gate-p5.sh` has no standing harness, so this is a session act and is stated as one. 19
renderings were driven against the file's own bytes: both predicates at their boundaries (2.59 /
2.60 / 2.61 points; 7.403x−7.403x, 7.403x−7.000x, 7.404x−7.000x), the reconstructed EPYC and skx
widths, all four arms of the all-rows test, and the aggregate's arithmetic and sentences with a
noisy host and with one noisy row per host. Predicted before it ran and confirmed: skx's ±0.9%
judges at 0.06x of width, zen5's ±13.2% reports at 0.930x, 2.3× its cap.

Three things are **not** exercised. The clause has never executed inside `gate-p5.sh` on a real
fleet — the predicates and message bytes are driven, the control flow reaching them is not. The
widths above are reconstructions from the disclosure percentages, not readings lifted from
archives. And the uncomputable-width arm is a can't-happen written as a refusal rather than
driven: the raw share reads the same two rows the bounded share just read successfully, so it
cannot be reached, which is exactly why it refuses instead of falling through to a judgement that
would green forever and tell no reader which it was.

## Rule 20 — a zero-width rank interval can mean the window stopped looking

*Ruled 2026-08-22. Issue #6, Scott's ruling on the #116 stop, authorizing the repair — "your stop
was the right protocol for the wrong-shaped reason to hold it" — and then minting a fifth law off
`build/ceil-recur.log`: "a zero-width rank CI can mean the window stopped looking … that's one
printf and it closes the concealment class — the third instrument this month whose confidence
statement was measuring its own construction rather than the host."*

### The fabrication that led here, and why it is rule 18 rather than a new rule

`tools/benchci`'s `ciFraction` reproduces benchstat's **display** half-width,
`d := max(Hi−Center, Center−Lo)`, which is correct for printing a `±x%` column and is the reason
`-verify` agrees with benchstat on all 42 cells. The defect was in the *consumer*:
`scripts/bench.sh`'s `bench_ratio_lo` reconstructed a denominator bound as `center × (1 + ci)`,
applying a half-width **selected from the wider side** symmetrically to both. benchstat's interval
is the distribution-free median CI `[x_(r), x_(n+1−r)]`, so **both bounds are order statistics —
literally sample values — and neither can lie outside the data.** A one-sided *low* tail was
therefore mirrored *upward* onto the denominator.

Scott's governing statement: **a CI bound must be a bound of something measured, and a symmetrized
one-sided interval is a number with no referent.** That is rule 18's principle — an instrument
measures a noun, and the noun is auditable separately from the number — reaching one level in,
from the timed region to the interval. Rule 18's *text* governs the timed region and is cited here
as principle-with-extension, never as though it already covered intervals.

Three independent refutations of the fabricated denominator, all printed in the fix commit:

1. **The arithmetic reproduces to 14 significant figures.** From keel-zen5's take-three ceiling,
   `Center=2291.5, Lo=1995, Hi=2292`: `max(0.5, 296.5) = 296.5`, `296.5/2291.5 =
   12.93912284529784%`, and `2291.5 × 1.1293912284529784 = 2588.0` — the exact denominator the
   gate used. The reported width was the **downward** reach of 296.5, applied upward.
2. **2588 exceeds anything ever observed.** The n=10 sample maxes at 2295; a fresh n=30 sample
   maxes at **2296**. The fabricated bound sits 12.7% above the largest of forty draws. A bound
   outside the sample range is the tell, and it is available without any distributional argument.
3. **Take four re-measured it.** The corrected share landed within 0.3 points of the prediction
   made from the repaired arithmetic before the run.

Direction, per rule 15: the old form could only ever **inflate a denominator** and therefore
**deflate a share** — `hi_den ≥ center_den` always — so it could not manufacture a pass, and that
is exactly why it survived so long. The repair moves shares *up* (zen5's ceiling share 42.7% →
48.25%). The extra care owed to a favorable direction is **procedural, not substantive**: it
buys the three refutations and the pinning fixtures, not a different verdict about whether the
number was fake.

### The rank window, and why 1/sqrt(n) is the wrong model of escalation

Verified against `benchmath`'s own `stats.QuantileCI(n, 0.5, 0.95)`: the rank pair is `[2, 9]` of
10, `[10, 21]` of 30, `[36, 55]` of 90. Those move **inward as a fraction of n** — 0.20/0.90 →
0.33/0.70 → 0.40/0.61 — so a contaminant present at a fixed *rate* is stepped over by rank at a
speed no variance argument predicts. At a fixed 20% contamination the reported width goes
**±21% → ±0% → ±0%** across n = 10/30/90.

### The recurrence: escalation concealed a live defect

`build/ceil-recur.log`, n=30 on keel-zen5, sorted: `1989 2290 2290 2290 2290 2291 ×16 2292 2293
2294 2294 2294 2295 2295 2295 2296`. The low mode **recurred** — it is in the sample — and
`[x_(10), x_(21)] = [2291, 2291]`, an **exact** zero width, over a range spanning 13.40% of the
center. Raising `-count` did not fix the host; it moved the window past the evidence.

This **partially inverts** the assumption under the same-precision-votes / higher-precision-
supersedes clause (`DESIGN.md` §4 Phase 3, ratified 2026-08-19). That clause survives intact for
variance-tightening, and the reason is that its claim is true *of the central mass*: a higher count
really is a strictly tighter estimator of the median. What it does not say, and now must, is that
the same move widens the blind spot outside the window — "strictly more informative" is true of the
center and false of the tails.

**One printed `0%`, two mechanisms.** The clause's existing limit already treats `0%` as inert,
but on the grounds of benchstat's *reporting resolution* — T21's `± 0%` meaning "narrower than
0.5%", a rounding band that can straddle a bar (#110). This is a different animal: an exact zero,
both bounds the same sample value. The two are indistinguishable in the column that prints them,
which is what makes the range disclosure load-bearing rather than decorative.

### The disclosure, and why it carries no threshold

`bench_describe` now prints the observed min/max beside every reading, and names the anomaly:

    2291 GFLOP/s +/- 0.0% [1989, 2296] RANK-WINDOW-BLIND(span 13.40% under a 0.0% interval)

**As of 2026-08-31 the same reading renders without a verdict** (#132, ruled below); the marker is
gone and the quantity it was deciding about is printed instead, on every row:

    2291 GFLOP/s +/- 0.0% [1989, 2296] D=inf (span 13.40% / interval 0.0000%)

Three properties are deliberate. It is **unthresholded** — a printed-width zero under a span
exceeding one quantum of that same display fires it (see the amendment below; as adopted it was
narrower, keying on an *exact* zero) — because a triviality cutoff would be the
tuned-after-the-fact constant §5 is built against, and a reader dismissing a 0.01% span can do so
from the printed span. The
**range is not an interval**: it is what was observed, it survives on a row whose CI is unbounded,
and no criterion divides by it, which is what lets it be read off archived CSVs without moving a
historical verdict. And it **moves no verdict** — stated as a limit, not a feature: a blind
reading still adjudicates, and the anomaly only makes it visible to whoever must decide to re-run.

The empty-field guard is the non-obvious part. Pre-#116 archives have three columns, and `%.4g` of
an empty awk field renders a confident `[0, 0]`; the guard exits before the range on any row that
does not carry one.

### Amendment, 2026-08-28: the printed line is the assertion

The clause above reserved "the tuned middle" to a ruling. The ruling dissolved the middle instead
of splitting it: **the marker keys on the width as printed**, so the trigger is a printed-width
zero under a span exceeding one quantum of that same display.

The reasoning that got there started from the wrong end. My objection to widening the trigger was
that testing the printed value "installs a cutoff chosen by a format string" — the right suspicion,
aimed at the right object, resolved by noticing what the format string actually does. It is not
*choosing* the sensitivity; it is **defining the assertion**. A reader's trust attaches to the line
in front of them, so the printed line is what the marker must defend, and `± 0.003%` renders
`0.0%` and makes the identical claim to that reader as an exact zero. The four exact zeros were
never a privileged class — they were an artifact of testing the stored value rather than the
published one. Symmetrically, the anomaly is restated as *the range refuting the printed claim*: a
span the interval could not have concealed if it meant what it displays. Both halves now come off
the same line, and the quantum is the display's own resolution, which is why nothing is left to
tune. Same move as #110's band-top, where the resolution of the display defined what a rounded
value could support.

Reach, measured rather than claimed — 114 stored CI readings across the 24 archived-era CSVs:

    prints as 0.0%   16
    exactly 0.00%     4     <- what the trigger saw before
    the reserved middle  12

Direction (rule 15): exact-zero is a strict subset of display-zero, so this can only add markers.
No row that was named goes silent, and it remains output — no verdict moves.

Coverage, with what it cannot see (rule 12): `bench_describe` **had no test before this
amendment**, which is why the band went unseen — the marker was driven by hand once and nothing
re-drove it. `scripts/remote-exec-test.sh` §9f now fixtures six cases, three of them real readings,
the discriminating pair being the `0.00%`/`0.07%` keel-skx printed across this era's two archives.
Unexercised: the archived path, because **no persisted CSV anywhere in the tree carries the seven
columns the range needs** — every one predates #116 — so the archive cannot witness this renderer
and the fixture is the whole of its coverage. **Lifted 2026-08-30; see "The archived path, and what
the corpus said about the trigger" below.**

### The tally printer (same ruling)

*"A certificate whose headline tally is the operator's grep, however honestly labeled, is a
certificate with a hand-counted spine. The run that signs v0.1.0 prints its own arithmetic."*
The previous session's 62/3/4/4/0 was my grep of the gate log, disclosed as such on #6. Now the
five verdict primitives in `scripts/remote.sh` each increment a counter and `gate_verdict` calls
`gate_tally` from **one** site, so all six gates gained it at once. Zero counts print rather than
being omitted, and a **zero total is an anomaly that sets `FAIL`** — a gate that reached a verdict
without rendering one adjudicated nothing, which is §5 rule 6's vacuous pass arriving through the
counter. The constraint worth recording: the helpers may never be called in a subshell, or the
increment is lost with the subshell's environment while the line still prints.

### Coverage, and what stays unexercised (§5 rule 12)

Driven: three Go fixtures pinning the one-sided form (the symmetric `ci` at
`12.93912284529784/100` *and* the honest `Hi` at 2292, held apart on purpose because the gap
between them **is** the defect; `sampleRange`; the seven-column CSV with ci/lo/hi going unbounded
**together** on one flag). `-verify` agrees with pinned benchstat on all 42 cells, so the display
column is provably untouched. The awk consumers were run end-to-end on the real archived CSV
(share 0.483, matching the 48.25% predicted pre-run). `RANK-WINDOW-BLIND` was driven **on purpose**
against the recurrence sample rather than inferred from a healthy run (and since #132 deleted it,
what those arms drive is `D` on three measured intervals, with its absence asserted by name). `clock_series` was exercised
across five shapes — six-field flat, legacy two-field flat, genuine decline, missing, unbounded —
after an audit found its `n != 6` guard would have silently reported "unbounded" on every host on
every run the moment `bench_stat` grew fields; that hazard was found by enumerating the 29 call
sites, not by a test. The tally was cross-checked against an ANSI-stripped grep of the same
gate-p0 log (11 PASS / 1 FAIL / 4 UNMEASURED / 0 BASELINE / 0 REPORTED, agreeing exactly), and
because that run left two classes at zero, `BASELINE`, `REPORTED` and the zero-total anomaly were
driven separately.

Not exercised. **No part of this has run on a real fleet** — the signing hosts were torn down
before it was written, so the range disclosure and the tally have never executed inside
`gate-p5.sh` against live remote readings, and gate-p0's own FAIL/UNMEASURED here are
fleet-absence rather than criterion failures. The n=90 rank pair is arithmetic from
`QuantileCI`, not a measured 90-sample run.

### The trigger's sensitivity is measured, and it is 1 of 3

The anomaly's blind spot is not a hypothetical, and running the repaired tool over the whole
recurrence log rather than the one row that motivated it is what showed that. **All three** of
zen5's 8-thread arms are contaminated, and only the `avx512` one renders an exact zero:

| arm | reported CI | interval width | observed range | span | named? |
|---|---|---|---|---|---|
| `avx512/8T` | ±0.0000% | 0 | [1989, 2296] | 13.40% | **YES** |
| `avx2/8T` | ±0.0873% | 1 | [1063, 1148] | 7.42% | no |
| `scalar/8T` | ±0.0861% | 0.1 | [100.9, 116.4] | 13.34% | no |

Two consequences, and the first is not about this rule at all. **The contaminant reaches the
scalar arm**, so it is not an AVX-512 frequency artifact; whatever intermittently costs zen5 13%
of its 8-thread throughput is indifferent to the instruction set, which points at scheduling and
away from the vector units. That is a lead for the attribution `Strsm`'s bimodality still needs,
recorded here because this log is where it was visible.

Second: the exact-zero trigger names **1 of the 3** blind rows. The obvious threshold-free
widening — fire when the window excluded observed data by more than its own width — was
implemented and **refuted by measurement in the same log**: it fires on all six rows, including
the three healthy 1-thread arms whose spans are ~1% and whose CIs are ordinary ±0.17–0.27%. The
reason is structural, which is why no tuning rescues it: at n=10 the pair `[2, 9]` excludes exactly
one sample at each end **by construction**, so "excluded more than it resolved" is almost always
true and carries no information. So the two available threshold-free triggers are 1/3 sensitive
with 0/3 false positives, or 3/3 sensitive with 3/3 false positives, and everything between them
needs a constant chosen by looking at these widths — which is Scott's call under the
amend-only-with-a-predating-standard rule and is not taken here. The conservative one ships, its
sensitivity stated in this table rather than in a claim, and the span prints on **every** row so
that the two rows the trigger misses are still legible to anyone who reads the reading.

### The archived path, and what the corpus said about the trigger

*Measured 2026-08-30, closing #133 and reporting to #132. No trigger changed.*

**The stated coverage limit was wrong about its own subject, and the correction is a large corpus
rather than a caveat.** The limit read "no persisted CSV carries the seven columns" — true, and
about persisted *CSVs*. Every `archive/*/*.txt` is a raw benchmark log, so the samples the seven
columns are computed from have been in the tree all along: `tools/benchci` re-derives them. The
one documented command is `for f in archive/*/*.txt; do go run ./tools/benchci "$f"; done`, and
`tools/benchci/archive_test.go` is that sweep as a **Go test**, so CI re-drives it — `go test ./...`
runs in CI and `remote-exec-test.sh` does not, which is the whole reason the fixtures alone were
memory-driven. (That placement costs the `scripts/` ledger nothing, and it costs `gate-docs.sh`'s apparatus
ledger nothing either — which is a hole in the reporter and not a property of the code. Line 327
excludes `*_test.go` from the `tools/` term while line 331 includes it for `bench/`, so **749 test
lines under `tools/` are counted in neither term**, 253 of them added here: honest apparatus 20044
and 2.79x against the reported 19295 and 2.69x. Same shape as the `bench/` hole that file corrected
on 2026-08-20, and left unfixed here on that entry's own instruction that a definition change be
argued rather than bundled into other work. Disclosed here and recorded on the paydown issue,
`#131`, which is where the ledger is being worked.)

What it asserts, all of it fail-driven before commit: the corpus re-derives to **80 files / 1420
readings / 0 unbounded** (as of 2026-08-30; floors, so a lost archive reds and a new era does not);
**no interval bound escapes its own samples** on any of the 1420, with a negative control feeding
the same predicate #116's own fabricated bound; the rank pairs `[2,9] / [6,15] / [10,21] / [18,32] /
[36,55]` pinned through `benchmath.AssumeNothing` itself over samples `1..n`, where each bound
comes back **equal to its own rank** — the cleanest available proof that both bounds are order
statistics; and the shipped `bench_describe` driven over a re-derived archived CSV, with the marker
**firing on purpose** there.

That last assertion is also #132's answer, and it is a *no*. The disparity ratio
`D = (max−min)/(hi−lo)` has achievable set `[1, ∞]` at every bounded n — `D ≥ 1` because both
bounds are order statistics, and `sup D = ∞` at a degenerate window — so **no finite cutoff follows
from the rank geometry**; only the two endpoints do, and both are already known (`D ≥ 1` is the
3/3-false-positive widening refuted above; `D = ∞` read at display resolution is what ships).
`D ≥ 1` held on 1420 of 1420, which is the positive control for the claim in the instrument that
motivated it (§5 rule 11).

The corpus did find a defect, and it is not the missing middle. The shipped trigger fires on **90 of
1420** readings (6.3%); the classes re-measure at **133 display-zero / 25 exact / 108 middle**, the
same 1:4 shape as the 16/4/12 above. But among the 18 per-host `pinned8` archives, of the **296**
readings whose printed width is within one quantum of zero under a span exceeding that quantum,
**78 are named and 218 are silent** — and sorted by span the two verdicts **interleave to the
bottom**, including on the `Scale` rows that feed the ratio criterion:

| span | printed width | named? | row |
|---|---|---|---|
| 15.93% | 0.0% | **YES** | `Ceiling/compute/avx2/threads=8` (zen5, `6ba6566`) |
| 15.74% | 0.1% | no | `Ceiling/compute/avx2/threads=8` (zen5, `6ba6566`) |
| 14.78% | 0.0% | **YES** | `Ceiling/compute/avx512/threads=8` |
| 13.76% | 0.1% | no | `Ceiling/compute/avx2/threads=8` |
| 6.51% | 0.0% | **YES** | `Scale/Ssymm/n=4096/threads=8` |
| 6.24% | 0.1% | no | `Scale/Ssymm/n=4096/threads=8` (`969c360`) |

**Corrected 2026-08-31, re-deriving every row for the ruling below.** Two labels above are wrong,
and neither changes the finding — the interleaving is real — but a table whose rows cannot be
found is not a record. Row 1 is **keel-zen4**, not zen5: the pair is *one arm on two hosts of one
sweep*, which is also how the prose below and `DESIGN.md` §5 rule 20 described it, and both said
"adjacent arms of one file". And the table sorted on span across **units**, which it does not say:
rows 1, 2, 5 and 6 are `sec/op` while rows 3 and 4 are `sec/op` and `GFLOP/s` respectively, so
rows 3–4 differ in unit as well as arm. Full provenance, as rendered by the shipped renderer:

| row | file | unit | reading |
|---|---|---|---|
| 1 | `6ba6566` keel-**zen4** `-3` | sec/op | `8.984e-05 s +/- 0.0% [8.977e-05, 0.0001041] D=204.5` |
| 2 | `6ba6566` keel-zen5 `-2` | sec/op | `7.32e-05 s +/- 0.1% [7.306e-05, 8.458e-05] D=180` |
| 3 | `6ba6566` keel-zen4 `-3` | sec/op | `0.0002157 s +/- 0.0% [0.0002154, 0.0002472] D=161` |
| 4 | `6ba6566` keel-zen4 `-3` | GFLOP/s | `933.7 GFLOP/s +/- 0.1% [806, 934.5] D=160.6` |
| 5 | `6ba6566` keel-zen5 `-2` | sec/op | `0.1281 s +/- 0.0% [0.1279, 0.1362] D=94.19` |
| 6 | `969c360` keel-zen4 `-3` | sec/op | `0.208 s +/- 0.1% [0.2075, 0.2205] D=54.81` |

The exhibit is one file's adjacent arms —
`archive/pinned8/bench-gate-p5-6ba6566-keel-zen5-20260823T004407Z-2.txt`, `scalar/threads=8` span
**27.99%** named at `0.0%`, `avx2/threads=8` span **15.74%** silent at `0.1%` — and the silent row's
n=30 sample is 28 draws inside 0.3% plus **two recurring** draws 15.5% slow at ranks 29 and 30,
which is precisely the recurrence mechanism this rule was written for. So what separates the two
rows is which side of the display quantum the *surviving* window's spread happened to land on, and
nothing whatever about the contamination: **the marker's sensitivity is an accident of how quiet the
clean mass was.** The trigger's equality at zero is the seam.

Not fixed here, deliberately. Widening it is a criterion amendment; it needs a standard that
predates the result, and the only predating standard available — the display quantum generalized to
`span > printed_width + quantum`, no new constant — measures **90.4%** firing on this corpus, which
is the 3/3 refutation above at 1420× the scale. A cutoff on `D` fires 9.1% at `D > 10` and 5.8% at
`D > 20`, but its Jaccard agreement with the shipped marker never exceeds **0.35** at any cutoff, so
it is a *different* set and not a refinement of this one. #132 carries the four options with their
measured rates and stays open for Scott. One thing the corpus settles for free: over a **genuine**
three-column `go tool benchstat -format=csv` of the same archive, that avx2 row prints `0.0%`
because benchstat rounds to `%.0f%%` — so the seven-column writer is what made the printed width
discriminating at all, and the historical logs were blind in both directions at once.

### Amendment, 2026-08-31: delete the decision, print the quantity (#132, ruled by Scott)

*The four options above were carried to Scott. He took none of them.*

> Print the disparity on every line, threshold nothing. The history is the argument: every
> thresholded form of this detector has had a blind band — exact-zero missed twelve, display-zero
> missed the 0.1% row that decided a verdict, and 6.3% would spend its life waiting for instance
> five to arrive at 6.2%. A detector that renders a verdict inherits a false-negative band from
> wherever its threshold sits; an instrument that reports the quantity has no band to be blind in.
> And the verdict was never load-bearing — width-admissibility judges the interval; the marker
> serves human readers, and readers are better served by the number than by an opinion about the
> number. "An instrument measures a noun" — the noun here is the disparity, and the decision was
> decoration. The four known instances get re-rendered in the docs as worked examples of reading
> D, so the judgment the threshold used to make badly is taught instead of automated.

`bench_describe` now closes every reading that carries a range with

    D=<value> (span <s>% / interval <i>%)

and `RANK-WINDOW-BLIND`, `QUANTUM` and the conjunction between them are gone. Three details are
not free choices. **`D` is computed from the real asymmetric bounds `hi − lo`, never from the
printed `± W%`**, which is the *mirrored symmetric* half-width `max(hi−c, c−Lo)/c` (#116,
`tools/benchci/main.go`) — a different quantity, so `D` is not recoverable from `W` by a reader or
by a later sweep, which is why **both operands print** beside it. `D = inf` at a zero-width window
and `D = 1` when a zero-width window sits over identical samples: `1` is the floor of the scale,
verified on 1420 of 1420 archived readings, because both bounds are order statistics and cannot
escape their own samples. And nothing else moved — no criterion reads `D`, so per rule 15 this is
output-only in the strict sense: it cannot turn a green red or a red green, and rule 19's
width-admissibility remains the only thing that judges an interval.

#### Reading D: the four instances, re-rendered

Every line below is printed by the shipped `bench_describe` over `tools/benchci`'s re-derivation of
the named archive file, and the "old verdict" column is what the deleted trigger did.

**1. The one that decided nine verdict lines.** keel-skx's 8-thread ceiling, the denominator of all
three of its share criteria, in `GFLOP/s`:

| run | reading | old verdict |
|---|---|---|
| confirmation `969c360` | `1444 GFLOP/s +/- 0.0% [1430, 1445] D=inf (span 1.04% / interval 0.0000%)` | **named** |
| take four `6ba6566` | `1444 GFLOP/s +/- 0.1% [1429, 1446] D=17 (span 1.18% / interval 0.0693%)` | silent |

*How to read it:* `D=inf` says the window collapsed to a point and 1.04% of sample disagreement is
outside it — the median is exact and the reading is not. `D=17` says the interval is real but 17×
too narrow to contain what was observed; the silent row was blinder in **span** (1.18% > 1.04%) and
said nothing at all. Neither reading is refused by anything; both are denominators you would want
to re-run before publishing a share off them, and the difference between `inf` and `17` is what
tells you the first is a degenerate window and the second merely an optimistic one.

**2. One arm, two hosts, one sweep.** `Ceiling/compute/avx2/threads=8` in `sec/op` at `6ba6566`:

| host | reading | old verdict |
|---|---|---|
| keel-zen4 `-3` | `8.984e-05 s +/- 0.0% [8.977e-05, 0.0001041] D=204.5 (span 15.93% / interval 0.0779%)` | **named** |
| keel-zen5 `-2` | `7.32e-05 s +/- 0.1% [7.306e-05, 8.458e-05] D=180 (span 15.74% / interval 0.0874%)` | silent |

*How to read it:* 204.5 against 180 is a 14% difference in blindness, and the verdicts differ
because one printed width rounded to `0.0%` and the other to `0.1%`. Read as `D`, these are the
same finding on two hosts — which is what a reader needs, because a contaminant on two hosts of one
sweep is a sweep-level problem and a contaminant on one is a host-level one.

**3. Two rows of one file.** keel-zen4 `-3` at `6ba6566`, and the pair the prose above called
adjacent arms:

| row | reading | old verdict |
|---|---|---|
| `avx512/threads=8`, sec/op | `0.0002157 s +/- 0.0% [0.0002154, 0.0002472] D=161 (span 14.78% / interval 0.0918%)` | **named** |
| `avx2/threads=8`, GFLOP/s | `933.7 GFLOP/s +/- 0.1% [806, 934.5] D=160.6 (span 13.76% / interval 0.0857%)` | silent |

*How to read it:* `D` = 160.995 and 160.625 unrounded. **This is the instance that convicts any
future bar on `D` as well**, and it is why Scott's "threshold nothing" is not merely a preference
for numbers: a cutoff placed anywhere between these two would be separating readings that differ by
0.2% in the quantity it claims to measure. There is no value of the constant that makes this pair
behave, which is the definition of a parameter chosen by looking at the data.

**4. The pair that reaches a ratio criterion.** `Scale/Ssymm/n=4096/threads=8` in `sec/op`, the arm
that feeds the scaling ratio:

| run | reading | old verdict |
|---|---|---|
| `6ba6566` keel-zen5 `-2` | `0.1281 s +/- 0.0% [0.1279, 0.1362] D=94.19 (span 6.51% / interval 0.0691%)` | **named** |
| `969c360` keel-zen4 `-3` | `0.208 s +/- 0.1% [0.2075, 0.2205] D=54.81 (span 6.24% / interval 0.1139%)` | silent |

*How to read it:* this is the pair Scott's "instance five to arrive at 6.2%" names — a span cutoff
at 6.3% separates them, and the fifth instance would land just under it. Here the named row *is*
the blinder one (94.19 > 54.81), so the trigger was right by accident; `D` says so without needing
to have been right, and 54.81 is still an interval 55× too narrow for its samples on a row a
criterion divides.

#### The one that convicts the trigger outright

Instances 2–4 show the trigger disagreeing with the severity. One archive file shows it
**inverting** it — `archive/pinned8/bench-gate-p5-6ba6566-keel-zen5-20260823T004407Z-2.txt`, one
host, one sweep, all three arms in `sec/op`:

    scalar/threads=8   9.025e-05 s +/- 0.0% [9.017e-05, 0.0001154] D=459.3 (span 27.99% / interval 0.0609%)   named
    avx2/threads=8     7.32e-05 s  +/- 0.1% [7.306e-05, 8.458e-05] D=180   (span 15.74% / interval 0.0874%)   SILENT
    avx512/threads=8   8.787e-05 s +/- 0.0% [8.77e-05, 9.483e-05]  D=142.6 (span 8.11% / interval 0.0569%)    named

It named 142.6 and skipped 180. Sorted by the thing it claimed to detect, its verdict is not
monotone — so this was never a *sensitivity* to be tuned; it was a discontinuity, and the deletion
is the only fix that does not leave a band. These three rows are pinned in
`tools/benchci/archive_test.go` for exactly that reason: the argument for the ruling is a
regression test, not a paragraph.

#### Coverage, and what it cannot see (rule 12)

Driven, all fail-driven before commit: the three arms above over a re-derived archived CSV, which
is the **only** place `D`'s division runs on measured intervals — every fixture in
`scripts/remote-exec-test.sh` §9f has `lo == hi` and therefore reaches only `inf` and the floor `1`
— plus §9g's take-four ceiling at `D=17`, whose fixture was **rewritten to the real reading**
because it had been hand-built as `[1398, 1451]` when only the printed width was under test, and a
fixture may not invent an operand. The retired marker's absence is asserted **by name** rather than
left to four exact-match strings, and `D`'s presence is asserted against the range's presence, so a
renderer that dropped `D` fails on one arm instead of on all of them; both new assertions were
positive-controlled by suppressing `D` and by re-emitting the marker, and each failed as written.
Unexercised: `D` on a row whose interval is unbounded, which is unreachable — `tools/benchci`
unbounds `ci`/`lo`/`hi` together on one flag (pinned by
`TestWriteCSVColumnsAndUnboundedTogether`) and the `ci == inf` branch returns before `D`. A real limit rather than a dead
branch, and its mechanism is not where I first put it: on a **rate** unit `D` is quantised by the
*samples*, not by any formatting in this tree. Go renders a `ReportMetric` rate at 4 significant
figures, so keel-skx's ceiling samples are literally the integers `1444 1445 1446` in the raw log,
every order statistic over them is an integer, and `D=17` above is `17/1` on a window whose true
width is somewhere in `(0, 2)`. `tools/benchci` is not the cause — it writes `lo`/`hi` with
`fmt.Sprint`, full precision — so a finer rate `D` would need the *benchmark* to report more digits.
`sec/op` has no such coarseness (the same arm's samples are 6-digit nanosecond integers), which is
why the `sec/op` reading of an arm is the one to trust when the two disagree. Instances 1 and 3
above quote `GFLOP/s` anyway, and on purpose: instance 1 **is** the gate's ceiling denominator,
which `bench_gflops` reads in `GFLOP/s` and in no other unit, and instance 3's second row is the
`GFLOP/s` row the sorted table selected. Both are quoted in the unit whose blindness actually
mattered, with the coarseness disclosed here rather than swapped away.

## Rule 21 — a record of deltas cannot see an injection, so what decides a measurement is stated totally

*Ruled 2026-08-29. keel's first release campaign, `detach.sh`-launched, which paid for three
AWS instances and benchmarked a lab box. Scott's elevation of three findings off that run to
permanent standing.*

### What happened, and the tell

`scripts/aws-fleet.sh up` launched a three-host judged fleet and wrote `.keel-hosts` correctly.
`tmux new-session` seeds a session from the tmux **server's** environment, and that server —
older than the run, pinned alive forever by `exit-empty off` — held `KEEL_REMOTE_HOSTS=antares`
from an earlier single-host session. `scripts/remote.sh:35` ranks that variable above
`.keel-hosts`. So the fleet billed at ~$24/hour while the gate measured a lab machine, and up to
**$192.70** bought nothing, of which roughly $170 was idle billing *after* the run had already
finished, because nothing woke anyone to read it.

**The tell was `gate-p5: RED` with zero FAILs.** Nothing was judged and refused; a measurement
was missing. The lone `UNMEASURED` said `all 1 configured host(s)` where three were paid for.
That diagnosis is available from the tally line alone, before any log is opened, and it is
available *only* because the verdict vocabulary distinguishes a refusal from an absence (rule 6).
A gate whose failure states were one colour would have reported this as a regression in the code.
So the first consequence is about output, not about environments: **the tally line is a
diagnostic instrument, and RED with zero FAILs is its statement that the apparatus, not the
subject, is what failed.**

### A stated assumption is not a check, and what it buys is diagnosis rather than prevention

The certificate this run was meant to produce ends with two lines headed *stated assumptions
(trusted, not verified; not verdicts)*, the first of which reads: **"the configured host set is
the fleet this gate is meant to measure."** That is exactly the proposition Run A violated. The
disclosure was correct, current, and load-bearing in the diagnosis — and it prevented nothing,
because a sentence naming what a gate trusts does not cause the gate to check it.

This is not an argument against the disclosure; the disclosure is why the cause was found in
minutes. It is an argument about what the disclosure is *for*. An assumption printed beside a
verdict converts a mystery into a lookup. It does not convert a trusted proposition into a
measured one, and treating it as though it does is how a known gap survives a run that costs
money. Where the trusted proposition decides **what is measured**, the honest response to
finding it in that list is to move it out of the list — which is what the guard below does.

### The asymmetry: carry and inherit are two directions of one line, and the malignant half was left open

The same comment block in `detach.sh` had diagnosed this mechanism correctly a day earlier, on
2026-08-28, and fixed **half** of it: the direction in which an override the caller *sets* is
dropped unless that very call happens to start the server. That is the benign half.

  - **A dropped override fails toward the configured fleet** — toward `.keel-hosts`, toward the
    thing the run was set up to measure. The cost is a wasted intention.
  - **An injected one fails toward whatever was last measured** — toward a value chosen by an
    unrelated session on an unknown date. The cost is a plausible measurement of the wrong
    subject.

Both directions are the same line of code and they are **separate defects**, because they fail
toward different places. Closing the benign half is not partial progress on the malignant one; it
is a fix whose success is evidence about the other direction only if someone tests the other
direction. Nobody did, and the untested half is the one that bills.

### A record of deltas cannot see an injection

`build/<name>.cmd` — the runner file `detach.sh` writes, whose whole purpose is to be the record
of what the run was — was **clean**. It enumerated what the caller had `export`ed, faithfully and
completely. Ambient state that was already wrong before the caller arrived is invisible to that
record *by construction*: there is no delta to enumerate, because nothing changed.

So the provenance rule: **an environment record must be total rather than incremental.** The
runner now clears the whole `KEEL_`/`BENCH_` namespace before re-exporting the carried set, which
makes the launched program a complete statement of itself rather than a diff against an unknown
base. This is the same blind spot as a checker that is silent about input it never parsed, moved
from parsing to enumeration: in both cases the absent thing produces output identical to the
healthy thing.

The corollary binds two claims that read as one and are not: *"the launcher carries my
overrides"* and *"the launcher cannot inject anything I did not set"* need two tests.

### What landed

  - `scripts/detach.sh` clears `KEEL_`/`BENCH_` wholesale in the generated runner, then
    re-exports the carried set, and echoes the carried **names** back to the caller — the
    failure it replaces was silent, so a caller who names a fleet sees it named back.
  - `scripts/aws-fleet.sh`'s `up` **refuses to launch** while `$KEEL_REMOTE_HOSTS` is set, unless
    `--measure-the-override` names that as the intent. Placed at the top of `cmd_up` so it is
    fatal before the first billable call: a warning in a detached run is read by nobody, and the
    money is spent by the time anyone reads it. The flag **honours** the override rather than
    clearing it — a flag that cleared it would be nothing but a way past the guard, which is the
    shape of an exemption wearing a checkbox.
  - The namespace deliberately left alone is stated at the site: `PATH`, `GOEXPERIMENT` and
    `GOMAXPROCS` are carried, `PATH` because it picks the `go` that builds the arms.

### Coverage, and what stays unexercised (§5 rule 12)

The guard was driven in all four arms before landing — override without the flag (refuses),
override with the flag (announces, proceeds), no override (silent), and against the real stale
value `antares` that cost the $192.70 (refuses) — and the probe needed a **positive control** to
mean anything: the first attempt set `KEEL_SPAWN=/nonexistent-spawn`, which trips the
launcher-presence check at top level *above* `cmd_up`, so every arm printed the same error and
the test discriminated nothing. An executable stub passes that check and dies downstream at
`fleet_json`, so "could not read the launcher's inventory" is the control proving the guard was
reached rather than skipped.

Three things this does not cover, stated rather than implied.

  1. **No arm of that probe launches anything.** What is exercised is the guard's decision, not
     its behaviour during a real launch, and deliberately so — the cheapest honest test of a
     money-spending path stops before the money.
  2. **`scripts/detach.sh` has no automated test at all**, verified rather than assumed: no file
     under `scripts/` names it. Its runner-generation is the code that decides what every
     detached measurement measures, and the only witness it has ever had is the pair of
     hand-driven directions above. That is rule 20's `bench_describe` shape exactly — driven once
     by hand, with nothing re-driving it — and it is the reason this rule's own fix could regress
     silently. Not repaired here: this ruling lands inside a release freeze whose condition is
     that the delta touch nothing the gate reads, and adding a test script is shell spent against
     that condition. It is a named debt with an owner, not a limitation (`#122`).
  3. **The tally-line tell is a reading discipline, not a check.** Nothing in the tree asserts
     that RED-with-zero-FAILs means look for a missing measurement. What is in the tree is the
     vocabulary that makes the inference possible; the inference itself lives here.

### Amendment, 2026-08-31 (`#146`) — a total restatement is only as total as the set of channels it enumerates

*Scott's words, ruling on `#146`: "rule 21's `.cmd` must enumerate the gate's full input closure,
env **and** files **and** defaults. That's rule 21 and rule 22 discovering they're the same rule
from opposite ends: the certificate's input closure is what the declaration must state, not merely
what the clear must reset."*

The fix above — clear the whole `KEEL_`/`BENCH_` namespace, re-export the carried set — was correct,
was tested, and held. Two days later the same shape arrived through a channel an `unset` loop cannot
reach. `gate-p3.sh`'s `sentinel_hosts()` ranked `.keel-sentinel` — untracked, gitignored,
machine-local, mtime three weeks old — **above** the configured fleet, so P2's throughput floor was
judged on one lab host (`janus.local`) in the `#113` re-measurement *and in the run that signed
`v0.1.0-a2`*, while `keel-skx`, `keel-zen4` and `keel-zen5` each printed *"not a sentinel, so P2's
floor is not judged here"*. `build/validate113-ba6f286.cmd` was **clean** — for the second time, by
a second mechanism, and for a reason that is structural rather than an oversight: the record was
complete over the channel it enumerated and silent over the one it did not.

So the requirement is not *clear the environment*. It is **enumerate the channels**, and a run's
record now states three:

| channel | how it is stated | why that form |
|---|---|---|
| environment | cleared wholesale, then re-exported (the original fix) | a delta cannot see an injection |
| files | every `.keel-*` copied in **verbatim**, with its mtime | a clear cannot reach a file, and a summary of a file is a delta again |
| defaults | by naming the **revision** the tree is at, plus clean/dirty | what a gate decides when nothing is set is a property of the tree, and a launcher has no business knowing any gate's defaults |

Three properties of the file half are load-bearing:

1. **By glob, never by list.** A hardcoded enumeration is exactly how the *next* decision file goes
   unrecorded, which is this rule's own failure mode applied to its own fix.
2. **Unfiltered, including files git could recover.** The obvious filter — skip tracked, clean files
   — would have excused `.keel-hosts`, the file at the centre of the original incident. A filter is
   where the silence comes back.
3. **An empty closure is stated.** A tree with no `.keel-*` prints the header and no members, so
   *enumerated, nothing there* is distinguishable from *never enumerated* — the same distinction as
   `died` versus `never-started`, applied to a channel instead of a run.

**The retirement of the channel is a different ruling and is not this rule's business.**
`.keel-sentinel` stopped being a selection input at all, `$KEEL_SENTINEL_HOST` now **fails** the
gate rather than being silently ignored, and an out-of-fleet sentinel is declared with
`$KEEL_SENTINEL_OUT_OF_FLEET` — a union with the fleet, printed into both the `.cmd` and the log's
declaration row. That is the migration ruling being enforced (judged measurement on AWS; the lab is
the dev tier). What *this* rule owns is only that the record's silence was structural.

**Coverage of the amendment, per §5 rule 12.** Item 2 of the coverage list above — *"`scripts/detach.sh`
has no automated test at all"* — is no longer true, and the debt it named (`#122`) was paid before
this amendment landed: `scripts/detach-test.sh` drives 16 arms, each shown to fail first against a
mutant with the fixing line reverted. This amendment has one of them (a planted `.keel-sentinel`
must reach the `.cmd`; with the copy reverted it does not). `sentinel_hosts` and
`sentinel_declaration` were additionally driven through all four states (fleet-only, declared
out-of-fleet, the retired variable's refusal, and an empty fleet) by a probe outside the tree.

**The on-fleet witness was taken 2026-09-01, and it narrows this list rather than clearing it**
(`archive/pinned8/p2onfleet-afb108e.log`, `gate-p3: GREEN`). Three of the resolver's renderings are
now witnessed by a real gate on real hosts: the fleet-only declaration row, the *"`.keel-sentinel`
exists … and is NOT read"* note with its mtime, and `judged as sentinels: keel-zen5 keel-zen4
keel-skx` followed by three P2-floor verdicts. What stays unexercised **by a gate**, and is
therefore still probe-only: the out-of-fleet branch of the declaration row
(`$KEEL_SENTINEL_OUT_OF_FLEET`), and the retired variable's refusal — which would enter a gate as a
`FAIL` in its tally, a path no gate run has ever executed. Those two are named here rather than
counted as covered because the probe is not the instrument the gate is. The defaults channel is
stated **by reference**, so a reader still has to check out the revision to see a default; that is a
deliberate indirection, not a gap, but it is one an amendment cannot close by asserting.

## Rule 22 — a criterion is applied by the mechanism it names, not by the surface form of its wording

*Ruled 2026-08-29, on questions 2 and 3 of the v0.1.0 release report (`#95`). Two criteria, one
correction: both were written in a shorthand that named a place, and both bind what the place was
shorthand **for**.*

### The criterion, and the two readings it had

The condition set for tagging away from the certified rev was that the delta "must touch
**nothing the gate reads** — no code, no scripts, and specifically not README.md". The delta
contained `scripts/aws-fleet.sh`. Read by its letter the answer is a re-run: a script is in it.
Read by its mechanism the answer is that the certificate transfers iff the gate would render
identically on the tagged tree — a claim about what the gate **builds, executes and reads**, not
about directory names.

The two readings were not settled by preference. The gate's execution closure was computed by
fixpoint from `gate-p5.sh` (9 scripts), each member was stripped of comments and of quoted-string
contents, and the four delta files were searched for in what remained: **0 sites**. `README.md` —
the gate's criterion 9 comparand, and so a file whose absence from the result would be absurd —
returned **6**. That control is what makes the four zeros a reading instead of a broken
instrument, and it earned its keep: the first run of this check handed nine paths to `grep` as one
nonexistent filename and reported the answer I wanted about files it had never opened.

Scott's ruling, and the canonical form the condition now has: *nothing in the gate's input
closure.* `aws-fleet.sh` is the **launcher** — it decides which hosts exist before any gate runs,
it computes nothing the certificate attests to, and the refusal added to it fires at launch time
in runs that have not happened yet. Holding the letter over that computed boundary would have
spent about $70 re-certifying a tree the gate provably renders identically on.

### The same correction on the budget rule

Question 3 was the `scripts/` cap: **+11** lines with no routine, kernel, or library fix to pay
for them. Read as ledger arithmetic the rule offers two exits, and **both destroy something the
rule exists to protect**. Compressing the new guard's comments deletes the lines recording what
the $192.70 bought — evidence, spent to satisfy a line counter. Manufacturing a library fix to
balance the ledger puts Go in the tag delta, and triggers exactly the re-run the first ruling
had just declined. The cap exists to stop apparatus sprawl. So: accepted as a **disclosed
overage**, on the post-tag paydown shelf with the rest.

### The guardrail, which is the whole rule

Nothing here licenses reading a rule loosely because it binds inconveniently. Three things held,
and the rule is available only when all three do:

1. **The mechanism is computed, not asserted** — and positive-controlled, so the check could have
   come out otherwise (rule 7). "It's only a launcher" as a claim is worth nothing; a 9-script
   closure with README.md at 6 sites is worth something.
2. **Both readings are stated in the artifact, with the seam named.** The release notes and the
   tag object both say in as many words that the delta includes a file under `scripts/`, and that
   this satisfies the operative test while failing a literal reading of the condition.
3. **The authority accepts the reading.** The disclosure was written and pushed *before* any
   ruling, so what was ruled on had the letter-violation in front of it.

One step away from this rule is deciding the lenient reading silently and reporting only the
conclusion — which in the artifact is **indistinguishable from having done the work**. The
distinguishing evidence is that the seam is named somewhere a reader who disagrees can act on it.

This is rule 14's move applied to a criterion rather than to a defect: rule 14 grades a defect by
what consults it and not by its code; this grades a file by what depends on it and not by the
directory it sits in.

### Coverage, and what stays unseen (§5 rule 12)

The closure is over **statically named** script invocations, and three things bound it.

1. **No `eval` in executable position anywhere in the closure** — verified, not assumed. The one
   occurrence of the word is prose at `scripts/roofline.sh:515`; the control is that three files
   under `scripts/` do contain it, so the pattern finds the word where it exists.
2. **The gate executes shell that exists in no tracked file.** `gate-p3.sh:295` pipes a heredoc to
   `bash -s` over ssh, and `remote.sh:1453` writes a runner and ships it to the host. Both
   *generators* are closure members, so a delta file could reach the far side only as text in one
   of the nine — which is what was searched. This is not a hole in the result; it is why the
   search had to be over generators rather than over an inventory of what runs.
3. **Nothing in the tree recomputes the closure.** It was computed once, by hand, for one release.
   The next release recomputes it rather than citing this number, because the closure is a
   property of the scripts at that revision. Not filed: the method above is the deliverable, and a
   fix smaller than its issue gets fixed rather than filed.

## Rule 23 — an absence claim states the scope its instrument actually searched

*Ruled 2026-08-30, on the second self-caught instance in two days. Scott: "an instrument scoped to
less than the claim, reporting in the subject's voice". The law he drew from it: that scope must
cover the claim's scope, or the finding is `unmeasured` rather than absent.*

### The two instances, which are one shape

| # | claim made | instrument | its actual scope | why it read as a finding |
|---|---|---|---|---|
| 1 | the 110× spill price is "25× larger than anything the report contains" | `docs/spill-report.md` | janus, vesta, antares — **no Sapphire Rapids** | true of the report, false of the tree; the adjective *µarch* named the scope and the comparand held it fixed |
| 2 | "neither `30.5%` nor `0.278%` appears in #104" | `gh issue view 104 --json body` | the opening post only | true of the body, false of the issue; the table was in a **comment**, worded exactly as cited |

The second was the more expensive: it "corrected" a **correct** citation in `CHANGELOG.md`,
`docs/upstream-plan.md` and a pushed commit message (`3a7ad60`), reverted at `110c4ec`. Both came
from a tool rather than a guess, which is precisely what lent them authority.

### Why the class survives being written down

**The wrong reading is always nearly right.** #104 does contain a second SPR peak (228.9) and rate
(0.6526), yielding 107.1× — near enough to 110× that the false correction looked arithmetically
motivated rather than unsearched. A near-miss on a plausible neighbour is the tell, and it is the
reason a scope error presents as a numerical one.

### What binds now

1. **Name the extent before concluding a string is absent.** For a GitHub citation, `--json
   body,comments`. For a file, the whole file rather than the section under discussion. For a
   fleet claim, the host set.
2. **Prefer the three-state resolution.** Real, and *off-subject*, is a verdict. Reaching for
   refuted-or-confirmed on a two-state axis is what converted a scope error into a numerical one
   both times.
3. **This is not rule 11.** Rule 11 says the instrument adjudicates; rule 23 says **which**
   instrument may. Neither may be cited for the other's job — the same separation 11 and 12
   already carry.

### Coverage, and what stays unseen (§5 rule 12)

**No mechanical check enforces this and none is proposed.** The failure is a mismatch between a
claim's scope and a query's scope, both of which are prose at the moment they are written; a linter
would have to read the claim. What the tree gets instead is the discipline list in
`docs/upstream-plan.md` §"The discipline every item inherits", item 2, which now names the
`--json body,comments` form for the one instrument that produced instance 2. The other scopes —
host sets, file extents — are unguarded, and both instances were caught by re-derivation rather
than by any check, which is the only mechanism this rule can honestly claim.

## Rule 24 — a pre-registered prediction is stated in the instrument's own output space, and at its row granularity

**Adopted 2026-08-31**, ruling on the #113 certificate comparison. Scott's ruling on Scott's own
prediction: *"the prediction failure is mine to own, and it earns a law: a pre-registered
prediction must be stated in the instrument's own output space. I supplied two prose deltas that
were digit-only, to be checked by a normalization that erases digits — so the table predicted
precisely what the instrument cannot render and nothing it can."*

### The instrument, and what it can render

The comparison that validates a re-measured certificate against its pinned one reduces both logs to
a set of normalized verdict keys:

    LC_ALL=C sed -E 's/\x1b\[[0-9;]*m//g' "$1" \
    | /usr/bin/grep -E '^  (PASS|FAIL|UNMEASURED|BASELINE|REPORTED) ' \
    | LC_ALL=C sed -E 's/zen4/zenFOUR/g; s/zen5/zenFIVE/g' \
    | LC_ALL=C sed -E 's/[0-9]+\.[0-9]+/#/g; s/[0-9]+/#/g' \
    | LC_ALL=C sed -E 's/zenFOUR/zen4/g; s/zenFIVE/zen5/g'

ANSI stripped because the logs carry `^[[32mPASS^[[0m`; the two-space anchor because `gate-p5`'s own
verdicts sit at exactly that indent while delegated ones are deeper; the letter mask because the
digit pass would otherwise collapse `keel-zen4` and `keel-zen5` into one host — which it did on the
first attempt, reading 58 and 57 distinct keys instead of 72, and would have published a wrong
member count. **The output space is wording.** Every numeral in it is `#`.

Both pre-registered deltas were numerals: `8 row(s)` → `9 row(s)` and `6.067x` → `6.066x`. Both were
confirmed against the raw logs, and **each accounts for 0 of the 7 changed members**, because in this
instrument both readings are `# row(s)` and `#x`.

### The reading that let it pass: injectivity is not identity

Both logs yield **72 verdict lines and 72 distinct keys**. That is an *injectivity* property — no two
verdicts of one run normalize together, which is what makes the key usable as an identity at all —
and it says nothing whatever about whether the two *sets* coincide. Read as agreement, it is
compatible with all 72 members differing. The set question needs a set operation, and it was run
only after the report had already been published:

    comm -3 <(norm pinned | sort) <(norm remeasured | sort)   # 7 lines each side

### The seven, classified individually — because "7 differ" is a count and the finding is which

Scott: *"Classify the seven individually in the record… because '7 differ' is a count, and the
finding is which."* `68a9bec` is the pinned release-a2 certificate (72 PASS / 0 FAIL, GREEN);
`ba6f286` is the #113 re-measurement (71 PASS / 1 FAIL, RED). Every attribution below is measured —
the wording changes by `git log -S … -- scripts/gate-p5.sh`, the numbers by re-reading both logs.

| # | Members | Class | Cause | Attributed to |
|---:|---:|---|---|---|
| 1 | **1** | **the run's actual FAIL** | `PASS  gate-p# is green on this commit (#a#bec), so every rate this gate divided by is still a measured one` became `FAIL  gate-p# is RED on this commit (exit #), so nothing above that divides by a measured rate means what it says` | `ba6f286`'s delegated `gate-p4`, itself RED on `gate-p3`'s `janus.local` sentinel |
| 2 | **3** | wording, per host | the Strsm class line's `>= #x, the regression bar ratified for this class` became `>= #x, the fleet bar ratified for this class`, once each on `keel-skx`, `keel-zen4`, `keel-zen5` | `75aae82` (#119) — **reported** in the first write-up |
| 3 | **1** | wording, aggregate | the single line summing the class bars: `…#x of its own single-thread rate for Strsm) (#/#)` became `…#x of its own single-thread rate for Strsm on the hosts that derived that floor, and a registered baseline less #x elsewhere (##)) (#/#)` | `75aae82` (#119) — **the second surface, NOT reported**, and why *"that is the entire diff"* was false |
| 4 | **2** | decoration | the peak-series window line, on `keel-skx` and `keel-zen4` only: `head->middle -#% … middle->tail +#%` became `head->middle +#% … middle->tail -#%` | measurement noise; **no verdict moved** — both runs render `so the windows are ties and a tie is not an order` |

That is 1 + 3 + 1 + 2 = 7, and the four classes are four different kinds of thing: one is the
result, four are one commit's wording (three of which were reported and one of which was not), and
two are decoration.

**Members 4 are worth their own paragraph, because they are the instrument's mirror image.** The
window steps are all an order of magnitude below their own floors in both runs, so both runs render
the identical tie verdict; only the *sign* of an immaterial step changed:

                 68a9bec  h->m       m->t          ba6f286  h->m       m->t
    keel-zen5             +0.1082%   -0.2246%               +0.2542%   -0.0900%   <- signs agree
    keel-zen4             -0.0186%   +0.0167%               +0.0112%   -0.0147%   <- flipped
    keel-skx              -0.0313%   +0.0052%               +0.0527%   -0.0472%   <- flipped

The normalization **erases the magnitude and keeps the sign**, so a step of 0.0112% and a step of
0.0186% are the same key while their signs are not — and `keel-zen5` differs only by having drawn
the same signs twice. So the same instrument that suppressed both predictions promoted two
non-events to full members, and 2 of 7 are noise wearing a member's clothes. That asymmetry is a
finding about the comparison, not about the run, and it is the third of #113's open questions.

### What binds now

1. **Push each predicted delta through the comparison before registering it.** If the normalization
   erases it, predict it against an instrument that can see it — `9 row(s)` is checkable by the
   criterion that prints it — or register it explicitly as invisible here. A prediction the
   instrument cannot render is not a weak constraint; it is none, wearing rigour.
2. **Predict a count in the instrument's own terms, and settle it with a set operation** in both
   directions. `comm -3`, not an eyeballed diff, and never over abridged output.
3. **A confirmed prediction is evidence about what it named.** It is not agreement about the
   artifact. `72/72` is injectivity; `|symmetric difference| = 0` is identity; only the second
   licenses *"that is the entire diff."*
4. **This is not rule 12.** Rule 12 says a coverage number states what it cannot see — a number
   that exists, with a stated hole. This says a prediction stated in units the instrument does not
   render sees *nothing*, so there is no number to qualify. Neither may be cited for the other.

### Coverage, and what stays unseen (§5 rule 12)

**No check enforces this, and the honest reason is that the comparison is not a script in this
tree** — it was assembled at the terminal for #113 and lives here as the block above, so there is no
place to add an assertion that a registered prediction survives normalization. Two specific holes,
stated rather than left implicit. (a) The classification above is only as complete as the
`^  (PASS|FAIL|…) ` anchor: both logs also carry deeper-indented delegated verdicts and prose lines
that no member could ever come from, and a change confined to those is invisible to this rule's
whole apparatus. (b) The sign-versus-magnitude asymmetry is **named here and unfixed** — narrowing
the normalization to erase signs too would suppress a real sign change on a step that mattered, and
that trade has not been measured, so it stays an open question on #113 rather than a silent choice
made in a `sed` expression.

### The second clause — row granularity, added 2026-09-01, one day after the first

**Adopted 2026-09-01**, ruling on #141's fork. Scott: *"a pre-registered predicate governs exactly
the rows it registered thresholds for — supplying thresholds after seeing the kc≠128 deltas would be
fresh judgment wearing pre-registration's clothes, which is the flattery-asymmetry's oldest trick in
either direction."* The headline gains its second half: right output space, **and** the instrument's
own row granularity — thresholds for every row the driver renders, or those rows are declared out of
domain in advance.

The first clause is about *units*: a prediction the normalization erases sees nothing. This one is
about *extent*: a prediction that names a family sees rows nobody characterised, and inherits a
verdict for each of them. They are the same failure — a predicate whose reach exceeds its
calibration — arriving from two directions, and neither is a special case of the other.

#### The predicate, and the rows it actually reached

The fork was registered over `Kernel/*/avx512`, and its thresholds came from `kc=128` — the only
panel depth `gate-p5` measures, so the only one with prior data. The drift driver's filter is
`BenchmarkKernel|BenchmarkPeak`, which on janus renders **four `kc` values per shape**. Re-derived
mechanically rather than recalled, from the tracked arm logs
`archive/pinned8/drift-janus-df999da-{base,new}.log` — both arms render the identical 23-row set, so
the partition is a property of the filter and the host and not of one arm:

| | rows | state |
|---|---:|---|
| predicate scope, thresholded (`2x32`, `4x32`, `6x32` at `kc=128`) | **3** | adjudicated |
| predicate scope, no threshold ever derived (`kc` = 8, 32, 512 × three shapes) | **9** | **`UNMEASURED`** |
| outside the predicate, declared out of domain in advance (8 `scalar` — `6x32` has no scalar arm — and 3 `Peak`) | **11** | out of domain |
| total rendered | **23** | |

**A count of my own, corrected in place rather than left standing.** The first write-up of this said
the predicate "covers 12 AVX-512 rows for which no threshold was ever derived", and the ruling
repeated the figure as *"the twelve unthresholded rows"*. Twelve is the predicate's whole **scope**;
**nine** were unthresholded. The shortfall and the reach are two quantities and the sentence used one
number for both — which is the same shape as rule 24's own subject, a claim stated at the wrong
granularity, so it is corrected here and inside DESIGN.md's rule text rather than being allowed to
stand because a ruling had already quoted it.

#### Why this is a defect and not a close call: one table returned both branches

Restricted to the three rows it had calibrated, the fork returned **transient** decisively. Read
literally over all twelve, it returned the other branch, on the nine rows nobody had characterised.
Both readings were true of one table, computed from one set of samples, by one predicate. **A fork
that can return either branch is not a pre-registration**, whatever its timestamp; the free
parameter simply moved from the threshold to the row set.

The verdict for a reached-but-unthresholded row is therefore **`UNMEASURED`** — the vocabulary rule 6
already provides, and the same third state rule 19 gives a row too wide to adjudicate. Not a pass,
not a fail, and not a quiet exclusion: the row prints, its delta is archived, and what is refused is
only the comparison against a bar that does not exist.

#### What the ruling settled on the merits, separately

#141's own question was answered on the evidence that *was* adjudicable, and Scott stated it as such:
*"Transient. The certificate's reading holds, now with two independent reproductions."* The grounds
were three, none of them the unthresholded rows: bands **11.4× narrower** than the excursion's,
medians agreeing to **+0.4%**, and a sentinel share of **47.94 / 47.83** against the certificate's
**47.90**. The kc≠128 direction-split does not vanish by being declared `UNMEASURED` — it **files as
a new finding**, and the trimpath witness is what will attribute it.

#### What binds now, beyond the first clause's three

(d) **Enumerate the exact rows the driver will print, before registering anything.** Run the filter,
or read an archived log produced by the same filter. The 23 rows above cost one `grep | sort -u` over
a log that already existed.

(e) **A row with no prior data is declared out of domain in the pre-registration itself**, in the same
sentence and with the same standing as an ambiguous band. That declaration is cheap before the fact
and unavailable after it.

(f) **When the two readings disagree, publish both, name the defect as the predicate's, and do not
retro-scope.** Supplying the missing thresholds afterwards is the amend-only-with-a-predating-standard
rule violated against one's own prediction, and rule 16's asymmetry says why it is not self-correcting:
a draw that flatters gets a threshold and a draw that indicts gets reopened, so the direction of the
repair is decided by the result it is repairing. Posting both readings without picking was ruled
correct; the pick belongs to the authority.

#### Coverage, and what stays unseen (§5 rule 12)

**No check enforces this either, and the shape of the hole is different from the first clause's.** The
row set is a *runtime* property of the driver: `AB_FILTER` is a regex, the shapes come from the
benchmark's own table, and which arms exist depends on the host's CPU features — janus renders no
`6x32/scalar` at all, and a host without AVX-512 would render none of the predicate's twelve. So an
enumeration is per host and per revision, and there is nothing to assert against at registration time
except a log from the same filter. Two specific limits. (a) The partition above is exact for
`df999da` on `janus.local` and is **not** portable: the same predicate on a different host has a
different scope, a different thresholded subset, and therefore a different `UNMEASURED` count. (b)
The enumeration is only as right as the pattern that runs it, and it has the same failure mode as its
subject: a bare `avx512` over the row list returns **13**, not 12, because `BenchmarkPeak/avx512`
matches a predicate written for `Kernel/*/avx512`. One row of scope error, in the check built to
catch scope errors, and it is caught here only because the four counts were made to sum to 23.

### The third clause — judge raw, print rounded, added 2026-09-01, the same day as the second

**Adopted 2026-09-01**, ruling on #147. Scott took neither option as posed — neither "naming the
column is enough" nor "every threshold carries a ≥0.05pp guard band":

> Neither option as posed — the existing law reaches down a layer. The finding is the display-quantum
> disease at the pre-registration layer, and the cure is the one already on the books: **judge raw,
> print rounded.** Predicates bind to a *named column* **and** adjudicate at full precision from the
> raw samples — never from the printed table. The guard band survives only for the historical case
> where raw samples don't exist: there, the boundary band is **one display quantum of the named
> column at that magnitude** — derived, not a fixed 0.05pp — and a comparison landing inside it
> renders neither-branch. That grounds the band in the claim's own resolution, same as rule 20's
> amendment, instead of minting a new constant.

#### The measurement that forced it

Per sample `GFLOP/s` is exactly `flops/call ÷ ns/op`, verified to the printed digit
(`3072.0 / 44.67 = 68.770987`, reported `68.77`). But `testing` formats each metric to **four
significant figures independently**, so one ulp is 0.023% at `43.85` and 0.014% at `70.08` and each
column carries its own quantization. Across #141's twelve predicate rows the `|sec/op|` vs
`|GFLOP/s|` discrepancy is **0.003–0.041 percentage points**, and two of the twelve sat inside that
band and **changed disposition between the two halves of one table**:

| row | `sec/op` | `GFLOP/s` | threshold | dispositions |
|---|---:|---:|---:|---|
| `Kernel/6x32/avx512/kc=8` | **−1.9954%** | **+2.0049%** | 2% | no vote / **dirty** |
| `Kernel/4x32/avx512/kc=512` | **−0.2945%** | **+0.3109%** | 0.30% | clean / no vote |

Under `GFLOP/s`, #141's branch S would have fired. What resolved it legitimately is that the
pre-registration named `sec/op` in advance — so the column choice was a decision and not a
convenience, which is the first clause doing its job one layer up from where it was written.

**A refuted explanation, kept because the refutation is the useful part.** The first cause reached
for was an even-`n` median interacting with the modes of #147's bimodal rows. Measured, the
discrepancy is **flat across all twelve rows** rather than concentrated on the bimodal one, which
refutes it. Two independent findings — modal samples and independent quantization — were one
sentence away from being fused into a single causal story with one mechanism carrying both.

#### What binds now, beyond the first three clauses

(g) **A predicate names its metric column, and adjudicates from the raw samples at full precision.**
The printed table is the rendering; the comparison operand is the value. Same discipline as rule 20's
`D`, computed from the real asymmetric bounds `hi − lo` and never from the printed `± W%`.

(h) **Where only historical logs exist, the band is derived, not chosen**: one display quantum of the
named column at the magnitude in play. A comparison landing inside it renders **neither branch** —
the `UNMEASURED` third state again, not a pass and not a fail.

#### Coverage, and what stays unseen (§5 rule 12)

No check enforces either clause. (a) The band is per magnitude, so it cannot be pinned as a constant
anywhere — `0.023%` at `43.85 ns` and `0.014%` at `70.08 GFLOP/s` are the same rule with different
values, and a fixture would have to carry the magnitudes to be meaningful. (b) The twelve-row
discrepancy figure is exact for W1 on `janus.local` and is **not** portable: it is a property of the
magnitudes those rows print at, so a host whose rates round differently has a different band. (c)
What the measurement cannot distinguish is a genuine sub-quantum difference from a rounding artifact
— by construction, since it is the resolution of the display — so the clause refuses rather than
resolves, and that is the point.

## Rule 25 — an A/B arm is an estimator, never a draw

**Adopted 2026-09-01**, ruling on #147. Scott, on the witness's outcome:

> The compared quantity is *modal*, and an n=1-vs-n=1 comparison cannot adjudicate a modal quantity
> no matter how tight each draw looks. That's rule 16 arriving at the comparison layer — *an arm is
> an estimator too, never a draw* — and it retroactively explains why this corner of the fleet has
> resisted three attribution attempts: everyone kept comparing single draws of a distribution that
> has more than one place to land.

### Why it is its own item and not a widening of rule 16

Rule 16's object is a **published reference** — a number minted for a later run to be checked
against, a README rate, a registered baseline, an archived denominator. An A/B arm is not that: it is
consumed inside its own run and published only as evidence for that run's verdict. Citing 16 for an
arm therefore lands on a **real-but-different rule and reads as well-formed**, which is the mis-mint
class the concordance audit exists to catch. Scott's ruling on the catch:

> It becomes its own item, cross-linked to 16 as the same shape on a new object — the constitution's
> established pattern — rather than a widening that would retroactively blur every existing citation
> of 16.

So the two rules are siblings over one shape, and neither is a special case of the other: 16 governs
a number that outlives its run, 25 governs a number consumed inside one.

### What happened

#141's `-trimpath` witness (W1) was a null A/B — one revision, both arms, so every delta is the host
between two windows. Its non-timing half worked exactly as ruled: both arms print
`sha256=34a87563622bb6c3…`, `bytes=5066630`, `flags=[-buildmode=exe -compiler=gc -trimpath=true]`,
the base arm having built in a worktree at a different path, so **the two arms are literally one
binary** and the two-binaries class is dead. The timing half was another matter. Classifying all 690
per-sample rows per arm — largest sorted-adjacent gap over 1.5% of the median **with both clusters
≥3 members** — partitions the three draws' 138 (row, arm) cells into **18 bimodal / 20
single-outlier / 100 clean**. The cluster-size condition is load-bearing: without it, 20 single wild
samples read as modes.

It tracks the **backend**, not `kc`: scalar 12 bimodal + 16 outlier of 48 cells, avx512 6 + 4 of 72,
`Peak/*` **0 of 18**. By `kc` the bimodal counts are 6/3/4/5 across 8/32/128/512 — flat, which
**refutes** the prediction (mine, from the allocation-size hypothesis) that it would concentrate at
small `kc`. And no row is bimodal in all six arms: the `≥2%` set re-rolled across the three draws as
`{2x32/kc=32, 2x32/kc=512, 6x32/kc=8}` → `{2x32/kc=512, 6x32/kc=128}` → `{}`. **Bimodality is a
property of the draw, not of the row** — which is exactly what makes a one-window arm an estimator
with unstated variance rather than an observation.

### The two signatures, and the discriminator

`testing`'s `launch()` sizes the final `b.N` from the ramp's measured rate plus a 20% margin, so
elapsed lands near `1.2 × benchtime`. **Measured on this data rather than recalled: 1.2043 s**, the
median over 240 samples from four unimodal rows. With that constant, predict each mode's `b.N` from
its *own* rate and from the *other* mode's rate and see which fits. Ten of the 18 cells are decisive
(best fit under 1%, beating the alternative by more than 3×).

| | signature A | signature B |
|---|---|---|
| cells | 2 | 8 |
| pattern | `HLHL…` strict period-2, 15\|15, slow mode on even `k` **15/15 in two independent runs** | long contiguous blocks, e.g. `HHHHHHHHHHHHHLLLLLLLLLLLLLLLLL` |
| gap | 3.81% / 3.88% | 15.52% |
| `b.N` fits | the **other** mode's rate, error **0.23% / 0.14%** | its **own** rate, errors **0.24–0.73%** |
| elapsed | **off target**: 1.2515 s vs 1.1560 s, ratio 1.0826 | on target: 1.2005 s vs 1.1956 s, ratio 1.0041 |
| reading | the mode **flips** between the ramp and the timed run | the mode **persists** across both |

Two artifact explanations are refuted. Reported `ns/op` is the timed run's own elapsed over its own
`b.N`, so a mis-sized `b.N` scales elapsed and leaves the rate unchanged. And cold-start
amortization has the **wrong sign** for signature A specifically: the slow mode carries *more*
iterations, and amortizing a fixed overhead over more iterations pushes `ns/op` down, not up.

**No mechanism is attributed.** Two hypotheses, each with a decisive test: per-execution allocation
placement (`BenchmarkKernel` calls `kernelPanels` **inside** the `b.Run` closure, `bench/kernel_test.go:50`,
so every ramp step *and* the timed run freshly `make`s three panels), and core/frequency placement as
in #15. `Peak/*` clean in 18/18 is suggestive of a memory-side story but **cannot separate them** —
`BenchmarkPeak` is the only benchmark that allocates nothing *and* touches no memory while timed, so
the two hypotheses share one control — and it is #14's argument on new data, **one witness, not two**
(§5 rule 10).

### What binds now

(a) **A probe of a modal quantity is multi-draw per arm with the mode structure characterized first,
or it does not run.** Scott: *"the next probe, if one runs, is multi-draw per arm with the mode
structure characterized first, or it doesn't run."*

(b) **The characterization states its own test before running it** — the statistic, not merely the
intent. For the authorized allocation-address test that means defining what counts as period-2
before any addresses are printed, on the same standing as any other pre-registration (rule 24).

(c) **A classifier built after the fact is a description, not a test, and says so.** The 1.5%/≥3
threshold above was chosen after seeing the data; it is stated that way wherever it is cited.

(d) **Before reading any A/B delta, ask whether the row is modal.** `go test -count` cannot answer
it across runs — every sample sits in one contiguous window — which is the same property that makes
it the inner loop and not the outer one.

### What it deliberately does not order

**No modality detector, and no criterion amendment.** Scott:

> No amendment — and here's the standard, stated now so it predates any result it might ever judge:
> the #110/#116-class criteria amend for modality **only if a verdict on a judged fleet row is
> demonstrated to have flipped via modal structure.** Until then #147 is a disclosure, and note the
> instruments for *seeing* modality already exist threshold-free — rule 20's range-beside-CI and
> #132's disparity statistic are precisely modality's signature made visible, and they got that way
> by deleting thresholds. An amendment now would mean building a modality *detector*, which means
> choosing a threshold, which is the class we just finished burying. One dev-tier host's finding
> doesn't reopen that grave.

The residue is likewise **disclosure and not a ledger entry**: the apparatus ledger counts shell
lines, an open question's record is its issue, and #147 holds the state honestly. Scott's grounds:
the residue *gates nothing* — janus is dev-tier, the sentinel is fleet-only by rule, and #141's own
question closed on T — and *"a measurement debt is owed when something judged depends on the answer;
nothing does."*

### Coverage, and what stays unseen (§5 rule 12)

- **One host.** `janus.local` only. Nothing here says whether vesta or antares behave this way; #15
  says vesta does something related *between* runs.
- **One `n`.** Whether the modes have a period longer than a 30-sample window, or a structure the
  classifier misses, is unmeasured.
- **The 20 outlier cells are unexplained**, not classified as clean — possibly the same mechanism
  with an unlucky split, possibly scheduling stalls, possibly both.
- **8 of 18 bimodal cells return no verdict** from the `b.N` discriminator; in one
  (`2x32/scalar/kc=128`, `df/base`) both hypotheses miss by ~270%. The test declines rather than
  picking the smaller error, so the two-signature account covers 10 cells and not 18.
- **No check enforces this rule**, and none plausibly could: whether a quantity is modal is a
  property of the samples, and asserting it automatically is the detector this rule declines to
  order.

## Rule 26 — when the prediction is the null, every treatment needs an applied-witness

**Adopted 2026-09-03**, ruling on #148's test 3. Scott, elevating a finding made while building that
test's driver:

> And the null-inversion insight earns a law, because it generalizes to every experiment this project
> will ever run whose registered prediction is "nothing changes": *when the prediction is the null,
> every treatment needs an applied-witness* — silent treatment-failure produces exactly the predicted
> result, so the usual risk asymmetry inverts and the vacuous-pass family gets a new most-dangerous
> member. Your four refusals are the complete set for this run: the binary digest (absolute
> boundaries transfer to no other artifact — a refusal, correctly, not a note), the inherited-env
> abort (an inherited `GOGC=off` would hand the *control* the treatment), the gctrace witness that
> `GOGC=off` *actually suppresses GC* before any arm runs, and asyncpreemptoff honestly declared
> out-of-domain — arrival proven, honoring not, the action named. That last one is the rule-12 form
> at its best: the experiment states what it cannot witness rather than pretending the set is
> complete.

### The inversion

Every other member of the vacuous-pass family has a tell, and the tell is that the apparatus's
failure *costs* the author the result they wanted: an absent instrument fails closed, a stuck one
prints a constant a reader can recognise, an unqueried cell renders as a dash, a search with a broken
pattern returns nothing where something was expected. The author is therefore the defect's first
victim and has an incentive to notice.

A null prediction reverses the incentive. If the registration says *this treatment changes nothing*,
then a treatment that never arrived leaves the subject sitting at the control level — which is the
predicted reading, digit for digit. The run corroborates its author, the table is clean, no verdict
vocabulary fires, and **corroboration is indistinguishable from an untreated control**. Nothing in
the output complains, because there is nothing for the output to complain about: the apparatus's
failure mode *is* the hypothesis.

### What was at risk

`archive/mech148/predictions-mech148.py` registers **12 (row, arm) cells — four rows × the three
treatment arms — every one `unimodal-at-confined`**, with `PREDICT_COUNTS` all `(60, 0, 0)`: the
prediction is that no treatment moves the confined level at all. Five arms, two of them controls, with
the registration's own `role` strings:

| arm | mask / env | role in the registration | what a silent treatment failure prints |
|---|---|---|---|
| `ref` | `0,1`, `GOMAXPROCS=1` | recovered control | the **confined** level — it reads as the collapse it exists to disprove |
| `c0` | `0`, `GOMAXPROCS=1` | collapse positive control | the confined level, which is also its correct reading |
| `gc` | `0`, `+GOGC=off` | removes GC assist and worker activity | the confined level, i.e. the prediction |
| `pre` | `0`, `+asyncpreemptoff=1` | removes async preemption signals | the confined level, i.e. the prediction |
| `both` | `0`, both | the joint arm | the confined level, i.e. the prediction |

The two controls fail in opposite directions, and that is the point: an unapplied mask makes `ref`
look collapsed, which would **manufacture** a finding, while an inherited `GOGC=off` makes `c0` carry
the `gc` arm's treatment, which would **erase** the contrast. Only the three treatment arms hold cells
the registration scores, and all three of them fail silently *toward* the prediction.

Three concrete mechanisms would have produced that table with no treatment applied anywhere:
`scripts/remote.sh:1687` interpolates `$KEEL_REMOTE_ENV` **unquoted** ahead of `env`, so the
assignments arrive only by word-splitting; a `GOGC` or `GODEBUG` inherited from the launching shell
would hand the **control** arm the treatment, collapsing the contrast from the other side; and a
binary rebuilt since test 2 would make the registered absolute boundaries describe a different
artifact.

### What binds now

**Each treatment names an observable that proves it arrived, that observable is read off the run
itself rather than assumed from the launcher, and a treatment with no available witness is declared
out of domain in the registration rather than carried as a quiet arm.** Four consequences, one per
refusal, with the driver's site for each:

| # | consequence | site | outcome on failure |
|---|---|---|---|
| a | prefer a witness the apparatus already prints | `keel-bench-gomaxprocs:` readback, per arm | `WARNING` + "treat as unmeasured" |
| b | the witness is non-judged and runs before the arms | `gcprobe()`, `^gc [0-9]` counts with and without `GOGC=off` | `exit 9` if suppression is not observed |
| c | provable arrival + unprovable effect gets rule 12's form | `asyncpreemptoff` declared out of domain | disclosed, action named |
| d | what a boundary was derived on is part of the treatment | `WANT_SHA` digest check | `exit 8`, never a rescale |

(a) is the cheapest form and the first place to look: `GOMAXPROCS` **leads every env string** so that
it is the value that breaks first if the word-split ever stops, and `bench/bench_test.go`'s readback
was *already printing* the value — the rule cost an ordering and an assertion, not a mechanism. The
inherited-env abort (`exit 4` over `KEEL_PIN_CPUS`, `GOGC`, `GODEBUG`, `GOMAXPROCS`) belongs to the
same consequence from the other end: it protects the control rather than the treatment, and under a
null prediction those are the same protection.

(d) is worth stating as a refusal rather than a courtesy. The registered boundaries are **absolute
GFLOP/s**, derived on `sha256=d0d46d26c15cc8b2`; the tempting repair when a rebuilt binary shows up
is to rescale them against this run's own reference arm. That would make the boundaries *chosen*, and
chosen in the direction that confirms a null — so the digest mismatch stops the run.

### Direction, and what it is not

Per rule 15, the direction is stated: an applied-witness can only convert a **corroboration into
`UNMEASURED`**. It never manufactures the alternative branch, because failing to prove that a
treatment arrived is silent about what would have happened had it arrived. That is the rule's safety
and equally its limit — it cannot rescue a run, only decline to bank one.

- **Not rule 24.** 24 asks whether a prediction is expressible in the instrument's own output space,
  at its own row granularity. A prediction can be registered in exactly the right units, at the right
  granularity, with thresholds for every row — and still be about nothing, because the treatment it
  names never reached the far side. 24 may not be cited for this.
- **Not rule 12.** 12 governs disclosing the reach of a claim; this governs whether the claim has any
  reach. They meet at consequence (c), where the honest move *is* a 12-style disclosure — which is
  the seam, not an identity.
- **Not rule 7's positive control.** A positive control asks whether the **instrument** can see a
  signal; this asks whether the **subject** was ever given one. The two failures are independent and
  a run can need both, as this one does.

### Coverage, and what stays unseen (§5 rule 12)

- **`asyncpreemptoff`'s honouring is unwitnessed.** Unknown `GODEBUG` keys are ignored silently and
  this binary counts no preemption signals, so `pre` and half of `both` rest on arrival alone. The
  action that would close it is a patched runtime carrying preemption counters. That action is
  already on the registration's record — `OUT_OF_DOMAIN` item 1 names "a patched runtime" as the fix
  for a *different* limitation of this campaign (sysmon, the scavenger, the netpoller) — so the action
  is disclosed while this limitation is not itself an enumerated item, and a limitation with a named
  action is not a debt.
- **The gctrace probe's own failure branch fails *open*.** If the probe returns nothing (an ssh
  hiccup, say), the arms still run and the log says the analyzer must read a confined result as
  consistent with both "no effect" and "no treatment". A network fault is not evidence about `GOGC`,
  so refusing there would let an unrelated hiccup destroy a whole run — but the consequence is that
  this one branch is enforced by **a human reading the log**, not by the driver.
- **The `GOMAXPROCS` canary witnesses the transport, not each variable.** It proves the env string
  word-split; it does not separately prove that `GOGC=off` was among the words that arrived. `gcprobe`
  covers `GOGC` from the runtime side, and nothing covers a hypothetical future variable added to an
  env string without its own witness.
- **No check enforces this rule across the repo**, and the driver's witnesses are exercised by
  `archive/mech148/test-driver-mech148.sh` only for the quietness gate and the arm table. Whether a
  registered prediction *is* a null is a property of the registration's prose, and asserting it
  mechanically is not attempted.
- **Consequence (d) is measured; (a)–(c) are still argued from mechanism.** Updated 2026-09-03, two
  hours after adoption, because the driver's first real launch **refused at `exit 8`** and the
  refusal was correct: the dev host cross-compiles every fleet run (`scripts/remote.sh:662`) and it
  had been upgraded go1.27.0 → go1.27.1, so the binary was `d10b953b924316d8` / 5066561 bytes where
  the boundaries were derived on `d0d46d26c15cc8b2` / 5066553. **No keel input had changed** —
  `git diff 97a21f4..HEAD -- '*.go' go.mod go.sum` is empty — so nothing in the tree, the commit
  log, or the launch command could have revealed it, and every arm would have been scored against
  boundaries belonging to a different artifact while printing the predicted confined level. Cause
  established by **reproduction** rather than inference: rebuilding at `GOTOOLCHAIN=go1.27.0`
  yields `d0d46d26c15cc8b2` / 5066553 exactly, so the registration needed no amendment and the
  driver now pins the toolchain. What (a)–(c) still lack is a live catch of their own: they have
  been driven on healthy paths and by hand only, so their *sensitivity* remains a mechanism
  argument (rule 10 — mechanism plus one healthy run is one witness).
- **The far side's toolchain was not in any witness before this.** The digest caught it downstream,
  which is enough to refuse but not enough to *name* the cause; naming it took a hand diff and a
  rebuild. A run whose log recorded the builder toolchain but had no digest bound to a registration
  would have printed the change and proceeded.
