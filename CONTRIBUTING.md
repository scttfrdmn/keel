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

## Workstreams and labels
- **Milestones are phases through v0.1.0 and workstreams after it.** P0–P5 were
  build phases; `upstream-go1.28`, `v0.1.1-hygiene`, `v0.2.0-arm64` and
  `v0.3.0-f64` are workstreams, and an issue belongs to the one whose *release*
  needs it, not the one whose subject matter it shares.
- **`workstream/*` labels cut across milestones** (`arm64`, `f64`, `hygiene`); the
  pre-existing `upstream` label already means "candidate golang/go issue" and is
  used instead of a `workstream/upstream`.
- **`science` means unscheduled on purpose, and carries no milestone.** Open
  questions and performance leads that block no release live there rather than in
  a backlog milestone, because a backlog milestone reads like a plan.
- **`external-clock` marks work whose pace someone else sets** — an upstream review
  queue, a toolchain release. **`needs-scott`** marks work that cannot be finished
  from this repo's keyboard: anything outward-facing, and anything requiring an
  identity or an account.
- **Upstream work has a root document**, `docs/upstream-plan.md`: the CL ledger,
  the shared per-CL evidence shape, and the figures that failed verification and
  must not be cited. Read it before touching anything labelled `upstream`.

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
- **A gate certificate transfers to a tag that is not the certified rev iff
  nothing in the gate's *input closure* changed** — what it builds, executes and
  reads, computed by fixpoint over the gate script, not judged by directory name
  (canonical form ruled 2026-08-29; DESIGN.md §5 rule 22). The certified rev's
  full SHA, the delta's exact file list, and a runnable `git diff --stat` go in
  the release notes; any file that passes that test while failing a literal
  reading of the condition is named there rather than resolved silently.
- **The certificate log is *tracked* and its sha256 is *published in the release
  notes*, both at tag time.** Present and verifiable are different properties and a
  certificate needs both. v0.1.0 shipped with neither: the notes cited
  `build/release-a2-68a9bec.log` by path, `build/` is gitignored, and no repository
  under any revision held a green `gate-p5` — so the tag's own certificate resolved
  only on the laptop that produced it, and there was no digest to check a recovered
  copy against. Home is `archive/pinned8/` beside the era's other logs. Publishing
  the digest is what makes the next recovery *provable* rather than merely
  corroborated; tracking alone freezes bytes from that moment on, which is not the
  same as showing they are the bytes that were cited.
- **A markdown tag message needs `git tag -a --cleanup=verbatim -F`.** The default
  cleanup strips every `#`-leading line as a comment, so `## Heading` lines vanish
  from the tag object with no warning and nothing to diff against — repro in one
  line, `printf '## H\nbody\n' | git stripspace --strip-comments`. v0.1.0's tag lost
  all three of its section headings this way; the prose survived, including the seam
  disclosure the tag exists to carry, and the intact copies are
  `build/release-notes-vX.Y.Z.md` and the GitHub release body. Check it after
  tagging: `git tag -l --format='%(contents)' vX.Y.Z` diffed against the notes file
  should differ by exactly the signature block.
- **The `github-pages` environment refuses a tag ref until told otherwise.**
  Enabling Pages auto-creates that environment with `custom_branch_policies: true`
  and one allowed ref, the branch `main` — which contradicts a deploy job gated on
  `refs/tags/v*`, and the contradiction renders as a job that fails in ~2s with
  **zero steps** and no log, not as an error message. v0.1.0 needed
  `gh api --method POST repos/:owner/:repo/environments/github-pages/deployment-branch-policies
  -f name='v*' -f type=tag` once, permanently.

## License
Apache-2.0. New files carry the two-line header:

    // Copyright 2026 Scott Friedman
    // SPDX-License-Identifier: Apache-2.0
