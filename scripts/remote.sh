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
# ALL FIVE VERDICT HELPERS LIVE HERE, AND NO GATE THAT SOURCES THIS FILE DEFINES
# ITS OWN (2026-08-16; scoped 2026-08-16 after the sentence was checked; a fifth
# added 2026-08-21).
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

# RUN_STAMP — the per-process half of an output path's key. #78 rev-stamped the delegated
# gate logs so a run at one rev could no longer destroy a run at another, and that left
# TWO RUNS AT ONE REV overwriting each other exactly as before: the same defect, in the
# same paths, past its own fix. Which is the ordinary case here, not a corner — DESIGN.md
# §4 allows one immediate re-run of a failing throughput sentinel with *both* outputs
# archived, and an instrument exercise runs one gate three times over at one rev. Not
# exported, so a delegated gate stamps its own log with its own process's stamp; seeded
# from the environment so a driver that wants one stamp across a chain can set it. One
# definition, because bench_csv's identical stamp was the second copy and this is where a
# third would have started to drift.
RUN_STAMP="${RUN_STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"

VERDICT_STAMP="${VERDICT_STAMP:-}"
pass()       { printf '  \033[32mPASS\033[0m  %s%s\n'        "$VERDICT_STAMP" "$1"; }
fail()       { printf '  \033[31mFAIL\033[0m  %s%s\n'        "$VERDICT_STAMP" "$1"; FAIL=1; }
unmeasured() { printf '  \033[33mUNMEASURED\033[0m  %s%s\n'  "$VERDICT_STAMP" "$1"; FAIL=1; }
info()       { printf '        %s%s\n'                       "$VERDICT_STAMP" "$1"; }
# baseline() — a criterion declining to judge a host its reference artifact predates,
# and recording the reference this run would propose instead (#6, ruled 2026-08-21).
# It is the one verdict helper that does NOT raise FAIL, and that is the whole of its
# content: green-compatible, because convicting a host for its absence from a run it
# was not in measures its admission date. It is not an escape hatch — a host reaches
# it at most once, and the run that renders it creates the archived log that makes the
# next absence an unmet obligation (FAIL) instead of newness. A colour of its own
# because folding it into info() would leave the one green-compatible non-pass
# invisible to a reader scanning the verdict column.
baseline()   { printf '  \033[36mBASELINE\033[0m  %s%s\n'    "$VERDICT_STAMP" "$1"; }

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
# of exactly #78's shape (a stray log picked up by something hungry for a reference, whose
# reader greps the last line), reachable from the environment. The stamp IS the synthetic-run
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

