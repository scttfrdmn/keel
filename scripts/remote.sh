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

# unmeasured MESSAGE — the gate is not green, and the log says why it is not a
# miss. Sets FAIL exactly as each gate's own `fail` does, so a criterion that
# reports this blocks the gate identically; what differs is what the red asserts
# about its cause.
#
# WHY THIS ONE PRIMITIVE LIVES HERE AND pass/fail DO NOT (#72). It is the only one
# more than one gate needs and none had. It was written for gate-p4's criterion 7
# under the #67 ruling, and then 21 further sites across gate-p3, gate-p4 and
# gate-p5 turned out to need it — sites whose own message text already said
# "unmeasured" while the label printed FAIL. Two of them are worth remembering as
# the shape of the defect: bench_expect's docs in scripts/bench.sh say an absent
# measurement has "exactly one verdict available to it — unmeasured" six lines
# above a caller printing FAIL, and gate-p5's race_verdict header argues that
# collapsing its states "sends whoever reads it looking for a race that is not
# there" immediately above the branches that collapse them. The comments knew the
# answer before the code did.
#
# Landing that fix by copying the definition into gate-p5.sh would have made three
# copies of a verdict primitive, and divergent verdict primitives across gates is
# how the delegated tally came to count two columns where the log had three. So it
# lives in the file all six gates already source. pass and fail stay per-gate for
# now: they are identical in all six and lifting them is a separate change with no
# defect behind it, which is not the kind of change to make while landing one.
#
# RELABELING IS NOT AMENDMENT, and the distinction is exact (ruled 2026-08-15).
# What makes a criterion green is untouched — there is no bucket here that converts
# red to not-red, and FAIL=1 is set by construction. What changes is the attributed
# cause, and DESIGN.md §5.6 is explicit that a gate red for the wrong reason is as
# untrustworthy as a gate green for the wrong reason. A FAIL on a race criterion
# asserts a race was found; when the run died before the detector could look, that
# assertion is false. Red with the right cause attached is *more* faithful to a
# hard-red criterion, not less.
#
# The word FAIL must not appear in the label: the delegating gates count verdict
# lines by grepping for it (gate-p4.sh:839, gate-p5.sh:987).
#
# WHICH OF THE TWO A SITE GETS (the three-way taxonomy, ruled 2026-08-15 on #73).
# #72 relabeled the sites whose own message text already said "unmeasured" while
# the label printed FAIL. #73 is the converse: every site that reports a *reason
# it could not look* and prints FAIL anyway. One rule decides all of them, and it
# is not the wording of the message:
#
#   FAIL        the gate obtained the reading the criterion asks for, and the
#               reading is wrong. A test that ran and failed. A build that ran
#               and failed. A value that was read and is out of bounds. A set
#               that was enumerated and is short. A count that was taken and
#               disagrees. A governor that answered 'powersave'.
#   UNMEASURED  the reading the criterion asks for does not exist. The host did
#               not answer. The run died. The marker is absent. The value is
#               unreadable. benchstat established no interval. There was no host
#               to ask.
#
# Tie-break for a site that could be argued either way: does the sentence it
# prints assert something about *keel*, or about the *measurement*? Only the
# first may be FAIL. "keel does not reach 60% of OpenBLAS" is a claim about keel,
# and a run with no hosts has not earned it.
#
# Neither is an exemption and neither is softer: both set FAIL=1 on the same
# line above, so a criterion reporting either blocks the gate identically. This
# is what "an unchecked precondition is not a met one" means once UNMEASURED
# exists as a first-class verdict — the older governor rule said "unreadable
# counts as unmet", written before there was a third column, and its intent was
# that unreadability is not an exemption. That intent is preserved exactly: an
# unreadable governor still stops the gate, and stops asserting the governor was
# wrong when the truth is nobody could look.
#
# Where one branch of a site was mixed, the sweep split it rather than choosing:
# an unreadable CPU count and a CPU count that reads short are different facts
# (gate-p5.sh:352), as are a forced run that reported its dispatch
# before failing and one that never got that far (gate-p5.sh:418), and a governor
# that changed mid-run and a governor unreadable mid-run (gate-p3.sh:1178,
# gate-p4.sh:653, gate-p5.sh:643). Each citation is the line of the `if`, and the
# last of them arrived a commit later than the rest: gate-p5's copy printed
# '${gov:-unknown}' inside one collapsed branch, so a sweep that read messages
# could not see it, and #76's guard is what made the branch reachable at all.
#
# VERDICT_STAMP prefixes the message of every verdict line. It is empty in every
# real run and set only by an instrument exercise -- gate-p3.sh's
# KEEL_INSTRUMENT_WIDEN_CI (#86), gate-p2.sh's KEEL_INSTRUMENT_EXERCISE -- so that
# a synthetic log self-describes line by line and no single line of it can be
# quoted as a gate result.
#
# ALL FOUR VERDICT HELPERS LIVE HERE, AND NO GATE THAT SOURCES THIS FILE DEFINES
# ITS OWN (2026-08-16; scoped 2026-08-16 after the sentence was checked).
#
# The sentence used to read "no gate defines its own", which gate-docs.sh
# falsifies: it sources nothing and defines pass/fail/info itself. That is not the
# drift this lift was about, and it was checked rather than assumed. gate-docs
# prints a different vocabulary on purpose (`ok`, not a colored `PASS`), contacts
# no host, is delegated by nothing — the Makefile and both docs.yml jobs run the
# whole script and read its exit status — and has no instrument-exercise mode, so
# no synthetic gate-docs log exists for a line to be quoted out of and there is
# nothing for a stamp to mark. Sourcing this file there would import the ssh
# machinery into the one gate that needs none, and change output CI reads. Left
# alone deliberately; the precondition is recorded at its own definition site.
# The comment above used to say "here and in each gate's own pass/fail/info", and
# warned in the next breath that "an overridden copy is a copy and copies drift".
# Both halves were true, and the drift had already happened: `unmeasured` was
# shared and stamped, while pass/fail/info were copied into all six gates and only
# gate-p3's copy -- the one written alongside the exercise that needed it -- ever
# applied the stamp. So gate-p2's first synthetic run would have printed a PASS
# line indistinguishable from a real certificate, which is the exact forgery the
# stamp exists to prevent, in the exact place a banner does not help.
#
# The five identical copies were NOT corrected in place. Uniformity across copies
# is not correctness: five agreeing copies were the wrong ones and the odd one out
# was right, so making a sixth good copy would leave the same defect available to
# gate-p6. Lifted instead, which is why every gate sources this file before it
# would have defined these. FAIL stays per-gate: it is a counter the gate owns,
# and these helpers only ever raise it.
VERDICT_STAMP="${VERDICT_STAMP:-}"
pass()       { printf '  \033[32mPASS\033[0m  %s%s\n'        "$VERDICT_STAMP" "$1"; }
fail()       { printf '  \033[31mFAIL\033[0m  %s%s\n'        "$VERDICT_STAMP" "$1"; FAIL=1; }
unmeasured() { printf '  \033[33mUNMEASURED\033[0m  %s%s\n'  "$VERDICT_STAMP" "$1"; FAIL=1; }
info()       { printf '        %s%s\n'                       "$VERDICT_STAMP" "$1"; }

