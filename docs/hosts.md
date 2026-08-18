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
| a clock established stable | for gate numbers, not correctness. DESIGN.md §5 rule 5 (as amended 2026-08-16) requires it of every measuring host, by whichever instrument the host has: the `performance` governor where `cpufreq` is readable, else `BenchmarkPeak` sampled at head/middle/tail. Both branches are live — `clock_gate`/`clock_head`/`clock_post`, and the guest branch is what the AWS fleet runs on, where `assert_governor` reads `absent` |
| an admission class | `evidentiary` (full-size, judged) or `correctness` (partial-size, reported); see the class table below. Unreadable is `unmeasured`, not `correctness` |

## Current targets

All three were verified on 2026-08-10 and all three run all three backends.

### Private cache sizes (read from sysfs 2026-08-12, not looked up)

| Host | µarch | L1d | L2 | L3 |
|---|---|---|---|---|
| vesta | Zen 4 | 32K | 1024K | 98304K (V-Cache) |
| janus | Skylake-X | 32K | 1024K | 22528K |
| antares | Zen 5 | **48K** | 1024K | 32768K |

This table is here because a reading of the #48 feed decomposition turned out to
need it, and getting it from memory would have been wrong: **Zen 5 has a 48 KB L1d
where Zen 4 and Skylake-X have 32 KB.** `remote_probe` now reports these from
`/sys/devices/system/cpu/cpu0/cache/index*` on every run, so the numbers arrive with
the measurement instead of being asserted next to it.

What they are used for: three of `BenchmarkFeed`'s arms reuse one packed panel pair
of `(MR+NR)·kk·4` bytes, which at NR=32 is 17/34/51/68 KB for `kc=128/256/384/512`
(2×32) and 18/36/54/72 KB (4×32). So the "reused pair" is L1-resident only at
`kc=128` on vesta and janus, and at `kc=128` *and* `256` on antares — and above that
threshold both panel arms feed from L2, where all four sizes fit on all three hosts.
The step between them is then a difference of locality *within* L2 rather than a
difference of level, which is a different measurement from the one the arm's name
suggests. The threshold moves with the host, which is exactly why the number has to
come from the host.

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