# instrument_exercise REASON — arm synthetic mode: stamp every verdict line, print the
# banner, and thereby make gate_verdict above withhold. Returns 1 on an empty REASON so
# a caller can branch on it without re-reading the environment, and does nothing else:
# it is read nowhere near a comparison, a threshold, a tally or a host list, so no
# rendering below can be forged by setting it.
#
# LIFTED FROM gate-p2.sh (2026-08-21), which had the only copy, the moment gate-p5
# needed the same three lines. A second copy of a forgery guard is how the first one
# comes to be the only one that works, and that is not a hypothetical here: it is
# VERDICT_STAMP's own history twenty lines up, where five copies of pass/fail existed
# and exactly one stamped. gate-p2's rationale for what its exercise DRIVES stays in
# gate-p2 — that is about its branch, not about this mechanism.
#
# THE EXPORT IS THE PART THAT WAS MISSING, and it is a defect of the copy rather than of
# the design, which is why lifting had to fix it rather than preserve it. gate-p2
# delegates to nothing, so a stamp that stopped at its own process cost it nothing.
# gate-p5 runs `bash scripts/gate-p4.sh`, which runs gate-p3: unexported, the child
# seeds VERDICT_STAMP empty, stamps not one of its ~50 verdict lines, and signs
# `gate-p4: GREEN` into a real gate-pN log path — over whatever host list the parent's
# exercise imposed, which for an exercise is usually one host presented as a whole
# fleet. A forgeable certificate produced by the machinery built to prevent one (#78).
# So the stamp crosses the process boundary with the run.
instrument_exercise() {
  local reason="${1:-}"
  [[ -n "$reason" ]] || return 1
  VERDICT_STAMP="[synthetic] "
  export VERDICT_STAMP
  echo "  ############################################################"
  echo "  ##  SYNTHETIC RUN -- NOT A GATE RESULT                    ##"
  echo "  ##  reason: $reason"
  echo "  ##  Every verdict line below carries a [synthetic] stamp. ##"
  echo "  ##  No verdict is signed; the exit code is 2.             ##"
  echo "  ##  This run judges the instrument, never the phase.      ##"
  echo "  ##  Delegated gates inherit the stamp and withhold too.   ##"
  echo "  ############################################################"
  echo
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
# `instance=` is one of host_admission's two inputs (#104): full size is not visible from
# inside the guest — a 4xlarge sees one socket and eight cores and looks entirely
# self-consistent — so the type is the fact and every field above is a consequence of it.
# Three outcomes kept apart because they are three different things (§5 rule 6): the
# type, `none` for a machine with no EC2 identity, `?` for no way to ask. `?` resolves to
# unmeasured, and `none` no longer resolves through the default arm — see host_admission.
#
# `virt=` is the second input, and it is here because `instance=none` was answering two
# questions with one word (#106). A machine with no EC2 identity is either bare metal —
# the *limiting* case of full size, owning its socket more completely than any instance
# type can demonstrate — or a non-EC2 guest of genuinely unknown size, and those are
# opposite classifications. The discriminator is the `hypervisor` CPU flag, read directly
# rather than inferred from an unrelated capability. Three tokens, and the third is the
# load-bearing one: `metal` (a flags line with no hypervisor flag), `guest` (the flag is
# there), `?` (there is no flags line to read — no /proc/cpuinfo, or an architecture that
# does not print one, which is this repo's own macOS dev host and any arm64 target).
# `?` must not read as `metal`, or an absent instrument would grant what it cannot see.
# WHAT THIS CANNOT SEE (§5 rule 12): the flag is CPUID.1:ECX.31, which a hypervisor sets
# by convention and may clear, so a guest configured to hide itself is classified `metal`.
# That is strictly stronger evidence than the governor-only route prototyped on #104 (a
# readable governor proves cpufreq exposure, not single tenancy) and strictly weaker than
# an attestation, which no pure-Go, no-credentials probe can obtain.
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

# KEEL_PIN_WIDTH / KEEL_PIN_SH — the CPU affinity mask DESIGN section 5 rule 5 adopted
# fleet-wide on 2026-08-21, and the code that was missing when it did.
#
# THE SURPRISE THIS FIXES, recorded because it is the more useful half of the change.
# The pinning was ruled, written into DESIGN section 5 rule 5 with its falsification
# condition, described in docs/hosts.md in the PRESENT TENSE ("Every judged benchmark
# invocation on every host runs under a CPU affinity mask of eight distinct physical
# cores"), and given a measurement era of its own in scripts/measurement-eras.tsv --
# and `git grep taskset` over the whole tree answered with four CHANGELOG lines, three
# doc paragraphs and two comments saying P2 needs none. Not one line of scripts/ set a
# mask. Three artifacts asserting a mechanism, zero implementing it: the law, the doc
# and the era ledger agreed with each other, which is one witness restated three times
# and not three witnesses (section 5 rule 10). Grep for a mechanism before publishing
# the number that depends on it.
#
# WHERE IT GOES, and why here rather than in each gate. remote_exec is the single place
# every gate, every sweep and every ensemble launches a binary on a host -- 20 call
# sites, all of them through this one function -- so a mask applied here cannot be
# deviated from by one gate, which is the same argument section 4 makes for keeping the
# benchmark mechanics in scripts/bench.sh.
#
# WHICH INVOCATIONS, exactly: the ones carrying -test.bench, because the rule binds
# "every judged benchmark invocation" and the ceiling arm is the same invocation as the
# rows it is the denominator of (one -test.bench=Scale|Peak|Ceiling), so numerator and
# denominator get the identical mask by construction rather than by two sites agreeing.
# The correctness runs (-test.v) are left free: a mask cannot make a wrong answer right,
# and pinning them would refuse to run the TEST SUITE on any host with fewer than eight
# physical cores in a node -- a coverage loss in exchange for nothing measured.
#
# IT REFUSES RATHER THAN FALLING BACK. No taskset, no NUMA topology in sysfs, or fewer
# than eight distinct physical cores inside a single node, and the measurement is not
# taken: status 121, a log saying so, and the caller reports it unmeasured. Free
# placement as a silent fallback is the one behaviour that must not exist, because it
# produces exactly the artifact the era ledger was built to make impossible -- a
# free-placement reading wearing a pinned8 label. "Fleet-wide and never selectively" is
# a constraint on the harness before it is one on the operator.
#
# NO APOSTROPHES ANYWHERE IN HERE, for the reason the gov probe above states.
KEEL_PIN_WIDTH=8
KEEL_PIN_SH='
keel_pin_mask() {
  want=$1
  # The sysfs root, overridable for the same reason KEEL_REMOTE_DIR is: this function
  # reads a topology, and a topology is the one input a caller may legitimately want to
  # supply from somewhere else. scripts/remote-exec-test.sh builds fake ones -- an
  # 18-core-per-node SMT host, a 4-core node, a node directory that does not exist -- and
  # drives all of them on the dev machine, which has no sysfs at all. Setting it in a real
  # run cannot forge a measurement: it selects a mask, and if that mask is not the one the
  # Go runtime ends up under, gate-p5 fails the placement criterion on the mismatch.
  S=${KEEL_SYSFS:-/sys/devices/system}
  T=$S/cpu
  # One node at a time, and the count restarts at each: eight cores spread over two
  # nodes is the cross-socket migration this mask exists to stop, so a partial run of
  # cores in node 0 may not be completed out of node 1.
  for nd in "$S"/node/node[0-9]*; do
    [ -r "$nd/cpulist" ] || continue
    out=; n=0
    for spec in $(tr "," " " < "$nd/cpulist"); do
      case $spec in
        *-*) a=${spec%%-*}; b=${spec##*-} ;;
        *)   a=$spec; b=$spec ;;
      esac
      c=$a
      while [ "$c" -le "$b" ]; do
        sib=
        [ -r "$T/cpu$c/topology/thread_siblings_list" ] &&
          sib=$(cat "$T/cpu$c/topology/thread_siblings_list")
        # No sibling list, no claim of distinctness: abandon this node and try the next
        # rather than assume the cpus are cores. Taking eight anyway would be four
        # hyperthreaded cores under a mask of width eight -- and the readback in gate-p5
        # cannot catch it, because GOMAXPROCS is 8 either way. The fixture that drives
        # this branch is the reason it exists: the first version fell back to sib=$c.
        if [ -z "$sib" ]; then break 2; fi
        first=${sib%%,*}; first=${first%%-*}
        # Only a threads first sibling, so the mask is eight distinct CORES and not four
        # cores twice over. Go reports the mask width as GOMAXPROCS, so this is also what
        # makes every row name carry -8: the width is readable off the measurement.
        if [ "$first" = "$c" ]; then
          out="${out}${out:+,}$c"
          n=$((n + 1))
          if [ "$n" -ge "$want" ]; then printf "%s\n" "$out"; return 0; fi
        fi
        c=$((c + 1))
      done
    done
  done
  return 1
}
'

