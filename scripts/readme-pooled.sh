#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# readme-pooled.sh — one benchstat estimator per published cell, for scripts/readme-numbers.sh.
#
# Emits `cpu_model <TAB> benchmark <TAB> median <TAB> lo <TAB> hi <TAB> N` for every GFLOP/s row
# in each published host's POOLED canonical sample archives, keyed by the benchmark name exactly
# as tools/benchci prints it, so the caller looks a cell up by constructing that name.
#
# THE CORPUS IS THE ONE THE GATE USES. scripts/readme-archives.tsv names it, keyed (cpu_model,
# era) with a `status` column, and gate-p5.sh readme_ref_archives reads the same rows for
# criterion 9 reference interval. Published and compared are therefore the same number computed
# the same way, which is what #165 asked for. A `superseded` row is never pooled: era pinned8
# archived two sample sets per host at 6ba6566 differing only in a timestamp, and the wrong one
# was reached for by hand on 2026-10-09.
#
# Era defaults to the current one; pass an era to reduce a different one.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/bench.sh
source scripts/bench.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

MAN=scripts/readme-archives.tsv
ERA="${1:-$(era_current scripts/measurement-eras.tsv)}"
[[ -n "$ERA" ]] || { echo "readme-pooled: no era resolves from scripts/measurement-eras.tsv" >&2; exit 2; }
[[ -r "$MAN" ]] || { echo "readme-pooled: cannot read $MAN" >&2; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
n=0
awk -F'\t' -v e="$ERA" '/^#/ || $1=="cpu_model" { next } NF>=6 && $2==e && $6=="canonical" { print $1 }' "$MAN" \
  | sort -u | while IFS= read -r cpu; do
  read -r -a arch <<<"$(awk -F'\t' -v c="$cpu" -v e="$ERA" \
    '/^#/ || $1=="cpu_model" { next } NF>=6 && $1==c && $2==e && $6=="canonical" { printf "%s ", $5 }' "$MAN")"
  [[ "${#arch[@]}" -gt 0 ]] || continue
  n=$((n + 1))
  cat "${arch[@]}" > "$TMP/pool$n.txt"
  go run ./tools/benchci "$TMP/pool$n.txt" > "$TMP/pool$n.csv" 2>/dev/null || continue
  # Columns 2/4/5 of benchci output are median/lo/hi; the `-N` suffix benchstat appends to a
  # name is stripped, the same way bench_stat strips it, so a constructed lookup key matches.
  awk -F, -v cpu="$cpu" -v nar="${#arch[@]}" '
    /^,/ { unit = $2; next }
    unit == "GFLOP/s" && $1 != "" {
      name = $1; sub(/-[0-9]+$/, "", name)
      printf "%s\t%s\t%s\t%s\t%s\t%s\n", cpu, name, $2, $4, $5, nar
    }' "$TMP/pool$n.csv"
done
