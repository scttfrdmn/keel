#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# citation-lint.sh — resolve every DESIGN.md rule citation in the tree against
# DESIGN.md's actual structure, and pin what each one lands on.
#
# # Why this exists, and the three things it does not do
#
# #85 began as a spelling problem and turned out to be three different problems
# wearing one costume. What survives is a check with a deliberately narrow claim,
# so read the limits before trusting a green:
#
#   1. RESOLVE, by meaning and not by spelling. Two notations are in use and both
#      are legitimate: the explicit `§5 rule 5` (83 sites) and the shorthand citation-lint:quote
#      `§5.5` (29 DESIGN-bound sites), which means "§5, item 5". An earlier draft citation-lint:quote
#      of this script condemned the shorthand *by form* — DESIGN.md has no
#      subsections, so `§X.Y` looked incoherent on its face. That draft would have
#      demanded 25 zero-semantic edits to correct citations in the document the
#      gates cite as grounds, which is minting rot in the name of style. Both
#      forms now resolve to (section, item) and are checked identically. #85's
#      sweep moved 15 sites from shorthand to explicit — the ones it found
#      MIS-MINTED — and left the 25 correct shorthand sites byte-for-byte.
#
#   2. SCOPE to the document the citation actually names. Four `§5.x` references
#      in this tree point at `docs/spill-report.md`, not DESIGN.md, and two point
#      at the BLIS paper's own §4.3 (Van Zee & van de Geijn, TOMS 2015). They are citation-lint:quote
#      indistinguishable from the DESIGN shorthand by form. A form-based rule
#      would have filed a peer-reviewed paper's section numbering as a defect in
#      keel's constitution — a false defect, which costs more than a missed one.
#      Non-DESIGN references are therefore DECLARED in the pin file's EXTERNAL
#      block, keyed by file and form and carrying the number of sites the declaration
#      covers, and an undeclared one is an error rather than a guess:
#      declared-then-checked, per DESIGN.md §5 rule 6.
#
#   3. PIN the audited baseline. Each distinct form is recorded beside the first
#      words of the rule it lands on. Renumbering or rewording DESIGN.md breaks
#      the match and this goes red.
#
# # What a green from this script does NOT mean
#
# It does not mean the citations are correct. A pin certifies STABILITY, never
# BIRTH-CORRECTNESS, and this is not a hypothetical: keel's 18 `§5.4` citations citation-lint:quote
# were minted wrong in the same commit that appended the rule as item 5
# (4643b63) — no renumbering ever happened, so every pin would have passed
# forever while 18 sites misdirected every reader. Two more sites (DESIGN.md:129
# citing §7 rule 6 for rule 7's denominator clause, l1_test.go:26 citing §5.2 for citation-lint:quote
# rule 1's tolerance clause) were mis-minted inside populations that resolve
# perfectly.
#
# Drift-detection, sameness-checking and mint-verification are three different
# instruments. This script is the first. The second is `make lint`'s other
# checks. The third cannot be automated: it is a human or model reading each
# cited ordinal against the content the citing site actually invokes, and it was
# run once over all 112 DESIGN-bound sites (see the pin file header for the
# census and date). The mechanical check below guarantees what it appears to only
# because that audit established mint-correctness upstream of it.
#
# A MENTION of a citation form is not a citation: a comment explaining a drift by
# showing the bad form, or prose naming a notation while describing this very check,
# invokes no rule's content and must not be resolved against one. (This file's own
# CHANGELOG entry tripped the narrowness check twice for exactly that reason — writing
# about `§4.3` is not citing `§4.3`.) Mark those lines `citation-lint:quote`
# and they are skipped; normalizing them would erase the record of the defect
# they document (same law as #79: history gets marked, never rewritten). Where a
# line carries both a quotation and real citations, scope the marker to the quoted
# form: `citation-lint:quote(5.4)`. See the `quoted` helper for why.
#
# Usage:
#   citation-lint.sh            check against docs/citation-targets.txt
#   citation-lint.sh --write    regenerate that file from the tree
#   citation-lint.sh --list     print every citation site, resolved

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
DESIGN="DESIGN.md"
PINS="docs/citation-targets.txt"
MODE="${1:-check}"

