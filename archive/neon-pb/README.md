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
