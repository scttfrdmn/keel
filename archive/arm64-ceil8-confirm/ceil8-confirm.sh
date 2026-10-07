#!/usr/bin/env bash
# ceil8 CONFIRMATION run — the first run era ceil8 judges (#177, step 3 of §5 rule 17(d)).
#
# ONE-OFF CAMPAIGN DRIVER, archived with the results as evidence (be6cca9: an archived driver
# is neither apparatus nor library). Frozen from the moment detach.sh starts it — bash reads a
# script incrementally, so editing it mid-run resumes the interpreter at a wrong byte offset.
#
# WHAT IS DIFFERENT FROM THE TRANSITION RUN. CEIL_FRACTION=42.8 is in force and
# CEIL_DERIVED_FROM names the two Graviton models, so both hosts resolve `fleet` and the share
# criterion JUDGES for the first time this era. A bar derived from one run and enforced on the
# next is the only arrangement under which it can fail, which is the whole point of step 3.
#
# ---- PRE-REGISTERED, and at the tally level as well as the row level so it can be wrong
#
# Every state below was driven through baseline_state against the shipped artifacts before
# launch, not predicted from prose.
#
# TALLY:  51 PASS / 2 FAIL / 0 UNMEASURED / 2 BASELINE / 0 REPORTED  (55 lines)
#   derived from the transition run's 42/2/1/10 by moving the 8 share BASELINEs to PASS and the
#   1 headline-aggregate UNMEASURED to PASS, leaving the 2 L1 BASELINEs. If the arithmetic is
#   wrong the tally is wrong, which is the point of writing it down.
#
# VERDICT: RED, and it CANNOT be green. Criterion 9's two README rows are stale because the
#   code got faster (#136), this run does not touch them, and regenerating them is a separate
#   reviewed act that needs a judged log of this era -- which is what this run produces. So the
#   deliverable here is the SHARE CRITERION moving from unjudged to judged-and-clear, not a
#   green gate. A green would mean something unexpected happened.
#
# ROWS, share of each host's own measured 8-thread ceiling, net of CI, judged at >= 42.8:
#   keel-gvt3  Sgemm ~73.5   Ssyrk ~79.3   Ssymm ~73.2   Strsm ~61.2    (3 draws, +/-0.2 except
#   keel-gvt4  Sgemm ~53.3   Ssyrk ~57.4   Ssymm ~53.0   Strsm ~45.4     Strsm at +/-0.5)
#   ALL EIGHT EXPECTED TO PASS. keel-gvt4 Strsm is the argmin and clears by exactly the 2.6-point
#   margin, so it is the only row that can plausibly flip: >5.73% fall in its 8-thread rate, or
#   >6.07% rise in that host's measured ceiling. Those two differ because a share is a ratio.
#
# THE ONE ALTERNATIVE OUTCOME THAT IS NOT A FAILURE, pre-registered so it cannot be rationalised
# afterwards: rule 19 sits AHEAD of bar selection, so if keel-gvt4 Strsm's share interval exceeds
# BASELINE_MARGIN's 2.6 points that row renders REPORTED (noise-limited) instead of PASS, and the
# tally becomes 50/2/0/2/1. It is the widest of the eight -- 0.80 / 1.40 / 1.40 points across the
# three archived draws, all under the cap -- so this is unlikely but it is the live alternative.
#
# THE BASE IMAGE IS NOT THE SAME ONE, and that is noticed BEFORE the readings rather than after.
# spawn resolves the Ubuntu AMI live, and the dry run two minutes before this launch returned
# ami-0e1ab5c876cc030e8 where the 2026-10-05 transition run got ami-0bec8cef5313300ad. An AMI
# refresh can move the guest kernel, and the kernel can move a memory-bandwidth ceiling, which is
# the DENOMINATOR of every share this run judges. So: the transition run read
# `Linux 7.0.0-1013-aws` on both hosts, and if this run's read-back differs AND any share moves
# by more than the +/-0.2 (or Strsm's +/-0.5) the three archived draws support, the image is a
# CANDIDATE CAUSE and the delta is attributable to nothing until it is separated. Pre-registering
# it is the whole point: a confound named after a surprising number is a story, and one named
# before it is a control. Both the AMI id and the uname land in the archive either way.
#
# L1 AND peak/* RENDER BASELINE AGAIN, which is a DEBT and not a defect. Their witness rows were
# deliberately not landed (§5 rule 16 refuses a single-draw reference, and for share/* a landed
# row would have made both hosts `conflict` -> FAIL), so judged-runs.tsv has no ceil8 row and
# baseline_spent misses: `new`, BASELINE, candidates re-emitted. The gate prints it as a debt
# rather than absorbing it. 2 BASELINE lines in gate-p5 (one per host, L1) and 2 in gate-p3.
set -euo pipefail
cd "$(dirname "$0")/.."

REV_EXPECT=5e769f7
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws
export AWS_REGION=us-east-1
export KEEL_FLEET_MARKET=on-demand
export KEEL_FLEET_TTL=8h
export KEEL_GOARCH=arm64
KEEL_FLEET="gvt3:c7g.16xlarge:ARM Neoverse-V1 (Graviton3)
gvt4:c8g.48xlarge:ARM Neoverse-V2 (Graviton4)"
export KEEL_FLEET

# Three independent reapers, each covering what the others cannot: this trap (normal exit,
# error, signal), spawn's --ttl 8h reaper (this shell dying or HANGING, which a trap cannot
# catch), and a verification that re-reads the launcher's own list rather than trusting the
# exit status of `down`.
TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc)"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's list, not the one this script wrote"
  if timeout 300 spawn list --state running 2>&1 | grep -i 'keel-'; then
    echo "!! A keel- INSTANCE IS STILL RUNNING AND BILLING. Terminate it by hand."
  else
    echo "   no keel- instance in 'spawn list --state running' — fleet is down"
  fi
  echo "   (a foreign instance in that list belongs to another project and is NOT ours to"
  echo "    touch; aws-fleet.sh down filters on startswith(\"keel-\"), verified before launch)"
}
trap teardown EXIT INT TERM

say "preflight"
REV="$(git rev-parse --short HEAD)"
echo "   HEAD=$REV (expected $REV_EXPECT)"
[[ "$REV" == "$REV_EXPECT" ]] || { echo "!! HEAD moved; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing"; exit 2; }
echo "   tree clean"
echo "   era:              $(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv')"
echo "   CEIL_FRACTION:    '$(sed -n 's/^CEIL_FRACTION=\(.*\)$/\1/p' scripts/gate-p5.sh)' (non-empty = this run JUDGES)"
echo "   CEIL_DERIVED_FROM: $(sed -n 's/^CEIL_DERIVED_FROM="\(.*\)"$/\1/p' scripts/gate-p5.sh)"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

say "launch: aws-fleet.sh up"
timeout 2700 scripts/aws-fleet.sh up
say "fleet status"
timeout 600 scripts/aws-fleet.sh status || true
say "provision: Go + OpenBLAS on both hosts"
timeout 5400 scripts/provision-openblas.sh --yes

say "gate-p5 (arm64, era ceil8 CONFIRMATION) — the first run this era judges"
set +e
timeout 18000 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"

say "post-run co-tenant census"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true
exit "$GATE_RC"
