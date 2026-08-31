#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# gate-lib.sh — helpers that were byte-identical in two or more gates. Sourced
# after remote.sh and bench.sh, which it depends on: require_bench calls
# bench_expect and unmeasured, so sourcing this first gives a function that fails
# only when a criterion tries to use it.
#
# Third instalment of an established lift: remote.sh:128-143 records pass/fail/
# info/unmeasured coming out of six gates, and 708ddbb did the governor check.
# Every body here was compared byte-for-byte across its copies before moving, and
# all seven were identical. What was NOT identical is the comments — and they
# diverged only in bookkeeping about the duplication itself ("Same helper, same
# wording and the same reason as gate-p3.sh", then "as gate-p3.sh and gate-p4.sh").
# That tax was quadratic in the number of copies, and being duplication-aware was
# the only work those lines did, so the lift deletes them rather than merging them.
# Each helper below keeps the fullest of its copies' comments, which is the one
# that stated the reason instead of naming the neighbours.
#
# WHAT DOES NOT BELONG HERE: the carried-from-P2 threshold constants
# (PEAK_FLOOR, ROOF_FLOOR, SWEEP_BEST_IPF, ...). gate-p3.sh:346-350 says why in
# terms — they are "duplicated rather than factored into a shared file on purpose",
# because two independent statements of a threshold are what make a divergence
# visible. This file existing is not an argument for moving them into it. What does
# belong here is the *reconciliation* of one against a live derivation
# (reconcile_sweep_best_ipf, below): a check is not a threshold, and both gates run
# the identical one.
#
# Nor do helpers that merely look alike. check_flops_decl was drafted for this file
# and refused: gate-p4's and gate-p5's flop-declaration checks differ in three of four
# messages and fork on control flow, so a shared body needed a mode flag. Ruled
# 2026-08-18 — when the lifted function must be told which caller it is, nothing was
# lifted. The law is one semantic, one definition; it never licensed lookalikes.

# require_bench LABEL LOG CSV UNIT NAME... — declare what a criterion is about to
# read, and give absence exactly one verdict.
#
# Returns 0 when every declared benchmark produced its full complement of rows, so
# the caller reads only measurements it has already confirmed exist. Otherwise it
# emits a single "unmeasured" failure naming the run as the suspect, and the caller
# must skip the criterion rather than interpret the gap. See bench_expect in
# scripts/bench.sh for why empty is not a readable value, and DESIGN.md §5 rule 6
# for why an unmeasured criterion may not resolve as either colour.
require_bench() {
  local label miss
  label="$1"; shift
  miss="$(bench_expect "$@")" && return 0
  unmeasured "$label $miss — a criterion cannot be resolved in either direction until every benchmark it reads has its rows, so this is neither a pass nor a miss"
  return 1
}

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

# audit_ipf_tile TILE FILE — audit_ipf for a tile named by its MRxNR, empty TILE
# meaning "no tile declared", which is not an error.
audit_ipf_tile() {
  [[ -n "$1" ]] || return 0
  audit_ipf "Kernel$1" "$2"
}

# field KEY LINE — the value of a `key=value` token in a marker line.
field() {
  awk -v k="$1" '{
    for (i = 1; i <= NF; i++) {
      n = index($i, "=")
      if (n && substr($i, 1, n - 1) == k) { print substr($i, n + 1); exit }
    }
  }' <<<"$2"
}

# marker NAME FILE — the last `keel-NAME:` value in FILE.
marker() { sed -n "s/.*keel-$1: *//p" "$2" | tail -1; }

# marker_all NAME FILE — every `keel-NAME:` value, one per line. The markers this
# reads are emitted once per size in gate-p3, and once per routine or once per
# (routine, size) in gate-p4 — three vocabularies over one shape, which is why the
# comment describes the shape and each caller declares its own vocabulary.
marker_all() { sed -n "s/.*keel-$1: *//p" "$2"; }

# set_has SET VALUE — is VALUE one of SET's comma-separated members.
set_has() { [[ ",$1," == *",$2,"* ]]; }

