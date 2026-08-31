#!/usr/bin/env bash
# Gate P4 — DESIGN.md §4/P4. Exits 0 only when every criterion for the phase
# holds; a red gate blocks the next phase, and there is no override flag.
#
# The 226 lines of front matter that used to sit here — what each criterion
# measures, what this gate refuses to decide, and the judgement call behind
# every threshold below — moved verbatim to docs/gates.md, section "P4", on
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
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0
# `unmeasured` is not defined here. It was, and #72 lifted it to scripts/remote.sh
# once 21 further sites across three gates turned out to need it: three copies of a
# verdict primitive is how the delegated tally came to count two columns where the
# log had three. Same effect on this gate's verdict as `fail` — DESIGN.md §5.6, an
# unmeasured criterion may not resolve as either colour, and this gate's colour is
# binary — with a label that distinguishes "Ssyrk is too slow" from "this reading
# cannot decide".

# ------------------------------------------------------------- the P4 routines
# Level 2 first, then the three Level-3 routines derived from P3's GEMM. The order
# is the order DESIGN.md §4/P4 lists them in, and it is also increasing order of
# how much of the loop nest each one reuses.
P4_ROUTINES="Sgemv Sger Ssyrk Ssymm Strsm"
# The three that must report P3's microkernel and P3's blocking parameters
# (criterion 5). Sgemv and Sger have no microkernel to report — they derive from
# Level 1 — so their config marker names an L1 backend instead.
P4_DERIVED_L3="Ssyrk Ssymm Strsm"

# lattice_req ROUTINE — this gate's own statement of the flag lattice, one
# "key:requirement" per line. Written from the BLAS definitions, not read from the
# test, because a requirement copied out of the thing it constrains constrains
# nothing.
#
# Two requirements are symbolic rather than literal:
#   @scalar  must include 0, 1 and a value that is neither (the special-cased
#            paths plus the general multiply)
#   @stride  must include 1, a stride greater than 1, and a negative stride
#            (unit, gathered, and BLAS's backwards vector)
lattice_req() {
  case "$1" in
    Sgemv) printf '%s\n' "trans:N,T" "alpha:@scalar" "beta:@scalar" "incx:@stride" "incy:@stride" ;;
    Sger)  printf '%s\n' "alpha:@scalar" "incx:@stride" "incy:@stride" ;;
    Ssyrk) printf '%s\n' "uplo:U,L" "trans:N,T" "alpha:@scalar" "beta:@scalar" ;;
    Ssymm) printf '%s\n' "side:L,R" "uplo:U,L" "alpha:@scalar" "beta:@scalar" ;;
    Strsm) printf '%s\n' "side:L,R" "uplo:U,L" "trans:N,T" "diag:N,U" "alpha:@scalar" ;;
  esac
}

# extra_req ROUTINE — the edge coverage required beyond the size x flag lattice.
# See criterion 6 for what each of the routine-specific ones proves and why an
# oracle comparison cannot see it.
extra_req() {
  case "$1" in
    Sgemv|Sger) printf '%s\n' ldpad zerodim argpanic nonfinite ;;
    Ssyrk)      printf '%s\n' ldpad zerodim argpanic nonfinite untouched-triangle ;;
    Ssymm)      printf '%s\n' ldpad zerodim argpanic nonfinite unreferenced-triangle ;;
    Strsm)      printf '%s\n' ldpad zerodim argpanic nonfinite unreferenced-triangle unit-diagonal-ignored ;;
  esac
}

# The sizes every P4 routine must sweep, and the exact/sampled boundary. See
# criterion 2 for why the list stops at 500 where P3's went to 2048.
P4_SIZES="1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 31 32 33 63 64 65 500"
P4_EXACT_MAX=65
P4_SAMPLE_MIN=256
# The backends every routine's runners must have exercised on an AVX-512 host: the
# widest and the scalar reference at minimum, which is what makes the comparison
# differential rather than merely oracle-checked (DESIGN.md §5 rule 2).
P4_BACKENDS="avx512 scalar"

# ------------------------------------------------------ carried from P2 (#19)
# Same shapes, same audit, same constants as gate-p2.sh and gate-p3.sh. Duplicated
# rather than factored on purpose, and the reason is unchanged: a P4 run should fail
# because P4 changed the kernel, not because someone edited one shared constant, and
# two independent statements of the property is what makes that visible.
KERN_PKG="./internal/vec"
KERN_FUNCS="Kernel2x32,Kernel4x32"
PEAK_FUNCS="avx512Peak,avx2Peak,scalarPeak"
SSADIR="build/ssa"

