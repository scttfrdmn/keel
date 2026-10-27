# NEON shim removal: exploration A/B (#191, 7538489)

`tools/shapegen -form raw` kernels against the shipped shim kernels, one test binary,
10 interleaved rounds, each pinned to core 2 (`taskset -c 2`). Hosts: c7g, c8g and c9g
4xlarge, on-demand us-east-1 (read-back: MIDR part 0xd40 Neoverse-V1, 0xd4f Neoverse-V2,
0xd84 Neoverse-V3). Exploration tier: these are correctness-class sizes, so nothing here is a
verdict. The run took 6m14s and the fleet was verified down. The denominators are each host's
own `BenchmarkPeak/neon` and `BenchmarkFMARoof64/neon` from the same rounds. There is no
OpenBLAS reference; this is a kernel-to-kernel A/B.

Median GFLOP/s at kc=512, as a % of that host's peak (`summary.txt` has all rows):

| kernel | insns/FMA | V1 | V2 | V3 |
|---|---|---|---|---|
| shim 3x24/2 (shipped) | 4.111 | 87.2% | 62.4% | 62.6% |
| raw 3x24/1 | 2.889 | 88.0% | 64.0% | 64.2% |
| raw 2x24/2 | 2.833 | 80.4% | 61.8% | 61.8% |
| shim D3x12/2 (shipped) | 4.111 | 87.2% | 63.3% | 63.5% |
| raw D3x12/1 | 2.889 | 88.1% | 64.9% | 64.6% |
| raw D2x12/2 | 2.833 | 81.3% | 62.6% | 62.7% |

**Prediction refuted.** Before the run I predicted that cutting insns/FMA from 4.111 to about
2.6 would move V2 from about 62% toward 85–95%. The instruction count fell by 30% (4.111 → 2.889)
and throughput rose by 1–2.6 points. On every host and at both elements, raw 3x24/1's worst
round beats shim 3x24/2's best, so the gain is real but small. It is not the lever.

**So instruction issue is not what holds V2/V3 at ~62%.** On V2, shim 3x24/2 works out to
0.624 × 2.9 FMA/cycle × 4.111 ≈ 7.4 insns/cycle, and raw 3x24/1 to 0.640 × 2.9 × 2.889 ≈
5.4. The "7–8 insns/cycle on every generation" in `../p2-arm64-c9g/README.md` is one
instruction mix's retirement rate, not a ceiling. What the three cores share is a rate per
FMA, not per instruction: on the shipped kernel, about 1.74 (V1, 0.872 × 2.0), 1.81 (V2, 0.624 × 2.9)
and 1.78 (V3, 0.626 × 2.85) FMA/cycle. That uses
the same peak-per-cycle figures as that README, so it is one derivation, not three
corroborations. Open hypothesis, untested: the broadcast (`VMOV` element insert, then `VDUP`)
or the load path, not the front end.

## Limiter probe (9cf8848) and denominator check (89ccacd)

Predictions are in `probe-prereg.md`, committed before each run. Both ran on the same three
4xlarge hosts as above, pinned to core 2, with 10 interleaved rounds.

**The limiter is the broadcast: predicted, and it held.** Pre-broadcast A (PB) beats raw
3x24/1 by 62% on V1, 59% on V2 and 43% on V3. No B loads (NB) moves nothing. Each
`BroadcastFloat32x4` lowers to 3 vector ops (zero copy, insert, `VDUP`). This is
golang/go#81352, open and labelled FixPending, with fix CL 827904 (cmd/compile: fold arm64
broadcasts of inserted scalars) at Code-Review+2 on PS2 and not submitted as of 2026-10-26.

**The denominator conflict is open.** The FMA-only probe (FO) reads 78.2 / 87.75 / 103.2
GFLOP/s, which is 1.89× / 1.35× / 1.28× `BenchmarkPeak/neon`, in the same binary on the same
core. My explanation, that the shim's 16 anchor NOPs bound the peak kernel, is **REFUTED**
(`peakcheck-89ccacd-summary.txt`). Its raw-archsimd twin, with 0 NOPs, reads 41.42 against
41.42 on V1, +8% on V2 and +5% on V3. 24 chains read no higher than 16. So `neonPeak`
under-reads FO for a reason not yet identified. Two differences stand out: all 16 of its FMAs
read the same two multiplicand registers, where FO's 18 spread over 9; and its operands are
exactly 1.0. **VOID by name: commit `89ccacd`'s subject clause "the NEON peak under-reads"
stands as an observation, but its body's "the denominator ... is NOP-bound" is refuted.**
`VOID-peakcheck-89ccacd-quoting.log` is the first launch of this check. It was killed by its
own trap before measuring anything, because of a quoting bug in the driver.

Why it matters: every arm64 P2 percentage divides by `neonPeak`/`neonPeakD`. If FO's rate is
the true ceiling, the shipped NEON kernels are at about 46–49% of it, below the 55% floor.
Nothing in the gate has changed. This is the open question, not a verdict.
