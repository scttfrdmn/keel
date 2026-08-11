#!/usr/bin/env bash
# Gate P3 — see DESIGN.md §4/P3. Written at the START of phase P3, then made
# green. Exits 0 only when every criterion for the phase holds. A red gate
# blocks the next phase; there is no override flag on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P3):
#   "full `Sgemm` matches oracle across a size sweep (1..17, 63,64,65, 500,
#    1000, 2048, plus transpose/beta/alpha combinations); single-thread >=60%
#    of OpenBLAS at 2048^3 (bench harness pulls OpenBLAS via build-tagged cgo,
#    dev machine only, never a package dependency)."
#
# How those are mechanized, and every judgement call involved:
#
#  1. THE SWEEP'S EXTENT IS ENFORCED, NOT TRUSTED. "matches oracle across a size
#     sweep" is a claim about coverage, and a green `go test` proves only that
#     whatever ran, passed. A test that quietly skipped 2048 because it was slow,
#     or that ran one transpose combination instead of four, reports the same
#     green. So the tests print coverage markers and this gate parses them: every
#     size in DESIGN.md's list must appear, the transpose lattice must be complete
#     (NN/NT/TN/TT), and alpha and beta must each include 0, 1 and a value that is
#     neither — 0 and 1 are the special-cased paths, so a lattice of only those
#     would exercise every shortcut and never the general multiply. The enumerated
#     sets must also multiply out to the reported combination count, so the marker
#     cannot claim combinations it did not run.
#
#  2. THE ORACLE'S COST IS A DECLARED PROPERTY OF EACH SIZE, NOT A SILENT
#     DEGRADATION. A float64 oracle at 2048^3 is 8.6 GFLOP per combination, and
#     the full lattice of that is hours. The sweep is therefore allowed to verify
#     the large sizes by checking a seeded random *sample* of C entries, each one
#     computed exactly in float64 — but only if it says so per size, in a marker
#     this gate reads. Sizes up to 65 must be verified in full; 500, 1000 and 2048
#     may be sampled, with a floor on the sample size and a printed seed so any
#     failure is replayable. The distinction is written here rather than left to
#     the test, because "we sampled" is exactly the kind of concession that starts
#     at 2048 and ends up applying to 17.
#
#  3. CORRECTNESS RUNS WHERE THE SHIPPED PATH RUNS. The dev host is darwin/arm64
#     and cannot execute archsimd at all (docs/toolchain-notes.md T1), so a local
#     `go test` exercises the scalar path and nothing else. The sweep's extent is
#     therefore audited from the log of a host that ran it with the AVX-512
#     backend live, and the scalar path is proved separately by the local run and
#     by a KEEL_FORCE=scalar run on an amd64 host — the P1 mechanism, because the
#     only way to show the fallback works on a machine that has AVX-512 is to make
#     it take the fallback.
#
#  4. P2's KERNEL PROPERTIES ARE RE-CHECKED HERE, BECAUSE P3 IS WHAT WOULD BREAK
#     THEM. Packing, edge handling and beta variants all add code around the
#     K-loop, and the failure mode is a loop that acquired a spill, a call or a
#     bounds check on the way to becoming usable. Those are compile-time
#     properties, so the audit is cheap and it runs on every gate from here on.
#     This is a carried-forward criterion, not a new one, and it does not become
#     optional because P3's own checks are green.
#
#  5. THE THROUGHPUT SENTINEL. The ruling on issue #19 made P2's floor
#     class-dependent: on an issue-bound host it is 0.90 x max_i(f_i.I_i) / I_b,
#     which *rises* as the kernel's instruction count falls, and by the same
#     arithmetic that host is the one that notices when a shape gets fatter. P3 is
#     the phase most likely to fatten one. So P2's verdict is re-run here, through
#     the same unit-tested pure function (scripts/roofline.sh) driven by the same
#     fixtures, which run before any benchmarking.
#
#     The sentinel is named by $KEEL_SENTINEL_HOST or .keel-sentinel (machine-local
#     and gitignored, like .keel-hosts: real hostnames are infrastructure, not
#     source — docs/hosts.md). If neither names one, the check runs on EVERY host.
#     Missing configuration must cost time, not coverage; a gate that skipped its
#     regression check because a file was absent would be a gate that got weaker
#     when someone cloned the repo.
#
#  6. THE OPENBLAS BAR, AND WHICH READING OF "DEV MACHINE ONLY" THIS IMPLEMENTS.
#     DESIGN.md says the OpenBLAS harness is cgo behind a build tag, "dev machine
#     only, never a package dependency". Read as "measure on the dev machine" the
#     criterion is vacuous here: this dev machine is darwin/arm64, where keel's
#     shipped AVX-512 path does not exist, so the ratio would compare
#     OpenBLAS-on-arm64 against keel's scalar fallback and answer a question
#     nobody asked. DESIGN.md §7 rule 7 cuts the same way — a ratio whose two
#     halves come from different silicon is not a ratio.
#
#     So this gate implements the other reading: the *comparison* is dev-only —
#     built behind the `openblas` tag, absent from the module's dependency graph,
#     never linked into anything keel ships — and it runs on the amd64 host where
#     both halves can execute. Concretely:
#       - the reference and keel's Sgemm are measured in the SAME benchmark
#         invocation on the SAME host, so they share a frequency and a thermal
#         state. P2's criterion 5 settled this: a ratio of two numbers taken from
#         separate runs is a worse measurement than either of them.
#       - the harness is built natively on that host from `git archive HEAD`, so
#         the number is attributable to a commit; the working tree must be clean
#         or the archive does not mean what it says.
#       - single-thread is enforced on both sides and *verified* from the
#         harness's own report (OpenBLAS's thread count and GOMAXPROCS). A
#         multi-threaded OpenBLAS would enlarge the denominator and make the bar
#         harder rather than easier — but it would also make it meaningless, and
#         refusing meaningless numbers in both directions is this gate's job.
#       - a host needs a Go toolchain and OpenBLAS for that, and the execution
#         hosts deliberately have neither (docs/hosts.md: cross-compiled static
#         binaries, nothing installed). Provisioning is Scott's to approve, so
#         when no host can produce a reference this gate FAILS and prints the
#         exact commands. It does not fall back to percent-of-peak. CLAUDE.md's
#         "the OpenBLAS reference when available; otherwise say it isn't" is a
#         rule about reporting numbers; using it to satisfy a gate criterion
#         would be weakening the gate, which is the one option never available.
#
#     Every host that has a reference must clear the bar, at least one must exist,
#     and at least one must clear it under the performance governor (DESIGN.md
#     §5.4 rule 5). Percent of measured peak is printed for every host either way,
#     because that number is informative even where it is not a criterion.
#
#  7. WHAT THIS GATE DOES NOT CHECK. "Beta handling as kernel variants, not
#     branches in the loop" and "packing SIMD-accelerated through the shim" are
#     P3 design instructions, not gate criteria. Both appear here as provenance —
#     the variant count and the packing backend come out of the config marker —
#     and criterion 4 enforces them structurally, since a branch or a call that
#     landed in the K-loop is exactly what the audit reports. Anything stronger
#     would be this gate inventing criteria the design document did not set.
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
SWEEP_BEST_IPF=4.438
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
# Exactly one '/' in the pattern, so `go test -bench` splits it into two depth
# elements: {Peak,Sgemm,OpenBLAS} then {avx512,n=2048}. Written as one string
# because those depth semantics bit gate-p2 once already (see its BENCH_FILTER).
SGEMM_BENCH_FILTER='Peak|Sgemm|OpenBLAS/avx512|n=2048'
OPENBLAS_REMOTE_DIR="${KEEL_OPENBLAS_DIR:-/tmp/keel-openblas-src}"