# ------------------------------------------------------------- P4's own bar
SYRK_FLOOR=0.85
GATE_SGEMM="Sgemm/n=2048"
GATE_SSYRK="Ssyrk/n=2048"
GATE_PEAK="Peak/avx512"
# THE PARENTHESES ARE LOAD-BEARING (issue #32, docs/toolchain-notes.md T15).
# `go test -bench` splits on top-level '|' FIRST, into an alternation of whole
# patterns, and only then splits each alternative on '/'. Unparenthesized, this
# would be four independent depth-unconstrained patterns and would run benchmarks
# nothing here reads while missing the ones it does.
P4_BENCH_FILTER='(Peak|Sgemm|Ssyrk)/(avx512|n=2048)'
# The delegated P3 gate's full output. build/ is gitignored; the path is printed
# because CLAUDE.md wants gate output verbatim in the umbrella issue.
# Revision-stamped for #78's reason — see the same assignment in gate-p5.sh.
# Run-stamped too, for the reason the same assignment in gate-p5.sh now records: the
# revision stamp left two runs at one rev colliding, which is #78 past its own fix.
P3LOG="build/gate-p3-under-p4-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)-$RUN_STAMP.log"

# p4_line NAME FILE ROUTINE — the keel-NAME line belonging to one routine. The P4
# markers are emitted once per routine, so `marker`'s last-wins reading would
# silently audit Strsm's coverage as if it were Sgemv's.
p4_line() { marker_row "$1" "$2" routine "$3"; }

# p4_verify_line FILE ROUTINE SIZE — the verification-mode line for one
# (routine, size) pair, of which there is one per size per routine.
p4_verify_line() {
  marker_all p4-verify "$1" | awk -v r="routine=$2" -v s="size=$3" '{
    rok = 0; sok = 0
    for (i = 1; i <= NF; i++) { if ($i == r) rok = 1; if ($i == s) sok = 1 }
    if (rok && sok) { print; exit }
  }'
}

# set_scalar_ok SET — 0, 1 and a general value are all present. Compared
# numerically, so "0", "-0" and "0.0" are one value and "1e0" is 1.
set_scalar_ok() {
  awk -v v="$1" 'BEGIN {
    n = split(v, a, ",")
    for (i = 1; i <= n; i++) {
      if (a[i] + 0 == 0) z = 1
      else if (a[i] + 0 == 1) o = 1
      else g = 1
    }
    exit !(z && o && g)
  }'
}

# set_stride_ok SET — a unit stride, a wider one, and a negative one.
set_stride_ok() {
  awk -v v="$1" 'BEGIN {
    n = split(v, a, ",")
    for (i = 1; i <= n; i++) {
      if (a[i] + 0 == 1) one = 1
      else if (a[i] + 0 > 1) wide = 1
      else if (a[i] + 0 < 0) neg = 1
    }
    exit !(one && wide && neg)
  }'
}

# lattice_product LINE — "product factors", the product of every enumerated set on
# a lattice line and how many sets that was, ignoring routine= and combos=.
#
# Taking the product over EVERY other key is what makes the check indifferent to
# which dimensions a routine happens to have: a test that adds a stride dimension,
# or one that sweeps a new flag, has to grow its combination count or say a number
# that does not match its own enumeration. A fixed list of factors per routine
# would have to be maintained here and would silently stop covering the thing it
# was written for.
lattice_product() {
  awk '{
    prod = 1; sets = 0
    for (i = 1; i <= NF; i++) {
      p = index($i, "=")
      if (!p) continue
      k = substr($i, 1, p - 1)
      if (k == "routine" || k == "combos") continue
      prod *= split(substr($i, p + 1), a, ",")
      sets++
    }
    printf "%d %d", prod, sets
  }' <<<"$1"
}

echo "== gate-p4: Level 2 + derived Level 3 =="
echo

# ------------------------------------------------------------- tree state (#63)
assert_no_strays

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

# The delegated P3 gate measures `git archive HEAD`, so a dirty tree makes it fail
# by construction. Checked here, before anything expensive, because arriving at a
# knowable failure after a full gate run is a waste rather than a finding.
TREE_CLEAN=1
if [[ -n "$(git status --porcelain)" ]]; then
  TREE_CLEAN=0
  fail "the working tree is dirty, so the delegated P3 gate (criterion 8) cannot run: its OpenBLAS reference is built from \`git archive HEAD\`, which would measure something other than what is here"
  info "  commit first; this gate's own criteria still run below"
fi

gate_tmpdir
SWEEPLOG="$BINDIR/sweep-avx512.log"
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"

# --------------------------------------------- the routines vs the float64 oracle
echo
echo "-- Level 2 and derived Level 3 vs the float64 oracle --"
info "the local run exercises the scalar path only (darwin/arm64 has no archsimd);"
info "every lattice below is audited from a host that ran it with avx512 live"

LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
test_verdict "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$LOG" "$LOCAL_OK" "all tests pass"
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
test_verdict "local, stock toolchain" "$LOG" "$STOCK_OK" "all tests pass without GOEXPERIMENT=simd"

resolve_fleet

