#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Stage B of a Skylake-SP stock blocking default (#26, #142): main against branch exp-skx-kc (issue
# class blocks the float32 nest at KC=1024), on the published rows themselves: BenchmarkScale
# Sgemm/Ssyrk/Ssymm/Strsm at n=4096, 1 and 8 threads, one c5n.18xlarge, stock go1.27.3, pinned to
# gate-p5's mask (cores 0-7 of socket 0). 4 -randlayout seeds x -test.count=5 per arm, arms
# alternating. Correctness on the instance first: the branch's Level-3 suite with its exit status.
# Predictions: prereg.md beside this file. Pattern: archive/blocking-skx-ab/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/blocking-skx-ab-b/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP"
H=keel-skx D=~/.xtmp/blkabB OLD=b5ed22d
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/old
git worktree add -q --detach $D/old $OLD || exit 1
for s in 1 2 3 4; do
  (cd $D/old && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/old-s$s.test ./bench/) || exit 1
  (GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/new-s$s.test ./bench/) || exit 1
done
GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -o $D/newkeel.test . || exit 1
git worktree remove --force $D/old
echo "== built $(date -u +%T) new $(git rev-parse --short HEAD), old $OLD, $(go version | cut -d' ' -f3)"
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
echo "== correctness on the instance (branch), $(date -u +%T)"
timeout 3000 ssh -o BatchMode=yes $H "./newkeel.test -test.run 'TestIssueClassKC|Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k|P5Determinism' -test.count=1 -test.v 2>&1 | grep -E '^(--- FAIL|ok|FAIL|PASS)' | sort | uniq -c; ./newkeel.test -test.run 'TestIssueClassKC|Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k|P5Determinism' -test.count=1 >/dev/null 2>&1; echo rc=\$?" 2>&1 | sed 's/^/   /'
for s in 1 2 3 4; do
  if [ $((s % 2)) = 1 ]; then o="old new"; else o="new old"; fi
  for a in $o; do
    timeout 3600 ssh -o BatchMode=yes $H "taskset -c $cpus8 ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/$a s$s /"
  done
done > $D/skx.txt
cat $D/skx.txt
echo "== done $(date -u +%T)"
