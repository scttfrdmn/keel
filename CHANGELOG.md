# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, minor versions may contain breaking changes.

## [Unreleased]

### Added
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
  connection cannot answer and should not try to. It handles no credentials, and
  it reports a non-`performance` governor rather than changing a machine's power
  policy. The Go tarball's digest is enforced against `$KEEL_GO_SHA256` when set
  and otherwise against `go.dev/dl?mode=json`, and it says which of the two it
  did, since only the first is provenance.
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

### Changed
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
