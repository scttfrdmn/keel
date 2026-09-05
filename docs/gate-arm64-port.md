<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# #155 execution spec: the gate's arm64 port, under the null-change constraint

The #137 fleet layer is arm64-aware and proven live; the gates are not. The first live `gate-p3`
run (2026-09-04) showed the gap and this file is the plan to close it. It exists so the port is
executed against a spec, not re-derived — every design decision below was established from the
code and from a local NEON run, not assumed.

## The constraint (Scott's ruling, #155)

The port is a **null change on amd64**: the certificate machinery has landed verdicts and
registered baselines on the amd64 fleet (era pinned8), so its rendering must be byte-identical
before and after. The arm64 branch is added *beside* the amd64 path, never *through* it, and
proven — not asserted — by re-driving the ported gate against tracked amd64 archives and diffing
the renderings to zero (see **Unit 1**).

## Why it is more than a constant swap

The `avx512` hardcoding is in benchmark **selection**, spread across ~30 surfaces in
`gate-p2.sh`/`gate-p3.sh`/`gate-p5.sh`. Three distinct categories, and they get three different
treatments:

1. **Structural — read from the run's own markers (the #153 pattern).** The backend name, the
   kernel IDs and the dispatch chains are already derived in Go from the registered kernels and
   emitted as markers: `keel-bench-kern: <tile>/<backend> (available: …)`,
   `keel-bench-peak-formula: <backend>: …`, `keel-p5-dispatch: l1=… kern=…`. The gate should read
   the backend WORD from these rather than hardcode `avx512`. This is byte-unchanged on amd64 for
   free: the marker says `avx512` there, so the rendered verdict text is identical, and says `neon`
   on arm64. Every `"avx512"` in a *verdict string* or a *row selector* (`GATE_PEAK="Peak/avx512"`,
   `AVX512_GREEN`, the "sweep ran green with the avx512 Sgemm" lines) is this category.
