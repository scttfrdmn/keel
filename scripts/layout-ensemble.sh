#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# layout-ensemble.sh — decide whether an A/B delta is caused by the code change
# or by where the change happened to put the code.
#
# Not a gate: this certifies nothing. It is the instrument #61/T22 asks for.
#
# # Why this exists
#
# T22 measured a +45.06% regression on Zen 5 in a function whose disassembly is
# byte-identical across the two builds. The cause was placement: a function two
# positions earlier grew by 160 bytes, which is a multiple of amd64's 32-byte
# function alignment but not of 64, so every later entry flipped from
# 64-byte-aligned to 64+32. Upstream that is golang/go#8717.
#
# The consequence is that statistical resolution is not attribution. That 45%
# arrived with both arms at +-0-1% and p=0.000, so "the delta is larger than its
# confidence interval" proves only that *something* differs between two
# binaries, and layout is something.
#
# The discriminator follows from the mechanism rather than from more samples:
#
#   - a placement artifact is a property of the binary pair. Move the layout and
#     it moves; its sign follows the binary, not the source change.
#   - a mechanism delta is a property of the code change. It survives a layout
#     perturbation; its sign follows the semantics.
#
# So this script builds an *ensemble*: the same two arms at several benign
# layout perturbations, and reports whether the delta is sign-consistent across
# the ensemble and larger than the ensemble's own layout spread.
#
# # How the perturbation works
#
# A generated file in internal/l1 defines an uncalled-but-not-eliminable
# function of a chosen size, sorted ahead of l1_amd64.go so the kernels move
# down by roughly its length. This is navytux's zzz() trick from
# golang/go#18977, which is where the technique is from.
#
# Two properties matter and both are checked rather than assumed:
#
#   - It must not change semantics. The pad is called only under a condition
#     that is false at init time, so it never executes; nothing else references
#     it.
#   - It must actually move the kernels. Go's linker eliminates dead code and
#     makes no promise about function order, so the script reads back each
#     kernel's entry address with `go tool objdump` and prints the alignment it
#     actually sampled. A perturbation that failed to move anything is reported
#     as such instead of silently padding the ensemble with duplicates.
#
# # Controls precede the subject in link order
#
# The protection is directional. At a fixed pad both arms put the *changed*
# function at the same address only because everything ahead of it in link order
# is unchanged; the change's own size delta displaces everything after it. On
# the e829a61 run the subject shrank 32 bytes and every routine downstream of it
# moved by 0x20 -- a multiple of 32 but not 64, so the mod-64 class flipped for
# all of them. That is exactly T22's mechanism, reproduced by the very commit
# being judged.
#
# So a routine downstream of the subject is placement-suspect between the arms
# no matter how clean its numbers look, and this is a rule of the instrument
# rather than a lucky property of one run. The script enforces it by grading:
# any measured routine whose entry address differs between the arms is printed
# with a `placement-confounded` label and excluded from the verdict set by
# construction. A warning would leave it citable; a demotion does not.
#
# # What it cannot do
#
# It samples a handful of placements, not the distribution. A delta that clears
# this bar is not proven to be mechanism -- it is proven not to be *these* four
# placements' worth of layout. That is a weaker claim than it looks and is the
# strongest one available without the alignment control golang/go#6752 would
# give.
#
# One confound is irreducible and is not worth chasing: the subject's own
# *interior* alignment moves with its own code change, since a function that
# emits fewer instructions lays its hot loop differently against the 64-byte
# lines. That is part of the change, not a confound to be separated from it.
# Perfect separation of a function from its own layout does not exist.
#
# An instruction-count delta bounds the code change, not its throughput
# consequence: reasoning from "3 fewer instructions in 25" to a percentage
# assumes removed instructions are fungible with the remainder, which holds only
# under uniform issue pressure. Report the count and the time as two facts.
#
# usage: scripts/layout-ensemble.sh BASE_REF [PAD_STEPS...]
#        default PAD_STEPS: 0 3 6 9   (statement counts, not byte counts)

set -euo pipefail

