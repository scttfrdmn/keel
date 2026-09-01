# The apparatus ledger

<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

One running total, itemized per session with commit hashes. Ruled 2026-08-31: *"'Booked
separately' would create a second ledger, and two ledgers is how a number gets to be true in
each and wrong in sum."* This file is that one total. Before 2026-08-31 the running total
lived only in [#131](https://github.com/scttfrdmn/keel/issues/131)'s comment thread, which is
itself the mechanism by which a second one could arise — a thread has no single current value,
only a series of claims about one.

The debt is measured in **net lines under the shell term**, which `gate-docs.sh` prints on every
push as the numerator of the apparatus ratio. The cap it discharges is CLAUDE.md's: *a session may
not add net lines to `scripts/` unless it also lands a routine, a kernel, or a library fix.*

**Shell is shell by content, not by name** — ruled 2026-08-31, and the term is now
`gate-docs.sh`'s `shell_files`: the `'*.sh'` glob unioned with every tracked-or-untracked file
whose shebang names a shell. It was the glob alone until then, which *"counts a 189-line test
while its 97-line subject is invisible… measuring by file extension, which is rule 22's
surface-form error turned on the ledger's own definition."* The whole tree was swept in the same
pass so the restatement happens once: `scripts/fakessh` is the only extensionless shell file in
it, and the other three non-`.sh` files under `scripts/` are TSV data. The correction is booked
as its own row below, at **+97**, and it is a definition correction rather than a spend — no line
was written for it. Controls, because an enumeration that finds one thing must be shown capable of
finding a second: a planted `#!/bin/bash -eu` is counted, a `#!/usr/bin/env python3` is not, a
file with no shebang is not, and the new list loses no member the glob held.

## How to read the ratio, and how not to

**The absolute shell count is the session-delta reading; the ratio is the headline disclosure
only.** Ratified 2026-08-31 as a standing rule, and it exists because both terms are five
digits: `+78` lines across three commits moved the historical ratio 1.77x → 1.78x, and the
next `+137` moved it 1.78x → 1.79x. **Both spends read as the same 0.01**, though one is 1.76×
the other — and the `+25` after them read as **0.00**, leaving 1.79x untouched. The library term
is measured constant at 8964 across every one of those revisions, so nothing in the denominator is
doing this. Two printed decimals cannot separate two spends that differ by 59 lines, and cannot see
a third at all. A number that advances by one hundredth per session cannot resolve a
session, and reading a *flat* ratio as a flat ledger is the specific error the rule forbids. Ask the shell term what a session cost. Ask the ratio only whether the project is
still growing apparatus faster than substance.

The same insensitivity has already been exploited once in the other direction: a commit spent
494 apparatus lines against zero library lines while the ratio it was judged by improved by
0.07, because its denominator moved too. When a ratio and its numerator disagree about the
sign of a session, the numerator is the one that cannot be flattered.

## Running total

Every delta below is **measured from the commit**, not quoted from a comment —
`git show --numstat --format= <rev> -- '*.sh'`, added minus removed. Re-derivable at any time,
which is the property a thread does not have. Measured 2026-08-31 at `48f9ed9`.

A commit that changes the shell term cannot name its own hash, so **its row is filled in by the
next commit that touches this file** — which is why the last row may briefly read `PENDING`. That
follow-up costs no shell lines, so it needs no row of its own; if it ever does, the two are one
entry. Amending the hash in would be the alternative and it is not one: the amend changes the
hash, so the row would be wrong again and wrong in a way that reads as right.

| Date | Commit | Δ shell | Total | What it was |
|---|---|---:|---:|---|
| 2026-08-30 | *(opening balance)* | +11 | 11 | The stated overage #131 was opened to discharge. Not a commit delta; the balance carried in. |
| 2026-08-30 | `4b27e92` | +21 | 32 | A lower bound rounded to nearest is not a bound. |
| 2026-08-30 | `75aae82` | +292 | 324 | Rule 17 binds the ratio criterion too, and the two lists. The single largest entry in the ledger. |
| 2026-08-30 | `891db4d` | +0 | 324 | The exclusion is per criterion. Listed at zero deliberately: the reconstruction from #131's thread had folded this into the `75aae82` figure, and measurement separates them. |
| 2026-08-30 | `a30db79` | +43 | 367 | The share criterion's rule-19 hatch and its guard. |
| 2026-08-31 | `667a06b` | +8 | 375 | `STRSM_FLOOR` becomes its formula's output. |
| 2026-08-31 | `25e8a86` | **−68** | 307 | Two A/B drivers were one harness. The first real paydown: a lift, not a deletion of comments. |
| 2026-08-31 | `ba6f286` | **−12** | 295 | Three lifts pay for the lines the denominator row cost. |
| 2026-08-31 | `8970a70` | +7 | 302 | The peak that leaked across hosts, hoisted to host scope. |
| 2026-08-31 | `5f345ee` | +53 | 355 | #100 arm B: the README denominator column is checked, not just non-empty. |
| 2026-08-31 | `2b09f6d` | +18 | 373 | #122: two words for two facts in `detach.sh stat`. |
| 2026-08-31 | `48f9ed9` | +137 | 510 | `detach-test.sh`. Authorised at ~90 by ruling; 137 disclosed, 23 of them the header stating the scope. |
| 2026-08-31 | `66706c2` | +1 | 511 | This file, plus the reading rule printed beside the ratio in `gate-docs.sh`. The ledger costs the ledger one line. |
| 2026-08-31 | `5e4557e` | +24 | 535 | `detach-test.sh` moves onto a private tmux server, with the isolation asserted and driven red. Perturbing the shared server is how the incident it tests was born. |
| 2026-08-31 | `PENDING` | **+97** | **632** | **Definition correction, not a spend.** Shell counted by content: `scripts/fakessh`, 97 lines of bash the `'*.sh'` glob never saw. The lines are as old as the file; only the counter changed. Its own `+16` of new shell in `gate-docs.sh` is the next row. |
| 2026-08-31 | `PENDING` | +16 | **648** | `shell_files` and the comment recording why the glob was wrong. Same commit as the row above, listed separately because one is a re-reading of lines that already existed and the other is lines that did not. |

**Current debt: +648 net shell lines** unpaid by a routine, a kernel or a library fix, of which
**+97 is a re-reading rather than a spend** — the tree did not grow, the counter stopped measuring
by file extension. Shell term 16195 (16082 under the old definition + 97 + 16), library term 8964,
historical ratio 1.81x.

The two rows are separate on purpose. Reported as one `+113` the entry would assert that a session
wrote 113 lines of apparatus, which is false by 97; folded into the definition note and left out of
the total it would be the second ledger the ruling above forbids. What a reader needs is the total
the cap is measured against **and** which part of it any session could have avoided writing.

Before this correction the historical series read: 15842 → 1.7673, 15920 → 1.7760, 16057 → 1.7913,
16082 → 1.7940 — three spends in one session of +78, +137 and +25, the first two printing as the
same 0.01 despite differing by 59 lines and the third printing as nothing at all. That is the
evidence for the reading rule above, and it is preserved here at the old definition because a
series measured under two definitions is not a series.

## What is owed against it, and what is not

Two entries above are authorised debt rather than overage: `48f9ed9` was ruled *"spend the ~90
lines… booked as debt"* on the grounds that the budget rule *"was never meant to starve the one
piece of apparatus whose failures are denominated in dollars rather than lines,"* and the
opening +11 predates the cap's current definition. Authorised is still owed; a ruling changes
who agreed to the debt, not whether it exists.

Two things are **not** paydown, stated so they cannot be attempted:

- **Deleting a comment that records a measured cost.** The comment is the record; removing it
  buys a line by destroying the reason the line was spent.
- **Manufacturing a library fix to balance the ledger.** The cap asks for substance shipped, and
  a fix written to move a counter is apparatus wearing the other term's name.

The honest forms are a **lift** (two implementations become one, as in `25e8a86` and `ba6f286`)
and shipping a routine or kernel that the apparatus already exists to measure.
