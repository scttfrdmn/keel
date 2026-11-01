#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #197 exploration tier: keel's AVX-512 kernels on Zen 5 under three toolchains: stock go1.27.3 (the
# shipped one), master@0d37007fda (base, CL 824624 PS2's own parent) and base + PS2 (cl,
# 7745cfc920). Each fork arm is built under 8 -randlayout seeds, so the seed spread is the layout
# floor any toolchain delta must clear (#61). Predictions: prereg.md beside this file. Pattern:
# archive/go-patches/cl827904/driver.sh, with the fail-closed teardown of the later campaigns.
#
#   driver.sh        (run from the keel root; toolchains from build-toolchains.sh)
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/go-patches/cl824624/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="zen5:c8a.2xlarge:AMD EPYC Turin"
D=~/.xtmp/go-ab824 H=keel-zen5 SEEDS="1 2 3 4 5 6 7 8"
FILTER='Kernel/(4x32|2x32)/avx512/kc=128$|Peak/avx512$'

echo "== build $(date -u +%T) keel $(git rev-parse --short HEAD)"
GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -o $D/stock.test ./bench/ || exit 1
echo "   stock: $(go version)"
for t in base cl; do
  for s in $SEEDS; do
    GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $D/$t/bin/go test -c -ldflags="-randlayout=$s" -o $D/$t-s$s.test ./bench/ || exit 1
  done
  echo "   $t: $($D/$t/bin/go version), seeds $SEEDS"
done

TORN=0
teardown() {
  [ "$TORN" = 1 ] && return; TORN=1
  echo "== TEARDOWN $(date -u +%T)"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED"
  local inv; inv="$(timeout 300 spawn list --state running --output json 2>/dev/null || true)"
  if ! jq -e 'type == "array"' >/dev/null 2>&1 <<<"$inv"; then echo "!! COULD NOT READ 'spawn list': fleet state UNKNOWN, assume BILLING"
  elif jq -e 'any(.[]; .name | startswith("keel-"))' >/dev/null <<<"$inv"; then echo "!! A keel- INSTANCE IS STILL BILLING"
  else echo "   no keel- instance running — fleet is down"; fi
}
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
echo "== launch $(date -u +%T)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
echo "== $H read-back"
timeout 120 ssh -o BatchMode=yes $H 'grep -m1 "model name" /proc/cpuinfo; nproc; uptime' 2>&1
timeout 600 scp -q -o BatchMode=yes $D/stock.test $D/base-s*.test $D/cl-s*.test $H: || { echo "!! scp failed"; exit 1; }

one() { # binary round
  timeout 900 ssh -o BatchMode=yes $H "taskset -c 2 ./$1.test -test.run XXX -test.bench '$FILTER' -test.benchtime 1s" 2>&1 \
    | grep -E '^Benchmark' | sed "s/^/r$2 $1 /"
}
for i in 1 2 3 4 5; do
  one stock $i
  if [ $((i % 2)) = 1 ]; then o="base cl"; else o="cl base"; fi
  for s in $SEEDS; do for t in $o; do one $t-s$s $i; done; done
done > $D/zen5.txt
cat $D/zen5.txt
echo "== done $(date -u +%T)"
