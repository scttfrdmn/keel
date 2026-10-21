# The first green L1 result, arm64 (#179/#180 confirmation)

Rev `dc519af`, era `ceil8`, us-east-1 on-demand, 2026-10-21 14:28Z–15:46Z (1h18m, ~$13).
`keel-gvt3` c7g.16xlarge (Neoverse-V1), `keel-gvt4` c8g.48xlarge (Neoverse-V2).

**55 PASS / 0 FAIL / 0 UNMEASURED / 0 BASELINE / 0 REPORTED → GREEN.** `prereg.md`, written before
either run, predicted ~55 PASS / 0 FAIL, which is exact.

- **L1 holds on both hosts, 24 of 24 rows each.** 12 rows render as `drift`, under their baseline
  but within the 5% band and named in the PASS line: 5 on gvt4, 7 on gvt3, the deepest 2.9% below.
  These include the 9 #180 rows, which keep their pre-change baselines, so their placement shift
  stays visible every run.
- **The named arm64 risk did not fire.** `Neoverse-V1 rate/Sdot/n=65536`, landed 2026-10-21 with
  one historical draw 5.9% below it, read 1.1% below: drift, inside the band.
- **Criterion 9 passes on both hosts**, so the regenerated README agrees with a fresh judged run.
- No candidate rows were proposed, which is correct: every judged row was already registered.
