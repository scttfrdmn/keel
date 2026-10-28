# Pre-broadcast-A shape selection (#198, 825d997)

Rule and predictions are in `prereg.md`, committed before the run. The run was on c7g, c8g
and c9g 4xlarge (Neoverse V1/V2/V3), pinned to core 2, 10 interleaved rounds, and took 14m06s;
the fleet was verified down. All numbers are in `summary.txt`. This is the exploration tier,
so it is a selection, not a verdict.

**The rule applied.** Score = geometric mean over the three hosts of median % of the
family-max peak at kc=128.

| element | 1st | 2nd | tie band (1%)? | tie-break MR·NR | ships | alternates |
|---|---|---|---|---|---|---|
| float32 | 5x16/8, 83.0% | 4x20/8, 82.5% | yes | 80 = 80, inert | **5x16/8** (highest score) | 4x20/8, 7x12/8 |
| float64 | 5x8/8, 83.1% | 4x10/8, 82.5% | yes | 40 = 40, inert | **5x8/8** (highest score) | 4x10/8, 7x6/8 |

The tie-break did not separate the top two at either element: both pairs have equal MR·NR.
So the rule's base clause, "the highest score ships", decided. That gap in the rule is
recorded here, not patched after the fact.

**Predictions held.** The preferred shape reached 86.2% (V1), 81.5% (V2) and 81.4% (V3) at
float32, and 86.2 / 81.6 / 81.4% at float64. The predicted minimums were 70% on V1 and 65%
on V2/V3, and the floor is 55%. In the same runs, the shipped broadcast-form kernels read
45.5–46.4%.

Kernel-level only. Whether a 4× (float32) or 2× (float64) larger packed A panel pays for
itself in a full Sgemm/Dgemm is measured after the packing path lands.

## The nest witness (def1b79): promoted at both elements

Rule and predictions are in `nest-prereg.md`. The run is one 2048³ Gemm through
`internal/block`'s nest, with `GOMAXPROCS=1`, pinned, in 5 interleaved rounds on the same
three hosts. It took 10m46s, and the fleet was verified down. All points are in
`nest-summary.txt`.

| element | shipped baseline, GFLOP/s (V1 / V2 / V3) | promoted point | GFLOP/s (V1 / V2 / V3) | gain |
|---|---|---|---|---|
| float32 | 3x24, KC=384, MC=144: 33.51 / 37.68 / 46.08 | **5x16/neon-pb, KC=384, MC=72** | 56.24 / 64.05 / 78.58 | +67.8 / +70.0 / +70.5% |
| float64 | 3x12, KC=512, MC=144: 16.58 / 19.05 / 23.67 | **5x8/neon-pb, KC=256, MC=72** | 29.11 / 30.79 / 40.20 | +75.6 / +61.6 / +69.8% |

On every host, the promoted point's worst round beats the baseline's best. **The prediction
of at least 25% held**: every pre-broadcast point listed beats the baseline, by 51–76%.

**How the tie rule resolved.** At float32 the top geomean (5x16, KC=384, MC=72, 65.66) had
three neighbours within 1%: (256, 36) 65.50, (384, 36) 65.37 and (256, 72) 65.35. The
first preference, the shipped (KC, MC) = (384, 144), was not among them (62.92). The
second, larger MC, left MC=72 at two KCs, and the base clause, highest geomean, decided
(384, 72). At float64 nothing was within 1% of the top.

The pre-broadcast kernels want a smaller MC than the shipped 144, at both elements, and a
shallower KC at float64. MC = 36, the point that holds the A block at the shipped bytes,
was competitive but not best. Per the rule, these are blocking values for pre-broadcast
kernels only. amd64 and the conventional NEON kernels keep theirs.
