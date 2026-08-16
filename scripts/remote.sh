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
# lines by grepping for it (gate-p4.sh:1051, gate-p5.sh:1133).
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
# (gate-p5.sh:531), as are a boost knob that did not move and a boost knob
# nobody could read (gate-p5.sh:839), a forced run that reported its dispatch
# before failing and one that never got that far (gate-p5.sh:597), and a governor
# that changed mid-run and a governor unreadable mid-run (gate-p3.sh:1231,
# gate-p4.sh:861, gate-p5.sh:817). Each citation is the line of the `if`, and the
# last of them arrived a commit later than the rest: gate-p5's copy printed
# '${gov:-unknown}' inside one collapsed branch, so a sweep that read messages
# could not see it, and #76's guard is what made the branch reachable at all.
unmeasured() { printf '  \033[33mUNMEASURED\033[0m  %s\n' "$1"; FAIL=1; }

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
# files, #82, which gate-p5.sh:100 already argues its own floor from). Both are
# checkable, therefore both are absent criteria rather than assumptions, and both
# are filed as such. A ledger that grows by absorbing checkable things is a way of
# writing "we did not get around to it" in a font that reads as rigour.
#
# The word FAIL/PASS/UNMEASURED must not appear in the label, for the same reason
# it must not in unmeasured(): the delegating gates count verdict lines by
# grepping anchored labels (gate-p4.sh:1052, gate-p5.sh:1143). The eight-space
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
# (remote_boost, gate-p3's ob_preflight and ob_coretype_sweep).
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

# assert_governor HOST PHASE [PROV] — the §5 rule 5 performance-governor
# precondition, asserted per host, defined once for all five measuring gates.
#
# PHASE  preamble | measured. Selects wording, and whether a provenance line is
#        printed. The preamble prints a verdict on every outcome including PASS, so
#        a `governor=` info line there would only restate it; at measurement time
#        the success path is deliberately silent (the preamble already passed that
#        host), so the info line is the only surviving record of what was read.
# PROV   pre-captured remote_probe output. Omit it and this probes the host itself;
#        pass it — gate-p5 needs the same reading for #66's boost knob — and no
#        second ssh happens. Passing an *empty string* is meaningful and is not the
#        same as omitting the argument: it says "already probed, and the host
#        produced nothing", which is the distinction this whole function is about.
#
# Sets, for the caller's control flow and its provenance text:
#   GOV_STATE   performance | wrong | unreadable | unreachable
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
# a `§5.4` citation naming a section DESIGN.md does not have (#85), an
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
assert_governor() {
  local host="$1" phase="${2:-preamble}" prov
  if [[ $# -ge 3 ]]; then prov="$3"; else prov="$(remote_probe "$host" || true)"; fi

  GOV_STATE=""; GOV_VALUE=""; GOV_SHOWN=""
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
    elif [[ -z "$GOV_VALUE" || "$GOV_VALUE" == unknown ]]; then
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
