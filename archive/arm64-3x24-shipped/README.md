# 3x24 shipped: judged arm64 campaign, us-east-1, rev 4374c4f, 2026-10-05

`gate-p5 tally: 48 PASS / 5 FAIL / 0 UNMEASURED / 2 BASELINE` -> RED, gate-p4 GREEN (64/0/0).
Dispatch verified `3x24/neon` on both judged hosts.

## The promotion is good on every absolute measure

Shares of each host's own measured 8-thread ceiling, all PASS with wide margins:
Sgemm 73.5% / 53.4% (bars 49.6 / 35.0), Ssyrk 79.3% / 57.5% (53.7 / 38.0),
Ssymm 73.2% / 53.2% (49.6 / 34.9). Single-thread Sgemm rose 32.53 -> 34.44 and
33.81 -> 37.36, which is why criterion 9's README rows disagree: the published
rows were measured at 4x16 and the code is now faster than them.

## The 5 FAILs are 2 causes

**Pre-stated (2):** criterion 9's README rows on both hosts, stale because the
library got faster. Resolvable by regeneration.

**Not pre-stated (3), one cause:** the two per-host Strsm rows and the aggregate
they feed. Strsm's absolute rates improved on BOTH arms and BOTH hosts --
1T +6.3%/+6.2%, 8T +2.8%/+3.9% -- and the T8/T1 ratio fell because 1T improved
more. That is the rank inversion §4/P5 retired the >=6x floor for, still live on
STRSM_FLOOR. Filed as #177. No bar was touched and the flip was not reverted.
