#!/usr/bin/env bash
# Gate P5 — see DESIGN.md §4/P5. Written at the START of phase P5, then made
# green. Exits 0 only when every criterion for the phase holds. A red gate
# blocks the next phase; there is no override flag on purpose. P5 is the last
# phase, so here "blocks the next phase" means blocks the release.
#
# THE CRITERIA, verbatim from DESIGN.md §4/P5's gate line:
#   ">=6x single-thread throughput at 8 cores on 4096^3; race detector clean;
#    `go vet`/`golangci-lint` clean; scalar-only build green on stock toolchain.
#    Retention stays reported rather than judged: #26's target is a direction to
#    work in, not a threshold invented after the fact."
#
# and from the design bullets above it, which say what is parallelized and what
# the parallelization may not cost:
#   "Parallelize the MC (ic) loop over a bounded worker pool sized by
#    `runtime.GOMAXPROCS(0)`; shared packed-B panel per NC iteration, per-worker
#    packed-A buffers from a `sync.Pool`. No background threads, no state between
#    calls."
#   "Runtime dispatch finalized: `avx512 -> avx2 -> scalar`, overridable by env
#    `KEEL_FORCE=scalar` for testing."
#   "The package must compile and pass (scalar path) *without* `GOEXPERIMENT=simd`."
#   "Scaling benchmark, README with honest numbers, `doc.go`."
#
# and the two rulings of 2026-08-12, also recorded in §4/P5: the phase is ordered
# internally (single-thread remediation, then the parallel nest, then this gate),
# and the scaling floor binds BY PARALLELISM CLASS rather than by routine list.
#
# HOW THOSE BECOME CHECKS, and every judgement call involved:
#
#  1. THE FLOOR BINDS BY CLASS: Sgemm, Ssyrk AND Ssymm ARE ALL JUDGED AT >=6x.
#     They are one parallelism class — GEMM-shaped nests over independent tiles,
#     no cross-iteration dependence — so the same parallelization is available to
#     each by construction, and P4 already proved they share the loop nest that
#     provides it. Judging only Sgemm would leave the place a serialization bug is
#     most likely — the triangular C-update masking, which decides per tile and
#     per row and could as easily decide per worker — measured by nothing. A
#     derived routine is not a lesser routine; it is the same code with a mask.
#
#     Strsm is a SECOND CLASS and its floor is DEFERRED TO A MEASUREMENT. Its
#     diagonal solves impose a dependency chain the other three do not have, and
#     how much parallelism it has left depends on side and shape. Writing 6x on it
#     here would be inventing a threshold without a model — the move this project
#     has now refused six times — so this gate REQUIRES Strsm's scaling to be
#     measured, REQUIRES the parallelism model behind that number to be stated
#     (what fraction of the work at this shape is rank update versus diagonal
#     solve), reports both, and judges neither. STRSM_FLOOR below stays empty for
#     exactly as long as that ratification has not happened; when DESIGN.md records
#     the ratified floor it is copied into that constant and binds from that commit
#     forward. The deferral is tracked on issue #37, so it is a named debt rather
#     than a permanent exemption — and this gate fails when the measurement or the
#     model is missing, because a deferral is a promise to measure and an
#     unmeasured deferral is an exemption with better manners.
#
#     BOTH ARMS RUN WITH BOOST OFF, AND THE BOOST-ON SPEEDUP PRINTS BESIDE THE
#     VERDICT (amended by ruling on #66; DESIGN.md §4/P5). Dividing an 8-thread
#     rate by a 1-thread rate taken on an idle machine is not a ratio: one thread
#     runs in a frequency regime eight threads physically cannot enter, so the
#     denominator carries a single-core boost clock and the criterion asks the nest
#     to beat silicon boost policy before it may demonstrate scaling. The evidence
#     that this was happening is in the shape of the misses, not in their
#     inconvenience: the two hosts that missed are the two that retain the MOST of
#     their own single-thread peak (Zen 4 92%, Zen 5 59%) while Skylake-X at 35%
#     cleared everything, twice. Scaling deficits do not sort by single-thread
#     excellence; boost tables do. So this gate SETS boost off on each host for the
#     judged pass, READS THE KNOB BACK — unreadable or unmoved counts as UNMET, never
#     as satisfied, exactly as scaling_governor does — measures both arms in that one
#     regime, restores boost, and measures a SECOND pass boost-on whose wall-clock
#     speedup is printed at equal prominence as reported-never-judged. That second
#     number is what a caller experiences and no reader gets the pass without it.
#     Consequence stated rather than buried: boost-off lowers the 1-thread arm more
#     than the 8-thread arm, so ratios RISE and are NOT comparable to the three
#     boost-on runs already in the record.
#
#  2. THE TWO RATES COME FROM ONE INVOCATION, SO THE THREAD COUNT IS PART OF THE
#     BENCHMARK NAME AND NOT PART OF THE FLAGS. `go test -cpu=1,8` looks like the
#     tool for this and is a trap: it distinguishes the two runs only by the
#     "-N" GOMAXPROCS suffix on the benchmark name, and both bench_stat and
#     bench_expect STRIP that suffix (scripts/bench.sh, so that a B/s row and a
#     sec/op row of the same benchmark can be told apart by unit). benchstat would
#     therefore aggregate the one-thread and the eight-thread samples into a single
#     row, and this gate would divide a mixture by itself and read 1.0x. So the
#     harness names the thread count (`Scale/Sgemm/n=4096/threads=8`) and sets
#     GOMAXPROCS itself, and the two rates are two rows of one CSV.
#
#     One invocation is also what makes the ratio a ratio: same binary, same
#     frequency, same thermal state, same page cache. Two invocations would have to
#     go through bench_gflops_lo, whose own doc comment forbids precisely this use.
#
#  3. THE PARALLELISM IS CHECKED, NOT ASSUMED. A row named threads=8 that silently
#     ran on one worker yields 1.0x and reads as a performance problem; a row named
#     threads=1 that ran on eight yields 1.0x and reads as the same thing. Both are
#     measurement failures wearing a performance failure's clothes. So the harness
#     declares, per row, the GOMAXPROCS it set and the number of workers the library
#     actually used, and this gate requires both to equal the thread count in the
#     name. "A bounded worker pool sized by runtime.GOMAXPROCS(0)" is the design
#     instruction; this is the part of it a gate can read.
#
#  4. THE HOST MUST HAVE THE CORES THE CRITERION NAMES, AND THE TOPOLOGY IS PART OF
#     THE RECORD. "At 8 cores" is not "at GOMAXPROCS=8 on whatever that lands on":
#     eight goroutines on four physical cores and their SMT siblings cannot reach 6x,
#     and would fail this gate for a reason that is not keel's. So the gate requires
#     >= 8 CPUs, prints threads-per-core and cores-per-socket per host, and pins
#     NOTHING — placement is the scheduler's, and issue #15 (vesta's bimodal Sdot,
#     CCD placement) is the open pinning decision this measurement feeds. If a host
#     misses the floor with sibling placement as the visible cause, that is a finding
#     to write up and take to a ruling, not a bar to lower here.
#
#  5. PARALLELISM IS A CORRECTNESS QUESTION BEFORE IT IS A THROUGHPUT ONE, AND THE
#     STANDARD IS BITWISE. Partitioning the MC loop splits C by row panels; it does
#     not reassociate any single output element's reduction. So a correct parallel
#     nest returns BIT-IDENTICAL results to the serial one at every thread count,
#     and the tests must say bitwise rather than fall back on a tolerance. That is
#     stronger than the oracle sweep can be, and it is also a design tripwire: if
#     some future implementation parallelizes the K loop, bitwise identity breaks,
#     and the correct response is a documented ruling on non-determinism, not a
#     widened epsilon. The declared thread counts must include 1, 8 and a value that
#     divides neither the block count nor the core count evenly (3), because an
#     off-by-one in a row partition hides perfectly at 1, 2, 4 and 8.
#
#  6. "NO BACKGROUND THREADS, NO STATE BETWEEN CALLS" IS CHECKABLE, SO IT IS
#     CHECKED. After a call returns, the goroutine count must be back at its
#     pre-call baseline — that is what "no background threads" means operationally,
#     and it is the difference between a pool that parks workers forever and one
#     that ends with the call. And a second identical call must produce a
#     bit-identical result, which is what catches a sync.Pool buffer that a later
#     call reads before writing. Both are named in a marker and required by name,
#     because both cost nothing until a caller runs keel inside something that
#     counts goroutines or calls it twice.
#
#  7. THE DISPATCH CHAINS ARE EXERCISED BY THIS GATE, NOT ONLY REPORTED BY THE
#     TESTS — AND THERE ARE TWO OF THEM. Level 1 dispatches avx512 -> avx2 ->
#     scalar; Level 3 dispatches avx512 -> scalar (ruled 2026-08-12, #40:
#     internal/kern has no AVX2 microkernel, no host here is AVX2-only silicon, and
#     KEEL_FORCE=avx2 on an AVX-512 part is evidence about correctness, not about
#     performance — so a third Level-3 rung would be advertised with nothing able
#     to back it). The tests declare both chains and that an unrecognized
#     KEEL_FORCE refuses to run; the gate additionally re-runs the binary with
#     KEEL_FORCE=scalar, avx2 and avx512 and requires the library's own
#     active-backend markers to name what was asked for, then runs it with a
#     nonsense value and requires a non-zero exit. A dispatch chain that is only
#     self-reported is a chain nobody pulled on.
#
#     The narrowing is enforced in the direction that costs something. For a
#     Level-1-only rung the gate requires the microkernel to come back *scalar*
#     and the library to say so: an avx2 microkernel appearing at Level 3 fails
#     here, because the ruling narrowed what is claimed and a claim that grows
#     back silently is what this gate exists to catch. The debt keeps its trigger
#     on #40 — an AVX2-native evidentiary host — so P5_KERN_CHAIN grows a rung by
#     a measurement, not by a session's convenience.
#
#     Marker contract, since the chains are now per level:
#       keel-p5-dispatch: l1=avx512,avx2,scalar kern=avx512,scalar
#     A single `chain=` field cannot state the Level-3 narrowing at all, and a
#     ruling that cannot be stated is one the next session re-litigates.
#
#  8. RETENTION IS PRINTED AND NOT JUDGED. DESIGN.md §4/P5 says so in the gate line
#     itself: issue #26 is a direction to work in. It is printed for the same reason
#     gate-p3 printed it — so P5's remediation stage answers to a re-runnable number
#     rather than a remembered one. Percent of (8 x the single-thread peak) is
#     printed on the same terms, and with its own caveat: that denominator assumes a
#     clock that does not drop with core count, which is not true of these parts, so
#     it is an over-estimate of the ceiling and is labelled as one.
#
#  9. THE README'S NUMBERS ARE RE-MEASURED BY THE GATE THAT SHIPS THEM. "README with
#     honest numbers" is the one design instruction here that could mean nothing, so
#     it is mechanized: the README carries a delimited table whose rows name a CPU
#     MODEL (never a hostname — .keel-hosts is gitignored infrastructure and real
#     hostnames are not source), a benchmark, a thread count, a rate, and the
#     DENOMINATOR that rate is a fraction of. This gate parses those rows, matches
#     each to the host whose CPU model it names, and fails if a published number
#     disagrees with this run by more than README_TOL. Any GFLOP/s figure OUTSIDE
#     that block fails too: a number a reader cannot trace to a machine and a
#     denominator is what §7 rule 7 forbids, and a number this gate cannot
#     re-measure is a claim rather than a measurement. This is the rule that stops
#     the published table drifting from the code one optimistic edit at a time.
#
# 10. P4's GATE IS CARRIED FORWARD BY RUNNING IT, WHICH CARRIES P3's AND P2's — AND
#     THAT IS THIS GATE'S ANTI-VACUITY GUARD. ">=6x single-thread throughput" is a
#     ratio whose DENOMINATOR this phase is chartered to change. Stage 1 makes the
#     serial path faster (#26, #36, #37, and the deferred measurements on #21/#22),
#     which makes this bar HARDER — good — but a parallel nest that accidentally
#     slowed the serial path would make it EASIER, and a bar that falls when the
#     code gets worse is not a bar. The absolute statements belong to P4 (Ssyrk >=
#     85% of the same host's Sgemm) and P3 (Sgemm >= 60% of same-host OpenBLAS) and
#     P2 (the dispatched microkernel's floor and spill audit), so this gate RUNS
#     scripts/gate-p4.sh, which runs scripts/gate-p3.sh. Restating any of those
#     thresholds here would create a second place where they live, and two places
#     is how a threshold gets quietly relaxed in one of them.
#
#     Consequences, all deliberate and all inherited from how gate-p4 does this:
#       - it runs LAST, because it is by far the most expensive check;
#       - the delegated chain builds `git archive HEAD`, as does the native
#         race-instrumented build below, so this gate refuses a dirty tree UP FRONT
#         rather than an hour in;
#       - the delegated output goes to a file whose path is printed, because
#         CLAUDE.md wants gate output verbatim in the umbrella issue and a summary
#         line is not that. gate-p4's log names gate-p3's in turn, so the whole
#         chain is readable from this one line;
#       - DESIGN.md §4's one-re-run allowance for a failing THROUGHPUT SENTINEL
#         reading applies inside the delegated gates exactly as when they are run
#         directly. It is the operator's allowance, this script does not implement
#         it, and it never applies to a correctness criterion.
#
# 11. WHAT THIS GATE DOES NOT CHECK, and why not. It sets no absolute GFLOP/s floor
#     at 4096 and no OpenBLAS comparison at 4096: DESIGN.md's P5 criterion is a
#     scaling ratio, the absolute bars live at 2048 in the delegated gates, and
#     inventing a second absolute bar at a second size would be this gate writing
#     criteria the design document did not set. It does not judge multi-threaded
#     OpenBLAS either — that comparison needs a stated threading model per library
#     before it means anything, and nobody has ratified one. And it does not check
#     release mechanics (a v0.1.0 CHANGELOG section, a tag): those belong to the
#     release, not to the phase gate.
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

