#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Provision the same-host OpenBLAS reference on every gate host (issue #23).
#
# WHY THIS IS A SEPARATE SCRIPT AND NOT PART OF THE GATE. scripts/gate-p3.sh
# connects with `-o BatchMode=yes` and can neither answer a sudo prompt nor be
# trusted to install software on somebody's machines. This script is the human half:
# Scott runs it, sudo prompts him directly over `ssh -t`, and nothing here ever
# reads, stores or forwards a credential. The gate's job is to measure what it finds
# and to fail loudly when it finds nothing.
#
# WHAT THE RULING ASKS FOR (#23, DESIGN.md §4/P3 as amended). One OpenBLAS per gate
# host, so that every ratio is same silicon, same thread count, same run. Three
# things must end up recorded for each host, because each of them can move a ratio:
#
#   1. the OpenBLAS version and build string       -- what the denominator is
#   2. the DYNAMIC_ARCH-selected kernel family     -- whether it is as fast as the
#                                                     host can be. A generic kernel
#                                                     reads LOW, which inflates
#                                                     keel's ratio: the failure
#                                                     direction a gate must not have
#   3. OPENBLAS_NUM_THREADS=1 taking effect        -- one thread on both sides
#
# So this script does not stop at "the package installed". It ships the tree, builds
# the openblas-tagged harness natively, runs it briefly, and prints the marker the
# library emits about itself — which is the same marker the gate will check. If the
# distro package selected a pre-AVX2 kernel, that is visible here, before a session
# is spent on a benchmark whose denominator was wrong.
#
# Usage:
#   scripts/provision-openblas.sh                 # every host in .keel-hosts
#   scripts/provision-openblas.sh vesta janus     # named hosts
#   scripts/provision-openblas.sh --check         # verify only, install nothing
#   scripts/provision-openblas.sh --yes           # skip the per-host confirmation
#
# Environment:
#   KEEL_GO_VERSION   the toolchain to install if the host has none new enough
#                     (default go1.26.5; must support GOEXPERIMENT=simd)
#   KEEL_GO_SHA256    the tarball digest to enforce. If unset, the digest is taken
#                     from go.dev/dl/?mode=json -- see verify_go_tarball for exactly
#                     what that does and does not prove.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh

GO_VERSION="${KEEL_GO_VERSION:-go1.26.5}"
GO_MIN_MINOR=26            # 1.26 is the first release with the simd experiment
GO_TARBALL="$GO_VERSION.linux-amd64.tar.gz"
SRC_DIR="${KEEL_OPENBLAS_DIR:-/tmp/keel-openblas-src}"

# Interactive options: no -n (sudo needs stdin) and -t (sudo wants a tty).
SSH_TTY_OPTS=(-t -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new)

CHECK_ONLY=0
ASSUME_YES=0
ARGS=()
for a in "$@"; do
  case "$a" in
    --check) CHECK_ONLY=1 ;;
    --yes|-y) ASSUME_YES=1 ;;
    -h|--help) sed -n '5,50p' "$0"; exit 0 ;;
    -*) echo "unknown option: $a" >&2; exit 2 ;;
    *) ARGS+=("$a") ;;
  esac
done

if [[ "${#ARGS[@]}" -gt 0 ]]; then
  HOSTS="$(printf '%s\n' "${ARGS[@]}")"
else
  HOSTS="$(remote_hosts)"
fi
if [[ -z "$HOSTS" ]]; then
  echo "no hosts: name them as arguments, in .keel-hosts, or in \$KEEL_REMOTE_HOSTS" >&2
  exit 2
fi

say()  { printf '\033[1m==\033[0m %s\n' "$1"; }
note() { printf '   %s\n' "$1"; }
bad()  { printf '   \033[31m!!\033[0m %s\n' "$1"; }

# confirm PROMPT — ask before changing somebody's machine, unless --yes.
#
# Reads from /dev/tty, not from stdin, and the reason is issue #28: the host loop at
# the bottom of this script used to be fed on stdin, so this `read` consumed the NEXT
# HOST as its answer. Three hosts named meant one was silently never visited, having
# been spent as a keystroke — and the interactive path, the whole point of this
# script, had therefore never once worked. The loop now reads on fd 3 as well; either
# fix alone leaves the other half of the collision in place, because `ssh -t` in the
# install functions also reads stdin to let sudo prompt.
#
# No tty and no --yes is a distinct outcome from a refusal, and says so. Printing
# "skipped" when nobody was asked reports a decision that was never made, which is
# the failure mode this script exists to avoid rather than commit. confirm() prints
# that outcome word itself, so callers say `confirm ... || return 1` and cannot
# describe a refusal that did not happen: the caller does not know which of the two
# it got, and the first version of this fix proved that by printing "skipped" under
# the message explaining that nothing had been asked.
confirm() {
  [[ "$ASSUME_YES" -eq 1 ]] && return 0
  local reply
  # Opened rather than tested with -r: /dev/tty exists and is readable by that test
  # even where there is no controlling terminal to open (bash then fails the redirect
  # with "Device not configured" on its own stderr, which is noise on top of a wrong
  # message). Try the open, quietly, and report what is actually true.
  if ! { : </dev/tty; } 2>/dev/null; then
    bad "no terminal to ask on, and --yes was not given, so consent cannot be obtained"
    bad "run this from a terminal, or pass --yes if you have already read what it does"
    return 1
  fi
  read -r -p "   $1 [y/N] " reply </dev/tty || {
    bad "could not read an answer from the terminal"
    return 1
  }
  [[ "$reply" == [yY]* ]] && return 0
  note "skipped: declined at the prompt"
  return 1
}

