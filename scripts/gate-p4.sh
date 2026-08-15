#!/usr/bin/env bash
# Gate P4 — see DESIGN.md §4/P4. Written at the START of phase P4, then made
# green. Exits 0 only when every criterion for the phase holds. A red gate
# blocks the next phase; there is no override flag on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P4):
#   "all routines green vs oracle incl. upper/lower x trans x unit-diag lattice
#    for `Strsm`; `Ssyrk` >=85% of `Sgemm` GFLOPS at same size."
#
# and the routines that "all routines" names, from the two bullets above it:
#   "`Sgemv` (both transposes), `Sger`."
#   "`Ssyrk`, `Ssymm` as blocked GEMM with triangular masking in the C-update;
#    `Strsm` as small unblocked triangular solves at the diagonal + GEMM
#    rank-updates elsewhere (the BLIS recipe ...)."
#
# How those are mechanized, and every judgement call involved:
#
#  1. THE LATTICES ARE ENFORCED, NOT TRUSTED — AND P4 IS WHERE THAT MATTERS MOST.
#     P3 had one flag pair (transA, transB); P4 has five routines carrying side,
#     uplo, trans, diag and two strides between them, and every one of those flags
#     selects a DIFFERENT CODE PATH rather than a different number. A green
#     `go test` proves only that whatever ran, passed, and "whatever ran" is the
#     entire question when the lattice has sixteen corners: Strsm with
#     side=Right/uplo=Upper/trans=Trans/diag=Unit is a different traversal of a
#     different triangle from side=Left/uplo=Lower/trans=NoTrans/diag=NonUnit, and
#     a test that quietly ran eight of the sixteen reports the same "ok".
#
#     So each routine prints one lattice marker enumerating every flag set it swept
#     and the number of combinations it ran, and this gate checks three things per
#     routine: that the required flag values are present (its own statement of the
#     lattice, written here from the BLAS definitions rather than read from the
#     test), that alpha and beta each include 0, 1 and a value that is neither —
#     0 and 1 are the special-cased paths, so a lattice of only those exercises
#     every shortcut and never the general multiply — and THAT THE ENUMERATED SETS
#     MULTIPLY OUT TO THE REPORTED COUNT. The product is taken over every key on
#     the line except `routine=` and `combos=`, which means a test that adds a
#     dimension must also grow its combination count: a marker cannot claim
#     coverage it did not run, and it cannot hide a dimension it swept only one
#     value of either.
#
#     DESIGN.md names "upper/lower x trans x unit-diag" for Strsm and this gate
#     additionally requires SIDE. Left and right are not two spellings of one
#     algorithm — one solves op(A)·X = alpha·B and the other X·op(A) = alpha·B,
#     with different traversal order and different panel shapes — so a lattice
#     without side is missing a factor of two on the routine DESIGN.md singles out
#     as the one needing a lattice at all. Adding checks a phase implies is
#     allowed; removing them is not.
#
#  2. THE ORACLE'S COST IS A DECLARED PROPERTY OF EACH SIZE, exactly as in P3 and
#     for the same reason: a float64 reference is affordable entry-by-entry at 65
#     and not at 500, so the sweep may verify a seeded random SAMPLE of entries
#     above a stated bound — but only if it says so per routine per size, with a
#     sample floor and a printed seed, in a marker this gate reads. Sizes up to 65
#     must be verified in full. The concession is written here rather than left to
#     the test because "we sampled" is the kind of thing that starts at 500 and
#     ends up applying to 17.
#
#     The size list is this gate's own and it is smaller than P3's on purpose:
#     1..17 for every remainder against any MR/NR, 31/32/33 and 63/64/65 for the
#     shipped NR and a power of two, and 500 as the one size past EVERY blocking
#     parameter (KC=384, MC=144). P3 already established the multi-block path at
#     500/1000/2048 for the routine underneath all of these; what P4 needs from a
#     large size is that its own masking and its own diagonal handling survive
#     more than one KC block, and one such size does that. 1000 and 2048 would
#     multiply the oracle cost — Strsm's reference is a triangular solve, so one
#     entry costs the whole solve — for a repetition of a fact rather than a new
#     one. Stated as a decision, not as an omission.
#
#  3. CORRECTNESS RUNS WHERE THE SHIPPED PATH RUNS, and the derived routines are
#     differential-tested across backends. The dev host is darwin/arm64 and cannot
#     execute archsimd at all (docs/toolchain-notes.md T1), so the lattices are
#     audited from the log of a host that ran them with the AVX-512 backend live.
#     Each routine also reports which backends its runners exercised, and this gate
#     requires at least the widest and the scalar reference on that host: DESIGN.md
#     §5 rule 2 asks for backend-vs-backend agreement independent of the oracle,
#     and for a routine derived from a shared microkernel that is the check which
#     distinguishes "the derivation is right" from "the derivation and the oracle
#     agree about a shape neither of them exercises".
#
#  4. P2's KERNEL PROPERTIES ARE RE-CHECKED HERE, BECAUSE P4 IS THE PHASE MOST
#     LIKELY TO ADD A SECOND KERNEL FAMILY. Triangular masking is the exact thing
#     internal/block's package doc declined to do with masked microkernels — "a
#     second family of microkernels ... doubling the kernel family doubles what has
#     to stay zero-spill" — so the audit that says the K-loop is still one loop with
#     no spills, no calls and no bounds checks runs on every gate from P2 onward and
#     does not become optional because P4's own checks are green. The registry drift
#     check comes with it: every shipped shape's recorded insns/FMA must equal what
#     the audit counts, and a shape the gate does not audit is unrankable rather
#     than lean, which is what makes "a new kernel appeared" visible here instead of
#     in a benchmark six months later.
#
#  5. THE DERIVATION IS CHECKED AS FAR AS A MARKER CAN CHECK IT, AND THE THROUGHPUT
#     RATIO IS WHAT MAKES IT MORE THAN A CLAIM. DESIGN.md's "as blocked GEMM with
#     triangular masking" and "small unblocked triangular solves at the diagonal +
#     GEMM rank-updates elsewhere" are design instructions; a gate cannot read an
#     implementation strategy out of a binary. What it can do is require the three
#     derived Level-3 routines to report the SAME microkernel and the SAME blocking
#     parameters the Sgemm in that same run dispatched to — an independent
#     reimplementation would have nothing to report, and a second kernel family
#     would report a different shape — and then, for Ssyrk, to measure the
#     consequence. A triple loop cannot reach 85% of a blocked GEMM's rate, so
#     criterion 7 is the derivation criterion with teeth and the markers are the
#     part that says which shape the teeth closed on.
#
#  6. WHAT "ALL ROUTINES GREEN VS ORACLE" INCLUDES BEYOND THE LATTICE. Every
#     routine gets the edge coverage P3's Sgemm has, because none of it is
#     Sgemm-specific: an ld wider than the matrix (how a caller passes a
#     submatrix — the case where an off-by-one writes into somebody else's data
#     rather than off the end of a slice), a zero dimension, argument panics, and
#     the non-finite rules (alpha == 0 must not read A, beta == 0 must not read C,
#     and a zero-padded panel meeting an infinity must not leak a NaN into C).
#
#     Three routine-specific ones are required on top, and each is a property no
#     oracle comparison can see, because they are about memory the routine must
#     NOT touch:
#       - Ssyrk: the untouched triangle. It updates one triangle of C and must
#         leave the other bit-identical, so the other one is poisoned and compared
#         bit-for-bit afterwards.
#       - Ssymm and Strsm: the unreferenced triangle of A. Both read one triangle
#         and must not read the other; poison the out-of-scope half with NaN and
#         the result must be unchanged, or the masking is wrong in the one
#         direction a correct-looking answer hides.
#       - Strsm: unit diagonal means the stored diagonal is NOT REFERENCED, which is
#         a documented BLAS guarantee callers rely on (a factored matrix stores L's
#         and U's diagonals in one array). Poison the diagonal, ask for diag=Unit,
#         and the answer must be the one for a unit diagonal.
#     These are named in the extras marker and required by name, so dropping one
#     is a red gate rather than a quiet regression in a test file.
#
#  7. `Ssyrk` >= 85% OF `Sgemm` GFLOPS AT SAME SIZE, AND THE NUMERATOR IS CHECKED.
#     This is the one criterion in the project where the risk is in the NUMERATOR
#     rather than the denominator. Ssyrk does about half of Sgemm's arithmetic at
#     the same n — it updates a triangle, n(n+1)/2 entries rather than n² — so a
#     harness that counted 2·n²·k useful flops instead of k·n·(n+1) would report
#     roughly TWICE Ssyrk's real rate and the 85% bar would be cleared by a routine
#     running at 43% of Sgemm. CLAUDE.md's "never a number without its denominator"
#     cuts the same way pointed at the top of the fraction, so:
#       - the harness DECLARES the flop count it used, with the dimensions it used
#         and the formula it applied, in a keel-bench-flops marker;
#       - this gate RECOMPUTES that count from n and k, from its own statement of
#         the two formulas below, and fails on disagreement;
#       - and it checks "at same size" rather than assuming it: Ssyrk's n and k
#         must equal Sgemm's m, n and k, from the same markers. Two rates measured
#         at two sizes are not a ratio (DESIGN.md §7 rule 7).
#     The declared count is the number the harness divides by, not a second
#     statement of it — see bench/bench_test.go, which reports the metric and prints
#     the declaration from one variable.
#
#     The two rates come from ONE benchmark invocation on one host, so they share a
#     frequency and a thermal state, and the ratio is taken net of both confidence
#     intervals (numerator down by its CI, denominator up by its own) — the P2
#     ruling on issue #14, and the reason bench_ratio_lo exists. Percent of measured
#     peak is printed for both as provenance and judged for neither: P4's criterion
#     is the ratio, and a percent-of-peak floor on Ssyrk would be a threshold
#     invented here rather than one DESIGN.md set.
#
#     THE VERDICT HAS THREE STATES, NOT TWO (ruled 2026-08-15 on issue #67). Net of
#     CI answers one question — "is the whole interval above the bar?" — and reading
#     its negative answer as "the ratio is below the bar" collapses two causes into
#     one FAIL, which DESIGN.md §5.6 forbids. It happened: janus read 87.6% raw with
#     ±4.0%/±3.0% intervals, bound 81.6%, FAIL; the same tree at the same commit read
#     87.0% raw at ±0.0%, bound 87.0%, PASS. The raw quantity agreed within 0.6
#     points. The FAIL recorded the weather. So:
#       - PASS when the interval sits at or above the bar. This is bench_ratio_lo >=
#         0.85, unchanged bit for bit — the third state is carved out of the old
#         FAIL, never out of the old PASS, and nothing that was below the bar gets
#         in on a lucky draw.
#       - FAIL when the whole interval sits below the bar. Now a claim about Ssyrk.
#       - UNMEASURED when the interval straddles the bar. The measurement cannot
#         decide; the gate stays not-green; the remedy is DESIGN.md §4's one
#         archived re-run, and for a host that is CHRONICALLY indeterminate here,
#         a higher KEEL_BENCH_COUNT for this criterion on that host. Never a wider
#         judgment, and never the raw ratio in place of the bound.
#     And the flip-headroom — the symmetric CI at which the bound would reach the
#     bar — prints per host per run, so the record shows how close each verdict ran
#     to undecidable instead of leaving the reader to solve for it.
#
#     Every criterion that reads a benchmark declares what it will read first
#     (require_bench, DESIGN.md §5 rule 6): an absent measurement has exactly one
#     verdict available to it, unmeasured, and never a silent pass or a red
#     attributed to something else.
#
#  8. P3's GATE IS CARRIED FORWARD BY RUNNING IT, NOT BY RESTATING IT — AND THAT IS
#     ALSO THIS GATE'S ANTI-VACUITY GUARD. "Ssyrk >= 85% of Sgemm" is a ratio
#     against a number P4 can move: the derived routines are meant to share
#     internal/block's loop nest, and sharing it means editing it. A masked C-update
#     path, a callback per tile, one more branch in the macro loop — any of those can
#     cost Sgemm throughput, and every point Sgemm loses makes P4's own criterion
#     EASIER. A gate whose bar falls when the code gets worse is not a bar.
#
#     The only ratified statement of "the blocked Sgemm is still fast enough" is
#     P3's own criterion: >= 60% of min(same-host OpenBLAS, roofline x measured
#     peak), on every gate host, with the coretype sweep that chooses the reference
#     and the classifier that chooses the denominator. Restating any of that here
#     would duplicate two hundred lines and, worse, would create a second place
#     where the threshold lives. So this gate RUNS scripts/gate-p3.sh and requires
#     it green. That carries the whole P3 contract rather than the one criterion —
#     the oracle sweep, the shape-agreement checks, P2's floor on the dispatched
#     microkernel — which is what "never begin phase N+1 with a red gate" means once
#     phase N+1 starts editing phase N's code.
#
#     Consequences, all deliberate:
#       - it is the expensive check, so it runs LAST: P4's own failures surface in
#         minutes instead of after a full P3 run.
#       - gate-p3 requires a clean working tree (its criterion 6 measures
#         `git archive HEAD`). So this gate refuses a dirty tree UP FRONT and does
#         not run the delegated gate at all in that case, rather than spending the
#         whole run to arrive at a failure that was knowable at the start. Two
#         failures are reported, the dirty tree and the P3 gate not having run,
#         because those are different facts.
#       - the delegated gate's full output is written to a file and its path is
#         printed, because CLAUDE.md requires gate output verbatim in the umbrella
#         issue and a summary line is not that.
#       - DESIGN.md §4's one-re-run allowance for a failing THROUGHPUT SENTINEL
#         reading applies to a sentinel criterion inside the delegated gate exactly
#         as it does when gate-p3 is run directly: one immediate re-run, both
#         outputs archived, never for a correctness criterion. It is the operator's
#         allowance and this script does not implement it.
#
#  9. WHAT THIS GATE DOES NOT CHECK. There is no percent-of-peak floor and no
#     OpenBLAS comparison for Ssymm, Strsm, Sgemv or Sger. DESIGN.md sets a
#     throughput criterion for exactly one P4 routine and holds the others to
#     correctness, so inventing bars for them here would be this gate writing
#     criteria the design document did not set — the mirror image of weakening one.
#     Their rates are worth measuring and P5 is where a scaling story gets told;
#     what P4 owes is that they are right, that they are derived rather than
#     rewritten, and that the routine DESIGN.md did put a number on holds it.
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
# `unmeasured` is not defined here. It was, and #72 lifted it to scripts/remote.sh
# once 21 further sites across three gates turned out to need it: three copies of a
# verdict primitive is how the delegated tally came to count two columns where the
# log had three. Same effect on this gate's verdict as `fail` — DESIGN.md §5.6, an
# unmeasured criterion may not resolve as either colour, and this gate's colour is
# binary — with a label that distinguishes "Ssyrk is too slow" from "this reading
# cannot decide".

