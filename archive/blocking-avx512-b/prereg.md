<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# KCDeep on every AVX-512 host, stage B on the published rows: predictions (#26, #142)

**Why.** `archive/blocking-fma-sweep/` found Skylake-SP's KC lever on Sapphire Rapids: kc=1024 beats
the shipped 384 by +11.9% at 8T, or +11.1% net of both CIs. KC=1024 at the shipped MC was
non-negative, net of CI, on Zen 5, Zen 4 and GNR. archsimd exposes features, not CPU identity, and
SPR shares the FMA class with Zen. So the simplest rule is to set KCDeep = 1024 on every AVX-512 host:
no new fingerprint, and the issue class is unchanged. This run measures that rule on the rows it
would move.

**Run.**
- **arms:** main (`9bf3bdc`) against branch `exp-amd64-kcdeep`. The branch's `applyClassBlocking`
  keys KCDeep on `activeKern.Name == kern.AVX512` instead of on the issue class.
- **hosts:** c8a.2xlarge (Zen 5), c7a.2xlarge (Zen 4), c7i.4xlarge (SPR) and c8i.4xlarge (GNR).
- **benchmark:** BenchmarkScale Sgemm/Ssyrk/Ssymm/Strsm at n=4096, at 1 and 8 threads, masked to
  one thread per physical core, on stock go1.27.3.
- **samples:** 4 `-randlayout` seeds × `-test.count=5` per arm, with the arms alternating.
- **correctness first:** the branch's Level-3 suite runs on each host, with its exit status.

**Predictions** (new/old, net of both CIs):
1. **Correctness:** rc=0 on all four hosts.
2. **SPR:** Sgemm, Ssyrk and Ssymm gain +6% to +14% at both thread counts.
3. **Zen 5, Zen 4:** Sgemm, Ssyrk and Ssymm stay within [−1%, +3%].
4. **GNR:** the same rows stay within [−2%, +5%].
5. **Strsm, every host:** within ±2%. KCDeep needs m and n ≥ MC, so Strsm's MB=64 rank updates
   keep KC=384.

**Ship rule.** If no row on any host regresses beyond its CI, the rule ships for every AVX-512 host.
Zen 5 and Zen 4 are README hosts, so a judged pair then follows before their rows move.
If a Zen row regresses, the every-AVX-512 rule does not ship. Scoping the change to SPR would need a
new feature fingerprint, which is a separate decision.
