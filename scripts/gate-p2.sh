#!/usr/bin/env bash
# Gate P2 — see DESIGN.md §4/P2. Written at the START of phase P2, then made
# green. Exits 0 only when every criterion for the phase holds. A red gate
# blocks the next phase; there is no override flag on purpose.
#
# THIS GATE IS A GO/NO-GO, NOT A HURDLE (CLAUDE.md). If it stays red after the
# documented kernel-shaping steps and one tile shrink, the deliverable is
# docs/spill-report.md and a blocked issue — not a weakened check and not a
# switch to hand-written assembly.
#
# Criteria (verbatim from DESIGN.md §4/P2):
#   "`spill-audit` reports 0 accumulator spills in the steady-state K-loop,
#    AND the raw microkernel (packed inputs, no blocking) hits >=55% of
#    measured peak."
#
# How those are mechanized, and every judgement call involved:
#
#  1. CORRECTNESS FIRST. A fast wrong kernel is not a P2 pass. The microkernel
#     is differential-tested against a scalar reference for the same tile, under
#     internal/oracle.Tolerance, on every backend each machine can execute —
#     locally (scalar only, stock toolchain) and on every amd64 host in
#     .keel-hosts, by the same cross-compile-and-ship path as P0/P1.
#
#  2. THE SPILL AUDIT IS A COMPILE-TIME PROPERTY, so it runs here on the dev
#     host, against the linux/amd64 object code the remote hosts will execute.
#     internal/spill parses `go build -gcflags=-S`, finds the steady-state
#     K-loop (the innermost loop carrying the arithmetic), and reports every
#     stack-relative reference inside it. "0 accumulator spills" is mechanized
#     as: no instruction in that loop body has both a vector register operand
#     and an (SP)-relative memory operand. Register-to-register copies are
#     counted and printed but are NOT spills — they cost issue slots, not
#     memory traffic, and DESIGN.md's criterion is about the latter. That
#     distinction decides this phase's verdict, so it is stated here rather
#     than left implied.
#
#     It is audited on the SHIPPED shapes only. DESIGN.md's own 12-accumulator
#     tile cannot be allocated on go1.26.5 (docs/toolchain-notes.md T10, issue
#     #18) and is kept as kern.ReferenceTile — audited and benchmarked, never
#     dispatched. Its audit runs below as evidence and is explicitly non-fatal,
#     because a gate that failed on a shape nothing ships would be measuring the
#     evidence instead of the product. Everything in kern.Kernels() is held to
#     zero.
#
#  3. THE OTHER SHAPING RULES ARE CHECKED ON THE LOOP BODY, NOT ON THE FILE.
#     DESIGN.md §4/P2 lists pre-sliced panels (bounds-check elimination), no
#     calls in the K-loop, and pointer-free data. All three are properties of the
#     steady-state loop, so all three are checked by the audit tool that already
#     identifies it: a call is a CALL in the body, and a surviving bounds check
#     is a branch out of the body to a block that calls runtime.panic*.
#
#     `-d=ssa/check_bce` is printed as provenance and is NOT the criterion. It
#     reports every bounds check in the package, and a kernel legitimately has
#     dozens outside the K-loop — `a[:kc*MR]` in the prologue and
#     `c[i*ldc:i*ldc+NR]` in the write-out both report one, and both cost nothing
#     amortized over K. Failing on those would make the criterion unsatisfiable
#     for reasons unrelated to what P2 is asking, so the count is printed and the
#     loop body is enforced.
#
#  4. THE PEAK KERNEL'S NO-MEMORY PROPERTY IS CHECKED HERE (issue #11).
#     internal/vec/peak.go rests on four properties; three are guarded by an
#     exact arithmetic witness that runs on every host, and the fourth —
#     nothing in the loop touches memory — cannot be seen by arithmetic. The
#     same audit tool checks it in -mode=nomemory, so the denominator P2
#     divides by is verified in the run that divides by it.
#
#  5. PERCENT OF PEAK IS MEASURED AGAINST A MEASURED PEAK, IN ONE RUN. The
#     numerator (the microkernel) and the denominator (BenchmarkPeak) are
#     measured in the *same* benchmark invocation on each host, so they share a
#     frequency and thermal state; a ratio of two numbers taken hours apart on
#     one machine would be a worse measurement than either of them. Both come
#     out of benchstat under the §5.4 methodology (issue #14): -count=10
#     -benchtime=1s, medians, and the 55% bar counts as cleared only net of
#     both confidence intervals. Every host that can run the kernel must clear
#     it, and at least one must do so under the performance governor.
#
#     Note what this bar is not: it is not 55% of a formula, and it is not the
#     best of N hosts. See docs/hosts.md and issue #15 for the one host whose
#     memory-touching benchmarks are bimodal between runs.
#
#     It IS the best of the shipped shapes, per host. Two zero-spill tiles ship
#     (2x32 and 4x32) because which one wins depends on the host's front-end
#     width and load ports rather than on anything in the source, and P3 will
#     dispatch to one of them. So the criterion is applied to the shape that
#     wins on that host, with every shape's number printed either way. Requiring
#     both to clear would fail a host for carrying a second kernel it would never
#     select; requiring only their average would hide the winner.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh

