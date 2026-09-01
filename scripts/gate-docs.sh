#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-docs.sh -- the documentation site's gate. Three checks, and no more than
# three: the site is a set of pages, not a measurement campaign, and every line of
# instrumentation past what the pages need is a line nobody asked for.
#
#   1. IT BUILDS, STRICTLY. `mkdocs build --strict` turns a broken link or a nav
#      entry with no page behind it into an error. A page missing from the nav is
#      not a smaller site; it is a page no reader can reach.
#
#   2. THE NUMBERS PAGE IS AN EXTRACTION, AND ITS CHECKS ACTUALLY FIRE. Nobody
#      types a rate onto the numbers page: scripts/docs-gen.sh lifts the table out
#      of README.md's keel-numbers block, which gate-p5 criterion 9 re-measures on
#      every benchmark host. That generator fails closed on nine malformations --
#      and a fail-closed check that has never been seen to fail is a claim, not a
#      check, so this gate drives all nine on purpose against fixtures and requires
#      each to be rejected. It also drives the well-formed fixture, because "every
#      case failed" is also what a generator broken in some unrelated way looks
#      like.
#
#   3. THE CITATIONS RESOLVE. scripts/citation-lint.sh, whose pinning half was
#      retired 2026-08-16 (DESIGN.md §5 rule 9 as amended) -- resolution is the
#      half that can fail, and it is run here because a DESIGN.md citation written
#      on a documentation page is a citation like any other. Note the property that shapes where such a citation may live: the
#      lint enumerates sites with `git ls-files`, so the generated pages are
#      INVISIBLE to it -- they are gitignored. A citation that needs to be linted
#      therefore belongs in the tracked generator, not in the page it emits.
#
# ANCHORS ARE IN SCOPE EVERYWHERE, since #93. They were not, for one commit: all 25
# `[Tn](#tn)` links in docs/toolchain-notes.md pointed at anchors that did not
# exist, so this gate failed on a broken anchor on any USER page and reported the
# records-page ones as a counted exclusion. The ruling that ended that state, and
# the reason the fix and the un-exclusion are one commit: "an exclusion that
# outlives its defect becomes a permanent blind spot with a good excuse." So the
# check below now has no exempt page, and mkdocs.yml validates anchors at `warn`.
# Both, deliberately: `warn` + `strict` fails the build, and the grep fails on the
# INFO-level lines that appear if that setting is ever lowered back to the default.
set -euo pipefail

SITE="doc-site"
LOG="build/docs-gate.log"

# These are this gate's own, and it is the only gate for which that is true --
# remote.sh holds the shared four and the other six gates source them. Deliberate:
# the vocabulary here is `ok`, not a colored `PASS`, this gate contacts no host,
# and nothing delegates to it (the Makefile and both docs.yml jobs run the whole
# script and read its exit status), so there are no verdict lines for another gate
# to count. What it therefore also has none of is remote.sh's VERDICT_STAMP, and
# that is only safe while the premise holds: the stamp exists so a line from a
# SYNTHETIC run cannot be quoted as a certificate, and this gate has no
# instrument-exercise mode, hence no synthetic run. STANDING PRECONDITION: if an
# instrument exercise is ever added here -- anything that makes this script emit a
# verdict line about fabricated input -- source remote.sh or add the stamp in the
# same commit, before the exercise. Otherwise the first synthetic run prints an
# `ok` indistinguishable from a real one, which is the forgery remote.sh:128
# documents gate-p2 having been one commit away from.
FAILS=0
pass() { printf '  ok    %s\n' "$*"; }
fail() { printf '  FAIL  %s\n' "$*"; FAILS=$((FAILS + 1)); }
info() { printf '        %s\n' "$*"; }
head_() { printf '\n%s\n' "$*"; }

