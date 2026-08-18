<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# P2 go/no-go report — the microkernel is spill-free and still misses the floor on one host

Required by DESIGN.md §4/P2 and §7 rule 4 (`CLAUDE.md`, "P2 is a go/no-go, not a
hurdle"). `scripts/gate-p2.sh` was **RED**. Work stopped here; nothing in P3 was
started, and no gate criterion was relaxed to change the colour.

> **Status: reopened 2026-08-18 — read §10 first, then §9.** §9 resolved this report on
> the three desktop hosts, which are retired. On the first *evidentiary* host under
> #104's ruling — a full-size `c7i.48xlarge` — both P2's floor and P3's mission ratio
> are red, and §10 shows the 55% floor is out of reach on a 2-FMA/cycle machine until
> golang/go#80829 lands *and* keel removes its own slice-advance overhead. §9 stands as
> the resolution for the fleet it was measured on.
>
> **Status of the original run: resolved — see §9.** The gate's *model*
> was amended by the ruling on issue #19 (the floor's denominator was wrong, not
> its value), the mechanism behind the ceiling was identified and filed upstream
> (T12/#20), and §5's instruction accounting was corrected — the toolchain gap is
> ~1.75× larger than this report originally measured. §§1–8 are preserved as the
> record of the red gate, which is the artifact this document exists to carry.

**The headline is not the one this file's name predicts.** The spill audit is
green — both shipped shapes have zero vector stack traffic, zero calls and zero
surviving bounds checks in the steady-state K-loop. What fails is the
55%-of-measured-peak floor, on one of three hosts, and the binding constraint is
**instructions issued per FMA**, not spills.

## 1. What the gate says

Verbatim from `scripts/gate-p2.sh`, 2026-08-11, go1.26.5:

```
-- spill audit: steady-state K-loop of the shipped shapes (Kernel2x32,Kernel4x32) --
        compile-time property, audited against the linux/amd64 object code the hosts run
        github.com/scttfrdmn/keel/internal/vec.Kernel2x32: steady-state loop [70,430] 74 insns for 16 arith (4.62 per arith): 0 vector stack refs, 8 reg copies, 8 broadcasts, 8 anchor nops, 0 calls, 0 bounds-check exits, 0 other mem refs
        github.com/scttfrdmn/keel/internal/vec.Kernel4x32: steady-state loop [101,341] 50 insns for 8 arith (6.25 per arith): 0 vector stack refs, 12 reg copies, 4 broadcasts, 2 anchor nops, 0 calls, 0 bounds-check exits, 0 other mem refs
  PASS  0 accumulator spills in the steady-state K-loop
  PASS  0 calls in the steady-state K-loop (no write barrier, no runtime helper)
  PASS  0 surviving bounds checks in the steady-state K-loop (panels are pre-sliced)
        ssa.html archived: build/ssa/Kernel2x32.html (5057363 bytes; gitignored, see KERNEL.md)
        ssa.html archived: build/ssa/Kernel4x32.html (4292139 bytes; gitignored, see KERNEL.md)

-- reference tile Kernel6x32: DESIGN.md's 12 accumulators, EVIDENCE ONLY --
        expected to spill on go1.26.5 (T10, issue #18); this audit cannot fail the gate
        github.com/scttfrdmn/keel/internal/vec.Kernel6x32: steady-state loop [152,1870] 270 insns for 48 arith (5.62 per arith): 90 vector stack refs, 50 reg copies, 24 broadcasts, 8 anchor nops, 0 calls, 0 bounds-check exits, 0 other mem refs
        spills as documented; the GFLOP/s cost of that is measured below

[ elided: the check_bce provenance section (57 bounds checks, all outside the
  K-loop) and the peak-kernel register-only section (avx512Peak 27 insns /
  12 arith, avx2Peak 23/10, scalarPeak 23/20, no memory in any loop) — both PASS ]

-- microkernel vs measured peak (>= 55% of measured; issue #14 methodology) --
        -count=10 -benchtime=1s; numerator and denominator measured in the same run
  PASS  cross-compiled linux/amd64 bench binary
        [vesta.local] governor=performance
        [vesta.local] peak   165.6 GFLOP/s +/- 0.0%
        [vesta.local] 6x32  50.44 GFLOP/s +/- 1.0% = 30.5% of peak (spills; not shipped)
        [vesta.local] 2x32/avx512/kc=128 153.1 GFLOP/s +/- 1.0% = 92.4% of peak, 91.5% net of CI
        [vesta.local] 4x32/avx512/kc=128 159.9 GFLOP/s +/- 0.0% = 96.6% of peak, 96.6% net of CI
  PASS  [vesta.local] best shipped shape 4x32/avx512/kc=128 at 96.6% of measured peak, 96.6% net of CI (>= 55%)
        [janus.local] governor=performance
        [janus.local] peak   215.9 GFLOP/s +/- 0.0%
        [janus.local] 6x32  39.24 GFLOP/s +/- 0.0% = 18.2% of peak (spills; not shipped)
        [janus.local] 2x32/avx512/kc=128 99.48 GFLOP/s +/- 0.0% = 46.1% of peak, 46.1% net of CI
        [janus.local] 4x32/avx512/kc=128 76.03 GFLOP/s +/- 0.0% = 35.2% of peak, 35.2% net of CI
  FAIL  [janus.local] best shipped shape 2x32/avx512/kc=128 at 46.1% of measured peak, only 46.1% net of CI (< 55%)
        [antares.local] governor=powersave
        [antares.local] peak   327.7 GFLOP/s +/- 0.0%
        [antares.local] 6x32  48.13 GFLOP/s +/- 0.0% = 14.7% of peak (spills; not shipped)
        [antares.local] 2x32/avx512/kc=128 174.1 GFLOP/s +/- 0.0% = 53.1% of peak, 53.1% net of CI
        [antares.local] 4x32/avx512/kc=128 210.2 GFLOP/s +/- 2.0% = 64.1% of peak, 62.8% net of CI
  PASS  [antares.local] best shipped shape 4x32/avx512/kc=128 at 64.1% of measured peak, 62.8% net of CI (>= 55%)
  PASS  the 55% floor was cleared under the performance governor (vesta.local)

gate-p2: RED
```

Reproducible: an independent run 40 minutes earlier gave janus 46.0% (vs 46.1%),
vesta 96.5% (96.6%) and antares 62.0% (64.1%). The failure is not variance.

Correctness is green everywhere: the differential tests pass locally on
darwin/arm64 (scalar) and on all three amd64 hosts with the `avx512` tile
exercised on each.

## 2. The scope question that has to be answered first

DESIGN.md §4/P2 says "the raw microkernel (packed inputs, no blocking) hits **≥55%
of measured peak**". It does not say on how many hosts. `scripts/gate-p2.sh`
resolves that as **every host that can run the kernel must clear it, and at least
one must do so under the `performance` governor** (criterion 5, and KERNEL.md §6).

Under a "one reference host" reading the gate would be green: vesta clears at
96.6%. Under the reading actually implemented it is red, because janus does not.
That is an ambiguity in the contract, and resolving it downward is exactly the
move `CLAUDE.md` forbids ("never weaken a gate to pass it"), so it is left open
as decision **D1** in §7 rather than settled here.

The strict reading is the one worth defending on the merits: P3 will dispatch
this kernel on all three machines, and a 46%-of-peak SGEMM on Skylake-X is a
product defect rather than a rounding error. P2 exists to find that before P3 is
built on top of it.

> **Superseded in P3 (issue #24).** The premise of that last paragraph — that P3
> dispatches one shape everywhere — no longer holds. P3 selects the microkernel
> shape per host from the same issue-bound/FMA-bound classification this report
> established, so janus now ships 2×32 rather than the 4×32 measured here; see
> KERNEL.md §8. The argument for the strict reading survives the change intact
> (it is *why* the shapes are now chosen per class), and the numbers below are
> left exactly as measured in P2 — this is a record of that phase, not a
> statement about what currently ships.

## 3. Diagnosis: janus is issue-bound, and this is measured, not modelled

Two independent derivations, both from keel's own numbers.

### 3.1 A clock-free test

If a loop is limited by instruction issue, its throughput scales inversely with
its instruction count per FMA. The two shipped shapes differ by construction:
2×32 ×4 is 74 instructions per 16 FMAs (4.625), 4×32 ×1 is 50 per 8 (6.250). So
an issue-bound host must show a 2×32 : 4×32 throughput ratio of
6.250 / 4.625 = **1.351**, and a host limited by anything else must not.

| host | 2×32 GFLOP/s | 4×32 GFLOP/s | ratio | |
|---|---|---|---|---|
| vesta.local | 153.1 | 159.9 | 0.957 | not issue-bound |
| janus.local | 99.48 | 76.03 | **1.308** | issue-bound on both shapes |
| antares.local | 174.1 | 210.2 | 0.828 | not issue-bound |

1.308 against a predicted 1.351 is a 3% miss. On the other two hosts the *wider,
more instruction-hungry* shape is the faster one, which an issue-bound machine
cannot do. No clock, no vendor table, and no microarchitectural assumption enters
this test.

### 3.2 Converting to instructions per cycle

Sustained core frequency was measured under an AVX-512 load rather than assumed —
`BenchmarkPeak/avx512` pinned to one core with `taskset`, that core's
`cpufreq/scaling_cur_freq` sampled five times during the run:

| host | CPU | sustained MHz | peak on the pinned core | **zmm FMA/cycle** |
|---|---|---|---|---|
| vesta.local | Ryzen 9 7950X3D (Zen 4) | 5040 | 160.7 GFLOP/s | **0.996** |
| janus.local | i9-9960X (Skylake-X) | 3472 | 216.0 GFLOP/s | **1.944** |
| antares.local | RYZEN AI MAX+ 395 (Zen 5) | 5139 | 328.2 GFLOP/s | **1.996** |

`FMA/cycle = (GFLOP/s ÷ 32) ÷ GHz`, since one `VFMADD213PS` on `Float32x16` is 16
lanes × 2 flops. The result is 1, 2, 2 to within 0.4% — so the double-pumped-vs-
full-width AVX-512 split that docs/hosts.md predicted is now a measured property
of these hosts rather than a citation, and `BenchmarkPeak` is measuring what it
claims to (issue #11).

The pinned peaks differ slightly from §1's (160.7 vs 165.6 on vesta) because
pinning to one core forgoes the scheduler's pick of the best-boosting core. The
two are not mixed: FMA/cycle uses the pinned peak and the pinned clock from the
same run, and the percent-of-peak fractions come from §1 where numerator and
denominator also share a run.

Then `insns/cycle = fraction_of_peak × FMA/cycle × insns/FMA`:

| host | FMA/cyc | insns/cycle on 2×32 | insns/cycle on 4×32 | |
|---|---|---|---|---|
| vesta.local | 0.996 | 4.26 | **6.02** | 4×32 issue-bound at ~6/cycle |
| janus.local | 1.944 | **4.15** | **4.28** | both shapes at ~4.2/cycle |
| antares.local | 1.996 | 4.90 | **8.00** | 4×32 issue-bound at ~8/cycle |

janus returns the same issue rate from two shapes whose instruction counts differ
by 35%. That is the signature of a front-end wall, and it is the non-circular half
of the argument: 4.15 comes from the 2×32 measurement, 4.28 from the 4×32
measurement, and they agree to 3%.

The bracketing numbers on the other two hosts — 6.02 and 8.00 instructions per
cycle — land on Zen 4's 6-wide and Zen 5's 8-wide dispatch, and janus's ~4.2 on
Skylake-X's 4-wide rename. Those widths are a cross-check on the derivation, not
an input to it.

### 3.3 Why 2 FMA/cycle is the hostile case

The instruction budget per FMA is `issue_width ÷ FMA_per_cycle`. A machine that
retires two 512-bit FMAs per cycle must feed them from the same front end, so it
has **half** the instruction budget per unit of arithmetic:

| host | issue width (derived) | FMA/cyc | insns/FMA at 100% of peak | insns/FMA at the 55% floor |
|---|---|---|---|---|
| vesta.local | ~6.0 | 1 | 6.0 | 10.9 |
| antares.local | ~8.0 | 2 | 4.0 | 7.3 |
| janus.local | ~4.2 | 2 | 2.1 | **3.88** |

The shipped kernel is at 4.625 instructions per FMA. vesta and antares have budget
to spare — that is why they clear at 96.6% and 64.1%. janus needs **≤3.88** and
gets 4.625, which predicts 46.1%. It measured 46.1%.

This also explains the otherwise-odd fact that the *slowest* machine of the three
in absolute GFLOP/s has the highest percent-of-peak: vesta's double-pumped
datapath asks for only one FMA per cycle, so a 6-instruction-per-FMA kernel
saturates it.

## 4. The floor is unreachable by source-level shaping on go1.26.5

The generator sweep audited **115 shapes** — `MR` × `NR` × unroll, in both the
scalar-broadcast form and the `Permute` form of the accumulate:

```
broadcast: 60 shapes, 30 zero-spill, best 4.625 insns/FMA  2x32 u=4 (74 insns / 16 FMA)
Permute  : 55 shapes, 31 zero-spill, best 4.438 insns/FMA  2x64 u=2 (71 insns / 16 FMA)
```

The shipped 2×32 ×4 *is* the broadcast form's optimum, across every shape the
allocator can hold without spilling.

**Nothing in the sweep reaches 3.88.** The best zero-spill shape in either form is
4.438, and that one is the `Permute` 2×64 which was dropped for the reasons in
KERNEL.md §3 (CSE merges identical `Permute` calls, and it would force the P3
packer to pad every A panel by 16 floats). Even adopting it, janus reaches
4.15 / (4.438 × 1.944) = **48.0%** — still short.

So this is not "we have not tried hard enough shapes". Every shape the register
allocator can hold without spilling costs more than 3.88 instructions per FMA, and
§5 says where those instructions go.

## 5. Where the 28 removable instructions are, and all of them are filed

> **Corrected 2026-08-11.** This section first credited two causes worth 16 of the
> 74 instructions (T9 anchor NOPs, T10 register copies) and concluded that both
> together clear the floor while either alone leaves janus at ~52%. That
> accounting was incomplete. Verifying the mechanism named in the ruling on #19
> found a third and larger lever — the FMA cannot take a broadcast memory
> multiplicand, though Go's assembler encodes exactly that instruction (T12,
> issue #20) — which accounts for 16 more instructions on its own. The corrected
> tables are below; §5.1 is the original pair, §5.2 the one that dominates them.
> The direction of the correction is worth stating plainly: the toolchain gap is
> ~1.75× larger than this report first claimed, not smaller.

### 5.1 The two register-pressure and debug-info costs

The 2×32 body is 74 instructions for 16 FMAs. Excerpt from
`spill-audit -func Kernel2x32 -v`, showing both overheads in the act:

```
00177 (GOROOT/simd/archsimd/slice_gen_amd64.go:291)  VMOVDQU64    192(SI), Z13
00184 (GOROOT/simd/archsimd/other_gen_amd64.go:265)  VBROADCASTSS X6, Z6
00190 (internal/vec/gemm_amd64.go:103)               VMOVDQU64    Z6, Z0      <-- T10: preserve the multiplicand
00196 (internal/vec/gemm_amd64.go:103)               VFMADD213PS  Z4, Z12, Z6
00202 (internal/vec/gemm_amd64.go:103)               VFMADD213PS  Z14, Z13, Z0
...
00072 (internal/vec/gemm_amd64.go:94)                XCHGL        AX, AX      <-- T9: statement-position anchor
```

The `VMOVDQU64 Z6, Z0` exists only because `VFMADD213PS` overwrites the operand it
is about to need again. Full histogram in KERNEL.md §3. The two blocks that are
toolchain overhead rather than arithmetic or addressing:

| block | count | cause | filed |
|---|---|---|---|
| `VMOVDQU64` register copies | 8 | only `VFMADD213PS512` exists, and it clobbers its first multiplicand, so `acc += a·b` cannot land in `acc` | T10 property 2, issue #18 |
| `XCHGL AX, AX` anchor NOPs | 8 | one per inlined archsimd call site whose instructions took the callee's source position | T9, issue #17 |

Sixteen instructions of 74 that do no work the algorithm asked for.

### 5.2 The operand that cannot be folded (T12, issue #20)

A hand-written SGEMM K-loop does not load and broadcast the B element at all. It
writes one instruction per FMA:

```asm
VFMADD231PS.BCST 12(SI), Z1, Z0     // zmm0 += zmm1 * broadcast(float32 at 12(SI))
```

Go's **assembler encodes this today**. Assembled with `go tool asm` and decoded
with `llvm-mc`, the seven bytes `62 f2 75 58 b8 46 03` are
`vfmadd231ps 12(%rsi){1to16}, %zmm1, %zmm0 ## zmm0 = (zmm1 * mem) + zmm0` — EVEX.512
with embedded broadcast, disp8 compression, accumulation in place, and memory as a
multiplicand. Every property the kernel wants, in one instruction.

The intrinsic layer cannot reach it, for three independent reasons (full repro in
`docs/toolchain-notes.md` T12):

1. Only 213-shaped FMA SSA ops exist for vectors, so the accumulation cannot land
   in the accumulator. (Scalar `VFMADD231SS/SD` *do* exist, and `math.FMA` uses
   them.)
2. The load-merge rule that does exist, `simdAMD64.rules:2774`, folds a memory
   operand into the 213 form's **addend** — which in a GEMM is the accumulator,
   the one operand that must stay in a register. The machinery is present but can only
   target the addend, so in a GEMM it can never fire usefully.
3. Nothing under `cmd/compile/internal/ssa/_gen/` ever emits the `.BCST` suffix,
   although `obj/x86/evex.go` fully supports it.

So this is one coherent root cause rather than a missing archsimd operation, and
it costs 16 instructions per 16 FMAs: 8 `VMOVSS` loads and 8 `VBROADCASTSS`. It
also subsumes T10's 8 register copies, since a 231-shaped FMA has nothing to
preserve.

### 5.3 Corrected projection

| body | insns | insns/FMA | projected janus % of peak |
|---|---|---|---|
| as shipped | 74 | 4.625 | **46.1% (measured)** |
| minus the 8 anchor NOPs (T9/#17) alone | 66 | 4.125 | 51.7% |
| minus the 8 register copies (T10/#18) alone | 66 | 4.125 | 51.7% |
| minus both | 58 | 3.625 | 58.8% |
| 231-shaped FMA, no embedded broadcast (#20 partly) | 66 | 4.125 | 51.7% |
| 231 + `.BCST`, anchors kept (#20) | 50 | 3.125 | 68.2% |
| **231 + `.BCST` + anchors halved (#17 + #20)** | **46** | **2.875** | **74.2%** |

Projected by holding janus's measured 4.145 instructions/cycle constant, which is
justified only because §3 established that janus is issue-bound; the same
projection would be meaningless for vesta or antares, where instruction count is
not what binds. It is a projection from a measured scaling law on that host, not a
measurement of a kernel that exists. (The independent roofline arithmetic agrees:
at `I = 2.875` the roofline is `2.250 ÷ 2.875 = 78.3%` of peak, and 74.2% is 94.8%
of it — the same attainment the kernel achieves today.)

**The corrected sharpest statement.** The embedded broadcast is the largest single
lever and dominates the pair this section originally identified: T9 + T10 together
reach 58.8%, while the operand fold alone reaches 68.2% and with T9 reaches ~74%.
The upstream case is one root cause plus one general debug-info issue, which
together account for roughly 2× of peak on this host — not two modest gaps that
happen to sum to the gate's margin.

## 6. What the ssa.html evidence does and does not show

DESIGN.md asks for the ssa.html evidence in this report. For *this* failure there
is nothing in it to show: the allocator spilled nothing, and the loop it produced
is the one the source asked for. The evidence for this failure is the instruction
count, which is §5.

The dumps do carry the evidence for the other P2 finding — DESIGN.md's own
12-accumulator tile, kept as `Kernel6x32` and never dispatched. (Written "32×6"
here and in DESIGN.md at the time; the doc was amended to the shipped orientation,
**MR=6, NR=32**, by ruling on #16 on 2026-08-16, so DESIGN.md's planned starting
tile and this spilling kernel are the same tile named consistently. Nothing
measured below changes — only the axis names it was recorded under.)

```
Kernel6x32: steady-state loop [152,1870] 270 insns for 48 arith (5.62 per arith):
  90 vector stack refs, 50 reg copies, 24 broadcasts, 8 anchor nops, 0 calls
```

90 spills per pass, and §1 measures what they cost: **30.5% of peak on vesta,
18.2% on janus, 14.7% on antares** — 3.2× to 4.4× slower than the shipped shape on
the same silicon. That is what the T10 register ceiling costs a kernel that ignores
it, and it is why the tile was shrunk.

Regenerate the dumps (~5–11 MB each, gitignored, because a stale copy would be
worse than none):

```
GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
  -pkg ./internal/vec -func Kernel2x32,Kernel4x32,Kernel6x32 -mode spill -ssa build/ssa
```

This requires `docs/toolchain-notes.md` T11's workaround to produce anything:
`GOSSAFUNC` is not in the build cache key, so a warm cache yields no file while
still printing `dumped SSA … to ./ssa.html`.

## 7. The decisions, and what is not being decided here

> **Resolved 2026-08-11 by the ruling on issue #19.** D1 was answered with
> *neither* reading, and D2/D3/D4 with "file upstream now, and gate against what
> the lowering can express". See §9; the list below is preserved as the state of
> the question when it was put.

DESIGN.md §4/P2 names three: "file upstream vs. avo fallback vs. accept AVX2
shapes". The scope question in §2 is a fourth, and it comes first.

- **D1 — scope of the floor.** Does "≥55% of measured peak" mean every AVX-512
  host, or one reference host? The gate implements the former. Answering "one
  reference host" makes it green today, at 96.6%.
- **D2 — file upstream and wait.** T9 (#17) and T10 (#18) are already written up
  with repros; §5 shows they jointly close the gap and that neither does alone.
  Go 1.27 is the natural horizon. This blocks P2 on someone else's release.
- **D3 — avo / assembly fallback for the microkernel.** Would close the gap
  immediately and costs the project its premise (pure Go on
  `GOEXPERIMENT=simd`). `CLAUDE.md` forbids taking this route unprompted, so it
  is listed, not started.
- **D4 — accept the finding as the result.** Record P2 as a documented NO-GO
  against the strict floor, restate the floor per datapath class — 1-FMA/cycle and
  2-FMA/cycle machines have instruction budgets differing by 2×, which a single
  percentage hides — and proceed to P3 with the constraint recorded. This is a
  change to the contract and should be made deliberately, not by adjusting a
  script.

A fifth option exists and is not recommended: dropping janus from the gate's host
set. It is the only machine in the fleet that exposes the constraint, which is an
argument for keeping it, not for removing it.

Not decided here, and not started: no shape was added or removed, no threshold
moved, no host dropped, and no assembly written.

## 8. Standing status (as of the RED gate, superseded by §9)

- `scripts/gate-p2.sh`: **RED**, on the janus.local percent-of-peak line only.
- Spill audit, call audit, bounds-check audit, peak-kernel register-only audit,
  correctness on three hosts, both builds and vet: **green**.
- P3: not started.

## 9. Resolution: the ruling on #19 and the amended floor

The decision was not to pick a scope for the 55% floor but to fix the floor's
denominator. Reference-host scoping is survivorship — it green-lights by ignoring
the one machine carrying information. Uniform every-host is gate-worship — it
demands the kernel beat the decode stage. The floor is instead expressed against
what the instruction set *as lowered* can deliver (DESIGN.md §4/P2, amended):

- **FMA-bound host → ≥55% of measured peak, unchanged.**
- **Issue-bound host → ≥90% of its issue roofline**, `maxᵢ(f_i·I_i) ÷ I`, with `I`
  taken from this report's own instruction counts. The host's FMA/cycle cancels, so
  no clock, `taskset` or `perf` counter enters the gate.

Under one written rule, all three hosts are green — not two rules and a wink:

| host | class | why | number | bar |
|---|---|---|---|---|
| vesta (Zen 4) | FMA-bound | ceiling mixes diverge 1.90× | 96.6% of peak | ≥55% |
| janus (Skylake-X) | issue-bound | ceiling mixes converge 1.023× over a 2.78× mix spread | 46.0% = **94.6% of a 48.6% roofline** | ≥90% |
| antares (Zen 5) | FMA-bound | ceiling mixes converge 1.091×, but the winning shape retires 158.5% of the roofline they imply, falsifying it | 62.3% net of CI | ≥55% |

Those are the amended gate's own numbers (`bash scripts/gate-p2.sh`, exit 0, log in
the closing comment on issue #3). They differ from §4's failing run by ≤0.5
percentage points on every host, in both directions.

Three properties keep this from being a weaker gate, and each is a fixture in
`scripts/roofline-test.sh` rather than a claim in prose:

1. **A host cannot classify itself issue-bound by being slow.** The naive test
   ("measured rate below what 55% requires") reduces to `f < 0.55`. The shipped
   test is convergence of structurally different mixes, which only a machine
   property produces; a merely-slow kernel yields divergent rates and faces 55%.
2. **A kernel cannot raise its own roofline by emitting more instructions.** A
   +40-instruction padded 2×32 scores 94.7% of its self-inflated roofline and is
   *refused*, because the shape must also be within 5% of the sweep's best
   zero-spill insns/FMA (KERNEL.md §3).
3. **The roofline test is not implied by the classifier that admits you to it.**
   The first implementation computed both over the same mix set, which makes
   `attain ≥ 1/cspread ≥ 0.909` an identity — above the 0.90 floor by
   construction, and on janus identical to it to four decimal places. The ceiling
   is therefore established by the mixes *other* than the shape under test.

Bounded leniency: because the peak kernel always pins the ceiling from beneath and
the shape guard caps the denominator, the amended floor never falls below
`0.90 × 2.25 ÷ 4.659 = 43.5%` of measured peak. And it **ratchets**: the floor is
`0.90 × maxᵢ p_i ÷ I`, monotone in `I`, so when #20 lands and `I` falls to ~2.875,
janus's required floor *rises* from today's 43.8% (`0.90 × 48.6%`) to 70.4% —
stricter than the 55% it replaced. The amendment needs no expiry clause because improving the
toolchain tightens it automatically.

**Standing task, and janus's new role.** keel issues #17, #18 and #20 are filed
upstream with keel as the minimal repro and §5.3 as the impact statement:

| upstream | what | keel note |
|---|---|---|
| [golang/go#80828](https://github.com/golang/go/issues/80828) | 512-bit values are allocated from 15 of the 32 zmm registers (`fpRegMaskAMD64` omits X16–X31 although the ops declare them) | T10 property 1, #18 |
| [golang/go#80829](https://github.com/golang/go/issues/80829) | no 231-shaped vector FMA, the load-merge rule folds the addend, nothing emits `.BCST` | T10 property 2 + T12, #18/#20 |
| [golang/go#80830](https://github.com/golang/go/issues/80830) | `BroadcastFloat32x16` is emulated; inlined non-intrinsic wrappers cost an anchor NOP per call site | T9, #17 |

When a
toolchain folds a broadcast memory operand into a 231-shaped FMA, re-audit and
re-measure janus expecting ~74%. Janus is now the fleet's regression sentinel: it
is the one host whose number moves when upstream lowering changes, which makes it
more useful red-then-roofline-green than it would have been quietly clearing 55%.

- `scripts/gate-p2.sh` under the amended model: see §1 of `CHANGELOG.md`
  `[Unreleased]` and the closing comment on issue #3 for the verbatim output.
- P3: unblocked.

## 10. Reopened 2026-08-18: the first evidentiary host, and the floor is out of reach on 2 FMA/cycle

§9 closed this report on three desktop hosts. The fleet is now AWS guests (§5 rule 5
amended, #66), and #104's ruling made *full-size instance of an approved family* the
evidentiary class. `c7i.48xlarge`, on-demand, is the first host to qualify. Both gates
are **RED** on it, so under `CLAUDE.md`'s P2 clause work stops here again.

`scripts/gate-p2.sh` and `scripts/gate-p3.sh` at `75bfaf5`, verbatim in the closing
comment on the campaign; log `build/campaign-spr-75bfaf5.log`.

### 10.1 The reading is silicon, not tenancy — #104 answered

| host | peak GFLOP/s | 4x32 GFLOP/s | 4x32 ÷ peak |
|---|---|---|---|
| `c7i.4xlarge` (partial, correctness class) | 240.2 | 82.06 | **34.16%** |
| `c7i.48xlarge` (full, evidentiary class) | 232.3 | 79.35 | **34.16%** |

The full-size host clocks 3.3% *lower* in absolute terms and lands on the same fraction
to three figures. A co-tenancy artifact cannot do that, because the numerator and the
denominator are measured in the same run on the same silicon and the ratio divides the
clock out. #104's hypothesis is refuted by the strongest test available to it: the
reading was always right, and what was missing was its evidentiary standing.

Peak is well measured here, which is what makes the ratio judgeable: the substitute
clock instrument read 232.2 / 232.3 / 232.4 across head, middle and tail, `+/- 0.0%`,
non-declining.

### 10.2 The two shipped shapes are limited by different resources

This is new, and it is why SPR is not admitted to §9's roofline branch. On SPR the best
shipped shape is **4x32 at 6.25 insns/FMA**, not janus's 2x32 at 4.625 — the shape with
*more* instructions per FMA is the faster one, inverting the objective §4's 115-shape
sweep minimises. The object code says why. 4x32's eight FMAs take eight distinct
addends, so its chains are independent:

```
00203  VFMADD213PS  Z7, Z12, Z8      <-- Z8 = Z12*Z8 + Z7
00209  VFMADD213PS  Z6, Z13, Z14
00221  VFMADD213PS  Z5, Z12, Z9      ... addends Z7 Z6 Z5 Z4 Z3 Z2 Z1 Z0, all distinct
00263  VFMADD213PS  Z0, Z13, Z2
00283  VMOVDQU64    Z2, Z0           <-- and then all eight are copied back,
00289  VMOVDQU64    Z11, Z1              because 213 leaves the sum in the
...                                      BROADCAST's register, not the accumulator's
00325  VMOVDQU64    Z8, Z7
```

2x32's do not — its addends are the previous FMA's results, a 4-long serial chain:

```
00134  VFMADD213PS  Z3, Z12, Z4      <-- writes Z4
00196  VFMADD213PS  Z4, Z12, Z6      <-- consumes it
00258  VFMADD213PS  Z6, Z5,  Z8      <-- consumes that
00334  VFMADD213PS  Z8, Z4,  Z3      <-- and that
```

So 4x32 is instruction-bound and 2x32 is **latency-bound**. On one FMA unit per cycle
that distinction is cheap and the two mixes converged on janus at 1.023×. On two units
it is not: the gate measured the ceiling mixes diverging **1.836×** (`[1.836x, 1.836x]`,
zero-width, wholly over the 1.10 bar), so the classifier calls the host `fma-bound` and
applies the flat 55%.

**It is right to.** Not because SPR is slow — §9's property 1 — but because neither
available ceiling mix can *demonstrate* a retirement ceiling here: one of the two is
limited by something else. Attainment against the roofline the formula would compute is
34.16 ÷ 36.00 = **94.9%**, under 1, so the issue-bound hypothesis is not falsified
either. It is undemonstrated for want of a suitable mix, which is a different thing
from false, and a different thing again from #86's noisy-peak straddle.

### 10.3 What clearing 55% on this host would require

At the peak loop's own audited retirement rate (2.25 insns per FMA at 232.3 GFLOP/s),
55% of peak needs **≤ 4.09 insns/FMA**. §4's sweep audited 115 shapes and its best
zero-spill result in either accumulate form is 4.438 (`Permute` 2x64) and 4.625
(broadcast 2x32). **No shape in the sweep clears 4.09**, and the two that come closest
are the latency-bound ones. Stripping every filed overhead from 4x32's 50 instructions,
in the order the causes are filed:

| strip | insns | insns/FMA | ceiling |
|---|---|---|---|
| as shipped | 50 | 6.25 | 36.0% |
| − 8 accumulator copies (golang/go#80829, in-place accumulate) | 42 | 5.25 | 42.9% |
| − 4 broadcast instructions, folded into the FMA as `.BCST` memory operands (golang/go#80829) | 38 | 4.75 | 47.4% |
| − 2 anchor NOPs (golang/go#80830) | 36 | 4.50 | 50.0% |
| − ~8 scalar slice-advance guards (**keel's own**, part 10.4 below) | 28 | 3.50 | 64.3% |

The three upstream fixes together land at 50.0% — **still short of 55%**. Only the
fourth lever crosses it, and that one is keel's. Stated the other way: a shape needs
≥ 8 independent chains *and* ≤ 4.09 insns/FMA simultaneously, no shipped shape has
both, and the generator sweep never optimised for chain length at all. Whether such a
shape fits is constrained by golang/go#80828 — 512-bit values come from 15 of the 32
zmm registers, and 8 accumulators + 2 B panels + 4 broadcasts is already 14.

These are projections from audited instruction counts under the retirement model in
part 10.2 above,
not measurements. What is measured is the 50 instructions, the 8 copies, and the 34.16%.

### 10.4 The one lever that is not upstream

`Kernel4x32`'s K-loop advances two slices per iteration (`gemm_amd64.go:182`,
`ap, bp = ap[4:], bp[32:]`). Each advance compiles to a branchless guard so the data
pointer never moves past the object — 8 scalar instructions per iteration for what an
index form does in two:

```
00120  ADDQ $-4, DI      00137  ADDQ $-32, R9
00124  MOVQ DI, DX       00141  MOVQ R9, R8
00127  NEGQ DX           00144  NEGQ R8
00130  SARQ $63, DX      00147  SARQ $63, R8
00134  ANDL $16, DX      00151  ANDL $128, R8
00269  ADDQ DX, BX       00272  ADDQ R8, SI
```

A counted loop indexing pre-sliced panels would remove these, but it risks
reintroducing the bounds checks KERNEL.md forbids in the K-loop — which is why the
slice-advance form was chosen. So it is real work with a real verification step
(`-gcflags=-d=ssa/check_bce`), not a free win.

### 10.5 P3's mission ratio, on the same host

DESIGN §1's headline criterion is also red, and against the *strongest* reference on
the host: gate-p3 swept six OpenBLAS coretypes and pinned `SkylakeX` at 202.30 GFLOP/s,
**+2.8% over** DYNAMIC_ARCH's own `SapphireRapids` choice (196.80), a 5.50 GFLOP/s
cross-family win against the sweep's own measured same-family drift of 3.30 (#35).

| | GFLOP/s | ÷ measured peak |
|---|---|---|
| OpenBLAS, pinned `SkylakeX`, 1 thread | 199.3 `+/- 0.0%` | 85.79% |
| keel `Sgemm` 2048³, `GOMAXPROCS=1` | 101.7 `+/- 1.0%` | 43.78% |

101.7 ÷ 199.3 = **51.03%**, against the ≥60% floor; 50.5% net of CI. OpenBLAS reaches
85.8% of the same measured peak, so the gap is not the blocking or the packing — it is
the microkernel's instruction count, the same cause as part 10.3 above.

One thing here is **not** load-bearing for either verdict, and gate-p3 already names it:
blocked `Sgemm` reaches 43.78% of peak while the 4x32 sweep entry reads 34.16%, which the
gate reports as *"retention: the blocked loop nest keeps 128% of its own
4x32/avx512/kc=128 microkernel … point estimates from two invocations — reported, never
judged"*. A routine cannot outrun the kernel it calls, so one of the two is not measuring
what its label says. The hypothesis to test first is that the sweep entry pays a
per-invocation cost the blocked nest amortises over many calls; it is falsified if the
sweep's rate is flat in the number of `kc=128` passes per timed iteration, since a fixed
prologue must dilute with more passes. Both numbers are below both floors, so no verdict
turns on it either way.
