<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#148`'s width sweep, `d3c1e82` on `janus.local`, 2026-09-02

The analysis is `docs/issue148-width-d3c1e82.md`. This directory is its evidence, tracked for the same
reason `archive/addr147/` is: an arm cited by a path under `build/` is cited on one operator's laptop
and nowhere else (`.gitignore` ignores `/build/`).

Eight arms of **one** binary, `sha256=d0d46d26c15cc8b2`, 5066553 bytes, `go1.27.0-X:simd`, `-trimpath`
— byte-identical to `archive/addr147/`'s, so the `w8` and `w1` arms here are further draws of that
same artifact rather than comparable builds. `bench-width148-d3c1e82-{a,b}{8,4,2,1}.txt` are the arms,
pass `a` in width order 8→1 and pass `b` reversed; `width148-d3c1e82.log` is the driver transcript,
including the sixteen host samples.

`driver-width148.sh` was **committed before the run** (`defd87d`, one fix in `d3c1e82`), which is the
one thing this directory does differently from `archive/addr147/`: that driver had to be written into
`/tmp` mid-flight, so its design had to be reconstructed from a transcript afterwards. Here the two
choices that are not in `#148`'s pre-registration — the mirrored second pass, and measuring all 20
rows with the 12 `avx512` rows as a declared control — sit at a hash that predates every number they
are read against.

**The extensions are load-bearing, and CI enforces it** (`tools/benchci`'s
`TestArchivedIntervalsNeverEscapeTheirSamples` globs `archive/*/*.txt`, re-derives every reading from
each match and fails on a file that yields none). Under `archive/`, `.txt` means *raw benchmark log*
and nothing else: the driver transcript is `.log` and the analysis output is `.out`. These eight logs
joined the corpus on arrival: 123 files / 1885 readings → **131 / 2365**, still 0 escapes.

That convention collides with `.gitignore`, and this directory is where it showed: line 3 is `*.out`,
so `analysis-width148.out` — the file this README and the report cite as the reproducibility witness —
was skipped by `git add -A` **in silence**. `archive/addr147/`'s five `.out` files are tracked only by
accident of history: they were committed under their first, wrong `.txt` names, and renaming a tracked
file keeps it tracked. `.gitignore` now carries `!archive/**/*.out`, and both directions were checked
(`git ls-files --others --ignored --exclude-standard archive/` is empty, and so is the same command
without `--ignored`).

| file | role |
|---|---|
| `analyze-width148.py` | The whole analysis, in the order declared before the run: control, then drift, then the registered criterion. `analysis-width148.out` is its output and reproduces byte for byte from either this directory or `build/`. |
| `driver-width148.sh` | The launcher. Refuses a dirty tree; digests the binary and compares it to `#147`'s; samples load and per-core frequency between arms and never during one. |

Reproduce from the repository root:

    python3 archive/width148/analyze-width148.py d3c1e82 archive/width148