# spawn_probe HOST — the LAUNCHER's record of HOST, as the provenance line's `spawn=` field
# (Scott's directive, 2026-08-19). Four tokens, apart for the reason `virt=` has three:
#
#   id:type:market  a single running launcher record, and what it says
#   none            the launcher answered and has no running record under this name
#   ambiguous       more than one running record answers to this name, so no join is safe
#   ?               there was no way to ask (no spawn, no jq, no credentials, or it failed)
#
# WHY a launcher-side field is a witness in kind and not one more probe, why `?` must never
# read as `none`, and what each token buys: docs/hosts.md, the 2026-08-19 amendment. Not
# repeated here — the argument was in both places for one commit, which is two copies of a
# claim that can drift apart, and the doc is where a reader looking for the class table
# already is.
#
# THE JOIN KEY IS THE SSH ALIAS, matched against spawn's `name` EXACTLY, which is a
# constraint on how the fleet is launched and not a heuristic: aws-fleet.sh launches each
# host as `spawn launch <alias>`, the same string it writes into ~/.ssh/config. (It said
# `--name` until 2026-08-19; that flag exists but spawn wants the name positionally, and a
# comment naming a flag the code does not pass is a mechanism claim a reader would grep
# for.) A missed join reads `none` and withholds
# admission; a join on something rename-proof like a public address would, on a reused
# address, grant admission to a reading from another machine.
#
# The table is fetched once and every host extracted from it BY NAME. That is not the hazard
# GOV_PROV names — a scalar global carrying the previous host's reading — because nothing
# here is keyed by position; it would become that hazard the moment an extraction fell back
# to "the first record", so it does not. The cache saves nothing in practice: every caller
# invokes remote_probe inside `$(...)`, which forks, so each host pays one all-region sweep
# (~9s measured) and gate-p5 pays two. Stated rather than fixed — a file-backed cache buys
# back under a minute of a twenty-five-minute run by holding a launcher reading across a
# boundary the shell currently guarantees it cannot cross.
KEEL_SPAWN="${KEEL_SPAWN:-spawn}"
KEEL_SPAWN_PROFILE="${KEEL_SPAWN_PROFILE:-aws}"
_KEEL_SPAWN_JSON=""
_KEEL_SPAWN_ASKED=0
spawn_probe() {
  local host="$1" n rec
  if [[ "$_KEEL_SPAWN_ASKED" -eq 0 ]]; then
    _KEEL_SPAWN_ASKED=1
    # No region filter: a host in another region must read as itself, not as `none`.
    # The cost is one all-region sweep per process, cached here.
    if command -v "$KEEL_SPAWN" >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
      _KEEL_SPAWN_JSON="$(AWS_PROFILE="$KEEL_SPAWN_PROFILE" "$KEEL_SPAWN" list \
        --state running --output json 2>/dev/null || true)"
      # `[]` is an answer and stays; anything unparseable is not an answer at all.
      jq -e 'type == "array"' >/dev/null 2>&1 <<<"$_KEEL_SPAWN_JSON" || _KEEL_SPAWN_JSON=""
    fi
  fi
  if [[ -z "$_KEEL_SPAWN_JSON" ]]; then printf '?'; return 0; fi
  n="$(jq --arg h "$host" '[.[] | select(.name == $h)] | length' <<<"$_KEEL_SPAWN_JSON" 2>/dev/null || true)"
  case "${n:-}" in
    0) printf 'none' ;;
    1) rec="$(jq -r --arg h "$host" \
         '[.[] | select(.name == $h)][0]
          | "\(.instance_id):\(.instance_type):\(if .spot then "spot" else "ondemand" end)"' \
         <<<"$_KEEL_SPAWN_JSON" 2>/dev/null || true)"
       # An extraction that came back empty after the count said one is a broken
       # instrument, not a record, and it says so rather than printing `::`.
       if [[ -n "$rec" && "$rec" != *"null"* ]]; then printf '%s' "$rec"; else printf '?'; fi ;;
    "") printf '?' ;;
    *) printf 'ambiguous' ;;
  esac
  return 0
}

