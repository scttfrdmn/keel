#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# exercise-baseline.sh [HOST] -- drive all three states of the BASELINE-REGISTERED
# verdict class on one real host, before the era-founding campaign has to.
#
# WHAT IS BEING EXERCISED. gate-p5's class (#6, ruled 2026-08-21) decides which bar
# governs a host from the contents of two tracked files, and the row loop that does it
# has never executed: no `frac` has been computed inside it, and `new`, `owing` and
# `registered` are all unentered. Its live debut would otherwise be the pinning
# transition -- the era-founding run, where a wiring bug costs a fleet-wide re-run and
# muddies the both-arms archive it exists to produce. Every induced exercise in this
# project's history found a real defect on first firing (Scott's ruling of 2026-08-21,
# which ordered this file before that campaign).
#
# THE CLASS NOW HAS TWO CRITERIA AND THEY DO NOT SHARE A DERIVATION SET (#119,
# 2026-08-30). Every read-back below is keyed twice: once to the share criterion (`reaches`,
# points, BASELINE_MARGIN, `(#6)`) and once to the ratio criterion (`scales`, x,
# STRSM_MARGIN, `(#119)`). Two consequences a reader of an older version of this file would
# get wrong. First, the host must sit outside BOTH derivation sets or one criterion renders
# `fleet` while the other renders the state being exercised -- which retires this file's own
# usage example: keel-skx derived STRSM_FLOOR and did not derive CEIL_FRACTION, so the host
# named here for two weeks would now exercise the share arm and skip the ratio arm entirely.
# Second, the ratio criterion's row can be out-resolved by rule 19 before it reaches the
# class at all, and that is disclosed per pass rather than absorbed: an unexercised arm is
# stated inside the result (§5 rule 12), never counted as a clean one.
#
# THREE PASSES, ONE HOST, and the third pass carries the era test inside it:
#
#   pass 1  registry empty, witness empty
#           -> BASELINE on every judged routine ("RECORDED as its candidate baseline")
#              and on the README criterion ("its absence here is this host's admission
#              date"), plus the fleet tally and the candidate rows.
#   pass 2  witness row landed, registry still empty
#           -> FAIL "BASELINE is spent" on both criteria, plus the debt line. This is
#              the branch that makes single-shot mean anything, and #114's fix moved
#              its witness from a build/ glob to a tracked file: unexercised, a repo
#              with one landed row and no registry row would have renewed BASELINE.
#   pass 3  registry landed from pass 1's own values, PLUS a decoy row in the WRONG ERA
#           -> judged at (that criterion's own baseline - its own margin), with the line
#              naming era `pinned8`. The decoy is a `free-placement` row at 99.0 for
#              the same CPU and criterion, written ABOVE the real rows because
#              baseline_lookup returns the first match: below them it would prove
#              nothing, since the correct row would win either way. So one pass drives
#              the registered state and both arms of era scoping, and the discriminator
#              is textual and declared here in advance -- a bar of `value - margin` at
#              era `pinned8` confirms scoping; a bar of `99.0 - margin` at era
#              `free-placement` refutes it and is a finding that outranks this run.
#
# WHY A SEAM AND NOT A SHIM. gate-p5.sh:75 carries the argument in full: every route to
# varying tracked content from outside is closed by other checks this gate enforces
# correctly, so the choice is a seam or a branch that never runs until it matters. What
# the seam does is substitute the three files' PATH; it touches no comparison, no margin
# and no tally, so each state below is the shipped code's own reading of rows it was
# handed. The preflight drives both of the seam's fail-closed refusals with `env -u`, so
# neither can pass by inheriting a variable this script happens to have set.
#
# WHAT IT COSTS. Three full gate-p5 runs on one host: three real benchmark sweeps plus
# three delegated gate-p4 chains. The delegated chain is a guaranteed UNMEASURED here --
# the exported stamp makes the child withhold its verdict, so the parent finds no verdict
# line to read -- and it is left in rather than skipped, because skipping it would need a
# branch in the gate and the gate is not modified by this file. Budget roughly an hour
# per pass on a c5n.18xlarge.
#
# HOW IT STAYS HONEST. KEEL_INSTRUMENT_EXERCISE is exported, so every verdict line is
# stamped [synthetic], the banner prints, GREEN/RED is withheld and the exit is 2 -- and
# this script audits the stamping afterwards rather than trusting it. Logs go under
# build/instrument-exercise-baseline-*, never a gate-pN-<rev> path (#78). No line of this
# script's own report may BEGIN with a verdict token, for the reason
# exercise-dead-host.sh records: a wrapped line of driver prose once inflated a
# stamped-line audit by one.
#
# Usage, detached like any long run (CLAUDE.md):
#   scripts/detach.sh run baseline-exercise-<rev> -- ./scripts/exercise-baseline.sh antares.local
set -euo pipefail

# say LINE... -- one place that writes to both the terminal and the combined log, so a
# report line cannot reach one and miss the other.
say() { printf '%s\n' "$@" | tee -a "$LOG"; }

# refuse REASON... -- a preflight that declines to spend host time. Exit 2 is this
# file's only exit, synthetic or aborted: neither is a gate result.
refuse() {
  printf '%s\n' "$@" >&2
  if [[ -n "${LOG:-}" ]]; then printf '%s\n' "$@" >>"$LOG"; fi
  exit 2
}

# strip -- the logs carry colour, every read-back below is a text match.
strip() { sed $'s/\033\\[[0-9;]*m//g' "$1"; }

