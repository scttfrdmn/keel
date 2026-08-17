#!/usr/bin/env bash
# Gate P5 — DESIGN.md §4/P5. Exits 0 only when every criterion for the phase
# holds, and there is no override flag. P5 is the last phase, so here "blocks
# the next phase" means blocks the release.
#
# The 240 lines of front matter that used to sit here — what each criterion
# measures, what this gate refuses to decide, and the judgement call behind
# every threshold below — moved verbatim to docs/gates.md, section "P5", on
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

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

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
# Ratified 2026-08-16 (#37, DESIGN.md §4/P5) as a REGRESSION BAR under the
# replacement model — the per-(jc,pc) B-packing residue plus the claim tail — after
# the printed work split failed as an Amdahl model on all nine readings (criterion 1
# above carries the arithmetic). Set below every observation on purpose: 7.0x is
# 0.403x under the lowest of nine (7.403x, janus) and above the 6.0x general floor,
# so Strsm stops being the routine with no threshold at all without pretending the
# number came from a ceiling. Binds from this commit forward, boost off both arms.
STRSM_FLOOR=7.0
# The falsified ceiling, recomputed from each run's own declared work split rather
# than carried as a constant, and reported beside the reading that clears it. Judged
# by nothing: a future reading landing BELOW it would refute nothing (the nine
# readings above it are what killed the model), so this is evidence kept live, not a
# criterion. Empty disables the line.
STRSM_AMDAHL_NOTE=1

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
#
# REVISION-STAMPED, like every top-level gate log and for the same reason (#78).
# A fixed path meant each delegated run destroyed the previous one, and the
# delegated log is the only copy of that evidence: a delegate's log is never
# separately archived, its BENCHCSV lives in a mktemp -d that is gone after the
# run, and build/ is gitignored. That bit during #73's verification — the run
# being verified overwrote the reference it was to be compared against, and the
# diff survived only because an unrelated standalone p4 log happened to exist.
# The tree is frozen for a run's whole life, so HEAD is the run's revision by
# construction; #68 is what will make a dirty tree at that revision say so, and
# until it lands the stamp distinguishes revisions but not tree states.
P4LOG="build/gate-p4-under-p5-$(git rev-parse --short HEAD 2>/dev/null || echo unknown).log"

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
# The ledger of what this gate trusts rather than checks (#73 tier C, ruled
# 2026-08-15). Declared here, where the fleet is named; printed beside the
# verdict by assumed_ledger below.
assume_fleet "$HOSTS"
require_disk
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"

