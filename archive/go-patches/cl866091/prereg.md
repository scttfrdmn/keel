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

## Result (`ba6e1a8`, `run-ba6e1a8.log`): the fold lifts every broadcast kernel 31–62%, and pre-broadcast still wins by 18–20%

Three hosts: c7g, c8g and c9g 4xlarge. Both toolchains passed the driver's identity check
(`go1.28-devel_b22ece6bf5` and `go1.28-devel_544d1b015d`). The run took 15m48s and produced 960
rows, all of them Blocking rows. The fleet was verified down by a readable inventory.

**The vec probes are UNMEASURED, not zero.** The driver's filter
`'^Benchmark(LimiterProbe/(shim|ctl|PB|FO)/|LoadProbe/Ctl/)'` puts `/` inside an alternation.
`go test -bench` splits its pattern on `/` into one regex per sub-benchmark level, so no vec
benchmark matched, and the host wrote no vec row. So predictions 1 (the kernel-level control) and 2
(the kernel deltas) have no reading. The nest-level pre-broadcast rows stand in as the control.

**Single-thread 2048³ Blocking at KC=384, median of 10 rounds, base → fold:**

| kernel (MC) | V1 | V2 | V3 |
|---|---|---|---|
| 3x24/neon (144) | 33.84 → 44.36, **+31.1%** | 37.96 → 51.47, **+35.6%** | 46.82 → 63.01, **+34.6%** |
| 4x16/neon (72) | 32.35 → 44.74, **+38.3%** | 34.50 → 48.94, **+41.9%** | 41.84 → 57.94, **+38.5%** |
| 8x8/neon (144) | 24.35 → 38.92, **+59.8%** | 27.05 → 42.88, **+58.5%** | 32.42 → 52.32, **+61.4%** |
| 5x16-pb (72), shipped | 56.49 → 56.10, −0.7% | 63.45 → 63.56, +0.2% | 77.04 → 76.86, −0.2% |
| 4x20-pb, 7x12-pb (both MCs) | −0.3% to +0.6% | −2.0% to +0.2% | −0.6% to +0.8% |

1. **Unmeasured at kernel level.** At nest level the pre-broadcast kernels, whose loops the fold
   leaves unchanged, move −2.0% to +0.8%. Only 5x16-pb is inside ±1% everywhere: two V2
   pre-broadcast cells (4x20-pb −2.0%, 7x12-pb −1.9%) sit outside it.
2. **Unmeasured** (the filter defect above).
3. ✓ **The nest follows.** Every broadcast-form kernel gains ≥ 30.7% on every host, against the
   predicted ≥ 15%. 8x8, with the most broadcasts per FMA, gains most: +58–62%. Not
   pre-registered: the gains exceed CL 827904 PS2's kernel-level +14.5–36%. These are nest
   readings, a different statistic, so they are not a contradiction of it.
4. ✓ **Pre-broadcast still wins.** The best fold-built conventional nest is 3x24/neon or 4x16/neon:
   - V1 44.74 against 5x16-pb's 56.10, −20.3%;
   - V2 51.47 against 63.56, −19.0%;
   - V3 63.01 against 76.86, −18.0%.

**Consequence for #163.** Adopting a go1.28 toolchain changes nothing in keel's arm64 dispatch: 5x16-pb
stays ahead by 18–20% at the nest. The fold closes about 40% of the gap between conventional and
pre-broadcast kernels (V2 3x24: 25.5 → 12.1 GFLOP/s behind), and the rest of the gap, plausibly the
remaining load and register pattern, stays. Adopting go1.28 does carry one known test change:
`TestARM64SeesRealSpiller`'s pinned 5 spills become 3.

## Kernel rerun (`driver-vec.sh`): predictions 1 and 2, re-registered before launch

