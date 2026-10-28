#!/bin/bash
# Exploration tier (#198, def1b79): the nest witness for pre-broadcast A.
# Rule and predictions: archive/neon-pb/nest-prereg.md.
set -uo pipefail
cd /Users/scttfrdmn/src/keel
export AWS_PROFILE=aws AWS_REGION=us-east-1 KEEL_FLEET_MARKET=on-demand KEEL_FLEET_TTL=2h
export KEEL_FLEET="gvt3:c7g.4xlarge:Neoverse-V1
gvt4:c8g.4xlarge:Neoverse-V2
gvt5:c9g.4xlarge:AWS Graviton5"
D=~/.xtmp/nest
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"; timeout 900 scripts/aws-fleet.sh down; timeout 300 spawn list --state running 2>&1 | grep -i keel- && echo "!! STILL BILLING" || echo "   fleet is down"; }
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
echo "== launch $(date -u +%T) rev $(git rev-parse --short HEAD)"; timeout 3600 scripts/aws-fleet.sh up || exit 1
for h in keel-gvt3 keel-gvt4 keel-gvt5; do
  echo "== $h read-back"
  timeout 120 ssh -o BatchMode=yes $h 'grep -m2 -E "CPU (implementer|part)" /proc/cpuinfo; lscpu | grep -E "Model name"; uptime' 2>&1
  timeout 300 scp -q -o BatchMode=yes $D/block.test $D/block64.test $h: || echo "!! $h scp failed"
done
one() { # host binary env filter round
  timeout 1800 ssh -o BatchMode=yes "$1" "$3 GOMAXPROCS=1 taskset -c 2 ./$2.test -test.run XXX -test.bench '$4' -test.benchtime 1s" 2>&1 | grep -E '^Benchmark' | sed "s/^/$1 r$5 $2 /"
}
run() {
  local h=$1
  for i in $(seq 1 5); do
    one $h block 'KEEL_BLOCKING_KC=256,384 KEEL_BLOCKING_MC=36,72,144 KEEL_BLOCKING_NC=2048' 'Blocking/(3x24|5x16|4x20|7x12)/(neon|neon-pb)/' $i
    one $h block64 'KEEL_BLOCKING64_KC=256,512 KEEL_BLOCKING64_MC=72,144 KEEL_BLOCKING64_NC=2048' 'Blocking64/(3x12|5x8|4x10|7x6)/(neon|neon-pb)/' $i
  done
}
for h in keel-gvt3 keel-gvt4 keel-gvt5; do run $h > $D/$h.txt & done; wait
cat $D/keel-gvt*.txt
echo "== done $(date -u +%T)"