# The measurement precondition, asserted rather than assumed (§5 rule 5), and the
# core count the criterion names (criterion 4).
if [[ -n "$HOSTS" ]]; then
  echo
  echo "-- hosts, governors and topology --"
  while read -r host; do
    [[ -n "$host" ]] || continue
    # This gate's copy of the governor check was the divergent one, and the divergence
    # was the *correct* version: it established that a reading exists before parsing
    # for one, so an unreachable host was never reported as an unreadable sysfs file.
    # That is the shape assert_governor was lifted from (#83). The probe output is
    # captured here rather than inside the helper because #66's boost check below
    # needs the same reading, and passing it in means one ssh instead of two.
    prov="$(remote_probe "$host" || true)"
    [[ -z "$prov" ]] || info "[$host] $prov"
    assert_governor "$host" preamble "$prov"
    if [[ "$GOV_STATE" == unreachable ]]; then
      continue
    fi
    # The frequency-regime knob, checked here so a host without one fails in the
    # preamble rather than halfway through its measurement window (#66). Reported as
    # state + path: the path is provenance, because the polarity differs by vendor
    # (cpufreq/boost 1 = permitted, intel_pstate/no_turbo 1 = forbidden) and a reader
    # checking this by hand needs to know which knob was read.
    bst="$(remote_boost "$host" || true)"
    if [[ -z "$bst" || "${bst%% *}" == unknown ]]; then
      unmeasured "[$host] no readable boost/turbo knob (${bst:-no answer}), so the scaling criterion's two arms cannot be asserted to share a frequency regime (#66): unreadable is not an exemption — it blocks this gate exactly as an unmet knob does, and stops claiming the knob was wrong when nobody could read it"
    else
      info "[$host] boost knob: ${bst%% *} at ${bst#* } — this gate sets it off for the judged pass and restores it"
    fi
    # Topology is provenance, and it is the first thing to look at if a host misses
    # the floor: eight goroutines across four cores and their siblings is a ceiling
    # that is not keel's (criterion 4, issue #15). Read out of the provenance line
    # printed above, not from a second ssh running lscpu — same facts, one round
    # trip fewer, and no dependency on a package that is not installed everywhere
    # (#82). It is also now in every gate's archived record, not just this one's.
    ncpu="$(sed -n 's/.*| \([0-9]*\) cpus |.*/\1/p' <<<"$prov")"
    smt="$(sed -n 's/.*| smt=\([0-9?]*\) |.*/\1/p' <<<"$prov")"
    info "[$host] smt=${smt:-?} threads/core, so the criterion's $P5_THREADS goroutines can span as few as $P5_THREADS/${smt:-?} physical cores; nothing is pinned either way, placement is the scheduler's (#15)"
    # The two branches are the three-way taxonomy's pair, and #73 rules them
    # apart deliberately: an unreadable count is a reading nobody got, a count
    # that reads short is a reading the gate has and the environment fails. Both
    # block; only the attribution differs.
    if [[ -z "$ncpu" ]]; then
      unmeasured "[$host] CPU count unreadable, so \"at $P5_THREADS cores\" cannot be verified here"
    elif [[ "$ncpu" -lt "$P5_THREADS" ]]; then
      fail "[$host] $ncpu CPUs were read and the criterion names $P5_THREADS: this host is too small to produce a reading of it"
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
  unmeasured "P5 needs an amd64 host with $P5_THREADS cores to execute the AVX-512 paths and none is configured, so they are unmeasured on real silicon"
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
    elif remote_vanished; then
      unmeasured "[$host] the test suite did not finish (#62), so this host says nothing about the vector backend either way"
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
      # A failed forced run splits two ways under #73's taxonomy, and the marker
      # is the discriminator: no dispatch line at all means the binary never got
      # far enough to say what it selected, so the override is unmeasured here;
      # a dispatch line plus a nonzero exit means the binary ran and something it
      # asserts about its own dispatch is untrue, which is a FAIL about keel.
      if [[ "$FOK" -ne 0 && -z "$got$gotk" ]]; then
        unmeasured "[$host] KEEL_FORCE=$want: the forced run failed before reporting any dispatch, so what the override selects is unmeasured here rather than wrong"
        sed 's/^/        /' "$LOG" | tail -20
      elif [[ "$FOK" -ne 0 ]]; then
        fail "[$host] KEEL_FORCE=$want: the forced run failed with dispatch reporting l1='${got:-none}' kern='${gotk:-none}', so the binary ran and something it asserts is untrue"
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
    # VANISHED IS CHECKED FIRST, and this is the one site in the tree where that
    # ordering is the difference between a check and a forgery: nonzero means PASS
    # here, so any status that is not the program's own certifies a refusal that was
    # never observed. It predates #62 — ssh returned 255 for a dead host and this
    # printed PASS for it — and #62 only made the class nameable.
    if remote_vanished; then
      unmeasured "[$host] the KEEL_FORCE=nonsense run did not finish (#62), so whether an unrecognized value is refused was never observed: a run that never happened is not a refusal, and a nonzero status is what this check reads as success"
      sed 's/^/        /' "$LOG" | tail -20
    elif [[ "$IOK" -ne 0 ]]; then
      pass "[$host] KEEL_FORCE=nonsense refuses to run rather than falling back silently"
    else
      fail "[$host] KEEL_FORCE=nonsense was accepted: an unrecognized force value must fail loudly, or a typo in somebody's harness measures the wrong backend for a year"
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the suite ran green with the avx512 microkernel live (audited from $AVX512_GREEN)"
  else
    unmeasured "no avx512-green suite run to audit, so the parallel markers below are unmeasured (a fleet that ran green without avx512 is a separate verdict, and the per-host lines above carry it)"
  fi
