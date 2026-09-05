#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# The #155 null-change witness runner (docs/gate-arm64-port.md, unit 1). It drives gate-p5 in
# REPLAY against a fixed corpus, so the gate's rendering is a deterministic function of that
# corpus and nothing else. Diffing two replays across a gate edit is the byte-unchanged proof
# the null-change ruling requires: identical rendering means the edit did not touch what the
# amd64 certificate machinery renders. See scripts/gate-replay.sh for the record/replay mode.
#
# Usage:
#   scripts/gate-witness.sh render OUT   — render the replay to file OUT (the witness reading)
#   scripts/gate-witness.sh check A B    — diff two readings; empty diff is a PASS
#   scripts/gate-witness.sh selfcheck    — prove the witness is non-vacuous: a planted GATE_PEAK
#                                          change MUST make render() differ, or the witness is inert
#
# SCOPE (DESIGN.md rule 12): the corpus here is a tracked THROUGHPUT sweep archive, so this
# witnesses the ceiling/scale/share/percent-of-peak rendering (the GATE_PEAK consumers). The
# dispatch/forced sections have no tracked log, so they render `unmeasured` identically in both
# replays and cancel; the dispatch section's own witness is the live amd64 baseline pair
# (captured pre-port on a lab host). The -race leg is skipped under replay (a native compile,
# outside the port's scope).
set -euo pipefail
cd "$(dirname "$0")/.."

CORPUS="${KEEL_WITNESS_DIR:-build/witness-corpus}"
HOST="${KEEL_WITNESS_HOST:-keel-skx}"
# A tracked evidentiary throughput sweep (keel-skx, Intel 8124M, era pinned8): full row set
# (Ceiling/compute/{avx512,avx2,scalar}, Peak/*, Scale/{Sgemm,Ssyrk,Ssymm,Strsm}).
ARCHIVE="${KEEL_WITNESS_ARCHIVE:-archive/pinned8/bench-gate-p5-6ba6566-keel-skx-20260823T004407Z-1.txt}"
# The provenance the stubbed remote_probe returns: the archive's host, classed evidentiary so
# the judged scale/share branches render (instance in KEEL_EVIDENTIARY_SIZES + spawn on-demand).
PROV='Intel(R) Xeon(R) Platinum 8124M CPU @ 3.00GHz | instance=c5n.18xlarge | virt=guest | 72 cpus | 36 cores | smt=2 | 2 sockets | governor=absent | tmux=yes | Linux 6.8.0-aws | L1d=32K L2=1024K L3=25344K | spawn=i-0witness:c5n.18xlarge:ondemand'

# _build_corpus — set up $CORPUS. Self-bootstrapping: a first replay against an EMPTY corpus
# logs the throughput sweep's call key to _misses.log (the key is a function of the binary and
# its -test.bench/-count/-benchtime, which this script does not want to recompute by hand); the
# archive is then placed at that key. Robust to a change in the gate's bench flags.
_build_corpus() {
  rm -rf "$CORPUS"; mkdir -p "$CORPUS"
  printf '%s\n' "$PROV" >"$CORPUS/probe-$HOST.out"
  KEEL_REPLAY=replay KEEL_REPLAY_DIR="$CORPUS" KEEL_REMOTE_HOSTS="$HOST" \
    bash scripts/gate-p5.sh >/dev/null 2>&1 || true
  local key
  key="$(awk -F'\t' '$2 ~ /-test\.bench=Scale\|Peak\|Ceiling/ {print $1; exit}' "$CORPUS/_misses.log" 2>/dev/null)"
  [[ -n "$key" ]] || { echo "gate-witness: could not find the throughput sweep call key" >&2; exit 1; }
  cp "$ARCHIVE" "$CORPUS/exec-$key.out"
  printf '0\n' >"$CORPUS/exec-$key.rc"
  rm -f "$CORPUS/_misses.log"
}

# _normalize — scrub the fields that vary run-to-run INDEPENDENTLY of the gate's code, so the
# diff isolates code. Two only, both proven run-specific by the determinism check and neither
# touched by the port: the RUN_STAMP timestamp embedded in archive/candidate filenames, and the
# live disk-headroom reading. NOT normalized: the git short-sha (the post-port replay runs on
# uncommitted edits, so HEAD is unchanged in both) or anything the port could plausibly move —
# a planted GATE_PEAK change still survives this scrub (selfcheck proves it).
_normalize() { sed -E -e 's/[0-9]{8}T[0-9]{6}Z/<TS>/g' -e 's/[0-9]+(\.[0-9]+)? MiB free/<N> MiB free/g'; }

render() {
  _build_corpus
  KEEL_REPLAY=replay KEEL_REPLAY_DIR="$CORPUS" KEEL_REMOTE_HOSTS="$HOST" \
    bash scripts/gate-p5.sh 2>&1 | _normalize >"$1" || true
  echo "gate-witness: rendered replay to $1 ($(wc -l <"$1") lines)"
}

case "${1:-}" in
  render) [[ -n "${2:-}" ]] || { echo "usage: $0 render OUT" >&2; exit 2; }; render "$2" ;;
  check)
    [[ -n "${3:-}" ]] || { echo "usage: $0 check A B" >&2; exit 2; }
    if diff -u "$2" "$3" >/tmp/gate-witness.diff; then
      echo "WITNESS PASS: renderings byte-identical ($2 == $3)"
    else
      echo "WITNESS FAIL: renderings differ — the edit changed the rendered gate output:"; cat /tmp/gate-witness.diff; exit 1
    fi ;;
  selfcheck)
    # Non-vacuity: a planted GATE_PEAK change MUST make render() differ.
    render /tmp/gate-witness.base
    cp scripts/gate-p5.sh /tmp/gate-p5.witnessorig
    trap 'cp /tmp/gate-p5.witnessorig scripts/gate-p5.sh' EXIT
    sed -i.bak 's|^GATE_PEAK="Peak/avx512"|GATE_PEAK="Peak/avx2"|' scripts/gate-p5.sh && rm -f scripts/gate-p5.sh.bak
    render /tmp/gate-witness.planted
    cp /tmp/gate-p5.witnessorig scripts/gate-p5.sh; trap - EXIT
    if diff -q /tmp/gate-witness.base /tmp/gate-witness.planted >/dev/null; then
      echo "NON-VACUITY FAIL: the witness did not detect a planted GATE_PEAK change — it is inert"; exit 1
    fi
    echo "NON-VACUITY PASS: the witness detects a planted GATE_PEAK change (fails first)" ;;
  *) echo "usage: $0 render OUT | check A B | selfcheck" >&2; exit 2 ;;
esac