# require_bench LABEL LOG CSV UNIT NAME... — declare what a criterion is about to
# read, and give absence exactly one verdict. Same helper, same wording and the
# same reason as gate-p3.sh and gate-p4.sh: see bench_expect in scripts/bench.sh,
# and DESIGN.md §5 rule 6 for why an unmeasured criterion may not resolve as
# either colour.
require_bench() {
  local label miss
  label="$1"; shift
  miss="$(bench_expect "$@")" && return 0
  unmeasured "$label $miss — a criterion cannot be resolved in either direction until every benchmark it reads has its rows, so this is neither a pass nor a miss"
  return 1
}

# ------------------------------------------------------------ P5's own bars
# The shape and the thread count DESIGN.md §4/P5 names, and the floor it sets.
P5_SIZE=4096
P5_THREADS=8
SCALE_FLOOR=6.0

# The two parallelism classes (criterion 1, ruled 2026-08-12). P5_JUDGED is one
# class: GEMM-shaped nests over independent tiles. P5_MEASURED is the other:
# Strsm, whose diagonal solves carry a dependency chain the others do not.
P5_JUDGED="Sgemm Ssyrk Ssymm"
P5_MEASURED="Strsm"
# Empty until DESIGN.md §4/P5 records a ratified floor for Strsm, derived from the
# measurement and the model this gate requires below. Fill it in only by way of
# that ratification: a number typed here without a model in DESIGN.md is exactly
# the invented threshold the deferral exists to avoid.
STRSM_FLOOR=""

# Benchmark row names. The thread count is IN THE NAME (criterion 2).
scale_name() { printf 'Scale/%s/n=%d/threads=%d' "$1" "$P5_SIZE" "$2"; }
GATE_PEAK="Peak/avx512"
# Two top-level alternatives, each with fewer elements than the names it selects,
# so each is depth-unconstrained and runs everything beneath it. That is the one
# reading of `go test -bench`'s two-level split which means what it looks like
# (issue #32, docs/toolchain-notes.md T15): the split on '|' happens FIRST, and
# only then does each alternative split on '/'. require_bench declares the exact
# rows this gate reads, so anything extra beneath these two costs time and nothing
# else.
P5_BENCH_FILTER='Scale|Peak'

