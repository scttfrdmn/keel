#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# citation-lint.sh — every DESIGN.md rule citation in the tree names a section and an
# item that exist. That is the entire claim; the limits below are the point of the file.
#
# PINNING WAS RETIRED 2026-08-16, and this is the one paydown item that removed a
# check, so it is recorded rather than quietly dropped. The script also used to record
# each distinct form beside the first ten words of the item it resolved to, so a
# renumber or a rewording went red. That bought drift detection for a document that has
# never been renumbered, at 663 lines across three files run on every push — while the
# failure that actually happened was 18 citations MINTED wrong (`§5.4` written for the # citation-lint:quote(5.4)
# rule appended as item 5, `4643b63`), which DESIGN.md §5 rule 9 says "every pin would
# therefore have passed forever" over. A renumber is now caught by review. §5 rule 9 is
# amended to match, because a rule mandating an instrument that no longer exists is
# worse than either the rule or the instrument alone.
#
# WHAT A GREEN DOES NOT MEAN — unchanged by the retirement, and now the only caveat
# left, so read it. Resolution is not mint-correctness. A citation that lands on a real
# item can still be the wrong item, and that check is a human or model reading the cited
# ordinal against the content the citing site actually invokes. It was run once, over
# all 112 DESIGN-bound sites on 2026-08-16; the census is in the externals file's
# header. Nothing here re-establishes it, and a sweep of that size needs reading again.
#
# TWO NOTATIONS ARE LEGITIMATE and both resolve to (section, item): the explicit
# `§5 rule 5` and the shorthand `§5.5`, which means "§5, item 5". Stated because an # citation-lint:quote(5.5)
# earlier draft condemned the shorthand *by form* — DESIGN.md has no subsections, so
# `§X.Y` looks incoherent on its face — and would have demanded 25 zero-semantic edits
# to correct citations in the document the gates cite as grounds. New citations use the
# explicit form; the audited shorthand sites stay byte-for-byte.
#
# A CITATION OF ANOTHER DOCUMENT IS NOT A DEFECT IN THIS ONE. Some sites use this
# notation for `docs/spill-report.md` or for the BLIS paper's own §4.3, and they are # citation-lint:quote(4.3)
# indistinguishable from DESIGN shorthand by form. They are DECLARED in the externals
# file, keyed by file and form and carrying the number of sites the declaration covers;
# an undeclared one is an error rather than a guess. Note that five of the nine
# declarations are load-bearing for resolution (their forms do not exist in DESIGN.md at
# all) and four cover forms that WOULD resolve — those four are declarations of meaning,
# unenforceable here, and #91 is the open issue that a self-reference resolving against
# the wrong document passes either way.
#
# A MENTION OF A FORM IS NOT A CITATION: prose showing a bad form to explain a defect
# invokes no rule's content. Mark those lines `citation-lint:quote`, scoped as
# `citation-lint:quote(5.4)` where the line also carries real citations. Normalizing # citation-lint:quote(5.4)
# them would erase the record of the defect they document. The marker must sit on the
# citation's OWN line — wrapping prose separates them and it silently stops working.
#
# Usage:  citation-lint.sh          check
#         citation-lint.sh --list   print every citation site, resolved
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
DESIGN="DESIGN.md"
EXT_FILE="docs/citation-externals.txt"
MODE="${1:-check}"

die() { printf 'citation-lint: %s\n' "$1" >&2; exit 2; }
[[ -f "$DESIGN" ]] || die "no $DESIGN to resolve against"

# Sections are `## N. Title`; inside one, a top-level ordered item is `N. ` at column 0.
# Continuation lines and nested lists are not items and must not be counted, or every
# rule number below them would shift.
items() {
  awk '
    /^## [0-9]+\./ { sec = $2; sub(/\./, "", sec); cursec = sec; next }
    /^[0-9]+\. / && cursec != "" { n = $1; sub(/\./, "", n); printf "%s\t%s\n", cursec, n }
  ' "$DESIGN"
}
ITEMS="$(items)"
[[ -n "$ITEMS" ]] || die "parsed no numbered items out of $DESIGN"