# gate_verdict NAME [DETAIL [RED_NOTE...]] — the last line of a gate log, which is the
# line a reader greps, and the exit status `detach.sh stat` records. Six copies, four
# byte-identical. DETAIL and RED_NOTE are gate-p2's and gate-p3's own wording, kept as
# parameters for the reason test_verdict's PHRASE is; PHASE is derived from NAME so the
# two cannot come to disagree.
#
# THE WITHHOLD DECIDES ON VERDICT_STAMP, NOT ON AN INSTRUMENT FLAG — why it is shareable,
# and a fix. p2 and p3 each tested their own flag, so a gate with no mode of its own had
# no branch at all, and VERDICT_STAMP is seeded from the environment just above. Measured
# at 2feb8d2: `VERDICT_STAMP='[synthetic] ' bash scripts/gate-p0.sh` stamped every
# criterion line and still signed the run `gate-p0: RED`, exit 1 — a forgeable certificate
# of exactly #78's shape, reachable from the environment. The stamp IS the synthetic-run
# signal, set in the same block that reads the flag, so deciding on it fails closed for
# every mode added later with no per-gate branch left to forget.
gate_verdict() {
  local name="$1" phase note; shift
  local detail="${1-}"; [[ $# -gt 0 ]] && shift
  phase="$(printf '%s' "${name#gate-}" | tr '[:lower:]' '[:upper:]')"
  echo
  if [[ -n "$VERDICT_STAMP" ]]; then
    echo "$name: VERDICT WITHHELD (${detail:-synthetic run: every verdict line above is stamped, so no line of this log is a gate result}; FAIL=$FAIL says which renderings fired, not whether $phase holds)"
    exit 2
  fi
  if [[ "$FAIL" -eq 0 ]]; then
    echo "$name: GREEN"
    exit 0
  fi
  echo "$name: RED" >&2
  for note in "$@"; do echo "$note" >&2; done
  exit 1
}

# assumed MESSAGE — declare a precondition the gate is TRUSTING, and add it to a
# ledger printed beside the verdict. This is not a fourth verdict. It sets no
# FAIL, it is indented as an `info` line, and the tallies cannot see it.
#
# WHY IT IS NOT A VERDICT (ruled 2026-08-15, closing #73's tier C). A precondition
# with no read-back mechanism *at all* — nothing to check, as opposed to something
# unreadable — cannot be given a verdict without destroying the distinction
# UNMEASURED exists to keep sharp. UNMEASURED means the gate could have looked and
# could not read; a bucket for "we did not look because there is nothing to look
# at" would blur exactly that, and it would blur it in the direction that matters,
# because UNMEASURED blocks the gate and an unverifiable assumption cannot without
# blocking every run forever. So: three verdicts, plus a stated-assumptions ledger
# that makes the certificate enumerate what it trusts instead of trusting silently.
#
# THE ADMISSION TEST, and it is deliberately hard to pass: is there any mechanism,
# however awkward, by which this gate could read the precondition back? If yes, it
# is a criterion this gate is missing — file it, do not launder it in here. Two
# candidates were rejected on exactly that ground while this was written: machine
# load (readable — /proc/loadavg, #81) and SMT state (readable — the siblings
# files, #82, which gate-p5.sh:343 already argues its own floor from). Both are
# checkable, therefore both are absent criteria rather than assumptions, and both
# are filed as such. A ledger that grows by absorbing checkable things is a way of
# writing "we did not get around to it" in a font that reads as rigour.
#
# The word FAIL/PASS/UNMEASURED must not appear in the label, for the same reason
# it must not in unmeasured(): the delegating gates count verdict lines by
# grepping anchored labels (gate-p4.sh:839, gate-p5.sh:987). The eight-space
# indent here is the `info` indent precisely so those anchors cannot match it.
ASSUMED=""
assumed() {
  printf '        assumed, unverifiable: %s\n' "$1"
  ASSUMED="${ASSUMED}${ASSUMED:+
}$1"
}

# assume_fleet HOSTS — the two assumptions every remote-measuring gate makes, in
# one place so six gates cannot word them six ways. Both survive the admission
# test above by being circular rather than merely inconvenient: every witness of a
# host's identity comes from the host itself.
assume_fleet() {
  [[ -n "${1:-}" ]] || return 0
  assumed "the configured host set is the fleet this gate is meant to measure: .keel-hosts (or \$KEEL_REMOTE_HOSTS) is the only statement of that intent, so there is no second source to check it against"
  assumed "each name reaches the machine it is meant to reach: every witness of a host's identity — cpuinfo, the CPU model that enters the record, the hostname itself — is reported BY the host under test, so a name pointed at the wrong machine reports consistently about the wrong machine"
}

# require_disk — the free-space precondition, read rather than assumed (#84).
#
# WHY A GATE OWNS THIS. The dev host's data volume filled during the 68fc493 batch
# and the gates printed `FAIL cross-compile of linux/amd64 bench binary` — a verdict
# about keel's build, for a build that was fine and a harness with nowhere to write.
# gate-p3 then died with no verdict line at all, and the delegating gates reported
# that death as the delegate's judgment. Free space is `df`, so this fails
# assumed()'s admission test above: readable and unread is a missing criterion.
#
# THE FLOOR IS MEASURED, and the dominant term is not the artifacts (measured on the
# dev host 2026-08-16, cold): a cold linux/amd64 build cache is 76 MiB for the first
# cross-compiled package and +1 MiB for the second, against 4.3–5.7 MiB per test
# binary — nine of them across the p5→p4→p3 chain, each gate holding its own
# `mktemp -d` — and 4.2–10.7 MiB per archived ssa.html, three of which gate-p2 keeps
# in build/ssa. Cold demand is therefore ~135 MiB, which is what makes the single
# reading known to be insufficient (137 MiB free, the run that failed) a fit rather
# than a coincidence. The floor is 512 MiB, 3.8x measured cold demand.
#
# What is NOT measured is a live peak: peak needs a run, and a run that ends with
# headroom may have passed near zero. So the reading is printed on every gate, green
# or not, which is how the fleet collects the distribution this floor was set without.
# There is deliberately no environment override — a knob that lowers a floor is a way
# to pass a gate by weakening it.
#
# ALWAYS RETURNS 0, for assert_governor's reason: an exit code is an implicit verdict
# channel, and under `set -euo pipefail` a loaded one (#80). Sets DISK_STATE to
# ok | low | unreadable.
DISK_FLOOR_MB=512
DISK_STATE=""
require_disk() {
  DISK_STATE=ok
  local t p label avail mnt seen=""
  t="$(mktemp -d 2>/dev/null || true)"
  # Both filesystems, because on darwin the gates' scratch is /var/folders/... —
  # neither /tmp nor the repo — and only one of the two is where build/ssa lands.
  # Deduplicated by mount point, so the usual case of one volume prints one line.
  for p in "${t:-}" "$PWD"; do
    [[ -n "$p" && -d "$p" ]] || continue
    [[ "$p" == "$PWD" ]] && label="repo, incl. build/" || label="scratch (mktemp -d)"
    # GUARDED, and the guard is the whole point (#76's idiom, remote_probe's header).
    # The first version read the two fields with `avail="$(df ... | awk ...)"`: under
    # `pipefail` an unreadable df makes that assignment fail, and under `set -e` the
    # gate then exits mid-preamble having printed no verdict line — which is #84's own
    # failure mode, reintroduced by #84's fix. Driving the unreadable branch on purpose
    # is what caught it; the ok and low branches passed either way.
    avail=""; mnt=""
    read -r avail mnt < <(df -Pk "$p" 2>/dev/null | awk 'NR==2 { print $(NF-2), $NF }') || true
    if [[ -z "$avail" || -z "$mnt" ]]; then
      DISK_STATE=unreadable
      unmeasured "free space behind $p ($label) is unreadable, so \"the harness can write\" stays an unchecked precondition: unreadable is not an exemption, and a volume that fills mid-run arrives as a FAIL about keel's build (#84)"
      continue
    fi
    case ",$seen," in *",$mnt,"*) continue ;; esac
    seen="$seen${seen:+,}$mnt"
    if [[ "$avail" -lt $((DISK_FLOOR_MB * 1024)) ]]; then
      DISK_STATE=low
      unmeasured "$((avail / 1024)) MiB free on $mnt ($label) is under this gate's $DISK_FLOOR_MB MiB floor: measured cold demand is ~135 MiB and the volume that filled had 137 MiB, so what this run would measure is its own scratch space (#84)"
    else
      info "disk headroom: $((avail / 1024)) MiB free on $mnt ($label), floor $DISK_FLOOR_MB MiB"
    fi
  done
  [[ -z "$t" ]] || rmdir "$t" 2>/dev/null || true
  return 0
}

# assumed_ledger — print the ledger, once, beside the verdict. Prints even when
# empty: "this gate declared no unverifiable assumption" is a statement a reader
# should be able to see the gate make, and its absence is indistinguishable from
# a gate that forgot to call this.
assumed_ledger() {
  echo
  echo "-- stated assumptions (trusted, not verified; not verdicts) --"
  if [[ -z "$ASSUMED" ]]; then
    printf '        none declared: every precondition this gate relies on has a read-back it performs above\n'
    return
  fi
  printf '        each line below is a precondition with no read-back mechanism, so it\n'
  printf '        carries no verdict. What the certificate asserts, it asserts ON these.\n'
  while IFS= read -r a; do
    [[ -n "$a" ]] || continue
    printf '          - %s\n' "$a"
  done <<<"$ASSUMED"
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

# assert_no_strays — worktree_strays as a gate criterion. All six gates opened
# with a byte-identical thirteen lines of this, each preceded by a seven-line
# restatement of the rationale above; the reasoning lives here, at the function
# it is about, and the gates now call it. No allowlist, no exemption (ruled
# 2026-08-14).
assert_no_strays() {
  local strays
  if strays="$(worktree_strays)"; then
    pass "no stray git worktrees (this repo is the only registered checkout)"
  else
    fail "a git worktree is registered besides this one, so either a measurement is in flight or its wreckage was left behind -- wait for it or kill it, then re-run"
    sed 's/^/        /' <<<"$strays"
  fi
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

# remote_build_test_or_fail PKG OUT LOG PASS_MSG FAIL_MSG — the guarded form of the call
# above, ten times over: build, render one verdict either way, and on failure indent the
# last 20 log lines. Both messages are parameters and neither is normalised — each site
# names its own package, and gate-p5's two say "of *the* linux/amd64 …" where p1-p4 omit
# the article. A de-duplication may not move a gate's output text (test_verdict's
# precedent), so only the shape is shared.
remote_build_test_or_fail() {
  if remote_build_test "$1" "$2" >"$3" 2>&1; then
    pass "$4"
  else
    fail "$5"
    sed 's/^/        /' "$3" | tail -20
  fi
}

# remote_probe HOST — print a one-line provenance record for HOST.
#
# The CPU model is not decoration: DESIGN.md §7 forbids reporting a
# measurement without the machine it came from, and a correctness result is
# just as host-specific as a benchmark (which microarchitecture confirmed the
# NaN operand order?).
#
# Cores, SMT width and sockets are here because `nproc` alone means two different
# things on the two arms of a cross-vendor comparison (#82): an Intel virtualized
# size gives 2 threads/core, AMD and Graviton give 1, so `GOMAXPROCS=8` is 8 cores
# on one arm and 4 on the other. Read from `thread_siblings_list` — one line per
# physical core, deduplicated — because that is the fact, where `nproc` is a
# consequence of it. `core_cpus_list` is the fallback (the newer name for the same
# file); SMT width is derived rather than read, and reports "?" unless it divides
# the CPU count exactly, since a non-integer ratio means the assumption behind the
# division is wrong on that host. Whether P5 then *requires* SMT off, or only that
# the state be recorded and unchanged between runs, is not settled here — #82 says
# it is a decision, and this is the reading half.
#
# NO SHELL GLOB CROSSES THE WIRE, and the reason is a defect this returned nothing
# for. sshd runs the *user's login shell*, not sh. Under zsh a pattern that matches
# nothing is an ERROR that aborts the whole remote command — not the sh behaviour of
# passing the pattern through for the next `[ -r ... ]` to reject — so on a host whose
# login shell is zsh this function printed nothing at all, and assert_governor read
# that empty output as `unreachable`: a host that answered perfectly, reported as one
# that never answered. Found because localhost became a legitimate far side for #62's
# exercise and localhost here is zsh. Every enumeration therefore goes through `find`
# with a QUOTED pattern, which no shell expands. The fleet's AMIs default to bash so
# this was not firing, which is the point — it would have fired on one contributor's
# host, once, as an unattributable UNMEASURED.
#
# `instance=` is host_admission's only input (#104): full size is not visible from inside
# the guest — a 4xlarge sees one socket and eight cores and looks entirely
# self-consistent — so the type is the fact and every field above is a consequence of it.
# Three outcomes kept apart because they are three different things (§5 rule 6): the
# type, `none` for a machine with no EC2 identity, `?` for no way to ask. `none` resolves
# to the restrictive class and `?` to unmeasured — fail-closed in both directions.
#
# `tmux=` is here because #62's supervisor can be absent, and an absent supervisor
# must not be a silent one. remote_exec degrades to an unsupervised run on a host
# without a usable tmux — the pre-#62 behaviour, still measured — so the fact that
# a measurement's lifetime was the ssh link's belongs in the archived record beside
# the governor, not in a variable nobody prints. Note it is read here with
# `command -v` under sshd's own PATH, which is not the login shell's: tmux is at
# /opt/homebrew/bin on the dev machine and invisible to `ssh host 'command -v tmux'`
# for exactly that reason. On Linux hosts it is /usr/bin/tmux and on the PATH.
#
# The private cache sizes are here for the same reason, added when #48's feed
# decomposition turned out to depend on one: several benchmarks hold a buffer they
# describe as L1-resident, whose size follows from the blocking parameters, and
# whether that description is true is a comparison against a number no host record
# carried. It is read from sysfs rather than assumed per microarchitecture — Zen 5
# has a 48 KB L1d where Zen 4 and Skylake-X have 32 KB, which is exactly the kind of
# difference a from-memory constant gets wrong. Missing files degrade to "?" rather
# than to a plausible default.
#
# CALLERS MUST GUARD THE SUBSTITUTION (#76). This returns ssh's exit status, which
# is 255 when the host does not answer, and every gate runs under `set -euo
# pipefail`. So `gov="$(remote_probe "$host" | sed -n '...')"` *terminates the
# gate* on an unreachable host — no verdict line, no verdict, exit 255, which
# DESIGN.md §5.6 forbids by name: a killed run is unmeasured, never an exit code.
# It also made the unreadable-value UNMEASURED branches that #73 wrote unreachable
# in precisely the case they exist for, and the delegating gates reported the death
# as `gate-pN is RED (exit 255)` — a red attributed to keel for a host that hung up.
# The idiom is `"$(remote_probe "$host" | sed -n '...' || true)"`: the value comes
# back empty, and empty is a reading nobody got, which the caller already knows how
# to print. Same for every other ssh-backed reader here and in the gates
# (gate-p3's ob_preflight and ob_coretype_sweep).
# KEEL_GOV_PROBE_SH — the far-side governor reading, sh, sets `gov`. Shared by every
# prober in the tree by concatenation into their ssh argument, because two independent
# readings of one knob is not redundancy, it is a pair that can disagree — and it did:
# provision-openblas.sh had its own one-liner and called an AWS guest `governor=unknown`
# on the same host where this cascade says `absent`, so the same machine was a defect to
# one script and a guest to the other.
#
# THREE TOKENS, NOT TWO, because DESIGN section 5 rule 5 (amended 2026-08-16) rules that
# "no cpufreq interface at all" and "the file is present and unreadable" are different
# causes and may not share one verdict: the first is a virtualized guest, which does not
# own the knob, and the second is a defect on a host that has it. Neither is read from
# "cpu MHz", which is present, plausible and a fixed nominal constant, so a harness
# sampling it would certify stability forever, where an absent directory fails closed.
#
# NO APOSTROPHES ANYWHERE IN HERE. It is one single-quoted string that becomes part of a
# single-quoted ssh argument, so a possessive in a COMMENT terminates the string and the
# far side receives a fragment. shellcheck catches it (SC1011), which is the only reason
# this is a comment and not an outage.
KEEL_GOV_PROBE_SH='
  if [ -r /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor ]; then
    gov=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo unknown)
    [ -n "$gov" ] || gov=unknown
  elif [ -d /sys/devices/system/cpu/cpu0/cpufreq ]; then
    gov=unreadable
  else
    gov=absent
  fi
'

remote_probe() {
  local host="$1"
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "$KEEL_GOV_PROBE_SH"'
    cpu=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2- | sed "s/^ *//")
    ncpu=$(nproc)
    T=/sys/devices/system/cpu
    uniq_lines() { find "$T" -maxdepth 3 -name "$1" -exec cat {} + 2>/dev/null | sort -u | wc -l | tr -d " "; }
    cores=$(uniq_lines thread_siblings_list)
    [ "${cores:-0}" -gt 0 ] || cores=$(uniq_lines core_cpus_list)
    if [ "${cores:-0}" -gt 0 ] && [ $((ncpu % cores)) -eq 0 ]; then smt=$((ncpu / cores)); else smt="?"; fi
    [ "${cores:-0}" -gt 0 ] || cores="?"
    sockets=$(uniq_lines physical_package_id)
    [ "${sockets:-0}" -gt 0 ] || sockets="?"
    cache=""
    for d in $(find "$T/cpu0/cache" -maxdepth 1 -name "index*" 2>/dev/null); do
      [ -r "$d/level" ] || continue
      lvl=$(cat "$d/level"); typ=$(cat "$d/type"); sz=$(cat "$d/size")
      case "$typ" in Instruction) continue ;; Data) tag="L${lvl}d" ;; *) tag="L${lvl}" ;; esac
      cache="$cache${cache:+ }$tag=$sz"
    done
    tmux=no
    command -v tmux >/dev/null 2>&1 && tmux=yes
    inst="?"
    if command -v curl >/dev/null 2>&1; then
      inst=none
      IM=http://169.254.169.254
      tok=$(curl -m 2 -sf -X PUT "$IM/latest/api/token" \
        -H "X-aws-ec2-metadata-token-ttl-seconds: 60" 2>/dev/null) || tok=""
      if [ -n "$tok" ]; then
        inst=$(curl -m 2 -sf -H "X-aws-ec2-metadata-token: $tok" \
          "$IM/latest/meta-data/instance-type" 2>/dev/null) || inst="?"
        [ -n "$inst" ] || inst="?"
      fi
    fi
    printf "%s | instance=%s | %s cpus | %s cores | smt=%s | %s sockets | governor=%s | tmux=%s | %s | %s\n" \
      "$cpu" "$inst" "$ncpu" "$cores" "$smt" "$sockets" "$gov" "$tmux" "$(uname -sr)" "${cache:-caches=?}"
  ' 2>/dev/null
}

