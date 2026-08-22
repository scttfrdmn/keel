#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# baseline-test.sh — fixtures for the BASELINE-REGISTERED class's three readers
# (#6, ruled 2026-08-21). scripts/host-baselines.tsv carries the design.
#
# WHY THIS FILE IS NOT OPTIONAL. The registry ships with zero data rows, so on a
# healthy run `registered` is unreachable, `owing` fires on one host and `new` on
# none. A green gate-p5 therefore says almost nothing about the branch that will
# decide skx's bar the day its row lands — and a readable constant certifies
# nothing: the way to trust a reading is to make the quantity move. Every case
# below moves it on purpose, in a temporary tree, against the shipped functions.
#
# The three states are decided from two facts and nothing else — a registry row,
# and an archived judged log that is not this run's. So the lattice is that
# product plus the ways each fact can be malformed.
set -u
# A cd that failed would run case 19 against whatever tree the caller happened to be
# in, and case 19 is the one asserting the tracked registry was not written.
cd "$(dirname "$0")/.." || exit 1
. scripts/remote.sh
. scripts/bench.sh
. scripts/gate-lib.sh

OK=0; BAD=0
ok()  { printf '  ok %s\n' "$1"; OK=$((OK + 1)); }
no()  { printf '  NOT OK %s\n' "$1"; BAD=$((BAD + 1)); }
is()  { if [[ "$2" == "$3" ]]; then ok "$1"; else no "$1 (got '$2', want '$3')"; fi; }

T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
CPU='Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz'
REG="$T/reg.tsv"
{
  printf '# a comment line, which is not a row\n'
  printf 'cpu_model\tcriterion\tbaseline\testimator\tsource\tas_of\tgrounds\n'
  printf '%s\tshare/Sgemm\t39.4\tmedian of 3 pinned runs (a,b,c)\tbuild/x.txt\t2026-08-21\tadmitted after the derivation set\n' "$CPU"
  printf '%s\tshare/Ssyrk\t39.7\tmedian of 3 pinned runs (a,b,c)\tbuild/x.txt\t2026-08-21\tadmitted after the derivation set\n' "$CPU"
  printf 'AMD EPYC 9R14\tshare/Sgemm\t61.0\tsingle run\tbuild/y.txt\t2026-08-21\tdisagrees with CEIL_DERIVED_FROM on purpose\n'
  printf '%s\tshare/Ssymm\t38.9\tshort row missing its grounds\n' "$CPU"
} >"$REG"

echo "-- baseline_lookup: a row, the wrong row, and every way there is no row --"
# 1. The registered case, which a healthy gate-p5 run cannot reach at all.
is 'a registered host x criterion returns its row' \
   "$(baseline_lookup "$REG" "$CPU" share/Sgemm | cut -f3)" '39.4'
# 2. Keyed on BOTH columns: one host's registration must not answer for another's
#    criterion, or a bar derived for Sgemm would silently govern Strsm.
is 'the same host, an unregistered criterion, is not registered' \
   "$(baseline_lookup "$REG" "$CPU" share/Strsm)" ''
is 'a different host, a registered criterion, is not registered' \
   "$(baseline_lookup "$REG" 'AMD EPYC 9R45' share/Sgemm)" ''
# 3. The header is data-shaped and must never answer. It has seven fields and would
#    otherwise register a host called "cpu_model" — harmless, until someone greps the
#    candidate file for a row count and gets one more than was landed.
is 'the header row is not a registration' \
   "$(baseline_lookup "$REG" cpu_model criterion)" ''
is 'a comment line is not a registration' \
   "$(baseline_lookup "$REG" '# a comment line, which is not a row' share/Sgemm)" ''
# 4. A short row is NOT a registration. This is the fail-closed direction on purpose:
#    a row missing its grounds is missing its provenance, and provenance is the row.
#    Registering it would put a bar in force with no recorded reason for existing.
is 'a row short of its provenance columns does not register' \
   "$(baseline_lookup "$REG" "$CPU" share/Ssymm)" ''
# 5. An absent registry and an absent row are one answer by design (see the function).
is 'a missing registry file registers nobody' \
   "$(baseline_lookup "$T/nope.tsv" "$CPU" share/Sgemm)" ''
is 'an empty registry registers nobody' \
   "$(: >"$T/empty.tsv"; baseline_lookup "$T/empty.tsv" "$CPU" share/Sgemm)" ''
# 6. Exact key, not substring: gate-p5 matches README rows by substring because those
#    are abbreviated by hand, but the registry key is the probe string verbatim. A
#    prefix answering for the whole would let one CPU generation inherit another's bar.
is 'a prefix of a registered model does not inherit its bar' \
   "$(baseline_lookup "$REG" 'Intel(R) Xeon(R) Platinum 8124M' share/Sgemm)" ''