main() {
  cd "$(dirname "$0")/.."
  REV="$(git rev-parse --short HEAD)"
  LOG="build/instrument-exercise-baseline-$REV.log"
  DIR="build/baseline-exercise-$REV"
  mkdir -p build
  : >"$LOG"

  # ---- the target host. Exactly one, and named rather than inferred where possible:
  # every cost and every verdict below is per host, and a second host would silently
  # triple the bill while making "1 of 1 rendered BASELINE" unreadable.
  if [[ -n "${1:-}" ]]; then
    HOST="$1"
  elif [[ -n "${KEEL_REMOTE_HOSTS:-}" ]]; then
    HOST="${KEEL_REMOTE_HOSTS}"
  else
    HOST="$(grep -v '^[[:space:]]*#' .keel-hosts 2>/dev/null | grep . | head -1 || true)"
  fi
  # shellcheck disable=SC2086  # word-splitting is the count being tested
  set -- $HOST
  if [[ "$#" -ne 1 ]]; then
    refuse "exercise-baseline: need exactly one host, got $# ('$HOST')." \
           "  Three sweeps per host is the bill, and the per-host verdict lines are what" \
           "  this exercise reads back. Pass one host name."
  fi

  say "== instrument exercise: the BASELINE-REGISTERED class, gate-p5 (#6) =="
  say "   rev:            $REV"
  say "   host:           $HOST  (measures normally; nothing about it is shimmed)"
  say "   substituted:    $DIR/{host-baselines,judged-runs,measurement-eras}.tsv"
  say "   NOT a gate run: verdict withheld, exit 2, every verdict line [synthetic]"
  say ""

  preflight
  build_substitution

  say ""
  say "-- pass 1 of 3: registry empty, witness empty (the 'new' state) --"
  run_pass 1 "baseline-class:new"
  readback_new

  land_witness
  say ""
  say "-- pass 2 of 3: witness landed, registry empty (the 'owing' state) --"
  run_pass 2 "baseline-class:owing"
  readback_owing

  land_registry
  say ""
  say "-- pass 3 of 3: registry landed + a wrong-era decoy (the 'registered' state) --"
  run_pass 3 "baseline-class:registered"
  readback_registered

  stamp_audit
  say ""
  say "instrument exercise: COMPLETE, verdict withheld (each pass exited 2, which is the"
  say "expected synthetic exit and a fact about this driver, not about P5)."
  # The coverage claim states what it could not see, at the end and not only three screens up
  # (§5 rule 12): a reader who takes the summary line is the reader this disclosure is for.
  if [[ "${SCALE_EXERCISED:-1}" -eq 0 ]]; then
    say "COVERAGE, stated here because the summary is what gets read: the RATIO criterion's"
    say "three states are UNEXERCISED in this run -- rule 19 out-resolved its row in pass 1, so"
    say "no candidate baseline was proposed for it and no bar of its could be registered. What"
    say "this run exercised is the share and README criteria. Re-run on a quieter host to reach"
    say "the arm #119 added; nothing about this driver needs changing to do it."
  fi
  say "log: $LOG   passes: $DIR/pass{1,2,3}.log   substituted rows: $DIR/*.tsv"
  exit 2
}

