#!/usr/bin/env bash
# Gate P3 — DESIGN.md §4/P3. Exits 0 only when every criterion for the phase
# holds; a red gate blocks the next phase, and there is no override flag.
#
# The 244 lines of front matter that used to sit here — what each criterion
# measures, what this gate refuses to decide, and the judgement call behind
# every threshold below — moved verbatim to docs/gates.md, section "P3", on
# 2026-08-16. Nothing was summarised and no criterion renumbered; the move was
# for size, scripts/ having reached 1.61x the shipping library with these four
# headers its largest single lever. The thresholds themselves still
# live only here, in code — including those carried from gate-p2, whose
# duplication is deliberate and argued at the constant.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh
# shellcheck source=scripts/roofline.sh
source scripts/roofline.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

# ------------------------------------------- the instrument exercise (#86)
# KEEL_INSTRUMENT_WIDEN_CI=<pct> widens every CEILING-SET mix's fraction-of-peak
# interval to at least +/-pct around its own point estimate, before
# classification. It exists to DRIVE the three-state verdict renderings on real
# silicon (authorized 2026-08-16 on #86), because on this fleet they are
# otherwise unreachable: janus's class-selecting interval is zero-width, so no
# value of the 1.10 bar can sit inside it, and antares's attainment is 162%,
# nowhere near 1. A rendering that has never executed is an untested branch in
# the instrument that issues the certificates.
#
# CORRECTION, 2026-08-19 (grounds: issue #110, DESIGN.md §5 rule 15). The
# sentence above is left as written and is wrong on one word: janus's interval
# is not zero-width and that was never a measurement. It read width 0 off a
# reported `+/- 0%`, which is the floor of benchstat's CSV reporting resolution
# ("narrower than 0.5%"), and the same rounded column decided a shipped P5
# verdict on one rounding step. The archive refutes the categorical claim on
# this very host: [1.014x, 1.034x] around 1.026 in one run, non-zero width.
# The conclusion survives WITH its denominator: 1.10 - 1.034 = 0.066 of margin
# against ~0.010 of quantization width, about six quanta -- so this knob is
# still needed, for a measured reason and not an impossibility one. The bounds
# it widens now arrive at full precision from tools/benchci, so what the knob
# simulates is a genuinely noisy host rather than a formatter's floor.
#
# What it does NOT do:
#   - It does not move a threshold. ISSUE_CONVERGE_MAX stays at its shipped 1.10
#     and prints in the banner below, because a bar nobody would ship is not a
#     simulation of anything; a host reading noisily is what happened on
#     2026-08-15, and that is what this reproduces.
#   - It does not touch the dispatched shape's own bounds, which feed the
#     result-deciding comparisons (§4's two-state boundaries). Only the ceiling
#     set moves, and the widening is monotone: an interval already wider than
#     +/-pct is left alone.
#   - It does not touch the register-only peak mix, whose f is 1 by definition
#     and whose interval is already folded into every other mix's bounds.
#   - It cannot produce a gate verdict. The run prints no GREEN and no RED, every
#     verdict line carries a [synthetic] stamp, and the exit code is its own.
# Artifact discipline (#78): the log belongs at build/instrument-exercise-*, never
# on a gate-pN-<rev> path where a reference-hungry diff could pick it up.
INSTRUMENT_WIDEN_CI="${KEEL_INSTRUMENT_WIDEN_CI:-}"
if [[ -n "$INSTRUMENT_WIDEN_CI" ]]; then
  if ! awk -v w="$INSTRUMENT_WIDEN_CI" 'BEGIN{exit !(w+0 > 0 && w+0 <= 100 && w ~ /^[0-9]+(\.[0-9]+)?$/)}'; then
    echo "gate-p3: KEEL_INSTRUMENT_WIDEN_CI must be a percentage in (0,100], got '$INSTRUMENT_WIDEN_CI'" >&2
    exit 2
  fi
  VERDICT_STAMP="[synthetic] "
fi

# instrument_widen MIX — echo a ceiling-set mix word `f:I[:f_lo[:f_hi]]`, widened
# iff the exercise is armed. A no-op otherwise, and a no-op always for a mix with
# no bounds to widen: the register-only peak enters as `1.0:I`, and its f is 1 by
# definition rather than by measurement, so there is no interval there to make
# noisy. Monotone — an end already further out than +/-pct stays where it is — so
# this can only ever widen a reading, never sharpen one.
instrument_widen() {
  local mix="$1"
  [[ -n "$INSTRUMENT_WIDEN_CI" ]] || { printf '%s' "$mix"; return; }
  awk -v mix="$mix" -v w="$INSTRUMENT_WIDEN_CI" 'BEGIN {
    n = split(mix, a, ":")
    if (n < 3 || a[3] == "") { print mix; exit }
    f = a[1] + 0; lo = a[3] + 0
    hi = (n >= 4 && a[4] != "") ? a[4] + 0 : f
    wlo = f * (1 - w / 100); whi = f * (1 + w / 100)
    if (wlo < lo) lo = wlo
    if (whi > hi) hi = whi
    printf "%s:%s:%.6f:%.6f\n", a[1], a[2], lo, hi
  }'
}

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

# ---------------------------------------------------------------- the sweep
# DESIGN.md §4/P3's list, verbatim. 1..17 covers every M and N remainder against
# any MR/NR a kernel might ship; 63/64/65 straddle a power of two; 500 and 1000
# are multiples of no blocking parameter; 2048 is the size the throughput
# criterion is stated at.
SWEEP_SIZES="1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 63 64 65 500 1000 2048"
# Sizes that must be verified against the oracle in full, not sampled.
SWEEP_EXACT_MAX=65
# Minimum sampled entries for a size above that: 256 exact float64 dot products
# spread over C by a seeded RNG.
SWEEP_SAMPLE_MIN=256
# The transpose lattice must be complete: a packing bug that transposes the wrong
# operand shows up in exactly one of these four.
SWEEP_TRANS="NN NT TN TT"

# ------------------------------------------------------ carried from P2 (#19)
# Same shapes, same audit, same pure verdict function and the same constants as
# gate-p2.sh. Duplicated rather than factored into a shared file on purpose: a
# P3 run should fail because P3 changed the kernel, not because someone edited
# one shared constant, and two independent statements of the threshold is what
# makes that visible.
KERN_PKG="./internal/vec"
KERN_FUNCS="Kernel2x32,Kernel4x32"
PEAK_FUNCS="avx512Peak,avx2Peak,scalarPeak"
PEAK_FLOOR=0.55
ROOF_FLOOR=0.90
ISSUE_CONVERGE_MAX=1.10
ISSUE_MIX_SPREAD_MIN=1.25
SWEEP_BEST_IPF=4.625
ROOF_SHAPE_SLACK=1.05
GATE_KERNELS="Kernel/2x32/avx512/kc=128 Kernel/4x32/avx512/kc=128"
GATE_PEAK_FUNC="avx512Peak"
KERN_BENCH_FILTER='Peak|Kernel/.*/.*/kc=128'
SSADIR="build/ssa"

# ------------------------------------------------------------- P3's own bar
OPENBLAS_FLOOR=0.60
GATE_SGEMM="Sgemm/n=2048"
GATE_OPENBLAS="OpenBLAS/n=2048"
GATE_PEAK="Peak/avx512"
# THE PARENTHESES ARE LOAD-BEARING (issue #32, docs/toolchain-notes.md T15).
# `go test -bench` does not match one regexp against the full name: it splits on
# top-level '|' FIRST, into an alternation of whole patterns, and only then splits
# each alternative on '/'. So the previous value here —
#   Peak|Sgemm|OpenBLAS/avx512|n=2048
# — was four alternatives, {Peak}, {Sgemm}, {OpenBLAS,avx512}, {n=2048}, and not the
# two-level filter its comment claimed. {OpenBLAS,avx512} can never match, because
# BenchmarkOpenBLAS's children are n=..., so THE REFERENCE BENCHMARK NEVER RAN and
# criterion 6 had no denominator to divide by; {Peak} and {Sgemm} were
# depth-unconstrained, so the gate also paid for three Sgemm sizes and two Peak
# variants it never reads. Parenthesized, the '|'s are ordinary regexp alternation
# inside one two-element pattern, and this runs exactly the three benchmarks the
# gate reads: Peak/avx512, Sgemm/n=2048, OpenBLAS/n=2048.
SGEMM_BENCH_FILTER='(Peak|Sgemm|OpenBLAS)/(avx512|n=2048)'
# Criterion 5b's cross-check run: the blocked Sgemm only, with the shape pinned to
# the other class. No Peak and no OpenBLAS, because what is wanted from it is one
# rate to compare against the dispatched shape's rate — not a ratio.
SGEMM_SHAPE_FILTER='Sgemm/n=2048'
OPENBLAS_REMOTE_DIR="${KEEL_OPENBLAS_DIR:-/tmp/keel-openblas-src}"

# The OpenBLAS kernel families that are AVX2-or-better on x86-64, lowercased.
# DYNAMIC_ARCH picks one at load time and reports it through
# openblas_get_corename(); anything not on this list — a generic build, a
# pre-AVX2 family, or a name added by a future OpenBLAS — fails the criterion.
#
# An allowlist rather than a denylist, deliberately, and it is the strictness that
# is the point: a reference that quietly runs a Nehalem kernel on a Skylake-X host
# is slow, and a slow reference makes keel look good. Every other failure mode of
# this gate costs a session; this one would cost the truth of a number. A new
# legitimate target name failing here costs whoever sees it one line of diff.
#
# The arm64 families (#137) are UNIONED in rather than switched by host arch, and the
# union is safe for exactly the reason the list is an allowlist and not a denylist: it
# contains no generic family, only tuned NEON/SVE kernels a Graviton reference may
# legitimately pin (armv8 is the NEON baseline OpenBLAS ships for Graviton, not a
# generic-C build; neoversen1/v1/v2 are the tuned targets). A cross-arch false accept is
# impossible -- an x86 host cannot report `neoversev2` nor an arm64 host `skylakex` -- so
# the union can only accept more legitimate corenames, never launder a slow one. The
# CEILING is still the swept fastest; this floor only rejects an ancient or generic family.
OPENBLAS_OK_CORES="haswell skylakex cooperlake sapphirerapids zen armv8 neoversen1 neoversev1 neoversev2"

# The coretype candidates the reference is swept over, and the reason the sweep
# exists at all (ruling on issue #31).
#
# The denominator is the FASTEST MEASURED same-host OpenBLAS across these, not
# whatever DYNAMIC_ARCH selected at load time. Measured on the three gate hosts:
# on vesta's Zen 4, DYNAMIC_ARCH picks an Intel-tuned Cooperlake kernel and the
# AVX2 Haswell kernel is 6.7% faster (159.5 vs 149.5 GFLOP/s), because Zen 4
# double-pumps AVX-512 through a 256-bit datapath. That is the same physics that
# makes keel dispatch 2x32 on an issue-bound host: upstream's dispatch consults an
# ISA feature bit, ours classifies the machine. Their heuristic is wrong on one of
# our three hosts, and a reference chosen by it is a denominator that flatters keel
# — the one property this denominator must never have.
#
# The allowlist above keeps its narrowed job (reject an ancient family: the FLOOR).
# The sweep supplies the CEILING. Neither substitutes for the other: an allowlist
# cannot tell "correct for this silicon" from "merely modern", and this one contains
# both the right and the wrong answer for every host here.
#
# A superset of the ruled set {default, Zen, Haswell, SkylakeX-where-valid}, because
# adding candidates can only raise the ceiling and never lower it, and because
# "where valid" is established by RUNNING each candidate rather than by consulting a
# table: one that cannot execute here is recorded as unavailable, not assumed absent.
OPENBLAS_CORETYPES="default Zen Haswell SkylakeX Cooperlake SapphireRapids"