# openblas_hosts prints the hosts that can produce an OpenBLAS reference, i.e.
# those with both a Go toolchain and OpenBLAS. Machine-local like .keel-hosts,
# and for the same reason.
openblas_hosts() {
  if [[ -n "${KEEL_OPENBLAS_HOSTS:-}" ]]; then
    tr ' ' '\n' <<<"$KEEL_OPENBLAS_HOSTS" | sed '/^$/d'
    return
  fi
  [[ -r .keel-openblas-hosts ]] || return 0
  sed -e 's/#.*//' -e '/^[[:space:]]*$/d' -e 's/[[:space:]]//g' .keel-openblas-hosts
}

# sentinel_hosts prints the hosts the P2 throughput regression re-runs on:
# whatever is configured, else every host (criterion 5).
sentinel_hosts() {
  if [[ -n "${KEEL_SENTINEL_HOST:-}" ]]; then
    tr ' ' '\n' <<<"$KEEL_SENTINEL_HOST" | sed '/^$/d'
    return
  fi
  if [[ -r .keel-sentinel ]]; then
    sed -e 's/#.*//' -e '/^[[:space:]]*$/d' -e 's/[[:space:]]//g' .keel-sentinel
    return
  fi
  remote_hosts
}

# marker NAME FILE — the value of the last `keel-NAME:` line in FILE. Test output
# arrives through t.Logf, so the marker may be indented and prefixed.
marker() { sed -n "s/.*keel-$1: *//p" "$2" | tail -1; }