# ---------------------------------------------------------------- preflight
#
# Every check here is cheap and every one of them decides what the next three hours
# measure, which is the argument gate-p2's exercise makes for running its shim's
# fixtures before spending host time. One of them contacts the host, once.
preflight() {
  say "-- preflight (nothing below spends a sweep) --"

  # 1. The class's own readers. gate-p5 checks these too, as a criterion; here they are
  # a precondition, because a substituted registry read by a broken reader renders a
  # state that means nothing.
  if ! BT="$(bash scripts/baseline-test.sh 2>&1)"; then
    printf '%s\n' "$BT" >>"$LOG"
    refuse "exercise-baseline: scripts/baseline-test.sh fails, so the readers this" \
           "  exercise drives do not pass their own controls. Not spending host time."
  fi
  say "   ok    baseline-test.sh green ($(grep -c '^  ok ' <<<"$BT") fixtures)"

  # 2. The seam is in the gate's own bytes, and its refusals precede the first line that
  # contacts a host. A seam whose guard sits after the host loop would refuse only after
  # spending the run it was meant to prevent -- and it is the ORDER that makes the guard
  # a guard, so the order is asserted from the file rather than read once and trusted.
  # `|| true` on both, and on every grep-in-a-substitution below: `set -o pipefail` turns a
  # no-match grep into a failed pipeline, so under `set -e` the assignment itself would kill
  # this script and the -z branch under it -- the one that says WHICH read failed -- could
  # never print. A silent exit 1 is exactly the diagnosis this preflight exists to replace.
  SEAM="$(grep -n 'BASELINE_DIR="\$KEEL_INSTRUMENT_BASELINE_DIR"' scripts/gate-p5.sh | cut -d: -f1 || true)"
  FIRSTREMOTE="$(grep -nE '(^|[^#a-z_])remote_(hosts|probe|exec|build_test_or_fail)' scripts/gate-p5.sh | head -1 | cut -d: -f1 || true)"
  if [[ -z "$SEAM" || -z "$FIRSTREMOTE" ]]; then
    refuse "exercise-baseline: cannot locate the seam ('$SEAM') or the gate's first remote" \
           "  call ('$FIRSTREMOTE') in scripts/gate-p5.sh. This driver is keyed to a gate it" \
           "  no longer recognises; re-read it before running it."
  fi
  if [[ "$SEAM" -ge "$FIRSTREMOTE" ]]; then
    refuse "exercise-baseline: the seam resolves at line $SEAM and the gate's first remote" \
           "  call is at line $FIRSTREMOTE, so its refusals cannot fire 'before contacting a" \
           "  host' as the comment beside them claims."
  fi
  say "   ok    seam at gate-p5.sh:$SEAM, first remote call at :$FIRSTREMOTE, so both refusals precede host contact"

  # 3. Both refusals, driven live. `env -u` rather than a bare assignment: if this
  # script is ever run by a shell that already exported KEEL_INSTRUMENT_EXERCISE, a
  # plain `KEEL_INSTRUMENT_BASELINE_DIR=... bash gate-p5.sh` would inherit the stamp and
  # the forgery guard would pass without being tested.
  #
  # The probes are bounded by an unresolvable host list, not by a timeout: if a refusal
  # is broken the gate proceeds, and every remote operation then fails fast against a
  # name that cannot resolve, so no measurement is taken and nothing is judged. That is
  # a finding, loudly, rather than a run to sit through -- and the #78 control below
  # checks that a broken refusal did not sign anything into a gate-pN-shaped path.
  # A GLOB, not a path: the delegated log carries a rev AND a per-process run stamp
  # (RUN_STAMP), so a probe that reached the delegate would write a NEW file rather than
  # change an existing one. Comparing one path's checksum would have watched the wrong
  # thing and reported ok -- the check has to be over the set.
  P4GLOB="build/gate-p4-under-p5-$REV-*.log"
  # shellcheck disable=SC2086  # deliberate glob expansion
  P4BEFORE="$(ls -1 $P4GLOB 2>/dev/null | sort | tr '\n' ' ' || true)"
  BOGUS="keel-exercise-no-such-host.invalid"

  set +e
  R1="$(env -u KEEL_INSTRUMENT_EXERCISE \
          KEEL_INSTRUMENT_BASELINE_DIR="$DIR" KEEL_REMOTE_HOSTS="$BOGUS" \
          bash scripts/gate-p5.sh 2>&1)"
  RC1=$?
  set -e
  if [[ "$RC1" -ne 2 ]] || ! grep -qF 'still be able to sign a verdict' <<<"$R1"; then
    printf '%s\n' "$R1" | tail -20 >>"$LOG"
    refuse "exercise-baseline: the substitution was NOT refused without an exercise stamp" \
           "  (exit $RC1). That combination judges every host against a substituted registry" \
           "  and can still sign gate-p5: GREEN, which is the one thing the seam may not" \
           "  permit. Fix the guard before anything else."
  fi
  say "   ok    substitution without KEEL_INSTRUMENT_EXERCISE: refused, exit 2, before any host"

  set +e
  R2="$(env KEEL_INSTRUMENT_EXERCISE="preflight:not-a-directory" \
          KEEL_INSTRUMENT_BASELINE_DIR="$DIR/definitely-not-here" KEEL_REMOTE_HOSTS="$BOGUS" \
          bash scripts/gate-p5.sh 2>&1)"
  RC2=$?
  set -e
  if [[ "$RC2" -ne 2 ]] || ! grep -qF 'is not a directory' <<<"$R2"; then
    printf '%s\n' "$R2" | tail -20 >>"$LOG"
    refuse "exercise-baseline: a non-existent substitution directory was NOT refused" \
           "  (exit $RC2). An unreadable substitution reads as an empty registry, which is" \
           "  one of the states below, so it must not be reachable by a typo."
  fi
  say "   ok    substitution pointing at a non-directory: refused, exit 2"

  # shellcheck disable=SC2086  # deliberate glob expansion
  P4AFTER="$(ls -1 $P4GLOB 2>/dev/null | sort | tr '\n' ' ' || true)"
  if [[ "$P4BEFORE" != "$P4AFTER" ]]; then
    refuse "exercise-baseline: the refusal probes changed $P4GLOB ('$P4BEFORE' -> '$P4AFTER')," \
           "  so a run this driver launched reached the delegated gate and wrote a" \
           "  gate-pN-shaped log. That is #78's shape, and it means a refusal fired late" \
           "  rather than early even though its exit code was right."
  fi
  say "   ok    no gate-pN-shaped log was written by either probe (#78 control over $P4GLOB)"

  # 4. The tree must be clean, and stays clean for the whole run: gate-p5 refuses the
  # delegated chain on a dirty tree (`git archive HEAD` would ship uncommitted bytes),
  # and every pass below rebuilds its own arms from HEAD.
  if [[ -n "$(git status --porcelain)" ]]; then
    refuse "exercise-baseline: the tree is dirty, so each pass would refuse its delegated" \
           "  chain and the three passes would not be three readings of one program." \
           "$(git status --short)"
  fi
  say "   ok    tree clean at $REV, and it stays frozen for this run's whole life"

  # 5. The era ledger this exercise scopes everything to. Read through the gate's own
  # reader, from the file the substitution will copy.
  # shellcheck source=scripts/gate-lib.sh
  source scripts/gate-lib.sh
  ERA="$(era_current scripts/measurement-eras.tsv)"
  if [[ -z "$ERA" ]]; then
    refuse "exercise-baseline: no era resolves from scripts/measurement-eras.tsv, so gate-p5" \
           "  would FAIL every reading for want of a scope and no state below would be the" \
           "  class's. Land an era row with its amendment first."
  fi
  PROV="closed"
  if era_provisional scripts/measurement-eras.tsv "$ERA"; then PROV="PROVISIONAL"; fi
  say "   ok    era resolves: $ERA ($PROV) -- every row this exercise writes is keyed to it"

  # 6. The host must be governed by the REGISTRY and not by the fleet bar, or the class
  # never fires. Both lists are read out of gate-p5.sh's bytes rather than restated here,
  # so a host added to either derivation set later cannot be exercised as if it were still
  # outside it -- and BOTH are required, because the two criteria's sets differ by one model
  # (#119): a host inside only one of them exercises one criterion and silently renders
  # `fleet` on the other, which reads in the log exactly like a clean single-criterion pass.
  # This is the one preflight step that contacts the host.
  CEIL_DERIVED_FROM="$(sed -n 's/^CEIL_DERIVED_FROM="\(.*\)"$/\1/p' scripts/gate-p5.sh | head -1)"
  SCALE_DERIVED_FROM="$(sed -n 's/^SCALE_DERIVED_FROM="\(.*\)"$/\1/p' scripts/gate-p5.sh | head -1)"
  if [[ -z "$CEIL_DERIVED_FROM" || -z "$SCALE_DERIVED_FROM" ]]; then
    refuse "exercise-baseline: cannot read CEIL_DERIVED_FROM ('$CEIL_DERIVED_FROM') or" \
           "  SCALE_DERIVED_FROM ('$SCALE_DERIVED_FROM') out of scripts/gate-p5.sh, so which" \
           "  bar governs $HOST is unknown to this driver for at least one criterion."
  fi
  # shellcheck source=scripts/remote.sh
  source scripts/remote.sh
  HCPU="$(remote_probe "$HOST" | cut -d'|' -f1 | sed 's/ *$//' || true)"
  if [[ -z "$HCPU" ]]; then
    refuse "exercise-baseline: $HOST did not answer a probe, so its CPU model -- the key to" \
           "  every row this exercise writes -- is unreadable. A host that cannot answer a" \
           "  probe cannot produce the sweep the class reads."
  fi
  # NAME=VALUE, not the name alone: `${!l}` on a name never assigned dies inside the process
  # substitution, whose status nothing checks, so `set -u` skipped the CEIL arm entirely and the
  # ok line below printed anyway (found live 2026-08-30 -- the value was in DERIVED_FROM). Direct
  # expansion in the `for` list aborts the driver instead, which is the only direction a guard may
  # fail. What the miss cost was nil, and by containment rather than by design: CEIL's set is a
  # subset of SCALE's today, so the arm that did run happened to cover both.
  for l in "CEIL_DERIVED_FROM=$CEIL_DERIVED_FROM" "SCALE_DERIVED_FROM=$SCALE_DERIVED_FROM"; do
    while IFS= read -r d; do
      # `if`, not `test && refuse`: an AND-list whose test is false returns 1 at statement
      # position, and under `set -e` that exits the driver -- on the FIRST entry the host does
      # not match, which is every entry in the passing case. The guard would have aborted the
      # exercise precisely when it had nothing to complain about.
      if [[ -n "$d" && "$HCPU" == *"$d"* ]]; then
        refuse "exercise-baseline: $HOST reports '$HCPU', which matches ${l%%=*} entry '$d'. That" \
          "  host is governed by that criterion's fleet bar, so the BASELINE-REGISTERED class" \
          "  never fires on it and pass 3 would hit the both-artifacts-claim-it FAIL instead." \
          "  Point this exercise at a host outside BOTH derivation sets (#119): they differ by" \
          "  one model, so 'outside the other one' is not the same question."
      fi
    done < <(printf '%s\n' "${l#*=}" | tr '|' '\n')
  done
  say "   ok    $HOST is '$HCPU', outside CEIL_DERIVED_FROM and SCALE_DERIVED_FROM, so the registry governs both criteria"

  # 7. What the shipped registry already says about this host, disclosed rather than
  # assumed away: the substitution is built from headers alone, so if scripts/ already
  # carried rows for this CPU the exercise's `new` state would differ from what a real
  # run sees, and a reader is owed that fact either way (§5 rule 12).
  SHIPPED_B="$(awk -F'\t' -v c="$HCPU" '!/^#/ && $1 != "cpu_model" && $1 == c' scripts/host-baselines.tsv | grep -c . || true)"
  SHIPPED_W="$(awk -F'\t' -v c="$HCPU" '!/^#/ && $1 != "cpu_model" && $1 == c' scripts/judged-runs.tsv | grep -c . || true)"
  say "   ok    shipped scripts/ carries $SHIPPED_B baseline row(s) and $SHIPPED_W witness row(s) for this CPU;"
  say "         the substitution starts from headers only, so pass 1 is the 'new' state whatever those say"

  # Read the judged routine list out of the gate too. The counts read back below are
  # per-routine, and typing 3 here would make this driver disagree with the gate the
  # first time a routine joins the class.
  JUDGED="$(sed -n 's/^P5_JUDGED="\(.*\)"$/\1/p' scripts/gate-p5.sh | head -1)"
  MARGIN="$(sed -n 's/^BASELINE_MARGIN=\([0-9.]*\)$/\1/p' scripts/gate-p5.sh | head -1)"
  MEASURED="$(sed -n 's/^P5_MEASURED="\(.*\)"$/\1/p' scripts/gate-p5.sh | head -1)"
  SMARGIN="$(sed -n 's/^STRSM_MARGIN=\([0-9.]*\)$/\1/p' scripts/gate-p5.sh | head -1)"
  if [[ -z "$JUDGED" || -z "$MARGIN" || -z "$MEASURED" || -z "$SMARGIN" ]]; then
    refuse "exercise-baseline: cannot read P5_JUDGED ('$JUDGED'), BASELINE_MARGIN ('$MARGIN')," \
           "  P5_MEASURED ('$MEASURED') or STRSM_MARGIN ('$SMARGIN') out of scripts/gate-p5.sh," \
           "  so the counts and bars below have nothing to check for at least one criterion."
  fi
  NJ="$(printf '%s\n' $JUDGED | grep -c . || true)"
  NM="$(printf '%s\n' $MEASURED | grep -c . || true)"
  # The criterion KEYS, built once here in the gate's own two spellings: every row this
  # driver lands or looks up is keyed by one of these, and two spellings typed at four sites
  # is how the decoy ends up under a key no criterion reads.
  CRITS=""
  for r in $JUDGED; do CRITS="$CRITS share/$r"; done
  for r in $MEASURED; do CRITS="$CRITS scale/$r"; done
  say "   ok    share criterion: $JUDGED ($NJ) at margin $MARGIN points; ratio criterion: $MEASURED ($NM) at margin ${SMARGIN}x"
  say "         -- all four read from the gate, and the margins are DIFFERENT UNITS, which is why neither is reused for the other (#119)"
  say ""
}