# ------------------------------------------------------------------ preflight
#
# A missing mkdocs is a missing tool, not a red gate: reporting "the site does not
# build" when nothing tried to build it is the same error class as reporting an
# exit code for a run that was killed.
check_tools() {
  head_ "0. tools"
  if ! command -v mkdocs >/dev/null 2>&1; then
    echo "gate-docs: mkdocs is not installed. Install it and re-run:" >&2
    echo "    pipx install mkdocs && pipx inject mkdocs mkdocs-material" >&2
    echo "gate-docs: UNRUN (no verdict -- the site was not built)" >&2
    exit 2
  fi
  # `mkdocs --version` prints "mkdocs, version 1.6.1 from ... (Python 3.13)", so
  # the last field is the Python version, not mkdocs'. Match the word after
  # "version".
  pass "mkdocs $(mkdocs --version 2>&1 | awk '{for (i=1;i<NF;i++) if ($i == "version") {print $(i+1); exit}}')"
  if ! python3 -c 'import material' >/dev/null 2>&1 &&
     ! mkdocs get-deps >/dev/null 2>&1; then
    info "could not confirm mkdocs-material by import; --strict will decide"
  fi
}

# ----------------------------------------------------- 1. the site builds
stage_build() {
  head_ "1. mkdocs build --strict"
  mkdir -p "$(dirname "$LOG")"

  if ! ./scripts/docs-gen.sh > "$LOG" 2>&1; then
    fail "scripts/docs-gen.sh refused to generate the site's pages"
    sed 's/^/        /' "$LOG"
    return
  fi
  pass "generated pages present ($(grep -c '^  \(linked\|extracted\)' "$LOG") of them)"

  if mkdocs build --strict >> "$LOG" 2>&1; then
    pass "mkdocs build --strict: clean (no broken link, no unreachable page)"
  else
    fail "mkdocs build --strict failed"
    sed 's/^/        /' "$LOG"
    return
  fi

  # Anchors, every page, no exemption. Redundant with mkdocs.yml's
  # `validation.links.anchors: warn` on a normally-configured build, and that is
  # the point: this one reads the log, so it still fails if the setting is lowered.
  local anchors
  anchors="$(grep -oE "Doc file '[^']+' contains a link '#[^']+', but there is no such anchor" "$LOG" || true)"
  if [[ -z "$anchors" ]]; then
    pass "no broken in-page anchors on any page, user or record"
  else
    fail "broken in-page anchor(s), $(wc -l <<<"$anchors" | tr -d ' ') of them:"
    sed 's/^/        /' <<<"$anchors"
  fi
}

# --------------------------- 2. the numbers extraction, driven until it refuses
#
# Each fixture is a README.md the generator must REJECT, plus one it must accept.
# The fixtures are minimal on purpose: they carry the shape the checks look at and
# nothing else, so a rejection cannot be credited to the wrong defect.
fixture_good() {
  cat <<'EOF'
# fixture
<!-- keel-numbers: begin -->
| CPU | benchmark | threads | GFLOP/s | denominator |
| --- | --- | --- | --- | --- |
| Some CPU | Sgemm | 1 | 100 | 50.0% of 200 GFLOP/s, measured in the same run |
<!-- keel-numbers: end -->
<!-- keel-caption: begin -->
All rows come from one run — `scripts/gate-p5.sh` at rev `abc1234`, log archived.
<!-- keel-caption: end -->
EOF
}

# Each case prints a README variant. `good` is the positive control.
fixture() {
  case "$1" in
    good)          fixture_good ;;
    no-begin)      fixture_good | grep -v 'keel-numbers: begin' ;;
    no-end)        fixture_good | grep -v 'keel-numbers: end' ;;
    reversed)      fixture_good | sed -e 's/keel-numbers: begin/KEEL-TMP/' \
                                      -e 's/keel-numbers: end/keel-numbers: begin/' \
                                      -e 's/KEEL-TMP/keel-numbers: end/' ;;
    empty-block)   fixture_good | grep -vE '^\||^All rows' ;;
    no-table)      fixture_good | sed 's/^|.*/a sentence where the table should be/' ;;
    no-separator)  fixture_good | grep -v '^| --- ' ;;
    ragged-row)    fixture_good | sed 's/^| Some CPU | Sgemm | 1 | 100 |.*/| Some CPU | Sgemm | 1 | 100 |/' ;;
    synthetic)     fixture_good | sed 's/measured in the same run/measured in the same run [synthetic]/' ;;
    no-rev)        fixture_good | sed 's/at rev `abc1234`, //' ;;
    no-caption)    fixture_good | grep -v 'keel-caption: ' ;;
    *)             echo "unknown fixture: $1" >&2; return 1 ;;
  esac
}

