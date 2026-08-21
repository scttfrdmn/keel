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
# shellcheck source=scripts/roofline.sh
#
# NEW HERE (p2 and p3 always had it), because the scaling aggregate now calls
# `fleet_coverage`. An undefined function is the one defect shellcheck cannot see, so the
# consequence was MEASURED rather than reasoned: without this line, `$(fleet_coverage ...)`
# under this gate's own `set -euo pipefail` expands to the empty string and does NOT abort
# — the failing command is inside the substitution — so a fleet that cleared every bar
# renders `UNMEASURED 0 of 3 ... could not be judged; the other 3 cleared`, a verdict
# contradicted by its own counts. `command not found` does reach stderr: not silent,
# unattributable, which in a 900-line gate log is one line's difference.
source scripts/roofline.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

# pass/fail/unmeasured/info come from scripts/remote.sh, which every gate sources
# above: they were copied into all six gates and only one copy applied
# VERDICT_STAMP. FAIL is this gate's own counter; those helpers only raise it.
FAIL=0

# ------------------------------------------------------------ P5's own bars
# The shape and the thread count DESIGN.md §4/P5 names, and the floor it sets.
P5_SIZE=4096
P5_THREADS=8

# THE 6.0x FLOOR IS RETIRED and the judged class is compared to a ceiling this gate
# measures on the host under test (ruling on #6, 2026-08-20). DESIGN.md §4/P5 carries
# the whole of it and is the authority: the rank inversion that falsified the floor,
# why the compute arm is measured AT 8 threads, why the memory term is not yet inside
# the min() and why that omission is in the strict direction, and how CEIL_FRACTION was
# derived. Not restated here — a criterion whose reasoning lives in two places has one
# witness and two things to keep in step (§5 rule 10).
#
# Retained only as the value the disclosure names as retired, never as a comparison:
SCALE_FLOOR_RETIRED=6.0
# A REGRESSION BAR, ratified 2026-08-21 (#6 Q2) on STRSM_FLOOR's precedent fifteen lines
# down; >=90% was refused. The derivation is printed at run time, so it is not restated
# here for the reason the paragraph above gives: two copies, one witness (§5 rule 10).
CEIL_FRACTION=57.8

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
# number came from a ceiling. Its nine readings are boost-off desktop measurements,
# so they are not comparable to this fleet's; the bar binds from this commit forward.
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
# The ceiling's own rows (ruling on #6). compute_name's width tracks GATE_PEAK's so the
# two peaks can never be read from different kernels; stream_name's patterns are the
# read-only and read-modify-write halves of the bandwidth bracket.
compute_name() { printf 'Ceiling/compute/%s/threads=%d' "${GATE_PEAK#Peak/}" "$1"; }
stream_name()  { printf 'Ceiling/stream/%s/threads=%d' "$1" "$2"; }
# Two top-level alternatives, each with fewer elements than the names it selects,
# so each is depth-unconstrained and runs everything beneath it. That is the one
# reading of `go test -bench`'s two-level split which means what it looks like
# (issue #32, docs/toolchain-notes.md T15): the split on '|' happens FIRST, and
# only then does each alternative split on '/'. require_bench declares the exact
# rows this gate reads, so anything extra beneath these two costs time and nothing
# else.
P5_BENCH_FILTER='Scale|Peak|Ceiling'

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

# p5_line NAME FILE ROUTINE — the keel-NAME line belonging to one routine. The P5
# markers are emitted once per routine, so `marker`'s last-wins reading would
# audit Strsm's parallel behaviour and call it Sgemm's.
p5_line() { marker_row "$1" "$2" routine "$3"; }

echo "== gate-p5: parallelism, dispatch, polish =="
echo

# ------------------------------------------------------------- tree state (#63)
assert_no_strays

# ------------------------------------------------------------------- builds
echo "-- builds, vet and lint --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "build (GOEXPERIMENT=simd)"; else fail "build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "build (stock toolchain, scalar path, no experiment)"; else fail "build (stock toolchain, scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

