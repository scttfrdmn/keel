#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# baseline-test.sh — fixtures for the BASELINE-REGISTERED class's readers (#6, ruled
# 2026-08-21). scripts/host-baselines.tsv, judged-runs.tsv and measurement-eras.tsv
# carry the design; this file is where every branch of them is driven.
#
# WHY THIS FILE IS NOT OPTIONAL. The registry ships with zero data rows, so on a
# healthy run `registered` is unreachable, `owing` fires on one host and `new` on
# none. A green gate-p5 therefore says almost nothing about the branch that will
# decide skx's bar the day its row lands — and a readable constant certifies
# nothing: the way to trust a reading is to make the quantity move. Every case
# below moves it on purpose, in a temporary tree, against the shipped functions.
#
# The three states are decided from two TRACKED facts and nothing else — a registry
# row, and a witness row saying this silicon was judged in this era before. So the
# lattice is that product, times the era each fact is scoped to, plus the ways each
# can be malformed. The era readers get their own group because the loophole guard
# lives in them: an era with no cited amendment must yield NOTHING rather than
# falling back to whichever era last parsed.
set -u
# A cd that failed would run the boundary cases against whatever tree the caller
# happened to be in, and those are the ones asserting the tracked files were not written.
cd "$(dirname "$0")/.." || exit 1
. scripts/remote.sh
. scripts/bench.sh
. scripts/gate-lib.sh

OK=0; BAD=0
ok()  { printf '  ok %s\n' "$1"; OK=$((OK + 1)); }
no()  { printf '  NOT OK %s\n' "$1"; BAD=$((BAD + 1)); }
is()  { if [[ "$2" == "$3" ]]; then ok "$1"; else no "$1 (got '$2', want '$3')"; fi; }

# Snapshotted HERE, before the first fixture, for case 30: the property is that nothing
# in this file touches the tracked artifacts, so the reference has to predate every case.
# `cksum` over the bytes rather than the text: a whole-file comparison makes the failure
# message a two-screen diff of the registry, and command substitution would strip the
# trailing newlines a write could consist entirely of. Only compared against itself, on
# one machine, in one run, so which CRC it is does not matter.
REG_BEFORE="$(cksum <scripts/host-baselines.tsv)"
WIT_BEFORE="$(cksum <scripts/judged-runs.tsv)"

T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
CPU='Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz'
ERA=pinned8
OLD=free-placement
REG="$T/reg.tsv"
{
  printf '# a comment line, which is not a row\n'
  printf 'cpu_model\tcriterion\tera\tbaseline\testimator\tsource\tas_of\tgrounds\n'
  printf '%s\tshare/Sgemm\t%s\t39.4\tmedian of 3 pinned runs (a,b,c)\tbuild/x.txt\t2026-08-21\tadmitted after the derivation set\n' "$CPU" "$ERA"
  printf '%s\tshare/Ssyrk\t%s\t39.7\tmedian of 3 pinned runs (a,b,c)\tbuild/x.txt\t2026-08-21\tadmitted after the derivation set\n' "$CPU" "$ERA"
  printf '%s\tshare/Strsm\t%s\t7.1\tmedian of 3 free runs (d,e,f)\tbuild/w.txt\t2026-08-01\ta row from the era before this one\n' "$CPU" "$OLD"
  printf 'AMD EPYC 9R14\tshare/Sgemm\t%s\t61.0\tsingle run\tbuild/y.txt\t2026-08-21\tdisagrees with CEIL_DERIVED_FROM on purpose\n' "$ERA"
  printf '%s\tshare/Ssymm\t%s\t38.9\tshort row missing its grounds\n' "$CPU" "$ERA"
  # A full-width row whose ERA COLUMN IS EMPTY. It exists for case 11 and nothing else:
  # without it, `era=""` matches no row by accident (every other row has an era), the
  # empty-era guard is untestable, and a mutant that deletes the guard survives. Which is
  # what happened on the first pass — the guard was right and the fixture was blind.
  printf '%s\tshare/Sgemm2\t\t40.0\tan era-less row\tbuild/v.txt\t2026-08-21\tno era, therefore no scope\n' "$CPU"
  # Exactly SEVEN fields — one short of the full width, i.e. the off-by-one a reader
  # inherits when a column is added. The 5-field row above does not catch it: NF >= 7 and
  # NF >= 8 both reject 5. Mutating the width check survived until this row existed.
  printf '%s\tshare/Sgemm3\t%s\t41.0\tan estimator\tbuild/u.txt\t2026-08-21\n' "$CPU" "$ERA"
} >"$REG"

