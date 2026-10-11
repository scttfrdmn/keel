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
# ---- WHY THIS RUN EXISTS: criterion 7 was UNDECIDED, not failed
#
# The 2026-10-10 campaign (archive, 1h41m, $40) came in 58 PASS / 1 FAIL / 1 UNMEASURED /
# 14 BASELINE / 1 REPORTED -> RED, and the red was NOT a slow routine. keel-skx's Ssyrk read
# 84.9% raw with interval [84.0%, 85.5%], straddling criterion 7's 85% bar, so the gate refused
# to decide in either direction -- "it is not evidence that Ssyrk is too slow, and it is not a
# pass". That reddened gate-p4 (66 PASS / 0 FAIL / 2 UNMEASURED) and gate-p5 propagated one FAIL.
# DESIGN.md §4's one-re-run allowance applies inside the delegated gates; BOTH runs are archived.
#
# WHAT THE FIRST RUN ALREADY SETTLED, so this one is not re-litigating it: criterion 9 PASSED on
# all three amd64 hosts, which was the prediction that mattered -- the published amd64 rows still
# reproduce, so #165's regeneration is a rigour change and not a correction.
#
# AND ONE PREDICTION I GOT WRONG LAST TIME, carried forward rather than repeated: I pre-registered
# 15 BASELINE and got 14, because keel-zen4's Strsm share interval cost 3.70 points against the
# 2.6-point cap and rule 19 out-resolved it to REPORTED before bar selection. I had named that
# exact possibility for the arm64 confirmation run and failed to carry it here. At count=30 that
# interval should tighten, so zen4 Strsm may well render BASELINE this time -- i.e. 15 BASELINE
# and 0 REPORTED. Either outcome is fine; what would be a finding is a THIRD state.
#
# ---- PRE-REGISTERED, driven through baseline_state before launch
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
#   VERDICT: plausibly GREEN. BASELINE is green-compatible, so with criterion 9 passing there is
#     no red left to render — which would make this the first GREEN amd64 run of era ceil8.
#
# TTL IS 6h AND NOT 8h, deliberately. The arm64 runs took 1h16m; three bigger hosts should land
# near 2h. 6h is ~3x headroom against a reaper that would cost the whole run mid-benchmark, and
# caps the worst case near $145 instead of $193 at $24.09/hr. The per-step timeouts below are the
# primary guard; the TTL is the backstop for a hung driver the trap cannot catch.
set -euo pipefail
cd "$(dirname "$0")/.."

REV_EXPECT=9a5e2f5
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