# assert_governor HOST PHASE [PROV] — the §5 rule 5 performance-governor
# precondition, asserted per host, defined once for all five measuring gates.
#
# PHASE  preamble | measured. Selects wording, and whether a provenance line is
#        printed. The preamble prints a verdict on every outcome including PASS, so
#        a `governor=` info line there would only restate it; at measurement time
#        the success path is deliberately silent (the preamble already passed that
#        host), so the info line is the only surviving record of what was read.
# PROV   pre-captured remote_probe output. Omit it and this probes the host itself;
#        pass it and no second ssh happens. Passing an *empty string* is meaningful and is not the
#        same as omitting the argument: it says "already probed, and the host
#        produced nothing", which is the distinction this whole function is about.
#
# Sets, for the caller's control flow and its provenance text:
#   GOV_STATE   performance | wrong | nocpufreq | unreadable | unreachable
#               nocpufreq and unreadable are separate by ruling, not by taste: §5
#               rule 5 (amended 2026-08-16) forbids them sharing a verdict, because a
#               guest with no cpufreq directory and a host whose knob is present and
#               unreadable are a platform fact and a defect respectively. Every gate
#               branches on `!= performance`, so a new state is refused by default and
#               nothing here can open a path by being added.
#   GOV_VALUE   the parsed value; empty when there is no reading at all
#   GOV_SHOWN   GOV_VALUE rendered for a log line, never manufacturing a reading
#
# IT ALWAYS RETURNS 0, and that is constitutional rather than stylistic. An exit
# code is an implicit verdict channel, and under `set -euo pipefail` it is a loaded
# one: a helper returning non-zero to mean "criterion not met" would, called bare in
# tail position, become the gate's own status and kill the run — and the failure
# would not be a wrong verdict but an *absent* one, a gate that dies having printed
# nothing, indistinguishable from a kill (#76, #80). "Always return 0, and state the
# outcome in a global if a caller needs it" is the correct form. This paragraph is
# here so that a later cleanup does not simplify it back into an exit code.
#
# WHY THIS IS ONE FUNCTION AND NOT FIVE COPIES (ruled 2026-08-16, closing #83).
# Four gates reported an *unreachable* host as "scaling_governor is unreadable" — a
# correct verdict, both block, with a false cause — while gate-p5's divergent copy
# had the guard. "Adopt gate-p5's version in the other four" was ruled insufficient:
# four better copies are the same failure mode with a better master, and the next
# labelling defect propagates just as cleanly. The drift inventory taken at the time
# found five independent divergences across the ten copies (two sites x five gates):
# a `§5.4` citation (citation-lint:quote) — inventoried then as naming "a section
# DESIGN.md does not have", which #85's audit corrected to a mis-minted item number:
# read as the shorthand "§5, item 4" the form resolves, to the benchmarks-are-tests
# rule rather than the methodology rule it meant — an
# `info "governor=${gov:-unknown}"` that printed a reading for a host which had
# answered nothing, "preamble checked it" against "preamble read it", a remediation
# hint present in one gate and absent in four, and p5's better parse. Agreement
# among four copies was never evidence about any of them.
#
# Calling the *gate's* pass/fail/info from here follows unmeasured()'s and
# assume_fleet()'s precedent rather than inventing an idiom: shell resolves them at
# call time, so each gate keeps its own primitives and this file keeps the wording.
GOV_STATE=""
GOV_VALUE=""
GOV_SHOWN=""
# The probe line this call read, so host_admission does not pay a second ssh round trip
# for a fact already on the wire. Reset with the rest: a stale provenance line is exactly
# what would let one host's instance type classify the next one.
GOV_PROV=""
assert_governor() {
  local host="$1" phase="${2:-preamble}" prov
  if [[ $# -ge 3 ]]; then prov="$3"; else prov="$(remote_probe "$host" || true)"; fi

  GOV_STATE=""; GOV_VALUE=""; GOV_SHOWN=""; GOV_PROV="$prov"
  if [[ -z "$prov" ]]; then
    # No reading exists. remote_probe's own `|| echo unknown` means a host that
    # answers always yields a governor= field, so empty output is the host not
    # answering — the one discriminator the four copies did not have.
    GOV_STATE=unreachable
    GOV_SHOWN="none (host produced no reading)"
  else
    GOV_VALUE="$(sed -n 's/.*governor=\([^ |]*\).*/\1/p' <<<"$prov")"
    if [[ "$GOV_VALUE" == performance ]]; then
      GOV_STATE=performance
      GOV_SHOWN="$GOV_VALUE"
    elif [[ "$GOV_VALUE" == absent ]]; then
      # A guest, not a defect. GOV_VALUE is cleared because `absent` is the probe's
      # word for "there was nothing to read", and leaving it in GOV_VALUE would put
      # the string `absent` into log lines that read as governor *values* — which is
      # how "governor is 'absent', not performance" would have been printed, a
      # sentence asserting a reading nobody took.
      GOV_STATE=nocpufreq
      GOV_VALUE=""
      GOV_SHOWN="none (no cpufreq interface — a virtualized guest does not own the knob)"
    elif [[ -z "$GOV_VALUE" || "$GOV_VALUE" == unknown || "$GOV_VALUE" == unreadable ]]; then
      GOV_STATE=unreadable
      GOV_SHOWN="unknown (host answered; scaling_governor unreadable)"
    else
      GOV_STATE=wrong
      GOV_SHOWN="$GOV_VALUE"
    fi
  fi

  if [[ "$phase" == measured ]]; then
    info "[$host] governor=$GOV_SHOWN"
    case "$GOV_STATE" in
      performance) : ;;  # silent on success: the preamble printed the PASS
      unreachable)
        unmeasured "[$host] unreachable at measurement time, so this host produced no governor reading and nothing measured here can be asserted to be covered by §5 rule 5 — unmeasured for want of an answer, and not a governor that changed" ;;
      nocpufreq)
        # No verdict here, deliberately, and this is the one branch in this case that
        # prints nothing at all. §5 rule 5 gives a host with no governor a different
        # instrument, not a worse one, and clock_post is what reads it — three peak
        # windows either side of the sweep, which cannot be judged until the sweep has
        # run. A verdict at this point would be a second one about the same host, and
        # the first to print is the one that would be believed.
        : ;;
      unreadable)
        unmeasured "[$host] the host answered but the governor is unreadable at measurement time, so nothing measured here can be asserted to be covered by §5 rule 5 — unmeasured, not a governor that changed" ;;
      wrong)
        fail "[$host] governor is '$GOV_VALUE' at measurement time, not performance: it changed after this gate's preamble checked it, so nothing measured here is covered by §5 rule 5" ;;
    esac
  else
    case "$GOV_STATE" in
      performance)
        pass "[$host] cpufreq governor is performance (§5 rule 5)" ;;
      unreachable)
        unmeasured "[$host] unreachable, so this target produced no governor reading at all: §5 rule 5 is unverified here for want of an answer, not for want of a readable file" ;;
      nocpufreq)
        # NO LONGER BLOCKS, and what changed is that the substitute exists: clock_post
        # in scripts/bench.sh reads BenchmarkPeak at the head, middle and tail of the
        # sweep and refuses a declining or unbounded series. §5 rule 5 licensed a
        # different instrument for a guest, never an exemption, and until the instrument
        # was written the honest state was `unmeasured` — opening the path first would
        # have been weakening a gate to pass it.
        #
        # Says what was observed, not what the host is. "No cpufreq directory" is the
        # reading; "a virtualized guest" is the usual explanation for it and is not
        # the only one — this fires on macOS too — so the inference is labelled as an
        # inference. A verdict line that names a cause it did not establish is the
        # same defect as the four gates that called an unreachable host unreadable.
        info "[$host] no cpufreq interface at all, so §5 rule 5's governor instrument does not exist on this host rather than having failed on it — the usual reason is a virtualized guest, which does not own the knob. Stability is established here by its ruled substitute instead: BenchmarkPeak at the head, middle and tail of the sweep, judged after the sweep runs"
        info "  [$host] separate from 'present and unreadable' by ruling: that is a defect on a host that has the knob, this is a host that has none" ;;
      unreadable)
        unmeasured "[$host] the host answered but scaling_governor is unreadable, so §5 rule 5 cannot be verified: an unchecked precondition is not a met one, and this blocks the gate exactly as a wrong governor does" ;;
      wrong)
        fail "[$host] cpufreq governor is '$GOV_VALUE', not performance (§5 rule 5): a ramping core produces cold readings that enter the record as measurements"
        info "  [$host] sudo cpupower frequency-set -g performance"
        info "  [$host] or: echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor" ;;
    esac
  fi
  return 0
}