echo "-- era_current: the loophole guard, which is a reader and not a paragraph --"
LED="$T/eras.tsv"
# 1. No ledger at all is no era. gate-p5 renders this FAIL, because a reading with no
#    scope may not be judged against a baseline that has one.
is 'a missing ledger resolves no era' "$(era_current "$T/nope.tsv")" ''
is 'an empty ledger resolves no era' "$(: >"$LED"; era_current "$LED")" ''
# 2. Comments and the header are data-shaped and must not answer, or the current era
#    would be named "era" and every registry row would fall out of scope at once.
{
  printf '# a comment\n'
  printf 'era\topened\tamendment\ttransition_archive\tgrounds\n'
} >"$LED"
is 'a ledger of only comments and a header resolves no era' "$(era_current "$LED")" ''
# 3. The last data row wins, which is what makes opening an era a one-line append.
printf '%s\tretroactive\tDESIGN.md §5 rule 17\tn/a\tthe era before eras\n' "$OLD" >>"$LED"
is 'the sole era is the current one' "$(era_current "$LED")" "$OLD"
printf '%s\t2026-08-21\tDESIGN.md §5 rule 5 + §5 rule 17\t—\tplacement pinned\n' "$ERA" >>"$LED"
is 'the last data row is the current era' "$(era_current "$LED")" "$ERA"
# 4. THE GUARD. A final row citing no amendment is not an era, and resolution yields
#    NOTHING — it does not skip back to the valid row above it. Falling back is the
#    whole failure mode: it would judge new-instrument readings against old-instrument
#    baselines, i.e. book the methodology delta as host drift, silently, on every host.
cp "$LED" "$T/led-bad.tsv"
printf 'convenient\t2026-08-22\t\t—\topened because a run went red\n' >>"$T/led-bad.tsv"
is 'an era citing no amendment resolves nothing, and does not fall back' \
   "$(era_current "$T/led-bad.tsv")" ''
cp "$LED" "$T/led-dash.tsv"
printf 'convenient\t2026-08-22\t—\t—\tan em-dash is not a citation\n' >>"$T/led-dash.tsv"
is 'an em-dash in the amendment column is not an amendment' \
   "$(era_current "$T/led-dash.tsv")" ''
cp "$LED" "$T/led-short.tsv"
printf 'convenient\t2026-08-22\tDESIGN.md §5 rule 17\n' >>"$T/led-short.tsv"
is 'a short era row resolves nothing' "$(era_current "$T/led-short.tsv")" ''
# One column short of the full width, which the 3-field row above cannot catch: an era row
# with no grounds is an era with no stated instrument change, and the amendment column
# alone is a citation without a claim. Same off-by-one the registry's 7-field row covers.
cp "$LED" "$T/led-nogrounds.tsv"
printf 'convenient\t2026-08-22\tDESIGN.md §5 rule 17\t—\n' >>"$T/led-nogrounds.tsv"
is 'an era row one column short of the full width resolves nothing' \
   "$(era_current "$T/led-nogrounds.tsv")" ''
# An unnamed era. This asserts the OUTCOME — nothing resolves — and not a branch: the
# reader has no name check, because an empty name prints an empty line and every caller
# reads that as no era. Kept because the outcome is what gate-p5 depends on.
cp "$LED" "$T/led-unnamed.tsv"
printf '\t2026-08-22\tDESIGN.md §5 rule 17\t—\tan era with no name\n' >>"$T/led-unnamed.tsv"
is 'an unnamed era resolves nothing' "$(era_current "$T/led-unnamed.tsv")" ''
# 5. A trailing blank line is formatting, not a malformed era. Without this the guard
#    would fire on every ledger a text editor had saved.
is 'a trailing blank line does not unresolve the era' \
   "$(printf '\n' >>"$LED"; era_current "$LED")" "$ERA"

echo "-- era_provisional: a disclosure about the archive, never a permission --"
# 6. `—` and empty both mean the both-arms transition archive has not landed. The empty
#    case gets its own ledger because it is a separate branch of the reader, and testing
#    only the em-dash let a mutant that dropped the empty check survive: `—` is a
#    convention and a blank cell is what a hand-edited tsv actually produces.
era_provisional "$LED" "$ERA" && ok 'an em-dash archive column reads provisional' || no 'provisional was missed'
era_provisional "$LED" "$OLD" && no 'a landed archive read provisional' || ok 'a landed archive is not provisional'
cp "$LED" "$T/led-blank.tsv"
printf 'blankarch\t2026-08-22\tDESIGN.md §5 rule 17\t\tan empty cell, not an em-dash\n' >>"$T/led-blank.tsv"
era_provisional "$T/led-blank.tsv" blankarch && ok 'an empty archive column reads provisional too' || no 'an empty archive column read as landed'
# 7. An era the ledger does not carry is not reported provisional. That is not a claim
#    it is closed: era_current is what decides whether an era exists, and gate-p5 calls
#    these two in that order. Recorded because the pair reads like one question.
era_provisional "$LED" nosucherathing && no 'an unknown era read provisional' || ok 'an unknown era is not reported provisional (era_current decides existence)'