# test_verdict NAME LOG RC PHRASE — the pass/fail/paste-the-tail triple that every
# gate wraps around a `go test` run. Eight copies, in all six gates, byte-identical
# but for the phrase: gate-p5 says "every test passes" where p3/p4 say "all tests
# pass", and p0/p1/p2 each name the package they ran. So PHRASE is a parameter and
# not a decision — normalising it would have edited three gates' output text, which
# is the one thing a de-duplication may not do.
test_verdict() {
  local name="$1" log="$2" ok="$3" phrase="$4"
  if [[ "$ok" -eq 0 ]]; then
    pass "[$name] $phrase"
  else
    fail "[$name] $phrase"
    sed 's/^/        /' "$log" | tail -40
  fi
}

# gate_tmpdir — the scratch paths every gate builds into, plus the trap that removes
# them. Sets LOG, BINDIR, BIN, BENCHBIN, BENCHLOG and BENCHCSV in the caller, which is
# the point: five gates declared the same six and then each added its own tail
# (AUDITKERN, SWEEPLOG, ALTCSV, KERNBIN, ...). The shared prefix lifts; the tail stays.
#
# THE SIGNAL TRAPS EXIT RATHER THAN CLEAN, which neither prior copy did, and each
# prior copy was wrong in its own direction. Measured, bash 3.2.57 and 5.3.15 alike:
#   SIGTERM, `EXIT` alone         -> cleaned, rc=143. The EXIT trap DOES run on an
#                                    untrapped fatal signal, so p1..p4 never leaked
#                                    and gate-p5's comment claiming otherwise was false.
#   SIGTERM, `EXIT INT TERM`      -> cleaned, then RESUMED. gate-p5 has been carrying
#                                    on past a TERM with its scratch dir deleted.
#   SIGINT to the process group   -> `EXIT` alone RESUMES at rc=0: a real Ctrl-C killed
#     (i.e. an actual Ctrl-C)        the `sleep`/`go test` child and the gate kept going.
# Exiting from the handler runs the EXIT trap, which removes the directory once: rc=143
# on TERM and rc=130 on a group SIGINT, neither RESUMED.
gate_tmpdir() {
  LOG="$(mktemp)"
  BINDIR="$(mktemp -d)"
  BIN="$BINDIR/keel.test"
  BENCHBIN="$BINDIR/bench.test"
  BENCHLOG="$BINDIR/bench.log"
  BENCHCSV="$BINDIR/bench.csv"
  trap 'rm -rf "$LOG" "$BINDIR"' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
}

# assert_kern_audit_drift LOG AUDITFILE HOST EXTRA — the insns/FMA internal/kern
# records for each shipped shape, against what the audit counts in the object code
# emitted. Checked from a marker a host produced, not from source: the point is what
# the shipped library believes. gate-p3 criterion 5b part 1 and gate-p4 criterion 4,
# whose executable lines were byte-identical; only the fail message's trailing clause
# differed, so EXTRA is a parameter for the reason test_verdict's PHRASE is.
#
# SETS DRIFT_CHECKED IN THE CALLER, hence absent from the local list. Both gates read
# it after their host loop to tell "no drift" from "no host reported the marker", and
# printing the second when every host did print it is #32/#33's misattribution.
assert_kern_audit_drift() {
  local log="$1" auditfile="$2" host="$3" extra="$4" kaudit bad="" pair rtile rval raud
  kaudit="$(marker bench-kern-audit "$log")"
  [[ -z "$DRIFT_CHECKED" && -n "$kaudit" ]] || return 0
  DRIFT_CHECKED="$host"
  for pair in $kaudit; do
    rtile="${pair%%/*}"; rval="${pair#*=}"
    raud="$(audit_ipf_tile "$rtile" "$auditfile")"
    if [[ -z "$raud" ]]; then
      bad="$bad ${rtile}(recorded $rval, not audited by this gate)"
    elif ! awk -v a="$rval" -v b="$raud" 'BEGIN{exit !(a - b < 0.001 && b - a < 0.001)}'; then
      bad="$bad ${rtile}(recorded $rval, audited $(printf '%.3f' "$raud"))"
    fi
  done
  if [[ -z "$bad" ]]; then
    pass "every shipped shape's recorded insns/FMA matches the audited object code ($kaudit)"
  else
    fail "the shape ranking reads stale instruction counts:$bad — internal/kern's registry has drifted from the K-loop it describes$extra"
  fi
}