# The thread counts the determinism test must cover; 3 is there because a
# row-partition off-by-one hides at every power of two (criterion 5).
P5_DET_THREADS="1 3 8"
# The properties the no-state marker must name (criterion 6).
P5_NOSTATE_REQ="goroutines-return-to-baseline repeat-call-bit-identical"
# The dispatch chains, in order, and the backends this gate itself forces
# (criterion 7).
#
# TWO CHAINS, NOT ONE, by ruling (2026-08-12, #40 — DESIGN.md §4/P5). Level 1
# dispatches all three rungs; Level 3 dispatches two, because internal/kern has no
# AVX2 microkernel and no host here is AVX2-only silicon, so a three-rung Level-3
# claim would advertise a link no gate can back. This gate is where that ruling is
# enforced, and it is enforced in the direction that costs something: the check
# below does not merely stop failing on avx2 at Level 3, it *requires* that
# forcing avx2 lands on a scalar microkernel and that KERN_CHAIN never names avx2.
# The narrowing is a criterion change, so it is written where the criteria are, with
# the ruling's date and number beside it; the day an AVX2 microkernel is measured on
# AVX2-native silicon, KERN_CHAIN grows a rung and this comment is what says why.
P5_L1_CHAIN="avx512,avx2,scalar"
P5_KERN_CHAIN="avx512,scalar"
P5_FORCED="scalar avx2 avx512"
# Backends that are Level 1 only: forcing one must produce that L1 backend and a
# *scalar* microkernel, with the library saying so.
P5_L1_ONLY="avx2"

# How far a published README number may sit from this run's measurement before it
# stops being the same claim (criterion 9). 5% is wider than any CI this project
# has recorded at 4096 and narrower than any optimistic rounding.
README_TOL=0.05
README_BEGIN='<!-- keel-numbers: begin -->'
README_END='<!-- keel-numbers: end -->'

# The delegated gate's full output. build/ is gitignored; the path is printed
# because CLAUDE.md wants gate output verbatim in the umbrella issue.
P4LOG="build/gate-p4-under-p5.log"

# Where a native, race-instrumented build lands on a remote host. Deliberately not
# gate-p3's OpenBLAS tree, so a P5 run cannot overwrite a P3 run's working copy.
P5_REMOTE_SRC="${P5_REMOTE_SRC:-/tmp/keel-p5-src}"

# ----------------------------------------------------------------- helpers
# marker NAME FILE — the value of the last `keel-NAME:` line in FILE. Test output
# arrives through t.Logf, so a marker may be indented and prefixed.
marker() { sed -n "s/.*keel-$1: *//p" "$2" | tail -1; }

# marker_all NAME FILE — every `keel-NAME:` value, one per line.
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

# p5_line NAME FILE ROUTINE — the keel-NAME line belonging to one routine. The P5
# markers are emitted once per routine, so `marker`'s last-wins reading would
# audit Strsm's parallel behaviour and call it Sgemm's.
p5_line() {
  marker_all "$1" "$2" | awk -v want="routine=$3" '{
    for (i = 1; i <= NF; i++) if ($i == want) { print; exit }
  }'
}

# bench_line NAME FILE BENCHNAME — the keel-bench-* line for one benchmark row.
bench_line() {
  marker_all "$1" "$2" | awk -v want="name=$3" '{
    for (i = 1; i <= NF; i++) if ($i == want) { print; exit }
  }'
}

# set_has SET MEMBER — comma-separated membership.
set_has() { [[ ",$1," == *",$2,"* ]]; }

# flops_expect ROUTINE LINE — this gate's own count of ROUTINE's USEFUL flops at
# the dimensions the marker declares, recomputed rather than trusted (the P4
# doctrine, carried forward because P5 puts two more routines into a ratio and the
# numerator is still where a flattering error would live):
#
#   Sgemm: 2*m*n*k        one multiply and one add per (i, j, p).
#   Ssyrk: k*n*(n+1)      the same, over one triangle including its diagonal.
#   Ssymm: 2*m*n*k        A is symmetric and k x k, but every entry of C still gets
#                         a full k-deep dot product: symm's saving is memory, never
#                         arithmetic, so its count is GEMM's.
#   Strsm: n*m*(m+1)      one multiply-add per (row, column) pair of one triangle
#                         including the diagonal, per right-hand side.
#
# USEFUL, not executed: a masked diagonal tile is computed whole and half of it
# discarded, and counting the discarded half as work would hide the very cost
# these ratios exist to expose.
flops_expect() {
  local m n k
  m="$(field m "$2")"; n="$(field n "$2")"; k="$(field k "$2")"
  case "$1" in
    Sgemm|Ssymm) awk -v m="$m" -v n="$n" -v k="$k" 'BEGIN { if (m == "" || n == "" || k == "") exit; printf "%.0f", 2 * m * n * k }' ;;
    Ssyrk)       awk -v n="$n" -v k="$k" 'BEGIN { if (n == "" || k == "") exit; printf "%.0f", k * n * (n + 1) }' ;;
    Strsm)       awk -v m="$m" -v n="$n" 'BEGIN { if (m == "" || n == "") exit; printf "%.0f", n * m * (m + 1) }' ;;
  esac
}

# flops_formula ROUTINE — the formula string the harness must say it applied. The
# arithmetic is checked above; this checks the STATEMENT, so a harness that changed
# its reasoning and happened to land on the same number at this one shape still has
# to say what it now believes.
flops_formula() {
  case "$1" in
    Sgemm|Ssymm) printf '2*m*n*k' ;;
    Ssyrk)       printf 'k*n*(n+1)' ;;
    Strsm)       printf 'n*m*(m+1)' ;;
  esac
}

echo "== gate-p5: parallelism, dispatch, polish =="
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
echo "-- builds, vet and lint --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "build (GOEXPERIMENT=simd)"; else fail "build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "build (stock toolchain, scalar path, no experiment)"; else fail "build (stock toolchain, scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

LOG="$(mktemp)"
BINDIR="$(mktemp -d)"
BIN="$BINDIR/keel.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
# The boost-on pass (#66): reported beside the verdict, and the regime the README's
# published wall-clock rates were taken in, so it is what re-measures them.
BENCHLOG_ON="$BINDIR/bench-boost-on.log"
BENCHCSV_ON="$BINDIR/bench-boost-on.csv"
SWEEPLOG="$BINDIR/sweep-avx512.log"
: >"$SWEEPLOG"

# Hosts whose boost knob this run has written, so the EXIT path can put them back.
# A gate that dies inside its measurement window must not leave a machine
# de-boosted: every later measurement on it — including this gate's own delegated
# gate-p4/p3/p2 runs, which are boost-on measurements — would silently shift regime
# and nothing downstream would know to ask. INT and TERM are trapped alongside EXIT
# because bash does not run an EXIT trap on an untrapped fatal signal, and being
# reaped is the normal way a long run ends here.
BOOST_TOUCHED=""
gate_cleanup() {
  local h
  for h in $BOOST_TOUCHED; do
    remote_boost_set "$h" on >/dev/null 2>&1 || true
  done
  rm -rf "$LOG" "$BINDIR"
}
trap gate_cleanup EXIT INT TERM

# golangci-lint is named by the criterion, so its ABSENCE is unmeasured rather
# than clean — the same rule a missing OpenBLAS gets in gate-p3. A check that did
# not run has no colour, and the operator gets the exact install command instead of
# a green with a hole in it.
if ! command -v golangci-lint >/dev/null 2>&1; then
  unmeasured "golangci-lint is not installed, so the criterion that names it is unmeasured rather than clean"
  info "  brew install golangci-lint"
elif GOEXPERIMENT=simd golangci-lint run ./... >"$LOG" 2>&1; then
  pass "golangci-lint clean ($(golangci-lint version 2>&1 | head -1))"