echo "-- baseline_lookup: a row, the wrong row, the wrong ERA, and every way there is none --"
# 8. The registered case, which a healthy gate-p5 run cannot reach at all.
is 'a registered host x criterion x era returns its row' \
   "$(baseline_lookup "$REG" "$CPU" share/Sgemm "$ERA" | cut -f4)" '39.4'
# 9. Keyed on all three columns: one host's registration must not answer for another's
#    criterion, or a bar derived for Sgemm would silently govern Strsm.
is 'the same host, an unregistered criterion, is not registered' \
   "$(baseline_lookup "$REG" "$CPU" share/Ssymm2 "$ERA")" ''
is 'a different host, a registered criterion, is not registered' \
   "$(baseline_lookup "$REG" 'AMD EPYC 9R45' share/Sgemm "$ERA")" ''
# 10. THE ERA SCOPING, which is the whole of the 2026-08-21 amendment. This host HAS a
#     Strsm baseline — measured under free placement. Under the pinned instrument it must
#     not be found, because applying it would compare a pinned reading against a free
#     reference and book the difference as this host's drift.
is "a baseline from another era does not govern this era's reading" \
   "$(baseline_lookup "$REG" "$CPU" share/Strsm "$ERA")" ''
is 'and it is still findable in its own era, so the row was not merely unreadable' \
   "$(baseline_lookup "$REG" "$CPU" share/Strsm "$OLD" | cut -f4)" '7.1'
# 11. An unresolved era registers nobody. Fail-closed, matching gate-p5's own FAIL: with
#     no era there is no scope, and a bar applied out of scope is worse than no bar. The
#     second assertion is the one with teeth: an era-less registry row is exactly what an
#     unresolved era would otherwise MATCH, so the guard is what stops "" from being a
#     scope of its own. Verified by mutation — deleting the guard survived until this row
#     existed (§5 rule 12: an unkillable mutant is stated inside the number).
is 'an empty era argument registers nobody' \
   "$(baseline_lookup "$REG" "$CPU" share/Sgemm '')" ''
is 'an empty era does not match an era-less registry row either' \
   "$(baseline_lookup "$REG" "$CPU" share/Sgemm2 '')" ''
# 12. The header is data-shaped and must never answer. It has eight fields and would
#     otherwise register a host called "cpu_model" — harmless, until someone greps the
#     candidate file for a row count and gets one more than was landed.
is 'the header row is not a registration' \
   "$(baseline_lookup "$REG" cpu_model criterion era)" ''
is 'a comment line is not a registration' \
   "$(baseline_lookup "$REG" '# a comment line, which is not a row' share/Sgemm "$ERA")" ''
# 13. A short row is NOT a registration. This is the fail-closed direction on purpose:
#     a row missing its grounds is missing its provenance, and provenance is the row.
#     Registering it would put a bar in force with no recorded reason for existing.
is 'a row short of its provenance columns does not register' \
   "$(baseline_lookup "$REG" "$CPU" share/Ssymm "$ERA")" ''
is 'a row one column short of the full width does not register either' \
   "$(baseline_lookup "$REG" "$CPU" share/Sgemm3 "$ERA")" ''
# 14. An absent registry and an absent row are one answer by design (see the function).
is 'a missing registry file registers nobody' \
   "$(baseline_lookup "$T/nope.tsv" "$CPU" share/Sgemm "$ERA")" ''
is 'an empty registry registers nobody' \
   "$(: >"$T/empty.tsv"; baseline_lookup "$T/empty.tsv" "$CPU" share/Sgemm "$ERA")" ''
# 15. Exact key, not substring: gate-p5 matches README rows by substring because those
#     are abbreviated by hand, but the registry key is the probe string verbatim. A
#     prefix answering for the whole would let one CPU generation inherit another's bar.
is 'a prefix of a registered model does not inherit its bar' \
   "$(baseline_lookup "$REG" 'Intel(R) Xeon(R) Platinum 8124M' share/Sgemm "$ERA")" ''

