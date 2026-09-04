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
#                     (default go1.27.0; must build the tree under GOEXPERIMENT=simd,
#                     which since T23's archsimd rename means 1.27.x and not merely
#                     "a toolchain with the experiment" — see main() for the two arms)
#   KEEL_GO_SHA256    the tarball digest to enforce. If unset, the digest is taken
#                     from go.dev/dl/?mode=json -- see verify_go_tarball for exactly
#                     what that does and does not prove.
set -euo pipefail

# Everything below is a function definition and the last line of the file is
# `main "$@"`. Bash reads a script incrementally as it executes it, so a script
# that does its work at the top level can be corrupted by an edit that lands
# mid-run: the parser resumes at a byte offset that now holds different text. This
# script changes somebody's machines over an interactive sudo, so a run of it
# resuming into different text is the worst version of that hazard in this repo.
# Defining everything before anything runs forces one whole-file parse before the
# first host is touched (#51).
#
# The configuration variables are assigned in main without `local`, so the install
# and verify helpers see them however they are reached.

say()  { printf '\033[1m==\033[0m %s\n' "$1"; }
note() { printf '   %s\n' "$1"; }
bad()  { printf '   \033[31m!!\033[0m %s\n' "$1"; }

# APT_WAIT — prefixed to every apt command below. The launcher's `cloud-init status
# --wait` is necessary and not sufficient: it returned done while cloud-init ran apt for
# six more minutes and provisioning died on the lists lock (CHANGELOG has the timestamps
# and the two rejected fixes, `DPkg::Lock::Timeout` and `flock`, with why each is inert).
# Gate on the LOCK, not on cloud-init — three guesses at the holder were each refuted by
# the journal, and this is right without knowing. 300s then EX_TEMPFAIL, so a host that
# cannot start apt fails by name instead of dying inside it.
APT_WAIT='w=0; while sudo fuser /var/lib/apt/lists/lock >/dev/null 2>&1; do
  w=$((w+5)); [ "$w" -le 300 ] || { echo "apt lists lock still held after ${w}s" >&2; exit 75; }
  sleep 5; done;'

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
# The governor half is $KEEL_GOV_PROBE_SH from remote.sh, concatenated in rather than
# written here, and the output field is named `governor=` so that assert_governor parses
# this line as readily as remote_probe's. This script used to read the knob with its own
# `cat ... || echo unknown`, which called an AWS guest `unknown` where remote.sh calls it
# `absent` — one host, two readings, and only one of them a cause a gate can act on.
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
  ssh -n -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new "$1" "$KEEL_GOV_PROBE_SH"'
    distro=unknown
    [ -r /etc/os-release ] && distro=$(sed -n "s/^ID=//p" /etc/os-release | tr -d \")
    go=none
    command -v go >/dev/null 2>&1 && go=$(go version | cut -d" " -f3)
    goat=none
    [ -x /usr/local/go/bin/go ] && goat=$(/usr/local/go/bin/go version | cut -d" " -f3)
    arch=$(uname -m)
    lib=none
    # /usr/local/lib FIRST (#137): the arm64 reference is a DYNAMIC_ARCH source build that
    # installs there, and it must be the one found even when a distro package also exists in
    # the triplet dir. Harmless on amd64, where the distro path is the only populated one and
    # /usr/local/lib is empty by convention here.
    for d in /usr/local/lib /usr/lib64 /usr/lib/x86_64-linux-gnu /usr/lib/aarch64-linux-gnu /usr/lib; do
      if [ -e "$d/libopenblas.so" ]; then lib="$d/libopenblas.so"; break; fi
    done
    cc=none
    for c in cc gcc clang; do
      if command -v "$c" >/dev/null 2>&1; then cc=$c; break; fi
    done
    printf "distro=%s arch=%s go=%s goat=%s lib=%s cc=%s governor=%s\n" "$distro" "$arch" "$go" "$goat" "$lib" "$cc" "$gov"
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

# go_new_enough VERSION — minor >= $GO_MIN_MINOR on the 1.x line. Minor only, so it
# reads "1.27.x" and not "go1.27.0 exactly": the rename this floor tracks landed in the
# minor, so any patch release of it builds the tree.
go_new_enough() {
  awk -v v="$1" -v min="$GO_MIN_MINOR" 'BEGIN {
    # A PRERELEASE IS REFUSED, and this is the ruling and not a taste. #70, 2026-08-16:
    # "rc3 is not admitted" — admitting one means a green certifying a floor the
    # toolchain does not name. The old pattern was `^go1\.` and split() on ".", so
    # `go1.27rc3` yielded a minor of 27 and passed; janus and antares carry exactly that
    # version alongside their /usr/local/go, so the hole had a host to bite. Anchored at
    # both ends, digits only.
    if (v !~ /^go1\.[0-9]+(\.[0-9]+)?$/) exit 1
    sub(/^go1\./, "", v)
    split(v, p, ".")
    exit !(p[1] + 0 >= min)
  }'
}

