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
#  5b. THE BAR IS ROOFLINE-RELATIVE ON AN ISSUE-BOUND HOST (DESIGN.md §4/P2 as
#     amended by the ruling on #19). The flat 55% has one denominator and so
#     assumes the front end can deliver the kernel's instruction mix at 55% of
#     FMA peak. On a host retiring two full-width FMAs per cycle from a ~4-wide
#     front end it cannot, and the flat bar becomes a demand that the kernel beat
#     the decode stage. Such a host is instead held to 90% of its issue roofline,
#     computed from the spill audit's own instruction counts.
#
#     Four things keep that from being a weaker gate rather than a differently
#     expressed one. None is asserted here: the decision is a pure function in
#     scripts/roofline.sh, and scripts/roofline-test.sh drives it with fixtures
#     that attempt each abuse and asserts each is refused. Those controls run at
#     the top of this gate, before any benchmarking.
#       - Classification uses evidence independent of the fraction being gated.
#         The naive test — measured rate < the rate 55% would require — reduces
#         algebraically to f < 0.55, i.e. "the host failed", and would make the
#         amendment self-granting. What is tested instead is whether an issue
#         *ceiling* exists: do structurally different instruction mixes converge
#         on one retirement rate? Only a machine property does that.
#       - The ceiling is established by the mixes OTHER than the shape under test.
#         With that shape included, attain reduces to p_best / max_i p_i >=
#         1/cspread >= 1/1.10 = 0.909, which already clears the 0.90 floor: every
#         host the classifier admits would pass by construction. The first draft
#         of this gate had exactly that hole, and it was widest in the binding
#         case — on janus the shape under test *is* the argmin of p_i, so
#         1/cspread and attain agreed to four places (0.9476).
#       - A ceiling the machine exceeds is not a ceiling. If the shape under test
#         retires above the rate the ceiling mixes converged on, the issue-bound
#         hypothesis is falsified by its own data and the host reverts to the flat
#         floor. This is also what stops the mirror abuse of understating the
#         ceiling, and it is what correctly returns antares (Zen 5) to FMA-bound.
#       - The roofline branch requires the shape to be near the sweep's best
#         insns/FMA, because a roofline computed from the instruction count of the
#         kernel under test rises as that kernel gets worse.
#     A merely-slow kernel produces divergent rates, classifies FMA-bound, and
#     faces the full 55%.
#
#     WHAT THE AMENDMENT COSTS, AS A NUMBER. The peak kernel is always in the
#     ceiling set with f = 1, so max_i p_i >= I_peak, and the shape guard caps the
#     denominator at SWEEP_BEST_IPF * ROOF_SHAPE_SLACK. The effective floor for an
#     issue-bound host is therefore never below
#         ROOF_FLOOR * I_peak / (SWEEP_BEST_IPF * ROOF_SHAPE_SLACK)
#         = 0.90 * 2.25 / 4.659 = 43.5% of measured peak.
#     That is the whole of the slack granted: from 55% to no less than 43.5%, and
#     only to a host that has independently demonstrated a front-end ceiling.
#
#     THE AMENDMENT RATCHETS; IT DOES NOT RETIRE. An earlier draft claimed it was
#     self-retiring — that when the lowering improves (T12/#20) the host would stop
#     classifying issue-bound and the flat floor would resume. The arithmetic says
#     otherwise, and better: with I falling 4.625 -> ~2.875, janus stays at its
#     front-end ceiling but the roofline *rises* to 2.250/2.875 = 78.3%, so the
#     required floor becomes 0.90 * 78.3% = 70.4% — stricter than the 55% the
#     amendment replaced. The floor is 0.90 * max_i p_i / I_b and max_i p_i is
#     pinned by the peak kernel, so it is monotone non-increasing in I: every
#     improvement to the lowering tightens this gate automatically. A kernel at 60%
#     of peak clears the flat 55% and fails the post-fix ratchet; that is a control
#     in roofline-test.sh.
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
# shellcheck source=scripts/roofline.sh
source scripts/roofline.sh

FAIL=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=1; }
info() { printf '        %s\n' "$1"; }

