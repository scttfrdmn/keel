#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Sapphire Rapids: does the AVX-512 deep KC have room beyond 1024? (#26, #142) The KC surface was still
# rising at the grid's edge, 1024, on SPR (archive/blocking-fma-sweep/, +11.9% at 8T), GNR and Skylake-SP.
# Four arms on the published rows: this tree with avx512KC = 1024 (shipped), 1536, 2048 and 3072,
# BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at n=4096, 1T and 8T, on a full-size c7i.48xlarge masked to one
# thread per physical core of socket 0. 4 -randlayout seeds x -test.count=5, arm order rotated per seed.
# Correctness on the instance first, for the furthest candidate (3072). Predictions: prereg.md beside this.
# Pattern: archive/blocking-skx-ab-f64/driver.sh.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/spr-kc-probe/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="spr:c7i.48xlarge:Intel Sapphire Rapids"
H=keel-spr D=~/.xtmp/sprkc HEADREV=$(git rev-parse --short HEAD) KS="1024 1536 2048 3072"
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/wt-*
for v in $KS; do
  git worktree add -q --detach $D/wt-$v $HEADREV || exit 1
  sed -i '' "s/^const avx512KC = 1024$/const avx512KC = $v/" $D/wt-$v/dispatch.go
  grep -q "^const avx512KC = $v$" $D/wt-$v/dispatch.go || { echo "!! candidate $v not applied"; exit 1; }
done
for s in 1 2 3 4; do for v in $KS; do
  (cd $D/wt-$v && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -ldflags="-randlayout=$s" -o $D/k$v-s$s.test ./bench/) || exit 1
done; done
(cd $D/wt-3072 && GOOS=linux GOARCH=amd64 GOEXPERIMENT=simd go test -c -o $D/k3072keel.test .) || exit 1
for v in $KS; do git worktree remove --force $D/wt-$v; done
echo "== built $(date -u +%T) rev $HEADREV (candidates $KS), $(go version | cut -d' ' -f3)"
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
timeout 120 ssh -o BatchMode=yes $H 'grep -m1 "model name" /proc/cpuinfo; nproc; lscpu | grep -E "Socket|NUMA node|L2|L3"; uptime' 2>&1
cpus8=$(timeout 60 ssh -o BatchMode=yes $H "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
echo "   mask (socket 0, one thread per physical core): $cpus8"
timeout 900 scp -q -o BatchMode=yes $D/*.test $H: || { echo "!! scp failed"; exit 1; }
echo "== correctness on the instance (avx512KC=3072), $(date -u +%T)"
timeout 3000 ssh -o BatchMode=yes $H "./k3072keel.test -test.run 'TestAVX512KC|Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k|P5Determinism' -test.count=1 >/dev/null 2>&1; echo rc=\$?" 2>&1 | sed 's/^/   /'
ARMS=(k1024 k1536 k2048 k3072)
{
for s in 1 2 3 4; do for k in 0 1 2 3; do a=${ARMS[$(( (k + s) % 4 ))]}
  timeout 3600 ssh -o BatchMode=yes $H "taskset -c $cpus8 ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 5" 2>&1 | grep -E '^Benchmark' | sed "s/^/$a s$s /"
done; done
} > $D/spr.txt
cat $D/spr.txt
echo "== done $(date -u +%T)"