# install_openblas HOST DISTRO — the distribution package, not a source build. The
# ruling asks for a pinned, recorded version; a distro package is both (it is
# recorded by the marker) and it is what any reader of these numbers can reproduce.
# install_cc HOST DISTRO — a C toolchain, which the harness needs and the package
# above does not pull in.
#
# Separate from install_openblas because the two are independently absent: the AWS
# guests came up with libopenblas-dev installed and no compiler at all, so a run that
# only installed the library on a fresh host reached `verify` and died there with
# `cgo: C compiler "gcc" not found`. bench/openblas.go is a cgo file, so a C compiler is
# as much a prerequisite as the library, and this script's whole premise is that a
# prerequisite is found missing here rather than in a benchmark. The desktop fleet hid
# it -- those were developer machines and had gcc for other reasons.
install_cc() {
  local host="$1" distro="$2" cmd
  case "$distro" in
    ubuntu|debian|pop|linuxmint)        cmd="$APT_WAIT sudo apt-get update && sudo apt-get install -y build-essential" ;;
    rhel|centos|rocky|almalinux|fedora) cmd="sudo dnf install -y gcc glibc-devel" ;;
    *) bad "unrecognized distro id '$distro'; install a C compiler by hand (cgo needs one)"; return 1 ;;
  esac
  note "on $host: $cmd"
  confirm "run it?" || return 1
  # shellcheck disable=SC2029  # $cmd is this script's own string, expanded here
  ssh "${SSH_TTY_OPTS[@]}" "$host" "$cmd"
}

install_openblas() {
  local host="$1" distro="$2" cmd
  case "$distro" in
    ubuntu|debian|pop|linuxmint) cmd="$APT_WAIT sudo apt-get update && sudo apt-get install -y libopenblas-dev" ;;
    rhel|centos|rocky|almalinux) cmd="sudo dnf install -y openblas-devel" ;;
    fedora)                      cmd="sudo dnf install -y openblas-devel" ;;
    *) bad "unrecognized distro id '$distro'; install an OpenBLAS development package by hand"; return 1 ;;
  esac
  note "on $host: $cmd"
  confirm "run it?" || return 1
  # shellcheck disable=SC2029  # $cmd is this script's own string, expanded here
  ssh "${SSH_TTY_OPTS[@]}" "$host" "$cmd"
}

