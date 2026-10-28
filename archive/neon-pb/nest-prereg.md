# The nest witness for pre-broadcast A: promotion rule and predictions (before the run, #198)

Kernel-level selection is done (`README.md`). Promotion to dispatch needs a full-GEMM
witness, as 3x24's did (`../neon-3x24/results.md`, +6.62% at Sgemm n=2048): the
pre-broadcast A block is 4× (float32) or 2× (float64) the bytes, and it is expanded once
per packed block.

**Instrument.** `BenchmarkBlocking` (float32) and `BenchmarkBlocking64` (float64): one
2048³ Gemm through `internal/block`'s nest, with `GOMAXPROCS=1`, pinned to core 2. The
shipped kernel and the three pre-broadcast candidates run in the same binary and process.
Hosts are c7g, c8g and c9g 4xlarge (Neoverse V1/V2/V3), in 5 interleaved rounds. Grid:
- float32: KC ∈ {256, 384}, MC ∈ {36, 72, 144}, NC = 2048, for 3x24/neon, 5x16/neon-pb,
  4x20/neon-pb and 7x12/neon-pb. MC = 36 holds the pre-broadcast A block at the shipped
  block's bytes (144/4).
- float64: KC ∈ {256, 512}, MC ∈ {72, 144}, NC = 2048, for 3x12/neon, 5x8/neon-pb,
  4x10/neon-pb and 7x6/neon-pb.

The shipped baseline is the shipped kernel at the shipped blocking: 3x24 at KC=384,
MC=144; 3x12 at KC=512, MC=144.

**Promotion rule, per element.** The candidate point is the (pre-broadcast kernel, KC, MC)
with the highest geometric mean over the three hosts of its median GFLOP/s. It is
promoted if, **on every host**, its worst round beats the shipped baseline's best round.
Within 1% of the top geomean, the point at the shipped KC and MC is preferred, then the
larger MC. If the chosen MC or KC differs from the shipped value, that is a blocking
change for pre-broadcast kernels only (amd64's blocking must not move), and it lands with
this log as its witness.

**Predictions.** The candidate point beats the shipped baseline by at least 25% on every
host at both elements. The basis: the kernel-level gain is about 1.8× (81–86% of peak
against 45.5–46.4%), and the nest retained 96–97% of 3x24's kernel rate at n=2048.
**REFUTED if** no pre-broadcast point beats the baseline on all three hosts. That would
mean the larger A block costs more in the nest than the kernel gains, and the shaping step
fails at the routine level even though it clears P2's kernel-level floor.
