#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-p5 on c9g (Neoverse-V3, Graviton5) in arm64 era famax (#196): its own campaign, founding
# and confirmation runs, so it gets README rows of its own. V3 is not in CEIL_DERIVED_FROM, so it
# is registry-governed and renders BASELINE on both runs; its rows land over N=2 afterwards, as
# famaxx86's did. Predictions: archive/p5-arm64-c9g/prereg.md, committed with this file.
#
#   driver-frozen.sh TIER TAG     TIER is explore (c9g.4xlarge, the whole path, no verdict) or
#                                 judged (c9g.48xlarge); TAG labels the run.
#
# KEEL_GOARCH is exported here because gate-p5 defaults to amd64 (archive/p5-arm64-famax/).
# KEEL_OPENBLAS_VERSION=v0.3.34 by Scott's 2026-10-27 ruling on #196: the reference is the newest
# OpenBLAS whose detection names the core, and 0.3.29 picks generic armv8sve on V3.
# Re-frozen for judged r1, after the exploration pass at be72f31; SELF below requires it.
set -euo pipefail
cd "$(dirname "$0")/../.."

# No braces in these messages: an unescaped closing brace ends a ${var:?msg} expansion.
TIER="${1:?usage: driver-frozen.sh TIER TAG -- TIER is explore or judged}"
TAG="${2:?usage: driver-frozen.sh TIER TAG -- TAG labels the run}"

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_GOARCH=arm64 KEEL_OPENBLAS_VERSION=v0.3.34
case "$TIER" in
explore)
  KEEL_FLEET="gvt5:c9g.4xlarge:Neoverse-V3"
  RATE="0.70" ;;
judged)
  KEEL_FLEET="gvt5:c9g.48xlarge:Neoverse-V3"
  RATE="8.35" ;;
*) echo "!! TIER must be explore or judged" >&2; exit 2 ;;
esac
export KEEL_FLEET

# HEAD must be the commit that added this driver: a driver cannot name its own hash, so it
# checks that nothing has landed since it was committed.
SELF="archive/p5-arm64-c9g/driver-frozen.sh"
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
  # An unreadable inventory is UNKNOWN, never "down": a reboot left this fleet billing ~2h40m
  # behind a grep that matched nothing because 'spawn list' itself had failed (#199).
  local inv
  inv="$(timeout 300 spawn list --state running --output json 2>/dev/null || true)"
  if ! jq -e 'type == "array"' >/dev/null 2>&1 <<<"$inv"; then
    echo "!! COULD NOT READ 'spawn list': fleet state UNKNOWN, assume it is BILLING. Check by hand."
  elif jq -e 'any(.[]; .name | startswith("keel-"))' >/dev/null <<<"$inv"; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
}
trap teardown EXIT
trap 'teardown; exit 143' INT TERM HUP

say "preflight (arm64 c9g, tier=$TIER, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
echo "   HEAD=$(git rev-parse --short HEAD), tree clean, KEEL_GOARCH=$KEEL_GOARCH, OpenBLAS $KEEL_OPENBLAS_VERSION, era: $ERA"
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
