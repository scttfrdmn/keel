#!/usr/bin/env bash
# #136 follow-up: measured peak per GB10 core type, which is (a) the denominator
# every published rate needs and (b) a direct test of whether the 2.96x gap
# between this sweep and archive/neon136 is the two core types differing in FMA
# pipe count as well as clock. cpu_capacity is a scalar-ish DMIPS measure and
# would not see a pipe-count difference at all.
set -uo pipefail
BIN=/tmp/keel-3x24/keel-bench.arm64
echo "host: $(hostname -s)  rev-under-test: a8f520e"
echo "binary sha256: $(sha256sum "$BIN" | awk '{print $1}')"
echo "loadavg: $(cat /proc/loadavg)"
echo
echo "=== core model (lscpu MIDR -> ARM core name) ==="
lscpu 2>/dev/null | grep -iE "model name|vendor|core|cpu\(s\)|bogomips|flags" | head -12
echo
for PIN in 19 0; do
  CAP=$(cat /sys/devices/system/cpu/cpu$PIN/cpu_capacity)
  PART=$(awk -v c=$PIN '/^processor/{p=$3} /^CPU part/{if(p==c){print $4; exit}}' /proc/cpuinfo)
  MAXF=$(cat /sys/devices/system/cpu/cpu$PIN/cpufreq/cpuinfo_max_freq 2>/dev/null || echo unknown)
  echo "########## cpu$PIN  capacity=$CAP  part=$PART  max_freq_khz=$MAXF ##########"
  taskset -c $PIN bash -c 'grep Cpus_allowed_list /proc/self/status'
  taskset -c $PIN "$BIN" -test.run=NONE -test.bench='Peak' -test.count=10 -test.benchtime=1s
  echo
done
echo "=== 4x16 and 3x24 on an A725, to test the core-type ratio on the SAME rows ==="
taskset -c 0 "$BIN" -test.run=NONE -test.bench='Kernel/(4x16|3x24)/neon/kc=512' -test.count=10 -test.benchtime=1s
echo "loadavg at end: $(cat /proc/loadavg)"
