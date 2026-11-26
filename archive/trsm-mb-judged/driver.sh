#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #37's two unmet done-when criteria: the diagonal solves' time share measured ON THE GATE HOSTS, and
# MB chosen from that measurement (both were dev-host only; status comment 2026-10-21). BenchmarkTrsmMB's
# full and solve-only arms (internal/block/nest_bench_test.go), the published shape n=4096, both sides,
# MB in {32,48,64,96,128,192,256}, each host's dispatched kernel only, 1T (core 2) and 8T (gate-p5's
# placement: one core per L3 domain where socket 0 has >= 8, else socket 0's first 8 physical cores),
# -test.count=4. Six README hosts at judged sizes, stock go1.27.4, this tree. Filters positive-controlled
# locally and smoked on ceres (4x32/avx512), janus (2x32/avx512) and pollux (5x16-pb/neon) via pueue: 28 of
# 28 rows each. A host whose first pass yields fewer than 28 rows stops with "!!". Predictions: prereg.md.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/trsm-mb-judged/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen5:c8a.48xlarge:AMD EPYC Turin
zen4:c7a.48xlarge:AMD EPYC Genoa
gvt3:c7g.16xlarge:Neoverse-V1
gvt4:c8g.48xlarge:Neoverse-V2
gvt5:c9g.48xlarge:Neoverse-V3"
HOSTS="keel-skx keel-zen5 keel-zen4 keel-gvt3 keel-gvt4 keel-gvt5" D=~/.xtmp/trsmmb
mkdir -p $D && rm -f $D/*.test $D/*.txt $D/*.rows
for arch in amd64 arm64; do
  GOOS=linux GOARCH=$arch GOEXPERIMENT=simd go test -c -o $D/blk-$arch.test ./internal/block/ || exit 1
  GOOS=linux GOARCH=$arch GOEXPERIMENT=simd go test -c -o $D/keel-$arch.test . || exit 1
done
echo "== built $(date -u +%T) rev $(git rev-parse --short HEAD), $(go version | cut -d' ' -f3)"
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
  local h=$1 arch k b spread packed mask c n t pin
  case $h in keel-gvt*) arch=arm64 k=5x16-pb b=neon;; keel-skx) arch=amd64 k=2x32 b=avx512;; *) arch=amd64 k=4x32 b=avx512;; esac
  echo "== $h read-back ($arch, $k/$b)"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m1 -E "model name|CPU part" /proc/cpuinfo; nproc; uptime' 2>&1
  spread=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,SOCKET,CACHE | grep -v '^#' | awk -F, '\$2==0{split(\$3,c,\":\"); if(!s[c[4]]++) print \$1}' | head -8 | paste -sd, -")
  packed=$(timeout 60 ssh -o BatchMode=yes $h "lscpu -p=CPU,CORE,SOCKET | grep -v '^#' | awk -F, '\$3==0 && !s[\$2]++{print \$1}' | head -8 | paste -sd, -")
  if [ "$(tr ',' '\n' <<<"$spread" | wc -l | tr -d ' ')" = 8 ]; then mask=$spread; echo "   8T mask spread: $mask"; else mask=$packed; echo "   8T mask packed: $mask"; fi
  [ "$(tr ',' '\n' <<<"$mask" | wc -l | tr -d ' ')" = 8 ] || { echo "!! $h: mask has fewer than 8 cpus; UNMEASURED"; return; }
  timeout 900 scp -q -o BatchMode=yes $D/blk-$arch.test $D/keel-$arch.test $h: || { echo "!! $h scp failed"; return; }
  echo "== $h correctness, $(date -u +%T)"
  timeout 1800 ssh -o BatchMode=yes $h "./keel-$arch.test -test.run 'Strsm|Trsm|Strmm' -test.count=1 >/dev/null 2>&1; echo rc=\$?" 2>&1 | sed 's/^/   /'
  P="^BenchmarkTrsmMB\$/^$k\$/^$b\$/^n=4096\$/^side=[LR]\$/./^(full|solve-only)\$"
  for c in 1 2 3 4; do
    for t in 1 8; do
      if [ $t = 1 ]; then pin=2; else pin=$mask; fi
      n=$(timeout 7200 ssh -o BatchMode=yes $h "KEEL_TRSM_N=4096 KEEL_TRSM_MB=32,48,64,96,128,192,256 GOMAXPROCS=$t taskset -c $pin ./blk-$arch.test -test.run XXX -test.bench '$P' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h t$t c$c /" | tee -a $D/$h.rows | wc -l | tr -d ' ')
      if [ $c = 1 ] && [ "$n" -lt 28 ]; then echo "!! $h pass 1 t$t produced $n rows, want 28; stopping this host"; return; fi
    done
  done
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt $D/keel-*.rows 2>/dev/null
echo "== done $(date -u +%T)"