# ------------------------------------------------------- the substituted rows
#
# Headers only, and copied from the shipped files rather than typed: a header this
# driver invented would be a second statement of the schema, and the readers key on
# column ordinals that already moved once when the era column landed.
build_substitution() {
  rm -rf "$DIR"
  mkdir -p "$DIR"
  for f in host-baselines.tsv judged-runs.tsv; do
    awk 'BEGIN{done=0} /^#/ {print; next} !done {print; done=1}' "scripts/$f" >"$DIR/$f"
    if [[ -n "$(awk -F'\t' '!/^#/ && $1 != "cpu_model" && $1 != "era" && NF > 1' "$DIR/$f")" ]]; then
      refuse "exercise-baseline: $DIR/$f kept a data row, so pass 1 is not the empty state."
    fi
  done
  # The era ledger is copied WHOLE: it is the scope, not a state being varied, and an
  # era ledger with its rows stripped resolves to nothing and fails every reading.
  cp scripts/measurement-eras.tsv "$DIR/measurement-eras.tsv"
  say "-- substituted artifacts under $DIR --"
  say "   registry: headers only ($(grep -c '' "$DIR/host-baselines.tsv") lines)"
  say "   witness:  headers only ($(grep -c '' "$DIR/judged-runs.tsv") lines)"
  say "   eras:     copied whole ($(grep -c '' "$DIR/measurement-eras.tsv") lines), so $ERA resolves as it ships"
}

