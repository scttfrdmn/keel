# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, minor versions may contain breaking changes.

## [Unreleased]

### Added
- `BenchmarkBlocking`'s grid is replaceable per axis from the environment
  (`KEEL_BLOCKING_KC`, `_MC`, `_NC`, comma-separated; `scripts/retention.sh sweep`
  forwards them, since sshd strips arbitrary env). A fine scan around a suspected
  point defect — janus at 2×32 has one at kc=384 — is then the same benchmark at a
  different grid rather than a second benchmark, and a 5-point scan is small enough
  to afford the full `-count=10` methodology, which sharp associativity effects need:
  a coarse grid can alias them into fiction in either direction. A malformed value is
  fatal rather than defaulted, because a sweep that silently measures a grid other
  than the one it was asked for is #49 again; and since every point names its own
  KC/MC/NC, the grid that actually ran stays readable out of the log. The sweep header
  no longer tells a reader of a `-count=10` log to re-measure it at 10, and now says
  what is true at any count: a top row here becomes a default only through #24's
  `kern.Class`, never by winning a sweep.
- `BenchmarkFeed` and `scripts/retention.sh feed`: the per-call decomposition of
  the blocked nest resolved against KC, which is the question the KC/MC/NC sweep
  left open (#48). The sweep found that janus's per-call penalty *rises* with KC
  (39.7 → 69.3 → 72.3 ns/call), so some term scales with a call's own duration
  rather than with the number of calls — and the two candidates, real C traffic
  and panel feed, both have KC-independent *totals*, so their size cannot separate
  them and only their shape against KC can. Six arms per (shape, KC), each one
  variable from the last, all making the identical call multiset at identical
  depths: `kernel-calls`, `loops` (the macro loops' own address arithmetic),
  `cold-c` (real C at the nest's own tile addresses), `cold-panels` (real panels
  at macro's own offsets), `nest-no-pack`, `full`. Per-call cost is then a
  measurement divided by an exact count and each column is a difference of two
  measured quantities — no slope, no fit through grid points. The six steps sum to
  `full − kernel-calls` by construction, so a hidden term shows up as a residual
  rather than being absorbed. The pack column's intercept is the term that
  separates the sweep's slope from the decomposition's per-call cost, which was
  previously argued rather than measured.
- Project skeleton: module layout, phase gate scripts, GitHub project bootstrap,
  CI (stock + `GOEXPERIMENT=simd` builds), design document (`DESIGN.md`).
- `internal/vec`, the simd shim, with all three backends over the ~14 ops SGEMM
  and the Level-1 routines need. Scalar backend is the executable spec and
  builds on a stock toolchain on every GOARCH; AVX-512 and AVX2 backends are
  written against the `go doc`-read archsimd API of go1.26.5.
- Differential test harness binding every vector backend to the scalar spec at
  **bit-exact** equality (the spec was written to match the vector units:
  single-rounding `MulAdd`, pairwise-halving `HSum`, sign-bit-masking `Abs`), over
  an adversarial pool covering NaN, ±Inf, denormals, −0, partial and empty
  slices, and every offset across the width boundary.
- Characterization tests pinning the scalar spec's own semantics, which run on a
  stock toolchain and on every GOARCH — including witnesses that fail if
  `MulAdd` stops being fused or `HSum` stops folding pairwise.
- `scripts/gate-p0.sh`: real P0 checks (both builds, vet, shim tests with
  enforced backend coverage, and FMA fusion verified by disassembly). Now
  scores one *execution target* per machine and requires that all three
  backends ran together on real silicon, naming which machine did it.
- `scripts/remote.sh`: ships a cross-compiled static `go test -c` binary to an
  amd64 host over ssh and runs it, so a `darwin/arm64` dev host can execute
  AVX-512 code — and, from P1 on, benchmark it — with no Go toolchain
  installed on the target. Hosts come from `.keel-hosts` (gitignored) or
  `$KEEL_REMOTE_HOSTS`; see `docs/hosts.md`.
- `docs/hosts.md`: the three amd64 execution targets, what each one is good for, and
  the Zen 4 double-pumped-AVX-512 caveat that P2's percent-of-peak denominator
  has to account for.
- `docs/toolchain-notes.md` T8: three ways a register-only FMA-saturation
  microbenchmark silently stops measuring a hardware ceiling — CSE merging
  identical accumulator chains into one (an ~8× understatement, hence an ~8×
  inflation of everything divided by it), `VFMADD213PS`'s clobbered multiplicand
  costing 26 register copies per iteration in the natural accumulator form, and
  constant multiplicands deleting the multiply while adding a load. Each with a
  verified repro. None is a compiler bug; all three read correctly in Go source
  and are wrong only in the disassembly, always in the flattering direction.
- `docs/toolchain-notes.md`: seven earlier field notes on `GOEXPERIMENT=simd` in go1.26.5,
  each with a minimal repro — archsimd being amd64-only, the `VFMADD213PS`
  lowering, the absent float32 `Abs`/bitwise ops, the absent `Float32x16`
  horizontal reduce, the absent portable `simd` package, the `Max`/`Min` NaN
  operand order, and `GOAMD64` not gating archsimd intrinsics.

- **Level 1 BLAS: `Sdot`, `Saxpy`, `Sscal`, `Snrm2`, `Sasum`, `Isamax`.** Unit
  stride dispatches to `internal/l1`'s per-backend kernels (one indirect call
  per call, none inside a loop); non-unit and negative strides take scalar
  loops in the `keel` package. `KEEL_FORCE=scalar|avx2|avx512` pins the
  backend and panics rather than silently downgrading if it is unavailable.
- `internal/l1`: unit-stride kernels for all three backends. Four independent
  accumulator chains per reduction so FMA latency does not serialize them — in
  the scalar path too, so the ≥4× gate is measured against what a competent Go
  programmer would write rather than a straw man. Remainders use masked partial
  loads/stores instead of a scalar tail.
- `internal/oracle`: float64 references for all six routines. Each reduction
  returns its value *and* the error scale (Σ|xᵢyᵢ| for a dot product), because a
  cancelling dot product has a tiny result and a large legitimate error bound,
  and a test that substitutes |result| for the scale is testing nothing.
- `Snrm2` over/underflow rescue: the vector kernel runs unguarded and its
  *result* is inspected — a sum of squares is monotonic, so one post-loop check
  catches everything a per-element check would, without a branch per element.
  Overflow or total underflow reruns in float64, where float32 inputs cannot
  overflow for any n below ~1e230, so the rescue is exact by construction.
- Level-1 test suite: every routine against the oracle on every backend the
  machine can execute, cross-backend differential at 2× tolerance (triangle
  inequality, not a fudge factor), 32 shapes chosen around the 8/16/32/64
  lane and unroll boundaries, six data patterns including cancelling and
  subnormal, strided and negative-strided coverage with poisoned gaps, NaN/±Inf
  propagation swept across body/remainder/masked-tail positions, and the
  argument-validation panics.
- `bench/`: benchmark harness reporting GFLOP/s with CPU model, core count,
  governor, clock snapshot, active backend, and the host's measured FMA peak with
  the formula printed beside it as a cross-check.
- `scripts/gate-p1.sh`: real P1 checks — both builds, vet, L1 tests with
  enforced per-backend coverage on every host, the whole suite re-run under
  `KEEL_FORCE=scalar` on machines that *have* AVX-512 (a scalar pass on arm64
  would not prove the override works), and the ≥4× Sdot ratio measured
  within-machine under the §5.4 methodology — benchstat median, cleared net of
  its confidence interval, on every host, with at least one clearing it under the
  `performance` governor. It also measures and prints each host's FMA peak and
  512/256 width ratio: not a P1 criterion, but P2 divides by that number, so both
  phases' figures share a measurement regime from the start.

- **A measured percent-of-peak denominator** (`internal/vec/peak.go`,
  `bench/BenchmarkPeak`), replacing the DESIGN.md formula. Register-only FMA
  saturation: no memory in the loop, twelve independent accumulator chains at
  512 bits (ten at 256 and ten scalar), each starting at a distinct value so CSE
  cannot merge them, and the accumulator in the destination operand so the
  lowering needs no register copies. Verified three ways — disassembly of the
  steady-state loop, an exact arithmetic witness
  (`TestPeakChainsAreIndependent`) that fails on any host if a chain does not
  survive compilation, and the same witness re-checked inside the benchmark that
  produces the number.
- `scripts/bench.sh`: the one gate benchmark methodology, shared by every gate
  so none can deviate from it — `-count=10 -benchtime=1s`, aggregated by the
  `benchstat` pinned as a module tool, thresholds cleared net of the reported
  confidence interval, and no verdict at all when benchstat cannot bound the
  distribution.
- `internal/spill`, the P2 spill audit: parses `go build -gcflags=-S`, identifies
  the steady-state K-loop (innermost loop carrying the arithmetic, excluding the
  `runtime.morestack` re-entry jump that spans every function), and classifies its
  body. It separates the three things a grep conflates: a *spill* (a vector
  register moved to or from `(SP)`), a *register copy* (`VFMADD213PS`'s clobbered
  multiplicand, an issue slot rather than memory traffic), and a *broadcast*
  (arithmetic setup). Also counts the `XCHGL AX, AX` statement anchors, which a
  listing parser cannot drop because they are not spelled `NOP`.
- **Bounds-check elimination is checked on the loop body, not on the package.** The
  audit reports a surviving bounds check as what it actually is in the object code
  — a conditional branch out of the loop body to a block that calls
  `runtime.panic*` — because the panic block is laid out after the hot path, so
  grepping the body for a `CALL` finds nothing. `-d=ssa/check_bce` is printed as
  provenance and is not the criterion: it reports dozens of legitimate checks
  outside the K-loop (`a[:kc*MR]` in the prologue, `c[i*ldc:i*ldc+NR]` in the
  write-out), each of which costs nothing amortized over K, so a gate that failed
  on them would be unsatisfiable for reasons unrelated to P2. Verified both ways
  against a positive and a negative control.
- **SGEMM microkernels** (`internal/vec/gemm_amd64.go`, registry in
  `internal/kern`): two zero-spill AVX-512 tiles, 2×32 unrolled ×4 and 4×32
  unrolled ×1, both differential-tested against a scalar twin of their own shape
  under `oracle.Tolerance`, over kc values covering 0, the sub-unroll cases, exact
  multiples, remainders, and 128, with guard rows and columns checked untouched.
  Both audit clean: 0 spills, 0 calls, 0 surviving bounds checks in the
  steady-state loop.
- `KERNEL.md`: the tile shape record P2 exists to produce — the register budget,
  the measured spill frontier over a 115-shape sweep, the full 74-instruction
  opcode histogram of the 2×32 body, the two shaping attempts that were measured
  and dropped, and the per-host winner.
- `bench/BenchmarkKernel`: every shape on packed L1-resident panels at kc ∈ {8,
  32, 128, 512}, reporting GFLOP/s (shapes do different work per call, so ns/op
  cannot compare them) and flops/call. Documented as an *upper bound* on what P3
  can deliver, because excluding packing and blocking is what makes it a
  measurement of the K-loop.
- `docs/spill-report.md`: the P2 go/no-go report, required by DESIGN.md §4/P2 when
  the gate is red. **The spill audit passed and the flat 55%-of-peak floor did
  not** — janus.local (Skylake-X) reached 46.1% against the floor, while vesta
  (Zen 4) reached 96.6% and antares (Zen 5) 64.1%. The binding constraint is
  instructions issued per FMA, not spills, established two ways from keel's own
  measurements: a clock-free test (the two shipped shapes' throughputs stand in the
  inverse ratio of their instruction counts on janus, 1.308 measured against 1.351
  predicted, and do not on the other two hosts) and a per-cycle derivation (both
  janus shapes return the same ~4.2 instructions per cycle despite differing 35% in
  instruction count). §7 listed four open decisions and stated which were not taken;
  the ruling on issue #19 took the first, and §9 now records the amended gate model
  and why it tightens rather than expires.
- **zmm FMA/cycle is now measured per host, not cited**: `BenchmarkPeak/avx512`
  pinned with `taskset` while sampling that core's `cpufreq/scaling_cur_freq` gives
  0.996 on vesta, 1.944 on janus, 1.996 on antares — Zen 4 double-pumping AVX-512
  over 256-bit datapaths, and both the Skylake-X and Zen 5 parts retiring two
  full-width FMAs per cycle. This is the column that explains why one host is
  issue-bound: a 2-FMA/cycle machine feeds twice the arithmetic from the same front
  end, so it has half the instruction budget per FMA
  (docs/spill-report.md §3.3).
- `KERNEL.md` §7, the per-host winner: **the winner flips, so both shapes ship.**
  The load-lean 4×32 wins on vesta (96.6% vs 92.4%) and antares (64.2% vs 53.1%);
  the instruction-lean 2×32 wins on janus (46.0% vs 35.2%). Shipping one shape on
  theory would have been wrong on at least one machine in this fleet, and the shape
  theory most favoured wins on the fewest hosts.
- `docs/toolchain-notes.md` T12 (issue #20): **the K-loop's ideal instruction
  exists in Go's assembler and cannot be reached from Go.** `go tool asm` plus
  `llvm-mc` confirm `VFMADD231PS.BCST 12(SI), Z1, Z0` encodes as the seven bytes
  `62 f2 75 58 b8 46 03` — EVEX.512, embedded broadcast, accumulate in place,
  memory as a *multiplicand*. Three independent reasons the intrinsic layer cannot
  emit it: only 213-shaped FMA SSA ops exist for vectors (scalar `VFMADD231SS/SD`
  do exist, for `math.FMA`); the one load-merge rule that exists
  (`simdAMD64.rules:2774`) can only fold memory into the 213 form's *addend*, which
  in a GEMM is the accumulator and the single operand that must stay in a register;
  and nothing under `ssa/_gen/` emits `.BCST`, though `obj/x86/evex.go` supports it.
  This is the largest term in the 2×32 budget — 74 → ~46 instructions, 4.625 →
  2.875 insns/FMA — and it supersedes `docs/spill-report.md` §5's original
  accounting, which credited only T9 and T10 and was therefore short by ~1.75×.
- **Three field notes are filed upstream against `golang/go`**, each with a
  self-contained repro built from scratch for the filing and re-verified against the
  go1.26.5 GOROOT, and each carrying janus's roofline table as its impact statement:
  [#80828](https://github.com/golang/go/issues/80828) (512-bit values allocated from
  15 of 32 zmm registers — a fresh sweep puts the zero-spill frontier at 13
  independent accumulators and shows no register above Z14 is ever named),
  [#80829](https://github.com/golang/go/issues/80829) (no 231-shaped vector FMA, the
  load-merge rule folds the addend, nothing emits `.BCST` — with the byte-identical
  `go tool asm` / `llvm-mc` encodings and a 9-instructions-for-2-FMAs GEMM row),
  [#80830](https://github.com/golang/go/issues/80830) (`BroadcastFloat32x16` is
  emulated as `SetElem`+`Broadcast1To16`; a one-line wrapper costs one anchor NOP per
  call site — 7 insns/0 NOPs direct against 11/4 wrapped, for identical arithmetic).
  Recorded in `docs/toolchain-notes.md` beside T9, T10 and T12 and in
  `docs/spill-report.md` §9.
- `scripts/roofline.sh`: the throughput verdict as a single pure function with no
  I/O, so the rule that decides a go/no-go can be read in one place and tested
  without a benchmark. Classifies a host FMA-bound or issue-bound from measured
  `(fraction-of-peak, audited insns/FMA)` pairs and returns
  `CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY`. The roofline is clock-free: with
  `f_i` the measured fraction of peak and `I_i` the audited insns/FMA,
  `roofline(I) = maxᵢ(f_i·I_i)/I` — the retirement rate cancels, so no clock,
  `taskset` or perf counter enters the gate.
- `scripts/roofline-test.sh`: 15 adversarial fixtures for that function, run by
  `gate-p2.sh` *before* any benchmarking, so a broken decision rule fails the gate
  on any host in a second. Fixtures feed measured `(f, I)` pairs rather than
  pre-reduced spreads, so a fixture cannot describe a host that could not exist —
  which is how one of the first hand-written negative controls was caught being
  fake. They include a kernel padded with 40 dead instructions trying to buy itself
  a roofline, a slow kernel on a wide host, a sandbagged alternate shape, a
  single-mix host, both sides of the 90.0% and +5.0% boundaries, and the post-T12-fix
  janus that needs 70.4% and the one that only makes 76.7% of its roofline.
- `docs/toolchain-notes.md` T11: `GOSSAFUNC` is not part of the build cache key,
  so a repeated build is a cache hit that writes no `ssa.html` — while replaying
  the cached compiler stderr, including `dumped SSA for <fn> to ./ssa.html`. Found
  by the gate requiring the archived dump and checking for the file instead of
  trusting the exit status: the requirement passed on one run and failed on the
  next, the audit's own `-gcflags=-S` compile having warmed the cache in between.
  `spill-audit` now gives each dump a private `GOCACHE`, which is in the lookup —
  0.63 s and 18 MB per function, discarded after.

- **`Sgemm`: the full Level-3 routine, row-major, all four transpose
  combinations, general alpha and beta.** `internal/pack` produces packed panels
  and `internal/block` drives the Goto/BLIS loop nest NC→KC→MC→NR→MR (`KC=384`,
  `MC=144`, `NC=4096`, exported vars, clamped to the problem and rounded down to
  whole tiles). Panels are k-major — `a[p*MR+i]`, `b[p*NR+j]` — so the P2-audited
  kernels read contiguously along the depth loop and nothing in the K-loop
  computes an address from a stride.
  - **alpha is folded into the packed A** (BLIS convention): O(mc·kc) multiplies
    instead of O(mc·nc·kc), it stays out of the audited K-loop entirely, and the
    extra rounding is covered by `oracle.Tolerance` — the float64 oracle folds it
    the same way, so the sweep tests the arithmetic keel actually performs.
  - **beta is applied outside the kernel**, once per C block before the first
    KC panel, in three variants selected by value (`beta == 1` returns,
    `beta == 0` clears, otherwise scales). No branch on beta inside any loop, and
    no separate kernel per variant.
  - **Edges are zero-padded panels plus a temp tile**, decided from the read API
    rather than from habit (issue #4, numbers deferred to issue #22). Masking is
    genuinely available in go1.26.5 — mask types, `Masked`/`Merge`,
    `LoadMasked`/`StoreMasked`, `LoadFloat32x16SlicePart`/`StoreSlicePart` — but a
    masked C update means a second kernel family, doubling what has to stay
    zero-spill under P2's audit that the P3 gate re-runs. So fringe tiles run the
    same kernel over zeros into an MR×NR scratch buffer and the valid
    sub-rectangle is copied back: one K-loop, byte-identical for interior and edge
    tiles. The padding is written, never assumed, because the buffers are reused
    across blocks. It also discards the `0·Inf = NaN` a padding lane can produce,
    which `TestSgemmNonFinite` pins.
  - **Packing is `copy` in the contiguous direction and a scalar loop in the
    transposing one**, also from the read API: `grep -l -i 'gather\|scatter'`
    over `$GOROOT/src/simd/archsimd/*.go` returns nothing at any width, so a
    strided store is not expressible. A 16×16 in-register transpose is
    constructible from `Permute`/`ConcatPermute`, which do exist; that is a
    permutation network to write, test and audit for a routine whose cost is
    O(mc·kc), so it waits for a measurement (issue #21).
  - Argument errors panic, as everywhere else in keel; `ld >= max(1, cols)` is
    checked even for an empty matrix, matching reference SGEMM's `LDA >= MAX(1,…)`.
    `k == 0` is the empty product (`C = beta*C`), `alpha == 0` never reads A or B
    so a NaN there cannot reach C, and `beta == 0` never reads C so an
    uninitialized destination is legal.
- **`Sgemm` differential suite against the float64 oracle** (`gemm_test.go`,
  `internal/oracle/gemm.go`): sizes 1–17, 63, 64, 65, 500, 1000, 2048 × {NN, NT,
  TN, TT} × alpha {0, 1, −0.75} × beta {0, 1, 0.5}. The full 36-combination
  lattice runs at every size where a fringe tile can occur (≤65) and one rotating
  combination at 500/1000/2048, because 36 combinations at 2048³ is 620 GFLOP per
  runner and what the large sizes exercise is the loop nest, not the flag
  handling. Every element is checked at or below 65; above it, four corners plus
  256 seeded samples. Each kernel shape in `kern.Kernels()` is run as its own
  runner alongside the public path, so a tile cannot ship untested. Plus leading-
  dimension padding with poison in the gaps (checked untouched afterwards),
  zero dimensions, 14 argument-panic cases, and the non-finite cases above.
- `internal/pack` differential test against a straightforward reference packer,
  420 combinations over both source geometries — it lives in the root package so
  its coverage marker comes from the same binary the gate ships to each host.
- `bench.BenchmarkSgemm` at n = 256/512/1024/2048, and an OpenBLAS reference
  (`BenchmarkOpenBLAS`, build tag `openblas`) making the identical call. The tag
  keeps it out of the module graph: nothing keel ships links a BLAS. It prints the
  library's own report of itself — version and build flags, `DYNAMIC_ARCH`-selected
  kernel family, thread count, and the CPU count that thread count restricts — so
  the denominator is identified rather than merely named. Compile- and run-verified
  against Homebrew OpenBLAS 0.3.34 on the dev host; the criterion itself needs each
  amd64 gate host to have both a Go toolchain and the library, which is still an
  open provisioning decision (`docs/hosts.md`, issue #23).
- `scripts/provision-openblas.sh`: installs the same-host OpenBLAS reference and a
  `GOEXPERIMENT=simd`-capable toolchain on each gate host, then *verifies* by
  building the `openblas`-tagged harness there and printing the marker the gate
  will check — version, `OPENBLAS_NUM_THREADS=1` read back from the library, and
  the selected kernel family against the gate's own allowlist. Separate from the
  gate on purpose: it needs interactive `sudo`, which the gate's `BatchMode=yes`
  connection cannot answer and should not try to. It handles no credentials, and it
  reports a non-`performance` governor rather than changing a machine's power
  policy — but reports it as a **failure**, since the gate now refuses such a host
  outright (issue #31): a machine whose library and toolchain are fine and whose
  governor is wrong is provisioned-and-unmeasurable, and finding that out here is
  cheaper than finding it out in a gate run. The Go tarball's digest is enforced
  against `$KEEL_GO_SHA256` when set and otherwise against `go.dev/dl?mode=json`,
  and it says which of the two it did, since only the first is provenance. That
  fallback lookup had never once run (issue #29): go.dev serves pretty-printed
  JSON, so splitting the objects on `{` alone left each object's fields on separate
  lines and the line-oriented `grep` matched only the `"filename"` line, never the
  neighbouring `"sha256"`. The expected digest was therefore empty for every
  version ever requested. It failed closed, so nothing unverified was ever
  installed — but `install_go` could not complete without `$KEEL_GO_SHA256`, which
  is why the check's silence went unnoticed: the visible symptom was a refusal, not
  a pass. It distinguishes "no usable toolchain"
  from "usable toolchain, not on the PATH the gate uses" and links the latter rather
  than deleting it (issue #27): probing only the ssh `PATH` would have had it
  `sudo rm -rf` a working go1.26.5 on antares to reinstall the same version, and the
  one irreversible action it can take on a host is now named in the prompt that
  authorizes it. Its `[y/N]` prompts read from `/dev/tty` and its host loop from
  fd 3 (issue #28): both used stdin, so each prompt consumed the *next host* as its
  answer — three hosts named meant one was never visited, having been spent as a
  keystroke, and the interactive path had therefore never worked at all. This is the
  hazard `scripts/remote.sh` already documents and defeats with `ssh -n`, in the one
  script that cannot use `-n` because `sudo` needs the `-t` tty. No tty and no
  `--yes` now fails with its own message instead of printing `skipped`, which had
  reported a decision nobody was asked to make.
- `docs/toolchain-notes.md` T13: `import "C"` in a `_test.go` file is rejected
  outright (`use of cgo in test … not supported`,
  `cmd/go/internal/modindex/read.go:589`), and it fails as `[setup failed]` — which
  in a script reads like a missing library rather than a rejected file layout. Not
  a simd note and not new, recorded because it shaped `bench/`'s file split: the
  cgo binding is a package file, the benchmark a test file, and since package files
  cannot see `_test.go` identifiers the provenance variable is set from the tagged
  test file's `init`.
- `docs/toolchain-notes.md` T14 (issue #25): `archsimd` reports CPU *features* and
  nothing else — no vendor, family, model or brand string — and the data is not
  merely unexported one layer down, since `internal/cpu` calls `cpuid(1, 0)` and
  keeps only `ecx`, discarding the signature word. Kernel *shape* selection is a
  per-µarch decision (KERNEL.md §7: the winner flips between hosts with identical
  feature sets), which is why every production BLAS dispatches on vendor plus
  family/model, so on this toolchain keel has to fingerprint a feature bundle
  instead. Recorded before the workaround, with the two upstream shapes that would
  retire it.
- `docs/toolchain-notes.md` T15 (issue #32): `go test -bench` splits a pattern on
  top-level `|` **before** `/`, so `A|B/c` is the alternation `{A}` or `{B,c}` and
  not the two-level filter `{A,B}` then `{c}` that it reads as. An alternative with
  fewer elements than the name is depth-unconstrained, and one with more matches no
  benchmark while still matching the *parent* partially — `simpleMatch.matches`
  returns `ok, partial = true, true` when `len(name) < len(m)` — so the parent is
  entered, prints its `init` output, and yields no result row. Not a compiler bug
  and not simd-specific; recorded for the failure mode, which is a filter that reads
  correctly to every reviewer, runs without error, and silently measures something
  other than what it names. Parentheses suppress both splits, which is the fix.
  Repro on janus, plus the audit of every filter in the repo.

- **Level 2 BLAS: `Sgemv` (both transposes) and `Sger`.** Both are row-at-a-time
  loops over `internal/l1`'s unit-stride kernels — a dot product per row of A
  untransposed, an axpy per row transposed, and an axpy per row for the rank-1
  update — so there is one indirect call per row of A and none inside a loop.
  Strided vectors are gathered into a contiguous buffer once per call rather than
  handled by a strided inner loop: the gather is O(n) against the routine's
  O(m·n), and it keeps the kernel path the one P1 measured.
  - Two documented deviations from reference SGEMV, both toward the rule the
    float64 oracle can check element by element. An empty reduction still applies
    beta (`y := beta*y` is the value of the expression when the sum is empty — the
    same rule as `Sgemm`'s `k == 0`, where reference returns early); and alpha
    multiplies the dot product rather than each element of the row.
  - **No zero-multiplier guard in either routine.** Reference SGER has one
    (`IF (TEMP.NE.ZERO)`) and reference SGEMV does not; keel takes the unguarded
    rule for both, because `0·Inf = NaN` must propagate and a skipped row would
    disagree with the oracle on exactly that input.
- **`Ssyrk`, `Ssymm` and `Strsm` as derivations on the P3 loop nest**, not as
  second implementations of it. Each one inherits the packing, the blocking
  parameters, the three beta variants, the zero-padded-panel edge strategy and the
  P2-audited K-loop; `scripts/gate-p4.sh` criterion 5 checks that inheritance from
  markers rather than trusting it, requiring every derived routine to report the
  same microkernel and the same `kc`/`mc`/`nc` that `Sgemm` dispatched in the same
  process.
  - **`Ssyrk` is one GEMM call with A on both sides and the C update masked to a
    triangle** (`internal/block/tri.go`). The mask is over C's *global* indices, so
    the loop nest threads the block offsets down to the macro-kernel; blocks
    entirely outside the triangle are never packed and never reach the kernel, and
    a tile that straddles the diagonal runs the existing scratch-tile path — the
    one the fringe already uses — and copies back a row range instead of a row.
    `triMask{on: false}` folds every predicate to a constant, so `Sgemm` pays
    nothing for the mask's existence. No new kernel family, and none of P2's audit
    surface is widened.
  - **`Ssymm` reflects the stored triangle of A into a dense square and is then one
    unmasked GEMM.** The cost is O(d²) of scratch and one extra pass, against the
    O(d²·n) of the multiply; pack-time mirroring is strictly better, also covers a
    future `Ssymv`, and is issue #36. `alpha == 0` returns `beta*C` without
    allocating or reading A, which is a correctness requirement rather than a
    shortcut: A's unreferenced triangle may hold anything.
  - **`Strsm` is the BLIS blocked recipe** (Van Zee & van de Geijn, TOMS 2015 §4.3;
    Goto & van de Geijn, TOMS 2008 §4): a GEMM rank update against the
    already-solved blocks, then an unblocked solve against one `MB`×`MB` diagonal
    block, in whichever of the four directions `side` and `uplo != trans` select.
    All the flops except the diagonal blocks' `O(m·MB·n)` go through the audited
    kernel. The diagonal solves divide rather than multiply by a reciprocal (a
    reciprocal changes the last bit and disagrees with the oracle for no gain) and
    do not skip a zero multiplier (0·Inf must propagate). They are scalar, `MB` is
    an untuned `var`, and both are issue #37.
  - `unit` diag means the stored diagonal is **not referenced**, not that it
    contains ones — the guarantee a caller holding an LU factorization in one array
    relies on. `alpha == 0` zeroes B without reading A at all, so a singular or
    infinite diagonal is legal on that call.
- **P4 differential suite against the float64 oracle** (`gemv_test.go`,
  `tri_test.go`, `internal/oracle/gemv.go`, `internal/oracle/l3.go`), over the
  P4 size list 1–17, 31, 32, 33, 63, 64, 65, 500.
  - The full flag lattice per routine: `Sgemv` 162 combinations (trans × alpha ×
    beta × incx × incy, each stride being unit, wider-than-one and negative),
    `Sger` 27, `Ssyrk` 36, `Ssymm` 36, `Strsm` 48 (side × uplo × trans × diag ×
    alpha). Level 2 runs its lattice at *every* size including 500 and verifies
    every output element there, because its entry-wise oracle is O(n) per output —
    an exhaustive comparison costs the same order as the routine. Level 3 runs the
    full lattice up to 65 and one flag corner rotated by runner index above it, and
    the reduction is stated in the markers rather than implied.
  - **Coverage is counted, not declared.** Each routine records the flag sets it
    swept *and the set of tuples it actually reached*; the gate multiplies the
    former and requires the product to equal the count of the latter. P3 could
    print its constants because `Sgemm` has one flag pair; with five routines and
    sixteen corners the interesting failure is no longer "the sweep is too small",
    it is "the sweep declares a lattice and skips part of it" — which no test
    failure would ever show.
  - **The properties an oracle comparison cannot see**, each about memory the
    routine must not touch: `Ssyrk`'s untouched triangle of C (poisoned and
    required back *bit-identical*, not merely close); `Ssymm`'s and `Strsm`'s
    unreferenced triangle of A and `Strsm`'s unit diagonal (poisoned with NaN, so
    a read propagates into the answer instead of being absorbed by a tolerance);
    leading-dimension padding on all three; zero dimensions; argument panics; and
    the non-finite rules, including an infinity meeting a zero-padded panel on a
    tile that straddles the diagonal — where the copy-back is what keeps the
    resulting NaN out of both C's other triangle and its padding.
  - `Strsm`'s test matrices are diagonally dominant on purpose. A triangular
    solve's error bound grows multiplicatively down the substitution
    (`oracle.Trsm` derives the recursion), so a random triangular matrix at
    n = 500 has a legitimate bound wide enough to admit anything — the test would
    pass on a broken routine and be reporting on the test matrix instead of on
    keel.
  - `internal/block/tri_test.go` checks the mask's three range predicates
    (`whole`, `none`, `rowRange`) against the element-wise `keeps` definition over
    every rectangle in a small square. They decide per tile and per row rather than
    per element, which is what makes the mask free and also what makes an
    off-by-one in them invisible at most sizes.
- `bench.BenchmarkSsyrk` at n = 256/512/1024/2048, beside `BenchmarkSgemm` at the
  same sizes, for P4's `Ssyrk >= 85% of Sgemm` criterion. **Both now declare their
  numerator**: a `keel-bench-flops` marker naming the flop count, the formula and
  the dimensions used, from the same `work` value the harness divides by, and
  `BenchmarkOpenBLAS` takes `Sgemm`'s. `Ssyrk` fills one triangle, so its count is
  `k*n*(n+1)` and not `2*m*n*k` — counting the latter would report about twice its
  real rate and the 85% bar would be cleared by a routine running at 43%. The
  counts are *useful* flops: the half of each diagonal tile that is computed and
  discarded is the cost the bar exists to measure, so it is not counted as work.
  Rule 7's "never a number without its denominator" pointed at the numerator.
- **`scripts/gate-p4.sh` is GREEN: 65 PASS / 0 FAIL** at `dd740e5`. P4's headline
  criterion — single-thread `Ssyrk` at ≥85% of the *same host's own* `Sgemm`, both
  rates from one benchmark invocation at n = k = 2048 under the `performance`
  governor, `-count=10 -benchtime=1s`, medians net of benchstat's confidence
  interval — is met on all three gate hosts:

  | Host | Ssyrk | Sgemm | Ssyrk / Sgemm | of measured peak |
  |---|---|---|---|---|
  | vesta, Zen 4 (7950X3D) | 134.4 | 148.1 | **90.8%** | 80.9% / 89.2% of 166.1 |
  | janus, Skylake-X (i9-9960X) | 69.11 | 77.27 | **89.4%** | 31.9% / 35.6% of 216.9 |
  | antares, Zen 5 (Ryzen AI MAX+ 395) | 168.5 | 191.3 | **88.1%** | 51.4% / 58.3% of 328.1 |

  GFLOP/s. Percent of measured peak is reported and not judged, here as in P3.
  Criterion 7 was measured twice — an earlier run of the same gate, aborted later
  in criterion 8, produced 90.6 / 89.8 / 88.2 — so the three ratios agree within
  0.4 points across two independent measurements, and the ordering by host is the
  same both times. The remaining 9–12 points are the derivation's stated cost:
  tiles that straddle the diagonal are computed whole and half-discarded, and the
  discarded half is deliberately absent from the numerator.
- P4's gate carries P3's rather than restating its threshold: criterion 8 runs
  `scripts/gate-p3.sh` on the same commit (**47 PASS / 0 FAIL**, its fourth green
  full run) and refuses a dirty tree as unmeasured. The Ssyrk/Sgemm ratio is a
  ratio against a number P4's own commits can move, so the denominator's bar is
  enforced by the gate that owns it — a second copy of "≥60% of OpenBLAS" in
  `gate-p4.sh` would be a number that can drift out of agreement with itself.

- `scripts/gate-p5.sh`: P5's criteria, written before P5's code (CLAUDE.md), and red
  in **23** places on the first clean-tree run (`583ca74`: 23 FAIL / 29 PASS,
  `gate-p5: RED`) — almost all of them "P5 has not been built yet", plus four
  findings the gate produced by being run at all, each now an issue rather than a
  drive-by fix (#38, #39, #40, #42). The count was 22 on the first run, taken
  against a dirty tree, which is the *less* red number for a reason worth keeping:
  three of those 22 were "this check could not run", and running them on a clean
  tree turned one skip into four real failures — the three per-host `-race`
  verdicts and their aggregate. A gate that cannot run its own checks understates
  itself. The delegated P4 gate is **green** on this commit (64 PASS / 0 FAIL),
  so every absolute rate the ≥6× ratio stands on is still a measured one.

  It is **red in 20 places as of `51d206f`** (20 FAIL / 32 PASS, `gate-p5: RED`),
  and the three that left are the #40 ruling landing: each per-host
  `KEEL_FORCE=avx2` verdict became a PASS that *asserts* the Level-3 ceiling
  (`l1=avx2, kern=4x32/scalar`) rather than one that excused the missing rung.
  Nothing else in the set moved — an intended delta of exactly three, which is the
  evidence the narrowing was surgical. The absent `keel-p5-dispatch` marker stayed
  **one** failure rather than becoming two when the check gained a second field: no
  marker emitted is one defect with one cause, and a count that inflated with the
  number of things the missing marker *would* have been checked against would
  overstate how much is wrong.

  The 20 all have an owner and a stage: **1** lint failure carrying two findings
  (#39's unchecked `os.RemoveAll` and #38's `-0.0` literal); **8** absent
  determinism/no-state markers, two per routine across the four; **1** absent
  dispatch marker; **3** `-race` deaths from `archsimd`'s `checkptr` violation
  (#42, ruled into #22's campaign) **plus 1** aggregate saying no host produced a
  reading at all; **3** hosts reporting the scaling ratios' inputs unmeasured
  **plus 1** aggregate for the headline criterion; and **2** shipping artifacts
  absent — `doc.go`, and the README's `keel-numbers` block. Stage 2 owns nine of
  them, stage 3 six, stage 1's one-liners the lint pair, and #22's campaign the
  four race lines.
  The judgement calls are in the script's header at length; the ones that shape
  the phase:
  - The ≥6× floor is judged on `Sgemm`, `Ssyrk` **and** `Ssymm` (one parallelism
    class) and measured-not-judged on `Strsm`, whose floor is deferred to that
    measurement plus a stated model (#37). `STRSM_FLOOR` is left empty in the
    script with a comment saying it may only be filled by a ratification recorded
    in `DESIGN.md` — the deferral is mechanized, not remembered.
  - Both rates come from **one** invocation with the thread count in the benchmark
    name (`Scale/Sgemm/n=4096/threads=8`), because `-cpu=1,8` distinguishes rows
    only by the `-N` suffix that `bench_stat` and `bench_expect` strip — benchstat
    would aggregate the one- and eight-thread samples into a single row and the
    gate would divide a mixture by itself and read 1.0×.
  - The parallelism is checked rather than assumed: each row declares the
    GOMAXPROCS it set and the workers the library used, and both must equal the
    thread count in its own name. A threads=8 row that ran on one worker and a
    threads=1 row that ran on eight both produce 1.0×, and both are measurement
    failures dressed as performance failures.
  - Flop counts are re-derived by the gate for all four routines
    (`2mnk`, `kn(n+1)`, `2mnk`, `nm(m+1)`), formula string included, so the
    numerator of a scaling ratio is verified rather than asserted.
  - The README's published numbers are re-measured by the gate that ships them:
    rows keyed by **CPU model** (never hostnames — `.keel-hosts` is gitignored
    infrastructure), each carrying its denominator, each within 5% of this run; and
    any `GFLOP/s` figure outside that block fails the gate outright.
  - Bitwise determinism against the serial nest at threads 1, 3 and 8 — 3 because a
    row-partition off-by-one hides at every power of two — since splitting the MC
    loop reassociates nothing and a tolerance here would be hiding something.
  - It runs `scripts/gate-p4.sh` (which runs `gate-p3.sh`, which carries P2's
    audit) rather than restating any absolute bar. "≥6× single-thread" is a ratio
    whose denominator this phase is chartered to *improve*; a parallel nest that
    slowed the serial path would make the bar easier, and a bar that falls when the
    code gets worse is not a bar.
- `docs/toolchain-notes.md` T16 (issue #41): on arm64, whether `a*a + c` is
  FMA-fused depends on whether the compiler **constant-folds** it first, and
  `-race` defeats the folding — so one source line yields `0` in a plain build and
  `2^-24` under `-race`, on the same machine and toolchain. Both readings are
  spec-compliant. Found by `gate-p5.sh`'s race criterion, which is the first thing
  in this project ever to run `-race` on the dev host: `internal/vec`'s
  `TestSpecMulAddIsFused` computed its *unfused* witness as `a*a + c` and its own
  vacuity guard fired rather than comparing the fused answer against itself. It had
  been passing everywhere for a reason nobody had written down — on amd64 because
  gc does not contract `x*y+z` there at all, and on arm64 only because the witness
  was folded before code generation. The witness now writes `float32(a*a) + c`,
  which forbids fusion by the spec's own rule: state the rounding you require
  rather than inheriting whatever the optimizer chose.
- `docs/toolchain-notes.md` T17 (issue #42): `archsimd`'s partial slice load/store
  are not `checkptr`-safe, so **`go test -race` is a fatal error** — not a warning —
  on any keel call whose length is not a multiple of the vector width.
  `LoadFloat32x16SlicePart` and `StoreSlicePart` reach their masked operation by
  converting `&s[0]` to a full-width `*[16]float32` inside an `unsafe` helper; the
  mask keeps the *instruction* in bounds, but `checkptr` instruments the
  *conversion* and cannot know that. Reproduced standalone on linux/amd64 with no
  keel code involved, and — the part that isolates the cause —
  `-gcflags=all=-d=checkptr` alone reproduces it identically, so this is not the
  race detector and cannot be dodged with race options. It is also
  data-dependent: it fires on how much room the *allocation* has past `&s[0]`, not
  on the slice's length, so a call site can be quiet for a whole suite and abort
  after an allocator layout change. Two consequences, and the second is the one
  that matters for v0.1.0: `gate-p5.sh`'s race criterion is unmeetable on amd64
  while keel calls these, and **any user who runs `go test -race` on their own
  code crashes inside a library they did not write**. A `checkptr`-clean local
  workaround is confirmed (copy the remainder into a full-width stack array, use
  the full-width `Load…Slice`/`StoreSlice`, which convert no pointer; cost is one
  64-byte zero-and-copy on the tail iteration, never in the K-loop) but is *not*
  applied: it changes the "remainders use masked partial loads" story, so the
  disposition is #42's to settle rather than a drive-by fix.
- `scripts/gate-p5.sh`'s race verdict now classifies a `checkptr` death as its own
  outcome, naming T17 and #42 — still a **FAIL**, because naming a cause is not
  meeting a criterion. Its diagnostic for the generic case also prints the *head*
  of the failing-test detail rather than the tail: on a multi-package failure the
  `--- FAIL:` lines that identify the cause precede the per-package summaries, and
  a `tail` had been dropping exactly the lines worth reading. Its summary of the
  delegated P4 gate also counts that gate's own verdict lines rather than every
  line containing the word: a bare `grep -c FAIL` matched gate-p3's summary line
  *inside* gate-p4's log ("47 PASS / 0 FAIL"), so a green delegated gate was
  reported as "65 PASS / 1 FAIL". The verdict itself was always taken from the
  delegated gate's exit code and was correct; only the number beside it lied.
- `keel.L1Chain()` and `keel.KernChain()`: the *advertised* dispatch chains, per
  level, as functions rather than as prose. They answer a different question from
  `AvailableL1Backends`/`AvailableKernels`, which report what is runnable on the
  machine in hand and are properly shorter on a host without AVX-512. This is the
  claim keel makes about itself, so `gate-p5.sh` reads it and checks it against
  `DESIGN.md` §4/P5 and against the backends that actually have implementations —
  which is how #40 was found. A claim kept in a function can be checked; a claim
  kept in prose can only be believed.

- **P5 stage 1 opens with the four ruled one-liners, and the lint criterion is
  green** (#38, #39, #34, #10):
  - **#38 — `Isamax`'s negative-zero case was two positive zeros.** `-0.0` in Go
    source is a *positive* zero: the minus applies to the untyped constant, and
    constant arithmetic has no signed zero to produce. So `[]float32{0, -0.0}` was a
    case that could not fail, and it had been the only negative-zero coverage for
    four phases. It now constructs the value the way `internal/vec`'s spec tests
    already do (`float32(math.Copysign(0, -1))`) and covers five cases instead of
    one: both tie orders, all-negative-zero, and `-0` losing to a positive and to a
    negative nonzero — the last two being what a sign-confused comparison would fail
    outright rather than by a tie-break. **They pass on all three backends on all
    three amd64 hosts**, so the vacuous test was hiding nothing: the test was the
    defect, not `Isamax`.
  - **#39 — the spill audit's `os.RemoveAll` reports instead of discarding.** Not
    load-bearing for correctness (the gate reads `dir/<fn>.html`, never the scratch
    dir), but each leak strands the private 18MB `GOCACHE` this function creates
    inside a gitignored directory nobody looks at. A silent `_ =` would make that
    invisible, and failing the audit over it would let a `chmod` suppress an SSA dump
    that was produced correctly — so it names the leak on stderr and keeps the
    primary error.
  - **#34 — `roofline 0.0%` no longer renders a not-applicable as a measured zero.**
    `p3_denominator`'s contract is unchanged: 0 remains the right sentinel to
    *return*. Only the rendering changed, and **the condition is `roof == 0`, not
    `src == openblas` as the issue proposed**: an issue-bound host whose `min()`
    picked the reference (`why=reference`) has a real roofline that was computed and
    compared, and printing `n/a` there would hide a number instead of a hole — the
    same misreading in the other direction. Test the sentinel, not a proxy for it.
  - **#10 — P0's criterion states fusion rather than an encoding.** It named
    `VFMADD231PS`, which is the form a hand-written K-loop wants and not what
    go1.26.5 emits (`VFMADD213PS`), so as written it would have failed a toolchain
    that satisfied what P0 actually needs: one instruction doing the multiply and the
    add with a single rounding. `gate-p0.sh` has always checked the property — one
    `VFMADD{132,213,231}PS` and zero separate `VMULPS`/`VADDPS` — so this brings
    `DESIGN.md` into line with the check rather than the reverse. Which operand order
    it is stays consequential and stays tracked where the consequences are: the
    roofline section, where 231-with-broadcast versus 213 is `I = 2.875` versus
    `4.625`.

- **An instrument for the retention gap, before any theory about it** (#26 —
  the blocked `Sgemm` keeps ~90% of its own dispatched microkernel on both Zen
  hosts and ~77% on janus). `internal/block/nest_bench_test.go` +
  `scripts/retention.sh`, and three properties it was built to have:
  - **The decomposition is measured, and its residual is reported.** One blocked
    `Sgemm` splits into `nest-no-pack` + `pack-a` + `pack-b`, so
    `residual = full − the three` is what the split does not explain — printed as a
    line of the table rather than absorbed into whichever part is under
    discussion. `nest-no-pack` packs one set of panels outside the timer and reuses
    those buffers for every block: wrong values, nothing reads the result, and
    identical cost structure (same microkernel calls at the same `kk`, same
    buffers, same C traffic, same beta pass, same fringe path). What it drops is
    named where the residual is defined: the pack calls, the cache interference
    between packing and the kernel that follows, and `gemm`'s three per-call
    allocations (`bp` alone is a zeroed 3.1 MB at n=2048).
  - **Retention becomes a ratio instead of a quotient.** `gate-p3.sh` prints it
    from two invocations with two peak measurements and says so, because
    `bench_ratio_lo` cannot reach across two CSVs. `BenchmarkNest` measures the
    microkernel *in the same invocation* at the depth the nest actually calls it
    with, so retention is bounded by both confidence intervals. It is still a ratio
    of medians and the script still says so.
  - **The parts provably walk the blocks the shipped nest walks.** The parts share
    one block generator, `nestBlocks`, which is a copy of `gemm`'s three outer
    loops — exactly the kind of copy that drifts. So it is not trusted by
    inspection: `TestNestBlocksDriveTheSameGemm` drives a full pack-and-multiply
    GEMM from the generator alone and requires it to equal `Gemm` element for
    element over seven shapes (remainders in each dimension, sub-tile sizes,
    `k=1`, and both non-square orientations). A drifted bound, a missed remainder
    block or a B panel packed at the wrong `(jc, pc)` fails it. `block.go` gained
    only `plan()`, the clamp arithmetic `gemm` already did, extracted so both sides
    read the blocking from one place.
  - Shape is a sub-benchmark dimension rather than a dispatch, which answers #26's
    third candidate — does the gap track the host's *class*? — with no
    `KEEL_KERN_CLASS` pinning: both shipped shapes' retention is measured on every
    host. It also avoids a second copy of `selectKern` in a package that cannot
    import the root one.
  - `scripts/retention.sh` is **not a gate**: it certifies nothing, moves no
    criterion, and exits 0 whatever it finds. `decompose` runs the standard gate
    methodology because its numbers are meant to be quoted; `sweep` (KC/MC/NC over
    a coarse grid at 2048³) runs at `-count=5` and is labelled EXPLORATORY in its
    own output — a point it nominates has to be re-measured under the full
    methodology before it could become a default. `NC` stops at 2048 because
    `plan()` clamps it to `n`, so every larger value is the same measurement under
    another name.
- **`BenchmarkPackDirections`, and the correction it exists to make measurable**
  (#21). Both #21 and #26 stated which pack direction transposes, and both stated
  it backwards: `APanels` passes `!trans` as `depthContig` and `BPanels` passes
  `trans`, so at `NN` — the shape every benchmark in this repo runs — it is **A**
  that takes the transposing branch and B that gets `memmove`, and the case where
  both directions transpose at once is `NT`. `internal/pack`'s own package doc had
  the rule right; the issue text misapplied it. The fix is a benchmark dimension
  rather than a prose edit: both directions, all four flag combinations, at the
  block shapes the nest uses, counting valid elements only (padding is zero-fill,
  not data movement).
  - It refutes #21's premise. The `copy`-based branch — the one the rule calls
    "already vectorized" — is **2.8× slower** than the transposing branch it
    replaces on the A side at `TN` (0.77 vs 2.19 Gelem/s, dev host, pure-Go
    `internal/pack`), because its run length is `blk = MR ∈ {2,4}`: 8 or 16 bytes
    per `copy` call, ~27,600 calls per pass. Cost scales as 1/`blk` and flattens
    once the run reaches 64 B, and removing the source stride entirely changes
    nothing (1.0–1.1×), which refutes the locality explanation that was tried
    first. So the rule "copy the contiguous axis" is right for B (`NR = 32`) and
    wrong for A, and the 16×16 `Permute` transpose is no longer the first thing to
    try there.
- **`internal/pack` gets tests of its own** (#45), before #21/#22/#36 change its
  loops. Four invariants that the root package's differential tests cannot see,
  because they are visible at this boundary and not in `Sgemm`'s output:
  - The packed **layout is checked against the doc's formula**, written out as index
    arithmetic on the source rather than by calling anything in the package — so a
    layout change has to be made twice, by someone who means it. A layout that
    changed consistently with the microkernels would otherwise pass every existing
    test while breaking the contract future kernels are written against.
  - "**The zeros are written, not assumed**" becomes an assertion. A poisoned buffer
    catches an unwritten slot; the case that actually happens in `gemm` is a slot
    holding a *previous pack's* plausible value, so a large block is packed and then
    a smaller one into the same buffer, with every slot the second claims required to
    hold what the second put there. Padding is compared bitwise, since a stale −0
    passes `== 0`.
  - **The two branches are required to be bit-identical**, which is the guard that
    makes #21's branch-selection change safe: the same logical matrix packed from a
    row-major source and from its transpose, over ±0, ±Inf, NaN, `MaxFloat32` and
    the smallest denormal. The one input where they provably differ — a *signalling*
    NaN, which `copy` moves untouched and `alpha*v` quiets — is documented as a known
    asymmetry rather than tested as a requirement, since no IEEE operation produces
    one and BLAS specifies nothing about NaN payloads.
  - `ALen`/`BLen` are **exactly enough**, and one float short panics *with both
    lengths named*. Asserting the message and not merely the panic is the point:
    deleting the guard outright still panics, from the panel re-slice, so a test that
    accepted any panic would pass over code with no guard.

  Nine mutations were applied to `pack.go` to check the suite can fail — dropped
  zero-fill in each branch, a transposed layout index, an ignored `alpha`, an
  off-by-one `valid`, an `ALen` that forgets the ragged panel, a deleted length
  guard, and `nb` taken from the buffer length instead of the shape. Eight are
  caught, each by the test that should catch it. The ninth (dropping `count == 0`
  from the early return) is an equivalent mutant, not a bug, and the test comment
  says so rather than the suite being tightened around a distinction that does not
  exist.

- `docs/toolchain-notes.md` T18: a **loop-invariant vector constant is
  re-materialized every iteration**. `BroadcastInt32x16(const)` written inside a loop
  stays inside it (`MOVL`/`VMOVD`/`VPBROADCASTD`, 3 instructions per iteration); the
  same constant written above the loop stays above it, and the function's total size
  barely changes — the instructions are relocated, not removed. The repro carries its
  own control (both loops, differing only in where the source puts the constant), and
  the entry is explicit that it cannot rule out the allocator rematerializing under
  the register pressure of a real four-accumulator kernel, which has to be re-audited
  in place. Answers #8: combined with T8's CSE, which shares the mask across the four
  unrolled `Abs512` calls *within* one iteration, the cost is 3 instructions per 64
  elements rather than 12 — so the answer to "does it hoist" is no, and the answer to
  "does it cost 12" is also no.

- **`scripts/l1-bench.sh`**: A/Bs the Level-1 routines at all four of
  `bench/bench_test.go`'s sizes — 1 KB, 16 KB, 256 KB and 4 MB of float32, i.e. L1-,
  L2-, L3- and memory-resident on all three hosts — between an arbitrary base ref and
  the working tree, on every configured host, under the standard methodology with
  benchstat p-values. Written for #47, whose loop-shaping change *lengthens* four of
  the ten loops while shortening six, and whose static counts therefore cannot say
  which way the routines move: at 4 MB per call the loop body is not the limit. Not a
  gate — it certifies nothing, moves no criterion, and exits 0 whatever it measures.
  The base build comes from a detached `git worktree`, not from stashing: a stash
  would mutate a tree another long-running measurement may be reading, and it would
  make the two arms differ by whatever else happened to be dirty.

### Fixed
- **The feed decomposition's noise floor printed as `0%`, which is not a number**
  (T21). benchstat rounds its confidence interval to a whole percent, so `0%` means
  only "under 0.5%" — and `scripts/retention.sh feed` was printing that percent as
  the column a reader uses to tell a resolved step from noise. On vesta it read `0%`
  on seven of eight rows, which says every step is resolved, including the ±0.60 ns
  ones; read correctly it bounds the floor at 0.5% of the arm, up to 4.4 ns on the
  4×32 kc=512 row — larger than three of the four panel-feed steps that row reports.
  The column is now `worst-ci` *and* a `floor` in nanoseconds, computed as
  `(p+0.5)% × the arm it belongs to` so the bound errs toward calling a step
  unresolved, steps below their row's floor are marked `*`, and each term says how
  many of its points are unresolved or negative before the reader reaches its spread.
  A negative cost is now named as an arm defect rather than reported as a cost. No
  gate verdict is affected: gates compare a median *net of* its CI against a floor,
  so a CI rounded down to zero can only make a passing threshold harder to reach —
  but for a *difference between two arms* the rounding is not conservative, and the
  whole feed instrument is differences.
- **`benchstat` was silently declining to compare the two arms of every A/B run**
  (#50, T20). It groups results into one table per distinct *configuration*, where a
  configuration is every `key: value` line in the log — and keel's provenance
  preamble, which exists so that no number ships without its denominator, is in that
  namespace. One of its markers, `keel-bench-clock-mhz`, is a live snapshot of the
  CPU's frequency range and so differs between any two runs on one host by
  construction. Two files that differ in one config key are printed as two
  independent one-column tables: no delta, no percentage, no p-value, exit status 0.
  The first run of `scripts/l1-bench.sh` produced three hosts × two builds × 20
  benchmarks of correct medians and not one comparison among them. `bench_compare`
  in `scripts/bench.sh` now ignores the keys that describe the run rather than the
  build (`$KEEL_BENCH_IGNORE`) **and then checks that a `vs base` column actually
  appeared**, printing `NOT COMPARED` plus the offending keys when it did not. No
  gate verdict was affected: gates aggregate a single log, where a forked table
  cannot lose a comparison. `scripts/l1-bench.sh` now goes through it, and its
  claim that "the deltas carry p-values" — printed above two tables that contained
  no deltas — is gone.
- **`scripts/retention.sh sweep` ran at `-count=10` while its header printed the
  `-count=5` it documents** (#49). `scripts/bench.sh` is sourced first and defaults
  `KEEL_BENCH_COUNT` to 10, so the sweep's own `${KEEL_BENCH_COUNT:-5}` could see
  neither the caller's setting nor its own default. The caller's value is now
  captured *before* sourcing, and the header prints two separate things: the count
  that was requested, and — per host, counted out of the log itself — the number of
  sample rows that actually arrived. A parameter read back out of the measurement
  cannot be shadowed by whatever set it. (The affected sweep is unharmed: 10 is the
  stronger discipline, so the error was in the safe direction, and its numbers stand
  as the exploratory numbers they were labelled.)
- **The sweep's `<- shipped` marker could never fire**, found while fixing #49: it
  matched the shipped triple's literal name, whose `nc=4096` is larger than any NC on
  the grid, so no row was ever marked — which reads as "the shipped point is not on
  the grid". It now marks the shipped KC/MC at the grid's largest NC, read back from
  the CSV, and the label says exactly that rather than implying more.
- **`scripts/retention.sh`, `scripts/l1-bench.sh`, `scripts/roofline-test.sh`,
  `scripts/provision-openblas.sh` and `scripts/bootstrap-github.sh` now define
  everything in functions and end with `main "$@"`** (#51, the convention; the six
  gate scripts are tracked there and go last, each needing a green run of its own).
  Bash
  reads a script incrementally as it executes it, so editing one mid-run can corrupt
  the parse position of the running copy — a hazard that had become a rule to
  remember ("never edit a running instrument"). Forcing a whole-file parse before any
  work begins makes the instrument immune instead. `scripts/roofline.sh` was on the
  list and is off it: it is three function definitions and no top-level work, so it
  is already immune as a sourced library. One behaviour change to declare rather than
  slip in: `provision-openblas.sh --help` prints a fixed line range of its own header,
  so that range was narrowed to end before the new wrapper comment.
- **`spill-audit` could not see a bounds check whose panic block was aligned**, and
  `gate-p2.sh` turns that count into a passing criterion — so the instrument
  certifying "0 surviving bounds checks in the steady-state K-loop" had a false-clean
  mode (#46). `Parse` drops `NOP` lines as zero-length pseudo-instructions, which is
  right for the T9 inlining marker but wrong for *alignment padding*: that owns its
  own offset and is several bytes wide. `reachesPanic` matched the branch target
  exactly, so when the compiler aligned an out-of-line panic block the branch pointed
  at padding, no instruction was found there, and the exit went uncounted. Targets now
  resolve to the first instruction at or after the offset, which finds the same
  instruction wherever an exact match existed — the change can only find *more* exits,
  never fewer. Regression test added to the hand-written listing
  (`TestAuditSeesAPanicBehindAlignmentPadding`), verified to fail against the old
  resolver.

  **The P2 criterion holds.** Re-audited with the fixed detector, `Kernel2x32` and
  `Kernel4x32` still report 0 bounds-check exits and 0 calls, and the three peak
  kernels are still register-only. P2's green was correct — but for a period it was
  correct without being verifiable, and those are not the same thing. Two published
  counts *are* revised, both in `internal/l1` (#47): `avx512Scal` and `avx2Scal` were
  reported clean and carry one each.

- **All ten `internal/l1` vector loops now compile with zero surviving bounds
  checks** (#47). They were written in P1, before P2 wrote the "pre-sliced panels"
  rule for kernels, and they were index-driven: a `for i := 0; i+64 <= len(x); i += 64`
  guard with `x[i+16 : i+32]` sub-slices. `prove` does not discharge those. From
  `i+64 <= len(x)` and `len(x) <= cap(x)` it will not take the step to
  `i+64 <= cap(x)`, and `i <= i+16` needs no-overflow reasoning it also does not do,
  so an unrolled body paid one check per offset sub-slice — `avx512Dot` ran 69
  instructions to issue four FMAs. Every loop is now driven by `len(x)` with
  *constant* offsets (`x[16:32]`, never `x[i+16:i+32]`) and re-slices at the bottom,
  which is the idiom `internal/vec`'s microkernels already use for their panels.
  The two-slice routines (`Dot`, `Axpy`) re-slice `y` to `len(x)` once above the
  loop *and* carry `len(y) >= step` in the guard: the re-slice alone is not enough,
  because the prover loses `len(y) == len(x)` across `y = y[step:]`, which left
  seven of `y`'s checks standing after `x`'s had all gone. `check_bce` on
  `l1_amd64.go` goes from 60 reports to 4 — one `IsSliceInBounds` per two-slice
  routine, which is the `y = y[:len(x)]` precondition itself, hoisted out of the
  loop and paid once per call instead of per iteration. (`l1.go`'s 21 are the
  scalar reference path and are untouched.) `spill-audit` reports 0
  bounds-check exits for all ten vector kernels, where it previously reported
  them for all ten.

  **The instruction counts do not all improve, and T19 is why.** Excluding T9's
  1-byte inlining NOPs: `avx512Asum` 41→20, `avx512Dot` 61→32, `avx512SumSq`
  42→21, `avx2Asum` 45→24, `avx2Dot` 61→32, `avx2SumSq` 42→21 — the six unrolled
  reductions roughly halve. But `avx512Axpy` 16→21, `avx2Axpy` 17→22,
  `avx512Scal` 11→12, `avx2Scal` 12→13. A slice advance guarded by `>=` costs
  seven instructions, not two, because the loop may leave the slice exactly empty
  and a slice's data pointer must not pass the end of its allocation, so the
  pointer bump is made conditional (`NEGQ`/`SARQ $63`/`ANDL`) — see
  docs/toolchain-notes.md T19, which has the three-function repro. An unrolled body
  amortizes that over four vector ops; a non-unrolled one cannot, and `Axpy`
  advances two slices. The `>` form collapses the advance to a single `ADDQ`, and
  is applicable here because the tail already absorbs a full vector through
  `LoadPart`/`StorePart` — but it moves the last full vector onto the masked path,
  so it is a second change wanting its own measurement rather than a free win.
  Runtime numbers for all five routines at L1-, L2-, L3- and memory-resident sizes
  are #47's remaining deliverable; the reassociation order is unchanged, so the
  results are bit-identical, not merely within tolerance.

### Changed
- **`internal/pack`'s contiguous branch no longer calls `memmove` for short runs**
  (#21). `copy()` on a slice of statically-unknown length is a `runtime.memmove`
  call — confirmed in the object code (`pack.go:169 CALL runtime.memmove`) — which is
  the right instrument for B, whose blocked axis is `NR = 32` (128 bytes per run),
  and the wrong one for A, whose blocked axis is `MR ∈ {2,4}`: one call per 8 or 16
  bytes, about 27,600 of them per pass. Below a `memmoveFloor` of 16 elements the
  branch now uses a plain assignment loop. The loop assigns rather than multiplying
  by an alpha known to be 1, so the *signalling*-NaN asymmetry `TestBranchesAgree`
  documents stays exactly where it was documented instead of moving.

  Dev host (Apple M4 Pro, `GOMAXPROCS` unset, `-benchtime=200ms -count=8`,
  `go tool benchstat`; a data-movement rate, so the denominator is each cell's own
  baseline and no percent-of-peak is claimed):

  | case | before | after | |
  |---|---|---|---|
  | `2x32/TN/pack-a` (blk=2) | 787.2m ± 2% | 1119.0m ± 6% Gelem/s | **+42.1%** (p=0.000) |
  | `4x32/TN/pack-a` (blk=4) | 1.529 ± 5% | 1.569 ± 3% Gelem/s | +2.6% (p=0.000) |
  | `2x32/NN/pack-a` (transposing) | 2.143 ± 13% | 2.067 ± 8% Gelem/s | ~ (p=0.185) |
  | `4x32/NN/pack-a` (transposing) | 2.365 ± 9% | 2.189 ± 4% Gelem/s | −7.5% (p=0.028) |

  Two caveats stated rather than smoothed. **The 4x32/NN −7.5% is in a branch this
  change does not touch**, so it is either noise (that cell's baseline varies ±9–12%
  and its 2x32 twin shows no change) or an instruction-layout effect from the
  `switch`; it needs the amd64 hosts to resolve and they were measuring #26 at the
  time. And **this does not close #21**: the contiguous branch at `blk=2` is still
  1.85× slower than the transposing branch, down from 2.72×, so the remaining gap is
  the per-k-step overhead of a two-element inner loop, not `memmove`.

  Worth being explicit about the blast radius: `APanels` passes `!trans`, so at `NN`
  — the shape every gate benchmark runs — A takes the *transposing* branch and this
  changes nothing. The gain is on `TN`/`TT`, i.e. for callers who pass `transA`.
- **P5's internal order is now stated: single-thread remediation, then the parallel
  loop nest, then the scaling gate** (`DESIGN.md` §4/P5, ruled 2026-08-12). #26
  (retention), #36 (Ssymm's dense expansion), #37 (Strsm's scalar diagonal solves)
  and the deferred measurements on #21/#22 all sequence *before* the
  parallelization rather than beside it: each is a single-thread cost that
  parallelization multiplies rather than hides, so parallelizing first would
  certify scaling curves for routines the same phase intends to change — every
  number re-measured and a record carrying two regimes. The certifying measurement
  comes last, over the final artifact, which is the same rule that made P3 keep the
  hardened re-run instead of the green it inherited. #21 and #22 join the campaign
  because they are the same code as #36/#37, and entering it twice is how a
  measurement ends up compared against a different build than it was taken on.
- **P5's scaling floor binds by parallelism class rather than by routine list**
  (`DESIGN.md` §4/P5, ruled 2026-08-12; the question was whether P4's derived
  routines are judged or only `Sgemm`).
  - `Sgemm`, `Ssyrk` and `Ssymm` are **one class — GEMM-shaped nests over
    independent tiles, no cross-iteration dependence — and the ≥6× floor binds all
    three.** Judging only `Sgemm` would let a serialization bug in the triangular
    C-update masking hide behind "the derived routines are reported, not judged",
    which is both the likeliest place to introduce a dependence and the measurement
    that would catch it.
  - `Strsm` is a **different class, and its floor is deferred to a measurement.**
    Its diagonal solves impose a dependency chain the other three lack, and its
    available parallelism varies with `side` and shape. P5 measures its scaling,
    reports it beside the judged three, and states the parallelism model behind the
    number — the rank-update/diagonal-solve split at the gate's shape; that pair
    sets the floor, which binds from the commit recording it forward. Recorded on
    issue #37 so the deferral is a named debt and not a standing exemption. Writing
    6× on `Strsm` today would be a threshold without a model, which is the move
    this project has now refused six times.
- **`Sgemm` selects its microkernel shape per host instead of taking the registry's
  first entry** (ruling on issue #24; `KERNEL.md` §8, `DESIGN.md` §4/P3). Both
  shipped shapes are zero-spill and neither dominates — 4×32 wins on Zen 4 and
  Zen 5, 2×32 wins on Skylake-X by 11 percentage points — so shipping one shape
  everywhere was a measured performance bug on one of three gate hosts. Dispatch
  now classifies the host with the same issue-bound/FMA-bound classification the
  gate's throughput model already defines and takes the shape extremal on that
  class's binding cost: fewest memory ops per FMA when arithmetic binds, fewest
  instructions per FMA when the front end does. `KEEL_KERN_CLASS=fma|issue`
  overrides the classification, and an unrecognized value panics rather than
  falling back, for the same reason `KEEL_FORCE` does.
  - The classification is a feature-bundle fingerprint, because no
    microarchitecture is readable from pure Go on this toolchain (T14, #25). It is
    printed with its grounds and checked against the gate's own measured verdict on
    every host on every run; disagreement is a gate failure.
  - `gate-p3.sh` criterion 5b: the throughput sentinel now judges the *dispatched*
    shape rather than whichever shipped shape measures fastest, fails if a
    passed-over shape beats it net of CI in the same invocation, re-measures the
    blocked `Sgemm` at 2048³ under the other class as a cross-check, and re-derives
    every shipped shape's recorded `InsnsPerFMA` from the object code so the
    ranking cannot come to rest on a stale count.
  - Found by the gate, not by a benchmark: P2's anti-vacuity shape guard refused
    the dispatched 4×32 a roofline on janus, which is what surfaced the bug.
  - The gate model now states a **one-retry allowance for throughput sentinel readings**
    (`DESIGN.md` §4): a failing sentinel triggers exactly one re-run, fails only if both
    runs fail, and both outputs are archived either way, so a pass-on-retry is visible
    in the record. It never applies to a correctness criterion — those fail on first
    miss, since a differential test that passes on retry has found a nondeterminism.
    The gate script is unchanged and still fails on first miss; the allowance is the
    operator's and the archive is what keeps it bounded. How often the retry is needed
    is itself the signal: a sentinel that needs it often is reporting that the margin
    is gone, not that it is noisy, and the answer is throughput or a re-derived bar
    rather than more retries.
  - `gate-p3.sh` also prints **retention** per host — the share of its own microkernel
    the blocked loop nest keeps — as reported-never-judged provenance beside
    percent-of-peak. vesta ~90%, antares ~92%, janus ~77%, so P5 inherits #26 as a
    re-runnable measurement instead of a remembered figure (`DESIGN.md` §4/P5 names it
    as a carried-in input). It is a ratio of two point estimates from two invocations
    and is labelled as such, with both inputs printed; nothing compares it to a
    threshold, because closing that gap means sweeping `KC`/`MC`/`NC`, which is P5 work.
  - `docs/spill-report.md` carries a superseded-by note: its P2-era prediction that
    "P3 will dispatch this kernel on all three machines" is the premise this change
    removes. The report's measurements are left as recorded — it documents P2, not
    what ships now.
- **DESIGN.md's 32×6 microkernel tile is not implementable on go1.26.5, and P2
  ships two smaller shapes instead** (`docs/toolchain-notes.md` T10, issue #18,
  rationale in `KERNEL.md`). Two independent properties of the toolchain, both
  read out of the compiler's own source and confirmed by probe: the register
  allocator offers SIMD values only X0–X14 (`fpRegMaskAMD64` is `0x7FFF0000`; X15
  is the ABI zero register and X16–X31 appear in no allocatable mask at all, even
  though `VFMADD213PS512`'s register shape lists them as legal), and only the
  `213` FMA form exists — it writes to its first multiplicand, so `acc += a·b` can
  never land in `acc` and always needs a live scratch register. DESIGN.md's budget
  of 12 accumulators + 2 B vectors + 1 broadcast = 15 is therefore one register
  short of allocatable, and measured across a 115-shape sweep *every*
  12-accumulator configuration spills. The zero-spill frontier is 8 accumulators;
  0.75 loads per FMA is a hard floor below it, since a lower ratio needs 9. The
  DESIGN.md tile is kept as `kern.ReferenceTile` — audited, differential-tested and
  benchmarked, deliberately absent from `kern.Kernels()` — so the cost of the
  constraint is a measured GFLOP/s number rather than an assertion, and the gate's
  zero-spill criterion stays binding on everything that ships.
- **The microkernel tile is reflected relative to DESIGN.md: MR rows × NR columns,
  vectors along N** (issue #16). keel's public API is row-major (DESIGN.md §3), so
  in an M-vectorized tile sixteen consecutive elements of a column are `ldc` apart
  and every accumulator lane lands on a different cache line — 192 scalar stores
  per tile, each needing a lane extract that archsimd only offers as a
  store-and-reload, i.e. a spill. The M-vectorized tile would fail P2's own audit
  for a reason unrelated to the compiler being audited. The arithmetic and the
  register pressure the phase was designed to test are unchanged.
- **The K-loop bodies live in `internal/vec`, not `internal/kern`.** Reaching
  archsimd through a shim costs one 1-byte `XCHGL` statement anchor per inlined
  wrapper *with a Go body* per call site (`docs/toolchain-notes.md` T9). Measured on
  the 2×32 body: 90 instructions with 24 anchor NOPs through two wrapper levels
  against 74 with 8 when `internal/vec` names archsimd directly — 27% of the loop
  body was anchors. `internal/kern` is now the shape registry, tile protocol and
  scalar reference; the "all simd imports in `internal/vec`" rule is unchanged.
- `scripts/gate-p2.sh`: real P2 checks. The throughput floor applies to the best
  *shipped* shape per host — P3 dispatches to one of the two, so failing a host for carrying
  a second kernel it would never select would measure the wrong thing — with every
  shape's number printed either way, numerator and denominator taken in the same
  benchmark invocation, and the audit of the deliberately-spilling reference tile
  run as explicitly non-fatal evidence.
- **`scripts/gate-p3.sh` is GREEN: 47 PASS / 0 FAIL**, on three consecutive full
  runs — at `6c0f722` (the closing verdict for P3) and again at `0271dd7`, which
  changed what criterion 6 executes on two of three hosts and so had to be
  re-measured rather than assumed: vesta 91.0%, janus 73.8%, antares 65.7%. P3's
  headline criterion — single-thread `Sgemm` at 2048³ against ≥60% of the same
  host's own OpenBLAS — is met on every gate host, each against a reference chosen
  by measurement and verified to have taken. All three fully-measured runs, for
  reproducibility (every number is from one invocation per host under the
  `performance` governor, `-count=10 -benchtime=1s`, medians net of benchstat's
  confidence interval):

  | Host | reference family | keel / denominator | run 1 | run 2 | run 3 |
  |---|---|---|---|---|---|
  | vesta, Zen 4 (7950X3D) | Haswell, +5.5 to +6.5% over `DYNAMIC_ARCH`'s Cooperlake | openblas, FMA-bound | 91.8% | 91.1% | 91.0% |
  | janus, Skylake-X (i9-9960X) | SkylakeX, unpinned | roofline, issue-capped | 73.5% (plain OpenBLAS 40.3%) | 73.6% (40.4%) | 73.8% (40.4%) |
  | antares, Zen 5 (Ryzen AI MAX+ 395) | Cooperlake, unpinned | openblas, FMA-bound | 65.3% | 64.5% | 65.7% |

  Reported, never judged: percent of measured peak is 88.1% (vesta), 46.1%
  (janus), 58.5% (antares), and retention — how much of its own microkernel the
  blocked loop nest keeps — is 91%, 77% and 91%, which is issue #26's P5 baseline
  as a measurement rather than a recollection. janus's sentinel, P2's floor re-run
  on the dispatched shape and the tightest margin in the gate, holds at 46.1% of
  peak = **94.8%** of its 48.6% issue roofline without using the one re-run the
  sentinel retry policy allows.
- `scripts/gate-p3.sh`: real P3 checks, written before any P3 code. Three things
  in it are decisions rather than transcriptions of DESIGN.md §4/P3, and are
  stated in the script so they can be argued with:
  - **The sweep's extent is enforced, not trusted.** A green `go test` proves
    only that whatever ran, passed, so the tests print coverage markers and the
    gate parses them: every size in DESIGN.md's list, the complete transpose
    lattice, and alpha and beta each covering 0, 1 *and* a general value — 0 and
    1 are the special-cased paths, so a lattice of only those would exercise
    every shortcut and never the general multiply. The enumerated sets must
    multiply out to the reported combination count, so a marker cannot claim
    combinations it did not run.
  - **The oracle's cost is a declared property of each size.** A float64 oracle
    at 2048³ is 8.6 GFLOP per combination. Sizes up to 65 must be verified in
    full; 500, 1000 and 2048 may be verified by a seeded sample of exactly
    computed entries, but only by saying so per size, with a floor of 256 entries
    and a printed seed. "We sampled" is the kind of concession that starts at
    2048 and ends up applying to 17.
  - **The OpenBLAS bar runs where both halves can execute.** Read as "measure on
    the dev machine", the ≥60% criterion is vacuous here: this dev host is
    `darwin/arm64`, so the ratio would compare OpenBLAS-on-arm64 against keel's
    scalar fallback. The gate instead keeps the *comparison* dev-only — behind the
    `openblas` build tag, out of the module's dependency graph — and runs it on the
    amd64 hosts in one invocation each, from a native build of `git archive HEAD`,
    with single-thread verified from the harness's own report on both sides. A host
    with no reference FAILS and gets the exact provisioning commands for its own
    distribution; percent-of-peak is not accepted as a substitute.
  - **The reference is same-host, on every gate host** (ruling on issue #23).
    There is no reference-host list and no golden machine: the only
    apples-to-apples ratio is same silicon, same thread count, same run, so each
    host is divided by its own OpenBLAS and the version and selected target are
    recorded beside every ratio. The `DYNAMIC_ARCH`-chosen kernel family
    (`openblas_get_corename()`) is *checked*, not just printed, against an
    AVX2-or-better allowlist, because that is the one part of the reference's
    configuration whose failure mode is in keel's favour — a generic kernel on an
    AVX-512 host reads low and inflates the ratio while the version, the thread
    count and the config string all still look right. An unrecognized name fails
    too. On an **issue-bound** host the denominator becomes
    `min(same-host OpenBLAS, roofline × measured peak)` (same ruling, citing
    #17/#18): OpenBLAS there is hand assembly folding accumulation and an embedded
    broadcast into single FMAs, which the intrinsic layer provably cannot emit
    (T12), so 60% of it is a demand on the decode stage rather than on the kernel.
    The decision is the pure function `p3_denominator`, unit-tested by nine new
    fixtures in `scripts/roofline-test.sh`: it can only ever *lower* a denominator,
    it applies only where P2's classifier independently says issue-bound, it
    carries P2's anti-vacuity shape guard against the shape `Sgemm` actually ran
    (which as this ships refuses the dispatched 4×32 — issue #24), and it retires
    itself with no expiry clause as the lowering improves. vesta and antares
    classify FMA-bound and face the unmodified criterion. Both ratios are printed
    on every host: §7 rule 7 applies to the gate's own arithmetic too.
  - **The reference is the fastest *measured* OpenBLAS on that host, chosen by a
    coretype sweep** (ruling on issue #31), not whatever `DYNAMIC_ARCH` selected.
    The gate forces `OPENBLAS_CORETYPE` through {default, Zen, Haswell, SkylakeX,
    Cooperlake, SapphireRapids}, records every candidate's achieved corename and
    GFLOP/s, pins the winner for the run that produces the ratio, and *verifies*
    the pin took by comparing the library's own `corename` in the measured run
    against the sweep's. The allowlist is now the floor ("modern enough"), the
    sweep the ceiling ("best this silicon can do"). `DYNAMIC_ARCH` dispatches on an
    ISA feature bit, so on vesta's Zen 4 it ships the full-width Cooperlake kernel
    onto a double-pumped 256-bit datapath: 149.5 GFLOP/s where the AVX2 Haswell
    kernel measures 159.5, a 6.7% understated denominator — keel's own issue #24
    with the vendors reversed, which the allowlist structurally cannot catch
    because it contains both the right and the wrong answer for every gate host.
    Selection is best-of-N; the number that enters the record is still measured
    under the full §5.4 methodology with the winner pinned. antares and janus
    default correctly (Cooperlake 297.2, SkylakeX 193.5) and the sweep confirms it
    by measurement rather than by assumption.
  - **The performance governor is asserted on every host in a preamble, not
    assumed** (same ruling). Anything but `performance` fails that host before a
    benchmark runs, and an unreadable governor fails too: an unchecked
    precondition is not a met one. It is then re-read at measurement time, so a
    governor that changed after the preamble fails that host rather than passing
    on a stale check. This replaces a tally that any single host could satisfy —
    "at least one host cleared the bar under the performance governor" — which let
    antares contribute numbers from `powersave`, where the first reading of its
    sweep was 245.0 GFLOP/s against a 296–297 steady state: an 18% error in a
    denominator, decided by how recently the core had been busy.
  - **`SGEMM_BENCH_FILTER` never ran the OpenBLAS benchmark** (issue #32,
    `docs/toolchain-notes.md` T15), found by the first gate run that could reach
    criterion 6. `go test -bench` splits a pattern on top-level `|` *first*, into
    an alternation of whole patterns, and only then splits each alternative on
    `/`. So `Peak|Sgemm|OpenBLAS/avx512|n=2048` was four alternatives — `{Peak}`,
    `{Sgemm}`, `{OpenBLAS,avx512}`, `{n=2048}` — and not the two-level filter the
    comment above it claimed. `{OpenBLAS,avx512}` cannot match anything, because
    `BenchmarkOpenBLAS`'s children are `n=…`, so the reference's benchmark never
    ran and criterion 6 had no denominator; `{Peak}` and `{Sgemm}` were
    depth-unconstrained, so every gate run also paid for three `Sgemm` sizes and
    two `Peak` variants it never reads. Now
    `(Peak|Sgemm|OpenBLAS)/(avx512|n=2048)`, where the parentheses are
    load-bearing: they suppress both splits and make the `|`s ordinary regexp
    alternation inside one two-element pattern. The audit of every other filter in
    the repo found no second instance. The failure message is also split in two,
    because "no result row to divide by" and "no reference on this host" had the
    same wording and only the first is a defect in the gate.
  - **The coretype sweep read the theoretical-peak provenance line as a benchmark
    rate** (issue #33), in the same run. Taking the maximum over every field
    followed by `GFLOP/s` on every line picked up
    `keel-bench-peak-formula: avx512: 368.9 GFLOP/s`, which is larger than any
    real rate and identical across candidates: all six tied on every host, the
    winner defaulted to whichever came first, and the sweep reported `+0.0%`
    against `DYNAMIC_ARCH`'s own choice — the 6.7% finding it exists to enforce,
    erased by its own parser. It now reads a rate only from a result row whose
    benchmark name is exactly the one requested, and a run that produces no such
    row fails the host as a gate defect instead of degrading to a number. With the
    fix, the gate's own sweep reproduces the finding: vesta `default` → 149.4,
    `Haswell` → **159.5**.
  - **The coretype pin is an intervention justified by a measured effect: no
    effect, no intervention** (ruling on issue #35). Pinning the fastest candidate
    crowns noise on a host where no distinct family is actually faster. Across two
    full gate runs, janus's and antares's winning *request* moved while the achieved
    *family* never did, with margins of 0.0–1.0% against a same-family drift the
    sweep measures at about 0.5% — a winner by a margin inside drift is a winner by
    dice, which is #33's lesson relocated from the parser to the selection layer.
    The sweep now asks whether a family the library did *not* already choose beats
    the one it did by more than this sweep's own noise floor, measured as the largest
    spread between candidates that landed on the same achieved corename. If it does,
    the winner is pinned and the margin is reported against the drift it had to
    clear; if it does not, the reference runs **unpinned**, the way the library runs
    itself, and "no cross-family winner beyond drift" is the recorded finding. On
    both hosts where this changes the outcome the decision is unanimous rather than
    marginal: the best cross-family candidate is slower than the default outright
    (janus Cooperlake 187.70 against SkylakeX 194.70; antares SkylakeX 298.10 against
    Cooperlake 298.60), so the old winner was an alias of the default's own family
    drawing a high sample. vesta's genuine cross-family win survives on both runs
    and is still pinned: Haswell at +5.5% and +6.5% over `DYNAMIC_ARCH`'s Cooperlake,
    against drifts of 1.20 and 0.40 GFLOP/s. The pin verification still runs in the
    unpinned case, where it compares two unpinned invocations and so catches a
    library whose unaided choice is not stable across runs.
  - **Absence is a first-class outcome in every criterion that reads a benchmark**
    (DESIGN.md §5 rule 6, new). `bench_expect` in `scripts/bench.sh` takes the
    benchmarks a criterion declares it will read and asserts a minimum row count
    for each of them — `-count` rows, not benchstat's minimum of 6 — before the
    criterion reads any of them. `bench_stat` printing nothing was previously
    indistinguishable at every call site from a benchmark that ran and reported
    nothing under the unit asked for, and each caller invented its own reading of
    empty; that indistinguishability is where #32 lived for the whole of P3. The
    three states are reported separately rather than collapsed into "missing",
    because a filter that did not select the benchmark, a run that died partway,
    and a benchmark reporting a metric the gate does not read have different
    causes. Three call sites in `gate-p3.sh`: criterion 5, criterion 6 — and the
    kernel sentinel, which is the one that would have survived audits. With
    `Peak/avx512` absent it reported *"no bounded percent-of-peak for any shipped
    shape"*: a red that blames the shapes for the absence of their denominator,
    inside the criterion that carries P2's floor forward, and believable enough to
    be read as a real result. Criterion 6 now declares the peak as well, where an
    absent peak used to reach `p3_denominator` as `peak=0`, silently reverting an
    issue-bound host to plain OpenBLAS — the strict direction, so never flattering,
    but a ruled denominator replaced on the strength of a measurement never taken.
  - **A non-discriminating coretype sweep fails instead of crowning a winner by
    candidate order** (issue #33's signature, mechanized). A sweep exists to
    discriminate, so distinct kernel families measuring an identical rate means the
    instrument is broken by construction: variance too low is as diagnostic as
    variance too high. The test is over distinct *achieved* corenames, not over
    candidates, because candidates that alias to one family are supposed to agree —
    vesta answers both `Cooperlake` and `SapphireRapids` with `corename=Cooperlake`,
    and counting candidates would fire on that and punish OpenBLAS for correctly
    reporting that two names are one family. Counting families also makes the check
    indifferent to the candidate list's composition. The halves are disjoint: the
    pin verification asks whether a request took, this asks whether the instrument
    can tell the families apart. Exact equality is the threshold and the
    measurements support it — two invocations of one family on vesta read 150.60 and
    149.80, so real rates of the same silicon do not tie to two decimals.
  - **DESIGN.md §5 rule 5 amended to match the ruling already applied to §4/P3**:
    *every* measuring host under the `performance` governor, asserted per host and
    re-read at the moment of measurement. The vicarious wording ("at least one
    measuring host") had survived in the source clause while its replacement was
    enforced downstream, so the gate cited a rule that contradicted its own
    behaviour.
  It also carries P2 forward: the spill/call/bounds-check audit re-runs on every
  gate from here on, because packing and edge handling are exactly what would
  break those properties, and P2's throughput floor is re-checked on a sentinel
  host so a K-loop that P3 made fatter is noticed (issue #19; janus, where
  instruction count binds). An unconfigured sentinel means *every* host is one:
  missing configuration costs time, never coverage.
- **The percent-of-peak denominator is now measured per host, not derived**
  (DESIGN.md §4/P2, issue #11). The formula remains as a printed cross-check.
  First measurements, single core, float32: Zen 4 (7950X3D) 165.6 GFLOP/s avx512
  against 165.5 avx2 — a width ratio of 1.00×, which is the double-pumped
  256-bit datapath measured directly rather than looked up; Skylake-X (i9-9960X)
  215.9 against 101.8, ratio 2.12×; Zen 5 (Ryzen AI MAX+ 395) 327.8 against
  164.0, ratio 2.00×, settling the open question about that part's datapath
  width. The formula overstates Zen 4 by 2.23× (the double-pump) and Skylake-X by
  1.30× (the AVX-512 frequency license, visible as an implied 3.37 GHz against a
  4.4 GHz max clock); on Zen 5 it lands within 1.01×. Unlike the L1 ratios, these
  reproduce to within 0.4% between runs — a register-only kernel has no cache or
  placement to be lucky about.
- **Gate benchmarks: `-count=10 -benchtime=1s`, benchstat medians, thresholds
  cleared net of CI** (DESIGN.md §5.4 rule 5, issue #14). `-benchtime=3x` is now
  for smoke runs only.
- **P1's Sdot ratios are re-derived under that methodology and supersede the
  first ones**, which came from `-benchtime=3x -count=5` reduced by
  min-of-samples. Net of CI, over three gate runs: 8.71×/7.18×/8.57× on Zen 4,
  7.55×/7.48×/7.53× on Skylake-X, 9.14×/8.96×/8.79× on Zen 5. The old numbers —
  4.28×, 5.91×, 4.09× — were not merely noisy but biased low by roughly 2×: three
  iterations of a 4096-element kernel measure cold caches and frequency ramp,
  both of which cost the vector path proportionally more. The remaining
  run-to-run drift on Zen 4 — bimodal, two runs within 1.6% and one 17% below
  them, invisible to a within-run confidence interval — is recorded in
  `docs/hosts.md` as an open question rather than averaged away: core placement
  across that part's two CCDs is the leading hypothesis (bimodality fits it and
  not thermal drift), and pinning changes what the measurement means.
- `docs/toolchain-notes.md` T9's flag list was wrong and is corrected: it cited
  `-gcflags=-N=0`, which is the *default* (`-N` is a boolean), and therefore claimed
  nothing. `-N` does remove the anchor NOPs, and is not a workaround — with
  optimizations off each statement's values go to the stack, so real instructions
  carry the caller's own positions and no anchor is needed. Found while building the
  standalone repro for the upstream filing, which is the point of building one.
- `oracle.Tolerance` gained an underflow floor term: `C·f(n)·(eps32·scale +
  eta32/2)`. Rounding error is only relatively bounded above the smallest
  representable magnitude; a float32 dot product of ~1e-25 elements has products
  below the smallest subnormal, returns exactly 0, and is *correctly rounded*
  while the float64 oracle says 6e-49. The relative-only model called that a
  failure. The new term is ~1e-44 for realistic n, thirty orders of magnitude
  below anything the relative term admits, so it cannot mask a real error.
- `scripts/remote.sh`: arguments are `printf %q`-quoted before crossing ssh.
  ssh concatenates its command words and hands the result to a remote shell, so
  `-test.bench='A|B'` was being parsed as a pipeline on the far side. It failed
  loudly here; the same expansion on a glob or a `$` would have quietly changed
  what got measured.
- `vec.ScalarMax`/`ScalarMin`: the operand-order claim for NaN and signed zero
  is now **verified on hardware** rather than marked UNVERIFIED. `x.Max(y)`
  returns `y` for NaN and for `max(±0, ∓0)`, as the spec already said —
  confirmed bit-exactly on Zen 4, Zen 5 and Skylake-X — three independent implementations
  of `VMAXPS`. Disassembly could not settle this; only execution could.
- Gate P0 is green. All 14 shim ops agree bit-exactly across scalar, AVX2 and
  AVX-512 on all three amd64 hosts, and both FMA wrappers lower to a single
  `VFMADD213PS`.
- Gate P1 is green. All six Level-1 routines match the float64 oracle on scalar,
  AVX2 and AVX-512 on all three amd64 hosts, pass again with dispatch forced to
  scalar on machines that have AVX-512, and clear the ≥4× Sdot floor at n=4096
  on every host, net of benchstat's confidence interval: 8.57× on Zen 4 (Ryzen 9
  7950X3D), 7.53× on Skylake-X (i9-9960X), 8.79× on Zen 5 (Ryzen AI MAX+ 395),
  with at least one host clearing it under the `performance` governor.
- **Gate P2 was RED on the flat 55%-of-peak floor, and P2 is a go/no-go rather
  than a hurdle, so work stopped there and the decision went to Scott (issue
  #19).** Every compile-time criterion passed on both shipped shapes — 0
  accumulator spills, 0 calls and 0 surviving bounds checks in the steady-state
  K-loop, `ssa.html` archived for each, all three peak kernels register-only — and
  correctness passed on all three amd64 hosts with the AVX-512 tile exercised on
  each. The single failing line was the floor on janus.local (Skylake-X, i9-9960X):
  46.1%, 46.1% net of CI, reproducing at 46.0% on an independent run. Nothing was
  relaxed to change the colour: no shape added or removed, no threshold moved, no
  host dropped, no assembly written.
- **The P2 throughput floor now has a class-dependent denominator, and the gate is
  green** (DESIGN.md §4/P2 amendment, ruling on issue #19). One written rule, not
  two rules and a wink: an **FMA-bound** host keeps the flat ≥55% of measured peak;
  an **issue-bound** host is held to **≥90% of its issue roofline**, computed from
  the spill audit's own instruction counts. Classification and floor are the pure
  function `scripts/roofline.sh`, and three properties keep it from being a licence:
  - **Independence.** The ceiling mixes are every mix *except* the shape being
    gated. The first draft included it, which made the 90% floor algebraically
    vacuous: with the shape under test in the ceiling set,
    `attain ≥ 1/cspread ≥ 1/1.10 = 0.909 > 0.90` as an identity, so no host could
    ever fail that criterion. Caught by trying to write a fixture that failed it.
  - **Falsification.** If the shape under test retires *above* the ceiling the other
    mixes set (`attain > 1.0`), the issue-bound hypothesis is disproved by its own
    data and the host reverts to the flat floor. This is what returns antares
    (Zen 5) to FMA-bound: its mixes converge to 1.091× but 4×32 retires at 158.5%
    of the 39.3% roofline they imply.
  - **Bounded leniency.** The register-only peak kernel is always in the ceiling set,
    pinning `maxᵢ p_i ≥ 2.25`, and a shape more than 5% above the 115-shape sweep's
    best 4.438 insns/FMA is refused a roofline outright — so a kernel cannot pad
    itself into a lower bar, and the effective floor can never fall below
    `0.90 × 2.25 / 4.659` = **43.5%** of measured peak.

  It also **ratchets rather than expires**, which is stronger than the
  "self-retiring" property first claimed for it (that claim was false — the
  arithmetic shows janus stays issue-bound after the fix). The floor is
  `0.90 × maxᵢ p_i / I_b`, monotone in the gated shape's instruction count, so
  fixing the lowering *tightens* the gate: with T9+T12 landed, janus's roofline is
  78.3% and its required floor 70.4% — above the 55% it replaced.
- **Gate P2 is green on all three amd64 hosts under that one rule**
  (`bash scripts/gate-p2.sh`, exit 0): vesta.local FMA-bound at 96.6% of measured
  peak (96.6% net of CI), antares.local FMA-bound at 64.2% (62.3% net of CI) after
  its issue-bound hypothesis is falsified, janus.local issue-bound at 46.0% of peak
  = 94.6% of its 48.6% issue roofline. The performance-governor requirement is met
  by janus. The compile-time criteria are unchanged and still binding, and the
  15 verdict fixtures run before any benchmark. janus.local becomes the standing
  regression sentinel for P3: it is the host where instruction count binds, so it is
  the host that notices when a shape gets fatter.
- **The advertised dispatch chain is now stated per level: Level 1
  `avx512→avx2→scalar`, Level 3 `avx512→scalar`** (`DESIGN.md` §3 and §4/P5,
  `dispatch.go`, ruling on issue #40). Runtime behaviour is unchanged — there has
  never been an AVX2 microkernel and `KEEL_FORCE=avx2` has always acted as a
  *ceiling* at Level 3, reporting `scalar` — but the documentation claimed a
  three-rung chain at both levels, and the missing rung was the discrepancy. The
  narrowing goes in the strict direction: no gate check was deleted. `gate-p5.sh`
  now **requires** that forcing a Level-1-only rung yields a scalar microkernel and
  that `kern=` never names `avx2`, so a claim that grows back silently fails the
  gate that found it. The coverage marker gained a field for the same reason
  (`keel-p5-dispatch: l1=avx512,avx2,scalar kern=avx512,scalar`): a ruling that
  cannot be stated is one the next session re-litigates. Level 1 keeps its AVX2
  path, which is measured and has been gated since P1. The AVX2 microkernel is
  **deferred with its unblocking condition named** rather than dropped — an
  AVX2-native evidentiary host, since `KEEL_FORCE=avx2` on an AVX-512 machine
  establishes correctness and says nothing about performance on a part that lacks
  AVX-512. Debt with a trigger, not a wish.
- **P5's `-race` criterion is not amendable to exclude `checkptr`, and #42 is
  merged into #22 as an admissibility condition** (`DESIGN.md` §4/P5, ruled
  2026-08-12). A library that fatals under `go test -race` inside a 1×1 `Sgemv`
  through its public API is unshippable to a Go audience: race-clean is table
  stakes, and checkptr-clean is what race-clean means for code holding `unsafe`.
  Excluding the pointer checker would certify keel safe minus the instrument that
  checks. So the criterion stands unamended and the T17 workaround lands inside
  #22's campaign rather than as a point patch — **stage 1 measures masked-partial
  against zero-padded-panel among `checkptr`-clean implementations only.** A faster
  variant that fatals under the pointer checker is disqualified, not ranked:
  admissibility first, then speed.
- **janus.local keeps its gate-host and sentinel roles through P5** (confirmed
  2026-08-12; `docs/hosts.md`, `DESIGN.md` §4/P5). The question was whether a host
  sitting at 31.9% of measured peak should go on certifying phases. It should: it is
  the only Intel part, the only issue-bound one, and the machine the roofline
  amendment's standing task names as the thing to re-measure when the lowering
  improves. Because that amendment ratchets — the floor is monotone non-increasing
  in the gated shape's instruction count — the exception tightens automatically
  instead of granting a permanent dispensation. A host whose low number is
  *explained*, by a model that gets stricter as the explanation goes away, carries
  more information per run than a host that simply passes.
- Filed upstream as **[golang/go#80856](https://github.com/golang/go/issues/80856)**:
  `archsimd`'s partial slice load/store are not `checkptr`-safe (toolchain note
  T17, issue #42). This one is a correctness-contract violation rather than a
  performance finding like #17/#18 — the helpers manufacture a full-width
  `*[16]float32` from a short slice, it reproduces without the race detector, and
  it is demonstrated through a public API at the minimum input. #42 carries the new
  `standing-task` label and no milestone, the same pattern as #17/#18: when
  upstream's helpers go `checkptr`-clean, keel's workaround retires.
- **`docs/hosts.md` no longer explains antares's 43% confidence interval with a
  governor that is not set** (issue #44). The host was in `powersave` when that
  interval was measured on 2026-08-11 and has been in `performance` since the
  OpenBLAS provisioning campaign — every gate run from 2026-08-12 reads
  `governor=performance` from the machine itself. Removing the suspect settles
  nothing, so the paragraph now records a question instead of an answer: the
  variance was **re-measured under the asserted governor** the same day, three runs,
  same benchmark and methodology. The 43% interval did not reproduce — the widest
  scalar interval was 14% — but three draws cannot exclude a rare event, so the
  result is a *bound* (≤14% in three runs), not an absolution. What the runs did
  establish is a location: the scalar median carried 3–14% intervals while the
  AVX-512 median carried 0–1% in the same runs on the same host, an order of
  magnitude apart. That comparison is differential — two kernels co-measured under
  identical conditions — so it is a property of one kernel rather than ambient
  machine noise. **Location identified, mechanism open:** "the kernel that touches
  memory" does not discriminate, because both kernels read the same two arrays over
  the same 32 KB working set; what differs is runtime per op (~10×), frequency
  sensitivity and issue character. Two candidate classes, neither favoured by the
  data — clock-domain exposure (a loop spanning 10× the wall time samples 10× more
  boost and thermal wander, which on a mobile APU is at least as available as any
  cache-hierarchy story) and memory-path behaviour at that working set. So the
  governor is not credited with fixing what it was never shown to cause, and the 43%
  now has a bound and a location but **no identified cause** — recorded on issue #15
  beside vesta's bimodal ratio, which is the same character of finding. A stale explanation is worse than a missing one, because the
  next unstable measurement here would be attributed to a setting nobody set. The
  dated measurement records are unchanged and now say they are dated; the P1 table
  notes that antares's rows predate the change and are not restated as current.
- **Stage 3 gets an evidentiary bare-metal host, and the ≥6× floor does not move**
  (`DESIGN.md` §4/P5, `docs/hosts.md`, ruling on issue #12). Cloud hosts split into
  two classes with different licences: **evidentiary** (`c7i.metal` for a true
  512-bit server datapath, `c7a.metal` for Zen 4 server) may produce published
  scaling curves, and is metal-only because on shared tenancy a noisy neighbour and
  an invisible frequency ceiling are indistinguishable from a bad loop nest;
  **correctness** (spot, any µarch) may widen the differential sweep, where a noisy
  neighbour cannot change a bit-exact answer. The motive is that 6× at 8 threads on
  a 16-core client part does not locate where the nest stops scaling — packing-buffer
  contention invisible at 16 threads is the whole show at 64. Adding hosts makes the
  gate **stricter**: the floor stays ≥6× at 8 threads on 4096³, every host must clear
  it, and the wider 16/32/64-thread curve is reported beside the judged number rather
  than becoming a threshold invented after seeing the data. Gates keep running on the
  three local hosts, and the metal hosts are launched only when stage 2's nest exists
  to be measured.
- Issue bookkeeping brought current, which finishes the ruling from two sessions
  back. The four pre-existing open issues outside P5 were triaged against a stated
  rule — ruled-but-unlanded one-liner, deferred-to-a-P5-measurement,
  upstream-dependent standing task, or overtaken by events — and #34, #10, #8 and
  #12 all moved to the P5 milestone with their sorting recorded on each. With every
  earlier milestone empty of open issues, **milestones P0, P1, P2, P3 and P4 are
  closed**; P5 is the only open one.
