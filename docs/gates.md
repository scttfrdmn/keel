<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The reasoning behind the phase gates

`scripts/gate-p2.sh`, `gate-p3.sh`, `gate-p4.sh` and `gate-p5.sh` each opened with a
front-matter essay: what each criterion measures, which judgement calls went into
mechanizing it, what the gate deliberately refuses to decide. Together those were **855
lines of comment above zero lines of code** — after the shebang, `gate-p4.sh` ran 226
comment lines and `gate-p5.sh` 240 before their first executable statement. They are
reproduced here **verbatim**, moved 2026-08-16, and each script keeps a pointer to its
section.

Three things this move deliberately does not do. It does not summarise: the text below is
the text that was there, `# ` prefixes stripped and nothing else, which is why it is
fenced rather than reflowed into markdown — a rewrap is an edit, and an edit to a ruling
is a new ruling. It does not renumber: every criterion keeps the number the gate prints.
And it does not move a threshold — those live in code, in the gate, in one place each, and
`gate-p3.sh` states why they are duplicated across gates rather than shared.

This page is not published to the documentation site. That is the same rule
[`docs/rulings.md`](rulings.md) followed in the other direction: relocating prose must not
change who can read it, and this prose was repo-only before the move. `DESIGN.md` §4 is
the authority for what a phase requires; a gate script is the authority for what is
actually checked; this file explains the distance between them and settles neither.

---

## P2 — the spill audit and the percent-of-peak floor

From `scripts/gate-p2.sh`, lines 2–146 at `c421486`.

```text
Gate P2 — see DESIGN.md §4/P2. Written at the START of phase P2, then made
green. Exits 0 only when every criterion for the phase holds. A red gate
blocks the next phase; there is no override flag on purpose.

THIS GATE IS A GO/NO-GO, NOT A HURDLE (CLAUDE.md). If it stays red after the
documented kernel-shaping steps and one tile shrink, the deliverable is
docs/spill-report.md and a blocked issue — not a weakened check and not a
switch to hand-written assembly.

Criteria (verbatim from DESIGN.md §4/P2):
  "`spill-audit` reports 0 accumulator spills in the steady-state K-loop,
   AND the raw microkernel (packed inputs, no blocking) hits >=55% of
   measured peak."

How those are mechanized, and every judgement call involved:

 1. CORRECTNESS FIRST. A fast wrong kernel is not a P2 pass. The microkernel
    is differential-tested against a scalar reference for the same tile, under
    internal/oracle.Tolerance, on every backend each machine can execute —
    locally (scalar only, stock toolchain) and on every amd64 host in
    .keel-hosts, by the same cross-compile-and-ship path as P0/P1.

 2. THE SPILL AUDIT IS A COMPILE-TIME PROPERTY, so it runs here on the dev
    host, against the linux/amd64 object code the remote hosts will execute.
    internal/spill parses `go build -gcflags=-S`, finds the steady-state
    K-loop (the innermost loop carrying the arithmetic), and reports every
    stack-relative reference inside it. "0 accumulator spills" is mechanized
    as: no instruction in that loop body has both a vector register operand
    and an (SP)-relative memory operand. Register-to-register copies are
    counted and printed but are NOT spills — they cost issue slots, not
    memory traffic, and DESIGN.md's criterion is about the latter. That
    distinction decides this phase's verdict, so it is stated here rather
    than left implied.

    It is audited on the SHIPPED shapes only. DESIGN.md's own 12-accumulator
    tile cannot be allocated on go1.26.5 (docs/toolchain-notes.md T10, issue
    #18) and is kept as kern.ReferenceTile — audited and benchmarked, never
    dispatched. Its audit runs below as evidence and is explicitly non-fatal,
    because a gate that failed on a shape nothing ships would be measuring the
    evidence instead of the product. Everything in kern.Kernels() is held to
    zero.

 3. THE OTHER SHAPING RULES ARE CHECKED ON THE LOOP BODY, NOT ON THE FILE.
    DESIGN.md §4/P2 lists pre-sliced panels (bounds-check elimination), no
    calls in the K-loop, and pointer-free data. All three are properties of the
    steady-state loop, so all three are checked by the audit tool that already
    identifies it: a call is a CALL in the body, and a surviving bounds check
    is a branch out of the body to a block that calls runtime.panic*.

    `-d=ssa/check_bce` is printed as provenance and is NOT the criterion. It
    reports every bounds check in the package, and a kernel legitimately has
    dozens outside the K-loop — `a[:kc*MR]` in the prologue and
    `c[i*ldc:i*ldc+NR]` in the write-out both report one, and both cost nothing
    amortized over K. Failing on those would make the criterion unsatisfiable
    for reasons unrelated to what P2 is asking, so the count is printed and the
    loop body is enforced.

 4. THE PEAK KERNEL'S NO-MEMORY PROPERTY IS CHECKED HERE (issue #11).
    internal/vec/peak.go rests on four properties; three are guarded by an
    exact arithmetic witness that runs on every host, and the fourth —
    nothing in the loop touches memory — cannot be seen by arithmetic. The
    same audit tool checks it in -mode=nomemory, so the denominator P2
    divides by is verified in the run that divides by it.

 5. PERCENT OF PEAK IS MEASURED AGAINST A MEASURED PEAK, IN ONE RUN. The
    numerator (the microkernel) and the denominator (BenchmarkPeak) are
    measured in the *same* benchmark invocation on each host, so they share a
    frequency and thermal state; a ratio of two numbers taken hours apart on
    one machine would be a worse measurement than either of them. Both come
    out of benchstat under the §5 rule 5 methodology (issue #14): -count=10
    -benchtime=1s, medians, and the 55% bar counts as cleared only net of
    both confidence intervals. Every host that can run the kernel must clear
    it, and at least one must do so on a host whose clock rule 5 counts as
    established stable — the performance governor here, that being what every
    host this gate has run against exposes.

    Note what this bar is not: it is not 55% of a formula, and it is not the
    best of N hosts. See docs/hosts.md and issue #15 for the one host whose
    memory-touching benchmarks are bimodal between runs.

 5b. THE BAR IS ROOFLINE-RELATIVE ON AN ISSUE-BOUND HOST (DESIGN.md §4/P2 as
    amended by the ruling on #19). The flat 55% has one denominator and so
    assumes the front end can deliver the kernel's instruction mix at 55% of
    FMA peak. On a host retiring two full-width FMAs per cycle from a ~4-wide
    front end it cannot, and the flat bar becomes a demand that the kernel beat
    the decode stage. Such a host is instead held to 90% of its issue roofline,
    computed from the spill audit's own instruction counts.

    Four things keep that from being a weaker gate rather than a differently
    expressed one. None is asserted here: the decision is a pure function in
    scripts/roofline.sh, and scripts/roofline-test.sh drives it with fixtures
    that attempt each abuse and asserts each is refused. Those controls run at
    the top of this gate, before any benchmarking.
      - Classification uses evidence independent of the fraction being gated.
        The naive test — measured rate < the rate 55% would require — reduces
        algebraically to f < 0.55, i.e. "the host failed", and would make the
        amendment self-granting. What is tested instead is whether an issue
        *ceiling* exists: do structurally different instruction mixes converge
        on one retirement rate? Only a machine property does that.
      - The ceiling is established by the mixes OTHER than the shape under test.
        With that shape included, attain reduces to p_best / max_i p_i >=
        1/cspread >= 1/1.10 = 0.909, which already clears the 0.90 floor: every
        host the classifier admits would pass by construction. The first draft
        of this gate had exactly that hole, and it was widest in the binding
        case — on janus the shape under test *is* the argmin of p_i, so
        1/cspread and attain agreed to four places (0.9476).
      - A ceiling the machine exceeds is not a ceiling. If the shape under test
        retires above the rate the ceiling mixes converged on, the issue-bound
        hypothesis is falsified by its own data and the host reverts to the flat
        floor. This is also what stops the mirror abuse of understating the
        ceiling, and it is what correctly returns antares (Zen 5) to FMA-bound.
      - The roofline branch requires the shape to be near the sweep's best
        insns/FMA, because a roofline computed from the instruction count of the
        kernel under test rises as that kernel gets worse.
    A merely-slow kernel produces divergent rates, classifies FMA-bound, and
    faces the full 55%.

    WHAT THIS BRANCH CANNOT SEE (DESIGN.md §5 rule 12; added 2026-09-01 on #141).
    Excluding the shape under test buys independence from that shape, not from the
    host. Every ceiling mix is measured on the same machine in the same
    invocation, so under a host-level effect the numerator and the denominator move
    together: a shape-UNIFORM excursion scales both of attain's terms by nearly the
    same factor and is ABSORBED — the criterion renders a clean verdict over it and
    reports nothing. Only a DIRECTION-SPLIT excursion reaches a verdict, and #141's
    third instance reached one by the accident of its shape. On janus between two
    gate runs the ceiling set stayed nearly still (6x32 +2.59% against 2x32
    -9.76%), so the roofline moved -1.4% (49.3% -> 48.6%) while the share moved
    -9.8% (47.9% -> 43.2%) and the excursion was reported. Had the same magnitude
    arrived uniformly across the shapes, this branch would have PASSed over it.
    So a green 5b is not a bound on host stability, and it cannot become one from
    the inside: an uncorrelated denominator would have to come from a different
    run, which is a different instrument. The one that measures it is a repeat on
    the suspect host recording ABSOLUTE rates -- scripts/ab-bench.sh drift -- since
    a ratio of two co-moving terms self-cancels the effect in question.

    WHAT THE AMENDMENT COSTS, AS A NUMBER. The peak kernel is always in the
    ceiling set with f = 1, so max_i p_i >= I_peak, and the shape guard caps the
    denominator at SWEEP_BEST_IPF * ROOF_SHAPE_SLACK. The effective floor for an
    issue-bound host is therefore never below
        ROOF_FLOOR * I_peak / (SWEEP_BEST_IPF * ROOF_SHAPE_SLACK)
        = 0.90 * 2.25 / 4.856 = 41.7% of measured peak.
    That is the whole of the slack granted: from 55% to no less than 41.7%, and
    only to a host that has independently demonstrated a front-end ceiling.
    Read 4.659 = 43.5% until 2026-08-18, when SWEEP_BEST_IPF was corrected from
    4.438 to 4.625 (#107). Only the cap moved: the floor an issue-bound host
    actually faces is 0.90 * max_i p_i / I_b from its own audited I_b, which for
    the shipped 2x32 is unchanged at 43.8%.

    THE AMENDMENT RATCHETS; IT DOES NOT RETIRE. An earlier draft claimed it was
    self-retiring — that when the lowering improves (T12/#20) the host would stop
    classifying issue-bound and the flat floor would resume. The arithmetic says
    otherwise, and better: with I falling 4.625 -> ~2.875, janus stays at its
    front-end ceiling but the roofline *rises* to 2.250/2.875 = 78.3%, so the
    required floor becomes 0.90 * 78.3% = 70.4% — stricter than the 55% the
    amendment replaced. The floor is 0.90 * max_i p_i / I_b and max_i p_i is
    pinned by the peak kernel, so it is monotone non-increasing in I: every
    improvement to the lowering tightens this gate automatically. A kernel at 60%
    of peak clears the flat 55% and fails the post-fix ratchet; that is a control
    in roofline-test.sh.

    It IS the best of the shipped shapes, per host. Two zero-spill tiles ship
    (2x32 and 4x32) because which one wins depends on the host's front-end
    width and load ports rather than on anything in the source, and P3 will
    dispatch to one of them. So the criterion is applied to the shape that
    wins on that host, with every shape's number printed either way. Requiring
    both to clear would fail a host for carrying a second kernel it would never
    select; requiring only their average would hide the winner.
```