else
  fail "golangci-lint reports findings"
  sed 's/^/        /' "$LOG" | tail -40
fi

# The delegated chain measures `git archive HEAD`, and so does the native
# race-instrumented build below. Checked here, before anything expensive, because
# reaching a knowable failure after an hour of benchmarks is a waste rather than a
# finding.
TREE_CLEAN=1
if [[ -n "$(git status --porcelain)" ]]; then
  TREE_CLEAN=0
  fail "the working tree is dirty, so neither the native race build nor the delegated P4 gate can run: both build \`git archive HEAD\`, which would measure something other than what is here"
  info "  commit first; this gate's own criteria still run below"
fi

# ------------------------------------------------ the scalar path, unassisted
echo
echo "-- the scalar path on a stock toolchain (a P5 criterion in its own right) --"
info "the fast paths are additive build tags, so the package must be usable the day"
info "somebody runs \`go get\` on a toolchain with no experiment enabled at all"
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
if [[ "$STOCK_OK" -eq 0 ]]; then
  pass "[local, stock toolchain] every test passes without GOEXPERIMENT=simd"
else
  fail "[local, stock toolchain] every test passes without GOEXPERIMENT=simd"
  sed 's/^/        /' "$LOG" | tail -40
fi
LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
if [[ "$LOCAL_OK" -eq 0 ]]; then
  pass "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] every test passes"
else
  fail "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] every test passes"
  sed 's/^/        /' "$LOG" | tail -40
fi

HOSTS="$(remote_hosts)"
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"

# The measurement precondition, asserted rather than assumed (§5 rule 5), and the
# core count the criterion names (criterion 4).
if [[ -n "$HOSTS" ]]; then
  echo
  echo "-- hosts, governors and topology --"
  while read -r host; do
    [[ -n "$host" ]] || continue
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      fail "[$host] unreachable"
      continue
    fi
    info "[$host] $prov"
    hgov="$(sed -n 's/.*governor=\([^ |]*\).*/\1/p' <<<"$prov")"
    if [[ "$hgov" == performance ]]; then
      pass "[$host] cpufreq governor is performance (§5 rule 5)"
    elif [[ -z "$hgov" || "$hgov" == unknown ]]; then
      fail "[$host] scaling_governor is unreadable, so §5 rule 5 cannot be verified; an unchecked precondition is not a met one"
    else
      fail "[$host] cpufreq governor is '$hgov', not performance (§5 rule 5): a ramping core produces cold readings that enter the record as measurements"
      info "  [$host] sudo cpupower frequency-set -g performance"
    fi
    # The frequency-regime knob, checked here so a host without one fails in the
    # preamble rather than halfway through its measurement window (#66). Reported as
    # state + path: the path is provenance, because the polarity differs by vendor
    # (cpufreq/boost 1 = permitted, intel_pstate/no_turbo 1 = forbidden) and a reader
    # checking this by hand needs to know which knob was read.
    bst="$(remote_boost "$host" || true)"
    if [[ -z "$bst" || "${bst%% *}" == unknown ]]; then
      fail "[$host] no readable boost/turbo knob (${bst:-no answer}), so the scaling criterion's two arms cannot be asserted to share a frequency regime (#66): unreadable counts as unmet"
    else
      info "[$host] boost knob: ${bst%% *} at ${bst#* } — this gate sets it off for the judged pass and restores it"
    fi
    # Topology is provenance, and it is the first thing to look at if a host misses
    # the floor: eight goroutines across four cores and their siblings is a ceiling
    # that is not keel's (criterion 4, issue #15).
    topo="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" 'lscpu 2>/dev/null | sed -n "s/^Thread(s) per core: *//p;s/^Core(s) per socket: *//p;s/^Socket(s): *//p" | tr "\n" " "' 2>/dev/null || true)"
    ncpu="$(sed -n 's/.*| \([0-9]*\) cpus |.*/\1/p' <<<"$prov")"
    info "[$host] threads-per-core / cores-per-socket / sockets: ${topo:-unreadable}; nothing is pinned, placement is the scheduler's (#15)"
    if [[ -z "$ncpu" ]]; then
      fail "[$host] CPU count unreadable, so \"at $P5_THREADS cores\" cannot be verified here"
    elif [[ "$ncpu" -lt "$P5_THREADS" ]]; then
      fail "[$host] $ncpu CPUs, fewer than the $P5_THREADS the criterion names: this host cannot produce a reading of it"
    else
      pass "[$host] $ncpu CPUs, enough for the $P5_THREADS the criterion names"
    fi
  done <<<"$HOSTS"
fi

# --------------------------------------- parallel correctness, before throughput
echo
echo "-- parallelism as a correctness property (criteria 5, 6, 7) --"
info "bitwise identity against the serial nest, no goroutine left running, no state"
info "carried between calls. The local run is scalar-only, so all of this is audited"
info "from a host that ran it with the avx512 microkernel live (toolchain-notes T1)."

AVX512_GREEN=""
if [[ -z "$HOSTS" ]]; then
  fail "P5 needs an amd64 host with $P5_THREADS cores to execute the AVX-512 paths; none configured"
else
  if remote_build_test . "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary (root package: parallel correctness)"
  else
    fail "cross-compile of the linux/amd64 test binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] the full test suite passes with the vector backend live"
    else
      fail "[$host] the full test suite passes with the vector backend live"
      sed 's/^/        /' "$LOG" | tail -60
    fi
    active="$(marker sgemm-active "$LOG")"
    if [[ "$OK" -eq 0 && "$active" == */avx512 ]]; then
      AVX512_GREEN="$host"
      cp "$LOG" "$SWEEPLOG"
    fi

    # ---- the dispatch chain, pulled on rather than reported (criterion 7)
    for want in $P5_FORCED; do
      FOK=0
      KEEL_REMOTE_ENV="KEEL_FORCE=$want" remote_exec "$host" "$BIN" \
        -test.v -test.run 'TestKeelForce|TestBackend|TestDispatch' >"$LOG" 2>&1 || FOK=$?
      got="$(marker l1-active "$LOG")"
      gotk="$(marker sgemm-active "$LOG")"
      # Failure states told apart because they have different causes, and a red
      # that names the wrong one is as untrustworthy as a green that does (issue
      # #32's lesson, applied to a correctness check).
      #
      # The Level-1-only rungs are the interesting case. Under the #40 ruling a
      # scalar microkernel there is the *specified* outcome, not a miss — so this
      # check asserts the ceiling rather than excusing it: forcing avx2 must give
      # an avx2 Level 1 AND a scalar microkernel, and the library must say so. A
      # run that quietly produced an avx2 microkernel would now fail here, which
      # is the point: the ruling narrowed what is claimed, and a claim that grows
      # back silently is exactly what this gate exists to catch.
      if [[ "$FOK" -ne 0 ]]; then
        fail "[$host] KEEL_FORCE=$want: the forced run failed"
        sed 's/^/        /' "$LOG" | tail -20
      elif [[ "$got" != "$want" ]]; then
        fail "[$host] KEEL_FORCE=$want was asked for and the L1 dispatcher selected '${got:-none}': the override is not wired to what dispatch actually selects"
      elif set_has "$want" "$P5_L1_ONLY"; then
        if [[ "$gotk" == *"/$want" ]]; then
          fail "[$host] KEEL_FORCE=$want produced an $want microkernel ('$gotk'), but the Level-3 chain does not claim one (#40): an unmeasured rung has appeared at Level 3 without the ruling that would justify advertising it"
        elif [[ "$gotk" != *"/scalar" ]]; then
          fail "[$host] KEEL_FORCE=$want gave l1=$got and a microkernel of '${gotk:-none}': the documented ceiling is scalar at Level 3, so this is neither the $want kernel nor the fallback"
        else
          pass "[$host] KEEL_FORCE=$want takes effect as a Level-1-only rung (l1=$got, kern=$gotk — the #40 ceiling, reported honestly)"
        fi
      elif [[ "$gotk" != *"/$want" ]]; then
        fail "[$host] KEEL_FORCE=$want reached the L1 dispatcher (l1=$got) and the microkernel came back as '${gotk:-none}': this rung is in the Level-3 chain, so a backend it does not reach is a wiring bug or a missing kernel"
      else
        pass "[$host] KEEL_FORCE=$want takes (l1=$got, kern=$gotk)"
      fi
    done
    IOK=0
    KEEL_REMOTE_ENV="KEEL_FORCE=nonsense" remote_exec "$host" "$BIN" -test.run TestNothingMatchesThis >"$LOG" 2>&1 || IOK=$?
    if [[ "$IOK" -ne 0 ]]; then
      pass "[$host] KEEL_FORCE=nonsense refuses to run rather than falling back silently"
    else
      fail "[$host] KEEL_FORCE=nonsense was accepted: an unrecognized force value must fail loudly, or a typo in somebody's harness measures the wrong backend for a year"
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the suite ran green with the avx512 microkernel live (audited from $AVX512_GREEN)"
  else
    fail "no host ran the suite green with the avx512 microkernel, so the parallel markers below are unauditable"
  fi
