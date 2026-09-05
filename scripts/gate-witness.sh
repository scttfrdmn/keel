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
# diff isolates code. Three, all proven run-specific and none touched by the port: the RUN_STAMP
# timestamp in archive/candidate filenames, the live disk-headroom reading, and the git short-sha
# (P5_REV) the gate stamps into those same filenames and its "on this commit (…)" line — scrubbed
# in the three contexts it appears, so the baseline is stable across commits (a re-render at any
# HEAD normalizes to <SHA>). NOT normalized: anything the port could move — a planted GATE_PEAK
# change still survives this scrub (selfcheck proves it fails first).
_normalize() {
  # Carry-chain (#158): gate-p4's criterion-7 (Ssyrk/Sgemm) and gate-p3's OpenBLAS coretype sweep run
  # LIVE via direct ssh (NOT remote_exec, so the corpus does not replay them — the #157 gap, confirmed
  # to span gate-p4 too). Their benchmark NUMBERS drift run-to-run. The gate-p4 port's amd64 branch is
  # byte-verbatim, so it touches NONE of these numbers; scrubbing the measurement values lets the diff
  # isolate the CODE effect (zero on amd64), while the fail-first (a KERN_FUNCS *text* change) survives
  # the scrub — proving non-vacuity. Full hermeticity is #157. Only in carry-chain mode: the unit-1
  # throughput witness must still catch CEIL_FRACTION (a %), so its scrub set is unchanged.
  # Interval scrub REQUIRES a space after the comma, so deterministic audit ranges (`loop [101,341]`,
  # no space) are preserved while measurement CIs (`[141, 142.3]`, space) are scrubbed.
  local extra=()
  [[ -z "${KEEL_WITNESS_NOBUILD:-}" ]] || extra=(
    -e 's#[0-9]+(\.[0-9]+)? GFLOP/s#<G> GFLOP/s#g'
    -e 's#\[[0-9]+(\.[0-9]+)?%?, [0-9]+(\.[0-9]+)?%?\]#[<I>]#g'
    -e 's#D=(inf|[0-9]+(\.[0-9]+)?)#D=<D>#g'
    -e 's#[0-9]+\.[0-9]+%#<P>%#g'
    -e 's#(raw|drift of|win of|by) [0-9]+(\.[0-9]+)?#\1 <N>#g'
    # A pre-existing bug (#159): gate-p4 calls fleet_shortfall (defined in roofline.sh) without
    # sourcing it, so bash prints `gate-p4.sh: line NNN: fleet_shortfall: command not found`. The
    # LINE NUMBER is position-dependent — any edit above it shifts it — so it is a run-position
    # artifact like <TS>/<SHA>, scrubbed so the port's null-change is not masked by its own added
    # lines. The error itself is pre-existing and identical before/after; #159 fixes the bug.
    -e 's#(gate-p[0-9]+\.sh: line )[0-9]+(: fleet_shortfall)#\1<N>\2#g'
  )
  sed -E \
    -e 's/[0-9]{8}T[0-9]{6}Z/<TS>/g' \
    -e 's/[0-9]+(\.[0-9]+)? MiB free/<N> MiB free/g' \
    -e 's/(gate-p[0-9]+-)[0-9a-f]{7,40}/\1<SHA>/g' \
    -e 's/(candidates-)[0-9a-f]{7,40}/\1<SHA>/g' \
    -e 's/(commit \()[0-9a-f]{7,40}/\1<SHA>/g' \
    "${extra[@]}"
}

