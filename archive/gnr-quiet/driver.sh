#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# A quieter Granite Rapids re-measure (#26, #142). On c8i.4xlarge the KC x MC sweep's 1T intervals
# were +/-8-10% and its 8T kc=256 lead was non-monotone (archive/blocking-fma-sweep/), so GNR read
# inconclusive. This repeats that sweep and stage B's published-row A/B (archive/blocking-avx512-b/) on
# a full-size c8i.48xlarge: no small-instance co-tenancy, more samples, and both placements where the
# part has 8+ L3 domains (archive/kcdeep-placement-ab/'s lesson).
#   sweep: BenchmarkBlocking 4x32 at 2048^3, KC {256,384,512,768,1024} x MC {72,144,288}, 1T and 8T,
#          4 -randlayout seeds x -test.count=5.
#   A/B:   O = 9bf3bdc (pre-KCDeep) vs N = this tree, BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm n=4096,
#          4 seeds x -test.count=5, arms alternating.
# Masks: packed = one thread on each of socket 0's first 8 physical cores; spread = the first core of
# each of 8 distinct L3 domains, run only if the part has 8 of them. Predictions: prereg.md beside this.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/gnr-quiet/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="gnr:c8i.48xlarge:Intel Granite Rapids"
H=keel-gnr D=~/.xtmp/gnrq
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/wt-O
git worktree add -q --detach $D/wt-O 9bf3bdc || exit 1
for s in 1 2 3 4; do
  (cd $D/wt-O && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/O-s$s.test ./bench/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/N-s$s.test ./bench/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/blk-s$s.test ./internal/block/) || exit 1
done
git worktree remove --force $D/wt-O
echo "== built $(date -u +%T) N=$(git rev-parse --short HEAD) O=9bf3bdc, $(go version | cut -d' ' -f3)"
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
timeout 120 ssh -o BatchMode=yes $H 'grep -m1 "model name" /proc/cpuinfo; nproc; lscpu | grep -E "Socket|NUMA node|L3"; uptime' 2>&1
# lscpu -p CACHE is L1d:L1i:L2:L3; field 4 of it is the L3 id.
packed=$(timeout 60 ssh -o BatchMode=yes $H "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
spread=$(timeout 60 ssh -o BatchMode=yes $H "lscpu -p=CPU,SOCKET,CACHE | grep -v '^#' | awk -F, '\$2==0{split(\$3,c,\":\"); if(!s[c[4]]++) print \$1}' | head -8 | paste -sd, -")
echo "   packed mask: $packed"; echo "   spread mask: $spread"
[ "$(tr ',' '\n' <<<"$packed" | wc -l | tr -d ' ')" = 8 ] || { echo "!! packed mask has fewer than 8 cpus; UNMEASURED"; exit 1; }
MASKS="packed"
if [ "$(tr ',' '\n' <<<"$spread" | wc -l | tr -d ' ')" = 8 ]; then MASKS="packed spread"; else echo "   fewer than 8 L3 domains on socket 0: spread is not a distinct placement here, packed only"; fi
timeout 900 scp -q -o BatchMode=yes $D/*.test $H: || { echo "!! scp failed"; exit 1; }
G='KEEL_BLOCKING_KC=256,384,512,768,1024 KEEL_BLOCKING_MC=72,144,288 KEEL_BLOCKING_NC=2048'
{
  for s in 1 2 3 4; do
    timeout 3600 ssh -o BatchMode=yes $H "$G GOMAXPROCS=1 taskset -c 2 ./blk-s$s.test -test.run XXX -test.bench 'Blocking/4x32/avx512/' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/sweep t1 one s$s /"
    for m in $MASKS; do [ $m = packed ] && mask=$packed || mask=$spread
      timeout 3600 ssh -o BatchMode=yes $H "$G GOMAXPROCS=8 taskset -c $mask ./blk-s$s.test -test.run XXX -test.bench 'Blocking/4x32/avx512/' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/sweep t8 $m s$s /"
    done
  done
  for s in 1 2 3 4; do
    if [ $((s % 2)) = 1 ]; then o="O N"; else o="N O"; fi
    for m in $MASKS; do [ $m = packed ] && mask=$packed || mask=$spread
      for a in $o; do
        timeout 3600 ssh -o BatchMode=yes $H "taskset -c $mask ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/ab $m $a s$s /"
      done
    done
  done
} > $D/gnr.txt
cat $D/gnr.txt
echo "== done $(date -u +%T)"
