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
5's ruling is still inside it because its 2026-08-16 amendment is live work (#23).

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
