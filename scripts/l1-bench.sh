#!/usr/bin/env bash
# A/B the Level-1 routines across the cache hierarchy: this ref against another.
#
# The measurement for issue #47 — internal/l1's loops were reshaped from
# index-driven to slice-advancing, which removed every surviving bounds check and
# roughly halved the six unrolled reductions' instruction counts, while making the
# four non-unrolled loops (Axpy, Scal, both widths) 1-5 instructions *longer* for
# the reason docs/toolchain-notes.md T19 gives. Static counts cannot say what
# either of those is worth, because these routines are not all issue-bound: at
# n = 1<<20 they are reading 4 MB per call and the loop body is not the limit.
#
# So this is NOT a gate: it certifies nothing, it changes no criterion, and it
# exits 0 whatever it measures. Its product is a table, per host, of the same
# benchmark run against two builds.
#
# The four sizes in bench/bench_test.go's `sizes` are the whole point of running
# this at all: 256, 4096, 65536 and 1<<20 float32s are 1 KB, 16 KB, 256 KB and
# 4 MB, i.e. L1-, L2-, L3- and memory-resident on all three hosts in
# docs/hosts.md. A loop-body change should show up at 1 KB and wash out at 4 MB,
# and if it does not, the explanation is not "the loop got shorter".
#
# The harness is scripts/ab.sh, shared with edge-bench.sh (#131): two builds, the
# base one from a detached worktree, then every configured host. Read its header
# for the mechanism and for why work never happens at a driver's top level (#51).
#
# usage: scripts/l1-bench.sh [BASE_REF]     (default: HEAD~1)
set -euo pipefail

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/ab.sh
  source scripts/ab.sh

  AB_TITLE="== l1-bench — issue #47. Not a gate: this certifies nothing. =="
  AB_FILTER="${KEEL_L1_FILTER:-BenchmarkL1S}"
  AB_TAG="l1"
  AB_NOTES=("sizes are bench/bench_test.go's: 1 KB / 16 KB / 256 KB / 4 MB of float32.")
  ab_run "${1:-HEAD~1}"

  echo "done. Numbers above are this host set at these two builds; nothing here is a criterion."
}

main "$@"
