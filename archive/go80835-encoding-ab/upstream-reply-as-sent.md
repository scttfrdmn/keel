@mauri870 — on "should this be tracked here or in a separate issue": I think separate, and I have
a measurement that bears on your worry about it getting lost when those CLs land.

**Your case is a boundary problem; the one I reported here is an interior one.** My report was an
AVX-512 GEMM microkernel where the compiler's 128-bit spill idiom is legacy-SSE encoded *inside*
an all-EVEX loop — 36 non-VEX `movups`, e.g. `movups %xmm2,0x210(%rsp)`, interleaved among 88
`vfmadd213ps`, 177 `vmovdqu64`, 44 `vmovss` and 44 `vbroadcastss`. A `VZEROUPPER` at the function
boundary cannot reach that: the transitions, if they are being paid, are paid mid-loop inside one
basic block, and the dirty uppers are ZMM-wide rather than YMM-wide. So "insert `VZEROUPPER`
before returns" would be a complete fix for your repro and would not touch mine.

**And the two CLs linked here do not touch mine either — measured, not assumed.** I built two
toolchains that differ *only* by CL 825185 and CL 825186. They are consecutive (`0b2fd4aa9c08` is
exactly one commit ahead of `fbea197d3279`, and that commit is the second CL), so I pinned the
base to the later one and reverted both from it. Same source tree, same host (i9-9960X,
Skylake-X, bare metal), `-trimpath`, and an identical `VERSION` in both arms so the only
difference is the reverts.

| | both CLs | both reverted |
|---|---|---|
| `compile` binary sha256 | `2f5c402e…` | `5a2eecfb…` (toolchains differ) |
| disassembly sha256, filename header stripped | `428d3575…` | `428d3575…` (**identical**) |
| non-VEX `movups`/`movaps` in the kernel | 36 | 36 |
| non-VEX `movups`/`movaps`, whole test binary | 6416 | 6416 |

396,471 lines of disassembly, byte-identical; the revert deleted 136 lines across 4 files and the
compilers hash differently, so it applied — it just doesn't reach this code. **So the interior
case survives those CLs landing, which is the thing you were worried about, one shape over.** On
that basis #80835 looks like at least two sub-cases to me, and the one that stays open after these
CLs is not obviously the one they were filed against.

**Scope, so this isn't weighted more than it deserves.** The 36 are all in a 6×32 tile my library
*benchmarks but does not ship* — it spills by design and is held out of dispatch. The two shipped
AVX-512 kernels (2×32, 4×32) carry **zero** non-VEX moves. So this is a real instance of the
encoding, in a shape no user's code path reaches.

**I also have no timing, and I'd rather say so than imply otherwise.** My figures are static
instruction counts. I have Intel AVX-512 hardware but no counter-based measurement of AVX-SSE
transitions, so I have attributed nothing to them — and the A/B above is precisely why I still
can't: with no encoding delta there is no counterfactual to time. Two things in my own data also
argue against assuming the penalty dominates: the 65× in the original report is Emerald Rapids,
while my Skylake-X host pays about **1.9×** on the same code, which a transition-penalty story has
to explain and so far doesn't. One other data point, from a separate sweep: `GOAMD64` v1 through v4
leave these encodings unchanged (with the control that the same sweep *does* move the package
listing at v3), so for the interior case `GOAMD64` is not the knob either.

Happy to file the boundary issue if you'd rather not, though it's your repro and your finding, so
I'd suggest you keep it. If it helps, the write-up with the listings is `docs/spill-report.md`
section 11.3–11.5 in github.com/scttfrdmn/keel, and the A/B above is under
`archive/go80835-encoding-ab/` including the section that states which parts are unmeasured.
