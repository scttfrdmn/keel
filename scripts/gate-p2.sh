#!/usr/bin/env bash
# Gate P2 — DESIGN.md §4/P2. Exits 0 only when every criterion for the phase
# holds; a red gate blocks the next phase, and there is no override flag.
#
# A GO/NO-GO, NOT A HURDLE (CLAUDE.md): if this stays red after the documented
# kernel-shaping steps and one tile shrink, the deliverable is
# docs/spill-report.md and a blocked issue, never a weakened check.
#
# The 145 lines of front matter that used to sit here — what each criterion
# measures, what this gate refuses to decide, and the judgement call behind
# every threshold below — moved verbatim to docs/gates.md, section "P2", on
# 2026-08-16. Nothing was summarised and no criterion renumbered; the move was
# for size, scripts/ having reached 1.61x the shipping library with these four
# headers its largest single lever. The thresholds themselves still
# live only here, in code.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh
# shellcheck source=scripts/roofline.sh
source scripts/roofline.sh

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

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

# ------------------------------------------- the instrument exercise (2026-08-16)
# KEEL_INSTRUMENT_EXERCISE=<reason> marks this run as synthetic. It exists so that
# criterion 5b's "2 judged, 1 not" aggregate can be driven on purpose: on a healthy
# fleet that branch never executes, and its first execution should not be the day a
# host is actually down.
#
# WHAT IT DROVE, AND WHAT IT NOW DRIVES. As written this targeted the PASS reading
# "every host that produced a judgeable throughput reading cleared its floor (2/2)"
# while a third host produced none -- a green line crediting two thirds of the fleet.
# The exercise printed exactly that, the reading of it became #90, and the ruling of
# 2026-08-16 deleted the branch: a partial fleet now resolves to UNMEASURED (see the
# aggregate below). So the target is the UNMEASURED that replaced it, and seeing the
# old PASS again would be a finding rather than a success. The branch is still one no
# complete fleet can reach, and UNMEASURED still blocks green, so what the exercise is
# for has not changed -- only which rendering it is aimed at.
#
# WHAT THIS VARIABLE DOES: sets VERDICT_STAMP, prints this banner, and withholds
# the verdict with its own exit code. That is the whole list.
#
# WHAT IT CANNOT DO, BY CONSTRUCTION: induce the branch. It is read at exactly the
# three places above and nowhere near a comparison, a threshold, a tally, or a host
# list. Set it on a healthy fleet and the run reports 3/3 with a stamp on it --
# which is worth knowing, because it means the aggregate cannot be forged from a
# flag. The induction has to come from OUTSIDE: scripts/exercise-dead-host.sh puts
# an `ssh` shim on PATH that refuses one host, so the gate finds that host
# unreachable through its ordinary machinery, from its natural cause, while the
# other two genuinely measure. A knob in the judging code would have proven only
# that the knob works (ruled 2026-08-16).
#
# Artifact discipline (#78): the log belongs at build/instrument-exercise-*, and
# the verdict is WITHHELD rather than computed, because what a reference-hungry
# reader greps for is the last line -- a synthetic run able to print "gate-p2:
# GREEN" would be a forgeable certificate no matter what this banner said.
INSTRUMENT_EXERCISE="${KEEL_INSTRUMENT_EXERCISE:-}"
if [[ -n "$INSTRUMENT_EXERCISE" ]]; then
  VERDICT_STAMP="[synthetic] "
  echo "  ############################################################"
  echo "  ##  SYNTHETIC RUN -- NOT A GATE RESULT                    ##"
  echo "  ##  reason: $INSTRUMENT_EXERCISE"
  echo "  ##  Every verdict line below carries a [synthetic] stamp. ##"
  echo "  ##  No GREEN and no RED is printed; the exit code is 2.   ##"
  echo "  ##  This run judges the instrument, never P2.             ##"
  echo "  ############################################################"
  echo
fi

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
# The ledger of what this gate trusts rather than checks (#73 tier C, ruled
# 2026-08-15). Declared here, where the fleet is named; printed beside the
# verdict by assumed_ledger below.
assume_fleet "$HOSTS"
require_disk

