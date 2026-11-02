#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The 12-accumulator 3x64 kernel on five x86 µarchs, and through the blocked GEMM on Zen 5.
# Branch exp-wide-tiles. Toolchains: stock go1.27.3 and master@0d37007fda + CL 824624 PS2.
# Correctness on every host first: internal/kern's oracle and scalar-tile tests and
# internal/block's oracle check of every measured kernel through the blocked Gemm, under
# both toolchains, with exit status kept. Predictions: prereg.md beside this file.
# Pattern: archive/wide-tiles-zen5/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/wide-tiles-fleet/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="zen5:c8a.2xlarge:AMD EPYC Turin
zen4:c7a.2xlarge:AMD EPYC Genoa
skx:c5n.4xlarge:Intel Skylake-SP
spr:c7i.4xlarge:Intel Sapphire Rapids
gnr:c8i.4xlarge:Intel Granite Rapids"
HOSTS="keel-zen5 keel-zen4 keel-skx keel-spr keel-gnr"
D=~/.xtmp/widefleet SEEDS="1 2 3 4 5 6 7 8" CL=~/.xtmp/go-ab824/cl/bin/go
KFILTER='Kernel/(2x32|4x32|3x64)/avx512/kc=128$|Peak/avx512$'
mkdir -p $D && rm -f $D/*.test $D/*.txt
echo "== build $(date -u +%T) rev $(git rev-parse --short HEAD)"
for s in $SEEDS; do
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/stk-s$s.test ./bench/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $CL test -c -ldflags="-randlayout=$s" -o $D/ps2-s$s.test ./bench/) || exit 1
done
for s in 1 2 3 4; do
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $CL test -c -ldflags="-randlayout=$s" -o $D/bps2-s$s.test ./internal/block/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/bstk-s$s.test ./internal/block/) || exit 1
done
for t in s c; do G=go; [ $t = c ] && G=$CL
  GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $G test -c -o $D/${t}kern.test ./internal/kern/ || exit 1
  GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $G test -c -o $D/${t}blk.test ./internal/block/ || exit 1
done
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

host() { # host: read-back, copy, correctness, kernel A/B; zen5 also the nest
  local h=$1
  echo "== $h read-back"; timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "model name" /proc/cpuinfo; nproc; uptime' 2>&1
  local files="$D/stk-s*.test $D/ps2-s*.test $D/skern.test $D/ckern.test $D/sblk.test $D/cblk.test"
  [ $h = keel-zen5 ] && files="$files $D/bps2-s*.test $D/bstk-s*.test"
  timeout 900 scp -q -o BatchMode=yes $files $h: || { echo "!! $h scp failed"; return; }
  for t in s c; do
    timeout 1500 ssh -o BatchMode=yes $h "./${t}kern.test -test.run 'TestKernelsAgainstOracle|TestVectorAgainstScalarTile' -test.count=1 -test.v 2>&1 | grep -cE '^    --- PASS.*3x64'; ./${t}kern.test -test.run 'TestKernelsAgainstOracle|TestVectorAgainstScalarTile' -test.count=1 >/dev/null 2>&1; echo kern-rc=\$?; ./${t}blk.test -test.run TestMeasuredKernelsGemmAgainstOracle -test.count=1 -test.v 2>&1 | grep -cE '^    --- PASS.*3x64'; ./${t}blk.test -test.run TestMeasuredKernelsGemmAgainstOracle -test.count=1 >/dev/null 2>&1; echo blk-rc=\$?" 2>&1 | sed "s/^/   [$h $t] /"
  done
  local one ARMS=(stk ps2)
  for i in 1 2 3 4 5; do for s in $SEEDS; do for k in 0 1; do
    a=${ARMS[$(( (k + i) % 2 ))]}
    timeout 900 ssh -o BatchMode=yes $h "taskset -c 2 ./$a-s$s.test -test.run XXX -test.bench '$KFILTER' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i $a-s$s /"
  done; done; done
  if [ $h = keel-zen5 ]; then
    for i in 1 2 3; do for s in 1 2 3 4; do
      timeout 1800 ssh -o BatchMode=yes $h "KEEL_BLOCKING_KC=128,192,256,384 KEEL_BLOCKING_MC=72,144,288 KEEL_BLOCKING_NC=2048 GOMAXPROCS=1 taskset -c 2 ./bps2-s$s.test -test.run XXX -test.bench 'Blocking/(4x32|3x64)/avx512/' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i nest-ps2-s$s /"
      timeout 900 ssh -o BatchMode=yes $h "KEEL_BLOCKING_KC=384 KEEL_BLOCKING_MC=144 KEEL_BLOCKING_NC=2048 GOMAXPROCS=1 taskset -c 2 ./bstk-s$s.test -test.run XXX -test.bench 'Blocking/4x32/avx512/' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i nest-stk-s$s /"
    done; done
  fi
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
