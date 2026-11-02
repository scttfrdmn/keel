#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Kernel4x32 with strict loop guards and a peeled last k-step (branch exp-kernel4x32-indexed,
# d77d9f8) against the shipped kernel (main, 5bcabca), each under stock go1.27.3 and under
# master@0d37007fda + CL 824624 PS2, on Zen 5. Every arm is built under 8 -randlayout seeds.
# Correctness on the instance comes first: internal/kern's oracle and scalar-tile tests and the
# root Level-3 sweeps, for the new kernel under both toolchains. Predictions: prereg.md beside
# this file. Pattern: archive/go-patches/cl824624/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/kernel4x32-strict/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="zen5:c8a.2xlarge:AMD EPYC Turin"
D=~/.xtmp/k4x32 H=keel-zen5 SEEDS="1 2 3 4 5 6 7 8" CL=~/.xtmp/go-ab824/cl/bin/go OLDREV=5bcabca
FILTER='Kernel/4x32/avx512/kc=128$|Peak/avx512$'
mkdir -p $D
rm -rf $D/old && git worktree add -q --detach $D/old $OLDREV || exit 1
gb() { # toolchain-go srcdir out [ldflags] -- build ./bench test binary for linux/amd64
  (cd "$2" && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd "$1" test -c ${4:+-ldflags="$4"} -o "$3" ./bench/) || exit 1
}
echo "== build $(date -u +%T) new $(git rev-parse --short HEAD), old $OLDREV"
for s in $SEEDS; do
  gb go "$D/old" $D/sold-s$s.test "-randlayout=$s"; gb go . $D/snew-s$s.test "-randlayout=$s"
  gb $CL "$D/old" $D/cold-s$s.test "-randlayout=$s"; gb $CL . $D/cnew-s$s.test "-randlayout=$s"
done
for t in s c; do G=go; [ $t = c ] && G=$CL
  GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $G test -c -o $D/${t}kern.test ./internal/kern/ || exit 1
  GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd $G test -c -o $D/${t}keel.test . || exit 1
done
echo "   stock $(go version | cut -d' ' -f3), cl $($CL version | cut -d' ' -f3)"
git worktree remove --force $D/old

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
timeout 900 scp -q -o BatchMode=yes $D/*.test $H: || { echo "!! scp failed"; exit 1; }

echo "== correctness on the instance (new kernel), $(date -u +%T)"
for t in s c; do
  timeout 1200 ssh -o BatchMode=yes $H "./${t}kern.test -test.run 'TestKernelsAgainstOracle|TestVectorAgainstScalarTile' -test.count=1 2>&1 | tail -2; ./${t}keel.test -test.run 'Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k|P5Determinism' -test.count=1 2>&1 | tail -2" 2>&1 | sed "s/^/   [$t] /"
done

one() { timeout 900 ssh -o BatchMode=yes $H "taskset -c 2 ./$1.test -test.run XXX -test.bench '$FILTER' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/r$2 $1 /"; }
ARMS=(sold snew cold cnew)
for i in 1 2 3 4 5; do
  for s in $SEEDS; do
    for k in 0 1 2 3; do one ${ARMS[$(( (k + i) % 4 ))]}-s$s $i; done
  done
done > $D/zen5.txt
cat $D/zen5.txt
echo "== done $(date -u +%T)"
