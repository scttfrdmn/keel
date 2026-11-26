#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #163 / #197: re-take the arm64 broadcast-fold A/B on the commit that actually landed. CL 827904 was
# merged, reverted (CL 866866) and relanded as CL 866091 (master 544d1b015d, 2026-11-25) with a changed
# rule set, so archive/go-patches/cl827904/'s numbers do not transfer. Toolchains, built from ~/src/go
# worktrees: base = master@b22ece6bf5 (the fold's parent), fold = master@544d1b015d; one commit apart.
# This keel tree for both arms. Exploration tier (#197 guardrail: patched-toolchain numbers are never a
# published row): c7g, c8g, c9g 4xlarge, core 2, 10 rounds, arm order alternating by round.
#   vec:   LimiterProbe (shim 3x24, raw 3x24 ctl, PB, FO) and LoadProbe/Ctl (the shipped 5x16-pb, whose
#          loop is byte-unchanged by the fold: 213 insns both arms, the control);
#   block: BenchmarkBlocking 2048^3 single-thread at KC=384, MC in {72,144}, every arm64 kernel.
# Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/go-patches/cl866091/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h GOTOOLCHAIN=local
export KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:Neoverse-V3"
HOSTS="keel-gvt3 keel-gvt4 keel-gvt5" D=~/.xtmp/cl866091
mkdir -p $D && rm -f $D/*.test $D/*.txt
for a in base:b22ece6bf5 fold:544d1b015d; do
  n=${a%%:*}; G=~/.xtmp/go$n/bin/go
  v=$($G version); [[ "$v" == *"${a#*:}"* ]] || { echo "!! $n toolchain is '$v', not ${a#*:}"; exit 2; }
  echo "== $n: $v"
  (GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd $G test -c -o $D/vec-$n.test ./internal/vec/ && GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd $G test -c -o $D/blk-$n.test ./internal/block/) || exit 1
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
host() {
  local h=$1 i t o
  echo "== $h read-back"; timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "CPU part" /proc/cpuinfo; nproc; uptime' 2>&1
  timeout 600 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  for i in $(seq 1 10); do
    if [ $((i % 2)) = 1 ]; then o="base fold"; else o="fold base"; fi
    for t in $o; do
      timeout 900 ssh -o BatchMode=yes $h "taskset -c 2 ./vec-$t.test -test.run XXX -test.bench '^Benchmark(LimiterProbe/(shim|ctl|PB|FO)/|LoadProbe/Ctl/)' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i $t /"
      timeout 900 ssh -o BatchMode=yes $h "KEEL_BLOCKING_KC=384 KEEL_BLOCKING_MC=72,144 KEEL_BLOCKING_NC=2048 GOMAXPROCS=1 taskset -c 2 ./blk-$t.test -test.run XXX -test.bench '^BenchmarkBlocking\$' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i $t /"
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