info() { printf '        %s\n' "$1"; }
warn() { printf '  \033[33mWARN\033[0m  %s\n' "$1"; }

# The kernel whose code the change under test touches, and a control whose code
# it must leave byte-identical. Both are read back per build.
CHANGED_FN='l1\.avx512Asum$'
CONTROL_FN='l1\.avx512Scal$'

# MEASURED — benchmark-name token -> the symbol that runs it. Grading needs this
# map because benchstat reports benchmark names while placement is a property of
# symbols. Only the avx512 symbols are listed: all three hosts dispatch there
# (gate-p5's matrix), so a KEEL_FORCE arm would need its own map and would get no
# grading from this one. A token absent from the map is treated as ungraded, and
# ungraded is demoted rather than cleared.
MEASURED=(
  'L1Sasum:l1\.avx512Asum$'
  'L1Sscal:l1\.avx512Scal$'
  'L1Sdot:l1\.avx512Dot$'
  'L1Saxpy:l1\.avx512Axpy$'
  'L1Snrm2:l1\.avx512SumSq$'
)

# write_pad DIR N — put an N-statement pad in DIR/internal/l1, or remove it for N=0.
#
# The file sorts before l1_amd64.go so the kernels land after it. `aa_` rather
# than `a_` because a single leading letter collides with nothing today but is
# the sort key most likely to acquire a neighbour later.
write_pad() {
  local dir="$1" n="$2" f="$1/internal/l1/aa_layoutpad_amd64.go" i
  if [[ "$n" == 0 ]]; then
    rm -f "$f"
    return
  fi
  {
    echo '// Copyright 2026 Scott Friedman'
    echo '// SPDX-License-Identifier: Apache-2.0'
    echo
    echo '//go:build goexperiment.simd && amd64'
    echo
    echo 'package l1'
    echo
    echo '// Generated by scripts/layout-ensemble.sh. Never commit this file: it exists'
    echo '// only to move the kernels below it in the object, and it is deleted when the'
    echo '// ensemble finishes. See that script for why.'
    echo
    echo 'var layoutPadSink int'
    echo
    echo '//go:noinline'
    echo 'func layoutPad(n int) int {'
    for ((i = 0; i < n; i++)); do
      echo "	layoutPadSink += n * $((i + 1))"
    done
    echo '	return layoutPadSink'
    echo '}'
    echo
    echo '// layoutPadSink is zero at init, so layoutPad never runs. The reference is'
    echo '// what keeps the linker from eliminating it; the guard is what keeps it from'
    echo '// having any effect if it somehow did.'
    echo 'func init() {'
    echo '	if layoutPadSink != 0 {'
    echo '		layoutPadSink = layoutPad(1)'
    echo '	}'
    echo '}'
  } >"$f"
}

# align_of BIN SYM — entry address of SYM in BIN, and that address mod 64.
align_of() {
  local a
  a="$(go tool objdump -s "$2" "$1" 2>/dev/null | awk 'NR==2{print $2}')"
  [[ -n "$a" ]] || { echo "?/?"; return; }
  printf '%s/%d\n' "$a" "$((a & 0x3f))"
}

# entry_of BIN SYM — bare entry address, or the empty string if SYM is absent.
entry_of() { go tool objdump -s "$2" "$1" 2>/dev/null | awk 'NR==2{print $2}'; }

# grade_pad PAD — the benchmark tokens that are placement-confounded at this pad,
# one per line: those whose symbol does not sit at the same address in both arms,
# plus those whose symbol could not be located at all. Unknown placement is not a
# clearance, so an absent symbol demotes rather than passes.
grade_pad() {
  local pad="$1" e tok sym a b
  for e in "${MEASURED[@]}"; do
    tok="${e%%:*}"
    sym="${e#*:}"
    a="$(entry_of "$BINDIR/base-$pad.bin" "$sym")"
    b="$(entry_of "$BINDIR/new-$pad.bin" "$sym")"
    if [[ -z "$a" || -z "$b" || "$a" != "$b" ]]; then
      printf '%s\n' "$tok"
    fi
  done
}

