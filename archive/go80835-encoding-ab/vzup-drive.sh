#!/usr/bin/env bash
# Local driver for the golang/go#80835 encoding-cost A/B on janus (keel #144).
#
# Build phase -> janus pueue `build` group (unmeasured: clones, two Go toolchains, objdump).
# Timing phase -> janus pueue `measured` group, the single measurement slot on that box, because
# its numbers are results and nothing measured runs outside the queue.
#
# THE WITNESS GATES THE SPEND OF THE MEASURED SLOT. scripts/labrun propagates the remote exit
# code, and the remote build phase exits 3 on a null treatment, so `&&` between the phases means
# a revert that did not change the encoding never reaches the timing phase at all.
set -euo pipefail
cd "$(dirname "$0")/.."

HOST=janus.local
REV_EXPECT=e4b6c6e
say() { printf '\n########## %s  (%s) ##########\n' "$*" "$(date -u +%H:%M:%SZ)"; }

say "preflight"
REV="$(git rev-parse --short HEAD)"
echo "   keel HEAD=$REV (expected $REV_EXPECT)"
[ "$REV" = "$REV_EXPECT" ] || { echo "!! HEAD moved; refusing"; exit 2; }
[ -z "$(git status --porcelain)" ] || { echo "!! tree DIRTY; refusing — both arms must build one source"; exit 2; }
echo "   tree clean"

say "ship keel source + the phase script"
ssh -o BatchMode=yes "$HOST" 'mkdir -p ~/vzup'
# git archive HEAD, the same frozen-source discipline the gates use: never the working tree.
git archive --format=tar HEAD | ssh -o BatchMode=yes "$HOST" 'cat > ~/vzup/keel-src.tar'
printf '%s\n' "$REV" | ssh -o BatchMode=yes "$HOST" 'cat > ~/vzup/keel-rev-tmp'
ssh -o BatchMode=yes "$HOST" 'mkdir -p ~/vzup/keel && mv ~/vzup/keel-rev-tmp ~/vzup/keel/.keel-rev'
scp -q -o BatchMode=yes build/vzup-phase.sh "$HOST:~/vzup/vzup-phase.sh"
ssh -o BatchMode=yes "$HOST" 'chmod +x ~/vzup/vzup-phase.sh'
echo "   shipped keel $REV and the phase script"

say "co-tenant census on $HOST before anything"
ssh -o BatchMode=yes "$HOST" 'uptime; pueue status 2>/dev/null | grep -cE "^ [0-9]+ +(Running|Queued)" | sed "s/^/  active pueue tasks: /"' || true

say "PHASE 1 (build group): two toolchains + the treatment-arrival witness"
LABRUN_DIR=/home/scttfrdmn/vzup scripts/labrun "$HOST" build -- ./vzup-phase.sh build

say "PHASE 2 (measured group): time both arms"
LABRUN_DIR=/home/scttfrdmn/vzup scripts/labrun "$HOST" measured -- ./vzup-phase.sh time

say "collect"
mkdir -p build/vzup-out
for f in bench-fixed.txt bench-legacy.txt make-fixed.log make-legacy.log; do
  scp -q -o BatchMode=yes "$HOST:~/vzup/$f" "build/vzup-out/$f" 2>/dev/null && echo "   got $f" || echo "   (no $f)"
done
say "driver done"