# The types admitted to the evidentiary class: the largest non-metal size in each
# approved family, which is a whole host and therefore a whole socket too — a
# conservative subset of what docs/hosts.md admits, not a restatement of it. Metal is
# absent because metal is retired, not because it was overlooked.
#
# An allowlist is safe here in the way a duplicated threshold is not, and that is why
# this is one: the default is the RESTRICTIVE class, so a stale list can only withhold a
# judgement, never grant one.
KEEL_EVIDENTIARY_SIZES="c7i.48xlarge c8i.96xlarge c7a.48xlarge c8a.48xlarge"

# host_admission PROV — set ADM_CLASS and ADM_INSTANCE from a provenance line
# (docs/hosts.md, ruled 2026-08-17 on #104). ADM_CLASS is one of:
#
#   evidentiary  perf may be judged: a floor is a claim about silicon this host owns
#   correctness  perf is reported, never judged, however high or low it reads
#   unknown      the class could not be read, which is `unmeasured` and not a class
#
# The third state is the one that matters: without it, the mechanism that excuses a
# partial-size reading from the floor is also a mechanism for laundering a red past it.
ADM_CLASS=""
ADM_INSTANCE=""
host_admission() {
  ADM_INSTANCE="$(sed -n 's/.*instance=\([^ |]*\).*/\1/p' <<<"${1-}")"
  case "$ADM_INSTANCE" in
    "" | "?") ADM_CLASS=unknown ;;
    *) case " $KEEL_EVIDENTIARY_SIZES " in
         *" $ADM_INSTANCE "*) ADM_CLASS=evidentiary ;;
         *) ADM_CLASS=correctness ;;
       esac ;;
  esac
}

