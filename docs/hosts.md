# Execution hosts

keel is developed on `darwin/arm64`, where `simd/archsimd` does not exist
(docs/toolchain-notes.md T1). Anything the *compiler* decides is checked
locally by cross-compiling; anything the *CPU* decides has to run on amd64
hardware. This file records how that happens and on which machines.

## How remote execution works

`scripts/remote.sh` cross-compiles a package's test binary for `linux/amd64`
with `go test -c`, ships it over `scp`, and runs it. Two consequences worth
stating plainly:

- **No toolchain is installed on the remote host.** `CGO_ENABLED=0` makes a
  static, pure-Go ELF binary; the host needs nothing but `sshd`. The compiler
  whose output gets executed is the same one the gate just version-checked, so
  there is no second toolchain to drift.
- **It works for benchmarks too.** `go test -c` includes `Benchmark*`, so
  P1's ≥4× check and P2's percent-of-peak measurement run the same way. What
  cannot cross the wire is anything needing local `perf`/`ssa.html`
  inspection, which is a compile-time artifact and stays local anyway.

Configure targets with `.keel-hosts` at the repo root (one host per line,
gitignored — see `.keel-hosts.example`) or `$KEEL_REMOTE_HOSTS`, which takes
precedence. Real hostnames are infrastructure, not source, so they are not
checked in.

## Requirements for a target

| Requirement | Why |
|---|---|
| amd64 | `simd/archsimd` is amd64-only on go1.26.5 |
| AVX2 + FMA | the AVX2 backend; `archsimd.X86.AVX2() && .FMA()` |
| AVX-512 F/CD/BW/DQ/VL | the bundle `archsimd.X86.AVX512()` gates on — all five, or the backend does not register |
| key-based ssh from the dev host | `remote.sh` uses `BatchMode=yes`; it never prompts and never handles credentials |
| fixed frequency governor | only for P2/P3 numbers; irrelevant to correctness |

## Current targets

All three were verified on 2026-08-10 and all three run all three backends.

### Zen 4 — AMD Ryzen 9 7950X3D, 16C/32T, Linux 6.17, `performance`

The primary target. DESIGN.md §4/P3 sizes its initial blocking parameters for
"a Zen4/Ice Lake-class target", so this is the machine those numbers are aimed
at. AVX-512 feature set is essentially complete for Zen 4 (F, DQ, IFMA, CD,
BW, VL, VBMI, VBMI2, VNNI, BITALG, VPOPCNTDQ, BF16, plus GFNI/VAES/VPCLMULQDQ).

keel uses none of that beyond F/DQ/CD/BW/VL. VNNI (int8 dot product) and BF16
are for quantized and reduced-precision inference, which DESIGN.md §2 lists as
non-goals for v0; they are noted here only so a future parking-lot item knows
the capability exists on this machine and not the other.

**Carries a caveat for P2.** Zen 4 implements AVX-512 on 256-bit datapaths:
a 512-bit FMA is split into two 256-bit µops issued to the two FMA pipes, so a
core retires one 512-bit FMA per cycle, not two. DESIGN.md §4/P2's peak
formula (`cores · freq · 2 FMA ports · 16 lanes · 2 flops`) assumes two
512-bit FMA units and therefore **overstates this machine's float32 peak by
2×**. Using it unmodified would make the 55%-of-peak floor a demand for ~110%
of what the hardware can do. The denominator has to be fixed before the P2
go/no-go can mean anything — tracked as issue #11, and no P2 work starts until
it is settled.

A corollary that matters for P2's spill audit: on Zen 4, AVX-512 and AVX2 have
the *same* float32 FLOP ceiling. AVX-512's advantage there is 32 architectural
vector registers instead of 16, plus masking and fewer instructions per unit of
work — which is exactly the currency the spill audit deals in, so a Zen 4
AVX-512 kernel can still beat its AVX2 twin without either exceeding a shared
peak.

### Skylake-X — Intel Core i9-9960X, 16C/32T, RHEL 9, `performance`

The second data point, and a genuinely different machine rather than a
duplicate:

- True 512-bit datapath with two 512-bit FMA units, so DESIGN.md's peak
  formula applies here as written. Useful as a cross-check on the Zen 4
  denominator question.
- The narrow AVX-512 tier — F, DQ, CD, BW, VL only. Since that is exactly the
  bundle `archsimd.X86.AVX512()` requires, it is the floor of what keel's
  AVX-512 backend may assume. If a future kernel reaches for VBMI2 or VNNI, it
  fails here first, which is the point.
- Heavy AVX-512 code triggers a substantial frequency license drop on this
  microarchitecture. Nominal clock is therefore the wrong denominator; P2's
  harness must measure frequency *during* the AVX-512 workload, which
  DESIGN.md §4/P2 already asks for ("CPUID + measured frequency") and which
  this machine makes non-optional.

### Zen 5 — AMD Ryzen AI MAX+ 395 (Strix Halo), 16C/32T, Linux 7.0, `powersave`

The newest microarchitecture available, and the widest AVX-512 feature set of
the three: F, DQ, CD, BW, VL, IFMA, VBMI, VBMI2, VNNI, BITALG, VPOPCNTDQ, BF16,
and `avx512_vp2intersect` — the last being genuinely unusual, since Intel
shipped it on Tiger Lake and then dropped it. keel uses none of it beyond
F/DQ/CD/BW/VL; it is recorded because it makes this the second machine (with
vesta) that could host any future reduced-precision parking-lot work.

Two things to settle before it produces a number, both instances of the same
principle:

- **Governor is `powersave`**, unlike the other two. Any benchmark taken here as
  it stands is measuring the governor as much as the kernel. Fix before
  measuring, or report it as a floor rather than a result.
- **Its FMA datapath width is unknown to this project and must be measured, not
  looked up.** AMD has shipped Zen 5 in both configurations — full 512-bit on
  some parts, 256-bit double-pumped on others — and which one an APU of this
  class uses is exactly the kind of detail that is easy to assert confidently
  and wrongly. With Zen 4 (double-pumped), Skylake-X (true 512-bit) and Zen 5
  (unknown) all in the pool, the argument in issue #11 stops being a Zen 4
  special case: the percent-of-peak denominator has to come from a measured FMA
  ceiling per host, not from a formula with a hardcoded port count.

Also worth noting for P3 rather than P2: this is an APU with unified LPDDR5X
rather than separate DIMM channels, so its bandwidth and cache hierarchy differ
structurally from the two desktop/HEDT parts. DESIGN.md §4/P3 sizes KC/MC/NC
against a cache hierarchy, so the blocking parameters that suit vesta should not
be assumed to transfer here. Measure per host.

## What has been verified here

2026-08-10, gate P0: the differential suite ran on all three hosts, bit-exact
across all 14 shim ops on all three backends. This settled the `Max`/`Min`
operand-order question (issue #9) that disassembly could not: `x.Max(y)`
returns `y` for NaN and for `max(±0, ∓0)` on Zen 4, Zen 5 and Skylake-X alike,
which is what the scalar spec already claimed. Three independent
implementations of `VMAXPS` agreeing is the reason that claim is now stated as
fact rather than as a convention keel happens to follow.