fi

# ---- what the parallel tests declared they proved
if [[ ! -s "$SWEEPLOG" ]]; then
  fail "no avx512 test log to audit, so determinism, no-state and the declared dispatch chain are all unverified"
else
  for r in $P5_JUDGED $P5_MEASURED; do
    det="$(p5_line p5-determinism "$SWEEPLOG" "$r")"
    if [[ -z "$det" ]]; then
      fail "$r: no keel-p5-determinism marker, so nothing says the parallel result equals the serial one"
    else
      mode="$(field mode "$det")"
      thr="$(field threads "$det")"
      TMISS=""
      for t in $P5_DET_THREADS; do set_has "$thr" "$t" || TMISS="$TMISS $t"; done
      if [[ "$mode" != bitwise-vs-serial ]]; then
        fail "$r: determinism declared as mode=${mode:-none}, not bitwise-vs-serial: splitting the MC loop does not reassociate any element's reduction, so equality here is exact and a tolerance would be hiding something"
      elif [[ -n "$TMISS" ]]; then
        fail "$r: determinism thread counts missing:$TMISS (declared: ${thr:-none}) — a row-partition off-by-one hides at every power of two"
      else
        pass "$r: bit-identical to the serial nest at threads=$thr"
      fi
    fi
    ns="$(p5_line p5-nostate "$SWEEPLOG" "$r")"
    if [[ -z "$ns" ]]; then
      fail "$r: no keel-p5-nostate marker (DESIGN.md §4/P5: no background threads, no state between calls)"
    else
      checks="$(field checks "$ns")"
      NMISS=""
      for c in $P5_NOSTATE_REQ; do set_has "$checks" "$c" || NMISS="$NMISS $c"; done
      if [[ -z "$NMISS" ]]; then
        pass "$r: no background threads and no state between calls ($checks)"
      else
        fail "$r: no-state checks missing:$NMISS (declared: ${checks:-none})"
      fi
    fi
  done
  # Both chains are checked, because the #40 ruling is precisely that there are
  # two of them. A single `chain=` field would make the Level-3 narrowing
  # unstateable, and an unstateable ruling is one the next session re-litigates.
  dc="$(marker p5-dispatch "$SWEEPLOG")"
  if [[ -z "$dc" ]]; then
    fail "no keel-p5-dispatch marker, so the library never stated the chains it selects from"
  else
    gl1="$(field l1 "$dc")"
    gk="$(field kern "$dc")"
    if [[ "$gl1" != "$P5_L1_CHAIN" ]]; then
      fail "the declared Level-1 chain is '${gl1:-none}', not '$P5_L1_CHAIN' (DESIGN.md §4/P5)"
    else
      pass "Level-1 chain declared as $P5_L1_CHAIN, and this gate forced every element of it above"
    fi
    if [[ "$gk" != "$P5_KERN_CHAIN" ]]; then
      fail "the declared Level-3 chain is '${gk:-none}', not '$P5_KERN_CHAIN' (DESIGN.md §4/P5, narrowed by the #40 ruling: no AVX2 microkernel exists and no host here is AVX2-only silicon, so a third rung would be advertised without evidence)"
    else
      pass "Level-3 chain declared as $P5_KERN_CHAIN — two rungs by ruling (#40), and the avx2 ceiling asserted per host above"
    fi
  fi
fi

# ------------------------------------------------------ the race detector
echo
echo "-- race detector (a named P5 criterion) --"
info "-race needs cgo and remote_build_test is CGO_ENABLED=0 by design, so the"
info "instrumented binary is built NATIVELY on each host from git archive HEAD — the"
info "same mechanism gate-p3 uses for its OpenBLAS harness. The dev host's own run is"
info "kept as well: it is arm64 and scalar-only, but the worker pool, the shared"
info "packed-B panel and the sync.Pool are backend-independent Go, so having two"
info "architectures' schedulers look at the same concurrency is worth the minute."
# race_verdict LABEL RC LOG — "clean", "a data race" and "a test failure under
# -race" are three states, and only the middle one is about concurrency. A gate
# that calls the third one "the race detector reports a finding" sends whoever
# reads it looking for a race that is not there; the detector prints WARNING: DATA
# RACE when it finds one, so that is what gets looked for.
race_verdict() {
  local label="$1" rc="$2" log="$3"
  if [[ "$rc" -eq 0 ]]; then
    pass "$label race detector clean"
  elif grep -q 'WARNING: DATA RACE' "$log"; then
    fail "$label the race detector reports a data race"
    sed -n '/WARNING: DATA RACE/,/^==================$/p' "$log" | sed 's/^/        /' | head -60
  elif grep -q 'checkptr: converted pointer straddles multiple allocations' "$log"; then
    # A known upstream defect, and still a FAIL: naming a cause is not the same
    # as meeting the criterion. `-race` implies -d=checkptr, and archsimd's
    # partial slice ops convert &s[0] to a full-width *[N]T, which checkptr
    # rejects fatally. See docs/toolchain-notes.md T17 and issue #42.
    #
    # Ruled 2026-08-12: this criterion is not amendable to exclude checkptr. So
    # the message names the fix's address rather than saying "awaiting a
    # disposition" — the disposition exists.
    #
    # That address was #22's edge campaign until 2026-08-15, and it was wrong:
    # #22 ranks edge-handling candidates and cannot clear a checkptr fatal in
    # archsimd's own helpers. The fix is upstream CL 761120 (30 //go:nocheckptr
    # on simd's unsafe_helpers.go), which ships in go1.27 — so the remediation
    # path is #70, the go1.27.0 floor on all three hosts, and then #69, which
    # ports internal/vec to 1.27's respelled load/store surface (T23) because
    # keel does not compile under 1.27 until it does. A compiler fix is not yet
    # a fix in keel's build, and citing the wrong remediation sends the reader
    # to a campaign that will never resolve this.
    unmeasured "$label the -race run died on archsimd's checkptr violation before it could measure anything, so the criterion is unmeasured (toolchain-notes T17, #42, upstream golang/go#80856, fixed by CL 761120 in go1.27 — the path here is #69's port behind #70's toolchain floor; the criterion is not amendable)"
    sed -n '/checkptr: converted pointer straddles/,/^testing\.tRunner/p' "$log" | sed 's/^/        /' | head -20
  else
    unmeasured "$label the -race run failed without the detector reporting a race, so the criterion is unmeasured: a test that fails under instrumentation says nothing either way about whether keel has a race"
    # head, not tail: on a multi-package failure the `--- FAIL:` lines that name
    # the cause come before the per-package summaries, and a tail dropped them.
    grep -E '^(---|[[:space:]]+---)|\.go:[0-9]+:' "$log" | sed 's/^/        /' | head -20
    grep -E '^(FAIL|ok|\?)[[:space:]]' "$log" | sed 's/^/        /' | head -6
  fi
}
RACE_LOCAL=0
GOEXPERIMENT=simd go test -race -count=1 ./... >"$LOG" 2>&1 || RACE_LOCAL=$?
race_verdict "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH), scalar path]" "$RACE_LOCAL" "$LOG"

