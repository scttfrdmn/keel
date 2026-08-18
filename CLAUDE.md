# CLAUDE.md — standing orders for building keel

You are building keel, a pure-Go float32 BLAS subset on `GOEXPERIMENT=simd`.
`DESIGN.md` is the contract: architecture (§3), phases and gates (§4),
testing philosophy (§5), risks and standing orders (§6). Read it fully
before your first action in any session.

## Toolchain
- Go 1.27rc (or newest 1.26.x) with `GOEXPERIMENT=simd`. Smoke-build before
  anything else each session (`make build && make stock`).
- The scalar path must always build on a stock toolchain (`make stock`).

## Long runs: nothing may die, and no run is anyone's to babysit
A gate or benchmark run must not be able to fail because of the lifetime of the
shell that started it. Two gate-p5 runs were killed 25–28 minutes in, and the
conclusion drawn at the time — hand the long gates to Scott so they outlive the
agent shell — was the wrong fix. **A measurement whose completion depends on who
typed the command has a defect in its harness.** Scott's ruling: "There is NO
reason your shell should ever die."

- **Launch every long run detached, via `scripts/detach.sh`.** `tmux new-session
  -d` daemonises off the caller's process group and session, so reaping the
  caller does not reach the work:

      scripts/detach.sh run gate-p5-<rev> -- ./scripts/gate-p5.sh
      scripts/detach.sh stat gate-p5-<rev>

  This is the mechanism for the *remote* half too: `remote.sh` runs each
  benchmark synchronously over ssh, so a driver that dies SIGHUPs a measurement
  mid-flight on the far side. Keeping the driver alive keeps every ssh under it
  alive. tmux is present on the dev host and all three benchmark hosts.
- **Never `sleep` to wait.** Background the wait or read the log file; interim
  progress is free.
- **A killed run is `unmeasured`, never an exit code.** `detach.sh stat` reports
  `vanished` when there is no status file, because inventing a verdict for work
  that did not finish is the one failure mode this whole apparatus exists to
  prevent (DESIGN.md §5.6).
- **The tree stays frozen for a run's whole life**, detached or not — including
  across a chain of per-host invocations, each of which rebuilds its own arms.

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
   as they land (edit the issue body). Every discovery gets **recorded** — a
   CHANGELOG line plus a commit message naming the surprise is a record, and
   so is an issue. A discovery whose fix is smaller than its issue gets
   **fixed**; file it when it is blocked, contested, needs a decision, or is
   too large for the session, properly labeled (`toolchain-report`, `perf`,
   `correctness`, `upstream`, `blocked`). The word that carries the intent is
   *silent* — small fixes were never the objection, unrecorded ones were.
   Amended 2026-08-16: read literally, this rule produced five
   gate-apparatus issues against one library defect in one session.
3. **Session end (mandatory):** comment on the umbrella issue with (a) what
   landed, with commit hashes; (b) gate status, pasting the gate script
   output verbatim; (c) open questions for Scott, each phrased so it can be
   answered in one line. If a decision is needed to proceed, say so
   explicitly and stop there.
4. **Gate green:** close the umbrella issue with the gate output in the
   closing comment, update `CHANGELOG.md` under `[Unreleased]`, and open
   nothing in the next milestone until the commit is pushed.
5. Toolchain surprises additionally get a row in `docs/toolchain-notes.md`
   with a minimal repro before any workaround lands. The repro is never
   abridged; the prose is capped (2026-08-16) — three lines of observation,
   the repro verbatim, three lines of what changed in the tree. Causal
   analysis and rejected hypotheses go in the issue the entry cites. Same cap
   as `CHANGELOG.md` entries, and for the same reason: those two files are 70%
   of all markdown here.
6. **Search the upstream tracker before any upstream filing, always.** Before
   opening anything on `golang/go` or another external tracker, search it —
   `gh api '/search/issues?q=repo:golang/go+<terms>'`, plus label and keyword
   variants — and read the near matches including their comments and any
   linked CL. This is the `--check`-before-a-gate rule applied to the tracker:
   cheap, and it has already paid once. T18 looked like a new
   register-allocation finding and was in fact `golang/go#79984`, open
   beforehand, with keel's exact shape already reported on it and a fix CL in
   flight — so the filing would have been a duplicate *carrying a wrong causal
   story*, which is worse than no filing. When the bug is already there the
   deliverable is a `standing-task` issue keyed to the existing issue and its
   fix CL, not a new report. Comment upstream only with a fact the issue
   lacks: a measured delta on a real kernel qualifies, a second repro of a
   known miss does not.

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

## The apparatus pays its own way (2026-08-16)
The measuring apparatus outgrew the thing measured: `scripts/` is 1.63× the whole
shipping library, `gate-p3.sh` alone is 5.7× the microkernel file it checks, and
one recent session filed five gate issues against one library defect. The gates
are not the problem — they produced the mission number and caught a ratio
measured under `powersave` — the *marginal* line is.

- **A session may not add net lines to `scripts/` unless it also lands a
  routine, a kernel, or a library fix.** Process work is paid for out of the same
  budget as the work it serves, not out of a separate one.
- Enforced by review, not by a check: `gate-docs.sh` prints
  `shell N / library M / ratio R` on every push, as a *report*. It cannot fail,
  deliberately — a red ratio would reward paying down shell instead of shipping.
- The cap is on `scripts/`, so a fix whose honest form is a comment is still
  allowed; it just spends budget. Prefer deleting a line to explaining one.

## Honesty over momentum
When something surprises you — a lowering, a spill, a flaky benchmark — the
deliverable is the documented surprise, not a quiet workaround. This project
is partly a field report on `GOEXPERIMENT=simd`; the notes have independent
value even where keel itself stalls.

Two rules bind what you may treat as *confirmed*, and DESIGN.md is the authority
for both, not this file: **§5 rule 10** — agreement across N sites is one witness,
so count independent derivations before counting corroboration — and **§5 rule 11** —
run a fixture against the reasoning that motivated it before publishing that
reasoning, and document a mutation that cannot be killed in principle as such.
