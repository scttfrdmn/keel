#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Graviton 4 (Neoverse V2) and Graviton 5 (Neoverse V3), judged sizes c8g.48xlarge and c9g.48xlarge,
# stock go1.27.3, masked to one core each of socket 0's first 8 physical cores. Two questions:
#  A. arm64 blocking at the published size. The pre-broadcast kernels' KCPB/MCPB = 384/72 were chosen
#     single-thread at n=2048 (archive/neon-pb/). Nine builds of this tree, KCPB {256,384,512} x MCPB
#     {36,72,144}, on BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm n=4096 at 1T and 8T, 4 -randlayout seeds x
#     -test.count=3, arm order rotated per seed.
#  B. Is V3 load-bound? BenchmarkLoadProbe (internal/vec/loadprobe_arm64_test.go: the shipped
#     KernelPB5x16 with A loads, B loads, or both hoisted out of the K-loop) at kc 128/384/512, plus
#     BenchmarkPeak/neon (the family-max ceiling) and the archived 3x24 BenchmarkLimiterProbe as the link
#     to archive/neon-raw/. 1T on core 2, -test.count=10.
# Correctness on each instance first: the Level-3 suite of the shipped arm. Predictions: prereg.md here.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/v3-load/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h KEEL_GOARCH=arm64
export KEEL_FLEET="gvt4:c8g.48xlarge:Neoverse-V2
gvt5:c9g.48xlarge:Neoverse-V3"
HOSTS="keel-gvt4 keel-gvt5" D=~/.xtmp/v3load HEADREV=$(git rev-parse --short HEAD)
ARMS=(); for k in 256 384 512; do for m in 36 72 144; do ARMS+=(k${k}m${m}); done; done
mkdir -p $D && rm -rf $D/*.test $D/*.txt $D/wt-*
B=(GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd)
for a in "${ARMS[@]}"; do
  k=${a#k}; k=${k%m*}; m=${a#*m}
  git worktree add -q --detach $D/wt-$a $HEADREV || exit 1
  sed -i '' -e "s/^	KCPB = 384$/	KCPB = $k/" -e "s/^	MCPB = 72$/	MCPB = $m/" $D/wt-$a/internal/block/block.go
  grep -q "^	KCPB = $k$" $D/wt-$a/internal/block/block.go && grep -q "^	MCPB = $m$" $D/wt-$a/internal/block/block.go || { echo "!! arm $a not applied"; exit 1; }
  for s in 1 2 3 4; do (cd $D/wt-$a && env "${B[@]}" go test -c -ldflags="-randlayout=$s" -o $D/$a-s$s.test ./bench/) || exit 1; done
done
(cd $D/wt-k384m72 && env "${B[@]}" go test -c -o $D/keel.test . && env "${B[@]}" go test -c -o $D/vec.test ./internal/vec/) || exit 1
for a in "${ARMS[@]}"; do git worktree remove --force $D/wt-$a; done
echo "== built $(date -u +%T) rev $HEADREV, arms ${ARMS[*]}, $(go version | cut -d' ' -f3)"
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
  local h=$1 cpus8 s k a
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m1 -iE "model name|CPU part" /proc/cpuinfo; nproc; lscpu | grep -E "Model name|Socket|NUMA node|L1d|L2|L3"; uptime' 2>&1
  cpus8=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
  echo "   mask (socket 0, one thread per physical core): $cpus8"
  [ "$(tr ',' '\n' <<<"$cpus8" | wc -l | tr -d ' ')" = 8 ] || { echo "!! $h: mask has fewer than 8 cpus; UNMEASURED"; return; }
  timeout 1800 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  echo "== $h correctness (shipped arm), $(date -u +%T)"
  timeout 3000 ssh -o BatchMode=yes $h "./keel.test -test.run 'Sgemm|Ssyrk|Ssymm|Strsm|Strmm|Ssyr2k|P5Determinism' -test.count=1 >/dev/null 2>&1; echo rc=\$?" 2>&1 | sed 's/^/   /'
  echo "== $h B: load probe, $(date -u +%T)"
  timeout 1800 ssh -o BatchMode=yes $h "taskset -c 2 ./vec.test -test.run XXX -test.bench '^Benchmark(LoadProbe|LimiterProbe)\$' -test.benchtime 1s -test.count 10" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h B vec /"
  timeout 1800 ssh -o BatchMode=yes $h "taskset -c 2 ./k384m72-s1.test -test.run XXX -test.bench '^BenchmarkPeak\$/neon' -test.benchtime 1s -test.count 10" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h B peak /"
  echo "== $h A: blocking at n=4096, $(date -u +%T)"
  for s in 1 2 3 4; do for k in $(seq 0 8); do a=${ARMS[$(( (k + 2*s) % 9 ))]}
    timeout 3600 ssh -o BatchMode=yes $h "taskset -c $cpus8 ./$a-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 3" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h A $a s$s /"
  done; done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