# The arm64 candidates (#137), selected per host by `uname -m` inside ob_coretype_sweep
# rather than unioned with the x86 list: an x86 coretype forced on Graviton reports "- -"
# (unavailable) and only adds noise, and worse, mixing arches muddies the SVE≈NEON reading
# this sweep is the instrument for. ARMV8 is the NEON kernel family; NEOVERSEV1 (Graviton3)
# and NEOVERSEV2 (Graviton4) are the SVE families -- the sweep's achieved rates across these
# three ARE the SVE≈NEON comparison (docs/neon-sweep.md's fleet half). NEOVERSEN1 is included
# as the tuned-but-NEON-only middle point; a candidate the silicon cannot run is recorded
# unavailable, exactly as on x86, so listing one absent on a given Graviton costs nothing.
OPENBLAS_CORETYPES_ARM64="default ARMV8 NEOVERSEN1 NEOVERSEV1 NEOVERSEV2"

# sentinel_hosts / sentinel_declaration moved to scripts/remote.sh (#146): the set is
# now derived FROM the fleet, so it belongs beside remote_hosts, and a second consumer
# (exercise-dead-host.sh) was re-parsing the retired file to compute its own answer.

# other_class CLASS — the class criterion 5b pins KEEL_KERN_CLASS to: the one
# dispatch did not choose on this host, so the cross-check measures the shape that
# was passed over. An unrecognized class yields nothing and the caller skips the
# cross-check with a reason rather than silently comparing a shape against itself.
other_class() {
  case "$1" in
    issue) printf 'fma' ;;
    fma)   printf 'issue' ;;
  esac
}

# ob_preflight HOST — what the same-host reference needs, before trying to build it.
#
# The native build failing with a linker error and thirty lines of go tooling is a
# worse gate line than "this host has no libopenblas.so", and the two are told apart
# here rather than by reading a compiler diagnostic. `go` is looked up as ssh finds
# it: a non-interactive, non-login shell, which is exactly how the build below runs,
# so a toolchain installed only in an interactive PATH reports as missing — which is
# the truth about this gate's ability to use it.
ob_preflight() {
  ssh "${KEEL_SSH_OPTS[@]}" "$1" '
    distro=unknown
    [ -r /etc/os-release ] && distro=$(sed -n "s/^ID=//p" /etc/os-release | tr -d \")
    go=none
    command -v go >/dev/null 2>&1 && go=$(go version | cut -d" " -f3)
    lib=none
    for d in /usr/local/lib /usr/lib64 /usr/lib/x86_64-linux-gnu /usr/lib/aarch64-linux-gnu /usr/lib; do
      if [ -e "$d/libopenblas.so" ]; then lib="$d/libopenblas.so"; break; fi
    done
    printf "distro=%s go=%s lib=%s\n" "$distro" "$go" "$lib"
  ' 2>/dev/null
}

# ob_provision_help HOST DISTRO — the exact commands needed on HOST, named per
# distribution because "install OpenBLAS" is not a command anybody can run.
#
# Printed, never executed: provisioning needs sudo, the gate connects with
# BatchMode=yes, and installing software on Scott's machines is his call, not this
# script's. scripts/provision-openblas.sh is the thing he runs.
ob_provision_help() {
  case "$2" in
    ubuntu|debian|pop|linuxmint)
      info "  [$1] sudo apt-get install -y libopenblas-dev" ;;
    rhel|centos|rocky|almalinux|fedora)
      info "  [$1] sudo dnf install -y openblas-devel" ;;
    *)
      info "  [$1] install an OpenBLAS development package (unrecognized distro id '${2:-unknown}')" ;;
  esac
  info "  [$1] plus a go1.26.5+ toolchain from go.dev/dl, on PATH for a non-login ssh"
  info "  [$1] scripts/provision-openblas.sh does both, with sudo prompts the gate cannot answer"
}

# ob_coretype_sweep HOST — the reference's ceiling, measured (issue #31).
#
# Prints one line per candidate: "requested achieved GFLOP/s". A candidate that
# cannot run here reports "- -": forcing a coretype the silicon cannot execute kills
# the harness, and that is a valid answer about this host rather than an error in the
# gate. A candidate that silently falls back reports the family it actually got, so
# the achieved name — never the requested one — is what gets compared and pinned.
# A candidate whose harness ran but produced no matching row reports `norow`, which
# the caller fails on: that is a defect in this gate, not a property of the host.
#
# The rate is read ONLY from a benchmark result row whose name is exactly the one
# requested (issue #33). The first version took the maximum over every field followed
# by "GFLOP/s" on every line, which silently picked up
#   keel-bench-peak-formula: avx512: 368.9 GFLOP/s (5.76 GHz x 2 FMA ports x 16 lanes)
# — a theoretical peak, larger than any real rate, identical across candidates. All
# six candidates then tied on every host, the winner was whichever came first
# (`default`), and the sweep reported +0.0% against DYNAMIC_ARCH's own choice: the
# 6.7% finding this function exists to enforce, erased by its own parser. Requiring
# the row name closes the neighbouring hole too, where a filter runs more benchmarks
# than intended (issue #32) and a different benchmark's rate is in reach.
#
# The rate is the best of $KEEL_BENCH_COUNT readings at the same -benchtime as every
# other number here, so the winner is chosen under the ratified methodology and not
# by a quick probe. Best-of-N rather than a median because this picks a candidate
# rather than reporting a result; the number that enters the record is measured again
# below, under full methodology, with the winner pinned.
#
# KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, and the script arrives on
# stdin. $BFLAGS is expanded at call time, by which point the methodology flags are
# set — this function is defined before them and must not capture them early.
ob_coretype_sweep() {
  local host="$1"
  # shellcheck disable=SC2087  # unquoted EOF on purpose: the candidate list, the
  # remote dir, the benchmark name and $BFLAGS are this gate's own values and must
  # expand here. Everything the remote shell owns is escaped as \$.
  ssh "${KEEL_SCP_OPTS[@]}" "$host" 'bash -s' 2>/dev/null <<EOF
cd '$OPENBLAS_REMOTE_DIR' || exit 1
case "\$(uname -m)" in
  aarch64|arm64) cts='$OPENBLAS_CORETYPES_ARM64' ;;
  *)             cts='$OPENBLAS_CORETYPES' ;;
esac
for ct in \$cts; do
  if [ "\$ct" = default ]; then unset OPENBLAS_CORETYPE; else export OPENBLAS_CORETYPE="\$ct"; fi
  out="\$(env GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 ./bench-ob.test \
    -test.run=NONE -test.bench='$GATE_OPENBLAS' $(printf '%s ' "${BFLAGS[@]}") 2>&1)" || {
    printf '%s - -\n' "\$ct"
    continue
  }
  core="\$(printf '%s\n' "\$out" | sed -n 's/.*corename=\([^ ]*\).*/\1/p' | tail -1)"
  best="\$(printf '%s\n' "\$out" | awk -v want='$GATE_OPENBLAS' '
    \$1 !~ /^Benchmark/ { next }
    {
      n = \$1
      sub(/-[0-9]+\$/, "", n)
      sub(/^Benchmark/, "", n)
      if (n != want) next
      for (i = 2; i < NF; i++) if (\$(i + 1) == "GFLOP/s" && \$i + 0 > m) m = \$i + 0
    }
    END { if (m > 0) printf "%.2f", m }')"
  printf '%s %s %s\n' "\$ct" "\${core:--}" "\${best:-norow}"
done
EOF
}

echo "== gate-p3: packing + blocking -> full Sgemm =="
echo

if [[ -n "$INSTRUMENT_WIDEN_CI" ]]; then
  echo "!! INSTRUMENT EXERCISE -- THIS IS NOT A GATE RUN AND ISSUES NO VERDICT !!"
  echo "!! KEEL_INSTRUMENT_WIDEN_CI=$INSTRUMENT_WIDEN_CI: every ceiling-set mix's percent-of-peak"
  echo "!! interval is widened to at least +/-${INSTRUMENT_WIDEN_CI}% around its own point estimate before"
  echo "!! classification, to drive the three-state renderings added in #86 on real silicon."
  echo "!! The thresholds are UNTOUCHED and shipped: converge_max=$ISSUE_CONVERGE_MAX,"
  echo "!! mix_spread_min=$ISSUE_MIX_SPREAD_MIN, roof_floor=$ROOF_FLOOR, peak_floor=$PEAK_FLOOR,"
  echo "!! openblas_floor=$OPENBLAS_FLOOR. The dispatched shape's own bounds are untouched too;"
  echo "!! only the ceiling set moves. Every verdict line below is stamped [synthetic] and the"
  echo "!! run ends with no GREEN and no RED. Authorized on #86, 2026-08-16."
  echo
fi

# ------------------------------------------------------------- tree state (#63)
assert_no_strays

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

gate_tmpdir
KERNBIN="$BINDIR/kern.test"
# Criterion 5b's cross-check run, kept beside the dispatched one rather than
# overwriting it: the comparison needs both rates at once.
ALTLOG="$BINDIR/bench-alt.log"
ALTCSV="$BINDIR/bench-alt.csv"
SWEEPLOG="$BINDIR/sweep-avx512.log"
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"

# ------------------------------------------------------- Sgemm vs the oracle
echo
echo "-- Sgemm vs the float64 oracle: size sweep x transpose x alpha x beta --"
info "the local run exercises the scalar path only (darwin/arm64 has no archsimd);"
info "the sweep's extent is audited below from a host that ran it with avx512 live"

LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
test_verdict "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$LOG" "$LOCAL_OK" "all tests pass"
# The scalar path must also pass on a stock toolchain: keel is a pure-Go library
# first and an experiment second.
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
test_verdict "local, stock toolchain" "$LOG" "$STOCK_OK" "all tests pass without GOEXPERIMENT=simd"

resolve_fleet

# ---- the measurement precondition, asserted rather than assumed (ruling with #31)
#
# DESIGN.md §5 rule 5 asks for a clock established stable, which on a host with a
# readable `cpufreq` — every host this gate runs against — means the performance
# governor. Until this ruling the gate
# READ scaling_governor and used it only to label criterion 6's pass line, so a host
# on `powersave` still contributed numbers to the record. antares did exactly that:
# its first OpenBLAS reading was 245.0 GFLOP/s against a 296-297 GFLOP/s steady
# state — a ramping-core artifact, indistinguishable in a log from a slow machine,
# and precisely what rule 5 exists to exclude.
#
# Now every host must be on `performance` or the gate is red. This is the same move
# as reading `threads=` back out of the library instead of trusting the environment
# that was passed to it: a precondition that is not checked is a precondition that
# drifts, and it drifts silently in whichever direction the machine happens to be
# configured. Unreadable counts as unmet — an unverified precondition is not a met
# one, and "unknown" is the answer a missing cpufreq sysfs gives on a VM.
#
# The check itself now lives in remote.sh's assert_governor (#83). This gate's copy
# carried two of the five drifts the lift removed. One was its citation of
# `§5.4 rule 5` (citation-lint:quote), inventoried at the time as "a section
# DESIGN.md does not have". #85's audit refined that: DESIGN.md has no
# subsections, but read as the shorthand "§5, item 4" — the notation 25 other
# correctly — the form resolves, to the benchmarks-are-tests rule rather than the
# methodology rule it meant. Mis-minted, not structurally absent, and the
# distinction matters because a pin would have frozen it forever. It was also the
# only one of the five with the
# `sudo tee` remediation hint for a host that has no cpupower — the union of the
# hints is what the lifted version prints.
while read -r host; do
  assert_governor "$host" preamble
  admission_readback "$host" "$GOV_PROV"
done < <(hosts_lines)