# audit_ipf FUNC FILE -> that function's audited instructions per FMA.
#
# Computed from the audit's own integer counts, not from its rounded "(N.NN per
# arith)" display, because this number is a gate input. "arith" appears twice on
# the line ("16 arith" and "per arith):"); only the first is a bare token.
audit_ipf() {
  awk -v fn=".$1: steady-state loop" '
    index($0, fn) {
      for (i = 2; i <= NF; i++) {
        if ($i == "insns") ins = $(i-1)
        if ($i == "arith") ar  = $(i-1)
      }
      if (ins != "" && ar != "" && ar + 0 > 0) { printf "%.6f", ins / ar; exit }
    }' "$2"
}

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
# --- the amended throughput floor (DESIGN.md §4/P2, ruling on #19) -----------
# An FMA-bound host is held to PEAK_FLOOR. An issue-bound host is held to
# ROOF_FLOOR of its issue roofline, because the flat floor there demands the
# kernel beat the decode stage: the front end cannot deliver the kernel's
# instruction mix at 55% of FMA peak no matter how the loop is written.
#
# The roofline is clock-free, which is why this gate needs no taskset and no
# hardware counters. With f_i the measured fraction of peak for mix i, R the
# host's FMA/cycle and I_i the audited instructions per FMA, the observed
# retirement rate is r_i = f_i * R * I_i, and the roofline for a shape with I
# instructions per FMA is r_max / (R * I). R is common to every mix on the host,
# so it cancels:
#
#       roofline(I) = max_i(f_i * I_i) / I
#
# Every input is already collected here: f_i from the benchmark, I_i from the
# spill audit's own instruction counts. The peak kernel is always one of the
# mixes, with f = 1 by definition since it *is* the denominator — and being the
# denominator, it is never the shape under test, so it is never excluded from the
# ceiling set. That is what pins max_i p_i >= I_peak from beneath.
#
# r_max is the max over mixes, not the mean: a ceiling is an upper bound on what
# was observed, and the max raises the roofline and so tightens the criterion.
ROOF_FLOOR=0.90
# Classification. A host is issue-bound iff structurally different instruction
# mixes converge on the same retirement rate. This must not be tested as
# "measured rate < the rate 55% would need", which expands to f < 0.55 — the
# host failed the floor, wearing a derivation. Convergence across mixes is a
# property of the machine; a merely-slow kernel yields *divergent* rates, lands
# in the FMA-bound branch, and faces the full 55%.
#
# The spreads below are computed over the ceiling mixes only, i.e. excluding the
# shape being gated. Including it makes attain >= 1/ISSUE_CONVERGE_MAX = 0.909
# an algebraic identity, and the 0.90 roofline floor then decides nothing; see
# criterion 5b and scripts/roofline.sh (INDEPENDENCE).
ISSUE_CONVERGE_MAX=1.10   # max/min of f_i*I_i across ceiling mixes, at or below
ISSUE_MIX_SPREAD_MIN=1.25 # ... while max/min of I_i is at least this
# Anti-vacuity guard on the roofline branch. A roofline computed from the
# instruction count of the kernel under test *rises* as that kernel gets worse,
# so "90% of roofline" would otherwise mean "90% of whatever we happened to
# emit". The shipped shape must also be within ROOF_SHAPE_SLACK of the best
# zero-spill insns/FMA in the recorded sweep (KERNEL.md §3), which no amount of
# padding can satisfy. This is also the term that bounds the amendment's slack at
# 43.5% of peak (criterion 5b).
SWEEP_BEST_IPF=4.438
ROOF_SHAPE_SLACK=1.05
# The shipped shapes' gate benchmarks, at the kc P3 will use. The floor applies to
# whichever of these wins on the host (criterion 5).
GATE_KERNELS="Kernel/2x32/avx512/kc=128 Kernel/4x32/avx512/kc=128"
GATE_REF_KERNEL="Kernel/6x32/avx512/kc=128"
GATE_PEAK="Peak/avx512"
# Benchmark sub-name -> audited function, so the roofline can pair each measured
# fraction with the instruction count of the loop that produced it. The peak
# kernel is the third mix, and its fraction is 1 by definition: it is the
# denominator.
GATE_KERNEL_FUNC_2x32="Kernel2x32"
GATE_KERNEL_FUNC_4x32="Kernel4x32"
GATE_PEAK_FUNC="avx512Peak"
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