# ---- the measurement precondition, asserted rather than assumed
#
# DESIGN.md §5 rule 5 as amended: EVERY measuring host's clock established stable,
# per host at the start of the gate and again at the moment of measurement, never
# satisfied by one host on behalf of another. This gate takes its own measurements
# (criterion 7), so it makes its own assertion rather than relying on the delegated
# P3 gate's. Which instrument establishes it is a property of the host: a readable
# `cpufreq` — every host this gate has run against — means the `performance`
# governor, asserted below. Unreadable still counts as unmet, an unverified
# precondition not being a met one. What the amendment changed is that a VM is no
# longer the same case: rule 5 now sends a guest, which has no `cpufreq` directory at
# all, to `BenchmarkPeak` sampled head/middle/tail, and rule 6 forbids "no interface"
# and "present but unreadable" sharing one verdict — the first is a guest, the second
# is a defect. That branch is not yet implemented here, so a guest blocks as
# `unmeasured`; the sentence this replaced read the two as one and called both unmet. The
# check itself now lives in remote.sh's assert_governor: this gate was the master the
# other three copied, and all four shared a mislabel none of them could reveal (#83).
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    assert_governor "$host" preamble
  done <<<"$HOSTS"
fi

AVX512_GREEN=""
if [[ -z "$HOSTS" ]]; then
  unmeasured "P4 needs an amd64 host to execute the AVX-512 paths and none is configured, so they are unmeasured on real silicon"
else
  remote_build_test_or_fail . "$BIN" "$LOG" \
    "cross-compiled linux/amd64 test binary (root package: P4 routines vs oracle)" \
    "cross-compile of linux/amd64 test binary"
  while read -r host; do
    [[ -n "$host" ]] || continue
    probe_or_unmeasured "$host" || continue
    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] the P4 lattices pass against the oracle"
    elif remote_vanished; then
      unmeasured "[$host] the P4 lattices did not finish (#62), so this host says nothing about them either way"
    else
      fail "[$host] the P4 lattices pass against the oracle"
      sed 's/^/        /' "$LOG" | tail -60
    fi
    active="$(marker sgemm-active "$LOG")"
    if [[ "$OK" -eq 0 && "$active" == */avx512 ]]; then
      AVX512_GREEN="$host"
      cp "$LOG" "$SWEEPLOG"
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the lattices ran green with the avx512 microkernel live (target: $AVX512_GREEN)"
  else
    unmeasured "no avx512-green lattice run to audit, so every extent check below it is unmeasured (a fleet that ran green without avx512 is a separate verdict, and the per-host lines above carry it)"
  fi
fi

# ------------------------------------------ what the lattices actually covered
echo
echo "-- lattice extent (criteria 1, 2, 3, 5 and 6: coverage is enforced, not trusted) --"
if [[ ! -s "$SWEEPLOG" ]]; then
  unmeasured "no avx512 lattice log to audit, so every routine's coverage is unmeasured"
