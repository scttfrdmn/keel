# keel — a float32 BLAS subset in pure Go on `GOEXPERIMENT=simd`

*Design document and Claude Code build plan. Working name "keel" (the foundation member of a hull) — rename at will.*

---

## 1. Mission

A pure-Go, cgo-free float32 BLAS subset whose hot kernels are written against the experimental `simd` / `simd/archsimd` packages (Go 1.26+, `GOEXPERIMENT=simd`). Target: ≥70% of single-threaded OpenBLAS SGEMM on AVX-512 hardware, with kernels that inline, respect safepoints, and parallelize on the caller's `GOMAXPROCS` via goroutines.

**Why it wins if it works:** preemption-safe, GC-friendly, deploys as a normal Go module, composes with the host application's scheduler. cgo-OpenBLAS can match the FLOPS but structurally cannot tell that story.

**Non-goals (v0):** float64 (Gonum owns it), complex types, full BLAS coverage, banded/packed storage, ARM64 (later; the architecture must make it a second microkernel, not a second library), float16 (no element type exists in the simd packages).

## 2. Scope: 15 routines, one keystone

| Level | Routines | Notes |
|---|---|---|
| 1 | `Sdot, Saxpy, Sscal, Snrm2, Sasum, Isamax` | Warm-up; validates shim + test harness |
| 2 | `Sgemv, Sger` | One packing dimension; rehearsal for L3 |
| 3 core | **`Sgemm`** | The keystone. All effort concentrates here |
| 3 derived | `Ssyrk, Strsm, Ssymm` | Blocked-GEMM + small corner kernels (BLIS decomposition) |

Row-major only in the public API (`float32` slices + explicit `ld` strides), matching Go idiom rather than Fortran heritage. Transpose flags supported; the packing routines absorb them for free.

## 3. Architecture

```
keel/
  keel.go              // public API: Sgemm(m,n,k, alpha, a, lda, b, ldb, beta, c, ldc) etc.
  dispatch.go          // runtime feature detect → avx512 / avx2 / scalar path
  internal/vec/        // THE SHIM. All simd/archsimd imports live here and only here.
    vec_avx512.go      //   //go:build goexperiment.simd && amd64
    vec_avx2.go
    vec_scalar.go      //   pure Go; always compiles; the semantic reference
  internal/kern/       // microkernels (straight-line, one intrinsic per op)
    sgemm_16x4_avx512.go
    sgemm_8x6_avx2.go
    sgemm_scalar.go
  internal/pack/       // panel packing (A: MC×KC col-panels; B: KC×NC row-panels)
  internal/block/      // Goto/BLIS loop nest (NC → KC → MC → NR → MR → microkernel)
  internal/oracle/     // float64 reference implementations for every routine
  internal/spill/      // spill-audit tooling (§6)
  bench/               // benchmark harness + optional cgo OpenBLAS reference (dev-only, build-tagged)
```

**The shim is load-bearing.** The simd API broke between 1.26 and 1.27 and will break again before graduation. Every intrinsic keel uses gets one wrapper in `internal/vec` (`Load16`, `Store16`, `FMA`, `Broadcast`, `HSum`, `MaskLoad`…). API churn is absorbed in one directory; kernels never import `simd` directly. The scalar implementation of the shim is the executable spec — every vector op has a scalar twin, and differential tests hold them equal.

## 4. Phases and gates

Every phase ends in an executable gate: a command that exits 0 or 1. No narrative "it works" — the gate script decides. Commit at every green gate.

### Phase 0 — Toolchain probe & shim (≈1 session)
- Install Go 1.27rc (or latest 1.26.x), verify `GOEXPERIMENT=simd go build` with a `//go:build goexperiment.simd` file importing `simd/archsimd` compiles and runs `X86.AVX512()` detection.
- Build `internal/vec` with the ~12 ops SGEMM needs, all three backends.
- **CRITICAL — read, don't recall:** run `go doc simd/archsimd` and read the package source under `$(go env GOROOT)/src/simd/` before writing any wrapper. The API is weeks old, has already had breaking renames, and any remembered function names are presumptively wrong. Copy exact identifiers from `go doc` output into the shim.
- Differential-test every vector op against its scalar twin on random + edge inputs (NaN, ±Inf, denormals, -0).
- **Gate P0:** shim tests pass under all three backends; `FMA` disassembles to a single `VFMADD231PS` (grep `-gcflags=-S` output). If no fused op exists in the current API, stop and surface it — mul+add is a half-of-peak ceiling and changes the target math.