# ------------------------------------------------------------- tree state (#63)
# `git status` sees uncommitted changes and nothing else. A registered worktree
# is a second checkout of another commit in this repo, invisible to that check,
# and it usually means an l1-bench.sh or layout-ensemble.sh run is in flight --
# which is exactly the condition that should stop a gate rather than an exception
# to carve out for. The tree is frozen for a measurement's life and a gate IS a
# measurement, so a gate concurrent with a benchmark was never legitimate. No
# allowlist, no exemption (ruled 2026-08-14). See worktree_strays in remote.sh.
if WORKTREE_STRAYS="$(worktree_strays)"; then
  pass "no stray git worktrees (this repo is the only registered checkout)"
else
  fail "a git worktree is registered besides this one, so either a measurement is in flight or its wreckage was left behind -- wait for it or kill it, then re-run"
  sed 's/^/        /' <<<"$WORKTREE_STRAYS"
fi

# ----------------------------------------------- the verdict's own controls
# Criterion 5b delegates to throughput_verdict in scripts/roofline.sh. Its
# adversarial fixtures run first, before any benchmarking: a decision function
# that admits a padded kernel would make every number below meaningless, and
# that is cheaper to learn now than after 40 minutes of remote benchmarks.
echo "-- throughput verdict controls (scripts/roofline-test.sh) --"
if RTLOG="$(bash scripts/roofline-test.sh 2>&1)"; then
  pass "roofline verdict controls ($(grep -c '^  ok ' <<<"$RTLOG") fixtures, incl. padded/slow/sandbagged kernels)"
else
  fail "roofline verdict controls"
  # shellcheck disable=SC2001  # prefixing every line; not a scalar substitution
  sed 's/^/        /' <<<"$RTLOG"
fi
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
# The audit logs outlive $LOG: the roofline branch of criterion 5b reads its
# instruction counts back out of them, long after $LOG has been reused.
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"
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
  unmeasured "P2 needs an amd64 host to execute the AVX-512 kernel and none is configured, so the kernel is unmeasured on real silicon"
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
      unmeasured "[$host] unreachable, so this target produced no reading"
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
     -pkg "$KERN_PKG" -func "$KERN_FUNCS" -mode spill -ssa "$SSADIR" >"$AUDITKERN" 2>&1; then
  cp "$AUDITKERN" "$LOG"
  sed 's/^/        /' "$LOG"
  pass "0 accumulator spills in the steady-state K-loop"
  pass "0 calls in the steady-state K-loop (no write barrier, no runtime helper)"
  pass "0 surviving bounds checks in the steady-state K-loop (panels are pre-sliced)"
else
  cp "$AUDITKERN" "$LOG"
  sed 's/^/        /' "$LOG"
  fail "spill audit: before touching this, read the go/no-go protocol in CLAUDE.md"
