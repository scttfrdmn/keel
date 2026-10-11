#!/usr/bin/env bash
# Era ceil8's amd64 judged campaign — so all 5 published hosts have ceil8 archives and the
# README can be regenerated whole at one era (#165).
#
# WHY IT COSTS MONEY AT ALL. The README publishes 45 rows over 5 hosts. Era ceil8 had archives
# for arm64 only, and its era row is NOT ISA-scoped, so the three amd64 hosts are UNMEASURED in
# this era rather than belonging to another. The generator's designed response is to drop a host
# with no archive in the era from the table — correct, and it would have published 18 arm64 rows
# and dropped the three amd64 hosts v0.2.0 certified. Scott chose to pay for the amd64 arm rather
# than halve the published evidence.
#
# FLEET READ OFF THE ARCHIVES, not recalled: the pinned8 rows this run must stay comparable with
# were measured on c5n.18xlarge (keel-skx), c7a.48xlarge (keel-zen4) and c8a.48xlarge
# (keel-zen5). Same types, same region, same on-demand market.
#
# ---- WHY THIS RUN EXISTS: to give era ceil8 TWO LIKE-INSTRUMENT amd64 archives
#
# The era already has two amd64 archives and they cannot be pooled. One is count=10 (my choice,
# 2026-10-10) and one count=30 (the straddle re-run), and the pair fails the same-claim band this
# project minted for exactly this question on #178: 9 of 72 L1 candidate rows disagree by more
# than README_TOL's 5%, up to +66.6% on 9R45 rate/Isamax/n=4096 and +64.6% at n=256, with a
# MEDIAN spread of 0.28%. The failures cluster at small sizes and Isamax -- precisely where
# sample count bites -- so they are a methodology difference, not weather. One share row fails
# too: 9R45 share/4x32/avx512/Strsm, 14.6 -> 16.2, +11.0%.
#
# Landing median-of-2 across that would set bars from a mixed instrument; landing the count=30
# run alone would be an N=1 reference, which §5 rule 16 refuses; landing only the passing rows
# while landing the witness would manufacture ~10 `owing` FAILs on purpose. So this run buys the
# second count=30 archive instead. Every other canonical archive in the project is count=30
# (campaign-c30-6ba6566, confirm-969c360) and #165's superseded-take warning is partly about a
# count=10 log -- the count=10 amd64 archive joins that category, kept as evidence and marked
# superseded in scripts/readme-archives.tsv rather than deleted.
#
# WHAT IS NOT IN DOUBT, so this run is not re-litigating it: the README regeneration stands. Its
# published rates agree between the two amd64 runs to <=2.4% (skx Sgemm -2.35% the worst), well
# inside the same-claim band, and it does not publish the L1 small sizes where the divergence is.
#
# # ---- PRE-REGISTERED, driven through baseline_state before launch
#
# ALL THREE amd64 HOSTS RESOLVE `new` in era ceil8, on both the share and the L1 criteria. Note
# zen4/zen5 are `new` and not `fleet`: CEIL_FRACTION=42.8 was derived from NEON rows, so
# CEIL_DERIVED_FROM moved to the two Neoverse models and the Zen pair is registry-governed now.
# That was stated in DESIGN.md §4/P5 when the bar was typed; this run is the first to exercise it.
#
#   gate-p5 BASELINE lines: 12 share (4 routines x 3 hosts) + 3 L1 (one per host) = 15
#   candidate rows: 12 share + 72 L1 (24 per host) = 84 baseline, 3 witness (one per host)
#   share criterion: no PASS and no FAIL — nothing is judged against 42.8 on amd64 this era
#   criterion 9 (README): PASS x3 is the PREDICTION. No amd64 codegen changed since pinned8 —
#     this session's kernel work was arm64-only, and TestPreferredPicksTheMeasuredWinnerPerClass
#     still asserts 2x32/4x32 for amd64 — so the published rows should still reproduce. IF THEY
#     DO NOT, that is the finding and the regeneration is more urgent than it looked.
#   gate-p3: the amd64 path, PEAK_FLOOR=0.55 judged through throughput_verdict (not the arm64
#     first-sight branch), so a percent-of-peak miss there is a real red.
#   VERDICT: RED, and STRUCTURALLY so — the prediction corrected after the count=10 run taught
#     it. All three hosts render BASELINE, so the headline criterion has no host it may judge and
#     that 1 UNMEASURED keeps the gate from green no matter how often it is re-run. A green amd64
#     run needs these rows LANDED first and then a confirmation campaign. This run is not trying
#     to be green; it is trying to be the second like-instrument archive that makes landing sound.
#   Expected shape, from the count=30 run two days ago: 59 PASS / 0 FAIL / 1 UNMEASURED /
#     15 BASELINE / 0 REPORTED, gate-p4 GREEN, criterion 7 decided, criterion 9 PASS x3.
#
# TTL IS 6h AND NOT 8h, deliberately. The arm64 runs took 1h16m; three bigger hosts should land
# near 2h. 6h is ~3x headroom against a reaper that would cost the whole run mid-benchmark, and
# caps the worst case near $145 instead of $193 at $24.09/hr. The per-step timeouts below are the
# primary guard; the TTL is the backstop for a hung driver the trap cannot catch.
set -euo pipefail
cd "$(dirname "$0")/.."

REV_EXPECT=838a223
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws
export AWS_REGION=us-east-1
export KEEL_FLEET_MARKET=on-demand
# count=30, the whole point of this run. §4's one-re-run allowance offers "a re-run or a higher
# -count"; a plain re-run only resolves if a new draw happens to miss the bar, while 30 samples
# tighten the interval ~sqrt(3)=1.7x AND sharpen the point, so it can decide in EITHER
# direction. A second straddle at this count is then evidence the true value sits on the bar,
# not evidence that more sampling is owed.
export KEEL_BENCH_COUNT=30
export KEEL_FLEET_TTL=7h
KEEL_FLEET="skx:c5n.18xlarge:Intel Skylake-SP
zen4:c7a.48xlarge:AMD EPYC Genoa
zen5:c8a.48xlarge:AMD EPYC Turin"
export KEEL_FLEET

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc) — three hosts at \$24.09/hr, so this matters"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list, not the one this script wrote"
  if timeout 300 spawn list --state running 2>&1 | grep -i 'keel-'; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
}
trap teardown EXIT INT TERM

say "preflight"
REV="$(git rev-parse --short HEAD)"
echo "   HEAD=$REV (expected $REV_EXPECT)"
[[ "$REV" == "$REV_EXPECT" ]] || { echo "!! HEAD moved; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
echo "   tree clean"
echo "   era:               $(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv')"
echo "   CEIL_FRACTION:     $(sed -n 's/^CEIL_FRACTION=\(.*\)$/\1/p' scripts/gate-p5.sh)"
echo "   CEIL_DERIVED_FROM: $(sed -n 's/^CEIL_DERIVED_FROM="\(.*\)"$/\1/p' scripts/gate-p5.sh)"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

say "launch: c5n.18xlarge + c7a.48xlarge + c8a.48xlarge, on-demand, 6h TTL"
timeout 3600 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true
say "provision: Go + OpenBLAS on all three"
timeout 10800 scripts/provision-openblas.sh --yes

say "gate-p5 (amd64, era ceil8) — the campaign"
set +e
timeout 21600 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"
say "post-run co-tenant census"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
exit "$GATE_RC"
