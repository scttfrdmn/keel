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
which is the property a thread does not have. Rows through `5e4557e` were measured 2026-08-31 at
`48f9ed9`; from `3087301` the pathspec gains every file `shell_files` finds by shebang, which today
is `scripts/fakessh` — so a future edit to it is a row like any other.

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
| 2026-08-31 | `3087301` | **+97** | **632** | **Definition correction, not a spend.** Shell counted by content: `scripts/fakessh`, 97 lines of bash the `'*.sh'` glob never saw. The lines are as old as the file; only the counter changed. Its own `+16` of new shell in `gate-docs.sh` is the next row. |
| 2026-08-31 | `3087301` | +16 | **648** | `shell_files` and the comment recording why the glob was wrong. Same commit as the row above, listed separately because one is a re-reading of lines that already existed and the other is lines that did not. |
| 2026-08-31 | `a65d39e` | +102 | **750** | **#146, ruled**: the sentinel rule (`.keel-sentinel` retired as a selection input, `sentinel_hosts`/`sentinel_declaration` lifted into `remote.sh` and a second copy in `exercise-dead-host.sh` deleted) plus the `.cmd` input-closure enumeration and its fail-first arm. Authorised debt: *"Land the sentinel rule and the closure enumeration."* Of the +102, measured per file: `remote.sh` +54 (the resolver, the declaration printer and the ruling recorded at the site), `detach.sh` +27, `detach-test.sh` +26 (the fail-first arm), `gate-p3.sh` **−9** and `exercise-dead-host.sh` +4 — the lift removed 29 lines and cost 13 back. |
| 2026-08-31 | `afb108e` | +4 | **754** | `detach.sh`: the file-channel copy is labelled AS OF LAUNCH, because a driver can rewrite a decision file inside the run it launched (`aws-fleet.sh up` writes `.keel-hosts`). Four lines of comment, caught before the fleet pass read the record and booked in the same session that spent them — the row exists because the +102 above was measured at `a65d39e` and the ratio reading taken there could not see a commit that came after it. |
| 2026-09-01 | `774c02a` | +36 | **790** | **The three-caller merge, and it did not come out neutral.** `ab-bench.sh` +151 against `l1-bench.sh` −43 and `edge-bench.sh` −75 is +33, plus +3 in `ab.sh` for the collision fix's comment. Ruled in advance: *"one parametrized caller replacing both is net-zero-or-better… if the parametrization turns out not to be net-neutral, run it anyway with the debt stated."* Stated. A counterfactual reading exists — a third thin caller would have cost ~163 against this file's 151 — and it is **not** booked, because the ledger measures the tree and the tree moved +36. What was refused as paydown: thinning the two deleted headers, which carry T19's instruction counts, the four cache-resident sizes and the three between-binary layout floors. |
| 2026-09-01 | `774c02a` | +5 | **795** | **EXEMPT, not owed** — see the section below. `gate-p3.sh` prints both terms of the sentinel's percent-of-peak on the line above the verdict that divides them. Same commit as the row above, listed separately because one is owed and one is not. |
| 2026-09-01 | `9862637` | +71 | **866** | **The planted-delta control, authorised as debt and over its estimate.** Scott: *"authorised as debt, and it was never optional… ~30 lines, its own ledger row, landed with the control shown to catch the plant."* Authorised at ~30; **+71 disclosed**, measured per hunk: `ab_control` and its doc block +47, `ab_control_samples` +8, `ab_arm_file` +6, the call site in `ab_run` +4, and +9/−5 converting `ab_host` to the shared namer. Half of it is the doc block, and it is what makes the row honest rather than shorter: the control exists because #141's checklist asked for it *before* the harness was written and it was skipped, and the block names that, the exact-×1.1 construction, and what the control cannot see (§5 rule 12). The `ab_arm_file` lift is the load-bearing part — the control drives the run's own two SHAs through the same namer `ab_host` uses, so the collision it was written for is covered by construction rather than by a second assertion. |
| 2026-09-01 | `9862637` | +58 | **924** | **The build-flags line, ruled as an instrument change.** Scott: *"the declaration row grows a **build-flags line** (the deeper fix: flags are part of the binary's provenance and belong in every log's self-description)."* `remote.sh` +32 (`build_settings`, plus `builder_toolchain` restated to carry the flags and the measured `set -e` fact behind its two-guard form) and `ab.sh` +26 (`ab_arm_provenance` and its two call sites). **Not exempt**, and the exemption was checked rather than assumed: it covers a criterion's own terms *on the signing path*, and `ab.sh`'s own header says its callers certify nothing. Booked in full. |
| 2026-09-01 | `PENDING` | +9 | **933** | **`-trimpath` in `remote_build_test`, ruled as an instrument change.** Scott: *"`-trimpath`: yes, now… the null-A/B-builds-two-binaries class dies at the root or it recurs forever."* `remote.sh` +10/−1, of which **one line is the flag** and nine are the comment that says why a build flag is a measurement decision — the measured 4-digests-from-4-paths fact, the `1.71/0.99/1.32%` layout floor it sits beside, and that the flag is readable back out of the artifact. Deliberately not shortened: this is the rule's own example of a fix whose honest form is a comment, and the cap permits that at the cost of budget rather than by exemption. Booked in full; **not** exempt, on the same reading as the row above. |