fi
for f in ${KERN_FUNCS//,/ }; do
  if [[ -s "$SSADIR/$f.html" ]]; then
    info "ssa.html archived: $SSADIR/$f.html ($(wc -c <"$SSADIR/$f.html" | tr -d ' ') bytes; gitignored, see KERNEL.md)"
  else
    unmeasured "no ssa.html archived for $f — the 'why' behind any spill would be unavailable, so this audit has nothing to read"
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
     -pkg ./internal/vec -func "$PEAK_FUNCS" -mode nomemory >"$AUDITPEAK" 2>&1; then
  cp "$AUDITPEAK" "$LOG"
  sed 's/^/        /' "$LOG"
  pass "every peak kernel's steady-state loop is register-only"
else
  cp "$AUDITPEAK" "$LOG"
  sed 's/^/        /' "$LOG"
  fail "a peak kernel's loop touches memory; the P2 denominator is not a ceiling"
fi

# ----------------------------------------- the throughput floor (§4/P2, #19)
echo
echo "-- microkernel vs measured peak (FMA-bound: >= 55%; issue-bound: >= 90% of roofline) --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; numerator and denominator measured in the same run"

# Instruction counts for the roofline, from the audits above. Missing counts are
# not a silent fallback to the flat floor: without them the host cannot be
# classified, so the gate says so and fails rather than guessing.
IPF_2x32="$(audit_ipf "$GATE_KERNEL_FUNC_2x32" "$AUDITKERN")"
IPF_4x32="$(audit_ipf "$GATE_KERNEL_FUNC_4x32" "$AUDITKERN")"
IPF_PEAK="$(audit_ipf "$GATE_PEAK_FUNC" "$AUDITPEAK")"
if [[ -n "$IPF_2x32" && -n "$IPF_4x32" && -n "$IPF_PEAK" ]]; then
  info "audited insns/FMA: 2x32 $(printf '%.3f' "$IPF_2x32"), 4x32 $(printf '%.3f' "$IPF_4x32"), $GATE_PEAK_FUNC $(printf '%.3f' "$IPF_PEAK") (roofline inputs)"
else
  unmeasured "could not read insns/FMA from the audits (2x32='$IPF_2x32' 4x32='$IPF_4x32' peak='$IPF_PEAK'), so hosts cannot be classified"
fi
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

PERF_GOV_HOST=""
N_CLEARED=0
if [[ -z "$HOSTS" ]]; then
  unmeasured "the 55% criterion needs an amd64 host and none is configured, so it is unmeasured rather than missed"
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
      unmeasured "[$host] the benchmark run failed, so this host's percent-of-peak is unmeasured rather than below the floor"
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
    BEST_LO=""; BEST_PT=""; BEST_ID=""; BEST_IPF=""; UNBOUNDED=""; MEASURED=""
    # Mixes, collected as ID:f:I. p_i = f_i * I_i is the observed retirement rate
    # divided by the host's FMA/cycle, and that common factor cancels out of both
    # the classifier and the roofline, so no clock measurement is needed (see
    # ROOF_FLOOR above). They are reduced to spreads only after the winner is
    # known, because the ceiling must be established by the mixes OTHER than the
    # shape under test — otherwise attain >= 1/cspread holds identically and the
    # roofline floor decides nothing (scripts/roofline.sh, INDEPENDENCE).
    MIXES=""
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
      case "$kname" in
        *2x32*) kipf="$IPF_2x32" ;;
        *4x32*) kipf="$IPF_4x32" ;;
        *)      kipf="" ;;
      esac
      info "[$host] ${kname##Kernel/} $(bench_describe "$kname" "$BENCHCSV" GFLOP/s) = $(awk -v r="$kpt" 'BEGIN{printf "%.1f", r * 100}')% of peak, $(awk -v r="$klo" 'BEGIN{printf "%.1f", r * 100}')% net of CI"
      [[ -n "$kipf" ]] && MIXES="$MIXES ${kname##Kernel/}|$kpt:$kipf"
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_ID="${kname##Kernel/}"; BEST_IPF="$kipf"
      fi
    done
    # The peak kernel is always a ceiling mix, and its fraction is 1 by
    # definition: it *is* the denominator. It is never the shape under test, so it
    # can never be excluded below — which is what pins the ceiling from beneath
    # and makes sandbagging an alternate shape break convergence instead of
    # lowering the roofline.
    [[ -n "$IPF_PEAK" ]] && MIXES="$MIXES $GATE_PEAK_FUNC|1.0:$IPF_PEAK"

    if [[ -z "$MEASURED" ]]; then
      info "[$host] no avx512 kernel sub-benchmark (CPU lacks it); not counted"
      continue
    fi
    if [[ -z "$BEST_LO" ]]; then
      unmeasured "[$host] no bounded percent-of-peak: benchstat established no confidence interval for${UNBOUNDED}"
      continue
    fi
    frac="$(awk -v r="$BEST_LO" 'BEGIN{printf "%.1f", r * 100}')"
    fracpt="$(awk -v r="$BEST_PT" 'BEGIN{printf "%.1f", r * 100}')"

    # --- classify the host and judge it (criterion 5b) ---------------------
    # The whole decision is scripts/roofline.sh's throughput_verdict, whose
    # adversarial fixtures ran at the top of this gate. Nothing is re-derived
    # here; this block only builds its inputs and renders its output.
    #
    # The ceiling set is every mix EXCEPT the shape under test. Excluding it is
    # what makes the roofline floor an independent measurement rather than an
    # algebraic restatement of the convergence classifier.
    CEIL=""
    for mx in $MIXES; do
      [[ "${mx%%|*}" == "$BEST_ID" ]] && continue
      CEIL="$CEIL ${mx#*|}"
    done
    if [[ -z "$BEST_IPF" ]]; then
      info "[$host] no audited insns/FMA for $BEST_ID; the host cannot be classified"
    fi
    # shellcheck disable=SC2086  # CEIL is a deliberate list of f:I words
    read -r CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY <<<"$(
      throughput_verdict "$BEST_LO" "${BEST_IPF:-0}" \
        "$PEAK_FLOOR" "$ROOF_FLOOR" "$ISSUE_CONVERGE_MAX" \
        "$ISSUE_MIX_SPREAD_MIN" "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" $CEIL)"
    csx="$(printf '%.3f' "$CSPREAD")"; msx="$(printf '%.3f' "$MSPREAD")"
    roofpc="$(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r * 100}')"
    attpc="$(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a * 100}')"
    floorpc="$(awk -v r="$ROOF" -v f="$ROOF_FLOOR" 'BEGIN{printf "%.1f", r * f * 100}')"
    slackpc="$(awk -v s="$ROOF_SHAPE_SLACK" 'BEGIN{printf "%.0f%%", (s-1)*100}')"

    case "$WHY" in
      nomixes)
        info "[$host] fewer than two ceiling mixes once $BEST_ID is excluded; no ceiling can be established, holding to the flat floor" ;;
      diverge)
        info "[$host] ceiling mixes disagree on retirement rate (${csx}x, over $ISSUE_CONVERGE_MAX) -> fma-bound: the front end is not the limit here" ;;
      samemix)
        info "[$host] ceiling mixes agree to ${csx}x but their insns/FMA differ by only ${msx}x (under $ISSUE_MIX_SPREAD_MIN), so the agreement is not evidence -> fma-bound" ;;
      falsified)
        info "[$host] ceiling mixes converge (${csx}x over a ${msx}x spread) but $BEST_ID retires *above* the ceiling they set — ${attpc}% of a ${roofpc}% roofline — so the issue-bound hypothesis is falsified by its own data -> fma-bound" ;;
      *)
        info "[$host] ceiling mixes converge ${csx}x over a ${msx}x spread in insns/FMA -> ${CLASS}-bound" ;;
    esac

    case "$CLASS/$RESULT" in
      fma/pass)
        pass "[$host] FMA-bound: best shipped shape $BEST_ID at ${fracpt}% of measured peak, ${frac}% net of CI (>= 55%)"
        N_CLEARED=$((N_CLEARED + 1))
        [[ "$gov" == "performance" ]] && PERF_GOV_HOST="$host" ;;
      fma/fail)
        fail "[$host] FMA-bound: best shipped shape $BEST_ID at ${fracpt}% of measured peak, only ${frac}% net of CI (< 55%)" ;;
      issue/refuse)
        fail "[$host] roofline refused: $BEST_ID at $(printf '%.3f' "${BEST_IPF:-0}") insns/FMA is more than $slackpc above the sweep best $SWEEP_BEST_IPF (KERNEL.md §3), so a roofline from its own instruction count would be self-serving; judged on nothing" ;;
      issue/pass)
        info "[$host] shape guard: $BEST_ID at $(printf '%.3f' "$BEST_IPF") insns/FMA, within $slackpc of the sweep best $SWEEP_BEST_IPF"
        info "[$host] issue roofline for $BEST_ID = ${roofpc}% of measured peak, so the effective floor here is ${floorpc}% (the flat 55% is unreachable: the front end cannot deliver this mix that fast)"
        pass "[$host] issue-bound: $BEST_ID at ${fracpt}% of peak (${frac}% net of CI) = ${attpc}% of its ${roofpc}% issue roofline (>= 90%)"
        N_CLEARED=$((N_CLEARED + 1))
        [[ "$gov" == "performance" ]] && PERF_GOV_HOST="$host" ;;
      issue/fail)
        info "[$host] issue roofline for $BEST_ID = ${roofpc}% of measured peak, effective floor ${floorpc}%"
        fail "[$host] issue-bound: $BEST_ID at ${frac}% net of CI = only ${attpc}% of its ${roofpc}% issue roofline (< 90%)" ;;
      *)
        fail "[$host] unclassifiable throughput verdict '$CLASS/$RESULT' (why=$WHY)" ;;
    esac
  done <<<"$HOSTS"

  if [[ "$N_CLEARED" -eq 0 ]]; then
    fail "no host cleared the throughput floor"
  fi
  if [[ -n "$PERF_GOV_HOST" ]]; then
    pass "the throughput floor was cleared under the performance governor ($PERF_GOV_HOST)"
  else
    fail "no host cleared the floor under the performance governor (§5.4 rule 5 requires one)"
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
