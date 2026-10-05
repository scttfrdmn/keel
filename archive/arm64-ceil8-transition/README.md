# Era `ceil8`'s founding reading — us-east-1, 2026-10-05, rev `119b673` (#177)

`42 PASS / 2 FAIL / 1 UNMEASURED / 10 BASELINE / 0 REPORTED` → **RED**, with `gate-p4`
GREEN beneath it (`64 PASS / 0 FAIL`). **The red is the two stale-README rows and nothing
else**, and the share criterion judged nothing by design: `CEIL_FRACTION` is suspended to
empty at this era's boundary, so this run REPORTS and a reviewed commit types the bar.

Fleet: `keel-gvt3` `c7g.16xlarge` (Neoverse-V1) and `keel-gvt4` `c8g.48xlarge`
(Neoverse-V2), both on-demand in us-east-1, TTL 8h, `$9.98/hr`, 1h16m wall. Driver
`ceil8-transition.sh`, archived here as evidence. Co-tenant census at launch and after,
in the driver log: one foreign instance throughout (`loop-l12-esc`, a `c8g.8xlarge`
belonging to another project) and both measured hosts are family-max whole hosts, so it
cannot share silicon with either.

## The pre-registration held at row granularity

The driver carries it, computed by driving `baseline_state` against the shipped artifacts
*before* launch rather than predicted from prose. Rendered against predicted:

| predicted | rendered |
|---|---|
| 8 `BASELINE` share lines (4 rows × 2 hosts) | **8** ✓ |
| 8 candidate baseline rows, keyed `share/3x24/neon/<row>` at era `ceil8` | **8**, every key shaped ✓ |
| 2 witness rows, one per host | **2** ✓ |
| no `PASS` and no `FAIL` on the share criterion | none ✓ |
| criterion 9 (README) still RED on both hosts | **2 FAIL** ✓ |
| `gate-p3` `peak/*` unaffected → `BASELINE` | **BASELINE** both hosts ✓ |

**One thing the pre-registration was silent on, stated rather than claimed as a hit:** the
headline aggregate rendered `UNMEASURED` — *"all 2 configured host(s) rendered BASELINE, so
the headline criterion has no host it may judge"*. That is the correct behaviour and it is
a change from the earlier draws, which rendered a `FAIL` aggregate (`0 of 2 governed gate
hosts cleared their class's bar`). The prediction named the per-row states and the bucket;
it did not name the aggregate's word, so this is an unpredicted detail that came out right,
not a verified prediction.

`gate-p3`'s peak readings moved with the kernel, as `#136` implies: `3x24/neon/kc=128`
reaches **62.6%** (gvt4) and **85.8%** (gvt3) of measured NEON peak, against the registered
`4x16` rows' 55.3 / 78.8 — +7.3 and +7.0 points, recorded as candidates and judged by
nothing.

## `Strsm` has a share for the first time, and it is the argmin

It was on a T8/T1 ratio until this amendment, so its share had never been computed. It is
**the lowest row on the fleet**, which makes it the bar's argmin — and a bar typed off a
quantity measured once would be a draw standing in for an estimator (§5 rule 16). So it was
**recomputed from the two earlier `3x24` draws rather than re-measured**: the 8-thread rate
and the ceiling were both taken in those runs, and only the division is new. The instrument
is the gate's own (`bench_csv` + `bench_ratio_lo` from `scripts/bench.sh`), and reading
*this* run's archive back through the same path reproduces the gate's printed 45.4 / 61.2
exactly, which is the positive control on the recomputation.

Share of each host's own measured 8-thread ceiling, net of CI, at `3x24/neon`:

| host | row | draw1 | draw2 | `ceil8` | median | lowest | widest interval |
|---|---|---|---|---|---|---|---|
| keel-gvt3 | `Sgemm` | 73.5 | 73.5 | 73.5 | 73.5 | 73.5 | 0.10 |
| keel-gvt3 | `Ssyrk` | 79.3 | 79.3 | 79.3 | 79.3 | 79.3 | 0.10 |
| keel-gvt3 | `Ssymm` | 73.2 | 73.1 | 73.2 | 73.2 | 73.1 | 0.10 |
| keel-gvt3 | `Strsm` | 61.3 | 61.7 | 61.2 | 61.3 | 61.2 | 1.20 |
| keel-gvt4 | `Sgemm` | 53.4 | 53.5 | 53.3 | 53.4 | 53.3 | 0.30 |
| keel-gvt4 | `Ssyrk` | 57.5 | 57.6 | 57.4 | 57.5 | 57.4 | 0.30 |
| keel-gvt4 | `Ssymm` | 53.2 | 53.2 | 53.0 | 53.2 | 53.0 | 0.40 |
| keel-gvt4 | **`Strsm`** | 45.8 | 45.0 | **45.4** | **45.4** | **45.0** | 1.40 |

All eight are admissible under §5 rule 19: widest interval **1.40 points** against the
**2.6**-point cap. The three `GEMM`-shaped rows reproduce across three draws to **≤0.2
points**, so the instrument is steady and the `Strsm` row's 0.8-point spread is its own
property, not the fleet's weather.

**The argmin is `keel-gvt4 Strsm` under both estimators and at the same value** — 45.4 as
this run's row and 45.4 as the median of three. That coincidence is worth stating, because
it means the bar does not depend on which estimator types it.

## What is NOT done here, and why

`CEIL_FRACTION` is **not typed in this commit**. §5 rule 17(d)'s three steps put the typing
in a *reviewed* commit between this run and the one that judges, and the construction has a
live question that is not the instrument's to settle — see the issue. The two candidates
differ by 0.4 points:

- **42.8** = 45.4 − 2.6, the **precedent form**: 44.2 was derived as "the lowest of six
  admissible judged rows in the founding campaign's take four, less 2.6 points", i.e. the
  lowest row of the typing run.
- **42.4** = 45.0 − 2.6, using the **lowest observation across this era's three draws**, so
  the bar sits a full margin below every reading that exists rather than below a median.

The registry rows are likewise **not** hand-landed. Under §5 rule 17(b) the gate proposes
and a reviewed commit lands; and whether these hosts need rows at all depends on the
`CEIL_DERIVED_FROM` question, since a host inside the derivation set is fleet-governed and
registers nothing.

## Files

- `gate-p5-arm64-ceil8-transition.log` — the gate, ANSI-stripped out of the driver log
- `gate-p4-under-p5-*.log`, `gate-p3-under-p4-*.log` — the carried chain
- `bench-gate-p5-*.txt` — the raw samples every figure above is recomputable from
- `baseline-candidates-*.tsv` (56 rows: 8 share + 48 L1), `witness-candidates-*.tsv` (2)
- `ceil8-transition.sh` — the driver, including its pre-registration block
- `driver-ceil8-transition-119b673.log` — launch, provisioning, co-tenant censuses, teardown

Verify with `shasum -c archive/arm64-ceil8-transition/DIGESTS.sha256` from the repo root.
