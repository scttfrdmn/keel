<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Cross-ISA positioning: what keel's amd64 and arm64 numbers say together

A synthesis from **archived** measurements, no fresh run. Its job is to place keel's
first arm64 (NEON) results beside its amd64 (AVX-512) results on a common footing, and
its main product is a **denominator correction**: the number most likely to be quoted
across ISAs — keel's percent-of-OpenBLAS — moves with the *reference's* per-µarch kernel
coverage, not with keel's own kernel quality. Read on the right axis, keel's per-lane
kernel efficiency is a normal member of the per-host spread on both ISAs; the raw
"31% of OpenBLAS on Graviton3" is not keel being slow.

## The axis that is NOT a kernel-quality metric, and why it matters

`share/Sgemm` (the gate-p5 criterion, `host-baselines.tsv`) is **single-thread Sgemm ÷
that host's own measured 8-thread ceiling** (`gate-p5.sh:810`). It is a *scaling* number
— how much of an 8-core ceiling one core reaches on the same silicon — and it is
memory-system-bound, not kernel-bound. Its archived values invite exactly the wrong
cross-ISA table:

| µarch | `share/Sgemm` (single ÷ own 8-thread ceiling) |
|---|---|
| Skylake-X (amd64) | 30.80 |
| Neoverse-V1 (arm64) | 52.20 |
| Neoverse-V2 (arm64) | 37.60 |

Taken as "kernel quality" this reads *arm64 ahead of amd64* — which is meaningless: it
says single-core scales to a larger fraction of the 8-core ceiling on Graviton, a fact
about the two machines' memory systems, not about either kernel. **Any cross-ISA claim
must name its denominator, and `share` is not the one.** The two axes below are.

## Axis 1 — percent-of-peak: the clean cross-ISA kernel-quality number

percent-of-peak divides by the *host's own* FMA compute ceiling, so it is uncontaminated
by any external reference. It is keel's L1-resident microkernel rate as a fraction of that
host's measured peak (the gate-p2/p3 numerator, packed panels, `BenchmarkKernel`).

| ISA | µarch (host) | best-tile percent-of-peak | source |
|---|---|---|---|
| amd64 (AVX-512) | Zen 4 (vesta) | 96.6% (4×32) | KERNEL.md §7 |
| amd64 (AVX-512) | Zen 5 (antares) | 64.2% (4×32) | KERNEL.md §7 |
| amd64 (AVX-512) | Skylake-X (janus) | 46.0% (2×32) | KERNEL.md §7 |
| arm64 (NEON) | GB10 Grace (castor) — **8×8, pre-tile-fix** | 32.8% | #137 item-3 replay |
| arm64 (NEON) | GB10 Grace — **4×16, post-fix (extrapolated)** | ≈40.6% | 32.8% × 1.237 (see below) |

Read this correctly: percent-of-peak already normalizes for lane width (peak = clock ×
lanes × FMA-ports), so NEON's four lanes against AVX-512's sixteen are *already in the
denominator*. On that normalized axis, keel's post-fix arm64 kernel (~40%) sits just below
Intel Skylake-X (46.0%) and well inside the amd64 spread (46–96%) — a normal member of the
per-host range, not a 2–3× outlier. Zen 4's 96.6% is a µarch fact (double-pumped AVX-512
over 256-bit datapaths, KERNEL.md §5), not a keel-vs-keel gap. The `neon-probe` states the
irreducible part plainly: per lane of work amd64 is ~2.24× more instruction-efficient
because a `Float32x16` FMA does four times a `Float32x4`'s work — that is the ISA, not a
miss, and percent-of-peak is where it belongs.

## Axis 2 — percent-of-OpenBLAS: a statement about the reference, not (only) keel

percent-of-OpenBLAS is keel's full-GEMM (2048³, single-thread) rate over the same host's
source-built OpenBLAS 0.3.29, pinned to the fastest swept coretype. It is the gate-p3
"mission ratio" (≥60% floor). It is the number most likely to be quoted — and the most
misleading across ISAs, because its denominator's quality varies by silicon.