# probe HOST — distro id, go version, libopenblas path, governor.
#
# `go=` is what the gate sees: the toolchain on a non-interactive, non-login ssh
# PATH, found the same way gate-p3.sh's ob_preflight finds it. A prerequisite this
# script calls satisfied and the gate calls missing would be the worst outcome here.
#
# `goat=` is what is INSTALLED at /usr/local/go, which is a different question and
# the one that decides install-versus-symlink (issue #27). Conflating them cost
# antares a `sudo rm -rf /usr/local/go` on a working go1.26.5 that was merely
# unlinked: "no usable toolchain" and "usable toolchain, wrong PATH" need different
# repairs, and only the first one justifies deleting anything.
probe() {
  ssh -n -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new "$1" '
    distro=unknown
    [ -r /etc/os-release ] && distro=$(sed -n "s/^ID=//p" /etc/os-release | tr -d \")
    go=none
    command -v go >/dev/null 2>&1 && go=$(go version | cut -d" " -f3)
    goat=none
    [ -x /usr/local/go/bin/go ] && goat=$(/usr/local/go/bin/go version | cut -d" " -f3)
    lib=none
    for d in /usr/lib64 /usr/lib/x86_64-linux-gnu /usr/lib /usr/local/lib; do
      if [ -e "$d/libopenblas.so" ]; then lib="$d/libopenblas.so"; break; fi
    done
    gov=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo unknown)
    printf "distro=%s go=%s goat=%s lib=%s governor=%s\n" "$distro" "$go" "$goat" "$lib" "$gov"
  ' 2>/dev/null
}

fieldof() {
  awk -v k="$1" '{
    for (i = 1; i <= NF; i++) {
      n = index($i, "=")
      if (n && substr($i, 1, n - 1) == k) { print substr($i, n + 1); exit }
    }
  }' <<<"$2"
}

# go_new_enough VERSION — "go1.26.5" or newer on the 1.x line.
go_new_enough() {
  awk -v v="$1" -v min="$GO_MIN_MINOR" 'BEGIN {
    if (v !~ /^go1\./) exit 1
    sub(/^go1\./, "", v)
    split(v, p, ".")
    exit !(p[1] + 0 >= min)
  }'
}

# install_openblas HOST DISTRO — the distribution package, not a source build. The
# ruling asks for a pinned, recorded version; a distro package is both (it is
# recorded by the marker) and it is what any reader of these numbers can reproduce.
install_openblas() {
  local host="$1" distro="$2" cmd
  case "$distro" in
    ubuntu|debian|pop|linuxmint) cmd="sudo apt-get update && sudo apt-get install -y libopenblas-dev" ;;
    rhel|centos|rocky|almalinux) cmd="sudo dnf install -y openblas-devel" ;;
    fedora)                      cmd="sudo dnf install -y openblas-devel" ;;
    *) bad "unrecognized distro id '$distro'; install an OpenBLAS development package by hand"; return 1 ;;
  esac
  note "on $host: $cmd"
  confirm "run it?" || return 1
  # shellcheck disable=SC2029  # $cmd is this script's own string, expanded here
  ssh "${SSH_TTY_OPTS[@]}" "$host" "$cmd"
}