# marker_row NAME FILE KEY VALUE — the first keel-NAME line carrying the token
# `KEY=VALUE`, or nothing. `marker`'s last-wins reading is wrong wherever a marker
# is emitted more than once per file, which is every P4 and P5 marker and gate-p3's
# per-size verification line; each caller names the key that distinguishes its own
# rows. Five copies of this awk existed, three of them as helpers and two inlined —
# including one inlined in gate-p4 three hundred lines below gate-p4's own helper,
# which is how a divergence starts.
marker_row() {
  marker_all "$1" "$2" | awk -v want="$3=$4" '{
    for (i = 1; i <= NF; i++) if ($i == want) { print; exit }
  }'
}

# flops_expect ROUTINE LINE — a gate's own count of ROUTINE's USEFUL flops at the
# dimensions the marker declares, recomputed rather than trusted:
#
#   Sgemm: 2*m*n*k        one multiply and one add per (i, j, p).
#   Ssyrk: k*n*(n+1)      the same, over one triangle including its diagonal —
#                         about half of Sgemm at the same n, so a wrong count moves
#                         gate-p4's ratio by 2x and its bar would be cleared by a
#                         routine that did not clear it.
#   Ssymm: 2*m*n*k        A is symmetric and k x k, but every entry of C still gets
#                         a full k-deep dot product: symm's saving is memory, never
#                         arithmetic, so its count is GEMM's.
#   Strsm: n*m*(m+1)      one multiply-add per (row, column) pair of one triangle
#                         including the diagonal, per right-hand side.
#
# USEFUL, not executed: a masked diagonal tile is computed whole and half of it
# discarded. Counting what the routine delivers rather than what it performed is
# what makes the ratios mean something — the discarded half is precisely the cost
# they exist to expose, and counting it as work would hide it.
#
# SHARING THIS DOES NOT WEAKEN CRITERION 7, which is the question a reader of
# gate-p3.sh:346-350 should ask here. That criterion is a gate checking the *Go
# harness*'s declared flops and formula against an independent recomputation; the
# independence that matters is gate-against-harness, and two gates agreeing on the
# recomputation costs it nothing. A shared *threshold* would be different, which is
# why the thresholds stay where they are. gate-p4 used to carry the Sgemm/Ssyrk
# arms only, and calls this for those two routines alone, so the extra arms are
# unreachable from there rather than newly permissive.
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
# arithmetic is checked by flops_expect; this checks the STATEMENT, so a harness
# that changed its reasoning and happened to land on the same number at one shape
# still has to say what it now believes.
flops_formula() {
  case "$1" in
    Sgemm|Ssymm) printf '2*m*n*k' ;;
    Ssyrk)       printf 'k*n*(n+1)' ;;
    Strsm)       printf 'n*m*(m+1)' ;;
  esac
}

# ---------------------------------------------------------------------------
# The BASELINE-REGISTERED class (#6, ruled 2026-08-21). scripts/host-baselines.tsv
# carries the whole of the design and is the authority; these are its readers, here
# rather than in gate-p5.sh only because a decision no harness can drive is a
# decision nothing checks, and scripts/baseline-test.sh drives every branch of these.

# era_current LEDGER — print the current era's id, or nothing at all.
#
# The current era is the LAST data row, and it is validated rather than searched for:
# a ledger whose final row is malformed, unnamed, or cites no amendment yields NOTHING,
# and the caller must treat that as unmeasured. Skipping a bad row to reach an older
# valid one would be the opposite of failing closed — it would silently judge new-era
# readings against old-era baselines, which is the exact misattribution eras exist to
# prevent. The amendment column is what an era is made of (measurement-eras.tsv states
# the two conditions), so an era with no cited amendment is not an era: that is the
# loophole guard, and it lives here because a guard only in prose is a wish.
era_current() {
  local tsv="$1"
  [[ -r "$tsv" ]] || return 0
  awk -F'\t' '
    /^#/ || /^[[:space:]]*$/ || $1 == "era" { next }
    { last = $0; n = NF }
    END {
      if (last == "") exit 0
      split(last, f, "\t")
      # No `f[1] != ""` conjunct: an unnamed era would print an empty line, which every
      # caller already reads as "no era" — the guard was unkillable by mutation because it
      # changed nothing observable. Deleted rather than explained; the fixture asserting an
      # unnamed era resolves nothing still stands, on the outcome instead of the branch.
      if (n >= 5 && f[3] != "" && f[3] != "—") print f[1]
    }' "$tsv"
}

