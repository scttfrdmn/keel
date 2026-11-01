#!/bin/bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# #197: build master@0d37007fda (CL 824624 PS2's own parent) and master@0d37007fda + PS2 (7745cfc920)
# side by side, from ~/src/go worktrees. Pattern: archive/go-patches/cl827904/build-toolchains.sh.
set -euo pipefail
cd ~/src/go
export GOROOT_BOOTSTRAP=$(go env GOROOT)
for pair in base:0d37007fda cl:7745cfc920; do
  n=${pair%%:*}; rev=${pair#*:}; d=~/.xtmp/go-ab824/$n
  [ -d "$d" ] || git worktree add -q --detach "$d" "$rev"
  echo "== $n $(git -C "$d" log --oneline -1) $(date -u +%T)"
  (cd "$d/src" && ./make.bash -no-banner 2>&1 | tail -3)
  "$d/bin/go" version
done
echo "== built $(date -u +%T)"
