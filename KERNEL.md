# KERNEL.md — the SGEMM microkernel: shape, register budget, and the spill frontier

This is the record P2 was run to produce (DESIGN.md §4/P2, issue #3). It says what
tile shapes keel's SGEMM microkernel uses, why those and not the ones DESIGN.md
specified, where the spill frontier is on `GOEXPERIMENT=simd` in go1.26.5, and
what it all measures.

Every number below is reproducible from the repo:

```
# compile-time properties (any host, no AVX-512 needed — it audits a cross-compile)
GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
  -pkg ./internal/vec -func Kernel2x32,Kernel4x32,Kernel6x32 -mode spill

# run-time properties (needs an AVX-512 host; see docs/hosts.md)
bash scripts/gate-p2.sh

# the kc sweep, which the gate does not run: it thresholds only kc=128, the value
# P3's blocking will use, and running the other four costs ~5 min per host
GOEXPERIMENT=simd go test ./bench -run XXX -bench 'Peak|Kernel' -count=10 -benchtime=1s
```

## 1. The shapes that ship

| shape | accumulators | unroll | live zmm | insns/FMA | loads/FMA | spills |
|-------|--------------|--------|----------|-----------|-----------|--------|
| 2×32  | 4            | 4      | 15       | 4.62      | 1.00      | 0      |
| 4×32  | 8            | 1      | 15       | 6.25      | 0.75      | 0      |

Both are `MR` rows × `NR` columns with the vectors along N: two 16-lane
`Float32x16` accumulators per row, the A operand arriving as a scalar broadcast.

Neither dominates the other, which is why both ship:

- **2×32 unroll 4** issues the fewest instructions per unit of arithmetic. The
  18 instructions of loop overhead (counters, compare, and the two panel
  re-slices) amortize over 16 FMAs instead of 8.
- **4×32 unroll 1** issues the fewest *loads* per unit of arithmetic. One pair of
  B-panel loads feeds four rows instead of two.

Which cost binds is a property of the host's front-end width and load ports, not
of the source. So the benchmark decides per host, §7 records the answer, and from
P3 on dispatch selects between them per host by the rule in §8.

`kern.Kernel` carries `MR`, `NR` and `Unroll` as fields rather than constants for
exactly this reason: the shape is a measurement result. Tests pack panels for
whatever shape they are handed, and the benchmark compares shapes against each
other in GFLOP/s.

## 2. Two departures from DESIGN.md, both forced

### 2.1 The tile is reflected: MR rows × NR columns

DESIGN.md §4/P2 specifies MR=32, NR=6 — vectors along M, six scalar columns. This
implementation exchanges M and N. The reason is that DESIGN.md §3 makes keel's
public API **row-major**, and in a row-major C, sixteen consecutive elements of a
*column* are `ldc` floats apart.

An M-vectorized tile would therefore have to write each accumulator lane to a
different cache line: 192 scalar stores per tile, each needing a lane extracted
from a vector register. archsimd has no cheap general lane extract for
`Float32x16`; the natural way to get lane *i* out is to store the vector and
reload one element — which is literally a spill. The M-vectorized tile would fail
P2's own spill audit by construction, for a reason that has nothing to do with the
compiler being audited.

Vectorizing along N makes every vector load and store — on the B panel and on C —
contiguous. The arithmetic, the accumulator count, and the register pressure the
phase was designed to test are unchanged. Filed as a discrepancy in the design
document (issue #16), not a departure from it.

### 2.2 There is no 12-accumulator shape, because it cannot be allocated

DESIGN.md budgeted 12 accumulators + 2 B vectors + 1 broadcast = 15 live zmm and
called that exactly full. Two properties of go1.26.5 say it is one register short
(`docs/toolchain-notes.md` T10, issue #18):

1. **The register allocator offers SIMD values only X0–X14.** `fpRegMaskAMD64` is
   `0x7FFF0000` — bits 16–30. X15 is the ABI zero register and is deliberately
   excluded; X16–X31 are bits 32–47 and appear in *no* allocatable mask at all,
   even though `VFMADD213PS512`'s register shape lists them as legal operands.
   Fifteen registers, not thirty-two.
2. **Only the `213` FMA form exists**, and `resultInArg0` is true on it, so it
   writes to its first multiplicand. An accumulate `acc += a·b` can therefore
   never land in `acc`'s own register, and needs one live scratch register beyond
   the working set — always.

12 + 2 + 1 scratch = 15, leaving nothing to hold the A broadcast. The tile spills,
and no amount of source rearrangement fixes it.

## 3. The spill frontier, measured

A generator swept 115 shapes (`MR` × `NR` × unroll) and audited each one's
steady-state loop. The frontier is at **8 accumulators**, and it is reached by
shrinking M rather than N — because N is where the vectors are, and each extra row
of M costs a broadcast and an accumulator pair.

| shape | acc | unroll | insns | FMAs | insns/FMA | loads/FMA | vector stack refs |
|-------|-----|--------|-------|------|-----------|-----------|-------------------|
| 2×32  | 4   | 4      | 74    | 16   | **4.62**  | 1.00      | 0                 |
| 3×32  | 6   | 2      | 60    | 12   | 5.00      | 0.83      | 0                 |
| 2×48  | 6   | 2      | 62    | 12   | 5.17      | 0.83      | 0                 |
| 1×64  | 4   | 4      | 86    | 16   | 5.38      | 1.25      | 0                 |
| 4×32  | 8   | 1      | 50    | 8    | 6.25      | **0.75**  | 0                 |
| 4×32  | 8   | 2      | 83    | 16   | 5.19      | 0.75      | **8**             |
| 2×64  | 8   | 2      | 85    | 16   | 5.31      | 0.75      | **6**             |
| 6×32  | 12  | 1      | 76    | 12   | 6.33      | 0.67      | **12**            |
| 6×32  | 12  | 4      | 270   | 48   | 5.62      | 0.67      | **90**            |

`insns` and `vector stack refs` are audited counts. `loads/FMA` is exact
arithmetic, not a measurement: with `MR` rows and `V` 16-lane vectors along N, one
pass reads `V·u` B vectors and `MR·u` A scalars for `MR·V·u` FMAs, so the ratio is
`1/MR + 1/V` and the unroll cancels out.

Three things in that table are worth stating explicitly.

**Unrolling raises peak register pressure, it does not lower it.** Go's SSA
scheduler hoists all of an unrolled body's independent loads to the top of the
body, so a ×2 unroll of 4×32 needs 8 hoisted A scalars where ×1 needs 4 — and that
is what pushes it over. The unroll factor is therefore a property of the shape, not
a free parameter: 2×32 tolerates ×4 (4 accumulators + 2 B + 8 scalars + 1 scratch =
15) and 4×32 tolerates only ×1 (8 + 2 + 4 + 1 = 15). Both sit exactly on the
ceiling.

**0.75 loads per FMA is a hard floor, and 4×32 ×1 sits exactly on it.** This one
follows from the arithmetic rather than from the sweep: getting `1/MR + 1/V` below
0.75 requires `MR ≥ 3` and `V ≥ 3`, hence `MR·V ≥ 9` accumulators — one more than
the frontier allows. So on go1.26.5 no zero-spill shape of any size can read less
than three quarters of a vector per FMA, and the sweep confirms it: every
lower-ratio shape in the table spills. That number, not the instruction count, is
what P3's memory hierarchy will have to feed.

**The two shipped shapes are the extremes, and the interior points are not
shipped.** 3×32 ×2 and 2×48 ×2 are clean and sit strictly between 2×32 and 4×32 on
both axes. They can only win if the optimum is interior rather than at an end,
which no model here predicts; shipping the two ends brackets them, and P3 can add
one if the measurements suggest the interior wins. Recorded so the omission is a
choice rather than an oversight.

**Widening N is not a cheaper way to get the same reuse.** 2×64 ×2 has 4×32 ×1's
load ratio and spills anyway: its four accumulator *pairs* are 8 accumulators, the
same as 4×32, but it needs 4 B vectors live instead of 2. Reuse along N costs
registers on the B side exactly as reuse along M costs them on the A side, and the
ceiling does not care which side spent them.

### The 2×32 instruction budget, in full

74 instructions per pass, 16 FMAs — the complete opcode histogram, not a model:

| what | count | opcodes |
|------|-------|---------|
| FMAs | 16 | `VFMADD213PS` |
| B-panel loads | 8 | `VMOVDQU64` from memory; two 16-lane loads per k-step |
| A scalar loads + broadcasts | 16 | `VMOVSS` then `VBROADCASTSS` — a broadcast from a slice element is **two** instructions, not one |
| register copies | 8 | `VMOVDQU64` register-to-register: the `213` form's clobbered multiplicand (T2) |
| loop overhead | 18 | 6 `ADDQ`, 2 each `MOVQ`/`NEGQ`/`SARQ`/`ANDL`/`CMPQ`, 2 branches |
| anchor NOPs | 8 | 1-byte `XCHGL AX, AX`, one per archsimd `Slice`-load call site (T9) |

The A operand costing two instructions per row per k-step is why the frontier moves
by shrinking M: each row of M buys one accumulator pair and costs a `VMOVSS`, a
`VBROADCASTSS`, and a live register, while each 16 columns of N buys an accumulator
pair and costs one shared load amortized over every row.

The `213`-form copies scale badly with pressure, which is most of why 4×32 looks
worse per instruction: 2×32 pays 8 copies for 16 FMAs (0.5 each), 4×32 pays **12
for 8** (1.5 each). At 8 live accumulators the allocator has no free register to
land an FMA result in and copies almost every one.

The loop overhead is the largest single non-arithmetic block, and it is the price of
bounds-check elimination. Both loop conditions are stated as slice *lengths*
(`len(bp) >= 128`) and both panels are re-sliced at the bottom of the body
(`ap, bp = ap[8:], bp[128:]`), because `len(bp) >= 128` is exactly the fact the
prover needs to know `bp[112:128]` is in range — and it is the loop condition, so
it holds by construction on entry. Indexing `bp[p*32+112]` from a counter instead
would need the prover to reason about a multiplication, which it does not do
reliably. Each re-slice compiles to five integer instructions: an `ADDQ` plus a
branchless `MOVQ`/`NEGQ`/`SARQ $63`/`ANDL` clamp that advances the pointer only if
the remainder is non-empty. Two panels, ten instructions, zero bounds checks.

### Two shaping attempts that were measured and dropped

- **`Permute` to dodge the copies.** `Float32x16.Permute` is bodyless with
  `resultInArg0: false`, so it can produce a fresh register without a copy. Swept
  across the same 115 shapes it moved 2×32 ×4 from 74 instructions to 73 (4.62 →
  4.56 insns/FMA, 16 copies → 10) and produced the sweep's single best zero-spill
  configuration, 2×64 ×2 at 71 instructions and 4.44 — a shape that *spills* in
  the broadcast form. But CSE merges identical `Permute` calls, defeating the
  trick exactly where it would matter most; it does not unlock any
  12-accumulator shape; and it would require the packer to pad every A panel by
  16 floats. A 1.4% instruction-count gain on the shipped shape — 4% if P3's
  packer also widens B to 64 columns — is not worth that coupling.
- **The bodyless array-pointer load.**
  `archsimd.LoadFloat32x16((*[16]float32)(bp[0:16]))` removes all 8 anchor NOPs —
  and introduces 12 vector stack references, for a net 78 instructions. Recorded
  in issue #17 with both measurements.

### The instruction the loop wants, and cannot ask for (T12, issue #20)

Rows 2–4 of the budget table above — 8 B-panel loads, 16 A load+broadcast pairs,
8 register copies — are 32 of the 74 instructions, and a hand-written K-loop emits
none of them. It writes one instruction per FMA:

```asm
VFMADD231PS.BCST 12(SI), Z1, Z0     // zmm0 += zmm1 * broadcast(float32 at 12(SI))
```

Go's **assembler already encodes this.** `go tool asm` plus `llvm-mc` confirm the
seven bytes `62 f2 75 58 b8 46 03` decode as
`vfmadd231ps 12(%rsi){1to16}, %zmm1, %zmm0` — EVEX.512, embedded broadcast, disp8
compression, accumulate in place, memory as a multiplicand. No shaping of the Go
source can reach it, for three reasons documented in `docs/toolchain-notes.md` T12:
only 213-shaped FMA SSA ops exist for vectors; the load-merge rule that does exist
folds memory into the 213 form's *addend*, which in a GEMM is the accumulator and
the one operand that must stay in a register; and nothing in the compiler's SSA
generators emits `.BCST`, though `obj/x86/evex.go` supports it.

This is not "archsimd is missing an operation" — `MulAdd` is exactly the right
source-level primitive. It is a lowering gap, and it is the largest term in the
budget: 74 → ~46 instructions, 4.625 → 2.875 insns/FMA. See
`docs/spill-report.md` §5.2–5.3 for the corrected accounting, which supersedes an
earlier version of that report crediting only T9 and T10.

## 4. Where the loop bodies live, and why

The K-loops are in `internal/vec/gemm_amd64.go`, not in `internal/kern`. CLAUDE.md
requires every `simd` import to be in `internal/vec`, so a kernel written in
`internal/kern` has to reach archsimd through one-line shims — and each level of
inlined wrapper *with a Go body* costs a 1-byte anchor NOP per call site, because
the generated instruction takes the wrapper's source position and the caller's
statement position then needs an anchor of its own (T9).

Measured on the 2×32 body, 16 FMAs per pass:

| layering | insns | anchor NOPs | insns/FMA |
|----------|-------|-------------|-----------|
| `kern` → `vec.Load512`/`vec.Broadcast512` → archsimd | 90 | 24 | 5.62 |
| `vec` → archsimd directly | 74 | 8 | 4.62 |

Two levels of wrapper, two NOPs per call site, sixteen call sites: 27% of the loop
body was anchor NOPs. Moving the loops one directory over removed one level and 16
instructions per pass, with no change to the rule and no archsimd import outside
`internal/vec`. `internal/kern` is now the shape registry, the tile protocol, and
the scalar reference.

## 5. The reference tile is kept, benchmarked, and never dispatched

`kern.ReferenceTile` is `Kernel6x32` — DESIGN.md's 12-accumulator tile, unrolled 4.
It spills 90 times per pass. It is exported, differentially tested, and
benchmarked, and it is deliberately **absent from `kern.Kernels()`**, so:

- nothing dispatches to it,
- the gate's zero-spill criterion stays binding on everything that ships,
- and the cost of the T10 register ceiling is a measured GFLOP/s number in the gate
  log rather than an assertion in a document.

It also carries no `InsnsPerFMA`, which since §8 is a second lock on the same door:
the shape ranking treats an unaudited count as unrankable, so there is no
arrangement of host classes under which a spilling tile could be selected. The
absence is deliberate for its own reason too — the gate re-derives that number from
the object code only for the shapes that ship, and a recorded measurement nothing
checks is a measurement that drifts.

`kern.Measured()` is `Kernels()` plus the reference tiles: what the benchmark runs.
The gate audits it too, non-fatally, and says so loudly if it ever *stops*
spilling — that would mean the register ceiling moved upstream and every shape here
should be revisited against the wider budget.

## 6. What the percent-of-peak number is, and is not

The P2 criterion is ≥55% of **measured** peak, where the denominator is
`BenchmarkPeak` — a register-only FMA-saturation loop with independent accumulator
chains and no memory traffic (issue #11, `internal/vec/peak.go`). Not a formula:
DESIGN.md's CPUID-and-clock formula is printed as a cross-check and diverges from
the measurement by ~2× on Zen 4, which is the double-pumped-AVX-512 datapath
showing up exactly where docs/hosts.md said it would.

The numerator is `BenchmarkKernel` on packed panels with no blocking around it. The
panels are packed once outside the timer and stay in L1 for the whole run. That is
what P2's criterion means — the question is what fraction of the FMA ceiling the
*K-loop* reaches, with packing and blocking excluded because those are P3's
subject — but it also means **this is an upper bound on what P3 can deliver**, and
it should be read as one. A number measured on cold panels would be a
memory-bandwidth measurement with a kernel attached.

Both numerator and denominator come out of the *same* benchmark invocation on each
host, so they share a frequency and thermal state. Both go through benchstat under
the §5.4 methodology (issue #14): `-count=10 -benchtime=1s`, medians, and the bar
counts as cleared only net of both confidence intervals.

The bar applies to the **best shipped shape per host**, since P3 will dispatch to
one of them; every shape's number is printed either way. It is not the best of N
hosts — every host that can run the kernel must clear it, and at least one must do
so under the `performance` governor.

## 7. Measured, per host — and both shapes win somewhere

`scripts/gate-p2.sh`, 2026-08-11, go1.26.5, `-count=10 -benchtime=1s`, numerator
and denominator from the same run on each host (issue #14). Percent of the host's
own measured `BenchmarkPeak`, never of a formula.

| host | CPU | zmm FMA/cyc | 2×32 ×4 | 4×32 ×1 | **winner** | 6×32 ref |
|---|---|---|---|---|---|---|
| vesta.local | Ryzen 9 7950X3D (Zen 4) | 1 | 152.9 GF/s — 92.4% | 159.9 GF/s — **96.6%** | 4×32 | 50.39 — 30.5% |
| janus.local | i9-9960X (Skylake-X) | 2 | 99.45 GF/s — **46.0%** | 75.98 GF/s — 35.2% | 2×32 | 39.2 — 18.1% |
| antares.local | RYZEN AI MAX+ 395 (Zen 5) | 2 | 174.0 GF/s — 53.1% | 210.5 GF/s — **64.2%** | 4×32 | 48.17 — 14.7% |

Two independent gate runs an hour apart agree to within 0.1 percentage point on
every cell (the earlier one read 46.1%, 64.1% and 30.5%), which is the
reproducibility claim §6 needs and the reason the winner column can be trusted.

**The winner flips, which is why both shapes ship.** §1 argued that neither
dominates and that the benchmark would have to decide per host; it did. The
load-lean 4×32 wins on vesta and antares by 4–11 percentage points, and the
instruction-lean 2×32 wins on janus by 11. A build that had picked one shape on
theory would have been wrong on at least one machine in this fleet — and the shape
that theory most favoured, 2×32 at the lowest instructions per FMA, is the one that
wins on the *fewest* hosts.

The `zmm FMA/cyc` column is measured, not looked up: `BenchmarkPeak/avx512` pinned
to one core with `taskset` while sampling that core's `cpufreq/scaling_cur_freq`
gives 0.996, 1.944 and 1.996 — Zen 4 double-pumping AVX-512 over 256-bit
datapaths, and both the Skylake-X and Zen 5 parts retiring two full-width FMAs per
cycle. It is the column that explains the rest of the table: a 2-FMA/cycle machine
must feed twice the arithmetic from the same front end, so it has half the
instruction budget per FMA, and instruction count starts to bind where it did not
before (docs/spill-report.md §3.3).

### One host is issue-bound, and the floor is expressed accordingly

**janus.local reaches 46.0% of its measured peak.** A flat 55% floor is unreachable
there by any arrangement of this source: janus is limited by instruction issue
rather than by spills (its two shapes' throughputs stand in the inverse ratio of
their instruction counts, 1.308 measured against 1.351 predicted, and both derive
the same ~4.2 instructions per cycle), it would need ≤3.88 instructions per FMA,
and no zero-spill shape in the 115-shape sweep is below 4.438. Its **issue
roofline is 48.6% of peak**, so the kernel is at 94.6% of what the instruction set
as-lowered can express.

That was a RED gate, and P2 is a go/no-go, so work stopped there and
`docs/spill-report.md` carries the evidence. The ruling on issue #19 then amended
the floor's *denominator* rather than its value (DESIGN.md §4/P2): FMA-bound hosts
keep ≥55% of measured peak; an issue-bound host is held to ≥90% of its issue
roofline, computed from the spill audit's own instruction counts. Under that one
rule all three hosts are green — vesta and antares FMA-bound at 96.6% and 62.3%
net of CI, janus issue-bound at 94.6% of 48.6%. The decision function is
`scripts/roofline.sh` and its adversarial fixtures are `scripts/roofline-test.sh`,
which run before any benchmarking in the gate.

The cause is filed, not absorbed: 8 anchor NOPs (T9/#17), 8 register copies
(T10/#18), and — the largest term, found while verifying the ruling — the
unreachable broadcast-memory FMA operand (T12/#20), together ~28 of the body's 74
instructions. When that lands, `I` falls to ~2.875 and janus's *required* floor
rises to 70.4%, above the 55% it replaced; see `docs/spill-report.md` §9 on why the
amendment ratchets rather than expires.

The two Zen hosts clear it comfortably. The reference tile's 14.7–30.5% is on the
same silicon with the same harness, which is the control: the gap between 30.5%
and 96.6% on vesta is what the register ceiling costs, and it is much larger than
the gap between the two shipped shapes anywhere.

## 8. Which shape ships on which host (P3, ruling on issue #24)

§7 measured the winner and left the choice to a human reading the table. P3 makes
it automatic, because a table nobody consults is a table the library disagrees
with: `Sgemm` took the first entry of `internal/kern`'s registry and therefore ran
4×32 on all three hosts, including the one where 2×32 measures 11 percentage
points faster. That was issue #24, found by the gate's own anti-vacuity shape
guard rather than by a benchmark.

**The rule.** Dispatch classifies the host with the same two-way classification
P2's throughput model already defines, and picks the shape that is extremal on
that class's binding cost:

| class | what binds the K-loop | shape rule | winner today |
|---|---|---|---|
| FMA-bound | arithmetic throughput; instructions are cheap | fewest **memory ops per FMA** | 4×32 ×1 — 0.75 |
| issue-bound | the front end retires a fixed number of instructions per cycle | fewest **instructions per FMA** | 2×32 ×4 — 4.625 |

so vesta and antares run 4×32 and janus runs 2×32 — in every case the shape §7
measured as that host's winner. That agreement is not a coincidence and it is not
a fit: on janus the two shapes' throughputs stand in the inverse ratio of their
instruction counts (1.308 measured, 1.351 predicted, §7), which is what
"issue-bound" means quantitatively. The rule is the physics, and the measurement
is the check on it — `scripts/gate-p3.sh` criterion 5b benchmarks both shapes in
one invocation on every host and fails if the passed-over one is faster net of its
confidence interval.

Per-target microkernel shapes are what BLIS and OpenBLAS both carry. "One shape
everywhere" was never a design principle here either — only the skeleton's
simplicity.

**How the class is decided, and its two error directions.** `archsimd` reports CPU
features and no vendor, family or model, and `internal/cpu` discards the CPUID
signature word, so the microarchitecture is unreachable from pure Go on this
toolchain (`docs/toolchain-notes.md` T14, issue #25). `kern.HostClass` therefore
fingerprints the generation from a feature bundle: AVX-512 **with**
AVX512_VBMI2 + AVX512_VPOPCNTDQ is Ice Lake / Zen 4 or later and treated as
FMA-bound; AVX-512 **without** them is the Skylake-X / Cascade Lake / Cooper Lake
generation and treated as issue-bound. A proxy that decides what ships has to be
checked against something measured, so the library prints its classification and
its grounds and the gate compares them against the class its own convergence test
derives from the measurements. `internal/kern/class_amd64.go` documents why
neither error direction can make a gate lenient; `KEEL_KERN_CLASS=fma|issue`
overrides the fingerprint, which is how the gate measures the shape dispatch did
*not* choose.

### janus's sentinel number is 2×32's, and the roofline moves with the shape

**Read this once, loudly, because the number looks like a property of the host and
is not.** The issue roofline is `max_i(f_i·I_i) / I_b` — it divides by `I_b`, the
instructions per FMA of *the shape being judged*. So janus's **48.6% is 2×32's
roofline**, not janus's. Under the other shape the same host's roofline would be

```
janus's ceiling mixes:  4×32  0.352 × 6.250 = 2.200
                        peak  1.000 × 2.250 = 2.250   <- pmax
judging 2×32:  2.250 / 4.625 = 48.6% of peak,  46.0% measured = 94.6% of it
judging 4×32:  2.250 / 6.250 = 36.0% of peak,  35.2% measured = 97.8% of it
```

which is to say a *fatter* kernel would have cleared the ≥90%-of-roofline floor
more comfortably than the fast one does. That is the vacuity P2's shape guard
exists to refuse, and it does: 6.25 insns/FMA is outside `sweep_best 4.438 × slack
1.05 = 4.659`, so 4×32 is granted no roofline at all and faces the unmodified bar.

What P3 changes is which shape that number describes. Through P2 the sentinel
judged whichever shipped shape measured fastest, while `Sgemm` dispatched to the
registry's first entry — so on janus the quoted 48.6% belonged to a shape the
library did not run. The gate now judges the **dispatched** shape, and criterion 5b
requires the library's classification to match the gate's measured one, so the two
can no longer diverge. The consequence for reading §7: janus's 46.0% / 94.6% of
48.6% is the shipped configuration's number as of P3, and it was already 2×32's
arithmetic before dispatch agreed — the figures do not move, their subject does.

The corollary is that a future shape change re-bases the floor rather than
inheriting it. Anything that lowers a shipped shape's `I_b` *raises* the roofline
it must clear (§7's last paragraph: at `I` ≈ 2.875 janus would owe 70.4%), and
anything that raises `I_b` lowers the roofline until the shape guard refuses it
one. Neither direction is a number to carry over from a previous run.