# era_provisional LEDGER ERA — 0 while the era's both-arms transition archive is absent.
# Provisional is a DISCLOSURE, not a permission: it widens nothing about what BASELINE
# covers, because what bounds BASELINE is the witness index. It exists so a run under a
# freshly opened era says so in its own log rather than reading as a settled state.
era_provisional() {
  local tsv="$1" era="$2"
  [[ -r "$tsv" && -n "$era" ]] || return 1
  awk -F'\t' -v e="$era" '
    /^#/ || $1 == "era" { next }
    NF >= 5 && $1 == e { if ($4 == "" || $4 == "—") prov = 1; exit }
    END { exit !prov }' "$tsv"
}

# baseline_lookup TSV CPU CRIT ERA — the registry row for one host × criterion × era,
# or nothing. Absent file and absent row are deliberately the SAME answer: "this host is
# not registered" is what the caller needs, and a missing registry is a repository with no
# registered hosts. A row from ANOTHER era does not match, and that is the point rather
# than an inconvenience — it is what makes an instrument change render BASELINE fleet-wide
# instead of booking the methodology delta against every host as drift. What separates
# newness from an unmet obligation is baseline_spent, below, and keeping the two apart is
# the point: a lookup that also guessed at newness would decide the verdict in the accessor.
baseline_lookup() {
  local tsv="$1" cpu="$2" crit="$3" era="$4"
  [[ -r "$tsv" && -n "$era" ]] || return 0
  awk -F'\t' -v c="$cpu" -v k="$crit" -v e="$era" '
    /^#/ || $1 == "cpu_model" { next }
    NF >= 8 && $1 == c && $2 == k && $3 == e { print; exit }' "$tsv"
}

# baseline_spent INDEX CPU ERA — 0 if this CPU model has already been judged in this era,
# which is what forbids a second BASELINE. The witness is scripts/judged-runs.tsv: TRACKED
# state, keyed (cpu_model, era), read by anyone with the repo.
#
# THE PREDECESSOR WAS A GLOB OVER build/, and it was right only on the machine that had
# been running the gate (#114). build/ is gitignored, so on a fresh clone, on CI, or on a
# second operator's machine every host was `new` forever and BASELINE renewed on every
# run — a per-machine witness defeats single-shot exactly as thoroughly as the permanent
# exemption the class was built to kill, and more quietly. The self-citation guard the
# glob needed (excluding this run's own archives by exact rev, so the run writing the
# evidence could not cite itself) is gone with it: a witness row is landed by a reviewed
# commit, so a run cannot mint its own precedent mid-flight. judged-runs.tsv states what
# the trade costs — reviewed-and-visible in place of automatic-and-invisible.
baseline_spent() {
  local tsv="$1" cpu="$2" era="$3"
  [[ -r "$tsv" && -n "$cpu" && -n "$era" ]] || return 1
  awk -F'\t' -v c="$cpu" -v e="$era" '
    /^#/ || $1 == "cpu_model" { next }
    NF >= 6 && $1 == c && $2 == e { found = 1; exit }
    END { exit !found }' "$tsv"
}

# baseline_state REGISTRY WITNESS CPU CRIT ERA DERIVED_FROM — which bar governs one host x
# one criterion, as ONE WORD (#119, extending §5 rule 17 to the ratio criteria).
#
# TWO CRITERIA CONSUME THIS, and that is why it is a function rather than a second copy.
# Until #119 only the share-of-ceiling criterion had per-host admission; the scaling-ratio
# bar judged every host that reported, including hosts absent from its derivation set by
# chronology rather than by any property of their code — which is rule 17's entire subject,
# unenforced one criterion over.
#
# NO CALLER FLAG, deliberately. The derivation list is an ARGUMENT because the two criteria
# genuinely have different ones — the share bar was derived on two CPU models and the ratio
# bar on three, skx being in the second and not the first — and a differing list is data, not
# a mode. What a state MEANS stays at the call site: the units, the margin, the wording and
# which counters drop are per criterion, and a classifier told which criterion it serves
# would be two functions wearing one name.
#
# States, fail-closed one first:
#   nokey       no CPU model, so no bar can be keyed to this host and none may be applied
#   conflict    the derivation list AND the registry both claim it: two artifacts disagree
#   fleet       in the derivation list, so the bar's artifact does not predate this admission
#   registered  a row for this era governs; the caller re-reads it with baseline_lookup
#   owing       no row, but the witness says this silicon was judged in this era already
#   new         no row and no witness: genuine newness, BASELINE's only legitimate state
baseline_state() {
  local reg="$1" wit="$2" cpu="$3" crit="$4" era="$5" derived="$6" d dv=0 row
  [[ -n "$cpu" ]] || { printf 'nokey\n'; return 0; }
  # Substring, and in this direction: a probe string that gains a suffix must not silently
  # move a host out of the derivation set and into registry governance, changing its bar.
  while IFS= read -r d; do
    [[ -n "$d" && "$cpu" == *"$d"* ]] && dv=1
  done < <(printf '%s\n' "${derived//|/$'\n'}")
  row="$(baseline_lookup "$reg" "$cpu" "$crit" "$era")"
  if [[ "$dv" -eq 1 ]]; then
    if [[ -n "$row" ]]; then printf 'conflict\n'; else printf 'fleet\n'; fi
  elif [[ -n "$row" ]]; then printf 'registered\n'
  elif baseline_spent "$wit" "$cpu" "$era"; then printf 'owing\n'
  else printf 'new\n'
  fi
}