FAIL=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=1; }
info() { printf '        %s\n' "$1"; }

# The tiles under audit, the peak kernels the denominator comes from, and the
# floor. KERNEL.md records why the tiles have these dimensions and which one wins
# where; the K-loop bodies are in internal/vec, not internal/kern, for the reason
# measured in the header comment of internal/vec/gemm_amd64.go.
KERN_PKG="./internal/vec"
KERN_FUNCS="Kernel2x32,Kernel4x32"
# DESIGN.md §4/P2's 12-accumulator tile. Audited for evidence, never shipped; see
# criterion 2 above and docs/toolchain-notes.md T10.
REF_FUNCS="Kernel6x32"
PEAK_FUNCS="avx512Peak,avx2Peak,scalarPeak"
PEAK_FLOOR=0.55
# The shipped shapes' gate benchmarks, at the kc P3 will use. The floor applies to
# whichever of these wins on the host (criterion 5).
GATE_KERNELS="Kernel/2x32/avx512/kc=128 Kernel/4x32/avx512/kc=128"
GATE_REF_KERNEL="Kernel/6x32/avx512/kc=128"
GATE_PEAK="Peak/avx512"
# The gate runs exactly what it reads. `Peak|Kernel` also ran the kc=32/64/256/512
# sweep — 20 sub-benchmarks per host, ~7 min each at -count=10, none of which any
# threshold here looks at. Go applies each slash-separated pattern element at its
# own depth and ignores the extra elements for a shallower benchmark, so `Peak`
# still matches under a four-element pattern. The kc sweep is a deliberate one-off;
# KERNEL.md §1 has the command.
BENCH_FILTER='Peak|Kernel/.*/.*/kc=128'
SSADIR="build/ssa"

echo "== gate-p2: microkernel + spill audit (GO/NO-GO) =="
echo

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

LOG="$(mktemp)"
BINDIR="$(mktemp -d)"
BIN="$BINDIR/keel.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
trap 'rm -rf "$LOG" "$BINDIR"' EXIT

# --------------------------------------------------- kernel correctness
echo
echo "-- microkernel correctness (differential vs the scalar tile reference) --"

score_run() {
  local name="$1" log="$2" ok="$3" cover
  if [[ "$ok" -eq 0 ]]; then
    pass "[$name] kernel tests pass"
  else
    fail "[$name] kernel tests pass"
    sed 's/^/        /' "$log" | tail -40
  fi
  cover="$(grep -o 'keel-kern-backends-exercised:.*' "$log" | tail -1 || true)"
  if [[ -z "$cover" ]]; then
    fail "[$name] no kernel backend-coverage marker in test output"
    return
  fi
  info "[$name] $cover"
}

LOCAL_OK=0
GOEXPERIMENT=simd go test -v -count=1 ./internal/kern/... >"$LOG" 2>&1 || LOCAL_OK=$?
score_run "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$LOG" "$LOCAL_OK"

HOSTS="$(remote_hosts)"
AVX512_GREEN=""
if [[ -z "$HOSTS" ]]; then
  fail "P2 needs an amd64 host to execute the AVX-512 kernel; none configured"
