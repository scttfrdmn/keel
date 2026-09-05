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
   preferred) or an arch-conditional block (amd64 verbatim). **This was scoped "gate-p2/p3 only" and
   that was wrong (#158, surfaced by the #137 campaign):** `gate-p4.sh` has the same source-fact names
   (`KERN_FUNCS`/`PEAK_FUNCS`/`GATE_PEAK`) AND is in gate-p5's carry closure (`gate-p5.sh` runs
   gate-p4, which runs gate-p3). A gate that carries another gate needs its whole carry chain ported,
   and the witness must drive the live **p5→p4→p3** chain on arm64 — not each gate in isolation, which
   is how gate-p4's amd64 hardcodes reached a billed Graviton run.

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
   - **(3b) the gate-p3 script — STATUS: (3a) done, (3b) scoped and pending.** The clean part is
     arch-conditional asserted constants (amd64 verbatim; arm64 `Kernel8x8,Kernel4x16` /
     `neonPeak,scalarPeak` / `neonPeak` / `Kernel/8x8/neon/kc=128 …` / `Peak/neon`), the filter
     per unit-2's treatments, threading `-goarch $KEEL_GOARCH` into `gate-lib.sh`'s
     `carry_p2_properties` (the tool now supports it, 3a) and the BCE `GOARCH`. The arm64 coretype
     sweep + OpenBLAS allowlist already landed (#137). The remote arm (OpenBLAS ratio, coretype
     sweep) uses a tracked gate-p3 archive as its corpus.
     **Two sub-gaps, RULED (a bar travels with its derivation set, never across ISAs — #155):**
     - *criterion 5b* (`SWEEP_BEST_IPF`/`reconcile_sweep_best_ipf`): **arch-gates to REPORTED on
       arm64 — DONE.** `SWEEP_BEST_IPF=4.625` and `shapegen -frontier` are amd64's zero-spill SHAPE
       frontier; running them against NEON would rank arm64 shapes by a frontier they do not
       execute (the rank-inversion defect). On arm64 the criterion renders REPORTED with its cause,
       never judged against a borrowed bar (SPR/GNR precedent). The arm64 shape frontier is a filed
       **v0.2.0 unit** (a `shapegen` arm64 port; #136's audit table is its seed data).
     - *`PEAK_FLOOR=0.55`*: **amd64-scoped; arm64 first-sight renders BASELINE per rule 17 —
       ENCODING PENDING.** An AVX-512-derived percent-of-peak floor on a 4-lane NEON kernel is a
       category error, not a strict bar; the rule-17 machinery already handles this (first sight
       registers, per-host floors type from arm64's own archives at N≥2, 17(c) keeps 0.55 on the
       hosts that derived it) — same treatment gate-p5's 55% received. The encoding arch-gates the
       sentinel + classification block (936-954 and the throughput_verdict, all amd64-bar-scoped)
       to BASELINE on arm64; it is the bounded next step, over gate-p3's most complex criterion.
     Verify against the gate-p3 witness (byte-unchanged amd64) and fire arm64 locally.
     **STATUS 2026-09-05: (3b) code complete; OpenBLAS arm WITNESSED STRUCTURALLY (ruling).**
     Source facts + both rulings encoded and verified (5b→REPORTED, percent-of-peak→BASELINE with
     the 0.55-unreachable assertion, amd64 zero-diff on the dispatch/local-audit and throughput
     arms). The third arm — the OpenBLAS ratio/sweep — is witnessed **structurally, not
     empirically**, by ruling: `git diff 5c1bc32..HEAD -- scripts/gate-p3.sh` shows the section's
     ONLY (3b) change is the dirty-tree guard at line 1104, `elif [[ -z "${KEEL_REPLAY:-}" && -n
     "$(git status --porcelain)" ]]`, which is a **no-op on real runs**: `KEEL_REPLAY` unset →
     `-z KEEL_REPLAY` true → the condition reduces to the original dirty check. The one other diff
     line, `SGEMM_BENCH_FILTER=…neon…`, is inside the arm64-only override block, off the amd64 path.
     So the amd64 OpenBLAS rendering is byte-unchanged **by inspection** — a structural proof (the
     unsafe change cannot reach the amd64 path) beats an empirical corpus that merely matches once,
     the same principle as the 0.55-unreachable assertion, and re-confirming a no-op with a
     five-ssh-point record/replay build is apparatus outgrowing what it serves.
     **This structural proof is valid ONLY for this diff.** A future substantive change to the
     OpenBLAS section expires the eye-proof and REQUIRES the empirical witness — the record/replay
     extension over gate-p3's five direct-ssh points (`ob_preflight`, the `git archive|ssh` ship,
     the harness build ssh, the harness/ratio measure ssh, `ob_coretype_sweep`; 2 are functions, 3
     are inline ssh needing extraction). Filed as a v0.2.0 unit with that inventory as its scoping.
3b-verified (2026-09-05, item 3): a clean arm64 gate-p3 replay matched its pre-registered shape
   exactly — spill audit PASS (NEON fit kernels 0 K-loop spills, neonPeak register-only), 5b
   REPORTED, percent-of-peak BASELINE ("8x8/neon reaches 32.8% of measured NEON peak, RECORDED as
   candidate baseline"), the 0.55 comparison ABSENT (0 lines), OpenBLAS UNMEASURED (first-sight, no
   arm64 OpenBLAS host in local replay — the live ratio/sweep is the campaign's Graviton run). All
   three unit-3 witness arms accounted for: dispatch/local-audit + throughput empirical, OpenBLAS
   structural (#157 files the empirical follow-up). **Unit 3 is complete.**
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
