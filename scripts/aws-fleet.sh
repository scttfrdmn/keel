#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# aws-fleet.sh -- launch, list and terminate keel's spot measurement fleet (#24).
#
#   scripts/aws-fleet.sh up       # launch, wait for sshd, write .keel-hosts
#   scripts/aws-fleet.sh status   # what is running, and what it has cost so far
#   scripts/aws-fleet.sh down     # terminate everything tagged Project=keel
#
# THIS SCRIPT SPENDS MONEY. Every guard below exists because the expensive failure
# here is not a wrong number, it is a fleet nobody remembered:
#
#   - a dead-man switch. Each guest runs `shutdown -h +$KEEL_FLEET_TTL_MIN` at boot
#     and is launched with instance-initiated-shutdown-behavior=terminate, so the
#     fleet dies on its own if this script, this shell, or this human forgets it.
#     A TTL that fires mid-benchmark costs one reading and #62 reports it `vanished`;
#     a TTL that does not exist costs a month of instance-hours.
#   - `down` selects by TAG, not by a list this script wrote, so it still finds
#     instances after a lost .keel-hosts or a machine reinstall.
#   - `up` refuses to run while any tagged instance is alive, because two fleets is
#     the shape that produces a forgotten one.
#
# The SOFTWARE half is not here: scripts/provision-openblas.sh already installs Go
# and libopenblas and verifies the openblas-tagged harness, on Ubuntu among others.
# This script's whole job is to make hosts exist and be reachable by name.
set -euo pipefail

cd "$(dirname "$0")/.."

REGION="${AWS_REGION:-us-east-1}"
TTL_MIN="${KEEL_FLEET_TTL_MIN:-480}"
KEY_NAME=keel-fleet
KEY_FILE="$HOME/.ssh/keel-fleet.pem"
SG_NAME=keel-fleet
SSH_CONF="$HOME/.ssh/config"
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
awsq() { aws --region "$REGION" "$@"; }

tagged() {
  # shellcheck disable=SC2016  # the backticks are JMESPath string literals, not a subshell
  awsq ec2 describe-instances \
    --filters Name=tag:Project,Values=keel Name=instance-state-name,Values=pending,running \
    --query 'Reservations[].Instances[].[InstanceId,Tags[?Key==`Role`]|[0].Value,InstanceType,PublicIpAddress,LaunchTime]' \
    --output text
}

ensure_key() {
  if awsq ec2 describe-key-pairs --key-names "$KEY_NAME" >/dev/null 2>&1; then
    [[ -r "$KEY_FILE" ]] || die "key pair $KEY_NAME exists in EC2 but $KEY_FILE is missing: delete the key pair or restore the file, because a fleet nobody can ssh to is a fleet nobody can terminate cleanly"
    return
  fi
  say "creating key pair $KEY_NAME -> $KEY_FILE"
  awsq ec2 create-key-pair --key-name "$KEY_NAME" --query KeyMaterial --output text > "$KEY_FILE"
  chmod 600 "$KEY_FILE"
}

# ensure_sg -- prints the group id and NOTHING ELSE on stdout, because the caller
# captures it. Its progress notes go to stderr: a helper whose stdout is a value
# cannot also narrate there, which is the same rule remote_exec follows for the
# opposite reason (its stdout IS the measurement's log).
ensure_sg() {
  local id ip
  id="$(awsq ec2 describe-security-groups --filters Name=group-name,Values="$SG_NAME" \
        --query 'SecurityGroups[0].GroupId' --output text 2>/dev/null || true)"
  if [[ -z "$id" || "$id" == None ]]; then
    id="$(awsq ec2 create-security-group --group-name "$SG_NAME" \
          --description "keel measurement fleet: ssh from the driver only" \
          --query GroupId --output text)"
    say "created security group $id" >&2
  fi
  # Re-authorized every `up` because a driver on DHCP or a different network gets a
  # new address, and the symptom of a stale rule is a fleet that boots, bills, and
  # never answers -- which reads exactly like a bad AMI.
  ip="$(curl -fsS https://checkip.amazonaws.com | tr -d '[:space:]')"
  [[ -n "$ip" ]] || die "could not determine this machine's public address"
  awsq ec2 authorize-security-group-ingress --group-id "$id" --protocol tcp --port 22 \
    --cidr "$ip/32" >/dev/null 2>&1 || true
  say "ssh ingress: $ip/32" >&2
  printf '%s' "$id"
}

