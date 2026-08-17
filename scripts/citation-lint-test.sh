#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Transition exercise for scripts/citation-lint.sh: drive every branch on purpose.
#
# A healthy tree reaches exactly one of the twelve cases below — the clean path. So a
# green `make lint` says nothing whatever about the other eleven, and an unchanged tally
# on a healthy run is not evidence that a newly added check works.
# Each case here induces the condition it names, asserts the branch that must fire,
# and restores the tree.
#
# Controls are numbered in the order they were WRITTEN, not the order they run: T10 is
# last on purpose (see below) and T11-T12 were added after it. Renumbering would silently
# invalidate every reference to a control by name, here and in CHANGELOG.md.
#
# Five of these controls exist because the checks they drive were added *after* the
# audit that established the baseline, and each guards a silent failure rather than a
# loud one:
#
#   T2 — the `citation-lint:quote` marker must share a line with the citation it
#        suppresses. Wrapping prose can separate them, and the marker then stops
#        working with no diff to notice. T2 removes a marker and asserts the site
#        rejoins the pinned set. (This caught a real break: rewrapping a comment in
#        gate-p3.sh moved a marker one line off its citation.)
#   T7 — an EXTERNAL declaration is keyed by (file, form), which is not intrinsically
#        unique. If KERNEL.md ever cites DESIGN.md's §5 rule 2 in the shorthand, the citation-lint:quote
#        declaration exempting KERNEL.md's `docs/spill-report.md` reference would
#        exempt the DESIGN citation too, and a real citation would leave the pinned
#        set unremarked. T7 induces exactly that and asserts BROAD EXT. This check
#        earned its keep immediately: writing the CHANGELOG entry for it added two
#        more BLIS `§4.3` references and the check caught its own documentation citation-lint:quote
#        widening an exemption by two sites — one of which, on reading, turned out not
#        to cite the paper at all.
#   T9 — a scoped marker (`citation-lint:quote(5.4)`) must suppress only the form it
#        names. Line granularity is too coarse for DESIGN.md, whose rules are single
#        2000-character lines; a bare marker on §5 rule 9 would have stopped checking citation-lint:quote
#        two live citations while printing nothing at all.
#   T11 — a scope token is matched literally, so one that names no form on its line
#        suppresses nothing. This does not fail the lint, which is why it needs a
#        control of the `clean:PATTERN` kind: the whole risk is a warning that stops
#        being printed and is therefore indistinguishable from a tree with nothing to
#        warn about. The mutation reproduces a defect the tree actually carried, and T12
#        continues from it to drive `citation-lint:nomarker`, the audit's one escape.
#
# T10 re-runs the clean path last: a restore that silently failed would otherwise let
# every later run inherit a tree this script broke. It is a control on the harness,
# not on the lint. (An earlier draft restored with `git checkout --`, which reverted
# two files to HEAD and discarded uncommitted work; file backups cannot do that.)
#
# Run: bash scripts/citation-lint-test.sh   (part of `make lint`, and standalone)
set -uo pipefail

