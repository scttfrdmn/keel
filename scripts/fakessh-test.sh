#!/usr/bin/env bash
# Copyright 2026 The keel Authors
# SPDX-License-Identifier: Apache-2.0
#
# fakessh-test.sh -- fixtures for scripts/fakessh, the dead-host induction shim.
#
# Why a shim gets its own controls: it is the instrument that decides what a
# 25-minute exercise measures. A fakessh that refused too much would make a healthy
# host look down and the exercise would report a branch firing for the wrong reason;
# one that refused too little would let the dead host answer and the exercise would
# be a fiction with a green-looking log. Neither failure announces itself, and both
# are cheap to rule out here.
#
# No network: KEEL_FAKESSH_REAL points at a stub that prints its own arguments, so
# "passed through" is observable without an actual ssh.
set -uo pipefail

# || exit, not a bare cd: this file runs without `set -e` (the fixtures need to see
# nonzero statuses), so an unguarded cd would carry on in the wrong directory and
# every fixture would fail for a reason that has nothing to do with the shim.
cd "$(dirname "$0")/.." || exit 2
FAKESSH="$PWD/scripts/fakessh"
FAILED=0

STUB="$(mktemp -d "${TMPDIR:-/tmp}/keel-fakessh-test.XXXXXX")"
trap 'rm -rf "$STUB"' EXIT
cat > "$STUB/realssh" <<'EOF'
#!/usr/bin/env bash
echo "PASSTHROUGH: $*"
exit 0
EOF
chmod +x "$STUB/realssh"

# check NAME WANT_RC WANT_SUBSTR -- args... : run fakessh and assert both its exit
# status and something about what it emitted. Status alone is not enough: 255 with the
# wrong message would not resemble an outage to anyone reading the log.
check() {
  local name="$1" wantrc="$2" wantsub="$3"; shift 3
  [[ "$1" == "--" ]] && shift
  local out rc
  out="$("$FAKESSH" "$@" 2>&1)"; rc=$?
  if [[ "$rc" == "$wantrc" ]] && [[ "$out" == *"$wantsub"* ]]; then
    printf '  ok    %-56s rc=%s\n' "$name" "$rc"
  else
    printf '  FAIL  %-56s rc=%s (want %s), out=%q (want substring %q)\n' \
      "$name" "$rc" "$wantrc" "$out" "$wantsub"
    FAILED=1
  fi
}

main() {
  echo "fakessh controls -- the dead-host induction shim."
  echo

  export KEEL_FAKESSH_REAL="$STUB/realssh"
  export KEEL_FAKESSH_DEAD="dead.example"

  echo "-- refusal looks like a real outage --"
  # 1. The whole purpose. 255 is ssh's own connection-failure status, and the message
  #    is ssh's own wording, so a gate's error path sees what an outage produces.
  check "the dead host is refused with ssh's own 255"  255 "No route to host" \
    -- -n dead.example 'echo hi'
  # 2. The refusal must not depend on argument position: remote.sh calls ssh with
  #    option bundles of varying length ahead of the host.
  check "refused wherever the host sits in the args"   255 "No route to host" \
    -- -n -o BatchMode=yes -o ConnectTimeout=5 dead.example 'uname -a'

  echo
  echo "-- everything else passes through untouched --"
  # 3. Two live hosts genuinely measure during the exercise, so pass-through is as
  #    load-bearing as refusal.
  check "a live host reaches the real ssh"             0   "PASSTHROUGH: -n live.example" \
    -- -n live.example 'echo hi'
  # 4. Arguments are forwarded verbatim, including the remote command. A shim that
  #    dropped or reordered them would break the two surviving hosts in a way that
  #    would read as a keel defect.
  check "arguments are forwarded verbatim"             0   "PASSTHROUGH: -n -o BatchMode=yes live.example uname -m" \
    -- -n -o BatchMode=yes live.example uname -m

  echo
  echo "-- matching is exact, never substring --"
  # 5. THE FAILURE THAT WOULD LOOK LIKE A FINDING. If matching were a substring test,
  #    a dead `zen4.local` would also kill `zen4.local.backup` and any host whose name
  #    contains it -- two hosts down, the aggregate reading 1/1, and the log would
  #    still say "one host unreachable".
  check "a host merely CONTAINING the dead name lives" 0   "PASSTHROUGH" \
    -- -n dead.example.backup 'echo hi'
  check "a host the dead name contains also lives"     0   "PASSTHROUGH" \
    -- -n dead 'echo hi'
  # 6. And an option VALUE equal to the dead name is not a host. Contrived, but it is
  #    the reason the shim compares whole words instead of parsing: the comparison has
  #    to be safe for arguments it does not understand.
  check "the dead name as a remote command is refused too" 255 "No route to host" \
    -- -n live.example dead.example
  echo "        ^ known and accepted: an argument equal to the dead host refuses"
  echo "          wherever it appears. remote.sh never passes a hostname as a remote"
  echo "          command, and the alternative is parsing ssh's option grammar, whose"
  echo "          failure mode is refusing the WRONG host silently. Reported, not hidden."

  echo
  echo "-- a misconfigured shim refuses to run rather than lie --"
  # 7. Both of these would produce a plausible-looking exercise that established
  #    nothing: no dead host means every host lives and the aggregate reads 3/3; no
  #    real ssh means the shim cannot resolve one without finding itself.
  ( unset KEEL_FAKESSH_DEAD
    check "no dead host configured: exits 2, not silently transparent" 2 "refuse nothing" \
      -- -n live.example 'echo hi' ) || FAILED=1
  ( unset KEEL_FAKESSH_REAL
    check "no real ssh configured: exits 2, does not self-resolve"     2 "finds itself" \
      -- -n live.example 'echo hi' ) || FAILED=1

  echo
  if [[ "$FAILED" -eq 0 ]]; then
    echo "fakessh controls: all pass"
  else
    echo "fakessh controls: FAILED" >&2
    exit 1
  fi
}

main "$@"