else
  L3_KERN="$(marker sgemm-active "$SWEEPLOG")"
  L3_CFG="$(marker sgemm-config "$SWEEPLOG")"
  L1_ACTIVE="$(marker l1-active "$SWEEPLOG")"
  info "P3's dispatched microkernel on this host: ${L3_KERN:-<no keel-sgemm-active marker>}"
  [[ -n "$L3_KERN" ]] || unmeasured "no keel-sgemm-active marker in the lattice log, so the derived routines' shape cannot be compared against Sgemm's (criterion 5)"

  for r in $P4_ROUTINES; do
    # ---- the flag lattice (criterion 1)
    lat="$(p4_line p4-lattice "$SWEEPLOG" "$r")"
    if [[ -z "$lat" ]]; then
      unmeasured "$r: no keel-p4-lattice marker, so the flags it swept cannot be read — coverage unknown is coverage unestablished"
    else
      info "$r lattice: ${lat#routine="$r" }"
      LBAD=""
      while read -r req; do
        [[ -n "$req" ]] || continue
        key="${req%%:*}"; want="${req#*:}"
        got="$(field "$key" "$lat")"
        if [[ -z "$got" ]]; then
          LBAD="$LBAD ${key}(absent: the routine has this flag and the sweep does not say it varied it)"
          continue
        fi
        case "$want" in
          @scalar)
            set_scalar_ok "$got" || LBAD="$LBAD ${key}=${got}(needs 0, 1 and a general value: 0 and 1 are the special-cased paths)" ;;
          @stride)
            set_stride_ok "$got" || LBAD="$LBAD ${key}=${got}(needs 1, a stride > 1 and a negative stride)" ;;
          *)
            for m in ${want//,/ }; do
              set_has "$got" "$m" || LBAD="$LBAD ${key}=${got}(missing $m)"
            done ;;
        esac
      done < <(lattice_req "$r")
      if [[ -z "$LBAD" ]]; then
        pass "$r: every flag in this gate's statement of the lattice was swept"
      else
        fail "$r: the flag lattice is incomplete:$LBAD"
      fi
      # The enumerated sets must multiply out to the reported count.
      ncombo="$(field combos "$lat")"
      read -r LPROD LSETS <<<"$(lattice_product "$lat")"
      if [[ -z "$ncombo" ]]; then
        unmeasured "$r: keel-p4-lattice has no combos= count, so its enumeration cannot be checked against what it ran"
      elif [[ "${LSETS:-0}" -eq 0 ]]; then
        fail "$r: keel-p4-lattice enumerates no flag sets at all"
      elif [[ "$ncombo" -eq "$LPROD" ]]; then
        pass "$r: combination count matches the enumerated sets ($ncombo = $LPROD over $LSETS dimensions)"
      else
        fail "$r: combination count $ncombo does not match its own enumeration ($LPROD over $LSETS dimensions): the marker claims combinations it did not run, or swept a dimension at one value"
      fi
    fi

    # ---- sizes and the oracle's cost (criterion 2)
    szline="$(p4_line p4-sizes "$SWEEPLOG" "$r")"
    sizes="$(field sizes "$szline")"
    if [[ -z "$sizes" ]]; then
      unmeasured "$r: no keel-p4-sizes marker, so the sizes it ran cannot be read"
    else
      SMISS=""
      for n in $P4_SIZES; do
        set_has "$sizes" "$n" || SMISS="$SMISS $n"
      done
      if [[ -z "$SMISS" ]]; then
        pass "$r: every size this gate requires ran (1..17, 31/32/33, 63/64/65, 500)"
      else
        fail "$r: sizes missing from the sweep:$SMISS"
      fi
    fi
    VMISS=""; VBAD=""
    for n in $P4_SIZES; do
      vline="$(p4_verify_line "$SWEEPLOG" "$r" "$n")"
      if [[ -z "$vline" ]]; then
        VMISS="$VMISS $n"
        continue
      fi
      mode="$(field mode "$vline")"
      if [[ "$n" -le "$P4_EXACT_MAX" ]]; then
        [[ "$mode" == exact ]] || VBAD="$VBAD ${n}:${mode:-none}(must be exact)"
        continue
      fi
      case "$mode" in
        exact) ;;
        sampled)
          s="$(field n "$vline")"
          if [[ -z "$s" ]] || [[ "$s" -lt "$P4_SAMPLE_MIN" ]]; then
            VBAD="$VBAD ${n}:sampled(${s:-0} < $P4_SAMPLE_MIN)"
          fi
          [[ -n "$(field seed "$vline")" ]] || VBAD="$VBAD ${n}:sampled(no seed= to replay with)" ;;
        *) VBAD="$VBAD ${n}:${mode:-none}" ;;
      esac
    done
    if [[ -z "$VMISS" && -z "$VBAD" ]]; then
      pass "$r: every size declares its oracle verification mode (exact up to $P4_EXACT_MAX, >= $P4_SAMPLE_MIN seeded exact entries above it)"
    else
      [[ -n "$VMISS" ]] && unmeasured "$r: no keel-p4-verify line for size(s):$VMISS — how those sizes were verified cannot be read"
      [[ -n "$VBAD" ]] && fail "$r: oracle verification too weak for size(s):$VBAD"
    fi

    # ---- differential across backends (criterion 3)
    bk="$(field backends "$(p4_line p4-backends "$SWEEPLOG" "$r")")"
    if [[ -z "$bk" ]]; then
      unmeasured "$r: no keel-p4-backends marker, so whether it was compared against another backend and not only the oracle cannot be read (§5 rule 2)"
    else
      BMISS=""
      for b in $P4_BACKENDS; do
        set_has "$bk" "$b" || BMISS="$BMISS $b"
      done
      if [[ -z "$BMISS" ]]; then
        pass "$r: differential across backends ($bk)"
      else
        fail "$r: backends missing from the differential test:$BMISS (ran: $bk)"
      fi
    fi

    # ---- the edge coverage no lattice contains (criterion 6)
    ex="$(field extras "$(p4_line p4-extra "$SWEEPLOG" "$r")")"
    if [[ -z "$ex" ]]; then
      unmeasured "$r: no keel-p4-extra marker, so its edge coverage cannot be read"
    else
      EMISS=""
      while read -r e; do
        [[ -n "$e" ]] || continue
        set_has "$ex" "$e" || EMISS="$EMISS $e"
      done < <(extra_req "$r")
      if [[ -z "$EMISS" ]]; then
        pass "$r: edge coverage beyond the lattice ($ex)"
      else
        fail "$r: required edge coverage missing:$EMISS (ran: $ex)"
      fi
    fi

    # ---- derived from P3's GEMM, not reimplemented beside it (criterion 5)
    cfg="$(p4_line p4-config "$SWEEPLOG" "$r")"
    if [[ -z "$cfg" ]]; then
      unmeasured "$r: no keel-p4-config marker, so what it derives from cannot be read"
      continue
    fi
    if [[ " $P4_DERIVED_L3 " == *" $r "* ]]; then
      rk="$(field kern "$cfg")"
      if [[ -z "$rk" ]]; then
        unmeasured "$r: keel-p4-config has no kern=, so this gate cannot tell a blocked derivation from a reimplementation"
      elif [[ -n "$L3_KERN" && "$rk" != "$L3_KERN" ]]; then
        fail "$r: runs microkernel $rk where Sgemm in the same run dispatched $L3_KERN — a second kernel family, or a shape chosen by something other than the classification P3's gate checks"
      else
        pass "$r: derived on the same microkernel Sgemm dispatched ($rk), path=$(field path "$cfg")"
      fi
      PBAD=""
      for p in kc mc nc; do
        rv="$(field "$p" "$cfg")"; sv="$(field "$p" "$L3_CFG")"
        if [[ -z "$rv" ]]; then
          PBAD="$PBAD ${p}(absent)"
        elif [[ -n "$sv" && "$rv" != "$sv" ]]; then
          PBAD="$PBAD ${p}($rv vs Sgemm's $sv)"
        fi
      done
      if [[ -z "$PBAD" ]]; then
        pass "$r: blocked with Sgemm's own parameters (kc=$(field kc "$cfg") mc=$(field mc "$cfg") nc=$(field nc "$cfg"))"
      else
        fail "$r: blocking parameters differ from Sgemm's or are unreported:$PBAD — P5 tunes one set of parameters, not four"
      fi
    else
      rl1="$(field l1 "$cfg")"
      if [[ -z "$rl1" ]]; then
        unmeasured "$r: keel-p4-config has no l1=, so what its inner loop derives from cannot be read"
      elif [[ -n "$L1_ACTIVE" && "$rl1" != "$L1_ACTIVE" ]]; then
        fail "$r: runs Level-1 backend $rl1 where this run's dispatched backend is $L1_ACTIVE"
      else
        pass "$r: derived on the dispatched Level-1 backend ($rl1), path=$(field path "$cfg")"
      fi
    fi
  done
