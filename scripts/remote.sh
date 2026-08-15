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

# worktree_strays — print every registered worktree other than this one, as
# `PATH REVISION`, and return 1 if there are any.
#
# WHY A GATE CARES (#63). The gates assert tree state with `git status`, which
# sees uncommitted changes and nothing else. A *registered worktree* is a second
# checkout of another commit inside the same repository, and it is invisible to
# every check we have. It is a hazard twice over:
#
#   - a stray build, path glob or tool invocation can read the wrong revision's
#     sources out of it;
#   - a future session finds it and cannot tell instrument residue from a live
#     measurement. That is the "is this state a result, or wreckage?" ambiguity
#     DESIGN.md §5.6 spends the gates' whole vocabulary eliminating.
#
# A worktree here usually means an `l1-bench.sh` or `layout-ensemble.sh` run is
# in flight, and that is exactly the condition that should stop a gate rather
# than an exception to be carved out for. The tree is frozen for a measurement's
# life; a gate IS a measurement; so a gate concurrent with a benchmark was never
# legitimate, and the runs have been serialized by hand all campaign for that
# reason. This mechanizes the serialization practice already imposed. One
# measurement at a time stops being discipline and becomes an assertion.
#
# Note the failure is reported on its own rather than folded into the dirty-tree
# check. A dirty tree breaks `git archive HEAD` and so breaks the delegated
# chain by construction; a stray worktree does not touch HEAD and breaks nothing
# mechanically. Sharing one flag would attribute a cause this does not have.
worktree_strays() {
  local main path head n=0
  main="$(git rev-parse --show-toplevel 2>/dev/null)" || return 0
  while IFS= read -r line; do
    case "$line" in
      'worktree '*) path="${line#worktree }" ;;
      'HEAD '*)
        head="${line#HEAD }"
        if [[ "$path" != "$main" ]]; then
          printf '%s %s\n' "$path" "${head:0:7}"
          n=$((n + 1))
        fi
        ;;
    esac
  done < <(git worktree list --porcelain 2>/dev/null)
  [[ "$n" -eq 0 ]]
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
#
# The private cache sizes are here for the same reason, added when #48's feed
# decomposition turned out to depend on one: several benchmarks hold a buffer they
# describe as L1-resident, whose size follows from the blocking parameters, and
# whether that description is true is a comparison against a number no host record
# carried. It is read from sysfs rather than assumed per microarchitecture — Zen 5
# has a 48 KB L1d where Zen 4 and Skylake-X have 32 KB, which is exactly the kind of
# difference a from-memory constant gets wrong. Missing files degrade to "?" rather
# than to a plausible default.
remote_probe() {
  local host="$1"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" '
    cpu=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2- | sed "s/^ *//")
    gov=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo unknown)
    cache=""
    for d in /sys/devices/system/cpu/cpu0/cache/index*; do
      [ -r "$d/level" ] || continue
      lvl=$(cat "$d/level"); typ=$(cat "$d/type"); sz=$(cat "$d/size")
      case "$typ" in Instruction) continue ;; Data) tag="L${lvl}d" ;; *) tag="L${lvl}" ;; esac
      cache="$cache${cache:+ }$tag=$sz"
    done
    printf "%s | %s cpus | governor=%s | %s | %s\n" \
      "$cpu" "$(nproc)" "$gov" "$(uname -sr)" "${cache:-caches=?}"
  ' 2>/dev/null
}

