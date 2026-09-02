<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#148`'s core discriminator, `97a21f4` on `janus.local`, 2026-09-02

The analysis is `docs/issue148-core-97a21f4.md`. This directory is its evidence, tracked for the same
reason `archive/width148/` is: an arm cited by a path under `build/` is cited on one operator's laptop
and nowhere else (`.gitignore` ignores `/build/`).

Test 1 (`archive/width148/`) could not ask which *core*, because `keel_pin_mask` derives its mask from
a width and its only width-1 answer is cpu0 — so "one core is not enough" and "cpu0 specifically" were
the same arm. `KEEL_PIN_CPUS` (`4dd6334`) names the cpus instead, and these four arms are what that
buys: `ref=0,1` (two physical cores), `c0=0`, `c5=5` (one core, not cpu0), `smt=0,16` (two logical
cpus, **one** physical core).

Eight arms of **one** binary, `sha256=d0d46d26c15cc8b2`, 5066553 bytes, `go1.27.0-X:simd`, `-trimpath`
— byte-identical to `archive/width148/`'s and `archive/addr147/`'s, so `ref` here is a further draw of
that same artifact rather than a comparable build.
`bench-core148-97a21f4-{a,b}{ref,c0,c5,smt}.txt` are the arms, pass `a` in order `ref c0 c5 smt` and
pass `b` reversed; `core148-97a21f4.log` is the driver transcript, including the sixteen host samples
and the topology preflight.

## What this directory does that `width148` did not

**The thresholds are a committed module, and the analyzer imports them.** `predictions-core148.py`
holds the bands, the truth table, the estimator choice and the admissible reference levels;
`analyze-core148.py` loads it with `importlib` and prints its git hash in its own section 0. So the
numbers applied are checkably the numbers committed before the data existed, rather than numbers the
analyzer happens to agree with. `4dd6334` predates every reading here.

**The verdict paths were driven red before there was data to drive them green.**
`test-analyze-core148.py` is 12 fixtures: the three branches each named uniquely, the out-of-domain
cell proved to be *doing* something rather than quietly ignored, six refusals (positive control not
collapsing, `smt` reporting two cores, a missing `keel-pin` line, a control row moved, an absent arm,
`ref` off the admissible levels), plus "no branch fits" and "the estimators disagree". A checker whose
fail paths never ran is an unread witness.

**Every cpu id came from the host.** `siblings0=0,16` was read from `thread_siblings_list` over ssh.
The preflight refuses rather than substituting — no sibling for cpu0 (SMT off), cpu0 and cpu1 siblings
of one core (`ref` would not be two cores and every ratio would carry the wrong denominator), cpu5
sharing cpu0's core (`c5` could not separate branch A), unreadable topology — and it refuses an
*inherited* `KEEL_PIN_CPUS`, which would otherwise give four arms one mask and make a null result look
real. Each arm's `cores=` readback is then checked against what the design requires, from the log
rather than from the request.

## The reading the archive is here to support

`docs/issue148-core-97a21f4.md` §5 corrects its own first draft. The median-based `smt/c0` sequence
(3.321×, 1.325×, 1.030×, 1.005×) reads as a graded recovery ordered by `flopsPerCall`, and it is not
one: the underlying distributions are complete recovery (60/60 samples), *intermittent* recovery
(20/60 recovered, 12/60 still at the confined level), and none (60/60 confined, twice). The 1.325× is
the midpoint of a switch, not a level. Reading the per-sample logs rather than the medians is what
found it — which is the argument for tracking them.

§6 is the second reason. Three of five within-arm excursions in this run are in `bref`, the pass-B
reference, whose window overlaps the one host sample whose 5-minute load average (`2.17`) no single
runnable thread explains. Both witnesses are in this directory: the excursions in the `.txt` files and
the load samples in the `.log`. The verdict is unchanged under pass A alone, pass B alone, and the
adversarial highest-reference choice.

## Conventions

**The extensions are load-bearing, and CI enforces it** (`tools/benchci`'s
`TestArchivedIntervalsNeverEscapeTheirSamples` globs `archive/*/*.txt`, re-derives every reading from
each match and fails on a file that yields none). Under `archive/`, `.txt` means *raw benchmark log*
and nothing else: the driver transcript is `.log` and the analysis output is `.out`. These eight logs
joined the corpus on arrival: 131 files / 2365 readings → **139 / 2845**, still 0 escapes.

`.gitignore` line 3 is `*.out` and line 12 is the `!archive/**/*.out` that `archive/width148/` had to
add for exactly this file; `analysis-core148.out` is tracked because of it. Line 54 (`__pycache__/`)
covers what `importlib` leaves behind when the analyzer loads the predictions module — the same class
of collision as the `addr147` `.out` incident, caught here by looking rather than by a check.

| file | role |
|---|---|
| `predictions-core148.py` | The bands, the 2×2 truth table, the estimator choice with the measurement that made it, the admissible `ref` levels, and the fourth candidate named. Committed at `4dd6334`, before the run. Imported, never restated. |
| `analyze-core148.py` | The analysis in the declared order: imported thresholds, shape, four interpretability checks that can refuse, the registered criterion under both estimators, then and only then the branch scoring. |
| `test-analyze-core148.py` | The 12 fixtures. Run it and every refusal fires. |
| `driver-core148.sh` | The launcher. Refuses an inherited mask and a dirty tree; topology preflight with four named refusals; digests the binary and compares it to `#147`'s; samples load and governor between arms and never during one. |
| `analysis-core148.out` | `analyze-core148.py`'s output, reproducing byte for byte from either this directory or `build/`. |

Reproduce from the repository root:

    python3 archive/core148/analyze-core148.py 97a21f4 archive/core148
    python3 archive/core148/test-analyze-core148.py