echo "-- baseline_spent: newness against an unmet obligation, from TRACKED state --"
WIT="$T/judged.tsv"
{
  printf '# a comment line, which is not a row\n'
  printf 'cpu_model\tera\trev\tas_of\thost\tarchive\n'
} >"$WIT"
# 16. Nothing witnessed at all: the genuinely-new case, the only one BASELINE is for.
baseline_spent "$WIT" "$CPU" "$ERA" && no 'an empty index claims a prior judgment' || ok 'an empty index is newness'
baseline_spent "$T/nope.tsv" "$CPU" "$ERA" && no 'a missing index claims a prior judgment' || ok 'a missing index is newness'
# 17. A landed row IS a prior judgment. This is the line that spends the exemption, and
#     unlike its predecessor it says so in tracked state rather than in gitignored
#     build/ output, which read right only on the operator's machine (#114).
printf '%s\t%s\t5ec5fea\t2026-08-21\tkeel-skx\tbuild/bench-gate-p5-5ec5fea-keel-skx-3.txt\n' "$CPU" "$ERA" >>"$WIT"
baseline_spent "$WIT" "$CPU" "$ERA" && ok 'a witness row spends BASELINE' || no 'a witness row was missed'
# 18. PER ERA. The same silicon judged in the previous era has NOT spent this era's
#     BASELINE — that is what makes an instrument change render BASELINE fleet-wide once
#     instead of convicting every host of an unmet registration it could not have met.
baseline_spent "$WIT" "$CPU" "$OLD" && no "an era's witness spent another era's BASELINE" || ok 'the witness is per era'
# 19. THE RENAME LOOPHOLE, closed by keying on the CPU model rather than the hostname —
#     the one deviation from the ruling's wording, recorded in judged-runs.tsv. Driven by
#     MOVING THE HOST COLUMN and showing the answer does not move: the row above was
#     witnessed by keel-skx, and a fleet that has renamed that machine still finds its
#     silicon spent. What this proves is that column 5 is provenance and not a key; it
#     cannot prove more than that, because the hostname is not an argument to the function
#     at all — which is the point, and is why re-asserting case 17 under a new label (the
#     first version of this case) proved nothing.
sed 's/keel-skx/keel-skx-renamed/' "$WIT" >"$T/judged-renamed.tsv"
is 'the witnessed hostname really did change in the fixture' \
   "$(awk -F'\t' '$1 != "cpu_model" && !/^#/ {print $5; exit}' "$T/judged-renamed.tsv")" 'keel-skx-renamed'
baseline_spent "$T/judged-renamed.tsv" "$CPU" "$ERA" && ok 'the same silicon under a renamed host is still spent (the key is the model)' || no 'a rename reopened the exemption'
# 20. Per silicon, not per fleet: one model's history must not spend another's exemption.
baseline_spent "$WIT" 'AMD EPYC 9R45' "$ERA" && no "another model's witness counted as this one's" || ok 'the witness is per CPU model'
# 21. Exact key here too, for the reason case 15 gives.
baseline_spent "$WIT" 'Intel(R) Xeon(R) Platinum 8124M' "$ERA" && no 'a model prefix spent a witness' || ok 'a model prefix does not spend a witness'
# 22. A short witness row does not spend. Same fail-closed direction as case 13: a row
#     missing its provenance is missing the archive the judgment is recomputable from,
#     and convicting a host on an unverifiable witness is the worse of the two errors.
printf 'AMD EPYC 9R14\t%s\tdeadbee\n' "$ERA" >>"$WIT"
baseline_spent "$WIT" 'AMD EPYC 9R14' "$ERA" && no 'a short witness row spent BASELINE' || ok 'a short witness row does not spend'
# 23. The header must not answer, or a host called "cpu_model" would arrive pre-spent.
baseline_spent "$WIT" cpu_model era && no 'the header row spent BASELINE' || ok 'the header row is not a witness'
# 24. An unreadable CPU model spends nothing. gate-p5 renders that host `unmeasured`
#     before it reaches here (no key, so no bar); this is the accessor agreeing rather
#     than guessing. An unresolved era likewise. Both are tested against an index that
#     CONTAINS the row an empty key would otherwise match — a full-width witness row with
#     an empty cpu_model, and another with an empty era. Without those two rows the empty
#     keys match nothing by luck, the guard is untestable, and deleting it survives; which
#     is what the first pass measured.
printf '\t%s\tdeadbee\t2026-08-21\tkeel-nameless\tbuild/b.txt\n' "$ERA" >>"$WIT"
printf 'AMD EPYC 9R45\t\tdeadbee\t2026-08-21\tkeel-eraless\tbuild/c.txt\n' >>"$WIT"
baseline_spent "$WIT" '' "$ERA" && no 'an empty CPU model spent BASELINE' || ok 'an empty CPU model spends nothing, even against an empty-keyed row'
baseline_spent "$WIT" 'AMD EPYC 9R45' '' && no 'an empty era spent BASELINE' || ok 'an empty era spends nothing, even against an era-less row'

