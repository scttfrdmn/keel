#!/usr/bin/env bash
# Gate P3 — see DESIGN.md §4/P3. Written at the START of phase P3, then made
# green. Exits 0 only when every criterion for the phase holds. A red gate
# blocks the next phase; there is no override flag on purpose.
#
# Criteria (verbatim from DESIGN.md §4/P3):
#   "full `Sgemm` matches oracle across a size sweep (1..17, 63,64,65, 500,
#    1000, 2048, plus transpose/beta/alpha combinations); single-thread >=60%
#    of OpenBLAS at 2048^3 (bench harness pulls OpenBLAS via build-tagged cgo,
#    dev machine only, never a package dependency)."
#
# How those are mechanized, and every judgement call involved:
#
#  1. THE SWEEP'S EXTENT IS ENFORCED, NOT TRUSTED. "matches oracle across a size
#     sweep" is a claim about coverage, and a green `go test` proves only that
#     whatever ran, passed. A test that quietly skipped 2048 because it was slow,
#     or that ran one transpose combination instead of four, reports the same
#     green. So the tests print coverage markers and this gate parses them: every
#     size in DESIGN.md's list must appear, the transpose lattice must be complete
#     (NN/NT/TN/TT), and alpha and beta must each include 0, 1 and a value that is
#     neither — 0 and 1 are the special-cased paths, so a lattice of only those
#     would exercise every shortcut and never the general multiply. The enumerated
#     sets must also multiply out to the reported combination count, so the marker
#     cannot claim combinations it did not run.
#
#  2. THE ORACLE'S COST IS A DECLARED PROPERTY OF EACH SIZE, NOT A SILENT
#     DEGRADATION. A float64 oracle at 2048^3 is 8.6 GFLOP per combination, and
#     the full lattice of that is hours. The sweep is therefore allowed to verify
#     the large sizes by checking a seeded random *sample* of C entries, each one
#     computed exactly in float64 — but only if it says so per size, in a marker
#     this gate reads. Sizes up to 65 must be verified in full; 500, 1000 and 2048
#     may be sampled, with a floor on the sample size and a printed seed so any
#     failure is replayable. The distinction is written here rather than left to
#     the test, because "we sampled" is exactly the kind of concession that starts
#     at 2048 and ends up applying to 17.
#
#  3. CORRECTNESS RUNS WHERE THE SHIPPED PATH RUNS. The dev host is darwin/arm64
#     and cannot execute archsimd at all (docs/toolchain-notes.md T1), so a local
#     `go test` exercises the scalar path and nothing else. The sweep's extent is
#     therefore audited from the log of a host that ran it with the AVX-512
#     backend live, and the scalar path is proved separately by the local run and
#     by a KEEL_FORCE=scalar run on an amd64 host — the P1 mechanism, because the
#     only way to show the fallback works on a machine that has AVX-512 is to make
#     it take the fallback.
#
#  4. P2's KERNEL PROPERTIES ARE RE-CHECKED HERE, BECAUSE P3 IS WHAT WOULD BREAK
#     THEM. Packing, edge handling and beta variants all add code around the
#     K-loop, and the failure mode is a loop that acquired a spill, a call or a
#     bounds check on the way to becoming usable. Those are compile-time
#     properties, so the audit is cheap and it runs on every gate from here on.
#     This is a carried-forward criterion, not a new one, and it does not become
#     optional because P3's own checks are green.
#
#  5. THE THROUGHPUT SENTINEL. The ruling on issue #19 made P2's floor
#     class-dependent: on an issue-bound host it is 0.90 x max_i(f_i.I_i) / I_b,
#     which *rises* as the kernel's instruction count falls, and by the same
#     arithmetic that host is the one that notices when a shape gets fatter. P3 is
#     the phase most likely to fatten one. So P2's verdict is re-run here, through
#     the same unit-tested pure function (scripts/roofline.sh) driven by the same
#     fixtures, which run before any benchmarking.
#
#     The sentinel is named by $KEEL_SENTINEL_HOST or .keel-sentinel (machine-local
#     and gitignored, like .keel-hosts: real hostnames are infrastructure, not
#     source — docs/hosts.md). If neither names one, the check runs on EVERY host.
#     Missing configuration must cost time, not coverage; a gate that skipped its
#     regression check because a file was absent would be a gate that got weaker
#     when someone cloned the repo.
#
#     IF THIS CRITERION READS RED, THE POLICY IS IN DESIGN.md §4 AND IT IS ONE
#     RE-RUN. The issue-bound roofline check is the tightest margin in this gate --
#     two independent runs on janus read 95.0% and 93.7% against a 90% floor -- so a
#     single dip is within the variance the bar was set above. Exactly one immediate
#     re-run; the criterion fails only if both runs fail; both outputs go into the
#     umbrella issue verbatim either way, so a pass on the second reading says so in
#     the record. Not a loop, and not available to any correctness criterion above:
#     a differential test that passes on retry has found a nondeterminism, which is
#     worth more attention than the gate colour rather than less. This script does
#     not implement the retry -- it fails on the first miss, and the allowance is the
#     operator's, because an auditable human re-run beats a script trusted to
#     confess that it retried.
#
#  5b. THE SHAPE MEASURED IS THE SHAPE DISPATCH SELECTS (ruling on issue #24).
#     P2 shipped two zero-spill shapes and neither dominates: 4x32 wins on Zen 4
#     and Zen 5, 2x32 wins on Skylake-X by 11 percentage points (KERNEL.md §7).
#     Dispatch now chooses per host, from the same issue-bound/FMA-bound
#     classification this gate's own model defines — FMA-bound takes the fewest
#     memory ops per FMA, issue-bound the fewest instructions per FMA. That fixed
#     the performance bug criterion 6b's shape guard found (janus was shipping
#     4x32), and it opens three ways for a gate to be lied to, so it closes all
#     three:
#
#       - EVALUATING ONE KERNEL WHILE SHIPPING ANOTHER. Criterion 5 used to judge
#         P2's floor against whichever shipped shape measured fastest on the host.
#         With a per-host choice that is no longer the same thing as the shape
#         `Sgemm` runs, so the sentinel now judges the DISPATCHED shape, named by
#         the keel-bench-kern marker of the run being judged. The other shapes
#         become the ceiling set, which is what INDEPENDENCE in scripts/roofline.sh
#         asks for anyway.
#
#       - A WRONG CHOICE PASSING QUIETLY. Both shapes are benchmarked in one
#         invocation, so "did dispatch pick the faster one" is answerable from the
#         same measurement: if another shape's lower bound exceeds the dispatched
#         shape's point estimate, this gate fails. That direction is the one the
#         classification can be wrong in without any other criterion noticing —
#         see internal/kern/class_amd64.go, which documents both error directions
#         and why the other one only ever makes this gate stricter. The margin is
#         CI-based rather than exact so the check cannot flake on noise.
#
#         The blocked `Sgemm` is then re-measured under KEEL_KERN_CLASS pinned to
#         the other class, and the dispatched shape must be no slower there either.
#         That run is a second invocation — weaker evidence than the same-run
#         kernel comparison above, and it is a cross-check, not the criterion —
#         but it is the only way to see the choice through packing and blocking,
#         which is where P3's number actually comes from.
#
#       - A CLASSIFICATION NOBODY CHECKED. The library cannot read a
#         microarchitecture: archsimd exposes CPU features and no vendor, family or
#         model, so HostClass fingerprints the generation from a feature bundle
#         (docs/toolchain-notes.md T14, issue #25). A proxy that decides what ships
#         must be checked against something measured, so the library prints its
#         classification and its grounds (keel-bench-kern-class) and this gate
#         compares them against the class its own convergence test derives from the
#         measurements. Disagreement is a gate failure, on every host, every run.
#
#     And because the ranking reads an audited instruction count recorded in Go
#     source, that number is re-derived from the object code here: every shipped
#     shape's Kernel.InsnsPerFMA must equal what the spill audit counts in its loop
#     body. A measurement in source that nothing recomputes is a measurement that
#     drifts, and this one decides what ships.
#
#  6. THE OPENBLAS BAR, AND WHICH READING OF "DEV MACHINE ONLY" THIS IMPLEMENTS.
#     DESIGN.md says the OpenBLAS harness is cgo behind a build tag, "dev machine
#     only, never a package dependency". Read as "measure on the dev machine" the
#     criterion is vacuous here: this dev machine is darwin/arm64, where keel's
#     shipped AVX-512 path does not exist, so the ratio would compare
#     OpenBLAS-on-arm64 against keel's scalar fallback and answer a question
#     nobody asked. DESIGN.md §7 rule 7 cuts the same way — a ratio whose two
#     halves come from different silicon is not a ratio.
#
#     So this gate implements the other reading: the *comparison* is dev-only —
#     built behind the `openblas` tag, absent from the module's dependency graph,
#     never linked into anything keel ships — and it runs on the amd64 hosts where
#     both halves can execute. Per the ruling on issue #23 that is EVERY gate host,
#     each against its own OpenBLAS: no cross-host reference and no golden machine,
#     because the only apples-to-apples ratio is same silicon, same thread count,
#     same run. Concretely:
#       - the reference and keel's Sgemm are measured in the SAME benchmark
#         invocation on the SAME host, so they share a frequency and a thermal
#         state. P2's criterion 5 settled this: a ratio of two numbers taken from
#         separate runs is a worse measurement than either of them.
#       - the harness is built natively on that host from `git archive HEAD`, so
#         the number is attributable to a commit; the working tree must be clean
#         or the archive does not mean what it says.
#       - single-thread is enforced on both sides and *verified* from the
#         harness's own report (OpenBLAS's thread count and GOMAXPROCS). A
#         multi-threaded OpenBLAS would enlarge the denominator and make the bar
#         harder rather than easier — but it would also make it meaningless, and
#         refusing meaningless numbers in both directions is this gate's job.
#       - the reference's own configuration is checked, not just recorded. A
#         DYNAMIC_ARCH build that selected a pre-AVX2 kernel on an AVX-512 host
#         reads low, and a low reference *inflates* keel's ratio — the one
#         direction in which a gate must never fail. So the selected kernel name
#         must be an AVX2-or-better target, and an unrecognized name fails too:
#         missing knowledge should cost a human a minute, not silently widen a bar.
#       - and the family is not merely allowed, it is CHOSEN BY MEASUREMENT
#         (ruling on issue #31). The reference is the fastest of a coretype sweep
#         forced through OPENBLAS_CORETYPE, best-of-N under this gate's own
#         methodology, pinned for the run that produces the ratio, with every
#         candidate's rate printed and the pin verified from the library's reported
#         corename. The allowlist is the floor; the sweep is the ceiling. It exists
#         because DYNAMIC_ARCH dispatches on an ISA feature bit: on vesta's Zen 4
#         it ships a full-width Cooperlake kernel onto a double-pumped 256-bit
#         datapath, where the AVX2 Haswell kernel is 6.7% faster — keel's own #24
#         bug with the vendors reversed. Trusting that selector would delegate the
#         denominator's one inviolable property to a heuristic this project has
#         measured to be wrong on one of its three hosts.
#       - a host needs a Go toolchain and OpenBLAS for that, and the execution
#         hosts deliberately have neither (docs/hosts.md: cross-compiled static
#         binaries, nothing installed). Provisioning is Scott's to approve, so a
#         host that cannot produce a reference FAILS this gate and gets the exact
#         commands for its own distro printed. It does not fall back to
#         percent-of-peak. CLAUDE.md's "the OpenBLAS reference when available;
#         otherwise say it isn't" is a rule about reporting numbers; using it to
#         satisfy a gate criterion would be weakening the gate, which is the one
#         option never available.
#
#     EVERY gate host must produce a reference and clear the bar, and every host
#     must be on the performance governor to be measured at all — asserted in a
#     preamble before any benchmark runs, not assumed and not noted afterwards
#     (DESIGN.md §5.4 rule 5, tightened by the ruling with #31). The old wording
#     here was "at least one must clear it under the performance governor", which
#     let antares contribute numbers from `powersave`: its first OpenBLAS reading
#     of the sweep was 245.0 GFLOP/s against a 296-297 steady state, i.e. an 18%
#     error in a denominator, decided by how recently the core had been busy. An
#     unreadable governor fails too, on the same principle as the unrecognized
#     kernel name: an unchecked precondition is not a met one. Percent
#     of measured peak is printed for every host either way, because that number is
#     informative even where it is not a criterion — and so is RETENTION, the share
#     of its own microkernel the blocked loop nest keeps, printed on the same
#     reported-never-judged footing. It is there because #26 is a named P5 input
#     (DESIGN.md §4/P5) and an input needs a number: janus keeps ~77% where the Zen
#     hosts keep 90-92%, and P5 should inherit that as a measurement it can re-run
#     rather than a figure someone remembers. Judging it here would be P3 annexing
#     P5's blocking-parameter work; printing it is how P3 hands that work over.
#
#  6b. THE DENOMINATOR ON AN ISSUE-BOUND HOST, AND WHY IT IS NOT A CONCESSION.
#     Also from the ruling on #23: where the P2 classifier says a host is
#     issue-bound, the denominator is min(same-host OpenBLAS, roofline x measured
#     peak). OpenBLAS's K-loop there is hand assembly folding accumulation and an
#     embedded broadcast into single FMAs — instructions the intrinsic layer
#     provably cannot emit (T12, #17/#18) — so it sits above the front-end ceiling
#     keel's kernels are capped by, and 60% of it is a demand on the decode stage.
#
#     The decision is the pure function `p3_denominator` in scripts/roofline.sh,
#     driven by the same fixtures as P2's verdict and running before any
#     benchmarking. It is one-sided (it can only lower a denominator, never raise
#     one), it applies only to a host the classifier admitted on independent
#     evidence, and it carries P2's anti-vacuity shape guard against the shape
#     `Sgemm` ACTUALLY RAN — read from the keel-bench-kern marker of the very run
#     that produced the ratio, not from the best shape on the shelf and not from a
#     different host's log. A fatter kernel therefore cannot buy itself a lower bar.
#     That guard is what surfaced issue #24: it refused the then-dispatched 4x32 on
#     janus, and the fix was dispatch, not the threshold — janus now ships 2x32 at
#     4.625 insns/FMA and is inside the guard on its merits (criterion 5b). Both
#     ratios, amended and plain, are printed on every host: the gate's own leniency
#     is a number, and §7 rule 7 applies to it too.
#
#  7. WHAT THIS GATE DOES NOT CHECK. "Beta handling as kernel variants, not
#     branches in the loop" and "packing SIMD-accelerated through the shim" are
#     P3 design instructions, not gate criteria. Both appear here as provenance —
#     the variant count and the packing backend come out of the config marker —
#     and criterion 4 enforces them structurally, since a branch or a call that
#     landed in the K-loop is exactly what the audit reports. Anything stronger
#     would be this gate inventing criteria the design document did not set.
#
#     Retention (#26) is in this category too, deliberately: it is measured and
#     printed on every host and judged on none. The gap it names is real throughput,
#     but it is packing and memory traffic rather than the front-end ceiling the
#     roofline models, so it is neither excused by the amendment nor netted out of
#     it — and closing it means sweeping KC/MC/NC, which DESIGN.md §4 parked in P5.
#     Turning it into a P3 criterion would be scope migration; dropping it would
#     leave P5 without a baseline. Printing it does neither.
set -euo pipefail

