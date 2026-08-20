#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# aws-fleet.sh -- launch, list and terminate keel's AWS measurement fleet (#12), over
# `spawn` (Scott's directive, 2026-08-19: instances via truffle/spawn exclusively).
#
#   scripts/aws-fleet.sh up       # launch, wait for sshd, write .keel-hosts
#   scripts/aws-fleet.sh status   # what is running, since when, and on which market
#   scripts/aws-fleet.sh down     # terminate every keel- instance the launcher knows
#
# THIS SCRIPT SPENDS MONEY, and the expensive failure is not a wrong number, it is a
# fleet nobody remembered. Two of the three guards that used to live here are now
# `spawn`'s and are better there:
#
#   - the dead-man switch is `--ttl`, enforced by the launcher's own reaper rather than
#     by a `shutdown -h` this script baked into userdata. A TTL that fires mid-benchmark
#     costs one reading and #62 reports it `vanished`; a TTL that does not exist costs a
#     month of instance-hours, and the old one depended on the guest's own init working.
#   - `down` selects by the LAUNCHER'S OWN NAME, not by a list this script wrote, so it
#     still finds instances after a lost .keel-hosts. That was the point of selecting by
#     tag before, and `spawn list` reports no tags (see fleet_json).
#   - `up` still refuses to run while any keel- instance is alive, because two fleets is
#     the shape that produces a forgotten one.
#
# The SOFTWARE half is not here: scripts/provision-openblas.sh installs Go and
# libopenblas and verifies the openblas-tagged harness. This script's whole job is to
# make hosts exist and be reachable by name.
set -euo pipefail

cd "$(dirname "$0")/.."

SPAWN="${KEEL_SPAWN:-spawn}"
export AWS_PROFILE="${KEEL_SPAWN_PROFILE:-aws}"
REGION="${AWS_REGION:-us-east-1}"
TTL="${KEEL_FLEET_TTL:-8h}"
# Overridable so the block writer can be DRIVEN, which is not a convenience: this is the
# step that failed once after three instances were already billing, and the only way to
# exercise it otherwise is against the operator's real ~/.ssh/config.
SSH_CONF="${KEEL_SSH_CONFIG:-$HOME/.ssh/config}"
BEGIN_MARK="# BEGIN keel-fleet (scripts/aws-fleet.sh)"
END_MARK="# END keel-fleet"

# role:instance-type:microarchitecture. Every type here has 8 PHYSICAL cores, read
# from describe-instance-types rather than inferred from vCPUs -- Intel reports 2
# threads/core and AMD 1, so equal vCPU counts would have meant unequal fleets and
# GOMAXPROCS=8 would name a different machine on each arm (#82).
FLEET=(
  "zen4:c7a.2xlarge:AMD EPYC Genoa"
  "zen5:c8a.2xlarge:AMD EPYC Turin"
  "spr:c7i.4xlarge:Intel Sapphire Rapids"
)

say()  { printf '  %s\n' "$*"; }
die()  { printf 'aws-fleet: %s\n' "$*" >&2; exit 1; }

command -v "$SPAWN" >/dev/null 2>&1 || die "no '$SPAWN' on PATH; the judged tier is defined over instances truffle/spawn launched"
command -v jq >/dev/null 2>&1 || die "no jq on PATH; spawn's inventory is JSON and this script will not parse it by hand"

# KEEL_FLEET overrides that list, one `role:type:uarch` spec per line, so a judged
# campaign can boot one full-size host without editing the exploration fleet -- which
# must keep describing what the partial-size readings were taken on.
#
# Each spec's SHAPE is checked, not just its presence: a blank-but-not-empty line -- one
# space, which is what a heredoc leaves behind -- passed a `-z` test and launched an
# instance with no role and no type. Whole list first, because roles launch in order and
# a malformed third spec otherwise bills the first two before the launcher rejects it.
if [[ -n "${KEEL_FLEET:-}" ]]; then
  FLEET=()
  while IFS= read -r spec; do
    [[ -n "${spec//[[:space:]]/}" ]] || continue
    [[ "$spec" =~ ^[^:]+:[^:]+:.+$ ]] || die "KEEL_FLEET spec '$spec' is not role:type:uarch"
    FLEET+=("$spec")
  done <<<"$KEEL_FLEET"
  [[ "${#FLEET[@]}" -gt 0 ]] || die "KEEL_FLEET is set but names no role:type:uarch spec"
fi