2. **Asserted expectations — arch-conditional constants, NOT derived.** `P5_L1_CHAIN`,
   `P5_KERN_CHAIN`, `P5_FORCED`, `P5_L1_ONLY` are the gate's *independent statement* of the #40
   dispatch ruling; deriving them from `KernChain()`/`L1Chain()` would make the check tautological
   (both sides from one Go source). These stay hardcoded but keyed on the target arch, amd64 branch
   equal to today's values verbatim:
   - amd64: `L1=avx512,avx2,scalar` `KERN=avx512,scalar` `FORCED="scalar avx2 avx512"` `L1_ONLY=avx2`
   - arm64: `L1=scalar` `KERN=neon,scalar` `FORCED="scalar"` `L1_ONLY=""`
   **arm64 forces only `scalar`**: `KEEL_FORCE=neon` would panic — `selectL1` has no NEON Level-1
   backend (#154), so a uniform force is impossible. The NEON *microkernel* is verified from the
   default (unforced) run's `kern=neon,scalar` marker instead of by forcing it.
3. **Source facts — the `-S` spill-audit function names.** `PEAK_FUNCS="avx512Peak,…"`,
   `KERN_FUNCS="Kernel2x32,…"`, `GATE_PEAK_FUNC` name Go symbols disassembled locally with
   `-gcflags=-S`; on arm64 they are `neonPeak` and the #136 tiles (`Kernel8x8`, `Kernel4x16`, …).
   Not in any marker. Either reflect the name off `PeakKernel.Run`/the kernel registry (structural,
   preferred) or an arch-conditional block (amd64 verbatim). This is gate-p2/p3 only.

## Structural differences that are not just names

- **VECTOR_GREEN, not AVX512_GREEN.** The parallel-correctness + dispatch section keys on a host
  running green with a *vector* microkernel. Generalise the test from `active == */avx512` to
  `active != */scalar` (arch-agnostic, structural) and read the backend word from the marker for
  the message — byte-unchanged on amd64, correct on arm64.
- **The theoretical-peak formula needs Linux `cpuinfo_max_freq`.** The dev M-series has no such
  file, so `keel-bench-peak-formula` reports `unavailable` locally — but the **measured** peak
  (`BenchmarkPeak/neon`) runs fine. Judged percent-of-peak therefore still needs Graviton (for the
  denominator); every SELECTION branch fires locally without it.
- **`KEEL_GOARCH=arm64` is env, not code.** `remote_build_test` already honours it (the #136 GB10
  sweep used it); the gate builds `$BENCHBIN` once, so a uniform-arch fleet just needs the env set.
  A mixed-arch fleet would need per-host builds — out of scope for #137's all-arm64 fleet.

## Units (execute and verify in order)

1. **The witness.** Two mechanisms, because the port has two kinds of arch-sensitive rendering and
   the tracked archives only cover one:
   - *Throughput analysis* (ceiling/scale/share, the `GATE_PEAK` consumers): a guarded replay mode
     (inert when unset) that feeds a tracked archive as the per-host `BENCHLOG` and stubs the
     host-touching preconditions (governor, clock, `remote_exec`), so it renders from fixed input.
     The "before" arm is always re-creatable (the archive is permanent), so ordering is free here.
     Deliverable: `render(archive)` identical before/after every later unit, diff zero.
   - *Dispatch/forced/race section*: its logs (the parallel-test run, the `KEEL_FORCE` runs) are
     NOT in the tracked archives, so its witness is a **live amd64 run**, and its rendering is
     deterministic (the markers and PASS/FAIL are functions of the code, not benchmark noise).
     **Capture this baseline at the pre-port revision BEFORE unit 2 touches anything** (Scott's
     ordering ruling): the "before" arm cannot be re-created after surgery without a checkout
     dance, so baseline-first-then-surgery is what keeps the witness cheap. A lab amd64 host
     (vesta/ceres/janus, no AWS spend) captures it.
2. **gate-p5 selection.** VECTOR_GREEN generalisation + arch-conditional expectations (category 2) +
   marker-derived backend words (category 1) + `GATE_PEAK`. Prove byte-unchanged via Unit 1; fire
   arm64 branches locally.
3. **gate-p3 selection + the spill-audit tool.** Two sub-parts, verified against the gate-p3
   witness (`archive/witness/preport-p3-local-*.log`, the source-fact arm; deterministic +
   non-vacuous, needs no host):
   - **(3a) the spill-audit tool arm64 port** (`internal/spill`) — the hardest, source-fact
     category. The tool hardcodes `GOARCH=amd64` (main.go:161-162,222-223) and its
     register/instruction classification is amd64-specific (`spill.go:58-59`). **Read the #13
     NEON-probe comment before designing this** — it records the load-bearing gotcha, the
     **anchor inversion**: amd64 anchors are 1-byte `XCHGL AX, AX` and bare `NOP` is dropped;
     **arm64 anchors are real 4-byte `HINT $0`** (`case ANOOP: return SYSHINT(0)`, size 4,
     `asm7.go`) carrying a source line and MUST be counted, while bare `NOP` must still be
     dropped — the *opposite* of the amd64 rule for the same mnemonic. Getting it wrong
     "silently triples every anchor count" (Scott hit it on the probe; same error class as #46).
     The golden tests (`spill_test.go` pattern) MUST include a NEON listing that fails under the
     amd64 anchor rule applied naively, so the inversion is pinned, not assumed. Arm64 also needs
     its own register-name and stack-reference tables. **Fixture: the 8×12 tile, a known spiller
     (5 spills, #136's NEON sweep)** — an arm64 audit that cannot see those five spills is not an
     audit, so it is the natural positive control.
   - **(3b) the gate-p3 script:** `KERN_FUNCS`/`PEAK_FUNCS`/`GATE_PEAK_FUNC` as arch-conditional
     asserted constants (amd64 verbatim; arm64 `Kernel8x8,Kernel4x16` / `neonPeak,scalarPeak` /
     `neonPeak`), `GATE_PEAK`/`GATE_KERNELS`/filter/`AVX512_GREEN` per the unit-2 treatments, the
     BCE `GOARCH`, and the arm64 coretype sweep + OpenBLAS core allowlist (already landed, #137).
     The remote arm (OpenBLAS ratio, coretype sweep) uses a tracked gate-p3 archive as its corpus.
4. **The campaign.** Respawn c7g+c8g, `KEEL_GOARCH=arm64`, two BASELINE passes → landing →
   confirmation + the SVE≈NEON table (`ob_coretype_sweep` is already arm64-aware) + teardown, per
   `docs/graviton-registration.md`.

## Branch-firing evidence (dev M-series, NEON, no spend, 2026-09-04)

Every arm64 selection target already exists and fires locally under `GOEXPERIMENT=simd`:

```
keel-p5-dispatch: l1=scalar kern=neon,scalar          (TestP5Dispatch PASS)
keel-bench-kern: 8x8/neon (available: 8x8/neon 4x16/neon 8x8/scalar 4x16/scalar)
BenchmarkPeak/neon-12        88.42 GFLOP/s
BenchmarkKernel/8x8/neon/kc=128    14.56 GFLOP/s
BenchmarkKernel/4x16/neon/kc=128   16.38 GFLOP/s
```

So the design is sound: the strings the ported gate selects on (`Peak/neon`, `Kernel/8x8/neon`,
`l1=scalar`, `kern=neon,scalar`) are produced by the shipped NEON build, verifiable without
Graviton. What the dev host cannot supply — the theoretical-peak denominator and the evidentiary
class — is exactly and only what the fleet is relaunched for.
