<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Graviton registration (#137): pre-registered expectations for keel's first judged arm64 fleet

This file is written and committed **before the ladder passes run**, per the campaign discipline
(the same rule `docs/cl2-d2-registration.md` follows): the outcome space of a measurement is fixed
in advance so a surprising reading is adjudicated against a prediction, not rationalised after it.
Everything below is a prediction about what the **instruments render**, at the row granularity they
render it — not a hope about what the silicon does.

## The fleet, as launched and read back

Two full-size on-demand Graviton hosts, `truffle`/`spawn` under `AWS_PROFILE=aws`, 8h TTL, launched
2026-09-04 from commit `64632e8` (arm64 fleet path) with the CPU-model key fixed in `c6e465a`. Both
class **evidentiary** (`host_admission`, verified live), so their perf may be judged.

| host | instance | µarch | CPU-model key | cores | sockets | smt | caches |
|---|---|---|---|---|---|---|---|
| `keel-gvt3` | `c7g.16xlarge` | Graviton3 / Neoverse V1 | `Neoverse-V1` | 64 | 1 | 1 | L1d 64K / L2 1024K / L3 32768K |
| `keel-gvt4` | `c8g.48xlarge` | Graviton4 / Neoverse V2 | `Neoverse-V2` | 192 | 2 | 1 | L1d 64K / L2 2048K / L3 36864K |

`smt=1` throughout: on Graviton a vCPU **is** a physical core, so the `#82` confound that forced the
amd64 exploration fleet onto describe-read sizes does not arise. `governor=absent` (arm64 guests have
no cpufreq knob); §5 rule 5's clock precondition is met by the peak-dispersion instrument over the
sweep, not by a governor read. `c8g.48xlarge` spans **two** Graviton4 sockets, as the amd64
`c7i.48xlarge`/`c7a.48xlarge` already do.

## The registration flow this campaign runs

Both CPU-model keys are **first-sight**: neither appears in `host-baselines.tsv`, in `judged-runs.tsv`,
nor in any bar's derivation set (`SCALE_DERIVED_FROM`, `CEIL_DERIVED_FROM`, both amd64-only). So:

1. **Two ladder passes** at the one frozen revision. The registered-baseline criteria render BASELINE
   (green-compatible, recorded not judged) and each emits a candidate row; two passes give the median
   its `N≥2`.
2. **A landing commit** — a *reviewed* act, never the gate's own — moves the candidate rows into
   `host-baselines.tsv` and writes the `judged-runs.tsv` witness (spending the single-shot per host,
   per era).
3. **A confirmation pass** — now `baseline_state=registered`, so those criteria judge at
   `baseline − margin`. These are keel's **first judged arm64 verdicts**.

## Pre-registered, per criterion (the instrument's own rows)

Structural — these are near-certain and a deviation is a defect, not a surprise:

- **Dispatch marker** (`keel-p5-dispatch`): `l1=scalar kern=neon,scalar` on both hosts. `KernChain`
  derives `[neon, scalar]` from the registered NEON kernels (#136/#153); `L1Chain` is `[scalar]`
  because there is no NEON Level-1 backend yet (#154). A marker that reads anything else — an `avx*`
  token, a `kern=scalar` with no `neon`, an `l1=neon` — falsifies the port's advertised shape.
- **L1 rows**: every Level-1 benchmark dispatches the **scalar** kernel (shipped state, #154). A NEON
  L1 rate would mean #154 landed unrecorded.
- **Conservation**: the scale aggregate's buckets partition the 2-host fleet — the six bucket counts
  sum to exactly 2, residual 0 (`scale_bucket`, the #90/#119 law).

Registered-baseline criteria — **BASELINE on the two ladder passes, judged on the confirmation pass**:

- **scale/Strsm** (`STRSM_FLOOR=6.066×`): both hosts outside `SCALE_DERIVED_FROM` → BASELINE, "no
  registered baseline and no witness row", candidate + witness rows emitted. Confirmation judges at
  `own_baseline − 0.403×`.
- **share/Sgemm, share/Ssyrk, share/Ssymm** (`CEIL_FRACTION=44.2`): both outside `CEIL_DERIVED_FROM`
  → BASELINE, candidate rows emitted. Confirmation judges at `own_baseline − 2.6` points.

Reported-not-judged, fleet-wide (not an arm64 property):

- **p5 ceiling scaling**: **NO FRACTION IN FORCE** — reported against each host's measured 8-thread
  ceiling, no pass/fail, exactly as on every amd64 host right now.

Fixed-floor criteria — **judged from the first pass** (no `baseline_state`; an absolute floor on a
live per-host reference, so nothing about them predates a first-sight host):

- **p3 OpenBLAS ratio** (≥60% of same-host OpenBLAS at 2048³): judged. Reference is the **source-built
  DYNAMIC_ARCH** OpenBLAS 0.3.29, pinned to the fastest swept coretype. *Expected to pass* — but this
  is the first NEON-vs-OpenBLAS-NEON ratio keel has ever measured, so the number is the finding.
- **p3 percent-of-peak** (≥55% of the NEON ceiling from `peak_arm64.go`): judged. *Expected to pass*,
  and this is the load-bearing uncertainty: the 55% floor was **derived on AVX-512 microkernels**. A
  first-sight arm64 FAIL here is therefore a **cross-ISA floor-transfer question** (is 55% the right
  floor for a four-lane NEON kernel?) before it is a keel regression — it is pre-registered as a
  finding to adjudicate, not an automatic P2/P3 STOP. The #136 sweep already showed the shipped tiles
  are zero-spill and competitive, so a gross miss would itself be surprising.
- **p4 syrk/gemm** (≥0.85): judged; co-tenancy divides out (it does not consult admission, by ruling),
  so it is as applicable on Graviton as anywhere. *Expected to pass.*

## The SVE≈NEON deliverable

`gate-p3.sh`'s `ob_coretype_sweep`, on the arm64 list selected by `uname -m`
(`default ARMV8 NEOVERSEN1 NEOVERSEV1 NEOVERSEV2`). The reference is one DYNAMIC_ARCH library; the
sweep forces each family at load time. **ARMV8 is the NEON kernel family; NEOVERSEV1 (Graviton3) and
NEOVERSEV2 (Graviton4) are the SVE families.** The published table is the achieved single-thread
GFLOP/s of each family on each host.

Pre-registered expectation: **SVE ≈ NEON** — the Neoverse SVE kernels do not materially outrun the
ARMV8 NEON kernels on 2048³ SGEMM, so the sweep is *non-discriminating* across the compute-bound
families (the same "identical rate across families" the x86 sweep guards as a broken instrument, here
the expected and reported result). This reproduces #136's on-host finding on rentable silicon. If the
Neoverse family instead wins by a wide margin, that is the more interesting result and the reference
must pin it (the sweep already does), and the SVE≈NEON claim is **refuted** — recorded as such.

## Teardown

Unconditional at campaign end, reconciled against three lists (`aws-fleet.sh down`; the launcher's own
inventory; the provider's running-instances for the tags), per the lesson that three of five instances
once kept billing after a teardown believed complete.
