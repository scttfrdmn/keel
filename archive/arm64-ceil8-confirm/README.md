# Era `ceil8`'s confirmation run — us-east-1, 2026-10-07, rev `5e769f7` (#177, step 3)

**`51 PASS / 2 FAIL / 0 UNMEASURED / 2 BASELINE / 0 REPORTED` → RED, which is exactly what was
pre-registered**, down to every count. `gate-p4` GREEN beneath it (`64 PASS / 0 FAIL`). The two
reds are criterion 9's stale README rows and nothing else; the driver said in advance that this
run *cannot* be green and that a green would mean something unexpected had happened.

**Step 3 is done: the share criterion judged for the first time this era, and all eight rows
clear `CEIL_FRACTION = 42.8`.** The bar was derived from the 2026-10-05 transition run and
enforced here, which is the only arrangement under which it can fail.

| host | row | point % | net of CI % | bar | headroom |
|---|---|---|---|---|---|
| keel-gvt3 | `Sgemm` | 73.60 | 73.5 | 42.8 | +30.7 |
| keel-gvt3 | `Ssyrk` | 79.40 | 79.3 | 42.8 | +36.5 |
| keel-gvt3 | `Ssymm` | 73.30 | 73.2 | 42.8 | +30.4 |
| keel-gvt3 | `Strsm` | 62.50 | 61.4 | 42.8 | +18.6 |
| keel-gvt4 | `Sgemm` | 53.80 | 53.5 | 42.8 | +10.7 |
| keel-gvt4 | `Ssyrk` | 58.00 | 57.7 | 42.8 | +14.9 |
| keel-gvt4 | `Ssymm` | 53.50 | 53.3 | 42.8 | +10.5 |
| keel-gvt4 | **`Strsm`** | 46.50 | **44.5** | 42.8 | **+1.7** |

Fleet: `keel-gvt3` `c7g.16xlarge` (Neoverse-V1), `keel-gvt4` `c8g.48xlarge` (Neoverse-V2), both
on-demand us-east-1, TTL 8h, `$9.98/hr`, 1h16m. Driver `ceil8-confirm.sh`, archived here.

## The pre-registered band was violated, and the violation is instructive

The driver predicted `keel-gvt4` `Strsm` at **~45.4 ±0.5**, from three archived draws at
45.8 / 45.0 / 45.4. It came in at **44.5** — below all three and outside the band.

**Decomposed with the gate's own instrument, the level did not move at all:**

| run | T8 rate | ceiling | point % | net of CI % | **interval width** |
|---|---|---|---|---|---|
| draw1 | 230.00 | 493.90 | 46.60 | 45.8 | 0.80 |
| draw2 | 228.95 | 493.85 | 46.40 | 45.0 | 1.40 |
| transition | 231.10 | 493.95 | 46.80 | 45.4 | 1.40 |
| **confirm** | **229.20** | **492.90** | **46.50** | **44.5** | **2.00** |

The point share, 46.50, is the most central of the four draws (range 46.40–46.80), and the
8-thread rate sits inside the prior range. **The entire 0.9-point fall in the judged share is
the confidence interval widening**, 1.40 → 2.00 points. Nothing got slower.

**So the pre-registration was stated on the wrong quantity, and that is the methodological
finding of this run.** A band on a *CI-deducted* share constrains `point − width`, which is two
terms; it cannot distinguish a routine slowing down from an instrument getting noisier, and
those call for opposite responses. The band should have been declared on the **point** and the
**width** separately. Stated here rather than quietly re-derived, because the band did its job —
it fired — and what it could not do is say *which* term moved.

## The pre-registered image confound is LIVE and NOT separated

