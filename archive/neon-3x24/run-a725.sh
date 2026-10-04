#!/usr/bin/env bash
# #136 Q2: the full NEON shape sweep on a Cortex-A725 (little) core, so the
# little-core ranking is complete rather than one answer about 8x12. Step 4's
# "a shallow spill is throughput-free" was measured on little cores and is
# refuted on the big one; this settles whether it holds where it was observed.
set -uo pipefail
PIN=0
BIN=/tmp/keel-q2/kb-a725.arm64
echo "host: $(hostname -s)  rev-under-test: 9f040cb"
echo "binary sha256: $(sha256sum "$BIN" | awk '{print $1}')"
echo "pinned cpu: $PIN  capacity: $(cat /sys/devices/system/cpu/cpu$PIN/cpu_capacity)  part: $(awk -v c=$PIN '/^processor/{p=$3} /^CPU part/{if(p==c){print $4; exit}}' /proc/cpuinfo)  max_khz: $(cat /sys/devices/system/cpu/cpu$PIN/cpufreq/cpuinfo_max_freq)"
echo "governor: $(cat /sys/devices/system/cpu/cpu$PIN/cpufreq/scaling_governor)"
echo "loadavg: $(cat /proc/loadavg)"
taskset -c $PIN bash -c 'grep Cpus_allowed_list /proc/self/status'
echo
taskset -c $PIN "$BIN" -test.run=NONE -test.bench='Kernel|Peak' -test.count=10 -test.benchtime=1s
echo "loadavg at end: $(cat /proc/loadavg)"