cd "$(dirname "$0")/.."
# shellcheck source=scripts/remote.sh
source scripts/remote.sh
# shellcheck source=scripts/bench.sh
source scripts/bench.sh
# shellcheck source=scripts/roofline.sh
source scripts/roofline.sh

FAIL=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAIL=1; }
info() { printf '        %s\n' "$1"; }

# ---------------------------------------------------------------- the sweep
# DESIGN.md §4/P3's list, verbatim. 1..17 covers every M and N remainder against
# any MR/NR a kernel might ship; 63/64/65 straddle a power of two; 500 and 1000
# are multiples of no blocking parameter; 2048 is the size the throughput
# criterion is stated at.
SWEEP_SIZES="1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 63 64 65 500 1000 2048"
# Sizes that must be verified against the oracle in full, not sampled.
SWEEP_EXACT_MAX=65
# Minimum sampled entries for a size above that: 256 exact float64 dot products
# spread over C by a seeded RNG.
SWEEP_SAMPLE_MIN=256
# The transpose lattice must be complete: a packing bug that transposes the wrong
# operand shows up in exactly one of these four.
SWEEP_TRANS="NN NT TN TT"

# ------------------------------------------------------ carried from P2 (#19)
# Same shapes, same audit, same pure verdict function and the same constants as
# gate-p2.sh. Duplicated rather than factored into a shared file on purpose: a
# P3 run should fail because P3 changed the kernel, not because someone edited
# one shared constant, and two independent statements of the threshold is what
# makes that visible.
KERN_PKG="./internal/vec"
KERN_FUNCS="Kernel2x32,Kernel4x32"
PEAK_FUNCS="avx512Peak,avx2Peak,scalarPeak"
PEAK_FLOOR=0.55
ROOF_FLOOR=0.90
ISSUE_CONVERGE_MAX=1.10
ISSUE_MIX_SPREAD_MIN=1.25
SWEEP_BEST_IPF=4.438
ROOF_SHAPE_SLACK=1.05
GATE_KERNELS="Kernel/2x32/avx512/kc=128 Kernel/4x32/avx512/kc=128"
GATE_PEAK_FUNC="avx512Peak"
KERN_BENCH_FILTER='Peak|Kernel/.*/.*/kc=128'
SSADIR="build/ssa"

# ------------------------------------------------------------- P3's own bar
OPENBLAS_FLOOR=0.60
GATE_SGEMM="Sgemm/n=2048"
GATE_OPENBLAS="OpenBLAS/n=2048"
GATE_PEAK="Peak/avx512"
# THE PARENTHESES ARE LOAD-BEARING (issue #32, docs/toolchain-notes.md T15).
# `go test -bench` does not match one regexp against the full name: it splits on
# top-level '|' FIRST, into an alternation of whole patterns, and only then splits
# each alternative on '/'. So the previous value here —
#   Peak|Sgemm|OpenBLAS/avx512|n=2048
# — was four alternatives, {Peak}, {Sgemm}, {OpenBLAS,avx512}, {n=2048}, and not the
# two-level filter its comment claimed. {OpenBLAS,avx512} can never match, because
# BenchmarkOpenBLAS's children are n=..., so THE REFERENCE BENCHMARK NEVER RAN and
# criterion 6 had no denominator to divide by; {Peak} and {Sgemm} were
# depth-unconstrained, so the gate also paid for three Sgemm sizes and two Peak
# variants it never reads. Parenthesized, the '|'s are ordinary regexp alternation
# inside one two-element pattern, and this runs exactly the three benchmarks the
# gate reads: Peak/avx512, Sgemm/n=2048, OpenBLAS/n=2048.
SGEMM_BENCH_FILTER='(Peak|Sgemm|OpenBLAS)/(avx512|n=2048)'
# Criterion 5b's cross-check run: the blocked Sgemm only, with the shape pinned to
# the other class. No Peak and no OpenBLAS, because what is wanted from it is one
# rate to compare against the dispatched shape's rate — not a ratio.
SGEMM_SHAPE_FILTER='Sgemm/n=2048'
OPENBLAS_REMOTE_DIR="${KEEL_OPENBLAS_DIR:-/tmp/keel-openblas-src}"

# The OpenBLAS kernel families that are AVX2-or-better on x86-64, lowercased.
# DYNAMIC_ARCH picks one at load time and reports it through
# openblas_get_corename(); anything not on this list — a generic build, a
# pre-AVX2 family, or a name added by a future OpenBLAS — fails the criterion.
#
# An allowlist rather than a denylist, deliberately, and it is the strictness that
# is the point: a reference that quietly runs a Nehalem kernel on a Skylake-X host
# is slow, and a slow reference makes keel look good. Every other failure mode of
# this gate costs a session; this one would cost the truth of a number. A new
# legitimate target name failing here costs whoever sees it one line of diff.
OPENBLAS_OK_CORES="haswell skylakex cooperlake sapphirerapids zen"

# The coretype candidates the reference is swept over, and the reason the sweep
# exists at all (ruling on issue #31).
#
# The denominator is the FASTEST MEASURED same-host OpenBLAS across these, not
# whatever DYNAMIC_ARCH selected at load time. Measured on the three gate hosts:
# on vesta's Zen 4, DYNAMIC_ARCH picks an Intel-tuned Cooperlake kernel and the
# AVX2 Haswell kernel is 6.7% faster (159.5 vs 149.5 GFLOP/s), because Zen 4
# double-pumps AVX-512 through a 256-bit datapath. That is the same physics that
# makes keel dispatch 2x32 on an issue-bound host: upstream's dispatch consults an
# ISA feature bit, ours classifies the machine. Their heuristic is wrong on one of
# our three hosts, and a reference chosen by it is a denominator that flatters keel
# — the one property this denominator must never have.
#
# The allowlist above keeps its narrowed job (reject an ancient family: the FLOOR).
# The sweep supplies the CEILING. Neither substitutes for the other: an allowlist
# cannot tell "correct for this silicon" from "merely modern", and this one contains
# both the right and the wrong answer for every host here.
#
# A superset of the ruled set {default, Zen, Haswell, SkylakeX-where-valid}, because
# adding candidates can only raise the ceiling and never lower it, and because
# "where valid" is established by RUNNING each candidate rather than by consulting a
# table: one that cannot execute here is recorded as unavailable, not assumed absent.
OPENBLAS_CORETYPES="default Zen Haswell SkylakeX Cooperlake SapphireRapids"

