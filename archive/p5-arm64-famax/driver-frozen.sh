#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-p5 on arm64 in era famax (#200): the runs that re-derive the README's Neoverse rows
# after #198 changed both the ceiling instrument (family-max neonPeakTile) and the
# dispatched kernel (5x16-pb/neon). Predictions: archive/p5-arm64-famax/prereg.md, committed
# with this file, before launch. Pattern: archive/strsm-refresh-amd64-r2/driver-frozen.sh.
#
#   driver-frozen.sh TIER TAG     TIER is explore (4xlarge, the whole path, no verdict) or
#                                 judged (c7g.16xlarge + c8g.48xlarge); TAG labels the run.
#
# KEEL_GOARCH IS EXPORTED HERE, not inherited. Eras are per-architecture since #200, and
# gate-p5 defaults to amd64, so a run that inherited no KEEL_GOARCH would judge arm64 hosts
# in amd64's era. The ceil8 r1 archive holds a VOID-wrong-goarch log from that dependency.
# V1 + V2 only: c9g (V3) needs OpenBLAS v0.3.34 first (#196), and has no README rows yet.
# Re-frozen 2026-10-28 after the exploration pass, for the fixes in 30af10d; the SELF check
# below requires the freezing commit to be HEAD.
set -euo pipefail
cd "$(dirname "$0")/../.."

# No braces in these messages: an unescaped closing brace ends a ${var:?msg} expansion.
TIER="${1:?usage: driver-frozen.sh TIER TAG -- TIER is explore or judged}"
TAG="${2:?usage: driver-frozen.sh TIER TAG -- TAG labels the run}"

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_GOARCH=arm64
case "$TIER" in
explore)
  KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2"
  RATE="1.27" ;;
judged)
  KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2"
  RATE="10.35" ;;
*) echo "!! TIER must be explore or judged" >&2; exit 2 ;;
esac
export KEEL_FLEET

# HEAD must be the commit that added this driver: a driver cannot name its own hash, so it
# checks that nothing has landed since it was committed.
SELF="archive/p5-arm64-famax/driver-frozen.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
ERA="$(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv arm64')"
[[ "$ERA" == famax ]] || { echo "!! arm64 resolves era '$ERA', not famax; refusing"; exit 2; }

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — \$$RATE/hr"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list"
  if timeout 300 spawn list --state running 2>&1 | grep -i 'keel-'; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
}
trap teardown EXIT
trap 'teardown; exit 143' INT TERM HUP

say "preflight (arm64, tier=$TIER, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
echo "   HEAD=$(git rev-parse --short HEAD), tree clean, KEEL_GOARCH=$KEEL_GOARCH, era: $ERA"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision"
timeout 10800 scripts/provision-openblas.sh --yes
say "gate-p5 (arm64, era $ERA, tier $TIER) — product is the archive"
set +e
timeout 21600 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"
exit "$GATE_RC"