echo "-- baseline_candidate: what the gate proposes, and what it must never write --"
CAND="$T/out/cand.tsv"
baseline_candidate "$CAND" "$CPU" share/Sgemm "$ERA" 39.4 'SINGLE DRAW' build/z.txt 2026-08-21 'grounds'
baseline_candidate "$CAND" "$CPU" share/Ssyrk "$ERA" 39.7 'SINGLE DRAW' build/z.txt 2026-08-21 'grounds'
# 25. Created with its parent directory, since build/ may not exist on a first run.
is 'the candidate file is created under a directory that did not exist' \
   "$([[ -s "$CAND" ]] && echo yes)" 'yes'
# 26. Appends rather than truncates: a fleet proposes one row per host x criterion, and
#     a candidate file holding only the last of them would send a reviewer to re-derive
#     the rest — which is precisely the re-derivation a fully formed row exists to avoid.
is 'two proposals both survive' "$(grep -c "^$CPU" "$CAND")" '2'
# 27. Exactly eight tab-separated fields, so a landed row is a paste and not a repair.
is "a candidate row has the registry's eight columns" \
   "$(awk -F'\t' '!/^#/ {print NF; exit}' "$CAND")" '8'
# 28. And it round-trips through the reader that will judge it, which is the only
#     definition of "fully formed" that matters: paste it in, and the bar exists.
is 'a proposed row reads back as a registration' \
   "$(baseline_lookup "$CAND" "$CPU" share/Ssyrk "$ERA" | cut -f4)" '39.7'
# 29. The SECOND row a BASELINE rendering proposes, into a second file: the witness that
#     spends it. Variadic on purpose — the two rows have different widths, and the pair
#     is what a reviewed commit lands. It round-trips through baseline_spent for the same
#     reason the baseline round-trips through baseline_lookup.
WCAND="$T/out/wit.tsv"
baseline_candidate "$WCAND" "$CPU" "$ERA" deadbee 2026-08-21 keel-skx build/a.txt
is "a candidate witness row has the index's six columns" \
   "$(awk -F'\t' '!/^#/ {print NF; exit}' "$WCAND")" '6'
baseline_spent "$WCAND" "$CPU" "$ERA" && ok 'a proposed witness row reads back as a witness' || no 'the proposed witness does not round-trip'
# 30. THE BOUNDARY, checked and not merely asserted: nothing above touched the tracked
#     files. An instrument that mints the reference it will judge against has certified
#     itself, so this is the one property whose violation is not a wrong number but a
#     wrong constitution. Both tracked artifacts, because the witness is now half of the
#     decision and a gate that could write it could spend its own exemption.
#
#     UNCHANGED, NOT EMPTY (repaired 2026-08-28). This case used to assert that both
#     tracked files hold zero data rows, using emptiness as a proxy for not-written. The
#     proxy held only while the registry was unpopulated. f0e9e0b landed three reviewed
#     baseline rows and one witness row — the registration this whole class exists to
#     serve — the control failed, and gate-p5 fail-closed on every bar, so the first
#     BASELINE-REGISTERED verdicts were never computed. A legitimate registration read
#     exactly like this script writing to the tree. The property was always *unchanged*,
#     and only a before/after comparison can state it; the same shape, for the same
#     reason, as exercise-baseline.sh's P4BEFORE.
is 'the tracked registry was not written' "$(cksum <scripts/host-baselines.tsv)" "$REG_BEFORE"
is 'the tracked witness index was not written' "$(cksum <scripts/judged-runs.tsv)" "$WIT_BEFORE"
# 31. And the shipped ledger resolves, which is the one case above that is about THIS
#     tree rather than a fixture: gate-p5 FAILs when it does not, so a malformed ledger
#     should be caught by `make lint` on every push and not by a $24/hr fleet.
is 'the shipped era ledger resolves an era' \
   "$([[ -n "$(era_current scripts/measurement-eras.tsv)" ]] && echo yes)" 'yes'

printf '\n%d ok, %d not ok\n' "$OK" "$BAD"
[[ "$BAD" -eq 0 ]]