# require_bench LABEL LOG CSV UNIT NAME... — declare what a criterion is about to
# read, and give absence exactly one verdict. Same helper, same wording and the
# same reason as gate-p3.sh: see bench_expect in scripts/bench.sh for why empty is
# not a readable value, and DESIGN.md §5 rule 6 for why an unmeasured criterion may
# not resolve as either colour.
require_bench() {
  local label miss
  label="$1"; shift
  miss="$(bench_expect "$@")" && return 0
  unmeasured "$label $miss — a criterion cannot be resolved in either direction until every benchmark it reads has its rows, so this is neither a pass nor a miss"
  return 1
}

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
P3LOG="build/gate-p3-under-p4.log"

# marker NAME FILE — the value of the last `keel-NAME:` line in FILE. Test output
# arrives through t.Logf, so the marker may be indented and prefixed.
marker() { sed -n "s/.*keel-$1: *//p" "$2" | tail -1; }

# marker_all NAME FILE — every `keel-NAME:` value, one per line, for the markers
# emitted once per routine or once per (routine, size).
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

# p4_line NAME FILE ROUTINE — the keel-NAME line belonging to one routine. The P4
# markers are emitted once per routine, so `marker`'s last-wins reading would
# silently audit Strsm's coverage as if it were Sgemv's.
p4_line() {
  marker_all "$1" "$2" | awk -v want="routine=$3" '{
    for (i = 1; i <= NF; i++) if ($i == want) { print; exit }
  }'
}

