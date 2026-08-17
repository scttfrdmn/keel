#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# docs-gen.sh -- put the generated pages into doc-site/ so `mkdocs build` has a
# complete tree. Run it before any build; `make docs` does.
#
# THE ONE LAW THIS SCRIPT EXISTS TO ENFORCE: nothing on the site is a
# hand-maintained duplicate of something the repository already holds. Two
# mechanisms, and the difference between them matters:
#
#   LINKED (records/*.md, except methodology). A symlink to the repository's own
#   file. mkdocs reads through it, so the page IS the record -- there is no second
#   copy to drift, and editing the site page edits the record, which is why these
#   are gitignored: an accidental edit should be visible as a change to
#   DESIGN.md, not invisible as a change to a page nobody diffs.
#
#   EXTRACTED (numbers.md, records/methodology.md). A section of a repository
#   file, reproduced verbatim with a generated frame around it. This is the form
#   that needs checks, because an extraction can succeed at producing a file and
#   fail at producing the right one -- so both extractions FAIL THE BUILD rather
#   than emit a thin page. A numbers page missing its numbers is not a page with
#   one table fewer; it is a page that says keel has no measurements.
#
# NEVER HAND-COPY A NUMBER ONTO THE NUMBERS PAGE. The rates live in README.md's
# keel-numbers block, which scripts/gate-p5.sh criterion 9 re-measures on every
# benchmark host and fails on a disagreement past README_TOL=0.05. Extracting from
# that block is what puts the site's figures under that gate; a number typed onto
# this page by hand would be under nothing. The denominator each row carries is
# required by DESIGN.md §7 rule 7 -- "never present a number without its
# denominator", and percent-of-measured-peak only when no OpenBLAS reference was
# taken -- which is why this script reads the denominator off the block instead of
# describing it from memory.
set -euo pipefail

README_BEGIN='<!-- keel-numbers: begin -->'
README_END='<!-- keel-numbers: end -->'

SITE="doc-site"
RECORDS="$SITE/records"

die() { echo "docs-gen: $*" >&2; exit 1; }
note() { echo "  $*"; }

# ---------------------------------------------------------------- the records
#
# FILE:PAGE pairs, repository path first. Order is the nav's order, so a reader
# of this list and a reader of mkdocs.yml see the same sequence.
records() {
  cat <<'EOF'
DESIGN.md:design.md
docs/toolchain-notes.md:toolchain-notes.md
KERNEL.md:kernel.md
CHANGELOG.md:changelog.md
CONTRIBUTING.md:contributing.md
EOF
}

link_records() {
  mkdir -p "$RECORDS"
  local pair src page
  while IFS= read -r pair; do
    src="${pair%%:*}"
    page="${pair##*:}"
    [[ -r "$src" ]] || die "records: $src is missing, so the site would ship a nav entry with no page"
    ln -sfn "$PWD/$src" "$RECORDS/$page"
    note "linked   $RECORDS/$page -> $src"
  done < <(records)
}

# ------------------------------------------------- extraction 1: the numbers
#
# The block, its shape, and its provenance are all checked before a byte is
# written, because every one of them is a way for this page to come out looking
# finished and saying something false.
numbers_block() {
  awk -v b="$README_BEGIN" -v e="$README_END" '
    index($0, b) { inb = 1; next }
    index($0, e) { inb = 0; next }
    inb          { print }
  ' README.md
}

# The revision the published rows were measured at, from the provenance sentence
# that follows the block. This is what as-of dates the page: a count labelled
# "now" goes wrong by itself, a count labelled with the run that produced it does
# not.
numbers_rev() {
  awk -v e="$README_END" '
    index($0, e) { after = 1; next }
    after && match($0, /rev `[0-9a-f]{7,40}`/) {
      s = substr($0, RSTART + 5, RLENGTH - 6); print s; exit
    }
  ' README.md
}

