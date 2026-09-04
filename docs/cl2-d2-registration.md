<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# CL 2 / D2 registration: what the LICM hoist is worth in keel, as an upper bound

D1 (`docs/cl2-d1-803220-ps1-antares.log`, `#126`) confirmed the *correctness* defect in CL 803220
patch set 1. D2 is the timing complement the upstream issue does not have: **measure what hoisting
the loop-invariant SIMD broadcast out of a keel kernel is worth.** This file is written and
committed BEFORE the timing run, per the campaign discipline (rules 24/26) and the plan's own
statement that D2's outcome space is pre-registered (`docs/upstream-plan.md`, "CL 2's recon").

## Subject

`avx512Asum` (`internal/l1/l1_amd64.go:227`), reached by `keel.Sasum`, benchmarked by
`BenchmarkL1Sasum` (`bench/bench_test.go:290`). It is the kernel `#54` names: the abs sign-mask
(`vec.AbsMask512()` → `archsimd.BroadcastInt32x16(keepMaskI32)`) is loop-invariant and hoisted **by
hand** into `m` because the compiler will not lift it (`golang/go#79984`). `#54`'s retirement
condition is to drop that manual `m` "when CL 803220 lands".

## The mechanism is already confirmed, at the instruction level (go1.27.0)

Measured on the dev host, `GOEXPERIMENT=simd GOOS=linux GOARCH=amd64`, `go1.27.0`, `go build
-gcflags=-S`, `avx512Asum` only:

| arm | source | `VPBROADCASTD` in `avx512Asum` | placement |
|---|---|---|---|
| A — hoisted (current HEAD) | `m := AbsMask512()` once; `AbsWith512(load, m)` | **1** | preheader (offset 00031, before the loop back-edge) |
| B — inline | `Abs512(load)` = `AbsWith512(load, AbsMask512())` | **4** | one inside each loop/tail body |

So on the shipping toolchain the compiler does **not** hoist the inline broadcast: the natural form
recomputes it, once per block and once per main-loop iteration. **"A correct fix hoists nothing in
keel's shape" is therefore refuted in one of its two senses** — the instruction *is* there to hoist,
the manual `m` is doing real work the compiler leaves undone. The other sense (the hoist saves an
instruction but no *time*) is exactly what the timing below decides, and is registered as O2.

## What D2 measures

`scripts/ab-bench.sh`-style two-build A/B (via `ab_run`): arm A = HEAD (hand-hoisted), arm B = the
working tree with the inline patch (`archive/cl2-d2/inline-abs512.patch`), both cross-compiled by
`remote_build_test` (`-trimpath`, so a null A/B is byte-identical — the two-binaries hazard is
closed at the build). Benchmarked on **antares.local** (AMD Ryzen AI MAX+ 395, linux/amd64,
`GOAMD64=v1`, the D1 host), now through the `measured` pueue group. Reported quantity: `benchstat`
of B-over-A per size row, `ns/op` and the derived GFLOP/s, with the confidence interval.

**The delta (B slower than A) is an UPPER BOUND on a correct fix, never an estimate.** Two reasons,
both structural: (1) a certainty-based barrier — Goetz's `Block.CertainCPUfeatures` — can only
*refuse* hoists that patch set 1 permits, so a correct fix hoists ≤ what a full manual hoist does;
(2) keel's kernels sit behind a dispatch feature-check, and if a correct preheader cannot prove the
feature it has, a correct fix hoists **none** of this. So a nonzero D2 number bounds the prize; it
does not predict that CL 803220-done-right collects it. That question is not measurable today and is
out of D2's domain.

## Registered outcome space (before the run)

Per size row n ∈ {256, 4096, 65536, 1048576}, on the A→B benchstat delta:

- **O1 — a real ceiling.** B is slower than A, the CI excludes zero. The delta is the upper bound
  the hoist buys at that n. Reported as the ceiling; `#54`'s manual `m` is earning it.
- **O2 — the hoist saves an instruction but no time.** CI includes zero despite arm B's 4 broadcasts.
  The recomputation overlaps other work (a `VPBROADCASTD` is a register op off the load/reduce
  critical path). Then the ceiling is ≈0 at that n, and `#54`'s manual `m` could retire **now**,
  independent of the CL, with no measurable loss — which would be a rewrite of `#54`, not a checkbox.

**Per-row expectation, registered so it can be wrong.** Sasum is a streaming reduction: ~4 bytes read
per element, one AND and one add of compute. At **n=1048576** (4 MB, out of cache) it is
memory-bandwidth bound and the register-only broadcast should hide entirely → **O2 predicted**. At
**n=65536** (256 KB, ~L2) mixed. At **n=4096** (16 KB, L1-resident) and **n=256** (1 KB) the loop is
compute/latency bound and the per-iteration broadcast is most likely to show → **O1 most likely
here if anywhere.** n=256 is additionally tail-heavy (few main-loop iterations), so its broadcast
cost is spread across the one-shot tail blocks rather than the hot loop; it is reported but is the
weakest row for attributing to the main-loop hoist.

