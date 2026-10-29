#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-p5 on amd64 in era famaxx86 (#199): the runs that re-derive the README's amd64 rows
# after the AVX-512 ceiling became the family max (x44) and the roofline's pin moved to the
# anchor. Predictions: archive/p5-amd64-famaxx86/prereg.md, committed with this file, before
# launch. Pattern: archive/strsm-refresh-amd64-r2/driver-frozen.sh.
#
#   driver-frozen.sh TIER TAG     TIER is explore (4xlarge, the whole path, no verdict) or
#                                 judged (c5n.18xlarge + c7a.48xlarge + c8a.48xlarge); TAG labels the run.
#
# KEEL_GOARCH IS EXPORTED HERE, not inherited. Eras are per-architecture since #200, and
# gate-p5 defaults to amd64 already; exporting it anyway makes the era this run resolves a
# property of the driver, not of the default (archive/p5-arm64-famax/ explains why).
set -euo pipefail
cd "$(dirname "$0")/../.."

# No braces in these messages: an unescaped closing brace ends a ${var:?msg} expansion.
TIER="${1:?usage: driver-frozen.sh TIER TAG -- TIER is explore or judged}"
TAG="${2:?usage: driver-frozen.sh TIER TAG -- TAG labels the run}"

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_GOARCH=amd64
case "$TIER" in
explore)
  KEEL_FLEET="skx:c5n.4xlarge:Intel Skylake-SP
zen4:c7a.2xlarge:AMD EPYC Genoa
zen5:c8a.2xlarge:AMD EPYC Turin"
  RATE="1.71" ;;
judged)
  KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
  RATE="24.09" ;;
*) echo "!! TIER must be explore or judged" >&2; exit 2 ;;
esac
export KEEL_FLEET

# HEAD must be the commit that added this driver: a driver cannot name its own hash, so it
# checks that nothing has landed since it was committed.
SELF="archive/p5-amd64-famaxx86/driver-frozen.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
ERA="$(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv amd64')"
[[ "$ERA" == famaxx86 ]] || { echo "!! amd64 resolves era '$ERA', not famaxx86; refusing"; exit 2; }

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

say "preflight (amd64, tier=$TIER, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
echo "   HEAD=$(git rev-parse --short HEAD), tree clean, KEEL_GOARCH=$KEEL_GOARCH, era: $ERA"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision"
timeout 10800 scripts/provision-openblas.sh --yes
say "gate-p5 (amd64, era $ERA, tier $TIER) — product is the archive"
set +e
timeout 21600 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"
exit "$GATE_RC"