# A tree just complete enough for the generator to run in, with the REAL script
# copied into it -- so the self-test exercises shipped code and there is no
# test-only branch inside it to be wrong in a different way.
fake_tree() {
  local t="$1"
  mkdir -p "$t/scripts" "$t/doc-site/records" "$t/docs"
  cp scripts/docs-gen.sh "$t/scripts/"
  # DERIVED from docs-gen's own records() table, not restated. A hand-kept copy of that
  # list silently omits a newly added record page, and the generator then dies on the
  # POSITIVE control -- so the gate reports "well-formed input was REJECTED" and blames
  # the generator for a defect in the fixture. That happened, on docs/rulings.md.
  local f
  while IFS= read -r f; do
    mkdir -p "$t/$(dirname "$f")"
    printf '# f\n' > "$t/$f"
  done < <(awk '/^records\(\)/{i=1} i&&/^EOF$/{exit} i&&/^[^ #(].*:.*\.md$/ {sub(/:.*/,""); print}' scripts/docs-gen.sh)
  # DESIGN.md last: it needs a real section 5, being the page that is extracted rather
  # than linked, and the loop above has just stubbed it.
  printf '# f\n\n## 5. Testing\n\n1. A rule.\n\n## 6. Next\n' > "$t/DESIGN.md"
}

stage_extraction() {
  head_ "2. the numbers extraction fails closed"
  local t out rc
  t="$(mktemp -d)"
  trap 'rm -rf "$t"' RETURN
  fake_tree "$t"

  # Positive control first: if this one fails, every rejection below is worthless
  # as evidence, because a generator that rejects everything looks identical.
  fixture good > "$t/README.md"
  if out="$("$t/scripts/docs-gen.sh" 2>&1)"; then
    if grep -q 'abc1234' "$t/doc-site/numbers.md" &&
       grep -q '| Some CPU | Sgemm | 1 | 100 |' "$t/doc-site/numbers.md" &&
       grep -q '1 rows over 1 CPU models' "$t/doc-site/numbers.md"; then
      pass "well-formed input: accepted, and the page carries the row, the rev and counts read from the block"
    else
      fail "well-formed input was accepted but the page does not carry the block's row, rev and counts"
      sed 's/^/        /' "$t/doc-site/numbers.md"
    fi
  else
    fail "well-formed input was REJECTED, so the rejections below prove nothing:"
    sed 's/^/        /' <<<"$out"
    return
  fi

  local case_ rejected=0
  for case_ in no-begin no-end reversed empty-block no-table no-separator ragged-row synthetic no-rev no-caption; do
    rm -f "$t/doc-site/numbers.md"
    fixture "$case_" > "$t/README.md"
    rc=0
    out="$("$t/scripts/docs-gen.sh" 2>&1)" || rc=$?
    if [[ "$rc" -eq 0 ]]; then
      fail "$case_: ACCEPTED -- a malformed numbers block would reach the site"
    elif [[ -s "$t/doc-site/numbers.md" ]]; then
      fail "$case_: rejected, but a numbers.md was left behind for mkdocs to serve"
    else
      rejected=$((rejected + 1))
      # The LAST docs-gen: line, not the first: the first is the progress banner,
      # and printing that for every case would make nine distinct rejections look
      # like one repeated non-answer.
      info "$case_ -> $(grep '^docs-gen:' <<<"$out" | tail -1 | cut -c11-)"
    fi
  done
  [[ "$rejected" -eq 9 ]] && pass "all 9 malformed inputs rejected, with no page written"

  # The methodology extraction, same treatment: two ways for section 5 to not be
  # section 5, both of which would otherwise emit a plausible-looking page.
  fixture good > "$t/README.md"
  printf '# f\n\n## 4. Other\n\ntext\n\n## 6. Next\n' > "$t/DESIGN.md"
  if "$t/scripts/docs-gen.sh" >/dev/null 2>&1; then
    fail "a DESIGN.md with no section 5 still produced a methodology page"
  else
    pass "DESIGN.md with no section 5: rejected"
  fi
  printf '# f\n\n## 5. Testing\n\nprose, no numbered rules\n\n## 6. Next\n' > "$t/DESIGN.md"
  if "$t/scripts/docs-gen.sh" >/dev/null 2>&1; then
    fail "a section 5 with no numbered rules still produced a methodology page"
  else
    pass "section 5 with no numbered rules: rejected"
  fi
}