gen_numbers() {
  [[ -r README.md ]] || die "README.md is unreadable, and it is the only source of the published rates"
  grep -qF "$README_BEGIN" README.md || die "README.md has no '$README_BEGIN' marker"
  grep -qF "$README_END" README.md   || die "README.md has no '$README_END' marker"

  local bl el
  bl="$(grep -nF "$README_BEGIN" README.md | head -1 | cut -d: -f1)"
  el="$(grep -nF "$README_END" README.md | head -1 | cut -d: -f1)"
  [[ "$bl" -lt "$el" ]] || die "README.md's keel-numbers markers are in the wrong order (begin at line $bl, end at line $el)"

  local block
  block="$(numbers_block)"
  [[ -n "$block" ]] || die "README.md's keel-numbers block is empty"

  # No synthetic figure may reach a numbers page, ever. A [synthetic]-stamped run
  # withholds its verdict by construction; a rate lifted out of one would arrive
  # here with the stamp stripped and nothing left to say it was not a measurement.
  if grep -qF '[synthetic]' <<<"$block"; then
    die "README.md's keel-numbers block contains a [synthetic] figure; a synthetic run's numbers are not published"
  fi

  # Shape: a header row, its separator, and at least one data row, every row with
  # the same cell count as the header. The count is read from the header rather
  # than hardcoded, so a column added to the README table does not fail here for
  # the wrong reason.
  local hdr cells rows bad
  hdr="$(grep -m1 '^|' <<<"$block")" || true
  [[ -n "$hdr" ]] || die "README.md's keel-numbers block contains no markdown table"
  cells="$(awk -F'|' '{ print NF }' <<<"$hdr")"
  grep -qE '^\|[ :-]*-[ :|-]*\|$' <<<"$block" || die "the keel-numbers table has no header separator row"
  rows="$(grep -c '^|' <<<"$block")"
  [[ "$rows" -ge 3 ]] || die "the keel-numbers table has $rows row(s); a header, a separator and at least one measurement are needed"
  bad="$(awk -F'|' -v want="$cells" '/^\|/ && NF != want { print "block line " NR ": " $0 }' <<<"$block")"
  [[ -z "$bad" ]] || die "the keel-numbers table has row(s) whose cell count differs from the header's $((cells - 2)):
$bad"

  local rev nrows nmodels denom
  rev="$(numbers_rev)"
  [[ -n "$rev" ]] || die "no 'rev \`<sha>\`' in README.md's provenance sentence after the block; the page has no run to date itself by"

  # Everything the context lines assert is counted from the block, not typed. The
  # data rows are found by position among the TABLE rows rather than by line
  # number, so a blank line or a sentence inside the block cannot shift the count
  # onto the separator row and report `---` as a CPU model.
  nrows="$((rows - 2))"
  nmodels="$(awk -F'|' '/^\|/ { n++; if (n > 2) { gsub(/^[ \t]+|[ \t]+$/, "", $2); print $2 } }' <<<"$block" | sort -u | grep -c . || true)"

  # Which denominator these rows carry is read off the block rather than assumed.
  # The percentages are currently against keel's own measured microkernel peak,
  # because no OpenBLAS reference was taken at these thread counts.
  # DESIGN.md §7 rule 7 is explicit about that case -- "if OpenBLAS is not
  # installed, say so and report percent-of-measured-peak only" -- so the else
  # branch is the rule being followed, not a fallback, and a page that said
  # "percent of OpenBLAS" over these rows would be the exact mislabel it forbids.
  # The if branch exists so the sentence follows the block if a reference is ever
  # added, rather than having to be remembered.
  #
  # Both citations in this file keep `§7 rule 7` on ONE line deliberately: citation-lint:quote(7 rule 7)
  # citation-lint's scanner reads a line at a time, so a citation wrapped after
  # the `§7` is a citation it cannot see. An unlinted citation is the failure
  # mode the lint exists to prevent, arriving by way of a text wrap.
  if grep -qi 'openblas' <<<"$block"; then
    denom="Each row states the denominator its percentage is against, and these rows carry an
OpenBLAS reference measured on the same host in the same run."
  else
    denom="Each row states the denominator its percentage is against. **No OpenBLAS reference was
taken at these thread counts**, so the only bar these rows are measured against is what
keel's own AVX-512 microkernel achieved on the same host in the same run: read the
percentages as how much of its own kernel the blocked nest keeps, not as a competitive
result."
  fi

  {
    echo "<!-- GENERATED by scripts/docs-gen.sh from README.md's keel-numbers block. Do not edit."
    echo "     Do not add a number to this page by hand: the block it comes from is re-measured by"
    echo "     scripts/gate-p5.sh on every benchmark host, and a hand-typed rate would be under"
    echo "     nothing. Edit README.md, or measure again. -->"
    echo
    echo "# Measured rates"
    echo
    echo "$nrows rows over $nmodels CPU models, all from one gate run, at revision \`$rev\`."
    echo
    echo "$denom"
    echo
    echo "[How these numbers are measured](records/methodology.md)"
    echo
    echo "$block"
  } > "$SITE/numbers.md"
  note "extracted $SITE/numbers.md ($nrows rows, $nmodels CPU models, rev $rev)"
}

# --------------------------------------------- extraction 2: the methodology
#
# Section 5 of DESIGN.md gets a page of its own because it is the section people
# deep-link: it is where the rules a published number has to satisfy are written.
gen_methodology() {
  [[ -r DESIGN.md ]] || die "DESIGN.md is unreadable"
  local body title rules
  body="$(awk '/^## 5\./ { inb = 1 } inb && /^## 6\./ { exit } inb' DESIGN.md)"
  [[ -n "$body" ]] || die "DESIGN.md has no '## 5.' section, so the methodology page cannot be extracted"
  title="$(sed -n '1s/^## [0-9]*\. *//p' <<<"$body")"
  [[ -n "$title" ]] || die "DESIGN.md's section 5 heading did not parse"
  rules="$(grep -cE '^[0-9]+\. ' <<<"$body" || true)"
  [[ "$rules" -ge 1 ]] || die "DESIGN.md's section 5 extracted with no numbered rules in it, which is not the section this page is for"

  mkdir -p "$RECORDS"
  {
    echo "<!-- GENERATED by scripts/docs-gen.sh from DESIGN.md section 5. Do not edit. -->"
    echo
    echo "# $title"
    echo
    echo "Section 5 of the [design document](design.md), reproduced verbatim. It held $rules"
    echo "numbered rules when this page was generated."
    echo
    sed '1d' <<<"$body"
  } > "$RECORDS/methodology.md"
  note "extracted $RECORDS/methodology.md ($rules numbered rules, from DESIGN.md section 5)"
}

main() {
  cd "$(dirname "$0")/.."
  [[ -d "$SITE" ]] || die "$SITE/ is missing; this script fills a site tree in, it does not create one"
  echo "docs-gen: filling in $SITE/"
  link_records
  gen_numbers
  gen_methodology
  echo "docs-gen: ok"
}

main "$@"
