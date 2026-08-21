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