# verify_go_tarball FILE — enforce a digest on the downloaded toolchain.
#
# With $KEEL_GO_SHA256 set, that digest is the pin and this is a real check against
# a value chosen away from the download. Without it, the digest comes from
# go.dev/dl/?mode=json, i.e. from the same origin as the tarball: that catches a
# truncated or mirrored download and proves nothing about go.dev itself. Which of
# the two happened is printed, because the difference matters and is invisible
# otherwise. No digest is hard-coded in this file: a constant nobody verified is
# worse than an honest description of a weaker check.
verify_go_tarball() {
  local file="$1" want="${KEEL_GO_SHA256:-}" got src
  got="$(shasum -a 256 "$file" 2>/dev/null | cut -d' ' -f1)"
  [[ -n "$got" ]] || got="$(sha256sum "$file" | cut -d' ' -f1)"
  if [[ -n "$want" ]]; then
    src="\$KEEL_GO_SHA256 (an independent pin)"
  else
    src="go.dev/dl?mode=json (same origin as the download: integrity, not provenance)"
    # `tr -d '\n'` first, and it is not cosmetic (issue #29): go.dev serves
    # pretty-printed JSON, so splitting on '{' alone leaves the object's fields on
    # separate lines and the line-oriented `grep` returns only the "filename" line —
    # the sha256 is on a neighbour and never reaches the sed. $want was therefore
    # empty for every version ever requested, and this check had never once run.
    want="$(curl -fsSL 'https://go.dev/dl/?mode=json&include=all' |
      tr -d '\n' | tr '{' '\n' | grep -F "\"$GO_TARBALL\"" |
      sed -n 's/.*"sha256": *"\([0-9a-f]\{64\}\)".*/\1/p' | head -1)"
  fi
  if [[ -z "$want" ]]; then
    bad "could not obtain a published digest for $GO_TARBALL"
    bad "the file downloaded here is sha256 $got"
    bad "compare it against go.dev/dl and re-run with KEEL_GO_SHA256=$got to proceed"
    return 1
  fi
  if [[ "$got" != "$want" ]]; then
    bad "digest mismatch for $GO_TARBALL: got $got, expected $want"
    return 1
  fi
  note "digest ok ($got) against $src"
}

# link_go HOST VERSION — put an already-installed /usr/local/go on the system PATH.
#
# The cheap repair, and the correct one when the host already has a new-enough
# toolchain that is merely unlinked (issue #27). It touches two symlinks and deletes
# nothing: replacing a working toolchain with a fresh copy of the same version is
# risk without benefit, and the risk lands on somebody else's machine.
link_go() {
  local host="$1"
  note "on $host: /usr/local/go is already $2, which is new enough; linking it onto the PATH"
  confirm "create /usr/local/bin/{go,gofmt} symlinks?" || return 1
  ssh "${SSH_TTY_OPTS[@]}" "$host" '
    set -e
    sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
    sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
    /usr/local/bin/go version
  '
}

# install_go HOST — $GO_VERSION into /usr/local/go and onto the system PATH.
#
# /etc/profile.d only helps interactive logins, and the gate builds over a
# non-interactive, non-login ssh, so the symlinks in /usr/local/bin are the part
# that actually matters. This is the same reason the gate's preflight looks up `go`
# the way ssh does rather than the way a human does.
#
# This DELETES /usr/local/go, so the caller must have established that what is there
# is absent or too old (issue #27). The deletion is named in the prompt for the same
# reason: it is the only irreversible thing this script does to a host.
install_go() {
  local host="$1" tmp
  note "on $host: install $GO_VERSION into /usr/local/go"
  [[ "${2:-none}" == none ]] ||
    note "this DELETES the existing /usr/local/go ($2), which cannot do GOEXPERIMENT=simd"
  confirm "download and install it?" || return 1
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  say "downloading $GO_TARBALL"
  curl -fSL --progress-bar -o "$tmp/$GO_TARBALL" "https://go.dev/dl/$GO_TARBALL" || return 1
  verify_go_tarball "$tmp/$GO_TARBALL" || return 1
  scp -q "${KEEL_SCP_OPTS[@]}" "$tmp/$GO_TARBALL" "$host:/tmp/$GO_TARBALL" || return 1
  # shellcheck disable=SC2029  # $GO_TARBALL expands here on purpose
  ssh "${SSH_TTY_OPTS[@]}" "$host" "
    set -e
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/$GO_TARBALL
    sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
    sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
    rm -f /tmp/$GO_TARBALL
    /usr/local/bin/go version
  "
}