AVX512_GREEN=""
SCALAR_FORCED=""
# Coverage state for the two aggregates at the end of this section (#73's tier C).
# AVX512_SEEN counts hosts whose sweep reported the avx512 backend exercised at
# all, which is a capability witness rather than a second marker: gemmRunners()
# appends one runner per kern.Kernels() entry (gemm_test.go:96-106) and
# vectorKernels() returns nil unless vec.HasAVX512() (internal/kern/kern_amd64.go
# :34-37), so `avx512` cannot appear in keel-sgemm-backends-exercised on a host
# that lacks it. There is no keel-sgemm-available marker and I am not inferring
# one -- the same check that caught an invented marker name in gate-p2.
AVX512_SEEN=0
# N_FORCED counts hosts that attempted the KEEL_FORCE=scalar run; N_FORCED_OK how
# many passed it. A host that was unreachable `continue`s long before that run, so
# zero attempts and zero passes are different facts about different fleets.
N_FORCED=0
N_FORCED_OK=0
if [[ -z "$HOSTS" ]]; then
  unmeasured "P3 needs an amd64 host to execute the AVX-512 Sgemm and none is configured, so the vector Sgemm is unmeasured on real silicon"
else
  remote_build_test_or_fail . "$BIN" "$LOG" \
    "cross-compiled linux/amd64 test binary (root package: Sgemm vs oracle)" \
    "cross-compile of linux/amd64 test binary"
  while read -r host; do
    probe_or_unmeasured "$host" || continue
    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] Sgemm sweep passes"
    elif remote_vanished; then
      unmeasured "[$host] the Sgemm sweep did not finish (#62), so this host says nothing about the sweep either way"
    else
      fail "[$host] Sgemm sweep passes"
      sed 's/^/        /' "$LOG" | tail -40
    fi
    backends="$(marker sgemm-backends-exercised "$LOG")"
    if [[ -z "$backends" ]]; then
      unmeasured "[$host] no keel-sgemm-backends-exercised marker, so what this host exercised cannot be read: coverage unknown is coverage unestablished"
    else
      info "[$host] backends exercised: $backends"
    fi
    # Counted without the OK condition: a host that exercised the avx512 kernels
    # and then failed the sweep is a host that HAD avx512, which is what makes the
    # aggregate below a FAIL rather than an UNMEASURED.
    if [[ " $backends " == *" avx512 "* ]]; then
      AVX512_SEEN=$((AVX512_SEEN + 1))
    fi
    if [[ "$OK" -eq 0 && " $backends " == *" avx512 "* ]]; then
      AVX512_GREEN="$host"
      cp "$LOG" "$SWEEPLOG"
    fi
    # The fallback, proved by taking it on a machine that has the alternative.
    FOK=0
    N_FORCED=$((N_FORCED + 1))
    KEEL_REMOTE_ENV="KEEL_FORCE=scalar" remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || FOK=$?
    if [[ "$FOK" -eq 0 ]]; then
      pass "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      SCALAR_FORCED="$host"
      N_FORCED_OK=$((N_FORCED_OK + 1))
    elif remote_vanished; then
      unmeasured "[$host] KEEL_FORCE=scalar: the sweep did not finish (#62), so the overridden dispatch is unmeasured here rather than broken"
    else
      fail "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      sed 's/^/        /' "$LOG" | tail -20
    fi
  done < <(hosts_lines)
  # The two aggregates, three-way over coverage state (#73's tier C). Both used to
  # collapse "the fleet came back short" into "the fleet had nothing to ask": a set
  # of hosts with no AVX-512 read as a FAIL saying no target ran it green, which is
  # an assertion about keel made from the absence of a measurement.
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the sweep ran green with the avx512 Sgemm live (target: $AVX512_GREEN)"
  elif [[ "$AVX512_SEEN" -gt 0 ]]; then
    fail "no target ran the Sgemm sweep green with the avx512 backend, though $AVX512_SEEN host(s) exercised it"
  else
    unmeasured "no host exercised the avx512 Sgemm at all, so whether the sweep passes with it live is unmeasured rather than short: there was no host to ask"
  fi
  # The old form failed only when NO host proved the fallback. Failing when any
  # host that tried it failed adds no criterion -- the per-host FAIL above already
  # blocks the gate for exactly those hosts -- it makes the aggregate a summary of
  # what was enforced instead of a weaker restatement of it.
  if [[ "$N_FORCED" -eq 0 ]]; then
    unmeasured "no host attempted the KEEL_FORCE=scalar run, so the dispatch override is unmeasured rather than unproved: every configured host dropped out before it"
  elif [[ "$N_FORCED_OK" -eq "$N_FORCED" ]]; then
    pass "every host that attempted it proved the scalar fallback under KEEL_FORCE=scalar ($N_FORCED_OK/$N_FORCED)"
  else
    fail "$((N_FORCED - N_FORCED_OK)) of $N_FORCED hosts that attempted the KEEL_FORCE=scalar run did not pass it (the per-host lines above say which)"
  fi
fi

# --------------------------------------------------- what the sweep covered
echo
echo "-- sweep extent (criteria 1 and 2: coverage is enforced, not trusted) --"
if [[ ! -s "$SWEEPLOG" ]]; then
  unmeasured "no avx512 sweep log to audit, so the sweep's extent is unmeasured rather than short"
else
  cfg="$(marker sgemm-config "$SWEEPLOG")"
  if [[ -z "$cfg" ]]; then
    unmeasured "no keel-sgemm-config marker (blocking params, edge strategy, beta variants, packing backend), so what the sweep configured cannot be read"
  else
    info "config: $cfg"
    for k in mr nr kc mc nc edge beta-variants pack; do
      [[ -n "$(field "$k" "$cfg")" ]] || unmeasured "keel-sgemm-config is missing $k=, so that part of the configuration cannot be read"
    done
  fi

  sizes="$(marker sgemm-sizes-exercised "$SWEEPLOG")"
  if [[ -z "$sizes" ]]; then
    unmeasured "no keel-sgemm-sizes-exercised marker, so the sizes the sweep ran cannot be read"
  else
    MISSING=""
    for n in $SWEEP_SIZES; do
      [[ " $sizes " == *" $n "* ]] || MISSING="$MISSING $n"
    done
    if [[ -z "$MISSING" ]]; then
      pass "every size in DESIGN.md §4/P3's sweep ran (1..17, 63, 64, 65, 500, 1000, 2048)"
    else
      fail "sizes missing from the sweep:$MISSING"
    fi
  fi

  combos="$(marker sgemm-combos-exercised "$SWEEPLOG")"
  if [[ -z "$combos" ]]; then
    unmeasured "no keel-sgemm-combos-exercised marker, so the transpose combinations the sweep ran cannot be read"
  else
    info "combinations: $combos"
    trans="$(field trans "$combos")"
    alphas="$(field alpha "$combos")"
    betas="$(field beta "$combos")"
    ncombo="$(field combos "$combos")"
    tmiss=""
    for t in $SWEEP_TRANS; do
      [[ ",$trans," == *",$t,"* ]] || tmiss="$tmiss $t"
    done
    if [[ -z "$tmiss" ]]; then
      pass "transpose lattice complete (NN NT TN TT)"
    else
      fail "transpose combinations missing:$tmiss"
    fi
    # 0 and 1 are the special-cased paths; a lattice of only those exercises
    # every shortcut and never the general multiply.
    for pair in "alpha:$alphas" "beta:$betas"; do
      nm="${pair%%:*}"; vals="${pair#*:}"
      if awk -v v="$vals" 'BEGIN {
            n = split(v, a, ",")
            for (i = 1; i <= n; i++) {
              if (a[i] + 0 == 0) zero = 1
              else if (a[i] + 0 == 1) one = 1
              else other = 1
            }
            exit !(zero && one && other)
          }'; then
        pass "$nm covers 0, 1 and a general value ($vals)"
      else
        fail "$nm = ${vals:-<none>} does not cover all of {0, 1, general}: only the special-cased paths would be tested"
      fi
    done
    if [[ -n "$ncombo" ]]; then
      expect="$(awk -v t="$trans" -v a="$alphas" -v b="$betas" \
        'BEGIN { printf "%d", split(t, x, ",") * split(a, y, ",") * split(b, z, ",") }')"
      if [[ "$ncombo" -eq "$expect" ]]; then
        pass "combination count matches the enumerated sets ($ncombo = $expect)"
      else
        fail "combination count $ncombo does not match the enumerated sets ($expect): the marker claims combinations it did not run"
      fi
    else
      unmeasured "keel-sgemm-combos-exercised has no combos= count, so its enumeration cannot be checked against what it ran"
    fi
  fi

  # Per-size oracle verification mode (criterion 2).
  VMISS=""; VBAD=""
  for n in $SWEEP_SIZES; do
    line="$(marker_row sgemm-verify "$SWEEPLOG" size "$n")"
    if [[ -z "$line" ]]; then
      VMISS="$VMISS $n"
      continue
    fi
    mode="$(field mode "$line")"
    if [[ "$n" -le "$SWEEP_EXACT_MAX" ]]; then
      [[ "$mode" == "exact" ]] || VBAD="$VBAD ${n}:${mode:-none}(must be exact)"
      continue
    fi
    case "$mode" in
      exact) ;;
      sampled)
        s="$(field n "$line")"
        if [[ -z "$s" ]] || [[ "$s" -lt "$SWEEP_SAMPLE_MIN" ]]; then
          VBAD="$VBAD ${n}:sampled(${s:-0} < $SWEEP_SAMPLE_MIN)"
        fi
        [[ -n "$(field seed "$line")" ]] || VBAD="$VBAD ${n}:sampled(no seed= to replay with)" ;;
      *) VBAD="$VBAD ${n}:${mode:-none}" ;;
    esac
  done
  if [[ -z "$VMISS" && -z "$VBAD" ]]; then
    pass "every size declares its oracle verification mode: exact up to $SWEEP_EXACT_MAX, >= $SWEEP_SAMPLE_MIN seeded exact entries above it"
    marker_all sgemm-verify "$SWEEPLOG" | tail -6 | sed 's/^/        /'
  else
    [[ -n "$VMISS" ]] && unmeasured "no keel-sgemm-verify line for size(s):$VMISS — how those sizes were verified cannot be read"
    [[ -n "$VBAD" ]] && fail "oracle verification too weak for size(s):$VBAD"
  fi

  # Not from DESIGN.md's list, and not optional either: a row stride wider than n
  # is how a caller passes a submatrix, a zero dimension is the empty product, and
  # the argument checks are what stands between a bad ld and an out-of-bounds
  # write. Adding checks a gate's phase implies is allowed; removing them is not.
  extras="$(marker sgemm-extra-exercised "$SWEEPLOG")"
  EMISS=""
  for e in ldpad zerodim argpanic; do
    [[ " $extras " == *" $e "* ]] || EMISS="$EMISS $e"
  done
  if [[ -z "$EMISS" ]]; then
    pass "edge coverage beyond the sweep: $extras"
  else
    fail "keel-sgemm-extra-exercised is missing:$EMISS"
  fi

  packm="$(marker pack-combos-exercised "$SWEEPLOG")"
  if [[ -n "$packm" ]]; then
    pass "packing differential-tested against its scalar reference ($packm)"
  else
    unmeasured "no keel-pack-combos-exercised marker, so whether packing is differential-tested cannot be read"
  fi
fi

carry_p2_properties P3 "packing and blocking"
reconcile_sweep_best_ipf "$SWEEP_BEST_IPF" "$LOG"

# ------------------------------------------ the throughput sentinel (P2, #19)
echo
echo "-- throughput sentinel (criteria 5 and 5b): P2's floor re-run on the dispatched shape --"
if RTLOG="$(bash scripts/roofline-test.sh 2>&1)"; then
  pass "roofline verdict controls ($(grep -c '^  ok ' <<<"$RTLOG") fixtures)"
else
  fail "roofline verdict controls"
  # shellcheck disable=SC2001  # prefixing every line; not a scalar substitution
  sed 's/^/        /' <<<"$RTLOG"
