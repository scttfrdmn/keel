#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-p5 at float64 (KEEL_ELEM=f64) in era f64init (#193): the exploration pass, the founding
# run and the confirmation run of the float64 performance gates, one fleet per architecture.
# Predictions: archive/p5-f64/prereg.md, committed with this file, before launch. Pattern:
# archive/p5-amd64-famaxx86/driver-frozen.sh and archive/p5-arm64-c9g/driver-frozen.sh.
#
#   driver-frozen.sh ARCH TIER TAG   ARCH is amd64 or arm64; TIER is explore (4xlarge-class, the
#                                    whole path, no verdict) or judged; TAG labels the run.
#
# KEEL_GOARCH and KEEL_ELEM are exported here, not inherited, so the era and the element this run
# resolves are properties of the driver and not of a default. The two architectures run one after
# the other, never at once: aws-fleet.sh up refuses a keel- instance its fleet does not name.
# arm64 uses OpenBLAS v0.3.34 on all three Graviton generations: Scott's 2026-10-27 ruling on #196
# made the reference the newest OpenBLAS whose detection names the core, and float64 has no
# earlier arm64 reference to stay comparable with.
set -euo pipefail
cd "$(dirname "$0")/../.."

# No braces in these messages: an unescaped closing brace ends a ${var:?msg} expansion.
ARCH="${1:?usage: driver-frozen.sh ARCH TIER TAG -- ARCH is amd64 or arm64}"
TIER="${2:?usage: driver-frozen.sh ARCH TIER TAG -- TIER is explore or judged}"
TAG="${3:?usage: driver-frozen.sh ARCH TIER TAG -- TAG labels the run}"

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_ELEM=f64
case "$ARCH/$TIER" in
amd64/explore)
  KEEL_FLEET="skx:c5n.4xlarge:Intel Skylake-SP
zen4:c7a.2xlarge:AMD EPYC Genoa
zen5:c8a.2xlarge:AMD EPYC Turin"
  RATE="1.71" ;;
amd64/judged)
  KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
  RATE="24.09" ;;
arm64/explore)
  KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:Neoverse-V3"
  RATE="1.97" ;;
arm64/judged)
  KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2
gvt5:c9g.48xlarge:Neoverse-V3"
  RATE="18.70" ;;
*) echo "!! ARCH must be amd64 or arm64 and TIER explore or judged" >&2; exit 2 ;;
esac
export KEEL_FLEET KEEL_GOARCH="$ARCH"
[[ "$ARCH" == arm64 ]] && export KEEL_OPENBLAS_VERSION=v0.3.34

# HEAD must be the commit that added this driver: a driver cannot name its own hash, so it
# checks that nothing has landed since it was committed.
SELF="archive/p5-f64/driver-frozen.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
ERA="$(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras-f64.tsv "$1"' _ "$ARCH")"
[[ "$ERA" == f64init ]] || { echo "!! $ARCH resolves float64 era '$ERA', not f64init; refusing"; exit 2; }

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — \$$RATE/hr"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list"
  # An unreadable inventory is UNKNOWN, never "down": a reboot left a fleet billing ~2h40m
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

say "preflight ($ARCH float64, tier=$TIER, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
echo "   HEAD=$(git rev-parse --short HEAD), tree clean, KEEL_GOARCH=$KEEL_GOARCH, KEEL_ELEM=$KEEL_ELEM, OpenBLAS ${KEEL_OPENBLAS_VERSION:-default}, era: $ERA"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision"
timeout 10800 scripts/provision-openblas.sh --yes
say "gate-p5 ($ARCH float64, era $ERA, tier $TIER) — product is the archive"
set +e
timeout 21600 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"
exit "$GATE_RC"