# ------------------------------------------------------------------ one pass
run_pass() {
  local n="$1" reason="$2"
  local plog="$DIR/pass$n.log"
  # A per-pass RUN_STAMP, EXPORTED so the candidate files land under a name this driver set
  # rather than one it guessed. It replaces an `rm -f` of the previous pass's files: the gate
  # is run-stamped now, so passes cannot pile into one file, and the workaround that used to
  # hide that from this harness alone is gone with it.
  local stamp="exercise-pass$n" f kind
  set +e
  RUN_STAMP="$stamp" \
  KEEL_REMOTE_HOSTS="$HOST" \
  KEEL_INSTRUMENT_EXERCISE="$reason" \
  KEEL_INSTRUMENT_BASELINE_DIR="$DIR" \
    bash scripts/gate-p5.sh >"$plog" 2>&1
  PASSRC=$?
  set -e
  strip "$plog" >"$DIR/pass$n.txt"
  cat "$DIR/pass$n.txt" >>"$LOG"
  # Destination names stay unstamped: four sites below read pass$n-<kind>-candidates-$REV.tsv.
  for kind in baseline witness; do
    f="build/$kind-candidates-$REV-$stamp.tsv"
    if [[ -e "$f" ]]; then cp "$f" "$DIR/pass$n-$kind-candidates-$REV.tsv"; fi
  done
  # The delegated chain is collected per pass by asking each log which log it names, down
  # two levels: gate-p5 names gate-p4's, gate-p4 names gate-p3's. Two levels because the
  # stamp has to survive two delegations, and surviving them is exactly what the `export`
  # 804fb75 lifted is for. The paths are READ rather than reconstructed -- three passes at
  # one rev used to overwrite one file until RUN_STAMP landed, so a driver that rebuilt the
  # name would be asserting it still knows a naming rule that had just changed.
  local src="$DIR/pass$n.txt" d
  DCOLLECTED=0
  while :; do
    d="$(dlogs "$src")"
    [[ -n "$d" && -e "$d" ]] || break
    cp "$d" "$DIR/pass$n-$(basename "$d")"
    DCOLLECTED=$((DCOLLECTED + 1))
    src="$d"
  done
  if [[ "$DCOLLECTED" -eq 0 ]]; then
    say "   pass $n names no delegated log, so the stamp audit below covers this pass's parent"
    say "   log ONLY -- and it says so there too, since a tally is read where it is printed"
  fi
  say "   pass $n exited $PASSRC (2 is the withheld-verdict exit), full output $plog"
  if [[ "$PASSRC" -ne 2 ]]; then
    refuse "exercise-baseline: pass $n exited $PASSRC, not 2. A synthetic run that reaches a" \
           "  signable verdict is the failure mode the stamp exists to prevent; read $plog" \
           "  before running another pass."
  fi
}

# count PATTERN N -- occurrences of a verdict phrase in pass N's stripped log.
count() { grep -cF "$1" "$DIR/pass$2.txt" || true; }

# preempted CRIT N -- did rule 19 out-resolve CRIT's reading on pass N, before it reached the
# bar block? Rule 19 sits AHEAD of bar selection for BOTH criteria, and noise is the likely case
# rather than the exotic one: 9 of this era's 15 archived Strsm rows are wider than STRSM_MARGIN,
# and on antares two of the three SHARE rows were (Ssyrk's 8-thread interval is +/-33.6%).
#
# Keyed per routine to that criterion's OWN refusal sentence: each long key matches exactly one
# of the gate's two refusal sentences where the shared prefix `NOISE-LIMITED, NOT JUDGED` matches
# both (measured against gate-p5.sh's bytes, 1/1/2). The short key would not mis-fire TODAY, and
# not because of anything here -- P5_JUDGED and P5_MEASURED are disjoint, so no routine can own a
# line on both sentences. The tail is what makes that independent of a list this driver reads at
# runtime and does not control.
preempted() {
  local k
  case "$1" in
    share/*) k='NOISE-LIMITED, NOT JUDGED: the intervals cost this share' ;;
    *)       k='NOT ELIGIBLE TO TYPE A FLOOR' ;;
  esac
  grep -F "] ${1#*/} " "$DIR/pass$2.txt" | grep -qF "$k"
}

# want_n KIND N -- how many of KIND's verdict lines pass N could possibly have rendered: one per
# routine of that criterion, less the ones rule 19 out-resolved above. Both criteria, because
# hardcoding $NJ for the share one made a correct gate read as a broken one on the first host
# whose readings were noisy (found live 2026-08-30, and it would have failed all three passes).
#
# PER PASS AND NOT LATCHED, which is the direction that can be wrong in only one way: each
# pass is its own measurement, so an arm too noisy to judge on one sweep can be quiet enough
# on the next, and an expectation carried forward would read a working arm as a broken one.
# Rule 19's refusal is the ONLY excuse admitted -- any other reason a line is missing leaves
# the expectation up and reads as a NO, which is where an unmeasured arm belongs.
want_n() {
  local n=0 c
  for c in $CRITS; do
    [[ "$c" == "$1/"* ]] || continue
    preempted "$c" "$2" || n=$((n + 1))
  done
  printf '%s\n' "$n"
}

