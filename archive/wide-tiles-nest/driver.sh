#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The follow-ups to archive/wide-tiles-fleet/, branch exp-wide-tiles: (1) the blocked GEMM with 3x64
# on Zen 4, Skylake-SP, Sapphire Rapids and Granite Rapids, single-thread, against each host's
# shipped shapes; (2) all five hosts at 8 threads, pinned to 8 physical cores; (3) on Zen 5, the
# kernel at kc=128 against kc=512, the L1-overflow test of 3x64's KC curve. Toolchains: stock
# go1.27.3 (anchor) and master@0d37007fda + CL 824624 PS2. Correctness first on every host.
# Predictions: prereg.md beside this file. Pattern: archive/wide-tiles-fleet/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/wide-tiles-nest/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="zen5:c8a.2xlarge:AMD EPYC Turin
zen4:c7a.2xlarge:AMD EPYC Genoa
skx:c5n.4xlarge:Intel Skylake-SP
spr:c7i.4xlarge:Intel Sapphire Rapids
gnr:c8i.4xlarge:Intel Granite Rapids"
HOSTS="keel-zen5 keel-zen4 keel-skx keel-spr keel-gnr"
D=~/.xtmp/widenest CL=~/.xtmp/go-ab824/cl/bin/go
mkdir -p $D && rm -f $D/*.test $D/*.txt
echo "== build $(date -u +%T) rev $(git rev-parse --short HEAD)"
for s in 1 2 3 4; do
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $CL test -c -ldflags="-randlayout=$s" -o $D/bps2-s$s.test ./internal/block/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/bstk-s$s.test ./internal/block/) || exit 1
done
for s in 1 2 3 4 5 6 7 8; do
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $CL test -c -ldflags="-randlayout=$s" -o $D/kps2-s$s.test ./bench/) || exit 1
done
GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $CL test -c -o $D/cblk.test ./internal/block/ || exit 1
GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -o $D/sblk.test ./internal/block/ || exit 1
echo "   stock $(go version | cut -d' ' -f3), cl $($CL version | cut -d' ' -f3)"

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

G1='KEEL_BLOCKING_KC=128,192,256,384 KEEL_BLOCKING_MC=72,144,288 KEEL_BLOCKING_NC=2048'
G8='KEEL_BLOCKING_KC=128,384 KEEL_BLOCKING_MC=144,288 KEEL_BLOCKING_NC=2048'
GS='KEEL_BLOCKING_KC=384 KEEL_BLOCKING_MC=144 KEEL_BLOCKING_NC=2048'
F='Blocking/(2x32|4x32|3x64)/avx512/'
host() {
  local h=$1 cpus8 tag
  echo "== $h read-back"; timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "model name" /proc/cpuinfo; nproc; lscpu | grep -E "L1d|L2"; uptime' 2>&1
  cpus8=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,CORE | grep -v '^#' | awk -F, '!s[\$2]++{print \$1}' | head -8 | paste -sd, -")
  echo "   8-thread mask (one thread per physical core): $cpus8"
  local files="$D/bps2-s*.test $D/bstk-s*.test $D/cblk.test $D/sblk.test"
  [ $h = keel-zen5 ] && files="$files $D/kps2-s*.test"
  timeout 900 scp -q -o BatchMode=yes $files $h: || { echo "!! $h scp failed"; return; }
  for t in s c; do
    timeout 1500 ssh -o BatchMode=yes $h "./${t}blk.test -test.run TestMeasuredKernelsGemmAgainstOracle -test.count=1 >/dev/null 2>&1; echo blk-rc=\$?" 2>&1 | sed "s/^/   [$h $t] /"
  done
  run() { # tag env procs pin binary filter
    timeout 2400 ssh -o BatchMode=yes $h "$2 GOMAXPROCS=$3 taskset -c $4 ./$5 -test.run XXX -test.bench '$6' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h $1 /"
  }
  for i in 1 2 3; do for s in 1 2 3 4; do
    [ $h != keel-zen5 ] && run "r$i t1-ps2-s$s" "$G1" 1 2 bps2-s$s.test "$F"
    [ $h != keel-zen5 ] && run "r$i t1-stk-s$s" "$GS" 1 2 bstk-s$s.test 'Blocking/(2x32|4x32)/avx512/'
    run "r$i t8-ps2-s$s" "$G8" 8 "$cpus8" bps2-s$s.test "$F"
    run "r$i t8-stk-s$s" "$GS" 8 "$cpus8" bstk-s$s.test 'Blocking/(2x32|4x32)/avx512/'
  done; done
  if [ $h = keel-zen5 ]; then
    for i in 1 2 3; do for s in 1 2 3 4 5 6 7 8; do
      run "r$i kc-ps2-s$s" "" 1 2 kps2-s$s.test 'Kernel/(4x32|3x64)/avx512/kc=(128|512)$'
    done; done
  fi
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
