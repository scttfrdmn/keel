#!/usr/bin/env bash
# Gate P0 — see DESIGN.md §4/P0. Exits 0 only when every criterion for the
# phase holds. A red gate blocks the next phase; there is no override flag
# on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P0):
#   "shim tests pass under all three backends; FMA disassembles to a single
#    fused multiply-add instruction — any VFMADD*PS form — rather than a
#    separate multiply and add (grep -gcflags=-S output)."
#
# Two notes on how those are mechanized here — both deliberate, and both
# printed in the gate output rather than hidden:
#
#  1. MNEMONIC. The criterion above is the reworded one (issue #10, 2026-08-12);
#     DESIGN.md originally named VFMADD231PS specifically. On go1.26.5 the
#     compiler lowers Float32x16.MulAdd to VFMADD213PS — the same FMA
#     instruction in a different operand-order encoding (132/213/231 differ only
#     in which operand carries the addend) — so the criterion as first written
#     would have failed a toolchain that satisfied what P0 actually needs. This
#     gate has always required the property rather than the encoding: exactly one
#     instruction from the VFMADD{132,213,231}PS family AND zero separate
#     VMULPS/VADDPS in the wrapper body ("mul+add is a half-of-peak ceiling"), so
#     the rewording brings DESIGN.md into line with the check, not the reverse.
#     Which operand order it is remains consequential and is tracked where the
#     consequences are — DESIGN.md's roofline section, where 231-with-broadcast
#     versus 213 is I = 2.875 versus 4.625 (T12, #17/#18/#20). The mnemonic found
#     is printed on every run. See docs/toolchain-notes.md.
#
#  2. HOST ARCH. simd/archsimd is amd64-only in go1.26.5 (its vector types
#     live in *_amd64.go files). The FMA disassembly check works from any
#     host via GOARCH=amd64 cross-compile, but the vector backends cannot be
#     *executed* off amd64. This gate does not paper over that: the
#     differential tests must report having actually exercised all three
#     backends, and if they cannot, the gate FAILS and says why. It does not
#     downgrade to "scalar-only green".
#
#  3. EXECUTION TARGETS. Since the dev host is darwin/arm64, "all three
#     backends ran" is a claim about some *other* machine. The gate therefore
#     runs the differential suite on every configured target — the local host
#     always, plus each host in .keel-hosts / $KEEL_REMOTE_HOSTS, reached by
#     shipping a cross-compiled static test binary (scripts/remote.sh) — and
#     requires that (a) every target that ran, passed, and (b) at least one
#     target exercised all three backends in one binary on one CPU, with that
#     machine named in the output. This is the same substantive requirement as
#     before, correctly attributed: a host without AVX-512 is no longer
#     *blamed* for lacking it, but nothing is accepted as green until real
#     silicon has executed it.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

echo "== gate-p0: toolchain probe & shim =="
echo

# ------------------------------------------------------------- tree state (#63)
assert_no_strays

# ---------------------------------------------------------------- toolchain
echo "-- toolchain --"
GOVER="$(go env GOVERSION)"
info "go: $GOVER   host: $(go env GOHOSTOS)/$(go env GOHOSTARCH)"
# DESIGN.md: Go 1.27rc or newest 1.26.x.
if [[ "$GOVER" =~ ^go1\.(2[6-9]|[3-9][0-9]) ]]; then
  pass "toolchain is go1.26+ ($GOVER)"
else
  fail "toolchain too old: $GOVER (need go1.26+ per DESIGN.md §4/P0)"
fi

if GOEXPERIMENT=simd go list simd/archsimd >/dev/null 2>&1; then
  pass "GOEXPERIMENT=simd exposes simd/archsimd"
else
  fail "GOEXPERIMENT=simd does not expose simd/archsimd"
fi

# ------------------------------------------------------------------- builds
echo
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then
  pass "make build (GOEXPERIMENT=simd)"
else
  fail "make build (GOEXPERIMENT=simd)"
fi
# The scalar path must always build on a stock toolchain (DESIGN.md §4/P5,
# held from P0 so it can never silently rot).
if go build ./... 2>&1; then
  pass "make stock (scalar path, no experiment)"
else
  fail "make stock (scalar path, no experiment)"
fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then
  pass "go vet (GOEXPERIMENT=simd)"
else
  fail "go vet (GOEXPERIMENT=simd)"
fi

# ------------------------------------------------- shim differential tests
echo
echo "-- shim differential tests --"
TESTLOG="$(mktemp)"
ASM="$(mktemp)"
TESTBIN="$(mktemp -u)/vec.test"
trap 'rm -f "$TESTLOG" "$ASM" "$TESTBIN"; rmdir "$(dirname "$TESTBIN")" 2>/dev/null || true' EXIT
mkdir -p "$(dirname "$TESTBIN")"

# FULL_COVER_TARGET names the machine on which all three backends actually
# executed. Empty at the end of this section means a red gate.
FULL_COVER_TARGET=""
N_FULLCAP=0

