# CLAUDE.md — standing orders for building keel

You are building keel, a pure-Go float32 BLAS subset on `GOEXPERIMENT=simd`.
`DESIGN.md` is the contract: architecture (§3), phases and gates (§4),
testing philosophy (§5), risks and standing orders (§6). Read it fully
before your first action in any session.

## Toolchain
- Go 1.27rc (or newest 1.26.x) with `GOEXPERIMENT=simd`. Smoke-build before
  anything else each session (`make build && make stock`).
- The scalar path must always build on a stock toolchain (`make stock`).

## The prime directive on the simd API
Never write or edit anything in `internal/vec` from memory. First run
`go doc simd/archsimd` and `go doc simd`, and read the sources under
`$(go env GOROOT)/src/simd/`. The API is experimental and has had breaking
renames between releases; identifiers recalled from training are
presumptively wrong. Copy exact names from `go doc` output.

## Phases and gates
- Work phases P0→P5 in order. Each phase's gate script
  (`scripts/gate-pN.sh`) is written at the START of the phase — replace the
  stub with real checks, then make them pass. Never weaken a gate to pass it.
- Commit on every green gate: `PN: <summary> [gate green]`.
- Never begin phase N+1 with a red gate. There is no override flag; do not
  add one.
- **P2 is a go/no-go, not a hurdle.** If the spill audit or the
  55%-of-peak floor fails after the documented kernel-shaping steps and one
  tile-shrink: STOP. Write `docs/spill-report.md` with the ssa.html
  evidence and assembly excerpt, open a `blocked`-labeled issue, and end
  the session asking for a decision. Do not proceed; do not switch to
  assembly on your own initiative.

## GitHub is the shared workspace (the back-and-forth channel)
Scott reviews progress through GitHub between sessions. Keep it truthful
and current using `gh`:

1. **Session start:** `gh issue list --milestone "<current phase>" --state open`
   and read new comments on the umbrella issue — Scott's course corrections
   arrive there. Treat unresolved questions from him as blocking input.
2. **During work:** check off items on the phase umbrella issue's task list
   as they land (edit the issue body). New discoveries become their own
   issues, properly labeled (`toolchain-report`, `perf`, `correctness`,
   `upstream`, `blocked`) — never silent drive-by fixes.
3. **Session end (mandatory):** comment on the umbrella issue with (a) what
   landed, with commit hashes; (b) gate status, pasting the gate script
   output verbatim; (c) open questions for Scott, each phrased so it can be
   answered in one line. If a decision is needed to proceed, say so
   explicitly and stop there.
4. **Gate green:** close the umbrella issue with the gate output in the
   closing comment, update `CHANGELOG.md` under `[Unreleased]`, and open
   nothing in the next milestone until the commit is pushed.
5. Toolchain surprises additionally get a row in `docs/toolchain-notes.md`
   with a minimal repro before any workaround lands.

## Code rules (recap; full text in CONTRIBUTING.md and DESIGN.md)
- All simd imports in `internal/vec`; scalar twin + differential test before
  any vector op merges.
- Tolerances only via `internal/oracle.Tolerance`; change the model with a
  numerics comment, never a single test's epsilon.
- Kernels: no calls in the K-loop, pre-sliced panels (verify BCE with
  `-gcflags=-d=ssa/check_bce`), pointer-free data.
- Benchmark numbers always ship with CPU model, theoretical peak, and the
  OpenBLAS reference when available; otherwise say it isn't and report
  percent-of-peak only. Never a number without its denominator.
- New files carry the two-line copyright/SPDX header. Every user-visible
  change lands in `CHANGELOG.md` `[Unreleased]` in the same commit.

## Honesty over momentum
When something surprises you — a lowering, a spill, a flaky benchmark — the
deliverable is the documented surprise, not a quiet workaround. This project
is partly a field report on `GOEXPERIMENT=simd`; the notes have independent
value even where keel itself stalls.