**Falsifier / the "worth a compiler fix" bar.** If every row is O2 (no row's CI excludes zero), D2's
ceiling is "negligible on this kernel" and the CL's value for keel is bounded near zero — a result
that reframes `#54` toward retire-the-workaround rather than await-the-fix. If any in-cache row is O1
with a delta ≥ 2% (CI-excluded), that row's figure is the ceiling reported to the CL thread.

## Method controls

- **Null A/B first** (BASE_REF=HEAD, clean tree): A-vs-A must read as a wash, or the harness itself
  is the confound (`ab.sh` handles the same-SHA naming; the reading is the positive control).
- **Assembly archived** for both arms (`-S` of `avx512Asum`), so the timing sits beside the 1-vs-4
  broadcast fact that motivates it (rule 11: the instrument adjudicates the reasoning).
- **go1.27.0**, the toolchain keel ships and freq150/D1 used — not the dev host's default go1.27.1.
- The working-tree patch is reverted after both binaries are built; the tree is frozen for the run's
  life, and the patch is tracked so the arm is reproducible.

## What D2 does not establish (§5 rule 12)

One kernel (`Sasum`), one host (antares), `GOAMD64=v1`, one toolchain. It bounds the prize for *this*
invariant on *this* kernel; it says nothing about the other keel sites with hand-hoisted broadcasts
(Axpy/Scal's alpha, the AVX2 Asum twin), nothing about whether a correct CL 803220 would collect any
of the ceiling behind keel's dispatch check, and nothing about the other `golang/go#79984` shapes D1
already scoped out. It is a timing upper bound, full stop.

---

## Result (2026-09-03, antares.local, go1.27.0, run `cl2-d2-361e872`)

Two builds, arm A = HEAD (`d0d46d26…`, hand-hoisted), arm B = the inline patch
(`1e177703…`), both `-trimpath`, `GOMAXPROCS=1` pinned to 8 cores one-per-domain, through
antares's `measured` pueue group. benchstat `-count=10`, B relative to A:

| n | working set | Δ sec/op (B vs A) | Δ GFLOP/s | outcome | predicted |
|---|---|---|---|---|---|
| 256 | 1 KB (L1) | **+13.52%** (p=0.000) | −11.91% | **O1** | O1-likely ✓ |
| 4096 | 16 KB (L1) | +1.21% (p=0.000) | −1.19% | O1 (small) | O1-likely ✓ |
| 65536 | 256 KB (~L2) | +4.48% (p=0.000) | −4.29% | O1 | mixed ✓ |
| 1048576 | 4 MB (DRAM) | **−0.92%** (p=0.000) | +0.91% | **O2** | O2 predicted ✓ |

**Ceiling: the hand-hoist is worth up to ≈13.5% at small in-cache vectors and nothing when
bandwidth-bound.** Every registered per-row expectation held: O2 at the 4 MB bandwidth-bound row
(the register-only broadcast is fully hidden — arm B is even a hair faster, a code-layout wash),
O1 at the in-cache rows, largest where the loop is shortest and most latency-bound. Two rows clear
the "worth a compiler fix" bar (≥2% CI-excluded): n=256 at 13.5% and n=65536 at 4.5%.

**Attribution audited at the instruction level (§5 rule 11), not inferred from the delta.** The two
binaries' `avx512Asum` (`archive/cl2-d2/asm-arm{A,B}-*.txt`) have **identical compute**: VADDPS 15=15,
VPANDD 9=9 (the abs-AND), VMOVDQU64 4=4 (the loads), VEXTRACTF64X4 2=2 (the reduction). The *only*
functional difference is mask materialization — VPBROADCASTD 1→4 and VMOVD 1→4, i.e. arm B rebuilds
the sign mask once per block instead of once per call. The extra XCHGL (25→40) is NOP alignment
padding, not work. So the delta is the hoist and nothing else.

**This is an upper bound, and the caveat is not rhetorical.** A correct CL 803220 recovers ≤ these
figures, and possibly none of them: keel's kernels sit behind an AVX-512 dispatch check, and Goetz's
own diagnosis is that a certainty barrier will refuse to hoist a feature the preheader cannot prove.
So the ceiling says the manual `m` is *earning up to 13.5%* today, not that the CL would.

**Consequence for `#54`.** Its retirement condition ("drop the Abs512 mask parameter when CL 803220
lands") is now a measured trade, not a checkbox: writing `avx512Asum` inline costs up to 13.5% at
n=256 unless the landed CL actually hoists in keel's dispatch shape. `#54` should be rewritten to
gate retirement on a re-run of this A/B against the *landed* toolchain showing the inline form
regains arm A's numbers — not on the CL merging.

## Posted upstream (2026-09-04)

Posted to CL 803220's thread on Scott's ruling (caveats first, framed as workload evidence, not
"what this CL buys keel"): comment `ed58a249_9b6527a0`, `/PATCHSET_LEVEL`, threaded as a reply to
D1's `9ffad9da_17ce3670`, `unresolved: true` — left unresolved so D1's still-open finding is not
cleared by the reply (the thread's state follows its last comment). Verified from Gerrit `/comments`
and `/detail`, not the POST response: messages 4 → 5, three PATCHSET_LEVEL comments, `updated
2026-09-04 16:02:21`. It completes the pairing D1 opened — the fault, then the reason it is worth
fixing — with both qualifiers ahead of the number.