# grade_rows RE — annotate benchstat rows whose benchmark matches RE. The label
# is what makes them uncitable; a WARN elsewhere in the output would not, since
# the row would still read as a result. RE empty means nothing was demoted.
grade_rows() {
  awk -v re="$1" '
    re != "" && $1 ~ re {
      print $0 "   << placement-confounded: EXCLUDED from the verdict set"
      next
    }
    # A geomean over a set containing a demoted row inherits the demotion. Left
    # unlabelled it would launder the excluded number back into a citable one,
    # which is the totals-ratio trap wearing the instrument as clothes.
    re != "" && $1 == "geomean" {
      print $0 "   << aggregates a placement-confounded row: EXCLUDED"
      next
    }
    { print }
  '
}

# body_of BIN SYM — normalised instruction bytes+mnemonics, for the byte-identity check.
body_of() {
  go tool objdump -s "$2" "$1" 2>/dev/null | awk 'NR>1{print $3, $4}'
}

build_arm() {
  local dir="$1" pad="$2" out="$3"
  write_pad "$dir" "$pad"
  (cd "$dir" && GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go test -c -o "$out" ./bench)
  write_pad "$dir" 0
}

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/remote.sh
  source scripts/remote.sh
  # shellcheck source=scripts/bench.sh
  source scripts/bench.sh

  local BASE_REF="${1:-HEAD~1}"
  shift || true
  local PADS=("$@")
  [[ ${#PADS[@]} -gt 0 ]] || PADS=(0 3 6 9)

  # Sscal is the control rather than Saxpy because the byte-identity check has to
  # be the *same* routine as the measured control, and avx512Axpy carries one
  # differing byte-field between any two placements — the displacement of an
  # off-hot-path CALL — which would trip that check for a reason that is not code
  # drift. avx512Scal is byte-identical across placements (T22: all 46
  # instructions), so it can carry both jobs.
  FILTER="${KEEL_L1_FILTER:-BenchmarkL1S(asum|scal)}"

  # bench.sh exports bench_flags, not BFLAGS — every caller populates the array
  # itself (gate-p1/p2/p4/p5, l1-bench). Omitting this aborted under `set -u`,
  # and it aborted at the *first host*, i.e. after paying for all eight builds.
  # The `while read` form rather than mapfile, matching the gates, so this does
  # not depend on a bash newer than 3.2.
  BFLAGS=()
  while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

  BINDIR="$(mktemp -d)"
  # Not `local`, for the same reason l1-bench.sh says so: the EXIT trap runs in
  # global scope after this function returns, and under `set -u` an unbound name
  # aborts the trap on its first command (#55).
  WORKTREE="$BINDIR/base"
  trap 'rm -f internal/l1/aa_layoutpad_amd64.go; git worktree remove --force "$WORKTREE" 2>/dev/null || true; rm -rf "$BINDIR"' EXIT

  echo "== layout-ensemble — #54/#61. Not a gate: this certifies nothing. =="
  echo
  git diff --quiet HEAD || warn "uncommitted changes are in the NEW arm"
  BASE_SHA="$(git rev-parse --short "$BASE_REF")"
  NEW_SHA="$(git rev-parse --short HEAD)"
  echo "base: $BASE_SHA   new: $NEW_SHA   pads: ${PADS[*]}"
  info "each pad is a statement count, not a byte count; the sampled entry"
  info "alignment is read back from the binary and printed below."
  echo

  git worktree add --detach "$WORKTREE" "$BASE_REF" >/dev/null

  # Build the whole ensemble first, so a build failure costs no host time and so
  # the alignment table can be read before anything is measured.
  local pad arm bin f
  echo "-- sampled placements (entry address / mod 64) --"
  printf '        %-5s %-26s %-26s %s\n' pad "$CHANGED_FN (changed)" "$CONTROL_FN (control)" arm
  for pad in "${PADS[@]}"; do
    for arm in base new; do
      bin="$BINDIR/$arm-$pad.bin"
      if [[ "$arm" == base ]]; then
        build_arm "$WORKTREE" "$pad" "$bin"
      else
        build_arm "$PWD" "$pad" "$bin"
      fi
      printf '        %-5s %-26s %-26s %s\n' \
        "$pad" "$(align_of "$bin" "$CHANGED_FN")" "$(align_of "$bin" "$CONTROL_FN")" "$arm"
    done
  done
  echo

  # The control must be byte-identical between the arms at a given pad, or it is
  # not a control and the ensemble cannot separate layout from code.
  for pad in "${PADS[@]}"; do
    if ! diff -q <(body_of "$BINDIR/base-$pad.bin" "$CONTROL_FN") \
                 <(body_of "$BINDIR/new-$pad.bin" "$CONTROL_FN") >/dev/null; then
      warn "pad=$pad: $CONTROL_FN differs between the arms; it is not a control here"
    fi
  done

  # Grade the placements before any host time is spent, so the exclusions are
  # known and printed before the numbers they apply to exist. Doing it after
  # would invite reading the rows first and the label second.
  local confounded=() re
  echo "-- placement grading (link order: only routines at or before the subject are safe) --"
  for pad in "${PADS[@]}"; do
    re="$(grade_pad "$pad" | paste -sd'|' -)"
    confounded+=("$re")
    if [[ -z "$re" ]]; then
      info "pad=$pad: nothing demoted"
    else
      info "pad=$pad: excluded from the verdict set: ${re//|/ }"
    fi
    # The subject sitting at different addresses in the two arms would mean the
    # ensemble has nothing to hold fixed, so it is a hard stop rather than a
    # demotion: every pad would be measuring layout and code at once.
    if [[ "$(entry_of "$BINDIR/base-$pad.bin" "$CHANGED_FN")" \
       != "$(entry_of "$BINDIR/new-$pad.bin" "$CHANGED_FN")" ]]; then
      warn "pad=$pad: $CHANGED_FN moved between the arms; this ensemble cannot separate layout from code"
      exit 1
    fi
  done
  echo

  local i
  for host in $(remote_hosts); do
    echo "-- $host --"
    remote_probe "$host" | sed 's/^/        /'
    i=-1
    for pad in "${PADS[@]}"; do
      i=$((i + 1))
      echo "        ---- pad=$pad ----"
      local ok=1
      for arm in base new; do
        if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BINDIR/$arm-$pad.bin" \
             "${BFLAGS[@]}" -test.bench="$FILTER" >"$BINDIR/$arm.log" 2>&1; then
          warn "[$host] pad=$pad: the $arm run failed; this placement is dropped"
          ok=0
          break
        fi
      done
      [[ $ok == 1 ]] || continue
      cp "$BINDIR/base.log" "$BINDIR/$BASE_SHA.txt"
      cp "$BINDIR/new.log" "$BINDIR/$NEW_SHA.txt"
      bench_compare "$BINDIR/$BASE_SHA.txt" "$BINDIR/$NEW_SHA.txt" 2>/dev/null |
        grade_rows "${confounded[$i]}" | sed 's/^/        /' ||
        warn "[$host] pad=$pad: the two arms were not compared"
    done
  done

  echo
  echo "reading the result: the changed routine's delta must keep one sign across"
  echo "every pad AND exceed the spread the control routine shows across the same"
  echo "pads. A control that moves as much as the subject means this ensemble"
  echo "resolves nothing, which is itself the answer."
  echo
  echo "rows labelled placement-confounded are not in the verdict set. They are"
  echo "printed because suppressing them would hide coverage, not because they are"
  echo "citable; their symbol sits at a different address in the two arms, so the"
  echo "number conflates the code change with T22's mechanism."
  echo
  echo "quote the floor as the control's excursion per host AND per size, with its"
  echo "shape -- max plus the rest of the distribution. A single excursion is a"
  echo "range, not a distribution, and a dense band argues differently from an"
  echo "outlier. The floor bounds cross-binary A/Bs only: same-binary comparisons"
  echo "(backend ratios, KEEL_FORCE arms) share a layout and are not subject to it."
}

main "$@"