fi
IPF_2x32="$(audit_ipf_tile 2x32 "$AUDITKERN")"
IPF_4x32="$(audit_ipf_tile 4x32 "$AUDITKERN")"
IPF_PEAK="$(audit_ipf "$GATE_PEAK_FUNC" "$AUDITPEAK")"
if [[ -n "$IPF_2x32" && -n "$IPF_4x32" && -n "$IPF_PEAK" ]]; then
  info "audited insns/FMA: 2x32 $(printf '%.3f' "$IPF_2x32"), 4x32 $(printf '%.3f' "$IPF_4x32"), $GATE_PEAK_FUNC $(printf '%.3f' "$IPF_PEAK")"
else
  unmeasured "could not read insns/FMA from the audits, so the sentinel cannot classify its host"
fi
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

SENTINELS="$(sentinel_hosts)"
sentinel_declaration
# Criterion 5 judges the sentinel hosts. Criterion 6b needs something else from the
# same measurement: a *classification* for every host whose Sgemm will be divided by
# an OpenBLAS number, because the amended denominator applies only where the host is
# issue-bound. So the loop runs on every host and only the pass/fail is restricted to
# $SENTINELS; each host's (class, pmax) is recorded under $BINDIR for the OpenBLAS
# section to read, rather than re-derived there from a second set of measurements.
#
# A host with no recorded verdict is treated as FMA-bound below, which is the strict
# direction: no roofline, no leniency, the unmodified 60%-of-OpenBLAS bar.
CLASSIFY="$(printf '%s\n%s\n' "$SENTINELS" "$HOSTS" | sed '/^[[:space:]]*$/d' | awk '!seen[$0]++')"
# Criterion 5b's registry drift check runs once, on the first host that reports the
# marker: the binary is the same everywhere, and three identical lines would say no
# more than one. Empty after the loop means it never ran, which fails.
DRIFT_CHECKED=""
if [[ -z "$SENTINELS" ]]; then
  unmeasured "no sentinel host and no hosts at all, so P2's floor is unmeasured on this run rather than missed"
else
  info "judged as sentinels: $(tr '\n' ' ' <<<"$SENTINELS")"
  if ! remote_build_test ./bench "$KERNBIN" >"$LOG" 2>&1; then
    fail "cross-compile of the linux/amd64 bench binary (kernel benchmarks)"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    # Judged only if the host is a sentinel; classified either way. Since #146 the
    # sentinel set contains the whole fleet, so every host here is judged and the
    # "not a sentinel" renderings below are unreachable by construction — kept, not
    # deleted, because they are the only thing that would print if a narrowing
    # decision ever reintroduced a subset, and their last two firings are the two
    # runs #146 is about.
    JUDGED=0
    [[ $'\n'"$SENTINELS"$'\n' == *$'\n'"$host"$'\n'* ]] && JUDGED=1
    if ! remote_exec "$host" "$KERNBIN" "${BFLAGS[@]}" -test.bench="$KERN_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      if [[ "$JUDGED" -eq 1 ]]; then
        unmeasured "[$host] sentinel: the kernel benchmark run failed, so this sentinel's classification is unmeasured"
        sed 's/^/        /' "$BENCHLOG" | tail -20
      else
        info "[$host] kernel benchmark run failed; unclassified, so criterion 6b treats it as FMA-bound (the strict reading)"
      fi
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
    # Declared before read: the peak denominator and both shipped shapes. This is NOT
    # degraded for an unjudged host the way a failed *run* is above, because the run
    # succeeded — a missing row after a successful run is this gate's filter or parser
    # misreading its own output, and that is a defect on every host equally rather
    # than a property of one.
    #
    # This is the call site that matters most. Without it, a missing Peak/avx512
    # leaves every shape with an empty bench_ratio_lo and the criterion reports "no
    # bounded percent-of-peak for any shipped shape" — a red that blames the shapes
    # for the absence of their denominator, inside the criterion that carries P2's
    # floor forward. That message is entirely believable, which is exactly what would
    # have made the misattribution durable (#32's species; DESIGN.md §5 rule 6).
    # shellcheck disable=SC2086  # GATE_KERNELS is a deliberate word list
    require_bench "[$host] the kernel sentinel's inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "$GATE_PEAK" $GATE_KERNELS || continue

    # ---- criterion 5b, part 1
    assert_kern_audit_drift "$BENCHLOG" "$AUDITKERN" "$host" ""

    # The shape this host's library actually dispatches to, and the class it chose
    # under, from the very run being judged.
    hkern="$(marker bench-kern "$BENCHLOG")"; hkern="${hkern%% *}"
    htile="${hkern%%/*}"
    hclassline="$(marker bench-kern-class "$BENCHLOG")"
    hclass="${hclassline%% *}"

    ACT_LO=""; ACT_PT=""; ACT_HI=""; ACT_ID=""; ACT_IPF=""
    BEST_LO=""; BEST_PT=""; BEST_ID=""; MIXES=""; RIVALS=""
    for kname in $GATE_KERNELS; do
      [[ -n "$(bench_stat "$kname" "$BENCHCSV" GFLOP/s)" ]] || continue
      klo="$(bench_ratio_lo "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      kpt="$(bench_ratio "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      # Both ends, because the spread these feed is a class-selecting comparison
      # and #86 grades those against the interval; see scripts/roofline.sh. A
      # fraction bounded below but not above cannot take part in a ceiling.
      khi="$(bench_ratio_hi "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$klo" || -z "$khi" ]]; then
        info "[$host] ${kname##Kernel/}: no CI, not counted"
        continue
      fi
      kid="${kname##Kernel/}"
      kipf="$(audit_ipf_tile "${kid%%/*}" "$AUDITKERN")"
      [[ -n "$kipf" ]] && MIXES="$MIXES $kid|$kpt:$kipf:$klo:$khi"
      if [[ "${kid%%/*}" == "$htile" ]]; then
        ACT_LO="$klo"; ACT_PT="$kpt"; ACT_HI="$khi"; ACT_ID="$kid"; ACT_IPF="$kipf"
      else
        RIVALS="$RIVALS $kid|$klo|$kpt"
      fi
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_ID="$kid"
      fi
    done
    [[ -n "$IPF_PEAK" ]] && MIXES="$MIXES $GATE_PEAK_FUNC|1.0:$IPF_PEAK"
    if [[ -z "$BEST_LO" ]]; then
      if [[ "$JUDGED" -eq 1 ]]; then
        unmeasured "[$host] sentinel: no bounded percent-of-peak for any shipped shape, so this sentinel decides nothing either way"
      else
        info "[$host] no bounded percent-of-peak; unclassified, so criterion 6b treats it as FMA-bound"
      fi
      continue
    fi
    # ---- criterion 5b, part 2: the sentinel judges the shape that ships.
    # Not the fastest one on the shelf: with a per-host choice those are different
    # claims, and the one P3's numbers rest on is the dispatched shape.
    if [[ -z "$ACT_LO" ]]; then
      fail "[$host] Sgemm dispatches to ${hkern:-an unreported shape}, which this gate did not benchmark ($GATE_KERNELS): the shipped kernel would go unjudged"
      continue
    fi
    if [[ "$ACT_ID" != "$BEST_ID" ]]; then
      info "[$host] dispatched $ACT_ID; fastest measured here is $BEST_ID ($(awk -v r="$BEST_PT" 'BEGIN{printf "%.1f", r*100}')% of peak vs $(awk -v r="$ACT_PT" 'BEGIN{printf "%.1f", r*100}')%) — checked below"
    fi
    # The ceiling set is every mix except the shape under test; see gate-p2.sh
    # criterion 5b and scripts/roofline.sh (INDEPENDENCE).
    CEIL=""
    for mx in $MIXES; do
      [[ "${mx%%|*}" == "$ACT_ID" ]] && continue
      CEIL="$CEIL $(instrument_widen "${mx#*|}")"
    done
    # shellcheck disable=SC2086  # CEIL is a deliberate list of f:I:f_lo:f_hi words
    read -r CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY CSLO CSHI ATTAINHI <<<"$(
      throughput_verdict "$ACT_LO" "$ACT_HI" "${ACT_IPF:-0}" \
        "$PEAK_FLOOR" "$ROOF_FLOOR" "$ISSUE_CONVERGE_MAX" \
        "$ISSUE_MIX_SPREAD_MIN" "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" $CEIL)"
    frac="$(awk -v r="$ACT_LO" 'BEGIN{printf "%.1f", r * 100}')"
    fracpt="$(awk -v r="$ACT_PT" 'BEGIN{printf "%.1f", r * 100}')"
    # Both terms of the percentage below, on the line above the verdict that uses it.
    # Printing the ratio alone made #141 reconstruct the numerator and denominator
    # from tracked samples after the fact, and reconstruction is what happens when
    # disclosure fails: neither term appeared in any log this gate wrote (e5aa85c).
    info "[$host] the ratio's two terms, same invocation: $ACT_ID $(bench_stat "Kernel/$ACT_ID" "$BENCHCSV" GFLOP/s | cut -d' ' -f1) GFLOP/s over $GATE_PEAK $(bench_stat "$GATE_PEAK" "$BENCHCSV" GFLOP/s | cut -d' ' -f1) GFLOP/s"
    # The spread and its interval are PRINTED here, not just consumed. This gate
    # used to discard both (`_CSPREAD _MSPREAD`), and on 2026-08-15 that cost a
    # day: janus reclassified fma-bound with why=diverge, four verdicts moved, and
    # the quantity that decided it was nowhere in the log — the reconstruction had
    # to be bounded backwards out of the threshold. A gate that will not say what
    # its verdict divided by is the defect #86 is about, one level down.
    info "[$host] ceiling spread $(printf '%.3f' "$CSPREAD")x, interval [$(printf '%.3f' "$CSLO")x, $(printf '%.3f' "$CSHI")x] against the $ISSUE_CONVERGE_MAX bar, over a $(printf '%.3f' "$MSPREAD")x insns/FMA spread -> ${CLASS}-bound (why=$WHY)"
    # The second class-selecting comparison, printed on the same terms whenever it was
    # reached: attainment against the ceiling its own mixes set, as an interval, so a
    # why=nearceiling verdict is legible from the numbers rather than from its label.
    if awk -v r="$ROOF" 'BEGIN{exit !(r>0)}'; then
      info "[$host] attainment $(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a*100}')%..$(awk -v a="$ATTAINHI" 'BEGIN{printf "%.1f", a*100}')% of the $(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r*100}')% roofline (a whole interval above 100% falsifies the ceiling; one that crosses it cannot decide)"
    fi

    # ---- criterion 5b, part 3: the classification that chose the shape, checked
    # against the one this gate measured. The library fingerprints a feature bundle
    # because no microarchitecture is readable from pure Go (T14, #25); this is the
    # measurement that says whether the fingerprint was right on this machine.
    # Three states, because this comparison has two arms and only one of them is a
    # fingerprint (#86). The library's arm is CPUID-derived and fixed per host; this
    # gate's arm is measured, and a measured arm can fail to decide. Reporting a
    # disagreement in that case would accuse the fixed arm on the strength of the
    # noisy one — which is exactly what happened on 2026-08-15, down to the log line
    # dispatching the operator to repair a fingerprint that was right.
    if [[ -z "$hclass" ]]; then
      unmeasured "[$host] no keel-bench-kern-class marker, so the classification the shape was chosen by cannot be read"
    elif [[ "$CLASS" == indeterminate ]]; then
      unmeasured "[$host] classification indeterminate this run (why=$WHY), so this gate's arm of the comparison cannot decide and the library's ${hclass}-bound fingerprint is neither confirmed nor contradicted — ${hclassline#* }"
    elif [[ "$hclass" == "$CLASS" ]]; then
      pass "[$host] the library's ${hclass}-bound classification matches this gate's measured verdict — ${hclassline#* }"
    else
      fail "[$host] the library classified this host ${hclass}-bound and chose $ACT_ID; this gate measures it ${CLASS}-bound — ${hclassline#* }"
      info "  [$host] a shape chosen from the wrong class is issue #24 again: fix internal/kern.HostClass's fingerprint, not this check"
    fi

    # ---- criterion 5b, part 4: did the choice lose throughput, in this same run?
    # A rival counts as faster only if its lower bound clears the dispatched shape's
    # point estimate, so noise cannot fail the gate and a real 11-point gap cannot
    # hide in it.
    LOST=""
    for rv in $RIVALS; do
      rid="${rv%%|*}"; rest="${rv#*|}"; rlo2="${rest%%|*}"; rpt2="${rest#*|}"
      if awk -v a="$rlo2" -v b="$ACT_PT" 'BEGIN{exit !(a > b)}'; then
        LOST="$LOST $rid($(awk -v r="$rpt2" 'BEGIN{printf "%.1f", r*100}')% of peak, ${rid} lower bound $(awk -v r="$rlo2" 'BEGIN{printf "%.1f", r*100}')% > dispatched ${fracpt}%)"
      fi
    done
    if [[ -z "$RIVALS" ]]; then
      info "[$host] only one shape measured, so there is nothing to prefer over $ACT_ID"
    elif [[ -z "$LOST" ]]; then
      pass "[$host] no other shipped shape beats the dispatched $ACT_ID here (same invocation, net of CI)"
    else
      fail "[$host] dispatch selected $ACT_ID and left throughput on the table:$LOST"
    fi
    attpc="$(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a * 100}')"
    roofpc="$(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r * 100}')"
    # What this host's sentinel measurement tells the later sections, in one file so
    # neither of them re-derives it from a second set of measurements:
    #
    #   CLASS  the verdict, for criterion 6b's denominator — and "indeterminate" is
    #          one of its values, carried forward deliberately so criterion 6b reports
    #          the same one cause instead of receiving a class the run could not make
    #          and dividing by whichever denominator it implies (#86)
    #   pmax   = max_i f_i·I_i, recovered from the verdict's own two outputs so that
    #          criterion 6b's roofline is built from the same ceiling this one judged
    #          against
    #   ACT_PT the dispatched microkernel's percent of peak, and ACT_ID the shape it
    #          was measured on, for the retention line (#26) — that line divides the
    #          blocked Sgemm by this, so it must be the same shape Sgemm ships, which
    #          is exactly what criterion 5b part 2 established above
    printf '%s %s %s %s\n' "$CLASS" \
      "$(awk -v r="$ROOF" -v i="${ACT_IPF:-0}" 'BEGIN{printf "%.6f", r * i}')" \
      "$ACT_PT" "$ACT_ID" \
      >"$BINDIR/class-$host"
    if [[ "$JUDGED" -eq 0 ]]; then
      # The roofline is quoted only where it can be used. On an FMA-bound host the
      # classifier still computes one — and on antares it computes 39.3%, from a
      # ceiling its own best shape then exceeds — so printing it beside "fma-bound"
      # would read as leniency that this host does not get.
      if [[ "$CLASS" == issue ]]; then
        info "[$host] classified issue-bound (roofline ${roofpc}%) for criterion 6b; not a sentinel, so P2's floor is not judged here"
      elif [[ "$CLASS" == indeterminate ]]; then
        # Not judged here either way, so this is an info line and not a verdict — but
        # criterion 6b below WILL be unmeasured on this host, and it should be readable
        # from here why, rather than as a surprise 400 lines later.
        info "[$host] classification indeterminate this run (why=$WHY), so criterion 6b has no denominator for this host and will report it unmeasured; not a sentinel, so P2's floor is not judged here"
      else
        info "[$host] classified ${CLASS}-bound for criterion 6b, so no roofline applies and it faces the unmodified bar; not a sentinel, so P2's floor is not judged here"
      fi
      continue
    fi
    case "$CLASS/$RESULT" in
      issue/pass)
        pass "[$host] sentinel: dispatched $ACT_ID holds P2's floor — ${fracpt}% of peak (${frac}% net of CI) = ${attpc}% of its ${roofpc}% issue roofline (>= 90%)" ;;
      */pass)
        pass "[$host] sentinel: dispatched $ACT_ID holds P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (>= 55%)" ;;
      indeterminate/*)
        # One cause, one label. The floor this host would be held to depends on the
        # class, so with no class there is no floor to be under or over — and the
        # reading itself is fine, which the line says so nobody re-measures the wrong
        # thing. Both floors are named because the gap between them IS the stake: on
        # janus it is 43.8% of peak against 55%.
        unmeasured "[$host] sentinel: classification indeterminate this run (why=$WHY), so P2's floor is unmeasured on this host rather than held or missed — dispatched $ACT_ID read ${fracpt}% of measured peak, ${frac}% net of CI, and that reading is not in question"
        info "  [$host] which floor applies is what could not be decided: 55% of peak if fma-bound, ${ROOF_FLOOR}x this run's roofline if issue-bound"
        info "  [$host] the remedy is precision, never a wider bar: re-measure (not §4's re-run being spent — there is no verdict to overturn), and raise KEEL_BENCH_COUNT for a host that is chronically indeterminate here" ;;
      */refuse)
        fail "[$host] sentinel: dispatched $ACT_ID at $(printf '%.3f' "${ACT_IPF:-0}") insns/FMA is outside the shape guard — P3 fattened the K-loop (why=$WHY)" ;;
      *)
        fail "[$host] sentinel: dispatched $ACT_ID fell below P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (why=$WHY)" ;;
    esac
  done <<<"$CLASSIFY"
  # No host produced the marker at all: the ranking's inputs are then unverified,
  # which is a failure to check rather than a check that passed.
  [[ -n "$DRIFT_CHECKED" ]] || unmeasured "no host reported keel-bench-kern-audit, so the registry's recorded insns/FMA are unchecked against the object code — unmeasured, not drifted"
