<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# P2 go/no-go report — the microkernel is spill-free and still misses the floor on one host

Required by DESIGN.md §4/P2 and §7 rule 4 (`CLAUDE.md`, "P2 is a go/no-go, not a
hurdle"). `scripts/gate-p2.sh` is **RED**. Work stopped here; nothing in P3 has
been started, and no gate criterion was relaxed to change the colour.

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

## 5. Where the 16 removable instructions are, and both are already filed

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

Sixteen instructions of 74 that do no work the algorithm asked for — and janus
needs twelve of them gone:

| body | insns/FMA | projected janus % of peak |
|---|---|---|
| as shipped, 74/16 | 4.625 | 46.1% (measured) |
| minus the 8 anchor NOPs (T9) | 4.125 | 51.7% |
| minus the 8 register copies (T10) | 4.125 | 51.7% |
| **minus both** | **3.625** | **58.8% — clears the floor** |

Projected by holding janus's measured 4.15 instructions/cycle constant, which is
justified only because §3 established that janus is issue-bound; the same
projection would be meaningless for vesta or antares, where instruction count is
not what binds. It is a projection from a measured scaling law on that host, not a
measurement of a kernel that exists.

**Either fix alone leaves janus at ~52% and still red. Both together clear the
floor.** That is the sharpest statement this report can make, and it is the
upstream case: two independently-filed, individually-modest gaps in an
experimental package, which together account for exactly the distance between what
keel achieves and what its own gate demands.

## 6. What the ssa.html evidence does and does not show

DESIGN.md asks for the ssa.html evidence in this report. For *this* failure there
is nothing in it to show: the allocator spilled nothing, and the loop it produced
is the one the source asked for. The evidence for this failure is the instruction
count, which is §5.

The dumps do carry the evidence for the other P2 finding — DESIGN.md's own
12-accumulator 32×6 tile, kept as `Kernel6x32` and never dispatched:

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

## 8. Standing status

- `scripts/gate-p2.sh`: **RED**, on the janus.local percent-of-peak line only.
- Spill audit, call audit, bounds-check audit, peak-kernel register-only audit,
  correctness on three hosts, both builds and vet: **green**.
- P3: not started.
