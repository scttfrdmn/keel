# Limiter probe: predictions, written before the run (#191)

`internal/vec/probe_arm64_test.go` holds raw 3x24/1 (`ctl`) and four variants, each with
one thing removed from the K-loop. `probe-gen.py` derived them mechanically from shapegen's
emission, and `probe-audit.txt` has their spill-audit lines (all zero-spill). In the
control, each broadcast lowers to 3 vector ops (copy the zero register, insert the element,
`VDUP`), so it issues 27 vector-pipe ops for 18 FMAs.

| probe | removed | insns/FMA |
|---|---|---|
| ctl | nothing | 2.89 |
| PB | the broadcast (A pre-broadcast, one q-load per row) | 2.39 |
| NB | B loads | 2.28 |
| PBNB | both | 1.78 |
| FO | everything but the FMAs and loop bookkeeping | 1.33 |

Predictions, on V2 and V3, as % of the host's measured NEON peak:
- If the broadcast is the limiter, PB is clearly above ctl (at least 8 points) and NB is not.
- If the B-load path is the limiter, NB is clearly above ctl and PB is not.
- FO reaches at least 90% of peak. **If it does not**, the limiter is in the loop's own
  structure (18 accumulators, one FMA each per iteration, or the bookkeeping), and neither
  the broadcast nor the loads explain the gap.

## Result of the limiter probe (9cf8848), and the denominator check it forced

The prediction "the broadcast is the limiter" **held**. PB beat ctl by 43–62% on every host,
and NB stayed within 0.4 GFLOP/s of ctl (`probe-9cf8848.log`). FO, the loop with nothing but
FMAs, read **131–189% of `BenchmarkPeak/neon`**. A loop cannot beat the true peak, so the
shipped NEON ceiling under-reads. spill-audit puts `neonPeak` and `neonPeakD` at 35 insns for
16 FMAs, 16 of them anchor NOPs (golang/go#80830), against 19 and 0 for their raw-archsimd
twins.

Predictions for the denominator check (`BenchmarkPeakProbe`, written before it runs):
- raw16 reads at least 1.25× shim16 on V2 and V3, and at least 1.7× on V1, at both elements.
  This is the FO ratio. **REFUTED if** raw16 is within 10% of shim16 anywhere.
- raw24 is within 5% of raw16: 16 chains already saturate.
- raw16 is within 10% of FO's FMA rate on each host.
