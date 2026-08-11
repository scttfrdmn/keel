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
  enforced backend coverage, and FMA fusion verified by disassembly).
- `docs/toolchain-notes.md`: six field notes on `GOEXPERIMENT=simd` in go1.26.5,
  each with a minimal repro — archsimd being amd64-only, the `VFMADD213PS`
  lowering, the absent float32 `Abs`/bitwise ops, the absent `Float32x16`
  horizontal reduce, the absent portable `simd` package, and the unverified
  `Max`/`Min` NaN operand order.
