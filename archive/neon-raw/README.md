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
