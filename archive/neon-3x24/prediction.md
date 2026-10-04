# Pre-registration — does the NEON zero-spill frontier shape beat the shipped tile?

**Registered 2026-10-03, at revision `6855709`, before any rate was measured on
any host.** Subject: #136's remaining decision. Instrument: `BenchmarkKernel` in
`bench/kernel_test.go`, one binary, all shapes in one invocation.

## Instrument and output space (§5 rule 24)

- **Metric column: `GFLOP/s`**, named in advance. `sec/op` cannot compare these
  shapes — a `3x24` tile and a `4x16` tile do different work per call — and the
  two columns quantize independently at four significant figures each, so a
  predicate that did not name one would be adjudicable two ways.
- **Adjudicated from raw samples at full precision** via `tools/benchci`, never
  from the printed `± W%`.
- `-count=10 -benchtime=1s`, §5 rule 5's methodology, so the readings are
  comparable to everything else in the record.
- **Rows the driver prints: 32**, enumerated by running the filter rather than
  predicted: 6 NEON shapes (`8x8`, `8x12`, `4x16`, `3x24`, `8x16`, `4x32`) × 4
  `kc` (8, 32, 128, 512), plus 2 scalar reference shapes (`8x8`, `4x16`) × 4 `kc`.
  Benchmark names will carry a `-1` suffix, not the dev host's `-12`: Go reports
  the affinity mask's width as `GOMAXPROCS`.

## Judged rows: 4 comparisons, and nothing else

Only the four `3x24/neon/kc=K` against `4x16/neon/kc=K` pairs. The other 24 rows
are **declared out of domain** here, in advance: they are reported with their
intervals and no threshold is set for them, because `8x12`/`8x16`/`4x32` spill
and are excluded from dispatch regardless, and the scalar rows are a correctness
path.

## The prediction, with the boundary derived rather than chosen

Both shapes are **zero-spill**, and they **tie exactly** on `MemOpsPerFMA` —
`1/3 + 4/24 = 1/4 + 4/16 = 0.5`, exactly in float64 — so the memory axis predicts
**parity** and the whole difference is instruction count: 4.111 against 5.000
insns/FMA, audited on the real file by `spill-audit -goarch arm64`.

That makes this measurement a **discriminator between two mechanisms**, and the
boundary between them is arithmetic from the two audited counts, not a number
chosen to be clearable:

- If the X925's **front end** binds this loop, the gain tracks the instruction
  ratio: `5.000/4.111 = 1.21626`, i.e. **+21.63%**.
- If the **FP pipes** bind it, both shapes saturate them, the tie on the memory
  axis is the whole story, and the gain is **~0%**.

**P1 (primary).** At `kc=512` the `3x24` gain over `4x16` lands **above
+10.81%** — the midpoint of those two poles. Above: front-end-bound, and the
frontier metric buys what it claims. Below: port-bound, and 4.111 overstates
what the shape is worth. Either outcome is a result; neither is a failure.

**P2 (direction).** `3x24` is faster than `4x16` at **all four** `kc`, intervals
disjoint. A row whose intervals overlap is `UNMEASURED` for that `kc`, not a tie
awarded to either shape.

**P3 (shape of the gain).** The gain is **larger at `kc=512` than at `kc=8`**,
because `kc` sets the steady-state share of the call and the instruction-count
advantage lives in the steady state. A flat or inverted profile refutes this and
says the advantage is in the prologue/epilogue, not the K-loop.

**P4 (positive control, on known data).** `8x8` is **slower than `4x16` at every
`kc`**. This ordering was measured on GB10 in #136 step 4 (+19% for `4x16`) and
witnessed again at full `Sgemm` in `972ee47` (+33.5%, placement-controlled), so
it is the arm that says this harness still measures the ordering the record
already contains. If P4 fails, nothing else here is readable.

**Reported, NOT judged.** `4x16/kc=512` against pollux's archived 21.38 GFLOP/s.
Deliberately unjudged: that reading is a different GB10 instance, unpinned, at a
different revision, so a band around it would measure the cross-era delta and
call it agreement. It is printed for orientation only.

## What proves each treatment arrived (§5 rule 5 / §5 rule 26)

Under a null-ish prediction an apparatus failure can read as a confirmation, so
each treatment names an observable read off the run itself:

1. **The shape arrived** — the row name carries the tile, and `3x24/neon` rows
   existing at all is the witness. `keel-bench-kern-audit` prints the audited
   counts the registry holds.
2. **Staging held** — `keel-bench-kern:` must still report **`4x16/neon`** as
   dispatched. `3x24` is a `referenceTile`, so a run that reported `3x24/neon`
   dispatched would mean the staging failed and the full-`Sgemm` arm below is
   measuring something other than what this file says.
3. **Pinning arrived** — the job prints its own `Cpus_allowed_list`, and the
   pinned cpu's `cpu_capacity` and `CPU part`. Required: `0xd85` (Cortex-X925).
   The GB10 is big.LITTLE at a ~1.4x capacity ratio, which is the same order as
   the effect under test, so an unwitnessed mask could manufacture the result.
   Both arms are in **one process on one core**, which is stronger than two
   pinned tasks: there is no between-task placement to control for.
4. **One binary** — every arm is a sub-benchmark of a single cross-compiled
   binary whose sha256 is recorded, so no two-build layout confound exists.
5. **`GOMAXPROCS`** — `keel-bench-gomaxprocs:` must read 1, and the row suffix
   `-1` must agree with it.

## Then, and only if P2 holds

A full `Sgemm/n=2048` arm, which is the measurement #136 actually asks for —
*"measured surviving packing and blocking, not just emitted leaner"* — since a
kernel-level win can be eaten by an `N%24` fringe path that `NR=16` does not pay.
That arm needs `3x24` dispatched, so it is a separate restricted build on the
same core, `972ee47`'s pattern, and it is multi-draw per arm (§5 rule 25): a
window median is a draw over which mode held the middle of the sort, and this
fleet has bimodality on the record.

**`3x24` does not ship on this file's prediction.** It ships, if it ships, on the
rows — and the losing arms are published either way.
