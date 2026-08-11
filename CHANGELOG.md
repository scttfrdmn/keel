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

### Changed
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