# admission_readback HOST PROV — the preamble line stating HOST's class, before any
# number from it is read. Stated for every CONFIGURED host, not only for the ones that
# survive to a judged criterion: the first version of this read the class at the floor
# check alone, and the aggregate's "1 of 3 not admitted" then described a fleet in which
# all three were not admitted — the verdict right and its sentence false, which is #37's
# label defect and #90's arriving from a third direction.
admission_readback() {
  host_admission "$2"
  info "[$1] admission class: $ADM_CLASS (instance=${ADM_INSTANCE:-unread})"
}

# adm_judgeable HOST PROV READING — may a perf verdict be formed from HOST's numbers?
# Returns 0 for the evidentiary class. Otherwise it emits the not-judged verdict itself
# and returns 1, so a caller gates on it and tallies:
#
#   adm_judgeable "$host" "$GOV_PROV" "$reading" || { NOTADM=$((NOTADM+1)); continue; }
#
# READING is the number that would have been judged, and it is printed either way: a
# class that withholds a judgement must not also withhold the measurement. Call it AFTER
# the reading is rendered and BEFORE the verdict — and before any "indeterminate" branch,
# because the two are not interchangeable. Indeterminate says "re-measure, uncapped";
# not-admitted says "no number from this host is judgeable", which no re-run fixes.
#
# #104, ruled 2026-08-17: a floor is a claim about silicon, and a partial-size guest
# shares its socket with tenants the run cannot see, so its reading is reported and never
# judged however high or low it reads. `c7i.4xlarge` read 34.2% of peak and a flat floor
# turned that into a P2 STOP on a host never admitted to the class the floor governs.
adm_judgeable() {
  host_admission "$2"
  case "$ADM_CLASS" in
    evidentiary) return 0 ;;
    correctness)
      info "[$1] correctness-class ($ADM_INSTANCE is not a full-size instance of an approved family): $3 — reported, not judged (docs/hosts.md)" ;;
    *)
      unmeasured "[$1] the admission class is unreadable (instance=${ADM_INSTANCE:-absent from the provenance line}), so this criterion is unmeasured here rather than cleared or missed: an unread identity must not be the mechanism that excuses a reading from a floor" ;;
  esac
  return 1
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
#
# THE MEASUREMENT NO LONGER LIVES INSIDE THE SSH CONNECTION (#62). It runs in a
# detached remote tmux session, so the ssh carries control messages and the
# program's lifetime is not the link's. The trigger for building this was the
# fleet leaving the LAN: on a billed cloud instance a transient drop is a real
# event class, and it used to SIGHUP a benchmark mid-flight and cost the run.
#
# THE COMMAND CROSSES AS DATA, NOT AS SHELL WORDS, and that is the whole defence
# against the hazard #62 named. Wrapping the existing command string in a tmux
# argument would give the far side a SECOND expansion on top of printf %q, and
# the failure mode of getting that wrong is not a loud error — it is a benchmark
# that measured something adjacent to what was asked for. So the command is
# written to a runner script LOCALLY, shipped by the scp that already ships the
# binary, and tmux is handed one quoted path. The bytes the far side executes are
# the bytes the old single-ssh form would have handed to its shell.
#
# THREE OUTCOMES, AND ONLY ONE OF THEM IS AN EXIT CODE. The runner writes the
# program's status to a status file after it exits — detach.sh's design, one hop
# out — so "finished badly" and "never finished" stay distinguishable:
#
#   REMOTE_STATE=ok        a status file exists; the return value is the
#                          program's own exit code, exactly as before.
#   REMOTE_STATE=vanished  the session is gone and no status was written. The
#                          return value is $REMOTE_EXIT_VANISHED, which is NOT a
#                          program exit code and must not be read as one — see
#                          remote_vanished below, and DESIGN.md §5.6.
#   REMOTE_SUPERVISED=no   there was no usable tmux, so the run was unsupervised
#                          (the pre-#62 behaviour). Not a failure and not an
#                          exemption: still measured, still reported, and the
#                          fact is in every gate's provenance line as `tmux=`
#                          because a supervisor that is silently absent is worse
#                          than one that is absent loudly.
#
# THE SUPERVISOR IS SILENT ON THE WIRE, deliberately and load-bearingly. This
# function's stdout IS the program's output: every caller redirects `2>&1` into a
# log that benchstat and anchored marker greps then parse, so one chatty line
# from the supervisor would land in the parsed data. Hence state travels in
# variables and never in output, and the log comes back through its own `cat`
# whose stdout is passed straight through — not through `$(...)`, which strips
# trailing newlines and would edit every log by a byte.
#
# The runner is shipped rather than fed on stdin because $KEEL_SSH_OPTS carries
# `-n`, which redirects stdin from /dev/null: a heredoc-fed remote shell reads EOF,
# executes nothing, and exits 0 — a command that silently did not run, reported as
# success. Observed on all three hosts of the desktop fleet; only reading the effect
# back caught it.
REMOTE_EXIT_VANISHED=125
REMOTE_STATE=""
REMOTE_SUPERVISED=""
REMOTE_WAIT_RETRIES="${REMOTE_WAIT_RETRIES:-12}"
REMOTE_EXEC_SEQ=0

