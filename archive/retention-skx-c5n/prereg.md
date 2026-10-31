<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Single-thread nest decomposition on Skylake-SP (c5n.18xlarge): predictions (#26, #142)

#142's re-derivation (2026-10-31) found that skx's nest deficit is entirely single-threaded: the
blocked nest keeps 77% of its 2x32 microkernel on keel-skx, against 95–96% on Zen. This run splits
that 77% into its parts on the host it was measured on.
- `scripts/retention.sh decompose`, then `feed`, unchanged.
- 2x32 and 4x32, n ∈ {1024, 2048}, `-count=10 -benchtime=1s`, `GOMAXPROCS=1`.
- One on-demand c5n.18xlarge, about $3.89/hr. The driver is `driver-frozen.sh`.

It replaces the janus attempt pre-registered on #26, which was discarded because a build-group
co-tenant shared janus throughout.

**Disclosed before the run:**
- The benchmark thread is not pinned. retention.sh has no affinity knob, and adding one would
  change the instrument. c5n.18xlarge has two sockets.
- There is no c5n decomposition from August, so the predictions on individual terms are about
  their ordering, not their magnitude. The baseline is gate-p3's c5n reading, 76–77% retention
  against a single kernel call, from the famaxx86 judged pair.

**Predictions:**
1. **Retention at 2x32/2048 against kernel-calls**, the nest's own call multiset, is 75–82%.
   - If it reads 90% or more, the 77% is a denominator artifact and #142's "single-thread nest
     deficit" is withdrawn.
2. **Retention by shape:** 4x32 retains more than 2x32 at both sizes, as on janus in August (87–88%
   against 78–80%).
3. **Feed:** panel feed is the largest non-pack term at 2x32 and at least as large as pack, as on
   janus in August. #188's VZEROUPPER fix shrank the pack-after-kernel cost, and nothing has touched
   the front-end cost of feeding the kernel.
   - If pack exceeds panel feed by 2x or more, the fix target is packing, not feed.
4. **The control run of the instrument passes:** the `padwaste` checks read 0, and the replayed
   depth multiset matches the nest's.

## Result (`d587114`, `run-d587114.log`)

29 minutes (about $1.90). The fleet was verified down by a readable inventory. Load at launch was 0.18.
**All four predictions held:**
1. **Retention:** 2x32/2048 retains **80.6%** of its kernel calls (80.2% net of CI).
2. **Shape:** 4x32 retains more than 2x32 at both sizes: 89.0% against 83.0% at n=1024, and 88.5%
   against 80.6% at n=2048.
3. **Feed:** at the shipped kc=384 (2x32), panel feed is **26.74 ms** of a 242.9 ms GEMM (11.0%),
   against C traffic 13.07 ms (5.4%) and pack 15.34 ms (6.3%).
4. **Controls:** padwaste 0.0000 in all four cases, and the depth multiset is the nest's.

**What the split says:**
- **Packing is small:** 3.0% of blocked time at 2x32/2048 (pack-a 1.9%, pack-b 1.1%).
- **The loop nest owns 14.1 of the 19.4 points given up.** feed resolves it into C traffic and panel
  feed, and their isolated steps overlap by up to 39.95 ns per call, so each is an upper bound.
- **Panel feed curves with KC:** +14.45, +32.60, +68.00, +101.65 ns per call over kc=128..512. The
  reused panel pair is 52,224 B at kc=384, past the 32 KB L1d, so the kernel feeds from L2.
- **gate-p3's 76–77% mixes two things.** Its denominator is the kernel at kc=128 (88.7 GFLOP/s).
  The kernel at the nest's kc=384 runs 84.6 GFLOP/s (ctiles=1, this run), because the same L1d
  overflow happens inside the kernel benchmark. So about 4 points of that 77% are the feed effect
  read as kernel speed.
- **One warning is in scope:** `full-transB` has no kernel denominator in every case. That is the
  instrument's existing behaviour, not this run's.