# record_target NAME LOG OK — score one execution target's test run.
#
# The suite prints exactly which backends it ran ops against, so a
# silently-skipped backend cannot masquerade as a green gate. A target that
# merely *lacks* a backend is reported, not failed; the aggregate check below
# is what insists some target had all three.
record_target() {
  local name="$1" log="$2" ok="$3" cover avail missing="" unexercised=""
  test_verdict "$name" "$log" "$ok" "internal/vec tests pass"

  cover="$(grep -o 'keel-backends-exercised:.*' "$log" | tail -1 || true)"
  if [[ -z "$cover" ]]; then
    unmeasured "[$name] no backend-coverage marker in test output, so what this run exercised cannot be read"
    info "expected a line 'keel-backends-exercised: ...' from TestBackendCoverage"
    return
  fi
  info "[$name] $cover"
  # The availability marker, which TestBackendCoverage prints beside the coverage
  # one (internal/vec/vec_diff_test.go:81-82). This gate used to skip it and
  # report every unexercised backend as `unavailable here` — a claim about the
  # CPU inferred from a claim about the run, which is the assumption #73's sweep
  # exists to retire. A backend that is available and was not exercised is a
  # skipped backend, and that is a FAIL; one the host does not have is neither.
  avail="$(grep -o 'keel-backends-available:.*' "$log" | tail -1 || true)"
  local have_avail=1
  if [[ -z "$avail" ]]; then
    have_avail=0
    unmeasured "[$name] no backend-availability marker, so an unexercised backend cannot be told from an absent one"
  else
    info "[$name] $avail"
  fi
  local want unavailable=""
  for want in scalar avx2 avx512; do
    grep -qE "keel-backends-exercised:.*(^| )$want( |$)" <<<"$cover" \
      || missing="$missing $want"
    grep -qE "keel-backends-available:.*(^| )$want( |$)" <<<"$avail" \
      || unavailable="$unavailable $want"
  done
  for want in $missing; do
    # `if` rather than `grep -q ... && assign`: the AND-list's status is grep's,
    # so a no-match would make the enclosing loop return non-zero under set -e.
    if grep -qE "keel-backends-available:.*(^| )$want( |$)" <<<"$avail"; then
      unexercised="$unexercised $want"
    fi
  done
  # Coverage state for the aggregate below (#73's tier C): a target with all
  # three available is one that could exercise all three, which is what separates
  # a fleet that came back short from a fleet with nothing to ask.
  [[ -z "$unavailable" ]] && N_FULLCAP=$((N_FULLCAP + 1))
  if [[ -z "$missing" ]]; then
    pass "[$name] exercised all three backends in one binary"
    # `if` rather than `[[ ... ]] && assign`, for the same reason as the loop
    # above but with teeth: this was the LAST command of this function, so when
    # $ok was non-zero the AND-list's non-zero status became record_target's
    # return status, and a function call is a simple command that `set -e` acts
    # on. A host whose tests failed while exercising all three backends
    # therefore killed the gate here -- after its last PASS, before the verdict
    # section -- exiting 1 with no `gate-p0: RED` line at all. Since RED also
    # exits 1, that reads as a truncated red gate rather than a harness fault
    # (#80, the #76 family in a new construct).
    if [[ "$ok" -eq 0 ]]; then FULL_COVER_TARGET="$name"; fi
  elif [[ -n "$unexercised" ]]; then
    fail "[$name] backends available here but not exercised:$unexercised"
  elif [[ "$have_avail" -eq 0 ]]; then
    # Without the availability marker there is nothing to license the word
    # "unavailable": every backend looks unavailable because the grep had no line
    # to match, not because the CPU lacks it. Saying `unavailable here` in this
    # branch would restate, as a finding, the very inference the UNMEASURED line
    # above says cannot be drawn. The gate is already blocked by that line.
    info "[$name] not exercised, and whether this host has them is unreadable:$missing"
  else
    info "[$name] backends unavailable here:$missing"
  fi
}

# Local target. Always run: it is what proves the scalar spec and the
# characterization tests hold on the dev host and on a non-amd64 GOARCH.
LOCAL_OK=0
GOEXPERIMENT=simd go test -v -count=1 ./internal/vec/... >"$TESTLOG" 2>&1 || LOCAL_OK=$?
record_target "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$TESTLOG" "$LOCAL_OK"

# Remote targets: ship a cross-compiled static test binary and run it there.
HOSTS="$(remote_hosts)"
# The ledger of what this gate trusts rather than checks (#73 tier C, ruled
# 2026-08-15). Declared here, where the fleet is named; printed beside the
# verdict by assumed_ledger below.
assume_fleet "$HOSTS"
require_disk
if [[ -z "$HOSTS" ]]; then
  info "no remote targets configured (.keel-hosts or \$KEEL_REMOTE_HOSTS)"