| ISA | µarch | keel % of same-host OpenBLAS | what the reference is |
|---|---|---|---|
| amd64 | Skylake-X (c7i.48xlarge, SPR) | 51.0% | AVX-512 kernel — strong |
| arm64 | Neoverse-V1 (Graviton3) — **8×8, pre-fix** | 31% | **V1 SVE kernel — strong** |
| arm64 | Neoverse-V2 (Graviton4) — **8×8, pre-fix** | 54% | **no V2 kernel — SVE-V1 fallback, weak** |

The V1/V2 inversion is the whole point. keel's *absolute* NEON rate is higher on V1 than
V2 (V1 is the faster core), yet its percent-of-OpenBLAS is **lower** on V1 (31%) than V2
(54%) — because OpenBLAS 0.3.29 ships a competent V1 SVE kernel (~78.8 GFLOP/s reference)
and **no** V2 kernel at all (V2 falls back to the slower V1-SVE codepath, ~35 GFLOP/s
reference; `docs/graviton-sve-neon.md`). So:

- **"31% on V1" is not keel being slow — it is OpenBLAS's V1 reference being strong.**
- **"54% on V2" is not keel being fast — it is OpenBLAS having no V2 kernel.**

The raw ratio tracks the reference's per-µarch coverage. amd64's 51% is a "strong
reference" case like V1; there is no amd64 analog of V2's coverage gap. This reframes the
sub-60% p3 verdict from "keel is behind" to "keel is measured against a reference whose
strength varies by silicon, and on V1 that reference is a hand-tuned SVE kernel keel has no
SVE intrinsics to answer yet (#162)."

## The headline

On the axis that isolates keel's own kernel (percent-of-peak), keel's NEON GEMM is a normal
member of the cross-ISA efficiency spread — post-fix ~40% of peak, beside Skylake-X's 46%.
On the axis that most people quote (percent-of-OpenBLAS), keel's arm64 number swings 31–54%
**on the reference's µarch coverage, not on keel** — high where OpenBLAS lacks a kernel, low
where OpenBLAS ships a strong SVE one. The remaining honest gap on V1 is the SVE kernel keel
cannot yet write (upstream #79781 / keel #162) and the by-element FMLA / broadcast-fold lowering
(#163) — both upstream, both tracked. Nothing here is keel arm64 kernel quality left on the floor
beyond the tile-selection +33% already landed (`972ee47`).

## What this is NOT (§5 rule 12)

- **Not a fresh judged result.** It is a positioning synthesis over archived measurements
  from different campaigns, hosts, and eras. No run was made for it.
- **The amd64 numbers are certificate-era** (pinned8, `969c360`/`6ba6566` for the SKX
  baselines; the SPR OpenBLAS ratio is `#104`'s `c7i.48xlarge`). The arm64 numbers are
  **8×8-era** (`029e24f` / #137), which **predates the tile fix `972ee47`**. Every arm64
  cell above is labeled pre-fix for that reason.
- **The post-fix direction is known but not re-judged.** On GB10 (castor, pinned to an
  X925 performance core) the shipped 4×16 tile measured **61.0 GFLOP/s vs 8×8's 45.7 —
  +33% at full 2048³ Sgemm, +23.7% at the isolated kernel** (#136 pollux sweep, 21.38 vs
  17.29 GFLOP/s at kc=512). The percent-of-peak extrapolation above applies the kernel-level
  +23.7%; the percent-of-OpenBLAS cells would move by roughly the full-GEMM +33% (V1 ≈31→41%,
  V2 ≈54→72%). These are *extrapolations from a characterization host*, not re-measurements.
- **percent-of-peak and percent-of-OpenBLAS are measured at different levels** — the former
  is the L1-resident microkernel (packing/blocking excluded, P2's subject), the latter is the
  full blocked 2048³ GEMM. They are not interchangeable and are never mixed in one column here.
- **GB10 percent-of-peak carries the boost-clock caveat** (`docs/graviton-sve-neon.md`): the
  Grace CPU has no cpufreq governor and boosts above the cpuinfo clock under load, so its peak
  *denominator* is less stable than a governed Graviton's. The judged tier is Graviton
  (homogeneous Neoverse, stable clock); GB10 is characterization.
- **A clean post-fix cross-ISA table is a future re-measure** — a Graviton relaunch (AWS
  spend) landing post-`972ee47` judged percent-of-OpenBLAS and percent-of-peak. Worth it only
  if the tag wants the current cross-ISA number; the finding here (denominators, reference
  coverage) does not depend on it.
