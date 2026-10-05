# Draw 2 at the shipped shape — us-east-1, 2026-10-05

`48 PASS / 5 FAIL / 0 UNMEASURED / 2 BASELINE` → RED, the **same five criteria as draw 1**,
which is the point: the reading reproduces, so the red is the bar and not noise.

## The two draws at `3x24/neon`, and they are tight

| row | keel-gvt4 (V2) draw1 / draw2 | keel-gvt3 (V1) draw1 / draw2 |
|---|---|---|
| share/Sgemm | 53.4 / 53.5 | 73.5 / 73.5 |
| share/Ssyrk | 57.5 / 57.6 | 79.3 / 79.3 |
| share/Ssymm | 53.2 / 53.2 | 73.2 / 73.1 |
| scale/Strsm | 7.595× / 7.577× | 7.438× / 7.450× |

Shares agree to ≤0.1 points, ratios to ≤0.018×.

## What draw 2 changed about the diagnosis

Not the bar's form. The registered `scale/Strsm` baselines (7.921× / 7.902×) were measured
at **`8x8/neon`** — verified from `archive/pinned8/bench-gate-p5-029e24f-*.txt`'s own
`keel-bench-kern` marker — and were being applied to `3x24/neon`, **two dispatch changes
later**. `peak/*` is keyed on the dispatched shape for exactly this reason (#167); its
`scale/*` and `share/*` siblings were left bare. That is the immediate cause of the red, and
it is a different defect from #177's wrong-quantity finding.

## What was NOT done here, and why

The registry rows are **not** hand-written from these two draws, even though the values are a
proper median-of-2 estimator. §5 rule 17(b): *"An instrument that mints the reference it will
later judge against has certified itself, so the gate emits a fully formed candidate row to a
scratch file and landing it is a reviewed commit act."* Hand-computing rows the gate never
emitted inverts that. With the keying landed, the next run has the gate propose correctly-keyed
candidates, and the figures above are the prior to check its proposal against.