# baseline_candidate FILE FIELDS... — append one fully formed candidate row, tab-joined.
# The gate calls this and nothing else: it never opens the registry or the witness index
# for writing, because an instrument that mints the reference it will judge against has
# certified itself. Emitting a complete row (not a diff, not a reminder) is what makes the
# reviewed commit act a review rather than a re-derivation. Variadic because a BASELINE
# rendering proposes TWO rows into two files — the baseline and the witness that spends it
# — and they have different widths; one of them landing without the other is the state
# judged-runs.tsv describes, not a state this function may paper over.
baseline_candidate() {
  local file="$1"; shift
  mkdir -p "$(dirname "$file")" || return 1
  [[ -s "$file" ]] || printf '# candidate rows proposed by gate-p5; landing them is a reviewed commit act (#6)\n' >"$file"
  local IFS=$'\t'
  printf '%s\n' "$*" >>"$file"
}

# reconcile_sweep_best_ipf VALUE LOG — check a stated SWEEP_BEST_IPF against a live
# derivation rather than trusting it (#107, ruled 2026-08-18).
#
# The constant states "the best insns/FMA any emittable, zero-spill shape reaches",
# and tools/shapegen re-derives exactly that from a 140-shape audit in ~7s. It was
# wrong for the whole life of the threshold: 4.438 was attributed to a Permute shape
# needing 16 index vectors against the 15 SIMD values go1.26.x allocates (T10), so it
# named a kernel that cannot exist — and nothing could have caught that while the
# figure was only ever read. A number that can be re-derived and is not is a summary
# cache with its invalidation protocol sitting unused one directory over.
#
# This is the mint check, not the drift check. Both gates still state the constant
# independently (see each one's carried-from-P2 block) and both call this, so a
# divergence between the two copies stays visible where it was and neither copy is
# trusted on its own.
reconcile_sweep_best_ipf() {
  local stated="$1" log="$2" got n shape
  if ! GOEXPERIMENT=simd go run ./tools/shapegen -frontier >"$log" 2>&1; then
    sed 's/^/        /' "$log" | tail -10
    fail "shapegen -frontier stated no frontier, so SWEEP_BEST_IPF=$stated is unreconciled (#107)"
    return
  fi
  read -r got n shape <"$log"
  if [[ -z "$got" || -z "$shape" ]] || ! awk -v a="$got" 'BEGIN { exit !(a > 0) }'; then
    sed 's/^/        /' "$log" | tail -5
    fail "shapegen -frontier printed no usable figure ($(wc -l <"$log" | tr -d ' ') lines), so SWEEP_BEST_IPF=$stated is unreconciled (#107)"
  elif ! awk -v a="$got" -v b="$stated" 'BEGIN { exit !(sprintf("%.3f", a) == sprintf("%.3f", b)) }'; then
    fail "SWEEP_BEST_IPF=$stated but shapegen -frontier derives $got ($shape, best of $n emittable zero-spill shapes): criterion 5b would judge the shipped shape against a figure no shape reaches (#107)"
  else
    pass "SWEEP_BEST_IPF=$stated reconciles against shapegen -frontier: $got from $shape, best of $n emittable zero-spill shapes"
  fi
}