# marker_all NAME FILE — every `keel-NAME:` value, one per line, for the markers
# emitted once per size.
marker_all() { sed -n "s/.*keel-$1: *//p" "$2"; }

# field KEY LINE — the value of a `key=value` token in a marker line.
field() {
  awk -v k="$1" '{
    for (i = 1; i <= NF; i++) {
      n = index($i, "=")
      if (n && substr($i, 1, n - 1) == k) { print substr($i, n + 1); exit }
    }
  }' <<<"$2"
}

# audit_ipf FUNC FILE — that function's audited instructions per FMA, from the
# audit's own integer counts. gate-p2.sh carries the same helper for the same
# reason: this number is a gate input, so it is not read off a rounded display.
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

echo "== gate-p3: packing + blocking -> full Sgemm =="
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
KERNBIN="$BINDIR/kern.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
SWEEPLOG="$BINDIR/sweep-avx512.log"
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"
trap 'rm -rf "$LOG" "$BINDIR"' EXIT

# ------------------------------------------------------- Sgemm vs the oracle
echo
echo "-- Sgemm vs the float64 oracle: size sweep x transpose x alpha x beta --"
info "the local run exercises the scalar path only (darwin/arm64 has no archsimd);"
info "the sweep's extent is audited below from a host that ran it with avx512 live"

LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
if [[ "$LOCAL_OK" -eq 0 ]]; then
  pass "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
else
  fail "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
  sed 's/^/        /' "$LOG" | tail -40
fi
# The scalar path must also pass on a stock toolchain: keel is a pure-Go library
# first and an experiment second.
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
if [[ "$STOCK_OK" -eq 0 ]]; then
  pass "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
else
  fail "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
  sed 's/^/        /' "$LOG" | tail -40
fi

HOSTS="$(remote_hosts)"
AVX512_GREEN=""
SCALAR_FORCED=""
if [[ -z "$HOSTS" ]]; then
  fail "P3 needs an amd64 host to execute the AVX-512 Sgemm; none configured"