# ------------------------------------------------------------ 3. the citations
stage_citations() {
  head_ "3. citation-lint"
  local out
  if out="$(./scripts/citation-lint.sh 2>&1)"; then
    pass "$(grep -iE '^[0-9]+ site|site\(s\)|sites over' <<<"$out" | tail -1)"
    pass "every DESIGN.md citation resolves; whether it resolves to the RIGHT rule is a reading, not a check"
  else
    fail "citation-lint failed:"
    sed 's/^/        /' <<<"$out"
  fi
  # WARN lines are relayed on a GREEN run, which is the only run they can appear on:
  # the lint's dead-scope audit reports without failing, so filtering the output down
  # to the summary line -- which this stage did until the audit existed -- would hide
  # every warning the lint can emit behind a passing gate. A warning nobody sees is
  # not a lighter check than a failure, it is an absent one.
  local warns
  warns="$(grep -E '^WARN ' <<<"$out" || true)"
  if [[ -n "$warns" ]]; then
    info "citation-lint warnings — sound, but the attribution is not; fix or drop each:"
    sed 's/^/        /' <<<"$warns"
  fi
  # The site's own pages are hand-written and tracked, so they are in scope for
  # the lint above. The generated ones are not, by construction -- said out loud
  # here so a future page's citation is not written into a file nothing reads.
  # The linked record pages are covered too, and derived from records() for the same
  # reason fake_tree is: .gitignore is a hand-kept copy of that table, and an omission
  # there does not fail -- `git add -A` commits a SYMLINK to a repository file, putting a
  # second path to DESIGN.md in the tree and a second population into the lint. Nearly
  # landed on docs/rulings.md, whose symlink staged clean.
  local want pages ungated p
  pages="doc-site/numbers.md doc-site/records/methodology.md $(
    awk '/^records\(\)/{i=1} i&&/^EOF$/{exit} i&&/^[^ #(].*:.*\.md$/ {sub(/.*:/,""); print "doc-site/records/" $0}' \
      scripts/docs-gen.sh | tr '\n' ' ')"
  want="$(wc -w <<<"$pages" | tr -d ' ')"
  ungated="$(git check-ignore $pages 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "$ungated" == "$want" ]]; then
    pass "all $want generated pages are gitignored, hence out of citation-lint's reach by design"
  else
    fail "expected every generated page to be gitignored; $ungated of $want are"
    for p in $pages; do git check-ignore -q "$p" || printf '        NOT ignored: %s\n' "$p"; done
  fi
}