# remote_exec HOST BIN [ARGS...] — ship BIN to HOST and run it there.
# stdout/stderr are the remote program's; the return status is its exit code.
#
# $KEEL_REMOTE_ENV is prepended to the remote command as environment
# assignments (e.g. KEEL_REMOTE_ENV="KEEL_FORCE=scalar"). sshd strips
# arbitrary env vars by default, so passing them through the command line is
# the reliable way rather than SendEnv/AcceptEnv on both sides.
#
# ARGS are quoted with printf %q before crossing the wire. ssh concatenates its
# command words and hands the result to a remote shell, so an unquoted argument
# is *shell input* on the far side: `-test.bench='A|B'` arrived at the remote
# bash as a pipeline and tried to execute B as a command. That failure was loud
# here, but the same expansion applied to a glob or a `$` would have silently
# altered what got measured.
remote_exec() {
  local host="$1" bin="$2"; shift 2
  local base; base="$(basename "$bin")"
  local args="" a
  for a in "$@"; do args+=" $(printf '%q' "$a")"; done
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "mkdir -p '$KEEL_REMOTE_DIR'" >/dev/null
  scp -q "${KEEL_SCP_OPTS[@]}" "$bin" "$host:$KEEL_REMOTE_DIR/$base"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" \
    "cd '$KEEL_REMOTE_DIR' && env ${KEEL_REMOTE_ENV:-} ./'$base'$args"
}

# ---------------------------------------------------------------------------
# Boost / turbo, the frequency regime the scaling ratio's two arms must share
# (DESIGN.md §4/P5, issue #66).
#
# Two knobs, because the polarity is inverted between vendors and nothing above
# this line should have to remember which vendor a host is:
#
#   amd-pstate / acpi-cpufreq  cpufreq/boost        1 = boost permitted
#   intel_pstate               intel_pstate/no_turbo 1 = turbo FORBIDDEN
#
# So both are normalised to off|on here. A host exposing neither reports
# `unknown`, and the gate treats unknown as unmet rather than as satisfied: a
# frequency regime that cannot be read cannot be asserted to be shared.

# remote_boost HOST — print "<off|on|unknown> <path-or-none>".
remote_boost() {
  ssh "${KEEL_SSH_OPTS[@]}" "$1" '
    a=/sys/devices/system/cpu/cpufreq/boost
    i=/sys/devices/system/cpu/intel_pstate/no_turbo
    if [ -r "$a" ]; then
      p=$a
      case "$(cat "$a")" in 0) s=off ;; 1) s=on ;; *) s=unknown ;; esac
    elif [ -r "$i" ]; then
      p=$i
      case "$(cat "$i")" in 1) s=off ;; 0) s=on ;; *) s=unknown ;; esac
    else
      s=unknown; p=none
    fi
    printf "%s %s\n" "$s" "$p"
  ' 2>/dev/null
}

# remote_boost_set HOST off|on — write the knob, normalising polarity. Returns
# nonzero if no knob exists or the write failed; says nothing on success. The
# caller must still read the state back with remote_boost: a write that returns 0
# and a knob that took the value are different facts, and this project asserts the
# second one (same reason the governor is re-read at measurement time).
# The value is spliced into the remote script rather than piped to `sh -s` on
# stdin, and that is not a style choice. $KEEL_SSH_OPTS carries `-n`, which
# redirects ssh's stdin from /dev/null, so a heredoc-fed remote shell reads EOF
# immediately, executes nothing, and **exits 0** — a write that silently did not
# happen, reported as success. The first version of this function did exactly that
# on all three hosts; only reading the knob back caught it, which is the same
# argument this file already makes about the governor. `want` is validated against
# a two-element allowlist before it is spliced, so the interpolation cannot carry
# shell metacharacters.
remote_boost_set() {
  local host="$1" want="$2"
  case "$want" in off | on) ;; *) return 4 ;; esac
  ssh "${KEEL_SSH_OPTS[@]}" "$host" '
    want='"$want"'
    a=/sys/devices/system/cpu/cpufreq/boost
    i=/sys/devices/system/cpu/intel_pstate/no_turbo
    if [ -r "$a" ]; then
      p=$a; case "$want" in off) v=0 ;; on) v=1 ;; esac
    elif [ -r "$i" ]; then
      p=$i; case "$want" in off) v=1 ;; on) v=0 ;; esac
    else
      exit 3
    fi
    printf "%s\n" "$v" | sudo -n tee "$p" >/dev/null
  ' 2>/dev/null
}
