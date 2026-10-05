#!/usr/bin/env bash
# ceil8 transition run — era ceil8's founding reading on the judged arm64 fleet (#177).
#
# ONE-OFF CAMPAIGN DRIVER. It lives in build/ while it runs and is archived with the
# results as EVIDENCE, per be6cca9's ruling that an archived driver is neither apparatus
# nor library. It is frozen from the moment detach.sh starts it: bash reads a script
# incrementally, so editing this file mid-run resumes the interpreter at a wrong byte
# offset (CLAUDE.md, "The tree stays frozen for a run's whole life").
#
# WHAT IT MEASURES AND WHAT IT CANNOT. CEIL_FRACTION is suspended to empty and era ceil8
# has no registered row for any host, so NOTHING here is judged: the two Neoverse hosts
# render BASELINE on all four share rows and propose candidates, and keel-skx/the Zen
# models are not in this fleet. This run REPORTS. A reviewed commit then types the bar
# from these rows, and the run after that is the first this era judges (§5 rule 17(d)).
#
# PRE-REGISTERED (DESIGN.md §4/P5, driven from baseline_state before launch):
#   keel-gvt3, keel-gvt4: `new` on share/3x24/neon/{Sgemm,Ssyrk,Ssymm,Strsm}
#     -> 8 BASELINE lines, 8 candidate baseline rows, 2 witness rows (one per host)
#     -> both hosts in the baseline-only bucket; no PASS and no FAIL on this criterion
#   criterion 9 (README): still RED on both hosts — the published rows are stale because
#     the code got faster, which this amendment does not touch
#   gate-p3 peak/3x24/neon/kc=128: `new` -> BASELINE, unchanged by this era boundary
#   The prior to check the proposals against (archive/arm64-3x24-draw2/README.md):
#     share Sgemm 53.4/53.5 and 73.5/73.5, Ssyrk 57.5/57.6 and 79.3/79.3,
#     Ssymm 53.2/53.2 and 73.2/73.1 — so a proposal far off those is the instrument,
#     not the silicon.
#
# -test.count=10, matching BOTH 3x24 draws rather than CLAUDE.md's count=30 note: the
# candidates this run proposes have to be poolable with those two draws as the stated
# prior, and a third draw at a different count is a different estimator.
set -euo pipefail
cd "$(dirname "$0")/.."

REV_EXPECT=119b673
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws
export AWS_REGION=us-east-1
export KEEL_FLEET_MARKET=on-demand
export KEEL_FLEET_TTL=8h
export KEEL_GOARCH=arm64
KEEL_FLEET="gvt3:c7g.16xlarge:ARM Neoverse-V1 (Graviton3)
gvt4:c8g.48xlarge:ARM Neoverse-V2 (Graviton4)"
export KEEL_FLEET

# ---- teardown on EVERY exit path, VERIFIED against the launcher's own list.
# Three independent reapers, because each covers a failure the others do not: this trap
# (normal exit, error, signal), spawn's --ttl 8h reaper (this whole shell dying or
# hanging), and the verification below (a `down` that reported success and left something
# billing — three of five instances kept billing once). A trap cannot catch a HANG, which
# is why every step below is timeout'd.
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
  echo "   (loop-l12-esc, if present, is another project's and is NOT ours to touch;"
  echo "    aws-fleet.sh down filters on startswith(\"keel-\"), verified before launch)"
}
trap teardown EXIT INT TERM

say "preflight"
REV="$(git rev-parse --short HEAD)"
echo "   HEAD=$REV (expected $REV_EXPECT)"
[[ "$REV" == "$REV_EXPECT" ]] || { echo "!! HEAD moved; refusing"; exit 2; }
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing (a dirty tree makes 'git archive HEAD' ship something nobody committed)"; exit 2; }
echo "   tree clean"
echo "   era: $(bash -c 'source scripts/gate-lib.sh; era_current scripts/measurement-eras.tsv')"
echo "   CEIL_FRACTION: '$(sed -n 's/^CEIL_FRACTION=\(.*\)$/\1/p' scripts/gate-p5.sh)' (empty = this run reports)"
say "co-tenant census at launch (recorded, per 'enumerate co-tenants before writing unexplained')"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

say "launch: aws-fleet.sh up"
timeout 2700 scripts/aws-fleet.sh up

say "fleet status"
timeout 600 scripts/aws-fleet.sh status || true

say "provision: Go + OpenBLAS on both hosts"
timeout 5400 scripts/provision-openblas.sh --yes

say "gate-p5 (arm64, era ceil8 transition) — the measurement"
set +e
timeout 18000 scripts/gate-p5.sh
GATE_RC=$?
set -e
say "gate-p5 exited $GATE_RC"

say "post-run co-tenant census"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

exit "$GATE_RC"
