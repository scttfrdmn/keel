#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The judged arm64 P2 re-run against the family-max NEON denominator, V1/V2/V3, both
# elements (#198). Predictions: archive/p2-arm64-family/prereg.md, committed with this
# file, before launch. The pattern follows
# archive/strsm-refresh-amd64-r2/driver-frozen.sh: arg-free, fleet read off the
# archives, teardown on EXIT verified against the launcher's own list, and every
# step timed out.
set -euo pipefail
cd "$(dirname "$0")/../.."
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2
gvt5:c9g.48xlarge:Neoverse-V3"
RATE="18.70"

# HEAD must be the commit that added this driver: a driver cannot name its own hash,
# so it checks that nothing has landed since it was committed.
SELF="archive/p2-arm64-family/driver-frozen.sh"
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
# No provisioning step: P2 runs cross-compiled test and bench binaries and uses no
# OpenBLAS. The first launch (ee5dac1) ran provision-openblas.sh out of the gate-p5
# driver pattern, and it refused this host: OpenBLAS 0.3.29 picks the generic armv8sve
# kernel on Neoverse-V3, which is off the reference allowlist
# (VOID-provision-refused-ee5dac1.log). That matters to gate-p3/p5, not to P2.
set +e
GATE_RC=0
for e in f64 f32; do
  say "gate-p2, KEEL_ARCH=arm64 KEEL_ELEM=$e"
  KEEL_ARCH=arm64 KEEL_ELEM=$e timeout 7200 scripts/gate-p2.sh
  rc=$?
  say "gate-p2 arm64 $e exited $rc"
  [ "$rc" -eq 0 ] || GATE_RC=$rc
done
set -e
exit "$GATE_RC"