RACE_HOSTS=0
if [[ -z "$HOSTS" ]]; then
  fail "no execution hosts, so the race detector never saw the vector path"
elif [[ "$TREE_CLEAN" -eq 0 ]]; then
  unmeasured "the native race build did not run: this gate refused a dirty tree above, and a check that could not run is unmeasured rather than clean"
else
  while read -r host; do
    [[ -n "$host" ]] || continue
    hgo="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" 'command -v go >/dev/null 2>&1 && go version || echo none' 2>/dev/null || echo none)"
    if [[ "$hgo" == none || -z "$hgo" ]]; then
      unmeasured "[$host] no Go toolchain, so -race cannot be built natively here and this host's vector path is unmeasured for races"
      continue
    fi
    # KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, which would close
    # stdin and hand tar an empty archive.
    # shellcheck disable=SC2029  # client-side expansion of a client-side path is intended
    if ! git archive --format=tar HEAD | ssh "${KEEL_SCP_OPTS[@]}" "$host" \
         "rm -rf '$P5_REMOTE_SRC' && mkdir -p '$P5_REMOTE_SRC' && tar -x -C '$P5_REMOTE_SRC'" >"$LOG" 2>&1; then
      fail "[$host] could not ship the source tree for a native race build"
      sed 's/^/        /' "$LOG" | tail -20
      continue
    fi
    # GOMAXPROCS is left at the host's default: the race detector wants as many
    # schedulers as the machine has, and since the pool sizes itself from
    # GOMAXPROCS(0) that is also the widest pool this host can produce.
    # shellcheck disable=SC2029  # client-side expansion of a client-side path is intended
    NRC=0
    ssh "${KEEL_SSH_OPTS[@]}" "$host" \
      "cd '$P5_REMOTE_SRC' && GOEXPERIMENT=simd CGO_ENABLED=1 go test -race -count=1 ./... 2>&1" >"$LOG" 2>&1 || NRC=$?
    race_verdict "[$host, vector path live, $hgo, GOMAXPROCS at the host default]" "$NRC" "$LOG"
    # An `[[ ... ]] && x` here would be the loop body's last command, and a false
    # test under `set -e` would end the gate mid-host-list.
    if [[ "$NRC" -eq 0 ]]; then
      RACE_HOSTS=$((RACE_HOSTS + 1))
    fi
  done <<<"$HOSTS"
  if [[ "$RACE_HOSTS" -eq 0 ]]; then
    unmeasured "no host produced a race-detector reading on the vector path, so that criterion is unmeasured rather than clean"
  fi
fi

# -------------------------------------------- scaling at 8 cores on 4096 cubed
echo
echo "-- scaling at $P5_THREADS cores on ${P5_SIZE}^3 (the headline criterion) --"
info "-test.count=$KEEL_BENCH_COUNT -test.benchtime=$KEEL_BENCH_TIME; one invocation per host with both thread"
info "counts inside it, and the floor counts as cleared only net of both intervals"
info "judged at >= ${SCALE_FLOOR}x: $P5_JUDGED — one parallelism class (ruled 2026-08-12)"
info "measured and reported, floor deferred to this measurement plus a stated model: $P5_MEASURED (#37)"

BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)
SCALE_HOSTS_OK=0
SCALE_HOSTS_MEASURED=0
if [[ -z "$HOSTS" ]]; then
  fail "no execution hosts, so the scaling criterion cannot be evaluated"