fi

carry_p2_properties P4 "triangular masking"

# ------------------------------------- Ssyrk >= 85% of Sgemm at the same size
echo
echo "-- Ssyrk vs Sgemm at 2048 (criterion 7): one invocation, both flop counts checked --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; the bar counts as cleared only net of both confidence intervals"

BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)
SYRK_CLEARED=0
SYRK_MEASURED=0
# Three outcomes are counted separately because they have three different remedies:
# cleared needs nothing, missed is a kernel to fix, indeterminate is precision to
# buy (#67). Collapsing the last two is the defect this criterion had.
SYRK_MISSED=0
SYRK_INDET=0
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"
if [[ -z "$HOSTS" ]]; then
  unmeasured "no execution hosts, so the Ssyrk/Sgemm ratio cannot be evaluated: unmeasured, not missed"
else
  remote_build_test_or_fail ./bench "$BENCHBIN" "$LOG" \
    "cross-compiled linux/amd64 bench binary (Sgemm + Ssyrk + peak)" \
    "cross-compile of linux/amd64 bench binary"
  DRIFT_CHECKED=""
  while read -r host; do
    [[ -n "$host" ]] || continue
    # Re-read at the moment of measurement, not merely in the preamble: a governor
    # that changed in between belongs to a machine somebody started using, and the
    # reading it produces is not one §5 rule 5 covers. This site used to print no
    # provenance line at all — the lifted helper prints one, so a passing host's
    # silent success still leaves a record of the value it passed on.
    assert_governor "$host" measured
    clock_gate "$host" || continue
    clock_head "$host" "$BENCHBIN" || continue
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" \
         -test.bench="$P4_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the Ssyrk/Sgemm benchmark run failed, so this host's ratio is unmeasured"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    # ---- criterion 4's registry drift check.
    # IT RUNS BEFORE THE ROW DECLARATION BELOW, and the order is the point. This
    # check reads a provenance marker the harness prints at startup; it does not
    # divide by any benchmark. Sitting after require_bench, a missing Ssyrk row
    # would skip it via `continue` and the aggregate check at the bottom of the loop
    # would then report "no host reported keel-bench-kern-audit" — a false statement
    # about a marker every host did print, and the exact misattribution issues #32
    # and #33 are about. A criterion is placed by what it depends on, not by where
    # it reads well.
    assert_kern_audit_drift "$BENCHLOG" "$AUDITKERN" "$host" \
      ", or P4 shipped a shape this gate does not audit"

    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
    # All three declared before any is read. Peak is in the list because the
    # provenance lines below divide by it, and an absent peak used to degrade into a
    # missing info line rather than into a verdict (DESIGN.md §5 rule 6).
    require_bench "[$host] the Ssyrk/Sgemm ratio's inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "$GATE_SSYRK" "$GATE_SGEMM" "$GATE_PEAK" || continue
    # Before any criterion divides by a rate: on a host with no governor, the clock this
    # sweep ran on is established here or not at all (§5 rule 5 as amended 2026-08-16). After require_bench
    # rather than before, so an absent peak is named as the missing row it is instead of
    # reaching the series as middle=none.
    clock_post "$host" "$BENCHBIN" "$BENCHCSV" || continue

    # ---- the numerator, checked (criterion 7)
    FBAD=""; SGD=""; SYD=""
    for r in Sgemm Ssyrk; do
      case "$r" in
        Sgemm) want_name="$GATE_SGEMM" ;;
        Ssyrk) want_name="$GATE_SSYRK" ;;
      esac
      fl="$(marker_row bench-flops "$BENCHLOG" name "$want_name")"
      if [[ -z "$fl" ]]; then
        FBAD="$FBAD ${r}(no keel-bench-flops declaration: the rate has an unstated numerator)"
        continue
      fi
      got="$(field flops "$fl")"
      exp="$(flops_expect "$r" "$fl")"
      wf="$(flops_formula "$r")"
      gf="$(field formula "$fl")"
      if [[ -z "$exp" ]]; then
        FBAD="$FBAD ${r}(declares no m/n/k, so its flop count cannot be recomputed)"
      elif [[ "$got" != "$exp" ]]; then
        FBAD="$FBAD ${r}(declares flops=$got, this gate computes $exp from n=$(field n "$fl") k=$(field k "$fl"))"
      elif [[ "$gf" != "$wf" ]]; then
        FBAD="$FBAD ${r}(declares formula=$gf, this gate's is $wf)"
      fi
      if [[ "$r" == Sgemm ]]; then SGD="$fl"; else SYD="$fl"; fi
    done
    if [[ -n "$FBAD" ]]; then
      fail "[$host] the ratio's flop counts do not check out:$FBAD — Ssyrk does about half of Sgemm's arithmetic, so a wrong count moves this ratio by 2x and the bar would be cleared by a routine that did not clear it"
      continue
    fi
    # "at same size", checked rather than assumed: two rates measured at two sizes
    # are not a ratio (DESIGN.md §7 rule 7).
    gm="$(field m "$SGD")"; gn="$(field n "$SGD")"; gk="$(field k "$SGD")"
    sn="$(field n "$SYD")"; sk="$(field k "$SYD")"
    if [[ "$gm" != "$gn" || "$gn" != "$gk" || "$sn" != "$gn" || "$sk" != "$gk" ]]; then
      fail "[$host] the two benchmarks are not at the same size: Sgemm m=$gm n=$gn k=$gk against Ssyrk n=$sn k=$sk"
      continue
    fi
    info "[$host] flop counts checked: Sgemm $(field flops "$SGD") = 2*m*n*k, Ssyrk $(field flops "$SYD") = k*n*(n+1), both at n=$gn k=$gk"
    info "[$host] Ssyrk $(bench_describe "$GATE_SSYRK" "$BENCHCSV" GFLOP/s), Sgemm $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s), peak $(bench_describe "$GATE_PEAK" "$BENCHCSV" GFLOP/s), one invocation"
    for r in "$GATE_SSYRK" "$GATE_SGEMM"; do
      rp="$(bench_ratio "$r" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      rpl="$(bench_ratio_lo "$r" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      [[ -n "$rp" ]] && info "[$host] ${r%%/*} = $(awk -v v="$rp" 'BEGIN{printf "%.1f", v*100}')% of measured peak ($(awk -v v="${rpl:-0}" 'BEGIN{printf "%.1f", v*100}')% net of CI) — reported, not a P4 criterion"
    done

    slo="$(bench_ratio_lo "$GATE_SSYRK" "$GATE_SGEMM" "$BENCHCSV" GFLOP/s)"
    shi="$(bench_ratio_hi "$GATE_SSYRK" "$GATE_SGEMM" "$BENCHCSV" GFLOP/s)"
    spt="$(bench_ratio "$GATE_SSYRK" "$GATE_SGEMM" "$BENCHCSV" GFLOP/s)"
    grade="$(bench_ratio_grade "$GATE_SSYRK" "$GATE_SGEMM" "$BENCHCSV" GFLOP/s "$SYRK_FLOOR")"
    if [[ -z "$grade" || "$grade" == unbounded ]]; then
      unmeasured "[$host] no bounded Ssyrk/Sgemm ratio: benchstat established no confidence interval, which is a failure to measure rather than a pass"
      continue
    fi
    SYRK_MEASURED=$((SYRK_MEASURED + 1))
    slopc="$(awk -v v="$slo" 'BEGIN{printf "%.1f", v*100}')"
    shipc="$(awk -v v="$shi" 'BEGIN{printf "%.1f", v*100}')"
    sptpc="$(awk -v v="$spt" 'BEGIN{printf "%.1f", v*100}')"
    # The flip-headroom diagnostic, printed on every host on every run whatever the
    # verdict: the symmetric CI at which this host's bound would land exactly on the
    # bar. A green with 1.16 points of allowance against a host that produces 3.0%
    # intervals is a green that turned on the weather, and until #67 nothing in the
    # log said so — the reader had to solve raw·(1−a)/(1+a) = bar themselves.
    hr="$(bench_ratio_headroom "$spt" "$SYRK_FLOOR")"
    sci="$(bench_stat "$GATE_SSYRK" "$BENCHCSV" GFLOP/s | awk '{ if ($2 == "inf") print "unbounded"; else printf "%.1f%%", $2*100 }')"
    gci="$(bench_stat "$GATE_SGEMM" "$BENCHCSV" GFLOP/s | awk '{ if ($2 == "inf") print "unbounded"; else printf "%.1f%%", $2*100 }')"
    info "[$host] criterion 7 interval [${slopc}%, ${shipc}%] about a raw ${sptpc}%, bar $(awk -v f="$SYRK_FLOOR" 'BEGIN{printf "%.1f", f*100}')%; observed CI Ssyrk +/- ${sci}, Sgemm +/- ${gci}; flip-headroom $(awk -v v="$hr" 'BEGIN{printf "%+.2f", v*100}')% symmetric CI"
    # T21: benchstat's CI is an integer percent, so "+/- 0.0%" means "under 0.5%",
    # not "zero". If the headroom is smaller than that rounding, a PASS computed
    # from a 0.0% arm is inside the formatting's own uncertainty and the interval
    # printed above is narrower than the measurement supports. Say so rather than
    # let the reader assume the bound is exact.
    if [[ "$sci" == "0.0%" || "$gci" == "0.0%" ]] &&
       awk -v v="$hr" 'BEGIN{exit !(v < 0.005)}'; then
      info "  CAUTION: flip-headroom is under 0.5% and at least one arm's CI printed as 0.0%, which T21 says only bounds it below 0.5% — this verdict lies inside benchstat's rounding, and settling it needs a higher -count on this host, not a re-read of this line"
    fi
    case "$grade" in
      pass)
        pass "[$host] Ssyrk holds ${sptpc}% of Sgemm's rate at n=$gn, ${slopc}% net of CI (>= 85%)"
        SYRK_CLEARED=$((SYRK_CLEARED + 1))
        ;;
      fail)
        fail "[$host] Ssyrk holds only ${sptpc}% of Sgemm's rate at n=$gn, and the whole interval [${slopc}%, ${shipc}%] is below 85%: the triangular derivation is losing more than its masked half"
        SYRK_MISSED=$((SYRK_MISSED + 1))
        ;;
      *)
        unmeasured "[$host] Ssyrk's interval [${slopc}%, ${shipc}%] straddles the 85% bar (raw ${sptpc}%), so this reading cannot decide the criterion in either direction — it is not evidence that Ssyrk is too slow, and it is not a pass"
        info "  remedy, in order: DESIGN.md §4's one immediate re-run with both outputs archived; and if this host is CHRONICALLY indeterminate here, raise KEEL_BENCH_COUNT for this criterion on this host. The bar does not move and the raw ratio is not graded in place of the bound — a true-below ratio would clear on a lucky draw (#67)."
        SYRK_INDET=$((SYRK_INDET + 1))
        ;;
    esac
  done <<<"$HOSTS"
  [[ -n "$DRIFT_CHECKED" ]] || unmeasured "no host reported keel-bench-kern-audit, so the registry's recorded insns/FMA are unchecked against the object code — unmeasured, not drifted"
  # The aggregate inherits the three states, in the order that keeps a real miss
  # from hiding behind a noisy host: one host below the bar is a red whatever the
  # others did, and only when nothing is below the bar does indeterminacy become
  # the reason the criterion did not resolve.
  #
  # #90's clause is appended where a count can silently omit a host: SYRK_MEASURED is
  # incremented only after three `continue` paths, so CLEARED+MISSED+INDET can fall short
  # of NHOSTS and these lines would read fleet-wide over a proper subset. gate-p2's 5b and
  # gate-p3's criterion 6 append the same clause; this is the aggregate that never got it.
  if [[ "$SYRK_MEASURED" -eq 0 ]]; then
    unmeasured "no host produced a bounded Ssyrk/Sgemm ratio at all, so criterion 7 is unmeasured rather than missed"
  elif [[ "$SYRK_MISSED" -gt 0 ]]; then
    fail "$SYRK_MISSED of $NHOSTS gate hosts are below the bar with the whole interval ($SYRK_CLEARED cleared, $SYRK_INDET undecidable); the criterion is per host, on the host's own Sgemm$(fleet_shortfall "$NHOSTS" "$SYRK_MEASURED")"
  elif [[ "$SYRK_INDET" -gt 0 ]]; then
    unmeasured "$SYRK_INDET of $NHOSTS gate hosts produced an interval straddling the bar and none produced one below it ($SYRK_CLEARED cleared): criterion 7 is undecided on this run, which is not the same as missed, and the gate stays not-green until a re-run or a higher -count settles it (#67)$(fleet_shortfall "$NHOSTS" "$SYRK_MEASURED")"
  elif [[ "$SYRK_CLEARED" -eq "$NHOSTS" ]]; then
    pass "every gate host cleared 85% of its own Sgemm ($SYRK_CLEARED/$NHOSTS)$(fleet_shortfall "$NHOSTS" "$SYRK_MEASURED")"
  else
    unmeasured "$SYRK_CLEARED of $NHOSTS gate hosts cleared the bar and the rest produced no verdict at all, so the host count and the verdict count disagree: criterion 7 covered fewer hosts than this gate believes it has"
  fi
