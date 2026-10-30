#!/bin/bash
# A/B rounds for archive/l2-move-ab: 6 rounds, arm order alternating.
set -u
B='^BenchmarkL2Par$/^(SgemvN|SgemvT|Sger|Ssyr|Ssyr2|StrmvN|StrmvT)$/^m=(1024|4096)$'
for r in 1 2 3 4 5 6; do
  if [ $((r % 2)) = 1 ]; then o="before after"; else o="after before"; fi
  for arm in $o; do
    echo "## round $r arm $arm"
    ./$arm.test -test.run '^$' -test.bench "$B" -test.cpu 1,8 -test.count 1 2>&1 | sed "s/^Benchmark/$arm Benchmark/" | grep -E "^$arm Benchmark|^##"
  done
done