# KEEL_FLEET_MARKET -- spot for exploration, on-demand for a judged run: "on-demand for
# judged runs because interruptions corrupt measurements, which is the only reason"
# (2026-08-17, reaffirmed 2026-08-19). A reclaim mid-measurement turns host-dollars into
# an honest-but-empty log.
#
# The default flipped to on-demand on 2026-08-19, when the judged tier became the normal
# case. It is only a REQUEST either way: host_admission reads the market back off the
# launcher's record and refuses to judge a spot host, so the two disagreeing is something
# the gate says out loud rather than something this variable settles.
#
# An empty value dies rather than defaulting -- `${VAR-on-demand}`, not `${VAR:-...}`,
# which would silently accept `MARKET=''`. `ondemand` is an accepted alias because that is
# the spelling spawn_probe writes into a provenance line, so the value an operator copies
# out of one is not a death.
MARKET="${KEEL_FLEET_MARKET-on-demand}"
case "$MARKET" in
  spot) ;;
  on-demand | ondemand) MARKET=on-demand ;;
  *) die "KEEL_FLEET_MARKET='$MARKET' is neither 'spot' nor 'on-demand' -- refusing to guess which one a judged run wanted" ;;
esac

# fleet_json -- the launcher's inventory of this project's instances, as a JSON array.
# Selected by NAME PREFIX because `spawn list` reports no tags, and validated as an array
# before use: an unparseable answer must not read as an empty fleet, which is the
# difference between "nothing is running" and "the launcher could not be asked" (§5 rule
# 6, and the same distinction `spawn_probe` draws between `none` and `?`).
fleet_json() {
  local out
  out="$("$SPAWN" list --state running --output json 2>/dev/null || true)"
  jq -e 'type == "array"' >/dev/null 2>&1 <<<"$out" ||
    die "could not read the launcher's inventory ('$SPAWN list'); refusing to act on an unknown fleet"
  jq '[.[] | select(.name | startswith("keel-"))]' <<<"$out"
}

# write_ssh_config BLOCK -- replace the managed block, or remove it if BLOCK is
# empty. Deletion is by awk with FIXED-STRING comparison, not by a sed range: the
# markers contain `/` and `(`, which are a delimiter and a group to sed, so
# `sed "/^$BEGIN_MARK$/,..."` fails to parse -- and it failed AFTER the launch
# had already succeeded, which is the argument for this being its own step below.
write_ssh_config() {
  local block="$1" tmp
  tmp="$(mktemp)"
  if [[ -r "$SSH_CONF" ]]; then
    awk -v b="$BEGIN_MARK" -v e="$END_MARK" '
      $0 == b { skip = 1; next }
      $0 == e { skip = 0; next }
      !skip' "$SSH_CONF" > "$tmp"
  fi
  [[ -z "$block" ]] || printf '%s\n%s%s\n' "$BEGIN_MARK" "$block" "$END_MARK" >> "$tmp"
  mkdir -p "$(dirname "$SSH_CONF")"
  mv "$tmp" "$SSH_CONF"
  chmod 600 "$SSH_CONF"
}