else
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary (Scale + Peak)"
  else
    fail "cross-compile of the linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    # Re-read at the moment of measurement, not only in the preamble: a governor
    # that changed in between belongs to a machine somebody started using.
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    if [[ "$gov" != performance ]]; then
      fail "[$host] governor is '${gov:-unknown}' at measurement time, not performance: it changed after this gate's preamble read it, so nothing measured here is covered by §5 rule 5"
      continue
    fi
    # ---- the frequency regime the judged ratio is taken in (criterion 1, #66)
    #
    # Set, then READ BACK. The first version of remote_boost_set piped its value to a
    # remote `sh -s`, which $KEEL_SSH_OPTS' `-n` fed from /dev/null: the write never
    # happened and the function returned 0 on all three hosts. Only the readback
    # caught it. So the readback is the assertion and the return code is a hint.
    if ! remote_boost_set "$host" off; then
      unmeasured "[$host] boost could not be set off, so this host cannot produce a same-regime ratio (#66); its scaling is unmeasured, not missed"
      continue
    fi
    BOOST_TOUCHED="$BOOST_TOUCHED $host"
    bst="$(remote_boost "$host" || true)"
    if [[ "${bst%% *}" != off ]]; then
      fail "[$host] boost reads '${bst%% *}' after being set off (${bst#* }): unmoved or unreadable counts as unmet, so the two arms cannot be asserted to share a regime (#66)"
      remote_boost_set "$host" on >/dev/null 2>&1 || true
      continue
    fi
    pass "[$host] boost is off and read back off at ${bst#* }, so both arms of this host's ratio are taken in one frequency regime (#66)"

    # GOMAXPROCS is NOT set here. The thread count belongs to the benchmark row
    # (criterion 2) and the harness sets it per row; pinning it in the environment
    # would cap the eight-thread row at whatever this line happened to say.
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$P5_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      fail "[$host] the scaling benchmark run failed (boost-off pass)"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      remote_boost_set "$host" on >/dev/null 2>&1 || true
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"

    # ---- restore, then the boost-on pass: reported beside the verdict, and the
    # regime the README's published wall-clock rates live in (#66, criterion 9).
    BOOST_ON_MEASURED=0
    # Truncated per host, not merely overwritten. This path is reused across the host
    # loop and the boost-on pass is allowed to fail without skipping the host, so a
    # stale file here would let one host's README rows be checked against another
    # host's rates — a green with the wrong provenance, which is worse than the red.
    : >"$BENCHCSV_ON"
    remote_boost_set "$host" on >/dev/null 2>&1 || true
    bon="$(remote_boost "$host" || true)"
    if [[ "${bon%% *}" != on ]]; then
      fail "[$host] boost did not come back on (reads '${bon%% *}' at ${bon#* }): this host is left in a modified state, so every later measurement on it — including this gate's delegated gate-p4 run — is suspect"
    elif ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$P5_BENCH_FILTER" >"$BENCHLOG_ON" 2>&1; then
      unmeasured "[$host] the boost-on benchmark run failed, so the wall-clock speedup a caller experiences is unmeasured for this host (#66) and the README's rows have nothing to be re-measured against"
      sed 's/^/        /' "$BENCHLOG_ON" | tail -20
    else
      bench_csv "$BENCHLOG_ON" >"$BENCHCSV_ON" 2>"$LOG" || true
      [[ -s "$LOG" ]] && sed 's/^/        benchstat (boost-on): /' "$LOG"
      BOOST_ON_MEASURED=1
    fi

    WANT_ROWS=()
    for r in $P5_JUDGED $P5_MEASURED; do
      WANT_ROWS+=("$(scale_name "$r" 1)" "$(scale_name "$r" "$P5_THREADS")")
    done
    require_bench "[$host] the scaling ratios' inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "${WANT_ROWS[@]}" "$GATE_PEAK" || continue

    HOST_CLEARED=1
    HOST_MEASURED=1
    for r in $P5_JUDGED $P5_MEASURED; do
      one="$(scale_name "$r" 1)"
      many="$(scale_name "$r" "$P5_THREADS")"

      # ---- the numerator and the thread count: both declared, both checked
      RBAD=""
      for row in "$one" "$many"; do
        want_t="${row##*threads=}"
        fl="$(bench_line bench-flops "$BENCHLOG" "$row")"
        th="$(bench_line bench-threads "$BENCHLOG" "$row")"
        if [[ -z "$fl" ]]; then
          RBAD="$RBAD ${row}(no keel-bench-flops declaration: its rate has an unstated numerator)"
        else
          got="$(field flops "$fl")"; exp="$(flops_expect "$r" "$fl")"
          gf="$(field formula "$fl")"; wf="$(flops_formula "$r")"
          if [[ -z "$exp" ]]; then
            RBAD="$RBAD ${row}(declares no dimensions, so its flop count cannot be recomputed)"
          elif [[ "$got" != "$exp" ]]; then
            RBAD="$RBAD ${row}(declares flops=$got, this gate computes $exp)"
          elif [[ "$gf" != "$wf" ]]; then
            RBAD="$RBAD ${row}(declares formula=$gf, this gate's is $wf)"
          fi
        fi
        if [[ -z "$th" ]]; then
          RBAD="$RBAD ${row}(no keel-bench-threads declaration: nothing says how many workers this row ran on)"
        else
          gp="$(field gomaxprocs "$th")"; wk="$(field workers "$th")"
          [[ "$gp" == "$want_t" ]] || RBAD="$RBAD ${row}(ran at GOMAXPROCS=${gp:-unstated} where its own name says threads=$want_t)"
          [[ "$wk" == "$want_t" ]] || RBAD="$RBAD ${row}(used ${wk:-unstated} worker(s) where the name says $want_t: a bounded pool sized by GOMAXPROCS(0) is the design instruction)"
        fi
      done
      if [[ -n "$RBAD" ]]; then
        fail "[$host] $r: the scaling ratio's inputs do not check out:$RBAD"
        HOST_CLEARED=0; HOST_MEASURED=0
        continue
      fi

      lo="$(bench_ratio_lo "$many" "$one" "$BENCHCSV" GFLOP/s)"
      pt="$(bench_ratio "$many" "$one" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$lo" ]]; then
        fail "[$host] $r: no bounded scaling ratio — benchstat established no interval, which is a failure to measure rather than a pass"
        HOST_CLEARED=0; HOST_MEASURED=0
        continue
      fi
      info "[$host] $r: boost off — 1 thread $(bench_describe "$one" "$BENCHCSV" GFLOP/s), $P5_THREADS threads $(bench_describe "$many" "$BENCHCSV" GFLOP/s)"
      # Reported, never judged (criterion 8). A single-thread peak times the thread
      # count is the only ceiling available here, and it assumes a clock that does
      # not drop with core count — which is exactly what it does on these parts. So
      # it over-estimates the ceiling, and says so.
      p1="$(bench_gflops "$GATE_PEAK" "$BENCHCSV")"
      m8="$(bench_gflops "$many" "$BENCHCSV")"
      if [[ -n "$p1" && -n "$m8" ]]; then
        info "[$host] $r: $(awk -v a="$m8" -v b="$p1" -v t="$P5_THREADS" 'BEGIN{printf "%.1f", 100*a/(b*t)}')% of ${P5_THREADS}x the single-thread avx512 peak ($p1 GFLOP/s, boost off) — reported, not a criterion, and that denominator still ignores multi-core frequency drop within the de-boosted regime"
      fi
      # The wall-clock number a caller actually experiences, at equal prominence with
      # the verdict and judged by nothing (#66). Boost on, so the 1-thread arm gets
      # the single-core clock it really would get; that is why this ratio is the lower
      # one and why it is not the criterion. Printed even when it is missing, because
      # "the pass was not accompanied by the user-visible number" is itself the thing
      # the ruling forbids happening silently.
      if [[ "$BOOST_ON_MEASURED" -eq 1 ]]; then
        onlo="$(bench_ratio_lo "$many" "$one" "$BENCHCSV_ON" GFLOP/s)"
        onpt="$(bench_ratio "$many" "$one" "$BENCHCSV_ON" GFLOP/s)"
        if [[ -n "$onlo" ]]; then
          info "[$host] $r: boost ON, the speedup a caller experiences — ${onpt}x, ${onlo}x net of CI, against an idle single-thread rate of $(bench_describe "$one" "$BENCHCSV_ON" GFLOP/s); reported at equal prominence, judged by nothing (#66)"
        else
          info "[$host] $r: boost ON ratio unbounded (benchstat established no interval), so the caller-experienced speedup is unmeasured for this routine rather than absent"
        fi
      else
        info "[$host] $r: boost ON pass did not run on this host, so the caller-experienced speedup is unmeasured — the judged ratio above stands alone, which #66 says it should not"
      fi

      if [[ " $P5_MEASURED " == *" $r "* ]]; then
        mdl="$(p5_line p5-model "$SWEEPLOG" "$r")"
        ru="$(field rank_update "$mdl")"; ds="$(field diag_solve "$mdl")"
        if [[ -z "$mdl" || -z "$ru" || -z "$ds" ]]; then
          fail "[$host] $r scales ${pt}x (${lo}x net of CI) but declares no parallelism model: its floor is deferred TO a measurement PLUS a model (#37), and a measurement without the model sets nothing"
          HOST_MEASURED=0
        elif ! awk -v a="$ru" -v b="$ds" 'BEGIN{s=a+b; exit !(s > 0.98 && s < 1.02)}'; then
          fail "[$host] $r's declared model does not account for its work: rank_update=$ru + diag_solve=$ds does not sum to 1"
          HOST_MEASURED=0
        elif [[ -z "$STRSM_FLOOR" ]]; then
          pass "[$host] $r scales ${pt}x, ${lo}x net of CI (boost off both arms); model at this shape: rank_update=$ru diag_solve=$ds — measured and reported, no ratified floor yet (#37), and this reading is the input to setting one"
        elif awk -v v="$lo" -v f="$STRSM_FLOOR" 'BEGIN{exit !(v >= f)}'; then
          pass "[$host] $r scales ${pt}x, ${lo}x net of CI (>= ${STRSM_FLOOR}x, the floor ratified for this class from its model)"
        else
          fail "[$host] $r scales ${pt}x, ${lo}x net of CI (< ${STRSM_FLOOR}x, the floor ratified for this class)"
          HOST_CLEARED=0
        fi
        continue
      fi

      if awk -v v="$lo" -v f="$SCALE_FLOOR" 'BEGIN{exit !(v >= f)}'; then
        pass "[$host] $r scales ${pt}x at $P5_THREADS threads, ${lo}x net of CI (>= ${SCALE_FLOOR}x, boost off both arms)"
      else
        fail "[$host] $r scales only ${pt}x at $P5_THREADS threads, ${lo}x net of CI (< ${SCALE_FLOOR}x, boost off both arms)"
        HOST_CLEARED=0
      fi
    done

    # ---- retention, printed and not judged (criterion 8, issue #26)
    rk="$(marker bench-retention "$BENCHLOG")"
    [[ -n "$rk" ]] && info "[$host] retention (the blocked nest against its own microkernel): $rk — reported, never judged; #26 is a direction to work in, not a threshold invented after the fact"

    # ---- the README's published numbers, against this run (criterion 9)
    #
    # Checked against the BOOST-ON pass, because that is the regime the published
    # wall-clock rates were taken in and a README row is a claim about what a caller
    # gets, not about the gate's judging regime. If that pass did not run there is no
    # comparison to make, and saying so is the only honest verdict available: a row
    # checked against nothing must not read as a row that agreed.
    if [[ ! -r README.md ]]; then
      fail "README.md is missing, and DESIGN.md §4/P5 asks for it with honest numbers in it"
    elif [[ "$BOOST_ON_MEASURED" -ne 1 ]]; then
      unmeasured "[$host] the boost-on pass produced no rates, so this host's README rows are unmeasured this run rather than agreeing or disagreeing (#66)"
    else
      hcpu="$(remote_probe "$host" | cut -d'|' -f1 | sed 's/ *$//')"
      RROWS="$(awk -v b="$README_BEGIN" -v e="$README_END" '
        index($0, b) { inb = 1; next }
        index($0, e) { inb = 0 }
        inb && /^\|/ { print }' README.md)"
      if [[ -z "$RROWS" ]]; then
        fail "[$host] README.md has no \`keel-numbers\` block, so its published numbers cannot be re-measured by the gate that ships them"
      else
        RMATCH=0; RBADN=""
        while IFS= read -r row; do
          [[ -n "$row" ]] || continue
          rcpu="$(awk -F'|' '{gsub(/^ +| +$/, "", $2); print $2}' <<<"$row")"
          rben="$(awk -F'|' '{gsub(/^ +| +$/, "", $3); print $3}' <<<"$row")"
          rthr="$(awk -F'|' '{gsub(/^ +| +$/, "", $4); print $4}' <<<"$row")"
          rgf="$(awk -F'|'  '{gsub(/^ +| +$/, "", $5); print $5}' <<<"$row")"
          rden="$(awk -F'|' '{gsub(/^ +| +$/, "", $6); print $6}' <<<"$row")"
          # the header row and its separator carry no measurement
          [[ -z "$rcpu" || "$rcpu" == CPU || "$rcpu" == ---* ]] && continue
          [[ "$hcpu" == *"$rcpu"* ]] || continue
          RMATCH=$((RMATCH + 1))
          if [[ -z "$rden" ]]; then
            RBADN="$RBADN ${rben}/threads=${rthr}(no denominator column: never a number without one, §7 rule 7)"
            continue
          fi
          mgf="$(bench_gflops "$(scale_name "$rben" "$rthr")" "$BENCHCSV_ON")"
          if [[ -z "$mgf" ]]; then
            RBADN="$RBADN ${rben}/threads=${rthr}(published, but this gate measured no such row)"
          elif ! awk -v a="$rgf" -v b="$mgf" -v t="$README_TOL" 'BEGIN{d=(a-b)/b; if (d<0) d=-d; exit !(d <= t)}'; then
            RBADN="$RBADN ${rben}/threads=${rthr}(README says $rgf, this run measured $mgf)"
          fi
        done <<<"$RROWS"
        if [[ "$RMATCH" -eq 0 ]]; then
          fail "[$host] README.md publishes no row for '$hcpu', so this host's numbers are either unpublished or published under a CPU it does not have"
        elif [[ -z "$RBADN" ]]; then
          pass "[$host] every README row for this CPU re-measures within 5% ($RMATCH row(s))"
        else
          fail "[$host] README rows disagree with this run:$RBADN"
        fi
      fi
    fi

    SCALE_HOSTS_MEASURED=$((SCALE_HOSTS_MEASURED + HOST_MEASURED))
    SCALE_HOSTS_OK=$((SCALE_HOSTS_OK + HOST_CLEARED))
  done <<<"$HOSTS"
  if [[ "$SCALE_HOSTS_MEASURED" -eq 0 ]]; then
    unmeasured "no host produced a complete set of scaling ratios, so the headline criterion is unmeasured rather than missed"
  elif [[ "$SCALE_HOSTS_OK" -eq "$NHOSTS" ]]; then
    pass "every gate host cleared ${SCALE_FLOOR}x against its own single-thread rate for $P5_JUDGED ($SCALE_HOSTS_OK/$NHOSTS)"
  else
    fail "$SCALE_HOSTS_OK of $NHOSTS gate hosts cleared the scaling floor; the criterion is per host, against that host's own single-thread rate"
  fi