echo "-- baseline_prior: newness against an unmet obligation --"
mkdir -p "$T/arch"
# 7. Nothing archived at all: the genuinely-new case, the only one BASELINE is for.
baseline_prior "$T/arch" keel-skx deadbee && no 'an empty archive claims a prior' || ok 'an empty archive is newness'
# 8. THE SELF-CITATION GUARD. The run currently writing its own archive must not read
#    it as precedent — that would make BASELINE unreachable for every host at once,
#    turning the class into a permanent FAIL the first time anyone ran it.
: >"$T/arch/bench-gate-p5-deadbee-keel-skx-3.txt"
baseline_prior "$T/arch" keel-skx deadbee && no 'this run cites itself as its own prior' || ok 'this run is not its own prior'
# 9. A different revision IS a prior. This is the line that spends the exemption.
: >"$T/arch/bench-gate-p5-5ec5fea-keel-skx-3.txt"
baseline_prior "$T/arch" keel-skx deadbee && ok 'an earlier revision is a prior' || no 'an earlier revision was missed'
# 10. Per host, not per fleet: one host's history must not spend another's exemption.
baseline_prior "$T/arch" keel-icx deadbee && no "another host's archive counted as this one's" || ok 'the witness is per host'
# 11. A revision that merely SHARES A PREFIX with this one is a different revision. The
#     exclusion anchors on the trailing '-', so 'deadbee' must not swallow 'deadbeef'.
: >"$T/arch/bench-gate-p5-deadbeef-keel-gnr-1.txt"
baseline_prior "$T/arch" keel-gnr deadbee && ok 'a longer rev sharing this prefix is a prior' || no 'a prefix collision hid a prior'
# 12. A host name that is a prefix of another must not match it, for the same reason.
: >"$T/arch/bench-gate-p5-5ec5fea-keel-zen5-1.txt"
baseline_prior "$T/arch" keel-zen deadbee && no 'keel-zen matched keel-zen5' || ok 'a host-name prefix does not match'
# 13. A non-p5 archive is not a p5 judged log. gate-p3 archives samples into the same
#     directory under its own name, and a P3 sweep judges nothing this class is about.
: >"$T/arch/bench-gate-p3-5ec5fea-keel-spr-1.txt"
baseline_prior "$T/arch" keel-spr deadbee && no "gate-p3's archive counted as a judged p5 log" || ok "another gate's archive is not a prior"
# 14. An absent archive directory is newness, not an error: a first run on a fresh
#     operator machine has no build/ yet, and crashing there would block the class.
baseline_prior "$T/gone" keel-skx deadbee && no 'a missing archive dir claimed a prior' || ok 'a missing archive dir is newness'

echo "-- baseline_candidate: what the gate proposes, and what it must never write --"
CAND="$T/out/cand.tsv"
baseline_candidate "$CAND" "$CPU" share/Sgemm 39.4 'SINGLE DRAW' build/z.txt 2026-08-21 'grounds'
baseline_candidate "$CAND" "$CPU" share/Ssyrk 39.7 'SINGLE DRAW' build/z.txt 2026-08-21 'grounds'
# 15. Created with its parent directory, since build/ may not exist on a first run.
is 'the candidate file is created under a directory that did not exist' \
   "$([[ -s "$CAND" ]] && echo yes)" 'yes'
# 16. Appends rather than truncates: a fleet proposes one row per host x criterion, and
#     a candidate file holding only the last of them would send a reviewer to re-derive
#     the rest — which is precisely the re-derivation a fully formed row exists to avoid.
is 'two proposals both survive' "$(grep -c "^$CPU" "$CAND")" '2'
# 17. Exactly seven tab-separated fields, so a landed row is a paste and not a repair.
is "a candidate row has the registry's seven columns" \
   "$(awk -F'\t' '!/^#/ {print NF; exit}' "$CAND")" '7'
# 18. And it round-trips through the reader that will judge it, which is the only
#     definition of "fully formed" that matters: paste it in, and the bar exists.
is 'a proposed row reads back as a registration' \
   "$(baseline_lookup "$CAND" "$CPU" share/Ssyrk | cut -f3)" '39.7'
# 19. THE BOUNDARY, checked and not merely asserted: nothing above touched the tracked
#     registry. An instrument that mints the reference it will judge against has
#     certified itself, so this is the one property whose violation is not a wrong
#     number but a wrong constitution.
is 'the tracked registry has no data rows and was not written' \
   "$(awk -F'\t' '!/^#/ && $1 != "cpu_model" && NF >= 7' scripts/host-baselines.tsv | wc -l | tr -d ' ')" '0'

printf '\n%d ok, %d not ok\n' "$OK" "$BAD"
[[ "$BAD" -eq 0 ]]