---

## P3 — Sgemm against the oracle and against OpenBLAS

From `scripts/gate-p3.sh`, lines 2–245 at `c421486`.

```text
Gate P3 — see DESIGN.md §4/P3. Written at the START of phase P3, then made
green. Exits 0 only when every criterion for the phase holds. A red gate
blocks the next phase; there is no override flag on purpose.

Criteria (verbatim from DESIGN.md §4/P3):
  "full `Sgemm` matches oracle across a size sweep (1..17, 63,64,65, 500,
   1000, 2048, plus transpose/beta/alpha combinations); single-thread >=60%
   of OpenBLAS at 2048^3 (bench harness pulls OpenBLAS via build-tagged cgo,
   dev machine only, never a package dependency)."

How those are mechanized, and every judgement call involved:

 1. THE SWEEP'S EXTENT IS ENFORCED, NOT TRUSTED. "matches oracle across a size
    sweep" is a claim about coverage, and a green `go test` proves only that
    whatever ran, passed. A test that quietly skipped 2048 because it was slow,
    or that ran one transpose combination instead of four, reports the same
    green. So the tests print coverage markers and this gate parses them: every
    size in DESIGN.md's list must appear, the transpose lattice must be complete
    (NN/NT/TN/TT), and alpha and beta must each include 0, 1 and a value that is
    neither — 0 and 1 are the special-cased paths, so a lattice of only those
    would exercise every shortcut and never the general multiply. The enumerated
    sets must also multiply out to the reported combination count, so the marker
    cannot claim combinations it did not run.

 2. THE ORACLE'S COST IS A DECLARED PROPERTY OF EACH SIZE, NOT A SILENT
    DEGRADATION. A float64 oracle at 2048^3 is 8.6 GFLOP per combination, and
    the full lattice of that is hours. The sweep is therefore allowed to verify
    the large sizes by checking a seeded random *sample* of C entries, each one
    computed exactly in float64 — but only if it says so per size, in a marker
    this gate reads. Sizes up to 65 must be verified in full; 500, 1000 and 2048
    may be sampled, with a floor on the sample size and a printed seed so any
    failure is replayable. The distinction is written here rather than left to
    the test, because "we sampled" is exactly the kind of concession that starts
    at 2048 and ends up applying to 17.

 3. CORRECTNESS RUNS WHERE THE SHIPPED PATH RUNS. The dev host is darwin/arm64
    and cannot execute archsimd at all (docs/toolchain-notes.md T1), so a local
    `go test` exercises the scalar path and nothing else. The sweep's extent is
    therefore audited from the log of a host that ran it with the AVX-512
    backend live, and the scalar path is proved separately by the local run and
    by a KEEL_FORCE=scalar run on an amd64 host — the P1 mechanism, because the
    only way to show the fallback works on a machine that has AVX-512 is to make
    it take the fallback.

 4. P2's KERNEL PROPERTIES ARE RE-CHECKED HERE, BECAUSE P3 IS WHAT WOULD BREAK
    THEM. Packing, edge handling and beta variants all add code around the
    K-loop, and the failure mode is a loop that acquired a spill, a call or a
    bounds check on the way to becoming usable. Those are compile-time
    properties, so the audit is cheap and it runs on every gate from here on.
    This is a carried-forward criterion, not a new one, and it does not become
    optional because P3's own checks are green.

 5. THE THROUGHPUT SENTINEL. The ruling on issue #19 made P2's floor
    class-dependent: on an issue-bound host it is 0.90 x max_i(f_i.I_i) / I_b,
    which *rises* as the kernel's instruction count falls, and by the same
    arithmetic that host is the one that notices when a shape gets fatter. P3 is
    the phase most likely to fatten one. So P2's verdict is re-run here, through
    the same unit-tested pure function (scripts/roofline.sh) driven by the same
    fixtures, which run before any benchmarking.

    The sentinel set is EVERY host in the fleet — the same .keel-hosts (or
    $KEEL_REMOTE_HOSTS) the rest of the gate reads, so there is no second host list
    to go stale. Missing configuration must cost time, not coverage; a gate that
    skipped its regression check because a file was absent would be a gate that got
    weaker when someone cloned the repo.

    RESTATED 2026-08-31 (#146, ruled). This read "the sentinel is named by
    $KEEL_SENTINEL_HOST or .keel-sentinel", and that precedence put a gitignored,
    machine-local file ABOVE the fleet: P2's floor was judged on one lab host in the
    #113 re-measurement and in the run that signed v0.1.0-a2, with all three fleet
    hosts printing "not a sentinel". .keel-sentinel is no longer read (its presence
    is reported, with its mtime, as not read); $KEEL_SENTINEL_HOST FAILS rather than
    being silently ignored; an out-of-fleet sentinel is declared with
    $KEEL_SENTINEL_OUT_OF_FLEET, which unions with the fleet and prints into the
    run's .cmd and into the log's declaration row. docs/hosts.md carries the
    correction in full.

    IF THIS CRITERION READS RED, THE POLICY IS IN DESIGN.md §4 AND IT IS ONE
    RE-RUN. The issue-bound roofline check is the tightest margin in this gate --
    two independent runs on janus read 95.0% and 93.7% against a 90% floor -- so a
    single dip is within the variance the bar was set above. THE SAMPLE HAS GROWN
    SINCE, in both directions, and it is stated here as of 2026-09-01: a third janus
    reading fell BELOW the floor at 87.2%..89.2% (#113/#141, on generated code that
    did not change), and the first on-fleet reading, keel-skx, came in at
    96.9%..97.1%. Four readings, one of them a fail, is still the sample this policy
    was priced for; what the low one changed is the priority of #141, not the bar. Exactly one immediate
    re-run; the criterion fails only if both runs fail; both outputs go into the
    umbrella issue verbatim either way, so a pass on the second reading says so in
    the record. Not a loop, and not available to any correctness criterion above:
    a differential test that passes on retry has found a nondeterminism, which is
    worth more attention than the gate colour rather than less. This script does
    not implement the retry -- it fails on the first miss, and the allowance is the
    operator's, because an auditable human re-run beats a script trusted to
    confess that it retried.

 5b. THE SHAPE MEASURED IS THE SHAPE DISPATCH SELECTS (ruling on issue #24).
    P2 shipped two zero-spill shapes and neither dominates: 4x32 wins on Zen 4
    and Zen 5, 2x32 wins on Skylake-X by 11 percentage points (KERNEL.md §7).
    Dispatch now chooses per host, from the same issue-bound/FMA-bound
    classification this gate's own model defines — FMA-bound takes the fewest
    memory ops per FMA, issue-bound the fewest instructions per FMA. That fixed
    the performance bug criterion 6b's shape guard found (janus was shipping
    4x32), and it opens three ways for a gate to be lied to, so it closes all
    three:

      - EVALUATING ONE KERNEL WHILE SHIPPING ANOTHER. Criterion 5 used to judge
        P2's floor against whichever shipped shape measured fastest on the host.
        With a per-host choice that is no longer the same thing as the shape
        `Sgemm` runs, so the sentinel now judges the DISPATCHED shape, named by
        the keel-bench-kern marker of the run being judged. The other shapes
        become the ceiling set, which is what INDEPENDENCE in scripts/roofline.sh
        asks for anyway.

      - A WRONG CHOICE PASSING QUIETLY. Both shapes are benchmarked in one
        invocation, so "did dispatch pick the faster one" is answerable from the
        same measurement: if another shape's lower bound exceeds the dispatched
        shape's point estimate, this gate fails. That direction is the one the
        classification can be wrong in without any other criterion noticing —
        see internal/kern/class_amd64.go, which documents both error directions
        and why the other one only ever makes this gate stricter. The margin is
        CI-based rather than exact so the check cannot flake on noise.

        The blocked `Sgemm` is then re-measured under KEEL_KERN_CLASS pinned to
        the other class, and the dispatched shape must be no slower there either.
        That run is a second invocation — weaker evidence than the same-run
        kernel comparison above, and it is a cross-check, not the criterion —
        but it is the only way to see the choice through packing and blocking,
        which is where P3's number actually comes from.

      - A CLASSIFICATION NOBODY CHECKED. The library cannot read a
        microarchitecture: archsimd exposes CPU features and no vendor, family or
        model, so HostClass fingerprints the generation from a feature bundle
        (docs/toolchain-notes.md T14, issue #25). A proxy that decides what ships
        must be checked against something measured, so the library prints its
        classification and its grounds (keel-bench-kern-class) and this gate
        compares them against the class its own convergence test derives from the
        measurements. Disagreement is a gate failure, on every host, every run.

    And because the ranking reads an audited instruction count recorded in Go
    source, that number is re-derived from the object code here: every shipped
    shape's Kernel.InsnsPerFMA must equal what the spill audit counts in its loop
    body. A measurement in source that nothing recomputes is a measurement that
    drifts, and this one decides what ships.

 6. THE OPENBLAS BAR, AND WHICH READING OF "DEV MACHINE ONLY" THIS IMPLEMENTS.
    DESIGN.md says the OpenBLAS harness is cgo behind a build tag, "dev machine
    only, never a package dependency". Read as "measure on the dev machine" the
    criterion is vacuous here: this dev machine is darwin/arm64, where keel's
    shipped AVX-512 path does not exist, so the ratio would compare
    OpenBLAS-on-arm64 against keel's scalar fallback and answer a question
    nobody asked. DESIGN.md §7 rule 7 cuts the same way — a ratio whose two
    halves come from different silicon is not a ratio.

    So this gate implements the other reading: the *comparison* is dev-only —
    built behind the `openblas` tag, absent from the module's dependency graph,
    never linked into anything keel ships — and it runs on the amd64 hosts where
    both halves can execute. Per the ruling on issue #23 that is EVERY gate host,
    each against its own OpenBLAS: no cross-host reference and no golden machine,
    because the only apples-to-apples ratio is same silicon, same thread count,
    same run. Concretely:
      - the reference and keel's Sgemm are measured in the SAME benchmark
        invocation on the SAME host, so they share a frequency and a thermal
        state. P2's criterion 5 settled this: a ratio of two numbers taken from
        separate runs is a worse measurement than either of them.
      - the harness is built natively on that host from `git archive HEAD`, so
        the number is attributable to a commit; the working tree must be clean
        or the archive does not mean what it says.
      - single-thread is enforced on both sides and *verified* from the
        harness's own report (OpenBLAS's thread count and GOMAXPROCS). A
        multi-threaded OpenBLAS would enlarge the denominator and make the bar
        harder rather than easier — but it would also make it meaningless, and
        refusing meaningless numbers in both directions is this gate's job.
      - the reference's own configuration is checked, not just recorded. A
        DYNAMIC_ARCH build that selected a pre-AVX2 kernel on an AVX-512 host
        reads low, and a low reference *inflates* keel's ratio — the one
        direction in which a gate must never fail. So the selected kernel name
        must be an AVX2-or-better target, and an unrecognized name fails too:
        missing knowledge should cost a human a minute, not silently widen a bar.
      - and the family is not merely allowed, it is CHOSEN BY MEASUREMENT
        (ruling on issue #31). The reference is the fastest of a coretype sweep
        forced through OPENBLAS_CORETYPE, best-of-N under this gate's own
        methodology, pinned for the run that produces the ratio, with every
        candidate's rate printed and the pin verified from the library's reported
        corename. The allowlist is the floor; the sweep is the ceiling. It exists
        because DYNAMIC_ARCH dispatches on an ISA feature bit: on vesta's Zen 4
        it ships a full-width Cooperlake kernel onto a double-pumped 256-bit
        datapath, where the AVX2 Haswell kernel is 6.7% faster — keel's own #24
        bug with the vendors reversed. Trusting that selector would delegate the
        denominator's one inviolable property to a heuristic this project has
        measured to be wrong on one of its three hosts.
      - a host needs a Go toolchain and OpenBLAS for that, and the execution
        hosts deliberately have neither (docs/hosts.md: cross-compiled static
        binaries, nothing installed). Provisioning is Scott's to approve, so a
        host that cannot produce a reference FAILS this gate and gets the exact
        commands for its own distro printed. It does not fall back to
        percent-of-peak. CLAUDE.md's "the OpenBLAS reference when available;
        otherwise say it isn't" is a rule about reporting numbers; using it to
        satisfy a gate criterion would be weakening the gate, which is the one
        option never available.

    EVERY gate host must produce a reference and clear the bar, and every host
    must have its clock established stable to be measured at all — asserted in a
    preamble before any benchmark runs, not assumed and not noted afterwards
    (DESIGN.md §5 rule 5, tightened by the ruling with #31, amended 2026-08-16 to
    name the instrument by what the host has: the performance governor where
    `cpufreq` is readable, which is every host this gate has run against and what
    the preamble below asserts). The old wording
    here was "at least one must clear it under the performance governor", which
    let antares contribute numbers from `powersave`: its first OpenBLAS reading
    of the sweep was 245.0 GFLOP/s against a 296-297 steady state, i.e. an 18%
    error in a denominator, decided by how recently the core had been busy. An
    unreadable governor fails too, on the same principle as the unrecognized
    kernel name: an unchecked precondition is not a met one. Percent
    of measured peak is printed for every host either way, because that number is
    informative even where it is not a criterion — and so is RETENTION, the share
    of its own microkernel the blocked loop nest keeps, printed on the same
    reported-never-judged footing. It is there because #26 is a named P5 input
    (DESIGN.md §4/P5) and an input needs a number: janus keeps ~77% where the Zen
    hosts keep 90-92%, and P5 should inherit that as a measurement it can re-run
    rather than a figure someone remembers. Judging it here would be P3 annexing
    P5's blocking-parameter work; printing it is how P3 hands that work over.

 6b. THE DENOMINATOR ON AN ISSUE-BOUND HOST, AND WHY IT IS NOT A CONCESSION.
    Also from the ruling on #23: where the P2 classifier says a host is
    issue-bound, the denominator is min(same-host OpenBLAS, roofline x measured
    peak). OpenBLAS's K-loop there is hand assembly folding accumulation and an
    embedded broadcast into single FMAs — instructions the intrinsic layer
    provably cannot emit (T12, #17/#18) — so it sits above the front-end ceiling
    keel's kernels are capped by, and 60% of it is a demand on the decode stage.

    The decision is the pure function `p3_denominator` in scripts/roofline.sh,
    driven by the same fixtures as P2's verdict and running before any
    benchmarking. It is one-sided (it can only lower a denominator, never raise
    one), it applies only to a host the classifier admitted on independent
    evidence, and it carries P2's anti-vacuity shape guard against the shape
    `Sgemm` ACTUALLY RAN — read from the keel-bench-kern marker of the very run
    that produced the ratio, not from the best shape on the shelf and not from a
    different host's log. A fatter kernel therefore cannot buy itself a lower bar.
    That guard is what surfaced issue #24: it refused the then-dispatched 4x32 on
    janus, and the fix was dispatch, not the threshold — janus now ships 2x32 at
    4.625 insns/FMA and is inside the guard on its merits (criterion 5b). Both
    ratios, amended and plain, are printed on every host: the gate's own leniency
    is a number, and §7 rule 7 applies to it too.

 7. WHAT THIS GATE DOES NOT CHECK. "Beta handling as kernel variants, not
    branches in the loop" and "packing SIMD-accelerated through the shim" are
    P3 design instructions, not gate criteria. Both appear here as provenance —
    the variant count and the packing backend come out of the config marker —
    and criterion 4 enforces them structurally, since a branch or a call that
    landed in the K-loop is exactly what the audit reports. Anything stronger
    would be this gate inventing criteria the design document did not set.

    Retention (#26) is in this category too, deliberately: it is measured and
    printed on every host and judged on none. The gap it names is real throughput,
    but it is packing and memory traffic rather than the front-end ceiling the
    roofline models, so it is neither excused by the amendment nor netted out of
    it — and closing it means sweeping KC/MC/NC, which DESIGN.md §4 parked in P5.
    Turning it into a P3 criterion would be scope migration; dropping it would
    leave P5 without a baseline. Printing it does neither.
```

