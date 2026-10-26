#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The judged float64 P2 run, amd64 (#191). Predictions are in prereg.md beside this
# file, committed with it, before launch. The pattern follows
# archive/strsm-refresh-amd64-r2/driver-frozen.sh: arg-free, fleet read off the
# archives, teardown on EXIT verified against the launcher's own list, and every
# step timed out.
set -euo pipefail
cd "$(dirname "$0")/../.."
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
RATE="24.09"

# HEAD must be the commit that added this driver: a driver cannot name its own hash,
# so it checks that nothing has landed since it was committed.
SELF="archive/p2-f64-amd64/driver-frozen.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — \$$RATE/hr"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  if timeout 300 spawn list --state running 2>&1 | grep -i 'keel-'; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
}
trap teardown EXIT INT TERM

say "preflight (HEAD=$(git rev-parse --short HEAD), TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision"
timeout 7200 scripts/provision-openblas.sh --yes
say "gate-p2, KEEL_ELEM=f64"
set +e
KEEL_ELEM=f64 timeout 7200 scripts/gate-p2.sh
GATE_RC=$?
set -e
say "gate-p2 f64 exited $GATE_RC"
exit "$GATE_RC"