### Phase 1 — Level 1 + test harness (≈1 session)
- Implement the six L1 routines on the shim; oracle versions in float64.
- Tolerance model (used everywhere): `|keel − oracle| ≤ C·f(n)·ε₃₂·scale`, with `f(n)=n` for dot-like reductions, `C≈4` slack for FMA/reassociation differences. Encode as one helper; never hand-pick per-test epsilons.
- Property tests: aliasing (`Saxpy(a, x, x)`), zero-length, stride ≠ 1, NaN propagation per BLAS convention.
- **Gate P1:** all L1 tests green on avx512 + scalar; benchmarks show ≥4× scalar for `Sdot` at n=4096.

### Phase 2 — Microkernel + spill audit (THE go/no-go, ≈1–2 sessions)
- Write `sgemm_32x6_avx512.go`. Tile choice: with 16-lane `Float32x16` vectors along M, a 16×4 tile gives only 4 accumulators — too few to hide FMA latency (~4-cycle latency × 2 ports wants ≥8–10 independent chains). Start at **MR=32, NR=6**: 2 vectors along M × 6 columns = **12 accumulators** + 2 A-registers + 1 B-broadcast = 15 live zmm, comfortable for the allocator. If the spill audit stays clean, grow toward MR=32, NR=12 (24 accumulators + 3 = 27 live zmm, near the practical ceiling); grow until spills appear, back off one step, record the frontier in `KERNEL.md`.
- Kernel-shaping rules: pre-slice packed panels to exact length outside the loop (bounds-check elimination — verify with `-gcflags=-d=ssa/check_bce`), pointer-free data (float32 slices ⇒ no write barriers), no function calls inside the K-loop, K-loop unrolled ×4.
- **Spill audit tooling** (`internal/spill`): a small Go program that runs `go build -gcflags=-S` on the kernel package and counts vector stores/loads to stack-relative addresses inside the K-loop body. Also emit `GOSSAFUNC=Kernel32x6 go build` and archive `ssa.html` per commit for human/CC inspection of *why* any spill happened.
- **Gate P2:** `spill-audit` reports **0 accumulator spills** in the steady-state K-loop, AND the raw microkernel (packed inputs, no blocking) hits **≥55% of measured peak**. If either fails after kernel-shaping and one shrink step: STOP, write `docs/spill-report.md` with the ssa.html evidence, and surface the decision (file upstream vs. avo fallback vs. accept AVX2 shapes). Do not silently proceed to Phase 3 on a broken kernel.
- **Peak is measured, not derived** (revised 2026-08-11, decision on issue #11). The original formula — cores·freq·2 FMA ports·16 lanes·2 flops — assumes two full-width 512-bit FMA units. Zen 4 has none: it double-pumps AVX-512 over 256-bit datapaths and retires one 512-bit FMA per cycle, so the formula overstates that host's ceiling by 2× and would turn a 55% floor into a demand for ~110% of the silicon. The denominator is instead `BenchmarkPeak` in `bench/`: `internal/vec`'s register-only FMA-saturation kernels (no memory in the loop, ≥8 independent accumulator chains, distinct chain starts so CSE cannot merge them), measured per host under the §5.4 methodology. The formula survives as a **printed cross-check only**; a measured/formula divergence of ≥1.5× is the double-pump signature and is reported as expected rather than as a fault. This keeps the 55% floor meaningful — it is 55% of what the machine demonstrably does — without hardcoding a microarchitecture table, and it carries to Zen 5 and to a future arm64 port unchanged.

### Phase 3 — Packing + blocking → full SGEMM (≈2 sessions)
- Goto three-level blocking: NC (B panel → L3), KC (depth → L2 residency of A panel), MC (A panel → L2), then MR×NR microkernel over L1. Initial params for a Zen4/Ice Lake-class target: KC=384, MC=144 (multiple of MR), NC=4096 (multiple of NR); make them vars, auto-tune later.
- Packing routines SIMD-accelerated through the shim; transpose flags handled here.
- Edge kernels: masked loads/stores for M%MR, N%NR remainders (the simd API's mask support determines whether this is masked ops or a scalar fringe — read the API, then choose; scalar fringe is acceptable at these sizes).
- Beta handling (β=0 ⇒ no C read; β=1 ⇒ no multiply) as kernel variants, not branches in the loop.
- **Gate P3:** full `Sgemm` matches oracle across a size sweep (1..17, 63,64,65, 500, 1000, 2048, plus transpose/beta/alpha combinations); single-thread ≥60% of OpenBLAS at 2048³ (bench harness pulls OpenBLAS via build-tagged cgo, dev machine only, never a package dependency).

### Phase 4 — Level 2 + derived Level 3 (≈2 sessions)
- `Sgemv` (both transposes), `Sger`.
- `Ssyrk`, `Ssymm` as blocked GEMM with triangular masking in the C-update; `Strsm` as small unblocked triangular solves at the diagonal + GEMM rank-updates elsewhere (the BLIS recipe — cite it in code comments so future readers know the shape is standard).
- **Gate P4:** all routines green vs oracle incl. upper/lower × trans × unit-diag lattice for `Strsm`; `Ssyrk` ≥85% of `Sgemm` GFLOPS at same size.

### Phase 5 — Parallelism, dispatch, polish (≈1–2 sessions)
- Parallelize the MC (ic) loop over a bounded worker pool sized by `runtime.GOMAXPROCS(0)`; shared packed-B panel per NC iteration, per-worker packed-A buffers from a `sync.Pool`. No background threads, no state between calls.
- Runtime dispatch finalized: `avx512 → avx2 → scalar`, overridable by env `KEEL_FORCE=scalar` for testing.
- The package must compile and pass (scalar path) *without* `GOEXPERIMENT=simd` — build tags make the fast paths additive. This is what lets it exist on proper `go get` terms the day the experiment graduates.
- Scaling benchmark, README with honest numbers, `doc.go`.
- **Gate P5:** ≥6× single-thread throughput at 8 cores on 4096³; race detector clean; `go vet`/`golangci-lint` clean; scalar-only build green on stock toolchain.

## 5. Testing philosophy (for the whole project)

1. **Oracle-driven:** float64 reference for every routine; the tolerance helper is the only place epsilons live.
2. **Differential across backends:** avx512 vs avx2 vs scalar on identical inputs must agree within reassociation tolerance — catches shim bugs independent of the oracle.
3. **Adversarial shapes first-class:** every dimension in {0,1,MR−1,MR,MR+1,…}; strides > width; overlapping aliases where BLAS defines behavior.
4. **Benchmarks are tests:** perf gates run in CI-mode locally (`go test -run=NONE -bench=Gate`) and the gate script parses the numbers. A silent 2× regression should fail a gate, not await a human.
5. **Gate benchmark methodology** (revised 2026-08-11, decision on issue #14): `-count=10 -benchtime=1s`, at least one measuring host under the `performance` governor, aggregated by `benchstat` (pinned as a module tool, so the version is in the repo). The gate compares **medians**, and a threshold counts as cleared only if it is cleared **net of benchstat's reported confidence interval** — numerator down by its CI, denominator up by its own. A distribution too wide for benchstat to bound is a failure to measure, not a pass. `-benchtime=3x` remains acceptable for smoke runs that inform no gate. Rationale: P1's first ratios came from `-benchtime=3x -count=5` reduced by min-of-samples, whose raw samples spread by 3.08× inside a single backend; ten one-second runs collapse that to low single-digit percent, and "clear the bar net of CI" means no gate can go green on a lucky run. Mechanics live in `scripts/bench.sh`, shared by every gate so none can quietly deviate.

## 6. Known risks & standing orders

| Risk | Standing order |
|---|---|
| simd API churn (1.26→1.27 already broke) | All imports in `internal/vec`; on toolchain bump, fix the shim only, rerun differential tests |
| Register allocator spills accumulators | Phase 2 gate + shrink protocol; never work around silently; produce `spill-report.md` with ssa.html |
| No FMA intrinsic / wrong lowering | Phase 0 gate; disassembly grep is the arbiter, not the doc comment |
| CC hallucinating the archsimd API | Mandatory `go doc` + GOROOT source read before shim edits; identifiers copied, never recalled |
| Bounds checks / write barriers in K-loop | `ssa/check_bce` in the spill-audit run; pointer-free kernel data |
| Perf numbers flattering on one machine | Bench harness records CPU model, frequency governor, and the host's *measured* peak alongside every result (§4/P2) |
| A wrong percent-of-peak denominator | Peak is measured by a register-only FMA-saturation kernel whose accumulator chains are verified to survive compilation (arithmetic witness) and whose loop is verified free of memory operands (disassembly); the formula is a printed cross-check only |

## 7. Claude Code kickoff prompt

Paste below into a fresh CC session in an empty repo. It is self-contained; it does not assume this document is present (but commit this document as `DESIGN.md` in the repo root first — the prompt tells CC to read it).

---

You are building **keel**, a pure-Go float32 BLAS subset using Go's experimental SIMD support. Read `DESIGN.md` in the repo root completely before writing any code — it defines architecture, phases, and gates. Rules of engagement:

1. **Toolchain:** Go 1.27rc (or newest 1.26.x) with `GOEXPERIMENT=simd`. Verify with a smoke build before anything else. All Makefile targets must set the env var; the scalar path must additionally build on a stock toolchain without it.
2. **Never recall the simd API — read it.** Before writing or editing anything in `internal/vec`, run `go doc simd/archsimd`, `go doc simd` and read the sources under `$(go env GOROOT)/src/simd/`. The package is experimental and has had breaking renames between releases; any identifier you remember from training is presumptively wrong. Copy exact names from `go doc` output.
3. **Work the phases in order (P0–P5 in DESIGN.md). Each phase ends with its gate script under `scripts/gate-pN.sh` exiting 0.** Write the gate script at the *start* of the phase, then make it pass. Commit on every green gate with message `P<N>: <summary> [gate green]`. Never begin phase N+1 with a red gate.
4. **Phase 2 is a go/no-go, not a hurdle.** If the spill audit or the 55%-of-peak floor fails after the documented kernel-shaping steps and one tile-shrink, STOP. Write `docs/spill-report.md` containing the ssa.html evidence and the assembly excerpt, and end the session asking for a decision. Do not proceed to Phase 3, do not switch to assembly on your own initiative.
5. **The scalar shim is the spec.** Every op in `internal/vec` exists in scalar form first, with differential tests binding the vector backends to it. A vector op without a scalar twin and a differential test does not merge.
6. **Tolerances come from the helper, nowhere else.** One `tolerance(n int, scale float32) float32` function per the model in DESIGN.md §5. If a test needs a looser bound, change the model with a comment explaining the numerics, never the individual test.
7. **Honesty in benchmarks.** Every benchmark result you report includes: CPU model, the rate against the host's *measured* peak (§4/P2 — never a formula-derived one), and the OpenBLAS reference number from the same machine when the cgo dev-harness is available. If OpenBLAS is not installed, say so and report percent-of-measured-peak only. Never present a number without its denominator, and never present two denominators for the same quantity.
8. **When the toolchain surprises you** (missing intrinsic, unexpected lowering, compiler bug), document it in `docs/toolchain-notes.md` with a minimal repro before working around it. These notes are a deliverable — this project is partly a field report on `GOEXPERIMENT=simd`.

Session cadence: P0 and P1 in the first session if they go smoothly. Start now with the toolchain probe.

---

## 8. After v0 (parking lot, not scope)

- ARM64/NEON microkernel via the 1.27 portable `simd` package — second kernel file, same machinery.
- Auto-tuned blocking params (`keeltune` writing a per-CPU config).
- `Hgemm` storage variant: float16-in/float32-compute via `x448/float16` conversion in the packing step — half-precision as a *packing format*, sidestepping the missing element type.
- The estimator layer above (the sklearn-shaped library) — a different repo that imports this one.