# sentinel_hosts prints the hosts the P2 throughput regression re-runs on:
# whatever is configured, else every host (criterion 5).
sentinel_hosts() {
  if [[ -n "${KEEL_SENTINEL_HOST:-}" ]]; then
    tr ' ' '\n' <<<"$KEEL_SENTINEL_HOST" | sed '/^$/d'
    return
  fi
  if [[ -r .keel-sentinel ]]; then
    sed -e 's/#.*//' -e '/^[[:space:]]*$/d' -e 's/[[:space:]]//g' .keel-sentinel
    return
  fi
  remote_hosts
}

# marker NAME FILE — the value of the last `keel-NAME:` line in FILE. Test output
# arrives through t.Logf, so the marker may be indented and prefixed.
marker() { sed -n "s/.*keel-$1: *//p" "$2" | tail -1; }

# marker_all NAME FILE — every `keel-NAME:` value, one per line, for the markers
# emitted once per size.
marker_all() { sed -n "s/.*keel-$1: *//p" "$2"; }

# field KEY LINE — the value of a `key=value` token in a marker line.
field() {
  awk -v k="$1" '{
    for (i = 1; i <= NF; i++) {
      n = index($i, "=")
      if (n && substr($i, 1, n - 1) == k) { print substr($i, n + 1); exit }
    }
  }' <<<"$2"
}

# audit_ipf FUNC FILE — that function's audited instructions per FMA, from the
# audit's own integer counts. gate-p2.sh carries the same helper for the same
# reason: this number is a gate input, so it is not read off a rounded display.
audit_ipf() {
  awk -v fn=".$1: steady-state loop" '
    index($0, fn) {
      for (i = 2; i <= NF; i++) {
        if ($i == "insns") ins = $(i-1)
        if ($i == "arith") ar  = $(i-1)
      }
      if (ins != "" && ar != "" && ar + 0 > 0) { printf "%.6f", ins / ar; exit }
    }' "$2"
}

# other_class CLASS — the class criterion 5b pins KEEL_KERN_CLASS to: the one
# dispatch did not choose on this host, so the cross-check measures the shape that
# was passed over. An unrecognized class yields nothing and the caller skips the
# cross-check with a reason rather than silently comparing a shape against itself.
other_class() {
  case "$1" in
    issue) printf 'fma' ;;
    fma)   printf 'issue' ;;
  esac
}

# audit_ipf_tile TILE FILE — the audited insns/FMA for a shape named as it appears
# in a marker ("4x32"), via the convention that internal/vec's loop body for that
# shape is KernelTILE. One place, because the sentinel and criterion 6b both need
# the mapping and a mismatch between them would be invisible.
audit_ipf_tile() {
  [[ -n "$1" ]] || return 0
  audit_ipf "Kernel$1" "$2"
}

# ob_preflight HOST — what the same-host reference needs, before trying to build it.
#
# The native build failing with a linker error and thirty lines of go tooling is a
# worse gate line than "this host has no libopenblas.so", and the two are told apart
# here rather than by reading a compiler diagnostic. `go` is looked up as ssh finds
# it: a non-interactive, non-login shell, which is exactly how the build below runs,
# so a toolchain installed only in an interactive PATH reports as missing — which is
# the truth about this gate's ability to use it.
ob_preflight() {
  ssh "${KEEL_SSH_OPTS[@]}" "$1" '
    distro=unknown
    [ -r /etc/os-release ] && distro=$(sed -n "s/^ID=//p" /etc/os-release | tr -d \")
    go=none
    command -v go >/dev/null 2>&1 && go=$(go version | cut -d" " -f3)
    lib=none
    for d in /usr/lib64 /usr/lib/x86_64-linux-gnu /usr/lib /usr/local/lib; do
      if [ -e "$d/libopenblas.so" ]; then lib="$d/libopenblas.so"; break; fi
    done
    printf "distro=%s go=%s lib=%s\n" "$distro" "$go" "$lib"
  ' 2>/dev/null
}

# ob_provision_help HOST DISTRO — the exact commands needed on HOST, named per
# distribution because "install OpenBLAS" is not a command anybody can run.
#
# Printed, never executed: provisioning needs sudo, the gate connects with
# BatchMode=yes, and installing software on Scott's machines is his call, not this
# script's. scripts/provision-openblas.sh is the thing he runs.
ob_provision_help() {
  case "$2" in
    ubuntu|debian|pop|linuxmint)
      info "  [$1] sudo apt-get install -y libopenblas-dev" ;;
    rhel|centos|rocky|almalinux|fedora)
      info "  [$1] sudo dnf install -y openblas-devel" ;;
    *)
      info "  [$1] install an OpenBLAS development package (unrecognized distro id '${2:-unknown}')" ;;
  esac
  info "  [$1] plus a go1.26.5+ toolchain from go.dev/dl, on PATH for a non-login ssh"
  info "  [$1] scripts/provision-openblas.sh does both, with sudo prompts the gate cannot answer"
}

# ob_coretype_sweep HOST — the reference's ceiling, measured (issue #31).
#
# Prints one line per candidate: "requested achieved GFLOP/s". A candidate that
# cannot run here reports "- -": forcing a coretype the silicon cannot execute kills
# the harness, and that is a valid answer about this host rather than an error in the
# gate. A candidate that silently falls back reports the family it actually got, so
# the achieved name — never the requested one — is what gets compared and pinned.
# A candidate whose harness ran but produced no matching row reports `norow`, which
# the caller fails on: that is a defect in this gate, not a property of the host.
#
# The rate is read ONLY from a benchmark result row whose name is exactly the one
# requested (issue #33). The first version took the maximum over every field followed
# by "GFLOP/s" on every line, which silently picked up
#   keel-bench-peak-formula: avx512: 368.9 GFLOP/s (5.76 GHz x 2 FMA ports x 16 lanes)
# — a theoretical peak, larger than any real rate, identical across candidates. All
# six candidates then tied on every host, the winner was whichever came first
# (`default`), and the sweep reported +0.0% against DYNAMIC_ARCH's own choice: the
# 6.7% finding this function exists to enforce, erased by its own parser. Requiring
# the row name closes the neighbouring hole too, where a filter runs more benchmarks
# than intended (issue #32) and a different benchmark's rate is in reach.
#
# The rate is the best of $KEEL_BENCH_COUNT readings at the same -benchtime as every
# other number here, so the winner is chosen under the ratified methodology and not
# by a quick probe. Best-of-N rather than a median because this picks a candidate
# rather than reporting a result; the number that enters the record is measured again
# below, under full methodology, with the winner pinned.
#
# KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, and the script arrives on
# stdin. $BFLAGS is expanded at call time, by which point the methodology flags are
# set — this function is defined before them and must not capture them early.
ob_coretype_sweep() {
  local host="$1"
  # shellcheck disable=SC2087  # unquoted EOF on purpose: the candidate list, the
  # remote dir, the benchmark name and $BFLAGS are this gate's own values and must
  # expand here. Everything the remote shell owns is escaped as \$.
  ssh "${KEEL_SCP_OPTS[@]}" "$host" 'bash -s' 2>/dev/null <<EOF
cd '$OPENBLAS_REMOTE_DIR' || exit 1
for ct in $OPENBLAS_CORETYPES; do
  if [ "\$ct" = default ]; then unset OPENBLAS_CORETYPE; else export OPENBLAS_CORETYPE="\$ct"; fi
  out="\$(env GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 ./bench-ob.test \
    -test.run=NONE -test.bench='$GATE_OPENBLAS' $(printf '%s ' "${BFLAGS[@]}") 2>&1)" || {
    printf '%s - -\n' "\$ct"
    continue
  }
  core="\$(printf '%s\n' "\$out" | sed -n 's/.*corename=\([^ ]*\).*/\1/p' | tail -1)"
  best="\$(printf '%s\n' "\$out" | awk -v want='$GATE_OPENBLAS' '
    \$1 !~ /^Benchmark/ { next }
    {
      n = \$1
      sub(/-[0-9]+\$/, "", n)
      sub(/^Benchmark/, "", n)
      if (n != want) next
      for (i = 2; i < NF; i++) if (\$(i + 1) == "GFLOP/s" && \$i + 0 > m) m = \$i + 0
    }
    END { if (m > 0) printf "%.2f", m }')"
  printf '%s %s %s\n' "\$ct" "\${core:--}" "\${best:-norow}"
done
EOF
}

echo "== gate-p3: packing + blocking -> full Sgemm =="
echo

# ------------------------------------------------------------------- builds
echo "-- builds --"
if GOEXPERIMENT=simd go build ./... 2>&1; then pass "make build (GOEXPERIMENT=simd)"; else fail "make build (GOEXPERIMENT=simd)"; fi
if go build ./... 2>&1; then pass "make stock (scalar path, no experiment)"; else fail "make stock (scalar path, no experiment)"; fi
if GOEXPERIMENT=simd go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd)"; else fail "go vet (GOEXPERIMENT=simd)"; fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 go vet ./... 2>&1; then pass "go vet (GOEXPERIMENT=simd, linux/amd64)"; else fail "go vet (GOEXPERIMENT=simd, linux/amd64)"; fi

LOG="$(mktemp)"
BINDIR="$(mktemp -d)"
BIN="$BINDIR/keel.test"
KERNBIN="$BINDIR/kern.test"
BENCHBIN="$BINDIR/bench.test"
BENCHLOG="$BINDIR/bench.log"
BENCHCSV="$BINDIR/bench.csv"
# Criterion 5b's cross-check run, kept beside the dispatched one rather than
# overwriting it: the comparison needs both rates at once.
ALTLOG="$BINDIR/bench-alt.log"
ALTCSV="$BINDIR/bench-alt.csv"
SWEEPLOG="$BINDIR/sweep-avx512.log"
AUDITKERN="$BINDIR/audit-kern.log"
AUDITPEAK="$BINDIR/audit-peak.log"
trap 'rm -rf "$LOG" "$BINDIR"' EXIT