The driver recorded, before any reading existed, that `spawn` had resolved a different Ubuntu
AMI (`ami-0e1ab5c876cc030e8` against the transition run's `ami-0bec8cef5313300ad`) and that an
image refresh can move the guest kernel, which can move the bandwidth ceiling that denominates
every share here. Read back: **`Linux 7.0.0-1014-aws`, against `7.0.0-1013-aws`** on 2026-10-05.
So the confound is real and it is the leading candidate for the widening:

- `keel-gvt4`'s `Strsm` width **doubled** (1.40 → 2.00) and its ceiling fell 1.05 GFLOP/s (0.21%).
- `keel-gvt3`, the **control**, saw the same kernel change and did **not** widen (1.20 → 1.10),
  and its rows reproduce to ≤0.1 points.

A 192-core two-socket box gaining scheduling jitter where a 64-core single-socket one does not
is a plausible kernel effect — and **one draw on the new image cannot distinguish it from one
noisy draw**. It is a candidate cause, unseparated, and nothing here is attributed to it. What
would separate it: a second run on this same AMI. The confound is named because it was named
*before* the number, which is the only thing that makes it a control rather than a story.

## What this says about 42.8's fragility, measured on all eight rows

The obvious worry is that `keel-gvt4` `Strsm` sits 1.7 points over the bar with a 2.00-point
interval, so noise reddens the gate. **It cannot.** Rule 19's width-admissibility check sits
*ahead* of bar selection, so a row whose interval exceeds `BASELINE_MARGIN`'s 2.6 points renders
`REPORTED` and is never compared. The width that would carry each row *below* 42.8 is the row's
own `point − 42.8`:

| row | width now | width needed to FAIL | reachable? |
|---|---|---|---|
| keel-gvt4 `Strsm` | 2.00 | **3.70** | no — rule 19 refuses at 2.6 first |
| keel-gvt4 `Ssymm` | 0.20 | 10.70 | no |
| keel-gvt4 `Sgemm` | 0.30 | 11.00 | no |
| keel-gvt4 `Ssyrk` | 0.30 | 15.20 | no |
| keel-gvt3 `Strsm` | 1.10 | 19.70 | no |
| keel-gvt3 `Ssymm` | 0.10 | 30.50 | no |
| keel-gvt3 `Sgemm` | 0.10 | 30.80 | no |
| keel-gvt3 `Ssyrk` | 0.10 | 36.60 | no |

On every row the width that would breach the bar exceeds the admissibility cap. **A widening
interval can cost coverage and can never produce a false red; only the point estimate falling
can fail this criterion.** The live risk is therefore `keel-gvt4` `Strsm` going *unjudged* — it
is at 2.00 of 2.6 — not failing. That is a property of the criterion's ordering, and it is
measured here rather than assumed.

## A gap this run exposed, and the 2026-10-06 typing commit created it

`peak/*` and the 48 L1 `rate/*` rows rendered `BASELINE` **again**, as pre-registered. But the
reason they will keep doing so is new: **the witness row is proposed at exactly one site**
(`scripts/gate-p5.sh:1456`, inside the share criterion's `new` arm), and both Graviton hosts now
resolve `fleet` there, because the typing commit moved them *into* `CEIL_DERIVED_FROM`. So no
witness row for `(Neoverse-*, ceil8)` is ever proposed, `baseline_spent` keeps missing, and two
other criteria render `BASELINE` on every run forever — green-compatible and silent, which is
the permanent exemption the class exists to kill.

Nothing is falsely passing today: both criteria are reported-not-judged anyway, with no bar
typed. What is broken is the path by which they would *become* judged. Filed rather than fixed:
the repair needs a decision about estimators, and the era now has **two** archives, so a
median-of-2 is available for all 50 rows (`peak/*` reads 62.6 / 85.8 then 62.8 / 85.8 — ≤0.2
points apart).

## Files

- `gate-p5-arm64-ceil8-confirm.log` — the gate, ANSI-stripped from the driver log
- `gate-p4-under-p5-*.log`, `gate-p3-under-p4-*.log` — the carried chain
- `bench-gate-p5-*.txt` — the raw samples every figure above is recomputable from
- `baseline-candidates-*.tsv` — 48 L1 rows; **no witness candidates**, which is the gap above
- `ceil8-confirm.sh` — the driver, with its pre-registration and the AMI confound named in advance
- `driver-ceil8-confirm-5e769f7.log` — launch, provisioning, co-tenant censuses, teardown

Verify with `shasum -c archive/arm64-ceil8-confirm/DIGESTS.sha256` from the repo root.