else
  if remote_build_test . "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary (root package: Sgemm vs oracle)"
  else
    fail "cross-compile of linux/amd64 test binary"
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
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] Sgemm sweep passes"
    else
      fail "[$host] Sgemm sweep passes"
      sed 's/^/        /' "$LOG" | tail -40
    fi
    backends="$(marker sgemm-backends-exercised "$LOG")"
    if [[ -z "$backends" ]]; then
      fail "[$host] no keel-sgemm-backends-exercised marker: coverage unknown is coverage unestablished"
    else
      info "[$host] backends exercised: $backends"
    fi
    if [[ "$OK" -eq 0 && " $backends " == *" avx512 "* ]]; then
      AVX512_GREEN="$host"
      cp "$LOG" "$SWEEPLOG"
    fi
    # The fallback, proved by taking it on a machine that has the alternative.
    FOK=0
    KEEL_REMOTE_ENV="KEEL_FORCE=scalar" remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || FOK=$?
    if [[ "$FOK" -eq 0 ]]; then
      pass "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      SCALAR_FORCED="$host"
    else
      fail "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      sed 's/^/        /' "$LOG" | tail -20
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the sweep ran green with the avx512 Sgemm live (target: $AVX512_GREEN)"
  else
    fail "no target ran the Sgemm sweep green with the avx512 backend"
  fi
  [[ -n "$SCALAR_FORCED" ]] || fail "no host proved the scalar fallback under KEEL_FORCE=scalar"
fi

# --------------------------------------------------- what the sweep covered
echo
echo "-- sweep extent (criteria 1 and 2: coverage is enforced, not trusted) --"
if [[ ! -s "$SWEEPLOG" ]]; then
  fail "no avx512 sweep log to audit; the sweep's extent is unverified"
else
  cfg="$(marker sgemm-config "$SWEEPLOG")"
  if [[ -z "$cfg" ]]; then
    fail "no keel-sgemm-config marker (blocking params, edge strategy, beta variants, packing backend)"
  else
    info "config: $cfg"
    for k in mr nr kc mc nc edge beta-variants pack; do
      [[ -n "$(field "$k" "$cfg")" ]] || fail "keel-sgemm-config is missing $k="
    done
  fi

  sizes="$(marker sgemm-sizes-exercised "$SWEEPLOG")"
  if [[ -z "$sizes" ]]; then
    fail "no keel-sgemm-sizes-exercised marker"
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
    fail "no keel-sgemm-combos-exercised marker"
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
      fail "keel-sgemm-combos-exercised has no combos= count"
    fi
  fi

  # Per-size oracle verification mode (criterion 2).
  VMISS=""; VBAD=""
  for n in $SWEEP_SIZES; do
    line="$(marker_all sgemm-verify "$SWEEPLOG" | awk -v want="size=$n" '{ for (i=1;i<=NF;i++) if ($i == want) { print; exit } }')"
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
    [[ -n "$VMISS" ]] && fail "no keel-sgemm-verify line for size(s):$VMISS"
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
    fail "no keel-pack-combos-exercised marker: packing is not differential-tested"
  fi
fi

# ------------------------------ P2's compile-time properties, carried forward
echo
echo "-- carried from P2 (criterion 4): the K-loop after packing and blocking --"
info "compile-time property, audited against the linux/amd64 object code the hosts run"
if GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
     -pkg "$KERN_PKG" -func "$KERN_FUNCS" -mode spill -ssa "$SSADIR" >"$AUDITKERN" 2>&1; then
  sed 's/^/        /' "$AUDITKERN"
  pass "0 accumulator spills in the steady-state K-loop (P2 property held)"
  pass "0 calls in the steady-state K-loop (P2 property held)"
  pass "0 surviving bounds checks in the steady-state K-loop (P2 property held)"
else
  sed 's/^/        /' "$AUDITKERN"
  fail "P3 broke a P2 kernel property; the audit above says which"
fi
if go run ./internal/spill/cmd/spill-audit \
     -pkg ./internal/vec -func "$PEAK_FUNCS" -mode nomemory >"$AUDITPEAK" 2>&1; then
  pass "every peak kernel's steady-state loop is still register-only (the denominator is still a ceiling)"
else
  sed 's/^/        /' "$AUDITPEAK"
  fail "a peak kernel's loop touches memory; the percent-of-peak denominator is not a ceiling"
fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 \
     go build -gcflags='-d=ssa/check_bce' "$KERN_PKG" ./internal/kern ./internal/pack ./internal/block 2>"$LOG"; then
  BCE_N="$(grep -c 'Found Is\(Slice\)\?InBounds' "$LOG" || true)"
  info "check_bce: ${BCE_N:-0} bounds check(s) across vec+kern+pack+block, all outside the K-loop (provenance; the criterion is the loop-body audit above)"
  pass "check_bce output recorded as provenance"
else
  sed 's/^/        /' "$LOG" | tail -20
  fail "build with -d=ssa/check_bce failed"
fi

# ------------------------------------------ the throughput sentinel (P2, #19)
echo
echo "-- throughput sentinel (criterion 5): P2's floor re-run, so a fatter K-loop is noticed --"
if RTLOG="$(bash scripts/roofline-test.sh 2>&1)"; then
  pass "roofline verdict controls ($(grep -c '^  ok ' <<<"$RTLOG") fixtures)"
else
  fail "roofline verdict controls"
  # shellcheck disable=SC2001  # prefixing every line; not a scalar substitution
  sed 's/^/        /' <<<"$RTLOG"
fi
IPF_2x32="$(audit_ipf Kernel2x32 "$AUDITKERN")"
IPF_4x32="$(audit_ipf Kernel4x32 "$AUDITKERN")"
IPF_PEAK="$(audit_ipf "$GATE_PEAK_FUNC" "$AUDITPEAK")"
if [[ -n "$IPF_2x32" && -n "$IPF_4x32" && -n "$IPF_PEAK" ]]; then
  info "audited insns/FMA: 2x32 $(printf '%.3f' "$IPF_2x32"), 4x32 $(printf '%.3f' "$IPF_4x32"), $GATE_PEAK_FUNC $(printf '%.3f' "$IPF_PEAK")"
else
  fail "could not read insns/FMA from the audits; the sentinel cannot classify its host"
fi
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

SENTINELS="$(sentinel_hosts)"
if [[ -z "$SENTINELS" ]]; then
  fail "no sentinel host and no hosts at all; P2's floor cannot be re-checked"