# p4_verify_line FILE ROUTINE SIZE — the verification-mode line for one
# (routine, size) pair, of which there is one per size per routine.
p4_verify_line() {
  marker_all p4-verify "$1" | awk -v r="routine=$2" -v s="size=$3" '{
    rok = 0; sok = 0
    for (i = 1; i <= NF; i++) { if ($i == r) rok = 1; if ($i == s) sok = 1 }
    if (rok && sok) { print; exit }
  }'
}

# set_has SET MEMBER — comma-separated membership.
set_has() { [[ ",$1," == *",$2,"* ]]; }

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

# audit_ipf FUNC FILE — that function's audited instructions per FMA, from the
# audit's own integer counts. Carried from gate-p2.sh/gate-p3.sh for the same
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

# audit_ipf_tile TILE FILE — the audited insns/FMA for a shape named as it appears
# in a marker ("4x32"), via the convention that internal/vec's loop body for that
# shape is KernelTILE.
audit_ipf_tile() {
  [[ -n "$1" ]] || return 0
  audit_ipf "Kernel$1" "$2"
}

# flops_expect ROUTINE LINE — this gate's own count of ROUTINE's USEFUL flops at
# the dimensions the marker declares (criterion 7).
#
# Sgemm: 2·m·n·k — one multiply and one add per (i, j, p).
# Ssyrk: k·n·(n+1) — the same per (i, j, p), over the n(n+1)/2 entries of one
#        triangle including the diagonal, i.e. about half of Sgemm at the same n.
#
# USEFUL, not executed: a masked C-update computes full MR×NR tiles on the diagonal
# and keeps half of each. Counting what the routine delivers rather than what it
# performed is the convention that makes the 85% bar mean something — the wasted
# half is precisely the cost the bar is measuring, and counting it as work would
# hide it.
flops_expect() {
  local m n k
  m="$(field m "$2")"; n="$(field n "$2")"; k="$(field k "$2")"
  case "$1" in
    Sgemm) awk -v m="$m" -v n="$n" -v k="$k" 'BEGIN { if (m == "" || n == "" || k == "") exit; printf "%.0f", 2 * m * n * k }' ;;
    Ssyrk) awk -v n="$n" -v k="$k" 'BEGIN { if (n == "" || k == "") exit; printf "%.0f", k * n * (n + 1) }' ;;
  esac
}

