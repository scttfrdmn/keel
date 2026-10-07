#!/usr/bin/env bash
# Record the project's FIRST arm64 KEEL_REPLAY corpus (#178, #167).
#
# WHY A CORPUS AND NOT JUST A FIX. gate-p3's arm64 peak criterion prints "this is its CANDIDATE
# BASELINE and this gate PROPOSES NO ROW FOR IT" and writes nothing; emitting a real candidate
# needs four pieces of state gate-p3 lacks (a rev, a run stamp, a candidates path, a per-host
# archive path), introduced under `set -euo pipefail` into a gate that runs inside every judged
# chain. Both tracked corpora in archive/witness/ are avx512, so that path could not be exercised
# anywhere -- which is also why #167 recorded its arm64 rendering as having "no standing harness".
# Recording one corpus here buys a $0 exercise path for this fix and for every future arm64 gate
# edit, instead of putting the risk on a $10/hr measurement run.
#
# WHY GRAVITON2 AND NOT A JUDGED PART. The criterion's first-sight arm is the one that needs
# exercising, and it fires only for a cpu_model with no registered row. Graviton3/4 are
# Neoverse-V1/V2 and BOTH now carry a peak/3x24/neon/kc=128 row at era ceil8, so either would
# render `registered` and exercise nothing. c6g is Graviton2 = Neoverse-N1, unregistered, and
# driven through baseline_state before launch: `new` -> peak_bucket `first-sight`. It is in
# OPENBLAS_OK_CORES (neoversen1), so the OpenBLAS leg is supported too.
#
# THIS HOST IS NOT EVIDENTIARY AND NOTHING HERE IS A RESULT. A c6g.2xlarge is a partial-size
# shared guest; remote.sh classifies it `correctness`. Its NUMBERS are not a measurement and no
# row from this run may ever be landed. What a corpus needs is FIXED input, not good input: two
# replays of one corpus render identically, which is the whole mechanism (scripts/gate-replay.sh).
set -euo pipefail
cd "$(dirname "$0")/.."

REV_EXPECT=0b6b649
CORPUS=build/witness-corpus-arm64
say() { printf '\n========== %s  (%s) ==========\n' "$*" "$(date -u +%H:%M:%SZ)"; }

export AWS_PROFILE=aws
export AWS_REGION=us-east-1
export KEEL_FLEET_MARKET=on-demand
export KEEL_FLEET_TTL=2h
export KEEL_GOARCH=arm64
KEEL_FLEET="gvt2:c6g.2xlarge:ARM Neoverse-N1 (Graviton2)"
export KEEL_FLEET

TORN=0
teardown() {
  local rc=$?
  [[ "$TORN" -eq 0 ]] || return 0
  TORN=1
  say "TEARDOWN (driver rc=$rc)"
  timeout 900 scripts/aws-fleet.sh down || echo "!! 'down' FAILED — read 'spawn list' below"
  say "teardown verification — the LAUNCHER's own list"
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
[[ -z "$(git status --porcelain)" ]] || { echo "!! tree DIRTY; refusing (gate-p3 does git archive HEAD)"; exit 2; }
echo "   tree clean"
echo "   corpus dir: $CORPUS"
rm -rf "$CORPUS"
say "co-tenant census at launch"
timeout 300 spawn list --state running 2>&1 | sed 's/^/   /' || true

say "launch: one c6g.2xlarge (Graviton2 / Neoverse-N1)"
timeout 2700 scripts/aws-fleet.sh up
timeout 600 scripts/aws-fleet.sh status || true

say "provision: Go + OpenBLAS"
timeout 7200 scripts/provision-openblas.sh --yes

say "RECORD the corpus: gate-p3 standalone, KEEL_GOARCH=arm64"
set +e
KEEL_REPLAY=record KEEL_REPLAY_DIR="$CORPUS" timeout 10800 bash scripts/gate-p3.sh
GATE_RC=$?
set -e
say "gate-p3 exited $GATE_RC (a RED gate still records a usable corpus; what matters is the calls)"

say "corpus contents"
if [[ -d "$CORPUS" ]]; then
  echo "   files: $(find "$CORPUS" -type f | grep -c . || true)"
  echo "   bytes: $(du -sk "$CORPUS" | awk '{print $1"K"}')"
  find "$CORPUS" -type f -name '*.cmd' | head -5 | while read -r f; do printf '   cmd: %s\n' "$(head -c 160 "$f")"; done
else
  echo "   !! NO CORPUS DIRECTORY — recording did not happen"
fi

say "did the first-sight arm fire? (the whole point)"
grep -c 'PROPOSES NO ROW FOR IT' "$CORPUS"/../gate-p3-*.log 2>/dev/null || true
exit "$GATE_RC"