section_exists() { awk -F'\t' -v s="$1" '$1 == s { f = 1 } END { exit !f }' <<<"$ITEMS"; }
item_exists() { awk -F'\t' -v s="$1" -v r="$2" '$1 == s && $2 == r { f = 1 } END { exit !f }' <<<"$ITEMS"; }

# `FILE<TAB>FORM<TAB>COUNT<TAB>the document it actually cites`, keyed by file and form so
# an edit that moves a line does not invalidate the declaration. COUNT is what keeps the
# exemption honest.
externals() {
  [[ -f "$EXT_FILE" ]] || return 0
  awk '!/^#/ && NF' "$EXT_FILE"
}
EXTERNAL="$(externals)"
is_external() {
  [[ -n "$EXTERNAL" ]] || return 1
  awk -F'\t' -v f="$1" -v c="$2" '$1 == f && $2 == c { g = 1 } END { exit !g }' <<<"$EXTERNAL"
}

# NOTE, learned the hard way: `git ls-files` sees only TRACKED files, so a new file's
# citations are invisible until it is committed — a green here means "every citation in a
# tracked file", and `git status` is part of reading it.
sites() {
  git ls-files -z -- '*.go' '*.sh' '*.md' \
    | xargs -0 grep -nHoE '§[0-9]+(\.[0-9]+| rule [0-9]+)' 2>/dev/null \
    | grep -v "^${EXT_FILE}:" || true
}

# The scope list omits the section sign deliberately: with it, the marker's own argument
# is a citation-shaped string the scanner would count as a site and then suppress,
# inflating the quoted tally by one per marker.
quoted() { # FILE LINE CITE
  local ln
  ln="$(sed -n "${2}p" "$1" 2>/dev/null)" || return 1
  case "$ln" in
    *citation-lint:quote\(*)
      local scope="${ln#*citation-lint:quote(}"; scope="${scope%%)*}"
      [[ ",$scope," == *",${3#§},"* ]] ;;
    *citation-lint:quote*) return 0 ;;
    *) return 1 ;;
  esac
}

FAIL=0
declare -a REPORT=()
N_SITES=0 N_EXT=0 N_QUOTE=0
EXTHIT=""

while IFS= read -r hit; do
  [[ -n "$hit" ]] || continue
  file="${hit%%:*}"; rest="${hit#*:}"
  line="${rest%%:*}"; cite="${rest#*:}"

  if quoted "$file" "$line" "$cite"; then N_QUOTE=$((N_QUOTE + 1)); continue; fi
  if is_external "$file" "$cite"; then
    N_EXT=$((N_EXT + 1)); EXTHIT="$EXTHIT$file"$'\t'"$cite"$'\n'; continue
  fi
  N_SITES=$((N_SITES + 1))

  sec="${cite%%[. ]*}"; sec="${sec#§}"
  num="${cite##*[. ]}"

  if ! section_exists "$sec"; then
    REPORT+=("NO SECTION  $file:$line  '$cite' — $DESIGN has no §$sec, and this site is not declared in $EXT_FILE")
    FAIL=1
  elif ! item_exists "$sec" "$num"; then
    REPORT+=("NO ITEM     $file:$line  '$cite' — §$sec has no item $num, and this site is not declared in $EXT_FILE")
    FAIL=1
  elif [[ "$MODE" == "--list" ]]; then
    REPORT+=("ok          $file:$line  '$cite' -> §$sec item $num")
  fi
done < <(sites)