gate_tmpdir
SWEEPLOG="$BINDIR/sweep-avx512.log"
: >"$SWEEPLOG"

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
test_verdict "local, stock toolchain" "$LOG" "$STOCK_OK" "every test passes without GOEXPERIMENT=simd"
LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
test_verdict "local $(go env GOHOSTOS)/$(go env GOHOSTARCH)" "$LOG" "$LOCAL_OK" "every test passes"

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
    # captured here rather than inside the helper so the topology read below shares
    # it: one ssh instead of two.
    prov="$(remote_probe "$host" || true)"
    [[ -z "$prov" ]] || info "[$host] $prov"
    assert_governor "$host" preamble "$prov"
    if [[ "$GOV_STATE" == unreachable ]]; then
      continue
    fi
    # The class is a property of the host, knowable before any benchmark runs — and
    # stated for every host that ANSWERED, which is why it is here and not only at the
    # floor check below (#90's shape: "1 of 3 not admitted" must not describe a fleet in
    # which all three were unclassified). docs/hosts.md said this gate's judged criteria
    # were "not yet" wired to admission; this is the gate the twelve-row re-measure runs
    # under, so an unwired judged criterion here is a campaign that grades rows without
    # consulting the class that decides whether they may be graded.
    #
    # AFTER the unreachable branch, not before it, and that ordering is a verdict
    # decision rather than tidiness: a host that never answered has an empty provenance
    # line, so host_admission would classify it `unknown` and print "no instance
    # identity was read from this host" — true, and a second verdict about one cause,
    # phrased as a fact about the host's identity rather than about its silence. §5 rule
    # 6, and the first line to print is the one that gets believed.
    admission_readback "$host" "$prov"
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
  remote_build_test_or_fail . "$BIN" "$LOG" \
    "cross-compiled linux/amd64 test binary (root package: parallel correctness)" \
    "cross-compile of the linux/amd64 test binary"
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
if [[ -n "$CEIL_FRACTION" ]]; then
  info "judged at >= ${CEIL_FRACTION}% of each host's own measured ${P5_THREADS}-thread ceiling: $P5_JUDGED — one parallelism class (ruled 2026-08-12; denominator ruled 2026-08-20, its 8-thread form and this fraction ratified 2026-08-21, #6)"
  info "  DERIVATION of ${CEIL_FRACTION}%: a regression bar set below every healthy observation, from the lowest judged row of build/gate-p5-651d1bd.log (keel-zen5 Ssyrk, 60.4% net of BOTH intervals) less 2.6 points of margin. >= 90% was refused because all nine judged rows sat under it. Derived from that run and enforced here, so this reading can fail it"
  info "  the 60.4% input is that run's row RE-DERIVED through bench_ratio_lo, not the 61.1% it printed: the bar's definition is unchanged, its input was computed by a site that dropped the ceiling's interval (repaired 2026-08-21, #6). The nine re-derived rows drop by 0.7 to 4.3 points, the argmin does not move"
else
  info "measured and reported against each host's own ${P5_THREADS}-thread ceiling, fraction deferred to this measurement: $P5_JUDGED — the ${SCALE_FLOOR_RETIRED}x cross-host floor is RETIRED (#6, 2026-08-20) and this class has no floor in force until the bandwidth term is measured on the fleet"
fi
if [[ -n "$STRSM_FLOOR" ]]; then
  info "judged at >= ${STRSM_FLOOR}x: $P5_MEASURED — a second class, and a REGRESSION BAR under the B-packing-residue model (ratified 2026-08-16, #37). The work split it prints is not that model: read as Amdahl it implies a ceiling all nine ratifying readings cleared (#89)"
else
  info "measured and reported, floor deferred to this measurement plus a stated model: $P5_MEASURED (#37)"
fi

BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)
SCALE_HOSTS_OK=0
SCALE_HOSTS_MEASURED=0
# The two counts the aggregate needs and used to derive. SCALE_HOSTS_MISSED is hosts
# that measured BELOW a bar, counted where the miss is graded rather than subtracted
# afterwards: #90 found the derived form printing "2 measured below it" for a fleet with
# one slow host and one that produced no ratio, and fleet_coverage's contract asks for
# the miss count precisely so that derivation cannot come back. SCALE_HOSTS_NOTADM is
# hosts whose numbers no bar governs (#104); it is neither a clear nor a miss, and the
# sentence must not let it read as either.
SCALE_HOSTS_MISSED=0
SCALE_HOSTS_NOTADM=0
if [[ -z "$HOSTS" ]]; then
  unmeasured "no execution hosts, so the scaling criterion cannot be evaluated: unmeasured, not missed"
