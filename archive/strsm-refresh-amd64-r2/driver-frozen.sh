#!/usr/bin/env bash
# Strsm refresh, one judged run. #37's fix made Strsm ~39% faster on AVX-512
# (archive/trsm-amd64-37/, dev tier), so the README's Strsm rows are stale on all
# five hosts and criterion 9 reds on them. This run's PRODUCT IS THE ARCHIVE, not a
# verdict: see build/prereg-37-judged.md, written before launch, which predicts RED
# on criterion 9's Strsm rows and PASS/BASELINE everywhere else.
#
# ARCH is the only parameter. N=2 is the published standard and means two runs per
# arch, because pooling two runs is what sees between-run drift and one run's
# internal CI cannot (#141). Each AWS run launches NEW instances, so a pair is two
# independent draws on different physical hosts rather than two windows on one.
#
# Fleet READ OFF THE ARCHIVES, never recalled: amd64 = c5n.18xlarge/c7a.48xlarge/
# c8a.48xlarge (keel-skx/zen4/zen5, era ceil8's own types); arm64 = c7g.16xlarge/
# c8g.48xlarge (keel-gvt3/gvt4).
#
# TTL IS 4h and the banner below prints the variable, not a literal: the earlier
# driver hardcoded "6h" in a say string while 7h was in force. The measured count=10
# amd64 run is 1h41m, so 4h is ~2.4x headroom and caps the worst case near $96.
set -euo pipefail
cd "$(dirname "$0")/.."

# NO BRACES IN THESE MESSAGES. An unescaped `}` inside ${var:?msg} terminates the
# expansion, so `${1:?usage: ... {amd64|arm64} TAG}` set ARCH to "amd64 TAG}" and
# the case below refused a perfectly good argument. Cost nothing because arg
# validation sits above every billing step, which is the ordering rule here: a
# driver must refuse bad input before it can rent anything.
ARCH="${1:?usage: strsm-refresh.sh ARCH TAG -- ARCH is amd64 or arm64}"
TAG="${2:?usage: strsm-refresh.sh ARCH TAG -- TAG labels the archive}"
# Run 1 was at 628c38a; run 2 is at 852ad55, which adds only the errcheck fix in
# tools/benchci (a package ./bench does not import). The measured binary's
# dependency-closure hash is identical at both revs (0da19ff8...), so the pair
# still pools. The arm64 ceil8 pair is the precedent for two revs in one pair.
REV_EXPECT=852ad55

say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws
export AWS_REGION=us-east-1
export KEEL_FLEET_MARKET=on-demand
export KEEL_FLEET_TTL=4h

case "$ARCH" in
amd64)
  KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
  RATE="24.09" ;;
arm64)
  KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2"
  RATE="10.35" ;;
*) echo "!! ARCH must be amd64 or arm64" >&2; exit 2 ;;
esac
export KEEL_FLEET

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — \$$RATE/hr, so this matters"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list, not this script's bookkeeping"
  if timeout 300 spawn list --state running 2>&1 | grep -i 'keel-'; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
}
trap teardown EXIT INT TERM

say "preflight ($ARCH, tag=$TAG, TTL=$KEEL_FLEET_TTL, rate=\$$RATE/hr)"
REV="$(git rev-parse --short HEAD)"
echo "   HEAD=$REV (expected $REV_EXPECT)"
[[ "$REV" == "$REV_EXPECT" ]] || { echo "!! HEAD moved; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
echo "   tree clean"
echo "   era: $(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv')"
say "co-tenant census at launch — nothing of ours should already be billing"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

say "launch"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision"
timeout 10800 scripts/provision-openblas.sh --yes

say "gate-p5 ($ARCH, era ceil8) — product is the archive, RED on criterion 9 is PREDICTED"
set +e
timeout 21600 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"
say "post-run census"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
exit "$GATE_RC"
