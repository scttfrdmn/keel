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
| `performance` governor | for gate numbers, not correctness. DESIGN.md §5.4 rule 5 requires at least one host to clear a perf bar under it; every host must still clear the bar |

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

**The double-pumped datapath, measured rather than looked up.** Zen 4
implements AVX-512 on 256-bit datapaths: a 512-bit FMA is split into two
256-bit µops, so a core retires one 512-bit FMA per cycle, not two. That is now
this project's own measurement rather than a citation — `BenchmarkPeak` on
2026-08-11 returned **165.6 GFLOP/s at 512 bits and 165.5 at 256**, a width
ratio of **1.00×**. Two vector widths with identical float32 ceilings is what a
double-pumped datapath looks like from software, and it needs no clock estimate
or port count to interpret.

DESIGN.md §4/P2's original peak formula (`freq · 2 FMA ports · 16 lanes · 2
flops`) predicts 368.9 GFLOP/s here, **2.23× the measured ceiling**, which
would have turned the 55%-of-peak floor into a demand for ~123% of what this
hardware can do. That is why peak is now measured per host and the formula kept
only as a printed cross-check (issue #11).

A corollary that matters for P2's spill audit: on Zen 4, AVX-512 and AVX2 have
the *same* float32 FLOP ceiling. AVX-512's advantage there is 32 architectural
vector registers instead of 16, plus masking and fewer instructions per unit of
work — which is exactly the currency the spill audit deals in, so a Zen 4
AVX-512 kernel can still beat its AVX2 twin without either exceeding a shared
peak.

### Skylake-X — Intel Core i9-9960X, 16C/32T, RHEL 9, `performance`

The second data point, and a genuinely different machine rather than a
duplicate:

- True 512-bit datapath with two 512-bit FMA units — **measured width ratio
  2.12×** (215.9 GFLOP/s at 512 bits, 101.8 at 256), the opposite shape from
  vesta and the reason having both machines settles the question that either one
  alone could not.
- The narrow AVX-512 tier — F, DQ, CD, BW, VL only. Since that is exactly the
  bundle `archsimd.X86.AVX512()` requires, it is the floor of what keel's
  AVX-512 backend may assume. If a future kernel reaches for VBMI2 or VNNI, it
  fails here first, which is the point.
- Heavy AVX-512 code triggers a substantial frequency license drop on this
  microarchitecture, and the peak measurement shows it: 215.9 GFLOP/s at 512
  bits implies **3.37 GHz sustained** under an FMA-saturated load, against the
  4.4 GHz `cpuinfo_max_freq` the formula uses — which accounts for essentially
  all of the formula's 1.30× overstatement here. Nominal clock is the wrong
  denominator on this machine; measuring the ceiling directly sidesteps the
  question, which is the second independent reason peak is measured (issue #11).

### Zen 5 — AMD Ryzen AI MAX+ 395 (Strix Halo), 16C/32T, Linux 7.0, `powersave`

The newest microarchitecture available, and the widest AVX-512 feature set of
the three: F, DQ, CD, BW, VL, IFMA, VBMI, VBMI2, VNNI, BITALG, VPOPCNTDQ, BF16,
and `avx512_vp2intersect` — the last being genuinely unusual, since Intel
shipped it on Tiger Lake and then dropped it. keel uses none of it beyond
F/DQ/CD/BW/VL; it is recorded because it makes this the second machine (with
vesta) that could host any future reduced-precision parking-lot work.

- **Full-width 512-bit datapath, and the fastest of the three.** AMD has
  shipped Zen 5 in both configurations — full 512-bit on some parts, 256-bit
  double-pumped on others — and which one an APU of this class uses was exactly
  the kind of detail that is easy to assert confidently and wrongly, so it was
  measured: **327.8 GFLOP/s at 512 bits against 164.0 at 256, a width ratio of
  2.00×**. This is a full-width part, and its measured ceiling is 1.98× vesta's
  despite being a mobile APU. The formula agrees here to within **1.01×**, which
  is worth noting precisely because it is the exception: the same formula is
  2.23× high on vesta and 1.30× high on janus.
- **Governor is `powersave`**, unlike the other two. It does not appear to cost
  this machine its ceiling — the register-only peak measurement is unaffected —
  but it does cost reproducibility on memory-touching kernels: one gate run
  reported the scalar Sdot median with a **43% confidence interval** (and a
  point-estimate speedup of 16.03×, against 8.96× the next run). The net-of-CI
  rule absorbed it correctly, reporting 9.14×, 8.96× and 8.79× across three runs,
  which is the rule doing its job rather than a reason to trust the point
  estimate.

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

2026-08-11, gate P1 under the revised measurement methodology (issues #11, #14):
each host's FMA ceiling measured by `bench.BenchmarkPeak`, single core,
`-count=10 -benchtime=1s`, benchstat medians.

| Host | µarch | 512-bit | 256-bit | width ratio | formula error | Sdot n=4096 speedup (net of CI, three runs) |
|---|---|---|---|---|---|---|
| vesta | Zen 4 | 165.6 | 165.5 | 1.00× | 2.23× high | 8.71× / 7.18× / 8.57× |
| janus | Skylake-X | 215.9 | 101.8 | 2.12× | 1.30× high | 7.55× / 7.48× / 7.53× |
| antares | Zen 5 | 327.8 | 164.0 | 2.00× | 1.01× high | 9.14× / 8.96× / 8.79× |

GFLOP/s, float32. The width ratio is the useful column: it is a direct
measurement of the datapath, requiring no clock estimate, port count or
frequency-license assumption, because both halves were measured on the same host
under identical conditions. The measured ceilings reproduce to within 0.4%
between gate runs; the Sdot ratios do not (see below), which is the difference
between a register-only kernel and one that touches memory.

**One thing not yet settled: run-to-run drift on vesta, and it is bimodal.**
Three gate runs over a few hours reported the same Sdot ratio as **8.71×, 7.18×
and 8.57×** net of CI, while benchstat's within-run interval was 0–2% every
time. janus reproduced to 0.9% across the three (7.55×, 7.48×, 7.53×) and
antares to 4% (9.14×, 8.96×, 8.79×). So this is specific to vesta, and the
*shape* is informative: two runs agree to 1.6% and one sits 17% below them.
Thermal drift would be monotonic in run order; a bimodal distribution with a
tight upper cluster is what core placement looks like. The 7950X3D has two CCDs,
one with 3D V-Cache and a lower clock ceiling, and nothing pins these
single-threaded benchmarks to either — the scheduler picks, and roughly a third
of the time here it picked the slower one.

Measured *peak* is unaffected (0.4% across the same three runs), because a
register-only kernel has nothing to be lucky about; the drift appears only once a
benchmark touches memory. A within-run confidence interval cannot see any of
this, so "cleared net of CI" is not by itself protection against it, and a P2
threshold on a memory-touching kernel should not be trusted to 17% on this host.
Settling it needs `taskset` pinning, which changes what the measurement *means*
— pin to the V-Cache CCD or the higher-clocked one? the honest answer differs
for an L1-resident kernel and for a blocked SGEMM — and so is a decision to take
deliberately rather than a fix to slip in.