# ------------------------------------------------------- Sgemm vs the oracle
echo
echo "-- Sgemm vs the float64 oracle: size sweep x transpose x alpha x beta --"
info "the local run exercises the scalar path only (darwin/arm64 has no archsimd);"
info "the sweep's extent is audited below from a host that ran it with avx512 live"

LOCAL_OK=0
GOEXPERIMENT=simd go test -count=1 ./... >"$LOG" 2>&1 || LOCAL_OK=$?
if [[ "$LOCAL_OK" -eq 0 ]]; then
  pass "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
else
  fail "[local $(go env GOHOSTOS)/$(go env GOHOSTARCH)] all tests pass"
  sed 's/^/        /' "$LOG" | tail -40
fi
# The scalar path must also pass on a stock toolchain: keel is a pure-Go library
# first and an experiment second.
STOCK_OK=0
go test -count=1 ./... >"$LOG" 2>&1 || STOCK_OK=$?
if [[ "$STOCK_OK" -eq 0 ]]; then
  pass "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
else
  fail "[local, stock toolchain] all tests pass without GOEXPERIMENT=simd"
  sed 's/^/        /' "$LOG" | tail -40
fi

HOSTS="$(remote_hosts)"

# ---- the measurement precondition, asserted rather than assumed (ruling with #31)
#
# DESIGN.md §5.4 rule 5 asks for the performance governor. Until this ruling the gate
# READ scaling_governor and used it only to label criterion 6's pass line, so a host
# on `powersave` still contributed numbers to the record. antares did exactly that:
# its first OpenBLAS reading was 245.0 GFLOP/s against a 296-297 GFLOP/s steady
# state — a ramping-core artifact, indistinguishable in a log from a slow machine,
# and precisely what rule 5 exists to exclude.
#
# Now every host must be on `performance` or the gate is red. This is the same move
# as reading `threads=` back out of the library instead of trusting the environment
# that was passed to it: a precondition that is not checked is a precondition that
# drifts, and it drifts silently in whichever direction the machine happens to be
# configured. Unreadable counts as unmet — an unverified precondition is not a met
# one, and "unknown" is the answer a missing cpufreq sysfs gives on a VM.
if [[ -n "$HOSTS" ]]; then
  while read -r host; do
    [[ -n "$host" ]] || continue
    hgov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    if [[ "$hgov" == performance ]]; then
      pass "[$host] cpufreq governor is performance (§5.4 rule 5)"
    elif [[ -z "$hgov" || "$hgov" == unknown ]]; then
      fail "[$host] scaling_governor is unreadable, so §5.4 rule 5 cannot be verified; an unchecked precondition is not a met one"
    else
      fail "[$host] cpufreq governor is '$hgov', not performance (§5.4 rule 5): a ramping core produces cold readings that enter the record as measurements"
      info "  [$host] sudo cpupower frequency-set -g performance"
      info "  [$host] or: echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor"
    fi
  done <<<"$HOSTS"
fi

AVX512_GREEN=""
SCALAR_FORCED=""
if [[ -z "$HOSTS" ]]; then
  fail "P3 needs an amd64 host to execute the AVX-512 Sgemm; none configured"
else
  if remote_build_test . "$BIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 test binary (root package: Sgemm vs oracle)"
  else
    fail "cross-compile of linux/amd64 test binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    prov="$(remote_probe "$host" || true)"
    if [[ -z "$prov" ]]; then
      fail "[$host] unreachable"
      continue
    fi
    info "[$host] $prov"
    OK=0
    remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || OK=$?
    if [[ "$OK" -eq 0 ]]; then
      pass "[$host] Sgemm sweep passes"
    else
      fail "[$host] Sgemm sweep passes"
      sed 's/^/        /' "$LOG" | tail -40
    fi
    backends="$(marker sgemm-backends-exercised "$LOG")"
    if [[ -z "$backends" ]]; then
      fail "[$host] no keel-sgemm-backends-exercised marker: coverage unknown is coverage unestablished"
    else
      info "[$host] backends exercised: $backends"
    fi
    if [[ "$OK" -eq 0 && " $backends " == *" avx512 "* ]]; then
      AVX512_GREEN="$host"
      cp "$LOG" "$SWEEPLOG"
    fi
    # The fallback, proved by taking it on a machine that has the alternative.
    FOK=0
    KEEL_REMOTE_ENV="KEEL_FORCE=scalar" remote_exec "$host" "$BIN" -test.v >"$LOG" 2>&1 || FOK=$?
    if [[ "$FOK" -eq 0 ]]; then
      pass "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      SCALAR_FORCED="$host"
    else
      fail "[$host] KEEL_FORCE=scalar: the sweep passes with dispatch overridden"
      sed 's/^/        /' "$LOG" | tail -20
    fi
  done <<<"$HOSTS"
  if [[ -n "$AVX512_GREEN" ]]; then
    pass "the sweep ran green with the avx512 Sgemm live (target: $AVX512_GREEN)"
  else
    fail "no target ran the Sgemm sweep green with the avx512 backend"
  fi
  [[ -n "$SCALAR_FORCED" ]] || fail "no host proved the scalar fallback under KEEL_FORCE=scalar"
fi

# --------------------------------------------------- what the sweep covered
echo
echo "-- sweep extent (criteria 1 and 2: coverage is enforced, not trusted) --"
if [[ ! -s "$SWEEPLOG" ]]; then
  fail "no avx512 sweep log to audit; the sweep's extent is unverified"
else
  cfg="$(marker sgemm-config "$SWEEPLOG")"
  if [[ -z "$cfg" ]]; then
    fail "no keel-sgemm-config marker (blocking params, edge strategy, beta variants, packing backend)"
  else
    info "config: $cfg"
    for k in mr nr kc mc nc edge beta-variants pack; do
      [[ -n "$(field "$k" "$cfg")" ]] || fail "keel-sgemm-config is missing $k="
    done
  fi

  sizes="$(marker sgemm-sizes-exercised "$SWEEPLOG")"
  if [[ -z "$sizes" ]]; then
    fail "no keel-sgemm-sizes-exercised marker"
  else
    MISSING=""
    for n in $SWEEP_SIZES; do
      [[ " $sizes " == *" $n "* ]] || MISSING="$MISSING $n"
    done
    if [[ -z "$MISSING" ]]; then
      pass "every size in DESIGN.md §4/P3's sweep ran (1..17, 63, 64, 65, 500, 1000, 2048)"
    else
      fail "sizes missing from the sweep:$MISSING"
    fi
  fi

  combos="$(marker sgemm-combos-exercised "$SWEEPLOG")"
  if [[ -z "$combos" ]]; then
    fail "no keel-sgemm-combos-exercised marker"
  else
    info "combinations: $combos"
    trans="$(field trans "$combos")"
    alphas="$(field alpha "$combos")"
    betas="$(field beta "$combos")"
    ncombo="$(field combos "$combos")"
    tmiss=""
    for t in $SWEEP_TRANS; do
      [[ ",$trans," == *",$t,"* ]] || tmiss="$tmiss $t"
    done
    if [[ -z "$tmiss" ]]; then
      pass "transpose lattice complete (NN NT TN TT)"
    else
      fail "transpose combinations missing:$tmiss"
    fi
    # 0 and 1 are the special-cased paths; a lattice of only those exercises
    # every shortcut and never the general multiply.
    for pair in "alpha:$alphas" "beta:$betas"; do
      nm="${pair%%:*}"; vals="${pair#*:}"
      if awk -v v="$vals" 'BEGIN {
            n = split(v, a, ",")
            for (i = 1; i <= n; i++) {
              if (a[i] + 0 == 0) zero = 1
              else if (a[i] + 0 == 1) one = 1
              else other = 1
            }
            exit !(zero && one && other)
          }'; then
        pass "$nm covers 0, 1 and a general value ($vals)"
      else
        fail "$nm = ${vals:-<none>} does not cover all of {0, 1, general}: only the special-cased paths would be tested"
      fi
    done
    if [[ -n "$ncombo" ]]; then
      expect="$(awk -v t="$trans" -v a="$alphas" -v b="$betas" \
        'BEGIN { printf "%d", split(t, x, ",") * split(a, y, ",") * split(b, z, ",") }')"
      if [[ "$ncombo" -eq "$expect" ]]; then
        pass "combination count matches the enumerated sets ($ncombo = $expect)"
      else
        fail "combination count $ncombo does not match the enumerated sets ($expect): the marker claims combinations it did not run"
      fi
    else
      fail "keel-sgemm-combos-exercised has no combos= count"
    fi
  fi

  # Per-size oracle verification mode (criterion 2).
  VMISS=""; VBAD=""
  for n in $SWEEP_SIZES; do
    line="$(marker_all sgemm-verify "$SWEEPLOG" | awk -v want="size=$n" '{ for (i=1;i<=NF;i++) if ($i == want) { print; exit } }')"
    if [[ -z "$line" ]]; then
      VMISS="$VMISS $n"
      continue
    fi
    mode="$(field mode "$line")"
    if [[ "$n" -le "$SWEEP_EXACT_MAX" ]]; then
      [[ "$mode" == "exact" ]] || VBAD="$VBAD ${n}:${mode:-none}(must be exact)"
      continue
    fi
    case "$mode" in
      exact) ;;
      sampled)
        s="$(field n "$line")"
        if [[ -z "$s" ]] || [[ "$s" -lt "$SWEEP_SAMPLE_MIN" ]]; then
          VBAD="$VBAD ${n}:sampled(${s:-0} < $SWEEP_SAMPLE_MIN)"
        fi
        [[ -n "$(field seed "$line")" ]] || VBAD="$VBAD ${n}:sampled(no seed= to replay with)" ;;
      *) VBAD="$VBAD ${n}:${mode:-none}" ;;
    esac
  done
  if [[ -z "$VMISS" && -z "$VBAD" ]]; then
    pass "every size declares its oracle verification mode: exact up to $SWEEP_EXACT_MAX, >= $SWEEP_SAMPLE_MIN seeded exact entries above it"
    marker_all sgemm-verify "$SWEEPLOG" | tail -6 | sed 's/^/        /'
  else
    [[ -n "$VMISS" ]] && fail "no keel-sgemm-verify line for size(s):$VMISS"
    [[ -n "$VBAD" ]] && fail "oracle verification too weak for size(s):$VBAD"
  fi

  # Not from DESIGN.md's list, and not optional either: a row stride wider than n
  # is how a caller passes a submatrix, a zero dimension is the empty product, and
  # the argument checks are what stands between a bad ld and an out-of-bounds
  # write. Adding checks a gate's phase implies is allowed; removing them is not.
  extras="$(marker sgemm-extra-exercised "$SWEEPLOG")"
  EMISS=""
  for e in ldpad zerodim argpanic; do
    [[ " $extras " == *" $e "* ]] || EMISS="$EMISS $e"
  done
  if [[ -z "$EMISS" ]]; then
    pass "edge coverage beyond the sweep: $extras"
  else
    fail "keel-sgemm-extra-exercised is missing:$EMISS"
  fi

  packm="$(marker pack-combos-exercised "$SWEEPLOG")"
  if [[ -n "$packm" ]]; then
    pass "packing differential-tested against its scalar reference ($packm)"
  else
    fail "no keel-pack-combos-exercised marker: packing is not differential-tested"
  fi