else
  if [[ -z "${KEEL_SENTINEL_HOST:-}" && ! -r .keel-sentinel ]]; then
    info "no sentinel configured, so every host is one: $(tr '\n' ' ' <<<"$SENTINELS")"
  fi
  if ! remote_build_test ./bench "$KERNBIN" >"$LOG" 2>&1; then
    fail "cross-compile of the linux/amd64 bench binary (kernel benchmarks)"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    if ! remote_exec "$host" "$KERNBIN" "${BFLAGS[@]}" -test.bench="$KERN_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      fail "[$host] sentinel: kernel benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    BEST_LO=""; BEST_PT=""; BEST_ID=""; BEST_IPF=""; MIXES=""
    for kname in $GATE_KERNELS; do
      [[ -n "$(bench_stat "$kname" "$BENCHCSV" GFLOP/s)" ]] || continue
      klo="$(bench_ratio_lo "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      kpt="$(bench_ratio "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$klo" ]]; then
        info "[$host] ${kname##Kernel/}: no CI, not counted"
        continue
      fi
      case "$kname" in
        *2x32*) kipf="$IPF_2x32" ;;
        *4x32*) kipf="$IPF_4x32" ;;
        *)      kipf="" ;;
      esac
      [[ -n "$kipf" ]] && MIXES="$MIXES ${kname##Kernel/}|$kpt:$kipf"
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_ID="${kname##Kernel/}"; BEST_IPF="$kipf"
      fi
    done
    [[ -n "$IPF_PEAK" ]] && MIXES="$MIXES $GATE_PEAK_FUNC|1.0:$IPF_PEAK"
    if [[ -z "$BEST_LO" ]]; then
      fail "[$host] sentinel: no bounded percent-of-peak for any shipped shape"
      continue
    fi
    # The ceiling set is every mix except the shape under test; see gate-p2.sh
    # criterion 5b and scripts/roofline.sh (INDEPENDENCE).
    CEIL=""
    for mx in $MIXES; do
      [[ "${mx%%|*}" == "$BEST_ID" ]] && continue
      CEIL="$CEIL ${mx#*|}"
    done
    # shellcheck disable=SC2086  # CEIL is a deliberate list of f:I words
    read -r CLASS _CSPREAD _MSPREAD ROOF ATTAIN RESULT WHY <<<"$(
      throughput_verdict "$BEST_LO" "${BEST_IPF:-0}" \
        "$PEAK_FLOOR" "$ROOF_FLOOR" "$ISSUE_CONVERGE_MAX" \
        "$ISSUE_MIX_SPREAD_MIN" "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" $CEIL)"
    frac="$(awk -v r="$BEST_LO" 'BEGIN{printf "%.1f", r * 100}')"
    fracpt="$(awk -v r="$BEST_PT" 'BEGIN{printf "%.1f", r * 100}')"
    attpc="$(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a * 100}')"
    roofpc="$(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r * 100}')"
    case "$CLASS/$RESULT" in
      issue/pass)
        pass "[$host] sentinel: $BEST_ID holds P2's floor — ${fracpt}% of peak (${frac}% net of CI) = ${attpc}% of its ${roofpc}% issue roofline (>= 90%)" ;;
      */pass)
        pass "[$host] sentinel: $BEST_ID holds P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (>= 55%)" ;;
      */refuse)
        fail "[$host] sentinel: $BEST_ID at $(printf '%.3f' "${BEST_IPF:-0}") insns/FMA is outside the shape guard — P3 fattened the K-loop (why=$WHY)" ;;
      *)
        fail "[$host] sentinel: $BEST_ID fell below P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (why=$WHY)" ;;
    esac
  done <<<"$SENTINELS"
fi

# ----------------------------------------------- Sgemm at 2048^3 vs OpenBLAS
echo
echo "-- Sgemm at 2048^3: percent of measured peak, and >= 60% of single-thread OpenBLAS --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; the bar counts as cleared only net of both confidence intervals"

if [[ -n "$HOSTS" ]]; then
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary (Sgemm + peak)"
  else
    fail "cross-compile of linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" \
         -test.bench="$SGEMM_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      fail "[$host] Sgemm benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    if [[ -z "$(bench_stat "$GATE_SGEMM" "$BENCHCSV" GFLOP/s)" ]]; then
      fail "[$host] no $GATE_SGEMM benchmark result"
      continue
    fi
    info "[$host] Sgemm 2048^3 $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s), peak $(bench_describe "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pk="$(bench_ratio "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    if [[ -n "$pk" ]]; then
      info "[$host] = $(awk -v r="$pk" 'BEGIN{printf "%.1f", r*100}')% of measured peak ($(awk -v r="${pklo:-0}" 'BEGIN{printf "%.1f", r*100}')% net of CI) — reported, not a P3 criterion"
    fi
  done <<<"$HOSTS"
fi

# The reference: same host, same invocation, built natively behind the cgo tag.
OB_HOSTS="$(openblas_hosts)"
OB_CLEARED=0
OB_PERF_GOV=""
if [[ -z "$OB_HOSTS" ]]; then
  fail "no OpenBLAS reference host configured, so the >= 60%-of-OpenBLAS criterion cannot be evaluated (percent-of-peak is NOT a substitute)"
  info "This criterion needs one amd64 host with a Go toolchain and OpenBLAS. The"
  info "execution hosts deliberately have neither (docs/hosts.md: cross-compiled"
  info "static binaries, nothing installed), and provisioning is Scott's to approve."
  info "Exact commands, for one host:"
  info "  Ubuntu:  sudo apt-get install -y libopenblas-dev"
  info "  RHEL 9:  sudo dnf install -y openblas-devel"
  info "  plus a go1.26.5+ toolchain (GOEXPERIMENT=simd support): distro packages"
  info "  are likely older, so install from go.dev/dl and put it on PATH."
  info "Then name the host in .keel-openblas-hosts (gitignored) or \$KEEL_OPENBLAS_HOSTS."
