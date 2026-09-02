<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#147`'s address-trace run, `537661a` on `janus.local`, 2026-09-02

The analysis is `docs/issue147-addr-537661a.md`. This directory is its evidence, tracked because an
arm cited by a path under `build/` is cited on one operator's laptop and nowhere else (`.gitignore`
ignores `/build/`).

Two arms of one binary, `sha256=d0d46d26c15cc8b2`, `go1.27.0-X:simd`, `-trimpath`:
`addr147-537661a-w8.log` / `-w8-trace.txt` at `mask=0,1,2,3,4,5,6,7 width=8`, and the `-w1` pair at
`mask=0 width=1`. `addr147-537661a.log` is the driver transcript; `driver-addr147.sh` is the launcher,
byte-identical to the copy frozen for the run's lifetime.

The instruments, and what each one is for:

| file | role |
|---|---|
| `analyze147.py` | R1–R5 and nothing else. `analysis-w8.txt` / `analysis-w1.txt` are its output and reproduce byte for byte. |
| `control147.py` | Nine synthetic rows whose verdicts are known by construction, so the verdict table is a read witness (`controls.txt`). |
| `prettycheck/` + `prettycheck147.py` | Checks the `testing.prettyPrint` transcription against the toolchain, with a positive control that must mismatch (`prettycheck.txt`). |
| `sensitivity147.py` | **Post-hoc, not registered** — an outlier-robust classifier, kept in its own file so it cannot be mistaken for the criterion (`sensitivity.txt`). |
| `crossarm147.py` | **Not registered** — the cross-arm level comparison (`crossarm.txt`). |

Reproduce any of them from the repository root, e.g.
`python3 archive/addr147/analyze147.py archive/addr147/addr147-537661a-w8.log archive/addr147/addr147-537661a-w8-trace.txt w8`.
