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
and the fixture is the whole of its coverage.

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
against the recurrence sample rather than inferred from a healthy run. `clock_series` was exercised
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
