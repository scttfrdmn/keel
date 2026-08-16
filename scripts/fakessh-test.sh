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
cp "$STUB/realssh" "$STUB/realscp"

# The shim selects its behaviour from the name it is invoked as, so the fixtures must
# invoke it through both names rather than calling the file directly.
ln -s "$FAKESSH" "$STUB/ssh"
ln -s "$FAKESSH" "$STUB/scp"

# check NAME WANT_RC WANT_SUBSTR -- args... : run fakessh AS ssh and assert both its
# exit status and something about what it emitted. Status alone is not enough: 255 with
# the wrong message would not resemble an outage to anyone reading the log.
check() {
  local name="$1" wantrc="$2" wantsub="$3"; shift 3
  [[ "$1" == "--" ]] && shift
  local out rc
  out="$("$STUB/ssh" "$@" 2>&1)"; rc=$?
  if [[ "$rc" == "$wantrc" ]] && [[ "$out" == *"$wantsub"* ]]; then
    printf '  ok    %-56s rc=%s\n' "$name" "$rc"
  else
    printf '  FAIL  %-56s rc=%s (want %s), out=%q (want substring %q)\n' \
      "$name" "$rc" "$wantrc" "$out" "$wantsub"
    FAILED=1
  fi
}

# checkscp -- the same, through the scp name. A separate helper rather than a mode
# argument because the two transports differ in what a refusal looks like, and a
# fixture that could not tell them apart would not have caught the miss that made this
# arm necessary.
checkscp() {
  local name="$1" wantrc="$2" wantsub="$3"; shift 3
  [[ "$1" == "--" ]] && shift
  local out rc
  out="$("$STUB/scp" "$@" 2>&1)"; rc=$?
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
  export KEEL_FAKESCP_REAL="$STUB/realscp"
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
  echo "-- scp is covered too, because it had to be told --"
  # 7. THE MISS THAT MADE THIS ARM EXIST. The first version shimmed ssh only, on the
  #    belief that everything crossed the wire through ssh's stdin. remote.sh:440 copies
  #    the bench binary with scp, so the dead host would have answered that call and the
  #    exercise would have been a fiction with a green-looking log. The `host:path` form
  #    is the one a bare-hostname comparison misses.
  checkscp "scp to the dead host, host:path form, is refused"  1 "No route to host" \
    -- -q /tmp/bench dead.example:/tmp/keel/bench
  # 8. scp's failure is not ssh's. It reports the connection error and then "lost
  #    connection", exiting 1 -- so a caller checking for 255 would read a dead host as
  #    a copy that merely failed, and vice versa.
  checkscp "and it exits 1 after 'lost connection', as scp does" 1 "lost connection" \
    -- -q /tmp/bench dead.example:/tmp/keel/bench
  # 9. Pass-through matters more here than for ssh: the two surviving hosts cannot run
  #    the benchmark at all if the binary does not land.
  checkscp "scp to a live host still copies"                   0 "PASSTHROUGH" \
    -- -q /tmp/bench live.example:/tmp/keel/bench
  # 10. A login on the host list would be an ordinary configuration, and the `user@host`
  #     forms have to refuse too or the exercise would quietly half-work.
  check    "user@dead is refused (ssh)"                        255 "No route to host" \
    -- -n scott@dead.example 'echo hi'
  checkscp "user@dead:path is refused (scp)"                     1 "No route to host" \
    -- -q /tmp/bench scott@dead.example:/tmp/keel/bench
  # 11. And the substring guard has to survive the new forms: a `:`-suffixed near-miss
  #     must still live, or fixture 5's protection would have been undone by fixture 7.
  checkscp "a host containing the dead name still copies"        0 "PASSTHROUGH" \
    -- -q /tmp/bench dead.example.backup:/tmp/keel/bench

  echo
  echo "-- an unknown invocation name has no behaviour --"
  # 12. Invoked as anything but ssh or scp, the shim exits 2 rather than guessing. If a
  #     third transport ever appears, this is the line that says so instead of the shim
  #     silently passing it through to a host it was told was dead.
  ln -sf "$FAKESSH" "$STUB/sftp"
  out="$("$STUB/sftp" dead.example 2>&1)"; rc=$?
  if [[ "$rc" == 2 && "$out" == *"neither ssh nor scp"* ]]; then
    printf '  ok    %-56s rc=%s\n' "invoked as sftp: exits 2, does not guess" "$rc"
  else
    printf '  FAIL  %-56s rc=%s out=%q\n' "invoked as sftp: exits 2, does not guess" "$rc" "$out"
    FAILED=1
  fi

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
