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
- `docs/toolchain-notes.md`: seven field notes on `GOEXPERIMENT=simd` in go1.26.5,
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
  governor, clock snapshot and active backend. Theoretical peak is deliberately
  **withheld** with the reason printed inline: the DESIGN.md formula assumes two
  512-bit FMA units and Zen 4 double-pumps, so percent-of-peak has no trustworthy
  denominator yet (issue #11, which also blocks P2).
- `scripts/gate-p1.sh`: real P1 checks — both builds, vet, L1 tests with
  enforced per-backend coverage on every host, the whole suite re-run under
  `KEEL_FORCE=scalar` on machines that *have* AVX-512 (a scalar pass on arm64
  would not prove the override works), and the ≥4× Sdot ratio measured
  within-machine from min-of-5 samples with every raw sample printed.

### Changed
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
  on every host: 4.28× on Zen 4 (Ryzen 9 7950X3D), 5.91× on Skylake-X (i9-9960X),
  4.09× on Zen 5 (Ryzen AI MAX+ 395). Those margins are narrow and the
  `-benchtime=3x` samples they come from spread by up to 3× within one backend;
  see the measurement-methodology issue before treating them as stable.