# flops_formula ROUTINE — the formula string the harness must say it applied. The
# arithmetic is checked above; this checks the STATEMENT, so a harness that changed
# its reasoning and arrived at the same number at one size still has to say so.
flops_formula() {
  case "$1" in
    Sgemm) printf '2*m*n*k' ;;
    Ssyrk) printf 'k*n*(n+1)' ;;
  esac
}

echo "== gate-p4: Level 2 + derived Level 3 =="
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

LOG="$(mktemp)"
BINDIR="$(mktemp -d)"
BIN="$BINDIR/keel.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
SWEEPLOG="$BINDIR/sweep-avx512.log"
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"
trap 'rm -rf "$LOG" "$BINDIR"' EXIT

# --------------------------------------------- the routines vs the float64 oracle
echo
echo "-- Level 2 and derived Level 3 vs the float64 oracle --"
info "the local run exercises the scalar path only (darwin/arm64 has no archsimd);"
info "every lattice below is audited from a host that ran it with avx512 live"

LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
if [[ "$LOCAL_OK" -eq 0 ]]; then
  pass "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
else
  fail "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
  sed 's/^/        /' "$LOG" | tail -40
fi
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
if [[ "$STOCK_OK" -eq 0 ]]; then
  pass "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
else
  fail "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
  sed 's/^/        /' "$LOG" | tail -40
