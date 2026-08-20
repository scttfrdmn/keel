# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, minor versions may contain breaking changes.

## [Unreleased]

### Added
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

### Changed
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

### Fixed
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
  only `openblas.go`'s 106 ever reached it, so **1549 lines of benchmark harness — including the 376-line
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

### Added
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

### Changed
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

### Fixed
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

### Changed
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

### Fixed
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

### Changed
- **The 115-shape generator behind the spill frontier is not in the tree and never was** (#107): only the audit half
  is, so the report's part 4 table cannot be regenerated and neither can `SWEEP_BEST_IPF=4.438`, which gate-p2
  criterion 5b reads.
  Recorded in the report; the rebuild's in-tree/out-of-tree question is Scott's.
- **`docs/spill-report.md` is reopened (part 10): P2 and P3 are both red on the first evidentiary host**, a full-size
  `c7i.48xlarge` — 34.2% of measured peak and a 51.0% mission ratio. 55% needs ≤ 4.09 insns/FMA against the shipped
  6.25, and golang/go#80829 plus #80830 together reach only 50.0%. The report's part 9 stands for the retired fleet.
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

### Fixed
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

### Added
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

### Changed
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

### Fixed
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

### Added
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

### Changed
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

### Fixed
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

### Added
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

### Fixed
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

### Changed
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

### Fixed
- `scripts/l1-bench.sh` **exited 1 on a fully successful run and leaked a git worktree and
  a temp dir on every invocation** (#55). `WORKTREE` was `local` to the function while the
  single-quoted `EXIT` trap expanded it at exit, in global scope, where the local no longer
  existed — so under `set -u` the trap died on its *first* command and `rm -rf "$BINDIR"`
  never ran, defeating the intent stated in the comment directly above it. Found by the #47
  A/B: every number printed on all three hosts, then `WORKTREE: unbound variable` and exit
  1. Same species as #33 and DESIGN.md §5.6 — a successful run that reports failure
  corrupts the record exactly as much as the reverse, because any wrapper reading the exit
  status sees a failed measurement next to a complete log.

### Added
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
  [#80828](https://github.com/golang/go/issues/80828) (512-bit values allocated from
  15 of 32 zmm registers — a fresh sweep puts the zero-spill frontier at 13
  independent accumulators and shows no register above Z14 is ever named),
  [#80829](https://github.com/golang/go/issues/80829) (no 231-shaped vector FMA, the
  load-merge rule folds the addend, nothing emits `.BCST` — with the byte-identical
  `go tool asm` / `llvm-mc` encodings and a 9-instructions-for-2-FMAs GEMM row),
  [#80830](https://github.com/golang/go/issues/80830) (`BroadcastFloat32x16` is
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

### Fixed
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

### Changed
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