remote_probe() {
  local host="$1"
  local line
  line="$(ssh "${KEEL_SSH_OPTS[@]}" "$host" "$KEEL_GOV_PROBE_SH"'
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
    virt="?"
    if [ -r /proc/cpuinfo ] && grep -q "^flags" /proc/cpuinfo; then
      if grep -m1 "^flags" /proc/cpuinfo | grep -qw hypervisor; then virt=guest; else virt=metal; fi
    fi
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
    printf "%s | instance=%s | virt=%s | %s cpus | %s cores | smt=%s | %s sockets | governor=%s | tmux=%s | %s | %s\n" \
      "$cpu" "$inst" "$virt" "$ncpu" "$cores" "$smt" "$sockets" "$gov" "$tmux" "$(uname -sr)" "${cache:-caches=?}"
  ' 2>/dev/null || true)"
  # The launcher-side field is spliced in HERE, and only when the host answered.
  # An EMPTY provenance line is the unreachable signal every caller keys on — it is
  # assert_governor's one discriminator between "answered, governor unreadable" and
  # "never answered" (#83). Appending `| spawn=?` unconditionally would make a host that
  # never answered produce a non-empty line, and it would then be classified instead of
  # reported silent: a verdict about identity earned by a machine that said nothing.
  [[ -n "$line" ]] || return 0
  printf '%s | spawn=%s\n' "$line" "$(spawn_probe "$host")"
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

# The EC2 instance types admitted to the evidentiary class: the largest non-metal size in
# each approved family, which is a whole host and therefore a whole socket too — a
# conservative subset of what docs/hosts.md admits, not a restatement of it.
#
# No metal SIZE is listed, and that is not a statement about metal. This list answers "is
# this instance type full size", a question only an instance type has; bare metal reaches
# the same class through its own named arm below rather than by being enumerated here.
# (Until 2026-08-19 this comment read "Metal is absent because metal is retired" — void by
# #109, which reclassified the lab fleet as the dev tier rather than retiring it.)
#
# An allowlist is safe here in the way a duplicated threshold is not, and that is why
# this is one: the default is the RESTRICTIVE class, so a stale list can only withhold a
# judgement, never grant one.
#
# c5n.18xlarge added 2026-08-21, justified by build/wave2-classify-7ac592a.log: read back as
# Xeon Platinum 8124M, 36 cores / 2 sockets, equal to c5n.metal's core count. c6i.32xlarge
# was measured in the same run and is NOT added -- fma-bound at 48.7% of peak, so admitting
# it would grant a judgement that fails P2 (#6 Q3, and see #86).
KEEL_EVIDENTIARY_SIZES="c7i.48xlarge c8i.96xlarge c7a.48xlarge c8a.48xlarge c5n.18xlarge"

# host_admission PROV — set ADM_CLASS and ADM_INSTANCE from a provenance line
# (docs/hosts.md, ruled 2026-08-17 on #104). ADM_CLASS is one of:
#
#   evidentiary  perf may be judged: a floor is a claim about silicon this host owns
#   correctness  perf is reported, never judged, however high or low it reads
#   unknown      the class could not be read, which is `unmeasured` and not a class
#
# The third state is the one that matters: without it, the mechanism that excuses a
# partial-size reading from the floor is also a mechanism for laundering a red past it.
#
# WHAT THE CLASS IS DERIVED FROM: *does this host own its whole socket*, with an approved
# instance type the EC2-specific way of establishing that and bare metal the direct way.
# Five arms, four of them restrictive:
#
#   instance= in KEEL_EVIDENTIARY_SIZES         evidentiary   full size, unchanged
#   instance=none, virt=metal, gov=performance  evidentiary   the bare-metal arm (#106)
#   instance=none, virt=metal, gov anything else   correctness   #79's case
#   instance=none, virt=guest                   correctness   a guest of unknown size
#   instance=none, virt=?                       correctness   the default: unread is unmet
#   instance= anything else                      correctness   not full size
#   instance= empty or ?                        unknown       ⇒ unmeasured
#
# ADDING AN ARM, NOT WIDENING THE DEFAULT (Scott's condition, ruled 2026-08-19 on #106).
# `instance=none` falling through to `correctness` was the right default and the wrong
# classification: unknown provenance failing closed is exactly what a `case` default is
# for, so that arm keeps its behaviour and bare metal gets its own evidence requirements
# beside it. Widening the default would trade a false demotion for a false admission and
# invert the allowlist's safety property — a stale list may only withhold a judgement,
# never grant one — and every unrecognised host thereafter would arrive judged.
#
# WHY THE BARE-METAL ARM CARRIES A GOVERNOR CONJUNCT WHERE THE INSTANCE ARM DOES NOT, so
# the asymmetry is not read as an oversight: the conjunct is what makes this the
# *pre-existing* §5 rule 5 instrument — the one that admitted the lab fleet in the first
# place — rather than a new grant, which is the whole basis of the 2026-08-17 ruling that
# lab admission predates this machinery and stands. A guest has no governor to assert and
# gets rule 5's substitute instrument instead, judged after its sweep by clock_post.
#
# The governor is parsed from THE SAME PROVENANCE LINE as instance= and virt=, not read
# out of assert_governor's GOV_STATE, so all three conjuncts necessarily describe one
# host: a global would let the previous host's reading classify this one, which is the
# hazard GOV_PROV exists to name. It is the same field assert_governor's cascade reads,
# and `evidentiary` here means exactly `GOV_STATE == performance` there — two derivations
# of one fact, so remote-exec-test.sh case 8 pins them to agree over every governor state
# rather than leaving the agreement to be assumed (§5 rule 10).
ADM_CLASS=""
ADM_INSTANCE=""
ADM_VIRT=""
# The launcher's token for this host, parsed from the same line as the other three.
ADM_SPAWN=""
# Why this class, in the words a verdict line can print. It replaces a hardcoded
# parenthetical in adm_judgeable that said "$ADM_INSTANCE is not a full-size instance of
# an approved family" for every correctness host — true of a 4xlarge and false of bare
# metal under powersave, which would have been refused with a cause it did not have.
ADM_WHY=""
host_admission() {
  local prov="${1-}" gov stype smarket
  ADM_INSTANCE="$(sed -n 's/.*instance=\([^ |]*\).*/\1/p' <<<"$prov")"
  ADM_VIRT="$(sed -n 's/.*virt=\([^ |]*\).*/\1/p' <<<"$prov")"
  ADM_SPAWN="$(sed -n 's/.*spawn=\([^ |]*\).*/\1/p' <<<"$prov")"
  gov="$(sed -n 's/.*governor=\([^ |]*\).*/\1/p' <<<"$prov")"
  case "$ADM_INSTANCE" in
    "" | "?")
      ADM_CLASS=unknown
      ADM_WHY="no instance identity was read from this host" ;;
    none)
      if [[ "$ADM_VIRT" == metal && "$gov" == performance ]]; then
        ADM_CLASS=evidentiary
        ADM_WHY="bare metal (no hypervisor flag) with governor=performance: a whole machine owns its socket more completely than any instance type can demonstrate"
      elif [[ "$ADM_VIRT" == metal ]]; then
        ADM_CLASS=correctness
        ADM_WHY="bare metal, but its clock is not established: governor=${gov:-unread}, not performance (§5 rule 5)"
      elif [[ "$ADM_VIRT" == guest ]]; then
        ADM_CLASS=correctness
        ADM_WHY="hypervisor flag present and no EC2 identity: a guest whose size this run cannot see"
      else
        ADM_CLASS=correctness
        ADM_WHY="no EC2 identity and virt=${ADM_VIRT:-unread}, so neither route to whole-socket ownership was established"
      fi ;;
    *) case " $KEEL_EVIDENTIARY_SIZES " in
         *" $ADM_INSTANCE "*)
           # An approved type is now NECESSARY AND NOT SUFFICIENT. Two conjuncts,
           # both from the launcher-side field and neither obtainable from the host:
           #
           #   (a) the launcher must agree this is that type. The size is the whole basis
           #       of this arm, and until 2026-08-19 the only witness of it was a reading
           #       taken inside the guest.
           #   (b) it must be on-demand. Ruled 2026-08-19: "on-demand for judged runs
           #       because interruptions corrupt measurements, which is the only reason."
           #       Until now that was an ASSUMPTION carried in aws-fleet.sh's
           #       KEEL_FLEET_MARKET — a variable the launcher declared and no gate read
           #       back. remote.sh's own test for an assumption is "is there any mechanism
           #       by which this gate could read the precondition back? If yes, it is a
           #       criterion this gate is missing", and spawn's record is that mechanism,
           #       so the assumption becomes a conjunct.
           case "$ADM_SPAWN" in
             # TWO ABSENCES, NOT ONE, and they were one line until the distinct-causes
             # check in remote-exec-test.sh refused a list in which they shared a string.
             # `virt=` already distinguishes them (via `${ADM_VIRT:-unread}`) and the
             # distinction is not cosmetic: an absent field says the line was produced by
             # a driver from before the launcher was a witness, which is fixed by
             # upgrading the driver; `?` says the probe ran and could not answer, which is
             # fixed by installing jq or supplying credentials. One remedy each.
             "")
               ADM_CLASS=correctness
               ADM_WHY="$ADM_INSTANCE is an approved full-size type, but its provenance line carries no spawn= field at all, so it was produced before the launcher became a witness and this run cannot ask who launched it" ;;
             "?")
               ADM_CLASS=correctness
               ADM_WHY="$ADM_INSTANCE is an approved full-size type, but the launcher could not be consulted (no spawn, no jq, or no credentials), and the judged tier is defined over what truffle/spawn launched — unread is unmet, exactly as it is for virt=" ;;
             none)
               ADM_CLASS=correctness
               ADM_WHY="$ADM_INSTANCE is an approved full-size type, but the launcher has no running record under this host's name, so its size rests on the guest's own testimony alone" ;;
             ambiguous)
               ADM_CLASS=correctness
               ADM_WHY="$ADM_INSTANCE is an approved full-size type, but more than one running launcher record answers to this host's name, and a join that picked one would attribute a reading to a machine that may not have produced it" ;;
             *)
               stype="${ADM_SPAWN#*:}"; smarket="${stype#*:}"; stype="${stype%%:*}"
               if [[ "$stype" != "$ADM_INSTANCE" ]]; then
                 # THE ONLY ARM THAT REACHES `unknown` THROUGH A READING RATHER THAN AN
                 # ABSENCE, and it is the reason for having a second witness at all. Two
                 # instruments that disagree do not average: the identity this arm needs
                 # is in dispute, so there is no class to read, and `correctness` would
                 # quietly grade the host as "small" when what is actually broken is the
                 # instrument. unmeasured, loudly.
                 ADM_CLASS=unknown
                 ADM_WHY="the two witnesses of this host's identity disagree: the guest reports instance=$ADM_INSTANCE and the launcher records it as $stype. One of them is wrong and this run cannot say which"
               elif [[ "$smarket" != ondemand ]]; then
                 ADM_CLASS=correctness
                 ADM_WHY="$ADM_INSTANCE, and the launcher agrees, but it was launched $smarket: a reclaim mid-sweep converts a judged reading into a truncated one, so spot is the exploration tier (ruled 2026-08-19)"
               else
                 ADM_CLASS=evidentiary
                 # NAMES THE CHECK, not a property this arm cannot test -- the same repair
                 # the rejection arm got, and it needed the same kind of evidence. The
                 # wording was "$ADM_INSTANCE is a full-size instance of an approved
                 # family", which is a claim about SIZE that nothing here evaluates:
                 # membership in one flat list is the whole test. Driving it on a live
                 # c7a.medium with that type temporarily admitted produced "c7a.medium is a
                 # full-size instance" in a real preamble. The list's members are in fact
                 # full-size, so the sentence was true of every host it had ever run on --
                 # which is exactly why only a positive control on a deliberately wrong
                 # type could show it, and why a healthy fleet would never have.
                 ADM_WHY="$ADM_INSTANCE is admitted to the evidentiary class by KEEL_EVIDENTIARY_SIZES, whose members are full-size types added one justifying read-back at a time, and the launcher independently records this host as $stype on-demand (${ADM_SPAWN%%:*})"
               fi ;;
           esac ;;
         *)
           ADM_CLASS=correctness
           # NAMES THE CHECK THIS CODE PERFORMED, which is membership in one flat list,
           # and not a cause it cannot test. The old wording — "not a full-size instance
           # of an approved family" — is a conjunction over two properties the classifier
           # never separates, and driving it against real launcher output produced it for
           # `c8g.48xlarge`, which IS full size and was rejected for its family. Same
           # defect as the one just fixed one arm down: a sentence that cannot say which
           # of its causes fired. Distinguishing them for real would need a family list
           # plus a size rule, so the honest fix is the sentence, not the classifier.
           ADM_WHY="$ADM_INSTANCE is not among the types admitted to the evidentiary class (KEEL_EVIDENTIARY_SIZES); a type is added there when a read-back on it justifies the addition, so absence covers both a partial size and a family never characterized" ;;
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
  # Every input the classifier read, `spawn=` included: a preamble that showed two of the
  # three inputs would present a `correctness` verdict with no visible cause the run after
  # spawn stops answering, which is the failure mode this line exists to prevent.
  info "[$1] admission class: $ADM_CLASS (instance=${ADM_INSTANCE:-unread}, virt=${ADM_VIRT:-unread}, spawn=${ADM_SPAWN:-unread}) — $ADM_WHY"
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
      info "[$1] correctness-class ($ADM_WHY): $3 — reported, not judged (docs/hosts.md)" ;;
    *)
      # $ADM_WHY, not a parenthetical retyped here. This arm was written when `unknown`
      # had exactly one cause — an absent identity — so naming that cause inline was
      # true. It no longer is: since the launcher became a second witness, `unknown` is
      # also reached when the two witnesses CONTRADICT each other, and the old wording
      # reported that as "absent from the provenance line" when the line in fact carried
      # two identities. A sentence that can only describe one of its causes is #37's
      # label defect (§5 rule 6: one cause, one verdict — and the verdict must be able
      # to say which cause it had).
      # "cannot settle", not "unread", for the same reason the parenthetical became
      # $ADM_WHY one line up: the tail was written when absence was the only route here,
      # and a disputed identity is emphatically read -- twice, differently. Both routes
      # share one consequence, which is what this clause is for.
      unmeasured "[$1] the admission class is unreadable ($ADM_WHY), so this criterion is unmeasured here rather than cleared or missed: an identity this run cannot settle must not be the mechanism that excuses a reading from a floor" ;;
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

  # Placement (DESIGN section 5 rule 5, adopted fleet-wide 2026-08-21). A benchmark
  # invocation is one carrying -test.bench; see KEEL_PIN_SH for why that is the line and
  # why the failure is a refusal. The predicate reads the arguments rather than a caller
  # flag, so a new sweep gets the mask by asking for benchmarks and cannot forget to.
  local a2 wants_bench=0
  for a2 in "$@"; do
    case "$a2" in -test.bench=*|-test.bench) wants_bench=1 ;; esac
  done
  local pin_pre=""
  if [[ "$wants_bench" -eq 1 ]]; then
    pin_pre="