fi

HOSTS="$(remote_hosts)"

# ---- the measurement precondition, asserted rather than assumed
#
# DESIGN.md §5 rule 5 as amended: EVERY measuring host under the performance
# governor, asserted per host at the start of the gate and again at the moment of
# measurement, never satisfied by one host on behalf of another. This gate takes its
# own measurements (criterion 7), so it makes its own assertion rather than relying
# on the delegated P3 gate's. Unreadable counts as unmet: an unverified precondition
# is not a met one, and "unknown" is what a missing cpufreq sysfs gives on a VM.
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    hgov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    if [[ "$hgov" == performance ]]; then
      pass "[$host] cpufreq governor is performance (§5 rule 5)"
    elif [[ -z "$hgov" || "$hgov" == unknown ]]; then
      fail "[$host] scaling_governor is unreadable, so §5 rule 5 cannot be verified; an unchecked precondition is not a met one"
    else
      fail "[$host] cpufreq governor is '$hgov', not performance (§5 rule 5): a ramping core produces cold readings that enter the record as measurements"
      info "  [$host] sudo cpupower frequency-set -g performance"
    fi
  done <<<"$HOSTS"
fi

AVX512_GREEN=""
if [[ -z "$HOSTS" ]]; then
  fail "P4 needs an amd64 host to execute the AVX-512 paths; none configured"