# remote_vanished — true when the last remote_exec's measurement did not finish.
# The clause every caller needs BEFORE it interprets a nonzero status, because
# "the binary exited nonzero" and "nobody saw the binary exit" are different
# facts and only the first is a claim about keel.
remote_vanished() { [[ "$REMOTE_STATE" == vanished ]]; }

remote_exec() {
  local host="$1" bin="$2"; shift 2
  local base; base="$(basename "$bin")"
  local args="" a
  for a in "$@"; do args+=" $(printf '%q' "$a")"; done

  REMOTE_STATE=""
  REMOTE_SUPERVISED=""
  REMOTE_EXEC_SEQ=$((REMOTE_EXEC_SEQ + 1))
  local id="keel-$$-$REMOTE_EXEC_SEQ"
  local runner="$KEEL_REMOTE_DIR/$id.sh" log="$KEEL_REMOTE_DIR/$id.log" st="$KEEL_REMOTE_DIR/$id.status"

  local tmpd; tmpd="$(mktemp -d)"
  # `\$?` is escaped so the status is read on the far side, at the far side's
  # moment; expanded here it would freeze this shell's last exit code into the
  # script and report it as the measurement's.
  cat > "$tmpd/$id.sh" <<EOF
#!/bin/sh
# Generated by remote_exec (keel #62). Not edited by hand and not reused: one
# runner per measurement, named for the tmux session that supervises it.
cd '$KEEL_REMOTE_DIR' || exit 127
{ env ${KEEL_REMOTE_ENV:-} ./'$base'$args; printf '%s\n' "\$?" > '$st'; } > '$log' 2>&1
EOF

  ssh "${KEEL_SSH_OPTS[@]}" "$host" "mkdir -p '$KEEL_REMOTE_DIR'" >/dev/null
  scp -q "${KEEL_SCP_OPTS[@]}" "$bin" "$tmpd/$id.sh" "$host:$KEEL_REMOTE_DIR/"
  rm -rf "$tmpd"

  # One launch, which also decides supervision on the far side — a separate
  # `command -v tmux` probe would be a third round trip to learn what the launch
  # already knows. tmux present but unable to start a server (no writable $HOME,
  # no /tmp) falls through to the unsupervised arm rather than losing the
  # measurement: that arm is exactly the behaviour every gate had before #62.
  local supervision
  supervision="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" "
    if command -v tmux >/dev/null 2>&1 && tmux new-session -d -s '$id' \"sh '$runner'\" 2>/dev/null; then
      echo supervised
    else
      echo unsupervised
      sh '$runner'
    fi" 2>/dev/null || true)"
  case "$supervision" in
    supervised*) REMOTE_SUPERVISED=yes ;;
    *)           REMOTE_SUPERVISED=no ;;
  esac

  # Wait on the far side, in one connection rather than a poll from here: a
  # 25-minute benchmark polled from the driver is 300 ssh handshakes, each of
  # which is a chance to fail at a moment when nothing was wrong. Retried
  # because a dropped link during the wait is now survivable — the session is
  # still running, and `has-session` makes reconnecting idempotent.
  local tok="" tries=0
  if [[ "$REMOTE_SUPERVISED" == yes ]]; then
    while :; do
      tok="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" "
        while tmux has-session -t '$id' 2>/dev/null; do sleep 3; done
        if [ -f '$st' ]; then cat '$st'; else echo vanished; fi" 2>/dev/null || true)"
      [[ -z "$tok" ]] || break
      tries=$((tries + 1))
      [[ "$tries" -lt "$REMOTE_WAIT_RETRIES" ]] || break
      sleep 5
    done
  else
    tok="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" \
      "if [ -f '$st' ]; then cat '$st'; else echo vanished; fi" 2>/dev/null || true)"
  fi

  # The log, byte-exact, on the caller's stdout. Emitted even when the run
  # vanished: a truncated log is evidence about where it stopped, and the
  # caller has REMOTE_STATE to keep it from being read as a complete one.
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "cat '$log' 2>/dev/null" || true
  ssh "${KEEL_SSH_OPTS[@]}" "$host" "rm -f '$runner' '$log' '$st'" >/dev/null 2>&1 || true

  case "$tok" in
    *[!0-9]* | "") REMOTE_STATE=vanished; return "$REMOTE_EXIT_VANISHED" ;;
    *)             REMOTE_STATE=ok; return "$tok" ;;
  esac
}

