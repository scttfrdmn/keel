# CLAUDE.md — standing orders for building keel

You are building keel, a pure-Go float32 BLAS subset on `GOEXPERIMENT=simd`.
`DESIGN.md` is the contract: architecture (§3), phases and gates (§4),
testing philosophy (§5), risks and standing orders (§6). Read it fully
before your first action in any session.

## Toolchain
- Go 1.27 or newer with `GOEXPERIMENT=simd` (1.26.x no longer compiles
  `internal/vec` — the names were swapped, T23). Smoke-build before anything
  else each session (`make build && make stock`).
- The scalar path must always build on a stock toolchain (`make stock`), whose
  floor stays Go 1.26.
- **`make build` cannot see an amd64-only break.** The dev host is darwin/arm64,
  where the build tags exclude `gemm_amd64.go` and both vector backends outright,
  so the session-start smoke build greened through a tree that compiled nowhere a
  benchmark runs. This is not hypothetical: it is how go1.27's `archsimd` rename
  reached a $3.888/hr fleet run before it reached a compiler error. `make build`
  now cross-builds `GOOS=linux GOARCH=amd64` as its second line, so the smoke step
  covers it — do not "simplify" that back to one build.

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
- **A killed run is `unmeasured`, never an exit code.** With no status file
  `detach.sh stat` reports `died` (a log exists, so it ran and was killed) or
  `never-started` (no log at all, most often a NAME that does not match what was
  launched), because inventing a verdict for work that did not finish is the one
  failure mode this whole apparatus exists to prevent (DESIGN.md §5.6). Those
  were one word, `vanished`, until #122: it named both causes in one breath, and
  a healthy 25-minute run read as a dead one. `remote.sh`'s `REMOTE_STATE=vanished`
  is a *different* mechanism with five gate consumers and keeps its own word.
- **The tree stays frozen for a run's whole life**, detached or not — including
  across a chain of per-host invocations, each of which rebuilds its own arms.
  This covers the *running script itself*, and that is a second hazard, not the
  same one: bash reads a script incrementally, so editing one mid-execution
  shifts byte offsets under the interpreter and it resumes at the wrong place.
  Result contamination is about what the run measures; this is about whether the
  run is still the program you launched. Don't touch what the machine is still
  reading — including a fix you have already justified.

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

**`#nn` means a GitHub issue and nothing else. A task-tool id is written
`T-nn`.** Ruled 2026-08-21 after the second occurrence: task ids and this
repo's issue numbers occupy the same low integers, are syntactically
identical, and a wrong one resolves to a real issue with a plausible
subject — so the collision is structurally unlintable and the only fix is
typographic. `ddd642f` minted 17 such citations and `820eac0` almost
minted 31, both against `#33`. Never carry a bare number from the task
tool into prose, a commit message, or a comment.

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

Three rules bind what you may treat as *confirmed*, and DESIGN.md is the authority
for all three, not this file: **§5 rule 10** — agreement across N sites is one
witness, so count independent derivations before counting corroboration; **§5 rule
11** — the instrument adjudicates, so run it against the reasoning that motivated it
before publishing that reasoning; **§5 rule 12** — a coverage claim enumerates what
it cannot see, so an unkillable mutant or an unreached arm is stated inside the
number. 11 and 12 are deliberately separate: neither may be cited for the other's
job.

# Running tests and benchmarks on lab hardware (pueue + uv)

