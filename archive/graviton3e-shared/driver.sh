#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The idle shared-loop gap (archive/graviton3e-pass/): Graviton3E runs the `shared` family loop 4.8%
# faster than Graviton3 on an idle core (43.3-43.6 vs 41.43), while `x44` and `fo` match. `shared` is
# the shim-built neonPeak, whose 35-instruction loop carries 16 anchor NOPs (golang/go#80830); x44 is
# raw archsimd with none (spill-audit). One c7g.16xlarge and one c7gn.16xlarge, idle, core 2:
#   - core identity: MIDR_EL1 / REVIDR_EL1 and /proc/cpuinfo variant/revision;
#   - BenchmarkPeakProbe (shim16 = neonPeak with NOPs, raw16 = its NOP-free archsimd twin, raw24,
#     and their float64 twins), BenchmarkPeakVariants, BenchmarkV1Probe (archive/neon-raw/'s V1
#     loop variants), and the family loops via BenchmarkCeiling family/*/threads=1;
#   - -test.count=10 each.
# Predictions: prereg.md beside this file.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
SELF="archive/graviton3e-shared/driver.sh"
[[ "$(git rev-parse HEAD)" == "$(git log -1 --format=%H -- "$SELF")" ]] || { echo "!! HEAD is not the commit that froze this driver; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=3h
export KEEL_FLEET="g3:c7g.16xlarge:Neoverse-V1
g3e:c7gn.16xlarge:Neoverse-V1"
HOSTS="keel-g3 keel-g3e" D=~/.xtmp/g3shared
mkdir -p $D && rm -f $D/*.test $D/*.txt
B=(GOOS=linux GOARCH=arm64 GOEXPERIMENT=simd)
env "${B[@]}" go test -c -o $D/vec.test ./internal/vec/ && env "${B[@]}" go test -c -o $D/bench.test ./bench/ || exit 1
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
  local h=$1
  echo "== $h identity"
  timeout 120 ssh -o BatchMode=yes $h 'curl -s -m 2 http://169.254.169.254/latest/meta-data/instance-type; echo; grep -m4 -E "CPU (implementer|variant|part|revision)" /proc/cpuinfo; for r in midr_el1 revidr_el1; do echo "$r $(cat /sys/devices/system/cpu/cpu2/regs/identification/$r 2>&1)"; done; uname -r; uptime' 2>&1 | sed 's/^/   /'
  timeout 900 scp -q -o BatchMode=yes $D/*.test $h: || { echo "!! $h scp failed"; return; }
  for b in PeakProbe PeakVariants V1Probe; do
    timeout 3600 ssh -o BatchMode=yes $h "taskset -c 2 ./vec.test -test.run XXX -test.bench '^Benchmark$b\$' -test.benchtime 1s -test.count 10" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h /"
  done
  timeout 3600 ssh -o BatchMode=yes $h "taskset -c 2 ./bench.test -test.run XXX -test.bench '^BenchmarkCeiling\$/family/.*/threads=1\$' -test.benchtime 1s -test.count 10" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h /"
}
for h in $HOSTS; do host $h > $D/$h.txt 2>&1 & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