elif [[ -n "$(git status --porcelain)" ]]; then
  fail "the working tree is dirty, so \`git archive HEAD\` would measure something other than what is here; commit first"
else
  while read -r host; do
    [[ -n "$host" ]] || continue
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    info "[$host] governor=${gov:-unknown}; building the openblas-tagged harness natively from git archive HEAD ($(git rev-parse --short HEAD))"
    # KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, which would close
    # stdin and hand tar an empty archive. The remote-side paths below expand
    # here, on the client, which is what is wanted — they are this script's
    # variables, not the remote shell's.
    # shellcheck disable=SC2029
    if ! git archive --format=tar HEAD | ssh "${KEEL_SCP_OPTS[@]}" "$host" \
         "rm -rf '$OPENBLAS_REMOTE_DIR' && mkdir -p '$OPENBLAS_REMOTE_DIR' && tar -x -C '$OPENBLAS_REMOTE_DIR'" >"$LOG" 2>&1; then
      fail "[$host] could not ship the source tree for a native build"
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
    OBARGS=""
    for a in "${BFLAGS[@]}" "-test.bench=$SGEMM_BENCH_FILTER"; do OBARGS+=" $(printf '%q' "$a")"; done
    # shellcheck disable=SC2029  # client-side expansion of a client-side path
    if ! ssh "${KEEL_SSH_OPTS[@]}" "$host" \
         "cd '$OPENBLAS_REMOTE_DIR' && env GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 ./bench-ob.test$OBARGS" >"$BENCHLOG" 2>&1; then
      fail "[$host] the openblas-tagged benchmark run failed"
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
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    if [[ -z "$(bench_stat "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)" ]]; then
      fail "[$host] no $GATE_OPENBLAS benchmark result to divide by"
      continue
    fi
    info "[$host] keel $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s) vs OpenBLAS $(bench_describe "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s), one invocation"
    rlo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    rpt="$(bench_ratio "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    if [[ -z "$rlo" ]]; then
      fail "[$host] no bounded keel/OpenBLAS ratio: benchstat established no confidence interval, which is a failure to measure rather than a pass"
      continue
    fi
    rlopc="$(awk -v r="$rlo" 'BEGIN{printf "%.1f", r*100}')"
    rptpc="$(awk -v r="$rpt" 'BEGIN{printf "%.1f", r*100}')"
    if awk -v r="$rlo" -v f="$OPENBLAS_FLOOR" 'BEGIN{exit !(r >= f)}'; then
      pass "[$host] Sgemm at 2048^3 is ${rptpc}% of single-thread OpenBLAS, ${rlopc}% net of CI (>= 60%)"
      OB_CLEARED=$((OB_CLEARED + 1))
      [[ "$gov" == "performance" ]] && OB_PERF_GOV="$host"
    else
      fail "[$host] Sgemm at 2048^3 is only ${rptpc}% of single-thread OpenBLAS, ${rlopc}% net of CI (< 60%)"
    fi
  done <<<"$OB_HOSTS"
  if [[ "$OB_CLEARED" -eq 0 ]]; then
    fail "no host cleared the 60%-of-OpenBLAS bar"
  fi
  if [[ -n "$OB_PERF_GOV" ]]; then
    pass "the OpenBLAS bar was cleared under the performance governor ($OB_PERF_GOV)"
  else
    fail "no host cleared the OpenBLAS bar under the performance governor (§5.4 rule 5 requires one)"
  fi
fi

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p3: GREEN"
  exit 0
fi
echo "gate-p3: RED" >&2
exit 1