else
  if remote_build_test ./internal/kern "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 kernel test binary"
  else
    fail "cross-compile of linux/amd64 kernel test binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      fail "[$host] unreachable"
      continue
    fi
    info "[$host] $prov"
    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    score_run "$host" "$LOG" "$OK"
    if [[ "$OK" -eq 0 ]] && grep -qE 'keel-kern-backends-exercised:.*(^| )avx512( |$)' "$LOG"; then
      AVX512_GREEN="$host"
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "kernel tests green with the avx512 tile exercised (target: $AVX512_GREEN)"
  else
    fail "no target ran the kernel tests green with the avx512 tile"
  fi
fi

# ------------------------------------------------------------- spill audit
echo
echo "-- spill audit: steady-state K-loop of the shipped shapes ($KERN_FUNCS) --"
info "compile-time property, audited against the linux/amd64 object code the hosts run"
if GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
     -pkg "$KERN_PKG" -func "$KERN_FUNCS" -mode spill -ssa "$SSADIR" >"$LOG" 2>&1; then
  sed 's/^/        /' "$LOG"
  pass "0 accumulator spills in the steady-state K-loop"
  pass "0 calls in the steady-state K-loop (no write barrier, no runtime helper)"
  pass "0 surviving bounds checks in the steady-state K-loop (panels are pre-sliced)"
else
  sed 's/^/        /' "$LOG"
  fail "spill audit: before touching this, read the go/no-go protocol in CLAUDE.md"
