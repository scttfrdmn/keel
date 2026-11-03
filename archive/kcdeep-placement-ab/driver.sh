#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Why did AVX-512 KCDeep gain 2-4 points more at 8T on the judged 48xlarge Zen hosts
# (archive/p5-zen-kcdeep/) than on the 2xlarge stage B (archive/blocking-avx512-b/)? (#26, #142)
# Three arms on c8a.48xlarge + c7a.48xlarge, stock go1.27.3:
#   R = c0c373d, the README's old Zen rev; O = 9bf3bdc, pre-KCDeep main; N = this tree (Go code == 35c2894).
# BenchmarkScale Sgemm/Ssyrk/Ssymm at n=4096, 1T and 8T, under two 8-core masks on socket 0:
#   spread = the first core of each of 8 distinct L3 domains (gate-p5's placement);
#   packed = 8 cores sharing cpu 0's L3 domain (the 2xlarge's situation).
# 4 -randlayout seeds x -test.count=5 per arm per mask, arm order rotated per seed.
# Predictions: prereg.md beside this file. Pattern: archive/blocking-avx512-b/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/kcdeep-placement-ab/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="zen5:c8a.48xlarge:AMD EPYC Turin
zen4:c7a.48xlarge:AMD EPYC Genoa"
HOSTS="keel-zen5 keel-zen4" D=~/.xtmp/kcplace
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/wt-*
for a in R:c0c373d O:9bf3bdc; do git worktree add -q --detach $D/wt-${a%%:*} ${a#*:} || exit 1; done
for s in 1 2 3 4; do
  for a in R O; do (cd $D/wt-$a && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/$a-s$s.test ./bench/) || exit 1; done
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/N-s$s.test ./bench/) || exit 1
done
for a in R O; do git worktree remove --force $D/wt-$a; done
echo "== built $(date -u +%T) N=$(git rev-parse --short HEAD) O=9bf3bdc R=c0c373d, $(go version | cut -d' ' -f3)"
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
host() {
  local h=$1 spread packed s m a o
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "model name" /proc/cpuinfo; nproc; lscpu | grep -E "L3|NUMA node0"; uptime' 2>&1
  # lscpu -p CACHE is L1d:L1i:L2:L3; field 4 of it is the L3 id.
  spread=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,SOCKET,CACHE | grep -v '^#' | awk -F, '\$2==0{split(\$3,c,\":\"); if(!s[c[4]]++) print \$1}' | head -8 | paste -sd, -")
  packed=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,SOCKET,CACHE | grep -v '^#' | awk -F, '\$2==0{split(\$3,c,\":\"); if(c[4]==l3||NR==1){l3=c[4]; print \$1}}' | head -8 | paste -sd, -")
  echo "   spread mask: $spread"; echo "   packed mask: $packed"
  for v in "$spread" "$packed"; do
    [ "$(tr ',' '\n' <<<"$v" | wc -l | tr -d ' ')" = 8 ] || { echo "!! $h: a mask has fewer than 8 cpus ($v); host UNMEASURED"; return; }
  done
  timeout 900 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  for s in 1 2 3 4; do
    case $s in 1) o="R O N";; 2) o="O N R";; 3) o="N R O";; 4) o="R N O";; esac
    for m in spread packed; do
      [ $m = spread ] && mask=$spread || mask=$packed
      for a in $o; do
        timeout 3600 ssh -o BatchMode=yes $h "taskset -c $mask ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$/^(Sgemm|Ssyrk|Ssymm)\$' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h $m $a s$s /"
      done
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