The block below is copied verbatim from `~/src/pueue-local/PUEUE.md` (Part 1), the
fleet-authoritative reference. **keel bridging note, not part of the block:** the
`## Long runs` section above governs the *local driver* on the dev host — a gate or
sweep still launches under `scripts/detach.sh` (local tmux) so the orchestrator
outlives the shell. That is not the same tmux the block forbids: the block's rule is
about the *remote, queued* command, which pueue detaches itself — wrapping THAT in
tmux frees the measured slot while the work runs. Local driver in tmux: required.
Remote queued command in tmux: forbidden. The wiring that routes measured remote runs
through pueue (retiring `remote_exec`'s own `#62` remote-tmux supervision) is a
measurement-core change pending Scott's go; until it lands, remote runs still use
remote.sh as documented there.

## Running tests and benchmarks on lab hardware

Lab machines are shared by several projects at once. Historically nothing
coordinated them, so two projects would land on the same box simultaneously and
each would see the interference as unexplained performance divergence in its own
results. All runs on lab hardware now go through **pueue**, a per-machine job
queue with a daemon on every host.

**Rule: nothing that gets measured runs outside the queue.** Anything whose
timing, throughput, or memory-bandwidth numbers will be recorded, compared, or
committed must be submitted to an exclusive group. Interactive pokes and
one-off debugging are fine to run directly, but their numbers are not results.

### Fleet and groups

A group is a named queue on one host with its own concurrency limit. Every host
has exactly three:

- **`measured`** — parallel=1. The single measurement slot on that box. Every
  run whose numbers get recorded goes here, *whatever* it exercises: Metal,
  CUDA, ROCm, or plain CPU.
- **`build`** — wide. Anything unmeasured: compiles, unit tests, lint.
- **`default`** — parallel=1. A deliberate fallback, so a job someone forgets to
  assign a group serializes instead of quietly stomping a benchmark.

`fleet.conf` is the source of truth; this table is a convenience copy.

| Host | Hardware | Accelerator | `measured` | `build` |
|---|---|---|---|---|
| `juno.local` | M4 Max, 16 core, 64GB | Metal | yes | 6 |
| `orion.local` | M4 Pro, 14 core, 48GB | Metal | yes | 6 |
| `maya.local` | M4, 10 core, 16GB | Metal | **no** | 4 |
| `indigo.local` | M4, 10 core, 32GB | Metal | **no** | 4 |
| `castor.local` | DGX Spark, GB10, 20 core, 121GB | CUDA `sm_121` (unified mem) | yes | 8 |
| `pollux.local` | DGX Spark, GB10, 20 core, 121GB | CUDA `sm_121` (unified mem) | yes | 8 |
| `antares.local` | Ryzen AI MAX+ 395, Radeon 8060S, 60GB | ROCm/HIP `gfx1151` (unified mem) | yes | 8 |
| `vesta.local` | Ryzen 9 7950X3D, RTX 4070 Ti SUPER, 61GB | CUDA — no toolkit installed | yes | 8 |
| `ceres.local` | Ryzen 9 9950X3D, RTX 5090, 123GB | CUDA `sm_120` | yes | 8 |
| `janus.local` | i9-9960X, 32 core, 62GB, 2× TITAN RTX | CUDA `sm_75` | yes | 8 |

**`maya` and `indigo` are `build`-only on purpose.** Their GUI console user is
somebody else, so they are in daily interactive use. pueue serializes queued jobs
against each other; it cannot serialize against a person. A measurement slot on a
machine you don't control is worse than none — it produces numbers that look
valid. Send compiles there, never benchmarks.

The group is not named after the device on purpose. **Exclusivity in pueue is
per group, not per host** — two parallel=1 groups on one box run at the same
time as each other, which is precisely the interference this setup exists to
prevent. Verified on 4.0.4: a task in `gpu` and a task in a second exclusive
group both went `Running` in the same second. So there is one exclusive group
per machine and it takes all measured work.

### CPU-only runs on a GPU box

Yes — submit them to that host's `measured` group like anything else. Do **not**
add a separate CPU group to get a second slot; you would get concurrency, not
isolation. A CPU-only run in `measured` waits for the GPU run ahead of it and
vice versa, which is what you want, because "CPU-only" never means "isolated":

- On `castor` / `pollux` (GB10) and `antares` (Strix Halo) the CPU and the
  accelerator share one LPDDR5X memory system. A bandwidth-hungry CPU run and a
  GPU run degrade each other badly. Exclusivity matters *more* here, not less.
- Even on `vesta` / `ceres` / `janus` with discrete VRAM, they share cores for
  the host-side driver threads, PCIe, and power/thermal headroom — a 32-thread
  CPU run will move a GPU benchmark's numbers.

If you genuinely want two things overlapping on one box, that is what `build`
is for, and its results are not results.

### CUDA

| Host | Toolkit | `nvcc` | Driver | Arch |
|---|---|---|---|---|
| `castor.local` | 13.2.2-1 | V13.2.86 | 595.84 | `sm_121` |
| `pollux.local` | 13.2.2-1 | V13.2.86 | 595.84 | `sm_121` |
| `ceres.local` | 13.2.2-1 | V13.2.86 | 595.71.05 | `sm_120` |
| `janus.local` | 12.9 | V12.9.86 | 610.57.04 | `sm_75` |
| `vesta.local` | **none** | — | 595.84 | `sm_89` |

`janus` is Rocky 9 and deliberately not on 13.2: its TITAN RTXs are Turing
(`sm_75`), a different generation from the Blackwell/GB10 hosts, so cross-host
number comparison isn't meaningful there anyway. Its driver (610.57.04, supports
CUDA 13.3) is well ahead of its 12.9 toolkit, which is the safe direction — the
error-222 trap below only bites when the driver is *behind* the toolkit.

