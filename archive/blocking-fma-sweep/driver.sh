#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Is there a KC/MC lever on the FMA-class hosts, as there was on Skylake-SP (archive/blocking-skx-ab*/)?
# BenchmarkBlocking at 2048³ for the dispatched 4x32 (and 2x32 for reference), stock go1.27.3, main,
# no code change, on Zen 5, Zen 4, Sapphire Rapids and Granite Rapids, at 1T (core 2) and 8T (8
# physical cores). 4 -randlayout seeds x -test.count=3 per point, pooled by tools/benchci.
# Predictions: prereg.md beside this file. Pattern: archive/blocking-skx-ab/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/blocking-fma-sweep/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="zen5:c8a.2xlarge:AMD EPYC Turin
zen4:c7a.2xlarge:AMD EPYC Genoa
spr:c7i.4xlarge:Intel Sapphire Rapids
gnr:c8i.4xlarge:Intel Granite Rapids"
HOSTS="keel-zen5 keel-zen4 keel-spr keel-gnr" D=~/.xtmp/fmasweep
mkdir -p $D && rm -f $D/*.test $D/*.txt
for s in 1 2 3 4; do (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/blk-s$s.test ./internal/block/) || exit 1; done
echo "== built $(date -u +%T) rev $(git rev-parse --short HEAD), $(go version | cut -d' ' -f3)"
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
G='KEEL_BLOCKING_KC=256,384,512,768,1024 KEEL_BLOCKING_MC=72,144,288 KEEL_BLOCKING_NC=2048'
host() {
  local h=$1 cpus8
  echo "== $h read-back"; timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "model name" /proc/cpuinfo; nproc; uptime' 2>&1
  cpus8=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,CORE | grep -v '^#' | awk -F, '!s[\$2]++{print \$1}' | head -8 | paste -sd, -")
  echo "   8-thread mask: $cpus8"
  timeout 900 scp -q -o BatchMode=yes $D/blk-s*.test $h: || { echo "!! $h scp failed"; return; }
  for t in 1 8; do pin=2; [ $t = 8 ] && pin=$cpus8
    for s in 1 2 3 4; do
      timeout 3600 ssh -o BatchMode=yes $h "$G GOMAXPROCS=$t taskset -c $pin ./blk-s$s.test -test.run XXX -test.bench 'Blocking/(2x32|4x32)/avx512/' -test.benchtime 1s -test.count 3" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h t$t s$s /"
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