# ---- the measurement precondition, asserted rather than assumed (#77)
#
# DESIGN.md §5 rule 5 as amended: EVERY measuring host's clock established stable,
# per host, never satisfied by one host on behalf of another. Rule 5 picks the
# instrument by what the host has; on a host with a readable `cpufreq` — every host
# this gate has run against — that is the `performance` governor, so the assertion
# below is the rule as it applies here rather than the rule entire. The guest branch
# (`BenchmarkPeak` at head/middle/tail) is not yet in this harness; a guest blocks as
# `unmeasured` in assert_governor instead of greening unasserted.
# This gate used to read the governor, print it as `info`, and assert only that
# *some* host had cleared the floor under `performance`. Both archived green
# gate-p1 logs on #2 show what that permits: a host on `powersave` whose rate is
# published, with the criterion satisfied by a different machine (#79). The
# check itself now lives in remote.sh's assert_governor: it was a copy of gate-p4's,
# and four such copies shared a mislabel none of them could reveal (#83).
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    assert_governor "$host" preamble
  done <<<"$HOSTS"
fi

AVX512_GREEN=""
AVX512_SEEN=0
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
    # Coverage state for the aggregate below (#73's tier C): the discriminator
    # between a fleet that could look and came back short, and a fleet with
    # nothing to ask.
    #
    # THE SAME MARKER, WITHOUT THE OK==0 CONDITION, and that is deliberate rather
    # than a copy of the line above. This gate has no separate availability
    # marker the way gate-p1 does (`keel-l1-available`); I wrote one into a first
    # draft of this block from the analogy and it does not exist. What makes the
    # single marker sufficient here is a fact about the library, checked rather
    # than assumed: kern.Backends() enumerates Kernels(), and vectorKernels()
    # returns nil unless vec.HasAVX512() (internal/kern/kern_amd64.go:34-37). So
    # `avx512` appearing in this marker at all means the host has the ISA, and
    # AVX512_GREEN's extra OK==0 is what separates "had it and the run failed"
    # from "had it and the run was green".
    if grep -qE 'keel-kern-backends-exercised:.*(^| )avx512( |$)' "$LOG"; then
      AVX512_SEEN=$((AVX512_SEEN + 1))
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "kernel tests green with the avx512 tile exercised (target: $AVX512_GREEN)"
  elif [[ "$AVX512_SEEN" -gt 0 ]]; then
    fail "no target ran the kernel tests green with the avx512 tile, though $AVX512_SEEN target(s) reported it available"
  else
    unmeasured "no target reported the avx512 tile available, so whether the kernel tests pass on it is unmeasured rather than short: there was no host to ask"
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