# --------------------------------------- the ratio: reported, never a verdict
#
# NOT a fourth check: the charter above still reads three, and this cannot fail. A
# ratio is not a correctness property, and a red one would create the incentive the
# cap exists to remove -- pay down shell to clear a gate rather than to ship a
# routine. Printed here because this is the only gate that contacts no host, hence
# the only one that runs on every push. Definitions printed beside the figure per
# §7 rule 7. tools/ and bench/ are Go and would otherwise land on the library side,
# where an instrument flatters the very ratio it is counted by. Both lines print: the
# historical one unchanged so the published 1.6x series stays comparable, and an
# apparatus one moving them across. Redefining one side silently would make a paydown
# claim unfalsifiable.
#
# bench/ MOVES WHOLE, TESTS INCLUDED, and that is the correction rather than the
# design (2026-08-20). The prior comment here said the library side "includes bench/
# and internal/spill, flattering the ratio by ~100 lines". Measured at this rev:
# bench/ is 1655 lines of which only openblas.go's 106 ever reached the library term,
# so the other 1549 -- the whole benchmark harness, including the 372-line ceiling
# instrument #6 landed the same day -- were counted in NEITHER term. A cap policed by
# a reporter blind to the largest apparatus directory in the tree is a cap with a hole
# in it, and the hole is invisible in exactly the direction that matters.
#
# STILL ON THE LIBRARY SIDE, DISCLOSED AND NOT MOVED: internal/spill's 837 non-test
# lines (spill.go plus cmd/spill-audit) are P2's audit instrument by function and
# would move on the same argument. They are left because "~100 lines" understated
# them by 837 and correcting a count is not licence to also redraw a boundary in the
# same commit -- that is a definition change, and §7 rule 7 wants it argued, not
# bundled into a bug fix.
#
# BOTH TERMS COUNT UNTRACKED-NOT-IGNORED FILES (2026-08-21). `git ls-files` alone is
# tracked-only, so a new script was invisible to this reporter until it was committed --
# and a counter whose job is to police NEW shell was blind in exactly the direction that
# flatters the commit being prepared. 820eac0 read 1.47x while its own 131-line fixture
# file was untracked; the honest figure was 1.49x. Both sides gained -co so correcting the
# shell term is not also a redefinition that lets untracked Go pay down the ratio.
stage_ratio() {
  head_ "apparatus ratio (reported, never a verdict)"
  local sh lib tool toollib bench benchlib ratio aratio oratio app libnet toolt
  sh="$(git ls-files -co --exclude-standard '*.sh' | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  lib="$(git ls-files -co --exclude-standard '*.go' | { grep -v '_test\.go$' || true; } | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  # tools/ AND bench/ each count EVERY *.go, tests included: the 2026-08-20 correction for
  # bench/ and its 2026-08-31 completion for tools/ (#131). The *lib variables are only the
  # part the library term already held, so subtracting them moves each across whole, once.
  tool="$(git ls-files -co --exclude-standard 'tools/*.go' | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  toollib="$(git ls-files -co --exclude-standard 'tools/*.go' | { grep -v '_test\.go$' || true; } | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  bench="$(git ls-files -co --exclude-standard 'bench/*.go' | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  benchlib="$(git ls-files -co --exclude-standard 'bench/*.go' | { grep -v '_test\.go$' || true; } | while read -r f; do wc -l < "$f"; done | awk '{n+=$1} END{print n+0}')"
  app=$((sh + tool + bench))
  toolt=$((tool - toollib))
  libnet=$((lib - toollib - benchlib))
  ratio="$(awk -v a="$sh" -v b="$lib" 'BEGIN { if (b) printf "%.2f", a / b; else printf "n/a" }')"
  aratio="$(awk -v a="$app" -v b="$libnet" 'BEGIN { if (b) printf "%.2f", a / b; else printf "n/a" }')"
  oratio="$(awk -v a="$((app - tool + toollib))" -v b="$libnet" 'BEGIN { if (b) printf "%.2f", a / b; else printf "n/a" }')"
  info "shell ${sh} / library ${lib} / ratio ${ratio}x"
  info "apparatus ${app} (shell ${sh} + tools/ ${tool} + bench/ ${bench}, of which ${toollib} + ${benchlib} were already counted as library) / library ${libnet} / ratio ${aratio}x"
  info "apparatus = shell + tools/ + bench/ INCLUDING their tests; library = *.go net of its tests. The asymmetry is deliberate and only defensible stated: the ratio measures maintenance burden against shipped substance, and a test of an instrument is burden. Both terms count tracked AND untracked-not-ignored, since a counter that polices new shell must see it before it is committed. internal/spill's audit instrument stays on the library side, disclosed. RESTATED 2026-08-31 (#131): tools/*_test.go was in NEITHER term, ${toolt} lines of it here; folding it in reads ${oratio}x under the pre-fold definition against ${aratio}x under this one, which is a DEFINITION correction of one tree and not a regression. The same fold measured 2.69x -> 2.79x on 2026-08-30 when the hole was found; both ends have moved with the tree since, which is why this line computes them instead of quoting them"
  info "HOW TO READ THIS: the absolute shell term (${sh}) is the session-delta reading; the ratio is the headline disclosure only. Standing rule, 2026-08-31. Both terms are five digits, so +78 lines and +137 lines both printed as one hundredth (1.77x -> 1.78x -> 1.79x) with the library term measured constant across all five revisions -- two printed decimals cannot separate two spends differing by 59 lines, and a FLAT RATIO IS NOT A FLAT LEDGER. The running total, itemized per commit and measured not quoted, is docs/apparatus-ledger.md"
}

main() {
  cd "$(dirname "$0")/.."
  echo "gate-docs: the documentation site"
  check_tools
  stage_build
  stage_extraction
  stage_citations
  stage_ratio
  head_ "verdict"
  if [[ "$FAILS" -eq 0 ]]; then
    echo "  GREEN -- the site builds strictly, the numbers page cannot be hand-written,"
    echo "  and every DESIGN.md citation on a tracked page resolves."
    exit 0
  fi
  echo "  RED -- $FAILS check(s) failed."
  exit 1
}

main "$@"