fi

# --------------------------------------- the documentation the phase promises
echo
echo "-- doc.go, and no rate outside the block this gate re-measures (criterion 9) --"
if [[ -r doc.go ]]; then
  if [[ -n "$(GOEXPERIMENT=simd go doc . 2>/dev/null | sed -n '2,$p' | tr -d '[:space:]')" ]]; then
    pass "doc.go carries a package comment that \`go doc\` renders"
  else
    fail "doc.go exists but \`go doc .\` renders no package documentation"
  fi
else
  fail "doc.go is missing (DESIGN.md §4/P5)"
fi
if [[ -r README.md ]]; then
  # The block's existence is checked HERE as well as inside the host loop, and it is
  # checked before the stray-rate scan, because a README with no rates in it at all
  # would otherwise pass the scan for the wrong reason: "no number outside the block"
  # is only a property worth reporting once there is a block. A vacuous pass is the
  # failure mode this project has spent five phases closing (§5 rule 6), and it does
  # not get to reappear in the last gate. It is also the only place the block's
  # absence is reported at all when the host loop never reached its own check.
  if ! grep -qF "$README_BEGIN" README.md || ! grep -qF "$README_END" README.md; then
    unmeasured "README.md has no \`keel-numbers\` block ($README_BEGIN ... $README_END), so there is nothing for this gate to re-measure and the criterion is unmeasured rather than met"
  fi
  STRAY="$(awk -v b="$README_BEGIN" -v e="$README_END" '
    index($0, b) { inb = 1 }
    index($0, e) { inb = 0; next }
    !inb && /GFLOP\/s/ { print FNR ": " $0 }' README.md)"
  if ! grep -q 'GFLOP/s' README.md; then
    # No verdict, deliberately: a check with nothing to check has no colour, and
    # printing a green here would be this gate reporting a property the README does
    # not yet have the content to have.
    info "README.md publishes no rate anywhere, so the stray-number scan has nothing to scan; the missing-block failure above is the one that binds"
  elif [[ -z "$STRAY" ]]; then
    pass "every GFLOP/s figure in README.md sits inside the block this gate re-measures"
  else
    fail "README.md states a rate outside the re-measured block, which makes it a claim rather than a measurement (§7 rule 7):"
    sed 's/^/        /' <<<"$STRAY" | head -10
  fi
fi

# ------------------------------------------- P4's gate, carried forward whole
echo
echo "-- carried from P4 (criterion 10): every absolute bar this ratio stands on --"
info "\">= ${SCALE_FLOOR}x single-thread\" is a ratio whose denominator this phase is chartered to"
info "change. Stage 1 makes the serial path faster, which makes this bar harder; a"
info "parallel nest that slowed it would make the bar easier. So the absolute bars are"
info "carried by running the gates that own them: gate-p4 runs gate-p3, which carries"
info "P2's kernel audit — three phases of thresholds, not one of them restated here."
if [[ "$TREE_CLEAN" -eq 0 ]]; then
  unmeasured "the delegated P4 gate did not run: this gate refused a dirty tree above, and a gate that could not run is unmeasured rather than green"
else
  mkdir -p "$(dirname "$P4LOG")"
  P4RC=0
  bash scripts/gate-p4.sh >"$P4LOG" 2>&1 || P4RC=$?
  info "full output: $P4LOG ($(grep -c '' "$P4LOG" || true) lines) — paste it verbatim into the umbrella issue beside this gate's own; it names gate-p3's log in turn"
  # Count the delegated gate's own verdict lines, not every line containing the
  # word: a bare `grep -c FAIL` also matches gate-p3's summary line *inside*
  # gate-p4's log ("47 PASS / 0 FAIL"), which made a green gate report 1 FAIL.
  #
  # UNMEASURED is counted as its own column, not folded into either. gate-p4's
  # criterion 7 is three-state since #67, and a tally that printed only PASS and
  # FAIL would show a straddled interval as neither — the same disappearing act
  # this comment's own bug did, one column over.
  P4_STRIP=$(sed $'s/\033\\[[0-9;]*m//g' "$P4LOG")
  info "$(printf '%s\n' "$P4_STRIP" | grep -c '^  PASS  ' || true) PASS / $(printf '%s\n' "$P4_STRIP" | grep -c '^  FAIL  ' || true) FAIL / $(printf '%s\n' "$P4_STRIP" | grep -c '^  UNMEASURED  ' || true) UNMEASURED, verdict: $(grep -E '^gate-p4: (GREEN|RED)' "$P4LOG" | tail -1)"
  if [[ "$P4RC" -eq 0 ]]; then
    pass "gate-p4 is green on this commit ($(git rev-parse --short HEAD)), so every rate this gate divided by is still a measured one"
  else
    fail "gate-p4 is RED on this commit (exit $P4RC), so nothing above that divides by a single-thread rate means what it says"
    printf '%s\n' "$P4_STRIP" | grep -E '^  (FAIL|UNMEASURED)  ' | sed 's/^/        /' | head -20
    info "  DESIGN.md §4's one-re-run allowance for a failing throughput sentinel applies inside the delegated gates exactly as it does when they are run directly: one immediate re-run, both outputs archived, never for a correctness criterion"
  fi
fi

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p5: GREEN"
  exit 0
fi
echo "gate-p5: RED" >&2
exit 1
