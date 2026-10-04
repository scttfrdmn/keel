#!/usr/bin/env bash
# #136 full-Sgemm arm: does 3x24's kernel-level win survive packing, blocking and
# an N%24 fringe? n=2048 is the case most hostile to NR=24 -- 4x16 divides both
# dimensions exactly, 3x24 pays an M-fringe (682*3+2) and an N-fringe (85*24+8).
#
# Three arms, 972ee47's pattern, INTERLEAVED in two rounds rather than run as
# three separate tasks: interleaving separates arm order from arm, and the round
# pair is the identically-configured control that separates host state from the
# treatment. All three pinned to the same X925 core.
set -uo pipefail
PIN=19
cd /tmp/keel-3x24
echo "host: $(hostname -s)  pinned cpu: $PIN  part=$(awk -v c=$PIN '/^processor/{p=$3} /^CPU part/{if(p==c){print $4; exit}}' /proc/cpuinfo)  cap=$(cat /sys/devices/system/cpu/cpu$PIN/cpu_capacity)"
echo "loadavg: $(cat /proc/loadavg)"
for b in keel-bench.arm64 keel-bench-armB.arm64 keel-bench-armC.arm64; do
  echo "$b sha256: $(sha256sum $b | awk '{print $1}')"
done
echo
for round in 1 2; do
  for arm in A:keel-bench.arm64 B:keel-bench-armB.arm64 C:keel-bench-armC.arm64; do
    name=${arm%%:*}; bin=${arm#*:}
    echo "########## round=$round arm=$name bin=$bin ##########"
    taskset -c $PIN ./$bin -test.run=NONE -test.bench='Sgemm/n=2048' -test.count=6 -test.benchtime=1s 2>&1 | grep -E "keel-bench-kern:|^BenchmarkSgemm|^PASS|^ok"
    echo
  done
done
echo "loadavg at end: $(cat /proc/loadavg)"
