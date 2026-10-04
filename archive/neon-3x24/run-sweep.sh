#!/usr/bin/env bash
# One-off #136 characterization sweep: 3x24 vs 4x16 on one Cortex-X925 core.
set -uo pipefail
PIN=19
BIN=/tmp/keel-3x24/keel-bench.arm64

echo "=== witnesses (read off this run, not assumed) ==="
echo "host: $(hostname -s)  kernel: $(uname -r)  $(uname -m)"
echo "rev-under-test: a8f520e (tree clean; binary built from this rev)"
echo "binary sha256: $(sha256sum "$BIN" | awk '{print $1}')"
echo "pinned cpu: $PIN  capacity: $(cat /sys/devices/system/cpu/cpu$PIN/cpu_capacity)"
echo "pinned cpu part: $(awk -v c=$PIN '/^processor/{p=$3} /^CPU part/{if(p==c){print $4; exit}}' /proc/cpuinfo)"
echo "governor: $(cat /sys/devices/system/cpu/cpu$PIN/cpufreq/scaling_governor 2>/dev/null || echo none)"
echo "loadavg at start: $(cat /proc/loadavg)"
echo "co-tenants (top 5 by cpu):"
ps -eo pcpu,comm --sort=-pcpu | head -6
echo
echo "=== affinity witness: what the pinned shell actually got ==="
taskset -c $PIN bash -c 'grep Cpus_allowed_list /proc/self/status'
echo
echo "=== sweep: one binary, all shapes, count=10 benchtime=1s ==="
taskset -c $PIN "$BIN" -test.run=NONE -test.bench='Kernel' -test.count=10 -test.benchtime=1s
echo
echo "=== loadavg at end: $(cat /proc/loadavg) ==="