# An exemption is only trustworthy while it is as narrow as it claims. Fewer sites than
# declared means the declaration is stale; more means it has silently grown to cover a
# citation nobody read, and the fix is to read the new site and bump the count.
if [[ -n "$EXTERNAL" ]]; then
  while IFS= read -r decl; do
    [[ -n "$decl" ]] || continue
    IFS=$'\t' read -r dfile dform dwant _ <<<"$decl"
    if ! [[ "$dwant" =~ ^[0-9]+$ ]]; then
      REPORT+=("BAD EXT     $EXT_FILE declares '$dform' in $dfile without a site count in its third column")
      FAIL=1
      continue
    fi
    n="$(awk -F'\t' -v f="$dfile" -v c="$dform" '$1 == f && $2 == c { n++ } END { print n + 0 }' <<<"$EXTHIT")"
    if [[ "$n" -lt "$dwant" ]]; then
      REPORT+=("STALE EXT   $EXT_FILE declares $dwant site(s) of '$dform' in $dfile, but the tree has $n — the reference it named is gone")
      FAIL=1
    elif [[ "$n" -gt "$dwant" ]]; then
      REPORT+=("BROAD EXT   $EXT_FILE declares $dwant site(s) of '$dform' in $dfile, but the tree now has $n — the exemption has grown to cover a citation it was never read against; read the new site and bump the count")
      FAIL=1
    fi
  done <<<"$EXTERNAL"
fi

# A dead scope token cannot make this script green when it should be red. What it does
# instead is worse to debug: the exemption misses, the site is checked as a live
# citation, and if that citation is also broken the red line says "bad citation" about
# one that is fine. Hence a WARN — nothing is unsound, but the next reader's attribution
# is. Only SCOPED markers are audited: a bare marker has no spelling to get wrong, and
# once its line's citations are deleted it is indistinguishable from a line that merely
# NAMES the marker, of which this tree has a dozen. `citation-lint:nomarker` opts out a
# line that both cites a rule and quotes a scoped marker verbatim.
markers() {
  git ls-files -z -- '*.go' '*.sh' '*.md' \
    | xargs -0 grep -nH 'citation-lint:quote(' 2>/dev/null \
    | grep -v "^${EXT_FILE}:" || true
}

N_WARN=0
while IFS= read -r hit; do
  [[ -n "$hit" ]] || continue
  mfile="${hit%%:*}"; mrest="${hit#*:}"; mline="${mrest%%:*}"
  mtext="$(sed -n "${mline}p" "$mfile" 2>/dev/null)" || continue
  [[ "$mtext" == *citation-lint:nomarker* ]] && continue
  mcites="$(grep -oE '§[0-9]+(\.[0-9]+| rule [0-9]+)' <<<"$mtext" || true)"
  [[ -n "$mcites" ]] || continue
  mscope="${mtext#*citation-lint:quote(}"; mscope="${mscope%%)*}"
  IFS=, read -ra mtoks <<<"$mscope"
  for tok in "${mtoks[@]}"; do
    if ! grep -qxF "§$tok" <<<"$mcites"; then
      REPORT+=("WARN MARKER $mfile:$mline  scope token '$tok' suppresses nothing on this line — a scope is matched literally, so check its spelling against the citation, or drop it")
      N_WARN=$((N_WARN + 1))
    fi
  done
done < <(markers)

if [[ "${#REPORT[@]}" -gt 0 ]]; then printf '%s\n' "${REPORT[@]}"; fi
if [[ "$FAIL" -eq 0 ]]; then
  # The warn count is printed even when it is zero: a tally that appears only when
  # nonzero is indistinguishable from a check that did not run.
  # The denominator is what was PARSED, not what exists: two sections is correct because
  # only §5 and §7 have top-level numbered items, and every §3/§4 citation is externally
  # declared or a marked mention. A parser that silently stopped early would also print a
  # small number here, so it is printed rather than asserted.
  printf 'citation-lint: %s sites over %s item(s) in %s section(s) resolve against %s (%s declared external, %s quoted, %s dead marker scope(s))\n' \
    "$N_SITES" "$(wc -l <<<"$ITEMS" | tr -d ' ')" \
    "$(awk -F'\t' '{print $1}' <<<"$ITEMS" | sort -u | wc -l | tr -d ' ')" \
    "$DESIGN" "$N_EXT" "$N_QUOTE" "$N_WARN"
fi
exit "$FAIL"
