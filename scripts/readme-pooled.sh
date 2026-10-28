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
# With no argument each host is pooled from ITS architecture's current era (column 7 of the
# manifest, eras being scoped per architecture since #200); pass an era to reduce that one
# era for every host instead.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/bench.sh
source scripts/bench.sh
# shellcheck source=scripts/gate-lib.sh
source scripts/gate-lib.sh

MAN=scripts/readme-archives.tsv
[[ -r "$MAN" ]] || { echo "readme-pooled: cannot read $MAN" >&2; exit 2; }
ERA_AMD64="${1:-$(era_current scripts/measurement-eras.tsv amd64)}"
ERA_ARM64="${1:-$(era_current scripts/measurement-eras.tsv arm64)}"
[[ -n "$ERA_AMD64" && -n "$ERA_ARM64" ]] || { echo "readme-pooled: an architecture resolves no era from scripts/measurement-eras.tsv" >&2; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
n=0
# A row is in scope iff its era is its own architecture's era; a row with no arch column is
# refused rather than guessed at.
inscope='/^#/ || $1=="cpu_model" { next } NF>=7 && $6=="canonical" && (($7=="amd64" && $2==ea) || ($7=="arm64" && $2==eb))'
awk -F'\t' -v ea="$ERA_AMD64" -v eb="$ERA_ARM64" "$inscope"' { print $1 }' "$MAN" \
  | sort -u | while IFS= read -r cpu; do
  read -r -a arch <<<"$(awk -F'\t' -v c="$cpu" -v ea="$ERA_AMD64" -v eb="$ERA_ARM64" \
    "$inscope"' && $1==c { printf "%s ", $5 }' "$MAN")"
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
