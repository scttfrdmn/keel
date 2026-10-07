<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #155 null-change witness — pre-port baseline (rev `949994d`)

The tracked pre-port artifacts for the #155 gate arm64 port (`docs/gate-arm64-port.md`, unit 1).
The port must leave the gate's amd64 rendering byte-identical; this directory is what "identical"
is checked against. Mechanism: `scripts/gate-replay.sh` (record/replay) + `scripts/gate-witness.sh`
(runner). Both are inert unless `KEEL_REPLAY` is set, so a normal gate run is byte-unchanged.

## The two arms (DESIGN.md rule 12 scope)

- **Throughput** — `preport-throughput-949994d.log` (102 lines). The ceiling/scale/share/
  percent-of-peak rendering (the `GATE_PEAK` consumers), replayed from a **tracked** evidentiary
  archive (`archive/pinned8/bench-gate-p5-6ba6566-keel-skx-20260823T004407Z-1.txt`, Intel 8124M)
  with an evidentiary provenance. Reproduce: `scripts/gate-witness.sh render OUT`. Fully
  self-contained — no host needed.
- **Dispatch + full gate** — `preport-vesta-949994d.log` (113 lines) from
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

## gate-p3 (unit 3)

- **Spill-audit source facts** — `preport-p3-local-69d5f21.log` (60 lines). gate-p3's `-S`
  spill audit is a LOCAL cross-compile disassembly (the tool targets amd64 from any host), so
  this arm needs no corpus: replay with an empty corpus renders the local audit (0 spills for
  `Kernel2x32`/`Kernel4x32`, register-only `avx512Peak`, `audited insns/FMA: 2x32 4.625 4x32
  6.250 avx512Peak 2.250`, shapegen frontier) while the remote arm renders unmeasured. Reproduce:
  `KEEL_REPLAY=replay KEEL_REPLAY_DIR=<dir with a probe-*.out> KEEL_REMOTE_HOSTS=keel-skx bash
  scripts/gate-p3.sh` through the normalizer. Deterministic (zero diff) and non-vacuous (a planted
  `GATE_PEAK_FUNC` change moves the audit line). This is unit 3's hardest category — the source-fact
  function names no marker derives — and the port must arch-condition them AND the spill-audit tool
  (`internal/spill`, which hardcodes GOARCH=amd64 and whose classification is amd64-specific per
  `spill.go`). The remote arm (OpenBLAS ratio, coretype sweep) will use a tracked gate-p3 archive.

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

## `p3-arm64-corpus-0b6b649.tar.gz` — the first arm64 replay corpus (2026-10-07, #178/#167)

Recorded by `p3-arm64-corpus-record.sh` (archived beside it here, since a driver is evidence) on **one `c6g.2xlarge`** (Graviton2 / **Neoverse-N1**),
us-east-1, on-demand, 16 minutes, **$0.07**. Unpack to `build/witness-corpus-arm64` and drive:

```
KEEL_REPLAY=replay KEEL_REPLAY_DIR=build/witness-corpus-arm64 \
KEEL_REMOTE_HOSTS=keel-gvt2 KEEL_GOARCH=arm64 bash scripts/gate-p3.sh
```

**Why it exists.** Both corpora above are `avx512`, so no arm64 gate path could be exercised
anywhere without a live arm64 host — which is why `#167` recorded its arm64 rendering as having
"no standing harness", and why `gate-p3`'s candidate-row emission (`#178`) could not be shipped
without putting its first exercise on a `$10/hr` measurement run.

**Why Graviton2 and not a judged part.** The criterion this corpus exists to drive is
`gate-p3`'s arm64 `peak/*` **first-sight** arm, which fires only for a `cpu_model` with no
registered row. Graviton3/4 are `Neoverse-V1`/`V2` and **both carry a
`peak/3x24/neon/kc=128` row at era `ceil8`**, so either would render `registered` and exercise
nothing. `Neoverse-N1` is unregistered — driven through `baseline_state` → `new` →
`peak_bucket` → `first-sight` *before* the launch.

**Nothing in it is a result.** A `c6g.2xlarge` is a partial-size shared guest that `remote.sh`
classifies `correctness`; the run reported `the admission class is unreadable` and rendered that
criterion `UNMEASURED`. **No row from this corpus may ever be landed**, and landing an
`Neoverse-N1` row would also destroy the corpus's only purpose by moving the arm it drives from
`first-sight` to `registered`. A corpus needs *fixed* input, not good input.

**Determinism, measured:** two replays render **74 lines byte-identical**, modulo one
`disk headroom: <N> MiB free` line, which is a varying local fact the gate reports and not gate
rendering. Live `gate-p3` tallied `16 PASS / 0 FAIL / 1 UNMEASURED / 1 BASELINE / 1 REPORTED`;
replay tallies `14 PASS` for the two criteria replay cannot serve (`git archive HEAD`, native
compiles — `gate-replay.sh`'s own documented scope), with the `BASELINE`/`UNMEASURED`/`REPORTED`
counts identical and the first-sight arm reached either way.

**Scope it does NOT cover** (§5 rule 12): the dispatch/forced sections have no recorded host
calls for the arm64 path beyond what this one host produced; the `-race` leg is replay-skipped;
and every number in it is a Graviton2 number, so it witnesses *rendering and control flow*,
never performance.
