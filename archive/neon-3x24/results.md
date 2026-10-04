# Results — the NEON frontier shape `3x24 u=2` against the shipped `4x16`

Adjudicated 2026-10-03 against the pre-registration in `prediction.md`, which was
committed at `a8f520e` before any rate existed. Characterization tier: GB10 is not
a judged host and nothing here is citable as a judged number.

## Provenance, so every rate has its denominator (§7 rule 7)

- **Host** `castor.local` — NVIDIA GB10, Linux aarch64 6.17.0-1032-nvidia, 20 cores:
  10x **Cortex-X925** (`0xd85`) + 10x **Cortex-A725** (`0xd87`), per `lscpu`'s MIDR
  mapping. Governor `performance`, load ≤0.31 at submit, pueue `measured` (parallel=1).
- **Placement** `taskset -c 19` (X925, `cpu_capacity` 1024, `cpuinfo_max_freq`
  3.90 GHz) for the headline sweep; `taskset -c 0` (A725, 718, 2.808 GHz) for the
  core-type arm. Affinity read back from `/proc/self/status` in every job.
- **Measured peak**, `BenchmarkPeak/neon`, register-only FMA saturation, n=10:
  **X925 124.20 GFLOP/s**, **A725 44.79 GFLOP/s**. This is the denominator; the
  formula is a cross-check only (§4/P2, #11).
- **No OpenBLAS reference exists on this host** — no cgo harness is built here — so
  every figure below is percent-of-**measured-peak** and nothing else, as §7 rule 7
  requires when the reference is absent.
- **Toolchain** go1.27.1 `GOEXPERIMENT=simd`, cross-compiled `-trimpath` on
  darwin/arm64. One binary for the whole sweep, sha256
  `3937ec9cf71b6e630d26641d8866021a21928bbc5e6cde45df6d5b1bd7b4d16d`.

## Kernel sweep, one binary, `count=10 benchtime=1s`, X925 cpu19

GFLOP/s medians; full-precision bounds via `tools/benchci`. 320 rows = 32 x 10.

| shape | kc=8 | kc=32 | kc=128 | kc=512 | `-S` audit | %-of-peak @kc=512 |
|---|---|---|---|---|---|---|
| **3x24 u=2** | **50.98** | **63.02** | **66.96** | **68.03** | fit, 4.111 ipf | **54.8%** |
| 4x16 (shipped) | 47.52 | 58.80 | 62.52 | 63.34 | fit, 5.000 ipf | 51.0% |
| 8x16 | 41.20 | 51.61 | 55.24 | 56.26 | spill(13) | 45.3% |
| 4x32 | 42.90 | 52.80 | 55.34 | 55.80 | spill(13) | 44.9% |
| 8x12 | 44.99 | 51.46 | 53.50 | 53.72 | spill(5) | 43.3% |
| 8x8 | 36.90 | 44.47 | 46.86 | 46.97 | fit, 5.750 ipf | 37.8% |

The two zero-spill shapes take the top two places and the frontier shape is the
fastest of all six, at every `kc`.

## The four registered predicates

- **P2 (direction) — CONFIRMED.** `3x24` faster at all four `kc`, intervals
  disjoint, **+7.07% to +7.39% net of both CIs**.
- **P4 (positive control) — CONFIRMED.** `8x8` slower than `4x16` at every `kc`,
  disjoint; +34.9% at `kc=512`, beside `972ee47`'s independent full-`Sgemm` +33.5%.
- **P1 (mechanism discriminator) — REFUTED, and informatively.** Registered boundary
  +10.81%, the midpoint of the front-end pole (+21.63%, the instruction ratio) and
  the port pole (~0%, the shapes tying exactly on `MemOpsPerFMA`). Measured **+7.40%**
  — below it. But the port pole is refuted too, so the X925 is **neither**: the
  17.8%-fewer-instruction advantage realizes **34.2%** of a pure issue-bound reading.
- **P3 — sign holds by 0.12 points, mechanism REFUTED.** The series is **non-monotone**
  (+7.28 / +7.17 / +7.10 / +7.40) with a resolvable dip at `kc=128` against ~0.03-point
  CIs. The predicted mechanism was that the gain lives in the steady state and rises
  with `kc`; it is flat over a 64x range, so that is not what carries it. An endpoint
  comparison on a non-monotone series is not scored as a confirmation.

## The discriminator worked — on the OTHER core type

The same two rows on an A725 (cpu0), same binary, same job:

| core | FMA pipes | clock | 3x24 | 4x16 | gain | of the +21.63% bound | %-of-peak |
|---|---|---|---|---|---|---|---|
| X925 | 4 | 3.90 GHz | 68.03 | 63.34 | **+7.40%** | 34.2% | 54.8 / 51.0 |
| A725 | 2 | 2.808 GHz | 25.81 | 21.37 | **+20.78%** | **96.1%** | 57.6 / 47.7 |

So the pre-registered boundary separates the two core types rather than failing:
**the A725 is issue-bound on this loop and realizes 96% of the instruction-count
advantage; the X925 is not and realizes 34%.** One kernel, two mechanisms, measured
in one run. P1 is still refuted as registered — it named `kc=512` on the host the
sweep ran on — but its framework is what made this legible.

## Full `Sgemm/n=2048`: the win survives packing, blocking and the fringe

Three arms, `972ee47`'s pattern, **interleaved over two rounds** rather than run as
three tasks, all on cpu19. n=2048 is the size most hostile to `NR=24`: `4x16` divides
both dimensions exactly while `3x24` pays an M-fringe (682·3+2) **and** an N-fringe
(85·24+8).

| arm | dispatched (witnessed) | n | median | min | max |
|---|---|---|---|---|---|
| A — ships today | `4x16/neon` | 12 | 61.705 | 61.64 | 61.74 |
| B — `3x24`-only build | `3x24/neon` | 12 | 65.905 | 65.81 | 65.94 |
| C — full registry, **unmodified dispatch** | `3x24/neon` | 12 | 65.790 | 65.73 | 65.82 |

- **C vs A: +6.62%.** Ranges fully disjoint (A max 61.74 < C min 65.73).
- **C vs B: −0.17%**, so carrying all six shapes costs essentially nothing against
  the restricted build, and C's marker proves `kern.Preferred` picks `3x24` out of the
  **full** registry with nothing hand-restricted.
- The fringe geometry costs **0.78 points** of the kernel-level 7.40 — 11% of the
  gain, at the worst size for it — not the gain.
- Retention against each shape's own microkernel: `3x24` 96.7%, `4x16` 97.4%. The
  0.7-point difference is the fringe, and both are far above janus's ~77%.
- **Round-to-round control:** every arm reproduces itself to ≤0.023% across the two
  rounds, so host state did not move and the arm differences are the treatment.

## Verdict

**`3x24 u=2` wins on both GB10 core types, at every `kc`, and at full `Sgemm`,
with every interval disjoint.** It is the leanest emittable zero-spill NEON shape,
it ties `4x16` exactly on `MemOpsPerFMA` and beats it on `InsnsPerFMA`, and
`kern.Preferred` selects it from the unmodified registry.

## What this does NOT establish (§5 rule 12)

- **Nothing on Neoverse.** The judged arm64 tier is Graviton (Neoverse-V1/V2), a
  different microarchitecture from both GB10 cores. The gain is bracketed by the two
  core types measured, **[+6.6%, +20.8%]**, and V1/V2 being 4-pipe-class FP makes the
  low end the likelier one — but that is inference, not measurement. No judged host
  ran this.
- **No OpenBLAS ratio**, on this host, for the reason stated above.
- **`8x12` on the A725 was not measured**, so the archived step-4 claim that a shallow
  spill is throughput-free is refuted on the **big** core (where `8x12` is 15.2%
  slower than `4x16`) and untested on the little one.
- **GB10 absolute rates carry a boost/placement context** and GB10 is
  characterization-only, by `docs/hosts.md` and #109.
