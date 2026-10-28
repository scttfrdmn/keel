# OpenBLAS on Graviton5: v0.3.34's corename, and the armv8sve → neoversev2 gap (#196)

Scott's ruling (2026-10-27): the reference on a host is the **newest OpenBLAS whose runtime
detection names that core**. c9g gets v0.3.34 via `KEEL_OPENBLAS_VERSION`, because upstream
maps Neoverse-V3 (0xd84) to its NEOVERSEV2 kernels from v0.3.33 on. The existing fleet stays
on v0.3.29. To do: measure `armv8sve` against `neoversev2` once on c9g, so the gap is a number.

**Instrument.** One c9g.4xlarge, on-demand us-east-1 (exploration tier), launched outside
keel's fleet under its own name, TTL and teardown, because a gate-p5 run is live on the
fleet at the same time. It is provisioned by `scripts/provision-openblas.sh` with
`KEEL_OPENBLAS_VERSION=v0.3.34`, from this branch's worktree. The gap is then measured with
**one library and one binary**: provision's own `bench-ob.test`, `BenchmarkOpenBLAS/n=2048`,
with `OPENBLAS_CORETYPE` forcing `ARMV8SVE` or `NEOVERSEV2`. Rounds alternate, 10 rounds,
pinned to core 2, `GOMAXPROCS=1`, `OPENBLAS_NUM_THREADS=1`. A forced coretype is the same
kernel family the old version selected, without any version-to-version drift. Every reading
prints OpenBLAS's own corename, which witnesses the family that ran.

**Predictions.**
1. provision's read-back under v0.3.34, unforced, names **corename=neoversev2**, and the
   allowlist accepts it. **REFUTED if** it names anything else.
2. The forced runs print `armv8sve` and `neoversev2` respectively. If a forced run prints
   the other name, the override did not take, and that pair is void.
3. `neoversev2` is faster than `armv8sve` at Sgemm n=2048, one thread. No magnitude is
   predicted: the gap is the deliverable.