fi

# ----------------------------------------------- Sgemm at 2048^3 vs OpenBLAS
echo
echo "-- Sgemm at 2048^3: percent of measured peak, and >= 60% of single-thread OpenBLAS --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; the bar counts as cleared only net of both confidence intervals"
info "each host is also run once more with KEEL_KERN_CLASS pinned to the other class (criterion 5b), which is where the extra Sgemm time goes"

if [[ -n "$HOSTS" ]]; then
  remote_build_test_or_fail ./bench "$BENCHBIN" "$LOG" \
    "cross-compiled linux/amd64 bench binary (Sgemm + peak)" \
    "cross-compile of linux/amd64 bench binary"
  while read -r host; do
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" \
         -test.bench="$SGEMM_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the Sgemm benchmark run failed, so this host's rates are unmeasured"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
    # Both, declared together: the percent-of-peak line, the retention line and the
    # 5b comparison all read the peak, and an absent peak used to degrade silently
    # into a missing info line rather than into a verdict.
    require_bench "[$host] the blocked Sgemm's inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "$GATE_SGEMM" "$GATE_PEAK" || continue
    info "[$host] Sgemm 2048^3 $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s), peak $(bench_describe "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pk="$(bench_ratio "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    if [[ -n "$pk" ]]; then
      info "[$host] = $(awk -v r="$pk" 'BEGIN{printf "%.1f", r*100}')% of measured peak ($(awk -v r="${pklo:-0}" 'BEGIN{printf "%.1f", r*100}')% net of CI) — reported, not a P3 criterion"
    fi

    # ---- retention: how much of its own microkernel the blocked loop nest keeps.
    # Reported, never judged — the same standing as percent-of-peak, and for the same
    # reason: P3's criterion is the ratio against OpenBLAS, and blocking-parameter
    # work is P5's by DESIGN.md §4. It is printed because #26 is a named P5 input and
    # an input needs a number: janus keeps ~77% of its microkernel where the Zen
    # hosts keep 90-92%, and P5 inherits that gap as a measurement rather than as a
    # recollection of one. A run that stops printing it is a run that quietly dropped
    # P5's baseline, so the line is absent only when a measurement is missing.
    #
    # THIS IS A RATIO OF TWO POINT ESTIMATES FROM TWO INVOCATIONS, and it is not
    # bounded net of CI, because the two percentages come from different CSVs with a
    # peak measurement each — bench_ratio_lo cannot reach across them, and inventing
    # a bound here would be the statistics-free denominator §7 rule 7 forbids. Both
    # inputs are printed beside it so the division is reconstructible and so nobody
    # is tempted to compare the quotient against anything.
    if [[ -r "$BINDIR/class-$host" ]]; then
      read -r _ _ kpct kshape <"$BINDIR/class-$host"
      if [[ -n "$pk" && -n "$kpct" && -n "$kshape" ]] &&
         awk -v k="$kpct" 'BEGIN{exit !(k > 0)}'; then
        info "[$host] retention: the blocked loop nest keeps $(awk -v s="$pk" -v k="$kpct" 'BEGIN{printf "%.0f", s / k * 100}')% of its own $kshape microkernel ($(awk -v r="$pk" 'BEGIN{printf "%.1f", r*100}')% of peak blocked vs $(awk -v r="$kpct" 'BEGIN{printf "%.1f", r*100}')% unblocked; point estimates from two invocations) — reported, never judged; P5 baseline for #26"
      else
        info "[$host] retention not computable: no bounded microkernel percent-of-peak recorded for this host, so #26's P5 baseline is missing this run"
      fi
    fi

    # ---- criterion 5b, part 5: the shape choice, seen through packing and blocking
    # The kernel comparison in the sentinel section is the stronger evidence — both
    # shapes in one invocation. This is the same question asked of the number P3's
    # criterion actually rests on, which no single invocation can answer: the shape
    # is fixed at init, so measuring the other one means running the binary again
    # with KEEL_KERN_CLASS pinned to the class dispatch did not use.
    #
    # Two invocations, so the comparison is one-sided and CI-based: the passed-over
    # shape has to beat the dispatched shape's point estimate from below its own
    # interval before this fails. That makes it insensitive to the frequency drift
    # between two adjacent runs, at the cost of not seeing a small regression — the
    # same-invocation check above is what sees those.
    hclass="$(marker bench-kern-class "$BENCHLOG")"; hclass="${hclass%% *}"
    hkern="$(marker bench-kern "$BENCHLOG")"; hkern="${hkern%% *}"
    alt="$(other_class "$hclass")"
    if [[ -z "$alt" ]]; then
      fail "[$host] the bench run reports kernel class '${hclass:-<none>}', which is not a class this gate knows, so the shape choice cannot be cross-checked at 2048^3"
    elif ! KEEL_REMOTE_ENV="GOMAXPROCS=1 KEEL_KERN_CLASS=$alt" remote_exec "$host" "$BENCHBIN" \
           "${BFLAGS[@]}" -test.bench="$SGEMM_SHAPE_FILTER" >"$ALTLOG" 2>&1; then
      unmeasured "[$host] the KEEL_KERN_CLASS=$alt Sgemm run failed, so the shape choice is uncorroborated at 2048^3 — unmeasured, not refuted (the no-interval branch below already says so)"
      sed 's/^/        /' "$ALTLOG" | tail -20
    else
      altkern="$(marker bench-kern "$ALTLOG")"; altkern="${altkern%% *}"
      bench_csv "$ALTLOG" >"$ALTCSV" 2>"$LOG" || true
      [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
      altlo="$(bench_gflops_lo "$GATE_SGEMM" "$ALTCSV")"
      altpt="$(bench_gflops "$GATE_SGEMM" "$ALTCSV")"
      disppt="$(bench_gflops "$GATE_SGEMM" "$BENCHCSV")"
      if [[ -z "$altkern" || "$altkern" == "$hkern" ]]; then
        info "[$host] KEEL_KERN_CLASS=$alt selects ${altkern:-the same shape} too, so both classes agree here and there is nothing to compare"
      elif [[ -z "$altlo" || -z "$disppt" ]]; then
        unmeasured "[$host] the KEEL_KERN_CLASS=$alt run established no bounded Sgemm rate, so the shape choice is unmeasured at 2048^3 rather than confirmed"
      elif awk -v a="$altlo" -v b="$disppt" 'BEGIN{exit !(a > b)}'; then
        fail "[$host] at 2048^3 the passed-over $altkern beats the dispatched $hkern: $(printf '%.1f' "$altpt") GFLOP/s, $(printf '%.1f' "$altlo") net of CI, against $(printf '%.1f' "$disppt")"
      else
        pass "[$host] the dispatched $hkern is no slower than $altkern at 2048^3 ($(printf '%.1f' "$disppt") vs $(printf '%.1f' "$altpt") GFLOP/s, $(printf '%.1f' "$altlo") net of CI; separate invocations)"
      fi
    fi
  done < <(hosts_lines)
fi

# The reference: same host, same invocation, built natively behind the cgo tag —
# on EVERY gate host (ruling on #23). There is no reference-host list any more; the
# only apples-to-apples ratio is same silicon, same thread count, same run, so a
# host that cannot produce its own reference cannot contribute to this criterion,
# and it says so per host instead of one host standing in for three.
OB_CLEARED=0
OB_MEASURED=0
# Hosts that produced a bounded ratio and it sat below the bar. Counted, not derived
# by subtraction from NHOSTS: a host can leave this loop without any verdict at all
# (no OpenBLAS reference, no confidence interval, no bounded amended ratio), and
# `NHOSTS - cleared - indet` silently relabels every one of those as a host that
# measured slow. That is the label defect ruling #37 turned up in P5's scaling
# aggregate, in the other direction — an aggregate must credit and blame exactly what
# it verified, so every exit from this loop increments exactly one tally and the
# leftover is named as a leftover.
OB_MISSED=0
# Hosts where the classification the run derived straddled a bar (#86) AND the two
# candidate denominators then disagreed about the verdict. Post-collapse (ruled
# 2026-08-16) an undecidable class alone no longer lands here: if both candidates
# agree, the verdict does not depend on the undecided thing and the host is graded.
# What remains is genuine split — the classification was worth something. Counted
# separately from a miss, because "did not clear 60%" and "there was no 60% of
# anything to clear" are different claims about keel's speed.
OB_INDET=0
# Hosts that produced a bounded ratio which this gate then declined to judge, because the
# host is not admitted to the evidentiary class (#104). Its own tally for the reason every
# other exit has one: a not-admitted host DID produce a ratio, so folding it into
# OB_NOCOVER would print "produced no ratio" about a number printed six lines above.
OB_NOTADM=0
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"
if [[ -z "$HOSTS" ]]; then
  unmeasured "no execution hosts, so the >= 60%-of-OpenBLAS criterion cannot be evaluated (percent-of-peak is NOT a substitute): unmeasured, not missed"
elif [[ -n "$(git status --porcelain)" ]]; then
  fail "the working tree is dirty, so \`git archive HEAD\` would measure something other than what is here; commit first"
else
  while read -r host; do
    # Re-read, and re-checked, because the preamble's assertion has to hold at the
    # moment of measurement and not merely at the start of the gate. A governor that
    # changed in between belongs to a machine somebody started using, and the reading
    # it produces is not one §5 rule 5 covers. This replaces the old
    # "at least one host cleared the bar under the performance governor" tally, which
    # was satisfied by any single host and therefore said nothing about this one.
    #
    # The governor's own provenance line comes from assert_governor; the fields below
    # are this gate's OpenBLAS preflight, which used to share that line.
    assert_governor "$host" measured
    pre="$(ob_preflight "$host" || true)"
    obdistro="$(field distro "$pre")"
    obgo="$(field go "$pre")"
    oblib="$(field lib "$pre")"
    info "[$host] distro=${obdistro:-unknown} go=${obgo:-none} libopenblas=${oblib:-none}"
    clock_gate "$host" || continue
    if [[ "$obgo" == none || -z "$obgo" || "$oblib" == none || -z "$oblib" ]]; then
      MISS=""
      [[ "$obgo"  == none || -z "$obgo"  ]] && MISS="a Go toolchain"
      [[ "$oblib" == none || -z "$oblib" ]] && MISS="${MISS:+$MISS and }libopenblas.so"
      unmeasured "[$host] no same-host OpenBLAS reference: this host is missing $MISS, so its ratio is unmeasured (percent-of-peak is NOT a substitute)"
      ob_provision_help "$host" "$obdistro"
      continue
    fi
    info "[$host] building the openblas-tagged harness natively from git archive HEAD ($(git rev-parse --short HEAD))"
    # KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, which would close
    # stdin and hand tar an empty archive. The remote-side paths below expand
    # here, on the client, which is what is wanted — they are this script's
    # variables, not the remote shell's.
    # shellcheck disable=SC2029
    if ! git archive --format=tar HEAD | ssh "${KEEL_SCP_OPTS[@]}" "$host" \
         "rm -rf '$OPENBLAS_REMOTE_DIR' && mkdir -p '$OPENBLAS_REMOTE_DIR' && tar -x -C '$OPENBLAS_REMOTE_DIR'" >"$LOG" 2>&1; then
      unmeasured "[$host] could not ship the source tree for a native build, so this host has no reference reading"
      sed 's/^/        /' "$LOG" | tail -20
      continue
    fi
    # shellcheck disable=SC2029  # client-side expansion of a client-side path
    if ! ssh "${KEEL_SSH_OPTS[@]}" "$host" \
         "cd '$OPENBLAS_REMOTE_DIR' && GOEXPERIMENT=simd CGO_ENABLED=1 go test -c -tags openblas -o bench-ob.test ./bench" >"$LOG" 2>&1; then
      fail "[$host] native build of the openblas-tagged bench harness failed"
      sed 's/^/        /' "$LOG" | tail -30
      continue
    fi
    # ---- the reference's ceiling, established before the run that counts (#31).
    # Every candidate is recorded, not just the winner: a provenance block that names
    # only the family used cannot be checked against the one that was rejected, and
    # the margin over DYNAMIC_ARCH's own choice is the part a reader needs in order
    # to know whether upstream's selector was wrong here.
    SWEEP="$(ob_coretype_sweep "$host" || true)"
    if [[ -z "$SWEEP" ]]; then
      unmeasured "[$host] the coretype sweep produced nothing, so the reference's ceiling is unmeasured and the denominator would be whatever DYNAMIC_ARCH happened to pick"
      continue
    fi
    info "[$host] coretype sweep, best of $KEEL_BENCH_COUNT at -benchtime=$KEEL_BENCH_TIME:"
    SWEEP_NOROW=0
    while read -r cq cach cgf; do
      [[ -n "$cq" ]] || continue
      if [[ "$cgf" == "-" ]]; then
        info "  [$host] OPENBLAS_CORETYPE=$cq: unavailable on this host"
      elif [[ "$cgf" == norow ]]; then
        SWEEP_NOROW=1
        bad_row="$cq"
      else
        info "  [$host] OPENBLAS_CORETYPE=$cq -> corename=$cach, $cgf GFLOP/s"
      fi
    done <<<"$SWEEP"
    # A harness that ran and produced no $GATE_OPENBLAS row is this gate misreading
    # its own output or misnaming its own benchmark, and #33 is what that looks like
    # when it is allowed to degrade into a number instead of a failure.
    if [[ "$SWEEP_NOROW" -eq 1 ]]; then
      fail "[$host] the sweep ran the harness for OPENBLAS_CORETYPE=$bad_row and it produced no $GATE_OPENBLAS result row: that is a defect in this gate's filter or parser, not a property of the host"
      continue
    fi
    # #33's signature, mechanized. A sweep exists to discriminate, so if every kernel
    # family it reached measured the same rate to the last printed digit, the
    # instrument is broken by construction and any "winner" is an artifact of the
    # order awk saw the candidates in. Variance too low is as diagnostic as variance
    # too high.
    #
    # The test is over DISTINCT ACHIEVED corenames, not over candidates, because
    # candidates that alias to one family are supposed to agree: vesta answers both
    # Cooperlake and SapphireRapids with corename=Cooperlake, and a check counting
    # candidates would fire on that and punish OpenBLAS for correctly reporting that
    # two names are one family. Counting families also makes this indifferent to the
    # candidate list's composition — three more aliases landing on Cooperlake cannot
    # perturb it, because the question is how many families were reached, not how many
    # requests were made. A host whose library ignores every request has reached one
    # family, ties with itself legitimately, and is caught by the pin verification
    # below instead: that half asks "did the request take", this half asks "can the
    # instrument tell the families apart".
    #
    # Exact equality is the threshold, and the measurements support it: two separate
    # invocations of one family on vesta read 150.60 and 149.80, so real rates of the
    # same silicon do not tie to two decimals. An exact tie across distinct families
    # is unambiguous.
    read -r SWEEP_FAMS SWEEP_SPREAD <<<"$(awk '
      $3 == "-" || $3 == "norow" { next }
      {
        fams[$2] = 1
        if (hi == "" || $3 + 0 > hi) hi = $3 + 0
        if (lo == "" || $3 + 0 < lo) lo = $3 + 0
      }
      END { n = 0; for (f in fams) n++; printf "%d %s", n, (hi == "" ? "-" : hi - lo) }' <<<"$SWEEP")"
    if [[ "${SWEEP_FAMS:-0}" -ge 2 ]] && awk -v s="$SWEEP_SPREAD" 'BEGIN{exit !(s + 0 == 0)}'; then
      fail "[$host] the coretype sweep is non-discriminating: $SWEEP_FAMS distinct kernel families measured an identical rate, so it cannot rank them and any winner would be an artifact of candidate order — the instrument is broken rather than the families equal (issue #33)"
      continue
    fi
    # ---- the pin is an intervention justified by a measured effect: no effect, no
    # intervention (ruling on issue #35).
    #
    # Ranking every candidate by rate and pinning the maximum crowns noise on a host
    # where no distinct family is actually faster. Across two full gate runs, janus's
    # and antares's winning *request* moved (SapphireRapids, then default; then
    # SapphireRapids, then Cooperlake) while the achieved *family* never did, with
    # margins of 0.0-1.0% — and the sweep measures a same-family drift of about 0.5%
    # itself, because two invocations of one kernel on one machine do not repeat
    # exactly. A winner by a margin inside that drift is a winner by dice: #33's
    # lesson relocated from the parser to the selection.
    #
    # So the question is not "which candidate was fastest" but "did a family the
    # library did not already choose beat the one it did, by more than this sweep's
    # own noise floor". If yes, pin it — that is what the #31 ruling exists for. If
    # no, run the reference the way the library runs itself, UNPINNED, and record
    # that as the finding. What reproduces on those hosts is the default family; the
    # noise-ranked alias does not, and pinning it would encode false precision into
    # the provenance line.
    #
    # The noise floor is measured, not assumed: the largest spread between candidates
    # that landed on the same achieved corename. Same silicon, same kernel, different
    # invocation — that is drift by construction. Hosts where no two candidates alias
    # yield 0, which makes the test "strictly faster", the strict direction.
    SWEEP_DRIFT="$(awk '
      $3 == "-" || $3 == "norow" { next }
      {
        if (!($2 in hi) || $3 + 0 > hi[$2]) hi[$2] = $3 + 0
        if (!($2 in lo) || $3 + 0 < lo[$2]) lo[$2] = $3 + 0
      }
      END { d = 0; for (f in hi) if (hi[f] - lo[f] > d) d = hi[f] - lo[f]; printf "%.2f", d }' <<<"$SWEEP")"
    read -r OBDEF_CORE OBDEF_RATE <<<"$(awk '$1 == "default" && $3 != "-" && $3 != "norow" { print $2, $3 }' <<<"$SWEEP")"
    # The best candidate from a family the default did NOT already select. Compared by
    # achieved corename, so an alias of the default's own family is not a contender.
    read -r XF_CT XF_CORE XF_RATE <<<"$(awk -v def="${OBDEF_CORE:-}" '
      $3 == "-" || $3 == "norow" || $2 == def { next }
      $3 + 0 > m { m = $3 + 0; best = $0 }
      END { print best }' <<<"$SWEEP")"
    if [[ -z "${OBDEF_RATE:-}" ]]; then
      # No baseline, so "beats the default" is not a question that can be asked. Fall
      # back to the fastest candidate and say which reading this is, rather than
      # presenting it as a win over a comparison that never happened.
      read -r OBCT OBCT_CORE OBCT_RATE <<<"$(awk '$3 != "-" && $3 != "norow" && $3 + 0 > m { m = $3 + 0; best = $0 } END { print best }' <<<"$SWEEP")"
      if [[ -z "${OBCT:-}" ]]; then
        unmeasured "[$host] no candidate coretype produced a rate, so the reference cannot be pinned to its best family and its ceiling is unmeasured"
        continue
      fi
      info "[$host] reference pinned to OPENBLAS_CORETYPE=$OBCT (corename=$OBCT_CORE, $OBCT_RATE GFLOP/s), the fastest candidate; the default selection produced no rate, so there is no baseline this can be called a win over"
    elif [[ -n "${XF_RATE:-}" ]] &&
         awk -v x="$XF_RATE" -v d="$OBDEF_RATE" -v n="$SWEEP_DRIFT" 'BEGIN{exit !(x - d > n)}'; then
      OBCT="$XF_CT"; OBCT_CORE="$XF_CORE"; OBCT_RATE="$XF_RATE"
      info "[$host] reference pinned to OPENBLAS_CORETYPE=$OBCT (corename=$OBCT_CORE, $OBCT_RATE GFLOP/s), $(awk -v w="$OBCT_RATE" -v d="$OBDEF_RATE" 'BEGIN{printf "%+.1f%%", (w / d - 1) * 100}') against DYNAMIC_ARCH's own choice (corename=$OBDEF_CORE, $OBDEF_RATE GFLOP/s): a cross-family win of $(awk -v w="$OBCT_RATE" -v d="$OBDEF_RATE" 'BEGIN{printf "%.2f", w - d}') GFLOP/s against this sweep's own measured same-family drift of $SWEEP_DRIFT"
    else
      OBCT=default; OBCT_CORE="$OBDEF_CORE"; OBCT_RATE="$OBDEF_RATE"
      if [[ -n "${XF_RATE:-}" ]]; then
        info "[$host] no cross-family winner beyond drift: the best family the library did not already choose is $XF_CORE (OPENBLAS_CORETYPE=$XF_CT) at $XF_RATE GFLOP/s against the default's $OBDEF_RATE, a difference of $(awk -v w="$XF_RATE" -v d="$OBDEF_RATE" 'BEGIN{printf "%+.2f", w - d}') GFLOP/s inside a measured same-family drift of $SWEEP_DRIFT"
      else
        info "[$host] no cross-family winner beyond drift: every candidate that produced a rate landed on the default's own $OBDEF_CORE"
      fi
      info "[$host] the reference therefore runs UNPINNED, the way the library runs itself (corename=$OBDEF_CORE, $OBDEF_RATE GFLOP/s) — the pin is an intervention justified by a measured effect, and there is none here (#35)"
    fi
    OBARGS=""
    for a in "${BFLAGS[@]}" "-test.bench=$SGEMM_BENCH_FILTER"; do OBARGS+=" $(printf '%q' "$a")"; done
    OBENV=(GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1)
    # Pinned only when the sweep chose something other than what the library picks
    # unaided, so the common case runs exactly the command it always did.
    [[ "$OBCT" != default ]] && OBENV+=("OPENBLAS_CORETYPE=$OBCT")
    # Opened here rather than at clock_gate above, and against the NATIVE harness: this
    # section's middle window is the Peak row inside the run below, which was built by the
    # host's own toolchain, so head and tail have to come from the same binary or the
    # trend test has a compiler in it. The bracket is deliberately tight — the coretype
    # sweep above is minutes of benchmarking, and a wider bracket would be answering a
    # question about a window the judged run does not occupy (§5 rule 5 as amended 2026-08-16).
    clock_head "$host" "@$OPENBLAS_REMOTE_DIR/bench-ob.test" || continue
    # shellcheck disable=SC2029  # client-side expansion of a client-side path
    if ! ssh "${KEEL_SSH_OPTS[@]}" "$host" \
         "cd '$OPENBLAS_REMOTE_DIR' && env ${OBENV[*]} ./bench-ob.test$OBARGS" >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the openblas-tagged benchmark run failed, so this host has no reference reading"
      sed 's/^/        /' "$BENCHLOG" | tail -30
      continue
    fi
    obm="$(marker bench-openblas "$BENCHLOG")"
    gmp="$(marker bench-gomaxprocs "$BENCHLOG")"
    info "[$host] openblas: ${obm:-<no marker>} | gomaxprocs: ${gmp:-<no marker>}"
    if [[ -z "$obm" || "$obm" == *"not available"* ]]; then
      fail "[$host] the harness reports no OpenBLAS reference despite the openblas build tag"
      continue
    fi
    if [[ "$(field threads "$obm")" != "1" ]]; then
      fail "[$host] OpenBLAS reports ${obm:+$(field threads "$obm")} thread(s); the criterion is single-thread on both sides, so this is not the comparison DESIGN.md asks for"
      continue
    fi
    if [[ "$gmp" != "1" ]]; then
      fail "[$host] GOMAXPROCS=${gmp:-unreported}; keel's side of the comparison must be single-threaded too"
      continue
    fi
    # The kernel family DYNAMIC_ARCH selected. Checked because it is the one part of
    # the reference's configuration whose failure mode is a *low* denominator, and a
    # low denominator flatters keel — see $OPENBLAS_OK_CORES above.
    obcore="$(field corename "$obm" | tr '[:upper:]' '[:lower:]')"
    if [[ -z "$obcore" ]]; then
      unmeasured "[$host] the OpenBLAS marker carries no corename=, so the reference's kernel family cannot be read and a generic kernel cannot be ruled out"
      continue
    fi
    CORE_OK=0
    for c in $OPENBLAS_OK_CORES; do [[ "$obcore" == "$c" ]] && CORE_OK=1; done
    if [[ "$CORE_OK" -eq 0 ]]; then
      fail "[$host] OpenBLAS selected the '$obcore' kernel, which is not on the AVX2-or-better allowlist ($OPENBLAS_OK_CORES); a reference slower than the host can be is a denominator that flatters keel"
      info "  [$host] if '$obcore' is a legitimate AVX2-or-better target, add it to OPENBLAS_OK_CORES in this script; if it is a generic build, the distro package needs replacing"
      continue
    fi
    # The pin has to be verified, not trusted: OPENBLAS_CORETYPE is honoured by the
    # library, not by us, and a pin that silently did not take would divide keel by a
    # slower reference than the one this gate chose and reported — the flattering
    # direction, arrived at through a line of output claiming otherwise.
    if [[ "$obcore" != "$(tr '[:upper:]' '[:lower:]' <<<"$OBCT_CORE")" ]]; then
      if [[ "$OBCT" == default ]]; then
        # The unpinned case is still checked, and it is not a vacuous check: the sweep
        # ran the default candidate too, so this compares two unpinned invocations and
        # catches a library whose unaided choice is not stable across runs.
        fail "[$host] the reference ran unpinned and selected corename=$obcore, but the sweep's own unpinned candidate selected corename=$OBCT_CORE: the library's unaided choice is not stable on this host, so the rate keel is about to be divided by is not the one the sweep measured"
      else
        fail "[$host] the coretype pin did not take: the sweep chose corename=$OBCT_CORE (OPENBLAS_CORETYPE=$OBCT) but the measured run reports corename=$obcore, so the number keel is about to be divided by is not the reference that was selected"
      fi
      continue
    fi
    if [[ "$OBCT" == default ]]; then
      info "[$host] reference kernel family: $obcore (on the AVX2-or-better allowlist, unpinned and matching the sweep's unpinned candidate)"
    else
      info "[$host] reference kernel family: $obcore (on the AVX2-or-better allowlist, and the sweep's winner as pinned)"
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
    # All three declared, not just the reference: criterion 6b reads the peak too, and
    # an absent peak used to reach p3_denominator as peak=0, which takes the nopeak
    # branch and silently reverts an issue-bound host to plain OpenBLAS. That is the
    # strict direction, so it could never flatter keel — but it would have replaced a
    # ruled denominator with a different one on the strength of a measurement that was
    # never taken, and printed "of a 0.00 GFLOP/s peak" while doing it.
    if ! require_bench "[$host] the keel/OpenBLAS ratio's inputs" \
         "$BENCHLOG" "$BENCHCSV" GFLOP/s "$GATE_SGEMM" "$GATE_OPENBLAS" "$GATE_PEAK"; then
      # Two very different states used to share one sentence here, and only one of them
      # is a missing reference (#32). The library is present and reported itself a few
      # lines above, so an absent row means the run did not produce it — a gate defect.
      # Print the names that did come back, because "the filter ran something else" is
      # what that looks like from here.
      info "  [$host] this host's OpenBLAS built, ran and reported itself above, so the absence is in what the gate asked to be run, not in the reference"
      info "  [$host] -test.bench=$SGEMM_BENCH_FILTER returned: $(awk -F, '/^Benchmark/ { n = $1; sub(/-[0-9]+$/, "", n); print n }' "$BENCHCSV" | sort -u | tr '\n' ' ')"
      continue
    fi
    # Closed before the mission ratio is formed: both of its halves came out of the run
    # this series brackets, so on a host with no governor this is where the clock they
    # were measured on is established or the ratio is unmeasured (§5 rule 5 as amended 2026-08-16).
    clock_post "$host" "@$OPENBLAS_REMOTE_DIR/bench-ob.test" "$BENCHCSV" || continue
    info "[$host] keel $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s) vs OpenBLAS $(bench_describe "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s), one invocation"
    rlo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    rpt="$(bench_ratio "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    if [[ -z "$rlo" ]]; then
      unmeasured "[$host] no bounded keel/OpenBLAS ratio: benchstat established no confidence interval, which is a failure to measure rather than a pass"
      continue
    fi
    rlopc="$(awk -v r="$rlo" 'BEGIN{printf "%.1f", r*100}')"
    rptpc="$(awk -v r="$rpt" 'BEGIN{printf "%.1f", r*100}')"
    OB_MEASURED=$((OB_MEASURED + 1))

    # ---- criterion 6b: which number this host's Sgemm is divided by (#23, #17/#18)
    # Every input is a measurement already taken, and all of them from THIS run on
    # THIS host: the reference rate, the peak rate, the shape Sgemm dispatched to,
    # and the classification recorded by the sentinel loop above.
    obclass="fma"; obpmax="0"
    if [[ -r "$BINDIR/class-$host" ]]; then read -r obclass obpmax _ _ <"$BINDIR/class-$host"; fi
    # "4x32/avx512 (available: ...)" -> Kernel4x32; audited, not assumed, because a
    # roofline built from the wrong shape's instruction count is the hole the shape
    # guard exists to close.
    obkern="$(marker bench-kern "$BENCHLOG")"; obkern="${obkern%% *}"
    obtile="${obkern%%/*}"
    i_active="$(audit_ipf_tile "$obtile" "$AUDITKERN")"
    ob_rate="$(bench_gflops "$GATE_OPENBLAS" "$BENCHCSV")"
    peak_rate="$(bench_gflops "$GATE_PEAK" "$BENCHCSV")"
    read -r obdenom obsrc obroof obwhy <<<"$(p3_denominator \
      "$obclass" "$obpmax" "${i_active:-0}" \
      "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" "${ob_rate:-0}" "${peak_rate:-0}")"

    # ---- the class this criterion divides by is itself derived (#86). When its
    # derivation straddled a bar there is no denominator, and the numerator is not the
    # thing in doubt: on 2026-08-15 keel's rate here was 76.81 GFLOP/s +/- 1.0%, the
    # healthiest link in the chain, and it was reported as a 39.5% FAIL because a peak
    # probe three steps upstream had moved the classification and with it the
    # denominator. A verdict cannot be more certain than the least certain link in its
    # derivation chain.
    #
    # Both candidate ratios are printed, which is the only place in this gate that a
    # ratio is shown against a denominator the run did not choose — and it is the one
    # place that must, because the quantity under discussion is exactly how much the
    # classification was worth. The candidates come from p3_denominator itself, called
    # once per hypothesis, so nothing here re-derives a denominator by hand.
    if [[ "$obsrc" == indeterminate ]]; then
      keelr="$(bench_gflops "$GATE_SGEMM" "$BENCHCSV")"
      hpklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      # ---- THE COLLAPSE, extended here from roofline.sh's class decision (ruled
      # 2026-08-16; same law, second site). UNMEASURED is for verdicts that VARY over
      # the uncertainty. When the class cannot be decided but BOTH candidate
      # denominators put this host on the same side of the bar, the verdict does not
      # depend on the undecided thing, and withholding it would make UNMEASURED fire on
      # immaterial doubt — spending the scarcity the three-state grading exists to
      # protect. Symmetric on purpose: two agreeing misses collapse to FAIL, so a
      # genuinely slow host cannot shelter behind a class the run could not derive.
      #
      # Judged on hlo, the net-of-CI bound, NOT on the point estimate printed beside
      # it. That is roofline.sh's hard-won form ("a collapse justified by the midpoint
      # would be exactly the noise-driven verdict this amendment exists to prevent"):
      # each hypothesis is graded on the same conservative quantity the decided path
      # grades, so a collapse says "clears under every reading of every candidate", not
      # "clears on average".
      #
      # An unbounded candidate is not agreement. p3_ratio_lo declines to bound a ratio
      # its inputs cannot support, and treating a missing bound as either verdict would
      # be the substitution it exists to refuse, so any unbounded candidate sends this
      # host to UNMEASURED with both branches still printed.
      NCAND=0; NCLEAR=0; NMISS=0
      for hypo in issue fma; do
        read -r hdenom hsrc hroof hwhy <<<"$(p3_denominator \
          "$hypo" "$obpmax" "${i_active:-0}" \
          "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" "${ob_rate:-0}" "${peak_rate:-0}")"
        hlo="$(p3_ratio_lo "$hsrc" "$rlo" "$hpklo" "$hroof")"
        hptpc="$(awk -v k="${keelr:-0}" -v d="$hdenom" 'BEGIN{printf "%.1f", (d > 0) ? k / d * 100 : 0}')"
        NCAND=$((NCAND + 1))
        if [[ -z "$hlo" ]]; then
          info "  [$host] had the class been ${hypo}-bound: $hsrc $(printf '%.2f' "$hdenom") GFLOP/s (why=$hwhy) -> ${hptpc}% point estimate, but NO bounded ratio against it, so this candidate has no verdict to agree or disagree with"
          continue
        fi
        hlopc="$(awk -v r="$hlo" 'BEGIN{printf "%.1f", r*100}')"
        if awk -v r="$hlo" -v f="$OPENBLAS_FLOOR" 'BEGIN{exit !(r >= f)}'; then
          NCLEAR=$((NCLEAR + 1)); hv="CLEARS"
        else
          NMISS=$((NMISS + 1)); hv="MISSES"
        fi
        info "  [$host] had the class been ${hypo}-bound: $hsrc $(printf '%.2f' "$hdenom") GFLOP/s (why=$hwhy) -> ${hptpc}%, ${hlopc}% net of CI, which $hv the 60% bar"
      done
      # Admission after both candidate readings are printed and before either becomes a
      # verdict (#104). This branch is the one where the two orders visibly differ: the
      # host is already indeterminate, and reporting that instead would say "re-measure"
      # about a host no re-measurement makes judgeable.
      if ! adm_judgeable "$host" "$GOV_PROV" \
           "Sgemm at 2048^3 reads ${rptpc}% of plain OpenBLAS, ${rlopc}% net of CI, under an indeterminate classification whose candidates split $NCLEAR clear / $NMISS miss"; then
        OB_NOTADM=$((OB_NOTADM + 1))
        continue
      fi
      # The decision itself is p3_collapse, in scripts/roofline.sh with fixtures,
      # for the reason p3_ratio_lo gives about its own arithmetic: an if-chain read
      # inline in a gate script is not evidence that it is right, and this one can
      # turn an UNMEASURED into a PASS.
      COLLAPSE="$(p3_collapse "$NCAND" "$NCLEAR" "$NMISS")"
      if [[ "$COLLAPSE" == pass ]]; then
        pass "[$host] Sgemm at 2048^3 clears the 60% bar under BOTH candidate denominators, so the undecidable classification does not change the verdict (why=agree-anyway; class=indeterminate, both branches printed above, each judged net of CI)"
        OB_CLEARED=$((OB_CLEARED + 1))
      elif [[ "$COLLAPSE" == fail ]]; then
        OB_MISSED=$((OB_MISSED + 1))
        fail "[$host] Sgemm at 2048^3 misses the 60% bar under BOTH candidate denominators, so the undecidable classification does not change the verdict (why=agree-anyway; class=indeterminate, both branches printed above, each judged net of CI) — the classification is not what is wrong here"
      else
        OB_INDET=$((OB_INDET + 1))
        unmeasured "[$host] classification indeterminate this run and the two candidate denominators DISAGREE (why=split: $NCLEAR clear, $NMISS miss, $((NCAND - NCLEAR - NMISS)) unbounded), so criterion 6 has no denominator on this host — keel measured $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s) here and that reading is not in question; the gate declines to pick a denominator by noise"
        info "  [$host] that gap is what the classification was worth, and it is why this is unmeasured rather than graded: the run could not say which of the two applies"
      fi
      continue
    fi
    # A roofline that does not apply is printed as n/a, not as 0.0% (issue #34).
    # p3_denominator returns 0 as its sentinel and that stays its contract; it is
    # only the rendering that must not dress a not-applicable as a measured zero.
    #
    # The condition is `roof == 0`, NOT `obsrc == openblas` as the issue first
    # proposed: an issue-bound host whose min() picked the reference (why=reference)
    # has a real roofline that was computed and compared, and printing n/a there
    # would hide a number instead of a hole — the same misreading in the other
    # direction. The sentinel is the thing to test, so test the sentinel.
    if awk -v r="$obroof" 'BEGIN{exit !(r>0)}'; then
      roofstr="roofline $(awk -v r="$obroof" 'BEGIN{printf "%.1f", r*100}')% of a $(printf '%.2f' "${peak_rate:-0}") GFLOP/s peak"
    else
      roofstr="roofline n/a (why=$obwhy) against a $(printf '%.2f' "${peak_rate:-0}") GFLOP/s peak"
    fi
    info "[$host] denominator: $obsrc $(printf '%.2f' "$obdenom") GFLOP/s (why=$obwhy, class=$obclass, Sgemm ran ${obkern:-unknown} at $(printf '%.3f' "${i_active:-0}") insns/FMA, $roofstr)"

    # Net of CI, in the same conservative direction as everything else here. Against
    # the roofline cap that is keel_lo/(roof · peak_hi) = pklo/roof; $rlo remains a
    # valid (weaker) bound on the same ratio because the cap is below OpenBLAS, so
    # the tighter of the two is taken rather than whichever came to hand.
    alo="$rlo"; apt="$rpt"
    if [[ "$obsrc" == roofline ]]; then
      pklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      alo="$(p3_ratio_lo roofline "$rlo" "$pklo" "$obroof")"
      if [[ -z "$alo" ]]; then
        unmeasured "[$host] no bounded Sgemm/peak ratio, so the amended denominator cannot be bounded either; that is a failure to measure, not a pass"
        continue
      fi
      apt="$(awk -v k="$(bench_gflops "$GATE_SGEMM" "$BENCHCSV")" -v d="$obdenom" 'BEGIN{printf "%.4f", k / d}')"
    fi
    alopc="$(awk -v r="$alo" 'BEGIN{printf "%.1f", r*100}')"
    aptpc="$(awk -v r="$apt" 'BEGIN{printf "%.1f", r*100}')"
    # Both ratios, always, whichever one binds (§7 rule 7 applies to this script's
    # arithmetic as much as to the library's benchmarks).
    if [[ "$obsrc" == roofline ]]; then
      info "[$host] amended (issue-capped): ${aptpc}%, ${alopc}% net of CI | plain OpenBLAS: ${rptpc}%, ${rlopc}% net of CI"
    else
      info "[$host] plain OpenBLAS is the denominator (why=$obwhy), so the amended and unmodified ratios are the same number"
    fi
    # Admission after the reading above and before the bar (#104). The mission ratio is
    # the number this whole gate exists for, so it is stated on every host; what the class
    # decides is whether 60% is a bar this host's silicon can be held to.
    if ! adm_judgeable "$host" "$GOV_PROV" \
         "Sgemm at 2048^3 reads ${aptpc}% of its $obsrc denominator, ${alopc}% net of CI (plain OpenBLAS ${rptpc}%, ${rlopc}%)"; then
      OB_NOTADM=$((OB_NOTADM + 1))
      continue
    fi
    if awk -v r="$alo" -v f="$OPENBLAS_FLOOR" 'BEGIN{exit !(r >= f)}'; then
      pass "[$host] Sgemm at 2048^3 is ${aptpc}% of its $obsrc denominator, ${alopc}% net of CI (>= 60%; plain OpenBLAS ${rptpc}%, ${rlopc}% net of CI)"
      OB_CLEARED=$((OB_CLEARED + 1))
    else
      OB_MISSED=$((OB_MISSED + 1))
      fail "[$host] Sgemm at 2048^3 is only ${aptpc}% of its $obsrc denominator, ${alopc}% net of CI (< 60%; plain OpenBLAS ${rptpc}%, ${rlopc}% net of CI)"
    fi
  done < <(hosts_lines)
  # Three-way over coverage state, the same shape as criterion 5b's aggregate and for
  # the same reason: a host that could not be judged must not be re-reported here as a
  # host that missed the bar. That would be one cause producing two verdicts, the
  # second of them a false claim about keel's speed.
  # Hosts that left the loop with no verdict for a reason that is not the split:
  # no OpenBLAS reference, no benchstat interval, no bounded amended ratio. Named,
  # because they are neither cleared nor slow and the sentence must not imply either.
  OB_NOCOVER=$((NHOSTS - OB_CLEARED - OB_MISSED - OB_INDET - OB_NOTADM))
  case "$(fleet_coverage "$NHOSTS" "$OB_MEASURED" "$OB_CLEARED" "$OB_MISSED" "$OB_INDET")" in
  unmeasured)
    unmeasured "no host produced a keel/OpenBLAS ratio at all, so criterion 6 is unmeasured rather than missed" ;;
  pass)
    pass "every gate host cleared 60% of its own single-thread OpenBLAS ($OB_CLEARED/$NHOSTS)" ;;
  fail)
    fail "$OB_CLEARED of $NHOSTS gate hosts cleared the bar and $OB_MISSED measured below it ($OB_INDET undecidable-and-split, $OB_NOTADM not admitted to the evidentiary class, $OB_NOCOVER produced no ratio); ruling #23 asks every host to clear its own reference" ;;
  *)
    unmeasured "$((OB_INDET + OB_NOTADM + OB_NOCOVER)) of $NHOSTS gate hosts could not be judged this run ($OB_INDET had an indeterminate classification whose two candidate denominators disagreed, $OB_NOTADM are not admitted to the evidentiary class so no ratio from them is judgeable, $OB_NOCOVER produced no bounded ratio at all); the other $OB_CLEARED cleared the bar, and no host measured below it" ;;
  esac
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
# An instrument exercise prints neither colour — the load-bearing half of the artifact
# discipline (#78), and gate_verdict's decision to make; see its definition.
gate_verdict gate-p3 \
  "instrument exercise, KEEL_INSTRUMENT_WIDEN_CI=$INSTRUMENT_WIDEN_CI"
