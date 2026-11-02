#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The float64 twin of the shipped issue-class KCDeep (#26, #142), on one c5n.18xlarge, stock go1.27.3,
# in one launch: stage A (BenchmarkBlocking64 grid with intervals at 1T and 8T) and stage B (the
# published float64 rows, BenchmarkLevel3F64 at n=4096, for main against branch exp-skx-kc64 built at
# three candidate block64.KCDeep values). Correctness on the instance first. Predictions: prereg.md.
# Pattern: archive/blocking-skx-ab/ and -ab-c/.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/blocking-skx-ab-f64/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP"
H=keel-skx D=~/.xtmp/blkf64 OLD=1ebbcd3 HEADREV=$(git rev-parse --short HEAD)
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/wt-*
git worktree add -q --detach $D/wt-old $OLD || exit 1
for v in 768 1024 1536; do
  git worktree add -q --detach $D/wt-$v $HEADREV || exit 1
  sed -i '' "s/^const issueClassKC64 = 1024$/const issueClassKC64 = $v/" $D/wt-$v/dispatch.go
  grep -q "^const issueClassKC64 = $v$" $D/wt-$v/dispatch.go || { echo "!! candidate $v not applied"; exit 1; }
done
for s in 1 2 3 4; do
  (cd $D/wt-old && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/old-s$s.test ./bench/) || exit 1
  for v in 768 1024 1536; do (cd $D/wt-$v && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/k$v-s$s.test ./bench/) || exit 1; done
  (cd $D/wt-old && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/blk64-s$s.test ./internal/block64/) || exit 1
done
(cd $D/wt-1024 && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -o $D/newkeel.test .) || exit 1
for w in old 768 1024 1536; do git worktree remove --force $D/wt-$w; done
echo "== built $(date -u +%T) branch $HEADREV (candidates 768/1024/1536), old $OLD, $(go version | cut -d' ' -f3)"
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
timeout 120 ssh -o BatchMode=yes $H 'grep -m1 "model name" /proc/cpuinfo; nproc; uptime' 2>&1
cpus8=$(timeout 60 ssh -o BatchMode=yes $H "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
echo "   mask (socket 0, one thread per physical core): $cpus8"
timeout 900 scp -q -o BatchMode=yes $D/*.test $H: || { echo "!! scp failed"; exit 1; }
echo "== correctness on the instance (branch, KCDeep64=1024), $(date -u +%T)"
timeout 3000 ssh -o BatchMode=yes $H "./newkeel.test -test.run 'TestIssueClassKC|Dgemm|Dsyrk|Dsymm|Dtrsm|Dtrmm|Dsyr2k|P5Determinism' -test.count=1 >/dev/null 2>&1; echo rc=\$?" 2>&1 | sed 's/^/   /'
G='KEEL_BLOCKING64_KC=512,768,1024,1536 KEEL_BLOCKING64_MC=144,288 KEEL_BLOCKING64_NC=2048'
{
for t in 1 8; do pin=2; [ $t = 8 ] && pin=$cpus8
  for s in 1 2 3 4; do
    timeout 3600 ssh -o BatchMode=yes $H "$G GOMAXPROCS=$t taskset -c $pin ./blk64-s$s.test -test.run XXX -test.bench 'Blocking64/' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/A t$t s$s /"
  done
done
ARMS=(old k768 k1024 k1536)
for s in 1 2 3 4; do for k in 0 1 2 3; do a=${ARMS[$(( (k + s) % 4 ))]}
  timeout 3600 ssh -o BatchMode=yes $H "taskset -c $cpus8 ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkLevel3F64\$' -test.benchtime 1s -test.count 3" 2>&1 | grep -E '^Benchmark' | sed "s/^/B $a s$s /"
done; done
} > $D/skx.txt
cat $D/skx.txt
echo "== done $(date -u +%T)"