render() {
  # KEEL_WITNESS_NOBUILD: replay a PRE-BUILT corpus as-is (the #158 carry-chain witness — a live
  # `record` run captured p5+p4+p3's calls including the real host probe, so _build_corpus's
  # throughput-only bootstrap would wipe it). Default (unit 1) still bootstraps from the archive.
  [[ -n "${KEEL_WITNESS_NOBUILD:-}" ]] || _build_corpus
  # Clear stale sub-gate logs so the capture below picks THIS render's, not a prior run's (the
  # bug that first read a stale P4LOG and reported a false null-change; #158).
  [[ -z "${KEEL_WITNESS_NOBUILD:-}" ]] || rm -f build/gate-p4-under-p5-*.log build/gate-p3-under-p4-*.log
  local p5out; p5out="$(mktemp)"
  KEEL_REPLAY=replay KEEL_REPLAY_DIR="$CORPUS" KEEL_REMOTE_HOSTS="$HOST" \
    bash scripts/gate-p5.sh >"$p5out" 2>&1 || true
  {
    printf '=== gate-p5 stdout ===\n'; cat "$p5out"
    # THE CARRY CHAIN: gate-p5 spawns gate-p4 spawns gate-p3, and each sub-gate's FULL audit
    # detail is written to its own P4LOG/P3LOG file — only the tally bubbles up to p5's stdout
    # (the selfcheck fail-first caught this: a gate-p4 KERN_FUNCS change moves the p4 audit
    # detail but not the tally, so a p5-stdout-only witness was inert to it). So the reading must
    # concatenate the sub-gate logs, where the gate-p4 port's edits actually render.
    if [[ -n "${KEEL_WITNESS_NOBUILD:-}" ]]; then
      local p4log p3log
      p4log="$(ls -t build/gate-p4-under-p5-*.log 2>/dev/null | head -1)"
      p3log="$(ls -t build/gate-p3-under-p4-*.log 2>/dev/null | head -1)"
      [[ -n "$p4log" && -f "$p4log" ]] && { printf '=== P4LOG ===\n'; cat "$p4log"; }
      [[ -n "$p3log" && -f "$p3log" ]] && { printf '=== P3LOG ===\n'; cat "$p3log"; }
    fi
  } | _normalize >"$1"
  rm -f "$p5out"
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
    # Non-vacuity: a planted change MUST make render() differ, or the witness is inert.
    #  - default (unit 1, gate-p5 throughput): CEIL_FRACTION, the share bar every scale criterion
    #    divides against — a value that survives into the rendering.
    #  - carry-chain (#158, KEEL_WITNESS_NOBUILD): plant a gate-p4 KERN_FUNCS change — the arm64
    #    kernel name into the amd64 path, the EXACT edit-shape the port makes. On amd64 the audit
    #    cannot find Kernel8x8, so the carried-P2 K-loop audit renders differently. This proves the
    #    witness sees a gate-p4 change through the p5→p4→p3 chain, not just a gate-p5 change.
    if [[ -n "${KEEL_WITNESS_NOBUILD:-}" ]]; then
      PLANT_FILE=scripts/gate-p4.sh
      PLANT_SED='s|^KERN_FUNCS="Kernel2x32,Kernel4x32"|KERN_FUNCS="Kernel2x32,Kernel8x8"|'
      PLANT_DESC="gate-p4 KERN_FUNCS (arm64 name into the amd64 path)"
    else
      PLANT_FILE=scripts/gate-p5.sh
      PLANT_SED='s|^CEIL_FRACTION=44.2|CEIL_FRACTION=43.0|'
      PLANT_DESC="gate-p5 CEIL_FRACTION"
    fi
    render /tmp/gate-witness.base
    cp "$PLANT_FILE" /tmp/gate-witness.plantorig
    trap 'cp /tmp/gate-witness.plantorig "$PLANT_FILE"' EXIT
    sed -i.bak "$PLANT_SED" "$PLANT_FILE" && rm -f "$PLANT_FILE.bak"
    render /tmp/gate-witness.planted
    cp /tmp/gate-witness.plantorig "$PLANT_FILE"; trap - EXIT
    if diff -q /tmp/gate-witness.base /tmp/gate-witness.planted >/dev/null; then
      echo "NON-VACUITY FAIL: the witness did not detect a planted $PLANT_DESC change — it is inert"; exit 1
    fi
    echo "NON-VACUITY PASS: the witness detects a planted $PLANT_DESC change (fails first)" ;;
  *) echo "usage: $0 render OUT | check A B | selfcheck" >&2; exit 2 ;;
esac