# write_ssh_config BLOCK -- replace the managed block, or remove it if BLOCK is
# empty. Deletion is by awk with FIXED-STRING comparison, not by a sed range: the
# markers contain `/` and `(`, which are a delimiter and a group to sed, so
# `sed "/^$BEGIN_MARK$/,..."` fails to parse -- and it failed AFTER run-instances
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
  [[ -z "$(tagged)" ]] || die "instances tagged Project=keel are already alive; run 'down' first (two fleets is how one gets forgotten)"
  local ami sg role type uarch iid ids=()
  ami="$(awsq ssm get-parameters \
    --names /aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id \
    --query 'Parameters[0].Value' --output text)"
  [[ "$ami" == ami-* ]] || die "no Ubuntu 24.04 AMI id from SSM (got '$ami')"
  say "region $REGION, AMI $ami, TTL ${TTL_MIN}m"
  ensure_key
  sg="$(ensure_sg)"

  # tmux is installed here and not by provision-openblas.sh because #62's supervisor
  # is a property of the HOST, not of the benchmark: a guest without it measures
  # fine and reports `tmux=no`, which is a quieter fleet than it looks.
  local userdata
  userdata="$(printf '#!/bin/bash\nshutdown -h +%s\nexport DEBIAN_FRONTEND=noninteractive\napt-get update -y\napt-get install -y tmux\n' "$TTL_MIN" | base64)"

  for spec in "${FLEET[@]}"; do
    IFS=: read -r role type uarch <<<"$spec"
    iid="$(awsq ec2 run-instances --image-id "$ami" --instance-type "$type" \
      --key-name "$KEY_NAME" --security-group-ids "$sg" --count 1 \
      --instance-initiated-shutdown-behavior terminate \
      --user-data "$userdata" \
      --block-device-mappings 'DeviceName=/dev/sda1,Ebs={VolumeSize=24,VolumeType=gp3,DeleteOnTermination=true}' \
      --instance-market-options 'MarketType=spot,SpotOptions={SpotInstanceType=one-time,InstanceInterruptionBehavior=terminate}' \
      --tag-specifications "ResourceType=instance,Tags=[{Key=Project,Value=keel},{Key=Role,Value=$role},{Key=Name,Value=keel-$role}]" \
      --query 'Instances[0].InstanceId' --output text)"
    ids+=("$iid")
    say "$role  $type  $iid  ($uarch)"
  done

  say "waiting for running state"
  awsq ec2 wait instance-running --instance-ids "${ids[@]}"
  cmd_wire
}

# cmd_wire -- everything between "the instances exist" and "the gates can reach them".
# Separate from `up`, and idempotent, because it is the part that failed once while
# three instances were already billing: a launch step that cannot be resumed forces
# the choice between hand-editing config and terminating a fleet that is fine.
cmd_wire() {
  local conf="" role host waited spec ip
  for spec in "${FLEET[@]}"; do
    IFS=: read -r role _ _ <<<"$spec"
    ip="$(awsq ec2 describe-instances --filters Name=tag:Role,Values="$role" Name=tag:Project,Values=keel \
          Name=instance-state-name,Values=running --query 'Reservations[].Instances[0].PublicIpAddress' --output text)"
    conf+="Host keel-$role
  HostName $ip
  User ubuntu
  IdentityFile $KEY_FILE
  StrictHostKeyChecking accept-new
  UserKnownHostsFile /dev/null
  LogLevel ERROR
"
  done
  write_ssh_config "$conf"

  # .keel-hosts holds ROLE names, never addresses: they land in gate logs, and gate
  # logs get pasted into issues. Same convention as keying published numbers by CPU
  # model -- the address lives in ~/.ssh/config, which is not this repo.
  { echo "# written by scripts/aws-fleet.sh -- the AWS spot measurement fleet (#24)"
    for spec in "${FLEET[@]}"; do IFS=: read -r role _ _ <<<"$spec"; echo "keel-$role"; done
  } > .keel-hosts
  say "wrote .keel-hosts and ~/.ssh/config block"

  say "waiting for sshd"
  for spec in "${FLEET[@]}"; do
    IFS=: read -r role _ _ <<<"$spec"; host="keel-$role"; waited=0
    until ssh -o BatchMode=yes -o ConnectTimeout=5 "$host" true 2>/dev/null; do
      waited=$((waited + 5)); [[ "$waited" -lt 300 ]] || die "$host never answered ssh"
      sleep 5
    done
    say "$host up after ${waited}s"
  done
  cmd_status
}

cmd_status() {
  local rows; rows="$(tagged)"
  if [[ -z "$rows" ]]; then say "no instances tagged Project=keel are alive"; return; fi
  printf '\n  %-20s %-14s %-16s %s\n' ROLE TYPE ADDRESS LAUNCHED
  while read -r iid role type ip launched; do
    printf '  %-20s %-14s %-16s %s\n' "keel-$role" "$type" "$ip" "$launched"
  done <<<"$rows"
  printf '\n  %s\n' "$(wc -l <<<"$rows" | tr -d ' ') instance(s) billing. 'down' terminates every one."
}

cmd_down() {
  local rows ids=(); rows="$(tagged)"
  if [[ -z "$rows" ]]; then
    say "nothing tagged Project=keel is alive"
  else
    while read -r iid _ _ _ _; do ids+=("$iid"); done <<<"$rows"
    awsq ec2 terminate-instances --instance-ids "${ids[@]}" \
      --query 'TerminatingInstances[].[InstanceId,CurrentState.Name]' --output text | sed 's/^/  /'
    say "waiting for terminated state"
    awsq ec2 wait instance-terminated --instance-ids "${ids[@]}"
  fi
  write_ssh_config ""
  say "removed the ~/.ssh/config block"
  # .keel-hosts is left in place on purpose: it names roles, not addresses, so it is
  # harmless, and blanking it would make the next gate run report "no fleet
  # configured" rather than a fleet that does not answer. Those are different
  # causes and the second one is the true one (§5 rule 6).
  say "left .keel-hosts naming the roles: a fleet that is down is not a fleet nobody configured"
}

case "${1:-}" in
  up)     cmd_up ;;
  wire)   cmd_wire ;;
  status) cmd_status ;;
  down)   cmd_down ;;
  *)      die "usage: $0 up|wire|status|down" ;;
esac