cmd_up() {
  [[ "$(fleet_json | jq length)" -eq 0 ]] ||
    die "keel- instances are already running; run 'down' first (two fleets is how one gets forgotten)"

  # THE ONE NON-SPAWN AWS CALL, and it is not an instance operation: an SSM parameter
  # read. The distro is pinned to Ubuntu 24.04 rather than taking spawn's AL2023 default
  # because provision-openblas.sh's package maps cover ubuntu/debian and rhel/fedora and
  # NOT `amzn`, so AL2023 would reach `unrecognized distro id` after the fleet was
  # billing. Changing the OS also changes which OpenBLAS build every published ratio is
  # measured against, and that is not a change to make as a side effect of a launcher
  # rewrite.
  local ami
  ami="$(aws --region "$REGION" ssm get-parameters \
    --names /aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id \
    --query 'Parameters[0].Value' --output text)"
  [[ "$ami" == ami-* ]] || die "no Ubuntu 24.04 AMI id from SSM (got '$ami')"
  say "region $REGION, AMI $ami, TTL $TTL, market $MARKET"

  # Spot is the PRESENCE of a flag, not a value for one, so the arm is an array that is
  # either empty or the whole option.
  local market_args=()
  [[ "$MARKET" != spot ]] || market_args=(--spot)

  # KEEL_FLEET_DRYRUN=1 appends --estimate-only: spawn prices the launch and returns
  # without creating anything. This exists so the invocation that spends can be validated
  # AS THE SHIPPED COMMAND rather than as a hand-typed mirror of it -- a mirror agrees
  # with the original until the day it is the reason a 48xlarge fleet fails on a rejected
  # flag. Flag validation is the WHOLE value: `48xlarge` is missing from spawn's rate table
  # and unmatched sizes default to xlarge (spawn#543), so the price reads 32x low.
  local dry_args=()
  [[ "${KEEL_FLEET_DRYRUN:-0}" != 1 ]] || { dry_args=(--estimate-only); say "DRY RUN: validating flags, nothing will be launched (spawn's price is 32x low, spawn#543)"; }

  local role type uarch spec
  for spec in "${FLEET[@]}"; do
    IFS=: read -r role type uarch <<<"$spec"
    # THE NAME IS THE JOIN KEY. It equals the ssh alias this script writes below and the
    # `name` spawn_probe matches on, so admission can only vouch for a host whose alias
    # and launcher record are the same string. Nothing enforces that from the far side;
    # it is enforced here, at the one place both are written.
    #
    # tmux via --command rather than --user-data: #62's supervisor is a property of the
    # HOST (a guest without it measures fine and reports `tmux=no`, which is a quieter
    # fleet than it looks), and spawn puts its own agent in userdata, so passing ours
    # would trade a supervisor for the launcher's reaper.
    #
    # --region IS NOT OPTIONAL HERE even though `list` deliberately omits it: an AMI id is
    # region-scoped, so a launch that took spawn's own default region while the AMI came
    # from $REGION would fail on an invalid-AMI error whose text names neither variable.
    # One region, read once, used for both.
    # THE NAME GOES POSITIONALLY. `--name` also exists and its help says "required", which
    # is how this was first written; the launcher rejected it with "accepts 1 arg(s),
    # received 0" because the usage line is `spawn launch <name>` and cobra wants the
    # positional regardless. Read the usage line, not the flag description -- the same rule
    # the simd API earns in CLAUDE.md, and it cost one dry run here instead of a fleet.
    "$SPAWN" launch "keel-$role" --instance-type "$type" --ami "$ami" \
      --region "$REGION" --ttl "$TTL" "${market_args[@]+"${market_args[@]}"}" \
      --tag Project=keel --tag Role="$role" \
      --command 'sudo apt-get update -y && sudo apt-get install -y tmux' \
      --wait-for-ssh -y "${dry_args[@]+"${dry_args[@]}"}" ||
      die "launching keel-$role ($type) failed; 'status' says what is billing"
    say "$role  $type  ($uarch, $MARKET)"
  done
  # A dry run has nothing to wire, and wiring would die on an empty inventory -- which
  # would read as a failed validation when the validation in fact passed.
  [[ "${KEEL_FLEET_DRYRUN:-0}" != 1 ]] || { say "dry run complete: no instances exist, so nothing was wired"; return 0; }
  cmd_wire
}