fi

# ---- what the parallel tests declared they proved
if [[ ! -s "$SWEEPLOG" ]]; then
  unmeasured "no avx512 test log to audit, so determinism, no-state and the declared dispatch chain are all unmeasured"
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
  unmeasured "no execution hosts, so the race detector never saw the vector path: unmeasured, not clean and not raced"
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
if [[ -n "$STRSM_FLOOR" ]]; then
  info "judged at >= ${STRSM_FLOOR}x: $P5_MEASURED — a second class, and a REGRESSION BAR under the B-packing-residue model (ratified 2026-08-16, #37). The work split it prints is not that model: read as Amdahl it implies a ceiling all nine ratifying readings cleared (#89)"
else
  info "measured and reported, floor deferred to this measurement plus a stated model: $P5_MEASURED (#37)"
fi

BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)
SCALE_HOSTS_OK=0
SCALE_HOSTS_MEASURED=0
if [[ -z "$HOSTS" ]]; then
  unmeasured "no execution hosts, so the scaling criterion cannot be evaluated: unmeasured, not missed"
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
    # Split the way its two twins were (#73): this site was outside that sweep
    # because the collapsed branch printed '${gov:-unknown}' and so never *said* it
    # could not look. #76's guard makes the empty case reachable rather than fatal,
    # and a reachable branch that attributes an unanswered host to a governor that
    # changed is the exact defect #73 named — now one function's problem (#83), which
    # also retires this copy's lone "preamble read it" wording for "checked it".
    assert_governor "$host" measured
    clock_gate "$host" || continue
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
    # The read-back, split by #73: the old single branch collapsed "the knob was
    # read and did not move" with "nobody could read the knob", and only the
    # first is a statement about the environment.
    if [[ -z "$bst" || "${bst%% *}" == unknown ]]; then
      unmeasured "[$host] boost is unreadable after being set off (${bst:-no answer}), so this host's two arms cannot be asserted to share a regime (#66): unreadable is not an exemption, it is a reading nobody got"
      remote_boost_set "$host" on >/dev/null 2>&1 || true
      continue
    elif [[ "${bst%% *}" != off ]]; then
      fail "[$host] boost reads '${bst%% *}' after being set off (${bst#* }): the knob was read and did not move, so the two arms cannot be asserted to share a regime (#66)"
      remote_boost_set "$host" on >/dev/null 2>&1 || true
      continue
    fi
    pass "[$host] boost is off and read back off at ${bst#* }, so both arms of this host's ratio are taken in one frequency regime (#66)"

    # HERE, not up at clock_gate, and this is the reason clock_head is its own call: the
    # boost knob above changes the regime the sweep will run in, so a head window taken
    # before it would be the first point of a three-point trend with an intervention
    # between it and the other two — a declining series this gate caused (§5 rule 5, #23).
    clock_head "$host" "$BENCHBIN" || { remote_boost_set "$host" on >/dev/null 2>&1 || true; continue; }

    # GOMAXPROCS is NOT set here. The thread count belongs to the benchmark row
    # (criterion 2) and the harness sets it per row; pinning it in the environment
    # would cap the eight-thread row at whatever this line happened to say.
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$P5_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the scaling benchmark run failed (boost-off pass), so this host's judged ratio is unmeasured — the same event the boost-on pass below already called unmeasured"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      remote_boost_set "$host" on >/dev/null 2>&1 || true
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    # Closed here, before the restore below, so all three windows sit inside the one
    # boost-off regime the judged ratio is taken in (§5 rule 5, #23). It brackets the
    # boost-off sweep only; the boost-on pass below is a second window this series says
    # nothing about, which costs nothing today because a host with no governor to assert
    # is also a host with no boost knob to set, and remote_boost_set refuses it above.
    clock_post "$host" "$BENCHBIN" "$BENCHCSV" || { remote_boost_set "$host" on >/dev/null 2>&1 || true; continue; }

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
        unmeasured "[$host] $r: no bounded scaling ratio — benchstat established no interval, which is a failure to measure rather than a pass"
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
          pass "[$host] $r scales ${pt}x, ${lo}x net of CI (>= ${STRSM_FLOOR}x, the regression bar ratified for this class 2026-08-16 under the B-packing-residue model, boost off both arms)"
        else
          fail "[$host] $r scales ${pt}x, ${lo}x net of CI (< ${STRSM_FLOOR}x, the regression bar ratified for this class)"
          HOST_CLEARED=0
        fi

        # What the printed split is, and — the part a reader cannot reconstruct — what
        # it is NOT. The gate still requires the work accounting above, so it still
        # prints; but it was once read as this routine's serial fraction, and nine
        # readings clearing the ceiling that reading implies is what retired the model
        # (#37/#89). Recomputed here from THIS run's declared split, so if the shape's
        # work accounting ever moves, the ceiling moves with it and the comparison
        # stays about the current run rather than about 2026-08-16.
        if [[ -n "$STRSM_AMDAHL_NOTE" ]]; then
          amd="$(awk -v s="$ds" -v p="$P5_THREADS" 'BEGIN{ if (s < 0 || s >= 1 || p < 1) exit 1; printf "%.4f", 1/(s + (1-s)/p) }')" || amd=""
          if [[ -z "$amd" ]]; then
            info "[$host] $r work split at this shape: rank_update=$ru diag_solve=$ds — a work split, NOT a serial fraction; no Amdahl ceiling computed from it this run (diag_solve out of range)"
          else
            rel="$(awk -v v="$lo" -v c="$amd" 'BEGIN{ if (c <= 0) exit 1; printf "%+.2f%%", (v/c - 1)*100 }')" || rel="?"
            side="$(awk -v v="$lo" -v c="$amd" 'BEGIN{print (v >= c) ? "above" : "below"}')"
            info "[$host] $r work split at this shape: rank_update=$ru diag_solve=$ds. This is a WORK split, not a serial fraction: read as Amdahl s=$ds at p=$P5_THREADS it implies a ceiling of ${amd}x, and this run's ${lo}x sits $rel $side it. Nine readings above that ceiling are what falsified the model (#37/#89); the bar above rests on the B-packing residue instead. Reported, judged by nothing"
          fi
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
      hcpu="$(remote_probe "$host" | cut -d'|' -f1 | sed 's/ *$//' || true)"
      RROWS="$(awk -v b="$README_BEGIN" -v e="$README_END" '
        index($0, b) { inb = 1; next }
        index($0, e) { inb = 0 }
        inb && /^\|/ { print }' README.md)"
      if [[ -z "$RROWS" ]]; then
        fail "[$host] README.md has no \`keel-numbers\` block, so its published numbers cannot be re-measured by the gate that ships them"
      elif [[ -z "$hcpu" ]]; then
        # The missing block above is checked first and stays FAIL: that is a fact
        # about this repository, true whoever asks. This one is a fact about the
        # host, and an empty CPU model matches no README row at all — which the
        # RMATCH branch below would have reported as "publishes no row for ''",
        # a claim about the README earned by a host that stopped answering (#76).
        unmeasured "[$host] the CPU model is unreadable, so this host's README rows cannot be located: an empty model matches no row, and that is not the README publishing none"
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
    # TWO BARS IN ONE TALLY, named rather than summarised. HOST_CLEARED is lowered by
    # a miss against SCALE_FLOOR on any of P5_JUDGED *or* against STRSM_FLOOR on
    # P5_MEASURED, so once #37's constant was typed this aggregate silently began
    # covering a routine its own sentence did not mention. A pass line that credits
    # less than it verified is the same defect as one that credits more; both leave a
    # reader unable to reconstruct which comparison moved (§5 rule 6).
    if [[ -n "$STRSM_FLOOR" ]]; then
      pass "every gate host cleared its class's bar against its own single-thread rate ($SCALE_HOSTS_OK/$NHOSTS): ${SCALE_FLOOR}x for $P5_JUDGED, ${STRSM_FLOOR}x for $P5_MEASURED"
    else
      pass "every gate host cleared ${SCALE_FLOOR}x against its own single-thread rate for $P5_JUDGED ($SCALE_HOSTS_OK/$NHOSTS); $P5_MEASURED is reported unjudged (#37)"
    fi
  else
    if [[ -n "$STRSM_FLOOR" ]]; then
      fail "$SCALE_HOSTS_OK of $NHOSTS gate hosts cleared their class's bar (${SCALE_FLOOR}x for $P5_JUDGED, ${STRSM_FLOOR}x for $P5_MEASURED); the criterion is per host and per class, against that host's own single-thread rate — the per-host lines above say which comparison missed"
    else
      fail "$SCALE_HOSTS_OK of $NHOSTS gate hosts cleared the scaling floor; the criterion is per host, against that host's own single-thread rate"
    fi
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
  P4V="$(grep -E '^gate-p4: (GREEN|RED)' "$P4LOG" | tail -1 || true)"
  info "$(printf '%s\n' "$P4_STRIP" | grep -c '^  PASS  ' || true) PASS / $(printf '%s\n' "$P4_STRIP" | grep -c '^  FAIL  ' || true) FAIL / $(printf '%s\n' "$P4_STRIP" | grep -c '^  UNMEASURED  ' || true) UNMEASURED, verdict: ${P4V:-none printed}"
  # Three-way on the delegate (#76), the identical shape to gate-p4's reading of
  # gate-p3 — and this is the deeper of the two chains, so a death inside gate-p3
  # arrives here twice removed: gate-p4 now reports it as UNMEASURED, and this gate
  # must not convert that into a RED of its own on the way past. Exit 0 with a GREEN
  # line, exit 1 with a RED line; anything else is a delegate that did not reach a
  # verdict, and DESIGN.md §5.6 forbids inventing one for it.
  if [[ "$P4RC" -eq 0 && "$P4V" == *GREEN* ]]; then
    pass "gate-p4 is green on this commit ($(git rev-parse --short HEAD)), so every rate this gate divided by is still a measured one"
  elif [[ "$P4RC" -eq 1 && "$P4V" == *RED* ]]; then
    fail "gate-p4 is RED on this commit (exit $P4RC), so nothing above that divides by a single-thread rate means what it says"
    printf '%s\n' "$P4_STRIP" | grep -E '^  (FAIL|UNMEASURED)  ' | sed 's/^/        /' | head -20
    info "  DESIGN.md §4's one-re-run allowance for a failing throughput sentinel applies inside the delegated gates exactly as it does when they are run directly: one immediate re-run, both outputs archived, never for a correctness criterion"
  else
    unmeasured "gate-p4 reached no verdict on this commit (exit $P4RC, verdict line: ${P4V:-none printed}), so the absolute bars this gate's ratio stands on are unverified rather than red"
    printf '%s\n' "$P4_STRIP" | tail -20 | sed 's/^/        /'
    info "  the delegated log is $P4LOG in full, and it names gate-p3's in turn; an exit that is neither 0 nor 1 is the delegate dying, which is a defect to find rather than a threshold to re-run"
  fi
fi

assumed_ledger

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p5: GREEN"
  exit 0
fi
echo "gate-p5: RED" >&2
exit 1