# verify HOST — build and run the openblas-tagged harness, and print the marker.
#
# This is the check that matters, and it is deliberately the gate's own: same build
# tag, same GOMAXPROCS=1 and OPENBLAS_NUM_THREADS=1 environment, same marker. A
# short -benchtime, because the question here is what the reference IS, not how fast
# it is; the gate measures the rate under §5.4's methodology.
verify() {
  local host="$1" out
  if [[ -n "$(git status --porcelain)" ]]; then
    bad "working tree is dirty; \`git archive HEAD\` would ship something other than what is here"
    return 1
  fi
  # shellcheck disable=SC2029  # client-side expansion of this script's own paths
  git archive --format=tar HEAD | ssh -o BatchMode=yes -o ConnectTimeout=10 \
      -o StrictHostKeyChecking=accept-new "$host" \
      "rm -rf '$SRC_DIR' && mkdir -p '$SRC_DIR' && tar -x -C '$SRC_DIR'" || return 1
  # shellcheck disable=SC2029  # client-side expansion of this script's own paths
  out="$(ssh -n -o BatchMode=yes -o ConnectTimeout=30 -o StrictHostKeyChecking=accept-new "$host" "
    cd '$SRC_DIR' &&
    GOEXPERIMENT=simd CGO_ENABLED=1 go test -c -tags openblas -o bench-ob.test ./bench &&
    env GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 \
      ./bench-ob.test -test.run=NONE -test.bench='OpenBLAS/n=2048' -test.benchtime=3x -test.count=1
  " 2>&1)" || {
    bad "the openblas-tagged harness did not build or run:"
    # shellcheck disable=SC2001  # prefixing every line, not a scalar substitution
    sed 's/^/      /' <<<"$out" | tail -20
    return 1
  }
  local marker core threads
  marker="$(sed -n 's/.*keel-bench-openblas: *//p' <<<"$out" | tail -1)"
  if [[ -z "$marker" || "$marker" == *"not available"* ]]; then
    bad "no OpenBLAS marker despite the build tag"
    return 1
  fi
  note "reference: $marker"
  core="$(fieldof corename "$marker")"
  threads="$(fieldof threads "$marker")"
  local ok=0
  # The gate's own allowlist, read from the gate so the two cannot drift apart.
  local allow
  allow="$(sed -n 's/^OPENBLAS_OK_CORES="\(.*\)"$/\1/p' scripts/gate-p3.sh)"
  for c in $allow; do [[ "$(tr '[:upper:]' '[:lower:]' <<<"$core")" == "$c" ]] && ok=1; done
  [[ "$threads" == "1" ]] || { bad "OPENBLAS_NUM_THREADS=1 did not take effect (threads=$threads)"; return 1; }
  if [[ "$ok" -eq 0 ]]; then
    bad "kernel family '$core' is not on the gate's AVX2-or-better allowlist ($allow)"
    bad "a reference slower than this host can be would inflate keel's ratio; the gate will refuse it"
    return 1
  fi
  note "OPENBLAS_NUM_THREADS=1 enforced, kernel family $core on the allowlist"
  note "$(sed -n 's/.*keel-bench-cpu: *//p' <<<"$out" | tail -1)"
  say "$host: ready"
}

RC=0
# The host list is read on fd 3, leaving stdin as the terminal (issue #28). Two things
# in this loop need stdin: confirm(), which asks the operator, and the `ssh -t` calls
# in install_openblas/install_go/link_go, which need a tty for sudo to prompt on. With
# `done <<<"$HOSTS"` both of those were reading the host list instead.
while read -r -u 3 host; do
  [[ -n "$host" ]] || continue
  say "$host"
  p="$(probe "$host")"
  if [[ -z "$p" ]]; then
    bad "cannot reach $host over ssh with BatchMode=yes (the gate uses the same)"
    RC=1
    continue
  fi
  distro="$(fieldof distro "$p")"; ver="$(fieldof go "$p")"
  atver="$(fieldof goat "$p")"
  lib="$(fieldof lib "$p")";       gov="$(fieldof governor "$p")"
  note "distro=$distro go=$ver (/usr/local/go=$atver) libopenblas=$lib governor=$gov"
  if [[ "$gov" != performance ]]; then
    note "governor is $gov. DESIGN.md §5.4 rule 5 needs at least one gate host on"
    note "performance; this script does not change a machine's power policy. To set it:"
    note "  sudo cpupower frequency-set -g performance   (or write scaling_governor)"
  fi

  if [[ "$lib" == none ]]; then
    if [[ "$CHECK_ONLY" -eq 1 ]]; then
      bad "no libopenblas.so (re-run without --check to install it)"
      RC=1; continue
    fi
    install_openblas "$host" "$distro" || { RC=1; continue; }
  fi
  if [[ "$ver" == none ]] || ! go_new_enough "$ver"; then
    if [[ "$CHECK_ONLY" -eq 1 ]]; then
      bad "go on the ssh PATH is $ver, and the harness needs $GO_VERSION or newer (re-run without --check)"
      [[ "$atver" != none ]] && go_new_enough "$atver" &&
        bad "note: /usr/local/go is already $atver; it only needs linking onto the PATH (#27)"
      RC=1; continue
    fi
    # Two different repairs for two different states (#27): link what is there if it
    # is new enough, install only when there is nothing usable to link.
    if [[ "$atver" != none ]] && go_new_enough "$atver"; then
      link_go "$host" "$atver" || { RC=1; continue; }
    else
      install_go "$host" "$atver" || { RC=1; continue; }
    fi
  fi
  verify "$host" || RC=1
done 3<<<"$HOSTS"

echo
if [[ "$RC" -eq 0 ]]; then
  echo "all hosts have a same-host OpenBLAS reference the gate can use"
else
  echo "some hosts are not ready; scripts/gate-p3.sh will fail those hosts by name" >&2
fi
exit "$RC"