else
  remote_build_test_or_fail ./bench "$BENCHBIN" "$LOG" \
    "cross-compiled linux/amd64 bench binary (Scale + Peak)" \
    "cross-compile of the linux/amd64 bench binary"
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
    # No boost knob is set: the fleet is virtualized and a guest owns none (ruled
    # 2026-08-17). §5 rule 5's instrument below establishes the clock instead.
    clock_head "$host" "$BENCHBIN" || continue

    # GOMAXPROCS is NOT set here. The thread count belongs to the benchmark row
    # (criterion 2) and the harness sets it per row; pinning it in the environment
    # would cap the eight-thread row at whatever this line happened to say.
    if ! remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" -test.bench="$P5_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      unmeasured "[$host] the scaling benchmark run failed, so this host's judged ratio is unmeasured rather than short of the floor"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" "$host" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchci: /' "$LOG"
    # The raw samples, kept (#110). Printed because a verdict that cannot be
    # recomputed from the numbers it was derived from is a verdict standing on a
    # log line — which is the state every judged run before this one is in, and
    # the reason their intervals can now only be re-read by band-top.
    info "[$host] samples archived: $BENCH_ARCHIVE"
    # The new instrument is checked against the one it replaced, on this run's own
    # samples. "benchci is benchstat plus resolution" is a claim until rounding its
    # CI back to %.0f%% reproduces benchstat's column cell for cell — and this gate
    # exists because the last instrument's defensible-looking behaviour was
    # published as safe without being run against the thing it described (#110).
    if go run ./tools/benchci -verify "$BENCHLOG" >/dev/null 2>"$LOG"; then
      pass "[$host] benchci reproduces the pinned benchstat: $(sed -n 's/.*-verify ok: //p' "$LOG")"
    else
      fail "[$host] benchci does not reproduce the pinned benchstat, so this host's intervals are not benchstat's statistics at higher resolution — they are a second opinion:"
      sed 's/^/        /' "$LOG"
    fi
    # Closes the series bracketing the judged sweep (§5 rule 5 as amended 2026-08-16).
    clock_post "$host" "$BENCHBIN" "$BENCHCSV" || continue

    WANT_ROWS=()
    for r in $P5_JUDGED $P5_MEASURED; do
      WANT_ROWS+=("$(scale_name "$r" 1)" "$(scale_name "$r" "$P5_THREADS")")
    done
    # The ceiling's compute rows are required, not optional: they are the denominator
    # of this gate's headline criterion, and a missing denominator is a failure to
    # measure rather than a criterion to skip.
    WANT_ROWS+=("$(compute_name 1)" "$(compute_name "$P5_THREADS")")
    require_bench "[$host] the scaling ratios' inputs" \
      "$BENCHLOG" "$BENCHCSV" GFLOP/s "${WANT_ROWS[@]}" "$GATE_PEAK" || continue

    # ---- the per-host attainable ceiling, derived and printed before anything divides
    # by it (ruling on #6, 2026-08-20). Printed once per host, not once per row: it is
    # a property of the machine, and three copies of one derivation would read as three
    # witnesses (§5 rule 10).
    CEIL8="$(bench_gflops "$(compute_name "$P5_THREADS")" "$BENCHCSV")"
    CEIL1="$(bench_gflops "$(compute_name 1)" "$BENCHCSV")"
    if [[ -z "$CEIL8" || -z "$CEIL1" ]]; then
      unmeasured "[$host] no measured ${P5_THREADS}-thread compute ceiling, so nothing here may be divided by one"
      continue
    fi
    # The shortfall 8x-the-1-thread-peak assumed away, now a number. Under 100% the old
    # denominator was unreachable; at or over it this host would have made it merely
    # unnecessary. NAMED AS A SHORTFALL AND NOT AS CLOCK DROOP: this line said "clock
    # droop" until the instrument was run on the dev host, which read 53.5% -- far past
    # anything a licence-level clock change explains, because 8 threads there land on a
    # mix of performance and efficiency cores. Core heterogeneity, SMT siblings sharing
    # one core's execution resources and shared-cache pressure all land in this same
    # number, and this gate distinguishes none of them (§5 rule 6: one cause, one
    # verdict, so a line that cannot separate three causes may not name one). The
    # criterion does not care which it is -- the point is that the shortfall is real and
    # measured rather than assumed to be zero.
    # Published because a summary that drops its CI makes its own correction unsizable (#6).
    CEIL8CI="$(bench_stat "$(compute_name "$P5_THREADS")" "$BENCHCSV" GFLOP/s | awk '{ printf "%.2f", $2*100 }')"
    info "[$host] ceiling: compute $CEIL8 GFLOP/s +/- ${CEIL8CI}% measured at $P5_THREADS threads, against $CEIL1 at 1 thread — $(awk -v a="$CEIL8" -v b="$CEIL1" -v t="$P5_THREADS" 'BEGIN{printf "%.1f", 100*a/(b*t)}')% of ${P5_THREADS}x the 1-thread reading. That shortfall is what the retired ${SCALE_FLOOR_RETIRED}x floor's denominator assumed away; this gate measures it and does not attribute it, since clock droop, core heterogeneity and shared-cache pressure are indistinguishable here"
    for p in dot axpy; do
      bw="$(bench_stat "$(stream_name "$p" "$P5_THREADS")" "$BENCHCSV" GB/s)"
      bw1="$(bench_stat "$(stream_name "$p" 1)" "$BENCHCSV" GB/s)"
      if [[ -z "$bw" ]]; then
        info "[$host] ceiling: stream/$p not measured this run, so the memory term contributes nothing to the min() — see the note on CEIL_FRACTION: omitting it is the strict direction"
      else
        bw1v="${bw1%% *}"
        info "[$host] ceiling: stream/$p ${bw%% *} GB/s at $P5_THREADS threads, ${bw1v:-none} at 1 — reported. NOT yet in the min(): converting GB/s to a FLOP/s bound needs a declared DRAM traffic count and no benchmark declares one, so the ceiling in force is the compute term alone, which can only UNDERSTATE how close this host is to its true ceiling"
      fi
    done

    HOST_CLEARED=1
    HOST_MEASURED=1
    HOST_MISSED=0
    HOST_NOTADM=0
    for r in $P5_JUDGED $P5_MEASURED; do
      one="$(scale_name "$r" 1)"
      many="$(scale_name "$r" "$P5_THREADS")"

      # ---- the numerator and the thread count: both declared, both checked
      RBAD=""
      for row in "$one" "$many"; do
        want_t="${row##*threads=}"
        fl="$(marker_row bench-flops "$BENCHLOG" name "$row")"
        th="$(marker_row bench-threads "$BENCHLOG" name "$row")"
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
      info "[$host] $r: 1 thread $(bench_describe "$one" "$BENCHCSV" GFLOP/s), $P5_THREADS threads $(bench_describe "$many" "$BENCHCSV" GFLOP/s)"
      # Reported, never judged (criterion 8). A single-thread peak times the thread
      # count is the only ceiling available here, and it assumes a clock that does
      # not drop with core count — which is exactly what it does on these parts. So
      # it over-estimates the ceiling, and says so.
      p1="$(bench_gflops "$GATE_PEAK" "$BENCHCSV")"
      m8="$(bench_gflops "$many" "$BENCHCSV")"
      if [[ -n "$p1" && -n "$m8" ]]; then
        info "[$host] $r: $(awk -v a="$m8" -v b="$p1" -v t="$P5_THREADS" 'BEGIN{printf "%.1f", 100*a/(b*t)}')% of ${P5_THREADS}x the single-thread avx512 peak ($p1 GFLOP/s) — reported, not a criterion, and that denominator ignores the clock's drop with core count"
      fi

      # Admission after the reading and before either bar (#104), the order p2 and p3 use:
      # the scaling is a fact worth having whatever the class, and only whether a FLOOR may
      # be applied is the class's business. A scaling floor is as much a claim about owned
      # silicon as percent-of-peak is — a partial-size guest's eight threads may sit on four
      # cores it shares with tenants this run cannot see.
      #
      # PER ROW, not once per host, so an unadmitted host prints its class up to three
      # times. Deliberate: adm_judgeable's contract is that it states the reading it
      # declines to judge, and each row has its own reading.
      if ! adm_judgeable "$host" "$GOV_PROV" \
           "$r scales ${pt}x at $P5_THREADS threads, ${lo}x net of CI"; then
        # Not a clear and not a miss. HOST_MEASURED drops too, because the aggregate's
        # `nmeas` counts hosts that produced a JUDGEABLE reading and this host produced
        # none — an unadmitted host left in nmeas would make "no host produced a
        # judgeable ratio" unreachable on a fleet where exactly that happened.
        HOST_NOTADM=1; HOST_CLEARED=0; HOST_MEASURED=0
        continue
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
          pass "[$host] $r scales ${pt}x, ${lo}x net of CI; model at this shape: rank_update=$ru diag_solve=$ds — measured and reported, no ratified floor yet (#37), and this reading is the input to setting one"
        elif awk -v v="$lo" -v f="$STRSM_FLOOR" 'BEGIN{exit !(v >= f)}'; then
          pass "[$host] $r scales ${pt}x, ${lo}x net of CI (>= ${STRSM_FLOOR}x, the regression bar ratified for this class 2026-08-16 under the B-packing-residue model)"
        else
          fail "[$host] $r scales ${pt}x, ${lo}x net of CI (< ${STRSM_FLOOR}x, the regression bar ratified for this class)"
          HOST_CLEARED=0
          HOST_MISSED=1
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

      # ---- achieved against this host's OWN measured ceiling (ruling on #6, 2026-08-20)
      #
      # BOTH intervals in one call, which bench.sh's contract requires: this site divided
      # bench_gflops_lo by bench_gflops until 2026-08-21 and flattered every share by the
      # ceiling's own CI. The T8/T1 ratio is still printed, because it is what every
      # published row and historical log is stated in and DESIGN.md §4/P5 re-adjudicates
      # them against this — but it is no longer what decides anything.
      ratio="$(bench_ratio_lo "$many" "$(compute_name "$P5_THREADS")" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$ratio" ]]; then
        unmeasured "[$host] $r: no bounded fraction of the ${P5_THREADS}-thread ceiling ($CEIL8 GFLOP/s) can be formed — an unbounded rate or a non-positive ceiling, and the printed ceiling says which. Either is a broken denominator rather than a verdict"
        HOST_CLEARED=0; HOST_MEASURED=0
        continue
      fi
      frac="$(awk -v x="$ratio" 'BEGIN{ printf "%.1f", 100*x }')"
      if [[ -z "$CEIL_FRACTION" ]]; then
        # The STRSM_FLOOR precedent (#37): measured, reported, and the input to setting
        # the bar rather than a bar itself. Named as unjudged so no reader can mistake a
        # silent pass for cleared coverage — this class HAS no floor in force right now.
        pass "[$host] $r reaches ${frac}% of this host's measured ${P5_THREADS}-thread ceiling ($CEIL8 GFLOP/s), scaling ${pt}x / ${lo}x net of CI — measured and REPORTED, no fraction ratified yet (#6): this reading is an input to setting one, and the retired ${SCALE_FLOOR_RETIRED}x floor is not applied"
      elif awk -v v="$frac" -v f="$CEIL_FRACTION" 'BEGIN{exit !(v >= f)}'; then
        pass "[$host] $r reaches ${frac}% of this host's measured ${P5_THREADS}-thread ceiling ($CEIL8 GFLOP/s) (>= ${CEIL_FRACTION}%), scaling ${pt}x / ${lo}x net of CI"
      else
        fail "[$host] $r reaches only ${frac}% of this host's measured ${P5_THREADS}-thread ceiling ($CEIL8 GFLOP/s) (< ${CEIL_FRACTION}%), scaling ${pt}x / ${lo}x net of CI"
        HOST_CLEARED=0
        HOST_MISSED=1
      fi
    done

    # ---- retention, printed and not judged (criterion 8, issue #26)
    rk="$(marker bench-retention "$BENCHLOG")"
    [[ -n "$rk" ]] && info "[$host] retention (the blocked nest against its own microkernel): $rk — reported, never judged; #26 is a direction to work in, not a threshold invented after the fact"

    # ---- the README's published numbers, against this run (criterion 9)
    #
    # Checked against the judged sweep, which since 2026-08-17 is the only pass there
    # is: a README row and the verdict now describe the same regime.
    if [[ ! -r README.md ]]; then
      fail "README.md is missing, and DESIGN.md §4/P5 asks for it with honest numbers in it"
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
          mgf="$(bench_gflops "$(scale_name "$rben" "$rthr")" "$BENCHCSV")"
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
    SCALE_HOSTS_MISSED=$((SCALE_HOSTS_MISSED + HOST_MISSED))
    SCALE_HOSTS_NOTADM=$((SCALE_HOSTS_NOTADM + HOST_NOTADM))
  done <<<"$HOSTS"
  # TWO BARS IN ONE TALLY, named rather than summarised, built ONCE because four renderings
  # of one clause is four places for the next constant to be typed into three of.
  # HOST_CLEARED is lowered by a miss against SCALE_FLOOR on any of P5_JUDGED *or* against
  # STRSM_FLOOR on P5_MEASURED, so once #37's constant was typed this aggregate silently
  # began covering a routine its own sentence did not mention — and a pass line crediting
  # less than it verified is the same defect as one crediting more (§5 rule 6).
  #
  # Hoisted is the CONSTANT LIST and not the possessive: collapsing both produced "2 of 3
  # gate hosts cleared its class's bar", caught by rendering the branches rather than
  # reading them, which is the whole argument for driving a verdict line.
  # Two bars, each independently deferrable to its own measurement since 2026-08-20, so
  # this is built from two halves rather than enumerated as four sentences — four
  # hand-written sentences is four places for the next constant to be typed into three
  # of. All four combinations were RENDERED before this landed, not read, for the reason
  # the paragraph above records; there is no harness for this file, so that check is a
  # session act and not a standing one (§5 rule 12: the gap is stated, not implied).
  if [[ -n "$CEIL_FRACTION" ]]; then
    BARS_J="${CEIL_FRACTION}% of each host's own ${P5_THREADS}-thread ceiling for $P5_JUDGED"
  else
    BARS_J="$P5_JUDGED reported against each host's own ceiling with no fraction in force (#6)"
  fi
  if [[ -n "$STRSM_FLOOR" ]]; then
    BARS_M="${STRSM_FLOOR}x for $P5_MEASURED"
  else
    BARS_M="$P5_MEASURED is reported unjudged (#37)"
  fi
  BARS="($BARS_J, $BARS_M)"
  # Hosts that left the loop with no verdict for a reason that is not admission: no
  # complete set of ratio inputs, no bounded interval, no declared parallelism model.
  # Named, because they are neither cleared nor slow.
  SCALE_NOCOVER=$((NHOSTS - SCALE_HOSTS_OK - SCALE_HOSTS_MISSED - SCALE_HOSTS_NOTADM))
  # Absence semantics decided once in `fleet_coverage`, not a third time here: the
  # divergent-copies defect at the verdict layer is what #90's sharing ruling was about, and
  # an inline chain here was the last candidate for it. NINDET is a literal 0 as a fact, not
  # a placeholder — gate-p3's indeterminate state is a split between two CANDIDATE
  # DENOMINATORS, and this criterion's denominator is the host's own single-thread rate, of
  # which there is exactly one. A host with no bounded ratio is counted above as no-coverage.
  case "$(fleet_coverage "$NHOSTS" "$SCALE_HOSTS_MEASURED" "$SCALE_HOSTS_OK" "$SCALE_HOSTS_MISSED" 0)" in
  unmeasured)
    unmeasured "no host produced a judgeable set of scaling ratios, so the headline criterion is unmeasured rather than missed ($SCALE_HOSTS_NOTADM of $NHOSTS not admitted to the evidentiary class, so no ratio from them is judgeable however high it reads; $SCALE_NOCOVER produced no complete set of ratios)" ;;
  pass)
    pass "every gate host cleared its class's bar $BARS against its own single-thread rate ($SCALE_HOSTS_OK/$NHOSTS)" ;;
  fail)
    fail "$SCALE_HOSTS_OK of $NHOSTS gate hosts cleared their class's bar $BARS against their own single-thread rate and $SCALE_HOSTS_MISSED measured below one ($SCALE_HOSTS_NOTADM not admitted to the evidentiary class, $SCALE_NOCOVER produced no ratio); the criterion is per host and per class — the per-host lines above say which comparison missed" ;;
  *)
    unmeasured "$((SCALE_HOSTS_NOTADM + SCALE_NOCOVER)) of $NHOSTS gate hosts could not be judged this run ($SCALE_HOSTS_NOTADM not admitted to the evidentiary class, so no scaling ratio from them is judgeable; $SCALE_NOCOVER produced no complete set of ratios at all); the other $SCALE_HOSTS_OK cleared their class's bar $BARS, and no host measured below one — the per-host PASSes above stand as measured" ;;
  esac
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
info "\">= ${SCALE_FLOOR_RETIRED}x single-thread\" was a ratio whose denominator this phase is chartered"
info "to change: stage 1 makes the serial path faster, which made that bar harder, and a"
info "parallel nest that SLOWED the serial path made it easier. This gate's own carried text"
info "predicted that hazard before it decided any verdict, and #6's ruling of 2026-08-20"
info "removed it — the denominator is now a measured hardware ceiling, which stage 1 cannot"
info "move in either direction. The absolute bars are still carried by running the gates that"
info "own them: gate-p4 runs gate-p3, which carries P2's kernel audit — three phases of"
info "thresholds, not one of them restated here."
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
gate_verdict gate-p5
