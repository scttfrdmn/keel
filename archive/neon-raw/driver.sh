#!/bin/bash
# Exploration-tier A/B of the raw NEON form against the shipped shim kernels (#191, 7538489).
# Not a verdict: 4xlarge sizes are correctness-class. Rounds interleave so no arm owns a window.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:AWS Graviton5"
D=~/.xtmp/rawab
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"; timeout 900 scripts/aws-fleet.sh down; timeout 300 spawn list --state running 2>&1 | grep -i keel- && echo "!! STILL BILLING" || echo "   fleet is down"; }
trap teardown EXIT INT TERM
echo "== launch $(date -u +%T) rev $(git rev-parse --short HEAD)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
for h in keel-gvt3 keel-gvt4 keel-gvt5; do
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m2 -E "CPU (implementer|part)" /proc/cpuinfo; lscpu | grep -E "Model name"; nproc; uptime' 2>&1
  timeout 300 scp -q -o BatchMode=yes $D/vec.test $D/bench.test $h: || { echo "!! $h scp failed"; continue; }
done
run() { # host
  local h=$1
  for i in $(seq 1 10); do
    timeout 600 ssh -o BatchMode=yes $h "taskset -c 2 ./vec.test -test.run XXX -test.bench RawNEON -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i /"
    timeout 300 ssh -o BatchMode=yes $h "taskset -c 2 ./bench.test -test.run XXX -test.bench 'Peak/neon\$|FMARoof64/neon\$' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$h r$i /"
  done
}
for h in keel-gvt3 keel-gvt4 keel-gvt5; do run $h > $D/$h.txt & done; wait
cat $D/keel-gvt*.txt
echo "== done $(date -u +%T)"
