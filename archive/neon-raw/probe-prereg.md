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