**Current debt: +933 net shell lines** unpaid by a routine, a kernel or a library fix, of which
**+97 is a re-reading rather than a spend** — the tree did not grow, the counter stopped measuring
by file extension — and **+5 is exempt**, so **+831 is owed** once both are set aside. Shell term
16480, library term 8964, historical ratio 1.84x (measured 2026-09-01 with the `-trimpath` row above
staged). Both readings of `9862637`'s `+129` agree — `git show --numstat --format= 9862637 -- '*.sh'`
gives 102 + 36 added against 5 + 4 removed, and `gate-docs.sh`'s own counter moved 16342 → 16471 —
which is the cross-check that makes the figure a measurement rather than a subtraction. The
`-trimpath` commit adds `+9` on the same two readings (`10 − 1` in `remote.sh`; the counter 16471 →
16480), for a **session total of +138** across two commits.

**And the ratio moved this time, 1.82x → 1.84x, which does not retire the reading rule above; it
illustrates the second half of it, and the arithmetic is the opposite of what the printed jump
suggests.** The library term is measured constant at 8964 across both readings, and the *unrounded*
increment this session is **smaller** than the one that printed as a single hundredth: +129 lines
moved 1.82307 → 1.83746, a delta of **0.01439**, while +137 lines one session earlier moved 1.77599
→ 1.79128, a delta of **0.01528**. The larger spend printed less. What decided the visible jump was
where each pair fell against a rounding boundary, not how much was spent — so a printed increment
cannot be read backwards into a line count in either direction, and nothing here asks it to: the
+129 came from the diff.

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

Three entries above are authorised debt rather than overage: `48f9ed9` was ruled *"spend the ~90
lines… booked as debt"* on the grounds that the budget rule *"was never meant to starve the one
piece of apparatus whose failures are denominated in dollars rather than lines,"* and the
opening +11 predates the cap's current definition. Authorised is still owed; a ruling changes
who agreed to the debt, not whether it exists. The +102 for #146 is the third: Scott ruled option (b) with the file channel retired *"plus (c) which is owed regardless"*, and a fix that removes a gitignored file's power to select a judged host is apparatus by every definition here — it ships no routine and no kernel. It is booked at its measured size, including the 26 lines of test that make the enumeration fail first. The `afb108e` row after it is the same authorisation reaching four more lines of comment, so the authorised total for #146 is **+106**, not +102 — which is the whole reason that row exists rather than being folded backwards into a figure already published.

The fourth, fifth and sixth are this session's, all three authorised in the #141 ruling of
2026-09-01. The control was authorised at *"~30 lines"* and cost **+71**; the build-flags line was
authorised without a figure and cost **+58**; `-trimpath` was authorised without a figure
(*"yes, now"*) and cost **+9**. None is exempt and none is paid for: this session shipped no routine,
no kernel and no library fix, so **+138 of the total above is authorised overage on the cap's own
terms** and is stated that way rather than argued down. The estimate is not the authorisation's
operative part — *"it was never optional"* is — but the gap between 30 and 71 belongs in the record,
because the next estimate is calibrated from this one. The `-trimpath` row is the counter-example
worth keeping beside it: one line of flag, nine of comment, and the ruling's *"prefer deleting a line
to explaining one"* loses here to the cap's own carve-out that a fix whose honest form is a comment is
still allowed and simply spends budget.

**The disclosure exemption** (standing since CEIL8CI, invoked 2026-09-01 for the `+5` above). A line
whose job is to print the terms a criterion on the signing path divides does not wait for a session
that ships a routine. The grounds are that its absence has a measured cost and the cap does not: the
sentinel published a ratio with neither term in any log, and #141 had to reconstruct both from
archived samples to check a published verdict. Reconstruction is what happens when disclosure fails.
The exemption is narrow on purpose — a criterion's *own* terms, on the signing path, one line — and
it is not a licence for gate apparatus generally, which is what most of the total above is.

Two things are **not** paydown, stated so they cannot be attempted:

- **Deleting a comment that records a measured cost.** The comment is the record; removing it
  buys a line by destroying the reason the line was spent.
- **Manufacturing a library fix to balance the ledger.** The cap asks for substance shipped, and
  a fix written to move a counter is apparatus wearing the other term's name.

The honest forms are a **lift** (two implementations become one, as in `25e8a86` and `ba6f286`)
and shipping a routine or kernel that the apparatus already exists to measure.
