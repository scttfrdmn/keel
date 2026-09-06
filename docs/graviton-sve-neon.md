<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# SVE vs NEON on Graviton: the coretype sweep (#137)

The `#137` pre-registration named this table as a deliverable and pre-registered the
expectation **SVE ≈ NEON** — that the Neoverse SVE kernels would not materially outrun the
ARMV8 NEON kernels on 2048³ single-thread SGEMM. The measurement **refutes that on Graviton3
(Neoverse-V1)** and leaves it muddled on Graviton4 (Neoverse-V2) for a reason that is about the
*reference*, not the silicon.

Method: `gate-p3.sh`'s `ob_coretype_sweep` forces each OpenBLAS 0.3.29 family (one DYNAMIC_ARCH
library, single-thread, `OPENBLAS_NUM_THREADS=1`) at load time on each host; the published number
is the achieved single-thread GFLOP/s. Reproduced across two ladder passes + the confirmation pass
at `029e24f`/`eeffceb`; the run below is `eeffceb`.

| OpenBLAS family | Neoverse-V1 (`c7g.16xlarge`) | Neoverse-V2 (`c8g.48xlarge`) |
|---|---|---|
| `ARMV8` (baseline NEON) | 63.30 | 36.35 |
| `NEOVERSEN1` (N1-tuned, NEON) | 63.27 | 36.45 |
| `NEOVERSEV1` (V1-tuned, **SVE**) | 78.70 | 34.56 |
| `NEOVERSEV2` (V2-tuned, **SVE**) | 78.96 | 34.42 |
| `default` (runtime auto) | 78.66 | 34.57 |

## Findings

- **SVE ≈ NEON is REFUTED on Neoverse-V1.** The V1-tuned SVE families (~78.8 GFLOP/s) beat the
  NEON families (ARMV8/NEOVERSEN1, ~63.3) by **~24%** — a wide margin, not parity. The library's
  runtime auto-detection correctly lands on the V1-SVE kernel (`default` 78.66 ≈ `NEOVERSEV1`).
  This is the more-interesting outcome the pre-registration flagged: recorded as a refutation.
- **On Neoverse-V2, SVE ≈ NEON holds — but the whole reference is anomalous.** Every V2 family
  runs at **~half** the V1 rate (~35 vs ~79 GFLOP/s), and NEON (ARMV8 36.35) is marginally *faster*
  than the SVE families (34.4–34.6). The cause is the reference: OpenBLAS 0.3.29 has **no
  Neoverse-V2 kernel** — `NEOVERSEV2` falls back to the V1-SVE codepath, and the runtime auto-detect
  picks that slower fallback (`default` 34.57 ≈ `NEOVERSEV1`), which is why the coretype sweep pins
  `NEOVERSEN1` (36.45, the fastest available) as gvt4's fair reference. The 2× V2-vs-V1 deficit on a
  newer, wider core is a property of OpenBLAS 0.3.29 on this silicon, not of the hardware.

## Consequence for keel's ratio

keel's NEON Sgemm judged against the fair swept reference is **54.4% on V2 / 31.0% on V1**. The V2
figure reads *higher* only because its OpenBLAS reference is anomalously low (~36 vs ~79); measured
against a V1-class reference rate, keel's arm64 NEON Sgemm sits near the V1 figure. The honest
takeaway is the V1 number — **~31% of a competent OpenBLAS SVE kernel** — and it is the gap NEON
L1/L2 and an arm64 SVE microkernel (#154 and successors) are written to close.