fi

# ------------------------------ P2's compile-time properties, carried forward
echo
echo "-- carried from P2 (criterion 4): the K-loop after packing and blocking --"
info "compile-time property, audited against the linux/amd64 object code the hosts run"
if GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit \
     -pkg "$KERN_PKG" -func "$KERN_FUNCS" -mode spill -ssa "$SSADIR" >"$AUDITKERN" 2>&1; then
  sed 's/^/        /' "$AUDITKERN"
  pass "0 accumulator spills in the steady-state K-loop (P2 property held)"
  pass "0 calls in the steady-state K-loop (P2 property held)"
  pass "0 surviving bounds checks in the steady-state K-loop (P2 property held)"
else
  sed 's/^/        /' "$AUDITKERN"
  fail "P3 broke a P2 kernel property; the audit above says which"
fi
if go run ./internal/spill/cmd/spill-audit \
     -pkg ./internal/vec -func "$PEAK_FUNCS" -mode nomemory >"$AUDITPEAK" 2>&1; then
  pass "every peak kernel's steady-state loop is still register-only (the denominator is still a ceiling)"
else
  sed 's/^/        /' "$AUDITPEAK"
  fail "a peak kernel's loop touches memory; the percent-of-peak denominator is not a ceiling"
fi
if GOEXPERIMENT=simd GOOS=linux GOARCH=amd64 \
     go build -gcflags='-d=ssa/check_bce' "$KERN_PKG" ./internal/kern ./internal/pack ./internal/block 2>"$LOG"; then
  BCE_N="$(grep -c 'Found Is\(Slice\)\?InBounds' "$LOG" || true)"
  info "check_bce: ${BCE_N:-0} bounds check(s) across vec+kern+pack+block, all outside the K-loop (provenance; the criterion is the loop-body audit above)"
  pass "check_bce output recorded as provenance"
else
  sed 's/^/        /' "$LOG" | tail -20
  fail "build with -d=ssa/check_bce failed"
fi

# ------------------------------------------ the throughput sentinel (P2, #19)
echo
echo "-- throughput sentinel (criteria 5 and 5b): P2's floor re-run on the dispatched shape --"
if RTLOG="$(bash scripts/roofline-test.sh 2>&1)"; then
  pass "roofline verdict controls ($(grep -c '^  ok ' <<<"$RTLOG") fixtures)"
else
  fail "roofline verdict controls"
  # shellcheck disable=SC2001  # prefixing every line; not a scalar substitution
  sed 's/^/        /' <<<"$RTLOG"
fi
IPF_2x32="$(audit_ipf_tile 2x32 "$AUDITKERN")"
IPF_4x32="$(audit_ipf_tile 4x32 "$AUDITKERN")"
IPF_PEAK="$(audit_ipf "$GATE_PEAK_FUNC" "$AUDITPEAK")"
if [[ -n "$IPF_2x32" && -n "$IPF_4x32" && -n "$IPF_PEAK" ]]; then
  info "audited insns/FMA: 2x32 $(printf '%.3f' "$IPF_2x32"), 4x32 $(printf '%.3f' "$IPF_4x32"), $GATE_PEAK_FUNC $(printf '%.3f' "$IPF_PEAK")"
else
  fail "could not read insns/FMA from the audits; the sentinel cannot classify its host"
fi
BFLAGS=()
while read -r f; do BFLAGS+=("$f"); done < <(bench_flags)

SENTINELS="$(sentinel_hosts)"
# Criterion 5 judges the sentinel hosts. Criterion 6b needs something else from the
# same measurement: a *classification* for every host whose Sgemm will be divided by
# an OpenBLAS number, because the amended denominator applies only where the host is
# issue-bound. So the loop runs on every host and only the pass/fail is restricted to
# $SENTINELS; each host's (class, pmax) is recorded under $BINDIR for the OpenBLAS
# section to read, rather than re-derived there from a second set of measurements.
#
# A host with no recorded verdict is treated as FMA-bound below, which is the strict
# direction: no roofline, no leniency, the unmodified 60%-of-OpenBLAS bar.
CLASSIFY="$(printf '%s\n%s\n' "$SENTINELS" "$HOSTS" | sed '/^[[:space:]]*$/d' | awk '!seen[$0]++')"
# Criterion 5b's registry drift check runs once, on the first host that reports the
# marker: the binary is the same everywhere, and three identical lines would say no
# more than one. Empty after the loop means it never ran, which fails.
DRIFT_CHECKED=""
if [[ -z "$SENTINELS" ]]; then
  fail "no sentinel host and no hosts at all; P2's floor cannot be re-checked"