fi

# ------------------------------------------- P3's gate, carried forward whole
echo
echo "-- carried from P3 (criterion 8): the gate P4 edits the code of --"
info "the Ssyrk/Sgemm ratio above is a ratio against a number P4 can move, so the"
info "denominator's own bar is carried by running the gate that owns it — not by"
info "restating a threshold in a second place"
if [[ "$TREE_CLEAN" -eq 0 ]]; then
  unmeasured "the delegated P3 gate did not run: this gate refused a dirty tree above, and a gate that cannot run is unmeasured rather than green"
else
  mkdir -p "$(dirname "$P3LOG")"
  P3RC=0
  bash scripts/gate-p3.sh >"$P3LOG" 2>&1 || P3RC=$?
  info "full output: $P3LOG ($(grep -c '' "$P3LOG" || true) lines) — paste it verbatim into the umbrella issue beside this gate's own"
  # Count the delegated gate's own verdict lines, anchored, not every line
  # containing the word (#71): a bare `grep -c FAIL` also matches any summary
  # line *inside* gate-p3's log, so a green delegate could report a FAIL it did
  # not have. Colour codes are stripped first because the anchor is at the start
  # of the line and the escape sequence sits inside the label.
  #
  # UNMEASURED is a column of its own, not folded into either neighbour. Six of
  # gate-p3's misses became UNMEASURED under #72, and a two-column tally would
  # have shown them as neither — the same disappearing act the unanchored grep
  # performed, one column over. gate-p5's tally of *this* gate has the identical
  # shape (gate-p5.sh:1143); they are two readers of one vocabulary.
  #
  # BASELINE joined the vocabulary on 2026-08-21 (#6) and both readers gained a
  # column the same day, before either could have swallowed one. It is emitted by
  # gate-p5 alone today, so neither tally can see one yet — which is exactly the
  # condition under which a missing column is invisible, and the reason to widen on
  # the day the helper lands rather than on the day a delegate first uses it.
  # REPORTED joined on 2026-08-22 (docs/rulings.md rule 19) and both readers widened
  # again the same day, on that precedent — a second green-compatible class is a
  # second way for a line to be swallowed by neither neighbour, and the two tallies
  # are still two readers of one vocabulary.
  P3_STRIP=$(sed $'s/\033\\[[0-9;]*m//g' "$P3LOG")
  P3V="$(grep -E '^gate-p3: (GREEN|RED)' "$P3LOG" | tail -1 || true)"
  info "$(printf '%s\n' "$P3_STRIP" | grep -c '^  PASS  ' || true) PASS / $(printf '%s\n' "$P3_STRIP" | grep -c '^  FAIL  ' || true) FAIL / $(printf '%s\n' "$P3_STRIP" | grep -c '^  UNMEASURED  ' || true) UNMEASURED / $(printf '%s\n' "$P3_STRIP" | grep -c '^  BASELINE  ' || true) BASELINE / $(printf '%s\n' "$P3_STRIP" | grep -c '^  REPORTED  ' || true) REPORTED, verdict: ${P3V:-none printed}"
  # Three-way on the delegate, for the same reason the criteria themselves are
  # (#76). gate-p3 exits 0 for GREEN and 1 for RED, and prints the matching line;
  # any other exit status means it died before reaching its own verdict — 255 when
  # an ssh died under it, 128+n when something signalled it. Reporting that as
  # "gate-p3 is RED" attributes to keel a red the delegate never issued, and it is
  # the delegated form of the rule DESIGN.md §5.6 states directly: a killed run is
  # unmeasured, never an exit code. The verdict LINE is checked alongside the
  # status because they are two independent witnesses of the same claim, and a
  # delegate that exits 0 having printed nothing has not certified anything.
  if [[ "$P3RC" -eq 0 && "$P3V" == *GREEN* ]]; then
    pass "gate-p3 is green on this commit ($(git rev-parse --short HEAD)), so P4's denominator is still the Sgemm P3 measured"
  elif [[ "$P3RC" -eq 1 && "$P3V" == *RED* ]]; then
    fail "gate-p3 is RED on this commit (exit $P3RC), so nothing above that divides by Sgemm means what it says"
    printf '%s\n' "$P3_STRIP" | grep -E '^  (FAIL|UNMEASURED)  ' | sed 's/^/        /' | head -20
    info "  DESIGN.md §4's one-re-run allowance applies to a failing THROUGHPUT SENTINEL reading inside that gate exactly as it does when it is run directly: one immediate re-run, both outputs archived, never for a correctness criterion"
  else
    unmeasured "gate-p3 reached no verdict on this commit (exit $P3RC, verdict line: ${P3V:-none printed}), so P4's denominator is unverified rather than red: this gate cannot report a delegate's death as the delegate's judgment"
    printf '%s\n' "$P3_STRIP" | tail -20 | sed 's/^/        /'
    info "  the delegated log is $P3LOG in full; an exit that is neither 0 nor 1 is the delegate dying, which is a defect to find rather than a threshold to re-run"
  fi
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
gate_verdict gate-p4
