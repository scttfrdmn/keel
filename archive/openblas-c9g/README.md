# OpenBLAS on Graviton5: v0.3.34 names the core, and the gap is +18.2% (#196, 271e013)

The predictions are in `prereg.md`, committed before the run, and the log is
`probe-c9g-4xlarge-271e013.log`. One c9g.4xlarge (Neoverse-V3, MIDR part 0xd84), on-demand
us-east-1, exploration tier, launched outside keel's fleet. It took about 7 minutes, and the
instance was verified terminated.

**1. v0.3.34 names the core.** provision's unforced read-back says `OpenBLAS 0.3.34
DYNAMIC_ARCH NO_AFFINITY neoversev2 ... corename=neoversev2 threads=1`, and the gate's
allowlist accepts it. **Held.**

**2. The override took in all 20 forced runs.** Every `OPENBLAS_CORETYPE=ARMV8SVE` reading
printed `corename=armv8sve`, and every `NEOVERSEV2` reading printed `corename=neoversev2`.
**Held.**

**3. The gap.** One library and one binary, alternating per round, 10 rounds, pinned to core
2, one thread, Sgemm n=2048:

| arm | median GFLOP/s | range |
|---|---|---|
| OpenBLAS forced `armv8sve` (what v0.3.29 selects on V3) | 73.09 | 72.99–73.18 |
| OpenBLAS `neoversev2` (what v0.3.34 selects) | 86.41 | 86.38–86.46 |
| keel `Sgemm` | 77.19 | 76.80–77.33 |

**`neoversev2` is 1.182× `armv8sve`, with disjoint ranges. Held.** The ruling's reason is
the third row: against the generic reference keel would read 105.6% of OpenBLAS, and against
the reference that names the core it reads 89.3%.

**What this log does not witness:** the keel kernel. The probe grepped for keel's dispatch
marker, and none was printed, so which microkernel produced 77.19 is not established here.
The figure is consistent with `5x16-pb/neon`: the nest witness read 78.58 GFLOP/s on V3 at
n=2048 (`../neon-pb/`). Consistent is not witnessed.
