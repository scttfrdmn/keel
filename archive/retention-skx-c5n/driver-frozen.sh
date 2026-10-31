#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The single-thread nest decomposition on Skylake-SP (#26, #142): scripts/retention.sh decompose
# and then feed, unchanged, on one c5n.18xlarge, the host behind #142's 77% retention. Moved here
# from janus because a burroughs build-group task shared janus for the whole of the first attempt
# (pueue's build group does not wait for measured, and this measurement is single-thread on a
# turbo-binned part). Predictions: archive/retention-skx-c5n/prereg.md, committed with this file.
# Pattern: archive/p5-f64/driver-frozen.sh, without the OpenBLAS provision (retention.sh
# cross-compiles its binary here and ships it; nothing host-side is built).
#
#   driver-frozen.sh TAG
set -euo pipefail
cd "$(dirname "$0")/../.."

TAG="${1:?usage: driver-frozen.sh TAG -- TAG labels the run}"
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_GOARCH=amd64
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP"
RATE="3.89"

SELF="archive/retention-skx-c5n/driver-frozen.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
[[ -z "${KEEL_REMOTE_HOSTS:-}" ]] || { echo "!! KEEL_REMOTE_HOSTS is set; the fleet's .keel-hosts must name the host; refusing"; exit 2; }

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — \$$RATE/hr"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list"
  # An unreadable inventory is UNKNOWN, never "down" (#199).
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

say "preflight (c5n.18xlarge, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
echo "   HEAD=$(git rev-parse --short HEAD), tree clean, KEEL_GOARCH=$KEEL_GOARCH"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
set +e
say "retention decompose — product is the archive"
timeout 5400 scripts/retention.sh decompose
DRC=$?
say "retention feed"
timeout 5400 scripts/retention.sh feed
FRC=$?
set -e
say "decompose exited $DRC, feed exited $FRC"
exit $(( DRC != 0 || FRC != 0 ))