# Coverage state, not a host name (#77 and #73's tier C). N_JUDGED counts hosts
# that produced a judgeable throughput verdict at all; N_CLEARED how many of
# those cleared their floor. Two counters because "nobody cleared the floor" and
# "nobody produced a reading" are different facts.
#
# NHOSTS is the third, and this file had no equivalent of it until #90: both counters
# above are grown inside the loop by hosts that answered, so every fraction they form
# has the survivors as its denominator. With one host of three unreachable the
# aggregate printed "(2/2)", which is true and reads as the whole fleet. The dead-host
# instrument exercise is what printed it; five green runs could not, because a complete
# fleet renders that branch identically either way.
# N_INDET is the fourth, added with #90's ruling: criterion 6 counts its indeterminate
# hosts explicitly so the aggregate can say WHICH failure to measure happened, and the
# shared decision takes it. Without it, "classification undecidable" and "host never
# answered" would collapse into one leftover and the log would not say which.
N_JUDGED=0
N_CLEARED=0
N_INDET=0
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"
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
    # Re-read at the moment of measurement, not merely in the preamble above: a
    # governor that changed in between belongs to a machine somebody started
    # using, and the reading it produces is not one §5 rule 5 covers. Silent on
    # success — the preamble printed the PASS — and the provenance info line is
    # therefore the only record of what was read on a passing host.
    assert_governor "$host" measured
    clock_gate "$host" || continue
    clock_head "$host" "$BENCHBIN" || continue
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$BENCH_FILTER" \
         >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the benchmark run failed, so this host's percent-of-peak is unmeasured rather than below the floor"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    # Before any criterion reads this sweep: on a host with no governor, the clock the
    # sweep ran on is established here or not at all (§5 rule 5, #23).
    clock_post "$host" "$BENCHBIN" "$BENCHCSV" || continue

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
    BEST_LO=""; BEST_PT=""; BEST_HI=""; BEST_ID=""; BEST_IPF=""; UNBOUNDED=""; MEASURED=""
    # Mixes, collected as ID:f:I:f_lo:f_hi. p_i = f_i * I_i is the observed
    # retirement rate divided by the host's FMA/cycle, and that common factor
    # cancels out of both the classifier and the roofline, so no clock measurement
    # is needed (see ROOF_FLOOR above). They are reduced to spreads only after the
    # winner is known, because the ceiling must be established by the mixes OTHER
    # than the shape under test — otherwise attain >= 1/cspread holds identically
    # and the roofline floor decides nothing (scripts/roofline.sh, INDEPENDENCE).
    #
    # Each mix carries both ends of its interval, because the spread it feeds is a
    # class-selecting comparison and #86 grades those three ways. The bounds are
    # measurements (bench_ratio_lo / bench_ratio_hi), not a symmetry assumption
    # about the point estimate.
    MIXES=""
    for kname in $GATE_KERNELS; do
      [[ -n "$(bench_stat "$kname" "$BENCHCSV" GFLOP/s)" ]] || continue
      MEASURED=yes
      klo="$(bench_ratio_lo "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      kpt="$(bench_ratio "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      khi="$(bench_ratio_hi "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      # Both ends or neither: a fraction bounded below but not above cannot take
      # part in a ceiling, and defaulting its upper end to the point estimate
      # would narrow an interval that was never measured.
      if [[ -z "$klo" || -z "$khi" ]]; then
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
      [[ -n "$kipf" ]] && MIXES="$MIXES ${kname##Kernel/}|$kpt:$kipf:$klo:$khi"
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_HI="$khi"; BEST_ID="${kname##Kernel/}"; BEST_IPF="$kipf"
      fi
    done
    # The peak kernel is always a ceiling mix, and its fraction is 1 by
    # definition: it *is* the denominator. It is never the shape under test, so it
    # can never be excluded below — which is what pins the ceiling from beneath
    # and makes sandbagging an alternate shape break convergence instead of
    # lowering the roofline.
    #
    # It carries no interval, and that is not an omission: f = 1 exactly, and the
    # peak's own confidence interval is already inside every other mix's bounds,
    # which are ratios against it. Giving it one would double-count the same noise.
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
    # shellcheck disable=SC2086  # CEIL is a deliberate list of f:I:f_lo:f_hi words
    read -r CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY CSLO CSHI ATTAINHI <<<"$(
      throughput_verdict "$BEST_LO" "$BEST_HI" "${BEST_IPF:-0}" \
        "$PEAK_FLOOR" "$ROOF_FLOOR" "$ISSUE_CONVERGE_MAX" \
        "$ISSUE_MIX_SPREAD_MIN" "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" $CEIL)"
    csx="$(printf '%.3f' "$CSPREAD")"; msx="$(printf '%.3f' "$MSPREAD")"
    cslox="$(printf '%.3f' "$CSLO")"; cshix="$(printf '%.3f' "$CSHI")"
    atthipc="$(awk -v a="$ATTAINHI" 'BEGIN{printf "%.1f", a * 100}')"
    roofpc="$(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r * 100}')"
    attpc="$(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a * 100}')"
    floorpc="$(awk -v r="$ROOF" -v f="$ROOF_FLOOR" 'BEGIN{printf "%.1f", r * f * 100}')"
    slackpc="$(awk -v s="$ROOF_SHAPE_SLACK" 'BEGIN{printf "%.0f%%", (s-1)*100}')"

    # INVARIANT, and it has drifted twice: every branch below publishes the measured
    # interval alongside the point estimate, or states why it has none. §5 rule 8's
    # publish-the-pair clause has no per-branch exemption, and the branch a reader
    # reaches is exactly the branch whose numbers they need — a pair present in eight
    # renderings and absent in the ninth is worse than uniform absence, because it
    # reads as "this one had no interval" rather than "this one forgot". `nomixes` is
    # the only true exemption and says so in its own text.
    case "$WHY" in
      nomixes)
        # The one branch with no interval to publish, and it is exempt for a reason
        # rather than by oversight: no ceiling was established, so there is no
        # ceiling-spread reading to bracket. Naming that here keeps the exemption
        # auditable — a future reader comparing branches sees a stated reason
        # instead of the gap §5 rule 8's publish-the-pair clause exists to close.
        info "[$host] fewer than two ceiling mixes once $BEST_ID is excluded; no ceiling can be established (so there is no spread to bracket), holding to the flat floor" ;;
      diverge)
        # The interval prints here too. This branch used to report the point spread
        # alone, and it is the branch the 2026-08-15 incident came out of — the one
        # place a reader most needs to see how far the whole reading sits from the
        # bar. §5 rule 8's publish-the-pair clause has no per-branch exemption.
        info "[$host] ceiling mixes disagree on retirement rate (${csx}x, interval [${cslox}x, ${cshix}x], wholly over $ISSUE_CONVERGE_MAX) -> fma-bound: the front end is not the limit here" ;;
      falsifiedanyway)
        info "[$host] the ceiling mixes' rate spread interval [${cslox}x, ${cshix}x] crosses the $ISSUE_CONVERGE_MAX bar, so whether they converge is undecided — but $BEST_ID retires at an attainment interval of [${attpc}%, ${atthipc}%] of the ${roofpc}% roofline the converged reading would imply, above the ceiling at every point of that interval, so both branches say fma-bound and the class is decided anyway" ;;
      samemixanyway)
        info "[$host] the ceiling mixes' rate spread interval [${cslox}x, ${cshix}x] crosses the $ISSUE_CONVERGE_MAX bar, but their insns/FMA differ by only ${msx}x (under $ISSUE_MIX_SPREAD_MIN), so no ceiling is established whether they converge or not -> fma-bound, decided anyway" ;;
      samemix)
        info "[$host] ceiling mixes agree to ${csx}x (interval [${cslox}x, ${cshix}x]) but their insns/FMA differ by only ${msx}x (under $ISSUE_MIX_SPREAD_MIN), so the agreement is not evidence -> fma-bound" ;;
      falsified)
        info "[$host] ceiling mixes converge (${csx}x, interval [${cslox}x, ${cshix}x], over a ${msx}x spread) but $BEST_ID retires *above* the ceiling they set — attainment interval [${attpc}%, ${atthipc}%] of a ${roofpc}% roofline — so the issue-bound hypothesis is falsified by its own data -> fma-bound" ;;
      nearconverge)
        info "[$host] the ceiling mixes' rate spread is ${csx}x with an interval of [${cslox}x, ${cshix}x], which crosses the $ISSUE_CONVERGE_MAX bar: whether this host has a front-end ceiling is not decidable from this run's measurements" ;;
      nearceiling)
        info "[$host] ceiling mixes converge (${csx}x, interval [${cslox}x, ${cshix}x], over a ${msx}x spread), but $BEST_ID's attainment interval reaches [${attpc}%, ${atthipc}%] of the ${roofpc}% roofline, crossing 100%: whether the machine exceeds the ceiling its own mixes set is not decidable from this run" ;;
      *)
        info "[$host] ceiling mixes converge ${csx}x (interval [${cslox}x, ${cshix}x], clear of $ISSUE_CONVERGE_MAX) over a ${msx}x spread in insns/FMA -> ${CLASS}-bound" ;;
    esac

    # A classification the run could not make is not a floor this host missed
    # (#86). It comes before N_JUDGED because the aggregate below counts hosts that
    # produced a judgeable reading, and this host did not: no class means no floor
    # to judge against. One cause, one label — and the label names the cause rather
    # than reporting a miss against whichever floor the noise happened to select.
    if [[ "$CLASS" == indeterminate ]]; then
      unmeasured "[$host] classification indeterminate this run (why=$WHY), so P2's floor is unmeasured here rather than missed — $BEST_ID read ${fracpt}% of measured peak, ${frac}% net of CI, and that reading is not in question"
      # Deliberately NOT "spend DESIGN.md §4's one archived re-run". That allowance
      # lets a red verdict be overturned by a second reading, and there is no verdict
      # here to overturn — so re-measuring an UNMEASURED is the ordinary response to a
      # missing measurement, uncapped, rather than an allowance being drawn down.
      info "  [$host] the remedy is precision, never a wider bar: re-measure (this is not §4's re-run being spent — there is no verdict to overturn), and if this host is chronically indeterminate here, raise KEEL_BENCH_COUNT for it"
      # Counted, then skipped. Before #90 this branch continued without counting, so
      # the aggregate could see only survivors and had no way to distinguish "the run
      # could not classify this host" from "this host never answered" -- both were the
      # same invisible leftover. The aggregate names them separately now, which it can
      # only do if the two are tallied separately here.
      N_INDET=$((N_INDET + 1))
      continue
    fi

    # Every host reaching here produced a judgeable verdict, including
    # issue/refuse: a refusal is a reading this gate took and reasoned about.
    N_JUDGED=$((N_JUDGED + 1))
    case "$CLASS/$RESULT" in
      fma/pass)
        pass "[$host] FMA-bound: best shipped shape $BEST_ID at ${fracpt}% of measured peak, ${frac}% net of CI (>= 55%)"
        N_CLEARED=$((N_CLEARED + 1)) ;;
      fma/fail)
        fail "[$host] FMA-bound: best shipped shape $BEST_ID at ${fracpt}% of measured peak, only ${frac}% net of CI (< 55%)" ;;
      issue/refuse)
        fail "[$host] roofline refused: $BEST_ID at $(printf '%.3f' "${BEST_IPF:-0}") insns/FMA is more than $slackpc above the sweep best $SWEEP_BEST_IPF (KERNEL.md §3), so a roofline from its own instruction count would be self-serving; judged on nothing" ;;
      issue/pass)
        info "[$host] shape guard: $BEST_ID at $(printf '%.3f' "$BEST_IPF") insns/FMA, within $slackpc of the sweep best $SWEEP_BEST_IPF"
        info "[$host] issue roofline for $BEST_ID = ${roofpc}% of measured peak, so the effective floor here is ${floorpc}% (the flat 55% is unreachable: the front end cannot deliver this mix that fast)"
        pass "[$host] issue-bound: $BEST_ID at ${fracpt}% of peak (${frac}% net of CI) = ${attpc}% of its ${roofpc}% issue roofline (>= 90%)"
        N_CLEARED=$((N_CLEARED + 1)) ;;
      issue/fail)
        info "[$host] issue roofline for $BEST_ID = ${roofpc}% of measured peak, effective floor ${floorpc}%"
        fail "[$host] issue-bound: $BEST_ID at ${frac}% net of CI = only ${attpc}% of its ${roofpc}% issue roofline (< 90%)" ;;
      *)
        fail "[$host] unclassifiable throughput verdict '$CLASS/$RESULT' (why=$WHY)" ;;
    esac
  done <<<"$HOSTS"

  # The aggregate, over ONE coverage decision shared with criterion 6 (`fleet_coverage`,
  # #90's ruling of 2026-08-16). Every host that got this far is under `performance`,
  # asserted twice above, so this line no longer names one host on the fleet's behalf.
  #
  # The ruling: a PASS reading "(2/2)" over a three-host fleet is a message-level truth
  # carrying a fleet-level assertion, and a fleet with an absent member has not measured
  # a claim about the fleet. So a partial fleet resolves to UNMEASURED here, exactly as
  # criterion 6's OpenBLAS aggregate already resolves it -- §5 rule 6 giving the absent
  # measurement its one available verdict. Two aggregates with different absence
  # semantics would be the divergent-copies defect relocated to the verdict layer, which
  # is what `fleet_shortfall` had just been built to end, so both now call one function.
  #
  # NOT A WEAKENING, and the same argument as in roofline.sh's header: `unmeasured` sets
  # FAIL=1, so UNMEASURED blocks green identically to FAIL. Nothing that was red can
  # become green by this change; a partial fleet simply stops being *describable* as a
  # whole one. Per-host PASSes above stay as measured -- only the aggregate refuses.
  #
  # The pass branch's fraction is over NHOSTS, not N_JUDGED: `fleet_coverage` returns
  # `pass` only when N_CLEARED == NHOSTS, so on that branch the two are equal and the
  # denominator is the fleet either way. Naming NHOSTS makes the claim's scope explicit
  # instead of leaving it to coincide, and `fleet_shortfall` stays appended as a
  # fail-closed belt: if the two ever diverge, the line says so rather than reading whole.
  N_NOCOVER=$((NHOSTS - N_JUDGED - N_INDET))
  case "$(fleet_coverage "$NHOSTS" "$N_JUDGED" "$N_CLEARED" "$((N_JUDGED - N_CLEARED))" "$N_INDET")" in
    unmeasured)
      # Two ways to land here, and they are not the same fact: no host produced a
      # judgeable reading, or the fleet's configured size could not be read (in which
      # case there is no denominator to judge coverage against at all).
      if [[ "$N_JUDGED" -eq 0 ]]; then
        unmeasured "no host produced a judgeable throughput reading, so the floor is unmeasured rather than uncleared$(fleet_shortfall "$NHOSTS" 0)"
      else
        unmeasured "$N_CLEARED of $N_JUDGED judged hosts cleared their floor, but this run could not read how many hosts were configured, so it cannot say whether that is the fleet$(fleet_shortfall "$NHOSTS" "$N_JUDGED")"
      fi ;;
    pass)
      pass "every one of the $NHOSTS configured gate hosts cleared its floor ($N_CLEARED/$NHOSTS), each under the performance governor asserted per host (§5 rule 5)$(fleet_shortfall "$NHOSTS" "$N_JUDGED")" ;;
    fail)
      fail "$((N_JUDGED - N_CLEARED)) of $N_JUDGED hosts that produced a judgeable throughput reading did not clear the floor (the per-host lines above say which)$(fleet_shortfall "$NHOSTS" "$N_JUDGED")" ;;
    *)
      # `partial`: nobody measured below the floor, but the fleet is not covered. The
      # counts are named separately because they are different failures to measure --
      # a host the run could not classify is not a host that never answered.
      unmeasured "the fleet's floor is unmeasured: $N_CLEARED of $NHOSTS configured hosts cleared it and none measured below it, but $((NHOSTS - N_CLEARED)) produced no floor verdict ($N_INDET indeterminate, $N_NOCOVER with no judgeable reading at all), and a fleet with an absent member has not measured a claim about the fleet (#90) — the per-host PASSes above stand as measured" ;;
  esac
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
echo
# An instrument exercise prints neither colour, exactly as gate-p3.sh does for
# KEEL_INSTRUMENT_WIDEN_CI and for the reason given there: the last line is what a
# reader greps, so a synthetic run must not be able to emit one that reads as a
# certificate. Exit 2 keeps `detach.sh stat` from recording it as either a pass or a
# failure of P2. FAIL is reported as a fact about which renderings fired.
if [[ -n "$INSTRUMENT_EXERCISE" ]]; then
  echo "gate-p2: VERDICT WITHHELD (instrument exercise, KEEL_INSTRUMENT_EXERCISE=$INSTRUMENT_EXERCISE; FAIL=$FAIL says which renderings fired, not whether P2 holds)"
  exit 2
fi
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p2: GREEN"
  exit 0
fi
echo "gate-p2: RED" >&2
echo "P2 is a go/no-go: if this is still red after the documented shaping steps" >&2
echo "and one tile shrink, write docs/spill-report.md and stop (CLAUDE.md)." >&2
exit 1
