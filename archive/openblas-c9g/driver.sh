#!/bin/bash
# #196 probe (wip-196 @ 271e013): OpenBLAS v0.3.34 on one c9g.4xlarge, outside keel's
# fleet (a gate-p5 run is live on it). Prediction: archive/openblas-c9g/prereg.md.
set -uo pipefail
export AWS_PROFILE=aws AWS_REGION=us-east-1
W=~/.xtmp/wt196 D=~/.xtmp/obls NAME=obls196-c9g AMI=ami-09a93d801db0bafae
export OBLS_SSH_CONF=$D/ssh_config
TORN=0
teardown() { [ "$TORN" = 1 ] && return; TORN=1; echo "== TEARDOWN $(date -u +%T)"
  timeout 600 spawn terminate "$NAME" --yes 2>&1 | tail -2
  timeout 300 spawn list --state running 2>&1 | grep -i "$NAME" && echo "!! $NAME STILL BILLING" || echo "   $NAME is down"; }
trap teardown EXIT; trap 'teardown; exit 143' INT TERM HUP
timeout 300 spawn list --state running 2>&1 | grep -q obls196 && { echo "!! an obls196 instance already exists; refusing"; TORN=1; exit 2; }
[ -z "$(git -C "$W" status --porcelain)" ] || { echo "!! worktree dirty; refusing"; TORN=1; exit 2; }
echo "== launch $(date -u +%T) wip-196 @ $(git -C "$W" rev-parse --short HEAD)"
timeout 1800 spawn launch "$NAME" --instance-type c9g.4xlarge --ami "$AMI" --region us-east-1 --ttl 2h \
  --tag Project=keel-probe --tag Issue=196 --command 'sudo apt-get update -y && sudo apt-get install -y tmux' \
  --wait-for-ssh -y || exit 1
row="$(timeout 300 spawn list --state running --output json | jq -r --arg n "$NAME" '.[] | select(.name == $n) | "\(.public_ip)\t\(.key_name)"')"
IFS=$'\t' read -r IP KEY <<<"$row"; [ -n "${IP:-}" ] || { echo "!! no address for $NAME"; exit 1; }
printf 'Host %s\n  HostName %s\n  User ubuntu\n  IdentityFile %s/.spawn/keys/%s\n  IdentitiesOnly yes\n  StrictHostKeyChecking accept-new\n' "$NAME" "$IP" "$HOME" "$KEY" > "$OBLS_SSH_CONF"
export PATH="$D/bin:$PATH"
echo "== read-back"; timeout 120 ssh -o BatchMode=yes "$NAME" 'lscpu | grep -E "Model name|^CPU\(s\)"; grep -m1 "CPU part" /proc/cpuinfo' 2>&1
echo "== provision v0.3.34 $(date -u +%T)"
(cd "$W" && KEEL_OPENBLAS_VERSION=v0.3.34 timeout 7200 scripts/provision-openblas.sh --yes "$NAME") 2>&1 | tail -25
echo "== rounds $(date -u +%T)"
for i in $(seq 1 10); do
  if [ $((i % 2)) = 1 ]; then o="ARMV8SVE NEOVERSEV2"; else o="NEOVERSEV2 ARMV8SVE"; fi
  for ct in $o; do
    timeout 900 ssh -o BatchMode=yes "$NAME" "cd /tmp/keel-openblas-src && env OPENBLAS_CORETYPE=$ct GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 taskset -c 2 ./bench-ob.test -test.run=NONE -test.bench='OpenBLAS/n=2048' -test.benchtime=1s" 2>&1 | grep -E 'corename|^BenchmarkOpenBLAS' | sed "s/^/r$i $ct /"
  done
  timeout 900 ssh -o BatchMode=yes "$NAME" "cd /tmp/keel-openblas-src && env GOMAXPROCS=1 taskset -c 2 ./bench-ob.test -test.run=NONE -test.bench='^BenchmarkSgemm\$/n=2048' -test.benchtime=1s" 2>&1 | grep -E 'keel-sgemm-active|^BenchmarkSgemm' | sed "s/^/r$i keel /"
done
echo "== done $(date -u +%T)"