---

## P4 — the routine lattice and Ssyrk's share of Sgemm

From `scripts/gate-p4.sh`, lines 2–227 at `c421486`.

```text
Gate P4 — see DESIGN.md §4/P4. Written at the START of phase P4, then made
green. Exits 0 only when every criterion for the phase holds. A red gate
blocks the next phase; there is no override flag on purpose.

Criteria (verbatim from DESIGN.md §4/P4):
  "all routines green vs oracle incl. upper/lower x trans x unit-diag lattice
   for `Strsm`; `Ssyrk` >=85% of `Sgemm` GFLOPS at same size."

and the routines that "all routines" names, from the two bullets above it:
  "`Sgemv` (both transposes), `Sger`."
  "`Ssyrk`, `Ssymm` as blocked GEMM with triangular masking in the C-update;
   `Strsm` as small unblocked triangular solves at the diagonal + GEMM
   rank-updates elsewhere (the BLIS recipe ...)."

How those are mechanized, and every judgement call involved:

 1. THE LATTICES ARE ENFORCED, NOT TRUSTED — AND P4 IS WHERE THAT MATTERS MOST.
    P3 had one flag pair (transA, transB); P4 has five routines carrying side,
    uplo, trans, diag and two strides between them, and every one of those flags
    selects a DIFFERENT CODE PATH rather than a different number. A green
    `go test` proves only that whatever ran, passed, and "whatever ran" is the
    entire question when the lattice has sixteen corners: Strsm with
    side=Right/uplo=Upper/trans=Trans/diag=Unit is a different traversal of a
    different triangle from side=Left/uplo=Lower/trans=NoTrans/diag=NonUnit, and
    a test that quietly ran eight of the sixteen reports the same "ok".

    So each routine prints one lattice marker enumerating every flag set it swept
    and the number of combinations it ran, and this gate checks three things per
    routine: that the required flag values are present (its own statement of the
    lattice, written here from the BLAS definitions rather than read from the
    test), that alpha and beta each include 0, 1 and a value that is neither —
    0 and 1 are the special-cased paths, so a lattice of only those exercises
    every shortcut and never the general multiply — and THAT THE ENUMERATED SETS
    MULTIPLY OUT TO THE REPORTED COUNT. The product is taken over every key on
    the line except `routine=` and `combos=`, which means a test that adds a
    dimension must also grow its combination count: a marker cannot claim
    coverage it did not run, and it cannot hide a dimension it swept only one
    value of either.

    DESIGN.md names "upper/lower x trans x unit-diag" for Strsm and this gate
    additionally requires SIDE. Left and right are not two spellings of one
    algorithm — one solves op(A)·X = alpha·B and the other X·op(A) = alpha·B,
    with different traversal order and different panel shapes — so a lattice
    without side is missing a factor of two on the routine DESIGN.md singles out
    as the one needing a lattice at all. Adding checks a phase implies is
    allowed; removing them is not.

 2. THE ORACLE'S COST IS A DECLARED PROPERTY OF EACH SIZE, exactly as in P3 and
    for the same reason: a float64 reference is affordable entry-by-entry at 65
    and not at 500, so the sweep may verify a seeded random SAMPLE of entries
    above a stated bound — but only if it says so per routine per size, with a
    sample floor and a printed seed, in a marker this gate reads. Sizes up to 65
    must be verified in full. The concession is written here rather than left to
    the test because "we sampled" is the kind of thing that starts at 500 and
    ends up applying to 17.

    The size list is this gate's own and it is smaller than P3's on purpose:
    1..17 for every remainder against any MR/NR, 31/32/33 and 63/64/65 for the
    shipped NR and a power of two, and 500 as the one size past EVERY blocking
    parameter (KC=384, MC=144). P3 already established the multi-block path at
    500/1000/2048 for the routine underneath all of these; what P4 needs from a
    large size is that its own masking and its own diagonal handling survive
    more than one KC block, and one such size does that. 1000 and 2048 would
    multiply the oracle cost — Strsm's reference is a triangular solve, so one
    entry costs the whole solve — for a repetition of a fact rather than a new
    one. Stated as a decision, not as an omission.

 3. CORRECTNESS RUNS WHERE THE SHIPPED PATH RUNS, and the derived routines are
    differential-tested across backends. The dev host is darwin/arm64 and cannot
    execute archsimd at all (docs/toolchain-notes.md T1), so the lattices are
    audited from the log of a host that ran them with the AVX-512 backend live.
    Each routine also reports which backends its runners exercised, and this gate
    requires at least the widest and the scalar reference on that host: DESIGN.md
    §5 rule 2 asks for backend-vs-backend agreement independent of the oracle,
    and for a routine derived from a shared microkernel that is the check which
    distinguishes "the derivation is right" from "the derivation and the oracle
    agree about a shape neither of them exercises".

 4. P2's KERNEL PROPERTIES ARE RE-CHECKED HERE, BECAUSE P4 IS THE PHASE MOST
    LIKELY TO ADD A SECOND KERNEL FAMILY. Triangular masking is the exact thing
    internal/block's package doc declined to do with masked microkernels — "a
    second family of microkernels ... doubling the kernel family doubles what has
    to stay zero-spill" — so the audit that says the K-loop is still one loop with
    no spills, no calls and no bounds checks runs on every gate from P2 onward and
    does not become optional because P4's own checks are green. The registry drift
    check comes with it: every shipped shape's recorded insns/FMA must equal what
    the audit counts, and a shape the gate does not audit is unrankable rather
    than lean, which is what makes "a new kernel appeared" visible here instead of
    in a benchmark six months later.

 5. THE DERIVATION IS CHECKED AS FAR AS A MARKER CAN CHECK IT, AND THE THROUGHPUT
    RATIO IS WHAT MAKES IT MORE THAN A CLAIM. DESIGN.md's "as blocked GEMM with
    triangular masking" and "small unblocked triangular solves at the diagonal +
    GEMM rank-updates elsewhere" are design instructions; a gate cannot read an
    implementation strategy out of a binary. What it can do is require the three
    derived Level-3 routines to report the SAME microkernel and the SAME blocking
    parameters the Sgemm in that same run dispatched to — an independent
    reimplementation would have nothing to report, and a second kernel family
    would report a different shape — and then, for Ssyrk, to measure the
    consequence. A triple loop cannot reach 85% of a blocked GEMM's rate, so
    criterion 7 is the derivation criterion with teeth and the markers are the
    part that says which shape the teeth closed on.

 6. WHAT "ALL ROUTINES GREEN VS ORACLE" INCLUDES BEYOND THE LATTICE. Every
    routine gets the edge coverage P3's Sgemm has, because none of it is
    Sgemm-specific: an ld wider than the matrix (how a caller passes a
    submatrix — the case where an off-by-one writes into somebody else's data
    rather than off the end of a slice), a zero dimension, argument panics, and
    the non-finite rules (alpha == 0 must not read A, beta == 0 must not read C,
    and a zero-padded panel meeting an infinity must not leak a NaN into C).

    Three routine-specific ones are required on top, and each is a property no
    oracle comparison can see, because they are about memory the routine must
    NOT touch:
      - Ssyrk: the untouched triangle. It updates one triangle of C and must
        leave the other bit-identical, so the other one is poisoned and compared
        bit-for-bit afterwards.
      - Ssymm and Strsm: the unreferenced triangle of A. Both read one triangle
        and must not read the other; poison the out-of-scope half with NaN and
        the result must be unchanged, or the masking is wrong in the one
        direction a correct-looking answer hides.
      - Strsm: unit diagonal means the stored diagonal is NOT REFERENCED, which is
        a documented BLAS guarantee callers rely on (a factored matrix stores L's
        and U's diagonals in one array). Poison the diagonal, ask for diag=Unit,
        and the answer must be the one for a unit diagonal.
    These are named in the extras marker and required by name, so dropping one
    is a red gate rather than a quiet regression in a test file.

 7. `Ssyrk` >= 85% OF `Sgemm` GFLOPS AT SAME SIZE, AND THE NUMERATOR IS CHECKED.
    This is the one criterion in the project where the risk is in the NUMERATOR
    rather than the denominator. Ssyrk does about half of Sgemm's arithmetic at
    the same n — it updates a triangle, n(n+1)/2 entries rather than n² — so a
    harness that counted 2·n²·k useful flops instead of k·n·(n+1) would report
    roughly TWICE Ssyrk's real rate and the 85% bar would be cleared by a routine
    running at 43% of Sgemm. CLAUDE.md's "never a number without its denominator"
    cuts the same way pointed at the top of the fraction, so:
      - the harness DECLARES the flop count it used, with the dimensions it used
        and the formula it applied, in a keel-bench-flops marker;
      - this gate RECOMPUTES that count from n and k, from its own statement of
        the two formulas below, and fails on disagreement;
      - and it checks "at same size" rather than assuming it: Ssyrk's n and k
        must equal Sgemm's m, n and k, from the same markers. Two rates measured
        at two sizes are not a ratio (DESIGN.md §7 rule 7).
    The declared count is the number the harness divides by, not a second
    statement of it — see bench/bench_test.go, which reports the metric and prints
    the declaration from one variable.

    The two rates come from ONE benchmark invocation on one host, so they share a
    frequency and a thermal state, and the ratio is taken net of both confidence
    intervals (numerator down by its CI, denominator up by its own) — the P2
    ruling on issue #14, and the reason bench_ratio_lo exists. Percent of measured
    peak is printed for both as provenance and judged for neither: P4's criterion
    is the ratio, and a percent-of-peak floor on Ssyrk would be a threshold
    invented here rather than one DESIGN.md set.

    THE VERDICT HAS THREE STATES, NOT TWO (ruled 2026-08-15 on issue #67). Net of
    CI answers one question — "is the whole interval above the bar?" — and reading
    its negative answer as "the ratio is below the bar" collapses two causes into
    one FAIL, which DESIGN.md §5.6 forbids. It happened: janus read 87.6% raw with
    ±4.0%/±3.0% intervals, bound 81.6%, FAIL; the same tree at the same commit read
    87.0% raw at ±0.0%, bound 87.0%, PASS. The raw quantity agreed within 0.6
    points. The FAIL recorded the weather. So:
      - PASS when the interval sits at or above the bar. This is bench_ratio_lo >=
        0.85, unchanged bit for bit — the third state is carved out of the old
        FAIL, never out of the old PASS, and nothing that was below the bar gets
        in on a lucky draw.
      - FAIL when the whole interval sits below the bar. Now a claim about Ssyrk.
      - UNMEASURED when the interval straddles the bar. The measurement cannot
        decide; the gate stays not-green; the remedy is DESIGN.md §4's one
        archived re-run, and for a host that is CHRONICALLY indeterminate here,
        a higher KEEL_BENCH_COUNT for this criterion on that host. Never a wider
        judgment, and never the raw ratio in place of the bound.
    And the flip-headroom — the symmetric CI at which the bound would reach the
    bar — prints per host per run, so the record shows how close each verdict ran
    to undecidable instead of leaving the reader to solve for it.

    Every criterion that reads a benchmark declares what it will read first
    (require_bench, DESIGN.md §5 rule 6): an absent measurement has exactly one
    verdict available to it, unmeasured, and never a silent pass or a red
    attributed to something else.

 8. P3's GATE IS CARRIED FORWARD BY RUNNING IT, NOT BY RESTATING IT — AND THAT IS
    ALSO THIS GATE'S ANTI-VACUITY GUARD. "Ssyrk >= 85% of Sgemm" is a ratio
    against a number P4 can move: the derived routines are meant to share
    internal/block's loop nest, and sharing it means editing it. A masked C-update
    path, a callback per tile, one more branch in the macro loop — any of those can
    cost Sgemm throughput, and every point Sgemm loses makes P4's own criterion
    EASIER. A gate whose bar falls when the code gets worse is not a bar.

    The only ratified statement of "the blocked Sgemm is still fast enough" is
    P3's own criterion: >= 60% of min(same-host OpenBLAS, roofline x measured
    peak), on every gate host, with the coretype sweep that chooses the reference
    and the classifier that chooses the denominator. Restating any of that here
    would duplicate two hundred lines and, worse, would create a second place
    where the threshold lives. So this gate RUNS scripts/gate-p3.sh and requires
    it green. That carries the whole P3 contract rather than the one criterion —
    the oracle sweep, the shape-agreement checks, P2's floor on the dispatched
    microkernel — which is what "never begin phase N+1 with a red gate" means once
    phase N+1 starts editing phase N's code.

    Consequences, all deliberate:
      - it is the expensive check, so it runs LAST: P4's own failures surface in
        minutes instead of after a full P3 run.
      - gate-p3 requires a clean working tree (its criterion 6 measures
        `git archive HEAD`). So this gate refuses a dirty tree UP FRONT and does
        not run the delegated gate at all in that case, rather than spending the
        whole run to arrive at a failure that was knowable at the start. Two
        failures are reported, the dirty tree and the P3 gate not having run,
        because those are different facts.
      - the delegated gate's full output is written to a file and its path is
        printed, because CLAUDE.md requires gate output verbatim in the umbrella
        issue and a summary line is not that.
      - DESIGN.md §4's one-re-run allowance for a failing THROUGHPUT SENTINEL
        reading applies to a sentinel criterion inside the delegated gate exactly
        as it does when gate-p3 is run directly: one immediate re-run, both
        outputs archived, never for a correctness criterion. It is the operator's
        allowance and this script does not implement it.

 9. WHAT THIS GATE DOES NOT CHECK. There is no percent-of-peak floor and no
    OpenBLAS comparison for Ssymm, Strsm, Sgemv or Sger. DESIGN.md sets a
    throughput criterion for exactly one P4 routine and holds the others to
    correctness, so inventing bars for them here would be this gate writing
    criteria the design document did not set — the mirror image of weakening one.
    Their rates are worth measuring and P5 is where a scaling story gets told;
    what P4 owes is that they are right, that they are derived rather than
    rewritten, and that the routine DESIGN.md did put a number on holds it.
```

