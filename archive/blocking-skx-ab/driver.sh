#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Stage A of shipping a Skylake-SP blocking default on stock Go (#26, #142): the archive/blocking-skx-c5n/
# lead (kc=768/mc=288, +7.0% single-thread on medians) re-measured WITH intervals, at 1 and 8 threads,
# and past the old grid's edge, on the host it came from. BenchmarkBlocking, unchanged, on main
# (no code change), stock go1.27.3, 4 -randlayout seeds × -test.count=5 per point, pooled.
# Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/blocking-skx-ab/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP"
H=keel-skx D=~/.xtmp/blkab
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
timeout 120 ssh -o BatchMode=yes $H 'grep -m1 "model name" /proc/cpuinfo; nproc; lscpu | grep -E "L1d|L2|L3|NUMA node0"; uptime' 2>&1
cpus8=$(timeout 60 ssh -o BatchMode=yes $H "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
echo "   8-thread mask (socket 0, one thread per physical core): $cpus8"
timeout 900 scp -q -o BatchMode=yes $D/blk-s*.test $H: || { echo "!! scp failed"; exit 1; }
G='KEEL_BLOCKING_KC=384,512,768,1024 KEEL_BLOCKING_MC=144,288,432 KEEL_BLOCKING_NC=2048'
for t in 1 8; do
  pin=2; [ $t = 8 ] && pin=$cpus8
  for s in 1 2 3 4; do
    timeout 3600 ssh -o BatchMode=yes $H "$G GOMAXPROCS=$t taskset -c $pin ./blk-s$s.test -test.run XXX -test.bench 'Blocking/(2x32|4x32)/avx512/' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/t$t s$s /"
  done
done > $D/skx.txt
cat $D/skx.txt
echo "== done $(date -u +%T)"