# dlogs LOG -- the delegated log path LOG names, if any. Keyed to the phrase the gates
# print and tolerant of ANY prefix: instrument_exercise stamps info lines too, so
# `[synthetic] ` sits between the indent and the phrase, and an anchored `^ *full output:`
# matched nothing on all three passes (found live, 2026-08-22). It failed closed -- the
# driver said the delegate was uncollected instead of auditing files it never read -- which
# is why that is a finding and not a false green. `sort -u` because gate-p4 names gate-p3's
# log twice, in the announcement and again in the exit-code note.
dlogs() {
  sed -n 's/^.*full output: \(build\/gate-p[0-9]-under-p[0-9]-[^ ]*\.log\).*/\1/p' "$1" |
    sort -u | tail -1
}

# --------------------------------------------------------------- read-backs
#
# Each outcome is keyed to its OWN phrase and reports NO on no match, which is the
# fail-closed direction: a gate whose wording changes without this driver being updated
# must say the branch did not fire, never mistake a neighbouring branch for the target.
readback_new() {
  local nb ns nr tally cand wit wantc
  nb="$(count 'RECORDED as its candidate baseline rather than judged (#6)' 1)"
  ns="$(count 'RECORDED as its candidate baseline rather than judged (#119)' 1)"
  nr="$(count "its absence here is this host's admission date" 1)"
  tally="$(grep -F 'rendered BASELINE this run in era' "$DIR/pass1.txt" | tail -1 || true)"
  cand="$(awk -F'\t' '!/^#/ && NF >= 8' "$DIR/pass1-baseline-candidates-$REV.tsv" 2>/dev/null | grep -c . || true)"
  wit="$(awk -F'\t' '!/^#/ && NF >= 6' "$DIR/pass1-witness-candidates-$REV.tsv" 2>/dev/null | grep -c . || true)"
  NWANT="$(want_n share 1)"
  SWANT="$(want_n scale 1)"
  wantc=$((NWANT + SWANT))
  say "   share criterion, BASELINE lines:  $nb (expected $NWANT of $NJ judged routine(s))"
  say "   ratio criterion, BASELINE lines:  $ns (expected $SWANT of $NM)"
  say "   README criterion, BASELINE lines: $nr (expected 1)"
  say "   candidate rows: $cand baseline (expected $wantc), $wit witness"
  say "   fleet tally:    ${tally:-none printed}"
  # Named per criterion, and a limit on the whole run rather than on this pass: pass 3 judges
  # against a row this pass proposes, so an arm rule 19 out-resolved here cannot be registered
  # there however quiet pass 3 turns out to be. Pass 3 reads that off the candidates file
  # directly instead of a latch -- the proposal's absence is the fact, and the file holds it.
  for c in $CRITS; do
    if preempted "$c" 1; then say "   DISCLOSED, unexercised for the whole run: rule 19 out-resolved $c"; fi
  done
  if [[ "$wantc" -lt $((NJ + NM)) ]]; then
    say "   -- and it is a limit on the whole run, not on this pass: those intervals are wider than"
    say "   the margin their own bar was set under, so no candidate baseline exists for them and"
    say "   pass 3 cannot register them however quiet pass 3 turns out to be. A quieter host or"
    say "   more samples is what would exercise those arms -- not a change to this driver."
  fi
  # An expectation computed from the run can reach zero, and zero-equals-zero is a green over
  # nothing measured -- the failure the whole apparatus exists to refuse (DESIGN.md §5.6). It is
  # not the same finding as a wrong count, so it is not the same message: a host too noisy on
  # EVERY criterion cannot exercise this class, which is a fact about the host.
  if [[ "$wantc" -eq 0 ]]; then
    refuse "exercise-baseline: rule 19 out-resolved every criterion on pass 1, so nothing" \
           "  reached the class and there is no rendering to check. Unmeasured, not clean:" \
           "  point this exercise at a quieter host, or raise the sample count."
  fi
  if [[ "$nb" -eq "$NWANT" && "$ns" -eq "$SWANT" && "$nr" -eq 1 &&
        "$cand" -eq "$wantc" && "$wit" -eq 1 ]]; then
    say "   YES for the 'new' state: every criterion of the class that could be reached"
    say "   rendered BASELINE on a host with no registry row and no witness row, and the gate"
    say "   proposed exactly the rows a reviewed commit would land -- $wantc baselines across"
    say "   both criteria and one witness, once per host and not once per routine."
  else
    say "   NO for the 'new' state: the counts above are not the class's empty-registry"
    say "   rendering, so passes 2 and 3 rest on nothing. Read $DIR/pass1.log."
    refuse "exercise-baseline: pass 1 did not render the 'new' state; stopping rather than" \
           "  spending two more sweeps on states built out of it."
  fi
}