else
  if remote_build_test . "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary (root package: P4 routines vs oracle)"
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
      pass "[$host] the P4 lattices pass against the oracle"
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
    fail "no host ran the P4 lattices green with the avx512 microkernel; their extent is unauditable"
  fi
fi

# ------------------------------------------ what the lattices actually covered
echo
echo "-- lattice extent (criteria 1, 2, 3, 5 and 6: coverage is enforced, not trusted) --"
if [[ ! -s "$SWEEPLOG" ]]; then
  fail "no avx512 lattice log to audit; every routine's coverage is unverified"
else
  L3_KERN="$(marker sgemm-active "$SWEEPLOG")"
  L3_CFG="$(marker sgemm-config "$SWEEPLOG")"
  L1_ACTIVE="$(marker l1-active "$SWEEPLOG")"
  info "P3's dispatched microkernel on this host: ${L3_KERN:-<no keel-sgemm-active marker>}"
  [[ -n "$L3_KERN" ]] || fail "no keel-sgemm-active marker in the lattice log, so the derived routines' shape cannot be compared against Sgemm's (criterion 5)"

  for r in $P4_ROUTINES; do
    # ---- the flag lattice (criterion 1)
    lat="$(p4_line p4-lattice "$SWEEPLOG" "$r")"
    if [[ -z "$lat" ]]; then
      fail "$r: no keel-p4-lattice marker, so the flags it swept are unknown — coverage unknown is coverage unestablished"
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
        fail "$r: keel-p4-lattice has no combos= count, so its enumeration cannot be checked against what it ran"
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
      fail "$r: no keel-p4-sizes marker"
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
      [[ -n "$VMISS" ]] && fail "$r: no keel-p4-verify line for size(s):$VMISS"
      [[ -n "$VBAD" ]] && fail "$r: oracle verification too weak for size(s):$VBAD"
    fi

    # ---- differential across backends (criterion 3)
    bk="$(field backends "$(p4_line p4-backends "$SWEEPLOG" "$r")")"
    if [[ -z "$bk" ]]; then
      fail "$r: no keel-p4-backends marker, so the routine was compared against the oracle but not against another backend (§5 rule 2)"
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
      fail "$r: no keel-p4-extra marker"
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
      fail "$r: no keel-p4-config marker, so what it derives from is unrecorded"
      continue
    fi
    if [[ " $P4_DERIVED_L3 " == *" $r "* ]]; then
      rk="$(field kern "$cfg")"
      if [[ -z "$rk" ]]; then
        fail "$r: keel-p4-config has no kern=, so this gate cannot tell a blocked derivation from a reimplementation"
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
        fail "$r: keel-p4-config has no l1=, so what its inner loop derives from is unrecorded"
      elif [[ -n "$L1_ACTIVE" && "$rl1" != "$L1_ACTIVE" ]]; then
        fail "$r: runs Level-1 backend $rl1 where this run's dispatched backend is $L1_ACTIVE"
      else
        pass "$r: derived on the dispatched Level-1 backend ($rl1), path=$(field path "$cfg")"
      fi
    fi
  done
fi

# ------------------------------ P2's compile-time properties, carried forward
echo
echo "-- carried from P2 (criterion 4): the K-loop after triangular masking --"
info "compile-time property, audited against the linux/amd64 object code the hosts run"
if GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
     -pkg "$KERN_PKG" -func "$KERN_FUNCS" -mode spill -ssa "$SSADIR" >"$AUDITKERN" 2>&1; then
  sed 's/^/        /' "$AUDITKERN"
  pass "0 accumulator spills in the steady-state K-loop (P2 property held)"
  pass "0 calls in the steady-state K-loop (P2 property held)"
  pass "0 surviving bounds checks in the steady-state K-loop (P2 property held)"
