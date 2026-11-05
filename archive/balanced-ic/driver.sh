#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #206's pre-registered A/B: main (O = a716d34) against branch exp-balanced-ic (N), whose nest shrinks
# MC per call so the ic units fill whole rounds of workers. BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at
# n=4096, 1T and 8T, on seven judged-size hosts, stock go1.27.3:
#   arm64  V1 c7g.16xlarge, V2 c8g.48xlarge, V3 c9g.48xlarge
#   amd64  Zen 5 c8a.48xlarge, Zen 4 c7a.48xlarge, Skylake-SP c5n.18xlarge, SPR c7i.48xlarge
# Mask: gate-p5's placement, one core per L3 domain on socket 0 where it has >= 8 domains (spread),
# else one thread on each of socket 0's first 8 physical cores. 4 -randlayout seeds x -test.count=5 per
# arm, arms alternating. Correctness on each host first: the branch's balance, determinism and Level-3
# tests. Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/balanced-ic/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2
gvt5:c9g.48xlarge:Neoverse-V3
zen5:c8a.48xlarge:AMD EPYC Turin
zen4:c7a.48xlarge:AMD EPYC Genoa
skx:c5n.18xlarge:Intel Skylake-SP
spr:c7i.48xlarge:Intel Sapphire Rapids"
HOSTS="keel-gvt3 keel-gvt4 keel-gvt5 keel-zen5 keel-zen4 keel-skx keel-spr" D=~/.xtmp/balic OLD=a716d34
mkdir -p $D && rm -rf $D/arm64 $D/amd64 $D/*.txt $D/old
git worktree add -q --detach $D/old $OLD || exit 1
for arch in arm64 amd64; do
  mkdir -p $D/$arch; B=(GOOS=linux GOARCH=$arch GOEXPERIMENT=simd)
  for s in 1 2 3 4; do
    (cd $D/old && env "${B[@]}" go test -c -ldflags="-randlayout=$s" -o $D/$arch/O-s$s.test ./bench/) || exit 1
    env "${B[@]}" go test -c -ldflags="-randlayout=$s" -o $D/$arch/N-s$s.test ./bench/ || exit 1
  done
  env "${B[@]}" go test -c -o $D/$arch/keel.test . && env "${B[@]}" go test -c -o $D/$arch/block.test ./internal/block/ && env "${B[@]}" go test -c -o $D/$arch/block64.test ./internal/block64/ || exit 1
done
git worktree remove --force $D/old
echo "== built $(date -u +%T) N=$(git rev-parse --short HEAD) O=$OLD, $(go version | cut -d' ' -f3)"
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
  local h=$1 arch spread packed mask s a o
  case $h in keel-gvt*) arch=arm64;; *) arch=amd64;; esac
  echo "== $h read-back ($arch)"
  timeout 120 ssh -o BatchMode=yes $h 'uname -m; grep -m1 -iE "model name|CPU part" /proc/cpuinfo; nproc; lscpu | grep -E "Model name|L3"; uptime' 2>&1
  # lscpu -p CACHE is L1d:L1i:L2:L3; field 4 of it is the L3 id.
  spread=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,SOCKET,CACHE | grep -v '^#' | awk -F, '\$2==0{split(\$3,c,\":\"); if(!s[c[4]]++) print \$1}' | head -8 | paste -sd, -")
  packed=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
  if [ "$(tr ',' '\n' <<<"$spread" | wc -l | tr -d ' ')" = 8 ]; then mask=$spread; echo "   mask spread: $mask"; else mask=$packed; echo "   mask packed: $mask"; fi
  [ "$(tr ',' '\n' <<<"$mask" | wc -l | tr -d ' ')" = 8 ] || { echo "!! $h: mask has fewer than 8 cpus; UNMEASURED"; return; }
  timeout 1800 scp -q -o BatchMode=yes $D/$arch/*.test $h: || { echo "!! $h scp failed"; return; }
  echo "== $h correctness (branch), $(date -u +%T)"
  timeout 3000 ssh -o BatchMode=yes $h "./block.test -test.run 'TestBalanceMC|TestPlanKCDeep' -test.count=1 >/dev/null 2>&1; echo block rc=\$?; ./block64.test -test.run 'ParallelIsSerial' -test.count=1 >/dev/null 2>&1; echo block64 rc=\$?; ./keel.test -test.run 'P5Determinism|Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k' -test.count=1 >/dev/null 2>&1; echo keel rc=\$?" 2>&1 | sed 's/^/   /'
  for s in 1 2 3 4; do
    if [ $((s % 2)) = 1 ]; then o="O N"; else o="N O"; fi
    for a in $o; do
      timeout 3600 ssh -o BatchMode=yes $h "taskset -c $mask ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h $a s$s /"
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