# build_openblas_arm64 HOST — the same-host reference on Graviton, built FROM SOURCE with
# DYNAMIC_ARCH=1 so every arm64 kernel family is present in one library and the load-time
# OPENBLAS_CORETYPE knob can select among them. gate-p3.sh's ob_coretype_sweep is the
# consumer, and the spread it measures across ARMV8 (NEON) and NEOVERSEV1/V2 (SVE) IS the
# SVE≈NEON deliverable #137 charters (docs/neon-sweep.md's fleet half).
#
# Source and not the distro package (the amd64 path's choice) for one measured reason, the
# same one openblasCorename() guards on x86: Ubuntu 24.04 ships OpenBLAS 0.3.26, whose arm64
# DYNAMIC_ARCH set predates the Neoverse V2 / SVE2 tuning this campaign measures against. A
# reference missing the host's best kernel reads LOW, which inflates keel's ratio — the one
# direction a denominator must never err. The version is pinned ($OPENBLAS_VERSION, a tag)
# and recorded by the marker, so it is as reproducible as a package.
#
# The distro package is REMOVED first (|| true — it is usually absent on the stock AMI): a
# `-lopenblas` with no `-L` resolves by ld's search order, and two libopenblas.so with the
# same SONAME is the one way this links the wrong one SILENTLY. One library on the host means
# the reference cannot be anything but the source build, which verify() then confirms by
# reading the version back out of the marker. USE_OPENMP=0 and the runtime OPENBLAS_NUM_THREADS=1
# keep it single-threaded exactly as the amd64 reference is; NUM_THREADS is built to nproc so
# the cap, not the build, sets the count.
OPENBLAS_VERSION="${KEEL_OPENBLAS_VERSION:-v0.3.29}"
OPENBLAS_BUILD_DIR="/tmp/keel-openblas-build"
build_openblas_arm64() {
  local host="$1" cmd
  note "on $host: build OpenBLAS $OPENBLAS_VERSION from source (DYNAMIC_ARCH=1) into /usr/local"
  note "  arm64 source build, not the distro package: 24.04's 0.3.26 predates the Neoverse V2/SVE2"
  note "  kernels this reference must not read below (openblasCorename's guard, on arm64)"
  confirm "remove any distro openblas, install build deps, clone $OPENBLAS_VERSION, build and 'sudo make install' it?" || return 1
  cmd="$APT_WAIT sudo apt-get remove -y libopenblas0 libopenblas-dev libopenblas0-pthread 2>/dev/null || true
    $APT_WAIT sudo apt-get update && sudo apt-get install -y gcc gfortran make git
    rm -rf '$OPENBLAS_BUILD_DIR'
    git clone --depth 1 --branch '$OPENBLAS_VERSION' https://github.com/OpenMathLib/OpenBLAS '$OPENBLAS_BUILD_DIR'
    make -C '$OPENBLAS_BUILD_DIR' -j\"\$(nproc)\" DYNAMIC_ARCH=1 TARGET=ARMV8 USE_OPENMP=0 NUM_THREADS=\"\$(nproc)\" FC=gfortran CC=gcc
    sudo make -C '$OPENBLAS_BUILD_DIR' PREFIX=/usr/local NO_STATIC=1 install
    sudo ldconfig
    rm -rf '$OPENBLAS_BUILD_DIR'"
  # shellcheck disable=SC2029  # $cmd is this script's own string, expanded here
  ssh "${SSH_TTY_OPTS[@]}" "$host" "set -e; $cmd"
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
# it is; the gate measures the rate under §5 rule 5's methodology.
verify() {
  local host="$1" goarch="${2:-}" out
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
  # On arm64 the reference MUST be the source build, not a distro libopenblas that won the
  # `-lopenblas` link (#137). openblas_get_config() carries the version, so read it back: a
  # missing version string means the wrong library linked despite build_openblas_arm64's
  # removal of the distro copy, and that is a silently-wrong denominator this must catch.
  if [[ "$goarch" == arm64 ]]; then
    local wantver="${OPENBLAS_VERSION#v}"
    if [[ "$marker" != *"$wantver"* ]]; then
      bad "the linked OpenBLAS is not the pinned source build: the marker names no '$wantver' ($OPENBLAS_VERSION)"
      bad "a distro libopenblas.so likely won the link; check that /usr/local/lib is on ldconfig and no package remains"
      return 1
    fi
    note "linked reference confirmed as the $OPENBLAS_VERSION source build"
  fi
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

main() {
  cd "$(dirname "$0")/.."
  # shellcheck source=scripts/remote.sh
  source scripts/remote.sh

  # go1.27.0, and the floor moved with it (2026-08-28). 1.26 WAS the first release with
  # the simd experiment, which is why the floor was 26 — but T23 swapped the archsimd
  # names, so from fed1e70 the tree does not compile on 1.26 at all. Two natively-built
  # fleet arms are the ones that care, because they are the only ones a host's own
  # toolchain compiles: gate-p5's `-race` leg (cgo, so it cannot be cross-built) and
  # gate-p3's openblas-tagged harness (cgo likewise). Both died on antares' go1.26.5 in
  # the p5-preflight-1689d0b run, the second one as `cannot use bp[0:16] (value of type
  # []float32) as *[16]float32 value in argument to archsimd.LoadFloat32x16` — the 1.26
  # compiler rejecting 1.27-form source. That took gate-p3 RED, gate-p4 RED with it, and
  # gate-p5 RED, so a stale default here is what stands between the fleet and a P5
  # certificate.
  #
  # `go_new_enough` is the sharper half. It gates on the MINOR only, so a host already
  # carrying 1.26.5 read as "new enough" and was linked rather than upgraded — the
  # provisioner would have declared a host ready for a harness it cannot build. Bumping
  # the default alone would have left that path intact.
  GO_VERSION="${KEEL_GO_VERSION:-go1.27.0}"
  GO_MIN_MINOR=27            # not 26: 1.26 predates T23's archsimd rename (#70)
  # GO_TARBALL is set PER HOST from its `uname -m` (#137): the arm64 Graviton fleet needs
  # linux-arm64, the amd64 fleet linux-amd64, and a fleet may mix the two. verify_go_tarball
  # and install_go read it as a global, so the loop assigns it before either is reached.
  SRC_DIR="${KEEL_OPENBLAS_DIR:-/tmp/keel-openblas-src}"

  # Interactive options: no -n (sudo needs stdin) and -t (sudo wants a tty).
  SSH_TTY_OPTS=(-t -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new)

  CHECK_ONLY=0
  ASSUME_YES=0
  local a p distro ver atver lib cc arch goarch HOSTS host
  local -a ARGS=()
  for a in "$@"; do
    case "$a" in
      --check) CHECK_ONLY=1 ;;
      --yes|-y) ASSUME_YES=1 ;;
      # The usage block ends at the `set -euo pipefail` line, and the wrapper
      # comment below it is about this file's shape rather than its interface, so
      # --help stops before it. It used to print to line 50, which reached a few
      # lines into the code.
      -h|--help) sed -n '5,43p' "$0"; exit 0 ;;
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

  RC=0
  # The host list is read on fd 3, leaving stdin as the terminal (issue #28). Two things