else
  if [[ -z "${KEEL_SENTINEL_HOST:-}" && ! -r .keel-sentinel ]]; then
    info "no sentinel configured, so every host is one: $(tr '\n' ' ' <<<"$SENTINELS")"
  fi
  if [[ "$(tr '\n' ' ' <<<"$CLASSIFY")" != "$(tr '\n' ' ' <<<"$SENTINELS")" ]]; then
    info "classifying every host for criterion 6b (judged as sentinels: $(tr '\n' ' ' <<<"$SENTINELS"))"
  fi
  if ! remote_build_test ./bench "$KERNBIN" >"$LOG" 2>&1; then
    fail "cross-compile of the linux/amd64 bench binary (kernel benchmarks)"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    # Judged only if configured as a sentinel; classified either way.
    JUDGED=0
    [[ $'\n'"$SENTINELS"$'\n' == *$'\n'"$host"$'\n'* ]] && JUDGED=1
    if ! remote_exec "$host" "$KERNBIN" "${BFLAGS[@]}" -test.bench="$KERN_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      if [[ "$JUDGED" -eq 1 ]]; then
        fail "[$host] sentinel: kernel benchmark run failed"
        sed 's/^/        /' "$BENCHLOG" | tail -20
      else
        info "[$host] kernel benchmark run failed; unclassified, so criterion 6b treats it as FMA-bound (the strict reading)"
      fi
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"

    # ---- criterion 5b, part 1: the registry's recorded insns/FMA vs the object code
    # Same binary on every host, so this is checked once — but it is checked from a
    # marker a host actually produced, not from source, because the point is to
    # compare what the shipped library believes against what the audit counts.
    kaudit="$(marker bench-kern-audit "$BENCHLOG")"
    if [[ -z "$DRIFT_CHECKED" && -n "$kaudit" ]]; then
      DRIFT_CHECKED="$host"
      DRIFT_BAD=""
      for pair in $kaudit; do
        rtile="${pair%%/*}"; rval="${pair#*=}"
        raud="$(audit_ipf_tile "$rtile" "$AUDITKERN")"
        if [[ -z "$raud" ]]; then
          DRIFT_BAD="$DRIFT_BAD ${rtile}(recorded $rval, not audited by this gate)"
        elif ! awk -v a="$rval" -v b="$raud" 'BEGIN{exit !(a - b < 0.001 && b - a < 0.001)}'; then
          DRIFT_BAD="$DRIFT_BAD ${rtile}(recorded $rval, audited $(printf '%.3f' "$raud"))"
        fi
      done
      if [[ -z "$DRIFT_BAD" ]]; then
        pass "every shipped shape's recorded insns/FMA matches the audited object code ($kaudit)"
      else
        fail "the shape ranking reads stale instruction counts:$DRIFT_BAD — internal/kern's registry has drifted from the K-loop it describes"
      fi
    fi

    # The shape this host's library actually dispatches to, and the class it chose
    # under, from the very run being judged.
    hkern="$(marker bench-kern "$BENCHLOG")"; hkern="${hkern%% *}"
    htile="${hkern%%/*}"
    hclassline="$(marker bench-kern-class "$BENCHLOG")"
    hclass="${hclassline%% *}"

    ACT_LO=""; ACT_PT=""; ACT_ID=""; ACT_IPF=""
    BEST_LO=""; BEST_PT=""; BEST_ID=""; MIXES=""; RIVALS=""
    for kname in $GATE_KERNELS; do
      [[ -n "$(bench_stat "$kname" "$BENCHCSV" GFLOP/s)" ]] || continue
      klo="$(bench_ratio_lo "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      kpt="$(bench_ratio "$kname" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      if [[ -z "$klo" ]]; then
        info "[$host] ${kname##Kernel/}: no CI, not counted"
        continue
      fi
      kid="${kname##Kernel/}"
      kipf="$(audit_ipf_tile "${kid%%/*}" "$AUDITKERN")"
      [[ -n "$kipf" ]] && MIXES="$MIXES $kid|$kpt:$kipf"
      if [[ "${kid%%/*}" == "$htile" ]]; then
        ACT_LO="$klo"; ACT_PT="$kpt"; ACT_ID="$kid"; ACT_IPF="$kipf"
      else
        RIVALS="$RIVALS $kid|$klo|$kpt"
      fi
      if [[ -z "$BEST_LO" ]] || awk -v a="$klo" -v b="$BEST_LO" 'BEGIN{exit !(a > b)}'; then
        BEST_LO="$klo"; BEST_PT="$kpt"; BEST_ID="$kid"
      fi
    done
    [[ -n "$IPF_PEAK" ]] && MIXES="$MIXES $GATE_PEAK_FUNC|1.0:$IPF_PEAK"
    if [[ -z "$BEST_LO" ]]; then
      if [[ "$JUDGED" -eq 1 ]]; then
        fail "[$host] sentinel: no bounded percent-of-peak for any shipped shape"
      else
        info "[$host] no bounded percent-of-peak; unclassified, so criterion 6b treats it as FMA-bound"
      fi
      continue
    fi
    # ---- criterion 5b, part 2: the sentinel judges the shape that ships.
    # Not the fastest one on the shelf: with a per-host choice those are different
    # claims, and the one P3's numbers rest on is the dispatched shape.
    if [[ -z "$ACT_LO" ]]; then
      fail "[$host] Sgemm dispatches to ${hkern:-an unreported shape}, which this gate did not benchmark ($GATE_KERNELS): the shipped kernel would go unjudged"
      continue
    fi
    if [[ "$ACT_ID" != "$BEST_ID" ]]; then
      info "[$host] dispatched $ACT_ID; fastest measured here is $BEST_ID ($(awk -v r="$BEST_PT" 'BEGIN{printf "%.1f", r*100}')% of peak vs $(awk -v r="$ACT_PT" 'BEGIN{printf "%.1f", r*100}')%) — checked below"
    fi
    # The ceiling set is every mix except the shape under test; see gate-p2.sh
    # criterion 5b and scripts/roofline.sh (INDEPENDENCE).
    CEIL=""
    for mx in $MIXES; do
      [[ "${mx%%|*}" == "$ACT_ID" ]] && continue
      CEIL="$CEIL ${mx#*|}"
    done
    # shellcheck disable=SC2086  # CEIL is a deliberate list of f:I words
    read -r CLASS _CSPREAD _MSPREAD ROOF ATTAIN RESULT WHY <<<"$(
      throughput_verdict "$ACT_LO" "${ACT_IPF:-0}" \
        "$PEAK_FLOOR" "$ROOF_FLOOR" "$ISSUE_CONVERGE_MAX" \
        "$ISSUE_MIX_SPREAD_MIN" "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" $CEIL)"
    frac="$(awk -v r="$ACT_LO" 'BEGIN{printf "%.1f", r * 100}')"
    fracpt="$(awk -v r="$ACT_PT" 'BEGIN{printf "%.1f", r * 100}')"

    # ---- criterion 5b, part 3: the classification that chose the shape, checked
    # against the one this gate measured. The library fingerprints a feature bundle
    # because no microarchitecture is readable from pure Go (T14, #25); this is the
    # measurement that says whether the fingerprint was right on this machine.
    if [[ -z "$hclass" ]]; then
      fail "[$host] no keel-bench-kern-class marker: the shape was chosen by a classification this gate cannot check"
    elif [[ "$hclass" == "$CLASS" ]]; then
      pass "[$host] the library's ${hclass}-bound classification matches this gate's measured verdict — ${hclassline#* }"
    else
      fail "[$host] the library classified this host ${hclass}-bound and chose $ACT_ID; this gate measures it ${CLASS}-bound — ${hclassline#* }"
      info "  [$host] a shape chosen from the wrong class is issue #24 again: fix internal/kern.HostClass's fingerprint, not this check"
    fi

    # ---- criterion 5b, part 4: did the choice lose throughput, in this same run?
    # A rival counts as faster only if its lower bound clears the dispatched shape's
    # point estimate, so noise cannot fail the gate and a real 11-point gap cannot
    # hide in it.
    LOST=""
    for rv in $RIVALS; do
      rid="${rv%%|*}"; rest="${rv#*|}"; rlo2="${rest%%|*}"; rpt2="${rest#*|}"
      if awk -v a="$rlo2" -v b="$ACT_PT" 'BEGIN{exit !(a > b)}'; then
        LOST="$LOST $rid($(awk -v r="$rpt2" 'BEGIN{printf "%.1f", r*100}')% of peak, ${rid} lower bound $(awk -v r="$rlo2" 'BEGIN{printf "%.1f", r*100}')% > dispatched ${fracpt}%)"
      fi
    done
    if [[ -z "$RIVALS" ]]; then
      info "[$host] only one shape measured, so there is nothing to prefer over $ACT_ID"
    elif [[ -z "$LOST" ]]; then
      pass "[$host] no other shipped shape beats the dispatched $ACT_ID here (same invocation, net of CI)"
    else
      fail "[$host] dispatch selected $ACT_ID and left throughput on the table:$LOST"
    fi
    attpc="$(awk -v a="$ATTAIN" 'BEGIN{printf "%.1f", a * 100}')"
    roofpc="$(awk -v r="$ROOF" 'BEGIN{printf "%.1f", r * 100}')"
    # What this host's sentinel measurement tells the later sections, in one file so
    # neither of them re-derives it from a second set of measurements:
    #
    #   CLASS  the verdict, for criterion 6b's denominator
    #   pmax   = max_i f_i·I_i, recovered from the verdict's own two outputs so that
    #          criterion 6b's roofline is built from the same ceiling this one judged
    #          against
    #   ACT_PT the dispatched microkernel's percent of peak, and ACT_ID the shape it
    #          was measured on, for the retention line (#26) — that line divides the
    #          blocked Sgemm by this, so it must be the same shape Sgemm ships, which
    #          is exactly what criterion 5b part 2 established above
    printf '%s %s %s %s\n' "$CLASS" \
      "$(awk -v r="$ROOF" -v i="${ACT_IPF:-0}" 'BEGIN{printf "%.6f", r * i}')" \
      "$ACT_PT" "$ACT_ID" \
      >"$BINDIR/class-$host"
    if [[ "$JUDGED" -eq 0 ]]; then
      # The roofline is quoted only where it can be used. On an FMA-bound host the
      # classifier still computes one — and on antares it computes 39.3%, from a
      # ceiling its own best shape then exceeds — so printing it beside "fma-bound"
      # would read as leniency that this host does not get.
      if [[ "$CLASS" == issue ]]; then
        info "[$host] classified issue-bound (roofline ${roofpc}%) for criterion 6b; not a sentinel, so P2's floor is not judged here"
      else
        info "[$host] classified ${CLASS}-bound for criterion 6b, so no roofline applies and it faces the unmodified bar; not a sentinel, so P2's floor is not judged here"
      fi
      continue
    fi
    case "$CLASS/$RESULT" in
      issue/pass)
        pass "[$host] sentinel: dispatched $ACT_ID holds P2's floor — ${fracpt}% of peak (${frac}% net of CI) = ${attpc}% of its ${roofpc}% issue roofline (>= 90%)" ;;
      */pass)
        pass "[$host] sentinel: dispatched $ACT_ID holds P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (>= 55%)" ;;
      */refuse)
        fail "[$host] sentinel: dispatched $ACT_ID at $(printf '%.3f' "${ACT_IPF:-0}") insns/FMA is outside the shape guard — P3 fattened the K-loop (why=$WHY)" ;;
      *)
        fail "[$host] sentinel: dispatched $ACT_ID fell below P2's floor — ${fracpt}% of peak, ${frac}% net of CI, ${CLASS}-bound (why=$WHY)" ;;
    esac
  done <<<"$CLASSIFY"
  # No host produced the marker at all: the ranking's inputs are then unverified,
  # which is a failure to check rather than a check that passed.
  [[ -n "$DRIFT_CHECKED" ]] || fail "no host reported keel-bench-kern-audit, so the registry's recorded insns/FMA were never checked against the object code"
fi

# ----------------------------------------------- Sgemm at 2048^3 vs OpenBLAS
echo
echo "-- Sgemm at 2048^3: percent of measured peak, and >= 60% of single-thread OpenBLAS --"
info "-count=$KEEL_BENCH_COUNT -benchtime=$KEEL_BENCH_TIME; the bar counts as cleared only net of both confidence intervals"
info "each host is also run once more with KEEL_KERN_CLASS pinned to the other class (criterion 5b), which is where the extra Sgemm time goes"

