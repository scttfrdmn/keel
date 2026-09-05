<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #155 null-change witness — pre-port baseline (rev `949994d`)

The tracked pre-port artifacts for the #155 gate arm64 port (`docs/gate-arm64-port.md`, unit 1).
The port must leave the gate's amd64 rendering byte-identical; this directory is what "identical"
is checked against. Mechanism: `scripts/gate-replay.sh` (record/replay) + `scripts/gate-witness.sh`
(runner). Both are inert unless `KEEL_REPLAY` is set, so a normal gate run is byte-unchanged.

## The two arms (DESIGN.md rule 12 scope)

- **Throughput** — `preport-throughput-949994d.txt` (102 lines). The ceiling/scale/share/
  percent-of-peak rendering (the `GATE_PEAK` consumers), replayed from a **tracked** evidentiary
  archive (`archive/pinned8/bench-gate-p5-6ba6566-keel-skx-20260823T004407Z-1.txt`, Intel 8124M)
  with an evidentiary provenance. Reproduce: `scripts/gate-witness.sh render OUT`. Fully
  self-contained — no host needed.
- **Dispatch + full gate** — `preport-vesta-949994d.txt` (113 lines) from
  `gate-p5-dispatch-corpus-949994d.tar.gz` (a live vesta.local record: the parallel-correctness
  run, the four `KEEL_FORCE` runs, the throughput sweep, and vesta's provenance). This is the
  irreproducible "before" arm — captured at the pre-port revision because the dispatch section's
  markers are a function of the pre-port code, which unit 2 changes. Reproduce: unpack the tarball
  into a dir `D` and run `KEEL_REPLAY=replay KEEL_REPLAY_DIR=D KEEL_REMOTE_HOSTS=vesta.local
  bash scripts/gate-p5.sh` through the normalizer.

Both arms are **deterministic** (two replays → zero diff after scrubbing the two run-specific
fields: the `RUN_STAMP` timestamp and the live disk-headroom reading — neither is gate code) and
the witness is **non-vacuous** (`scripts/gate-witness.sh selfcheck`: a planted `GATE_PEAK=Peak/avx2`
moves the rendering, so a zero-diff is a finding, not silence).

## Not witnessed here (covered elsewhere, or out of scope)

The local builds/vet/lint, the local scalar/simd test runs, and the `-race` leg are skipped under
replay: they are deterministic-or-timing-noisy, they build/test the Go tree rather than exercise
the arch selection the port changes, and skipping them keeps a replay to seconds. They render
identically in a real run before and after the port by construction (the port edits gate scripts,
not Go code).

## Using it for unit 2/3

1. On the pre-port tree, the baselines above are the expected rendering.
2. Make the port edits (uncommitted, so HEAD stays `949994d` and the git-sha in the rendering does
   not move).
3. Re-render each arm; `diff` against the baseline. **Zero diff = byte-unchanged on amd64**, stated
   in the commit. A non-zero diff is the port touching amd64 rendering — stop and localize it.