if ! command -v taskset >/dev/null 2>&1; then
  echo 'keel-pin: REFUSED, no taskset on this host. DESIGN section 5 rule 5 pins fleet-wide and never selectively, so this measurement is not taken rather than taken unpinned.' > '$log'
  printf '%s\n' 121 > '$st'; exit 121
fi
KEEL_MASK=\$(keel_pin_mask $KEEL_PIN_WIDTH) || KEEL_MASK=
if [ -z \"\$KEEL_MASK\" ]; then
  echo 'keel-pin: REFUSED, no $KEEL_PIN_WIDTH distinct physical cores inside a single NUMA node could be selected from sysfs on this host. Not taken rather than taken unpinned (DESIGN section 5 rule 5).' > '$log'
  printf '%s\n' 121 > '$st'; exit 121
fi
PIN=\"taskset -c \$KEEL_MASK\"
PINLINE=\"keel-pin: mask=\$KEEL_MASK width=$KEEL_PIN_WIDTH\""
  fi

  local tmpd; tmpd="$(mktemp -d)"
  # `\$?` is escaped so the status is read on the far side, at the far side's
  # moment; expanded here it would freeze this shell's last exit code into the
  # script and report it as the measurement's.
  #
  # PIN and PINLINE are empty for every non-benchmark invocation, so those runs exec
  # exactly what they used to and their LOGS are byte-identical -- the runner script is
  # not, since it carries the selector either way, and remote-exec-test.sh case 1 checks
  # the log rather than taking the claim. The `keel-pin:` line is printed INSIDE the block,
  # immediately before
  # the binary, so it lands in the log the caller reads and `$?` still belongs to the
  # measurement -- the mask is then recoverable from the same artifact as the numbers it
  # shaped, instead of from this driver's memory of what it asked for.
  cat > "$tmpd/$id.sh" <<EOF
#!/bin/sh
# Generated by remote_exec (keel #62). Not edited by hand and not reused: one
# runner per measurement, named for the tmux session that supervises it.
cd '$KEEL_REMOTE_DIR' || exit 127
PIN=; PINLINE=
$KEEL_PIN_SH
$pin_pre
{ [ -n "\$PINLINE" ] && echo "\$PINLINE"; env ${KEEL_REMOTE_ENV:-} \$PIN ./'$base'$args; printf '%s\n' "\$?" > '$st'; } > '$log' 2>&1
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

