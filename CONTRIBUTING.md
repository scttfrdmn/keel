# Contributing to keel

keel is being built in a human-directed, agent-executed loop (Claude Code
doing the typing, humans holding the gates). The process rules below bind
both kinds of contributor.

## Process
- **Phases and gates.** Work proceeds through phases P0–P5 (DESIGN.md §4),
  tracked as GitHub milestones. Each phase's gate script
  (`scripts/gate-pN.sh`) is written at the start of the phase and must exit
  0 before the next phase begins. No overrides.
- **Issues are the source of truth.** Each phase has an umbrella issue with
  its task checklist. Discoveries get **recorded**, and the record is a
  CHANGELOG line plus a commit message that says what surprised you — filing
  an issue is one way to record, not the only one. A discovery whose fix is
  smaller than the issue describing it gets **fixed**; file it when it is
  blocked, contested, needs a decision, or is too large for the session, with
  the right labels (`toolchain-report`, `perf`, `correctness`, `blocked`,
  `upstream`). The thing forbidden is a **silent** fix, never a small one.
  *(Amended 2026-08-16. Read literally, the old wording produced five
  gate-apparatus issues against one library defect in a single session, each
  issue longer than the fix it described.)*
- **Commits.** Small, single-purpose, present tense. Gate-closing commits
  use `PN: <summary> [gate green]`.

## Code rules
- All `simd`/`archsimd` imports live in `internal/vec` only.
- Every vector op has a scalar twin and a differential test before it merges.
- Tolerances come from `internal/oracle.Tolerance` only.
- Benchmark numbers are never reported without CPU model + theoretical peak.

## Versioning & releases
- [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html). While
  major is 0, minor bumps may break API.
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/): every user-visible
  change lands in `CHANGELOG.md` under `[Unreleased]` in the same PR/commit.
- **`[Unreleased]` groups by session; release sections use canonical type
  headers; the collapse happens at the version cut and never before.** The two
  formats serve two readers: session groups are provenance, and they are how
  this project's development ledger works; a release section is the deliverable,
  and users get one Added/Changed/Fixed set per version. Flattening `[Unreleased]`
  early churns thousands of lines to serve nobody — measured at 3128 on
  2026-08-19 — so the collapse is a one-time editorial pass at the tag, over
  content that has stopped moving.
- Releases are tagged `vX.Y.Z`; v0.1.0 requires gates P0–P5 green plus the
  scalar-only build passing on a stock toolchain.

## License
Apache-2.0. New files carry the two-line header:

    // Copyright 2026 Scott Friedman
    // SPDX-License-Identifier: Apache-2.0
