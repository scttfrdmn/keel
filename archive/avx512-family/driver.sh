#!/bin/bash
# Exploration tier (#199, 7b55f74): the AVX-512 ceiling family on Skylake-SP, Zen 4, Zen 5.
# Reading: archive/avx512-family/prereg.md.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="skx:c5n.4xlarge:Intel Skylake-SP
zen4:c7a.2xlarge:AMD EPYC Genoa
zen5:c8a.2xlarge:AMD EPYC Turin"
D=/Users/scttfrdmn/.xtmp/avxfam
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"; timeout 900 scripts/aws-fleet.sh down; timeout 300 spawn list --state running 2>&1 | grep -i keel- && echo "!! STILL BILLING" || echo "   fleet is down"; }
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
echo "== launch $(date -u +%T) rev $(git rev-parse --short HEAD)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
HOSTS="keel-skx keel-zen4 keel-zen5"
for h in $HOSTS; do
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'lscpu | grep -E "Model name|^CPU\(s\)|Thread"; grep -o -m1 avx512f /proc/cpuinfo; uptime' 2>&1
  timeout 300 scp -q -o BatchMode=yes $D/vec.test $D/bench.test $h: || echo "!! $h scp failed"
  echo "== $h witness"; timeout 300 ssh -o BatchMode=yes $h './vec.test -test.run "PeakFamily|PeakCanonical|PeakChains" -test.v' 2>&1 | grep -E '^(--- |ok|FAIL|PASS)'
done
run() {
  local h=$1
  if ! timeout 300 ssh -o BatchMode=yes $h './vec.test -test.run "PeakFamily|PeakCanonical|PeakChains"' >/dev/null 2>&1; then echo "!! $h witness FAILED, not timed"; return; fi
  for i in $(seq 1 10); do
    timeout 600 ssh -o BatchMode=yes $h "taskset -c 2 ./bench.test -test.run XXX -test.bench 'FMAFamily|Peak/avx512\$|FMARoof64/avx512\$' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i /"
  done
}
for h in $HOSTS; do run $h > $D/$h.txt & done; wait
cat $D/keel-*.txt
echo "== done $(date -u +%T)"
