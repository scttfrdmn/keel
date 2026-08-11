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

### Changed
- `vec.ScalarMax`/`ScalarMin`: the operand-order claim for NaN and signed zero
  is now **verified on hardware** rather than marked UNVERIFIED. `x.Max(y)`
  returns `y` for NaN and for `max(±0, ∓0)`, as the spec already said —
  confirmed bit-exactly on Zen 4, Zen 5 and Skylake-X — three independent implementations
  of `VMAXPS`. Disassembly could not settle this; only execution could.
- Gate P0 is green. All 14 shim ops agree bit-exactly across scalar, AVX2 and
  AVX-512 on all three amd64 hosts, and both FMA wrappers lower to a single
  `VFMADD213PS`.