### Zen 5 — AMD Ryzen AI MAX+ 395 (Strix Halo), 16C/32T, Linux 7.0, `performance`

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
- **The 43% confidence interval on this host has no explanation on record, and
  that is the honest state of it** (issue #44). On 2026-08-11 this machine was in
  `powersave` while the other two were in `performance`, and one gate run reported
  the scalar Sdot median with a **43% confidence interval** (point-estimate speedup
  16.03×, against 8.96× the next run). The governor was the obvious suspect and was
  written here as the cause. It has since been set to `performance` during the
  OpenBLAS provisioning campaign — every gate run from 2026-08-12 reads
  `governor=performance` from the machine itself — which removes the suspect
  without settling anything: the variance was measured under the old setting and
  has not been re-measured under the new one.

  **Re-measured under the asserted governor on 2026-08-12, and the governor was not
  the whole story.** Three independent runs of the same benchmark under the same
  methodology (`GateSdot` at n=4096, `-count=10 -benchtime=1s`, benchstat median):

  | run | scalar median | avx512 median | ratio | net of CI |
  |---|---|---|---|---|
  | 1 | 9.437e-07 s ± 8.0% | 9.505e-08 s ± 0.0% | 9.93× | 9.13× |
  | 2 | 1.039e-06 s ± 14.0% | 9.514e-08 s ± 1.0% | 10.92× | 9.29× |
  | 3 | 1.017e-06 s ± 3.0% | 9.519e-08 s ± 0.0% | 10.69× | 10.37× |

  Two readings, and both belong here:

  - **The 43% interval did not reproduce.** The widest scalar interval in three runs
    was 14%. That does not *exclude* a 43% outlier — three draws cannot rule out a
    rare event, and the original 43% was itself one run in three — so the honest form
    is a bound, not an absolution: under `performance`, three runs produced ≤14%.
  - **A scalar-path instability remains, and the location is established.** The
    scalar median's interval ranged 3–14% while the AVX-512 median's ranged 0–1% in
    the same runs, on the same host, minutes apart. That comparison is
    *differential* — two kernels co-measured under identical conditions — so an
    order of magnitude between them is a property of one kernel rather than ambient
    machine noise. The governor therefore cannot be credited with having caused what
    was never shown to be caused by it, and the 43% has **no identified cause**: it
    has a bound and a location.

  **Location identified, mechanism open, and the difference matters.** "The kernel
  that touches memory" is the tempting phrase and it does not discriminate: both
  kernels read the same two arrays over the same 32 KB working set. What actually
  differs between them is runtime per op (~10×), frequency sensitivity and issue
  character. So there are two candidate classes, and neither is favoured by the data
  in hand:

  1. **Clock-domain exposure.** A scalar loop spanning ten times the wall time per
     measurement samples ten times more boost and thermal wander, which on a mobile
     APU under a shared power budget is at least as available a mechanism as anything
     in the cache hierarchy.
  2. **Memory-path behaviour** at that working set — the reading the phrase above
     assumes, which would need evidence the two kernels' memory traffic differs in a
     way their identical arrays do not already fix.

  Both untested. This is the same *character* of finding as vesta's bimodal Sdot
  ratio and lives in the same place: issue #15, the run-to-run instability and
  pinning decision, which now carries both hosts' data. Keeping the mechanism open
  is the same discipline that reopened the 43% in the first place — a plausible
  phrase is not an explanation.

  What is not in question: the register-only peak measurement was unaffected (0.4%
  across three runs), so whatever this is, it appears only once a benchmark touches
  memory. The net-of-CI rule absorbed it correctly either way, reporting 9.13×,
  9.29× and 10.37× across the three runs above, which is the rule doing its job
  rather than a reason to trust the point estimate.

  > **Correction, 2026-08-16 (#79).** This paragraph previously read "reporting
  > 9.14×, 8.96× and 8.79× across three runs". That trio is not from these runs: it
  > was published on 2026-08-11 by `4643b63`, *before* the governor fix, and the
  > three runs tabled immediately above report 9.13× / 9.29× / 10.37×. The
  > misattribution also reversed the paragraph's own point — the trio it quoted
  > spans 4.0%, the runs it was describing span 13.6%, so it cited a tight spread as
  > evidence while the data in front of it was the loose one. The 2026-08-11 trio has
  > no surviving derivation at all (see the note at the summary table below); the
  > figures here are the ones this section actually measured.

Also worth noting for P3 rather than P2: this is an APU with unified LPDDR5X
rather than separate DIMM channels, so its bandwidth and cache hierarchy differ
structurally from the two desktop/HEDT parts. DESIGN.md §4/P3 sizes KC/MC/NC
against a cache hierarchy, so the blocking parameters that suit vesta should not
be assumed to transfer here. Measure per host.

## Cloud hosts: two admission classes (ruled 2026-08-12 on #12; amended 2026-08-17 on #104)

Two classes, and the distinction is what each one is allowed to produce:

| class | machines | produces | admission requires |
|---|---|---|---|
| **evidentiary** | a **full-size (whole-socket)** instance of an approved family | judged perf verdicts; published rows; the stage-3 curves | full size **and** a passing preamble: clock stability established by §5 rule 5's instrument for the host it is, and the instance type in the provenance block |
| **correctness** | any partial-size guest, any µarch | differential and correctness coverage; perf numbers **reported, never judged** | nothing beyond reachability |

**Amended 2026-08-17 (ruling on #104): the evidentiary class is full-size, not metal.**
Scott's earlier ruling retired bare metal outright — *"no one will ever use this library
with bare metal"* — which left the class with no members and the harness judging perf on
whatever guest answered. The property #12 actually argued for was exclusive ownership of
the silicon under measurement, and a whole-socket instance has that while sharing the
deployment model a caller really uses. So the *size* carries the admission, and the
preamble carries the proof.

The reason for the split is that the two roles have different failure modes. A
correctness run either agrees bit-for-bit with the scalar spec or it does not, and a
noisy neighbour cannot change that answer; a throughput run on shared tenancy cannot
distinguish a noisy neighbour or an invisible frequency ceiling from a bad loop nest.

**A number from a correctness-class host is reported-not-judged no matter what it reads**,
high or low, which is the half that had been missing: #104 put a *low* reading from
`c7i.4xlarge` — 16 vCPU of a Sapphire Rapids socket, 8 physical cores — through a floor
written for a machine keel owned, and called the result a P2 STOP. Class is checked
before the number is trusted, not after it surprises someone. And the check fails closed:
an **unreadable** class is `unmeasured`, never "not judged" — otherwise the mechanism that
excuses a partial-size reading is also a mechanism for laundering a red.

What the evidentiary hosts are *for*: the ≥6× floor was written when the largest gate
host had 16 cores. 6× at 8 threads on a client part with client memory channels says
little about where the parallel nest actually stops scaling — packing-buffer contention
invisible at 16 threads is the whole show at 64. **The floor does not move**; the
full-size hosts add a wider curve (16/32/64 threads) reported beside the judged number,
and they must clear the same ≥6× every other host clears, so adding them can only
make the gate stricter.

Mechanics: `scripts/aws-fleet.sh up|wire|status|down`, one entry per host with its class
recorded, torn down at session end. **Launched when there is something to measure** — a
full-size instance running during stage 1 would bill for hours and measure nothing. The
standing grant (2026-08-17) covers repeats and re-runs of an instance type already
approved; a **new** type is Scott's call each time.

## What P3 asks of every host, and of one

**The OpenBLAS reference is same-host, on all three** (ruling on issue #23).
There is no reference-host list and no golden machine. P3's second criterion is
≥60% of single-thread OpenBLAS at 2048³, and the only apples-to-apples version of
that ratio is same silicon, same thread count, same run — so each host is divided
by *its own* OpenBLAS, in one benchmark invocation, from a native build of the
`openblas`-tagged cgo harness shipped as `git archive HEAD`. A cross-host ratio
would compare two microarchitectures and attribute the difference to keel
(DESIGN.md §7 rule 7); this dev host would be the worst case of that, being
`darwin/arm64` where keel's AVX-512 path does not exist at all.

That makes the amd64 hosts the one place in this project that needs a Go
toolchain *and* a system library installed, rather than a cross-compiled static
binary. None of the three had either as of 2026-08-11:

| Host | distro ID | `go` on a non-login `ssh` PATH | `libopenblas.so` | governor |
|---|---|---|---|---|
| vesta | `ubuntu` | none | none | performance |
| janus | `rocky` | none | none | performance |
| antares | `ubuntu` | none | none | powersave → performance (set during provisioning; #44) |

(janus runs Rocky Linux 9, not RHEL proper — the `dnf` package name is the same.)

Provisioning is Scott's decision and Scott's `sudo`, so it lives in
`scripts/provision-openblas.sh`, which he runs; the gate connects with
`BatchMode=yes` and never handles a credential. A host that cannot produce a
reference **fails** the gate by name, with the exact commands for its own distro
printed. It does not fall back to percent-of-peak: CLAUDE.md's "the OpenBLAS
reference when available; otherwise say it isn't" is a rule about reporting
numbers, and using it to satisfy a criterion would be weakening a gate.

Three things get recorded per host, because each of them moves the ratio:

1. **the OpenBLAS version and build string** — what the denominator *is*;
2. **`OPENBLAS_NUM_THREADS=1` taking effect**, read back from
   `openblas_get_num_threads()` rather than assumed from the environment;
3. **the `DYNAMIC_ARCH`-selected kernel family**, from
   `openblas_get_corename()`, checked against an AVX2-or-better allowlist
   (`haswell skylakex cooperlake sapphirerapids zen`).

The third is the one that is easy to omit and the only one whose failure mode is
*in keel's favour*: a package that quietly selects a generic or pre-AVX2 kernel
on an AVX-512 host produces a reference that reads low, which inflates keel's
ratio while the version string, the thread count and the config line all still
look right. An unrecognized name fails too — missing knowledge should cost a
human one line of diff, not silently widen a bar.

On an **issue-bound** host the denominator is
`min(same-host OpenBLAS, roofline × measured peak)` (the same ruling, citing
#17/#18): OpenBLAS's K-loop there is hand assembly folding accumulation and an
embedded broadcast into single FMAs, which the intrinsic layer provably cannot
emit (T12), so it sits *above* the front-end ceiling keel's kernels are capped
by. vesta and antares classify FMA-bound and therefore face the unmodified
criterion — which keeps at least one host where the comparison is completely
unassisted. Both ratios, amended and plain, are printed on every host.

Note what stays true regardless: the `openblas` tag keeps that harness out of the
module's dependency graph, so nothing keel ships ever links OpenBLAS.

*The dev host has OpenBLAS, and that is useful for exactly one thing.* Homebrew
OpenBLAS 0.3.34 is installed here, so the tagged harness is compile- and
run-verifiable locally — the cgo declarations, the linker flags, the provenance
plumbing and the thread pin were all checked this way before the gate ever shipped
them to a remote host (see toolchain note T13 for the file layout cgo forced).
What it cannot do is produce the criterion, and the measurements say why more
plainly than the architecture argument does:

| Config (n=1024, single sub-benchmark) | Rate |
|---|---|
| OpenBLAS, `OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1` | ~1410 GFLOP/s |
| OpenBLAS, unpinned (`threads=12`) | ~1400 GFLOP/s |
| keel, scalar backend (n=256) | ~6 GFLOP/s |

The pin works and is honest — `openblas_get_num_threads()` reports 1, and user CPU
time equals wall time to within a few percent, so one thread really is doing the
work. That one thread reaches roughly ten times what this core's NEON FMA units
can issue, and adding eleven more threads buys nothing, which together say the
reference is running on the M4's matrix unit rather than on its vector units
(config target `vortexm4`). Against that, keel's *scalar* path is the only thing
this host can run. A ratio of ~1:200 between a matrix coprocessor and a Go scalar
loop is not evidence about the AVX-512 kernel P3's criterion is actually about.

Two of the longer local runs of the *identical* pinned configuration also came in
at 119 and 712 GFLOP/s — up to 12× run-to-run — which is what a laptop with no
governor to pin and an aggressive QoS scheduler looks like. DESIGN.md §5 rule 5's
methodology exists to exclude exactly this, and it is a second, independent reason
no number from this machine is reportable.

**The throughput sentinel** is the one role P3 assigns to a single host, named in
`.keel-sentinel` or `$KEEL_SENTINEL_HOST` — same format as `.keel-hosts`, and
gitignored for the same reason. The ruling
on issue #19 left P2's floor with a class-dependent denominator, and on an
issue-bound host that floor *rises* as the kernel's instruction count falls. The
same arithmetic makes such a host the one that notices a K-loop getting fatter,
which is the risk P3 carries — packing, edge handling and beta variants all add
code around the loop. So gate P3 re-runs P2's verdict there. janus is the
sentinel today: it is the host where instruction count binds (46.0% of peak,
94.6% of a 48.6% issue roofline). If the file is absent, every host is a
sentinel — missing configuration costs time, never coverage.

**janus keeps that role through P5** (confirmed 2026-08-12 by ruling). The
question raised was whether a host sitting at 31.9% of peak should go on
certifying phases, and the answer is that this is precisely the host worth
keeping: the only Intel part, the only issue-bound one, and the machine the
roofline section's standing task names as the thing to re-measure when the
lowering improves. Because the amendment ratchets — the floor is monotone
non-increasing in `I_b` — the exception tightens automatically rather than
granting a permanent dispensation. A host whose low number is *explained*, by a
model that gets stricter as the explanation goes away, carries more information
per run than a host that simply passes.

Gate P3 classifies *every* host with that same measurement even when only one is
judged by it, because the amended denominator above applies only where the
classifier says issue-bound. A host whose classification could not be measured is
treated as FMA-bound and faces the unmodified bar, which is the strict direction.

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

Those rows are dated, and one of them is dated for a reason: **antares was in
`powersave` when its figures were taken** and is in `performance` now (#44), so its
Sdot column in particular is not directly comparable with a re-run today. The
measurements stand as records of what was measured; they are not restated as
current.

> **Provenance of antares's Sdot trio, audited 2026-08-16 (#79).** Stronger than
> "dated": the trio is **unprovenanced**. No archived log in this tree contains
> `8.96×` as an Sdot ratio at all, so the derivation of at least one of its three
> entries no longer exists — DESIGN.md §5 rule 8's terminal case, a derived figure
> whose log is gone. Readings of the same quantity under the same methodology with
> the governor read back as `performance` are consistently higher: four archived
> `gate-p1` logs give 9.626× / 9.716× / 9.728× / 9.740× net of CI, and this
> document's own 2026-08-12 re-measurement gives 9.13× / 9.29× / 10.37×. Against
> those, the published `8.79×` is 9–10% low.
>
> The leading explanation for the trio's shape is arithmetic rather than physical:
> `9.14` is, to three significant figures, the **raw** ratio of the run whose
> net-of-CI is `8.79`, so a list captioned "net of CI" may hold one raw value and
> "three runs" may be two. That is a hypothesis, not a finding.
>
> The governor is **not** credited with causing the gap. #44 established that it was
> a suspect *removed*, not a cause *shown*, and inferring here what that
> investigation declined to infer from data would repeat the error one document
> later. The figures are marked, not rewritten (#79's ruling): the gap is real, the
> direction is consistent across seven independent readings, and the cause is
> unestablished.

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
antares to 4% (9.14×, 8.96×, 8.79× — but see the provenance note below; the
`performance`-era readings of that host span 13.6%, not 4%). So this is specific to
vesta, and the
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
