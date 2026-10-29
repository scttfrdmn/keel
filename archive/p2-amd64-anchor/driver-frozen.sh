#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The judged amd64 P2 re-run on the family-max AVX-512 ceiling with the roofline anchor (#199),
# both elements. Predictions: archive/p2-amd64-anchor/prereg.md, committed with this file,
# before launch. Pattern: archive/p2-f64-amd64/driver-frozen.sh, minus its OpenBLAS provisioning,
# which P2 does not use (the c9g precedent, archive/p2-arm64-c9g/).
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
SELF="archive/p2-amd64-anchor/driver-frozen.sh"
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
trap teardown EXIT
trap 'teardown; exit 143' INT TERM HUP

say "preflight (HEAD=$(git rev-parse --short HEAD), TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
set +e
GATE_RC=0
for e in f64 f32; do
  say "gate-p2, KEEL_ARCH=amd64 KEEL_ELEM=$e"
  KEEL_ARCH=amd64 KEEL_ELEM=$e timeout 7200 scripts/gate-p2.sh
  rc=$?
  say "gate-p2 amd64 $e exited $rc"
  [ "$rc" -eq 0 ] || GATE_RC=$rc
done
set -e
exit "$GATE_RC"
