# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, minor versions may contain breaking changes.

## [Unreleased]

### Added
- **The planted-delta control, beside the A/B harness and run before any host is touched** —
  `ab_control` in `scripts/ab.sh` (`#141`, authorized 2026-09-01: *"an A/B that can't see a planted
  delta hasn't measured a real one"*). #141's own checklist asked for this **before** the instrument
  was written and it was skipped, which is how a harness that could not fail got built: `ab_host`
  named both arm files by SHA alone, a null A/B resolves both SHAs to one string, the second `cp`
  overwrote the first, and benchstat compared a file to itself and printed `~ (p=1.000)` — the exact
  token `bench_compare` reads as no drift. The control writes two synthetic sample sets differing by
  an exact `×1.1`, **through the same `ab_arm_file` that names the measured arms and with this run's
  own two SHAs**, and requires the recovered delta to read `+10.00%`; anything else exits 2 before a
  host is contacted. Driven both ways rather than asserted: with `ab_arm_file` reverted to the
  pre-fix naming the control prints `a +10.00% plant read as 'no delta at all'` and benchstat's own
  `abc1234.txt#0` / `abc1234.txt#1` self-comparison beneath it. It cannot see whether the host's rows survive
  the pipeline, nor whether either arm ran; what it establishes is that the comparison is alive.
  Authorized at *"~30 lines"* and **+71 measured**, with the flags line a further **+58**; both are
  booked as authorized debt in `docs/apparatus-ledger.md`, unpaid, and the estimate-to-actual gap is
  stated there rather than argued down. The apparatus ratio moved 1.82x → 1.84x on a *smaller*
  unrounded increment than the +137 that printed as one hundredth a session earlier (0.01439 against
  0.01528, library term constant at 8964) — which is the reading rule earning itself again, not
  retiring: the +129 came from the diff.
- **Every A/B log now states what the far side will execute** — `ab_arm_provenance` prints each arm's
  sha256, size and the build flags read out of the binary with `go version -m` (`build_settings` in
  `scripts/remote.sh`), and `builder_toolchain`'s existing line carries the flags too, so a mid-run
  **flag** change is now as visible as a mid-run compiler change (`#58`, `#141`). Read off the
  artifact, never off the shell that meant to pass them. Measured at `ed17c57`: four builds of
  `./bench` from four paths gave **four distinct digests**, and the size is not monotone in path
  length — a 9-character path produced a *larger* binary than the 25-character repo path, and paths
  of length 9 and 44 produced identical sizes with different digests, so a size comparison alone
  could have missed the difference outright. This adds a word to an `info` line, so it moves no
  verdict and no normalized certificate key (rule 24's comparison anchors on `^  (PASS|FAIL|…) `);
  a raw-text diff against the pinned `v0.1.0` certificate will show it, and
  `docs/certificates/v0.1.0.md` says so.
- **`scripts/ab-bench.sh {l1|edge|drift} [BASE_REF]` — one parametrized caller in place of
  `l1-bench.sh` and `edge-bench.sh`, and it carries #141's decisive instrument** (`#141`, `#131`,
  ruled: *"two callers keyed to two benchmark sets plus a third for this one is the wrong shape"*).
  The three presets differ only in the strings `ab.sh` already took as parameters, so the preset
  table is the entire difference and nothing else asks which one it serves — #131's own rule, applied
  to the callers. `drift` is a **null A/B**: `BASE_REF` defaults to `HEAD`, both arms are one source,
  and the only difference between them is elapsed time on the machine, which is the quantity
  `-count` cannot reach (every sample of one benchmark sits in one contiguous window). It refuses a
  dirty tree, because a null A/B with uncommitted work in it is an unlabelled comparison of two
  builds; its filter and count are the gate's, for comparability with the archived `janus` runs.
  **Measured, not estimated: `+41` net shell, of which the refactor is `+36`.** Against the tree
  that is debt and it is booked as such in `docs/apparatus-ledger.md`; against the counterfactual a
  third thin caller would have cost `~163` total against this file's `151`. Every measured fact in
  the two deleted headers survives — the four cache-resident sizes, T19's instruction counts, #22's
  A/B/C framing, why `BenchmarkSgemm` is excluded, the void-on-control rule and the three
  between-binary layout floors (1.71/0.99/1.32%). One claim did **not** survive because it had gone
  false: `edge-bench.sh`'s closing note said its numbers came from *"the hosts' own go1.26.5"*, and
  `remote.sh` cross-compiles locally and ships a static binary, so no host's toolchain has ever
  entered one of these runs.
- **`docs/certificates/v0.1.0.md` — the canonical evolving provenance record for the tag**, with the
  release body reduced to a pointer at it (`#146`, ruled). Three appended addenda is an addendum
  ladder: a document whose corrections are ordered by when they were written rather than by what they
  correct. The tag object, its commit and the body's existing text are untouched — a dated record
  does not go false, it goes stale — but a reader consults the body **today**, and the second
  addendum's present-tense *"has therefore **never been judged** on `skx`, `zen4` or `zen5`"* went
  false at `7daf0f2`. The third addendum states that in one sentence with the log linked and its
  sha256, and points at the tracked file; corrections accumulate there from now on, where they are
  diffable, reviewable and inside `citation-lint.sh`'s enumeration — verified, not assumed: the new
  file contributes one citation site and it resolves.
- **§5 rule 24 — a pre-registered prediction is stated in the instrument's own output space AND at
  that instrument's own row granularity, or it constrains nothing** (`DESIGN.md`, `docs/rulings.md`).
  The **second clause arrived 2026-09-01, one day after the first**, ruling on #141's fork (*"a
  pre-registered predicate governs exactly the rows it registered thresholds for — supplying
  thresholds after seeing the kc≠128 deltas would be fresh judgment wearing pre-registration's
  clothes"*). The fork was registered over `Kernel/*/avx512` with thresholds derived at `kc=128`, the
  only depth `gate-p5` measures, while the drift driver renders four `kc` values per shape:
  re-derived mechanically from the tracked `archive/pinned8/drift-janus-df999da-{base,new}.log`, whose
  two arms render an identical row set, the 23 rendered rows partition as **3 adjudicated + 9 reached
  without a threshold + 11 out of domain** (8 `Kernel` scalar — `6x32` has no scalar arm — and 3
  `Peak`). Restricted to its calibrated three the fork returned one branch decisively; read literally
  it returned the other. **One table, one predicate, both branches** — so the free parameter had moved
  from the threshold to the row set, and the verdict for a reached-but-unthresholded row is now
  `UNMEASURED`, never a branch. Also corrected here: the first write-up said the predicate covers
  *"12 AVX-512 rows for which no threshold was ever derived"* and the ruling quoted the figure —
  twelve is the **scope**, nine were unthresholded, and a sentence that used one number for two
  quantities is rule 24's own subject one level up. #141 itself closes on **transient** for the
  certificate's 47.90% reading, on the adjudicable evidence only (bands 11.4× narrower than the
  excursion's, medians agreeing to +0.4%, sentinel share 47.94/47.83); the kc≠128 direction-split
  files as a new finding rather than as a branch verdict. Scott's ruling on his own prediction for
  the #113 certificate comparison: both registered deltas were digit-only (`8 row(s)` → `9 row(s)`,
  `6.067x` → `6.066x`), the comparison normalizes every numeral to `#`, so each accounts for **0 of
  the 7 changed members** — a prediction the instrument cannot render is not a weak constraint but
  none, wearing rigour. Recorded with it: `72` verdict lines and `72` distinct keys is an
  **injectivity** property, not a set identity, and reading it as agreement is what let *"that is
  the entire diff"* be published over an abridged rendering. The seven members are now classified
  individually rather than counted — 1 the run's actual FAIL, 4 one commit's wording across two
  surfaces (3 reported at the time, 1 not), and **2 decoration**: window-step sign flips on
  `keel-skx` and `keel-zen4` whose magnitudes are an order of magnitude under their own floors, so
  both runs render the same tie verdict. The normalization erases the magnitude and keeps the sign,
  which is the mirror image of the same defect and stays open on #113.

### Changed
- **`gate-p3` prints both terms of the sentinel's percent-of-peak, on the line above the verdict
  that divides them** (`gate-p3.sh`, `#141`, ruled — the disclosure exemption: *"criterion-honesty
  lines on the signing path don't wait for the ledger"*). The criterion published a ratio and neither
  term appeared in any log it wrote, which is how #141's numerator and denominator had to be
  **reconstructed** from archived samples after the fact — and reconstruction is what happens when
  disclosure fails. Rendered from the tracked `68a9bec` samples before landing: *"the ratio's two
  terms, same invocation: 2x32/avx512/kc=128 99.52 GFLOP/s over Peak/avx512 207.75 GFLOP/s"*, whose
  quotient reproduces the gate's published 47.9% exactly; an absent benchmark name renders empty
  rather than a spurious number, checked as the negative control. **`+5` shell, ledger-stated exempt.**
- **Criterion 5b's roofline branch gains its §5 rule 12 coverage line** (`docs/gates.md`, `#141`,
  ruled). Excluding the shape under test from the ceiling set buys independence from that *shape*,
  not from the *host*: every ceiling mix is measured on the same machine in the same invocation, so a
  shape-uniform host effect scales numerator and denominator together and is **absorbed** — the
  criterion renders a clean verdict over it and reports nothing. Only a **direction-split** excursion
  reaches a verdict, and #141's third instance reached one by the accident of its shape: on `janus`
  the ceiling set stayed nearly still (`6x32` +2.59% against `2x32` −9.76%), so the roofline moved
  −1.4% (49.3% → 48.6%) while the share moved −9.8% (47.9% → 43.2%). A green 5b is therefore not a
  bound on host stability and cannot become one from the inside — an uncorrelated denominator would
  have to come from a different run, which is a different instrument, and that instrument is now
  `scripts/ab-bench.sh drift`.
- **P2's throughput floor is judged on every fleet host; `.keel-sentinel` is no longer a selection
  input** (`remote.sh`'s new `sentinel_hosts`/`sentinel_declaration`, `gate-p3.sh`, `#146`, ruled).
  The retired precedence — `$KEEL_SENTINEL_HOST` > `.keel-sentinel` > the fleet — let a gitignored,
  machine-local file with a three-week-old mtime decide which host held a **judged** role, so P2's
  floor was judged on one lab box (`janus.local`) in the #113 re-measurement *and in the run that
  signed `v0.1.0-a2`*, with `keel-skx`, `keel-zen4` and `keel-zen5` each printing *"not a sentinel,
  so P2's floor is not judged here"*. Now: the set is derived from `.keel-hosts`/`$KEEL_REMOTE_HOSTS`
  and nothing else; a present `.keel-sentinel` is reported **with its mtime** as not read; a stale
  `$KEEL_SENTINEL_HOST` **fails** the gate rather than being silently ignored, because a no-op
  override reads exactly like an honoured one; and an out-of-fleet sentinel for deliberate
  characterization work is declared with `$KEEL_SENTINEL_OUT_OF_FLEET`, which **unions** with the
  fleet — no flag can subtract a fleet witness — and prints into both the run's `.cmd` and the log's
  declaration row. Scott's ruling says *"a fleet host chosen by a declared deterministic rule"*; the
  rule implemented is **every** fleet host, disclosed at the site and on #146, because any proper
  subset needs a tie-break among equals and judging more hosts can only turn a green red. The
  resolver was lifted rather than copied: `exercise-dead-host.sh` was re-parsing the same file to
  compute its own scope sentence and would have kept printing a confident, wrong one.
- **A run's `.cmd` states its whole input closure — environment *and* files *and* defaults**
  (`detach.sh`, `#146(c)`, ruled). §5 rule 21's namespace clear made the record complete over one
  channel; an `unset` loop cannot reach a file, so the record of the run that signed `v0.1.0-a2` was
  clean for the second time by a second mechanism. Every `.keel-*` file is now copied in verbatim
  with its mtime, **by glob and not by list** (a hardcoded list is how the next decision file goes
  unenumerated) and **unfiltered** (the obvious filter, skip tracked-and-clean, would have excused
  `.keel-hosts` — the file at the centre of the original incident); the defaults channel is stated by
  naming the revision, which is also the first time the `.cmd` recorded which tree it launched. An
  empty closure prints its header with no members, so *enumerated, nothing there* is distinguishable
  from *never enumerated*. `detach-test.sh` gains the arm, shown to fail first: with the copy
  reverted, a planted `.keel-sentinel` that decides a judged host leaves no trace. 16 arms, GREEN.
- **§5 rule 21 amended, and rule 22 cross-referenced: a total restatement is only as total as the
  set of channels it enumerates** (`DESIGN.md`, `docs/rulings.md`). Scott's ruling: *"rule 21 and
  rule 22 discovering they're the same rule from opposite ends: the certificate's input closure is
  what the declaration must state, not merely what the clear must reset."* The amendment also
  retires rule 21's own coverage item 2 — *"`scripts/detach.sh` has no automated test at all"* — and
  states what stays unexercised. That list named the first on-fleet P2 judgment as a witness not yet
  taken; it was taken on 2026-09-01 (next entry), which narrows the list to the out-of-fleet branch
  and the refusal path rather than clearing it.
- **`docs/hosts.md` and `docs/gates.md`**: the sentinel-selection prose is corrected in place with
  its date, and the *"janus keeps that role through P5"* clause of 2026-08-12 is marked **VOID for
  any judged run** — janus's standing as the host where instruction count binds is a statement about
  silicon, not about who judges. Both files now carry the on-fleet result in place of the
  *"never been judged"* clause, which is exactly the kind of sentence that goes false without an edit.
- **P2's floor is judged on the fleet, and it holds on all three hosts** — the first such reading in
  the project, deliberately taken on a **non-certificate** run so that a certificate is not a
  rendering's debut (`archive/pinned8/p2onfleet-afb108e.log`, `gate-p3: GREEN`, 52 PASS / 0 FAIL / 0
  UNMEASURED, `KEEL_BENCH_COUNT=30`, `go1.27.0`, on-demand `c5n.18xlarge`/`c7a.48xlarge`/`c8a.48xlarge`,
  63 min, ≤$25.37 at a measured $24.09/hr). Each figure divides by that host's own `BenchmarkPeak`
  register-only ceiling **measured in the same invocation as the kernel**, and by nothing else:
  `keel-zen5` 187.3 / 287.7 GFLOP/s = 65.1% and `keel-zen4` 113.3 / 117.0 = 96.8%, both fma-bound
  against the flat 55% floor; `keel-skx` 88.79 / 185.6 = 47.8%, which is 96.9% of that shape's 49.3%
  issue roofline against the 90% bar. The two fma-bound percentages are
  not comparable to each other — 96.8% of 117.0 is the smaller rate. This criterion has no OpenBLAS
  term at all. The gate's separate mission criterion does, it passed on all three in the same log, and
  its denominator is **not** the same quantity on each. `keel-zen5`: 64.6% of plain 1-thread OpenBLAS
  0.3.26, unpinned at `corename=Cooperlake`, its coretype sweep having found no cross-family winner
  beyond drift. `keel-zen4`: 90.4% of plain OpenBLAS **pinned to `Haswell`**, which its own sweep
  measured 5.5% above DYNAMIC_ARCH's `Cooperlake` choice (a 5.90 GFLOP/s cross-family win against a
  0.20 GFLOP/s same-family drift) — on Genoa the AVX2 reference kernel beats the AVX-512 one, the same
  direction as that host's 117.0 GFLOP/s peak. `keel-skx`: 75.9% of its issue-capped **roofline**
  denominator, and 39.1% of plain OpenBLAS — the amended denominator doing exactly the work it was
  ruled for. The declaration row rendered as designed: *every fleet host and nothing else*, with the
  machine-local `.keel-sentinel` reported by name and mtime as **not read**.
- **The sentinel criterion's percent-of-peak is published as a ratio only: neither its numerator nor
  its denominator appears anywhere in the gate log.** Found while writing the entry above — the
  first draft attributed the *Sgemm* section's peaks (288.2 / 117.0 / 186.3 GFLOP/s) to the sentinel,
  and two of the three are the wrong number for it, because that section is a separate invocation. The
  real terms live only in the run's `bench-*.txt` sample files, which are written under gitignored
  `build/`, so a published percent-of-peak was recoverable on one laptop and nowhere else — the exact
  thing `archive/pinned8/README.md` says the archive exists to prevent. All 13 sample files from this
  run are now tracked beside its log; each names its own host, `gomaxprocs` and toolchain in its
  header, so the mapping is a property of the files rather than of their names. What is **not** fixed:
  the gate still prints only the ratio, and printing the two terms is a one-line `info` in
  `gate-p3.sh` deferred to a session that can pay for shell under the apparatus cap.
- **The same defect at a worse site: `janus` was the sole judged sentinel of both the `v0.1.0-a2`
  certificate and #113's RED, and it is named nowhere in either run's tracked log.** Zero mentions
  of `janus` in `release-a2-68a9bec.log` or `validate113-ba6f286.log`; the sentinel verdicts, the
  ceiling-spread intervals and every term behind them were under gitignored `build/` only. The 2
  `gate-p3-under-p4` logs and their 28 samples are now tracked in `archive/pinned8/`. Re-derived
  from them, both published shares reproduce exactly — 99.52/207.75 = 47.9% at `68a9bec`,
  89.81/207.95 = 43.2% at `ba6f286` — and the artifacts then say something no percentage in either
  log could: **all three `BenchmarkPeak` medians agree to within 0.16% while four of five
  `BenchmarkKernel` medians move 5-12%**, the largest on a *scalar* shape that shares no code with
  the vector path. The instrument was constant across the pair and is now shown to be: identical
  `keel-pin:` mask (`nodedoms=1`, so rule 5's spread amendment is a no-op on that host), identical
  governor, `gomaxprocs` and dispatched shape, and the same `go1.27.0-X:simd` builder in both gate
  logs — `remote.sh` cross-compiles `keel.test` locally and `scp`s a static binary, so a host's own
  toolchain cannot enter a measurement at all. The one field that moved is the `clock-mhz` *snapshot*
  max, 3799 → 3701. Reported on #141, where it sharpens the attribution rather than settling it.
  Corrected in the same commit: `archive/pinned8/README.md` claimed the 13 `afb108e` samples "are
  spread-form and say so" — 9 are, and the other 4 say the opposite by carrying no `keel-pin:` line.
- **The apparatus ratio counts shell by content, not by file extension** (`gate-docs.sh`'s new
  `shell_files`: the `'*.sh'` glob unioned with a shebang sweep of every tracked-or-untracked
  file). Ruled 2026-08-31: a ledger that *"counts a 189-line test while its 97-line subject is
  invisible"* is *"rule 22's surface-form error turned on the ledger's own definition."* The
  subject is `scripts/fakessh`, 97 lines of bash with no extension, and the whole tree was swept
  in the same pass so the restatement happens once — it is the only such file; the other three
  non-`.sh` files under `scripts/` are TSV data. Shell term 16082 → **16195**, ratio 1.79x →
  1.81x, of which **+97 is a definition correction and +16 a spend**, booked as two rows in
  `docs/apparatus-ledger.md` because one number asserting a 113-line session would be false by 97.
  Controlled rather than assumed: a planted `#!/bin/bash -eu` is counted, `#!/usr/bin/env python3`
  is not, a file with no shebang is not, and no member the glob held was lost. The positive
  control's first reading was a **false miss** — `shell_files | grep -q` returns 141, because
  `grep -q` exits on its first match and `pipefail` reports the producer's SIGPIPE, so the arm
  asserted against a captured listing instead.

### Fixed
- **`ab.sh` named both arms' comparison files by commit SHA alone, so a null A/B compared one file
  with itself and printed a wash by construction** (`ab.sh`, found while building `drift`). With
  `BASE_REF` resolving to `HEAD` both arms produce the same short SHA, the second `cp` overwrote the
  first, and `benchstat` was handed one path twice. **Measured, on synthetic logs with a planted
  +10% delta:** the colliding names print `1.099µ ± 0%` against `1.099µ ± 0%`, `~ (p=1.000 n=10)` —
  and it prints a `vs base` column, which is the token `bench_compare` checks for, so the harness
  would have *reported success* and the run would have answered "no drift" whatever the host did. The
  arm is now in the file name (`base-<sha>` / `new-<sha>`), which makes the degenerate case work by
  construction rather than by a special case and labels benchstat's columns by arm in every run; the
  same pair then renders `998.8n` against `1099.2n`, `+10.06% (p=0.000 n=10)`. This is the defect
  class the whole `drift` preset exists to rule on, sitting inside the instrument that was to rule on
  it — a false PASS available only to the one measurement that had never been taken.
- **Two of #141's five kernel "movers" are not resolved, and the structural claim resting on the
  scalar arm is withdrawn** (`#141`, corrected against the samples tracked at `f84a9ac`). The
  published decomposition — *"four of five `BenchmarkKernel` medians moving 5–12% in both
  directions, the largest mover a scalar shape sharing no code with the vector path"* — is true of
  the **medians** and overstates what the intervals support. Recomputed with both runs' own
  confidence intervals: `2x32/avx512` −9.76% is disjoint by 9.2 GFLOP/s and `6x32/avx512` +2.59% by
  0.49, both comfortable; `4x32/avx512` −5.19% is disjoint by **0.01** and `2x32/scalar` −12.25% by
  **0.0014** on a value near 1, i.e. touching; and **`4x32/scalar` −5.86% overlaps and is not
  resolved at all**. So the two solidly resolved movers are both AVX-512 kernels, and the inference
  *"whatever moved is not in `internal/vec`, established without reference to the source diff"* no
  longer has the scalar evidence it rested on. **What replaces it is stronger and needs no cross-run
  median comparison:** between the two runs the *within-run* interval on every memory-touching kernel
  widened — `4x32/avx512` ×51 (0.105% → 5.378%), `2x32/avx512` ×8.8, `6x32/avx512` ×8.0,
  `4x32/scalar` ×1.95 — while the three register-only `BenchmarkPeak` arms held at ×1.00, ×2.00 and
  ×1.66 and stayed under 0.136% in absolute terms. The excursion is an **imprecision** in memory-touching
  measurement, not a shift, and that is mechanically the ceiling-spread width going 0.003 → 0.099,
  since the spread is computed from those intervals. **The memory/register split is a correlation and
  not an attribution, by #141's own checklist**: *"a timing arm cannot distinguish these"* — a claim
  that the memory system is the mechanism needs #53's L1d-miss counts, and every arm here is a timing
  arm. What is established is which rows widened, grouped by a property they happen to share.
  `2x32/scalar` is uninformative in either
  direction: it printed ±21.9% and ±6.9% in the two runs, with a 4.3× min-to-max range inside one of
  them.
- **#141 answered on the host that raised it: the widening was a transient excursion, and the
  certificate's sentinel reading is the reproducible one** (`#141`, `archive/pinned8/drift-janus-df999da*.log`,
  `scripts/ab-bench.sh drift` on `janus.local` at `df999da`, `n=30` per arm, `-benchtime=1s`,
  `GOMAXPROCS=1`, `keel-pin width=8`, `governor=performance`, `go1.27.0-X:simd`; **no OpenBLAS
  reference on this host**, so every figure is percent-of-measured-peak or a within-host delta).
  The three calibrated `kc=128` AVX-512 bands read **0.066/0.085/0.091%** and **0.145/0.095/0.147%**
  in the two arms, against `ba6f286`'s 5.378/1.681/1.779% — the widest here is 11.4× narrower than the
  narrowest there, and narrower than the pre-excursion run on two of three rows. Nothing landed in the
  0.30–1.00% band that was **declared ambiguous before the run**. The medians agree independently:
  +0.36/+0.39/+0.09% against the pre-excursion run where the excursion was −5.19/−9.76/+2.59%. So the
  sentinel share — the ratio whose two terms this session put in the gate log — reads **47.94%** and
  **47.83%** here against **47.90%** in the certificate, and the `43.2%` that failed P2's floor was
  the outlier.
  **Three things the run found that the fork did not ask for.** (1) *The arms are not one binary.*
  `remote_build_test` passes no `-trimpath` and the base arm builds inside a worktree at a different
  path, so the two binaries differ (5,089,864 vs 5,089,752 bytes, different sha256) — and this repo's
  own between-binary layout floor is 1.71/0.99/1.32%, the same order as the deltas. Settled by symbol
  address rather than assumed: all 8 `internal/vec` text symbols sit at byte-identical addresses in
  both arms, and of 3635 text symbols the 196 that moved are **all** `go.shape` generic instantiations
  in `internal/sync`, `slices`, `sync` and `runtime` — no keel symbol among them, no symbol in one
  binary only. (2) *The host is not quiet between windows even so.* Two arms of one build, kernels at
  identical addresses, minutes apart: `2x32/avx512/kc=32` −3.08%, `2x32/avx512/kc=512` −2.30%,
  `6x32/avx512/kc=8` −2.04%, all `p=0.000`, and `2x32/scalar/kc=32` **+15.60%** at `p=0.000` with
  ±0.158% and ±0.109% bands. Opposite signs at `p=0.000` is a **direction split**, which is precisely
  the case criterion 5b's new coverage line says a roofline cannot absorb. (3) *The pre-registration
  was too coarse and the defect is ours:* the predicate was written over `Kernel/*/avx512` while its
  thresholds were calibrated at `kc=128` alone, and the driver's filter expands that to four `kc`
  values per shape — so the fork read literally returns the other branch on rows no threshold ever
  covered. Rule 24 one level down: the output *space* was right, the row *granularity* was not.

### Added
- **`docs/apparatus-ledger.md` — one running total for the apparatus debt, itemized per commit**,
  and `gate-docs.sh` now prints how to read its own ratio. Ruled 2026-08-31: *"'booked separately'
  would create a second ledger, and two ledgers is how a number gets to be true in each and wrong
  in sum."* Until now the total lived only in #131's comment thread, which has no single current
  value — only a series of claims about one. Every delta in the table is **measured from the
  commit** (`git show --numstat --format= <rev> -- '*.sh'`) rather than quoted, which is the
  property a thread does not have; that already corrected one line, `891db4d`, which the
  reconstruction had folded into `75aae82`'s +292 and which measures **+0**. Current debt **+511**.
  The standing reading rule beside the figure: **the absolute shell term is the session-delta
  reading, the ratio is the headline disclosure only.** Both terms are five digits, so `+78` lines
  and `+137` lines each printed as one hundredth (1.77x → 1.78x → 1.79x) with the library term
  measured constant at 8964 across all five revisions — two printed decimals cannot separate two
  spends differing by 59 lines, and a flat ratio is not a flat ledger.
- **`scripts/detach-test.sh` — `detach.sh`'s incident log, made executable** (161 lines, wired
  into `make lint`). The harness that launches every long run had produced three behavioural
  incidents in a week and had no test at all; one of them idled a three-host fleet for eight
  hours, so its failures are denominated in dollars rather than lines. Each case runs **twice**:
  against a mutant copy with the fixing line reverted, where the arm must **reproduce the
  incident**, and against the shipped script, where it must not. A harness test that has never
  failed proves only that the harness cannot be tested — and this one earned its form, because
  both fail-first arms caught a real defect before the file certified anything (a sed mutation
  that silently matched nothing, and `set -o pipefail` turning `stat`'s by-design exit 1 into a
  false NOMATCH from a `grep -q` that had in fact matched; every assertion now matches a captured
  string, never a pipeline's status).
  Covered, all three from the log: the **dropped override** (`:117`), the **injected override**
  (`:116`, the eight-hour fleet), and **one word for two facts** (#122, asserted as
  distinguishability plus the prefix hint). Plus a positive control that `stat` still reports a
  finished run's exit code, and a scope guard that `remote.sh`'s separate `REMOTE_STATE=vanished`
  was not renamed along the way.
  **Not covered, stated rather than implied:** the fourth incident of that week — a 45-second-old
  log read as a seven-hour stall — was a defect in an ad-hoc *probe* of a detached run, not in
  this script, which has no mtime, age or timezone logic at all (the same grep finds such logic
  in three sibling scripts, so the zero is a reading). Nothing in `detach.sh` to drive, so no arm.
  Without tmux the file prints `SKIP` and exits **0**, naming that none of the arms ran — a
  tmux-less machine should not fail `make lint`, and a skip that reads like a pass is the thing
  this file exists against. Both branches driven on purpose.
  **Every arm runs on a private tmux server** (`TMUX_TMPDIR` under the test's own temp dir, which
  moves the socket for this shell *and* for the `detach.sh` children it spawns), per #122's
  instruction that the operator's server be neither read nor perturbed. This was not cosmetic: the
  first version set a global variable on the shared server, and setting one there while a gate run
  is live reaches every session that server starts afterwards. Asserted rather than assumed, and
  driven red on purpose — and the assertion's own first reading was a false leak, because macOS
  `mktemp -d` yields `/var/folders/…` while tmux reports the resolved `/private/var/folders/…`.
  **Not covered, from #122's own list rather than from the incident log:** that the stated
  exceptions `PATH`/`GOEXPERIMENT`/`GOMAXPROCS` survive the namespace clear. It has never failed,
  the only-what-happened scope excludes it, and a regression there breaks every run loudly.
- **gate-p5 criterion 9 now checks the README's denominator column, not just that it is
  non-empty** (#100, arm B). Each host publishes 9 rows whose 9 denominator sentences reduce to
  **one** measured quantity — the 1-thread avx512 peak, or that peak times the thread count — so
  the arm requires all 1-thread rows to name one value, all 8-thread rows to name one value,
  those two to satisfy the peak-times-cores identity within the printed figures' own resolution,
  and the published peak to sit within `README_TOL` of the `Peak/avx512` this run measured.
  Rows at any other thread count are **counted and named in the verdict** rather than refused:
  a future row at another width would be correct and this arm has no identity for it (§5 rule
  12). An unparseable denominator cell is a refusal, never a fall-through.
  **The identity is computed in integers, and the instrument is why.** #100 predicted that `≤`
  on floats would suffice for the boundary pair; the fixture refuted that prediction *before it
  shipped*. `8 × 192.6` is `1540.79999999999995` in doubles, so Intel's correct `192.6 / 1541.2`
  pair reads a delta of `0.40000000000009095` against a band of `0.40000000000000002` — **the
  row that motivated `≤` fails under `≤`**. Both figures are decimals the README printed, so
  scaling each to units-in-its-last-place decides it exactly instead of picking an epsilon.
  **Arm A (`P == 100·R/D`) is refused, not deferred**, and recorded as considered: its band is
  derived from how many digits each figure happens to be printed to and ranges 0.05→0.868 points
  across the block, so on its widest row it cannot see a share mistyped by half a point — #110's
  class. Arm C (the tail template) is unlanded and unclaimed. The residual after B is the
  `N=2 archives` count, a claim about the archive set rather than about this run.

### Fixed
- **`detach.sh stat` said `vanished ... killed or never started`, naming both causes at once**
  (#122). They call for opposite next actions — a run that started and was killed has a log to
  read and host-minutes already spent; a run that never started has nothing, and the usual cause
  is a NAME that does not match what was launched. Now two words: **`died`** (a log exists) and
  **`never-started`** (no log at all), both still exiting 1, because a run without a status file
  is unmeasured either way and an exit code is exactly what neither has. `never-started` names
  the doubled-prefix cause outright, since `sess_of` prefixes `keel-` unconditionally and
  querying `validate113-ba6f286` as `keel-validate113-ba6f286` reported the old line for a run
  that was healthy and 25 minutes in. **The prefix is hinted at, never stripped**: `keel-foo` is
  a legal NAME and a `stat` that rewrote or rejected one would break a run merely named that way,
  so the fix for the false alarm is a sentence and not a transformation — a case the fixture
  pins by running a legitimately `keel-`-prefixed job end to end. `remote.sh`'s
  `REMOTE_STATE=vanished` is a **different mechanism** with five gate consumers and its own test
  suite, and keeps its word untouched. Also fixed in passing: `wc -l` renders with leading
  padding on macOS, so both this line and the pre-existing `running` line printed
  `(       1 lines so far)`.
- **gate-p5's 1-thread avx512 peak was per-host state living in the per-routine loop, past both
  of that loop's `continue`s** (`scripts/gate-p5.sh`). A host whose every routine bailed carried
  the *previous* host's peak. **Latent on the judged fleet** — the only reader sat in the
  iteration that assigned it, so no rendered verdict or log line changes — but it is a
  false-PASS mechanism the moment anything after the loop reads it, and #100's Arm B is exactly
  that reader. Hoisted to host scope beside `CEIL8`/`CEIL1`, whose own comment already made the
  argument the peak had escaped: it is a property of the machine. Non-empty by construction —
  `require_bench` demands the row under GFLOP/s via the same `bench_stat` call, and renders
  `unmeasured` naming the cause when it is absent. The `-n` test went with it, so an empty peak
  would now render `awk: division by zero` instead of skipping a line silently; **deliberately
  not replaced by a `-z` guard**, because the only degenerate shape `bench_stat` can produce
  past `require_bench` is a field shift printing the CI where the median belongs, which is
  non-empty and wrong — a guard for the impossible case that misses the possible one.
- **v0.1.0's certificate log is now tracked, and no repository revision had ever held a green
  `gate-p5`** (`archive/pinned8/release-a2-68a9bec.log`, sha256
  `e8c5889a081d2e08dbe5f43f4605e2693ce650b75fd465015959f38ed7baaa1e`, 947,745 bytes). The
  release notes cite the certificate as `build/release-a2-68a9bec.log`; `build/` is gitignored,
  and the three logs in `archive/pinned8/` are all RED — so the tag's own certificate resolved
  on one laptop and nowhere else. #114's law at the top of the stack. **Identity is
  corroborated, not proven, and the notes published no digest to check against**: the witnesses
  are the log's own rev stamp `68a9bec`, its tally matching the notes verbatim, and an mtime
  (`2026-08-29T21:49:57Z`) that predates the tag by 70 minutes. Tracking freezes the bytes from
  now on; it cannot show they are the bytes that were cited, which is why CONTRIBUTING's tag
  checklist now requires the digest *in the notes* and not only a tracked file. #113's
  validating run archives beside it (`validate113-ba6f286.log`, sha256
  `24a4ae860eb0c5b9ae97bc2134cc5ee67b4ca18c35c78316d73b9f5a81587f23`). Found while building a
  verdict-set diff and discovering the comparison basis was not in the repository. **Same class,
  not fixed here because its home needs a ruling:** CONTRIBUTING names
  `build/release-notes-vX.Y.Z.md` as an intact copy of the tag message and that file is
  gitignored too.

### Added
- **`keel-bench-toolchain` in the provenance preamble** (`bench/bench_test.go`, feeding `#118`).
  The compiler is part of the instrument and no bench artifact recorded it: of the 45 bench
  artifacts under `archive/pinned8/` not one names a toolchain, so "was this row taken under
  go1.26.x or go1.27.0" — the question `#118` exists to rule on — was not answerable from the rows
  themselves. `runtime.Version()` also carries the GOEXPERIMENT, verified by making the value move
  rather than by reading it once: `go1.27.0-X:simd` under `GOEXPERIMENT=simd` and `go1.27.0`
  without, on a host where `keel-bench-backend` says `scalar` either way and so could not have
  distinguished them. Deliberately **not** added to `KEEL_BENCH_IGNORE`, so a cross-toolchain A/B
  now fails closed through `bench_compare` naming the key that forked benchstat's table (`#50`,
  T20); nothing in the tree compares across toolchains today, since every arm-building path sets
  `GOEXPERIMENT=simd`. This records provenance and settles nothing — whether go1.27.0 opens an era
  in `scripts/measurement-eras.tsv` is still `#118`'s ruling to make.

### Changed
- **The ceiling every judged share divides by is a published, re-measured row** (`#113`).
  `CEIL_FRACTION` judges each host's routines against that host's own measured 8-thread ceiling,
  and those rates lived in README's *caption* — outside the region `gate-p5` criterion 9
  re-measures, the exact defect `0bbf964` was reverted for. The blocker was criterion 9's row
  checker hardwiring `scale_name`, so `Ceiling/compute` resolved to
  `Scale/Ceiling/compute/n=4096/threads=8`, matched nothing, and was reported as "this gate
  measured no such row" — a true refusal about the wrong thing. A `row_name` resolver keyed on the
  published benchmark column fixes it, and yields EMPTY for a path-like name no family addresses,
  because unaddressable and unmeasured are different facts about a published row. README grew
  three rows (24 → 27, one per host: 1444, 2292 and 933.8 GFLOP/s) and the caption's
  "deliberately not republished here" sentence is now the opposite claim. Verified before landing
  by resolving each published row through criterion 9's own `bench_gflops` against that host's
  archived bench log — 27/27 matched inside `README_TOL`, all three ceiling rows included — with
  a drifted row and an unaddressable row each shown to be refused, so the green distinguishes
  checked from skipped. `#113`'s own premise that "the twelve archived logs carry no
  measured-ceiling row at all" measures **false**: 2709 `Ceiling/compute` lines are in the
  archive and every archived revision's gate already computed `CEIL8` from a measured ceiling,
  not a proxy — which is why this needed no fleet run to publish, and which reopens whether that
  issue's permanent-limitation clause holds for the reason it states.
- **Six gates stopped carrying the fleet preamble, four stopped carrying the probe block, and
  two stopped carrying P2's compile-time audit** (`#131`). `resolve_fleet` (`remote.sh`) replaces
  six byte-identical copies — one md5 across `gate-p0`..`gate-p5`; `probe_or_unmeasured` replaces
  four verbatim copies in which the probe's value never outlived the block, so nothing is
  exported; `carry_p2_properties` (`gate-lib.sh`) replaces `gate-p3`'s and `gate-p4`'s copies of
  the spill/call/BCE audit, which differed only in the phase name and the loop's description —
  two arguments, not a mode flag. Each lift was proven byte-equivalent to the code it replaced
  before landing: 8 scenarios for `carry_p2_properties` (both callers × each failure branch, with
  `go` stubbed to echo the argv so a changed command line shows as a diff), plus both branches of
  the probe inside a real loop to prove `|| continue` still skips the body. Every comparison
  carries a control that must fail. `gate-p2`'s ancestral copy of the audit is deliberately not
  folded in: its wording and ordering differ, and a divergent copy is a finding to settle rather
  than a caller to convert. **Refused one lift and recording it**: the four gates' governor
  preamble loops differ in body — `gate-p2`/`gate-p3` call `admission_readback`, `gate-p1`/`p4`
  do not, and `gate-p4` benchmarks too — so the distinction is arbitrary and sharing it would
  need a flag telling the function which caller it is. Ledger, `gate-docs.sh`'s own expressions:
  `shell 15854 / library 8964 / 1.77x` at the start of the session, `shell 15842 / ... / 1.77x`
  at the end, both 2026-08-31. Net `scripts/` **−12**, which discharges the `+11` shelf; the
  ratio does not move at two decimals on 12 lines, so the counts are the reading, not the ratio.
- **Both of CL 824624's falsifiers re-measured, and the mechanism finding given a second
  derivation** (`#127`). The watch protocol's rule 5 makes every number in a review reply pass
  what a CL description passes, so the two figures in `FMAPrefers231`'s doc comment could not be
  quoted as they stood — they were measured when the CL was written. Four `FMAPrefers231` bodies,
  `spill-audit` on `internal/vec`, the log tracked as `docs/cl1-falsifiers-fcc5822.log` because
  `build/` is gitignored and these figures are bound for a public reply: unconditional rewriting
  takes the Horner shape 27→**53** insns and 0→**26** copies, its AVX2 twin 23→44 and 0→21; the
  narrow "closes the phi" test leaves **21 of 26** copies on `Kernel6x32`, whose copies run
  45 → 40 → 19 across no-rewrite, narrow and as-mailed. 14 predictions stated before the run, 14
  confirmed. Arm A was measured first *and* last and came back `diff`-identical, which is what
  caught that the pre-built compiler was stamped at a prior amend of the same CL — with a
  byte-identical tree, so the reading held for a reason no mtime shows. The `commuted Op`
  objection now also has a wording-free form: 213 computes `arg0*arg1 + arg2`, symmetric in the
  two arguments that field swaps, so no chain of them moves `arg2`. Patch set 1 needs no
  correction and was not amended; the reply built on these figures went out the same day, below.
- **`docs/upstream-plan.md` records CL 824624's first review** (`#127`). The trybots passed
  (`LUCI-TryBot-Result+1` on revision `fcc582225c`); Jorropo left one unresolved comment saying
  operand-form selection is regalloc's job and pointing at his commuted-regalloc series. The
  architectural point is conceded in the record; the specific mechanism is not available for this
  instruction, and the reason is the field's own comment — `778820`'s `commuted Op` commutes "the
  first two arguments", where `(VFMADD213PS x y z) => (VFMADD231PS z x y)` is a three-cycle,
  because the forms differ in which of three operands is the destination. The decision among
  defend / split / rebase-on-an-unvoted-stack was Scott's; he ruled *defend*, and the reply is now
  sent — see below. Patch set 1 was never amended.
- **CL 824624's review reply is sent, and the CL-number bridge was its send gate** (`#127`).
  Threaded under Jorropo's objection at 21:46:11Z, left **unresolved** because the architectural
  point is not settled and is the objector's to resolve; confirmed from Gerrit's `/detail` and
  `/comments` rather than the POST body — messages 8→9, comments 1→2. The split was rejected on
  the draft's own caveat, that a half defining six ops no rule emits is unreachable code. The gate
  caught a real miscue: Jorropo named "CL 778460" and the draft cited `778820` twice. Verified via
  `/detail` on both plus `/revisions/current/related` — `778820` is the parent `778460` sits on
  and the only member touching `ssa/op.go` — so both sites now name his number first and the
  member second. Calibration measured, not assumed: `golang/build`'s owners table lists Jorropo a
  **secondary owner of `cmd/compile/internal/ssa`**, 164 commits in `src/cmd/compile`, which is
  what makes deference-on-architecture-firmness-on-mechanism the right posture rather than a
  guess. One sentence was killed before sending — an unprompted offer to hold CL 1 behind his
  series, a schedule concession no one requested.
- **`scripts/ab.sh` — one A/B harness, lifted out of `l1-bench.sh` and `edge-bench.sh`** (`#131`
  paydown). The two drivers were identical for the entire skeleton — worktree, trap,
  cross-compile, methodology preamble, host loop, comparison — and differed in four strings, so
  each is a parameter (`AB_TITLE`, `AB_FILTER`, `AB_TAG`, plus optional `AB_NOTES` and
  `AB_BASE_HINT`) and none is a mode flag: nothing in `ab.sh` asks which caller it is serving,
  which is `#131`'s criterion 6. Its own file rather than more of `remote.sh`, because every gate
  sources `remote.sh` and an A/B driver's machinery has no business inside the closure a release
  certificate transfers across. `remote.sh` also gained `warn`, `remote_require_hosts` and
  `remote_host_header`, ending four dead copies of `info`/`warn` (each already overridden by
  `remote.sh`'s at every call site) and three copies each of the empty-fleet guard and the
  per-host banner. **`l1-bench.sh` now archives its logs under `build/l1-<host>-<sha>.log`**,
  which it never did — the lift gave both callers `edge-bench.sh`'s §5.8 behaviour. Neither
  driver had a test, so the lift is verified by a 39-assertion exercise that executes both real
  files end to end with the transport stubbed, driving every failure arm: a failed build arm with
  and without the fixture hint, a failed run arm, an uncompared host, a silent host beside a live
  one, an empty fleet, and a caller that sets neither optional variable. `shellcheck -x -S style`
  over `scripts/` falls 113 → 107 findings, itemised: −4 `SC2329` (the dead `info` definitions it
  had been reporting all along, unread because nothing runs it) and −4/+2 `SC2064` (the
  deliberate trap expansion, now once instead of twice).
- **The stray-toolchain exposure is in the module cache, not `~/sdk`** (`#134` inventory, feeding
  `#121`; `docs/hosts.md`). Probed read-only on all three lab hosts 2026-08-31: `go` resolves to
  `/usr/local/go/bin/go` at go1.27.0 on every one, so `#134`'s provenance criterion discharges
  negative. `janus`'s `~/sdk` toolchains have **no `go1.X` shims** — the doc said they did — so only
  an absolute path reaches them; meanwhile `GOTOOLCHAIN=auto` everywhere selects out of
  `$GOMODCACHE/golang.org/toolchain@*`, where `janus` holds five, four below the go1.27 floor. Inert
  for keel (`go.mod` asks `go 1.26`, installed is 1.27.0) and inert for the fleet either way, since
  `remote.sh:498` cross-compiles every artifact on the dev host. The `~/sdk` purge is unrun: the
  recursive remote delete was refused by the sandbox and needs Scott. Also: the probe's first
  liveness check matched its own command line and reported all three hosts busy — a self-matching
  `pgrep`, re-run with the pattern built at runtime under an `sshd` positive control.
- **`RANK-WINDOW-BLIND` is deleted; every reading now prints its disparity `D` instead** (`#132`,
  ruled by Scott 2026-08-31; `scripts/bench.sh`). `bench_describe` closes each reading that
  carries a range with `D=<v> (span <s>% / interval <i>%)` where `D = (max−min)/(hi−lo)` — how
  many interval widths of sample disagreement the reading holds — and the marker, `QUANTUM` and
  the conjunction between them are gone. **The history is the argument**: every thresholded form
  of this detector had a false-negative band (exact-zero missed 12 readings, printed-zero missed
  the row that was the denominator of nine verdict lines), and the trigger's verdict was not even
  monotone in the blindness it claimed to detect — in one archive file, one host, one sweep, all
  in `sec/op`, it named `D=142.6` and stayed silent on `D=180`. `D` reads off the **real
  asymmetric** bounds, never the printed `± W%`, which is the mirrored symmetric half-width and a
  different quantity; both operands therefore print beside it. Output-only (§5 rule 15): no
  criterion reads `D`, so no verdict can move. The four documented instances are worked through
  as reading examples in `docs/rulings.md` rule 20, three of them pinned in
  `tools/benchci/archive_test.go`, and §9g's keel-skx fixtures were rewritten to their
  **re-derived** readings because an operand may not be invented. Coverage limit stated rather
  than papered over: on a **rate** unit `D` is quantised by the samples themselves — Go renders a
  `ReportMetric` rate at 4 significant figures — so the `sec/op` reading of an arm is the one to
  trust when the two disagree.
- **The apparatus ledger's own ratio was flattering itself; `tools/*_test.go` is folded in**
  (`#131`, ruled by Scott 2026-08-31; `scripts/gate-docs.sh`). The reporter excluded
  `tools/*_test.go` from the `tools/` term while including `bench/`'s tests in `bench/`, so 769
  lines were counted in **neither** term — a disclosure metric flattering by omission, in the one
  artifact whose entire value is that it doesn't. `tools/` now moves whole, as `bench/` has since
  2026-08-20, and the reported apparatus ratio restates **2.70x → 2.80x** on one tree (2.69x →
  2.79x when the hole was found on 2026-08-30; both ends have moved with the tree since, which is
  why the line now *computes* the pre-fold ratio rather than quoting it). This is a definition
  correction, not a regression, and the definition sentence prints beside the number: apparatus =
  shell + `tools/` + `bench/` **including their tests**, library = `*.go` net of its tests. The
  asymmetry is deliberate — the ratio measures maintenance burden against shipped substance, and
  a test of an instrument is burden — and it is only defensible stated. `internal/spill`'s audit
  instrument stays on the library side, still disclosed and still not moved.
- **`STRSM_FLOOR` is re-typed `6.067x` → `6.066x`, which is 0.001x MORE LENIENT** (`#119`,
  ruled by Scott 2026-08-31; `scripts/gate-p5.sh`, `scripts/readme-numbers.sh`, and the caption
  `readme-numbers.sh` generates into `README.md`). **This is an instrument-precision
  re-derivation and not a re-measurement**: no sample, host or sweep moved. `bench_ratio_lo`
  has rounded *down* since `#143`, so the argmin's bound is `6.469` and the formula's output on
  its own inputs is `6.469 - 0.403 = 6.066` exactly, where the shipped `6.067` was
  `6.4699 - 0.403 = 6.0669` rounded to *nearest* — `#143`'s cured display path fossilised in a
  constant. `#119` found the delta, disclosed it, and let the constant stand as the safe
  direction; that disposition is **VOID as to its conclusion**, the grounds being that a
  criterion constant which disagrees with its own formula on its own inputs is a fossil, and a
  bar's authority is its derivation rather than its publication history.
  **Re-derived, not accepted on the ruling's word**: the three take-four rows through
  `tools/benchci` + `bench_ratio_lo` give `lo` = 6.831136 / 6.635663 / 6.469880 for
  keel-skx / keel-zen4 / keel-zen5, reproducing the four-decimal figures the gate publishes.
  **Three witnesses that it flips no verdict** — a verdict can move only for a bound in
  `[6.066, 6.067)` — 33 rows recomputed from both eras' raw archives (0 in band, 47 logs with
  no bounded ratio counted as skipped rather than as clean); 184 rendered bounds over the 62
  logs that print one (0 in band, nearest 6.017 below and 6.094 above, so the band is bracketed
  on both sides); and `readme-numbers.sh` re-run under the new constant still printing
  *"9 of the 12 routine-host pairs … clear the bars"*, byte-identical but for the constant it
  quotes. Witnesses 1 and 2 were positive-controlled against a planted in-band 6.066.
  **One figure corrected in passing, same cause**: the three rule-19 interval widths published
  as `0.229/0.220/0.040x` are `0.230/0.221/0.040x` as the gate's own rule-19 sentence renders
  them (full precision 0.229233/0.220796/0.039598), so `0.220` matched no path at all — the
  corrections move in the stricter direction and all three rows stay admissible against the
  0.403x cap. `docs/rulings.md` carries the ruling, the derivation, all three witnesses and
  their controls; `DESIGN.md` §4's pre-registered sensitivities are re-derived to **>6.24%**
  and **>6.66%** by the arithmetic that reproduces the 6.23/6.64 pair it published against
  6.067.

### Fixed
- **The era guard's loophole argument was resting on a count that had gone false**
  (`scripts/measurement-eras.tsv`, feeding `#118`). Its paragraph justified `pinned8`'s two
  re-foundings as costless because `judged-runs.tsv` had no data rows and `host-baselines.tsv` was
  header-only, so no host had an exemption to return. True on 2026-08-22 when the last re-founding
  landed; false since 2026-08-28, when keel-skx registered `share/Sgemm`, `share/Ssyrk` and
  `share/Ssymm` in that era. The sentence is now dated as-of, with the change and its consequence
  stated: a *further* re-founding of `pinned8` would return a live exemption and has to earn the
  guard's two conditions on their own merits rather than on emptiness. Which is exactly what
  `#118` is asking for, so the argument had to be repaired before the ruling could be priced.
- **The live BASELINE-REGISTERED exercise refused a gate that was behaving correctly, and its
  own derivation-set guard was checking one of the two sets it printed `ok` for (`#119`).**
  First firing on antares.local found both. (1) The share criterion had no rule-19 escape
  hatch: the driver expected `$NJ` BASELINE lines, "one per judged routine", where two of three
  share readings had rendered `NOISE-LIMITED, NOT JUDGED` — Ssyrk's 8-thread interval is
  `±33.6%`, costing that share 14.70 points against a 2.6-point margin. The ratio criterion has
  had that hatch since 2026-08-22; the share one was a hardcoded count, and it would have
  failed all three passes. `swant` is replaced by `preempted CRIT N` and `want_n KIND N`,
  per routine and per pass for both criteria, keyed to each criterion's own refusal sentence
  (the shared prefix matches both, measured 1/1/2 against the gate's bytes; disjoint
  `P5_JUDGED`/`P5_MEASURED` is what makes the short key harmless *today*). Pass 3's skip arm is
  lifted out of the ratio branch and now keys to pass 1's candidates file rather than a latch.
  A computed expectation can reach zero, so a `wantc == 0` refusal is added: zero-equals-zero
  is a green over nothing measured. (2) `${!l}` on `CEIL_DERIVED_FROM` — a name never assigned,
  the value being in `DERIVED_FROM` — dies inside a process substitution whose status nothing
  checks, so `set -u` skipped the CEIL arm and the `ok` line asserting the host was outside
  **both** sets printed anyway. The arm's loss cost nothing, and by containment rather than
  design: `CEIL_DERIVED_FROM` is a subset of `SCALE_DERIVED_FROM` today. Direct `NAME=VALUE`
  expansion in the `for` list now aborts the driver instead. Verified against the failing run's
  own archived artifacts: `want_n share 1` returns 1 where the hardcoded expectation was 3, and
  2 == the 2 candidate rows that landed. All three new arms driven red and back; 66 fixtures
  green; `+43` net lines to `scripts/`, disclosed, joining `#131`. **The three-pass exercise
  then completed for the first time** — 'new', 'owing' and 'registered' all YES, 417 verdict
  lines across 9 logs all `[synthetic]` and none signed — and it needed the fix at pass 3, not
  at pass 1: `share/Ssyrk` was quiet enough to be judged on pass 1 and noise-limited on pass 3,
  where the lifted skip arm reported it `SKIPPED and unexercised` where the old code had only
  "NO line naming a registered baseline at all". Ssyrk was noise-limited on the previous run's
  sweep, quiet on this one's pass 1, noisy again on pass 3 — three sweeps of one host, so
  noise-limiting is a property of the sweep and per-pass-unlatched is observed rather than
  argued. Era scoping holds in both directions on the same lookup (`39.3% = 41.9 - 2.6`,
  `37.3% = 39.9 - 2.6`, `5.379x = 5.782x - 0.403`, all era `pinned8`, the wrong-era decoy at
  99.0 never consulted), and the registered arm is exercised on 3 of 4 criteria with the fourth
  named.
- **A suspended fleet bar could excuse a host that had a registered bar of its own (`#119`).**
  The share criterion's last guard read `[[ -z "$CEIL_FRACTION" ]]`, testing the *fleet
  constant* after `$BBAR` had already been resolved to this host's own baseline less the
  margin — so had the fleet fraction ever been suspended again, a registered host would have
  been passed as "no fraction in force" against a bar sitting two lines above. Both criteria now
  test the **resolved** bar. Unreachable today (44.2 is typed) and strictly stricter when
  reached, so it is checked in a fixture rather than asserted.
- **The rounding band `#143` is about is real, one-sided and permissive — and it is not at
  the site the issue names.** `$ratio` already arrives from `bench_ratio_lo` at `%.3f`, so
  gate-p5's `frac="$(printf %.1f, 100*$ratio)"` is *lossless* and the prescribed fix moves no
  verdict; driven, a true share of `44.150060%` still cleared the `44.2` bar. The band lives
  in `bench.sh`'s `printf "%.3f"`, which rounds to **nearest** and so can return a value
  above the bound it computed. `bench_ratio_lo` and `p3_ratio_lo` now round **down** and
  `bench_ratio_hi` **up**, which is `#116`'s own standard (a bound must bound something
  measured) and makes `lo >= bar` decide exactly what the unrounded bound decides, since
  every bar here is on the 3-decimal lattice. `bench_ratio` is unchanged: a point estimate
  has no direction to be conservative in. **The archive is clear** — 539 comparable verdict
  lines over 272 archived logs, none inside the band, and the census is exact rather than
  approximate because a verdict is in band exactly when its rendered margin is 0. Tightest
  archived margins: **0.400 points** and **0.004 in ratio units**, both 8× the band (`#143`'s
  46× is the *certificate's* tightest margin, 2.3 points; over the whole archive it is 0.4).
  Seven-arm before/after control: the three in-band arms flip PASS→FAIL and the four that
  must not move stay PASS. **The arm `#143`'s own acceptance criterion could not supply:**
  its prescription — "compare the raw `$ratio`" — is a units error, since `$ratio` is a
  fraction and every bar is in points, and taken verbatim it FAILs *everything*. The
  specified control would have read that as success, because it too expects FAIL. The gate
  now compares `100*$ratio`. Two further findings recorded rather than fixed here:
  `roofline-test.sh`'s `checkr` tolerance is `0.0005`, exactly this band, so the existing
  fixture harness for the lower-bound function is structurally blind to the class; and
  `bench_ratio_grade`'s "pass condition unchanged, bit for bit" claim is now false by half a
  quantum, so it is restated as history rather than left standing.

### Added
- **Rule 20's archived path is exercised by CI, and it says the marker's selectivity is an
  accident of the display quantum (`#133`, reporting to `#132`).** The stated coverage limit —
  "no persisted CSV carries the seven columns" — was about persisted *CSVs*; every
  `archive/*/*.txt` holds the raw samples, so `tools/benchci` re-derives them and
  `tools/benchci/archive_test.go` re-drives the sweep under `go test ./...` (which CI runs and
  `remote-exec-test.sh` is not): **80 files, 1420 readings, 0 unbounded**, no interval bound
  escaping its own samples on any of them, the rank pairs pinned through
  `benchmath.AssumeNothing` itself, and the shipped `bench_describe` driven over a re-derived
  archived CSV with `RANK-WINDOW-BLIND` firing on purpose. **No trigger changed.** The corpus
  answers `#132` in the negative — the disparity ratio's achievable set is `[1, ∞]` at every
  bounded n, so no finite cutoff follows from the geometry (`D ≥ 1` on 1420 of 1420) — and
  finds the real seam instead: of 296 near-zero readings in the 18 per-host `pinned8` archives,
  78 are named and 218 silent, **interleaving** down the span ordering (15.93% named at `0.0%`
  against 15.74% silent at `0.1%`, adjacent arms of one file, the silent one carrying two
  recurring 15.5%-slow draws at ranks 29–30 of 30). Two `remote-exec-test.sh` fixture notes
  claimed a width printing `0.1%` "asserts nothing the range refutes"; correct as pins,
  false as statements, renamed to name the miss class.
- **§5 rule 17 now binds the ratio criterion too (`#119`).** gate-p5's `Strsm` scaling bar was
  one fleet constant judging every host that reported, including hosts outside its derivation
  set by chronology rather than by any property of their code — the exact asymmetry rule 17
  settled for the share criterion on 2026-08-21. The classifier is **lifted, not copied**:
  `baseline_state` in `gate-lib.sh` answers in one word (`nokey`/`conflict`/`fleet`/`registered`/
  `owing`/`new`) and both criteria call it **with no mode flag** — the derivation list is an
  *argument*, because the two criteria genuinely have different lists, and that is data rather
  than a mode. **The two lists differ by one model, and that is the finding:** `STRSM_FLOOR`'s
  6.067x was derived from three rows (keel-zen5/zen4/skx) where `CEIL_FRACTION`'s 44.2% was
  derived from two, so reusing the share criterion's list would have handed keel-skx a
  `BASELINE` for a bar it helped set — rule 17 run backwards. The margin is settled **by rule
  and then measured**: rule 17(c) requires the fleet bar's own constant, `STRSM_MARGIN=0.403`
  is already in ratio units and predates the era's rows by six days, and its adequacy was
  checked by recomputing all three archived take-four rows under the current instrument
  (`lo` = 6.831/6.635/6.469x, intervals 0.040/0.221/0.230x wide — the margin is **1.75× the
  widest**). `BASELINE_MARGIN`'s 2.6 is *points of share*: carried across units it would set a
  registered host's bar looser than the **retired** 6.0x floor. Stated inside the number (§5
  rule 12): 9 of this era's 15 archived `Strsm` rows are wider than 0.403x, so rule 19
  out-resolves them before bar selection and the class will register from a minority of its own
  rows; and the widest width *printed in a gate log* for this era, 0.929x, is a pre-`#116`
  rendering, so the census that produced it pooled across an instrument change — which is why
  the check was redone from the archives instead of from the logs. Byproduct: re-running
  `STRSM_FLOOR`'s formula under the post-`#143` instrument now types **6.066**, so the shipped
  6.067 is 0.001x *stricter* than its own formula yields — disclosed and deliberately not
  re-typed, since re-typing downward is loosening. Those same two 0.001 gaps positive-control
  `#143`. Derivation and both lists in `docs/rulings.md` rule 17; nine fixtures in
  `scripts/baseline-test.sh` (66 ok), one of them asserting the two *shipped* lists really do
  differ, negative-controlled against a copy with them unified. **No certificate-fleet verdict
  moves**, resolved against the real tracked registry rather than argued: all three fleet models
  render `fleet` on `scale/Strsm`, so the comparison is bar-for-bar what it was, keel-skx stays
  `registered` on its three `share/*` rows, and an `ARM Neoverse-V2` renders `new` on both
  criteria — which is the unblocking `#137` needs. That check also exposed a gap in
  `host-baselines.tsv`'s own header, now fixed: it excluded rows for "the models that derived
  `CEIL_FRACTION`" when the exclusion is per criterion, so the 8124M is legitimately
  registry-governed on `share/*` and fleet-governed on `scale/Strsm`, and adding the missing row
  *for consistency* would convict it of a conflict on a bar it helped derive.
- **`exercise-baseline.sh` drives both criteria of the class, and its own documented host was
  the wrong one (`#119`).** Every read-back is now keyed twice — `reaches`/points/2.6/`(#6)`
  for the share criterion, `scales`/x/0.403/`(#119)` for the ratio one — because the phrase
  `RECORDED as its candidate baseline rather than judged` had become common to both and would
  have counted one criterion's line against the other's expectation. Three consequences: the
  target host must sit outside **both** derivation sets, which retires this file's own two-week-
  old usage example (`keel-skx` derived `STRSM_FLOOR`, so it would have exercised the share arm
  and silently rendered `fleet` on the ratio arm); the wrong-era decoy is landed under every
  criterion key, since a decoy only under `share/` leaves the ratio lookup with nothing above
  its real row while the log still reads *era scoping held*; and rule 19 can out-resolve the
  ratio row before it reaches the class at all, which is disclosed per pass and again in the
  closing summary rather than counted as a clean arm. Verified without host time: eight arms
  against synthesized pass logs — all-good, rule-19-in-pass-1, registered-then-noisy,
  **ratio margin in share units**, **share margin in ratio units**, decoy-consulted, and a
  missing line with no rule-19 excuse — each read the intended verdict, the four false ones
  as NO. Two reporting defects that surfaced only from those controls are fixed: the NO
  paragraph told the era story alone where there are now two causes, and the skip line offered
  two causes joined by *or* when the driver knows which. Still unrun on a host; the live
  three-pass exercise is the arm64 prerequisite it always was.
- **The keel rev CL 2 is verified against is pinned at `ac0f6508e2a4ba6bcbf123e6f397c38f92650574`**,
  cited by SHA in CL 2's description footnote, because keel's `internal/spill` audit is CL 2's
  instrument while hygiene work edits `scripts/` underneath it. The pin certifies stability and
  not birth-correctness (`#143` is open at that rev); admissible only because `#143` is a
  `gate-p5.sh` share defect and CL 2 reads instruction counts, and the pin advances if any CL 2
  number turns out to come from a share. Reviewers were assigned 2026-08-31 (Keith Randall,
  Martin Möhrmann, Jorropo) with no votes or comments — assignment is not a response.
- **CL 1 is mailed: `golang/go` CL 824624**, https://go-review.googlesource.com/c/go/+/824624,
  status `NEW`, branch `master`, +343/−0, `Change-Id: Ifb1d4f47…` — keel's first change on the
  Go tree. Confirmed from Gerrit's `/detail`, not from the push output. The mailed rev is
  `fcc582225c` rather than the reviewed `7e50c40693`: Gerrit rejected the first push because
  `~/.gitconfig` supplies a GitHub noreply author address that is not registered on the
  account, fixed with a clone-local `user.email` and `--amend --author`. Tree byte-identical
  to the witnessed `f70aa0a5b4` and message byte-identical to the reviewed `7e50c40693`, both
  0 diff lines, so the delta touches neither code nor description. **Two apparatus gaps it
  exposed:** no check anywhere asked whether Gerrit would accept the author — every one asked
  about content — and the first `mail` was piped to `tail`, so the `$?` printed beside it was
  `tail`'s and read `0` on a **rejected** push. Supersedes the "still unmailed" entries below.
- **CL 1 is written and verified, one commit, still unmailed.** Branch `keel-cl1-fma`
  at `01f89f15da` (`03b7769900` and `7021eebc03` squashed in, `Change-Id: Ifb1d4f47…`):
  six packed `VFMADD231P{S,D}` ops and a late-lower rule conditional on
  `z.Uses == 1 || z.Op == ssaop.OpPhi`. **The unconditional form it started as is
  wrong** — it took `avx512Peak`'s loop from 27 instructions and 0 copies to 53 and 26,
  because the peak kernels carry the accumulator as a *multiplicand* and want 213. Five
  loops measured against predictions stated before the run, all five exact: `Kernel4x32`
  50/12 → **38/0**, `Kernel2x32` 74/8 → **66/0**, `Kernel6x32` 219/45 → **193/19**, both
  peak kernels unchanged. Both codegen assertions were driven to failure, and a rebuild
  without the `OpPhi` clause proved it load-bearing (`Kernel4x32` then converts nothing).
  Two claims withdrawn: the scalar rule is **not** precedent for an unconditional rewrite
  (there is no scalar 213 op, so it selects nothing), and a narrower phi test fires only
  at unroll 1. **Not mailed; the description awaits Scott's read (ruling 1).**
- **CL 1's description takes three ruled edits, and the third had to be measured because
  the justification offered for it is false** (`7e50c40693`, amended from `f70aa0a5b4`,
  tree byte-identical, `Change-Id` preserved). The census `231=132 / 213=29` with the 29
  split 22 peak declines + 7 folded-addend load forms, and the copy rate qualified as *per
  FMA whose multiplicand is still live afterwards*, needed no new measurement. The phi
  clause did: a multi-use phi whose extra reader is inside the loop and wants the pre-FMA
  accumulator costs **1 copy under 213 against 2 under 231**, so "231 is never worse for a
  phi" does not hold, while the same loop with loop-invariant multiplicands is level at 2
  against 2 — the discriminator is multiplicand liveness, not the use count. The clause
  stands on `rewrite.go:857`'s standard, a load-clobber detector documented as not needing
  to be perfect because regalloc inserts the move. Two instrument defects found first, both
  quantities pinned so they could not move: a `-gcflags=-S` listing written into the
  package directory (Go reads a `.s` there as assembly source) failed every later build in
  **both** arms while a check greened on the empty file, and a copies counter excluding any
  line with `(` excluded all of them. Mailed as CL 824624; see the entry above.
- **CL 1's 231 operand order now has an amd64 *execution* witness, and producing it found
  a scope defect in its own first attempt** (`build/fma-witness4.log`). asmcheck greps
  opcodes, not operand order, and the dev host cannot run amd64, so nothing else in the
  verification executes the emitted code. Phase A perturbs the operand order alone —
  condition held constant, so the destination is a multiplicand again — and must fail on a
  host; Phase B is the witness. Clean, every prediction stated first and exact: census
  **213=29 / 231=132** whole-module, no `PD` forms, **15 of 15** package-host runs across
  three x86 parts. Two facts inside the number: Phase A caught **3 of 5** packages, and
  `vec` and `pack` *passed* a knowingly-wrong permutation because `internal/vec`'s own
  tests never call `FMA512`/`FMA256` in register form; and the first attempt **omitted 42
  of the 132 emitted 231s** — 32%, all in `internal/l1`, which has no tests of its own and
  runs only under the root package. That is the "674 instructions" failure again, a count
  at one scope attached to a claim at another, and what made it look deliberate was a
  scope-justifying comment whose second clause did not follow from its first. Rescoping to
  `./...` made the omission evidence: under Phase A the root package **FAILs**. No
  published number moves; `avx2Axpy` turns out to emit both forms, its addend being a
  freshly loaded `y` the load form folds.
- **Three numbers in CL 1's description were corrected before it was ever shown, by
  re-measuring the quantities each one named** (now `f70aa0a5b4`). "The same 674
  instructions" was unreproducible under every counting rule the listing admits — 651
  all-lines, 619 less pseudo-ops, 506 positioned — while the identity it decorated was
  real; the empty diff now ships with its rule named and a negative control (826 lines
  against the stock arm). "A 10-chain AVX2 loop regresses likewise" was inferred from an
  FMA count, which shows the rule fired and not what it cost; measured, `avx2Peak` goes
  23/0 → **44/21**, 2.30 → 4.40 per FMA. And the 19 residual copies are **18** X-register
  broadcast moves plus **1** Z rotation, not 19 X moves — `upstream-plan.md`'s own table
  had this right, so the prose was the defect. The stock column is now one arm of one
  toolchain (`FMAPrefers231` forced false) rather than three archives of differing scope,
  which is what made the 674 misattribution possible.
- **Every upstream CL's verification run now carries `-all_codegen` preceded by a
  deliberately false assertion shown to fail** (`docs/upstream-plan.md` discipline item
  9, Scott's ruling): *a prerequisite is proven by its failure mode, not by its
  presence.* Both of `#124`'s boxes had a mode that greens on nothing and both were
  first ticked on the cold path — a true assertion passing would have satisfied both
  while proving neither. §5 rule 7 at the upstream workstream; item 7's `GOARCH=amd64`
  clause is the scope half (§5 rule 23) and is checked separately.
- **`#124`'s environment half is discharged, and the `test/codegen` harness has a
  green-on-nothing mode.** Go built from source at `603439a1c6`, `git codereview hooks`
  installed, compiler read off the artifact. A throwaway case in CL 1's exact shape
  (`goexperiment.simd && amd64`, `Float32x16.MulAdd`) was run, driven to fail on purpose,
  and deleted. **Without `-all_codegen` a *false* assertion yields `--- SKIP` under an
  overall `PASS` and one `ok` line, exit 0** — `defaultAllCodeGen()` keys on a
  `gotip-linux-amd64` builder prefix, so every amd64 assertion is inert on this
  darwin/arm64 host and a CL verification run that omits the flag is indistinguishable
  from success. Two more: a bare `amd64:` assertion is checked at **all four `GOAMD64`
  levels separately** (the control failed four times, v1–v4), so CL 1's `VFMADD231PS`
  must hold at every level or name one; and **`VFMADD213PS` is emitted at `GOAMD64=v1`**,
  an independent derivation — different instrument from the `spill-audit` sweep — of the
  mechanism behind `golang/go#80835`'s reported invariance, and of that mechanism only.
- **CL 1 exists as a failing test, committed and unmailed.** Branch `keel-cl1-fma`,
  `03b7769900`, `Change-Id: Ifb1d4f47…`, subject marked `[WIP, unmailed]`: two cases in
  `test/codegen/simd.go`, one per half of `golang/go#80829`, failing at all four `GOAMD64`
  levels. **Mailing is not authorized and has not happened.** Two corrections came out of
  writing it. `#124`'s codereview box **was first ticked on the hook file's existence** —
  the cold path; the warm path is a commit, and it aborted with `exec: git-codereview: not
  found` because `GOPATH/bin` is off `PATH`. Now proven the other way, by a commit that
  produced a Change-Id. And the mechanism: **`213` puts a *multiplicand* in the
  destination**, not the addend as my first draft of the test comment said, so the copy
  appears only where that multiplicand stays live — at the first FMA of the case and not
  the second, which makes "one copy per FMA" the wrong rate to quote.
- **`docs/toolchain-notes.md` T29: `go doc` builds the package for the host arch**, so
  `go doc simd/archsimd Float32x16.MulAdd` on darwin/arm64 answers *"symbol Float32x16 is
  not a type"* for a correct, shipping name — the instrument the prime directive mandates,
  denying what memory had right. Discipline item 7 now requires `GOARCH=amd64`. The
  doc line also states `MulAdd`'s form as `VFMADD213PS` in upstream's own reference,
  which is `golang/go#80829`'s gap.
- **Reported keel's round-trip on `golang/go#80835`** ([comment](https://github.com/golang/go/issues/80835#issuecomment-5471826854))
  as a third manifestation of the reporter's issue, no priority claimed: the
  compiler's 128-bit spill idiom is legacy-SSE encoded *inside* an all-EVEX loop —
  **36 of `Kernel6x32`'s 44 vector stack refs**, 18 round-trips per iteration, among
  48 ZMM `VFMADD213PS`. So a boundary `VZEROUPPER` or `ClearAVXUpperBits` cannot reach
  this shape, and the dirty uppers are ZMM-wide. **`GOAMD64` does not move it** — v1
  through v4 diff clean, controlled by a sweep that does move the package listing at
  v3. No timing: the shape is unshipped and nothing was attributed, stated so in the
  comment. Correction to `bb3284e`'s message, which claimed keel's notes lacked the
  precise form of `golang/go#80829`: **T12 recorded it on 2026-08-11**
  (`toolchain-notes.md:852-859`, `spill-report.md:295`), naming the generator where my
  re-read counted its output. Today's pass is corroboration under §5 rule 10, and adds
  only that the gap is unchanged on go1.27.0.
- **Replied to `golang/go#80828`** ([comment](https://github.com/golang/go/issues/80828#issuecomment-5471635162)),
  answering `cherrymui`'s 17-day-old question with a seven-row re-swept table, and
  disclosing that the issue **duplicates `golang/go#78753`** — closed 2026-05-26,
  fixed by CL 768262, unconditional. The headline is not the register count:
  **every spill go1.27.0 removed came back as a register-to-register copy, one for
  one** — zero net instructions at N=14 and N=15, 1 at N=16, 11 of 79 at N=20 —
  because `213`-only FMA forms (`golang/go#80829`) mean an accumulator cannot be
  written in place. Controls: N=13 reproduces all three of the issue's own go1.26.5
  columns, and every row on both toolchains closes as `insns = N + refs + copies + 3`.
  Left open upstream: `simdRegMaskAMD64` is still X0–X14 and `compatRegs` intersects
  with it, yet `Z16`–`Z23` are allocated, so the path that reaches them is unidentified.

- **The population check CL 1's one-for-one figure was admitted on** — every copy in
  four steady-state loops classified as *rescue* (a `VFMADD213PS` overwrites the
  register a pre-FMA copy touched) or *rotation* (a post-FMA copy restoring the
  loop-header assignment). `Kernel4x32` 12 = 4 + 8, which is CL 1's cited figure
  reached mechanically; `Chains13` 27 = 13 + 14; `Chains14` 26 = 12 + 14; **0
  unattributed in all three**, and **0 of 83 FMAs across the four loops writes its own
  addend**. `Kernel6x32` is the positive control at **15 unattributed** — the `X`
  scalar moves of the broadcast round-trip, `golang/go#80835`'s subject, in the kernel
  CL 1 does not cite. Disclosed with it: that `231` removes these copies follows from
  the operand form and is **not measured**, since no toolchain emits it here yet — a
  caveat that binds the cited 12 equally, 8 of which are rotation copies.

- DESIGN.md §5 rule 23 and `docs/rulings.md` rule 23: **an absence claim states the
  scope its instrument actually searched**, and that scope must cover the claim's
  scope or the finding is `unmeasured` rather than absent. Ruled on the second
  instance in two days — a host set without Sapphire Rapids adjudicating a µarch
  claim, then `--json body` adjudicating a citation whose table was in a comment.
  Signature: the instrument reports in the subject's voice. Distinct from rule 11,
  which says the instrument adjudicates; 23 says which instrument may.

- `docs/toolchain-notes.md` header: **every entry is dated by its toolchain**, as a
  law about claims. *The toolchain moved under a true finding* — #18 was right and
  go1.27.0 falsified it with nobody touching keel — so a codegen claim carries its
  `go version` inside the claim, a superseded entry is marked at its head rather
  than rewritten, and a toolchain bump re-opens every entry it could touch.

- `docs/upstream-plan.md`: **CL 1 adjudicated against `golang/go#80835` and does not
  re-key** — adjacent, not the same defect nor a superset. One symptom, three
  separable causes: `golang/go#80830` (emulated broadcast, why the wrapper exists), `golang/go#80835`
  (legacy-SSE encoding, the price), `golang/go#80829` (`231` and `.BCST`, CL 1). `golang/go#80835` is
  assigned to `JunyangShao`, so keel's contribution there is `#144`'s one comment —
  the round-trip is a **third** manifestation alongside the shift count and stack
  zeroing — not a CL racing an assigned maintainer.

- `docs/toolchain-notes.md` T28 and `docs/spill-report.md` §11: **#18's cause is
  refuted on go1.27.0.** 31 of 32 vector registers are allocatable, not 15
  (`specialRegMaskAMD64` supplies X16–X31; only X15 is in no mask), identically
  under `GOAMD64` v1/v3/v4. `Kernel6x32`'s steady-state stack refs fell 90 → 44 and
  36 of the 44 are broadcast-scalar round-trips through legacy-SSE `MOVUPS`
  (`golang/go#80835`), with only 3 of 12 accumulators spilling and X24–X31 idle. #18
  was right when written — its N=20 repro was above the frontier and named nothing
  above `Z14`; the toolchain moved under a true finding. Corrected at all three
  sites that carried the old cause, and CL 1's register-allocation half is
  **withdrawn from the description, not deferred**. Opened
  [#144](https://github.com/scttfrdmn/keel/issues/144) as the `standing-task` keyed
  to `golang/go#80835` for the 36 round-trips.

- **`3a7ad60`'s commit message carries a false correction; this supersedes it.** It
  says the 109.7× pairing's provenance *"was wrong … not from #104"*. It was right:
  the table is in a #104 **comment**, and the check that declared it fabricated read
  `gh issue view --json body`, which omits comments. `docs/upstream-plan.md`'s
  discipline list now says a citation check reads `body,comments`. The
  `build/gate-p2-f19a977.log` citation added alongside it is sound and is kept as
  the primary source.

- DESIGN.md §5 rule 22 and `docs/rulings.md` rule 22: a criterion is applied by
  the mechanism it names, not by the surface form of its wording. Ruled on the
  v0.1.0 release report — the tag-delta condition's canonical form is *nothing in
  the gate's input closure*, and the `scripts/` cap forbids apparatus sprawl
  rather than compelling either of the two things its ledger arithmetic would.
  Available only with the mechanism computed and positive-controlled, both
  readings disclosed, and the reading accepted by the authority.

- `docs/neon-probe.md`: the arm64 lowering probe (#129), static only.
  Accumulate-in-place is **absent** on arm64, because `simdARM64.rules:242` rotates
  the accumulator into arg0 of a `resultInArg0` op — a lowering-rule fact, not an
  architectural one, since `VFMADD231PS` also writes its own accumulator and amd64
  still misses it. Broadcast is **present and worse**: identical emulated Go source,
  but arm64 keeps a dead zero-init `VMOV` for 3 instructions where amd64 needs 1.
  The wrapper anchor is **present, same cause, 4 bytes not 1**. Supplies the figure
  CL 1 needs: **12 register-to-register moves per 8 FMAs, 24.0% of `Kernel4x32`'s
  50-instruction loop**, agreeing with `spill-audit`'s own `12 reg copies`. The
  register-limit finding this entry first carried is retracted below.

- `docs/upstream-plan.md`, the root doc for the only workstream with an external
  clock: the CL ledger against verified upstream issue numbers, the shared per-CL
  evidence shape, the freeze budget, and the three figures that failed
  verification and may not be cited. Two of those were wrong in the plan it
  descends from — `T17` is **+15.5%** static instructions and not −15.5%, and it
  was never paid; the nest-on-SKX gap is **+33.4%** nest improvement and not
  "18.4-pt" — re-derived 2026-08-30, the decomposition's product is 43.34% and the
  bar 57.8, so even the points form reads 14.46 pt and `18.4` has no provenance
  anywhere in the tree. The third, the "110× spill price", was recorded here as unlocated and
  is now located — see the retraction below. Also
  records that the plan's "CL 1: `golang/go#80830` embedded-broadcast lowering"
  was mis-keyed: embedded broadcast is `golang/go#80829`, the third CL.
- CONTRIBUTING.md documents the post-v0.1.0 label and milestone taxonomy:
  milestones are workstreams now rather than phases, `workstream/*` cuts across
  them, `science` means unscheduled and carries no milestone deliberately, and
  `external-clock`/`needs-scott` mark the two kinds of work this repo's keyboard
  cannot finish alone.

### Changed
- **`builder_toolchain` dropped the GOEXPERIMENT from a from-source toolchain** — its
  `awk '{print $2}'` reads field 2, which is the whole stamp for a release
  (`go1.27.0-X:simd`) but only the version for a devel build (`go1.28-devel_<sha>
  <date> X:simd`), where `X:` is the last field. So `#58`'s own instrument, the one
  that exists so a gate log cannot carry two compilers without saying so, would have
  recorded a compiler without saying which experiment — on the first verification run
  against a patched compiler, which is exactly what `#124` enables. Found by *running*
  it against both toolchains rather than reading it: released stamp unchanged
  byte-for-byte, repeat call silent (on-change only), and the `CHANGED mid-run` branch
  driven on purpose, naming both compilers with the experiment now kept. Net +0 lines
  in `scripts/`.

- **Retracted: the `GOAMD64` v1/v3/v4 invariance does not refute CL 767380's *"v4 or
  higher"* framing.** That CL was **abandoned 2026-04-17**; the merged fix is CL
  768262, unconditional. The invariance therefore *confirms* what landed. The claim
  came from gabyhelp's "Related Code Changes" line, which prints CL subjects without
  statuses — read the thread, not the title, applied to a CL. Corrected in T28 and
  `docs/upstream-plan.md`, whose discipline item 2 now requires a Gerrit status check
  before any CL is cited. The drafted reply's *"9 refs against 25, roughly 3×"* was
  dropped in the same verification: `25` has no provenance, and the issue's own table
  says **44** at N=20.

- `docs/toolchain-notes.md` T28: the go1.26.5-vs-go1.27.0 comparison is licensed by
  the N=13 control and the closing identity rather than left disclaimed. The earlier
  note that *"the reg-copy column does not compare"* is superseded — it does, and the
  comparison is the finding. N=15's highest register is `Z17`, measured; the sweep
  gained the `insns` and `reg copies` columns and N=15/24/31 stack refs.

- **`docs/upstream-plan.md`: the CL order is keyed to the subject** (ruled
  2026-08-30). The plan
  landed one day earlier carrying a transposition — embedded-broadcast lowering was
  to lead, and that content is `golang/go#80829`, not the `golang/go#80830` the
  ordering named. `golang/go#80829` is now CL 1 and `golang/go#80830` CL 3; each
  issue kept its own subject, evidence and verified upstream number, because
  swapping the *references* onto unchanged bodies would have re-created the mis-key
  in a better-hidden form. Two of the ordering's three premises did not survive
  verification and are recorded beside it: `golang/go#80829` is the **largest** of
  the three CLs, not the smallest, and what lets it lead is that its two halves are
  independently mailable; the premise that does verify is the payoff, via #104's
  P2 STOP.

- **Retracted the same day, by the condition attached to the ruling that accepted
  it: the `110×` spill price is real, is keel's own measurement, and is off-subject
  for CL 1.** #104 measures `Kernel6x32` at 30.5% of keel-zen4's peak against 0.278%
  of keel-spr's — 30.5/0.278 = **109.7×**, the percent-of-peak pairing #18's
  four-pairing enumeration missed; the same quantity reads **121.1×** on Sapphire
  Rapids in #18's six-host table. Primary source for both terms, added 2026-08-30:
  `build/gate-p2-f19a977.log:77-89`, one sweep. The refutation compared a *µarch* claim against
  `docs/spill-report.md`, whose hosts are janus, vesta and antares — no Sapphire
  Rapids — so "25× larger than anything in the report" was true of the report and
  false of the tree. It still may not enter a CL description: the compiler emits the
  same spilled code on every host, so only the silicon's price differs, which is a
  well-formed claim pointing at the wrong mechanism. **Also retracted:** the probe's
  reading that AVX-512's `w`/`w31` masks exempt keel's kernels from the 15-register
  limit. On go1.27.0 `Kernel6x32` allocates `Z16`–`Z23` — 23 vector registers for a
  tile needing ~15 values — and still carries 90 vector stack refs, with
  `Kernel2x32` and `Kernel4x32` at max `Z14` as the positive control. #18's cause is
  open, the reg-alloc half stays out of CL 1's description, and `ssa.html`
  adjudicates.
- **`doc-site/limits.md` splits "what keel does not do" into four claims that were
  previously one list**: commitments that will not change (row-major only, no
  `Isamax` vector kernel, subset-not-whole-BLAS, panics not error returns),
  roadmapped work with issue numbers (NEON, the AVX2 microkernel, f16/bf16 pending
  an upstream element type), one open question (float64 — where the blocker is the
  oracle, not the kernels), and parked ideas (complex, packed/banded, int8/VNNI,
  auto-tuning). A flat list made a schedule read as a promise and a commitment read
  as an oversight; the page is now the single canonical statement, and README's
  Scope section links to it instead of restating the split.
- **Three stale status claims corrected on public pages.** README, the doc-site
  landing page and `limits.md`'s Maturity section all still said there was no
  tagged release and that P5 was in progress with its gate red. v0.1.0 is tagged
  and gate-p5 is green at `72 PASS / 0 FAIL`; all three now say so, name the
  certificate, and name the two known shortfalls (#104, #111) rather than omitting
  them.
- CONTRIBUTING.md states the certificate-transfer condition next to the tag
  requirement, so the next release finds it where the release rules are.
- CONTRIBUTING.md records the two mechanisms that bit while publishing v0.1.0,
  both of which fail silently. `git tag -a -F` applies the default cleanup, which
  strips every `#`-leading line as a comment, so all three `## ` section headings
  vanished from the tag object with nothing to diff against — the prose survived,
  including the seam disclosure, and the fix for next time is `--cleanup=verbatim`
  plus a diff of `%(contents)` against the notes file. And enabling Pages
  auto-creates a `github-pages` environment whose only allowed ref is the branch
  `main`, which contradicts a deploy job gated on `refs/tags/v*`; the contradiction
  renders as a job that fails in ~2s with **zero steps and no log**, so the tag
  needed a `type=tag`, `name=v*` deployment-branch policy added once.

## [0.1.0] - 2026-08-29

The first tagged release. `[Unreleased]` had grown to 26 session groups over 4,875 lines;
per CONTRIBUTING.md the collapse to canonical type headers happens at the version cut and
never before, so this section is that one-time editorial pass. Every bullet below is the
text it had when it landed — the pass regrouped lines and rewrote none, checked by
comparing the multiset of non-blank content lines before and after (4,783 either side, 0
lost, 0 invented). Session provenance therefore lives in the git history from here, which
is where it was always recoverable; what a reader of a release wants is one Added/Changed/
Removed/Fixed set, and that is what this is.

### Added

- **`aws-fleet.sh up` refuses to launch while `$KEEL_REMOTE_HOSTS` is set**, unless
  `--measure-the-override` names that as the intent. `remote.sh:35` ranks that variable above the
  `.keel-hosts` the launch writes, so a set value bills a fleet and measures something else — $192.70
  of it. The guard is the first statement in `cmd_up`, fatal before the first billable call, because a
  detached run's warning is read by nobody and the money is gone by the time anyone reads it. The flag
  **honours** the override rather than clearing it: a flag that cleared it would be nothing but a way
  past the guard. Driven in all four arms, including against the real stale value `antares`, with a
  positive control — the launcher-presence check sits *above* `cmd_up`, so the obvious probe
  (`KEEL_SPAWN=/nonexistent`) short-circuits before the guard and discriminates nothing.
- **DESIGN.md §5 rule 21 and `docs/rulings.md` rule 21: a record of deltas cannot see an injection, so
  what decides a measurement is stated totally.** Three findings from the first release campaign, in
  one law. (a) An environment record that enumerates what the caller *changed* is blind by construction
  to state that was already wrong — `build/<name>.cmd` was clean on the poisoned run. (b) Carry and
  inherit are two directions of one line and two separate defects: a dropped override fails toward the
  configured fleet, an injected one toward whatever was last measured, and the 2026-08-28 fix closed
  only the benign half. (c) A stated assumption is not a check — the certificate's own
  `stated assumptions (trusted, not verified)` block names "the configured host set is the fleet this
  gate is meant to measure", the exact proposition the run violated; the disclosure bought the
  diagnosis and prevented nothing, which is grounds to move a *what-is-measured* assumption off that
  list and into a guard. (d) `RED` with zero `FAIL`s is the tally line stating that the apparatus, not
  the subject, failed — readable only because the vocabulary separates a refusal from an absence.
  Coverage stated per §5 rule 12: `detach.sh` has **no automated test at all** (verified — no file
  under `scripts/` names it), so rule 21's own fix can regress silently; filed as `#122` rather than
  repaired here, because the tag delta may not add shell.

- **The `go 1.26` floor is now measured, and it is TRUE** (`.github/workflows/ci.yml`, new `floor` job). Four files
  say `go.mod`'s directive is the *scalar* path's floor and cannot express "1.27 if `GOEXPERIMENT=simd`"; nothing
  checked it, because both existing jobs pin 1.27.x — so after T23's rename the floor was a claim no instrument
  could see, on the arm that rots quietly (the vector path breaks loudly). Measured on `antares`, a real go1.26.5
  linux/amd64 host, before being automated: `go build ./...` and `go test ./...` both returned 0 at HEAD, and the
  same tree under `GOEXPERIMENT=simd` returned 1 with the T23 errors. **That contrast is the floor's content**,
  which is why the job must never set `GOEXPERIMENT`. Recorded because it will catch someone: the first attempt
  shipped the tree by `git archive` and `tools/shapegen`'s two fixed-point tests failed at
  `git rev-parse --show-toplevel` — a tarball has no `.git`, so that is the transport, not the toolchain.
- **Every gate now records which Go compiled the binary the fleet ran, read off the artifact** (`builder_toolchain`
  in `scripts/remote.sh`, called by `remote_build_test_or_fail`; issue #58). `go version <ELF>` reports the compiler
  *and* the GOEXPERIMENT out of the cross-compiled test binary — `go1.27.0-X:simd` — where a bare `go version`
  reports the driver in the calling shell, a different fact that `GOTOOLCHAIN` can make differ. This closes a real
  gap rather than a decorative one: the dev host cross-compiles everything the fleet runs, its `go` moved from
  1.26.x to 1.27.0, and no archive name carries a compiler field (rev and run stamp only). Printed on *change*, so a
  toolchain that moves mid-run announces itself; exercised across five arms, including a seeded stale value to make
  the quantity move and an unreadable artifact to prove the reading is left undisturbed. Deliberately not set inside
  `remote_build_test`: every caller redirects that function's stdout to a build log, and two call it in a subshell.
- **Both of P5's scaling bars are TYPED, in one commit, from one run's rows: `CEIL_FRACTION = 44.2` and
  `STRSM_FLOOR = 6.067x`** (`scripts/gate-p5.sh`, `scripts/readme-numbers.sh`; the pinned8 era re-founded on
  the spread mask and `instrument=v2`). Derived from the founding campaign's take four, **recomputed from the
  archived raw samples under #116's honest CI bounds** rather than from a fresh fleet run: the share bar is the
  argmin of six admissible rows, `keel-zen5` `Ssymm` at 46.8150% net of both intervals, less the same 2.6
  points; the ratio bar is the argmin of three judged rows, `keel-zen5` at 6.4699x net of CI, less the 0.403x
  `STRSM_MARGIN` that predates these readings by six days. **The share bar fell 6.8 points from the suspended
  51.0 and 95.0% of that fall is the denominator** — #115 lifted the ceiling's fork/join out of its own timed
  region, so the reference rose +14.6% while the bar-setting rate fell 0.74%; the new rate over the old ceiling
  would have typed 50.8, the old rate over the new ceiling 44.6. A lower bar because the reference got more
  honest, not because the nest got slower. **`6.067x` lands within 1.1% of the retired cross-host `6.0x` by
  coincidence and says so at the constant**: different quantity, different placement, different silicon, and
  the rank inversion that retired 6.0x is untouched — carried to three decimals precisely so it is not
  typographically confusable with the number `readme-numbers.sh` publishes *as* retired. Both bars are enforced
  on the next run, which is the property that lets them fail.
- **The spread amendment's own motivating prediction is refuted by the run it was made to enable** (§5 rule 11;
  DESIGN.md §4/P5). Of the four substantive pre-registered predictions, three fell. Every 8-thread rate on both
  EPYC hosts *dropped past its own interval* — 8 rows of 8, `keel-zen5` -0.7 to -4.0%, `keel-zen4` -4.0 to
  -7.3% — where the L3-capacity argument predicted rises, and no kernel changed in the range, so the mask owns
  the delta. The re-derived bar landed 6.8 points *below* the 51.0 it was predicted to exceed, and no `Strsm`
  ratio cleared 7.0x net of CI. **The control did more than confirm: it made the largest move attributable.**
  `keel-skx`'s mask is degenerate (per-socket L3, same `0..7` both ways) and its rates moved -2.1% to +0.5%,
  yet its ceiling rose the *most*, +23.4% — which a mask that did not change cannot cause. So the ceiling's
  rise belongs to #115's instrument repair, not to placement, and two amendments that landed together are
  separable after all. The amendment itself stands on the grounds that survive: a standing placement bars
  flattery-shopping whether or not it happens to be faster, which was never the claim under test.
- **The founding campaign ran twice on the same commit and fleet, and the pair is what made the instrument
  legible** (`6ba6566`, 2026-08-22/23; `build/campaign-6ba6566.log`, `build/campaign-c30-6ba6566.log`).
  Take three at `-count=10`: 56 PASS / 3 FAIL / 5 UNMEASURED / 4 BASELINE / 5 REPORTED. Take four at
  `KEEL_BENCH_COUNT=30`: 62 / 3 / 4 / 4 / **0**. Both RED, on the same three reds. Rule 19 executed on a real
  fleet for the first time and fired fail-closed, and take four vindicated it: the zen5 `Strsm` reading it
  refused a vote to was **9.8% off** its own re-measurement (427.15 → 388.85 GFLOP/s at 8 threads).
  **The surprise is that the lost resolution was not the host's.** zen5's ceiling contamination is real
  (2 of 10 low, recurring 1 in 30) but cost nothing; `benchci`'s `ciFraction` mirrored a *one-sided* median CI
  onto the side with a 0.5 GFLOP/s half-width, so the gate divided by 2588 GFLOP/s — above every sample ever
  taken there. Filed as #116 with the arithmetic; conservative in the safe direction, so no published figure
  is overstated. Re-measurement confirms it independently: the corrected take-three shares (48.25/47.56/46.64)
  land within ~0.3 points of take four's judged 48.0/47.9/46.8, against a shipped rendering off by 5+.
  `CEIL_FRACTION` now has six admissible rows over **two** hosts, spread 46.8–75.4%.

- **A third verdict class for a reading whose interval is too wide to adjudicate its own criterion**
  (`REPORTED`, `scripts/remote.sh`; DESIGN.md §5 rule 19, `docs/rulings.md` rule 19; ruled 2026-08-22 on #6).
  The spread mask widened `Strsm`'s intervals 3–15× — ±0.75/0.87% to ±5.13/10.28/13.24% on zen5 — while
  keel-skx's identical-mask control did not move, so a per-row placement exception was considered and
  **refused**: the mask reweights a bimodality both placements exhibit rather than measuring a truer
  quantity, and picking the quieter placement hides a real behaviour of the routine. What lands instead is
  #105's existing three-state clause extended to the scaling and share criteria. A row whose width exceeds
  **the criterion's own declared slack** — 2.6 points of share, or 0.403x for `Strsm`, both predating these
  readings — prints, is archived, and has only its *comparison* refused; a row within cap judges normally.
  So keel-skx judges and the two EPYCs report. Bars type from admissible rows only or stay empty. The class
  is the first that is **per row**, which added a per-host bucket for a host every row of which was
  out-resolved: `HOST_CLEARED` starts at 1 and is only ever lowered, so without it such a host would have
  been counted as a clean sweep. Driven, not read: 19 renderings against the file's own bytes, both
  predicates at their boundaries, all four arms of the all-rows test, and the aggregate's four sentences.
  What stays unexercised is inside the ruling, per §5 rule 12.

- **A run's archives are named for HEAD at the moment each file is written, not for the code that produced
  them, so one launch can carry two revision labels** (`scripts/bench.sh:117`, measured 2026-08-22). The run
  stamp is pinned on first use (line 115) precisely because a per-process counter discriminated nothing
  between runs; the **rev beside it is recomputed on every archive** and has the same defect the stamp was
  fixed for. Today's `gate-p3` sweep group shows it directly: one `BENCH_ARCHIVE_RUN` value, `20260822T185826Z`,
  split across **`bench-gate-p3-450a783-...-1.txt`** (written 19:05:47Z) and **13 files
  `bench-gate-p3-2a5bfa3-...-{2..14}.txt`** (19:08:11Z–19:23:49Z) — because `2a5bfa3` was committed at
  19:07:14Z, under nine minutes into the group. Nothing distinguishes the two labels but the clock. This is
  the freeze rule's hazard leaving a visible trace rather than a new one: `2a5bfa3` touched `scripts/remote.sh`,
  which is the file a live run re-invokes per host and which bash reads incrementally, so the label flip marks
  the same commit that could have moved bytes under the interpreter. **What the labels cannot tell you is which
  code ran** — that is the finding, and it is why a rev label is not provenance. **REPAIRED the next day**
  (ruled 2026-08-22 on #115: freeze-exempt on the criterion-honesty clause, because the founding campaign's
  archives must name what wrote them). `BENCH_ARCHIVE_REV` is now pinned on first use exactly as the stamp
  above it is, one `:-` default and nothing else, and the field's *position* in the filename is unchanged
  because `readme-numbers.sh` reads the rev by offset from the `bench-gate-p5-` prefix. Exercised with a live
  positive control rather than by inspection: a shadow `git` returning a different rev per call gives
  `rev0001 rev0002 rev0003` unpinned and `rev0001` three times with one `rev-parse` call pinned. **The first
  attempt at that exercise was a dead instrument and is recorded as one** — the counter incremented inside a
  command substitution, so the fake `git` returned `rev0001` every time and unpinned code would have produced
  the identical output (the readable-constant trap: an instrument that cannot move certifies nothing). Redone
  file-backed.
- **The 8-thread compute ceiling was measuring the wrong noun, and the repair is a fidelity repair that
  predates the mask** (`bench/ceiling_test.go`, ruled 2026-08-22 on #115; new DESIGN §5 rule 18). `computeArm`
  forked eight goroutines and joined them **inside every `b.N` iteration**, so the timed region under the
  project's most load-bearing denominator was **78% arithmetic and 22% convening**. Scott's formulation is the
  new rule: *the ceiling's definition is what eight threads can compute, and an instrument whose measured
  quantity is 78% compute and 22% choreography is measuring the wrong noun.* Workers are now created before the
  timer, each **warmed by a full op on the cpu the mask gave it**, and parked on a `start` channel; the timed
  region is one `close(start)`, the steady-state loops, and one `wg.Wait()`. Excess over ideal goes **1.290 →
  1.010** (zen4 spread) and **1.599 → 1.014** (zen5 spread), duty cycle **0.67–0.88 → 0.97–0.99** — so the
  removed time was time the workers spent *parked*. This is not cleanup after the spread mask: at **25.5 µs**
  per fork/join the confined-era ceilings carried the same contamination at smaller amplitude, so the mask was
  an amplifier of a harness term and booking the repair against placement would have made a placement decision
  out of a harness defect. `internal/par` forks per call too and is **left alone**, because the term is 22% of a
  267710 ns ceiling op and 0.03% of a 204126044 ns `Sgemm/n=4096/threads=8` op — §5 rule 14, a severity that is
  a function of deployment, and this one was deployed as a denominator. `streamArm` also keeps its per-iteration
  fork deliberately: ~0.5% on a ≥10 ms op, REPORTED and never in the `min()`, and hoisting it would move
  first-touch page faults out of the timed region, which is its own decision with its own magnitude to measure
  rather than something smuggled in beside this one. **What the hoist trades for, disclosed at the arm:** v1 was
  `b.N`-independent by construction and v2 amortizes one convening over `b.N`, so a short run under-reads —
  **74.15 GFLOP/s at `1x` rising to 127.5 at `5000x`, converged by `500x`** — and the gates' `-benchtime=1s`
  sits far inside the flat region at ~0.005%, which is a fact about the gate configuration and not about the
  arm. An `ops=` field was drafted and **backed out**: `declareCeiling` keeps one line per row and `testing`
  calls the benchmark once per ramping `b.N` trial, so the `b.N=1` call wins and it printed `ops=1` beside a row
  that measured 3853 iterations. Every pre-existing field is `b.N`-invariant, which is why that dedup was safe
  before and why the trap is recorded instead of worked around.
- **The `free-placement` era's evidence is now in the repository: 35 archives at `archive/free-placement/`,
  which is half of `pinned8`'s both-arms condition landed before the run that lands the other half**
  (2026-08-21, ruled on #6). The era ledger makes a both-arms transition archive a condition of an era existing,
  and the arm it wanted was cited as a path under gitignored `build/` — an arm that resolves on one operator's
  laptop and nowhere else, which is the defect #114 already names for the BASELINE witness. 464 KB of plain text
  buys a citation that resolves in any clone. **The set is closed by construction**, because `remote_exec` refuses
  with status 121 rather than running a benchmark free, so `INDEX.tsv` was generated once from the files beside it
  and has no tracked generator: apparatus for a set that cannot grow has nothing left to measure.
  Read out of the archives rather than off their names: **15 `ladder` archives** (the judged benchmarks plus
  `Peak`, one per revision × host over five revisions and four CPU models) and **20 `clock-window` archives**
  (`Peak/avx512` alone, §5 rule 5's substitute clock instrument). **Twelve of the fifteen carry the `Ceiling`
  arm** — the three at `ce43bca` predate it, so a share over the measured attainable ceiling is recomputable at
  four revisions and not five, and per CPU model that is 4 / 4 / 2 / 2 for zen4 / zen5 / gnr / skx.
  The arm ships with the disclosure it was ratified with: **these are single draws per configuration.** Ten
  `-count` samples give within-run spread, but a model's several archives sit at different *revisions*, so they
  are one draw each of several configurations rather than several draws of one — §5 rule 16 applies to this arm
  exactly as to the README rows measured beside it, which is why `host-baselines.tsv` refuses to import keel-skx's
  baseline from `5ec5fea` or `33de3b2`. So **the era mapping's precision is bounded by the free arm's own
  spread**, and this arm cannot bound its own run-to-run component: a pinned row differing from its free
  counterpart by less than that has not been shown to differ. Nothing was upgraded to make the transition read
  better. Recorded and not fixed here: the 20 clock windows cannot be mapped back to the head/middle/tail series
  the gate log reports, and **two causes are available** — `peak_window` calls `bench_csv "$log"` untagged
  (`scripts/bench.sh:509`), so the window's own name reaches the scratch log and not the archive, and these
  predate `RUN_STAMP`, so a second run at one revision overwrote the first. Either explains four preserved windows
  per revision where a three-host fleet running a three-invocation series implies six; neither is asserted. The
  tag is a one-word fix held until after the era-founding run, because `readme-numbers.sh` parses these names and
  the moment before a founding run is the worst one to change them. It costs nothing published: every number
  citing this era cites a `ladder` archive, and those name their host with no collision to resolve.
- **Wave 2 classified two Intel AVX-512 hosts and they split: SKX admits to the judged set, ICX does not**
  (#6 Q3, 2026-08-21; `build/wave2-classify-7ac592a.log`). `keel-skx` — **Xeon Platinum 8124M**, `c5n.18xlarge`,
  36 cores / 2 sockets, equal to `c5n.metal`'s core count — classifies **issue-bound**: ceiling mixes converge
  `1.023×` over a `2.778×` spread in insns/FMA, interval `[1.017×, 1.026×]`, clear of `1.10`. That is janus's
  class and the roofline exception P2 has had since #19, so `c5n.18xlarge` joins `KEEL_EVIDENTIARY_SIZES` in this
  commit, which cites the read-back that justifies it — the addition follows the evidence and never precedes it,
  because the allowlist's safety property is that a stale list may only *withhold* a judgement. `keel-icx` —
  **Xeon Platinum 8375C**, `c6i.32xlarge`, 64 cores / 2 sockets — classifies **fma-bound**, is held to the flat
  55% floor and reads **48.7%** of measured peak, so it is not added: admitting it would grant a judgement that
  fails P2. **Signing fleet: c7a.48xlarge + c8a.48xlarge + c5n.18xlarge.** ICX joins `gnr` and `spr` as
  characterization. Both hosts ran OpenBLAS 0.3.26 `DYNAMIC_ARCH corename=SkylakeX`, on the allowlist.
- **ICX's class turned on 0.25% of margin, which makes #86 a measurement rather than a prediction** (2026-08-21).
  Its ceiling spread is `1.1028` against a `converge_max` of `1.10`, on a zero-width interval. Re-running
  `throughput_verdict` — gate-p2's own pure function, so the instrument adjudicates rather than a rederivation
  (§5 rule 11) — with `converge_max` raised to `1.11` and nothing else changed returns `issue … 0.907758 pass`
  where the measured inputs return `fma … fail diverge`: **one unchanged keel rate, two opposite verdicts, a
  quarter of a percent apart on a classifier threshold**, which is exactly the flip #86 was filed on. Held as a
  **sensitivity probe and not a second verdict**, for two stated reasons: the mix bounds were reconstructed from
  the log's displayed rates rather than read from the gate's arguments — exact for ICX, whose printed `[1.103×,
  1.103×]` reproduces, and *wrong* for SKX, whose printed `1.017×` lower bound the reconstruction misses
  (harmless there, since the register-only peak kernel is the argmax and its `f` is `1.0` by definition, so
  neither class nor attainment moves) — and the counterfactual `0.9078` clears `0.90` by 0.8 points while resting
  on a rate read to three figures. The finding is the fragility, not the counterfactual verdict.
- **`BenchmarkCeiling` measures both halves of P5's per-host attainable ceiling** (ruling on #6, 2026-08-20),
  the denominator that replaces the retired 6.0x cross-host scaling floor: `compute` runs `BenchmarkPeak`'s
  register-only FMA kernels concurrently on 1 and 8 threads, so the clock droop with core count is *inside*
  the reading rather than disclaimed beside it as `gate-p5`'s old `8x the single-thread peak` info line had
  to; `stream` probes bandwidth with keel's own `Sdot` (2 reads, 4 accumulator chains) and `Saxpy` (2 reads
  + 1 write, no chain), whose byte counts are countable rather than modelled and which are already
  differential-tested. Working set is sized from the *measured* LLC (4x it, floored at 256 MB) because a
  stream that fits in L3 reports L3 as memory bandwidth. Two disclosures inside the numbers: `axpy`'s figure
  counts 12 architectural bytes/element where write-allocate makes the bus traffic up to 16, so the truth is
  bracketed at x1.33 and both patterns are published; and the per-thread sink discipline holds without an
  instrument, since T17 makes `-race` fatal on amd64 wherever archsimd's partial ops are reached.
- **Allocating a stream buffer inside a benchmark arm made the arm's timing a function of which arm ran
  first, worth 12x.** `stream/dot/threads=8` first read **15.4 GB/s against its own 1-thread arm's 24.1** — a
  decline with thread count no memory system produces. Each arm wanted ~4 GB at the 8-thread size, so the
  first sample paid page faults and later ones ran against the previous arm's garbage. Allocated once for the
  process and reused, the same arm reads **196 GB/s**. Recorded because the wrong number was *plausible*: it
  had the shape of a real bandwidth saturation finding.

- **README's twelve-row block is republished from the judged AWS campaign: 24 rows, three new CPUs, rev `ce43bca`.**
  `scripts/readme-numbers.sh build/gate-p5-ce43bca.log` regenerated both marked regions and `docs-gen.sh` extracted
  `doc-site/numbers.md` from them, so the published rates now describe EPYC 9R14, EPYC 9R45 and Xeon 6975P-C instead
  of the lab fleet. The generated caption discloses 4 of the 12 scaling ratios as below `gate-p5`'s floors, 3 outright
  and 1 (9R45 Ssymm, 6.011x point / 5.801x net) decided by measurement precision. Published from the **first** of the
  two runs DESIGN.md §4's one-re-run allowance permits, both archived (`build/run1-ce43bca/`, `build/run2-ce43bca/`),
  because it is the only run in which every configured host produced rows — the gate's own README criterion is per
  host. Stating the favourability plainly: that choice also makes the caption read 4-of-12 rather than run 2's 4-of-8,
  since run 2 produced no gnr rows at all.
- **keel gets 47.6% of OpenBLAS on Granite Rapids — below P3's 60% floor — and the same host's scaling looks the best
  in the fleet for the same reason.** Run 2, `keel-gnr` (Xeon 6975P-C, c8i.96xlarge): keel 102 GFLOP/s against a
  pinned-SkylakeX 214.3, i.e. 47.6% (47.2% net of CI). Run 1 left this UNMEASURED on the clock instrument below, and
  the miss was already implicit in run 1's own printed numbers (104.3 against 214.70 = 48.6%) — an UNMEASURED is not a
  pass, and this one hid the campaign's worst reading for a whole run. The mechanism is measured, not inferred: keel's
  Sgemm keeps **42.5%** of gnr's own 1-core peak against **88.0%** on Zen 4, its 4x32/avx512 microkernel reaches
  **33.4%** of that peak unblocked against **96.8%** on Zen 4, and the wider kernel buys *nothing* there (104.3 against
  104.7 GFLOP/s for 2x32) while buying **+25.6%** on Zen 5 (169.9 against 135.3). So the dispatch selects the wide
  microkernel on Granite Rapids for no gain, and the headroom that leaves is exactly what makes gnr's 8-thread ratio
  the only one in any log to clear 6.0x. One mechanism, two opposite-looking verdicts (§5 rule 6).
- **P5's 6.0x scaling floor is rank-ordered against per-core efficiency, in the direction that refuses the best host.**
  Sgemm at `ce43bca`, single-thread rate as a fraction of the host's own measured 1-core peak against the 8-thread
  ratio the floor judges: zen4 88.0% -> 5.792x FAIL, zen5 58.9% -> 5.960x FAIL, gnr 42.5% -> 6.609x PASS. Monotone
  across all three, and the host keeping the most of its hardware at 8 threads (zen4, 65.9% of 8x peak against gnr's
  34.3%) is the one refused. Not an AWS artifact: Sgemm has never cleared 6.0x on any AMD host in any log here —
  vesta 5.708-5.823x, antares 5.477-5.766x — while both Intel hosts clear it. Six machines but one mechanism, so §5
  rule 10 makes that one witness with a microarchitecture split, which is the question wave 2 (#104) exists to ask.
  `gate-p4`'s own carried text predicts the hazard ("a ratio whose denominator this phase is chartered to change");
  what is new is that it now decides verdicts. The floor is unchanged pending a ruling — amending a criterion after
  seeing the result is not mine to do.
- **§5 rule 5's substitute clock instrument refuses on ordering alone, and did so on half of one host's readings.**
  With no cpufreq interface on a virtualized guest, stability is established by BenchmarkPeak at the head, middle and
  tail of the sweep, refusing any strictly-decreasing triple. `keel-gnr`'s four triples across the two runs: 245.25 >
  245.20 > 245.15 (refused), 245.10 < 245.15 < 245.25 (passed), 245.15 / 245.45 / 245.15 (passed), 245.20 > 245.15 >
  245.10 (refused). Total spread across all twelve windows is 0.35 GFLOP/s — **0.14%**, on a 0.05 quantum — and two
  of four strictly-decreasing triples is ~12% under exchangeable readings, so what reproduced was the test refusing,
  not a clock declining. Cost: gnr's P3 ratio in run 1 and its entire P5 row set in run 2, which is why neither run
  carries a complete gnr. The instrument prints `+/- 0.0%` intervals it then discards in favour of the ordering.
  Unchanged pending a ruling, for the same reason as the floor above.
- **On Zen 4 the fastest OpenBLAS kernel family is the AVX2 one, so "AVX2-or-better" is an ordering that does not hold
  on every host it grades.** `gate-p3` sweeps `OPENBLAS_CORETYPE` per host and pins the winner, and at `ce43bca` it
  pinned `Haswell` on Zen 4 (112.20 GFLOP/s) over every AVX-512 option (SkylakeX 107.90, Cooperlake 106.30) — Zen 4
  runs AVX-512 on a 256-bit datapath, so those kernels buy no throughput and pay their overhead anyway. `SkylakeX` was
  pinned on Zen 5 (+0.5%) and Granite Rapids (+2.1%), both margins the log itself reports as barely clear of the
  sweep's own same-family drift (1.30 against 1.10; 4.40 against 4.10). `provision-openblas.sh`'s allowlist is
  therefore a class check only, and the **pin** is what protects the denominator. *Retracted in the same breath:* the
  first version of this entry claimed the allowlist let an unpinned reference inflate keel's ratio by up to 5.2%. It
  cannot — the gate never divides by `DYNAMIC_ARCH`'s unaided pick. That claim was published without first running the
  instrument that already settled it (§5 rule 11), and that instrument's log refuted it two hours later. The
  measurement was sound and independently reproduces the pin's margin (+5.2% against the gate's +5.3% on Zen 4, from a
  different harness); only the conclusion drawn beside it was wrong. `docs/hosts.md` carries the void by name.
- **The launcher is a second witness of a host's identity, and an approved instance type alone no longer admits one to
  the judged tier.** Scott's judged-tier directive, 2026-08-19: *"spawn metadata required in provenance by the admission
  preamble."* The point is evidential, not bookkeeping. Every other field in a provenance line is *the host describing
  itself* — the circularity the rejected `assume_fleet` design was refused for — and `instance=`, though answered by the
  control plane at 169.254.169.254 rather than by the guest, is still a reading taken inside the guest over a link the
  guest could serve itself. `remote_probe` now splices a `spawn=` field read on the **driver's** side of the wire, from
  the EC2 control plane under the driver's own credentials: the one statement about a host's identity the host cannot
  author. Four tokens (`id:type:market` / `none` / `ambiguous` / `?`), and `?` must never read as `none` because "the
  launcher denies launching this" and "no launcher was consulted" are different facts, only the first of which is
  evidence about the host. `host_admission` gains a **market** conjunct at the same time, which is a criterion that
  became *readable* rather than a new rule: on-demand was already required but only *declared*, by a
  `KEEL_FLEET_MARKET` variable the launcher set and no gate read back, and `spawn list`'s `spot` boolean is the mechanism
  `remote.sh:212`'s own test asks for. A type **contradiction** between the two witnesses lands on `unknown` ⇒
  `unmeasured`, not `correctness`, because grading it would report "not full size" when what is broken is the instrument.
  Lab hosts are untouched: `instance=none` takes the bare-metal arm and never consults the launcher.
- **gate-p5's scaling criterion consults admission, which was the last judged criterion forming verdicts without asking**
  (#104). `docs/hosts.md` said admission was wired into gate-p2's 5b and *"not yet"* into gate-p3's or gate-p5's — stale
  for gate-p3, which has called `adm_judgeable` at two sites all along, and true for gate-p5. This is the gate the
  twelve-row re-measure runs under, so an unwired judged criterion here is a campaign that grades rows without consulting
  the class that decides whether they may be graded. A scaling floor is as much a claim about owned silicon as a
  percent-of-peak floor is: a partial-size guest's eight threads may sit on four cores it shares with tenants the run
  cannot see. The aggregate now goes through the shared `fleet_coverage` rather than a fourth inline absence chain (#90),
  which required counting floor misses instead of deriving them — the derived form printed "2 measured below it" for a
  fleet with one slow host and one that produced no ratio at all.
- **`tools/benchci` is the gate's summarizer, and it is benchstat plus resolution — measured, not asserted** (#110).
  Ruled 2026-08-19: *"the instrument's quantum exceeds every margin it adjudicated"*, so band-top arithmetic in the
  rounded domain can never make these verdicts measurement-decided and *"the instrument must gain resolution, and the
  resolution exists"*. `benchtab` is `internal` to `cmd/benchstat` and cannot be imported, so fidelity is *replicated*
  — `benchfmt` + `benchproc` + `benchmath`, benchstat's own default projections (`-table .config` with unit, `-row
  .fullname`, `-col .file`), `-confidence 0.95`, `benchmath.DefaultThresholds`, and the median-vs-mean assumption read
  from the file's own unit metadata — and then *proved*. `-verify` reruns the pinned `go tool benchstat -format=csv`
  over the same input and requires every center to match bit for bit and every CI to match after `%.0f%%` rounding,
  **in both directions**, because a cell benchstat summarized and this tool did not is a row silently dropped from every
  criterion that reads it. Result on the four archived raw logs: 10 cells each, **0 disagreements**, with CIs like
  `1.0847442793233681%` and `0.30286946459710257%` where benchstat printed `1%` and `0%`. What it deliberately cannot
  see (§5 rule 12): no `vs base` column, no p-values, no geomean, because no gate reads the CSV for those —
  `bench_compare` calls benchstat's *text* output, which this defect never touched and which stays pinned to benchstat.
  `bench_stat`'s parse is unchanged, so `bench_ratio_lo`'s formula is bit-identical and only its inputs got sharper.
  `gate-p5.sh` now fails if `-verify` fails, because intervals that do not reproduce benchstat are not benchstat's
  statistics at higher resolution — they are a second opinion about them.
- **`bench_csv` archives the raw samples, which is the prerequisite the ruling could not have named** (#110). The ruling
  said to read *"the archived raw bench output"*; there was none. `gate_tmpdir` put `BENCHLOG` inside a `mktemp -d`
  under an EXIT trap, so **every judged run in this project's history destroyed its own samples** and the rounded gate
  log is the only surviving record of all 22. Four lines at the single chokepoint all eleven call sites already went
  through. The path is *set* in `BENCH_ARCHIVE` and printed by the caller, never written to stderr: every gate relays
  that stream under a `benchci:` label, so announcing an archive there would arrive labelled as a measurement warning
  and fire each gate's "any warnings?" branch on every run.
- **`BenchmarkTrsmMB` sweeps `MB`, because the arithmetic moved #37's question** (#37). The diagonal solves are 1.59% of
  the *work* at m=n=4096, and charging the rank updates the full 166.05 GFLOP/s peak still leaves 69.1% of Strsm's
  measured time unaccounted for; two countable terms of that residual — solve flops `n·m·(MB+1)` and a `~n·m²/(2·MB)`
  rank-update repack, 5.37e8 elements at MB=64 — move in opposite directions with `MB`, which has never been swept.
  `TestTrsmMBCounts` and `TestTrsmSolveReplay` check the counts and the replay where the benchmark itself skips.
- **`tools/shapegen` is the microkernel shape generator, in-tree and verified at its mint** (#107). `-verify` re-emits
  the three shipped kernels and compares each against `internal/vec` by source text *and* by audit report — 74/16,
  50/8, 270/48 — and `-sweep`'s first run re-derives all nine rows of KERNEL.md §3 exactly, from an independent
  enumeration. Counted as apparatus, not library, in gate-docs' ratio (947 lines at `e1c6340`).
- **`scripts/readme-numbers.sh` writes README's published rates and their caption from a gate log** (#6). Ruled
  2026-08-19: *"the caption regenerates from the table, never the reverse"* — and an emitter is that ruling's executable
  form, because hand-assembling 24 rows again is the defect scheduled for recurrence, not the cheap alternative. It
  never measures. `docs-gen.sh` already refused to let `doc-site/numbers.md` be a hand-copy of README's `keel-numbers`
  block; that law stopped one file short of the numbers, and the last hand-maintained duplicate in the chain was the
  published one. What it cost: the block's own ratios put five rows on three hosts below the gate's scaling floors while
  the caption three lines below said the floor was missed *"on the two hosts that keep the most of their single-thread
  peak"*. The scaling verdict is **extracted, never recomputed** — the first draft re-derived it from the point estimate
  and named 3 shortfalls where the gate fails 5, because the gate judges net of CI; checking that the floor *constants*
  matched `gate-p5.sh` did not catch it, since the constants agreed and the predicate did not (DESIGN.md §5 rule 11).
  Net `scripts/` lines are ruling-mandated with the offsetting lift owed; for the ledger, this instrument retires a
  hand-maintained duplicate. Apparatus ratio **after**, read from `gate-docs.sh` with this file tracked: shell 11939 /
  library 8280 / **1.44×**. The 1.41× first written here was the ratio *before* — `git ls-files` does not count an
  untracked emitter, so the honest-accounting sentence beside it was measured on a tree that did not yet contain the
  thing being accounted for. Naming its own cost by 257 lines too few is the defect this entry claims to avoid.

- **`aws-fleet.sh` launches on-demand for judged runs** (`KEEL_FLEET_MARKET`, ruled 2026-08-17), tags each instance
  with its market and shows it in `status`. An unrecognized or empty market is refused, not defaulted; `KEEL_FLEET`
  specs are shape-checked, since a whitespace-only line passed `-z` and launched an instance with no role or type.
- **A host with no governor now has a clock instrument instead of an exemption** (§5 rule 5 as amended 2026-08-16):
  `clock_gate`/`clock_head`/`clock_post` sample `BenchmarkPeak` in three separate invocations either side of a
  sweep — `-count` is Go's inner loop, so a sweep's own peak rows are one contiguous window and are the *middle*
  only. Threshold-free by construction: every window bounded, and the three medians not monotonically declining.
  Live on the AWS guests, where `assert_governor` reads `absent`. gate-p1 is the one gate it cannot reach and
  says so — its sweep runs the root package's binary, which has no `BenchmarkPeak`.
- **The measurement fleet is AWS spot, launched by `scripts/aws-fleet.sh up|wire|status|down`** (#12): three
  right-sized guests (`c7a`/`c8a`/`c7i`, 8 physical cores each, read from `describe-instance-types` rather than
  inferred from vCPUs) with a boot-time `shutdown -h` dead-man switch, `down` selecting by tag rather than by a
  written list, and `up` refusing while a tagged fleet is alive. Measured on the live guests: `governor=absent`,
  AVX-512 on all three, and #82's premise real — AMD reports `smt=1`, Intel `smt=2`.
- **A remote measurement now outlives its ssh connection** (#62): `remote_exec` scp's a generated runner and
  supervises it with tmux on the far side, recovering the exit status from a status file. An unfinished run
  returns `vanished` (125), never a program code, and a missing supervisor is reported as `tmux=` in every
  gate's provenance instead of being assumed away. `scripts/remote-exec-test.sh` drives all four branches
  against `ssh localhost`, so none of it waited on a fleet.
- **Every gate reads free disk rather than assuming it** (#84): `require_disk` prints the headroom each run
  had and `unmeasured`s below a **512 MiB floor — measured**, at 3.8× a cold run's ~135 MiB demand, whose
  dominant term is a 76 MiB cold cross-compile cache and not the binaries. That is what makes the 137 MiB
  volume that filled a fit rather than a coincidence. A live peak is still owed; the printed line collects it.
- **First measurement of the AWS fleet: gate-p2 is RED, and P2's 55% floor has met Intel silicon for the first
  time** (#104, `blocked`). `keel-spr` (Xeon 8488C, Sapphire Rapids) puts 4x32 at 34.2% of a 228.9 GFLOP/s
  measured peak; `keel-zen4` 96.8%, `keel-zen5` 65.3%. Not a regression — retired `janus.local` measured 35.2%
  and passed because it classified `issue-bound` (43.8% floor); spr classifies `fma-bound`, so the flat 55%
  applies. Single reading: the repeat was **reclaimed by spot mid-run**, the first real reclamation here, and it
  landed inside a measurement exactly as `vanished`/`unmeasured` were built for.
- **gate-p5 cannot measure this fleet at all** (#66): a guest has neither `cpufreq/boost` nor
  `intel_pstate/no_turbo`, verified by ssh on two arms, so `remote_boost_set` exits 3 and `gate-p5.sh:664`
  skips every host. The `continue` is inside the loop holding criterion 9, so re-measuring the README rate
  block — the ruled precondition for the v0.1.0 tag — went dark. Fixed below.

- **`citation-lint.sh` now WARNs when a scoped quote marker names a form that is not on its line**, and
  the one dead token in the tree is gone. A scope is matched *literally*: the token `7.7` is a string,
  not a notation, so it does not suppress §7 rule 7 even though both spellings resolve to the same rule <!-- citation-lint:quote -->
  when the citation itself is checked. The header now says so in as many words. **The check is the point,
  though, not the note:** a mis-spelled or outdated token does not fail quietly in the safe direction, it fails *loudly
  with the wrong attribution* — the exemption misses, the site is checked as a live citation, and when
  that citation is itself broken the red line reads "bad citation" about a citation that is fine. Ruling
  of 2026-08-16: *"one loop over the markers, and the class closes instead of the instance."* The real
  finding was `CHANGELOG.md:801`, whose scope carried `4.3` after the `§4.3` it named was pushed to the <!-- citation-lint:quote(4.3) -->
  next line by a rewrap — the same wrapping hazard `T2` covers for bare markers, and the same
  exclusion-outliving-its-defect shape #93 closed on the mkdocs side, since a token also goes dead with
  no edit at all when the quoted text is reworded.
  **Reported, not failed, and therefore relayed:** nothing is unsound when a scope is dead, so this is a
  WARN — which made `gate-docs.sh`'s citation stage a hiding place, because it filtered the lint's output
  down to the summary line and every warning the lint can emit appears only on a run that passes. It now
  relays `WARN` lines, driven on purpose: with the `4.3` token reinstated the gate prints the warning and
  still reports GREEN. `T11` is the control (`0 dead marker scope` asserted on the clean path, so the
  dozen lines in this tree that merely *name* the marker are pinned as mentions), and it needed a third
  assertion form in the harness, `clean:PATTERN` — rc 0 **and** the text present, because a control that
  checked only the exit status would pass on a build where the warning was never printed at all.
  **Two limits, stated rather than discovered later:** only *scoped* markers are audited, since a bare
  one has no spelling to get wrong and, once the citations on its line are deleted, is indistinguishable
  from a line naming the marker; and a marker on a citation-free line is read as a mention. The one
  reachable false positive — a line that both cites a rule and quotes a scoped marker verbatim, argument
  included — opts out with `citation-lint:nomarker`, whose scope is the audit and nothing else: `quoted`
  parses only the *first* scoped marker on a line, so a genuine marker written after a quoted one is
  unreachable, which was measured rather than assumed. Such a line cannot suppress its own citations by
  any spelling, and the fix is to wrap it. Both branches are driven — `T11` asserts the warning fires,
  `T12` appends the declaration to the same line and asserts it goes quiet — because a `nomarker` nobody
  exercises is the dead configuration this whole check exists to report.
- **A documentation site, and a gate that will not let it publish a number nobody measured** (#92).
  `mkdocs.yml` + five user pages under `doc-site/` (Home, Usage, Capabilities & limits, Numbers,
  Troubleshooting) and **one** nav entry at the end for the project records. The ruling that shapes it,
  2026-08-16: *"users will not care about the design notes. They want straight usable information"* — so
  DESIGN.md, the §5 methodology, the toolchain field notes, KERNEL.md, CHANGELOG and CONTRIBUTING are
  present, linked once, and never in the user path, and nothing on a user page summarises them.
  **The records are served by symlink, not copy** (verified empirically that mkdocs reads through a
  symlink inside `docs_dir` before the design depended on it): the page *is* the file, so there is no
  second copy to drift, and they are gitignored so the tree does not gain a second path to DESIGN.md.
  **The numbers page is an extraction, not a transcription** — `scripts/docs-gen.sh` lifts the table out
  of README.md's `keel-numbers` block, the same block gate-p5 criterion 9 re-measures on all three
  benchmark hosts against `README_TOL=0.05`, which is what puts the site's figures under that gate; a
  rate typed onto the page by hand would be under nothing. Its context lines are *counted* from the block
  (24 rows, 3 CPU models, rev `083cbdb`) rather than typed, and it carries exactly one link out, to the
  methodology.
  **The generator fails closed on nine malformations and the gate drives all nine on purpose**
  (`scripts/gate-docs.sh`): missing or reversed markers, an empty block, no table, no header separator, a
  row whose cell count disagrees with the header, a `[synthetic]` stamp anywhere in the block, and a
  missing provenance rev — plus a well-formed positive control, because "everything was rejected" is also
  what a generator broken in some unrelated way looks like, and two more for the methodology extraction
  (no §5; a §5 with no numbered rules). A fail-closed check nobody has watched fail is a claim, not a
  check. The gate is those checks plus `mkdocs build --strict` plus `citation-lint.sh`, and CI's
  `docs.yml` calls the script rather than restating it, on hosted runners only.
  **Two defects found by rendering rather than by reading.** The home page's card grid needed
  `md_in_html`; without it `<div markdown>` was emitted verbatim and the markdown inside it served as
  literal text — and `--strict` said nothing, because links it never parses are links it cannot validate.
  And every one of the 25 `[Tn](#tn)` cross-references in `docs/toolchain-notes.md` pointed at an anchor
  that does not exist — a defect in the record, not in the site, fixed below (#93). `mkdocs.yml` now sets
  `validation.links.anchors: warn`, which is **not** mkdocs' default: the default is `info`, printed and
  passed, and a message with no verdict attached is why 25 dead links survived to a site build.
  **The deploy job is written and not enabled**: it is gated to `workflow_dispatch` or a `v*` tag, so it
  is skipped on every push and pull request, and it will fail rather than publish until Pages is turned
  on — enabling Pages and flipping the repository public remain Scott's actions at tag time.
- **The project graphics, folded into the README, the site's home page and the site's chrome** (#92).
  `doc-site/assets/` holds one copy of each: `keel-hero.webp` (1600×533, the logo and tagline banner) at
  the top of both `README.md` and `doc-site/index.md`, `keel-mark-white.png` as the header logo,
  `keel-favicon.png`, and `keel-social.jpg`. **One copy, per the site's own law** — README points at
  `doc-site/assets/…` and the pages at `assets/…`, so there is no second path to an image and no symlink
  or gitignore entry to keep in step.
  **Format is chosen by who consumes the file, and the bytes were counted because they ship in the module
  zip.** Everything tracked in the module root is downloaded by every `go get`, which is not true of a
  normal site's images, so the hero is webp at 78 KB against 745 KB for the same pixels as a truecolour
  PNG. `keel-social.jpg` is JPEG. The four images are 183 KB together.
  **GitHub has two image paths that do not accept the same formats, which is the whole reason those two
  files differ**, and checking a claim already written down is what surfaced it. A file *committed* and
  embedded by relative path is fetched from `raw.githubusercontent.com`, which serves `.webp` as
  `image/webp` — verified against a public repo's tracked webp, because this repo is private and its own
  rendering cannot be read yet — so the hero renders. A file *uploaded* through the web UI goes through
  the attachment allowlist, which `docs.github.com` states as PNG, GIF, JPEG and SVG, **no webp**; the
  social-preview upload is that path. The trap is that "GitHub supports webp" and "GitHub does not" are
  both true of one path and false of the other, so either sentence applied to both is wrong in one
  direction. The first draft of this entry asserted the permissive half for both.
  **PSNR picked the wrong encoding, and looking at the pixels overruled it.** A 256-colour PNG measured
  *better* than webp q92 (40.3 dB against 39.2 dB) and was visibly worse: this artwork is mostly a
  near-white gradient field, where palette quantisation spends its error on dither speckle that the eye
  reads as grain, while webp spends it on a softening of a faint dot grid that is invisible at display
  size. Two encodings with the same error budget are not equally wrong, and a single scalar cannot say
  which — the decision was made from a 300%-magnified crop of the smoothest region of both.
  **A white silhouette needs a dark header, so the header is now the artwork's navy** (`#04274c`, the
  most common ink in the mark) via `primary: custom` and `doc-site/assets/extra.css`, which is Material's
  documented mechanism for a brand colour rather than a workaround; the accent moves from indigo to teal
  to match. **That override introduced a defect visible only in the theme's built stylesheet.** Material
  derives `--md-typeset-a-color` from the primary colour, so links became navy — 15:1 against white and
  therefore perfectly readable while being almost exactly as dark as the body text they must be
  distinguishable *from*; contrast was never the failing quantity. Worse, `palette.css` sets slate's link
  colour to the primary and then overrides it per *named* primary (pink, blue, teal, …), none of which
  `custom` matches, so **dark mode inherited the navy at 1.07:1 — links that cannot be seen at all.** A
  build is green either way and no page source shows it. Both are now stated explicitly: `#19608c` at
  6.8:1 in light mode, the brand teal at 5.7:1 in slate.
  **The gate already covers this, and was made to prove it**: `mkdocs build --strict` fails on an image
  path that resolves to nothing, driven on purpose by typo'ing the hero's filename (`FAIL mkdocs build
  --strict failed`, naming the link) before the assets were trusted. `keel-social.jpg` is referenced by
  no page deliberately — it is the source for the repository's social-preview image, and uploading it is
  Scott's action at tag time, like enabling Pages.
- **`example_test.go`: five runnable `Example` functions, which is how this project will document a call
  from now on** (#92). `ExampleSgemm`, `ExampleSgemm_submatrix`, `ExampleSaxpy`, `ExampleSaxpy_stride`,
  `ExampleTranspose`. The rule behind the form is §5 rule 8 — *a summary is a cache with no invalidation
  protocol* — applied to documentation: a prose code block is a cache of the API and nothing invalidates
  it, while an `Example` is compiled and run by `go test` and cannot silently stop matching the package.
  It collected immediately: `ExampleSaxpy`'s hand-computed `// Output:` was wrong in the last element
  (44 for 40 + 2·4) and the first run said so. Every example uses small integer-valued matrices, because
  integers in that range are exact in float32 and so is any sum of them that stays in range — an example
  whose expected output depended on the summation order would be an example that fails on one backend.
  Two of the five go past the three that were asked for: `_submatrix` makes the `lda > n` contract
  runnable rather than pictorial, and `_stride` does the same for `n` versus `len(x)`. Those are the two
  parts of this API a caller gets wrong first, and both were prose-only before.
- **`scripts/fakessh` + `scripts/fakessh-test.sh` + `scripts/exercise-dead-host.sh`: a dead host,
  induced environmentally, to fire gate-p2's fleet-incomplete aggregate once on purpose** (ruled
  2026-08-16). Criterion 5b's PASS reads *"every host that produced a judgeable throughput reading
  cleared its floor (N/M)"*, and on this fleet it has only ever printed 3/3 — so the rendering where a
  **green line credits a proper subset of the fleet** has never executed. A host will be down someday;
  its first firing should not double as its first test. `fakessh` goes on `PATH` as `ssh` and refuses
  one host with ssh's own exit 255 and "No route to host" while the other two genuinely build, run and
  measure, so the gate discovers the outage through the same machinery a real one would use, unmodified
  and unaware. **A flag telling the gate to skip a host would have proven only that the flag works.**
  `KEEL_INSTRUMENT_EXERCISE` exists but is deliberately impotent — it stamps every line `[synthetic]`,
  prints a banner, withholds GREEN/RED, exits 2, logs to `build/instrument-exercise-*` (#78), and is
  read nowhere near a comparison, threshold, tally or host list; set it on a healthy fleet and the run
  reports a stamped 3/3, which puts on record that the aggregate cannot be forged from a flag. The shim
  has 16 fixtures of its own, run before any host time is spent: the substring cases are the ones that
  matter, since a dead `zen4.local` matched loosely would also kill `zen4.local.backup` and the log
  would still say "one host unreachable". **The shim covers `scp` because the guard made it.** The
  first version shimmed `ssh` only, on the belief that everything crossed the wire through ssh's
  stdin; `remote.sh:440` copies the bench binary with `scp`, so the dead host would have answered that
  call and the exercise would have been a fiction with a green-looking log. The driver's transport
  assertion caught it on the first launch, before any host time was spent -- which is the argument for
  asserting a premise you are confident about. The guard now enumerates what is covered rather than
  asserting an absence, so a transport added later fails loudly instead of quietly reaching a host
  declared dead, and refusal is per-transport (ssh exits 255; scp exits 1 after "lost connection", so
  a caller checking for 255 cannot read a dead host as a copy that merely failed). The driver also verifies the branch actually fired rather
  than reporting that the run happened, checks `remote.sh` has not grown an `scp`/`rsync` path the
  ssh-only shim would miss, and prints whether the dead host is also the sentinel so the collateral
  scope is computed rather than inferred. Doubles as the fleet's first degraded-mode rehearsal.
  **`DESIGN.md` has cited "the `fakessh` dead-host arm" as established discipline since `1c3eaca`, and
  no `fakessh` existed until now** — the comparison was written from intent and read as precedent for
  four commits. It is annotated as such in place and made true, rather than repointed: a citation to a
  mechanism that does not exist is what the citation lint was built for, occurring in prose the lint
  does not reach.
  **It ran, and the branch fired** (`build/instrument-exercise-dead-host-5ade3ff.log`, 132 lines):
  `antares.local` unreachable, vesta and janus genuinely measured and cleared their own floors
  (96.6% of measured peak FMA-bound; 94.8% of a 48.6% issue roofline), and criterion 5b printed the
  rendering that had never executed — a PASS crediting `2/2` with the third host reported separately as
  UNMEASURED. Discipline audited from the log rather than asserted: **25 of 25 verdict lines stamped
  `[synthetic]`, zero unstamped, the only verdict token printed is `VERDICT WITHHELD`**, `GREEN`/`RED`
  never emitted, exit 2. And it paid for itself immediately — the line it fired is #90, above.
  **The target then moved, because the ruling deleted the branch.** The PASS quoted above no longer
  exists; a partial fleet resolves to UNMEASURED, so the driver's read-back now looks for that rendering
  and treats the old `(2/2)` PASS as a finding — post-#90 it is unreachable, so seeing it would mean
  either the outage was not induced or `fleet_coverage` is wrong. The read-back keys each of criterion
  5b's four branches on a phrase of its own and reports NO when nothing matches, which is the fail-closed
  direction: the previous version keyed on the exact wording the ruling removed and would have reported
  INDETERMINATE on the very run meant to prove the fix. A second exercise run at the new revision is
  owed to the code that replaced the branch, and the first run's log stands as the record of the finding.

- **Issue #22's candidate C: the fringe add-back is vectorized, behind the kernel's own
  dispatch.** A tile that crosses the edge of the matrix or the edge of a triangle is
  computed at full MR×NR into a scratch tile, and only its live sub-rectangle is added back
  into C; that add-back was a scalar loop and is now `vec.AddTile512` / `vec.AddRow512`, with
  `vec.ScalarAddTile` / `vec.ScalarAddRow` as their executable spec. The kernel and the
  scratch tile are untouched, which is the point: C costs the P2 zero-spill audit nothing,
  where candidate B — a masked C update inside the microkernel — would have doubled the
  audit surface for a coverage subset. B stays unbuilt pending a measured gap.
  `AddTile`/`AddRow` hung off `kern.Kernel` rather than a package-level var in
  `internal/block` so that `KEEL_FORCE=scalar` would force the add-back too (nil being a
  registration bug that panics on the first fringe tile, because a silent scalar fallback
  would let a shape that forgot to populate them measure as if it had) — **those two fields
  are gone again now that A ships**, and the requirement they carried is inherited by C′;
  see the closing note below. Two live-region
  shapes, two call counts: a rectangular fringe takes `AddTile` (one indirect call per tile,
  row loop inside the callee), and a mask-crossing tile takes `AddRow` per row, because a
  diagonal tile's live window is a different `[lo, hi)` on every row and no single rectangle
  covers it. Differential tests are **bit-equality, not `oracle.Tolerance`** — every output
  element is the sum of exactly two inputs, so there is no association to choose and a
  disagreement would be about one IEEE add — and they cover a guard sentinel past the live
  window (the property a full-width store would break silently), non-finite inputs including
  the ±Inf/NaN a zero-padded panel produces, and `ldc > jn` always, since a helper ignoring
  `ldc` would pass any test with `ldc == jn`. Green on AVX-512 hardware (Zen 4), along with
  the root, `internal/block` and `internal/kern` suites; the scalar twins are exercised by
  every Level-3 oracle test on the dev host. One consequence stated rather than discovered:
  the masked tail puts an `archsimd` partial op on a Level-3 path for the first time, so
  T17's `-race`/`-d=checkptr` fatal now reaches Level 3 as well. No gate moves — those four
  criteria are already unmeasured on all three hosts for exactly this reason (#42, fixed
  upstream by CL 761120 in go1.27) — but the surface is wider.
  **Measured, and it does not win** (`build/edge-fba229f.log`, 180 lines, three hosts,
  `-count=10`, `GOMAXPROCS=1`, A=`757acb8` vs C=`fba229f`): geomean −0.87% on Zen 4,
  **+0.34% on Skylake-X and +1.52% on Zen 5**, i.e. one host slightly better and two
  slightly worse, over per-shape swings of −7.92% to +8.19% that agree across hosts on
  almost nothing. The interior controls bound the unreadable band at 1.14%: five of the six
  control readings resolve statistically and **disagree in sign across hosts**
  (2048³ wash/+0.69/+0.10, 4096³ −0.69/−0.39/+1.14), which is T22's function-alignment
  signature rather than a harness defect, since at those sizes m and n are multiples of MR
  and NR and neither arm can enter the fringe branch at all. Read above that band, C wins
  the small square fringes (m=31 −3.83/−2.26, m=63 **−7.92** on Zen 4) and loses the thin
  ones (m=5/n=2048 **+8.19** on Zen 5, +5.69 on Skylake-X; m=2048/n=33 +6.72 on Zen 4).
  That is the risk this file's own header predicted before the run — "for a 4×32 tile the
  whole add-back is at most 8 full-width ops, so an indirect call per row could plausibly
  cost more than the scalar loop it replaced" — so the indirect call, not the vector
  arithmetic, is the term to attack.
- **#22 ruled and closed: A ships, C is archived as measured-correct-and-slower.**
  `internal/block` goes back to the inlined scalar add-back it had at `757acb8`, and the two
  `kern.Kernel` function fields are removed rather than repointed at the scalar twins —
  repointing would have left the *incumbent* paying the per-row indirect call that sank C, so
  keeping A means keeping A as measured. `internal/vec/edge_amd64.go` and its four
  differential tests stay in the tree, called by nothing, because C′ starts from that arm and
  the tests are what keep it from rotting into a wrong starting point (in particular the
  masked-store guard sentinel: with no caller, nothing else would notice that property
  breaking).
  **Ledger, so a future reader need not re-derive it.** A: incumbent, retained. C: correct,
  measured, loses; mechanism *located* — the per-row indirect call, densest on thin shapes.
  C′: the named follow-up, this arithmetic with the call removed (monomorphize per backend,
  or hoist the dispatch out of the row loop so it amortizes per tile), keeping the per-row
  live window C had and re-establishing `KEEL_FORCE=scalar` forcing the add-back. B (masked
  C update in the microkernel): still unbuilt, and one rung further back than the first
  reading of this run suggested — the run localized *C's* loss term, and never isolated A's
  fringe cost against a fringe-free baseline, so "A leaves resolvable edge cost on the
  table" is unestablished and B's trigger as worded is **not** met. If C′ wins, B is moot
  forever (superset coverage, no hot-loop branch); if C′ also loses to a scalar loop, the
  vector-add-back thesis is dead on its own terms and B is the last live question.
  Reopening trigger: post-green, or skinny-GEMM performance becoming a criterion someone
  states. The campaign closes by decision at ±1% geomean on fringe shapes, with the 1.27
  floor having resolved its original admissibility purpose.
- **#22's wash criterion amended, on grounds measured before the run it judges** — "controls
  within the host's between-binary layout floor", not p > 0.05, in `bench/edge_test.go` and
  `scripts/edge-bench.sh` where the criterion lives. The floors are **1.71% (Zen 4), 0.99%
  (Skylake-X), 1.32% (Zen 5)**: the largest resolved |sec/op| excursion of the layout
  ensemble's *control* routine, whose code is byte-identical in both its binaries
  (`build/layout-ensemble-e829a61.log`, #54/#61) — the same position the interior controls
  are in here, since at n=2048 and n=4096 neither arm enters the fringe branch. Every
  control excursion in the A/C run is inside its host's floor, so the run stands. A p-value
  was the wrong instrument for this question and the project had already shown why:
  statistical resolution and attribution decoupled under #54/#61, and that campaign measured
  the size of the gap. The amendment imports a standard that predates the measurement it now
  judges, which is what distinguishes it from a criterion rewritten around its own result;
  the corroborating signature is that the sub-floor control deltas **disagree in sign across
  hosts**, as placement does and a shared-code mechanism would not.
- **`docs/toolchain-notes.md` T25 — four spellings of the same SIMD loop, 36 instructions
  versus 13** (#74). Found writing the above, whose loop is one vector add and three memory
  ops, small enough that spelling dominates object code. In descending order of size:
  `Load512(x[j:])` keeps `archsimd`'s own `CMPQ $16`/panic *and* a five-instruction
  conditional pointer advance per operand where `Load512(x[j:j+Lanes])` folds both; the
  guard `j+Lanes <= len(x)` keeps a bounds check where the identical `j < len(x)-Lanes+1`
  does not; `dst = dst[:len(src)]` does not remove the survivor but **moves it onto the
  resliced operand**, and reslicing `src` instead swaps which one keeps it; and hoisting one
  loop-invariant limit is free while hoisting both into a `min` local puts both checks back.
  Identical on go1.26.6, go1.26.5 and go1.27rc3. This **partly corrects T19**, which recorded
  the `i+4 <= len(x)` miss, prescribed the slice-advancing rewrite, and was adopted by all
  ten `internal/l1` loops: the strict-`<` guard is a second remedy that keeps the loop
  indexed. No l1 loop is wrong — they use `x[0:16]` and so never paid the largest property —
  but the note read as if its prescription were the only one. Nothing is filed upstream and
  nothing should be: the class is known and open (golang/go#17370, #25197, #28941, with
  #80146 the fix in flight), a second repro of a known miss earns no comment, and the number
  that would — a *time*, not a static count on a memory-bound loop — is what #74 is open for.
- **`bench/edge_test.go` and `scripts/edge-bench.sh` — the fixture and harness for #22's
  edge-handling ranking, landed ahead of the candidate they measure.** Not a gate: the
  script certifies nothing, moves no criterion and exits 0 whatever it reads; its product
  is a per-host A/B table. The ordering is forced by the mechanism, not preference —
  `edge-bench.sh` builds its base arm from a detached `git worktree` at `BASE_REF` and its
  new arm from the working tree (the same shape as `l1-bench.sh`, and for the same reason: a
  stash would mutate a tree another measurement may be reading), so the benchmark must
  already exist at `BASE_REF` before any candidate can be measured against it. Committing
  the fixture first is what makes the comparison possible at all.
  **The fixture is the part that could produce a wrong verdict.** At 2048³ the candidates
  are byte-identical almost everywhere they execute — 2048 is a multiple of both MR (2 or 4)
  and NR (32), so Sgemm's fringe branch is entered zero times, and an A/B measured there
  reports the layout floor with the edge code never running, which is #48's tautology trap
  in a new costume. So: seven ragged gemm shapes straddling MR and NR at large k
  (31/33/63/65 square, m=3 and m=5 at n=k=2048, and one large ragged 2048×33×2048), plus the
  **mask-crossing** shapes that are the half of candidate C's coverage a masked microkernel
  structurally cannot reach — a diagonal tile's live region is a per-row `[lo, hi)`, so
  Ssyrk and Ssymm take the scratch-tile path at *every* size, including edge-free ones.
  `Ssyrk n=2048` is therefore an edge-heavy case that looks like an interior one, and it is
  criterion 7's own numerator, so a C win there moves the 85%-of-Sgemm ratio `gate-p4`
  grades. Last come the **interior controls at n=2048 and n=4096, which are the named
  falsifier**: the candidates differ only in a branch those shapes never take, so if they
  differ there the harness is measuring something else and the run is void — the script says
  so in its closing line and the controls are read first.
  Three work declarations are new (`gemmWorkMNK`, `syrkWorkNK`, `symmWorkMN`) rather than
  widening `gemmWork`, which is the shared declaration that makes `BenchmarkSgemm` and
  `BenchmarkOpenBLAS` provably one numerator; putting a third caller between those two would
  cost more than it saves. As everywhere else in this package the flop counts are *useful*
  flops, not executed ones — counting the discarded half of a diagonal tile would hide the
  exact cost this fixture exists to measure.
- **The parallel nest (P5).** The Level-3 routines now distribute their work over a
  bounded pool of goroutines sized by `runtime.GOMAXPROCS(0)`, started per call and
  joined before the call returns. GOMAXPROCS is the only knob; there is no keel-specific
  one, because it is the knob a caller already has and already expects a Go library to
  respect.

  Three properties are contract, and each has a test that fails on it rather than a
  comment claiming it:
  - **The result is bit-identical at every thread count.** The parallel axis is the MC
    (`ic`) loop, which partitions C by row panels and reassociates no single output
    element's sum, so parallel equals serial *exactly* — not within a tolerance. A float32
    BLAS whose answer moved with the core count would be a different library on every
    machine. Checked bitwise at GOMAXPROCS 1, 3 and 8 over five shapes covering all four
    routines — both Strsm sides, because the two sides split different axes and so reach
    the blocked nest with different ic-block counts at the same GOMAXPROCS
    (`TestP5Determinism`; 3 threads because a row-partition off-by-one hides at every
    power of two).
  - **Nothing outlives a call.** No resident pool, no background goroutine, and a repeated
    identical call returns identical bits (`TestP5NoState`) — the second half being what a
    pooled packed-A buffer could break.
  - **GOMAXPROCS=1 is the serial nest, in the calling goroutine**, with no goroutine, no
    atomic and no scheduling in the path (`internal/par.TestRunSerialUsesNoGoroutine`).
    Every keel number published so far was measured at GOMAXPROCS=1, so the whole prior
    campaign's denominators are untouched by construction rather than by re-measurement.

  New internal package `internal/par`: per-call workers claiming units dynamically, with
  each worker's *first* unit assigned statically so the returned worker count is exact
  rather than a race outcome. Dynamic claiming rather than a range per worker because
  Ssyrk's `ic` blocks differ in work by a factor of the block count; for a lower mask the
  claim order is reversed, which is longest-processing-time-first with the sort replaced by
  the one fact the mask already tells us.

  **Trsm parallelizes on a second axis, and this is an extension of DESIGN.md's
  instruction rather than a silent deviation.** DESIGN.md names the MC loop, but Trsm's
  rank update is MB×n×k with MB=64 — fewer rows than one MC block — so that loop has
  exactly one iteration there and splitting it would yield one worker whatever GOMAXPROCS
  says. Trsm instead splits its right-hand sides at the top level (columns of B for a left
  solve, rows for a right one), which are fully independent, and runs the rank updates
  inside `noSplit` so the two levels cannot multiply. That is also why `gate-p5.sh`
  judges Trsm as a separate parallelism class (#37).

  `noSplit` runs those updates through `par.Serial`, which exists because the first version
  wrote `par.Run(1, body)` — and `Run`'s argument is the number of *units*, so that ran ic
  block 0 and silently dropped the rest. Right-side Strsm at m=n=500 came out wrong by
  4.2e-2 against the float64 oracle at GOMAXPROCS ≤ 2 and right by 5.6e-7 at GOMAXPROCS=8,
  since a wide pool cuts each strip below MC and `ceil(m/MC)` really is 1 there. It never
  shipped, and it is recorded because the sweep should have caught it and could not: the
  large-size reduction in `TestStrsmSweep` cannot reach a right-side corner on any host
  here (#64). The right-side shape is now one of `TestP5Determinism`'s five.
- **`keel.Workers`, `keel.WorkersLastCall`, `keel.GOMAXPROCS`** — the pool's sizing rule,
  the worker count the last Level-3 call actually used, and GOMAXPROCS as the nest reads
  it. Instrumentation, not configuration: `WorkersLastCall` exists because a benchmark row
  named `threads=8` that silently ran on one worker reports a 1.0× ratio, which reads as a
  performance problem when it is a measurement failure — and the two want opposite
  responses. Only the library can answer it, so the library answers it.
- **`Scale/{Sgemm,Ssyrk,Ssymm,Strsm}/n=4096/threads={1,8}` benchmark rows**, the input to
  P5's headline criterion. Both thread counts run in one process (a run per thread count
  would put the two arms of a ratio in two page-cache states and two frequency histories),
  each row sets GOMAXPROCS itself and declares what it got, and each carries its numerator:
  new `keel-bench-threads` markers alongside the existing `keel-bench-flops`, with
  `symmWork` (2·m·n·k — symm's saving is memory, never arithmetic) and `trsmWork`
  (n·m·(m+1), left side). `gate-p5.sh` recomputes both and fails on disagreement.
- **`block.TrsmWork`**, the parallelism model P5 requires beside Strsm's measured scaling
  (#37): how the useful flops divide between the blocked rank updates and the serial
  diagonal solves, walked over the same MB partition Trsm walks. Absolute counts, not
  fractions, so a caller can check their sum against the total before dividing by it — a
  pair of fractions summing to 1 is self-consistent no matter how wrong both are, and
  `TestP5TrsmModel` checks the sum against the gate's own n·m·(m+1) at three shapes,
  including a ragged one, before printing either fraction. At 4096, left side, MB=64:
  rank_update=0.98413, diag_solve=0.01587.
- **`pack.BPanelsPart`/`pack.NPanels`, and `block.packB`: the shared B pack is parallel**
  (#65). The pack is O(kc·nc) against the region's O(m·nc·kc), which is the argument for
  leaving it serial and it is wrong — it sits *between* two parallel regions with every
  worker idle, and Amdahl does not care what fraction of the work it is, only what
  fraction of the time. At n=k=4096 with NC=4096 and KC=384 the serial version was eleven
  single-threaded copies of 1.57M floats inside a call whose parallel part takes ~90 ms at
  eight threads. Panels are a partition of the buffer, so the split needs no lock, no
  ordering and no reduction, and the result is bit-identical because packing copies and
  scales rather than accumulating. `BPanelsPart` takes the whole buffer and checks it
  against the whole block's `BLen`, so a partition off-by-one panics rather than leaving a
  panel holding whatever the pooled buffer held last.

  The measurement that identified it is the first gate-p5 run on the nest
  (`build/gate-p5-175098d.log`, `175098d`): **Sgemm, Ssyrk and Ssymm missed the ≥6.0×
  floor on all three hosts and Strsm cleared it on all three** (6.93–7.12× net of CI) —
  and Strsm is precisely the routine whose parallel region *encloses* its packing rather
  than being enclosed by it.

  The second run (`build/gate-p5-083cbdb.log`, `083cbdb`) measured the fix. Change in the
  8-thread scaling ratio, net of CI:

  | routine | Ryzen 9 7950X3D | i9-9960X | Ryzen AI MAX+ 395 |
  | --- | --- | --- | --- |
  | Sgemm | +2.0% | +3.2% | +3.5% |
  | Ssyrk | **+32.6%** | **+21.4%** | **+38.3%** |
  | Ssymm | +0.8% | +6.2% | +1.2% |
  | Strsm | +1.0% | −2.6% | −1.1% |

  Ssyrk now clears the floor on all three hosts and Skylake-X clears all four routines;
  the scaling criterion went from 8 failing rows to 4, and the gate from 51 PASS / 15 FAIL
  to 54 / 12. **The first explanation written here was wrong and is withdrawn.** It said
  Ssyrk pays the same pack for half the flops, which predicts Ssyrk gaining ~2× what
  Sgemm gains; measured, it gained 16.2× / 6.7× / 10.9× as much. The term that does
  explain that spread is which pack branch each routine takes. A NoTrans Ssyrk is
  dispatched as `gemm(kn, trans, !trans, …)`, so its B operand is Aᵀ and its B pack runs
  the transposing element-at-a-time loop, while Sgemm NN and Ssymm pack B contiguously
  through `copy()`. Parallelising a scalar transpose buys an order of magnitude more than
  parallelising a memmove — which also means the remaining four misses have a different
  cause, and #66 rather than this bullet carries it. Strsm's two small regressions are
  within the run-to-run spread of a ratio whose parallel region this change does not
  touch.
- **README publishes its first measured rates** — 24 rows, four Level-3 routines × 1 and 8
  threads × three CPUs, all from the single `083cbdb` gate-p5 run, keyed by CPU model and
  never by hostname. The rows live inside the `keel-numbers` block that `gate-p5.sh`
  re-measures on every host it runs on, so a stale row fails the gate rather than aging
  quietly; the same criterion also fails on a row with an empty denominator column and on
  any rate published outside the block.

  Every row's denominator is keel's **own** AVX-512 microkernel peak from the same run,
  because no OpenBLAS reference was taken at these thread counts — stated in the README
  rather than left to be inferred from a missing column. And that denominator is measured
  on an idle machine, so multiplying it by 8 asks the parallel nest to beat a single-core
  boost clock it never runs at; the 8-thread percentages are therefore a floor on the
  nest's efficiency, not an estimate of it (#66).
- **`vec.AbsMask512`/`AbsWith512` and the 256-bit pair**, so a caller can build the abs
  sign-bit mask once and reuse it across a loop. `Abs512` now delegates to
  `AbsWith512(x, AbsMask512())`, which keeps the sign-bit trick written exactly once and
  leaves the existing differential test against `ScalarAbs` covering both spellings.
  `internal/l1`'s two `Asum` kernels hoist the mask by hand, as `Axpy` and `Scal` already
  do for `alpha`'s broadcast: the compiler will not lift it, because LICM does not lift SIMD
  ops (#54, T18, `golang/go#79984`). Correctness verified on all three hosts across all
  three backends (12/12 host×backend combinations). The kernel goes 222 → 185 instructions.

  **Measured through `scripts/layout-ensemble.sh` at four link-order placements per host**
  (`build/layout-ensemble-e829a61.log`), because a ~3-instructions-in-25 change sits inside
  the 45% layout envelope T22 demonstrated on Zen 5. `Sasum` sec/op at n=4096, base → new,
  one column per placement:

  | host | pad=0 | pad=3 | pad=6 | pad=9 | floor (control, this size) |
  | --- | --- | --- | --- | --- | --- |
  | Ryzen 9 7950X3D (Zen 4) | −18.61% | −18.62% | −18.04% | −17.43% | 1.71% |
  | i9-9960X (Skylake-X) | −13.52% | −13.45% | −13.05% | −13.42% | 0.20% |
  | Ryzen AI MAX+ 395 (Zen 5) | −0.53% | **+1.41%** | −0.74% | **+0.95%** | 0.44% |

  So the hoist is worth **−17.4…−18.6% on Zen 4 and −13.1…−13.5% on Skylake-X** at 16 KiB,
  sign-consistent across every placement and 10× and 67× the floor its own control sets at
  that size — or 10× and 14× against the host-wide control maxima (1.71% on Zen 4, 0.99% on
  Skylake-X), which is the conservative denominator. It is **not attributed on Zen 5**,
  where the sign follows the kernel's
  entry alignment mod 64 rather than the source change: both mod-64 = 32 placements are
  negative and both mod-64 = 0 placements are positive, at n=4096 and again at n=256
  (−3.75/−3.71% against +9.43/+8.26%). The remaining term is the change's own interior
  geometry — 185 instructions lie differently against the 64-byte lines than 222 do — which
  is part of the change rather than a confound, but it means the win is not portable to
  Zen 5 by anything the source can express. By regime on the two hosts where it is
  attributed: −3.24…−4.40% (Zen 4) and −4.70…−4.86% (Skylake-X) at 1 KiB, the figures above
  at 16 KiB, −4.7…−9.4% at 256 KiB on Skylake-X but at or below the floor at that size on
  Zen 4, and nothing at 4 MiB on any host. It pays where issue slots bind, not where
  bandwidth does. One cell is excluded as unexplained rather than folded in: Skylake-X at
  4 MiB reads +0.31…+0.63% at three placements and +10.46% at the fourth, with its control
  at +0.07% in the same cell — that is #60's anomaly territory and this change does not get
  to annex it.

  For calibration: 222 → 185 bounds the *code change*, not its throughput consequence. The
  earlier "~3 instructions in 25, so ≤12%" reasoning here was wrong, because it assumed
  removed instructions are fungible with the remainder — true only under uniform issue
  pressure. Retires with #54 when CL 803220 lands.
- **All six gates now assert `git worktree list` as well as `git status`** (#63). `git status`
  sees uncommitted changes and nothing else; a registered worktree is a second checkout of
  another commit inside the same repository and was invisible to every check we had. It is a
  hazard twice over: a stray build or path glob can read the wrong revision's sources out of
  it, and a later session finds it unable to tell instrument residue from a live measurement —
  the "is this a result, or wreckage?" ambiguity DESIGN.md §5.6 exists to eliminate. A
  worktree here usually means an `l1-bench.sh` or `layout-ensemble.sh` run is in flight, which
  is precisely the condition that should stop a gate rather than an exception to carve out
  for: the tree is frozen for a measurement's life, a gate *is* a measurement, so a gate
  concurrent with a benchmark was never legitimate. The runs have been serialized by hand all
  campaign for that reason and nothing enforced it. **No allowlist and no
  concurrent-benchmark exemption** — the gate fails, printing each offending worktree's path
  and revision, since the remedy depends on which it is: wait for the owner, or kill it. One
  measurement at a time stops being discipline and becomes an assertion. Reported separately
  from the dirty-tree failure rather than folded into it, because a dirty tree breaks
  `git archive HEAD` and so breaks the delegated chain by construction while a stray worktree
  does not touch HEAD and breaks nothing mechanically; sharing one flag would attribute a
  cause this does not have. The occasion was a real one: a worktree at `e5bce33` had sat in
  `/private/tmp` since the #47 A/B. Two plausible causes were checked before anything was
  filed and both were false — `l1-bench.sh` and `layout-ensemble.sh` put their worktrees under
  `mktemp -d` and already remove them in an EXIT trap (#55), and bash *does* run EXIT traps on
  death by SIGTERM, SIGHUP and SIGINT, so only SIGKILL escapes. It was a hand-typed
  `git worktree add` with no owner. Filing the assumed defect would have reported one that
  does not exist.
- **`scripts/layout-ensemble.sh`** decides whether an A/B delta is caused by a code change
  or by where the change happened to put the code — see the commit and #61. It grades
  placements as well as sampling them: because a code change displaces everything *after*
  it in link order, only routines at or before the changed function are comparable between
  the arms, so any measured routine whose entry address differs between the two binaries is
  labelled `placement-confounded` and **excluded from the verdict set by construction**
  rather than merely warned about — a warned row is still a citable row. A `geomean` over a
  demoted row inherits the demotion. A symbol the grader cannot locate demotes rather than
  clears, and the changed function itself moving between arms is a hard stop. On the
  e829a61 ensemble this catches `avx512SumSq` (`Snrm2`), the first routine after the
  subject, which shifts 0x20 — a multiple of 32 but not 64, i.e. T22's mechanism reproduced
  by the very commit under test.
- **`scripts/detach.sh`** runs a long gate or benchmark detached under `tmux`, so no run's
  completion depends on the lifetime of the shell that started it. Two `gate-p5` runs had
  been killed 25–28 minutes into the carried p5→p4→p3→p2 chain, and the response at the
  time was to have a human invoke the gate instead — which treated a harness defect as a
  scheduling problem. `tmux new-session -d` daemonises off the caller's process group and
  session, so reaping the caller cannot reach the work; this covers the remote half as
  well, since `scripts/remote.sh` runs each benchmark synchronously over `ssh` and a dead
  driver `SIGHUP`s a measurement in flight on the far side. `stat` reports `vanished`
  rather than an exit code when no status file was written, because a run that was killed
  is `unmeasured`, not failed (DESIGN.md §5.6). `wait` blocks on a `tmux wait-for` channel
  rather than polling, so noticing a finished run costs neither time nor latency. The
  channel is signalled on death by signal as well as on normal exit — otherwise a `wait` on
  a killed run would block forever — and from `kill` itself, which does not depend on the
  dying shell getting to run its trap. Signalling deliberately does *not* write the status
  file on signal death, so a killed run still reports `vanished` rather than acquiring an
  invented exit code. The session sets `exit-empty off`: a recorded `wait-for` signal lives
  in the tmux server, so letting the server exit with the last session would strand any
  later `wait` on a channel nobody will ever signal.

- `TestNoWritePastEnd` — a sentinel past the end of every writing routine's slice, at each
  length where a vector loop can end on an exact fit (0, 1, 7, 8, 9, 15, 16, 17, 24, 31, 32,
  33, 47, 48, 63, 64, 65, 80, 96, 127, 128, 129, 1024) on every available backend. The
  `>`-form's exact-fit epilogue is a *full-width store*, which is the one operation that can
  scribble past `len` while remaining valid Go — `x[0:16]` on a 15-element slice with spare
  capacity is legal and silently writes a 16th element. An oracle comparison over `x[:n]`
  cannot see that, so the check is a sentinel rather than a value. Mutation-tested against
  the standing order in DESIGN.md §5.7: an epilogue firing one element early (`>= lanes-1`)
  is caught at n=15, 31, 47, 63 and 127 on avx512, so the passing result is evidence rather
  than a check that could not have failed. The reductions get the paired check — same input
  with padding present versus trimmed must give bit-identical results, since `n` bounds the
  arithmetic.
- **DESIGN.md §5.7 — "a check that could not have come out otherwise is not evidence."** Two
  confirmations in this campaign were structural rather than evidential, and the pair is now
  a named trap with a standing order: #48's *tautology trap* (a "second, independent" check
  that was `4.00 ÷ (per-call rise)`, arithmetic on the quantity it claimed to corroborate)
  and T18's *unvaried control* (two arms differing in two ways at once, effect attributed to
  the salient one, three irrelevant variables held while the decisive one was never varied
  alone). The order is two questions: name the result that would have falsified each
  confirmation, and name the single variable that differs between each isolation's arms. Plus
  a corollary for compiler forensics — blame flows upstream through the pass pipeline, so a
  pass whose fingerprints are on the assembly may be correctly processing the output of one
  that did nothing.
- **CLAUDE.md standing order: search the upstream tracker before any upstream filing.** The
  `--check`-before-a-gate rule applied to the tracker, promoted from instinct after it paid
  once: T18 looked like a new register-allocation finding and was already
  [golang/go#79984](https://github.com/golang/go/issues/79984) with keel's exact shape
  reported on it and a fix CL in flight, so filing would have produced a duplicate carrying
  a wrong causal story. When the bug already exists the deliverable is a `standing-task`
  issue keyed to it (#54), and an upstream comment needs a fact the issue lacks — a measured
  delta on a real kernel qualifies, a second repro does not.
- `BenchmarkFeed` now documents a **measured residency caveat on its own denominator**
  (#48). `kernel-calls` reuses one hot panel pair to remove memory effects, and at one KC
  per host that backfires: a pair that *just* overflows L1d churns on capacity misses
  where a plainly-too-large one streams with the prefetcher working, so the denominator
  gets slower than the nest it is meant to bound. The two resolved negatives in a 24-row
  `nest-no-pack − kernel-calls` matrix each sit at the first KC where `(MR+NR)·kk·4`
  exceeds *that host's own* L1d — Zen 4 at 34816 B against 32 KB (−16.90 ns/call, 14.3×
  its noise floor) and Zen 5 at 52224 B against 48 KB (−3.30, 2.3×) — while Skylake-X,
  with the same 32 KB L1d as the Zen 4 part, never dips. Crossing L1d is necessary and
  not sufficient. Documented rather than fixed, which is what #48's own criterion asked
  for, and unfixable by shrinking the pair since `kk` is fixed by the call multiset every
  arm shares.
- `docs/toolchain-notes.md` T18, now **reduced to a minimal repro, and the reduction
  identifies a pre-existing upstream bug rather than a new one** (#8). The effect is that
  LICM does not lift SIMD ops: a `BroadcastInt32x16` constant with loop-invariant operands
  stays in the loop body and is rebuilt every iteration. That is
  [golang/go#79984](https://github.com/golang/go/issues/79984), open since before this
  note, with keel's exact shape (constant built inside a helper, expected inline-then-hoist,
  worked around by passing it in, ~4× throughput) already reported there as an addendum,
  and a fix in flight in [CL 803220](https://go-review.googlesource.com/c/go/+/803220) —
  work-in-progress as of 2026-08-12, so not in go1.26.5 and the workaround still applies.
  Nothing was filed upstream; the repro adds no fact the issue does not already carry.
  **Two claims from the previous T18 entries are withdrawn in the note.** (a) The isolated
  trigger was written as "the invariant reaches the loop as a CSE'd value from inside the
  callee rather than as a live local". A third arm with **no helper function at all** — the
  constant written directly in the loop body — emits the identical defect, 124 bytes
  against 123, so the callee, the inlining and the CSE are incidental. (b) The re-audit's
  causal reading, that the allocator "clobbers a live loop-invariant value with eleven
  registers free" and that this is *why* nothing hoists, has the direction backwards:
  because the build is inside the body, the mask is dead at that fourth `AndNot`, so
  reusing its register is free and correct. The clobber is a consequence of the missing
  hoist, not its cause, and there is no operand-assignment defect in the dump. The
  hand-hoisted arm was never successful LICM — its definition was already outside the loop
  and never needed lifting. What survives unchanged: the measured rematerialization, the
  three-loops-and-six-builds count in `avx512Asum`, and the fix #8 needs (a mask parameter
  on `Abs512`, hence an `internal/vec` API change with a scalar twin and differential).
- `docs/toolchain-notes.md` T18, re-audited in place — the question T18 left open (#8). In
  the real `avx512Asum` the loop-invariant sign mask is rebuilt at the back-edge target,
  and the fourth inlined `VPANDND` writes its result over the register holding the mask,
  with the highest vector register the function touches being `Z7`, so `Z8`–`Z15` sit idle
  well inside T10's 15-register ceiling. **The causal claim this entry originally drew from
  that — that the clobber is *why* nothing hoists, that it is the same character as T8, and
  that the cost is in operand assignment — is withdrawn by the reduction above**; the mask
  is dead at that instruction precisely *because* the build was left in the body, so the
  reuse is free and the idle registers are beside the point. Three loops rematerialize it, not one (the
  16-lane cleanup loop has the worst ratio, 3 mask instructions against 2 useful vector
  ops); `avx2Asum` does the same with `Y9`–`Y15` idle; and the unrolled body is 29
  instructions of which 8 do the arithmetic. The AVX-512 form folds its load into the
  operation and the AVX2 form does not, which is recorded as an observation needing its
  own repro rather than asserted as a note. No workaround landed: the hoist changes
  `internal/vec`'s API and edits the same loop bodies as #47, so the two get measured
  together.
- `docs/hosts.md`: each host's private cache sizes, read from sysfs rather than looked
  up — vesta 32K/1024K/98304K, janus 32K/1024K/22528K, antares **48K**/1024K/32768K.
  Zen 5's L1d is 48 KB where Zen 4's and Skylake-X's are 32 KB, which is exactly the
  kind of constant a from-memory value gets wrong, and the #48 feed decomposition needs
  it: the reused panel pair is `(MR+NR)·kk·4` bytes, so which KC keeps it L1-resident
  differs *per host* (kc=128 on vesta and janus, kc=128 and 256 on antares).
- `scripts/retention.sh feed` prints each step as a whole-GEMM total as well as a
  per-call cost: the point's own per-call ns times the exact call count out of the
  `keel-feed-panels` marker, in ms of one GEMM, with each term's share of `full`. A
  per-call cost says how expensive a term is but not whether it matters, since the
  call count itself falls 4× across the KC grid — and the totals are where the
  analytic predictions become checkable. C traffic must fall exactly like the call
  count and on janus at 4×32 it does (12.29 → 3.01 ms, ×4.08 against the ×4.00 the
  run's own call counts predict); pack must be flat and is, to within 3.5 ms of 20;
  panel feed has a KC-independent total, so its *rise* is the #48 finding restated in
  milliseconds, worst at kc=384 on janus at 2×32 (24.46 ms, 11.0% of the GEMM) —
  which is independent corroboration of the point defect the KC sweep found there,
  from a second instrument. The predicted C ratio is read off the run's own call
  counts rather than hardcoded, since it is `⌈k/KC₁⌉/⌈k/KCₙ⌉` and this script never
  sees k; a ratio whose endpoint is negative or below its noise floor is refused with
  the reason rather than printed; and a log written before the marker existed gets no
  totals table at all, because a reconstructed count could disagree with the one the
  ns/call column was divided by.
- `BenchmarkBlocking`'s grid is replaceable per axis from the environment
  (`KEEL_BLOCKING_KC`, `_MC`, `_NC`, comma-separated; `scripts/retention.sh sweep`
  forwards them, since sshd strips arbitrary env). A fine scan around a suspected
  point defect — janus at 2×32 has one at kc=384 — is then the same benchmark at a
  different grid rather than a second benchmark, and a 5-point scan is small enough
  to afford the full `-count=10` methodology, which sharp associativity effects need:
  a coarse grid can alias them into fiction in either direction. A malformed value is
  fatal rather than defaulted, because a sweep that silently measures a grid other
  than the one it was asked for is #49 again; and since every point names its own
  KC/MC/NC, the grid that actually ran stays readable out of the log. The sweep header
  no longer tells a reader of a `-count=10` log to re-measure it at 10, and now says
  what is true at any count: a top row here becomes a default only through #24's
  `kern.Class`, never by winning a sweep.
- `BenchmarkFeed` and `scripts/retention.sh feed`: the per-call decomposition of
  the blocked nest resolved against KC, which is the question the KC/MC/NC sweep
  left open (#48). The sweep found that janus's per-call penalty *rises* with KC
  (39.7 → 69.3 → 72.3 ns/call), so some term scales with a call's own duration
  rather than with the number of calls — and the two candidates, real C traffic
  and panel feed, both have KC-independent *totals*, so their size cannot separate
  them and only their shape against KC can. Six arms per (shape, KC), each one
  variable from the last, all making the identical call multiset at identical
  depths: `kernel-calls`, `loops` (the macro loops' own address arithmetic),
  `cold-c` (real C at the nest's own tile addresses), `cold-panels` (real panels
  at macro's own offsets), `nest-no-pack`, `full`. Per-call cost is then a
  measurement divided by an exact count and each column is a difference of two
  measured quantities — no slope, no fit through grid points. The six steps sum to
  `full − kernel-calls` by construction, so a hidden term shows up as a residual
  rather than being absorbed. The pack column's intercept is the term that
  separates the sweep's slope from the decomposition's per-call cost, which was
  previously argued rather than measured.
- Project skeleton: module layout, phase gate scripts, GitHub project bootstrap,
  CI (stock + `GOEXPERIMENT=simd` builds), design document (`DESIGN.md`).
- `internal/vec`, the simd shim, with all three backends over the ~14 ops SGEMM
  and the Level-1 routines need. Scalar backend is the executable spec and
  builds on a stock toolchain on every GOARCH; AVX-512 and AVX2 backends are
  written against the `go doc`-read archsimd API of go1.26.5.
- Differential test harness binding every vector backend to the scalar spec at
  **bit-exact** equality (the spec was written to match the vector units:
  single-rounding `MulAdd`, pairwise-halving `HSum`, sign-bit-masking `Abs`), over
  an adversarial pool covering NaN, ±Inf, denormals, −0, partial and empty
  slices, and every offset across the width boundary.
- Characterization tests pinning the scalar spec's own semantics, which run on a
  stock toolchain and on every GOARCH — including witnesses that fail if
  `MulAdd` stops being fused or `HSum` stops folding pairwise.
- `scripts/gate-p0.sh`: real P0 checks (both builds, vet, shim tests with
  enforced backend coverage, and FMA fusion verified by disassembly). Now
  scores one *execution target* per machine and requires that all three
  backends ran together on real silicon, naming which machine did it.
- `scripts/remote.sh`: ships a cross-compiled static `go test -c` binary to an
  amd64 host over ssh and runs it, so a `darwin/arm64` dev host can execute
  AVX-512 code — and, from P1 on, benchmark it — with no Go toolchain
  installed on the target. Hosts come from `.keel-hosts` (gitignored) or
  `$KEEL_REMOTE_HOSTS`; see `docs/hosts.md`.
- `docs/hosts.md`: the three amd64 execution targets, what each one is good for, and
  the Zen 4 double-pumped-AVX-512 caveat that P2's percent-of-peak denominator
  has to account for.
- `docs/toolchain-notes.md` T8: three ways a register-only FMA-saturation
  microbenchmark silently stops measuring a hardware ceiling — CSE merging
  identical accumulator chains into one (an ~8× understatement, hence an ~8×
  inflation of everything divided by it), `VFMADD213PS`'s clobbered multiplicand
  costing 26 register copies per iteration in the natural accumulator form, and
  constant multiplicands deleting the multiply while adding a load. Each with a
  verified repro. None is a compiler bug; all three read correctly in Go source
  and are wrong only in the disassembly, always in the flattering direction.
- `docs/toolchain-notes.md`: seven earlier field notes on `GOEXPERIMENT=simd` in go1.26.5,
  each with a minimal repro — archsimd being amd64-only, the `VFMADD213PS`
  lowering, the absent float32 `Abs`/bitwise ops, the absent `Float32x16`
  horizontal reduce, the absent portable `simd` package, the `Max`/`Min` NaN
  operand order, and `GOAMD64` not gating archsimd intrinsics.

- **Level 1 BLAS: `Sdot`, `Saxpy`, `Sscal`, `Snrm2`, `Sasum`, `Isamax`.** Unit
  stride dispatches to `internal/l1`'s per-backend kernels (one indirect call
  per call, none inside a loop); non-unit and negative strides take scalar
  loops in the `keel` package. `KEEL_FORCE=scalar|avx2|avx512` pins the
  backend and panics rather than silently downgrading if it is unavailable.
- `internal/l1`: unit-stride kernels for all three backends. Four independent
  accumulator chains per reduction so FMA latency does not serialize them — in
  the scalar path too, so the ≥4× gate is measured against what a competent Go
  programmer would write rather than a straw man. Remainders use masked partial
  loads/stores instead of a scalar tail.
- `internal/oracle`: float64 references for all six routines. Each reduction
  returns its value *and* the error scale (Σ|xᵢyᵢ| for a dot product), because a
  cancelling dot product has a tiny result and a large legitimate error bound,
  and a test that substitutes |result| for the scale is testing nothing.
- `Snrm2` over/underflow rescue: the vector kernel runs unguarded and its
  *result* is inspected — a sum of squares is monotonic, so one post-loop check
  catches everything a per-element check would, without a branch per element.
  Overflow or total underflow reruns in float64, where float32 inputs cannot
  overflow for any n below ~1e230, so the rescue is exact by construction.
- Level-1 test suite: every routine against the oracle on every backend the
  machine can execute, cross-backend differential at 2× tolerance (triangle
  inequality, not a fudge factor), 32 shapes chosen around the 8/16/32/64
  lane and unroll boundaries, six data patterns including cancelling and
  subnormal, strided and negative-strided coverage with poisoned gaps, NaN/±Inf
  propagation swept across body/remainder/masked-tail positions, and the
  argument-validation panics.
- `bench/`: benchmark harness reporting GFLOP/s with CPU model, core count,
  governor, clock snapshot, active backend, and the host's measured FMA peak with
  the formula printed beside it as a cross-check.
- `scripts/gate-p1.sh`: real P1 checks — both builds, vet, L1 tests with
  enforced per-backend coverage on every host, the whole suite re-run under
  `KEEL_FORCE=scalar` on machines that *have* AVX-512 (a scalar pass on arm64
  would not prove the override works), and the ≥4× Sdot ratio measured
  within-machine under the §5 rule 5 methodology — benchstat median, cleared net of
  its confidence interval, on every host, with at least one clearing it under the
  `performance` governor. It also measures and prints each host's FMA peak and
  512/256 width ratio: not a P1 criterion, but P2 divides by that number, so both
  phases' figures share a measurement regime from the start.

- **A measured percent-of-peak denominator** (`internal/vec/peak.go`,
  `bench/BenchmarkPeak`), replacing the DESIGN.md formula. Register-only FMA
  saturation: no memory in the loop, twelve independent accumulator chains at
  512 bits (ten at 256 and ten scalar), each starting at a distinct value so CSE
  cannot merge them, and the accumulator in the destination operand so the
  lowering needs no register copies. Verified three ways — disassembly of the
  steady-state loop, an exact arithmetic witness
  (`TestPeakChainsAreIndependent`) that fails on any host if a chain does not
  survive compilation, and the same witness re-checked inside the benchmark that
  produces the number.
- `scripts/bench.sh`: the one gate benchmark methodology, shared by every gate
  so none can deviate from it — `-count=10 -benchtime=1s`, aggregated by the
  `benchstat` pinned as a module tool, thresholds cleared net of the reported
  confidence interval, and no verdict at all when benchstat cannot bound the
  distribution.
- `internal/spill`, the P2 spill audit: parses `go build -gcflags=-S`, identifies
  the steady-state K-loop (innermost loop carrying the arithmetic, excluding the
  `runtime.morestack` re-entry jump that spans every function), and classifies its
  body. It separates the three things a grep conflates: a *spill* (a vector
  register moved to or from `(SP)`), a *register copy* (`VFMADD213PS`'s clobbered
  multiplicand, an issue slot rather than memory traffic), and a *broadcast*
  (arithmetic setup). Also counts the `XCHGL AX, AX` statement anchors, which a
  listing parser cannot drop because they are not spelled `NOP`.
- **Bounds-check elimination is checked on the loop body, not on the package.** The
  audit reports a surviving bounds check as what it actually is in the object code
  — a conditional branch out of the loop body to a block that calls
  `runtime.panic*` — because the panic block is laid out after the hot path, so
  grepping the body for a `CALL` finds nothing. `-d=ssa/check_bce` is printed as
  provenance and is not the criterion: it reports dozens of legitimate checks
  outside the K-loop (`a[:kc*MR]` in the prologue, `c[i*ldc:i*ldc+NR]` in the
  write-out), each of which costs nothing amortized over K, so a gate that failed
  on them would be unsatisfiable for reasons unrelated to P2. Verified both ways
  against a positive and a negative control.
- **SGEMM microkernels** (`internal/vec/gemm_amd64.go`, registry in
  `internal/kern`): two zero-spill AVX-512 tiles, 2×32 unrolled ×4 and 4×32
  unrolled ×1, both differential-tested against a scalar twin of their own shape
  under `oracle.Tolerance`, over kc values covering 0, the sub-unroll cases, exact
  multiples, remainders, and 128, with guard rows and columns checked untouched.
  Both audit clean: 0 spills, 0 calls, 0 surviving bounds checks in the
  steady-state loop.
- `KERNEL.md`: the tile shape record P2 exists to produce — the register budget,
  the measured spill frontier over a 115-shape sweep, the full 74-instruction
  opcode histogram of the 2×32 body, the two shaping attempts that were measured
  and dropped, and the per-host winner.
- `bench/BenchmarkKernel`: every shape on packed L1-resident panels at kc ∈ {8,
  32, 128, 512}, reporting GFLOP/s (shapes do different work per call, so ns/op
  cannot compare them) and flops/call. Documented as an *upper bound* on what P3
  can deliver, because excluding packing and blocking is what makes it a
  measurement of the K-loop.
- `docs/spill-report.md`: the P2 go/no-go report, required by DESIGN.md §4/P2 when
  the gate is red. **The spill audit passed and the flat 55%-of-peak floor did
  not** — janus.local (Skylake-X) reached 46.1% against the floor, while vesta
  (Zen 4) reached 96.6% and antares (Zen 5) 64.1%. The binding constraint is
  instructions issued per FMA, not spills, established two ways from keel's own
  measurements: a clock-free test (the two shipped shapes' throughputs stand in the
  inverse ratio of their instruction counts on janus, 1.308 measured against 1.351
  predicted, and do not on the other two hosts) and a per-cycle derivation (both
  janus shapes return the same ~4.2 instructions per cycle despite differing 35% in
  instruction count). §7 listed four open decisions and stated which were not taken;
  the ruling on issue #19 took the first, and §9 now records the amended gate model
  and why it tightens rather than expires.
- **zmm FMA/cycle is now measured per host, not cited**: `BenchmarkPeak/avx512`
  pinned with `taskset` while sampling that core's `cpufreq/scaling_cur_freq` gives
  0.996 on vesta, 1.944 on janus, 1.996 on antares — Zen 4 double-pumping AVX-512
  over 256-bit datapaths, and both the Skylake-X and Zen 5 parts retiring two
  full-width FMAs per cycle. This is the column that explains why one host is
  issue-bound: a 2-FMA/cycle machine feeds twice the arithmetic from the same front
  end, so it has half the instruction budget per FMA
  (docs/spill-report.md §3.3).
- `KERNEL.md` §7, the per-host winner: **the winner flips, so both shapes ship.**
  The load-lean 4×32 wins on vesta (96.6% vs 92.4%) and antares (64.2% vs 53.1%);
  the instruction-lean 2×32 wins on janus (46.0% vs 35.2%). Shipping one shape on
  theory would have been wrong on at least one machine in this fleet, and the shape
  theory most favoured wins on the fewest hosts.
- `docs/toolchain-notes.md` T12 (issue #20): **the K-loop's ideal instruction
  exists in Go's assembler and cannot be reached from Go.** `go tool asm` plus
  `llvm-mc` confirm `VFMADD231PS.BCST 12(SI), Z1, Z0` encodes as the seven bytes
  `62 f2 75 58 b8 46 03` — EVEX.512, embedded broadcast, accumulate in place,
  memory as a *multiplicand*. Three independent reasons the intrinsic layer cannot
  emit it: only 213-shaped FMA SSA ops exist for vectors (scalar `VFMADD231SS/SD`
  do exist, for `math.FMA`); the one load-merge rule that exists
  (`simdAMD64.rules:2774`) can only fold memory into the 213 form's *addend*, which
  in a GEMM is the accumulator and the single operand that must stay in a register;
  and nothing under `ssa/_gen/` emits `.BCST`, though `obj/x86/evex.go` supports it.
  This is the largest term in the 2×32 budget — 74 → ~46 instructions, 4.625 →
  2.875 insns/FMA — and it supersedes `docs/spill-report.md` §5's original
  accounting, which credited only T9 and T10 and was therefore short by ~1.75×.
- **Three field notes are filed upstream against `golang/go`**, each with a
  self-contained repro built from scratch for the filing and re-verified against the
  go1.26.5 GOROOT, and each carrying janus's roofline table as its impact statement:
  [golang/go#80828](https://github.com/golang/go/issues/80828) (512-bit values allocated from
  15 of 32 zmm registers — a fresh sweep puts the zero-spill frontier at 13
  independent accumulators and shows no register above Z14 is ever named),
  [golang/go#80829](https://github.com/golang/go/issues/80829) (no 231-shaped vector FMA, the
  load-merge rule folds the addend, nothing emits `.BCST` — with the byte-identical
  `go tool asm` / `llvm-mc` encodings and a 9-instructions-for-2-FMAs GEMM row),
  [golang/go#80830](https://github.com/golang/go/issues/80830) (`BroadcastFloat32x16` is
  emulated as `SetElem`+`Broadcast1To16`; a one-line wrapper costs one anchor NOP per
  call site — 7 insns/0 NOPs direct against 11/4 wrapped, for identical arithmetic).
  Recorded in `docs/toolchain-notes.md` beside T9, T10 and T12 and in
  `docs/spill-report.md` §9.
- `scripts/roofline.sh`: the throughput verdict as a single pure function with no
  I/O, so the rule that decides a go/no-go can be read in one place and tested
  without a benchmark. Classifies a host FMA-bound or issue-bound from measured
  `(fraction-of-peak, audited insns/FMA)` pairs and returns
  `CLASS CSPREAD MSPREAD ROOF ATTAIN RESULT WHY`. The roofline is clock-free: with
  `f_i` the measured fraction of peak and `I_i` the audited insns/FMA,
  `roofline(I) = maxᵢ(f_i·I_i)/I` — the retirement rate cancels, so no clock,
  `taskset` or perf counter enters the gate.
- `scripts/roofline-test.sh`: 15 adversarial fixtures for that function, run by
  `gate-p2.sh` *before* any benchmarking, so a broken decision rule fails the gate
  on any host in a second. Fixtures feed measured `(f, I)` pairs rather than
  pre-reduced spreads, so a fixture cannot describe a host that could not exist —
  which is how one of the first hand-written negative controls was caught being
  fake. They include a kernel padded with 40 dead instructions trying to buy itself
  a roofline, a slow kernel on a wide host, a sandbagged alternate shape, a
  single-mix host, both sides of the 90.0% and +5.0% boundaries, and the post-T12-fix
  janus that needs 70.4% and the one that only makes 76.7% of its roofline.
- `docs/toolchain-notes.md` T11: `GOSSAFUNC` is not part of the build cache key,
  so a repeated build is a cache hit that writes no `ssa.html` — while replaying
  the cached compiler stderr, including `dumped SSA for <fn> to ./ssa.html`. Found
  by the gate requiring the archived dump and checking for the file instead of
  trusting the exit status: the requirement passed on one run and failed on the
  next, the audit's own `-gcflags=-S` compile having warmed the cache in between.
  `spill-audit` now gives each dump a private `GOCACHE`, which is in the lookup —
  0.63 s and 18 MB per function, discarded after.

- **`Sgemm`: the full Level-3 routine, row-major, all four transpose
  combinations, general alpha and beta.** `internal/pack` produces packed panels
  and `internal/block` drives the Goto/BLIS loop nest NC→KC→MC→NR→MR (`KC=384`,
  `MC=144`, `NC=4096`, exported vars, clamped to the problem and rounded down to
  whole tiles). Panels are k-major — `a[p*MR+i]`, `b[p*NR+j]` — so the P2-audited
  kernels read contiguously along the depth loop and nothing in the K-loop
  computes an address from a stride.
  - **alpha is folded into the packed A** (BLIS convention): O(mc·kc) multiplies
    instead of O(mc·nc·kc), it stays out of the audited K-loop entirely, and the
    extra rounding is covered by `oracle.Tolerance` — the float64 oracle folds it
    the same way, so the sweep tests the arithmetic keel actually performs.
  - **beta is applied outside the kernel**, once per C block before the first
    KC panel, in three variants selected by value (`beta == 1` returns,
    `beta == 0` clears, otherwise scales). No branch on beta inside any loop, and
    no separate kernel per variant.
  - **Edges are zero-padded panels plus a temp tile**, decided from the read API
    rather than from habit (issue #4, numbers deferred to issue #22). Masking is
    genuinely available in go1.26.5 — mask types, `Masked`/`Merge`,
    `LoadMasked`/`StoreMasked`, `LoadFloat32x16SlicePart`/`StoreSlicePart` — but a
    masked C update means a second kernel family, doubling what has to stay
    zero-spill under P2's audit that the P3 gate re-runs. So fringe tiles run the
    same kernel over zeros into an MR×NR scratch buffer and the valid
    sub-rectangle is copied back: one K-loop, byte-identical for interior and edge
    tiles. The padding is written, never assumed, because the buffers are reused
    across blocks. It also discards the `0·Inf = NaN` a padding lane can produce,
    which `TestSgemmNonFinite` pins.
  - **Packing is `copy` in the contiguous direction and a scalar loop in the
    transposing one**, also from the read API: `grep -l -i 'gather\|scatter'`
    over `$GOROOT/src/simd/archsimd/*.go` returns nothing at any width, so a
    strided store is not expressible. A 16×16 in-register transpose is
    constructible from `Permute`/`ConcatPermute`, which do exist; that is a
    permutation network to write, test and audit for a routine whose cost is
    O(mc·kc), so it waits for a measurement (issue #21).
  - Argument errors panic, as everywhere else in keel; `ld >= max(1, cols)` is
    checked even for an empty matrix, matching reference SGEMM's `LDA >= MAX(1,…)`.
    `k == 0` is the empty product (`C = beta*C`), `alpha == 0` never reads A or B
    so a NaN there cannot reach C, and `beta == 0` never reads C so an
    uninitialized destination is legal.
- **`Sgemm` differential suite against the float64 oracle** (`gemm_test.go`,
  `internal/oracle/gemm.go`): sizes 1–17, 63, 64, 65, 500, 1000, 2048 × {NN, NT,
  TN, TT} × alpha {0, 1, −0.75} × beta {0, 1, 0.5}. The full 36-combination
  lattice runs at every size where a fringe tile can occur (≤65) and one rotating
  combination at 500/1000/2048, because 36 combinations at 2048³ is 620 GFLOP per
  runner and what the large sizes exercise is the loop nest, not the flag
  handling. Every element is checked at or below 65; above it, four corners plus
  256 seeded samples. Each kernel shape in `kern.Kernels()` is run as its own
  runner alongside the public path, so a tile cannot ship untested. Plus leading-
  dimension padding with poison in the gaps (checked untouched afterwards),
  zero dimensions, 14 argument-panic cases, and the non-finite cases above.
- `internal/pack` differential test against a straightforward reference packer,
  420 combinations over both source geometries — it lives in the root package so
  its coverage marker comes from the same binary the gate ships to each host.
- `bench.BenchmarkSgemm` at n = 256/512/1024/2048, and an OpenBLAS reference
  (`BenchmarkOpenBLAS`, build tag `openblas`) making the identical call. The tag
  keeps it out of the module graph: nothing keel ships links a BLAS. It prints the
  library's own report of itself — version and build flags, `DYNAMIC_ARCH`-selected
  kernel family, thread count, and the CPU count that thread count restricts — so
  the denominator is identified rather than merely named. Compile- and run-verified
  against Homebrew OpenBLAS 0.3.34 on the dev host; the criterion itself needs each
  amd64 gate host to have both a Go toolchain and the library, which is still an
  open provisioning decision (`docs/hosts.md`, issue #23).
- `scripts/provision-openblas.sh`: installs the same-host OpenBLAS reference and a
  `GOEXPERIMENT=simd`-capable toolchain on each gate host, then *verifies* by
  building the `openblas`-tagged harness there and printing the marker the gate
  will check — version, `OPENBLAS_NUM_THREADS=1` read back from the library, and
  the selected kernel family against the gate's own allowlist. Separate from the
  gate on purpose: it needs interactive `sudo`, which the gate's `BatchMode=yes`
  connection cannot answer and should not try to. It handles no credentials, and it
  reports a non-`performance` governor rather than changing a machine's power
  policy — but reports it as a **failure**, since the gate now refuses such a host
  outright (issue #31): a machine whose library and toolchain are fine and whose
  governor is wrong is provisioned-and-unmeasurable, and finding that out here is
  cheaper than finding it out in a gate run. The Go tarball's digest is enforced
  against `$KEEL_GO_SHA256` when set and otherwise against `go.dev/dl?mode=json`,
  and it says which of the two it did, since only the first is provenance. That
  fallback lookup had never once run (issue #29): go.dev serves pretty-printed
  JSON, so splitting the objects on `{` alone left each object's fields on separate
  lines and the line-oriented `grep` matched only the `"filename"` line, never the
  neighbouring `"sha256"`. The expected digest was therefore empty for every
  version ever requested. It failed closed, so nothing unverified was ever
  installed — but `install_go` could not complete without `$KEEL_GO_SHA256`, which
  is why the check's silence went unnoticed: the visible symptom was a refusal, not
  a pass. It distinguishes "no usable toolchain"
  from "usable toolchain, not on the PATH the gate uses" and links the latter rather
  than deleting it (issue #27): probing only the ssh `PATH` would have had it
  `sudo rm -rf` a working go1.26.5 on antares to reinstall the same version, and the
  one irreversible action it can take on a host is now named in the prompt that
  authorizes it. Its `[y/N]` prompts read from `/dev/tty` and its host loop from
  fd 3 (issue #28): both used stdin, so each prompt consumed the *next host* as its
  answer — three hosts named meant one was never visited, having been spent as a
  keystroke, and the interactive path had therefore never worked at all. This is the
  hazard `scripts/remote.sh` already documents and defeats with `ssh -n`, in the one
  script that cannot use `-n` because `sudo` needs the `-t` tty. No tty and no
  `--yes` now fails with its own message instead of printing `skipped`, which had
  reported a decision nobody was asked to make.
- `docs/toolchain-notes.md` T13: `import "C"` in a `_test.go` file is rejected
  outright (`use of cgo in test … not supported`,
  `cmd/go/internal/modindex/read.go:589`), and it fails as `[setup failed]` — which
  in a script reads like a missing library rather than a rejected file layout. Not
  a simd note and not new, recorded because it shaped `bench/`'s file split: the
  cgo binding is a package file, the benchmark a test file, and since package files
  cannot see `_test.go` identifiers the provenance variable is set from the tagged
  test file's `init`.
- `docs/toolchain-notes.md` T14 (issue #25): `archsimd` reports CPU *features* and
  nothing else — no vendor, family, model or brand string — and the data is not
  merely unexported one layer down, since `internal/cpu` calls `cpuid(1, 0)` and
  keeps only `ecx`, discarding the signature word. Kernel *shape* selection is a
  per-µarch decision (KERNEL.md §7: the winner flips between hosts with identical
  feature sets), which is why every production BLAS dispatches on vendor plus
  family/model, so on this toolchain keel has to fingerprint a feature bundle
  instead. Recorded before the workaround, with the two upstream shapes that would
  retire it.
- `docs/toolchain-notes.md` T15 (issue #32): `go test -bench` splits a pattern on
  top-level `|` **before** `/`, so `A|B/c` is the alternation `{A}` or `{B,c}` and
  not the two-level filter `{A,B}` then `{c}` that it reads as. An alternative with
  fewer elements than the name is depth-unconstrained, and one with more matches no
  benchmark while still matching the *parent* partially — `simpleMatch.matches`
  returns `ok, partial = true, true` when `len(name) < len(m)` — so the parent is
  entered, prints its `init` output, and yields no result row. Not a compiler bug
  and not simd-specific; recorded for the failure mode, which is a filter that reads
  correctly to every reviewer, runs without error, and silently measures something
  other than what it names. Parentheses suppress both splits, which is the fix.
  Repro on janus, plus the audit of every filter in the repo.

- **Level 2 BLAS: `Sgemv` (both transposes) and `Sger`.** Both are row-at-a-time
  loops over `internal/l1`'s unit-stride kernels — a dot product per row of A
  untransposed, an axpy per row transposed, and an axpy per row for the rank-1
  update — so there is one indirect call per row of A and none inside a loop.
  Strided vectors are gathered into a contiguous buffer once per call rather than
  handled by a strided inner loop: the gather is O(n) against the routine's
  O(m·n), and it keeps the kernel path the one P1 measured.
  - Two documented deviations from reference SGEMV, both toward the rule the
    float64 oracle can check element by element. An empty reduction still applies
    beta (`y := beta*y` is the value of the expression when the sum is empty — the
    same rule as `Sgemm`'s `k == 0`, where reference returns early); and alpha
    multiplies the dot product rather than each element of the row.
  - **No zero-multiplier guard in either routine.** Reference SGER has one
    (`IF (TEMP.NE.ZERO)`) and reference SGEMV does not; keel takes the unguarded
    rule for both, because `0·Inf = NaN` must propagate and a skipped row would
    disagree with the oracle on exactly that input.
- **`Ssyrk`, `Ssymm` and `Strsm` as derivations on the P3 loop nest**, not as
  second implementations of it. Each one inherits the packing, the blocking
  parameters, the three beta variants, the zero-padded-panel edge strategy and the
  P2-audited K-loop; `scripts/gate-p4.sh` criterion 5 checks that inheritance from
  markers rather than trusting it, requiring every derived routine to report the
  same microkernel and the same `kc`/`mc`/`nc` that `Sgemm` dispatched in the same
  process.
  - **`Ssyrk` is one GEMM call with A on both sides and the C update masked to a
    triangle** (`internal/block/tri.go`). The mask is over C's *global* indices, so
    the loop nest threads the block offsets down to the macro-kernel; blocks
    entirely outside the triangle are never packed and never reach the kernel, and
    a tile that straddles the diagonal runs the existing scratch-tile path — the
    one the fringe already uses — and copies back a row range instead of a row.
    `triMask{on: false}` folds every predicate to a constant, so `Sgemm` pays
    nothing for the mask's existence. No new kernel family, and none of P2's audit
    surface is widened.
  - **`Ssymm` reflects the stored triangle of A into a dense square and is then one
    unmasked GEMM.** The cost is O(d²) of scratch and one extra pass, against the
    O(d²·n) of the multiply; pack-time mirroring is strictly better, also covers a
    future `Ssymv`, and is issue #36. `alpha == 0` returns `beta*C` without
    allocating or reading A, which is a correctness requirement rather than a
    shortcut: A's unreferenced triangle may hold anything.
  - **`Strsm` is the BLIS blocked recipe** (Van Zee & van de Geijn, TOMS 2015 §4.3;
    Goto & van de Geijn, TOMS 2008 §4): a GEMM rank update against the
    already-solved blocks, then an unblocked solve against one `MB`×`MB` diagonal
    block, in whichever of the four directions `side` and `uplo != trans` select.
    All the flops except the diagonal blocks' `O(m·MB·n)` go through the audited
    kernel. The diagonal solves divide rather than multiply by a reciprocal (a
    reciprocal changes the last bit and disagrees with the oracle for no gain) and
    do not skip a zero multiplier (0·Inf must propagate). They are scalar, `MB` is
    an untuned `var`, and both are issue #37.
  - `unit` diag means the stored diagonal is **not referenced**, not that it
    contains ones — the guarantee a caller holding an LU factorization in one array
    relies on. `alpha == 0` zeroes B without reading A at all, so a singular or
    infinite diagonal is legal on that call.
- **P4 differential suite against the float64 oracle** (`gemv_test.go`,
  `tri_test.go`, `internal/oracle/gemv.go`, `internal/oracle/l3.go`), over the
  P4 size list 1–17, 31, 32, 33, 63, 64, 65, 500.
  - The full flag lattice per routine: `Sgemv` 162 combinations (trans × alpha ×
    beta × incx × incy, each stride being unit, wider-than-one and negative),
    `Sger` 27, `Ssyrk` 36, `Ssymm` 36, `Strsm` 48 (side × uplo × trans × diag ×
    alpha). Level 2 runs its lattice at *every* size including 500 and verifies
    every output element there, because its entry-wise oracle is O(n) per output —
    an exhaustive comparison costs the same order as the routine. Level 3 runs the
    full lattice up to 65 and one flag corner rotated by runner index above it, and
    the reduction is stated in the markers rather than implied.
  - **Coverage is counted, not declared.** Each routine records the flag sets it
    swept *and the set of tuples it actually reached*; the gate multiplies the
    former and requires the product to equal the count of the latter. P3 could
    print its constants because `Sgemm` has one flag pair; with five routines and
    sixteen corners the interesting failure is no longer "the sweep is too small",
    it is "the sweep declares a lattice and skips part of it" — which no test
    failure would ever show.
  - **The properties an oracle comparison cannot see**, each about memory the
    routine must not touch: `Ssyrk`'s untouched triangle of C (poisoned and
    required back *bit-identical*, not merely close); `Ssymm`'s and `Strsm`'s
    unreferenced triangle of A and `Strsm`'s unit diagonal (poisoned with NaN, so
    a read propagates into the answer instead of being absorbed by a tolerance);
    leading-dimension padding on all three; zero dimensions; argument panics; and
    the non-finite rules, including an infinity meeting a zero-padded panel on a
    tile that straddles the diagonal — where the copy-back is what keeps the
    resulting NaN out of both C's other triangle and its padding.
  - `Strsm`'s test matrices are diagonally dominant on purpose. A triangular
    solve's error bound grows multiplicatively down the substitution
    (`oracle.Trsm` derives the recursion), so a random triangular matrix at
    n = 500 has a legitimate bound wide enough to admit anything — the test would
    pass on a broken routine and be reporting on the test matrix instead of on
    keel.
  - `internal/block/tri_test.go` checks the mask's three range predicates
    (`whole`, `none`, `rowRange`) against the element-wise `keeps` definition over
    every rectangle in a small square. They decide per tile and per row rather than
    per element, which is what makes the mask free and also what makes an
    off-by-one in them invisible at most sizes.
- `bench.BenchmarkSsyrk` at n = 256/512/1024/2048, beside `BenchmarkSgemm` at the
  same sizes, for P4's `Ssyrk >= 85% of Sgemm` criterion. **Both now declare their
  numerator**: a `keel-bench-flops` marker naming the flop count, the formula and
  the dimensions used, from the same `work` value the harness divides by, and
  `BenchmarkOpenBLAS` takes `Sgemm`'s. `Ssyrk` fills one triangle, so its count is
  `k*n*(n+1)` and not `2*m*n*k` — counting the latter would report about twice its
  real rate and the 85% bar would be cleared by a routine running at 43%. The
  counts are *useful* flops: the half of each diagonal tile that is computed and
  discarded is the cost the bar exists to measure, so it is not counted as work.
  Rule 7's "never a number without its denominator" pointed at the numerator.
- **`scripts/gate-p4.sh` is GREEN: 65 PASS / 0 FAIL** at `dd740e5`. P4's headline
  criterion — single-thread `Ssyrk` at ≥85% of the *same host's own* `Sgemm`, both
  rates from one benchmark invocation at n = k = 2048 under the `performance`
  governor, `-count=10 -benchtime=1s`, medians net of benchstat's confidence
  interval — is met on all three gate hosts:

  | Host | Ssyrk | Sgemm | Ssyrk / Sgemm | of measured peak |
  |---|---|---|---|---|
  | vesta, Zen 4 (7950X3D) | 134.4 | 148.1 | **90.8%** | 80.9% / 89.2% of 166.1 |
  | janus, Skylake-X (i9-9960X) | 69.11 | 77.27 | **89.4%** | 31.9% / 35.6% of 216.9 |
  | antares, Zen 5 (Ryzen AI MAX+ 395) | 168.5 | 191.3 | **88.1%** | 51.4% / 58.3% of 328.1 |

  GFLOP/s. Percent of measured peak is reported and not judged, here as in P3.
  Criterion 7 was measured twice — an earlier run of the same gate, aborted later
  in criterion 8, produced 90.6 / 89.8 / 88.2 — so the three ratios agree within
  0.4 points across two independent measurements, and the ordering by host is the
  same both times. The remaining 9–12 points are the derivation's stated cost:
  tiles that straddle the diagonal are computed whole and half-discarded, and the
  discarded half is deliberately absent from the numerator.
- P4's gate carries P3's rather than restating its threshold: criterion 8 runs
  `scripts/gate-p3.sh` on the same commit (**47 PASS / 0 FAIL**, its fourth green
  full run) and refuses a dirty tree as unmeasured. The Ssyrk/Sgemm ratio is a
  ratio against a number P4's own commits can move, so the denominator's bar is
  enforced by the gate that owns it — a second copy of "≥60% of OpenBLAS" in
  `gate-p4.sh` would be a number that can drift out of agreement with itself.

- `scripts/gate-p5.sh`: P5's criteria, written before P5's code (CLAUDE.md), and red
  in **23** places on the first clean-tree run (`583ca74`: 23 FAIL / 29 PASS,
  `gate-p5: RED`) — almost all of them "P5 has not been built yet", plus four
  findings the gate produced by being run at all, each now an issue rather than a
  drive-by fix (#38, #39, #40, #42). The count was 22 on the first run, taken
  against a dirty tree, which is the *less* red number for a reason worth keeping:
  three of those 22 were "this check could not run", and running them on a clean
  tree turned one skip into four real failures — the three per-host `-race`
  verdicts and their aggregate. A gate that cannot run its own checks understates
  itself. The delegated P4 gate is **green** on this commit (64 PASS / 0 FAIL),
  so every absolute rate the ≥6× ratio stands on is still a measured one.

  It is **red in 20 places as of `51d206f`** (20 FAIL / 32 PASS, `gate-p5: RED`),
  and the three that left are the #40 ruling landing: each per-host
  `KEEL_FORCE=avx2` verdict became a PASS that *asserts* the Level-3 ceiling
  (`l1=avx2, kern=4x32/scalar`) rather than one that excused the missing rung.
  Nothing else in the set moved — an intended delta of exactly three, which is the
  evidence the narrowing was surgical. The absent `keel-p5-dispatch` marker stayed
  **one** failure rather than becoming two when the check gained a second field: no
  marker emitted is one defect with one cause, and a count that inflated with the
  number of things the missing marker *would* have been checked against would
  overstate how much is wrong.

  The 20 all have an owner and a stage: **1** lint failure carrying two findings
  (#39's unchecked `os.RemoveAll` and #38's `-0.0` literal); **8** absent
  determinism/no-state markers, two per routine across the four; **1** absent
  dispatch marker; **3** `-race` deaths from `archsimd`'s `checkptr` violation
  (#42, ruled into #22's campaign) **plus 1** aggregate saying no host produced a
  reading at all; **3** hosts reporting the scaling ratios' inputs unmeasured
  **plus 1** aggregate for the headline criterion; and **2** shipping artifacts
  absent — `doc.go`, and the README's `keel-numbers` block. Stage 2 owns nine of
  them, stage 3 six, stage 1's one-liners the lint pair, and #22's campaign the
  four race lines.
  The judgement calls are in the script's header at length; the ones that shape
  the phase:
  - The ≥6× floor is judged on `Sgemm`, `Ssyrk` **and** `Ssymm` (one parallelism
    class) and measured-not-judged on `Strsm`, whose floor is deferred to that
    measurement plus a stated model (#37). `STRSM_FLOOR` is left empty in the
    script with a comment saying it may only be filled by a ratification recorded
    in `DESIGN.md` — the deferral is mechanized, not remembered.
  - Both rates come from **one** invocation with the thread count in the benchmark
    name (`Scale/Sgemm/n=4096/threads=8`), because `-cpu=1,8` distinguishes rows
    only by the `-N` suffix that `bench_stat` and `bench_expect` strip — benchstat
    would aggregate the one- and eight-thread samples into a single row and the
    gate would divide a mixture by itself and read 1.0×.
  - The parallelism is checked rather than assumed: each row declares the
    GOMAXPROCS it set and the workers the library used, and both must equal the
    thread count in its own name. A threads=8 row that ran on one worker and a
    threads=1 row that ran on eight both produce 1.0×, and both are measurement
    failures dressed as performance failures.
  - Flop counts are re-derived by the gate for all four routines
    (`2mnk`, `kn(n+1)`, `2mnk`, `nm(m+1)`), formula string included, so the
    numerator of a scaling ratio is verified rather than asserted.
  - The README's published numbers are re-measured by the gate that ships them:
    rows keyed by **CPU model** (never hostnames — `.keel-hosts` is gitignored
    infrastructure), each carrying its denominator, each within 5% of this run; and
    any `GFLOP/s` figure outside that block fails the gate outright.
  - Bitwise determinism against the serial nest at threads 1, 3 and 8 — 3 because a
    row-partition off-by-one hides at every power of two — since splitting the MC
    loop reassociates nothing and a tolerance here would be hiding something.
  - It runs `scripts/gate-p4.sh` (which runs `gate-p3.sh`, which carries P2's
    audit) rather than restating any absolute bar. "≥6× single-thread" is a ratio
    whose denominator this phase is chartered to *improve*; a parallel nest that
    slowed the serial path would make the bar easier, and a bar that falls when the
    code gets worse is not a bar.
- `docs/toolchain-notes.md` T16 (issue #41): on arm64, whether `a*a + c` is
  FMA-fused depends on whether the compiler **constant-folds** it first, and
  `-race` defeats the folding — so one source line yields `0` in a plain build and
  `2^-24` under `-race`, on the same machine and toolchain. Both readings are
  spec-compliant. Found by `gate-p5.sh`'s race criterion, which is the first thing
  in this project ever to run `-race` on the dev host: `internal/vec`'s
  `TestSpecMulAddIsFused` computed its *unfused* witness as `a*a + c` and its own
  vacuity guard fired rather than comparing the fused answer against itself. It had
  been passing everywhere for a reason nobody had written down — on amd64 because
  gc does not contract `x*y+z` there at all, and on arm64 only because the witness
  was folded before code generation. The witness now writes `float32(a*a) + c`,
  which forbids fusion by the spec's own rule: state the rounding you require
  rather than inheriting whatever the optimizer chose.
- `docs/toolchain-notes.md` T17 (issue #42): `archsimd`'s partial slice load/store
  are not `checkptr`-safe, so **`go test -race` is a fatal error** — not a warning —
  on any keel call whose length is not a multiple of the vector width.
  `LoadFloat32x16SlicePart` and `StoreSlicePart` reach their masked operation by
  converting `&s[0]` to a full-width `*[16]float32` inside an `unsafe` helper; the
  mask keeps the *instruction* in bounds, but `checkptr` instruments the
  *conversion* and cannot know that. Reproduced standalone on linux/amd64 with no
  keel code involved, and — the part that isolates the cause —
  `-gcflags=all=-d=checkptr` alone reproduces it identically, so this is not the
  race detector and cannot be dodged with race options. It is also
  data-dependent: it fires on how much room the *allocation* has past `&s[0]`, not
  on the slice's length, so a call site can be quiet for a whole suite and abort
  after an allocator layout change. Two consequences, and the second is the one
  that matters for v0.1.0: `gate-p5.sh`'s race criterion is unmeetable on amd64
  while keel calls these, and **any user who runs `go test -race` on their own
  code crashes inside a library they did not write**. A `checkptr`-clean local
  workaround is confirmed (copy the remainder into a full-width stack array, use
  the full-width `Load…Slice`/`StoreSlice`, which convert no pointer; cost is one
  64-byte zero-and-copy on the tail iteration, never in the K-loop) but is *not*
  applied: it changes the "remainders use masked partial loads" story, so the
  disposition is #42's to settle rather than a drive-by fix.
- `scripts/gate-p5.sh`'s race verdict now classifies a `checkptr` death as its own
  outcome, naming T17 and #42 — still a **FAIL**, because naming a cause is not
  meeting a criterion. Its diagnostic for the generic case also prints the *head*
  of the failing-test detail rather than the tail: on a multi-package failure the
  `--- FAIL:` lines that identify the cause precede the per-package summaries, and
  a `tail` had been dropping exactly the lines worth reading. Its summary of the
  delegated P4 gate also counts that gate's own verdict lines rather than every
  line containing the word: a bare `grep -c FAIL` matched gate-p3's summary line
  *inside* gate-p4's log ("47 PASS / 0 FAIL"), so a green delegated gate was
  reported as "65 PASS / 1 FAIL". The verdict itself was always taken from the
  delegated gate's exit code and was correct; only the number beside it lied.
- `keel.L1Chain()` and `keel.KernChain()`: the *advertised* dispatch chains, per
  level, as functions rather than as prose. They answer a different question from
  `AvailableL1Backends`/`AvailableKernels`, which report what is runnable on the
  machine in hand and are properly shorter on a host without AVX-512. This is the
  claim keel makes about itself, so `gate-p5.sh` reads it and checks it against
  `DESIGN.md` §4/P5 and against the backends that actually have implementations —
  which is how #40 was found. A claim kept in a function can be checked; a claim
  kept in prose can only be believed.

- **P5 stage 1 opens with the four ruled one-liners, and the lint criterion is
  green** (#38, #39, #34, #10):
  - **#38 — `Isamax`'s negative-zero case was two positive zeros.** `-0.0` in Go
    source is a *positive* zero: the minus applies to the untyped constant, and
    constant arithmetic has no signed zero to produce. So `[]float32{0, -0.0}` was a
    case that could not fail, and it had been the only negative-zero coverage for
    four phases. It now constructs the value the way `internal/vec`'s spec tests
    already do (`float32(math.Copysign(0, -1))`) and covers five cases instead of
    one: both tie orders, all-negative-zero, and `-0` losing to a positive and to a
    negative nonzero — the last two being what a sign-confused comparison would fail
    outright rather than by a tie-break. **They pass on all three backends on all
    three amd64 hosts**, so the vacuous test was hiding nothing: the test was the
    defect, not `Isamax`.
  - **#39 — the spill audit's `os.RemoveAll` reports instead of discarding.** Not
    load-bearing for correctness (the gate reads `dir/<fn>.html`, never the scratch
    dir), but each leak strands the private 18MB `GOCACHE` this function creates
    inside a gitignored directory nobody looks at. A silent `_ =` would make that
    invisible, and failing the audit over it would let a `chmod` suppress an SSA dump
    that was produced correctly — so it names the leak on stderr and keeps the
    primary error.
  - **#34 — `roofline 0.0%` no longer renders a not-applicable as a measured zero.**
    `p3_denominator`'s contract is unchanged: 0 remains the right sentinel to
    *return*. Only the rendering changed, and **the condition is `roof == 0`, not
    `src == openblas` as the issue proposed**: an issue-bound host whose `min()`
    picked the reference (`why=reference`) has a real roofline that was computed and
    compared, and printing `n/a` there would hide a number instead of a hole — the
    same misreading in the other direction. Test the sentinel, not a proxy for it.
  - **#10 — P0's criterion states fusion rather than an encoding.** It named
    `VFMADD231PS`, which is the form a hand-written K-loop wants and not what
    go1.26.5 emits (`VFMADD213PS`), so as written it would have failed a toolchain
    that satisfied what P0 actually needs: one instruction doing the multiply and the
    add with a single rounding. `gate-p0.sh` has always checked the property — one
    `VFMADD{132,213,231}PS` and zero separate `VMULPS`/`VADDPS` — so this brings
    `DESIGN.md` into line with the check rather than the reverse. Which operand order
    it is stays consequential and stays tracked where the consequences are: the
    roofline section, where 231-with-broadcast versus 213 is `I = 2.875` versus
    `4.625`.

- **An instrument for the retention gap, before any theory about it** (#26 —
  the blocked `Sgemm` keeps ~90% of its own dispatched microkernel on both Zen
  hosts and ~77% on janus). `internal/block/nest_bench_test.go` +
  `scripts/retention.sh`, and three properties it was built to have:
  - **The decomposition is measured, and its residual is reported.** One blocked
    `Sgemm` splits into `nest-no-pack` + `pack-a` + `pack-b`, so
    `residual = full − the three` is what the split does not explain — printed as a
    line of the table rather than absorbed into whichever part is under
    discussion. `nest-no-pack` packs one set of panels outside the timer and reuses
    those buffers for every block: wrong values, nothing reads the result, and
    identical cost structure (same microkernel calls at the same `kk`, same
    buffers, same C traffic, same beta pass, same fringe path). What it drops is
    named where the residual is defined: the pack calls, the cache interference
    between packing and the kernel that follows, and `gemm`'s three per-call
    allocations (`bp` alone is a zeroed 3.1 MB at n=2048).
  - **Retention becomes a ratio instead of a quotient.** `gate-p3.sh` prints it
    from two invocations with two peak measurements and says so, because
    `bench_ratio_lo` cannot reach across two CSVs. `BenchmarkNest` measures the
    microkernel *in the same invocation* at the depth the nest actually calls it
    with, so retention is bounded by both confidence intervals. It is still a ratio
    of medians and the script still says so.
  - **The parts provably walk the blocks the shipped nest walks.** The parts share
    one block generator, `nestBlocks`, which is a copy of `gemm`'s three outer
    loops — exactly the kind of copy that drifts. So it is not trusted by
    inspection: `TestNestBlocksDriveTheSameGemm` drives a full pack-and-multiply
    GEMM from the generator alone and requires it to equal `Gemm` element for
    element over seven shapes (remainders in each dimension, sub-tile sizes,
    `k=1`, and both non-square orientations). A drifted bound, a missed remainder
    block or a B panel packed at the wrong `(jc, pc)` fails it. `block.go` gained
    only `plan()`, the clamp arithmetic `gemm` already did, extracted so both sides
    read the blocking from one place.
  - Shape is a sub-benchmark dimension rather than a dispatch, which answers #26's
    third candidate — does the gap track the host's *class*? — with no
    `KEEL_KERN_CLASS` pinning: both shipped shapes' retention is measured on every
    host. It also avoids a second copy of `selectKern` in a package that cannot
    import the root one.
  - `scripts/retention.sh` is **not a gate**: it certifies nothing, moves no
    criterion, and exits 0 whatever it finds. `decompose` runs the standard gate
    methodology because its numbers are meant to be quoted; `sweep` (KC/MC/NC over
    a coarse grid at 2048³) runs at `-count=5` and is labelled EXPLORATORY in its
    own output — a point it nominates has to be re-measured under the full
    methodology before it could become a default. `NC` stops at 2048 because
    `plan()` clamps it to `n`, so every larger value is the same measurement under
    another name.
- **`BenchmarkPackDirections`, and the correction it exists to make measurable**
  (#21). Both #21 and #26 stated which pack direction transposes, and both stated
  it backwards: `APanels` passes `!trans` as `depthContig` and `BPanels` passes
  `trans`, so at `NN` — the shape every benchmark in this repo runs — it is **A**
  that takes the transposing branch and B that gets `memmove`, and the case where
  both directions transpose at once is `NT`. `internal/pack`'s own package doc had
  the rule right; the issue text misapplied it. The fix is a benchmark dimension
  rather than a prose edit: both directions, all four flag combinations, at the
  block shapes the nest uses, counting valid elements only (padding is zero-fill,
  not data movement).
  - It refutes #21's premise. The `copy`-based branch — the one the rule calls
    "already vectorized" — is **2.8× slower** than the transposing branch it
    replaces on the A side at `TN` (0.77 vs 2.19 Gelem/s, dev host, pure-Go
    `internal/pack`), because its run length is `blk = MR ∈ {2,4}`: 8 or 16 bytes
    per `copy` call, ~27,600 calls per pass. Cost scales as 1/`blk` and flattens
    once the run reaches 64 B, and removing the source stride entirely changes
    nothing (1.0–1.1×), which refutes the locality explanation that was tried
    first. So the rule "copy the contiguous axis" is right for B (`NR = 32`) and
    wrong for A, and the 16×16 `Permute` transpose is no longer the first thing to
    try there.
- **`internal/pack` gets tests of its own** (#45), before #21/#22/#36 change its
  loops. Four invariants that the root package's differential tests cannot see,
  because they are visible at this boundary and not in `Sgemm`'s output:
  - The packed **layout is checked against the doc's formula**, written out as index
    arithmetic on the source rather than by calling anything in the package — so a
    layout change has to be made twice, by someone who means it. A layout that
    changed consistently with the microkernels would otherwise pass every existing
    test while breaking the contract future kernels are written against.
  - "**The zeros are written, not assumed**" becomes an assertion. A poisoned buffer
    catches an unwritten slot; the case that actually happens in `gemm` is a slot
    holding a *previous pack's* plausible value, so a large block is packed and then
    a smaller one into the same buffer, with every slot the second claims required to
    hold what the second put there. Padding is compared bitwise, since a stale −0
    passes `== 0`.
  - **The two branches are required to be bit-identical**, which is the guard that
    makes #21's branch-selection change safe: the same logical matrix packed from a
    row-major source and from its transpose, over ±0, ±Inf, NaN, `MaxFloat32` and
    the smallest denormal. The one input where they provably differ — a *signalling*
    NaN, which `copy` moves untouched and `alpha*v` quiets — is documented as a known
    asymmetry rather than tested as a requirement, since no IEEE operation produces
    one and BLAS specifies nothing about NaN payloads.
  - `ALen`/`BLen` are **exactly enough**, and one float short panics *with both
    lengths named*. Asserting the message and not merely the panic is the point:
    deleting the guard outright still panics, from the panel re-slice, so a test that
    accepted any panic would pass over code with no guard.

  Nine mutations were applied to `pack.go` to check the suite can fail — dropped
  zero-fill in each branch, a transposed layout index, an ignored `alpha`, an
  off-by-one `valid`, an `ALen` that forgets the ragged panel, a deleted length
  guard, and `nb` taken from the buffer length instead of the shape. Eight are
  caught, each by the test that should catch it. The ninth (dropping `count == 0`
  from the early return) is an equivalent mutant, not a bug, and the test comment
  says so rather than the suite being tightened around a distinction that does not
  exist.

- `docs/toolchain-notes.md` T18: a **loop-invariant vector constant is
  re-materialized every iteration**. `BroadcastInt32x16(const)` written inside a loop
  stays inside it (`MOVL`/`VMOVD`/`VPBROADCASTD`, 3 instructions per iteration); the
  same constant written above the loop stays above it, and the function's total size
  barely changes — the instructions are relocated, not removed. The repro carries its
  own control (both loops, differing only in where the source puts the constant), and
  the entry is explicit that it cannot rule out the allocator rematerializing under
  the register pressure of a real four-accumulator kernel, which has to be re-audited
  in place. Answers #8: combined with T8's CSE, which shares the mask across the four
  unrolled `Abs512` calls *within* one iteration, the cost is 3 instructions per 64
  elements rather than 12 — so the answer to "does it hoist" is no, and the answer to
  "does it cost 12" is also no.

- **`scripts/l1-bench.sh`**: A/Bs the Level-1 routines at all four of
  `bench/bench_test.go`'s sizes — 1 KB, 16 KB, 256 KB and 4 MB of float32, i.e. L1-,
  L2-, L3- and memory-resident on all three hosts — between an arbitrary base ref and
  the working tree, on every configured host, under the standard methodology with
  benchstat p-values. Written for #47, whose loop-shaping change *lengthens* four of
  the ten loops while shortening six, and whose static counts therefore cannot say
  which way the routines move: at 4 MB per call the loop body is not the limit. Not a
  gate — it certifies nothing, moves no criterion, and exits 0 whatever it measures.
  The base build comes from a detached `git worktree`, not from stashing: a stash
  would mutate a tree another long-running measurement may be reading, and it would
  make the two arms differ by whatever else happened to be dirty.

### Changed

- **The share denominator was the one reading in the log with no range and no marker**
  (`scripts/gate-p5.sh`, `scripts/remote-exec-test.sh` §9g, ruled 2026-08-28 on #6). The per-host ceiling
  line hand-built its own `+/- %.2f%%` and never called `bench_describe`, so rule 20's disclosure — the
  range beside the interval, and `RANK-WINDOW-BLIND` when a printed zero sits under a span refuting it —
  reached every judged rate and **not** the number all three of them are divided by. keel-skx's
  confirmation run printed `ceiling: compute 1444 GFLOP/s +/- 0.00%` on exactly that line. Routed through
  the one renderer; the nine verdict lines that cite the ceiling now cite `CEIL8P`, the disclosure's own
  value token, because `%.4g` and benchstat's raw CSV field differ on a 5-significant-figure rate and a
  log stating its denominator two ways is the defect one layer along. **No fixture over `bench_describe`
  could have caught this** — the renderer was correct and simply unreached — so §9g extracts the three
  lines from `gate-p5.sh`'s own bytes and evals them with `info` stubbed, plus a positive-controlled
  static check that no site still renders the raw field.
- **Era `pinned8` is CLOSED, and `keel-skx` is the first host in `scripts/host-baselines.tsv`**
  (`scripts/measurement-eras.tsv`, `scripts/judged-runs.tsv`, `archive/pinned8/`, #6). The spread arm's
  21 archives are preserved and indexed (`INDEX-spread.tsv`: 9 `ladder` at `instrument=v2`, 12
  `clock-window`), which fills `transition_archive`; all three judged CPU models appear in both arms
  (9R14 11/4/3, 9R45 10/5/3, 8124M 6/15/3) and gnr's 8-free/0-pinned stays the stated exclusion.
  **What forced the preservation was a live defect, not the era condition**: the published README rows
  cited their evidence as `build/` paths and `build/` is gitignored, so 24 green numbers rested on logs
  no reader could open — #114 at the *publication* layer. `readme-numbers.sh` now cites the path as
  invoked instead of a basename under a hardcoded `build/`, and the page was regenerated against the
  preserved logs; **no rate moved**. skx's three share baselines are **30.80 / 31.40 / 30.35**, medians
  over the era's two judged archives — with take four **recomputed under #116**, because a share is net
  of both intervals and the #116 fix landed *between* the two revs, so pooling its printed 29.7/31.3/29.8
  would have averaged across a CI-instrument change and registered two of three baselines 0.35 and 0.15
  points **low**, i.e. permanently looser. Baselines and witness landed together: a witness alone is the
  `owing` state, which is FAIL.
- **Rule 20's `RANK-WINDOW-BLIND` marker keys on the width as *printed*, not as stored** (`scripts/bench.sh`,
  DESIGN.md §5 rule 20 amended 2026-08-28, ruling on #6). The printed line is the assertion, so `± 0.003%`
  renders `0.0%`, makes a reader the identical claim as an exact zero, and earns identical scrutiny; the
  second half is the range refuting that claim, at one quantum of the same display. **16 of 114 archived CI
  readings print as `0.0%` and only 4 are exact**, so the reserved middle was 12 readings wide. Output only,
  and exact-zero ⊂ display-zero, so no marker is lost and no verdict moves. `bench_describe` had no test
  before this; six fixtures now cover it (`remote-exec-test.sh` §9f) and the archived path stays unexercised —
  no CSV in the tree carries #116's seven columns.
- **The published numbers are per-row medians over an era's archives, and the verdicts beside them still are
  not** (`scripts/readme-numbers.sh`, #6; the ratified repair). The generator now takes any number of
  `gate-p5` logs and reduces each cell across them (§5 rule 16), every cell naming its own estimator —
  currently `median of N=2 archives` on all 24 rows, over rev `969c360` (judged) and `6ba6566`. Verdicts do
  **not** pool: a verdict belongs to the gate that rendered it and two cannot be averaged, so the judged log
  is identified **by content** rather than by position, two judged logs are refused outright, and argument
  order is provably irrelevant — both orders print skx `Sgemm` 1T as `66.01`, which is `(66.18 + 65.84)/2`
  recomputed by hand. The cross-check against each run's own printed share moved per-archive, because a
  pooled median matches no single log by construction. **Host coverage moved with the era**: `keel-skx`'s
  eight rows are born, and `Intel(R) Xeon(R) 6975P-C` leaves with 8 free archives and 0 pinned — a stated
  exclusion in `scripts/measurement-eras.tsv` since 2026-08-22, not an unmet condition. A departing host is
  the one change 24 green rows cannot report, so the generator diffs the CPU-model set against the **last
  published** block — not the working copy, which would print the notice on the first run and drop it on the
  second — and says it in the caption, failing closed when `HEAD` is unreadable. **Confirmation semantics,
  stated because they are weaker than a green:** rows regenerated *from* these archives agree with these
  archives by construction, so criterion 9's meaningful green arrives on the **next** judged run.
- **`scripts/detach.sh` forwards no environment, so a detached run can be a different program than the
  same command typed directly** (measured 2026-08-22; no code change, recorded and worked around at the
  call site). The generated runner is `cd $ROOT` plus a `printf '%q '` of argv and nothing else, so a
  `KEEL_*` variable exported in the caller's shell is simply absent inside the tmux session. Found by
  launching the founding campaign's fleet with `KEEL_FLEET` set to the judged sizes and watching it bring
  up the **exploration** sizes instead (`c7a.2xlarge`, `c8a.2xlarge`); killed, torn down twice, both
  instances confirmed `terminated` by instance-id, ≈$0.09. Then positively controlled rather than
  reasoned: the detached run printed `KEEL_PROBE_VAR=[<unset>]` where the identical command run directly
  printed `[set-by-caller]`. Compounded by `tmux set-option -s exit-empty off` (line 114), which keeps the
  server alive between sessions, so a later run inherits whichever caller first started it. The fix is not
  invisible forwarding: launches that need a variable use the inline `env VAR=... ./script` form at the
  call site, which costs no `scripts/` lines and is what a judged run wants anyway — a stray `KEEL_*`
  in an operator's shell cannot leak into a measurement it does not name.
- **Ceiling readings taken before the hoist are marked `instrument=v1` and era-scoped, NOT re-adjudicated**
  (ruled 2026-08-22 on #115; `scripts/measurement-eras.tsv`, DESIGN §5 rule 18). The v1 bias was host- and
  mask-dependent **and varied per run** — that variance *was* the ceiling scatter that drew attention in the
  first place — so no correction formula recovers an archived share honestly, and one applied anyway would
  publish a precision the data never had. Nothing needed recovering: **no bar was typed from those readings, no
  registry row landed from them, no verdict rests on them.** The counterfactual is recorded because it is the
  argument for the pre-commitment discipline rather than for the repair — **had `CEIL_FRACTION` been typed at
  the 74.3% the v1 ceilings supported, this repair would now be forcing a published bar retraction with every
  verdict derived from it.** A defect in a denominator resolved into a version label. The version is emitted by
  the instrument in its own declaration row (`instrument=v2`), not only in prose, because a reader comparing two
  archives needs the noun's version where they read the number. `pinned8` is therefore founded a **third** time
  and still has never closed; a fresh era id was considered and refused, since the era id bounds
  *registrations* while the artifact fields (`doms=`/`nodedoms=`, `instrument=`) bound *configurations*, and
  the re-founding returns nobody's exemption — `scripts/judged-runs.tsv` has no data rows and
  `scripts/host-baselines.tsv` is header-only, so no host has ever spent a `BASELINE` in this era.
- **§5 rule 5's spread mask is FINAL as the standing judged placement, and its cost is now stated as measured
  rather than as asserted** (ruled 2026-08-22 on #6; §7 rule 7 extended with it). T-52 removed the only standing
  argument against spread — the ceiling scatter was the harness, not the placement — so the choice was decided
  on spread's own profile: deterministic, and bandwidth-honest on the L1 rows by 5.96×/4.65×, which a confined
  mask would have buried. **The ruling's "4–6% GEMM cost" did not survive being recomputed from the archives and
  is not what landed.** Median-of-medians on the judged 8-thread rows at n=4096, confined against spread, as
  `Sgemm / Ssyrk / Ssymm`: keel-zen4 **4.32 / 5.49 / 4.30%**, keel-zen5 **1.16 / 3.40 / 0.46%**, keel-skx
  **1.70 / 0.30 / 1.17%**. keel-skx is a **null band, not a third data point** — its two masks are byte-identical
  (`nodedoms=1`), so its 0.30–1.70% is what this cross-era, cross-revision comparison reads when the mask does
  not change at all. zen4 clears that band; **zen5 straddles it**, so the honest statement is a few percent on
  Zen 4 and at-or-near the noise floor on Zen 5, and one fleet-wide figure would have been zen4's number wearing
  the fleet's name. The one-L3-copy-per-CCD mechanism is recorded as **unmeasured** beside it. **A fourth
  limitation is stated inside the number, and it is an argument against spread that T-52 did not remove**
  (§5 rule 12): the GEMM-class rows are tight under spread (CIs **0.045–0.41%**, two outliers at 1.41% and
  1.85%) but **`Strsm` is not, and this mask is why** — its 8-thread interval goes ±2.32/2.97% → ±6.52/9.48/11.84%
  on zen4 and ±0.75/0.87% → ±5.13/10.28/13.24% on zen5, a 3–15× widening, while **keel-skx's identical-mask
  control does not move** (±0.50–1.30% → ±0.82–1.00%). In raw samples on zen5 that is 399–412 GFLOP/s confined
  against 334–455 spread: a 3.3% total spread becoming 36%, consistent with this mask reweighting the bimodal
  distribution T-45 and T-52 both found rather than with a new cost. It bears on one decision and not on the
  placement, so it is disclosed rather than netted out: **whether `Strsm` can be judged at all under a mask
  that gives it a 5–13% interval is open, and `STRSM_FLOOR` stays empty until it is answered.**
- **Both of P5's bars are SUSPENDED TO EMPTY for the era-founding run, and one of them was RED when it happened**
  (`scripts/gate-p5.sh`, `scripts/readme-numbers.sh`, `DESIGN.md` §4/P5; ruled on #6, 2026-08-22). `CEIL_FRACTION`
  goes because its denominator is the compute ceiling *measured under the mask in force* and the confined mask is a
  different instrument; **counted rather than assumed, 51.0 judged exactly one run** (`build/gate-p5-d2fe477.log`:
  six rows, two models, all PASS, and the argmin row read 53.6% again — clearing by exactly its own 2.6-point
  margin, a reproduction to one decimal rather than a defect). `STRSM_FLOOR` joins it and becomes era-scoped, for
  a reason of its own: it is a ratio over the **1-thread** rate, the one arm this fleet has measured bimodal and
  placement-sensitive, and rule 5's controls are explicitly silent there. **The suspension removes three live
  FAILs — zen5 6.982/6.756, zen4 7.257/6.881, skx 6.806/6.637 against 7.0× — and that is disclosed first because
  it is the shape of a criterion weakened to pass.** What makes it a re-derivation: the ruling scoping the bar to
  the era was made with those three reds already on the record, and §5 rule 17(d) predates them. **Pre-committed:
  if the spread rows land below 7.0× net of CI, the value does not get typed** — a bar under 7.0 would be
  loosening after a red, so it goes to Scott as a finding and the class stays REPORTED. `BASELINE_MARGIN` is
  deliberately **not** suspended: 2.6 points is an *input* to the formula rather than an output of the run, so the
  circularity clause has nothing to say about it, and what suspends it this era is the artifact — both registries
  are header-only for `pinned8`, so the branch that subtracts it cannot execute. Five predictions are
  pre-registered in DESIGN §4/P5 before the run (§5 rule 11), including **the control that tests the amendment
  hardest: `keel-skx` must not move**, its L3 being per socket so the spread enumeration is degenerate there.
- **The README caption could print "every pair clears the bars" when NO bar was in force** (`readme-numbers.sh`).
  Vacuously true, and it reads as a pass — §5 rule 8's defect published in the one file a reader checks. Both
  suspended bars now render in words (the `>= %sx` hole was already anticipated for the ceiling bar and not for
  Strsm), and the both-suspended case prints **NONE of the pairs was judged**. A second, worse case was found by
  driving it: the bars are read from this tree and the rows from a **log**, so a log written while a bar was in
  force carries FAIL verdicts a suspended bar cannot produce — the constants read-back compares two scripts and
  cannot see it. That combination now **refuses** rather than choosing which sentence to publish. Stated inside
  the number (§5 rule 12): the both-suspended branch is **unexercised by any archived log** — every log in
  `build/` predates the suspension and now takes the refusal instead, so the string was rendered against one of
  them before the refusal landed, which proves formatting and not reachability. **And the founding run will not
  reach it either**, which is the find that matters: a `BASELINE` verdict line carries no scaling clause, so this
  program refuses the whole log the moment one host renders BASELINE — `keel-skx` does, this era, which makes it
  the blocker on regenerating the README as medians over this era (#6).
- **§5 rule 5's mask is AMENDED to one core per cache domain, and `pinned8` finalizes on the spread form**
  (2026-08-22, ruled on #6; `scripts/remote.sh`, `DESIGN.md` §5 rule 5, `docs/hosts.md`, `docs/gates.md`,
  `scripts/measurement-eras.tsv`). The 2026-08-21 form took the first eight distinct physical cores of one NUMA
  node in ascending order, and **on EPYC 9R45 that is definitionally one CCD** — sysfs reports `index3`'s
  `shared_cpu_list` as exactly eight cores wide, so "eight cores, one node" and "one L3" were the same set by
  construction rather than by accident, on the host the fleet's widest readings come from. Measured against it
  (T-45, three controls): 8-thread stream **dot 56.26 → 335.48 GB/s (5.96×)** and **axpy 75.89 → 353.0 GB/s
  (4.65×)** one-core-per-CCD, with the repeat arm reproducing to **0.10%**, the 1T arms invariant across masks, and
  arm A's 1T axpy matching the untouched harness to **0.01%** (66.29 vs 66.30) — so the delta is placement and
  not the instrument. The spread arm also **exceeds free placement by 1.69× / 1.55×**, which is what makes this
  publication honesty and not precision: the confined mask would have regenerated the README's bandwidth-bound
  `dot`/`axpy` rows several-fold low, and **§5 rule 16 forbids a systematically underselling reference in exactly
  the spirit it forbids the max-draw overselling one**. The enumeration is stated in the law as a function of
  topology alone — partition a node's distinct physical cores by their **highest cache `level`** (keyed on the
  lowest cpu id in `shared_cpu_list`), order domains ascending by that key, round-robin one core per domain per
  pass taking the lowest unused core — so nothing here is a chosen mask. `level`, never `index3` by number, is
  `bench/ceiling_test.go`'s `llcBytes` discipline reused: the harness makes no microarchitecture claim it would
  have to maintain. **A missing cache level now refuses exactly as missing thread siblings does** (status 121,
  amended refusal string), because falling back to consecutive cores would file a confined reading under this
  era's label — the era-ledger forgery one layer in from free placement. **On `keel-skx` the amendment is
  degenerate and that is the correctness argument**: one L3 domain per socket means the spread form returns
  consecutive `0..7`, byte-identical to the old answer and the right mask there, so this refines the 2026-08-21
  form rather than replacing it — and the new arms are therefore driven by fixtures written on purpose (a
  12-domain EPYC node, a 2-domain wrap-around, a cache-blind cpu, no cache at all) instead of resting on an
  unchanged expectation. The 24 archives at `archive/pinned8/` are preserved unedited as the **provisional
  confined arm**, superseded by the amendment, and they **self-identify** by carrying no `doms=` field: the two
  arms are separable by their own witnesses with no file touched. `transition_archive` stays `—` because the
  spread-form campaign is now what fills it; the gnr hole no longer figures, the 2026-08-22 ruling having scoped
  the both-arms condition to **judged** hosts, so `keel-gnr` is a stated exclusion in the era row rather than an
  unmet condition.
- **The pin line records the mask's SHAPE, because `GOMAXPROCS` cannot witness it** (`scripts/remote.sh`,
  `scripts/bench.sh`, `scripts/gate-p5.sh`, criterion 4). Amending the banner surfaced the gap: the gate would
  have asserted a spread its archive could not show, since `gomaxprocs=8` reads identically under a confined and
  a spread mask — the field the old criterion cross-checked is blind to the very defect the amendment fixes. The
  line now carries `doms=` (each selected core's domain, in mask order) and `nodedoms=` (how many the chosen node
  offered), and the invariant `distinct(doms) == min(width, nodedoms)` **and** `max_count − min_count ≤ 1` is
  checked off the artifact per row. **The surprise: a fully confined mask has imbalance 0, not 7.** Eight cores
  in one domain are perfectly *balanced* over that one domain — predicted 7, the fixture read 0 — so balance
  cannot see confinement at all and `nodedoms` is load-bearing rather than belt-and-braces; the two terms catch
  disjoint defects. A row missing both fields is **`unmeasured`, not a failure**, naming the provisional confined
  arm as exactly that shape, since a pre-amendment archive is not a broken post-amendment one (§5 rule 6). The
  invariant was lifted into `bench.sh` as `bench_pin_spread` so it is drivable from log fixtures: the criterion
  needs a fleet, and the confined arm it must still classify is a shape **no working host can now produce**.
  `remote-exec-test.sh` grew section 9d over that seam (7 cases) and every pre-existing pin fixture gained cache
  topology, without which all of them would have started refusing. **`scripts/` budget, disclosed rather than
  netted out** (CLAUDE.md, "the apparatus pays its own way"): this pair of entries lands net shell with **no**
  library, kernel or routine beside it: **+279 net lines in `scripts/`, moving `gate-docs.sh`'s reported ratio
  1.63× → 1.66×** as measured 2026-08-22, and the move is attributable to the numerator alone because the library
  term does not change in this commit (`verify_test.go` is a test and is not counted). It is the instrument the
  era's founding campaign is about to be measured with, so deferring it wastes fleet time rather than lines, and
  the paydown is owed post-tag — but the ratio moved the wrong way and saying so is the rule's whole mechanism.
- **The `pinned8` era's arm is preserved and tracked at `archive/pinned8/` — 24 archives, `INDEX.tsv`, mapped to the
  free arm by `cpu_model`** (#6). It existed only as untracked `build/` output, one `make clean` from gone, on an era
  whose evidentiary half cost a three-host fleet to measure. Membership is the **measured predicate "carries a
  `keel-pin:` line"**, not a revision range: what makes an archive pinned is that the mask applied to it, and that
  line is the only witness. Applying the predicate also returned exactly 35 *unpinned* archives whose names match the
  35 already in `archive/free-placement/`, an independent check that the free arm was preserved whole. All 24 report
  the same `mask=0,1,2,3,4,5,6,7 width=8` and `gomaxprocs=8`, which is what rule 5's "fleet-wide and never
  selectively" looks like in the evidence. **The count corrects a figure published hours earlier on #6 as 15**: that
  was the two runs in front of me, and the predicate found `e64b34e`'s nine as well.
- **`transition_archive` stays `—` and `pinned8` stays PROVISIONAL, deliberately** (`scripts/measurement-eras.tsv`;
  #6). Granite Rapids (`Intel(R) Xeon(R) 6975P-C`) appears 8 times in the free arm and 0 times in the pinned one,
  being a characterization host absent from the pinned fleet, so "the same fleet measured under the old instrument
  and the new one" holds for three of the four models the free era touched. That is arguably consistent with the
  condition rather than a breach — a characterization host is not part of the fleet the era governs — but reading it
  as met is a judgment about what "the same fleet" means, and certifying a condition with a stated hole in it is not
  a call to make in the commit that files the evidence for it. Preserving is unambiguous and done; closing is asked.
- **The reported 8-thread stream ceiling is a single-CCD figure, ~6× below the socket, because §5 rule 5's affinity
  mask packs all 8 threads into one CCD** (`keel-zen5`, EPYC 9R45; #6). sysfs gives `shared_cpu_list` for
  `index3` as exactly 8 cores wide, so `keel_pin_mask`'s "first 8 distinct physical cores in order inside one NUMA
  node" is definitionally one L3 domain. A three-arm probe — mask `0-7`, then `0,8,16,…,56`, then `0-7` again —
  measures 8-thread **dot 56.26 → 335.48 GB/s (5.96×)** and **axpy 75.89 → 353.0 GB/s (4.65×)**, which fully accounts
  for the era's collapse at `be5bb91` (dot 198.7 → 56.525, axpy 227.4 → 76.195) and then some: the spread arm
  *exceeds* the free era's own reading by 1.69× and 1.55×, so scheduler placement was leaving CCD bandwidth on the
  table before the mask existed too. Three controls: the repeated first arm reproduces itself to 0.10%
  (so not drift or thermal), the 1-thread arms are invariant across masks (so not the instrument), and arm A's
  1-thread axpy matches the gate harness to 0.01% (66.29 vs 66.30). **Consequence is forward-looking**: the figure is
  inert today — `scripts/gate-p5.sh` reports it and keeps the `min()` on the compute term alone, which stays the
  strict direction — but it is documented there as a candidate for that `min()`, and promoting a 6×-low bandwidth
  into a ceiling would understate the ceiling in the flattering direction. The compute ceiling is unaffected,
  needing no cross-CCD traffic, so DESIGN.md's strict-direction argument survives. **No headroom multiple is quoted
  here on purpose**: converting GB/s to a FLOP/s bound needs a declared DRAM traffic count, and as that same line
  says, no benchmark declares one — the honest statement is the measured bandwidth ratio, not a modelled headroom
  derived from a denominator this tree does not have. **The mask was not changed** — rule 5 is
  law and it pins fleet-wide, never selectively, so whether to spread across CCDs is Scott's call (#6). `keel-zen4`
  is *inferred, not probed*: its harness bandwidth fell comparably (103.25 → 39.265 GB/s at 8 threads) on the same
  vendor topology. `keel-skx`, which has no CCDs, *gained* under pinning (52.5 → 88.0 GB/s) and is the negative
  control. Probe is ad-hoc and deliberately untracked: it varies rule 5's mask and carries no provenance block, so
  it is **not citable as a keel measurement** and adds no `scripts/` lines.
- **The headline criterion's PASS and FAIL lines named a denominator retired two days earlier** (`scripts/gate-p5.sh`;
  #6). Both said hosts cleared "their class's bar … against their own single-thread rate", but the judged share has
  divided by each host's own measured 8-thread ceiling since the 2026-08-20 ruling; only `Strsm` still divides by a
  single-thread rate, and `$BARS` spans both classes. Each bar now names its own denominator instead, and the
  `gate-p4`-is-RED caveat — which disclaimed only what "divides by a single-thread rate" — now disclaims measured
  rates generally, having stopped reaching the headline criterion at all. Worst on the **green** path: the PASS
  string is what ships verbatim in a gate-green closing comment. A **fourth site** turned up afterwards, while
  adjudicating the confirmation log (`gate-p5.sh:1189`): it justified `NINDET = 0` by that same retired denominator
  — "this criterion's denominator is the host's own single-thread rate, of which there is exactly one". The
  conclusion survives, since each class still has exactly one denominator and so there is no candidate split, but
  the reason named the wrong one now that the criterion spans both classes. Lower severity than the three verdict
  strings by §5 rule 14 — a comment ships to the next editor, not into a closing comment — and fixed rather than
  filed, at no net `scripts/` lines.
- **`CEIL_FRACTION = 51.0`, typed for `pinned8` from the era-founding run, with its derivation set narrowed from
  three CPU models to two** (`scripts/gate-p5.sh`, `scripts/readme-numbers.sh`, `DESIGN.md` §4/P5; #6). The minimum
  judged row on `build/gate-p5-be5bb91.log` is `keel-zen5` `Ssymm` at 53.6% net of both intervals, less the same
  2.6 points of margin. **The bar fell 6.8 points and the cause is the denominator, not the kernels:** on both
  derivation hosts the ceiling outran every judged rate, so all six raw shares fell — zen5 1568.5 → 1999.5 GFLOP/s,
  +27.5%, against +6.2/+17.5/+9.0%; zen4 713.6 → 817.4, +14.5%, against +9.7/+13.7/+5.7%. zen4 is not zen5's mirror
  image but the same phenomenon: two of its net-of-interval shares rose regardless, pinning having collapsed the rate
  intervals (1.0–8.3% → 0.0–0.8%), and a share net of both intervals has three terms. The argmin moved with it,
  `Ssyrk` → `Ssymm`. `Intel(R) Xeon(R) 6975P-C` leaves `CEIL_DERIVED_FROM`: `gnr` is characterization, and
  characterization hosts are "never mixed into the citable set" — the 2026-08-21 ruling applied, not a new decision,
  and no verdict moves because the model is not in the fleet. Stated inside the number (§5 rule 12): **no Intel
  silicon derived this bar**, and the two models that did spread 53.6–89.9%, so one fleet bar is set by the weakest
  host. `keel-skx` does not register beside it — one draw is not an estimator (§5 rule 16) and its witness row may
  not land without its baseline, so it re-renders `BASELINE` on the confirmation run, printed as a debt by design.
  Found by exercising the twin against that log and **left for the README regeneration to fix, in the file it already
  has to touch**: `readme-numbers.sh` dies on a `BASELINE` verdict line, because that branch prints no `scaling …x /
  …x net of CI` clause and the parser requires four numbers. It fails closed, so nothing is published wrongly — but
  no log containing a BASELINE host can generate a caption at all, and the fix is a *counted* exclusion ("a published
  row is born from a judged run") rather than a skip, since a parser silent about what it never read greens like a
  clean one. Pre-existing: the era-founding run is simply the first log this class has ever appeared in.
- **`CEIL_FRACTION` is retired at the era boundary, and empty is its pre-registered state rather than its
  fallback** (`scripts/gate-p5.sh`, `scripts/readme-numbers.sh`, `DESIGN.md` §4/P5; ruled 2026-08-21 on #6, the
  same day 57.8 was ratified — §5 rule 17(d), a derived constant re-derives by its own formula over new-era
  inputs). 57.8 came from free-placement medians on the three `CEIL_DERIVED_FROM` models, and those models are
  `DERIV=1` — the fleet-bar hosts the BASELINE-REGISTERED class deliberately does not shield. Left standing it
  would judge their *pinned* readings against a *free* bar: the methodology delta booked as host drift, arriving
  through the bar rather than through the registry, which is the one error the era boundary exists to prevent.
  **Counted rather than assumed, it judged one complete run** — `gate-p5-33de3b2.log`, nine rows, no `keel-pin`
  line, and its RED is the observation that minted rule 17; `gate-p5-fdd23d4.log` cites it in the preamble and
  never reached a judged row. It never judged a pinned reading and now never will. Nor can the `pinned8` value be
  pre-typed, which is a construction and not a scheduling problem: the formula's inputs are the era-founding run's
  own outputs, so a bar derived from the rows it judges is cleared by its own argmin by exactly the margin every
  time, certifying arithmetic rather than silicon. So the transition run **reports** through the branch already in
  the gate, a reviewed commit types the value from those rows with its derivation printed, and the confirmation run
  is the first this era judges — #37's rhythm, and 57.8's own. Both copies empty together because
  `readme-numbers.sh` reads the gate's line back verbatim and dies on disagreement, which is why the second file is
  a second edit and not a second decision. The preamble's empty-fraction line had also kept the *previous*
  deferral's reason ("until the bandwidth term is measured on the fleet"), false since the ceiling was measured, so
  it now names the live cause; the era-founding log is this era's constitution and may not carry a stale one. The
  historical re-adjudication is undisturbed — a free-placement bar over free-placement archives is intra-era by
  construction, so its 35-of-105 stands and does not become a cross-era claim.
- **The BASELINE-REGISTERED class gets a synthetic driver before the era-founding run, not after it**
  (`scripts/exercise-baseline.sh`, ordered by Scott's ruling of 2026-08-21 on #6). The class decides which bar
  governs a host from the contents of two tracked files, and **its row loop had never executed**: no `frac`
  computed inside it, `new`/`owing`/`registered` all unentered. Its live debut would otherwise have been the
  pinning transition — the era-founding campaign — where a wiring bug costs a fleet-wide re-run and muddies the
  both-arms archive that campaign exists to produce. Three passes on one real host through the
  `KEEL_INSTRUMENT_BASELINE_DIR` seam: empty registry and empty witness (BASELINE on both criteria of the class,
  with the candidate rows a reviewed commit would land); witness landed and registry still empty (`BASELINE is
  spent` on both, plus the debt line — the branch #114's fix created, and unexercised a repo with one landed row
  would have renewed BASELINE instead); registry landed from pass 1's own fracs **with a wrong-era decoy row at
  99.0 written above the real rows**, so one pass drives the registered state and both arms of era scoping, since
  `baseline_lookup` returns the first match and a decoy below the real row would prove nothing. The discriminator
  is textual and declared before the run: a bar of `frac − 2.6` naming era `pinned8` confirms scoping, a bar of
  96.4 naming `free-placement` refutes it. The preflight spends no sweep and drives **both** of the seam's
  fail-closed refusals with `env -u`, so neither can pass by inheriting a variable the driver happens to have
  set; it also asserts from gate-p5's own bytes that the seam resolves *before* the gate's first remote call, and
  reads `CEIL_DERIVED_FROM`, `P5_JUDGED` and `BASELINE_MARGIN` out of the gate rather than restating them. Each
  read-back is keyed to its own pass's phrase and reports NO on no match, and the run finishes with a stamp audit
  over the parent *and* the three delegated logs — `^  TOKEN  ` followed by `[synthetic] `, plus zero signed
  `gate-pN:` lines anywhere.
- **The BASELINE-REGISTERED witness was a glob over gitignored output, so it was right on one machine and wrong
  everywhere else** (#114, fixed 2026-08-21 on Scott's direction, folded into the era commit). `baseline_prior`
  asked `build/bench-gate-p5-*-<host>-*.txt` whether a host had been judged before. `build/` is gitignored, so on
  a fresh clone, on CI, or on a second operator's machine the answer is always *no*: every host is new forever and
  `BASELINE` renews on every run. **A per-machine witness defeats single-shot exactly as thoroughly as the
  permanent exemption the class was built to kill**, and more quietly, because it fails only for readers who are
  not the operator. The witness is now `scripts/judged-runs.tsv` — tracked, keyed `(cpu_model, era)`, proposed by
  the gate beside the baseline row it spends. The trade is stated where the old scope disclosure was, not deleted:
  **automatic-and-invisible for reviewed-and-visible**. A session that lands neither row leaves the host
  unregistered and re-renders `BASELINE`; the debt line gate-p5 already prints is what makes that repeat visible,
  and the message that used to say "spent — this run's own archive is the prior log" now says spent *only once the
  witness lands*, because the old wording over-promised in the same direction the defect did. Keyed on the CPU
  model rather than the hostname, deviating from the ruling's wording on purpose and recording why: the registry
  next door keys on the probe string, and a hostname key would hand a renamed host a second exemption.
- **A baseline now belongs to the era of the instrument that measured it, and the pinning adoption is an era
  boundary** (DESIGN.md §5 rule 17 clause (d), ruled 2026-08-21 on #6). Read literally, "one `BASELINE` per host,
  ever" said skx must be judged against a baseline imported from `5ec5fea`/`33de3b2`. Scott's ruling refuses that
  on **misattribution** grounds rather than on convenience: those are free-placement readings, §5 rule 5 pinned
  placement fleet-wide the same day, so judging pinned readings against unpinned baselines would book **the
  methodology delta as host drift** — the cross-denominator sin the registry exists to prevent, arriving through
  the registry. An instrument change is therefore the "dated re-registration citing a named change" door opened
  fleet-wide: every host renders `BASELINE` once per era, registrations land from the new instrument's medians
  with rule 16's estimator honestly stated, derived constants re-derive by the same formula over new-era inputs.
  **The loophole guard is a reader and not a paragraph:** an era exists only via a dated §5/§7 amendment plus a
  both-arms transition archive, both recorded in `scripts/measurement-eras.tsv`; a current era citing no amendment
  resolves to *nothing* and gate-p5 renders `FAIL`; and resolution deliberately does **not** skip a malformed row
  to reach a valid one, because falling back is precisely the misattribution the clause forbids. `free-placement`,
  the era before eras, is named retroactively and left undated — it was not a concept while it ran. Consequence
  for this tree: the registry is empty for the whole fleet rather than for one host, and the transition run's
  green is the first green under `pinned8`.
- **21 mutants driven against the new readers, 21 killed, and a 22nd deleted for being unkillable.** Three
  survived the first pass and every one was a **blind fixture rather than redundant code**: an empty-era guard
  with no era-less registry row to match, a width check with no row one column short, and an empty-key guard with
  no empty-keyed witness row. Each got the row that makes the guard matter. The fourth survivor was genuine — an
  unnamed-era check whose deletion changed nothing observable, since an empty name prints an empty line and every
  caller reads that as no era — so it was deleted rather than explained, and the fixture asserting the outcome
  stands. One fixture case was also **vacuous on the first pass**: the rename case re-asserted a neighbour under
  a new label, since the hostname is not an argument at all; it now moves the host column and shows the answer
  does not move. Fixtures 22 → 50. Mutation is a session act with no standing harness, and gate-p5's pass line
  says so inside the number (§5 rule 12).
- **Apparatus ledger for this commit, both lines** (the standing clause, ruled recorded-owed-parked): **+236 net
  `*.sh` lines against zero library lines**, ratio **1.49x → 1.52x**. A third figure the counter cannot see:
  **114 lines of tracked `.tsv`** (the era ledger, the witness index, the registry's new header) which
  `gate-docs.sh` does not count, because its shell term is `*.sh`. So the apparatus grew by ~350 lines and the
  report shows 236 — disclosed as a share rather than left to be diffed, and whether the counter should widen is
  Scott's call, not a fix to slip in beside the thing being measured. The paydown lift is owed post-tag.
- **The apparatus-ratio report could not see a new script until it was committed, so it understated the cost of
  the commit being prepared.** `gate-docs.sh` counted *tracked* `*.sh`, so `scripts/baseline-test.sh` (131 lines)
  was invisible while it was still untracked. `820eac0`'s message therefore published **1.47x**; the correct
  figure for that commit is **1.45x → 1.49x** (shell 12693 → 13062, library 8778 unmoved). Both readings came
  from the same instrument minutes apart and the delta is exactly the new file's line count, so the arithmetic
  reconciles — but the metric that polices new shell was blind to new shell at the only moment consulting it
  could change a decision. That is the gameable-denominator hazard already recorded here, running in the other
  direction: not a denominator that absorbs the cost, a numerator not yet told about it. **Fixed
  rather than disclosed**: both terms now count `git ls-files -co --exclude-standard`, so the reporter sees a
  script the moment it exists. Both sides gained the flag, because correcting only the shell term would have
  been a redefinition letting untracked Go pay the ratio down. Proved by making the quantity move — a 7-line
  untracked probe raised `shell` by exactly 7 and removing it restored the reading, since a constant that is
  merely readable certifies nothing.
- **31 citations in this commit named `#6` only after `gh issue view` refuted `#33`/`#34`/`#36`, which were task
  ids.** Same failure mode as `ddd642f` (2026-08-18, "17 citations pointed at the wrong issue"), against the
  same number `#33`, three days later — so the recorded lesson did not prevent the recurrence. It cannot be linted away
  either: task ids and this repo's issue numbers occupy the same low integers and are syntactically identical,
  so no local check can discriminate them, and `#33` resolves to a real open-shaped issue with a plausible
  subject. The discriminator is a network query or a human reading, which is why the rule is *never carry the
  number into prose* rather than *check it later*. Caught before commit; the blind substitution was refused
  because seven tracked files cite the genuine `#33` (the coretype-sweep defect), so only lines this diff added
  were rewritten.
- **A criterion may not judge a host its reference artifact predates: the BASELINE-REGISTERED class**
  (#6, ruled 2026-08-21; DESIGN.md §5 rule 17, `docs/rulings.md` rule 17). gate-p5 convicted `keel-skx` for
  publishing no README row — a row that is *born* from a judged run, so the host's first judged run could not
  have had one, and the criterion was reading its admission date. Now three states, decided from the archive
  and never from a flag: no registry row and no prior archived judged log is newness (`BASELINE`, a fifth
  verdict colour that does not raise `FAIL`); no row **with** a prior log is an unmet registration (`FAIL`);
  a row is judged at `baseline − 2.6`, the same margin `CEIL_FRACTION` uses, derivation printed. The
  exemption closes structurally rather than by vigilance — the run that renders `BASELINE` creates the prior
  log that forbids it next time — and the consequence is immediate: this machine's archive already holds two
  judged skx runs, so **skx renders `FAIL` today, not `BASELINE`**, its exemption having been spent by the
  runs that found the problem. The gate emits a fully formed candidate row to
  `build/baseline-candidates-<rev>.tsv` and never writes `scripts/host-baselines.tsv`; the registry ships with
  **zero data rows**, because skx's share baseline cannot be imported from `5ec5fea` or `33de3b2` — those are
  single draws under the instrument the pinning-transition campaign replaces, and a published reference is an estimator, never a draw
  (§5 rule 16). Two limitations stated inside the number: the prior-log witness is per **operator machine**,
  not per repository (`build/` is gitignored, and the widening action — a tracked judged-run index — is
  named and filed as #114); and the unreadable-CPU branch resolves to `UNMEASURED` fail-closed and is
  unexercised.
- **The scaling aggregate's denominator now excludes BASELINE hosts, and two of its sentences were wrong**
  (#6, 2026-08-21). `fleet_coverage` passes only when `nclear` equals its denominator, so leaving a
  green-compatible host in `NHOSTS` would have resolved every such fleet to `partial` and blocked green
  silently, by arithmetic, one function from the branch that renders the verdict. Rendering the six fleet
  shapes rather than reading them — the practice this file's own comment records — caught two further
  defects the change introduced: a healthy fleet's headline PASS carried `0 of 3 rendered BASELINE`, noise
  that invites a reader to think the class fired; and an all-new fleet printed "no host produced a judgeable
  set of scaling ratios … 0 produced no complete set of ratios", of which the first clause is false about the
  mechanism and the second contradicts it. All-BASELINE is now its own sentence, and a zero denominator
  resolves to `UNMEASURED` — a fleet on which no host has a bar has measured nothing judged.
- **Both delegated verdict tallies gained a BASELINE column on the day the vocabulary did** (2026-08-21).
  `gate-p4.sh` over gate-p3's log and `gate-p5.sh` over gate-p4's each counted three columns where the
  vocabulary now has four. Neither can see a `BASELINE` yet — gate-p5 alone emits them — which is exactly the
  condition under which a missing column is invisible, and the reason to widen on the day the helper lands
  rather than on the day a delegate first uses one. The stale `gate-p5.sh:987` cross-citation in that comment
  was repointed at the line it names.
- **`scripts/baseline-test.sh`: 22 fixtures, five mutants driven** (#6, 2026-08-21). The registry ships empty,
  so a healthy run cannot reach the `registered` branch at all and a green gate would say nothing about the
  code that will set a per-host bar. Each of five mutations — a dropped `NF >= 7` guard, a dropped header
  skip, a dropped self-citation guard, a lost host-name anchor, a truncating append — was caught by exactly
  the one case aimed at it. Wired into both `make lint` and gate-p5, deliberately: lint runs on every push
  and catches a broken reader before a $24/hr fleet renders a bar from it, and the gate's copy is what makes
  the published log self-certifying.
- **§5 rule 16: a published reference is an estimator, never a draw** (ruling on #6, 2026-08-21; `DESIGN.md` §5,
  `docs/rulings.md` rule 16). `gate-p5` criterion 9 convicted two `keel-zen4` README rows on `33de3b2`, and both
  published values were the **maximum** of their six-run history on the same physical instance (`Ssymm/8T`
  610.8…**654.3**, `Strsm/1T` 35.66…**37.61**), so a 5% band was spent on the reference's own bias — and neither
  disagreement resolves at the intervals the two runs actually measured (`Ssymm/8T` |diff| 38.90 against 45.21 of
  half-widths; `Strsm/1T` 1.95 against 4.00). Single-draw publication makes the check measure the reference draw's
  *altitude within its own spread* rather than the code, with a sign set by luck: "a high draw manufactures future
  reds exactly as a low draw would manufacture future flattery." The ratified repair is **median over the archived
  runs, each row stating its estimator** — a repair and not an amendment, because the band is untouched at 5% and
  the criterion's standard was always "today within 5% of what this host does". The **interval-aware variant was
  drafted, computed and refused** on direction: all four rows in question tie at their archived intervals, so it
  would have retired two reds and convicted nothing, and a correction that only acquits on the data in hand is a
  loosening wearing rigour's vocabulary. Its first computation was itself wrong — it assumed the reference carried
  *this* run's tighter interval — which is why the recomputation is what refused it. The 7.1% peak-to-peak spread
  stays **unmeasured** as to noise-versus-code-change: the archive's one same-revision repeat spread 0.71%, which
  does not decompose it, and the ruling rests on the estimator argument alone.
- **§5 rule 5: placement is pinned, fleet-wide and never selectively; §7 rule 7 gains placement and estimator to
  what a reported number must state** (ruling on #6, 2026-08-21; `DESIGN.md` §5/§7, `docs/hosts.md`). Every judged
  invocation runs under an affinity mask of eight distinct physical cores in one NUMA node, and the ceiling arm
  under the identical one — a share whose numerator and denominator came from different placement methodologies is
  not a share. Four independent readings that the free instrument was reporting the draw: `keel-zen4`'s `Strsm`
  verdict red at `5ec5fea` and green at `33de3b2` on unchanged code; `keel-zen5`'s `Ssyrk` clearing by 0.4 points a
  bar its derivation set 2.6 below every healthy row, at ±5.0% where the ladder read ±0.90%; `keel-skx`'s `Strsm`
  clearing 7.0 at its median and failing net of CI; and criterion 9's band narrower than the spread it judges. A
  red that turns green under a tighter estimator of the same quantity is supersession working, so it is disclosed
  with both readings side by side rather than avoided — the transition campaign runs **both** arms and archives
  both. Stated as a falsifiable prediction (§5 rule 15): the pinned arm's intervals must narrow materially, or the
  adoption is refuted by its own transition run and reverts. Two limitations inside the number (§5 rule 12): the
  mask pins `threads=1` rows to a *node* and not a core, so the ±0.11% a one-core probe read for skx's 1T `Sgemm`
  (against ±14.6% unpinned, at `-count=20` versus the gate's 10) is not what it promises; and Go reports the mask's
  width as `GOMAXPROCS`, so rows carry `-8` where the free arm carried `-192` or `-72`.
- **§5 rule 12 gains clause (c): a hole no future action can close goes inside the number and is never filed as a
  debt** (ruling on #6, 2026-08-21; `DESIGN.md` §5, `docs/rulings.md` rule 12). Scott had filed the archive's
  inability to resolve a `gnr`-class row — true share would have to reach **153.9%**, i.e. never — as a post-tag
  refinement for #113's re-measured ceiling row, which is a *forward-looking* instrument pointed at twelve finished
  runs. His own correction is the rule: "a permanently unfixable limitation filed as a debt is a lie about the
  future — it goes inside the number instead," because a debt entry promises eventual payment and an unpayable one
  reads as *known, scheduled* where the truth is *known, permanent*. The operational test is to **name the future
  action that would remove the limitation**; a forward-looking instrument aimed at completed runs is the tell that
  there isn't one. Two things the clause leaves alone: the debt correctly scoped to what a future action *can*
  reach (`gnr` and `spr` carrying ceilings dated to `651d1bd` with nothing re-measuring them, #113's row
  conditional on #111's readmission ruling), and the enforced bar, which divides by each host's measured 8-thread
  ceiling — `k` appears only in the retrospective proxy, so 58.5% is vacuous for no host it judges. Recorded with
  the session's mirror-image error beside it: the loose `k` was attributed to `zen4` because zen4's name sat next
  to the 35 resolving rows, when zen4 owns them *because* its proxy is the tightest of the three (0.765 against
  gnr's 0.380). Adjacency is not attribution.
- **The ceiling's 8-thread form is RATIFIED as the amendment: the deviation was the ruling's own law applied to
  itself** (ruling on #6 Q1, 2026-08-21; `DESIGN.md` §4/P5). The literal `8 × 1T` denominator is *"measured
  denominators, never formulas"* violated by the sentence that states it — arithmetic blind to the all-core
  frequency, shared-cache and memory-controller effects only an 8-thread run reveals. Its consequence is
  disqualifying rather than merely loose: Granite Rapids reads **32.8–33.2%** of its ceiling under the formula
  and **86.3–87.4%** under the measured form, and that 33% *is* the front-end deficit #104 already owns, judged
  by P2's derived-ceiling criterion. Under the formula it would leak into the *scaling* verdict too, double-counting
  one cause across two criteria (§5 rule 6). The **54-point swing is recorded as the amendment's grounds**, because
  a move that large is exactly what a thumb on the scale looks like; it inverts the ordering the bullet's own
  falsifier depended on, and that is a diagnosis of the formula, not of the refinement.
- **`CEIL_FRACTION = 58.5`, typed as a REGRESSION BAR on `STRSM_FLOOR`'s precedent; ≥90% is REFUSED** (ruling on
  #6 Q2, 2026-08-21; `scripts/gate-p5.sh`, `scripts/readme-numbers.sh`, `DESIGN.md` §4/P5). All nine judged rows
  sit below 90%, and a fraction no observation reaches is an aspiration rather than a floor — pre-registration
  protects a *standard* from post-hoc tuning, it does not immunise a *model* from nine-of-nine contrary readings.
  A blocked GEMM at the gate's 4096³ carries pack, sync and imbalance costs an embarrassingly-parallel compute
  ceiling does not model, so the rows are reporting the real overhead band. Derived from the lowest judged row of
  `build/gate-p5-651d1bd.log` — `keel-zen5` `Ssyrk` at **61.1%**, already net of CI since the numerator is
  `bench_gflops_lo` — less **2.6 points** of margin, with the derivation printed on every run. Derived from one
  run and **enforced on later ones**, so it can genuinely fail, and **invariant to Q3's judged-set change**:
  `gnr`'s rows sit 25 points above the minimum, so dropping them leaves 61.1% where it stood.
- **Re-adjudicating the archive under the ratified bar resolves 35 of 105 rows, and one host's resolving power is
  structurally zero** (#6, 2026-08-21; supersedes the 2026-08-20 *"resolves none of them"* entry below, which was
  written against the refused ≥90%). The population is now defined **structurally** — every archived `gate-p5` log
  carrying no measured-ceiling row, which is **12 files**, not the eleven that clause claimed — so it does not go
  false with each new run: 105 judged rows, **23.8%** to **74.9%**, of which **35 definitively cleared their own
  ceilings** and 70 stay unresolved. Since `published_share = k × true_share` with `k` = ceiling/(8×1T) per host,
  the archive can resolve a `zen4`-class row that truly reached **76.5%**, a `zen5`-class row at **85.3%** and a
  `gnr`-class row at **153.9%** — that is, never. All 35 are Zen 4 rows for precisely that reason: the
  concentration measures how loose the proxy denominator is per µarch and is **not** a ranking of the hosts
  (§5 rule 12 wants that stated inside the number, not beside it).
- **`gnr` drops to CHARACTERIZATION and the signing fleet's Intel arm comes from wave 2** (ruling on #6 Q3,
  2026-08-21; `scripts/aws-fleet.sh`, `docs/hosts.md`). Tagging v0.1.0 with a disclosed red is out — CONTRIBUTING's
  tag condition is green gates — and unparking #111 is out under the freeze's own test, because the feed-bound
  class needs its own derived-OpenBLAS-ratio legislation and *the certificate does not need this host*. What it
  needs is an Intel AVX-512 arm judged under machinery that already exists, and **SKX and ICX are issue-bound
  silicon — janus's class — whose roofline exception has been law since P2**. The candidates launch at their
  family's largest non-metal size — **c5n.18xlarge (36 cores) and c6i.32xlarge (64 cores)**, each equal to its
  metal sibling's core count — because `remote.sh` classifies anything smaller `correctness`, so one read-back both
  classifies the µarch and justifies the `KEEL_EVIDENTIARY_SIZES` addition that lets it be judged at all; a
  read-back at the 8-core exploration size would have certified nothing. `aws-fleet.sh`'s `FLEET` gains the two
  roles at the exploration size and is relabelled as what it is — a launcher list, not an admission list, since the
  first draft of this entry called it the signing fleet at 2xlarge and would have booted a fleet no gate may judge.
  Whichever lands in the issue-bound class joins the judged set, both if both do. `gnr` and `spr` sit beside each
  other as labelled characterization rows on antares's consumer-row precedent. The cost is stated rather than
  absorbed: `gnr`'s scaling rows are **no longer re-measured by any gate**, they are dated to `651d1bd`, and being
  non-citable is what that fact earns them. Keeping it in-fleet as measured-but-unjudged is **not available in the
  instrument** — `$SENTINELS` restricts criterion 5's verdict only, while #111 lives in gate-p3's `OB_*` criterion,
  which judges every host in `.keel-hosts` with no such branch. Building one is filed, not done here.
- **The published block is re-measured on the judged fleet under the derived ceiling, and the caption now carries
  the criterion's own readings** (#6, 2026-08-20; `build/gate-p5-651d1bd.log`, 24 rows from one run on
  c7a.48xlarge / c8a.48xlarge / c8i.96xlarge). Measured 8-thread ceilings and what the judged routines reached of
  them: **Zen 4 713.6 GFLOP/s, 82.5-90.0%**; **Zen 5 1568.5 GFLOP/s, 61.1-65.3%**; **Granite Rapids 742.2
  GFLOP/s, 86.3-87.4%**. Those ceilings are **76%, 68% and 38%** of 8x each host's own 1-thread peak — a
  different factor per host, which is the retired floor's defect in measured form rather than as the rank
  inversion that motivated the ruling. The caption had been publishing the 8x-1T share it calls "not a score"
  while withholding every share the gate judges by.
- **P5's 6.0x cross-host scaling floor is RETIRED; the judged three are compared to a ceiling measured on
  each host** (ruling on #6, 2026-08-20; `DESIGN.md` §4/P5, `CEIL_FRACTION` in `scripts/gate-p5.sh`). The
  falsifier is a rank inversion, not a miscalibration: at `ce43bca` the floor refused Zen 4 holding **65.9%**
  of 8x its own core peak and passed Granite Rapids holding **34.3%**, monotone across all three hosts, and no
  AMD host has cleared it in any log here. A fixed T8/T1 ratio rewards a *bad single-thread baseline* — the
  host whose one thread already saturates its memory system has no ratio headroom left. The bar becomes
  `min(8-thread measured compute, measured bandwidth bound)` from `BenchmarkCeiling`, judged
  achieved-against-own-ceiling with the derivation printed. **`CEIL_FRACTION` is empty**, deferred to the
  fleet measurement on #37's ratified precedent: the fraction is computed, printed and reported, and no host
  fails on it. The memory term is not yet in the `min()` and the omission is strict — `min(c,b) <= c`, so the
  printed fraction is a lower bound and cannot pass a host the full ceiling would fail — but the bandwidth
  rows are measured every run anyway, because a term that can only *rescue* a host must not be the unmeasured
  one. `scripts/readme-numbers.sh` now refuses to regenerate the published block from a pre-ruling log rather
  than republishing it under a bar that no longer exists; `docs/gates.md`'s verbatim P5 lift is annotated, not
  edited, since an archive that gets quietly corrected is neither.

- **Running `BenchmarkCeiling` for the first time refuted two sentences written beside it** (#6, 2026-08-20;
  §5 rule 11). (1) The read-only `dot` probe was called the unambiguous half of the bandwidth bracket, and on
  the dev host's scalar path it read **32.7 GB/s at one thread against `axpy`'s 44.6** — the read-only probe
  slower than the read-modify-write one, which no memory system does. `Sdot`'s own throughput was the limit,
  not memory: a probe below the memory bound measures the probe. At 8 threads they converge (190.6 vs 193.0),
  which is what both being memory-limited looks like, so the arm is a bandwidth reading only where its
  8-thread figure sits near `axpy`'s. (2) `gate-p5` called the 8-thread compute shortfall "the clock droop
  with core count"; the dev host read **53.5% of 8x its 1-thread rate**, far past any licence-level clock
  change, because 8 threads there land on a mix of performance and efficiency cores. Core heterogeneity, SMT
  siblings and shared-cache pressure all land in that one number and the gate separates none of them, so it
  now reports the shortfall and attributes nothing (§5 rule 6).
- **Re-adjudicating the historical scaling verdicts under the derived form resolves none of them, and that is
  the finding** (#6, 2026-08-20). Neither half of the ceiling was ever measured — `grep` across all eleven
  archived `gate-p5` logs finds no 8-thread compute row, because `BenchmarkPeak` has only ever had a 1-thread
  arm — so the only recoverable denominator is `8 x 1T`, which the clock droop makes an *upper* bound on the
  true 8-thread ceiling. Every archived share is therefore a **lower bound** on achieved-against-own-ceiling,
  which resolves in one direction only: at-or-above the fraction is a definitive clear, below it is
  **unresolved and never a retroactive failure**. Recomputed from the logs: 105 judged rows spanning
  **23.8%** (`janus` `Ssyrk`, `175098d`) to **74.9%** (`vesta` `Ssyrk`, `117b78f`), **none** at 90%. This
  corrects a claim committed to `DESIGN.md` §4/P5 hours earlier in the same session, that the compute half was
  recoverable for every archived run; it is not, and the check that refuted it was a `grep` this session
  should have run before writing the sentence.

- **`scripts/aws-fleet.sh` launches through `spawn` instead of raw `aws ec2 run-instances`, and the fleet is selected by
  the launcher's own name.** Scott's directive, 2026-08-19: *"instances via truffle/spawn under `AWS_PROFILE=aws`
  exclusively."* Three things this script guarded now belong to the launcher and are better there — the dead-man switch is
  `--ttl` enforced by spawn's reaper rather than a `shutdown -h` baked into userdata that depended on the guest's own init
  working, the key pair and security group are spawn's, and `--wait-for-ssh` replaces a poll loop. The launched **name**
  (positional — `--name` exists but cobra wants `spawn launch <name>`) equals the ssh
  alias by construction, because that string is the key `spawn_probe` joins a provenance line on: a fleet this script can
  find is exactly a fleet admission can vouch for. `spawn list` reports **no** `tags` field, so the `Project=keel` tag
  selection used by `up`'s guard, `status` and `down` is re-keyed to the `keel-` name prefix, which preserves the property
  the tag was for (not a list this script wrote, so `down` still works after a lost `.keel-hosts`). New knobs:
  `KEEL_FLEET_TTL` (default `8h`), `KEEL_FLEET_DRYRUN=1` (appends `--estimate-only`, so the invocation that spends is
  validated *as the shipped command* and not as a hand-typed mirror of it — flag validation only: spawn's rate table has
  no `32xlarge`/`96xlarge` key, so those fall to its xlarge default and read **32x / 90x low**, spawn#543; `48xlarge` and
  `24xlarge` are present and land within 7%, so the error is per-size and always in the understating direction),
  `KEEL_SSH_CONFIG` (so the block writer can be
  driven against a throwaway file rather than the operator's real config — that step is the one that failed once *after*
  three instances were already billing). `KEEL_FLEET_MARKET`'s default flips to `on-demand` now that the judged tier is
  the normal case, and `ondemand` is an accepted alias because that is the spelling `spawn_probe` writes into a provenance
  line. One `aws` call survives, an SSM parameter read for the Ubuntu 24.04 AMI: not an instance operation, and pinned
  rather than taking spawn's AL2023 default because `provision-openblas.sh`'s package maps do not cover `amzn` and would
  reach `unrecognized distro id` after the fleet was billing — changing the OS also changes which OpenBLAS build every
  published ratio is measured against. Two defects found before any spend: **every `ssh` in the verification loop takes
  `-n`**, because `ssh` reads the loop's stdin, which is the herestring holding the remaining hosts — measured at 1 host
  visited of 3, and the loop *succeeded*, handing a silently partial verification to a judged run; and `--region` is
  passed to `launch`, since an AMI id is region-scoped and spawn's own default region would have failed on an
  invalid-AMI error naming neither variable. The name goes **positionally** (`spawn launch <name>`); `--name` also exists
  and its help says "required", which is how this was first written and what the launcher rejected. Apparatus ledger:
  **+21 net `scripts/` lines against 0 library lines**, so the prediction that this rewrite would pay back the previous
  commit's +286 is **refuted** — it deleted 155 lines and added 176, the deletions code and the additions mostly the
  comments justifying the delegation, which is the "prefer deleting a line to explaining one" rule failing in the
  direction the rule exists to catch. Correction to `1ff4130`'s message: the baseline it published as "shell 12322" was a
  mid-work worktree reading; `git ls-files`-counted `*.sh` was **12007** at `HEAD~1` and **12293** after that commit. The
  ratio it printed, 1.40×, is right at either number, but a budget figure is the thing under review here and a stale
  numerator is not available as a rounding detail.
- **§5 rule 15: a conservativeness claim about an instrument is a testable claim, so direction-of-error is a
  measurement** (#110, `docs/rulings.md` rule 15). Scott's ruling on the second defect, the one in the writing rather
  than the arithmetic: *"'safe direction' asserted from reasoning, inverted by the instrument's actual behavior,
  published without being run against the thing it described."* Why this is the worst place to skip §5 rule 11 and not
  the most forgivable — conservativeness is self-recommending, so a bound believed pessimistic is never asked for
  evidence, and if the sign is backwards the word "conservative" is exactly what stops the next reader looking. The two
  prose sites the previous entry left open are corrected as *substantive*, dated, with the original visible: `DESIGN.md`
  §4 and `scripts/gate-p3.sh`'s instrument-exercise header both asserted a **measured** interval was `zero-width`,
  reading width 0 off a reported `± 0%` that means "narrower than 0.5%". **The archive refutes the categorical form on
  the same host and the same comparison** — janus reads `[1.014x, 1.034x]` around 1.026 in one archived run, non-zero
  width, from a run whose CIs did not happen to round to `0%`. The conclusion survives *with a denominator it never had*:
  1.10 − 1.034 = **0.066** of margin against ~**0.010** of quantization width, about six quanta, so
  `KEEL_INSTRUMENT_WIDEN_CI` is still needed to reach the three-state renderings — for a measured reason instead of an
  impossibility one. `docs/spill-report.md`'s `[1.836x, 1.836x]` loses the word and keeps the number: **0.736** of margin
  against ~0.02, thirty-odd quanta clear. Five *other* `zero-width` sites (`roofline.sh` ×4, `roofline-test.sh`) are
  deliberately untouched — they describe a fixture given **no bounds**, which the input format *defines* as zero-width,
  so they are constructions and not readings, and correcting them would assert something false about a definition. One
  word, two meanings, one of them a measurement. Also recorded: antares's `[1.077x, 1.100x]` sits flush with the 1.10
  bar, well inside one quantum, and its class does not move only because the collapse rule added 2026-08-16 for an
  unrelated reason yields `fma-bound` on *both* branches — a rule written to stop an `UNMEASURED` the data settles is
  what kept this defect off that verdict, which was luck in the precise sense that nobody had checked.
- **The archive re-read finds three moved verdicts, not one, and two were invisible to the first pattern** (#110).
  `benchci -bandtop` is the only instrument that can read a run whose samples were destroyed, which is every run to
  date; it states each rounded row as the interval its rounding supports and re-adjudicates at the pessimistic edge.
  Over all **16** archived gate-p5 logs, **192 rows**, **3 verdicts move** — janus `Strsm` `7.0101 PASS → 6.9404 FAIL`
  (the flip on camera), plus vesta and antares `Ssymm` at `6.0170 → 5.9562` and `6.0307 → 5.9703`, both against the 6.0
  bar, both on `boost off` runs. The two new ones were found by the tool *refusing* ten of the sixteen logs rather than
  by review: the first pattern required `ROUTINE: 1 thread …` and those ten carry a `boost off — ` annotation there,
  which names a different measurement condition and is now captured and printed rather than skipped. Fail-closed on a
  zero row count earned its place — a pattern narrower than its input greens exactly like a clean parse when the only
  report is a count. Every band-top line prints "only ever toward FAIL" with the moved count beside it, so §5 rule 15's
  own sign claim is checkable on each run instead of asserted once: widening an interval cannot raise a floor net of CI.
  Band-top is for history **only**, per the ruling — forward runs have samples.
- **This session's apparatus spending, and the trap in the number that reports it** (#110). Net `scripts/` **+68**,
  `tools/` **+494**, library **±0** — the cap is violated, ruling-mandated, with the offsetting lift owed and named here
  rather than argued away. The instructive part is that `gate-docs.sh`'s two ratio lines move in **opposite directions**
  on this one change: the historical line reads 1.44× → **1.37×**, an apparent 0.07 *improvement*, because its
  denominator is all tracked non-test Go and it therefore absorbs the new instrument as though it were library. The
  apparatus line, whose denominator excludes `tools/`, reads 1.80× → **1.88×** with that denominator **identical at 7217
  on both sides** — which is what makes the comparison clean and the zero-library-lines claim exact. A session can
  improve the ratio it is capped by, by spending. That is the flattery the second line was added to expose, caught
  paying out.
- **T21's consequence is corrected: an integer-percent CI is lenient for a floor, not conservative** (#110). It read
  *"no shipped criterion is wrong because of this"*; with the CI read as 0 the check becomes median ≥ floor, which is
  **easier** to pass than median net of CI ≥ floor. The gate has since produced the verdict that sentence excluded —
  `janus` Strsm flipped FAIL → PASS between two runs on a **0.014%** move in the point estimate (7.0098 → 7.0101),
  because one arm's reported CI crossed 0.5% from `1%` to `0%` and with both arms at `0%` `bench_ratio_lo` returns the
  raw ratio, which is the degeneracy DESIGN.md's P4 clause exists to prevent. All 48 CI readings in the two logs are
  integer percents; `benchmath.Summary` holds float64 bounds and `benchtab.ToCSV` reuses the `%.0f%%` *display* string
  for the machine-readable column. One rounding step is worth 0.1386 on a 7.0 ratio against margins of 0.011 and 0.081.
  Left as a dated correction, not a rewrite: T21's observation and repro are right, its reasoning was published without
  being run against the instrument it described (§5 rule 11). §4's new escalation bullet no longer claims a `0%` reading
  was never undecidable — it is *more* likely undecidable, since the band can straddle the bar. Unfixed pending a
  decision: `bench_ratio_lo`, and the `zero-width`-interval justifications at `DESIGN.md:116` and `gate-p3.sh:30`, which
  are properties of the formatter and not of `janus`. *(All three are fixed as of the two entries above, in the same
  unreleased cycle; this sentence is kept because it is what the decision was requested against. The ruling arrived the
  same day: `tools/benchci` supplies the resolution, and the two prose sites are corrected substantively under §5 rule
  15.)*
- **README's 24 published rates are re-measured at `335ea9d`, and their caption is now generated with them** (#6).
  Ruled 2026-08-19: criterion 9 had already ordered the re-measure, because the three stale `Ssymm` rows disagreed with
  the shipped tree by 5.06–9.43% *on the gate's own denominator* — `(a−b)/b` with **this run's** value as base, not the
  published row, which understated the breach 3:1 and put all three outside `README_TOL` rather than one. The regenerated
  caption names 4 of 12 scaling ratios below the floor and splits them by cause: 3 sit below it outright, 1 clears the
  point estimate and misses only net of CI. Two prose claims that had decayed against the numbers beside them are gone:
  the floor was "missed on the two hosts that keep the most of their single-thread peak" (five rows, three hosts, and
  `antares` keeps the *least*), and a stale row "cannot survive a gate run" (it turns the gate red; it does not thereby
  cease to exist). One denominator defect was in the emitter's own first caption — "All 12 rows" over a 24-row table,
  since 12 counts ratios and 24 counts rows, and the conflation is older than the script. `scripts/docs-gen.sh` extracts
  the caption region onto `doc-site/numbers.md` and dies without it: rows published without their floor disclosure are
  not a thinner page, they are a flattering one. Both fail-closed branches were driven on purpose, not inferred.
- **`CONTRIBUTING.md` states when `[Unreleased]`'s session grouping collapses**: at the version cut and never before.
  Ruled 2026-08-19 after an attempt to merge `[Unreleased]`'s 21 session-grouped `### Added/Changed/Fixed` sections into
  four canonical ones churned 3128 lines and was reverted. Session groups are provenance for this project's ledger; a
  release section is the deliverable users read. Two formats, two readers, one scheduled conversion.
- **`solveRight`'s row loop moves to the outside; bit-identical, 4.90× locally** (#37). The strided nest re-walked B's
  live window once per `(j, p)` pair, so every scalar operation touched a different cache line and the rate sat at
  0.213–0.232 GFLOP/s across a 16× change in `MB` — flat, because nothing about it varied with the block. The sweep at
  `e8662ba` carries its own control: `solveLeft` and `solveRight` do *identical* flop counts on identical partitions and
  differed by 7.6–12.4× (0.224 against 2.20 GFLOP/s at `MB`=64 on vesta), same scalar arithmetic, the one structural
  difference being that solveLeft's inner loop was already unit-stride. Rows of X are independent, so hoisting the row
  loop reassociates nothing: `TestSolveRightInterchangeIsBitIdentical` holds the new nest against the old one verbatim
  over 6 shapes × 8 flag combinations by `math.Float32bits`, with no tolerance in it, and the test is shown to fail on a
  1-ulp reversal of the `p` loop. `BenchmarkSolveRightInterchange` runs both arms in one binary and one process, since
  the quantity is a ratio and a cross-build ratio would carry these hosts' layout noise. **Measured on three
  `evidentiary` hosts** (`build/trsm37-8441a18.log`, `evidentiary=3 correctness=0 unknown=0 of 3 configured`): the
  isolated solve is **4.21× / 4.97× / 5.77×** faster on Zen 4 / Skylake-X / Zen 5, and `Strsm` side=R at n=2048, MB=64,
  1 thread is **2.51× faster end-to-end** on vesta, its solve 2.94–3.07× (two estimators whose 4.5% spread *is* the
  T22 layout systematic, under T22's 7.04% Zen 4 bound). `solveRight`'s share of the call falls 90.7% → 77.4% for
  3.17% of the flops. `side=L` is byte-identical across the two commits and moves up to 5.1%, so this table's
  cross-commit floor is measured at ~5% and not the ~2% first asserted (§7 rule 7). What remains is the accumulator's
  serial dependency, which caps the scalar arm at one element per subtract latency and is what #37's vector arm
  addresses by widening across rows; its headroom is host-dependent, since post-interchange Skylake-X's two solves sit
  1.21× apart against Zen 4's 3.20×.

- **`DESIGN.md` §5 gains rule 14: a defect's severity is a function of its deployment context, not its code**, appended
  so no ordinal moved (#106). #106's "latent, not active" was true when written and false once #109 made the lab a
  signing tier, with no byte of the defect changed; re-admitting against the unrepaired classifier would have demoted
  all three lab hosts by a bug's signature instead of by evidence. Incident in `docs/rulings.md`, rule 14.
- **`DESIGN.md` §5 gains rule 13: two cost terms are comparable only at their rates**, appended so no ordinal moved
  (#37). A count is not a time, so ordering two terms by their counts predicts a direction only the rates can supply —
  rule 7's flop-share-is-not-a-time-share one level up. Clause (b) is Scott's: a constant whose optimum depends on
  another term's rate is tuned jointly with that term, **once**. `MB` therefore stays 64 and the interim 17–33% is
  priced on #37. Incident in `docs/rulings.md`, rule 13.
- **The `MB` sweep refutes the prediction that motivated it: smaller is faster today** (#37). At the shipped `MB`=64,
  n=2048, 1 thread, the diagonal solves take 53.2% of the time for 3.17% of the work (2.1 GFLOP/s, 1.3% of peak), and
  `MB`=32 is 17–33% faster on all three hosts. Both predicted *directions* held; the conclusion did not, because the
  countable term was weighted over the un-rate-checked one. `MB` must not be retuned apart from #37: at a 8× faster
  solve the best point moves to `MB`=128. Right side costs 3.3–5.9× the left at identical flops. Reported-class.
- **`DESIGN.md` §5 gains rules 10, 11 and 12**, appended so no ordinal moved (#36). Rule 10: cross-site agreement
  certifies propagation, never truth. Rule 11: an instrument's output overrules its author's claims about it. Rule 12: a
  coverage claim enumerates what it cannot see. 11 and 12 were fused for one commit; split so neither can be miscited.
- **`BenchmarkSymmNarrow`'s wide-n control holds at one thread only, and a flop share is not a time share** (#36). Every
  shipped kernel has NR=32 and the nest pads n out to it, so n=1 measured 35.9× the n=1024 row's per-column time; at
  GOMAXPROCS=8 every row moved, the control included. Both corrections are in the fixture's own comment.
- **`Ssymm` reads its symmetric operand in place rather than reflecting it into a dense square** (#36).
  `internal/pack.ASymPanels`/`BSymPanelsPart` split each run at the diagonal, dropping an O(d²) allocation (67 MB at
  n=4096; measured 16,904,645 → 127,365 B/op at n=2048) and a d² pass over A. Bit-for-bit identical, pinned by
  `TestSymPackMatchesExpansion`. Fixture: `BenchmarkSymmNarrow`.
- **`tools/shapegen -uarch NAME:WIDTH:PORTS:LATENCY` scores a sweep against any front end**, default unchanged at
  `skylake-x:4:2:4`. The SPR and arm64 constants are deliberately *not* listed in the source: taken on the command
  line, each re-sweep records them in its own log beside whoever sourced them instead of minting three integers.
- **Issue width is a load-bearing input to P2's "unreachable", not a background constant.** Re-scoring the same 140
  audited shapes at width 6 moves the frontier off the shipped 2×32 ×4 — which becomes dependency-bound at 32.00
  flops/cycle — onto 3×32 ×2 at 38.40, and the ceiling from 43% of the FMA peak to 60%. Predicted before the run.
- **The shape objective was missing its dependency term.** Ranking is now
  `cycles = max(I/W, F/P, (F/A)·L)`, and the measured answer corrects this project's own guess: the shipped 2×32 ×4 is
  *issue*-bound at 18.50 cycles, not latency-bound, so the corrected objective and the old insns/FMA one agree on the
  frontier. The amendment reorders only the low-accumulator corner (2×48 ×2 above 1×48 ×8, 24.77 against 24.00).
- **gate-p2's `SWEEP_BEST_IPF` is corrected 4.438 → 4.625, and stops being a trusted constant** (#107, ruled
  2026-08-18). 4.438 was attributed to Permute 2×64 ×2, which needs `MR·U == 16` to read its A panel the way the
  shipped kernels read theirs — 16 index vectors live against the 15 SIMD values go1.26.x allocates (T10) — so it
  named a kernel that cannot exist. `e1c6340`'s enumeration re-derives the best *emittable* zero-spill figure as
  4.625. The correction is **adverse**: 4.625 sits further from janus's required ≤3.88 than 4.438 did. Both gates now
  reconcile their own copy against `shapegen -frontier` on every run and fail on mismatch.
- **That repair loosens the amendment's stated worst case, opposite in sign to the shaping argument it settles.** The
  bound `0.90 × 2.25 ÷ (SWEEP_BEST_IPF × 1.05)` moves 43.5% → 41.7% of peak. No host's verdict changes and the floor
  the shipped 2×32 actually faces is unchanged at 43.8%, computed from its own audited `I_b`; only the cap the guard
  permits rose. 4.625 *is* that shape's figure, so criterion 5b now reads ratio 1.000 and its live content is drift
  off the frontier rather than distance from it — 4×32 at 6.250 is still refused.
- **"2×32 is latency-bound" was wrong and had been published upstream.** On SPR its chain floors the body at 16.00
  cycles where it measures 30.24 — 1.89× too loose to bind — and it uses 54.4% of its instruction supply against
  4×32's 94.9%, clock-free against the peak loop. Corrected in `docs/spill-report.md` §10.2, which argued the
  inversion without ever recording 2×32's 61.45 GFLOP/s against the 232.3 peak, and on golang/go#80829.
- **17 citations minted the 4.438 → 4.625 repair against `#33`, which is a live unrelated gate defect.** The number
  was a *task* id, and the task tool prints its ids in issue syntax, so it transcribes with no doubt-step. Repointed
  to #107; `52a69af`'s pushed commit message still carries the wrong one and cannot be. Recorded rather than quietly
  fixed because a citation landing on a real-but-different issue reads as well-formed.
- **CI never ran `gofmt`, so an unformatted file sat at HEAD through green runs** — `internal/spill/spill.go`'s var
  block lost its alignment when a comment split it. Added to the stock job in *gating* form, since `gofmt -l` exits 0
  whether or not it lists anything, and both branches driven on purpose before landing.
- **`layout-ensemble.sh` cleared a benchmark it never graded instead of demoting it.** `grade_pad` iterates the
  token→symbol map, so a row outside that map — reachable via `KEEL_L1_FILTER` — printed unlabelled and citable while
  the comment above the map claimed the opposite. Demoted in `grade_rows` now, geomean inheriting it.
- **`gate-docs.sh` prints an apparatus line beside the historical shell/library one**, moving `tools/*.go` across.
  An instrument counted on the library side would flatter the ratio it is counted by; both lines print so the
  published 1.6× series stays comparable.

- **The 115-shape generator behind the spill frontier is not in the tree and never was** (#107): only the audit half
  is, so the report's part 4 table cannot be regenerated and neither can `SWEEP_BEST_IPF=4.438`, which gate-p2
  criterion 5b reads.
  Recorded in the report; the rebuild's in-tree/out-of-tree question is Scott's.
- **`docs/spill-report.md` is reopened (part 10): P2 and P3 are both red on the first evidentiary host**, a full-size
  `c7i.48xlarge` — 34.2% of measured peak and a 51.0% mission ratio. 55% needs ≤ 4.09 insns/FMA against the shipped
  6.25, and golang/go#80829 plus golang/go#80830 together reach only 50.0%. The report's part 9 stands for the retired fleet.
- **gate-p3's mission ratio is now decided by the admission machinery, not merely taken on an admitted host**
  (#104/#30): `admission_readback` and `adm_judgeable` in `remote.sh` gate both of criterion 6's verdict paths,
  gate-p2's inline copy calls them, and a not-admitted host has its own tally so the aggregate stops calling it a
  host that produced no ratio. gate-p5's scaling floor is **not** wired yet and says so.
- **The evidentiary host class is full-size, not bare metal, and a correctness-class number is
  reported-not-judged whatever it reads** (#104, ruled 2026-08-17): retiring metal had left the class empty
  while the harness went on judging perf on any guest that answered — which is how `c7i.4xlarge`'s 34.2%
  became a P2 STOP. Class is read before the number is trusted, and an unreadable class is `unmeasured`.
- **The harness reads the class it is told to check** (#104): the provenance line carries `instance=` (IMDSv2;
  `none` for no EC2 identity, `?` for no way to ask), `host_admission` resolves it against a declared
  `KEEL_EVIDENTIARY_SIZES`, and gate-p2's criterion 5b reports rather than judges a non-admitted host. **No
  currently provisioned instance type is admitted**, so P2's floor is now `unmeasured` fleet-wide rather than
  missed. gate-p3's and gate-p5's judged perf criteria are the same shape and are **not yet** wired.

- **Every gate's provenance line now records cores, SMT width and sockets, not just `nproc`** (#82): read from
  `thread_siblings_list` (with `core_cpus_list` as the fallback), because `GOMAXPROCS=8` is 8 cores on a
  1-thread/core arm and 4 on a 2-thread/core one. Replaces gate-p5's private `lscpu` ssh — one round trip
  fewer, one fewer package assumed present, and the fact reaches every gate's archive rather than one. Whether
  P5 *requires* SMT off, or only that the state be recorded, is still open on #82.
- **The four phase gates' front matter moved to `docs/gates.md`, verbatim.** 855 comment lines standing above
  zero lines of code — `gate-p4.sh` ran 226 of them before its first statement — become one page plus a
  ~12-line pointer per script; every gate body below `set -euo pipefail` is byte-identical to its predecessor.
  **−809 lines in `scripts/`**, 1.61× → 1.50×. Not published: the prose was repo-only before the move.
- **A rule and a ruling are now different files.** `DESIGN.md` §5 rules 6–9 keep their operative clauses at
  their existing ordinals — no citation moved — and their incident histories move to `docs/rulings.md`,
  published as a record page rather than dropped off the site. §5 falls from 2,275 words to 1,530.

- **`gate-docs.sh` now prints `shell N / library M / ratio R` on every push, and a standing order in
  `CLAUDE.md` caps net additions to `scripts/`**: a session may not grow the apparatus unless it also lands
  a routine, a kernel, or a library fix. Reported, never a verdict — a red ratio would reward paying down
  shell instead of shipping. The instrument costs 20 shell lines and says so: 1.63× → **1.64×**.
- **DESIGN.md §7 is no longer titled "Claude Code kickoff prompt"** (H4). It opened *"Paste below into a
  fresh CC session in an empty repo… it does not assume this document is present"* — a prompt living inside
  the document it disclaimed, for a repo that is not empty. Framing deleted, eight rules kept verbatim at
  frozen ordinals: rules 2/4/7/8 are pinned and cited from ~40 sites, so renumbering is not a formatting
  choice. Verified by diffing all eight parsed rule leads against HEAD — byte-identical.
- **Two standing rules amended rather than obeyed literally** (2026-08-16 ruling: *retire any rules we do not
  really need*). The issue-per-discovery rule (`CONTRIBUTING.md`, `CLAUDE.md`) now says a discovery whose fix
  is smaller than its issue gets **fixed** and recorded — the word carrying the intent was always *silent*,
  never *small*. `docs/toolchain-notes.md` entries get the CHANGELOG's 1–3-line cap on prose, with the repro
  never abridged and the causal analysis in the issue; existing entries stay as dated records.
- **DESIGN.md §5 rule 5 now demands a stable clock rather than a `performance` governor** (amended
  2026-08-16, forced by the ruling to measure on AWS instances). No guest can satisfy the old wording: there
  is no `cpufreq` directory, so `remote.sh:410` resolves to `unmeasured` and blocks every gate — correctly,
  but it makes the pivot impossible rather than merely awkward. Where `cpufreq` is readable the governor
  assertion is unchanged; where it is absent, stability comes from `BenchmarkPeak` sampled at head, middle
  and tail. Same benchmark, no new bar. The harness half is not written yet — see the follow-up tasks.
- **Nine sites that stated rule 5 as "the performance governor" now state it as the rule reads**, each
  saying which instrument applies to the hosts it actually describes (`gate-p1/p2/p3/p4`, `bench.sh`,
  `provision-openblas.sh`, `docs/hosts.md`, and §4's own #31 ruling record). The two *printed* pass lines
  are deliberately left claiming the governor: they must name the instrument that ran, so they change with
  the harness, not before it.
- **Every GitHub Action in both workflows moved to a Node 24 major**, in one sweep and outside any feature
  work: `checkout@v4→v7`, `setup-go@v5→v7`, `setup-python@v5→v7`, `upload-artifact@v4→v7`,
  `configure-pages@v5→v6`, `upload-pages-artifact@v3→v5`, `deploy-pages@v4→v5`. Node 20 was deprecated on
  the runners (changelog 2025-09-19) and the old majors were **already being forced onto Node 24 with a
  warning annotation on every run** — read off the annotations of run `31986295744` rather than inferred,
  so the bump changes the declared runtime and not the one that was executing. Ruled 2026-08-16:
  *"deprecated runner images eventually become broken CI, and broken CI during the tag window would be the
  worst possible timing for infrastructure this project depends on for its certificate."* Both workflows
  together, because half a sweep is an inconsistency with no gate benefit.
  **Two things in the path were behavioural, and neither was taken on trust.** `setup-go` v6 sets
  `GOTOOLCHAIN=local` (actions/setup-go#460), so `go` can no longer silently download a newer toolchain to
  satisfy `go.mod` — it errors. That is the behaviour this project wants, since a silent substitution makes
  the Go version a fact about the runner instead of about the pin, and `go 1.26` in `go.mod` against
  `go-version: 1.26.x` needs no download; both jobs now print `go version` and `go env GOTOOLCHAIN`, on the
  same read-back-not-reasoning grounds as #88's dispatch markers. `upload-pages-artifact` v4 stopped
  including hidden files, which would publish a site missing a dotfile and say nothing; `find build/site
  -name '.*'` returns zero on a real build, so no `include-hidden-files` is set, and the pinned
  `mkdocs-material==9.6.22` is what makes that a stable fact rather than a lucky one. The deploy job's
  gating condition is byte-for-byte unchanged: this sweep publishes nothing.
- **`doc.go` rewritten for someone with a matrix to multiply, and `types.go`'s four flag types documented
  at all** (#92, ruled 2026-08-16: *"users will not care about the design notes. They want straight usable
  information"*). The test each paragraph now has to pass is **"would a person with a matrix to multiply
  still be reading?"** — so the package comment leads with the two build modes, the row-major/`ld`
  convention as a picture of an actual array, and a minimal `Sgemm` call, and it no longer contains the
  paragraphs about what took two phases to establish or why the Level-3 chain has two rungs. Nothing was
  deleted from the project's account of itself; the reasoning that had accumulated in godoc moved to
  comments that `go doc` does not render (`L1Chain`, `WorkersLastCall`), which is where a gate's grounds
  belong. `types.go` had **no doc comments whatsoever** on `Transpose`, `Uplo`, `Side`, `Diag` or their
  eight constants — the flags every call site passes were the least documented part of the API.
  **No performance number appears in godoc any more.** A doc comment is a contract and a rate is a
  measurement; the `# Numbers` section became one link to the site's numbers page, which is generated
  from the block gate-p5 re-measures. Read back rather than assumed: `go doc` rendered for all 38
  exported symbols, 38 leading with their own name (const groups checked for naming each member), and the
  read-back's matcher was self-tested against a non-matching lead first, because a checker that cannot
  say CHECK proves nothing when it says ok.
- **`DESIGN.md` §4/P2 now specifies the tile orientation that shipped, and states the naming convention
  in the same sentence** (#16, ruled 2026-08-16). The doc had said `MR=32, NR=6` — *"a pre-implementation
  fossil in BLIS's column-major orientation, written before the row-major decision propagated through the
  design"*, and **there is no intended column-major internal C**. It now reads MR=6, NR=32 with the
  convention inline (a tile `MR`×`NR` is MR rows × NR columns of row-major C, vectors along N), because
  stating the convention is what stops the question recurring: the discrepancy had been re-derived by a
  reader at least twice, and both times the doc gave an orientation instead of a rule. **The amendment
  falsified three sibling sites, found by sweeping on purpose rather than by anything reporting them:**
  `KERNEL.md` §2 called the reflection a departure that DESIGN.md had just ratified; `internal/spill/README.md`
  documented a spill-audit invocation in which **not one of four tokens was real** (`cmd/spillaudit`,
  `-fn`, `./internal/kern`, `Kernel32x6` — the command is `spill-audit -func`, the package is
  `./internal/vec`, and no kernel ever bore that name), so the documented command could not run and was
  corrected and then *executed* to confirm; `docs/spill-report.md` described DESIGN.md's planned tile and
  the spilling kernel as if they were different tiles. They are the same tile, now named consistently.
  One more correction landed in the amendment itself: the reflected 12-accumulator tile **is** `Kernel6x32`
  and **is not** comfortable for the allocator — 90 vector stack refs, 50 register copies, 5.62 insns per
  arithmetic op, 30.5% of measured peak on Zen 4 against 96.6% for `Kernel4x32` — so the doc's tile is a
  falsified prediction and now reads as one instead of as description.
- **All four verdict helpers now live in `scripts/remote.sh`, and no gate defines its own.** `remote.sh`
  warned in one breath that `VERDICT_STAMP` applies "in each gate's own pass/fail/info" and in the next
  that "an overridden copy is a copy and copies drift" — and the drift had already happened:
  `unmeasured` was shared and stamped, while `pass`/`fail`/`info` were copied into all six gates and
  **only gate-p3's copy applied the stamp**, because gate-p3 is where the exercise that needed it was
  written. So gate-p2's first synthetic run would have printed PASS lines indistinguishable from a real
  certificate — the exact forgery the stamp prevents, in the one place a banner does not help. The five
  identical copies were **not** corrected in place: uniformity across copies is not correctness, the
  five agreeing copies were the wrong ones and the odd one out was right, so a sixth good copy would
  have left the defect available to gate-p6. Verified output-neutral by running gate-p0 either side of
  the lift and diffing (identical), then verified load-bearing by arming the stamp and confirming zero
  unstamped verdict lines — a check that was impossible to pass before.
- **The collapse rule now reaches criterion 6: an undecidable Sgemm classification whose two candidate
  denominators agree yields a verdict, `why=agree-anyway`** (ruled 2026-08-16). #86 gave the gate a
  three-state grade and a rule for the class decision: `UNMEASURED` is for verdicts that *vary* over
  the uncertainty, so a doubt whose resolutions all lead to the same verdict is immaterial
  (`why=falsifiedanyway`, `why=samemixanyway`). Criterion 6 had not inherited it — an indeterminate
  class went straight to `UNMEASURED` even when both candidate denominators put the host on the same
  side of the 60% bar, which spends the scarcity three-state grading exists to protect. Now the log
  prints both candidates side by side, each graded **on its own net-of-CI bound and never the point
  estimate** (a collapse justified by a midpoint would be exactly the noise-driven verdict the
  amendment prevents), and agreement decides. **Symmetric**: two agreeing misses collapse to `FAIL`,
  so a slow host cannot shelter behind a class the run could not derive. A candidate with no bounded
  ratio supplies no agreement — assent by omission would let a missing measurement buy a pass — so
  that stays `UNMEASURED` with both branches printed. The decision is `p3_collapse` in
  `scripts/roofline.sh` with 7 fixtures, not an if-chain inline in a gate.
- **Criterion 6's fleet aggregate no longer reports a host that produced no measurement as a host that
  measured slow.** Found while rewording that aggregate for the collapse above, and it is the same
  label defect ruling #37 turned up in P5's scaling aggregate, pointing the other way: the miss count
  was **derived** as `nhosts - cleared - indeterminate`, so a fleet with one slow host and one host
  that never produced a ratio (no OpenBLAS reference, no benchstat interval) printed *"2 measured
  below it"* — a right verdict carrying a false claim about keel's speed. Worse, the `UNMEASURED`
  branch required `cleared + indeterminate == nhosts` exactly, so a fleet with a no-coverage host and
  **zero** misses fell through to `FAIL` and blamed speed for a hole. Every exit from the per-host
  loop now increments exactly one tally, the leftover is named as a leftover, and the aggregate is
  `p3_coverage` in `scripts/roofline.sh` with 6 more fixtures — extracted for `p3_collapse`'s stated
  reason, that an if-chain able to turn a `FAIL` into an `UNMEASURED` is not shown correct by a
  healthy run driving one of its branches. Confirmed discriminating rather than assumed: replayed
  against the old inline logic, the no-coverage fixture is `FAIL` there and `partial` here. The
  printed sentence itself is verified by reading, not by fixture, and the comment says so.
- **CI's `GOEXPERIMENT=simd` job now states the size of its own claim: dispatch is pinned to scalar and
  the read-back is asserted and published** (#88, ruled 2026-08-16). GitHub's runners have no AVX-512,
  so that job built keel with the simd toolchain and then ran **the scalar fallback** — while its name
  implied the vector kernels had run, and nothing in the log said which backend registered. #87's
  SIGILL was that gap biting. Both obvious fixes were rejected: asserting *"a vector backend
  registered"* fails spuriously on whatever the runner lottery deals, and merely documenting
  *"scalar-ish under a simd toolchain"* leaves the green's meaning varying run to run. Instead
  **determinism plus read-back** — the judged leg runs `KEEL_FORCE=scalar` and then asserts the
  markers agree (`keel-l1-active` = `scalar`, `keel-sgemm-active` = `*/scalar`), so the claim is
  identical every run *and* the pin is proven to have taken rather than assumed, which is
  `dispatch.go`'s no-silent-downgrade property checked from outside. A missing marker fails in its own
  right, as in the gates: "the tests passed" and "here is what they covered" are two different facts.
  The job summary spells out what is claimed (all paths compile; correctness verified on scalar,
  deterministically) and what is not (that any vector kernel executed) — vector-backend evidence
  stays on the three-host fleet. A second leg runs the suite under default dispatch with the selected
  backend **reported and not judged**: correctness still counts, but which backend produced it is a
  fact about the machine, and it is the one path by which a future AVX-512 runner would add real
  vector coverage and say so unprompted. Two verification details, both found by checking rather than
  assuming: `go test` shows a passing package's stdout **only under `-v`**, so the read-back comes
  from a probe whose `-run` pattern deliberately matches nothing (TestMain still speaks, exit 0)
  rather than from `-v` over tens of thousands of subtests; and `-count=1` is load-bearing, because
  the test cache **replays a previous run's recorded stdout** and would publish a read-back taken
  under a different environment. `set -o pipefail` is stated where a status crosses a pipe into
  `tee`, this repo having shipped that particular swallowed verdict three times in other forms. The
  assertion was driven through all four of its branches — correct scalar read-back, wrong Level-1
  backend, wrong Level-3 backend, markers absent — with the passing case as the positive control that
  proves it discriminates. **The read-back falsified a claim on its first run.** The job's own comment
  said the vector path "is not exercised here at all"; the opportunistic leg read back
  `keel-l1-active: avx2`. GitHub's runners lack AVX-512 but *have* AVX2, so Level 1 selects a vector
  backend while the Level-3 microkernels fall back — vector code has been running in CI all along, one
  level below where anyone was looking. The summary now reports each level separately rather than as
  one vector-or-not bit, since a true "this runner exercised a vector path" would have concealed which.
  That is the case for printing a read-back instead of reasoning about a dispatch: the reasoning here
  was wrong, cheaply, in public.
- **`Strsm`'s scaling floor is ratified at 7.0×, and the model the deferral asked for is recorded as
  *falsified* rather than restated** (#37/#89; `DESIGN.md` §4/P5 carries the grounds). `gate-p5.sh`
  deferred `STRSM_FLOOR` to "this measurement plus a stated model" for five phases, and the
  measurement arrived nine times over — three runs × three µarchs, `n=4096`, `threads=8`, boost off
  both arms, **7.403–7.668× net of CI**, recomputed from the logs rather than carried. The stated
  model is the work split the gate prints, `rank_update=0.98413 diag_solve=0.01587`; read
  `diag_solve` as an Amdahl serial fraction, as "state the model behind the number" asks, and at
  p=8 it implies a ceiling of **7.2001×** that **all nine readings clear** — the lowest by +2.82%.
  A ceiling the data walks through is a dead premise, not a premise needing adjustment, and **the
  favourable direction makes it no less a falsification**: §5 rule 7's objection to a check that
  cannot come out badly applies exactly as well to a bound that cannot come out binding. What the
  arithmetic says mechanically is a finding about the nest — the implied serial fractions
  (0.619–1.152%) sit *below* the 1.587% work share, which is the signature of diagonal solves that
  overlap the rank updates, because `Trsm` splits right-hand sides at the top (`MB=64 < MC=144`
  leaves the `ic` loop one iteration) and the solves ride that split. The replacement model is
  #65's per-`(jc, pc)` B-packing residue plus the makespan tail of the last claimed unit, measured
  at that 0.62–1.15%. **7.0× is a regression bar, not a derived threshold**: derived *from* the
  falsified ceiling it would be theatre, whereas set 0.403× (5.76%) below the lowest of nine
  observations it is margin — the same thing the ≥6× floor is. 7.4× was rejected as unshippable at
  0.04% of headroom. The gate keeps printing the work split, now saying in words that it is a work
  split and not a serial fraction, and reprints the ceiling it would imply beside the reading that
  clears it, **recomputed from each run's own declared split** so the comparison is about the
  current run; judged by nothing, since a future reading *below* that ceiling would refute nothing.
  One label defect fixed in passing: `HOST_CLEARED` is lowered by a miss against either bar, so
  typing the constant silently widened what the scaling aggregate covers while its sentence still
  named only the judged three — both aggregate lines now name both bars, because a pass that
  credits less than it verified is the same defect as one that credits more.
- **The host classification is graded in three states, so a class the measurement cannot decide
  reports `UNMEASURED` instead of picking a side** (#86; ruled: *"a verdict cannot be more certain
  than the least certain link in its derivation chain"*, and `DESIGN.md` §4/P3 records the grounds).
  The class is an *input* four criteria divide by, and on 2026-08-15 one noisy reading of it moved
  all four: janus's ceiling spread crossed `converge_max` (1.10), the convergence test returned
  `why=diverge`, the host reclassified fma-bound, and the gate then (a) failed the class-agreement
  check telling the operator to fix `internal/kern.HostClass`'s fingerprint, which was right;
  (b) faced the sentinel with the flat 55% instead of 90% of its roofline, failing at 42.7% of
  peak; (c) switched criterion 6b's denominator from `roofline 105.52` to `openblas 194.35`,
  turning **75.1% PASS into 39.5% FAIL**; and (d) followed with criterion 6's aggregate. Against a
  *fixed* denominator those same two readings differ by **1.3 points** (39.5% vs 40.8% of plain
  OpenBLAS, `build/gate-p3-under-p4-708ddbb-run1.log:131` and `-708ddbb.log:130`). The 35-point
  swing was the denominator. So the two comparisons that *select* the class — ceiling spread vs
  `converge_max`, and attainment vs 1.0, the falsification test — now consume `bench_ratio_lo`
  and `bench_ratio_hi` (#67's measured bounds, not a symmetry assumption) and grade three ways:
  clear on either side keeps its verdict, an interval spanning the bar yields class
  `indeterminate` and `RESULT=unmeasured`. `p3_denominator` gained a fourth exit
  (`classindeterminate`) so an undecided class cannot fall through to a confident `fma-bound`,
  and `p3_ratio_lo` fails closed on it. The *result* boundaries — 90% of roofline, the flat 55% —
  stay two-state with `DESIGN.md` §4's one archived re-run, because that allowance was priced for
  one verdict and a propagating input spends four. Re-measuring an `indeterminate` is therefore
  not the allowance being spent: there is no verdict to overturn.
- **A straddle whose two branches agree now decides the class anyway, instead of reporting
  `indeterminate` on a comparison it did not need** (#86, the converse edge of the same law).
  The amendment above withheld a class whenever the ceiling spread's interval crossed
  `converge_max` — including when *both* readings of that crossing led to the same class:
  converged means the ceiling exists and attainment above it falsifies it (fma-bound), while not
  converged means no ceiling was established (fma-bound). Neither branch reads `roof`, and the
  flat floor that applies is the same expression in both. Reported as `why=falsifiedanyway` /
  `why=samemixanyway` with the straddling interval still printed, so the record says *could not
  decide this, did not need to*. The collapse is tested over the **whole** interval —
  `roof_hi = pmax_hi / I` is the highest ceiling the reading admits and falsification must hold
  at every point of it — never at the midpoint, and it can only move a host from `indeterminate`
  to a class both branches already gave. Found by a real reading grazing the bar, not by review:
  on 2026-08-16 antares measured a spread interval of `[1.077, 1.100]` against the 1.10 bar while
  retiring at 162.2%–165.2% of the roofline that interval implies
  (`build/gate-p3-86-e53f7cc.log:79-80`). One rounding step higher and a host falsifying its claimed
  ceiling by 62% would have been reported "classification indeterminate". Three fixtures cover
  both directions of the collapse and the case that must stay undecided
  (`scripts/roofline-test.sh`, 43 fixtures).
- **`gate-p3` can be made to exercise its own three-state renderings, and such a run issues no
  verdict** (#86). `KEEL_INSTRUMENT_WIDEN_CI=<pct>` widens every ceiling-set mix's
  percent-of-peak interval to at least ±pct around its own point estimate before classification,
  leaving every shipped threshold untouched and printed. It exists because on this fleet the
  `indeterminate` renderings are otherwise unreachable — janus's class-selecting interval is
  zero-width, so no value of the 1.10 bar can sit inside it — and an unexercised branch in the
  instrument that issues the certificates is untested code. The synthetic input is therefore on
  the *measurement* side, reproducing the 2026-08-15 condition of a host reading noisily rather
  than a bar nobody would ship. Such a run prints a banner naming the override, stamps every
  verdict line `[synthetic]`, prints **neither `GREEN` nor `RED`**, and exits 2; its log belongs
  at `build/instrument-exercise-*` and never on a `gate-pN-<rev>` path (#78). `VERDICT_STAMP` is
  one shared variable read by `unmeasured` in `scripts/remote.sh` and by each gate's own
  `pass`/`fail`/`info`, rather than a per-gate override of those helpers, because an overridden
  copy is a copy and copies drift.
- **`gate-p3` prints the ceiling spread, the mix spread and the attainment interval it decided
  the class on** (#86). It read all three into `_`-prefixed discards, so when the classification
  flipped, the quantity that flipped it was in no log and had to be bounded backwards out of the
  threshold — which is how the first reconstruction of the incident above blamed the 90%
  attainment floor rather than the convergence test three fields earlier. §5 rule 8's "publish the
  underlying pair beside the derived figure", applied to a class instead of a percentage.
- **The performance-governor check is now one function, `assert_governor` in `scripts/remote.sh`,
  called from all five measuring gates** (#83; ruled: *"four better copies are the same failure
  mode with a better master, and the next labelling defect propagates just as cleanly — the fix
  for copy-drift is ending the copying"*). Four gates reported an **unreachable** host as
  `scaling_governor is unreadable`: the right verdict class with a false cause, because a host
  that answers always yields a `governor=` field, so empty probe output means the host never
  spoke. `gate-p5`'s copy was the divergent one and also the *correct* one — it established that
  a reading exists before parsing for one — so the lift's shape came from the outlier, not from
  the four that agreed. The helper exposes `GOV_STATE` (`performance` | `wrong` | `unreadable` |
  `unreachable`) and **always returns 0**: an exit code is an implicit verdict channel, and under
  `set -euo pipefail` a helper returning non-zero for "criterion not met" becomes the gate's own
  status in tail position and kills the run with *no verdict line at all* — an absent verdict, not
  a wrong one, indistinguishable from a kill (#76, #80). Enumerated before the lift: **five
  independent drifts across the ten copies** (two sites × five gates) — a `§5.4` citation <!-- citation-lint:quote --> naming a
  section `DESIGN.md` does not have (#85), `info "governor=${gov:-unknown}"` printing a reading for
  a host that answered nothing, "preamble checked it" vs "preamble read it", a `sudo tee`
  remediation hint present in one gate and absent in four (the lifted version prints the union),
  and `gate-p5`'s better parse. Agreement among four copies was never evidence about any of them.
- **Every measuring host must now be under the performance governor, asserted per host, in
  `gate-p1` and `gate-p2`** (#77; ruled before the green because a criterion that moves
  between runs for a naming reason is a defect in the instrument). Both gates read the
  governor and then asserted only that *some* host had cleared its bar under `performance` —
  a criterion any single machine satisfied on the others' behalf, which therefore said
  nothing about the others. It is not hypothetical: both archived green `gate-p1` logs have
  the Zen 5 host on `powersave`, and one of their rates is published at `CHANGELOG.md:2151`
  (#79). The fix is `gate-p4.sh:562`'s three-way, copied rather than reinvented so the two
  read alike: PASS per host, FAIL for a wrong governor, UNMEASURED for an unreadable one,
  in a preamble over the whole fleet — plus a silent re-read at the moment of measurement,
  because a governor that changed in between belongs to a machine somebody started using.
  `PERF_GOV_HOST` is deleted from both gates; nothing now records a precondition as a host
  name. Note that Scott's suggested mechanism (hostname normalization, `.local` drift, a
  DHCP-sensitive ordering) was not the mechanism — it was last-writer-wins over the hosts
  that *pass*, with a stable host list — and a canonical-name fix would have left the hole
  exactly as it is.
- **Seven single-witness gate aggregates became three-way over coverage state** (#73 tier C;
  `gate-p0` ×1, `-p1` ×2, `-p2` ×2, `-p3` ×2). An aggregate whose only state was "did any host clear
  this" collapsed two different facts into one FAIL: a fleet that looked and came back short,
  and a fleet with nothing to ask. `fail "no target ran the Sgemm sweep green with the avx512
  backend"` on three hosts that have no AVX-512 is an assertion about *keel* made from the
  absence of a measurement. Each now counts hosts that **produced** the reading beside hosts
  that **satisfied** it — `N_RATIO`/`N_CLEARED`, `N_JUDGED`, `N_FULLCAP`, `AVX512_SEEN`,
  `N_FORCED` — so zero readings reads UNMEASURED and readings that came back short read FAIL
  with the shortfall counted. The discriminators are witnesses the library already prints,
  verified from source rather than inferred by analogy: `keel-backends-available`
  (`internal/vec/vec_diff_test.go:82`), `keel-l1-available` (`l1_test.go:144`), and — where
  no availability marker exists — the *exercised* marker, which is runtime-filtered because
  `vectorKernels()` returns nil unless `vec.HasAVX512()` (`internal/kern/kern_amd64.go:34`).
  A first draft of the `gate-p2` counter used a `keel-kern-available` marker that does not
  exist; it would have made the count always zero, i.e. masked a real FAIL as UNMEASURED,
  which is the one substitution this taxonomy exists to prevent. The count is seven by this
  rule, stated so it is reproducible: added aggregate-level `unmeasured` branches in the diff
  — those with no `[$host]` or `[$name]` prefix — one per conversion, 1+2+2+2. It said "Six"
  in the commit that landed the change, written from the plan instead of from the diff, which
  is DESIGN.md §5 rule 8's own failure mode inside the batch that was auditing for it.
  Tally movement in the verified logs at `68fc493`, since an added assertion must be shown
  non-perturbing: `gate-p0` unchanged at 19/0/0; `gate-p1` 26→29 and `gate-p2` 22→25, each
  three per-host governor PASS plus one aggregate replaced 1-for-1; delegated `gate-p3` 48→49;
  delegated `gate-p4` unchanged at 65/0/0; `gate-p5` unchanged at 65/0/4. The p3 delta is the
  one that needed explaining and it is not a strengthening: that aggregate was previously
  `[[ -n "$SCALAR_FORCED" ]] || fail ...`, a **fail-only guard, silent on success**, so
  converting it to a three-way necessarily prints a PASS on a healthy fleet where the other
  six had a PASS to replace. No blocking power changed — the per-host `fail` at
  `gate-p3.sh:707` already blocked for a host that failed the forced run — and the old
  aggregate was satisfiable by any *single* host, which is #77's single-witness shape in a
  second place, harmless there only because the per-host verdict covered it.
- **`gate-p0` now reads the backend-availability marker instead of inferring the CPU from the
  run** (#73). Every unexercised backend was reported as `unavailable here` — a claim about
  the silicon deduced from a claim about the test run — so a backend the host *has* and the
  suite *skipped* was indistinguishable from one the host lacks. Available-and-unexercised is
  now a FAIL, absent is an `info`, and a missing availability marker is UNMEASURED with the
  wording of the fallthrough branch changed too: without that marker there is nothing to
  license the word "unavailable", so it no longer restates as a finding the inference the
  UNMEASURED line one row above says cannot be drawn.
- **A stated-assumptions ledger, printed beside every gate's verdict** (#73 tier C, ruled
  2026-08-15; `scripts/remote.sh`, all six gates). A precondition with **no read-back
  mechanism at all** — nothing to check, as distinct from something unreadable — gets no
  verdict: three verdicts (PASS/FAIL/UNMEASURED) plus an `assumed, unverifiable:` line, so
  the certificate enumerates what it is trusting instead of trusting it silently. A fourth
  verdict category would blur the one distinction UNMEASURED keeps sharp — *could have
  looked, could not read* — and would blur it in the direction that matters, since
  UNMEASURED blocks. `assumed()` sets no `FAIL`, prints at the `info` indent, and is
  invisible to the anchored tallies the delegating gates count with (driven and checked:
  a log carrying a full ledger tallies 1/0/1 against 1 PASS and 1 UNMEASURED). Two entries
  qualify, both circular rather than merely inconvenient: the configured host set being the
  intended fleet, and each name reaching the machine it is meant to reach — every witness of
  a host's identity is reported *by* the host under test. The admission test is deliberately
  hard, and two candidates failed it while it was written: machine load (#81) and SMT state
  (#82) are both readable, therefore both are missing criteria and are filed as such rather
  than laundered into the ledger.
- **Delegated gate logs are revision-stamped** (#78, before the green). `gate-p5` wrote its
  delegated `gate-p4` log to a fixed path and `gate-p4` did the same for `gate-p3`, so each
  run destroyed the only copy of the previous run's evidence. It bit during #73's own
  verification: the run being verified overwrote the reference it was to be compared
  against, and the diff survived only because an unrelated standalone `gate-p4` log happened
  to exist — which is not an archival strategy. #68 remains orthogonal and open: a stamped
  filename cannot distinguish a clean tree from a dirty one at the same revision, and a
  self-describing log that has been overwritten is still gone. Two defects, two fixes.
- **The converse sweep: 74 gate sites that report a reason the gate could not look now print
  UNMEASURED, under one rule written down where the primitive lives** (ruled 2026-08-15 on
  #73; `scripts/remote.sh`). #72 relabeled the 21 sites whose own message text already said
  "unmeasured" under a FAIL label. This is the other direction — every site whose message
  gives a *reason it could not measure* and prints FAIL anyway — and it lands before the
  first green rather than after, because unlike #72 it is not verdict-neutral by
  construction: it checks preconditions that were previously assumed, and a certificate
  issued by a gate carrying unchecked preconditions is a green with an asterisk. The rule,
  and it is a rule and not a phrase list: **FAIL when the gate obtained the reading the
  criterion asks for and the reading is wrong** — a test that ran and failed, a set that was
  enumerated and is short, a governor that answered `powersave`; **UNMEASURED when the
  reading does not exist** — the host did not answer, the run died, the marker is absent,
  benchstat established no interval, there was no host to ask. Tie-break for an arguable
  site: does the sentence assert something about *keel* or about the *measurement*? Only the
  first may be FAIL, and "keel does not reach 60% of OpenBLAS" is a claim about keel that a
  run with no hosts has not earned. **This harmonises the older governor rule rather than
  weakening it**: "unreadable counts as unmet" was written before UNMEASURED existed as a
  third column, and its intent — unreadability is not an exemption — is preserved exactly,
  since `unmeasured()` sets `FAIL=1` on the same line `fail` does. An unreadable governor
  still stops the gate; it stops *asserting the governor was wrong* when the truth is nobody
  could look. Distribution: 2 in `gate-p0`, 6 in `gate-p1`, 7 in `gate-p2`, 27 in `gate-p3`,
  19 in `gate-p4`, 13 in `gate-p5`. **Four of them were split rather than relabeled**, because
  one branch was two facts wearing one label: a boost knob nobody could read against one that
  did not move (`gate-p5`), a forced run that failed before reporting its dispatch against one
  that reported it and failed anyway (`gate-p5` — the marker is the discriminator), and a
  governor unreadable at measurement time against one that changed mid-run (`gate-p3`,
  `gate-p4`). Both branches of each still block. What **stays** FAIL is the audited residue,
  not what the grep missed: every "missing from the sweep", "lattice is incomplete", "does
  not match its own enumeration", "not on the allowlist", "below the bar", "governor is
  `powersave`", "only N of M configured targets ran" and every failed build or failed test —
  all readings the gate has. `gate-p5.sh:534` is the paired case worth reading beside its
  twin: **$ncpu CPUs were read** and the criterion names more, so the environment is wrong
  and the label says so, two lines below the unreadable-count branch that now says
  UNMEASURED. The sweep was audited in two batches: #73's own 37-site population, then the 37
  sites in the same *conditions* whose wording had kept them out of it — a phrase-defined
  population is not condition-closed, and relabeling `no execution hosts, so the scaling
  criterion cannot be evaluated` while leaving `P5 needs an amd64 host … none configured`
  three hundred lines up would have replaced one same-event-opposite-label defect with
  another.
- **21 gate criteria that already said "unmeasured" in their own message text now say it in
  their label, and no gate's path to green moved** (ruled 2026-08-15 on #72; `DESIGN.md`
  §5.6). Auditing every `fail` call across the six gates for what it *asserts* turned up 21
  whose message says the criterion could not be resolved — no toolchain, no reachable host,
  a run that died before it measured — under a label that says a check ran and observed a
  violation. Two of them indict themselves: `bench_expect`'s docs in `scripts/bench.sh` say
  an absent measurement has "exactly one verdict available to it — unmeasured" six lines
  above a caller printing FAIL, and `gate-p5`'s `race_verdict` header argues that collapsing
  its three states "sends whoever reads it looking for a race that is not there" immediately
  above the branches that collapse them. **Relabeling is not amendment**: `unmeasured()`
  sets `FAIL=1` exactly as each gate's `fail` does, so every one of these still blocks its
  gate, and what makes a criterion green is untouched — including the four `-race` criteria,
  whose non-amendable hard-red ruling of 2026-08-12 stands unchanged. What moves is the
  attributed cause, and a gate red for the wrong reason is as untrustworthy as one green for
  the wrong reason. Distribution: 13 in `gate-p5`, 6 in `gate-p3`, 2 in `gate-p4`. The sites
  are an audited list, not a grep: `gate-p5.sh:595` matches the same pattern and stays a
  FAIL, because "an unmeasured rung has appeared at Level 3" is a noun phrase about the rung
  and that check ran and observed a real violation — a blanket substitution would have
  relabeled a real miss as not-measured, the one direction of this change that *would* have
  been a weakening. `unmeasured()` is lifted to `scripts/remote.sh`, the file all six gates
  source; `gate-p5` did not define the primitive at all while 13 of its sites needed it, and
  minting a third divergent copy of a verdict primitive is how the delegated tally came to
  count two columns where the log had three. Also fixed in the same commit: `race_verdict`'s
  citation of #22's edge campaign as the checkptr remediation, which is the wrong address —
  the fix is upstream CL 761120 shipping in `go1.27`, so the path is #70's floor then #69's
  port (T23), and a red verdict that cites a campaign which will never resolve it is
  misattribution's little sibling. **Proven not to weaken anything by rerunning it**, which
  is the only form of proof this change admits: `gate-p5` went from 65 PASS / 4 FAIL / 0
  UNMEASURED to 65 PASS / **0 FAIL / 4 UNMEASURED**, both RED, both 215 lines, with the four
  messages identical line-for-line once the labels are normalised — the sole textual
  difference being the remediation pointer this commit deliberately corrected. Delegated
  `gate-p4: GREEN 65/0/0` and `gate-p3: GREEN 48/0/0` on both sides, unchanged.
  `build/gate-p5-8954f6d.log` against the archived `gate-p5-117b78f`.
- **`gate-p4`'s tally of the delegated P3 gate is anchored and has a third column** (#71).
  It counted with a bare `grep -c 'FAIL'`, which also matches any summary line inside
  `gate-p3`'s log, so a green delegate could be reported with a FAIL it did not have — the
  defect `gate-p5`'s tally of `gate-p4` already had fixed, in the file one level down. It
  now strips colour, anchors on `^  PASS  ` / `^  FAIL  ` / `^  UNMEASURED  `, and prints
  UNMEASURED as its own column, which it must: six of `gate-p3`'s misses became UNMEASURED
  above, and a two-column tally would have shown them as neither. The RED excerpt below it
  quotes both FAIL and UNMEASURED lines for the same reason.
- **gate-p4's criterion 7 is graded in three states, and the bar did not move** (ruled
  2026-08-15 on #67; `DESIGN.md` §4/P4). `Ssyrk ≥ 85% of Sgemm` was judged net of CI, which
  answers one question — "is the whole interval above the bar?" — and its negative answer
  was being read as "Ssyrk is too slow". Those are different claims, and §5.6 forbids one
  verdict standing for two causes. It had already bitten: janus read 87.6% raw with
  ±4.0%/±3.0% intervals and FAILed at a bound of 81.6%, then read 87.0% raw at ±0.0% on the
  same tree at the same commit and PASSed. **The raw quantity agreed to within 0.6 points;
  the FAIL was reporting the weather**, and it spent §4's single re-run allowance to find
  that out. So: **PASS** when the interval sits at or above the bar — `bench_ratio_lo >=
  0.85`, unchanged bit for bit — **FAIL** when the whole interval sits below it, and a new
  **UNMEASURED** when it straddles. The third state is carved out of the old FAIL and never
  out of the old PASS, so nothing that was below the bar can clear it on a lucky draw; that
  is why the raw ratio is *not* graded in place of the bound. Replayed against all six
  archived criterion-7 readings before landing: five stayed PASS, the noisy janus reading
  moved FAIL → UNMEASURED, and every synthetic edge case (exactly on the bar, unbounded
  arm, denominator interval reaching zero, missing benchmark) lands where it should.
  New in `scripts/bench.sh`: `bench_ratio_hi`, `bench_ratio_grade`, `bench_ratio_headroom`.
  The gate now also prints, per host per run, the interval, both observed CIs and the
  **flip-headroom** — the symmetric CI at which the bound would land exactly on the bar,
  `(raw − bar)/(raw + bar)`: 4.17% on the 7950X3D, 1.16% on the i9-9960X, 1.85% on the AI
  MAX+ 395, against intervals those hosts produce up to 1.0%, 3.0% and 2.0% in one run. Two
  of three were deciding this criterion on how quiet the machine was and nothing in the log
  said so. And where the headroom is under 0.5% with an arm whose CI printed `0.0%`, the
  gate says the verdict lies inside benchstat's integer-percent rounding (T21) rather than
  letting the bound read as exact. The remedy for UNMEASURED is precision — one archived
  re-run, then a higher `-count` for this criterion on a chronically undecidable host —
  never a wider judgment. `gate-p5`'s delegated tally counts the new label as its own
  column, because a tally with two columns would have made a straddled interval vanish.
- **`docs/toolchain-notes.md` gains T23 and T24, and T17's "no keel change at all" is
  corrected: `go1.27` moves the floor, but keel does not compile on it yet.** The
  2026-08-15 ruling makes go1.27 keel's minimum and orders `-race` on the vector path as
  the first act under it. Probed on all three benchmark hosts with `go1.27rc3` installed
  to its own prefix (`/usr/local/go1.27rc3`, leaving `/usr/local/go` on `go1.26.5` so no
  published number's compiler moves silently — #58), and the probe was written with four
  outcomes rather than two: `compile-fail` / `checkptr` / `data-race` / `clean`. It came
  back **`compile-fail`, identically on all three hosts**, which a two-outcome probe would
  have reported as "checkptr still broken". `simd/archsimd`'s load/store were renamed with
  a **swap** — the slice forms took over the bare names (`LoadFloat32x16Slice` →
  `LoadFloat32x16`, `StoreSlice` → `Store`), the displaced array forms gained an `Array`
  suffix, and the `…SlicePart` forms became `…Part` *and grew a return value* — because
  `archsimd` is converging on the naming convention of the portable `simd` package that
  1.27 also ships. `go build -gcflags=-e ./...` gives 51 errors, all type errors, in 3
  files: 39 pure renames, 4 renames that gain a return value, 8 array-form sites that need
  the `Array` suffix. **Every one is compile-caught**, because `*[N]float32` and
  `[]float32` are mutually unassignable, so the swap cannot rebind silently — the property
  that makes "it still compiles" sufficient evidence here and insufficient in general.
  So the chain to the four `-race` criteria is **floor → port `internal/vec` → `-race` →
  criteria**, not floor → `-race` → criteria, and the criteria are `unmeasured` on 1.27
  rather than clear. The port is specified but **held**: `go1.27.0` final does not exist
  yet (go.dev/dl has rc3 as tip, 1.26.6 as stable), the ruling's own condition is 1.27.0
  final installed and read back on all three hosts, and landing it now would break three
  hosts on `go1.26.5` and the dev host on `go1.26.6` at once. The consolation is DESIGN.md
  §3's shim bet paying off against a real API break: 3 files, 51 lines, **zero change to
  keel's own API** (T5, T17, T23, T24; #42, #22).
- **T5 resolves the way it guessed and the conclusion drawn from it does not: 1.27 ships
  the portable `simd` package, and DESIGN.md §8's plan to park the ARM64 kernel behind it
  needs revisiting rather than resuming** (T24). The package is there with amd64, arm64 and
  wasm backends plus a pure-Go emulated fallback, one vector type per primitive numeric
  type, and a bridge both ways (`ToArch() any`, `Float32sFromArch[T]`). But its vector
  length is a **runtime** quantity — `simd.VectorBitSize()`, `simd.Emulated()`,
  `(x Float32s) Len() int`, no compile-time constant anywhere — and a register-blocked
  microkernel is exactly the thing that cannot be vector-length-agnostic: `MR`, `NR` and
  `Lanes = 16` are constants because the accumulator tile is a fixed set of named
  registers, which is the whole content of the P2 spill audit. There is also no
  `GetLo`/`GetHi` on `Float32s`, so `HSum`'s bit-exact fold tree is not expressible either.
  The kernel stays on `archsimd`. The interesting half is the other one: an
  emulated-plus-arm64 vector path is the first thing that could run a *vector* differential
  test **on the dev host**, which T1 has forced onto ssh since P0 — but an emulated FMA
  that fuses where hardware does not would be a worse oracle than none, so that is filed as
  a question, not adopted. fixed upstream in go1.27, and the
  `-race` doubt that would have made the fix useless is measured away.** CL 761120 marks all
  30 `archsimd` `pa*` helpers `//go:nocheckptr`; the pragma count in
  `src/simd/archsimd/unsafe_helpers.go` is 30 on `go1.27rc1` and 0 in go1.26.6, so every
  1.26.x reproduces T17 and 1.27 does not. golang/go#42880 (*"-race does not obey
  go:nocheckptr"*, open since 2020) appeared to mean the fix silences `-d=checkptr` but not
  `-race` — which would have left keel's four `-race` criteria blocked on a five-year-old
  issue rather than on a merged CL. Reading that thread instead of its title settles it the
  other way: the failing conversion there was inside a *function literal*, where no `//go:`
  directive can attach, and `-race` was incidental. Measured on a declared cross-package
  helper carrying T17's exact conversion, the pragma suppresses the fatal under `-race` and
  under `-d=checkptr`, while the no-pragma control fatals under both — and the mechanism is
  `inline/inl.go:349-351` refusing to inline a `go:nocheckptr` function under any
  `Checkptr != 0` build, confirmed by counting `CALL`s in the object code (0 uninstrumented,
  1 under each instrumented arm). **So the copy-into-a-full-width-array workaround is a
  1.26.x bridge, not a permanent spelling**, and its price — all ten Level-1 kernels losing
  `nosplit`, `internal/l1` +15.5% static instructions — is paid only while keel builds on
  1.26.x. `checkptr`-cleanliness as an admissibility condition on #22's candidates is
  satisfied by the toolchain rather than by a workaround, so the masked-partial candidate
  gets measured as written (#42, #22).
- **The scaling criterion's two arms now run in one frequency regime, and the boost-on
  speedup prints beside the verdict** (ruled on #66; `DESIGN.md` §4/P5). As first
  written, "≥6× single-thread throughput at 8 cores" divided an 8-thread rate by a
  1-thread rate taken on an idle machine — but one thread runs at a boost clock eight
  threads physically cannot reach, so the criterion asked the nest to overcome silicon
  boost policy before it was allowed to demonstrate scaling. **A denominator measured in
  a regime the numerator cannot legally enter is not a ratio, it is a handicap.**

  The diagnosis came from the shape of the misses rather than from their inconvenience:
  the two hosts that missed are the two retaining the *most* of their own single-thread
  peak (Zen 4 92%, Zen 5 59%), while Skylake-X at 35% cleared all four routines twice.
  Scaling deficits do not sort themselves by single-thread excellence; boost tables do.

  `scripts/gate-p5.sh` now sets `cpufreq/boost` (AMD) or `intel_pstate/no_turbo` (Intel)
  off per host, **reads the knob back** — unreadable or unmoved counts as *unmet*, never
  as satisfied, exactly as `scaling_governor` is re-read at measurement time — judges
  ≥6× there, restores boost, and takes a **second pass boost-on** whose wall-clock
  speedup against the idle single-thread rate prints at equal prominence as
  reported-never-judged. That second number is what a caller experiences and no reader
  gets the pass without it. Hosts are restored on `EXIT INT TERM`, because a gate that
  dies mid-window must not leave a machine de-boosted for the delegated gate-p4/p3/p2
  runs, which are boost-on measurements.

  **Stated rather than buried: this makes the criterion easier.** Smoke-measured on the
  Ryzen 9 7950X3D at n=4096 Sgemm (`-test.count=3 -test.benchtime=0.4s`, a §5 rule 5
  smoke run informing no gate): 1 thread 152.6/151.7/152.6 → 122.7/122.4/122.3 GFLOP/s,
  8 threads 905.2/895.0 → 774.6/772.9. So boost off costs the 1-thread arm 19.8% and the
  8-thread arm 14.0%, and the ratio rises 5.90× → 6.32× — which is enough to move vesta
  Sgemm's 5.74× miss across the floor, and is *not* obviously enough for the two Ssymm
  misses at 5.63× and 5.34×. The gate decides that, not this paragraph. The de-boosted
  regime also lowers the formula cross-check (`cpuinfo_max_freq` 5.76 → 4.20 GHz, peak
  368.9 → 268.9), so the boost-off pass's percent-of-peak lines are quoted against a
  de-boosted peak measured in the same pass. The justification is not that the new number is nicer
  but that the old one was not a ratio — and the honest consequence is that boost-off
  ratios are **not comparable** to the three boost-on runs already in the record. The
  README's published rates stay boost-on, since a published row is a claim about what a
  caller gets, and criterion 9 re-measures them against the boost-on pass accordingly.
- **`DESIGN.md` §5 gains rule 8: a summary is a cache with no invalidation protocol.**
  Derived figures are recomputed from the log at the moment of writing, never carried
  forward from prose. Third documented instance across two authors, which is what makes
  it a named trap rather than a slip.
- **`DESIGN.md` §5 gains rule 9: a citation is a claim about where the grounds are, and
  it is checked like any other claim** (#85). Every gate cites `DESIGN.md` for its
  authority, so a citation landing on the wrong rule misdirects every reader who follows
  it while looking exactly like a correct one. The rule names three non-interchangeable
  instruments — **resolution** (does it land), **pinning** (has it moved), and
  **mint-verification** (was it ever right, which cannot be automated) — and legislates
  the forward convention: new citations use the explicit `§5 rule 5` form; audited-correct
  `§X.Y` shorthand stays byte-for-byte; a citation naming another document is declared,
  not guessed at.
- **`make lint` and CI's stock job now run `scripts/citation-lint.sh`** (#85), which
  resolves every `DESIGN.md` rule citation in the tree against `DESIGN.md`'s actual
  structure and pins each distinct form beside the first words of the item it lands on
  (`docs/citation-targets.txt`). It lives in the stock job because it needs a git
  checkout but no toolchain, so it cannot be skipped.
  - **The premise the check was opened on was false, and that changed the law rather
    than the sweep.** The 18 `§5.4` citations were assumed to be a *renumbering* casualty. <!-- citation-lint:quote -->
    They are not: `git show 4643b63:DESIGN.md` shows the methodology rule was already
    item **5** in the same commit that wrote the first `§5.4`, and §7's eight leads are <!-- citation-lint:quote -->
    byte-identical from `6a862d7` to `HEAD`. Nothing was ever renumbered — the ordinals
    were **mis-minted at birth**. So a pin certifies *stability*, never
    *birth-correctness*, and would have frozen all 18 defects with perfect fidelity while
    passing forever. The baseline is therefore **audited**, once, at the meaning level:
    every citation's ordinal read against the content the citing site actually invokes,
    with the audit's date and census recorded in the pin file so a later reader knows the
    freeze rested on a reading rather than an assumption.
  - **Census: 120 sites; 8 naming another document; 112 `DESIGN`-bound, of which 92 land
    on the content they invoke and 20 were mis-minted.** Sixteen were rewritten to the
    explicit form (14 × `§5.4`, plus two found *inside* populations that resolved <!-- citation-lint:quote -->
    perfectly: `DESIGN.md`'s own §7 citation of the tolerances rule where it argued the
    denominator rule, and `l1_test.go`'s §5 citation of the differential rule where it
    argued the tolerance-model rule). Four are deliberate quotations of the bad form and
    are **marked, never normalised** — the record of the defect is the point, same law as
    #79. The 25 audited-correct shorthand sites were left untouched: rewriting a correct
    citation in the document the gates cite as grounds is churn dressed as rigour.
  - **The two mis-minted sites outside the `§5.4` population were found by an instrument <!-- citation-lint:quote -->
    the other two cannot supply: cross-site argument identity.** `scripts/gate-p5.sh` and
    `DESIGN.md` made the *same* argument — "an advertised chain whose middle link no gate
    can back is a claim, not a measurement" — under two different rule numbers, and
    reading the rule bodies picked the denominator rule over the tolerances rule. Content
    adjudicates, not majority.
  - **Scoping to the document a citation actually names is the check's load-bearing
    feature, and it was proven by near-miss.** An earlier draft condemned `§X.Y` *by form*,
    on the true observation that `DESIGN.md` has no subsections. That draft would have
    demanded 25 zero-semantic edits to correct citations **and** filed a peer-reviewed
    paper's section numbering (Van Zee & van de Geijn, TOMS 2015 §4.3) as a defect in
    keel's constitution — the false-defect class #63's near-filing established the norm
    against. Such references are now **declared** in the pin file, each declaration
    carrying **the number of sites it covers**, so an exemption cannot silently widen:
    fewer means it is stale, more means it has grown to cover a citation nobody read,
    and the fix is to read the new site and bump the count deliberately. The invariant
    is not "exactly one site" — `CHANGELOG.md` legitimately cites the paper twice — it
    is *exactly the number someone read*. That check caught its own documentation: this
    entry added two more `§4.3` references, the count went red, and reading them found <!-- citation-lint:quote(4.3) -->
    one that does not cite the paper at all (it names the notation while explaining the
    marker) and so belonged in a quote marker rather than the exemption.
- **`scripts/citation-lint-test.sh` drives all ten of the lint's branches on purpose**
  (#85). A healthy tree reaches exactly one of them, so a green from the lint alone is
  not evidence that any check but the clean path works — and three of these controls guard
  a *silent* failure rather than a loud one. Two defects were caught by running it rather
  than reading it: rewrapping a comment moved a `citation-lint:quote` marker one line off
  its citation, which stops suppression with no diff to notice; and the harness's own
  `EXIT` trap referenced a `local` variable out of scope, leaking its temp directory. An
  earlier draft also restored with `git checkout --`, which reverted two files to `HEAD`
  and discarded uncommitted work — file backups cannot do that.
- **The check's coverage is bounded by `git ls-files`, and that bound bit immediately**
  (#85). `sites()` enumerates tracked files only, so a new file's citations are invisible
  until it is committed: both new scripts were untracked while being written, the lint
  could not see its own sources, and **the commit that added them turned a green tree
  red** — 4 unresolvable mentions of the BLIS `§4.3` appeared the instant they became <!-- citation-lint:quote(4.3) -->
  tracked. The verification that missed it is worth naming too, because it is a repeat:
  `make lint 2>&1 | tail -2` returns *`tail`'s* exit status, so a failing build read as a
  passing one, the same swallowed-verdict shape as a `permission denied` once read as a
  clean grep. A green from this check means *"every citation in a tracked file"*, and
  `git status` is part of reading it. The commit that recorded all this then repeated the
  shape a third way: `make lint; git add …` commits whether the lint passed or not, so a
  second red tree reached `main` and was fixed in the commit after. A verification that
  does not gate the action it precedes is a report, not a check — `&&`, always.
- **The quote marker takes a scoped form, `citation-lint:quote(5.4)`** (#85), because
  line granularity is too coarse for this document: `DESIGN.md`'s numbered rules are
  single 2000-character lines, and §5 rule 9 quotes `§5.4` on a line that also carries a <!-- citation-lint:quote(5.4) -->
  live `§5 rule 5` and an external `§4.3`. A bare marker there would have stopped <!-- citation-lint:quote(4.3,5 rule 5) -->
  checking all three while printing nothing — over-suppression by construction, the same
  failure the declaration-narrowness check guards on the other side.
- **Every `gate-p2` classification branch now publishes the measured interval beside the
  point estimate, or states why it has none** (#86, `scripts/gate-p2.sh`). Three branches
  still reported the point spread alone — `samemix`, `falsified` and `nearceiling` — and
  `falsified`/`falsifiedanyway` reported a point attainment where an interval was
  available. §5 rule 8's publish-the-pair clause has no per-branch exemption, and the
  branch a reader reaches is exactly the branch whose numbers they need: a pair present
  in eight renderings and absent in the ninth reads as *"this one had no interval"*
  rather than *"this one forgot"*. `nomixes` is the one true exemption — no ceiling was
  established, so there is no spread to bracket — and now says so in its own text. The
  invariant is recorded above the `case` so the next reader sees a rule rather than a
  pattern; it has drifted twice.

- **All ten `internal/l1` vector loops now guard with `>` rather than `>=`**, which removes
  the branchless conditional pointer bump that bounds-check elimination had bought them
  (T19, #47). Under `>=`, `x = x[16:]` may leave the slice empty and an empty slice may not
  carry a past-the-end pointer, so the advance compiles to `MOVQ`/`NEGQ`/`SARQ`/`ANDL`/`ADDQ`
  — five instructions computing an offset that is 64 on every iteration but the last, paid
  twice by the routines that advance two slices. Under `>` the emptiness case is gone and it
  collapses to `ADDQ $64, AX`. Steady-state loop instructions (linux/amd64, go1.26.6, whole
  body including T9's NOPs): `avx512Axpy` 26 → **15**, `avx512Scal` 16 → **9**, `avx512Dot`
  52 → **44**, `avx512Asum` 29 → **25**, `avx512SumSq` 34 → **29**, and the avx2 twins 29 →
  18, 19 → 12, 52 → 44, 41 → 37, 33 → 29. Vector-op counts are unchanged in all ten, and all
  ten still audit at 0 bounds-check exits, 0 calls and 0 vector stack references. The
  reductions gain least because their advance amortizes over 64 elements where Axpy's and
  Scal's amortizes over 16 — which is the shape of #47's regression, since Saxpy was the
  routine that lost 40.65%.

  A `>` guard exits with a full vector unconsumed, and the partial tail cannot absorb it:
  `LoadFloat32x16SlicePart` is documented as equivalent to a full load at 16 or more
  elements, so it would silently ignore a 17th. The two loop shapes handle that differently.
  The reductions' existing 16-wide mop-up loop keeps its `>=` guard and drains the ≤64
  elements in at most four iterations, which was expected to leave their tail unaffected and
  did not — see the `### Fixed` entry above, which is the measured correction. Axpy and Scal
  have no second loop and get an explicit exact-fit epilogue running the body once at *full*
  width. Leaving that to the masked tail instead was rejected on the deciding ground that it
  would make **every** exact-multiple call execute a partial op where today only ragged
  lengths do, and partial ops are what #42 makes fatal under `-race`: the epilogue holds that
  frequency where it was rather than trading instructions for a wider `checkptr` blast
  radius. Two other shapes were measured and rejected — an early `return` on exact fit to
  hand `prove` a `len != 16` fact (unused: Scal 16 → 15, Axpy *worse* at 27), and dropping
  the redundant `&& len(y) > 16` conjunct (Axpy 15 → 24, so the conjunct is load-bearing).

- **`internal/pack`'s contiguous branch no longer calls `memmove` for short runs**
  (#21). `copy()` on a slice of statically-unknown length is a `runtime.memmove`
  call — confirmed in the object code (`pack.go:169 CALL runtime.memmove`) — which is
  the right instrument for B, whose blocked axis is `NR = 32` (128 bytes per run),
  and the wrong one for A, whose blocked axis is `MR ∈ {2,4}`: one call per 8 or 16
  bytes, about 27,600 of them per pass. Below a `memmoveFloor` of 16 elements the
  branch now uses a plain assignment loop. The loop assigns rather than multiplying
  by an alpha known to be 1, so the *signalling*-NaN asymmetry `TestBranchesAgree`
  documents stays exactly where it was documented instead of moving.

  Dev host (Apple M4 Pro, `GOMAXPROCS` unset, `-benchtime=200ms -count=8`,
  `go tool benchstat`; a data-movement rate, so the denominator is each cell's own
  baseline and no percent-of-peak is claimed):

  | case | before | after | |
  |---|---|---|---|
  | `2x32/TN/pack-a` (blk=2) | 787.2m ± 2% | 1119.0m ± 6% Gelem/s | **+42.1%** (p=0.000) |
  | `4x32/TN/pack-a` (blk=4) | 1.529 ± 5% | 1.569 ± 3% Gelem/s | +2.6% (p=0.000) |
  | `2x32/NN/pack-a` (transposing) | 2.143 ± 13% | 2.067 ± 8% Gelem/s | ~ (p=0.185) |
  | `4x32/NN/pack-a` (transposing) | 2.365 ± 9% | 2.189 ± 4% Gelem/s | −7.5% (p=0.028) |

  Two caveats stated rather than smoothed. **The 4x32/NN −7.5% is in a branch this
  change does not touch**, so it is either noise (that cell's baseline varies ±9–12%
  and its 2x32 twin shows no change) or an instruction-layout effect from the
  `switch`; it needs the amd64 hosts to resolve and they were measuring #26 at the
  time. And **this does not close #21**: the contiguous branch at `blk=2` is still
  1.85× slower than the transposing branch, down from 2.72×, so the remaining gap is
  the per-k-step overhead of a two-element inner loop, not `memmove`.

  Worth being explicit about the blast radius: `APanels` passes `!trans`, so at `NN`
  — the shape every gate benchmark runs — A takes the *transposing* branch and this
  changes nothing. The gain is on `TN`/`TT`, i.e. for callers who pass `transA`.
- **P5's internal order is now stated: single-thread remediation, then the parallel
  loop nest, then the scaling gate** (`DESIGN.md` §4/P5, ruled 2026-08-12). #26
  (retention), #36 (Ssymm's dense expansion), #37 (Strsm's scalar diagonal solves)
  and the deferred measurements on #21/#22 all sequence *before* the
  parallelization rather than beside it: each is a single-thread cost that
  parallelization multiplies rather than hides, so parallelizing first would
  certify scaling curves for routines the same phase intends to change — every
  number re-measured and a record carrying two regimes. The certifying measurement
  comes last, over the final artifact, which is the same rule that made P3 keep the
  hardened re-run instead of the green it inherited. #21 and #22 join the campaign
  because they are the same code as #36/#37, and entering it twice is how a
  measurement ends up compared against a different build than it was taken on.
- **P5's scaling floor binds by parallelism class rather than by routine list**
  (`DESIGN.md` §4/P5, ruled 2026-08-12; the question was whether P4's derived
  routines are judged or only `Sgemm`).
  - `Sgemm`, `Ssyrk` and `Ssymm` are **one class — GEMM-shaped nests over
    independent tiles, no cross-iteration dependence — and the ≥6× floor binds all
    three.** Judging only `Sgemm` would let a serialization bug in the triangular
    C-update masking hide behind "the derived routines are reported, not judged",
    which is both the likeliest place to introduce a dependence and the measurement
    that would catch it.
  - `Strsm` is a **different class, and its floor is deferred to a measurement.**
    Its diagonal solves impose a dependency chain the other three lack, and its
    available parallelism varies with `side` and shape. P5 measures its scaling,
    reports it beside the judged three, and states the parallelism model behind the
    number — the rank-update/diagonal-solve split at the gate's shape; that pair
    sets the floor, which binds from the commit recording it forward. Recorded on
    issue #37 so the deferral is a named debt and not a standing exemption. Writing
    6× on `Strsm` today would be a threshold without a model, which is the move
    this project has now refused six times.
- **`Sgemm` selects its microkernel shape per host instead of taking the registry's
  first entry** (ruling on issue #24; `KERNEL.md` §8, `DESIGN.md` §4/P3). Both
  shipped shapes are zero-spill and neither dominates — 4×32 wins on Zen 4 and
  Zen 5, 2×32 wins on Skylake-X by 11 percentage points — so shipping one shape
  everywhere was a measured performance bug on one of three gate hosts. Dispatch
  now classifies the host with the same issue-bound/FMA-bound classification the
  gate's throughput model already defines and takes the shape extremal on that
  class's binding cost: fewest memory ops per FMA when arithmetic binds, fewest
  instructions per FMA when the front end does. `KEEL_KERN_CLASS=fma|issue`
  overrides the classification, and an unrecognized value panics rather than
  falling back, for the same reason `KEEL_FORCE` does.
  - The classification is a feature-bundle fingerprint, because no
    microarchitecture is readable from pure Go on this toolchain (T14, #25). It is
    printed with its grounds and checked against the gate's own measured verdict on
    every host on every run; disagreement is a gate failure.
  - `gate-p3.sh` criterion 5b: the throughput sentinel now judges the *dispatched*
    shape rather than whichever shipped shape measures fastest, fails if a
    passed-over shape beats it net of CI in the same invocation, re-measures the
    blocked `Sgemm` at 2048³ under the other class as a cross-check, and re-derives
    every shipped shape's recorded `InsnsPerFMA` from the object code so the
    ranking cannot come to rest on a stale count.
  - Found by the gate, not by a benchmark: P2's anti-vacuity shape guard refused
    the dispatched 4×32 a roofline on janus, which is what surfaced the bug.
  - The gate model now states a **one-retry allowance for throughput sentinel readings**
    (`DESIGN.md` §4): a failing sentinel triggers exactly one re-run, fails only if both
    runs fail, and both outputs are archived either way, so a pass-on-retry is visible
    in the record. It never applies to a correctness criterion — those fail on first
    miss, since a differential test that passes on retry has found a nondeterminism.
    The gate script is unchanged and still fails on first miss; the allowance is the
    operator's and the archive is what keeps it bounded. How often the retry is needed
    is itself the signal: a sentinel that needs it often is reporting that the margin
    is gone, not that it is noisy, and the answer is throughput or a re-derived bar
    rather than more retries.
  - `gate-p3.sh` also prints **retention** per host — the share of its own microkernel
    the blocked loop nest keeps — as reported-never-judged provenance beside
    percent-of-peak. vesta ~90%, antares ~92%, janus ~77%, so P5 inherits #26 as a
    re-runnable measurement instead of a remembered figure (`DESIGN.md` §4/P5 names it
    as a carried-in input). It is a ratio of two point estimates from two invocations
    and is labelled as such, with both inputs printed; nothing compares it to a
    threshold, because closing that gap means sweeping `KC`/`MC`/`NC`, which is P5 work.
  - `docs/spill-report.md` carries a superseded-by note: its P2-era prediction that
    "P3 will dispatch this kernel on all three machines" is the premise this change
    removes. The report's measurements are left as recorded — it documents P2, not
    what ships now.
- **DESIGN.md's 32×6 microkernel tile is not implementable on go1.26.5, and P2
  ships two smaller shapes instead** (`docs/toolchain-notes.md` T10, issue #18,
  rationale in `KERNEL.md`). Two independent properties of the toolchain, both
  read out of the compiler's own source and confirmed by probe: the register
  allocator offers SIMD values only X0–X14 (`fpRegMaskAMD64` is `0x7FFF0000`; X15
  is the ABI zero register and X16–X31 appear in no allocatable mask at all, even
  though `VFMADD213PS512`'s register shape lists them as legal), and only the
  `213` FMA form exists — it writes to its first multiplicand, so `acc += a·b` can
  never land in `acc` and always needs a live scratch register. DESIGN.md's budget
  of 12 accumulators + 2 B vectors + 1 broadcast = 15 is therefore one register
  short of allocatable, and measured across a 115-shape sweep *every*
  12-accumulator configuration spills. The zero-spill frontier is 8 accumulators;
  0.75 loads per FMA is a hard floor below it, since a lower ratio needs 9. The
  DESIGN.md tile is kept as `kern.ReferenceTile` — audited, differential-tested and
  benchmarked, deliberately absent from `kern.Kernels()` — so the cost of the
  constraint is a measured GFLOP/s number rather than an assertion, and the gate's
  zero-spill criterion stays binding on everything that ships.
- **The microkernel tile is reflected relative to DESIGN.md: MR rows × NR columns,
  vectors along N** (issue #16). keel's public API is row-major (DESIGN.md §3), so
  in an M-vectorized tile sixteen consecutive elements of a column are `ldc` apart
  and every accumulator lane lands on a different cache line — 192 scalar stores
  per tile, each needing a lane extract that archsimd only offers as a
  store-and-reload, i.e. a spill. The M-vectorized tile would fail P2's own audit
  for a reason unrelated to the compiler being audited. The arithmetic and the
  register pressure the phase was designed to test are unchanged.
- **The K-loop bodies live in `internal/vec`, not `internal/kern`.** Reaching
  archsimd through a shim costs one 1-byte `XCHGL` statement anchor per inlined
  wrapper *with a Go body* per call site (`docs/toolchain-notes.md` T9). Measured on
  the 2×32 body: 90 instructions with 24 anchor NOPs through two wrapper levels
  against 74 with 8 when `internal/vec` names archsimd directly — 27% of the loop
  body was anchors. `internal/kern` is now the shape registry, tile protocol and
  scalar reference; the "all simd imports in `internal/vec`" rule is unchanged.
- `scripts/gate-p2.sh`: real P2 checks. The throughput floor applies to the best
  *shipped* shape per host — P3 dispatches to one of the two, so failing a host for carrying
  a second kernel it would never select would measure the wrong thing — with every
  shape's number printed either way, numerator and denominator taken in the same
  benchmark invocation, and the audit of the deliberately-spilling reference tile
  run as explicitly non-fatal evidence.
- **`scripts/gate-p3.sh` is GREEN: 47 PASS / 0 FAIL**, on three consecutive full
  runs — at `6c0f722` (the closing verdict for P3) and again at `0271dd7`, which
  changed what criterion 6 executes on two of three hosts and so had to be
  re-measured rather than assumed: vesta 91.0%, janus 73.8%, antares 65.7%. P3's
  headline criterion — single-thread `Sgemm` at 2048³ against ≥60% of the same
  host's own OpenBLAS — is met on every gate host, each against a reference chosen
  by measurement and verified to have taken. All three fully-measured runs, for
  reproducibility (every number is from one invocation per host under the
  `performance` governor, `-count=10 -benchtime=1s`, medians net of benchstat's
  confidence interval):

  | Host | reference family | keel / denominator | run 1 | run 2 | run 3 |
  |---|---|---|---|---|---|
  | vesta, Zen 4 (7950X3D) | Haswell, +5.5 to +6.5% over `DYNAMIC_ARCH`'s Cooperlake | openblas, FMA-bound | 91.8% | 91.1% | 91.0% |
  | janus, Skylake-X (i9-9960X) | SkylakeX, unpinned | roofline, issue-capped | 73.5% (plain OpenBLAS 40.3%) | 73.6% (40.4%) | 73.8% (40.4%) |
  | antares, Zen 5 (Ryzen AI MAX+ 395) | Cooperlake, unpinned | openblas, FMA-bound | 65.3% | 64.5% | 65.7% |

  Reported, never judged: percent of measured peak is 88.1% (vesta), 46.1%
  (janus), 58.5% (antares), and retention — how much of its own microkernel the
  blocked loop nest keeps — is 91%, 77% and 91%, which is issue #26's P5 baseline
  as a measurement rather than a recollection. janus's sentinel, P2's floor re-run
  on the dispatched shape and the tightest margin in the gate, holds at 46.1% of
  peak = **94.8%** of its 48.6% issue roofline without using the one re-run the
  sentinel retry policy allows.
- `scripts/gate-p3.sh`: real P3 checks, written before any P3 code. Three things
  in it are decisions rather than transcriptions of DESIGN.md §4/P3, and are
  stated in the script so they can be argued with:
  - **The sweep's extent is enforced, not trusted.** A green `go test` proves
    only that whatever ran, passed, so the tests print coverage markers and the
    gate parses them: every size in DESIGN.md's list, the complete transpose
    lattice, and alpha and beta each covering 0, 1 *and* a general value — 0 and
    1 are the special-cased paths, so a lattice of only those would exercise
    every shortcut and never the general multiply. The enumerated sets must
    multiply out to the reported combination count, so a marker cannot claim
    combinations it did not run.
  - **The oracle's cost is a declared property of each size.** A float64 oracle
    at 2048³ is 8.6 GFLOP per combination. Sizes up to 65 must be verified in
    full; 500, 1000 and 2048 may be verified by a seeded sample of exactly
    computed entries, but only by saying so per size, with a floor of 256 entries
    and a printed seed. "We sampled" is the kind of concession that starts at
    2048 and ends up applying to 17.
  - **The OpenBLAS bar runs where both halves can execute.** Read as "measure on
    the dev machine", the ≥60% criterion is vacuous here: this dev host is
    `darwin/arm64`, so the ratio would compare OpenBLAS-on-arm64 against keel's
    scalar fallback. The gate instead keeps the *comparison* dev-only — behind the
    `openblas` build tag, out of the module's dependency graph — and runs it on the
    amd64 hosts in one invocation each, from a native build of `git archive HEAD`,
    with single-thread verified from the harness's own report on both sides. A host
    with no reference FAILS and gets the exact provisioning commands for its own
    distribution; percent-of-peak is not accepted as a substitute.
  - **The reference is same-host, on every gate host** (ruling on issue #23).
    There is no reference-host list and no golden machine: the only
    apples-to-apples ratio is same silicon, same thread count, same run, so each
    host is divided by its own OpenBLAS and the version and selected target are
    recorded beside every ratio. The `DYNAMIC_ARCH`-chosen kernel family
    (`openblas_get_corename()`) is *checked*, not just printed, against an
    AVX2-or-better allowlist, because that is the one part of the reference's
    configuration whose failure mode is in keel's favour — a generic kernel on an
    AVX-512 host reads low and inflates the ratio while the version, the thread
    count and the config string all still look right. An unrecognized name fails
    too. On an **issue-bound** host the denominator becomes
    `min(same-host OpenBLAS, roofline × measured peak)` (same ruling, citing
    #17/#18): OpenBLAS there is hand assembly folding accumulation and an embedded
    broadcast into single FMAs, which the intrinsic layer provably cannot emit
    (T12), so 60% of it is a demand on the decode stage rather than on the kernel.
    The decision is the pure function `p3_denominator`, unit-tested by nine new
    fixtures in `scripts/roofline-test.sh`: it can only ever *lower* a denominator,
    it applies only where P2's classifier independently says issue-bound, it
    carries P2's anti-vacuity shape guard against the shape `Sgemm` actually ran
    (which as this ships refuses the dispatched 4×32 — issue #24), and it retires
    itself with no expiry clause as the lowering improves. vesta and antares
    classify FMA-bound and face the unmodified criterion. Both ratios are printed
    on every host: §7 rule 7 applies to the gate's own arithmetic too.
  - **The reference is the fastest *measured* OpenBLAS on that host, chosen by a
    coretype sweep** (ruling on issue #31), not whatever `DYNAMIC_ARCH` selected.
    The gate forces `OPENBLAS_CORETYPE` through {default, Zen, Haswell, SkylakeX,
    Cooperlake, SapphireRapids}, records every candidate's achieved corename and
    GFLOP/s, pins the winner for the run that produces the ratio, and *verifies*
    the pin took by comparing the library's own `corename` in the measured run
    against the sweep's. The allowlist is now the floor ("modern enough"), the
    sweep the ceiling ("best this silicon can do"). `DYNAMIC_ARCH` dispatches on an
    ISA feature bit, so on vesta's Zen 4 it ships the full-width Cooperlake kernel
    onto a double-pumped 256-bit datapath: 149.5 GFLOP/s where the AVX2 Haswell
    kernel measures 159.5, a 6.7% understated denominator — keel's own issue #24
    with the vendors reversed, which the allowlist structurally cannot catch
    because it contains both the right and the wrong answer for every gate host.
    Selection is best-of-N; the number that enters the record is still measured
    under the full §5 rule 5 methodology with the winner pinned. antares and janus
    default correctly (Cooperlake 297.2, SkylakeX 193.5) and the sweep confirms it
    by measurement rather than by assumption.
  - **The performance governor is asserted on every host in a preamble, not
    assumed** (same ruling). Anything but `performance` fails that host before a
    benchmark runs, and an unreadable governor fails too: an unchecked
    precondition is not a met one. It is then re-read at measurement time, so a
    governor that changed after the preamble fails that host rather than passing
    on a stale check. This replaces a tally that any single host could satisfy —
    "at least one host cleared the bar under the performance governor" — which let
    antares contribute numbers from `powersave`, where the first reading of its
    sweep was 245.0 GFLOP/s against a 296–297 steady state: an 18% error in a
    denominator, decided by how recently the core had been busy.
  - **`SGEMM_BENCH_FILTER` never ran the OpenBLAS benchmark** (issue #32,
    `docs/toolchain-notes.md` T15), found by the first gate run that could reach
    criterion 6. `go test -bench` splits a pattern on top-level `|` *first*, into
    an alternation of whole patterns, and only then splits each alternative on
    `/`. So `Peak|Sgemm|OpenBLAS/avx512|n=2048` was four alternatives — `{Peak}`,
    `{Sgemm}`, `{OpenBLAS,avx512}`, `{n=2048}` — and not the two-level filter the
    comment above it claimed. `{OpenBLAS,avx512}` cannot match anything, because
    `BenchmarkOpenBLAS`'s children are `n=…`, so the reference's benchmark never
    ran and criterion 6 had no denominator; `{Peak}` and `{Sgemm}` were
    depth-unconstrained, so every gate run also paid for three `Sgemm` sizes and
    two `Peak` variants it never reads. Now
    `(Peak|Sgemm|OpenBLAS)/(avx512|n=2048)`, where the parentheses are
    load-bearing: they suppress both splits and make the `|`s ordinary regexp
    alternation inside one two-element pattern. The audit of every other filter in
    the repo found no second instance. The failure message is also split in two,
    because "no result row to divide by" and "no reference on this host" had the
    same wording and only the first is a defect in the gate.
  - **The coretype sweep read the theoretical-peak provenance line as a benchmark
    rate** (issue #33), in the same run. Taking the maximum over every field
    followed by `GFLOP/s` on every line picked up
    `keel-bench-peak-formula: avx512: 368.9 GFLOP/s`, which is larger than any
    real rate and identical across candidates: all six tied on every host, the
    winner defaulted to whichever came first, and the sweep reported `+0.0%`
    against `DYNAMIC_ARCH`'s own choice — the 6.7% finding it exists to enforce,
    erased by its own parser. It now reads a rate only from a result row whose
    benchmark name is exactly the one requested, and a run that produces no such
    row fails the host as a gate defect instead of degrading to a number. With the
    fix, the gate's own sweep reproduces the finding: vesta `default` → 149.4,
    `Haswell` → **159.5**.
  - **The coretype pin is an intervention justified by a measured effect: no
    effect, no intervention** (ruling on issue #35). Pinning the fastest candidate
    crowns noise on a host where no distinct family is actually faster. Across two
    full gate runs, janus's and antares's winning *request* moved while the achieved
    *family* never did, with margins of 0.0–1.0% against a same-family drift the
    sweep measures at about 0.5% — a winner by a margin inside drift is a winner by
    dice, which is #33's lesson relocated from the parser to the selection layer.
    The sweep now asks whether a family the library did *not* already choose beats
    the one it did by more than this sweep's own noise floor, measured as the largest
    spread between candidates that landed on the same achieved corename. If it does,
    the winner is pinned and the margin is reported against the drift it had to
    clear; if it does not, the reference runs **unpinned**, the way the library runs
    itself, and "no cross-family winner beyond drift" is the recorded finding. On
    both hosts where this changes the outcome the decision is unanimous rather than
    marginal: the best cross-family candidate is slower than the default outright
    (janus Cooperlake 187.70 against SkylakeX 194.70; antares SkylakeX 298.10 against
    Cooperlake 298.60), so the old winner was an alias of the default's own family
    drawing a high sample. vesta's genuine cross-family win survives on both runs
    and is still pinned: Haswell at +5.5% and +6.5% over `DYNAMIC_ARCH`'s Cooperlake,
    against drifts of 1.20 and 0.40 GFLOP/s. The pin verification still runs in the
    unpinned case, where it compares two unpinned invocations and so catches a
    library whose unaided choice is not stable across runs.
  - **Absence is a first-class outcome in every criterion that reads a benchmark**
    (DESIGN.md §5 rule 6, new). `bench_expect` in `scripts/bench.sh` takes the
    benchmarks a criterion declares it will read and asserts a minimum row count
    for each of them — `-count` rows, not benchstat's minimum of 6 — before the
    criterion reads any of them. `bench_stat` printing nothing was previously
    indistinguishable at every call site from a benchmark that ran and reported
    nothing under the unit asked for, and each caller invented its own reading of
    empty; that indistinguishability is where #32 lived for the whole of P3. The
    three states are reported separately rather than collapsed into "missing",
    because a filter that did not select the benchmark, a run that died partway,
    and a benchmark reporting a metric the gate does not read have different
    causes. Three call sites in `gate-p3.sh`: criterion 5, criterion 6 — and the
    kernel sentinel, which is the one that would have survived audits. With
    `Peak/avx512` absent it reported *"no bounded percent-of-peak for any shipped
    shape"*: a red that blames the shapes for the absence of their denominator,
    inside the criterion that carries P2's floor forward, and believable enough to
    be read as a real result. Criterion 6 now declares the peak as well, where an
    absent peak used to reach `p3_denominator` as `peak=0`, silently reverting an
    issue-bound host to plain OpenBLAS — the strict direction, so never flattering,
    but a ruled denominator replaced on the strength of a measurement never taken.
  - **A non-discriminating coretype sweep fails instead of crowning a winner by
    candidate order** (issue #33's signature, mechanized). A sweep exists to
    discriminate, so distinct kernel families measuring an identical rate means the
    instrument is broken by construction: variance too low is as diagnostic as
    variance too high. The test is over distinct *achieved* corenames, not over
    candidates, because candidates that alias to one family are supposed to agree —
    vesta answers both `Cooperlake` and `SapphireRapids` with `corename=Cooperlake`,
    and counting candidates would fire on that and punish OpenBLAS for correctly
    reporting that two names are one family. Counting families also makes the check
    indifferent to the candidate list's composition. The halves are disjoint: the
    pin verification asks whether a request took, this asks whether the instrument
    can tell the families apart. Exact equality is the threshold and the
    measurements support it — two invocations of one family on vesta read 150.60 and
    149.80, so real rates of the same silicon do not tie to two decimals.
  - **DESIGN.md §5 rule 5 amended to match the ruling already applied to §4/P3**:
    *every* measuring host under the `performance` governor, asserted per host and
    re-read at the moment of measurement. The vicarious wording ("at least one
    measuring host") had survived in the source clause while its replacement was
    enforced downstream, so the gate cited a rule that contradicted its own
    behaviour.
  It also carries P2 forward: the spill/call/bounds-check audit re-runs on every
  gate from here on, because packing and edge handling are exactly what would
  break those properties, and P2's throughput floor is re-checked on a sentinel
  host so a K-loop that P3 made fatter is noticed (issue #19; janus, where
  instruction count binds). An unconfigured sentinel means *every* host is one:
  missing configuration costs time, never coverage.
- **The percent-of-peak denominator is now measured per host, not derived**
  (DESIGN.md §4/P2, issue #11). The formula remains as a printed cross-check.
  First measurements, single core, float32: Zen 4 (7950X3D) 165.6 GFLOP/s avx512
  against 165.5 avx2 — a width ratio of 1.00×, which is the double-pumped
  256-bit datapath measured directly rather than looked up; Skylake-X (i9-9960X)
  215.9 against 101.8, ratio 2.12×; Zen 5 (Ryzen AI MAX+ 395) 327.8 against
  164.0, ratio 2.00×, settling the open question about that part's datapath
  width. The formula overstates Zen 4 by 2.23× (the double-pump) and Skylake-X by
  1.30× (the AVX-512 frequency license, visible as an implied 3.37 GHz against a
  4.4 GHz max clock); on Zen 5 it lands within 1.01×. Unlike the L1 ratios, these
  reproduce to within 0.4% between runs — a register-only kernel has no cache or
  placement to be lucky about.
- **Gate benchmarks: `-count=10 -benchtime=1s`, benchstat medians, thresholds
  cleared net of CI** (DESIGN.md §5 rule 5, issue #14). `-benchtime=3x` is now
  for smoke runs only.
- **P1's Sdot ratios are re-derived under that methodology and supersede the
  first ones**, which came from `-benchtime=3x -count=5` reduced by
  min-of-samples. Net of CI, over three gate runs: 8.71×/7.18×/8.57× on Zen 4,
  7.55×/7.48×/7.53× on Skylake-X, 9.14×/8.96×/8.79× on Zen 5 (**the Zen 5 trio is
  unprovenanced — no archived log contains `8.96×`, and `performance`-era readings
  of the same quantity run 9.13×–10.37×; see #79 and the provenance note in
  `docs/hosts.md`**). The old numbers —
  4.28×, 5.91×, 4.09× — were not merely noisy but biased low by roughly 2×: three
  iterations of a 4096-element kernel measure cold caches and frequency ramp,
  both of which cost the vector path proportionally more. The remaining
  run-to-run drift on Zen 4 — bimodal, two runs within 1.6% and one 17% below
  them, invisible to a within-run confidence interval — is recorded in
  `docs/hosts.md` as an open question rather than averaged away: core placement
  across that part's two CCDs is the leading hypothesis (bimodality fits it and
  not thermal drift), and pinning changes what the measurement means.
- `docs/toolchain-notes.md` T9's flag list was wrong and is corrected: it cited
  `-gcflags=-N=0`, which is the *default* (`-N` is a boolean), and therefore claimed
  nothing. `-N` does remove the anchor NOPs, and is not a workaround — with
  optimizations off each statement's values go to the stack, so real instructions
  carry the caller's own positions and no anchor is needed. Found while building the
  standalone repro for the upstream filing, which is the point of building one.
- `oracle.Tolerance` gained an underflow floor term: `C·f(n)·(eps32·scale +
  eta32/2)`. Rounding error is only relatively bounded above the smallest
  representable magnitude; a float32 dot product of ~1e-25 elements has products
  below the smallest subnormal, returns exactly 0, and is *correctly rounded*
  while the float64 oracle says 6e-49. The relative-only model called that a
  failure. The new term is ~1e-44 for realistic n, thirty orders of magnitude
  below anything the relative term admits, so it cannot mask a real error.
- `scripts/remote.sh`: arguments are `printf %q`-quoted before crossing ssh.
  ssh concatenates its command words and hands the result to a remote shell, so
  `-test.bench='A|B'` was being parsed as a pipeline on the far side. It failed
  loudly here; the same expansion on a glob or a `$` would have quietly changed
  what got measured.
- `vec.ScalarMax`/`ScalarMin`: the operand-order claim for NaN and signed zero
  is now **verified on hardware** rather than marked UNVERIFIED. `x.Max(y)`
  returns `y` for NaN and for `max(±0, ∓0)`, as the spec already said —
  confirmed bit-exactly on Zen 4, Zen 5 and Skylake-X — three independent implementations
  of `VMAXPS`. Disassembly could not settle this; only execution could.
- Gate P0 is green. All 14 shim ops agree bit-exactly across scalar, AVX2 and
  AVX-512 on all three amd64 hosts, and both FMA wrappers lower to a single
  `VFMADD213PS`.
- Gate P1 is green. All six Level-1 routines match the float64 oracle on scalar,
  AVX2 and AVX-512 on all three amd64 hosts, pass again with dispatch forced to
  scalar on machines that have AVX-512, and clear the ≥4× Sdot floor at n=4096
  on every host, net of benchstat's confidence interval: 8.57× on Zen 4 (Ryzen 9
  7950X3D), 7.53× on Skylake-X (i9-9960X), 8.79× on Zen 5 (Ryzen AI MAX+ 395),
  with at least one host clearing it under the `performance` governor. (**The Zen 5
  figure is unprovenanced and reads 9–10% low against `performance`-era
  measurements of the same quantity; #79, and the provenance note in
  `docs/hosts.md`. It is marked rather than restated — the gate it cleared, it
  cleared.**)
- **Gate P2 was RED on the flat 55%-of-peak floor, and P2 is a go/no-go rather
  than a hurdle, so work stopped there and the decision went to Scott (issue
  #19).** Every compile-time criterion passed on both shipped shapes — 0
  accumulator spills, 0 calls and 0 surviving bounds checks in the steady-state
  K-loop, `ssa.html` archived for each, all three peak kernels register-only — and
  correctness passed on all three amd64 hosts with the AVX-512 tile exercised on
  each. The single failing line was the floor on janus.local (Skylake-X, i9-9960X):
  46.1%, 46.1% net of CI, reproducing at 46.0% on an independent run. Nothing was
  relaxed to change the colour: no shape added or removed, no threshold moved, no
  host dropped, no assembly written.
- **The P2 throughput floor now has a class-dependent denominator, and the gate is
  green** (DESIGN.md §4/P2 amendment, ruling on issue #19). One written rule, not
  two rules and a wink: an **FMA-bound** host keeps the flat ≥55% of measured peak;
  an **issue-bound** host is held to **≥90% of its issue roofline**, computed from
  the spill audit's own instruction counts. Classification and floor are the pure
  function `scripts/roofline.sh`, and three properties keep it from being a licence:
  - **Independence.** The ceiling mixes are every mix *except* the shape being
    gated. The first draft included it, which made the 90% floor algebraically
    vacuous: with the shape under test in the ceiling set,
    `attain ≥ 1/cspread ≥ 1/1.10 = 0.909 > 0.90` as an identity, so no host could
    ever fail that criterion. Caught by trying to write a fixture that failed it.
  - **Falsification.** If the shape under test retires *above* the ceiling the other
    mixes set (`attain > 1.0`), the issue-bound hypothesis is disproved by its own
    data and the host reverts to the flat floor. This is what returns antares
    (Zen 5) to FMA-bound: its mixes converge to 1.091× but 4×32 retires at 158.5%
    of the 39.3% roofline they imply.
  - **Bounded leniency.** The register-only peak kernel is always in the ceiling set,
    pinning `maxᵢ p_i ≥ 2.25`, and a shape more than 5% above the 115-shape sweep's
    best 4.438 insns/FMA is refused a roofline outright — so a kernel cannot pad
    itself into a lower bar, and the effective floor can never fall below
    `0.90 × 2.25 / 4.659` = **43.5%** of measured peak.

  It also **ratchets rather than expires**, which is stronger than the
  "self-retiring" property first claimed for it (that claim was false — the
  arithmetic shows janus stays issue-bound after the fix). The floor is
  `0.90 × maxᵢ p_i / I_b`, monotone in the gated shape's instruction count, so
  fixing the lowering *tightens* the gate: with T9+T12 landed, janus's roofline is
  78.3% and its required floor 70.4% — above the 55% it replaced.
- **Gate P2 is green on all three amd64 hosts under that one rule**
  (`bash scripts/gate-p2.sh`, exit 0): vesta.local FMA-bound at 96.6% of measured
  peak (96.6% net of CI), antares.local FMA-bound at 64.2% (62.3% net of CI) after
  its issue-bound hypothesis is falsified, janus.local issue-bound at 46.0% of peak
  = 94.6% of its 48.6% issue roofline. The performance-governor requirement is met
  by janus. The compile-time criteria are unchanged and still binding, and the
  15 verdict fixtures run before any benchmark. janus.local becomes the standing
  regression sentinel for P3: it is the host where instruction count binds, so it is
  the host that notices when a shape gets fatter.
- **The advertised dispatch chain is now stated per level: Level 1
  `avx512→avx2→scalar`, Level 3 `avx512→scalar`** (`DESIGN.md` §3 and §4/P5,
  `dispatch.go`, ruling on issue #40). Runtime behaviour is unchanged — there has
  never been an AVX2 microkernel and `KEEL_FORCE=avx2` has always acted as a
  *ceiling* at Level 3, reporting `scalar` — but the documentation claimed a
  three-rung chain at both levels, and the missing rung was the discrepancy. The
  narrowing goes in the strict direction: no gate check was deleted. `gate-p5.sh`
  now **requires** that forcing a Level-1-only rung yields a scalar microkernel and
  that `kern=` never names `avx2`, so a claim that grows back silently fails the
  gate that found it. The coverage marker gained a field for the same reason
  (`keel-p5-dispatch: l1=avx512,avx2,scalar kern=avx512,scalar`): a ruling that
  cannot be stated is one the next session re-litigates. Level 1 keeps its AVX2
  path, which is measured and has been gated since P1. The AVX2 microkernel is
  **deferred with its unblocking condition named** rather than dropped — an
  AVX2-native evidentiary host, since `KEEL_FORCE=avx2` on an AVX-512 machine
  establishes correctness and says nothing about performance on a part that lacks
  AVX-512. Debt with a trigger, not a wish.
- **P5's `-race` criterion is not amendable to exclude `checkptr`, and #42 is
  merged into #22 as an admissibility condition** (`DESIGN.md` §4/P5, ruled
  2026-08-12). A library that fatals under `go test -race` inside a 1×1 `Sgemv`
  through its public API is unshippable to a Go audience: race-clean is table
  stakes, and checkptr-clean is what race-clean means for code holding `unsafe`.
  Excluding the pointer checker would certify keel safe minus the instrument that
  checks. So the criterion stands unamended and the T17 workaround lands inside
  #22's campaign rather than as a point patch — **stage 1 measures masked-partial
  against zero-padded-panel among `checkptr`-clean implementations only.** A faster
  variant that fatals under the pointer checker is disqualified, not ranked:
  admissibility first, then speed.
- **janus.local keeps its gate-host and sentinel roles through P5** (confirmed
  2026-08-12; `docs/hosts.md`, `DESIGN.md` §4/P5). The question was whether a host
  sitting at 31.9% of measured peak should go on certifying phases. It should: it is
  the only Intel part, the only issue-bound one, and the machine the roofline
  amendment's standing task names as the thing to re-measure when the lowering
  improves. Because that amendment ratchets — the floor is monotone non-increasing
  in the gated shape's instruction count — the exception tightens automatically
  instead of granting a permanent dispensation. A host whose low number is
  *explained*, by a model that gets stricter as the explanation goes away, carries
  more information per run than a host that simply passes.
- Filed upstream as **[golang/go#80856](https://github.com/golang/go/issues/80856)**:
  `archsimd`'s partial slice load/store are not `checkptr`-safe (toolchain note
  T17, issue #42). This one is a correctness-contract violation rather than a
  performance finding like #17/#18 — the helpers manufacture a full-width
  `*[16]float32` from a short slice, it reproduces without the race detector, and
  it is demonstrated through a public API at the minimum input. #42 carries the new
  `standing-task` label and no milestone, the same pattern as #17/#18: when
  upstream's helpers go `checkptr`-clean, keel's workaround retires.
- **`docs/hosts.md` no longer explains antares's 43% confidence interval with a
  governor that is not set** (issue #44). The host was in `powersave` when that
  interval was measured on 2026-08-11 and has been in `performance` since the
  OpenBLAS provisioning campaign — every gate run from 2026-08-12 reads
  `governor=performance` from the machine itself. Removing the suspect settles
  nothing, so the paragraph now records a question instead of an answer: the
  variance was **re-measured under the asserted governor** the same day, three runs,
  same benchmark and methodology. The 43% interval did not reproduce — the widest
  scalar interval was 14% — but three draws cannot exclude a rare event, so the
  result is a *bound* (≤14% in three runs), not an absolution. What the runs did
  establish is a location: the scalar median carried 3–14% intervals while the
  AVX-512 median carried 0–1% in the same runs on the same host, an order of
  magnitude apart. That comparison is differential — two kernels co-measured under
  identical conditions — so it is a property of one kernel rather than ambient
  machine noise. **Location identified, mechanism open:** "the kernel that touches
  memory" does not discriminate, because both kernels read the same two arrays over
  the same 32 KB working set; what differs is runtime per op (~10×), frequency
  sensitivity and issue character. Two candidate classes, neither favoured by the
  data — clock-domain exposure (a loop spanning 10× the wall time samples 10× more
  boost and thermal wander, which on a mobile APU is at least as available as any
  cache-hierarchy story) and memory-path behaviour at that working set. So the
  governor is not credited with fixing what it was never shown to cause, and the 43%
  now has a bound and a location but **no identified cause** — recorded on issue #15
  beside vesta's bimodal ratio, which is the same character of finding. A stale explanation is worse than a missing one, because the
  next unstable measurement here would be attributed to a setting nobody set. The
  dated measurement records are unchanged and now say they are dated; the P1 table
  notes that antares's rows predate the change and are not restated as current.
- **Stage 3 gets an evidentiary bare-metal host, and the ≥6× floor does not move**
  (`DESIGN.md` §4/P5, `docs/hosts.md`, ruling on issue #12). Cloud hosts split into
  two classes with different licences: **evidentiary** (`c7i.metal` for a true
  512-bit server datapath, `c7a.metal` for Zen 4 server) may produce published
  scaling curves, and is metal-only because on shared tenancy a noisy neighbour and
  an invisible frequency ceiling are indistinguishable from a bad loop nest;
  **correctness** (spot, any µarch) may widen the differential sweep, where a noisy
  neighbour cannot change a bit-exact answer. The motive is that 6× at 8 threads on
  a 16-core client part does not locate where the nest stops scaling — packing-buffer
  contention invisible at 16 threads is the whole show at 64. Adding hosts makes the
  gate **stricter**: the floor stays ≥6× at 8 threads on 4096³, every host must clear
  it, and the wider 16/32/64-thread curve is reported beside the judged number rather
  than becoming a threshold invented after seeing the data. Gates keep running on the
  three local hosts, and the metal hosts are launched only when stage 2's nest exists
  to be measured.
- Issue bookkeeping brought current, which finishes the ruling from two sessions
  back. The four pre-existing open issues outside P5 were triaged against a stated
  rule — ruled-but-unlanded one-liner, deferred-to-a-P5-measurement,
  upstream-dependent standing task, or overtaken by events — and #34, #10, #8 and
  #12 all moved to the P5 milestone with their sorting recorded on each. With every
  earlier milestone empty of open issues, **milestones P0, P1, P2, P3 and P4 are
  closed**; P5 is the only open one.

### Removed

- **`remote_build_test_or_fail` replaces the eleven guarded cross-compile blocks in gate-p0 through gate-p5** (D1),
  beside the `remote_build_test` it wraps. Both messages are parameters: gate-p5's two say *"cross-compile of **the**
  linux/amd64 …"* where p0–p4 omit the article, so normalising either would have moved a gate's output. The first
  survey said *ten* — it enumerated `gate-p1.sh` through `gate-p5.sh` and never looked at gate-p0. **−34 lines in
  `scripts/`**, with gate-p0's four five-line verdict blocks collapsed to the one-liner the other five gates already
  use: 30 of 30 arms byte-identical, and gate-p0's whole 43-line output unchanged live but for its disk reading.
- **`gate_verdict` replaces the verdict tail of all six gates** (D1), in `scripts/remote.sh` beside the four verdict
  helpers it belongs with. gate-p2's go/no-go tail and p2/p3's withhold wording are parameters, so no gate's output
  text moves on any path reachable today — proven over FAIL × stamp × flag, 16 identical and 12 intended. **−19
  lines in `scripts/`.**
- **`gate_tmpdir` replaces the six scratch paths and the cleanup trap in gate-p1 through gate-p5** (D1); each gate's
  own tail (`AUDITKERN`, `SWEEPLOG`, `ALTCSV`, `KERNBIN`, …) stays where it was. **−7 lines in `scripts/`** — 35
  lines of duplication out, most of it back as the measurement the fix rests on.
- **`assert_kern_audit_drift` replaces the registry-drift check in gate-p3 and gate-p4** (D1), whose executable
  lines were byte-identical; the fail message's trailing clause is a parameter, so neither gate's output text moves.
  **−10 lines in `scripts/`**, and the caller-visible `DRIFT_CHECKED` is now documented rather than incidental.
- **`test_verdict` replaces eight copies of the pass/fail/paste-the-tail triple** around every gate's `go test` run,
  in all six gates (D1). The phrase is a parameter, not a normalisation: gate-p5 says *"every test passes"* where
  p3/p4 say *"all tests pass"*, and editing that would have changed three gates' output. **−25 lines in `scripts/`.**
- **Seven byte-identical gate helpers now live in `scripts/gate-lib.sh`** (Workstream D1's lift): `require_bench`,
  `audit_ipf`, `audit_ipf_tile`, `field`, `marker`, `marker_all`, `set_has`, out of gate-p2/p3/p4/p5. The comments
  diverged only in bookkeeping about the duplication itself, a tax quadratic in the copies. **−69 lines in
  `scripts/`**, 1.63× → 1.62×.
- **One `marker_row` replaces five copies of one awk, and the flop-count pair is shared** (Workstream D1):
  `p4_line`/`p5_line`/`bench_line` plus two *inlined* copies — one of them in gate-p4 three hundred lines below
  gate-p4's own helper — and `flops_expect`/`flops_formula`, of which gate-p5's was already a strict superset.
  **−81 lines in `scripts/`**, 1.62× → 1.61×.
- **The boost apparatus is retired and gate-p5 can measure the fleet again** (#66, ruled 2026-08-17 —
  *"the cloud does not have that"*): `remote_boost`/`remote_boost_set`, the second boost-on pass, and the
  precondition that refused every guest are gone; criterion 9 now reads the one sweep there is. The 2026-08-15
  finding stands and the handicap is disclosed beside the ratio instead of removed. **−167 lines in `scripts/`.**
- **Citation *pinning* is retired; resolution stays.** `docs/citation-targets.txt` → `docs/citation-externals.txt`
  (declarations only); `DESIGN.md` §5 rule 9 and §7 amended, because they mandated the instrument; control `T3`
  removed with its ordinal left vacant. **−182 lines in `scripts/`** — not the plan's −600 — and 1.64× → 1.61×.

### Fixed

- **A detached run inherited its host list from a daemon older than the run, and measured the wrong machine.**
  `tmux new-session` seeds a session from the tmux *server's* environment, and that server held
  `KEEL_REMOTE_HOSTS=antares` from an earlier single-host session. `remote.sh:35` gives that variable precedence
  over `.keel-hosts`, so the first release run (`release-a-d88486a`) launched `c5n.18xlarge` + `c7a.48xlarge` +
  `c8a.48xlarge` at a combined $24.0873/hr, wrote a correct `.keel-hosts` naming all three, and then benchmarked a
  lab box: `37 PASS / 0 FAIL / 1 UNMEASURED / 4 BASELINE`, `gate-p5: RED`. **RED with zero FAILs was the tell** —
  the UNMEASURED reads "all 1 configured host(s)" where three were paid for. The fleet then idled to its 8h TTL,
  bounding the spend at $192.70 for zero AWS evidence; the exact figure is unverifiable because terminated
  instances have aged out of the EC2 API. The same comment block diagnosed this mechanism correctly on 2026-08-28
  and fixed **half** of it: an override the caller *sets* is dropped unless that call starts the server. The
  complement is a separate defect — a variable the caller does *not* set is injected by the server and outranks the
  run's own configuration — and it is the worse one, because the dropped-override branch fails toward the
  configured fleet while this one fails toward whatever was last measured. The runner now clears the whole
  `KEEL_`/`BENCH_` namespace before re-exporting the carried set, so `build/<name>.cmd` is a complete statement of
  the environment and not just of the deltas. Driven in both directions against the real stale value before it was
  cleared: bare `tmux new-session` reproduced `KRH=[antares]` / `hosts=[antares]`, and the fixed launcher gave
  `KRH=[unset]` / `hosts=[keel-zen5 keel-zen4 keel-skx]`. Scope stated rather than implied (§5 rule 12): `AWS_*` is
  deliberately *not* cleared, since `aws-fleet.sh:37` pins `AWS_PROFILE` itself and nothing there selects what gets
  measured. The stale global was also removed with `tmux set-environment -gu`, but the launcher fix is the durable
  half — the server outlives any one cleanup.
- **The provisioner installed a toolchain that cannot build the tree, and read an existing one as "new enough"**
  (`scripts/provision-openblas.sh`). Its default was `go1.26.5` and its floor `GO_MIN_MINOR=26`, both correct when
  1.26 was the first release with the simd experiment and both stale since T23's rename: from `fed1e70` the tree
  does not compile on 1.26 at all. The two arms that care are the ones a *host's own* toolchain compiles, because
  cgo forbids cross-building them — gate-p5's `-race` leg and gate-p3's openblas-tagged harness. Both died on
  antares in `p5-preflight-1689d0b`, the second as `cannot use bp[0:16] (value of type []float32) as *[16]float32
  value in argument to archsimd.LoadFloat32x16`, which took gate-p3 RED, gate-p4 RED with it, and gate-p5 RED.
  The floor was the sharper half: gating on the minor only, a host already carrying 1.26.5 read as new enough and
  was *linked rather than upgraded*, so the provisioner would have called a host ready for a harness it cannot
  build. Default now `go1.27.0` (stable, published for linux/amd64) and floor `27`. Verified in situ under
  `--check`, which now says `go on the ssh PATH is go1.26.5, and the harness needs go1.27.0 or newer` and predicts
  gate-p3 will fail those hosts by name — the condition caught before a fleet run instead of during one. Then
  discharged on the fleet (#70 authorized): `antares` and `janus` read back `go1.27.0` on a fresh connection rather
  than off the install's own output, digest matched against `$KEEL_GO_SHA256` as an independent pin, `go1.27rc3`
  removed rather than left beside, and `verify` natively built the very harness that had failed. `vesta` answered
  neither name, so it is `unmeasured` — two read-backs of three, not three.
- **A named P5 criterion could never render a verdict, and the condition was already in the tree as a footnote.**
  With `go1.27.0` installed the `-race` leg finally *built*, and then failed three `tools/shapegen` tests
  identically on both hosts: `locating the repo root: exit status 128`. `repoRoot()` shelled out to `git rev-parse
  --show-toplevel`, and that leg ships the tree by `git archive HEAD`, which has no `.git`. The same failure was
  measured the same day in CI's floor arm and recorded at `ci.yml:71` as "the transport, not the toolchain" —
  true, and shelved there as an oddity awaiting anyone who reproduced the floor off an export. What that missed is
  that a *second* arm ships by the same transport, so the cost was not an odd red but a criterion that could never
  be measured on any benchmark host. `repoRoot` now walks up for `go.mod`, which its own comment always named:
  `git rev-parse` returns the *repository* root, a different thing that coincides with the module root only in a
  checkout. Everything these tests read is present in the export; only the lookup wasn't. Driven in all three
  states before landing — reproduced in an extracted export (2 FAIL), all three pass there afterwards including
  the audit test that writes under the export's own `internal/vec`, and the not-found branch driven from `/tmp`
  (`shapegen: no go.mod in /tmp or any parent`, rc=2). Two failures there against three here is arm-dependent and
  not a miscount: the floor arm sets no `GOEXPERIMENT`, so it does not build `audit_simd_test.go`.
- **The candidate files accumulated across runs, so landing one could pin a registry row to a contaminated
  archive.** `gate-p5` named them by revision alone and `baseline_candidate` appends, so two runs at one rev piled
  into one file: after `p5-clean-b5cef4f` the witness file held three rows from a clean run beside two from a run
  whose hosts had been sshed into mid-measurement, with duplicate `(cpu_model, era)` keys and nothing in the file
  distinguishing them. Column 6 is the archive a witness is recomputable from, so the reviewed-commit safeguard
  held only for a reviewer who remembered which timestamp was clean. Now `-<rev>-<RUN_STAMP>`. The accumulation was
  *already known*: `exercise-baseline.sh` documented it and `rm -f`'d the previous pass's files, which protected the
  synthetic path and left the production path — the one whose rows get landed — unprotected. That workaround is
  deleted rather than explained. Driven with a negative control: two hosts in one run still append to one file
  (2 rows), a second run writes its own (1 row, no cross-run archive leak), and the old unstamped name reproduces
  the defect at 2 rows in 1 file.
- **The BASELINE summary line told the operator to land the witness rows, and following it manufactures the FAIL
  it was written to prevent.** It names only `witness-candidates`, so a reader who lands those and stops has
  produced exactly the *owing* state — a tracked witness for `(cpu_model, era)` with no baseline row — which
  `gate-p5` renders as an unmet-registration FAIL. Landing both in the same commit is not the escape either: a
  baseline candidate emitted from a single run says so in its own estimator column (`SINGLE DRAW … NOT landable
  as-is (§5 rule 16)`), so a fleet with one archived run in an era can supply neither half. Caught by reading
  `:1219` before landing `p5-clean-b5cef4f`'s five witness rows, which would have converted three green BASELINE
  verdicts into reds and pinned column 6 to a gitignored path. The line now states the both-or-neither rule and
  why one run cannot satisfy it; the rows are not landed.
- **`docs/hosts.md` recorded `vesta` as `unmeasured` for a reason that never happened, and the host had a
  toolchain the whole time.** The sentence said vesta "answered neither `vesta` nor `vesta.local`", so two
  read-backs of three. The log says otherwise: `provision-vesta-b5cef4f.log` reached `vesta.local` on the first
  attempt and read its state correctly — `distro=ubuntu go=none (/usr/local/go=none) … governor=powersave` — and
  then printed `could not read an answer from the terminal` at `confirm "run it?"`, because the run was detached.
  A launcher unable to obtain consent was transcribed as a host unable to answer, which is the instrument
  reporting in the subject's voice; the fix was `--yes` and no network changed. `confirm()` needs no repair — it
  fails closed and its message is accurate — but the interaction of two of our own scripts is now written down:
  `detach.sh` supplies a tmux pane, so `confirm()`'s `: </dev/tty` open *succeeds* and the `read` behind it hits
  EOF, taking the one branch whose wording names a terminal instead of the absence of one. Corrected with the
  present state, verified two ways: `go version go1.27.0 linux/amd64` read back off `vesta.local` on a fresh
  connection, digest `675c26c4…` matched against `$KEEL_GO_SHA256`, and `gate-p5`'s own provenance stamping all
  three `-race` rows `go1.27.0` — three of three.
- **Addressing a lab host by its bare name silently splits one machine into two, and nothing in the tree said
  so.** Measured 2026-08-29: every lab host resolves the bare name to a Tailscale address and the `.local` name to
  a LAN one (`vesta` → `100.82.237.84` vs `vesta.local` → `192.168.6.153`; `janus` and `antares` likewise). The
  cost is not connectivity — both forms reach the machine — it is provenance. The hostname is column 5 of a
  witness row and is interpolated into the archive filename, and `build/witness-candidates-b5cef4f.tsv` is the
  proof: rows 2 and 4 are the same Ryzen AI MAX+ 395 under `antares` and `antares.local`, citing
  `bench-gate-p5-…-antares-…` and `bench-gate-p5-…-antares.local-…`. What keeps this out of correctness is the
  registry key `(cpu_model, era)`, identical on both rows by design — the hostname is "provenance, never a key" —
  but a reader reconciling archives by host counts a fleet member that does not exist. The trust state differs per
  form too: vesta's reimage invalidated the key `known_hosts` holds for its Tailscale address, so the bare form
  warns `REMOTE HOST IDENTIFICATION HAS CHANGED` and survives only because pubkey auth still succeeds. Recorded in
  `docs/hosts.md` beside the `.keel-hosts` configuration it governs. **The measurement above is dated because the
  machine moved under it**: `grep -c vesta /etc/hosts` is 0 today, and reading that as "the `/etc/hosts` pin I
  blamed on `#70` never existed" is the error this bullet nearly shipped as a retraction. `/etc/hosts` has
  `mtime=2026-08-28T22:38:44Z` — *after* that comment — and `/etc/hosts.bak` preserves the quoted line verbatim,
  `100.78.211.16  vesta.local vesta`. So the `#70` attribution was right and is now spent, not withdrawn: the pin
  named **both** forms, which is why `.local` failed too, and the tailnet address moved out from under it
  (`100.78.211.16` → `100.82.237.84`). Reimage and pin are one mechanism in series, not two competing ones. A past
  claim can only be judged against the artifact *as it stood*, and both witnesses to that — an mtime and a `.bak`
  — were adjacent to the file being read.
- **`docs/hosts.md` called the amd64 requirement the toolchain's, and it is keel's.** The row read
  "`simd/archsimd` is amd64-only on go1.26.5", true when written and stale since: on go1.27.0
  `$GOROOT/src/simd/archsimd` ships 9 arm64 files (`ops_arm64.go`, `types_arm64.go`, `slice_gen_arm64.go`, …)
  tagged `goexperiment.simd` alone, and the portable `simd` package two more. What is amd64-only is every backend
  in `internal/vec` (`//go:build goexperiment.simd && amd64`). The distinction matters because it moves where the
  blocker is: a NEON port is a kernel this repo has not written, and the *binding* constraint is neither the
  toolchain nor the kernel but that `CEIL_FRACTION`, `STRSM_FLOOR` and `SYRK_FLOOR` were all derived on amd64, so
  an arm64 host is judged outside its derivation set on every bar at once.
- **The same floor admitted a prerelease, which `#70` rules inadmissible.** `^go1\.` plus `split` on `.` gave
  `go1.27rc3` a minor of 27, and janus and antares carry exactly that version alongside their `/usr/local/go`, so
  the hole had a host to bite. Anchored to digits at both ends; 11 cases exercised against the shipped function,
  `go1.27rc3` / `go1.27rc1` / `go1.30rc1` / `go1.27beta1` all refused, `go1.27.0` / `go1.27.1` / `go1.28.0`
  admitted.
- **`scripts/detach.sh` dropped the caller's environment, so a detached run could measure a different fleet than
  the one it was told to** — and it did: `KEEL_REMOTE_HOSTS=antares scripts/detach.sh run … -- ./scripts/gate-p5.sh`
  ran the gate against a stale `.keel-hosts` and produced a log of `UNMEASURED` against `keel-skx`, a host that no
  longer resolves. That reads as a fleet outage and is a launcher defect. The mechanism makes it worse than a flat
  drop: `tmux new-session` seeds a session from the **server's** environment, so an override arrives only when that
  call is what starts the server, and `exit-empty off` pins the server for the machine's lifetime. Measured both
  branches on 2026-08-28 — no server: `KEEL_REMOTE_HOSTS=[antares]`; server already up: `[unset]` — so the first
  detached run of a host's life honours its overrides and every later one silently does not. The runner script now
  carries an enumerated environment (`PATH`, `GOEXPERIMENT`, `GOMAXPROCS`, `KEEL_*`, `BENCH_*`), which makes
  `build/<name>.cmd` readable as a statement of what was measured, and the launcher echoes the names it carried.
  `PATH` is on the list because it picks the `go` that builds the arms; it matched the server's here only because
  one profile started both. Proven on the branch that failed, with a server already up.
- **`docs/hosts.md` said no toolchain is installed on the remote hosts, and that the gate version-checks the
  compiler whose output runs — both false, the second interestingly so.** There is a second toolchain and gate-p5's
  `-race` arm needs it (`go test -c -race` under `CGO_ENABLED=0` is refused outright), so `janus` and `antares` carry
  `/usr/local/go` = `go1.26.5`; `vesta` did not answer ssh on 2026-08-28 and is `unmeasured`, not assumed. And the
  drift argument ran backwards: the version the gate printed was the **host's**, which is the compiler that produces
  nothing *except* that one arm — so the binary whose provenance mattered was the one no check read. `b0e5b37` is
  what makes the sentence's premise true for the first time.
- **CI's runner may or may not have AVX-512, and it is drawn per run** (`.github/workflows/ci.yml`). Two runs of
  one workflow forty minutes apart read back `avx512 avx2 scalar` (`33225065217`, red — T27) and `avx2 scalar`
  (`33226797681`, green). The consequence outruns the erratum: **the green run never executed the path the red
  run failed on**, so CI green is not evidence for the T27 fix — the fleet is — and a red AVX-512 finding here
  can be converted to green by a retry landing on other silicon. The design is unchanged, because its
  rejected-options paragraph already said "whatever hardware the runner lottery deals"; what was wrong was a flat
  claim about the fleet sitting two paragraphs above an argument assuming the opposite. Three prose sites and the
  Level-3 summary branch now key off the printed availability rows instead of asserting a cause.
- **`gate-p5`'s race criterion called a compile failure "a test failure under instrumentation"**
  (`scripts/gate-p5.sh`). The `-race` arm is built natively on each host — it must be, since `-race` needs cgo and
  `remote_build_test` is `CGO_ENABLED=0` — so it is the one place the *host's* toolchain compiles keel. The port
  made the tree need go1.27's archsimd names while the fleet's `/usr/local/go` is go1.26.5, so from `fed1e70` every
  host takes this path, and it fell to the `else`: a sentence about a test that never ran and instrumentation never
  applied. Now a fourth verdict state, `UNMEASURED` either way, so no judgment changes — the reader's causal story
  does. Exercised on a real `go1.26.5` failure from `janus` plus three controls proving the new arm does not capture
  the neighbouring cases; the marker is deliberately `[build failed]` and not `^go: `, since the real log carries two
  `go: downloading` lines.
- **`Sasum`'s AVX-512 tail returned `-0`, because the ternlog rewrite transposes `AndNot`'s operands**
  (`internal/vec/vec_avx512.go`, `internal/vec/vec_avx2.go`; docs/toolchain-notes T27). `ssa.rewriteTern`
  folds a tree of vector logical ops into one `VPTERNLOGD` and builds the imm8 in `computeTT`, whose
  `sloAndNot` case reads `Args[0]` as the non-negated operand — AMD64's `VPANDND` carries the negated one
  there, so the immediate is the one for `y &^ x`. Abs now spells itself `And` against the complement mask:
  the fused immediate for three ANDs is `0x80`, bit 7 alone, **invariant under every permutation of the
  inputs**, so a pass that transposes them has no wrong answer available. `AndNot` is the only
  non-commutative op in that switch and so the only one exposed. **Two logical ops in one expression is the
  whole trigger** — a lone `AndNot` is left unfused and is correct — and go1.27.0's `LoadFloat32x16Part`
  supplies the second, since `Masked` is an `And`. Three claims here replace a first story that was wrong in
  mechanism ("go1.27.0 swaps `VPANDND`'s operands", refuted by there being no `VPANDND` in the object code at
  all): the bug reproduces identically **under go1.26.5**, so the 1.27 floor exposed it rather than caused
  it; `archsimd`'s own `Float32x16.Abs()` escapes only because the rewrite skips unsigned vectors
  (golang/go#79666, open), i.e. its correctness rests on a second bug; and the AVX2 twin was never wrong,
  because `VPTERNLOGD` is an AVX-512 encoding at every width and keel's AVX2 routines compile under an AVX2
  feature context — so its move is prophylactic and retracts no measurement. Verified in the shipped kernel:
  9 `VPANDND` became 9 `VPANDD` at identical displacements and the single ternlog kept its slot with `$112`
  becoming `$128`, so **no rate is re-measured**. 56 package-runs green afterwards (7 packages × 4 dispatch
  pins × 2 AVX-512 hosts, `avx512` active and asserted), against three failing tests before.
- **keel is ported to the `go1.27.0` `simd/archsimd` API, which the dev host moved to between sessions**
  (`internal/vec/{gemm,vec_avx2,vec_avx512,vec_scalar}*.go`, `tools/shapegen/emit.go`, `internal/block/block.go`;
  docs/toolchain-notes T23 amendment). The load/store surface renames with a **swap** — the slice forms take the
  bare names, the array forms they displace gain an `Array` suffix — so the array-form sites had to be rewritten
  *before* the slice sweep, or the two APIs merge into one name. 51 type errors, the same count T23 measured
  against `rc3`, so the rename table is identical between rc3 and final. keel's own `vec.Load512`/`StorePart512`
  surface keeps its names **and** signatures: the `…Part` wrappers absorb archsimd's new lane count rather than
  passing it on, so nothing outside `internal/vec` changed. **`gemm_amd64.go` is generated and `tools/shapegen`
  emits it**, which T23's file inventory missed — `internal/kern`'s fixed-point test caught the generator
  drifting from its output. Two 1.27 facts recorded and deliberately not acted on: `paFloat32x16` now carries
  `//go:nocheckptr` (CL 761120, merged, which closed golang/go#78413 — the still-open golang/go#80856 is a
  duplicate), and that settles `-d=checkptr` **only**, because golang/go#42880 records that `-race` does *not*
  obey `go:nocheckptr` — so the `-race` half is now predicted still broken, and an AVX-512 host's `-race` run
  stays the decisive branch; and `(Float32x16) Abs()` now exists, retiring the bitcast workaround at #54's
  convenience rather than during a freeze.
- **The vector path's floor is now Go 1.27, and CI is where that is stated** (`.github/workflows/ci.yml`,
  `Makefile`, `DESIGN.md` §4/P0 + §"standing orders" 1, `CLAUDE.md`, `README.md`, `doc.go`,
  `doc-site/{index,limits,troubleshooting}.md`). Because the rename is a *swap*, no source satisfies both
  toolchains: CI still pinned `1.26.x` and so failed the port with the same 51 errors mirrored. `go.mod` cannot
  carry this requirement — `archsimd` ships with the toolchain and is not a module dependency — so CI's pin is
  the tree's only machine-checked statement of which API keel is written against. The scalar path's floor stays
  Go 1.26. Two user-facing docs said the *opposite* as of 2026-08-16 ("use Go 1.26.x for the vector path") and
  are corrected, with the direction change called out, since it has now pointed both ways in twelve days.
- **`make build` cross-builds `GOOS=linux GOARCH=amd64`** (`Makefile`, `CLAUDE.md`). The dev host is
  darwin/arm64, where the build tags exclude `gemm_amd64.go` and both vector backends, so the session-start
  smoke build greened on a tree that compiled nowhere a benchmark runs — which is how the `archsimd` rename
  reached a \$3.888/hr fleet run before it reached a compiler error. Verified by reintroducing one 1.26 name:
  native-only build passes, `make build` fails naming the line.
- **`baseline-test.sh`'s boundary control asserts *unchanged* rather than *empty*** (`scripts/baseline-test.sh`).
  It checked that the tracked registry and witness index hold zero data rows, using emptiness as a proxy for
  "this script did not write them". `f0e9e0b`'s three reviewed baseline rows and one witness row — the
  registration the whole BASELINE-REGISTERED class exists to serve — broke the proxy, gate-p5 fail-closed on
  every bar, and the first registered-baseline verdicts were never computed. Now a `cksum` taken before the
  first fixture and compared after. Both branches driven on purpose: a forged registry row and a
  **single-newline** write to the witness index each turn the run red.
- **CI runs `make lint`, not a hand-copied subset of it** (`.github/workflows/ci.yml`, `scripts/gate-p5.sh`).
  The workflow listed three of the target's four steps and never gained the fourth, `baseline-test.sh`. So
  gate-p5's comment beside that check — "lint runs on every push … and catches a broken reader before a
  \$24/hr fleet renders a bar from it" — was false for the week it mattered, and the bill was the run above.
  The comment is corrected in place rather than deleted, because what made it durable is worth naming: it
  asserted a behaviour of *another file*, where nothing checked that the behaviour existed.

- **A failed 8-thread parse published the previous routine's rate** (`scripts/readme-numbers.sh`, #6). `t1`
  and `t8` were never cleared between rate lines, so a line whose `8 threads` match missed carried the last
  successful routine's figures into the table under the new routine's name — silently, because a stale float
  reads exactly like a measured one. Both are now cleared explicitly and the mismatch refuses the run;
  controlled by mangling one rate line, which fires on all three hosts. Found while auditing a check added in
  the same pass and then deleted as unreachable — `one` and `eight` parse from the same line, so their pools
  are symmetric by construction. The unreachable check was apparatus that could not pay its way, and taking
  it out is what exposed the reachable defect beneath it.
- **The README generator read a verdict line's prefix and not its prose, so a run judged by no bar published
  as clearing bars derived from it** (`scripts/readme-numbers.sh`, #6/#37). Take four's rows are prefixed
  `PASS` and say `NO FRACTION IN FORCE` / `NO FLOOR IN FORCE` in the same sentence — `PASS` only because
  nothing failed, and nothing failed because `44.2%` and `6.067x` were typed *from* that run. New `REPORTED`
  class (prose outranks prefix, one classifier for both verdict shapes) plus a `BASELINE` parse that no longer
  refuses a whole log for want of a ratio no baseline row has. The caption now names both unjudged classes
  separately, keeps one denominator across all four branches, and refuses to headline `0 of 12 clear` for a
  run nothing tested. Positive control: the phrase appears 10x in take four, 0x in the confirmation log.
- **The gates divided by a CI bound above every sample ever taken** (#116, 2026-08-22; DESIGN.md §5 rule 20,
  `docs/rulings.md` rule 20). `bench_ratio_lo` reconstructed a bound as `center × (1 + ci)`, but benchstat's
  median CI is `[x_(r), x_(n+1−r)]` — **both bounds order statistics, so neither can lie outside the data** —
  and `ciFraction` reports the *wider* half-width. keel-zen5's ceiling reached 296.5 GFLOP/s down and 0.5 up;
  mirrored, that made a denominator of **2588** against a sample max of 2295 (2296 over 30 draws). `benchci`
  now emits the honest `lo`/`hi` plus the observed `min`/`max`, and the consumers read the bounds instead of
  reconstructing them. `ciFraction` is **unchanged on purpose**: it is benchstat's display quantity and
  `-verify` still agrees on all 42 cells, so no historical `±%` moved. Direction, per §5 rule 15: the old form
  could only deflate a share (`hi_den ≥ center_den`), so it never manufactured a pass — zen5's ceiling share
  was 42.7% and is 48.25% — which is why it survived, and why the care owed its favorable direction is
  procedural.
- **A zero-width interval can mean the rank window stopped looking, so every reading now prints its range**
  (same ruling). At n=30 the zen5 contaminant **recurred** (1989 is in the sample) yet
  `[x_(10), x_(21)] = [2291, 2291]` reported **±0.00%** over a span of 13.40%: raising `-count` concealed the
  defect it was ordered to resolve. The rank pair marches inward faster than any variance argument — `[2, 9]`
  of 10, `[10, 21]` of 30, `[36, 55]` of 90 — so at fixed 20% contamination the width goes ±21% → ±0% → ±0%.
  `bench_describe` prints `[min, max]` beside every reading and names `RANK-WINDOW-BLIND(span …)`, driven on
  purpose rather than inferred. Unthresholded deliberately, and it moves no verdict. **Sensitivity measured,
  not claimed: the exact-zero trigger names 1 of the 3 blind rows in that log** — `avx2` (span 7.42%) and
  `scalar` (13.34%) sit under *nearly* zero ±0.087% intervals — and the threshold-free widening was refuted in
  the same log, firing on three healthy rows too, since the rank pair excludes an extreme by construction.
  A second finding fell out: the contaminant reaches the **scalar** arm, so it is not an AVX-512 frequency
  artifact. Also repaired en route:
  `clock_series`'s field-count guard would have reported "unbounded" on every host on every run the moment
  `bench_stat` grew a column — found by enumerating its 29 call sites, not by a test.
- **Every gate now prints its own verdict arithmetic** (ruled 2026-08-22 on #6: *"the run that signs v0.1.0
  prints its own arithmetic"*). The previous campaign's headline tally was the operator's grep, disclosed as
  such. The five verdict primitives count, and `gate_verdict` calls `gate_tally` from one site, so all six
  gates gained it at once; a **zero total sets `FAIL`**, since a gate that reached a verdict without rendering
  one adjudicated nothing. Cross-checked against an ANSI-stripped grep of the same gate-p0 log (11/1/4/0/0,
  exact), with `BASELINE`, `REPORTED` and the zero-total anomaly driven separately.
- **The 8-thread compute ceiling under-reads by 10–37%, and it is the harness, not the machine** (#115,
  2026-08-22). `computeArm` re-forks 8 goroutines and joins them every one of `b.N` iterations.
  `BenchmarkT52Hoist` is the same arm with the fork lifted out of the loop — same work, same flops formula,
  `b.N` joins reduced to one — and paired against the shipped arm on each host under each mask it reads
  **1.290 → 1.010** (keel-zen4 spread), **1.599 → 1.014** (keel-zen5 spread), **1.147 → 1.029** and
  **1.157 → 1.007** confined, as multiples of eight times each host's own 1-thread rate. Duty cycle rises to
  **0.97–0.99** on all six arms, so the lost time is workers *parked*: **the spread-mask collapse is a
  fork/join cost, and the spread-vs-confined gap goes from 0.143 to −0.019 and from 0.442 to 0.007.**
  Placement was correct throughout — 8 running threads on 8 distinct masked cpus — and keel-skx, whose two
  masks are byte-identical, is flat in every arm both before and after. `internal/par/par.go:109` forks per
  call too, so the routines share the shape; what differs is op duration, **204 ms for
  `Scale/Sgemm/n=4096/threads=8` against 268 µs for the ceiling**, a factor of 762, which makes a ~60 µs
  fork/join **22% of a ceiling op and 0.03% of a routine op**. Both sides of every judged share pay it and
  only the denominator pays it at a rate that matters — so **this is the cause of the impossible denominator
  below, not a second finding** (§5 rule 14: severity is a function of deployment context). Hoisting the
  shipped arm changes the criterion's denominator, so it is blocked on a ruling.
- **A ceiling below a rate it denominates is now refused, not divided** (#6, 2026-08-22). `gate-p5`'s share
  criterion published three plausible passes over an impossible denominator on `keel-zen4`: the measured
  8-thread ceiling read **461.4 GFLOP/s ± 62.68%** while the three judged 8-thread rates read 675.1, 704.3 and
  662.3 — **146.3%, 152.6% and 143.5% of it** — and the criterion printed 89.8%, 93.7% and 88.1%, because
  `bench_ratio_lo` divides the numerator's *lower* bound by the denominator's *upper* bound. That construction
  is correct for a floor and inverts here into a plausibility generator: the wider the ceiling's interval, the
  more comfortable the impossible share looks. `bench_ceiling_impossible` and `bench_ceiling_refused` compare
  the **points**, deliberately not the intervals — there is no confidence level at which dividing by an
  impossible ceiling becomes a verdict — and the criterion answers `unmeasured`, naming a mismeasured
  denominator rather than a regression. **Host-level and not per-row**, which the second pass is what decides:
  it read 690.85 with `Ssyrk` at 101.6% and `Sgemm`/`Ssymm` at 97.5%/95.8%, and a per-row test would have let
  those two PASS against a denominator their own sibling proves is not a ceiling. Twice lucky is not a method.
  Six fixture arms in `remote-exec-test.sh` §9e carry both passes' own rows, `keel-zen5` from the same run as
  the healthy control, a rate exactly at the ceiling (reached, not impossible), an absent rate, and a zero
  ceiling left to the branch that already owns it. Third member of #110's family — an instrument whose output
  was decided by something other than the quantity it names.
- **The spread mask's shape was computed and then thrown away, because the runner read the mask through a
  command substitution** (`scripts/remote.sh`, found by the era-founding run at `450a783` on all three judged
  hosts). `keel_pin_mask` returns the mask on stdout and records its *shape* — `KEEL_PIN_DOMLIST`,
  `KEEL_PIN_NODEDOMS` — in globals, deliberately, because stdout is what every other caller parses. The
  runner then called it as `KEEL_MASK=$(keel_pin_mask 8)`, and a command substitution is a **subshell**: the
  mask came back and the shape died with the child. Every host measured under a *correct* spread mask
  (`0,8,16,24,32,40,48,56` on both EPYCs, the degenerate `0,1,2,3,4,5,6,7` on `keel-skx`) and recorded
  `doms= nodedoms=`, which `gate-p5`'s new shape criterion correctly called **UNMEASURED on all three** —
  fail-closed, so the cost was host-minutes and not a false verdict. The mask is now a global too
  (`KEEL_PIN_MASK`), cleared on entry with the other two so a refusal cannot leave a stale mask behind, and
  one in-shell call yields all three fields the pin line needs: "mask recorded, shape absent" is no longer
  representable. **The fixtures passed throughout and were never wrong** — they called the selector directly,
  which is the one form that keeps globals, i.e. the form the runner did *not* use; `shape_case` now redirects
  stdout to a file instead and asserts the global mask and the printed mask are the same string. Driven
  end-to-end on `keel-zen4`, `keel-zen5` and `keel-skx` after the fix: all three pin lines carry their shape
  and all three satisfy `distinct(doms) == min(width, nodedoms)` with imbalance 0. The founding run's numbers
  are **discarded rather than salvaged**, though the mask under them was right: reconstructing a shape after
  the fact from a host's topology is exactly what recording it was meant to replace (§5 rule 5).
- **`benchci -verify` failed on every pinned host, because the mask's own provenance line is a CSV field
  with commas in it** (`tools/benchci/main.go`, found on the era-founding run at `be5bb91`). `remote_exec`
  prints `keel-pin: mask=0,1,2,3,4,5,6,7 width=8` *inside* the benchmark block, which makes it a
  configuration line; `benchstat -format=csv` therefore quotes it; and `verifyAgainstBenchstat` split
  benchstat's output on commas, so the line arrived as **eight fields** and was recorded as a data cell named
  `"keel-pin: mask=0` that no summarizer can reproduce. The gate went red for a metadata line, with the
  statistics untouched. **The guard that should have caught it was passing by luck**: every earlier
  comma-bearing configuration line (`keel-bench-clock-mhz`, `keel-bench-peak-method`, the `Ceiling/stream`
  names) happens to split into exactly two fields and was absorbed by a `len < 3` test, so the parser had
  been wrong since it was written and no archive had yet exercised it. The tool **wrote** its CSV with
  `encoding/csv` and **read** benchstat's with `strings.Split`; it now reads with the same package it writes
  with, and a CSV it cannot parse is an error rather than a skip — unparsed input greens exactly like clean
  input. Verified on all three pinned archives and on three `free-placement` archives as a regression
  control; `verify_test.go` asserts the config-line case, the fail-closed case, and both directions of the
  differential, and each was **driven against a reverted parser**, which reproduces the gate log's failure
  line verbatim.
- **The scaling aggregate printed a negative host count — `-1 produced no ratio`** (`scripts/gate-p5.sh`,
  found on the same run). BASELINE is decided per (host, criterion), and keel-skx rendered it on the share
  class while being judged and *missing* the `Strsm` bar in the same run — the first fleet where one host
  splits across classes. The per-host subtraction from the judged denominator counted it anyway, so the host
  sat in two buckets and `SCALE_NOCOVER`, the one term still **derived** rather than counted, went to −1.
  That is #90's second finding recurring at the last place it could: the miss count was moved to a counter
  in 2026-08-16 precisely so derivation could not come back, and this term was missed. Now two counters —
  `SCALE_HOSTS_BASE` reports how many hosts rendered the class, `SCALE_HOSTS_BASEONLY` counts those it was
  the *whole* of the verdict for, and only the latter leaves the denominator. A residual that still goes
  negative now **fails** with the bucket tally printed, rather than being published as a quantity. Driving
  the six fleet shapes through the corrected arithmetic also found a **second, never-reached instance**: a
  host that clears both bars while rendering BASELINE on the README criterion broke the old form identically.
  `gate-p5.sh` has no standing harness, so those shapes were rendered as a session act and not a landed one
  (§5 rule 12). The verdict does not move — `be5bb91` is RED either way; what was wrong was the sentence and
  the denominator under it.
- **The gate told every reader that nothing was pinned, one line above every number the mask shaped**
  (2026-08-22). `804fb75` put the affinity mask in `remote_exec` and added the readback criterion, and left
  `gate-p5`'s per-host provenance line saying *"nothing is pinned either way, placement is the scheduler's"* —
  printed once per host, immediately above that host's ratios, and it would have been printed into the founding
  log of the `pinned8` era. The line now states the hazard the mask removes (`smt=2` means eight goroutines
  *unmasked* could span four physical cores and their siblings) and then the mask that removes it, reading
  `$KEEL_PIN_WIDTH` from `remote.sh` rather than retyping `8`. `docs/gates.md`'s criterion 4 is corrected the same
  way, including the refusal — no `taskset` or no eight-core node is status 121 and nothing measured. Neither site
  says issue #15 is closed, because it is not: the *decision* it asked for was ruled and implemented, and #15
  closes on vesta's rows being re-measured under the mask, since an adoption closes on measurement rather than on
  a ruling. That reasoning is now recorded on #15 itself with the four measured grounds, including the
  **±0.11% pinned against ±14.6% unpinned** probe that is #15's own phenomenon.
- **The exercise driver's own `[synthetic]` stamp defeated the driver's own reader, and the audit then claimed
  coverage it did not have** (found live on the first firing, 2026-08-22). `run_pass` collected the delegated
  gate's log by matching `^ *full output: build/gate-p4-under-p5-…`, but `instrument_exercise` stamps `info`
  lines too, so `[synthetic] ` sits between the indent and the phrase and the anchored pattern matched nothing on
  all three passes. It **failed closed** — the driver said the delegate was uncollected rather than tallying files
  it had never read — and then `stamp_audit`, skipping absent files with `continue`, totalled three parent logs
  and concluded *"all 129 verdict lines carry `[synthetic]` … parent or delegate"*. The disclosure existed three
  screens earlier, once per pass, and the summary line contradicted it. Now: the extractor tolerates any prefix
  before the phrase and walks the chain **two levels** (gate-p5 names gate-p4's log, gate-p4 names gate-p3's),
  because surviving two delegations is precisely what the lifted `export` is for; and the audit prints the file
  count beside the total, asserts the expected nine, and calls a missing log **NO on coverage** rather than
  reporting the surviving lines as clean. Both branches driven against this run's real bytes: nine logs / **420
  verdict lines / 0 unstamped / 0 signed**, and with one log removed it refuses rather than reporting 382.
- **#78's fix rev-stamped the delegated gate logs and left two runs at one rev overwriting each other, which is
  the same defect past its own fix** (found 2026-08-21 from the other end, while writing the exercise driver's
  own `#78` control). `build/gate-p4-under-p5-<rev>.log` and `build/gate-p3-under-p4-<rev>.log` stopped a run at
  one revision from destroying a run at another — and the collision that remained is the *ordinary* case here,
  not a corner: DESIGN.md §4 allows one immediate re-run of a failing throughput sentinel with **both outputs
  archived**, and an instrument exercise runs one gate three times over at one rev. Both paths now carry
  `RUN_STAMP`, a per-process UTC stamp defined once in `scripts/remote.sh` and deliberately **not exported**, so
  a delegated gate stamps its own log with its own process's stamp while a driver that wants one stamp across a
  chain can set it in the environment. `bench_csv`'s identical stamp — landed hours earlier, from the archive
  side of the same defect — now defers to it, so one process's samples and its delegated logs are joinable by
  stamp instead of merely being distinct. **Two independent discoveries of one naming rule** (§5 rule 10): the
  archive path was caught by measurement, this one by writing a control that had to know where a log would land.
- **Two runs at one rev on one host wrote one archive path, so the second overwrote the first's samples**
  (found 2026-08-21 while building the synthetic exercise, measured rather than reasoned). `bench_csv` keyed the
  path on gate, rev, host and a counter — and the counter was per *process*, so it discriminated archives inside
  a run and nothing at all between runs. Two shells sourcing `scripts/bench.sh` and calling the function both
  printed `build/bench-<gate>-804fb75-keel-probe-1.txt`, the second `cp` silently replaced the first's numbers,
  and the gate printed the path as that run's own archive either way. The two things this project most often does
  with one rev are exactly the two that collide: DESIGN.md §4's one-immediate-re-run allowance for a failing
  throughput sentinel says **both outputs archived**, which the naming made impossible, and an instrument exercise
  runs one gate three times over to drive three states. A per-process UTC run stamp now sits between the host and
  the counter; it goes at the **end** because `readme-numbers.sh` reads the rev by offset from `bench-gate-p5-`
  and both `build/bench-gate-p5-*-<host>-*.txt` globs still match, so an appended field costs no reader.
- **The fleet-wide CPU affinity mask was law, doc and measurement era three times over, and no line of code
  applied it** (§5 rule 5, found and implemented 2026-08-21 while building the synthetic exercise of the
  BASELINE-REGISTERED class, ruled on #6).
  `docs/hosts.md` said in the present tense that *"every judged benchmark invocation on every host runs under a
  CPU affinity mask of eight distinct physical cores"*; `scripts/measurement-eras.tsv` had already opened a
  `pinned8` era for it; DESIGN §5 rule 5 carried its falsification condition. `git grep taskset` over the whole
  tree returned four CHANGELOG lines, three doc paragraphs, and two comments saying P2 needs none. **Three
  artifacts asserting a mechanism and zero implementing it is one witness restated three times** (§5 rule 10) —
  grep for a mechanism before publishing the number that depends on it. Now: `remote_exec` — the single launcher
  all 20 remote call sites funnel through, so no gate can deviate — selects a mask of eight first-thread cores
  inside *one* NUMA node for every invocation carrying `-test.bench`, and **refuses rather than falling back**
  (no `taskset`, no sibling lists, or fewer than eight cores in any one node ⇒ status 121, nothing measured),
  because a silent free-placement fallback produces precisely the artifact the era ledger exists to make
  impossible: a free-placement reading wearing a `pinned8` label. Correctness runs (`-test.v`) stay free — a mask
  cannot make a wrong answer right, and pinning them would refuse the test suite on small hosts for nothing
  measured. The mask is printed into the benchmark log immediately before the binary, so it travels into the
  archive with the numbers it shaped, and `gate-p5` reads placement back **twice** — the mask the harness asked
  for (`bench_pin`) against the width Go saw through its own affinity (`bench_gomaxprocs`) — because a requested
  mask that did not take is invisible to the side that requested it. A declined mask and a broken sweep are told
  apart there, being opposite causes of one `unmeasured` (§5 rule 6). Nine selector fixtures in
  `scripts/remote-exec-test.sh` drive shapes no fleet host has, and one of them **created a branch**: the first
  version fell back to `sib=$c` when a topology had no `thread_siblings_list`, which hands back eight cpus that
  may be four hyperthreaded cores — and the `GOMAXPROCS` readback cannot catch that, since the width is 8 either
  way. Distinctness unprovable now abandons the node. Every keel benchmark number published before this commit
  was measured under free placement, which is what the era boundary is for.
- **skx's judged shortfall factors onto the one term nothing excuses, and two published skx figures were a bad
  draw** (#6, 2026-08-21; `build/onethread-decomp-3fceaa9.log`). Pinning the 1-thread arm shows the ladder's
  `keel1` of 59.16 ±15.12% was low: the truth is 66.56 ±0.11%, so skx's 1T efficiency rises 30.83% → **34.72%**
  and its normalized scaling falls 1.406 → **1.248**; the 43.34% share does not move, both inputs being
  8-thread. In the form that reads no 1T keel rate at all, `share = (µkernel/ceil1)(keel8/8µkernel)(8·ceil1/ceil8)`
  = 0.4633 × 0.5822 × 1.6067, only the middle term has headroom — the nest needs +33.4% while the issue-bound
  kernel would need 118.6 GFLOP/s against a ~93 cap. **"Fleet's best parallelizer" is withdrawn**: skx scales
  5.6% better than zen5, not 16%, the rest being credit for its own ceiling droop.
- **`gate-p5.sh` published an unbounded ceiling interval by formatting an infinity.** `bench_stat` returns
  `inf` when benchstat established none, which `printf "%.2f"` renders per-awk rather than as the absent
  measurement it is; it now prints `+/- unbounded`, matching `bench_gflops_lo`'s existing contract that an
  unbounded reading is not measured. Both branches exercised deliberately.
- **`docs/hosts.md` called janus "the only Intel part, the only issue-bound one" — a count stated as a
  permanent property**, false by four Intel hosts and a second issue-bound one without any edit. The clause is
  now dated and followed by bound-class rows for all six measured hosts. Two consequences recorded there:
  issue-boundedness tracks the µarch (janus 46.0% of peak, keel-skx 46.1%, same front end, one instrument, so
  one witness), and **ICX measured fma-bound**, refuting half the premise that launched wave 2.
- **The ceiling-share criterion divided a CI-deducted rate by a bare point estimate, flattering every judged
  share; `CEIL_FRACTION` re-types from 58.5 to 57.8 as a consequence** (#6, 2026-08-21, ratified as a repair
  rather than an amendment). `gate-p5.sh` formed the share as `bench_gflops_lo / bench_gflops` — the numerator
  net of its interval, the denominator's interval simply dropped. `bench.sh`'s own contract forbids exactly
  that, in the docstring of the function used: *"Do not use it to build a ratio"*, with `bench_ratio_lo` named
  as the remedy and the fraction-of-peak case named as the example. **The standard that adjudicates this
  predates the run and is the library's, not the criterion's**, which is why restoring it is a bug fix and not
  a post-hoc rewrite — and the direction seals it, since the correction is strictly stricter. A census of both
  call sites confirms one violation, not a pattern: `gate-p3.sh:1024` divides nothing, compares across two
  CSVs with no shared denominator, and sits inside the stated exemption.
  **The bar re-derives because its input did.** 57.8 was never a free constant but a formula — *lowest judged
  row less 2.6 points* — whose input the defective site computed. Re-deriving all nine judged rows of
  `build/gate-p5-651d1bd.log` through `bench_ratio_lo` moves them down **0.7 to 4.3 points**, in proportion to
  each host's ceiling CI (`zen5` 1.12% → −0.7, `zen4` 2.07% → −1.8, `gnr` 5.11% → −4.2): **the noisiest
  denominator was the most flattered**, which is #86's flip hazard reappearing one gate later. The minimum row
  goes 61.1% → **60.4%** (`keel-zen5` `Ssyrk`), the **argmin does not move**, and 60.4 − 2.6 = **57.8**. The
  bar's *definition* is unchanged; its input honesty improved. Verified before it shipped: the instrument
  reproduces all nine *published* shares to the printed digit (§5 rule 11), so the corrected column is a
  correction and not a second method. The archive re-adjudication is unmoved — **the same 35 of 105 rows
  resolve**, since none lands in the 57.8–58.5 band — and its per-host resolving thresholds re-derive to 75.6%
  (`zen4`-class), 84.3% (`zen5`), 152.1% (`gnr`, still *never*).
  **`CEIL_FRACTION = 58.5` is superseded wherever it still reads as live** — the entries below it in this same
  release, which record the ratification honestly as of that hour, name the value it was ratified at.
- **The gate now prints the ceiling's confidence interval, so this class of correction is never unsizable
  again** (#6, 2026-08-21). Sizing the repair above needed the ceiling CIs, and the gate log had never carried
  them — only raw `go test` benchmark logs did, which no archive policy promises to keep. They survived here by
  luck. This is the `BENCHLOG` law's corollary collecting a second time: **a summary that drops the CI it was
  built from makes its own correction unsizable.** One line, no new machinery: the repair makes the old
  non-positive-ceiling branch unreachable, and deleting it pays for this print exactly, so `scripts/` closes the
  session at **net zero** — which matters because this session lands no routine or kernel to spend against.
- **Four citations of "#22" for the re-measured ceiling row pointed at a closed edges-campaign issue; the work is
  now filed as #113** (2026-08-21). The number was a **task-tracker id transcribed into prose as an issue
  number** — GitHub #22 is *"Edges: measure masked C update against zero-padded panels + temp tile"*, closed, and
  every one of the tree's twenty-odd pre-existing #22 citations correctly means that campaign. Only today's four
  meant the ceiling row. The failure is a known one with a known control, and the control was run and still
  missed: I grepped `#[0-9]+` out of the diff and checked each number against `gh issue view`, but skipped #22 as
  already-known because it had appeared in the ruling I was implementing. **A number arriving from upstream is
  not a verified number** — it inherits whatever produced it, and this one came from a task list. Worse than a
  dangling reference: #22's real subject is edge handling in the single-thread nest, adjacent enough to P5 that
  the citation reads plausibly. #113 states its own scope limit under §5 rule 12 clause (c) — it is
  forward-looking and explicitly does *not* claim to reach the archive.
  **A second instance surfaced hours later, and this time the control caught it.** `gate-p5.sh`'s
  ceiling-share comment said *"#17 re-adjudicates them against this"*; GitHub #17 is the T9 anchor-NOP
  finding, while **task** #17 was "re-adjudicate every historical scaling verdict" — the same
  transcription, pre-existing in the tree rather than authored today. Grepping `#[0-9]+` out of the diff
  and checking each against `gh issue view` is what found it, on a line the repair happened to touch.
  Repointed to **DESIGN.md §4/P5**, where the re-adjudication actually lives as law, since it has no
  issue of its own. The tree's other twenty-two pre-existing `#17` citations all correctly mean T9 (counted
  2026-08-21), so this is one site — and the class is now **two instances with two different task ids**,
  which is what makes it a class rather than an accident.
- **The caption fixing one honesty defect published three rates no instrument re-measures, and `gate-p5`
  criterion 9 caught it on the re-run** (#6, 2026-08-20). `0bbf964` put the ceilings in README's caption, which
  sits *outside* the block criterion 9 re-measures, so they were claims: criterion 9 passed at `651d1bd` and
  failed at `0bbf964` on the same README. The caption now states shares only, says why the rates are absent, and
  names the log that carries them. A **second, latent** instance was found and driven on purpose in the same
  scan — the ceiling-shortfall sentence also printed `GFLOP/s`, unreachable only because `CEIL_FRACTION` ships
  deferred-empty, so ratifying a fraction would have reddened criterion 9 for a reason unrelated to the
  shortfall. Publishing the ceilings in README means first making them re-measured rows: `compute_name` already
  emits `Ceiling/compute/avx512/threads=8`, but the block's row checker hardwires `scale_name`, so it needs a
  resolver keyed on the published benchmark column plus a validating fleet run. Not done here, and not because
  it is wrong.
- **`gate-docs`' apparatus ratio was blind to the largest apparatus directory in the tree, and its comment said
  otherwise** (2026-08-20). `library` is tracked `*.go` less `*_test.go` and `bench/` is 1655 lines of which
  only `openblas.go`'s 106 ever reached it, so **1549 lines of benchmark harness — including the 372-line
  ceiling instrument landed the same day — were counted in neither term**: a cap policed by a reporter that
  cannot see the spending. `bench/` now moves whole, tests included. The reported ratio goes **1.97x → 2.23x**;
  the historical `shell / library` line is unchanged so the published 1.6x series stays comparable. The old
  comment claimed the library side held `bench/` and `internal/spill` "flattering the ratio by ~100 lines" —
  right for `bench/`'s 106, silent about `internal/spill`'s **837**, which are disclosed and deliberately *not*
  moved, since correcting a count is not licence to redraw a boundary in the same commit.
- **Rule 5's new magnitude gate reached a correct verdict through a self-contradicting sentence** (#6,
  2026-08-20). `bench.sh`'s non-declining `printf` was unconditional, so at one resolved step it printed *"1 of
  2 adjacent steps resolves a decline … so the windows are ties"* — a sentence whose own count denies its
  conclusion. Split: no resolved step is a tie, one resolved step is a non-monotone excursion, and rule 5 passes
  it because rule 5 names a *monotone* decline. Message-only; all four branches were driven and every verdict
  word is unchanged. Only `keel-gnr` exposed it — both AMD hosts took the tie path, where the wording was right.
- **`DESIGN.md` §4/P5 stated the retired denominator as the ceiling's formula, one sentence above the text
  contradicting it** (#6, 2026-08-20). The headline read `min(8 × measured 1-thread compute, …)` while its own
  sub-bullet said the compute arm is measured *at 8 threads*. The headline now states the implemented form, and
  the deviation from the ruling's literal text is recorded with its measured consequence: the two forms put
  Granite Rapids at 33% and 87% of its ceiling respectively, a 54-point swing that **inverts the rank ordering
  the bullet's own falsifier depended on**, so it is carried as an open question rather than settled in-tree.
- **`aws-fleet.sh up` could not resume a half-launched fleet, so finishing one cost terminating it**
  (2026-08-20). Roles launch in sequence with `--wait-for-ssh`, and the first judged launch was killed between
  its second and third instance; `up` then refused to run at all while any `keel-` instance was alive, leaving
  the only route to a complete fleet the termination of two healthy 48xlarges that were fine. `up` now skips a
  role already running and dies only on a running instance the requested `FLEET` does not name — which is the
  forgotten fleet the guard was written for, and is still fatal. Same argument that made `cmd_wire` idempotent,
  learned the same way, one step earlier in the same script.
- **§5 rule 5's clock check was flagging coin flips as thermal events; it now judges at full precision
  against a floor its own intervals set** (ruling on #6, 2026-08-20). The test asked only `head > middle >
  tail` on the `GFLOP/s` column, whose quantum at 245 is coarser than any decline it ever reported — it
  refused two of four `keel-gnr` triples across a 0.14% total spread, where a random triple is strictly
  decreasing one time in six. It now reads `sec/op` at `tools/benchci`'s full float64 (42× finer from the
  same samples, `docs/toolchain-notes.md` T26) and counts a step only where the two windows' intervals are
  disjoint, floor `(1+cA)/(1-cB)-1`, so no threshold is added and none is tunable. `keel-gnr`'s refused
  triple replays as `stable` (-0.0019% against a 0.2311% floor) and a 2%/3% droop still refuses.

- **The README emitter lost its revision on exactly the runs it is used for.** `readme-numbers.sh` read the sha out of
  `"gate-p4 is green on this commit (<sha>)"` — a sentence `gate-p5.sh:888` prints only on the GREEN branch, while a
  red run prints `(exit N)` instead. So every publication from a red log silently captioned itself rev `unrecorded`,
  and the campaign's whole purpose is publishing runs that carry disclosed shortfalls. The sha now comes off an
  archived sample path (`bench-gate-p5-<rev>-…`), which every run writes regardless of verdict; verified by driving
  both arms — `ce43bca` on the real log, `unrecorded` on a copy with those lines stripped, so the parse is reading the
  source it names rather than matching something else. Net zero lines in `scripts/`. Residual, disclosed rather than
  fixed: a missing rev still publishes the word instead of failing closed, and it is `docs-gen.sh:89` that dies on it,
  one file downstream.
- **Provisioning waits on apt's lock, because the launcher's readiness signal is necessary and not sufficient.**
  `keel-gnr` (c8i.96xlarge) died on `E: Unable to lock directory /var/lib/apt/lists/` with no OpenBLAS, on a
  $17.99/hour host, after `cmd_wire` had reported `cloud-init settled`: `cloud-init status --wait` returned done at
  06:42 and cloud-init (pid 5253) went on running apt until 06:48:24, its last line being spawn's `--command` tmux
  install — so `--command` runs *under* cloud-init, not after it, and the launcher's comment saying otherwise was
  wrong. The gate is on the **lock**, not on cloud-init: three guesses at the holder (esm-cache, unattended-upgrades,
  apt-daily) were each refuted by the journal, and waiting on the lock is right without knowing which. Two fixes were
  measured and rejected first — `DPkg::Lock::Timeout` covers the dpkg frontend lock, not the lists directory, and
  apt-get still fails in under a second with `-o DPkg::Lock::Timeout=300` set; a `flock` reproduction of a held lock
  held nothing at all, because `flock(2)` and apt's `fcntl` record locks are independent lock families. Both arms of
  the new gate were driven on purpose on a live host: it waited 21s where it had failed in 1s, and returned
  EX_TEMPFAIL against a lock held past its cap, so a host that cannot start apt fails by name instead of dying inside it.
- **The `evidentiary` grant arm claimed a size it never checked, and only a positive control on a deliberately wrong type
  could show it.** Its preamble read *"`<type>` is a full-size instance of an approved family"*, but membership in one
  flat list — `KEEL_EVIDENTIARY_SIZES` — is the entire test `host_admission` performs; nothing there evaluates size. The
  list's members *are* full-size, so the sentence was true of every host the arm had ever run on, which is precisely why a
  healthy fleet could never expose it. Driving it on a live `c7a.medium` with that type temporarily admitted printed
  "c7a.medium is a full-size instance" in a real admission preamble. Now the sentence names the check performed —
  admitted *by the allowlist*, whose members are full-size types added one justifying read-back at a time. This is the
  same defect, and the same repair, as the rejection arm one branch down, whose "not a full-size instance of an approved
  family" was a conjunction over two properties the classifier never separates: a verdict must be able to say which of
  its causes fired (§5 rule 6). It is also the first end-to-end exercise of the grant arm and of the
  witness-contradiction arm against a **live launcher record** rather than a fixture — the contradiction arm was driven by
  forging `instance=c7a.48xlarge` onto a real `spawn=…:c7a.medium:ondemand` line, and reached `unknown` ⇒ `unmeasured`
  naming both readings.
- **Bare metal reaches the evidentiary class by a named arm; the default stays restrictive** (#106). `host_admission`
  read the class from `instance=` alone, so bare metal — which has no EC2 identity — fell through the `case` default to
  `correctness`, and a re-admission run would have demoted vesta/janus/antares by a classifier bug rather than by
  evidence. Driven against the pre-fix code on the same line the lab hosts produce: `correctness` before, `evidentiary`
  after. The probe gains a `virt=` field (the `hypervisor` CPU flag, read from `/proc/cpuinfo`'s `flags` line, `?` where
  there is no such line — an absent instrument must not grant what it cannot see), and the class now reads
  `instance=`, `virt=` and `governor=` from one provenance line. Scott's condition, ruled 2026-08-19: **name the class,
  do not widen the default** — a permissive default would trade a false demotion for a false admission and invert the
  allowlist's safety property, so `instance=none` still fails closed unless `virt=metal` *and* `governor=performance`.
  Four arms must refuse and are driven doing it (`remote-exec-test.sh` case 8/8b, 33 checks): bare metal under
  `powersave` (#79's case), bare metal with no cpufreq at all, a `hypervisor`-flagged guest, and the default arm on a
  provenance line with no `virt=` field. The four refusals are checked to name four *distinct* causes, since the
  hardcoded parenthetical this replaces told a bare-metal host it was "not a full-size instance". The governor
  conjunct is now a second derivation of a fact `assert_governor` also derives, so the two are pinned to agree over
  every governor state rather than assumed to (§5 rule 10). Fixture-only, stated: the guest arm has no live host to
  read (the AWS fleet's last guest is retired), so *that a real guest emits `virt=guest`* is an inference from
  `CPUID.1:ECX.31` tested against a line the fixture wrote.
- **`shapegen` reports a candidate dir it cannot delete instead of discarding the error** (#107, following #39). The
  gate's lint criterion caught `defer os.RemoveAll(dir)` at `tools/shapegen/main.go:174` — the same unchecked-cleanup
  defect #39 already fixed in `spill-audit`, reintroduced by a new file. #39's resolution is followed rather than
  re-derived: report on stderr, keep the primary error, never fail an audit over a cleanup, because a `chmod` on a
  scratch dir must not suppress a report that was produced correctly. Driven on purpose rather than inferred from a
  green run — `internal/vec` made unwritable mid-compile, and the branch prints the real path and errno; the healthy
  `-frontier` run still prints `4.625 34 2x32 u=4 broadcast`, exit 0, no dir left behind. `-sweep` calls `audit` once
  per candidate, so the silent form accumulated dot-directories 34 at a time.

- **`rows_per_bench` moved from `retention.sh` to `bench.sh`, beside the `KEEL_BENCH_COUNT` it reads back** (#49). A
  driver that sourced `bench.sh` for the count then could not count its own log — #49's shape a second time, in a
  caller. All nine `bench.sh` consumers now get the read-back, one line lighter in `scripts/`.
- **gate-p4's criterion 7 aggregate was the one fleet aggregate that never got #90's coverage clause.**
  `SYRK_MEASURED` is counted after three `continue` paths, so a host that produced no bounded ratio is absent from
  every counter and the fail and indeterminate lines read fleet-wide over a proper subset — *"1 of 3 gate hosts are
  below the bar (0 cleared, 0 undecidable)"* with two hosts silent. All three now append `fleet_shortfall`, as
  gate-p2's 5b and gate-p3's criterion 6 do.
- **A gate could sign a synthetic run** — #78's forgeable certificate, reachable from the environment with nothing
  edited. The verdict line read each gate's *own* instrument flag, so the four gates without one had no withhold
  branch, and `VERDICT_STAMP` is seeded from the environment: `VERDICT_STAMP='[synthetic] ' bash scripts/gate-p0.sh`
  stamped every criterion line and still signed the run `gate-p0: RED`. `gate_verdict` now decides on the stamp,
  which covers every instrument mode added later with no per-gate branch to forget.
- **Ctrl-C did not stop any gate, and a SIGTERM to gate-p5 deleted its scratch directory and let it keep running.**
  A group SIGINT killed the `go test` child and the gate resumed at rc=0; gate-p5's `EXIT INT TERM` cleanup handler
  removed `$BINDIR` and also resumed. The signal traps now `exit`, and exiting runs the one EXIT trap: rc=130 and
  rc=143, neither resuming. Measured on bash 3.2.57 and 5.3.15 — the claim the old form rested on, that bash skips
  the EXIT trap on an untrapped fatal signal, is false on both.

- **`aws-fleet.sh up` handed over hosts whose own boot-time `apt-get` was still running**, so the first judged
  on-demand campaign died at `Could not get lock /var/lib/apt/lists/lock ... held by process 3085 (apt-get)` —
  the launcher's userdata racing the provisioner. `wire` now waits on `cloud-init status --wait` per host.
- **The spill parser silently moved a function's body onto the one before it** (#99, *not* dormant as filed —
  `type:.eq.[2]interface {}` and `type:.eq.[4]interface {}` are unmatched headers in the audited packages today):
  `^(\S+) STEXT` cannot match a symbol holding a space, which every generic instantiated over a struct shape does.
  Headers now parse in full, an unparsable one is an error rather than a skip, and `Find` resolves a short name
  through an instantiation's type-argument list. Re-audited: gate-p2's `0 vector stack refs` is unchanged.
- **The P4 sweeps' large-size arm never ran a right-side solve, on any host** (#64): `cs[ri%len(cs)]` aliased
  every runner into the front of a corner list that varies `side` slowest, and `max(ri)` is 2 on a scalar host
  and 4 on an AVX-512 one. `cornerFor` spreads them instead, so `Strsm`/`Ssymm` `side=R` and `Ssyrk` `uplo=L`
  now run at n=500 — the only size where the blocked path runs at all. **`Ssyrk` was affected too**, contrary to
  the issue: 4 corners varying `uplo` slowest leave `uplo=L` unreached below 4 runners, and its own comment
  claimed all four were covered. `TestSweepCornerCoverage` asserts the index space spans the leading flag at
  every runner count from 2 to 8, and names the corners a given host does not reach.

- **Fourteen `#23`/`#24` citations named the wrong issue**: they were local task ids, and GitHub #23 (same-host
  OpenBLAS ruling) and #24 (the 2×32/4×32 dispatch class) both exist and are *also* cited correctly in this
  tree, so one number carried two meanings. Clock sites now cite `§5 rule 5 as amended 2026-08-16`, the fleet
  cites #12. `citation-lint` resolves only `§N`, so nothing could have caught this.
- **"No cpufreq interface" and "the governor will not read" shared one verdict**, which §5 rule 5's
  2026-08-16 amendment forbids: the first is a virtualized guest that does not own the knob, the second a
  defect on a host that has it. `remote_probe` now emits three tokens and `assert_governor` has a fifth
  state, `nocpufreq`. It still blocks — the amendment licenses a substitute instrument, not an exemption,
  and that instrument (`BenchmarkPeak` head/middle/tail) was unbuilt as this shipped. Proven by a
  *changed* reading: this dev machine moved `unreadable` → `nocpufreq`, and the other four states are
  driven from synthetic probe lines because no one host can produce them all.
- **gate-p5's `KEEL_FORCE=nonsense` check certified a refusal it never observed.** It reads *nonzero* as PASS,
  so ssh's 255 for a dead host printed PASS; #62 only made the class nameable. `vanished` is now tested first
  there and beside the existing `else` at four more sites, printing UNMEASURED — same verdict, right cause.
- **A non-matching glob aborted the entire remote probe under zsh**, so a host that answered perfectly was
  reported `unreachable`: sshd runs the *login* shell, and `ssh h 'for d in /nope/*; do :; done; echo B'`
  never reaches `echo B`. Every enumeration in `remote_probe` now goes through `find` with a quoted `-name`.
  The fleet's AMIs default to bash, which is why this had not fired — one contributor's host, once, as an
  unattributable UNMEASURED.
- **Seven of the eleven `gate-pN.sh:<line>` citations in other files were already stale, by 4 to 273 lines.**
  Exposed because relocating the headers shifted every line: re-pointing by arithmetic landed one on
  `done <<<"$HOSTS"`, so all eleven were resolved by content instead. No checker follows — a line-existence
  check would have passed on all seven, which is the argument that retired pinning.
- **Adding a record page needed four hand-kept copies of one list, and two failed silently.** `fake_tree` and
  the gitignore-coverage check now derive from `docs-gen.sh`'s own `records()` table; the missing `.gitignore`
  entry staged a symlink to `DESIGN.md` clean, which is how it was found.
- **The pre-public audit's `.local`-hostname enumeration named three files and there are six** (#95).
  Conclusion unchanged — non-routable LAN names, no IP or key — but `DESIGN.md:73`'s `janus.local` is
  hand-written prose, not a pasted log, so the don't-falsify-the-evidence argument never covered it and
  it broke keel's own key-by-CPU-model convention; fixed to `janus`. The first search greened because
  `git grep -E '\.local\b'` matches nothing (`\b` is not POSIX ERE).
- **`remote.sh`'s "no gate defines its own" verdict helpers was a universal claim `gate-docs.sh` falsifies.**
  Scoped to the six gates that source it, with the reason gate-docs is legitimately outside (own vocabulary, no
  host, delegated by nothing, no instrument-exercise mode so no `VERDICT_STAMP` to honour) — and the standing
  precondition recorded at gate-docs' own definition site, where adding an exercise would make the hole real.
- **Four statements in the docs were false against the code**, each fixed rather than filed: `parallel.go`
  and `p5_test.go` claimed every published number was measured at `GOMAXPROCS=1` (README publishes an
  8-thread arm per Level-3 routine), `DESIGN.md` §2's heading said 15 routines over a table summing to 12,
  and `doc.go` promised the two backend prints always agree — on AVX2-only silicon, which is what CI runs,
  it prints `avx2 scalar` (#40). README's undated "currently missed" scaling floor is now dated to the run
  it describes.
- **A kernel emitting `NaN` passed the two tests whose job is to catch a wrong kernel** (#98):
  `math.Abs(got-want) > tol` is false against NaN. Non-finite results are now rejected before the
  magnitude comparison, and a *matched* NaN/`Inf` pair no longer counts as backend agreement. The two
  differential tests stay unexercised on arm64, which has no vector backend to differentiate against.
- **`Snrm2` returned up to 24.78% relative error for small inputs** (#97): gradual underflow leaves a
  *subnormal*, so the `s > 0` rescue guard let a sum that had lost significance take the fast path.
  Now guarded by `s >= n·2^-126`. The AVX-512/AVX2 accumulations of this path are unexercised on arm64.
- **All 25 `[Tn](#tn)` cross-references in `docs/toolchain-notes.md` pointed at anchors that do not
  exist, and the gate's exclusion for them came out in the same commit** (#93). The headings carry their
  titles (`## T1 — simd/archsimd is amd64-only; …`), so both renderers slugify the whole line and the
  anchor that exists is `#t1--simdarchsimd-is-amd64-only-…`. Clicking `T18` in the summary table reloaded
  the page. **Dead in GitHub's rendering of the file too**, since before there was a site — the site build
  is only what surfaced it, which is the render-don't-assume rule reaching prose. Fixed with an explicit
  `<a name="tn" id="tn"></a>` before each heading rather than by rewriting the links, because GitHub and
  the `toc` extension slugify the em dash differently and no single spelling of the link satisfies both;
  `attr_list`'s `{ #t1 }` is unsuitable for the same reason, MkDocs honouring it and GitHub not. The form
  was verified through GitHub's own markdown pipeline before 25 of them were written (`gh api /markdown`
  returns `<a name="user-content-t1" id="user-content-t1">` beside `href="#t1"`, the pair GitHub's
  fragment remapping resolves), and through a site build afterwards.
  **The un-exclusion is the other half of this commit, by ruling:** *"an exclusion that outlives its
  defect becomes a permanent blind spot with a good excuse."* `gate-docs.sh`'s anchor check no longer
  exempts the records pages, and `mkdocs.yml` raises `validation.links.anchors` to `warn` so `strict`
  errors on one. Both branches were driven on purpose before either was trusted: with `warn`, mkdocs
  fails the build; with the setting lowered back to `info`, the gate's grep over the build log is what
  fails. Lowering it does not buy silence, it moves which check goes red.
- **gate-p2's criterion 5b aggregate divided by the hosts that answered and never named the fleet it
  was asked about** (#90), so with `antares.local` unreachable it printed *"every host that produced a
  judgeable throughput reading cleared its floor (2/2)"* — arithmetically true, and reading as
  fleet-wide over two thirds of the fleet. `gate-p2.sh` had no `NHOSTS` at all; both counters are grown
  inside the per-host loop by hosts that answered. **The dead-host exercise found this on its first run,
  in the line it was built to fire**, which is the argument for the exercise: five green P2 runs could
  not have found it, because a complete fleet renders that branch identically either way. The new
  `fleet_shortfall` in `roofline.sh` appends the clause naming how many of the configured hosts produced
  no judgeable reading and what fraction of the fleet the line therefore covers; six fixtures
  (36–41, 70 total) cover it, including the case a healthy fleet drives — complete coverage prints
  *nothing*, so a helper that appended unconditionally would look right until the log that matters. The
  fraction stays over the survivors: they are the honest numerator of what was judged, and the defect
  was the missing statement of how many were asked. **The verdict is untouched**, and the run as a whole
  was never fooled — the absent host tripped three separate criteria, `FAIL=1`, and a real run would
  have printed `gate-p2: RED`. So this was message-level, not a green certificate over a partial fleet.
  gate-p3's OpenBLAS aggregate had the identical shape and was fixed at its own call site in `64a05e1`
  (`OB_NOCOVER`); this is the sibling that fix left standing, and the helper is shared so there cannot
  be a third.
- **And then the verdict itself: a partial fleet now resolves to `UNMEASURED`, and both fleet aggregates
  decide absence in one place** (#90, ruled 2026-08-16 — the open question the entry above left for
  Scott). *"A PASS reading `(2/2)` over a three-host fleet is a message-level truth carrying a
  fleet-level assertion — the criterion's claim is about the fleet, and a fleet with an absent member
  hasn't measured that claim."* §5 rule 6 gives the absent measurement its one available verdict;
  criterion 6's aggregate already spoke this way, and **two aggregates with different absence semantics
  is the divergent-copies defect relocated to the verdict layer** — the thing `fleet_shortfall` had just
  been built to end. So `p3_coverage` is renamed `fleet_coverage` (the phase prefix was the invitation to
  grow a p2 twin) and criterion 5b calls it instead of its own inline three-branch `if`. gate-p2 gained
  the `N_INDET` counter it never had: its indeterminate branch used to `continue` without counting, so
  "the run could not classify this host" and "this host never answered" were one invisible leftover, and
  the new UNMEASURED names them separately. **Not a weakening** — `unmeasured()` sets `FAIL`, so
  UNMEASURED blocks green identically; a partial fleet simply stops being *describable* as a whole one,
  and per-host PASSes stand as measured. Five p2-shaped fixtures added (31–35, 70 total) because the two
  callers feed the function different shapes — p2 excludes an indeterminate host from the measured count
  where criterion 6 includes it, so the same fleet arrives as `3 2 2 0 1` from one and `3 3 2 0 1` from
  the other. All five of criterion 5b's renderings and all nine of the exercise driver's read-back
  outcomes were driven before this landed, by extracting the verdict lines from `gate-p2.sh`'s own bytes
  and feeding them to the read-back — the call-site half is what a fixture cannot reach, and it is where
  the previous fix was verified by reading alone.
  **The branch fired in a real gate** (`build/instrument-exercise-dead-host-73c40c8.log`, 128 lines):
  `antares.local` unreachable, vesta and janus genuinely measuring and clearing their own floors
  (4×32 at 96.7% of measured peak FMA-bound; 2×32 at 94.6% of a 48.6% issue roofline), and criterion 5b
  printed `UNMEASURED … 2 of 3 configured hosts cleared it and none measured below it, but 1 produced no
  floor verdict (0 indeterminate, 1 with no judgeable reading at all)`. The driver's read-back reports
  **YES** against three independent checks: the configured count matches `.keel-hosts`, at least one host
  is reported absent, and cleared + indeterminate + absent accounts for the whole fleet. Discipline
  audited from the log rather than asserted: **25 of 25 verdict lines stamped `[synthetic]`, zero
  unstamped**, `GREEN`/`RED` never emitted, `VERDICT WITHHELD` the only verdict token, exit 2. Reproducible
  against the first run at `5ade3ff`: vesta 96.6% → 96.7% of peak, janus 94.8% → 94.6% of its roofline.
  **One defect found by that audit and fixed here:** the read-back's own YES prose wrapped so a line
  *began* with `UNMEASURED`, and a verdict-line count over the log read 26 where the gate emitted 25 — the
  driver's commentary about a verdict counted as a verdict. Rewrapped, with the constraint stated: no line
  of a synthetic log's own report may begin with a verdict token. Same class as the `GREEN`/`RED` grep that
  hit the banner's promise text, in the one file where a log being mistakable for certification matters most.

- **The citation lint's pin file was reproducible only on the platform that wrote it**
  (#87, `scripts/citation-lint.sh`, `docs/citation-targets.txt`). Each pin recorded
  `substr(text, 1, 56)` of its rule, and `substr` counts *bytes* in BSD awk and
  *characters* in an awk built against a UTF-8 locale. DESIGN.md §7 rule 2 contains an em
  dash, so the macOS-generated pin held 54 characters where the runner computed 56, and
  the very first CI run that ever reached this script failed T1 — on a tree that was
  clean. `LC_ALL=C` is not the fix: BSD awk is byte-oriented whatever the locale. The lead
  is now the first ten whitespace-separated words, which is the same number under every
  implementation and cannot truncate mid-character and commit an invalid byte to a tracked
  file. Checked by computing all 17 leads under both semantics and diffing them: identical,
  where the old form differed on the em-dash line. Only BSD awk is installed here, so that
  equality is a check against character semantics rather than against a second awk — CI is
  the empirical confirmation.
- **`go test ./...` died with `SIGILL` on any amd64 CPU without AVX-512** (#87,
  `internal/kern/kern_amd64.go`). `vectorKernels()` returns `nil` unless
  `vec.HasAVX512()`; `referenceTiles()`, fourteen lines below it, carried no such guard,
  and `ReferenceTile`'s `Fn` is `vec.Kernel6x32` — `archsimd.Float32x16` throughout, with
  no scalar fallback inside it. `Measured()` is `Kernels()` plus that tile and six test and
  benchmark sites iterate `Measured()`, so those hosts executed EVEX-encoded instructions
  (`instruction bytes: 0x62 0xf1 0xfd 0x48 …`) and crashed in `internal/kern` and
  `internal/block`. **Dispatch was never at risk** — `Kernels()` is guarded, `Preferred`
  cannot rank a tile whose `InsnsPerFMA` is zero, and nothing ships this tile — but the
  suites were unrunnable for anyone on pre-Skylake-X Intel or pre-Zen-4 AMD. The guard goes
  on the registry rather than into six `t.Skip`s: `Measured()`'s contract is *"the kernels
  this host can run"*, and one that hands out an unrunnable kernel is the defect. Typechecked
  for `linux/amd64` under `GOEXPERIMENT=simd`; **it cannot be run here** — the dev host is
  arm64 and all three gate hosts have AVX-512, so CI is the only instrument that can
  confirm it, which is most of the explanation for how it survived three days.
- **`TestP5Determinism` expected a worker count that moved with the host's core count**
  (#87, `p5_test.go`). `Workers` takes a *unit* count and returns
  `min(GOMAXPROCS(0), units)`; the assertion passed `procs` as the unit count *and*
  evaluated it outside the `withProcs` closure, so it read the ambient GOMAXPROCS. At ≥ 8
  cores it coincidentally equals `procs`; on a 2-core runner it was 2 against an actual 3
  and 8, and the library was right in all ten failing lines. Now asserts `workers == procs`,
  which is host-independent because `par.Workers` reads `GOMAXPROCS(0)` and not `NumCPU`,
  and which holds for every shape here because each has more units than `max(p5Threads)`
  (`m=1200` over `MC=144` is 9 ic blocks against 8 threads). Shrink `p5M` and it goes red
  correctly — the arm would have stopped partitioning max-way. Reproduced and fixed under
  `GOMAXPROCS=2`, re-checked at 1 and 4.
- **`gate-p0` exited 1 with no verdict line at all when a host failed its tests while
  exercising all three backends** (#80; `scripts/gate-p0.sh`). `[[ "$ok" -eq 0 ]] &&
  FULL_COVER_TARGET="$name"` was the last command of `record_target`, so on a failing run the
  AND-list's non-zero status became the function's return status — and `set -e`, which
  exempts every command of an AND-OR list but the last, does *not* exempt a function call.
  The gate died after that host's final PASS, before its verdict section, exiting 1; since
  RED also exits 1, the log reads as a truncated red gate rather than as a harness that
  died. It is the #76 family in a different construct, so #76's fix does not cover it. It had
  never fired because it needs a failing test *and* complete coverage: the dev host never has
  complete coverage and the remote hosts had not failed. Driven on the extracted function to
  confirm both the death and the fix, and all seven tail-position `&&` sites in `scripts/`
  were audited — this was the only one whose status could escape (the other six have code
  following them in the same body). The five untouched sites stay `&&` on purpose: rewriting
  them would obscure which one mattered.
- **A host that stopped answering mid-loop killed the gate with exit 255 instead of producing
  a verdict, and the delegating gate reported the death as its delegate's RED** (#76;
  `scripts/gate-p1.sh`, `-p2`, `-p3`, `-p4`, `-p5`, `scripts/remote.sh`). Ten command
  substitutions read a value over ssh without guarding the status — `gov="$(remote_probe
  "$host" | sed -n 's/.*governor=…/\1/p')"` — and every gate runs under `set -euo pipefail`,
  so an unreachable host terminated the gate at that line: no verdict line, no verdict, exit
  255. That is the failure mode `DESIGN.md` §5.6 forbids by name — a killed run is
  unmeasured, never an exit code — and it had two further consequences. The
  unreadable-value UNMEASURED branches #73 had just finished writing were **unreachable in
  precisely the case they exist for**, because the gate died two lines above them. And
  `gate-p4`/`gate-p5` turned the death into `gate-pN is RED (exit 255)`: a red attributed to
  keel for a host that hung up. Found by the #73 sweep's own positive-exercise probe —
  `KEEL_REMOTE_HOSTS=keel-no-such-host.invalid ./scripts/gate-p1.sh` stopped after 21 lines
  with no verdict — which is the argument for exercising a relabel rather than only
  tally-diffing it: the probe was looking for the new label and found a defect underneath it.
  Three parts to the fix. **(1)** All ten substitutions guarded with `|| true`, the idiom
  already in-tree at `gate-p0.sh:189`, so the value comes back empty and empty is a reading
  nobody got, which each caller already knows how to print: seven governor reads (`gate-p1`,
  `-p2`, `-p3` ×2, `-p4` ×2, `-p5`), `gate-p5`'s CPU-model read, and `gate-p3`'s
  `ob_preflight` and `ob_coretype_sweep`. #76 enumerated eight; the two OpenBLAS preflight
  helpers have the identical shape and were missed in it. The rule is now written on
  `remote_probe` itself rather than left to each caller to remember. **(2)** The delegating
  gates read their delegate three ways instead of two: exit 0 *with* a `GREEN` line is PASS,
  exit 1 *with* a `RED` line is FAIL, and anything else — 255 under a dead ssh, 128+n under a
  signal — is **UNMEASURED**, because a gate that died before reaching its own verdict has not
  issued one for this gate to relay. Both witnesses are checked, status and printed line,
  since a delegate that exits 0 having printed nothing has certified nothing. **(3)** Two
  sites the guard makes reachable, which #73's sweep could not see: `gate-p5`'s
  measurement-time governor check printed `'${gov:-unknown}'` inside one collapsed FAIL, so a
  sweep reading messages had no way to notice it could not look — now split exactly like its
  `gate-p3`/`gate-p4` twins — and `gate-p5`'s README re-measurement, where an unreadable CPU
  model matches no published row and would have been reported as `README.md publishes no row
  for ''`, a claim about the README earned by a host that stopped answering. Verified by
  re-running the probe: `gate-p0`, `-p1` and `-p2` now each reach `RED` and exit 1 against an
  unresolvable host, printing UNMEASURED for the target and keeping the aggregate coverage
  FAIL beneath it; before the fix `-p1` and `-p2` died at 255. The delegated UNMEASURED branch
  is a backstop and is verified by inspection only — now that the deaths it catches are fixed,
  no known input produces one, though the pre-fix `gate-p1` produced exactly that condition.
  Also corrected: four stale line citations in `remote.sh`'s split-site list, which pointed 4
  to 11 lines above the `if`s they name, and the two delegated-tally cross-references.
- **`DESIGN.md` §5 rule 8 cited the wrong instance, produced by the rule itself.** The rule
  landed in `2eda333` naming a #65 correction that "took its Sgemm gains from the wrong row
  (`+2.1 / +6.2 / +5.6%` against the actual `+2.0 / +3.2 / +3.5%`)". Both sets are correct
  figures of *different quantities*: the first are 8-thread rate deltas, the second are
  changes in the 8-thread scaling ratio net of CI, which is what the table those cells came
  from says in its own caption. All twelve published cells were right, the withdrawn
  `16.2 / 6.7 / 10.9` ratio was right, and the `15.6 / 3.1 / 7.0` that replaced it is
  withdrawn in its turn — retracted on #21. What supplied the false confirmation was a
  coincidence: Ssymm/janus's ratio delta is +6.21% and Sgemm/janus's rate delta is +6.22%,
  so a matching figure read as a copied cell. One pair of logs answers "how much did Sgemm
  gain on janus" three ways — +6.22% (8-thread rate, 466.2 → 495.2 GFLOP/s), +4.22% (raw
  ratio, 6.211 → 6.473), +3.20% (ratio net of CI, 6.090 → 6.285) — because the change moved
  the 1-thread denominator too (75.06 → 76.50, +1.92%). That the serial arm moved at all is
  itself worth having: parallelising the shared B pack was not expected to touch the
  1-thread path, and on janus it did. Rule 8 gains the clause naming its own failure mode:
  **recompute the same quantity, not merely from the same log** — it instructs distrust of
  the published figure and trust in the fresh recomputation, so a quantity mismatch converts
  directly into a confident false correction. A disagreement with a published number is a
  question, not a verdict. The rule's occurrence count stays at three, but the third is now
  a `gate-p5: 64 PASS / 7 FAIL` tally carried out of a session summary and published in
  prose against the log's `64 PASS / 5 FAIL` — a genuine instance of the carry-forward trap,
  found while correcting the misattributed one.
- **`remote_boost_set` wrote nothing and returned success**, in its first form, on all
  three hosts. `$KEEL_SSH_OPTS` carries `-n`, which redirects ssh's stdin from
  `/dev/null`, so feeding the remote `sh -s` from a heredoc gave it immediate EOF: it
  executed nothing and exited 0. Only reading the knob back caught it — the same
  argument `scripts/remote.sh` already made about the governor, now with a second
  instance behind it. The value is spliced into the remote script instead, after
  validation against a two-element allowlist so the interpolation cannot carry shell
  metacharacters.
- **`gate-p5.sh` could have checked one host's README rows against another host's
  rates.** `BENCHCSV_ON` is a fixed path reused across the host loop, and the boost-on
  pass is allowed to fail without skipping the host, so a stale file would have produced
  a green with the wrong provenance. It is truncated per host and the README criterion
  reports *unmeasured* when that pass produced nothing, rather than agreeing with
  whatever was left behind.

- **The six `internal/l1` reductions lost 4.5–17.4% at n=256** when the `>` guards landed,
  on all three hosts, and now have an exact-fit epilogue of their own. `Sdot`, `Sasum` and
  `Snrm2` regressed by +6.2/+13.7/+9.4%, +9.6/+12.8/+17.4% and +4.5/+6.6/+9.8% on Zen 4 /
  Skylake-X / Zen 5 respectively (`build/l1ab-a2b76eb.log`). 256 is an exact multiple of
  `step512`, so where `>=` consumed it in four unrolled iterations, `>` stops with 64
  elements left and hands them to the 16-wide mop-up loop — which accumulates into `a0`
  alone, replacing four independent FMAs with a four-deep dependent chain at ~4-cycle
  latency. That is the stall the four-accumulator design exists to prevent, reintroduced in
  the tail; the prediction that the mop-up loop made an epilogue unnecessary was wrong, and
  is corrected in place rather than quietly dropped. The epilogue runs the unrolled body once
  at `step` width with all four accumulators live and ends with `x = x[:0]` — truncating a
  length cannot produce a past-the-end pointer, so it costs no conditional bump, and the
  mop-up loop and partial tail below fall through untaken. No partial op is involved, so
  #42's `checkptr` blast radius is unchanged. All six hot loops are byte-identical to before
  the epilogue (same instruction ranges, still 0 bounds-check exits / 0 calls / 0 vector
  stack references), and the four surviving `IsSliceInBounds` are the same four `y = y[:len(x)]`
  precondition re-slices as before. The existing differential suite already covers the new
  branch: dropping the epilogue's `a3` term fails `TestSdot` at n=64, 128 and 4096.

  Verified on all three hosts: the three routines improved at n=256 on 9/9 host×routine cells,
  by 14.9/22.3/18.8% (`Sdot`), 14.4/15.6/22.7% (`Sasum`) and 10.5/22.4/5.4% (`Snrm2`) on Zen 4 /
  Skylake-X / Zen 5, so all three are now at or below their pre-#47 timings except `Snrm2` on
  Zen 5, which remains 3.8% above it. **This commit is nevertheless net negative on one host**:
  geomean sec/op moved −2.36% on Zen 4 and −3.49% on Skylake-X but **+2.52% on Zen 5**, because
  `Saxpy` and `Sscal` — which this diff does not touch, and whose disassembly is byte-identical
  across the two builds — lost up to 45.06% and 19.40% there. That is a code-placement effect,
  not a semantic one: `avx512Dot` grew by 160 bytes, which is a multiple of Go's 32-byte function
  alignment but not of 64, flipping every later function's entry from 64-byte-aligned to 64+32.
  It is `golang/go#8717`, keel #61 and `docs/toolchain-notes.md` T22, and it is disclosed here
  rather than netted out because the fix is not what caused it and reverting the fix would only
  re-win the placement lottery while restoring the dependent-chain stall.

- `scripts/l1-bench.sh` **exited 1 on a fully successful run and leaked a git worktree and
  a temp dir on every invocation** (#55). `WORKTREE` was `local` to the function while the
  single-quoted `EXIT` trap expanded it at exit, in global scope, where the local no longer
  existed — so under `set -u` the trap died on its *first* command and `rm -rf "$BINDIR"`
  never ran, defeating the intent stated in the comment directly above it. Found by the #47
  A/B: every number printed on all three hosts, then `WORKTREE: unbound variable` and exit
  1. Same species as #33 and DESIGN.md §5.6 — a successful run that reports failure
  corrupts the record exactly as much as the reverse, because any wrapper reading the exit
  status sees a failed measurement next to a complete log.

- **The feed decomposition's residual column reached −23.40 ns/call and the
  instrument said nothing about it.** `rest` is the only column whose *sign* carries
  information: it holds the nest's remaining real work (beta pass, fringe branch,
  mask checks), which is positive, plus the interaction between C traffic and panel
  feed, which is not sign-definite. On janus at 2×32 — the shipped shape, and the
  memory-bound one — it runs `+0.20 → −7.40 → −23.40 → −21.50`, i.e. the two
  streams overlap in time, so isolating each one overstates it and the C-traffic and
  panel-feed columns are *upper bounds* there, not estimates. At 4×32 on the same
  host it runs `+4.35 → +6.10 → +10.00 → +14.50`, which is unaccounted nest work
  and would make those columns lower bounds. Either reading changes what the two
  columns above mean, so `feed_rest` now names the dominant sign, its size, and
  which direction it biases the steps — printed last, because it says how far the
  three columns above it can be trusted. A reader should not have to derive the
  sub-additivity of a decomposition from a column the decomposition printed.
- **Three of `BenchmarkFeed`'s arms describe their panels as L1-resident, and above
  `kc=128` they are not.** One kernel call needs an MR×kk A panel and an NR×kk B
  panel, so the reused pair is `(MR+NR)·kk·4` bytes: at NR=32 that is 17 KB at
  kc=128 but 34, 51 and 68 KB at 256, 384 and 512, against a 32 KB L1d on Skylake-X
  and Zen 4. Where the premise fails, both panel arms feed from L2 and
  `cold-panels − loops` is a difference of *locality within one level* rather than of
  level — which is what the vesta run's panel-feed column looks like: resolved and
  positive at kc=128, and at or below its noise floor (or negative) at every larger
  KC. The size cannot be fixed by allocating less, because kk is fixed by the call
  multiset every arm must share. So it is printed instead: a `keel-feed-panels:`
  marker per point gives reused-panel, rotating-C and real-panel byte counts,
  `remote_probe` now reports each host's private cache sizes from sysfs (no host
  record carried them, and Zen 5's L1d is 48 KB where Zen 4's is 32 KB, so a
  from-memory constant would have been wrong), and the panel-feed column says to read
  itself against both. Every doc comment that claimed L1 residency for these buffers
  now says "reused panel pair" and where the residency actually holds.
- **The feed decomposition's noise floor printed as `0%`, which is not a number**
  (T21). benchstat rounds its confidence interval to a whole percent, so `0%` means
  only "under 0.5%" — and `scripts/retention.sh feed` was printing that percent as
  the column a reader uses to tell a resolved step from noise. On vesta it read `0%`
  on seven of eight rows, which says every step is resolved, including the ±0.60 ns
  ones; read correctly it bounds the floor at 0.5% of the arm, up to 4.4 ns on the
  4×32 kc=512 row — larger than three of the four panel-feed steps that row reports.
  The column is now `worst-ci` *and* a `floor` in nanoseconds, computed as
  `(p+0.5)% × the arm it belongs to` so the bound errs toward calling a step
  unresolved, steps below their row's floor are marked `*`, and each term says how
  many of its points are unresolved or negative before the reader reaches its spread.
  A negative cost is now named as an arm defect rather than reported as a cost. No
  gate verdict is affected: gates compare a median *net of* its CI against a floor,
  so a CI rounded down to zero can only make a passing threshold harder to reach —
  but for a *difference between two arms* the rounding is not conservative, and the
  whole feed instrument is differences.
- **`benchstat` was silently declining to compare the two arms of every A/B run**
  (#50, T20). It groups results into one table per distinct *configuration*, where a
  configuration is every `key: value` line in the log — and keel's provenance
  preamble, which exists so that no number ships without its denominator, is in that
  namespace. One of its markers, `keel-bench-clock-mhz`, is a live snapshot of the
  CPU's frequency range and so differs between any two runs on one host by
  construction. Two files that differ in one config key are printed as two
  independent one-column tables: no delta, no percentage, no p-value, exit status 0.
  The first run of `scripts/l1-bench.sh` produced three hosts × two builds × 20
  benchmarks of correct medians and not one comparison among them. `bench_compare`
  in `scripts/bench.sh` now ignores the keys that describe the run rather than the
  build (`$KEEL_BENCH_IGNORE`) **and then checks that a `vs base` column actually
  appeared**, printing `NOT COMPARED` plus the offending keys when it did not. No
  gate verdict was affected: gates aggregate a single log, where a forked table
  cannot lose a comparison. `scripts/l1-bench.sh` now goes through it, and its
  claim that "the deltas carry p-values" — printed above two tables that contained
  no deltas — is gone.
- **`scripts/retention.sh sweep` ran at `-count=10` while its header printed the
  `-count=5` it documents** (#49). `scripts/bench.sh` is sourced first and defaults
  `KEEL_BENCH_COUNT` to 10, so the sweep's own `${KEEL_BENCH_COUNT:-5}` could see
  neither the caller's setting nor its own default. The caller's value is now
  captured *before* sourcing, and the header prints two separate things: the count
  that was requested, and — per host, counted out of the log itself — the number of
  sample rows that actually arrived. A parameter read back out of the measurement
  cannot be shadowed by whatever set it. (The affected sweep is unharmed: 10 is the
  stronger discipline, so the error was in the safe direction, and its numbers stand
  as the exploratory numbers they were labelled.)
- **The sweep's `<- shipped` marker could never fire**, found while fixing #49: it
  matched the shipped triple's literal name, whose `nc=4096` is larger than any NC on
  the grid, so no row was ever marked — which reads as "the shipped point is not on
  the grid". It now marks the shipped KC/MC at the grid's largest NC, read back from
  the CSV, and the label says exactly that rather than implying more.
- **`scripts/retention.sh`, `scripts/l1-bench.sh`, `scripts/roofline-test.sh`,
  `scripts/provision-openblas.sh` and `scripts/bootstrap-github.sh` now define
  everything in functions and end with `main "$@"`** (#51, the convention; the six
  gate scripts are tracked there and go last, each needing a green run of its own).
  Bash
  reads a script incrementally as it executes it, so editing one mid-run can corrupt
  the parse position of the running copy — a hazard that had become a rule to
  remember ("never edit a running instrument"). Forcing a whole-file parse before any
  work begins makes the instrument immune instead. `scripts/roofline.sh` was on the
  list and is off it: it is three function definitions and no top-level work, so it
  is already immune as a sourced library. One behaviour change to declare rather than
  slip in: `provision-openblas.sh --help` prints a fixed line range of its own header,
  so that range was narrowed to end before the new wrapper comment.
- **`spill-audit` could not see a bounds check whose panic block was aligned**, and
  `gate-p2.sh` turns that count into a passing criterion — so the instrument
  certifying "0 surviving bounds checks in the steady-state K-loop" had a false-clean
  mode (#46). `Parse` drops `NOP` lines as zero-length pseudo-instructions, which is
  right for the T9 inlining marker but wrong for *alignment padding*: that owns its
  own offset and is several bytes wide. `reachesPanic` matched the branch target
  exactly, so when the compiler aligned an out-of-line panic block the branch pointed
  at padding, no instruction was found there, and the exit went uncounted. Targets now
  resolve to the first instruction at or after the offset, which finds the same
  instruction wherever an exact match existed — the change can only find *more* exits,
  never fewer. Regression test added to the hand-written listing
  (`TestAuditSeesAPanicBehindAlignmentPadding`), verified to fail against the old
  resolver.

  **The P2 criterion holds.** Re-audited with the fixed detector, `Kernel2x32` and
  `Kernel4x32` still report 0 bounds-check exits and 0 calls, and the three peak
  kernels are still register-only. P2's green was correct — but for a period it was
  correct without being verifiable, and those are not the same thing. Two published
  counts *are* revised, both in `internal/l1` (#47): `avx512Scal` and `avx2Scal` were
  reported clean and carry one each.

- **All ten `internal/l1` vector loops now compile with zero surviving bounds
  checks** (#47). They were written in P1, before P2 wrote the "pre-sliced panels"
  rule for kernels, and they were index-driven: a `for i := 0; i+64 <= len(x); i += 64`
  guard with `x[i+16 : i+32]` sub-slices. `prove` does not discharge those. From
  `i+64 <= len(x)` and `len(x) <= cap(x)` it will not take the step to
  `i+64 <= cap(x)`, and `i <= i+16` needs no-overflow reasoning it also does not do,
  so an unrolled body paid one check per offset sub-slice — `avx512Dot` ran 69
  instructions to issue four FMAs. Every loop is now driven by `len(x)` with
  *constant* offsets (`x[16:32]`, never `x[i+16:i+32]`) and re-slices at the bottom,
  which is the idiom `internal/vec`'s microkernels already use for their panels.
  The two-slice routines (`Dot`, `Axpy`) re-slice `y` to `len(x)` once above the
  loop *and* carry `len(y) >= step` in the guard: the re-slice alone is not enough,
  because the prover loses `len(y) == len(x)` across `y = y[step:]`, which left
  seven of `y`'s checks standing after `x`'s had all gone. `check_bce` on
  `l1_amd64.go` goes from 60 reports to 4 — one `IsSliceInBounds` per two-slice
  routine, which is the `y = y[:len(x)]` precondition itself, hoisted out of the
  loop and paid once per call instead of per iteration. (`l1.go`'s 21 are the
  scalar reference path and are untouched.) `spill-audit` reports 0
  bounds-check exits for all ten vector kernels, where it previously reported
  them for all ten.

  **The instruction counts do not all improve, and T19 is why.** Excluding T9's
  1-byte inlining NOPs: `avx512Asum` 41→20, `avx512Dot` 61→32, `avx512SumSq`
  42→21, `avx2Asum` 45→24, `avx2Dot` 61→32, `avx2SumSq` 42→21 — the six unrolled
  reductions roughly halve. But `avx512Axpy` 16→21, `avx2Axpy` 17→22,
  `avx512Scal` 11→12, `avx2Scal` 12→13. A slice advance guarded by `>=` costs
  seven instructions, not two, because the loop may leave the slice exactly empty
  and a slice's data pointer must not pass the end of its allocation, so the
  pointer bump is made conditional (`NEGQ`/`SARQ $63`/`ANDL`) — see
  docs/toolchain-notes.md T19, which has the three-function repro. An unrolled body
  amortizes that over four vector ops; a non-unrolled one cannot, and `Axpy`
  advances two slices. The `>` form collapses the advance to a single `ADDQ`, and
  is applicable here because the tail already absorbs a full vector through
  `LoadPart`/`StorePart` — but it moves the last full vector onto the masked path,
  so it is a second change wanting its own measurement rather than a free win.
  Runtime numbers for all five routines at L1-, L2-, L3- and memory-resident sizes
  are #47's remaining deliverable; the reassociation order is unchanged, so the
  results are bit-identical, not merely within tolerance.