else
  sed 's/^/        /' "$AUDITKERN"
  fail "P4 broke a P2 kernel property; the audit above says which"
fi
if go run ./internal/spill/cmd/spill-audit \
     -pkg ./internal/vec -func "$PEAK_FUNCS" -mode nomemory >"$AUDITPEAK" 2>&1; then
  pass "every peak kernel's steady-state loop is still register-only (the denominator is still a ceiling)"
else
  sed 's/^/        /' "$AUDITPEAK"
  fail "a peak kernel's loop touches memory; the percent-of-peak denominator is not a ceiling"
fi
# Every package the derived routines can live in, so a bounds check introduced by a
# triangular index expression is visible as provenance even where it is outside the
# K-loop the criterion covers.
BCE_PKGS="$KERN_PKG ./internal/kern ./internal/pack ./internal/block"
# shellcheck disable=SC2086 # BCE_PKGS is a deliberate list of package patterns
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 \
     go build -gcflags='-d=ssa/check_bce' $BCE_PKGS 2>"$LOG"; then
  BCE_N="$(grep -c 'Found Is\(Slice\)\?InBounds' "$LOG" || true)"
  info "check_bce: ${BCE_N:-0} bounds check(s) across vec+kern+pack+block, all outside the K-loop (provenance; the criterion is the loop-body audit above)"
  pass "check_bce output recorded as provenance"
else
  sed 's/^/        /' "$LOG" | tail -20
  fail "build with -d=ssa/check_bce failed"
fi

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
  fail "no execution hosts, so the Ssyrk/Sgemm ratio cannot be evaluated"
