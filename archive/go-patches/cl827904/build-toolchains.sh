#!/bin/bash
# #197: build master@46f508b3aa and master@46f508b3aa + CL 827904 PS2 (213d6fb3d2) side by side.
set -euo pipefail
cd ~/src/go
export GOROOT_BOOTSTRAP=$(go env GOROOT)
for pair in base:46f508b3aa365918ac78a63340ce2343306835af cl:213d6fb3d2b9919e894eecbf22eea93c609762ab; do
  n=${pair%%:*}; rev=${pair#*:}; d=~/.xtmp/go-ab/$n
  [ -d "$d" ] || git worktree add -q --detach "$d" "$rev"
  echo "== $n $(git -C "$d" log --oneline -1) $(date -u +%T)"
  (cd "$d/src" && ./make.bash -no-banner 2>&1 | tail -3)
  "$d/bin/go" version
done
echo "== built $(date -u +%T)"