# Whole-file parse before the first check runs, per #51: bash reads a script
# incrementally, so a top-level script can be corrupted by an edit landing mid-run.
main() {
  cd "$(dirname "$0")/.." || exit 2

  # FILES and BAK are deliberately NOT `local`: the EXIT trap runs after main returns,
  # so a local BAK is out of scope by then and `set -u` turns the cleanup into an
  # error that leaks the temp directory. Caught by running this file, not by reading it.
  FILES=(docs/citation-targets.txt DESIGN.md CHANGELOG.md KERNEL.md l1_test.go)
  BAK="$(mktemp -d)"
  trap 'restore; rm -rf "$BAK"' EXIT

  save() { local f; for f in "${FILES[@]}"; do cp "$f" "$BAK/${f//\//_}"; done; }
  restore() { local f; for f in "${FILES[@]}"; do cp "$BAK/${f//\//_}" "$f"; done; }

  FAILED=0
  # run WANT LABEL [ARGS...] -- three forms of WANT, because the lint has three kinds
  # of outcome and collapsing them loses the distinction that matters:
  #
  #   clean            rc 0, nothing asserted about the text
  #   clean:PATTERN    rc 0 AND the output matches PATTERN -- for a WARN, which by
  #                    design does not fail. Asserting only rc here would pass on a
  #                    build where the warning was never emitted at all.
  #   PATTERN          rc non-zero AND the output matches PATTERN
  #
  # Each outcome is keyed to its own phrase, and a WANT that matches nothing fails:
  # a check anchored to what the script prints *today* would certify the past.
  run() {
    local want="$1" label="$2"; shift 2
    local out rc
    out="$(bash scripts/citation-lint.sh "$@" 2>&1)"; rc=$?
    printf '\n=== %s\n' "$label"
    printf '%s\n' "$out" | grep -vE '^ok ' | sed '/^$/d'
    printf 'rc=%s\n' "$rc"
    if [[ "$want" == clean ]]; then
      if [[ "$rc" -ne 0 ]]; then printf '  UNEXPECTED: wanted a clean pass\n'; FAILED=1; fi
    elif [[ "$want" == clean:* ]]; then
      if [[ "$rc" -ne 0 ]]; then printf '  UNEXPECTED: wanted a clean pass\n'; FAILED=1
      elif ! grep -qE "${want#clean:}" <<<"$out"; then
        printf '  UNEXPECTED: passed, but without %s\n' "${want#clean:}"; FAILED=1
      fi
    else
      if [[ "$rc" -eq 0 ]]; then printf '  UNEXPECTED: wanted a failure\n'; FAILED=1
      elif ! grep -qE "$want" <<<"$out"; then
        printf '  UNEXPECTED: failed, but not with %s\n' "$want"; FAILED=1
      fi
    fi
  }

  save
  printf 'citation-lint transition exercise — every branch driven on purpose.\n'
  printf 'An unchanged healthy tree reaches only T1.\n'

  # The count is asserted, not just the rc. A dozen lines in this tree NAME the quote
  # marker while citing nothing, and the marker audit must read them as mentions; if it
  # ever starts counting them, this line is where that shows up.
  run 'clean:0 dead marker scope' "T1 clean path — the ONLY branch an unchanged healthy tree reaches"

  perl -i -pe 's/ <!-- citation-lint:quote -->//' CHANGELOG.md
  run 'MOVED' "T2 quote marker removed from CHANGELOG.md — proves the marker is load-bearing"
  restore

  perl -i -pe 's/^8\. \*\*A summary is a cache/8. **A summary is a CACHE/' DESIGN.md
  run 'MOVED' "T3 a pinned DESIGN.md rule reworded — the drift the pin exists to catch"
  restore

  perl -i -ne 'print unless m{^internal/block/tri\.go\t}' docs/citation-targets.txt
  run 'NO SECTION' "T4 tri.go's BLIS declaration deleted — an undeclared other-document citation"
  restore

  perl -i -pe 's{^## PINS}{docs/hosts.md\t\xc2\xa79.9\t1\ta citation that does not exist\n\n## PINS}' \
    docs/citation-targets.txt
  run 'STALE EXT' "T5 a declaration for a citation not in the tree — a stale exemption"
  restore

  # A declaration without a site count cannot be checked for narrowness at all, so it
  # is rejected rather than defaulted: a silently-assumed count is the thing the count
  # column exists to prevent.
  perl -i -pe 's{^(internal/block/tri\.go\t\xc2\xa74\.3)\t1\t}{$1\t}' docs/citation-targets.txt
  run 'BAD EXT' "T6 a declaration missing its site count — rejected, not defaulted"
  restore

  perl -i -pe 's{the \xc2\xa75 rule 5 methodology}{the \xc2\xa75.2 differential rule and the \xc2\xa75 rule 5 methodology}' \
    KERNEL.md
  run 'BROAD EXT' "T7 a second KERNEL.md shorthand '§5.2' — the exemption silently widens to cover it"  # citation-lint:quote
  restore

  perl -i -pe 's/DESIGN\.md \xc2\xa75 rule 1\)/DESIGN.md \xc2\xa75 rule 99)/' l1_test.go
  run 'NO ITEM' "T8 a citation of an ordinal its section does not have"
  restore

  # A scoped marker must suppress ONLY the form it names. DESIGN.md's §5 rule 9 is a citation-lint:quote
  # single 2000-character line carrying three forms: a quoted `§5.4`, an EXTERNAL citation-lint:quote
  # `§4.3` naming the BLIS paper, and a live `§5 rule 5`. A bare marker there would citation-lint:quote
  # have stopped checking all three, printing nothing and looking identical to a pass
  # -- which is why the scoped form exists at all. Drive it: break the live citation
  # on that same line and assert the lint still catches it.
  perl -i -pe 's/new citations use the explicit `\xc2\xa75 rule 5`/new citations use the explicit `\xc2\xa75 rule 99`/' DESIGN.md
  run 'NO ITEM' "T9 a broken citation sharing a line with a scoped quote marker — proves the scope is narrow"
  restore

  # A scope token is matched literally, so a mis-spelled or outdated one suppresses
  # nothing -- and does not fail: the citation it was meant to exempt is simply checked.
  # That is sound but MISATTRIBUTING, because the resulting red line (when the citation
  # is also broken) says "bad citation" about a citation that is fine. Hence a WARN, and
  # hence a control that asserts rc 0 alongside the warning: a WARN nobody can see is
  # the same as no check. The mutation below restores a defect that was real in this
  # tree until the commit that added the audit -- `4.3` on a line whose `§4.3` had moved citation-lint:quote
  # to the next one when the prose was rewrapped, which is the same wrapping hazard T2
  # covers for bare markers.
  perl -i -pe 's{<!-- citation-lint:quote\(5\.4\) -->}{<!-- citation-lint:quote(5.4,4.3) -->}' CHANGELOG.md
  run "clean:WARN MARKER CHANGELOG.md.*'4\.3' suppresses nothing" \
    "T11 a scope token naming a form not on its line — warns, and does NOT fail"

  # T12 continues from T11's mutated tree on purpose: the escape hatch is only meaningful
  # against a line that WOULD warn, and asserting it on a quiet line would prove nothing.
  # `citation-lint:nomarker` exists for the one reachable false positive -- a line that
  # cites a rule and also quotes a scoped marker verbatim -- and an escape nobody drives
  # is exactly the dead configuration this audit was added to report.
  perl -i -pe 's{(<!-- citation-lint:quote\(5\.4,4\.3\) -->)}{$1 <!-- citation-lint:nomarker -->}' CHANGELOG.md
  run 'clean:0 dead marker scope' "T12 the same dead token, declared with citation-lint:nomarker — silent"
  restore

  run 'clean:0 dead marker scope' "T10 clean path again — a control on this harness, not on the lint"

  echo
  if [[ "$FAILED" -eq 0 ]]; then
    echo "citation-lint controls: all 12 transitions fired as specified"
  else
    echo "citation-lint controls: FAILED" >&2
    exit 1
  fi
}

main "$@"
