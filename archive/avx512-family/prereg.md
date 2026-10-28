# Does avx512Peak under-read? The AVX-512 ceiling family (#199), before the run

`avx512Peak`/`avx512PeakD` have the structure that made `neonPeak` under-read by 1.26–1.89×
(#198): 12 FMAs that all read one multiplicand and one addend register. By audit they carry
12 anchor NOPs (27 insns per 12 FMAs). `internal/vec/peak_family_amd64.go` adds two members:
- **x44**: a 4x4 outer product over 8 distinct operand registers, in `avx512Peak`'s
  in-place form, at 19 insns per 16 FMAs, with no copies and no NOPs;
- **fo**: the shipped 4x32 kernel loop (4x16 at float64) with its loads and broadcasts
  hoisted, at 30 insns per 8 FMAs, 16 of them 213-form register copies. The 12-chain
  6x32 was tried first. Hoisted, it spilled 22 times (go1.27.3 allocates only 15 vector
  registers, T10/#18), so it cannot be a ceiling.

All three are register-only by `spill-audit -mode nomemory -goarch amd64`. The exact
witness test (`TestPeakFamilyWitness`) runs on each host before any timing, and a failure
stops that host. Exploration tier: c5n.4xlarge (Skylake-SP), c7a.2xlarge (Zen 4) and
c8a.2xlarge (Zen 5), pinned to core 2, 10 interleaved rounds.

**The reading, fixed now.** If a member's median beats the `avx512` row by more than 1% on
a host, `avx512Peak` under-reads there. The family-max rule (DESIGN.md P2, 2026-10-27) then
extends to amd64, and the instrument has changed on both architectures, which is what #200's
era question turns on. If none does, `avx512Peak` stands as a ceiling on these CPUs, and the
#200 era boundary is arm64-only.

**What the record already bounds.** The skx and zen4 float64 kernels reached 93.5% and 96.5%
of `avx512Peak` (`a4a2ca1`), so on those hosts a member can beat it by at most about 7% and
4%. Zen 5 (65.2%) is not bounded that way. I make no directional prediction beyond these
bounds.
