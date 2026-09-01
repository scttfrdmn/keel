#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# ab.sh — the two-build A/B harness behind scripts/ab-bench.sh.
#
# Sourced, never executed, from a main() that has already cd'd to the repository
# root; it sources remote.sh and bench.sh itself. Its callers are NOT gates: they
# certify nothing, change no criterion, and exit 0 whatever they measure. Their
# product is a table, per host, of the same benchmarks against two builds.
#
# Deliberately its own file rather than more of remote.sh, which every gate
# sources: an A/B driver's machinery has no business inside the closure a release
# certificate has to transfer across.
#
# Lifted from ~160 lines the two drivers of the day held in common (#131). They
# were identical for the entire skeleton — worktree, trap, cross-compile,
# methodology preamble, host loop, comparison — and differed in four strings, so
# each of those is a parameter and none of them is a mode flag: nothing below asks
# which caller it is serving. A caller sets these, then calls `ab_run BASE_REF`:
#
#   AB_TITLE      the banner line
#   AB_FILTER     the -test.bench pattern
#   AB_TAG        archive prefix under build/
#   AB_NOTES      optional array of methodology lines, after the shared ones
#   AB_BASE_HINT  optional extra stderr text when the BASE arm fails to build
#
# Everything here is a function definition or a source, and each caller is one
# function plus `main "$@"`. Bash reads a script incrementally as it executes it,
# so a script doing its work at the top level can be corrupted by an edit that
# lands mid-run: the parser resumes at a byte offset that now holds different
# text. Defining everything before anything runs forces one whole-file parse
# before the first host is touched, which makes the instrument immune instead of
# leaving "never edit a running instrument" as a rule someone has to remember
# (#51).
#
# The base build comes from a detached git worktree at BASE_REF, not from
# stashing: a stash would mutate the tree another long-running measurement may be
# reading, and it would make the two arms differ by whatever else was dirty. The
# new build comes from the working tree, so a candidate can be measured before it
# is committed — which also means BASE_REF must already contain the fixture.
#
# The comparison goes through bench_compare rather than calling benchstat
# directly. The l1 measurement's first run is why that function exists: benchstat groups
# by configuration, keel's provenance preamble is in that namespace, and one
# volatile key in it (keel-bench-clock-mhz, a live snapshot) silently turned every
# A/B into two independent one-column tables with no delta in them — #50,
# docs/toolchain-notes.md T20. The output looked like a comparison. A comparison
# that did not happen is now a failed measurement and says so.
#
# GOMAXPROCS=1, as every single-thread measurement in this repo does.

# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

# ab_host HOST — both arms on one host, then the comparison.
#
# Both arms run back to back on the same machine before anything is compared, so
# the two builds see the same thermal and residency state as nearly as this setup
# can arrange. A host whose first arm fails reports nothing at all rather than
# half a table.
ab_host() {
  local host="$1" arm bin
  for arm in base new; do
    bin="$BASE_BIN"
    [[ "$arm" == new ]] && bin="$NEW_BIN"
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$bin" "${BFLAGS[@]}" \
         -test.bench="$AB_FILTER" >"$BINDIR/$arm.log" 2>&1; then
      warn "[$host] the $arm run failed; nothing is reported for this host"
      sed 's/^/        /' "$BINDIR/$arm.log" | tail -20
      return 0
    fi
  done
  # benchstat labels the columns from the file names, so the temp files carry the
  # arm AND its commit rather than being passed as base.txt/new.txt. The arm is in
  # the name because the SHAs are not always distinct: a null A/B (BASE_REF=HEAD,
  # the drift preset) resolves both to one short SHA, and naming by SHA alone made
  # the second `cp` overwrite the first, so benchstat was handed one file twice and
  # printed a wash by construction. The build/ copies are the durable ones: a
  # ranking is recomputed from the logs rather than from a script's stdout
  # (DESIGN.md §5.8).
  cp "$BINDIR/base.log" "$BINDIR/base-$BASE_SHA.txt"
  cp "$BINDIR/new.log" "$BINDIR/new-$NEW_SHA.txt"
  cp "$BINDIR/base.log" "build/$AB_TAG-$host-base-$BASE_SHA.log"
  cp "$BINDIR/new.log" "build/$AB_TAG-$host-new-$NEW_SHA.log"
  # The caller's `set -o pipefail` is what makes this test bench_compare's verdict
  # rather than sed's: without it the indent pipe would swallow the one status that
  # says whether a comparison happened.
  if ! bench_compare "$BINDIR/base-$BASE_SHA.txt" "$BINDIR/new-$NEW_SHA.txt" | sed 's/^/        /'; then
    warn "[$host] the two arms were not compared, so this host contributes no delta"
  fi
}

# ab_run BASE_REF — build both arms, then measure every configured host.
ab_run() {
  local BASE_REF="$1" DIRTY="" note host

  BINDIR="$(mktemp -d)"
  # Deliberately not `local`: the EXIT trap below runs in global scope, after this
  # function has returned, so a local would be unbound there — and under `set -u`
  # that aborts the trap on its first command, leaking both the worktree and
  # BINDIR while exiting 1 on a successful run (#55).
  WORKTREE="$BINDIR/base"
  # The worktree is removed as well as deleted: leaving it registered would make
  # `git worktree list` grow one stale entry per run. Both paths are expanded here,
  # at definition time, rather than left to the trap's own scope.
  trap "git worktree remove --force '$WORKTREE' >/dev/null 2>&1 || true; rm -rf '$BINDIR'" EXIT

  BASE_BIN="$BINDIR/base.test"
  NEW_BIN="$BINDIR/new.test"
  mkdir -p build

  echo "$AB_TITLE"
  echo

  BASE_SHA="$(git rev-parse --short "$BASE_REF")"
  NEW_SHA="$(git rev-parse --short HEAD)"
  git diff --quiet HEAD -- || DIRTY=" + uncommitted changes"
  echo "base: $BASE_SHA ($BASE_REF)"
  echo "new:  $NEW_SHA (working tree$DIRTY)"

  remote_require_hosts

  git worktree add --detach "$WORKTREE" "$BASE_REF" >/dev/null
  if ! ( cd "$WORKTREE" && remote_build_test ./bench "$BASE_BIN" ) >"$BINDIR/build" 2>&1; then
    echo "cross-compile of the base bench binary failed:" >&2
    sed 's/^/  /' "$BINDIR/build" >&2
    [[ -n "${AB_BASE_HINT:-}" ]] && echo "$AB_BASE_HINT" >&2
    exit 2
  fi
  if ! remote_build_test ./bench "$NEW_BIN" >"$BINDIR/build" 2>&1; then
    echo "cross-compile of the new bench binary failed:" >&2
    sed 's/^/  /' "$BINDIR/build" >&2
    exit 2
  fi
  echo "built two linux/amd64 bench binaries (static)"

  mapfile -t BFLAGS < <(bench_flags)
  echo "methodology: -count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME, GOMAXPROCS=1, filter=$AB_FILTER"
  info "benchstat compares the two logs, ignoring $KEEL_BENCH_IGNORE — configuration"
  info "keys that describe the run rather than the build, and which forked the table"
  info "into two delta-free halves when they were not ignored (#50, T20)."
  for note in "${AB_NOTES[@]:-}"; do [[ -n "$note" ]] && info "$note"; done
  echo

  while read -r host; do
    [[ -n "$host" ]] || continue
    remote_host_header "$host" || continue
    ab_host "$host"
    echo
  done <<<"$HOSTS"
}
