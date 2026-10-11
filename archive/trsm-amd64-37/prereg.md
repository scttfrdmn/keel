# Pre-registration: does the diagonal-solve fix pay more on AVX-512? (#37, 2026-10-11)

Written before any arm runs. Dev tier first, on purpose: this is a MECHANISM
question, and the lab answers it for free. A judged AWS run is for the citable
number afterwards, not for deciding whether the mechanism is real.

## The prediction being tested
Strsm/Sgemm single-thread efficiency is 0.804 and 0.813 on keel's two NEON hosts
against 0.336, 0.356 and 0.358 on its three AVX-512 hosts — three agreeing at
0.35, two at 0.81, split exactly on vector width. Inverting Amdahl against the
work split every judged gate log prints (rank_update=0.98413,
diag_solve=0.01587) puts **~65% of Strsm's time on AVX-512 inside 1.6% of its
work**, against 13-22% measured on NEON. If that is right, the two substitutions
already landed (`090eb32` axpy, `06f35a4` divide) are worth substantially more
here than the +11.1% they bought on NEON.

## Instrument
`scripts/ab-bench.sh trsm`, base `79d629a` (the commit before the axpy), two
cross-compiled static linux/amd64 binaries of `./bench`, filter `Scale`, n=4096,
threads 1 and 8, `-count=10 -benchtime=1s`. Submitted to janus's `measured` pueue
group, launched detached.

## Rows and thresholds, in the instrument's own output space
Eight rows per host: `Scale/{Sgemm,Ssyrk,Ssymm,Strsm}/n=4096/threads={1,8}`.

- **CONTROLS — `Sgemm`, `Ssyrk`, `Ssymm`, both thread counts.** No diagonal
  solve; their code is byte-identical in both arms. Each must land inside
  janus's between-binary layout floor, **0.99%** (`build/layout-ensemble-e829a61.log`,
  the Skylake-X figure the `edge` preset already uses). **A control above its
  floor VOIDS the run** — #22's ruling, and not a formality here: the arm64 run
  of this same change moved untouched code 8.4% when `block.gemm.func1` shifted
  16->32 mod 64 (#61).
- **TREATMENT — `Strsm/n=4096/threads=1`**, the row the README publishes and
  criterion 9 judges, and the side `solveLeft` serves (`keel.Left`).
  - **CONFIRMED** if sec/op improves by **more than 15%**. The model's own range
    is -30% to -50% (65% of time in the solve, solve 2-4x faster), so 15% is a
    deliberately slack floor that still clears every control floor by 15x.
  - **REFUTED** if the improvement is **under 5%**. That would mean the ~65%
    share is wrong and the NEON-vs-AVX512 efficiency split has another cause.
  - **INCONCLUSIVE** between 5% and 15%: real but far below the model, which
    would itself need explaining rather than claiming.
- **`Strsm/threads=8`**: predicted to improve but NOT used for the verdict. The
  diagonal solve is the serial dependency, so the 8-thread figure mixes the fix
  with Amdahl and cannot be read as a clean measure of either.

## What this cannot settle
janus is bare metal and virtualized AWS is the judged tier, so nothing here is
publishable: the README's numbers come from AWS and a citable improvement needs
an AWS run. janus also carries its own filed pathologies — within-run bimodality
in 18 of 138 cells (#147), a 13-16% between-arm level term no treatment explains
(#150) — so a median here may not be a level. The predicted effect is 2-4x that
level term, which is why janus can still answer a yes/no mechanism question it
could not answer for a 3% one. The controls are what say whether the host
behaved.
