#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The strengthening pass for archive/graviton3e/ (a blog post's evidence). Per host, stock go1.27.3:
#   1. clock (archive/graviton3e/clock) on core 2, idle;
#   2. the THROTTLE CURVE: with k = 0,1,3,7,15,31,63 loader cores running the FMA-only peak loop
#      (BenchmarkPeak/neon) on cores other than 2, core 2's FMA-only peak and its clock;
#   3. SELECTIVITY: with 63 loaders, core 2 runs the three family loops at their own FMA densities
#      (BenchmarkCeiling family/{fo,x44,shared}/threads=1);
#   4. clock on core 2, idle, after everything.
#   driver.sh main   two fresh c7g.16xlarge + two fresh c7gn.16xlarge (instance-to-instance spread)
#   driver.sh hpc    one hpc7g.16xlarge (the other Graviton3E type; offered in us-east-1a only, and
#                    aws-fleet cannot pin a zone, so it is its own fleet and may fail to launch)
# Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/graviton3e-pass/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
MODE="${1:?usage: driver.sh main or hpc}"
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
case "$MODE" in
  main) export KEEL_FLEET="g3a:c7g.16xlarge:Neoverse-V1
g3b:c7g.16xlarge:Neoverse-V1
g3ea:c7gn.16xlarge:Neoverse-V1
g3eb:c7gn.16xlarge:Neoverse-V1"; HOSTS="keel-g3a keel-g3b keel-g3ea keel-g3eb" ;;
  hpc) export KEEL_FLEET="g3hpc:hpc7g.16xlarge:Neoverse-V1"; HOSTS="keel-g3hpc" ;;
  *) echo "!! MODE must be main or hpc"; exit 2 ;;
esac
D=~/.xtmp/g3pass-$MODE
mkdir -p $D && rm -f $D/*.test $D/*.txt
B=(GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd)
env "${B[@]}" go test -c -o $D/bench.test ./bench/ && env "${B[@]}" go test -c -o $D/clock.test ./archive/graviton3e/clock/ || exit 1
echo "== built $(date -u +%T) rev $(git rev-parse --short HEAD) mode $MODE, $(go version | cut -d' ' -f3)"
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
PK="./bench.test -test.run XXX -test.bench '^BenchmarkPeak\$/^neon\$' -test.benchtime 1s"
# loaders K CMD: start K FMA-only loaders on the first K cores other than 2, run CMD on core 2, then
# wait for them. Each loader runs 30 s, which outlasts every CMD here.
LOAD='loaders() { local k=$1; shift; local n=0 c; for c in $(seq 0 63); do [ $c = 2 ] && continue; [ $n -ge $k ] && break; taskset -c $c ./bench.test -test.run XXX -test.bench "^BenchmarkPeak\$/^neon\$" -test.benchtime 30s -test.count 1 >/dev/null 2>&1 & n=$((n+1)); done; [ $k -gt 0 ] && sleep 3; taskset -c 2 "$@"; wait; }'
host() {
  local h=$1 k
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m1 "CPU part" /proc/cpuinfo; nproc; cat /sys/devices/virtual/dmi/id/product_name 2>/dev/null; curl -s -m 2 http://169.254.169.254/latest/meta-data/instance-type 2>/dev/null || true; echo; curl -s -m 2 http://169.254.169.254/latest/meta-data/instance-id 2>/dev/null || true; echo; uptime' 2>&1
  timeout 900 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  run() { timeout 3600 ssh -o BatchMode=yes $h "$LOAD; $2" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h $1 /"; }
  run idle0 "taskset -c 2 $CLK -test.count 3"
  for k in 0 1 3 7 15 31 63; do
    run "curve-k$k" "loaders $k $PK -test.count 3"
    run "clock-k$k" "loaders $k $CLK -test.count 2"
  done
  run select63 "loaders 63 ./bench.test -test.run XXX -test.bench '^BenchmarkCeiling\$/family/.*/threads=1\$' -test.benchtime 1s -test.count 2"
  run idle1 "taskset -c 2 $CLK -test.count 3"
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