if [[ -n "$HOSTS" ]]; then
  if remote_build_test ./bench "$BENCHBIN" >"$LOG" 2>&1; then
    pass "cross-compiled linux/amd64 bench binary (Sgemm + peak)"
  else
    fail "cross-compile of linux/amd64 bench binary"
    sed 's/^/        /' "$LOG" | tail -20
  fi
  while read -r host; do
    [[ -n "$host" ]] || continue
    if ! KEEL_REMOTE_ENV="GOMAXPROCS=1" remote_exec "$host" "$BENCHBIN" "${BFLAGS[@]}" \
         -test.bench="$SGEMM_BENCH_FILTER" >"$BENCHLOG" 2>&1; then
      fail "[$host] Sgemm benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -20
      continue
    fi
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    if [[ -z "$(bench_stat "$GATE_SGEMM" "$BENCHCSV" GFLOP/s)" ]]; then
      fail "[$host] no $GATE_SGEMM benchmark result"
      continue
    fi
    info "[$host] Sgemm 2048^3 $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s), peak $(bench_describe "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pk="$(bench_ratio "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    pklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
    if [[ -n "$pk" ]]; then
      info "[$host] = $(awk -v r="$pk" 'BEGIN{printf "%.1f", r*100}')% of measured peak ($(awk -v r="${pklo:-0}" 'BEGIN{printf "%.1f", r*100}')% net of CI) — reported, not a P3 criterion"
    fi

    # ---- retention: how much of its own microkernel the blocked loop nest keeps.
    # Reported, never judged — the same standing as percent-of-peak, and for the same
    # reason: P3's criterion is the ratio against OpenBLAS, and blocking-parameter
    # work is P5's by DESIGN.md §4. It is printed because #26 is a named P5 input and
    # an input needs a number: janus keeps ~77% of its microkernel where the Zen
    # hosts keep 90-92%, and P5 inherits that gap as a measurement rather than as a
    # recollection of one. A run that stops printing it is a run that quietly dropped
    # P5's baseline, so the line is absent only when a measurement is missing.
    #
    # THIS IS A RATIO OF TWO POINT ESTIMATES FROM TWO INVOCATIONS, and it is not
    # bounded net of CI, because the two percentages come from different CSVs with a
    # peak measurement each — bench_ratio_lo cannot reach across them, and inventing
    # a bound here would be the statistics-free denominator §7 rule 7 forbids. Both
    # inputs are printed beside it so the division is reconstructible and so nobody
    # is tempted to compare the quotient against anything.
    if [[ -r "$BINDIR/class-$host" ]]; then
      read -r _ _ kpct kshape <"$BINDIR/class-$host"
      if [[ -n "$pk" && -n "$kpct" && -n "$kshape" ]] &&
         awk -v k="$kpct" 'BEGIN{exit !(k > 0)}'; then
        info "[$host] retention: the blocked loop nest keeps $(awk -v s="$pk" -v k="$kpct" 'BEGIN{printf "%.0f", s / k * 100}')% of its own $kshape microkernel ($(awk -v r="$pk" 'BEGIN{printf "%.1f", r*100}')% of peak blocked vs $(awk -v r="$kpct" 'BEGIN{printf "%.1f", r*100}')% unblocked; point estimates from two invocations) — reported, never judged; P5 baseline for #26"
      else
        info "[$host] retention not computable: no bounded microkernel percent-of-peak recorded for this host, so #26's P5 baseline is missing this run"
      fi
    fi

    # ---- criterion 5b, part 5: the shape choice, seen through packing and blocking
    # The kernel comparison in the sentinel section is the stronger evidence — both
    # shapes in one invocation. This is the same question asked of the number P3's
    # criterion actually rests on, which no single invocation can answer: the shape
    # is fixed at init, so measuring the other one means running the binary again
    # with KEEL_KERN_CLASS pinned to the class dispatch did not use.
    #
    # Two invocations, so the comparison is one-sided and CI-based: the passed-over
    # shape has to beat the dispatched shape's point estimate from below its own
    # interval before this fails. That makes it insensitive to the frequency drift
    # between two adjacent runs, at the cost of not seeing a small regression — the
    # same-invocation check above is what sees those.
    hclass="$(marker bench-kern-class "$BENCHLOG")"; hclass="${hclass%% *}"
    hkern="$(marker bench-kern "$BENCHLOG")"; hkern="${hkern%% *}"
    alt="$(other_class "$hclass")"
    if [[ -z "$alt" ]]; then
      fail "[$host] the bench run reports kernel class '${hclass:-<none>}', which is not a class this gate knows, so the shape choice cannot be cross-checked at 2048^3"
    elif ! KEEL_REMOTE_ENV="GOMAXPROCS=1 KEEL_KERN_CLASS=$alt" remote_exec "$host" "$BENCHBIN" \
           "${BFLAGS[@]}" -test.bench="$SGEMM_SHAPE_FILTER" >"$ALTLOG" 2>&1; then
      fail "[$host] the KEEL_KERN_CLASS=$alt Sgemm run failed, so the shape choice is uncorroborated at 2048^3"
      sed 's/^/        /' "$ALTLOG" | tail -20
    else
      altkern="$(marker bench-kern "$ALTLOG")"; altkern="${altkern%% *}"
      bench_csv "$ALTLOG" >"$ALTCSV" 2>"$LOG" || true
      [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
      altlo="$(bench_gflops_lo "$GATE_SGEMM" "$ALTCSV")"
      altpt="$(bench_gflops "$GATE_SGEMM" "$ALTCSV")"
      disppt="$(bench_gflops "$GATE_SGEMM" "$BENCHCSV")"
      if [[ -z "$altkern" || "$altkern" == "$hkern" ]]; then
        info "[$host] KEEL_KERN_CLASS=$alt selects ${altkern:-the same shape} too, so both classes agree here and there is nothing to compare"
      elif [[ -z "$altlo" || -z "$disppt" ]]; then
        fail "[$host] the KEEL_KERN_CLASS=$alt run established no bounded Sgemm rate, so the shape choice is unmeasured at 2048^3 rather than confirmed"
      elif awk -v a="$altlo" -v b="$disppt" 'BEGIN{exit !(a > b)}'; then
        fail "[$host] at 2048^3 the passed-over $altkern beats the dispatched $hkern: $(printf '%.1f' "$altpt") GFLOP/s, $(printf '%.1f' "$altlo") net of CI, against $(printf '%.1f' "$disppt")"
      else
        pass "[$host] the dispatched $hkern is no slower than $altkern at 2048^3 ($(printf '%.1f' "$disppt") vs $(printf '%.1f' "$altpt") GFLOP/s, $(printf '%.1f' "$altlo") net of CI; separate invocations)"
      fi
    fi
  done <<<"$HOSTS"
fi

# The reference: same host, same invocation, built natively behind the cgo tag —
# on EVERY gate host (ruling on #23). There is no reference-host list any more; the
# only apples-to-apples ratio is same silicon, same thread count, same run, so a
# host that cannot produce its own reference cannot contribute to this criterion,
# and it says so per host instead of one host standing in for three.
OB_CLEARED=0
OB_MEASURED=0
NHOSTS="$(sed '/^[[:space:]]*$/d' <<<"$HOSTS" | grep -c . || true)"
if [[ -z "$HOSTS" ]]; then
  fail "no execution hosts, so the >= 60%-of-OpenBLAS criterion cannot be evaluated (percent-of-peak is NOT a substitute)"
elif [[ -n "$(git status --porcelain)" ]]; then
  fail "the working tree is dirty, so \`git archive HEAD\` would measure something other than what is here; commit first"
else
  while read -r host; do
    [[ -n "$host" ]] || continue
    gov="$(remote_probe "$host" | sed -n 's/.*governor=\([^ |]*\).*/\1/p')"
    pre="$(ob_preflight "$host")"
    obdistro="$(field distro "$pre")"
    obgo="$(field go "$pre")"
    oblib="$(field lib "$pre")"
    info "[$host] governor=${gov:-unknown} distro=${obdistro:-unknown} go=${obgo:-none} libopenblas=${oblib:-none}"
    # Re-read, and re-checked, because the preamble's assertion has to hold at the
    # moment of measurement and not merely at the start of the gate. A governor that
    # changed in between belongs to a machine somebody started using, and the reading
    # it produces is not one §5.4 rule 5 covers. This replaces the old
    # "at least one host cleared the bar under the performance governor" tally, which
    # was satisfied by any single host and therefore said nothing about this one.
    if [[ "$gov" != performance ]]; then
      fail "[$host] governor is '${gov:-unknown}' at measurement time, not performance: it changed after this gate's preamble checked it, so nothing measured here is covered by §5.4 rule 5"
      continue
    fi
    if [[ "$obgo" == none || -z "$obgo" || "$oblib" == none || -z "$oblib" ]]; then
      MISS=""
      [[ "$obgo"  == none || -z "$obgo"  ]] && MISS="a Go toolchain"
      [[ "$oblib" == none || -z "$oblib" ]] && MISS="${MISS:+$MISS and }libopenblas.so"
      fail "[$host] no same-host OpenBLAS reference: this host is missing $MISS, so its ratio is unmeasured (percent-of-peak is NOT a substitute)"
      ob_provision_help "$host" "$obdistro"
      continue
    fi
    info "[$host] building the openblas-tagged harness natively from git archive HEAD ($(git rev-parse --short HEAD))"
    # KEEL_SCP_OPTS, not KEEL_SSH_OPTS: the latter carries -n, which would close
    # stdin and hand tar an empty archive. The remote-side paths below expand
    # here, on the client, which is what is wanted — they are this script's
    # variables, not the remote shell's.
    # shellcheck disable=SC2029
    if ! git archive --format=tar HEAD | ssh "${KEEL_SCP_OPTS[@]}" "$host" \
         "rm -rf '$OPENBLAS_REMOTE_DIR' && mkdir -p '$OPENBLAS_REMOTE_DIR' && tar -x -C '$OPENBLAS_REMOTE_DIR'" >"$LOG" 2>&1; then
      fail "[$host] could not ship the source tree for a native build"
      sed 's/^/        /' "$LOG" | tail -20
      continue
    fi
    # shellcheck disable=SC2029  # client-side expansion of a client-side path
    if ! ssh "${KEEL_SSH_OPTS[@]}" "$host" \
         "cd '$OPENBLAS_REMOTE_DIR' && GOEXPERIMENT=simd CGO_ENABLED=1 go test -c -tags openblas -o bench-ob.test ./bench" >"$LOG" 2>&1; then
      fail "[$host] native build of the openblas-tagged bench harness failed"
      sed 's/^/        /' "$LOG" | tail -30
      continue
    fi
    # ---- the reference's ceiling, established before the run that counts (#31).
    # Every candidate is recorded, not just the winner: a provenance block that names
    # only the family used cannot be checked against the one that was rejected, and
    # the margin over DYNAMIC_ARCH's own choice is the part a reader needs in order
    # to know whether upstream's selector was wrong here.
    SWEEP="$(ob_coretype_sweep "$host")"
    if [[ -z "$SWEEP" ]]; then
      fail "[$host] the coretype sweep produced nothing, so the reference's ceiling is unmeasured and the denominator would be whatever DYNAMIC_ARCH happened to pick"
      continue
    fi
    info "[$host] coretype sweep, best of $KEEL_BENCH_COUNT at -benchtime=$KEEL_BENCH_TIME:"
    SWEEP_NOROW=0
    while read -r cq cach cgf; do
      [[ -n "$cq" ]] || continue
      if [[ "$cgf" == "-" ]]; then
        info "  [$host] OPENBLAS_CORETYPE=$cq: unavailable on this host"
      elif [[ "$cgf" == norow ]]; then
        SWEEP_NOROW=1
        bad_row="$cq"
      else
        info "  [$host] OPENBLAS_CORETYPE=$cq -> corename=$cach, $cgf GFLOP/s"
      fi
    done <<<"$SWEEP"
    # A harness that ran and produced no $GATE_OPENBLAS row is this gate misreading
    # its own output or misnaming its own benchmark, and #33 is what that looks like
    # when it is allowed to degrade into a number instead of a failure.
    if [[ "$SWEEP_NOROW" -eq 1 ]]; then
      fail "[$host] the sweep ran the harness for OPENBLAS_CORETYPE=$bad_row and it produced no $GATE_OPENBLAS result row: that is a defect in this gate's filter or parser, not a property of the host"
      continue
    fi
    read -r OBCT OBCT_CORE OBCT_RATE <<<"$(awk '$3 != "-" && $3 + 0 > m { m = $3 + 0; best = $0 } END { print best }' <<<"$SWEEP")"
    if [[ -z "${OBCT:-}" || "${OBCT_RATE:--}" == "-" ]]; then
      fail "[$host] no candidate coretype produced a rate, so the reference cannot be pinned to its best family and its ceiling is unmeasured"
      continue
    fi
    OBDEF_RATE="$(awk '$1 == "default" && $3 != "-" { print $3 }' <<<"$SWEEP")"
    if [[ -n "$OBDEF_RATE" ]]; then
      info "[$host] reference pinned to OPENBLAS_CORETYPE=$OBCT (corename=$OBCT_CORE, $OBCT_RATE GFLOP/s), $(awk -v w="$OBCT_RATE" -v d="$OBDEF_RATE" 'BEGIN{printf "%+.1f%%", (w / d - 1) * 100}') against DYNAMIC_ARCH's own choice ($OBDEF_RATE GFLOP/s)"
    else
      info "[$host] reference pinned to OPENBLAS_CORETYPE=$OBCT (corename=$OBCT_CORE, $OBCT_RATE GFLOP/s); the default selection produced no rate to compare it against"
    fi
    OBARGS=""
    for a in "${BFLAGS[@]}" "-test.bench=$SGEMM_BENCH_FILTER"; do OBARGS+=" $(printf '%q' "$a")"; done
    OBENV=(GOMAXPROCS=1 OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1)
    # Pinned only when the sweep chose something other than what the library picks
    # unaided, so the common case runs exactly the command it always did.
    [[ "$OBCT" != default ]] && OBENV+=("OPENBLAS_CORETYPE=$OBCT")
    # shellcheck disable=SC2029  # client-side expansion of a client-side path
    if ! ssh "${KEEL_SSH_OPTS[@]}" "$host" \
         "cd '$OPENBLAS_REMOTE_DIR' && env ${OBENV[*]} ./bench-ob.test$OBARGS" >"$BENCHLOG" 2>&1; then
      fail "[$host] the openblas-tagged benchmark run failed"
      sed 's/^/        /' "$BENCHLOG" | tail -30
      continue
    fi
    obm="$(marker bench-openblas "$BENCHLOG")"
    gmp="$(marker bench-gomaxprocs "$BENCHLOG")"
    info "[$host] openblas: ${obm:-<no marker>} | gomaxprocs: ${gmp:-<no marker>}"
    if [[ -z "$obm" || "$obm" == *"not available"* ]]; then
      fail "[$host] the harness reports no OpenBLAS reference despite the openblas build tag"
      continue
    fi
    if [[ "$(field threads "$obm")" != "1" ]]; then
      fail "[$host] OpenBLAS reports ${obm:+$(field threads "$obm")} thread(s); the criterion is single-thread on both sides, so this is not the comparison DESIGN.md asks for"
      continue
    fi
    if [[ "$gmp" != "1" ]]; then
      fail "[$host] GOMAXPROCS=${gmp:-unreported}; keel's side of the comparison must be single-threaded too"
      continue
    fi
    # The kernel family DYNAMIC_ARCH selected. Checked because it is the one part of
    # the reference's configuration whose failure mode is a *low* denominator, and a
    # low denominator flatters keel — see $OPENBLAS_OK_CORES above.
    obcore="$(field corename "$obm" | tr '[:upper:]' '[:lower:]')"
    if [[ -z "$obcore" ]]; then
      fail "[$host] the OpenBLAS marker carries no corename=, so the reference's kernel family is unknown and a generic kernel cannot be ruled out"
      continue
    fi
    CORE_OK=0
    for c in $OPENBLAS_OK_CORES; do [[ "$obcore" == "$c" ]] && CORE_OK=1; done
    if [[ "$CORE_OK" -eq 0 ]]; then
      fail "[$host] OpenBLAS selected the '$obcore' kernel, which is not on the AVX2-or-better allowlist ($OPENBLAS_OK_CORES); a reference slower than the host can be is a denominator that flatters keel"
      info "  [$host] if '$obcore' is a legitimate AVX2-or-better target, add it to OPENBLAS_OK_CORES in this script; if it is a generic build, the distro package needs replacing"
      continue
    fi
    # The pin has to be verified, not trusted: OPENBLAS_CORETYPE is honoured by the
    # library, not by us, and a pin that silently did not take would divide keel by a
    # slower reference than the one this gate chose and reported — the flattering
    # direction, arrived at through a line of output claiming otherwise.
    if [[ "$obcore" != "$(tr '[:upper:]' '[:lower:]' <<<"$OBCT_CORE")" ]]; then
      fail "[$host] the coretype pin did not take: the sweep chose corename=$OBCT_CORE (OPENBLAS_CORETYPE=$OBCT) but the measured run reports corename=$obcore, so the number keel is about to be divided by is not the reference that was selected"
      continue
    fi
    info "[$host] reference kernel family: $obcore (on the AVX2-or-better allowlist, and the sweep's winner as pinned)"
    bench_csv "$BENCHLOG" >"$BENCHCSV" 2>"$LOG" || true
    [[ -s "$LOG" ]] && sed 's/^/        benchstat: /' "$LOG"
    if [[ -z "$(bench_stat "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)" ]]; then
      # Two very different states used to share this one sentence, and only one of
      # them is a missing reference (#32). The library is present and reported itself
      # a few lines above, so if the row is absent the run did not produce it — a
      # gate defect. Say which, and print the names that did come back, because
      # "the filter ran something else" is what that looks like from here.
      fail "[$host] the benchmark run produced no $GATE_OPENBLAS result row, although this host's OpenBLAS built, ran and reported itself above: the reference is present, so this is a defect in what the gate asked to be run rather than a missing reference"
      info "  [$host] -test.bench=$SGEMM_BENCH_FILTER returned: $(awk -F, '/^Benchmark/ { n = $1; sub(/-[0-9]+$/, "", n); print n }' "$BENCHCSV" | sort -u | tr '\n' ' ')"
      continue
    fi
    info "[$host] keel $(bench_describe "$GATE_SGEMM" "$BENCHCSV" GFLOP/s) vs OpenBLAS $(bench_describe "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s), one invocation"
    rlo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    rpt="$(bench_ratio "$GATE_SGEMM" "$GATE_OPENBLAS" "$BENCHCSV" GFLOP/s)"
    if [[ -z "$rlo" ]]; then
      fail "[$host] no bounded keel/OpenBLAS ratio: benchstat established no confidence interval, which is a failure to measure rather than a pass"
      continue
    fi
    rlopc="$(awk -v r="$rlo" 'BEGIN{printf "%.1f", r*100}')"
    rptpc="$(awk -v r="$rpt" 'BEGIN{printf "%.1f", r*100}')"
    OB_MEASURED=$((OB_MEASURED + 1))

    # ---- criterion 6b: which number this host's Sgemm is divided by (#23, #17/#18)
    # Every input is a measurement already taken, and all of them from THIS run on
    # THIS host: the reference rate, the peak rate, the shape Sgemm dispatched to,
    # and the classification recorded by the sentinel loop above.
    obclass="fma"; obpmax="0"
    if [[ -r "$BINDIR/class-$host" ]]; then read -r obclass obpmax _ _ <"$BINDIR/class-$host"; fi
    # "4x32/avx512 (available: ...)" -> Kernel4x32; audited, not assumed, because a
    # roofline built from the wrong shape's instruction count is the hole the shape
    # guard exists to close.
    obkern="$(marker bench-kern "$BENCHLOG")"; obkern="${obkern%% *}"
    obtile="${obkern%%/*}"
    i_active="$(audit_ipf_tile "$obtile" "$AUDITKERN")"
    ob_rate="$(bench_gflops "$GATE_OPENBLAS" "$BENCHCSV")"
    peak_rate="$(bench_gflops "$GATE_PEAK" "$BENCHCSV")"
    read -r obdenom obsrc obroof obwhy <<<"$(p3_denominator \
      "$obclass" "$obpmax" "${i_active:-0}" \
      "$SWEEP_BEST_IPF" "$ROOF_SHAPE_SLACK" "${ob_rate:-0}" "${peak_rate:-0}")"
    info "[$host] denominator: $obsrc $(printf '%.2f' "$obdenom") GFLOP/s (why=$obwhy, class=$obclass, Sgemm ran ${obkern:-unknown} at $(printf '%.3f' "${i_active:-0}") insns/FMA, roofline $(awk -v r="$obroof" 'BEGIN{printf "%.1f", r*100}')% of a $(printf '%.2f' "${peak_rate:-0}") GFLOP/s peak)"

    # Net of CI, in the same conservative direction as everything else here. Against
    # the roofline cap that is keel_lo/(roof · peak_hi) = pklo/roof; $rlo remains a
    # valid (weaker) bound on the same ratio because the cap is below OpenBLAS, so
    # the tighter of the two is taken rather than whichever came to hand.
    alo="$rlo"; apt="$rpt"
    if [[ "$obsrc" == roofline ]]; then
      pklo="$(bench_ratio_lo "$GATE_SGEMM" "$GATE_PEAK" "$BENCHCSV" GFLOP/s)"
      alo="$(p3_ratio_lo roofline "$rlo" "$pklo" "$obroof")"
      if [[ -z "$alo" ]]; then
        fail "[$host] no bounded Sgemm/peak ratio, so the amended denominator cannot be bounded either; that is a failure to measure, not a pass"
        continue
      fi
      apt="$(awk -v k="$(bench_gflops "$GATE_SGEMM" "$BENCHCSV")" -v d="$obdenom" 'BEGIN{printf "%.4f", k / d}')"
    fi
    alopc="$(awk -v r="$alo" 'BEGIN{printf "%.1f", r*100}')"
    aptpc="$(awk -v r="$apt" 'BEGIN{printf "%.1f", r*100}')"
    # Both ratios, always, whichever one binds (§7 rule 7 applies to this script's
    # arithmetic as much as to the library's benchmarks).
    if [[ "$obsrc" == roofline ]]; then
      info "[$host] amended (issue-capped): ${aptpc}%, ${alopc}% net of CI | plain OpenBLAS: ${rptpc}%, ${rlopc}% net of CI"
    else
      info "[$host] plain OpenBLAS is the denominator (why=$obwhy), so the amended and unmodified ratios are the same number"
    fi
    if awk -v r="$alo" -v f="$OPENBLAS_FLOOR" 'BEGIN{exit !(r >= f)}'; then
      pass "[$host] Sgemm at 2048^3 is ${aptpc}% of its $obsrc denominator, ${alopc}% net of CI (>= 60%; plain OpenBLAS ${rptpc}%, ${rlopc}% net of CI)"
      OB_CLEARED=$((OB_CLEARED + 1))
    else
      fail "[$host] Sgemm at 2048^3 is only ${aptpc}% of its $obsrc denominator, ${alopc}% net of CI (< 60%; plain OpenBLAS ${rptpc}%, ${rlopc}% net of CI)"
    fi
  done <<<"$HOSTS"
  if [[ "$OB_MEASURED" -eq 0 ]]; then
    fail "no host produced a keel/OpenBLAS ratio at all, so criterion 6 is unmeasured rather than missed"
  elif [[ "$OB_CLEARED" -eq "$NHOSTS" ]]; then
    pass "every gate host cleared 60% of its own single-thread OpenBLAS ($OB_CLEARED/$NHOSTS)"
  else
    fail "$OB_CLEARED of $NHOSTS gate hosts cleared the bar; ruling #23 asks every host to clear its own reference"
  fi
fi

# ------------------------------------------------------------------ verdict
echo
if [[ "$FAIL" -eq 0 ]]; then
  echo "gate-p3: GREEN"
  exit 0
fi
echo "gate-p3: RED" >&2
exit 1
