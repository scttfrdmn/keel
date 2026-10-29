# gate-p5 on amd64 in era famaxx86 (#199)

The founding and confirmation runs of amd64's own era, opened when the AVX-512 ceiling became the
family-max x44 loop and the roofline's pin moved to the anchor. The kernels are unchanged since ceil8.
Only the denominator moved. Predictions and results, in the order they were written, are in
`prereg.md`. The driver is `driver-frozen.sh`.

| run | rev | tally (P/F/U/B/R) | notes |
|---|---|---|---|
| exploration 1 | f2804ee | — | died at `-race` on a dev-host reboot; unmeasured. The teardown check failed open, and was fixed in d14cee0 |
| exploration 2 | d14cee0 | 55/4/1/3/3 | small instances; every prediction held |
| judged r1 | f18f110 | 56/3/1/14/4 | founding run; skx `Ceiling/compute` 1565.5, x44/shared 1.125 |
| judged r2 | c0c373d | 56/3/1/14/4 | confirmation; refuted on keel-zen5 Ssymm's 5.54% spread |

Instances: c5n.18xlarge (Intel Xeon Platinum 8124M, Skylake-SP), c7a.48xlarge (AMD EPYC 9R14, Zen 4),
c8a.48xlarge (AMD EPYC 9R45, Zen 5), us-east-1, on-demand, $24.09/hr together. Every rate's denominator
is the host's own measured ceiling, printed beside it in the README. No OpenBLAS reference was taken at
these thread counts. Landed: 27 of 29 baseline rows, plus the skx and zen4 witness rows (zen5's is held).
The README's amd64 rows are regenerated from r1 and r2.
