# `golang/go#80835`: do CL 825185 + CL 825186 change keel's encodings? — janus, 2026-10-09

**No. They change nothing in keel's compiled output, measured.** This archive is the evidence for
retracting the status change posted on keel #144 earlier the same day.

## The experiment

The counterfactual is upstream's own fix, which is what made this measurable without
hand-patching encodings. CL 825185 (`fbea197d3279`) and CL 825186 (`0b2fd4aa9c08`) are both
CPU-feature propagation changes — a block that does not know it has AVX emits legacy-SSE — and
they are **consecutive**: `0b2fd4aa9c08` is exactly one commit ahead of `fbea197d3279`, and that
commit *is* the second CL. So the base was pinned to the later one and **both were reverted from
it**, isolating the pair with nothing else in between.

- host: `janus` (i9-9960X, Skylake-X, bare metal, Rocky 9), idle, 0 active queue tasks at submit
- two toolchains built from one checkout, `VERSION` identical in both arms on purpose — they must
  differ by the two reverts and nothing else, and that string is embedded in every binary
- one keel source tree, shipped by `git archive HEAD` at `e4b6c6e`, built with keel's own line:
  `GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c -trimpath -o … ./bench`
- builds in janus's `build` pueue group; the timing phase was gated behind the witness and
  **never ran**, so the `measured` slot was never spent

## The result

| check | fixed (both CLs) | legacy (both reverted) |
|---|---|---|
| `compile` binary sha256 | `2f5c402e…` | `5a2eecfb…` — **toolchains differ** |
| normalized disassembly sha256 | `428d3575…` | `428d3575…` — **identical** |
| legacy `movups`/`movaps`, three kernels | **36** | **36** |
| legacy `movups`/`movaps`, whole binary | **6416** | **6416** |

396,471 lines of disassembly, byte-identical; the only difference between the two `objdump`
outputs was the **filename in the header**. The revert is real — 136 deleted lines across 4
files, and the two `compile` binaries hash differently — so the toolchains differ and the
treatment simply does not reach this code.

The 36 are unambiguously legacy SSE, e.g. `movups %xmm2,0x210(%rsp)`: the 128-bit spill idiom
`docs/spill-report.md` section 11.3 describes, interleaved among EVEX work. Vector-move census inside
`Kernel6x32|Kernel2x32|Kernel4x32`: 177 `vmovdqu64`, 88 `vfmadd213ps`, 44 `vmovss`,
44 `vbroadcastss`, 36 `movups`. (All 88 FMAs are **213**-shaped, which is `golang/go#80829`'s
accumulate-in-place gap — corroboration of a separate finding, not evidence for this one.)

## What this does and does not establish (§5 rule 12)

- **Establishes:** those two CLs do not alter keel's encodings. keel #144 is **not** "fixed
  upstream, awaiting a toolchain" — go1.28 will ship both CLs and these 36 moves.
- **Does not establish** anything about the encoding's *cost*. No timing was taken, by design:
  the witness gated it, and with no counterfactual available the A/B route to a time delta is
  **moot rather than merely blocked**. Cost would need hardware counters (janus has the PMU
  exposed but no `perf` installed, and whether Skylake-X exposes an AVX↔SSE transition-assist
  event at all is unresolved) or a hand-patched binary.
- **Does not establish** anything about go1.27.1 vs go1.28 as shipped. A tip-minus-two-CLs
  toolchain is one nobody ships; this isolates the two CLs and nothing else.
- **Says nothing about mauri870's boundary case.** Their repro is a missing `VZEROUPPER` at
  function return; this measures interior encodings. Different locations, and this run did not
  build their repro.

## An instrument defect in the run itself, stated rather than quietly fixed

`vzup-phase.sh.AS-RUN`'s `vex_moves()` greps `\bv(movups|movaps|movdqu)\b`, which **cannot match
`vmovdqu64`** — the trailing `64` breaks the word boundary — so the witness table printed
`VEX vmov* = 0` for kernels that contain 177 of them. The copy here is **as run**, not corrected,
because an archive of a corrected script misrepresents the run.

It did **not** affect the verdict: the gate compares the *legacy* counts only
(`[ "$LF" = "$LL" ]`), and `\b(movups|movaps)\b` is correct — `v` is a word character, so that
pattern cannot match `vmovups`. The broken regex produced a misleading informational column and
nothing else. It was caught because a zero VEX count is impossible for an AVX-512 body, which is
the only reason the census above was run instead of the table being published.

## Files

- `driver-vzup-80835-e4b6c6e.log` — the full run: preflight, ship, both builds, the witness, the refusal
- `measurements.md` — every hash and count in the table above, as produced on the host. The
  extension is `.md` and not `.txt` **because `archive/*/*.txt` is a test corpus glob**:
  `tools/benchci`'s `archivedLogs` sweeps it and `TestArchivedIntervalsNeverEscapeTheirSamples`
  correctly refuses a member that re-derives 0 readings. This is the SECOND time that glob has
  been tripped by a non-benchmark `.txt` in one day (`region-probe.txt`, hours earlier), which is
  recorded in `CHANGELOG.md` as evidence that the membership test wants fixing at the glob rather
  than by renaming files a third time.
- `vzup-drive.sh`, `vzup-phase.sh.AS-RUN` — the local driver and the remote phase script as run

Verify with `shasum -c archive/go80835-encoding-ab/DIGESTS.sha256` from the repo root.
