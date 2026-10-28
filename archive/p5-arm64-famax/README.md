# gate-p5 on arm64 in era famax: the Neoverse README rows re-derived (#200)

The predictions for every run are in `prereg.md`, committed before each launch, and the
driver is `driver-frozen.sh`. Hosts: c7g.16xlarge (Neoverse-V1) and c8g.48xlarge
(Neoverse-V2), on-demand us-east-1, judged tier. The exploration pass was on 4xlarge sizes.

| run | rev | role | tally | duration |
|---|---|---|---|---|
| exploration | e3ddc1b | path witness | 39 PASS / 4 FAIL / 2 UNMEASURED / 2 BASELINE | 1h40m, ~$2 |
| r1 | 085f23e | famax founding run, no fraction in force | 51 PASS / 2 FAIL / 0 / 2 BASELINE | 1h17m, ~$13.3 |
| r2 | 37d7c13 | first judged run, `CEIL_FRACTION` 47.0 | 51 PASS / 2 FAIL / 0 / 2 BASELINE | ~1h20m, ~$13.8 |

The exploration pass found five defects, fixed in `30af10d`; `prereg.md` lists them, one of
them self-inflicted. In r1 and r2 the only FAILs are criterion 9 on the stale ceil8 README
rows, which is what these runs exist to replace.

**r2 against its predictions: every one held.** CEIL_FRACTION 47.0 was in force with
famax's derivation lines. All eight share rows passed, the tightest being keel-gvt4 Strsm at
**49.8%** (predicted 49.6 ± 1.5, bar 47.0). Sgemm 1T was **59.10 (V1) and 65.38 (V2)**,
inside the bands 58.0–59.2 and 64.7–66.1. gate-p4 GREEN, `-race` clean, and fo the
family max at 8 threads.

**Landed with this archive (N=2, r1 + r2):** 48 L1 `rate/*` rows, all inside the 5%
same-claim band (widest cross-draw spread 3.52%, none held); `peak/5x16-pb/neon/kc=128` at
86.25 (V1) and 81.3 (V2); and both hosts' witness rows, from r1, the founding run, as in
ceil8. All 50 registry rows resolve `registered` under `baseline_state` with the gates'
production arguments. An unlanded famax criterion on a witnessed host resolves `owing`, and
a ceil8 row still resolves its own era. Era `famax` is closed, with its transition archive
filled in `measurement-eras.tsv`.

**The README block** is regenerated from the two amd64 ceil8 logs and r1 + r2. The 27 amd64
rows are byte-identical, and the 18 Neoverse rows are pooled over N=2 on the new kernel and
denominator. Sgemm 1T is 58.78 (V1) and 65.39 (V2) GFLOP/s, at 75.2% and 77.5% of 1-thread
peaks of 78.16 and 84.42. The ceil8 rows read 34.39 and 37.33, at 83.0% and 60.3% of 41.42
and 61.87.
