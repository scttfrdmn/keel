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
# visible. This file existing is not an argument for moving them into it.

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