# cmd_wire -- everything between "the instances exist" and "the gates can reach them".
# Separate from `up`, and idempotent, because it is the part that failed once while
# three instances were already billing: a launch step that cannot be resumed forces
# the choice between hand-editing config and terminating a fleet that is fine.
cmd_wire() {
  local conf="" rows host ip key
  # The address AND the key both come from the launcher's record, not from constants
  # here: spawn owns the key pair now, and a hardcoded IdentityFile would be this
  # script asserting something it no longer decides.
  rows="$(fleet_json | jq -r '.[] | "\(.name)\t\(.public_ip)\t\(.key_name)"')"
  [[ -n "$rows" ]] || die "no keel- instances are running, so there is nothing to wire"
  while IFS=$'\t' read -r host ip key; do
    [[ -n "$ip" && "$ip" != null ]] || die "$host has no public address in the launcher's record"
    conf+="Host $host
  HostName $ip
  User ubuntu
  IdentityFile $HOME/.spawn/keys/$key
  IdentitiesOnly yes
  StrictHostKeyChecking accept-new
  UserKnownHostsFile /dev/null
  LogLevel ERROR
"
  done <<<"$rows"
  write_ssh_config "$conf"

  # .keel-hosts holds the LAUNCHER'S NAMES, never addresses: they land in gate logs, and
  # gate logs get pasted into issues. Same convention as keying published numbers by CPU
  # model -- the address lives in ~/.ssh/config, which is not this repo. It is also the
  # provenance join key, so what a gate reads here is what admission looks up.
  { echo "# written by scripts/aws-fleet.sh -- the AWS measurement fleet (#12)"
    jq -r '.[].name' <<<"$(fleet_json)"
  } > .keel-hosts
  say "wrote .keel-hosts and ~/.ssh/config block"

  # spawn's --wait-for-ssh already waited for sshd, so what is checked here is OUR
  # config, which is a different failure: an alias that resolves to the wrong address,
  # or a key spawn renamed. One attempt, not a poll.
  #
  # EVERY ssh IN THIS LOOP TAKES -n, and that is load-bearing rather than tidy: ssh reads
  # stdin, the loop's stdin is the herestring holding the REMAINING hosts, so the first
  # unredirected ssh swallows them and the loop ends after one iteration. It would have
  # ended by SUCCEEDING -- checking host one, reporting nothing about two and three, and
  # handing a silently partial verification to a judged run. A checker is silent about
  # what it never read.
  while IFS=$'\t' read -r host _ _; do
    ssh -n -o BatchMode=yes -o ConnectTimeout=10 "$host" true 2>/dev/null ||
      die "$host does not answer through the alias this script just wrote (the launcher says sshd is up, so suspect the config block, not the boot)"
    # sshd answers BEFORE cloud-init finishes -- so a provisioner starting here races the
    # boot's own apt and loses the lock. Measured on a judged on-demand run: "Could not
    # get lock /var/lib/apt/lists/lock ... held by process 3085 (apt-get)", ten minutes
    # into an $8.568/hour instance, leaving a host with no OpenBLAS and therefore no
    # mission ratio. A sleep would be a guess here, and the guess that reads as fine is
    # the expensive one; `status --wait` is the host saying its own boot is done.
    if ssh -n -o BatchMode=yes "$host" 'command -v cloud-init >/dev/null 2>&1'; then
      if ssh -n -o BatchMode=yes "$host" 'cloud-init status --wait >/dev/null 2>&1'; then
        say "$host cloud-init settled"
      else
        say "$host cloud-init did not finish clean, so this host booted degraded; a provisioner may still work and the boot-time apt is at least over"
      fi
    else
      say "$host has no cloud-init, so there is no boot-time apt to wait for"
    fi
    # --command's tmux install runs after spored setup, which is after this wait, so it
    # is REPORTED and not required: `tmux=no` in a provenance line is the honest reading
    # and #62's supervisor says so loudly on its own.
    ssh -n -o BatchMode=yes "$host" 'command -v tmux >/dev/null 2>&1' &&
      say "$host has tmux" || say "$host has NO tmux yet: the supervisor (#62) needs it, so re-check before a judged run"
  done <<<"$rows"
  cmd_status
}

cmd_status() {
  local rows; rows="$(fleet_json)"
  if [[ "$(jq length <<<"$rows")" -eq 0 ]]; then say "no keel- instances are running"; return; fi
  printf '\n  %-20s %-16s %-16s %-10s %-8s %s\n' NAME TYPE ADDRESS MARKET TTL LAUNCHED
  # `spot` is a boolean the launcher records, so unlike the old Market TAG there is no
  # "unread" state to render: either the field is there or fleet_json already died.
  jq -r '.[] | [.name, .instance_type, (.public_ip // "none"),
                (if .spot then "spot" else "on-demand" end),
                (.ttl // "-"), .launch_time] | @tsv' <<<"$rows" |
    while IFS=$'\t' read -r n t ip m ttl l; do
      printf '  %-20s %-16s %-16s %-10s %-8s %s\n' "$n" "$t" "$ip" "$m" "$ttl" "$l"
    done
  printf '\n  %s instance(s) billing. '"'"'down'"'"' terminates every one.\n' "$(jq length <<<"$rows")"
}

cmd_down() {
  local rows name
  rows="$(fleet_json | jq -r '.[].name')"
  if [[ -z "$rows" ]]; then
    say "no keel- instances are running"
  else
    while read -r name; do
      say "terminating $name"
      "$SPAWN" terminate "$name" --yes || say "  '$SPAWN terminate $name' failed -- CHECK 'status', because this one may still be billing"
    done <<<"$rows"
  fi
  write_ssh_config ""
  say "removed the ~/.ssh/config block"
  # .keel-hosts is left in place on purpose: it names launcher names, not addresses, so it
  # is harmless, and blanking it would make the next gate run report "no fleet
  # configured" rather than a fleet that does not answer. Those are different causes and
  # the second one is the true one (§5 rule 6).
  say "left .keel-hosts naming the fleet: a fleet that is down is not a fleet nobody configured"
}

case "${1:-}" in
  up)     cmd_up ;;
  wire)   cmd_wire ;;
  status) cmd_status ;;
  down)   cmd_down ;;
  *)      die "usage: $0 up|wire|status|down" ;;
esac