die() { printf 'citation-lint: %s\n' "$1" >&2; exit 2; }
[[ -f "$DESIGN" ]] || die "no $DESIGN to resolve against"

# ---------------------------------------------------------------- the document
# Sections are `## N. Title`. Inside a section, a top-level ordered item is a
# line starting `N. ` at column 0; its lead is the first 56 characters of the
# item's text with markdown emphasis stripped. Continuation lines and nested
# lists are not items and must not be counted, or every rule number would shift.
rules() {
  awk '
    /^## [0-9]+\./ {
      sec = $2; sub(/\./, "", sec)
      cursec = sec
      next
    }
    /^[0-9]+\. / && cursec != "" {
      n = $1; sub(/\./, "", n)
      text = $0; sub(/^[0-9]+\. */, "", text)
      gsub(/\*\*/, "", text); gsub(/\*/, "", text); gsub(/`/, "", text)
      printf "%s\t%s\t%s\n", cursec, n, substr(text, 1, 56)
    }
  ' "$DESIGN"
}

RULES="$(rules)"
[[ -n "$RULES" ]] || die "parsed no numbered rules out of $DESIGN"

lookup() { # SECTION ITEM -> lead, empty if absent
  awk -F'\t' -v s="$1" -v r="$2" '$1 == s && $2 == r { print $3; exit }' <<<"$RULES"
}
section_exists() { awk -F'\t' -v s="$1" '$1 == s { f = 1 } END { exit !f }' <<<"$RULES"; }

# --------------------------------------------------------- declared externals
# Lines of the form `FILE<TAB>FORM<TAB>COUNT<TAB>document it actually cites` in the
# pin file's EXTERNAL block. Keyed by file and form rather than line number, so an
# edit that moves a line does not invalidate the declaration -- but COUNT is what
# keeps the exemption honest. See the narrowness check below.
externals() {
  [[ -f "$PINS" ]] || return 0
  awk '/^## EXTERNAL/ { e = 1; next } /^## PINS/ { e = 0 } e && !/^#/ && NF' "$PINS"
}
EXTERNAL="$(externals)"

is_external() { # FILE FORM
  [[ -n "$EXTERNAL" ]] || return 1
  awk -F'\t' -v f="$1" -v c="$2" '$1 == f && $2 == c { found = 1 } END { exit !found }' <<<"$EXTERNAL"
}

# ------------------------------------------------------------- the citations
# NOTE, learned the hard way: `git ls-files` sees only TRACKED files, so a new file's
# citations are invisible to this check until it is committed. These two scripts were
# untracked while being written, the lint could not see its own sources, and the commit
# that added them turned a green tree red. A green here means "every citation in a
# tracked file", and `git status` is part of reading it. (A second self-reference to
# watch: a line that documents the marker by naming it is thereby suppressed. Benign
# where such lines only mention forms, which is the only place they occur, but it is a
# property of a substring match rather than a decision anyone made.)
#
# Both notations, in one pass: `§5 rule 5` and `§5.5`. The pin file is a .txt and citation-lint:quote
# so is not matched by these globs, but exclude it explicitly anyway -- a lint
# that lints its own baseline reports its pins as sites.
sites() {
  git ls-files -z -- '*.go' '*.sh' '*.md' \
    | xargs -0 grep -nHoE '§[0-9]+(\.[0-9]+| rule [0-9]+)' 2>/dev/null \
    | grep -v "^${PINS}:" || true
}

# A marker suppresses the line it sits on, and it MUST sit on the citation's own line
# -- a marker one line off silently stops working, which is control T2's job. Two forms:
#
#   citation-lint:quote           suppress every citation on this line
#   citation-lint:quote(5.4,7.9)  suppress only these forms on this line
#
# The scope list omits the section sign deliberately: with it, the marker's own
# argument is a citation-shaped string on the line and the scanner counts it as a
# site it then suppresses, inflating the quoted tally by one per marker and leaving a
# reader unable to reconcile "four quotations" with "five quoted".
#
# The scoped form exists because line granularity is too coarse for this document:
# DESIGN.md's numbered rules are single lines of two thousand characters, so a bare
# marker on §5 rule 9 -- which quotes `§5.4` to explain the defect -- would stop citation-lint:quote
# checking every other citation in the same rule. That is over-suppression by
# construction, and it is exactly the failure mode the EXTERNAL narrowness check
# guards against on the other side, so it gets the same treatment: an exemption is
# only trustworthy while it is as narrow as it claims.
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
RESOLVED=""
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
    REPORT+=("NO SECTION  $file:$line  '$cite' — $DESIGN has no §$sec, and this site is not declared EXTERNAL in $PINS")
    FAIL=1
    continue
  fi
  tgt="$(lookup "$sec" "$num")"
  if [[ -z "$tgt" ]]; then
    REPORT+=("NO ITEM     $file:$line  '$cite' — §$sec has no item $num, and this site is not declared EXTERNAL in $PINS")
    FAIL=1
    continue
  fi
  RESOLVED="$RESOLVED$cite"$'\t'"$tgt"$'\n'
  if [[ "$MODE" == "--list" ]]; then REPORT+=("ok          $file:$line  '$cite' -> $tgt"); fi
done < <(sites)

PINNED="$(sort -u <<<"$RESOLVED" | sed '/^$/d')"

# ------------------------------------------------- the declarations themselves
# An EXTERNAL line is keyed by (file, form), which is not intrinsically unique: if
# KERNEL.md ever cites DESIGN.md's §5 rule 2 in the shorthand, the declaration that citation-lint:quote
# exempts KERNEL.md's `docs/spill-report.md` reference would silently exempt it too,
# and a real DESIGN citation would leave the pinned set without a word.
#
# So a declaration states HOW MANY sites it covers, and must match exactly that many.
# The invariant is not "exactly one site" -- CHANGELOG.md legitimately cites the BLIS
# paper's §4.3 three times -- it is "exactly the number of sites someone read". Fewer citation-lint:quote
# means the declaration is stale; more means it has silently grown to cover a citation
# no one checked, and the fix is to read the new site and bump the count deliberately.
# An exemption is only trustworthy while it is as narrow as it claims to be.
if [[ -n "$EXTERNAL" ]]; then
  while IFS= read -r decl; do
    [[ -n "$decl" ]] || continue
    IFS=$'\t' read -r dfile dform dwant _ <<<"$decl"
    if ! [[ "$dwant" =~ ^[0-9]+$ ]]; then
      REPORT+=("BAD EXT     $PINS declares '$dform' in $dfile without a site count in its third column")
      FAIL=1
      continue
    fi
    n="$(awk -F'\t' -v f="$dfile" -v c="$dform" '$1 == f && $2 == c { n++ } END { print n + 0 }' <<<"$EXTHIT")"
    if [[ "$n" -lt "$dwant" ]]; then
      REPORT+=("STALE EXT   $PINS declares $dwant site(s) of '$dform' in $dfile, but the tree has $n — the reference it named is gone")
      FAIL=1
    elif [[ "$n" -gt "$dwant" ]]; then
      REPORT+=("BROAD EXT   $PINS declares $dwant site(s) of '$dform' in $dfile, but the tree now has $n — the exemption has grown to cover a citation it was never read against; read the new site and bump the count")
      FAIL=1
    fi
  done <<<"$EXTERNAL"
fi

case "$MODE" in
  --write)
    { echo "# Copyright 2026 Scott Friedman"
      echo "# SPDX-License-Identifier: Apache-2.0"
      echo "#"
      echo "# Generated by scripts/citation-lint.sh --write (#85). Two blocks."
      echo "#"
      echo "# PINS: one line per distinct DESIGN.md citation form in the tree, beside the"
      echo "# first words of the item it resolves to. A changed line means DESIGN.md"
      echo "# renumbered or an item was reworded; read the diff before regenerating."
      echo "#"
      echo "# EXTERNAL: citations in DESIGN.md's own notation that name a DIFFERENT"
      echo "# document, declared per file and form -- with the number of sites each"
      echo "# declaration covers, so it cannot silently grow -- and so exempted knowingly"
      echo "# rather than guessed at. An undeclared unresolvable citation is an error."
      echo "#"
      echo "# AUTHORITY. This baseline is AUDITED, not assumed. On 2026-08-16 every"
      echo "# DESIGN-bound citation site was read at the meaning level -- cited ordinal"
      echo "# against the content the citing site actually invokes -- because a pin"
      echo "# certifies stability, never birth-correctness, and would freeze a"
      echo "# wrong-from-birth ordinal with perfect fidelity. Census BEFORE the sweep:"
      echo "#   120 sites; 8 naming another document; 112 DESIGN-bound, of which 92"
      echo "#   land on the content the citing site invokes and 20 were mis-minted"
      echo "#   (18 x '§5.4' + DESIGN.md:129 '§7 rule 6' + l1_test.go:26 '§5.2')."  # citation-lint:quote
      echo "# The sweep rewrote 16 of the 20 to the explicit form and marked the other"
      echo "# 4 citation-lint:quote -- they are deliberate quotations of the bad form,"
      echo "# and normalizing them would erase the defect they document. Of the 92 that"
      echo "# were already right, the 25 using the §X.Y shorthand were left byte-for-byte:"
      echo "# rewriting a correct citation in the document the gates cite as grounds is"
      echo "# churn dressed as rigour."
      echo "#"
      echo "# The CURRENT tally is deliberately not recorded here. It changes every time"
      echo "# anyone cites a rule, and a hardcoded copy of it would be a cache with no"
      echo "# invalidation protocol -- the trap DESIGN.md §5 rule 8 names. The script"
      echo "# prints it on every run; the census above is a dated historical fact about"
      echo "# one audit and does not go stale."
      echo "# A pin failure therefore means a reference MOVED off an audited target. It"
      echo "# does not re-establish that the target was ever right: that took reading,"
      echo "# once, and a later sweep of this size needs reading again."
      echo ""
      echo "## EXTERNAL"
      if [[ -n "$EXTERNAL" ]]; then printf '%s\n' "$EXTERNAL"; fi
      echo ""
      echo "## PINS"
      echo "$PINNED"
    } > "$PINS"
    printf 'citation-lint: wrote %s (%s forms, %s external declarations)\n' \
      "$PINS" "$(wc -l <<<"$PINNED" | tr -d ' ')" "$(printf '%s' "$EXTERNAL" | grep -c . || true)"
    ;;
  --list)
    if [[ "${#REPORT[@]}" -gt 0 ]]; then printf '%s\n' "${REPORT[@]}"; fi
    printf '%s DESIGN-bound sites, %s declared external, %s quoted\n' "$N_SITES" "$N_EXT" "$N_QUOTE"
    ;;
  *)
    if [[ ! -f "$PINS" ]]; then
      REPORT+=("NO PINS     $PINS is missing; run citation-lint.sh --write and commit it")
      FAIL=1
    else
      DIFF="$(diff <(awk '/^## PINS/ { p = 1; next } p && !/^#/ && NF' "$PINS") \
                   <(printf '%s\n' "$PINNED") || true)"
      if [[ -n "$DIFF" ]]; then
        REPORT+=("MOVED       a citation now resolves to different text than $PINS pins:")
        while IFS= read -r d; do REPORT+=("            $d"); done <<<"$DIFF"
        FAIL=1
      fi
    fi
    ;;
esac

if [[ "$MODE" != "--list" && "$MODE" != "--write" ]]; then
  if [[ "${#REPORT[@]}" -gt 0 ]]; then printf '%s\n' "${REPORT[@]}"; fi
  if [[ "$FAIL" -eq 0 ]]; then
    printf 'citation-lint: %s sites over %s forms resolve and match %s (%s external, %s quoted)\n' \
      "$N_SITES" "$(wc -l <<<"$PINNED" | tr -d ' ')" "$PINS" "$N_EXT" "$N_QUOTE"
  fi
fi
exit "$FAIL"
