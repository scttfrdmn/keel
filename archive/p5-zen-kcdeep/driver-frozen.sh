#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# gate-p5 on Zen 5 and Zen 4 alone (c8a.48xlarge, c7a.48xlarge), era famaxx86: the judged pair that
# re-derives the README's Zen rows after every AVX-512 host began blocking full-height float32 GEMMs at
# KC=1024 (block.KCDeep, #26/#142, CHANGELOG [Unreleased], archive/blocking-avx512-b/). Skylake-SP's rows
# were already re-derived under KCDeep (archive/p5-skx-kcdeep/) and its dispatch is unchanged since, so it
# is not re-run. The instrument is unchanged, so the era is too. Predictions: archive/p5-zen-kcdeep/
# prereg.md, committed with this file, before launch.
#
#   driver-frozen.sh TAG     TAG labels the run (r1, r2)
#
set -euo pipefail
cd "$(dirname "$0")/../.."

# No braces in these messages: an unescaped closing brace ends a ${var:?msg} expansion.
TAG="${1:?usage: driver-frozen.sh TAG -- TAG labels the run}"
TIER=judged

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=4h
export KEEL_GOARCH=amd64
KEEL_FLEET="zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
RATE="20.20"
export KEEL_FLEET

# HEAD must be the commit that added this driver: a driver cannot name its own hash, so it
# checks that nothing has landed since it was committed.
SELF="archive/p5-zen-kcdeep/driver-frozen.sh"
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

say "preflight (amd64 zen-only, tier=$TIER, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
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