`/usr/local/cuda/bin` is prepended to PATH in `~/.bashrc` — deliberately there and
not in an interactive rc, because pueue captures the environment at submit time
from the non-interactive ssh env, so a login-shell-only PATH is invisible to every
queued build. `/usr/local/cuda` is an `update-alternatives` symlink, so naming it
rather than a versioned directory means bumping the alternative moves the fleet.

Watch for a second `nvcc`: Ubuntu's own `nvidia-cuda-toolkit` package installs one
at `/usr/bin/nvcc` a whole major version behind (12.0.140 next to a 13.2 install on
`ceres`) and it wins on the default PATH. That package is still installed on
`ceres`; the PATH order is what keeps it from being used.

**Prefer `-arch=sm_NN` over relying on PTX JIT.** Not required any more, but still
right for benchmarking: JIT-compiling PTX at first kernel launch puts a compile
cost inside your timings. It used to be mandatory on the Sparks, and the failure
mode is worth recognising because it will recur if driver and toolkit ever drift
apart again — a default-arch build compiles cleanly and then dies at launch with

```
CUDA error 222: the provided PTX was compiled with an unsupported toolchain.
```

which means the driver's JIT is older than the toolkit that emitted the PTX. Check
`cudaDriverGetVersion` against `cudaRuntimeGetVersion`: if the driver number is
lower, that's the bug. Compiling native SASS sidesteps it; matching the driver to
the toolkit fixes it properly.

`vesta` has a driver but **no CUDA toolkit at all** — it can run CUDA binaries, not
build them.

Hostnames are `.local` (mDNS) throughout. Some also resolve bare, but the
manifest and every example use the `.local` form.

### Submitting

```bash
# fire and forget — a notification arrives when it finishes
ssh juno.local "pueue add -g measured -w ~/src/umami -l umami/$(git rev-parse --short HEAD) -- ./bench.sh --full"

# watch it
ssh juno.local "pueue status"
ssh juno.local "pueue follow 12"      # live stdout, like tail -f
ssh juno.local "pueue log 12"         # after the fact

# submit and block on the result (see scripts/labrun below)
scripts/labrun juno.local measured -- ./bench.sh --full
```

`-w` sets the working directory, `-l` a human label, and everything after `--`
is the command. Do not wrap the command in `tmux` — pueue already detaches it,
captures stdout and stderr, and survives your ssh session ending. Wrapping in
`tmux new-session -d` breaks queueing, because the task returns immediately and
frees the slot while the real work is still running.

### `scripts/labrun` — submit, wait, propagate the exit code

Use this for anything scripted or CI-like. It exists because **`pueue wait`
exits 0 even when the task failed**, so a naive submit-and-wait silently passes.

```bash
#!/usr/bin/env bash
# labrun <host> <group> -- <command...>
set -euo pipefail
host=$1; group=$2; shift 2
# not `[ ... ] && shift` — under set -e a false test there exits the script
if [ "${1:-}" = "--" ]; then shift; fi
label="$(basename "$PWD")/$(git rev-parse --short HEAD 2>/dev/null || echo nogit)"
remote_dir=${LABRUN_DIR:-$PWD}

# Two levels of quoting, both required. pueue re-joins argv with plain spaces and
# runs the result through a shell, so the command has to arrive as ONE
# already-shell-safe argument or its grouping is silently lost — `sh -c 'exit 42'`
# becomes `sh -c exit 42`, which exits 0 and reports Success.
cmd=$(printf '%q ' "$@")
id=$(ssh "$host" "pueue add -g '$group' -w '$remote_dir' -l '$label' -p -- $(printf '%q' "$cmd")")
echo "submitted $host:$group task $id ($label)" >&2
ssh "$host" "pueue wait -q $id" >/dev/null

read -r code state < <(ssh "$host" "pueue status --json" | uv run --no-project python3 -c '
import json,sys
t=json.load(sys.stdin)["tasks"][sys.argv[1]]
d=(t.get("status") or {}).get("Done")
if not d: print("99 Unfinished"); raise SystemExit
r=d["result"]
if r=="Success": print("0 Success")
elif isinstance(r,dict) and "Failed" in r: print(r["Failed"],"Failed")
else: print(1, r if isinstance(r,str) else list(r)[0])
' "$id")
[ -n "${code:-}" ] || { code=98; state=NoResult; }

ssh "$host" "pueue log $id --lines 40" >&2
echo "task $id: $state (exit $code)" >&2
exit "$code"
```

### Python: uv, always

