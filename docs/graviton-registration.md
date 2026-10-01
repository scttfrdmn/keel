<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Graviton registration (#137): pre-registered expectations for keel's first judged arm64 fleet

This file is written and committed **before the ladder passes run**, per the campaign discipline
(the same rule `docs/cl2-d2-registration.md` follows): the outcome space of a measurement is fixed
in advance so a surprising reading is adjudicated against a prediction, not rationalised after it.
Everything below is a prediction about what the **instruments render**, at the row granularity they
render it — not a hope about what the silicon does.

## Amendment 2026-09-05 (before any pass ran): two criteria re-typed to match the #155 rulings

This file was committed at `fa5dca4`, before #155's two cross-ISA rulings landed. Those rulings
(committed in #155, so predating this campaign session — nothing here has been measured yet) changed
the arm64 gate's rendering for two criteria, so the predictions below are corrected to say what the
gate now actually renders (`gate-p3.sh:685`, `:838`, verified by the #155 item-3 local replay).
Importing a standard that predates the run is the legitimate amend; rewriting after a reading is not.
Both changes are the same principle — *a bar travels with its derivation set, never across ISAs*:

- **p3 percent-of-peak** moves from *fixed-floor, judged from the first pass* to *registered-baseline,
  BASELINE on the ladder passes*. Ruling 2 (#155): `PEAK_FLOOR=0.55` is amd64-derived (an AVX-512
  microkernel floor), so on a first-sight 4-lane NEON kernel it is a category error, not a bar. The
  gate renders BASELINE and records the candidate baseline; `throughput_verdict`, where the 0.55
  comparison lives, is unreachable on arm64 by construction. The "is 55% the right floor for NEON?"
  question the original text posed as a finding-to-adjudicate is thereby *answered in advance* by the
  ruling — it is not, so arm64 registers its own floor per rule 17 rather than being judged against a
  borrowed one.
- **criterion 5b** (shape-frontier reconciliation, `SWEEP_BEST_IPF`) is added to *reported-not-judged*.
  Ruling 1 (#155): `SWEEP_BEST_IPF=4.625` and `shapegen -frontier` are amd64's zero-spill SHAPE
  frontier; running them against NEON would rank arm64 shapes by a frontier they do not execute (a
  rank inversion). REPORTED with its cause; the arm64 frontier is a filed v0.2.0 unit (#156).

## The fleet, as launched and read back

Two full-size on-demand Graviton hosts, `truffle`/`spawn` under `AWS_PROFILE=aws`, 8h TTL. The first
launch (2026-09-04, commit `64632e8`) surfaced #155 and was torn down. A second launch (2026-09-05,
`5822b39`) got as far as pass 1 and surfaced **#158** — gate-p4 was never ported to arm64, and
gate-p5 carries it (p5→p4→p3), so the whole chain went RED (empty peak window, audit against absent
amd64 symbols); torn down. This campaign **relaunches at the frozen post-#160/#161 revision `5f73895`**:
all three #155 units plus #158 green, gate-p4 now rendering the arm64 tier under a proven carry-chain
witness (amd64 byte-identical; arm64 audit resolves Kernel8x8/Kernel4x16 zero-spill and neonPeak
register-only). **Amend-with-disclosure (2026-09-05):** #158 does not move any criterion prediction
below — it *enables* them, clearing the peak-window cascade so percent-of-peak/scale render and making
p4 syrk/gemm renderable at all (it was RED-by-cascade before, not judged). #160/#161 (2026-09-06) similarly *enable* — not move — the criteria: the governor-less peak-substitute clock now establishes (scale/ceiling register) and the arm64 L2 differential passes; witnessed end-to-end on a native GB10 corpus, all residual reds being castor-only environment (no OpenBLAS, not evidentiary), which Graviton resolves. The CPU-model key was fixed
in `c6e465a`. The read-back table below is
from the first launch and is **re-verified live on relaunch** (same instance families → same keys and
caches expected; any divergence is disclosed at launch, not assumed). Both class **evidentiary**
(`host_admission`, verified live), so their perf may be judged.

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

Structural — these are near-certain and a deviation is a defect, not a surprise. **Amended 2026-09-09
for the rev being certified, `c220834`:** the predictions below were written for the pre-tile-fix,
pre-L1 rev (#137 era, `5f73895`/`029e24f`) and are corrected to what `c220834` *verifiably dispatches*
— the tile fix (`972ee47`) and the NEON Level-1 backend (`f1bfaa9`) landed since. This is a rev-matched
prediction, not a result-matched one: it is a structural fact about the shipping binary, confirmed from
the dispatch layer without any fleet — `dispatch_ladder_arm64.go:25` reads `L1Chain()=[neon,scalar]`,
and the rendered markers are `l1=neon,scalar kern=neon,scalar` / `keel-l1-active: neon` /
`keel-sgemm-active: 4x16/neon`. The prior prediction (`l1=scalar`, with `l1=neon` as a *falsifier*)
described code that no longer ships; leaving it would make the cert self-falsify on its own marker.

- **Dispatch marker** (`keel-p5-dispatch`): `l1=neon,scalar kern=neon,scalar` on both hosts, with
  `keel-sgemm-active: 4x16/neon`. `KernChain` derives `[neon, scalar]` from the registered NEON kernels
  (#136/#153); `L1Chain` is `[neon, scalar]` because the NEON Level-1 backend now ships (#154, `f1bfaa9`).
  A marker that reads anything else — an `avx*` token, `l1=scalar` (which would mean #154 regressed), a
  `kern` tile other than `4x16/neon` (which would mean the audited-`InsnsPerFMA` tile-rank fix `972ee47`
  regressed), or a `kern=scalar` with no `neon` — falsifies the port's advertised shape.
- **L1 rows**: every Level-1 benchmark dispatches the **NEON** kernel (shipped state, #154 landed at
  `f1bfaa9`). A **scalar** L1 rate would now mean #154 regressed (the inverse of the pre-amend prediction).
- **Conservation**: the scale aggregate's buckets partition the 2-host fleet — the six bucket counts
  sum to exactly 2, residual 0 (`scale_bucket`, the #90/#119 law).

Registered-baseline criteria — **BASELINE on the two ladder passes, judged on the confirmation pass**:

- **scale/Strsm** (`STRSM_FLOOR=6.066×`): both hosts outside `SCALE_DERIVED_FROM` → BASELINE, "no
  registered baseline and no witness row", candidate + witness rows emitted. Confirmation judges at
  `own_baseline − 0.403×`.
- **share/Sgemm, share/Ssyrk, share/Ssymm** (`CEIL_FRACTION=44.2`): both outside `CEIL_DERIVED_FROM`
  → BASELINE, candidate rows emitted. Confirmation judges at `own_baseline − 2.6` points.
- **p3 percent-of-peak** (amended 2026-09-05 — see above; `gate-p3.sh:838`): both hosts first-sight →
  BASELINE, "RECORDED as its candidate baseline, not judged against `PEAK_FLOOR=0.55`" (that floor and
  the issue/fma frontier are amd64-derived), candidate row emitted. A reviewed landing commit types
  the arm64 percent-of-peak floor from the host's own archives; the confirmation pass judges against
  that own-derived floor, not against 0.55. The #136 sweep showed the shipped tiles are zero-spill and
  competitive, so the recorded fraction is the finding, not a hurdle. **Amended 2026-09-09 for
  `c220834`:** the item-3 local replay rendered `8x8/neon reaches 32.8% of measured NEON peak` — but
  that was the pre-tile-fix rev; `c220834` dispatches **`4x16/neon`** (the `972ee47` tile-rank fix), so
  the shape both hosts should reproduce is `4x16/neon`, and the recorded fraction is expected **~40%**
  of measured NEON peak (32.8% × the +24% kernel-level tile delta measured on GB10; directional, from a
  characterization host — the Graviton number is what the run RECORDS). This stays BASELINE with the
  0.55 comparison absent, so the expectation does not gate — it records what it measures — but it is
  stated rev-matched rather than 8×8-stale.

Reported-not-judged, fleet-wide (not an arm64 property):

- **p5 ceiling scaling**: **NO FRACTION IN FORCE** — reported against each host's measured 8-thread
  ceiling, no pass/fail, exactly as on every amd64 host right now.
- **criterion 5b** (shape-frontier reconciliation, `SWEEP_BEST_IPF`; amended 2026-09-05 — see above;
  `gate-p3.sh:685`): REPORTED on arm64, not judged — the amd64 zero-spill frontier does not execute on
  NEON, so ranking arm64 shapes by it is a rank inversion. The arm64 frontier is filed as a v0.2.0
  unit (#156). This one *is* an arm64 property (unlike p5 ceiling scaling), by the ruling.

Fixed-floor criteria — **judged from the first pass** (no `baseline_state`; an absolute floor on a
live per-host reference, so nothing about them predates a first-sight host):

- **p3 OpenBLAS ratio** (`OPENBLAS_FLOOR=0.60`): **REPORTED on arm64, not judged — ruled 2026-09-09,
  from #155's "a bar travels with its derivation set, never across ISAs."** The v0.2.0-readiness
  pre-flight found this floor would red the arm64 gate on Graviton-V1: the 60% bar was derived on amd64
  against an **AVX-512** reference, but on Graviton the swept reference (`neoversev1/v2`) is OpenBLAS's
  **SVE** kernel, and keel's kernel is NEON — archsimd exposes no SVE (#162). So 60%-of-an-SVE-reference
  measures the archsimd ISA-access gap, not keel's kernel (keel reads ~31% of the V1 SVE reference at
  8×8, ~41% post-tile-fix — clearing 60% needs ~+94%, which is the SVE kernel, not a tile choice). This
  is the *same* foreign-reference category error that sent PEAK_FLOOR→BASELINE and criterion 5b→
  arm64-own; the cross-ISA positioning finding (`docs/cross-isa-positioning.md`) is the evidence — the
  raw percent-of-OpenBLAS tracks the *reference's* per-µarch coverage (SVE on V1, none on V2), not
  keel. `gate-p3.sh` now renders the ratio **REPORTED** on arm64 (per-host and aggregate), verbatim
  visible on the certificate and in the docs — the gate stops scoring an ISA-access gap as a keel FAIL,
  it does **not** hide the ratio. REPORTED not registered-baseline: the denominator is a foreign ISA's
  kernel, not keel's own, so there is no keel drift floor to register. **Legislated from principle
  before any log was read**, importing only #155 (which predates this campaign); amd64 keeps the 60%
  floor byte-unchanged (witnessed zero-diff). *(This bullet previously read "stays fixed-floor… no
  ruling moved it"; that was written before the cross-ISA finding and is now corrected — a ruling did.)*
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

## Reconciliation against the certified runs (2026-10-01, #155)

The pre-registration above is now adjudicated against what the instruments actually
rendered. Evidence: `archive/cert-v0.2.0/gate-p5-arm64-49165ca.log` (the certified leg)
and the two independent registration fleets at `ca4b952` (`archive/pinned8/gvt-reg2-p[12]-ca4b952.log`),
all at `-test.count=30` in us-east-1. Three predictions held; one is refuted, and the
refutation is the more interesting result the pre-registration said it would be.

**Dispatch marker — CONFIRMED.** Both hosts render `l1=neon,scalar kern=neon,scalar`, and the
gate's own criterion 7 passes on that, so the `#154` NEON L1 backend and the `972ee47`
tile-rank fix both ship as the amended prediction said. No `avx*` token, no `l1=scalar`.

**OpenBLAS ratio REPORTED, not judged — CONFIRMED.** Three REPORTED lines (per host plus the
fleet aggregate), ratios printed rather than hidden: gvt4 66.1%, gvt3 41.3%.

**percent-of-peak — RECORDED, and the prediction missed high.** Predicted `~40%` of measured
NEON peak for `4x16/neon`; recorded **78.8% on gvt3 (Neoverse-V1)** and **55.3% on gvt4
(Neoverse-V2)**, reproducible to 0.1 points across the independent fleets. The prediction does
not gate — it was explicitly BASELINE-recorded with the 0.55 comparison absent — but the miss is
worth naming precisely, because its basis was `32.8% x 1.24`: a GB10 characterization reading
scaled by a kernel-level tile delta measured on different silicon. A cross-host extrapolation is
directional evidence and was labelled as such; what this shows is that it should not be read as
a point estimate even when it is the only number available. The recorded fractions are the
finding.

**SVE ≈ NEON — REFUTED, and the two hosts answer oppositely.** `ob_coretype_sweep`, best of 30
at `-benchtime=1s`, single-thread 2048^3 SGEMM against one DYNAMIC_ARCH OpenBLAS 0.3.29 with the
family forced at load time:

| `OPENBLAS_CORETYPE` | resolved `corename` | gvt3 / Graviton3 (V1) | gvt4 / Graviton4 (V2) |
|---|---|---|---|
| `default` | `neoversev1` | **79.10** | 63.01 |
| `ARMV8` (NEON) | `armv8` | 63.65 | **71.39** |
| `NEOVERSEN1` | `neoversen1` | 64.07 | 70.92 |
| `NEOVERSEV1` (SVE) | `neoversev1` | **79.18** | 63.02 |
| `NEOVERSEV2` | `neoversev1` | 79.11 | 62.87 |

The pre-registered expectation was that the sweep would be *non-discriminating* across the
compute-bound families. It discriminates on both hosts, by 13-24%, and in opposite directions:

- On **Graviton3**, the SVE family wins by **+24%** over NEON (79.18 vs 63.65), and the library
  already chooses it unpinned — "no cross-family winner beyond drift".
- On **Graviton4**, **NEON wins by +13%** over the family the library chooses (71.39 vs 63.01),
  so the reference is pinned to `ARMV8`.

Two things explain the inversion and both are findings in their own right. First,
`OPENBLAS_CORETYPE=NEOVERSEV2` resolves to **`corename=neoversev1`** on Graviton4: 0.3.29 has no
distinct V2 kernel family, so "SVE on V2" is really V1-tuned SVE kernels running on V2 silicon,
and those lose to NEON there. Second, this is the mechanism behind the ratio asymmetry reported
above — gvt3's reference is a 79 GFLOP/s SVE kernel while gvt4's is a 71 GFLOP/s NEON one, so
keel divides by a materially stronger denominator on Graviton3, which is most of why its ratio
reads 41.3% against gvt4's 66.1%.

So the claim "the Neoverse SVE kernels do not materially outrun the ARMV8 NEON kernels" is
**false on Graviton3** and true-but-inverted on Graviton4. For `#162` (the arm64 SVE microkernel)
this sharpens the case rather than settling it: SVE is worth ~24% on V1, and a V1-tuned SVE
kernel is worth *less than NEON* on V2 — so the lever is real but micro-architecture-specific,
and a keel SVE path would need per-uarch tuning rather than one SVE kernel, which is exactly the
cost the standing task has to price.
