#!/bin/bash
# Exploration-tier gate-p2 amd64 pre-check on the family-max AVX-512 ceiling (#199, c98cd9f).
# Predictions: archive/p2-amd64-family/prereg.md.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="skx:c5n.4xlarge:Intel Skylake-SP
zen4:c7a.2xlarge:AMD EPYC Genoa
zen5:c8a.2xlarge:AMD EPYC Turin"
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"; timeout 900 scripts/aws-fleet.sh down; timeout 300 spawn list --state running 2>&1 | grep -i keel- && echo "!! STILL BILLING" || echo "   fleet is down"; }
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
[ -z "$(git status --porcelain)" ] || { echo "!! tree dirty"; TORN=1; exit 2; }
echo "== launch $(date -u +%T) rev $(git rev-parse --short HEAD)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
for e in f64 f32; do echo "######## amd64 $e"; KEEL_ARCH=amd64 KEEL_ELEM=$e timeout 3600 scripts/gate-p2.sh; echo "######## amd64 $e rc=$?"; done
