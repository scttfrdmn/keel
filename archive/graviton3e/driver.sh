#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# What does "Graviton3E: ~35% higher vector performance than Graviton3" mean? One c7g.16xlarge
# (Graviton3) against one c7gn.16xlarge (Graviton3E), same Neoverse V1 core, AWS-stated sustained
# clock 2.6 GHz on both, stock go1.27.3, this tree. Per host, in order:
#   A. clock (archive/graviton3e/clock, validated 3.884 Gops/s vs cpufreq 3.900 GHz on a GB10 X925)
#      on core 2, idle; 5 samples.
#   B. under all-core vector load: 63 loaders (BenchmarkPeak/neon, the FMA-only loop) on every core
#      but 2, then the clock probe and BenchmarkPeak/neon on core 2, then the loaders end.
#   C. BenchmarkPeak/neon 1T on core 2, idle; BenchmarkCeiling (compute + stream) on cores 0-7.
#   D. the shipped 5x16-pb kernel: BenchmarkLoadProbe Ctl, 1T on core 2.
#   E. BenchmarkScale n=4096 1T/8T on cores 0-7, 2 -randlayout seeds x -test.count=5.
#   F. A again, idle, after everything.
# Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/graviton3e/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="gvt3:c7g.16xlarge:Neoverse-V1
gvt3e:c7gn.16xlarge:Neoverse-V1"
HOSTS="keel-gvt3 keel-gvt3e" D=~/.xtmp/g3e
mkdir -p $D && rm -f $D/*.test $D/*.txt
B=(GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd)
for s in 1 2; do env "${B[@]}" go test -c -ldflags="-randlayout=$s" -o $D/bench-s$s.test ./bench/ || exit 1; done
env "${B[@]}" go test -c -o $D/vec.test ./internal/vec/ && env "${B[@]}" go test -c -o $D/clock.test ./archive/graviton3e/clock/ || exit 1
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
CLK="./clock.test -test.run XXX -test.bench Clock -test.benchtime 1s"
PK="./bench-s1.test -test.run XXX -test.bench '^BenchmarkPeak\$/^neon\$' -test.benchtime 1s"
host() {
  local h=$1
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "CPU part" /proc/cpuinfo; nproc; lscpu | grep -E "Model name|L1d|L2|L3|NUMA node"; ls /sys/devices/system/cpu/cpu0/cpufreq 2>&1 | head -1; uptime' 2>&1
  timeout 900 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  run() { timeout 3600 ssh -o BatchMode=yes $h "$2" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h $1 /"; }
  run A-idle "taskset -c 2 $CLK -test.count 5"
  run B-loaded "for c in \$(seq 0 63); do [ \$c = 2 ] && continue; taskset -c \$c ./bench-s1.test -test.run XXX -test.bench '^BenchmarkPeak\$/^neon\$' -test.benchtime 25s -test.count 1 >/dev/null 2>&1 & done; sleep 3; taskset -c 2 $CLK -test.count 5; taskset -c 2 $PK -test.count 5; wait"
  run C-peak "taskset -c 2 $PK -test.count 10"
  run C-ceil "taskset -c 0-7 ./bench-s1.test -test.run XXX -test.bench '^BenchmarkCeiling\$' -test.benchtime 1s -test.count 5"
  run D-kernel "taskset -c 2 ./vec.test -test.run XXX -test.bench '^BenchmarkLoadProbe\$/Ctl' -test.benchtime 1s -test.count 10"
  for s in 1 2; do run E-scale-s$s "taskset -c 0-7 ./bench-s$s.test -test.run XXX -test.bench '^BenchmarkScale\$' -test.benchtime 1s -test.count 5"; done
  run F-idle "taskset -c 2 $CLK -test.count 5"
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
