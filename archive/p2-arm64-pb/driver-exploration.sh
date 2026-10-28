#!/bin/bash
# Exploration-tier pre-check of gate-p2's arm64 arms on the promoted pre-broadcast kernels
# (#198, ddf6e2d): the whole remote path with the new names, before the judged run.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:AWS Graviton5"
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"; timeout 900 scripts/aws-fleet.sh down; timeout 300 spawn list --state running 2>&1 | grep -i keel- && echo "!! STILL BILLING" || echo "   fleet is down"; }
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
echo "== launch $(date -u +%T) rev $(git rev-parse --short HEAD)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
for e in f64 f32; do echo "######## arm64 $e"; KEEL_ARCH=arm64 KEEL_ELEM=$e timeout 3600 scripts/gate-p2.sh; echo "######## arm64 $e rc=$?"; done
