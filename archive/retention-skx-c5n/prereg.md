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