readback_owing() {
  local ns nx nr debt renewed n2 w2
  ns="$(count 'BASELINE is spent (#6). Land the candidate row' 2)"
  nx="$(count 'BASELINE is spent (#119). Land the candidate row' 2)"
  nr="$(count 'so its numbers are unpublished rather than unborn' 2)"
  debt="$(grep -F 'hosts owing registration:' "$DIR/pass2.txt" | tail -1 || true)"
  # BOTH criteria's phrases, because a renewal on either is #114's defect and the substring
  # they used to share would have counted one criterion's renewal against the other's silence.
  renewed="$(count 'RECORDED as its candidate baseline rather than judged (#6)' 2)"
  renewed=$((renewed + $(count 'RECORDED as its candidate baseline rather than judged (#119)' 2) ))
  n2="$(want_n share 2)"
  w2="$(want_n scale 2)"
  say "   share criterion, spent FAILs:  $ns (expected $n2 of $NJ)"
  say "   ratio criterion, spent FAILs:  $nx (expected $w2 of $NM)"
  say "   README criterion, spent FAILs: $nr (expected 1)"
  say "   BASELINE renewals:             $renewed (expected 0 -- a renewal here is #114's defect)"
  say "   debt line:                     ${debt:-none printed}"
  if [[ "$ns" -eq "$n2" && "$nx" -eq "$w2" && "$nr" -eq 1 &&
        "$renewed" -eq 0 && -n "$debt" ]]; then
    say "   YES for the 'owing' state: one landed witness row and no registry row converts"
    say "   the same absence pass 1 read as newness into an unmet obligation, on every"
    say "   criterion that reached the class, and the debt is printed rather than absorbed."
  else
    say "   NO for the 'owing' state. A nonzero renewal count is the specific failure #114"
    say "   named: a witness the gate cannot see means BASELINE never spends."
    refuse "exercise-baseline: pass 2 did not render the 'owing' state; stopping."
  fi
}

readback_registered() {
  local line bar bval bera bad=0 seen=0 elig=0 c r want u m fmt verb
  say "   the decoy: a free-placement row at 99.0 sits ABOVE the real rows, so a bar of"
  say "   99.0 less that criterion's own margin, or an era of free-placement in any line"
  say "   below, refutes era scoping."
  for c in $CRITS; do
    r="${c#*/}"
    # Each criterion is read in ITS OWN units, margin, verb and precision, all four taken from
    # the gate: `%`/2.6/`reaches`/%.1f for the share criterion, `x`/0.403/`scales`/%.3f for the
    # ratio one (#119). A single set carried across both is #110's units error, and here it
    # would not even find a line to be wrong about.
    case "$c" in
      share/*) u='%'; m="$MARGIN"; verb='reaches'; fmt='%.1f' ;;
      *)       u='x'; m="$SMARGIN"; verb='scales';  fmt='%.3f' ;;
    esac
    want="$(awk -F'\t' -v k="$c" '!/^#/ && $2 == k {print $4; exit}' "$DIR/pass1-baseline-candidates-$REV.tsv")"
    # Lifted out of the ratio arm (2026-08-30): rule 19 out-resolves SHARE rows too, two of three
    # on antares, and this loop's only other answer for a criterion with nothing registered is
    # "NO line naming a registered baseline at all" -- a red for a gate that behaved correctly.
    # WHICH of the two, not either: the driver knows, and "or" in a skip line is a reader's dead
    # end. They are different facts about the fleet -- one says the reference was never proposed,
    # the other says this sweep was too noisy. The empty `want` is the first fact itself rather
    # than a latch's memory of it: pass 1's proposal is either in that file or it is not.
    if [[ -z "$want" ]]; then
      say "   $c: SKIPPED and unexercised -- pass 1 proposed no candidate baseline for it, so"
      say "     there is nothing registered to judge against here. Stated, not counted as clean."
      continue
    fi
    if preempted "$c" 3; then
      say "   $c: SKIPPED and unexercised -- a row WAS registered from pass 1, and rule 19"
      say "     out-resolved THIS pass's reading before it reached the bar. Stated, not clean."
      continue
    fi
    elig=$((elig + 1))
    line="$(grep -F "] $r $verb" "$DIR/pass3.txt" | grep -F 'registered baseline' | tail -1 || true)"
    if [[ -z "$line" ]]; then
      say "   $c: NO line naming a registered baseline at all"
      bad=1; continue
    fi
    seen=$((seen + 1))
    bar="$(sed -n "s/.*(>= \([0-9.]*\)$u, this host.s registered baseline.*/\1/p;s/.*(< \([0-9.]*\)$u, this host.s registered baseline.*/\1/p" <<<"$line" | head -1)"
    bval="$(sed -n "s/.*registered baseline \([0-9.]*\)$u (era \([^)]*\)).*/\1/p" <<<"$line" | head -1)"
    bera="$(sed -n "s/.*registered baseline \([0-9.]*\)$u (era \([^)]*\)).*/\2/p" <<<"$line" | head -1)"
    local expect
    expect="$(awk -v b="$want" -v m="$m" -v f="$fmt" 'BEGIN{printf f, b-m}')"
    if [[ "$bera" != "$ERA" ]]; then
      say "   $c: era scoping REFUTED -- the bar came from era '$bera', not '$ERA'"
      bad=1
    elif [[ "$bval" != "$want" ]]; then
      say "   $c: the bar rests on baseline $bval$u where the landed row says $want$u"
      bad=1
    elif [[ "$bar" != "$expect" ]]; then
      say "   $c: bar $bar$u where $want - $m = $expect$u"
      bad=1
    else
      say "   $c: judged at $bar$u = $want$u - $m, era $bera (the decoy's 99.0 was not consulted)"
    fi
  done
  if [[ "$bad" -eq 0 && "$seen" -eq "$elig" && "$elig" -gt 0 ]]; then
    say "   YES for the 'registered' state, and era scoping holds in both directions: the"
    say "   in-era row governed every criterion that reached the class ($seen of $((NJ + NM)))"
    say "   and the wrong-era row above it was not read, on the same pass and one lookup."
  else
    # TWO CAUSES, NAMED SEPARATELY. This paragraph used to tell the era story alone, which was
    # the only way to fail it while one criterion had one margin. A wrong margin -- 2.6
    # subtracted from a ratio, the units error #119 refused by rule -- now fails it too, and
    # the per-criterion lines above say which happened. One verdict standing for two causes is
    # what DESIGN.md §5.6 forbids, and a report of it is no exception.
    say "   NO for the 'registered' state, and the lines above say which of two things went"
    say "   wrong. An era of free-placement is the finding that outranks the rest of this run:"
    say "   a baseline from the retired instrument judging pinned readings is the exact"
    say "   misattribution the era boundary exists to prevent. A bar that is neither the"
    say "   landed value nor that value less THAT criterion's own margin is the other: the"
    say "   share criterion is in points and the ratio criterion is in x, and a margin carried"
    say "   between them is #110's units error arriving one criterion over (#119)."
  fi
}

