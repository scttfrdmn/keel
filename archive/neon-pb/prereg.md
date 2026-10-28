# Pre-broadcast-A shape selection: rule and predictions (written before the run, #198)

`tools/shapegen -form pb` (A packed with each element replicated across a register's lanes)
has 160 emittable zero-spill shapes at each element, and a frontier of 1.556 insns/FMA
(4x20/8 at float32, 4x10/8 at float64), against the broadcast form's 4.111. Nine
candidates per element, all spill-free, sit in `internal/vec/pb_arm64_test.go`. Each is
held to a definition-order reference (positive-controlled: a planted operand fails at kc=8,
in the ×8 loop). They are timed by `BenchmarkPBTile` beside `Peak/neon` / `FMARoof64/neon`
(the family-max denominator) on c7g, c8g and c9g 4xlarge (Neoverse V1/V2/V3), pinned, in
10 interleaved rounds.

Candidates: float32 4x20/8, 5x16/8, 3x28/8, 7x12/8, 4x16/8, 3x24/8, 6x12/8, 3x24/4, 4x20/4;
float64 at half the columns, the same nine.

**Selection rule.** Per element, each candidate's score is the geometric mean over the three
hosts of its median GFLOP/s at kc=128 (gate-p2's kc) as a fraction of that host's peak
median. The highest score ships as the preferred shape. If the top two scores are within
1% of each other, the one with the smaller MR·NR ships, because it has the smaller fringe
and A panel. The next two distinct MRxNR shapes by score ship as alternates (gate-p2's
classifier mixes, and Preferred's ranking). The 1% tie band and the order are fixed here,
before any number exists.

**Predictions.** The preferred shape reaches at least 65% of peak on V2 and V3 and at least
70% on V1, at both elements. The basis: raw 3x24/1 with pre-broadcast A (PB) reached 69–75%
of the FMA-only loop at kc=512 (`../neon-raw/`), and the U8 shapes have fewer insns/FMA.
**REFUTED if** the preferred shape is under 55% anywhere. That would mean the shaping step
does not clear the floor even before packing costs are counted.

Kernel-level only: whether pre-broadcast packing pays for itself in a full Sgemm/Dgemm,
where the A panel is 4× (float32) or 2× (float64) larger, is a separate measurement after
the packing path lands.