fi
for f in ${KERN_FUNCS//,/ }; do
  if [[ -s "$SSADIR/$f.html" ]]; then
    info "ssa.html archived: $SSADIR/$f.html ($(wc -c <"$SSADIR/$f.html" | tr -d ' ') bytes; gitignored, see KERNEL.md)"
  else
    fail "no ssa.html archived for $f — the 'why' behind any spill would be unavailable"
  fi
done

# ------------------------------- the tile that cannot be allocated (evidence)
echo
echo "-- reference tile $REF_FUNCS: DESIGN.md's 12 accumulators, EVIDENCE ONLY --"
info "expected to spill on go1.26.5 (T10, issue #18); this audit cannot fail the gate"
REF_OK=0
GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
  -pkg "$KERN_PKG" -func "$REF_FUNCS" -mode spill -ssa "$SSADIR" >"$LOG" 2>&1 || REF_OK=$?
grep -E 'steady-state loop' "$LOG" | sed 's/^/        /' || true
if [[ "$REF_OK" -eq 0 ]]; then
  # Worth knowing loudly: it would mean the register ceiling moved, and the
  # shipped shapes should be revisited against the wider budget.
  info "NOTE: $REF_FUNCS no longer spills. T10 may be fixed upstream — reopen #18."
else
  info "spills as documented; the GFLOP/s cost of that is measured below"
fi

# ------------------------------------------------------ other shaping rules
echo
echo "-- kernel shaping: bounds checks outside the K-loop (provenance, not a check) --"
info "the criterion is the loop-body check above; this is the whole-package count"
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 \
     go build -gcflags='-d=ssa/check_bce' "$KERN_PKG" ./internal/kern 2>"$LOG"; then
  BCE_N="$(grep -c 'Found Is\(Slice\)\?InBounds' "$LOG" || true)"
  info "check_bce: ${BCE_N:-0} bounds check(s) in $KERN_PKG + ./internal/kern, all outside the K-loop"
  info "  (prologue slices a[:kc*MR]/b[:kc*NR], the write-out's c[i*ldc:...], and the scalar reference)"
  grep 'Found Is' "$LOG" | sed 's/^/        /' | head -5 || true
  [[ "${BCE_N:-0}" -gt 5 ]] && info "  ... $((BCE_N - 5)) more; full output is in the gate log"
  pass "check_bce output recorded as provenance"
else
  sed 's/^/        /' "$LOG" | tail -20
  fail "build with -d=ssa/check_bce failed"
fi

# ------------------------------------- the denominator's no-memory property
echo
echo "-- peak kernels: no memory in the loop (issue #11, property 1) --"
if go run ./internal/spill/cmd/spill-audit \
     -pkg ./internal/vec -func "$PEAK_FUNCS" -mode nomemory >"$LOG" 2>&1; then
  sed 's/^/        /' "$LOG"
  pass "every peak kernel's steady-state loop is register-only"
else
  sed 's/^/        /' "$LOG"
  fail "a peak kernel's loop touches memory; the P2 denominator is not a ceiling"
fi

# ----------------------------------------- >= 55% of measured peak (§4/P2)
echo
echo "-- microkernel vs measured peak (>= 55% of measured; issue #14 methodology) --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; numerator and denominator measured in the same run"
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

PERF_GOV_HOST=""
N_CLEARED=0
if [[ -z "$HOSTS" ]]; then
  fail "the 55% criterion needs an amd64 host; none configured"
else
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary"
  else
    fail "cross-compile of linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    info "[$host] governor=${gov:-unknown}"
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$BENCH_FILTER" \
         >"$BENCHLOG" 2>&1; then
      fail "[$host] benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"

    info "[$host] peak   $(bench_describe "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    # DESIGN.md's tile, printed on every host so the cost of the T10 register
    # ceiling is a number in the gate log rather than an assertion in a doc.
    if [[ -n "$(bench_stat "$GATE_REF_KERNEL" "$BENCHCSV" GFLOP/s)" ]]; then
      refpt="$(bench_ratio "$GATE_REF_KERNEL" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      info "[$host] 6x32  $(bench_describe "$GATE_REF_KERNEL" "$BENCHCSV" GFLOP/s) = $(awk -v r="${refpt:-0}" 'BEGIN{printf "%.1f", r * 100}')% of peak (spills; not shipped)"
    fi

    # The floor applies to the shape that wins here (criterion 5). Every shape's
    # number is printed either way, and "measured but unbounded" is a failure to
    # measure rather than a shape that lost.
    BEST_LO=""; BEST_PT=""; BEST_ID=""; UNBOUNDED=""; MEASURED=""
    for kname in $GATE_KERNELS; do
      [[ -n "$(bench_stat "$kname" "$BENCHCSV" GFLOP/s)" ]] || continue
      MEASURED=yes
      klo="$(bench_ratio_lo "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      kpt="$(bench_ratio "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$klo" ]]; then
        UNBOUNDED="$UNBOUNDED $kname"
        info "[$host] ${kname##Kernel/} $(bench_describe "$kname" "$BENCHCSV" GFLOP/s) — no CI, not counted"
        continue
      fi
      info "[$host] ${kname##Kernel/} $(bench_describe "$kname" "$BENCHCSV" GFLOP/s) = $(awk -v r="$kpt" 'BEGIN{printf "%.1f", r * 100}')% of peak, $(awk -v r="$klo" 'BEGIN{printf "%.1f", r * 100}')% net of CI"
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_ID="${kname##Kernel/}"
      fi
    done

    if [[ -z "$MEASURED" ]]; then
      info "[$host] no avx512 kernel sub-benchmark (CPU lacks it); not counted"
      continue
    fi
    if [[ -z "$BEST_LO" ]]; then
      fail "[$host] no bounded percent-of-peak: benchstat established no confidence interval for${UNBOUNDED}"
      continue
    fi
    frac="$(awk -v r="$BEST_LO" 'BEGIN{printf "%.1f", r * 100}')"
    fracpt="$(awk -v r="$BEST_PT" 'BEGIN{printf "%.1f", r * 100}')"
    if awk -v r="$BEST_LO" -v f="$PEAK_FLOOR" 'BEGIN{exit !(r >= f)}'; then
      pass "[$host] best shipped shape $BEST_ID at ${fracpt}% of measured peak, ${frac}% net of CI (>= 55%)"
      N_CLEARED=$((N_CLEARED + 1))
      [[ "$gov" == "performance" ]] && PERF_GOV_HOST="$host"
    else
      fail "[$host] best shipped shape $BEST_ID at ${fracpt}% of measured peak, only ${frac}% net of CI (< 55%)"
    fi
  done <<<"$HOSTS"

  if [[ "$N_CLEARED" -eq 0 ]]; then
    fail "no host cleared the 55% floor"
  fi
  if [[ -n "$PERF_GOV_HOST" ]]; then
    pass "the 55% floor was cleared under the performance governor ($PERF_GOV_HOST)"
  else
    fail "no host cleared 55% under the performance governor (§5.4 rule 5 requires one)"
  fi
fi

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p2: GREEN"
  exit 0
fi
echo "gate-p2: RED" >&2
echo "P2 is a go/no-go: if this is still red after the documented shaping steps" >&2
echo "and one tile shrink, write docs/spill-report.md and stop (CLAUDE.md)." >&2
exit 1