# in this loop need stdin: confirm(), which asks the operator, and the `ssh -t` calls
# in install_openblas/install_go/link_go, which need a tty for sudo to prompt on. With
# `done <<<"$HOSTS"` both of those were reading the host list instead.
  while read -r -u 3 host; do
    say "$host"
    p="$(probe "$host")"
    if [[ -z "$p" ]]; then
      bad "cannot reach $host over ssh with BatchMode=yes (the gate uses the same)"
      RC=1
      continue
    fi
    distro="$(fieldof distro "$p")"; ver="$(fieldof go "$p")"
    atver="$(fieldof goat "$p")"
    lib="$(fieldof lib "$p")";       cc="$(fieldof cc "$p")"
    arch="$(fieldof arch "$p")"
    case "$arch" in
      aarch64|arm64) goarch=arm64 ;;
      x86_64|amd64)  goarch=amd64 ;;
      *) bad "unrecognized machine arch '$arch' from uname -m; cannot pick a go tarball or reference build"; RC=1; continue ;;
    esac
    GO_TARBALL="$GO_VERSION.linux-$goarch.tar.gz"
    # Classified by assert_governor, not by comparing the token here: it owns the
    # five-state taxonomy, and a second reader of one field is how the same knob came to
    # have two meanings in this tree. Only the parse is wanted, so its verdict lines go
    # to /dev/null — this script prints `bad`/`note`, and a gate PASS stamp from a
    # provisioning run would be a certificate nobody earned.
    assert_governor "$host" preamble "$p" >/dev/null 2>&1
    note "distro=$distro go=$ver (/usr/local/go=$atver) libopenblas=$lib cc=$cc governor=$GOV_SHOWN"
    # A non-performance governor is a failure, not a note (ruling with issue #31). It
    # used to say (citation-lint:quote) "§5.4 rule 5 needs at least one gate host on performance", which
    # was true of the old gate and let antares sit on `powersave` while contributing
    # numbers: its first OpenBLAS reading was 245.0 GFLOP/s against a 296-297 steady
    # state. gate-p3.sh fails any host that is not on performance, so a host in that
    # state is not provisioned-and-ready, it is provisioned-and-unusable — and this
    # script exists to find that out before a session is spent on it.
    #
    # Still reported rather than changed: a power policy is a standing property of
    # somebody's machine, not a thing a provisioning script should quietly flip and
    # leave flipped.
    #
    # `nocpufreq` is the one state this script must NOT fail, and the reason is not that
    # the fleet is now guests — it is that provisioning has no sweep. §5 rule 5 as
    # amended asks for a clock established stable by whichever instrument the host has,
    # and on a guest that instrument is BenchmarkPeak sampled head/middle/tail, which is
    # a property of a twenty-minute measurement and cannot be read at install time by
    # anything. So the honest output here is which verdict is being deferred and to
    # whom, rather than `unknown, not performance` — which is what this printed for all
    # three AWS guests, naming a defect on hosts that simply have no knob.
    GOV_OK=1
    if [[ "$GOV_STATE" == nocpufreq ]]; then
      note "no cpufreq interface, so there is no governor to set and none to check: this"
      note "  host's clock is established by the gate's peak dispersion instead, which"
      note "  needs a sweep and so cannot be read from here. Deferred, not passed."
    elif [[ "$GOV_STATE" != performance ]]; then
      GOV_OK=0
      bad "governor is $GOV_SHOWN, and the gate fails any host not on performance (DESIGN.md §5 rule 5)"
      note "this script does not change a machine's power policy. To set it:"
      note "  sudo cpupower frequency-set -g performance"
      note "  or: echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor"
    fi

    # OpenBLAS: the amd64 reference is the distro package (reproducible, recorded by the
    # marker); the arm64 reference is a DYNAMIC_ARCH SOURCE build (#137), forced even when a
    # distro package is present because 0.3.26 predates the Neoverse V2/SVE2 kernels — see
    # build_openblas_arm64. `lib` under /usr/local means the source build is already there;
    # anything else on arm64 (none, or a distro path) triggers the source build, which removes
    # the distro copy so the reference cannot be ambiguous.
    if [[ "$goarch" == arm64 ]]; then
      if [[ "$lib" != /usr/local/* ]]; then
        if [[ "$CHECK_ONLY" -eq 1 ]]; then
          bad "no /usr/local source OpenBLAS (lib=$lib); re-run without --check to build it from source"
          RC=1; continue
        fi
        build_openblas_arm64 "$host" || { RC=1; continue; }
        lib=/usr/local/lib/libopenblas.so   # what the build just installed; verify() re-reads it live
      fi
    elif [[ "$lib" == none ]]; then
      if [[ "$CHECK_ONLY" -eq 1 ]]; then
        bad "no libopenblas.so (re-run without --check to install it)"
        RC=1; continue
      fi
      install_openblas "$host" "$distro" || { RC=1; continue; }
    fi
    if [[ "$cc" == none ]]; then
      if [[ "$CHECK_ONLY" -eq 1 ]]; then
        bad "no C compiler on the PATH, and bench/openblas.go is a cgo file (re-run without --check)"
        RC=1; continue
      fi
      install_cc "$host" "$distro" || { RC=1; continue; }
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
    verify "$host" "$goarch" || RC=1
    # After verify(), so that a host whose library and toolchain are fine but whose
    # governor is wrong ends on the reason it still cannot be measured, rather than on
    # "ready". The installs above are not skipped for it: the package work is valid and
    # worth keeping, and re-running once the governor is set should have nothing left
    # to do but re-verify.
    if [[ "$GOV_OK" -eq 0 ]]; then
      bad "$host is provisioned but not measurable: governor=$GOV_SHOWN, not performance"
      RC=1
    fi
  done 3< <(hosts_lines)

  echo
  if [[ "$RC" -eq 0 ]]; then
    echo "all hosts have a same-host OpenBLAS reference the gate can use"
  else
    echo "some hosts are not ready; scripts/gate-p3.sh will fail those hosts by name" >&2
  fi
  exit "$RC"
}

main "$@"
