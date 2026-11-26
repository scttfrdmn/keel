#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The kernel-level rerun for archive/go-patches/cl866091/ (#163, #197). driver.sh's vec filter put `/`
# inside an alternation, which `go test -bench` splits per sub-benchmark level, so no vec row ran. This
# re-takes only that half with one regex per level, positive-controlled locally on both toolchains
# (7 rows each: LimiterProbe shim/ctl/PB/FO at kc=512, LoadProbe Ctl at kc=128/384/512) and smoked on
# a linux/arm64 lab host before launch. Same toolchains (base = master@b22ece6bf5, fold = master@
# 544d1b015d), this keel tree, c7g/c8g/c9g 4xlarge, core 2, 10 rounds, arm order alternating.
# A host whose round 1 yields fewer than 7 rows per arm stops with "!!" (fail-fast, not a silent hole).
# Predictions: prereg.md (predictions 1 and 2, unchanged) and its "kernel rerun" addendum.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/go-patches/cl866091/driver-vec.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h GOTOOLCHAIN=local
export KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:Neoverse-V3"
HOSTS="keel-gvt3 keel-gvt4 keel-gvt5" D=~/.xtmp/cl866091v
mkdir -p $D && rm -f $D/*.test $D/*.txt $D/*.rows
for a in base:b22ece6bf5 fold:544d1b015d; do
  n=${a%%:*}; G=~/.xtmp/go$n/bin/go
  v=$($G version); [[ "$v" == *"${a#*:}"* ]] || { echo "!! $n toolchain is '$v', not ${a#*:}"; exit 2; }
  echo "== $n: $v"
  GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd $G test -c -o $D/vec-$n.test ./internal/vec/ || exit 1
done
echo "== built $(date -u +%T) keel $(git rev-parse --short HEAD)"
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
P1='^BenchmarkLimiterProbe$/^(shim|ctl|PB|FO)$'
P2='^BenchmarkLoadProbe$/^Ctl$'
host() {
  local h=$1 i t o n
  echo "== $h read-back"; timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "CPU part" /proc/cpuinfo; nproc; uptime' 2>&1
  timeout 600 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  for i in $(seq 1 10); do
    if [ $((i % 2)) = 1 ]; then o="base fold"; else o="fold base"; fi
    for t in $o; do
      n=0
      for P in "$P1" "$P2"; do
        n=$((n + $(timeout 900 ssh -o BatchMode=yes $h "taskset -c 2 ./vec-$t.test -test.run XXX -test.bench '$P' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i $t /" | tee -a $D/$h.rows | wc -l)))
      done
      if [ $i = 1 ] && [ $n -lt 7 ]; then echo "!! $h round 1 arm $t produced $n rows, want 7; stopping this host"; return; fi
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt $D/keel-*.rows 2>/dev/null
echo "== done $(date -u +%T)"