else
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary (Sgemm + Ssyrk + peak)"
  else
    fail "cross-compile of linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  DRIFT_CHECKED=""
  while read -r host; do
    [[ -n "$host" ]] || continue
    # Re-read at the moment of measurement, not merely in the preamble: a governor
    # that changed in between belongs to a machine somebody started using, and the
    # reading it produces is not one §5 rule 5 covers.
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    if [[ "$gov" != performance ]]; then
      fail "[$host] governor is '${gov:-unknown}' at measurement time, not performance: it changed after this gate's preamble checked it, so nothing measured here is covered by §5 rule 5"
      continue
    fi
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" \
         -test.bench="$P4_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      fail "[$host] the Ssyrk/Sgemm benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    # ---- criterion 4's registry drift check, from a marker a host actually
    # produced. Same binary everywhere, so it is checked once, but it is checked
    # against what the shipped library believes rather than against source.
    #
    # IT RUNS BEFORE THE ROW DECLARATION BELOW, and the order is the point. This
    # check reads a provenance marker the harness prints at startup; it does not
    # divide by any benchmark. Sitting after require_bench, a missing Ssyrk row
    # would skip it via `continue` and the aggregate check at the bottom of the loop
    # would then report "no host reported keel-bench-kern-audit" — a false statement
    # about a marker every host did print, and the exact misattribution issues #32
    # and #33 are about. A criterion is placed by what it depends on, not by where
    # it reads well.
    kaudit="$(marker bench-kern-audit "$BENCHLOG")"
    if [[ -z "$DRIFT_CHECKED" && -n "$kaudit" ]]; then
      DRIFT_CHECKED="$host"
      DRIFT_BAD=""
      for pair in $kaudit; do
        rtile="${pair%%/*}"; rval="${pair#*=}"
        raud="$(audit_ipf_tile "$rtile" "$AUDITKERN")"
        if [[ -z "$raud" ]]; then
          DRIFT_BAD="$DRIFT_BAD ${rtile}(recorded $rval, not audited by this gate)"
        elif ! awk -v a="$rval" -v b="$raud" 'BEGIN{exit !(a - b < 0.001 && b - a < 0.001)}'; then
          DRIFT_BAD="$DRIFT_BAD ${rtile}(recorded $rval, audited $(printf '%.3f' "$raud"))"
        fi
      done
      if [[ -z "$DRIFT_BAD" ]]; then
        pass "every shipped shape's recorded insns/FMA matches the audited object code ($kaudit)"
      else
        fail "the shape ranking reads stale instruction counts:$DRIFT_BAD — internal/kern's registry has drifted from the K-loop it describes, or P4 shipped a shape this gate does not audit"
      fi
    fi

    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    # All three declared before any is read. Peak is in the list because the
    # provenance lines below divide by it, and an absent peak used to degrade into a
    # missing info line rather than into a verdict (DESIGN.md §5 rule 6).
    require_bench "[$host] the Ssyrk/Sgemm ratio's inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "$GATE_SSYRK" "$GATE_SGEMM" "$GATE_PEAK" || continue

    # ---- the numerator, checked (criterion 7)
    FBAD=""; SGD=""; SYD=""
    for r in Sgemm Ssyrk; do
      case "$r" in
        Sgemm) want_name="$GATE_SGEMM" ;;
        Ssyrk) want_name="$GATE_SSYRK" ;;
      esac
      fl="$(marker_all bench-flops "$BENCHLOG" | awk -v want="name=$want_name" '{
        for (i = 1; i <= NF; i++) if ($i == want) { print; exit }
      }')"
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
  [[ -n "$DRIFT_CHECKED" ]] || fail "no host reported keel-bench-kern-audit, so the registry's recorded insns/FMA were never checked against the object code"
  # The aggregate inherits the three states, in the order that keeps a real miss
  # from hiding behind a noisy host: one host below the bar is a red whatever the
  # others did, and only when nothing is below the bar does indeterminacy become
  # the reason the criterion did not resolve.
  if [[ "$SYRK_MEASURED" -eq 0 ]]; then
    unmeasured "no host produced a bounded Ssyrk/Sgemm ratio at all, so criterion 7 is unmeasured rather than missed"
  elif [[ "$SYRK_MISSED" -gt 0 ]]; then
    fail "$SYRK_MISSED of $NHOSTS gate hosts are below the bar with the whole interval ($SYRK_CLEARED cleared, $SYRK_INDET undecidable); the criterion is per host, on the host's own Sgemm"
  elif [[ "$SYRK_INDET" -gt 0 ]]; then
    unmeasured "$SYRK_INDET of $NHOSTS gate hosts produced an interval straddling the bar and none produced one below it ($SYRK_CLEARED cleared): criterion 7 is undecided on this run, which is not the same as missed, and the gate stays not-green until a re-run or a higher -count settles it (#67)"
  elif [[ "$SYRK_CLEARED" -eq "$NHOSTS" ]]; then
    pass "every gate host cleared 85% of its own Sgemm ($SYRK_CLEARED/$NHOSTS)"
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
  # shape (gate-p5.sh:1098); they are two readers of one vocabulary.
  P3_STRIP=$(sed $'s/\033\\[[0-9;]*m//g' "$P3LOG")
  info "$(printf '%s\n' "$P3_STRIP" | grep -c '^  PASS  ' || true) PASS / $(printf '%s\n' "$P3_STRIP" | grep -c '^  FAIL  ' || true) FAIL / $(printf '%s\n' "$P3_STRIP" | grep -c '^  UNMEASURED  ' || true) UNMEASURED, verdict: $(grep -E '^gate-p3: (GREEN|RED)' "$P3LOG" | tail -1)"
  if [[ "$P3RC" -eq 0 ]]; then
    pass "gate-p3 is green on this commit ($(git rev-parse --short HEAD)), so P4's denominator is still the Sgemm P3 measured"
  else
    fail "gate-p3 is RED on this commit (exit $P3RC), so nothing above that divides by Sgemm means what it says"
    printf '%s\n' "$P3_STRIP" | grep -E '^  (FAIL|UNMEASURED)  ' | sed 's/^/        /' | head -20
    info "  DESIGN.md §4's one-re-run allowance applies to a failing THROUGHPUT SENTINEL reading inside that gate exactly as it does when it is run directly: one immediate re-run, both outputs archived, never for a correctness criterion"
  fi
fi

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p4: GREEN"
  exit 0
fi
echo "gate-p4: RED" >&2
exit 1