**uv is the only Python toolchain on the fleet.** Never call the system
`python3` directly, and never use pyenv, conda, or a hand-rolled venv. Every
host is bootstrapped with a pinned uv (0.12.9) and a pinned uv-managed
interpreter (3.13.15 — the patch is pinned too) installed as the default
`python3` in `~/.local/bin`, ahead of `/usr/bin` and Homebrew. `UV_MANAGED_PYTHON=1` is
exported fleet-wide, so uv will refuse to fall back to an OS interpreter even if
one is closer at hand.

The point is the same as pinning pueue itself: an interpreter that is 3.9 on one
box and 3.14 on another turns into performance divergence that looks like it
came from your code.

```bash
# in a project with a pyproject.toml — uv resolves and syncs the environment
ssh juno.local "pueue add -g measured -w ~/src/umami -- uv run pytest -q bench/"

# a throwaway snippet with no project around it
uv run --no-project python3 -c 'import json,sys; ...'

# stdlib isn't enough? declare deps inline, no venv to manage
uv run --no-project --with numpy python3 analyze.py
```

`--no-project` matters more than it looks: without it, `uv run` walks up from
the working directory, finds a `pyproject.toml`, and syncs that project's
environment before running your one-liner. Inside a queued job that is a
surprise write and a surprise delay.

Do not export `UV_PYTHON` to force a version fleet-wide. It overrides
`requires-python` rather than deferring to it, so any project pinned to a
different version fails outright. Pin the provider, let the project pick the
version.

### Recording provenance

Every recorded result must carry the host, the group, and whether the box was
otherwise busy. Contention that isn't recorded turns into a mystery later.
Capture at minimum:

```bash
ssh "$host" "pueue status --json"   # concurrent tasks at submit time
uname -sm; hostname -s              # on the runner
```

and store `host`, `group`, `task_id`, and the concurrent-task count alongside
the numbers.

### Gotchas that will bite

- **`pueue wait` always exits 0.** Check the task result explicitly.
- **On Secure Boot hosts, never `dkms install --force` an NVIDIA module.** All the
  Linux GPU hosts have Secure Boot enabled. Ubuntu's prebuilt
  `linux-modules-nvidia-*` packages are signed with an enrolled key; a DKMS build
  is signed with a local key that is not, so it loads fine on paper and then fails
  at boot with `modprobe: ERROR: could not insert 'nvidia': Key was rejected by
  service` — GPU gone. If a kernel upgrade leaves DKMS and the prebuilt package
  fighting over the same module ("already installed, override by specifying
  --force"), the fix is `dkms uninstall nvidia/<ver> -k <kernel>` to restore the
  signed original, *not* `--force`. Install the driver and
  `linux-modules-nvidia-<ver>-open-nvidia-hwe-24.04` in one apt transaction and the
  collision doesn't arise.
- **A driver bump on the Sparks drags a new kernel with it.** The `nvidia-hwe`
  stack moves together, so verify `modinfo -k <new-kernel> nvidia` resolves to
  `kernel/nvidia-595-open/nvidia.ko` (signed, prebuilt) and not `updates/dkms/`
  *before* rebooting. Rebooting into a kernel with no matching module is precisely
  what leaves a box in janus's state.
- **Two parallel=1 groups on one host do not exclude each other.** They run
  concurrently, so adding a second exclusive group to "separate" CPU from GPU
  work buys you nothing but a false sense of isolation. One `measured` group per
  host, always. Verified on 4.0.4.
- **Commands run through a shell — twice.** pueue joins the argv you give it back
  into a single string with plain spaces, then hands that to your shell, so inner
  quoting must survive two levels of parsing. `pueue add -- sh -c 'exit 42'`
  arrives as `sh -c exit 42`, which exits **0 and reports Success**. Verified on
  4.0.4. Wrap the whole command in quotes (`pueue add -- "sh -c \"exit 42\""`),
  or use `-e/--escape`, or just use `labrun`, which handles it. Also put `--`
  before the command so pueue doesn't eat your flags.
- **The environment is captured at submit time**, from the non-interactive ssh
  environment — not your interactive shell. Anything set in `.zshrc` will not be
  there. Pass what you need explicitly on the command line.
- **Client and daemon versions must match** (pinned fleet-wide to 4.0.4). If
  pueue starts erroring after an upgrade, the daemon needs a restart.
- **uv and the interpreter are pinned too.** `pueue-fleet.sh doctor` reports
  both, and flags a `python3` that isn't uv-managed. Bump `UV_VERSION` /
  `UV_PYTHON` in `fleet.conf` and re-run `bootstrap` (or `python`) — never
  upgrade one host by hand.
- **Labels are not available to the completion notification** — only id, group,
  command, path, result, exit code, and timing. Keep `-w` pointed at the repo
  root so the notification can identify the project by directory.