---

## P5 — threading, the race detector, and the stock-toolchain build

From `scripts/gate-p5.sh`, lines 2–241 at `c421486`.

> **`SCALE_FLOOR` and the ≥6× criterion quoted below are RETIRED** (ruling on issue #6,
> 2026-08-20; `DESIGN.md` §4/P5 carries the amendment and the falsifier). The judged class
> is now compared to a per-host attainable ceiling this project measures, under
> `CEIL_FRACTION`. The text below is **not** edited to match, because it is a verbatim lift
> dated to `c421486` and a verbatim archive that gets quietly corrected is neither verbatim
> nor an archive. Read it as the reasoning that was live then, including the reasoning that
> the ruling overturned — the passage comparing `STRSM_FLOOR` to `SCALE_FLOOR` as "just
> margin" is exactly the premise the rank inversion falsified.

```text
Gate P5 — see DESIGN.md §4/P5. Written at the START of phase P5, then made
green. Exits 0 only when every criterion for the phase holds. A red gate
blocks the next phase; there is no override flag on purpose. P5 is the last
phase, so here "blocks the next phase" means blocks the release.

THE CRITERIA, verbatim from DESIGN.md §4/P5's gate line:
  ">=6x single-thread throughput at 8 cores on 4096^3; race detector clean;
   `go vet`/`golangci-lint` clean; scalar-only build green on stock toolchain.
   Retention stays reported rather than judged: #26's target is a direction to
   work in, not a threshold invented after the fact."

and from the design bullets above it, which say what is parallelized and what
the parallelization may not cost:
  "Parallelize the MC (ic) loop over a bounded worker pool sized by
   `runtime.GOMAXPROCS(0)`; shared packed-B panel per NC iteration, per-worker
   packed-A buffers from a `sync.Pool`. No background threads, no state between
   calls."
  "Runtime dispatch finalized: `avx512 -> avx2 -> scalar`, overridable by env
   `KEEL_FORCE=scalar` for testing."
  "The package must compile and pass (scalar path) *without* `GOEXPERIMENT=simd`."
  "Scaling benchmark, README with honest numbers, `doc.go`."

and the two rulings of 2026-08-12, also recorded in §4/P5: the phase is ordered
internally (single-thread remediation, then the parallel nest, then this gate),
and the scaling floor binds BY PARALLELISM CLASS rather than by routine list.

HOW THOSE BECOME CHECKS, and every judgement call involved:

 1. THE FLOOR BINDS BY CLASS: Sgemm, Ssyrk AND Ssymm ARE ALL JUDGED AT >=6x.
    They are one parallelism class — GEMM-shaped nests over independent tiles,
    no cross-iteration dependence — so the same parallelization is available to
    each by construction, and P4 already proved they share the loop nest that
    provides it. Judging only Sgemm would leave the place a serialization bug is
    most likely — the triangular C-update masking, which decides per tile and
    per row and could as easily decide per worker — measured by nothing. A
    derived routine is not a lesser routine; it is the same code with a mask.

    Strsm is a SECOND CLASS, and its floor is now RATIFIED AT >=7.0x — but the
    model this gate spent five phases asking for is the one thing the ratification
    THREW AWAY (ruled 2026-08-16, #37/#89). Both halves matter, so both are here.

    The deferral, as it stood: Strsm's diagonal solves impose a dependency chain
    the other three do not have, and how much parallelism it has left depends on
    side and shape. Writing 6x on it without a model would have been inventing a
    threshold — the move this project has refused six times — so the gate required
    the scaling to be measured, required the parallelism model behind it to be
    stated (what fraction of the work at this shape is rank update versus diagonal
    solve), reported both and judged neither.

    THE MODEL IT REQUIRED IS FALSIFIED, and by its own favourite direction. Read
    diag_solve as an Amdahl serial fraction — which is what "state the model behind
    the number" was asking for — and s=0.01587 at p=8 gives a hard ceiling of
    1/(0.01587 + 0.98413/8) = 7.2001x. Nine readings across three µarchs came in at
    7.403–7.668 net of CI, and ALL NINE SIT ABOVE THAT CEILING; the lowest, janus at
    7.403, is +2.82% above a number that was supposed to be unreachable. A model
    whose ceiling the data clears is not a model needing adjustment, it is a dead
    premise: the solves are not serial in effect, they overlap the rank updates,
    because Trsm splits its right-hand sides at the top (MB=64 < MC=144 leaves the
    ic loop one iteration, so the split had to go elsewhere) and the solves ride
    that split. That the falsification points the *favourable* way makes it no less
    a falsification, which is why this comment says falsified and not "refined".

    THE REPLACEMENT MODEL is the per-(jc,pc) B-packing residue plus the makespan
    tail of the last unit claimed — #65's Amdahl term for this nest, measured here
    as an implied serial fraction of 0.62–1.15% across the nine readings, strictly
    BELOW the 1.587% work share, which is the arithmetic signature of solves that
    parallelize. STRSM_FLOOR is 7.0x under that model: a REGRESSION BAR set below
    every observation, not a threshold derived from a ceiling. The distinction is
    the whole ruling — a floor derived from a ceiling the data refutes would be
    theatre, while a floor sitting 0.403x (5.76%) under the lowest of nine readings
    is just margin, which is what SCALE_FLOOR is too. 7.4x was the alternative and
    is rejected as unshippable: it would leave janus 0.04% of headroom and flake.

    Its nine readings are boost-off desktop measurements and are not comparable to
    a virtualized reading (#66, below). The
    work split keeps printing, because it is a true fact about the shape and the
    input to the falsification — but the log now says in words that it is a work
    split and not the serial fraction, and reprints the ceiling it would imply
    beside the reading that clears it, so the dead premise stays visibly dead
    rather than becoming folklore. #89 tracks the criterion wording itself.

    THE CLOCK IS MEASURED, NOT SET (amended by ruling 2026-08-17 on #66,
    superseding the boost-off methodology; DESIGN.md §4/P5). The 2026-08-15
    finding stands: dividing an 8-thread rate by a 1-thread rate taken on an idle
    machine is not a ratio, because one thread runs in a frequency regime eight
    threads physically cannot enter — and the evidence was in the shape of the
    misses, not their inconvenience (the two hosts that missed retained the MOST
    of their own single-thread peak, Zen 4 92% and Zen 5 59%, while Skylake-X at
    35% cleared everything twice; scaling deficits do not sort by single-thread
    excellence, boost tables do). But its remedy — set the knob off on both arms
    and read it back — is unavailable on a virtualized fleet, which exposes
    neither cpufreq/boost nor intel_pstate/no_turbo, so the precondition refused
    every host and took criterion 9 with it. This gate now sets nothing: criterion
    1's substitute instrument measures the clock both arms shared, and the
    residual handicap is disclosed beside the ratio rather than removed. It is
    also what a caller on the same instance type gets.

 2. THE TWO RATES COME FROM ONE INVOCATION, SO THE THREAD COUNT IS PART OF THE
    BENCHMARK NAME AND NOT PART OF THE FLAGS. `go test -cpu=1,8` looks like the
    tool for this and is a trap: it distinguishes the two runs only by the
    "-N" GOMAXPROCS suffix on the benchmark name, and both bench_stat and
    bench_expect STRIP that suffix (scripts/bench.sh, so that a B/s row and a
    sec/op row of the same benchmark can be told apart by unit). benchstat would
    therefore aggregate the one-thread and the eight-thread samples into a single
    row, and this gate would divide a mixture by itself and read 1.0x. So the
    harness names the thread count (`Scale/Sgemm/n=4096/threads=8`) and sets
    GOMAXPROCS itself, and the two rates are two rows of one CSV.

    One invocation is also what makes the ratio a ratio: same binary, same
    frequency, same thermal state, same page cache. Two invocations would have to
    go through bench_gflops_lo, whose own doc comment forbids precisely this use.

 3. THE PARALLELISM IS CHECKED, NOT ASSUMED. A row named threads=8 that silently
    ran on one worker yields 1.0x and reads as a performance problem; a row named
    threads=1 that ran on eight yields 1.0x and reads as the same thing. Both are
    measurement failures wearing a performance failure's clothes. So the harness
    declares, per row, the GOMAXPROCS it set and the number of workers the library
    actually used, and this gate requires both to equal the thread count in the
    name. "A bounded worker pool sized by runtime.GOMAXPROCS(0)" is the design
    instruction; this is the part of it a gate can read.

 4. THE HOST MUST HAVE THE CORES THE CRITERION NAMES, AND THE TOPOLOGY IS PART OF
    THE RECORD. "At 8 cores" is not "at GOMAXPROCS=8 on whatever that lands on":
    eight goroutines on four physical cores and their SMT siblings cannot reach 6x,
    and would fail this gate for a reason that is not keel's. So the gate requires
    >= 8 CPUs, prints threads-per-core and cores-per-socket per host, and PINS
    every judged invocation to eight distinct physical cores within one NUMA node,
    ONE PER CACHE DOMAIN since 2026-08-22 — the ceiling arm under the identical
    mask, since a share whose numerator and denominator came from different
    placement methodologies is not a share. No taskset, no such node, or no cache
    level to partition it by, is status 121 and nothing measured: a free-placement
    reading wearing a `pinned8` label is the one artifact the era ledger exists to
    make impossible, and a *confined* reading wearing it is the same forgery one
    layer in — which is why the mask records the domain of each core it selected
    and the criterion asserts distinct(domains) = min(width, the node's), rather
    than reading GOMAXPROCS back and calling the shape established. It cannot:
    GOMAXPROCS is 8 whether the eight cores share one CCD or eight, and on
    keel-zen5 the confined form measured 5.96× less stream bandwidth. That is the
    pinning decision issue #15 (vesta's bimodal Sdot,
    CCD placement) asked for, ruled into DESIGN §5 rule 5 on 2026-08-21, amended
    to the spread form on 2026-08-22, and implemented in `remote_exec`; #15 stays
    open until its own rows are
    re-measured under the mask, because an adoption closes on measurement and not
    on a ruling. If a host misses the floor with sibling placement as the visible
    cause, that is a finding to write up and take to a ruling, not a bar to lower
    here — and now the refusal is how that finding arrives.

 5. PARALLELISM IS A CORRECTNESS QUESTION BEFORE IT IS A THROUGHPUT ONE, AND THE
    STANDARD IS BITWISE. Partitioning the MC loop splits C by row panels; it does
    not reassociate any single output element's reduction. So a correct parallel
    nest returns BIT-IDENTICAL results to the serial one at every thread count,
    and the tests must say bitwise rather than fall back on a tolerance. That is
    stronger than the oracle sweep can be, and it is also a design tripwire: if
    some future implementation parallelizes the K loop, bitwise identity breaks,
    and the correct response is a documented ruling on non-determinism, not a
    widened epsilon. The declared thread counts must include 1, 8 and a value that
    divides neither the block count nor the core count evenly (3), because an
    off-by-one in a row partition hides perfectly at 1, 2, 4 and 8.

 6. "NO BACKGROUND THREADS, NO STATE BETWEEN CALLS" IS CHECKABLE, SO IT IS
    CHECKED. After a call returns, the goroutine count must be back at its
    pre-call baseline — that is what "no background threads" means operationally,
    and it is the difference between a pool that parks workers forever and one
    that ends with the call. And a second identical call must produce a
    bit-identical result, which is what catches a sync.Pool buffer that a later
    call reads before writing. Both are named in a marker and required by name,
    because both cost nothing until a caller runs keel inside something that
    counts goroutines or calls it twice.

 7. THE DISPATCH CHAINS ARE EXERCISED BY THIS GATE, NOT ONLY REPORTED BY THE
    TESTS — AND THERE ARE TWO OF THEM. Level 1 dispatches avx512 -> avx2 ->
    scalar; Level 3 dispatches avx512 -> scalar (ruled 2026-08-12, #40:
    internal/kern has no AVX2 microkernel, no host here is AVX2-only silicon, and
    KEEL_FORCE=avx2 on an AVX-512 part is evidence about correctness, not about
    performance — so a third Level-3 rung would be advertised with nothing able
    to back it). The tests declare both chains and that an unrecognized
    KEEL_FORCE refuses to run; the gate additionally re-runs the binary with
    KEEL_FORCE=scalar, avx2 and avx512 and requires the library's own
    active-backend markers to name what was asked for, then runs it with a
    nonsense value and requires a non-zero exit. A dispatch chain that is only
    self-reported is a chain nobody pulled on.

    The narrowing is enforced in the direction that costs something. For a
    Level-1-only rung the gate requires the microkernel to come back *scalar*
    and the library to say so: an avx2 microkernel appearing at Level 3 fails
    here, because the ruling narrowed what is claimed and a claim that grows
    back silently is what this gate exists to catch. The debt keeps its trigger
    on #40 — an AVX2-native evidentiary host — so P5_KERN_CHAIN grows a rung by
    a measurement, not by a session's convenience.

    Marker contract, since the chains are now per level:
      keel-p5-dispatch: l1=avx512,avx2,scalar kern=avx512,scalar
    A single `chain=` field cannot state the Level-3 narrowing at all, and a
    ruling that cannot be stated is one the next session re-litigates.

 8. RETENTION IS PRINTED AND NOT JUDGED. DESIGN.md §4/P5 says so in the gate line
    itself: issue #26 is a direction to work in. It is printed for the same reason
    gate-p3 printed it — so P5's remediation stage answers to a re-runnable number
    rather than a remembered one. Percent of (8 x the single-thread peak) is
    printed on the same terms, and with its own caveat: that denominator assumes a
    clock that does not drop with core count, which is not true of these parts, so
    it is an over-estimate of the ceiling and is labelled as one.

 9. THE README'S NUMBERS ARE RE-MEASURED BY THE GATE THAT SHIPS THEM. "README with
    honest numbers" is the one design instruction here that could mean nothing, so
    it is mechanized: the README carries a delimited table whose rows name a CPU
    MODEL (never a hostname — .keel-hosts is gitignored infrastructure and real
    hostnames are not source), a benchmark, a thread count, a rate, and the
    DENOMINATOR that rate is a fraction of. This gate parses those rows, matches
    each to the host whose CPU model it names, and fails if a published number
    disagrees with this run by more than README_TOL. Any GFLOP/s figure OUTSIDE
    that block fails too: a number a reader cannot trace to a machine and a
    denominator is what §7 rule 7 forbids, and a number this gate cannot
    re-measure is a claim rather than a measurement. This is the rule that stops
    the published table drifting from the code one optimistic edit at a time.

10. P4's GATE IS CARRIED FORWARD BY RUNNING IT, WHICH CARRIES P3's AND P2's — AND
    THAT IS THIS GATE'S ANTI-VACUITY GUARD. ">=6x single-thread throughput" is a
    ratio whose DENOMINATOR this phase is chartered to change. Stage 1 makes the
    serial path faster (#26, #36, #37, and the deferred measurements on #21/#22),
    which makes this bar HARDER — good — but a parallel nest that accidentally
    slowed the serial path would make it EASIER, and a bar that falls when the
    code gets worse is not a bar. The absolute statements belong to P4 (Ssyrk >=
    85% of the same host's Sgemm) and P3 (Sgemm >= 60% of same-host OpenBLAS) and
    P2 (the dispatched microkernel's floor and spill audit), so this gate RUNS
    scripts/gate-p4.sh, which runs scripts/gate-p3.sh. Restating any of those
    thresholds here would create a second place where they live, and two places
    is how a threshold gets quietly relaxed in one of them.

    Consequences, all deliberate and all inherited from how gate-p4 does this:
      - it runs LAST, because it is by far the most expensive check;
      - the delegated chain builds `git archive HEAD`, as does the native
        race-instrumented build below, so this gate refuses a dirty tree UP FRONT
        rather than an hour in;
      - the delegated output goes to a file whose path is printed, because
        CLAUDE.md wants gate output verbatim in the umbrella issue and a summary
        line is not that. gate-p4's log names gate-p3's in turn, so the whole
        chain is readable from this one line;
      - DESIGN.md §4's one-re-run allowance for a failing THROUGHPUT SENTINEL
        reading applies inside the delegated gates exactly as when they are run
        directly. It is the operator's allowance, this script does not implement
        it, and it never applies to a correctness criterion.

11. WHAT THIS GATE DOES NOT CHECK, and why not. It sets no absolute GFLOP/s floor
    at 4096 and no OpenBLAS comparison at 4096: DESIGN.md's P5 criterion is a
    scaling ratio, the absolute bars live at 2048 in the delegated gates, and
    inventing a second absolute bar at a second size would be this gate writing
    criteria the design document did not set. It does not judge multi-threaded
    OpenBLAS either — that comparison needs a stated threading model per library
    before it means anything, and nobody has ratified one. And it does not check
    release mechanics (a v0.1.0 CHANGELOG section, a tag): those belong to the
    release, not to the phase gate.
```