Scott, 2026-11-25: "rerun - check what you are doing first this time". Before launch:
- **The filters were positive-controlled.** `^BenchmarkLimiterProbe$/^(shim|ctl|PB|FO)$` and
  `^BenchmarkLoadProbe$/^Ctl$` use one regex per sub-benchmark level. Both toolchains matched
  exactly the 7 intended rows on the dev host.
- **The driver's exact remote command string was smoked.** On linux/arm64 binaries, parsed by a
  remote shell on pollux through pueue's `build` group, it gave 7 of 7 rows per arm.
- **The driver now fails fast:** a host whose round 1 yields fewer than 7 rows per arm stops with
  `!!` rather than leaving a silent hole.

**The loops under these two toolchains** (spill-audit, parent → fold):

| loop | instructions, parent → fold |
|---|---|
| shim `Kernel3x24` | 79 → 67 |
| raw `KernelP3x24Ctl` | 48 → 42 |
| `KernelP3x24PB` | 39 → 39 |
| `KernelP3x24FO` | 24 → 24 |
| shipped `KernelPB5x16` | 213 → 213 |

The last three are byte-unchanged, so they are exact controls.

**Predictions** (fold/base, median of 10 rounds):
1. **Controls:** LimiterProbe `PB` and `FO`, and LoadProbe `Ctl` at all three depths, are within
   ±1% on every host.
2. **Broadcast loops:** `ctl` (raw 3x24) and `shim` gain within ±5 points of CL 827904 PS2's
   archived deltas:
   - `ctl`: V1 +26%, V2 +36%, V3 +32%;
   - `shim`: V1 +29%, V2 +27%, V3 +15%.

### Result (`ff9c4fe`, `run-vec-ff9c4fe.log`): controls exact, raw 3x24 as predicted, shim 3x24 well above prediction

The rerun took 7m01s and produced 140 rows per host: 7 × 2 arms × 10 rounds, the full count. The
fail-fast never fired, and the fleet was verified down by a readable inventory.

| probe (fold/base, median of 10) | V1 | V2 | V3 |
|---|---|---|---|
| shim 3x24 (`Kernel3x24`, 79 → 67 instructions) | 36.12 → 49.27, **+36.4%** | 40.51 → 56.51, **+39.5%** | 49.03 → 66.91, **+36.5%** |
| raw 3x24 (`ctl`, 48 → 42) | 36.43 → 45.88, +26.0% | 41.53 → 56.59, +36.3% | 49.10 → 63.88, +30.1% |
| `PB` (unchanged loop) | −0.0% | −0.0% | +0.1% |
| `FO` (unchanged loop) | +0.3% | +0.4% | +0.1% |
| shipped 5x16-pb, LoadProbe Ctl kc=128 / 384 / 512 | +0.0 / −0.5 / −0.1% | +0.0 / +0.2 / −0.1% | −0.0 / −0.1 / −0.1% |

1. ✓ **Controls, 21 of 21 within ±1%.** The largest is −0.5%, V1 Ctl kc=384, whose base arm alone
   spans 66.51–69.52.
2. **Split.**
   - ✓ Raw 3x24 matches CL 827904 PS2 within its band on all three: +26.0 / +36.3 / +30.1 against
     +26 / +36 / +32.
   - ✗ **The shim 3x24 gains far more than PS2 did:** +36.4 / +39.5 / +36.5 against +29 / +27 /
     +15. It is above the ±5 band on every host, by 2.4, 7.5 and 16.5 points.

   The relanded rule set does more for the shim's code than PS2 did. One plausible reading is that
   it also folds constant broadcasts to `VMOVI`. Under the fold, the shim (3x24/2) **overtakes** the
   raw 3x24/1 on V1 (49.27 against 45.88) and V3 (66.91 against 63.88). Under the base the order is
   the reverse. Why the gain is largest on V3 is unattributed.

The nest-level conclusion stands, from `run-ba6e1a8.log`: pre-broadcast 5x16-pb still leads the best
conventional kernel by 18–20% on every host.
