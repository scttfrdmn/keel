#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# NEON MR×NR sweep, #136 step 4. CHARACTERIZATION ONLY — no judged verdicts, no bars, no
# registry (the arm64 judged tier awaits #73/#119). It builds the arm64 NEON binary here,
# ships it, and runs BenchmarkKernel on one GB10 host through the pueue `measured` group.
# The shapes come from kern.Measured() = 8x8, 4x16 (non-spilling) + 8x12, 8x16, 4x32
# (referenceTiles, spilling), so the sweep grades the -S audit's fit/spill verdicts against
# the clock. Pre-registration: docs/neon-sweep.md step 4.
#
# The quietness guard is a plain 1-minute-load bound, not the pedestal-subtracted form: the
# pedestal subtracts the DRIVER's own contribution to the measured host's load, and here the
# driver runs on the dev host while the work runs on GB10 — the driver is not a co-tenant, so
# there is nothing to subtract and the pedestal form degenerates to the bound. pueue serializes
# the measured slot against other projects regardless.
set -uo pipefail
cd /Users/scttfrdmn/src/keel || exit 3
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
HOST="${HOST:-pollux.local}"; export KEEL_REMOTE_HOSTS="$HOST"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.27.0}"
export KEEL_GOARCH=arm64
COUNT="${COUNT:-20}"; BTIME="${BTIME:-1s}"; LOADMAX="${LOADMAX:-4.0}"
say() { printf '\n=== %s ===\n' "$*"; }

say "provenance — characterization on GB10, no verdicts, no bars (#136 step 4)"
date -u +%FT%TZ
echo "rev: $(git rev-parse HEAD)"
git diff --quiet HEAD -- && echo "tree clean: yes" || echo "tree clean: NO"
prov="$(remote_probe "$HOST")"
[ -n "$prov" ] || { echo "REFUSED: $HOST did not answer a provenance probe"; exit 4; }
echo "$prov"
case "$prov" in *governor=performance*) echo "governor: performance OK (§5 rule 5)" ;;
  *) echo "REFUSED: $HOST governor is not performance (§5 rule 5)"; exit 5 ;; esac

say "quietness — plain 1-min load bound (driver is not a co-tenant here; pedestal degenerates)"
la="$(ssh "${KEEL_SSH_OPTS[@]}" "$HOST" 'cut -d" " -f1 /proc/loadavg' 2>/dev/null | tr -d ' \n')"
[ -n "$la" ] || { echo "REFUSED: no /proc/loadavg from $HOST"; exit 6; }
if awk -v l="$la" -v m="$LOADMAX" 'BEGIN{exit !(l+0 < m+0)}'; then echo "quiet: 1-min load $la < $LOADMAX"
else echo "REFUSED: 1-min load $la not under $LOADMAX; relaunch when quiet"; exit 6; fi

say "build the arm64 NEON binary (-trimpath), toolchain read back off the artifact"
BIN="$(mktemp -d)/bench.test"
remote_build_test ./bench "$BIN" || { echo "REFUSED: arm64 build failed"; exit 7; }
echo "binary: $(build_settings "$BIN")"
builder_toolchain "$BIN"

say "sweep — BenchmarkKernel, kern.Measured() neon shapes, GOMAXPROCS=1, count=$COUNT, via pueue measured"
KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$HOST" "$BIN" \
  -test.run=NONE -test.bench=BenchmarkKernel -test.count="$COUNT" -test.benchtime="$BTIME"
rc=$?
echo "REMOTE_STATE=$REMOTE_STATE REMOTE_QUEUE=${REMOTE_QUEUE:-} REMOTE_GROUP=${REMOTE_GROUP:-} rc=$rc"

say "done"
date -u +%FT%TZ
