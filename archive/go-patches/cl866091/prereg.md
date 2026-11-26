<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# CL 866091 (the relanded arm64 broadcast fold): predictions (#163, #197)

**Why.** golang/go#81352 is closed by **CL 866091**, master `544d1b015d`, merged 2026-11-25.
- CL 827904 merged on 2026-11-24 and was reverted by CL 866866, because a new `VDUP` rule shadowed
  the constant-broadcast rules.
- The reland changes the rule set: constant `VDUP` now folds to `VMOVI` in ARM64.rules.

So `archive/go-patches/cl827904/`'s A/B, taken on PS2 at a different base, does not transfer. This
is keel #163's trigger, and the go1.28 line will carry the commit (master is `goversion` 28).

**Already measured locally** (spill-audit `-goarch arm64`, this keel tree; parent `b22ece6bf5` →
fold `544d1b015d`):

| loop | instructions, parent → fold |
|---|---|
| shim `Kernel3x24` | 79 → **67** |
| shim `Kernel4x16` | 48 → **40** |
| `Kernel8x12` | 80 → 62, and spilled accumulators 5 → **3** |
| shipped `KernelPB5x16` | 213 → **213, byte-unchanged** |

The 8x12 change turns `internal/spill`'s `TestARM64SeesRealSpiller` red on master: it pins 5
spills, from docs/neon-sweep.md. That is attributed to this one commit. Under go1.27.3,
`KernelPB5x16` is 249 instructions, so master's pre-fold base already shrank it by 36. That change
is upstream of this commit and not the subject here.

**Run** (`driver.sh`). The two toolchains are one commit apart, and both arms use this keel tree.
- **Hosts:** c7g, c8g and c9g 4xlarge, at the exploration tier. The #197 guardrail applies: no
  patched-toolchain number becomes a published row.
- **Samples:** core 2, 10 rounds, with the arm order alternating.

**Predictions** (fold/base, medians):
1. **The control:** `LoadProbe/Ctl` (the shipped 5x16-pb kernel) and `LimiterProbe/PB` are within
   ±1% on every host, since their loops are unchanged.
2. **The broadcast kernels gain about what CL 827904 PS2 gained,** within ±5 points of its archived
   deltas:
   - raw 3x24 (`ctl`): V1 +26%, V2 +36%, V3 +32%;
   - shim 3x24: +29%, +27% and +15%.
3. **The nest follows.** Single-thread 2048³ Blocking for 3x24/neon and 4x16/neon gains ≥ 15% at
   both MCs on every host. The pre-broadcast kernels (5x16-pb, 4x20-pb, 7x12-pb) stay within ±1%.
4. **The pre-broadcast kernel still wins.** At its own shipped blocking (384/72), the best fold-built
   conventional nest stays ≥ 5% below 5x16-pb on every host, so nothing in keel's dispatch would
   change on adopting go1.28. If instead a conventional kernel reaches within 5% of 5x16-pb on some
   host, the 4×-A-panel pre-broadcast design is worth re-deciding on go1.28. That decision would be
   a #163 follow-up.
