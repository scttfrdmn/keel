#!/usr/bin/env bash
# Gate P0 — see DESIGN.md §4/P0. Exits 0 only when every criterion for the
# phase holds. A red gate blocks the next phase; there is no override flag
# on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P0):
#   "shim tests pass under all three backends; FMA disassembles to a single
#    VFMADD231PS (grep -gcflags=-S output)."
#
# Two notes on how those are mechanized here — both deliberate, and both
# printed in the gate output rather than hidden:
#
#  1. MNEMONIC. DESIGN.md predicted VFMADD231PS. On go1.26.5 the compiler
#     lowers Float32x16.MulAdd to VFMADD213PS — the same FMA instruction in a
#     different operand-order encoding (132/213/231 differ only in which
#     operand carries the addend). The gate therefore requires exactly one
#     instruction from the VFMADD{132,213,231}PS family AND zero separate
#     VMULPS/VADDPS in the wrapper body, which is the property DESIGN.md
#     actually cares about ("mul+add is a half-of-peak ceiling"). The
#     mnemonic found is printed on every run. See docs/toolchain-notes.md.
#
#  2. HOST ARCH. simd/archsimd is amd64-only in go1.26.5 (its vector types
#     live in *_amd64.go files). The FMA disassembly check works from any
#     host via GOARCH=amd64 cross-compile, but the vector backends cannot be
#     *executed* off amd64. This gate does not paper over that: the
#     differential tests must report having actually exercised all three
#     backends, and if they cannot, the gate FAILS and says why. It does not
#     downgrade to "scalar-only green".
set -euo pipefail

cd "$(dirname "$0")/.."

FAIL=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=1; }
info() { printf '        %s\n' "$1"; }

echo "== gate-p0: toolchain probe & shim =="
echo

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
trap 'rm -f "$TESTLOG" "$ASM"' EXIT
if GOEXPERIMENT=simd go test -v -count=1 ./internal/vec/... >"$TESTLOG" 2>&1; then
  pass "internal/vec tests pass"
else
  fail "internal/vec tests pass"
  sed 's/^/        /' "$TESTLOG" | tail -40
fi

# The suite prints exactly which backends it actually ran ops against, so a
# silently-skipped backend cannot masquerade as a green gate.
COVER="$(grep -o 'keel-backends-exercised:.*' "$TESTLOG" | tail -1 || true)"
if [[ -z "$COVER" ]]; then
  fail "no backend-coverage marker in test output"
  info "expected a line 'keel-backends-exercised: ...' from TestBackendCoverage"
else
  info "$COVER"
  for want in scalar avx2 avx512; do
    if grep -qE "keel-backends-exercised:.*(^| )$want( |$)" <<<"$COVER"; then
      pass "differential tests exercised backend: $want"
    else
      fail "backend NOT exercised on this host: $want"
    fi
  done
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

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p0: GREEN"
  exit 0
fi
echo "gate-p0: RED" >&2
exit 1