# ------------------------------------------------------------ landing the rows
#
# Landing means writing into the SUBSTITUTED files under build/, never scripts/: a
# reviewed commit is the only thing that lands a real row, and a dirty tree would stop
# the next pass's delegated chain anyway. The rows are pass 1's candidates verbatim,
# including the estimator column that says a single draw is not landable as-is (§5 rule
# 16) -- which is true of the real campaign and is exactly what a reader of this log
# should see attached to the bar pass 3 is judged against.
land_witness() {
  awk -F'\t' '!/^#/ && NF >= 6' "$DIR/pass1-witness-candidates-$REV.tsv" >>"$DIR/judged-runs.tsv"
  say "   landed into $DIR/judged-runs.tsv: $(awk -F'\t' '!/^#/ && $1 != "cpu_model" && NF >= 6' "$DIR/judged-runs.tsv" | grep -c . || true) witness row(s)"
}

land_registry() {
  # THE DECOY GOES FIRST, and the order is the whole test: baseline_lookup prints the
  # first matching row and stops, so a wrong-era row below the right one would be
  # skipped whether or not the era column is honoured. Above it, only era matching keeps
  # it out.
  local c
  # One decoy per criterion KEY, from $CRITS: a decoy written only under `share/` would leave
  # the ratio criterion's lookup with nothing above its real row, so the era test would not
  # cover the criterion #119 just added -- and the log would still say era scoping held.
  for c in $CRITS; do
    printf '%s\t%s\tfree-placement\t99.0\tSYNTHETIC DECOY -- a wrong-era row this run must not consult\t—\t%s\tsynthetic: era-scoping negative control (scripts/exercise-baseline.sh)\n' \
      "$HCPU" "$c" "$(date -u +%Y-%m-%d)" >>"$DIR/host-baselines.tsv"
  done
  awk -F'\t' '!/^#/ && NF >= 8' "$DIR/pass1-baseline-candidates-$REV.tsv" >>"$DIR/host-baselines.tsv"
  say "   landed into $DIR/host-baselines.tsv: $((NJ + NM)) decoy row(s) at era free-placement, then"
  say "   $(awk -F'\t' -v e="$ERA" '!/^#/ && $3 == e' "$DIR/host-baselines.tsv" | grep -c . || true) real row(s) at era $ERA, in that order"
}

# ----------------------------------------------------------------- the stamp
#
# Trust nothing about the labelling either. Every verdict line in every pass must carry
# the stamp, and the pattern is the tight one: `^  TOKEN  `, so a line of driver prose
# that happens to contain the word cannot be counted as a verdict (the failure recorded
# in exercise-dead-host.sh). The delegated logs are audited too -- that is what the
# missing `export` cost, and an unstamped child is the whole reason it was lifted.
#
# THE TALLY NAMES ITS OWN DOMAIN, which is repair and not decoration. On this driver's
# first firing no delegated log was collected (see dlogs), so this audit read three parent
# logs, totalled 129, and concluded "all 129 verdict lines carry [synthetic] ... parent or
# delegate". The disclosure existed -- three screens earlier, once per pass -- and the
# summary line contradicted it. So: the file count is printed with the total, an expected
# count is asserted, and an uncollected delegate is a NO rather than a footnote. A checker
# is silent about what it never parsed.
stamp_audit() {
  local n total=0 unstamped=0 f files=0
  say ""
  say "-- stamp audit: every verdict line of every log this run collected --"
  for f in "$DIR"/pass[123].txt "$DIR"/pass[123]-gate-p[0-9]-under-p[0-9]-*.log; do
    [[ -e "$f" ]] || continue
    files=$((files + 1))
    local raw
    raw="$(strip "$f")"
    n="$(grep -cE '^  (PASS|FAIL|UNMEASURED|BASELINE|REPORTED)  ' <<<"$raw" || true)"
    local u
    u="$(grep -E '^  (PASS|FAIL|UNMEASURED|BASELINE|REPORTED)  ' <<<"$raw" | grep -vcF '[synthetic] ' || true)"
    total=$((total + n)); unstamped=$((unstamped + u))
    say "   $(basename "$f"): $n verdict line(s), $u unstamped"
  done
  local signed
  signed="$(grep -hE '^gate-p[0-9]: (GREEN|RED)' "$DIR"/pass[123].txt "$DIR"/pass[123]-gate-p[0-9]-under-p[0-9]-*.log 2>/dev/null | grep -c . || true)"
  say "   signed verdict lines across every log: $signed (expected 0)"
  # 9 = three passes x (parent + gate-p4 + gate-p3). Asserted rather than assumed, because
  # every way this audit can be wrong is a way of reading fewer files than it claims.
  local want=$((3 * 3))
  if [[ "$files" -ne "$want" ]]; then
    say "   NO on coverage: $files log(s) audited where the three-pass chain is $want (parent,"
    say "   gate-p4, gate-p3 per pass). The stamp is unaudited wherever a log is missing, and"
    say "   an unstamped delegate is precisely what the lifted \`export\` exists to prevent, so"
    say "   this reads as no exercise of the stamp rather than as $total clean lines."
  elif [[ "$unstamped" -eq 0 && "$signed" -eq 0 && "$total" -gt 0 ]]; then
    say "   YES: all $total verdict lines across $files log(s) carry [synthetic] and nothing"
    say "   signed a verdict -- three passes, each of parent, gate-p4 and gate-p3."
  else
    say "   NO: $unstamped of $total verdict lines are unstamped and $signed verdict lines"
    say "   were signed. A synthetic log that reads as a gate result is worse than no"
    say "   exercise; quarantine $DIR before anyone reads it as evidence."
  fi
}

main "$@"
