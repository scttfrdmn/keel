#!/usr/bin/env bash
# #176 discriminator: does this host give 8 pinned threads 8 cores of REGISTER-ONLY
# compute? No memory in the timed region, so bandwidth/cache cannot explain a shortfall.
set -uo pipefail
B=/tmp/keel-probe/probe.arm64
echo "host: $(hostname -s)  $(uname -m)  kernel=$(uname -r)"
echo "region-hint: $(curl -s --max-time 3 http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null || echo unknown)"
echo "instance:    $(curl -s --max-time 3 http://169.254.169.254/latest/meta-data/instance-type 2>/dev/null || echo unknown)"
echo "cpus=$(nproc) smt=$(lscpu | awk -F: '/Thread\(s\) per core/{print $2+0}')  model=$(lscpu | awk -F: '/Model name/{gsub(/^ +/,"",$2);print $2;exit}')"
echo "loadavg: $(cat /proc/loadavg)"
echo "binary sha256: $(sha256sum $B | cut -c1-20)"
echo
for t in 1 8; do
  echo "########## threads=$t (taskset 0-$((t-1))) ##########"
  taskset -c 0-$((t-1)) bash -c 'grep Cpus_allowed_list /proc/self/status'
  taskset -c 0-$((t-1)) env GOMAXPROCS=$t "$B" -test.run=NONE -test.bench="Ceiling/compute/neon/threads=$t" -test.count=10 -test.benchtime=1s 2>&1 | grep -E "^BenchmarkCeiling|keel-bench-gomaxprocs"
  echo
done
echo "loadavg at end: $(cat /proc/loadavg)"