else
  if remote_build_test ./internal/vec/ "$TESTBIN" >"$TESTLOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary (static, GOAMD64 default)"
  else
    fail "cross-compile of linux/amd64 test binary"
    sed 's/^/        /' "$TESTLOG" | tail -20
  fi
  # Count configured vs. actually-scored targets and compare at the end. A
  # target that disappears mid-loop must not pass unnoticed — the first
  # version of this loop lost every host after the first (ssh drained the
  # loop's stdin) and still printed GREEN off one machine.
  N_CONFIGURED=0
  N_SCORED=0
  while read -r host; do
    [[ -n "$host" ]] || continue
    N_CONFIGURED=$((N_CONFIGURED + 1))
    provenance="$(remote_probe "$host" || true)"
    if [[ -z "$provenance" ]]; then
      # Unreachable blocks the gate, never a silent skip: a configured target
      # that quietly vanishes is how a gate starts lying. It reads UNMEASURED
      # rather than FAIL because nothing about this target was learned — the
      # count below is what turns a vanished target into a coverage failure,
      # and it is a number the gate did take (#73).
      unmeasured "[$host] unreachable, or /proc/cpuinfo unreadable: this target answered nothing, so its coverage is unmeasured rather than failed"
      continue
    fi
    info "[$host] $provenance"
    REMOTE_OK=0
    remote_exec "$host" "$TESTBIN" -test.v >"$TESTLOG" 2>&1 || REMOTE_OK=$?
    record_target "$host" "$TESTLOG" "$REMOTE_OK"
    N_SCORED=$((N_SCORED + 1))
  done <<<"$HOSTS"

  if [[ "$N_SCORED" -eq "$N_CONFIGURED" ]]; then
    pass "every configured remote target ran ($N_SCORED/$N_CONFIGURED)"
  else
    fail "only $N_SCORED of $N_CONFIGURED configured remote targets ran"
  fi
fi

if [[ -n "$FULL_COVER_TARGET" ]]; then
  pass "all three backends exercised on real silicon (target: $FULL_COVER_TARGET)"
elif [[ "$N_FULLCAP" -gt 0 ]]; then
  fail "no execution target exercised all three backends, though $N_FULLCAP target(s) reported all three available"
else
  unmeasured "no execution target reported all three backends available, so exercising them is unmeasured rather than missed: there was no host to ask"
  info "simd/archsimd is amd64-only and the AVX-512 backend needs the"
  info "F/CD/BW/DQ/VL bundle at runtime. Point .keel-hosts at an amd64"
  info "AVX-512 host; see docs/hosts.md and docs/toolchain-notes.md T1."
fi

# ------------------------------------------------------------ FMA lowering
# The arbiter is the disassembly, not the doc comment (DESIGN.md §6).
echo
echo "-- FMA lowering (disassembly is the arbiter) --"
if ! GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 GOAMD64=v3 \
     go build -gcflags='-S' ./internal/vec/ >"$ASM" 2>&1; then
  fail "amd64 disassembly build of internal/vec"
  sed 's/^/        /' "$ASM" | tail -20
else
  pass "amd64 disassembly build of internal/vec"

  # Isolate one function body: from its STEXT line to the next STEXT line.
  body() { awk -v fn="$1" '
    index($0, fn" STEXT") == 1 {inside=1; next}
    inside && / STEXT / {inside=0}
    inside {print}' "$ASM"; }

  check_fused() {
    local fn="$1" want_reg="$2" b n_fma n_mul n_add mnem
    b="$(body "$fn")"
    if [[ -z "$b" ]]; then
      fail "$fn: symbol not found in -gcflags=-S output"
      return
    fi
    n_fma=$(grep -cE 'VFMADD(132|213|231)PS' <<<"$b" || true)
    n_mul=$(grep -cE '\bVMULPS\b'            <<<"$b" || true)
    n_add=$(grep -cE '\bVADDPS\b'            <<<"$b" || true)
    mnem=$(grep -oE 'VFMADD(132|213|231)PS.*' <<<"$b" | head -1 | tr -s ' \t' ' ')
    if [[ "$n_fma" -eq 1 && "$n_mul" -eq 0 && "$n_add" -eq 0 ]]; then
      pass "$fn: single fused instruction, no separate mul/add"
      info "lowering: $mnem"
      if ! grep -qE "$want_reg" <<<"$b"; then
        fail "$fn: fused op does not use $want_reg registers"
      fi
    else
      fail "$fn: expected 1 VFMADD*PS + 0 VMULPS + 0 VADDPS; got fma=$n_fma mul=$n_mul add=$n_add"
      sed 's/^/        /' <<<"$b"
    fi
  }

  # AVX-512 (zmm) and AVX2 (ymm) fused-multiply-add wrappers.
  check_fused 'github.com/scttfrdmn/keel/internal/vec.FMA512' 'Z[0-9]+'
  check_fused 'github.com/scttfrdmn/keel/internal/vec.FMA256' 'Y[0-9]+'
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p0: GREEN"
  exit 0
fi
echo "gate-p0: RED" >&2
exit 1
