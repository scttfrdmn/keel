#!/usr/bin/env bash
# Shared helper: execute linux/amd64 Go test and benchmark binaries on a
# remote host, from a dev host that cannot run them.
#
# WHY THIS EXISTS. simd/archsimd is amd64-only in go1.26.5 (docs/toolchain-notes
# T1) and keel's dev host is darwin/arm64. Everything the *compiler* decides
# can be checked here by cross-compiling; everything the *CPU* decides —
# whether VMAXPS really returns its second operand for NaN, what fraction of
# peak a kernel reaches — cannot. This closes that gap without installing a Go
# toolchain anywhere: `go test -c` emits a static, pure-Go, dependency-free
# ELF binary, so the host toolchain that the gate already verified is the one
# whose output gets executed. The remote machine needs nothing but sshd.
#
# Sourced by scripts/gate-p*.sh; not meant to be run directly.
#
# Configuration, in precedence order:
#   1. $KEEL_REMOTE_HOSTS  — space-separated host list
#   2. .keel-hosts at the repo root — one host per line, # comments allowed.
#      Machine-local and gitignored: real hostnames are infrastructure, not
#      source. See .keel-hosts.example and docs/hosts.md.
# Unset means "no remote targets", which is not an error here — it is the
# caller's gate that decides whether missing coverage is fatal.

KEEL_REMOTE_DIR="${KEEL_REMOTE_DIR:-/tmp/keel-remote}"
# -n is not optional: without it ssh inherits and drains the caller's stdin,
# so an `ssh` inside a `while read host` loop eats the remaining host list and
# every target after the first silently disappears. That bug produced a GREEN
# gate that had tested exactly one of two machines.
KEEL_SSH_OPTS=(-n -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new)
# scp rejects -n, so it gets the same options minus that one.
KEEL_SCP_OPTS=(-o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new)

# remote_hosts prints the configured hosts, one per line.
remote_hosts() {
  if [[ -n "${KEEL_REMOTE_HOSTS:-}" ]]; then
    printf '%s\n' $KEEL_REMOTE_HOSTS
    return
  fi
  local f
  f="$(git rev-parse --show-toplevel 2>/dev/null || echo .)/.keel-hosts"
  [[ -r "$f" ]] || return 0
  sed -e 's/#.*//' -e '/^[[:space:]]*$/d' -e 's/[[:space:]]//g' "$f"
}

# remote_build_test PKG OUT — cross-compile PKG's test binary for linux/amd64.
#
# CGO_ENABLED=0 guarantees a static binary that does not care which libc the
# remote distro ships (the two reference hosts are Ubuntu and RHEL 9).
# GOAMD64 is deliberately left at its default (v1): archsimd emits its
# intrinsics identically at every GOAMD64 level, and keel dispatches on
# runtime feature detection, so the binary under test is built exactly the way
# a released keel binary would be. See docs/toolchain-notes T7.
remote_build_test() {
  local pkg="$1" out="$2"
  GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go test -c -o "$out" "$pkg"
}

# remote_probe HOST — print a one-line provenance record for HOST.
#
# The CPU model is not decoration: DESIGN.md §7 forbids reporting a
# measurement without the machine it came from, and a correctness result is
# just as host-specific as a benchmark (which microarchitecture confirmed the
# NaN operand order?).
remote_probe() {
  local host="$1"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" '
    cpu=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2- | sed "s/^ *//")
    gov=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo unknown)
    printf "%s | %s cpus | governor=%s | %s\n" "$cpu" "$(nproc)" "$gov" "$(uname -sr)"
  ' 2>/dev/null
}

# remote_exec HOST BIN [ARGS...] — ship BIN to HOST and run it there.
# stdout/stderr are the remote program's; the return status is its exit code.
remote_exec() {
  local host="$1" bin="$2"; shift 2
  local base; base="$(basename "$bin")"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "mkdir -p '$KEEL_REMOTE_DIR'" >/dev/null
  scp -q "${KEEL_SCP_OPTS[@]}" "$bin" "$host:$KEEL_REMOTE_DIR/$base"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "cd '$KEEL_REMOTE_DIR' && ./'$base' $*"
}
